// Package fingerprint provides request fingerprinting middleware for Mizu.
package fingerprint

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"

	"github.com/go-mizu/mizu"
)

type contextKey struct{}

// Info contains fingerprint information.
type Info struct {
	Hash       string
	Components map[string]string
}

// Options configures the fingerprint middleware.
type Options struct {
	// Headers to include in fingerprint.
	// Default: common headers.
	Headers []string

	// IncludeIP includes client IP in fingerprint.
	// Default: true.
	IncludeIP bool

	// IncludeMethod includes HTTP method.
	// Default: false.
	IncludeMethod bool

	// IncludePath includes request path.
	// Default: false.
	IncludePath bool

	// Custom adds custom components to fingerprint.
	Custom func(c *mizu.Ctx) map[string]string
}

// Default headers for fingerprinting
var defaultHeaders = []string{
	"User-Agent",
	"Accept",
	"Accept-Language",
	"Accept-Encoding",
	"Connection",
	"Sec-Ch-Ua",
	"Sec-Ch-Ua-Mobile",
	"Sec-Ch-Ua-Platform",
}

// New creates fingerprint middleware with default options.
func New() mizu.Middleware {
	return WithOptions(Options{})
}

// WithOptions creates fingerprint middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	if len(opts.Headers) == 0 {
		opts.Headers = defaultHeaders
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			components := make(map[string]string)

			// Collect header components
			for _, header := range opts.Headers {
				if value := c.Request().Header.Get(header); value != "" {
					components[header] = value
				}
			}

			// Include IP if requested
			if opts.IncludeIP {
				components["IP"] = getClientIP(c)
			}

			// Include method if requested
			if opts.IncludeMethod {
				components["Method"] = c.Request().Method
			}

			// Include path if requested
			if opts.IncludePath {
				components["Path"] = c.Request().URL.Path
			}

			// Add custom components
			if opts.Custom != nil {
				for k, v := range opts.Custom(c) {
					components[k] = v
				}
			}

			// Generate hash
			hash := generateHash(components)

			info := &Info{
				Hash:       hash,
				Components: components,
			}

			// Store in context
			ctx := context.WithValue(c.Context(), contextKey{}, info)
			req := c.Request().WithContext(ctx)
			*c.Request() = *req

			return next(c)
		}
	}
}

// Get retrieves fingerprint info from context.
func Get(c *mizu.Ctx) *Info {
	if info, ok := c.Context().Value(contextKey{}).(*Info); ok {
		return info
	}
	return &Info{Components: make(map[string]string)}
}

// Hash returns the fingerprint hash.
func Hash(c *mizu.Ctx) string {
	return Get(c).Hash
}

func generateHash(components map[string]string) string {
	// Sort keys for consistent ordering
	keys := make([]string, 0, len(components))
	for k := range components {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build hash input
	var builder strings.Builder
	for _, k := range keys {
		builder.WriteString(k)
		builder.WriteString(":")
		builder.WriteString(components[k])
		builder.WriteString("|")
	}

	// Generate SHA256 hash
	hash := sha256.Sum256([]byte(builder.String()))
	return hex.EncodeToString(hash[:])
}

func getClientIP(c *mizu.Ctx) string {
	if xff := c.Request().Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xri := c.Request().Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	return c.Request().RemoteAddr
}

// HeadersOnly creates middleware that fingerprints only specific headers.
func HeadersOnly(headers ...string) mizu.Middleware {
	return WithOptions(Options{Headers: headers})
}

// WithIP creates middleware that includes IP in fingerprint.
func WithIP() mizu.Middleware {
	return WithOptions(Options{IncludeIP: true})
}

// Full creates middleware with comprehensive fingerprinting.
func Full() mizu.Middleware {
	return WithOptions(Options{
		IncludeIP:     true,
		IncludeMethod: true,
		IncludePath:   true,
	})
}
