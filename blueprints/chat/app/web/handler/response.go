// Package handler provides HTTP handlers for the chat API.
package handler

import (
	"github.com/go-mizu/mizu"
)

// Response is the standard API response.
type Response struct {
	Data  any    `json:"data,omitempty"`
	Error *Error `json:"error,omitempty"`
}

// Error represents an API error.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Success sends a successful response.
func Success(c *mizu.Ctx, data any) error {
	return c.JSON(200, Response{Data: data})
}

// Created sends a 201 response.
func Created(c *mizu.Ctx, data any) error {
	return c.JSON(201, Response{Data: data})
}

// NoContent sends a 204 response.
func NoContent(c *mizu.Ctx) error {
	c.Writer().WriteHeader(204)
	return nil
}

// ErrorResponse sends an error response.
func ErrorResponse(c *mizu.Ctx, status int, code, message string) error {
	return c.JSON(status, Response{
		Error: &Error{Code: code, Message: message},
	})
}

// BadRequest sends a 400 response.
func BadRequest(c *mizu.Ctx, message string) error {
	return ErrorResponse(c, 400, "BAD_REQUEST", message)
}

// Unauthorized sends a 401 response.
func Unauthorized(c *mizu.Ctx, message string) error {
	return ErrorResponse(c, 401, "UNAUTHORIZED", message)
}

// Forbidden sends a 403 response.
func Forbidden(c *mizu.Ctx, message string) error {
	return ErrorResponse(c, 403, "FORBIDDEN", message)
}

// NotFound sends a 404 response.
func NotFound(c *mizu.Ctx, message string) error {
	return ErrorResponse(c, 404, "NOT_FOUND", message)
}

// Conflict sends a 409 response.
func Conflict(c *mizu.Ctx, message string) error {
	return ErrorResponse(c, 409, "CONFLICT", message)
}

// InternalError sends a 500 response.
func InternalError(c *mizu.Ctx, message string) error {
	return ErrorResponse(c, 500, "INTERNAL_ERROR", message)
}
