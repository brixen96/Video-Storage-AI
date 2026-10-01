package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/brixen96/video-storage-ai/internal/models"
)

// CompanionTaskService handles task operations for the shared task board
type CompanionTaskService struct {
	db *sql.DB
}

// NewCompanionTaskService creates a new task service
func NewCompanionTaskService(db *sql.DB) *CompanionTaskService {
	return &CompanionTaskService{db: db}
}

// CreateTask creates a new task
func (s *CompanionTaskService) CreateTask(task *models.CompanionTask) error {
	if task.Title == "" {
		return fmt.Errorf("title is required")
	}
	if task.Status == "" {
		task.Status = models.CompanionTaskStatusOpen
	}
	if task.Priority == 0 {
		task.Priority = models.PriorityMedium
	}
	if task.CreatedBy == "" {
		task.CreatedBy = "companion"
	}

	tagsJSON, err := json.Marshal(task.Tags)
	if err != nil {
		tagsJSON = []byte("[]")
	}

	now := time.Now()
	task.CreatedAt = now
	task.UpdatedAt = now

	result, err := s.db.Exec(`
		INSERT INTO companion_tasks (title, description, status, priority, created_by, assigned_to, tags, related_file, related_line, error_context, resolution, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		task.Title, task.Description, task.Status, task.Priority, task.CreatedBy,
		task.AssignedTo, string(tagsJSON), task.RelatedFile, task.RelatedLine,
		task.ErrorContext, task.Resolution, task.CreatedAt, task.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get task ID: %w", err)
	}
	task.ID = id
	return nil
}

// GetTask retrieves a task by ID
func (s *CompanionTaskService) GetTask(id int64) (*models.CompanionTask, error) {
	task := &models.CompanionTask{}
	var tagsJSON string
	var completedAt sql.NullTime

	err := s.db.QueryRow(`
		SELECT id, title, description, status, priority, created_by, assigned_to,
		       tags, related_file, related_line, error_context, resolution,
		       created_at, updated_at, completed_at
		FROM companion_tasks WHERE id = ?`, id).Scan(
		&task.ID, &task.Title, &task.Description, &task.Status, &task.Priority,
		&task.CreatedBy, &task.AssignedTo, &tagsJSON, &task.RelatedFile,
		&task.RelatedLine, &task.ErrorContext, &task.Resolution,
		&task.CreatedAt, &task.UpdatedAt, &completedAt)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	if completedAt.Valid {
		task.CompletedAt = &completedAt.Time
	}
	json.Unmarshal([]byte(tagsJSON), &task.Tags)
	return task, nil
}

// ListTasks retrieves tasks with optional filters
func (s *CompanionTaskService) ListTasks(status string, createdBy string, limit int) ([]models.CompanionTask, error) {
	query := "SELECT id, title, description, status, priority, created_by, assigned_to, tags, related_file, related_line, error_context, resolution, created_at, updated_at, completed_at FROM companion_tasks WHERE 1=1"
	args := []interface{}{}

	if status != "" && status != "all" {
		query += " AND status = ?"
		args = append(args, status)
	}
	if createdBy != "" {
		query += " AND created_by = ?"
		args = append(args, createdBy)
	}

	query += " ORDER BY priority DESC, created_at DESC"

	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []models.CompanionTask
	for rows.Next() {
		var task models.CompanionTask
		var tagsJSON string
		var completedAt sql.NullTime

		if err := rows.Scan(&task.ID, &task.Title, &task.Description, &task.Status,
			&task.Priority, &task.CreatedBy, &task.AssignedTo, &tagsJSON,
			&task.RelatedFile, &task.RelatedLine, &task.ErrorContext, &task.Resolution,
			&task.CreatedAt, &task.UpdatedAt, &completedAt); err != nil {
			continue
		}

		if completedAt.Valid {
			task.CompletedAt = &completedAt.Time
		}
		json.Unmarshal([]byte(tagsJSON), &task.Tags)
		tasks = append(tasks, task)
	}
	return tasks, nil
}

// UpdateTaskStatus updates a task's status
func (s *CompanionTaskService) UpdateTaskStatus(id int64, status string, resolution string) error {
	now := time.Now()
	completedAt := sql.NullTime{}
	if status == models.CompanionTaskStatusCompleted {
		completedAt = sql.NullTime{Time: now, Valid: true}
	}

	_, err := s.db.Exec(`
		UPDATE companion_tasks SET status = ?, resolution = ?, updated_at = ?, completed_at = ?
		WHERE id = ?`, status, resolution, now, completedAt, id)
	return err
}

// UpdateTask updates task fields
func (s *CompanionTaskService) UpdateTask(id int64, updates map[string]interface{}) error {
	setClauses := []string{}
	args := []interface{}{}

	if v, ok := updates["title"].(string); ok {
		setClauses = append(setClauses, "title = ?")
		args = append(args, v)
	}
	if v, ok := updates["description"].(string); ok {
		setClauses = append(setClauses, "description = ?")
		args = append(args, v)
	}
	if v, ok := updates["status"].(string); ok {
		setClauses = append(setClauses, "status = ?")
		args = append(args, v)
	}
	if v, ok := updates["priority"].(float64); ok {
		setClauses = append(setClauses, "priority = ?")
		args = append(args, int(v))
	}
	if v, ok := updates["assigned_to"].(string); ok {
		setClauses = append(setClauses, "assigned_to = ?")
		args = append(args, v)
	}
	if v, ok := updates["resolution"].(string); ok {
		setClauses = append(setClauses, "resolution = ?")
		args = append(args, v)
	}

	if len(setClauses) == 0 {
		return fmt.Errorf("no fields to update")
	}

	setClauses = append(setClauses, "updated_at = ?")
	args = append(args, time.Now())
	args = append(args, id)

	query := fmt.Sprintf("UPDATE companion_tasks SET %s WHERE id = ?", joinTaskStrings(setClauses, ", "))
	_, err := s.db.Exec(query, args...)
	return err
}

// DeleteTask removes a task
func (s *CompanionTaskService) DeleteTask(id int64) error {
	_, err := s.db.Exec("DELETE FROM companion_tasks WHERE id = ?", id)
	return err
}

// GetTaskStats returns task statistics
func (s *CompanionTaskService) GetTaskStats() (map[string]interface{}, error) {
	stats := map[string]interface{}{}

	var total, open, inProgress, completed, archived int
	s.db.QueryRow("SELECT COUNT(*) FROM companion_tasks").Scan(&total)
	s.db.QueryRow("SELECT COUNT(*) FROM companion_tasks WHERE status = 'open'").Scan(&open)
	s.db.QueryRow("SELECT COUNT(*) FROM companion_tasks WHERE status = 'in_progress'").Scan(&inProgress)
	s.db.QueryRow("SELECT COUNT(*) FROM companion_tasks WHERE status = 'completed'").Scan(&completed)
	s.db.QueryRow("SELECT COUNT(*) FROM companion_tasks WHERE status = 'archived'").Scan(&archived)

	stats["total"] = total
	stats["open"] = open
	stats["in_progress"] = inProgress
	stats["completed"] = completed
	stats["archived"] = archived

	return stats, nil
}

// joinStrings joins strings with a separator (internal helper)
func joinTaskStrings(strs []string, sep string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}