// Package csrf provides Cross-Site Request Forgery (CSRF) protection middleware for Mizu.
//
// # Overview
//
// The csrf middleware protects against Cross-Site Request Forgery attacks by generating
// and validating cryptographic tokens. It ensures that form submissions and state-changing
// requests originate from your application, not from malicious third-party sites.
//
// # Quick Start
//
// Basic usage with secure cookies (production):
//
//	app := mizu.New()
//	app.Use(csrf.Protect([]byte("32-byte-secret-key-here-12345")))
//
// Development mode with insecure cookies (HTTP):
//
//	app.Use(csrf.ProtectDev([]byte("32-byte-secret-key-here-12345")))
//
// # How It Works
//
// The middleware implements the Double Submit Cookie pattern with HMAC signatures:
//
// 1. Token Generation: On safe methods (GET, HEAD, OPTIONS, TRACE), a cryptographic
// token is generated using crypto/rand and signed with HMAC-SHA256.
//
// 2. Token Storage: The token is stored in a cookie and made available in the request
// context for embedding in forms or meta tags.
//
// 3. Token Validation: On unsafe methods (POST, PUT, DELETE, PATCH), the middleware
// validates that the token from the request matches the token in the cookie and that
// the HMAC signature is valid.
//
// 4. Double Submit: Both the cookie (sent automatically by the browser) and the
// request token (sent explicitly) must match and have valid signatures.
//
// # Token Format
//
// Tokens use the format: base64(random_bytes).base64(hmac_sha256(random_bytes))
//
// This provides:
//   - Cryptographic randomness from crypto/rand
//   - Tamper detection via HMAC signature
//   - URL-safe encoding with base64 URL encoding
//
// # Configuration
//
// Customize the middleware with Options:
//
//	app.Use(csrf.New(csrf.Options{
//	    Secret:         []byte("your-32-byte-secret-key-here"),
//	    TokenLength:    32,
//	    TokenLookup:    "header:X-CSRF-Token",
//	    CookieName:     "_csrf",
//	    CookiePath:     "/",
//	    CookieMaxAge:   86400,
//	    CookieSecure:   true,
//	    CookieHTTPOnly: true,
//	    SameSite:       http.SameSiteStrictMode,
//	    SkipPaths:      []string{"/api/webhook"},
//	}))
//
// # Token Lookup
//
// The TokenLookup option supports multiple sources with "source:name" format:
//
//   - "header:X-CSRF-Token" - Read from request header (default)
//   - "form:_csrf" - Read from form field
//   - "query:csrf" - Read from query parameter
//
// # Using Tokens in Forms
//
// Extract the token and include it in your forms:
//
//	app.Get("/form", func(c *mizu.Ctx) error {
//	    token := csrf.Token(c)
//	    return c.HTML(200, `
//	        <form method="POST" action="/submit">
//	            <input type="hidden" name="_csrf" value="`+token+`">
//	            <button type="submit">Submit</button>
//	        </form>
//	    `)
//	})
//
// Or use the TemplateField helper:
//
//	app.Get("/form", func(c *mizu.Ctx) error {
//	    field := csrf.TemplateField(c)
//	    return c.HTML(200, `
//	        <form method="POST" action="/submit">
//	            `+field+`
//	            <button type="submit">Submit</button>
//	        </form>
//	    `)
//	})
//
// # Using Tokens in JavaScript
//
// For AJAX requests, include the token in request headers:
//
//	<meta name="csrf-token" content="{{ .CSRFToken }}">
//
//	<script>
//	const token = document.querySelector('meta[name="csrf-token"]').content;
//
//	fetch('/api/data', {
//	    method: 'POST',
//	    headers: {
//	        'X-CSRF-Token': token,
//	        'Content-Type': 'application/json'
//	    },
//	    body: JSON.stringify(data)
//	});
//	</script>
//
// # Error Handling
//
// The middleware provides two error types:
//
//   - ErrTokenMissing: CSRF token not found in request
//   - ErrTokenInvalid: CSRF token validation failed
//
// Customize error responses with ErrorHandler:
//
//	app.Use(csrf.New(csrf.Options{
//	    Secret: secret,
//	    ErrorHandler: func(c *mizu.Ctx, err error) error {
//	        if err == csrf.ErrTokenMissing {
//	            return c.HTML(403, "<h1>Missing security token</h1>")
//	        }
//	        return c.HTML(403, "<h1>Invalid security token</h1>")
//	    },
//	}))
//
// # Skipping Paths
//
// Exclude specific paths from CSRF validation (useful for webhooks or public APIs):
//
//	app.Use(csrf.New(csrf.Options{
//	    Secret: secret,
//	    SkipPaths: []string{
//	        "/api/webhook",
//	        "/api/public",
//	    },
//	}))
//
// # Security Considerations
//
// For production use, follow these best practices:
//
// 1. Secret Key: Use a strong, random 32-byte secret generated with GenerateSecret()
// or from a secure random source. Never commit secrets to version control.
//
// 2. Cookie Settings: Always set CookieSecure to true in production to ensure cookies
// are only sent over HTTPS. Use CookieHTTPOnly to prevent JavaScript access.
//
// 3. SameSite: Use http.SameSiteStrictMode or http.SameSiteLaxMode to prevent
// cross-site cookie transmission.
//
// 4. HTTPS Required: The Secure flag requires HTTPS. Always use TLS in production.
//
// 5. Token Rotation: Tokens are per-session. Consider regenerating after login/logout
// events for additional security.
//
// # Recommended Production Settings
//
//	app.Use(csrf.New(csrf.Options{
//	    Secret:         []byte(os.Getenv("CSRF_SECRET")),
//	    CookieSecure:   true,
//	    CookieHTTPOnly: true,
//	    SameSite:       http.SameSiteStrictMode,
//	    CookieMaxAge:   3600, // 1 hour
//	}))
//
// # Implementation Details
//
// Token Generation: Uses crypto/rand for cryptographically secure random bytes,
// then creates an HMAC-SHA256 signature to prevent tampering. Both the random
// bytes and signature are base64-encoded and joined with a period separator.
//
// Token Validation: Employs constant-time comparison (subtle.ConstantTimeCompare)
// to prevent timing attacks, and validates both the token match and HMAC signature
// using hmac.Equal for secure comparison.
//
// Context Storage: Tokens are stored in the request context using a private
// contextKey type to prevent collisions with other middleware or application code.
//
// Path Skipping: The SkipPaths option uses a map for O(1) lookup performance,
// allowing efficient bypass of CSRF validation for specific routes.
//
// # Functions
//
// Main middleware constructors:
//   - New(opts Options) - Create middleware with custom options
//   - Protect(secret []byte) - Create middleware with secure cookies (production)
//   - ProtectDev(secret []byte) - Create middleware with insecure cookies (development)
//
// Token utilities:
//   - Token(c *mizu.Ctx) string - Extract CSRF token from context
//   - TemplateField(c *mizu.Ctx) string - Generate HTML hidden input field
//   - GenerateSecret() []byte - Generate secure random 32-byte secret
//   - TokenExpiry(opts Options) time.Time - Calculate token expiration time
//
// # Related Middlewares
//
// For comprehensive security, combine CSRF with:
//   - secure: HTTPS enforcement and security headers
//   - helmet: Additional security headers (CSP, HSTS, etc.)
//   - cors: Cross-Origin Resource Sharing controls
package csrf
