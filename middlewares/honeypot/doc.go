// Package honeypot provides middleware for detecting and blocking malicious requests
// in Mizu applications.
//
// The honeypot middleware acts as a security trap that monitors requests to commonly
// attacked paths (such as /admin, /.env, /wp-admin) and blocks IPs that access them.
// This helps protect applications from automated vulnerability scanners and bot attacks.
//
// # Basic Usage
//
// Use the default honeypot configuration to monitor common attack paths:
//
//	app := mizu.New()
//	app.Use(honeypot.New())
//
// # Custom Paths
//
// Configure specific paths to monitor:
//
//	app.Use(honeypot.Paths("/secret", "/internal", "/debug"))
//
// # Preset Path Groups
//
// Use predefined path groups for common attack vectors:
//
//	// Monitor admin-related paths
//	app.Use(honeypot.AdminPaths())
//
//	// Monitor config file paths
//	app.Use(honeypot.ConfigPaths())
//
//	// Monitor database paths
//	app.Use(honeypot.DatabasePaths())
//
// # Custom Configuration
//
// Configure block duration, callbacks, and custom responses:
//
//	app.Use(honeypot.WithOptions(honeypot.Options{
//	    BlockDuration: 24 * time.Hour,
//	    OnTrap: func(ip, path string) {
//	        log.Printf("HONEYPOT: IP %s triggered trap at %s", ip, path)
//	    },
//	    Response: func(c *mizu.Ctx) error {
//	        time.Sleep(5 * time.Second) // Slow down attackers
//	        return c.Text(404, "Not Found")
//	    },
//	}))
//
// # Form Field Honeypot
//
// Detect bots that automatically fill hidden form fields:
//
//	app.Post("/contact", contactHandler, honeypot.Form("website"))
//
// The corresponding HTML form should include a hidden field:
//
//	<form method="POST" action="/contact">
//	    <input type="text" name="website" style="display:none">
//	    <input type="text" name="email">
//	    <textarea name="message"></textarea>
//	    <button type="submit">Send</button>
//	</form>
//
// # How It Works
//
// 1. Request arrives at a monitored honeypot path
// 2. Client IP is extracted (supports X-Forwarded-For and X-Real-IP headers)
// 3. IP is added to an in-memory block list with expiration
// 4. OnTrap callback is triggered (if configured) for logging/alerting
// 5. Custom or default response (404 Not Found) is returned
// 6. Future requests from the blocked IP receive 403 Forbidden
//
// # Thread Safety
//
// The block list uses sync.RWMutex for concurrent access and includes automatic
// cleanup of expired entries every 10 minutes.
//
// # Security Considerations
//
// - Choose honeypot paths that legitimate users won't access
// - Be aware of potential false positives
// - Attackers may rotate IPs to bypass blocking
// - Consider using X-Forwarded-For for accurate IP detection behind proxies
// - Monitor and analyze trapped IPs for attack patterns
// - Combine with other security measures like rate limiting and IP filtering
//
// # Performance
//
// - Path lookup: O(1) using hash map
// - Concurrent access: Thread-safe with RWMutex
// - Memory usage: ~24-32 bytes per blocked IP
// - Automatic cleanup: Runs every 10 minutes
// - Recommended for moderate traffic; consider external storage for high-traffic scenarios
package honeypot
