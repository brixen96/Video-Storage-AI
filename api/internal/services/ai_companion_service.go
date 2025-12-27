package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
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
func (s *AICompanionService) SubscribeToEvents() <-chan CompanionEvent {
	s.mu.Lock()
	defer s.mu.Unlock()

	eventChan := make(chan CompanionEvent, 10)
	s.subscribers = append(s.subscribers, eventChan)
	return eventChan
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
				select {
				case subscriber <- event:
				default:
					// Skip if subscriber channel is full
				}
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

	// What can you do
	if strings.Contains(messageLower, "what can you do") || strings.Contains(messageLower, "help") && strings.Contains(messageLower, "companion") {
		var response strings.Builder
		response.WriteString("🤖 I'm your AI Companion! Here's what I can help with:\n\n")
		response.WriteString("📊 Information:\n")
		response.WriteString("- Ask about library stats, video counts, performer counts\n")
		response.WriteString("- Check thumbnail coverage and performance\n")
		response.WriteString("- Get top performers list\n\n")
		response.WriteString("⚡ Monitoring:\n")
		response.WriteString("- I continuously monitor your library health\n")
		response.WriteString("- Notify you about performance opportunities\n")
		response.WriteString("- Alert on missing metadata or thumbnails\n")
		response.WriteString("- Daily library reports\n\n")
		response.WriteString("💡 Try asking:\n")
		response.WriteString("- 'What's my library overview?'\n")
		response.WriteString("- 'Thumbnail status?'\n")
		response.WriteString("- 'Top performers?'\n")
		response.WriteString("- 'How can I improve performance?'\n")
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
