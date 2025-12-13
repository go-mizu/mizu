// Package concurrency provides middleware for limiting concurrent request processing in Mizu applications.
//
// The concurrency middleware uses a semaphore pattern to control the number of requests
// processed simultaneously, preventing resource exhaustion and ensuring stable performance
// under high load conditions.
//
// # Overview
//
// This middleware is useful when you need to:
//   - Limit CPU-intensive operations
//   - Control database connection usage
//   - Prevent memory exhaustion
//   - Ensure predictable resource consumption
//
// # Basic Usage
//
// Create a concurrency limiter with a maximum number of concurrent requests:
//
//	app := mizu.New()
//
//	// Allow maximum 50 concurrent requests
//	app.Use(concurrency.New(50))
//
// # Custom Configuration
//
// Use WithOptions for more control over the middleware behavior:
//
//	app.Use(concurrency.WithOptions(concurrency.Options{
//	    Max: 50,
//	    ErrorHandler: func(c *mizu.Ctx) error {
//	        return c.JSON(503, map[string]string{
//	            "error": "Too many concurrent requests",
//	        })
//	    },
//	}))
//
// # Per-Route Limits
//
// Apply different limits to different route groups:
//
//	// Heavy endpoints with lower limit
//	heavy := app.Group("/heavy")
//	heavy.Use(concurrency.New(5))
//
//	// Light endpoints with higher limit
//	light := app.Group("/api")
//	light.Use(concurrency.New(100))
//
// # Variants
//
// The package provides three variants:
//
//  1. New/WithOptions - Non-blocking: Immediately rejects requests when at capacity
//  2. Blocking - Blocks requests until a slot becomes available
//  3. WithContext - Blocks while respecting context cancellation
//
// Example using the blocking variant:
//
//	// Wait for slot availability instead of rejecting
//	app.Use(concurrency.Blocking(50))
//
// Example using the context-aware variant:
//
//	// Respect request context cancellation
//	app.Use(concurrency.WithContext(50))
//
// # Implementation Details
//
// The middleware uses buffered channels as semaphores to control concurrency:
//   - A buffered channel with capacity equal to Max limits concurrent requests
//   - Non-blocking variant uses select with default case to reject immediately
//   - Blocking variant blocks on channel send until slot is available
//   - Deferred channel receive ensures slots are released even on panics
//
// # Error Handling
//
// When the concurrency limit is reached:
//   - Default: Returns 503 Service Unavailable with "Server at capacity" message
//   - Sets Retry-After header to "1" second
//   - Custom ErrorHandler: Full control over response format and status code
//
// Special cases:
//   - Max <= 0: All requests are immediately rejected
//   - WithContext: Returns context error if request context is cancelled while waiting
//
// # Best Practices
//
//   - Set limits based on measured resource capacity (CPU, memory, connections)
//   - Monitor concurrent request counts in production
//   - Use different limits for different workload types
//   - Combine with timeout middleware to prevent hung requests
//   - Consider using Blocking variant for critical endpoints that should never reject
//
// # Security Considerations
//
// The concurrency middleware helps prevent denial-of-service scenarios by:
//   - Limiting resource consumption during traffic spikes
//   - Preventing cascading failures from resource exhaustion
//   - Providing predictable degradation under overload
//
// However, note that:
//   - It does not prevent malicious actors from holding slots
//   - Should be combined with rate limiting for comprehensive protection
//   - Does not account for varying request resource costs
package concurrency
