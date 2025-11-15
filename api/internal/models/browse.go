package models

import "time"

// BrowseItem represents a file or folder in the browser
type BrowseItem struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`      // Relative path from library root
	FullPath  string    `json:"full_path"` // Absolute file path
	Type      string    `json:"type"`      // "folder", "video", or "file"
	IsDir     bool      `json:"is_dir"`
	Size      int64     `json:"size"`
	Modified  time.Time `json:"modified"`
	Extension string    `json:"extension,omitempty"`

	// Video-specific fields (populated later if needed)
	Duration   float64 `json:"duration,omitempty"`
	Width      int     `json:"width,omitempty"`
	Height     int     `json:"height,omitempty"`
	FrameRate  float64 `json:"frame_rate,omitempty"`
	Thumbnail  string  `json:"thumbnail,omitempty"`
	VideoID    *int64  `json:"video_id,omitempty"` // If already in database
	InDatabase bool    `json:"in_database"`

	// Marking flags
	NotInterested bool `json:"not_interested"`
	InEditList    bool `json:"in_edit_list"`
}

// BrowseResponse represents the response from browsing a directory
type BrowseResponse struct {
	LibraryID   int64        `json:"library_id"`
	LibraryName string       `json:"library_name"`
	CurrentPath string       `json:"current_path"`
	FullPath    string       `json:"full_path"`
	Items       []BrowseItem `json:"items"`
	TotalItems  int          `json:"total_items"`
	FolderCount int          `json:"folder_count"`
	VideoCount  int          `json:"video_count"`
	FileCount   int          `json:"file_count"`
}

// VideoMarkingUpdate represents an update to video marking
type VideoMarkingUpdate struct {
	FilePath      string `json:"file_path" binding:"required"`
	NotInterested *bool  `json:"not_interested,omitempty"`
	InEditList    *bool  `json:"in_edit_list,omitempty"`
}
