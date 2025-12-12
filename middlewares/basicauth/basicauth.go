// Package basicauth provides HTTP Basic Authentication middleware for Mizu.
package basicauth

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/go-mizu/mizu"
)

// ValidatorFunc validates username and password.
type ValidatorFunc func(username, password string) bool

// Options configures the basic auth middleware.
type Options struct {
	// Realm is the authentication realm.
	// Default: "Restricted".
	Realm string

	// Validator is the credential validation function.
	Validator ValidatorFunc

	// ErrorHandler handles authentication failures.
	ErrorHandler func(c *mizu.Ctx) error
}

// New creates basic auth middleware with a static credentials map.
func New(credentials map[string]string) mizu.Middleware {
	return WithOptions(Options{
		Validator: func(user, pass string) bool {
			expected, ok := credentials[user]
			if !ok {
				return false
			}
			return secureCompare(pass, expected)
		},
	})
}

// WithValidator creates basic auth middleware with a custom validator.
func WithValidator(fn ValidatorFunc) mizu.Middleware {
	return WithOptions(Options{Validator: fn})
}

// WithRealm creates basic auth middleware with a custom realm.
func WithRealm(realm string, credentials map[string]string) mizu.Middleware {
	return WithOptions(Options{
		Realm: realm,
		Validator: func(user, pass string) bool {
			expected, ok := credentials[user]
			if !ok {
				return false
			}
			return secureCompare(pass, expected)
		},
	})
}

// WithOptions creates basic auth middleware with the specified options.
func WithOptions(opts Options) mizu.Middleware {
	if opts.Validator == nil {
		panic("basicauth: validator is required")
	}
	if opts.Realm == "" {
		opts.Realm = "Restricted"
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			auth := c.Request().Header.Get("Authorization")
			if auth == "" {
				return unauthorized(c, opts)
			}

			if !strings.HasPrefix(auth, "Basic ") {
				return unauthorized(c, opts)
			}

			decoded, err := base64.StdEncoding.DecodeString(auth[6:])
			if err != nil {
				return unauthorized(c, opts)
			}

			parts := strings.SplitN(string(decoded), ":", 2)
			if len(parts) != 2 {
				return unauthorized(c, opts)
			}

			username, password := parts[0], parts[1]
			if !opts.Validator(username, password) {
				return unauthorized(c, opts)
			}

			return next(c)
		}
	}
}

func unauthorized(c *mizu.Ctx, opts Options) error {
	c.Header().Set("WWW-Authenticate", `Basic realm="`+opts.Realm+`"`)
	if opts.ErrorHandler != nil {
		return opts.ErrorHandler(c)
	}
	return c.Text(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
}

// secureCompare performs constant-time string comparison.
func secureCompare(a, b string) bool {
	// Hash both strings to ensure constant time comparison
	// even for strings of different lengths
	aHash := sha256.Sum256([]byte(a))
	bHash := sha256.Sum256([]byte(b))
	return subtle.ConstantTimeCompare(aHash[:], bHash[:]) == 1
}

// Accounts is a map of username to password for convenience.
type Accounts map[string]string
