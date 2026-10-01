package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/brixen96/video-storage-ai/internal/database"
)

// AIMonitorService tracks AI Companion API calls, tool executions, and costs
type AIMonitorService struct {
	db *sql.DB
}

// NewAIMonitorService creates a new AI monitor service
func NewAIMonitorService() *AIMonitorService {
	return &AIMonitorService{
		db: database.GetDB(),
	}
}

// AICallLog represents a single API call to OpenRouter
type AICallLog struct {
	ID               int64     `json:"id"`
	InteractionID    string    `json:"interaction_id"`
	Model            string    `json:"model"`
	Endpoint         string    `json:"endpoint"`
	StatusCode       int       `json:"status_code"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	ReasoningTokens  int       `json:"reasoning_tokens"`
	TotalTokens      int       `json:"total_tokens"`
	CostUSD          float64   `json:"cost_usd"`
	LatencyMs        int       `json:"latency_ms"`
	RetryCount       int       `json:"retry_count"`
	Success          bool      `json:"success"`
	ErrorMessage     string    `json:"error_message,omitempty"`
	RequestSizeBytes int       `json:"request_size_bytes"`
	ResponseSizeBytes int      `json:"response_size_bytes"`
	CreatedAt        time.Time `json:"created_at"`
}

// AIToolLog represents a tool execution
type AIToolLog struct {
	ID            int64     `json:"id"`
	InteractionID string    `json:"interaction_id"`
	ToolName      string    `json:"tool_name"`
	Arguments     string    `json:"arguments,omitempty"`
	ResultSummary string    `json:"result_summary,omitempty"`
	Success       bool      `json:"success"`
	ErrorMessage  string    `json:"error_message,omitempty"`
	LatencyMs     int       `json:"latency_ms"`
	CreatedAt     time.Time `json:"created_at"`
}

// DashboardMetrics contains aggregated metrics for the monitoring dashboard
type DashboardMetrics struct {
	// Call statistics
	TotalCalls       int     `json:"total_calls"`
	SuccessfulCalls  int     `json:"successful_calls"`
	FailedCalls      int     `json:"failed_calls"`
	AvgLatencyMs     float64 `json:"avg_latency_ms"`
	TotalTokensUsed int     `json:"total_tokens_used"`
	TotalCostUSD     float64 `json:"total_cost_usd"`

	// Token breakdown
	TotalPromptTokens     int `json:"total_prompt_tokens"`
	TotalCompletionTokens int `json:"total_completion_tokens"`
	TotalReasoningTokens  int `json:"total_reasoning_tokens"`

	// Tool statistics
	TotalToolCalls    int     `json:"total_tool_calls"`
	SuccessfulTools   int     `json:"successful_tools"`
	FailedTools       int     `json:"failed_tools"`
	AvgToolLatencyMs  float64 `json:"avg_tool_latency_ms"`

	// Model breakdown
	ModelBreakdown []ModelStats `json:"model_breakdown"`

	// Recent activity
	RecentCalls []AICallLog `json:"recent_calls"`

	// Cost by day (last 7 days)
	CostByDay []DailyCost `json:"cost_by_day"`
}

// ModelStats contains statistics for a specific model
type ModelStats struct {
	Model        string  `json:"model"`
	CallCount    int     `json:"call_count"`
	TotalTokens  int     `json:"total_tokens"`
	TotalCostUSD float64 `json:"total_cost_usd"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
}

// DailyCost represents cost for a specific day
type DailyCost struct {
	Date      string  `json:"date"`
	CostUSD   float64 `json:"cost_usd"`
	CallCount int     `json:"call_count"`
}

// LogCall logs an API call to OpenRouter
func (s *AIMonitorService) LogCall(callLog *AICallLog) error {
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}

	_, err := s.db.Exec(`
		INSERT INTO ai_call_logs (interaction_id, model, endpoint, status_code,
			prompt_tokens, completion_tokens, reasoning_tokens, total_tokens,
			cost_usd, latency_ms, retry_count, success, error_message,
			request_size_bytes, response_size_bytes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, callLog.InteractionID, callLog.Model, callLog.Endpoint, callLog.StatusCode,
		callLog.PromptTokens, callLog.CompletionTokens, callLog.ReasoningTokens, callLog.TotalTokens,
		callLog.CostUSD, callLog.LatencyMs, callLog.RetryCount, callLog.Success, callLog.ErrorMessage,
		callLog.RequestSizeBytes, callLog.ResponseSizeBytes)

	if err != nil {
		log.Printf("Failed to log AI call: %v", err)
	}

	return err
}

// LogToolExecution logs a tool execution
func (s *AIMonitorService) LogToolExecution(toolLog *AIToolLog) error {
	if s.db == nil {
		return fmt.Errorf("database not initialized")
	}

	_, err := s.db.Exec(`
		INSERT INTO ai_tool_logs (interaction_id, tool_name, arguments,
			result_summary, success, error_message, latency_ms)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, toolLog.InteractionID, toolLog.ToolName, toolLog.Arguments,
		toolLog.ResultSummary, toolLog.Success, toolLog.ErrorMessage, toolLog.LatencyMs)

	if err != nil {
		log.Printf("Failed to log AI tool execution: %v", err)
	}

	return err
}

// GetDashboardMetrics returns aggregated metrics for the monitoring dashboard
func (s *AIMonitorService) GetDashboardMetrics() (*DashboardMetrics, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	metrics := &DashboardMetrics{}

	// Call statistics
	s.db.QueryRow(`SELECT COUNT(*) FROM ai_call_logs`).Scan(&metrics.TotalCalls)
	s.db.QueryRow(`SELECT COUNT(*) FROM ai_call_logs WHERE success = 1`).Scan(&metrics.SuccessfulCalls)
	s.db.QueryRow(`SELECT COUNT(*) FROM ai_call_logs WHERE success = 0`).Scan(&metrics.FailedCalls)
	s.db.QueryRow(`SELECT COALESCE(AVG(latency_ms), 0) FROM ai_call_logs`).Scan(&metrics.AvgLatencyMs)
	s.db.QueryRow(`SELECT COALESCE(SUM(total_tokens), 0) FROM ai_call_logs`).Scan(&metrics.TotalTokensUsed)
	s.db.QueryRow(`SELECT COALESCE(SUM(cost_usd), 0) FROM ai_call_logs`).Scan(&metrics.TotalCostUSD)

	// Token breakdown
	s.db.QueryRow(`SELECT COALESCE(SUM(prompt_tokens), 0) FROM ai_call_logs`).Scan(&metrics.TotalPromptTokens)
	s.db.QueryRow(`SELECT COALESCE(SUM(completion_tokens), 0) FROM ai_call_logs`).Scan(&metrics.TotalCompletionTokens)
	s.db.QueryRow(`SELECT COALESCE(SUM(reasoning_tokens), 0) FROM ai_call_logs`).Scan(&metrics.TotalReasoningTokens)

	// Tool statistics
	s.db.QueryRow(`SELECT COUNT(*) FROM ai_tool_logs`).Scan(&metrics.TotalToolCalls)
	s.db.QueryRow(`SELECT COUNT(*) FROM ai_tool_logs WHERE success = 1`).Scan(&metrics.SuccessfulTools)
	s.db.QueryRow(`SELECT COUNT(*) FROM ai_tool_logs WHERE success = 0`).Scan(&metrics.FailedTools)
	s.db.QueryRow(`SELECT COALESCE(AVG(latency_ms), 0) FROM ai_tool_logs`).Scan(&metrics.AvgToolLatencyMs)

	// Model breakdown
	rows, err := s.db.Query(`
		SELECT model, COUNT(*) as call_count, SUM(total_tokens) as total_tokens,
			SUM(cost_usd) as total_cost, COALESCE(AVG(latency_ms), 0) as avg_latency
		FROM ai_call_logs
		GROUP BY model
		ORDER BY call_count DESC
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var stats ModelStats
			rows.Scan(&stats.Model, &stats.CallCount, &stats.TotalTokens, &stats.TotalCostUSD, &stats.AvgLatencyMs)
			metrics.ModelBreakdown = append(metrics.ModelBreakdown, stats)
		}
	}

	// Recent calls (last 20)
	rows, err = s.db.Query(`
		SELECT id, interaction_id, model, endpoint, status_code,
			prompt_tokens, completion_tokens, reasoning_tokens, total_tokens,
			cost_usd, latency_ms, retry_count, success, error_message,
			request_size_bytes, response_size_bytes, created_at
		FROM ai_call_logs
		ORDER BY created_at DESC
		LIMIT 20
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var call AICallLog
			rows.Scan(&call.ID, &call.InteractionID, &call.Model, &call.Endpoint,
				&call.StatusCode, &call.PromptTokens, &call.CompletionTokens,
				&call.ReasoningTokens, &call.TotalTokens, &call.CostUSD,
				&call.LatencyMs, &call.RetryCount, &call.Success,
				&call.ErrorMessage, &call.RequestSizeBytes, &call.ResponseSizeBytes,
				&call.CreatedAt)
			metrics.RecentCalls = append(metrics.RecentCalls, call)
		}
	}

	// Cost by day (last 7 days)
	rows, err = s.db.Query(`
		SELECT DATE(created_at) as date, SUM(cost_usd) as cost, COUNT(*) as count
		FROM ai_call_logs
		WHERE created_at >= datetime('now', '-7 days')
		GROUP BY DATE(created_at)
		ORDER BY date DESC
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var daily DailyCost
			rows.Scan(&daily.Date, &daily.CostUSD, &daily.CallCount)
			metrics.CostByDay = append(metrics.CostByDay, daily)
		}
	}

	return metrics, nil
}

// GetCallHistory returns paginated call history with filters
func (s *AIMonitorService) GetCallHistory(limit, offset int, model string, successOnly bool) ([]AICallLog, int, error) {
	if s.db == nil {
		return nil, 0, fmt.Errorf("database not initialized")
	}

	// Build query with filters
	whereClause := "WHERE 1=1"
	var args []interface{}

	if model != "" {
		whereClause += " AND model = ?"
		args = append(args, model)
	}
	if successOnly {
		whereClause += " AND success = 1"
	}

	// Get total count
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM ai_call_logs %s", whereClause)
	s.db.QueryRow(countQuery, args...).Scan(&total)

	// Get paginated results
	query := fmt.Sprintf(`
		SELECT id, interaction_id, model, endpoint, status_code,
			prompt_tokens, completion_tokens, reasoning_tokens, total_tokens,
			cost_usd, latency_ms, retry_count, success, error_message,
			request_size_bytes, response_size_bytes, created_at
		FROM ai_call_logs %s
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, whereClause)
	args = append(args, limit, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var calls []AICallLog
	for rows.Next() {
		var call AICallLog
		rows.Scan(&call.ID, &call.InteractionID, &call.Model, &call.Endpoint,
			&call.StatusCode, &call.PromptTokens, &call.CompletionTokens,
			&call.ReasoningTokens, &call.TotalTokens, &call.CostUSD,
			&call.LatencyMs, &call.RetryCount, &call.Success,
			&call.ErrorMessage, &call.RequestSizeBytes, &call.ResponseSizeBytes,
			&call.CreatedAt)
		calls = append(calls, call)
	}

	return calls, total, nil
}

// GetToolStats returns tool usage statistics
func (s *AIMonitorService) GetToolStats() ([]map[string]interface{}, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	rows, err := s.db.Query(`
		SELECT tool_name, COUNT(*) as count,
			SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END) as success_count,
			SUM(CASE WHEN success = 0 THEN 1 ELSE 0 END) as failure_count,
			COALESCE(AVG(latency_ms), 0) as avg_latency
		FROM ai_tool_logs
		GROUP BY tool_name
		ORDER BY count DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []map[string]interface{}
	for rows.Next() {
		var toolName string
		var count, successCount, failureCount int
		var avgLatency float64
		rows.Scan(&toolName, &count, &successCount, &failureCount, &avgLatency)

		stats = append(stats, map[string]interface{}{
			"tool_name":     toolName,
			"count":         count,
			"success_count": successCount,
			"failure_count": failureCount,
			"avg_latency":   avgLatency,
		})
	}

	return stats, nil
}

// CalculateCost calculates the cost for a given model and token counts
// Pricing per 1M tokens (as of late 2024)
func CalculateCost(model string, promptTokens, completionTokens int) float64 {
	// Define pricing for known models (per 1M tokens)
	pricing := map[string]struct {
		Prompt     float64
		Completion float64
	}{
		"deepseek/deepseek-chat":          {0.14, 0.28},
		"deepseek/deepseek-v4-flash":      {0.14, 0.28},
		"deepseek/deepseek-coder":         {0.14, 0.28},
		"openai/gpt-4o":                   {2.50, 10.00},
		"openai/gpt-4o-mini":              {0.15, 0.60},
		"anthropic/claude-3-5-sonnet":     {3.00, 15.00},
		"anthropic/claude-3-5-haiku":      {0.80, 4.00},
		"google/gemini-pro-1.5":           {1.25, 5.00},
		"qwen/qwen-3.7-flash":            {0.03, 0.13},
		"qwen/qwen3.8-omni-flash":        {0.03, 0.13},
	}

	// Find matching pricing (try exact match first, then prefix match)
	var promptPrice, completionPrice float64
	if p, ok := pricing[model]; ok {
		promptPrice = p.Prompt
		completionPrice = p.Completion
	} else {
		// Try to find a partial match
		for modelKey, p := range pricing {
			if strings.Contains(model, modelKey) || strings.Contains(modelKey, model) {
				promptPrice = p.Prompt
				completionPrice = p.Completion
				break
			}
		}
	}

	// Default pricing if model not found
	if promptPrice == 0 && completionPrice == 0 {
		promptPrice = 0.50
		completionPrice = 2.00
	}

	// Calculate cost (pricing is per 1M tokens)
	cost := (float64(promptTokens) * promptPrice / 1000000) +
		(float64(completionTokens) * completionPrice / 1000000)

	return cost
}

// LLMCallResult contains the result of an LLM call with metadata for monitoring
type LLMCallResult struct {
	InteractionID string
	Model         string
	StatusCode    int
	PromptTokens  int
	CompletionTokens int
	ReasoningTokens  int
	RequestBytes  int
	ResponseBytes int
	LatencyMs     int
	Success       bool
	Error         string
}

// MonitorLLMCall logs an LLM call to the monitoring system
func (s *AIMonitorService) MonitorCall(result *LLMCallResult) {
	cost := CalculateCost(result.Model, result.PromptTokens, result.CompletionTokens)

	logEntry := &AICallLog{
		InteractionID:    result.InteractionID,
		Model:            result.Model,
		Endpoint:         "/v1/chat/completions",
		StatusCode:       result.StatusCode,
		PromptTokens:     result.PromptTokens,
		CompletionTokens: result.CompletionTokens,
		ReasoningTokens:  result.ReasoningTokens,
		TotalTokens:      result.PromptTokens + result.CompletionTokens,
		CostUSD:          cost,
		LatencyMs:        result.LatencyMs,
		Success:          result.Success,
		ErrorMessage:     result.Error,
		RequestSizeBytes: result.RequestBytes,
		ResponseSizeBytes: result.ResponseBytes,
	}

	if err := s.LogCall(logEntry); err != nil {
		log.Printf("[AIMonitor] Failed to log call: %v", err)
	}

	// Log summary
	if result.Success {
		log.Printf("[AIMonitor] LLM call: model=%s tokens=%d/%d cost=$%.6f latency=%dms",
			result.Model, result.PromptTokens, result.CompletionTokens, cost, result.LatencyMs)
	} else {
		log.Printf("[AIMonitor] LLM call FAILED: model=%s error=%s latency=%dms",
			result.Model, result.Error, result.LatencyMs)
	}
}

// MonitorTool logs a tool execution to the monitoring system
func (s *AIMonitorService) MonitorTool(interactionID, toolName string, args map[string]interface{}, result interface{}, err error, latencyMs int) {
	argsJSON, _ := json.Marshal(args)
	resultSummary := fmt.Sprintf("%v", result)
	if len(resultSummary) > 500 {
		resultSummary = resultSummary[:500] + "..."
	}

	toolLog := &AIToolLog{
		InteractionID: interactionID,
		ToolName:      toolName,
		Arguments:     string(argsJSON),
		ResultSummary: resultSummary,
		Success:       err == nil,
		LatencyMs:     latencyMs,
	}

	if err != nil {
		toolLog.ErrorMessage = err.Error()
	}

	if logErr := s.LogToolExecution(toolLog); logErr != nil {
		log.Printf("[AIMonitor] Failed to log tool execution: %v", logErr)
	}
}
