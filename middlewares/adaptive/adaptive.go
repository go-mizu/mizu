// Package adaptive provides adaptive rate limiting middleware for Mizu.
package adaptive

import (
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-mizu/mizu"
)

// Options configures the adaptive rate limiter.
type Options struct {
	// InitialRate is the starting rate per second.
	// Default: 100.
	InitialRate int

	// MinRate is the minimum rate.
	// Default: 10.
	MinRate int

	// MaxRate is the maximum rate.
	// Default: 1000.
	MaxRate int

	// TargetLatency is the target response latency.
	// Default: 100ms.
	TargetLatency time.Duration

	// ErrorThreshold is the error rate threshold (0-1).
	// Default: 0.1 (10%).
	ErrorThreshold float64

	// AdjustInterval is how often to adjust the rate.
	// Default: 1s.
	AdjustInterval time.Duration

	// KeyFunc extracts the rate limit key.
	// Default: client IP.
	KeyFunc func(c *mizu.Ctx) string

	// ErrorHandler handles rate limit errors.
	ErrorHandler func(c *mizu.Ctx) error
}

// Limiter is an adaptive rate limiter.
type Limiter struct {
	opts Options

	currentRate int64
	tokens      int64
	lastRefill  int64

	// Metrics
	requestCount int64
	errorCount   int64
	totalLatency int64

	mu sync.RWMutex
}

// NewLimiter creates a new adaptive limiter.
func NewLimiter(opts Options) *Limiter {
	if opts.InitialRate < 0 {
		opts.InitialRate = 100
	}
	if opts.MinRate < 0 {
		opts.MinRate = 10
	}
	if opts.MaxRate < 0 {
		opts.MaxRate = 1000
	}
	if opts.TargetLatency == 0 {
		opts.TargetLatency = 100 * time.Millisecond
	}
	if opts.ErrorThreshold == 0 {
		opts.ErrorThreshold = 0.1
	}
	if opts.AdjustInterval == 0 {
		opts.AdjustInterval = time.Second
	}

	l := &Limiter{
		opts:        opts,
		currentRate: int64(opts.InitialRate),
		tokens:      int64(opts.InitialRate),
		lastRefill:  time.Now().UnixNano(),
	}

	go l.adjustLoop()

	return l
}

func (l *Limiter) adjustLoop() {
	ticker := time.NewTicker(l.opts.AdjustInterval)
	defer ticker.Stop()

	for range ticker.C {
		l.adjust()
	}
}

func (l *Limiter) adjust() {
	l.mu.Lock()
	defer l.mu.Unlock()

	reqCount := atomic.SwapInt64(&l.requestCount, 0)
	errCount := atomic.SwapInt64(&l.errorCount, 0)
	totalLat := atomic.SwapInt64(&l.totalLatency, 0)

	if reqCount == 0 {
		return
	}

	// Calculate metrics
	errorRate := float64(errCount) / float64(reqCount)
	avgLatency := time.Duration(totalLat / reqCount)

	currentRate := atomic.LoadInt64(&l.currentRate)
	newRate := currentRate

	// Adjust based on error rate
	if errorRate > l.opts.ErrorThreshold {
		newRate = int64(float64(currentRate) * 0.8) // Reduce by 20%
	} else if avgLatency > l.opts.TargetLatency {
		newRate = int64(float64(currentRate) * 0.9) // Reduce by 10%
	} else if avgLatency < l.opts.TargetLatency/2 && errorRate < l.opts.ErrorThreshold/2 {
		newRate = int64(float64(currentRate) * 1.1) // Increase by 10%
	}

	// Clamp to bounds
	if newRate < int64(l.opts.MinRate) {
		newRate = int64(l.opts.MinRate)
	}
	if newRate > int64(l.opts.MaxRate) {
		newRate = int64(l.opts.MaxRate)
	}

	atomic.StoreInt64(&l.currentRate, newRate)
}

func (l *Limiter) allow() bool {
	now := time.Now().UnixNano()
	rate := atomic.LoadInt64(&l.currentRate)

	l.mu.Lock()
	defer l.mu.Unlock()

	// Refill tokens
	elapsed := now - l.lastRefill
	refill := (elapsed * rate) / int64(time.Second)
	l.tokens += refill
	if l.tokens > rate {
		l.tokens = rate
	}
	l.lastRefill = now

	if l.tokens > 0 {
		l.tokens--
		return true
	}
	return false
}

// Middleware returns the adaptive rate limiter middleware.
func (l *Limiter) Middleware() mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			if !l.allow() {
				if l.opts.ErrorHandler != nil {
					return l.opts.ErrorHandler(c)
				}
				c.Header().Set("Retry-After", "1")
				return c.Text(http.StatusTooManyRequests, "Rate limit exceeded")
			}

			start := time.Now()
			atomic.AddInt64(&l.requestCount, 1)

			err := next(c)

			latency := time.Since(start).Nanoseconds()
			atomic.AddInt64(&l.totalLatency, latency)

			if err != nil {
				atomic.AddInt64(&l.errorCount, 1)
			}

			return err
		}
	}
}

// CurrentRate returns the current rate limit.
func (l *Limiter) CurrentRate() int64 {
	return atomic.LoadInt64(&l.currentRate)
}

// Stats returns current limiter statistics.
func (l *Limiter) Stats() Stats {
	return Stats{
		CurrentRate:  atomic.LoadInt64(&l.currentRate),
		MinRate:      int64(l.opts.MinRate),
		MaxRate:      int64(l.opts.MaxRate),
		RequestCount: atomic.LoadInt64(&l.requestCount),
		ErrorCount:   atomic.LoadInt64(&l.errorCount),
	}
}

// Stats contains limiter statistics.
type Stats struct {
	CurrentRate  int64
	MinRate      int64
	MaxRate      int64
	RequestCount int64
	ErrorCount   int64
}

// New creates adaptive rate limiting middleware.
func New(opts Options) mizu.Middleware {
	l := NewLimiter(opts)
	return l.Middleware()
}

// Simple creates adaptive middleware with simple defaults.
func Simple() mizu.Middleware {
	return New(Options{
		InitialRate: 100,
		MinRate:     10,
		MaxRate:     1000,
	})
}

// HighThroughput creates middleware optimized for high throughput.
func HighThroughput() mizu.Middleware {
	return New(Options{
		InitialRate:   1000,
		MinRate:       100,
		MaxRate:       10000,
		TargetLatency: 50 * time.Millisecond,
	})
}

// Conservative creates middleware with conservative limits.
func Conservative() mizu.Middleware {
	return New(Options{
		InitialRate:    50,
		MinRate:        5,
		MaxRate:        200,
		TargetLatency:  200 * time.Millisecond,
		ErrorThreshold: 0.05,
	})
}
