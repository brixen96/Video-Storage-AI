package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

// Performer represents a performer in the database
type Performer struct {
	ID             int64              `json:"id"`
	Name           string             `json:"name"`
	PreviewPath    sql.NullString     `json:"preview_path" db:"preview_path"`
	ThumbnailPath  sql.NullString     `json:"thumbnail_path" db:"thumbnail_path"` // Static thumbnail from preview video
	FolderPath     sql.NullString     `json:"folder_path" db:"folder_path"`
	VideoCount     int                `json:"video_count"` // Number of videos linked to this performer
	Category       string             `json:"category" db:"category"` // 'regular', 'zoo', or '3d'
	Metadata       string             `json:"-"` // Raw JSON from DB
	MetadataObj    *PerformerMetadata `json:"metadata,omitempty"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
	IsLCPCandidate bool               `json:"is_lcp_candidate,omitempty"`
}

// PerformerCreate represents the data needed to create a new performer
type PerformerCreate struct {
	Name        string      `json:"name" binding:"required"`
	PreviewPath string      `json:"preview_path"`
	FolderPath  string      `json:"folder_path"`
	Category    string      `json:"category"` // 'regular', 'zoo', or '3d'
	Metadata    *PerformerMetadata `json:"metadata,omitempty"`
}

// PerformerUpdate represents the data that can be updated for a performer
type PerformerUpdate struct {
	Name          *string     `json:"name,omitempty"`
	PreviewPath   *string     `json:"preview_path,omitempty"`
	ThumbnailPath *string     `json:"thumbnail_path,omitempty"`
	FolderPath    *string     `json:"folder_path,omitempty"`
	Category      *string     `json:"category,omitempty"` // 'regular', 'zoo', or '3d'
	Metadata      *PerformerMetadata `json:"metadata,omitempty"`
}

// PerformerMetadata represents additional performer information, often from external sources
type PerformerMetadata struct {
	Birthdate    string   `json:"birthdate,omitempty"`
	Birthplace   string   `json:"birthplace,omitempty"`
	Height       string   `json:"height,omitempty"`
	Weight       string   `json:"weight,omitempty"`
	Ethnicity    string   `json:"ethnicity,omitempty"`
	HairColor    string   `json:"hair_color,omitempty"`
	EyeColor     string   `json:"eye_color,omitempty"`
	Measurements string   `json:"measurements,omitempty"`
	Tattoos      string   `json:"tattoos,omitempty"`
	Piercings    string   `json:"piercings,omitempty"`
	Bio          string   `json:"bio,omitempty"`
	CareerStart  int      `json:"career_start,omitempty"`
	CareerEnd    int      `json:"career_end,omitempty"`
	Aliases      []string `json:"aliases,omitempty"`
	URLs         []string `json:"urls,omitempty"`
	ImageURL     string   `json:"image_url,omitempty"`

	// Raw response from AdultDataLink API
	AdultDataLinkResponse map[string]interface{} `json:"adult_data_link_response,omitempty"`
}

// UnmarshalMetadata parses the metadata JSON string into MetadataObj
func (p *Performer) UnmarshalMetadata() error {
	if p.Metadata == "" || p.Metadata == "{}" {
		return nil
	}
	return json.Unmarshal([]byte(p.Metadata), &p.MetadataObj)
}

// MarshalMetadata converts MetadataObj to a JSON string for database storage
func (p *Performer) MarshalMetadata() error {
	bytes, err := json.Marshal(p.MetadataObj)
	p.Metadata = string(bytes)
	return err
}

// MarshalJSON provides custom JSON marshaling to handle sql.NullString fields
func (p *Performer) MarshalJSON() ([]byte, error) {
	type Alias Performer
	return json.Marshal(&struct {
		PreviewPath   string `json:"preview_path"`
		ThumbnailPath string `json:"thumbnail_path"`
		FolderPath    string `json:"folder_path"`
		*Alias
	}{
		PreviewPath:   p.PreviewPath.String,
		ThumbnailPath: p.ThumbnailPath.String,
		FolderPath:    p.FolderPath.String,
		Alias:         (*Alias)(p),
	})
}