// Package version provides API versioning middleware for Mizu.
package version

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-mizu/mizu"
)

type contextKey struct{}

// Options configures the versioning middleware.
type Options struct {
	// DefaultVersion is used when version is not specified.
	DefaultVersion string

	// Header is the header name for version.
	// Default: "Accept-Version".
	Header string

	// QueryParam is the query parameter name.
	// Default: "version".
	QueryParam string

	// PathPrefix extracts version from URL path prefix (e.g., /v1/...).
	PathPrefix bool

	// Supported is a list of supported versions.
	// If empty, all versions are allowed.
	Supported []string

	// Deprecated is a list of deprecated versions.
	Deprecated []string

	// ErrorHandler handles unsupported versions.
	ErrorHandler func(c *mizu.Ctx, version string) error
}

// New creates versioning middleware.
func New(opts Options) mizu.Middleware {
	if opts.Header == "" {
		opts.Header = "Accept-Version"
	}
	if opts.QueryParam == "" {
		opts.QueryParam = "version"
	}

	supported := make(map[string]bool)
	for _, v := range opts.Supported {
		supported[v] = true
	}

	deprecated := make(map[string]bool)
	for _, v := range opts.Deprecated {
		deprecated[v] = true
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			version := ""

			// Try header first
			version = c.Request().Header.Get(opts.Header)

			// Try query parameter
			if version == "" {
				version = c.Query(opts.QueryParam)
			}

			// Try path prefix
			if version == "" && opts.PathPrefix {
				path := c.Request().URL.Path
				if len(path) > 1 {
					parts := strings.SplitN(path[1:], "/", 2)
					if len(parts) > 0 && isVersionString(parts[0]) {
						version = parts[0]
					}
				}
			}

			// Use default
			if version == "" {
				version = opts.DefaultVersion
			}

			// Check if supported
			if len(supported) > 0 && !supported[version] {
				if opts.ErrorHandler != nil {
					return opts.ErrorHandler(c, version)
				}
				return c.Text(http.StatusBadRequest, "Unsupported API version: "+version)
			}

			// Add deprecation header
			if deprecated[version] {
				c.Header().Set("Deprecation", "true")
				c.Header().Set("Sunset", "See documentation for migration guide")
			}

			// Store version in context
			ctx := context.WithValue(c.Context(), contextKey{}, version)
			req := c.Request().WithContext(ctx)
			*c.Request() = *req

			return next(c)
		}
	}
}

// FromHeader reads version from header.
func FromHeader(header string) mizu.Middleware {
	return New(Options{Header: header})
}

// FromPath extracts version from URL path.
func FromPath() mizu.Middleware {
	return New(Options{PathPrefix: true})
}

// FromQuery reads version from query parameter.
func FromQuery(param string) mizu.Middleware {
	return New(Options{QueryParam: param})
}

// GetVersion extracts version from context.
func GetVersion(c *mizu.Ctx) string {
	if v, ok := c.Context().Value(contextKey{}).(string); ok {
		return v
	}
	return ""
}

// Get is an alias for GetVersion.
func Get(c *mizu.Ctx) string {
	return GetVersion(c)
}

func isVersionString(s string) bool {
	if len(s) == 0 {
		return false
	}
	// Check for v1, v2, V1, etc.
	if (s[0] == 'v' || s[0] == 'V') && len(s) > 1 {
		for _, c := range s[1:] {
			if c < '0' || c > '9' {
				if c != '.' {
					return false
				}
			}
		}
		return true
	}
	return false
}
