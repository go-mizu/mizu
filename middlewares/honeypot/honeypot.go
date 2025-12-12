// Package honeypot provides honeypot middleware for Mizu.
package honeypot

import (
	"net/http"
	"sync"
	"time"

	"github.com/go-mizu/mizu"
)

// Options configures the honeypot middleware.
type Options struct {
	// Paths is the list of honeypot paths to monitor.
	// Default: common paths.
	Paths []string

	// BlockDuration is how long to block detected attackers.
	// Default: 1h.
	BlockDuration time.Duration

	// Response is the honeypot response.
	// Default: 404.
	Response func(c *mizu.Ctx) error

	// OnTrap is called when a honeypot is triggered.
	OnTrap func(ip string, path string)
}

// Default honeypot paths
var defaultPaths = []string{
	"/admin", "/administrator", "/wp-admin", "/wp-login.php",
	"/phpmyadmin", "/pma", "/mysql", "/sql",
	"/.env", "/.git", "/.svn", "/config.php",
	"/backup", "/backup.sql", "/dump.sql",
	"/shell", "/cmd", "/exec", "/eval",
	"/xmlrpc.php", "/wp-config.php",
	"/server-status", "/server-info",
}

// New creates honeypot middleware with default options.
func New() mizu.Middleware {
	return WithOptions(Options{})
}

// WithOptions creates honeypot middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	if len(opts.Paths) == 0 {
		opts.Paths = defaultPaths
	}
	if opts.BlockDuration == 0 {
		opts.BlockDuration = time.Hour
	}
	if opts.Response == nil {
		opts.Response = func(c *mizu.Ctx) error {
			return c.Text(http.StatusNotFound, "Not Found")
		}
	}

	pathMap := make(map[string]bool)
	for _, path := range opts.Paths {
		pathMap[path] = true
	}

	blocked := &blockList{
		ips:      make(map[string]time.Time),
		duration: opts.BlockDuration,
	}
	go blocked.cleanup()

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			ip := getClientIP(c)

			// Check if IP is blocked
			if blocked.isBlocked(ip) {
				return c.Text(http.StatusForbidden, "Access denied")
			}

			// Check if request path is a honeypot
			path := c.Request().URL.Path
			if pathMap[path] {
				// Block IP
				blocked.block(ip)

				// Call trap handler
				if opts.OnTrap != nil {
					opts.OnTrap(ip, path)
				}

				return opts.Response(c)
			}

			return next(c)
		}
	}
}

type blockList struct {
	mu       sync.RWMutex
	ips      map[string]time.Time
	duration time.Duration
}

func (b *blockList) block(ip string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.ips[ip] = time.Now().Add(b.duration)
}

func (b *blockList) isBlocked(ip string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if expiry, ok := b.ips[ip]; ok {
		return time.Now().Before(expiry)
	}
	return false
}

func (b *blockList) cleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	for range ticker.C {
		b.mu.Lock()
		now := time.Now()
		for ip, expiry := range b.ips {
			if now.After(expiry) {
				delete(b.ips, ip)
			}
		}
		b.mu.Unlock()
	}
}

func getClientIP(c *mizu.Ctx) string {
	if xff := c.Request().Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	if xri := c.Request().Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	return c.Request().RemoteAddr
}

// Paths creates honeypot middleware with specific paths.
func Paths(paths ...string) mizu.Middleware {
	return WithOptions(Options{Paths: paths})
}

// AdminPaths creates honeypot for admin paths.
func AdminPaths() mizu.Middleware {
	return Paths(
		"/admin", "/administrator", "/wp-admin", "/admin.php",
		"/login", "/admin/login", "/adminpanel",
	)
}

// ConfigPaths creates honeypot for config paths.
func ConfigPaths() mizu.Middleware {
	return Paths(
		"/.env", "/.git", "/.svn", "/.htaccess",
		"/config.php", "/config.yml", "/settings.php",
		"/wp-config.php", "/configuration.php",
	)
}

// DatabasePaths creates honeypot for database paths.
func DatabasePaths() mizu.Middleware {
	return Paths(
		"/phpmyadmin", "/pma", "/mysql", "/sql",
		"/backup.sql", "/dump.sql", "/database.sql",
		"/db.sql", "/data.sql",
	)
}

// Form creates honeypot middleware for form fields.
func Form(field string) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			// Check if honeypot field is filled
			if c.Request().FormValue(field) != "" {
				// This is likely a bot
				return c.Text(http.StatusBadRequest, "invalid request")
			}
			return next(c)
		}
	}
}
