// Package ratelimit provides rate limiting middleware for Mizu.
package ratelimit

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-mizu/mizu"
)

// Options configures the rate limit middleware.
type Options struct {
	// Rate is the number of requests allowed per interval.
	Rate int

	// Interval is the time window for rate limiting.
	Interval time.Duration

	// Burst is the maximum burst capacity.
	// Default: same as Rate.
	Burst int

	// KeyFunc extracts the rate limit key from the request.
	// Default: client IP.
	KeyFunc func(c *mizu.Ctx) string

	// Headers enables rate limit headers in response.
	// Default: true.
	Headers bool

	// ErrorHandler handles rate limit exceeded.
	ErrorHandler func(c *mizu.Ctx) error

	// Skip is a function to skip rate limiting for certain requests.
	Skip func(c *mizu.Ctx) bool
}

// RateLimitInfo contains rate limit status.
type RateLimitInfo struct {
	Limit     int
	Remaining int
	Reset     time.Time
}

// Store is the interface for rate limit storage.
type Store interface {
	Allow(key string, rate int, interval time.Duration, burst int) (bool, RateLimitInfo)
}

// New creates rate limiter with requests per interval.
func New(rate int, interval time.Duration) mizu.Middleware {
	return WithOptions(Options{
		Rate:     rate,
		Interval: interval,
	})
}

// WithOptions creates rate limiter with options.
func WithOptions(opts Options) mizu.Middleware {
	return WithStore(NewMemoryStore(), opts)
}

// WithStore creates rate limiter with custom store.
//
//nolint:cyclop // Rate limiting requires multiple limit and key extraction checks
func WithStore(store Store, opts Options) mizu.Middleware {
	if opts.Rate <= 0 {
		opts.Rate = 100
	}
	if opts.Interval <= 0 {
		opts.Interval = time.Minute
	}
	if opts.Burst <= 0 {
		opts.Burst = opts.Rate
	}
	if opts.KeyFunc == nil {
		opts.KeyFunc = func(c *mizu.Ctx) string {
			return c.ClientIP()
		}
	}
	if !opts.Headers {
		opts.Headers = true
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			if opts.Skip != nil && opts.Skip(c) {
				return next(c)
			}

			key := opts.KeyFunc(c)
			allowed, info := store.Allow(key, opts.Rate, opts.Interval, opts.Burst)

			if opts.Headers {
				c.Header().Set("X-RateLimit-Limit", strconv.Itoa(info.Limit))
				c.Header().Set("X-RateLimit-Remaining", strconv.Itoa(info.Remaining))
				c.Header().Set("X-RateLimit-Reset", strconv.FormatInt(info.Reset.Unix(), 10))
			}

			if !allowed {
				c.Header().Set("Retry-After", strconv.Itoa(int(time.Until(info.Reset).Seconds())+1))
				if opts.ErrorHandler != nil {
					return opts.ErrorHandler(c)
				}
				return c.Text(http.StatusTooManyRequests, "Too Many Requests")
			}

			return next(c)
		}
	}
}

// MemoryStore is an in-memory rate limit store.
type MemoryStore struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	cleanup time.Duration
}

type bucket struct {
	tokens    float64
	lastCheck time.Time
	rate      int
	interval  time.Duration
}

// NewMemoryStore creates a new in-memory store.
func NewMemoryStore() *MemoryStore {
	s := &MemoryStore{
		buckets: make(map[string]*bucket),
		cleanup: 10 * time.Minute,
	}
	go s.cleanupLoop()
	return s
}

// Allow checks if a request is allowed.
func (s *MemoryStore) Allow(key string, rate int, interval time.Duration, burst int) (bool, RateLimitInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	b, ok := s.buckets[key]
	if !ok {
		b = &bucket{
			tokens:    float64(burst),
			lastCheck: now,
			rate:      rate,
			interval:  interval,
		}
		s.buckets[key] = b
	}

	// Token bucket algorithm
	elapsed := now.Sub(b.lastCheck)
	b.lastCheck = now

	// Add tokens based on elapsed time
	tokensToAdd := float64(rate) * elapsed.Seconds() / interval.Seconds()
	b.tokens += tokensToAdd
	if b.tokens > float64(burst) {
		b.tokens = float64(burst)
	}

	// Calculate reset time
	reset := now.Add(interval)

	info := RateLimitInfo{
		Limit:     rate,
		Remaining: int(b.tokens),
		Reset:     reset,
	}

	if b.tokens < 1 {
		return false, info
	}

	b.tokens--
	info.Remaining = int(b.tokens)
	return true, info
}

func (s *MemoryStore) cleanupLoop() {
	ticker := time.NewTicker(s.cleanup)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for key, b := range s.buckets {
			if now.Sub(b.lastCheck) > s.cleanup {
				delete(s.buckets, key)
			}
		}
		s.mu.Unlock()
	}
}

// PerSecond creates rate limiter allowing n requests per second.
func PerSecond(n int) mizu.Middleware {
	return New(n, time.Second)
}

// PerMinute creates rate limiter allowing n requests per minute.
func PerMinute(n int) mizu.Middleware {
	return New(n, time.Minute)
}

// PerHour creates rate limiter allowing n requests per hour.
func PerHour(n int) mizu.Middleware {
	return New(n, time.Hour)
}
