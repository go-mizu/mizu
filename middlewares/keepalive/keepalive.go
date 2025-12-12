// Package keepalive provides Keep-Alive header management middleware for Mizu.
package keepalive

import (
	"fmt"
	"time"

	"github.com/go-mizu/mizu"
)

// Options configures the keepalive middleware.
type Options struct {
	// Timeout is the keep-alive timeout.
	// Default: 60s.
	Timeout time.Duration

	// MaxRequests is the maximum requests per connection.
	// Default: 100.
	MaxRequests int

	// DisableKeepAlive disables keep-alive entirely.
	// Default: false.
	DisableKeepAlive bool
}

// New creates keepalive middleware with default options.
func New() mizu.Middleware {
	return WithOptions(Options{})
}

// WithOptions creates keepalive middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	if opts.Timeout == 0 {
		opts.Timeout = 60 * time.Second
	}
	if opts.MaxRequests == 0 {
		opts.MaxRequests = 100
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			if opts.DisableKeepAlive {
				c.Header().Set("Connection", "close")
				return next(c)
			}

			// Check if client supports keep-alive
			connection := c.Request().Header.Get("Connection")
			if connection == "close" {
				c.Header().Set("Connection", "close")
				return next(c)
			}

			// Set keep-alive headers
			c.Header().Set("Connection", "keep-alive")
			c.Header().Set("Keep-Alive", fmt.Sprintf("timeout=%d, max=%d",
				int(opts.Timeout.Seconds()), opts.MaxRequests))

			return next(c)
		}
	}
}

// Disable creates middleware that disables keep-alive.
func Disable() mizu.Middleware {
	return WithOptions(Options{DisableKeepAlive: true})
}

// WithTimeout creates middleware with a specific timeout.
func WithTimeout(timeout time.Duration) mizu.Middleware {
	return WithOptions(Options{Timeout: timeout})
}

// WithMax creates middleware with a specific max requests.
func WithMax(maxRequests int) mizu.Middleware {
	return WithOptions(Options{MaxRequests: maxRequests})
}
