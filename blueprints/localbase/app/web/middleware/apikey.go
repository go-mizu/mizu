package middleware

import (
	"os"
	"strings"

	"github.com/go-mizu/mizu"
)

// APIKeyConfig holds the configuration for API key validation.
type APIKeyConfig struct {
	// AnonKey is the anonymous/publishable API key (default: from env LOCALBASE_ANON_KEY)
	AnonKey string
	// ServiceKey is the service role API key (default: from env LOCALBASE_SERVICE_KEY)
	ServiceKey string
	// HeaderName is the header to read the API key from (default: "apikey")
	HeaderName string
	// Optional is true if API key is not required (for backward compatibility)
	Optional bool
}

// DefaultAPIKeyConfig returns the default API key configuration.
func DefaultAPIKeyConfig() *APIKeyConfig {
	return &APIKeyConfig{
		AnonKey:    getEnv("LOCALBASE_ANON_KEY", "sb_publishable_ACJWlzQHlZjBrEguHvfOxg_3BJgxAaH"),
		ServiceKey: getEnv("LOCALBASE_SERVICE_KEY", ""),
		HeaderName: "apikey",
		Optional:   true, // Make optional for backward compatibility during development
	}
}

// APIKey returns a middleware that validates API keys.
// It accepts both the anon key and service key.
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

			// If no API key and optional, continue
			if apiKey == "" && config.Optional {
				return next(c)
			}

			// Validate API key
			if apiKey == "" {
				return c.JSON(401, map[string]any{
					"code":    "PGRST302",
					"message": "Anonymous access disabled, authentication required",
					"details": nil,
					"hint":    "Provide an apikey header or Authorization Bearer token",
				})
			}

			// Check if API key matches anon or service key
			validKey := false
			isServiceRole := false

			if config.AnonKey != "" && apiKey == config.AnonKey {
				validKey = true
			}
			if config.ServiceKey != "" && apiKey == config.ServiceKey {
				validKey = true
				isServiceRole = true
			}

			// For backward compatibility, also accept the legacy test key
			if apiKey == "test-api-key" {
				validKey = true
			}

			if !validKey && !config.Optional {
				return c.JSON(401, map[string]any{
					"code":    "PGRST301",
					"message": "JWT invalid or API key invalid",
					"details": nil,
					"hint":    nil,
				})
			}

			// Store API key info in context for later use
			if validKey {
				c.Request().Header.Set("X-Localbase-Role", func() string {
					if isServiceRole {
						return "service_role"
					}
					return "anon"
				}())
			}

			return next(c)
		}
	}
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
