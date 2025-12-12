package responselog

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

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "response body")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !strings.Contains(buf.String(), "status=200") {
		t.Errorf("expected status in log, got %q", buf.String())
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

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "test response")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !strings.Contains(buf.String(), "test response") {
		t.Errorf("expected body in log, got %q", buf.String())
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
		c.Header().Set("X-Custom", "value")
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !strings.Contains(buf.String(), "headers") {
		t.Error("expected headers in log")
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

func TestWithOptions_SkipStatuses(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Logger:       logger,
		SkipStatuses: []int{200},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if strings.Contains(buf.String(), "status=200") {
		t.Error("expected 200 to be skipped")
	}
}

func TestWithOptions_MaxBodySize(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Logger:      logger,
		LogBody:     true,
		MaxBodySize: 5,
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "this is a longer response")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Body should be truncated to 5 chars
	if strings.Contains(buf.String(), "longer") {
		t.Error("expected body to be truncated")
	}
}

func TestErrorsOnly(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	// Can't easily test ErrorsOnly() without setting logger
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Logger:       logger,
		LogBody:      true,
		SkipStatuses: []int{200},
	}))

	app.Get("/ok", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})
	app.Get("/error", func(c *mizu.Ctx) error {
		return c.Text(http.StatusInternalServerError, "error")
	})

	// OK response should be skipped
	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if strings.Contains(buf.String(), "/ok") {
		t.Error("expected /ok to be skipped")
	}

	// Error response should be logged
	req = httptest.NewRequest(http.MethodGet, "/error", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !strings.Contains(buf.String(), "/error") {
		t.Error("expected /error to be logged")
	}
}

func TestDurationLogged(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	app := mizu.NewRouter()
	app.Use(WithLogger(logger))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !strings.Contains(buf.String(), "duration") {
		t.Error("expected duration in log")
	}
}

func TestSizeLogged(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	app := mizu.NewRouter()
	app.Use(WithLogger(logger))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "12345")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !strings.Contains(buf.String(), "size=5") {
		t.Errorf("expected size in log, got %q", buf.String())
	}
}
