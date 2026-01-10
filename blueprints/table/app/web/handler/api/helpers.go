package api

import (
	"net/http"

	"github.com/go-mizu/mizu"
)

// Response is the standard API response format.
type Response struct {
	Success bool   `json:"success,omitempty"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

// OK returns a successful response with data.
func OK(c *mizu.Ctx, data any) error {
	return c.JSON(http.StatusOK, data)
}

// Created returns a 201 response with data.
func Created(c *mizu.Ctx, data any) error {
	return c.JSON(http.StatusCreated, data)
}

// NoContent returns a 204 response.
func NoContent(c *mizu.Ctx) error {
	c.Writer().WriteHeader(http.StatusNoContent)
	return nil
}

// BadRequest returns a 400 response with error message.
func BadRequest(c *mizu.Ctx, msg string) error {
	return c.JSON(http.StatusBadRequest, map[string]string{"message": msg})
}

// Unauthorized returns a 401 response.
func Unauthorized(c *mizu.Ctx, msg string) error {
	return c.JSON(http.StatusUnauthorized, map[string]string{"message": msg})
}

// Forbidden returns a 403 response.
func Forbidden(c *mizu.Ctx, msg string) error {
	return c.JSON(http.StatusForbidden, map[string]string{"message": msg})
}

// NotFound returns a 404 response.
func NotFound(c *mizu.Ctx, msg string) error {
	return c.JSON(http.StatusNotFound, map[string]string{"message": msg})
}

// InternalError returns a 500 response.
func InternalError(c *mizu.Ctx, msg string) error {
	return c.JSON(http.StatusInternalServerError, map[string]string{"message": msg})
}
