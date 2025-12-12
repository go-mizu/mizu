// Package maxconns provides maximum concurrent connections middleware for Mizu.
package maxconns

import (
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/go-mizu/mizu"
)

// Options configures the maxconns middleware.
type Options struct {
	// Max is the maximum number of concurrent connections.
	// Default: 100.
	Max int

	// PerIP limits connections per IP.
	// Default: 0 (no per-IP limit).
	PerIP int

	// ErrorHandler handles when limit is reached.
	ErrorHandler func(c *mizu.Ctx) error
}

// New creates maxconns middleware with a default limit.
func New(max int) mizu.Middleware {
	return WithOptions(Options{Max: max})
}

// WithOptions creates maxconns middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	// Max of 0 or negative means immediate rejection
	if opts.Max <= 0 {
		return func(next mizu.Handler) mizu.Handler {
			return func(c *mizu.Ctx) error {
				return handleLimit(c, opts)
			}
		}
	}

	var (
		current  int64
		perIP    = make(map[string]int)
		perIPMu  sync.RWMutex
	)

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			// Check global limit
			if atomic.LoadInt64(&current) >= int64(opts.Max) {
				return handleLimit(c, opts)
			}

			// Check per-IP limit if configured
			var ip string
			if opts.PerIP > 0 {
				ip = getClientIP(c)
				perIPMu.RLock()
				count := perIP[ip]
				perIPMu.RUnlock()

				if count >= opts.PerIP {
					return handleLimit(c, opts)
				}

				// Increment per-IP counter
				perIPMu.Lock()
				perIP[ip]++
				perIPMu.Unlock()

				defer func() {
					perIPMu.Lock()
					perIP[ip]--
					if perIP[ip] == 0 {
						delete(perIP, ip)
					}
					perIPMu.Unlock()
				}()
			}

			// Increment global counter
			atomic.AddInt64(&current, 1)
			defer atomic.AddInt64(&current, -1)

			return next(c)
		}
	}
}

func handleLimit(c *mizu.Ctx, opts Options) error {
	if opts.ErrorHandler != nil {
		return opts.ErrorHandler(c)
	}
	c.Header().Set("Retry-After", "60")
	return c.Text(http.StatusServiceUnavailable, "Too many connections")
}

func getClientIP(c *mizu.Ctx) string {
	if xff := c.Request().Header.Get("X-Forwarded-For"); xff != "" {
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}
	if xri := c.Request().Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	return c.Request().RemoteAddr
}

// PerIP creates middleware limiting connections per IP.
func PerIP(max int) mizu.Middleware {
	return WithOptions(Options{Max: 1000000, PerIP: max})
}

// Global creates middleware with a global connection limit.
func Global(max int) mizu.Middleware {
	return New(max)
}

// Counter provides a connection counter for monitoring.
type Counter struct {
	current int64
	max     int64
}

// NewCounter creates a new connection counter.
func NewCounter(max int) *Counter {
	return &Counter{max: int64(max)}
}

// Current returns the current number of connections.
func (c *Counter) Current() int64 {
	return atomic.LoadInt64(&c.current)
}

// Max returns the maximum allowed connections.
func (c *Counter) Max() int64 {
	return c.max
}

// Middleware returns a middleware using this counter.
func (c *Counter) Middleware() mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(ctx *mizu.Ctx) error {
			if atomic.LoadInt64(&c.current) >= c.max {
				ctx.Header().Set("Retry-After", "60")
				return ctx.Text(http.StatusServiceUnavailable, "Too many connections")
			}

			atomic.AddInt64(&c.current, 1)
			defer atomic.AddInt64(&c.current, -1)

			return next(ctx)
		}
	}
}
