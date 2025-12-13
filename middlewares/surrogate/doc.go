// Package surrogate provides CDN surrogate header management middleware for Mizu.
//
// The surrogate middleware manages Surrogate-Control and Surrogate-Key headers
// for CDN cache control. It supports popular CDNs like Fastly, Varnish, and CloudFront.
//
// # Overview
//
// This middleware enables fine-grained cache control at the CDN edge by managing
// surrogate keys and control directives. Surrogate keys allow for selective cache
// purging, while surrogate control directives manage cache behavior.
//
// # Basic Usage
//
// Create a new surrogate middleware with default settings:
//
//	app := mizu.New()
//	app.Use(surrogate.New())
//
//	app.Get("/article/:id", func(c *mizu.Ctx) error {
//	    surrogate.Add(c, "article-"+c.Param("id"))
//	    return c.JSON(200, article)
//	})
//
// # Configuration
//
// The middleware can be configured with various options:
//
//	app.Use(surrogate.WithOptions(surrogate.Options{
//	    MaxAge:               3600,  // 1 hour cache
//	    StaleWhileRevalidate: 600,   // 10 minutes stale
//	    DefaultKeys:          []string{"site", "global"},
//	}))
//
// # CDN Presets
//
// The package provides preset configurations for popular CDNs:
//
//   - Fastly(): Configures for Fastly CDN using Surrogate-Key header
//   - Varnish(): Configures for Varnish using xkey header
//   - CloudFront(): Configures for AWS CloudFront using x-amz-meta-surrogate-key header
//
// Example:
//
//	app.Use(surrogate.Fastly())
//
// # Key Management
//
// Keys can be added, retrieved, or cleared within handlers:
//
//	// Add keys
//	surrogate.Add(c, "article-123", "articles")
//
//	// Get all keys
//	keys := surrogate.Get(c)
//	allKeys := keys.Get()
//
//	// Clear all keys
//	surrogate.Clear(c)
//
// # Advanced Configuration
//
// Custom headers and default keys can be configured:
//
//	app.Use(surrogate.WithOptions(surrogate.Options{
//	    Header:               "Custom-Key-Header",
//	    ControlHeader:        "Custom-Control-Header",
//	    DefaultKeys:          []string{"site"},
//	    MaxAge:               3600,
//	    StaleWhileRevalidate: 600,
//	    StaleIfError:         86400,
//	}))
//
// # Surrogate-Control Directives
//
// The middleware automatically builds the Surrogate-Control header from options:
//
//   - MaxAge: Sets max-age=N directive (in seconds)
//   - StaleWhileRevalidate: Sets stale-while-revalidate=N directive
//   - StaleIfError: Sets stale-if-error=N directive
//
// Example output: "Surrogate-Control: max-age=3600, stale-while-revalidate=600"
//
// # Context-Based Architecture
//
// The middleware uses Go's context.Context to store surrogate keys during request
// processing. This ensures keys are properly scoped to each request and can be
// modified by any handler in the chain.
//
// The workflow is:
//  1. Middleware creates a Keys instance and stores it in request context
//  2. Default keys (if configured) are automatically added
//  3. Handler chain executes, allowing handlers to add/modify keys
//  4. After handler completion, middleware sets headers based on accumulated keys
//
// # Best Practices
//
//   - Use meaningful, granular keys for selective purging
//   - Group related content with shared keys (e.g., "articles", "homepage")
//   - Set appropriate TTLs based on content freshness requirements
//   - Plan your purge strategy before implementing keys
//   - Use CDN presets when possible for correct header configuration
//
// # Integration with CDNs
//
// Fastly:
//
//	Surrogate-Key: article-123 homepage
//	Surrogate-Control: max-age=3600
//
// Varnish (with xkey vmod):
//
//	xkey: article-123 homepage
//	Surrogate-Control: max-age=3600
//
// # Thread Safety
//
// The middleware is safe for concurrent use. Each request gets its own Keys
// instance stored in the request context, ensuring no shared state between requests.
package surrogate
