// Package requestid provides request ID generation and propagation middleware.
package requestid

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"github.com/go-mizu/mizu"
)

type contextKey struct{}

// Options configures the request ID middleware.
type Options struct {
	// Header is the header name for request ID.
	// Default: "X-Request-ID".
	Header string

	// Generator generates new request IDs.
	// Default: UUID v4 style random ID.
	Generator func() string

	// ContextKey is unused; use FromContext instead.
	ContextKey string
}

// New creates a request ID middleware with default options.
func New() mizu.Middleware {
	return WithOptions(Options{})
}

// WithOptions creates a request ID middleware with the specified options.
func WithOptions(opts Options) mizu.Middleware {
	if opts.Header == "" {
		opts.Header = "X-Request-ID"
	}
	if opts.Generator == nil {
		opts.Generator = generateID
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			id := c.Request().Header.Get(opts.Header)
			if id == "" {
				id = opts.Generator()
			}

			// Store in context
			ctx := context.WithValue(c.Context(), contextKey{}, id)
			req := c.Request().WithContext(ctx)
			*c.Request() = *req

			// Set response header
			c.Header().Set(opts.Header, id)

			return next(c)
		}
	}
}

// FromContext extracts the request ID from the context.
func FromContext(c *mizu.Ctx) string {
	if id, ok := c.Context().Value(contextKey{}).(string); ok {
		return id
	}
	return ""
}

// Get is an alias for FromContext.
func Get(c *mizu.Ctx) string {
	return FromContext(c)
}

// generateID generates a random ID similar to UUID v4.
func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	// Set version (4) and variant (2) bits for UUID v4 compatibility
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return hex.EncodeToString(b[:4]) + "-" +
		hex.EncodeToString(b[4:6]) + "-" +
		hex.EncodeToString(b[6:8]) + "-" +
		hex.EncodeToString(b[8:10]) + "-" +
		hex.EncodeToString(b[10:])
}
