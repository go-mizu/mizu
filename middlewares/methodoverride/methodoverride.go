// Package methodoverride provides HTTP method override middleware for Mizu.
package methodoverride

import (
	"net/http"
	"strings"

	"github.com/go-mizu/mizu"
)

// Options configures the method override middleware.
type Options struct {
	// Header is the header name for method override.
	// Default: "X-HTTP-Method-Override".
	Header string

	// FormField is the form field name for method override.
	// Default: "_method".
	FormField string

	// Methods is the list of allowed override methods.
	// Default: PUT, PATCH, DELETE.
	Methods []string
}

var defaultMethods = []string{"PUT", "PATCH", "DELETE"}

// New creates method override middleware.
func New() mizu.Middleware {
	return WithOptions(Options{})
}

// WithOptions creates method override middleware with options.
func WithOptions(opts Options) mizu.Middleware {
	if opts.Header == "" {
		opts.Header = "X-HTTP-Method-Override"
	}
	if opts.FormField == "" {
		opts.FormField = "_method"
	}
	if len(opts.Methods) == 0 {
		opts.Methods = defaultMethods
	}

	allowedMethods := make(map[string]bool)
	for _, m := range opts.Methods {
		allowedMethods[strings.ToUpper(m)] = true
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			// Only override POST requests
			if c.Request().Method != http.MethodPost {
				return next(c)
			}

			// Check header first
			override := c.Request().Header.Get(opts.Header)
			if override == "" {
				// Check query parameter
				override = c.Query(opts.FormField)
			}
			if override == "" {
				// Check form field (only for form content types)
				ct := c.Request().Header.Get("Content-Type")
				if strings.HasPrefix(ct, "application/x-www-form-urlencoded") ||
					strings.HasPrefix(ct, "multipart/form-data") {
					if form, err := c.Form(); err == nil {
						override = form.Get(opts.FormField)
					}
				}
			}

			override = strings.ToUpper(override)
			if override != "" && allowedMethods[override] {
				c.Request().Method = override
			}

			return next(c)
		}
	}
}
