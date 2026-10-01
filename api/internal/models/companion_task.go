package models

import "time"

// CompanionTask represents a shared task between AI Companion and developer
type CompanionTask struct {
	ID            int64      `json:"id"`
	Title         string     `json:"title"`
	Description   string     `json:"description"`
	Status        string     `json:"status"` // open, in_progress, completed, archived
	Priority      int        `json:"priority"` // 1-10, higher = more important
	CreatedBy     string     `json:"created_by"` // companion, developer, system
	AssignedTo    string     `json:"assigned_to"`
	Tags          []string   `json:"tags"`
	RelatedFile   string     `json:"related_file"`
	RelatedLine   int        `json:"related_line"`
	ErrorContext  string     `json:"error_context"`
	Resolution    string     `json:"resolution"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
}

// TaskStatus constants
const (
	CompanionTaskStatusOpen       = "open"
	CompanionTaskStatusInProgress = "in_progress"
	CompanionTaskStatusCompleted  = "completed"
	CompanionTaskStatusArchived   = "archived"
)

// TaskPriority constants
const (
	PriorityLow      = 1
	PriorityMedium   = 5
	PriorityHigh     = 8
	PriorityCritical = 10
)