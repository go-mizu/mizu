// Package slash provides trailing slash handling middleware for Mizu.
package slash

import (
	"net/http"
	"strings"

	"github.com/go-mizu/mizu"
)

// Add redirects to URL with trailing slash.
func Add() mizu.Middleware {
	return AddCode(http.StatusMovedPermanently)
}

// AddCode redirects with specific status code.
func AddCode(code int) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			path := c.Request().URL.Path
			if path != "/" && !strings.HasSuffix(path, "/") {
				target := path + "/"
				if c.Request().URL.RawQuery != "" {
					target += "?" + c.Request().URL.RawQuery
				}
				return c.Redirect(code, target)
			}
			return next(c)
		}
	}
}

// Remove redirects to URL without trailing slash.
func Remove() mizu.Middleware {
	return RemoveCode(http.StatusMovedPermanently)
}

// RemoveCode redirects with specific status code.
func RemoveCode(code int) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			path := c.Request().URL.Path
			if path != "/" && strings.HasSuffix(path, "/") {
				target := strings.TrimSuffix(path, "/")
				if c.Request().URL.RawQuery != "" {
					target += "?" + c.Request().URL.RawQuery
				}
				return c.Redirect(code, target)
			}
			return next(c)
		}
	}
}
