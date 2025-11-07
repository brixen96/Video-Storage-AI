package models

import (
	"encoding/json"
	"time"
)

// Performer represents a performer in the database
type Performer struct {
	ID             int64       `json:"id"`
	Name           string      `json:"name"`
	PreviewPath    string      `json:"preview_path"`
	FolderPath     string      `json:"folder_path"`
	SceneCount     int         `json:"scene_count"`
	Zoo            bool        `json:"zoo"`
	Metadata       string      `json:"-"` // Raw JSON from DB
	MetadataObj    *PerformerMetadata `json:"metadata,omitempty"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
	IsLCPCandidate bool        `json:"is_lcp_candidate,omitempty"`
}

// PerformerCreate represents the data needed to create a new performer
type PerformerCreate struct {
	Name        string      `json:"name" binding:"required"`
	PreviewPath string      `json:"preview_path"`
	FolderPath  string      `json:"folder_path"`
	Metadata    *PerformerMetadata `json:"metadata,omitempty"`
}

// PerformerUpdate represents the data that can be updated for a performer
type PerformerUpdate struct {
	Name        *string     `json:"name,omitempty"`
	PreviewPath *string     `json:"preview_path,omitempty"`
	FolderPath  *string     `json:"folder_path,omitempty"`
	SceneCount  *int        `json:"scene_count,omitempty"`
	Zoo         *bool       `json:"zoo,omitempty"`
	Metadata    *PerformerMetadata `json:"metadata,omitempty"`
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