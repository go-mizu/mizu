// Package timeout provides request timeout middleware for Mizu.
package timeout

import (
	"context"
	"net/http"
	"time"

	"github.com/go-mizu/mizu"
)

// Options configures the timeout middleware.
type Options struct {
	// Timeout is the maximum duration for request processing.
	Timeout time.Duration

	// ErrorHandler is called when a timeout occurs.
	// If nil, returns 503 Service Unavailable.
	ErrorHandler func(c *mizu.Ctx) error

	// ErrorMessage is the message returned on timeout.
	// Default: "Service Unavailable".
	ErrorMessage string
}

// New creates a timeout middleware with the specified duration.
func New(timeout time.Duration) mizu.Middleware {
	return WithOptions(Options{Timeout: timeout})
}

// WithOptions creates a timeout middleware with the specified options.
func WithOptions(opts Options) mizu.Middleware {
	if opts.Timeout <= 0 {
		opts.Timeout = 30 * time.Second
	}
	if opts.ErrorMessage == "" {
		opts.ErrorMessage = "Service Unavailable"
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			ctx, cancel := context.WithTimeout(c.Context(), opts.Timeout)
			defer cancel()

			// Create new request with timeout context
			req := c.Request().WithContext(ctx)
			*c.Request() = *req

			// Channel to receive handler result
			done := make(chan error, 1)

			go func() {
				done <- next(c)
			}()

			select {
			case err := <-done:
				return err
			case <-ctx.Done():
				if opts.ErrorHandler != nil {
					return opts.ErrorHandler(c)
				}
				return c.Text(http.StatusServiceUnavailable, opts.ErrorMessage)
			}
		}
	}
}
