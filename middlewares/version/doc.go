// Package version provides API versioning middleware for the Mizu web framework.
//
// The version middleware enables API versioning through multiple detection methods:
// HTTP headers, query parameters, or URL path prefixes. It supports version validation,
// deprecation warnings, and custom error handling.
//
// # Basic Usage
//
// The simplest usage with a default version:
//
//	app := mizu.New()
//	app.Use(version.New(version.Options{
//	    DefaultVersion: "v1",
//	}))
//
//	app.Get("/users", func(c *mizu.Ctx) error {
//	    v := version.Get(c)
//	    // Handle based on version
//	    return c.JSON(200, users)
//	})
//
// # Version Detection
//
// The middleware detects versions from multiple sources in priority order:
//
//  1. HTTP Header (default: "Accept-Version")
//  2. Query Parameter (default: "version")
//  3. URL Path Prefix (e.g., /v1/users, /v2/users)
//  4. Default Version (fallback)
//
// Example with header-based versioning:
//
//	app.Use(version.New(version.Options{
//	    DefaultVersion: "v1",
//	    Header:         "Accept-Version",
//	}))
//
//	// Client request:
//	// GET /api/users HTTP/1.1
//	// Accept-Version: v2
//
// Example with path-based versioning:
//
//	app.Use(version.New(version.Options{
//	    PathPrefix: true,
//	}))
//
//	// GET /v1/users → version "v1"
//	// GET /v2/users → version "v2"
//
// Example with query parameter:
//
//	app.Use(version.New(version.Options{
//	    DefaultVersion: "v1",
//	    QueryParam:     "api_version",
//	}))
//
//	// GET /users?api_version=v2
//
// # Version Validation
//
// The middleware supports allowlisting specific versions:
//
//	app.Use(version.New(version.Options{
//	    DefaultVersion: "v2",
//	    Supported:      []string{"v1", "v2", "v3"},
//	    ErrorHandler: func(c *mizu.Ctx, v string) error {
//	        return c.JSON(400, map[string]string{
//	            "error": "Unsupported API version: " + v,
//	        })
//	    },
//	}))
//
// Unsupported versions will trigger the error handler or return a default 400 response.
//
// # Deprecation Support
//
// Mark versions as deprecated to automatically add deprecation headers:
//
//	app.Use(version.New(version.Options{
//	    DefaultVersion: "v3",
//	    Supported:      []string{"v1", "v2", "v3"},
//	    Deprecated:     []string{"v1"},
//	}))
//
// Requests with deprecated versions will include these response headers:
//
//	Deprecation: true
//	Sunset: See documentation for migration guide
//
// # Helper Functions
//
// The package provides convenience functions for common use cases:
//
//	// Version from header only
//	app.Use(version.FromHeader("X-API-Version"))
//
//	// Version from path only
//	app.Use(version.FromPath())
//
//	// Version from query only
//	app.Use(version.FromQuery("v"))
//
// # Retrieving Version
//
// Use GetVersion or Get to retrieve the current request version:
//
//	app.Get("/users", func(c *mizu.Ctx) error {
//	    v := version.Get(c)
//
//	    switch v {
//	    case "v2":
//	        return c.JSON(200, usersV2)
//	    default:
//	        return c.JSON(200, usersV1)
//	    }
//	})
//
// Both GetVersion and Get are functionally identical.
//
// # Version String Format
//
// Valid version strings must match the pattern:
//   - Start with 'v' or 'V'
//   - Followed by digits (0-9)
//   - Optional dots (.) separating version components
//
// Valid examples: v1, v2, V1, v1.0, v1.2.3
// Invalid examples: api, empty string, v, va
//
// # Configuration Options
//
// The Options struct supports the following configuration:
//
//   - DefaultVersion: Version to use when not specified in request
//   - Header: HTTP header name for version (default: "Accept-Version")
//   - QueryParam: Query parameter name (default: "version")
//   - PathPrefix: Enable extraction from URL path prefix
//   - Supported: List of allowed versions (empty allows all)
//   - Deprecated: List of deprecated versions (triggers warning headers)
//   - ErrorHandler: Custom handler for unsupported versions
//
// # Implementation Details
//
// The middleware uses a context-based approach to store version information:
//   - Version is stored in request context using a private context key
//   - Supported/deprecated versions are stored in maps for O(1) lookup
//   - Version string validation uses character-by-character checks (no regex)
//   - Single pass through version sources with early exit on first match
//
// # Thread Safety
//
// The middleware is safe for concurrent use. Version information is stored
// per-request in the context and does not share state between requests.
package version
