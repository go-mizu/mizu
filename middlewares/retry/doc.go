// Package retry provides automatic retry middleware for handling transient failures in Mizu applications.
//
// The retry middleware automatically retries failed requests with configurable backoff strategies,
// making it ideal for handling temporary network issues, service unavailability, and other transient errors.
//
// # Features
//
//   - Configurable retry attempts with exponential backoff
//   - Custom retry conditions based on status codes or errors
//   - Callback hooks for monitoring retry attempts
//   - Helper functions for common retry patterns
//   - Response writer wrapping to prevent premature response commits
//
// # Basic Usage
//
// Create a retry middleware with default settings (3 retries, 100ms initial delay):
//
//	app := mizu.New()
//	app.Use(retry.New())
//
// # Configuration
//
// Customize retry behavior using Options:
//
//	app.Use(retry.WithOptions(retry.Options{
//	    MaxRetries: 5,
//	    Delay:      100 * time.Millisecond,
//	    MaxDelay:   5 * time.Second,
//	    Multiplier: 2.0,
//	}))
//
// # Retry Conditions
//
// Control when retries occur using built-in helpers:
//
//	// Retry only on specific status codes
//	app.Use(retry.WithOptions(retry.Options{
//	    MaxRetries: 3,
//	    RetryIf:    retry.RetryOn(502, 503, 504),
//	}))
//
//	// Retry only on errors
//	app.Use(retry.WithOptions(retry.Options{
//	    MaxRetries: 3,
//	    RetryIf:    retry.RetryOnError(),
//	}))
//
// Or implement custom retry logic:
//
//	app.Use(retry.WithOptions(retry.Options{
//	    MaxRetries: 3,
//	    RetryIf: func(c *mizu.Ctx, err error, attempt int) bool {
//	        // Custom logic to determine if retry should occur
//	        return err != nil && attempt < 2
//	    },
//	}))
//
// # Monitoring Retries
//
// Use the OnRetry callback to log or track retry attempts:
//
//	app.Use(retry.WithOptions(retry.Options{
//	    MaxRetries: 3,
//	    OnRetry: func(c *mizu.Ctx, err error, attempt int) {
//	        log.Printf("Retrying request (attempt %d): %v", attempt, err)
//	    },
//	}))
//
// # Exponential Backoff
//
// The middleware implements exponential backoff by default:
//
//	delay = min(initialDelay * (multiplier ^ attempt), maxDelay)
//
// With default settings (100ms delay, 2.0 multiplier):
//   - Attempt 1: 100ms delay
//   - Attempt 2: 200ms delay
//   - Attempt 3: 400ms delay
//
// # Best Practices
//
//   - Use exponential backoff (multiplier > 1.0) to prevent thundering herd
//   - Set reasonable max retries (typically 3-5) to avoid excessive delays
//   - Only retry idempotent operations (GET, PUT, DELETE) to prevent duplicate side effects
//   - Consider adding jitter in distributed systems to prevent synchronized retries
//   - Use OnRetry callback to log retry attempts for debugging and monitoring
//
// # Implementation Details
//
// The middleware uses a custom retryResponseWriter that wraps the HTTP response writer
// to capture status codes without committing the response. This allows the middleware
// to retry requests even after handlers have attempted to write responses.
//
// Each retry iteration:
//  1. Sleeps for the calculated delay duration
//  2. Increases delay for next attempt using the multiplier
//  3. Calls OnRetry callback if configured
//  4. Wraps response writer to capture status
//  5. Invokes the next handler
//  6. Checks RetryIf condition to determine if retry should continue
//
// The middleware stops retrying when:
//   - The handler succeeds without error
//   - A successful status code (< 500) is returned
//   - The RetryIf function returns false
//   - Maximum retry attempts are exhausted
//
// # Helper Functions
//
//   - RetryOn(codes ...int): Creates a RetryIf function for specific HTTP status codes
//   - RetryOnError(): Creates a RetryIf function that retries only on errors
//   - NoRetry(): Creates a RetryIf function that disables all retries
//
// # Performance Considerations
//
// The middleware blocks the goroutine during retry delays using time.Sleep.
// For high-concurrency applications, ensure retry delays and max attempts are
// configured appropriately to avoid excessive resource consumption.
//
// Response writer wrapping adds minimal overhead and does not allocate additional
// memory beyond the wrapper struct itself.
package retry
