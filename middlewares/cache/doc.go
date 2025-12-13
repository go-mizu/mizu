// Package cache provides HTTP Cache-Control header middleware for the Mizu web framework.
//
// The cache middleware enables fine-grained control over HTTP caching behavior by setting
// appropriate Cache-Control headers on responses. It supports all standard Cache-Control
// directives and provides convenient helper functions for common caching patterns.
//
// # Basic Usage
//
// The simplest way to use the cache middleware is with the New function:
//
//	app := mizu.New()
//	app.Use(cache.New(time.Hour))  // Cache for 1 hour
//
// # Configuration Options
//
// The Options struct provides full control over cache behavior:
//
//	type Options struct {
//	    MaxAge               time.Duration  // Browser cache duration
//	    SMaxAge              time.Duration  // CDN/proxy cache duration
//	    Public               bool           // Allow public caching
//	    Private              bool           // Restrict to browser only
//	    NoCache              bool           // Require revalidation
//	    NoStore              bool           // Don't cache at all
//	    NoTransform          bool           // Prevent transformations
//	    MustRevalidate       bool           // Must check freshness
//	    ProxyRevalidate      bool           // Proxy must revalidate
//	    Immutable            bool           // Content never changes
//	    StaleWhileRevalidate time.Duration  // Serve stale while fetching
//	    StaleIfError         time.Duration  // Serve stale on errors
//	}
//
// # Helper Functions
//
// The package provides several convenience functions for common scenarios:
//
//   - New(maxAge): Basic caching with max-age directive
//   - Public(maxAge): Public caching suitable for CDNs
//   - Private(maxAge): Private caching for user-specific content
//   - Immutable(maxAge): Immutable content that never changes
//   - Static(maxAge): Static assets with public and immutable directives
//   - SWR(maxAge, stale): Stale-while-revalidate for improved performance
//
// # Examples
//
// Public Cache for Static Assets:
//
//	app.Get("/static/*", staticHandler, cache.Public(24*time.Hour))
//
// Private Cache for User Data:
//
//	app.Get("/profile", profileHandler, cache.Private(time.Hour))
//
// Immutable Assets with Long Cache:
//
//	app.Get("/assets/*", assetHandler, cache.Immutable(365*24*time.Hour))
//
// Stale-While-Revalidate Pattern:
//
//	app.Get("/api/feed", feedHandler, cache.SWR(time.Minute, time.Hour))
//
// Full Configuration:
//
//	app.Use(cache.WithOptions(cache.Options{
//	    MaxAge:               time.Hour,
//	    SMaxAge:              24 * time.Hour,
//	    Public:               true,
//	    MustRevalidate:       true,
//	    StaleWhileRevalidate: 10 * time.Minute,
//	}))
//
// # Implementation Details
//
// The middleware constructs the Cache-Control header by combining the specified directives
// based on the Options configuration. It converts time.Duration values to seconds as required
// by the HTTP specification.
//
// The middleware only sets the Cache-Control header if it hasn't already been set by the
// handler or previous middleware, allowing for fine-grained control and overrides when needed.
//
// When no options are specified (empty Options struct), the middleware defaults to "no-cache"
// to ensure safe behavior.
//
// # Cache-Control Directives
//
// The middleware supports all standard HTTP Cache-Control directives:
//
//   - public: Response may be cached by any cache (browsers, CDNs, proxies)
//   - private: Response is intended for a single user and should only be cached by browsers
//   - no-cache: Cache must revalidate with origin server before using cached content
//   - no-store: Response must not be stored in any cache
//   - no-transform: Intermediaries must not transform the response
//   - must-revalidate: Cache must verify status of stale resources
//   - proxy-revalidate: Same as must-revalidate but only for shared caches
//   - max-age=N: Maximum time (in seconds) a resource is considered fresh
//   - s-maxage=N: Overrides max-age for shared caches (CDNs, proxies)
//   - immutable: Resource will never change during its freshness lifetime
//   - stale-while-revalidate=N: Cache may serve stale content while revalidating
//   - stale-if-error=N: Cache may serve stale content if error occurs
//
// # Best Practices
//
//   - Use private or no-store for sensitive or user-specific data
//   - Use public and immutable for versioned static assets
//   - Leverage stale-while-revalidate to improve perceived performance
//   - Combine max-age and s-maxage for different cache tiers (browser vs CDN)
//   - Never use public for personalized content
//   - Consider using no-cache or no-store for authenticated routes
//
// # Security Considerations
//
// Always use appropriate cache directives for sensitive data. The private directive ensures
// content is only cached by the user's browser, while no-store prevents caching entirely.
// Never use public caching for user-specific or authenticated content.
package cache
