package models

import "time"

// AIConfig represents the LM Studio AI configuration
type AIConfig struct {
	ID           int64     `json:"id"`
	LMStudioURL  string    `json:"lmStudioUrl"`
	ModelName    string    `json:"modelName"`
	Temperature  float64   `json:"temperature"`
	MaxTokens    int       `json:"maxTokens"`
	Enabled      bool      `json:"enabled"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}
