package services

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/brixen96/video-storage-ai/internal/database"
	"github.com/brixen96/video-storage-ai/internal/models"
)

// DashboardService provides dashboard-related data
type DashboardService struct {
	db                *sql.DB
	activityService   *ActivityService
	autoUpdateService *AutoUpdateService
}

// NewDashboardService creates a new DashboardService
func NewDashboardService(activitySvc *ActivityService, autoUpdateSvc *AutoUpdateService) *DashboardService {
	return &DashboardService{
		db:                database.GetDB(),
		activityService:   activitySvc,
		autoUpdateService: autoUpdateSvc,
	}
}

// GetStats returns aggregated dashboard statistics
func (s *DashboardService) GetStats() (*models.DashboardStats, error) {
	stats := &models.DashboardStats{}

	// Total local videos
	s.db.QueryRow("SELECT COUNT(*) FROM videos").Scan(&stats.TotalVideos)

	// Total scraped videos
	s.db.QueryRow("SELECT COUNT(*) FROM scraped_videos").Scan(&stats.TotalScrapedVideos)

	// Total scraped threads
	s.db.QueryRow("SELECT COUNT(*) FROM scraped_threads").Scan(&stats.TotalThreads)

	// Total performers
	s.db.QueryRow("SELECT COUNT(*) FROM performers").Scan(&stats.TotalPerformers)

	// Watched threads count
	s.db.QueryRow("SELECT COUNT(*) FROM scraped_threads WHERE is_watched = 1").Scan(&stats.WatchedThreads)

	// Running tasks
	s.db.QueryRow("SELECT COUNT(*) FROM activity_logs WHERE status = 'running'").Scan(&stats.RunningTasks)

	// Pending tasks
	s.db.QueryRow("SELECT COUNT(*) FROM activity_logs WHERE status = 'pending'").Scan(&stats.PendingTasks)

	// Recent new items (last 24 hours from dashboard feed)
	s.db.QueryRow(`
		SELECT COUNT(*) FROM dashboard_feed_items
		WHERE item_type = 'new_content'
		AND created_at >= datetime('now', '-1 day')
	`).Scan(&stats.RecentNewItems)

	return stats, nil
}

// GetFeedItems returns recent feed items
func (s *DashboardService) GetFeedItems(limit, offset int) ([]*models.DashboardFeedItem, int, error) {
	// Get total count
	var total int
	s.db.QueryRow("SELECT COUNT(*) FROM dashboard_feed_items").Scan(&total)

	// Get items
	rows, err := s.db.Query(`
		SELECT id, item_type, title, message, icon, color, scraper_type, metadata, created_at
		FROM dashboard_feed_items
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []*models.DashboardFeedItem
	for rows.Next() {
		item := &models.DashboardFeedItem{}
		var scraperType sql.NullString
		err := rows.Scan(
			&item.ID, &item.ItemType, &item.Title, &item.Message,
			&item.Icon, &item.Color, &scraperType, &item.Metadata, &item.CreatedAt,
		)
		if err != nil {
			continue
		}
		if scraperType.Valid {
			item.ScraperType = scraperType.String
		}
		items = append(items, item)
	}

	return items, total, nil
}

// AddFeedItem adds a new item to the dashboard feed
func (s *DashboardService) AddFeedItem(item *models.DashboardFeedItem) error {
	if item.Icon == "" {
		item.Icon = "bell"
	}
	if item.Color == "" {
		item.Color = "primary"
	}
	if item.Metadata == "" {
		item.Metadata = "{}"
	}

	var scraperType interface{}
	if item.ScraperType == "" {
		scraperType = nil
	} else {
		scraperType = item.ScraperType
	}

	result, err := s.db.Exec(`
		INSERT INTO dashboard_feed_items (item_type, title, message, icon, color, scraper_type, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, item.ItemType, item.Title, item.Message, item.Icon, item.Color, scraperType, item.Metadata)

	if err != nil {
		return err
	}

	id, _ := result.LastInsertId()
	item.ID = id
	item.CreatedAt = time.Now()

	return nil
}

// AddNewContentFeedItem adds a feed item for new content found
func (s *DashboardService) AddNewContentFeedItem(scraperType string, count int, details map[string]interface{}) error {
	displayName, icon, color := models.GetScraperDisplayInfo(scraperType)

	metadata, _ := json.Marshal(details)

	item := &models.DashboardFeedItem{
		ItemType:    "new_content",
		Title:       displayName,
		Message:     formatNewContentMessage(count),
		Icon:        icon,
		Color:       color,
		ScraperType: scraperType,
		Metadata:    string(metadata),
	}

	return s.AddFeedItem(item)
}

// AddScrapeCompletedFeedItem adds a feed item for completed scrape
func (s *DashboardService) AddScrapeCompletedFeedItem(scraperType, message string, success bool) error {
	displayName, icon, color := models.GetScraperDisplayInfo(scraperType)

	itemType := "scrape_completed"
	if !success {
		itemType = "error"
		color = "danger"
	}

	item := &models.DashboardFeedItem{
		ItemType:    itemType,
		Title:       displayName,
		Message:     message,
		Icon:        icon,
		Color:       color,
		ScraperType: scraperType,
	}

	return s.AddFeedItem(item)
}

// AddSystemFeedItem adds a system notification to the feed
func (s *DashboardService) AddSystemFeedItem(title, message string) error {
	item := &models.DashboardFeedItem{
		ItemType: "system",
		Title:    title,
		Message:  message,
		Icon:     "info-circle",
		Color:    "info",
	}

	return s.AddFeedItem(item)
}

// CleanOldFeedItems removes feed items older than specified days
func (s *DashboardService) CleanOldFeedItems(daysToKeep int) (int64, error) {
	result, err := s.db.Exec(`
		DELETE FROM dashboard_feed_items
		WHERE created_at < datetime('now', '-' || ? || ' days')
	`, daysToKeep)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// GetRunningTasks returns currently running tasks
func (s *DashboardService) GetRunningTasks() ([]*models.ActivityLog, error) {
	rows, err := s.db.Query(`
		SELECT id, task_type, status, message, progress, details,
			   started_at, completed_at, is_paused, paused_at
		FROM activity_logs
		WHERE status = 'running' OR status = 'pending'
		ORDER BY started_at DESC
		LIMIT 10
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*models.ActivityLog
	for rows.Next() {
		task := &models.ActivityLog{}
		var details sql.NullString
		var completedAt, pausedAt sql.NullTime
		err := rows.Scan(
			&task.ID, &task.TaskType, &task.Status, &task.Message,
			&task.Progress, &details, &task.StartedAt, &completedAt,
			&task.IsPaused, &pausedAt,
		)
		if err != nil {
			continue
		}
		if details.Valid {
			task.Details = details.String
		}
		if completedAt.Valid {
			task.CompletedAt = &completedAt.Time
		}
		if pausedAt.Valid {
			task.PausedAt = &pausedAt.Time
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// GetScraperStatusCards returns status cards for dashboard
func (s *DashboardService) GetScraperStatusCards() ([]*models.ScraperStatusCard, error) {
	return s.autoUpdateService.GetScraperStatusCards()
}

// formatNewContentMessage formats a message for new content
func formatNewContentMessage(count int) string {
	if count == 1 {
		return "Found 1 new item"
	}
	return formatPlural(count, "new item", "new items")
}

// formatPlural formats a count with singular/plural form
func formatPlural(count int, singular, plural string) string {
	if count == 1 {
		return "1 " + singular
	}
	return formatInt(count) + " " + plural
}

// formatInt formats an integer with commas
func formatInt(n int) string {
	if n < 1000 {
		return itoa(n)
	}
	return itoa(n/1000) + "," + padLeft(itoa(n%1000), 3, '0')
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

func padLeft(s string, length int, pad rune) string {
	for len(s) < length {
		s = string(pad) + s
	}
	return s
}
