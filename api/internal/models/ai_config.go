package models

import "time"

// AIConfig represents the AI provider configuration (OpenRouter or compatible)
type AIConfig struct {
	ID              int64     `json:"id"`
	APIURL          string    `json:"apiUrl"`
	APIKey          string    `json:"apiKey"`
	ModelName       string    `json:"modelName"`
	Temperature     float64   `json:"temperature"`
	MaxTokens       int       `json:"maxTokens"`
	ReasoningEffort string    `json:"reasoningEffort"` // none, low, medium, high
	Enabled         bool      `json:"enabled"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}
