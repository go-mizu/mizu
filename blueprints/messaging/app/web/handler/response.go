// Package handler provides HTTP handlers.
package handler

import (
	"net/http"

	"github.com/go-mizu/mizu"
)

// Response represents an API response.
type Response struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

// Success sends a success response.
func Success(c *mizu.Ctx, data any) error {
	return c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
	})
}

// Created sends a 201 response.
func Created(c *mizu.Ctx, data any) error {
	return c.JSON(http.StatusCreated, Response{
		Success: true,
		Data:    data,
	})
}

// BadRequest sends a 400 response.
func BadRequest(c *mizu.Ctx, message string) error {
	return c.JSON(http.StatusBadRequest, Response{
		Success: false,
		Error:   message,
	})
}

// Unauthorized sends a 401 response.
func Unauthorized(c *mizu.Ctx, message string) error {
	return c.JSON(http.StatusUnauthorized, Response{
		Success: false,
		Error:   message,
	})
}

// Forbidden sends a 403 response.
func Forbidden(c *mizu.Ctx, message string) error {
	return c.JSON(http.StatusForbidden, Response{
		Success: false,
		Error:   message,
	})
}

// NotFound sends a 404 response.
func NotFound(c *mizu.Ctx, message string) error {
	return c.JSON(http.StatusNotFound, Response{
		Success: false,
		Error:   message,
	})
}

// InternalError sends a 500 response.
func InternalError(c *mizu.Ctx, message string) error {
	return c.JSON(http.StatusInternalServerError, Response{
		Success: false,
		Error:   message,
	})
}
