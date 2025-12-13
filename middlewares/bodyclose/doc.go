// Package bodyclose provides middleware to ensure HTTP request bodies are properly closed
// after processing, preventing resource leaks and enabling connection reuse.
//
// # Overview
//
// The bodyclose middleware automatically closes request bodies using Go's defer mechanism,
// ensuring cleanup happens regardless of whether the handler succeeds, fails, or panics.
// It optionally drains the body before closing to enable HTTP keep-alive connection reuse.
//
// # Basic Usage
//
// Use New() to create middleware with default settings:
//
//	app := mizu.New()
//	app.Use(bodyclose.New())
//
// # Draining Behavior
//
// By default, the middleware does not drain bodies. Use Drain() to enable draining:
//
//	app.Use(bodyclose.Drain())
//
// Or disable draining explicitly:
//
//	app.Use(bodyclose.NoDrain())
//
// # Custom Configuration
//
// Use WithOptions for fine-grained control:
//
//	app.Use(bodyclose.WithOptions(bodyclose.Options{
//		DrainBody: true,
//		MaxDrain:  16 * 1024, // 16KB
//	}))
//
// # Why It Matters
//
// Without proper body closing:
//   - HTTP connections may not be reused, degrading performance
//   - Memory leaks can occur from unclosed readers
//   - Connection pool exhaustion under high load
//
// # Connection Reuse
//
// When DrainBody is enabled, the middleware drains up to MaxDrain bytes before closing.
// This signals to the HTTP client that the connection can be safely reused, improving
// performance in scenarios with connection pooling and HTTP keep-alive.
//
// # Performance
//
// The middleware has minimal overhead:
//   - Adds only a single defer statement per request
//   - Draining is capped by MaxDrain (default: 8KB) to prevent memory exhaustion
//   - Nil body check avoids unnecessary processing for requests without bodies
//
// # Best Practices
//
//   - Place early in the middleware chain to ensure cleanup even if later middleware fails
//   - Enable draining (DrainBody: true) when using HTTP connection pooling
//   - Adjust MaxDrain based on typical request body sizes and memory constraints
//   - Use with bodylimit middleware to prevent unbounded body sizes
package bodyclose
