// Package static provides middleware for serving static files in Mizu applications.
//
// The static middleware enables serving files from either a local filesystem directory
// or an embedded fs.FS interface. It supports index files, directory browsing, cache
// control, URL prefixes, and custom 404 handlers.
//
// # Basic Usage
//
// Serve files from a directory:
//
//	app := mizu.New()
//	app.Use(static.New("./public"))
//
// # Embedded Filesystem
//
// Serve files from an embedded filesystem:
//
//	//go:embed static/*
//	var staticFS embed.FS
//
//	app.Use(static.WithFS(staticFS))
//
// # URL Prefix
//
// Serve files with a URL prefix:
//
//	app.Use(static.WithOptions(static.Options{
//	    Root:   "./public",
//	    Prefix: "/static",
//	}))
//	// Files available at /static/css/style.css, etc.
//
// # Cache Control
//
// Enable caching with Cache-Control headers:
//
//	app.Use(static.WithOptions(static.Options{
//	    Root:   "./public",
//	    MaxAge: 86400, // 1 day in seconds
//	}))
//
// # Directory Browsing
//
// Enable directory listing for folders without index files:
//
//	app.Use(static.WithOptions(static.Options{
//	    Root:   "./files",
//	    Browse: true,
//	}))
//
// # Custom Index File
//
// Use a different index file name:
//
//	app.Use(static.WithOptions(static.Options{
//	    Root:  "./public",
//	    Index: "default.htm",
//	}))
//
// # Custom 404 Handler
//
// Handle missing files with a custom handler:
//
//	app.Use(static.WithOptions(static.Options{
//	    Root: "./public",
//	    NotFoundHandler: func(c *mizu.Ctx) error {
//	        return c.Text(404, "File not found")
//	    },
//	}))
//
// # Implementation Details
//
// The middleware implements two serving strategies:
//
// 1. Direct File Serving (default): Uses http.ServeContent for individual files,
// providing proper Content-Type detection, range request support, and conditional
// request handling with If-Modified-Since headers.
//
// 2. FileServer Mode (when Browse is enabled): Uses http.FileServer to generate
// HTML directory listings automatically.
//
// Path resolution is handled safely with filepath.Clean to prevent directory
// traversal attacks. The middleware checks file existence before serving and
// falls through to the next handler if files are not found (unless a custom
// NotFoundHandler is configured).
//
// # Performance Considerations
//
// The middleware includes several optimizations:
//   - Custom itoa function to avoid allocations when setting Cache-Control headers
//   - Direct file serving without unnecessary FileServer overhead for individual files
//   - Minimal path manipulation and string operations
//
// # Security
//
// Best practices for production use:
//   - Disable directory browsing (Browse: false) in production
//   - Use appropriate cache settings for static assets
//   - Serve files from embedded filesystem for portable deployments
//   - Use versioned filenames for cache busting
package static
