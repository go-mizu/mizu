// Package prometheus provides Prometheus metrics middleware for the Mizu web framework.
//
// This middleware collects HTTP metrics and exposes them in Prometheus format, enabling
// integration with Prometheus monitoring systems, Grafana dashboards, and AlertManager.
//
// # Features
//
//   - Request counter with method, path, and status labels
//   - Request duration histogram with configurable buckets
//   - Request and response size histograms
//   - Active requests gauge
//   - Thread-safe metric collection using atomic operations
//   - Customizable metric names with namespace and subsystem support
//   - Path skipping for health checks and metrics endpoints
//
// # Basic Usage
//
// The simplest way to use the middleware is with default settings:
//
//	app := mizu.New()
//
//	// Collect metrics on all routes
//	app.Use(prometheus.New())
//
//	// Expose metrics endpoint
//	app.Get("/metrics", prometheus.Handler())
//
// # Custom Configuration
//
// You can customize the middleware behavior using the Options struct:
//
//	metrics := prometheus.NewMetrics(prometheus.Options{
//		Namespace: "myapp",
//		Subsystem: "api",
//		Buckets:   []float64{0.1, 0.5, 1.0, 5.0},
//		SkipPaths: []string{"/health", "/ping"},
//	})
//
//	app.Use(metrics.Middleware())
//	app.Get("/metrics", metrics.Handler())
//
// # Metrics Collected
//
// The middleware collects the following metrics:
//
//   - http_requests_total: Counter tracking total HTTP requests with method, path, and status labels
//   - http_request_duration_seconds: Histogram of request latencies in seconds
//   - http_request_size_bytes: Histogram of request sizes in bytes
//   - http_response_size_bytes: Histogram of response sizes in bytes
//   - http_requests_active: Gauge of currently active requests
//
// # Metric Naming
//
// Metric names can be prefixed with namespace and subsystem:
//
//   - With namespace "myapp" and subsystem "api": myapp_api_http_requests_total
//   - With namespace "myapp" only: myapp_http_requests_total
//   - With subsystem "api" only: api_http_requests_total
//   - Default: http_requests_total
//
// # Path Skipping
//
// By default, the metrics endpoint path is automatically skipped to avoid recording
// metric scraping requests. You can configure additional paths to skip:
//
//	metrics := prometheus.NewMetrics(prometheus.Options{
//		SkipPaths: []string{"/health", "/ready", "/live"},
//	})
//
// # Prometheus Configuration
//
// Configure Prometheus to scrape your application:
//
//	scrape_configs:
//	  - job_name: 'myapp'
//	    static_configs:
//	      - targets: ['localhost:8080']
//	    metrics_path: '/metrics'
//	    scrape_interval: 15s
//
// # Thread Safety
//
// All operations in this package are thread-safe:
//
//   - Metric recording uses sync.RWMutex for map access and sync/atomic for counters
//   - Histogram observations are protected by individual mutex locks
//   - Request counting uses atomic operations for optimal performance
//
// # Best Practices
//
//   - Use consistent naming conventions for namespaces and subsystems
//   - Avoid high cardinality labels (e.g., user IDs, timestamps)
//   - Skip health check and metrics endpoints to reduce noise
//   - Set appropriate Prometheus scrape intervals based on your needs
//   - Use custom buckets that match your SLA requirements
//
// # Example: Production Setup
//
//	metrics := prometheus.NewMetrics(prometheus.Options{
//		Namespace: "mycompany",
//		Subsystem: "userservice",
//		Buckets:   []float64{0.01, 0.05, 0.1, 0.5, 1.0, 2.5, 5.0},
//		SkipPaths: []string{"/health", "/metrics"},
//	})
//
//	app := mizu.New()
//	app.Use(metrics.Middleware())
//
//	// Register endpoints
//	metrics.RegisterEndpoint(app) // Registers at /metrics by default
//
//	app.Get("/api/users", handleUsers)
//	app.Get("/health", handleHealth)
//
// # Advanced Usage
//
// Access metric statistics programmatically:
//
//	totalReqs := metrics.TotalRequests()
//	activeReqs := metrics.ActiveRequests()
//
//	fmt.Printf("Total: %d, Active: %d\n", totalReqs, activeReqs)
//
// Export metrics in Prometheus text format:
//
//	output := metrics.Export()
//	fmt.Println(output)
package prometheus
