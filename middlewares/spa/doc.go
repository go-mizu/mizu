// Package spa provides middleware for serving Single Page Applications (SPAs)
// in Mizu web applications.
//
// # Overview
//
// The spa middleware enables proper hosting of client-side routed applications
// (React, Vue, Angular, etc.) by implementing a fallback mechanism that serves
// the index file for any unmatched routes. This allows the client-side router
// to handle routing without server-side 404 errors.
//
// # Basic Usage
//
// Serve SPA from a directory:
//
//	app := mizu.New()
//	app.Use(spa.New("./dist"))
//
// # Using Embedded Filesystem
//
// For portable deployments, use Go's embed.FS:
//
//	//go:embed dist/*
//	var distFS embed.FS
//
//	app := mizu.New()
//	app.Use(spa.WithFS(distFS))
//
// # Configuration Options
//
// The Options struct provides fine-grained control:
//
//	app.Use(spa.WithOptions(spa.Options{
//	    Root:         "./dist",           // Root directory to serve from
//	    FS:           nil,                 // Optional fs.FS (takes precedence over Root)
//	    Index:        "index.html",        // Fallback file for SPA routing
//	    Prefix:       "/app",              // URL prefix for SPA routes
//	    IgnorePaths:  []string{"/api"},    // Paths to skip (pass through to handlers)
//	    MaxAge:       31536000,            // Cache duration for static assets (seconds)
//	    IndexMaxAge:  0,                   // Cache duration for index file (0 = no cache)
//	}))
//
// # API Route Integration
//
// Define API routes before the SPA middleware to ensure they're handled correctly:
//
//	app := mizu.New()
//
//	// API routes first
//	api := app.Group("/api")
//	api.Get("/users", listUsers)
//	api.Post("/users", createUser)
//
//	// SPA middleware last
//	app.Use(spa.WithOptions(spa.Options{
//	    Root:        "./dist",
//	    IgnorePaths: []string{"/api"},
//	}))
//
// # How It Works
//
// When a request comes in:
//
//  1. Checks if the path matches any IgnorePaths (e.g., /api) - if so, passes through
//  2. Strips the configured Prefix if present
//  3. Checks if the requested file exists in the filesystem
//  4. If file exists: serves it with MaxAge cache headers
//  5. If not: serves the index file with IndexMaxAge cache headers
//  6. Client-side router handles the routing
//
// # Cache Control Strategy
//
// The middleware implements a dual cache control strategy optimized for SPAs:
//
// Static Assets (MaxAge):
//   - Recommended: Set to a long duration (e.g., 31536000 for 1 year)
//   - Use content hashing in filenames (e.g., app.a1b2c3.js) for cache busting
//   - Ensures optimal performance for unchanging assets
//
// Index File (IndexMaxAge):
//   - Default: 0 (no cache) - ensures users always get latest routing config
//   - Sets "Cache-Control: no-cache, no-store, must-revalidate" when 0
//   - Can be increased if SPA routing config rarely changes
//
// # Security
//
// The middleware includes several security measures:
//
//   - All file paths are cleaned using filepath.Clean() to prevent directory traversal
//   - Only serves files from the configured root directory or filesystem
//   - Directory listings are disabled (directories trigger index fallback)
//   - Uses http.ServeContent for secure file delivery
//
// # File Serving
//
// The middleware uses http.ServeContent() for optimal file delivery:
//
//   - Automatic MIME type detection based on file extension
//   - HTTP range request support for partial content (e.g., video streaming)
//   - Proper Last-Modified header handling
//   - Efficient byte streaming with io.ReadSeeker interface
//
// # Common Patterns
//
// Skip multiple path prefixes:
//
//	app.Use(spa.WithOptions(spa.Options{
//	    Root:        "./dist",
//	    IgnorePaths: []string{"/api", "/health", "/metrics"},
//	}))
//
// Custom index file:
//
//	app.Use(spa.WithOptions(spa.Options{
//	    Root:  "./public",
//	    Index: "app.html",
//	}))
//
// Serve SPA at a specific path:
//
//	app.Use(spa.WithOptions(spa.Options{
//	    Root:   "./dist",
//	    Prefix: "/app",
//	}))
//	// SPA available at /app/*
//
// Production-ready configuration with optimal caching:
//
//	app.Use(spa.WithOptions(spa.Options{
//	    FS:          distFS,
//	    Root:        "dist",
//	    IgnorePaths: []string{"/api", "/health"},
//	    MaxAge:      31536000, // 1 year for static assets
//	    IndexMaxAge: 0,        // No cache for index.html
//	}))
package spa
