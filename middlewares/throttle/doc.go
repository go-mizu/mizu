// Package throttle provides request throttling middleware for Mizu.
//
// The throttle middleware limits the number of concurrent requests being processed,
// preventing resource exhaustion and controlling server load. It uses a semaphore-based
// approach with a configurable backlog queue for waiting requests.
//
// # Basic Usage
//
// Limit concurrent requests to 100:
//
//	app := mizu.New()
//	app.Use(throttle.New(100))
//
// # Configuration Options
//
// The middleware supports several configuration options:
//
//   - Limit: Maximum number of concurrent requests (default: 100)
//   - Backlog: Maximum number of requests waiting in queue (default: 1000)
//   - Timeout: Maximum time a request will wait for a slot (default: 30s)
//   - OnThrottle: Callback function invoked when a request is throttled
//
// # Examples
//
// Configure with backlog queue:
//
//	app.Use(throttle.WithOptions(throttle.Options{
//	    Limit:   50,
//	    Backlog: 200,
//	}))
//
// Configure with timeout:
//
//	app.Use(throttle.WithOptions(throttle.Options{
//	    Limit:   50,
//	    Timeout: 10 * time.Second,
//	}))
//
// Disable backlog (immediate rejection when slots full):
//
//	app.Use(throttle.WithOptions(throttle.Options{
//	    Limit:      100,
//	    Backlog:    0,
//	    BacklogSet: true,
//	}))
//
// Use throttle callback for monitoring:
//
//	app.Use(throttle.WithOptions(throttle.Options{
//	    Limit: 100,
//	    OnThrottle: func(c *mizu.Ctx) {
//	        log.Printf("Request throttled: %s", c.Request().URL.Path)
//	    },
//	}))
//
// # Implementation Details
//
// The middleware uses a buffered channel as a semaphore to control concurrency.
// Each slot in the semaphore represents permission to process one request.
//
// Request flow:
//  1. Request arrives and attempts to acquire a slot (non-blocking)
//  2. If slot available, request proceeds immediately
//  3. If no slot available, check backlog capacity
//  4. If backlog full, reject with 503 Service Unavailable
//  5. Otherwise, wait with timeout for slot to become available
//  6. On timeout or context cancellation, reject request
//
// The backlog counter is protected by a mutex for thread safety.
//
// # Error Responses
//
// The middleware returns 503 Service Unavailable in two scenarios:
//   - "service busy": Backlog capacity exceeded
//   - "request timeout": Timeout while waiting for a slot
//
// # Thread Safety
//
// The middleware is safe for concurrent use. The semaphore channel provides
// lock-free concurrency control, while a mutex protects the backlog counter.
//
// # Performance Considerations
//
// The semaphore-based approach has minimal overhead for requests that acquire
// slots immediately. Waiting requests use Go's timer functionality which is
// efficient even with many concurrent timers.
//
// Default values are chosen to balance protection and usability:
//   - Limit: 100 (suitable for most applications)
//   - Backlog: 1000 (allows for traffic bursts)
//   - Timeout: 30s (prevents indefinite waiting)
//
// Adjust these values based on your application's characteristics and
// resource constraints.
package throttle
