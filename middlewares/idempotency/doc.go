// Package idempotency provides middleware for ensuring idempotent request handling
// by caching responses based on idempotency keys.
//
// # Overview
//
// The idempotency middleware prevents duplicate operations by storing and replaying
// responses for requests with the same idempotency key. This is critical for
// operations like payment processing, resource creation, and webhook handling where
// duplicate requests could cause unintended side effects.
//
// # Basic Usage
//
//	app := mizu.New()
//	app.Use(idempotency.New())
//
//	app.Post("/payments", func(c *mizu.Ctx) error {
//	    // This will only execute once per idempotency key
//	    result := processPayment()
//	    return c.JSON(200, result)
//	})
//
// # Configuration
//
// The middleware can be configured with custom options:
//
//	app.Use(idempotency.WithOptions(idempotency.Options{
//	    KeyHeader: "X-Request-Id",           // Custom header name
//	    Methods:   []string{"POST", "PUT"},  // HTTP methods to cache
//	    Lifetime:  24 * time.Hour,           // Response cache duration
//	    KeyGenerator: func(key string, c *mizu.Ctx) string {
//	        // Custom key generation logic
//	        userID := c.Request().Header.Get("X-User-ID")
//	        return userID + ":" + key
//	    },
//	}))
//
// # Store Interface
//
// The middleware uses a Store interface for response persistence. A built-in
// MemoryStore is provided for single-instance deployments:
//
//	store := idempotency.NewMemoryStore()
//	defer store.Close()
//	app.Use(idempotency.WithStore(store, idempotency.Options{}))
//
// For multi-instance deployments, implement the Store interface with a distributed
// backend like Redis:
//
//	type Store interface {
//	    Get(key string) (*Response, error)
//	    Set(key string, resp *Response) error
//	    Delete(key string) error
//	}
//
// # Response Caching
//
// When a request with an idempotency key is processed:
//
//  1. The middleware checks if a cached response exists for the key
//  2. If found, the cached response is replayed with the Idempotent-Replayed header
//  3. If not found, the handler executes normally and the response is cached
//
// Cached responses include:
//   - HTTP status code
//   - All response headers
//   - Complete response body
//   - Expiration timestamp
//
// # Cache Key Generation
//
// By default, cache keys are generated using SHA-256 hashing of:
//   - The idempotency key from the request header
//   - The HTTP method (POST, PUT, etc.)
//   - The request URL path
//
// This ensures the same idempotency key can be safely reused across different
// endpoints and methods.
//
// # Thread Safety
//
// The built-in MemoryStore uses sync.RWMutex for thread-safe concurrent access
// and runs a background cleanup goroutine to remove expired entries every 10 minutes.
//
// # Best Practices
//
//   - Generate unique keys client-side (e.g., UUIDs)
//   - Include user context in key generation for multi-tenant applications
//   - Set appropriate TTL based on your use case
//   - Use distributed store (Redis) for multi-instance deployments
//   - Always use idempotency keys for payment and financial operations
//
// # Response Headers
//
// The middleware adds the following header to replayed responses:
//
//	Idempotent-Replayed: true
//
// This allows clients to distinguish between original and replayed responses.
package idempotency
