// Package hedge provides hedged request middleware for Mizu.
//
// Hedged requests (also known as "backup requests" or "tail tolerance") are a latency
// reduction technique where the same request is sent to multiple backends after a delay,
// and the first response to complete is used. This helps reduce tail latency by hedging
// against slow responses.
//
// # Overview
//
// The hedge middleware sends parallel backup requests after a configurable delay,
// returning the first successful response. This is particularly effective for:
//   - Reducing p99 latency in systems with variable response times
//   - Handling slow backend responses caused by temporary issues
//   - Improving user experience for read-heavy workloads
//
// # Basic Usage
//
// The simplest way to use the hedge middleware is with default settings:
//
//	app := mizu.New()
//	app.Use(hedge.New())
//
// This uses a 100ms delay before triggering hedge requests.
//
// # Custom Configuration
//
// For more control, use WithOptions to configure delay, max hedges, and callbacks:
//
//	app.Use(hedge.WithOptions(hedge.Options{
//	    Delay:     100 * time.Millisecond,  // Wait 100ms before hedging
//	    MaxHedges: 2,                        // Allow up to 2 concurrent hedges
//	    Timeout:   30 * time.Second,         // Overall request timeout
//	    OnHedge: func(hedgeNum int) {
//	        log.Printf("Hedge %d triggered", hedgeNum)
//	    },
//	    OnComplete: func(hedgeNum int, duration time.Duration) {
//	        log.Printf("Request completed by hedge %d in %v", hedgeNum, duration)
//	    },
//	}))
//
// # Conditional Hedging
//
// Use ShouldHedge to control which requests are hedged. This is important for
// ensuring only idempotent operations are hedged:
//
//	app.Use(hedge.WithOptions(hedge.Options{
//	    Delay: 100 * time.Millisecond,
//	    ShouldHedge: func(r *http.Request) bool {
//	        // Only hedge GET requests (idempotent)
//	        return r.Method == http.MethodGet
//	    },
//	}))
//
// Or use the convenience function:
//
//	app.Use(hedge.Conditional(func(r *http.Request) bool {
//	    return r.Method == http.MethodGet
//	}))
//
// # How It Works
//
// The hedge middleware implements the following flow:
//
//  1. Request arrives and processing begins
//  2. Original request is sent to the backend
//  3. If no response within Delay duration, a hedge request is sent
//  4. Additional hedges are sent at Delay * hedgeNum intervals (up to MaxHedges)
//  5. The first request to complete successfully wins
//  6. Winner's response is returned, other requests are cancelled
//  7. Statistics are updated to track hedge effectiveness
//
// # Implementation Details
//
// The middleware uses several techniques to ensure correctness and performance:
//
//   - Request body buffering: The original request body is read and buffered to allow
//     multiple identical requests to be sent
//   - Response recording: Each request writes to a custom responseRecorder that captures
//     headers, status code, and body
//   - Atomic winner selection: Uses atomic operations (CompareAndSwapInt32) to ensure
//     only the first completing request is used
//   - Context propagation: Each request receives metadata through context.Context,
//     including hedge number and total hedges
//   - Graceful cancellation: When a winner is selected, remaining requests receive
//     a cancellation signal via context
//
// # Statistics
//
// The middleware tracks comprehensive statistics through the Hedger.Stats() method:
//
//	hedger := hedge.NewHedger(hedge.Options{...})
//	app.Use(hedger.Middleware())
//
//	// Later, get statistics
//	stats := hedger.Stats()
//	fmt.Printf("Total: %d, Hedged: %d, Triggered: %d\n",
//	    stats.TotalRequests,
//	    stats.HedgedRequests,
//	    stats.HedgesTriggered)
//
// Statistics include:
//   - TotalRequests: All requests processed
//   - HedgedRequests: Requests eligible for hedging
//   - HedgesTriggered: Number of hedge requests actually sent
//   - WinsByOriginal: Original request completed first
//   - WinsByHedge: Hedge request completed first
//
// # Accessing Hedge Information
//
// Within handlers, you can access hedge metadata using GetHedgeInfo:
//
//	app.Get("/api/data", func(c *mizu.Ctx) error {
//	    info := hedge.GetHedgeInfo(c)
//	    if info != nil {
//	        log.Printf("Hedge %d of %d", info.HedgeNumber, info.TotalHedges)
//	    }
//	    return c.JSON(http.StatusOK, data)
//	})
//
// Or check if the current request is a hedge:
//
//	if hedge.IsHedge(c) {
//	    // This is a hedged request (not the original)
//	}
//
// # Best Practices
//
//   - Only hedge idempotent operations: Non-idempotent operations (POST, PUT, DELETE)
//     should not be hedged as they may cause duplicate side effects
//   - Set delay based on p50 latency: The delay should be set around your median
//     response time to balance latency reduction with resource usage
//   - Monitor hedge rate: Track how often hedges are triggered to understand if
//     the delay is appropriate
//   - Ensure backend capacity: Hedging increases backend load, so ensure your
//     backend can handle the additional requests
//   - Use appropriate timeout: Set Timeout to prevent hedges from running indefinitely
//
// # When to Use
//
// Hedge middleware is most effective in these scenarios:
//   - High-latency backends with variable response times
//   - P99 latency optimization requirements
//   - Read-heavy workloads where duplicated reads are acceptable
//   - Systems where occasional backend slowness impacts user experience
//
// # When NOT to Use
//
// Avoid hedge middleware in these cases:
//   - Non-idempotent operations that cause side effects
//   - Resource-intensive operations where duplication is costly
//   - When backend cannot handle the additional load
//   - Systems with consistent, predictable response times
//
// # Configuration Options
//
// Options struct fields:
//
//   - Delay (time.Duration): Time to wait before sending first hedge request.
//     Default: 100ms
//   - MaxHedges (int): Maximum number of hedge requests to send. The total number
//     of concurrent requests will be MaxHedges + 1 (original). Default: 1
//   - Timeout (time.Duration): Overall timeout for all requests. If no request
//     completes within this time, the request fails. Default: 30s
//   - ShouldHedge (func(*http.Request) bool): Function to determine if a request
//     should be hedged. If nil, all requests are hedged. Default: nil
//   - OnHedge (func(int)): Callback invoked when a hedge is triggered. Receives
//     the hedge number. Default: nil
//   - OnComplete (func(int, time.Duration)): Callback invoked when a request completes.
//     Receives the hedge number (0 = original) and duration. Default: nil
//
// # Related Middlewares
//
//   - timeout: Sets request timeout limits
//   - retry: Retries failed requests with backoff
//   - circuitbreaker: Prevents cascading failures
package hedge
