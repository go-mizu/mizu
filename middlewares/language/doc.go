// Package language provides content negotiation middleware for detecting and managing
// user language preferences in Mizu web applications.
//
// # Overview
//
// The language middleware automatically detects the user's preferred language from multiple
// sources and makes it available throughout the request lifecycle. It supports:
//
//   - Query parameter detection (?lang=es)
//   - Cookie-based persistence
//   - Accept-Language header parsing with quality values
//   - Path prefix detection (/en/page)
//   - Regional language variants (en-US, en-GB)
//
// # Detection Priority
//
// The middleware checks sources in the following order:
//
//  1. Query parameter (highest priority)
//  2. Path prefix (if enabled)
//  3. Cookie
//  4. Accept-Language header
//  5. Default language (fallback)
//
// # Basic Usage
//
// Simple language detection with default settings:
//
//	app := mizu.New()
//	app.Use(language.New("en", "es", "fr"))
//
//	app.Get("/", func(c *mizu.Ctx) error {
//	    lang := language.Get(c)
//	    return c.JSON(200, map[string]string{"language": lang})
//	})
//
// # Advanced Configuration
//
// Customize detection sources and behavior:
//
//	app.Use(language.WithOptions(language.Options{
//	    Supported:  []string{"en", "es", "fr", "de"},
//	    Default:    "en",
//	    QueryParam: "lang",
//	    CookieName: "preferred_language",
//	    PathPrefix: true,
//	}))
//
// # Path Prefix Detection
//
// When PathPrefix is enabled, the middleware automatically strips the language
// prefix from the URL path:
//
//	// Request: GET /fr/api/users
//	// Detected language: "fr"
//	// Path after middleware: /api/users
//
// # Regional Variants
//
// The middleware supports both base language codes and regional variants:
//
//	app.Use(language.New("en", "en-US", "en-GB"))
//
//	// Supports: en, en-US, en-GB
//	// Automatically matches "en" to "en-US" if "en" is not explicitly supported
//
// # Accept-Language Header
//
// The middleware properly parses Accept-Language headers with quality values:
//
//	Accept-Language: en;q=0.5, fr;q=0.9, es;q=0.3
//	// Result: "fr" (highest quality value)
//
// # Context Retrieval
//
// Two equivalent methods for retrieving the detected language:
//
//	lang := language.Get(c)         // Primary method
//	lang := language.FromContext(c) // Alias for compatibility
//
// # Implementation Details
//
// The middleware uses the following internal mechanisms:
//
//   - Context storage with a private contextKey type for isolation
//   - Case-insensitive language matching for user convenience
//   - Quality-based sorting for Accept-Language header parsing
//   - Efficient map-based lookup for supported language validation
//
// # Best Practices
//
//   - Use ISO 639-1 language codes (en, es, fr, de, ja, etc.)
//   - Provide a language switcher UI for explicit user control
//   - Persist user preferences in cookies for consistency
//   - Support common language variants for your target audience
//   - Always specify a sensible default language
//
// # Thread Safety
//
// The middleware is safe for concurrent use. Each request gets its own context
// with the detected language stored independently.
package language
