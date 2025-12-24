// Package handler provides HTTP handlers.
package handler

import (
	"strconv"

	"github.com/go-mizu/mizu"
)

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

// BoolQuery parses a boolean query parameter with a default value.
func BoolQuery(c *mizu.Ctx, key string, defaultVal bool) bool {
	val := c.Query(key)
	if val == "" {
		return defaultVal
	}
	return val == "true" || val == "1"
}

// Success returns a 200 JSON response.
func Success(c *mizu.Ctx, data interface{}) error {
	return c.JSON(200, data)
}

// Created returns a 201 JSON response.
func Created(c *mizu.Ctx, data interface{}) error {
	return c.JSON(201, data)
}

// NoContent returns a 204 response.
func NoContent(c *mizu.Ctx) error {
	c.Writer().WriteHeader(204)
	return nil
}

// BadRequest returns a 400 JSON error.
func BadRequest(c *mizu.Ctx, msg string) error {
	return c.JSON(400, map[string]string{"error": msg})
}

// Unauthorized returns a 401 JSON error.
func Unauthorized(c *mizu.Ctx) error {
	return c.JSON(401, map[string]string{"error": "unauthorized"})
}

// Forbidden returns a 403 JSON error.
func Forbidden(c *mizu.Ctx) error {
	return c.JSON(403, map[string]string{"error": "forbidden"})
}

// NotFound returns a 404 JSON error.
func NotFound(c *mizu.Ctx, resource string) error {
	return c.JSON(404, map[string]string{"error": resource + " not found"})
}

// Conflict returns a 409 JSON error.
func Conflict(c *mizu.Ctx, msg string) error {
	return c.JSON(409, map[string]string{"error": msg})
}

// UnprocessableEntity returns a 422 JSON error.
func UnprocessableEntity(c *mizu.Ctx, msg string) error {
	return c.JSON(422, map[string]string{"error": msg})
}

// InternalError returns a 500 JSON error.
func InternalError(c *mizu.Ctx, err error) error {
	return c.JSON(500, map[string]string{"error": "internal server error"})
}
