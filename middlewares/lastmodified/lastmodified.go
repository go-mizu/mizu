// Package lastmodified provides Last-Modified header handling middleware for Mizu.
package lastmodified

import (
	"net/http"
	"time"

	"github.com/go-mizu/mizu"
)

// Options configures the lastmodified middleware.
type Options struct {
	// TimeFunc returns the last modified time for a request.
	TimeFunc func(c *mizu.Ctx) time.Time

	// SkipPaths are paths to skip.
	SkipPaths []string
}

// New creates lastmodified middleware with a time function.
func New(timeFunc func(c *mizu.Ctx) time.Time) mizu.Middleware {
	return WithOptions(Options{TimeFunc: timeFunc})
}

// WithOptions creates lastmodified middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	skipPaths := make(map[string]bool)
	for _, p := range opts.SkipPaths {
		skipPaths[p] = true
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			// Skip configured paths
			if skipPaths[c.Request().URL.Path] {
				return next(c)
			}

			// Only check GET and HEAD
			method := c.Request().Method
			if method != http.MethodGet && method != http.MethodHead {
				return next(c)
			}

			// Get last modified time
			var lastMod time.Time
			if opts.TimeFunc != nil {
				lastMod = opts.TimeFunc(c)
			}

			if lastMod.IsZero() {
				return next(c)
			}

			// Set Last-Modified header
			lastModStr := lastMod.UTC().Format(http.TimeFormat)
			c.Header().Set("Last-Modified", lastModStr)

			// Check If-Modified-Since
			ifModSince := c.Request().Header.Get("If-Modified-Since")
			if ifModSince != "" {
				t, err := http.ParseTime(ifModSince)
				if err == nil && !lastMod.Truncate(time.Second).After(t) {
					c.Writer().WriteHeader(http.StatusNotModified)
					return nil
				}
			}

			return next(c)
		}
	}
}

// Static creates middleware with a static last modified time.
func Static(t time.Time) mizu.Middleware {
	return New(func(_ *mizu.Ctx) time.Time {
		return t
	})
}

// Now creates middleware that sets current time as last modified.
func Now() mizu.Middleware {
	return New(func(_ *mizu.Ctx) time.Time {
		return time.Now()
	})
}

// StartupTime creates middleware using application startup time.
func StartupTime() mizu.Middleware {
	startTime := time.Now()
	return New(func(_ *mizu.Ctx) time.Time {
		return startTime
	})
}

// FromHeader creates middleware that reads time from a header.
func FromHeader(header string) mizu.Middleware {
	return New(func(c *mizu.Ctx) time.Time {
		h := c.Request().Header.Get(header)
		if h == "" {
			return time.Time{}
		}
		t, _ := http.ParseTime(h)
		return t
	})
}
