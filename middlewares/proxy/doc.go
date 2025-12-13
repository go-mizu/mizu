// Package proxy provides reverse proxy middleware for the Mizu web framework.
//
// The proxy middleware enables forwarding HTTP requests to upstream servers with support
// for path rewriting, request/response modification, load balancing, and custom error handling.
//
// # Basic Usage
//
// Forward all requests to a single upstream server:
//
//	app := mizu.New()
//	app.Use(proxy.New("http://backend:8080"))
//
// # Path Rewriting
//
// Transform request paths before forwarding:
//
//	app.Use(proxy.WithOptions(proxy.Options{
//	    Target: mustParseURL("http://backend:8080"),
//	    Rewrite: func(path string) string {
//	        return strings.TrimPrefix(path, "/api/v1")
//	    },
//	}))
//
// # Request Modification
//
// Modify requests before forwarding to upstream:
//
//	app.Use(proxy.WithOptions(proxy.Options{
//	    Target: mustParseURL("http://backend:8080"),
//	    ModifyRequest: func(req *http.Request) {
//	        req.Header.Set("X-Source", "gateway")
//	        req.Header.Del("Authorization")
//	    },
//	}))
//
// # Response Modification
//
// Modify responses before returning to client:
//
//	app.Use(proxy.WithOptions(proxy.Options{
//	    Target: mustParseURL("http://backend:8080"),
//	    ModifyResponse: func(resp *http.Response) error {
//	        resp.Header.Set("X-Proxy", "mizu")
//	        return nil
//	    },
//	}))
//
// # Load Balancing
//
// Distribute requests across multiple upstreams using round-robin:
//
//	app.Use(proxy.Balancer([]string{
//	    "http://server1:8080",
//	    "http://server2:8080",
//	    "http://server3:8080",
//	}))
//
// # Custom Timeouts
//
// Configure request timeout for upstream servers:
//
//	app.Use(proxy.WithOptions(proxy.Options{
//	    Target:  mustParseURL("http://backend:8080"),
//	    Timeout: 60 * time.Second,
//	}))
//
// # Host Header Preservation
//
// Preserve the original Host header when forwarding:
//
//	app.Use(proxy.WithOptions(proxy.Options{
//	    Target:       mustParseURL("http://backend:8080"),
//	    PreserveHost: true,
//	}))
//
// # Error Handling
//
// Handle proxy errors with custom logic:
//
//	app.Use(proxy.WithOptions(proxy.Options{
//	    Target: mustParseURL("http://backend:8080"),
//	    ErrorHandler: func(c *mizu.Ctx, err error) error {
//	        log.Printf("proxy error: %v", err)
//	        return c.Text(http.StatusServiceUnavailable, "Service Unavailable")
//	    },
//	}))
//
// # Forwarded Headers
//
// The proxy middleware automatically adds standard forwarding headers:
//   - X-Forwarded-For: Client IP address chain
//   - X-Forwarded-Host: Original host from the request
//   - X-Forwarded-Proto: Protocol (http or https) based on TLS
//
// # Implementation Details
//
// Request Flow:
//  1. Parse and validate target URL
//  2. Apply path rewriting if configured
//  3. Create proxy request with same method, context, and body
//  4. Copy all headers from original request
//  5. Set X-Forwarded-* headers
//  6. Handle Host header based on PreserveHost option
//  7. Apply ModifyRequest callback if configured
//  8. Send request to upstream with timeout
//  9. Apply ModifyResponse callback if configured
//  10. Stream response body to client
//
// Load Balancer:
//   - Uses simple round-robin algorithm
//   - Maintains request counter for target selection
//   - Counter is not thread-safe but provides adequate distribution
//
// Performance:
//   - Response bodies are streamed using io.Copy (no buffering)
//   - Connection pooling via http.DefaultTransport
//   - Context propagation for timeout and cancellation
//   - Efficient header copying
//
// # Configuration Options
//
// Options struct fields:
//   - Target: Upstream server URL (required)
//   - Rewrite: Function to transform request path
//   - ModifyRequest: Function to modify proxy request
//   - ModifyResponse: Function to modify proxy response
//   - Transport: Custom HTTP transport (defaults to http.DefaultTransport)
//   - Timeout: Request timeout (defaults to 30s)
//   - PreserveHost: Keep original Host header (defaults to false)
//   - ErrorHandler: Custom error handler (defaults to 502 Bad Gateway)
package proxy
