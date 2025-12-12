// Package vary provides Vary header management middleware for Mizu.
package vary

import (
	"strings"

	"github.com/go-mizu/mizu"
)

// Options configures the vary middleware.
type Options struct {
	// Headers to include in Vary.
	Headers []string

	// Auto automatically detects headers to add to Vary.
	// Default: false.
	Auto bool
}

// New creates vary middleware with specified headers.
func New(headers ...string) mizu.Middleware {
	return WithOptions(Options{Headers: headers})
}

// WithOptions creates vary middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			// Execute handler first
			err := next(c)

			// Add Vary headers
			if len(opts.Headers) > 0 {
				add(c, opts.Headers...)
			}

			// Auto-detect
			if opts.Auto {
				autoDetect(c)
			}

			return err
		}
	}
}

// add adds headers to the Vary response header.
func add(c *mizu.Ctx, headers ...string) {
	existing := c.Header().Get("Vary")
	existingSet := make(map[string]bool)

	if existing != "" {
		for _, h := range strings.Split(existing, ",") {
			existingSet[strings.TrimSpace(strings.ToLower(h))] = true
		}
	}

	var toAdd []string
	for _, h := range headers {
		if !existingSet[strings.ToLower(h)] {
			toAdd = append(toAdd, h)
			existingSet[strings.ToLower(h)] = true
		}
	}

	if len(toAdd) > 0 {
		if existing != "" {
			c.Header().Set("Vary", existing+", "+strings.Join(toAdd, ", "))
		} else {
			c.Header().Set("Vary", strings.Join(toAdd, ", "))
		}
	}
}

// autoDetect automatically adds common Vary headers.
func autoDetect(c *mizu.Ctx) {
	// Check if content negotiation headers were used
	if c.Request().Header.Get("Accept") != "" {
		add(c, "Accept")
	}
	if c.Request().Header.Get("Accept-Encoding") != "" {
		add(c, "Accept-Encoding")
	}
	if c.Request().Header.Get("Accept-Language") != "" {
		add(c, "Accept-Language")
	}
}

// Add is a helper to add Vary headers within a handler.
func Add(c *mizu.Ctx, headers ...string) {
	add(c, headers...)
}

// AcceptEncoding creates middleware that adds Accept-Encoding to Vary.
func AcceptEncoding() mizu.Middleware {
	return New("Accept-Encoding")
}

// Accept creates middleware that adds Accept to Vary.
func Accept() mizu.Middleware {
	return New("Accept")
}

// AcceptLanguage creates middleware that adds Accept-Language to Vary.
func AcceptLanguage() mizu.Middleware {
	return New("Accept-Language")
}

// Origin creates middleware that adds Origin to Vary (for CORS).
func Origin() mizu.Middleware {
	return New("Origin")
}

// All creates middleware with common Vary headers.
func All() mizu.Middleware {
	return New("Accept", "Accept-Encoding", "Accept-Language")
}

// Auto creates middleware that auto-detects Vary headers.
func Auto() mizu.Middleware {
	return WithOptions(Options{Auto: true})
}
