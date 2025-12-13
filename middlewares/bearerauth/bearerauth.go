// Package bearerauth provides Bearer token authentication middleware for Mizu.
package bearerauth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/go-mizu/mizu"
)

type contextKey struct{}

// TokenValidator validates a bearer token.
type TokenValidator func(token string) bool

// TokenValidatorWithContext validates a token and returns claims.
type TokenValidatorWithContext func(token string) (claims any, valid bool)

// Options configures the bearer auth middleware.
type Options struct {
	// Validator validates the bearer token.
	Validator TokenValidator

	// ValidatorWithContext validates and returns claims.
	ValidatorWithContext TokenValidatorWithContext

	// Header is the header to read the token from.
	// Default: "Authorization".
	Header string

	// AuthScheme is the authorization scheme.
	// Default: "Bearer".
	AuthScheme string

	// ContextKey is unused; use FromContext instead.
	ContextKey string

	// ErrorHandler handles authentication failures.
	ErrorHandler func(c *mizu.Ctx, err error) error
}

// New creates bearer auth middleware with a token validator.
func New(validator TokenValidator) mizu.Middleware {
	return WithOptions(Options{Validator: validator})
}

// WithHeader creates bearer auth middleware reading from a custom header.
func WithHeader(header string, validator TokenValidator) mizu.Middleware {
	return WithOptions(Options{
		Header:    header,
		Validator: validator,
	})
}

// WithOptions creates bearer auth middleware with the specified options.
func WithOptions(opts Options) mizu.Middleware {
	if opts.Validator == nil && opts.ValidatorWithContext == nil {
		panic("bearerauth: validator is required")
	}
	if opts.Header == "" {
		opts.Header = "Authorization"
	}
	if opts.AuthScheme == "" {
		opts.AuthScheme = "Bearer"
	}

	prefix := opts.AuthScheme + " "

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			auth := c.Request().Header.Get(opts.Header)
			if auth == "" {
				return handleError(c, opts, ErrTokenMissing)
			}

			if !strings.HasPrefix(auth, prefix) {
				return handleError(c, opts, ErrInvalidScheme)
			}

			token := strings.TrimPrefix(auth, prefix)
			if token == "" {
				return handleError(c, opts, ErrTokenMissing)
			}

			// Validate token
			var claims any
			var valid bool

			if opts.ValidatorWithContext != nil {
				claims, valid = opts.ValidatorWithContext(token)
			} else {
				valid = opts.Validator(token)
			}

			if !valid {
				return handleError(c, opts, ErrTokenInvalid)
			}

			// Store token/claims in context
			if claims != nil {
				ctx := context.WithValue(c.Context(), contextKey{}, claims)
				req := c.Request().WithContext(ctx)
				*c.Request() = *req
			} else {
				ctx := context.WithValue(c.Context(), contextKey{}, token)
				req := c.Request().WithContext(ctx)
				*c.Request() = *req
			}

			return next(c)
		}
	}
}

func handleError(c *mizu.Ctx, opts Options, err error) error {
	if opts.ErrorHandler != nil {
		return opts.ErrorHandler(c, err)
	}
	if errors.Is(err, ErrTokenMissing) {
		return c.Text(http.StatusUnauthorized, err.Error())
	}
	return c.Text(http.StatusForbidden, err.Error())
}

// FromContext extracts the token or claims from context.
func FromContext(c *mizu.Ctx) any {
	return c.Context().Value(contextKey{})
}

// Token extracts the token string from context.
func Token(c *mizu.Ctx) string {
	if token, ok := c.Context().Value(contextKey{}).(string); ok {
		return token
	}
	return ""
}

// Claims extracts claims from context (when using ValidatorWithContext).
func Claims[T any](c *mizu.Ctx) (T, bool) {
	claims, ok := c.Context().Value(contextKey{}).(T)
	return claims, ok
}

// Error types
type authError string

func (e authError) Error() string { return string(e) }

const (
	// ErrTokenMissing is returned when the token is not found.
	ErrTokenMissing authError = "token missing"

	// ErrTokenInvalid is returned when the token is invalid.
	ErrTokenInvalid authError = "token invalid"

	// ErrInvalidScheme is returned when the auth scheme is wrong.
	ErrInvalidScheme authError = "invalid auth scheme"
)
