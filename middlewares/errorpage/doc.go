// Package errorpage provides custom error page middleware for handling HTTP errors.
//
// The errorpage middleware intercepts HTTP error responses (status codes >= 400)
// and replaces them with custom error pages. It uses a response writer wrapper to
// capture status codes before they are sent to the client, allowing conditional
// rendering of branded error pages.
//
// # Features
//
//   - Default error pages for common HTTP error codes (400, 401, 403, 404, 405, 500, 502, 503, 504)
//   - Customizable HTML templates using Go's html/template package
//   - Custom error handlers for specific status codes
//   - Custom 404 handler support
//   - Response writer wrapper that intercepts status codes without immediate propagation
//
// # Basic Usage
//
//	app := mizu.New()
//	app.Use(errorpage.New())
//
// # Custom Error Pages
//
// You can configure custom error pages for specific status codes:
//
//	app.Use(errorpage.WithOptions(errorpage.Options{
//	    Pages: map[int]*errorpage.Page{
//	        404: {Code: 404, Title: "Not Found", Message: "The page you requested does not exist."},
//	        500: {Code: 500, Title: "Server Error", Message: "Something went wrong on our end."},
//	    },
//	}))
//
// # Custom Templates
//
// Override the default template with your own HTML template:
//
//	app.Use(errorpage.WithOptions(errorpage.Options{
//	    DefaultTemplate: `<!DOCTYPE html>
//	    <html>
//	    <head><title>{{.Title}}</title></head>
//	    <body>
//	        <h1>Error {{.Code}}</h1>
//	        <p>{{.Message}}</p>
//	    </body>
//	    </html>`,
//	}))
//
// # Custom Handlers
//
// Use custom handlers for specific error codes or all errors:
//
//	app.Use(errorpage.WithOptions(errorpage.Options{
//	    NotFoundHandler: func(c *mizu.Ctx) error {
//	        return c.JSON(404, map[string]string{"error": "not found"})
//	    },
//	    ErrorHandler: func(c *mizu.Ctx, code int) error {
//	        return c.JSON(code, map[string]string{"error": "an error occurred"})
//	    },
//	}))
//
// # Implementation Details
//
// The middleware uses a statusWriter wrapper that implements http.ResponseWriter
// to intercept WriteHeader() calls. This allows the middleware to:
//
//   - Capture status codes without immediately sending them to the client
//   - Determine whether a response body has been written
//   - Conditionally render error pages only when appropriate (status >= 400 and no body written)
//   - Execute custom error handlers before rendering default pages
//
// The error handling flow is:
//
//  1. Request enters middleware
//  2. Response writer is wrapped with statusWriter
//  3. Next handler is called
//  4. If status code >= 400 and no body written:
//     - Custom error handler is invoked (if configured)
//     - Custom 404 handler is invoked for 404s (if configured)
//     - Appropriate error page is rendered using templates
//  5. Response is sent to client
//
// # Helper Functions
//
// The package provides several helper functions for common use cases:
//
//   - New() - Creates middleware with default error pages
//   - WithOptions(opts Options) - Creates middleware with custom configuration
//   - Custom(pages map[int]*Page) - Creates middleware with custom pages
//   - NotFound() - Creates middleware that shows a simple 404 page
//   - Page404(title, message string) - Helper to create a 404 page
//   - Page500(title, message string) - Helper to create a 500 page
//
// # Best Practices
//
//   - Keep error pages simple and fast to render
//   - Include navigation links back to the home page
//   - Log errors for monitoring and debugging
//   - Use different pages for client errors (4xx) vs server errors (5xx)
//   - Avoid making external API calls in error page templates
//   - Test error pages thoroughly to ensure they don't introduce additional errors
package errorpage
