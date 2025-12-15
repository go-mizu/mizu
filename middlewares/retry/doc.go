/*
Package retry provides automatic retry middleware for handling transient failures in Mizu applications.

The retry middleware retries failed requests with configurable backoff strategies.
It is designed for temporary failures such as network hiccups, upstream unavailability,
or short-lived service errors.

# Features

  - Configurable retry attempts with exponential backoff
  - Custom retry conditions based on errors or HTTP status codes
  - Callback hooks for observing retry attempts
  - Helper functions for common retry policies
  - Buffered response handling to avoid premature response commits

# Basic Usage

Create a retry middleware with default settings (3 retries, 100ms initial delay):

	app := mizu.NewRouter()
	app.Use(retry.New())

# Configuration

Customize retry behavior using Options:

	app.Use(retry.WithOptions(retry.Options{
		MaxRetries: 5,
		Delay:      100 * time.Millisecond,
		MaxDelay:   5 * time.Second,
		Multiplier: 2.0,
	}))

# Retry Conditions

Control when retries occur using built-in helpers:

	// Retry only on specific status codes
	app.Use(retry.WithOptions(retry.Options{
		MaxRetries: 3,
		RetryIf:    retry.RetryOn(502, 503, 504),
	}))

	// Retry only on errors
	app.Use(retry.WithOptions(retry.Options{
		MaxRetries: 3,
		RetryIf:    retry.RetryOnError(),
	}))

Or implement custom retry logic:

	app.Use(retry.WithOptions(retry.Options{
		MaxRetries: 3,
		RetryIf: func(c *mizu.Ctx, err error, attempt int) bool {
			// Custom logic to determine whether a retry should occur
			return err != nil && attempt < 2
		},
	}))

# Monitoring Retries

Use the OnRetry callback to log or observe retry attempts:

	app.Use(retry.WithOptions(retry.Options{
		MaxRetries: 3,
		OnRetry: func(c *mizu.Ctx, err error, attempt int) {
			log.Printf("retry attempt %d: %v", attempt, err)
		},
	}))

# Exponential Backoff

The middleware applies exponential backoff by default:

	delay = min(initialDelay * (multiplier ^ attempt), maxDelay)

With default settings (100ms delay, multiplier 2.0):

  - Attempt 1: 100ms delay
  - Attempt 2: 200ms delay
  - Attempt 3: 400ms delay

# Best Practices

  - Prefer exponential backoff with multiplier greater than 1.0
  - Limit retries to a small number, typically 3 to 5
  - Retry only idempotent operations such as GET, PUT, or DELETE
  - Consider adding jitter in distributed systems to avoid synchronized retries
  - Use OnRetry to log attempts for debugging and monitoring

# Implementation Details

The middleware uses a buffered response writer that captures headers, status code,
and body without writing to the underlying ResponseWriter immediately.

This allows the middleware to safely retry even if the handler attempted to write
a response, while guaranteeing that the final response is written exactly once.

Each retry attempt:

 1. Sleeps for the current backoff delay
 2. Increases the delay for the next attempt
 3. Invokes the OnRetry callback, if configured
 4. Wraps the response writer to buffer output
 5. Calls the next handler
 6. Evaluates RetryIf to decide whether to retry or commit the response

Retrying stops when:

  - The handler succeeds without error
  - A non-retriable status code is produced
  - RetryIf returns false
  - The maximum number of retries is reached

# Helper Functions

  - RetryOn(codes ...int) creates a RetryIf function for specific HTTP status codes
  - RetryOnError() retries only when an error is returned
  - NoRetry() disables retries entirely

# Performance Considerations

The middleware blocks the current goroutine during backoff using time.Sleep.
Configure retry limits and delays carefully in high-concurrency environments.

Response buffering adds minimal overhead and avoids unsafe mutations of the
underlying ResponseWriter.
*/
package retry
