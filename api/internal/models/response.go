package models

import "time"

// APIResponse is a generic API response wrapper
type APIResponse struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// PaginatedResponse wraps paginated data
type PaginatedResponse struct {
	Success    bool        `json:"success"`
	Data       interface{} `json:"data"`
	Pagination Pagination  `json:"pagination"`
	Timestamp  time.Time   `json:"timestamp"`
}

// Pagination contains pagination metadata
type Pagination struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Success   bool      `json:"success"`
	Error     string    `json:"error"`
	Details   string    `json:"details,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// SuccessResponse creates a successful API response
func SuccessResponse(data interface{}, message string) APIResponse {
	return APIResponse{
		Success:   true,
		Message:   message,
		Data:      data,
		Timestamp: time.Now(),
	}
}

// ErrorResponseMsg creates an error response
func ErrorResponseMsg(err string, details string) ErrorResponse {
	return ErrorResponse{
		Success:   false,
		Error:     err,
		Details:   details,
		Timestamp: time.Now(),
	}
}

// NewPaginatedResponse creates a paginated response
func NewPaginatedResponse(data interface{}, page, limit int, total int64) PaginatedResponse {
	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	return PaginatedResponse{
		Success: true,
		Data:    data,
		Pagination: Pagination{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
		Timestamp: time.Now(),
	}
}
