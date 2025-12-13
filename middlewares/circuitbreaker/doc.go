// Package circuitbreaker implements the circuit breaker pattern middleware for Mizu.
//
// The circuit breaker pattern prevents cascading failures by monitoring request failures
// and temporarily blocking requests when error thresholds are exceeded, giving the system
// time to recover.
//
// # Overview
//
// The circuit breaker operates as a state machine with three states:
//
//   - Closed: Normal operation. Requests pass through to the handler. Failures are counted.
//   - Open: Protective state. All requests are immediately rejected without calling the handler.
//   - Half-Open: Recovery testing. A limited number of requests are allowed to test if the system has recovered.
//
// # State Transitions
//
// The circuit breaker transitions between states based on failures and timeouts:
//
//   - Closed -> Open: When the number of consecutive failures reaches the Threshold.
//   - Open -> Half-Open: After the Timeout period expires.
//   - Half-Open -> Closed: When MaxRequests consecutive successful requests complete.
//   - Half-Open -> Open: When any request fails during the half-open state.
//
// # Basic Usage
//
// Use the default configuration with New():
//
//	app := mizu.New()
//	app.Use(circuitbreaker.New())
//
// This creates a circuit breaker that:
//   - Opens after 5 consecutive failures (Threshold)
//   - Stays open for 30 seconds (Timeout)
//   - Allows 1 request in half-open state (MaxRequests)
//
// # Custom Configuration
//
// Configure the circuit breaker with WithOptions():
//
//	app.Use(circuitbreaker.WithOptions(circuitbreaker.Options{
//	    Threshold:   10,                    // Open after 10 failures
//	    Timeout:     time.Minute,           // Stay open for 1 minute
//	    MaxRequests: 3,                     // Allow 3 test requests in half-open
//	    OnStateChange: func(from, to State) {
//	        log.Printf("Circuit: %s -> %s", from, to)
//	    },
//	    IsFailure: func(err error) bool {
//	        // Only count 5xx errors as failures
//	        return err != nil && isServerError(err)
//	    },
//	    ErrorHandler: func(c *mizu.Ctx) error {
//	        return c.JSON(503, map[string]string{
//	            "error": "Service temporarily unavailable",
//	        })
//	    },
//	}))
//
// # Failure Detection
//
// By default, any non-nil error from the handler is considered a failure.
// Use the IsFailure option to customize failure detection:
//
//	app.Use(circuitbreaker.WithOptions(circuitbreaker.Options{
//	    IsFailure: func(err error) bool {
//	        // Don't count client errors (4xx) as failures
//	        var httpErr *HTTPError
//	        if errors.As(err, &httpErr) && httpErr.Code < 500 {
//	            return false
//	        }
//	        return err != nil
//	    },
//	}))
//
// # State Monitoring
//
// Monitor state changes for alerting and observability:
//
//	app.Use(circuitbreaker.WithOptions(circuitbreaker.Options{
//	    OnStateChange: func(from, to circuitbreaker.State) {
//	        if to == circuitbreaker.StateOpen {
//	            // Alert operations team
//	            metrics.IncrementCircuitOpen()
//	            alerting.SendAlert("Circuit breaker opened")
//	        }
//	    },
//	}))
//
// # Per-Route Circuit Breakers
//
// Different routes can have independent circuit breakers:
//
//	externalAPI := circuitbreaker.WithOptions(circuitbreaker.Options{
//	    Threshold: 3,
//	    Timeout:   time.Minute,
//	})
//
//	database := circuitbreaker.WithOptions(circuitbreaker.Options{
//	    Threshold: 5,
//	    Timeout:   30 * time.Second,
//	})
//
//	app.Get("/api/external", externalHandler, externalAPI)
//	app.Get("/api/data", dataHandler, database)
//
// # Error Handling
//
// When the circuit is open, requests are rejected with a 503 Service Unavailable response
// by default. Customize this with the ErrorHandler option:
//
//	app.Use(circuitbreaker.WithOptions(circuitbreaker.Options{
//	    ErrorHandler: func(c *mizu.Ctx) error {
//	        return c.JSON(503, map[string]string{
//	            "error":   "Service unavailable",
//	            "message": "Please try again later",
//	            "retry":   "30s",
//	        })
//	    },
//	}))
//
// # Thread Safety
//
// The circuit breaker is thread-safe and uses a mutex to protect concurrent access
// to internal state, counters, and timestamps. It can safely handle concurrent requests
// from multiple goroutines.
//
// # Best Practices
//
//   - Set Threshold based on your normal error rates and tolerance
//   - Use Timeout values that give your system enough time to recover
//   - Monitor state changes with OnStateChange for alerting
//   - Use different circuit breakers for different external dependencies
//   - Implement fallback responses with ErrorHandler when possible
//   - Use IsFailure to distinguish between client errors and server failures
package circuitbreaker
