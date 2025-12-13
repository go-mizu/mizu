// Package h2c provides HTTP/2 Cleartext (h2c) middleware for the Mizu web framework.
//
// Overview
//
// The h2c middleware enables HTTP/2 connections over unencrypted TCP connections,
// allowing HTTP/2 protocol benefits without TLS. This is useful for development
// environments, internal services behind TLS-terminating proxies, and testing scenarios.
//
// The middleware supports two HTTP/2 connection methods:
//
//  1. HTTP/2 Prior Knowledge - Direct HTTP/2 connections where the client knows
//     in advance that the server supports HTTP/2 (RFC 7540, Section 3.4)
//
//  2. HTTP/2 Upgrade - HTTP/1.1 upgrade mechanism using the h2c upgrade protocol
//     (RFC 7540, Section 3.2)
//
// Security Warning
//
// H2C transmits data without encryption. Only use it in:
//   - Development and testing environments
//   - Internal services behind TLS-terminating load balancers
//   - Scenarios where network-level security is guaranteed
//
// Never expose h2c endpoints directly to the public internet.
//
// Basic Usage
//
// Create a Mizu application with h2c middleware:
//
//	app := mizu.NewRouter()
//	app.Use(h2c.New())
//
//	app.Get("/", func(c *mizu.Ctx) error {
//	    if h2c.IsHTTP2(c) {
//	        return c.Text(200, "HTTP/2 connection!")
//	    }
//	    return c.Text(200, "HTTP/1.1 connection")
//	})
//
// Advanced Usage
//
// Customize h2c behavior with options:
//
//	app.Use(h2c.WithOptions(h2c.Options{
//	    AllowUpgrade: true,
//	    AllowDirect:  true,
//	    OnUpgrade: func(r *http.Request) {
//	        log.Printf("HTTP/2 upgrade from %s", r.RemoteAddr)
//	    },
//	}))
//
// Detection Only
//
// Use Detect() to identify h2c connections without handling the upgrade:
//
//	app.Use(h2c.Detect())
//	app.Get("/", func(c *mizu.Ctx) error {
//	    info := h2c.GetInfo(c)
//	    if info.IsHTTP2 {
//	        // Handle HTTP/2 connection
//	    }
//	    return c.Text(200, "OK")
//	})
//
// Wrapping Standard Handlers
//
// Wrap standard http.Handler with h2c support:
//
//	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//	    w.Write([]byte("Hello HTTP/2!"))
//	})
//	h2cHandler := h2c.Wrap(handler)
//	http.ListenAndServe(":8080", h2cHandler)
//
// Connection Information
//
// Access HTTP/2 connection details through the Info struct:
//
//	info := h2c.GetInfo(c)
//	if info.IsHTTP2 {
//	    if info.Direct {
//	        // Direct HTTP/2 connection (prior knowledge)
//	    } else if info.Upgraded {
//	        // Upgraded from HTTP/1.1
//	    }
//	}
//
// HTTP2-Settings Header
//
// Parse HTTP/2 settings from the upgrade request:
//
//	settings, err := h2c.ParseSettings(r)
//	if err != nil {
//	    // Handle invalid settings
//	}
//	// Process settings bytes
//
// Connection Preface Detection
//
// Detect HTTP/2 connection preface in raw data:
//
//	data, _ := bufReader.Peek(24)
//	if h2c.IsHTTP2Preface(data) {
//	    // Handle HTTP/2 connection
//	}
//
// Testing
//
// Test h2c endpoints with curl:
//
//	# HTTP/2 prior knowledge
//	curl --http2-prior-knowledge http://localhost:8080/
//
//	# HTTP/1.1 upgrade to h2c
//	curl --http2 http://localhost:8080/
//
// Implementation Details
//
// The middleware operates by:
//
//  1. Inspecting incoming requests for HTTP/2 indicators
//  2. Validating h2c upgrade requests (Connection, Upgrade, HTTP2-Settings headers)
//  3. Using http.Hijacker to take control of the TCP connection for upgrades
//  4. Sending 101 Switching Protocols response
//  5. Storing connection information in request context
//
// The Info struct provides connection details accessible via GetInfo() and IsHTTP2()
// helper functions throughout the request lifecycle.
//
// Buffered connections are supported through BufferedConn, which wraps net.Conn
// with a bufio.Reader for efficient peeking and reading.
//
// Best Practices
//
//   - Always use behind a TLS-terminating reverse proxy in production
//   - Enable logging via OnUpgrade callback for monitoring
//   - Validate client behavior during development
//   - Use Detect() mode when you only need connection information
//   - Test both prior knowledge and upgrade paths
//
// Related Packages
//
//   - net/http: Standard HTTP server and client
//   - golang.org/x/net/http2: Official HTTP/2 implementation
//   - github.com/go-mizu/mizu: Mizu web framework
//
// References
//
//   - RFC 7540: Hypertext Transfer Protocol Version 2 (HTTP/2)
//   - RFC 7541: HPACK: Header Compression for HTTP/2
//
package h2c
