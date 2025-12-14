package cors2

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/mizu"
)

type Options struct {
	Origin        string
	Methods       string
	Headers       string
	ExposeHeaders string
	Credentials   bool
	MaxAge        time.Duration
}

func New() mizu.Middleware { return WithOptions(Options{}) }

//nolint:cyclop
func WithOptions(opts Options) mizu.Middleware {
	if opts.Origin == "" {
		opts.Origin = "*"
	}
	if opts.Methods == "" {
		opts.Methods = "GET, POST, PUT, DELETE, OPTIONS"
	}
	if opts.Headers == "" {
		opts.Headers = "Content-Type, Authorization"
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			req := c.Request()
			h := c.Header()
			origin := req.Header.Get("Origin")

			// Origin handling
			switch {
			case opts.Origin == "*":
				// If credentials are allowed, we must echo the request origin (cannot be "*").
				if opts.Credentials && origin != "" {
					h.Set("Access-Control-Allow-Origin", origin)
					addVary(h, "Origin")
				} else {
					h.Set("Access-Control-Allow-Origin", "*")
				}

			case origin != "" && matchOrigin(origin, opts.Origin):
				h.Set("Access-Control-Allow-Origin", origin)
				addVary(h, "Origin")
			}

			if opts.Credentials {
				h.Set("Access-Control-Allow-Credentials", "true")
			}

			if opts.ExposeHeaders != "" {
				h.Set("Access-Control-Expose-Headers", opts.ExposeHeaders)
			}

			// Preflight: OPTIONS + Access-Control-Request-Method
			if req.Method == http.MethodOptions &&
				req.Header.Get("Access-Control-Request-Method") != "" {

				// Required for cache correctness
				addVary(h, "Access-Control-Request-Method")
				addVary(h, "Access-Control-Request-Headers")

				h.Set("Access-Control-Allow-Methods", opts.Methods)
				h.Set("Access-Control-Allow-Headers", opts.Headers)

				if opts.MaxAge > 0 {
					h.Set("Access-Control-Max-Age", strconv.Itoa(int(opts.MaxAge.Seconds())))
				}

				return c.NoContent()
			}

			return next(c)
		}
	}
}

func matchOrigin(origin, pattern string) bool {
	if pattern == "*" {
		return true
	}
	return strings.EqualFold(origin, pattern)
}

func AllowOrigin(origin string) mizu.Middleware {
	return WithOptions(Options{Origin: origin})
}

func AllowAll() mizu.Middleware {
	return WithOptions(Options{
		Origin:      "*",
		Methods:     "GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS",
		Headers:     "Origin, Content-Type, Accept, Authorization, X-Requested-With",
		Credentials: false,
		MaxAge:      12 * time.Hour,
	})
}

func AllowCredentials(origin string) mizu.Middleware {
	return WithOptions(Options{
		Origin:      origin,
		Credentials: true,
	})
}

func addVary(h http.Header, value string) {
	if value == "" {
		return
	}
	for _, v := range h.Values("Vary") {
		for _, part := range strings.Split(v, ",") {
			if strings.EqualFold(strings.TrimSpace(part), value) {
				return
			}
		}
	}
	h.Add("Vary", value)
}
