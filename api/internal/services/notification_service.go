package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/brixen96/video-storage-ai/internal/models"
)

// NotificationWebSocketHub interface for broadcasting notifications
type NotificationWebSocketHub interface {
	BroadcastNotification(notification *models.Notification)
}

var notificationWsHub NotificationWebSocketHub

// SetNotificationWebSocketHub sets the WebSocket hub for broadcasting
func SetNotificationWebSocketHub(hub NotificationWebSocketHub) {
	notificationWsHub = hub
}

// NotificationService handles notification management
type NotificationService struct {
	db *sql.DB
}

// NewNotificationService creates a new notification service
func NewNotificationService(db *sql.DB) *NotificationService {
	return &NotificationService{
		db: db,
	}
}

// Create creates a new notification
func (s *NotificationService) Create(req *models.CreateNotificationRequest) (*models.Notification, error) {
	// Set defaults
	if req.Priority == "" {
		req.Priority = "normal"
	}
	if req.Category == "" {
		req.Category = "system"
	}

	// Serialize metadata
	metadataJSON := "{}"
	if req.Metadata != nil {
		bytes, err := json.Marshal(req.Metadata)
		if err == nil {
			metadataJSON = string(bytes)
		}
	}

	now := time.Now()
	result, err := s.db.Exec(`
		INSERT INTO notifications (
			type, priority, title, message, category,
			action_url, action_label, metadata,
			related_entity_type, related_entity_id,
			expires_at, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		req.Type, req.Priority, req.Title, req.Message, req.Category,
		req.ActionURL, req.ActionLabel, metadataJSON,
		req.RelatedEntityType, req.RelatedEntityID,
		req.ExpiresAt, now,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create notification: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	notification, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Broadcast notification via WebSocket
	if notificationWsHub != nil {
		notificationWsHub.BroadcastNotification(notification)
	}

	return notification, nil
}

// GetByID retrieves a notification by ID
func (s *NotificationService) GetByID(id int64) (*models.Notification, error) {
	var notification models.Notification
	err := s.db.QueryRow(`
		SELECT
			id, type, priority, title, message, category,
			action_url, action_label, metadata,
			is_read, is_archived, related_entity_type, related_entity_id,
			created_at, read_at, expires_at
		FROM notifications
		WHERE id = ?
	`, id).Scan(
		&notification.ID, &notification.Type, &notification.Priority,
		&notification.Title, &notification.Message, &notification.Category,
		&notification.ActionURL, &notification.ActionLabel, &notification.Metadata,
		&notification.IsRead, &notification.IsArchived,
		&notification.RelatedEntityType, &notification.RelatedEntityID,
		&notification.CreatedAt, &notification.ReadAt, &notification.ExpiresAt,
	)

	if err != nil {
		return nil, err
	}

	return &notification, nil
}

// GetAll retrieves all notifications with filters
func (s *NotificationService) GetAll(unreadOnly bool, priority string, category string, limit int, offset int) ([]*models.Notification, error) {
	query := `
		SELECT
			id, type, priority, title, message, category,
			action_url, action_label, metadata,
			is_read, is_archived, related_entity_type, related_entity_id,
			created_at, read_at, expires_at
		FROM notifications
		WHERE is_archived = 0
		AND (expires_at IS NULL OR expires_at > datetime('now'))
	`
	args := []interface{}{}

	if unreadOnly {
		query += " AND is_read = 0"
	}

	if priority != "" {
		query += " AND priority = ?"
		args = append(args, priority)
	}

	if category != "" {
		query += " AND category = ?"
		args = append(args, category)
	}

	query += " ORDER BY priority DESC, created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []*models.Notification
	for rows.Next() {
		var notification models.Notification
		if err := rows.Scan(
			&notification.ID, &notification.Type, &notification.Priority,
			&notification.Title, &notification.Message, &notification.Category,
			&notification.ActionURL, &notification.ActionLabel, &notification.Metadata,
			&notification.IsRead, &notification.IsArchived,
			&notification.RelatedEntityType, &notification.RelatedEntityID,
			&notification.CreatedAt, &notification.ReadAt, &notification.ExpiresAt,
		); err != nil {
			continue
		}
		notifications = append(notifications, &notification)
	}

	return notifications, nil
}

// MarkAsRead marks a notification as read
func (s *NotificationService) MarkAsRead(id int64) error {
	now := time.Now()
	_, err := s.db.Exec(`
		UPDATE notifications
		SET is_read = 1, read_at = ?
		WHERE id = ?
	`, now, id)
	return err
}

// MarkAllAsRead marks all notifications as read
func (s *NotificationService) MarkAllAsRead() error {
	now := time.Now()
	_, err := s.db.Exec(`
		UPDATE notifications
		SET is_read = 1, read_at = ?
		WHERE is_read = 0 AND is_archived = 0
	`, now)
	return err
}

// Archive archives a notification
func (s *NotificationService) Archive(id int64) error {
	_, err := s.db.Exec(`
		UPDATE notifications
		SET is_archived = 1
		WHERE id = ?
	`, id)
	return err
}

// Delete deletes a notification
func (s *NotificationService) Delete(id int64) error {
	_, err := s.db.Exec(`DELETE FROM notifications WHERE id = ?`, id)
	return err
}

// DeleteOld deletes old archived notifications
func (s *NotificationService) DeleteOld(daysToKeep int) (int64, error) {
	result, err := s.db.Exec(`
		DELETE FROM notifications
		WHERE is_archived = 1
		AND created_at < datetime('now', ?)
	`, fmt.Sprintf("-%d days", daysToKeep))

	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// GetStats returns notification statistics
func (s *NotificationService) GetStats() (*models.NotificationStats, error) {
	stats := &models.NotificationStats{
		ByPriority: make(map[string]int64),
		ByCategory: make(map[string]int64),
	}

	// Total and unread count
	err := s.db.QueryRow(`
		SELECT
			COUNT(*) as total,
			COALESCE(SUM(CASE WHEN is_read = 0 THEN 1 ELSE 0 END), 0) as unread
		FROM notifications
		WHERE is_archived = 0
		AND (expires_at IS NULL OR expires_at > datetime('now'))
	`).Scan(&stats.Total, &stats.Unread)

	if err != nil {
		return nil, err
	}

	// By priority
	rows, err := s.db.Query(`
		SELECT priority, COUNT(*) as count
		FROM notifications
		WHERE is_archived = 0 AND is_read = 0
		GROUP BY priority
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var priority string
			var count int64
			if err := rows.Scan(&priority, &count); err == nil {
				stats.ByPriority[priority] = count
			}
		}
	}

	// By category
	rows2, err := s.db.Query(`
		SELECT category, COUNT(*) as count
		FROM notifications
		WHERE is_archived = 0 AND is_read = 0
		GROUP BY category
	`)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var category string
			var count int64
			if err := rows2.Scan(&category, &count); err == nil {
				stats.ByCategory[category] = count
			}
		}
	}

	return stats, nil
}

// NotifyJobCompleted creates a notification for completed job
func (s *NotificationService) NotifyJobCompleted(jobID int64, jobType string, success bool, message string) error {
	priority := "normal"
	title := fmt.Sprintf("Job Completed: %s", jobType)
	notifType := "job_completed"

	if !success {
		priority = "high"
		title = fmt.Sprintf("Job Failed: %s", jobType)
		notifType = "job_failed"
	}

	req := &models.CreateNotificationRequest{
		Type:              notifType,
		Priority:          priority,
		Title:             title,
		Message:           message,
		Category:          "scheduler",
		ActionURL:         fmt.Sprintf("/scheduler"),
		ActionLabel:       "View Scheduler",
		RelatedEntityType: "job",
		RelatedEntityID:   &jobID,
	}

	_, err := s.Create(req)
	return err
}

// NotifyScrapeCompleted creates a notification for completed scrape
func (s *NotificationService) NotifyScrapeCompleted(threadID int64, threadTitle string, linksFound int) error {
	req := &models.CreateNotificationRequest{
		Type:     "scrape_completed",
		Priority: "normal",
		Title:    "Scrape Completed",
		Message:  fmt.Sprintf("Found %d download links in: %s", linksFound, threadTitle),
		Category: "scraper",
		ActionURL: fmt.Sprintf("/scraper/%d", threadID),
		ActionLabel: "View Thread",
		RelatedEntityType: "thread",
		RelatedEntityID: &threadID,
		Metadata: map[string]interface{}{
			"links_found": linksFound,
			"thread_title": threadTitle,
		},
	}

	_, err := s.Create(req)
	return err
}

// NotifyScrapeCompletedEnhanced creates an enhanced notification with comprehensive scrape statistics
func (s *NotificationService) NotifyScrapeCompletedEnhanced(threadID int64, threadTitle string, postsFound int, linksFound int, duration time.Duration, isIncremental bool) error {
	// Format duration nicely
	durationStr := formatDuration(duration)

	// Create appropriate message based on scrape type
	var message string
	var priority string = "normal"

	if isIncremental {
		message = fmt.Sprintf("🔄 Incremental update: %d new posts, %d download links • %s", postsFound, linksFound, durationStr)
		if postsFound == 0 {
			message = fmt.Sprintf("✓ Thread up-to-date: No new posts found • %s", durationStr)
			priority = "low"
		}
	} else {
		message = fmt.Sprintf("✨ New thread scraped: %d posts, %d download links • %s", postsFound, linksFound, durationStr)
		if linksFound > 50 {
			priority = "high" // Highlight large finds
		}
	}

	req := &models.CreateNotificationRequest{
		Type:              "scrape_completed",
		Priority:          priority,
		Title:             "Scrape Completed",
		Message:           message,
		Category:          "scraper",
		ActionURL:         fmt.Sprintf("/scraper/%d", threadID),
		ActionLabel:       "View Thread",
		RelatedEntityType: "thread",
		RelatedEntityID:   &threadID,
		Metadata: map[string]interface{}{
			"thread_title":   threadTitle,
			"posts_found":    postsFound,
			"links_found":    linksFound,
			"duration_ms":    duration.Milliseconds(),
			"is_incremental": isIncremental,
		},
	}

	_, err := s.Create(req)
	return err
}

// NotifyLinksVerified creates a notification for link verification results
func (s *NotificationService) NotifyLinksVerified(threadID int64, threadTitle string, totalLinks int, deadLinks int) error {
	priority := "normal"
	if deadLinks > totalLinks/2 {
		priority = "high" // More than half are dead
	}

	req := &models.CreateNotificationRequest{
		Type:     "links_verified",
		Priority: priority,
		Title:    "Link Verification Complete",
		Message:  fmt.Sprintf("%d/%d links dead in: %s", deadLinks, totalLinks, threadTitle),
		Category: "downloads",
		ActionURL: fmt.Sprintf("/scraper/%d", threadID),
		ActionLabel: "View Links",
		RelatedEntityType: "thread",
		RelatedEntityID: &threadID,
		Metadata: map[string]interface{}{
			"total_links": totalLinks,
			"dead_links": deadLinks,
		},
	}

	_, err := s.Create(req)
	return err
}

// NotifySystemHealthDegraded creates a notification for degraded system health
func (s *NotificationService) NotifySystemHealthDegraded(component string, details string) error {
	req := &models.CreateNotificationRequest{
		Type:        "system_health_degraded",
		Priority:    "urgent",
		Title:       "System Health Degraded",
		Message:     fmt.Sprintf("%s: %s", component, details),
		Category:    "system",
		ActionURL:   "/system-health",
		ActionLabel: "View System Health",
		Metadata: map[string]interface{}{
			"component": component,
			"details": details,
		},
	}

	_, err := s.Create(req)
	return err
}

// NotifyAICostThreshold creates a notification when AI costs exceed threshold
func (s *NotificationService) NotifyAICostThreshold(cost float64, threshold float64) error {
	req := &models.CreateNotificationRequest{
		Type:     "ai_cost_threshold",
		Priority: "high",
		Title:    "AI Cost Alert",
		Message:  fmt.Sprintf("AI costs ($%.2f) exceeded threshold ($%.2f)", cost, threshold),
		Category: "ai",
		ActionURL: "/system-health",
		ActionLabel: "View AI Usage",
		Metadata: map[string]interface{}{
			"cost": cost,
			"threshold": threshold,
		},
	}

	_, err := s.Create(req)
	return err
}
