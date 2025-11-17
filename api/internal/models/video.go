package models

import "time"

// Video represents a video file with its metadata
type Video struct {
	ID            int64      `json:"id" db:"id"`
	LibraryID     int64      `json:"library_id" db:"library_id"`
	Title         string     `json:"title" db:"title" binding:"required"`
	FilePath      string     `json:"file_path" db:"file_path" binding:"required"`
	FileSize      int64      `json:"file_size" db:"file_size"`
	Duration      float64    `json:"duration" db:"duration"`
	Codec         string     `json:"codec" db:"codec"`
	Resolution    string     `json:"resolution" db:"resolution"`
	Bitrate       int64      `json:"bitrate" db:"bitrate"`
	FPS           float64    `json:"fps" db:"fps"`
	ThumbnailPath string     `json:"thumbnail_path" db:"thumbnail_path"`
	Date          string     `json:"date" db:"date"`
	Rating        int        `json:"rating" db:"rating"`
	Description   string     `json:"description" db:"description"`
	IsFavorite    bool       `json:"is_favorite" db:"is_favorite"`
	IsPinned      bool       `json:"is_pinned" db:"is_pinned"`
	NotInterested bool       `json:"not_interested" db:"not_interested"`
	InEditList    bool       `json:"in_edit_list" db:"in_edit_list"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
	LastPlayedAt  *time.Time `json:"last_played_at,omitempty" db:"last_played_at"`
	PlayCount     int        `json:"play_count" db:"play_count"`

	// Relationships (loaded separately)
	Performers []Performer `json:"performers,omitempty"`
	Tags       []Tag       `json:"tags,omitempty"`
	Studios    []Studio    `json:"studios,omitempty"`
	Groups     []Group     `json:"groups,omitempty"`
}

// VideoCreate represents the data needed to create a video
type VideoCreate struct {
	LibraryID     int64   `json:"library_id" binding:"required"`
	Title         string  `json:"title" binding:"required"`
	FilePath      string  `json:"file_path" binding:"required"`
	FileSize      int64   `json:"file_size"`
	Duration      float64 `json:"duration"`
	Codec         string  `json:"codec"`
	Resolution    string  `json:"resolution"`
	Bitrate       int64   `json:"bitrate"`
	FPS           float64 `json:"fps"`
	ThumbnailPath string  `json:"thumbnail_path"`
}

// VideoUpdate represents the data that can be updated
type VideoUpdate struct {
	Title         *string  `json:"title,omitempty"`
	StudioID      *int64   `json:"studio_id,omitempty"`
	GroupID       *int64   `json:"group_id,omitempty"`
	PerformerIDs  []int64  `json:"performer_ids,omitempty"`
	TagIDs        []int64  `json:"tag_ids,omitempty"`
	Date          *string  `json:"date,omitempty"`
	Rating        *int     `json:"rating,omitempty"`
	Description   *string  `json:"description,omitempty"`
	IsFavorite    *bool    `json:"is_favorite,omitempty"`
	IsPinned      *bool    `json:"is_pinned,omitempty"`
	PlayCount     *int     `json:"play_count,omitempty"`
	NotInterested *bool    `json:"not_interested,omitempty"`
	InEditList    *bool    `json:"in_edit_list,omitempty"`
}

// VideoSearchQuery represents search parameters
type VideoSearchQuery struct {
	Query         string  `json:"query" form:"query"`
	LibraryID     int64   `json:"library_id" form:"library_id"`
	PerformerID   int64   `json:"performer_id" form:"performer_id"`
	StudioID      int64   `json:"studio_id" form:"studio_id"`
	GroupID       int64   `json:"group_id" form:"group_id"`
	TagID         int64   `json:"tag_id" form:"tag_id"`
	TagIDs        []int64 `json:"tag_ids" form:"tag_ids"`
	Zoo           *bool   `json:"zoo" form:"zoo"`
	Resolution    string  `json:"resolution" form:"resolution"`
	MinDuration   float64 `json:"min_duration" form:"min_duration"`
	MaxDuration   float64 `json:"max_duration" form:"max_duration"`
	MinSize       int64   `json:"min_size" form:"min_size"`
	MaxSize       int64   `json:"max_size" form:"max_size"`
	DateFrom      string  `json:"date_from" form:"date_from"`
	DateTo        string  `json:"date_to" form:"date_to"`
	HasPreview    *bool   `json:"has_preview" form:"has_preview"`
	MissingMeta   *bool   `json:"missing_metadata" form:"missing_metadata"`
	NotInterested *bool   `json:"not_interested" form:"not_interested"`
	InEditList    *bool   `json:"in_edit_list" form:"in_edit_list"`
	SortBy        string  `json:"sort_by" form:"sort_by"`       // created_at, duration, play_count, title
	SortOrder     string  `json:"sort_order" form:"sort_order"` // asc, desc
	Page          int     `json:"page" form:"page"`
	Limit         int     `json:"limit" form:"limit"`
}
