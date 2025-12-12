package logger

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	var buf bytes.Buffer
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Output: &buf,
	}))

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "hello")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	log := buf.String()
	if !strings.Contains(log, "200") {
		t.Errorf("expected log to contain status 200, got: %s", log)
	}
	if !strings.Contains(log, "GET") {
		t.Errorf("expected log to contain method GET, got: %s", log)
	}
	if !strings.Contains(log, "/test") {
		t.Errorf("expected log to contain path /test, got: %s", log)
	}
}

func TestWithOptions_CustomFormat(t *testing.T) {
	var buf bytes.Buffer
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Output: &buf,
		Format: "[${method}] ${path} -> ${status}\n",
	}))

	app.Get("/api", func(c *mizu.Ctx) error {
		return c.Text(http.StatusCreated, "created")
	})

	req := httptest.NewRequest(http.MethodGet, "/api", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	expected := "[GET] /api -> 201\n"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}

func TestWithOptions_Skip(t *testing.T) {
	var buf bytes.Buffer
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Output: &buf,
		Skip: func(c *mizu.Ctx) bool {
			return c.Request().URL.Path == "/health"
		},
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

	if buf.Len() != 0 {
		t.Errorf("expected no log for /health, got: %s", buf.String())
	}

	// API should be logged
	buf.Reset()
	req = httptest.NewRequest(http.MethodGet, "/api", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !strings.Contains(buf.String(), "/api") {
		t.Error("expected log for /api")
	}
}

func TestWithOptions_Headers(t *testing.T) {
	var buf bytes.Buffer
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Output: &buf,
		Format: "${header:X-Request-ID}\n",
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "test-123")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if buf.String() != "test-123\n" {
		t.Errorf("expected 'test-123\\n', got %q", buf.String())
	}
}

func TestWithOptions_AllTags(t *testing.T) {
	var buf bytes.Buffer
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Output: &buf,
		Format: "${host}|${protocol}|${referer}|${user_agent}|${bytes_out}|${query}\n",
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "hello")
	})

	req := httptest.NewRequest(http.MethodGet, "/?foo=bar", nil)
	req.Header.Set("Referer", "http://example.com")
	req.Header.Set("User-Agent", "TestAgent")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	log := buf.String()
	if !strings.Contains(log, "http://example.com") {
		t.Error("expected referer in log")
	}
	if !strings.Contains(log, "TestAgent") {
		t.Error("expected user agent in log")
	}
	if !strings.Contains(log, "foo=bar") {
		t.Error("expected query in log")
	}
}

func TestWithOptions_XForwardedFor(t *testing.T) {
	var buf bytes.Buffer
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Output: &buf,
		Format: "${ip}\n",
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "1.2.3.4, 5.6.7.8")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if buf.String() != "1.2.3.4\n" {
		t.Errorf("expected '1.2.3.4\\n', got %q", buf.String())
	}
}

func TestWithOptions_XRealIP(t *testing.T) {
	var buf bytes.Buffer
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Output: &buf,
		Format: "${ip}\n",
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Real-IP", "10.0.0.1")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if buf.String() != "10.0.0.1\n" {
		t.Errorf("expected '10.0.0.1\\n', got %q", buf.String())
	}
}

func TestDefaultOutput(t *testing.T) {
	// Just ensure it doesn't panic with nil output
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
