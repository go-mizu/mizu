// Package timing provides Server-Timing header middleware for performance monitoring.
//
// The timing middleware adds Server-Timing headers to HTTP responses, enabling
// server-side performance measurement and reporting. These metrics are visible
// in browser DevTools, making it easy to identify performance bottlenecks.
//
// # Basic Usage
//
// Add the middleware to automatically track total request duration:
//
//	app := mizu.New()
//	app.Use(timing.New())
//
//	app.Get("/", func(c *mizu.Ctx) error {
//	    return c.JSON(200, data)
//	})
//	// Response: Server-Timing: total;dur=12.34
//
// # Custom Metrics
//
// Track specific operations using the Add function:
//
//	app.Get("/users", func(c *mizu.Ctx) error {
//	    start := time.Now()
//	    users, err := db.GetUsers()
//	    timing.Add(c, "db", time.Since(start), "Database query")
//
//	    return c.JSON(200, users)
//	})
//	// Response: Server-Timing: total;dur=45.67, db;dur=30.12;desc="Database query"
//
// # Start/Stop Pattern
//
// Use Start for cleaner timing control:
//
//	app.Get("/data", func(c *mizu.Ctx) error {
//	    stop := timing.Start(c, "external_api")
//	    resp, err := http.Get("https://api.example.com/data")
//	    if err != nil {
//	        return err
//	    }
//	    stop("External API call")
//
//	    return c.JSON(200, resp.Body)
//	})
//
// # Track Helper
//
// Track function execution with the Track helper:
//
//	app.Get("/compute", func(c *mizu.Ctx) error {
//	    var result int
//	    timing.Track(c, "calculation", func() {
//	        result = expensiveCalculation()
//	    })
//
//	    return c.JSON(200, map[string]int{"result": result})
//	})
//
// # Multiple Metrics
//
// Track multiple operations in a single request:
//
//	app.Get("/dashboard", func(c *mizu.Ctx) error {
//	    stopAuth := timing.Start(c, "auth")
//	    user := authenticateUser(c)
//	    stopAuth("Authentication")
//
//	    stopData := timing.Start(c, "data")
//	    dashboard := loadDashboard(user.ID)
//	    stopData("Load dashboard data")
//
//	    return c.JSON(200, dashboard)
//	})
//	// Response: Server-Timing: total;dur=120.50, auth;dur=15.20;desc="Authentication", data;dur=80.30;desc="Load dashboard data"
//
// # Implementation Details
//
// The middleware uses a context-based storage mechanism with the following
// characteristics:
//
//   - Thread-safe metric storage using sync.Mutex
//   - Automatic total duration tracking from request start
//   - Metrics converted to milliseconds with 2 decimal precision
//   - Server-Timing header format: name;dur=<ms>;desc="<description>"
//   - Safe for concurrent metric recording across goroutines
//
// # Browser Integration
//
// Server-Timing metrics appear in browser DevTools:
//
//   - Open DevTools → Network tab
//   - Select a request
//   - View Timing tab for metric breakdown
//
// # Best Practices
//
//   - Track database queries, cache lookups, and external API calls
//   - Use descriptive metric names (short and consistent)
//   - Add descriptions for clarity
//   - Prefer Start/Stop pattern for cleaner code
//   - Avoid tracking trivial operations (< 1ms)
//
// # Thread Safety
//
// All functions are safe for concurrent use. The timingData structure
// uses mutex locks to protect metric operations:
//
//   - Add() locks before appending metrics
//   - Header generation locks before reading metrics
//   - Multiple goroutines can safely record metrics
package timing
