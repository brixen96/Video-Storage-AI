package models

import (
	"encoding/json"
	"time"
)

// Studio represents a production studio
type Studio struct {
	ID          int64     `json:"id" db:"id"`
	Name        string    `json:"name" db:"name" binding:"required"`
	LogoPath    string    `json:"logo_path" db:"logo_path"`
	Description string    `json:"description" db:"description"`
	FoundedDate string    `json:"founded_date" db:"founded_date"` // Store as string for flexibility
	Country     string    `json:"country" db:"country"`
	Metadata    string    `json:"-" db:"metadata"` // JSON string in DB
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`

	// Parsed metadata
	MetadataObj *StudioMetadata `json:"metadata,omitempty" db:"-"`
}

// StudioMetadata represents additional studio information
type StudioMetadata struct {
	Website      string   `json:"website,omitempty"`
	ParentStudio string   `json:"parent_studio,omitempty"`
	Subsidiaries []string `json:"subsidiaries,omitempty"`
	Genres       []string `json:"genres,omitempty"`
	ExternalID   string   `json:"external_id,omitempty"`
}

// StudioCreate represents the data needed to create a studio
type StudioCreate struct {
	Name        string          `json:"name" binding:"required"`
	LogoPath    string          `json:"logo_path"`
	Description string          `json:"description"`
	FoundedDate string          `json:"founded_date"`
	Country     string          `json:"country"`
	Metadata    *StudioMetadata `json:"metadata,omitempty"`
}

// StudioUpdate represents the data that can be updated
type StudioUpdate struct {
	Name        *string         `json:"name,omitempty"`
	LogoPath    *string         `json:"logo_path,omitempty"`
	Description *string         `json:"description,omitempty"`
	FoundedDate *string         `json:"founded_date,omitempty"`
	Country     *string         `json:"country,omitempty"`
	Metadata    *StudioMetadata `json:"metadata,omitempty"`
}

// StudioWithGroups includes the studio's groups
type StudioWithGroups struct {
	Studio
	Groups []Group `json:"groups,omitempty"`
}

// MarshalMetadata converts StudioMetadata to JSON string
func (s *Studio) MarshalMetadata() error {
	if s.MetadataObj == nil {
		s.Metadata = "{}"
		return nil
	}
	data, err := json.Marshal(s.MetadataObj)
	if err != nil {
		return err
	}
	s.Metadata = string(data)
	return nil
}

// UnmarshalMetadata parses the JSON metadata string
func (s *Studio) UnmarshalMetadata() error {
	if s.Metadata == "" || s.Metadata == "{}" {
		return nil
	}
	s.MetadataObj = &StudioMetadata{}
	return json.Unmarshal([]byte(s.Metadata), s.MetadataObj)
}
