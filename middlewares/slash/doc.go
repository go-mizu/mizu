// Package slash provides trailing slash handling middleware for Mizu.
//
// The slash middleware normalizes URL paths by adding or removing trailing slashes
// through HTTP redirects. This is useful for maintaining URL consistency across your
// application and improving SEO by preventing duplicate content issues.
//
// # Features
//
//   - Add or remove trailing slashes with configurable HTTP status codes
//   - Preserve query strings during redirection
//   - Automatic root path protection to prevent infinite redirects
//   - Zero-allocation fast path for URLs that don't need modification
//   - Simple string operations without regular expressions
//
// # Usage
//
// Add trailing slashes to all URLs (except root):
//
//	app := mizu.New()
//	app.Use(slash.Add())
//	// /about    → 301 → /about/
//	// /contact  → 301 → /contact/
//	// /         → no redirect
//
// Remove trailing slashes from all URLs (except root):
//
//	app := mizu.New()
//	app.Use(slash.Remove())
//	// /about/   → 301 → /about
//	// /contact/ → 301 → /contact
//	// /         → no redirect
//
// Use custom HTTP status codes:
//
//	app.Use(slash.AddCode(302))     // Temporary redirect
//	app.Use(slash.RemoveCode(307))  // Preserve HTTP method
//
// # Behavior
//
//   - The root path "/" is never modified to prevent infinite redirects
//   - Query strings are always preserved during redirection
//   - Default status code is 301 (Moved Permanently) for SEO benefits
//   - Supports any valid HTTP redirect status code (3xx)
//
// # Performance
//
// The middleware is designed for minimal overhead:
//
//   - Zero allocations when no redirect is needed
//   - Single string concatenation for building redirect URLs
//   - Early exit for root path and already-normalized URLs
//   - No regular expression parsing
//
// # Best Practices
//
//   - Choose one trailing slash style (with or without) and use it consistently
//   - Use 301 (default) for permanent redirects to gain SEO benefits
//   - Apply slash middleware early in the middleware chain
//   - Avoid using both Add() and Remove() in the same application
//
// # Implementation Details
//
// The middleware works by intercepting requests and checking the URL path:
//
//  1. Extract the request path from c.Request().URL.Path
//  2. Skip processing if the path is the root "/"
//  3. Check if the path has (or lacks) a trailing slash
//  4. If modification is needed, build the target URL
//  5. Preserve query strings by appending c.Request().URL.RawQuery
//  6. Execute redirect with c.Redirect(code, target)
//  7. Otherwise, pass the request to the next handler
//
// For more information, see https://github.com/go-mizu/mizu/tree/main/middlewares/slash
package slash
