package web

import (
	"net/http"
	"sync"
	"time"

	"github.com/go-mizu/mizu"
)

// RateLimiter provides IP-based rate limiting for authentication endpoints.
type RateLimiter struct {
	mu       sync.RWMutex
	requests map[string]*clientRequests
	limit    int           // Maximum requests per window
	window   time.Duration // Time window
}

type clientRequests struct {
	count     int
	resetTime time.Time
}

// NewRateLimiter creates a new rate limiter.
// limit: maximum number of requests allowed per window
// window: time window duration
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string]*clientRequests),
		limit:    limit,
		window:   window,
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// Allow checks if a request from the given IP should be allowed.
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	client, exists := rl.requests[ip]
	if !exists || now.After(client.resetTime) {
		rl.requests[ip] = &clientRequests{
			count:     1,
			resetTime: now.Add(rl.window),
		}
		return true
	}

	if client.count >= rl.limit {
		return false
	}

	client.count++
	return true
}

// cleanup periodically removes expired entries.
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, client := range rl.requests {
			if now.After(client.resetTime) {
				delete(rl.requests, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// getClientIP extracts the client IP from the request.
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxied requests)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Fall back to RemoteAddr
	return r.RemoteAddr
}

// RateLimit returns a middleware that rate limits requests.
func (rl *RateLimiter) RateLimit(next mizu.Handler) mizu.Handler {
	return func(c *mizu.Ctx) error {
		ip := getClientIP(c.Request())

		if !rl.Allow(ip) {
			return c.JSON(http.StatusTooManyRequests, map[string]string{
				"error": "too many requests, please try again later",
			})
		}

		return next(c)
	}
}

// AuthRateLimiter is a pre-configured rate limiter for authentication endpoints.
// It allows 10 requests per minute per IP.
var AuthRateLimiter = NewRateLimiter(10, time.Minute)
