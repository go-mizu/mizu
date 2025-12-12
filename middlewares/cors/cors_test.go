package cors

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(Options{
		AllowOrigins: []string{"http://example.com"},
		AllowMethods: []string{"GET", "POST"},
		AllowHeaders: []string{"Content-Type"},
	}))

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("allows matching origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "http://example.com")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
		if rec.Header().Get("Access-Control-Allow-Origin") != "http://example.com" {
			t.Error("expected Access-Control-Allow-Origin header")
		}
	})

	t.Run("ignores non-matching origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "http://other.com")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Error("should not set Access-Control-Allow-Origin for non-matching origin")
		}
	})

	t.Run("handles preflight", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/test", nil)
		req.Header.Set("Origin", "http://example.com")
		req.Header.Set("Access-Control-Request-Method", "POST")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Errorf("expected status %d, got %d", http.StatusNoContent, rec.Code)
		}
		if rec.Header().Get("Access-Control-Allow-Methods") == "" {
			t.Error("expected Access-Control-Allow-Methods header")
		}
	})

	t.Run("ignores requests without origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Error("should not set CORS headers without Origin")
		}
	})
}

func TestAllowAll(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(AllowAll())

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://any-origin.com")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("expected Access-Control-Allow-Origin: *")
	}
}

func TestWithOrigins(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOrigins("http://a.com", "http://b.com"))

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	tests := []struct {
		origin   string
		expected string
	}{
		{"http://a.com", "http://a.com"},
		{"http://b.com", "http://b.com"},
		{"http://c.com", ""},
	}

	for _, tt := range tests {
		t.Run(tt.origin, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Origin", tt.origin)
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)

			got := rec.Header().Get("Access-Control-Allow-Origin")
			if got != tt.expected {
				t.Errorf("origin %q: expected %q, got %q", tt.origin, tt.expected, got)
			}
		})
	}
}

func TestNew_AllowCredentials(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(Options{
		AllowOrigins:     []string{"http://example.com"},
		AllowCredentials: true,
	}))

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Error("expected Access-Control-Allow-Credentials: true")
	}

	// Should set specific origin, not wildcard
	if rec.Header().Get("Access-Control-Allow-Origin") != "http://example.com" {
		t.Error("should set specific origin when credentials allowed")
	}
}

func TestNew_ExposeHeaders(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(Options{
		AllowOrigins:  []string{"*"},
		ExposeHeaders: []string{"X-Custom-Header", "X-Another"},
	}))

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	exposed := rec.Header().Get("Access-Control-Expose-Headers")
	if exposed != "X-Custom-Header, X-Another" {
		t.Errorf("expected exposed headers, got %q", exposed)
	}
}

func TestNew_MaxAge(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(Options{
		AllowOrigins: []string{"*"},
		MaxAge:       12 * time.Hour,
	}))

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	maxAge := rec.Header().Get("Access-Control-Max-Age")
	if maxAge != "43200" {
		t.Errorf("expected max-age 43200, got %q", maxAge)
	}
}

func TestNew_AllowOriginFunc(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(Options{
		AllowOriginFunc: func(origin string) bool {
			// Allow all subdomains of example.com
			return origin == "http://example.com" ||
				strings.HasSuffix(origin, ".example.com")
		},
	}))

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	tests := []struct {
		origin  string
		allowed bool
	}{
		{"http://example.com", true},
		{"http://api.example.com", true},
		{"http://other.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.origin, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Origin", tt.origin)
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)

			got := rec.Header().Get("Access-Control-Allow-Origin")
			if tt.allowed && got != tt.origin {
				t.Errorf("expected origin %q to be allowed", tt.origin)
			}
			if !tt.allowed && got != "" {
				t.Errorf("expected origin %q to be denied", tt.origin)
			}
		})
	}
}

func TestNew_PrivateNetwork(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(Options{
		AllowOrigins:        []string{"*"},
		AllowPrivateNetwork: true,
	}))

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Access-Control-Request-Private-Network", "true")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Private-Network") != "true" {
		t.Error("expected Access-Control-Allow-Private-Network: true")
	}
}

func TestNew_VaryHeader(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(Options{
		AllowOrigins: []string{"http://example.com"},
	}))

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	vary := rec.Header().Get("Vary")
	if vary != "Origin" {
		t.Errorf("expected Vary: Origin, got %q", vary)
	}
}
