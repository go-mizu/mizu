// Package middleware provides HTTP middleware for the messaging application.
package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/go-mizu/mizu"
)

// RateLimiter implements a simple in-memory rate limiter using the token bucket algorithm.
type RateLimiter struct {
	mu       sync.RWMutex
	entries  map[string]*rateLimitEntry
	limit    int
	window   time.Duration
	stopCh   chan struct{}
}

type rateLimitEntry struct {
	count     int
	expiresAt time.Time
	mu        sync.Mutex
}

// NewRateLimiter creates a new rate limiter.
// limit: maximum number of requests allowed in the window
// window: time window for rate limiting
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		entries: make(map[string]*rateLimitEntry),
		limit:   limit,
		window:  window,
		stopCh:  make(chan struct{}),
	}
	go rl.cleanup()
	return rl
}

// Allow checks if a request from the given key should be allowed.
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	entry, exists := rl.entries[key]
	if !exists {
		entry = &rateLimitEntry{
			expiresAt: time.Now().Add(rl.window),
		}
		rl.entries[key] = entry
	}
	rl.mu.Unlock()

	entry.mu.Lock()
	defer entry.mu.Unlock()

	now := time.Now()
	if now.After(entry.expiresAt) {
		entry.count = 0
		entry.expiresAt = now.Add(rl.window)
	}

	entry.count++
	return entry.count <= rl.limit
}

// Remaining returns the number of requests remaining for the given key.
func (rl *RateLimiter) Remaining(key string) int {
	rl.mu.RLock()
	entry, exists := rl.entries[key]
	rl.mu.RUnlock()

	if !exists {
		return rl.limit
	}

	entry.mu.Lock()
	defer entry.mu.Unlock()

	if time.Now().After(entry.expiresAt) {
		return rl.limit
	}

	remaining := rl.limit - entry.count
	if remaining < 0 {
		return 0
	}
	return remaining
}

// Stop stops the cleanup goroutine.
func (rl *RateLimiter) Stop() {
	close(rl.stopCh)
}

// cleanup periodically removes expired entries.
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.window * 2)
	defer ticker.Stop()

	for {
		select {
		case <-rl.stopCh:
			return
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			for key, entry := range rl.entries {
				entry.mu.Lock()
				if now.After(entry.expiresAt) {
					delete(rl.entries, key)
				}
				entry.mu.Unlock()
			}
			rl.mu.Unlock()
		}
	}
}

// getClientIP extracts the client IP from the request.
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for reverse proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the chain
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	addr := r.RemoteAddr
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[:i]
		}
	}
	return addr
}

// RateLimit returns a middleware that rate limits requests by client IP.
func RateLimit(limiter *RateLimiter) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			ip := getClientIP(c.Request())

			if !limiter.Allow(ip) {
				c.Writer().Header().Set("Retry-After", "60")
				c.Writer().Header().Set("X-RateLimit-Remaining", "0")
				c.Writer().WriteHeader(http.StatusTooManyRequests)
				return c.JSON(http.StatusTooManyRequests, map[string]any{
					"success": false,
					"error":   "Too many requests. Please try again later.",
				})
			}

			c.Writer().Header().Set("X-RateLimit-Remaining",
				string(rune('0'+limiter.Remaining(ip))))

			return next(c)
		}
	}
}

// RateLimitByKey returns a middleware that rate limits by a custom key function.
func RateLimitByKey(limiter *RateLimiter, keyFn func(*mizu.Ctx) string) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			key := keyFn(c)
			if key == "" {
				key = getClientIP(c.Request())
			}

			if !limiter.Allow(key) {
				c.Writer().Header().Set("Retry-After", "60")
				c.Writer().WriteHeader(http.StatusTooManyRequests)
				return c.JSON(http.StatusTooManyRequests, map[string]any{
					"success": false,
					"error":   "Too many requests. Please try again later.",
				})
			}

			return next(c)
		}
	}
}

// Common rate limiters for different endpoints
var (
	// LoginLimiter: 5 attempts per IP per minute
	LoginLimiter = NewRateLimiter(5, time.Minute)

	// RegisterLimiter: 3 attempts per IP per 10 minutes
	RegisterLimiter = NewRateLimiter(3, 10*time.Minute)

	// APILimiter: 100 requests per minute per user/IP
	APILimiter = NewRateLimiter(100, time.Minute)

	// WebSocketLimiter: 10 connections per IP per minute
	WebSocketLimiter = NewRateLimiter(10, time.Minute)
)
