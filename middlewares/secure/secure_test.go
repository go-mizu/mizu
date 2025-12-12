package secure

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

	t.Run("redirects to HTTPS", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusMovedPermanently {
			t.Errorf("expected %d, got %d", http.StatusMovedPermanently, rec.Code)
		}
		loc := rec.Header().Get("Location")
		if loc != "https://example.com/" {
			t.Errorf("expected https redirect, got %s", loc)
		}
	})
}

func TestWithOptions_NoSSLRedirect(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		SSLRedirect:        false,
		ContentTypeNosniff: true,
		FrameDeny:          true,
		XSSProtection:      "1; mode=block",
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}

	// Check security headers
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("expected X-Content-Type-Options: nosniff")
	}
	if rec.Header().Get("X-Frame-Options") != "DENY" {
		t.Error("expected X-Frame-Options: DENY")
	}
	if rec.Header().Get("X-XSS-Protection") != "1; mode=block" {
		t.Error("expected X-XSS-Protection header")
	}
}

func TestWithOptions_HSTS(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		SSLRedirect:          false,
		STSSeconds:           31536000,
		STSIncludeSubdomains: true,
		STSPreload:           true,
		ForceSTSHeader:       true,
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	sts := rec.Header().Get("Strict-Transport-Security")
	expected := "max-age=31536000; includeSubDomains; preload"
	if sts != expected {
		t.Errorf("expected %q, got %q", expected, sts)
	}
}

func TestWithOptions_CustomFrameOptions(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		SSLRedirect:        false,
		CustomFrameOptions: "SAMEORIGIN",
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("X-Frame-Options") != "SAMEORIGIN" {
		t.Error("expected X-Frame-Options: SAMEORIGIN")
	}
}

func TestWithOptions_CSP(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		SSLRedirect:           false,
		ContentSecurityPolicy: "default-src 'self'",
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Security-Policy") != "default-src 'self'" {
		t.Error("expected Content-Security-Policy header")
	}
}

func TestWithOptions_ReferrerPolicy(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		SSLRedirect:    false,
		ReferrerPolicy: "strict-origin-when-cross-origin",
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Referrer-Policy") != "strict-origin-when-cross-origin" {
		t.Error("expected Referrer-Policy header")
	}
}

func TestWithOptions_SSLHost(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		SSLRedirect:          true,
		SSLHost:              "secure.example.com",
		SSLTemporaryRedirect: true,
	}))

	app.Get("/path", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/path?q=1", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusTemporaryRedirect {
		t.Errorf("expected %d, got %d", http.StatusTemporaryRedirect, rec.Code)
	}
	if rec.Header().Get("Location") != "https://secure.example.com/path?q=1" {
		t.Errorf("unexpected redirect location: %s", rec.Header().Get("Location"))
	}
}

func TestWithOptions_ProxyHeader(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		SSLRedirect:        true,
		ContentTypeNosniff: true,
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Should not redirect since X-Forwarded-Proto is https
	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestWithOptions_IsDevelopment(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		SSLRedirect:   true,
		IsDevelopment: true,
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Should not redirect in development mode
	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}
