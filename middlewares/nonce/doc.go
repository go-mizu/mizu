// Package nonce provides middleware for generating cryptographic nonces and
// integrating them with Content Security Policy (CSP) headers in Mizu applications.
//
// # Overview
//
// The nonce middleware generates unique, cryptographically secure nonces for each
// HTTP request and automatically adds them to Content Security Policy headers. This
// enables secure inline script and style execution while maintaining strong CSP
// protection against XSS attacks.
//
// # Quick Start
//
// Basic usage with default settings:
//
//	app := mizu.New()
//	app.Use(nonce.New())
//
//	app.Get("/", func(c *mizu.Ctx) error {
//	    n := nonce.Get(c)
//	    return c.HTML(200, `<script nonce="`+n+`">alert('Safe!')</script>`)
//	})
//
// # Configuration
//
// The middleware can be customized using the Options struct:
//
//	app.Use(nonce.WithOptions(nonce.Options{
//	    Length:     32,                              // Nonce byte length (default: 16)
//	    Header:     "Content-Security-Policy",       // CSP header name (default)
//	    Directives: []string{"script-src", "style-src"}, // CSP directives (default)
//	    BasePolicy: "default-src 'self'",            // Existing policy to extend
//	    Generator:  customGeneratorFunc,             // Custom nonce generator
//	}))
//
// # Preset Middleware Functions
//
// The package provides several convenience constructors for common use cases:
//
// Script-only nonces:
//
//	app.Use(nonce.ForScripts())
//
// Style-only nonces:
//
//	app.Use(nonce.ForStyles())
//
// Extending an existing CSP policy:
//
//	app.Use(nonce.WithBasePolicy("default-src 'self'; img-src *"))
//
// Report-only mode for testing:
//
//	app.Use(nonce.ReportOnly())
//
// # Helper Functions
//
// The package provides helper functions for HTML attribute generation:
//
//	app.Get("/", func(c *mizu.Ctx) error {
//	    scriptAttr := nonce.ScriptTag(c) // Returns: nonce="abc123..."
//	    styleAttr := nonce.StyleTag(c)   // Returns: nonce="abc123..."
//	    html := fmt.Sprintf(`
//	        <script %s>console.log('safe');</script>
//	        <style %s>body { color: blue; }</style>
//	    `, scriptAttr, styleAttr)
//	    return c.HTML(200, html)
//	})
//
// # Integration with CSP
//
// The middleware automatically constructs CSP headers with nonce values:
//
//	app.Use(nonce.New())
//	// Response header: Content-Security-Policy: script-src 'self' 'nonce-abc123...'; style-src 'self' 'nonce-abc123...'
//
// When using with a base policy:
//
//	app.Use(nonce.WithOptions(nonce.Options{
//	    BasePolicy: "default-src 'self'; img-src *; font-src 'self'",
//	}))
//	// Nonces are added to script-src and style-src while preserving other directives
//
// # Security Considerations
//
// The nonce middleware implements several security best practices:
//
//   - Cryptographic randomness: Uses crypto/rand for secure random generation
//   - Per-request uniqueness: Each request generates a new nonce
//   - Base64 encoding: Nonces are base64-encoded for safe HTML embedding
//   - CSP integration: Automatically formats nonces per CSP specification ('nonce-...')
//
// # Implementation Details
//
// Nonce Generation:
//   - Default length: 16 bytes (produces 22 character base64 strings)
//   - Encoding: Base64 raw standard encoding (no padding)
//   - Source: crypto/rand for cryptographic security
//
// Context Storage:
//   - Nonces are stored in request context with a private key type
//   - Type-safe retrieval via Get() function
//   - No risk of key collision with other middleware
//
// CSP Header Construction:
//   - Parses existing base policies when provided
//   - Adds nonce to specified directives (default: script-src, style-src)
//   - Format: 'nonce-{base64-value}' as per CSP Level 3 specification
//   - Merges with existing directive values when base policy is present
//
// # Testing
//
// The package includes comprehensive test coverage for:
//   - Basic nonce generation and CSP header setting
//   - Custom length configuration
//   - Custom directive specification
//   - Base policy extension
//   - Custom generator functions
//   - Helper functions (ScriptTag, StyleTag)
//   - Preset middleware functions
//   - Nonce uniqueness across requests
//   - Behavior without middleware (graceful degradation)
//
// # Examples
//
// Complete example with CSP integration:
//
//	package main
//
//	import (
//	    "fmt"
//	    "github.com/go-mizu/mizu"
//	    "github.com/go-mizu/mizu/middlewares/nonce"
//	)
//
//	func main() {
//	    app := mizu.New()
//
//	    // Apply nonce middleware
//	    app.Use(nonce.WithOptions(nonce.Options{
//	        BasePolicy: "default-src 'self'",
//	    }))
//
//	    app.Get("/", func(c *mizu.Ctx) error {
//	        n := nonce.Get(c)
//	        html := fmt.Sprintf(`
//	            <!DOCTYPE html>
//	            <html>
//	            <head>
//	                <style nonce="%s">
//	                    body { font-family: Arial; }
//	                </style>
//	            </head>
//	            <body>
//	                <h1>CSP with Nonces</h1>
//	                <script nonce="%s">
//	                    console.log('This inline script is allowed!');
//	                </script>
//	            </body>
//	            </html>
//	        `, n, n)
//	        return c.HTML(200, html)
//	    })
//
//	    app.Listen(":3000")
//	}
package nonce
