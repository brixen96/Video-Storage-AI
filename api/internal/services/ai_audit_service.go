package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/brixen96/video-storage-ai/internal/models"
)

// AIAuditService handles AI interaction logging and auditing
type AIAuditService struct {
	db *sql.DB
}

// NewAIAuditService creates a new AI audit service
func NewAIAuditService(db *sql.DB) *AIAuditService {
	return &AIAuditService{
		db: db,
	}
}

// AIAuditEntry represents data to create an audit log
type AIAuditEntry struct {
	InteractionType string
	Operation       string
	UserQuery       string
	AIPrompt        string
	AIResponse      string
	ContextData     map[string]interface{}
	PerformerID     *int64
	ThreadID        *int64
	VideoID         *int64
	TokensUsed      int
	CostUSD         float64
	ResponseTimeMs  int
	Success         bool
	ErrorMessage    string
}

// LogInteraction logs an AI interaction
func (s *AIAuditService) LogInteraction(entry *AIAuditEntry) (*models.AIAuditLog, error) {
	// Serialize context data
	contextJSON := "{}"
	if entry.ContextData != nil {
		bytes, err := json.Marshal(entry.ContextData)
		if err == nil {
			contextJSON = string(bytes)
		}
	}

	// Insert log
	result, err := s.db.Exec(`
		INSERT INTO ai_audit_logs (
			interaction_type, operation, user_query, ai_prompt, ai_response,
			context_data, performer_id, thread_id, video_id,
			tokens_used, cost_usd, response_time_ms, success, error_message
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		entry.InteractionType, entry.Operation, entry.UserQuery, entry.AIPrompt, entry.AIResponse,
		contextJSON, entry.PerformerID, entry.ThreadID, entry.VideoID,
		entry.TokensUsed, entry.CostUSD, entry.ResponseTimeMs, entry.Success, entry.ErrorMessage,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to log AI interaction: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	// Fetch and return the created log
	return s.GetByID(id)
}

// GetByID retrieves an audit log by ID
func (s *AIAuditService) GetByID(id int64) (*models.AIAuditLog, error) {
	var log models.AIAuditLog
	err := s.db.QueryRow(`
		SELECT
			id, interaction_type, operation, user_query, ai_prompt, ai_response,
			context_data, performer_id, thread_id, video_id,
			tokens_used, cost_usd, response_time_ms, success, error_message, created_at
		FROM ai_audit_logs
		WHERE id = ?
	`, id).Scan(
		&log.ID, &log.InteractionType, &log.Operation, &log.UserQuery, &log.AIPrompt, &log.AIResponse,
		&log.ContextData, &log.PerformerID, &log.ThreadID, &log.VideoID,
		&log.TokensUsed, &log.CostUSD, &log.ResponseTimeMs, &log.Success, &log.ErrorMessage, &log.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &log, nil
}

// GetAll retrieves audit logs with pagination and filters
func (s *AIAuditService) GetAll(interactionType string, limit int, offset int) ([]*models.AIAuditLog, error) {
	query := `
		SELECT
			id, interaction_type, operation, user_query, ai_prompt, ai_response,
			context_data, performer_id, thread_id, video_id,
			tokens_used, cost_usd, response_time_ms, success, error_message, created_at
		FROM ai_audit_logs
		WHERE 1=1
	`
	args := []interface{}{}

	if interactionType != "" {
		query += " AND interaction_type = ?"
		args = append(args, interactionType)
	}

	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*models.AIAuditLog
	for rows.Next() {
		var log models.AIAuditLog
		if err := rows.Scan(
			&log.ID, &log.InteractionType, &log.Operation, &log.UserQuery, &log.AIPrompt, &log.AIResponse,
			&log.ContextData, &log.PerformerID, &log.ThreadID, &log.VideoID,
			&log.TokensUsed, &log.CostUSD, &log.ResponseTimeMs, &log.Success, &log.ErrorMessage, &log.CreatedAt,
		); err != nil {
			continue
		}
		logs = append(logs, &log)
	}

	return logs, nil
}

// GetStats returns aggregate statistics
func (s *AIAuditService) GetStats() (*models.AIAuditStats, error) {
	var stats models.AIAuditStats

	// Overall stats
	err := s.db.QueryRow(`
		SELECT
			COUNT(*) as total,
			COALESCE(SUM(tokens_used), 0) as total_tokens,
			COALESCE(SUM(cost_usd), 0) as total_cost,
			COALESCE(AVG(CASE WHEN success = 1 THEN 1.0 ELSE 0.0 END), 0) * 100 as success_rate,
			COALESCE(AVG(response_time_ms), 0) as avg_response_time
		FROM ai_audit_logs
	`).Scan(
		&stats.TotalInteractions,
		&stats.TotalTokens,
		&stats.TotalCostUSD,
		&stats.SuccessRate,
		&stats.AvgResponseTimeMs,
	)

	if err != nil {
		return nil, err
	}

	// Last 24 hours
	_ = s.db.QueryRow(`
		SELECT
			COUNT(*) as count,
			COALESCE(SUM(cost_usd), 0) as cost
		FROM ai_audit_logs
		WHERE created_at >= datetime('now', '-1 day')
	`).Scan(&stats.Last24HourCount, &stats.Last24HourCost)

	// By type
	stats.InteractionsByType = make(map[string]int64)
	rows, err := s.db.Query(`
		SELECT interaction_type, COUNT(*) as count
		FROM ai_audit_logs
		GROUP BY interaction_type
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var interactionType string
			var count int64
			if err := rows.Scan(&interactionType, &count); err == nil {
				stats.InteractionsByType[interactionType] = count
			}
		}
	}

	return &stats, nil
}

// GetByPerformer retrieves audit logs for a specific performer
func (s *AIAuditService) GetByPerformer(performerID int64, limit int) ([]*models.AIAuditLog, error) {
	rows, err := s.db.Query(`
		SELECT
			id, interaction_type, operation, user_query, ai_prompt, ai_response,
			context_data, performer_id, thread_id, video_id,
			tokens_used, cost_usd, response_time_ms, success, error_message, created_at
		FROM ai_audit_logs
		WHERE performer_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`, performerID, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*models.AIAuditLog
	for rows.Next() {
		var log models.AIAuditLog
		if err := rows.Scan(
			&log.ID, &log.InteractionType, &log.Operation, &log.UserQuery, &log.AIPrompt, &log.AIResponse,
			&log.ContextData, &log.PerformerID, &log.ThreadID, &log.VideoID,
			&log.TokensUsed, &log.CostUSD, &log.ResponseTimeMs, &log.Success, &log.ErrorMessage, &log.CreatedAt,
		); err != nil {
			continue
		}
		logs = append(logs, &log)
	}

	return logs, nil
}

// GetByThread retrieves audit logs for a specific thread
func (s *AIAuditService) GetByThread(threadID int64, limit int) ([]*models.AIAuditLog, error) {
	rows, err := s.db.Query(`
		SELECT
			id, interaction_type, operation, user_query, ai_prompt, ai_response,
			context_data, performer_id, thread_id, video_id,
			tokens_used, cost_usd, response_time_ms, success, error_message, created_at
		FROM ai_audit_logs
		WHERE thread_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`, threadID, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*models.AIAuditLog
	for rows.Next() {
		var log models.AIAuditLog
		if err := rows.Scan(
			&log.ID, &log.InteractionType, &log.Operation, &log.UserQuery, &log.AIPrompt, &log.AIResponse,
			&log.ContextData, &log.PerformerID, &log.ThreadID, &log.VideoID,
			&log.TokensUsed, &log.CostUSD, &log.ResponseTimeMs, &log.Success, &log.ErrorMessage, &log.CreatedAt,
		); err != nil {
			continue
		}
		logs = append(logs, &log)
	}

	return logs, nil
}

// Search searches audit logs by user query or AI response
func (s *AIAuditService) Search(query string, limit int) ([]*models.AIAuditLog, error) {
	rows, err := s.db.Query(`
		SELECT
			id, interaction_type, operation, user_query, ai_prompt, ai_response,
			context_data, performer_id, thread_id, video_id,
			tokens_used, cost_usd, response_time_ms, success, error_message, created_at
		FROM ai_audit_logs
		WHERE user_query LIKE ? OR ai_response LIKE ? OR operation LIKE ?
		ORDER BY created_at DESC
		LIMIT ?
	`, "%"+query+"%", "%"+query+"%", "%"+query+"%", limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*models.AIAuditLog
	for rows.Next() {
		var log models.AIAuditLog
		if err := rows.Scan(
			&log.ID, &log.InteractionType, &log.Operation, &log.UserQuery, &log.AIPrompt, &log.AIResponse,
			&log.ContextData, &log.PerformerID, &log.ThreadID, &log.VideoID,
			&log.TokensUsed, &log.CostUSD, &log.ResponseTimeMs, &log.Success, &log.ErrorMessage, &log.CreatedAt,
		); err != nil {
			continue
		}
		logs = append(logs, &log)
	}

	return logs, nil
}

// DeleteOldLogs deletes logs older than the specified number of days
func (s *AIAuditService) DeleteOldLogs(daysToKeep int) (int64, error) {
	result, err := s.db.Exec(`
		DELETE FROM ai_audit_logs
		WHERE created_at < datetime('now', ?)
	`, fmt.Sprintf("-%d days", daysToKeep))

	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// LogAIInteraction is a helper to time and log AI interactions automatically
func (s *AIAuditService) LogAIInteraction(
	interactionType, operation, userQuery, aiPrompt string,
	performerID, threadID, videoID *int64,
	contextData map[string]interface{},
	fn func() (string, int, float64, error),
) (string, error) {
	start := time.Now()

	// Execute the AI function
	response, tokens, cost, err := fn()

	// Calculate response time
	responseTimeMs := int(time.Since(start).Milliseconds())

	// Create audit entry
	entry := &AIAuditEntry{
		InteractionType: interactionType,
		Operation:       operation,
		UserQuery:       userQuery,
		AIPrompt:        aiPrompt,
		AIResponse:      response,
		ContextData:     contextData,
		PerformerID:     performerID,
		ThreadID:        threadID,
		VideoID:         videoID,
		TokensUsed:      tokens,
		CostUSD:         cost,
		ResponseTimeMs:  responseTimeMs,
		Success:         err == nil,
	}

	if err != nil {
		entry.ErrorMessage = err.Error()
	}

	// Log the interaction
	_, logErr := s.LogInteraction(entry)
	if logErr != nil {
		// Don't fail the operation if logging fails, just log the error
		fmt.Printf("Failed to log AI interaction: %v\n", logErr)
	}

	return response, err
}
