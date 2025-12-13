// Package canary provides canary deployment middleware for gradual traffic shifting.
//
// The canary middleware enables canary deployments by routing a percentage of traffic
// to different handlers, allowing gradual rollouts, A/B testing, and blue-green deployments.
//
// # Basic Usage
//
// Simple percentage-based canary routing:
//
//	app := mizu.New()
//	app.Use(canary.New(10)) // Route 10% to canary
//
//	app.Get("/", canary.Route(
//	    func(c *mizu.Ctx) error { return c.Text(200, "Canary") },
//	    func(c *mizu.Ctx) error { return c.Text(200, "Stable") },
//	))
//
// # Advanced Configuration
//
// Use WithOptions for full control over canary behavior:
//
//	app.Use(canary.WithOptions(canary.Options{
//	    Percentage: 10,
//	    Header:     "X-Canary",  // Override with header
//	    Cookie:     "canary",     // Check cookie for sticky sessions
//	    Selector:   customSelector, // Custom selection logic
//	}))
//
// # Header and Cookie Overrides
//
// Force canary routing via headers or cookies:
//
//	// Header: X-Canary: true
//	// Cookie: canary=true
//
// These take precedence over percentage-based selection.
//
// # Custom Selectors
//
// Implement custom selection logic:
//
//	app.Use(canary.WithOptions(canary.Options{
//	    Selector: func(c *mizu.Ctx) bool {
//	        // Route beta users to canary
//	        return c.Request().Header.Get("User-Type") == "beta"
//	    },
//	}))
//
// Built-in selectors:
//
//	canary.RandomSelector(10)              // 10% random selection
//	canary.HeaderSelector("X-Beta", "1")   // Based on header value
//	canary.CookieSelector("beta", "true")  // Based on cookie value
//
// # Routing Handlers
//
// Route to different handlers based on canary status:
//
//	app.Get("/api", canary.Route(canaryHandler, stableHandler))
//
// # Conditional Middleware
//
// Apply different middleware based on canary status:
//
//	app.Use(canary.Middleware(canaryMw, stableMw))
//
// # Release Manager
//
// Manage multiple canary releases:
//
//	manager := canary.NewReleaseManager()
//	manager.Set("feature-x", 5)  // 5% for feature-x
//	manager.Set("feature-y", 20) // 20% for feature-y
//
//	if manager.ShouldUseCanary("feature-x") {
//	    // Use canary version of feature-x
//	}
//
// # Implementation Details
//
// Traffic Distribution:
//   - Uses atomic counter-based distribution for predictable percentages
//   - Counter increments on each request: counter % 100 < percentage
//   - More deterministic than random selection
//   - No sticky sessions by default (stateless)
//
// Selection Precedence:
//  1. Header override (if Header option set)
//  2. Cookie override (if Cookie option set)
//  3. Custom selector (if Selector option set)
//  4. Percentage-based selection
//
// Context Storage:
//   - Canary decision stored in request context
//   - Retrieved via IsCanary(c) function
//   - Available throughout request lifecycle
//
// # Security Considerations
//
// The middleware uses math/rand for RandomSelector (not crypto/rand):
//   - Canary selection is not security-critical
//   - Performance-optimized for high-traffic scenarios
//   - Default counter-based approach is fully deterministic
//
// # Best Practices
//
//   - Start with low percentages (1-5%) and increase gradually
//   - Monitor error rates and performance metrics for canary traffic
//   - Use header/cookie overrides for internal testing before rollout
//   - Implement proper logging to track canary vs stable traffic
//   - Consider using sticky sessions for stateful applications
//   - Have rollback plan ready (reduce percentage to 0)
//
// # Examples
//
// Gradual rollout with dynamic percentage:
//
//	percent := atomic.Int32{}
//	percent.Store(1)
//
//	app.Use(canary.WithOptions(canary.Options{
//	    Selector: func(c *mizu.Ctx) bool {
//	        return rand.Intn(100) < int(percent.Load())
//	    },
//	}))
//
//	// Increase via admin endpoint
//	app.Post("/admin/canary/increase", func(c *mizu.Ctx) error {
//	    current := percent.Load()
//	    if current < 100 {
//	        percent.Store(current + 5)
//	    }
//	    return c.JSON(200, map[string]int{"percentage": int(percent.Load())})
//	})
//
// User-based canary with sticky sessions:
//
//	app.Use(canary.WithOptions(canary.Options{
//	    Percentage: 10,
//	    Cookie:     "canary_user",
//	}))
//
//	app.Use(func(next mizu.Handler) mizu.Handler {
//	    return func(c *mizu.Ctx) error {
//	        if canary.IsCanary(c) {
//	            c.SetCookie(&http.Cookie{
//	                Name:   "canary_user",
//	                Value:  "true",
//	                MaxAge: 86400, // 24 hours
//	            })
//	        }
//	        return next(c)
//	    }
//	})
//
// Feature-specific canary routing:
//
//	manager := canary.NewReleaseManager()
//	manager.Set("new-checkout", 10)
//	manager.Set("new-search", 25)
//
//	app.Get("/checkout", func(c *mizu.Ctx) error {
//	    if manager.ShouldUseCanary("new-checkout") {
//	        return newCheckoutHandler(c)
//	    }
//	    return oldCheckoutHandler(c)
//	})
//
//	app.Get("/search", func(c *mizu.Ctx) error {
//	    if manager.ShouldUseCanary("new-search") {
//	        return newSearchHandler(c)
//	    }
//	    return oldSearchHandler(c)
//	})
package canary
