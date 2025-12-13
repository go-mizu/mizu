// Package csrf2 provides enhanced CSRF (Cross-Site Request Forgery) protection
// middleware for the Mizu web framework using the double-submit cookie pattern.
//
// # Overview
//
// The csrf2 middleware implements stateless CSRF protection by storing a
// cryptographically secure token in both a cookie and requiring it to be
// submitted in requests. This approach is ideal for stateless applications
// and Single Page Applications (SPAs).
//
// # Features
//
//   - Double-submit cookie pattern for stateless protection
//   - Cryptographically secure token generation with SHA-256 signing
//   - Constant-time token comparison to prevent timing attacks
//   - Flexible token submission (header, form field, or query parameter)
//   - Configurable cookie security options (Secure, HttpOnly, SameSite)
//   - Origin validation with whitelist support
//   - Token masking for additional security
//   - Request fingerprinting capabilities
//   - HTML helpers for form inputs and meta tags
//   - Custom error handling
//   - Path and HTTP method exclusion
//
// # Basic Usage
//
//	import (
//		"github.com/go-mizu/mizu"
//		"github.com/go-mizu/mizu/middlewares/csrf2"
//	)
//
//	func main() {
//		app := mizu.New()
//
//		// Basic usage with default options
//		app.Use(csrf2.New("your-secret-key"))
//
//		app.Post("/submit", func(c *mizu.Ctx) error {
//			return c.JSON(200, map[string]string{"status": "ok"})
//		})
//
//		app.Listen(":3000")
//	}
//
// # Configuration
//
// The middleware can be configured using the WithOptions function:
//
//	app.Use(csrf2.WithOptions(csrf2.Options{
//		Secret:         "your-secret-key",
//		TokenLength:    32,
//		CookieName:     "_csrf",
//		HeaderName:     "X-CSRF-Token",
//		FormField:      "_csrf",
//		CookiePath:     "/",
//		CookieMaxAge:   86400,
//		CookieSecure:   true,
//		CookieHTTPOnly: true,
//		CookieSameSite: http.SameSiteStrictMode,
//		SkipPaths:      []string{"/api/webhook"},
//		SkipMethods:    []string{"GET", "HEAD", "OPTIONS", "TRACE"},
//	}))
//
// # Token Generation
//
// Tokens are generated using a secure process:
//  1. Generate random bytes using crypto/rand
//  2. Append Unix timestamp for rotation support
//  3. Sign with SHA-256 using the secret key
//  4. Encode with base64 URL-safe encoding
//
// # Token Validation
//
// Validation uses constant-time comparison to prevent timing attacks:
//  1. Decode both cookie and submitted tokens
//  2. Validate minimum token length
//  3. Perform XOR-based constant-time comparison
//  4. Return result without early exit
//
// # Token Submission
//
// Tokens can be submitted in three ways (checked in order):
//
// 1. HTTP Header (recommended for APIs):
//
//	fetch('/api/submit', {
//		method: 'POST',
//		headers: {
//			'X-CSRF-Token': getCsrfToken(),
//			'Content-Type': 'application/json'
//		},
//		body: JSON.stringify(data)
//	});
//
// 2. Form Field:
//
//	<form method="POST" action="/submit">
//		<input type="hidden" name="_csrf" value="{{.Token}}">
//		<button type="submit">Submit</button>
//	</form>
//
// 3. Query Parameter:
//
//	POST /submit?_csrf=token_value
//
// # Helper Functions
//
// The package provides several helper functions:
//
//   - GetToken(c *mizu.Ctx) string - Retrieves the CSRF token from context
//   - Token() mizu.Handler - Returns a handler that provides the token as JSON
//   - FormInput(c *mizu.Ctx, fieldName string) string - Generates HTML hidden input
//   - MetaTag(c *mizu.Ctx) string - Generates HTML meta tag for SPAs
//   - Mask(token string) string - Applies XOR-based masking to token
//   - Unmask(maskedToken string) string - Removes masking from token
//   - Fingerprint(c *mizu.Ctx) string - Generates request fingerprint
//
// # Form Integration Example
//
//	app.Get("/form", func(c *mizu.Ctx) error {
//		token := csrf2.GetToken(c)
//		return c.HTML(200, `
//			<html>
//			<head>` + csrf2.MetaTag(c) + `</head>
//			<body>
//				<form method="POST" action="/submit">
//					` + csrf2.FormInput(c, "") + `
//					<input type="text" name="data">
//					<button type="submit">Submit</button>
//				</form>
//			</body>
//			</html>
//		`)
//	})
//
// # SPA Integration Example
//
//	// Server-side
//	app.Get("/api/csrf-token", csrf2.Token())
//
//	// Client-side JavaScript
//	function getCsrfToken() {
//		return document.cookie
//			.split('; ')
//			.find(row => row.startsWith('_csrf='))
//			?.split('=')[1];
//	}
//
//	fetch('/api/submit', {
//		method: 'POST',
//		headers: {
//			'X-CSRF-Token': getCsrfToken(),
//			'Content-Type': 'application/json'
//		},
//		body: JSON.stringify(data)
//	});
//
// # Origin Validation
//
// Enable origin validation for additional security:
//
//	app.Use(csrf2.WithOptions(csrf2.Options{
//		Secret:         "your-secret-key",
//		ValidateOrigin: true,
//		AllowedOrigins: []string{"https://example.com", "https://app.example.com"},
//	}))
//
// # Custom Error Handling
//
//	app.Use(csrf2.WithOptions(csrf2.Options{
//		Secret: "your-secret-key",
//		ErrorHandler: func(c *mizu.Ctx) error {
//			return c.JSON(403, map[string]string{
//				"error": "CSRF validation failed",
//				"code":  "CSRF_ERROR",
//			})
//		},
//	}))
//
// # Token Masking
//
// Use token masking for additional security against BREACH attacks:
//
//	app.Get("/form", func(c *mizu.Ctx) error {
//		token := csrf2.GetToken(c)
//		maskedToken := csrf2.Mask(token)
//		return c.HTML(200, `<input type="hidden" name="_csrf" value="` + maskedToken + `">`)
//	})
//
// The middleware will automatically unmask tokens during validation.
//
// # Security Considerations
//
//   - Always use HTTPS in production to prevent token interception
//   - Set CookieSecure to true when using HTTPS
//   - Use SameSite Strict for maximum protection in modern browsers
//   - Validate origin for cross-origin request protection
//   - Regenerate tokens after authentication state changes
//   - Use appropriate token length (default 32 bytes is recommended)
//   - Consider token rotation for long-lived sessions
//
// # Best Practices
//
//   - Include the middleware before your route handlers
//   - Use SkipPaths for public webhooks and health checks
//   - Configure CookieSameSite based on your application needs
//   - Test CSRF protection in your application's test suite
//   - Monitor CSRF validation failures for potential attacks
//   - Use custom error handlers to log validation failures
//
// # Performance Considerations
//
//   - Token generation uses crypto/rand for security
//   - Validation uses constant-time comparison (no timing attacks)
//   - Tokens are cached in request context for efficiency
//   - Cookie operations are handled by Go's standard library
//   - No session storage required (stateless design)
//
// # Compatibility
//
// The csrf2 middleware is compatible with:
//   - RESTful APIs
//   - Single Page Applications (SPAs)
//   - Traditional form-based applications
//   - Mobile applications
//   - Microservices architectures
package csrf2
