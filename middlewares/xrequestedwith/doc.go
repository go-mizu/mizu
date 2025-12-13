// Package xrequestedwith provides middleware for validating the X-Requested-With HTTP header.
//
// The X-Requested-With header is commonly used to identify AJAX requests and can serve as
// an additional layer of CSRF (Cross-Site Request Forgery) protection. This middleware
// validates the presence and value of this header for specified HTTP methods and paths.
//
// # Basic Usage
//
// The simplest usage requires the default "XMLHttpRequest" value for state-changing methods:
//
//	app := mizu.New()
//	app.Use(xrequestedwith.New())
//
// By default, this skips validation for GET, HEAD, and OPTIONS methods, as these are
// considered safe methods that don't modify server state.
//
// # Custom Configuration
//
// Use WithOptions to customize the middleware behavior:
//
//	app.Use(xrequestedwith.WithOptions(xrequestedwith.Options{
//	    Value:       "MyCustomValue",  // Require custom header value
//	    SkipMethods: []string{"GET"},  // Skip only GET requests
//	    SkipPaths:   []string{"/webhook", "/health"}, // Skip specific paths
//	    ErrorHandler: func(c *mizu.Ctx) error {
//	        return c.JSON(http.StatusForbidden, map[string]string{
//	            "error": "AJAX requests only",
//	        })
//	    },
//	}))
//
// # Function Variants
//
// The package provides several convenience functions:
//
// New() creates middleware with default settings (XMLHttpRequest, skip safe methods).
//
// WithOptions() allows full customization of all options.
//
// Require() creates middleware that validates a specific custom value:
//
//	app.Use(xrequestedwith.Require("FetchRequest"))
//
// AJAXOnly() creates strict validation for all HTTP methods (no methods skipped):
//
//	app.Use(xrequestedwith.AJAXOnly())
//
// # Detection Helper
//
// The IsAJAX helper function checks if a request has the X-Requested-With header
// without enforcing validation:
//
//	app.Get("/data", func(c *mizu.Ctx) error {
//	    if xrequestedwith.IsAJAX(c) {
//	        return c.JSON(200, data)  // Return JSON for AJAX
//	    }
//	    return c.HTML(200, page)      // Return HTML for browser
//	})
//
// # Implementation Details
//
// The middleware performs validation in the following order:
//
//  1. Check if the request method is in the skip list
//  2. Check if the request path is in the skip paths
//  3. Validate the X-Requested-With header value (case-insensitive)
//  4. Call error handler or return default 400 Bad Request if validation fails
//
// Header comparison is case-insensitive using strings.EqualFold, so "XMLHttpRequest",
// "xmlhttprequest", and "XMLHTTPREQUEST" are all considered equivalent.
//
// # Security Considerations
//
// While X-Requested-With provides an additional security layer, it should NOT be used
// as the sole CSRF protection mechanism. Best practices include:
//
//   - Combining with CSRF token validation
//   - Using SameSite cookies
//   - Implementing proper Origin/Referer validation
//   - Applying defense-in-depth security strategies
//
// The X-Requested-With header can be set by any client, so it only prevents simple
// CSRF attacks where the attacker cannot control request headers. Modern CSRF tokens
// provide stronger protection.
//
// # Client Integration
//
// Popular JavaScript libraries automatically set this header:
//
// jQuery sets it automatically for all AJAX requests:
//
//	$.ajax({url: '/api', type: 'POST', data: data});
//
// For native Fetch API or XMLHttpRequest, set it manually:
//
//	fetch('/api', {
//	    method: 'POST',
//	    headers: {
//	        'X-Requested-With': 'XMLHttpRequest'
//	    },
//	    body: JSON.stringify(data)
//	});
//
// # Performance
//
// The middleware uses pre-built maps for skip methods and paths, providing O(1)
// lookup performance. Header validation uses a single string comparison, making
// the middleware very efficient with minimal overhead.
package xrequestedwith
