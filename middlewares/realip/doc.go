// Package realip provides middleware for extracting the real client IP address from proxy headers.
//
// # Overview
//
// The realip middleware is essential when running behind load balancers, reverse proxies,
// or CDN services. It extracts the real client IP from standard proxy headers while
// providing security features to prevent IP spoofing.
//
// # Basic Usage
//
//	app := mizu.New()
//	app.Use(realip.New())
//
//	app.Get("/", func(c *mizu.Ctx) error {
//	    ip := realip.Get(c)
//	    return c.Text(200, "Your IP: "+ip)
//	})
//
// # Trusted Proxies
//
// For security, you should configure trusted proxies to prevent clients from
// spoofing their IP addresses:
//
//	app.Use(realip.WithTrustedProxies(
//	    "10.0.0.0/8",
//	    "192.168.0.0/16",
//	))
//
// The middleware supports both CIDR notation and single IP addresses. Single IPs
// are automatically converted to /32 (IPv4) or /128 (IPv6) CIDR blocks.
//
// # Custom Configuration
//
// You can customize which headers to check and their priority order:
//
//	app.Use(realip.WithOptions(realip.Options{
//	    TrustedProxies: []string{"10.0.0.0/8"},
//	    TrustedHeaders: []string{
//	        "X-Real-IP",
//	        "X-Forwarded-For",
//	    },
//	}))
//
// # Default Headers
//
// When no custom headers are specified, the middleware checks these headers in order:
//   - X-Forwarded-For (supports comma-separated IP lists)
//   - X-Real-IP
//   - CF-Connecting-IP (Cloudflare)
//   - True-Client-IP
//
// # IP Extraction Process
//
// The middleware follows this systematic process:
//
//  1. Extract IP from Request.RemoteAddr using net.SplitHostPort
//  2. Check if the remote IP is in the trusted proxy list
//  3. If trusted, iterate through configured headers in priority order
//  4. Parse and validate IPs from headers (handles comma-separated values)
//  5. Fall back to RemoteAddr if no valid IP found in headers
//
// # Security Considerations
//
// Without TrustedProxies configured, ALL proxy headers are trusted. This is convenient
// for development but NOT recommended for production environments, as clients can easily
// spoof their IP addresses by setting proxy headers.
//
// Always configure TrustedProxies in production to only trust headers from known
// proxy servers, load balancers, or CDN edge nodes.
//
// # Context Storage
//
// The real IP is stored in the request context using a private contextKey{} type,
// preventing key collisions with other middleware or application code. Use the
// FromContext() or Get() functions to retrieve the stored IP.
//
// # IPv6 Support
//
// The middleware fully supports IPv6 addresses in both proxy headers and CIDR
// configuration. IPv6 addresses are automatically detected and handled correctly.
package realip
