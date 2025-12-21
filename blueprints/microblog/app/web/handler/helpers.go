package handler

import (
	"strconv"

	"github.com/go-mizu/mizu"
)

// ErrorResponse creates an error response.
func ErrorResponse(code, message string) map[string]any {
	return map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
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
