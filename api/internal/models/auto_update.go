package models

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ScraperAutoUpdateConfig stores auto-update configuration for each scraper
type ScraperAutoUpdateConfig struct {
	ID              int64      `json:"id"`
	ScraperType     string     `json:"scraper_type"`
	Enabled         bool       `json:"enabled"`
	IntervalMinutes int        `json:"interval_minutes"`
	MaxPagesToCheck int        `json:"max_pages_to_check"`
	LastCheckedAt   *time.Time `json:"last_checked_at"`
	LastFoundCount  int        `json:"last_found_count"`
	TotalFoundCount int        `json:"total_found_count"`
	CheckCount      int        `json:"check_count"`
	ConfigJSON      string     `json:"config_json"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`

	// Parsed config (not stored directly)
	Config *ScraperConfig `json:"config,omitempty"`
}

// ScraperConfig holds scraper-specific configuration
type ScraperConfig struct {
	// For PornHub
	Keywords []string `json:"keywords,omitempty"`

	// For LuxureTV/Sexuria
	Categories []string `json:"categories,omitempty"`

	// For SimpCity - forum URLs to monitor
	URLs []string `json:"urls,omitempty"`

	// General options
	SortBy string `json:"sort_by,omitempty"`
}

// ParseConfig parses the JSON config into the Config struct
func (s *ScraperAutoUpdateConfig) ParseConfig() error {
	if s.ConfigJSON == "" || s.ConfigJSON == "{}" {
		s.Config = &ScraperConfig{}
		return nil
	}
	s.Config = &ScraperConfig{}
	return json.Unmarshal([]byte(s.ConfigJSON), s.Config)
}

// MarshalConfig converts Config struct back to JSON string
func (s *ScraperAutoUpdateConfig) MarshalConfig() error {
	if s.Config == nil {
		s.ConfigJSON = "{}"
		return nil
	}
	data, err := json.Marshal(s.Config)
	if err != nil {
		return err
	}
	s.ConfigJSON = string(data)
	return nil
}

// ScraperCheckpoint tracks what content has been seen to detect new items
type ScraperCheckpoint struct {
	ID              int64     `json:"id"`
	ScraperType     string    `json:"scraper_type"`
	CheckpointType  string    `json:"checkpoint_type"` // 'last_video_url', 'last_post_id', 'last_external_id', 'last_reply_count'
	CheckpointValue string    `json:"checkpoint_value"`
	TargetID        string    `json:"target_id,omitempty"` // keyword, category, thread_id
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// DashboardFeedItem represents an item in the dashboard activity feed
type DashboardFeedItem struct {
	ID          int64     `json:"id"`
	ItemType    string    `json:"item_type"` // 'new_content', 'scrape_completed', 'error', 'system'
	Title       string    `json:"title"`
	Message     string    `json:"message"`
	Icon        string    `json:"icon"`
	Color       string    `json:"color"` // 'primary', 'success', 'warning', 'danger', 'info'
	ScraperType string    `json:"scraper_type,omitempty"`
	Metadata    string    `json:"metadata"`
	CreatedAt   time.Time `json:"created_at"`
}

// NewContentResult represents the result of checking for new content
type NewContentResult struct {
	ScraperType   string                 `json:"scraper_type"`
	NewItemsFound int                    `json:"new_items_found"`
	CheckedAt     time.Time              `json:"checked_at"`
	Details       map[string]interface{} `json:"details,omitempty"`
	NewItems      []NewItemInfo          `json:"new_items,omitempty"`
}

// NewItemInfo contains info about a newly found item
type NewItemInfo struct {
	ID        int64  `json:"id,omitempty"`  // Internal DB ID (for threads)
	URL       string `json:"url"`
	Title     string `json:"title"`
	Thumbnail string `json:"thumbnail,omitempty"`
	Type      string `json:"type"`     // 'video', 'thread', 'post', 'item'
	Duration  int    `json:"duration"` // Duration in seconds (for videos)
	Views     int    `json:"views"`    // View count (for videos)
}

// ScraperStatusCard represents the status of a scraper for dashboard display
type ScraperStatusCard struct {
	ScraperType      string     `json:"scraper_type"`
	DisplayName      string     `json:"display_name"`
	Icon             string     `json:"icon"`
	Color            string     `json:"color"`
	Enabled          bool       `json:"enabled"`
	LastCheckedAt    *time.Time `json:"last_checked_at"`
	LastFoundCount   int        `json:"last_found_count"`
	TotalFoundCount  int        `json:"total_found_count"`
	NextScheduledRun *time.Time `json:"next_scheduled_run"`
	TotalItems       int        `json:"total_items"`
	Status           string     `json:"status"` // 'idle', 'running', 'error', 'disabled'
	IntervalMinutes  int        `json:"interval_minutes"`
}

// DashboardStats holds aggregated statistics for the dashboard
type DashboardStats struct {
	TotalVideos             int `json:"total_videos"`
	TotalScrapedVideos      int `json:"total_scraped_videos"`
	TotalDownloadedScraped  int `json:"total_downloaded_scraped"`  // Scraped videos with downloaded_at set
	TotalThreads            int `json:"total_threads"`
	TotalPerformers         int `json:"total_performers"`
	WatchedThreads          int `json:"watched_threads"`
	RunningTasks            int `json:"running_tasks"`
	PendingTasks            int `json:"pending_tasks"`
	RecentNewItems          int `json:"recent_new_items"` // New items in last 24h
	VideosWithoutPerformer  int `json:"videos_without_performer"`
}

// AutoUpdateConfigUpdate represents an update request for auto-update config
type AutoUpdateConfigUpdate struct {
	Enabled         *bool   `json:"enabled,omitempty"`
	IntervalMinutes *int    `json:"interval_minutes,omitempty"`
	MaxPagesToCheck *int    `json:"max_pages_to_check,omitempty"`
	ConfigJSON      *string `json:"config_json,omitempty"`
}

// GetScraperDisplayInfo returns display information for a scraper type
func GetScraperDisplayInfo(scraperType string) (displayName, icon, color string) {
	switch scraperType {
	case "simpcity":
		return "SimpcityIS", "comments", "purple"
	case "pornhub":
		return "PornHub", "play-circle", "orange"
	case "rule34video":
		return "Rule34Video", "cube", "pink"
	case "luxuretv":
		return "LuxureTV", "film", "red"
	case "sexuria":
		return "Sexuria", "download", "blue"
	case "motherless":
		return "Motherless", "globe", "amber"
	case "leakvids":
		return "LeakVids", "film", "red"
	default:
		return scraperType, "globe", "gray"
	}
}

// GetScraperDisplayInfoWithDB returns display info, resolving custom_* types from the DB
func GetScraperDisplayInfoWithDB(db *sql.DB, scraperType string) (displayName, icon, color string) {
	// Try built-in scrapers first
	displayName, icon, color = GetScraperDisplayInfo(scraperType)

	// If it's a custom tab, look up the actual name from the database
	if strings.HasPrefix(scraperType, "custom_") {
		tabIDStr := strings.TrimPrefix(scraperType, "custom_")
		tabID, err := strconv.ParseInt(tabIDStr, 10, 64)
		if err == nil {
			var tabName, tabIcon, tabColor string
			err := db.QueryRow("SELECT name, icon, color FROM scraper_tabs WHERE id = ?", tabID).Scan(&tabName, &tabIcon, &tabColor)
			if err == nil {
				return tabName, tabIcon, tabColor
			}
		}
	}

	return displayName, icon, color
}

// GetScraperDisplayName returns just the display name, resolving custom_* types from the DB
func GetScraperDisplayName(db *sql.DB, scraperType string) string {
	name, _, _ := GetScraperDisplayInfoWithDB(db, scraperType)
	return name
}

// ScraperTypeInfo holds a scraper type with its display name for frontend use
type ScraperTypeInfo struct {
	Type        string `json:"type"`
	DisplayName string `json:"display_name"`
}

// ResolveScraperTypeNames converts raw scraper type strings to ScraperTypeInfo with display names
func ResolveScraperTypeNames(db *sql.DB, types []string) []ScraperTypeInfo {
	result := make([]ScraperTypeInfo, 0, len(types))
	for _, t := range types {
		name := GetScraperDisplayName(db, t)
		result = append(result, ScraperTypeInfo{
			Type:        t,
			DisplayName: fmt.Sprintf("%s", name),
		})
	}
	return result
}
