package rest

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-mizu/mizu"
)

// AuthConfig configures JWT authentication for the REST server.
type AuthConfig struct {
	// JWTSecret is the secret key used to verify JWT tokens.
	JWTSecret string

	// Optional: Allow anonymous access to public buckets even without a valid JWT.
	// If false, all requests require authentication.
	AllowAnonymousPublic bool
}

// contextKey is used for storing values in context.
type contextKey string

const (
	// ctxKeyClaims stores the JWT claims in the request context.
	ctxKeyClaims contextKey = "jwt_claims"
	// ctxKeyRole stores the user role from JWT claims.
	ctxKeyRole contextKey = "user_role"
	// ctxKeyUserID stores the user ID from JWT claims.
	ctxKeyUserID contextKey = "user_id"
)

// Claims represents JWT claims compatible with Supabase Auth.
// This mirrors the structure in cli/literest/core/jwt.go but is defined
// here to avoid circular dependencies.
type Claims struct {
	Sub  string `json:"sub,omitempty"`  // Subject (user ID)
	Aud  string `json:"aud,omitempty"`  // Audience
	Iss  string `json:"iss,omitempty"`  // Issuer
	Iat  int64  `json:"iat,omitempty"`  // Issued at
	Exp  int64  `json:"exp,omitempty"`  // Expiration
	Role string `json:"role,omitempty"` // User role
}

// AuthMiddleware returns a Mizu middleware that validates JWT tokens.
//
// The middleware:
// - Extracts JWT token from Authorization: Bearer header
// - Validates the token signature and expiration
// - Stores claims in the request context
// - For public bucket operations, allows access without auth if configured
// - Returns 401 Unauthorized for invalid/missing tokens on protected resources
func AuthMiddleware(config AuthConfig) func(mizu.Handler) mizu.Handler {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			authHeader := c.Request().Header.Get("Authorization")

			// If no auth header provided
			if authHeader == "" {
				if config.AllowAnonymousPublic {
					// Allow request to proceed for potential public bucket access
					// The handler will check bucket permissions
					return next(c)
				}
				return writeError(c, http.StatusUnauthorized, fmt.Errorf("missing authorization header"))
			}

			// Parse Bearer token
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				return writeError(c, http.StatusUnauthorized, fmt.Errorf("invalid authorization header format"))
			}

			token := parts[1]
			if token == "" {
				return writeError(c, http.StatusUnauthorized, fmt.Errorf("missing token"))
			}

			// Verify token
			claims, err := verifyToken(token, config.JWTSecret)
			if err != nil {
				return writeError(c, http.StatusUnauthorized, fmt.Errorf("invalid token: %w", err))
			}

			// Store claims in request context
			ctx := c.Context()
			ctx = context.WithValue(ctx, ctxKeyClaims, claims)
			ctx = context.WithValue(ctx, ctxKeyRole, claims.Role)
			ctx = context.WithValue(ctx, ctxKeyUserID, claims.Sub)

			// Create new request with updated context
			req := c.Request().WithContext(ctx)
			*c.Request() = *req

			return next(c)
		}
	}
}

// GetClaims retrieves JWT claims from the request context.
func GetClaims(c *mizu.Ctx) (*Claims, bool) {
	claims, ok := c.Context().Value(ctxKeyClaims).(*Claims)
	return claims, ok
}

// GetRole retrieves the user role from the request context.
func GetRole(c *mizu.Ctx) (string, bool) {
	role, ok := c.Context().Value(ctxKeyRole).(string)
	return role, ok
}

// GetUserID retrieves the user ID from the request context.
func GetUserID(c *mizu.Ctx) (string, bool) {
	userID, ok := c.Context().Value(ctxKeyUserID).(string)
	return userID, ok
}

// RequireAuth is a middleware that requires authentication for an endpoint.
// Unlike AuthMiddleware which can allow anonymous access, this always requires a valid token.
func RequireAuth(config AuthConfig) func(mizu.Handler) mizu.Handler {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return writeError(c, http.StatusUnauthorized, fmt.Errorf("missing authorization header"))
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				return writeError(c, http.StatusUnauthorized, fmt.Errorf("invalid authorization header format"))
			}

			token := parts[1]
			claims, err := verifyToken(token, config.JWTSecret)
			if err != nil {
				return writeError(c, http.StatusUnauthorized, fmt.Errorf("invalid token: %w", err))
			}

			ctx := c.Context()
			ctx = context.WithValue(ctx, ctxKeyClaims, claims)
			ctx = context.WithValue(ctx, ctxKeyRole, claims.Role)
			ctx = context.WithValue(ctx, ctxKeyUserID, claims.Sub)

			// Create new request with updated context
			req := c.Request().WithContext(ctx)
			*c.Request() = *req

			return next(c)
		}
	}
}

// RequireRole returns a middleware that requires a specific role.
func RequireRole(config AuthConfig, roles ...string) func(mizu.Handler) mizu.Handler {
	return func(next mizu.Handler) mizu.Handler {
		// Chain auth middleware with role check
		authHandler := RequireAuth(config)(next)

		return func(c *mizu.Ctx) error {
			// Run auth middleware first
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return writeError(c, http.StatusUnauthorized, fmt.Errorf("missing authorization header"))
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				return writeError(c, http.StatusUnauthorized, fmt.Errorf("invalid authorization header format"))
			}

			token := parts[1]
			claims, err := verifyToken(token, config.JWTSecret)
			if err != nil {
				return writeError(c, http.StatusUnauthorized, fmt.Errorf("invalid token: %w", err))
			}

			// Check if user has one of the required roles
			hasRole := false
			for _, requiredRole := range roles {
				if claims.Role == requiredRole {
					hasRole = true
					break
				}
			}

			if !hasRole {
				return writeError(c, http.StatusForbidden, fmt.Errorf("insufficient permissions"))
			}

			// Store claims in context
			ctx := c.Context()
			ctx = context.WithValue(ctx, ctxKeyClaims, claims)
			ctx = context.WithValue(ctx, ctxKeyRole, claims.Role)
			ctx = context.WithValue(ctx, ctxKeyUserID, claims.Sub)

			req := c.Request().WithContext(ctx)
			*c.Request() = *req

			return authHandler(c)
		}
	}
}

// IsAuthenticated checks if the current request has valid authentication.
func IsAuthenticated(c *mizu.Ctx) bool {
	_, ok := GetClaims(c)
	return ok
}
