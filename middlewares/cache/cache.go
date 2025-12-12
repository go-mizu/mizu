// Package cache provides Cache-Control header middleware for Mizu.
package cache

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/mizu"
)

// Options configures the cache middleware.
type Options struct {
	MaxAge               time.Duration
	SMaxAge              time.Duration
	Public               bool
	Private              bool
	NoCache              bool
	NoStore              bool
	NoTransform          bool
	MustRevalidate       bool
	ProxyRevalidate      bool
	Immutable            bool
	StaleWhileRevalidate time.Duration
	StaleIfError         time.Duration
}

// New creates cache middleware with max-age.
func New(maxAge time.Duration) mizu.Middleware {
	return WithOptions(Options{MaxAge: maxAge})
}

// WithOptions creates cache middleware with options.
func WithOptions(opts Options) mizu.Middleware {
	cacheControl := buildCacheControl(opts)

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			err := next(c)

			// Only set cache headers for successful responses
			// and if not already set
			if c.Header().Get("Cache-Control") == "" {
				c.Header().Set("Cache-Control", cacheControl)
			}

			return err
		}
	}
}

func buildCacheControl(opts Options) string {
	var parts []string

	if opts.Public {
		parts = append(parts, "public")
	}
	if opts.Private {
		parts = append(parts, "private")
	}
	if opts.NoCache {
		parts = append(parts, "no-cache")
	}
	if opts.NoStore {
		parts = append(parts, "no-store")
	}
	if opts.NoTransform {
		parts = append(parts, "no-transform")
	}
	if opts.MustRevalidate {
		parts = append(parts, "must-revalidate")
	}
	if opts.ProxyRevalidate {
		parts = append(parts, "proxy-revalidate")
	}
	if opts.MaxAge > 0 {
		parts = append(parts, fmt.Sprintf("max-age=%d", int(opts.MaxAge.Seconds())))
	}
	if opts.SMaxAge > 0 {
		parts = append(parts, fmt.Sprintf("s-maxage=%d", int(opts.SMaxAge.Seconds())))
	}
	if opts.Immutable {
		parts = append(parts, "immutable")
	}
	if opts.StaleWhileRevalidate > 0 {
		parts = append(parts, fmt.Sprintf("stale-while-revalidate=%d", int(opts.StaleWhileRevalidate.Seconds())))
	}
	if opts.StaleIfError > 0 {
		parts = append(parts, fmt.Sprintf("stale-if-error=%d", int(opts.StaleIfError.Seconds())))
	}

	if len(parts) == 0 {
		return "no-cache"
	}
	return strings.Join(parts, ", ")
}

// Public creates middleware for public cacheable content.
func Public(maxAge time.Duration) mizu.Middleware {
	return WithOptions(Options{
		Public: true,
		MaxAge: maxAge,
	})
}

// Private creates middleware for private cacheable content.
func Private(maxAge time.Duration) mizu.Middleware {
	return WithOptions(Options{
		Private: true,
		MaxAge:  maxAge,
	})
}

// Immutable creates middleware for immutable content.
func Immutable(maxAge time.Duration) mizu.Middleware {
	return WithOptions(Options{
		Public:    true,
		MaxAge:    maxAge,
		Immutable: true,
	})
}

// Static creates middleware suitable for static assets.
func Static(maxAge time.Duration) mizu.Middleware {
	return WithOptions(Options{
		Public:    true,
		MaxAge:    maxAge,
		Immutable: true,
	})
}

// SWR creates middleware with stale-while-revalidate.
func SWR(maxAge, stale time.Duration) mizu.Middleware {
	return WithOptions(Options{
		MaxAge:               maxAge,
		StaleWhileRevalidate: stale,
	})
}
