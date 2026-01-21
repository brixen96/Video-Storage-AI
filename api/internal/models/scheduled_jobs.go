package models

import "time"

// ScheduledJob represents an automated job configuration
type ScheduledJob struct {
	ID             int64     `json:"id" db:"id"`
	JobType        string    `json:"job_type" db:"job_type"`           // scrape_thread, verify_links, cleanup_old_data, etc.
	ScheduleType   string    `json:"schedule_type" db:"schedule_type"` // cron, interval, once
	ScheduleConfig string    `json:"schedule_config" db:"schedule_config"` // JSON config
	TargetType     string    `json:"target_type" db:"target_type"`     // thread, performer, global
	TargetID       *int64    `json:"target_id" db:"target_id"`
	Enabled        bool      `json:"enabled" db:"enabled"`
	LastRunAt      *time.Time `json:"last_run_at" db:"last_run_at"`
	NextRunAt      *time.Time `json:"next_run_at" db:"next_run_at"`
	RunCount       int       `json:"run_count" db:"run_count"`
	SuccessCount   int       `json:"success_count" db:"success_count"`
	FailureCount   int       `json:"failure_count" db:"failure_count"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

// JobExecutionHistory represents a job execution record
type JobExecutionHistory struct {
	ID           int64     `json:"id" db:"id"`
	JobID        int64     `json:"job_id" db:"job_id"`
	StartedAt    time.Time `json:"started_at" db:"started_at"`
	CompletedAt  *time.Time `json:"completed_at" db:"completed_at"`
	Status       string    `json:"status" db:"status"` // running, completed, failed
	ResultData   string    `json:"result_data" db:"result_data"` // JSON result
	ErrorMessage string    `json:"error_message" db:"error_message"`
	DurationMs   int       `json:"duration_ms" db:"duration_ms"`
}

// ScheduleConfig represents the parsed schedule configuration
type ScheduleConfig struct {
	// For interval schedules
	IntervalMinutes int `json:"interval_minutes,omitempty"`

	// For cron schedules
	CronExpression string `json:"cron_expression,omitempty"`

	// For once schedules
	RunAt *time.Time `json:"run_at,omitempty"`

	// Job-specific config
	AutoStart      bool   `json:"auto_start,omitempty"`
	MaxRetries     int    `json:"max_retries,omitempty"`
	TimeoutMinutes int    `json:"timeout_minutes,omitempty"`
}

// JobResult represents the result of a job execution
type JobResult struct {
	Success      bool                   `json:"success"`
	ItemsProcessed int                  `json:"items_processed,omitempty"`
	ItemsSucceeded int                  `json:"items_succeeded,omitempty"`
	ItemsFailed    int                  `json:"items_failed,omitempty"`
	Message      string                 `json:"message,omitempty"`
	Details      map[string]interface{} `json:"details,omitempty"`
}
