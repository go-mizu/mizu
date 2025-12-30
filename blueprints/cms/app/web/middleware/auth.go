// Package middleware provides HTTP middleware.
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-mizu/blueprints/cms/feature/auth"
	"github.com/go-mizu/mizu"
)

// userContextKey is the context key for user data.
type userContextKey struct{}

// UserFromContext extracts user data from context.
func UserFromContext(ctx context.Context) map[string]any {
	user, _ := ctx.Value(userContextKey{}).(map[string]any)
	return user
}

// Auth provides authentication middleware.
type Auth struct {
	authService auth.API
}

// NewAuth creates a new Auth middleware.
func NewAuth(authService auth.API) *Auth {
	return &Auth{authService: authService}
}

// RequireAuth requires authentication for the route.
func (m *Auth) RequireAuth(next mizu.Handler) mizu.Handler {
	return func(c *mizu.Ctx) error {
		token := m.extractToken(c)
		if token == "" {
			return c.JSON(http.StatusUnauthorized, map[string]any{
				"errors": []map[string]string{
					{"message": "You are not allowed to perform this action."},
				},
			})
		}

		userID, collection, err := m.authService.ValidateToken(token)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]any{
				"errors": []map[string]string{
					{"message": "You are not allowed to perform this action."},
				},
			})
		}

		// Get full user data
		user, err := m.authService.Me(c.Context(), collection, token)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]any{
				"errors": []map[string]string{
					{"message": "You are not allowed to perform this action."},
				},
			})
		}

		// Add user to context
		userData := map[string]any{
			"id":         userID,
			"collection": collection,
			"email":      user.Email,
			"firstName":  user.FirstName,
			"lastName":   user.LastName,
			"roles":      user.Roles,
		}
		ctx := context.WithValue(c.Request().Context(), userContextKey{}, userData)
		*c.Request() = *c.Request().WithContext(ctx)

		return next(c)
	}
}

// OptionalAuth attempts to authenticate but doesn't require it.
func (m *Auth) OptionalAuth(next mizu.Handler) mizu.Handler {
	return func(c *mizu.Ctx) error {
		token := m.extractToken(c)
		if token == "" {
			return next(c)
		}

		userID, collection, err := m.authService.ValidateToken(token)
		if err != nil {
			return next(c)
		}

		user, err := m.authService.Me(c.Context(), collection, token)
		if err != nil {
			return next(c)
		}

		userData := map[string]any{
			"id":         userID,
			"collection": collection,
			"email":      user.Email,
			"firstName":  user.FirstName,
			"lastName":   user.LastName,
			"roles":      user.Roles,
		}
		ctx := context.WithValue(c.Request().Context(), userContextKey{}, userData)
		*c.Request() = *c.Request().WithContext(ctx)

		return next(c)
	}
}

func (m *Auth) extractToken(c *mizu.Ctx) string {
	// Try Authorization header first
	authHeader := c.Request().Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	// Try cookie
	if cookie, err := c.Request().Cookie("payload-token"); err == nil {
		return cookie.Value
	}

	return ""
}
