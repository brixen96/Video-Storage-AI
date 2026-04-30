package models

// VideoLink represents a confirmed or suggested link between an Edited and a Backup video.
type VideoLink struct {
	ID            int64   `json:"id"`
	EditedVideoID int64   `json:"edited_video_id"`
	BackupVideoID int64   `json:"backup_video_id"`
	Confidence    float64 `json:"confidence"`
	Status        string  `json:"status"` // suggested | confirmed | rejected
	MatchReasons  string  `json:"match_reasons"` // JSON array e.g. ["duration","filename"]
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`

	// Populated for display
	EditedVideo *LinkedVideoEntry `json:"edited_video,omitempty"`
	BackupVideo *LinkedVideoEntry `json:"backup_video,omitempty"`
}

// LinkedVideoEntry is a minimal video projection used inside VideoLink for display.
type LinkedVideoEntry struct {
	ID           int64   `json:"id"`
	Title        string  `json:"title"`
	FilePath     string  `json:"file_path"`
	Duration     float64 `json:"duration"`
	LibraryID    int64   `json:"library_id"`
	LibraryName  string  `json:"library_name"`
}
