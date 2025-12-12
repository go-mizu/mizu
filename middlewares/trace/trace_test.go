package trace

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	var span *Span
	app.Get("/", func(c *mizu.Ctx) error {
		span = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if span == nil {
		t.Fatal("expected span to be created")
	}
	if span.TraceID == "" {
		t.Error("expected trace ID")
	}
	if span.SpanID == "" {
		t.Error("expected span ID")
	}
}

func TestWithOptions_ServiceName(t *testing.T) {
	var capturedSpan *Span

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		ServiceName: "test-service",
		OnSpan: func(span *Span) {
			capturedSpan = span
		},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if capturedSpan.Tags["service"] != "test-service" {
		t.Errorf("expected service 'test-service', got %q", capturedSpan.Tags["service"])
	}
}

func TestWithOptions_PropagateTraceID(t *testing.T) {
	var span *Span

	app := mizu.NewRouter()
	app.Use(New())

	app.Get("/", func(c *mizu.Ctx) error {
		span = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Trace-ID", "existing-trace-id")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if span.TraceID != "existing-trace-id" {
		t.Errorf("expected existing trace ID, got %q", span.TraceID)
	}
}

func TestWithOptions_ParentID(t *testing.T) {
	var span *Span

	app := mizu.NewRouter()
	app.Use(New())

	app.Get("/", func(c *mizu.Ctx) error {
		span = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Trace-ID", "trace-1")
	req.Header.Set("X-Parent-ID", "parent-span-1")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if span.ParentID != "parent-span-1" {
		t.Errorf("expected parent ID, got %q", span.ParentID)
	}
}

func TestWithOptions_OnSpan(t *testing.T) {
	var capturedSpan *Span

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		OnSpan: func(span *Span) {
			capturedSpan = span
		},
	}))

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if capturedSpan == nil {
		t.Fatal("expected span to be captured")
	}
	if capturedSpan.Name != "GET /test" {
		t.Errorf("expected span name 'GET /test', got %q", capturedSpan.Name)
	}
	if capturedSpan.Duration == 0 {
		t.Error("expected duration to be set")
	}
}

func TestWithOptions_Sampler(t *testing.T) {
	var spanCreated bool

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Sampler: func(c *mizu.Ctx) bool {
			return false // Don't trace
		},
		OnSpan: func(span *Span) {
			spanCreated = true
		},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if spanCreated {
		t.Error("expected span to not be created when sampler returns false")
	}
}

func TestAddTag(t *testing.T) {
	var capturedSpan *Span

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		OnSpan: func(span *Span) {
			capturedSpan = span
		},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		AddTag(c, "custom.tag", "custom-value")
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if capturedSpan.Tags["custom.tag"] != "custom-value" {
		t.Errorf("expected custom tag, got %q", capturedSpan.Tags["custom.tag"])
	}
}

func TestAddEvent(t *testing.T) {
	var capturedSpan *Span

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		OnSpan: func(span *Span) {
			capturedSpan = span
		},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		AddEvent(c, "database.query", map[string]string{"sql": "SELECT *"})
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if len(capturedSpan.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(capturedSpan.Events))
	}
	if capturedSpan.Events[0].Name != "database.query" {
		t.Errorf("expected event name 'database.query', got %q", capturedSpan.Events[0].Name)
	}
}

func TestTraceID(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	var traceID string
	app.Get("/", func(c *mizu.Ctx) error {
		traceID = TraceID(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if traceID == "" {
		t.Error("expected trace ID")
	}
}

func TestSpanID(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	var spanID string
	app.Get("/", func(c *mizu.Ctx) error {
		spanID = SpanID(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if spanID == "" {
		t.Error("expected span ID")
	}
}

func TestCollector(t *testing.T) {
	collector := NewCollector()

	app := mizu.NewRouter()
	app.Use(WithCollector(collector))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	// Make 3 requests
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}

	spans := collector.Spans()
	if len(spans) != 3 {
		t.Errorf("expected 3 spans, got %d", len(spans))
	}

	collector.Clear()
	if len(collector.Spans()) != 0 {
		t.Error("expected 0 spans after clear")
	}
}

func TestResponseHeader(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("X-Trace-ID") == "" {
		t.Error("expected X-Trace-ID response header")
	}
}

func TestHTTPHeaders(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	var headers http.Header
	app.Get("/", func(c *mizu.Ctx) error {
		headers = HTTPHeaders(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if headers.Get("X-Trace-ID") == "" {
		t.Error("expected trace ID in headers")
	}
	if headers.Get("X-Parent-ID") == "" {
		t.Error("expected parent ID in headers")
	}
}
