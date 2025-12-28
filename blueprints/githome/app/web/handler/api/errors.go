package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-mizu/mizu"
)

// Error represents a GitHub API error response
type Error struct {
	Message          string        `json:"message"`
	DocumentationURL string        `json:"documentation_url,omitempty"`
	Errors           []ErrorDetail `json:"errors,omitempty"`
}

// ErrorDetail represents a detailed error
type ErrorDetail struct {
	Resource string `json:"resource,omitempty"`
	Field    string `json:"field,omitempty"`
	Code     string `json:"code,omitempty"`
	Message  string `json:"message,omitempty"`
}

// JSON sends a JSON response
func JSON(c *mizu.Ctx, code int, v any) error {
	return c.JSON(code, v)
}

// WriteError sends an error response
func WriteError(c *mizu.Ctx, status int, message string) error {
	return c.JSON(status, &Error{
		Message: message,
	})
}

// ValidationError sends a validation error response
func ValidationError(c *mizu.Ctx, message string, errors []ErrorDetail) error {
	return c.JSON(http.StatusUnprocessableEntity, &Error{
		Message: message,
		Errors:  errors,
	})
}

// NotFound sends a 404 response
func NotFound(c *mizu.Ctx, resource string) error {
	return c.JSON(http.StatusNotFound, &Error{
		Message: fmt.Sprintf("%s not found", resource),
	})
}

// Forbidden sends a 403 response
func Forbidden(c *mizu.Ctx, message string) error {
	if message == "" {
		message = "Must have admin rights to Repository"
	}
	return c.JSON(http.StatusForbidden, &Error{
		Message: message,
	})
}

// Unauthorized sends a 401 response
func Unauthorized(c *mizu.Ctx) error {
	return c.JSON(http.StatusUnauthorized, &Error{
		Message: "Requires authentication",
	})
}

// BadRequest sends a 400 response
func BadRequest(c *mizu.Ctx, message string) error {
	return c.JSON(http.StatusBadRequest, &Error{
		Message: message,
	})
}

// Conflict sends a 409 response
func Conflict(c *mizu.Ctx, message string) error {
	return c.JSON(http.StatusConflict, &Error{
		Message: message,
	})
}

// NoContent sends a 204 response
func NoContent(c *mizu.Ctx) error {
	return c.NoContent()
}

// Created sends a 201 response with JSON body
func Created(c *mizu.Ctx, v any) error {
	return c.JSON(http.StatusCreated, v)
}

// Accepted sends a 202 response with JSON body
func Accepted(c *mizu.Ctx, v any) error {
	return c.JSON(http.StatusAccepted, v)
}

// SetLinkHeader sets the Link header for pagination
func SetLinkHeader(c *mizu.Ctx, page, perPage, totalCount int) {
	req := c.Request()
	baseURL := req.URL.Path
	query := req.URL.Query()

	totalPages := (totalCount + perPage - 1) / perPage
	if totalPages == 0 {
		totalPages = 1
	}

	var links []string

	// First page
	if page > 1 {
		query.Set("page", "1")
		links = append(links, fmt.Sprintf(`<%s?%s>; rel="first"`, baseURL, query.Encode()))
	}

	// Prev page
	if page > 1 {
		query.Set("page", strconv.Itoa(page-1))
		links = append(links, fmt.Sprintf(`<%s?%s>; rel="prev"`, baseURL, query.Encode()))
	}

	// Next page
	if page < totalPages {
		query.Set("page", strconv.Itoa(page+1))
		links = append(links, fmt.Sprintf(`<%s?%s>; rel="next"`, baseURL, query.Encode()))
	}

	// Last page
	if page < totalPages {
		query.Set("page", strconv.Itoa(totalPages))
		links = append(links, fmt.Sprintf(`<%s?%s>; rel="last"`, baseURL, query.Encode()))
	}

	if len(links) > 0 {
		c.Header().Set("Link", strings.Join(links, ", "))
	}
}
