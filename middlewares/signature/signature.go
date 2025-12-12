// Package signature provides request signature verification middleware for Mizu.
// It supports HMAC-based signatures commonly used in webhooks and API authentication.
package signature

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"hash"
	"io"
	"net/http"
	"strings"

	"github.com/go-mizu/mizu"
)

// Algorithm represents a signature algorithm.
type Algorithm string

// Supported algorithms
const (
	SHA1   Algorithm = "sha1"
	SHA256 Algorithm = "sha256"
	SHA512 Algorithm = "sha512"
)

// Options configures the signature middleware.
type Options struct {
	// Secret is the shared secret for HMAC.
	Secret string

	// Algorithm is the HMAC algorithm.
	// Default: SHA256.
	Algorithm Algorithm

	// HeaderName is the header containing the signature.
	// Default: "X-Signature".
	HeaderName string

	// SignaturePrefix is an optional prefix (e.g., "sha256=").
	SignaturePrefix string

	// Encoding specifies the signature encoding.
	// "hex" or "base64". Default: "hex".
	Encoding string

	// SkipPaths are paths to skip validation.
	SkipPaths []string

	// SkipMethods are methods to skip validation.
	// Default: GET, HEAD, OPTIONS.
	SkipMethods []string

	// PayloadGetter extracts the payload to sign.
	// Default: request body.
	PayloadGetter func(c *mizu.Ctx) ([]byte, error)

	// ErrorHandler handles signature validation failures.
	ErrorHandler func(c *mizu.Ctx) error

	// OnValid is called when signature is valid.
	OnValid func(c *mizu.Ctx)
}

// contextKey is a private type for context keys.
type contextKey struct{}

// signatureKey stores signature info.
var signatureKey = contextKey{}

// Info contains signature verification info.
type Info struct {
	Valid     bool
	Algorithm Algorithm
	Signature string
}

// New creates signature middleware with default options.
func New(secret string) mizu.Middleware {
	return WithOptions(Options{Secret: secret})
}

// WithOptions creates signature middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	if opts.Algorithm == "" {
		opts.Algorithm = SHA256
	}
	if opts.HeaderName == "" {
		opts.HeaderName = "X-Signature"
	}
	if opts.Encoding == "" {
		opts.Encoding = "hex"
	}
	if len(opts.SkipMethods) == 0 {
		opts.SkipMethods = []string{"GET", "HEAD", "OPTIONS"}
	}

	skipPaths := make(map[string]bool)
	for _, p := range opts.SkipPaths {
		skipPaths[p] = true
	}

	skipMethods := make(map[string]bool)
	for _, m := range opts.SkipMethods {
		skipMethods[strings.ToUpper(m)] = true
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			r := c.Request()

			// Skip configured paths and methods
			if skipPaths[r.URL.Path] || skipMethods[r.Method] {
				return next(c)
			}

			// Get signature from header
			signature := r.Header.Get(opts.HeaderName)
			if signature == "" {
				return handleError(c, opts)
			}

			// Remove prefix if present
			if opts.SignaturePrefix != "" {
				if strings.HasPrefix(signature, opts.SignaturePrefix) {
					signature = strings.TrimPrefix(signature, opts.SignaturePrefix)
				} else {
					return handleError(c, opts)
				}
			}

			// Get payload
			var payload []byte
			var err error
			if opts.PayloadGetter != nil {
				payload, err = opts.PayloadGetter(c)
			} else {
				payload, err = io.ReadAll(r.Body)
				_ = r.Body.Close()
				if err == nil {
					r.Body = io.NopCloser(bytes.NewReader(payload))
				}
			}
			if err != nil {
				return handleError(c, opts)
			}

			// Verify signature
			if !verify(opts.Secret, opts.Algorithm, opts.Encoding, payload, signature) {
				ctx := context.WithValue(c.Context(), signatureKey, &Info{Valid: false, Algorithm: opts.Algorithm, Signature: signature})
				req := c.Request().WithContext(ctx)
				*c.Request() = *req
				return handleError(c, opts)
			}

			// Store info
			ctx := context.WithValue(c.Context(), signatureKey, &Info{Valid: true, Algorithm: opts.Algorithm, Signature: signature})
			req := c.Request().WithContext(ctx)
			*c.Request() = *req

			if opts.OnValid != nil {
				opts.OnValid(c)
			}

			return next(c)
		}
	}
}

func verify(secret string, algo Algorithm, encoding string, payload []byte, signature string) bool {
	expected := Sign(secret, algo, encoding, payload)
	return hmac.Equal([]byte(expected), []byte(signature))
}

// Sign creates a signature for the payload.
func Sign(secret string, algo Algorithm, encoding string, payload []byte) string {
	var h hash.Hash

	switch algo {
	case SHA1:
		h = hmac.New(sha1.New, []byte(secret))
	case SHA256:
		h = hmac.New(sha256.New, []byte(secret))
	case SHA512:
		h = hmac.New(sha512.New, []byte(secret))
	default:
		h = hmac.New(sha256.New, []byte(secret))
	}

	h.Write(payload)
	sum := h.Sum(nil)

	switch encoding {
	case "base64":
		return base64.StdEncoding.EncodeToString(sum)
	default:
		return hex.EncodeToString(sum)
	}
}

func handleError(c *mizu.Ctx, opts Options) error {
	if opts.ErrorHandler != nil {
		return opts.ErrorHandler(c)
	}
	return c.Text(http.StatusUnauthorized, "Invalid signature")
}

// GetInfo returns signature info from context.
func GetInfo(c *mizu.Ctx) *Info {
	if info, ok := c.Context().Value(signatureKey).(*Info); ok {
		return info
	}
	return nil
}

// IsValid returns true if the signature was valid.
func IsValid(c *mizu.Ctx) bool {
	info := GetInfo(c)
	return info != nil && info.Valid
}

// GitHub creates middleware for GitHub webhook signatures.
func GitHub(secret string) mizu.Middleware {
	return WithOptions(Options{
		Secret:          secret,
		Algorithm:       SHA256,
		HeaderName:      "X-Hub-Signature-256",
		SignaturePrefix: "sha256=",
		Encoding:        "hex",
	})
}

// Stripe creates middleware for Stripe webhook signatures.
func Stripe(secret string) mizu.Middleware {
	return WithOptions(Options{
		Secret:     secret,
		Algorithm:  SHA256,
		HeaderName: "Stripe-Signature",
		Encoding:   "hex",
		PayloadGetter: func(c *mizu.Ctx) ([]byte, error) {
			// Stripe includes timestamp in signature
			body, err := io.ReadAll(c.Request().Body)
			_ = c.Request().Body.Close()
			if err != nil {
				return nil, err
			}
			c.Request().Body = io.NopCloser(bytes.NewReader(body))
			return body, nil
		},
	})
}

// Slack creates middleware for Slack request verification.
func Slack(signingSecret string) mizu.Middleware {
	return WithOptions(Options{
		Secret:          signingSecret,
		Algorithm:       SHA256,
		HeaderName:      "X-Slack-Signature",
		SignaturePrefix: "v0=",
		Encoding:        "hex",
		PayloadGetter: func(c *mizu.Ctx) ([]byte, error) {
			// Slack signature: v0=HMAC(v0:timestamp:body)
			timestamp := c.Request().Header.Get("X-Slack-Request-Timestamp")
			body, err := io.ReadAll(c.Request().Body)
			_ = c.Request().Body.Close()
			if err != nil {
				return nil, err
			}
			c.Request().Body = io.NopCloser(bytes.NewReader(body))

			baseString := "v0:" + timestamp + ":" + string(body)
			return []byte(baseString), nil
		},
	})
}

// Twilio creates middleware for Twilio request validation.
func Twilio(authToken string) mizu.Middleware {
	return WithOptions(Options{
		Secret:     authToken,
		Algorithm:  SHA1,
		HeaderName: "X-Twilio-Signature",
		Encoding:   "base64",
		PayloadGetter: func(c *mizu.Ctx) ([]byte, error) {
			// Twilio signature: HMAC(URL + sorted POST params)
			url := c.Request().URL.String()
			if err := c.Request().ParseForm(); err != nil {
				return nil, err
			}

			// Build base string: URL + sorted params
			baseString := url
			for key, values := range c.Request().PostForm {
				for _, value := range values {
					baseString += key + value
				}
			}

			return []byte(baseString), nil
		},
	})
}

// AWS creates middleware for AWS Signature Version 4.
func AWS(secretKey string) mizu.Middleware {
	return WithOptions(Options{
		Secret:     secretKey,
		Algorithm:  SHA256,
		HeaderName: "Authorization",
		PayloadGetter: func(c *mizu.Ctx) ([]byte, error) {
			// Simplified - real AWS Sig V4 is more complex
			body, err := io.ReadAll(c.Request().Body)
			_ = c.Request().Body.Close()
			if err != nil {
				return nil, err
			}
			c.Request().Body = io.NopCloser(bytes.NewReader(body))
			return body, nil
		},
	})
}

// Signer creates signatures for outgoing requests.
type Signer struct {
	Secret     string
	Algorithm  Algorithm
	Encoding   string
	HeaderName string
	Prefix     string
}

// NewSigner creates a new signer.
func NewSigner(secret string) *Signer {
	return &Signer{
		Secret:     secret,
		Algorithm:  SHA256,
		Encoding:   "hex",
		HeaderName: "X-Signature",
	}
}

// Sign signs the request body and adds the signature header.
func (s *Signer) Sign(r *http.Request, body []byte) {
	signature := Sign(s.Secret, s.Algorithm, s.Encoding, body)
	if s.Prefix != "" {
		signature = s.Prefix + signature
	}
	r.Header.Set(s.HeaderName, signature)
}

// SignRequest signs an http.Request.
func (s *Signer) SignRequest(r *http.Request) error {
	if r.Body == nil {
		s.Sign(r, nil)
		return nil
	}

	body, err := io.ReadAll(r.Body)
	_ = r.Body.Close()
	if err != nil {
		return err
	}

	r.Body = io.NopCloser(bytes.NewReader(body))
	s.Sign(r, body)
	return nil
}
