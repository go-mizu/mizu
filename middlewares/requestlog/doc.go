// Package requestlog provides detailed request logging middleware for Mizu.
//
// This middleware offers structured logging of HTTP requests using Go's log/slog
// package with configurable fields, sensitive data redaction, and flexible
// filtering options.
//
// # Features
//
//   - Structured logging with slog for easy log aggregation
//   - Configurable request field logging (headers, body, query params)
//   - Automatic sensitive header redaction (Authorization, Cookie, X-API-Key)
//   - Path and method-based skip logic
//   - Request body preservation for downstream handlers
//   - Request duration measurement
//   - Error logging with full context
//
// # Basic Usage
//
//	app := mizu.New()
//	app.Use(requestlog.New())
//
// # Custom Logger
//
//	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
//	app.Use(requestlog.WithLogger(logger))
//
// # Advanced Configuration
//
//	app.Use(requestlog.WithOptions(requestlog.Options{
//	    Logger:           customLogger,
//	    LogHeaders:       true,
//	    LogBody:          true,
//	    MaxBodySize:      8192, // 8KB
//	    SkipPaths:        []string{"/health", "/metrics"},
//	    SkipMethods:      []string{"OPTIONS"},
//	    SensitiveHeaders: []string{"Authorization", "X-API-Key", "X-Custom-Secret"},
//	}))
//
// # Convenience Functions
//
// The package provides several convenience constructors:
//
//   - Full(): Logs headers and body
//   - HeadersOnly(): Logs only headers
//   - BodyOnly(): Logs only body
//
// Example:
//
//	app.Use(requestlog.Full())
//
// # Logged Fields
//
// Standard fields logged for every request:
//   - method: HTTP method (GET, POST, etc.)
//   - path: Request URL path
//   - remote_addr: Client IP address
//   - duration: Request processing time
//
// Optional fields (when enabled):
//   - query: Query parameters (always logged if present)
//   - headers: Request headers (when LogHeaders: true)
//   - body: Request body (when LogBody: true)
//   - error: Error message (when handler returns error)
//
// # Security Considerations
//
// When enabling body or header logging, be aware of:
//   - Passwords and tokens in request bodies
//   - Authorization headers and cookies
//   - Personally Identifiable Information (PII)
//   - Compliance requirements (GDPR, HIPAA, etc.)
//
// Always configure SensitiveHeaders to protect credentials and use SkipPaths
// for authentication endpoints when body logging is enabled.
//
// # Performance
//
// The middleware is optimized for production use:
//   - Skip paths/methods are pre-computed into maps for O(1) lookup
//   - Body reading is limited by MaxBodySize (default 4KB)
//   - Request body is preserved for downstream handlers
//   - Minimal allocation overhead for standard requests
//
// # Example Output
//
// Text format (default):
//
//	time=2024-01-15T10:30:00.000Z level=INFO msg=request method=GET path=/api/users remote_addr=192.168.1.1 duration=1.234ms
//
// JSON format (with slog.NewJSONHandler):
//
//	{
//	  "time": "2024-01-15T10:30:00Z",
//	  "level": "INFO",
//	  "msg": "request",
//	  "method": "GET",
//	  "path": "/api/users",
//	  "remote_addr": "192.168.1.1",
//	  "duration": "1.234ms"
//	}
package requestlog
