// Package bulkhead provides the bulkhead pattern middleware for Mizu.
//
// The bulkhead middleware implements the bulkhead isolation pattern, which limits
// concurrent requests to prevent cascade failures and ensure fair resource allocation.
// It acts as a protective barrier that isolates failures and prevents resource exhaustion.
//
// # Overview
//
// The bulkhead pattern is named after the partitioned sections of a ship's hull. Just as
// bulkheads prevent a breach in one section from sinking the entire ship, this middleware
// prevents failures in one part of your application from affecting others by limiting
// concurrent access to resources.
//
// # Key Features
//
//   - Concurrent request limiting with semaphore-based slot management
//   - Configurable waiting queue for requests when bulkhead is full
//   - Support for multiple named bulkheads for isolating different services
//   - Real-time statistics for monitoring active and waiting requests
//   - Custom error handlers for rejected requests
//   - Path-based bulkhead creation for automatic isolation by route
//
// # Basic Usage
//
// Simple bulkhead with a maximum of 10 concurrent requests:
//
//	app := mizu.New()
//	app.Use(bulkhead.New(bulkhead.Options{
//	    MaxConcurrent: 10,
//	    MaxWait:       5,
//	}))
//
// # Configuration Options
//
// The Options struct provides the following configuration:
//
//   - Name: Optional bulkhead name for identification
//   - MaxConcurrent: Maximum number of concurrent requests (default: 10)
//   - MaxWait: Maximum number of requests that can wait for a slot (default: 10)
//   - ErrorHandler: Custom error handler for rejected requests
//
// # Advanced Usage
//
// Using a bulkhead manager for multiple isolated compartments:
//
//	manager := bulkhead.NewManager()
//	apiSvc := manager.Get("api", 100, 50)
//	dbSvc := manager.Get("database", 20, 10)
//
//	app.Use("/api", apiSvc.Middleware())
//	app.Use("/db", dbSvc.Middleware())
//
// Path-based automatic bulkhead creation:
//
//	manager := bulkhead.NewManager()
//	app.Use(bulkhead.ForPath(manager, 10, 5))
//	// Automatically creates separate bulkheads for each unique path
//
// Custom error handling:
//
//	app.Use(bulkhead.New(bulkhead.Options{
//	    MaxConcurrent: 10,
//	    MaxWait:       5,
//	    ErrorHandler: func(c *mizu.Ctx) error {
//	        return c.JSON(503, map[string]string{
//	            "error": "Service temporarily unavailable",
//	            "retry_after": "5s",
//	        })
//	    },
//	}))
//
// # Implementation Details
//
// The bulkhead uses a buffered channel as a semaphore to control concurrent access:
//
//  1. When a request arrives, it attempts a non-blocking acquire of a slot
//  2. If a slot is available, the request proceeds immediately
//  3. If no slot is available, it checks the waiting queue capacity
//  4. If the queue is not full, it waits for a slot to become available
//  5. If the queue is full, the request is rejected with 503 Service Unavailable
//  6. Slots are released via defer to ensure cleanup even if handlers panic
//
// Thread safety is ensured using sync.Mutex for protecting the waiting counter
// and sync.RWMutex for the manager's bulkhead map.
//
// # Statistics
//
// Each bulkhead tracks statistics that can be retrieved for monitoring:
//
//	stats := bulkhead.Stats()
//	fmt.Printf("Active: %d/%d, Waiting: %d/%d, Available: %d\n",
//	    stats.Active, stats.MaxActive,
//	    stats.Waiting, stats.MaxWaiting,
//	    stats.Available)
//
// For managers, you can get aggregated statistics for all bulkheads:
//
//	allStats := manager.Stats()
//	for name, stats := range allStats {
//	    fmt.Printf("Bulkhead %s: Active=%d, Available=%d\n",
//	        name, stats.Active, stats.Available)
//	}
//
// # HTTP Status Codes
//
// The middleware returns the following HTTP status codes:
//
//   - 503 Service Unavailable: Returned when the bulkhead is full (both active slots
//     and waiting queue are at capacity) unless a custom ErrorHandler is provided
//
// # Best Practices
//
//   - Set MaxConcurrent based on your application's resource capacity
//   - Use different bulkheads for different services to prevent cross-contamination
//   - Monitor rejection rates to identify capacity issues
//   - Combine with circuit breaker middleware for comprehensive resilience
//   - Set MaxWait to a reasonable value to prevent excessive memory usage
//   - Use the Manager for organizing multiple bulkheads in complex applications
//
// # Related Middlewares
//
// The bulkhead middleware works well in combination with:
//
//   - circuitbreaker: Prevents repeated calls to failing services
//   - ratelimit: Controls the rate of incoming requests over time
//   - timeout: Limits the duration of request processing
//
// # Performance Considerations
//
// The bulkhead middleware has minimal performance overhead:
//
//   - Slot acquisition/release: O(1) channel operations
//   - Statistics retrieval: O(1) with mutex lock
//   - Manager lookup: O(1) with read lock for existing bulkheads
//
// The use of buffered channels for semaphores provides excellent performance
// even under high concurrency scenarios.
package bulkhead
