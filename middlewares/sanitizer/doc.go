// Package sanitizer provides request sanitization middleware for the Mizu web framework.
//
// The sanitizer middleware protects applications from injection attacks by cleaning
// and normalizing user input. It processes query parameters, form data, and other
// request inputs through a configurable pipeline of sanitization operations.
//
// # Features
//
// The sanitizer middleware offers multiple sanitization strategies:
//
//   - HTML escaping: Converts special characters to HTML entities
//   - HTML tag stripping: Removes HTML tags including script and style elements
//   - Whitespace trimming: Removes leading and trailing spaces
//   - Non-printable character removal: Filters control characters while preserving newlines, tabs, and carriage returns
//   - Length limiting: Truncates values to a maximum length
//   - Field-level control: Whitelist or blacklist specific fields
//
// # Basic Usage
//
// Use the default sanitizer with sensible defaults for XSS prevention:
//
//	app := mizu.New()
//	app.Use(sanitizer.New())
//
// The default configuration enables HTML escaping, whitespace trimming, and
// non-printable character removal.
//
// # Custom Configuration
//
// Configure specific sanitization rules using WithOptions:
//
//	app.Use(sanitizer.WithOptions(sanitizer.Options{
//	    StripTags:  true,
//	    TrimSpaces: true,
//	    MaxLength:  1000,
//	    Fields:     []string{"name", "email", "comment"},
//	}))
//
// # Preset Configurations
//
// The package provides preset configurations for common use cases:
//
//	// XSS prevention (escapes HTML)
//	app.Use(sanitizer.XSS())
//
//	// Strip all HTML tags
//	app.Use(sanitizer.StripHTML())
//
//	// Only trim whitespace
//	app.Use(sanitizer.Trim())
//
// # Field Filtering
//
// Control which fields are sanitized using whitelist or blacklist approaches:
//
//	// Only sanitize specific fields
//	app.Use(sanitizer.WithOptions(sanitizer.Options{
//	    HTMLEscape: true,
//	    Fields:     []string{"user_input", "comment"},
//	}))
//
//	// Sanitize all fields except specific ones
//	app.Use(sanitizer.WithOptions(sanitizer.Options{
//	    HTMLEscape: true,
//	    Exclude:    []string{"html_content", "raw_data"},
//	}))
//
// # Standalone Functions
//
// The package also exports standalone functions for sanitizing individual values:
//
//	// Custom sanitization with options
//	clean := sanitizer.Sanitize(userInput, sanitizer.Options{
//	    HTMLEscape: true,
//	    TrimSpaces: true,
//	})
//
//	// HTML sanitization helper
//	safe := sanitizer.SanitizeHTML(untrustedHTML)
//
//	// Strip HTML tags
//	text := sanitizer.StripTagsString("<p>Hello</p>")
//
//	// Trim whitespace
//	trimmed := sanitizer.TrimString("  hello  ")
//
//	// Apply all sanitization operations
//	cleaned := sanitizer.Clean(dirtyInput)
//
// # Sanitization Pipeline
//
// When sanitizing values, operations are applied in this order:
//
//  1. Trim spaces (if enabled)
//  2. Strip non-printable characters (if enabled)
//  3. Strip HTML tags (if enabled)
//  4. HTML escape (if enabled)
//  5. Truncate to max length (if configured)
//
// # Request Processing
//
// The middleware processes different types of request data:
//
//   - Query parameters: Always sanitized
//   - Form data: Sanitized for POST, PUT, and PATCH requests
//   - Both r.Form and r.PostForm are processed
//
// # Security Considerations
//
// While the sanitizer middleware provides important defense-in-depth protection,
// it should not be your only security measure:
//
//   - Use parameterized queries to prevent SQL injection
//   - Apply proper output encoding in templates
//   - Implement Content Security Policy headers
//   - Validate data types and formats with the validator middleware
//   - Consider context-specific sanitization (e.g., different rules for plain text vs. rich content)
//
// The sanitizer is designed to prevent common injection attacks but should be
// combined with other security best practices for comprehensive protection.
//
// # Performance
//
// The middleware is optimized for performance:
//
//   - Field and exclude maps are pre-built at initialization
//   - Regex patterns for tag stripping are compiled once
//   - Request data is modified in-place to minimize allocations
//   - Field lookups use hash maps for O(1) complexity
//
// # Example Application
//
//	package main
//
//	import (
//	    "github.com/go-mizu/mizu"
//	    "github.com/go-mizu/mizu/middlewares/sanitizer"
//	)
//
//	func main() {
//	    app := mizu.New()
//
//	    // Apply sanitization to all routes
//	    app.Use(sanitizer.New())
//
//	    app.Post("/comment", func(c *mizu.Ctx) error {
//	        // Input is already sanitized
//	        comment := c.FormValue("comment")
//	        // ... process comment safely
//	        return c.JSON(200, map[string]string{"status": "ok"})
//	    })
//
//	    app.Listen(":3000")
//	}
package sanitizer
