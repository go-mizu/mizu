package keepalive

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

	if rec.Header().Get("Connection") != "keep-alive" {
		t.Errorf("expected keep-alive, got %q", rec.Header().Get("Connection"))
	}
	if rec.Header().Get("Keep-Alive") == "" {
		t.Error("expected Keep-Alive header")
	}
}

func TestWithOptions_CustomTimeout(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{Timeout: 120 * time.Second}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	keepAlive := rec.Header().Get("Keep-Alive")
	if keepAlive == "" {
		t.Fatal("expected Keep-Alive header")
	}
	// Should contain timeout=120
	expected := "timeout=120, max=100"
	if keepAlive != expected {
		t.Errorf("expected %q, got %q", expected, keepAlive)
	}
}

func TestWithOptions_CustomMax(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{MaxRequests: 500}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	keepAlive := rec.Header().Get("Keep-Alive")
	if keepAlive != "timeout=60, max=500" {
		t.Errorf("expected max=500, got %q", keepAlive)
	}
}

func TestWithOptions_Disable(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{DisableKeepAlive: true}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Connection") != "close" {
		t.Errorf("expected connection close, got %q", rec.Header().Get("Connection"))
	}
}

func TestWithOptions_ClientClose(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Connection", "close")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Connection") != "close" {
		t.Errorf("expected close when client requests, got %q", rec.Header().Get("Connection"))
	}
}

func TestDisable(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Disable())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Connection") != "close" {
		t.Error("expected disabled keep-alive")
	}
}

func TestWithTimeout(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithTimeout(30 * time.Second))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	keepAlive := rec.Header().Get("Keep-Alive")
	if keepAlive != "timeout=30, max=100" {
		t.Errorf("expected timeout=30, got %q", keepAlive)
	}
}

func TestWithMax(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithMax(200))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	keepAlive := rec.Header().Get("Keep-Alive")
	if keepAlive != "timeout=60, max=200" {
		t.Errorf("expected max=200, got %q", keepAlive)
	}
}
