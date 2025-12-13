// Package rewrite provides URL rewriting middleware for the Mizu web framework.
//
// The rewrite middleware transforms URL paths internally without sending redirects
// to the client. This allows the server to process requests at different paths while
// keeping the browser URL unchanged.
//
// # Basic Usage
//
// Create a simple prefix rewrite:
//
//	app := mizu.New()
//	app.Use(rewrite.New(rewrite.Prefix("/api/v1", "/api")))
//
// # Rule Types
//
// The package supports two types of rewrite rules:
//
// 1. Prefix Rules - Simple string prefix matching:
//
//	rewrite.Prefix("/old", "/new")
//	// /old/path → /new/path
//
// 2. Regex Rules - Pattern matching with capture groups:
//
//	rewrite.Regex(`^/posts/(\d+)$`, "/articles/$1")
//	// /posts/123 → /articles/123
//
// # Multiple Rules
//
// Multiple rules can be combined, with the first matching rule being applied:
//
//	app.Use(rewrite.New(
//	    rewrite.Prefix("/api/v1", "/api"),
//	    rewrite.Prefix("/legacy", "/current"),
//	    rewrite.Regex(`^/user/(\d+)$`, "/users/$1"),
//	))
//
// # Advanced Configuration
//
// For more control, use WithOptions:
//
//	app.Use(rewrite.WithOptions(rewrite.Options{
//	    Rules: []rewrite.Rule{
//	        {Match: "/old", Rewrite: "/new", Regex: false},
//	        {Match: `^/api/v(\d+)/(.*)$`, Rewrite: "/api/$2", Regex: true},
//	    },
//	}))
//
// # Implementation Details
//
// The middleware operates by modifying the Request.URL.Path before the request
// reaches downstream handlers. Regex patterns are compiled once during initialization
// for optimal performance. Rule matching stops at the first match (first-match-wins).
//
// # Performance
//
// - Regex patterns are compiled at middleware initialization
// - Per-request overhead is O(r) where r is the number of rules
// - Processing stops after the first matching rule
// - Prefix matching uses optimized string operations
//
// # Thread Safety
//
// The middleware is thread-safe and maintains no request-specific state.
// All regex patterns are compiled during initialization and shared across requests.
//
// # Rewrite vs Redirect
//
// Unlike redirects, rewrites:
// - Do not change the client-visible URL
// - Do not require an additional HTTP round-trip
// - Are invisible to the client and search engines
// - Are ideal for internal routing and API versioning
//
// For client-visible URL changes, use the redirect middleware instead.
package rewrite
