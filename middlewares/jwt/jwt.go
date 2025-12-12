// Package jwt provides JWT authentication middleware for Mizu.
package jwt

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-mizu/mizu"
)

type contextKey struct{}

// Options configures the JWT middleware.
type Options struct {
	// Secret is the HMAC secret for HS256.
	Secret []byte

	// TokenLookup specifies where to find the token.
	// Default: "header:Authorization".
	TokenLookup string

	// AuthScheme is the authorization scheme.
	// Default: "Bearer".
	AuthScheme string

	// Claims are required claims to validate.
	Claims map[string]any

	// Issuer is the required issuer.
	Issuer string

	// Audience is the required audience.
	Audience []string

	// ContextKey is unused; use Claims() instead.
	ContextKey string

	// ErrorHandler handles authentication failures.
	ErrorHandler func(c *mizu.Ctx, err error) error
}

// New creates JWT middleware with HMAC signing.
func New(secret []byte) mizu.Middleware {
	return WithOptions(Options{Secret: secret})
}

// WithOptions creates JWT middleware with options.
func WithOptions(opts Options) mizu.Middleware {
	if len(opts.Secret) == 0 {
		panic("jwt: secret is required")
	}
	if opts.TokenLookup == "" {
		opts.TokenLookup = "header:Authorization"
	}
	if opts.AuthScheme == "" {
		opts.AuthScheme = "Bearer"
	}

	parts := strings.SplitN(opts.TokenLookup, ":", 2)
	if len(parts) != 2 {
		panic("jwt: invalid TokenLookup format")
	}
	source, name := parts[0], parts[1]

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			var token string

			switch source {
			case "header":
				auth := c.Request().Header.Get(name)
				if auth == "" {
					return handleError(c, opts, ErrTokenMissing)
				}
				if opts.AuthScheme != "" {
					if !strings.HasPrefix(auth, opts.AuthScheme+" ") {
						return handleError(c, opts, ErrInvalidScheme)
					}
					token = strings.TrimPrefix(auth, opts.AuthScheme+" ")
				} else {
					token = auth
				}
			case "query":
				token = c.Query(name)
			case "cookie":
				if cookie, err := c.Cookie(name); err == nil {
					token = cookie.Value
				}
			}

			if token == "" {
				return handleError(c, opts, ErrTokenMissing)
			}

			claims, err := validateToken(token, opts.Secret)
			if err != nil {
				return handleError(c, opts, err)
			}

			// Validate standard claims
			if err := validateClaims(claims, opts); err != nil {
				return handleError(c, opts, err)
			}

			// Store claims in context
			ctx := context.WithValue(c.Context(), contextKey{}, claims)
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
	if err == ErrTokenMissing {
		return c.Text(http.StatusUnauthorized, err.Error())
	}
	return c.Text(http.StatusForbidden, err.Error())
}

// Claims extracts claims from context.
func GetClaims(c *mizu.Ctx) map[string]any {
	if claims, ok := c.Context().Value(contextKey{}).(map[string]any); ok {
		return claims
	}
	return nil
}

// Subject extracts subject claim from context.
func Subject(c *mizu.Ctx) string {
	claims := GetClaims(c)
	if claims == nil {
		return ""
	}
	if sub, ok := claims["sub"].(string); ok {
		return sub
	}
	return ""
}

func validateToken(token string, secret []byte) (map[string]any, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrTokenMalformed
	}

	// Verify signature (HS256)
	message := parts[0] + "." + parts[1]
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, ErrTokenMalformed
	}

	h := hmac.New(sha256.New, secret)
	h.Write([]byte(message))
	expected := h.Sum(nil)

	if !hmac.Equal(signature, expected) {
		return nil, ErrTokenInvalid
	}

	// Decode payload
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrTokenMalformed
	}

	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, ErrTokenMalformed
	}

	return claims, nil
}

func validateClaims(claims map[string]any, opts Options) error {
	now := time.Now().Unix()

	// Check exp
	if exp, ok := claims["exp"].(float64); ok {
		if int64(exp) < now {
			return ErrTokenExpired
		}
	}

	// Check nbf
	if nbf, ok := claims["nbf"].(float64); ok {
		if int64(nbf) > now {
			return ErrTokenNotYetValid
		}
	}

	// Check iss
	if opts.Issuer != "" {
		if iss, ok := claims["iss"].(string); !ok || iss != opts.Issuer {
			return ErrInvalidIssuer
		}
	}

	// Check aud
	if len(opts.Audience) > 0 {
		aud, ok := claims["aud"]
		if !ok {
			return ErrInvalidAudience
		}

		var audiences []string
		switch v := aud.(type) {
		case string:
			audiences = []string{v}
		case []any:
			for _, a := range v {
				if s, ok := a.(string); ok {
					audiences = append(audiences, s)
				}
			}
		}

		found := false
		for _, required := range opts.Audience {
			for _, actual := range audiences {
				if required == actual {
					found = true
					break
				}
			}
		}
		if !found {
			return ErrInvalidAudience
		}
	}

	return nil
}

// Error types
var (
	ErrTokenMissing     = errors.New("token missing")
	ErrTokenMalformed   = errors.New("token malformed")
	ErrTokenInvalid     = errors.New("token invalid")
	ErrTokenExpired     = errors.New("token expired")
	ErrTokenNotYetValid = errors.New("token not yet valid")
	ErrInvalidScheme    = errors.New("invalid auth scheme")
	ErrInvalidIssuer    = errors.New("invalid issuer")
	ErrInvalidAudience  = errors.New("invalid audience")
)
