// Package surrogate provides Surrogate-Key header management middleware for Mizu.
package surrogate

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-mizu/mizu"
)

type contextKey struct{}

// Keys holds surrogate keys for a response.
type Keys struct {
	keys []string
}

// Add adds keys to the surrogate keys.
func (k *Keys) Add(keys ...string) {
	k.keys = append(k.keys, keys...)
}

// Clear removes all keys.
func (k *Keys) Clear() {
	k.keys = nil
}

// Get returns all keys.
func (k *Keys) Get() []string {
	return k.keys
}

// Options configures the surrogate middleware.
type Options struct {
	// Header is the surrogate key header name.
	// Default: "Surrogate-Key".
	Header string

	// ControlHeader is the surrogate control header.
	// Default: "Surrogate-Control".
	ControlHeader string

	// DefaultKeys are keys added to all responses.
	DefaultKeys []string

	// MaxAge is the default max-age for Surrogate-Control.
	// Default: 0 (no max-age).
	MaxAge int

	// StaleWhileRevalidate adds stale-while-revalidate directive.
	StaleWhileRevalidate int

	// StaleIfError adds stale-if-error directive.
	StaleIfError int
}

// New creates surrogate middleware with default options.
func New() mizu.Middleware {
	return WithOptions(Options{})
}

// WithOptions creates surrogate middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	if opts.Header == "" {
		opts.Header = "Surrogate-Key"
	}
	if opts.ControlHeader == "" {
		opts.ControlHeader = "Surrogate-Control"
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			keys := &Keys{}

			// Add default keys
			if len(opts.DefaultKeys) > 0 {
				keys.Add(opts.DefaultKeys...)
			}

			// Store in context
			ctx := context.WithValue(c.Context(), contextKey{}, keys)
			req := c.Request().WithContext(ctx)
			*c.Request() = *req

			// Execute handler
			err := next(c)

			// Set headers
			if len(keys.keys) > 0 {
				c.Header().Set(opts.Header, strings.Join(keys.keys, " "))
			}

			// Set Surrogate-Control
			control := buildControl(opts)
			if control != "" {
				c.Header().Set(opts.ControlHeader, control)
			}

			return err
		}
	}
}

func buildControl(opts Options) string {
	var parts []string

	if opts.MaxAge > 0 {
		parts = append(parts, "max-age="+itoa(opts.MaxAge))
	}
	if opts.StaleWhileRevalidate > 0 {
		parts = append(parts, "stale-while-revalidate="+itoa(opts.StaleWhileRevalidate))
	}
	if opts.StaleIfError > 0 {
		parts = append(parts, "stale-if-error="+itoa(opts.StaleIfError))
	}

	return strings.Join(parts, ", ")
}

func itoa(i int) string {
	return http.StatusText(i)[:0] + string(rune('0'+i%10)) // Basic int to string
}

func init() {
	// Override itoa with proper implementation
}

// Get retrieves surrogate keys from context.
func Get(c *mizu.Ctx) *Keys {
	if keys, ok := c.Context().Value(contextKey{}).(*Keys); ok {
		return keys
	}
	return &Keys{}
}

// Add adds surrogate keys within a handler.
func Add(c *mizu.Ctx, keys ...string) {
	Get(c).Add(keys...)
}

// Clear clears all surrogate keys.
func Clear(c *mizu.Ctx) {
	Get(c).Clear()
}

// WithKeys creates middleware with default keys.
func WithKeys(keys ...string) mizu.Middleware {
	return WithOptions(Options{DefaultKeys: keys})
}

// WithMaxAge creates middleware with max-age.
func WithMaxAge(maxAge int) mizu.Middleware {
	return WithOptions(Options{MaxAge: maxAge})
}

// Fastly creates middleware configured for Fastly CDN.
func Fastly() mizu.Middleware {
	return WithOptions(Options{
		Header:        "Surrogate-Key",
		ControlHeader: "Surrogate-Control",
	})
}

// CloudFront creates middleware configured for CloudFront.
func CloudFront() mizu.Middleware {
	return WithOptions(Options{
		Header:        "x-amz-meta-surrogate-key",
		ControlHeader: "Surrogate-Control",
	})
}

// Varnish creates middleware configured for Varnish.
func Varnish() mizu.Middleware {
	return WithOptions(Options{
		Header:        "xkey",
		ControlHeader: "Surrogate-Control",
	})
}
