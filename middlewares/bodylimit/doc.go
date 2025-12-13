// Package bodylimit provides middleware for limiting HTTP request body sizes in Mizu applications.
//
// The bodylimit middleware protects applications against large payload attacks and resource
// exhaustion by enforcing maximum request body sizes. It uses a two-stage validation approach
// for efficient and reliable enforcement.
//
// # Features
//
//   - Content-Length header pre-validation for early rejection
//   - Runtime enforcement using http.MaxBytesReader
//   - Customizable error handlers
//   - Helper functions for common size units (KB, MB, GB)
//   - Configurable limits per route or globally
//
// # Basic Usage
//
// Apply a simple body limit to all routes:
//
//	app := mizu.New()
//	app.Use(bodylimit.New(bodylimit.MB(10))) // 10MB limit
//
//	app.Post("/upload", func(c *mizu.Ctx) error {
//	    // Handle upload
//	    return c.Text(200, "Upload successful")
//	})
//
// # Custom Error Handler
//
// Provide custom error responses when limits are exceeded:
//
//	app.Use(bodylimit.WithHandler(bodylimit.MB(5), func(c *mizu.Ctx) error {
//	    return c.JSON(413, map[string]string{
//	        "error": "Request body too large",
//	        "max_size": "5MB",
//	    })
//	}))
//
// # Advanced Configuration
//
// Use WithOptions for full control over middleware behavior:
//
//	app.Use(bodylimit.WithOptions(bodylimit.Options{
//	    Limit: bodylimit.MB(20),
//	    ErrorHandler: func(c *mizu.Ctx) error {
//	        log.Printf("Body limit exceeded from %s", c.IP())
//	        return c.Text(413, "Payload too large")
//	    },
//	}))
//
// # Per-Route Limits
//
// Apply different limits to specific routes:
//
//	// Global limit: 1MB
//	app.Use(bodylimit.New(bodylimit.MB(1)))
//
//	// File upload route: 100MB
//	app.Post("/upload", uploadHandler, bodylimit.New(bodylimit.MB(100)))
//
//	// Avatar upload: 2MB
//	app.Post("/avatar", avatarHandler, bodylimit.New(bodylimit.MB(2)))
//
// # Size Helper Functions
//
// The package provides convenient functions for specifying sizes:
//
//	bodylimit.KB(500)  // 500 kilobytes = 512,000 bytes
//	bodylimit.MB(10)   // 10 megabytes = 10,485,760 bytes
//	bodylimit.GB(1)    // 1 gigabyte = 1,073,741,824 bytes
//
// # Implementation Details
//
// The middleware enforces limits in two stages:
//
// 1. Pre-flight validation: Checks the Content-Length header before reading the body.
// If the header indicates the body exceeds the limit, returns 413 immediately.
//
// 2. Runtime enforcement: Wraps the request body with http.MaxBytesReader to enforce
// the limit during read operations. This handles cases where Content-Length is unknown
// or incorrect.
//
// # Default Behavior
//
// When no limit is specified or the limit is set to 0 or negative, the middleware
// defaults to 1MB (1 << 20 bytes). The default error response is HTTP 413 with
// plain text message "Request Entity Too Large".
//
// # Security Considerations
//
// - Always set appropriate limits based on your application's requirements
// - Use smaller limits for JSON API endpoints (typically KB(100) to MB(1))
// - Use larger limits only for file upload endpoints
// - Monitor rejected requests to detect potential attacks
// - Consider implementing rate limiting alongside body limiting
//
// # Performance
//
// The middleware is highly efficient:
//   - Content-Length validation happens before body reading
//   - http.MaxBytesReader provides minimal overhead
//   - No buffering of request bodies
//   - Suitable for high-throughput applications
package bodylimit
