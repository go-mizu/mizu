// Package header provides request/response header manipulation middleware for Mizu.
package header

import (
	"github.com/go-mizu/mizu"
)

// Options configures the header middleware.
type Options struct {
	// Request headers to set.
	Request map[string]string

	// Response headers to set.
	Response map[string]string

	// RequestRemove is a list of request headers to remove.
	RequestRemove []string

	// ResponseRemove is a list of response headers to remove.
	ResponseRemove []string
}

// New creates header middleware with response headers.
func New(headers map[string]string) mizu.Middleware {
	return WithOptions(Options{Response: headers})
}

// WithOptions creates header middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			// Set request headers
			for key, value := range opts.Request {
				c.Request().Header.Set(key, value)
			}

			// Remove request headers
			for _, key := range opts.RequestRemove {
				c.Request().Header.Del(key)
			}

			// Set response headers before calling next
			for key, value := range opts.Response {
				c.Writer().Header().Set(key, value)
			}

			err := next(c)

			// Remove response headers after handler
			for _, key := range opts.ResponseRemove {
				c.Writer().Header().Del(key)
			}

			return err
		}
	}
}

// Set creates middleware that sets a single response header.
func Set(key, value string) mizu.Middleware {
	return New(map[string]string{key: value})
}

// SetRequest creates middleware that sets a single request header.
func SetRequest(key, value string) mizu.Middleware {
	return WithOptions(Options{Request: map[string]string{key: value}})
}

// Remove creates middleware that removes response headers.
func Remove(keys ...string) mizu.Middleware {
	return WithOptions(Options{ResponseRemove: keys})
}

// RemoveRequest creates middleware that removes request headers.
func RemoveRequest(keys ...string) mizu.Middleware {
	return WithOptions(Options{RequestRemove: keys})
}

// Common security headers

// XSSProtection adds X-XSS-Protection header.
func XSSProtection() mizu.Middleware {
	return Set("X-XSS-Protection", "1; mode=block")
}

// NoSniff adds X-Content-Type-Options header.
func NoSniff() mizu.Middleware {
	return Set("X-Content-Type-Options", "nosniff")
}

// FrameDeny adds X-Frame-Options: DENY header.
func FrameDeny() mizu.Middleware {
	return Set("X-Frame-Options", "DENY")
}

// FrameSameOrigin adds X-Frame-Options: SAMEORIGIN header.
func FrameSameOrigin() mizu.Middleware {
	return Set("X-Frame-Options", "SAMEORIGIN")
}

// HSTS adds Strict-Transport-Security header.
func HSTS(maxAge int, includeSubdomains, preload bool) mizu.Middleware {
	value := "max-age=" + itoa(maxAge)
	if includeSubdomains {
		value += "; includeSubDomains"
	}
	if preload {
		value += "; preload"
	}
	return Set("Strict-Transport-Security", value)
}

// CSP adds Content-Security-Policy header.
func CSP(policy string) mizu.Middleware {
	return Set("Content-Security-Policy", policy)
}

// ReferrerPolicy adds Referrer-Policy header.
func ReferrerPolicy(policy string) mizu.Middleware {
	return Set("Referrer-Policy", policy)
}

// PermissionsPolicy adds Permissions-Policy header.
func PermissionsPolicy(policy string) mizu.Middleware {
	return Set("Permissions-Policy", policy)
}

// Common content headers

// ContentType adds Content-Type header.
func ContentType(contentType string) mizu.Middleware {
	return Set("Content-Type", contentType)
}

// JSON sets Content-Type to application/json.
func JSON() mizu.Middleware {
	return ContentType("application/json; charset=utf-8")
}

// HTML sets Content-Type to text/html.
func HTML() mizu.Middleware {
	return ContentType("text/html; charset=utf-8")
}

// Text sets Content-Type to text/plain.
func Text() mizu.Middleware {
	return ContentType("text/plain; charset=utf-8")
}

// XML sets Content-Type to application/xml.
func XML() mizu.Middleware {
	return ContentType("application/xml; charset=utf-8")
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
