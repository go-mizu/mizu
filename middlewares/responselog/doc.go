// Package responselog provides HTTP response logging middleware for Mizu framework.
//
// The responselog middleware captures and logs HTTP response details including status codes,
// response times, body content, and headers. It's designed for debugging, monitoring, and
// performance tracking in development and production environments.
//
// # Features
//
//   - Response body logging with configurable size limits
//   - Response header logging
//   - Automatic request duration tracking
//   - Response size tracking
//   - Path-based filtering to skip specific endpoints
//   - Status code filtering to skip specific response codes
//   - Automatic log level selection (ERROR for 4xx/5xx, INFO otherwise)
//   - Integration with Go's standard log/slog package
//
// # Basic Usage
//
// The simplest way to use the middleware is with default settings:
//
//	app := mizu.New()
//	app.Use(responselog.New())
//
// This will log basic response information (method, path, status, duration, size) to stdout.
//
// # Custom Configuration
//
// For more control, use WithOptions to configure the middleware:
//
//	app.Use(responselog.WithOptions(responselog.Options{
//		Logger:      customLogger,      // Custom slog.Logger instance
//		LogBody:     true,               // Enable body logging
//		LogHeaders:  true,               // Enable header logging
//		MaxBodySize: 8192,               // Log up to 8KB of body
//		SkipPaths:   []string{"/health"}, // Don't log health checks
//		SkipStatuses: []int{200, 204},   // Don't log success responses
//	}))
//
// # Convenience Functions
//
// The package provides several convenience functions for common use cases:
//
// WithLogger creates middleware with a custom logger:
//
//	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
//	app.Use(responselog.WithLogger(logger))
//
// Full creates middleware that logs everything (body and headers):
//
//	app.Use(responselog.Full())
//
// ErrorsOnly creates middleware that only logs error responses:
//
//	app.Use(responselog.ErrorsOnly())
//
// # Response Capture Mechanism
//
// The middleware works by wrapping the http.ResponseWriter with a custom responseCapture
// type that intercepts WriteHeader and Write calls. This allows it to:
//
//   - Capture the HTTP status code
//   - Capture response body content up to MaxBodySize
//   - Measure request processing duration
//   - Track response size
//
// The original ResponseWriter is restored after processing to ensure compatibility
// with other middlewares and handlers.
//
// # Logging Behavior
//
// Response logs include the following fields:
//
//   - method: HTTP request method (GET, POST, etc.)
//   - path: Request URL path
//   - status: HTTP status code
//   - duration: Time taken to process the request (as a string)
//   - size: Response body size in bytes
//   - headers: Response headers (if LogHeaders is true)
//   - body: Response body content (if LogBody is true, truncated to MaxBodySize)
//
// Log entries are automatically assigned appropriate levels:
//
//   - Status codes >= 400 are logged as ERROR
//   - Status codes < 400 are logged as INFO
//
// # Performance Considerations
//
// Body logging can impact performance and memory usage, especially for large responses.
// Consider these best practices:
//
//   - Set MaxBodySize to a reasonable limit (default is 4KB)
//   - Disable body logging in production unless necessary for debugging
//   - Use SkipPaths to exclude high-traffic endpoints like health checks
//   - Use SkipStatuses to exclude successful responses if only errors are needed
//
// # Security Considerations
//
// When logging response bodies:
//
//   - Be aware that sensitive data (authentication tokens, personal information, etc.)
//     may be included in response bodies
//   - Ensure logs are stored securely and access is properly controlled
//   - Consider implementing custom filtering for sensitive fields
//   - Review logged data to ensure compliance with privacy regulations
//
// # Example: Production Configuration
//
// A typical production configuration might look like:
//
//	app.Use(responselog.WithOptions(responselog.Options{
//		Logger:       productionLogger,
//		LogBody:      false,              // Disable for performance/security
//		LogHeaders:   false,              // Disable for performance/security
//		SkipPaths:    []string{"/health", "/metrics"}, // Skip monitoring endpoints
//		SkipStatuses: []int{200, 201, 204}, // Only log non-success responses
//	}))
//
// This configuration focuses on logging errors and unusual responses while
// minimizing performance impact and security risks.
package responselog
