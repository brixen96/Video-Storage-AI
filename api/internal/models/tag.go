package models

import "time"

// Tag represents a tag that can be applied to videos
type Tag struct {
	ID        int64     `json:"id" db:"id"`
	Name      string    `json:"name" db:"name" binding:"required"`
	Color     string    `json:"color" db:"color"` // Hex color code
	Icon      string    `json:"icon" db:"icon"`   // Font Awesome icon name
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// TagCreate represents the data needed to create a tag
type TagCreate struct {
	Name  string `json:"name" binding:"required"`
	Color string `json:"color"`
	Icon  string `json:"icon"`
}

// TagUpdate represents the data that can be updated
type TagUpdate struct {
	Name  *string `json:"name,omitempty"`
	Color *string `json:"color,omitempty"`
	Icon  *string `json:"icon,omitempty"`
}

// TagMergeRequest represents a request to merge multiple tags
type TagMergeRequest struct {
	SourceTagIDs []int64 `json:"source_tag_ids" binding:"required,min=1"`
	TargetTagID  int64   `json:"target_tag_id" binding:"required"`
}

// TagWithCount includes the number of videos with this tag
type TagWithCount struct {
	Tag
	VideoCount int `json:"video_count" db:"video_count"`
}
