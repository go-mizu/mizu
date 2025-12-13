// Package timeout provides request timeout middleware for the Mizu web framework.
//
// The timeout middleware enforces a maximum duration for request processing.
// If a handler takes longer than the specified timeout, the request is cancelled
// and an error response is returned to the client.
//
// # Basic Usage
//
// Create a timeout middleware with a specific duration:
//
//	app := mizu.New()
//	app.Use(timeout.New(30 * time.Second))
//
// # Configuration Options
//
// The middleware can be configured using the Options struct:
//
//	app.Use(timeout.WithOptions(timeout.Options{
//	    Timeout:      10 * time.Second,
//	    ErrorMessage: "Request timeout",
//	    ErrorHandler: func(c *mizu.Ctx) error {
//	        return c.JSON(504, map[string]string{
//	            "error": "Request timed out",
//	        })
//	    },
//	}))
//
// # How It Works
//
// The middleware uses Go's context.WithTimeout to create a deadline for each request:
//
//  1. Creates a context with the specified timeout duration
//  2. Replaces the request's context with the timeout context
//  3. Executes the next handler in a goroutine
//  4. Waits for either:
//     - The handler to complete (returns the result)
//     - The timeout to expire (returns 503 Service Unavailable)
//
// # Context Cancellation
//
// Handlers can check if a timeout has occurred by monitoring the context:
//
//	app.Get("/data", func(c *mizu.Ctx) error {
//	    for i := 0; i < 100; i++ {
//	        select {
//	        case <-c.Context().Done():
//	            return c.Context().Err() // Timeout occurred
//	        default:
//	            processItem(i)
//	        }
//	    }
//	    return c.JSON(200, results)
//	})
//
// # Per-Route Timeouts
//
// Different routes can have different timeout durations:
//
//	app.Use(timeout.New(30 * time.Second)) // Global default
//	app.Post("/export", exportHandler, timeout.New(5*time.Minute)) // Long timeout
//	app.Get("/health", healthHandler, timeout.New(5*time.Second))   // Short timeout
//
// # Default Values
//
// If timeout options are not fully specified, the following defaults apply:
//
//   - Timeout Duration: 30 seconds (if Timeout <= 0)
//   - Error Message: "Service Unavailable"
//   - HTTP Status: 503 Service Unavailable (if no ErrorHandler provided)
//
// # Concurrency Safety
//
// The middleware is safe for concurrent use. Each request gets its own:
//   - Timeout context isolated from other requests
//   - Buffered channel (size 1) to prevent goroutine leaks
//   - Deferred cancel function to properly release resources
package timeout
