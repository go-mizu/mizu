// Package etag provides HTTP ETag generation middleware for the Mizu framework.
//
// # Overview
//
// The etag middleware automatically generates ETag headers for HTTP responses,
// enabling efficient caching through conditional requests. When a client sends
// a matching If-None-Match header, the middleware returns a 304 Not Modified
// response, saving bandwidth and improving performance.
//
// # Features
//
//   - Automatic ETag generation using CRC32 hashing (customizable)
//   - Support for both strong and weak ETags
//   - Conditional request handling (If-None-Match)
//   - Wildcard support (If-None-Match: *)
//   - Consistent ETags for identical content
//   - Method filtering (GET and HEAD only)
//   - Error response filtering (no ETags for 4xx/5xx)
//
// # Quick Start
//
// Basic usage with default settings (strong ETags, CRC32 hashing):
//
//	app := mizu.New()
//	app.Use(etag.New())
//
//	app.Get("/api/data", func(c *mizu.Ctx) error {
//	    return c.JSON(200, map[string]string{"message": "Hello"})
//	})
//
// # Weak ETags
//
// Weak ETags are prefixed with W/ and indicate semantic equivalence rather
// than byte-for-byte identity. Use them for dynamic content that may vary
// slightly but is semantically the same:
//
//	app.Use(etag.Weak())
//	// Generates: ETag: W/"abc123def456"
//
// # Custom Hash Function
//
// You can provide a custom hash function for specific requirements:
//
//	import (
//	    "crypto/sha256"
//	    "encoding/hex"
//	)
//
//	app.Use(etag.WithOptions(etag.Options{
//	    HashFunc: func(body []byte) string {
//	        h := sha256.Sum256(body)
//	        return hex.EncodeToString(h[:16])
//	    },
//	}))
//
// # How It Works
//
// The middleware operates in the following sequence:
//
//  1. Request arrives - middleware checks if method is GET or HEAD
//  2. Response buffering - middleware buffers the response body
//  3. Hash calculation - generates ETag from buffered content
//  4. Conditional check - compares with If-None-Match header
//  5. Response decision:
//     - If match: returns 304 Not Modified with ETag header
//     - If no match: returns full response with ETag header
//
// # Technical Implementation
//
// The middleware uses a buffered writer approach that wraps the original
// http.ResponseWriter. This allows it to:
//
//   - Capture the complete response body for hashing
//   - Determine the final status code
//   - Make conditional decisions before sending the response
//
// By default, CRC32 (IEEE polynomial) is used for hash generation because it:
//
//   - Provides fast computation suitable for HTTP caching
//   - Offers sufficient uniqueness for cache validation
//   - Has low memory overhead
//
// # ETag Formats
//
// Strong ETags (default):
//
//	ETag: "abc123def456"
//
// Indicates byte-for-byte identical content. Use for static files or content
// that must be exactly the same.
//
// Weak ETags:
//
//	ETag: W/"abc123def456"
//
// Indicates semantically equivalent content. Use for dynamic content that may
// have minor variations (timestamps, formatting) but represents the same data.
//
// # Behavior Details
//
// Method Filtering:
//
// Only GET and HEAD requests receive ETag processing. Other methods (POST, PUT,
// DELETE, etc.) are passed through without modification.
//
// Status Code Filtering:
//
// ETags are only generated for successful responses (2xx status codes) with
// non-empty bodies. Error responses (4xx, 5xx) do not receive ETag headers.
//
// Wildcard Support:
//
// The middleware supports RFC 7232 wildcard matching. When a client sends
// If-None-Match: *, the middleware always returns 304 Not Modified for
// successful responses.
//
// # Best Practices
//
// Use strong ETags for:
//
//   - Static files (CSS, JavaScript, images)
//   - API responses with stable content
//   - Downloaded files
//
// Use weak ETags for:
//
//   - Dynamic content with minor variations
//   - HTML pages with timestamps
//   - Responses that may have formatting differences
//
// Middleware Ordering:
//
// Place the ETag middleware after compression middleware to ensure ETags
// are calculated on the final, compressed content:
//
//	app.Use(compress.New())
//	app.Use(etag.New())
//
// Combine with Cache-Control:
//
// Use ETag middleware together with Cache-Control headers for a complete
// caching strategy:
//
//	app.Use(etag.New())
//	app.Use(cache.New(cache.Options{
//	    MaxAge: 3600,
//	}))
//
// # Performance Considerations
//
// The middleware buffers the entire response body in memory before sending it
// to the client. For very large responses, this may impact memory usage.
// Consider using streaming responses or skipping ETag generation for large
// files.
//
// The default CRC32 hash function is optimized for speed. For applications
// requiring cryptographic hash functions, provide a custom HashFunc, but be
// aware this may impact performance.
//
// # RFC Compliance
//
// This middleware implements ETag handling according to:
//
//   - RFC 7232: Hypertext Transfer Protocol (HTTP/1.1): Conditional Requests
//   - RFC 9110: HTTP Semantics (ETag header field)
//
// # See Also
//
//   - cache middleware: For Cache-Control header management
//   - lastmodified middleware: For Last-Modified header support
//   - compress middleware: For response compression
package etag
