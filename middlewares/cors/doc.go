// Package cors provides Cross-Origin Resource Sharing (CORS) middleware for Mizu.
//
// CORS is a security feature that allows web applications running at one origin
// to access resources from a different origin. This middleware implements the
// W3C CORS specification by handling preflight requests and setting appropriate
// response headers.
//
// # Quick Start
//
// For development, use AllowAll() to permit all origins:
//
//	app := mizu.New()
//	app.Use(cors.AllowAll())
//
// For production, use specific origins:
//
//	app.Use(cors.WithOrigins("https://example.com", "https://app.example.com"))
//
// # Configuration
//
// The middleware supports extensive configuration through the Options struct:
//
//   - AllowOrigins: List of permitted origins (default: ["*"])
//   - AllowMethods: Permitted HTTP methods (default: GET, POST, HEAD)
//   - AllowHeaders: Permitted request headers (default: Origin, Content-Type, Accept)
//   - ExposeHeaders: Headers exposed to the browser (default: none)
//   - AllowCredentials: Enable cookie/credential support (default: false)
//   - MaxAge: Preflight cache duration (default: 0)
//   - AllowOriginFunc: Custom origin validation function (default: nil)
//   - AllowPrivateNetwork: Enable Private Network Access (default: false)
//
// # Examples
//
// Custom configuration with credentials:
//
//	app.Use(cors.New(cors.Options{
//		AllowOrigins:     []string{"https://app.example.com"},
//		AllowCredentials: true,
//		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
//		AllowHeaders:     []string{"Authorization", "Content-Type"},
//		MaxAge:           12 * time.Hour,
//	}))
//
// Dynamic origin validation:
//
//	app.Use(cors.New(cors.Options{
//		AllowOriginFunc: func(origin string) bool {
//			// Allow all subdomains of example.com
//			return strings.HasSuffix(origin, ".example.com")
//		},
//	}))
//
// # Implementation Details
//
// The middleware processes requests in the following order:
//
//  1. Extracts the Origin header from the incoming request
//  2. If no Origin header is present, skips CORS processing
//  3. Validates the origin against allowed origins or validation function
//  4. If origin is not allowed, continues without setting CORS headers
//  5. Sets Access-Control-Allow-Origin header (wildcard or specific)
//  6. Adds Vary: Origin header when using specific origins
//  7. Sets Access-Control-Allow-Credentials if enabled
//  8. Sets Access-Control-Expose-Headers if configured
//  9. For OPTIONS requests (preflight):
//     - Sets Access-Control-Allow-Methods
//     - Sets Access-Control-Allow-Headers
//     - Sets Access-Control-Max-Age if configured
//     - Handles Private Network Access if enabled
//     - Returns 204 No Content
//  10. For regular requests, continues to the next handler
//
// # Security Considerations
//
// When using CORS middleware, keep these security practices in mind:
//
//   - Never use AllowAll() in production - it exposes your API to any origin
//   - When AllowCredentials is true, you must specify exact origins (no wildcards)
//   - Use AllowOriginFunc for complex origin validation logic
//   - Only expose headers that clients actually need
//   - Set appropriate MaxAge to reduce preflight request overhead
//
// # Private Network Access
//
// The middleware supports the Private Network Access specification, which allows
// public websites to request resources from private networks (e.g., localhost,
// local network devices). Enable this only when necessary:
//
//	app.Use(cors.New(cors.Options{
//		AllowOrigins:        []string{"https://example.com"},
//		AllowPrivateNetwork: true,
//	}))
//
// When enabled, the middleware responds to preflight requests containing the
// Access-Control-Request-Private-Network header by setting
// Access-Control-Allow-Private-Network to true.
//
// # Best Practices
//
//   - Use specific origin lists in production environments
//   - Cache preflight requests with MaxAge to reduce OPTIONS overhead
//   - Combine with authentication middleware for protected endpoints
//   - Monitor and log CORS-related errors during development
//   - Test CORS configuration with actual client applications
//
// For more information, see: https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS
package cors
