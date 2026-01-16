package middleware

import (
	"sync"
	"time"

	"github.com/go-mizu/mizu"
)

// RateLimitConfig configures the rate limiter.
type RateLimitConfig struct {
	// Requests is the maximum number of requests allowed per window.
	Requests int
	// Window is the time window for rate limiting.
	Window time.Duration
	// KeyFunc extracts the rate limit key from the request (default: IP address).
	KeyFunc func(c *mizu.Ctx) string
}

// DefaultRateLimitConfig returns a default rate limit configuration.
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		Requests: 100,
		Window:   time.Minute,
		KeyFunc: func(c *mizu.Ctx) string {
			return c.Request().RemoteAddr
		},
	}
}

// AuthRateLimitConfig returns a stricter rate limit configuration for auth endpoints.
func AuthRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		Requests: 10, // Only 10 auth attempts per minute per IP
		Window:   time.Minute,
		KeyFunc: func(c *mizu.Ctx) string {
			return c.Request().RemoteAddr
		},
	}
}

type rateLimitEntry struct {
	count   int
	resetAt time.Time
}

// RateLimiter implements a simple in-memory rate limiter.
type RateLimiter struct {
	mu      sync.RWMutex
	entries map[string]*rateLimitEntry
	config  *RateLimitConfig
}

// NewRateLimiter creates a new rate limiter.
func NewRateLimiter(config *RateLimitConfig) *RateLimiter {
	if config == nil {
		config = DefaultRateLimitConfig()
	}

	rl := &RateLimiter{
		entries: make(map[string]*rateLimitEntry),
		config:  config,
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// cleanup periodically removes expired entries.
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, entry := range rl.entries {
			if entry.resetAt.Before(now) {
				delete(rl.entries, key)
			}
		}
		rl.mu.Unlock()
	}
}

// Allow checks if a request should be allowed.
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	entry, exists := rl.entries[key]

	if !exists || entry.resetAt.Before(now) {
		// Create new entry or reset expired entry
		rl.entries[key] = &rateLimitEntry{
			count:   1,
			resetAt: now.Add(rl.config.Window),
		}
		return true
	}

	if entry.count >= rl.config.Requests {
		return false
	}

	entry.count++
	return true
}

// RateLimit returns a middleware that rate limits requests.
func RateLimit(config *RateLimitConfig) mizu.Middleware {
	limiter := NewRateLimiter(config)

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			key := config.KeyFunc(c)

			if !limiter.Allow(key) {
				return c.JSON(429, map[string]any{
					"statusCode": 429,
					"error":      "Too Many Requests",
					"message":    "Rate limit exceeded. Please try again later.",
				})
			}

			return next(c)
		}
	}
}
