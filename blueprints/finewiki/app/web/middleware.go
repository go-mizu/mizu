// app/web/middleware.go
package web

import (
	"fmt"
	"log"
	"time"

	"github.com/go-mizu/mizu"
)

// Logging returns a middleware that logs each request.
func Logging() mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			start := time.Now()

			err := next(c)

			elapsed := time.Since(start)
			status := c.StatusCode()
			if status == 0 {
				status = 200
			}

			log.Printf("%s %s %d %s",
				c.Request().Method,
				c.Request().URL.Path,
				status,
				elapsed.Round(time.Microsecond),
			)

			return err
		}
	}
}

// Recovery returns a middleware that recovers from panics.
func Recovery() mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) (err error) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("panic recovered: %v", r)
					_ = c.Text(500, "Internal Server Error")
				}
			}()
			return next(c)
		}
	}
}

// RequestID returns a middleware that adds a request ID header.
func RequestID() mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			id := c.Request().Header.Get("X-Request-ID")
			if id == "" {
				id = generateRequestID()
			}
			c.Header().Set("X-Request-ID", id)
			return next(c)
		}
	}
}

// generateRequestID generates a simple request ID based on timestamp.
func generateRequestID() string {
	return time.Now().Format("20060102150405.000000")
}

// Cache returns a middleware that sets cache headers for static content.
func Cache(maxAge time.Duration) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			c.Header().Set("Cache-Control", "public, max-age="+formatSeconds(maxAge))
			return next(c)
		}
	}
}

// NoCache returns a middleware that prevents caching.
func NoCache() mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			c.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			c.Header().Set("Pragma", "no-cache")
			c.Header().Set("Expires", "0")
			return next(c)
		}
	}
}

func formatSeconds(d time.Duration) string {
	return fmt.Sprintf("%d", int(d.Seconds()))
}
