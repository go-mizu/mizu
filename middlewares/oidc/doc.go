// Package oidc provides OpenID Connect (OIDC) authentication middleware for Mizu applications.
//
// The OIDC middleware validates JWT tokens issued by OpenID Connect providers,
// verifying claims and providing access control based on groups, roles, and scopes.
//
// # Basic Usage
//
// Simple OIDC authentication with issuer and client ID:
//
//	app := mizu.New()
//	app.Use(oidc.New("https://accounts.google.com", "your-client-id"))
//
//	app.Get("/api/profile", func(c *mizu.Ctx) error {
//	    claims := oidc.GetClaims(c)
//	    return c.JSON(200, claims)
//	})
//
// # Advanced Configuration
//
// Configure the middleware with custom options:
//
//	app.Use(oidc.WithOptions(oidc.Options{
//	    IssuerURL:       "https://accounts.google.com",
//	    ClientID:        "your-client-id",
//	    Audience:        "custom-audience",
//	    SkipPaths:       []string{"/health", "/public"},
//	    RefreshInterval: 30 * time.Minute,
//	    OnError: func(c *mizu.Ctx, err error) error {
//	        log.Printf("Auth error: %v", err)
//	        return c.JSON(401, map[string]string{"error": "unauthorized"})
//	    },
//	}))
//
// # Token Extraction
//
// By default, tokens are extracted from the Authorization header using Bearer scheme.
// Custom extraction can be configured:
//
//	app.Use(oidc.WithOptions(oidc.Options{
//	    IssuerURL: "https://issuer.example.com",
//	    ClientID:  "client-id",
//	    TokenExtractor: func(r *http.Request) string {
//	        // Extract from query parameter
//	        return r.URL.Query().Get("access_token")
//	    },
//	}))
//
// # Claims Access
//
// Access validated claims from the context:
//
//	app.Get("/api/user", func(c *mizu.Ctx) error {
//	    claims := oidc.GetClaims(c)
//	    if claims == nil {
//	        return c.JSON(401, map[string]string{"error": "unauthorized"})
//	    }
//
//	    return c.JSON(200, map[string]any{
//	        "subject": claims.Subject,
//	        "email":   claims.Email,
//	        "name":    claims.Name,
//	    })
//	})
//
// # Authorization Middleware
//
// Require specific groups, roles, or scopes:
//
//	// Require admin group
//	adminRoutes := app.Group("/admin")
//	adminRoutes.Use(oidc.RequireGroup("admin"))
//
//	// Require editor role
//	app.Get("/api/edit", oidc.RequireRole("editor")(func(c *mizu.Ctx) error {
//	    return c.JSON(200, map[string]string{"status": "ok"})
//	}))
//
//	// Require specific scope
//	app.Get("/api/read", oidc.RequireScope("read:api")(func(c *mizu.Ctx) error {
//	    return c.JSON(200, map[string]string{"status": "ok"})
//	}))
//
// # Claims Validation
//
// The middleware validates the following:
//
//   - Token format (3-part JWT structure)
//   - Issuer (iss claim matches configured IssuerURL)
//   - Audience (aud claim matches configured Audience or ClientID)
//   - Expiration (exp claim is in the future)
//   - Not Before (nbf claim is in the past, if present)
//
// # Custom Claims
//
// Access custom provider-specific claims using the Raw field:
//
//	claims := oidc.GetClaims(c)
//	if customField, ok := claims.Raw["custom_field"].(string); ok {
//	    // Use custom field
//	}
//
// # Error Handling
//
// The middleware returns the following errors:
//
//   - ErrNoToken: No token provided in request
//   - ErrInvalidToken: Token format is invalid
//   - ErrTokenExpired: Token expiration time has passed
//   - ErrInvalidIssuer: Token issuer doesn't match configuration
//   - ErrInvalidAudience: Token audience doesn't match configuration
//   - ErrKeyNotFound: Signing key not found in JWKS
//   - ErrInvalidSignature: Token signature verification failed
//
// Custom error handling can be configured via OnError option:
//
//	app.Use(oidc.WithOptions(oidc.Options{
//	    IssuerURL: "https://issuer.example.com",
//	    ClientID:  "client-id",
//	    OnError: func(c *mizu.Ctx, err error) error {
//	        // Log error
//	        log.Printf("OIDC error: %v", err)
//
//	        // Return custom response
//	        if err == oidc.ErrTokenExpired {
//	            return c.JSON(401, map[string]string{
//	                "error": "token_expired",
//	                "message": "Please log in again",
//	            })
//	        }
//
//	        return c.JSON(401, map[string]string{"error": "unauthorized"})
//	    },
//	}))
//
// # Path Skipping
//
// Skip authentication for specific paths:
//
//	app.Use(oidc.WithOptions(oidc.Options{
//	    IssuerURL: "https://issuer.example.com",
//	    ClientID:  "client-id",
//	    SkipPaths: []string{
//	        "/health",
//	        "/public",
//	        "/docs",
//	    },
//	}))
//
// # Integration with OIDC Providers
//
// Google:
//
//	app.Use(oidc.New("https://accounts.google.com", "your-google-client-id"))
//
// Auth0:
//
//	app.Use(oidc.New("https://your-tenant.auth0.com/", "your-auth0-client-id"))
//
// Keycloak:
//
//	app.Use(oidc.New("https://keycloak.example.com/realms/myrealm", "your-client-id"))
//
// Azure AD:
//
//	app.Use(oidc.New("https://login.microsoftonline.com/{tenant}/v2.0", "your-client-id"))
//
// # Security Considerations
//
//   - Always use HTTPS in production
//   - Configure appropriate token expiration times
//   - Regularly rotate signing keys (configure RefreshInterval)
//   - Store only necessary claims in sessions
//   - Use SkipPaths instead of disabling middleware for public routes
//   - Implement rate limiting for authentication endpoints
//   - Log authentication failures for security monitoring
//
// # Performance
//
// The middleware is optimized for performance:
//
//   - JWKS keys are cached and refreshed at configured intervals
//   - Claims are stored in request context for fast access
//   - Token validation is performed once per request
//   - Path skipping uses a map for O(1) lookup
package oidc
