// Package adaptive provides adaptive rate limiting middleware for the Mizu web framework.
//
// The adaptive middleware dynamically adjusts rate limits based on real-time system metrics
// such as latency, error rate, and resource utilization. Unlike static rate limiters, it
// automatically adapts to changing load conditions.
//
// # Overview
//
// Adaptive rate limiting uses a token bucket algorithm with dynamic rate adjustment.
// It continuously monitors request metrics and adjusts the rate limit to maintain
// target performance levels while maximizing throughput.
//
// # Basic Usage
//
// Create an adaptive rate limiter with custom options:
//
//	app := mizu.New()
//	app.Use(adaptive.New(adaptive.Options{
//	    InitialRate:    100,
//	    MinRate:        10,
//	    MaxRate:        1000,
//	    TargetLatency:  100 * time.Millisecond,
//	    ErrorThreshold: 0.1,
//	    AdjustInterval: time.Second,
//	}))
//
// Or use one of the preset configurations:
//
//	// Simple configuration with defaults
//	app.Use(adaptive.Simple())
//
//	// High throughput configuration
//	app.Use(adaptive.HighThroughput())
//
//	// Conservative configuration
//	app.Use(adaptive.Conservative())
//
// # How It Works
//
// The middleware operates in three phases:
//
// 1. Metrics Collection: Tracks request count, error count, and total latency
// atomically during request processing.
//
// 2. Rate Adjustment: Periodically (based on AdjustInterval) analyzes metrics
// and adjusts the current rate:
//   - High error rate (> ErrorThreshold): Reduce rate by 20%
//   - High latency (> TargetLatency): Reduce rate by 10%
//   - Good performance (low latency + low errors): Increase rate by 10%
//
// 3. Token Bucket Enforcement: Uses a token bucket algorithm to enforce the
// current rate limit, refilling tokens based on elapsed time.
//
// # Configuration Options
//
// InitialRate: The starting rate limit (requests per second). Default: 100.
//
// MinRate: The minimum rate limit. The limiter will never adjust below this value.
// Default: 10.
//
// MaxRate: The maximum rate limit. The limiter will never adjust above this value.
// Default: 1000.
//
// TargetLatency: The desired response latency. If average latency exceeds this,
// the rate will be reduced. Default: 100ms.
//
// ErrorThreshold: The acceptable error rate (0-1). If the error rate exceeds this,
// the rate will be reduced. Default: 0.1 (10%).
//
// AdjustInterval: How frequently to recalculate and adjust the rate limit.
// Default: 1s.
//
// KeyFunc: Optional function to extract the rate limit key from the context.
// Can be used for per-user or per-IP rate limiting. Default: nil (global limiter).
//
// ErrorHandler: Optional custom error handler called when requests are rate limited.
// Default: returns 429 Too Many Requests with "Retry-After: 1" header.
//
// # Advanced Usage
//
// Access the limiter instance for statistics and monitoring:
//
//	limiter := adaptive.NewLimiter(adaptive.Options{
//	    InitialRate: 100,
//	    MinRate:     10,
//	    MaxRate:     1000,
//	})
//
//	app.Use(limiter.Middleware())
//
//	// Later, get statistics
//	stats := limiter.Stats()
//	fmt.Printf("Current rate: %d\n", stats.CurrentRate)
//	fmt.Printf("Requests: %d, Errors: %d\n", stats.RequestCount, stats.ErrorCount)
//
// Custom error handling:
//
//	app.Use(adaptive.New(adaptive.Options{
//	    InitialRate: 100,
//	    ErrorHandler: func(c *mizu.Ctx) error {
//	        return c.JSON(http.StatusTooManyRequests, map[string]interface{}{
//	            "error":      "rate_limit_exceeded",
//	            "message":    "Please slow down your requests",
//	            "retryAfter": 1,
//	        })
//	    },
//	}))
//
// # Implementation Details
//
// The adaptive limiter uses atomic operations for all metric updates to ensure
// thread-safety without locks during the hot path (request processing). A single
// background goroutine performs the periodic rate adjustments, which use a mutex
// to safely read and reset metrics.
//
// Token refill uses a precise algorithm that accounts for fractional tokens:
//
//	elapsed := now - lastRefill
//	refill := (elapsed * currentRate) / 1_second
//
// This ensures accurate rate limiting even at high frequencies.
//
// # Best Practices
//
// 1. Start Conservative: Begin with conservative limits and let the system
// adjust upward. It's safer to increase capacity gradually than to overwhelm
// your system.
//
// 2. Set Realistic Targets: Choose TargetLatency based on your actual SLA
// requirements. Monitor your baseline latency under normal load.
//
// 3. Monitor Adjustments: Track how the rate changes over time to understand
// your system's capacity and identify performance issues.
//
// 4. Combine with Circuit Breaker: Use alongside a circuit breaker for
// comprehensive protection against cascading failures.
//
// 5. Tune AdjustInterval: Shorter intervals respond faster to load changes
// but may oscillate. Longer intervals are more stable but slower to adapt.
//
// # Thread Safety
//
// All operations are thread-safe. The limiter can be safely used concurrently
// across multiple goroutines and requests.
package adaptive
