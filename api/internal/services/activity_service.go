package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/brixen96/video-storage-ai/internal/database"
	"github.com/brixen96/video-storage-ai/internal/models"
)

// ActivityService handles activity logging and monitoring
type ActivityService struct {
	db *sql.DB
}

// WebSocketHub interface for broadcasting updates
type WebSocketHub interface {
	BroadcastActivityUpdate(activity *models.Activity)
	BroadcastStatusUpdate(status *models.ActivityStatus)
	BroadcastSystemEvent(event string)
	BroadcastConsoleLog(log *models.ConsoleLog)
	BroadcastLinkVerificationComplete(threadID int64, active, dead, expired, total int)
}

// Global WebSocket hub reference (will be set by API layer)
var wsHub WebSocketHub

// SetWebSocketHub sets the WebSocket hub for broadcasting
func SetWebSocketHub(hub WebSocketHub) {
	wsHub = hub
}

// NewActivityService creates a new activity service
func NewActivityService() *ActivityService {
	return &ActivityService{
		db: database.GetDB(),
	}
}

// Create creates a new activity log
func (s *ActivityService) Create(create *models.ActivityLogCreate) (*models.Activity, error) {
	var detailsJSON []byte
	var err error
	if create.Details != nil {
		detailsJSON, err = json.Marshal(create.Details)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal details: %w", err)
		}
	}

	query := `
		INSERT INTO activity_logs (task_type, status, message, progress, details, started_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	result, err := s.db.Exec(query, create.TaskType, create.Status, create.Message, create.Progress, detailsJSON, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create activity: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert ID: %w", err)
	}

	// Get the created activity
	activity, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Broadcast the new activity so Active Tasks panel updates immediately
	if err := s.BroadcastUpdate(activity); err != nil {
		log.Printf("failed to broadcast activity update: %v", err)
	}

	// Also broadcast status update for the counts
	if err := s.BroadcastStatusUpdate(); err != nil {
		log.Printf("failed to broadcast status update: %v", err)
	}

	return activity, nil
}

// GetByID retrieves an activity log by ID
func (s *ActivityService) GetByID(id int64) (*models.Activity, error) {
	query := `
		SELECT id, task_type, status, message, progress, details, started_at, updated_at, completed_at,
		       is_paused, paused_at, checkpoint
		FROM activity_logs
		WHERE id = ?
	`

	var activity models.Activity
	var detailsJSON []byte
	var checkpointJSON []byte
	var completedAt sql.NullTime
	var pausedAt sql.NullTime

	err := s.db.QueryRow(query, id).Scan(
		&activity.ID,
		&activity.TaskType,
		&activity.Status,
		&activity.Message,
		&activity.Progress,
		&detailsJSON,
		&activity.StartedAt,
		&activity.UpdatedAt,
		&completedAt,
		&activity.IsPaused,
		&pausedAt,
		&checkpointJSON,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("activity not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get activity: %w", err)
	}

	if completedAt.Valid {
		activity.CompletedAt = &completedAt.Time
	}

	if pausedAt.Valid {
		activity.PausedAt = &pausedAt.Time
	}

	// Details is stored as JSON string in the database
	if len(detailsJSON) > 0 {
		activity.Details = string(detailsJSON)
		activity.UnmarshalDetails()
	}

	// Checkpoint is stored as JSON string in the database
	if len(checkpointJSON) > 0 {
		activity.Checkpoint = string(checkpointJSON)
		activity.UnmarshalCheckpoint()
	}

	return &activity, nil
}

// GetAll retrieves all activity logs with optional filtering
func (s *ActivityService) GetAll(status string, taskType string, limit int) ([]models.Activity, error) {
	query := `
		SELECT id, task_type, status, message, progress, details, started_at, updated_at, completed_at, error,
		       is_paused, paused_at, checkpoint
		FROM activity_logs
		WHERE 1=1
	`
	args := []interface{}{}

	if status != "" {
		// Support comma-separated status values (e.g., "running,paused")
		statuses := strings.Split(status, ",")
		if len(statuses) == 1 {
			query += " AND status = ?"
			args = append(args, status)
		} else {
			placeholders := make([]string, len(statuses))
			for i, s := range statuses {
				placeholders[i] = "?"
				args = append(args, strings.TrimSpace(s))
			}
			query += " AND status IN (" + strings.Join(placeholders, ",") + ")"
		}
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
		return nil, fmt.Errorf("failed to query activities: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("failed to close rows: %v", err)
		}
	}()

	activities := make([]models.Activity, 0) // Initialize as empty slice, not nil
	for rows.Next() {
		var activity models.Activity
		var detailsJSON []byte
		var checkpointJSON []byte
		var completedAt sql.NullTime
		var pausedAt sql.NullTime
		var errorMsg sql.NullString

		err := rows.Scan(
			&activity.ID,
			&activity.TaskType,
			&activity.Status,
			&activity.Message,
			&activity.Progress,
			&detailsJSON,
			&activity.StartedAt,
			&activity.UpdatedAt,
			&completedAt,
			&errorMsg,
			&activity.IsPaused,
			&pausedAt,
			&checkpointJSON,
		)
		if err != nil {
			continue
		}

		if completedAt.Valid {
			activity.CompletedAt = &completedAt.Time
		}

		if pausedAt.Valid {
			activity.PausedAt = &pausedAt.Time
		}

		if errorMsg.Valid {
			activity.Error = &errorMsg.String
		}

		// Details is stored as JSON string in the database
		if len(detailsJSON) > 0 {
			activity.Details = string(detailsJSON)
			// Unmarshal to DetailsObj for JSON serialization
			if err := activity.UnmarshalDetails(); err != nil {
				log.Printf("failed to unmarshal activity details: %v", err)
			}
		}

		// Checkpoint is stored as JSON string in the database
		if len(checkpointJSON) > 0 {
			activity.Checkpoint = string(checkpointJSON)
			// Unmarshal to CheckpointObj for JSON serialization
			if err := activity.UnmarshalCheckpoint(); err != nil {
				log.Printf("failed to unmarshal activity checkpoint: %v", err)
			}
		}

		activities = append(activities, activity)
	}

	return activities, nil
}

// Update updates an existing activity log
func (s *ActivityService) Update(id int, update *models.ActivityLogUpdate) (*models.Activity, error) {
	// Build dynamic update query
	query := "UPDATE activity_logs SET updated_at = ?"
	args := []interface{}{time.Now()}

	if update.Status != nil {
		query += ", status = ?"
		args = append(args, *update.Status)
	}

	if update.Message != nil {
		query += ", message = ?"
		args = append(args, *update.Message)
	}

	if update.Progress != nil {
		query += ", progress = ?"
		args = append(args, *update.Progress)
	}

	if update.Details != nil {
		detailsJSON, err := json.Marshal(update.Details)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal details: %w", err)
		}
		query += ", details = ?"
		args = append(args, detailsJSON)
	}

	// If status is completed or failed, set completed_at
	if update.Status != nil && (*update.Status == models.TaskStatusCompleted || *update.Status == models.TaskStatusFailed) {
		query += ", completed_at = ?"
		args = append(args, time.Now())
	}

	query += " WHERE id = ?"
	args = append(args, id)

	_, err := s.db.Exec(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update activity: %w", err)
	}

	// Get the updated activity
	activity, err := s.GetByID(int64(id))
	if err != nil {
		return nil, err
	}

	// Broadcast the individual activity update
	if err := s.BroadcastUpdate(activity); err != nil {
		log.Printf("failed to broadcast activity update: %v", err)
	}

	// Broadcast the overall status update
	if err := s.BroadcastStatusUpdate(); err != nil {
		log.Printf("failed to broadcast status update: %v", err)
	}

	return activity, nil
}

// Delete deletes an activity log
func (s *ActivityService) Delete(id int64) error {
	query := "DELETE FROM activity_logs WHERE id = ?"
	result, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete activity: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("activity not found")
	}

	if err := s.BroadcastStatusUpdate(); err != nil {
		log.Printf("failed to broadcast status update: %v", err)
	}

	return nil
}

// GetStatus returns the current status of all activities
func (s *ActivityService) GetStatus() (*models.ActivityStatus, error) {
	status := &models.ActivityStatus{}

	// Count tasks by status
	query := `
        SELECT 
            COUNT(CASE WHEN status = ? THEN 1 END) as running,
            COUNT(CASE WHEN status = ? THEN 1 END) as pending,
            COUNT(CASE WHEN status = ? THEN 1 END) as completed,
            COUNT(CASE WHEN status = ? THEN 1 END) as failed
        FROM activity_logs
    `

	err := s.db.QueryRow(query,
		models.TaskStatusRunning,
		models.TaskStatusPending,
		models.TaskStatusCompleted,
		models.TaskStatusFailed,
	).Scan(&status.RunningTasks, &status.PendingTasks, &status.CompletedTasks, &status.FailedTasks)

	if err != nil {
		return nil, fmt.Errorf("failed to get activity status: %w", err)
	}

	// Get current running tasks
	currentQuery := `
        SELECT id, task_type, status, message, progress, started_at, completed_at, details
        FROM activity_logs
        WHERE status = ?
        ORDER BY started_at DESC
        LIMIT 10
    `

	rows, err := s.db.Query(currentQuery, models.TaskStatusRunning)
	if err != nil {
		return status, nil // Return status even if current tasks query fails
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("failed to close rows: %v", err)
		}
	}()

	status.CurrentTasks = make([]models.ActivityLog, 0)
	for rows.Next() {
		var activity models.ActivityLog
		var completedAt sql.NullTime
		err := rows.Scan(
			&activity.ID,
			&activity.TaskType,
			&activity.Status,
			&activity.Message,
			&activity.Progress,
			&activity.StartedAt,
			&completedAt,
			&activity.Details,
		)
		if err != nil {
			continue
		}

		if completedAt.Valid {
			activity.CompletedAt = &completedAt.Time
		}

		// Unmarshal details
		if err := activity.UnmarshalDetails(); err != nil {
			log.Printf("failed to unmarshal details: %v", err)
		}

		status.CurrentTasks = append(status.CurrentTasks, activity)
	}

	return status, nil
}

// GetRecent retrieves the most recent activity logs
func (s *ActivityService) GetRecent(limit int) ([]models.Activity, error) {
	return s.GetAll("", "", limit)
}

// CleanOld removes old completed activity logs
func (s *ActivityService) CleanOld(daysOld int) (int64, error) {
	query := `
		DELETE FROM activity_logs
		WHERE status IN (?, ?)
		AND completed_at < datetime('now', '-' || ? || ' days')
	`

	result, err := s.db.Exec(query, models.TaskStatusCompleted, models.TaskStatusFailed, daysOld)
	if err != nil {
		return 0, fmt.Errorf("failed to clean old activities: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if err := s.BroadcastStatusUpdate(); err != nil {
		log.Printf("failed to broadcast status update: %v", err)
	}

	return count, nil
}

// ClearAll removes all activity logs
func (s *ActivityService) ClearAll() (int64, error) {
	query := "DELETE FROM activity_logs"

	result, err := s.db.Exec(query)
	if err != nil {
		return 0, fmt.Errorf("failed to clear all activities: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if err := s.BroadcastStatusUpdate(); err != nil {
		log.Printf("failed to broadcast status update: %v", err)
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
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("failed to close rows: %v", err)
		}
	}()

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
func (s *ActivityService) StartTask(taskType, message string, details map[string]interface{}) (*models.Activity, error) {
	activity := &models.Activity{
		TaskType:  taskType,
		Status:    models.TaskStatusRunning,
		Message:   message,
		Progress:  0,
	}

	// Marshal details to JSON string
	var detailsJSON string
	if len(details) > 0 {
		detailsBytes, err := json.Marshal(details)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal task details: %w", err)
		}
		detailsJSON = string(detailsBytes)
	} else {
		detailsJSON = "{}" // Empty JSON object
	}
	activity.Details = detailsJSON

	query := `
        INSERT INTO activity_logs (task_type, status, message, details, progress, started_at, updated_at, completed_at, error)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
    `

	now := time.Now()
	activity.StartedAt = now

	result, err := s.db.Exec(
		query,
		activity.TaskType,
		activity.Status,
		activity.Message,
		detailsJSON,
		0,   // progress
		now, // started_at
		now, // updated_at
		nil, // completed_at
		nil, // error
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create activity: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get activity ID: %w", err)
	}

	activity.ID = int(id)

	// Broadcast update
	if err := s.BroadcastUpdate(activity); err != nil {
		log.Printf("failed to broadcast activity update: %v", err)
	}

	return s.GetByID(int64(id))
}

// CompleteTask is a helper to mark a task as completed
func (s *ActivityService) CompleteTask(id int64, message string) error {
	status := models.TaskStatusCompleted
	progress := 100
	update := &models.ActivityLogUpdate{
		Status:   &status,
		Message:  &message,
		Progress: &progress,
	}

	_, err := s.Update(int(id), update)
	if err == nil {
		s.checkAndBroadcastIdle()
	}
	return err
}

// FailTask is a helper to mark a task as failed
func (s *ActivityService) FailTask(id int, errorMsg string) error {
	status := models.TaskStatusFailed
	update := &models.ActivityLogUpdate{
		Status:  &status,
		Message: &errorMsg,
	}

	_, err := s.Update(id, update)
	if err == nil {
		s.checkAndBroadcastIdle()
	}
	return err
}

// UpdateProgress is a helper to update task progress
func (s *ActivityService) UpdateProgress(id int, progress int, message string) error {
	update := &models.ActivityLogUpdate{
		Progress: &progress,
		Message:  &message,
	}

	_, err := s.Update(id, update)
	return err
}

// checkAndBroadcastIdle checks if there are any active tasks and broadcasts an "idle" event if not.
func (s *ActivityService) checkAndBroadcastIdle() {
	status, err := s.GetStatus()
	if err != nil {
		log.Printf("Error getting status for idle check: %v", err)
		return
	}

	if status.RunningTasks == 0 && status.PendingTasks == 0 {
		if wsHub != nil {
			wsHub.BroadcastSystemEvent("idle")
		}
	}
}

// GetTasksByStatus retrieves tasks filtered by status with pagination
func (s *ActivityService) GetTasksByStatus(status string, limit, offset int) ([]models.Activity, int, error) {
	// Get total count
	var total int
	countQuery := "SELECT COUNT(*) FROM activity_logs WHERE status = ?"
	err := s.db.QueryRow(countQuery, status).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count tasks: %w", err)
	}

	// Get tasks
	query := `
		SELECT id, task_type, status, message, progress, details, started_at, updated_at, completed_at, error
		FROM activity_logs
		WHERE status = ?
		ORDER BY started_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := s.db.Query(query, status, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query tasks: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("failed to close rows: %v", err)
		}
	}()

	var activities []models.Activity
	for rows.Next() {
		var activity models.Activity
		var detailsJSON []byte
		var completedAt sql.NullTime
		var errorMsg sql.NullString

		err := rows.Scan(
			&activity.ID,
			&activity.TaskType,
			&activity.Status,
			&activity.Message,
			&activity.Progress,
			&detailsJSON,
			&activity.StartedAt,
			&activity.UpdatedAt,
			&completedAt,
			&errorMsg,
		)
		if err != nil {
			continue
		}

		if completedAt.Valid {
			activity.CompletedAt = &completedAt.Time
		}

		if errorMsg.Valid {
			activity.Error = &errorMsg.String
		}

		// Details is stored as JSON string in the database
		if len(detailsJSON) > 0 {
			activity.Details = string(detailsJSON)
		}

		activities = append(activities, activity)
	}

	return activities, total, nil
}

// BroadcastUpdate sends a real-time update via WebSocket
func (s *ActivityService) BroadcastUpdate(activity *models.Activity) error {
	hub := wsHub
	if hub != nil {
		hub.BroadcastActivityUpdate(activity)
	}
	return nil
}

// Create creates a new activity log entry
func (s *ActivityService) CreateLog(create *models.ActivityLogCreate) (*models.ActivityLog, error) {
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

	// Broadcast the new activity via WebSocket
	s.BroadcastUpdateLog(activity)
	s.BroadcastStatusUpdate()

	return activity, nil
}

// GetByID retrieves an activity log by ID
func (s *ActivityService) GetLogByID(id int64) (*models.ActivityLog, error) {
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
func (s *ActivityService) GetAllLogs(status string, taskType string, limit int) ([]models.ActivityLog, error) {
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
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("failed to close rows: %v", err)
		}
	}()

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
func (s *ActivityService) UpdateLog(id int64, update *models.ActivityLogUpdate) (*models.ActivityLog, error) {
	// Get existing activity
	activity, err := s.GetLogByID(id)
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

	// Broadcast the update via WebSocket
	s.BroadcastUpdateLog(activity)
	if update.Completed {
		s.BroadcastStatusUpdate()
	}

	return activity, nil
}

// Delete deletes an activity log
func (s *ActivityService) DeleteLog(id int64) error {
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
func (s *ActivityService) GetStatusAll() (*models.ActivityStatus, error) {
	status := &models.ActivityStatus{}

	// Count all statuses in a single query
	err := s.db.QueryRow(`
		SELECT
			COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0)
		FROM activity_logs
	`, models.TaskStatusRunning, models.TaskStatusPending, models.TaskStatusCompleted, models.TaskStatusFailed).Scan(
		&status.RunningTasks,
		&status.PendingTasks,
		&status.CompletedTasks,
		&status.FailedTasks,
	)
	if err != nil {
		status.RunningTasks = 0
		status.PendingTasks = 0
		status.CompletedTasks = 0
		status.FailedTasks = 0
	}

	// Get current running tasks
	currentTasks, err := s.GetAllLogs(models.TaskStatusRunning, "", 10)
	if err != nil {
		currentTasks = []models.ActivityLog{}
	}
	status.CurrentTasks = currentTasks

	return status, nil
}

// GetRecent retrieves the most recent activity logs
func (s *ActivityService) GetRecentLogs(limit int) ([]models.ActivityLog, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.GetAllLogs("", "", limit)
}

// CleanOld removes old completed activity logs
func (s *ActivityService) CleanOldLogs(daysOld int) (int64, error) {
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
func (s *ActivityService) GetStatsByTypeLogs() (map[string]int, error) {
	query := `
		SELECT task_type, COUNT(*) as count
		FROM activity_logs
		GROUP BY task_type
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query stats: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("failed to close rows: %v", err)
		}
	}()

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
func (s *ActivityService) StartTaskLog(taskType, message string, details map[string]interface{}) (*models.ActivityLog, error) {
	create := &models.ActivityLogCreate{
		TaskType: taskType,
		Status:   models.TaskStatusRunning,
		Message:  message,
		Progress: 0,
		Details:  details,
	}
	return s.CreateLog(create)
}

// CompleteTask is a helper to mark a task as completed
func (s *ActivityService) CompleteTaskLog(id int64, message string) error {
	update := &models.ActivityLogUpdate{
		Status:    stringPtr(models.TaskStatusCompleted),
		Message:   stringPtr(message),
		Progress:  intPtr(100),
		Completed: true,
	}
	_, err := s.UpdateLog(id, update)
	return err
}

// FailTask is a helper to mark a task as failed
func (s *ActivityService) FailTaskLog(id int64, errorMsg string) error {
	update := &models.ActivityLogUpdate{
		Status:    stringPtr(models.TaskStatusFailed),
		Message:   stringPtr(errorMsg),
		Completed: true,
	}
	_, err := s.UpdateLog(id, update)
	return err
}

// UpdateProgress is a helper to update task progress
func (s *ActivityService) UpdateProgressLog(id int64, progress int, message string) error {
	update := &models.ActivityLogUpdate{
		Progress: intPtr(progress),
		Message:  stringPtr(message),
	}
	_, err := s.UpdateLog(id, update)
	return err
}

// IsActivityPaused returns true when the given activity log has been paused by
// the user. Safe to call with activityID = 0 (returns false).
// Scraper services call this once per page to honour user pause requests.
func IsActivityPaused(activityID int64) bool {
	if activityID <= 0 {
		return false
	}
	svc := NewActivityService()
	activity, err := svc.GetByID(activityID)
	return err == nil && activity.IsPaused
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

// BroadcastUpdateLog sends a real-time update via WebSocket
func (s *ActivityService) BroadcastUpdateLog(activityLog *models.ActivityLog) error {
	// Convert ActivityLog to Activity for broadcasting
	activity := &models.Activity{
		ID:          int(activityLog.ID),
		TaskType:    activityLog.TaskType,
		Status:      activityLog.Status,
		Message:     activityLog.Message,
		Progress:    activityLog.Progress,
		StartedAt:   activityLog.StartedAt,
		CompletedAt: activityLog.CompletedAt,
		UpdatedAt:   time.Now(),
	}

	hub := wsHub
	if hub != nil {
		hub.BroadcastActivityUpdate(activity)
	}
	return nil
}

// BroadcastStatusUpdate sends a real-time status update via WebSocket
func (s *ActivityService) BroadcastStatusUpdate() error {
	status, err := s.GetStatus()
	if err != nil {
		return err
	}

	hub := wsHub
	if hub != nil {
		hub.BroadcastStatusUpdate(status)
	}
	return nil
}

// PauseTask pauses a running task and preserves its checkpoint state
// Note: checkpoint parameter is ignored - we preserve the existing checkpoint
// that was saved by SaveCheckpoint during task execution
func (s *ActivityService) PauseTask(id int64, checkpoint map[string]interface{}) error {
	now := time.Now()

	// Only update is_paused and paused_at - DO NOT touch checkpoint!
	// The checkpoint is saved periodically by SaveCheckpoint during task execution
	// and must be preserved when pausing
	query := `
		UPDATE activity_logs
		SET is_paused = 1, paused_at = ?
		WHERE id = ? AND status = 'running'
	`

	result, err := s.db.Exec(query, now, id)
	if err != nil {
		return fmt.Errorf("failed to pause task: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("task not found or not running")
	}

	// Get updated activity for broadcast
	activity, err := s.GetByID(id)
	if err == nil && activity != nil {
		log.Printf("✅ Task %d paused successfully (checkpoint preserved: %s)", id, activity.Checkpoint)
		s.BroadcastUpdate(activity)
		s.BroadcastStatusUpdate()
	}

	return nil
}

// ResumeTask resumes a paused task and returns the checkpoint data
func (s *ActivityService) ResumeTask(id int64) (map[string]interface{}, error) {
	// First get the task to verify it's paused and get checkpoint
	activity, err := s.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	log.Printf("🔄 ResumeTask %d - IsPaused: %v, Raw checkpoint: %s", id, activity.IsPaused, activity.Checkpoint)

	if !activity.IsPaused {
		return nil, fmt.Errorf("task is not paused")
	}

	// Unmarshal checkpoint before clearing it
	if err := activity.UnmarshalCheckpoint(); err != nil {
		return nil, fmt.Errorf("failed to unmarshal checkpoint: %w", err)
	}

	// Store checkpoint data before we clear it
	checkpointData := activity.CheckpointObj
	log.Printf("🔄 ResumeTask %d - Parsed checkpoint: %+v", id, checkpointData)

	// Clear paused state but keep the checkpoint for reference
	query := `
		UPDATE activity_logs
		SET is_paused = 0, paused_at = NULL
		WHERE id = ?
	`

	_, err = s.db.Exec(query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to resume task: %w", err)
	}

	// Get updated activity for broadcast
	resumedActivity, err := s.GetByID(id)
	if err == nil && resumedActivity != nil {
		s.BroadcastUpdate(resumedActivity)
		s.BroadcastStatusUpdate()
	}

	log.Printf("✅ Task %d resumed with checkpoint: %+v", id, checkpointData)
	return checkpointData, nil
}

// SaveCheckpoint saves task checkpoint data without pausing
func (s *ActivityService) SaveCheckpoint(id int64, checkpoint map[string]interface{}) error {
	// Marshal checkpoint data
	checkpointJSON := "{}"
	if checkpoint != nil {
		data, err := json.Marshal(checkpoint)
		if err != nil {
			return fmt.Errorf("failed to marshal checkpoint: %w", err)
		}
		checkpointJSON = string(data)
	}

	query := `
		UPDATE activity_logs
		SET checkpoint = ?
		WHERE id = ?
	`

	result, err := s.db.Exec(query, checkpointJSON, id)
	if err != nil {
		return fmt.Errorf("failed to save checkpoint: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	log.Printf("💾 SaveCheckpoint for task %d: %s (rows affected: %d)", id, checkpointJSON, rowsAffected)

	return nil
}

// CancelTask cancels a running or paused task permanently
// Unlike pause, cancel marks the task as cancelled and cannot be resumed
func (s *ActivityService) CancelTask(id int64) error {
	// Get the task to verify it exists and can be cancelled
	activity, err := s.GetByID(id)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Can only cancel tasks that are running or paused
	if activity.Status != "running" {
		return fmt.Errorf("task cannot be cancelled (status: %s)", activity.Status)
	}

	// First pause the task to stop execution (this sets is_paused = 1)
	// The scraper will detect this and stop
	if !activity.IsPaused {
		err = s.PauseTask(id, nil)
		if err != nil {
			log.Printf("⚠️ Warning: failed to pause task %d before cancel: %v", id, err)
		}
	}

	// Then update status to cancelled
	cancelledStatus := "cancelled"
	message := "Task cancelled by user"
	_, err = s.Update(int(id), &models.ActivityLogUpdate{
		Status:    &cancelledStatus,
		Message:   &message,
		Completed: true,
	})
	if err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	log.Printf("❌ Task %d cancelled by user", id)

	// Get updated activity for broadcast
	cancelledActivity, err := s.GetByID(id)
	if err == nil && cancelledActivity != nil {
		s.BroadcastUpdate(cancelledActivity)
		s.BroadcastStatusUpdate()
	}

	return nil
}

// RetryTask resets a failed or cancelled task for retry
// Returns the activity with its details so the handler can restart the appropriate scraper
func (s *ActivityService) RetryTask(id int64) (*models.Activity, error) {
	// Get the task
	activity, err := s.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	// Can only retry failed or cancelled tasks
	if activity.Status != "failed" && activity.Status != "cancelled" {
		return nil, fmt.Errorf("task cannot be retried (status: %s)", activity.Status)
	}

	// Reset task to running state with 0 progress
	runningStatus := "running"
	progress0 := 0
	retryMessage := "Retrying task..."
	_, err = s.Update(int(id), &models.ActivityLogUpdate{
		Status:    &runningStatus,
		Progress:  &progress0,
		Message:   &retryMessage,
		Completed: false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to reset task status: %w", err)
	}

	// Clear paused state and checkpoint for fresh start
	query := `
		UPDATE activity_logs
		SET is_paused = 0, paused_at = NULL, checkpoint = '{}'
		WHERE id = ?
	`
	_, err = s.db.Exec(query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to clear task state: %w", err)
	}

	log.Printf("🔄 Task %d reset for retry", id)

	// Get updated activity with details for restart
	updatedActivity, err := s.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated task: %w", err)
	}

	// Broadcast update
	s.BroadcastUpdate(updatedActivity)
	s.BroadcastStatusUpdate()

	return updatedActivity, nil
}
