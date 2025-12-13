// Package keyauth provides API key authentication middleware for Mizu.
//
// The keyauth middleware validates API keys from headers, query parameters,
// or cookies. It's designed for machine-to-machine authentication, service-to-service
// communication, and securing public APIs without user sessions.
//
// # Basic Usage
//
// Create middleware with a static list of valid keys:
//
//	app := mizu.New()
//	app.Use(keyauth.New(keyauth.ValidateKeys(
//	    "key-abc123",
//	    "key-def456",
//	)))
//
// # Custom Validation
//
// Provide a custom validator function for database lookups or complex validation:
//
//	app.Use(keyauth.New(func(key string) (bool, error) {
//	    // Look up key in database
//	    apiKey, err := db.GetAPIKey(key)
//	    if err != nil {
//	        if err == sql.ErrNoRows {
//	            return false, nil // Key not found
//	        }
//	        return false, err // Database error
//	    }
//
//	    // Check if key is active and not expired
//	    if !apiKey.Active || apiKey.ExpiresAt.Before(time.Now()) {
//	        return false, nil
//	    }
//
//	    return true, nil
//	}))
//
// # Key Sources
//
// The middleware can extract keys from different sources using the KeyLookup option:
//
//	// From header (default: X-API-Key)
//	keyauth.WithOptions(keyauth.Options{
//	    KeyLookup: "header:X-API-Key",
//	    Validator: validateKey,
//	})
//
//	// From query parameter
//	keyauth.WithOptions(keyauth.Options{
//	    KeyLookup: "query:api_key",
//	    Validator: validateKey,
//	})
//
//	// From cookie
//	keyauth.WithOptions(keyauth.Options{
//	    KeyLookup: "cookie:auth_token",
//	    Validator: validateKey,
//	})
//
// # Auth Scheme
//
// Support authorization headers with scheme prefixes:
//
//	// Expect "ApiKey xxx" in Authorization header
//	keyauth.WithOptions(keyauth.Options{
//	    KeyLookup:  "header:Authorization",
//	    AuthScheme: "ApiKey",
//	    Validator:  validateKey,
//	})
//
// The middleware automatically strips the scheme prefix before validation.
//
// # Accessing the Validated Key
//
// Retrieve the validated API key in handlers for logging or rate limiting:
//
//	func handler(c *mizu.Ctx) error {
//	    key := keyauth.Get(c)
//	    c.Logger().Info("API request", "key", key[:8]+"...")
//	    return c.JSON(200, data)
//	}
//
// # Custom Error Handling
//
// Provide custom error responses:
//
//	keyauth.WithOptions(keyauth.Options{
//	    Validator: validateKey,
//	    ErrorHandler: func(c *mizu.Ctx, err error) error {
//	        if err == keyauth.ErrKeyMissing {
//	            return c.JSON(401, map[string]string{
//	                "error": "API key required",
//	                "hint":  "Include X-API-Key header",
//	            })
//	        }
//	        return c.JSON(403, map[string]string{
//	            "error": "Invalid API key",
//	        })
//	    },
//	})
//
// # Error Types
//
// The middleware defines two error constants:
//
//   - ErrKeyMissing: API key not found in the request (returns 401 Unauthorized)
//   - ErrKeyInvalid: API key validation failed (returns 403 Forbidden)
//
// When the validator function returns an error, the middleware treats it as
// a validation failure and returns 403 Forbidden with the error message.
//
// # Security Best Practices
//
//   - Generate keys using cryptographically secure random sources
//   - Hash keys in database, never store plain text
//   - Implement key rotation without downtime
//   - Use scoped keys with limited permissions
//   - Log key usage without logging full keys
//   - Add rate limiting per API key
//   - Support key expiration
//
// # Implementation Details
//
// The middleware follows this validation pipeline:
//
//  1. Extract key from the configured source (header, query, or cookie)
//  2. If key is missing, return 401 Unauthorized
//  3. Execute the custom validator function
//  4. If validation fails or returns error, return 403 Forbidden
//  5. Store validated key in request context
//  6. Call next handler
//
// The validated key is stored using a private contextKey type to prevent
// context key collisions and ensure type-safe retrieval.
//
// # Performance Considerations
//
// The ValidateKeys helper creates a map for O(1) key lookup:
//
//	validator := keyauth.ValidateKeys("key1", "key2", "key3")
//
// This is efficient for static key lists. For database lookups, implement
// caching in your validator function to reduce database queries.
package keyauth
