// Package vary provides Vary header management middleware for Mizu.
//
// The Vary HTTP header tells caches which request headers affect the response
// content. This is crucial for proper cache control and content negotiation.
//
// # Basic Usage
//
// Use the New function to create middleware with specific headers:
//
//	app := mizu.New()
//	app.Use(vary.New("Accept-Encoding"))
//
// # Multiple Headers
//
// Specify multiple headers that affect the response:
//
//	app.Use(vary.New("Accept-Encoding", "Accept-Language", "Authorization"))
//
// # Convenience Functions
//
// The package provides several convenience functions for common use cases:
//
//	app.Use(vary.AcceptEncoding())  // Vary: Accept-Encoding
//	app.Use(vary.Accept())          // Vary: Accept
//	app.Use(vary.AcceptLanguage())  // Vary: Accept-Language
//	app.Use(vary.Origin())          // Vary: Origin (CORS)
//	app.Use(vary.All())             // Vary: Accept, Accept-Encoding, Accept-Language
//
// # Auto-Detection
//
// Enable automatic detection of content negotiation headers:
//
//	app.Use(vary.Auto())
//
// This automatically adds Vary headers based on request headers (Accept,
// Accept-Encoding, Accept-Language) if they are present.
//
// # Custom Options
//
// Use WithOptions for advanced configuration:
//
//	app.Use(vary.WithOptions(vary.Options{
//		Headers: []string{"Authorization", "X-Custom"},
//		Auto:    true,
//	}))
//
// # Adding Headers in Handlers
//
// Use the Add helper function to add Vary headers within handlers:
//
//	app.Get("/api/data", func(c *mizu.Ctx) error {
//		vary.Add(c, "X-Custom-Header")
//		return c.JSON(data)
//	})
//
// # Implementation Details
//
// The middleware processes headers after the handler executes to capture all
// Vary requirements. It prevents duplicates using case-insensitive comparison
// and properly merges with existing Vary headers set by the application.
//
// # Common Patterns
//
// API with authentication:
//
//	app.Use(vary.New("Authorization"))
//
// Multilingual site:
//
//	app.Use(vary.New("Accept-Language"))
//
// Compressed responses:
//
//	app.Use(vary.New("Accept-Encoding"))
//
// # Best Practices
//
//   - Include all headers that affect the response content
//   - Use with compression middleware for proper cache control
//   - Consider CDN caching implications when choosing headers
//   - Avoid over-varying as it reduces cache effectiveness
//
// # Caching Behavior
//
// The Vary header tells caches to store separate versions of the response
// based on the specified request headers. For example:
//
//	Vary: Accept-Encoding
//
// This tells caches to store different versions for gzip, deflate, and
// uncompressed responses.
//
// # Integration with CDNs
//
// Most CDNs respect the Vary header. Common CDN configurations:
//
//   - CloudFront: Supports Vary on Accept-Encoding, Accept, and custom headers
//   - Cloudflare: Automatically handles Accept-Encoding
//   - Fastly: Full Vary header support
//
// Always test your CDN configuration with the Vary headers you use.
package vary
