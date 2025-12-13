// Package ipfilter provides IP whitelist and blacklist middleware for Mizu.
//
// The ipfilter middleware filters HTTP requests based on client IP addresses,
// supporting both single IP addresses and CIDR notation for IP ranges. It can
// operate in whitelist mode (allow only specified IPs) or blacklist mode (deny
// only specified IPs).
//
// # Basic Usage
//
// Whitelist mode - allow only specific IPs:
//
//	app := mizu.New()
//	app.Use(ipfilter.Allow("192.168.1.0/24", "10.0.0.1"))
//
// Blacklist mode - deny specific IPs:
//
//	app.Use(ipfilter.Deny("192.168.1.100", "10.0.0.0/8"))
//
// # Configuration Options
//
// The middleware supports several configuration options through the Options struct:
//
//   - AllowList: List of IP addresses or CIDR ranges to allow
//   - DenyList: List of IP addresses or CIDR ranges to deny
//   - DenyByDefault: When true, denies all IPs except those in AllowList
//   - TrustProxy: When true, uses X-Forwarded-For header for IP extraction
//   - ErrorHandler: Custom error handler for denied requests
//
// # Advanced Usage
//
// Combined allow and deny lists:
//
//	app.Use(ipfilter.New(ipfilter.Options{
//	    AllowList: []string{"192.168.1.0/24"},
//	    DenyList:  []string{"192.168.1.100"},
//	}))
//
// Behind a proxy with custom error handler:
//
//	app.Use(ipfilter.New(ipfilter.Options{
//	    AllowList:     []string{"203.0.113.0/24"},
//	    DenyByDefault: true,
//	    TrustProxy:    true,
//	    ErrorHandler: func(c *mizu.Ctx) error {
//	        return c.JSON(403, map[string]string{
//	            "error": "Access denied",
//	            "ip":    c.ClientIP(),
//	        })
//	    },
//	}))
//
// # Helper Functions
//
// The package provides convenience functions for common scenarios:
//
// Localhost only:
//
//	app.Use(ipfilter.Localhost())
//
// Private networks only:
//
//	app.Use(ipfilter.Private())
//
// # IP Address Formats
//
// The middleware supports multiple IP address formats:
//
//   - Single IPv4: "192.168.1.100"
//   - IPv4 CIDR: "192.168.1.0/24"
//   - Single IPv6: "::1"
//   - IPv6 CIDR: "fc00::/7"
//
// Single IP addresses are automatically converted to CIDR notation internally
// (IPv4 to /32, IPv6 to /128) for consistent matching logic.
//
// # Filtering Logic
//
// The middleware applies filtering in the following order:
//
//  1. Extract client IP (from RemoteAddr or X-Forwarded-For if TrustProxy is enabled)
//  2. Check if IP is in DenyList - if yes, deny immediately
//  3. If DenyByDefault is true, check if IP is in AllowList - if no, deny
//  4. Allow the request to proceed
//
// This means the DenyList always takes precedence over the AllowList.
//
// # Security Considerations
//
// When using this middleware, consider the following security aspects:
//
//   - Only enable TrustProxy when behind a trusted reverse proxy or load balancer
//   - X-Forwarded-For headers can be spoofed if not properly validated by a trusted proxy
//   - Consider both IPv4 and IPv6 addresses when configuring IP filters
//   - IP-based filtering can be bypassed using VPNs or proxies
//   - Combine IP filtering with other authentication methods for sensitive resources
//
// # Performance
//
// The middleware is designed for performance:
//
//   - IP parsing and CIDR range calculation happen once at initialization
//   - IP matching uses Go's efficient net.IPNet.Contains() method
//   - DenyList is checked before AllowList for fast rejection of blocked IPs
//   - No regular expressions or string manipulation on the request hot path
//
// # Examples
//
// Restrict admin routes to localhost:
//
//	admin := app.Group("/admin")
//	admin.Use(ipfilter.Localhost())
//	admin.Get("/dashboard", dashboardHandler)
//
// Allow office network but block specific problematic IP:
//
//	app.Use(ipfilter.New(ipfilter.Options{
//	    AllowList: []string{"192.168.0.0/16"},
//	    DenyList:  []string{"192.168.1.100"},
//	}))
//
// For more examples and detailed documentation, visit:
// https://go-mizu.dev/middlewares/ipfilter
package ipfilter
