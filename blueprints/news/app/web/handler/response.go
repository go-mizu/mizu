package handler

import (
	"github.com/go-mizu/mizu"
)

// JSON response helpers

// Success returns a JSON success response.
func Success(c *mizu.Ctx, data any) error {
	return c.JSON(200, data)
}

// Created returns a JSON created response.
func Created(c *mizu.Ctx, data any) error {
	return c.JSON(201, data)
}

// NoContent returns a 204 response.
func NoContent(c *mizu.Ctx) error {
	return c.Text(204, "")
}

// BadRequest returns a 400 error.
func BadRequest(c *mizu.Ctx, message string) error {
	return c.JSON(400, map[string]string{"error": message})
}

// Unauthorized returns a 401 error.
func Unauthorized(c *mizu.Ctx) error {
	return c.JSON(401, map[string]string{"error": "unauthorized"})
}

// Forbidden returns a 403 error.
func Forbidden(c *mizu.Ctx) error {
	return c.JSON(403, map[string]string{"error": "forbidden"})
}

// NotFound returns a 404 error.
func NotFound(c *mizu.Ctx, resource string) error {
	return c.JSON(404, map[string]string{"error": resource + " not found"})
}

// Conflict returns a 409 error.
func Conflict(c *mizu.Ctx, message string) error {
	return c.JSON(409, map[string]string{"error": message})
}

// InternalError returns a 500 error.
func InternalError(c *mizu.Ctx, err error) error {
	return c.JSON(500, map[string]string{"error": "internal server error"})
}
