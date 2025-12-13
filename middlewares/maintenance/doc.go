// Package maintenance provides maintenance mode middleware for Mizu applications.
//
// This middleware enables maintenance mode functionality, allowing you to temporarily
// block access to your application while performing maintenance operations. It supports
// various configuration options including IP whitelisting, path whitelisting, custom
// handlers, and dynamic runtime control.
//
// # Basic Usage
//
// Enable maintenance mode for all requests:
//
//	app := mizu.New()
//	app.Use(maintenance.New(true))
//
// # Custom Configuration
//
// Configure maintenance mode with custom options:
//
//	app.Use(maintenance.WithOptions(maintenance.Options{
//	    Enabled:    true,
//	    Message:    "We're upgrading our systems. Back soon!",
//	    RetryAfter: 1800, // 30 minutes
//	}))
//
// # IP Whitelisting
//
// Allow specific IPs to bypass maintenance mode:
//
//	app.Use(maintenance.WithOptions(maintenance.Options{
//	    Enabled: true,
//	    Whitelist: []string{
//	        "192.168.1.100",    // Admin workstation
//	        "10.0.0.0/8",       // Internal network
//	    },
//	}))
//
// # Path Whitelisting
//
// Allow specific paths to bypass maintenance mode:
//
//	app.Use(maintenance.WithOptions(maintenance.Options{
//	    Enabled: true,
//	    WhitelistPaths: []string{
//	        "/health",          // Health checks
//	        "/status",          // Status endpoint
//	        "/api/webhooks",    // Critical webhooks
//	    },
//	}))
//
// # Dynamic Control
//
// Control maintenance mode at runtime:
//
//	mode := maintenance.NewMode(maintenance.Options{
//	    Message: "Maintenance in progress",
//	})
//
//	app.Use(mode.Middleware())
//
//	// Enable/disable dynamically
//	app.Post("/admin/maintenance/enable", func(c *mizu.Ctx) error {
//	    mode.Enable()
//	    return c.Text(200, "Maintenance enabled")
//	})
//
//	app.Post("/admin/maintenance/disable", func(c *mizu.Ctx) error {
//	    mode.Disable()
//	    return c.Text(200, "Maintenance disabled")
//	})
//
//	app.Get("/admin/maintenance/status", func(c *mizu.Ctx) error {
//	    return c.JSON(200, map[string]bool{
//	        "enabled": mode.IsEnabled(),
//	    })
//	})
//
// # Scheduled Maintenance
//
// Schedule maintenance for a specific time period:
//
//	start := time.Date(2024, 12, 15, 2, 0, 0, 0, time.UTC)
//	end := time.Date(2024, 12, 15, 4, 0, 0, 0, time.UTC)
//
//	app.Use(maintenance.ScheduledMaintenance(start, end))
//
// # Custom Handler
//
// Use a custom handler for maintenance responses:
//
//	app.Use(maintenance.WithOptions(maintenance.Options{
//	    Enabled: true,
//	    Handler: func(c *mizu.Ctx) error {
//	        return c.HTML(503, `
//	            <html>
//	                <head><title>Maintenance</title></head>
//	                <body>
//	                    <h1>Under Maintenance</h1>
//	                    <p>We'll be back shortly.</p>
//	                </body>
//	            </html>
//	        `)
//	    },
//	}))
//
// # Dynamic Check Function
//
// Use a custom function to determine maintenance state:
//
//	app.Use(maintenance.WithOptions(maintenance.Options{
//	    Check: func() bool {
//	        // Check external flag (e.g., from database or config)
//	        return config.IsMaintenanceMode()
//	    },
//	}))
//
// # Configuration Options
//
// The Options struct supports the following fields:
//
//   - Enabled (bool): Enable maintenance mode. Default: false
//   - Message (string): Response message. Default: "Service is under maintenance"
//   - RetryAfter (int): Retry-After header in seconds. Default: 3600
//   - StatusCode (int): HTTP status code. Default: 503
//   - Handler (mizu.Handler): Custom maintenance handler
//   - Whitelist ([]string): Allowed IP addresses during maintenance
//   - WhitelistPaths ([]string): Paths that bypass maintenance
//   - Check (func() bool): Dynamic maintenance check function
//
// # Thread Safety
//
// The Mode type uses atomic operations for thread-safe state management,
// making it safe to enable/disable/toggle maintenance mode from multiple
// goroutines concurrently.
//
// # Response Headers
//
// When maintenance mode is active, the middleware sets the following headers:
//
//   - Status: 503 Service Unavailable (or custom StatusCode)
//   - Retry-After: Number of seconds (from RetryAfter option)
//   - Content-Type: text/plain (or as set by custom Handler)
//
// # IP Detection
//
// The middleware detects client IP addresses from the following sources (in order):
//
//  1. X-Forwarded-For header
//  2. X-Real-IP header
//  3. RemoteAddr from the request
//
// # Best Practices
//
//   - Whitelist health check endpoints to keep monitoring active
//   - Use scheduled maintenance for planned downtime windows
//   - Whitelist admin IPs for debugging during maintenance
//   - Set appropriate Retry-After values to inform clients when to retry
//   - Provide clear and helpful maintenance messages
//   - Use dynamic control for emergency maintenance scenarios
package maintenance
