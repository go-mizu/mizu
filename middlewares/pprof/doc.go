// Package pprof provides profiling endpoints middleware for performance analysis.
//
// The pprof middleware integrates Go's built-in net/http/pprof package with Mizu,
// exposing standard profiling endpoints for debugging and performance optimization.
//
// # Quick Start
//
// Basic usage with default settings:
//
//	app := mizu.New()
//	app.Use(pprof.New())
//	// Access at http://localhost:8080/debug/pprof/
//
// # Configuration
//
// Customize the URL prefix:
//
//	app.Use(pprof.WithOptions(pprof.Options{
//	    Prefix: "/internal/profiling",
//	}))
//	// Access at http://localhost:8080/internal/profiling/
//
// # Available Endpoints
//
// The middleware exposes the following profiling endpoints:
//
//   - /debug/pprof/         - Index page with all profiles
//   - /debug/pprof/cmdline  - Command line arguments
//   - /debug/pprof/profile  - CPU profile (30s default)
//   - /debug/pprof/symbol   - Symbol lookup
//   - /debug/pprof/trace    - Execution trace
//   - /debug/pprof/heap     - Heap memory profile
//   - /debug/pprof/goroutine - Goroutine stack traces
//   - /debug/pprof/block    - Blocking profile
//   - /debug/pprof/mutex    - Mutex contention profile
//   - /debug/pprof/allocs   - Memory allocations
//   - /debug/pprof/threadcreate - Thread creation profile
//
// # Using the Profiles
//
// CPU profile (30 seconds):
//
//	go tool pprof http://localhost:8080/debug/pprof/profile
//
// Heap profile:
//
//	go tool pprof http://localhost:8080/debug/pprof/heap
//
// Goroutine dump:
//
//	curl http://localhost:8080/debug/pprof/goroutine?debug=2
//
// # Security Considerations
//
// IMPORTANT: Pprof endpoints expose sensitive application internals including:
//   - Memory contents and allocation patterns
//   - Active goroutines and their stack traces
//   - CPU usage patterns
//   - Application command line arguments
//
// In production environments, always:
//   - Use authentication (e.g., basicauth middleware)
//   - Restrict by IP address (e.g., ipfilter middleware)
//   - Use a separate admin port
//   - Or disable entirely
//
// Protected setup example:
//
//	// Only enable with authentication
//	admin := app.Group("/admin")
//	admin.Use(basicauth.New(basicauth.Options{
//	    Users: map[string]string{"admin": "secret"},
//	}))
//	admin.Use(pprof.WithOptions(pprof.Options{
//	    Prefix: "/admin/pprof",
//	}))
//
// # Implementation Details
//
// The middleware uses path-based routing to delegate requests to the appropriate
// pprof handlers from the standard library. It supports:
//
//   - Configurable URL prefix (default: "/debug/pprof")
//   - Automatic prefix normalization (removes trailing slashes)
//   - Pass-through for non-matching paths
//   - Direct delegation to net/http/pprof handlers
//
// The routing logic:
//  1. Checks if request path matches the configured prefix
//  2. Extracts the subpath after the prefix
//  3. Routes to appropriate handler (Index, Cmdline, Profile, Symbol, Trace, or named profiles)
//  4. Passes non-matching requests to the next middleware
//
// # Best Practices
//
//   - Never expose pprof publicly in production
//   - Use authentication for protected access
//   - Enable only when debugging is needed
//   - Use execution traces sparingly (high overhead)
//   - Keep CPU profiles short (30-60 seconds)
//   - Combine with IP filtering for additional security
//
// # Example: Conditional Pprof
//
// Enable only in development:
//
//	if os.Getenv("ENV") == "development" {
//	    app.Use(pprof.New())
//	}
//
// # Example: IP Restricted
//
// Allow only from localhost:
//
//	app.Use(ipfilter.WithOptions(ipfilter.Options{
//	    Whitelist: []string{"127.0.0.1", "::1"},
//	    PathPrefix: "/debug/pprof",
//	}))
//	app.Use(pprof.New())
package pprof
