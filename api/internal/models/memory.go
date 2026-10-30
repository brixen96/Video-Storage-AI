package models

import "time"

// Memory represents an AI companion long-term memory entry
type Memory struct {
	ID         int64     `json:"id" db:"id"`
	Key        string    `json:"key" db:"key"`                 // Unique identifier for memory
	Value      string    `json:"value" db:"value"`             // Memory content
	Category   string    `json:"category" db:"category"`       // preference, fact, insight, task, note, profile
	Importance int       `json:"importance" db:"importance"`   // 1-10 importance level
	Source     string    `json:"source" db:"source"`           // manual, auto, browser, profile
	HitCount   int       `json:"hit_count" db:"hit_count"`     // times injected into context
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty" db:"last_used_at"`
}

// MemoryCategory represents memory category types
type MemoryCategory string

const (
	MemoryCategoryPreference MemoryCategory = "preference"
	MemoryCategoryFact       MemoryCategory = "fact"
	MemoryCategoryInsight    MemoryCategory = "insight"
	MemoryCategoryTask       MemoryCategory = "task"
	MemoryCategoryNote       MemoryCategory = "note"
)
