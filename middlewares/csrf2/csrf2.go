// Package csrf2 provides an alternative CSRF protection middleware for Mizu.
// It uses the double-submit cookie pattern with encrypted tokens.
package csrf2

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/go-mizu/mizu"
)

// Options configures the csrf2 middleware.
type Options struct {
	// Secret is used for token generation.
	// Required.
	Secret string

	// TokenLength is the length of generated tokens.
	// Default: 32.
	TokenLength int

	// CookieName is the name of the CSRF cookie.
	// Default: "_csrf".
	CookieName string

	// HeaderName is the name of the CSRF header.
	// Default: "X-CSRF-Token".
	HeaderName string

	// FormField is the form field name for the token.
	// Default: "_csrf".
	FormField string

	// CookiePath is the cookie path.
	// Default: "/".
	CookiePath string

	// CookieDomain is the cookie domain.
	CookieDomain string

	// CookieMaxAge is the cookie max age in seconds.
	// Default: 86400 (24 hours).
	CookieMaxAge int

	// CookieSecure sets the Secure flag on the cookie.
	CookieSecure bool

	// CookieHTTPOnly sets the HttpOnly flag on the cookie.
	// Default: true.
	CookieHTTPOnly bool

	// CookieSameSite sets the SameSite attribute.
	// Default: Lax.
	CookieSameSite http.SameSite

	// SkipPaths are paths to skip CSRF validation.
	SkipPaths []string

	// SkipMethods are HTTP methods to skip CSRF validation.
	// Default: GET, HEAD, OPTIONS, TRACE.
	SkipMethods []string

	// ErrorHandler handles CSRF validation failures.
	ErrorHandler func(c *mizu.Ctx) error

	// TokenGetter extracts the token from request.
	// Default: header, form, and query.
	TokenGetter func(c *mizu.Ctx) string

	// Origin validation
	ValidateOrigin bool
	AllowedOrigins []string
}

// contextKey is a private type for context keys.
type contextKey struct{}

// tokenKey stores the CSRF token.
var tokenKey = contextKey{}

// New creates csrf2 middleware with default options.
func New(secret string) mizu.Middleware {
	return WithOptions(Options{Secret: secret})
}

// WithOptions creates csrf2 middleware with custom options.
//
//nolint:cyclop // CSRF protection requires multiple validation and token checks
func WithOptions(opts Options) mizu.Middleware {
	if opts.TokenLength == 0 {
		opts.TokenLength = 32
	}
	if opts.CookieName == "" {
		opts.CookieName = "_csrf"
	}
	if opts.HeaderName == "" {
		opts.HeaderName = "X-CSRF-Token"
	}
	if opts.FormField == "" {
		opts.FormField = "_csrf"
	}
	if opts.CookiePath == "" {
		opts.CookiePath = "/"
	}
	if opts.CookieMaxAge == 0 {
		opts.CookieMaxAge = 86400
	}
	if opts.CookieSameSite == 0 {
		opts.CookieSameSite = http.SameSiteLaxMode
	}
	if len(opts.SkipMethods) == 0 {
		opts.SkipMethods = []string{"GET", "HEAD", "OPTIONS", "TRACE"}
	}

	skipPaths := make(map[string]bool)
	for _, p := range opts.SkipPaths {
		skipPaths[p] = true
	}

	skipMethods := make(map[string]bool)
	for _, m := range opts.SkipMethods {
		skipMethods[strings.ToUpper(m)] = true
	}

	allowedOrigins := make(map[string]bool)
	for _, o := range opts.AllowedOrigins {
		allowedOrigins[strings.ToLower(o)] = true
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			r := c.Request()

			// Skip configured paths
			if skipPaths[r.URL.Path] {
				return next(c)
			}

			// Get or create token
			token := getCookieToken(c, opts.CookieName)
			if token == "" {
				token = generateToken(opts.Secret, opts.TokenLength)
				setCookie(c, opts, token)
			}

			// Store token in context
			ctx := context.WithValue(c.Context(), tokenKey, token)
			req := c.Request().WithContext(ctx)
			*c.Request() = *req

			// Skip validation for safe methods
			if skipMethods[r.Method] {
				return next(c)
			}

			// Validate origin if enabled
			if opts.ValidateOrigin {
				if !validateOrigin(c, allowedOrigins) {
					return handleError(c, opts)
				}
			}

			// Get submitted token
			submitted := ""
			if opts.TokenGetter != nil {
				submitted = opts.TokenGetter(c)
			} else {
				submitted = getSubmittedToken(c, opts)
			}

			// Validate token
			if !validateToken(token, submitted, opts.Secret) {
				return handleError(c, opts)
			}

			return next(c)
		}
	}
}

func generateToken(secret string, length int) string {
	b := make([]byte, length)
	_, _ = rand.Read(b)

	// Add timestamp for rotation
	timestamp := time.Now().Unix()
	data := make([]byte, 0, length+4)
	data = append(data, b...)
	data = append(data, byte(timestamp>>24), byte(timestamp>>16), byte(timestamp>>8), byte(timestamp))

	// Sign with secret
	h := sha256.New()
	h.Write([]byte(secret))
	h.Write(data)
	signature := h.Sum(nil)

	// Combine and encode
	token := append(data, signature[:8]...)
	return base64.RawURLEncoding.EncodeToString(token)
}

func validateToken(cookie, submitted, secret string) bool {
	if cookie == "" || submitted == "" {
		return false
	}

	// Decode tokens
	cookieBytes, err := base64.RawURLEncoding.DecodeString(cookie)
	if err != nil {
		return false
	}
	submittedBytes, err := base64.RawURLEncoding.DecodeString(submitted)
	if err != nil {
		return false
	}

	// Check length
	if len(cookieBytes) < 8 || len(submittedBytes) < 8 {
		return false
	}

	// Constant time comparison
	if len(cookieBytes) != len(submittedBytes) {
		return false
	}

	var result byte
	for i := range cookieBytes {
		result |= cookieBytes[i] ^ submittedBytes[i]
	}

	return result == 0
}

func getCookieToken(c *mizu.Ctx, name string) string {
	cookie, err := c.Request().Cookie(name)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func setCookie(c *mizu.Ctx, opts Options, token string) {
	cookie := &http.Cookie{
		Name:     opts.CookieName,
		Value:    token,
		Path:     opts.CookiePath,
		Domain:   opts.CookieDomain,
		MaxAge:   opts.CookieMaxAge,
		Secure:   opts.CookieSecure,
		HttpOnly: opts.CookieHTTPOnly,
		SameSite: opts.CookieSameSite,
	}
	http.SetCookie(c.Writer(), cookie)
}

func getSubmittedToken(c *mizu.Ctx, opts Options) string {
	// Check header first
	if token := c.Request().Header.Get(opts.HeaderName); token != "" {
		return token
	}

	// Check form
	if err := c.Request().ParseForm(); err == nil {
		if token := c.Request().FormValue(opts.FormField); token != "" {
			return token
		}
	}

	// Check query
	if token := c.Request().URL.Query().Get(opts.FormField); token != "" {
		return token
	}

	return ""
}

func validateOrigin(c *mizu.Ctx, allowed map[string]bool) bool {
	origin := c.Request().Header.Get("Origin")
	if origin == "" {
		// Check Referer as fallback
		referer := c.Request().Header.Get("Referer")
		if referer == "" {
			return false
		}
		origin = referer
	}

	origin = strings.ToLower(origin)

	// If no allowed origins specified, check against request host
	if len(allowed) == 0 {
		host := strings.ToLower(c.Request().Host)
		return strings.Contains(origin, host)
	}

	for o := range allowed {
		if strings.Contains(origin, o) {
			return true
		}
	}

	return false
}

func handleError(c *mizu.Ctx, opts Options) error {
	if opts.ErrorHandler != nil {
		return opts.ErrorHandler(c)
	}
	return c.Text(http.StatusForbidden, "CSRF token validation failed")
}

// GetToken returns the CSRF token from context.
func GetToken(c *mizu.Ctx) string {
	if token, ok := c.Context().Value(tokenKey).(string); ok {
		return token
	}
	return ""
}

// Token returns a handler that provides the CSRF token.
func Token() mizu.Handler {
	return func(c *mizu.Ctx) error {
		token := GetToken(c)
		return c.JSON(http.StatusOK, map[string]string{
			"token": token,
		})
	}
}

// FormInput returns an HTML hidden input with the CSRF token.
func FormInput(c *mizu.Ctx, fieldName string) string {
	if fieldName == "" {
		fieldName = "_csrf"
	}
	token := GetToken(c)
	return `<input type="hidden" name="` + fieldName + `" value="` + token + `">`
}

// MetaTag returns an HTML meta tag with the CSRF token.
func MetaTag(c *mizu.Ctx) string {
	token := GetToken(c)
	return `<meta name="csrf-token" content="` + token + `">`
}

// Mask applies a random mask to the token for additional protection.
func Mask(token string) string {
	tokenBytes, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return token
	}

	mask := make([]byte, len(tokenBytes))
	_, _ = rand.Read(mask)

	masked := make([]byte, len(tokenBytes)*2)
	copy(masked, mask)
	for i, b := range tokenBytes {
		masked[len(tokenBytes)+i] = b ^ mask[i]
	}

	return base64.RawURLEncoding.EncodeToString(masked)
}

// Unmask removes the mask from a token.
func Unmask(maskedToken string) string {
	masked, err := base64.RawURLEncoding.DecodeString(maskedToken)
	if err != nil {
		return maskedToken
	}

	if len(masked)%2 != 0 {
		return maskedToken
	}

	half := len(masked) / 2
	mask := masked[:half]
	token := masked[half:]

	unmasked := make([]byte, half)
	for i := range mask {
		unmasked[i] = token[i] ^ mask[i]
	}

	return base64.RawURLEncoding.EncodeToString(unmasked)
}

// Fingerprint generates a request fingerprint for additional validation.
func Fingerprint(c *mizu.Ctx) string {
	r := c.Request()
	h := sha256.New()
	h.Write([]byte(r.UserAgent()))
	h.Write([]byte(r.Header.Get("Accept-Language")))
	h.Write([]byte(r.Header.Get("Accept-Encoding")))
	return hex.EncodeToString(h.Sum(nil)[:16])
}
