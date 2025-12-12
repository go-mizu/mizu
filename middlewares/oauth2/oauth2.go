// Package oauth2 provides OAuth 2.0 resource server middleware for Mizu.
package oauth2

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-mizu/mizu"
)

type contextKey struct{}

// Token represents an OAuth 2.0 access token.
type Token struct {
	Value     string
	Type      string
	Scope     []string
	ExpiresAt time.Time
	Subject   string
	Issuer    string
	Claims    map[string]any
}

// Options configures the OAuth 2.0 middleware.
type Options struct {
	// Validator validates access tokens.
	Validator TokenValidator

	// IntrospectionURL is the token introspection endpoint.
	IntrospectionURL string

	// ClientID for introspection.
	ClientID string

	// ClientSecret for introspection.
	ClientSecret string

	// RequiredScopes are scopes required for access.
	RequiredScopes []string

	// TokenLookup specifies where to find the token.
	// Default: "header:Authorization".
	TokenLookup string

	// ErrorHandler handles validation failures.
	ErrorHandler func(c *mizu.Ctx, err error) error
}

// TokenValidator validates an access token.
type TokenValidator func(token string) (*Token, error)

// New creates OAuth 2.0 middleware with a validator.
func New(validator TokenValidator) mizu.Middleware {
	return WithOptions(Options{Validator: validator})
}

// WithOptions creates OAuth 2.0 middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	if opts.TokenLookup == "" {
		opts.TokenLookup = "header:Authorization"
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			// Extract token
			token := extractToken(c, opts.TokenLookup)
			if token == "" {
				return handleError(c, opts, ErrMissingToken)
			}

			// Validate token
			var tokenInfo *Token
			var err error

			if opts.Validator != nil {
				tokenInfo, err = opts.Validator(token)
			} else if opts.IntrospectionURL != "" {
				tokenInfo, err = introspectToken(opts, token)
			} else {
				return handleError(c, opts, ErrNoValidator)
			}

			if err != nil {
				return handleError(c, opts, err)
			}

			// Check expiration
			if !tokenInfo.ExpiresAt.IsZero() && time.Now().After(tokenInfo.ExpiresAt) {
				return handleError(c, opts, ErrExpiredToken)
			}

			// Check required scopes
			if len(opts.RequiredScopes) > 0 {
				if !hasScopes(tokenInfo.Scope, opts.RequiredScopes) {
					return handleError(c, opts, ErrInsufficientScope)
				}
			}

			// Store token in context
			ctx := context.WithValue(c.Context(), contextKey{}, tokenInfo)
			req := c.Request().WithContext(ctx)
			*c.Request() = *req

			return next(c)
		}
	}
}

// Error types
type oauthError string

func (e oauthError) Error() string { return string(e) }

const (
	ErrMissingToken      oauthError = "missing access token"
	ErrInvalidToken      oauthError = "invalid access token"
	ErrExpiredToken      oauthError = "access token expired"
	ErrInsufficientScope oauthError = "insufficient scope"
	ErrNoValidator       oauthError = "no token validator configured"
)

func extractToken(c *mizu.Ctx, lookup string) string {
	parts := strings.SplitN(lookup, ":", 2)
	if len(parts) != 2 {
		return ""
	}

	source, key := parts[0], parts[1]

	switch source {
	case "header":
		auth := c.Request().Header.Get(key)
		if strings.HasPrefix(auth, "Bearer ") {
			return strings.TrimPrefix(auth, "Bearer ")
		}
		return auth
	case "query":
		return c.Request().URL.Query().Get(key)
	case "form":
		return c.Request().FormValue(key)
	}

	return ""
}

func introspectToken(opts Options, token string) (*Token, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest(http.MethodPost, opts.IntrospectionURL, strings.NewReader("token="+token))
	if err != nil {
		return nil, ErrInvalidToken
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if opts.ClientID != "" {
		req.SetBasicAuth(opts.ClientID, opts.ClientSecret)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, ErrInvalidToken
	}
	defer func() { _ = resp.Body.Close() }()

	var result struct {
		Active    bool     `json:"active"`
		Scope     string   `json:"scope"`
		Subject   string   `json:"sub"`
		Issuer    string   `json:"iss"`
		ExpiresAt int64    `json:"exp"`
		ClientID  string   `json:"client_id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, ErrInvalidToken
	}

	if !result.Active {
		return nil, ErrInvalidToken
	}

	return &Token{
		Value:     token,
		Type:      "Bearer",
		Scope:     strings.Split(result.Scope, " "),
		Subject:   result.Subject,
		Issuer:    result.Issuer,
		ExpiresAt: time.Unix(result.ExpiresAt, 0),
	}, nil
}

func hasScopes(have, need []string) bool {
	haveSet := make(map[string]bool)
	for _, s := range have {
		haveSet[s] = true
	}
	for _, s := range need {
		if !haveSet[s] {
			return false
		}
	}
	return true
}

func handleError(c *mizu.Ctx, opts Options, err error) error {
	if opts.ErrorHandler != nil {
		return opts.ErrorHandler(c, err)
	}

	status := http.StatusUnauthorized
	if err == ErrInsufficientScope {
		status = http.StatusForbidden
	}

	c.Header().Set("WWW-Authenticate", "Bearer")
	return c.Text(status, err.Error())
}

// Get retrieves the token from context.
func Get(c *mizu.Ctx) *Token {
	if token, ok := c.Context().Value(contextKey{}).(*Token); ok {
		return token
	}
	return nil
}

// Subject returns the token subject.
func Subject(c *mizu.Ctx) string {
	if token := Get(c); token != nil {
		return token.Subject
	}
	return ""
}

// Scopes returns the token scopes.
func Scopes(c *mizu.Ctx) []string {
	if token := Get(c); token != nil {
		return token.Scope
	}
	return nil
}

// HasScope checks if token has a scope.
func HasScope(c *mizu.Ctx, scope string) bool {
	for _, s := range Scopes(c) {
		if s == scope {
			return true
		}
	}
	return false
}

// RequireScopes creates middleware requiring specific scopes.
func RequireScopes(scopes ...string) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			token := Get(c)
			if token == nil {
				return c.Text(http.StatusUnauthorized, "Authentication required")
			}

			if !hasScopes(token.Scope, scopes) {
				return c.Text(http.StatusForbidden, "Insufficient scope")
			}

			return next(c)
		}
	}
}
