package feature

import (
	"github.com/go-mizu/mizu"
	"github.com/google/uuid"
)

// GetUserID extracts the user ID from the request context
// In production, this would come from JWT middleware
func GetUserID(c *mizu.Ctx) uuid.UUID {
	// Try to get from header (simplified auth for development)
	if userIDStr := c.Header().Get("X-User-ID"); userIDStr != "" {
		if id, err := uuid.Parse(userIDStr); err == nil {
			return id
		}
	}
	return uuid.Nil
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// SuccessResponse represents a generic success response
type SuccessResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// PaginatedResponse represents a paginated response
type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Total      int         `json:"total"`
	Page       int         `json:"page"`
	PerPage    int         `json:"per_page"`
	TotalPages int         `json:"total_pages"`
}
