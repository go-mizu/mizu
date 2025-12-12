// Package rbac provides role-based access control middleware for Mizu.
package rbac

import (
	"context"
	"net/http"

	"github.com/go-mizu/mizu"
)

type contextKey struct{}

// User represents an authenticated user with roles.
type User struct {
	ID          string
	Roles       []string
	Permissions []string
}

// Options configures the RBAC middleware.
type Options struct {
	// GetUser extracts user from context.
	GetUser func(c *mizu.Ctx) *User

	// ErrorHandler handles authorization failures.
	ErrorHandler func(c *mizu.Ctx) error
}

// Set stores user in context.
func Set(c *mizu.Ctx, user *User) {
	ctx := context.WithValue(c.Context(), contextKey{}, user)
	req := c.Request().WithContext(ctx)
	*c.Request() = *req
}

// Get retrieves user from context.
func Get(c *mizu.Ctx) *User {
	if user, ok := c.Context().Value(contextKey{}).(*User); ok {
		return user
	}
	return nil
}

// HasRole checks if user has a role.
func HasRole(c *mizu.Ctx, role string) bool {
	user := Get(c)
	if user == nil {
		return false
	}
	for _, r := range user.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasPermission checks if user has a permission.
func HasPermission(c *mizu.Ctx, permission string) bool {
	user := Get(c)
	if user == nil {
		return false
	}
	for _, p := range user.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// HasAnyRole checks if user has any of the roles.
func HasAnyRole(c *mizu.Ctx, roles ...string) bool {
	for _, role := range roles {
		if HasRole(c, role) {
			return true
		}
	}
	return false
}

// HasAllRoles checks if user has all of the roles.
func HasAllRoles(c *mizu.Ctx, roles ...string) bool {
	for _, role := range roles {
		if !HasRole(c, role) {
			return false
		}
	}
	return true
}

// RequireRole creates middleware requiring a specific role.
func RequireRole(role string) mizu.Middleware {
	return RequireAnyRole(role)
}

// RequireAnyRole creates middleware requiring any of the roles.
func RequireAnyRole(roles ...string) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			if !HasAnyRole(c, roles...) {
				return c.Text(http.StatusForbidden, "Access denied")
			}
			return next(c)
		}
	}
}

// RequireAllRoles creates middleware requiring all of the roles.
func RequireAllRoles(roles ...string) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			if !HasAllRoles(c, roles...) {
				return c.Text(http.StatusForbidden, "Access denied")
			}
			return next(c)
		}
	}
}

// RequirePermission creates middleware requiring a specific permission.
func RequirePermission(permission string) mizu.Middleware {
	return RequireAnyPermission(permission)
}

// RequireAnyPermission creates middleware requiring any of the permissions.
func RequireAnyPermission(permissions ...string) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			user := Get(c)
			if user == nil {
				return c.Text(http.StatusForbidden, "Access denied")
			}

			for _, p := range permissions {
				if HasPermission(c, p) {
					return next(c)
				}
			}

			return c.Text(http.StatusForbidden, "Access denied")
		}
	}
}

// RequireAllPermissions creates middleware requiring all permissions.
func RequireAllPermissions(permissions ...string) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			for _, p := range permissions {
				if !HasPermission(c, p) {
					return c.Text(http.StatusForbidden, "Access denied")
				}
			}
			return next(c)
		}
	}
}

// Admin creates middleware requiring admin role.
func Admin() mizu.Middleware {
	return RequireRole("admin")
}

// Authenticated creates middleware requiring authenticated user.
func Authenticated() mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			if Get(c) == nil {
				return c.Text(http.StatusUnauthorized, "Authentication required")
			}
			return next(c)
		}
	}
}

// WithErrorHandler wraps a role middleware with custom error handling.
func WithErrorHandler(middleware mizu.Middleware, handler func(c *mizu.Ctx) error) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			// Create a wrapper to intercept 403 errors
			err := middleware(next)(c)
			if err != nil {
				return handler(c)
			}
			return nil
		}
	}
}
