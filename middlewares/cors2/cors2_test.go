package cors2

import (
	"net/http"
	"net/http/httptest"
	"testing"

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

	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("expected wildcard origin, got %q", rec.Header().Get("Access-Control-Allow-Origin"))
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

		if rec.Header().Get("Access-Control-Allow-Origin") != "http://example.com" {
			t.Errorf("expected specific origin, got %q", rec.Header().Get("Access-Control-Allow-Origin"))
		}
	})

	t.Run("non-matching origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Origin", "http://other.com")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Errorf("expected no origin header, got %q", rec.Header().Get("Access-Control-Allow-Origin"))
		}
	})
}

func TestPreflight(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Post("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected %d, got %d", http.StatusNoContent, rec.Code)
	}

	if rec.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("expected Allow-Methods header")
	}
	if rec.Header().Get("Access-Control-Allow-Headers") == "" {
		t.Error("expected Allow-Headers header")
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

	if rec.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Error("expected credentials header")
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

	if rec.Header().Get("Access-Control-Expose-Headers") != "X-Custom-Header, X-Another" {
		t.Errorf("expected expose headers, got %q", rec.Header().Get("Access-Control-Expose-Headers"))
	}
}

func TestWithOptions_MaxAge(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		MaxAge: 3600 * 1e9, // 1 hour in nanoseconds
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Max-Age") != "3600" {
		t.Errorf("expected max-age 3600, got %q", rec.Header().Get("Access-Control-Max-Age"))
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

	if rec.Header().Get("Access-Control-Allow-Origin") != "http://allowed.com" {
		t.Errorf("expected allowed origin, got %q", rec.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestAllowAll(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(AllowAll())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "http://any.com")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("expected wildcard origin")
	}
	if rec.Header().Get("Access-Control-Max-Age") == "" {
		t.Error("expected max-age header")
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

	if rec.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Error("expected credentials to be allowed")
	}
}
