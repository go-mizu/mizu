// Package oauth2 provides OAuth 2.0 resource server middleware for the Mizu web framework.
//
// # Overview
//
// This package implements OAuth 2.0 token validation for protecting API endpoints.
// It validates access tokens using either custom validation logic or RFC 7662 token
// introspection endpoints.
//
// # Features
//
//   - Multiple token sources: Authorization header, query parameters, form data
//   - Flexible token validation: custom validator or introspection endpoint
//   - Automatic scope verification
//   - Token expiration checking
//   - Context-based token access
//   - Configurable error handling
//   - Support for RFC 7662 token introspection
//
// # Basic Usage
//
// Create middleware with a custom validator:
//
//	validator := func(token string) (*oauth2.Token, error) {
//	    // Validate token against your auth server or database
//	    if isValid(token) {
//	        return &oauth2.Token{
//	            Value:   token,
//	            Subject: "user123",
//	            Scope:   []string{"read", "write"},
//	        }, nil
//	    }
//	    return nil, oauth2.ErrInvalidToken
//	}
//
//	app := mizu.New()
//	app.Use(oauth2.New(validator))
//
// # Token Introspection
//
// Use OAuth 2.0 token introspection (RFC 7662):
//
//	app.Use(oauth2.WithOptions(oauth2.Options{
//	    IntrospectionURL: "https://auth.example.com/introspect",
//	    ClientID:         "resource-server",
//	    ClientSecret:     "secret",
//	}))
//
// # Scope Enforcement
//
// Require specific scopes for access:
//
//	app.Use(oauth2.WithOptions(oauth2.Options{
//	    Validator:      validator,
//	    RequiredScopes: []string{"admin", "write"},
//	}))
//
// Or use the RequireScopes middleware for specific routes:
//
//	app.Use(oauth2.New(validator))
//	app.Get("/admin", oauth2.RequireScopes("admin"), adminHandler)
//
// # Token Lookup
//
// Configure where to find the token:
//
//	// From Authorization header (default)
//	TokenLookup: "header:Authorization"
//
//	// From query parameter
//	TokenLookup: "query:access_token"
//
//	// From form field
//	TokenLookup: "form:access_token"
//
// # Accessing Token Information
//
// Retrieve token data in handlers:
//
//	app.Get("/profile", func(c *mizu.Ctx) error {
//	    token := oauth2.Get(c)
//	    subject := oauth2.Subject(c)
//	    scopes := oauth2.Scopes(c)
//
//	    if oauth2.HasScope(c, "admin") {
//	        // Admin-specific logic
//	    }
//
//	    return c.JSON(200, map[string]interface{}{
//	        "user": subject,
//	        "scopes": scopes,
//	    })
//	})
//
// # Custom Error Handling
//
// Implement custom error responses:
//
//	app.Use(oauth2.WithOptions(oauth2.Options{
//	    Validator: validator,
//	    ErrorHandler: func(c *mizu.Ctx, err error) error {
//	        return c.JSON(401, map[string]string{
//	            "error": "unauthorized",
//	            "message": err.Error(),
//	        })
//	    },
//	}))
//
// # Security Considerations
//
//   - Always use HTTPS in production to protect tokens in transit
//   - Store client secrets securely (environment variables, secret managers)
//   - Validate token expiration times
//   - Use minimum required scopes (principle of least privilege)
//   - Set appropriate timeouts for introspection requests
//   - Monitor and log authentication failures
//
// # Error Types
//
// The package defines the following error constants:
//
//   - ErrMissingToken: No token found in request
//   - ErrInvalidToken: Token validation failed
//   - ErrExpiredToken: Token expiration time has passed
//   - ErrInsufficientScope: Token lacks required scopes
//   - ErrNoValidator: No validation method configured
//
// # Standards Compliance
//
// This implementation follows:
//
//   - RFC 6749: OAuth 2.0 Authorization Framework
//   - RFC 6750: OAuth 2.0 Bearer Token Usage
//   - RFC 7662: OAuth 2.0 Token Introspection
package oauth2
