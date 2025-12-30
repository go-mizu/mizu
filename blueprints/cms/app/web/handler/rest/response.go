// Package rest provides WordPress REST API v2 compatible handlers.
package rest

import (
	"strconv"

	"github.com/go-mizu/mizu"
)

// WPError represents a WordPress REST API error response.
type WPError struct {
	Code    string       `json:"code"`
	Message string       `json:"message"`
	Data    *WPErrorData `json:"data,omitempty"`
}

// WPErrorData contains additional error data.
type WPErrorData struct {
	Status int                    `json:"status"`
	Params map[string]string      `json:"params,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// Success returns a success response with optional pagination headers.
func Success(c *mizu.Ctx, data interface{}) error {
	return c.JSON(200, data)
}

// SuccessWithPagination returns a success response with pagination headers.
func SuccessWithPagination(c *mizu.Ctx, data interface{}, total, totalPages int) error {
	c.Header().Set("X-WP-Total", strconv.Itoa(total))
	c.Header().Set("X-WP-TotalPages", strconv.Itoa(totalPages))
	return c.JSON(200, data)
}

// Created returns a 201 created response.
func Created(c *mizu.Ctx, data interface{}) error {
	return c.JSON(201, data)
}

// NoContent returns a 204 no content response.
func NoContent(c *mizu.Ctx) error {
	return c.Text(204, "")
}

// Deleted returns a response for a deleted resource.
func Deleted(c *mizu.Ctx, data interface{}) error {
	return c.JSON(200, map[string]interface{}{
		"deleted":  true,
		"previous": data,
	})
}

// BadRequest returns a 400 bad request response.
func BadRequest(c *mizu.Ctx, message string) error {
	return c.JSON(400, WPError{
		Code:    "rest_invalid_param",
		Message: message,
		Data:    &WPErrorData{Status: 400},
	})
}

// Unauthorized returns a 401 unauthorized response.
func Unauthorized(c *mizu.Ctx, message string) error {
	return c.JSON(401, WPError{
		Code:    "rest_not_logged_in",
		Message: message,
		Data:    &WPErrorData{Status: 401},
	})
}

// Forbidden returns a 403 forbidden response.
func Forbidden(c *mizu.Ctx, message string) error {
	return c.JSON(403, WPError{
		Code:    "rest_forbidden",
		Message: message,
		Data:    &WPErrorData{Status: 403},
	})
}

// NotFound returns a 404 not found response.
func NotFound(c *mizu.Ctx, message string) error {
	return c.JSON(404, WPError{
		Code:    "rest_post_invalid_id",
		Message: message,
		Data:    &WPErrorData{Status: 404},
	})
}

// Conflict returns a 409 conflict response.
func Conflict(c *mizu.Ctx, message string) error {
	return c.JSON(409, WPError{
		Code:    "rest_post_exists",
		Message: message,
		Data:    &WPErrorData{Status: 409},
	})
}

// InternalError returns a 500 internal server error response.
func InternalError(c *mizu.Ctx, message string) error {
	return c.JSON(500, WPError{
		Code:    "rest_internal_error",
		Message: message,
		Data:    &WPErrorData{Status: 500},
	})
}

// ParsePagination parses pagination query parameters.
func ParsePagination(c *mizu.Ctx) (page, perPage int) {
	page = 1
	perPage = 10

	if p := c.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}

	if pp := c.Query("per_page"); pp != "" {
		if v, err := strconv.Atoi(pp); err == nil && v > 0 && v <= 100 {
			perPage = v
		}
	}

	return
}

// ParseOrder parses order query parameters.
func ParseOrder(c *mizu.Ctx, defaultOrderBy, defaultOrder string) (orderBy, order string) {
	orderBy = c.Query("orderby")
	if orderBy == "" {
		orderBy = defaultOrderBy
	}

	order = c.Query("order")
	if order == "" {
		order = defaultOrder
	}

	return
}
