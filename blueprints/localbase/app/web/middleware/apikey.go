package middleware

import (
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/go-mizu/mizu"
	"github.com/golang-jwt/jwt/v5"
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
	// ValidateSignature enables JWT signature validation (default: true in production)
	ValidateSignature bool
}

// JWTClaims represents the full claims from a Supabase-compatible JWT.
// This includes both API key JWTs (anon/service_role) and user JWTs.
type JWTClaims struct {
	// Standard claims
	Sub string `json:"sub"` // User ID (for user JWTs)
	Aud string `json:"aud"` // Audience
	Iss string `json:"iss"` // Issuer
	Exp int64  `json:"exp"` // Expiration
	Iat int64  `json:"iat"` // Issued at

	// Supabase-specific claims
	Role         string         `json:"role"`          // anon, authenticated, service_role
	Email        string         `json:"email"`         // User email
	Phone        string         `json:"phone"`         // User phone
	AppMetadata  map[string]any `json:"app_metadata"`  // App metadata
	UserMetadata map[string]any `json:"user_metadata"` // User metadata
	AAL          string         `json:"aal"`           // Authentication Assurance Level
	SessionID    string         `json:"session_id"`    // Session ID
	IsAnonymous  bool           `json:"is_anonymous"`  // Anonymous user flag

	// Raw claims for full access
	Raw map[string]any `json:"-"`
}

// Header names for storing JWT information
const (
	HeaderRole      = "X-Localbase-Role"
	HeaderJWTClaims = "X-Localbase-JWT-Claims"
	HeaderUserID    = "X-Localbase-User-ID"
	HeaderUserEmail = "X-Localbase-User-Email"
)

// DefaultAPIKeyConfig returns the default API key configuration.
// Uses the same default keys as Supabase local development.
func DefaultAPIKeyConfig() *APIKeyConfig {
	// Supabase local development default keys
	defaultAnonKey := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZS1kZW1vIiwicm9sZSI6ImFub24iLCJleHAiOjE5ODM4MTI5OTZ9.CRXP1A7WOeoJeXxjNni43kdQwgnWNReilDMblYTn_I0"
	defaultServiceKey := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZS1kZW1vIiwicm9sZSI6InNlcnZpY2Vfcm9sZSIsImV4cCI6MTk4MzgxMjk5Nn0.EGIM96RAZx35lJzdJsyH-qQwv8Hdp7fsn3W0YpN81IU"
	defaultJWTSecret := "super-secret-jwt-token-with-at-least-32-characters-long"

	// Enable signature validation by default unless explicitly disabled
	validateSig := getEnv("LOCALBASE_VALIDATE_JWT", "true") == "true"

	return &APIKeyConfig{
		AnonKey:           getEnv("LOCALBASE_ANON_KEY", defaultAnonKey),
		ServiceKey:        getEnv("LOCALBASE_SERVICE_KEY", defaultServiceKey),
		JWTSecret:         getEnv("LOCALBASE_JWT_SECRET", defaultJWTSecret),
		HeaderName:        "apikey",
		Optional:          true, // Make optional for backward compatibility
		ValidateSignature: validateSig,
	}
}

// APIKey returns a middleware that validates API keys and JWTs.
// It accepts Supabase-compatible JWT tokens and extracts the role.
// When ValidateSignature is enabled, it validates the JWT signature and expiration.
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
				c.Request().Header.Set(HeaderRole, "anon")
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

			// Parse and validate JWT
			var claims *JWTClaims
			var err error

			if config.ValidateSignature {
				claims, err = validateAndParseJWT(apiKey, config.JWTSecret)
				if err != nil {
					// Check if it's a known key (for backward compatibility)
					if apiKey == config.AnonKey || apiKey == config.ServiceKey || apiKey == "test-api-key" {
						// Try parsing without validation for known keys
						claims, _ = parseJWTClaims(apiKey)
					} else if !config.Optional {
						return c.JSON(401, map[string]any{
							"statusCode": 401,
							"error":      "Unauthorized",
							"message":    "Invalid JWT: " + err.Error(),
						})
					}
				}
			} else {
				claims, _ = parseJWTClaims(apiKey)
			}

			// Determine role
			var role string
			if claims != nil {
				role = claims.Role
			}

			// Fallback: check if it matches known keys directly
			if role == "" {
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
			c.Request().Header.Set(HeaderRole, role)

			// Store claims as JSON in header for database RLS use
			if claims != nil {
				if claimsJSON, err := json.Marshal(claims.Raw); err == nil {
					c.Request().Header.Set(HeaderJWTClaims, string(claimsJSON))
				}
				// Store sub (user ID) separately for easy access
				if claims.Sub != "" {
					c.Request().Header.Set(HeaderUserID, claims.Sub)
				}
				// Store email if present
				if claims.Email != "" {
					c.Request().Header.Set(HeaderUserEmail, claims.Email)
				}
			}

			return next(c)
		}
	}
}

// validateAndParseJWT validates the JWT signature and parses claims.
func validateAndParseJWT(tokenString, secret string) (*JWTClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, jwt.ErrSignatureInvalid
	}

	mapClaims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, jwt.ErrTokenInvalidClaims
	}

	// Check expiration
	if exp, ok := mapClaims["exp"].(float64); ok {
		if time.Now().Unix() > int64(exp) {
			return nil, jwt.ErrTokenExpired
		}
	}

	return mapClaimsToJWTClaims(mapClaims), nil
}

// parseJWTClaims parses JWT claims without validating the signature.
// Used for backward compatibility and known keys.
func parseJWTClaims(tokenString string) (*JWTClaims, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, jwt.ErrTokenMalformed
	}

	// Parse without validation
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	token, _, err := parser.ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return nil, err
	}

	mapClaims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, jwt.ErrTokenInvalidClaims
	}

	return mapClaimsToJWTClaims(mapClaims), nil
}

// mapClaimsToJWTClaims converts jwt.MapClaims to our JWTClaims struct.
func mapClaimsToJWTClaims(mapClaims jwt.MapClaims) *JWTClaims {
	claims := &JWTClaims{
		Raw: make(map[string]any),
	}

	// Copy all claims to Raw
	for k, v := range mapClaims {
		claims.Raw[k] = v
	}

	// Extract known fields
	if v, ok := mapClaims["sub"].(string); ok {
		claims.Sub = v
	}
	if v, ok := mapClaims["aud"].(string); ok {
		claims.Aud = v
	}
	if v, ok := mapClaims["iss"].(string); ok {
		claims.Iss = v
	}
	if v, ok := mapClaims["exp"].(float64); ok {
		claims.Exp = int64(v)
	}
	if v, ok := mapClaims["iat"].(float64); ok {
		claims.Iat = int64(v)
	}
	if v, ok := mapClaims["role"].(string); ok {
		claims.Role = v
	}
	if v, ok := mapClaims["email"].(string); ok {
		claims.Email = v
	}
	if v, ok := mapClaims["phone"].(string); ok {
		claims.Phone = v
	}
	if v, ok := mapClaims["app_metadata"].(map[string]any); ok {
		claims.AppMetadata = v
	}
	if v, ok := mapClaims["user_metadata"].(map[string]any); ok {
		claims.UserMetadata = v
	}
	if v, ok := mapClaims["aal"].(string); ok {
		claims.AAL = v
	}
	if v, ok := mapClaims["session_id"].(string); ok {
		claims.SessionID = v
	}
	if v, ok := mapClaims["is_anonymous"].(bool); ok {
		claims.IsAnonymous = v
	}

	return claims
}

// GetRole extracts the role from the request headers.
func GetRole(c *mizu.Ctx) string {
	role := c.Request().Header.Get(HeaderRole)
	if role == "" {
		return "anon"
	}
	return role
}

// IsServiceRole checks if the current request has service_role privileges.
func IsServiceRole(c *mizu.Ctx) bool {
	return GetRole(c) == "service_role"
}

// GetUserID extracts the user ID (sub claim) from the request headers.
func GetUserID(c *mizu.Ctx) string {
	return c.Request().Header.Get(HeaderUserID)
}

// GetUserEmail extracts the user email from the request headers.
func GetUserEmail(c *mizu.Ctx) string {
	return c.Request().Header.Get(HeaderUserEmail)
}

// GetJWTClaims extracts the full JWT claims from the request headers.
// Returns nil if no claims are present.
func GetJWTClaims(c *mizu.Ctx) *JWTClaims {
	claimsJSON := c.Request().Header.Get(HeaderJWTClaims)
	if claimsJSON == "" {
		return nil
	}

	var raw map[string]any
	if err := json.Unmarshal([]byte(claimsJSON), &raw); err != nil {
		return nil
	}

	// Convert raw map to JWTClaims
	claims := &JWTClaims{Raw: raw}
	if v, ok := raw["sub"].(string); ok {
		claims.Sub = v
	}
	if v, ok := raw["aud"].(string); ok {
		claims.Aud = v
	}
	if v, ok := raw["iss"].(string); ok {
		claims.Iss = v
	}
	if v, ok := raw["exp"].(float64); ok {
		claims.Exp = int64(v)
	}
	if v, ok := raw["iat"].(float64); ok {
		claims.Iat = int64(v)
	}
	if v, ok := raw["role"].(string); ok {
		claims.Role = v
	}
	if v, ok := raw["email"].(string); ok {
		claims.Email = v
	}
	if v, ok := raw["phone"].(string); ok {
		claims.Phone = v
	}
	if v, ok := raw["app_metadata"].(map[string]any); ok {
		claims.AppMetadata = v
	}
	if v, ok := raw["user_metadata"].(map[string]any); ok {
		claims.UserMetadata = v
	}
	if v, ok := raw["aal"].(string); ok {
		claims.AAL = v
	}
	if v, ok := raw["session_id"].(string); ok {
		claims.SessionID = v
	}
	if v, ok := raw["is_anonymous"].(bool); ok {
		claims.IsAnonymous = v
	}

	return claims
}

// GetJWTClaimsJSON returns the raw JWT claims as JSON string.
// Returns empty string if no claims are present.
func GetJWTClaimsJSON(c *mizu.Ctx) string {
	return c.Request().Header.Get(HeaderJWTClaims)
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

// RequireServiceRole returns a middleware that requires service_role for access.
// This is used to protect admin endpoints.
func RequireServiceRole() mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			if GetRole(c) != "service_role" {
				return c.JSON(403, map[string]any{
					"statusCode": 403,
					"error":      "Forbidden",
					"message":    "service_role required for admin endpoints",
				})
			}
			return next(c)
		}
	}
}

// RequireAuthenticated returns a middleware that requires authentication.
// Accepts authenticated users and service_role.
func RequireAuthenticated() mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			role := GetRole(c)
			if role == "anon" {
				return c.JSON(401, map[string]any{
					"statusCode": 401,
					"error":      "Unauthorized",
					"message":    "authentication required",
				})
			}
			return next(c)
		}
	}
}
