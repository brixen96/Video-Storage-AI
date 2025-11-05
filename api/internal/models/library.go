package models

import "time"

// Library represents a video library/collection
type Library struct {
	ID        int64     `json:"id" db:"id"`
	Name      string    `json:"name" db:"name" binding:"required"`
	Path      string    `json:"path" db:"path" binding:"required"`
	Primary   bool      `json:"primary" db:"primary_lib"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	// Stats (loaded separately)
	VideoCount     int `json:"video_count,omitempty" db:"-"`
	PerformerCount int `json:"performer_count,omitempty" db:"-"`
}

// LibraryCreate represents the data needed to create a library
type LibraryCreate struct {
	Name    string `json:"name" binding:"required"`
	Path    string `json:"path" binding:"required"`
	Primary bool   `json:"primary"`
}

// LibraryUpdate represents the data that can be updated
type LibraryUpdate struct {
	Name    *string `json:"name,omitempty"`
	Path    *string `json:"path,omitempty"`
	Primary *bool   `json:"primary,omitempty"`
}

// LibraryWithStats includes library statistics
type LibraryWithStats struct {
	Library
	TotalSize      int64 `json:"total_size"`
	LastScanned    *time.Time `json:"last_scanned,omitempty"`
	ScanInProgress bool  `json:"scan_in_progress"`
}
