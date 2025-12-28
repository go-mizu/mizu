package handler

import (
	"net/http"

	"github.com/go-mizu/mizu"
)

// Response is a standard API response
type Response struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
}

// OK returns a 200 response with data
func OK(c *mizu.Ctx, data any) error {
	return c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
	})
}

// Created returns a 201 response with data
func Created(c *mizu.Ctx, data any) error {
	return c.JSON(http.StatusCreated, Response{
		Success: true,
		Data:    data,
	})
}

// NoContent returns a 204 response
func NoContent(c *mizu.Ctx) error {
	c.Status(http.StatusNoContent)
	return nil
}

// BadRequest returns a 400 response
func BadRequest(c *mizu.Ctx, msg string) error {
	return c.JSON(http.StatusBadRequest, Response{
		Success: false,
		Error:   msg,
	})
}

// Unauthorized returns a 401 response
func Unauthorized(c *mizu.Ctx, msg string) error {
	return c.JSON(http.StatusUnauthorized, Response{
		Success: false,
		Error:   msg,
	})
}

// Forbidden returns a 403 response
func Forbidden(c *mizu.Ctx, msg string) error {
	return c.JSON(http.StatusForbidden, Response{
		Success: false,
		Error:   msg,
	})
}

// NotFound returns a 404 response
func NotFound(c *mizu.Ctx, msg string) error {
	return c.JSON(http.StatusNotFound, Response{
		Success: false,
		Error:   msg,
	})
}

// Conflict returns a 409 response
func Conflict(c *mizu.Ctx, msg string) error {
	return c.JSON(http.StatusConflict, Response{
		Success: false,
		Error:   msg,
	})
}

// InternalError returns a 500 response
func InternalError(c *mizu.Ctx, msg string) error {
	return c.JSON(http.StatusInternalServerError, Response{
		Success: false,
		Error:   msg,
	})
}

// List is a paginated list response
type List struct {
	Items      any `json:"items"`
	Total      int `json:"total"`
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	TotalPages int `json:"total_pages"`
}

// OKList returns a 200 response with paginated data
func OKList(c *mizu.Ctx, items any, total, page, perPage int) error {
	totalPages := total / perPage
	if total%perPage > 0 {
		totalPages++
	}
	return c.JSON(http.StatusOK, Response{
		Success: true,
		Data: List{
			Items:      items,
			Total:      total,
			Page:       page,
			PerPage:    perPage,
			TotalPages: totalPages,
		},
	})
}
