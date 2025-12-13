// Package lastmodified provides middleware for HTTP Last-Modified header handling
// and conditional request processing according to RFC 7232.
//
// The middleware automatically handles Last-Modified headers and processes
// If-Modified-Since conditional requests to enable efficient HTTP caching.
// When a resource has not been modified since the time specified in the client's
// If-Modified-Since header, the middleware returns a 304 Not Modified response,
// saving bandwidth and processing time.
//
// # Basic Usage
//
// The simplest way to use the middleware is with a time function that returns
// the last modification time for the resource:
//
//	app := mizu.New()
//	app.Use(lastmodified.New(func(c *mizu.Ctx) time.Time {
//	    return getResourceModificationTime(c.Request().URL.Path)
//	}))
//
// # Static Time
//
// For resources that have a fixed modification time, use the Static helper:
//
//	deployTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
//	app.Use(lastmodified.Static(deployTime))
//
// # Startup Time
//
// For versioned assets that change with deployments, use StartupTime:
//
//	app.Use(lastmodified.StartupTime())
//
// # Advanced Configuration
//
// Use WithOptions for more control over the middleware behavior:
//
//	app.Use(lastmodified.WithOptions(lastmodified.Options{
//	    TimeFunc: func(c *mizu.Ctx) time.Time {
//	        return getModTime(c.Request().URL.Path)
//	    },
//	    SkipPaths: []string{"/api/health", "/metrics"},
//	}))
//
// # How It Works
//
// The middleware follows this process:
//
//  1. Skips non-GET/HEAD requests (Last-Modified only applies to safe methods)
//  2. Calls the configured TimeFunc to get the resource's modification time
//  3. Sets the Last-Modified header with the time in HTTP-date format
//  4. Checks for If-Modified-Since header in the request
//  5. If the resource hasn't been modified since that time, returns 304 Not Modified
//  6. Otherwise, continues to the next handler which sends the full response
//
// # Time Handling
//
// - All times are automatically converted to UTC before being set in headers
// - Comparison is done at second precision (subsecond values are truncated)
// - Zero time.Time values skip middleware processing entirely
// - Uses http.ParseTime for flexible date parsing from If-Modified-Since
//
// # HTTP Specification Compliance
//
// The middleware implements RFC 7232 (Conditional Requests) specification:
//
//   - Last-Modified header uses HTTP-date format (RFC 5322)
//   - 304 Not Modified response has no body
//   - Only processes GET and HEAD methods
//   - Proper time comparison using truncation to second precision
//
// # Best Practices
//
//   - Combine with ETag middleware for robust caching strategy
//   - Set accurate modification times based on actual resource changes
//   - Use for static or semi-static content where modification time is trackable
//   - Configure Cache-Control headers alongside Last-Modified for complete cache control
//   - Use SkipPaths to exclude dynamic endpoints that change on every request
//
// # Helper Functions
//
// The package provides several helper functions for common use cases:
//
//   - New(timeFunc): Create middleware with custom time function
//   - Static(t): Use a fixed modification time
//   - Now(): Use current time (useful for testing)
//   - StartupTime(): Use application startup time
//   - FromHeader(header): Read modification time from a custom request header
//   - WithOptions(opts): Full configuration with skip paths and custom behavior
package lastmodified
