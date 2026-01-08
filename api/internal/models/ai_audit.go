package models

import "time"

// AIAuditLog represents an AI interaction audit log
type AIAuditLog struct {
	ID               int64     `json:"id" db:"id"`
	InteractionType  string    `json:"interaction_type" db:"interaction_type"` // chat, search, tag_suggest, analysis, etc.
	Operation        string    `json:"operation" db:"operation"`               // Specific operation performed
	UserQuery        string    `json:"user_query" db:"user_query"`             // User's input/question
	AIPrompt         string    `json:"ai_prompt" db:"ai_prompt"`               // Full prompt sent to AI
	AIResponse       string    `json:"ai_response" db:"ai_response"`           // AI's response
	ContextData      string    `json:"context_data" db:"context_data"`         // JSON with additional context
	PerformerID      *int64    `json:"performer_id" db:"performer_id"`
	ThreadID         *int64    `json:"thread_id" db:"thread_id"`
	VideoID          *int64    `json:"video_id" db:"video_id"`
	TokensUsed       int       `json:"tokens_used" db:"tokens_used"`
	CostUSD          float64   `json:"cost_usd" db:"cost_usd"`
	ResponseTimeMs   int       `json:"response_time_ms" db:"response_time_ms"`
	Success          bool      `json:"success" db:"success"`
	ErrorMessage     string    `json:"error_message" db:"error_message"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}

// AIAuditStats represents aggregate statistics for AI usage
type AIAuditStats struct {
	TotalInteractions int64   `json:"total_interactions"`
	TotalTokens       int64   `json:"total_tokens"`
	TotalCostUSD      float64 `json:"total_cost_usd"`
	SuccessRate       float64 `json:"success_rate"`
	AvgResponseTimeMs int     `json:"avg_response_time_ms"`
	InteractionsByType map[string]int64 `json:"interactions_by_type"`
	Last24HourCount   int64   `json:"last_24_hour_count"`
	Last24HourCost    float64 `json:"last_24_hour_cost"`
}
