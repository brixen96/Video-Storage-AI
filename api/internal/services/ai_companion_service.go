package services

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
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

	// Fire daily briefing after a short delay (gives app time to fully start)
	go func() {
		time.Sleep(6 * time.Second)
		s.generateDailyBriefing()
	}()

	// Run library insights check shortly after startup
	go func() {
		time.Sleep(15 * time.Second)
		s.runLibraryInsights()
	}()

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

// GetFullContext gathers comprehensive app state for the LLM system prompt
func (s *AICompanionService) GetFullContext() string {
	var ctx strings.Builder

	// Library stats
	var videoCount, performerCount, tagCount, libraryCount int
	s.db.QueryRow("SELECT COUNT(*) FROM videos").Scan(&videoCount)
	s.db.QueryRow("SELECT COUNT(*) FROM performers").Scan(&performerCount)
	s.db.QueryRow("SELECT COUNT(*) FROM tags").Scan(&tagCount)
	s.db.QueryRow("SELECT COUNT(*) FROM libraries").Scan(&libraryCount)

	ctx.WriteString("## Current App State\n\n")
	var favCount, ratedCount int
	var avgRating float64
	s.db.QueryRow("SELECT COUNT(*) FROM videos WHERE is_favorite = 1").Scan(&favCount)
	s.db.QueryRow("SELECT COUNT(*), COALESCE(AVG(CAST(rating AS FLOAT)), 0) FROM videos WHERE rating > 0").Scan(&ratedCount, &avgRating)

	ctx.WriteString(fmt.Sprintf("### Library\n- Videos: %d\n- Performers: %d\n- Tags: %d\n- Libraries: %d\n- Favorites: %d | Rated: %d (avg %.1f/5)\n\n", videoCount, performerCount, tagCount, libraryCount, favCount, ratedCount, avgRating))

	// List all libraries with names and video counts
	libRows, libErr := s.db.Query("SELECT l.name, l.path, COUNT(v.id) FROM libraries l LEFT JOIN videos v ON l.id = v.library_id GROUP BY l.id ORDER BY COUNT(v.id) DESC")
	if libErr == nil {
		defer libRows.Close()
		ctx.WriteString("### Libraries\n")
		for libRows.Next() {
			var name, path string
			var cnt int
			if libRows.Scan(&name, &path, &cnt) == nil {
				ctx.WriteString(fmt.Sprintf("  - **%s** (%d videos) → %s\n", name, cnt, path))
			}
		}
		ctx.WriteString("\n")
	}

	// Video linking (Edited↔Backup pairing) stats
	var confirmedLinks, suggestedLinks, rejectedLinks int
	s.db.QueryRow("SELECT COUNT(*) FROM video_links WHERE status = 'confirmed'").Scan(&confirmedLinks)
	s.db.QueryRow("SELECT COUNT(*) FROM video_links WHERE status = 'suggested'").Scan(&suggestedLinks)
	s.db.QueryRow("SELECT COUNT(*) FROM video_links WHERE status = 'rejected'").Scan(&rejectedLinks)
	if confirmedLinks+suggestedLinks+rejectedLinks > 0 {
		ctx.WriteString(fmt.Sprintf("### Video Pairing (Edited↔Backup)\n- Confirmed links: %d | Suggested (pending review): %d | Rejected: %d\n\n", confirmedLinks, suggestedLinks, rejectedLinks))
	}

	// Scraper state
	var scrapedVideoCount, threadCount, postCount int
	s.db.QueryRow("SELECT COUNT(*) FROM scraped_videos").Scan(&scrapedVideoCount)
	s.db.QueryRow("SELECT COUNT(*) FROM scraped_threads").Scan(&threadCount)
	s.db.QueryRow("SELECT COUNT(*) FROM scraped_posts").Scan(&postCount)

	// Per-source scraped counts
	var phCount, ltCount, sxCount, r34Count int
	s.db.QueryRow("SELECT COUNT(*) FROM scraped_videos WHERE source = 'pornhub'").Scan(&phCount)
	s.db.QueryRow("SELECT COUNT(*) FROM scraped_videos WHERE source = 'luxuretv'").Scan(&ltCount)
	s.db.QueryRow("SELECT COUNT(*) FROM scraped_videos WHERE source = 'sexuria'").Scan(&sxCount)
	s.db.QueryRow("SELECT COUNT(*) FROM scraped_videos WHERE source = 'rule34video'").Scan(&r34Count)

	ctx.WriteString(fmt.Sprintf("### Scrapers\n- Scraped Videos Total: %d (PornHub: %d, LuxureTV: %d, Sexuria: %d, Rule34Video: %d)\n", scrapedVideoCount, phCount, ltCount, sxCount, r34Count))
	ctx.WriteString(fmt.Sprintf("- Forum Threads: %d, Posts: %d\n", threadCount, postCount))

	// Running tasks from activity_logs
	rows, err := s.db.Query(`
		SELECT task_type, status, message, created_at
		FROM activity_logs
		WHERE status = 'running'
		ORDER BY created_at DESC
		LIMIT 10
	`)
	if err == nil {
		defer rows.Close()
		runningTasks := []string{}
		for rows.Next() {
			var taskType, status, message string
			var createdAt time.Time
			if rows.Scan(&taskType, &status, &message, &createdAt) == nil {
				runningTasks = append(runningTasks, fmt.Sprintf("  - [%s] %s (started %s ago)", taskType, message, formatDuration(time.Since(createdAt))))
			}
		}
		if len(runningTasks) > 0 {
			ctx.WriteString("\n### Running Tasks\n")
			for _, t := range runningTasks {
				ctx.WriteString(t + "\n")
			}
		} else {
			ctx.WriteString("\n### Running Tasks\nNo tasks currently running.\n")
		}
	}

	// Recent completed activity
	recentRows, err := s.db.Query(`
		SELECT task_type, status, message, created_at
		FROM activity_logs
		WHERE status != 'running'
		ORDER BY created_at DESC
		LIMIT 5
	`)
	if err == nil {
		defer recentRows.Close()
		ctx.WriteString("\n### Recent Activity\n")
		for recentRows.Next() {
			var taskType, status, message string
			var createdAt time.Time
			if recentRows.Scan(&taskType, &status, &message, &createdAt) == nil {
				ctx.WriteString(fmt.Sprintf("  - [%s] %s - %s (%s ago)\n", taskType, status, message, formatDuration(time.Since(createdAt))))
			}
		}
	}

	// JDownloader state
	var jdConnType, jdConnStatus string
	err = s.db.QueryRow("SELECT connection_type, connection_status FROM jdownloader_config LIMIT 1").Scan(&jdConnType, &jdConnStatus)
	if err == nil {
		ctx.WriteString(fmt.Sprintf("\n### JDownloader\n- Connection: %s (%s)\n", jdConnType, jdConnStatus))
		var jdHistoryCount int
		s.db.QueryRow("SELECT COUNT(*) FROM jdownloader_history").Scan(&jdHistoryCount)
		ctx.WriteString(fmt.Sprintf("- Links Sent: %d\n", jdHistoryCount))
	}

	// Recent errors (last hour)
	var errorCount int
	s.db.QueryRow("SELECT COUNT(*) FROM console_logs WHERE level = 'error' AND created_at > datetime('now', '-1 hour')").Scan(&errorCount)
	ctx.WriteString(fmt.Sprintf("\n### System Health\n- Errors (last hour): %d\n", errorCount))

	// Companion uptime
	s.mu.RLock()
	uptime := time.Duration(0)
	if s.isRunning {
		uptime = time.Since(s.startTime)
	}
	s.mu.RUnlock()
	ctx.WriteString(fmt.Sprintf("- AI Companion Uptime: %s\n", formatDuration(uptime)))

	// Memories (top important ones)
	memRows, err := s.db.Query("SELECT key, value FROM memories ORDER BY importance DESC, updated_at DESC LIMIT 5")
	if err == nil {
		defer memRows.Close()
		memories := []string{}
		for memRows.Next() {
			var key, value string
			if memRows.Scan(&key, &value) == nil {
				memories = append(memories, fmt.Sprintf("  - %s: %s", key, value))
			}
		}
		if len(memories) > 0 {
			ctx.WriteString("\n### Remembered Context\n")
			for _, m := range memories {
				ctx.WriteString(m + "\n")
			}
		}
	}

	return ctx.String()
}

// GetAIConfig retrieves the AI/LM Studio configuration from the database
func (s *AICompanionService) GetAIConfig() models.AIConfig {
	config := models.AIConfig{
		ID:          1,
		LMStudioURL: "http://localhost:1234",
		Temperature: 0.7,
		MaxTokens:   2000,
		Enabled:     true,
	}
	s.db.QueryRow(`SELECT id, lm_studio_url, model_name, temperature, max_tokens, enabled FROM ai_config WHERE id = 1`).Scan(
		&config.ID, &config.LMStudioURL, &config.ModelName, &config.Temperature, &config.MaxTokens, &config.Enabled,
	)
	return config
}

// TestLMStudioConnection tests connectivity to LM Studio
func (s *AICompanionService) TestLMStudioConnection() map[string]interface{} {
	config := s.GetAIConfig()
	result := map[string]interface{}{
		"connected": false,
		"url":       config.LMStudioURL,
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(config.LMStudioURL + "/v1/models")
	if err != nil {
		result["error"] = fmt.Sprintf("Connection failed: %v", err)
		return result
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		result["error"] = "Failed to read response"
		return result
	}

	var modelsResp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &modelsResp); err != nil {
		result["error"] = "Invalid response from LM Studio"
		return result
	}

	result["connected"] = true
	models := []string{}
	for _, m := range modelsResp.Data {
		models = append(models, m.ID)
	}
	result["models"] = models
	if len(models) > 0 {
		result["message"] = fmt.Sprintf("Connected! %d model(s) available", len(models))
	} else {
		result["message"] = "Connected but no models loaded"
	}
	return result
}

// ================== LLM Chat Integration ==================

// lmToolDefinition represents a tool definition for the LM Studio API
type lmToolDefinition struct {
	Type     string          `json:"type"`
	Function lmToolFunction  `json:"function"`
}

type lmToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// getLMToolDefinitions returns all tool definitions for the LLM
func getLMToolDefinitions() []lmToolDefinition {
	return []lmToolDefinition{
		{Type: "function", Function: lmToolFunction{
			Name: "get_library_stats", Description: "Get overall statistics about the video library including total videos, performers, tags, studios, and database size.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}, "required": []string{}},
		}},
		{Type: "function", Function: lmToolFunction{
			Name: "search_videos",
			Description: "Rich video search supporting natural-language criteria: filter by title, performer name, tag, studio, library, duration range, minimum rating, and favorite status. Also supports sort order. Use this to answer queries like 'Nicole Aniston videos over 30 minutes', 'my favorites rated 4+', 'newest videos in Backup', or 'random video suggestion'.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"query":        map[string]interface{}{"type": "string", "description": "Title keyword search (optional)"},
				"performer":    map[string]interface{}{"type": "string", "description": "Filter by performer full name (optional)"},
				"tag":          map[string]interface{}{"type": "string", "description": "Filter by tag name (optional)"},
				"studio":       map[string]interface{}{"type": "string", "description": "Filter by studio name (optional)"},
				"library":      map[string]interface{}{"type": "string", "description": "Filter by library name, e.g. 'Backup' (optional)"},
				"min_duration": map[string]interface{}{"type": "number", "description": "Minimum duration in seconds, e.g. 1800 for 30 min (optional)"},
				"max_duration": map[string]interface{}{"type": "number", "description": "Maximum duration in seconds (optional)"},
				"min_rating":   map[string]interface{}{"type": "number", "description": "Minimum rating 1-5 (optional)"},
				"is_favorite":  map[string]interface{}{"type": "boolean", "description": "If true, return only favorites (optional)"},
				"sort_by":      map[string]interface{}{"type": "string", "description": "Sort order: 'recent' (default), 'rating', 'duration', 'title', 'random'"},
				"limit":        map[string]interface{}{"type": "number", "description": "Max results (default: 20, max: 50)"},
			}},
		}},
		{Type: "function", Function: lmToolFunction{
			Name: "search_performers", Description: "Search for performers by name. Returns performer details.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"query": map[string]interface{}{"type": "string", "description": "Search query to filter performers by name"},
				"limit": map[string]interface{}{"type": "number", "description": "Max results (default: 50)"},
			}, "required": []string{}},
		}},
		{Type: "function", Function: lmToolFunction{
			Name: "get_top_performers", Description: "Get performers with the most videos.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"limit": map[string]interface{}{"type": "number", "description": "Number of top performers (default: 10)"},
			}, "required": []string{}},
		}},
		{Type: "function", Function: lmToolFunction{
			Name: "get_all_tags", Description: "Get all tags with their names and video counts.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}, "required": []string{}},
		}},
		{Type: "function", Function: lmToolFunction{
			Name: "get_all_libraries", Description: "Get all video libraries with names, paths, and video counts.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}, "required": []string{}},
		}},
		{Type: "function", Function: lmToolFunction{
			Name: "save_memory", Description: "Store important information in long-term memory that persists forever.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"key":        map[string]interface{}{"type": "string", "description": "Unique identifier for this memory"},
				"value":      map[string]interface{}{"type": "string", "description": "The information to remember"},
				"category":   map[string]interface{}{"type": "string", "enum": []string{"preference", "fact", "insight", "task", "note"}, "description": "Type of memory"},
				"importance": map[string]interface{}{"type": "number", "description": "Importance level 1-10"},
			}, "required": []string{"key", "value", "category"}},
		}},
		{Type: "function", Function: lmToolFunction{
			Name: "recall_memories", Description: "Search and retrieve stored memories.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"query": map[string]interface{}{"type": "string", "description": "Search query to find relevant memories"},
			}, "required": []string{}},
		}},
		// New app-awareness tools
		{Type: "function", Function: lmToolFunction{
			Name: "get_running_tasks", Description: "Get currently running tasks/activities in the app (scraping, scanning, etc).",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}, "required": []string{}},
		}},
		{Type: "function", Function: lmToolFunction{
			Name: "get_scraper_status", Description: "Get status of all scrapers including video counts per source, recent scrape activity, and auto-update configs.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}, "required": []string{}},
		}},
		{Type: "function", Function: lmToolFunction{
			Name: "get_jdownloader_status", Description: "Get JDownloader connection status, queue info, and recent download history.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}, "required": []string{}},
		}},
		{Type: "function", Function: lmToolFunction{
			Name: "get_recent_errors", Description: "Get recent errors from the console log (last hour).",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"limit": map[string]interface{}{"type": "number", "description": "Max errors to return (default: 10)"},
			}, "required": []string{}},
		}},
		{Type: "function", Function: lmToolFunction{
			Name: "search_scraped_content", Description: "Search scraped videos and forum posts by keyword.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"query":  map[string]interface{}{"type": "string", "description": "Search keyword"},
				"source": map[string]interface{}{"type": "string", "description": "Filter by source: pornhub, luxuretv, sexuria, rule34video, simpcity"},
				"limit":  map[string]interface{}{"type": "number", "description": "Max results (default: 10)"},
			}, "required": []string{"query"}},
		}},
		{Type: "function", Function: lmToolFunction{
			Name: "get_recent_activity", Description: "Get recent activity log entries showing what happened in the app.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"limit": map[string]interface{}{"type": "number", "description": "Max entries (default: 10)"},
			}, "required": []string{}},
		}},
		{Type: "function", Function: lmToolFunction{
			Name:        "suggest_scraper_config",
			Description: "Fetch and analyze a website to suggest scraper configuration including site type and pagination pattern. Use this when the user wants to add a new scraper tab for a website and provides one or two page URLs.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"page1_url": map[string]interface{}{"type": "string", "description": "URL of the first/listing page of the site"},
					"page2_url": map[string]interface{}{"type": "string", "description": "URL of the second page (optional but helps detect pagination pattern)"},
				},
				"required": []string{"page1_url"},
			},
		}},
		{Type: "function", Function: lmToolFunction{
			Name: "get_favorites", Description: "Get the user's favorited videos with titles, ratings, and performers.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"limit": map[string]interface{}{"type": "number", "description": "Max results (default: 20)"},
			}, "required": []string{}},
		}},
		{Type: "function", Function: lmToolFunction{
			Name: "get_top_rated", Description: "Get the highest-rated videos sorted by rating descending, with performer names.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"limit":      map[string]interface{}{"type": "number", "description": "Max results (default: 20)"},
				"min_rating": map[string]interface{}{"type": "number", "description": "Minimum rating to include (1-5, default: 1)"},
			}, "required": []string{}},
		}},
		{Type: "function", Function: lmToolFunction{
			Name: "get_library_videos", Description: "List recent videos in a specific library (e.g. 'Downloads - Local', 'Backup'). Returns title, path, performers, and tags.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"library_name": map[string]interface{}{"type": "string", "description": "Library name or partial name to search for"},
				"limit":        map[string]interface{}{"type": "number", "description": "Max results (default: 20)"},
			}, "required": []string{"library_name"}},
		}},
		{Type: "function", Function: lmToolFunction{
			Name: "scan_library_structure", Description: "List the folder/directory structure of a library up to 3 levels deep. Use this to understand how a library (e.g. 'Backup') is organized before suggesting placement.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"library_name": map[string]interface{}{"type": "string", "description": "Library name or partial name"},
				"max_depth":    map[string]interface{}{"type": "number", "description": "How many levels deep to scan (default: 2, max: 4)"},
			}, "required": []string{"library_name"}},
		}},
		{Type: "function", Function: lmToolFunction{
			Name: "suggest_video_placement", Description: "Analyze a video's metadata (performers, tags, title) and suggest the best matching subfolders in the Backup library. Call scan_library_structure first if you haven't already.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"video_id":       map[string]interface{}{"type": "number", "description": "ID of the video to place"},
				"target_library": map[string]interface{}{"type": "string", "description": "Target library name (default: 'Backup')"},
			}, "required": []string{"video_id"}},
		}},
		{Type: "function", Function: lmToolFunction{
			Name: "move_video_to_library", Description: "Move a video file to a subfolder in another library and update the database. Use suggest_video_placement first to get the right subfolder path.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"video_id":        map[string]interface{}{"type": "number", "description": "ID of the video to move"},
				"target_library":  map[string]interface{}{"type": "string", "description": "Target library name (e.g. 'Backup')"},
				"target_subfolder": map[string]interface{}{"type": "string", "description": "Relative subfolder path inside target library (e.g. 'Performers/Alice'). Empty = library root."},
			}, "required": []string{"video_id", "target_library"}},
		}},
		{Type: "function", Function: lmToolFunction{
			Name: "analyze_library_structure", Description: "Deep-analyze a library's organizational structure by examining what performers, tags, and studios already exist in each folder. Returns the detected pattern (e.g. Performers/<Name>/, Categories/<Tag>/), dominant entities per folder, and improvement suggestions. Use this to understand how Backup is organized before suggesting placements.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"library_name": map[string]interface{}{"type": "string", "description": "Library name or partial name to analyze"},
				"max_depth":    map[string]interface{}{"type": "number", "description": "Max folder depth to analyze (default: 3)"},
			}, "required": []string{"library_name"}},
		}},
		{Type: "function", Function: lmToolFunction{
			Name:        "get_daily_briefing",
			Description: "Generate a fresh daily briefing with library stats, new videos since yesterday, unorganized downloads, untagged count, and recent errors. Use this when the user asks for a status update, summary, or 'what happened today'.",
			Parameters:  map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		}},
		{Type: "function", Function: lmToolFunction{
			Name:        "suggest_videos_to_watch",
			Description: "Recommend videos the user hasn't favorited or rated highly yet, based on their taste profile built from favorites and 4★+ rated videos. Matches by shared performers, tags, and studios. Use when the user asks 'what should I watch?', 'suggest something', or 'recommend a video'.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"count":    map[string]interface{}{"type": "number", "description": "How many suggestions to return (default: 5)"},
				"library":  map[string]interface{}{"type": "string", "description": "Restrict suggestions to a specific library (optional)"},
			}},
		}},
		{Type: "function", Function: lmToolFunction{
			Name:        "get_performer_stats",
			Description: "Get a comprehensive profile for a performer: total videos, total watch time, avg rating, rating distribution, favorite count, library breakdown (which libraries their videos are in), top tags, top studios, recent additions, and date range. Use when the user asks 'tell me about [performer]', 'how many [name] videos do I have', or wants a performer overview.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"performer": map[string]interface{}{"type": "string", "description": "Performer name to look up (partial match supported)"},
			}, "required": []string{"performer"}},
		}},
		{Type: "function", Function: lmToolFunction{
			Name:        "find_duplicate_videos",
			Description: "Scan the video library for likely duplicate files — same filename in multiple places, same title across libraries, or same performers with similar duration. Returns suspect pairs grouped by confidence level. Use this when the user asks to find duplicates or clean up the library.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"library_name": map[string]interface{}{"type": "string", "description": "Restrict search to a specific library (optional — omit to search all libraries)"},
				"limit":        map[string]interface{}{"type": "number", "description": "Max number of duplicate groups to return (default: 30)"},
			}},
		}},
		{Type: "function", Function: lmToolFunction{
			Name:        "find_orphaned_videos",
			Description: "Check which videos in the database no longer have their file on disk — the file was moved, renamed, or deleted outside the app. Returns a list of orphaned DB entries. Always run this before delete_video_records. Use when the user asks about broken entries, missing files, or wants to clean up the library.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"library_name": map[string]interface{}{"type": "string", "description": "Restrict check to a specific library (optional)"},
				"limit":        map[string]interface{}{"type": "number", "description": "Max videos to scan (default: 2000)"},
			}},
		}},
		{Type: "function", Function: lmToolFunction{
			Name:        "delete_video_records",
			Description: "Delete video entries from the database by ID. Only deletes the DB record — never touches files on disk. Use this to clean up orphaned entries found by find_orphaned_videos. ALWAYS confirm with the user before calling this tool.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"video_ids": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "number"},
					"description": "List of video IDs to remove from the database",
				},
			}, "required": []string{"video_ids"}},
		}},
		{Type: "function", Function: lmToolFunction{
			Name:        "find_untagged_videos",
			Description: "Find videos that have no tags, optionally filtered by library or performer. Returns video IDs, titles, and performer names so you can call suggest_tags_for_video then apply_tags_to_video on each. Use when the user wants to bulk-tag their library or asks 'which videos have no tags?'.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"library_name": map[string]interface{}{"type": "string", "description": "Restrict to a specific library (optional)"},
				"performer":    map[string]interface{}{"type": "string", "description": "Restrict to videos featuring a specific performer (optional)"},
				"limit":        map[string]interface{}{"type": "number", "description": "Max results (default: 20, max: 50)"},
			}},
		}},
		{Type: "function", Function: lmToolFunction{
			Name:        "bulk_apply_tags",
			Description: "Automatically suggest and apply performer-based tags to a batch of videos in one operation. For each video, finds the most common tags from other videos featuring the same performers and applies them. Use after find_untagged_videos for efficient bulk tagging. ALWAYS tell the user how many videos will be tagged and confirm before running.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"video_ids": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "number"},
					"description": "List of video IDs to auto-tag",
				},
				"max_tags_per_video": map[string]interface{}{"type": "number", "description": "Max tags to apply per video (default: 5)"},
			}, "required": []string{"video_ids"}},
		}},
		{Type: "function", Function: lmToolFunction{
			Name:        "get_library_health",
			Description: "Generate a comprehensive health report of the entire library: total counts, untagged %, unrated %, videos with no performers, recent additions, and a health score (0-100). Use when the user asks 'how's my library?', 'library health', or 'give me an overview'.",
			Parameters:  map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		}},
		{Type: "function", Function: lmToolFunction{
			Name:        "suggest_tags_for_video",
			Description: "Suggest tags for a video based on what tags are commonly applied to other videos featuring the same performers. Excludes tags already applied. Use when the user asks 'what tags should I add?', 'tag this video', or wants tag suggestions for a specific video.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"video_id": map[string]interface{}{"type": "number", "description": "ID of the video to suggest tags for"},
				"limit":    map[string]interface{}{"type": "number", "description": "Max suggestions to return (default: 10)"},
			}, "required": []string{"video_id"}},
		}},
		{Type: "function", Function: lmToolFunction{
			Name:        "apply_tags_to_video",
			Description: "Apply one or more tags to a video by name. Creates tags that don't exist yet. Use after suggest_tags_for_video or when the user explicitly asks to tag a video with specific tags.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"video_id": map[string]interface{}{"type": "number", "description": "ID of the video to tag"},
				"tags": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "List of tag names to apply",
				},
			}, "required": []string{"video_id", "tags"}},
		}},
		{Type: "function", Function: lmToolFunction{
			Name:        "set_video_rating",
			Description: "Set the star rating of a video (0-5). Use when the user asks to rate a video or says something like 'give this a 4 star rating'. 0 clears the rating.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"video_id": map[string]interface{}{"type": "number", "description": "ID of the video to rate"},
				"rating":   map[string]interface{}{"type": "number", "description": "Rating from 0 (clear) to 5 (max)"},
			}, "required": []string{"video_id", "rating"}},
		}},
		{Type: "function", Function: lmToolFunction{
			Name:        "toggle_video_favorite",
			Description: "Toggle the favorite flag on a video — marks it as a favorite if not already, or removes the favorite if it is. Use when the user says 'favorite this video', 'add to favorites', or 'remove from favorites'.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"video_id": map[string]interface{}{"type": "number", "description": "ID of the video to toggle favorite on"},
			}, "required": []string{"video_id"}},
		}},
		{Type: "function", Function: lmToolFunction{
			Name:        "get_video_link_stats",
			Description: "Get statistics about Edited↔Backup video pairing: how many confirmed links exist, how many are pending review, which library has the most unlinked videos, and overall link coverage percentage. Use when the user asks about paired videos, library sync status, or 'how many Edited videos have a Backup counterpart'.",
			Parameters:  map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		}},
		{Type: "function", Function: lmToolFunction{
			Name:        "get_unlinked_videos",
			Description: "List videos in a specific library that do not have a confirmed link to a video in the paired library. Use this to find which Edited videos are missing a Backup counterpart (or vice versa). Pairs with get_video_link_stats to give a full picture.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"library_name": map[string]interface{}{"type": "string", "description": "Library name to check for unlinked videos (e.g. 'Edited' or 'Backup')"},
				"limit":        map[string]interface{}{"type": "number", "description": "Max results to return (default: 20)"},
			}, "required": []string{"library_name"}},
		}},
	}
}

// ExecuteTool executes a tool call from the LLM and returns the result
func (s *AICompanionService) ExecuteTool(name string, args map[string]interface{}) (interface{}, error) {
	switch name {
	case "get_library_stats":
		var videoCount, performerCount, tagCount, libraryCount, studioCount int
		s.db.QueryRow("SELECT COUNT(*) FROM videos").Scan(&videoCount)
		s.db.QueryRow("SELECT COUNT(*) FROM performers").Scan(&performerCount)
		s.db.QueryRow("SELECT COUNT(*) FROM tags").Scan(&tagCount)
		s.db.QueryRow("SELECT COUNT(*) FROM libraries").Scan(&libraryCount)
		s.db.QueryRow("SELECT COUNT(*) FROM studios").Scan(&studioCount)
		return map[string]interface{}{
			"total_videos": videoCount, "total_performers": performerCount,
			"total_tags": tagCount, "total_libraries": libraryCount, "total_studios": studioCount,
		}, nil

	case "search_videos":
		titleQ, _ := args["query"].(string)
		performer, _ := args["performer"].(string)
		tag, _ := args["tag"].(string)
		studio, _ := args["studio"].(string)
		library, _ := args["library"].(string)
		minDur := floatArg(args, "min_duration", 0)
		maxDur := floatArg(args, "max_duration", 0)
		minRating := intArg(args, "min_rating", 0)
		isFavOnly, _ := args["is_favorite"].(bool)
		sortBy, _ := args["sort_by"].(string)
		limit := intArg(args, "limit", 20)
		if limit > 50 {
			limit = 50
		}

		var conds []string
		var qArgs []interface{}

		if titleQ != "" {
			conds = append(conds, "v.title LIKE ?")
			qArgs = append(qArgs, "%"+titleQ+"%")
		}
		if performer != "" {
			conds = append(conds, `EXISTS (SELECT 1 FROM video_performers vp JOIN performers p ON p.id = vp.performer_id WHERE vp.video_id = v.id AND LOWER(p.name) LIKE LOWER(?))`)
			qArgs = append(qArgs, "%"+performer+"%")
		}
		if tag != "" {
			conds = append(conds, `EXISTS (SELECT 1 FROM video_tags vt JOIN tags t ON t.id = vt.tag_id WHERE vt.video_id = v.id AND LOWER(t.name) LIKE LOWER(?))`)
			qArgs = append(qArgs, "%"+tag+"%")
		}
		if studio != "" {
			conds = append(conds, `EXISTS (SELECT 1 FROM video_studios vs JOIN studios s ON s.id = vs.studio_id WHERE vs.video_id = v.id AND LOWER(s.name) LIKE LOWER(?))`)
			qArgs = append(qArgs, "%"+studio+"%")
		}
		if library != "" {
			conds = append(conds, "LOWER(l.name) LIKE LOWER(?)")
			qArgs = append(qArgs, "%"+library+"%")
		}
		if minDur > 0 {
			conds = append(conds, "v.duration >= ?")
			qArgs = append(qArgs, minDur)
		}
		if maxDur > 0 {
			conds = append(conds, "v.duration <= ?")
			qArgs = append(qArgs, maxDur)
		}
		if minRating > 0 {
			conds = append(conds, "COALESCE(v.rating, 0) >= ?")
			qArgs = append(qArgs, minRating)
		}
		if isFavOnly {
			conds = append(conds, "v.is_favorite = 1")
		}

		where := ""
		if len(conds) > 0 {
			where = "WHERE " + strings.Join(conds, " AND ")
		}

		orderClause := "ORDER BY v.created_at DESC"
		switch strings.ToLower(sortBy) {
		case "rating":
			orderClause = "ORDER BY COALESCE(v.rating, 0) DESC"
		case "duration":
			orderClause = "ORDER BY COALESCE(v.duration, 0) DESC"
		case "title":
			orderClause = "ORDER BY v.title ASC"
		case "random":
			orderClause = "ORDER BY RANDOM()"
		}

		searchQ := fmt.Sprintf(`
			SELECT v.id, v.title,
			       COALESCE(v.duration, 0),
			       COALESCE(v.rating, 0),
			       COALESCE(v.is_favorite, 0),
			       COALESCE(v.thumbnail_path, ''),
			       COALESCE(l.name, ''),
			       COALESCE((SELECT GROUP_CONCAT(p2.name, ', ')
			                 FROM video_performers vp2 JOIN performers p2 ON p2.id = vp2.performer_id
			                 WHERE vp2.video_id = v.id), '') AS performers,
			       COALESCE((SELECT GROUP_CONCAT(t2.name, ', ')
			                 FROM video_tags vt2 JOIN tags t2 ON t2.id = vt2.tag_id
			                 WHERE vt2.video_id = v.id), '') AS tags
			FROM videos v
			LEFT JOIN libraries l ON v.library_id = l.id
			%s
			%s
			LIMIT ?`, where, orderClause)

		qArgs = append(qArgs, limit)
		rows, err := s.db.Query(searchQ, qArgs...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		results := []map[string]interface{}{}
		for rows.Next() {
			var id int64
			var title, thumbPath, libName, performers, tags string
			var duration float64
			var rating, isFav int
			if rows.Scan(&id, &title, &duration, &rating, &isFav, &thumbPath, &libName, &performers, &tags) == nil {
				results = append(results, map[string]interface{}{
					"id": id, "title": title,
					"duration":       formatSeconds(duration),
					"duration_s":     int(duration),
					"rating":         rating,
					"is_favorite":    isFav > 0,
					"thumbnail_path": thumbPath,
					"library":        libName,
					"performers":     performers,
					"tags":           tags,
				})
			}
		}
		return map[string]interface{}{"count": len(results), "videos": results}, nil

	case "search_performers":
		query, _ := args["query"].(string)
		limit := intArg(args, "limit", 50)
		rows, err := s.db.Query(`SELECT p.id, p.name, p.video_count FROM performers p WHERE p.name LIKE ? ORDER BY p.video_count DESC LIMIT ?`, "%"+query+"%", limit)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		results := []map[string]interface{}{}
		for rows.Next() {
			var id int64
			var name string
			var videoCount int
			if rows.Scan(&id, &name, &videoCount) == nil {
				results = append(results, map[string]interface{}{"id": id, "name": name, "video_count": videoCount})
			}
		}
		return results, nil

	case "get_top_performers":
		limit := intArg(args, "limit", 10)
		rows, err := s.db.Query(`SELECT p.name, COUNT(vp.video_id) as cnt FROM performers p LEFT JOIN video_performers vp ON p.id = vp.performer_id GROUP BY p.id ORDER BY cnt DESC LIMIT ?`, limit)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		results := []map[string]interface{}{}
		for rows.Next() {
			var name string
			var cnt int
			if rows.Scan(&name, &cnt) == nil {
				results = append(results, map[string]interface{}{"name": name, "video_count": cnt})
			}
		}
		return results, nil

	case "get_all_tags":
		rows, err := s.db.Query(`SELECT t.id, t.name, COUNT(vt.video_id) as cnt FROM tags t LEFT JOIN video_tags vt ON t.id = vt.tag_id GROUP BY t.id ORDER BY cnt DESC LIMIT 100`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		results := []map[string]interface{}{}
		for rows.Next() {
			var id int64
			var name string
			var cnt int
			if rows.Scan(&id, &name, &cnt) == nil {
				results = append(results, map[string]interface{}{"id": id, "name": name, "video_count": cnt})
			}
		}
		return results, nil

	case "get_all_libraries":
		rows, err := s.db.Query(`SELECT l.id, l.name, l.path, COUNT(v.id) as cnt FROM libraries l LEFT JOIN videos v ON l.id = v.library_id GROUP BY l.id`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		results := []map[string]interface{}{}
		for rows.Next() {
			var id int64
			var name, path string
			var cnt int
			if rows.Scan(&id, &name, &path, &cnt) == nil {
				results = append(results, map[string]interface{}{"id": id, "name": name, "path": path, "video_count": cnt})
			}
		}
		return results, nil

	case "save_memory":
		key, _ := args["key"].(string)
		value, _ := args["value"].(string)
		category, _ := args["category"].(string)
		importance := intArg(args, "importance", 5)
		if key == "" || value == "" {
			return map[string]interface{}{"error": "key and value are required"}, nil
		}
		mem := &models.Memory{Key: key, Value: value, Category: category, Importance: importance}
		err := s.SaveMemory(mem)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"saved": true, "key": key}, nil

	case "recall_memories":
		query, _ := args["query"].(string)
		if query == "" {
			memories, err := s.GetMemories("", 20)
			if err != nil {
				return nil, err
			}
			return memories, nil
		}
		memories, err := s.SearchMemories(query)
		if err != nil {
			return nil, err
		}
		return memories, nil

	case "get_running_tasks":
		rows, err := s.db.Query(`SELECT id, task_type, status, message, created_at FROM activity_logs WHERE status = 'running' ORDER BY created_at DESC LIMIT 20`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		results := []map[string]interface{}{}
		for rows.Next() {
			var id int64
			var taskType, status, message string
			var createdAt time.Time
			if rows.Scan(&id, &taskType, &status, &message, &createdAt) == nil {
				results = append(results, map[string]interface{}{
					"id": id, "task_type": taskType, "status": status, "message": message,
					"started_ago": formatDuration(time.Since(createdAt)),
				})
			}
		}
		return map[string]interface{}{"running_tasks": results, "count": len(results)}, nil

	case "get_scraper_status":
		result := map[string]interface{}{}
		// Video counts per source
		sources := []string{"pornhub", "luxuretv", "sexuria", "rule34video"}
		for _, src := range sources {
			var cnt int
			s.db.QueryRow("SELECT COUNT(*) FROM scraped_videos WHERE source = ?", src).Scan(&cnt)
			result[src+"_count"] = cnt
		}
		var threadCount, postCount int
		s.db.QueryRow("SELECT COUNT(*) FROM scraped_threads").Scan(&threadCount)
		s.db.QueryRow("SELECT COUNT(*) FROM scraped_posts").Scan(&postCount)
		result["simpcity_threads"] = threadCount
		result["simpcity_posts"] = postCount
		// Recent scraper activity
		rows, err := s.db.Query(`SELECT task_type, status, message, created_at FROM activity_logs WHERE task_type LIKE '%scrape%' OR task_type LIKE '%scraper%' ORDER BY created_at DESC LIMIT 5`)
		if err == nil {
			defer rows.Close()
			recent := []map[string]interface{}{}
			for rows.Next() {
				var taskType, status, message string
				var createdAt time.Time
				if rows.Scan(&taskType, &status, &message, &createdAt) == nil {
					recent = append(recent, map[string]interface{}{"task_type": taskType, "status": status, "message": message, "when": formatDuration(time.Since(createdAt)) + " ago"})
				}
			}
			result["recent_activity"] = recent
		}
		return result, nil

	case "get_jdownloader_status":
		result := map[string]interface{}{}
		var connType, connStatus string
		err := s.db.QueryRow("SELECT connection_type, connection_status FROM jdownloader_config LIMIT 1").Scan(&connType, &connStatus)
		if err != nil {
			result["configured"] = false
			return result, nil
		}
		result["configured"] = true
		result["connection_type"] = connType
		result["connection_status"] = connStatus
		// Recent history
		var sentCount, failedCount int
		s.db.QueryRow("SELECT COUNT(*) FROM jdownloader_history WHERE status = 'sent'").Scan(&sentCount)
		s.db.QueryRow("SELECT COUNT(*) FROM jdownloader_history WHERE status = 'failed'").Scan(&failedCount)
		result["total_sent"] = sentCount
		result["total_failed"] = failedCount
		// Recent sends
		rows, err := s.db.Query(`SELECT link_url, package_name, status, sent_at FROM jdownloader_history ORDER BY sent_at DESC LIMIT 5`)
		if err == nil {
			defer rows.Close()
			recent := []map[string]interface{}{}
			for rows.Next() {
				var url, pkg, status string
				var sentAt time.Time
				if rows.Scan(&url, &pkg, &status, &sentAt) == nil {
					recent = append(recent, map[string]interface{}{"url": url, "package": pkg, "status": status, "when": formatDuration(time.Since(sentAt)) + " ago"})
				}
			}
			result["recent_sends"] = recent
		}
		return result, nil

	case "get_recent_errors":
		limit := intArg(args, "limit", 10)
		rows, err := s.db.Query(`SELECT source, level, message, created_at FROM console_logs WHERE level = 'error' ORDER BY created_at DESC LIMIT ?`, limit)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		results := []map[string]interface{}{}
		for rows.Next() {
			var source, level, message string
			var createdAt time.Time
			if rows.Scan(&source, &level, &message, &createdAt) == nil {
				results = append(results, map[string]interface{}{"source": source, "level": level, "message": message, "when": formatDuration(time.Since(createdAt)) + " ago"})
			}
		}
		return map[string]interface{}{"errors": results, "count": len(results)}, nil

	case "search_scraped_content":
		query, _ := args["query"].(string)
		source, _ := args["source"].(string)
		limit := intArg(args, "limit", 10)
		if query == "" {
			return map[string]interface{}{"error": "query is required"}, nil
		}
		sqlQuery := `SELECT source, title, performers, url FROM scraped_videos WHERE (title LIKE ? OR performers LIKE ? OR tags LIKE ?)`
		sqlArgs := []interface{}{"%" + query + "%", "%" + query + "%", "%" + query + "%"}
		if source != "" {
			sqlQuery += " AND source = ?"
			sqlArgs = append(sqlArgs, source)
		}
		sqlQuery += " ORDER BY discovered_at DESC LIMIT ?"
		sqlArgs = append(sqlArgs, limit)
		rows, err := s.db.Query(sqlQuery, sqlArgs...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		results := []map[string]interface{}{}
		for rows.Next() {
			var src, title, performers, url string
			if rows.Scan(&src, &title, &performers, &url) == nil {
				results = append(results, map[string]interface{}{"source": src, "title": title, "performers": performers, "url": url})
			}
		}
		return map[string]interface{}{"results": results, "count": len(results)}, nil

	case "get_recent_activity":
		limit := intArg(args, "limit", 10)
		rows, err := s.db.Query(`SELECT task_type, status, message, created_at FROM activity_logs ORDER BY created_at DESC LIMIT ?`, limit)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		results := []map[string]interface{}{}
		for rows.Next() {
			var taskType, status, message string
			var createdAt time.Time
			if rows.Scan(&taskType, &status, &message, &createdAt) == nil {
				results = append(results, map[string]interface{}{"task_type": taskType, "status": status, "message": message, "when": formatDuration(time.Since(createdAt)) + " ago"})
			}
		}
		return results, nil

	case "suggest_scraper_config":
		page1URL, _ := args["page1_url"].(string)
		page2URL, _ := args["page2_url"].(string)
		if page1URL == "" {
			return map[string]interface{}{"error": "page1_url is required"}, nil
		}
		return s.fetchSiteInfo(page1URL, page2URL)

	case "get_library_videos":
		libraryName, _ := args["library_name"].(string)
		limit := intArg(args, "limit", 20)
		var libID int64
		var libPath string
		err := s.db.QueryRow("SELECT id, path FROM libraries WHERE name LIKE ?", "%"+libraryName+"%").Scan(&libID, &libPath)
		if err != nil {
			return map[string]interface{}{"error": "library not found: " + libraryName}, nil
		}
		rows, err := s.db.Query(`
			SELECT v.id, v.title, COALESCE(v.thumbnail_path,''), COALESCE(v.rating,0), COALESCE(v.is_favorite,0),
			       GROUP_CONCAT(p.name, ', ') as performers
			FROM videos v
			LEFT JOIN video_performers vp ON v.id = vp.video_id
			LEFT JOIN performers p ON vp.performer_id = p.id
			WHERE v.library_id = ?
			GROUP BY v.id
			ORDER BY v.created_at DESC
			LIMIT ?`, libID, limit)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		results := []map[string]interface{}{}
		for rows.Next() {
			var id int64
			var title, thumbPath string
			var rating, isFav int
			var performers sql.NullString
			if rows.Scan(&id, &title, &thumbPath, &rating, &isFav, &performers) == nil {
				results = append(results, map[string]interface{}{
					"id": id, "title": title,
					"thumbnail_path": thumbPath,
					"rating": rating, "is_favorite": isFav > 0,
					"performers": performers.String,
				})
			}
		}
		return map[string]interface{}{"library": libraryName, "videos": results, "count": len(results)}, nil

	case "scan_library_structure":
		libraryName, _ := args["library_name"].(string)
		maxDepth := intArg(args, "max_depth", 2)
		if maxDepth > 4 {
			maxDepth = 4
		}
		var libID int64
		var libPath string
		err := s.db.QueryRow("SELECT id, path FROM libraries WHERE name LIKE ?", "%"+libraryName+"%").Scan(&libID, &libPath)
		if err != nil {
			return map[string]interface{}{"error": "library not found: " + libraryName}, nil
		}
		type folderInfo struct {
			RelPath    string
			Depth      int
			VideoCount int
		}
		folders := []map[string]interface{}{}
		_ = filepath.Walk(libPath, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil || !info.IsDir() || path == libPath {
				return nil
			}
			relPath, _ := filepath.Rel(libPath, path)
			parts := strings.Split(filepath.ToSlash(relPath), "/")
			depth := len(parts)
			if depth > maxDepth {
				return filepath.SkipDir
			}
			var count int
			s.db.QueryRow("SELECT COUNT(*) FROM videos WHERE library_id = ? AND file_path LIKE ?",
				libID, "%"+string(filepath.Separator)+relPath+string(filepath.Separator)+"%").Scan(&count)
			// Also check files directly in this folder
			var direct int
			s.db.QueryRow("SELECT COUNT(*) FROM videos WHERE library_id = ? AND file_path LIKE ?",
				libID, "%"+string(filepath.Separator)+relPath+string(filepath.Separator)+"%.%").Scan(&direct)
			folders = append(folders, map[string]interface{}{
				"path": filepath.ToSlash(relPath), "depth": depth, "video_count": count + direct,
			})
			return nil
		})
		return map[string]interface{}{
			"library": libraryName, "root": libPath,
			"folders": folders, "total_folders": len(folders),
		}, nil

	case "suggest_video_placement":
		videoID := int64(intArg(args, "video_id", 0))
		targetLibraryName, _ := args["target_library"].(string)
		if targetLibraryName == "" {
			targetLibraryName = "Backup"
		}
		// Get video info
		var title, filePath string
		var srcLibID int64
		err := s.db.QueryRow("SELECT title, file_path, library_id FROM videos WHERE id = ?", videoID).Scan(&title, &filePath, &srcLibID)
		if err != nil {
			return map[string]interface{}{"error": fmt.Sprintf("video %d not found", videoID)}, nil
		}
		var srcLibName string
		s.db.QueryRow("SELECT name FROM libraries WHERE id = ?", srcLibID).Scan(&srcLibName)
		// Get performers
		perfRows, _ := s.db.Query("SELECT p.name FROM performers p JOIN video_performers vp ON p.id = vp.performer_id WHERE vp.video_id = ?", videoID)
		performers := []string{}
		for perfRows.Next() {
			var n string
			perfRows.Scan(&n)
			performers = append(performers, n)
		}
		perfRows.Close()
		// Get tags
		tagRows, _ := s.db.Query("SELECT t.name FROM tags t JOIN video_tags vt ON t.id = vt.tag_id WHERE vt.video_id = ?", videoID)
		tags := []string{}
		for tagRows.Next() {
			var n string
			tagRows.Scan(&n)
			tags = append(tags, n)
		}
		tagRows.Close()
		// Get studios
		studioRows, _ := s.db.Query("SELECT s.name FROM studios s JOIN video_studios vs ON s.id = vs.studio_id WHERE vs.video_id = ?", videoID)
		studios := []string{}
		for studioRows.Next() {
			var n string
			studioRows.Scan(&n)
			studios = append(studios, n)
		}
		studioRows.Close()
		// Get target library
		var tgtLibID int64
		var tgtLibPath string
		err = s.db.QueryRow("SELECT id, path FROM libraries WHERE name LIKE ?", "%"+targetLibraryName+"%").Scan(&tgtLibID, &tgtLibPath)
		if err != nil {
			return map[string]interface{}{"error": "target library not found: " + targetLibraryName}, nil
		}
		// Check learned placement memories for performers (highest priority)
		learnedFolders := map[string]string{}
		for _, p := range performers {
			if mems, err := s.SearchMemories("placement " + p); err == nil {
				for _, mem := range mems {
					if strings.Contains(mem.Value, "→") {
						parts := strings.SplitN(mem.Value, "→", 2)
						if len(parts) == 2 {
							subfolder := strings.TrimSpace(strings.Split(parts[1], " in ")[0])
							learnedFolders[subfolder] = "✅ learned: " + p
						}
					}
				}
			}
		}

		// Tier 1: content-based — find folders that already contain videos by the same performers
		performerFolders := map[string]string{} // performer name → subfolder
		for _, p := range performers {
			var existingPath string
			err2 := s.db.QueryRow(
				`SELECT v.file_path FROM videos v
				 JOIN video_performers vp ON v.id = vp.video_id
				 JOIN performers perf ON perf.id = vp.performer_id
				 WHERE v.library_id = ? AND LOWER(perf.name) = LOWER(?)
				 LIMIT 1`, tgtLibID, p).Scan(&existingPath)
			if err2 == nil && existingPath != "" {
				rel, err3 := filepath.Rel(tgtLibPath, filepath.Dir(existingPath))
				if err3 == nil {
					performerFolders[p] = filepath.ToSlash(rel)
				}
			}
		}

		// Walk target library folders (max 3 levels)
		type scored struct {
			path    string
			score   float64
			reason  string
			learned bool
		}
		var candidates []scored
		_ = filepath.Walk(tgtLibPath, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil || !info.IsDir() || path == tgtLibPath {
				return nil
			}
			relPath, _ := filepath.Rel(tgtLibPath, path)
			parts := strings.Split(filepath.ToSlash(relPath), "/")
			if len(parts) > 3 {
				return filepath.SkipDir
			}
			folderSlash := filepath.ToSlash(relPath)
			folderLower := strings.ToLower(folderSlash)
			sc := 0.0
			var reasons []string
			isLearned := false

			if reason, ok := learnedFolders[folderSlash]; ok {
				sc += 20
				reasons = append(reasons, reason)
				isLearned = true
			}
			// Tier 1: folder already contains videos by this performer (content-based, highest confidence)
			for _, p := range performers {
				if existing, ok := performerFolders[p]; ok && strings.EqualFold(existing, folderSlash) {
					sc += 15
					reasons = append(reasons, "existing videos by "+p)
				}
			}
			// Tier 2: exact full performer name in folder base or path (full name only, not word fragments)
			for _, p := range performers {
				folderBase := strings.ToLower(filepath.Base(filepath.FromSlash(folderSlash)))
				if strings.EqualFold(folderBase, p) {
					sc += 10
					reasons = append(reasons, "performer folder: "+p)
				} else if strings.Contains(folderLower, strings.ToLower(p)) && len(strings.Fields(p)) > 1 {
					// Only multi-word names (e.g. "Nicole Aniston") allowed as substring — avoids single word false positives
					sc += 8
					reasons = append(reasons, "performer in path: "+p)
				}
			}
			for _, st := range studios {
				if strings.Contains(folderLower, strings.ToLower(st)) {
					sc += 7
					reasons = append(reasons, "studio: "+st)
				}
			}
			for _, t := range tags {
				if strings.Contains(folderLower, strings.ToLower(t)) {
					sc += 4
					reasons = append(reasons, "tag: "+t)
				}
			}
			// No title/filename word matching — leads to false positives
			if sc > 0 {
				candidates = append(candidates, scored{
					path: folderSlash, score: sc,
					reason: strings.Join(reasons, "; "), learned: isLearned,
				})
			}
			return nil
		})
		sort.Slice(candidates, func(i, j int) bool { return candidates[i].score > candidates[j].score })
		suggestions := []map[string]interface{}{}
		for i, c := range candidates {
			if i >= 5 {
				break
			}
			suggestions = append(suggestions, map[string]interface{}{
				"subfolder": c.path, "score": c.score, "reason": c.reason, "learned": c.learned,
			})
		}
		return map[string]interface{}{
			"video_id": videoID, "title": title,
			"source_library": srcLibName, "current_path": filepath.Base(filePath),
			"performers": performers, "tags": tags, "studios": studios,
			"target_library": targetLibraryName, "suggestions": suggestions,
		}, nil

	case "move_video_to_library":
		videoID := int64(intArg(args, "video_id", 0))
		targetLibraryName, _ := args["target_library"].(string)
		targetSubfolder, _ := args["target_subfolder"].(string)
		// Get video
		var title, filePath string
		var srcLibID int64
		err := s.db.QueryRow("SELECT title, file_path, library_id FROM videos WHERE id = ?", videoID).Scan(&title, &filePath, &srcLibID)
		if err != nil {
			return map[string]interface{}{"error": fmt.Sprintf("video %d not found", videoID)}, nil
		}
		var srcLibPath, srcLibName string
		s.db.QueryRow("SELECT name, path FROM libraries WHERE id = ?", srcLibID).Scan(&srcLibName, &srcLibPath)
		// Get target library
		var tgtLibID int64
		var tgtLibPath string
		err = s.db.QueryRow("SELECT id, path FROM libraries WHERE name LIKE ?", "%"+targetLibraryName+"%").Scan(&tgtLibID, &tgtLibPath)
		if err != nil {
			return map[string]interface{}{"error": "target library not found: " + targetLibraryName}, nil
		}
		// Build full source path (file_path may be absolute)
		srcFullPath := filePath
		if !filepath.IsAbs(filePath) {
			srcFullPath = filepath.Join(srcLibPath, filePath)
		}
		if _, statErr := os.Stat(srcFullPath); os.IsNotExist(statErr) {
			return map[string]interface{}{"error": "source file not found on disk: " + srcFullPath}, nil
		}
		filename := filepath.Base(srcFullPath)
		// Build target full path
		var tgtFullPath string
		if targetSubfolder != "" {
			tgtFullPath = filepath.Join(tgtLibPath, filepath.FromSlash(targetSubfolder), filename)
		} else {
			tgtFullPath = filepath.Join(tgtLibPath, filename)
		}
		// Check target doesn't already exist
		if _, statErr := os.Stat(tgtFullPath); statErr == nil {
			return map[string]interface{}{"error": "a file with that name already exists at the destination: " + tgtFullPath}, nil
		}
		// Create target directory
		if err := os.MkdirAll(filepath.Dir(tgtFullPath), 0755); err != nil {
			return map[string]interface{}{"error": "failed to create target directory: " + err.Error()}, nil
		}
		// Move (try rename first, fall back to copy+delete for cross-device)
		if err := os.Rename(srcFullPath, tgtFullPath); err != nil {
			srcData, readErr := os.ReadFile(srcFullPath)
			if readErr != nil {
				return map[string]interface{}{"error": "failed to read source: " + readErr.Error()}, nil
			}
			if writeErr := os.WriteFile(tgtFullPath, srcData, 0644); writeErr != nil {
				return map[string]interface{}{"error": "failed to write target: " + writeErr.Error()}, nil
			}
			os.Remove(srcFullPath)
		}
		// Update DB record
		_, dbErr := s.db.Exec("UPDATE videos SET file_path = ?, library_id = ?, updated_at = ? WHERE id = ?",
			tgtFullPath, tgtLibID, time.Now(), videoID)
		if dbErr != nil {
			return map[string]interface{}{
				"warning":  "file moved but DB update failed: " + dbErr.Error(),
				"new_path": tgtFullPath,
			}, nil
		}
		// Learn the pattern: remember which subfolder this performer's videos go to
		if targetSubfolder != "" {
			perfRows, _ := s.db.Query("SELECT p.name FROM performers p JOIN video_performers vp ON p.id = vp.performer_id WHERE vp.video_id = ?", videoID)
			for perfRows.Next() {
				var perfName string
				if perfRows.Scan(&perfName) == nil {
					memKey := "placement_" + strings.ToLower(strings.ReplaceAll(perfName, " ", "_"))
					_ = s.SaveMemory(&models.Memory{
						Key:        memKey,
						Value:      fmt.Sprintf("Performer '%s' → %s in %s", perfName, targetSubfolder, targetLibraryName),
						Category:   "insight",
						Importance: 8,
					})
				}
			}
			perfRows.Close()
		}
		return map[string]interface{}{
			"success":          true,
			"video_id":         videoID,
			"title":            title,
			"from_library":     srcLibName,
			"to_library":       targetLibraryName,
			"destination_path": tgtFullPath,
		}, nil

	case "analyze_library_structure":
		libraryName, _ := args["library_name"].(string)
		maxDepth := intArg(args, "max_depth", 3)
		if maxDepth > 5 {
			maxDepth = 5
		}
		// Find library
		var libID int64
		var libPath, libNameFull string
		err := s.db.QueryRow("SELECT id, name, path FROM libraries WHERE name LIKE ?", "%"+libraryName+"%").Scan(&libID, &libNameFull, &libPath)
		if err != nil {
			return map[string]interface{}{"error": "library not found: " + libraryName}, nil
		}

		// Load all videos + their metadata in this library
		type videoMeta struct {
			fileDir    string
			performers []string
			tags       []string
			studios    []string
		}
		var allVideos []videoMeta

		rows, err := s.db.Query("SELECT id, file_path FROM videos WHERE library_id = ?", libID)
		if err != nil {
			return nil, err
		}
		type idPath struct{ id int64; path string }
		var vidList []idPath
		for rows.Next() {
			var id int64
			var fp string
			if rows.Scan(&id, &fp) == nil {
				vidList = append(vidList, idPath{id, fp})
			}
		}
		rows.Close()

		for _, v := range vidList {
			vm := videoMeta{fileDir: filepath.ToSlash(filepath.Dir(v.path))}
			// performers
			pr, _ := s.db.Query("SELECT p.name FROM performers p JOIN video_performers vp ON p.id = vp.performer_id WHERE vp.video_id = ?", v.id)
			for pr.Next() {
				var n string; pr.Scan(&n); vm.performers = append(vm.performers, n)
			}
			pr.Close()
			// tags
			tr, _ := s.db.Query("SELECT t.name FROM tags t JOIN video_tags vt ON t.id = vt.tag_id WHERE vt.video_id = ?", v.id)
			for tr.Next() {
				var n string; tr.Scan(&n); vm.tags = append(vm.tags, n)
			}
			tr.Close()
			// studios
			sr, _ := s.db.Query("SELECT st.name FROM studios st JOIN video_studios vs ON st.id = vs.studio_id WHERE vs.video_id = ?", v.id)
			for sr.Next() {
				var n string; sr.Scan(&n); vm.studios = append(vm.studios, n)
			}
			sr.Close()
			allVideos = append(allVideos, vm)
		}

		// Group by folder (relative to library root, up to maxDepth)
		type folderStats struct {
			videoCount       int
			performerCounts  map[string]int
			tagCounts        map[string]int
			studioCounts     map[string]int
		}
		folderMap := map[string]*folderStats{}
		libPathSlash := filepath.ToSlash(libPath)

		for _, vm := range allVideos {
			rel := vm.fileDir
			if strings.HasPrefix(rel, libPathSlash) {
				rel = strings.TrimPrefix(rel, libPathSlash)
				rel = strings.TrimPrefix(rel, "/")
			}
			parts := strings.Split(rel, "/")
			if len(parts) > maxDepth {
				parts = parts[:maxDepth]
			}
			folderKey := strings.Join(parts, "/")
			if folderKey == "" {
				folderKey = "."
			}
			if folderMap[folderKey] == nil {
				folderMap[folderKey] = &folderStats{
					performerCounts: map[string]int{},
					tagCounts:       map[string]int{},
					studioCounts:    map[string]int{},
				}
			}
			fs := folderMap[folderKey]
			fs.videoCount++
			for _, p := range vm.performers { fs.performerCounts[p]++ }
			for _, t := range vm.tags { fs.tagCounts[t]++ }
			for _, st := range vm.studios { fs.studioCounts[st]++ }
		}

		// Detect dominant entity type per folder + overall pattern
		type folderAnalysis struct {
			Folder      string   `json:"folder"`
			VideoCount  int      `json:"video_count"`
			DominantType string  `json:"dominant_type"` // "performer", "tag", "studio", "mixed"
			TopEntities []string `json:"top_entities"`
		}
		var analyzed []folderAnalysis
		performerPatternCount := 0
		tagPatternCount := 0
		studioPatternCount := 0

		for folder, fs := range folderMap {
			if folder == "." {
				continue
			}
			topPerf := topNFromMap(fs.performerCounts, 3)
			topTag  := topNFromMap(fs.tagCounts, 3)
			topStudio := topNFromMap(fs.studioCounts, 3)

			maxPerf := 0; for _, c := range fs.performerCounts { if c > maxPerf { maxPerf = c } }
			maxTag  := 0; for _, c := range fs.tagCounts { if c > maxTag { maxTag = c } }
			maxStudio := 0; for _, c := range fs.studioCounts { if c > maxStudio { maxStudio = c } }

			domType := "mixed"
			var topEntities []string
			switch {
			case maxPerf >= maxTag && maxPerf >= maxStudio && maxPerf > 0:
				domType = "performer"
				topEntities = topPerf
				performerPatternCount++
			case maxTag >= maxPerf && maxTag >= maxStudio && maxTag > 0:
				domType = "tag"
				topEntities = topTag
				tagPatternCount++
			case maxStudio > 0:
				domType = "studio"
				topEntities = topStudio
				studioPatternCount++
			}
			analyzed = append(analyzed, folderAnalysis{
				Folder: folder, VideoCount: fs.videoCount,
				DominantType: domType, TopEntities: topEntities,
			})
		}
		sort.Slice(analyzed, func(i, j int) bool {
			return analyzed[i].VideoCount > analyzed[j].VideoCount
		})

		// Detect overall organizational pattern
		detectedPattern := "mixed"
		if performerPatternCount > tagPatternCount && performerPatternCount > studioPatternCount {
			detectedPattern = "performer-based (Performers/<Name>/)"
		} else if tagPatternCount > performerPatternCount && tagPatternCount > studioPatternCount {
			detectedPattern = "category-based (Categories/<Tag>/)"
		} else if studioPatternCount > 0 {
			detectedPattern = "studio-based (Studios/<Name>/)"
		}

		// Cap output to top 50 folders
		if len(analyzed) > 50 {
			analyzed = analyzed[:50]
		}

		return map[string]interface{}{
			"library":           libNameFull,
			"total_videos":      len(allVideos),
			"total_folders":     len(folderMap),
			"detected_pattern":  detectedPattern,
			"folders":           analyzed,
		}, nil

	case "get_daily_briefing":
		var vidCount, perfCount, downloadsCount, untaggedCount, errorCount, runningTasks int
		s.db.QueryRow("SELECT COUNT(*) FROM videos").Scan(&vidCount)
		s.db.QueryRow("SELECT COUNT(*) FROM performers").Scan(&perfCount)
		s.db.QueryRow(`SELECT COUNT(*) FROM videos v JOIN libraries l ON v.library_id = l.id WHERE LOWER(l.name) LIKE '%download%'`).Scan(&downloadsCount)
		s.db.QueryRow(`SELECT COUNT(*) FROM videos WHERE id NOT IN (SELECT DISTINCT video_id FROM video_tags)`).Scan(&untaggedCount)
		s.db.QueryRow(`SELECT COUNT(*) FROM console_logs WHERE level = 'error' AND created_at > datetime('now', '-24 hours')`).Scan(&errorCount)
		s.db.QueryRow(`SELECT COUNT(*) FROM activity_logs WHERE status = 'running'`).Scan(&runningTasks)

		var snapshotJSON string
		s.db.QueryRow("SELECT value FROM memories WHERE key = 'daily_snapshot'").Scan(&snapshotJSON)
		prevCount := vidCount
		if snapshotJSON != "" {
			var snap map[string]int
			if json.Unmarshal([]byte(snapshotJSON), &snap) == nil {
				prevCount = snap["video_count"]
			}
		}
		newVideos := vidCount - prevCount

		// Recent scraper activity
		type recentJob struct {
			TaskType  string `json:"task_type"`
			Message   string `json:"message"`
			Status    string `json:"status"`
			CreatedAt string `json:"created_at"`
		}
		var recentJobs []recentJob
		jobRows, _ := s.db.Query(`SELECT task_type, COALESCE(message,''), status, created_at FROM activity_logs
			WHERE created_at > datetime('now', '-24 hours') AND status IN ('completed','failed')
			ORDER BY created_at DESC LIMIT 5`)
		if jobRows != nil {
			for jobRows.Next() {
				var j recentJob
				jobRows.Scan(&j.TaskType, &j.Message, &j.Status, &j.CreatedAt)
				recentJobs = append(recentJobs, j)
			}
			jobRows.Close()
		}

		return map[string]interface{}{
			"total_videos":      vidCount,
			"total_performers":  perfCount,
			"new_videos_delta":  newVideos,
			"downloads_waiting": downloadsCount,
			"untagged_videos":   untaggedCount,
			"errors_24h":        errorCount,
			"running_tasks":     runningTasks,
			"recent_jobs":       recentJobs,
		}, nil

	case "find_orphaned_videos":
		libFilter, _ := args["library_name"].(string)
		limit := intArg(args, "limit", 2000)

		baseQ := `SELECT v.id, v.title, v.file_path, COALESCE(l.name, '') FROM videos v LEFT JOIN libraries l ON v.library_id = l.id`
		var qArgs []interface{}
		if libFilter != "" {
			baseQ += " WHERE LOWER(l.name) LIKE LOWER(?)"
			qArgs = append(qArgs, "%"+libFilter+"%")
		}
		baseQ += " LIMIT ?"
		qArgs = append(qArgs, limit)

		rows, err := s.db.Query(baseQ, qArgs...)
		if err != nil {
			return nil, err
		}
		type orphan struct {
			ID       int64  `json:"id"`
			Title    string `json:"title"`
			FilePath string `json:"file_path"`
			Library  string `json:"library"`
		}
		var orphans []orphan
		var scanned int
		for rows.Next() {
			var o orphan
			rows.Scan(&o.ID, &o.Title, &o.FilePath, &o.Library)
			scanned++
			if _, statErr := os.Stat(o.FilePath); os.IsNotExist(statErr) {
				orphans = append(orphans, o)
			}
		}
		rows.Close()

		return map[string]interface{}{
			"scanned":        scanned,
			"orphaned_count": len(orphans),
			"orphans":        orphans,
			"note":           "These DB entries have no file on disk. Use delete_video_records with the listed IDs to remove them — files are NOT touched.",
		}, nil

	case "delete_video_records":
		rawIDs, ok := args["video_ids"]
		if !ok {
			return map[string]interface{}{"error": "video_ids required"}, nil
		}
		idSlice, ok := rawIDs.([]interface{})
		if !ok || len(idSlice) == 0 {
			return map[string]interface{}{"error": "video_ids must be a non-empty array"}, nil
		}
		var ids []int64
		for _, raw := range idSlice {
			switch v := raw.(type) {
			case float64:
				ids = append(ids, int64(v))
			case int64:
				ids = append(ids, v)
			case int:
				ids = append(ids, int64(v))
			}
		}
		if len(ids) == 0 {
			return map[string]interface{}{"error": "no valid video IDs provided"}, nil
		}

		// Build IN clause
		ph := make([]string, len(ids))
		iArgs := make([]interface{}, len(ids))
		for i, id := range ids {
			ph[i] = "?"
			iArgs[i] = id
		}
		inClause := strings.Join(ph, ",")

		// Delete from junction tables first, then videos
		tables := []string{"video_performers", "video_tags", "video_studios", "dashboard_feed_items"}
		colMap := map[string]string{
			"video_performers":    "video_id",
			"video_tags":          "video_id",
			"video_studios":       "video_id",
			"dashboard_feed_items": "item_id",
		}
		for _, tbl := range tables {
			col := colMap[tbl]
			_, _ = s.db.Exec(fmt.Sprintf("DELETE FROM %s WHERE %s IN (%s)", tbl, col, inClause), iArgs...)
		}
		res, err := s.db.Exec(fmt.Sprintf("DELETE FROM videos WHERE id IN (%s)", inClause), iArgs...)
		if err != nil {
			return map[string]interface{}{"error": "delete failed: " + err.Error()}, nil
		}
		deleted, _ := res.RowsAffected()
		log.Printf("[AI] delete_video_records: removed %d DB entries (IDs: %v)", deleted, ids)

		return map[string]interface{}{
			"deleted_count": deleted,
			"requested_ids": ids,
			"note":          "Database records removed. No files were touched on disk.",
		}, nil

	case "find_duplicate_videos":
		targetLib, _ := args["library_name"].(string)
		limit := intArg(args, "limit", 30)

		type vidInfo struct {
			ID        int64
			Title     string
			FilePath  string
			Duration  float64
			LibID     int64
			LibName   string
			Filename  string
			NormTitle string
			PerfKey   string
		}

		baseQ := `SELECT v.id, v.title, v.file_path, COALESCE(v.duration, 0), v.library_id, l.name
		           FROM videos v JOIN libraries l ON v.library_id = l.id`
		var qArgs []interface{}
		if targetLib != "" {
			baseQ += " WHERE LOWER(l.name) LIKE ?"
			qArgs = append(qArgs, "%"+strings.ToLower(targetLib)+"%")
		}
		rows, err := s.db.Query(baseQ, qArgs...)
		if err != nil {
			return nil, err
		}
		var videos []vidInfo
		for rows.Next() {
			var v vidInfo
			rows.Scan(&v.ID, &v.Title, &v.FilePath, &v.Duration, &v.LibID, &v.LibName)
			v.Filename = strings.ToLower(filepath.Base(v.FilePath))
			v.NormTitle = normalizeVideoTitle(v.Title)
			videos = append(videos, v)
		}
		rows.Close()

		// Load performers per video — sorted so the key is order-stable
		perfsByVideo := map[int64][]string{}
		perfRows, _ := s.db.Query(`SELECT vp.video_id, p.name FROM video_performers vp
			JOIN performers p ON p.id = vp.performer_id ORDER BY vp.video_id, p.name`)
		if perfRows != nil {
			for perfRows.Next() {
				var vid int64
				var name string
				perfRows.Scan(&vid, &name)
				perfsByVideo[vid] = append(perfsByVideo[vid], name)
			}
			perfRows.Close()
		}
		for i := range videos {
			if perfs := perfsByVideo[videos[i].ID]; len(perfs) > 0 {
				videos[i].PerfKey = strings.Join(perfs, "|")
			}
		}

		type dupGroup struct {
			Reason     string                   `json:"reason"`
			Confidence float64                  `json:"confidence"`
			Videos     []map[string]interface{} `json:"videos"`
		}

		seen := map[string]bool{}
		var groups []dupGroup

		toEntry := func(v vidInfo) map[string]interface{} {
			return map[string]interface{}{
				"id": v.ID, "title": v.Title,
				"library": v.LibName, "file": v.Filename,
				"duration_s": int(v.Duration),
			}
		}
		pairKey := func(a, b int64) string {
			if a < b {
				return fmt.Sprintf("%d:%d", a, b)
			}
			return fmt.Sprintf("%d:%d", b, a)
		}
		addPair := func(a, b vidInfo, reason string, conf float64) {
			pk := pairKey(a.ID, b.ID)
			if seen[pk] {
				return
			}
			seen[pk] = true
			groups = append(groups, dupGroup{
				Reason:     reason,
				Confidence: conf,
				Videos:     []map[string]interface{}{toEntry(a), toEntry(b)},
			})
		}

		// Tier 1: Identical filename (highest confidence)
		filenameIdx := map[string][]vidInfo{}
		for _, v := range videos {
			if v.Filename == "" {
				continue
			}
			filenameIdx[v.Filename] = append(filenameIdx[v.Filename], v)
		}
		for _, vids := range filenameIdx {
			if len(vids) < 2 {
				continue
			}
			for i := 0; i < len(vids); i++ {
				for j := i + 1; j < len(vids); j++ {
					addPair(vids[i], vids[j], "identical filename", 0.95)
				}
			}
		}

		// Tier 2: Matching normalized title across different libraries
		titleIdx := map[string][]vidInfo{}
		for _, v := range videos {
			if len(v.NormTitle) < 5 {
				continue
			}
			titleIdx[v.NormTitle] = append(titleIdx[v.NormTitle], v)
		}
		for _, vids := range titleIdx {
			if len(vids) < 2 {
				continue
			}
			for i := 0; i < len(vids); i++ {
				for j := i + 1; j < len(vids); j++ {
					if vids[i].LibID == vids[j].LibID {
						continue
					}
					durDiff := vids[i].Duration - vids[j].Duration
					if durDiff < 0 {
						durDiff = -durDiff
					}
					conf := 0.75
					reason := "same title (different libraries)"
					if durDiff <= 10 {
						conf = 0.88
						reason = fmt.Sprintf("same title, duration matches (±%.0fs)", durDiff)
					}
					addPair(vids[i], vids[j], reason, conf)
				}
			}
		}

		// Tier 3: Same performer set + similar duration across different libraries
		// Group by performer key + duration bucket (nearest 15s)
		perfDurIdx := map[string][]vidInfo{}
		for _, v := range videos {
			if v.PerfKey == "" {
				continue
			}
			bucket := (int(v.Duration) / 15) * 15
			key := fmt.Sprintf("%s@%d", v.PerfKey, bucket)
			perfDurIdx[key] = append(perfDurIdx[key], v)
		}
		for _, vids := range perfDurIdx {
			if len(vids) < 2 {
				continue
			}
			for i := 0; i < len(vids); i++ {
				for j := i + 1; j < len(vids); j++ {
					if vids[i].LibID == vids[j].LibID {
						continue
					}
					durDiff := vids[i].Duration - vids[j].Duration
					if durDiff < 0 {
						durDiff = -durDiff
					}
					if durDiff > 30 {
						continue
					}
					addPair(vids[i], vids[j],
						fmt.Sprintf("same performers, similar duration (±%.0fs)", durDiff), 0.70)
				}
			}
		}

		// Sort by confidence desc, cap at limit
		sort.Slice(groups, func(i, j int) bool { return groups[i].Confidence > groups[j].Confidence })
		if len(groups) > limit {
			groups = groups[:limit]
		}

		return map[string]interface{}{
			"total_suspects": len(groups),
			"duplicates":     groups,
			"note":           "Review each pair — same filename/title is a strong signal, same performers+duration is weaker. Use video IDs to move or delete.",
		}, nil

	case "get_performer_stats":
		performerQuery, _ := args["performer"].(string)
		if performerQuery == "" {
			return map[string]interface{}{"error": "performer name required"}, nil
		}
		// Find performer — prefer exact match, fall back to LIKE
		var perfID int64
		var perfName string
		err := s.db.QueryRow(
			"SELECT id, name FROM performers WHERE LOWER(name) = LOWER(?) LIMIT 1", performerQuery).Scan(&perfID, &perfName)
		if err != nil {
			err = s.db.QueryRow(
				"SELECT id, name FROM performers WHERE LOWER(name) LIKE LOWER(?) ORDER BY LENGTH(name) ASC LIMIT 1",
				"%"+performerQuery+"%").Scan(&perfID, &perfName)
		}
		if err != nil {
			return map[string]interface{}{"error": "performer not found: " + performerQuery}, nil
		}

		// Totals: count, duration, avg rating
		var videoCount, favCount int
		var totalDuration, avgRating float64
		s.db.QueryRow(`
			SELECT COUNT(*),
			       COALESCE(SUM(v.duration), 0),
			       COALESCE(AVG(CASE WHEN v.rating > 0 THEN CAST(v.rating AS FLOAT) END), 0)
			FROM videos v JOIN video_performers vp ON v.id = vp.video_id
			WHERE vp.performer_id = ?`, perfID).Scan(&videoCount, &totalDuration, &avgRating)
		s.db.QueryRow(`
			SELECT COUNT(*) FROM videos v JOIN video_performers vp ON v.id = vp.video_id
			WHERE vp.performer_id = ? AND v.is_favorite = 1`, perfID).Scan(&favCount)

		// Rating distribution
		ratingDist := map[string]int{"0★ (unrated)": 0, "1★": 0, "2★": 0, "3★": 0, "4★": 0, "5★": 0}
		rdRows, _ := s.db.Query(`
			SELECT COALESCE(v.rating, 0), COUNT(*)
			FROM videos v JOIN video_performers vp ON v.id = vp.video_id
			WHERE vp.performer_id = ?
			GROUP BY COALESCE(v.rating, 0)`, perfID)
		if rdRows != nil {
			for rdRows.Next() {
				var r, cnt int
				rdRows.Scan(&r, &cnt)
				key := fmt.Sprintf("%d★", r)
				if r == 0 {
					key = "0★ (unrated)"
				}
				ratingDist[key] = cnt
			}
			rdRows.Close()
		}

		// Library breakdown
		type nameCount struct {
			Name  string `json:"name"`
			Count int    `json:"count"`
		}
		var libBreakdown []nameCount
		lbRows, _ := s.db.Query(`
			SELECT l.name, COUNT(v.id)
			FROM videos v
			JOIN video_performers vp ON v.id = vp.video_id
			JOIN libraries l ON v.library_id = l.id
			WHERE vp.performer_id = ?
			GROUP BY l.id ORDER BY COUNT(v.id) DESC`, perfID)
		if lbRows != nil {
			for lbRows.Next() {
				var nc nameCount
				lbRows.Scan(&nc.Name, &nc.Count)
				libBreakdown = append(libBreakdown, nc)
			}
			lbRows.Close()
		}

		// Top tags (across all their videos)
		var topTags []nameCount
		tagRows, _ := s.db.Query(`
			SELECT t.name, COUNT(DISTINCT vt.video_id)
			FROM tags t
			JOIN video_tags vt ON t.id = vt.tag_id
			JOIN video_performers vp ON vt.video_id = vp.video_id
			WHERE vp.performer_id = ?
			GROUP BY t.id ORDER BY COUNT(DISTINCT vt.video_id) DESC
			LIMIT 10`, perfID)
		if tagRows != nil {
			for tagRows.Next() {
				var nc nameCount
				tagRows.Scan(&nc.Name, &nc.Count)
				topTags = append(topTags, nc)
			}
			tagRows.Close()
		}

		// Top studios
		var topStudios []nameCount
		stRows, _ := s.db.Query(`
			SELECT s.name, COUNT(DISTINCT vs.video_id)
			FROM studios s
			JOIN video_studios vs ON s.id = vs.studio_id
			JOIN video_performers vp ON vs.video_id = vp.video_id
			WHERE vp.performer_id = ?
			GROUP BY s.id ORDER BY COUNT(DISTINCT vs.video_id) DESC
			LIMIT 5`, perfID)
		if stRows != nil {
			for stRows.Next() {
				var nc nameCount
				stRows.Scan(&nc.Name, &nc.Count)
				topStudios = append(topStudios, nc)
			}
			stRows.Close()
		}

		// 5 most recent videos
		type recentVid struct {
			ID      int64  `json:"id"`
			Title   string `json:"title"`
			AddedAt string `json:"added_at"`
			Rating  int    `json:"rating"`
			Library string `json:"library"`
		}
		var recentVideos []recentVid
		recRows, _ := s.db.Query(`
			SELECT v.id, v.title, v.created_at, COALESCE(v.rating, 0), COALESCE(l.name, '')
			FROM videos v
			JOIN video_performers vp ON v.id = vp.video_id
			LEFT JOIN libraries l ON v.library_id = l.id
			WHERE vp.performer_id = ?
			ORDER BY v.created_at DESC LIMIT 5`, perfID)
		if recRows != nil {
			for recRows.Next() {
				var rv recentVid
				recRows.Scan(&rv.ID, &rv.Title, &rv.AddedAt, &rv.Rating, &rv.Library)
				recentVideos = append(recentVideos, rv)
			}
			recRows.Close()
		}

		// Date range
		var firstAdded, lastAdded string
		s.db.QueryRow(`
			SELECT MIN(v.created_at), MAX(v.created_at)
			FROM videos v JOIN video_performers vp ON v.id = vp.video_id
			WHERE vp.performer_id = ?`, perfID).Scan(&firstAdded, &lastAdded)

		return map[string]interface{}{
			"performer_id":         perfID,
			"performer_name":       perfName,
			"total_videos":         videoCount,
			"favorite_count":       favCount,
			"total_watch_time":     formatSeconds(totalDuration),
			"total_watch_time_hrs": fmt.Sprintf("%.1fh", totalDuration/3600),
			"avg_rating":           fmt.Sprintf("%.2f", avgRating),
			"rating_distribution":  ratingDist,
			"libraries":            libBreakdown,
			"top_tags":             topTags,
			"top_studios":          topStudios,
			"recent_videos":        recentVideos,
			"first_added":          firstAdded,
			"last_added":           lastAdded,
		}, nil

	case "suggest_videos_to_watch":
		count := intArg(args, "count", 5)
		if count > 20 {
			count = 20
		}
		libFilter, _ := args["library"].(string)

		// ── Step 1: Build taste profile from favorites + 4★+ videos ──────────────
		profQ := `SELECT id FROM videos WHERE is_favorite = 1 OR COALESCE(rating, 0) >= 4`
		profRows, err := s.db.Query(profQ)
		if err != nil {
			return nil, err
		}
		var profileIDs []int64
		profileSet := map[int64]bool{}
		for profRows.Next() {
			var id int64
			profRows.Scan(&id)
			profileIDs = append(profileIDs, id)
			profileSet[id] = true
		}
		profRows.Close()

		if len(profileIDs) == 0 {
			return map[string]interface{}{
				"message": "No favorites or 4★+ rated videos found. Rate or favorite some videos first so I can learn your taste.",
			}, nil
		}

		// Count performer/tag/studio frequency across profile videos
		perfFreq := map[int64]int{}
		tagFreq := map[int64]int{}
		studioFreq := map[int64]int{}

		buildInPlaceholders := func(n int) string {
			if n == 0 {
				return ""
			}
			ph := make([]string, n)
			for i := range ph {
				ph[i] = "?"
			}
			return strings.Join(ph, ",")
		}
		int64sToIface := func(ids []int64) []interface{} {
			out := make([]interface{}, len(ids))
			for i, id := range ids {
				out[i] = id
			}
			return out
		}

		if len(profileIDs) > 0 {
			ph := buildInPlaceholders(len(profileIDs))
			iargs := int64sToIface(profileIDs)

			pr, _ := s.db.Query(fmt.Sprintf("SELECT performer_id FROM video_performers WHERE video_id IN (%s)", ph), iargs...)
			if pr != nil {
				for pr.Next() {
					var pid int64
					pr.Scan(&pid)
					perfFreq[pid]++
				}
				pr.Close()
			}
			tr, _ := s.db.Query(fmt.Sprintf("SELECT tag_id FROM video_tags WHERE video_id IN (%s)", ph), iargs...)
			if tr != nil {
				for tr.Next() {
					var tid int64
					tr.Scan(&tid)
					tagFreq[tid]++
				}
				tr.Close()
			}
			sr, _ := s.db.Query(fmt.Sprintf("SELECT studio_id FROM video_studios WHERE video_id IN (%s)", ph), iargs...)
			if sr != nil {
				for sr.Next() {
					var sid int64
					sr.Scan(&sid)
					studioFreq[sid]++
				}
				sr.Close()
			}
		}

		// Sort frequency maps and extract top IDs
		type kv64 struct{ k int64; v int }
		toSortedIDs := func(m map[int64]int, n int) []int64 {
			var pairs []kv64
			for k, v := range m {
				pairs = append(pairs, kv64{k, v})
			}
			sort.Slice(pairs, func(i, j int) bool { return pairs[i].v > pairs[j].v })
			out := make([]int64, 0, n)
			for i, p := range pairs {
				if i >= n {
					break
				}
				out = append(out, p.k)
			}
			return out
		}
		perfIDList := toSortedIDs(perfFreq, 20)
		tagIDList := toSortedIDs(tagFreq, 20)
		studioIDList := toSortedIDs(studioFreq, 10)

		if len(perfIDList) == 0 && len(tagIDList) == 0 && len(studioIDList) == 0 {
			return map[string]interface{}{
				"message": "Your favorite videos don't have enough metadata (performers/tags/studios) to build recommendations. Try adding performer and tag data to your videos.",
			}, nil
		}

		// ── Step 2: Score candidates via SQL ──────────────────────────────────────
		// Build a SQL score expression using EXISTS counts
		var scoreExpr strings.Builder
		var scoreArgs []interface{}

		if len(perfIDList) > 0 {
			ph := buildInPlaceholders(len(perfIDList))
			scoreExpr.WriteString(fmt.Sprintf(`
				(SELECT COUNT(*) FROM video_performers vp2 WHERE vp2.video_id = v.id AND vp2.performer_id IN (%s)) * 3`, ph))
			for _, id := range perfIDList {
				scoreArgs = append(scoreArgs, id)
			}
		}
		if len(tagIDList) > 0 {
			ph := buildInPlaceholders(len(tagIDList))
			if scoreExpr.Len() > 0 {
				scoreExpr.WriteString(" + ")
			}
			scoreExpr.WriteString(fmt.Sprintf(`
				(SELECT COUNT(*) FROM video_tags vt2 WHERE vt2.video_id = v.id AND vt2.tag_id IN (%s)) * 1`, ph))
			for _, id := range tagIDList {
				scoreArgs = append(scoreArgs, id)
			}
		}
		if len(studioIDList) > 0 {
			ph := buildInPlaceholders(len(studioIDList))
			if scoreExpr.Len() > 0 {
				scoreExpr.WriteString(" + ")
			}
			scoreExpr.WriteString(fmt.Sprintf(`
				(SELECT COUNT(*) FROM video_studios vs2 WHERE vs2.video_id = v.id AND vs2.studio_id IN (%s)) * 2`, ph))
			for _, id := range studioIDList {
				scoreArgs = append(scoreArgs, id)
			}
		}

		// Exclude profile videos (cap at 200 to stay under SQLite's param limit)
		excludeIDs := profileIDs
		if len(excludeIDs) > 200 {
			excludeIDs = excludeIDs[:200]
		}
		excludePH := buildInPlaceholders(len(excludeIDs))

		libCond := ""
		var libArgs []interface{}
		if libFilter != "" {
			libCond = "AND LOWER(l.name) LIKE LOWER(?)"
			libArgs = append(libArgs, "%"+libFilter+"%")
		}

		candQ := fmt.Sprintf(`
			SELECT v.id, v.title, COALESCE(v.duration, 0), COALESCE(v.rating, 0), COALESCE(l.name, ''),
			       COALESCE(v.thumbnail_path, ''),
			       COALESCE((SELECT GROUP_CONCAT(p.name, ', ') FROM video_performers vp JOIN performers p ON p.id = vp.performer_id WHERE vp.video_id = v.id), '') AS performers,
			       (%s) AS score
			FROM videos v
			LEFT JOIN libraries l ON v.library_id = l.id
			WHERE v.id NOT IN (%s)
			  AND COALESCE(v.is_favorite, 0) = 0
			  %s
			HAVING score > 0
			ORDER BY score DESC, RANDOM()
			LIMIT ?`, scoreExpr.String(), excludePH, libCond)

		var allArgs []interface{}
		allArgs = append(allArgs, scoreArgs...)
		allArgs = append(allArgs, int64sToIface(excludeIDs)...)
		allArgs = append(allArgs, libArgs...)
		allArgs = append(allArgs, count*3) // fetch 3× to allow for diversity

		candRows, err := s.db.Query(candQ, allArgs...)
		if err != nil {
			log.Printf("[AI] suggest_videos_to_watch query error: %v", err)
			return nil, err
		}
		defer candRows.Close()

		type suggestion struct {
			ID            int64
			Title         string
			Duration      float64
			Rating        int
			Library       string
			ThumbnailPath string
			Performers    string
			Score         float64
		}
		var suggestions []suggestion
		for candRows.Next() {
			var s suggestion
			candRows.Scan(&s.ID, &s.Title, &s.Duration, &s.Rating, &s.Library, &s.ThumbnailPath, &s.Performers, &s.Score)
			suggestions = append(suggestions, s)
		}
		if len(suggestions) > count {
			suggestions = suggestions[:count]
		}

		// Build result, enriching each suggestion with which specific performers/tags matched
		results := make([]map[string]interface{}, 0, len(suggestions))
		for _, sg := range suggestions {
			entry := map[string]interface{}{
				"id":             sg.ID,
				"title":          sg.Title,
				"duration":       formatSeconds(sg.Duration),
				"rating":         sg.Rating,
				"thumbnail_path": sg.ThumbnailPath,
				"library":        sg.Library,
				"performers":     sg.Performers,
				"score":          sg.Score,
			}

			// Matched performers (from taste profile)
			if len(perfIDList) > 0 {
				ph := buildInPlaceholders(len(perfIDList))
				mArgs := append([]interface{}{sg.ID}, int64sToIface(perfIDList)...)
				mArgs = append(mArgs, 3)
				mRows, mErr := s.db.Query(fmt.Sprintf(
					`SELECT p.name FROM video_performers vp JOIN performers p ON p.id=vp.performer_id
					 WHERE vp.video_id=? AND vp.performer_id IN (%s) LIMIT ?`, ph), mArgs...)
				if mErr == nil {
					var mp []string
					for mRows.Next() {
						var n string
						if mRows.Scan(&n) == nil {
							mp = append(mp, n)
						}
					}
					mRows.Close()
					if len(mp) > 0 {
						entry["matched_performers"] = mp
					}
				}
			}

			// Matched tags (from taste profile)
			if len(tagIDList) > 0 {
				ph := buildInPlaceholders(len(tagIDList))
				tArgs := append([]interface{}{sg.ID}, int64sToIface(tagIDList)...)
				tArgs = append(tArgs, 3)
				tRows, tErr := s.db.Query(fmt.Sprintf(
					`SELECT t.name FROM video_tags vt JOIN tags t ON t.id=vt.tag_id
					 WHERE vt.video_id=? AND vt.tag_id IN (%s) LIMIT ?`, ph), tArgs...)
				if tErr == nil {
					var mt []string
					for tRows.Next() {
						var n string
						if tRows.Scan(&n) == nil {
							mt = append(mt, n)
						}
					}
					tRows.Close()
					if len(mt) > 0 {
						entry["matched_tags"] = mt
					}
				}
			}

			results = append(results, entry)
		}

		return map[string]interface{}{
			"suggestions":    results,
			"based_on":       fmt.Sprintf("taste profile built from %d favorite/4★+ videos", len(profileIDs)),
			"top_performers": len(perfIDList),
			"top_tags":       len(tagIDList),
		}, nil

	case "get_favorites":
		limit := intArg(args, "limit", 20)
		rows, err := s.db.Query(`
			SELECT v.id, v.title, COALESCE(v.duration, 0), COALESCE(v.rating, 0),
			       COALESCE(v.thumbnail_path, ''),
			       GROUP_CONCAT(p.name, ', ') as performers
			FROM videos v
			LEFT JOIN video_performers vp ON v.id = vp.video_id
			LEFT JOIN performers p ON vp.performer_id = p.id
			WHERE v.is_favorite = 1
			GROUP BY v.id
			ORDER BY v.updated_at DESC
			LIMIT ?`, limit)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		results := []map[string]interface{}{}
		for rows.Next() {
			var id int64
			var title, thumbPath string
			var duration float64
			var rating int
			var performers sql.NullString
			if rows.Scan(&id, &title, &duration, &rating, &thumbPath, &performers) == nil {
				results = append(results, map[string]interface{}{
					"id": id, "title": title, "duration": formatSeconds(duration),
					"rating": rating, "is_favorite": true,
					"thumbnail_path": thumbPath,
					"performers":     performers.String,
				})
			}
		}
		return map[string]interface{}{"favorites": results, "count": len(results)}, nil

	case "get_top_rated":
		limit := intArg(args, "limit", 20)
		minRating := intArg(args, "min_rating", 1)
		rows, err := s.db.Query(`
			SELECT v.id, v.title, COALESCE(v.duration, 0), v.rating,
			       COALESCE(v.thumbnail_path, ''),
			       GROUP_CONCAT(p.name, ', ') as performers
			FROM videos v
			LEFT JOIN video_performers vp ON v.id = vp.video_id
			LEFT JOIN performers p ON vp.performer_id = p.id
			WHERE v.rating >= ?
			GROUP BY v.id
			ORDER BY v.rating DESC, v.updated_at DESC
			LIMIT ?`, minRating, limit)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		results := []map[string]interface{}{}
		for rows.Next() {
			var id int64
			var title, thumbPath string
			var duration float64
			var rating int
			var performers sql.NullString
			if rows.Scan(&id, &title, &duration, &rating, &thumbPath, &performers) == nil {
				results = append(results, map[string]interface{}{
					"id": id, "title": title, "duration": formatSeconds(duration),
					"rating":         rating,
					"thumbnail_path": thumbPath,
					"performers":     performers.String,
				})
			}
		}
		return map[string]interface{}{"top_rated": results, "count": len(results)}, nil

	case "find_untagged_videos":
		libFilter, _ := args["library_name"].(string)
		perfFilter, _ := args["performer"].(string)
		limit := intArg(args, "limit", 20)
		if limit > 50 {
			limit = 50
		}

		var conds []string
		var qArgs []interface{}
		conds = append(conds, `NOT EXISTS (SELECT 1 FROM video_tags vt WHERE vt.video_id = v.id)`)
		if libFilter != "" {
			conds = append(conds, "LOWER(l.name) LIKE LOWER(?)")
			qArgs = append(qArgs, "%"+libFilter+"%")
		}
		if perfFilter != "" {
			conds = append(conds, `EXISTS (SELECT 1 FROM video_performers vp JOIN performers p ON p.id = vp.performer_id WHERE vp.video_id = v.id AND LOWER(p.name) LIKE LOWER(?))`)
			qArgs = append(qArgs, "%"+perfFilter+"%")
		}
		where := "WHERE " + strings.Join(conds, " AND ")
		qArgs = append(qArgs, limit)

		rows, err := s.db.Query(fmt.Sprintf(`
			SELECT v.id, v.title, COALESCE(v.thumbnail_path,''), COALESCE(v.rating,0),
			       COALESCE((SELECT GROUP_CONCAT(p.name,', ') FROM video_performers vp JOIN performers p ON p.id=vp.performer_id WHERE vp.video_id=v.id),'') AS performers
			FROM videos v LEFT JOIN libraries l ON l.id=v.library_id
			%s ORDER BY v.created_at DESC LIMIT ?`, where), qArgs...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		results := []map[string]interface{}{}
		for rows.Next() {
			var id int64
			var title, thumbPath, performers string
			var rating int
			if rows.Scan(&id, &title, &thumbPath, &rating, &performers) == nil {
				results = append(results, map[string]interface{}{
					"id": id, "title": title,
					"thumbnail_path": thumbPath,
					"rating":         rating,
					"performers":     performers,
				})
			}
		}
		return map[string]interface{}{
			"untagged_videos": results,
			"count":           len(results),
			"note":            "For each video, call suggest_tags_for_video(video_id) then apply_tags_to_video(video_id, tags) to tag it.",
		}, nil

	case "get_library_health":
		var total, rated, untagged, noPerformers, recentAdded int
		s.db.QueryRow(`SELECT COUNT(*) FROM videos`).Scan(&total)
		s.db.QueryRow(`SELECT COUNT(*) FROM videos WHERE rating IS NOT NULL AND rating > 0`).Scan(&rated)
		s.db.QueryRow(`SELECT COUNT(*) FROM videos v WHERE NOT EXISTS (SELECT 1 FROM video_tags vt WHERE vt.video_id = v.id)`).Scan(&untagged)
		s.db.QueryRow(`SELECT COUNT(*) FROM videos v WHERE NOT EXISTS (SELECT 1 FROM video_performers vp WHERE vp.video_id = v.id)`).Scan(&noPerformers)
		s.db.QueryRow(`SELECT COUNT(*) FROM videos WHERE created_at > datetime('now', '-7 days')`).Scan(&recentAdded)

		var totalPerformers, totalTags, totalStudios, totalLibraries int
		s.db.QueryRow(`SELECT COUNT(*) FROM performers`).Scan(&totalPerformers)
		s.db.QueryRow(`SELECT COUNT(*) FROM tags`).Scan(&totalTags)
		s.db.QueryRow(`SELECT COUNT(*) FROM studios`).Scan(&totalStudios)
		s.db.QueryRow(`SELECT COUNT(*) FROM libraries`).Scan(&totalLibraries)

		// Library breakdown
		libRows, _ := s.db.Query(`SELECT l.name, COUNT(v.id) as cnt FROM libraries l LEFT JOIN videos v ON v.library_id = l.id GROUP BY l.id, l.name ORDER BY cnt DESC`)
		var libraries []map[string]interface{}
		if libRows != nil {
			for libRows.Next() {
				var lname string
				var cnt int
				if libRows.Scan(&lname, &cnt) == nil {
					libraries = append(libraries, map[string]interface{}{"name": lname, "videos": cnt})
				}
			}
			libRows.Close()
		}

		// Health score: weighted penalties per issue category
		score := 100
		if total > 0 {
			untaggedPct := untagged * 100 / total
			unratedPct := (total - rated) * 100 / total
			noPerformerPct := noPerformers * 100 / total
			if untaggedPct > 80 {
				score -= 30
			} else if untaggedPct > 50 {
				score -= 20
			} else if untaggedPct > 20 {
				score -= 10
			}
			if unratedPct > 80 {
				score -= 25
			} else if unratedPct > 60 {
				score -= 15
			} else if unratedPct > 40 {
				score -= 8
			}
			if noPerformerPct > 50 {
				score -= 20
			} else if noPerformerPct > 25 {
				score -= 10
			} else if noPerformerPct > 10 {
				score -= 5
			}
		}
		if score < 0 {
			score = 0
		}

		grade := "A"
		switch {
		case score < 40:
			grade = "D"
		case score < 55:
			grade = "C"
		case score < 70:
			grade = "B"
		}

		return map[string]interface{}{
			"health_score":        score,
			"health_grade":        grade,
			"totals": map[string]interface{}{
				"videos": total, "performers": totalPerformers,
				"tags": totalTags, "studios": totalStudios, "libraries": totalLibraries,
			},
			"untagged":         untagged,
			"untagged_pct":     func() int { if total == 0 { return 0 }; return untagged * 100 / total }(),
			"unrated":          total - rated,
			"unrated_pct":      func() int { if total == 0 { return 0 }; return (total - rated) * 100 / total }(),
			"no_performers":    noPerformers,
			"no_performer_pct": func() int { if total == 0 { return 0 }; return noPerformers * 100 / total }(),
			"recent_additions": recentAdded,
			"libraries":        libraries,
		}, nil

	case "bulk_apply_tags":
		rawIDs, ok := args["video_ids"]
		if !ok {
			return map[string]interface{}{"error": "video_ids required"}, nil
		}
		idSlice, ok := rawIDs.([]interface{})
		if !ok || len(idSlice) == 0 {
			return map[string]interface{}{"error": "video_ids must be a non-empty array"}, nil
		}
		maxTags := intArg(args, "max_tags_per_video", 5)
		if maxTags < 1 {
			maxTags = 1
		}
		if maxTags > 15 {
			maxTags = 15
		}

		var videoIDs []int64
		for _, r := range idSlice {
			switch v := r.(type) {
			case float64:
				videoIDs = append(videoIDs, int64(v))
			case int64:
				videoIDs = append(videoIDs, v)
			case int:
				videoIDs = append(videoIDs, int64(v))
			}
		}
		if len(videoIDs) == 0 {
			return map[string]interface{}{"error": "no valid video IDs"}, nil
		}

		type bulkResult struct {
			ID          int64    `json:"id"`
			Title       string   `json:"title"`
			TagsApplied []string `json:"tags_applied"`
			Skipped     bool     `json:"skipped,omitempty"` // no performers
		}

		var results []bulkResult
		totalApplied := 0

		perfPH := func(n int) string {
			p := make([]string, n)
			for i := range p {
				p[i] = "?"
			}
			return strings.Join(p, ",")
		}

		for _, vid := range videoIDs {
			var title string
			s.db.QueryRow("SELECT title FROM videos WHERE id=?", vid).Scan(&title)

			// Get this video's performer IDs
			perfRows, _ := s.db.Query("SELECT performer_id FROM video_performers WHERE video_id=?", vid)
			var perfArgs []interface{}
			for perfRows.Next() {
				var pid int64
				if perfRows.Scan(&pid) == nil {
					perfArgs = append(perfArgs, pid)
				}
			}
			perfRows.Close()

			if len(perfArgs) == 0 {
				results = append(results, bulkResult{ID: vid, Title: title, Skipped: true})
				continue
			}

			// Find top tags from sibling videos by same performers, excluding already-applied tags
			qArgs := append(perfArgs, vid, vid, maxTags)
			tagRows, err := s.db.Query(fmt.Sprintf(`
				SELECT t.id, t.name, COUNT(*) as freq
				FROM video_performers vp
				JOIN video_tags vt ON vt.video_id = vp.video_id
				JOIN tags t ON t.id = vt.tag_id
				WHERE vp.performer_id IN (%s)
				  AND vp.video_id != ?
				  AND NOT EXISTS (SELECT 1 FROM video_tags ex WHERE ex.video_id=? AND ex.tag_id=t.id)
				GROUP BY t.id, t.name
				ORDER BY freq DESC
				LIMIT ?`, perfPH(len(perfArgs))), qArgs...)

			var applied []string
			if err == nil {
				for tagRows.Next() {
					var tagID int64
					var tagName string
					var freq int
					if tagRows.Scan(&tagID, &tagName, &freq) == nil {
						if _, err := s.db.Exec("INSERT OR IGNORE INTO video_tags (video_id, tag_id) VALUES (?,?)", vid, tagID); err == nil {
							applied = append(applied, tagName)
						}
					}
				}
				tagRows.Close()
			}

			results = append(results, bulkResult{ID: vid, Title: title, TagsApplied: applied})
			totalApplied += len(applied)
		}

		skipped := 0
		for _, r := range results {
			if r.Skipped {
				skipped++
			}
		}
		log.Printf("[AI] bulk_apply_tags: processed %d videos, applied %d tags total", len(videoIDs), totalApplied)

		return map[string]interface{}{
			"processed":          len(videoIDs),
			"total_tags_applied": totalApplied,
			"skipped_no_performers": skipped,
			"videos":             results,
		}, nil

	case "suggest_tags_for_video":
		videoID := int64(floatArg(args, "video_id", 0))
		limit := intArg(args, "limit", 10)
		if videoID == 0 {
			return map[string]interface{}{"error": "video_id required"}, nil
		}

		// Get performers on this video
		perfRows, err := s.db.Query(`
			SELECT p.id FROM video_performers vp
			JOIN performers p ON p.id = vp.performer_id
			WHERE vp.video_id = ?`, videoID)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}, nil
		}
		var perfIDs []interface{}
		for perfRows.Next() {
			var pid int64
			if perfRows.Scan(&pid) == nil {
				perfIDs = append(perfIDs, pid)
			}
		}
		perfRows.Close()

		if len(perfIDs) == 0 {
			return map[string]interface{}{"suggestions": []interface{}{}, "note": "No performers linked to this video — cannot suggest tags"}, nil
		}

		// Get tags already on this video
		existingRows, _ := s.db.Query(`SELECT tag_id FROM video_tags WHERE video_id = ?`, videoID)
		existingTags := map[int64]bool{}
		for existingRows.Next() {
			var tid int64
			if existingRows.Scan(&tid) == nil {
				existingTags[tid] = true
			}
		}
		existingRows.Close()

		// Find top tags applied to other videos by same performers, excluding already-applied tags
		ph := make([]string, len(perfIDs))
		for i := range perfIDs {
			ph[i] = "?"
		}
		inClause := strings.Join(ph, ",")
		qArgs := append(perfIDs, videoID)
		tagRows, err := s.db.Query(fmt.Sprintf(`
			SELECT t.id, t.name, COUNT(*) as freq
			FROM video_performers vp
			JOIN video_tags vt ON vt.video_id = vp.video_id
			JOIN tags t ON t.id = vt.tag_id
			WHERE vp.performer_id IN (%s)
			  AND vp.video_id != ?
			GROUP BY t.id, t.name
			ORDER BY freq DESC
			LIMIT ?`, inClause), append(qArgs, limit)...)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}, nil
		}
		defer tagRows.Close()

		var suggestions []map[string]interface{}
		for tagRows.Next() {
			var tid int64
			var tname string
			var freq int
			if tagRows.Scan(&tid, &tname, &freq) == nil && !existingTags[tid] {
				suggestions = append(suggestions, map[string]interface{}{
					"tag_id": tid, "name": tname, "frequency": freq,
				})
			}
		}
		if suggestions == nil {
			suggestions = []map[string]interface{}{}
		}
		return map[string]interface{}{
			"video_id":    videoID,
			"suggestions": suggestions,
			"count":       len(suggestions),
		}, nil

	case "apply_tags_to_video":
		videoID := int64(floatArg(args, "video_id", 0))
		if videoID == 0 {
			return map[string]interface{}{"error": "video_id required"}, nil
		}
		rawTags, ok := args["tags"]
		if !ok {
			return map[string]interface{}{"error": "tags array required"}, nil
		}
		tagSlice, ok := rawTags.([]interface{})
		if !ok || len(tagSlice) == 0 {
			return map[string]interface{}{"error": "tags must be a non-empty array"}, nil
		}

		var applied []string
		var skipped []string
		for _, rawTag := range tagSlice {
			tagName, ok := rawTag.(string)
			if !ok || strings.TrimSpace(tagName) == "" {
				continue
			}
			tagName = strings.TrimSpace(tagName)

			// Find or create tag
			var tagID int64
			err := s.db.QueryRow(`SELECT id FROM tags WHERE LOWER(name) = LOWER(?)`, tagName).Scan(&tagID)
			if err == sql.ErrNoRows {
				res, err := s.db.Exec(`INSERT INTO tags (name) VALUES (?)`, tagName)
				if err != nil {
					skipped = append(skipped, tagName+" (create failed)")
					continue
				}
				tagID, _ = res.LastInsertId()
			} else if err != nil {
				skipped = append(skipped, tagName+" (lookup failed)")
				continue
			}

			// Link to video (ignore duplicate)
			_, err = s.db.Exec(`INSERT OR IGNORE INTO video_tags (video_id, tag_id) VALUES (?, ?)`, videoID, tagID)
			if err != nil {
				skipped = append(skipped, tagName+" (link failed)")
			} else {
				applied = append(applied, tagName)
			}
		}

		return map[string]interface{}{
			"video_id": videoID,
			"applied":  applied,
			"skipped":  skipped,
			"note":     fmt.Sprintf("Applied %d tag(s) to video %d", len(applied), videoID),
		}, nil

	case "set_video_rating":
		videoID := int64(floatArg(args, "video_id", 0))
		rating := intArg(args, "rating", -1)
		if videoID == 0 {
			return map[string]interface{}{"error": "video_id required"}, nil
		}
		if rating < 0 || rating > 5 {
			return map[string]interface{}{"error": "rating must be 0-5"}, nil
		}

		// Verify video exists and get current state
		var oldRating int
		var title string
		err := s.db.QueryRow(`SELECT COALESCE(rating, 0), title FROM videos WHERE id = ?`, videoID).Scan(&oldRating, &title)
		if err == sql.ErrNoRows {
			return map[string]interface{}{"error": fmt.Sprintf("video %d not found", videoID)}, nil
		} else if err != nil {
			return map[string]interface{}{"error": err.Error()}, nil
		}

		_, err = s.db.Exec(`UPDATE videos SET rating = ? WHERE id = ?`, rating, videoID)
		if err != nil {
			return map[string]interface{}{"error": "update failed: " + err.Error()}, nil
		}
		log.Printf("[AI] set_video_rating: video %d (%s) rated %d (was %d)", videoID, title, rating, oldRating)

		return map[string]interface{}{
			"video_id":   videoID,
			"title":      title,
			"old_rating": oldRating,
			"new_rating": rating,
		}, nil

	case "toggle_video_favorite":
		videoID := int64(floatArg(args, "video_id", 0))
		if videoID == 0 {
			return map[string]interface{}{"error": "video_id required"}, nil
		}

		var isFav int
		var title string
		err := s.db.QueryRow(`SELECT COALESCE(is_favorite, 0), title FROM videos WHERE id = ?`, videoID).Scan(&isFav, &title)
		if err == sql.ErrNoRows {
			return map[string]interface{}{"error": fmt.Sprintf("video %d not found", videoID)}, nil
		} else if err != nil {
			return map[string]interface{}{"error": err.Error()}, nil
		}

		newFav := 0
		if isFav == 0 {
			newFav = 1
		}
		_, err = s.db.Exec(`UPDATE videos SET is_favorite = ? WHERE id = ?`, newFav, videoID)
		if err != nil {
			return map[string]interface{}{"error": "update failed: " + err.Error()}, nil
		}

		action := "added to favorites"
		if newFav == 0 {
			action = "removed from favorites"
		}
		log.Printf("[AI] toggle_video_favorite: video %d (%s) %s", videoID, title, action)

		return map[string]interface{}{
			"video_id":    videoID,
			"title":       title,
			"is_favorite": newFav == 1,
			"action":      action,
		}, nil

	case "get_video_link_stats":
		var confirmed, suggested, rejected int
		s.db.QueryRow("SELECT COUNT(*) FROM video_links WHERE status = 'confirmed'").Scan(&confirmed)
		s.db.QueryRow("SELECT COUNT(*) FROM video_links WHERE status = 'suggested'").Scan(&suggested)
		s.db.QueryRow("SELECT COUNT(*) FROM video_links WHERE status = 'rejected'").Scan(&rejected)

		// Per-library breakdown: how many videos have a confirmed link vs total
		type libLinkStat struct {
			Name      string
			Total     int
			Linked    int
			Coverage  float64
		}
		libRows, err := s.db.Query(`
			SELECT l.name, COUNT(v.id) AS total,
			       SUM(CASE WHEN EXISTS (
			           SELECT 1 FROM video_links vl
			           WHERE (vl.edited_video_id = v.id OR vl.backup_video_id = v.id)
			             AND vl.status = 'confirmed'
			       ) THEN 1 ELSE 0 END) AS linked
			FROM libraries l
			LEFT JOIN videos v ON v.library_id = l.id
			GROUP BY l.id ORDER BY l.name`)
		libStats := []libLinkStat{}
		if err == nil {
			defer libRows.Close()
			for libRows.Next() {
				var ls libLinkStat
				if libRows.Scan(&ls.Name, &ls.Total, &ls.Linked) == nil {
					if ls.Total > 0 {
						ls.Coverage = float64(ls.Linked) / float64(ls.Total) * 100
					}
					libStats = append(libStats, ls)
				}
			}
		}

		result := map[string]interface{}{
			"confirmed_links":     confirmed,
			"suggested_links":     suggested,
			"rejected_links":      rejected,
			"total_links":         confirmed + suggested + rejected,
			"library_breakdown":   libStats,
		}
		return result, nil

	case "get_unlinked_videos":
		libraryName, _ := args["library_name"].(string)
		limit := intArg(args, "limit", 20)
		if limit > 100 {
			limit = 100
		}
		if libraryName == "" {
			return map[string]interface{}{"error": "library_name is required"}, nil
		}

		rows, err := s.db.Query(`
			SELECT v.id, v.title, v.duration, COALESCE(v.is_missing, 0)
			FROM videos v
			INNER JOIN libraries l ON l.id = v.library_id
			WHERE LOWER(l.name) LIKE LOWER(?)
			  AND NOT EXISTS (
			      SELECT 1 FROM video_links vl
			      WHERE (vl.edited_video_id = v.id OR vl.backup_video_id = v.id)
			        AND vl.status = 'confirmed'
			  )
			ORDER BY v.title ASC
			LIMIT ?`, "%"+libraryName+"%", limit)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}, nil
		}
		defer rows.Close()

		type unlinkedEntry struct {
			ID        int64   `json:"id"`
			Title     string  `json:"title"`
			Duration  float64 `json:"duration"`
			IsMissing bool    `json:"is_missing"`
		}
		var unlinked []unlinkedEntry
		for rows.Next() {
			var e unlinkedEntry
			if rows.Scan(&e.ID, &e.Title, &e.Duration, &e.IsMissing) == nil {
				unlinked = append(unlinked, e)
			}
		}
		if unlinked == nil {
			unlinked = []unlinkedEntry{}
		}
		return map[string]interface{}{
			"library":        libraryName,
			"unlinked_count": len(unlinked),
			"videos":         unlinked,
		}, nil

	default:
		return map[string]interface{}{"error": fmt.Sprintf("Unknown tool: %s", name)}, nil
	}
}

// fetchSiteInfo fetches a page and returns structural info for scraper config suggestions
func (s *AICompanionService) fetchSiteInfo(page1URL, page2URL string) (interface{}, error) {
	client := &http.Client{Timeout: 12 * time.Second}

	// Detect pagination if page2 provided
	paginationInfo := map[string]interface{}{
		"template":    "",
		"starts_at":   1,
		"confidence":  "unknown",
		"notes":       "No second page URL provided",
	}
	if page2URL != "" {
		// Inline pagination detection (mirrors the handler logic)
		detected := detectPaginationForAI(page1URL, page2URL)
		paginationInfo = detected
	}

	// Fetch page1 HTML
	req, err := http.NewRequest("GET", page1URL, nil)
	if err != nil {
		return map[string]interface{}{
			"error":          fmt.Sprintf("Invalid URL: %v", err),
			"pagination":     paginationInfo,
		}, nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	resp, err := client.Do(req)
	if err != nil {
		return map[string]interface{}{
			"error":          fmt.Sprintf("Failed to fetch page: %v", err),
			"pagination":     paginationInfo,
		}, nil
	}
	defer resp.Body.Close()

	// Read up to 80KB
	limitReader := io.LimitReader(resp.Body, 80*1024)
	bodyBytes, err := io.ReadAll(limitReader)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("Failed to read page: %v", err)}, nil
	}
	html := string(bodyBytes)

	// Extract page title
	title := ""
	titleRe := regexp.MustCompile(`(?i)<title[^>]*>([^<]+)</title>`)
	if m := titleRe.FindStringSubmatch(html); len(m) > 1 {
		title = strings.TrimSpace(m[1])
	}

	// Strip all tags for content summary
	stripTags := regexp.MustCompile(`<[^>]+>`)
	text := stripTags.ReplaceAllString(html, " ")
	// Collapse whitespace
	wsRe := regexp.MustCompile(`\s+`)
	text = strings.TrimSpace(wsRe.ReplaceAllString(text, " "))
	if len(text) > 2000 {
		text = text[:2000]
	}

	// Count structural elements
	linkCount := strings.Count(strings.ToLower(html), "<a ")
	imgCount := strings.Count(strings.ToLower(html), "<img ")
	videoCount := strings.Count(strings.ToLower(html), "<video") + strings.Count(strings.ToLower(html), "iframe")

	return map[string]interface{}{
		"url":               page1URL,
		"page_title":        title,
		"status_code":       resp.StatusCode,
		"link_count":        linkCount,
		"image_count":       imgCount,
		"video_embed_count": videoCount,
		"content_preview":   text,
		"pagination":        paginationInfo,
	}, nil
}

// detectPaginationForAI is a lightweight version of analyzePaginationURLs for use in the AI service
func detectPaginationForAI(url1, url2 string) map[string]interface{} {
	// Compare the two URLs to find a numeric difference
	// Simple approach: find the token that changed
	if len(url1) == 0 || len(url2) == 0 {
		return map[string]interface{}{"template": "", "confidence": "low", "notes": "Missing URLs"}
	}

	// Find common prefix
	prefix := ""
	for i := 0; i < len(url1) && i < len(url2); i++ {
		if url1[i] == url2[i] {
			prefix += string(url1[i])
		} else {
			break
		}
	}

	// Find common suffix (reversed)
	suffix := ""
	for i := 0; i < len(url1) && i < len(url2); i++ {
		if url1[len(url1)-1-i] == url2[len(url2)-1-i] {
			suffix = string(url1[len(url1)-1-i]) + suffix
		} else {
			break
		}
	}

	middle1 := url1[len(prefix) : len(url1)-len(suffix)]
	middle2 := url2[len(prefix) : len(url2)-len(suffix)]

	n1, e1 := strconv.Atoi(middle1)
	n2, e2 := strconv.Atoi(middle2)
	if e1 == nil && e2 == nil && n2 == n1+1 {
		template := prefix + "{page}" + suffix
		return map[string]interface{}{
			"template":   template,
			"starts_at":  n1,
			"confidence": "high",
			"notes":      fmt.Sprintf("Page number changes from %d to %d between the two URLs", n1, n2),
		}
	}

	return map[string]interface{}{
		"template":   "",
		"starts_at":  1,
		"confidence": "low",
		"notes":      "Could not auto-detect pagination pattern",
	}
}

// normalizeVideoTitle strips resolution/quality markers for duplicate-title comparison
var (
	reBrackets   = regexp.MustCompile(`[\(\[\{][^\)\]\}]*[\)\]\}]`)
	reWhitespace = regexp.MustCompile(`\s+`)
)

func normalizeVideoTitle(title string) string {
	s := strings.ToLower(title)
	s = reBrackets.ReplaceAllString(s, " ")
	s = strings.NewReplacer(
		"1080p", "", "720p", "", "480p", "", "360p", "",
		"2160p", "", "4k", "", "uhd", "", "full hd", "", " hd ", " ",
		"x264", "", "x265", "", "hevc", "", "avc", "",
		".mp4", "", ".mkv", "", ".avi", "", ".wmv", "",
	).Replace(s)
	s = strings.TrimSpace(reWhitespace.ReplaceAllString(s, " "))
	return s
}

// intArg extracts an int from args with a default value
func intArg(args map[string]interface{}, key string, defaultVal int) int {
	if v, ok := args[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		case int64:
			return int(n)
		}
	}
	return defaultVal
}

// floatArg extracts a float64 from args with a default value
func floatArg(args map[string]interface{}, key string, defaultVal float64) float64 {
	if v, ok := args[key]; ok {
		switch n := v.(type) {
		case float64:
			return n
		case int:
			return float64(n)
		case int64:
			return float64(n)
		}
	}
	return defaultVal
}

// formatSeconds converts a duration in seconds to "H:MM:SS" or "M:SS"
func formatSeconds(secs float64) string {
	total := int(secs)
	h := total / 3600
	m := (total % 3600) / 60
	s := total % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

// topNFromMap returns the top N keys by value from a map
func topNFromMap(m map[string]int, n int) []string {
	type kv struct {
		key string
		val int
	}
	var pairs []kv
	for k, v := range m {
		pairs = append(pairs, kv{k, v})
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].val > pairs[j].val })
	out := make([]string, 0, n)
	for i, p := range pairs {
		if i >= n {
			break
		}
		out = append(out, p.key)
	}
	return out
}

// ChatLLM processes user messages using LM Studio with tool calling, falling back to rule-based
func (s *AICompanionService) ChatLLM(message string, history []map[string]string) (string, error) {
	config := s.GetAIConfig()
	if !config.Enabled {
		log.Printf("[AI] LLM is disabled in config")
		return "", fmt.Errorf("LLM disabled in config")
	}

	log.Printf("[AI] ChatLLM called - URL: %s, Model: %s, Message: %.80s...", config.LMStudioURL, config.ModelName, message)
	startTime := time.Now()

	// Build system prompt with full app context
	appContext := s.GetFullContext()
	systemPrompt := `You are the AI Companion for a Video Storage application. You have direct access to the app's database and systems through tool calls.

You are helpful, concise, and knowledgeable about the user's video library, scrapers, JDownloader integration, and system health.

When the user asks about data (videos, performers, stats, etc.), use the appropriate tool to get real-time data rather than guessing.
When the user asks about app status (running tasks, errors, scrapers, JDownloader), use the relevant status tools.
You can save and recall memories to remember user preferences.
You can help organize videos: use scan_library_structure to understand how Backup is organized, suggest_video_placement to find the best matching subfolder for a video, and move_video_to_library to actually move it. Always confirm with the user before moving files.
You can find duplicate files: use find_duplicate_videos to detect videos stored multiple times across libraries (same filename, same title, or same performers with similar duration).
You can find orphaned DB entries: use find_orphaned_videos to check which DB records have no file on disk, then delete_video_records to clean them up (DB only, never touches disk). ALWAYS confirm with the user before calling delete_video_records.
You can profile a performer: use get_performer_stats to get total videos, watch time, rating breakdown, top tags/studios, library distribution, and recent additions for any performer.
You can interact with the library: suggest_tags_for_video (recommends tags based on performer history), apply_tags_to_video (add tags by name, creates new ones if needed), set_video_rating (rate 0-5 stars), toggle_video_favorite (mark/unmark as favorite). For Edited↔Backup pairing: get_video_link_stats (coverage overview), get_unlinked_videos (find unmatched videos in a library).
You can generate a library health report: get_library_health returns a health score (0-100), grade (A-D), untagged %, unrated %, videos with no performers, and a per-library breakdown. Use when the user asks 'how's my library?' or 'library overview'.
For bulk tagging: call find_untagged_videos to get a list of IDs, then call bulk_apply_tags(video_ids) to auto-tag all of them in one shot using performer history. Tell the user how many videos will be affected and confirm before calling bulk_apply_tags. Use suggest_tags_for_video + apply_tags_to_video only for individual videos.

Keep responses concise and formatted with markdown. Use bullet points and bold text for readability.

` + appContext

	// Build messages
	messages := []map[string]interface{}{
		{"role": "system", "content": systemPrompt},
	}

	// Add conversation history
	for _, h := range history {
		if h["role"] != "" && h["content"] != "" {
			messages = append(messages, map[string]interface{}{"role": h["role"], "content": h["content"]})
		}
	}

	// Add current user message
	messages = append(messages, map[string]interface{}{"role": "user", "content": message})

	tools := getLMToolDefinitions()

	// Tool calling loop (max 5 iterations)
	for iteration := 0; iteration < 5; iteration++ {
		reqBody := map[string]interface{}{
			"messages":    messages,
			"tools":       tools,
			"temperature": config.Temperature,
			"max_tokens":  config.MaxTokens,
			"stream":      false,
		}
		if config.ModelName != "" {
			reqBody["model"] = config.ModelName
		}

		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			return "", fmt.Errorf("failed to marshal request: %w", err)
		}

		client := &http.Client{Timeout: 120 * time.Second}
		resp, err := client.Post(config.LMStudioURL+"/v1/chat/completions", "application/json", bytes.NewReader(bodyBytes))
		if err != nil {
			log.Printf("[AI] LM Studio connection failed after %v: %v", time.Since(startTime), err)
			return "", fmt.Errorf("LM Studio connection failed: %w", err)
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to read LM Studio response: %w", err)
		}

		if resp.StatusCode != 200 {
			return "", fmt.Errorf("LM Studio returned status %d: %s", resp.StatusCode, string(respBody))
		}

		var result struct {
			Choices []struct {
				Message struct {
					Role      string `json:"role"`
					Content   string `json:"content"`
					ToolCalls []struct {
						ID       string `json:"id"`
						Type     string `json:"type"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
		}

		if err := json.Unmarshal(respBody, &result); err != nil {
			return "", fmt.Errorf("failed to parse LM Studio response: %w", err)
		}

		if len(result.Choices) == 0 {
			return "", fmt.Errorf("no choices in LM Studio response")
		}

		choice := result.Choices[0]

		// If no tool calls, return the content
		if len(choice.Message.ToolCalls) == 0 || choice.FinishReason == "stop" {
			content := choice.Message.Content
			if content == "" {
				content = "I processed your request but have no additional information to share."
			}
			log.Printf("[AI] LLM response received in %v (iteration %d, %d chars)", time.Since(startTime), iteration+1, len(content))
			return content, nil
		}

		// Process tool calls
		// Add assistant message with tool calls to messages
		messages = append(messages, map[string]interface{}{
			"role":       "assistant",
			"content":    choice.Message.Content,
			"tool_calls": choice.Message.ToolCalls,
		})

		for _, tc := range choice.Message.ToolCalls {
			var toolArgs map[string]interface{}
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &toolArgs); err != nil {
				toolArgs = map[string]interface{}{}
			}

			log.Printf("[AI] Tool call: %s(%v)", tc.Function.Name, toolArgs)

			toolResult, err := s.ExecuteTool(tc.Function.Name, toolArgs)
			if err != nil {
				toolResult = map[string]interface{}{"error": err.Error()}
			}

			resultJSON, _ := json.Marshal(toolResult)

			messages = append(messages, map[string]interface{}{
				"role":         "tool",
				"tool_call_id": tc.ID,
				"content":      string(resultJSON),
			})
		}
	}

	return "I ran out of tool calling iterations. Please try a simpler question.", nil
}

// extractVideoCards pulls a normalised video card list from a tool result.
// Returns nil when the tool doesn't produce a video list.
// Capped at 10 cards to keep the UI readable.
func extractVideoCards(toolName string, result interface{}) []map[string]interface{} {
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return nil
	}
	listKey := ""
	switch toolName {
	case "search_videos", "get_library_videos":
		listKey = "videos"
	case "suggest_videos_to_watch":
		listKey = "suggestions"
	case "get_favorites":
		listKey = "favorites"
	case "get_top_rated":
		listKey = "top_rated"
	case "find_untagged_videos":
		listKey = "untagged_videos"
	default:
		return nil
	}
	rawList, ok := resultMap[listKey]
	if !ok {
		return nil
	}
	list, ok := rawList.([]map[string]interface{})
	if !ok {
		return nil
	}
	if len(list) > 10 {
		list = list[:10]
	}
	return list
}

// ChatLLMStream processes user messages using LM Studio with tool calling and streams the final
// text response chunk-by-chunk via onChunk. onCards is called once (if non-nil) when the
// response involves video results — the frontend uses these to render inline video cards.
// Returns false from onChunk to abort (client gone).
func (s *AICompanionService) ChatLLMStream(message string, history []map[string]string, onChunk func(string) bool, onCards func([]map[string]interface{})) error {
	config := s.GetAIConfig()
	if !config.Enabled {
		return fmt.Errorf("LLM disabled in config")
	}

	appContext := s.GetFullContext()
	systemPrompt := `You are the AI Companion for a Video Storage application. You have direct access to the app's database and systems through tool calls.

You are helpful, concise, and knowledgeable about the user's video library, scrapers, JDownloader integration, and system health.

When the user asks about data (videos, performers, stats, etc.), use the appropriate tool to get real-time data rather than guessing.
When the user asks about app status (running tasks, errors, scrapers, JDownloader), use the relevant status tools.
You can save and recall memories to remember user preferences.
You can help organize videos: use scan_library_structure to understand how Backup is organized, suggest_video_placement to find the best matching subfolder for a video, and move_video_to_library to actually move it. Always confirm with the user before moving files.
You can find duplicate files: use find_duplicate_videos to detect videos stored multiple times across libraries (same filename, same title, or same performers with similar duration).
You can find orphaned DB entries: use find_orphaned_videos to check which DB records have no file on disk, then delete_video_records to clean them up (DB only, never touches disk). ALWAYS confirm with the user before calling delete_video_records.
You can profile a performer: use get_performer_stats to get total videos, watch time, rating breakdown, top tags/studios, library distribution, and recent additions for any performer.
You can interact with the library: suggest_tags_for_video (recommends tags based on performer history), apply_tags_to_video (add tags by name, creates new ones if needed), set_video_rating (rate 0-5 stars), toggle_video_favorite (mark/unmark as favorite). For Edited↔Backup pairing: get_video_link_stats (coverage overview), get_unlinked_videos (find unmatched videos in a library).
You can generate a library health report: get_library_health returns a health score (0-100), grade (A-D), untagged %, unrated %, videos with no performers, and a per-library breakdown. Use when the user asks 'how's my library?' or 'library overview'.
For bulk tagging: call find_untagged_videos to get a list of IDs, then call bulk_apply_tags(video_ids) to auto-tag all of them in one shot using performer history. Tell the user how many videos will be affected and confirm before calling bulk_apply_tags. Use suggest_tags_for_video + apply_tags_to_video only for individual videos.

Keep responses concise and formatted with markdown. Use bullet points and bold text for readability.

` + appContext

	messages := []map[string]interface{}{{"role": "system", "content": systemPrompt}}
	for _, h := range history {
		if h["role"] != "" && h["content"] != "" {
			messages = append(messages, map[string]interface{}{"role": h["role"], "content": h["content"]})
		}
	}
	messages = append(messages, map[string]interface{}{"role": "user", "content": message})

	tools := getLMToolDefinitions()
	client := &http.Client{Timeout: 120 * time.Second}

	// Accumulate video card data across all tool iterations
	var collectedCards []map[string]interface{}

	type accToolCall struct {
		id       string
		name     string
		argsJSON strings.Builder
	}

	for iteration := 0; iteration < 5; iteration++ {
		reqBody := map[string]interface{}{
			"messages":    messages,
			"tools":       tools,
			"temperature": config.Temperature,
			"max_tokens":  config.MaxTokens,
			"stream":      true,
		}
		if config.ModelName != "" {
			reqBody["model"] = config.ModelName
		}

		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}

		resp, err := client.Post(config.LMStudioURL+"/v1/chat/completions", "application/json", bytes.NewReader(bodyBytes))
		if err != nil {
			return fmt.Errorf("LM Studio connection failed: %w", err)
		}

		var accContent strings.Builder
		var accCalls []accToolCall
		hasToolCalls := false

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := line[6:]
			if data == "[DONE]" {
				break
			}

			var chunk struct {
				Choices []struct {
					Delta struct {
						Content   string `json:"content"`
						ToolCalls []struct {
							Index    int    `json:"index"`
							ID       string `json:"id"`
							Function struct {
								Name      string `json:"name"`
								Arguments string `json:"arguments"`
							} `json:"function"`
						} `json:"tool_calls"`
					} `json:"delta"`
				} `json:"choices"`
			}
			if err := json.Unmarshal([]byte(data), &chunk); err != nil || len(chunk.Choices) == 0 {
				continue
			}

			delta := chunk.Choices[0].Delta

			// Accumulate tool call fragments
			for _, tc := range delta.ToolCalls {
				hasToolCalls = true
				for len(accCalls) <= tc.Index {
					accCalls = append(accCalls, accToolCall{})
				}
				if tc.ID != "" {
					accCalls[tc.Index].id = tc.ID
				}
				if tc.Function.Name != "" {
					accCalls[tc.Index].name = tc.Function.Name
				}
				accCalls[tc.Index].argsJSON.WriteString(tc.Function.Arguments)
			}

			// Stream text — safe to emit immediately because tool calls and content
			// are mutually exclusive in a single LLM response
			if delta.Content != "" {
				accContent.WriteString(delta.Content)
				if !hasToolCalls {
					if !onChunk(delta.Content) {
						resp.Body.Close()
						return nil // client disconnected
					}
				}
			}
		}
		resp.Body.Close()

		if !hasToolCalls {
			// Final answer already streamed; ensure non-empty
			if accContent.Len() == 0 {
				onChunk("I processed your request but have no additional information to share.")
			}
			if onCards != nil && len(collectedCards) > 0 {
				onCards(collectedCards)
			}
			return nil
		}

		// Build assistant message with tool calls for next iteration
		toolCallsMsg := make([]map[string]interface{}, len(accCalls))
		for i, tc := range accCalls {
			toolCallsMsg[i] = map[string]interface{}{
				"id": tc.id, "type": "function",
				"function": map[string]interface{}{"name": tc.name, "arguments": tc.argsJSON.String()},
			}
		}
		messages = append(messages, map[string]interface{}{
			"role": "assistant", "content": accContent.String(), "tool_calls": toolCallsMsg,
		})

		for i, tc := range accCalls {
			var toolArgs map[string]interface{}
			if err := json.Unmarshal([]byte(tc.argsJSON.String()), &toolArgs); err != nil {
				toolArgs = map[string]interface{}{}
			}
			log.Printf("[AI] Stream tool call: %s(%v)", tc.name, toolArgs)
			toolResult, err := s.ExecuteTool(tc.name, toolArgs)
			if err != nil {
				toolResult = map[string]interface{}{"error": err.Error()}
			}
			// Collect video cards for inline rendering
			if cards := extractVideoCards(tc.name, toolResult); len(cards) > 0 {
				collectedCards = append(collectedCards, cards...)
			}
			resultJSON, _ := json.Marshal(toolResult)
			tcID := tc.id
			if tcID == "" {
				tcID = fmt.Sprintf("call_%d", i)
			}
			messages = append(messages, map[string]interface{}{
				"role": "tool", "tool_call_id": tcID, "content": string(resultJSON),
			})
		}
	}

	onChunk("I ran out of tool calling iterations. Please try a simpler question.")
	return nil
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

// analyzeNewFile generates proactive placement suggestions when a video lands in a Downloads library
func (s *AICompanionService) analyzeNewFile(filePath, libraryName string) {
	// Wait for file to stabilize
	time.Sleep(5 * time.Second)

	filename := filepath.Base(filePath)

	// Only generate placement suggestions for download-type libraries
	if !strings.Contains(strings.ToLower(libraryName), "download") {
		s.emitEvent(CompanionEvent{
			Type:      "insight",
			Source:    "analysis_engine",
			Message:   fmt.Sprintf("New file in **%s**: `%s`", libraryName, filename),
			Severity:  "info",
			Timestamp: time.Now(),
		})
		return
	}

	// Find Backup library
	var tgtLibID int64
	var tgtLibPath, tgtLibName string
	if err := s.db.QueryRow("SELECT id, name, path FROM libraries WHERE LOWER(name) LIKE ?", "%backup%").Scan(&tgtLibID, &tgtLibName, &tgtLibPath); err != nil {
		s.emitEvent(CompanionEvent{
			Type:      "insight",
			Source:    "analysis_engine",
			Message:   fmt.Sprintf("📥 **New download**: `%s`\n\nNo Backup library configured.", filename),
			Severity:  "info",
			Timestamp: time.Now(),
		})
		return
	}

	// Poll DB for up to 30s — the library scanner may not have run yet
	var videoID int64
	var performers, tags, studios []string
	for i := 0; i < 6; i++ {
		if i > 0 {
			time.Sleep(5 * time.Second)
		}
		var vid int64
		if s.db.QueryRow("SELECT id FROM videos WHERE file_path = ?", filePath).Scan(&vid) == nil && vid > 0 {
			videoID = vid
			perfRows, _ := s.db.Query("SELECT p.name FROM performers p JOIN video_performers vp ON p.id = vp.performer_id WHERE vp.video_id = ?", videoID)
			for perfRows.Next() {
				var n string
				perfRows.Scan(&n)
				performers = append(performers, n)
			}
			perfRows.Close()
			tagRows, _ := s.db.Query("SELECT t.name FROM tags t JOIN video_tags vt ON t.id = vt.tag_id WHERE vt.video_id = ?", videoID)
			for tagRows.Next() {
				var n string
				tagRows.Scan(&n)
				tags = append(tags, n)
			}
			tagRows.Close()
			studioRows, _ := s.db.Query("SELECT st.name FROM studios st JOIN video_studios vs ON st.id = vs.studio_id WHERE vs.video_id = ?", videoID)
			for studioRows.Next() {
				var n string
				studioRows.Scan(&n)
				studios = append(studios, n)
			}
			studioRows.Close()
			break
		}
	}

	// Check learned placement memories for known performers
	learnedFolders := map[string]string{} // subfolder → reason
	for _, p := range performers {
		if mems, err := s.SearchMemories("placement " + p); err == nil {
			for _, mem := range mems {
				if strings.Contains(mem.Value, "→") {
					parts := strings.SplitN(mem.Value, "→", 2)
					if len(parts) == 2 {
						subfolder := strings.TrimSpace(strings.Split(parts[1], " in ")[0])
						learnedFolders[subfolder] = "✅ learned: " + p
					}
				}
			}
		}
	}

	// Score Backup folders against metadata + filename
	type candidate struct {
		path    string
		score   float64
		reason  string
		learned bool
	}
	var candidates []candidate

	// Build a map of performer → existing folder in Backup (content-based, highest confidence)
	// This finds folders that ALREADY contain videos by the same performer
	performerFolders := map[string]string{} // performer name → subfolder
	for _, p := range performers {
		var existingPath string
		err := s.db.QueryRow(
			`SELECT v.file_path FROM videos v
			 JOIN video_performers vp ON v.id = vp.video_id
			 JOIN performers perf ON perf.id = vp.performer_id
			 WHERE v.library_id = ? AND LOWER(perf.name) = LOWER(?)
			 LIMIT 1`, tgtLibID, p).Scan(&existingPath)
		if err == nil && existingPath != "" {
			rel, err2 := filepath.Rel(tgtLibPath, filepath.Dir(existingPath))
			if err2 == nil {
				performerFolders[p] = filepath.ToSlash(rel)
			}
		}
	}

	_ = filepath.Walk(tgtLibPath, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil || !info.IsDir() || path == tgtLibPath {
			return nil
		}
		relPath, _ := filepath.Rel(tgtLibPath, path)
		if len(strings.Split(filepath.ToSlash(relPath), "/")) > 3 {
			return filepath.SkipDir
		}
		folderSlash := filepath.ToSlash(relPath)
		folderLower := strings.ToLower(folderSlash)
		sc := 0.0
		var reasons []string
		isLearned := false

		if reason, ok := learnedFolders[folderSlash]; ok {
			sc += 20
			reasons = append(reasons, reason)
			isLearned = true
		}
		// Tier 1: folder already contains videos by this performer (content-based)
		for _, p := range performers {
			if existing, ok := performerFolders[p]; ok && strings.EqualFold(existing, folderSlash) {
				sc += 15
				reasons = append(reasons, "existing videos by "+p)
			}
		}
		// Tier 2: exact full performer name in folder path
		for _, p := range performers {
			folderBase := strings.ToLower(filepath.Base(filepath.FromSlash(folderSlash)))
			if strings.EqualFold(folderBase, p) {
				sc += 10
				reasons = append(reasons, "performer folder: "+p)
			} else if strings.Contains(folderLower, strings.ToLower(p)) && len(strings.Fields(p)) > 1 {
				// Only match multi-word full names (e.g. "Nicole Aniston") in path, not single words
				sc += 8
				reasons = append(reasons, "performer in path: "+p)
			}
		}
		for _, st := range studios {
			if strings.Contains(folderLower, strings.ToLower(st)) {
				sc += 7
				reasons = append(reasons, "studio: "+st)
			}
		}
		for _, t := range tags {
			if strings.Contains(folderLower, strings.ToLower(t)) {
				sc += 4
				reasons = append(reasons, "tag: "+t)
			}
		}
		// No filename word matching — that leads to false positives

		if sc > 0 {
			candidates = append(candidates, candidate{path: folderSlash, score: sc, reason: strings.Join(reasons, "; "), learned: isLearned})
		}
		return nil
	})

	sort.Slice(candidates, func(i, j int) bool { return candidates[i].score > candidates[j].score })
	if len(candidates) > 3 {
		candidates = candidates[:3]
	}

	// Format message
	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("📥 **New download detected**: `%s`\n\n", filename))
	if len(performers) > 0 {
		msg.WriteString(fmt.Sprintf("**Performers**: %s\n\n", strings.Join(performers, ", ")))
	}
	if len(candidates) > 0 {
		msg.WriteString(fmt.Sprintf("**Suggested placements in %s:**\n", tgtLibName))
		for i, c := range candidates {
			msg.WriteString(fmt.Sprintf("%d. `%s` — %s\n", i+1, c.path, c.reason))
		}
		if videoID != 0 {
			msg.WriteString(fmt.Sprintf("\n*Video ID: **%d*** — tell me which number to move it to.", videoID))
		} else {
			msg.WriteString("\n*Not indexed yet* — scan the Downloads library first, then ask me to move it.")
		}
	} else {
		msg.WriteString(fmt.Sprintf("No matching folders found in **%s**.\n", tgtLibName))
		msg.WriteString("Ask me to scan its structure to see what's available.")
	}

	eventData := map[string]interface{}{"file_path": filePath, "library": libraryName}
	if videoID != 0 {
		eventData["video_id"] = videoID
	}
	s.emitEvent(CompanionEvent{
		Type:      "suggestion",
		Source:    "analysis_engine",
		Message:   msg.String(),
		Data:      eventData,
		Severity:  "info",
		Timestamp: time.Now(),
	})
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

// generateDailyBriefing emits a proactive morning summary once per ~20 hours.
// It also saves a video-count snapshot so it can report deltas the next day.
func (s *AICompanionService) generateDailyBriefing() {
	// Check when we last sent a briefing
	var lastBriefingStr string
	s.db.QueryRow("SELECT value FROM memories WHERE key = 'last_daily_briefing'").Scan(&lastBriefingStr)
	if lastBriefingStr != "" {
		if t, err := time.Parse(time.RFC3339, lastBriefingStr); err == nil {
			if time.Since(t) < 20*time.Hour {
				log.Printf("[AI] Daily briefing skipped — last sent %s ago", formatDuration(time.Since(t)))
				return
			}
		}
	}

	// Current stats
	var videoCount, performerCount int
	s.db.QueryRow("SELECT COUNT(*) FROM videos").Scan(&videoCount)
	s.db.QueryRow("SELECT COUNT(*) FROM performers").Scan(&performerCount)

	// Delta since last snapshot
	var snapshotJSON string
	s.db.QueryRow("SELECT value FROM memories WHERE key = 'daily_snapshot'").Scan(&snapshotJSON)
	prevVideoCount := videoCount
	if snapshotJSON != "" {
		var snap map[string]int
		if json.Unmarshal([]byte(snapshotJSON), &snap) == nil {
			prevVideoCount = snap["video_count"]
		}
	}
	newVideos := videoCount - prevVideoCount

	// Videos sitting in Downloads library
	var downloadsCount int
	s.db.QueryRow(`SELECT COUNT(*) FROM videos v JOIN libraries l ON v.library_id = l.id WHERE LOWER(l.name) LIKE '%download%'`).Scan(&downloadsCount)

	// Untagged videos
	var untaggedCount int
	s.db.QueryRow(`SELECT COUNT(*) FROM videos WHERE id NOT IN (SELECT DISTINCT video_id FROM video_tags)`).Scan(&untaggedCount)

	// Recent errors (last 24h)
	var errorCount int
	s.db.QueryRow(`SELECT COUNT(*) FROM console_logs WHERE level = 'error' AND created_at > datetime('now', '-24 hours')`).Scan(&errorCount)

	// Currently running tasks
	var runningTasks int
	s.db.QueryRow(`SELECT COUNT(*) FROM activity_logs WHERE status = 'running'`).Scan(&runningTasks)

	// Quick orphan sample — check first 200 videos for missing files (fast, SSD-friendly)
	orphanCount := 0
	sampleRows, _ := s.db.Query(`SELECT file_path FROM videos ORDER BY created_at DESC LIMIT 200`)
	if sampleRows != nil {
		for sampleRows.Next() {
			var fp string
			sampleRows.Scan(&fp)
			if _, statErr := os.Stat(fp); os.IsNotExist(statErr) {
				orphanCount++
			}
		}
		sampleRows.Close()
	}

	// Build the briefing message
	var msg strings.Builder
	msg.WriteString("## 🌅 Daily Briefing\n\n")

	// Library headline
	if newVideos > 0 {
		msg.WriteString(fmt.Sprintf("📈 **%d new video(s)** added since last briefing — library now at **%d total**\n\n", newVideos, videoCount))
	} else if newVideos < 0 {
		msg.WriteString(fmt.Sprintf("📚 Library: **%d videos** across **%d performers**\n\n", videoCount, performerCount))
	} else {
		msg.WriteString(fmt.Sprintf("📚 Library unchanged: **%d videos** across **%d performers**\n\n", videoCount, performerCount))
	}

	// Downloads waiting
	if downloadsCount > 0 {
		msg.WriteString(fmt.Sprintf("📥 **%d video(s)** in Downloads waiting to be organized into Backup\n\n", downloadsCount))
	}

	// Untagged
	if untaggedCount > 0 {
		msg.WriteString(fmt.Sprintf("🏷️ **%d video(s)** have no tags — ask me to help tag them\n\n", untaggedCount))
	}

	// Errors
	if errorCount > 0 {
		msg.WriteString(fmt.Sprintf("⚠️ **%d error(s)** logged in the last 24 hours — ask me to check for errors\n\n", errorCount))
	} else {
		msg.WriteString("✅ No errors logged in the last 24 hours\n\n")
	}

	// Running tasks
	if runningTasks > 0 {
		msg.WriteString(fmt.Sprintf("⚙️ **%d task(s)** currently running in the background\n\n", runningTasks))
	}

	// Orphan warning (sampled check)
	if orphanCount > 0 {
		msg.WriteString(fmt.Sprintf("🔍 **%d video(s)** in the last 200 sampled appear to be missing from disk — ask me to run a full orphan scan\n\n", orphanCount))
	}

	msg.WriteString("_Ask me anything, or say **\"help me organize my downloads\"** to get started!_")

	// Persist snapshot + timestamp
	newSnap, _ := json.Marshal(map[string]int{"video_count": videoCount})
	s.db.Exec(`INSERT OR REPLACE INTO memories (key, value, category, importance, created_at, updated_at)
		VALUES ('daily_snapshot', ?, 'system', 1, datetime('now'), datetime('now'))`, string(newSnap))
	s.db.Exec(`INSERT OR REPLACE INTO memories (key, value, category, importance, created_at, updated_at)
		VALUES ('last_daily_briefing', ?, 'system', 1, datetime('now'), datetime('now'))`, time.Now().Format(time.RFC3339))

	log.Printf("[AI] Emitting daily briefing (newVideos=%d, downloads=%d, errors=%d, orphanSample=%d)", newVideos, downloadsCount, errorCount, orphanCount)

	s.emitEvent(CompanionEvent{
		Type:      "insight",
		Source:    "daily_briefing",
		Message:   msg.String(),
		Severity:  "info",
		Timestamp: time.Now(),
	})
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
			log.Println("🔍 AI Companion: Running periodic library analysis...")
			s.runLibraryInsights()
		}
	}
}

// runLibraryInsights checks for actionable library conditions and emits proactive insights.
// Each insight type has its own cooldown key in the memories table to avoid spam.
func (s *AICompanionService) runLibraryInsights() {
	s.checkInsight(
		"insight_unrated_recent",
		24*time.Hour,
		func() (bool, string, map[string]interface{}) {
			var count int
			s.db.QueryRow(`SELECT COUNT(*) FROM videos WHERE (rating IS NULL OR rating = 0) AND created_at > datetime('now', '-14 days')`).Scan(&count)
			if count < 6 {
				return false, "", nil
			}
			msg := fmt.Sprintf("⭐ **%d videos** added in the last 2 weeks haven't been rated yet. Rating helps me give you better recommendations — ask me to help rate them!", count)
			return true, msg, map[string]interface{}{"unrated_recent": count, "action": "rate_videos"}
		},
	)

	s.checkInsight(
		"insight_untagged_videos",
		48*time.Hour,
		func() (bool, string, map[string]interface{}) {
			var untagged, total int
			s.db.QueryRow(`SELECT COUNT(*) FROM videos v WHERE NOT EXISTS (SELECT 1 FROM video_tags vt WHERE vt.video_id = v.id)`).Scan(&untagged)
			s.db.QueryRow(`SELECT COUNT(*) FROM videos`).Scan(&total)
			if untagged < 15 || total == 0 {
				return false, "", nil
			}
			pct := untagged * 100 / total
			msg := fmt.Sprintf("🏷️ **%d videos (%d%%)** in your library have no tags. Ask me to suggest tags — I can analyse performers to recommend the best ones!", untagged, pct)
			return true, msg, map[string]interface{}{"untagged": untagged, "total": total, "action": "suggest_tags"}
		},
	)

	s.checkInsight(
		"insight_unorganized_downloads",
		12*time.Hour,
		func() (bool, string, map[string]interface{}) {
			var count int
			s.db.QueryRow(`
				SELECT COUNT(*) FROM videos v
				JOIN libraries l ON l.id = v.library_id
				WHERE LOWER(l.name) LIKE '%download%'
				  AND NOT EXISTS (SELECT 1 FROM video_performers vp WHERE vp.video_id = v.id)
			`).Scan(&count)
			if count < 8 {
				return false, "", nil
			}
			msg := fmt.Sprintf("📥 **%d unlinked videos** in your Downloads library have no performers attached. These are likely freshly downloaded files that haven't been processed yet. Want me to help organise them?", count)
			return true, msg, map[string]interface{}{"unlinked_downloads": count, "action": "organize_downloads"}
		},
	)

	s.checkInsight(
		"insight_rating_coverage",
		72*time.Hour,
		func() (bool, string, map[string]interface{}) {
			var rated, total int
			s.db.QueryRow(`SELECT COUNT(*) FROM videos WHERE rating IS NOT NULL AND rating > 0`).Scan(&rated)
			s.db.QueryRow(`SELECT COUNT(*) FROM videos`).Scan(&total)
			if total < 20 {
				return false, "", nil
			}
			pct := rated * 100 / total
			if pct >= 30 {
				return false, "", nil
			}
			msg := fmt.Sprintf("📊 Only **%d%% of your library** (%d / %d videos) has been rated. The more you rate, the smarter my recommendations get. Ask me to pull up your unrated videos!", pct, rated, total)
			return true, msg, map[string]interface{}{"rated": rated, "total": total, "rating_pct": pct, "action": "rate_videos"}
		},
	)
}

// checkInsight runs a single insight check with cooldown guard.
// fn returns (shouldFire, message, data). If shouldFire is false nothing is emitted.
func (s *AICompanionService) checkInsight(cooldownKey string, cooldown time.Duration, fn func() (bool, string, map[string]interface{})) {
	var lastStr string
	s.db.QueryRow("SELECT value FROM memories WHERE key = ?", cooldownKey).Scan(&lastStr)
	if lastStr != "" {
		if t, err := time.Parse(time.RFC3339, lastStr); err == nil {
			if time.Since(t) < cooldown {
				return
			}
		}
	}

	shouldFire, msg, data := fn()
	if !shouldFire {
		return
	}

	s.db.Exec(`INSERT OR REPLACE INTO memories (key, value, category, importance, created_at, updated_at)
		VALUES (?, ?, 'system', 1, datetime('now'), datetime('now'))`, cooldownKey, time.Now().Format(time.RFC3339))

	s.emitEvent(CompanionEvent{
		Type:      "insight",
		Source:    "library_insights",
		Message:   msg,
		Data:      data,
		Severity:  "info",
		Timestamp: time.Now(),
	})
	log.Printf("[AI] Library insight fired: %s", cooldownKey)
}

// monitorActivityLogs monitors activity_logs table for task completions
func (s *AICompanionService) monitorActivityLogs() {
	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
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
	allErrors, _, err := consoleLogSvc.GetAll(50, 0, "", "error", "")
	if err != nil {
		return "", fmt.Errorf("failed to retrieve error logs: %w", err)
	}

	// Filter out noisy/benign errors
	errors := make([]*models.ConsoleLog, 0)
	for _, e := range allErrors {
		if !isNoisyError(e.Message) {
			errors = append(errors, e)
		}
	}

	if len(errors) == 0 {
		return "✅ No significant errors detected! Your system is running smoothly.", nil
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
	response.WriteString(fmt.Sprintf("⚠️ Found %d significant errors (filtered %d noisy). Analysis:\n\n", len(errors), len(allErrors)-len(errors)))

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

// isNoisyError returns true if the error message is benign/expected and should not trigger alerts
func isNoisyError(message string) bool {
	noisyPatterns := []string{
		"Error closing WebSocket connection",
		"use of closed network connection",
		"broken pipe",
		"connection reset by peer",
		"websocket: close",
		"context canceled",
		"http: Server closed",
		"[AI] Tool call",
		"[AI] LLM",
		"[AI] ChatLLM",
		"Network Error",
		"Failed to load dashboard",
		"Failed to load running tasks",
	}
	lowerMsg := strings.ToLower(message)
	for _, pattern := range noisyPatterns {
		if strings.Contains(lowerMsg, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
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

			// Filter out noisy/benign errors
			significantErrors := make([]*models.ConsoleLog, 0)
			for _, e := range errors {
				if !isNoisyError(e.Message) {
					significantErrors = append(significantErrors, e)
				}
			}

			// Alert if new significant errors appeared
			if len(significantErrors) > lastErrorCount && len(significantErrors) > 0 {
				newErrors := len(significantErrors) - lastErrorCount
				recentError := significantErrors[0]

				s.emitEvent(CompanionEvent{
					Type:    "notification",
					Source:  "error_monitor",
					Message: fmt.Sprintf("🚨 %d new error(s) detected! Latest: %s", newErrors, recentError.Message),
					Data: map[string]interface{}{
						"new_errors":    newErrors,
						"total_errors":  len(significantErrors),
						"latest_source": recentError.Source,
						"latest_level":  recentError.Level,
					},
					Severity:  "error",
					Timestamp: time.Now(),
				})
			}

			lastErrorCount = len(significantErrors)
		}
	}
}

// ================== Chat Helper Methods ==================

// generateAppUpdate creates a comprehensive status update of the app
func (s *AICompanionService) generateAppUpdate() (string, error) {
	var response strings.Builder
	response.WriteString("📊 **App Status Update**\n\n")

	// Library stats
	var videoCount, performerCount, tagCount int
	s.db.QueryRow("SELECT COUNT(*) FROM videos").Scan(&videoCount)
	s.db.QueryRow("SELECT COUNT(*) FROM performers").Scan(&performerCount)
	s.db.QueryRow("SELECT COUNT(*) FROM tags").Scan(&tagCount)
	response.WriteString(fmt.Sprintf("**Library:** %d videos, %d performers, %d tags\n\n", videoCount, performerCount, tagCount))

	// Running tasks
	rows, err := s.db.Query(`SELECT task_type, message, created_at FROM activity_logs WHERE status = 'running' ORDER BY created_at DESC LIMIT 5`)
	if err == nil {
		defer rows.Close()
		running := []string{}
		for rows.Next() {
			var taskType, message string
			var createdAt time.Time
			if rows.Scan(&taskType, &message, &createdAt) == nil {
				running = append(running, fmt.Sprintf("  - [%s] %s (%s ago)", taskType, message, formatDuration(time.Since(createdAt))))
			}
		}
		if len(running) > 0 {
			response.WriteString("**Running Tasks:**\n")
			for _, r := range running {
				response.WriteString(r + "\n")
			}
			response.WriteString("\n")
		} else {
			response.WriteString("**Running Tasks:** None - system is idle\n\n")
		}
	}

	// Recent activity
	recentRows, err := s.db.Query(`SELECT task_type, status, message, created_at FROM activity_logs WHERE status != 'running' ORDER BY created_at DESC LIMIT 5`)
	if err == nil {
		defer recentRows.Close()
		response.WriteString("**Recent Activity:**\n")
		hasActivity := false
		for recentRows.Next() {
			var taskType, status, message string
			var createdAt time.Time
			if recentRows.Scan(&taskType, &status, &message, &createdAt) == nil {
				hasActivity = true
				response.WriteString(fmt.Sprintf("  - [%s] %s - %s (%s ago)\n", taskType, status, message, formatDuration(time.Since(createdAt))))
			}
		}
		if !hasActivity {
			response.WriteString("  No recent activity.\n")
		}
		response.WriteString("\n")
	}

	// Scraper summary
	var scrapedTotal int
	s.db.QueryRow("SELECT COUNT(*) FROM scraped_videos").Scan(&scrapedTotal)
	if scrapedTotal > 0 {
		response.WriteString(fmt.Sprintf("**Scraped Content:** %d total items\n", scrapedTotal))
	}

	// JDownloader
	var jdStatus string
	err = s.db.QueryRow("SELECT connection_status FROM jdownloader_config LIMIT 1").Scan(&jdStatus)
	if err == nil {
		response.WriteString(fmt.Sprintf("**JDownloader:** %s\n", jdStatus))
	}

	// Errors
	var errorCount int
	s.db.QueryRow("SELECT COUNT(*) FROM console_logs WHERE level = 'error' AND created_at > datetime('now', '-1 hour')").Scan(&errorCount)
	if errorCount > 0 {
		response.WriteString(fmt.Sprintf("\n⚠️ **%d error(s)** in the last hour\n", errorCount))
	} else {
		response.WriteString("\n✅ No errors in the last hour\n")
	}

	return response.String(), nil
}

// looksLikePerformerName checks if a message looks like a performer name (2-4 capitalized words, no common keywords)
func (s *AICompanionService) looksLikePerformerName(message string) bool {
	words := strings.Fields(message)
	if len(words) < 1 || len(words) > 5 {
		return false
	}

	// Must have at least one word starting with uppercase
	hasUppercase := false
	for _, w := range words {
		if len(w) > 0 && w[0] >= 'A' && w[0] <= 'Z' {
			hasUppercase = true
		}
	}
	if !hasUppercase {
		return false
	}

	// Reject if contains common keywords
	lower := strings.ToLower(message)
	commonWords := []string{"how", "what", "where", "when", "why", "show", "find", "search", "get", "list", "help", "status", "update", "check", "error", "video", "tag", "scraper", "library", "jdownloader", "hello", "hi", "hey", "please", "thanks", "yes", "no"}
	for _, w := range commonWords {
		if strings.Contains(lower, w) {
			return false
		}
	}

	// Check if this name exists in the performers table
	var exists int
	s.db.QueryRow("SELECT COUNT(*) FROM performers WHERE name LIKE ?", "%"+message+"%").Scan(&exists)
	return exists > 0
}

// searchPerformerAcrossAll searches for a performer name in both library and scraped content
func (s *AICompanionService) searchPerformerAcrossAll(name string) (string, error) {
	var response strings.Builder

	// Search scraped videos
	var scrapedCount int
	s.db.QueryRow("SELECT COUNT(*) FROM scraped_videos WHERE performers LIKE ? OR title LIKE ?", "%"+name+"%", "%"+name+"%").Scan(&scrapedCount)

	if scrapedCount > 0 {
		response.WriteString(fmt.Sprintf("\n🔍 **Scraped Content:** Found %d items mentioning \"%s\"\n", scrapedCount, name))

		rows, err := s.db.Query(`SELECT source, title FROM scraped_videos WHERE performers LIKE ? OR title LIKE ? ORDER BY discovered_at DESC LIMIT 5`, "%"+name+"%", "%"+name+"%")
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var source, title string
				if rows.Scan(&source, &title) == nil {
					response.WriteString(fmt.Sprintf("  - [%s] %s\n", strings.ToUpper(source), title))
				}
			}
			if scrapedCount > 5 {
				response.WriteString(fmt.Sprintf("  _... and %d more_\n", scrapedCount-5))
			}
		}
	}

	// Search forum posts
	var postCount int
	s.db.QueryRow("SELECT COUNT(*) FROM scraped_posts WHERE plain_text LIKE ?", "%"+name+"%").Scan(&postCount)
	if postCount > 0 {
		response.WriteString(fmt.Sprintf("\n💬 **Forum Posts:** %d posts mention \"%s\"\n", postCount, name))
	}

	if response.Len() == 0 {
		return "", nil
	}
	return response.String(), nil
}

// searchPerformerWithTag searches for a performer's content with a specific tag/keyword
func (s *AICompanionService) searchPerformerWithTag(performerName, tag string) (string, error) {
	var response strings.Builder
	response.WriteString(fmt.Sprintf("🔍 **Searching \"%s\" + \"%s\"**\n\n", performerName, tag))

	// Search in library videos
	rows, err := s.db.Query(`
		SELECT v.title
		FROM videos v
		JOIN video_performers vp ON v.id = vp.video_id
		JOIN performers p ON vp.performer_id = p.id
		WHERE p.name LIKE ? AND v.title LIKE ?
		LIMIT 10
	`, "%"+performerName+"%", "%"+tag+"%")

	libraryCount := 0
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var title string
			if rows.Scan(&title) == nil {
				libraryCount++
				response.WriteString(fmt.Sprintf("📹 %s\n", title))
			}
		}
	}

	// Search scraped content
	scrapedRows, err := s.db.Query(`
		SELECT source, title FROM scraped_videos
		WHERE (performers LIKE ? OR title LIKE ?) AND (title LIKE ? OR tags LIKE ?)
		ORDER BY discovered_at DESC LIMIT 10
	`, "%"+performerName+"%", "%"+performerName+"%", "%"+tag+"%", "%"+tag+"%")

	scrapedCount := 0
	if err == nil {
		defer scrapedRows.Close()
		for scrapedRows.Next() {
			var source, title string
			if scrapedRows.Scan(&source, &title) == nil {
				scrapedCount++
				response.WriteString(fmt.Sprintf("🔗 [%s] %s\n", strings.ToUpper(source), title))
			}
		}
	}

	// Search forum posts
	var postCount int
	s.db.QueryRow("SELECT COUNT(*) FROM scraped_posts WHERE plain_text LIKE ? AND plain_text LIKE ?", "%"+performerName+"%", "%"+tag+"%").Scan(&postCount)

	if libraryCount == 0 && scrapedCount == 0 && postCount == 0 {
		response.WriteString(fmt.Sprintf("No results found for \"%s\" with \"%s\" in library, scraped videos, or forum posts.", performerName, tag))
	} else {
		response.WriteString(fmt.Sprintf("\n**Totals:** %d library videos, %d scraped items, %d forum posts\n", libraryCount, scrapedCount, postCount))
	}

	return response.String(), nil
}

// extractNameAfterPhrase extracts a name/subject after a trigger phrase
func extractNameAfterPhrase(message string, phrases []string) string {
	lower := strings.ToLower(message)
	for _, phrase := range phrases {
		idx := strings.Index(lower, phrase)
		if idx >= 0 {
			after := strings.TrimSpace(message[idx+len(phrase):])
			// Remove surrounding quotes if present
			after = strings.Trim(after, "\"'")
			if after != "" {
				return after
			}
		}
	}
	return ""
}

// listForumPosts lists actual forum posts matching a performer name and optional keyword
func (s *AICompanionService) listForumPosts(performerName, keyword string, limit int) (string, error) {
	var response strings.Builder
	var rows *sql.Rows
	var err error

	if keyword != "" {
		response.WriteString(fmt.Sprintf("💬 **Forum posts mentioning \"%s\" + \"%s\":**\n\n", performerName, keyword))
		rows, err = s.db.Query(`
			SELECT sp.plain_text, sp.author, st.title as thread_title
			FROM scraped_posts sp
			LEFT JOIN scraped_threads st ON sp.thread_id = st.id
			WHERE sp.plain_text LIKE ? AND sp.plain_text LIKE ?
			ORDER BY sp.scraped_at DESC LIMIT ?
		`, "%"+performerName+"%", "%"+keyword+"%", limit)
	} else {
		response.WriteString(fmt.Sprintf("💬 **Forum posts mentioning \"%s\":**\n\n", performerName))
		rows, err = s.db.Query(`
			SELECT sp.plain_text, sp.author, st.title as thread_title
			FROM scraped_posts sp
			LEFT JOIN scraped_threads st ON sp.thread_id = st.id
			WHERE sp.plain_text LIKE ?
			ORDER BY sp.scraped_at DESC LIMIT ?
		`, "%"+performerName+"%", limit)
	}

	if err != nil {
		return "Failed to search forum posts.", err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var text, author, threadTitle string
		if rows.Scan(&text, &author, &threadTitle) == nil {
			count++
			// Truncate long post text
			if len(text) > 200 {
				text = text[:200] + "..."
			}
			response.WriteString(fmt.Sprintf("**%d.** [%s] by **%s**\n> %s\n\n", count, threadTitle, author, text))
		}
	}

	if count == 0 {
		return fmt.Sprintf("No forum posts found matching those criteria."), nil
	}

	response.WriteString(fmt.Sprintf("_Showing %d results_", count))
	return response.String(), nil
}

// searchScrapedContentByKeyword searches all scraped content (videos + posts) by keyword
func (s *AICompanionService) searchScrapedContentByKeyword(keyword string, limit int) (string, error) {
	var response strings.Builder
	response.WriteString(fmt.Sprintf("🔍 **Scraped content matching \"%s\":**\n\n", keyword))

	// Search scraped videos
	videoRows, err := s.db.Query(`
		SELECT source, title, performers, url
		FROM scraped_videos
		WHERE title LIKE ? OR performers LIKE ? OR tags LIKE ?
		ORDER BY discovered_at DESC LIMIT ?
	`, "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", limit)

	videoCount := 0
	if err == nil {
		defer videoRows.Close()
		for videoRows.Next() {
			var source, title, performers, url string
			if videoRows.Scan(&source, &title, &performers, &url) == nil {
				videoCount++
				response.WriteString(fmt.Sprintf("🎬 [%s] %s", strings.ToUpper(source), title))
				if performers != "" {
					response.WriteString(fmt.Sprintf(" (%s)", performers))
				}
				response.WriteString("\n")
			}
		}
	}

	// Search forum posts
	postRows, err := s.db.Query(`
		SELECT sp.plain_text, sp.author, st.title as thread_title
		FROM scraped_posts sp
		LEFT JOIN scraped_threads st ON sp.thread_id = st.id
		WHERE sp.plain_text LIKE ?
		ORDER BY sp.scraped_at DESC LIMIT ?
	`, "%"+keyword+"%", limit)

	postCount := 0
	if err == nil {
		defer postRows.Close()
		if videoCount > 0 {
			response.WriteString("\n")
		}
		for postRows.Next() {
			var text, author, threadTitle string
			if postRows.Scan(&text, &author, &threadTitle) == nil {
				postCount++
				if len(text) > 150 {
					text = text[:150] + "..."
				}
				response.WriteString(fmt.Sprintf("💬 [%s] by %s: %s\n", threadTitle, author, text))
			}
		}
	}

	if videoCount == 0 && postCount == 0 {
		return fmt.Sprintf("No scraped content found matching \"%s\".", keyword), nil
	}

	response.WriteString(fmt.Sprintf("\n**Found:** %d videos, %d forum posts", videoCount, postCount))
	return response.String(), nil
}

// Chat processes user messages with rule-based intelligence
func (s *AICompanionService) Chat(message string, history []map[string]string) (string, error) {
	messageLower := strings.ToLower(message)

	// ===== CONVERSATION HISTORY CONTEXT =====
	// Check last assistant response for context-aware follow-ups
	var lastAssistantMsg string
	if len(history) > 0 {
		for i := len(history) - 1; i >= 0; i-- {
			if history[i]["role"] == "assistant" {
				lastAssistantMsg = history[i]["content"]
				break
			}
		}
	}

	// ===== FOLLOW-UP HANDLING =====
	// Use conversation history to handle follow-up questions
	if lastAssistantMsg != "" {
		lastLower := strings.ToLower(lastAssistantMsg)

		// Extract quoted terms from previous response for context
		reQuoted := regexp.MustCompile(`"([^"]+)"`)
		prevQuotedTerms := reQuoted.FindAllStringSubmatch(lastAssistantMsg, -1)

		// "List/show the forum posts" after a response that mentioned forum posts
		if (strings.Contains(messageLower, "list") || strings.Contains(messageLower, "show") || strings.Contains(messageLower, "can you list") || strings.Contains(messageLower, "can you show")) &&
			(strings.Contains(messageLower, "forum") || strings.Contains(messageLower, "post")) &&
			strings.Contains(lastLower, "forum posts") {
			// Extract performer name and search term from previous response
			re := regexp.MustCompile(`(?i)Searching "([^"]+)" \+ "([^"]+)"`)
			matches := re.FindStringSubmatch(lastAssistantMsg)
			if len(matches) >= 3 {
				return s.listForumPosts(matches[1], matches[2], 10)
			}
			if len(prevQuotedTerms) > 0 {
				return s.listForumPosts(prevQuotedTerms[0][1], "", 10)
			}
		}

		// "Is this from X scraper?" / "Is that from Simpcity?" - source questions
		if (strings.Contains(messageLower, "is this from") || strings.Contains(messageLower, "is that from") ||
			strings.Contains(messageLower, "is the") && strings.Contains(messageLower, "from") ||
			strings.Contains(messageLower, "which scraper") || strings.Contains(messageLower, "what source") ||
			strings.Contains(messageLower, "where") && (strings.Contains(messageLower, "from") || strings.Contains(messageLower, "come from"))) {

			// Check if previous response has thread/source info
			if strings.Contains(lastLower, "forum posts") || strings.Contains(lastLower, "scraped") || strings.Contains(lastLower, "thread") {
				var response strings.Builder
				response.WriteString("The forum posts are from the **Simpcity Forum Scraper**. ")
				response.WriteString("This scraper collects posts and download links from forum threads.\n\n")

				// Show which threads contained the data
				if len(prevQuotedTerms) > 0 {
					searchTerm := prevQuotedTerms[0][1]
					rows, err := s.db.Query(`
						SELECT st.title, COUNT(*) as cnt
						FROM scraped_posts sp
						JOIN scraped_threads st ON sp.thread_id = st.id
						WHERE sp.plain_text LIKE ?
						GROUP BY st.id
						ORDER BY cnt DESC LIMIT 5
					`, "%"+searchTerm+"%")
					if err == nil {
						defer rows.Close()
						response.WriteString(fmt.Sprintf("**Threads containing \"%s\":**\n", searchTerm))
						for rows.Next() {
							var title string
							var cnt int
							if rows.Scan(&title, &cnt) == nil {
								response.WriteString(fmt.Sprintf("  - %s (%d posts)\n", title, cnt))
							}
						}
					}
				}
				return response.String(), nil
			}
		}

		// "Yes" / "Yes please" / "Sure" - affirmative follow-up with context
		if messageLower == "yes" || messageLower == "yes please" || messageLower == "sure" || messageLower == "yeah" || messageLower == "more" || messageLower == "go on" || messageLower == "tell me more" {
			// If previous response mentioned forum posts, list them
			if strings.Contains(lastLower, "forum posts") {
				re := regexp.MustCompile(`(?i)Searching "([^"]+)" \+ "([^"]+)"`)
				matches := re.FindStringSubmatch(lastAssistantMsg)
				if len(matches) >= 3 {
					return s.listForumPosts(matches[1], matches[2], 10)
				}
				if len(prevQuotedTerms) > 0 {
					return s.listForumPosts(prevQuotedTerms[0][1], "", 10)
				}
			}

			// If previous response mentioned scraped content, show more
			if strings.Contains(lastLower, "scraped content") || strings.Contains(lastLower, "scraped items") {
				if len(prevQuotedTerms) > 0 {
					return s.searchScrapedContentByKeyword(prevQuotedTerms[0][1], 15)
				}
			}

			// If previous response was about a performer, show full details
			rePerfName := regexp.MustCompile(`\*\*([A-Z][a-zA-Z]+(?: [A-Z][a-zA-Z]+)+)\*\*`)
			perfMatches := rePerfName.FindStringSubmatch(lastAssistantMsg)
			if len(perfMatches) >= 2 {
				result, err := s.FindPerformer(perfMatches[1])
				if err == nil {
					scrapedInfo, _ := s.searchPerformerAcrossAll(perfMatches[1])
					if scrapedInfo != "" {
						result += "\n" + scrapedInfo
					}
					return result, nil
				}
			}
		}

		// "Can you list/show" generic follow-up
		if (strings.Contains(messageLower, "can you") || strings.Contains(messageLower, "show me") || strings.Contains(messageLower, "what are")) &&
			!strings.Contains(messageLower, "scraper") && !strings.Contains(messageLower, "forum") {
			// Try to use context from previous response
			if strings.Contains(lastLower, "scraped content") || strings.Contains(lastLower, "scraped items") {
				if len(prevQuotedTerms) > 0 {
					return s.searchScrapedContentByKeyword(prevQuotedTerms[0][1], 15)
				}
			}
			if strings.Contains(lastLower, "forum posts") {
				if len(prevQuotedTerms) > 0 {
					return s.listForumPosts(prevQuotedTerms[0][1], "", 10)
				}
			}
		}
	}

	// ===== GREETINGS =====
	if messageLower == "hi" || messageLower == "hello" || messageLower == "hey" || messageLower == "yo" || messageLower == "sup" || strings.HasPrefix(messageLower, "hello ") || strings.HasPrefix(messageLower, "hi ") || strings.HasPrefix(messageLower, "hey ") {
		var videoCount, performerCount int
		s.db.QueryRow("SELECT COUNT(*) FROM videos").Scan(&videoCount)
		s.db.QueryRow("SELECT COUNT(*) FROM performers").Scan(&performerCount)
		var runningCount int
		s.db.QueryRow("SELECT COUNT(*) FROM activity_logs WHERE status = 'running'").Scan(&runningCount)
		greeting := fmt.Sprintf("Hey! 👋 Your library has **%d videos** and **%d performers**.", videoCount, performerCount)
		if runningCount > 0 {
			greeting += fmt.Sprintf(" There are **%d tasks** currently running.", runningCount)
		} else {
			greeting += " Everything is idle right now."
		}
		greeting += "\n\nWhat would you like to know?"
		return greeting, nil
	}

	// ===== UPDATE / WHAT'S HAPPENING =====
	if messageLower == "update" || messageLower == "give me an update" || messageLower == "what's new" || messageLower == "whats new" || strings.Contains(messageLower, "what's happening") || strings.Contains(messageLower, "whats happening") || (strings.Contains(messageLower, "give") && strings.Contains(messageLower, "update")) || messageLower == "overview" || messageLower == "summary" {
		return s.generateAppUpdate()
	}

	// ===== SHORT CONVERSATIONAL FOLLOW-UPS (no history context) =====
	if lastAssistantMsg == "" && (messageLower == "yes" || messageLower == "yes please" || messageLower == "sure" || messageLower == "ok" || messageLower == "yeah" || messageLower == "tell me more" || messageLower == "more" || messageLower == "go on") {
		return "Could you be more specific about what you'd like to know? I can help with:\n\n" +
			"- **Performer info**: Ask about any performer by name (e.g. \"Kendra Lust\")\n" +
			"- **Library stats**: \"How many videos?\", \"Top performers\"\n" +
			"- **Scrapers**: \"Scraper summary\", \"PornHub stats\"\n" +
			"- **System**: \"Status\", \"Check for errors\"\n" +
			"- **JDownloader**: \"JDownloader status\"", nil
	}

	// ===== DIRECT PERFORMER NAME LOOKUP =====
	// If the message looks like a performer name (2-4 words, capitalized, no common keywords)
	if s.looksLikePerformerName(message) {
		result, err := s.FindPerformer(message)
		if err == nil && !strings.Contains(result, "couldn't find") {
			return result, nil
		}
		// Also search scraped content for this name
		scrapedResult, _ := s.searchPerformerAcrossAll(message)
		if scrapedResult != "" {
			return scrapedResult, nil
		}
		// Fall through to other patterns
	}

	// ===== "ALL DATA ON X" / "EVERYTHING ABOUT X" =====
	if (strings.Contains(messageLower, "all data") || strings.Contains(messageLower, "everything about") || strings.Contains(messageLower, "tell me about") || strings.Contains(messageLower, "info on") || strings.Contains(messageLower, "details on") || strings.Contains(messageLower, "details about")) {
		// Extract the name after the keyword phrase
		name := extractNameAfterPhrase(message, []string{"all data on", "everything about", "tell me about", "info on", "details on", "details about"})
		if name != "" {
			result, err := s.FindPerformer(name)
			if err == nil {
				// Also check scraped content
				scrapedInfo, _ := s.searchPerformerAcrossAll(name)
				if scrapedInfo != "" {
					result += "\n" + scrapedInfo
				}
				return result, nil
			}
		}
	}

	// ===== "DOES X HAVE ANY Y" pattern =====
	if strings.Contains(messageLower, "does") && (strings.Contains(messageLower, "have any") || strings.Contains(messageLower, "have a")) {
		// Extract performer name and search term
		re := regexp.MustCompile(`(?i)does\s+["']?(.+?)["']?\s+have\s+(?:any|a)\s+["']?(.+?)["']?\s*(?:scenes?|videos?|content)?[?]?$`)
		matches := re.FindStringSubmatch(message)
		if len(matches) >= 3 {
			performerName := strings.TrimSpace(matches[1])
			searchTerm := strings.TrimSpace(matches[2])
			return s.searchPerformerWithTag(performerName, searchTerm)
		}
	}

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
	// Enhanced to handle: "Does thread X have any Y posts?", "Does X contain Y?", etc.
	if (strings.Contains(messageLower, "post") || strings.Contains(messageLower, "thread")) &&
		(strings.Contains(messageLower, "contain") || strings.Contains(messageLower, "include") ||
			strings.Contains(messageLower, "have") || strings.Contains(messageLower, "with") ||
			strings.Contains(messageLower, "mention") || strings.Contains(messageLower, "about") ||
			strings.Contains(messageLower, "does") || strings.Contains(messageLower, "any")) {

		// Extract thread name if specified
		var threadName string
		var searchKeyword string

		// Look for quoted text first (thread name or keyword)
		re := regexp.MustCompile(`["']([^"']+)["']`)
		matches := re.FindAllStringSubmatch(message, -1)

		if len(matches) >= 2 {
			// Two quoted texts found
			// Check which comes first - if "thread" or "scraper" comes before first quote, it's the thread name
			firstQuoteIndex := strings.Index(message, matches[0][0])
			threadIndex := strings.Index(strings.ToLower(message[:firstQuoteIndex]), "thread")
			scraperIndex := strings.Index(strings.ToLower(message[:firstQuoteIndex]), "scraper")

			if threadIndex != -1 || scraperIndex != -1 {
				// First quoted text is thread name, second is keyword
				threadName = matches[0][1]
				searchKeyword = matches[1][1]
			} else {
				// First is keyword, second might be thread name
				searchKeyword = matches[0][1]
				threadName = matches[1][1]
			}
		} else if len(matches) == 1 {
			// Only one quoted text - determine if it's thread name or keyword
			// If "thread" or "scraper" appears before the quote, it's a thread name
			quoteIndex := strings.Index(message, matches[0][0])
			preText := strings.ToLower(message[:quoteIndex])
			if strings.Contains(preText, "thread") || strings.Contains(preText, "scraper") {
				threadName = matches[0][1]
			} else {
				searchKeyword = matches[0][1]
			}
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

	// ===== SEARCH SCRAPED CONTENT (broad pattern) =====
	// Handles: "search for X from scrapers", "find X in scraped content", "search scrapers for X"
	if (strings.Contains(messageLower, "search") || strings.Contains(messageLower, "find") || strings.Contains(messageLower, "look")) &&
		(strings.Contains(messageLower, "scraper") || strings.Contains(messageLower, "scraped") || strings.Contains(messageLower, "from the scraper")) {

		// Extract keyword - try quotes first
		re := regexp.MustCompile(`["']([^"']+)["']`)
		matches := re.FindAllStringSubmatch(message, -1)
		keyword := ""
		if len(matches) > 0 {
			keyword = matches[0][1]
		}

		// If no quotes, extract the search subject
		if keyword == "" {
			// Try "search for X from", "find X from", "search X from"
			reFor := regexp.MustCompile(`(?i)(?:search|find|look)\s+(?:for\s+)?(.+?)\s+(?:from|in|on)\s+(?:the\s+)?scrap`)
			m := reFor.FindStringSubmatch(message)
			if len(m) >= 2 {
				keyword = strings.TrimSpace(m[1])
			}
		}

		// Try "search scrapers for X"
		if keyword == "" {
			reAfter := regexp.MustCompile(`(?i)(?:search|find|look)\s+(?:the\s+)?scrapers?\s+(?:for\s+)(.+)`)
			m := reAfter.FindStringSubmatch(message)
			if len(m) >= 2 {
				keyword = strings.TrimSpace(m[1])
			}
		}

		// Last resort: take everything after "search for" or "find"
		if keyword == "" {
			words := strings.Fields(message)
			for i, w := range words {
				wl := strings.ToLower(w)
				if (wl == "for" || wl == "find" || wl == "search") && i+1 < len(words) {
					remaining := strings.Join(words[i+1:], " ")
					// Remove "from scrapers", "in scraped", etc.
					remaining = regexp.MustCompile(`(?i)\s*(from|in|on)\s+(the\s+)?scrap\w+\s*$`).ReplaceAllString(remaining, "")
					remaining = strings.TrimSpace(remaining)
					if remaining != "" && remaining != "scrapers" && remaining != "scraped" {
						keyword = remaining
					}
					break
				}
			}
		}

		if keyword != "" {
			// Remove trailing "videos" or "content" from keyword
			keyword = regexp.MustCompile(`(?i)\s*(videos?|content|items?)\s*$`).ReplaceAllString(keyword, "")
			keyword = strings.TrimSpace(keyword)
			if keyword != "" {
				return s.searchScrapedContentByKeyword(keyword, 15)
			}
		}
	}

	// Scraper stats (only for stats/summary requests, not searches)
	if (strings.Contains(messageLower, "scraper") && (strings.Contains(messageLower, "stat") || strings.Contains(messageLower, "summary") || strings.Contains(messageLower, "overview") || messageLower == "scraper" || messageLower == "scrapers")) ||
		(strings.Contains(messageLower, "scraped") && !strings.Contains(messageLower, "search") && !strings.Contains(messageLower, "find")) ||
		(messageLower == "forum" || messageLower == "forum stats" || (strings.Contains(messageLower, "forum") && strings.Contains(messageLower, "stat"))) {
		var threadCount, postCount, linkCount, activeLinks int
		s.db.QueryRow("SELECT COUNT(*) FROM scraped_threads").Scan(&threadCount)
		s.db.QueryRow("SELECT COUNT(*) FROM scraped_posts").Scan(&postCount)
		s.db.QueryRow("SELECT COUNT(*) FROM scraped_download_links").Scan(&linkCount)
		s.db.QueryRow("SELECT COUNT(*) FROM scraped_download_links WHERE status = 'active'").Scan(&activeLinks)

		// Get video scraper stats
		var totalScrapedVideos, pornhubCount, luxuretvCount, sexuriaCount int
		s.db.QueryRow("SELECT COUNT(*) FROM scraped_videos").Scan(&totalScrapedVideos)
		s.db.QueryRow("SELECT COUNT(*) FROM scraped_videos WHERE source = 'pornhub'").Scan(&pornhubCount)
		s.db.QueryRow("SELECT COUNT(*) FROM scraped_videos WHERE source = 'luxuretv'").Scan(&luxuretvCount)
		s.db.QueryRow("SELECT COUNT(*) FROM scraped_videos WHERE source = 'sexuria'").Scan(&sexuriaCount)

		var response strings.Builder
		response.WriteString("🕷️ Web Scraper Statistics:\n\n")
		response.WriteString("**Forum Scraper:**\n")
		response.WriteString(fmt.Sprintf("📰 Threads Scraped: %d\n", threadCount))
		response.WriteString(fmt.Sprintf("💬 Posts Collected: %d\n", postCount))
		response.WriteString(fmt.Sprintf("🔗 Download Links: %d\n", linkCount))
		response.WriteString(fmt.Sprintf("✅ Active Links: %d\n\n", activeLinks))

		if totalScrapedVideos > 0 {
			response.WriteString("**Video Scrapers:**\n")
			response.WriteString(fmt.Sprintf("🎬 Total Scraped Videos: %d\n", totalScrapedVideos))
			if pornhubCount > 0 {
				response.WriteString(fmt.Sprintf("   • PornHub: %d videos\n", pornhubCount))
			}
			if luxuretvCount > 0 {
				response.WriteString(fmt.Sprintf("   • LuxureTV: %d videos\n", luxuretvCount))
			}
			if sexuriaCount > 0 {
				response.WriteString(fmt.Sprintf("   • Sexuria: %d items\n", sexuriaCount))
			}
			response.WriteString("\n")
		}

		response.WriteString("💡 Tip: Ask me about 'pornhub videos', 'luxuretv stats', or 'sexuria items' for detailed info!")
		return response.String(), nil
	}

	// PornHub specific queries
	if strings.Contains(messageLower, "pornhub") {
		var count int
		s.db.QueryRow("SELECT COUNT(*) FROM scraped_videos WHERE source = 'pornhub'").Scan(&count)

		if count == 0 {
			return "No PornHub videos have been scraped yet. Use the Scraper page to start scraping PornHub videos.", nil
		}

		var response strings.Builder
		response.WriteString(fmt.Sprintf("🟠 **PornHub Scraped Videos: %d**\n\n", count))

		// Get top categories
		categoryRows, err := s.db.Query(`
			SELECT site_type, COUNT(*) as cnt
			FROM scraped_videos
			WHERE source = 'pornhub'
			GROUP BY site_type
			ORDER BY cnt DESC
			LIMIT 5
		`)
		if err == nil {
			defer categoryRows.Close()
			response.WriteString("**Top Categories:**\n")
			for categoryRows.Next() {
				var siteType string
				var cnt int
				if categoryRows.Scan(&siteType, &cnt) == nil {
					response.WriteString(fmt.Sprintf("• %s: %d videos\n", siteType, cnt))
				}
			}
			response.WriteString("\n")
		}

		// Get top performers from scraped videos
		performerRows, err := s.db.Query(`
			SELECT performers, COUNT(*) as cnt
			FROM scraped_videos
			WHERE source = 'pornhub' AND performers != '' AND performers IS NOT NULL
			GROUP BY performers
			ORDER BY cnt DESC
			LIMIT 5
		`)
		if err == nil {
			defer performerRows.Close()
			response.WriteString("**Top Performers:**\n")
			hasPerformers := false
			for performerRows.Next() {
				var performers string
				var cnt int
				if performerRows.Scan(&performers, &cnt) == nil && performers != "" {
					hasPerformers = true
					response.WriteString(fmt.Sprintf("• %s: %d videos\n", performers, cnt))
				}
			}
			if !hasPerformers {
				response.WriteString("No performer data collected yet.\n")
			}
			response.WriteString("\n")
		}

		// Get average rating and views
		var avgRating float64
		var avgViews float64
		s.db.QueryRow("SELECT AVG(rating), AVG(views) FROM scraped_videos WHERE source = 'pornhub' AND rating > 0").Scan(&avgRating, &avgViews)
		if avgRating > 0 {
			response.WriteString(fmt.Sprintf("**Stats:**\n• Avg Rating: %.1f%%\n", avgRating))
			if avgViews > 0 {
				response.WriteString(fmt.Sprintf("• Avg Views: %.0f\n", avgViews))
			}
		}

		return response.String(), nil
	}

	// LuxureTV specific queries
	if strings.Contains(messageLower, "luxuretv") || strings.Contains(messageLower, "luxure") {
		var count int
		s.db.QueryRow("SELECT COUNT(*) FROM scraped_videos WHERE source = 'luxuretv'").Scan(&count)

		if count == 0 {
			return "No LuxureTV videos have been scraped yet. Use the Scraper page to start scraping LuxureTV videos.", nil
		}

		var response strings.Builder
		response.WriteString(fmt.Sprintf("🎬 **LuxureTV Scraped Videos: %d**\n\n", count))

		// Get categories breakdown
		categoryRows, err := s.db.Query(`
			SELECT site_type, COUNT(*) as cnt
			FROM scraped_videos
			WHERE source = 'luxuretv'
			GROUP BY site_type
			ORDER BY cnt DESC
			LIMIT 10
		`)
		if err == nil {
			defer categoryRows.Close()
			response.WriteString("**Categories:**\n")
			for categoryRows.Next() {
				var siteType string
				var cnt int
				if categoryRows.Scan(&siteType, &cnt) == nil {
					response.WriteString(fmt.Sprintf("• %s: %d videos\n", siteType, cnt))
				}
			}
			response.WriteString("\n")
		}

		// Get content type breakdown
		contentRows, err := s.db.Query(`
			SELECT COALESCE(content_type, 'regular') as ct, COUNT(*) as cnt
			FROM scraped_videos
			WHERE source = 'luxuretv'
			GROUP BY ct
			ORDER BY cnt DESC
		`)
		if err == nil {
			defer contentRows.Close()
			response.WriteString("**Content Types:**\n")
			for contentRows.Next() {
				var contentType string
				var cnt int
				if contentRows.Scan(&contentType, &cnt) == nil {
					response.WriteString(fmt.Sprintf("• %s: %d videos\n", contentType, cnt))
				}
			}
		}

		return response.String(), nil
	}

	// Sexuria specific queries
	if strings.Contains(messageLower, "sexuria") {
		var count int
		s.db.QueryRow("SELECT COUNT(*) FROM scraped_videos WHERE source = 'sexuria'").Scan(&count)

		if count == 0 {
			return "No Sexuria items have been scraped yet. Use the Scraper page to start scraping Sexuria galleries.", nil
		}

		var response strings.Builder
		response.WriteString(fmt.Sprintf("🔞 **Sexuria Scraped Items: %d**\n\n", count))

		// Get top pornstars
		performerRows, err := s.db.Query(`
			SELECT performers, COUNT(*) as cnt
			FROM scraped_videos
			WHERE source = 'sexuria' AND performers != '' AND performers IS NOT NULL
			GROUP BY performers
			ORDER BY cnt DESC
			LIMIT 10
		`)
		if err == nil {
			defer performerRows.Close()
			response.WriteString("**Top Performers:**\n")
			hasPerformers := false
			for performerRows.Next() {
				var performers string
				var cnt int
				if performerRows.Scan(&performers, &cnt) == nil && performers != "" {
					hasPerformers = true
					response.WriteString(fmt.Sprintf("• %s: %d galleries\n", performers, cnt))
				}
			}
			if !hasPerformers {
				response.WriteString("Performer data included in item details.\n")
			}
			response.WriteString("\n")
		}

		// Get top tags
		tagRows, err := s.db.Query(`
			SELECT tags, COUNT(*) as cnt
			FROM scraped_videos
			WHERE source = 'sexuria' AND tags != '' AND tags IS NOT NULL
			GROUP BY tags
			ORDER BY cnt DESC
			LIMIT 10
		`)
		if err == nil {
			defer tagRows.Close()
			response.WriteString("**Popular Tags:**\n")
			hasTags := false
			for tagRows.Next() {
				var tags string
				var cnt int
				if tagRows.Scan(&tags, &cnt) == nil && tags != "" {
					hasTags = true
					// Show first few tags if it's a comma-separated list
					if len(tags) > 50 {
						tags = tags[:47] + "..."
					}
					response.WriteString(fmt.Sprintf("• %s (%d)\n", tags, cnt))
				}
			}
			if !hasTags {
				response.WriteString("Tag data included in item details.\n")
			}
		}

		return response.String(), nil
	}

	// Search scraped videos
	if (strings.Contains(messageLower, "search") || strings.Contains(messageLower, "find")) &&
		(strings.Contains(messageLower, "scraped video") || strings.Contains(messageLower, "scraped content")) {

		// Extract search term from quotes
		re := regexp.MustCompile(`["']([^"']+)["']`)
		matches := re.FindAllStringSubmatch(message, -1)

		searchTerm := ""
		if len(matches) > 0 {
			searchTerm = matches[0][1]
		}

		if searchTerm == "" {
			return "Please specify what to search for. Try: 'search scraped videos for \"keyword\"'", nil
		}

		rows, err := s.db.Query(`
			SELECT source, title, performers, url
			FROM scraped_videos
			WHERE title LIKE ? OR performers LIKE ? OR tags LIKE ?
			ORDER BY discovered_at DESC
			LIMIT 10
		`, "%"+searchTerm+"%", "%"+searchTerm+"%", "%"+searchTerm+"%")

		if err != nil {
			return "Failed to search scraped videos.", err
		}
		defer rows.Close()

		var response strings.Builder
		response.WriteString(fmt.Sprintf("🔍 **Search Results for '%s':**\n\n", searchTerm))

		count := 0
		for rows.Next() {
			var source, title, performers, url string
			if rows.Scan(&source, &title, &performers, &url) == nil {
				count++
				response.WriteString(fmt.Sprintf("%d. [%s] %s\n", count, strings.ToUpper(source), title))
				if performers != "" {
					response.WriteString(fmt.Sprintf("   Performers: %s\n", performers))
				}
			}
		}

		if count == 0 {
			return fmt.Sprintf("No scraped videos found matching '%s'.", searchTerm), nil
		}

		response.WriteString(fmt.Sprintf("\nFound %d results.", count))
		return response.String(), nil
	}

	// Videos by performer query
	if strings.Contains(messageLower, "video") && (strings.Contains(messageLower, "by") || strings.Contains(messageLower, "with") || strings.Contains(messageLower, "featuring")) &&
		(strings.Contains(messageLower, "performer") || strings.Contains(messageLower, "actor") || strings.Contains(messageLower, "model")) {

		// Extract performer name from quotes or after "by/with/featuring"
		re := regexp.MustCompile(`["']([^"']+)["']`)
		matches := re.FindAllStringSubmatch(message, -1)

		performerName := ""
		if len(matches) > 0 {
			performerName = matches[0][1]
		} else {
			words := strings.Fields(message)
			for i, word := range words {
				wordLower := strings.ToLower(word)
				if (wordLower == "by" || wordLower == "with" || wordLower == "featuring") && i+1 < len(words) {
					performerName = words[i+1]
					break
				}
			}
		}

		if performerName == "" {
			return "Please specify a performer name. Try: 'videos by \"Performer Name\"'", nil
		}

		query := `
			SELECT v.id, v.title, v.path
			FROM videos v
			JOIN video_performers vp ON v.id = vp.video_id
			JOIN performers p ON vp.performer_id = p.id
			WHERE p.name LIKE ?
			ORDER BY v.created_at DESC
			LIMIT 10
		`

		rows, err := s.db.Query(query, "%"+performerName+"%")
		if err != nil {
			return "Failed to search videos by performer.", err
		}
		defer rows.Close()

		var results []string
		count := 0
		for rows.Next() {
			var id int64
			var title, path string
			if err := rows.Scan(&id, &title, &path); err != nil {
				continue
			}
			results = append(results, fmt.Sprintf("- %s (ID: %d)", title, id))
			count++
		}

		if count == 0 {
			return fmt.Sprintf("No videos found featuring '%s'. Check the performer name and try again.", performerName), nil
		}

		var response strings.Builder
		response.WriteString(fmt.Sprintf("🎬 Videos featuring '%s' (%d results):\n\n", performerName, count))
		for _, result := range results {
			response.WriteString(result + "\n")
		}
		if count >= 10 {
			response.WriteString("\n(Showing first 10 results)")
		}

		return response.String(), nil
	}

	// Videos by tag query
	if strings.Contains(messageLower, "video") && (strings.Contains(messageLower, "tagged") || strings.Contains(messageLower, "tag")) {
		// Extract tag name from quotes or after "tagged/tag"
		re := regexp.MustCompile(`["']([^"']+)["']`)
		matches := re.FindAllStringSubmatch(message, -1)

		tagName := ""
		if len(matches) > 0 {
			tagName = matches[0][1]
		}

		if tagName == "" {
			return "Please specify a tag name. Try: 'videos tagged \"BBC\"'", nil
		}

		query := `
			SELECT v.id, v.title, COUNT(vt.tag_id) as tag_count
			FROM videos v
			JOIN video_tags vt ON v.id = vt.video_id
			JOIN tags t ON vt.tag_id = t.id
			WHERE t.name LIKE ?
			GROUP BY v.id
			ORDER BY v.created_at DESC
			LIMIT 10
		`

		rows, err := s.db.Query(query, "%"+tagName+"%")
		if err != nil {
			return "Failed to search videos by tag.", err
		}
		defer rows.Close()

		var results []string
		count := 0
		for rows.Next() {
			var id int64
			var title string
			var tagCount int
			if err := rows.Scan(&id, &title, &tagCount); err != nil {
				continue
			}
			results = append(results, fmt.Sprintf("- %s (ID: %d)", title, id))
			count++
		}

		if count == 0 {
			return fmt.Sprintf("No videos found with tag '%s'.", tagName), nil
		}

		var response strings.Builder
		response.WriteString(fmt.Sprintf("🏷️ Videos tagged '%s' (%d results):\n\n", tagName, count))
		for _, result := range results {
			response.WriteString(result + "\n")
		}
		if count >= 10 {
			response.WriteString("\n(Showing first 10 results)")
		}

		return response.String(), nil
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

	// Enhanced performer information query with scraped threads
	if (strings.Contains(messageLower, "performer") || strings.Contains(messageLower, "about")) &&
		(strings.Contains(messageLower, "information") || strings.Contains(messageLower, "details") ||
			strings.Contains(messageLower, "tell me about") || strings.Contains(messageLower, "info")) {

		// Extract performer name from quotes or after keywords
		re := regexp.MustCompile(`["']([^"']+)["']`)
		matches := re.FindAllStringSubmatch(message, -1)

		performerName := ""
		if len(matches) > 0 {
			performerName = matches[0][1]
		} else {
			words := strings.Fields(message)
			for i, word := range words {
				wordLower := strings.ToLower(word)
				if (wordLower == "performer" || wordLower == "about") && i+1 < len(words) {
					performerName = words[i+1]
					break
				}
			}
		}

		if performerName == "" {
			return "Please specify a performer name. Try: 'tell me about performer \"Name\"'", nil
		}

		// Get performer details including metadata and scraped threads
		query := `
			SELECT p.id, p.name, p.metadata, p.preview_path, p.thumbnail_path,
				COUNT(DISTINCT vp.video_id) as video_count,
				COUNT(DISTINCT pst.thread_id) as thread_count
			FROM performers p
			LEFT JOIN video_performers vp ON p.id = vp.performer_id
			LEFT JOIN performer_scraped_threads pst ON p.id = pst.performer_id
			WHERE p.name LIKE ?
			GROUP BY p.id
			LIMIT 1
		`

		var id int64
		var name, metadata, previewPath, thumbnailPath string
		var videoCount, threadCount int

		err := s.db.QueryRow(query, "%"+performerName+"%").Scan(
			&id, &name, &metadata, &previewPath, &thumbnailPath,
			&videoCount, &threadCount)

		if err != nil {
			return fmt.Sprintf("I couldn't find information about performer '%s'. Try checking the exact name on the Performers page.", performerName), nil
		}

		var response strings.Builder
		response.WriteString(fmt.Sprintf("👤 **%s**\n\n", name))
		response.WriteString(fmt.Sprintf("📹 Videos: %d\n", videoCount))
		response.WriteString(fmt.Sprintf("🕷️ Scraped Threads: %d\n", threadCount))

		// Parse and display metadata if available
		if metadata != "" && metadata != "{}" {
			var metaMap map[string]interface{}
			if err := json.Unmarshal([]byte(metadata), &metaMap); err == nil {
				if age, ok := metaMap["age"].(float64); ok && age > 0 {
					response.WriteString(fmt.Sprintf("🎂 Age: %.0f\n", age))
				}
				if birthdate, ok := metaMap["birthdate"].(string); ok && birthdate != "" {
					response.WriteString(fmt.Sprintf("📅 Birthdate: %s\n", birthdate))
				}
				if height, ok := metaMap["height"].(string); ok && height != "" {
					response.WriteString(fmt.Sprintf("📏 Height: %s\n", height))
				}
				if breast, ok := metaMap["breast"].(string); ok && breast != "" {
					response.WriteString(fmt.Sprintf("💎 Breast: %s\n", breast))
				}
			}
		}

		if previewPath != "" {
			response.WriteString("✅ Has preview video\n")
		}
		if thumbnailPath != "" {
			response.WriteString("✅ Has thumbnail\n")
		}

		// Get scraped thread titles if any
		if threadCount > 0 {
			threadQuery := `
				SELECT st.title
				FROM scraped_threads st
				JOIN performer_scraped_threads pst ON st.id = pst.thread_id
				WHERE pst.performer_id = ?
				LIMIT 5
			`
			rows, err := s.db.Query(threadQuery, id)
			if err == nil {
				defer rows.Close()
				response.WriteString("\n📋 Recent Scraped Threads:\n")
				count := 0
				for rows.Next() {
					var title string
					if rows.Scan(&title) == nil {
						count++
						response.WriteString(fmt.Sprintf("%d. %s\n", count, title))
					}
				}
				if threadCount > 5 {
					response.WriteString(fmt.Sprintf("... and %d more\n", threadCount-5))
				}
			}
		}

		return response.String(), nil
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

	// Default: try to find anything matching the message as a search
	// First, try as a performer name (direct DB lookup)
	var performerMatch string
	s.db.QueryRow("SELECT name FROM performers WHERE name LIKE ? LIMIT 1", "%"+message+"%").Scan(&performerMatch)
	if performerMatch != "" {
		result, err := s.FindPerformer(performerMatch)
		if err == nil {
			scrapedInfo, _ := s.searchPerformerAcrossAll(performerMatch)
			if scrapedInfo != "" {
				result += "\n" + scrapedInfo
			}
			return result, nil
		}
	}

	// Try as video search
	var videoMatch string
	s.db.QueryRow("SELECT title FROM videos WHERE title LIKE ? LIMIT 1", "%"+message+"%").Scan(&videoMatch)
	if videoMatch != "" {
		results, err := s.SearchVideos(message)
		if err == nil && len(results) > 0 {
			var response strings.Builder
			response.WriteString(fmt.Sprintf("🎬 Found videos matching \"%s\":\n\n", message))
			for _, r := range results {
				response.WriteString(r + "\n")
			}
			return response.String(), nil
		}
	}

	// Default fallback with quick summary
	var videoCount, performerCount int
	s.db.QueryRow("SELECT COUNT(*) FROM videos").Scan(&videoCount)
	s.db.QueryRow("SELECT COUNT(*) FROM performers").Scan(&performerCount)

	return fmt.Sprintf("I'm not sure how to help with that. Here's what I can do:\n\n"+
		"📊 **Your Library:** %d videos, %d performers\n\n"+
		"**Try asking:**\n"+
		"- A performer name (e.g. \"Kendra Lust\")\n"+
		"- \"Give me an update\" for app status\n"+
		"- \"How many videos?\" for stats\n"+
		"- \"Top performers\" for rankings\n"+
		"- \"Scraper summary\" for scraping info\n"+
		"- \"Check for errors\" for system health\n"+
		"- \"Help\" for full capabilities list",
		videoCount, performerCount), nil
}

// ================== Chat History ==================

// SaveChatMessage persists a chat message to the database
func (s *AICompanionService) SaveChatMessage(role, content, source string) {
	_, _ = s.db.Exec(
		`INSERT INTO ai_chat_messages (role, content, source, created_at) VALUES (?, ?, ?, ?)`,
		role, content, source, time.Now(),
	)
	// Prune: keep only the 500 most recent messages
	_, _ = s.db.Exec(`DELETE FROM ai_chat_messages WHERE id NOT IN (
		SELECT id FROM ai_chat_messages ORDER BY created_at DESC LIMIT 500
	)`)
}

// GetChatHistory returns the most recent N messages, oldest-first
func (s *AICompanionService) GetChatHistory(limit int) ([]map[string]interface{}, error) {
	rows, err := s.db.Query(
		`SELECT id, role, content, source, created_at FROM ai_chat_messages
		 ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []map[string]interface{}
	for rows.Next() {
		var id int64
		var role, content, source, createdAt string
		if rows.Scan(&id, &role, &content, &source, &createdAt) == nil {
			results = append(results, map[string]interface{}{
				"id": id, "role": role, "content": content,
				"source": source, "created_at": createdAt,
			})
		}
	}
	// Reverse to oldest-first order
	for i, j := 0, len(results)-1; i < j; i, j = i+1, j-1 {
		results[i], results[j] = results[j], results[i]
	}
	return results, nil
}

// ClearChatHistory deletes all persisted chat messages
func (s *AICompanionService) ClearChatHistory() error {
	_, err := s.db.Exec("DELETE FROM ai_chat_messages")
	return err
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
