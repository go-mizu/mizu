// Package timezone provides timezone detection middleware for Mizu.
package timezone

import (
	"context"
	"net/http"
	"time"

	"github.com/go-mizu/mizu"
)

type contextKey struct{}

// Info contains timezone information.
type Info struct {
	Name     string
	Location *time.Location
	Offset   int // Offset in seconds from UTC
}

// Options configures the timezone middleware.
type Options struct {
	// Header is the header to check for timezone.
	// Default: "X-Timezone".
	Header string

	// Cookie is the cookie name to check.
	// Default: "timezone".
	Cookie string

	// QueryParam is the query parameter to check.
	// Default: "tz".
	QueryParam string

	// Default is the default timezone if not detected.
	// Default: "UTC".
	Default string

	// SetCookie sets a cookie with the detected timezone.
	// Default: false.
	SetCookie bool

	// CookieMaxAge is the cookie max age.
	// Default: 30 days.
	CookieMaxAge int

	// Lookup order: "header,cookie,query" (comma-separated).
	// Default: "header,cookie,query".
	Lookup string
}

// New creates timezone middleware with default options.
func New() mizu.Middleware {
	return WithOptions(Options{})
}

// WithOptions creates timezone middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	if opts.Header == "" {
		opts.Header = "X-Timezone"
	}
	if opts.Cookie == "" {
		opts.Cookie = "timezone"
	}
	if opts.QueryParam == "" {
		opts.QueryParam = "tz"
	}
	if opts.Default == "" {
		opts.Default = "UTC"
	}
	if opts.CookieMaxAge == 0 {
		opts.CookieMaxAge = 30 * 24 * 60 * 60 // 30 days
	}
	if opts.Lookup == "" {
		opts.Lookup = "header,cookie,query"
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			tzName := detectTimezone(c, opts)

			// Load location
			loc, err := time.LoadLocation(tzName)
			if err != nil {
				loc, _ = time.LoadLocation(opts.Default)
				tzName = opts.Default
			}

			// Calculate offset
			_, offset := time.Now().In(loc).Zone()

			info := &Info{
				Name:     tzName,
				Location: loc,
				Offset:   offset,
			}

			// Set cookie if requested
			if opts.SetCookie {
				http.SetCookie(c.Writer(), &http.Cookie{
					Name:     opts.Cookie,
					Value:    tzName,
					Path:     "/",
					MaxAge:   opts.CookieMaxAge,
					HttpOnly: true,
					SameSite: http.SameSiteLaxMode,
				})
			}

			// Store in context
			ctx := context.WithValue(c.Context(), contextKey{}, info)
			req := c.Request().WithContext(ctx)
			*c.Request() = *req

			return next(c)
		}
	}
}

func detectTimezone(c *mizu.Ctx, opts Options) string {
	// Parse lookup order
	lookups := []string{"header", "cookie", "query"}
	if opts.Lookup != "" {
		lookups = splitLookup(opts.Lookup)
	}

	for _, lookup := range lookups {
		var tz string
		switch lookup {
		case "header":
			tz = c.Request().Header.Get(opts.Header)
		case "cookie":
			if cookie, err := c.Request().Cookie(opts.Cookie); err == nil {
				tz = cookie.Value
			}
		case "query":
			tz = c.Request().URL.Query().Get(opts.QueryParam)
		}
		if tz != "" {
			return tz
		}
	}

	return opts.Default
}

func splitLookup(s string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			if i > start {
				result = append(result, s[start:i])
			}
			start = i + 1
		}
	}
	if start < len(s) {
		result = append(result, s[start:])
	}
	return result
}

// Get retrieves timezone info from context.
func Get(c *mizu.Ctx) *Info {
	if info, ok := c.Context().Value(contextKey{}).(*Info); ok {
		return info
	}
	loc, _ := time.LoadLocation("UTC")
	return &Info{Name: "UTC", Location: loc, Offset: 0}
}

// Location returns the time.Location from context.
func Location(c *mizu.Ctx) *time.Location {
	return Get(c).Location
}

// Name returns the timezone name from context.
func Name(c *mizu.Ctx) string {
	return Get(c).Name
}

// Offset returns the UTC offset in seconds.
func Offset(c *mizu.Ctx) int {
	return Get(c).Offset
}

// Now returns the current time in the detected timezone.
func Now(c *mizu.Ctx) time.Time {
	return time.Now().In(Get(c).Location)
}

// FromHeader creates middleware that reads timezone from a specific header.
func FromHeader(header string) mizu.Middleware {
	return WithOptions(Options{
		Header: header,
		Lookup: "header",
	})
}

// FromCookie creates middleware that reads timezone from a cookie.
func FromCookie(name string) mizu.Middleware {
	return WithOptions(Options{
		Cookie: name,
		Lookup: "cookie",
	})
}

// WithDefault creates middleware with a default timezone.
func WithDefault(tz string) mizu.Middleware {
	return WithOptions(Options{Default: tz})
}
