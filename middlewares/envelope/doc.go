// Package envelope provides response envelope middleware for Mizu framework.
//
// The envelope middleware wraps all JSON responses in a consistent structure,
// adding metadata like status indicators, request IDs, and custom fields.
// This ensures uniform API responses across your application.
//
// # Basic Usage
//
// Use the default configuration with New():
//
//	app := mizu.New()
//	app.Use(envelope.New())
//
//	app.Get("/users", func(c *mizu.Ctx) error {
//	    return c.JSON(200, users)
//	})
//	// Response: {"success": true, "data": [...users]}
//
// # Custom Configuration
//
// Customize field names and metadata inclusion:
//
//	app.Use(envelope.WithOptions(envelope.Options{
//	    SuccessField: "ok",
//	    DataField:    "result",
//	    ErrorField:   "message",
//	    IncludeMeta:  true,
//	}))
//	// Response: {"ok": true, "result": {...}, "meta": {...}}
//
// # Helper Functions
//
// The package provides convenience functions for common HTTP responses:
//
//	app.Get("/user/:id", func(c *mizu.Ctx) error {
//	    user, err := findUser(c.Param("id"))
//	    if err != nil {
//	        return envelope.NotFound(c, "User not found")
//	    }
//	    return envelope.Success(c, user)
//	})
//
//	app.Post("/users", func(c *mizu.Ctx) error {
//	    user, err := createUser(c)
//	    if err != nil {
//	        return envelope.BadRequest(c, "Invalid user data")
//	    }
//	    return envelope.Created(c, user)
//	})
//
// # Response Structure
//
// Success responses (status 200-399):
//
//	{
//	    "success": true,
//	    "data": { ... }
//	}
//
// Error responses (status 400+):
//
//	{
//	    "success": false,
//	    "error": "Error message"
//	}
//
// With metadata enabled:
//
//	{
//	    "success": true,
//	    "data": { ... },
//	    "meta": {
//	        "status_code": 200,
//	        "request_id": "req-123"
//	    }
//	}
//
// # Implementation Details
//
// The middleware uses a custom response writer (envelopeWriter) to intercept
// and buffer the original response. After the handler completes:
//
//  1. Content-Type is checked against configured types (default: application/json)
//  2. Response body is parsed as JSON
//  3. Success status is determined from HTTP status code (200-399)
//  4. Original response is wrapped in the envelope structure
//  5. Envelope is marshaled and written to the original response writer
//
// Only responses with matching content types are wrapped. Non-matching responses
// pass through unmodified.
//
// # Content-Type Filtering
//
// By default, only application/json responses are wrapped. You can customize
// this with the ContentTypes option:
//
//	app.Use(envelope.WithOptions(envelope.Options{
//	    ContentTypes: []string{"application/json", "application/vnd.api+json"},
//	}))
//
// # Error Handling
//
// The middleware intelligently extracts error messages from responses:
//
//  1. Checks for "error" field in response body
//  2. Checks for "message" field in response body
//  3. Falls back to handler error if available
//  4. Otherwise includes the original response in the data field
//
// # Available Helper Functions
//
//   - Success(c, data) - 200 OK with data
//   - Created(c, data) - 201 Created with data
//   - NoContent(c) - 204 No Content
//   - BadRequest(c, message) - 400 Bad Request
//   - Unauthorized(c, message) - 401 Unauthorized
//   - Forbidden(c, message) - 403 Forbidden
//   - NotFound(c, message) - 404 Not Found
//   - InternalError(c, message) - 500 Internal Server Error
//
// These helpers automatically create properly formatted envelope responses
// with appropriate HTTP status codes.
package envelope
