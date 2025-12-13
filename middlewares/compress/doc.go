// Package compress provides HTTP response compression middleware for the Mizu web framework.
//
// The compress middleware automatically compresses HTTP responses using gzip or deflate
// encoding when the client supports it. This reduces bandwidth usage and improves load
// times for text-based content.
//
// # Features
//
//   - Automatic compression using gzip or deflate algorithms
//   - Configurable compression levels (1-9)
//   - Minimum response size threshold to avoid compressing small responses
//   - Content-Type filtering to only compress text-based content
//   - Writer pooling for optimal performance and reduced GC pressure
//   - Automatic encoding selection based on client Accept-Encoding header
//   - Support for streaming responses via http.Flusher interface
//
// # Basic Usage
//
// Enable gzip compression with default settings:
//
//	app := mizu.New()
//	app.Use(compress.Gzip())
//
// Enable deflate compression:
//
//	app.Use(compress.Deflate())
//
// # Custom Compression Level
//
// Use a specific compression level (1-9):
//
//	// Best compression (slower)
//	app.Use(compress.GzipLevel(9))
//
//	// Fastest compression
//	app.Use(compress.GzipLevel(1))
//
// # Advanced Configuration
//
// Configure compression with custom options:
//
//	app.Use(compress.New(compress.Options{
//	    Level:    6,     // Compression level (1-9)
//	    MinSize:  1024,  // Only compress responses > 1KB
//	    ContentTypes: []string{
//	        "text/plain",
//	        "text/html",
//	        "application/json",
//	    },
//	}))
//
// # Auto-Select Encoding
//
// Automatically select gzip or deflate based on Accept-Encoding header:
//
//	app.Use(compress.New(compress.Options{
//	    MinSize: 512,
//	}))
//
// # Implementation Details
//
// The middleware uses a buffering strategy to determine whether compression should be applied:
//
//  1. Response data is buffered until it reaches the MinSize threshold
//  2. The middleware checks if compression should be applied based on:
//     - Client Accept-Encoding header support
//     - Content-Type of the response
//     - Existing Content-Encoding header (skips if already encoded)
//  3. If compression is appropriate, the buffered data is compressed and sent
//
// Writer pooling is used to reuse compression writers across requests, reducing
// garbage collection overhead and improving performance.
//
// # Compression Levels
//
// Compression level determines the trade-off between speed and compression ratio:
//
//   - Level 1: Best speed, least compression
//   - Level 6: Default balance (recommended for most use cases)
//   - Level 9: Best compression, slowest
//
// # Default Content Types
//
// By default, the following content types are compressed:
//
//   - text/html, text/css, text/plain, text/javascript, text/xml
//   - application/json, application/javascript, application/xml
//   - application/xhtml+xml, application/rss+xml, application/atom+xml
//   - image/svg+xml
//
// # Behavior
//
// The middleware automatically:
//
//   - Sets the Content-Encoding header to the selected algorithm
//   - Adds Vary: Accept-Encoding header for proper caching
//   - Removes Content-Length header when compressing
//   - Skips compression if:
//   - Client doesn't support the encoding
//   - Response is already encoded
//   - Response size is below MinSize threshold
//   - Content-Type is not in the allowed list
//
// # Best Practices
//
//   - Use level 6 for most applications (good balance of speed and compression)
//   - Set MinSize to at least 1024 bytes to avoid overhead on small responses
//   - Only compress text-based content types (binary content rarely benefits)
//   - Place the compress middleware before other response-modifying middleware
//
// # Performance Considerations
//
// The middleware uses sync.Pool to reuse compression writers, which significantly
// reduces memory allocations and garbage collection pressure. The buffering strategy
// ensures that small responses are not compressed, avoiding unnecessary CPU overhead.
//
// For streaming responses, the middleware supports http.Flusher to allow chunked
// compression of long-running responses.
package compress
