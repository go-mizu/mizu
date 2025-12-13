// Package favicon provides favicon serving middleware for Mizu framework.
//
// The favicon middleware efficiently serves favicon files by caching them in memory
// and handling GET and HEAD requests properly. It prevents favicon requests from
// reaching application routes and provides multiple configuration options.
//
// # Features
//
//   - Memory caching for optimal performance
//   - Automatic content-type detection
//   - Support for multiple data sources (file, bytes, fs.FS)
//   - Configurable URL path and cache duration
//   - HEAD request support
//   - SVG favicon support
//   - Empty and redirect modes
//
// # Basic Usage
//
// Serve favicon from a file:
//
//	app := mizu.New()
//	app.Use(favicon.New("./static/favicon.ico"))
//
// # Advanced Usage
//
// Serve from embedded data with custom cache control:
//
//	//go:embed favicon.ico
//	var faviconData []byte
//
//	app.Use(favicon.WithOptions(favicon.Options{
//	    Data:   faviconData,
//	    MaxAge: 86400, // 24 hours
//	}))
//
// Serve from fs.FS interface:
//
//	app.Use(favicon.WithOptions(favicon.Options{
//	    FS:   embedFS,
//	    File: "assets/favicon.ico",
//	}))
//
// Custom URL path:
//
//	app.Use(favicon.WithOptions(favicon.Options{
//	    Data: iconData,
//	    URL:  "/icon.png",
//	}))
//
// # Special Modes
//
// Return empty response (204 No Content):
//
//	app.Use(favicon.Empty())
//
// Redirect to external URL:
//
//	app.Use(favicon.Redirect("https://cdn.example.com/favicon.ico"))
//
// Serve SVG favicon:
//
//	svgData := []byte(`<svg>...</svg>`)
//	app.Use(favicon.SVG(svgData))
//
// # Implementation Details
//
// The middleware loads favicon data once during initialization and caches it in memory.
// No disk I/O occurs during request handling, ensuring optimal performance.
//
// Content-Type is automatically detected using http.DetectContentType(), with a fallback
// to "image/x-icon" for ICO files that may not be recognized.
//
// The middleware only intercepts requests matching the configured URL path (default: /favicon.ico)
// and only handles GET and HEAD methods. All other requests pass through to the next handler.
//
// Cache-Control headers are set with a default max-age of 86400 seconds (24 hours),
// which can be customized via the Options.MaxAge field.
//
// # Error Handling
//
// File loading errors during middleware initialization will cause a panic, ensuring
// misconfiguration is caught during application startup rather than at runtime.
//
// # Performance
//
//   - Single file read at startup
//   - Zero disk I/O during request handling
//   - Minimal memory allocation per request
//   - Early path matching prevents unnecessary processing
//
// # Best Practices
//
//   - Use embedded data (go:embed) for portable deployments
//   - Set appropriate cache duration based on deployment frequency
//   - Place favicon middleware before other middleware to avoid unnecessary processing
//   - Use Empty() mode if no favicon is needed to prevent 404 logs
package favicon
