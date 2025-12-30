package rest

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

// ListResponse is the standard paginated list response.
type ListResponse struct {
	Success bool `json:"success"`
	Data    any  `json:"data"`
	Meta    Meta `json:"meta"`
}

// Meta contains pagination metadata.
type Meta struct {
	Total      int `json:"total"`
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	TotalPages int `json:"total_pages"`
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

// List returns a paginated list response.
func List(c *mizu.Ctx, data any, total, page, perPage int) error {
	totalPages := total / perPage
	if total%perPage > 0 {
		totalPages++
	}
	return c.JSON(http.StatusOK, ListResponse{
		Success: true,
		Data:    data,
		Meta: Meta{
			Total:      total,
			Page:       page,
			PerPage:    perPage,
			TotalPages: totalPages,
		},
	})
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
