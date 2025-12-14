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
	// Use "*" to allow any origin (credentials must be false in that case).
	// You can also use wildcard patterns like "https://*.example.com".
	// Default: []string{"*"} (when AllowOriginFunc is nil).
	AllowOrigins []string

	// AllowMethods is a list of allowed HTTP methods.
	// Default: GET, POST, HEAD.
	AllowMethods []string

	// AllowHeaders is a list of allowed request headers.
	// Use "*" to reflect the browser preflight request headers.
	// Default: Origin, Content-Type, Accept.
	AllowHeaders []string

	// ExposeHeaders is a list of headers exposed to the browser.
	ExposeHeaders []string

	// AllowCredentials indicates whether credentials are allowed.
	// When true, Access-Control-Allow-Origin cannot be "*".
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
//
//nolint:cyclop // CORS handling requires multiple header and origin checks
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

	allowAllOrigins := opts.AllowOriginFunc == nil && len(opts.AllowOrigins) == 1 && strings.TrimSpace(opts.AllowOrigins[0]) == "*"

	// If credentials are enabled, we must not use "*" as Allow-Origin.
	if opts.AllowCredentials && allowAllOrigins {
		allowAllOrigins = false
	}

	allowMethods := strings.Join(opts.AllowMethods, ", ")
	allowHeaders := strings.Join(opts.AllowHeaders, ", ")
	exposeHeaders := strings.Join(opts.ExposeHeaders, ", ")

	maxAge := ""
	if opts.MaxAge > 0 {
		maxAge = strconv.Itoa(int(opts.MaxAge.Seconds()))
	}

	reflectRequestHeaders := len(opts.AllowHeaders) == 1 && opts.AllowHeaders[0] == "*"

	originAllowed := func(origin string) bool {
		if origin == "" {
			return false
		}
		if opts.AllowOriginFunc != nil {
			return opts.AllowOriginFunc(origin)
		}
		if allowAllOrigins {
			return true
		}
		for _, o := range opts.AllowOrigins {
			o = strings.TrimSpace(o)
			if o == "" {
				continue
			}
			if o == origin {
				return true
			}
			// Support simple wildcard patterns like "https://*.example.com"
			if strings.Contains(o, "*") && wildcardMatchOrigin(o, origin) {
				return true
			}
		}
		return false
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			req := c.Request()
			origin := req.Header.Get("Origin")
			if origin == "" {
				return next(c)
			}

			if !originAllowed(origin) {
				return next(c)
			}

			h := c.Header()

			// Access-Control-Allow-Origin and Vary: Origin when echoing.
			if allowAllOrigins && !opts.AllowCredentials {
				h.Set("Access-Control-Allow-Origin", "*")
			} else {
				h.Set("Access-Control-Allow-Origin", origin)
				addVary(h, "Origin")
			}

			if opts.AllowCredentials {
				h.Set("Access-Control-Allow-Credentials", "true")
			}
			if exposeHeaders != "" {
				h.Set("Access-Control-Expose-Headers", exposeHeaders)
			}

			// Preflight is OPTIONS + Access-Control-Request-Method.
			isPreflight := req.Method == http.MethodOptions && req.Header.Get("Access-Control-Request-Method") != ""
			if !isPreflight {
				return next(c)
			}

			// Required Vary keys for cache correctness.
			addVary(h, "Access-Control-Request-Method")
			addVary(h, "Access-Control-Request-Headers")

			h.Set("Access-Control-Allow-Methods", allowMethods)

			if reflectRequestHeaders {
				if rh := req.Header.Get("Access-Control-Request-Headers"); rh != "" {
					h.Set("Access-Control-Allow-Headers", rh)
				}
			} else if allowHeaders != "" {
				h.Set("Access-Control-Allow-Headers", allowHeaders)
			}

			if maxAge != "" {
				h.Set("Access-Control-Max-Age", maxAge)
			}

			if opts.AllowPrivateNetwork && req.Header.Get("Access-Control-Request-Private-Network") == "true" {
				h.Set("Access-Control-Allow-Private-Network", "true")
				addVary(h, "Access-Control-Request-Private-Network")
			}

			return c.NoContent()
		}
	}
}

// AllowAll creates a permissive CORS middleware (for development).
func AllowAll() mizu.Middleware {
	return New(Options{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodHead, http.MethodOptions},
		AllowHeaders:     []string{"*"}, // reflect request headers for preflight
		ExposeHeaders:    nil,           // "*" is not reliably supported here
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	})
}

// WithOrigins creates a CORS middleware allowing specific origins.
func WithOrigins(origins ...string) mizu.Middleware {
	return New(Options{
		AllowOrigins: origins,
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodHead, http.MethodOptions},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"},
	})
}

func addVary(h http.Header, v string) {
	if v == "" {
		return
	}
	existing := h.Values("Vary")
	for _, e := range existing {
		for _, part := range strings.Split(e, ",") {
			if strings.EqualFold(strings.TrimSpace(part), v) {
				return
			}
		}
	}
	h.Add("Vary", v)
}

// wildcardMatchOrigin matches patterns like "https://*.example.com" against an origin.
func wildcardMatchOrigin(pattern, origin string) bool {
	p := strings.TrimSpace(pattern)
	o := strings.TrimSpace(origin)
	if p == "" || o == "" {
		return false
	}
	if p == "*" {
		return true
	}
	i := strings.IndexByte(p, '*')
	if i < 0 {
		return p == o
	}
	pre := p[:i]
	suf := p[i+1:]
	if !strings.HasPrefix(o, pre) {
		return false
	}
	if !strings.HasSuffix(o, suf) {
		return false
	}
	mid := o[len(pre) : len(o)-len(suf)]
	return mid != ""
}
