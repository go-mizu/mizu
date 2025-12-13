// Package secure provides HTTPS enforcement and comprehensive security headers middleware for Mizu.
//
// The secure middleware enforces HTTPS connections and adds essential security headers
// to protect web applications from common vulnerabilities. It combines SSL redirect
// functionality with security header management in a single, configurable middleware.
//
// # Features
//
//   - Automatic HTTP to HTTPS redirection
//   - HSTS (HTTP Strict Transport Security) support with preload option
//   - Security headers (X-Content-Type-Options, X-Frame-Options, X-XSS-Protection)
//   - Content Security Policy (CSP) configuration
//   - Referrer Policy configuration
//   - Proxy-aware HTTPS detection
//   - Development mode bypass
//
// # Basic Usage
//
//	app := mizu.New()
//	app.Use(secure.New())
//
// This creates middleware with default security settings:
//   - SSLRedirect: enabled
//   - ContentTypeNosniff: enabled
//   - FrameDeny: enabled
//   - XSSProtection: "1; mode=block"
//
// # Custom Configuration
//
// Use WithOptions for custom security configuration:
//
//	app.Use(secure.WithOptions(secure.Options{
//	    SSLRedirect:         true,
//	    STSSeconds:          31536000, // 1 year
//	    STSIncludeSubdomains: true,
//	    STSPreload:          true,
//	    ContentTypeNosniff:  true,
//	    FrameDeny:           true,
//	    ContentSecurityPolicy: "default-src 'self'",
//	    ReferrerPolicy:      "strict-origin-when-cross-origin",
//	}))
//
// # HTTPS Detection
//
// The middleware detects HTTPS connections in two ways:
//
//  1. Direct TLS connection (request.TLS != nil)
//  2. Proxy headers (X-Forwarded-Proto, X-Forwarded-SSL, etc.)
//
// When behind a reverse proxy or load balancer, configure ProxyHeaders:
//
//	app.Use(secure.WithOptions(secure.Options{
//	    SSLRedirect: true,
//	    ProxyHeaders: []string{
//	        "X-Forwarded-Proto",
//	        "X-Forwarded-SSL",
//	    },
//	}))
//
// # SSL Redirection
//
// When SSLRedirect is enabled and HTTP is detected:
//   - Constructs HTTPS URL: https://<host><requestURI>
//   - Uses SSLHost if specified, otherwise uses request host
//   - Returns 301 (Moved Permanently) by default
//   - Returns 307 (Temporary Redirect) if SSLTemporaryRedirect is true
//
// Example with custom SSL host:
//
//	app.Use(secure.WithOptions(secure.Options{
//	    SSLRedirect: true,
//	    SSLHost:     "secure.example.com",
//	}))
//
// # HSTS Configuration
//
// HTTP Strict Transport Security (HSTS) instructs browsers to only use HTTPS:
//
//	app.Use(secure.WithOptions(secure.Options{
//	    STSSeconds:           31536000, // 1 year
//	    STSIncludeSubdomains: true,     // Apply to subdomains
//	    STSPreload:           true,     // Enable preload list
//	}))
//
// Important considerations:
//   - HSTS header is only sent on HTTPS connections (unless ForceSTSHeader is true)
//   - Browsers remember HSTS policy for max-age duration
//   - Start with short duration (e.g., 3600 seconds) and increase gradually
//   - Ensure all subdomains support HTTPS before enabling STSIncludeSubdomains
//   - Submit to hstspreload.org for browser preload list inclusion
//
// # Security Headers
//
// The middleware sets the following security headers based on configuration:
//
//	Strict-Transport-Security: max-age=<seconds>; includeSubDomains; preload
//	X-Content-Type-Options: nosniff
//	X-Frame-Options: DENY | SAMEORIGIN | custom
//	X-XSS-Protection: 1; mode=block
//	Content-Security-Policy: <policy>
//	Referrer-Policy: <policy>
//
// # Development Mode
//
// Use IsDevelopment flag to disable all security features for local development:
//
//	app.Use(secure.WithOptions(secure.Options{
//	    IsDevelopment: true,
//	}))
//
// This bypasses all security checks and header modifications, allowing
// HTTP connections and skipping all security headers.
//
// # Performance
//
// The middleware is optimized for performance:
//   - Custom itoa function avoids heap allocations for integer-to-string conversion
//   - Case-insensitive header matching using strings.EqualFold
//   - Early return in development mode
//   - Conditional header setting based on configuration
//
// # Example: Environment-Based Configuration
//
//	func setupSecurity(app *mizu.App) {
//	    isDev := os.Getenv("ENV") == "development"
//
//	    app.Use(secure.WithOptions(secure.Options{
//	        IsDevelopment:       isDev,
//	        SSLRedirect:         !isDev,
//	        STSSeconds:          31536000,
//	        STSIncludeSubdomains: true,
//	        ContentTypeNosniff:  true,
//	        FrameDeny:           true,
//	        ContentSecurityPolicy: "default-src 'self'",
//	        ReferrerPolicy:      "strict-origin-when-cross-origin",
//	    }))
//	}
//
// # Security Best Practices
//
//  1. Start with conservative HSTS settings and increase gradually
//  2. Test thoroughly before enabling HSTS preload
//  3. Ensure valid SSL certificate before enabling HSTS
//  4. Use IsDevelopment flag for local development
//  5. Configure ProxyHeaders when behind load balancers
//  6. Implement comprehensive Content Security Policy
//  7. Review and update security headers regularly
//
// # Related Middlewares
//
//   - helmet: Detailed security headers configuration
//   - redirect: General-purpose URL redirection
package secure
