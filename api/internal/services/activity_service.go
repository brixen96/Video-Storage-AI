package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
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

	if err := s.BroadcastStatusUpdate(); err != nil {
		log.Printf("failed to broadcast status update: %v", err)
	}

	return s.GetByID(id)
}

// GetByID retrieves an activity log by ID
func (s *ActivityService) GetByID(id int64) (*models.Activity, error) {
	query := `
		SELECT id, task_type, status, message, progress, details, started_at, updated_at, completed_at
		FROM activity_logs
		WHERE id = ?
	`

	var activity models.Activity
	var detailsJSON []byte
	var completedAt sql.NullTime

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

	// Details is stored as JSON string in the database
	if len(detailsJSON) > 0 {
		activity.Details = string(detailsJSON)
	}

	return &activity, nil
}

// GetAll retrieves all activity logs with optional filtering
func (s *ActivityService) GetAll(status string, taskType string, limit int) ([]models.Activity, error) {
	query := `
		SELECT id, task_type, status, message, progress, details, started_at, updated_at, completed_at, error
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
		return nil, fmt.Errorf("failed to query activities: %w", err)
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
