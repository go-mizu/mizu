// Package fingerprint provides request fingerprinting middleware for the Mizu web framework.
//
// # Overview
//
// The fingerprint middleware generates unique identifiers (fingerprints) for incoming HTTP
// requests based on various request attributes such as headers, IP addresses, HTTP methods,
// and request paths. These fingerprints can be used for client identification, bot detection,
// analytics tracking, rate limiting, and fraud detection.
//
// # Quick Start
//
// The simplest way to use the middleware is with default settings:
//
//	import (
//	    "github.com/go-mizu/mizu"
//	    "github.com/go-mizu/mizu/middlewares/fingerprint"
//	)
//
//	app := mizu.New()
//	app.Use(fingerprint.New())
//
//	app.Get("/", func(c *mizu.Ctx) error {
//	    hash := fingerprint.Hash(c)
//	    return c.JSON(200, map[string]string{"fingerprint": hash})
//	})
//
// # Configuration Options
//
// The middleware can be configured with various options using the Options struct:
//
//   - Headers: List of HTTP headers to include in fingerprint (default: common headers)
//   - IncludeIP: Include client IP address in fingerprint (default: false)
//   - IncludeMethod: Include HTTP method in fingerprint (default: false)
//   - IncludePath: Include request path in fingerprint (default: false)
//   - Custom: Function to add custom components to fingerprint
//
// # Custom Configuration
//
// Example with custom options:
//
//	app.Use(fingerprint.WithOptions(fingerprint.Options{
//	    Headers:       []string{"User-Agent", "Accept-Language"},
//	    IncludeIP:     true,
//	    IncludeMethod: true,
//	    IncludePath:   false,
//	    Custom: func(c *mizu.Ctx) map[string]string {
//	        return map[string]string{
//	            "session_id": c.Cookie("session_id"),
//	        }
//	    },
//	}))
//
// # Convenience Functions
//
// Several convenience functions are provided for common configurations:
//
//	// Include only specific headers
//	app.Use(fingerprint.HeadersOnly("User-Agent", "Accept-Language"))
//
//	// Include IP address
//	app.Use(fingerprint.WithIP())
//
//	// Include IP, method, path, and all default headers
//	app.Use(fingerprint.Full())
//
// # Retrieving Fingerprint Information
//
// Once the middleware is active, fingerprint information can be retrieved in handlers:
//
//	app.Get("/debug", func(c *mizu.Ctx) error {
//	    info := fingerprint.Get(c)
//	    return c.JSON(200, map[string]any{
//	        "hash":       info.Hash,
//	        "components": info.Components,
//	    })
//	})
//
// The Hash() function provides quick access to just the fingerprint hash:
//
//	hash := fingerprint.Hash(c)
//
// # Default Headers
//
// When no headers are specified, the following default headers are used:
//
//   - User-Agent
//   - Accept
//   - Accept-Language
//   - Accept-Encoding
//   - Connection
//   - Sec-Ch-Ua
//   - Sec-Ch-Ua-Mobile
//   - Sec-Ch-Ua-Platform
//
// # Hash Generation
//
// Fingerprint hashes are generated using the SHA256 algorithm:
//
//  1. All components (headers, IP, etc.) are collected
//  2. Component keys are sorted alphabetically for consistency
//  3. Components are concatenated as "key:value|key:value|..."
//  4. A SHA256 hash is computed from the concatenated string
//  5. The hash is returned as a 64-character hexadecimal string
//
// # IP Address Detection
//
// When IncludeIP is enabled, the client IP is extracted with the following priority:
//
//  1. X-Forwarded-For header (first IP in comma-separated list)
//  2. X-Real-IP header
//  3. Request RemoteAddr
//
// # Use Cases
//
// Rate Limiting: Use fingerprints to track request counts per unique client:
//
//	var requestCounts = make(map[string]int)
//	app.Use(func(next mizu.Handler) mizu.Handler {
//	    return func(c *mizu.Ctx) error {
//	        hash := fingerprint.Hash(c)
//	        requestCounts[hash]++
//	        if requestCounts[hash] > 100 {
//	            return c.Text(429, "Rate limit exceeded")
//	        }
//	        return next(c)
//	    }
//	})
//
// Bot Detection: Identify known bot fingerprints:
//
//	var knownBots = map[string]bool{"abc123...": true}
//	app.Use(func(next mizu.Handler) mizu.Handler {
//	    return func(c *mizu.Ctx) error {
//	        if knownBots[fingerprint.Hash(c)] {
//	            return c.Text(403, "Bot detected")
//	        }
//	        return next(c)
//	    }
//	})
//
// Analytics: Track unique visitors and their behavior:
//
//	app.Use(func(next mizu.Handler) mizu.Handler {
//	    return func(c *mizu.Ctx) error {
//	        info := fingerprint.Get(c)
//	        analytics.TrackVisitor(info.Hash, info.Components)
//	        return next(c)
//	    }
//	})
//
// # Thread Safety
//
// The middleware is thread-safe. Each request receives its own context and Info struct,
// with no shared state between requests.
//
// # Privacy Considerations
//
// When using this middleware, consider the following best practices:
//
//   - Use minimal headers for GDPR/privacy compliance
//   - Do not rely solely on fingerprints for authentication
//   - Store fingerprints in hashed form
//   - Combine fingerprints with other signals for bot detection
//   - Use for legitimate purposes like analytics and fraud detection, not user tracking
//
// # Performance
//
// The middleware is optimized for performance:
//
//   - Minimal memory allocation through pre-sized maps
//   - Efficient string concatenation using strings.Builder
//   - Single-pass component collection
//   - Hash computed only once per request
package fingerprint
