package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	m, middleware := New()

	app := mizu.NewRouter()
	app.Use(middleware)

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	stats := m.Stats()
	if stats.RequestCount != 1 {
		t.Errorf("expected 1 request, got %d", stats.RequestCount)
	}
}

func TestMetrics_StatusCodes(t *testing.T) {
	m, middleware := New()

	app := mizu.NewRouter()
	app.Use(middleware)

	app.Get("/ok", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})
	app.Get("/error", func(c *mizu.Ctx) error {
		return c.Text(http.StatusInternalServerError, "error")
	})

	// OK request
	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Error request
	req = httptest.NewRequest(http.MethodGet, "/error", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	stats := m.Stats()
	if stats.StatusCodes[200] != 1 {
		t.Errorf("expected 1 200 status, got %d", stats.StatusCodes[200])
	}
	if stats.StatusCodes[500] != 1 {
		t.Errorf("expected 1 500 status, got %d", stats.StatusCodes[500])
	}
	if stats.ErrorCount != 1 {
		t.Errorf("expected 1 error, got %d", stats.ErrorCount)
	}
}

func TestMetrics_PathCounts(t *testing.T) {
	m, middleware := New()

	app := mizu.NewRouter()
	app.Use(middleware)

	app.Get("/api/users", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "users")
	})
	app.Get("/api/posts", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "posts")
	})

	// Multiple requests to users
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}

	// One request to posts
	req := httptest.NewRequest(http.MethodGet, "/api/posts", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	stats := m.Stats()
	if stats.PathCounts["/api/users"] != 3 {
		t.Errorf("expected 3 requests to /api/users, got %d", stats.PathCounts["/api/users"])
	}
	if stats.PathCounts["/api/posts"] != 1 {
		t.Errorf("expected 1 request to /api/posts, got %d", stats.PathCounts["/api/posts"])
	}
}

func TestMetrics_Handler(t *testing.T) {
	m, middleware := New()

	app := mizu.NewRouter()
	app.Use(middleware)

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})
	app.Get("/metrics", m.Handler())

	// Make a request
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Get metrics
	req = httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !strings.Contains(rec.Header().Get("Content-Type"), "application/json") {
		t.Error("expected JSON content type")
	}
}

func TestMetrics_Prometheus(t *testing.T) {
	m, middleware := New()

	app := mizu.NewRouter()
	app.Use(middleware)

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})
	app.Get("/metrics", m.Prometheus())

	// Make a request
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Get metrics
	req = httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "http_requests_total") {
		t.Error("expected prometheus metric")
	}
}

func TestMetrics_Reset(t *testing.T) {
	m, middleware := New()

	app := mizu.NewRouter()
	app.Use(middleware)

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	// Make requests
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}

	m.Reset()

	stats := m.Stats()
	if stats.RequestCount != 0 {
		t.Errorf("expected 0 after reset, got %d", stats.RequestCount)
	}
}

func TestMetrics_Concurrent(t *testing.T) {
	m, middleware := New()

	app := mizu.NewRouter()
	app.Use(middleware)

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)
		}()
	}

	wg.Wait()

	stats := m.Stats()
	if stats.RequestCount != 100 {
		t.Errorf("expected 100 requests, got %d", stats.RequestCount)
	}
}

func TestMetrics_AverageDuration(t *testing.T) {
	m, middleware := New()

	app := mizu.NewRouter()
	app.Use(middleware)

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}

	stats := m.Stats()
	if stats.AverageDurationMs <= 0 {
		t.Errorf("expected positive average duration, got %f", stats.AverageDurationMs)
	}
}

func TestMetrics_JSON(t *testing.T) {
	m, _ := New()

	data, err := m.JSON()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(string(data), "request_count") {
		t.Error("expected JSON to contain request_count")
	}
}
