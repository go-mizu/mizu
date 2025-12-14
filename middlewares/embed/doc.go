/*
Package embed provides middleware and handlers for serving files from Go embedded filesystems.

The embed package enables serving static files, templates, and assets from io/fs.FS
implementations such as embed.FS. This allows single-binary deployments without
external file dependencies while keeping behavior consistent with net/http.

# Basic usage

The simplest way to use the embed middleware is with New:

	//go:embed static/*
	var staticFS embed.FS

	app := mizu.NewRouter()
	app.Use(embed.New(staticFS))

This serves files from the root of the embedded filesystem. Requests fall through
to the next handler when no file is found.

# Serving from subdirectories

Use Static or WithOptions to serve from a specific subdirectory:

	//go:embed assets/*
	var assetsFS embed.FS

	app.Use(embed.Static(assetsFS, "assets"))
	// assets/css/app.css is available at /css/app.css

# Configuration options

For more control, use WithOptions and Options:

	app.Use(embed.WithOptions(fsys, embed.Options{
		Root:    "public",        // Subdirectory within fs
		Index:   "index.html",    // Index file name
		MaxAge:  3600,            // Cache-Control max-age in seconds
		NotFoundHandler: func(c *mizu.Ctx) error {
			return c.Text(404, "not found")
		},
	}))

# Caching

Enable caching using WithCaching:

	app.Use(embed.WithCaching(staticFS, 86400)) // 24 hours

This sets Cache-Control: max-age for served files.

# Single-page applications

SPA provides routing suitable for SPAs by serving the index file for unknown paths:

	//go:embed dist/*
	var distFS embed.FS

	app.Use(embed.SPA(distFS, "index.html"))

Examples:

	GET /assets/app.js     -> serves the actual file
	GET /app/dashboard    -> serves index.html

# Handler functions

In addition to middleware, the package provides handlers for direct routing:

	app.Get("/static/{path...}", embed.Handler(staticFS))

	app.Get("/assets/{path...}",
		embed.HandlerWithOptions(assetsFS, embed.Options{
			Root:   "assets",
			MaxAge: 3600,
		}),
	)

Handlers do not fall through. They always attempt to serve files directly.

# Implementation details

The implementation is based on io/fs.FS and http.FileServer.

Key behaviors:
  - Paths are normalized and cleaned before serving
  - Canonicalization prevents http.FileServer redirects
  - Directory requests automatically try the index file
  - Middleware falls through when files are missing
  - Handlers serve files directly without fallthrough

Performance considerations

  - Embedded files increase binary size
  - Keep assets reasonably sized
  - Custom itoa avoids strconv allocation for cache headers
  - http.FileServer handles ranges, content types, and conditional requests

Security notes

  - Path traversal is prevented using path.Clean
  - Only embedded files are accessible
  - Directory browsing is disabled by default
  - Browse option is reserved and not currently implemented

# Middleware vs handler

Middleware:
  - New
  - WithOptions
  - Static
  - WithCaching
  - SPA

Handlers:
  - Handler
  - HandlerWithOptions
*/
package embed
