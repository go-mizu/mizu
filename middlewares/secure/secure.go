// Package secure provides HTTPS enforcement middleware for Mizu.
package secure

import (
	"net/http"
	"strings"

	"github.com/go-mizu/mizu"
)

// Options configures the secure middleware.
type Options struct {
	// SSLRedirect redirects HTTP to HTTPS.
	// Default: true.
	SSLRedirect bool

	// SSLHost is the host to redirect to for HTTPS.
	// Default: "" (same host).
	SSLHost string

	// SSLTemporaryRedirect uses 307 instead of 301.
	// Default: false.
	SSLTemporaryRedirect bool

	// STSSeconds sets Strict-Transport-Security max-age.
	// Default: 0 (disabled).
	STSSeconds int64

	// STSIncludeSubdomains adds includeSubDomains to HSTS.
	// Default: false.
	STSIncludeSubdomains bool

	// STSPreload adds preload to HSTS.
	// Default: false.
	STSPreload bool

	// ForceSTSHeader forces STS header even on HTTP.
	// Default: false.
	ForceSTSHeader bool

	// ContentTypeNosniff adds X-Content-Type-Options: nosniff.
	// Default: true.
	ContentTypeNosniff bool

	// FrameDeny adds X-Frame-Options: DENY.
	// Default: true.
	FrameDeny bool

	// CustomFrameOptions overrides X-Frame-Options.
	// Default: "".
	CustomFrameOptions string

	// XSSProtection adds X-XSS-Protection header.
	// Default: "1; mode=block".
	XSSProtection string

	// ContentSecurityPolicy sets Content-Security-Policy header.
	// Default: "".
	ContentSecurityPolicy string

	// ReferrerPolicy sets Referrer-Policy header.
	// Default: "".
	ReferrerPolicy string

	// IsDevelopment disables all security features.
	// Default: false.
	IsDevelopment bool

	// ProxyHeaders is the list of headers to check for HTTPS.
	// Default: []string{"X-Forwarded-Proto"}.
	ProxyHeaders []string
}

// New creates secure middleware with default options.
func New() mizu.Middleware {
	return WithOptions(Options{
		SSLRedirect:        true,
		ContentTypeNosniff: true,
		FrameDeny:          true,
		XSSProtection:      "1; mode=block",
	})
}

// WithOptions creates secure middleware with custom options.
//
//nolint:cyclop // Security middleware requires multiple header and policy checks
func WithOptions(opts Options) mizu.Middleware {
	if len(opts.ProxyHeaders) == 0 {
		opts.ProxyHeaders = []string{"X-Forwarded-Proto"}
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			if opts.IsDevelopment {
				return next(c)
			}

			r := c.Request()
			isSSL := r.TLS != nil

			// Check proxy headers for HTTPS
			if !isSSL {
				for _, header := range opts.ProxyHeaders {
					if strings.EqualFold(r.Header.Get(header), "https") {
						isSSL = true
						break
					}
				}
			}

			// SSL redirect
			if opts.SSLRedirect && !isSSL {
				host := opts.SSLHost
				if host == "" {
					host = r.Host
				}

				url := "https://" + host + r.URL.RequestURI()
				status := http.StatusMovedPermanently
				if opts.SSLTemporaryRedirect {
					status = http.StatusTemporaryRedirect
				}
				return c.Redirect(status, url)
			}

			// Set security headers
			w := c.Writer()

			// HSTS
			if opts.STSSeconds > 0 && (isSSL || opts.ForceSTSHeader) {
				sts := "max-age=" + itoa(opts.STSSeconds)
				if opts.STSIncludeSubdomains {
					sts += "; includeSubDomains"
				}
				if opts.STSPreload {
					sts += "; preload"
				}
				w.Header().Set("Strict-Transport-Security", sts)
			}

			// X-Content-Type-Options
			if opts.ContentTypeNosniff {
				w.Header().Set("X-Content-Type-Options", "nosniff")
			}

			// X-Frame-Options
			if opts.CustomFrameOptions != "" {
				w.Header().Set("X-Frame-Options", opts.CustomFrameOptions)
			} else if opts.FrameDeny {
				w.Header().Set("X-Frame-Options", "DENY")
			}

			// X-XSS-Protection
			if opts.XSSProtection != "" {
				w.Header().Set("X-XSS-Protection", opts.XSSProtection)
			}

			// Content-Security-Policy
			if opts.ContentSecurityPolicy != "" {
				w.Header().Set("Content-Security-Policy", opts.ContentSecurityPolicy)
			}

			// Referrer-Policy
			if opts.ReferrerPolicy != "" {
				w.Header().Set("Referrer-Policy", opts.ReferrerPolicy)
			}

			return next(c)
		}
	}
}

func itoa(i int64) string {
	if i == 0 {
		return "0"
	}
	var b [20]byte
	pos := len(b)
	neg := i < 0
	if neg {
		i = -i
	}
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		b[pos] = '-'
	}
	return string(b[pos:])
}
