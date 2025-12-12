// Package concurrency provides concurrent request limiting middleware for Mizu.
package concurrency

import (
	"net/http"

	"github.com/go-mizu/mizu"
)

// Options configures the concurrency middleware.
type Options struct {
	// Max is the maximum concurrent requests.
	// Default: 100.
	Max int

	// ErrorHandler handles when limit is reached.
	ErrorHandler func(c *mizu.Ctx) error
}

// New creates concurrency middleware with a semaphore limit.
func New(max int) mizu.Middleware {
	return WithOptions(Options{Max: max})
}

// WithOptions creates concurrency middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	if opts.Max <= 0 {
		// Max of 0 or negative means immediate rejection
		return func(next mizu.Handler) mizu.Handler {
			return func(c *mizu.Ctx) error {
				if opts.ErrorHandler != nil {
					return opts.ErrorHandler(c)
				}
				c.Header().Set("Retry-After", "1")
				return c.Text(http.StatusServiceUnavailable, "Server at capacity")
			}
		}
	}

	sem := make(chan struct{}, opts.Max)

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			// Try to acquire semaphore
			select {
			case sem <- struct{}{}:
				// Acquired
				defer func() { <-sem }()
				return next(c)
			default:
				// At capacity
				if opts.ErrorHandler != nil {
					return opts.ErrorHandler(c)
				}
				c.Header().Set("Retry-After", "1")
				return c.Text(http.StatusServiceUnavailable, "Server at capacity")
			}
		}
	}
}

// Blocking creates middleware that blocks until a slot is available.
func Blocking(max int) mizu.Middleware {
	sem := make(chan struct{}, max)

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			// Block until slot available
			sem <- struct{}{}
			defer func() { <-sem }()
			return next(c)
		}
	}
}

// WithContext creates middleware that respects context cancellation.
func WithContext(max int) mizu.Middleware {
	sem := make(chan struct{}, max)

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
				return next(c)
			case <-c.Context().Done():
				return c.Context().Err()
			}
		}
	}
}
