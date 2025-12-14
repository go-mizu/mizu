/*
Package cors provides Cross-Origin Resource Sharing (CORS) middleware for Mizu.

CORS is a browser security feature that controls whether a page from one origin
may access resources from another origin. This middleware implements common CORS
behavior by validating the Origin header, handling preflight requests, and
setting the appropriate response headers.

This middleware only runs when the request includes an Origin header. If Origin
is missing, the request is treated as same-origin or non-browser and passes
through unchanged.

# Quick Start

Development, allow any origin:

	app := mizu.New()
	app.Use(cors.AllowAll())

Production, allow specific origins:

	app.Use(cors.WithOrigins("https://example.com", "https://app.example.com"))

# Configuration

Options fields:

  - AllowOrigins: Allowed origins (default: ["*"] when AllowOriginFunc is nil).
    Supports wildcard patterns like "https://*.example.com".
  - AllowMethods: Allowed methods for preflight (default: GET, POST, HEAD)
  - AllowHeaders: Allowed request headers for preflight (default: Origin, Content-Type, Accept).
    If AllowHeaders contains "*", the middleware reflects Access-Control-Request-Headers during preflight.
  - ExposeHeaders: Response headers visible to the browser (default: none)
  - AllowCredentials: Allow cookies and credentials (default: false)
  - MaxAge: Preflight cache duration (default: 0)
  - AllowOriginFunc: Custom origin validator (default: nil)
  - AllowPrivateNetwork: Private Network Access support (default: false)

Notes:

  - If AllowCredentials is true, Access-Control-Allow-Origin will not be "*".
    The middleware echoes the request Origin when it is allowed.
  - If AllowOrigins is ["*"] and AllowCredentials is true, the middleware still
    echoes the request Origin (it never returns "*" with credentials enabled).

# Examples

Custom configuration with credentials:

	app.Use(cors.New(cors.Options{
		AllowOrigins:     []string{"https://app.example.com"},
		AllowCredentials: true,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		MaxAge:           12 * time.Hour,
	}))

Dynamic origin validation:

	app.Use(cors.New(cors.Options{
		AllowOriginFunc: func(origin string) bool {
			return origin == "https://example.com" || strings.HasSuffix(origin, ".example.com")
		},
	}))

Wildcard origin patterns:

	app.Use(cors.New(cors.Options{
		AllowOrigins: []string{"https://*.example.com"},
	}))

# Behavior

For requests with an Origin header:

 1. Validate the origin (AllowOriginFunc or AllowOrigins)
 2. If allowed, set Access-Control-Allow-Origin
    - "*" when configured and credentials are disabled
    - Otherwise echo the request Origin
 3. Add Vary: Origin when echoing a specific origin
 4. If enabled, set Access-Control-Allow-Credentials: true
 5. If configured, set Access-Control-Expose-Headers

Preflight requests are detected as:

  - Method is OPTIONS
  - Header Access-Control-Request-Method is present

For preflight requests, the middleware also sets:

  - Access-Control-Allow-Methods
  - Access-Control-Allow-Headers (explicit list or reflected)
  - Access-Control-Max-Age (when MaxAge > 0)

It also adds Vary for preflight caching correctness:

  - Vary: Access-Control-Request-Method
  - Vary: Access-Control-Request-Headers
  - Vary: Access-Control-Request-Private-Network (when used)

If the request is OPTIONS without Access-Control-Request-Method, the middleware
does not treat it as a preflight request and allows the route handler to run.

# Security Notes

  - Do not use AllowAll in production unless you truly want public cross-origin access
  - When AllowCredentials is enabled, keep the origin list tight
  - Only allow headers and expose headers that clients actually need
  - Prefer authentication and authorization as the primary security controls

# Private Network Access

When AllowPrivateNetwork is enabled, the middleware responds to preflight
requests that include:

	Access-Control-Request-Private-Network: true

by setting:

	Access-Control-Allow-Private-Network: true

Enable this only when you need browser clients on public origins to talk to
private network targets.

For more information, see:
https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS
*/
package cors
