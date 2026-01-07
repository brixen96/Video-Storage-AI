package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/brixen96/video-storage-ai/internal/database"
	"github.com/brixen96/video-storage-ai/internal/models"
	"github.com/fsnotify/fsnotify"
)

// Recommendation represents a suggestion made by the AI Companion
type Recommendation struct {
	ID          string
	Type        string    // thumbnail_generation, scanning, metadata_fetch, etc.
	TaskType    string    // maps to activity_logs.task_type
	Message     string
	SuggestedAt time.Time
	Context     map[string]interface{}
}

// AICompanionService is the always-running AI companion
type AICompanionService struct {
	db                *sql.DB
	ctx               context.Context
	cancel            context.CancelFunc
	watchers          map[string]*fsnotify.Watcher // library_id -> watcher
	eventChan         chan CompanionEvent
	subscribers       []chan CompanionEvent
	mu                sync.RWMutex
	isRunning         bool
	startTime         time.Time
	eventsProcessed   int64
	recommendations   map[string]*Recommendation // recommendation_id -> Recommendation
	lastActivityCheck time.Time                  // Last time we checked activity_logs
}

// CompanionEvent represents an event the AI detected
type CompanionEvent struct {
	Type      string                 `json:"type"` // file_added, file_removed, file_modified, insight, notification
	Source    string                 `json:"source"` // file_watcher, analysis_engine, health_monitor
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Severity  string                 `json:"severity"` // info, warning, error, critical
	Timestamp time.Time              `json:"timestamp"`
}

// NewAICompanionService creates and initializes the AI Companion
func NewAICompanionService() *AICompanionService {
	ctx, cancel := context.WithCancel(context.Background())
	return &AICompanionService{
		db:                database.GetDB(),
		ctx:               ctx,
		cancel:            cancel,
		watchers:          make(map[string]*fsnotify.Watcher),
		eventChan:         make(chan CompanionEvent, 100),
		subscribers:       make([]chan CompanionEvent, 0),
		recommendations:   make(map[string]*Recommendation),
		lastActivityCheck: time.Now(),
		isRunning:         false,
	}
}

// Start begins the AI Companion background service
func (s *AICompanionService) Start() error {
	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		return fmt.Errorf("AI Companion is already running")
	}
	s.isRunning = true
	s.startTime = time.Now()
	s.mu.Unlock()

	log.Println("🤖 Starting AI Companion Service...")

	// Start file watchers for all libraries
	if err := s.startFileWatchers(); err != nil {
		log.Printf("Warning: Failed to start file watchers: %v", err)
	}

	// Start event processor
	go s.processEvents()

	// Start background monitoring routines
	go s.monitorLibraryHealth()
	go s.performPeriodicAnalysis()
	go s.monitorActivityLogs()
	go s.MonitorConsoleLogsForErrors()

	log.Println("✅ AI Companion Service started successfully")
	return nil
}

// Stop gracefully shuts down the AI Companion
func (s *AICompanionService) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return fmt.Errorf("AI Companion is not running")
	}

	log.Println("🛑 Stopping AI Companion Service...")

	// Cancel context
	s.cancel()

	// Close all file watchers
	for libID, watcher := range s.watchers {
		if err := watcher.Close(); err != nil {
			log.Printf("Error closing watcher for library %s: %v", libID, err)
		}
	}

	s.isRunning = false
	log.Println("✅ AI Companion Service stopped")
	return nil
}

// GetStatus returns the current status of the AI Companion
func (s *AICompanionService) GetStatus() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	uptime := time.Duration(0)
	if s.isRunning {
		uptime = time.Since(s.startTime)
	}

	return map[string]interface{}{
		"running":           s.isRunning,
		"uptime_seconds":    uptime.Seconds(),
		"active_watchers":   len(s.watchers),
		"events_processed":  s.eventsProcessed,
		"start_time":        s.startTime,
		"subscriber_count":  len(s.subscribers),
	}
}

// SubscribeToEvents allows clients to receive real-time events
// Returns the event channel and an unsubscribe function
func (s *AICompanionService) SubscribeToEvents() (<-chan CompanionEvent, func()) {
	s.mu.Lock()
	defer s.mu.Unlock()

	eventChan := make(chan CompanionEvent, 10)
	s.subscribers = append(s.subscribers, eventChan)

	// Return unsubscribe function
	unsubscribe := func() {
		s.mu.Lock()
		defer s.mu.Unlock()

		// Find and remove the channel from subscribers
		for i, sub := range s.subscribers {
			if sub == eventChan {
				// Remove by swapping with last element and truncating
				s.subscribers[i] = s.subscribers[len(s.subscribers)-1]
				s.subscribers = s.subscribers[:len(s.subscribers)-1]
				close(eventChan)
				break
			}
		}
	}

	return eventChan, unsubscribe
}

// startFileWatchers initializes file system watchers for all libraries
func (s *AICompanionService) startFileWatchers() error {
	// Get all libraries from database
	rows, err := s.db.Query(`SELECT id, name, path FROM libraries`)
	if err != nil {
		return fmt.Errorf("failed to query libraries: %w", err)
	}
	defer rows.Close()

	libraries := []struct {
		ID   int64
		Name string
		Path string
	}{}

	for rows.Next() {
		var lib struct {
			ID   int64
			Name string
			Path string
		}
		if err := rows.Scan(&lib.ID, &lib.Name, &lib.Path); err != nil {
			log.Printf("Error scanning library: %v", err)
			continue
		}
		libraries = append(libraries, lib)
	}

	// Start watcher for each library
	for _, lib := range libraries {
		if err := s.addLibraryWatcher(lib.ID, lib.Name, lib.Path); err != nil {
			log.Printf("Warning: Failed to add watcher for library '%s': %v", lib.Name, err)
		} else {
			log.Printf("📂 Watching library: %s (%s)", lib.Name, lib.Path)
		}
	}

	return nil
}

// addLibraryWatcher adds a file watcher for a specific library
func (s *AICompanionService) addLibraryWatcher(id int64, name, path string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	// Add the library path to the watcher
	if err := watcher.Add(path); err != nil {
		watcher.Close()
		return fmt.Errorf("failed to watch path: %w", err)
	}

	libKey := fmt.Sprintf("%d", id)
	s.watchers[libKey] = watcher

	// Start monitoring this watcher
	go s.watchLibrary(watcher, name, path)

	return nil
}

// watchLibrary monitors file system events for a library
func (s *AICompanionService) watchLibrary(watcher *fsnotify.Watcher, libraryName, libraryPath string) {
	for {
		select {
		case <-s.ctx.Done():
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			s.handleFileEvent(event, libraryName, libraryPath)
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error for %s: %v", libraryName, err)
		}
	}
}

// handleFileEvent processes file system events
func (s *AICompanionService) handleFileEvent(event fsnotify.Event, libraryName, libraryPath string) {
	// Filter for video files only
	ext := strings.ToLower(filepath.Ext(event.Name))
	videoExts := map[string]bool{
		".mp4": true, ".mkv": true, ".avi": true, ".mov": true,
		".wmv": true, ".flv": true, ".webm": true, ".m4v": true,
	}

	if !videoExts[ext] {
		return
	}

	var eventType string
	var message string

	switch {
	case event.Op&fsnotify.Create == fsnotify.Create:
		eventType = "file_added"
		message = fmt.Sprintf("New video detected: %s", filepath.Base(event.Name))
	case event.Op&fsnotify.Remove == fsnotify.Remove:
		eventType = "file_removed"
		message = fmt.Sprintf("Video removed: %s", filepath.Base(event.Name))
	case event.Op&fsnotify.Write == fsnotify.Write:
		eventType = "file_modified"
		message = fmt.Sprintf("Video modified: %s", filepath.Base(event.Name))
	case event.Op&fsnotify.Rename == fsnotify.Rename:
		eventType = "file_renamed"
		message = fmt.Sprintf("Video renamed: %s", filepath.Base(event.Name))
	default:
		return // Ignore other events
	}

	// Emit event
	s.emitEvent(CompanionEvent{
		Type:   eventType,
		Source: "file_watcher",
		Message: message,
		Data: map[string]interface{}{
			"library":   libraryName,
			"file_path": event.Name,
			"operation": event.Op.String(),
		},
		Severity:  "info",
		Timestamp: time.Now(),
	})

	// Autonomous decision: If new file, trigger analysis after short delay
	if eventType == "file_added" {
		go s.analyzeNewFile(event.Name, libraryName)
	}
}

// analyzeNewFile performs autonomous analysis on a new file
func (s *AICompanionService) analyzeNewFile(filePath, libraryName string) {
	// Wait a bit to ensure file is fully written
	time.Sleep(5 * time.Second)

	// Emit analysis event
	s.emitEvent(CompanionEvent{
		Type:    "insight",
		Source:  "analysis_engine",
		Message: fmt.Sprintf("Analyzing new video in %s", libraryName),
		Data: map[string]interface{}{
			"file_path": filePath,
			"library":   libraryName,
		},
		Severity:  "info",
		Timestamp: time.Now(),
	})

	// TODO: Trigger actual analysis (metadata extraction, performer detection, etc.)
}

// processEvents processes events and broadcasts to subscribers
func (s *AICompanionService) processEvents() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case event := <-s.eventChan:
			s.mu.Lock()
			s.eventsProcessed++
			s.mu.Unlock()

			// Broadcast to all subscribers
			s.mu.RLock()
			for _, subscriber := range s.subscribers {
				// Use select with default to avoid blocking
				// Also wrap in recover to handle closed channel panics
				func() {
					defer func() {
						if r := recover(); r != nil {
							// Channel was closed, ignore
						}
					}()
					select {
					case subscriber <- event:
					default:
						// Skip if subscriber channel is full
					}
				}()
			}
			s.mu.RUnlock()

			// Log important events
			if event.Severity == "warning" || event.Severity == "error" || event.Severity == "critical" {
				log.Printf("[AI Companion] %s: %s", strings.ToUpper(event.Severity), event.Message)
			}
		}
	}
}

// emitEvent sends an event to the event channel
func (s *AICompanionService) emitEvent(event CompanionEvent) {
	select {
	case s.eventChan <- event:
	default:
		log.Printf("Event channel full, dropping event: %s", event.Message)
	}
}

// monitorLibraryHealth periodically checks library health
func (s *AICompanionService) monitorLibraryHealth() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.checkLibraryHealth()
		}
	}
}

// checkLibraryHealth performs health checks on libraries
func (s *AICompanionService) checkLibraryHealth() {
	// Get comprehensive library stats
	var videoCount, performerCount, tagCount, studioCount int
	var performersWithThumbnails, performersWithPreviews, performersWithoutThumbnails int
	var videosWithThumbnails, videosWithPreviews int

	s.db.QueryRow("SELECT COUNT(*) FROM videos").Scan(&videoCount)
	s.db.QueryRow("SELECT COUNT(*) FROM performers").Scan(&performerCount)
	s.db.QueryRow("SELECT COUNT(*) FROM tags").Scan(&tagCount)
	s.db.QueryRow("SELECT COUNT(*) FROM studios").Scan(&studioCount)

	// Check performer thumbnail coverage
	s.db.QueryRow("SELECT COUNT(*) FROM performers WHERE thumbnail_path IS NOT NULL AND thumbnail_path != ''").Scan(&performersWithThumbnails)
	s.db.QueryRow("SELECT COUNT(*) FROM performers WHERE preview_path IS NOT NULL AND preview_path != ''").Scan(&performersWithPreviews)
	s.db.QueryRow("SELECT COUNT(*) FROM performers WHERE preview_path IS NOT NULL AND preview_path != '' AND (thumbnail_path IS NULL OR thumbnail_path = '')").Scan(&performersWithoutThumbnails)

	// Check video thumbnail coverage
	s.db.QueryRow("SELECT COUNT(*) FROM videos WHERE thumbnail_path IS NOT NULL AND thumbnail_path != ''").Scan(&videosWithThumbnails)
	s.db.QueryRow("SELECT COUNT(*) FROM videos WHERE preview_path IS NOT NULL AND preview_path != ''").Scan(&videosWithPreviews)

	// === PERFORMER THUMBNAIL MONITORING ===
	// Alert if performers have previews but no thumbnails (performance issue)
	if performersWithoutThumbnails > 5 {
		message := fmt.Sprintf("⚡ Performance Opportunity: %d performers have preview videos but no thumbnails. Generate thumbnails for faster page loading!", performersWithoutThumbnails)

		// Track this recommendation
		s.trackRecommendation(
			"performance_optimization",
			"performer_thumbnail_generation",
			message,
			map[string]interface{}{
				"performers_without_thumbnails": performersWithoutThumbnails,
				"performers_with_previews":      performersWithPreviews,
			},
		)

		s.emitEvent(CompanionEvent{
			Type:    "notification",
			Source:  "health_monitor",
			Message: message,
			Data: map[string]interface{}{
				"performers_without_thumbnails": performersWithoutThumbnails,
				"performers_with_previews":      performersWithPreviews,
				"action":                        "generate_performer_thumbnails",
			},
			Severity:  "info",
			Timestamp: time.Now(),
		})
	}

	// === TAGGING MONITORING ===
	if videoCount > 50 && tagCount < 5 {
		s.emitEvent(CompanionEvent{
			Type:    "notification",
			Source:  "health_monitor",
			Message: fmt.Sprintf("📋 Organization Tip: You have %d videos but only %d tags. Consider using Smart Tagging to better organize your library.", videoCount, tagCount),
			Data: map[string]interface{}{
				"video_count": videoCount,
				"tag_count":   tagCount,
				"action":      "smart_tagging",
			},
			Severity:  "warning",
			Timestamp: time.Now(),
		})
	}

	// === PERFORMER LINKING MONITORING ===
	if videoCount > 100 && performerCount < 10 {
		s.emitEvent(CompanionEvent{
			Type:    "notification",
			Source:  "health_monitor",
			Message: fmt.Sprintf("🔗 Metadata Alert: %d videos detected with only %d performers. Try Auto-Link Performers to improve organization.", videoCount, performerCount),
			Data: map[string]interface{}{
				"video_count":     videoCount,
				"performer_count": performerCount,
				"action":          "auto_link_performers",
			},
			Severity:  "warning",
			Timestamp: time.Now(),
		})
	}

	// === VIDEO PREVIEW MONITORING ===
	videosWithoutPreviews := videoCount - videosWithPreviews
	if videosWithoutPreviews > 20 && videoCount > 50 {
		coveragePercent := float64(videosWithPreviews) / float64(videoCount) * 100
		s.emitEvent(CompanionEvent{
			Type:    "notification",
			Source:  "health_monitor",
			Message: fmt.Sprintf("🎬 Preview Coverage: Only %.1f%% of videos have previews. Generate previews for better browsing experience.", coveragePercent),
			Data: map[string]interface{}{
				"videos_without_previews": videosWithoutPreviews,
				"total_videos":           videoCount,
				"coverage_percent":       coveragePercent,
				"action":                 "generate_previews",
			},
			Severity:  "info",
			Timestamp: time.Now(),
		})
	}

	// === METADATA COMPLETENESS ===
	var performersWithoutMetadata int
	s.db.QueryRow("SELECT COUNT(*) FROM performers WHERE category = 'regular' AND (metadata IS NULL OR metadata = '{}' OR metadata = '')").Scan(&performersWithoutMetadata)
	if performersWithoutMetadata > 10 {
		s.emitEvent(CompanionEvent{
			Type:    "notification",
			Source:  "health_monitor",
			Message: fmt.Sprintf("📊 Metadata Incomplete: %d performers are missing metadata. Fetch from AdultDataLink for richer information.", performersWithoutMetadata),
			Data: map[string]interface{}{
				"performers_without_metadata": performersWithoutMetadata,
				"action":                      "fetch_metadata",
			},
			Severity:  "info",
			Timestamp: time.Now(),
		})
	}

	// === LIBRARY HEALTH SUMMARY (Every 24 hours) ===
	uptime := time.Since(s.startTime)
	if uptime > 24*time.Hour && int(uptime.Hours())%24 == 0 {
		thumbnailCoverage := float64(0)
		if performersWithPreviews > 0 {
			thumbnailCoverage = float64(performersWithThumbnails) / float64(performersWithPreviews) * 100
		}

		s.emitEvent(CompanionEvent{
			Type:    "notification",
			Source:  "health_monitor",
			Message: fmt.Sprintf("📈 Daily Library Report: %d videos, %d performers, %d tags, %d studios. Performer thumbnail coverage: %.1f%%", videoCount, performerCount, tagCount, studioCount, thumbnailCoverage),
			Data: map[string]interface{}{
				"video_count":         videoCount,
				"performer_count":     performerCount,
				"tag_count":           tagCount,
				"studio_count":        studioCount,
				"thumbnail_coverage":  thumbnailCoverage,
				"uptime_hours":        int(uptime.Hours()),
			},
			Severity:  "info",
			Timestamp: time.Now(),
		})
	}
}

// performPeriodicAnalysis runs automated analysis tasks
func (s *AICompanionService) performPeriodicAnalysis() {
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			// TODO: Run automated tasks based on learned patterns
			log.Println("🔍 AI Companion: Running periodic analysis...")
		}
	}
}

// monitorActivityLogs monitors activity_logs table for task completions
func (s *AICompanionService) monitorActivityLogs() {
	ticker := time.NewTicker(5 * time.Second) // Check every 5 seconds
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.checkActivityLogUpdates()
		}
	}
}

// checkActivityLogUpdates checks for new completed activities
func (s *AICompanionService) checkActivityLogUpdates() {
	// Query for activities that completed since last check
	query := `
		SELECT id, task_type, status, message, progress, started_at, completed_at, details
		FROM activity_logs
		WHERE status = 'completed'
		AND completed_at IS NOT NULL
		AND completed_at > ?
		ORDER BY completed_at ASC
	`

	rows, err := s.db.Query(query, s.lastActivityCheck)
	if err != nil {
		log.Printf("Failed to query activity logs: %v", err)
		return
	}
	defer rows.Close()

	s.mu.Lock()
	currentTime := time.Now()
	s.mu.Unlock()

	for rows.Next() {
		var activity models.ActivityLog
		err := rows.Scan(
			&activity.ID,
			&activity.TaskType,
			&activity.Status,
			&activity.Message,
			&activity.Progress,
			&activity.StartedAt,
			&activity.CompletedAt,
			&activity.Details,
		)
		if err != nil {
			log.Printf("Failed to scan activity log: %v", err)
			continue
		}

		// Unmarshal details
		if err := activity.UnmarshalDetails(); err != nil {
			log.Printf("Failed to unmarshal activity details: %v", err)
		}

		// Process the completed activity
		s.processCompletedActivity(&activity)
	}

	// Update last check time
	s.mu.Lock()
	s.lastActivityCheck = currentTime
	s.mu.Unlock()
}

// processCompletedActivity processes a completed activity and reacts if it matches a recommendation
func (s *AICompanionService) processCompletedActivity(activity *models.ActivityLog) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if this activity matches any of our recommendations
	var matchedRecommendation *Recommendation
	for id, rec := range s.recommendations {
		if rec.TaskType == activity.TaskType {
			matchedRecommendation = rec
			delete(s.recommendations, id) // Remove once matched
			break
		}
	}

	// Generate contextual response based on task type and whether it was recommended
	var responseMessage string
	var severity string = "info"

	if matchedRecommendation != nil {
		// User followed our recommendation!
		responseMessage = s.generateFollowUpResponse(activity, matchedRecommendation)
		severity = "info"
	} else {
		// User initiated this task on their own - still acknowledge
		responseMessage = s.generateAcknowledgmentResponse(activity)
		severity = "info"
	}

	// Emit the response event
	if responseMessage != "" {
		s.emitEvent(CompanionEvent{
			Type:    "notification",
			Source:  "activity_monitor",
			Message: responseMessage,
			Data: map[string]interface{}{
				"activity_id":       activity.ID,
				"task_type":         activity.TaskType,
				"was_recommended":   matchedRecommendation != nil,
				"completion_time":   activity.CompletedAt,
				"details":           activity.DetailsObj,
			},
			Severity:  severity,
			Timestamp: time.Now(),
		})
	}

	// Perform post-completion analysis
	s.analyzeCompletionImpact(activity)
}

// generateFollowUpResponse creates a contextual response when user follows a recommendation
func (s *AICompanionService) generateFollowUpResponse(activity *models.ActivityLog, rec *Recommendation) string {
	switch activity.TaskType {
	case "performer_thumbnail_generation":
		totalCount := 0
		if val, ok := activity.DetailsObj["total_count"].(float64); ok {
			totalCount = int(val)
		}
		return fmt.Sprintf("🎉 Great! I see you generated thumbnails for %d performers. The Performers page should load much faster now. Your library performance is improving!", totalCount)

	case "video_thumbnail_generation":
		return "🎉 Excellent! Video thumbnails have been generated. Your library browsing experience should be much smoother now!"

	case "thumbnail_generation_batch":
		totalCount := 0
		if val, ok := activity.DetailsObj["total_count"].(float64); ok {
			totalCount = int(val)
		}
		return fmt.Sprintf("✅ Excellent! Video thumbnails generated for %d files. Browse performance is now optimized.", totalCount)

	case "library_scan":
		return "🔍 Library scan completed! I've indexed all the new content. Your library is now up to date."

	case "performer_scan":
		scannedCount := 0
		if val, ok := activity.DetailsObj["scanned_count"].(float64); ok {
			scannedCount = int(val)
		}
		return fmt.Sprintf("👤 Performer scan finished! Processed %d performers. All performer previews are now updated.", scannedCount)

	case "metadata_fetch":
		return "📋 Metadata fetching complete! Your content now has enriched information from external sources."

	case "ai_tagging":
		return "🏷️ AI tagging finished! Your videos are now automatically categorized and easier to discover."

	default:
		return fmt.Sprintf("✅ Task '%s' completed successfully! Thanks for keeping your library optimized.", activity.TaskType)
	}
}

// generateAcknowledgmentResponse creates a response for user-initiated tasks
func (s *AICompanionService) generateAcknowledgmentResponse(activity *models.ActivityLog) string {
	switch activity.TaskType {
	case "performer_thumbnail_generation":
		return "✅ Performer thumbnails generated! Your Performers page should load significantly faster now."

	case "video_thumbnail_generation":
		return "✅ Video thumbnails generated successfully! Your videos now have preview thumbnails."

	case "thumbnail_generation_batch":
		return "✅ Video thumbnails generated successfully! Browse performance improved."

	case "library_scan":
		return "✅ Library scan complete! All content has been indexed."

	case "performer_scan":
		return "✅ Performer scan complete! All performer previews updated."

	default:
		// Only acknowledge major tasks, return empty for minor ones
		return ""
	}
}

// analyzeCompletionImpact analyzes the impact of a completed task and suggests next steps
func (s *AICompanionService) analyzeCompletionImpact(activity *models.ActivityLog) {
	// After thumbnail generation, check if there are other optimization opportunities
	if activity.TaskType == "performer_thumbnail_generation" {
		// Check if video thumbnails also need generation
		var videosNeedingThumbnails int
		s.db.QueryRow(`
			SELECT COUNT(*)
			FROM videos
			WHERE (thumbnail_path IS NULL OR thumbnail_path = '')
		`).Scan(&videosNeedingThumbnails)

		if videosNeedingThumbnails > 100 {
			s.emitEvent(CompanionEvent{
				Type:    "notification",
				Source:  "activity_monitor",
				Message: fmt.Sprintf("💡 Next optimization: %d videos don't have thumbnails. Consider generating them for even better performance!", videosNeedingThumbnails),
				Data: map[string]interface{}{
					"videos_needing_thumbnails": videosNeedingThumbnails,
					"action":                    "generate_video_thumbnails",
				},
				Severity:  "info",
				Timestamp: time.Now(),
			})
		}
	}

	// After library scan, suggest metadata enrichment
	if activity.TaskType == "library_scan" {
		var videosWithoutMetadata int
		s.db.QueryRow(`
			SELECT COUNT(*)
			FROM videos
			WHERE (metadata IS NULL OR metadata = '' OR metadata = '{}')
		`).Scan(&videosWithoutMetadata)

		if videosWithoutMetadata > 50 {
			s.emitEvent(CompanionEvent{
				Type:    "notification",
				Source:  "activity_monitor",
				Message: fmt.Sprintf("📋 Suggestion: %d videos have minimal metadata. Fetch metadata to enrich your library!", videosWithoutMetadata),
				Data: map[string]interface{}{
					"videos_without_metadata": videosWithoutMetadata,
					"action":                  "fetch_metadata",
				},
				Severity:  "info",
				Timestamp: time.Now(),
			})
		}
	}
}

// trackRecommendation stores a recommendation that was given to the user
func (s *AICompanionService) trackRecommendation(recType, taskType, message string, context map[string]interface{}) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := fmt.Sprintf("%s_%d", recType, time.Now().UnixNano())
	rec := &Recommendation{
		ID:          id,
		Type:        recType,
		TaskType:    taskType,
		Message:     message,
		SuggestedAt: time.Now(),
		Context:     context,
	}

	s.recommendations[id] = rec
	return id
}

// ================== Advanced Query Functions ==================

// SearchVideos searches for videos by various criteria
func (s *AICompanionService) SearchVideos(query string) ([]string, error) {
	searchQuery := `
		SELECT v.title, v.file_path, p.name as performer_name
		FROM videos v
		LEFT JOIN video_performers vp ON v.id = vp.video_id
		LEFT JOIN performers p ON vp.performer_id = p.id
		WHERE v.title LIKE ? OR p.name LIKE ?
		GROUP BY v.id
		LIMIT 10
	`

	rows, err := s.db.Query(searchQuery, "%"+query+"%", "%"+query+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []string{}
	for rows.Next() {
		var title, filePath string
		var performerName sql.NullString
		if err := rows.Scan(&title, &filePath, &performerName); err != nil {
			continue
		}

		result := fmt.Sprintf("📹 %s", title)
		if performerName.Valid && performerName.String != "" {
			result += fmt.Sprintf(" (featuring %s)", performerName.String)
		}
		results = append(results, result)
	}

	if len(results) == 0 {
		return []string{fmt.Sprintf("No videos found matching '%s'", query)}, nil
	}

	return results, nil
}

// FindPerformer searches for a performer and returns detailed info
func (s *AICompanionService) FindPerformer(name string) (string, error) {
	query := `
		SELECT p.id, p.name, p.category, COUNT(vp.video_id) as video_count,
		       p.thumbnail_path, p.preview_path
		FROM performers p
		LEFT JOIN video_performers vp ON p.id = vp.performer_id
		WHERE p.name LIKE ?
		GROUP BY p.id
		LIMIT 1
	`

	var id int64
	var performerName, category string
	var videoCount int
	var thumbnailPath, previewPath sql.NullString

	err := s.db.QueryRow(query, "%"+name+"%").Scan(&id, &performerName, &category, &videoCount, &thumbnailPath, &previewPath)
	if err == sql.ErrNoRows {
		return fmt.Sprintf("I couldn't find a performer matching '%s'. Try checking the Performers page or scanning for new performers.", name), nil
	}
	if err != nil {
		return "", err
	}

	var response strings.Builder
	response.WriteString(fmt.Sprintf("👤 Performer: %s\n", performerName))
	response.WriteString(fmt.Sprintf("Category: %s\n", category))
	response.WriteString(fmt.Sprintf("Videos: %d\n", videoCount))

	hasThumbnail := thumbnailPath.Valid && thumbnailPath.String != ""
	hasPreview := previewPath.Valid && previewPath.String != ""

	response.WriteString(fmt.Sprintf("Thumbnail: %s\n", map[bool]string{true: "✅", false: "❌"}[hasThumbnail]))
	response.WriteString(fmt.Sprintf("Preview: %s\n", map[bool]string{true: "✅", false: "❌"}[hasPreview]))

	return response.String(), nil
}

// GetTaggedVideos finds videos with specific tags
func (s *AICompanionService) GetTaggedVideos(tagName string) ([]string, error) {
	query := `
		SELECT v.title, t.name as tag_name
		FROM videos v
		INNER JOIN video_tags vt ON v.id = vt.video_id
		INNER JOIN tags t ON vt.tag_id = t.id
		WHERE t.name LIKE ?
		LIMIT 10
	`

	rows, err := s.db.Query(query, "%"+tagName+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []string{}
	for rows.Next() {
		var title, tag string
		if err := rows.Scan(&title, &tag); err != nil {
			continue
		}
		results = append(results, fmt.Sprintf("🏷️ %s [%s]", title, tag))
	}

	if len(results) == 0 {
		return []string{fmt.Sprintf("No videos found with tag '%s'", tagName)}, nil
	}

	return results, nil
}

// CountPostsWithKeyword counts posts containing a specific keyword
func (s *AICompanionService) CountPostsWithKeyword(keyword string, threadID ...int64) (int, error) {
	var query string
	var args []interface{}
	var count int

	if len(threadID) > 0 && threadID[0] > 0 {
		query = "SELECT COUNT(*) FROM scraped_posts WHERE thread_id = ? AND plain_text LIKE ?"
		args = []interface{}{threadID[0], "%" + keyword + "%"}
	} else {
		query = "SELECT COUNT(*) FROM scraped_posts WHERE plain_text LIKE ?"
		args = []interface{}{"%" + keyword + "%"}
	}

	err := s.db.QueryRow(query, args...).Scan(&count)
	return count, err
}

// SearchThreadPosts searches for text in scraped forum posts
func (s *AICompanionService) SearchThreadPosts(searchTerm string, threadID ...int64) ([]map[string]interface{}, error) {
	var query string
	var args []interface{}

	if len(threadID) > 0 && threadID[0] > 0 {
		// Search within specific thread
		query = `
			SELECT sp.id, sp.plain_text, sp.author, sp.post_number, st.title, st.id as thread_id
			FROM scraped_posts sp
			INNER JOIN scraped_threads st ON sp.thread_id = st.id
			WHERE sp.thread_id = ? AND sp.plain_text LIKE ?
			ORDER BY sp.post_number ASC
			LIMIT 50
		`
		args = []interface{}{threadID[0], "%" + searchTerm + "%"}
	} else {
		// Search across all posts
		query = `
			SELECT sp.id, sp.plain_text, sp.author, sp.post_number, st.title, st.id as thread_id
			FROM scraped_posts sp
			INNER JOIN scraped_threads st ON sp.thread_id = st.id
			WHERE sp.plain_text LIKE ?
			ORDER BY st.id, sp.post_number ASC
			LIMIT 50
		`
		args = []interface{}{"%" + searchTerm + "%"}
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []map[string]interface{}{}
	for rows.Next() {
		var postID, threadID int64
		var plainText, author, threadTitle string
		var postNumber int

		err = rows.Scan(&postID, &plainText, &author, &postNumber, &threadTitle, &threadID)
		if err != nil {
			continue
		}

		// Truncate text to 300 chars for display
		excerpt := plainText
		if len(excerpt) > 300 {
			excerpt = excerpt[:300] + "..."
		}

		results = append(results, map[string]interface{}{
			"post_id":      postID,
			"thread_id":    threadID,
			"thread_title": threadTitle,
			"post_number":  postNumber,
			"author":       author,
			"excerpt":      excerpt,
			"full_text":    plainText,
		})
	}

	return results, nil
}

// GetThreadByName finds a thread by partial name match
func (s *AICompanionService) GetThreadByName(threadName string) (int64, string, error) {
	var threadID int64
	var title string

	query := `
		SELECT id, title
		FROM scraped_threads
		WHERE title LIKE ?
		LIMIT 1
	`

	err := s.db.QueryRow(query, "%"+threadName+"%").Scan(&threadID, &title)
	if err != nil {
		return 0, "", err
	}

	return threadID, title, nil
}

// GetRecentVideos returns recently added videos
func (s *AICompanionService) GetRecentVideos(limit int) ([]string, error) {
	if limit <= 0 || limit > 20 {
		limit = 10
	}

	query := `
		SELECT title, created_at
		FROM videos
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []string{}
	for rows.Next() {
		var title string
		var createdAt time.Time
		if err := rows.Scan(&title, &createdAt); err != nil {
			continue
		}

		timeAgo := formatDuration(time.Since(createdAt))
		results = append(results, fmt.Sprintf("📹 %s (%s ago)", title, timeAgo))
	}

	return results, nil
}

// AnalyzeStorageUsage provides insights into storage usage
func (s *AICompanionService) AnalyzeStorageUsage() (string, error) {
	var totalSize, videoCount int64
	var avgSize float64

	s.db.QueryRow("SELECT COUNT(*), COALESCE(SUM(file_size), 0), COALESCE(AVG(file_size), 0) FROM videos").Scan(&videoCount, &totalSize, &avgSize)

	// Get size by library
	libraryQuery := `
		SELECT l.name, COUNT(v.id) as count, COALESCE(SUM(v.file_size), 0) as size
		FROM libraries l
		LEFT JOIN videos v ON l.id = v.library_id
		GROUP BY l.id
		ORDER BY size DESC
		LIMIT 5
	`

	rows, err := s.db.Query(libraryQuery)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var response strings.Builder
	response.WriteString("💾 Storage Analysis:\n\n")
	response.WriteString(fmt.Sprintf("Total Videos: %d\n", videoCount))
	response.WriteString(fmt.Sprintf("Total Size: %.2f GB\n", float64(totalSize)/(1024*1024*1024)))
	response.WriteString(fmt.Sprintf("Average Video Size: %.2f MB\n\n", avgSize/(1024*1024)))

	response.WriteString("Top Libraries by Size:\n")
	for rows.Next() {
		var name string
		var count, size int64
		if err := rows.Scan(&name, &count, &size); err != nil {
			continue
		}
		response.WriteString(fmt.Sprintf("- %s: %d videos, %.2f GB\n", name, count, float64(size)/(1024*1024*1024)))
	}

	return response.String(), nil
}

// ================== Predictive Analytics & Insights ==================

// PredictLibraryGrowth analyzes library growth trends and predicts future growth
func (s *AICompanionService) PredictLibraryGrowth() (string, error) {
	// Get video creation timestamps
	query := `
		SELECT DATE(created_at) as date, COUNT(*) as count
		FROM videos
		WHERE created_at >= DATE('now', '-30 days')
		GROUP BY DATE(created_at)
		ORDER BY date DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	dailyCounts := []int{}
	for rows.Next() {
		var date string
		var count int
		if err := rows.Scan(&date, &count); err != nil {
			continue
		}
		dailyCounts = append(dailyCounts, count)
	}

	if len(dailyCounts) == 0 {
		return "📊 Not enough data to predict library growth. Add more videos over time to see trends!", nil
	}

	// Calculate average daily growth
	total := 0
	for _, count := range dailyCounts {
		total += count
	}
	avgPerDay := float64(total) / float64(len(dailyCounts))

	// Get total videos
	var totalVideos int
	s.db.QueryRow("SELECT COUNT(*) FROM videos").Scan(&totalVideos)

	// Predict next 30 days
	predicted30Days := int(avgPerDay * 30)
	predicted90Days := int(avgPerDay * 90)

	var response strings.Builder
	response.WriteString("📈 Library Growth Prediction:\n\n")
	response.WriteString(fmt.Sprintf("Current Library Size: %d videos\n", totalVideos))
	response.WriteString(fmt.Sprintf("Average Daily Growth: %.1f videos/day\n\n", avgPerDay))
	response.WriteString("Predictions:\n")
	response.WriteString(fmt.Sprintf("• In 30 days: ~%d videos (+%d)\n", totalVideos+predicted30Days, predicted30Days))
	response.WriteString(fmt.Sprintf("• In 90 days: ~%d videos (+%d)\n", totalVideos+predicted90Days, predicted90Days))

	if avgPerDay > 10 {
		response.WriteString("\n🚀 Your library is growing rapidly! Consider scheduling regular maintenance tasks.")
	} else if avgPerDay > 5 {
		response.WriteString("\n📊 Steady growth detected. Your library is expanding at a healthy pace.")
	} else if avgPerDay > 0 {
		response.WriteString("\n🌱 Slow but steady growth. Perfect for manageable library maintenance.")
	}

	return response.String(), nil
}

// AnalyzePerformerQuality provides quality scoring for performers based on metadata completeness
func (s *AICompanionService) AnalyzePerformerQuality() (string, error) {
	query := `
		SELECT
			p.name,
			CASE WHEN p.thumbnail_path IS NOT NULL AND p.thumbnail_path != '' THEN 1 ELSE 0 END as has_thumb,
			CASE WHEN p.preview_path IS NOT NULL AND p.preview_path != '' THEN 1 ELSE 0 END as has_preview,
			CASE WHEN p.metadata IS NOT NULL AND p.metadata != '{}' AND p.metadata != '' THEN 1 ELSE 0 END as has_metadata,
			COUNT(vp.video_id) as video_count
		FROM performers p
		LEFT JOIN video_performers vp ON p.id = vp.performer_id
		WHERE p.category = 'regular'
		GROUP BY p.id
		HAVING video_count > 0
		ORDER BY video_count DESC
		LIMIT 10
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	type PerformerScore struct {
		Name       string
		Score      int
		VideoCount int
	}

	performers := []PerformerScore{}
	for rows.Next() {
		var name string
		var hasThumb, hasPreview, hasMetadata, videoCount int
		if err := rows.Scan(&name, &hasThumb, &hasPreview, &hasMetadata, &videoCount); err != nil {
			continue
		}

		// Calculate quality score (out of 100)
		score := 0
		score += hasThumb * 30      // 30 points for thumbnail
		score += hasPreview * 30    // 30 points for preview
		score += hasMetadata * 40   // 40 points for metadata

		performers = append(performers, PerformerScore{
			Name:       name,
			Score:      score,
			VideoCount: videoCount,
		})
	}

	if len(performers) == 0 {
		return "No performer data available for quality analysis.", nil
	}

	var response strings.Builder
	response.WriteString("⭐ Top Performers - Quality Score:\n\n")

	for i, p := range performers {
		emoji := "❌"
		if p.Score >= 90 {
			emoji = "🌟"
		} else if p.Score >= 70 {
			emoji = "✅"
		} else if p.Score >= 50 {
			emoji = "⚠️"
		}

		response.WriteString(fmt.Sprintf("%d. %s %s - %d/100 (%d videos)\n", i+1, emoji, p.Name, p.Score, p.VideoCount))
	}

	return response.String(), nil
}

// GenerateInsights provides AI-powered insights about library usage and patterns
func (s *AICompanionService) GenerateInsights() ([]string, error) {
	insights := []string{}

	// Insight 1: Collection focus
	var topPerformerCount int
	s.db.QueryRow(`
		SELECT COUNT(vp.video_id)
		FROM video_performers vp
		INNER JOIN performers p ON vp.performer_id = p.id
		GROUP BY p.id
		ORDER BY COUNT(vp.video_id) DESC
		LIMIT 1
	`).Scan(&topPerformerCount)

	if topPerformerCount > 20 {
		insights = append(insights, fmt.Sprintf("🎯 Collection Focus: You have a strong collection focus with %d+ videos of your top performer!", topPerformerCount))
	}

	// Insight 2: Organization level
	var totalVideos, taggedVideos int
	s.db.QueryRow("SELECT COUNT(*) FROM videos").Scan(&totalVideos)
	s.db.QueryRow("SELECT COUNT(DISTINCT video_id) FROM video_tags").Scan(&taggedVideos)

	if totalVideos > 0 {
		orgLevel := float64(taggedVideos) / float64(totalVideos) * 100
		if orgLevel > 70 {
			insights = append(insights, fmt.Sprintf("🏆 Excellent Organization: %.0f%% of your videos are tagged!", orgLevel))
		} else if orgLevel < 20 && totalVideos > 50 {
			insights = append(insights, "📋 Organization Opportunity: Most videos are untagged. Try Smart Tagging for better discovery!")
		}
	}

	// Insight 3: Recent activity
	var recentScans int
	s.db.QueryRow(`
		SELECT COUNT(*)
		FROM activity_logs
		WHERE task_type LIKE '%scan%'
		AND created_at >= DATETIME('now', '-7 days')
	`).Scan(&recentScans)

	if recentScans > 5 {
		insights = append(insights, "🔄 Active Management: You're frequently scanning your library. Great job staying organized!")
	}

	// Insight 4: Storage efficiency
	var avgSize, totalSize int64
	s.db.QueryRow("SELECT COALESCE(AVG(file_size), 0), COALESCE(SUM(file_size), 0) FROM videos").Scan(&avgSize, &totalSize)

	if avgSize > 0 {
		avgGB := float64(avgSize) / (1024 * 1024 * 1024)
		if avgGB > 2 {
			insights = append(insights, fmt.Sprintf("💾 High Quality: Average video size is %.2f GB - you're collecting high-quality content!", avgGB))
		}
	}

	// Insight 5: Metadata richness
	var performersWithMetadata, totalPerformers int
	s.db.QueryRow("SELECT COUNT(*) FROM performers WHERE category = 'regular'").Scan(&totalPerformers)
	s.db.QueryRow("SELECT COUNT(*) FROM performers WHERE category = 'regular' AND metadata IS NOT NULL AND metadata != '{}' AND metadata != ''").Scan(&performersWithMetadata)

	if totalPerformers > 0 {
		metadataPercent := float64(performersWithMetadata) / float64(totalPerformers) * 100
		if metadataPercent > 80 {
			insights = append(insights, "📚 Rich Metadata: Your library has excellent metadata coverage!")
		}
	}

	if len(insights) == 0 {
		insights = append(insights, "🌱 Growing Library: Keep adding content and organizing your library to unlock more insights!")
	}

	return insights, nil
}

// ================== Duplicate Detection & Content Analysis ==================

// DetectPotentialDuplicates finds videos that might be duplicates based on various criteria
func (s *AICompanionService) DetectPotentialDuplicates() ([]string, error) {
	duplicates := []string{}

	// Find videos with very similar titles
	query := `
		SELECT v1.title, v2.title, v1.file_size, v2.file_size
		FROM videos v1
		INNER JOIN videos v2 ON v1.id < v2.id
		WHERE (
			-- Similar titles (case insensitive)
			LOWER(v1.title) = LOWER(v2.title)
			OR
			-- Same file size
			v1.file_size = v2.file_size
		)
		LIMIT 20
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var title1, title2 string
		var size1, size2 int64
		if err := rows.Scan(&title1, &title2, &size1, &size2); err != nil {
			continue
		}

		if title1 == title2 {
			duplicates = append(duplicates, fmt.Sprintf("📹 Duplicate Title: '%s' (%.2f MB)", title1, float64(size1)/(1024*1024)))
		} else if size1 == size2 {
			duplicates = append(duplicates, fmt.Sprintf("📏 Same Size: '%s' vs '%s' (%.2f MB)", title1, title2, float64(size1)/(1024*1024)))
		}
	}

	if len(duplicates) == 0 {
		return []string{"✅ No obvious duplicates detected!"}, nil
	}

	return duplicates, nil
}

// AnalyzeLibraryHealth provides a comprehensive health score
func (s *AICompanionService) AnalyzeLibraryHealth() (string, error) {
	var response strings.Builder
	response.WriteString("🏥 Library Health Report:\n\n")

	score := 100
	issues := []string{}

	// Check video coverage
	var totalVideos, videosWithThumbs, videosWithPreviews int
	s.db.QueryRow("SELECT COUNT(*) FROM videos").Scan(&totalVideos)
	s.db.QueryRow("SELECT COUNT(*) FROM videos WHERE thumbnail_path IS NOT NULL AND thumbnail_path != ''").Scan(&videosWithThumbs)
	s.db.QueryRow("SELECT COUNT(*) FROM videos WHERE preview_path IS NOT NULL AND preview_path != ''").Scan(&videosWithPreviews)

	thumbCoverage := 0.0
	if totalVideos > 0 {
		thumbCoverage = float64(videosWithThumbs) / float64(totalVideos) * 100
	}

	if thumbCoverage < 50 {
		score -= 20
		issues = append(issues, fmt.Sprintf("❌ Low thumbnail coverage: %.0f%%", thumbCoverage))
	} else if thumbCoverage < 80 {
		score -= 10
		issues = append(issues, fmt.Sprintf("⚠️ Medium thumbnail coverage: %.0f%%", thumbCoverage))
	} else {
		issues = append(issues, fmt.Sprintf("✅ Good thumbnail coverage: %.0f%%", thumbCoverage))
	}

	// Check performer metadata
	var totalPerformers, performersWithMetadata int
	s.db.QueryRow("SELECT COUNT(*) FROM performers WHERE category = 'regular'").Scan(&totalPerformers)
	s.db.QueryRow("SELECT COUNT(*) FROM performers WHERE category = 'regular' AND metadata IS NOT NULL AND metadata != '{}' AND metadata != ''").Scan(&performersWithMetadata)

	metadataCoverage := 0.0
	if totalPerformers > 0 {
		metadataCoverage = float64(performersWithMetadata) / float64(totalPerformers) * 100
	}

	if metadataCoverage < 30 {
		score -= 15
		issues = append(issues, fmt.Sprintf("❌ Low metadata coverage: %.0f%%", metadataCoverage))
	} else if metadataCoverage < 70 {
		score -= 5
		issues = append(issues, fmt.Sprintf("⚠️ Medium metadata coverage: %.0f%%", metadataCoverage))
	} else {
		issues = append(issues, fmt.Sprintf("✅ Good metadata coverage: %.0f%%", metadataCoverage))
	}

	// Check for recent errors
	consoleLogSvc := NewConsoleLogService()
	errors, _, _ := consoleLogSvc.GetAll(10, 0, "", "error", "")
	if len(errors) > 5 {
		score -= 15
		issues = append(issues, fmt.Sprintf("❌ %d recent errors detected", len(errors)))
	} else if len(errors) > 0 {
		score -= 5
		issues = append(issues, fmt.Sprintf("⚠️ %d minor errors", len(errors)))
	} else {
		issues = append(issues, "✅ No recent errors")
	}

	// Check organization (tags)
	var videoCount, taggedVideoCount int
	s.db.QueryRow("SELECT COUNT(*) FROM videos").Scan(&videoCount)
	s.db.QueryRow("SELECT COUNT(DISTINCT video_id) FROM video_tags").Scan(&taggedVideoCount)

	tagCoverage := 0.0
	if videoCount > 0 {
		tagCoverage = float64(taggedVideoCount) / float64(videoCount) * 100
	}

	if tagCoverage < 20 {
		score -= 10
		issues = append(issues, fmt.Sprintf("❌ Poor organization: %.0f%% videos tagged", tagCoverage))
	} else if tagCoverage < 50 {
		score -= 5
		issues = append(issues, fmt.Sprintf("⚠️ Moderate organization: %.0f%% videos tagged", tagCoverage))
	} else {
		issues = append(issues, fmt.Sprintf("✅ Well organized: %.0f%% videos tagged", tagCoverage))
	}

	// Determine health rating
	rating := "Excellent"
	emoji := "🌟"
	if score < 60 {
		rating = "Poor"
		emoji = "⚠️"
	} else if score < 75 {
		rating = "Fair"
		emoji = "📊"
	} else if score < 90 {
		rating = "Good"
		emoji = "👍"
	}

	response.WriteString(fmt.Sprintf("Overall Health: %s %d/100 %s\n\n", emoji, score, rating))
	response.WriteString("Details:\n")
	for _, issue := range issues {
		response.WriteString(fmt.Sprintf("• %s\n", issue))
	}

	if score < 90 {
		response.WriteString("\n💡 Run 'optimization suggestions' for improvement tips!")
	}

	return response.String(), nil
}

// RecommendContent suggests content based on library analysis
func (s *AICompanionService) RecommendContent() ([]string, error) {
	recommendations := []string{}

	// Find top performers
	query := `
		SELECT p.name, COUNT(vp.video_id) as video_count
		FROM performers p
		INNER JOIN video_performers vp ON p.id = vp.performer_id
		GROUP BY p.id
		ORDER BY video_count DESC
		LIMIT 3
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	topPerformers := []string{}
	for rows.Next() {
		var name string
		var count int
		if err := rows.Scan(&name, &count); err != nil {
			continue
		}
		topPerformers = append(topPerformers, name)
	}

	if len(topPerformers) > 0 {
		recommendations = append(recommendations, fmt.Sprintf("⭐ You seem to enjoy content featuring: %s", strings.Join(topPerformers, ", ")))
		recommendations = append(recommendations, fmt.Sprintf("💡 Tip: Search for more videos with these performers!"))
	}

	// Find popular tags
	tagQuery := `
		SELECT t.name, COUNT(vt.video_id) as usage_count
		FROM tags t
		INNER JOIN video_tags vt ON t.id = vt.tag_id
		GROUP BY t.id
		ORDER BY usage_count DESC
		LIMIT 3
	`

	tagRows, err := s.db.Query(tagQuery)
	if err != nil {
		return recommendations, nil
	}
	defer tagRows.Close()

	topTags := []string{}
	for tagRows.Next() {
		var name string
		var count int
		if err := tagRows.Scan(&name, &count); err != nil {
			continue
		}
		topTags = append(topTags, name)
	}

	if len(topTags) > 0 {
		recommendations = append(recommendations, fmt.Sprintf("🏷️ Your most used tags: %s", strings.Join(topTags, ", ")))
	}

	if len(recommendations) == 0 {
		return []string{"📚 Start organizing your library with tags and performers to get personalized recommendations!"}, nil
	}

	return recommendations, nil
}

// ================== Learning & Pattern Recognition ==================

// LearnFromUserAction records user actions to learn patterns
func (s *AICompanionService) LearnFromUserAction(actionType, context string, details map[string]interface{}) error {
	// Store as a memory for pattern recognition
	memory := &models.Memory{
		Key:        fmt.Sprintf("action_%s_%d", actionType, time.Now().Unix()),
		Value:      context,
		Category:   "user_action",
		Importance: 1, // Actions are low importance unless they reveal patterns
	}

	// Add details as JSON if provided
	if len(details) > 0 {
		detailsJSON, _ := json.Marshal(details)
		memory.Value = fmt.Sprintf("%s|%s", context, string(detailsJSON))
	}

	return s.SaveMemory(memory)
}

// DetectUsagePatterns analyzes user behavior and suggests workflows
func (s *AICompanionService) DetectUsagePatterns() ([]string, error) {
	patterns := []string{}

	// Check library scan frequency
	var lastScanTime time.Time
	err := s.db.QueryRow(`
		SELECT MAX(completed_at)
		FROM activity_logs
		WHERE task_type = 'video_scan' AND status = 'completed'
	`).Scan(&lastScanTime)

	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	if !lastScanTime.IsZero() {
		daysSinceLastScan := int(time.Since(lastScanTime).Hours() / 24)
		if daysSinceLastScan > 7 {
			patterns = append(patterns, fmt.Sprintf("📅 Pattern: It's been %d days since your last library scan. Consider scanning to find new content!", daysSinceLastScan))
		}
	}

	// Check if user frequently views certain categories
	var favoriteTagRows int
	s.db.QueryRow(`
		SELECT COUNT(DISTINCT tag_id)
		FROM video_tags
		GROUP BY tag_id
		HAVING COUNT(*) > 10
		LIMIT 1
	`).Scan(&favoriteTagRows)

	if favoriteTagRows > 0 {
		patterns = append(patterns, "🏷️ Pattern: You have well-organized tags! Consider using Smart Tagging to automatically categorize new videos.")
	}

	// Check performer engagement
	var performersViewed int
	s.db.QueryRow(`
		SELECT COUNT(*) FROM performers WHERE thumbnail_path IS NOT NULL AND thumbnail_path != ''
	`).Scan(&performersViewed)

	if performersViewed < 5 {
		patterns = append(patterns, "👤 Pattern: Limited performer thumbnails detected. Generate thumbnails for faster browsing!")
	}

	return patterns, nil
}

// SuggestNextAction provides contextual suggestions based on recent activity
func (s *AICompanionService) SuggestNextAction() (string, error) {
	// Check most recent completed task
	var recentTask string
	var recentTime time.Time

	err := s.db.QueryRow(`
		SELECT task_type, completed_at
		FROM activity_logs
		WHERE status = 'completed'
		ORDER BY completed_at DESC
		LIMIT 1
	`).Scan(&recentTask, &recentTime)

	if err == sql.ErrNoRows {
		return "💡 Start by scanning a library to index your videos!", nil
	}
	if err != nil {
		return "", err
	}

	// Suggest logical next steps based on recent task
	switch recentTask {
	case "video_scan", "library_scan":
		// Check if thumbnails need generation
		var videosWithoutThumbnails int
		s.db.QueryRow("SELECT COUNT(*) FROM videos WHERE thumbnail_path IS NULL OR thumbnail_path = ''").Scan(&videosWithoutThumbnails)

		if videosWithoutThumbnails > 20 {
			return fmt.Sprintf("💡 Next Step: You just scanned videos! Consider generating thumbnails for %d videos to improve browsing.", videosWithoutThumbnails), nil
		}
		return "✅ Library scan complete! You might want to check the Videos page to browse your content.", nil

	case "performer_thumbnail_generation":
		return "✅ Performer thumbnails generated! Visit the Performers page to see the improved loading speed.", nil

	case "metadata_fetch":
		return "✅ Metadata fetched! Your content is now enriched with detailed information.", nil

	default:
		return "💡 Everything looks good! Check 'optimization suggestions' for improvement ideas.", nil
	}
}

// ScheduleAutomatedTasks suggests optimal times for maintenance tasks
func (s *AICompanionService) ScheduleAutomatedTasks() ([]string, error) {
	suggestions := []string{}

	// Suggest library scan if not done recently
	var lastScan sql.NullTime
	s.db.QueryRow(`
		SELECT MAX(completed_at)
		FROM activity_logs
		WHERE task_type LIKE '%scan%' AND status = 'completed'
	`).Scan(&lastScan)

	if !lastScan.Valid || time.Since(lastScan.Time) > 7*24*time.Hour {
		suggestions = append(suggestions, "📅 Weekly Task: Schedule a library scan to index new content")
	}

	// Suggest thumbnail maintenance
	var thumbailCoverage float64
	s.db.QueryRow(`
		SELECT CAST(COUNT(CASE WHEN thumbnail_path IS NOT NULL AND thumbnail_path != '' THEN 1 END) AS REAL) / COUNT(*)
		FROM videos
	`).Scan(&thumbailCoverage)

	if thumbailCoverage < 0.8 {
		suggestions = append(suggestions, fmt.Sprintf("🖼️ Maintenance Task: Only %.0f%% of videos have thumbnails. Run thumbnail generation.", thumbailCoverage*100))
	}

	// Suggest metadata cleanup
	var performersNeedingMetadata int
	s.db.QueryRow(`
		SELECT COUNT(*) FROM performers
		WHERE category = 'regular' AND (metadata IS NULL OR metadata = '{}')
	`).Scan(&performersNeedingMetadata)

	if performersNeedingMetadata > 20 {
		suggestions = append(suggestions, fmt.Sprintf("📊 Maintenance Task: %d performers missing metadata. Fetch from AdultDataLink.", performersNeedingMetadata))
	}

	if len(suggestions) == 0 {
		return []string{"✅ No maintenance tasks needed! Your library is well maintained."}, nil
	}

	return suggestions, nil
}

// ================== Task Execution Functions ==================

// ExecuteLibraryScan triggers a library scan for a specific library
func (s *AICompanionService) ExecuteLibraryScan(libraryID int64) error {
	consoleLogSvc := NewConsoleLogService()
	consoleLogSvc.LogAICompanion("info", "AI Companion initiated library scan", map[string]interface{}{
		"library_id": libraryID,
		"trigger":    "ai_companion",
	})

	videoSvc := NewVideoService(NewActivityService(), NewLibraryService(), NewPerformerService())
	go func() {
		if err := videoSvc.ScanLibrary(libraryID); err != nil {
			consoleLogSvc.LogAICompanion("error", "AI Companion library scan failed", map[string]interface{}{
				"library_id": libraryID,
				"error":      err.Error(),
			})
		}
	}()

	return nil
}

// ExecutePerformerScan triggers a performer scan
func (s *AICompanionService) ExecutePerformerScan() error {
	consoleLogSvc := NewConsoleLogService()
	consoleLogSvc.LogAICompanion("info", "AI Companion initiated performer scan", map[string]interface{}{
		"trigger": "ai_companion",
	})

	performerScanSvc := NewPerformerScanService()
	go func() {
		if _, err := performerScanSvc.ScanPerformerFolders(); err != nil {
			consoleLogSvc.LogAICompanion("error", "AI Companion performer scan failed", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}()

	return nil
}

// SuggestOptimizations analyzes the system and suggests optimizations
func (s *AICompanionService) SuggestOptimizations() ([]string, error) {
	suggestions := []string{}

	// Check for performers without thumbnails
	var performersWithoutThumbnails int
	s.db.QueryRow("SELECT COUNT(*) FROM performers WHERE preview_path IS NOT NULL AND preview_path != '' AND (thumbnail_path IS NULL OR thumbnail_path = '')").Scan(&performersWithoutThumbnails)
	if performersWithoutThumbnails > 5 {
		suggestions = append(suggestions, fmt.Sprintf("⚡ Performance: Generate thumbnails for %d performers to improve page load times", performersWithoutThumbnails))
	}

	// Check for videos without thumbnails
	var videosWithoutThumbnails int
	s.db.QueryRow("SELECT COUNT(*) FROM videos WHERE thumbnail_path IS NULL OR thumbnail_path = ''").Scan(&videosWithoutThumbnails)
	if videosWithoutThumbnails > 20 {
		suggestions = append(suggestions, fmt.Sprintf("📸 Media: Generate thumbnails for %d videos for better browsing", videosWithoutThumbnails))
	}

	// Check for videos without previews
	var videosWithoutPreviews int
	s.db.QueryRow("SELECT COUNT(*) FROM videos WHERE preview_path IS NULL OR preview_path = ''").Scan(&videosWithoutPreviews)
	if videosWithoutPreviews > 50 {
		suggestions = append(suggestions, fmt.Sprintf("🎬 Media: Generate preview storyboards for %d videos", videosWithoutPreviews))
	}

	// Check for performers without metadata
	var performersWithoutMetadata int
	s.db.QueryRow("SELECT COUNT(*) FROM performers WHERE category = 'regular' AND (metadata IS NULL OR metadata = '{}' OR metadata = '')").Scan(&performersWithoutMetadata)
	if performersWithoutMetadata > 10 {
		suggestions = append(suggestions, fmt.Sprintf("📊 Metadata: Fetch metadata for %d performers from AdultDataLink", performersWithoutMetadata))
	}

	// Check console logs for recent errors
	consoleLogSvc := NewConsoleLogService()
	errors, _, _ := consoleLogSvc.GetAll(10, 0, "", "error", "")
	if len(errors) > 0 {
		suggestions = append(suggestions, fmt.Sprintf("🚨 Errors: %d recent errors detected - run 'check for errors' to analyze", len(errors)))
	}

	if len(suggestions) == 0 {
		suggestions = append(suggestions, "✅ Your system is running smoothly! No optimizations needed at this time.")
	}

	return suggestions, nil
}

// ================== Console Log Integration ==================

// GetRecentConsoleLogs retrieves recent console logs with optional filtering
func (s *AICompanionService) GetRecentConsoleLogs(limit int, source, level string) ([]*models.ConsoleLog, error) {
	consoleLogSvc := NewConsoleLogService()
	logs, _, err := consoleLogSvc.GetAll(limit, 0, source, level, "")
	return logs, err
}

// AnalyzeConsoleErrors checks console logs for errors and provides insights
func (s *AICompanionService) AnalyzeConsoleErrors() (string, error) {
	consoleLogSvc := NewConsoleLogService()

	// Get recent errors
	errors, _, err := consoleLogSvc.GetAll(50, 0, "", "error", "")
	if err != nil {
		return "", fmt.Errorf("failed to retrieve error logs: %w", err)
	}

	if len(errors) == 0 {
		return "✅ No recent errors detected! Your system is running smoothly.", nil
	}

	// Analyze error patterns
	errorPatterns := make(map[string]int)
	sources := make(map[string]int)

	for _, log := range errors {
		// Count by source
		sources[log.Source]++

		// Categorize errors
		msg := strings.ToLower(log.Message)
		if strings.Contains(msg, "library") || strings.Contains(msg, "scan") {
			errorPatterns["library_scan"]++
		} else if strings.Contains(msg, "scraper") || strings.Contains(msg, "thread") {
			errorPatterns["scraper"]++
		} else if strings.Contains(msg, "database") || strings.Contains(msg, "sql") {
			errorPatterns["database"]++
		} else if strings.Contains(msg, "api") || strings.Contains(msg, "request") {
			errorPatterns["api"]++
		} else {
			errorPatterns["other"]++
		}
	}

	var response strings.Builder
	response.WriteString(fmt.Sprintf("⚠️ Found %d recent errors. Analysis:\n\n", len(errors)))

	response.WriteString("Error Sources:\n")
	for source, count := range sources {
		response.WriteString(fmt.Sprintf("- %s: %d errors\n", source, count))
	}

	response.WriteString("\nError Categories:\n")
	for category, count := range errorPatterns {
		response.WriteString(fmt.Sprintf("- %s: %d errors\n", category, count))
	}

	// Provide recommendations
	if errorPatterns["library_scan"] > 0 {
		response.WriteString("\n💡 Recommendation: Library scan errors detected. Check library paths and permissions.")
	}
	if errorPatterns["scraper"] > 0 {
		response.WriteString("\n💡 Recommendation: Scraper errors detected. Check network connectivity and target URLs.")
	}
	if errorPatterns["database"] > 0 {
		response.WriteString("\n💡 Recommendation: Database errors detected. Check database integrity and disk space.")
	}

	return response.String(), nil
}

// MonitorConsoleLogsForErrors periodically checks for new errors
func (s *AICompanionService) MonitorConsoleLogsForErrors() {
	ticker := time.NewTicker(2 * time.Minute) // Check every 2 minutes
	defer ticker.Stop()

	lastErrorCount := 0

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			consoleLogSvc := NewConsoleLogService()
			errors, _, err := consoleLogSvc.GetAll(10, 0, "", "error", "")
			if err != nil {
				continue
			}

			// Alert if new errors appeared
			if len(errors) > lastErrorCount && len(errors) > 0 {
				newErrors := len(errors) - lastErrorCount

				// Get the most recent error
				recentError := errors[0]

				s.emitEvent(CompanionEvent{
					Type:    "notification",
					Source:  "error_monitor",
					Message: fmt.Sprintf("🚨 %d new error(s) detected! Latest: %s", newErrors, recentError.Message),
					Data: map[string]interface{}{
						"new_errors":    newErrors,
						"total_errors":  len(errors),
						"latest_source": recentError.Source,
						"latest_level":  recentError.Level,
					},
					Severity:  "error",
					Timestamp: time.Now(),
				})
			}

			lastErrorCount = len(errors)
		}
	}
}

// Chat processes user messages with rule-based intelligence
func (s *AICompanionService) Chat(message string, history []map[string]string) (string, error) {
	messageLower := strings.ToLower(message)

	// Rule-based responses for common queries
	if strings.Contains(messageLower, "how many videos") {
		var count int
		if err := s.db.QueryRow("SELECT COUNT(*) FROM videos").Scan(&count); err != nil {
			return "", err
		}
		return fmt.Sprintf("You have %d videos in your library.", count), nil
	}

	if strings.Contains(messageLower, "how many performers") {
		var count int
		if err := s.db.QueryRow("SELECT COUNT(*) FROM performers").Scan(&count); err != nil {
			return "", err
		}
		return fmt.Sprintf("You have %d performers in your library.", count), nil
	}

	// Top performers query
	if strings.Contains(messageLower, "top performers") || strings.Contains(messageLower, "most videos") {
		query := `
			SELECT p.name, COUNT(vp.video_id) as video_count
			FROM performers p
			LEFT JOIN video_performers vp ON p.id = vp.performer_id
			GROUP BY p.id
			ORDER BY video_count DESC
			LIMIT 10
		`
		rows, err := s.db.Query(query)
		if err != nil {
			return "", err
		}
		defer rows.Close()

		var result strings.Builder
		result.WriteString("Top 10 Performers:\n")
		count := 1
		for rows.Next() {
			var name string
			var videoCount int
			if err := rows.Scan(&name, &videoCount); err != nil {
				continue
			}
			result.WriteString(fmt.Sprintf("%d. %s (%d videos)\n", count, name, videoCount))
			count++
		}
		return result.String(), nil
	}

	// Console log queries
	if strings.Contains(messageLower, "console log") || strings.Contains(messageLower, "logs") && strings.Contains(messageLower, "error") {
		if strings.Contains(messageLower, "error") || strings.Contains(messageLower, "analyze") {
			return s.AnalyzeConsoleErrors()
		}

		// General console log stats
		consoleLogSvc := NewConsoleLogService()
		stats, err := consoleLogSvc.GetStats()
		if err != nil {
			return "Failed to retrieve console log statistics.", err
		}

		var response strings.Builder
		response.WriteString("📋 Console Log Statistics:\n\n")

		if totalLogs, ok := stats["total_logs"].(int64); ok {
			response.WriteString(fmt.Sprintf("Total Logs: %d\n", totalLogs))
		}
		if bySource, ok := stats["by_source"].(map[string]interface{}); ok {
			response.WriteString("\nBy Source:\n")
			for source, count := range bySource {
				response.WriteString(fmt.Sprintf("- %s: %v\n", source, count))
			}
		}
		if byLevel, ok := stats["by_level"].(map[string]interface{}); ok {
			response.WriteString("\nBy Level:\n")
			for level, count := range byLevel {
				response.WriteString(fmt.Sprintf("- %s: %v\n", level, count))
			}
		}

		return response.String(), nil
	}

	// Check for errors command
	if strings.Contains(messageLower, "check") && strings.Contains(messageLower, "error") {
		return s.AnalyzeConsoleErrors()
	}

	// Recent activity from console logs
	if strings.Contains(messageLower, "recent activity") || strings.Contains(messageLower, "what happened") {
		consoleLogSvc := NewConsoleLogService()
		recentLogs, _, err := consoleLogSvc.GetAll(10, 0, "", "info", "")
		if err != nil {
			return "Failed to retrieve recent activity.", err
		}

		if len(recentLogs) == 0 {
			return "No recent activity to report.", nil
		}

		var response strings.Builder
		response.WriteString("📝 Recent Activity:\n\n")

		for i, log := range recentLogs {
			if i >= 5 {
				break
			}
			timeAgo := time.Since(log.CreatedAt)
			response.WriteString(fmt.Sprintf("%d. [%s] %s (%s ago)\n",
				i+1, log.Source, log.Message, formatDuration(timeAgo)))
		}

		return response.String(), nil
	}

	// Search forum posts for specific keywords
	if (strings.Contains(messageLower, "post") || strings.Contains(messageLower, "thread")) &&
		(strings.Contains(messageLower, "contain") || strings.Contains(messageLower, "include") ||
			strings.Contains(messageLower, "have") || strings.Contains(messageLower, "with") ||
			strings.Contains(messageLower, "mention") || strings.Contains(messageLower, "about")) {

		// Extract thread name if specified
		var threadName string
		var searchKeyword string

		// Look for quoted text first (thread name or keyword)
		re := regexp.MustCompile(`["']([^"']+)["']`)
		matches := re.FindAllStringSubmatch(message, -1)

		if len(matches) >= 1 {
			// First quoted text is likely the thread name
			threadName = matches[0][1]
		}
		if len(matches) >= 2 {
			// Second quoted text is likely the search keyword
			searchKeyword = matches[1][1]
		} else if len(matches) == 1 {
			// Only one quoted text - assume it's the keyword
			searchKeyword = matches[0][1]
			threadName = ""
		}

		// If no quotes, try to extract keyword from common patterns
		if searchKeyword == "" {
			words := strings.Fields(message)
			for i, word := range words {
				wordLower := strings.ToLower(word)
				if (wordLower == "containing" || wordLower == "include" || wordLower == "with" ||
					wordLower == "mention" || wordLower == "about" || wordLower == "have") && i+1 < len(words) {
					nextWord := words[i+1]
					// Remove punctuation
					searchKeyword = strings.TrimRight(nextWord, "?.!")
					break
				}
			}
		}

		if searchKeyword == "" {
			return "Please specify what to search for. Try: 'how many posts contain \"BBC\"' or 'search thread \"Marsha May\" for \"IR\"'", nil
		}

		var response strings.Builder

		// If thread name specified, search within that thread
		if threadName != "" {
			threadID, threadTitle, err := s.GetThreadByName(threadName)
			if err != nil {
				return fmt.Sprintf("Thread '%s' not found. Try checking the exact thread name on the Scraper page.", threadName), nil
			}

			count, err := s.CountPostsWithKeyword(searchKeyword, threadID)
			if err != nil {
				return "Failed to search posts.", err
			}

			response.WriteString(fmt.Sprintf("🔍 Search Results\n\n"))
			response.WriteString(fmt.Sprintf("Thread: \"%s\"\n", threadTitle))
			response.WriteString(fmt.Sprintf("Keyword: \"%s\"\n\n", searchKeyword))
			response.WriteString(fmt.Sprintf("**Found %d posts** containing this keyword.\n\n", count))

			// Show sample results if count is reasonable
			if count > 0 && count <= 20 {
				results, _ := s.SearchThreadPosts(searchKeyword, threadID)
				response.WriteString("Sample matches:\n\n")
				for i, result := range results {
					if i >= 5 {
						response.WriteString(fmt.Sprintf("_... and %d more posts_\n", count-5))
						break
					}
					response.WriteString(fmt.Sprintf("**Post #%d** by %s:\n%s\n\n",
						result["post_number"], result["author"], result["excerpt"]))
				}
			}

			return response.String(), nil
		}

		// Search across all threads
		count, err := s.CountPostsWithKeyword(searchKeyword)
		if err != nil {
			return "Failed to search posts.", err
		}

		response.WriteString(fmt.Sprintf("🔍 Global Search Results\n\n"))
		response.WriteString(fmt.Sprintf("Keyword: \"%s\"\n\n", searchKeyword))
		response.WriteString(fmt.Sprintf("**Found %d posts** across all scraped threads.\n\n", count))

		if count > 0 && count <= 20 {
			results, _ := s.SearchThreadPosts(searchKeyword)
			response.WriteString("Recent matches:\n\n")
			threadCounts := make(map[string]int)
			for _, result := range results {
				threadTitle := result["thread_title"].(string)
				threadCounts[threadTitle]++
			}

			response.WriteString("Breakdown by thread:\n")
			for thread, cnt := range threadCounts {
				response.WriteString(fmt.Sprintf("• %s: %d posts\n", thread, cnt))
			}
		}

		return response.String(), nil
	}

	// Scraper stats
	if strings.Contains(messageLower, "scraper") || strings.Contains(messageLower, "scraped") || strings.Contains(messageLower, "forum") {
		var threadCount, postCount, linkCount, activeLinks int
		s.db.QueryRow("SELECT COUNT(*) FROM scraped_threads").Scan(&threadCount)
		s.db.QueryRow("SELECT COUNT(*) FROM scraped_posts").Scan(&postCount)
		s.db.QueryRow("SELECT COUNT(*) FROM scraped_download_links").Scan(&linkCount)
		s.db.QueryRow("SELECT COUNT(*) FROM scraped_download_links WHERE status = 'active'").Scan(&activeLinks)

		return fmt.Sprintf("🕷️ Web Scraper Statistics:\n\n"+
			"📰 Threads Scraped: %d\n"+
			"💬 Posts Collected: %d\n"+
			"🔗 Download Links: %d\n"+
			"✅ Active Links: %d\n\n"+
			"💡 Tip: Use 'Auto-Link Performers' to match threads with performers!",
			threadCount, postCount, linkCount, activeLinks), nil
	}

	// Search for videos
	if (strings.Contains(messageLower, "find") || strings.Contains(messageLower, "search")) && strings.Contains(messageLower, "video") {
		// Extract search term (simple extraction - everything after "for" or "video")
		words := strings.Fields(message)
		searchTerm := ""
		for i, word := range words {
			if strings.ToLower(word) == "for" && i+1 < len(words) {
				searchTerm = strings.Join(words[i+1:], " ")
				break
			} else if strings.ToLower(word) == "video" && i+1 < len(words) {
				searchTerm = strings.Join(words[i+1:], " ")
				break
			}
		}

		if searchTerm == "" {
			return "Please specify what video you're looking for. Try: 'search for video [name]'", nil
		}

		results, err := s.SearchVideos(searchTerm)
		if err != nil {
			return "Failed to search videos.", err
		}

		var response strings.Builder
		response.WriteString(fmt.Sprintf("Search results for '%s':\n\n", searchTerm))
		for _, result := range results {
			response.WriteString(result + "\n")
		}

		return response.String(), nil
	}

	// Find performer
	if (strings.Contains(messageLower, "find") || strings.Contains(messageLower, "search") || strings.Contains(messageLower, "who is")) && strings.Contains(messageLower, "performer") {
		// Extract performer name
		words := strings.Fields(message)
		performerName := ""
		for i, word := range words {
			if (strings.ToLower(word) == "performer" || strings.ToLower(word) == "is") && i+1 < len(words) {
				performerName = strings.Join(words[i+1:], " ")
				break
			}
		}

		if performerName == "" {
			return "Please specify which performer you're looking for. Try: 'find performer [name]'", nil
		}

		return s.FindPerformer(performerName)
	}

	// Recent videos
	if strings.Contains(messageLower, "recent") && strings.Contains(messageLower, "video") {
		results, err := s.GetRecentVideos(10)
		if err != nil {
			return "Failed to retrieve recent videos.", err
		}

		var response strings.Builder
		response.WriteString("📅 Recently Added Videos:\n\n")
		for _, result := range results {
			response.WriteString(result + "\n")
		}

		return response.String(), nil
	}

	// Tagged videos
	if strings.Contains(messageLower, "tag") && (strings.Contains(messageLower, "video") || strings.Contains(messageLower, "with")) {
		// Extract tag name
		words := strings.Fields(message)
		tagName := ""
		for i, word := range words {
			if strings.ToLower(word) == "tag" && i+1 < len(words) {
				tagName = strings.Join(words[i+1:], " ")
				tagName = strings.TrimSuffix(tagName, " videos")
				tagName = strings.TrimSuffix(tagName, " video")
				break
			}
		}

		if tagName == "" {
			return "Please specify which tag you're looking for. Try: 'videos with tag [name]'", nil
		}

		results, err := s.GetTaggedVideos(tagName)
		if err != nil {
			return "Failed to search tagged videos.", err
		}

		var response strings.Builder
		response.WriteString(fmt.Sprintf("Videos tagged '%s':\n\n", tagName))
		for _, result := range results {
			response.WriteString(result + "\n")
		}

		return response.String(), nil
	}

	// Storage analysis
	if strings.Contains(messageLower, "storage") || strings.Contains(messageLower, "disk space") || strings.Contains(messageLower, "how much space") {
		return s.AnalyzeStorageUsage()
	}

	// Pattern detection
	if strings.Contains(messageLower, "pattern") || strings.Contains(messageLower, "usage") && strings.Contains(messageLower, "habit") {
		patterns, err := s.DetectUsagePatterns()
		if err != nil {
			return "Failed to analyze usage patterns.", err
		}

		if len(patterns) == 0 {
			return "I haven't detected any specific usage patterns yet. Keep using the system and I'll learn your preferences!", nil
		}

		var response strings.Builder
		response.WriteString("🧠 Detected Usage Patterns:\n\n")
		for i, pattern := range patterns {
			response.WriteString(fmt.Sprintf("%d. %s\n", i+1, pattern))
		}

		return response.String(), nil
	}

	// Next action suggestion
	if strings.Contains(messageLower, "what should i do") || strings.Contains(messageLower, "what next") || strings.Contains(messageLower, "next step") {
		return s.SuggestNextAction()
	}

	// Duplicate detection
	if strings.Contains(messageLower, "duplicate") || strings.Contains(messageLower, "find duplicates") {
		duplicates, err := s.DetectPotentialDuplicates()
		if err != nil {
			return "Failed to detect duplicates.", err
		}

		var response strings.Builder
		response.WriteString("🔍 Duplicate Detection Results:\n\n")
		for i, dup := range duplicates {
			if i >= 10 {
				response.WriteString(fmt.Sprintf("\n... and %d more potential duplicates", len(duplicates)-10))
				break
			}
			response.WriteString(fmt.Sprintf("%d. %s\n", i+1, dup))
		}

		return response.String(), nil
	}

	// Library health check
	if strings.Contains(messageLower, "health") || strings.Contains(messageLower, "health check") || strings.Contains(messageLower, "health score") {
		return s.AnalyzeLibraryHealth()
	}

	// Content recommendations
	if strings.Contains(messageLower, "recommend") && !strings.Contains(messageLower, "optimization") {
		recommendations, err := s.RecommendContent()
		if err != nil {
			return "Failed to generate recommendations.", err
		}

		var response strings.Builder
		response.WriteString("💝 Personalized Recommendations:\n\n")
		for _, rec := range recommendations {
			response.WriteString(fmt.Sprintf("• %s\n", rec))
		}

		return response.String(), nil
	}

	// Maintenance schedule
	if strings.Contains(messageLower, "maintenance") || strings.Contains(messageLower, "schedule") || strings.Contains(messageLower, "task") && strings.Contains(messageLower, "list") {
		tasks, err := s.ScheduleAutomatedTasks()
		if err != nil {
			return "Failed to generate maintenance schedule.", err
		}

		var response strings.Builder
		response.WriteString("🗓️ Recommended Maintenance Tasks:\n\n")
		for i, task := range tasks {
			response.WriteString(fmt.Sprintf("%d. %s\n", i+1, task))
		}
		response.WriteString("\nTip: I'll automatically remind you when these tasks become important!")

		return response.String(), nil
	}

	// Performer information query
	if strings.Contains(messageLower, "performer") && strings.Contains(messageLower, "information") {
		// Extract performer name from message (simple extraction)
		// This would be better with LLM, but we'll do basic pattern matching
		words := strings.Fields(message)
		for i, word := range words {
			if strings.ToLower(word) == "performer" && i+1 < len(words) {
				performerName := strings.Trim(words[i+1], `"'`)

				query := `
					SELECT p.name, COUNT(vp.video_id) as video_count
					FROM performers p
					LEFT JOIN video_performers vp ON p.id = vp.performer_id
					WHERE p.name LIKE ?
					GROUP BY p.id
					LIMIT 1
				`
				var name string
				var videoCount int
				err := s.db.QueryRow(query, "%"+performerName+"%").Scan(&name, &videoCount)
				if err != nil {
					return fmt.Sprintf("I couldn't find information about performer '%s'. Try asking about 'top performers' instead.", performerName), nil
				}
				return fmt.Sprintf("Performer: %s\nVideos: %d", name, videoCount), nil
			}
		}
	}

	// Thumbnail status queries
	if strings.Contains(messageLower, "thumbnail") && (strings.Contains(messageLower, "status") || strings.Contains(messageLower, "coverage")) {
		var performersWithThumbnails, performersWithPreviews, performersWithoutThumbnails int
		s.db.QueryRow("SELECT COUNT(*) FROM performers WHERE thumbnail_path IS NOT NULL AND thumbnail_path != ''").Scan(&performersWithThumbnails)
		s.db.QueryRow("SELECT COUNT(*) FROM performers WHERE preview_path IS NOT NULL AND preview_path != ''").Scan(&performersWithPreviews)
		s.db.QueryRow("SELECT COUNT(*) FROM performers WHERE preview_path IS NOT NULL AND preview_path != '' AND (thumbnail_path IS NULL OR thumbnail_path = '')").Scan(&performersWithoutThumbnails)

		coverage := float64(0)
		if performersWithPreviews > 0 {
			coverage = float64(performersWithThumbnails) / float64(performersWithPreviews) * 100
		}

		var response strings.Builder
		response.WriteString("📸 Performer Thumbnail Status:\n")
		response.WriteString(fmt.Sprintf("- Performers with thumbnails: %d/%d (%.1f%%)\n", performersWithThumbnails, performersWithPreviews, coverage))
		if performersWithoutThumbnails > 0 {
			response.WriteString(fmt.Sprintf("- ⚡ Missing thumbnails: %d performers\n", performersWithoutThumbnails))
			response.WriteString("\nTip: Generate thumbnails on the Tasks page to improve page load performance!")
		} else {
			response.WriteString("\n✅ All performers with previews have thumbnails!")
		}
		return response.String(), nil
	}

	// Optimization suggestions
	if strings.Contains(messageLower, "optimize") || strings.Contains(messageLower, "suggestion") || strings.Contains(messageLower, "recommend") {
		suggestions, err := s.SuggestOptimizations()
		if err != nil {
			return "Failed to analyze system for optimizations.", err
		}

		var response strings.Builder
		response.WriteString("💡 System Optimization Suggestions:\n\n")
		for i, suggestion := range suggestions {
			response.WriteString(fmt.Sprintf("%d. %s\n", i+1, suggestion))
		}
		response.WriteString("\nTip: Visit the Tasks page to execute these optimizations!")

		return response.String(), nil
	}

	// Performance suggestions
	if strings.Contains(messageLower, "performance") || strings.Contains(messageLower, "slow") || strings.Contains(messageLower, "faster") {
		var performersWithoutThumbnails int
		s.db.QueryRow("SELECT COUNT(*) FROM performers WHERE preview_path IS NOT NULL AND preview_path != '' AND (thumbnail_path IS NULL OR thumbnail_path = '')").Scan(&performersWithoutThumbnails)

		if performersWithoutThumbnails > 0 {
			message := fmt.Sprintf("I've detected a performance optimization opportunity! You have %d performers with preview videos but no thumbnails. Generating thumbnails will significantly speed up the Performers page. Head to the Tasks page and click 'Generate Performer Thumbnails' to improve performance.", performersWithoutThumbnails)

			// Track this recommendation so we can react when user follows it
			s.trackRecommendation(
				"chat_performance_suggestion",
				"performer_thumbnail_generation",
				message,
				map[string]interface{}{
					"performers_without_thumbnails": performersWithoutThumbnails,
					"suggested_via":                 "chat",
				},
			)

			return message, nil
		}
		return "Your library is well-optimized! All performers with previews have thumbnails for fast loading.", nil
	}

	// Library statistics
	if strings.Contains(messageLower, "library stats") || strings.Contains(messageLower, "library statistics") || strings.Contains(messageLower, "overview") {
		var videoCount, performerCount, studioCount, tagCount int
		var performersWithThumbnails, performersWithPreviews int
		var videosWithPreviews int

		s.db.QueryRow("SELECT COUNT(*) FROM videos").Scan(&videoCount)
		s.db.QueryRow("SELECT COUNT(*) FROM performers").Scan(&performerCount)
		s.db.QueryRow("SELECT COUNT(*) FROM studios").Scan(&studioCount)
		s.db.QueryRow("SELECT COUNT(*) FROM tags").Scan(&tagCount)
		s.db.QueryRow("SELECT COUNT(*) FROM performers WHERE thumbnail_path IS NOT NULL AND thumbnail_path != ''").Scan(&performersWithThumbnails)
		s.db.QueryRow("SELECT COUNT(*) FROM performers WHERE preview_path IS NOT NULL AND preview_path != ''").Scan(&performersWithPreviews)
		s.db.QueryRow("SELECT COUNT(*) FROM videos WHERE preview_path IS NOT NULL AND preview_path != ''").Scan(&videosWithPreviews)

		thumbnailCoverage := float64(0)
		if performersWithPreviews > 0 {
			thumbnailCoverage = float64(performersWithThumbnails) / float64(performersWithPreviews) * 100
		}
		videoPreviewCoverage := float64(0)
		if videoCount > 0 {
			videoPreviewCoverage = float64(videosWithPreviews) / float64(videoCount) * 100
		}

		var response strings.Builder
		response.WriteString("📊 Library Overview:\n\n")
		response.WriteString("Content:\n")
		response.WriteString(fmt.Sprintf("- Videos: %d\n", videoCount))
		response.WriteString(fmt.Sprintf("- Performers: %d\n", performerCount))
		response.WriteString(fmt.Sprintf("- Studios: %d\n", studioCount))
		response.WriteString(fmt.Sprintf("- Tags: %d\n\n", tagCount))
		response.WriteString("Media Coverage:\n")
		response.WriteString(fmt.Sprintf("- Video previews: %.1f%%\n", videoPreviewCoverage))
		response.WriteString(fmt.Sprintf("- Performer thumbnails: %.1f%%\n", thumbnailCoverage))

		return response.String(), nil
	}

	// Predict library growth
	if strings.Contains(messageLower, "predict") || strings.Contains(messageLower, "growth") || strings.Contains(messageLower, "library growth") {
		return s.PredictLibraryGrowth()
	}

	// Performer quality analysis
	if strings.Contains(messageLower, "quality") && strings.Contains(messageLower, "performer") || strings.Contains(messageLower, "performer quality") {
		return s.AnalyzePerformerQuality()
	}

	// Generate insights
	if strings.Contains(messageLower, "insight") || strings.Contains(messageLower, "show insights") {
		insights, err := s.GenerateInsights()
		if err != nil {
			return "Failed to generate insights.", err
		}

		var response strings.Builder
		response.WriteString("💡 AI-Powered Insights:\n\n")
		for i, insight := range insights {
			response.WriteString(fmt.Sprintf("%d. %s\n", i+1, insight))
		}

		if len(insights) == 0 {
			response.WriteString("I haven't gathered enough data to provide insights yet. Keep using the system!")
		}

		return response.String(), nil
	}

	// AI Task Commands - Auto-link performers
	if strings.Contains(messageLower, "auto") && (strings.Contains(messageLower, "link") || strings.Contains(messageLower, "performers")) {
		return `## 🔗 Auto-Link Performers

I can automatically detect and link performers to videos based on filename analysis!

**What I do:**
- Scan video filenames for performer names
- Calculate confidence scores for each match
- Auto-apply 100% confidence matches
- Provide suggestions for manual review

**To run this task:**
Navigate to the **AI Page** and click "Start Analysis" in the "Auto-Link Performers" card.

**Expected results:**
- Videos analyzed
- Matches found with confidence scores
- Automatic links applied for high-confidence matches

Would you like me to explain any other AI capabilities?`, nil
	}

	// AI Task Commands - Smart Tagging
	if strings.Contains(messageLower, "smart") && strings.Contains(messageLower, "tag") || strings.Contains(messageLower, "suggest tags") {
		return `## 🏷️ Smart Tagging

I can suggest relevant tags for your videos based on intelligent analysis!

**What I analyze:**
- Video titles and filenames
- Performer information
- Existing tag patterns
- Content keywords

**To run this task:**
Go to the **AI Page** and click "Start Analysis" in the "Smart Tagging" card.

**Features:**
- High confidence tags (85%+) auto-applied
- Tag suggestions with reasoning
- Bulk tag application

Try asking: "What tags should I use?" or "How do I organize my videos with tags?"`, nil
	}

	// AI Task Commands - Duplicate Detection
	if (strings.Contains(messageLower, "find") || strings.Contains(messageLower, "detect")) && strings.Contains(messageLower, "duplicate") {
		return `## 🔍 Duplicate Detection

I can find duplicate and similar videos across your library!

**Detection methods:**
- Exact file size matches
- Similar titles
- Identical durations
- Content fingerprinting

**What you'll get:**
- Duplicate groups with similarity scores
- File comparison details
- Recommendations on which to keep
- Easy deletion options

**To run this task:**
Visit the **AI Page** and use the "Duplicate Detection" feature.

**Tip:** I compare videos by size, title, and metadata to ensure accuracy!`, nil
	}

	// AI Task Commands - Scene Detection
	if strings.Contains(messageLower, "scene") && (strings.Contains(messageLower, "detect") || strings.Contains(messageLower, "detection")) {
		return `## 🎬 Scene Detection

I can analyze videos to detect scene changes and create chapter markers!

**Capabilities:**
- Detect scene transitions
- Identify chapter boundaries
- Timestamp each scene
- Categorize scene types

**Use cases:**
- Breaking down long videos
- Creating chapter navigation
- Content analysis

**To activate:**
Go to the **AI Page** → "Scene Detection" card → Click "Start Analysis"

This is perfect for multi-scene content!`, nil
	}

	// AI Task Commands - Quality Analysis
	if (strings.Contains(messageLower, "quality") && strings.Contains(messageLower, "analyz")) || strings.Contains(messageLower, "video quality") {
		return `## 📊 Video Quality Analysis

I can analyze video technical quality and identify issues!

**What I check:**
- **Resolution** - 4K, 1080p, 720p, etc.
- **Bitrate** - Video encoding quality
- **Codec** - Video compression format
- **Frame rate** - Smoothness
- **File integrity** - Corruption detection

**Issues detected:**
- Low resolution videos
- Poor encoding quality
- Corrupted files
- Suboptimal bitrates

**To run:**
Navigate to **AI Page** → "Quality Analysis" → "Start Analysis"

**Results:** Detailed quality scores and actionable recommendations!`, nil
	}

	// AI Task Commands - Content Classification
	if strings.Contains(messageLower, "classif") || strings.Contains(messageLower, "categoriz") {
		return `## 📁 Content Classification

I can automatically categorize your videos into intelligent groups!

**Classification criteria:**
- Content type analysis
- Performer patterns
- Tag associations
- Studio organization
- Genre detection

**Benefits:**
- Better organization
- Easier searching
- Pattern recognition
- Collection insights

**To use:**
**AI Page** → "Content Classification" → "Start Analysis"

I'll analyze your entire library and suggest categories!`, nil
	}

	// AI Task Commands - Auto-Naming
	if strings.Contains(messageLower, "auto") && strings.Contains(messageLower, "nam") || strings.Contains(messageLower, "rename") {
		return `## ✍️ Auto-Naming Suggestions

I can generate better filenames based on video metadata!

**Naming scheme includes:**
- Performer names
- Studio name
- Production date
- Content tags
- Sequential numbering

**Example format:**
` + "`[Studio] Performer1_Performer2 - Title (YYYY-MM-DD)`" + `

**To generate suggestions:**
**AI Page** → "Auto-Naming" → "Start Analysis"

**Features:**
- Consistent naming across library
- Metadata-driven names
- Batch renaming support
- Preview before applying

Keep your library organized with intelligent naming!`, nil
	}

	// What can you do
	if strings.Contains(messageLower, "what can you do") || strings.Contains(messageLower, "help") && strings.Contains(messageLower, "companion") {
		var response strings.Builder
		response.WriteString("🤖 I'm your AI Companion! Here's what I can help with:\n\n")
		response.WriteString("📊 Information & Stats:\n")
		response.WriteString("- Library stats, video counts, performer counts\n")
		response.WriteString("- Thumbnail coverage and performance analysis\n")
		response.WriteString("- Top performers list\n")
		response.WriteString("- Scraper statistics and forum data\n")
		response.WriteString("- Storage usage and disk space analysis\n\n")
		response.WriteString("🔍 Smart Search:\n")
		response.WriteString("- 'Search for video [name]' - Find videos\n")
		response.WriteString("- 'Find performer [name]' - Performer details\n")
		response.WriteString("- 'Videos with tag [name]' - Tagged content\n")
		response.WriteString("- 'Recent videos' - Latest additions\n\n")
		response.WriteString("📋 Console Logs & Monitoring:\n")
		response.WriteString("- 'Console log stats' - View log statistics\n")
		response.WriteString("- 'Check for errors' - Analyze recent errors\n")
		response.WriteString("- 'Recent activity' - See what's been happening\n")
		response.WriteString("- I automatically alert on new errors!\n\n")
		response.WriteString("⚡ System Optimization:\n")
		response.WriteString("- 'Optimization suggestions' - Get recommendations\n")
		response.WriteString("- I continuously monitor library health\n")
		response.WriteString("- Alert on performance opportunities\n")
		response.WriteString("- Provide actionable improvement tips\n\n")
		response.WriteString("🧠 Intelligent Learning:\n")
		response.WriteString("- 'Usage patterns' - See detected patterns\n")
		response.WriteString("- 'What should I do next?' - Get smart suggestions\n")
		response.WriteString("- 'Maintenance schedule' - View recommended tasks\n")
		response.WriteString("- I learn from your actions over time\n\n")
		response.WriteString("🔍 Advanced Analysis:\n")
		response.WriteString("- 'Find duplicates' - Detect duplicate videos\n")
		response.WriteString("- 'Library health' - Get comprehensive health score\n")
		response.WriteString("- 'Recommendations' - Personalized content suggestions\n")
		response.WriteString("- Smart duplicate detection by title and size\n\n")
		response.WriteString("📈 Predictive Analytics:\n")
		response.WriteString("- 'Predict growth' - Library growth predictions\n")
		response.WriteString("- 'Performer quality' - Quality scoring analysis\n")
		response.WriteString("- 'Show insights' - AI-powered insights\n")
		response.WriteString("- Analyze trends and predict future states\n\n")
		response.WriteString("🤖 AI Task Capabilities:\n")
		response.WriteString("- 'Auto-link performers' - Automatic performer detection\n")
		response.WriteString("- 'Smart tagging' - Intelligent tag suggestions\n")
		response.WriteString("- 'Detect duplicates' - Find duplicate videos\n")
		response.WriteString("- 'Scene detection' - Analyze video scenes\n")
		response.WriteString("- 'Quality analysis' - Check video quality\n")
		response.WriteString("- 'Content classification' - Categorize videos\n")
		response.WriteString("- 'Auto-naming' - Generate better filenames\n")
		response.WriteString("- Ask me about any AI task for details!\n\n")
		response.WriteString("💡 Try asking:\n")
		response.WriteString("- 'What's my library health?'\n")
		response.WriteString("- 'Find duplicates'\n")
		response.WriteString("- 'Predict library growth'\n")
		response.WriteString("- 'Performer quality'\n")
		response.WriteString("- 'Show insights'\n")
		response.WriteString("- 'Find performer [name]'\n")
		response.WriteString("- 'Recent videos'\n")
		response.WriteString("- 'Storage usage'\n")
		response.WriteString("- 'Check for errors'\n")
		response.WriteString("- 'What should I do next?'\n")
		response.WriteString("- 'Give me recommendations'\n")
		return response.String(), nil
	}

	// Status check
	if strings.Contains(messageLower, "status") || strings.Contains(messageLower, "how are you") {
		status := s.GetStatus()
		uptimeSeconds := status["uptime_seconds"].(float64)
		uptime := time.Duration(uptimeSeconds * float64(time.Second))
		return fmt.Sprintf("I'm running smoothly! Uptime: %s, Watching %d libraries, Processed %d events.",
			uptime.Round(time.Second), status["active_watchers"], status["events_processed"]), nil
	}

	// Task execution suggestions
	if strings.Contains(messageLower, "scan") || strings.Contains(messageLower, "task") {
		return "I can help you run tasks! However, tasks are best executed through the Task Center page in the app. You can find quick action buttons there for:\n- Auto-Link Performers\n- Smart Tagging\n- Duplicate Detection\n- Quality Analysis\n\nOr you can ask me specific questions about your library!", nil
	}

	// Help / capabilities
	if strings.Contains(messageLower, "help") || strings.Contains(messageLower, "what can you do") {
		return "I'm your AI Companion! I can help you with:\n\n" +
			"📊 Library Information:\n" +
			"- 'How many videos/performers do I have?'\n" +
			"- 'Library statistics'\n" +
			"- 'Top performers'\n\n" +
			"🤖 System Status:\n" +
			"- 'Status' or 'How are you?'\n\n" +
			"🧠 Memory:\n" +
			"- I remember our conversations\n" +
			"- I learn your preferences over time\n\n" +
			"For complex questions or natural conversations, I can connect to LM Studio for advanced AI capabilities!", nil
	}

	// Default response if no rule matches
	return "I understand your question. For advanced natural language processing, I can connect to LM Studio if needed. For now, I can help with:\n" +
		"- Library statistics ('how many videos?')\n" +
		"- Performer information ('top performers')\n" +
		"- System status ('status')\n" +
		"- General help ('help')", nil
}

// ================== Memory Management ==================

// SaveMemory stores a memory in the database
func (s *AICompanionService) SaveMemory(memory *models.Memory) error {
	now := time.Now()
	memory.CreatedAt = now
	memory.UpdatedAt = now

	query := `INSERT INTO memories (key, value, category, importance, created_at, updated_at)
	          VALUES (?, ?, ?, ?, ?, ?)
	          ON CONFLICT(key) DO UPDATE SET value=?, updated_at=?`

	_, err := s.db.Exec(query,
		memory.Key, memory.Value, memory.Category, memory.Importance, memory.CreatedAt, memory.UpdatedAt,
		memory.Value, now)

	return err
}

// GetMemories retrieves memories with optional filters
func (s *AICompanionService) GetMemories(category string, limit int) ([]models.Memory, error) {
	query := `SELECT id, key, value, category, importance, created_at, updated_at FROM memories`
	args := []interface{}{}

	if category != "" && category != "all" {
		query += ` WHERE category = ?`
		args = append(args, category)
	}

	query += ` ORDER BY importance DESC, updated_at DESC`

	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	memories := []models.Memory{}
	for rows.Next() {
		var m models.Memory
		if err := rows.Scan(&m.ID, &m.Key, &m.Value, &m.Category, &m.Importance, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		memories = append(memories, m)
	}

	return memories, nil
}

// SearchMemories searches memories by query string
func (s *AICompanionService) SearchMemories(query string) ([]models.Memory, error) {
	searchQuery := `SELECT id, key, value, category, importance, created_at, updated_at FROM memories
	                WHERE key LIKE ? OR value LIKE ?
	                ORDER BY importance DESC, updated_at DESC
	                LIMIT 20`

	searchPattern := "%" + query + "%"
	rows, err := s.db.Query(searchQuery, searchPattern, searchPattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	memories := []models.Memory{}
	for rows.Next() {
		var m models.Memory
		if err := rows.Scan(&m.ID, &m.Key, &m.Value, &m.Category, &m.Importance, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		memories = append(memories, m)
	}

	return memories, nil
}

// UpdateMemory updates an existing memory
func (s *AICompanionService) UpdateMemory(id int64, value string) error {
	query := `UPDATE memories SET value = ?, updated_at = ? WHERE id = ?`
	_, err := s.db.Exec(query, value, time.Now(), id)
	return err
}

// DeleteMemory deletes a memory by ID
func (s *AICompanionService) DeleteMemory(id int64) error {
	query := `DELETE FROM memories WHERE id = ?`
	_, err := s.db.Exec(query, id)
	return err
}

// GetRecentEvents returns recent events (for debugging/monitoring)
func (s *AICompanionService) GetRecentEvents(limit int) []CompanionEvent {
	// This would ideally be stored in a circular buffer or database
	// For now, return empty slice (events are currently only streamed)
	return []CompanionEvent{}
}

// Helper function to marshal event data
func (e *CompanionEvent) ToJSON() string {
	data, _ := json.Marshal(e)
	return string(data)
}

// formatDuration formats a duration into a human-readable string
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d seconds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%d minutes", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%d hours", int(d.Hours()))
	}
	return fmt.Sprintf("%d days", int(d.Hours()/24))
}
