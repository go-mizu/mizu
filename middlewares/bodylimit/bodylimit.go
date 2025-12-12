// Package bodylimit provides request body size limiting middleware for Mizu.
package bodylimit

import (
	"net/http"

	"github.com/go-mizu/mizu"
)

// Options configures the body limit middleware.
type Options struct {
	// Limit is the maximum body size in bytes.
	// Default: 1MB (1 << 20).
	Limit int64

	// ErrorHandler is called when the body exceeds the limit.
	// If nil, returns 413 Request Entity Too Large.
	ErrorHandler func(c *mizu.Ctx) error
}

// New creates a body limit middleware with the specified limit in bytes.
func New(limit int64) mizu.Middleware {
	return WithOptions(Options{Limit: limit})
}

// WithHandler creates a body limit middleware with a custom error handler.
func WithHandler(limit int64, handler func(*mizu.Ctx) error) mizu.Middleware {
	return WithOptions(Options{Limit: limit, ErrorHandler: handler})
}

// WithOptions creates a body limit middleware with the specified options.
func WithOptions(opts Options) mizu.Middleware {
	if opts.Limit <= 0 {
		opts.Limit = 1 << 20 // 1MB default
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			if c.Request().ContentLength > opts.Limit {
				if opts.ErrorHandler != nil {
					return opts.ErrorHandler(c)
				}
				return c.Text(http.StatusRequestEntityTooLarge, http.StatusText(http.StatusRequestEntityTooLarge))
			}

			// Wrap the body with MaxBytesReader
			c.Request().Body = http.MaxBytesReader(c.Writer(), c.Request().Body, opts.Limit)

			return next(c)
		}
	}
}

// KB returns bytes for kilobytes.
func KB(n int64) int64 { return n * 1024 }

// MB returns bytes for megabytes.
func MB(n int64) int64 { return n * 1024 * 1024 }

// GB returns bytes for gigabytes.
func GB(n int64) int64 { return n * 1024 * 1024 * 1024 }
