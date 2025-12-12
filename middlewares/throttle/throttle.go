// Package throttle provides request throttling middleware for Mizu.
package throttle

import (
	"net/http"
	"sync"
	"time"

	"github.com/go-mizu/mizu"
)

// Options configures the throttle middleware.
type Options struct {
	// Limit is the maximum concurrent requests.
	// Default: 100.
	Limit int

	// Backlog is the maximum number of queued requests waiting for a slot.
	// Set to 0 for no waiting queue (immediate rejection when all slots are busy).
	// Default: 1000.
	Backlog int

	// BacklogSet indicates whether Backlog was explicitly set.
	// This is used internally to distinguish between Backlog=0 (no backlog)
	// and unset (use default).
	BacklogSet bool

	// Timeout is how long to wait for a slot.
	// Default: 30s.
	Timeout time.Duration

	// OnThrottle is called when a request is throttled.
	OnThrottle func(c *mizu.Ctx)
}

// New creates throttle middleware with limit.
func New(limit int) mizu.Middleware {
	return WithOptions(Options{Limit: limit})
}

// WithOptions creates throttle middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	if opts.Limit == 0 {
		opts.Limit = 100
	}
	// Only use default backlog if not explicitly set
	// Check if BacklogSet or if Backlog > 0 (non-zero value means intentionally set)
	if !opts.BacklogSet && opts.Backlog == 0 {
		opts.Backlog = 1000
	}
	if opts.Timeout == 0 {
		opts.Timeout = 30 * time.Second
	}

	// Semaphore for limiting concurrent requests
	sem := make(chan struct{}, opts.Limit)

	// Fill semaphore
	for i := 0; i < opts.Limit; i++ {
		sem <- struct{}{}
	}

	// Backlog counter
	var backlogCount int
	var backlogMu sync.Mutex

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			// Try to get a slot immediately (non-blocking)
			select {
			case <-sem:
				// Got a slot immediately, process the request
				err := next(c)
				sem <- struct{}{}
				return err
			default:
				// No slot available, check backlog
			}

			// Check backlog capacity
			backlogMu.Lock()
			if backlogCount >= opts.Backlog {
				backlogMu.Unlock()
				if opts.OnThrottle != nil {
					opts.OnThrottle(c)
				}
				return c.Text(http.StatusServiceUnavailable, "service busy")
			}
			backlogCount++
			backlogMu.Unlock()

			// Wait for a slot with timeout
			timer := time.NewTimer(opts.Timeout)
			defer timer.Stop()

			select {
			case <-sem:
				// Got a slot
			case <-timer.C:
				backlogMu.Lock()
				backlogCount--
				backlogMu.Unlock()
				if opts.OnThrottle != nil {
					opts.OnThrottle(c)
				}
				return c.Text(http.StatusServiceUnavailable, "request timeout")
			case <-c.Request().Context().Done():
				backlogMu.Lock()
				backlogCount--
				backlogMu.Unlock()
				return c.Request().Context().Err()
			}

			// Process request
			err := next(c)

			// Release slot
			sem <- struct{}{}

			backlogMu.Lock()
			backlogCount--
			backlogMu.Unlock()

			return err
		}
	}
}

// Concurrency is an alias for New.
func Concurrency(limit int) mizu.Middleware {
	return New(limit)
}
