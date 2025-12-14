/*
Package cors2 provides a simplified CORS (Cross-Origin Resource Sharing) middleware for Mizu.

This middleware handles CORS headers and preflight requests with support for:
  - Wildcard and exact origin matching
  - Configurable allowed methods and headers
  - Credentials support with correct origin echoing
  - Preflight response caching
  - Custom exposed headers

# Basic Usage

Use the default configuration to allow all origins:

	app := mizu.New()
	app.Use(cors2.New())

# Custom Configuration

Configure specific origins and options:

	app.Use(cors2.WithOptions(cors2.Options{
		Origin:      "https://example.com",
		Methods:     "GET, POST, PUT, DELETE",
		Headers:     "Content-Type, Authorization",
		Credentials: true,
		MaxAge:      12 * time.Hour,
	})

# Convenience Functions

The package provides helper functions for common scenarios:

	// Allow a specific origin
	app.Use(cors2.AllowOrigin("https://example.com"))

	// Allow all origins with extended configuration
	app.Use(cors2.AllowAll())

	// Allow credentials from a specific origin
	app.Use(cors2.AllowCredentials("https://trusted.com"))

# Options

The Options struct configures the CORS middleware:

  - Origin: The allowed origin (default: "*")
  - Methods: Comma-separated list of allowed HTTP methods
    (default: "GET, POST, PUT, DELETE, OPTIONS")
  - Headers: Comma-separated list of allowed headers
    (default: "Content-Type, Authorization")
  - ExposeHeaders: Comma-separated list of headers exposed to the browser
  - Credentials: Whether to allow credentials (default: false)
  - MaxAge: Duration to cache preflight responses (default: 0, no caching)

# Preflight Requests

The middleware automatically handles CORS preflight OPTIONS requests when
the Access-Control-Request-Method header is present:

  - Returns HTTP 204 No Content
  - Sets Access-Control-Allow-Methods
  - Sets Access-Control-Allow-Headers
  - Sets Access-Control-Max-Age when configured
  - Adds proper Vary headers for cache correctness

# Origin Handling

Origin matching supports:
  - Wildcard "*" to allow all origins
  - Exact origin matching (case-insensitive)

When Credentials is false:
  - Origin "*" results in Access-Control-Allow-Origin: "*"

When Credentials is true:
  - The request Origin is echoed back
  - Vary: Origin is added automatically
  - This applies even when Origin is configured as "*"

# Security Considerations

When allowing credentials (cookies, authorization headers, or TLS client
certificates):

  - Browsers do not allow Access-Control-Allow-Origin: "*"
  - The middleware automatically echoes the request origin instead
  - Vary: Origin is added to prevent cache poisoning
  - Prefer explicit origins in production environments

# Examples

Allow a specific origin with credentials:

	app.Use(cors2.WithOptions(cors2.Options{
		Origin:      "https://app.example.com",
		Credentials: true,
	}))

Configure for a development environment:

	app.Use(cors2.AllowAll())

Configure for production with stricter requirements:

	app.Use(cors2.WithOptions(cors2.Options{
		Origin:        "https://app.example.com",
		Methods:       "GET, POST, PUT, DELETE",
		Headers:       "Authorization, Content-Type, X-Request-ID",
		ExposeHeaders: "X-Total-Count, X-Page-Count",
		Credentials:   true,
		MaxAge:        12 * time.Hour,
	}))
*/
package cors2
