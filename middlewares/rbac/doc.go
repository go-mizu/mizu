// Package rbac provides role-based access control (RBAC) middleware for the Mizu web framework.
//
// The rbac package implements a flexible role and permission-based authorization system
// that integrates seamlessly with Mizu's middleware chain. It allows you to protect routes
// based on user roles, permissions, or authentication status.
//
// # Overview
//
// The package centers around a User struct that contains roles and permissions. User
// information is stored in the request context and can be accessed throughout the
// request lifecycle.
//
// # Basic Usage
//
// Store a user in the context:
//
//	user := &rbac.User{
//	    ID: "123",
//	    Roles: []string{"admin", "editor"},
//	    Permissions: []string{"read:posts", "write:posts"},
//	}
//	rbac.Set(c, user)
//
// Protect a route with role requirements:
//
//	app.Get("/admin", handler, rbac.RequireRole("admin"))
//	app.Get("/dashboard", handler, rbac.RequireAnyRole("admin", "manager"))
//	app.Get("/super", handler, rbac.RequireAllRoles("admin", "superuser"))
//
// Protect routes with permission requirements:
//
//	app.Post("/posts", createPost, rbac.RequirePermission("write:posts"))
//	app.Delete("/posts/:id", deletePost, rbac.RequireAllPermissions("write:posts", "admin:access"))
//
// # Context Management
//
// User information is stored in the request context using a type-safe context key:
//
//   - Set(c *mizu.Ctx, user *User) stores a user in the context
//   - Get(c *mizu.Ctx) *User retrieves the user from the context, returns nil if not found
//
// # Role and Permission Checking
//
// The package provides helper functions for checking access without middleware:
//
//   - HasRole(c, role) checks if the user has a specific role
//   - HasAnyRole(c, roles...) checks if the user has any of the specified roles
//   - HasAllRoles(c, roles...) checks if the user has all specified roles
//   - HasPermission(c, permission) checks if the user has a specific permission
//
// Example:
//
//	if rbac.HasRole(c, "admin") {
//	    // User is an admin
//	}
//
// # Middleware Functions
//
// Role-based middlewares enforce role requirements:
//
//   - RequireRole(role) requires a specific role
//   - RequireAnyRole(roles...) requires any of the specified roles (OR logic)
//   - RequireAllRoles(roles...) requires all specified roles (AND logic)
//
// Permission-based middlewares enforce permission requirements:
//
//   - RequirePermission(permission) requires a specific permission
//   - RequireAnyPermission(permissions...) requires any of the permissions
//   - RequireAllPermissions(permissions...) requires all permissions
//
// Convenience middlewares:
//
//   - Admin() is a shorthand for RequireRole("admin")
//   - Authenticated() checks if any user is present in the context
//
// # Error Handling
//
// By default, authorization failures return:
//   - HTTP 403 Forbidden with "Access denied" text for role/permission failures
//   - HTTP 401 Unauthorized with "Authentication required" for missing users
//
// Custom error handlers can be wrapped using WithErrorHandler:
//
//	middleware := rbac.WithErrorHandler(
//	    rbac.RequireRole("admin"),
//	    func(c *mizu.Ctx) error {
//	        return c.JSON(403, map[string]string{
//	            "error": "Insufficient permissions",
//	        })
//	    },
//	)
//
// # Integration Example
//
// A complete example integrating with authentication:
//
//	app := mizu.New()
//
//	// Authentication middleware that sets user in context
//	app.Use(func(next mizu.Handler) mizu.Handler {
//	    return func(c *mizu.Ctx) error {
//	        // Extract user from JWT, session, etc.
//	        user := extractUserFromAuth(c)
//	        if user != nil {
//	            rbac.Set(c, user)
//	        }
//	        return next(c)
//	    }
//	})
//
//	// Public routes
//	app.Get("/", homeHandler)
//
//	// Authenticated routes
//	app.Get("/profile", profileHandler, rbac.Authenticated())
//
//	// Admin-only routes
//	app.Get("/admin", adminHandler, rbac.Admin())
//
//	// Permission-based routes
//	app.Post("/posts", createPost, rbac.RequirePermission("write:posts"))
//
// # Thread Safety
//
// The package is safe for concurrent use. User data is stored in the request context,
// which is immutable and request-scoped.
package rbac
