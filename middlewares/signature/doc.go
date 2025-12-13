// Package signature provides request signature verification middleware for Mizu.
//
// The signature middleware validates HMAC-based signatures on incoming requests,
// commonly used for webhook verification and API authentication. It supports
// multiple algorithms (SHA-1, SHA-256, SHA-512) and encoding formats (hex, base64).
//
// # Basic Usage
//
// Create middleware with default options (SHA-256, hex encoding):
//
//	app := mizu.New()
//	app.Use(signature.New("your-secret-key"))
//
// # Configuration
//
// Customize the middleware with WithOptions:
//
//	app.Use(signature.WithOptions(signature.Options{
//		Secret:          "your-secret-key",
//		Algorithm:       signature.SHA256,
//		HeaderName:      "X-Signature",
//		SignaturePrefix: "sha256=",
//		Encoding:        "hex",
//		SkipPaths:       []string{"/health"},
//		SkipMethods:     []string{"GET", "HEAD", "OPTIONS"},
//	}))
//
// # Webhook Providers
//
// Pre-configured helpers for popular webhook providers:
//
// GitHub webhooks:
//
//	app.Post("/webhook/github", handler, signature.GitHub("github-secret"))
//
// Stripe webhooks:
//
//	app.Post("/webhook/stripe", handler, signature.Stripe("stripe-secret"))
//
// Slack request verification:
//
//	app.Post("/webhook/slack", handler, signature.Slack("slack-signing-secret"))
//
// Twilio request validation:
//
//	app.Post("/webhook/twilio", handler, signature.Twilio("twilio-auth-token"))
//
// AWS Signature Version 4 (simplified):
//
//	app.Post("/webhook/aws", handler, signature.AWS("aws-secret-key"))
//
// # Client-Side Signing
//
// Sign outgoing requests using the Signer:
//
//	signer := signature.NewSigner("secret")
//	signer.Algorithm = signature.SHA256
//	signer.Encoding = "hex"
//	signer.HeaderName = "X-Signature"
//	signer.Prefix = "sha256="
//
//	req, _ := http.NewRequest("POST", url, bytes.NewReader(payload))
//	signer.SignRequest(req)
//
// Or sign payload directly:
//
//	signature := signature.Sign("secret", signature.SHA256, "hex", payload)
//
// # Signature Verification
//
// Access verification results in handlers:
//
//	app.Post("/webhook", func(c *mizu.Ctx) error {
//		if signature.IsValid(c) {
//			// Signature is valid
//			info := signature.GetInfo(c)
//			log.Printf("Valid signature: %s (%s)", info.Signature, info.Algorithm)
//		}
//		return c.Text(200, "ok")
//	})
//
// # Advanced Configuration
//
// Custom payload extraction:
//
//	app.Use(signature.WithOptions(signature.Options{
//		Secret: "secret",
//		PayloadGetter: func(c *mizu.Ctx) ([]byte, error) {
//			// Custom logic to extract payload
//			timestamp := c.Request().Header.Get("X-Timestamp")
//			body, _ := io.ReadAll(c.Request().Body)
//			return []byte(timestamp + string(body)), nil
//		},
//	}))
//
// Custom error handling:
//
//	app.Use(signature.WithOptions(signature.Options{
//		Secret: "secret",
//		ErrorHandler: func(c *mizu.Ctx) error {
//			log.Printf("Signature validation failed: %s", c.Request().URL.Path)
//			return c.JSON(403, map[string]string{
//				"error": "Invalid signature",
//			})
//		},
//	}))
//
// Success callback:
//
//	app.Use(signature.WithOptions(signature.Options{
//		Secret: "secret",
//		OnValid: func(c *mizu.Ctx) {
//			log.Printf("Valid signature from %s", c.Request().RemoteAddr)
//		},
//	}))
//
// # Security Considerations
//
// - Uses constant-time comparison (hmac.Equal) to prevent timing attacks
// - Request body is preserved after reading for downstream handlers
// - GET, HEAD, and OPTIONS methods are skipped by default
// - SHA-256 is the recommended algorithm (SHA-512 for higher security)
// - SHA-1 is supported for legacy webhook providers (e.g., Twilio)
// - Secrets should be at least 32 bytes of random data
// - Consider adding timestamp validation to prevent replay attacks
//
// # Algorithm Support
//
// The following HMAC algorithms are supported:
//
//   - SHA1:   HMAC-SHA1 (legacy, use only for compatibility)
//   - SHA256: HMAC-SHA256 (recommended, default)
//   - SHA512: HMAC-SHA512 (high security)
//
// # Encoding Formats
//
// Signatures can be encoded in:
//
//   - "hex":    Hexadecimal encoding (default, lowercase)
//   - "base64": Standard base64 encoding
//
// # Examples
//
// For comprehensive examples and use cases, see:
// https://github.com/go-mizu/mizu/tree/main/middlewares/signature
package signature
