// Package forwarded provides middleware for parsing and handling X-Forwarded-* headers
// in proxy and load balancer environments.
//
// # Overview
//
// The forwarded middleware extracts client information from proxy headers, enabling
// applications running behind reverse proxies to access the original client's IP address,
// protocol, host, and other forwarded metadata. It supports both the legacy X-Forwarded-*
// headers and the RFC 7239 Forwarded header standard.
//
// # Basic Usage
//
//	app := mizu.New()
//	app.Use(forwarded.New())
//
//	app.Get("/", func(c *mizu.Ctx) error {
//	    info := forwarded.Get(c)
//	    return c.JSON(200, map[string]any{
//	        "clientIP": info.ClientIP,
//	        "proto":    info.Proto,
//	        "host":     info.Host,
//	    })
//	})
//
// # Trusted Proxies
//
// For security, you should explicitly specify which proxy IPs to trust:
//
//	app.Use(forwarded.WithOptions(forwarded.Options{
//	    TrustProxy: true,
//	    TrustedProxies: []string{
//	        "10.0.0.0/8",
//	        "192.168.1.0/24",
//	    },
//	}))
//
// # Headers Supported
//
// The middleware parses the following headers when proxy trust is enabled:
//
//   - X-Forwarded-For: Client IP chain (uses first IP as original client)
//   - X-Forwarded-Host: Original host requested by client
//   - X-Forwarded-Proto: Original protocol (http/https)
//   - X-Forwarded-Port: Original port number
//   - X-Forwarded-Prefix: Path prefix added by reverse proxy
//   - Forwarded: RFC 7239 standardized forwarded header
//
// # Helper Functions
//
// The package provides convenient helper functions for common use cases:
//
//	// Get full forwarded information
//	info := forwarded.Get(c)
//
//	// Get only client IP
//	ip := forwarded.ClientIP(c)
//
//	// Get protocol (defaults to "http" if not set)
//	proto := forwarded.Proto(c)
//
//	// Get host
//	host := forwarded.Host(c)
//
// # Security Considerations
//
// IMPORTANT: Only enable TrustProxy when your application is actually behind a trusted
// reverse proxy or load balancer. Untrusted clients can easily spoof X-Forwarded-* headers,
// potentially bypassing IP-based security controls.
//
// When TrustProxy is enabled without TrustedProxies configuration, ALL proxy headers
// are trusted. This is acceptable for development but should be avoided in production.
//
// Always validate and sanitize client IPs if using them for security decisions such as:
//   - Rate limiting
//   - Access control
//   - Geo-blocking
//   - Logging for security purposes
//
// # RFC 7239 Support
//
// The middleware fully supports RFC 7239 Forwarded headers, which provide a standardized
// format for forwarding information. Example:
//
//	Forwarded: for=192.0.2.60;proto=https;host=example.org
//
// IPv6 addresses in brackets are properly handled:
//
//	Forwarded: for="[2001:db8::1]";proto=https
//
// # Implementation Details
//
// The middleware stores forwarded information in the request context using a private
// context key, ensuring it's accessible throughout the request lifecycle without
// polluting the global namespace.
//
// IP parsing handles various formats including:
//   - IPv4 addresses: 192.168.1.1
//   - IPv6 addresses: 2001:db8::1
//   - Addresses with ports: 192.168.1.1:8080 (port is stripped)
//   - Bracketed IPv6: [2001:db8::1] (brackets removed)
//
// When multiple IPs appear in X-Forwarded-For (proxy chain), the first IP is always
// used as it represents the original client. Subsequent IPs represent intermediate proxies.
package forwarded
