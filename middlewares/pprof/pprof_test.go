package pprof

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
		return c.Text(http.StatusOK, "home")
	})

	t.Run("index page", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "pprof") {
			t.Error("expected pprof index page")
		}
	})

	t.Run("cmdline", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/debug/pprof/cmdline", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("symbol", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/debug/pprof/symbol", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("heap profile", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/debug/pprof/heap", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("goroutine profile", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/debug/pprof/goroutine", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("non-pprof path", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Body.String() != "home" {
			t.Errorf("expected 'home', got %q", rec.Body.String())
		}
	})
}

func TestWithOptions_CustomPrefix(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Prefix: "/profiling",
	}))

	req := httptest.NewRequest(http.MethodGet, "/profiling/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestWithOptions_PrefixWithTrailingSlash(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Prefix: "/debug/pprof/",
	}))

	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/heap", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}
