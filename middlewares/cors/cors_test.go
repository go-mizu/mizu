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

	app.Get("/test", func(c *mizu.Ctx) error { return c.Text(http.StatusOK, "ok") })
	app.Post("/test", func(c *mizu.Ctx) error { return c.Text(http.StatusOK, "ok") })
	app.Options("/test", func(c *mizu.Ctx) error { return c.NoContent() })

	t.Run("allows matching origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "http://example.com")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://example.com" {
			t.Fatalf("expected Access-Control-Allow-Origin %q, got %q", "http://example.com", got)
		}
		vary := strings.Join(rec.Header().Values("Vary"), ",")
		if !containsVary(vary, "Origin") {
			t.Fatalf("expected Vary to contain Origin, got %q", vary)
		}
	})

	t.Run("ignores non-matching origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "http://other.com")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Fatalf("should not set Access-Control-Allow-Origin for non-matching origin, got %q", got)
		}
	})

	t.Run("handles preflight", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/test", nil)
		req.Header.Set("Origin", "http://example.com")
		req.Header.Set("Access-Control-Request-Method", "POST")
		req.Header.Set("Access-Control-Request-Headers", "Content-Type")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
		}
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://example.com" {
			t.Fatalf("expected Access-Control-Allow-Origin %q, got %q", "http://example.com", got)
		}
		if got := rec.Header().Get("Access-Control-Allow-Methods"); got == "" {
			t.Fatalf("expected Access-Control-Allow-Methods header")
		}
		if got := rec.Header().Get("Access-Control-Allow-Headers"); got == "" {
			t.Fatalf("expected Access-Control-Allow-Headers header")
		}

		vary := strings.Join(rec.Header().Values("Vary"), ",")
		if !containsVary(vary, "Origin") ||
			!containsVary(vary, "Access-Control-Request-Method") ||
			!containsVary(vary, "Access-Control-Request-Headers") {
			t.Fatalf("expected Vary to contain Origin, Access-Control-Request-Method, Access-Control-Request-Headers; got %q", vary)
		}
	})

	t.Run("ignores OPTIONS without preflight headers", func(t *testing.T) {
		app2 := mizu.NewRouter()
		app2.Use(New(Options{
			AllowOrigins: []string{"http://example.com"},
		}))
		app2.Handle(http.MethodOptions, "/test", func(c *mizu.Ctx) error {
			return c.Text(http.StatusOK, "options-ok")
		})

		req := httptest.NewRequest(http.MethodOptions, "/test", nil)
		req.Header.Set("Origin", "http://example.com")
		rec := httptest.NewRecorder()
		app2.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("ignores requests without origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Fatalf("should not set CORS headers without Origin, got %q", got)
		}
	})
}

func TestAllowAll(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(AllowAll())

	app.Get("/test", func(c *mizu.Ctx) error { return c.Text(http.StatusOK, "ok") })

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://any-origin.com")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected Access-Control-Allow-Origin %q, got %q", "*", got)
	}
}

func TestAllowAll_PreflightReflectsHeaders(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(AllowAll())

	app.Get("/test", func(c *mizu.Ctx) error { return c.Text(http.StatusOK, "ok") })
	app.Options("/test", func(c *mizu.Ctx) error { return c.NoContent() })

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "http://any-origin.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "X-One, X-Two")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected %d, got %d", http.StatusNoContent, rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected Access-Control-Allow-Origin %q, got %q", "*", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got != "X-One, X-Two" {
		t.Fatalf("expected reflected allow headers %q, got %q", "X-One, X-Two", got)
	}
}

func TestWithOrigins(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOrigins("http://a.com", "http://b.com"))

	app.Get("/test", func(c *mizu.Ctx) error { return c.Text(http.StatusOK, "ok") })

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
				t.Fatalf("origin %q: expected %q, got %q", tt.origin, tt.expected, got)
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

	app.Get("/test", func(c *mizu.Ctx) error { return c.Text(http.StatusOK, "ok") })

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("expected Access-Control-Allow-Credentials %q, got %q", "true", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://example.com" {
		t.Fatalf("expected specific origin when credentials allowed, got %q", got)
	}
	vary := strings.Join(rec.Header().Values("Vary"), ",")
	if !containsVary(vary, "Origin") {
		t.Fatalf("expected Vary to contain Origin, got %q", vary)
	}
}

func TestNew_WildcardWithCredentialsEchoesOrigin(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(Options{
		AllowOrigins:     []string{"*"},
		AllowCredentials: true,
	}))

	app.Get("/test", func(c *mizu.Ctx) error { return c.Text(http.StatusOK, "ok") })

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://example.com" {
		t.Fatalf("expected echoed origin, got %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("expected credentials true, got %q", got)
	}
}

func TestNew_ExposeHeaders(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(Options{
		AllowOrigins:  []string{"*"},
		ExposeHeaders: []string{"X-Custom-Header", "X-Another"},
	}))

	app.Get("/test", func(c *mizu.Ctx) error { return c.Text(http.StatusOK, "ok") })

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	exposed := rec.Header().Get("Access-Control-Expose-Headers")
	if exposed != "X-Custom-Header, X-Another" {
		t.Fatalf("expected exposed headers %q, got %q", "X-Custom-Header, X-Another", exposed)
	}
}

func TestNew_MaxAge(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(Options{
		AllowOrigins: []string{"*"},
		MaxAge:       12 * time.Hour,
	}))

	app.Get("/test", func(c *mizu.Ctx) error { return c.Text(http.StatusOK, "ok") })
	app.Options("/test", func(c *mizu.Ctx) error { return c.NoContent() })

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	maxAge := rec.Header().Get("Access-Control-Max-Age")
	if maxAge != "43200" {
		t.Fatalf("expected max-age %q, got %q", "43200", maxAge)
	}
}

func TestNew_AllowOriginFunc(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(Options{
		AllowOriginFunc: func(origin string) bool {
			return origin == "http://example.com" || strings.HasSuffix(origin, ".example.com")
		},
	}))

	app.Get("/test", func(c *mizu.Ctx) error { return c.Text(http.StatusOK, "ok") })

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
				t.Fatalf("expected origin %q to be allowed, got %q", tt.origin, got)
			}
			if !tt.allowed && got != "" {
				t.Fatalf("expected origin %q to be denied, got %q", tt.origin, got)
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

	app.Get("/test", func(c *mizu.Ctx) error { return c.Text(http.StatusOK, "ok") })
	app.Options("/test", func(c *mizu.Ctx) error { return c.NoContent() })

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	req.Header.Set("Access-Control-Request-Private-Network", "true")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Private-Network"); got != "true" {
		t.Fatalf("expected Access-Control-Allow-Private-Network %q, got %q", "true", got)
	}
	vary := strings.Join(rec.Header().Values("Vary"), ",")
	if !containsVary(vary, "Access-Control-Request-Private-Network") {
		t.Fatalf("expected Vary to contain Access-Control-Request-Private-Network, got %q", vary)
	}
}

func TestNew_VaryHeader(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(Options{
		AllowOrigins: []string{"http://example.com"},
	}))

	app.Get("/test", func(c *mizu.Ctx) error { return c.Text(http.StatusOK, "ok") })

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	vary := strings.Join(rec.Header().Values("Vary"), ",")
	if !containsVary(vary, "Origin") {
		t.Fatalf("expected Vary to contain Origin, got %q", vary)
	}
}

func containsVary(vary, token string) bool {
	for _, part := range strings.Split(vary, ",") {
		if strings.EqualFold(strings.TrimSpace(part), token) {
			return true
		}
	}
	return false
}
