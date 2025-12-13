// Package chaos provides chaos engineering middleware for testing application resilience.
//
// # Overview
//
// The chaos middleware allows you to inject controlled failures and latency into your
// application to test its resilience and error handling capabilities. This is useful
// for chaos engineering, integration testing, and validating fallback mechanisms.
//
// # Basic Usage
//
//	import "github.com/go-mizu/mizu/middlewares/chaos"
//
//	app := mizu.New()
//
//	// Inject 10% errors with random latency
//	app.Use(chaos.WithOptions(chaos.Options{
//	    Enabled:    true,
//	    ErrorRate:  10,
//	    LatencyMin: 100 * time.Millisecond,
//	    LatencyMax: 500 * time.Millisecond,
//	}))
//
// # Error Injection
//
// Inject HTTP errors with configurable rate and status code:
//
//	// Fail 20% of requests with 500 error
//	app.Use(chaos.Error(20, 500))
//
// # Latency Injection
//
// Add artificial latency to requests:
//
//	// Add 100-500ms latency to all requests
//	app.Use(chaos.Latency(100*time.Millisecond, 500*time.Millisecond))
//
// # Selective Chaos
//
// Use selectors to target specific requests:
//
//	// Only affect specific paths
//	app.Use(chaos.WithOptions(chaos.Options{
//	    Enabled:   true,
//	    ErrorRate: 50,
//	    Selector:  chaos.PathSelector("/api/orders", "/api/payments"),
//	}))
//
//	// Only affect write operations
//	app.Use(chaos.WithOptions(chaos.Options{
//	    Enabled:   true,
//	    ErrorRate: 10,
//	    Selector:  chaos.MethodSelector("POST", "PUT", "DELETE"),
//	}))
//
//	// Only affect requests with specific header
//	app.Use(chaos.WithOptions(chaos.Options{
//	    Enabled:   true,
//	    ErrorRate: 100,
//	    Selector:  chaos.HeaderSelector("X-Chaos-Test"),
//	}))
//
// # Dynamic Control
//
// Use Controller for runtime configuration changes:
//
//	controller := chaos.NewController()
//	app.Use(controller.Middleware())
//
//	// Control chaos via API
//	app.Post("/admin/chaos/enable", func(c *mizu.Ctx) error {
//	    controller.Enable()
//	    return c.Text(200, "Chaos enabled")
//	})
//
//	app.Post("/admin/chaos/disable", func(c *mizu.Ctx) error {
//	    controller.Disable()
//	    return c.Text(200, "Chaos disabled")
//	})
//
// # Implementation Details
//
// Random Number Generation:
//   - Uses math/rand for performance (intentionally weak RNG for chaos testing)
//   - Error injection uses percentage-based probability (0-100)
//   - Latency is calculated as: LatencyMin + rand.Int63n(LatencyMax - LatencyMin)
//
// Request Flow:
//  1. Check if chaos is enabled
//  2. Apply selector filter (if configured)
//  3. Inject latency (if configured) using time.Sleep
//  4. Inject error based on probability (if configured)
//  5. Pass to next handler (if no error injected)
//
// Selector Performance:
//   - PathSelector and MethodSelector use maps for O(1) lookup
//   - HeaderSelector uses standard library header access
//
// # Safety Considerations
//
// Always protect chaos control endpoints with authentication:
//
//	admin := app.Group("/admin")
//	admin.Use(basicauth.New(basicauth.Options{
//	    Users: map[string]string{"admin": "secret"},
//	}))
//	admin.Post("/chaos/enable", enableChaosHandler)
//
// Best practices:
//   - Never enable in production without safeguards
//   - Use selectors to limit scope of chaos injection
//   - Start with low error rates and increase gradually
//   - Monitor application metrics during chaos testing
//   - Use header-triggered chaos for CI/CD integration tests
//
// # Configuration Options
//
// Options struct fields:
//   - Enabled (bool): Enable chaos injection, default: false
//   - ErrorRate (int): Percentage of requests to fail (0-100), default: 0
//   - ErrorCode (int): HTTP status code for errors, default: 500
//   - LatencyMin (time.Duration): Minimum latency to inject, default: 0
//   - LatencyMax (time.Duration): Maximum latency to inject, default: 0
//   - Selector (func(*mizu.Ctx) bool): Filter which requests to affect, default: nil (all requests)
//
// # Controller Methods
//
//   - NewController() creates a new controller (disabled by default)
//   - Enable() enables chaos injection
//   - Disable() disables chaos injection
//   - IsEnabled() returns current enabled state
//   - SetErrorRate(rate int) sets error injection percentage
//   - SetErrorCode(code int) sets HTTP status code for errors
//   - SetLatency(min, max time.Duration) sets latency range
//   - SetSelector(func(*mizu.Ctx) bool) sets request filter
//   - Middleware() returns middleware using controller configuration
//
// # Built-in Selectors
//
//   - PathSelector(paths ...string) matches specific URL paths
//   - MethodSelector(methods ...string) matches HTTP methods
//   - HeaderSelector(header string) matches requests with specific header
package chaos
