// Package feature provides feature flag middleware for controlled feature rollouts,
// A/B testing, and gradual deployment in Mizu applications.
//
// # Overview
//
// The feature middleware enables runtime feature toggling using a provider-based
// architecture. It supports static flags, in-memory providers, and custom providers
// for integration with feature flag services or databases.
//
// # Basic Usage
//
// Create middleware with static feature flags:
//
//	app := mizu.New()
//	app.Use(feature.New(feature.Flags{
//	    "dark_mode":    &feature.Flag{Name: "dark_mode", Enabled: true},
//	    "new_checkout": &feature.Flag{Name: "new_checkout", Enabled: false},
//	}))
//
//	app.Get("/", func(c *mizu.Ctx) error {
//	    if feature.IsEnabled(c, "dark_mode") {
//	        return c.JSON(200, darkModeResponse)
//	    }
//	    return c.JSON(200, normalResponse)
//	})
//
// # Dynamic Flags with Memory Provider
//
// Use the memory provider for runtime flag updates:
//
//	provider := feature.NewMemoryProvider()
//	app.Use(feature.WithProvider(provider))
//
//	// Toggle flags at runtime
//	provider.Enable("beta_feature")
//	provider.Disable("deprecated_feature")
//	provider.Toggle("experimental")
//
// # Route Protection
//
// Protect routes requiring specific feature flags:
//
//	// Single flag required
//	app.Get("/beta",
//	    feature.Require("beta_access", nil),
//	    betaHandler,
//	)
//
//	// All flags required
//	app.Get("/admin",
//	    feature.RequireAll([]string{"admin", "super_user"}, nil),
//	    adminHandler,
//	)
//
//	// Any flag required
//	app.Get("/premium",
//	    feature.RequireAny([]string{"premium", "trial"}, nil),
//	    premiumHandler,
//	)
//
// # Custom Providers
//
// Implement the Provider interface for custom flag sources:
//
//	type DatabaseProvider struct {
//	    db *sql.DB
//	}
//
//	func (p *DatabaseProvider) GetFlags(c *mizu.Ctx) (feature.Flags, error) {
//	    // Load flags from database
//	    rows, err := p.db.Query("SELECT name, enabled FROM feature_flags")
//	    if err != nil {
//	        return nil, err
//	    }
//	    defer rows.Close()
//
//	    flags := make(feature.Flags)
//	    for rows.Next() {
//	        var name string
//	        var enabled bool
//	        rows.Scan(&name, &enabled)
//	        flags[name] = &feature.Flag{Name: name, Enabled: enabled}
//	    }
//	    return flags, nil
//	}
//
//	app.Use(feature.WithProvider(&DatabaseProvider{db: db}))
//
// # Flag Metadata
//
// Flags support metadata for analytics and rollout tracking:
//
//	provider.SetFlag(&feature.Flag{
//	    Name:        "new_checkout",
//	    Enabled:     true,
//	    Description: "New checkout flow",
//	    Metadata: map[string]any{
//	        "rollout_percentage": 50,
//	        "target_users":       "premium",
//	        "experiment_id":      "exp-123",
//	    },
//	})
//
// # Architecture
//
// The middleware uses a provider-based architecture:
//
//   - Provider Interface: Defines GetFlags(c *mizu.Ctx) (Flags, error)
//   - Context Storage: Flags stored in request context for handler access
//   - Thread Safety: MemoryProvider uses sync.RWMutex for concurrent access
//   - Immutable Returns: Providers return copies to prevent external modification
//
// # Performance
//
//   - Static Provider: O(n) copy per request
//   - Memory Provider: O(n) copy with RLock, writes use Lock
//   - Flag Lookup: O(1) map lookup from context
//   - Minimal memory overhead with single context value
//
// # Best Practices
//
//   - Use descriptive flag names
//   - Document flag purposes with Description field
//   - Clean up old flags after feature rollout completes
//   - Use percentage rollouts for risky features
//   - Implement custom provider for production environments
//   - Include metadata for analytics and tracking
//
// # Error Handling
//
// If a provider returns an error, the middleware continues with an empty flag map,
// ensuring application availability even when the flag provider is unavailable.
package feature
