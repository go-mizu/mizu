package cors2

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
	app.Use(New())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected wildcard origin, got %q", got)
	}
}

func TestWithOptions_SpecificOrigin(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{Origin: "http://example.com"}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("matching origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Origin", "http://example.com")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://example.com" {
			t.Fatalf("expected specific origin, got %q", got)
		}
		vary := getVary(rec.Header())
		if !containsVary(vary, "Origin") {
			t.Fatalf("expected Vary to contain Origin, got %q", vary)
		}
	})

	t.Run("non-matching origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Origin", "http://other.com")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Fatalf("expected no origin header, got %q", got)
		}
	})
}

func TestPreflight(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Post("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})
	app.Options("/", func(c *mizu.Ctx) error {
		return c.NoContent()
	})

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected %d, got %d", http.StatusNoContent, rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Fatalf("expected Allow-Methods header")
	}
	if rec.Header().Get("Access-Control-Allow-Headers") == "" {
		t.Fatalf("expected Allow-Headers header")
	}

	vary := getVary(rec.Header())
	if !containsVary(vary, "Access-Control-Request-Method") {
		t.Fatalf("expected Vary to contain Access-Control-Request-Method, got %q", vary)
	}
	if !containsVary(vary, "Access-Control-Request-Headers") {
		t.Fatalf("expected Vary to contain Access-Control-Request-Headers, got %q", vary)
	}
}

func TestWithOptions_Credentials(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Origin:      "http://example.com",
		Credentials: true,
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://example.com" {
		t.Fatalf("expected allowed origin, got %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("expected credentials header, got %q", got)
	}

	vary := getVary(rec.Header())
	if !containsVary(vary, "Origin") {
		t.Fatalf("expected Vary to contain Origin, got %q", vary)
	}
}

func TestWithOptions_WildcardWithCredentialsEchoesOrigin(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Origin:      "*",
		Credentials: true,
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://example.com" {
		t.Fatalf("expected echoed origin, got %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("expected credentials header, got %q", got)
	}

	vary := getVary(rec.Header())
	if !containsVary(vary, "Origin") {
		t.Fatalf("expected Vary to contain Origin, got %q", vary)
	}
}

func TestWithOptions_ExposeHeaders(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		ExposeHeaders: "X-Custom-Header, X-Another",
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Expose-Headers"); got != "X-Custom-Header, X-Another" {
		t.Fatalf("expected expose headers, got %q", got)
	}
}

func TestWithOptions_MaxAge(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		MaxAge: 1 * time.Hour,
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})
	app.Options("/", func(c *mizu.Ctx) error {
		return c.NoContent()
	})

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Max-Age"); got != "3600" {
		t.Fatalf("expected max-age 3600, got %q", got)
	}
}

func TestAllowOrigin(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(AllowOrigin("http://allowed.com"))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://allowed.com")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://allowed.com" {
		t.Fatalf("expected allowed origin, got %q", got)
	}
}

func TestAllowAll_Preflight(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(AllowAll())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})
	app.Options("/", func(c *mizu.Ctx) error {
		return c.NoContent()
	})

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "http://any.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	req.Header.Set("Access-Control-Request-Headers", "X-One, X-Two")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected %d, got %d", http.StatusNoContent, rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected wildcard origin, got %q", got)
	}
	if rec.Header().Get("Access-Control-Max-Age") == "" {
		t.Fatalf("expected max-age header")
	}

	vary := getVary(rec.Header())
	if !containsVary(vary, "Access-Control-Request-Method") {
		t.Fatalf("expected Vary to contain Access-Control-Request-Method, got %q", vary)
	}
	if !containsVary(vary, "Access-Control-Request-Headers") {
		t.Fatalf("expected Vary to contain Access-Control-Request-Headers, got %q", vary)
	}
}

func TestAllowCredentials(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(AllowCredentials("http://trusted.com"))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://trusted.com")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://trusted.com" {
		t.Fatalf("expected allowed origin, got %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("expected credentials to be allowed, got %q", got)
	}

	vary := getVary(rec.Header())
	if !containsVary(vary, "Origin") {
		t.Fatalf("expected Vary to contain Origin, got %q", vary)
	}
}

func getVary(h http.Header) string {
	return strings.Join(h.Values("Vary"), ",")
}

func containsVary(vary, token string) bool {
	for _, part := range strings.Split(vary, ",") {
		if strings.EqualFold(strings.TrimSpace(part), token) {
			return true
		}
	}
	return false
}
