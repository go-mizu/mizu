// Package basicauth provides HTTP Basic Authentication middleware for Mizu.
//
// HTTP Basic Authentication is a simple authentication scheme built into the HTTP protocol.
// It prompts users for a username and password, which are sent with each request as a
// base64-encoded string in the Authorization header.
//
// # Quick Start
//
// The simplest way to use basicauth is with static credentials:
//
//	app := mizu.New()
//	credentials := map[string]string{
//	    "admin": "secret",
//	    "user":  "password",
//	}
//	app.Use(basicauth.New(credentials))
//
// # Configuration Options
//
// The middleware supports several configuration methods:
//
//   - New(credentials): Create middleware with static credentials map
//   - WithValidator(fn): Use custom validation function (e.g., database lookup)
//   - WithRealm(realm, credentials): Customize the authentication realm
//   - WithOptions(opts): Full configuration with all options
//
// # Custom Validation
//
// For dynamic credential validation (e.g., from a database):
//
//	app.Use(basicauth.WithValidator(func(user, pass string) bool {
//	    dbUser, err := db.GetUser(user)
//	    if err != nil {
//	        return false
//	    }
//	    return bcrypt.CompareHashAndPassword(
//	        []byte(dbUser.PasswordHash),
//	        []byte(pass),
//	    ) == nil
//	}))
//
// # Custom Error Handling
//
// Customize the response when authentication fails:
//
//	app.Use(basicauth.WithOptions(basicauth.Options{
//	    Validator: func(user, pass string) bool {
//	        return user == "admin" && pass == "secret"
//	    },
//	    ErrorHandler: func(c *mizu.Ctx) error {
//	        return c.JSON(401, map[string]string{
//	            "error": "Invalid credentials",
//	        })
//	    },
//	}))
//
// # Route-Specific Protection
//
// Protect only specific routes or groups:
//
//	// Public routes
//	app.Get("/", publicHandler)
//
//	// Protected admin routes
//	adminAuth := basicauth.New(map[string]string{"admin": "secret"})
//	app.Get("/admin", adminDashboard, adminAuth)
//
//	// Or protect an entire group
//	admin := app.Group("/admin")
//	admin.Use(adminAuth)
//	admin.Get("/", adminDashboard)
//	admin.Get("/users", adminUsers)
//
// # Security Features
//
// The middleware implements several security best practices:
//
//   - Constant-time comparison: Uses crypto/subtle to prevent timing attacks
//   - SHA-256 hashing: Normalizes password lengths before comparison
//   - WWW-Authenticate header: Proper HTTP Basic Auth challenge on failure
//
// The built-in secureCompare function protects against timing attacks by:
// 1. Hashing both strings with SHA-256 to normalize lengths
// 2. Using subtle.ConstantTimeCompare for comparison
// 3. Preventing information leakage through response timing
//
// # Security Considerations
//
// Important security notes:
//
//   - Always use HTTPS: Basic auth sends credentials base64-encoded, not encrypted
//   - Use strong passwords: Avoid dictionary words and short passwords
//   - Hash stored passwords: Never store plain-text passwords
//   - Rate limiting: Combine with rate limiting middleware to prevent brute force
//   - Production use: Consider OAuth2 or JWT for production user-facing authentication
//
// # Implementation Details
//
// The middleware implements RFC 7617 (HTTP Basic Authentication) with the following flow:
//
//  1. Extract Authorization header from request
//  2. Verify it starts with "Basic " prefix
//  3. Decode base64-encoded credentials
//  4. Split on first colon to get username:password
//  5. Call validator function to verify credentials
//  6. On success, call next handler; on failure, return 401 with WWW-Authenticate header
//
// The middleware returns 401 Unauthorized in these cases:
//   - Missing Authorization header
//   - Invalid authorization scheme (not "Basic")
//   - Invalid base64 encoding
//   - Malformed credentials (no colon separator)
//   - Validator function returns false
//
// # Related Middlewares
//
//   - bearerauth: Token-based authentication
//   - keyauth: API key authentication
//   - secure: HTTPS enforcement
//   - ratelimit: Rate limiting for brute force protection
package basicauth
