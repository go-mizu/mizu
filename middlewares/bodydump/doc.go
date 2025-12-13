// Package bodydump provides middleware for capturing and dumping HTTP request and response bodies.
//
// # Overview
//
// The bodydump middleware intercepts request and response bodies, allowing you to inspect, log,
// or process them for debugging and monitoring purposes. It captures bodies up to a configurable
// size limit and provides flexible filtering options.
//
// # Basic Usage
//
// Create a simple body dumper that logs both request and response:
//
//	app := mizu.New()
//	app.Use(bodydump.New(func(c *mizu.Ctx, reqBody, resBody []byte) {
//	    log.Printf("Request: %s", reqBody)
//	    log.Printf("Response: %s", resBody)
//	}))
//
// # Configuration Options
//
// The Options struct provides fine-grained control over body dumping behavior:
//
//	app.Use(bodydump.WithOptions(bodydump.Options{
//	    Request:          true,                              // Dump request bodies
//	    Response:         true,                              // Dump response bodies
//	    MaxSize:          64 * 1024,                         // Maximum 64KB per body
//	    Handler:          dumpHandler,                       // Callback function
//	    SkipPaths:        []string{"/health", "/metrics"},   // Skip specific paths
//	    SkipContentTypes: []string{"image/jpeg"},            // Skip content types
//	}))
//
// # Specialized Dumpers
//
// Use convenience functions for specific use cases:
//
//	// Dump only requests
//	app.Use(bodydump.RequestOnly(func(c *mizu.Ctx, body []byte) {
//	    log.Printf("Request: %s", body)
//	}))
//
//	// Dump only responses
//	app.Use(bodydump.ResponseOnly(func(c *mizu.Ctx, body []byte) {
//	    log.Printf("Response: %s", body)
//	}))
//
// # Implementation Details
//
// Request Body Capture:
// The middleware reads the request body using io.LimitReader to respect the MaxSize limit,
// then restores it with io.NopCloser so downstream handlers can still access it.
//
// Response Body Capture:
// A custom responseCapture wrapper intercepts ResponseWriter.Write() calls to capture
// the response body while simultaneously writing to the original response writer.
//
// # Performance Considerations
//
//   - Path and content-type filtering uses hash maps for O(1) lookups
//   - Body capture is limited by MaxSize to prevent memory exhaustion
//   - Request bodies are read once and restored for handler use
//   - Response capture adds minimal overhead through write-through operation
//
// # Security Warning
//
// Body dumping may capture sensitive data such as passwords, tokens, or personal information.
// Never use this middleware in production environments without implementing proper filtering
// and security measures. Consider:
//
//   - Filtering sensitive fields from dumped bodies
//   - Skipping authentication endpoints
//   - Using content-type filters to avoid binary data
//   - Implementing size limits to prevent memory issues
//
// # Default Values
//
//   - MaxSize: 64KB (65,536 bytes)
//   - Request: true (enabled)
//   - Response: true (enabled)
//
// # Examples
//
// Skip specific paths:
//
//	app.Use(bodydump.WithOptions(bodydump.Options{
//	    Handler:   dumpHandler,
//	    SkipPaths: []string{"/health", "/metrics"},
//	}))
//
// Skip binary content types:
//
//	app.Use(bodydump.WithOptions(bodydump.Options{
//	    Handler:          dumpHandler,
//	    SkipContentTypes: []string{"image/jpeg", "image/png", "application/octet-stream"},
//	}))
//
// Limit capture size:
//
//	app.Use(bodydump.WithOptions(bodydump.Options{
//	    Handler: dumpHandler,
//	    MaxSize: 1024, // Only capture first 1KB
//	}))
package bodydump
