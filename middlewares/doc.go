// Package middlewares provides a comprehensive collection of HTTP middlewares
// for the Mizu web framework.
//
// All middlewares in this package follow Go standard library conventions:
//   - Use only standard library dependencies
//   - Provide sensible defaults
//   - Support the Options pattern for configuration
//   - Are safe for concurrent use
//
// # Organization
//
// Middlewares are organized into sub-packages by functionality:
//
//	basicauth    - HTTP Basic Authentication
//	bearerauth   - Bearer token authentication
//	bodylimit    - Request body size limiting
//	cache        - Cache-Control headers
//	compress     - Response compression (gzip, deflate)
//	cors         - Cross-Origin Resource Sharing
//	csrf         - CSRF protection
//	etag         - ETag generation
//	helmet       - Security headers
//	ratelimit    - Rate limiting
//	recover      - Panic recovery
//	requestid    - Request ID generation
//	timeout      - Request timeout
//	...and more
//
// # Usage
//
// Import the specific middleware package you need:
//
//	import "github.com/go-mizu/mizu/middlewares/cors"
//
//	app := mizu.New()
//	app.Use(cors.AllowAll())
//
// # Design Principles
//
//   - Zero external dependencies
//   - Composable with other middlewares
//   - Configurable via Options pattern
//   - Safe defaults for security
//   - Comprehensive test coverage
package middlewares
