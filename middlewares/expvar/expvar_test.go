package expvar

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

	// "/" behaves like a catch-all in ServeMux, so middleware can intercept /debug/vars.
	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "home")
	})

	t.Run("expvar endpoint", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/debug/vars", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "cmdline") {
			t.Fatalf("expected expvar output with cmdline, got %q", rec.Body.String())
		}
	})

	t.Run("non-expvar path", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
		}
		if rec.Body.String() != "home" {
			t.Fatalf("expected 'home', got %q", rec.Body.String())
		}
	})
}

func TestWithOptions_CustomPath(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{Path: "/metrics/vars"}))

	// Ensure at least one route exists so middleware is applied.
	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/metrics/vars", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestHelpers(t *testing.T) {
	counter := NewInt("test_counter")
	counter.Add(5)
	if counter.Value() != 5 {
		t.Fatalf("expected 5, got %d", counter.Value())
	}

	gauge := NewFloat("test_gauge")
	gauge.Set(3.14)
	if gauge.Value() != 3.14 {
		t.Fatalf("expected 3.14, got %f", gauge.Value())
	}

	status := NewString("test_status")
	status.Set("running")
	if status.Value() != "running" {
		t.Fatalf("expected 'running', got %q", status.Value())
	}

	m := NewMap("test_map")
	m.Add("requests", 100)
	if m.Get("requests").String() != "100" {
		t.Fatalf("expected '100', got %q", m.Get("requests").String())
	}

	if Get("test_counter") == nil {
		t.Fatalf("expected to find test_counter")
	}

	count := 0
	Do(func(kv KeyValue) {
		count++
	})
	if count == 0 {
		t.Fatalf("expected at least one expvar")
	}
}

func TestJSON(t *testing.T) {
	s := JSON()
	if !strings.HasPrefix(s, "{") || !strings.HasSuffix(s, "}") {
		t.Fatalf("expected JSON object, got %q", s)
	}
}
