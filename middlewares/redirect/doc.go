// Package redirect provides URL redirection middleware for the Mizu web framework.
//
// This package offers comprehensive redirect functionality including HTTPS enforcement,
// WWW/non-WWW domain handling, trailing slash redirects, and custom rule-based redirects
// with regex pattern support.
//
// # Features
//
//   - HTTPS enforcement with customizable status codes
//   - WWW and non-WWW domain redirects
//   - Trailing slash addition
//   - Custom redirect rules with exact match or regex patterns
//   - Query string preservation across all redirect types
//   - Regex capture group support ($1, $2, etc.)
//
// # Basic Usage
//
// HTTPS Redirect:
//
//	app := mizu.New()
//	app.Use(redirect.HTTPSRedirect())
//
// WWW Domain Handling:
//
//	// Add www subdomain
//	app.Use(redirect.WWWRedirect())
//
//	// Remove www subdomain
//	app.Use(redirect.NonWWWRedirect())
//
// Trailing Slash:
//
//	app.Use(redirect.TrailingSlashRedirect())
//
// # Custom Redirect Rules
//
// Simple exact match redirects:
//
//	app.Use(redirect.New([]redirect.Rule{
//		{From: "/old-page", To: "/new-page", Code: 301},
//		{From: "/blog", To: "/articles", Code: 302},
//	}))
//
// Regex pattern redirects with capture groups:
//
//	app.Use(redirect.New([]redirect.Rule{
//		{
//			From:  `^/posts/(\d+)$`,
//			To:    "/articles/$1",
//			Regex: true,
//			Code:  301,
//		},
//		{
//			From:  `^/user/(\w+)/post/(\d+)$`,
//			To:    "/users/$1/articles/$2",
//			Regex: true,
//			Code:  301,
//		},
//	}))
//
// # Redirect Codes
//
// The package supports standard HTTP redirect status codes:
//   - 301 (Moved Permanently): Default for permanent redirects, SEO-friendly
//   - 302 (Found): Temporary redirect
//   - 307 (Temporary Redirect): Temporary redirect that preserves HTTP method
//   - 308 (Permanent Redirect): Permanent redirect that preserves HTTP method
//
// # HTTPS Detection
//
// HTTPS redirects detect secure connections through:
//   - c.Request().TLS: Direct TLS connection
//   - X-Forwarded-Proto header: Proxy/load balancer forwarded protocol
//
// # Query String Preservation
//
// All redirect types automatically preserve query strings. For example:
//   - /old?foo=bar redirects to /new?foo=bar
//   - /users/123?page=2 redirects to /profile/123?page=2
//
// # Performance
//
//   - Regex patterns are compiled once during middleware initialization
//   - Rules are evaluated in order; first match wins
//   - Minimal overhead for HTTPS and domain redirects
//   - Query string handling uses efficient string operations
//
// # Best Practices
//
// 1. Order matters when combining redirects:
//
//	app.Use(redirect.HTTPSRedirect())       // First: Secure the connection
//	app.Use(redirect.NonWWWRedirect())      // Then: Normalize the domain
//
// 2. Use appropriate status codes:
//   - 301 for permanent moves (affects SEO)
//   - 307/308 to preserve HTTP method (POST, PUT, etc.)
//
// 3. Optimize custom rules:
//   - Place frequently matched rules first
//   - Keep regex patterns simple
//   - Use exact matches when possible
//
// # Thread Safety
//
// All middleware functions are thread-safe and can be safely used with concurrent requests.
// Regex patterns are compiled during initialization, making runtime handling concurrent-safe.
package redirect
