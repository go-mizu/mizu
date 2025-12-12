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

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "home")
	})

	t.Run("expvar endpoint", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/debug/vars", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "cmdline") {
			t.Error("expected expvar output with cmdline")
		}
	})

	t.Run("non-expvar path", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Body.String() != "home" {
			t.Errorf("expected 'home', got %q", rec.Body.String())
		}
	})
}

func TestWithOptions_CustomPath(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Path: "/metrics/vars",
	}))

	req := httptest.NewRequest(http.MethodGet, "/metrics/vars", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestHelpers(t *testing.T) {
	// Test NewInt
	counter := NewInt("test_counter")
	counter.Add(5)
	if counter.Value() != 5 {
		t.Errorf("expected 5, got %d", counter.Value())
	}

	// Test NewFloat
	gauge := NewFloat("test_gauge")
	gauge.Set(3.14)
	if gauge.Value() != 3.14 {
		t.Errorf("expected 3.14, got %f", gauge.Value())
	}

	// Test NewString
	status := NewString("test_status")
	status.Set("running")
	if status.Value() != "running" {
		t.Errorf("expected 'running', got %q", status.Value())
	}

	// Test NewMap
	m := NewMap("test_map")
	m.Add("requests", 100)
	if m.Get("requests").String() != "100" {
		t.Errorf("expected '100', got %q", m.Get("requests").String())
	}

	// Test Get
	if Get("test_counter") == nil {
		t.Error("expected to find test_counter")
	}

	// Test Do
	count := 0
	Do(func(kv KeyValue) {
		count++
	})
	if count == 0 {
		t.Error("expected at least one expvar")
	}
}

func TestJSON(t *testing.T) {
	json := JSON()
	if !strings.HasPrefix(json, "{") || !strings.HasSuffix(json, "}") {
		t.Error("expected valid JSON object")
	}
}

