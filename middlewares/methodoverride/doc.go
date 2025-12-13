// Package methodoverride provides HTTP method override middleware for Mizu.
//
// This middleware enables HTML forms to use HTTP methods beyond GET and POST by
// allowing method override through headers, query parameters, or form fields.
// Since HTML forms only natively support GET and POST methods, this middleware
// provides a standard way to handle PUT, PATCH, and DELETE operations.
//
// # Basic Usage
//
// The simplest way to use the middleware is with default settings:
//
//	app := mizu.New()
//	app.Use(methodoverride.New())
//
// # Override Methods
//
// The middleware checks for method override in the following priority order:
//
//  1. HTTP Header (default: X-HTTP-Method-Override)
//  2. Query Parameter (default: _method)
//  3. Form Field (default: _method)
//
// # HTML Form Example
//
// To submit a PUT request from an HTML form:
//
//	<form method="POST" action="/users/123">
//	    <input type="hidden" name="_method" value="PUT">
//	    <input type="text" name="name" value="John">
//	    <button type="submit">Update</button>
//	</form>
//
// # AJAX Example
//
// For AJAX requests, use the header approach:
//
//	fetch('/users/123', {
//	    method: 'POST',
//	    headers: {
//	        'X-HTTP-Method-Override': 'PUT',
//	        'Content-Type': 'application/json'
//	    },
//	    body: JSON.stringify({name: 'John'})
//	});
//
// # Configuration
//
// Customize the middleware behavior with options:
//
//	app.Use(methodoverride.WithOptions(methodoverride.Options{
//	    Header:    "X-Method",              // Custom header name
//	    FormField: "method",                // Custom form field name
//	    Methods:   []string{"PUT", "DELETE"}, // Restrict allowed methods
//	}))
//
// # Security
//
// The middleware implements several security measures:
//
//   - Only POST requests can be overridden
//   - Only explicitly allowed methods are permitted (default: PUT, PATCH, DELETE)
//   - Invalid methods are silently ignored
//   - Form field checking only occurs for appropriate content types
//
// Always use this middleware in combination with CSRF protection when handling
// form submissions:
//
//	app.Use(csrf.New())
//	app.Use(methodoverride.New())
//
// # Implementation Details
//
// The middleware validates override values case-insensitively and converts them
// to uppercase. Allowed methods are stored in a map for O(1) lookup performance.
//
// Form field checking only occurs when the request content type is either
// application/x-www-form-urlencoded or multipart/form-data, avoiding unnecessary
// form parsing for JSON or other content types.
package methodoverride
