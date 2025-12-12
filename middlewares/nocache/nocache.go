// Package nocache provides middleware to prevent caching.
package nocache

import "github.com/go-mizu/mizu"

// New creates middleware that sets headers to prevent caching.
func New() mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			h := c.Header()
			h.Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate, max-age=0")
			h.Set("Pragma", "no-cache")
			h.Set("Expires", "0")
			h.Set("Surrogate-Control", "no-store")
			return next(c)
		}
	}
}
