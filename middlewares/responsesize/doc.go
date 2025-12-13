// Package responsesize provides middleware for tracking response sizes in Mizu applications.
//
// # Overview
//
// The responsesize middleware monitors the number of bytes written to HTTP responses,
// enabling bandwidth monitoring, metrics collection, and detection of response size anomalies.
//
// # Basic Usage
//
// Simple response size tracking:
//
//	app := mizu.New()
//	app.Use(responsesize.New())
//
// # With Callback
//
// Track response sizes with a custom callback:
//
//	app.Use(responsesize.WithCallback(func(c *mizu.Ctx, size int64) {
//	    metrics.RecordResponseSize(c.Request().URL.Path, size)
//	}))
//
// # Custom Options
//
// Configure with custom options:
//
//	app.Use(responsesize.WithOptions(responsesize.Options{
//	    OnSize: func(c *mizu.Ctx, size int64) {
//	        if size > 1024*1024 { // > 1MB
//	            log.Printf("Large response: %s %d bytes", c.Request().URL.Path, size)
//	        }
//	    },
//	}))
//
// # Retrieving Size Information
//
// Access response size during request handling:
//
//	app.Get("/status", func(c *mizu.Ctx) error {
//	    // Get size info from context
//	    info := responsesize.Get(c)
//	    currentSize := info.BytesWritten()
//
//	    // Or use the helper function
//	    size := responsesize.BytesWritten(c)
//
//	    return c.JSON(http.StatusOK, map[string]int64{"bytes": size})
//	})
//
// # Implementation Details
//
// The middleware uses a response writer wrapper (trackingWriter) that intercepts
// all Write() calls to count bytes. It uses atomic operations for thread-safe
// counting and stores size information in the request context for retrieval
// anywhere in the handler chain.
//
// # Thread Safety
//
// All byte counting operations use sync/atomic to ensure thread-safe access,
// making the middleware safe for use with concurrent writes.
package responsesize
