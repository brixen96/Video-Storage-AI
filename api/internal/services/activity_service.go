package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/brixen96/video-storage-ai/internal/database"
	"github.com/brixen96/video-storage-ai/internal/models"
)

// ActivityService handles activity logging and monitoring
type ActivityService struct {
	db *sql.DB
}

// NewActivityService creates a new activity service
func NewActivityService() *ActivityService {
	return &ActivityService{
		db: database.DB,
	}
}

// Create creates a new activity log
func (s *ActivityService) Create(create *models.ActivityLogCreate) (*models.ActivityLog, error) {
	activity := &models.ActivityLog{
		TaskType:   create.TaskType,
		Status:     create.Status,
		Message:    create.Message,
		Progress:   create.Progress,
		StartedAt:  time.Now(),
		DetailsObj: create.Details,
	}

	// Marshal details to JSON
	if err := activity.MarshalDetails(); err != nil {
		return nil, fmt.Errorf("failed to marshal details: %w", err)
	}

	query := `
		INSERT INTO activity_logs (task_type, status, message, progress, started_at, details)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	result, err := s.db.Exec(query, activity.TaskType, activity.Status, activity.Message,
		activity.Progress, activity.StartedAt, activity.Details)
	if err != nil {
		return nil, fmt.Errorf("failed to insert activity log: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert ID: %w", err)
	}

	activity.ID = id
	return activity, nil
}

// GetByID retrieves an activity log by ID
func (s *ActivityService) GetByID(id int64) (*models.ActivityLog, error) {
	query := `
		SELECT id, task_type, status, message, progress, started_at, completed_at, details
		FROM activity_logs
		WHERE id = ?
	`

	var activity models.ActivityLog
	err := s.db.QueryRow(query, id).Scan(
		&activity.ID, &activity.TaskType, &activity.Status, &activity.Message,
		&activity.Progress, &activity.StartedAt, &activity.CompletedAt, &activity.Details,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("activity log not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query activity log: %w", err)
	}

	// Unmarshal details
	if err := activity.UnmarshalDetails(); err != nil {
		return nil, fmt.Errorf("failed to unmarshal details: %w", err)
	}

	return &activity, nil
}

// GetAll retrieves all activity logs with optional filtering
func (s *ActivityService) GetAll(status string, taskType string, limit int) ([]models.ActivityLog, error) {
	query := `
		SELECT id, task_type, status, message, progress, started_at, completed_at, details
		FROM activity_logs
		WHERE 1=1
	`
	args := []interface{}{}

	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}

	if taskType != "" {
		query += " AND task_type = ?"
		args = append(args, taskType)
	}

	query += " ORDER BY started_at DESC"

	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query activity logs: %w", err)
	}
	defer rows.Close()

	var activities []models.ActivityLog
	for rows.Next() {
		var activity models.ActivityLog
		err := rows.Scan(
			&activity.ID, &activity.TaskType, &activity.Status, &activity.Message,
			&activity.Progress, &activity.StartedAt, &activity.CompletedAt, &activity.Details,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan activity log: %w", err)
		}

		// Unmarshal details
		if err := activity.UnmarshalDetails(); err != nil {
			// Log error but don't fail the whole query
			activity.DetailsObj = nil
		}

		activities = append(activities, activity)
	}

	return activities, nil
}

// Update updates an existing activity log
func (s *ActivityService) Update(id int64, update *models.ActivityLogUpdate) (*models.ActivityLog, error) {
	// Get existing activity
	activity, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Update fields if provided
	if update.Status != nil {
		activity.Status = *update.Status
	}
	if update.Message != nil {
		activity.Message = *update.Message
	}
	if update.Progress != nil {
		activity.Progress = *update.Progress
	}
	if update.Details != nil {
		activity.DetailsObj = update.Details
	}
	if update.Completed {
		now := time.Now()
		activity.CompletedAt = &now
	}

	// Marshal details
	if err := activity.MarshalDetails(); err != nil {
		return nil, fmt.Errorf("failed to marshal details: %w", err)
	}

	query := `
		UPDATE activity_logs
		SET status = ?, message = ?, progress = ?, completed_at = ?, details = ?
		WHERE id = ?
	`

	_, err = s.db.Exec(query, activity.Status, activity.Message, activity.Progress,
		activity.CompletedAt, activity.Details, id)
	if err != nil {
		return nil, fmt.Errorf("failed to update activity log: %w", err)
	}

	return activity, nil
}

// Delete deletes an activity log
func (s *ActivityService) Delete(id int64) error {
	query := `DELETE FROM activity_logs WHERE id = ?`
	result, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete activity log: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("activity log not found")
	}

	return nil
}

// GetStatus returns the current status of all activities
func (s *ActivityService) GetStatus() (*models.ActivityStatus, error) {
	status := &models.ActivityStatus{}

	// Count by status
	err := s.db.QueryRow("SELECT COUNT(*) FROM activity_logs WHERE status = ?", models.TaskStatusRunning).Scan(&status.RunningTasks)
	if err != nil {
		status.RunningTasks = 0
	}

	err = s.db.QueryRow("SELECT COUNT(*) FROM activity_logs WHERE status = ?", models.TaskStatusPending).Scan(&status.PendingTasks)
	if err != nil {
		status.PendingTasks = 0
	}

	err = s.db.QueryRow("SELECT COUNT(*) FROM activity_logs WHERE status = ?", models.TaskStatusCompleted).Scan(&status.CompletedTasks)
	if err != nil {
		status.CompletedTasks = 0
	}

	err = s.db.QueryRow("SELECT COUNT(*) FROM activity_logs WHERE status = ?", models.TaskStatusFailed).Scan(&status.FailedTasks)
	if err != nil {
		status.FailedTasks = 0
	}

	// Get current running tasks
	currentTasks, err := s.GetAll(models.TaskStatusRunning, "", 10)
	if err != nil {
		currentTasks = []models.ActivityLog{}
	}
	status.CurrentTasks = currentTasks

	return status, nil
}

// GetRecent retrieves the most recent activity logs
func (s *ActivityService) GetRecent(limit int) ([]models.ActivityLog, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.GetAll("", "", limit)
}

// CleanOld removes old completed activity logs
func (s *ActivityService) CleanOld(daysOld int) (int64, error) {
	if daysOld <= 0 {
		daysOld = 30
	}

	cutoffDate := time.Now().AddDate(0, 0, -daysOld)

	query := `
		DELETE FROM activity_logs
		WHERE status IN (?, ?)
		AND completed_at < ?
	`

	result, err := s.db.Exec(query, models.TaskStatusCompleted, models.TaskStatusFailed, cutoffDate)
	if err != nil {
		return 0, fmt.Errorf("failed to clean old logs: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return count, nil
}

// GetStatsByType returns activity counts grouped by task type
func (s *ActivityService) GetStatsByType() (map[string]int, error) {
	query := `
		SELECT task_type, COUNT(*) as count
		FROM activity_logs
		GROUP BY task_type
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query stats: %w", err)
	}
	defer rows.Close()

	stats := make(map[string]int)
	for rows.Next() {
		var taskType string
		var count int
		if err := rows.Scan(&taskType, &count); err != nil {
			continue
		}
		stats[taskType] = count
	}

	return stats, nil
}

// StartTask is a helper to create and start a new task
func (s *ActivityService) StartTask(taskType, message string, details map[string]interface{}) (*models.ActivityLog, error) {
	create := &models.ActivityLogCreate{
		TaskType: taskType,
		Status:   models.TaskStatusRunning,
		Message:  message,
		Progress: 0,
		Details:  details,
	}
	return s.Create(create)
}

// CompleteTask is a helper to mark a task as completed
func (s *ActivityService) CompleteTask(id int64, message string) error {
	update := &models.ActivityLogUpdate{
		Status:    stringPtr(models.TaskStatusCompleted),
		Message:   stringPtr(message),
		Progress:  intPtr(100),
		Completed: true,
	}
	_, err := s.Update(id, update)
	return err
}

// FailTask is a helper to mark a task as failed
func (s *ActivityService) FailTask(id int64, errorMsg string) error {
	update := &models.ActivityLogUpdate{
		Status:    stringPtr(models.TaskStatusFailed),
		Message:   stringPtr(errorMsg),
		Completed: true,
	}
	_, err := s.Update(id, update)
	return err
}

// UpdateProgress is a helper to update task progress
func (s *ActivityService) UpdateProgress(id int64, progress int, message string) error {
	update := &models.ActivityLogUpdate{
		Progress: intPtr(progress),
		Message:  stringPtr(message),
	}
	_, err := s.Update(id, update)
	return err
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

// GetTasksByStatus retrieves tasks filtered by status with pagination
func (s *ActivityService) GetTasksByStatus(status string, limit, offset int) ([]models.ActivityLog, int, error) {
	// Get total count
	var total int
	countQuery := "SELECT COUNT(*) FROM activity_logs WHERE status = ?"
	err := s.db.QueryRow(countQuery, status).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count tasks: %w", err)
	}

	// Get tasks
	query := `
		SELECT id, task_type, status, message, progress, started_at, completed_at, details
		FROM activity_logs
		WHERE status = ?
		ORDER BY started_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := s.db.Query(query, status, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query tasks: %w", err)
	}
	defer rows.Close()

	var activities []models.ActivityLog
	for rows.Next() {
		var activity models.ActivityLog
		err := rows.Scan(
			&activity.ID, &activity.TaskType, &activity.Status, &activity.Message,
			&activity.Progress, &activity.StartedAt, &activity.CompletedAt, &activity.Details,
		)
		if err != nil {
			continue
		}

		// Unmarshal details
		activity.UnmarshalDetails()
		activities = append(activities, activity)
	}

	return activities, total, nil
}

// BroadcastUpdate sends a real-time update (placeholder for WebSocket integration)
func (s *ActivityService) BroadcastUpdate(activity *models.ActivityLog) error {
	// This will be implemented when we add WebSocket support
	// For now, we'll just log that an update should be broadcast
	data, _ := json.Marshal(activity)
	fmt.Printf("Broadcasting activity update: %s\n", string(data))
	return nil
}
