// Package hypermedia provides HATEOAS (Hypermedia as the Engine of Application State)
// middleware for building self-documenting RESTful APIs with hypermedia links.
//
// # Overview
//
// The hypermedia middleware automatically injects hypermedia links into JSON responses,
// enabling API discoverability and client navigation. It supports multiple formats
// including HAL+JSON and custom link structures.
//
// # Features
//
//   - Automatic self-link generation
//   - Custom link injection via handlers
//   - HAL+JSON format support
//   - Embedded resource support
//   - Pagination link generation
//   - Dynamic link providers
//   - Custom links key configuration
//
// # Basic Usage
//
//	import "github.com/go-mizu/mizu/middlewares/hypermedia"
//
//	app := mizu.New()
//	app.Use(hypermedia.New())
//
//	app.Get("/users/:id", func(c *mizu.Ctx) error {
//	    user := getUser(c.Param("id"))
//	    hypermedia.AddLink(c, hypermedia.Link{
//	        Href: "/users/" + user.ID + "/orders",
//	        Rel:  "orders",
//	    })
//	    return c.JSON(200, user)
//	})
//
// # Configuration
//
// The middleware can be configured with custom options:
//
//	app.Use(hypermedia.WithOptions(hypermedia.Options{
//	    BaseURL:  "https://api.example.com",
//	    SelfLink: true,
//	    LinksKey: "_links",
//	    LinkProvider: func(path string, method string) hypermedia.Links {
//	        // Return context-aware links
//	        return nil
//	    },
//	}))
//
// # HAL+JSON Support
//
// Create HAL-compliant responses with embedded resources:
//
//	user := hypermedia.NewHAL(map[string]any{
//	    "id":   "123",
//	    "name": "John Doe",
//	})
//	user.AddLink("self", hypermedia.Link{Href: "/users/123"})
//	user.AddLink("orders", hypermedia.Link{Href: "/users/123/orders"})
//
//	// Embed related resources
//	order := hypermedia.NewHAL(map[string]any{"id": "456"})
//	user.Embed("orders", *order)
//
//	return c.JSON(200, user)
//
// # Collections and Pagination
//
// Generate paginated collections with navigation links:
//
//	collection := hypermedia.NewCollection(
//	    users,           // items
//	    100,             // total items
//	    2,               // current page
//	    10,              // page size
//	    "/users",        // base URL
//	)
//
// This automatically generates first, prev, next, and last links based on
// the pagination state.
//
// # Link Management
//
// Links can be added, retrieved, and replaced during request handling:
//
//	// Add a single link
//	hypermedia.AddLink(c, hypermedia.Link{Href: "/path", Rel: "related"})
//
//	// Add multiple links
//	hypermedia.AddLinks(c,
//	    hypermedia.Link{Href: "/path1", Rel: "rel1"},
//	    hypermedia.Link{Href: "/path2", Rel: "rel2"},
//	)
//
//	// Get current links
//	links := hypermedia.GetLinks(c)
//
//	// Replace all links
//	hypermedia.SetLinks(c, newLinks)
//
// # Architecture
//
// The middleware uses a response recorder pattern:
//
//  1. Links are stored in request context
//  2. Response is buffered using a custom recorder
//  3. JSON responses are parsed and modified
//  4. Links are injected into the JSON structure
//  5. Modified response is written to client
//
// Only JSON responses (Content-Type: application/json) are processed.
// Other content types pass through unchanged.
//
// # Performance
//
//   - Links stored as pointers to avoid copying
//   - JSON parsing only for application/json responses
//   - Non-JSON responses have zero overhead
//   - Minimal buffering impact
//
// # Security
//
//   - Base URL validation prevents injection
//   - Links added server-side only
//   - Automatic TLS detection for scheme
//   - No user input in link generation
//
// # Best Practices
//
//   - Always include a self link for resources
//   - Use consistent link relation names
//   - Document custom link relations
//   - Consider HAL+JSON for standardization
//   - Avoid exposing sensitive data in URLs
//   - Use link providers for dynamic links
//
// # Link Structure
//
// Links follow this structure:
//
//	type Link struct {
//	    Href   string // Required: URL of the linked resource
//	    Rel    string // Required: Relationship type
//	    Method string // Optional: HTTP method (GET, POST, etc.)
//	    Title  string // Optional: Human-readable description
//	    Type   string // Optional: Media type hint
//	}
//
// # Example Output
//
// With the middleware enabled, a simple JSON response:
//
//	{"id": "123", "name": "John"}
//
// Becomes a hypermedia response:
//
//	{
//	    "id": "123",
//	    "name": "John",
//	    "_links": [
//	        {
//	            "href": "https://api.example.com/users/123",
//	            "rel": "self",
//	            "method": "GET"
//	        }
//	    ]
//	}
package hypermedia
