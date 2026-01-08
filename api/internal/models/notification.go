package models

import "time"

// Notification represents a system notification
type Notification struct {
	ID                int64      `json:"id" db:"id"`
	Type              string     `json:"type" db:"type"` // job_completed, job_failed, links_verified, scrape_completed, etc.
	Priority          string     `json:"priority" db:"priority"` // low, normal, high, urgent
	Title             string     `json:"title" db:"title"`
	Message           string     `json:"message" db:"message"`
	Category          string     `json:"category" db:"category"` // system, scraper, downloads, ai, etc.
	ActionURL         string     `json:"action_url" db:"action_url"`
	ActionLabel       string     `json:"action_label" db:"action_label"`
	Metadata          string     `json:"metadata" db:"metadata"` // JSON
	IsRead            bool       `json:"is_read" db:"is_read"`
	IsArchived        bool       `json:"is_archived" db:"is_archived"`
	RelatedEntityType string     `json:"related_entity_type" db:"related_entity_type"` // thread, job, activity, etc.
	RelatedEntityID   *int64     `json:"related_entity_id" db:"related_entity_id"`
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
	ReadAt            *time.Time `json:"read_at" db:"read_at"`
	ExpiresAt         *time.Time `json:"expires_at" db:"expires_at"`
}

// NotificationPreference represents user notification preferences
type NotificationPreference struct {
	ID              int64     `json:"id" db:"id"`
	NotificationType string    `json:"notification_type" db:"notification_type"`
	Enabled         bool      `json:"enabled" db:"enabled"`
	PriorityFilter  string    `json:"priority_filter" db:"priority_filter"` // all, high_only, urgent_only
	DeliveryMethods string    `json:"delivery_methods" db:"delivery_methods"` // JSON array: ["in_app", "email", "webhook"]
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// NotificationStats represents notification statistics
type NotificationStats struct {
	Total      int64            `json:"total"`
	Unread     int64            `json:"unread"`
	ByPriority map[string]int64 `json:"by_priority"`
	ByCategory map[string]int64 `json:"by_category"`
}

// CreateNotificationRequest represents a notification creation request
type CreateNotificationRequest struct {
	Type              string                 `json:"type" binding:"required"`
	Priority          string                 `json:"priority"`
	Title             string                 `json:"title" binding:"required"`
	Message           string                 `json:"message" binding:"required"`
	Category          string                 `json:"category"`
	ActionURL         string                 `json:"action_url"`
	ActionLabel       string                 `json:"action_label"`
	Metadata          map[string]interface{} `json:"metadata"`
	RelatedEntityType string                 `json:"related_entity_type"`
	RelatedEntityID   *int64                 `json:"related_entity_id"`
	ExpiresAt         *time.Time             `json:"expires_at"`
}
