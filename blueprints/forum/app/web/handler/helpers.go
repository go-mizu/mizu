package handler

import (
	"strconv"

	"github.com/go-mizu/mizu"
)

// ErrorResponse creates an error response.
func ErrorResponse(message string) map[string]any {
	return map[string]any{
		"error": message,
	}
}

// DataResponse creates a success response.
func DataResponse(data any) map[string]any {
	return map[string]any{
		"data": data,
	}
}

// IntQuery parses an integer query parameter with a default value.
func IntQuery(c *mizu.Ctx, key string, defaultVal int) int {
	val := c.Query(key)
	if val == "" {
		return defaultVal
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return i
}

// StringQuery gets a string query parameter with a default value.
func StringQuery(c *mizu.Ctx, key, defaultVal string) string {
	val := c.Query(key)
	if val == "" {
		return defaultVal
	}
	return val
}
