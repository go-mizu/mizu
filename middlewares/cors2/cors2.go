// Package cors2 provides a simplified CORS middleware for Mizu.
package cors2

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/mizu"
)

// Options configures the simple CORS middleware.
type Options struct {
	// Origin is the allowed origin.
	// Default: "*".
	Origin string

	// Methods are the allowed methods.
	// Default: "GET, POST, PUT, DELETE, OPTIONS".
	Methods string

	// Headers are the allowed headers.
	// Default: "Content-Type, Authorization".
	Headers string

	// ExposeHeaders are headers exposed to the browser.
	ExposeHeaders string

	// Credentials allows credentials.
	// Default: false.
	Credentials bool

	// MaxAge is the preflight cache duration.
	// Default: 0 (no caching).
	MaxAge time.Duration
}

// New creates simple CORS middleware allowing all origins.
func New() mizu.Middleware {
	return WithOptions(Options{})
}

// WithOptions creates CORS middleware with custom options.
//
//nolint:cyclop // CORS handling requires multiple header checks
func WithOptions(opts Options) mizu.Middleware {
	if opts.Origin == "" {
		opts.Origin = "*"
	}
	if opts.Methods == "" {
		opts.Methods = "GET, POST, PUT, DELETE, OPTIONS"
	}
	if opts.Headers == "" {
		opts.Headers = "Content-Type, Authorization"
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			origin := c.Request().Header.Get("Origin")

			// Set CORS headers
			if opts.Origin == "*" {
				c.Header().Set("Access-Control-Allow-Origin", "*")
			} else if origin != "" && matchOrigin(origin, opts.Origin) {
				c.Header().Set("Access-Control-Allow-Origin", origin)
			}

			if opts.Credentials {
				c.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			if opts.ExposeHeaders != "" {
				c.Header().Set("Access-Control-Expose-Headers", opts.ExposeHeaders)
			}

			// Handle preflight
			if c.Request().Method == http.MethodOptions {
				c.Header().Set("Access-Control-Allow-Methods", opts.Methods)
				c.Header().Set("Access-Control-Allow-Headers", opts.Headers)

				if opts.MaxAge > 0 {
					c.Header().Set("Access-Control-Max-Age", strconv.Itoa(int(opts.MaxAge.Seconds())))
				}

				c.Writer().WriteHeader(http.StatusNoContent)
				return nil
			}

			return next(c)
		}
	}
}

func matchOrigin(origin, pattern string) bool {
	// Simple matching - exact match or wildcard
	if pattern == "*" {
		return true
	}
	return strings.EqualFold(origin, pattern)
}

// AllowOrigin creates middleware allowing a specific origin.
func AllowOrigin(origin string) mizu.Middleware {
	return WithOptions(Options{Origin: origin})
}

// AllowAll creates middleware allowing all origins with credentials.
func AllowAll() mizu.Middleware {
	return WithOptions(Options{
		Origin:      "*",
		Methods:     "GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS",
		Headers:     "Origin, Content-Type, Accept, Authorization, X-Requested-With",
		Credentials: false,
		MaxAge:      12 * time.Hour,
	})
}

// AllowCredentials creates middleware allowing credentials from a specific origin.
func AllowCredentials(origin string) mizu.Middleware {
	return WithOptions(Options{
		Origin:      origin,
		Credentials: true,
	})
}
