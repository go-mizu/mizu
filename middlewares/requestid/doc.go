// Package requestid provides middleware for request ID generation and propagation.
//
// The requestid middleware generates or propagates unique request IDs for distributed
// tracing and debugging. Each request gets a unique identifier that can be logged and
// passed to downstream services for end-to-end request tracking.
//
// # Basic Usage
//
// Use the middleware with default settings to generate UUID v4-style request IDs:
//
//	app := mizu.New()
//	app.Use(requestid.New())
//
//	app.Get("/", func(c *mizu.Ctx) error {
//	    id := requestid.Get(c)
//	    c.Logger().Info("processing request", "request_id", id)
//	    return c.JSON(200, map[string]string{"id": id})
//	})
//
// # Custom Configuration
//
// Configure custom header names and ID generators:
//
//	app.Use(requestid.WithOptions(requestid.Options{
//	    Header: "X-Trace-ID",
//	    Generator: func() string {
//	        return fmt.Sprintf("req-%d", time.Now().UnixNano())
//	    },
//	}))
//
// # ID Propagation
//
// If an incoming request contains a request ID header, it is preserved and propagated:
//
//	Client sends: X-Request-ID: abc123
//	Server responds: X-Request-ID: abc123
//
// If no request ID is present, a new one is generated:
//
//	Client sends: (no header)
//	Server generates: X-Request-ID: 550e8400-e29b-41d4-a716-446655440000
//
// # Retrieving Request IDs
//
// Extract the request ID from the context using Get() or FromContext():
//
//	func handler(c *mizu.Ctx) error {
//	    id := requestid.Get(c)
//	    // Use ID for logging, tracing, etc.
//	}
//
// # Implementation Details
//
// The middleware uses a private struct type as the context key to prevent collisions
// with other middleware or application code. The default generator creates UUID v4-style
// identifiers using crypto/rand for cryptographic randomness.
//
// UUID v4 Format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
//   - Version bits (byte 6): 0x40 (version 4)
//   - Variant bits (byte 8): 0x80 (RFC 4122 variant 2)
//   - 32 hexadecimal characters in 5 groups (8-4-4-4-12)
//
// # Best Practices
//
//   - Add the middleware early in the middleware chain
//   - Include request IDs in all application logs
//   - Propagate request IDs to downstream services
//   - Use request IDs for correlating errors across services
//
// # Example: Logging with Request ID
//
//	app.Use(requestid.New())
//
//	app.Get("/users/:id", func(c *mizu.Ctx) error {
//	    logger := c.Logger().With("request_id", requestid.Get(c))
//	    logger.Info("fetching user")
//
//	    user, err := fetchUser(c.Param("id"))
//	    if err != nil {
//	        logger.Error("failed to fetch user", "error", err)
//	        return err
//	    }
//
//	    return c.JSON(200, user)
//	})
//
// # Example: Passing to Downstream Services
//
//	func callDownstreamService(c *mizu.Ctx) error {
//	    req, _ := http.NewRequest("GET", "http://api.example.com/data", nil)
//	    req.Header.Set("X-Request-ID", requestid.Get(c))
//
//	    client := &http.Client{}
//	    resp, err := client.Do(req)
//	    // Handle response...
//	}
package requestid
