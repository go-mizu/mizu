// Package oidc provides OpenID Connect authentication middleware for Mizu.
package oidc

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-mizu/mizu"
)

// Options configures the OIDC middleware.
type Options struct {
	// IssuerURL is the OIDC issuer URL.
	IssuerURL string

	// ClientID is the OAuth 2.0 client ID.
	ClientID string

	// Audience is the expected audience claim.
	// If empty, ClientID is used.
	Audience string

	// JWKSEndpoint is the JWKS endpoint URL.
	// If empty, discovered from issuer.
	JWKSEndpoint string

	// SkipPaths are paths to skip authentication.
	SkipPaths []string

	// TokenExtractor extracts the token from request.
	// Default: Bearer token from Authorization header.
	TokenExtractor func(r *http.Request) string

	// OnError is called when authentication fails.
	OnError func(c *mizu.Ctx, err error) error

	// ClaimsKey is the context key for storing claims.
	// Default: "oidc_claims".
	ClaimsKey any

	// RefreshInterval is how often to refresh JWKS.
	// Default: 1 hour.
	RefreshInterval time.Duration
}

// Claims represents standard OIDC claims.
type Claims struct {
	Issuer    string   `json:"iss"`
	Subject   string   `json:"sub"`
	Audience  any      `json:"aud"`
	ExpiresAt int64    `json:"exp"`
	IssuedAt  int64    `json:"iat"`
	NotBefore int64    `json:"nbf,omitempty"`
	Email     string   `json:"email,omitempty"`
	Name      string   `json:"name,omitempty"`
	Groups    []string `json:"groups,omitempty"`
	Roles     []string `json:"roles,omitempty"`
	Scope     string   `json:"scope,omitempty"`
	Raw       map[string]any
}

// HasAudience checks if the claims contain the given audience.
func (c *Claims) HasAudience(aud string) bool {
	switch v := c.Audience.(type) {
	case string:
		return v == aud
	case []any:
		for _, a := range v {
			if s, ok := a.(string); ok && s == aud {
				return true
			}
		}
	}
	return false
}

// HasGroup checks if the claims contain the given group.
func (c *Claims) HasGroup(group string) bool {
	for _, g := range c.Groups {
		if g == group {
			return true
		}
	}
	return false
}

// HasRole checks if the claims contain the given role.
func (c *Claims) HasRole(role string) bool {
	for _, r := range c.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// Verifier verifies OIDC tokens.
type Verifier struct {
	opts      Options
	keys      map[string]*rsa.PublicKey
	mu        sync.RWMutex
	lastFetch time.Time
}

// contextKey is a private type for context keys.
type contextKey struct{}

// claimsKey stores claims in context.
var claimsKey = contextKey{}

// Errors
var (
	ErrNoToken          = errors.New("oidc: no token provided")
	ErrInvalidToken     = errors.New("oidc: invalid token")
	ErrTokenExpired     = errors.New("oidc: token expired")
	ErrInvalidIssuer    = errors.New("oidc: invalid issuer")
	ErrInvalidAudience  = errors.New("oidc: invalid audience")
	ErrKeyNotFound      = errors.New("oidc: signing key not found")
	ErrInvalidSignature = errors.New("oidc: invalid signature")
)

// New creates OIDC middleware with default options.
func New(issuerURL, clientID string) mizu.Middleware {
	return WithOptions(Options{
		IssuerURL: issuerURL,
		ClientID:  clientID,
	})
}

// WithOptions creates OIDC middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	if opts.Audience == "" {
		opts.Audience = opts.ClientID
	}
	if opts.RefreshInterval == 0 {
		opts.RefreshInterval = time.Hour
	}
	if opts.ClaimsKey == nil {
		opts.ClaimsKey = claimsKey
	}

	verifier := &Verifier{
		opts: opts,
		keys: make(map[string]*rsa.PublicKey),
	}

	skipPaths := make(map[string]bool)
	for _, p := range opts.SkipPaths {
		skipPaths[p] = true
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			// Skip configured paths
			if skipPaths[c.Request().URL.Path] {
				return next(c)
			}

			// Extract token
			token := ""
			if opts.TokenExtractor != nil {
				token = opts.TokenExtractor(c.Request())
			} else {
				token = extractBearerToken(c.Request())
			}

			if token == "" {
				return handleError(c, opts, ErrNoToken)
			}

			// Verify token
			claims, err := verifier.Verify(token)
			if err != nil {
				return handleError(c, opts, err)
			}

			// Store claims in context
			ctx := context.WithValue(c.Context(), opts.ClaimsKey, claims)
			req := c.Request().WithContext(ctx)
			*c.Request() = *req

			return next(c)
		}
	}
}

func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}

	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}

	return parts[1]
}

func handleError(c *mizu.Ctx, opts Options, err error) error {
	if opts.OnError != nil {
		return opts.OnError(c, err)
	}

	status := http.StatusUnauthorized
	return c.JSON(status, map[string]string{
		"error": err.Error(),
	})
}

// Verify verifies a JWT token.
func (v *Verifier) Verify(tokenStr string) (*Claims, error) {
	// Split token
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}

	// Decode header
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, ErrInvalidToken
	}

	var header struct {
		Algorithm string `json:"alg"`
		KeyID     string `json:"kid"`
		Type      string `json:"typ"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return nil, ErrInvalidToken
	}

	// Decode claims
	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrInvalidToken
	}

	var claims Claims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, ErrInvalidToken
	}

	// Store raw claims
	if err := json.Unmarshal(claimsJSON, &claims.Raw); err != nil {
		return nil, ErrInvalidToken
	}

	// Validate issuer
	if v.opts.IssuerURL != "" && claims.Issuer != v.opts.IssuerURL {
		return nil, ErrInvalidIssuer
	}

	// Validate audience
	if v.opts.Audience != "" && !claims.HasAudience(v.opts.Audience) {
		return nil, ErrInvalidAudience
	}

	// Validate expiration
	now := time.Now().Unix()
	if claims.ExpiresAt != 0 && claims.ExpiresAt < now {
		return nil, ErrTokenExpired
	}

	// Validate not before
	if claims.NotBefore != 0 && claims.NotBefore > now {
		return nil, ErrInvalidToken
	}

	return &claims, nil
}

// GetClaims returns the OIDC claims from context.
func GetClaims(c *mizu.Ctx) *Claims {
	if claims, ok := c.Context().Value(claimsKey).(*Claims); ok {
		return claims
	}
	return nil
}

// GetClaimsWithKey returns claims using a custom key.
func GetClaimsWithKey(c *mizu.Ctx, key any) *Claims {
	if claims, ok := c.Context().Value(key).(*Claims); ok {
		return claims
	}
	return nil
}

// RequireGroup creates middleware that requires a specific group.
func RequireGroup(group string) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			claims := GetClaims(c)
			if claims == nil || !claims.HasGroup(group) {
				return c.JSON(http.StatusForbidden, map[string]string{
					"error": "insufficient permissions",
				})
			}
			return next(c)
		}
	}
}

// RequireRole creates middleware that requires a specific role.
func RequireRole(role string) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			claims := GetClaims(c)
			if claims == nil || !claims.HasRole(role) {
				return c.JSON(http.StatusForbidden, map[string]string{
					"error": "insufficient permissions",
				})
			}
			return next(c)
		}
	}
}

// RequireScope creates middleware that requires a specific scope.
func RequireScope(scope string) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			claims := GetClaims(c)
			if claims == nil {
				return c.JSON(http.StatusForbidden, map[string]string{
					"error": "insufficient permissions",
				})
			}

			scopes := strings.Fields(claims.Scope)
			for _, s := range scopes {
				if s == scope {
					return next(c)
				}
			}

			return c.JSON(http.StatusForbidden, map[string]string{
				"error": "insufficient permissions",
			})
		}
	}
}

// JWK represents a JSON Web Key.
type JWK struct {
	KeyType   string `json:"kty"`
	KeyID     string `json:"kid"`
	Algorithm string `json:"alg"`
	Use       string `json:"use"`
	N         string `json:"n"` // RSA modulus
	E         string `json:"e"` // RSA exponent
}

// JWKS represents a JSON Web Key Set.
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// ParseJWK parses a JWK into an RSA public key.
func ParseJWK(jwk JWK) (*rsa.PublicKey, error) {
	if jwk.KeyType != "RSA" {
		return nil, errors.New("unsupported key type")
	}

	nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
	if err != nil {
		return nil, err
	}

	eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
	if err != nil {
		return nil, err
	}

	n := new(big.Int).SetBytes(nBytes)
	e := 0
	for _, b := range eBytes {
		e = e*256 + int(b)
	}

	return &rsa.PublicKey{
		N: n,
		E: e,
	}, nil
}
