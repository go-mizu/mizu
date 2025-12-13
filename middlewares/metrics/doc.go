// Package metrics provides simple metrics collection middleware for the Mizu web framework.
//
// Overview
//
// The metrics middleware tracks HTTP request metrics including request counts, error rates,
// latencies, status codes, and per-path statistics. It provides thread-safe metrics collection
// with minimal performance overhead using atomic operations.
//
// Key Features:
//   - Request and error counting
//   - Active request tracking (in-flight requests)
//   - Average request duration calculation
//   - Per-status-code tracking
//   - Per-path request counting
//   - JSON and Prometheus format export
//   - Thread-safe concurrent access
//
// Basic Usage
//
// Create a metrics instance and register the middleware:
//
//	m := metrics.NewMetrics()
//	app.Use(m.Middleware())
//
// Or use the convenience function that returns both:
//
//	m, middleware := metrics.New()
//	app.Use(middleware)
//
// Expose metrics via HTTP endpoint:
//
//	// JSON format
//	app.Get("/metrics", m.Handler())
//
//	// Prometheus format
//	app.Get("/metrics", m.Prometheus())
//
// Accessing Statistics
//
// Retrieve current metrics programmatically:
//
//	stats := m.Stats()
//	fmt.Printf("Total requests: %d\n", stats.RequestCount)
//	fmt.Printf("Error rate: %.2f%%\n", float64(stats.ErrorCount)/float64(stats.RequestCount)*100)
//	fmt.Printf("Average duration: %.2fms\n", stats.AverageDurationMs)
//	fmt.Printf("Active requests: %d\n", stats.ActiveRequests)
//
// Reset Metrics
//
// Clear all metrics back to zero:
//
//	m.Reset()
//
// Thread Safety
//
// All operations are thread-safe. The middleware uses atomic operations for counters
// and a read-write mutex for maps to ensure correct concurrent behavior without
// sacrificing performance.
//
// Metrics Collected
//
// The middleware automatically collects:
//   - RequestCount: Total number of HTTP requests processed
//   - ErrorCount: Number of requests with errors (status >= 400 or handler error)
//   - TotalDuration: Cumulative request processing time in nanoseconds
//   - ActiveRequests: Current number of in-flight requests
//   - StatusCodes: Map of HTTP status codes to their counts
//   - PathCounts: Map of URL paths to their request counts
//
// Output Formats
//
// JSON format (via Handler()):
//
//	{
//	  "request_count": 1000,
//	  "error_count": 42,
//	  "active_requests": 5,
//	  "average_duration_ms": 12.34,
//	  "status_codes": {
//	    "200": 900,
//	    "404": 30,
//	    "500": 12
//	  },
//	  "path_counts": {
//	    "/api/users": 450,
//	    "/api/posts": 550
//	  }
//	}
//
// Prometheus format (via Prometheus()):
//
//	# HELP http_requests_total Total HTTP requests
//	# TYPE http_requests_total counter
//	http_requests_total 1000
//	# HELP http_errors_total Total HTTP errors
//	# TYPE http_errors_total counter
//	http_errors_total 42
//	# HELP http_active_requests Current active requests
//	# TYPE http_active_requests gauge
//	http_active_requests 5
//
// Implementation Details
//
// The middleware uses a custom statusCapture wrapper that implements http.ResponseWriter
// to intercept status codes. This allows accurate tracking even when status codes are
// set by downstream handlers or frameworks.
//
// Performance Considerations:
//   - Atomic operations for main counters (no mutex overhead)
//   - RWMutex for maps (allows concurrent reads)
//   - Minimal memory allocations
//   - Custom itoa() to avoid fmt package overhead
//   - Non-intrusive monitoring (doesn't modify requests/responses)
//
// Example
//
// Complete example with metrics collection and exposure:
//
//	package main
//
//	import (
//	    "github.com/go-mizu/mizu"
//	    "github.com/go-mizu/mizu/middlewares/metrics"
//	)
//
//	func main() {
//	    app := mizu.New()
//
//	    // Create and register metrics middleware
//	    m, middleware := metrics.New()
//	    app.Use(middleware)
//
//	    // Application routes
//	    app.Get("/", func(c *mizu.Ctx) error {
//	        return c.Text(200, "Hello, World!")
//	    })
//
//	    app.Get("/api/users", func(c *mizu.Ctx) error {
//	        return c.JSON(200, map[string]string{"status": "ok"})
//	    })
//
//	    // Metrics endpoints
//	    app.Get("/metrics", m.Handler())           // JSON format
//	    app.Get("/metrics/prometheus", m.Prometheus()) // Prometheus format
//
//	    app.Listen(":8080")
//	}
package metrics
