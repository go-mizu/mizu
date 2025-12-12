package otel

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestSpanCreation(t *testing.T) {
	processor := &InMemoryProcessor{}

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		ServiceName:   "test-service",
		SpanProcessor: processor,
	}))

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	spans := processor.Spans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	span := spans[0]
	if span.Name != "GET /test" {
		t.Errorf("expected name 'GET /test', got %q", span.Name)
	}

	if span.Context.TraceID == "" {
		t.Error("expected trace ID to be set")
	}

	if span.Context.SpanID == "" {
		t.Error("expected span ID to be set")
	}
}

func TestSpanAttributes(t *testing.T) {
	processor := &InMemoryProcessor{}

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		SpanProcessor:  processor,
	}))

	app.Get("/api/users", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	req.Header.Set("User-Agent", "TestAgent")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	span := processor.Spans()[0]

	if span.Attributes["service.name"] != "test-service" {
		t.Errorf("expected service.name='test-service', got %v", span.Attributes["service.name"])
	}

	if span.Attributes["service.version"] != "1.0.0" {
		t.Errorf("expected service.version='1.0.0', got %v", span.Attributes["service.version"])
	}

	if span.Attributes["http.method"] != "GET" {
		t.Errorf("expected http.method='GET', got %v", span.Attributes["http.method"])
	}

	if span.Attributes["http.status_code"] != 200 {
		t.Errorf("expected http.status_code=200, got %v", span.Attributes["http.status_code"])
	}

	if span.Attributes["http.user_agent"] != "TestAgent" {
		t.Errorf("expected http.user_agent='TestAgent', got %v", span.Attributes["http.user_agent"])
	}
}

func TestW3CTracePropagation(t *testing.T) {
	processor := &InMemoryProcessor{}

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Propagator:    "traceparent",
		SpanProcessor: processor,
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	// Send request with traceparent header
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("traceparent", "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	span := processor.Spans()[0]

	// Should inherit trace ID
	if span.Context.TraceID != "0af7651916cd43dd8448eb211c80319c" {
		t.Errorf("expected inherited trace ID, got %q", span.Context.TraceID)
	}

	// Should have new span ID
	if span.Context.SpanID == "b7ad6b7169203331" {
		t.Error("expected new span ID, got parent span ID")
	}

	// Should have parent set
	if span.Parent.SpanID != "b7ad6b7169203331" {
		t.Errorf("expected parent span ID, got %q", span.Parent.SpanID)
	}

	// Response should have traceparent header
	if rec.Header().Get("traceparent") == "" {
		t.Error("expected traceparent header in response")
	}
}

func TestB3Propagation(t *testing.T) {
	processor := &InMemoryProcessor{}

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Propagator:    "b3",
		SpanProcessor: processor,
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	// Single header format
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("b3", "80f198ee56343ba864fe8b2a57d3eff7-e457b5a2e4d86bd1-1")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	span := processor.Spans()[0]

	if span.Context.TraceID != "80f198ee56343ba864fe8b2a57d3eff7" {
		t.Errorf("expected inherited trace ID, got %q", span.Context.TraceID)
	}

	// Response should have B3 headers
	if rec.Header().Get("X-B3-TraceId") == "" {
		t.Error("expected X-B3-TraceId header in response")
	}
}

func TestB3MultiHeaderPropagation(t *testing.T) {
	processor := &InMemoryProcessor{}

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Propagator:    "b3",
		SpanProcessor: processor,
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	// Multi-header format
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-B3-TraceId", "463ac35c9f6413ad48485a3953bb6124")
	req.Header.Set("X-B3-SpanId", "0020000000000001")
	req.Header.Set("X-B3-Sampled", "1")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	span := processor.Spans()[0]

	if span.Context.TraceID != "463ac35c9f6413ad48485a3953bb6124" {
		t.Errorf("expected inherited trace ID, got %q", span.Context.TraceID)
	}
}

func TestSkipPaths(t *testing.T) {
	processor := &InMemoryProcessor{}

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		SkipPaths:     []string{"/health"},
		SpanProcessor: processor,
	}))

	app.Get("/health", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})
	app.Get("/api", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	// Health check should be skipped
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// API should be traced
	req = httptest.NewRequest(http.MethodGet, "/api", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if len(processor.Spans()) != 1 {
		t.Errorf("expected 1 span (health skipped), got %d", len(processor.Spans()))
	}
}

func TestSampler(t *testing.T) {
	processor := &InMemoryProcessor{}

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Sampler: func(path string) bool {
			return path != "/notrack"
		},
		SpanProcessor: processor,
	}))

	app.Get("/track", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})
	app.Get("/notrack", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/track", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	req = httptest.NewRequest(http.MethodGet, "/notrack", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if len(processor.Spans()) != 1 {
		t.Errorf("expected 1 span, got %d", len(processor.Spans()))
	}
}

func TestErrorStatus(t *testing.T) {
	processor := &InMemoryProcessor{}

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		SpanProcessor: processor,
	}))

	app.Get("/error", func(c *mizu.Ctx) error {
		return c.Text(http.StatusInternalServerError, "error")
	})

	req := httptest.NewRequest(http.MethodGet, "/error", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	span := processor.Spans()[0]

	if span.Status.Code != StatusError {
		t.Errorf("expected StatusError, got %d", span.Status.Code)
	}
}

func TestOnStartOnEnd(t *testing.T) {
	var startCalled, endCalled bool

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		OnStart: func(span *Span) {
			startCalled = true
		},
		OnEnd: func(span *Span) {
			endCalled = true
		},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !startCalled {
		t.Error("expected OnStart to be called")
	}
	if !endCalled {
		t.Error("expected OnEnd to be called")
	}
}

func TestGetSpan(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	var spanFound bool

	app.Get("/", func(c *mizu.Ctx) error {
		span := GetSpan(c)
		spanFound = span != nil
		if span != nil {
			span.SetAttribute("custom", "value")
			span.AddEvent("test-event", map[string]any{"key": "value"})
		}
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !spanFound {
		t.Error("expected span to be in context")
	}
}

func TestSpanDuration(t *testing.T) {
	processor := &InMemoryProcessor{}

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		SpanProcessor: processor,
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		time.Sleep(time.Millisecond) // Ensure measurable duration on Windows
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	span := processor.Spans()[0]

	if span.Duration() <= 0 {
		t.Error("expected positive duration")
	}

	if span.EndTime.IsZero() {
		t.Error("expected end time to be set")
	}
}

func TestSpanContextIsValid(t *testing.T) {
	valid := SpanContext{
		TraceID: "abc",
		SpanID:  "123",
	}
	if !valid.IsValid() {
		t.Error("expected valid context")
	}

	invalid := SpanContext{}
	if invalid.IsValid() {
		t.Error("expected invalid context")
	}
}

func TestSpanContextIsSampled(t *testing.T) {
	sampled := SpanContext{TraceFlags: 0x01}
	if !sampled.IsSampled() {
		t.Error("expected sampled")
	}

	notSampled := SpanContext{TraceFlags: 0x00}
	if notSampled.IsSampled() {
		t.Error("expected not sampled")
	}
}

func TestInMemoryProcessor(t *testing.T) {
	processor := &InMemoryProcessor{}

	span := &Span{
		Name: "test",
		Context: SpanContext{
			TraceID: "abc",
			SpanID:  "123",
		},
	}

	processor.Process(span)

	if len(processor.Spans()) != 1 {
		t.Errorf("expected 1 span, got %d", len(processor.Spans()))
	}

	processor.Clear()

	if len(processor.Spans()) != 0 {
		t.Error("expected 0 spans after clear")
	}
}
