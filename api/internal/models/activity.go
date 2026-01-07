package models

import (
	"encoding/json"
	"time"
)

// Task status constants
const (
	TaskStatusPending   = "pending"
	TaskStatusRunning   = "running"
	TaskStatusCompleted = "completed"
	TaskStatusFailed    = "failed"
)

// Task type constants
const (
	TaskTypeScanning  = "scanning"
	TaskTypeIndexing  = "indexing"
	TaskTypeAITagging = "ai_tagging"
	TaskTypeMetadata  = "metadata"
	TaskTypeThumbnail = "thumbnail"
	TaskTypeTranscode = "transcode"
)

// Activity represents a background task or operation (new schema)
type Activity struct {
	ID            int                    `json:"id" db:"id"`
	TaskType      string                 `json:"task_type" db:"task_type"`
	Status        string                 `json:"status" db:"status"`
	Message       string                 `json:"message" db:"message"`
	Details       string                 `json:"-" db:"details"`
	DetailsObj    map[string]interface{} `json:"details,omitempty" db:"-"`
	Progress      int                    `json:"progress" db:"progress"`
	StartedAt     time.Time              `json:"started_at" db:"started_at"`
	CompletedAt   *time.Time             `json:"completed_at,omitempty" db:"completed_at"`
	UpdatedAt     time.Time              `json:"updated_at" db:"updated_at"`
	Error         *string                `json:"error,omitempty" db:"error"`
	IsPaused      bool                   `json:"is_paused" db:"is_paused"`
	PausedAt      *time.Time             `json:"paused_at,omitempty" db:"paused_at"`
	Checkpoint    string                 `json:"-" db:"checkpoint"`
	CheckpointObj map[string]interface{} `json:"checkpoint,omitempty" db:"-"`
}

// UnmarshalDetails converts JSON string from database to DetailsObj
func (a *Activity) UnmarshalDetails() error {
	if a.Details == "" || a.Details == "{}" {
		a.DetailsObj = make(map[string]interface{})
		return nil
	}
	return json.Unmarshal([]byte(a.Details), &a.DetailsObj)
}

// UnmarshalCheckpoint converts JSON string from database to CheckpointObj
func (a *Activity) UnmarshalCheckpoint() error {
	if a.Checkpoint == "" || a.Checkpoint == "{}" {
		a.CheckpointObj = make(map[string]interface{})
		return nil
	}
	return json.Unmarshal([]byte(a.Checkpoint), &a.CheckpointObj)
}

// ActivityLog represents a background task or operation (legacy schema)
type ActivityLog struct {
	ID          int64      `json:"id" db:"id"`
	TaskType    string     `json:"task_type" db:"task_type" binding:"required"`
	Status      string     `json:"status" db:"status" binding:"required"`
	Message     string     `json:"message" db:"message"`
	Progress    int        `json:"progress" db:"progress"`
	StartedAt   time.Time  `json:"started_at" db:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	Details     string     `json:"-" db:"details"`
	IsPaused    bool       `json:"is_paused" db:"is_paused"`
	PausedAt    *time.Time `json:"paused_at,omitempty" db:"paused_at"`
	Checkpoint  string     `json:"-" db:"checkpoint"`

	// Parsed details
	DetailsObj     map[string]interface{} `json:"details,omitempty" db:"-"`
	CheckpointObj  map[string]interface{} `json:"checkpoint,omitempty" db:"-"`
}

// MarshalDetails converts DetailsObj to JSON string for database storage
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

// UnmarshalDetails converts JSON string from database to DetailsObj
func (a *ActivityLog) UnmarshalDetails() error {
	if a.Details == "" {
		a.DetailsObj = make(map[string]interface{})
		return nil
	}
	return json.Unmarshal([]byte(a.Details), &a.DetailsObj)
}

// MarshalCheckpoint converts CheckpointObj to JSON string for database storage
func (a *ActivityLog) MarshalCheckpoint() error {
	if a.CheckpointObj == nil {
		a.Checkpoint = "{}"
		return nil
	}
	data, err := json.Marshal(a.CheckpointObj)
	if err != nil {
		return err
	}
	a.Checkpoint = string(data)
	return nil
}

// UnmarshalCheckpoint converts JSON string from database to CheckpointObj
func (a *ActivityLog) UnmarshalCheckpoint() error {
	if a.Checkpoint == "" || a.Checkpoint == "{}" {
		a.CheckpointObj = make(map[string]interface{})
		return nil
	}
	return json.Unmarshal([]byte(a.Checkpoint), &a.CheckpointObj)
}

// ActivityLogCreate represents the data needed to create a new activity log
type ActivityLogCreate struct {
	TaskType string                 `json:"task_type" binding:"required"`
	Status   string                 `json:"status" binding:"required"`
	Message  string                 `json:"message"`
	Progress int                    `json:"progress"`
	Details  map[string]interface{} `json:"details"`
}

// ActivityLogUpdate represents the data that can be updated in an activity log
type ActivityLogUpdate struct {
	Status    *string                `json:"status"`
	Message   *string                `json:"message"`
	Progress  *int                   `json:"progress"`
	Details   map[string]interface{} `json:"details"`
	Completed bool                   `json:"completed"`
}

// ActivityStatus represents the current status of all activities
type ActivityStatus struct {
	RunningTasks   int           `json:"running_tasks"`
	PendingTasks   int           `json:"pending_tasks"`
	CompletedTasks int           `json:"completed_tasks"`
	FailedTasks    int           `json:"failed_tasks"`
	CurrentTasks   []ActivityLog `json:"current_tasks"`
}

// ActivityStats represents statistics about activities
type ActivityStats struct {
	TotalTasks     int            `json:"total_tasks"`
	CompletedTasks int            `json:"completed_tasks"`
	FailedTasks    int            `json:"failed_tasks"`
	RunningTasks   int            `json:"running_tasks"`
	PendingTasks   int            `json:"pending_tasks"`
	TasksByType    map[string]int `json:"tasks_by_type"`
}
