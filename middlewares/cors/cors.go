// Package cors provides Cross-Origin Resource Sharing middleware for Mizu.
package cors

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/mizu"
)

// Options configures the CORS middleware.
type Options struct {
	// AllowOrigins is a list of allowed origins.
	// Use "*" to allow any origin.
	// Default: []string{"*"}.
	AllowOrigins []string

	// AllowMethods is a list of allowed HTTP methods.
	// Default: GET, POST, HEAD.
	AllowMethods []string

	// AllowHeaders is a list of allowed request headers.
	// Default: Origin, Content-Type, Accept.
	AllowHeaders []string

	// ExposeHeaders is a list of headers exposed to the browser.
	ExposeHeaders []string

	// AllowCredentials indicates whether credentials are allowed.
	// When true, AllowOrigins cannot be "*".
	AllowCredentials bool

	// MaxAge indicates how long preflight results can be cached.
	// Default: 0 (no caching).
	MaxAge time.Duration

	// AllowOriginFunc is a custom function to validate origins.
	// If set, AllowOrigins is ignored.
	AllowOriginFunc func(origin string) bool

	// AllowPrivateNetwork enables Private Network Access support.
	AllowPrivateNetwork bool
}

// New creates a CORS middleware with the specified options.
func New(opts Options) mizu.Middleware {
	if len(opts.AllowOrigins) == 0 && opts.AllowOriginFunc == nil {
		opts.AllowOrigins = []string{"*"}
	}
	if len(opts.AllowMethods) == 0 {
		opts.AllowMethods = []string{"GET", "POST", "HEAD"}
	}
	if len(opts.AllowHeaders) == 0 {
		opts.AllowHeaders = []string{"Origin", "Content-Type", "Accept"}
	}

	// Only use wildcard origin when not using AllowOriginFunc
	allowAllOrigins := opts.AllowOriginFunc == nil && len(opts.AllowOrigins) == 1 && opts.AllowOrigins[0] == "*"
	allowMethods := strings.Join(opts.AllowMethods, ", ")
	allowHeaders := strings.Join(opts.AllowHeaders, ", ")
	exposeHeaders := strings.Join(opts.ExposeHeaders, ", ")
	maxAge := ""
	if opts.MaxAge > 0 {
		maxAge = strconv.Itoa(int(opts.MaxAge.Seconds()))
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			origin := c.Request().Header.Get("Origin")
			if origin == "" {
				return next(c)
			}

			// Check if origin is allowed
			allowed := false
			if opts.AllowOriginFunc != nil {
				allowed = opts.AllowOriginFunc(origin)
			} else if allowAllOrigins {
				allowed = true
			} else {
				for _, o := range opts.AllowOrigins {
					if o == origin {
						allowed = true
						break
					}
				}
			}

			if !allowed {
				return next(c)
			}

			h := c.Header()

			// Set origin header
			if allowAllOrigins && !opts.AllowCredentials {
				h.Set("Access-Control-Allow-Origin", "*")
			} else {
				h.Set("Access-Control-Allow-Origin", origin)
				h.Add("Vary", "Origin")
			}

			if opts.AllowCredentials {
				h.Set("Access-Control-Allow-Credentials", "true")
			}

			if exposeHeaders != "" {
				h.Set("Access-Control-Expose-Headers", exposeHeaders)
			}

			// Handle preflight request
			if c.Request().Method == http.MethodOptions {
				h.Set("Access-Control-Allow-Methods", allowMethods)
				h.Set("Access-Control-Allow-Headers", allowHeaders)

				if maxAge != "" {
					h.Set("Access-Control-Max-Age", maxAge)
				}

				if opts.AllowPrivateNetwork {
					if c.Request().Header.Get("Access-Control-Request-Private-Network") == "true" {
						h.Set("Access-Control-Allow-Private-Network", "true")
					}
				}

				return c.NoContent()
			}

			return next(c)
		}
	}
}

// AllowAll creates a permissive CORS middleware (for development).
func AllowAll() mizu.Middleware {
	return New(Options{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"*"},
		ExposeHeaders:    []string{"*"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	})
}

// WithOrigins creates a CORS middleware allowing specific origins.
func WithOrigins(origins ...string) mizu.Middleware {
	return New(Options{
		AllowOrigins: origins,
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"},
	})
}
