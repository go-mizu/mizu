// Package captcha provides CAPTCHA verification middleware for Mizu.
package captcha

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-mizu/mizu"
)

// Provider represents a CAPTCHA provider.
type Provider string

const (
	// ProviderRecaptchaV2 is Google reCAPTCHA v2.
	ProviderRecaptchaV2 Provider = "recaptcha_v2"
	// ProviderRecaptchaV3 is Google reCAPTCHA v3.
	ProviderRecaptchaV3 Provider = "recaptcha_v3"
	// ProviderHCaptcha is hCaptcha.
	ProviderHCaptcha Provider = "hcaptcha"
	// ProviderTurnstile is Cloudflare Turnstile.
	ProviderTurnstile Provider = "turnstile"
	// ProviderCustom allows custom verification.
	ProviderCustom Provider = "custom"
)

// Options configures the captcha middleware.
type Options struct {
	// Provider is the CAPTCHA provider.
	// Default: ProviderRecaptchaV2.
	Provider Provider

	// Secret is the secret key for verification.
	Secret string

	// TokenLookup specifies where to find the token.
	// Default: "form:g-recaptcha-response".
	TokenLookup string

	// MinScore is the minimum score for v3 providers.
	// Default: 0.5.
	MinScore float64

	// Verifier is a custom verification function.
	Verifier func(token string, c *mizu.Ctx) (bool, error)

	// ErrorHandler handles verification failures.
	ErrorHandler func(c *mizu.Ctx, err error) error

	// SkipPaths are paths to skip verification.
	SkipPaths []string

	// Timeout for verification request.
	// Default: 10s.
	Timeout time.Duration
}

// Verification URLs
var verifyURLs = map[Provider]string{
	ProviderRecaptchaV2: "https://www.google.com/recaptcha/api/siteverify",
	ProviderRecaptchaV3: "https://www.google.com/recaptcha/api/siteverify",
	ProviderHCaptcha:    "https://hcaptcha.com/siteverify",
	ProviderTurnstile:   "https://challenges.cloudflare.com/turnstile/v0/siteverify",
}

// New creates captcha middleware with default options.
func New(secret string) mizu.Middleware {
	return WithOptions(Options{Secret: secret})
}

// WithOptions creates captcha middleware with custom options.
//
//nolint:cyclop // CAPTCHA verification requires multiple provider and option checks
func WithOptions(opts Options) mizu.Middleware {
	if opts.Provider == "" {
		opts.Provider = ProviderRecaptchaV2
	}
	if opts.TokenLookup == "" {
		opts.TokenLookup = "form:g-recaptcha-response"
	}
	if opts.MinScore == 0 {
		opts.MinScore = 0.5
	}
	if opts.Timeout == 0 {
		opts.Timeout = 10 * time.Second
	}

	skipPaths := make(map[string]bool)
	for _, path := range opts.SkipPaths {
		skipPaths[path] = true
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			// Skip configured paths
			if skipPaths[c.Request().URL.Path] {
				return next(c)
			}

			// Only check unsafe methods
			method := c.Request().Method
			if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
				return next(c)
			}

			// Extract token
			token := extractToken(c, opts.TokenLookup)
			if token == "" {
				return handleError(c, opts, ErrMissingToken)
			}

			// Verify token
			var valid bool
			var err error

			if opts.Verifier != nil {
				valid, err = opts.Verifier(token, c)
			} else {
				valid, err = verifyToken(opts, token, c)
			}

			if err != nil {
				return handleError(c, opts, err)
			}

			if !valid {
				return handleError(c, opts, ErrInvalidToken)
			}

			return next(c)
		}
	}
}

// Error types
type captchaError string

func (e captchaError) Error() string { return string(e) }

//nolint:gosec // G101: False positive - these are error message constants, not credentials
const (
	ErrMissingToken captchaError = "captcha token missing"
	ErrInvalidToken captchaError = "captcha verification failed"
	ErrVerifyFailed captchaError = "captcha verification request failed"
)

func extractToken(c *mizu.Ctx, lookup string) string {
	parts := strings.SplitN(lookup, ":", 2)
	if len(parts) != 2 {
		return ""
	}

	source, key := parts[0], parts[1]

	switch source {
	case "form":
		return c.Request().FormValue(key)
	case "header":
		return c.Request().Header.Get(key)
	case "query":
		return c.Request().URL.Query().Get(key)
	}

	return ""
}

func verifyToken(opts Options, token string, c *mizu.Ctx) (bool, error) {
	verifyURL, ok := verifyURLs[opts.Provider]
	if !ok {
		return false, ErrVerifyFailed
	}

	// Prepare form data
	data := url.Values{}
	data.Set("secret", opts.Secret)
	data.Set("response", token)

	// Add remote IP
	ip := getClientIP(c)
	if ip != "" {
		data.Set("remoteip", ip)
	}

	// Create client with timeout
	client := &http.Client{Timeout: opts.Timeout}

	// Make verification request
	resp, err := client.PostForm(verifyURL, data)
	if err != nil {
		return false, ErrVerifyFailed
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, ErrVerifyFailed
	}

	// Parse response
	var result struct {
		Success     bool     `json:"success"`
		Score       float64  `json:"score"`
		Action      string   `json:"action"`
		ChallengeTS string   `json:"challenge_ts"`
		Hostname    string   `json:"hostname"`
		ErrorCodes  []string `json:"error-codes"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return false, ErrVerifyFailed
	}

	if !result.Success {
		return false, nil
	}

	// Check score for v3 providers
	if opts.Provider == ProviderRecaptchaV3 || opts.Provider == ProviderTurnstile {
		if result.Score < opts.MinScore {
			return false, nil
		}
	}

	return true, nil
}

func handleError(c *mizu.Ctx, opts Options, err error) error {
	if opts.ErrorHandler != nil {
		return opts.ErrorHandler(c, err)
	}
	return c.Text(http.StatusBadRequest, err.Error())
}

func getClientIP(c *mizu.Ctx) string {
	if xff := c.Request().Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xri := c.Request().Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	addr := c.Request().RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		return addr[:idx]
	}
	return addr
}

// ReCaptchaV2 creates middleware for Google reCAPTCHA v2.
func ReCaptchaV2(secret string) mizu.Middleware {
	return WithOptions(Options{
		Provider: ProviderRecaptchaV2,
		Secret:   secret,
	})
}

// ReCaptchaV3 creates middleware for Google reCAPTCHA v3.
func ReCaptchaV3(secret string, minScore float64) mizu.Middleware {
	return WithOptions(Options{
		Provider: ProviderRecaptchaV3,
		Secret:   secret,
		MinScore: minScore,
	})
}

// HCaptcha creates middleware for hCaptcha.
func HCaptcha(secret string) mizu.Middleware {
	return WithOptions(Options{
		Provider:    ProviderHCaptcha,
		Secret:      secret,
		TokenLookup: "form:h-captcha-response",
	})
}

// Turnstile creates middleware for Cloudflare Turnstile.
func Turnstile(secret string) mizu.Middleware {
	return WithOptions(Options{
		Provider:    ProviderTurnstile,
		Secret:      secret,
		TokenLookup: "form:cf-turnstile-response",
	})
}

// Custom creates middleware with a custom verifier.
func Custom(verifier func(token string, c *mizu.Ctx) (bool, error), tokenLookup string) mizu.Middleware {
	return WithOptions(Options{
		Provider:    ProviderCustom,
		Verifier:    verifier,
		TokenLookup: tokenLookup,
	})
}
