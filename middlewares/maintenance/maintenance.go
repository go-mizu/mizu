// Package maintenance provides maintenance mode middleware for Mizu.
package maintenance

import (
	"net/http"
	"sync/atomic"
	"time"

	"github.com/go-mizu/mizu"
)

// Options configures the maintenance middleware.
type Options struct {
	// Enabled enables maintenance mode.
	// Default: false.
	Enabled bool

	// Message is the maintenance message.
	// Default: "Service is under maintenance".
	Message string

	// RetryAfter is the Retry-After header value in seconds.
	// Default: 3600.
	RetryAfter int

	// StatusCode is the response status code.
	// Default: 503.
	StatusCode int

	// Handler is a custom handler for maintenance mode.
	Handler mizu.Handler

	// Whitelist is a list of allowed IPs during maintenance.
	Whitelist []string

	// WhitelistPaths is a list of paths that bypass maintenance.
	WhitelistPaths []string

	// Check is a function that determines if maintenance is enabled.
	// Takes precedence over Enabled.
	Check func() bool
}

// Mode represents a controllable maintenance mode.
type Mode struct {
	enabled int32
	opts    Options
}

// New creates maintenance middleware with options.
func New(enabled bool) mizu.Middleware {
	return WithOptions(Options{Enabled: enabled})
}

// WithOptions creates maintenance middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	if opts.Message == "" {
		opts.Message = "Service is under maintenance"
	}
	if opts.RetryAfter == 0 {
		opts.RetryAfter = 3600
	}
	if opts.StatusCode == 0 {
		opts.StatusCode = http.StatusServiceUnavailable
	}

	whitelistMap := make(map[string]bool)
	for _, ip := range opts.Whitelist {
		whitelistMap[ip] = true
	}

	pathMap := make(map[string]bool)
	for _, path := range opts.WhitelistPaths {
		pathMap[path] = true
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			// Check if maintenance is enabled
			var enabled bool
			if opts.Check != nil {
				enabled = opts.Check()
			} else {
				enabled = opts.Enabled
			}

			if !enabled {
				return next(c)
			}

			// Check whitelist IPs
			if len(whitelistMap) > 0 {
				clientIP := getClientIP(c)
				if whitelistMap[clientIP] {
					return next(c)
				}
			}

			// Check whitelist paths
			if len(pathMap) > 0 {
				if pathMap[c.Request().URL.Path] {
					return next(c)
				}
			}

			// Return maintenance response
			if opts.Handler != nil {
				return opts.Handler(c)
			}

			c.Writer().Header().Set("Retry-After", itoa(opts.RetryAfter))
			return c.Text(opts.StatusCode, opts.Message)
		}
	}
}

// NewMode creates a controllable maintenance mode.
func NewMode(opts Options) *Mode {
	var enabled int32
	if opts.Enabled {
		enabled = 1
	}
	return &Mode{
		enabled: enabled,
		opts:    opts,
	}
}

// Enable enables maintenance mode.
func (m *Mode) Enable() {
	atomic.StoreInt32(&m.enabled, 1)
}

// Disable disables maintenance mode.
func (m *Mode) Disable() {
	atomic.StoreInt32(&m.enabled, 0)
}

// IsEnabled returns whether maintenance mode is enabled.
func (m *Mode) IsEnabled() bool {
	return atomic.LoadInt32(&m.enabled) == 1
}

// Toggle toggles maintenance mode.
func (m *Mode) Toggle() {
	for {
		old := atomic.LoadInt32(&m.enabled)
		var new int32
		if old == 0 {
			new = 1
		}
		if atomic.CompareAndSwapInt32(&m.enabled, old, new) {
			break
		}
	}
}

// Middleware returns the middleware for this mode.
func (m *Mode) Middleware() mizu.Middleware {
	opts := m.opts
	opts.Check = m.IsEnabled
	return WithOptions(opts)
}

// ScheduledMaintenance schedules maintenance for a specific time period.
func ScheduledMaintenance(start, end time.Time) mizu.Middleware {
	return WithOptions(Options{
		Check: func() bool {
			now := time.Now()
			return now.After(start) && now.Before(end)
		},
	})
}

func getClientIP(c *mizu.Ctx) string {
	// Check X-Forwarded-For
	if xff := c.Request().Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	// Check X-Real-IP
	if xri := c.Request().Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Use RemoteAddr
	return c.Request().RemoteAddr
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [20]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(b[pos:])
}
