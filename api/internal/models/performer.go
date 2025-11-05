package models

import (
	"encoding/json"
	"time"
)

// Performer represents a performer/actor in videos
type Performer struct {
	ID          int64     `json:"id" db:"id"`
	Name        string    `json:"name" db:"name" binding:"required"`
	PreviewPath string    `json:"preview_path" db:"preview_path"`
	FolderPath  string    `json:"folder_path" db:"folder_path"`
	SceneCount  int       `json:"scene_count" db:"scene_count"`
	Zoo         bool      `json:"zoo" db:"zoo"`
	Metadata    string    `json:"-" db:"metadata"` // JSON string in DB
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`

	// Parsed metadata (loaded from Metadata JSON field)
	MetadataObj *PerformerMetadata `json:"metadata,omitempty" db:"-"`
}

// PerformerMetadata represents additional performer information from external APIs
type PerformerMetadata struct {
	// Commonly used flat fields for easy access
	Bio         string   `json:"bio,omitempty"`
	Birthdate   string   `json:"birthdate,omitempty"`
	Birthplace  string   `json:"birthplace,omitempty"`
	Height      string   `json:"height,omitempty"`
	Weight      string   `json:"weight,omitempty"`
	Ethnicity   string   `json:"ethnicity,omitempty"`
	HairColor   string   `json:"hair_color,omitempty"`
	EyeColor    string   `json:"eye_color,omitempty"`
	Measurements string  `json:"measurements,omitempty"`
	Tattoos     string   `json:"tattoos,omitempty"`
	Piercings   string   `json:"piercings,omitempty"`
	CareerStart int      `json:"career_start,omitempty"`
	CareerEnd   int      `json:"career_end,omitempty"`
	Aliases     []string `json:"aliases,omitempty"`
	URLs        []string `json:"urls,omitempty"`
	ImageURL    string   `json:"image_url,omitempty"`
	ExternalID  string   `json:"external_id,omitempty"` // AdultDataLink ID

	// Full AdultDataLink response (stored for advanced use)
	AdultDataLinkResponse map[string]interface{} `json:"adult_data_link_response,omitempty"`
}

// PerformerCreate represents the data needed to create a performer
type PerformerCreate struct {
	Name        string              `json:"name" binding:"required"`
	PreviewPath string              `json:"preview_path"`
	FolderPath  string              `json:"folder_path"`
	Metadata    *PerformerMetadata  `json:"metadata,omitempty"`
}

// PerformerUpdate represents the data that can be updated
type PerformerUpdate struct {
	Name        *string             `json:"name,omitempty"`
	PreviewPath *string             `json:"preview_path,omitempty"`
	FolderPath  *string             `json:"folder_path,omitempty"`
	SceneCount  *int                `json:"scene_count,omitempty"`
	Metadata    *PerformerMetadata  `json:"metadata,omitempty"`
}

// PerformerWithVideos includes the performer's videos
type PerformerWithVideos struct {
	Performer
	Videos []Video `json:"videos,omitempty"`
}

// MarshalMetadata converts PerformerMetadata to JSON string for database storage
func (p *Performer) MarshalMetadata() error {
	if p.MetadataObj == nil {
		p.Metadata = "{}"
		return nil
	}
	data, err := json.Marshal(p.MetadataObj)
	if err != nil {
		return err
	}
	p.Metadata = string(data)
	return nil
}

// UnmarshalMetadata parses the JSON metadata string into PerformerMetadata
func (p *Performer) UnmarshalMetadata() error {
	if p.Metadata == "" || p.Metadata == "{}" {
		return nil
	}
	p.MetadataObj = &PerformerMetadata{}
	return json.Unmarshal([]byte(p.Metadata), p.MetadataObj)
}
