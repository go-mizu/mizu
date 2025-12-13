// Package conditional provides HTTP conditional request handling middleware for Mizu.
//
// The conditional middleware implements RFC 7232 conditional request handling,
// enabling efficient caching through ETags and Last-Modified headers. It automatically
// handles If-None-Match and If-Modified-Since headers, returning 304 Not Modified
// responses when appropriate.
//
// # Features
//
//   - ETag generation using MD5 hashing (configurable)
//   - Support for both strong and weak ETags
//   - Last-Modified header handling with custom modification time functions
//   - Automatic 304 Not Modified responses
//   - Custom ETag generation functions
//   - Optimized for GET and HEAD requests only
//
// # Basic Usage
//
// Default configuration with ETag support:
//
//	app := mizu.New()
//	app.Use(conditional.New())
//
//	app.Get("/api/data", func(c *mizu.Ctx) error {
//	    return c.JSON(http.StatusOK, data)
//	})
//
// # Advanced Configuration
//
// Custom options with both ETag and Last-Modified:
//
//	app.Use(conditional.WithOptions(conditional.Options{
//	    ETag:         true,
//	    WeakETag:     true,  // Use weak ETags
//	    LastModified: true,
//	    ModTimeFunc: func(c *mizu.Ctx) time.Time {
//	        return getResourceModTime(c.Request().URL.Path)
//	    },
//	}))
//
// # Convenience Functions
//
// ETag only:
//
//	app.Use(conditional.ETagOnly())
//
// Last-Modified only:
//
//	app.Use(conditional.LastModifiedOnly(func(c *mizu.Ctx) time.Time {
//	    return time.Now()
//	}))
//
// Combined ETag and Last-Modified:
//
//	app.Use(conditional.WithModTime(func(c *mizu.Ctx) time.Time {
//	    return getModTime(c)
//	}))
//
// # Custom ETag Generation
//
// For better performance or custom hashing:
//
//	app.Use(conditional.WithOptions(conditional.Options{
//	    ETag: true,
//	    ETagFunc: func(body []byte) string {
//	        // Use faster hash or version-based ETag
//	        return fmt.Sprintf("v%d-%d", version, len(body))
//	    },
//	}))
//
// # How It Works
//
// 1. The middleware captures the response body and status code
// 2. For successful responses (2xx), it generates an ETag from the body content
// 3. If configured, it sets the Last-Modified header using the provided function
// 4. It checks incoming If-None-Match and If-Modified-Since headers
// 5. If the resource hasn't changed, it returns 304 Not Modified
// 6. Otherwise, it sends the full response with caching headers
//
// # Performance Considerations
//
//   - Response bodies are buffered for ETag generation
//   - MD5 hashing is used by default (non-cryptographic use)
//   - Only processes GET and HEAD requests
//   - Skips processing for error responses (non-2xx)
//   - Consider custom ETagFunc for large responses
//
// # HTTP Compliance
//
// The middleware implements conditional request handling as specified in:
//   - RFC 7232: Hypertext Transfer Protocol (HTTP/1.1): Conditional Requests
//
// It properly handles:
//   - If-None-Match with ETag comparison
//   - If-Modified-Since with Last-Modified comparison
//   - Weak vs strong ETag validation
//   - 304 Not Modified responses with appropriate headers
//
// # Security Notes
//
// MD5 is used for ETag generation, which is appropriate for this use case.
// ETags are content fingerprints for caching, not cryptographic signatures.
// The middleware includes nolint directives to suppress security warnings
// about MD5 usage in non-security contexts.
package conditional
