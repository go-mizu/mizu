// Package requestsize provides middleware for tracking HTTP request sizes in Mizu applications.
//
// # Overview
//
// The requestsize middleware monitors and records the size of incoming HTTP requests,
// providing both the Content-Length header value and the actual number of bytes read
// from the request body. This is useful for bandwidth monitoring, metrics collection,
// and traffic pattern analysis.
//
// # Basic Usage
//
// Simple request size tracking:
//
//	app := mizu.New()
//	app.Use(requestsize.New())
//
//	app.Post("/upload", func(c *mizu.Ctx) error {
//	    info := requestsize.Get(c)
//	    log.Printf("Request size: %d bytes (Content-Length: %d)",
//	        info.BytesRead, info.ContentLength)
//	    return c.Text(200, "OK")
//	})
//
// # With Callback
//
// Execute custom logic when request processing completes:
//
//	app.Use(requestsize.WithCallback(func(c *mizu.Ctx, info *requestsize.Info) {
//	    metrics.RecordRequestSize(c.Request().URL.Path, info.BytesRead)
//	}))
//
// # Architecture
//
// The middleware uses a wrapping mechanism to track bytes read from the request body:
//
// 1. Wraps http.Request.Body with a trackingBody implementation
// 2. Intercepts Read() calls to count bytes
// 3. Stores size information in the request context
// 4. Executes optional callbacks after request processing
//
// # Helper Functions
//
// The package provides several convenience functions for retrieving size information:
//
//   - Get(c) - Returns the complete Info struct
//   - ContentLength(c) - Returns only the Content-Length header value
//   - BytesRead(c) - Returns only the actual bytes read
//
// # Use Cases
//
//   - Bandwidth monitoring and analysis
//   - Request size metrics collection
//   - Traffic pattern detection
//   - Integration with monitoring systems
//   - Logging request payload sizes
//
// # Performance Considerations
//
// The middleware adds minimal overhead:
//
//   - Single context value allocation
//   - Byte counting during existing Read() operations
//   - No buffering or additional memory allocation
//   - Deferred callback execution only if configured
//
// # Integration
//
// Works well with other Mizu middlewares:
//
//   - Combine with responsesize for complete bandwidth tracking
//   - Use with bodylimit for size enforcement
//   - Integrate with metrics middleware for centralized monitoring
//
// For more information and examples, see: https://github.com/go-mizu/mizu
package requestsize
