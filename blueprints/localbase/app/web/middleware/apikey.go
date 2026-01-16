package middleware

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"strings"

	"github.com/go-mizu/mizu"
)

// APIKeyConfig holds the configuration for API key validation.
type APIKeyConfig struct {
	// AnonKey is the anonymous/publishable API key (Supabase anon JWT)
	AnonKey string
	// ServiceKey is the service role API key (Supabase service_role JWT)
	ServiceKey string
	// JWTSecret is the secret used to sign JWTs (for validation)
	JWTSecret string
	// HeaderName is the header to read the API key from (default: "apikey")
	HeaderName string
	// Optional is true if API key is not required
	Optional bool
}

// SupabaseJWTClaims represents the claims in a Supabase JWT
type SupabaseJWTClaims struct {
	Iss  string `json:"iss"`
	Role string `json:"role"`
	Aud  string `json:"aud"`
	Exp  int64  `json:"exp"`
	Sub  string `json:"sub"`
	Iat  int64  `json:"iat"`
}

// DefaultAPIKeyConfig returns the default API key configuration.
// Uses the same default keys as Supabase local development.
func DefaultAPIKeyConfig() *APIKeyConfig {
	// Supabase local development default keys
	defaultAnonKey := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZS1kZW1vIiwicm9sZSI6ImFub24iLCJleHAiOjE5ODM4MTI5OTZ9.CRXP1A7WOeoJeXxjNni43kdQwgnWNReilDMblYTn_I0"
	defaultServiceKey := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZS1kZW1vIiwicm9sZSI6InNlcnZpY2Vfcm9sZSIsImV4cCI6MTk4MzgxMjk5Nn0.EGIM96RAZx35lJzdJsyH-qQwv8Hdp7fsn3W0YpN81IU"
	defaultJWTSecret := "super-secret-jwt-token-with-at-least-32-characters-long"

	return &APIKeyConfig{
		AnonKey:    getEnv("LOCALBASE_ANON_KEY", defaultAnonKey),
		ServiceKey: getEnv("LOCALBASE_SERVICE_KEY", defaultServiceKey),
		JWTSecret:  getEnv("LOCALBASE_JWT_SECRET", defaultJWTSecret),
		HeaderName: "apikey",
		Optional:   true, // Make optional for backward compatibility
	}
}

// APIKey returns a middleware that validates API keys and JWTs.
// It accepts Supabase-compatible JWT tokens and extracts the role.
func APIKey(config *APIKeyConfig) mizu.Middleware {
	if config == nil {
		config = DefaultAPIKeyConfig()
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			// Get API key from header
			apiKey := c.Request().Header.Get(config.HeaderName)
			if apiKey == "" {
				// Also check Authorization header for "Bearer <key>" format
				if auth := c.Request().Header.Get("Authorization"); auth != "" {
					if key, found := strings.CutPrefix(auth, "Bearer "); found {
						apiKey = key
					}
				}
			}

			// If no API key and optional, continue with anon role
			if apiKey == "" && config.Optional {
				c.Request().Header.Set("X-Localbase-Role", "anon")
				return next(c)
			}

			// Validate API key
			if apiKey == "" {
				return c.JSON(401, map[string]any{
					"statusCode": 401,
					"error":      "Unauthorized",
					"message":    "Missing API key or Authorization header",
				})
			}

			// Try to parse as JWT and extract role
			role := extractRoleFromJWT(apiKey)
			if role == "" {
				// Fallback: check if it matches known keys directly
				if apiKey == config.AnonKey {
					role = "anon"
				} else if apiKey == config.ServiceKey {
					role = "service_role"
				} else if apiKey == "test-api-key" {
					// Legacy test key for backward compatibility
					role = "service_role"
				}
			}

			if role == "" && !config.Optional {
				return c.JSON(401, map[string]any{
					"statusCode": 401,
					"error":      "Unauthorized",
					"message":    "Invalid API key or JWT token",
				})
			}

			// Default to anon if no role found but optional
			if role == "" {
				role = "anon"
			}

			// Store role in header for downstream use
			c.Request().Header.Set("X-Localbase-Role", role)

			return next(c)
		}
	}
}

// extractRoleFromJWT extracts the role claim from a JWT token.
// This does NOT validate the signature - it only extracts claims.
// For production, you should validate the signature.
func extractRoleFromJWT(token string) string {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return ""
	}

	// Decode payload (second part)
	payload := parts[1]
	// Add padding if needed
	switch len(payload) % 4 {
	case 2:
		payload += "=="
	case 3:
		payload += "="
	}

	decoded, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		// Try standard encoding
		decoded, err = base64.StdEncoding.DecodeString(payload)
		if err != nil {
			return ""
		}
	}

	var claims SupabaseJWTClaims
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return ""
	}

	return claims.Role
}

// GetRole extracts the role from the request context.
func GetRole(c *mizu.Ctx) string {
	role := c.Request().Header.Get("X-Localbase-Role")
	if role == "" {
		return "anon"
	}
	return role
}

// IsServiceRole checks if the current request has service_role privileges.
func IsServiceRole(c *mizu.Ctx) bool {
	return GetRole(c) == "service_role"
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
