// Package otel provides OpenTelemetry tracing middleware for Mizu.
// This is a lightweight implementation compatible with OpenTelemetry propagation.
package otel

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-mizu/mizu"
)

// Options configures the otel middleware.
type Options struct {
	// ServiceName is the name of the service.
	ServiceName string

	// ServiceVersion is the version of the service.
	ServiceVersion string

	// TracerName is the name of the tracer.
	TracerName string

	// SkipPaths are paths to skip from tracing.
	SkipPaths []string

	// Propagator specifies the propagation format.
	// Default: "traceparent" (W3C Trace Context).
	Propagator string

	// Sampler determines whether to sample spans.
	// Return true to sample, false to skip.
	Sampler func(path string) bool

	// OnStart is called when a span starts.
	OnStart func(span *Span)

	// OnEnd is called when a span ends.
	OnEnd func(span *Span)

	// SpanProcessor processes completed spans.
	SpanProcessor SpanProcessor
}

// SpanProcessor processes spans.
type SpanProcessor interface {
	Process(span *Span)
}

// SpanContext holds trace context.
type SpanContext struct {
	TraceID    string
	SpanID     string
	TraceFlags byte
	TraceState string
}

// IsValid returns true if the span context is valid.
func (sc SpanContext) IsValid() bool {
	return sc.TraceID != "" && sc.SpanID != ""
}

// IsSampled returns true if sampling flag is set.
func (sc SpanContext) IsSampled() bool {
	return sc.TraceFlags&0x01 == 0x01
}

// Span represents a trace span.
type Span struct {
	Name       string
	Context    SpanContext
	Parent     SpanContext
	StartTime  time.Time
	EndTime    time.Time
	Status     SpanStatus
	Attributes map[string]any
	Events     []SpanEvent
	Links      []SpanLink
	mu         sync.Mutex
}

// SpanStatus represents the span status.
type SpanStatus struct {
	Code        StatusCode
	Description string
}

// StatusCode represents the status of a span.
type StatusCode int

// Status codes
const (
	StatusUnset StatusCode = iota
	StatusOK
	StatusError
)

// SpanEvent represents an event within a span.
type SpanEvent struct {
	Name       string
	Time       time.Time
	Attributes map[string]any
}

// SpanLink represents a link to another span.
type SpanLink struct {
	Context    SpanContext
	Attributes map[string]any
}

// contextKey is a private type for context keys.
type contextKey struct{}

// spanKey stores the current span.
var spanKey = contextKey{}

// New creates otel middleware with default options.
func New() mizu.Middleware {
	return WithOptions(Options{})
}

// WithOptions creates otel middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	if opts.ServiceName == "" {
		opts.ServiceName = "unknown-service"
	}
	if opts.TracerName == "" {
		opts.TracerName = "github.com/go-mizu/mizu/middlewares/otel"
	}
	if opts.Propagator == "" {
		opts.Propagator = "traceparent"
	}

	skipPaths := make(map[string]bool)
	for _, p := range opts.SkipPaths {
		skipPaths[p] = true
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			path := c.Request().URL.Path

			// Skip configured paths
			if skipPaths[path] {
				return next(c)
			}

			// Check sampler
			if opts.Sampler != nil && !opts.Sampler(path) {
				return next(c)
			}

			// Extract parent context from headers
			parentCtx := extractContext(c.Request(), opts.Propagator)

			// Create span
			span := &Span{
				Name:       fmt.Sprintf("%s %s", c.Request().Method, path),
				StartTime:  time.Now(),
				Attributes: make(map[string]any),
				Events:     make([]SpanEvent, 0),
				Links:      make([]SpanLink, 0),
			}

			// Generate trace context
			if parentCtx.IsValid() {
				span.Parent = parentCtx
				span.Context = SpanContext{
					TraceID:    parentCtx.TraceID,
					SpanID:     generateSpanID(),
					TraceFlags: parentCtx.TraceFlags,
					TraceState: parentCtx.TraceState,
				}
			} else {
				span.Context = SpanContext{
					TraceID:    generateTraceID(),
					SpanID:     generateSpanID(),
					TraceFlags: 0x01, // Sampled
				}
			}

			// Add default attributes
			span.SetAttribute("service.name", opts.ServiceName)
			span.SetAttribute("service.version", opts.ServiceVersion)
			span.SetAttribute("http.method", c.Request().Method)
			span.SetAttribute("http.url", c.Request().URL.String())
			span.SetAttribute("http.host", c.Request().Host)
			span.SetAttribute("http.user_agent", c.Request().UserAgent())
			span.SetAttribute("http.route", path)

			if opts.OnStart != nil {
				opts.OnStart(span)
			}

			// Inject context into response headers
			injectContext(c.Header(), span.Context, opts.Propagator)

			// Store span in context
			ctx := context.WithValue(c.Context(), spanKey, span)
			req := c.Request().WithContext(ctx)
			*c.Request() = *req

			// Wrap response writer to capture status
			rw := &responseWriter{
				ResponseWriter: c.Writer(),
				statusCode:     http.StatusOK,
			}
			c.SetWriter(rw)

			// Execute handler
			err := next(c)

			// End span
			span.EndTime = time.Now()
			span.SetAttribute("http.status_code", rw.statusCode)

			if err != nil {
				span.Status = SpanStatus{
					Code:        StatusError,
					Description: err.Error(),
				}
				span.SetAttribute("error", true)
				span.SetAttribute("error.message", err.Error())
			} else if rw.statusCode >= 400 {
				span.Status = SpanStatus{
					Code:        StatusError,
					Description: http.StatusText(rw.statusCode),
				}
			} else {
				span.Status = SpanStatus{Code: StatusOK}
			}

			if opts.OnEnd != nil {
				opts.OnEnd(span)
			}

			if opts.SpanProcessor != nil {
				opts.SpanProcessor.Process(span)
			}

			return err
		}
	}
}

func extractContext(r *http.Request, propagator string) SpanContext {
	switch propagator {
	case "traceparent":
		return extractW3CTraceContext(r)
	case "b3":
		return extractB3Context(r)
	default:
		return extractW3CTraceContext(r)
	}
}

func extractW3CTraceContext(r *http.Request) SpanContext {
	traceparent := r.Header.Get("traceparent")
	if traceparent == "" {
		return SpanContext{}
	}

	// Format: version-traceId-spanId-flags
	parts := strings.Split(traceparent, "-")
	if len(parts) != 4 {
		return SpanContext{}
	}

	flags, _ := hex.DecodeString(parts[3])
	var traceFlags byte
	if len(flags) > 0 {
		traceFlags = flags[0]
	}

	return SpanContext{
		TraceID:    parts[1],
		SpanID:     parts[2],
		TraceFlags: traceFlags,
		TraceState: r.Header.Get("tracestate"),
	}
}

func extractB3Context(r *http.Request) SpanContext {
	// Check single header format first
	b3 := r.Header.Get("b3")
	if b3 != "" {
		parts := strings.Split(b3, "-")
		if len(parts) >= 2 {
			ctx := SpanContext{
				TraceID: parts[0],
				SpanID:  parts[1],
			}
			if len(parts) >= 3 && parts[2] == "1" {
				ctx.TraceFlags = 0x01
			}
			return ctx
		}
	}

	// Multi-header format
	traceID := r.Header.Get("X-B3-TraceId")
	spanID := r.Header.Get("X-B3-SpanId")
	if traceID == "" || spanID == "" {
		return SpanContext{}
	}

	var flags byte
	if r.Header.Get("X-B3-Sampled") == "1" {
		flags = 0x01
	}

	return SpanContext{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: flags,
	}
}

func injectContext(h http.Header, ctx SpanContext, propagator string) {
	switch propagator {
	case "traceparent":
		injectW3CTraceContext(h, ctx)
	case "b3":
		injectB3Context(h, ctx)
	default:
		injectW3CTraceContext(h, ctx)
	}
}

func injectW3CTraceContext(h http.Header, ctx SpanContext) {
	traceparent := fmt.Sprintf("00-%s-%s-%02x", ctx.TraceID, ctx.SpanID, ctx.TraceFlags)
	h.Set("traceparent", traceparent)
	if ctx.TraceState != "" {
		h.Set("tracestate", ctx.TraceState)
	}
}

func injectB3Context(h http.Header, ctx SpanContext) {
	sampled := "0"
	if ctx.IsSampled() {
		sampled = "1"
	}
	h.Set("X-B3-TraceId", ctx.TraceID)
	h.Set("X-B3-SpanId", ctx.SpanID)
	h.Set("X-B3-Sampled", sampled)
}

func generateTraceID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func generateSpanID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// SetAttribute sets an attribute on the span.
func (s *Span) SetAttribute(key string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Attributes[key] = value
}

// AddEvent adds an event to the span.
func (s *Span) AddEvent(name string, attrs map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Events = append(s.Events, SpanEvent{
		Name:       name,
		Time:       time.Now(),
		Attributes: attrs,
	})
}

// AddLink adds a link to another span.
func (s *Span) AddLink(ctx SpanContext, attrs map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Links = append(s.Links, SpanLink{
		Context:    ctx,
		Attributes: attrs,
	})
}

// SetStatus sets the span status.
func (s *Span) SetStatus(code StatusCode, description string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = SpanStatus{
		Code:        code,
		Description: description,
	}
}

// Duration returns the span duration.
func (s *Span) Duration() time.Duration {
	if s.EndTime.IsZero() {
		return time.Since(s.StartTime)
	}
	return s.EndTime.Sub(s.StartTime)
}

// GetSpan returns the current span from context.
func GetSpan(c *mizu.Ctx) *Span {
	if span, ok := c.Context().Value(spanKey).(*Span); ok {
		return span
	}
	return nil
}

// SpanFromContext returns the current span from Go context.
func SpanFromContext(ctx context.Context) *Span {
	if span, ok := ctx.Value(spanKey).(*Span); ok {
		return span
	}
	return nil
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// InMemoryProcessor stores spans in memory for testing.
type InMemoryProcessor struct {
	mu    sync.Mutex
	spans []*Span
}

// Process adds a span to the processor.
func (p *InMemoryProcessor) Process(span *Span) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.spans = append(p.spans, span)
}

// Spans returns all processed spans.
func (p *InMemoryProcessor) Spans() []*Span {
	p.mu.Lock()
	defer p.mu.Unlock()
	result := make([]*Span, len(p.spans))
	copy(result, p.spans)
	return result
}

// Clear clears all spans.
func (p *InMemoryProcessor) Clear() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.spans = nil
}

// PrintProcessor prints spans to stdout.
type PrintProcessor struct{}

// Process prints the span.
func (p *PrintProcessor) Process(span *Span) {
	fmt.Printf("[TRACE] %s | TraceID: %s | SpanID: %s | Duration: %v | Status: %d\n",
		span.Name, span.Context.TraceID, span.Context.SpanID,
		span.Duration(), span.Status.Code)
}
