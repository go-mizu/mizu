// Package mirror provides request mirroring middleware for Mizu.
//
// The mirror middleware duplicates HTTP requests to one or more target servers
// for traffic shadowing, testing, and analysis. Mirrored requests run asynchronously
// by default and don't affect the original response.
//
// # Basic Usage
//
// Mirror all traffic to a single target:
//
//	app := mizu.New()
//	app.Use(mirror.New("https://staging.example.com"))
//
// # Multiple Targets
//
// Mirror to multiple targets simultaneously:
//
//	app.Use(mirror.New(
//	    "https://staging.example.com",
//	    "https://analytics.example.com",
//	))
//
// # Percentage-Based Mirroring
//
// Mirror a percentage of traffic using the Percentage helper:
//
//	app.Use(mirror.WithOptions(mirror.Options{
//	    Targets: []mirror.Target{
//	        mirror.Percentage("https://staging.example.com", 50), // 50% of traffic
//	        mirror.Percentage("https://canary.example.com", 10),  // 10% of traffic
//	    },
//	}))
//
// # Advanced Configuration
//
// Customize timeout, sync mode, and callbacks:
//
//	app.Use(mirror.WithOptions(mirror.Options{
//	    Targets: []mirror.Target{
//	        {URL: "https://staging.example.com", Percentage: 100},
//	    },
//	    Timeout: 10 * time.Second,
//	    Async:   true,
//	    OnError: func(target string, err error) {
//	        log.Printf("Mirror to %s failed: %v", target, err)
//	    },
//	    OnSuccess: func(target string, resp *http.Response) {
//	        log.Printf("Mirror to %s: status %d", target, resp.StatusCode)
//	    },
//	}))
//
// # Implementation Details
//
// The middleware implements traffic mirroring with these characteristics:
//
//   - Request Cloning: Reads and stores request body in memory when CopyBody is enabled
//   - Asynchronous Execution: Runs mirror requests in goroutines by default (configurable)
//   - Percentage Sampling: Uses counter-based modulo for traffic sampling
//   - Header Injection: Adds X-Mirrored-From header to identify mirrored requests
//   - Error Isolation: Mirror failures never affect the original request/response
//
// # Configuration Options
//
// Options provides fine-grained control over mirroring behavior:
//
//   - Targets: List of mirror targets with URLs and percentage thresholds
//   - Timeout: HTTP client timeout for mirrored requests (default: 5s)
//   - Async: Execute mirrors asynchronously to avoid latency (default: true)
//   - CopyBody: Copy request body for POST/PUT requests (default: true)
//   - OnError: Callback invoked when mirror request fails
//   - OnSuccess: Callback invoked when mirror request succeeds
//
// # Use Cases
//
//   - Shadow Testing: Test new versions with production traffic
//   - Load Testing: Send copies to test infrastructure
//   - Analytics: Duplicate to analytics services
//   - Canary Deployment: Route small percentage to canary servers
//   - Response Comparison: Validate new deployments against production
//
// # Performance Considerations
//
//   - Async mode (default) ensures zero latency impact on original requests
//   - Memory overhead is proportional to request body size when CopyBody is enabled
//   - Each mirrored request spawns one goroutine per target in async mode
//   - Sync mode adds mirror request latency to original request processing
//
// # Best Practices
//
//   - Use async mode (default) to avoid adding latency to production requests
//   - Set appropriate timeouts to prevent hanging mirror requests
//   - Monitor error rates on mirror targets using OnError callback
//   - Use percentage-based mirroring for gradual rollouts and canary testing
//   - Avoid mirroring to targets with side effects unless intentional
//   - Disable CopyBody for GET requests to reduce memory usage
package mirror
