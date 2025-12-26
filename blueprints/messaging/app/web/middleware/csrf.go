package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"sync"
	"time"

	"github.com/go-mizu/mizu"
)

// CSRFConfig holds CSRF protection configuration.
type CSRFConfig struct {
	// TokenLength is the length of CSRF tokens in bytes.
	TokenLength int

	// TokenExpiry is how long tokens are valid.
	TokenExpiry time.Duration

	// CookieName is the name of the CSRF cookie.
	CookieName string

	// HeaderName is the name of the CSRF header.
	HeaderName string

	// FormFieldName is the name of the CSRF form field.
	FormFieldName string

	// Secure sets the Secure flag on cookies.
	Secure bool

	// SameSite sets the SameSite attribute on cookies.
	SameSite http.SameSite
}

// DefaultCSRFConfig returns the default CSRF configuration.
func DefaultCSRFConfig() CSRFConfig {
	return CSRFConfig{
		TokenLength:   32,
		TokenExpiry:   time.Hour,
		CookieName:    "csrf_token",
		HeaderName:    "X-CSRF-Token",
		FormFieldName: "_csrf",
		Secure:        true,
		SameSite:      http.SameSiteStrictMode,
	}
}

// CSRFStore manages CSRF tokens.
type CSRFStore struct {
	mu      sync.RWMutex
	tokens  map[string]time.Time
	config  CSRFConfig
	stopCh  chan struct{}
}

// NewCSRFStore creates a new CSRF token store.
func NewCSRFStore(config CSRFConfig) *CSRFStore {
	s := &CSRFStore{
		tokens: make(map[string]time.Time),
		config: config,
		stopCh: make(chan struct{}),
	}
	go s.cleanup()
	return s
}

// Generate creates a new CSRF token.
func (s *CSRFStore) Generate() (string, error) {
	b := make([]byte, s.config.TokenLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	token := base64.URLEncoding.EncodeToString(b)

	s.mu.Lock()
	s.tokens[token] = time.Now().Add(s.config.TokenExpiry)
	s.mu.Unlock()

	return token, nil
}

// Validate checks if a CSRF token is valid.
// Tokens are single-use and deleted after validation.
func (s *CSRFStore) Validate(token string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	expiry, exists := s.tokens[token]
	if !exists {
		return false
	}

	// Delete the token (single-use)
	delete(s.tokens, token)

	return time.Now().Before(expiry)
}

// ValidateWithoutConsume checks if a token is valid without consuming it.
// Use this for read-only operations that should still verify CSRF.
func (s *CSRFStore) ValidateWithoutConsume(token string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	expiry, exists := s.tokens[token]
	if !exists {
		return false
	}

	return time.Now().Before(expiry)
}

// Stop stops the cleanup goroutine.
func (s *CSRFStore) Stop() {
	close(s.stopCh)
}

// cleanup periodically removes expired tokens.
func (s *CSRFStore) cleanup() {
	ticker := time.NewTicker(s.config.TokenExpiry / 2)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()
			for token, expiry := range s.tokens {
				if now.After(expiry) {
					delete(s.tokens, token)
				}
			}
			s.mu.Unlock()
		}
	}
}

// CSRF returns middleware that protects against CSRF attacks.
// It generates tokens for GET requests and validates them for state-changing methods.
func CSRF(store *CSRFStore) mizu.Middleware {
	cfg := store.config

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			method := c.Request().Method

			// For safe methods, generate and set a new token
			if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
				token, err := store.Generate()
				if err != nil {
					return err
				}

				// Set CSRF token cookie
				http.SetCookie(c.Writer(), &http.Cookie{
					Name:     cfg.CookieName,
					Value:    token,
					Path:     "/",
					HttpOnly: false, // Must be accessible to JavaScript
					Secure:   cfg.Secure,
					SameSite: cfg.SameSite,
					MaxAge:   int(cfg.TokenExpiry.Seconds()),
				})

				// Also set in response header for easy access
				c.Writer().Header().Set(cfg.HeaderName, token)

				return next(c)
			}

			// For state-changing methods, validate the token
			// Skip CSRF validation for API endpoints using Bearer token auth
			if c.Request().Header.Get("Authorization") != "" {
				return next(c)
			}

			// Get token from header or form
			token := c.Request().Header.Get(cfg.HeaderName)
			if token == "" {
				token = c.Request().FormValue(cfg.FormFieldName)
			}

			if token == "" {
				c.Writer().WriteHeader(http.StatusForbidden)
				return c.JSON(http.StatusForbidden, map[string]any{
					"success": false,
					"error":   "CSRF token required",
				})
			}

			if !store.Validate(token) {
				c.Writer().WriteHeader(http.StatusForbidden)
				return c.JSON(http.StatusForbidden, map[string]any{
					"success": false,
					"error":   "Invalid or expired CSRF token",
				})
			}

			return next(c)
		}
	}
}

// CSRFWithConfig returns CSRF middleware with custom configuration.
func CSRFWithConfig(config CSRFConfig) mizu.Middleware {
	store := NewCSRFStore(config)
	return CSRF(store)
}

// SkipCSRF returns middleware that skips CSRF validation for specific paths.
func SkipCSRF(paths []string, csrfMiddleware mizu.Middleware) mizu.Middleware {
	pathSet := make(map[string]bool)
	for _, p := range paths {
		pathSet[p] = true
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			if pathSet[c.Request().URL.Path] {
				return next(c)
			}
			return csrfMiddleware(next)(c)
		}
	}
}
