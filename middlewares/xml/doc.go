// Package xml provides XML content handling middleware for Mizu web framework.
//
// # Overview
//
// The xml middleware enables XML content negotiation, parsing, and response helpers
// for building XML-based APIs. It supports SOAP compatibility and legacy system integration.
//
// # Features
//
//   - XML request body parsing with auto-parse option
//   - XML response generation with configurable formatting
//   - Content negotiation between XML and JSON formats
//   - Pretty printing support for development
//   - XML declaration control
//   - Error response formatting
//
// # Basic Usage
//
//	app := mizu.New()
//	app.Use(xml.New())
//
//	app.Get("/data", func(c *mizu.Ctx) error {
//	    return xml.Response(c, 200, data)
//	})
//
// # Parsing XML Requests
//
// Use Bind to parse XML request bodies:
//
//	type User struct {
//	    XMLName xml.Name `xml:"user"`
//	    ID      int      `xml:"id"`
//	    Name    string   `xml:"name"`
//	}
//
//	app.Post("/users", func(c *mizu.Ctx) error {
//	    var user User
//	    if err := xml.Bind(c, &user); err != nil {
//	        return err
//	    }
//	    return c.NoContent()
//	})
//
// # Content Negotiation
//
// The middleware supports automatic content negotiation based on the Accept header:
//
//	app.Use(xml.New())
//	app.Use(xml.ContentNegotiation())
//
//	app.Get("/data", func(c *mizu.Ctx) error {
//	    // Returns XML if Accept: application/xml
//	    // Returns JSON if Accept: application/json
//	    return xml.Respond(c, 200, data)
//	})
//
// # Configuration Options
//
// Customize middleware behavior with Options:
//
//	app.Use(xml.WithOptions(xml.Options{
//	    Indent:         "  ",              // Pretty print with 2 spaces
//	    Prefix:         "",                // No prefix
//	    ContentType:    "application/xml", // Default content type
//	    AutoParse:      true,              // Auto-parse XML bodies
//	    XMLDeclaration: true,              // Include <?xml ?> declaration
//	}))
//
// # Pretty Printing
//
// Enable formatted XML output for development:
//
//	app.Use(xml.Pretty("  ")) // 2-space indentation
//
// # Error Responses
//
// Send structured XML error responses:
//
//	app.Get("/resource/:id", func(c *mizu.Ctx) error {
//	    if !exists(c.Param("id")) {
//	        return xml.SendError(c, 404, "resource not found")
//	    }
//	    return xml.Response(c, 200, resource)
//	})
//
// # Wrapping Data
//
// Wrap data in a named root element:
//
//	users := []User{{ID: 1, Name: "Alice"}, {ID: 2, Name: "Bob"}}
//	wrapped := xml.Wrap("users", users)
//	return xml.Response(c, 200, wrapped)
//	// Output: <users><user>...</user><user>...</user></users>
//
// # Content Types
//
// The middleware recognizes the following XML content types:
//   - application/xml (standard)
//   - text/xml (alternative)
//
// # Context Storage
//
// The middleware uses context values to store:
//   - Parsed request body (when AutoParse is enabled)
//   - Middleware options
//   - Preferred response format (XML or JSON)
//
// These values are accessible through package functions like Body() and PreferredFormat().
package xml
