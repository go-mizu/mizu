// Package keepalive provides HTTP keep-alive connection management middleware for Mizu.
//
// The keepalive middleware controls HTTP connection persistence by managing the
// Connection and Keep-Alive headers. It enables efficient connection reuse for
// improved performance in high-traffic scenarios.
//
// # Basic Usage
//
// Enable keep-alive with default settings (60s timeout, 100 max requests):
//
//	app := mizu.New()
//	app.Use(keepalive.New())
//
// # Custom Configuration
//
// Configure custom timeout and maximum requests:
//
//	app.Use(keepalive.WithOptions(keepalive.Options{
//	    Timeout:     120 * time.Second,
//	    MaxRequests: 500,
//	}))
//
// # Disabling Keep-Alive
//
// Disable keep-alive for specific routes or globally:
//
//	app.Use(keepalive.Disable())
//
// Or use the DisableKeepAlive option:
//
//	app.Use(keepalive.WithOptions(keepalive.Options{
//	    DisableKeepAlive: true,
//	}))
//
// # Helper Functions
//
// Use convenience functions for common configurations:
//
//	// Set custom timeout only
//	app.Use(keepalive.WithTimeout(30 * time.Second))
//
//	// Set custom max requests only
//	app.Use(keepalive.WithMax(200))
//
// # How It Works
//
// The middleware operates by:
//
//  1. Checking if keep-alive is disabled via configuration
//  2. Inspecting the client's Connection header to detect close requests
//  3. Setting appropriate response headers:
//     - Connection: keep-alive (or close if disabled/requested)
//     - Keep-Alive: timeout={seconds}, max={requests}
//
// # Client Negotiation
//
// The middleware respects client preferences. If a client sends Connection: close,
// the middleware will honor this request and set Connection: close in the response,
// even if keep-alive is enabled in the configuration.
//
// # Use Cases
//
//   - High-traffic APIs requiring efficient connection reuse
//   - Services with frequent requests from the same clients
//   - Performance optimization for connection overhead reduction
//   - Controlled connection persistence management
//
// # Best Practices
//
//   - Use default settings for most applications
//   - Set reasonable timeout values based on your traffic patterns
//   - Monitor connection counts to detect potential resource issues
//   - Consider disabling for long-polling or server-sent events endpoints
//   - Balance timeout and max requests to optimize for your use case
//
// # HTTP Headers
//
// The middleware sets the following headers:
//
//	Connection: keep-alive
//	Keep-Alive: timeout=60, max=100
//
// When keep-alive is disabled:
//
//	Connection: close
//
// # Performance Considerations
//
// Keep-alive connections reduce latency and overhead by reusing TCP connections.
// However, long-lived connections consume server resources. The timeout and
// max requests settings help balance performance and resource utilization.
//
// Default settings (60s timeout, 100 max requests) work well for most applications,
// but should be tuned based on your specific traffic patterns and infrastructure.
package keepalive
