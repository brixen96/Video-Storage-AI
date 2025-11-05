package models

import (
	"encoding/json"
	"time"
)

// Group represents a sub-label or series under a studio
type Group struct {
	ID          int64     `json:"id" db:"id"`
	StudioID    int64     `json:"studio_id" db:"studio_id" binding:"required"`
	Name        string    `json:"name" db:"name" binding:"required"`
	LogoPath    string    `json:"logo_path" db:"logo_path"`
	Description string    `json:"description" db:"description"`
	Metadata    string    `json:"-" db:"metadata"` // JSON string in DB
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`

	// Parsed metadata
	MetadataObj *GroupMetadata `json:"metadata,omitempty" db:"-"`

	// Relationships
	Studio *Studio `json:"studio,omitempty" db:"-"`
}

// GroupMetadata represents additional group information
type GroupMetadata struct {
	SeriesNumber int      `json:"series_number,omitempty"`
	VideoCount   int      `json:"video_count,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	ExternalID   string   `json:"external_id,omitempty"`
}

// GroupCreate represents the data needed to create a group
type GroupCreate struct {
	StudioID    int64          `json:"studio_id" binding:"required"`
	Name        string         `json:"name" binding:"required"`
	LogoPath    string         `json:"logo_path"`
	Description string         `json:"description"`
	Metadata    *GroupMetadata `json:"metadata,omitempty"`
}

// GroupUpdate represents the data that can be updated
type GroupUpdate struct {
	Name        *string        `json:"name,omitempty"`
	LogoPath    *string        `json:"logo_path,omitempty"`
	Description *string        `json:"description,omitempty"`
	Metadata    *GroupMetadata `json:"metadata,omitempty"`
}

// GroupWithVideos includes the group's videos
type GroupWithVideos struct {
	Group
	Videos []Video `json:"videos,omitempty"`
}

// MarshalMetadata converts GroupMetadata to JSON string
func (g *Group) MarshalMetadata() error {
	if g.MetadataObj == nil {
		g.Metadata = "{}"
		return nil
	}
	data, err := json.Marshal(g.MetadataObj)
	if err != nil {
		return err
	}
	g.Metadata = string(data)
	return nil
}

// UnmarshalMetadata parses the JSON metadata string
func (g *Group) UnmarshalMetadata() error {
	if g.Metadata == "" || g.Metadata == "{}" {
		return nil
	}
	g.MetadataObj = &GroupMetadata{}
	return json.Unmarshal([]byte(g.Metadata), g.MetadataObj)
}
