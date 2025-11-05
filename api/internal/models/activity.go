package models

import (
	"encoding/json"
	"time"
)

// ActivityLog represents a background task or operation
type ActivityLog struct {
	ID          int64      `json:"id" db:"id"`
	TaskType    string     `json:"task_type" db:"task_type" binding:"required"` // scanning, indexing, ai_tagging, etc.
	Status      string     `json:"status" db:"status" binding:"required"`       // pending, running, completed, failed
	Message     string     `json:"message" db:"message"`
	Progress    int        `json:"progress" db:"progress"` // 0-100
	StartedAt   time.Time  `json:"started_at" db:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	Details     string     `json:"-" db:"details"` // JSON string in DB

	// Parsed details
	DetailsObj map[string]interface{} `json:"details,omitempty" db:"-"`
}

// ActivityLogCreate represents the data needed to create an activity log
type ActivityLogCreate struct {
	TaskType string                 `json:"task_type" binding:"required"`
	Status   string                 `json:"status" binding:"required"`
	Message  string                 `json:"message"`
	Progress int                    `json:"progress"`
	Details  map[string]interface{} `json:"details,omitempty"`
}

// ActivityLogUpdate represents the data that can be updated
type ActivityLogUpdate struct {
	Status    *string                `json:"status,omitempty"`
	Message   *string                `json:"message,omitempty"`
	Progress  *int                   `json:"progress,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Completed bool                   `json:"completed,omitempty"` // Set CompletedAt to now
}

// ActivityStatus represents the current status of all running tasks
type ActivityStatus struct {
	RunningTasks  int           `json:"running_tasks"`
	PendingTasks  int           `json:"pending_tasks"`
	CompletedTasks int          `json:"completed_tasks"`
	FailedTasks   int           `json:"failed_tasks"`
	CurrentTasks  []ActivityLog `json:"current_tasks"`
}

// TaskType constants
const (
	TaskTypeScanning       = "scanning"
	TaskTypeIndexing       = "indexing"
	TaskTypeAITagging      = "ai_tagging"
	TaskTypeMetadataFetch  = "metadata_fetch"
	TaskTypeThumbnailGen   = "thumbnail_generation"
	TaskTypeVideoAnalysis  = "video_analysis"
	TaskTypeFileOperation  = "file_operation"
)

// TaskStatus constants
const (
	TaskStatusPending   = "pending"
	TaskStatusRunning   = "running"
	TaskStatusCompleted = "completed"
	TaskStatusFailed    = "failed"
)

// MarshalDetails converts Details map to JSON string
func (a *ActivityLog) MarshalDetails() error {
	if a.DetailsObj == nil {
		a.Details = "{}"
		return nil
	}
	data, err := json.Marshal(a.DetailsObj)
	if err != nil {
		return err
	}
	a.Details = string(data)
	return nil
}

// UnmarshalDetails parses the JSON details string
func (a *ActivityLog) UnmarshalDetails() error {
	if a.Details == "" || a.Details == "{}" {
		return nil
	}
	a.DetailsObj = make(map[string]interface{})
	return json.Unmarshal([]byte(a.Details), &a.DetailsObj)
}
