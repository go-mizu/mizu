// Package chaos provides chaos engineering middleware for Mizu.
package chaos

import (
	"math/rand" //nolint:gosec // G404: Chaos testing uses math/rand intentionally - crypto/rand not needed
	"net/http"
	"time"

	"github.com/go-mizu/mizu"
)

// Options configures the chaos middleware.
type Options struct {
	// ErrorRate is the percentage of requests to fail (0-100).
	// Default: 0.
	ErrorRate int

	// ErrorCode is the HTTP status code to return on error.
	// Default: 500.
	ErrorCode int

	// LatencyMin is the minimum latency to add.
	// Default: 0.
	LatencyMin time.Duration

	// LatencyMax is the maximum latency to add.
	// Default: 0.
	LatencyMax time.Duration

	// Enabled enables chaos injection.
	// Default: false.
	Enabled bool

	// Selector determines which requests to apply chaos to.
	Selector func(c *mizu.Ctx) bool
}

// New creates chaos middleware with default options.
func New() mizu.Middleware {
	return WithOptions(Options{Enabled: true})
}

// WithOptions creates chaos middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	if opts.ErrorCode == 0 {
		opts.ErrorCode = http.StatusInternalServerError
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			if !opts.Enabled {
				return next(c)
			}

			// Check selector
			if opts.Selector != nil && !opts.Selector(c) {
				return next(c)
			}

			// Apply latency
			if opts.LatencyMax > 0 {
				latency := opts.LatencyMin
				if opts.LatencyMax > opts.LatencyMin {
					latency += time.Duration(rand.Int63n(int64(opts.LatencyMax - opts.LatencyMin))) //nolint:gosec // G404: Chaos testing uses weak RNG intentionally
				}
				time.Sleep(latency)
			}

			// Inject error
			if opts.ErrorRate > 0 && rand.Intn(100) < opts.ErrorRate { //nolint:gosec // G404: Chaos testing uses weak RNG intentionally
				return c.Text(opts.ErrorCode, "chaos: injected error")
			}

			return next(c)
		}
	}
}

// Error creates middleware that injects errors.
func Error(rate int, code int) mizu.Middleware {
	return WithOptions(Options{
		Enabled:   true,
		ErrorRate: rate,
		ErrorCode: code,
	})
}

// Latency creates middleware that injects latency.
func Latency(min, max time.Duration) mizu.Middleware {
	return WithOptions(Options{
		Enabled:    true,
		LatencyMin: min,
		LatencyMax: max,
	})
}

// Controller manages chaos configuration dynamically.
type Controller struct {
	opts     Options
	enabled  bool
}

// NewController creates a new chaos controller.
func NewController() *Controller {
	return &Controller{
		opts:    Options{ErrorCode: http.StatusInternalServerError},
		enabled: false,
	}
}

// Enable enables chaos injection.
func (c *Controller) Enable() {
	c.enabled = true
}

// Disable disables chaos injection.
func (c *Controller) Disable() {
	c.enabled = false
}

// IsEnabled returns whether chaos is enabled.
func (c *Controller) IsEnabled() bool {
	return c.enabled
}

// SetErrorRate sets the error rate.
func (c *Controller) SetErrorRate(rate int) {
	c.opts.ErrorRate = rate
}

// SetErrorCode sets the error code.
func (c *Controller) SetErrorCode(code int) {
	c.opts.ErrorCode = code
}

// SetLatency sets the latency range.
func (c *Controller) SetLatency(min, max time.Duration) {
	c.opts.LatencyMin = min
	c.opts.LatencyMax = max
}

// SetSelector sets the request selector.
func (c *Controller) SetSelector(selector func(*mizu.Ctx) bool) {
	c.opts.Selector = selector
}

// Middleware returns middleware using this controller.
func (c *Controller) Middleware() mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(ctx *mizu.Ctx) error {
			if !c.enabled {
				return next(ctx)
			}

			// Check selector
			if c.opts.Selector != nil && !c.opts.Selector(ctx) {
				return next(ctx)
			}

			// Apply latency
			if c.opts.LatencyMax > 0 {
				latency := c.opts.LatencyMin
				if c.opts.LatencyMax > c.opts.LatencyMin {
					latency += time.Duration(rand.Int63n(int64(c.opts.LatencyMax - c.opts.LatencyMin))) //nolint:gosec // G404: Chaos testing uses weak RNG intentionally
				}
				time.Sleep(latency)
			}

			// Inject error
			if c.opts.ErrorRate > 0 && rand.Intn(100) < c.opts.ErrorRate { //nolint:gosec // G404: Chaos testing uses weak RNG intentionally
				return ctx.Text(c.opts.ErrorCode, "chaos: injected error")
			}

			return next(ctx)
		}
	}
}

// PathSelector creates a selector for specific paths.
func PathSelector(paths ...string) func(*mizu.Ctx) bool {
	pathMap := make(map[string]bool)
	for _, p := range paths {
		pathMap[p] = true
	}
	return func(c *mizu.Ctx) bool {
		return pathMap[c.Request().URL.Path]
	}
}

// HeaderSelector creates a selector based on header presence.
func HeaderSelector(header string) func(*mizu.Ctx) bool {
	return func(c *mizu.Ctx) bool {
		return c.Request().Header.Get(header) != ""
	}
}

// MethodSelector creates a selector for specific methods.
func MethodSelector(methods ...string) func(*mizu.Ctx) bool {
	methodMap := make(map[string]bool)
	for _, m := range methods {
		methodMap[m] = true
	}
	return func(c *mizu.Ctx) bool {
		return methodMap[c.Request().Method]
	}
}
