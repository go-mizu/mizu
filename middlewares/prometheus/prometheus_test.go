package prometheus

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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

func TestMetricsEndpoint(t *testing.T) {
	metrics := NewMetrics(Options{})

	app := mizu.NewRouter()
	app.Use(metrics.Middleware())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})
	app.Get("/metrics", metrics.Handler())

	// Make some requests
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}

	// Get metrics
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}

	body := rec.Body.String()

	if !strings.Contains(body, "http_requests_total") {
		t.Error("expected http_requests_total metric")
	}

	if !strings.Contains(body, "http_request_duration_seconds") {
		t.Error("expected http_request_duration_seconds metric")
	}
}

func TestRequestCounter(t *testing.T) {
	metrics := NewMetrics(Options{})

	app := mizu.NewRouter()
	app.Use(metrics.Middleware())

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	// Make requests
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}

	if metrics.TotalRequests() != 3 {
		t.Errorf("expected 3 total requests, got %d", metrics.TotalRequests())
	}
}

func TestDifferentStatusCodes(t *testing.T) {
	metrics := NewMetrics(Options{})

	app := mizu.NewRouter()
	app.Use(metrics.Middleware())

	app.Get("/ok", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})
	app.Get("/error", func(c *mizu.Ctx) error {
		return c.Text(http.StatusInternalServerError, "error")
	})
	app.Get("/metrics", metrics.Handler())

	// OK request
	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Error request
	req = httptest.NewRequest(http.MethodGet, "/error", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Get metrics
	req = httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	body := rec.Body.String()

	if !strings.Contains(body, `status="200"`) {
		t.Error("expected status 200 label")
	}

	if !strings.Contains(body, `status="500"`) {
		t.Error("expected status 500 label")
	}
}

func TestNamespace(t *testing.T) {
	metrics := NewMetrics(Options{
		Namespace: "myapp",
		Subsystem: "http",
	})

	app := mizu.NewRouter()
	app.Use(metrics.Middleware())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})
	app.Get("/metrics", metrics.Handler())

	// Make request
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Get metrics
	req = httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	body := rec.Body.String()

	if !strings.Contains(body, "myapp_http_http_requests_total") {
		t.Error("expected namespaced metric name")
	}
}

func TestSkipPaths(t *testing.T) {
	metrics := NewMetrics(Options{
		SkipPaths: []string{"/health"},
	})

	app := mizu.NewRouter()
	app.Use(metrics.Middleware())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})
	app.Get("/health", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "healthy")
	})

	// Request to /health (should be skipped)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Request to / (should be recorded)
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Only the / request should be counted
	if metrics.TotalRequests() != 1 {
		t.Errorf("expected 1 total request (health skipped), got %d", metrics.TotalRequests())
	}
}

func TestHistogramBuckets(t *testing.T) {
	metrics := NewMetrics(Options{
		Buckets: []float64{0.1, 0.5, 1.0, 5.0},
	})

	app := mizu.NewRouter()
	app.Use(metrics.Middleware())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})
	app.Get("/metrics", metrics.Handler())

	// Make request
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Get metrics
	req = httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	body := rec.Body.String()

	// Check for custom bucket values
	if !strings.Contains(body, `le="0.1"`) {
		t.Error("expected le=0.1 bucket")
	}
	if !strings.Contains(body, `le="5"`) {
		t.Error("expected le=5 bucket")
	}
}

func TestActiveRequests(t *testing.T) {
	metrics := NewMetrics(Options{})

	// Initially no active requests
	if metrics.ActiveRequests() != 0 {
		t.Errorf("expected 0 active requests, got %d", metrics.ActiveRequests())
	}
}

func TestRegisterEndpoint(t *testing.T) {
	metrics := NewMetrics(Options{
		MetricsPath: "/custom-metrics",
	})

	app := mizu.NewRouter()
	app.Use(metrics.Middleware())
	metrics.RegisterEndpoint(app)

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	// Make a request first
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Get metrics from custom path
	req = httptest.NewRequest(http.MethodGet, "/custom-metrics", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}

	if !strings.Contains(rec.Header().Get("Content-Type"), "text/plain") {
		t.Error("expected text/plain content type")
	}
}

func TestExportFormat(t *testing.T) {
	metrics := NewMetrics(Options{})

	app := mizu.NewRouter()
	app.Use(metrics.Middleware())

	app.Get("/api/users", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "users")
	})
	app.Get("/metrics", metrics.Handler())

	// Make request
	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Get metrics
	req = httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	body := rec.Body.String()

	// Check format
	if !strings.Contains(body, "# HELP") {
		t.Error("expected HELP comments")
	}
	if !strings.Contains(body, "# TYPE") {
		t.Error("expected TYPE comments")
	}
	if !strings.Contains(body, "_bucket{") {
		t.Error("expected histogram buckets")
	}
	if !strings.Contains(body, "_sum") {
		t.Error("expected histogram sum")
	}
	if !strings.Contains(body, "_count") {
		t.Error("expected histogram count")
	}
}

func TestResponseSizeTracking(t *testing.T) {
	metrics := NewMetrics(Options{})

	app := mizu.NewRouter()
	app.Use(metrics.Middleware())

	app.Get("/large", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, strings.Repeat("x", 1000))
	})
	app.Get("/metrics", metrics.Handler())

	// Make request
	req := httptest.NewRequest(http.MethodGet, "/large", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Get metrics
	req = httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	body := rec.Body.String()

	if !strings.Contains(body, "http_response_size_bytes") {
		t.Error("expected response size metric")
	}
}

func TestMethodLabels(t *testing.T) {
	metrics := NewMetrics(Options{})

	app := mizu.NewRouter()
	app.Use(metrics.Middleware())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})
	app.Post("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusCreated, "created")
	})
	app.Get("/metrics", metrics.Handler())

	// GET request
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// POST request
	req = httptest.NewRequest(http.MethodPost, "/", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Get metrics
	req = httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	body := rec.Body.String()

	if !strings.Contains(body, `method="GET"`) {
		t.Error("expected GET method label")
	}
	if !strings.Contains(body, `method="POST"`) {
		t.Error("expected POST method label")
	}
}
