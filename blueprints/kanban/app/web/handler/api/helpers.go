package api

import (
	"net/http"

	"github.com/go-mizu/mizu"
)

// Response is the standard API response format.
type Response struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

// errResponse creates an error response map (legacy, for compatibility).
func errResponse(msg string) map[string]string {
	return map[string]string{"error": msg}
}

// OK returns a successful response with data.
func OK(c *mizu.Ctx, data any) error {
	return c.JSON(http.StatusOK, Response{Success: true, Data: data})
}

// Created returns a 201 response with data.
func Created(c *mizu.Ctx, data any) error {
	return c.JSON(http.StatusCreated, Response{Success: true, Data: data})
}

// BadRequest returns a 400 response with error message.
func BadRequest(c *mizu.Ctx, msg string) error {
	return c.JSON(http.StatusBadRequest, Response{Success: false, Error: msg})
}

// Unauthorized returns a 401 response.
func Unauthorized(c *mizu.Ctx, msg string) error {
	return c.JSON(http.StatusUnauthorized, Response{Success: false, Error: msg})
}

// Forbidden returns a 403 response.
func Forbidden(c *mizu.Ctx, msg string) error {
	return c.JSON(http.StatusForbidden, Response{Success: false, Error: msg})
}

// NotFound returns a 404 response.
func NotFound(c *mizu.Ctx, msg string) error {
	return c.JSON(http.StatusNotFound, Response{Success: false, Error: msg})
}

// InternalError returns a 500 response.
func InternalError(c *mizu.Ctx, msg string) error {
	return c.JSON(http.StatusInternalServerError, Response{Success: false, Error: msg})
}
