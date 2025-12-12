package version

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(Options{
		DefaultVersion: "v1",
	}))

	var capturedVersion string
	app.Get("/api/test", func(c *mizu.Ctx) error {
		capturedVersion = GetVersion(c)
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("from header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		req.Header.Set("Accept-Version", "v2")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if capturedVersion != "v2" {
			t.Errorf("expected v2, got %q", capturedVersion)
		}
	})

	t.Run("from query", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/test?version=v3", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if capturedVersion != "v3" {
			t.Errorf("expected v3, got %q", capturedVersion)
		}
	})

	t.Run("default version", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if capturedVersion != "v1" {
			t.Errorf("expected v1 (default), got %q", capturedVersion)
		}
	})
}

func TestNew_PathPrefix(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(Options{
		PathPrefix: true,
	}))

	var capturedVersion string
	app.Get("/v1/api/test", func(c *mizu.Ctx) error {
		capturedVersion = GetVersion(c)
		return c.Text(http.StatusOK, "ok")
	})
	app.Get("/v2/api/test", func(c *mizu.Ctx) error {
		capturedVersion = GetVersion(c)
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("v1", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/api/test", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if capturedVersion != "v1" {
			t.Errorf("expected v1, got %q", capturedVersion)
		}
	})

	t.Run("v2", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v2/api/test", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if capturedVersion != "v2" {
			t.Errorf("expected v2, got %q", capturedVersion)
		}
	})
}

func TestNew_Supported(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(Options{
		Supported: []string{"v1", "v2"},
	}))

	app.Get("/api/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("supported version", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		req.Header.Set("Accept-Version", "v1")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("unsupported version", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		req.Header.Set("Accept-Version", "v3")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})
}

func TestNew_Deprecated(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(Options{
		Deprecated: []string{"v1"},
	}))

	app.Get("/api/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Accept-Version", "v1")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Deprecation") != "true" {
		t.Error("expected Deprecation header")
	}
}

func TestFromHeader(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(FromHeader("X-API-Version"))

	var capturedVersion string
	app.Get("/test", func(c *mizu.Ctx) error {
		capturedVersion = GetVersion(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Api-Version", "2.0")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if capturedVersion != "2.0" {
		t.Errorf("expected 2.0, got %q", capturedVersion)
	}
}

func TestFromPath(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(FromPath())

	var capturedVersion string
	app.Get("/v1.2/test", func(c *mizu.Ctx) error {
		capturedVersion = GetVersion(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/v1.2/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if capturedVersion != "v1.2" {
		t.Errorf("expected v1.2, got %q", capturedVersion)
	}
}

func TestFromQuery(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(FromQuery("api_version"))

	var capturedVersion string
	app.Get("/test", func(c *mizu.Ctx) error {
		capturedVersion = GetVersion(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test?api_version=3.0", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if capturedVersion != "3.0" {
		t.Errorf("expected 3.0, got %q", capturedVersion)
	}
}

func TestGet(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(Options{DefaultVersion: "v1"}))

	var v1, v2 string
	app.Get("/test", func(c *mizu.Ctx) error {
		v1 = GetVersion(c)
		v2 = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if v1 != v2 {
		t.Errorf("GetVersion and Get should return same value")
	}
}

func TestIsVersionString(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"v1", true},
		{"v2", true},
		{"V1", true},
		{"v1.0", true},
		{"v1.2.3", true},
		{"api", false},
		{"", false},
		{"v", false},
		{"va", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := isVersionString(tt.input); got != tt.expected {
				t.Errorf("isVersionString(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}
