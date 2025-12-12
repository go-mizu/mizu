// Package contenttype provides Content-Type validation middleware for Mizu.
package contenttype

import (
	"net/http"
	"strings"

	"github.com/go-mizu/mizu"
)

// Require creates middleware requiring specific content types.
func Require(types ...string) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			// Only check requests with body
			method := c.Request().Method
			if method != http.MethodPost && method != http.MethodPut &&
				method != http.MethodPatch {
				return next(c)
			}

			ct := c.Request().Header.Get("Content-Type")
			if ct == "" {
				return c.Text(http.StatusUnsupportedMediaType, "Content-Type required")
			}

			// Extract media type without parameters
			mediaType := ct
			if idx := strings.Index(ct, ";"); idx != -1 {
				mediaType = strings.TrimSpace(ct[:idx])
			}

			for _, t := range types {
				if strings.EqualFold(mediaType, t) {
					return next(c)
				}
			}

			return c.Text(http.StatusUnsupportedMediaType, "Unsupported Media Type")
		}
	}
}

// RequireJSON requires application/json content type.
func RequireJSON() mizu.Middleware {
	return Require("application/json")
}

// RequireForm requires form content types.
func RequireForm() mizu.Middleware {
	return Require("application/x-www-form-urlencoded", "multipart/form-data")
}

// Default sets default Content-Type if not present.
func Default(contentType string) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			if c.Request().Header.Get("Content-Type") == "" {
				c.Request().Header.Set("Content-Type", contentType)
			}
			return next(c)
		}
	}
}

// SetResponse sets the response Content-Type header.
func SetResponse(contentType string) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			c.Header().Set("Content-Type", contentType)
			return next(c)
		}
	}
}
