// Package otel provides OpenTelemetry-compatible distributed tracing middleware for Mizu.
//
// This package implements a lightweight tracing solution compatible with OpenTelemetry
// propagation standards (W3C Trace Context and B3) without requiring external dependencies.
// It provides distributed tracing capabilities for tracking requests across service boundaries.
//
// # Features
//
//   - W3C Trace Context propagation (default)
//   - B3 propagation format support (single and multi-header)
//   - Parent-child span relationships
//   - Custom span attributes and events
//   - Configurable sampling strategies
//   - Path filtering for health checks
//   - Lifecycle hooks (OnStart/OnEnd)
//   - Pluggable span processors
//   - Thread-safe operations
//
// # Basic Usage
//
//	app := mizu.New()
//
//	// Add OpenTelemetry middleware
//	app.Use(otel.New())
//
//	app.Get("/api/users", func(c *mizu.Ctx) error {
//	    return c.JSON(200, users)
//	})
//
// # Configuration
//
// Configure the middleware with custom options:
//
//	processor := &otel.InMemoryProcessor{}
//
//	app.Use(otel.WithOptions(otel.Options{
//	    ServiceName:    "my-service",
//	    ServiceVersion: "1.0.0",
//	    Propagator:     "traceparent", // or "b3"
//	    SkipPaths:      []string{"/health", "/metrics"},
//	    Sampler: func(path string) bool {
//	        // Sample 100% of /api paths
//	        return strings.HasPrefix(path, "/api")
//	    },
//	    OnStart: func(span *otel.Span) {
//	        span.SetAttribute("environment", "production")
//	    },
//	    OnEnd: func(span *otel.Span) {
//	        log.Printf("Span completed: %s (duration: %v)", span.Name, span.Duration())
//	    },
//	    SpanProcessor: processor,
//	}))
//
// # Custom Spans
//
// Create custom child spans within handlers:
//
//	app.Get("/process", func(c *mizu.Ctx) error {
//	    span := otel.GetSpan(c)
//	    if span != nil {
//	        span.SetAttribute("user.id", userID)
//	        span.AddEvent("processing started", map[string]any{
//	            "timestamp": time.Now(),
//	        })
//	    }
//	    return c.JSON(200, result)
//	})
//
// # Trace Propagation
//
// The middleware automatically handles trace context propagation:
//
//   - Extracts parent context from incoming request headers
//   - Generates new TraceID for root spans or inherits from parent
//   - Generates new SpanID for each span
//   - Injects trace context into response headers
//
// Supported propagation formats:
//
//   - W3C Trace Context: traceparent/tracestate headers
//   - B3: single header or multi-header format
//
// # Span Processors
//
// Built-in processors:
//
//	// In-memory storage for testing
//	processor := &otel.InMemoryProcessor{}
//	defer func() {
//	    for _, span := range processor.Spans() {
//	        log.Printf("Span: %s (duration: %v)", span.Name, span.Duration())
//	    }
//	}()
//
//	// Print to stdout
//	processor := &otel.PrintProcessor{}
//
// Implement custom processors by satisfying the SpanProcessor interface:
//
//	type CustomProcessor struct{}
//
//	func (p *CustomProcessor) Process(span *otel.Span) {
//	    // Export to external tracing system
//	    exportToJaeger(span)
//	}
//
// # Span Context
//
// The SpanContext holds trace propagation information:
//
//   - TraceID: 128-bit unique identifier for the entire trace
//   - SpanID: 64-bit unique identifier for the current span
//   - TraceFlags: 8-bit flags (bit 0 indicates sampling)
//   - TraceState: Optional vendor-specific data
//
// # Thread Safety
//
// All span operations are thread-safe:
//
//   - Span methods use mutex locks for attribute/event/link operations
//   - InMemoryProcessor is protected with mutex
//   - Safe for concurrent request handling
//
// # Performance
//
// Optimize performance with sampling and path filtering:
//
//	app.Use(otel.WithOptions(otel.Options{
//	    // Skip health checks
//	    SkipPaths: []string{"/health", "/ready", "/metrics"},
//
//	    // Sample 10% of traffic
//	    Sampler: func(path string) bool {
//	        return rand.Float64() < 0.1
//	    },
//	}))
//
// # Integration with External Systems
//
// While this package doesn't include external exporters, you can easily
// integrate with tracing backends using custom SpanProcessors:
//
//	type JaegerProcessor struct {
//	    endpoint string
//	}
//
//	func (p *JaegerProcessor) Process(span *otel.Span) {
//	    // Convert span to Jaeger format and export
//	    jaegerSpan := convertToJaeger(span)
//	    sendToJaeger(p.endpoint, jaegerSpan)
//	}
//
// # Compatibility
//
// This implementation is compatible with:
//
//   - OpenTelemetry specification for trace context propagation
//   - W3C Trace Context standard
//   - B3 propagation (Zipkin compatible)
//   - Any system that accepts standard trace headers
//
// For more information, see the OpenTelemetry documentation:
// https://opentelemetry.io/docs/concepts/signals/traces/
package otel
