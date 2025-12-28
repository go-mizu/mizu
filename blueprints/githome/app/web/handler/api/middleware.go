package api

import (
	"context"
	"strings"

	"github.com/go-mizu/blueprints/githome/feature/users"
	"github.com/go-mizu/mizu"
)

// RequireAuth returns middleware that requires authentication
func RequireAuth(usersAPI users.API) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			user := extractUser(c, usersAPI)
			if user == nil {
				return Unauthorized(c)
			}
			ctx := context.WithValue(c.Context(), UserContextKey, user)
			*c.Request() = *c.Request().WithContext(ctx)
			return next(c)
		}
	}
}

// OptionalAuth returns middleware that optionally authenticates
func OptionalAuth(usersAPI users.API) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			user := extractUser(c, usersAPI)
			if user != nil {
				ctx := context.WithValue(c.Context(), UserContextKey, user)
				*c.Request() = *c.Request().WithContext(ctx)
			}
			return next(c)
		}
	}
}

// extractUser extracts user from Authorization header
func extractUser(c *mizu.Ctx, usersAPI users.API) *users.User {
	auth := c.Request().Header.Get("Authorization")
	if auth == "" {
		return nil
	}

	// Support Basic auth and Bearer token
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 {
		return nil
	}

	switch strings.ToLower(parts[0]) {
	case "basic":
		return extractBasicAuth(c, usersAPI)
	case "bearer", "token":
		return extractTokenAuth(c, usersAPI, parts[1])
	default:
		return nil
	}
}

// extractBasicAuth extracts user from Basic auth
func extractBasicAuth(c *mizu.Ctx, usersAPI users.API) *users.User {
	username, password, ok := c.Request().BasicAuth()
	if !ok {
		return nil
	}

	user, err := usersAPI.Authenticate(c.Context(), username, password)
	if err != nil {
		return nil
	}
	return user
}

// extractTokenAuth extracts user from Bearer token
func extractTokenAuth(c *mizu.Ctx, usersAPI users.API, token string) *users.User {
	// TODO: Implement token-based authentication
	// For now, treat token as user login for simplicity
	user, err := usersAPI.GetByLogin(c.Context(), token)
	if err != nil {
		return nil
	}
	return user
}
