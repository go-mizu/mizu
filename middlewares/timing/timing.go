// Package timing provides Server-Timing header middleware for Mizu.
package timing

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-mizu/mizu"
)

type contextKey struct{}

type timingData struct {
	mu      sync.Mutex
	metrics []metric
	start   time.Time
}

type metric struct {
	name        string
	duration    time.Duration
	description string
}

// New creates timing middleware.
func New() mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			td := &timingData{
				start: time.Now(),
			}

			ctx := context.WithValue(c.Context(), contextKey{}, td)
			req := c.Request().WithContext(ctx)
			*c.Request() = *req

			err := next(c)

			// Build Server-Timing header
			td.mu.Lock()
			defer td.mu.Unlock()

			var parts []string
			// Add total time
			total := time.Since(td.start)
			parts = append(parts, fmt.Sprintf("total;dur=%.2f", float64(total.Microseconds())/1000))

			// Add custom metrics
			for _, m := range td.metrics {
				part := fmt.Sprintf("%s;dur=%.2f", m.name, float64(m.duration.Microseconds())/1000)
				if m.description != "" {
					part += fmt.Sprintf(`;desc="%s"`, m.description)
				}
				parts = append(parts, part)
			}

			c.Header().Set("Server-Timing", strings.Join(parts, ", "))

			return err
		}
	}
}

// Add adds a metric to the Server-Timing header.
func Add(c *mizu.Ctx, name string, duration time.Duration, description string) {
	td, ok := c.Context().Value(contextKey{}).(*timingData)
	if !ok {
		return
	}

	td.mu.Lock()
	defer td.mu.Unlock()

	td.metrics = append(td.metrics, metric{
		name:        name,
		duration:    duration,
		description: description,
	})
}

// Start starts a timing measurement and returns a stop function.
func Start(c *mizu.Ctx, name string) func(description string) {
	start := time.Now()
	return func(description string) {
		Add(c, name, time.Since(start), description)
	}
}

// Track is a convenience function to track a named operation.
func Track(c *mizu.Ctx, name string, fn func()) {
	start := time.Now()
	fn()
	Add(c, name, time.Since(start), "")
}
