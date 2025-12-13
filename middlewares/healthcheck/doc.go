// Package healthcheck provides health check endpoint middleware for Mizu applications.
//
// This package implements liveness and readiness probe endpoints commonly used in
// container orchestration platforms like Kubernetes. It supports custom health checks
// for databases, caches, and other external dependencies.
//
// # Overview
//
// The healthcheck middleware provides two types of endpoints:
//
//   - Liveness probes: Simple endpoints that indicate the application is running
//   - Readiness probes: Endpoints that verify the application and its dependencies are ready to serve traffic
//
// # Quick Start
//
// Basic usage with default endpoints:
//
//	app := mizu.New()
//	healthcheck.Register(app, healthcheck.Options{})
//	// Creates GET /healthz (liveness) and GET /readyz (readiness)
//
// With custom health checks:
//
//	healthcheck.Register(app, healthcheck.Options{
//	    Checks: []healthcheck.Check{
//	        healthcheck.DBCheck("postgres", db.Ping),
//	    },
//	})
//
// # Health Checks
//
// Health checks are executed concurrently with configurable timeouts.
// Each check must implement the Check interface:
//
//	type Check struct {
//	    Name    string                           // Check name for reporting
//	    Check   func(ctx context.Context) error  // Check function
//	    Timeout time.Duration                    // Check timeout (default: 5s)
//	}
//
// Example custom check:
//
//	check := healthcheck.Check{
//	    Name: "redis",
//	    Check: func(ctx context.Context) error {
//	        return redisClient.Ping(ctx).Err()
//	    },
//	    Timeout: 2 * time.Second,
//	}
//
// # Helper Functions
//
// The package provides helper functions for common check types:
//
//   - DBCheck: Creates a database ping check with 5s timeout
//   - HTTPCheck: Creates an HTTP endpoint check with 10s timeout
//
// Example:
//
//	dbCheck := healthcheck.DBCheck("postgres", db.PingContext)
//	httpCheck := healthcheck.HTTPCheck("api", "https://api.example.com/health")
//
// # Response Format
//
// Liveness endpoints return plain text:
//
//	ok
//
// Readiness endpoints return JSON with check results:
//
//	{
//	    "status": "ok",
//	    "checks": {
//	        "postgres": "ok",
//	        "redis": "ok"
//	    }
//	}
//
// On failure, the status is "error" and check values contain error messages:
//
//	{
//	    "status": "error",
//	    "checks": {
//	        "postgres": "ok",
//	        "redis": "connection refused"
//	    }
//	}
//
// # HTTP Status Codes
//
//   - 200: All checks passed (healthy)
//   - 503: One or more checks failed (unhealthy)
//
// # Kubernetes Integration
//
// Example Kubernetes probe configuration:
//
//	livenessProbe:
//	  httpGet:
//	    path: /healthz
//	    port: 8080
//	  initialDelaySeconds: 5
//	  periodSeconds: 10
//
//	readinessProbe:
//	  httpGet:
//	    path: /readyz
//	    port: 8080
//	  initialDelaySeconds: 5
//	  periodSeconds: 5
//
// # Implementation Details
//
// The middleware uses the following design patterns:
//
//   - Concurrent execution: All checks run in parallel using goroutines
//   - Thread-safe: Results are protected by sync.Mutex
//   - Context-based: Each check runs with context.WithTimeout
//   - Non-blocking: WaitGroup ensures all checks complete before response
//
// # Best Practices
//
//   - Keep liveness checks simple (no external dependencies)
//   - Include all critical dependencies in readiness checks
//   - Set appropriate timeouts for each check type
//   - Use readiness probes to control traffic during startup/shutdown
//   - Monitor check execution time to detect slow dependencies
//
// # Performance Considerations
//
// All checks execute concurrently, so total execution time is determined by
// the slowest check, not the sum of all checks. Configure timeouts appropriately
// to prevent long-running checks from delaying responses.
package healthcheck
