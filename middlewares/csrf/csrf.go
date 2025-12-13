// Package csrf provides Cross-Site Request Forgery protection middleware for Mizu.
package csrf

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-mizu/mizu"
)

type contextKey struct{}

var (
	// ErrTokenMissing is returned when CSRF token is not found.
	ErrTokenMissing = errors.New("csrf: token missing")

	// ErrTokenInvalid is returned when CSRF token is invalid.
	ErrTokenInvalid = errors.New("csrf: token invalid")
)

// Options configures the CSRF middleware.
type Options struct {
	// Secret is the key used for token generation.
	// Required.
	Secret []byte

	// TokenLength is the length of the random token.
	// Default: 32.
	TokenLength int

	// TokenLookup specifies where to find the token.
	// Format: "source:name" where source is header, form, or query.
	// Default: "header:X-CSRF-Token".
	TokenLookup string

	// CookieName is the name of the CSRF cookie.
	// Default: "_csrf".
	CookieName string

	// CookiePath is the path for the CSRF cookie.
	// Default: "/".
	CookiePath string

	// CookieMaxAge is the max age in seconds.
	// Default: 86400 (24 hours).
	CookieMaxAge int

	// CookieSecure sets the Secure flag.
	CookieSecure bool

	// CookieHTTPOnly sets the HTTPOnly flag.
	// Default: true.
	CookieHTTPOnly bool

	// CookieSameSite sets the SameSite attribute.
	// Default: Lax.
	SameSite http.SameSite

	// ErrorHandler is called when token validation fails.
	ErrorHandler func(c *mizu.Ctx, err error) error

	// SkipPaths are paths to skip CSRF validation.
	SkipPaths []string
}

// New creates a CSRF protection middleware.
//
//nolint:cyclop // CSRF protection requires multiple validation checks
func New(opts Options) mizu.Middleware {
	if len(opts.Secret) == 0 {
		panic("csrf: secret is required")
	}
	if opts.TokenLength == 0 {
		opts.TokenLength = 32
	}
	if opts.TokenLookup == "" {
		opts.TokenLookup = "header:X-CSRF-Token"
	}
	if opts.CookieName == "" {
		opts.CookieName = "_csrf"
	}
	if opts.CookiePath == "" {
		opts.CookiePath = "/"
	}
	if opts.CookieMaxAge == 0 {
		opts.CookieMaxAge = 86400
	}
	if opts.SameSite == 0 {
		opts.SameSite = http.SameSiteLaxMode
	}
	if !opts.CookieHTTPOnly {
		opts.CookieHTTPOnly = true
	}

	// Parse token lookup
	parts := strings.SplitN(opts.TokenLookup, ":", 2)
	if len(parts) != 2 {
		panic("csrf: invalid TokenLookup format")
	}
	lookupSource := parts[0]
	lookupName := parts[1]

	skipPaths := make(map[string]bool)
	for _, p := range opts.SkipPaths {
		skipPaths[p] = true
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			// Skip for safe methods
			method := c.Request().Method
			if method == http.MethodGet || method == http.MethodHead ||
				method == http.MethodOptions || method == http.MethodTrace {
				return handleSafeMethod(c, next, opts)
			}

			// Skip paths
			if skipPaths[c.Request().URL.Path] {
				return next(c)
			}

			// Get token from cookie
			cookie, err := c.Cookie(opts.CookieName)
			if err != nil || cookie.Value == "" {
				return handleError(c, opts, ErrTokenMissing)
			}

			// Get token from request
			var reqToken string
			switch lookupSource {
			case "header":
				reqToken = c.Request().Header.Get(lookupName)
			case "form":
				if form, err := c.Form(); err == nil {
					reqToken = form.Get(lookupName)
				}
			case "query":
				reqToken = c.Query(lookupName)
			}

			if reqToken == "" {
				return handleError(c, opts, ErrTokenMissing)
			}

			// Validate token
			if !validateToken(cookie.Value, reqToken, opts.Secret) {
				return handleError(c, opts, ErrTokenInvalid)
			}

			// Store token in context
			ctx := context.WithValue(c.Context(), contextKey{}, cookie.Value)
			req := c.Request().WithContext(ctx)
			*c.Request() = *req

			return next(c)
		}
	}
}

func handleSafeMethod(c *mizu.Ctx, next mizu.Handler, opts Options) error {
	// Check for existing token
	cookie, err := c.Cookie(opts.CookieName)
	var token string

	if err != nil || cookie.Value == "" {
		// Generate new token
		token = generateToken(opts.TokenLength, opts.Secret)

		// Set cookie
		c.SetCookie(&http.Cookie{
			Name:     opts.CookieName,
			Value:    token,
			Path:     opts.CookiePath,
			MaxAge:   opts.CookieMaxAge,
			Secure:   opts.CookieSecure,
			HttpOnly: opts.CookieHTTPOnly,
			SameSite: opts.SameSite,
		})
	} else {
		token = cookie.Value
	}

	// Store token in context
	ctx := context.WithValue(c.Context(), contextKey{}, token)
	req := c.Request().WithContext(ctx)
	*c.Request() = *req

	return next(c)
}

func handleError(c *mizu.Ctx, opts Options, err error) error {
	if opts.ErrorHandler != nil {
		return opts.ErrorHandler(c, err)
	}
	return c.Text(http.StatusForbidden, err.Error())
}

// Token extracts the CSRF token from context.
func Token(c *mizu.Ctx) string {
	if token, ok := c.Context().Value(contextKey{}).(string); ok {
		return token
	}
	return ""
}

// generateToken creates a new CSRF token.
func generateToken(length int, secret []byte) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}

	// Create HMAC signature
	h := hmac.New(sha256.New, secret)
	h.Write(b)
	sig := h.Sum(nil)

	// Encode token and signature
	token := base64.RawURLEncoding.EncodeToString(b)
	signature := base64.RawURLEncoding.EncodeToString(sig)

	return token + "." + signature
}

// validateToken verifies the CSRF token.
func validateToken(cookieToken, requestToken string, secret []byte) bool {
	// Tokens must match
	if subtle.ConstantTimeCompare([]byte(cookieToken), []byte(requestToken)) != 1 {
		return false
	}

	// Verify signature
	parts := strings.SplitN(cookieToken, ".", 2)
	if len(parts) != 2 {
		return false
	}

	tokenBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}

	sigBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}

	h := hmac.New(sha256.New, secret)
	h.Write(tokenBytes)
	expectedSig := h.Sum(nil)

	return hmac.Equal(sigBytes, expectedSig)
}

// TemplateField returns an HTML hidden input field with the CSRF token.
func TemplateField(c *mizu.Ctx) string {
	return `<input type="hidden" name="_csrf" value="` + Token(c) + `">`
}

// Protect is an alias for New with common options.
func Protect(secret []byte) mizu.Middleware {
	return New(Options{
		Secret:       secret,
		CookieSecure: true,
	})
}

// ProtectDev creates CSRF middleware suitable for development (insecure cookies).
func ProtectDev(secret []byte) mizu.Middleware {
	return New(Options{
		Secret:       secret,
		CookieSecure: false,
	})
}

// GenerateSecret generates a secure random secret.
func GenerateSecret() []byte {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return b
}

// TokenExpiry returns the token expiry time based on cookie max age.
func TokenExpiry(opts Options) time.Time {
	if opts.CookieMaxAge == 0 {
		opts.CookieMaxAge = 86400
	}
	return time.Now().Add(time.Duration(opts.CookieMaxAge) * time.Second)
}
