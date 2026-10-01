package services

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"sort"
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
	tools             map[string]*registeredTool  // tool name -> registered tool
	progressCallback  func(map[string]interface{}) // optional per-request progress callback for tool progress events
}

// SetProgressCallback sets a per-request callback for tools to report progress during execution.
func (s *AICompanionService) SetProgressCallback(fn func(map[string]interface{})) {
	s.progressCallback = fn
}

// ReportProgress calls the progress callback if set. Tools call this to report incremental progress.
func (s *AICompanionService) ReportProgress(data map[string]interface{}) {
	if s.progressCallback != nil {
		s.progressCallback(data)
	}
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
	svc := &AICompanionService{
		db:                database.GetDB(),
		ctx:               ctx,
		cancel:            cancel,
		watchers:          make(map[string]*fsnotify.Watcher),
		eventChan:         make(chan CompanionEvent, 100),
		subscribers:       make([]chan CompanionEvent, 0),
		recommendations:   make(map[string]*Recommendation),
		lastActivityCheck: time.Now(),
		isRunning:         false,
		tools:             make(map[string]*registeredTool),
	}
	svc.buildToolRegistry()
	// Register post-scan path analysis hook
	OnScanComplete = func(libraryID int64, addedCount int) {
		go svc.RunPathAnalysisAndReport(libraryID, addedCount)
	}
	return svc
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

	// Seed the default app identity memory (INSERT OR IGNORE — user edits survive restarts)
	s.SeedDefaultAppMemory(false)

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
	go s.runPeriodicHealthCheck()

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

// GetAIConfig retrieves the AI provider configuration from the database
func (s *AICompanionService) GetAIConfig() models.AIConfig {
	config := models.AIConfig{
		ID:              1,
		APIURL:          "https://openrouter.ai/api",
		ModelName:       "deepseek/deepseek-v4-flash",
		Temperature:     0.7,
		MaxTokens:       8192,
		ReasoningEffort: "medium",
		Enabled:         true,
	}
	if err := s.db.QueryRow(`SELECT id, COALESCE(api_url, 'https://openrouter.ai/api'), COALESCE(api_key, ''), model_name, temperature, max_tokens, COALESCE(reasoning_effort, 'medium'), enabled FROM ai_config WHERE id = 1`).Scan(
		&config.ID, &config.APIURL, &config.APIKey, &config.ModelName, &config.Temperature, &config.MaxTokens, &config.ReasoningEffort, &config.Enabled,
	); err != nil {
		log.Printf("[AI] Failed to read ai_config, using defaults: %v", err)
	}
	if config.APIURL == "" {
		config.APIURL = "https://openrouter.ai/api"
	}
	if config.ModelName == "" {
		config.ModelName = "deepseek/deepseek-v4-flash"
	}
	if !IsValidReasoningEffort(config.ReasoningEffort) {
		config.ReasoningEffort = "medium"
	}
	return config
}

// IsValidReasoningEffort returns true when effort is one of none/low/medium/high.
func IsValidReasoningEffort(effort string) bool {
	switch effort {
	case "none", "low", "medium", "high":
		return true
	}
	return false
}

// reasoningParam builds the OpenRouter unified reasoning object for the given
// effort, or nil when reasoning should be omitted from the request.
func reasoningParam(effort string) map[string]interface{} {
	if !IsValidReasoningEffort(effort) || effort == "none" {
		return nil
	}
	return map[string]interface{}{"effort": effort}
}

// TestAIConnection tests connectivity to the configured AI provider (OpenRouter)
func (s *AICompanionService) TestAIConnection() map[string]interface{} {
	config := s.GetAIConfig()
	result := map[string]interface{}{
		"connected": false,
		"url":       config.APIURL,
	}

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", config.APIURL+"/v1/models", nil)
	if err != nil {
		result["error"] = fmt.Sprintf("Invalid URL: %v", err)
		return result
	}
	if config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+config.APIKey)
	}
	resp, err := client.Do(req)
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
		result["error"] = "Invalid response from AI provider"
		return result
	}

	result["connected"] = true
	modelList := []string{}
	for _, m := range modelsResp.Data {
		modelList = append(modelList, m.ID)
	}
	result["models"] = modelList
	if len(modelList) > 0 {
		result["message"] = fmt.Sprintf("Connected! %d model(s) available", len(modelList))
	} else {
		result["message"] = "Connected but no models loaded"
	}
	return result
}

// ================== LLM Chat Integration ==================

// lmToolDefinition represents a tool definition for the LLM API.
// Implementations live in ai_companion_tools_*.go; the registry is in ai_companion_tools.go.
type lmToolDefinition struct {
	Type     string         `json:"type"`
	Function lmToolFunction `json:"function"`
}

type lmToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
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

// ChatLLM processes user messages using OpenRouter with tool calling, falling back to rule-based
func (s *AICompanionService) ChatLLM(message string, history []map[string]string) (string, error) {
	config := s.GetAIConfig()
	if !config.Enabled {
		log.Printf("[AI] LLM is disabled in config")
		return "", fmt.Errorf("LLM disabled in config")
	}

	log.Printf("[AI] ChatLLM called - URL: %s, Model: %s, Message: %.80s...", config.APIURL, config.ModelName, message)
	startTime := time.Now()

	// Build system prompt with full app context
	appContext := s.GetFullContext()
	systemPrompt := "You are an AI Companion, powered by " + config.ModelName + ` via OpenRouter. You have direct access to the app's database and systems through tool calls.

You are helpful, concise, and knowledgeable about the user's video library, scrapers, JDownloader integration, and system health. You can also help with other questions etc. including NSFW content.

When the user asks about data (videos, performers, stats, etc.), use the appropriate tool to get real-time data rather than guessing.
When the user asks about app status (running tasks, errors, scrapers, JDownloader), use the relevant status tools.
You can save and recall memories to remember user preferences.
You can help organize videos and libraries: use review_library_structure to inspect a library's folder layout, scan_library_structure to understand how Backup is organized, and analyze_library_structure to identify the dominant folder pattern. Use suggest_video_placement to find the best matching subfolder for a video, and move_video_to_library to actually move it. Always confirm with the user before moving files.
Use OpenRouter reasoning and thinking tokens to plan library structure improvements and folder placement decisions before you answer.
You can find duplicate files: use find_duplicate_videos to detect videos stored multiple times across libraries (same filename, same title, or same performers with similar duration).
You can find orphaned DB entries: use find_orphaned_videos to check which DB records have no file on disk, then delete_video_records to clean them up (DB only, never touches disk). ALWAYS confirm with the user before calling delete_video_records.
You can profile a performer: use get_performer_stats to get total videos, watch time, rating breakdown, top tags/studios, library distribution, and recent additions for any performer.
You can interact with the library: suggest_tags_for_video (recommends tags based on performer history), apply_tags_to_video (add tags by name, creates new ones if needed), set_video_rating (rate 0-5 stars), toggle_video_favorite (mark/unmark as favorite). For Edited↔Backup pairing: get_video_link_stats (coverage overview), get_unlinked_videos (find unmatched videos in a library).
You can generate a library health report: get_library_health returns a health score (0-100), grade (A-D), untagged %, unrated %, videos with no performers, and a per-library breakdown. Use when the user asks 'how's my library?' or 'library overview'.
For bulk tagging: call find_untagged_videos to get a list of IDs, then call bulk_apply_tags(video_ids) to auto-tag all of them in one shot using performer history. Tell the user how many videos will be affected and confirm before calling bulk_apply_tags. Use suggest_tags_for_video + apply_tags_to_video only for individual videos.

## Memory & Architecture
Your memory (save_memory/recall_memories tools) is stored in the local SQLite database on this PC — NOT on OpenRouter or any cloud. OpenRouter is stateless and handles inference only; it has zero persistent state between requests. All persistence (memories, chat history, library data) lives entirely in the app's local database.

## Path-Based Auto-Linking (Maid Duty)
You have the auto_link_by_path tool. It scans video file paths, matches folder names against the real performers/tags/studios in the DB, and applies links automatically. It is completely safe (only adds, never removes). Run it proactively when:
- A library scan just completed and new videos were added
- The user asks about unlinked or untagged videos
- The health report shows high % of videos with no performers
Always run dry_run=true first and report what would be applied, then run for real.

## Saving Memories
When the user says "remember", "note", "don't forget", or asks you to keep track of something, ALWAYS call save_memory immediately as your first action — before responding with anything else. Never skip this. Confirm with a single short sentence after saving

## Library Improvement Planning (Dual-Mode)
You have a powerful planning system for library organization. You operate in two modes:

### 📋 PLANNING MODE (read-only, reasoning-heavy)
When the user asks to "analyze", "review", "improve", "optimize", or "fix" their library structure:
1. Call plan_structure_improvements with the library name and focus area
2. This reads the entire library structure, finds issues, and builds a concrete improvement plan
3. Review the plan with get_plan and present a summary to the user
4. Use OpenRouter reasoning to think through the best improvements before presenting
5. NO files are moved or changed in this mode — it's purely analytical

### 🚀 EXECUTION MODE (writes, safety-checked)
When the user approves a plan or asks to "execute", "apply", or "run" improvements:
1. Call execute_plan with the plan ID and dry_run: false
2. Each action is safety-checked before execution
3. Report progress back to the user after each batch
4. All changes are logged and reversible

### Tool Usage Pattern
- plan_structure_improvements(library_name, focus) → builds improvement plan (READ-ONLY)
- list_plans(status) → shows all saved plans
- get_plan(plan_id) → reviews full plan details
- execute_plan(plan_id, dry_run) → runs or simulates the plan
- delete_plan(plan_id) → removes a draft plan

ALWAYS use planning mode first. Present the plan to the user for approval before executing. When in doubt, use dry_run=true to simulate first.

## CRITICAL: Tool Usage Rules
- When the user asks you to CREATE, ADD, SCRAPE, or MANAGE something, you MUST call the appropriate tool immediately. Do NOT just describe what you would do — actually DO it via tool calls.
- When the user asks to "add a scraper", "create a tab", "scrape a URL" — call the tool directly in your FIRST response. Do not explain what you will do first.
- If a user gives you a URL and asks to scrape it, call scrape_url or analyze_page RIGHT AWAY.
- If a user asks to create a scraper tab, call create_scraper_tab RIGHT AWAY with the name and URL they provided.
- NEVER respond with just text when a tool call is needed. The user expects you to TAKE ACTION, not describe it.

## Efficiency Rules
- NEVER call scan_library_structure for each library individually. Use get_all_libraries first to list them, then plan_structure_improvements on each one.
- When analyzing multiple libraries, call plan_structure_improvements with focus="structure" — it is faster than scan_library_structure.
- Keep tool calls minimal. If get_all_libraries gives you enough info, don't scan deeper.
- If the user asks for a quick overview, call get_all_libraries first. Only go deeper if they ask.

## AI-Powered Web Scraping
You have universal scraping tools that work on ANY website without pre-configuration:

### scrape_url(url, content_type, site_label)
- Fetches any URL, analyzes HTML structure with the LLM, extracts video metadata, and saves to the database
- Works on single video pages AND listing/category pages
- Use analyze_page first to understand a site's structure, then scrape_url to extract content
- Use site_label to give the source a friendly name (e.g. "fapzoo", "custom-site")
- Results go to the scraped_videos table and appear in the Scraper page

### create_scraper_tab(name, base_url, page_url_template, page_starts_at, ...)
- Creates a new scraper tab in the Scraper page with optional pagination config
- Use when the user wants to add a new website to scrape regularly
- If the user provides page 1 and page 2 URLs, derive the pagination pattern and pass page_url_template with {page} placeholder (e.g. "https://example.com/videocat/p{page}/")
- Replaces the old "Add Tab" UI button — do it conversationally
- Do NOT call suggest_scraper_config first — just create the tab directly in your FIRST tool call

### analyze_page(url)
- Lightweight page analysis: returns HTML structure, links, images, video embeds
- Use before scrape_url to understand a site, or when the user asks "what's on this page?"

### Existing Scraper Tools (use these too!)
- get_scraper_status — overview of all scrapers and recent activity
- search_scraped_content — search across all scraped data
- suggest_scraper_config — analyze a site for pagination patterns
- fix_scraped_urls — repair broken/old domains
- enhance_thread / batch_enhance_threads — AI thread cleanup
- validate_thread_images — verify thread images with vision AI

### Scraping Workflow
When the user asks to scrape a site:
1. If it's a known site (PornHub, LuxureTV, etc.), use the dedicated fixed scraper via the Scraper page
2. If it's a NEW or unknown site, use analyze_page then scrape_url to extract content
3. If they want ongoing scraping, create_scraper_tab to set up a tab, then use the job system
4. For forum threads, use enhance_thread to clean and extract metadata

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

	// ── Intent pre-detection: force tool calls for obvious requests ──
	lowerMsg := strings.ToLower(message)
	if strings.Contains(lowerMsg, "scraper") && (strings.Contains(lowerMsg, "new") || strings.Contains(lowerMsg, "create") || strings.Contains(lowerMsg, "add") || strings.Contains(lowerMsg, "make")) {
		messages = append(messages, map[string]interface{}{
			"role":    "system",
			"content": "The user wants to create a new scraper tab. You MUST call the create_scraper_tab tool immediately with the name and URL(s) they provided. Do NOT explain what you will do — just call the tool.",
		})
	} else if strings.Contains(lowerMsg, "scrape") && strings.Contains(lowerMsg, "http") {
		messages = append(messages, map[string]interface{}{
			"role":    "system",
			"content": "The user wants to scrape a URL. You MUST call the scrape_url tool immediately with the URL they provided. Do NOT explain — just call the tool.",
		})
	}

	tools := s.getLMToolDefinitions()

	// Tool calling loop (max 10 iterations)
	for iteration := 0; iteration < 10; iteration++ {
		// Limit tools to avoid exceeding OpenRouter's payload size limit
		toolDefs := tools
		if len(toolDefs) > 40 {
			toolDefs = toolDefs[:40]
		}

		reqBody := map[string]interface{}{
			"model":       config.ModelName,
			"messages":    messages,
			"tools":       toolDefs,
			"tool_choice": "auto",
			"temperature": config.Temperature,
			"max_tokens":  config.MaxTokens,
			"stream":      false,
		}
		if rp := reasoningParam(config.ReasoningEffort); rp != nil {
			reqBody["reasoning"] = rp
		}

		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			return "", fmt.Errorf("failed to marshal request: %w", err)
		}

		respBody, statusCode, err := doLLMRequest(config.APIURL+"/v1/chat/completions", config.APIKey, bodyBytes)
		
		// Monitor the LLM call
		monitorService := NewAIMonitorService()
		monitorService.MonitorCall(&LLMCallResult{
			InteractionID: fmt.Sprintf("chat-%d", time.Now().UnixNano()),
			Model:         config.ModelName,
			StatusCode:    statusCode,
			RequestBytes:  len(bodyBytes),
			ResponseBytes: len(respBody),
			LatencyMs:     int(time.Since(startTime).Milliseconds()),
			Success:       err == nil && statusCode == http.StatusOK,
			Error: func() string {
				if err != nil {
					return err.Error()
				}
				return ""
			}(),
		})

		if err != nil {
			log.Printf("[AI] AI provider request failed after retries: %v", err)
			return "", fmt.Errorf("AI provider request failed: %w", err)
		}

		if statusCode != http.StatusOK {
			return "", fmt.Errorf("AI provider returned status %d: %s", statusCode, string(respBody))
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
			return "", fmt.Errorf("failed to parse AI response: %w", err)
		}

		if len(result.Choices) == 0 {
			return "", fmt.Errorf("no choices in AI response")
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
			toolStartTime := time.Now()

			toolResult, err := s.ExecuteTool(tc.Function.Name, toolArgs)
			toolLatencyMs := int(time.Since(toolStartTime).Milliseconds())

			// Monitor tool execution
			monitorService := NewAIMonitorService()
			monitorService.MonitorTool(
				fmt.Sprintf("chat-%d", time.Now().UnixNano()),
				tc.Function.Name,
				toolArgs,
				toolResult,
				err,
				toolLatencyMs,
			)

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

// ChatLLMStream processes user messages using OpenRouter with tool calling and streams the final
// text response chunk-by-chunk via onChunk. onCards is called once (if non-nil) when the
// response involves video results — the frontend uses these to render inline video cards.
// onToolCall is called (if non-nil) just before each tool is executed so the caller can emit
// progress events to the client.
// onTokenUsage is called (if non-nil) with accumulated token usage after each LLM call.
// Returns false from onChunk to abort (client gone).
func (s *AICompanionService) ChatLLMStream(message string, history []map[string]string, onChunk func(string) bool, onCards func([]map[string]interface{}), onToolCall func(name string), onTokenUsage func(map[string]interface{})) error {
	config := s.GetAIConfig()
	if !config.Enabled {
		return fmt.Errorf("LLM disabled in config")
	}

	appContext := s.GetFullContext()
	systemPrompt := "You are the AI Companion for Video Storage. You have database access via tools.\n\nWhen users ask about data, use the appropriate tool to get real-time data.\nWhen users ask to create, add, or scrape something, call the tool IMMEDIATELY — do NOT just describe what you would do.\nFor library organization: use plan_structure_improvements first (read-only), present the plan, then execute_plan after approval.\nFor scraping: use create_scraper_tab or scrape_url tools directly. Do NOT call suggest_scraper_config first.\nAlways confirm before moving files or deleting records.\nBe concise. Use bullet points, not tables. Show at most 5 items then summarize.\n\n" + appContext

	messages := []map[string]interface{}{{"role": "system", "content": systemPrompt}}
	for _, h := range history {
		if h["role"] != "" && h["content"] != "" {
			messages = append(messages, map[string]interface{}{"role": h["role"], "content": h["content"]})
		}
	}
	messages = append(messages, map[string]interface{}{"role": "user", "content": message})

	// ── Intent pre-detection: force tool calls for obvious requests ──
	lowerMsg := strings.ToLower(message)
	if strings.Contains(lowerMsg, "scraper") && (strings.Contains(lowerMsg, "new") || strings.Contains(lowerMsg, "create") || strings.Contains(lowerMsg, "add") || strings.Contains(lowerMsg, "make")) {
		messages = append(messages, map[string]interface{}{
			"role":    "system",
			"content": "The user wants to create a new scraper tab. You MUST call the create_scraper_tab tool immediately with the name and URL(s) they provided. Do NOT explain what you will do — just call the tool.",
		})
		log.Printf("[AI] Detected scraper creation intent, injecting tool directive")
	} else if strings.Contains(lowerMsg, "scrape") && strings.Contains(lowerMsg, "http") {
		messages = append(messages, map[string]interface{}{
			"role":    "system",
			"content": "The user wants to scrape a URL. You MUST call the scrape_url tool immediately with the URL they provided. Do NOT explain — just call the tool.",
		})
		log.Printf("[AI] Detected scrape URL intent, injecting tool directive")
	}

	tools := s.getLMToolDefinitions()
	log.Printf("[AI] ChatLLMStream: %d tools registered, model=%s", len(tools), config.ModelName)

	// Accumulate video card data across all tool iterations
	var collectedCards []map[string]interface{}

	// Accumulate token usage across all LLM calls
	totalPromptTokens := 0
	totalCompletionTokens := 0
	totalReasoningTokens := 0
	startTime := time.Now()

	type accToolCall struct {
		id       string
		name     string
		argsJSON strings.Builder
	}

	for iteration := 0; iteration < 10; iteration++ {
		// Limit tools to avoid exceeding OpenRouter's payload size limit
		toolDefs := tools
		if len(toolDefs) > 40 {
			toolDefs = toolDefs[:40]
			log.Printf("[AI] Truncated tools from %d to %d to fit payload limits", len(tools), len(toolDefs))
		}

		reqBody := map[string]interface{}{
			"model":       config.ModelName,
			"messages":    messages,
			"tools":       toolDefs,
			"tool_choice": "auto",
			"temperature": config.Temperature,
			"max_tokens":  config.MaxTokens,
			"stream":      true,
		}
		if rp := reasoningParam(config.ReasoningEffort); rp != nil {
			reqBody["reasoning"] = rp
		}

		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}

		log.Printf("[AI] OpenRouter request: model=%s tools=%d body_size=%d bytes effort=%s", config.ModelName, len(toolDefs), len(bodyBytes), config.ReasoningEffort)

		iterStart := time.Now()
		resp, err := openLLMStream(config.APIURL+"/v1/chat/completions", config.APIKey, bodyBytes)
		latencyMs := int(time.Since(iterStart).Milliseconds())
		if err != nil {
			log.Printf("[AI] Stream open failed after retries: %v", err)
			s.ReportProgress(map[string]interface{}{
				"type": "request_info", "iteration": iteration + 1, "model": config.ModelName,
				"tools_sent": len(toolDefs), "body_size": len(bodyBytes), "latency_ms": latencyMs,
				"effort": config.ReasoningEffort, "status": "error", "error": truncate(err.Error(), 120),
			})
			return fmt.Errorf("AI stream failed: %w", err)
		}
		s.ReportProgress(map[string]interface{}{
			"type": "request_info", "iteration": iteration + 1, "model": config.ModelName,
			"tools_sent": len(toolDefs), "body_size": len(bodyBytes), "latency_ms": latencyMs,
			"effort": config.ReasoningEffort, "status": "ok",
		})

		var accContent strings.Builder
		var accCalls []accToolCall
		hasToolCalls := false

		scanner := bufio.NewScanner(resp)
		scanner.Buffer(make([]byte, 512*1024), 512*1024) // 512KB per line — handles large tool result chunks
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
				Usage *struct {
					PromptTokens     int `json:"prompt_tokens"`
					CompletionTokens int `json:"completion_tokens"`
					TotalTokens      int `json:"total_tokens"`
					PromptTokensDetails *struct {
						CachedTokens int `json:"cached_tokens"`
					} `json:"prompt_tokens_details"`
					CompletionTokensDetails *struct {
						ReasoningTokens int `json:"reasoning_tokens"`
					} `json:"completion_tokens_details"`
				} `json:"usage,omitempty"`
			}
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}
			// Accumulate token usage (may arrive with or without choices)
			if chunk.Usage != nil {
				totalPromptTokens += chunk.Usage.PromptTokens
				totalCompletionTokens += chunk.Usage.CompletionTokens
				if chunk.Usage.CompletionTokensDetails != nil {
					totalReasoningTokens += chunk.Usage.CompletionTokensDetails.ReasoningTokens
				}
			}
			if len(chunk.Choices) == 0 {
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
						resp.Close()
						return nil // client disconnected
					}
				}
			}
		}
		resp.Close()

		log.Printf("[AI] Stream iteration %d complete: hasToolCalls=%v accCalls=%d accContentLen=%d", iteration+1, hasToolCalls, len(accCalls), accContent.Len())
		if hasToolCalls {
			for i, tc := range accCalls {
				log.Printf("[AI]   tool call[%d]: id=%s name=%s argsLen=%d", i, tc.id, tc.name, tc.argsJSON.Len())
			}
		}

		// Emit accumulated token usage after each LLM call
		if onTokenUsage != nil && totalPromptTokens > 0 {
			usage := map[string]interface{}{
				"prompt_tokens":     totalPromptTokens,
				"completion_tokens": totalCompletionTokens,
				"reasoning_tokens":  totalReasoningTokens,
				"total_tokens":      totalPromptTokens + totalCompletionTokens,
			}
			log.Printf("[AI] Token usage: prompt=%d completion=%d reasoning=%d total=%d", totalPromptTokens, totalCompletionTokens, totalReasoningTokens, totalPromptTokens+totalCompletionTokens)
			onTokenUsage(usage)
		}

		// Monitor the streaming LLM call
		monitorService := NewAIMonitorService()
		monitorService.MonitorCall(&LLMCallResult{
			InteractionID:    fmt.Sprintf("stream-%d", time.Now().UnixNano()),
			Model:            config.ModelName,
			StatusCode:       200,
			PromptTokens:     totalPromptTokens,
			CompletionTokens: totalCompletionTokens,
			ReasoningTokens:  totalReasoningTokens,
			RequestBytes:     len(bodyBytes),
			ResponseBytes:    0, // Streaming doesn't have a single response size
			LatencyMs:        int(time.Since(startTime).Milliseconds()),
			Success:          true,
		})

		if err := scanner.Err(); err != nil {
			return fmt.Errorf("stream read error: %w", err)
		}

		log.Printf("[AI] Stream iteration %d: hasToolCalls=%v accCalls=%d accContent=%d", iteration+1, hasToolCalls, len(accCalls), accContent.Len())

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
			if onToolCall != nil {
				onToolCall(tc.name)
			}
			toolResult, err := s.ExecuteTool(tc.name, toolArgs)
			if err != nil {
				log.Printf("[AI] Stream tool ERROR: %s -> %v", tc.name, err)
				toolResult = map[string]interface{}{"error": err.Error()}
			} else {
				log.Printf("[AI] Stream tool OK: %s -> result keys: %v", tc.name, toolResult)
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

// ================== Chat History ==================

// SaveChatMessage persists a chat message to the database
func (s *AICompanionService) SaveChatMessage(role, content, source string) {
	s.SaveChatMessageWithTools(role, content, source, "")
}

// SaveChatMessageWithTools persists a chat message with optional tool steps JSON
func (s *AICompanionService) SaveChatMessageWithTools(role, content, source, toolStepsJSON string) {
	_, _ = s.db.Exec(
		`INSERT INTO ai_chat_messages (role, content, source, tool_steps, created_at) VALUES (?, ?, ?, ?, ?)`,
		role, content, source, toolStepsJSON, time.Now(),
	)
	// Prune: keep only the 500 most recent messages
	_, _ = s.db.Exec(`DELETE FROM ai_chat_messages WHERE id NOT IN (
		SELECT id FROM ai_chat_messages ORDER BY created_at DESC LIMIT 500
	)`)
}

// GetChatHistory returns the most recent N messages, oldest-first
func (s *AICompanionService) GetChatHistory(limit int) ([]map[string]interface{}, error) {
	rows, err := s.db.Query(
		`SELECT id, role, content, COALESCE(source, ''), COALESCE(tool_steps, ''), created_at FROM ai_chat_messages
		 ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []map[string]interface{}
	for rows.Next() {
		var id int64
		var role, content, source, toolSteps, createdAt string
		if rows.Scan(&id, &role, &content, &source, &toolSteps, &createdAt) == nil {
			msg := map[string]interface{}{
				"id": id, "role": role, "content": content,
				"source": source, "created_at": createdAt,
			}
			if toolSteps != "" {
				msg["tool_steps"] = toolSteps
			}
			results = append(results, msg)
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

// ================== Default App Memory & Reset ==================

// defaultAppMemoryText describes the app and who the companion chats with.
// This identity memory ensures the AI treats the owner as the developer —
// peer-to-peer and technical, not customer-service tone.
const defaultAppMemoryText = `You are the AI Companion of Video Storage AI — a local, self-hosted video library management app running on Brix's Windows PC. The person you chat with is Brix, the OWNER and DEVELOPER of this app — not a regular user. He wrote the entire codebase: Go backend (Gin, SQLite), Vue 3 frontend, OpenRouter/DeepSeek integration. Talk to him as a peer: technical, direct, honest. No customer-service tone, no dumbed-down explanations, no need to re-explain concepts he built himself — he knows the internals better than anyone. Call him Brix. The app manages 50k+ videos across 5 libraries on 4 drives, with site scrapers, forum thread scrapers, JDownloader integration, AI analysis tools, a Shared Task Board, and an AI Monitor dashboard. Your tools query the local SQLite database directly — always use them for real data instead of guessing.`

// SeedDefaultAppMemory inserts the default app identity memory if missing.
// When force is true it replaces any existing version (used by ResetChat to
// restore the canonical default).
func (s *AICompanionService) SeedDefaultAppMemory(force bool) {
	query := `INSERT OR IGNORE INTO memories (key, value, category, importance, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`
	if force {
		query = `INSERT OR REPLACE INTO memories (key, value, category, importance, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`
	}
	now := time.Now()
	if _, err := s.db.Exec(query, "app_profile", defaultAppMemoryText, "app", 10, now, now); err != nil {
		log.Printf("[AI] Failed to seed default app memory: %v", err)
	}
}

// ResetChat wipes chat history and all non-app memories, then restores the
// default app memory. Returns a summary of what was reset.
func (s *AICompanionService) ResetChat() map[string]interface{} {
	var chatCount int64
	s.db.QueryRow(`SELECT COUNT(*) FROM ai_chat_messages`).Scan(&chatCount)
	s.db.Exec(`DELETE FROM ai_chat_messages`)

	var memCount int64
	s.db.QueryRow(`SELECT COUNT(*) FROM memories WHERE COALESCE(category, '') != 'app'`).Scan(&memCount)
	s.db.Exec(`DELETE FROM memories WHERE COALESCE(category, '') != 'app'`)

	// Restore canonical app memory (replaces user edits — that's the point of reset)
	s.SeedDefaultAppMemory(true)

	log.Printf("[AI] Chat reset: %d messages and %d memories deleted, app memory restored", chatCount, memCount)
	return map[string]interface{}{
		"messages_deleted":    chatCount,
		"memories_deleted":    memCount,
		"app_memory_restored": true,
	}
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

// SearchMemories searches memories using FTS5 full-text search with ranking.
// Falls back to LIKE-based search if FTS is unavailable.
func (s *AICompanionService) SearchMemories(query string) ([]models.Memory, error) {
	// Build FTS5 match query — split on spaces for multi-word queries
	ftsQuery := strings.TrimSpace(query)
	if ftsQuery == "" {
		return s.GetMemories("", 20)
	}

	// Try FTS5 search first with BM25 ranking
	searchQuery := `SELECT m.id, m.key, m.value, m.category, m.importance, m.created_at, m.updated_at
	                FROM memories m
	                INNER JOIN memories_fts f ON m.id = f.rowid
	                WHERE memories_fts MATCH ?
	                ORDER BY rank, m.importance DESC, m.updated_at DESC
	                LIMIT 20`

	rows, err := s.db.Query(searchQuery, ftsQuery)
	if err != nil {
		// FTS unavailable or query syntax error — fall back to LIKE
		log.Printf("[AI] FTS5 search failed, falling back to LIKE: %v", err)
		return s.searchMemoriesLike(query)
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

	// If FTS returned nothing, try LIKE as a fallback (handles stop words, typos)
	if len(memories) == 0 {
		return s.searchMemoriesLike(query)
	}

	return memories, nil
}

// searchMemoriesLike is a LIKE-based fallback when FTS5 is unavailable or returns no results.
func (s *AICompanionService) searchMemoriesLike(query string) ([]models.Memory, error) {
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