package models

import "time"

// VideoList represents a named list of videos
type VideoList struct {
	ID          int64     `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	Color       string    `json:"color" db:"color"`
	Icon        string    `json:"icon" db:"icon"`
	IsDefault   bool      `json:"is_default" db:"is_default"`
	Position    int       `json:"position" db:"position"`
	VideoCount  int       `json:"video_count" db:"video_count"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`

	// Loaded separately
	Items []VideoListItem `json:"items,omitempty"`
}

// VideoListItem represents a video in a list
type VideoListItem struct {
	ID             int64     `json:"id" db:"id"`
	ListID         int64     `json:"list_id" db:"list_id"`
	VideoID        *int64    `json:"video_id,omitempty" db:"video_id"`
	ScrapedVideoID *int64    `json:"scraped_video_id,omitempty" db:"scraped_video_id"`
	Position       int       `json:"position" db:"position"`
	Notes          string    `json:"notes" db:"notes"`
	AddedAt        time.Time `json:"added_at" db:"added_at"`

	// Video details (loaded via JOIN)
	Title         string  `json:"title,omitempty"`
	ThumbnailPath string  `json:"thumbnail_path,omitempty"`
	ThumbnailURL  string  `json:"thumbnail_url,omitempty"`
	Duration      float64 `json:"duration,omitempty"`
	FileSize      int64   `json:"file_size,omitempty"`
	Resolution    string  `json:"resolution,omitempty"`
	Source        string  `json:"source,omitempty"`
	IsScraped     bool    `json:"is_scraped"`
	FilePath      string  `json:"file_path,omitempty"`
	LibraryID     int64   `json:"library_id,omitempty"`
	LibraryName   string  `json:"library_name,omitempty"`
}

// VideoListCreate represents data needed to create a new list
type VideoListCreate struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Color       string `json:"color"`
	Icon        string `json:"icon"`
}

// VideoListUpdate represents data that can be updated on a list
type VideoListUpdate struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Color       *string `json:"color,omitempty"`
	Icon        *string `json:"icon,omitempty"`
	Position    *int    `json:"position,omitempty"`
}

// AddToListRequest represents a request to add videos to a list
type AddToListRequest struct {
	VideoIDs        []int64 `json:"video_ids,omitempty"`
	ScrapedVideoIDs []int64 `json:"scraped_video_ids,omitempty"`
	Notes           string  `json:"notes,omitempty"`
}

// RemoveFromListRequest represents a request to remove videos from a list
type RemoveFromListRequest struct {
	VideoIDs        []int64 `json:"video_ids,omitempty"`
	ScrapedVideoIDs []int64 `json:"scraped_video_ids,omitempty"`
	ItemIDs         []int64 `json:"item_ids,omitempty"`
}

// VideoListStats represents statistics about a video list
type VideoListStats struct {
	TotalVideos    int     `json:"total_videos"`
	LocalVideos    int     `json:"local_videos"`
	ScrapedVideos  int     `json:"scraped_videos"`
	TotalDuration  float64 `json:"total_duration"`
	TotalSize      int64   `json:"total_size"`
	AverageRating  float64 `json:"average_rating"`
}
