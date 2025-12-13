// Package jwt provides JSON Web Token (JWT) authentication middleware for Mizu.
//
// The JWT middleware validates tokens from HTTP requests and makes claims
// available to handlers through the request context. It supports HMAC-SHA256
// (HS256) signature verification and standard JWT claims validation.
//
// # Features
//
//   - HMAC-SHA256 signature verification
//   - Multiple token sources (header, query, cookie)
//   - Standard claims validation (exp, nbf, iss, aud)
//   - Configurable authorization scheme
//   - Custom error handling
//   - Type-safe context storage
//
// # Basic Usage
//
//	app := mizu.New()
//	secret := []byte("your-secret-key-at-least-32-bytes")
//
//	// Protect all routes
//	app.Use(jwt.New(secret))
//
//	app.Get("/protected", func(c *mizu.Ctx) error {
//	    claims := jwt.GetClaims(c)
//	    userID := jwt.Subject(c)
//	    return c.JSON(200, map[string]string{"user": userID})
//	})
//
// # Configuration
//
// The middleware can be configured using the Options struct:
//
//	app.Use(jwt.WithOptions(jwt.Options{
//	    Secret:      secret,
//	    TokenLookup: "header:Authorization",  // or "query:token" or "cookie:jwt"
//	    AuthScheme:  "Bearer",
//	    Issuer:      "my-app",
//	    Audience:    []string{"api", "web"},
//	    ErrorHandler: func(c *mizu.Ctx, err error) error {
//	        return c.JSON(401, map[string]string{"error": err.Error()})
//	    },
//	}))
//
// # Token Validation Process
//
// The middleware performs the following validation steps:
//
//  1. Token Extraction: Extracts the token from the configured source
//     (Authorization header, query parameter, or cookie).
//
//  2. Structure Validation: Ensures the token has exactly three parts
//     (header.payload.signature) separated by dots.
//
//  3. Signature Verification: Verifies the HMAC-SHA256 signature using
//     the provided secret key. Uses constant-time comparison to prevent
//     timing attacks.
//
//  4. Payload Decoding: Decodes the base64url-encoded payload and
//     unmarshals the JSON claims.
//
//  5. Claims Validation: Validates standard JWT claims:
//     - exp (expiration): Rejects tokens past their expiration time
//     - nbf (not before): Rejects tokens used before their valid time
//     - iss (issuer): Validates against configured issuer if specified
//     - aud (audience): Validates against configured audience list if specified
//
// # Error Handling
//
// The middleware defines specific error types for different failure scenarios:
//
//   - ErrTokenMissing: No token found in the configured source (401 Unauthorized)
//   - ErrTokenMalformed: Invalid token structure or encoding (403 Forbidden)
//   - ErrTokenInvalid: Invalid signature (403 Forbidden)
//   - ErrTokenExpired: Token has expired (403 Forbidden)
//   - ErrTokenNotYetValid: Token not yet valid (nbf claim) (403 Forbidden)
//   - ErrInvalidScheme: Wrong authorization scheme (403 Forbidden)
//   - ErrInvalidIssuer: Token issuer doesn't match configured issuer (403 Forbidden)
//   - ErrInvalidAudience: Token audience doesn't match configured audience (403 Forbidden)
//
// Custom error handlers can be provided via the ErrorHandler option to
// customize error responses.
//
// # Accessing Claims
//
// Claims are stored in the request context and can be accessed using helper functions:
//
//	// Get all claims
//	claims := jwt.GetClaims(c)
//	email := claims["email"].(string)
//
//	// Get subject (user ID)
//	userID := jwt.Subject(c)
//
// # Security Considerations
//
//   - Use strong secrets (at least 32 bytes for HS256)
//   - Always validate token expiration (automatically checked by middleware)
//   - Use HTTPS to prevent token interception
//   - Use short expiration times with refresh tokens for long sessions
//   - Validate issuer to prevent tokens from other applications
//   - Include only necessary data in claims to minimize token size
//
// # Token Sources
//
// The middleware supports three token sources, configured via TokenLookup:
//
//  1. Header (default): "header:Authorization"
//     Extracts from Authorization header with configurable scheme.
//     Example: "Bearer <token>"
//
//  2. Query: "query:token"
//     Extracts from URL query parameter.
//     Example: /api/resource?token=<token>
//
//  3. Cookie: "cookie:jwt"
//     Extracts from HTTP cookie.
//     Example: Cookie: jwt=<token>
//
// # Standard JWT Claims
//
// The middleware validates the following standard JWT claims:
//
//   - sub (subject): User or entity identifier
//   - iss (issuer): Token issuer
//   - aud (audience): Intended audience (supports string or array)
//   - exp (expiration): Expiration timestamp (Unix time)
//   - nbf (not before): Valid from timestamp (Unix time)
//   - iat (issued at): Issue timestamp (Unix time)
//
// # Examples
//
// Protect a route group:
//
//	// Public routes
//	app.Get("/", publicHandler)
//
//	// Protected API routes
//	api := app.Group("/api")
//	api.Use(jwt.New(secret))
//	api.Get("/users", listUsers)
//	api.Post("/users", createUser)
//
// Token from query parameter:
//
//	app.Use(jwt.WithOptions(jwt.Options{
//	    Secret:      secret,
//	    TokenLookup: "query:token",
//	    AuthScheme:  "", // No scheme for query params
//	}))
//
// Token from cookie:
//
//	app.Use(jwt.WithOptions(jwt.Options{
//	    Secret:      secret,
//	    TokenLookup: "cookie:auth_token",
//	}))
//
// Custom error handling:
//
//	app.Use(jwt.WithOptions(jwt.Options{
//	    Secret: secret,
//	    ErrorHandler: func(c *mizu.Ctx, err error) error {
//	        return c.JSON(401, map[string]string{
//	            "error": "Invalid or expired token",
//	        })
//	    },
//	}))
package jwt
