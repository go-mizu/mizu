// Package xrequestedwith provides X-Requested-With header validation middleware for Mizu.
package xrequestedwith

import (
	"net/http"
	"strings"

	"github.com/go-mizu/mizu"
)

// Options configures the X-Requested-With middleware.
type Options struct {
	// Value is the required header value.
	// Default: "XMLHttpRequest".
	Value string

	// SkipMethods are HTTP methods to skip.
	// Default: GET, HEAD, OPTIONS.
	SkipMethods []string

	// SkipPaths are paths to skip.
	SkipPaths []string

	// ErrorHandler handles missing/invalid header.
	ErrorHandler func(c *mizu.Ctx) error
}

// New creates middleware requiring XMLHttpRequest header.
func New() mizu.Middleware {
	return WithOptions(Options{})
}

// WithOptions creates middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	if opts.Value == "" {
		opts.Value = "XMLHttpRequest"
	}
	if opts.SkipMethods == nil {
		opts.SkipMethods = []string{"GET", "HEAD", "OPTIONS"}
	}

	skipMethods := make(map[string]bool)
	for _, m := range opts.SkipMethods {
		skipMethods[strings.ToUpper(m)] = true
	}

	skipPaths := make(map[string]bool)
	for _, p := range opts.SkipPaths {
		skipPaths[p] = true
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			// Skip configured methods
			if skipMethods[c.Request().Method] {
				return next(c)
			}

			// Skip configured paths
			if skipPaths[c.Request().URL.Path] {
				return next(c)
			}

			// Check header
			header := c.Request().Header.Get("X-Requested-With")
			if !strings.EqualFold(header, opts.Value) {
				if opts.ErrorHandler != nil {
					return opts.ErrorHandler(c)
				}
				return c.Text(http.StatusBadRequest, "X-Requested-With header required")
			}

			return next(c)
		}
	}
}

// Require creates middleware requiring a specific header value.
func Require(value string) mizu.Middleware {
	return WithOptions(Options{Value: value})
}

// AJAXOnly creates middleware that only allows AJAX requests.
func AJAXOnly() mizu.Middleware {
	return WithOptions(Options{
		Value:       "XMLHttpRequest",
		SkipMethods: []string{}, // Check all methods
	})
}

// IsAJAX checks if a request has the X-Requested-With header.
func IsAJAX(c *mizu.Ctx) bool {
	return strings.EqualFold(c.Request().Header.Get("X-Requested-With"), "XMLHttpRequest")
}
