// Package contenttype provides middleware for Content-Type validation and enforcement in Mizu applications.
//
// # Overview
//
// The contenttype package offers a set of middleware functions that validate incoming request
// Content-Type headers and set response Content-Type headers. It ensures that your API endpoints
// receive the expected content types and can enforce consistent response types.
//
// Key Features
//
//   - Request Content-Type validation with customizable allowed types
//   - Automatic parameter stripping (charset, boundary) for flexible matching
//   - Method-aware validation (only POST, PUT, PATCH by default)
//   - Default Content-Type injection for requests without headers
//   - Response Content-Type setting for consistent API responses
//
// # Usage
//
// Basic JSON-only endpoint:
//
//	app := mizu.New()
//	app.Use(contenttype.RequireJSON())
//
//	app.Post("/api/users", func(c *mizu.Ctx) error {
//	    var user User
//	    if err := c.BindJSON(&user); err != nil {
//	        return err
//	    }
//	    return c.JSON(201, user)
//	})
//
// Form submission validation:
//
//	app.Post("/contact", contactHandler, contenttype.RequireForm())
//
// Multiple allowed content types:
//
//	app.Use(contenttype.Require("application/json", "application/xml"))
//
// Set default for missing Content-Type:
//
//	app.Use(contenttype.Default("application/json"))
//
// Set response Content-Type:
//
//	app.Use(contenttype.SetResponse("application/json; charset=utf-8"))
//
// Per-route configuration:
//
//	api := app.Group("/api")
//	api.Use(contenttype.RequireJSON())
//
//	forms := app.Group("/forms")
//	forms.Use(contenttype.RequireForm())
//
// # Validation Behavior
//
// The middleware only validates requests with body content (POST, PUT, PATCH methods).
// GET, DELETE, and other methods bypass validation entirely.
//
// During validation, the middleware:
//
//   - Extracts the Content-Type header from the request
//   - Strips media type parameters (e.g., "application/json; charset=utf-8" becomes "application/json")
//   - Performs case-insensitive comparison against allowed types
//   - Returns HTTP 415 (Unsupported Media Type) if validation fails
//
// # Error Responses
//
// When validation fails, the middleware returns:
//
//   - 415 Unsupported Media Type with body "Content-Type required" if header is missing
//   - 415 Unsupported Media Type with body "Unsupported Media Type" if header doesn't match
//
// Common Content Types
//
//	application/json                      - JSON data
//	application/xml                       - XML data
//	application/x-www-form-urlencoded     - URL-encoded form data
//	multipart/form-data                   - File uploads and multipart forms
//	text/plain                            - Plain text
//
// Best Practices
//
//   - Use RequireJSON() for REST API endpoints to ensure JSON payloads
//   - Use RequireForm() for traditional HTML form submissions
//   - Combine Default() with Require() to provide fallback behavior
//   - Apply SetResponse() for consistent API response content types
//   - Configure middleware at the router group level for route-specific rules
//
// # Design Decisions
//
// Parameter Stripping: The middleware ignores media type parameters during validation,
// allowing flexible matching. For example, "application/json; charset=utf-8" will match
// a requirement for "application/json".
//
// Method-Specific: Only HTTP methods that typically include request bodies (POST, PUT, PATCH)
// are validated. This prevents unnecessary validation on GET, DELETE, and similar methods.
//
// Fail-Fast: Validation occurs before the request handler executes, preventing invalid
// requests from reaching your application logic.
//
// Composable: Multiple middleware functions can be chained together for complex scenarios,
// such as requiring specific input types while setting specific output types.
package contenttype
