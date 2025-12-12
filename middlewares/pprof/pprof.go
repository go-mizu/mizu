// Package pprof provides profiling endpoints middleware for Mizu.
package pprof

import (
	"net/http/pprof"
	"strings"

	"github.com/go-mizu/mizu"
)

// Options configures the pprof middleware.
type Options struct {
	// Prefix is the URL prefix for pprof endpoints.
	// Default: "/debug/pprof".
	Prefix string
}

// New creates pprof middleware with default options.
func New() mizu.Middleware {
	return WithOptions(Options{})
}

// WithOptions creates pprof middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	if opts.Prefix == "" {
		opts.Prefix = "/debug/pprof"
	}
	opts.Prefix = strings.TrimSuffix(opts.Prefix, "/")

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			path := c.Request().URL.Path

			if !strings.HasPrefix(path, opts.Prefix) {
				return next(c)
			}

			subpath := strings.TrimPrefix(path, opts.Prefix)
			if subpath == "" || subpath == "/" {
				pprof.Index(c.Writer(), c.Request())
				return nil
			}

			switch subpath {
			case "/cmdline":
				pprof.Cmdline(c.Writer(), c.Request())
			case "/profile":
				pprof.Profile(c.Writer(), c.Request())
			case "/symbol":
				pprof.Symbol(c.Writer(), c.Request())
			case "/trace":
				pprof.Trace(c.Writer(), c.Request())
			default:
				// Handle named profiles like /heap, /goroutine, etc.
				name := strings.TrimPrefix(subpath, "/")
				pprof.Handler(name).ServeHTTP(c.Writer(), c.Request())
			}
			return nil
		}
	}
}
