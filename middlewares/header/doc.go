// Package header provides request/response header manipulation middleware for Mizu.
//
// The header middleware offers flexible header manipulation capabilities for both
// request and response headers. It includes convenience functions for common security
// headers, content-type headers, and custom header operations.
//
// # Basic Usage
//
// Set a single response header:
//
//	app.Use(header.Set("X-Custom-Header", "value"))
//
// Set multiple response headers:
//
//	app.Use(header.New(map[string]string{
//	    "X-App-Name":    "MyApp",
//	    "X-App-Version": "1.0.0",
//	}))
//
// Remove response headers:
//
//	app.Use(header.Remove("Server", "X-Powered-By"))
//
// # Request Headers
//
// The middleware can also manipulate request headers before they reach the handler:
//
//	app.Use(header.SetRequest("X-Request-Source", "web"))
//	app.Use(header.RemoveRequest("Cookie"))
//
// # Security Headers
//
// The package includes convenience functions for common security headers:
//
//	app.Use(header.XSSProtection())     // X-XSS-Protection: 1; mode=block
//	app.Use(header.NoSniff())           // X-Content-Type-Options: nosniff
//	app.Use(header.FrameDeny())         // X-Frame-Options: DENY
//	app.Use(header.FrameSameOrigin())   // X-Frame-Options: SAMEORIGIN
//
// Configure HSTS (HTTP Strict Transport Security):
//
//	app.Use(header.HSTS(31536000, true, true)) // 1 year, includeSubDomains, preload
//
// Set Content Security Policy:
//
//	app.Use(header.CSP("default-src 'self'; script-src 'self' cdn.example.com"))
//
// Set Referrer Policy:
//
//	app.Use(header.ReferrerPolicy("strict-origin-when-cross-origin"))
//
// Set Permissions Policy:
//
//	app.Use(header.PermissionsPolicy("geolocation=(), microphone=()"))
//
// # Content-Type Headers
//
// Set content type with convenience functions:
//
//	app.Use(header.JSON())  // application/json; charset=utf-8
//	app.Use(header.HTML())  // text/html; charset=utf-8
//	app.Use(header.Text())  // text/plain; charset=utf-8
//	app.Use(header.XML())   // application/xml; charset=utf-8
//
// # Advanced Configuration
//
// For complex scenarios, use WithOptions for full control:
//
//	app.Use(header.WithOptions(header.Options{
//	    Response: map[string]string{
//	        "X-Frame-Options":        "DENY",
//	        "X-Content-Type-Options": "nosniff",
//	    },
//	    ResponseRemove: []string{"Server"},
//	    Request: map[string]string{
//	        "X-Request-Source": "web",
//	    },
//	    RequestRemove: []string{"Cookie"},
//	}))
//
// # Execution Order
//
// The middleware processes headers in the following order:
//
//  1. Set request headers (Options.Request)
//  2. Remove request headers (Options.RequestRemove)
//  3. Set response headers (Options.Response)
//  4. Execute next handler
//  5. Remove response headers (Options.ResponseRemove)
//
// This order ensures that request headers are available to downstream handlers,
// and response headers set by handlers can be properly removed if needed.
//
// # Performance Considerations
//
// The middleware uses a custom itoa function for integer-to-string conversion
// in the HSTS function to minimize memory allocations. All convenience functions
// delegate to WithOptions, which creates a single middleware closure.
//
// # Security Best Practices
//
//   - Remove sensitive headers (Server, X-Powered-By) to avoid information disclosure
//   - Set security headers at the application level for consistent protection
//   - Use HSTS for all production HTTPS applications
//   - Configure Content Security Policy to prevent XSS attacks
//   - Set appropriate Referrer-Policy to control referrer information
//
// For comprehensive security header management, consider using the helmet middleware
// which bundles multiple security headers together.
package header
