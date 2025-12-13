// Package keyauth provides API key authentication middleware for Mizu.
package keyauth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/go-mizu/mizu"
)

type contextKey struct{}

// KeyValidator validates an API key.
type KeyValidator func(key string) (valid bool, err error)

// Options configures the API key middleware.
type Options struct {
	// Validator validates the API key.
	Validator KeyValidator

	// KeyLookup specifies where to find the key.
	// Format: "source:name".
	// Sources: header, query, cookie.
	// Default: "header:X-API-Key".
	KeyLookup string

	// AuthScheme is the scheme prefix for header lookup.
	// Default: "" (no scheme).
	AuthScheme string

	// ContextKey is unused; use FromContext instead.
	ContextKey string

	// ErrorHandler handles authentication failures.
	ErrorHandler func(c *mizu.Ctx, err error) error
}

// New creates API key middleware with validator.
func New(validator KeyValidator) mizu.Middleware {
	return WithOptions(Options{Validator: validator})
}

// WithOptions creates API key middleware with options.
//
//nolint:cyclop // API key validation requires multiple source and validation checks
func WithOptions(opts Options) mizu.Middleware {
	if opts.Validator == nil {
		panic("keyauth: validator is required")
	}
	if opts.KeyLookup == "" {
		opts.KeyLookup = "header:X-API-Key"
	}

	// Parse key lookup
	parts := strings.SplitN(opts.KeyLookup, ":", 2)
	if len(parts) != 2 {
		panic("keyauth: invalid KeyLookup format")
	}
	source, name := parts[0], parts[1]

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			var key string

			switch source {
			case "header":
				key = c.Request().Header.Get(name)
				if opts.AuthScheme != "" && strings.HasPrefix(key, opts.AuthScheme+" ") {
					key = strings.TrimPrefix(key, opts.AuthScheme+" ")
				}
			case "query":
				key = c.Query(name)
			case "cookie":
				if cookie, err := c.Cookie(name); err == nil {
					key = cookie.Value
				}
			}

			if key == "" {
				return handleError(c, opts, ErrKeyMissing)
			}

			valid, err := opts.Validator(key)
			if err != nil {
				return handleError(c, opts, err)
			}
			if !valid {
				return handleError(c, opts, ErrKeyInvalid)
			}

			// Store key in context
			ctx := context.WithValue(c.Context(), contextKey{}, key)
			req := c.Request().WithContext(ctx)
			*c.Request() = *req

			return next(c)
		}
	}
}

func handleError(c *mizu.Ctx, opts Options, err error) error {
	if opts.ErrorHandler != nil {
		return opts.ErrorHandler(c, err)
	}
	if errors.Is(err, ErrKeyMissing) {
		return c.Text(http.StatusUnauthorized, err.Error())
	}
	return c.Text(http.StatusForbidden, err.Error())
}

// FromContext extracts the API key from context.
func FromContext(c *mizu.Ctx) string {
	if key, ok := c.Context().Value(contextKey{}).(string); ok {
		return key
	}
	return ""
}

// Get is an alias for FromContext.
func Get(c *mizu.Ctx) string {
	return FromContext(c)
}

// Error types
type keyError string

func (e keyError) Error() string { return string(e) }

const (
	// ErrKeyMissing is returned when the API key is not found.
	ErrKeyMissing keyError = "API key missing"

	// ErrKeyInvalid is returned when the API key is invalid.
	ErrKeyInvalid keyError = "API key invalid"
)

// ValidateKeys creates a validator for a static list of keys.
func ValidateKeys(keys ...string) KeyValidator {
	keySet := make(map[string]bool)
	for _, k := range keys {
		keySet[k] = true
	}
	return func(key string) (bool, error) {
		return keySet[key], nil
	}
}
