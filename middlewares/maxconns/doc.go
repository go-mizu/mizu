// Package maxconns provides middleware for limiting concurrent connections in Mizu applications.
//
// Overview
//
// The maxconns middleware protects server resources by enforcing limits on the maximum
// number of concurrent connections. It supports both global connection limits and
// per-IP connection limits to prevent resource exhaustion and mitigate DoS attacks.
//
// # Basic Usage
//
// Simple global connection limit:
//
//	app := mizu.New()
//	app.Use(maxconns.New(1000)) // Limit to 1000 concurrent connections
//
// # Configuration Options
//
// The middleware can be configured using the Options struct:
//
//	app.Use(maxconns.WithOptions(maxconns.Options{
//	    Max: 1000,              // Maximum concurrent connections
//	    PerIP: 10,              // Maximum connections per IP address
//	    ErrorHandler: func(c *mizu.Ctx) error {
//	        return c.JSON(503, map[string]string{
//	            "error": "Server at capacity",
//	        })
//	    },
//	}))
//
// # Per-IP Limiting
//
// Limit connections per IP address:
//
//	app.Use(maxconns.PerIP(5)) // Max 5 concurrent connections per IP
//
// # Connection Monitoring
//
// Use the Counter type for monitoring active connections:
//
//	counter := maxconns.NewCounter(1000)
//	app.Use(counter.Middleware())
//
//	app.Get("/stats", func(c *mizu.Ctx) error {
//	    return c.JSON(200, map[string]int64{
//	        "active": counter.Current(),
//	        "max":    counter.Max(),
//	    })
//	})
//
// # Implementation Details
//
// The middleware uses atomic operations for lock-free global connection counting
// and read-write mutexes for per-IP tracking. This provides high performance with
// minimal contention in concurrent scenarios.
//
// Global connection tracking:
//   - Atomic operations (LoadInt64/AddInt64) for thread-safe counting
//   - No mutex overhead for global limit checks
//   - Counter automatically decremented via defer on request completion
//
// Per-IP connection tracking:
//   - Map-based storage protected by sync.RWMutex
//   - Automatic cleanup of zero-count entries
//   - IP extraction from X-Forwarded-For, X-Real-IP, or RemoteAddr
//
// # Error Handling
//
// By default, when the connection limit is reached:
//   - Returns HTTP 503 Service Unavailable
//   - Sets "Retry-After: 60" header
//   - Returns plain text message "Too many connections"
//
// Custom error handlers can override this behavior through the Options.ErrorHandler field.
//
// # Thread Safety
//
// All functions and methods in this package are safe for concurrent use.
// The middleware safely handles concurrent requests without data races.
package maxconns
