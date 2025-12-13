// Package fallback provides fallback response middleware for Mizu.
//
// The fallback middleware enables graceful degradation by providing alternative
// responses when primary handlers fail. It supports error handling, panic recovery,
// and status code-based fallbacks.
//
// # Basic Usage
//
// Create a simple fallback handler:
//
//	app := mizu.New()
//	app.Use(fallback.New(func(c *mizu.Ctx, err error) error {
//		return c.JSON(200, map[string]string{
//			"message": "Service temporarily unavailable",
//			"error":   err.Error(),
//		})
//	}))
//
// # Configuration Options
//
// The middleware supports various configuration options through the Options struct:
//
//   - Handler: Custom error handler function
//   - NotFoundHandler: Specific handler for 404 errors
//   - StatusCodes: Map of status codes to handlers
//   - CatchPanic: Enable panic recovery (default: false)
//   - DefaultMessage: Message for unhandled errors (default: "An error occurred")
//
// # Panic Recovery
//
// Enable panic recovery to catch and handle panics:
//
//	app.Use(fallback.WithOptions(fallback.Options{
//		CatchPanic: true,
//		Handler: func(c *mizu.Ctx, err error) error {
//			log.Printf("Panic caught: %v", err)
//			return c.Text(500, "Internal server error")
//		},
//	}))
//
// # Convenience Functions
//
// The package provides several convenience functions for common use cases:
//
//   - Default(message): Simple text fallback
//   - JSON(): JSON error responses
//   - Redirect(url, code): Redirect on error
//   - Chain(handlers...): Sequential fallback chain
//   - NotFound(handler): Custom 404 handler
//   - ForStatus(code, handler): Status code-specific handler
//
// # Chained Fallbacks
//
// Create a chain of fallback handlers with conditional logic:
//
//	app.Use(fallback.Chain(
//		func(c *mizu.Ctx, err error) (bool, error) {
//			if errors.Is(err, ErrDatabase) {
//				return true, c.JSON(503, map[string]string{
//					"error": "Database unavailable",
//				})
//			}
//			return false, nil
//		},
//		func(c *mizu.Ctx, err error) (bool, error) {
//			return true, c.Text(500, "Internal server error")
//		},
//	))
//
// # Implementation Details
//
// The middleware uses a wrapper pattern to intercept errors:
//
//  1. Wraps the next handler in the chain
//  2. Optionally sets up panic recovery using defer/recover
//  3. Executes the next handler
//  4. Catches any returned errors
//  5. Delegates to the appropriate fallback handler based on configuration
//
// For status code-based fallbacks, the middleware uses a responseCapture type
// that wraps http.ResponseWriter to intercept status codes before they're written.
//
// Panics are converted to errors using the internal panicError type, which handles
// both error and non-error panic values.
//
// # Thread Safety
//
// The middleware is safe for concurrent use. However, if using shared state
// in fallback handlers (like caches), ensure proper synchronization:
//
//	var cache atomic.Value
//	app.Use(fallback.New(func(c *mizu.Ctx, err error) error {
//		return c.JSON(200, cache.Load())
//	}))
//
// # Best Practices
//
//   - Keep fallback responses lightweight and fast
//   - Log when fallback is triggered for monitoring
//   - Monitor fallback rate to detect service degradation
//   - Use cached data when possible for better performance
//   - Consider circuit breaker patterns for dependent services
//   - Set appropriate HTTP status codes in fallback responses
//
// # Performance Considerations
//
// The middleware adds minimal overhead:
//
//   - Error path: Only executed when errors occur
//   - Panic recovery: Small overhead from defer when enabled
//   - Response capture: Extra allocation for status code interception
//
// For high-performance scenarios, disable panic recovery if not needed and
// use error-based fallbacks instead of status code-based ones.
package fallback
