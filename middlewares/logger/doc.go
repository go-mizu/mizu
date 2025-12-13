// Package logger provides HTTP request logging middleware for the Mizu framework.
//
// The logger middleware captures and logs HTTP request details including method,
// path, status code, response time, client IP, and custom fields. It supports
// customizable output formats and destinations.
//
// # Features
//
//   - Configurable log format with template tags
//   - Custom output destination (stdout, files, etc.)
//   - Request filtering with Skip function
//   - Client IP detection with proxy support
//   - Response size tracking
//   - Request timing measurement
//
// # Basic Usage
//
//	app := mizu.New()
//	app.Use(logger.New())
//
//	app.Get("/", func(c *mizu.Ctx) error {
//	    return c.Text(200, "Hello!")
//	})
//	// Output: 2024/01/15 - 10:30:00 | 200 | 1.2ms | 127.0.0.1 | GET /
//
// # Custom Format
//
// The Format option accepts a template string with the following tags:
//
//   - ${time} - Current timestamp
//   - ${status} - HTTP status code
//   - ${method} - HTTP method (GET, POST, etc.)
//   - ${path} - Request path
//   - ${latency} - Request duration
//   - ${ip} - Client IP address
//   - ${host} - Request host
//   - ${protocol} - HTTP protocol version
//   - ${referer} - Referer header
//   - ${user_agent} - User-Agent header
//   - ${bytes_in} - Request body size
//   - ${bytes_out} - Response body size
//   - ${query} - URL query string
//   - ${header:X-Name} - Custom header value
//   - ${form:field} - Form field value
//
// Example with custom format:
//
//	app.Use(logger.WithOptions(logger.Options{
//	    Format: "[${method}] ${path} -> ${status}\n",
//	}))
//	// Output: [GET] /api/users -> 200
//
// # JSON Logging
//
// For structured logging and log aggregation:
//
//	app.Use(logger.WithOptions(logger.Options{
//	    Format: `{"method":"${method}","path":"${path}","status":${status},"latency":"${latency}"}` + "\n",
//	}))
//
// # File Output
//
// To write logs to a file:
//
//	file, _ := os.OpenFile("access.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
//	defer file.Close()
//
//	app.Use(logger.WithOptions(logger.Options{
//	    Output: file,
//	}))
//
// # Filtering Requests
//
// Use the Skip function to exclude certain requests from logging:
//
//	app.Use(logger.WithOptions(logger.Options{
//	    Skip: func(c *mizu.Ctx) bool {
//	        return c.Request().URL.Path == "/health"
//	    },
//	}))
//
// # Client IP Detection
//
// The middleware intelligently detects the client IP address:
//
//  1. Checks X-Forwarded-For header (uses first IP if multiple)
//  2. Checks X-Real-IP header
//  3. Falls back to RemoteAddr
//
// This ensures accurate IP logging when behind proxies or load balancers.
//
// # Performance
//
// The logger middleware is designed for minimal overhead:
//
//   - Zero allocations for skipped requests
//   - Single response writer wrapper
//   - Streaming-friendly (no response buffering)
//   - Efficient string replacement for format tags
//
// # Thread Safety
//
// The logger middleware is safe for concurrent use. Each request gets its own
// response writer wrapper and timing measurements.
package logger
