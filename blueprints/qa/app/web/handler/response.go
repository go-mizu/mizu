package handler

import "github.com/go-mizu/mizu"

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

// Success returns a success response.
func Success(c *mizu.Ctx, data any) error {
	return c.JSON(200, Response{Data: data})
}

// Created returns a created response.
func Created(c *mizu.Ctx, data any) error {
	return c.JSON(201, Response{Data: data})
}

// ErrorResponse returns an error response.
func ErrorResponse(c *mizu.Ctx, status int, code, message string) error {
	return c.JSON(status, Response{Error: &Error{Code: code, Message: message}})
}

// BadRequest returns a bad request response.
func BadRequest(c *mizu.Ctx, message string) error {
	return ErrorResponse(c, 400, "BAD_REQUEST", message)
}

// Unauthorized returns an unauthorized response.
func Unauthorized(c *mizu.Ctx, messages ...string) error {
	msg := "Authentication required"
	if len(messages) > 0 {
		msg = messages[0]
	}
	return ErrorResponse(c, 401, "UNAUTHORIZED", msg)
}

// NotFound returns a not found response.
func NotFound(c *mizu.Ctx, what string) error {
	return ErrorResponse(c, 404, "NOT_FOUND", what+" not found")
}

// InternalError returns an internal error response.
func InternalError(c *mizu.Ctx) error {
	return ErrorResponse(c, 500, "INTERNAL_ERROR", "An internal error occurred")
}
