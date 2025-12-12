// Package trace provides distributed tracing middleware for Mizu.
package trace

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/go-mizu/mizu"
)

type contextKey struct{}

// Span represents a trace span.
type Span struct {
	TraceID   string
	SpanID    string
	ParentID  string
	Name      string
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration
	Status    SpanStatus
	Tags      map[string]string
	Events    []Event
}

// SpanStatus represents the span status.
type SpanStatus int

const (
	StatusUnset SpanStatus = iota
	StatusOK
	StatusError
)

// Event represents a span event.
type Event struct {
	Name      string
	Timestamp time.Time
	Tags      map[string]string
}

// Options configures the trace middleware.
type Options struct {
	// ServiceName is the service name for spans.
	// Default: "mizu-service".
	ServiceName string

	// TraceHeader is the header for propagating trace ID.
	// Default: "X-Trace-ID".
	TraceHeader string

	// ParentHeader is the header for propagating parent span ID.
	// Default: "X-Parent-ID".
	ParentHeader string

	// OnSpan is called when a span completes.
	OnSpan func(span *Span)

	// Sampler determines if a request should be traced.
	// Default: all requests are traced.
	Sampler func(c *mizu.Ctx) bool
}

// New creates trace middleware with default options.
func New() mizu.Middleware {
	return WithOptions(Options{})
}

// WithOptions creates trace middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	if opts.ServiceName == "" {
		opts.ServiceName = "mizu-service"
	}
	if opts.TraceHeader == "" {
		opts.TraceHeader = "X-Trace-ID"
	}
	if opts.ParentHeader == "" {
		opts.ParentHeader = "X-Parent-ID"
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			// Check sampler
			if opts.Sampler != nil && !opts.Sampler(c) {
				return next(c)
			}

			// Get or create trace ID
			traceID := c.Request().Header.Get(opts.TraceHeader)
			if traceID == "" {
				traceID = generateID()
			}

			// Get parent ID
			parentID := c.Request().Header.Get(opts.ParentHeader)

			// Create span
			span := &Span{
				TraceID:   traceID,
				SpanID:    generateID(),
				ParentID:  parentID,
				Name:      c.Request().Method + " " + c.Request().URL.Path,
				StartTime: time.Now(),
				Tags: map[string]string{
					"http.method": c.Request().Method,
					"http.path":   c.Request().URL.Path,
					"service":     opts.ServiceName,
				},
			}

			// Store in context
			ctx := context.WithValue(c.Context(), contextKey{}, span)
			req := c.Request().WithContext(ctx)
			*c.Request() = *req

			// Set response headers for propagation
			c.Header().Set(opts.TraceHeader, traceID)

			// Execute handler
			err := next(c)

			// Complete span
			span.EndTime = time.Now()
			span.Duration = span.EndTime.Sub(span.StartTime)

			if err != nil {
				span.Status = StatusError
				span.Tags["error"] = err.Error()
			} else {
				span.Status = StatusOK
			}

			// Call span handler
			if opts.OnSpan != nil {
				opts.OnSpan(span)
			}

			return err
		}
	}
}

func generateID() string {
	bytes := make([]byte, 16)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// Get retrieves the current span from context.
func Get(c *mizu.Ctx) *Span {
	if span, ok := c.Context().Value(contextKey{}).(*Span); ok {
		return span
	}
	return nil
}

// TraceID returns the current trace ID.
func TraceID(c *mizu.Ctx) string {
	if span := Get(c); span != nil {
		return span.TraceID
	}
	return ""
}

// SpanID returns the current span ID.
func SpanID(c *mizu.Ctx) string {
	if span := Get(c); span != nil {
		return span.SpanID
	}
	return ""
}

// AddTag adds a tag to the current span.
func AddTag(c *mizu.Ctx, key, value string) {
	if span := Get(c); span != nil {
		span.Tags[key] = value
	}
}

// AddEvent adds an event to the current span.
func AddEvent(c *mizu.Ctx, name string, tags map[string]string) {
	if span := Get(c); span != nil {
		span.Events = append(span.Events, Event{
			Name:      name,
			Timestamp: time.Now(),
			Tags:      tags,
		})
	}
}

// SetStatus sets the span status.
func SetStatus(c *mizu.Ctx, status SpanStatus) {
	if span := Get(c); span != nil {
		span.Status = status
	}
}

// Collector collects completed spans.
type Collector struct {
	spans []*Span
}

// NewCollector creates a new span collector.
func NewCollector() *Collector {
	return &Collector{}
}

// Collect collects a span.
func (c *Collector) Collect(span *Span) {
	c.spans = append(c.spans, span)
}

// Spans returns collected spans.
func (c *Collector) Spans() []*Span {
	return c.spans
}

// Clear clears collected spans.
func (c *Collector) Clear() {
	c.spans = nil
}

// WithCollector creates middleware with a collector.
func WithCollector(collector *Collector) mizu.Middleware {
	return WithOptions(Options{
		OnSpan: collector.Collect,
	})
}

// W3CTraceContext creates middleware using W3C Trace Context headers.
func W3CTraceContext() mizu.Middleware {
	return WithOptions(Options{
		TraceHeader:  "traceparent",
		ParentHeader: "tracestate",
	})
}

// HTTPHeaders returns headers for propagating trace context.
func HTTPHeaders(c *mizu.Ctx) http.Header {
	span := Get(c)
	if span == nil {
		return nil
	}

	headers := make(http.Header)
	headers.Set("X-Trace-ID", span.TraceID)
	headers.Set("X-Parent-ID", span.SpanID)
	return headers
}
