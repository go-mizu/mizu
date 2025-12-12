// Package canary provides canary release middleware for Mizu.
package canary

import (
	"context"
	"math/rand"
	"sync/atomic"

	"github.com/go-mizu/mizu"
)

type contextKey struct{}

// Release represents a canary release configuration.
type Release struct {
	// Percentage is the percentage of traffic to route to canary.
	// Range: 0-100.
	Percentage int

	// counter for round-robin (internal)
	counter uint64
}

// Options configures the canary middleware.
type Options struct {
	// Percentage is the default canary percentage.
	// Default: 10.
	Percentage int

	// Cookie stores canary assignment in cookie.
	// Default: "".
	Cookie string

	// Header is the header to check for canary override.
	// Default: "X-Canary".
	Header string

	// Selector determines if request should use canary.
	// Takes precedence over Percentage.
	Selector func(c *mizu.Ctx) bool
}

// New creates canary middleware with percentage.
func New(percentage int) mizu.Middleware {
	return WithOptions(Options{Percentage: percentage})
}

// WithOptions creates canary middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	if opts.Header == "" {
		opts.Header = "X-Canary"
	}

	var counter uint64

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			var isCanary bool

			// Check header override
			if header := c.Request().Header.Get(opts.Header); header != "" {
				isCanary = header == "true" || header == "1"
			} else if opts.Cookie != "" {
				// Check cookie
				if cookie, err := c.Cookie(opts.Cookie); err == nil {
					isCanary = cookie.Value == "true" || cookie.Value == "1"
				}
			}

			// Use selector if provided
			if !isCanary && opts.Selector != nil {
				isCanary = opts.Selector(c)
			}

			// Fall back to percentage-based selection
			if !isCanary {
				// Use counter for deterministic distribution
				current := atomic.AddUint64(&counter, 1)
				isCanary = int(current%100) < opts.Percentage
			}

			// Store in context
			ctx := context.WithValue(c.Context(), contextKey{}, isCanary)
			req := c.Request().WithContext(ctx)
			*c.Request() = *req

			return next(c)
		}
	}
}

// IsCanary returns whether the request is using canary.
func IsCanary(c *mizu.Ctx) bool {
	if val, ok := c.Context().Value(contextKey{}).(bool); ok {
		return val
	}
	return false
}

// Route routes to different handlers based on canary status.
func Route(canary, stable mizu.Handler) mizu.Handler {
	return func(c *mizu.Ctx) error {
		if IsCanary(c) {
			return canary(c)
		}
		return stable(c)
	}
}

// Middleware creates middleware that applies different middlewares.
func Middleware(canaryMw, stableMw mizu.Middleware) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		canaryHandler := canaryMw(next)
		stableHandler := stableMw(next)

		return func(c *mizu.Ctx) error {
			if IsCanary(c) {
				return canaryHandler(c)
			}
			return stableHandler(c)
		}
	}
}

// ReleaseManager manages canary releases.
type ReleaseManager struct {
	releases map[string]*Release
}

// NewReleaseManager creates a new release manager.
func NewReleaseManager() *ReleaseManager {
	return &ReleaseManager{
		releases: make(map[string]*Release),
	}
}

// Set sets a canary release configuration.
func (m *ReleaseManager) Set(name string, percentage int) {
	m.releases[name] = &Release{Percentage: percentage}
}

// Get gets a canary release configuration.
func (m *ReleaseManager) Get(name string) *Release {
	return m.releases[name]
}

// ShouldUseCanary determines if a request should use canary.
func (m *ReleaseManager) ShouldUseCanary(name string) bool {
	release := m.releases[name]
	if release == nil {
		return false
	}

	current := atomic.AddUint64(&release.counter, 1)
	return int(current%100) < release.Percentage
}

// RandomSelector creates a random-based selector.
func RandomSelector(percentage int) func(*mizu.Ctx) bool {
	return func(c *mizu.Ctx) bool {
		return rand.Intn(100) < percentage
	}
}

// HeaderSelector creates a header-based selector.
func HeaderSelector(header, value string) func(*mizu.Ctx) bool {
	return func(c *mizu.Ctx) bool {
		return c.Request().Header.Get(header) == value
	}
}

// CookieSelector creates a cookie-based selector.
func CookieSelector(name, value string) func(*mizu.Ctx) bool {
	return func(c *mizu.Ctx) bool {
		if cookie, err := c.Cookie(name); err == nil {
			return cookie.Value == value
		}
		return false
	}
}
