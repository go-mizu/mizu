// Package audit provides comprehensive HTTP request/response audit logging middleware for Mizu.
//
// # Overview
//
// The audit middleware captures detailed information about every HTTP request and response
// for compliance, security monitoring, debugging, and analytics purposes. It provides a
// flexible handler pattern that supports synchronous, asynchronous, and batched processing
// of audit entries.
//
// # Basic Usage
//
// Create audit middleware with a simple handler function:
//
//	app := mizu.New()
//	app.Use(audit.New(func(entry *audit.Entry) {
//	    log.Printf("%s %s -> %d (%v)", entry.Method, entry.Path, entry.Status, entry.Latency)
//	}))
//
// # Configuration
//
// Use WithOptions for advanced configuration:
//
//	app.Use(audit.WithOptions(audit.Options{
//	    Handler: func(entry *audit.Entry) {
//	        saveToDatabase(entry)
//	    },
//	    IncludeRequestBody: true,
//	    MaxBodySize:        4096,
//	    RequestIDHeader:    "X-Request-ID",
//	    Skip: func(c *mizu.Ctx) bool {
//	        return c.Request().URL.Path == "/health"
//	    },
//	    Metadata: func(c *mizu.Ctx) map[string]string {
//	        return map[string]string{
//	            "version": "1.0",
//	            "env":     os.Getenv("ENV"),
//	        }
//	    },
//	}))
//
// # Audit Entry
//
// Each audit entry contains:
//   - Timestamp: Request start time
//   - RequestID: Value from configurable header (default: X-Request-ID)
//   - Method: HTTP method (GET, POST, etc.)
//   - Path: Request URL path
//   - Query: Raw query string
//   - RemoteAddr: Client IP address and port
//   - UserAgent: User-Agent header value
//   - RequestBody: Request body content (if enabled)
//   - Status: HTTP response status code
//   - Latency: Request processing duration
//   - Error: Error message (if handler returned an error)
//   - Metadata: Custom key-value pairs
//
// # Handler Patterns
//
// Synchronous Handler (simple, blocks request):
//
//	app.Use(audit.New(func(entry *audit.Entry) {
//	    log.Println(entry)
//	}))
//
// Asynchronous Channel Handler (non-blocking):
//
//	ch := make(chan *audit.Entry, 100)
//	go func() {
//	    for entry := range ch {
//	        saveToDatabase(entry)
//	    }
//	}()
//	app.Use(audit.New(audit.ChannelHandler(ch)))
//
// Buffered Handler (batch processing):
//
//	handler := audit.NewBufferedHandler(
//	    100,         // batch size
//	    time.Minute, // flush interval
//	    func(entries []*audit.Entry) {
//	        bulkInsert(entries)
//	    },
//	)
//	defer handler.Close()
//	app.Use(audit.New(handler.Handler()))
//
// # Request Body Capture
//
// Request body capture is disabled by default for performance and privacy. When enabled,
// the middleware reads up to MaxBodySize bytes from the request body and restores it
// for downstream handlers:
//
//	app.Use(audit.WithOptions(audit.Options{
//	    Handler:            logEntry,
//	    IncludeRequestBody: true,
//	    MaxBodySize:        4096, // limit body capture size
//	}))
//
// # Performance Considerations
//
// For high-traffic applications:
//   - Use ChannelHandler or BufferedHandler to avoid blocking requests
//   - Set reasonable MaxBodySize to prevent memory issues
//   - Use Skip function to exclude health checks and metrics endpoints
//   - Consider buffering writes when logging to databases or files
//
// # Security and Compliance
//
// The audit middleware is designed for compliance and security monitoring:
//   - Captures complete request/response metadata
//   - Supports request body logging (opt-in)
//   - Includes request IDs for correlation
//   - Records errors and status codes
//   - Extensible metadata for custom fields
//
// Always be mindful of sensitive data when capturing request bodies or custom metadata.
//
// # Thread Safety
//
// The middleware is safe for concurrent use. The BufferedHandler uses mutex protection
// for internal state management.
package audit
