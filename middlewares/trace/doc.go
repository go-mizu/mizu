// Package trace provides distributed tracing middleware for the Mizu web framework.
//
// The trace middleware enables distributed tracing by propagating trace context
// across service boundaries. It creates and manages spans for each request,
// tracking timing, status, and metadata throughout the request lifecycle.
//
// # Basic Usage
//
// Create a trace middleware with default options:
//
//	app := mizu.New()
//	app.Use(trace.New())
//
//	app.Get("/api", func(c *mizu.Ctx) error {
//	    traceID := trace.TraceID(c)
//	    log.Printf("Trace: %s", traceID)
//	    return c.JSON(200, data)
//	})
//
// # Configuration
//
// Customize the middleware with options:
//
//	app.Use(trace.WithOptions(trace.Options{
//	    ServiceName:  "my-service",
//	    TraceHeader:  "X-Request-ID",
//	    ParentHeader: "X-Parent-Span-ID",
//	    OnSpan: func(span *trace.Span) {
//	        // Send span to your tracing backend
//	        log.Printf("Span completed: %s (duration: %v)", span.Name, span.Duration)
//	    },
//	    Sampler: func(c *mizu.Ctx) bool {
//	        // Sample 10% of requests
//	        return rand.Float64() < 0.1
//	    },
//	}))
//
// # W3C Trace Context
//
// Use W3C Trace Context standard headers:
//
//	app.Use(trace.W3CTraceContext())
//
// This uses the "traceparent" and "tracestate" headers for propagation.
//
// # Adding Metadata
//
// Enrich spans with custom tags and events:
//
//	app.Get("/users/:id", func(c *mizu.Ctx) error {
//	    userID := c.Param("id")
//
//	    // Add custom tag
//	    trace.AddTag(c, "user.id", userID)
//
//	    // Add event
//	    trace.AddEvent(c, "database.query", map[string]string{
//	        "table": "users",
//	        "id":    userID,
//	    })
//
//	    // Retrieve user data...
//	    return c.JSON(200, user)
//	})
//
// # Propagating to Downstream Services
//
// Propagate trace context when calling other services:
//
//	app.Get("/", func(c *mizu.Ctx) error {
//	    // Create HTTP request to downstream service
//	    req, _ := http.NewRequest("GET", "http://service-b/api", nil)
//
//	    // Inject trace headers
//	    headers := trace.HTTPHeaders(c)
//	    for key, values := range headers {
//	        for _, value := range values {
//	            req.Header.Add(key, value)
//	        }
//	    }
//
//	    resp, _ := http.DefaultClient.Do(req)
//	    defer resp.Body.Close()
//
//	    return c.JSON(200, result)
//	})
//
// # Collecting Spans
//
// Use a collector for testing or custom backends:
//
//	collector := trace.NewCollector()
//	app.Use(trace.WithCollector(collector))
//
//	// After requests are processed
//	for _, span := range collector.Spans() {
//	    fmt.Printf("Span: %s, Duration: %v\n", span.Name, span.Duration)
//	}
//
//	// Clear for next test
//	collector.Clear()
//
// # Span Structure
//
// Each span contains:
//   - TraceID: Unique identifier for the entire distributed trace
//   - SpanID: Unique identifier for this specific operation
//   - ParentID: Reference to the parent span (if any)
//   - Name: Descriptive name (e.g., "GET /api/users")
//   - StartTime/EndTime: Timing information
//   - Duration: Calculated from start and end times
//   - Status: StatusUnset, StatusOK, or StatusError
//   - Tags: Key-value metadata (e.g., "http.method", "service")
//   - Events: Timestamped events that occurred during the span
//
// # Thread Safety
//
// The trace middleware is safe for concurrent use. Each request gets its own
// span stored in the request context. The Collector type is not thread-safe
// and should be used carefully in concurrent scenarios.
//
// # Performance Considerations
//
// Tracing adds minimal overhead:
//   - ID generation uses crypto/rand (secure but slower than PRNG)
//   - Context storage and retrieval is fast
//   - Sampling can reduce overhead for high-traffic services
//   - OnSpan callbacks should be non-blocking for best performance
//
// # Integration with Tracing Systems
//
// The OnSpan callback allows integration with any tracing backend:
//
//	app.Use(trace.WithOptions(trace.Options{
//	    OnSpan: func(span *trace.Span) {
//	        // Send to Jaeger, Zipkin, DataDog, etc.
//	        sendToJaeger(span)
//	    },
//	}))
//
// For OpenTelemetry integration, see the otel middleware.
package trace
