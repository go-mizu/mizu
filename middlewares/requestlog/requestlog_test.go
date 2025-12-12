package requestlog

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	app := mizu.NewRouter()
	app.Use(WithLogger(logger))

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !strings.Contains(buf.String(), "method=GET") {
		t.Errorf("expected method in log, got %q", buf.String())
	}
	if !strings.Contains(buf.String(), "path=/test") {
		t.Errorf("expected path in log, got %q", buf.String())
	}
}

func TestWithOptions_LogHeaders(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Logger:     logger,
		LogHeaders: true,
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Custom", "value")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !strings.Contains(buf.String(), "headers") {
		t.Error("expected headers in log")
	}
}

func TestWithOptions_LogBody(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Logger:  logger,
		LogBody: true,
	}))

	app.Post("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("test body"))
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !strings.Contains(buf.String(), "test body") {
		t.Errorf("expected body in log, got %q", buf.String())
	}
}

func TestWithOptions_SkipPaths(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Logger:    logger,
		SkipPaths: []string{"/health"},
	}))

	app.Get("/health", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if strings.Contains(buf.String(), "/health") {
		t.Error("expected /health to be skipped")
	}
}

func TestWithOptions_SkipMethods(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Logger:      logger,
		SkipMethods: []string{"OPTIONS"},
	}))

	app.Handle("OPTIONS", "/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if strings.Contains(buf.String(), "OPTIONS") {
		t.Error("expected OPTIONS to be skipped")
	}
}

func TestWithOptions_SensitiveHeaders(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Logger:     logger,
		LogHeaders: true,
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if strings.Contains(buf.String(), "secret-token") {
		t.Error("expected sensitive header to be redacted")
	}
	if !strings.Contains(buf.String(), "REDACTED") {
		t.Error("expected REDACTED marker")
	}
}

func TestFull(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	// Can't easily test Full() without setting logger, so test the options
	opts := Options{
		Logger:     logger,
		LogHeaders: true,
		LogBody:    true,
	}

	app := mizu.NewRouter()
	app.Use(WithOptions(opts))

	app.Post("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("body"))
	req.Header.Set("X-Test", "value")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !strings.Contains(buf.String(), "headers") {
		t.Error("expected headers in full log")
	}
}

func TestQueryParams(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	app := mizu.NewRouter()
	app.Use(WithLogger(logger))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/?foo=bar&baz=qux", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !strings.Contains(buf.String(), "query") {
		t.Error("expected query params in log")
	}
}

func TestBodyPreserved(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Logger:  logger,
		LogBody: true,
	}))

	var bodyRead string
	app.Post("/", func(c *mizu.Ctx) error {
		body := make([]byte, 100)
		n, _ := c.Request().Body.Read(body)
		bodyRead = string(body[:n])
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("preserved body"))
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if bodyRead != "preserved body" {
		t.Errorf("expected body to be preserved, got %q", bodyRead)
	}
}
