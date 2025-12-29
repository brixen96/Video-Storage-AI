package models

import (
	"encoding/json"
	"time"
)

// ConsoleLog represents a system-wide log entry
type ConsoleLog struct {
	ID        int64                  `json:"id" db:"id"`
	Source    string                 `json:"source" db:"source"`       // api, ai_companion, frontend
	Level     string                 `json:"level" db:"level"`         // debug, info, warning, error
	Message   string                 `json:"message" db:"message"`
	Details   string                 `json:"-" db:"details"`           // JSON string
	DetailsObj map[string]interface{} `json:"details,omitempty" db:"-"`
	CreatedAt time.Time              `json:"created_at" db:"created_at"`
}

// ParseDetails parses the details JSON string into DetailsObj
func (cl *ConsoleLog) ParseDetails() error {
	if cl.Details == "" || cl.Details == "{}" {
		cl.DetailsObj = make(map[string]interface{})
		return nil
	}

	if err := json.Unmarshal([]byte(cl.Details), &cl.DetailsObj); err != nil {
		return err
	}

	return nil
}

// MarshalDetails serializes DetailsObj into the Details JSON string
func (cl *ConsoleLog) MarshalDetails() error {
	if cl.DetailsObj == nil {
		cl.Details = "{}"
		return nil
	}

	detailsJSON, err := json.Marshal(cl.DetailsObj)
	if err != nil {
		return err
	}

	cl.Details = string(detailsJSON)
	return nil
}
