/*
Package bearerauth provides Bearer token authentication middleware for Mizu.

# Overview

The bearerauth middleware validates Bearer tokens from the Authorization header,
commonly used for API authentication with JWTs, OAuth tokens, or custom tokens.
It implements RFC 6750 Bearer Token Usage standard.

# Basic Usage

Simple token validation:

	app := mizu.New()
	app.Use(bearerauth.New(func(token string) bool {
		return token == "my-secret-token"
	}))

# Validator Types

Two validator types are available:

TokenValidator - Simple boolean validation:

	type TokenValidator func(token string) bool

TokenValidatorWithContext - Returns claims along with validation result:

	type TokenValidatorWithContext func(token string) (claims any, valid bool)

# Configuration Options

The Options struct allows full configuration:

	type Options struct {
		Validator                TokenValidator                // Simple token validator
		ValidatorWithContext     TokenValidatorWithContext     // Validator that returns claims
		Header                   string                        // Header to read token from (default: "Authorization")
		AuthScheme               string                        // Auth scheme prefix (default: "Bearer")
		ErrorHandler             func(*mizu.Ctx, error) error  // Custom error handler
	}

# Examples

JWT validation with claims:

	type UserClaims struct {
		UserID   string
		Username string
		Role     string
	}

	app.Use(bearerauth.WithOptions(bearerauth.Options{
		ValidatorWithContext: func(token string) (any, bool) {
			claims, err := parseJWT(token)
			if err != nil {
				return nil, false
			}
			return &UserClaims{
				UserID:   claims["sub"].(string),
				Username: claims["username"].(string),
				Role:     claims["role"].(string),
			}, true
		},
	}))

	// Access claims in handler
	func protectedHandler(c *mizu.Ctx) error {
		claims, ok := bearerauth.Claims[*UserClaims](c)
		if !ok {
			return c.Text(401, "Unauthorized")
		}
		return c.JSON(200, map[string]string{
			"user": claims.Username,
			"role": claims.Role,
		})
	}

Custom header and auth scheme:

	app.Use(bearerauth.WithOptions(bearerauth.Options{
		Header:     "X-API-Token",
		AuthScheme: "Token",
		Validator: func(token string) bool {
			return validateToken(token)
		},
	}))

Database token lookup:

	app.Use(bearerauth.WithOptions(bearerauth.Options{
		ValidatorWithContext: func(token string) (any, bool) {
			session, err := db.GetSession(token)
			if err != nil || session.IsExpired() {
				return nil, false
			}
			return &UserInfo{
				ID:    session.UserID,
				Email: session.UserEmail,
			}, true
		},
	}))

# Error Handling

Three error types are defined:

	ErrTokenMissing   - Token not found in request (401 Unauthorized)
	ErrTokenInvalid   - Token validation failed (403 Forbidden)
	ErrInvalidScheme  - Auth scheme doesn't match (403 Forbidden)

Custom error handler:

	app.Use(bearerauth.WithOptions(bearerauth.Options{
		Validator: validateToken,
		ErrorHandler: func(c *mizu.Ctx, err error) error {
			switch err {
			case bearerauth.ErrTokenMissing:
				return c.JSON(401, map[string]string{
					"error": "Authentication required",
				})
			case bearerauth.ErrTokenInvalid:
				return c.JSON(403, map[string]string{
					"error": "Invalid token",
				})
			default:
				return c.JSON(401, map[string]string{
					"error": err.Error(),
				})
			}
		},
	}))

# Context Data Extraction

Three helper functions extract authentication data from context:

FromContext - Returns raw token or claims (type any):

	data := bearerauth.FromContext(c)

Token - Returns token string (when using Validator):

	token := bearerauth.Token(c)

Claims - Type-safe claims extraction (when using ValidatorWithContext):

	claims, ok := bearerauth.Claims[UserClaims](c)
	if ok {
		// Use claims
	}

# Security Considerations

1. Token Security - Use cryptographically secure tokens
2. HTTPS Required - Always use HTTPS to protect tokens in transit
3. Token Expiration - Implement token expiration
4. Secure Storage - Store tokens securely on the client
5. Revocation - Implement token revocation for logout/security

# Implementation Details

The middleware follows this validation flow:

1. Extracts the configured header (default: "Authorization")
2. Verifies the auth scheme matches (default: "Bearer")
3. Extracts the token after the scheme prefix
4. Validates the token using the configured validator
5. Stores token or claims in request context using a private context key

The private contextKey prevents key collisions with other middleware or
application code. When ValidatorWithContext returns claims, those are stored
in context; otherwise, the token string is stored.

# Best Practices

- Use short-lived tokens with refresh tokens for better security
- Validate token claims (expiration, issuer, audience)
- Log authentication failures for security monitoring
- Consider rate limiting failed authentication attempts
- Always validate tokens on the server side
- Rotate tokens regularly
*/
package bearerauth
