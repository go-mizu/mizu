// Package embed provides middleware for serving files from Go's embedded filesystem.
//
// The embed middleware enables serving static files, templates, and assets from
// embedded filesystems (embed.FS), allowing for single-binary deployments without
// external file dependencies.
//
// # Basic Usage
//
// The simplest way to use the embed middleware is with the New function:
//
//	//go:embed static/*
//	var staticFS embed.FS
//
//	app := mizu.New()
//	app.Use(embed.New(staticFS))
//
// This serves files from the root of the embedded filesystem.
//
// # Serving from Subdirectories
//
// Use the Static helper to serve files from a specific subdirectory:
//
//	//go:embed assets/*
//	var assetsFS embed.FS
//
//	app.Use(embed.Static(assetsFS, "assets"))
//	// Files at assets/css/app.css will be available at /css/app.css
//
// # Configuration Options
//
// For more control, use WithOptions with the Options struct:
//
//	app.Use(embed.WithOptions(fsys, embed.Options{
//	    Root:    "public",           // Serve from this subdirectory
//	    Index:   "index.html",        // Index file name
//	    Browse:  false,               // Directory browsing
//	    MaxAge:  3600,                // Cache-Control max-age in seconds
//	    NotFoundHandler: customHandler, // Custom 404 handler
//	}))
//
// # Caching
//
// Enable caching with the WithCaching helper:
//
//	app.Use(embed.WithCaching(staticFS, 86400)) // Cache for 24 hours
//
// # Single-Page Applications
//
// The SPA function provides special handling for single-page applications,
// serving index.html for routes that don't match actual files:
//
//	//go:embed dist/*
//	var distFS embed.FS
//
//	app.Use(embed.SPA(distFS, "dist/index.html"))
//	// GET /app/dashboard -> serves index.html
//	// GET /assets/app.js -> serves actual file
//
// # Handler Functions
//
// The package also provides handler functions (non-middleware) for direct use:
//
//	app.Get("/static/{path...}", embed.Handler(staticFS))
//	app.Get("/assets/{path...}", embed.HandlerWithOptions(assetsFS, embed.Options{
//	    Root:   "assets",
//	    MaxAge: 3600,
//	}))
//
// # Implementation Details
//
// The middleware uses Go's io/fs.FS interface and http.FileServer for file serving.
// When a file is not found, it can either:
//   - Call a custom NotFoundHandler if configured
//   - Pass through to the next middleware (for middleware functions)
//   - Return a 404 error (for handler functions)
//
// Path handling includes automatic cleaning and normalization. Directory requests
// automatically try to serve the index file (default: index.html).
//
// # Middleware vs Handler
//
// The package provides two types of functions:
//
// Middleware functions (New, WithOptions, Static, SPA, WithCaching):
//   - Fall through to the next handler if file not found
//   - Useful for serving static files alongside dynamic routes
//
// Handler functions (Handler, HandlerWithOptions):
//   - Do not fall through, serve files directly
//   - Useful when you want dedicated routes for file serving
//
// # Performance Considerations
//
// - Embedded files are compiled into the binary and loaded into memory
// - Keep embedded files reasonably sized to avoid large binaries
// - The custom itoa function avoids allocations when setting cache headers
// - http.FileServer handles range requests, conditional requests, and content-type detection
//
// # Security Notes
//
// - Path traversal is prevented by path.Clean
// - Only files explicitly embedded are accessible
// - Directory browsing is disabled by default
// - The Browse option is present but not currently implemented
package embed
