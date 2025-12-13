// Package ratelimit provides token bucket rate limiting middleware for Mizu web applications.
//
// # Overview
//
// The ratelimit middleware implements the token bucket algorithm to control request rates,
// protecting applications from abuse, ensuring fair usage, and preventing resource exhaustion.
// It provides flexible rate limiting based on various criteria such as IP address, API keys,
// or custom user identifiers.
//
// # Quick Start
//
// Basic usage with default IP-based rate limiting:
//
//	import (
//	    "github.com/go-mizu/mizu"
//	    "github.com/go-mizu/mizu/middlewares/ratelimit"
//	)
//
//	app := mizu.New()
//
//	// Allow 100 requests per minute per IP
//	app.Use(ratelimit.PerMinute(100))
//
// # Configuration Options
//
// The middleware supports various configuration options through the Options struct:
//
//   - Rate: Number of requests allowed per interval (default: 100)
//   - Interval: Time window for rate limiting (default: 1 minute)
//   - Burst: Maximum burst capacity (default: same as Rate)
//   - KeyFunc: Function to extract rate limit key from request (default: client IP)
//   - Headers: Include rate limit headers in response (default: true)
//   - ErrorHandler: Custom handler for rate limit exceeded errors
//   - Skip: Function to skip rate limiting for certain requests
//
// # Convenience Functions
//
// The package provides convenience functions for common time intervals:
//
//	// 10 requests per second
//	app.Use(ratelimit.PerSecond(10))
//
//	// 100 requests per minute
//	app.Use(ratelimit.PerMinute(100))
//
//	// 1000 requests per hour
//	app.Use(ratelimit.PerHour(1000))
//
// # Custom Configuration
//
// For advanced use cases, use WithOptions for full control:
//
//	app.Use(ratelimit.WithOptions(ratelimit.Options{
//	    Rate:     100,
//	    Interval: time.Minute,
//	    Burst:    200, // Allow burst up to 200
//	    KeyFunc: func(c *mizu.Ctx) string {
//	        // Rate limit by API key instead of IP
//	        return c.Request().Header.Get("X-API-Key")
//	    },
//	    Skip: func(c *mizu.Ctx) bool {
//	        // Skip health check endpoints
//	        return c.Request().URL.Path == "/health"
//	    },
//	}))
//
// # Rate Limiting by User
//
// Rate limit authenticated users by their user ID:
//
//	app.Use(ratelimit.WithOptions(ratelimit.Options{
//	    Rate:     100,
//	    Interval: time.Minute,
//	    KeyFunc: func(c *mizu.Ctx) string {
//	        if user := GetUser(c); user != nil {
//	            return user.ID
//	        }
//	        return c.ClientIP() // Fallback to IP
//	    },
//	}))
//
// # Custom Error Handling
//
// Provide custom error responses when rate limit is exceeded:
//
//	app.Use(ratelimit.WithOptions(ratelimit.Options{
//	    Rate:     100,
//	    Interval: time.Minute,
//	    ErrorHandler: func(c *mizu.Ctx) error {
//	        return c.JSON(429, map[string]any{
//	            "error":       "Rate limit exceeded",
//	            "retry_after": c.Header().Get("Retry-After"),
//	        })
//	    },
//	}))
//
// # Response Headers
//
// When Headers option is enabled (default), the following headers are included in responses:
//
//   - X-RateLimit-Limit: Maximum requests allowed per interval
//   - X-RateLimit-Remaining: Remaining requests in current window
//   - X-RateLimit-Reset: Unix timestamp when the rate limit resets
//   - Retry-After: Seconds to wait before retrying (only when rate limited)
//
// # Token Bucket Algorithm
//
// The middleware uses the token bucket algorithm for rate limiting:
//
//  1. Each bucket starts with Burst tokens
//  2. Each request consumes one token
//  3. Tokens refill continuously at Rate per Interval
//  4. Bucket capacity is capped at Burst tokens
//  5. Request is allowed only if at least 1 token is available
//
// This algorithm allows for burst traffic while maintaining average rate limits.
//
// # Custom Storage Backend
//
// For distributed systems, implement a custom Store interface:
//
//	type Store interface {
//	    Allow(key string, rate int, interval time.Duration, burst int) (bool, RateLimitInfo)
//	}
//
// Example with Redis or other distributed storage:
//
//	type RedisStore struct {
//	    client *redis.Client
//	}
//
//	func (s *RedisStore) Allow(key string, rate int, interval time.Duration, burst int) (bool, RateLimitInfo) {
//	    // Implement token bucket algorithm using Redis
//	    // Return whether request is allowed and current rate limit info
//	}
//
//	app.Use(ratelimit.WithStore(&RedisStore{client}, ratelimit.Options{
//	    Rate:     100,
//	    Interval: time.Minute,
//	}))
//
// # Memory Store
//
// The default MemoryStore provides:
//
//   - Thread-safe operation using sync.Mutex
//   - Per-key bucket tracking for independent rate limits
//   - Automatic cleanup of stale buckets (10 minute timeout)
//   - Precise token refill using float64 calculations
//
// Note: MemoryStore is suitable for single-instance deployments. For distributed
// systems, implement a custom Store backed by Redis, Memcached, or similar.
//
// # Thread Safety
//
// The MemoryStore implementation is fully thread-safe and can handle concurrent
// requests safely. Custom Store implementations should also ensure thread safety.
//
// # Best Practices
//
//   - Set reasonable limits based on expected usage patterns
//   - Use different rate limits for different endpoint types
//   - Include rate limit headers to inform clients of their usage
//   - Monitor rate limit hits to tune limits appropriately
//   - Consider tiered limits for different user types (free vs. premium)
//   - Use Skip function to exclude health checks and metrics endpoints
//   - Implement custom Store for distributed deployments
package ratelimit
