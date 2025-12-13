// Package nocache provides middleware to prevent HTTP response caching.
//
// The nocache middleware sets multiple HTTP headers to prevent browsers,
// proxies, and CDNs from caching responses. This is essential for serving
// sensitive data, user-specific content, or frequently changing information.
//
// # Usage
//
// Basic usage:
//
//	app := mizu.New()
//	app.Use(nocache.New())
//
// Apply to specific routes:
//
//	app.Get("/api/user", userHandler, nocache.New())
//
// Apply to route groups:
//
//	api := app.Group("/api")
//	api.Use(nocache.New())
//	api.Get("/users", listUsers)
//
// # Headers Set
//
// The middleware sets the following HTTP headers:
//
//   - Cache-Control: no-store, no-cache, must-revalidate, proxy-revalidate, max-age=0
//   - Pragma: no-cache (HTTP/1.0 compatibility)
//   - Expires: 0
//   - Surrogate-Control: no-store (CDN/proxy control)
//
// # When to Use
//
// The nocache middleware should be used for:
//
//   - User-specific data (profiles, settings)
//   - Authentication and authorization responses
//   - Frequently changing data (real-time updates)
//   - Sensitive information (financial data, personal details)
//   - API endpoints that should always return fresh data
//
// # Performance
//
// The middleware has minimal performance impact:
//
//   - Zero-allocation implementation
//   - Only 4 header writes per request
//   - No configuration or state management overhead
//   - Thread-safe for concurrent requests
//
// # Best Practices
//
//   - Apply at the route level for fine-grained control
//   - Combine with HTTPS for maximum security
//   - Avoid using globally on static assets
//   - Test cache behavior with browser developer tools
//   - Consider CDN and proxy caching behavior in production
package nocache
