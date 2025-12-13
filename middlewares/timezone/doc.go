// Package timezone provides timezone detection middleware for Mizu web applications.
//
// The timezone middleware automatically detects and stores user timezone information
// from multiple sources including HTTP headers, cookies, and query parameters. It
// provides a flexible and configurable solution for time localization in web applications.
//
// # Features
//
//   - Multi-source timezone detection (headers, cookies, query parameters)
//   - Configurable lookup order and precedence
//   - Automatic fallback to default timezone for invalid inputs
//   - Optional cookie management for timezone persistence
//   - Helper functions for common timezone operations
//   - IANA timezone database support
//
// # Basic Usage
//
//	app := mizu.New()
//	app.Use(timezone.New())
//
//	app.Get("/time", func(c *mizu.Ctx) error {
//	    tz := timezone.Get(c)
//	    now := time.Now().In(tz.Location)
//	    return c.JSON(200, map[string]interface{}{
//	        "timezone": tz.Name,
//	        "time":     now.Format(time.RFC3339),
//	        "offset":   tz.Offset,
//	    })
//	})
//
// # Configuration
//
// The middleware can be configured using the Options struct:
//
//	app.Use(timezone.WithOptions(timezone.Options{
//	    Header:       "X-Timezone",           // Header to check for timezone
//	    Cookie:       "timezone",             // Cookie name to check
//	    QueryParam:   "tz",                   // Query parameter to check
//	    Default:      "UTC",                  // Default timezone if not detected
//	    SetCookie:    true,                   // Automatically set timezone cookie
//	    CookieMaxAge: 30 * 24 * 60 * 60,     // Cookie max age (30 days)
//	    Lookup:       "header,cookie,query",  // Detection order
//	}))
//
// # Detection Sources
//
// The middleware checks multiple sources in configurable order:
//
//  1. HTTP Header (default: "X-Timezone")
//  2. Cookie (default: "timezone")
//  3. Query Parameter (default: "tz")
//
// The lookup order can be customized using the Lookup option. The first non-empty
// value found is used as the timezone.
//
// # Helper Functions
//
// The package provides several helper functions for common operations:
//
//   - Get(c): Returns complete timezone Info struct
//   - Location(c): Returns time.Location for time conversions
//   - Name(c): Returns timezone name string
//   - Offset(c): Returns UTC offset in seconds
//   - Now(c): Returns current time in detected timezone
//
// # Convenience Constructors
//
// Several convenience constructors are provided for common use cases:
//
//	// Only check specific header
//	app.Use(timezone.FromHeader("X-User-Timezone"))
//
//	// Only check specific cookie
//	app.Use(timezone.FromCookie("user_tz"))
//
//	// Set custom default timezone
//	app.Use(timezone.WithDefault("America/New_York"))
//
// # Client-Side Integration
//
// For accurate timezone detection, use JavaScript to detect the client timezone
// and send it via cookie:
//
//	document.cookie = `tz=${Intl.DateTimeFormat().resolvedOptions().timeZone}`;
//
// # Best Practices
//
//   - Always store times in UTC in your database
//   - Convert to user timezone only for display purposes
//   - Use IANA timezone names (e.g., "America/New_York", not "EST")
//   - Prefer client-side detection via JavaScript for accuracy
//   - Set the timezone cookie to avoid repeated header/query lookups
//
// # Error Handling
//
// The middleware handles invalid timezone names gracefully by falling back to
// the configured default timezone (UTC by default). This ensures the application
// continues to function even with malformed input.
//
// # Context Storage
//
// Timezone information is stored in the request context as an Info struct containing:
//   - Name: IANA timezone identifier
//   - Location: Parsed time.Location object
//   - Offset: UTC offset in seconds
//
// This information remains available throughout the request lifecycle and can be
// accessed using the provided helper functions.
package timezone
