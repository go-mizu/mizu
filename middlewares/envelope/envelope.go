// Package envelope provides response envelope middleware for Mizu.
package envelope

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/go-mizu/mizu"
)

// Response is the envelope response structure.
type Response struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
	Meta    *Meta  `json:"meta,omitempty"`
}

// Meta contains response metadata.
type Meta struct {
	StatusCode int    `json:"status_code"`
	RequestID  string `json:"request_id,omitempty"`
	Timestamp  int64  `json:"timestamp,omitempty"`
}

// Options configures the envelope middleware.
type Options struct {
	// IncludeMeta includes metadata in response.
	// Default: true.
	IncludeMeta bool

	// RequestIDHeader is the request ID header name.
	// Default: "X-Request-ID".
	RequestIDHeader string

	// ErrorField is the error field name.
	// Default: "error".
	ErrorField string

	// SuccessField is the success field name.
	// Default: "success".
	SuccessField string

	// DataField is the data field name.
	// Default: "data".
	DataField string

	// ContentTypes is the list of content types to wrap.
	// Default: ["application/json"].
	ContentTypes []string
}

// New creates envelope middleware with default options.
func New() mizu.Middleware {
	return WithOptions(Options{})
}

// WithOptions creates envelope middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	if opts.RequestIDHeader == "" {
		opts.RequestIDHeader = "X-Request-ID"
	}
	if opts.ErrorField == "" {
		opts.ErrorField = "error"
	}
	if opts.SuccessField == "" {
		opts.SuccessField = "success"
	}
	if opts.DataField == "" {
		opts.DataField = "data"
	}
	if len(opts.ContentTypes) == 0 {
		opts.ContentTypes = []string{"application/json"}
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			// Capture response
			rw := &envelopeWriter{
				ResponseWriter: c.Writer(),
				body:           &bytes.Buffer{},
				status:         http.StatusOK,
			}
			c.SetWriter(rw)

			err := next(c)

			// Check if we should wrap the response
			contentType := c.Writer().Header().Get("Content-Type")
			shouldWrap := false
			for _, ct := range opts.ContentTypes {
				if contentType == ct || contentType == ct+"; charset=utf-8" {
					shouldWrap = true
					break
				}
			}

			if !shouldWrap {
				return err
			}

			// Build envelope response
			success := rw.status >= 200 && rw.status < 400

			envelope := map[string]any{
				opts.SuccessField: success,
			}

			// Parse original body as JSON
			var data any
			if rw.body.Len() > 0 {
				json.Unmarshal(rw.body.Bytes(), &data)
			}

			if success {
				envelope[opts.DataField] = data
			} else {
				// Try to extract error message
				if errMap, ok := data.(map[string]any); ok {
					if errMsg, ok := errMap["error"].(string); ok {
						envelope[opts.ErrorField] = errMsg
					} else if errMsg, ok := errMap["message"].(string); ok {
						envelope[opts.ErrorField] = errMsg
					} else {
						envelope[opts.DataField] = data
					}
				} else if err != nil {
					envelope[opts.ErrorField] = err.Error()
				}
			}

			// Add metadata
			if opts.IncludeMeta {
				envelope["meta"] = map[string]any{
					"status_code": rw.status,
					"request_id":  c.Request().Header.Get(opts.RequestIDHeader),
				}
			}

			// Re-encode response
			respBody, _ := json.Marshal(envelope)

			// Write to original writer
			c.Writer().Header().Set("Content-Type", "application/json")
			rw.ResponseWriter.WriteHeader(rw.status)
			rw.ResponseWriter.Write(respBody)

			return err
		}
	}
}

type envelopeWriter struct {
	http.ResponseWriter
	body        *bytes.Buffer
	status      int
	wroteHeader bool
}

func (w *envelopeWriter) WriteHeader(code int) {
	w.status = code
	w.wroteHeader = true
}

func (w *envelopeWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.status = http.StatusOK
	}
	return w.body.Write(b)
}

// Success creates a success response.
func Success(c *mizu.Ctx, data any) error {
	return c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
	})
}

// Error creates an error response.
func Error(c *mizu.Ctx, status int, message string) error {
	return c.JSON(status, Response{
		Success: false,
		Error:   message,
	})
}

// Created creates a 201 Created response.
func Created(c *mizu.Ctx, data any) error {
	return c.JSON(http.StatusCreated, Response{
		Success: true,
		Data:    data,
	})
}

// NoContent creates a 204 No Content response.
func NoContent(c *mizu.Ctx) error {
	c.Writer().WriteHeader(http.StatusNoContent)
	return nil
}

// BadRequest creates a 400 Bad Request response.
func BadRequest(c *mizu.Ctx, message string) error {
	return Error(c, http.StatusBadRequest, message)
}

// Unauthorized creates a 401 Unauthorized response.
func Unauthorized(c *mizu.Ctx, message string) error {
	return Error(c, http.StatusUnauthorized, message)
}

// Forbidden creates a 403 Forbidden response.
func Forbidden(c *mizu.Ctx, message string) error {
	return Error(c, http.StatusForbidden, message)
}

// NotFound creates a 404 Not Found response.
func NotFound(c *mizu.Ctx, message string) error {
	return Error(c, http.StatusNotFound, message)
}

// InternalError creates a 500 Internal Server Error response.
func InternalError(c *mizu.Ctx, message string) error {
	return Error(c, http.StatusInternalServerError, message)
}
