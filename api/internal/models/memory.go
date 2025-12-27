package models

import "time"

// Memory represents an AI companion long-term memory entry
type Memory struct {
	ID         int64     `json:"id" db:"id"`
	Key        string    `json:"key" db:"key"`                 // Unique identifier for memory
	Value      string    `json:"value" db:"value"`             // Memory content
	Category   string    `json:"category" db:"category"`       // preference, fact, insight, task, note
	Importance int       `json:"importance" db:"importance"`   // 1-10 importance level
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
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
