// Package nonce provides CSP nonce generation middleware for Mizu.
package nonce

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/go-mizu/mizu"
)

type contextKey struct{}

// Options configures the nonce middleware.
type Options struct {
	// Length is the nonce byte length before encoding.
	// Default: 16 (produces 22 char base64).
	Length int

	// Header is the CSP header to set.
	// Default: Content-Security-Policy.
	Header string

	// Directives specifies which CSP directives get the nonce.
	// Default: ["script-src", "style-src"].
	Directives []string

	// BasePolicy is the base CSP policy to extend.
	// Default: "".
	BasePolicy string

	// Generator is a custom nonce generator.
	Generator func() (string, error)
}

// New creates nonce middleware with default options.
func New() mizu.Middleware {
	return WithOptions(Options{})
}

// WithOptions creates nonce middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	if opts.Length == 0 {
		opts.Length = 16
	}
	if opts.Header == "" {
		opts.Header = "Content-Security-Policy"
	}
	if len(opts.Directives) == 0 {
		opts.Directives = []string{"script-src", "style-src"}
	}
	if opts.Generator == nil {
		opts.Generator = func() (string, error) {
			return generateNonce(opts.Length)
		}
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			// Generate nonce
			nonce, err := opts.Generator()
			if err != nil {
				return err
			}

			// Store in context
			ctx := context.WithValue(c.Context(), contextKey{}, nonce)
			req := c.Request().WithContext(ctx)
			*c.Request() = *req

			// Build CSP header
			csp := buildCSP(opts, nonce)
			c.Header().Set(opts.Header, csp)

			return next(c)
		}
	}
}

// Get retrieves the nonce from context.
func Get(c *mizu.Ctx) string {
	if nonce, ok := c.Context().Value(contextKey{}).(string); ok {
		return nonce
	}
	return ""
}

// ScriptTag returns a nonce attribute for script tags.
func ScriptTag(c *mizu.Ctx) string {
	nonce := Get(c)
	if nonce == "" {
		return ""
	}
	return fmt.Sprintf(`nonce="%s"`, nonce)
}

// StyleTag returns a nonce attribute for style tags.
func StyleTag(c *mizu.Ctx) string {
	return ScriptTag(c)
}

func generateNonce(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawStdEncoding.EncodeToString(bytes), nil
}

func buildCSP(opts Options, nonce string) string {
	nonceStr := fmt.Sprintf("'nonce-%s'", nonce)

	// Parse base policy if provided
	directives := make(map[string]string)
	if opts.BasePolicy != "" {
		for _, part := range strings.Split(opts.BasePolicy, ";") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			spaceIdx := strings.Index(part, " ")
			if spaceIdx == -1 {
				directives[part] = ""
			} else {
				directives[part[:spaceIdx]] = part[spaceIdx+1:]
			}
		}
	}

	// Add nonce to specified directives
	for _, dir := range opts.Directives {
		if existing, ok := directives[dir]; ok {
			directives[dir] = existing + " " + nonceStr
		} else {
			directives[dir] = "'self' " + nonceStr
		}
	}

	// Build CSP string
	var parts []string
	for dir, value := range directives {
		if value == "" {
			parts = append(parts, dir)
		} else {
			parts = append(parts, dir+" "+value)
		}
	}

	return strings.Join(parts, "; ")
}

// ForScripts creates middleware that adds nonce to script-src only.
func ForScripts() mizu.Middleware {
	return WithOptions(Options{
		Directives: []string{"script-src"},
	})
}

// ForStyles creates middleware that adds nonce to style-src only.
func ForStyles() mizu.Middleware {
	return WithOptions(Options{
		Directives: []string{"style-src"},
	})
}

// WithBasePolicy creates middleware with an existing CSP policy.
func WithBasePolicy(policy string) mizu.Middleware {
	return WithOptions(Options{
		BasePolicy: policy,
	})
}

// ReportOnly creates middleware that uses Content-Security-Policy-Report-Only header.
func ReportOnly() mizu.Middleware {
	return WithOptions(Options{
		Header: "Content-Security-Policy-Report-Only",
	})
}
