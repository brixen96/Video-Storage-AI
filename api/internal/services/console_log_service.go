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

// ConsoleLogService handles console logging
type ConsoleLogService struct {
	db *sql.DB
}

// NewConsoleLogService creates a new console log service
func NewConsoleLogService() *ConsoleLogService {
	return &ConsoleLogService{
		db: database.GetDB(),
	}
}

// LogEntry creates a new console log entry
func (s *ConsoleLogService) LogEntry(source, level, message string, details map[string]interface{}) error {
	var detailsJSON []byte
	var err error

	if details != nil {
		detailsJSON, err = json.Marshal(details)
		if err != nil {
			return fmt.Errorf("failed to marshal details: %w", err)
		}
	} else {
		detailsJSON = []byte("{}")
	}

	query := `
		INSERT INTO console_logs (source, level, message, details, created_at)
		VALUES (?, ?, ?, ?, ?)
	`

	_, err = s.db.Exec(query, source, level, message, detailsJSON, time.Now())
	if err != nil {
		return fmt.Errorf("failed to create console log: %w", err)
	}

	// Broadcast to WebSocket if available
	if wsHub != nil {
		// TODO: Add WebSocket broadcast for console logs
		// wsHub.BroadcastConsoleLog(...)
	}

	return nil
}

// GetAll retrieves console logs with filters and pagination
func (s *ConsoleLogService) GetAll(limit, offset int, source, level, search string) ([]*models.ConsoleLog, int, error) {
	// Build WHERE clause
	whereClause := "WHERE 1=1"
	args := []interface{}{}

	if source != "" {
		whereClause += " AND source = ?"
		args = append(args, source)
	}

	if level != "" {
		whereClause += " AND level = ?"
		args = append(args, level)
	}

	if search != "" {
		whereClause += " AND message LIKE ?"
		args = append(args, "%"+search+"%")
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM console_logs " + whereClause
	var total int
	err := s.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get total count: %w", err)
	}

	// Get logs with pagination
	query := "SELECT id, source, level, message, details, created_at FROM console_logs " +
		whereClause + " " +
		"ORDER BY created_at DESC " +
		"LIMIT ? OFFSET ?"

	args = append(args, limit, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query console logs: %w", err)
	}
	defer rows.Close()

	logs := []*models.ConsoleLog{}
	for rows.Next() {
		cl := &models.ConsoleLog{}
		err := rows.Scan(&cl.ID, &cl.Source, &cl.Level, &cl.Message, &cl.Details, &cl.CreatedAt)
		if err != nil {
			log.Printf("error scanning console log row: %v", err)
			continue
		}

		// Parse details JSON
		if err := cl.ParseDetails(); err != nil {
			log.Printf("error parsing details for log %d: %v", cl.ID, err)
		}

		logs = append(logs, cl)
	}

	return logs, total, nil
}

// GetByID retrieves a console log by ID
func (s *ConsoleLogService) GetByID(id int64) (*models.ConsoleLog, error) {
	query := `
		SELECT id, source, level, message, details, created_at
		FROM console_logs
		WHERE id = ?
	`

	cl := &models.ConsoleLog{}
	err := s.db.QueryRow(query, id).Scan(&cl.ID, &cl.Source, &cl.Level, &cl.Message, &cl.Details, &cl.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("console log not found")
		}
		return nil, fmt.Errorf("failed to get console log: %w", err)
	}

	if err := cl.ParseDetails(); err != nil {
		log.Printf("error parsing details for log %d: %v", cl.ID, err)
	}

	return cl, nil
}

// Delete deletes a console log by ID
func (s *ConsoleLogService) Delete(id int64) error {
	query := `DELETE FROM console_logs WHERE id = ?`

	result, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete console log: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("console log not found")
	}

	return nil
}

// DeleteAll deletes all console logs
func (s *ConsoleLogService) DeleteAll() (int64, error) {
	query := `DELETE FROM console_logs`

	result, err := s.db.Exec(query)
	if err != nil {
		return 0, fmt.Errorf("failed to delete all console logs: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}

// DeleteOlderThan deletes console logs older than a specified duration
func (s *ConsoleLogService) DeleteOlderThan(days int) (int64, error) {
	query := `DELETE FROM console_logs WHERE created_at < datetime('now', '-' || ? || ' days')`

	result, err := s.db.Exec(query, days)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old console logs: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}

// Helper functions for logging from different sources

// LogAPI logs an API-related message
func (s *ConsoleLogService) LogAPI(level, message string, details map[string]interface{}) error {
	return s.LogEntry("api", level, message, details)
}

// LogAICompanion logs an AI Companion-related message
func (s *ConsoleLogService) LogAICompanion(level, message string, details map[string]interface{}) error {
	return s.LogEntry("ai_companion", level, message, details)
}

// LogFrontend logs a frontend-related message
func (s *ConsoleLogService) LogFrontend(level, message string, details map[string]interface{}) error {
	return s.LogEntry("frontend", level, message, details)
}

// GetStats returns statistics about console logs
func (s *ConsoleLogService) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get total count
	var total int
	err := s.db.QueryRow("SELECT COUNT(*) FROM console_logs").Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}
	stats["total"] = total

	// Get count by source
	sourceQuery := `
		SELECT source, COUNT(*) as count
		FROM console_logs
		GROUP BY source
	`
	sourceRows, err := s.db.Query(sourceQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get source counts: %w", err)
	}
	defer sourceRows.Close()

	sourceCounts := make(map[string]int)
	for sourceRows.Next() {
		var source string
		var count int
		if err := sourceRows.Scan(&source, &count); err != nil {
			log.Printf("error scanning source row: %v", err)
			continue
		}
		sourceCounts[source] = count
	}
	stats["by_source"] = sourceCounts

	// Get count by level
	levelQuery := `
		SELECT level, COUNT(*) as count
		FROM console_logs
		GROUP BY level
	`
	levelRows, err := s.db.Query(levelQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get level counts: %w", err)
	}
	defer levelRows.Close()

	levelCounts := make(map[string]int)
	for levelRows.Next() {
		var level string
		var count int
		if err := levelRows.Scan(&level, &count); err != nil {
			log.Printf("error scanning level row: %v", err)
			continue
		}
		levelCounts[level] = count
	}
	stats["by_level"] = levelCounts

	return stats, nil
}
