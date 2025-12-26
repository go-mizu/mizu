package middleware

import (
	"github.com/go-mizu/mizu"
)

// SecurityConfig holds configuration for security headers.
type SecurityConfig struct {
	// Dev indicates development mode (relaxed CSP)
	Dev bool

	// FrameOptions controls X-Frame-Options header
	// Default: "DENY"
	FrameOptions string

	// ContentTypeNoSniff enables X-Content-Type-Options: nosniff
	// Default: true
	ContentTypeNoSniff bool

	// XSSProtection enables legacy X-XSS-Protection header
	// Default: true
	XSSProtection bool

	// ReferrerPolicy sets the Referrer-Policy header
	// Default: "strict-origin-when-cross-origin"
	ReferrerPolicy string

	// CSP sets the Content-Security-Policy header
	// Default: strict policy for production
	CSP string

	// HSTS sets Strict-Transport-Security header
	// Only set in production with TLS
	HSTSMaxAge int
}

// DefaultSecurityConfig returns the default security configuration.
func DefaultSecurityConfig(dev bool) SecurityConfig {
	cfg := SecurityConfig{
		Dev:                dev,
		FrameOptions:       "DENY",
		ContentTypeNoSniff: true,
		XSSProtection:      true,
		ReferrerPolicy:     "strict-origin-when-cross-origin",
		HSTSMaxAge:         31536000, // 1 year
	}

	if dev {
		// Relaxed CSP for development
		cfg.CSP = "default-src * 'unsafe-inline' 'unsafe-eval' data: blob:; " +
			"script-src * 'unsafe-inline' 'unsafe-eval'; " +
			"style-src * 'unsafe-inline'"
	} else {
		// Strict CSP for production
		cfg.CSP = "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' 'unsafe-eval' https://cdn.tailwindcss.com https://cdnjs.cloudflare.com; " +
			"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; " +
			"img-src 'self' data: blob: https:; " +
			"font-src 'self' data: https://fonts.gstatic.com; " +
			"connect-src 'self' wss: https:; " +
			"media-src 'self' blob:; " +
			"object-src 'none'; " +
			"frame-ancestors 'none'; " +
			"base-uri 'self'; " +
			"form-action 'self'"
	}

	return cfg
}

// SecurityHeaders returns a middleware that sets security-related HTTP headers.
func SecurityHeaders(cfg SecurityConfig) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			h := c.Writer().Header()

			// Prevent clickjacking
			if cfg.FrameOptions != "" {
				h.Set("X-Frame-Options", cfg.FrameOptions)
			}

			// Prevent MIME type sniffing
			if cfg.ContentTypeNoSniff {
				h.Set("X-Content-Type-Options", "nosniff")
			}

			// XSS protection for legacy browsers
			if cfg.XSSProtection {
				h.Set("X-XSS-Protection", "1; mode=block")
			}

			// Referrer policy
			if cfg.ReferrerPolicy != "" {
				h.Set("Referrer-Policy", cfg.ReferrerPolicy)
			}

			// Content Security Policy
			if cfg.CSP != "" {
				h.Set("Content-Security-Policy", cfg.CSP)
			}

			// Permissions Policy (formerly Feature-Policy)
			// Allow microphone for voice messages
			h.Set("Permissions-Policy",
				"geolocation=(), "+
					"microphone=(self), "+
					"camera=(), "+
					"payment=(), "+
					"usb=(), "+
					"magnetometer=(), "+
					"gyroscope=(), "+
					"accelerometer=()")

			// HSTS - only for HTTPS connections
			if !cfg.Dev && c.Request().TLS != nil && cfg.HSTSMaxAge > 0 {
				h.Set("Strict-Transport-Security",
					"max-age=31536000; includeSubDomains; preload")
			}

			// Prevent caching of sensitive data
			// Apply to API routes
			if isAPIRoute(c.Request().URL.Path) {
				h.Set("Cache-Control", "no-store, no-cache, must-revalidate, private")
				h.Set("Pragma", "no-cache")
				h.Set("Expires", "0")
			}

			return next(c)
		}
	}
}

// isAPIRoute checks if the path is an API route.
func isAPIRoute(path string) bool {
	return len(path) >= 4 && path[:4] == "/api"
}

// CORS returns a middleware that sets CORS headers.
func CORS(allowedOrigins []string, dev bool) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			origin := c.Request().Header.Get("Origin")
			if origin == "" {
				return next(c)
			}

			allowed := false
			for _, o := range allowedOrigins {
				if origin == o {
					allowed = true
					break
				}
			}

			// In dev mode, allow localhost
			if dev && !allowed {
				devOrigins := []string{
					"http://localhost",
					"http://localhost:8080",
					"http://127.0.0.1",
					"http://127.0.0.1:8080",
				}
				for _, o := range devOrigins {
					if origin == o {
						allowed = true
						break
					}
				}
			}

			if allowed {
				h := c.Writer().Header()
				h.Set("Access-Control-Allow-Origin", origin)
				h.Set("Access-Control-Allow-Credentials", "true")
				h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				h.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-CSRF-Token")
				h.Set("Access-Control-Max-Age", "86400")
				h.Set("Vary", "Origin")
			}

			// Handle preflight requests
			if c.Request().Method == "OPTIONS" {
				c.Writer().WriteHeader(204)
				return nil
			}

			return next(c)
		}
	}
}
