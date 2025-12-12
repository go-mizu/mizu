package header

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(map[string]string{
		"X-Custom":  "value",
		"X-Another": "another",
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("X-Custom") != "value" {
		t.Errorf("expected X-Custom header 'value', got %q", rec.Header().Get("X-Custom"))
	}
	if rec.Header().Get("X-Another") != "another" {
		t.Errorf("expected X-Another header 'another', got %q", rec.Header().Get("X-Another"))
	}
}

func TestWithOptions_RequestHeaders(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Request: map[string]string{
			"X-Injected": "injected",
		},
	}))

	var capturedHeader string
	app.Get("/", func(c *mizu.Ctx) error {
		capturedHeader = c.Request().Header.Get("X-Injected")
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if capturedHeader != "injected" {
		t.Errorf("expected injected header, got %q", capturedHeader)
	}
}

func TestWithOptions_RemoveHeaders(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		RequestRemove:  []string{"X-Remove-Me"},
		ResponseRemove: []string{"X-Server"},
	}))

	var removedFromRequest bool
	app.Get("/", func(c *mizu.Ctx) error {
		removedFromRequest = c.Request().Header.Get("X-Remove-Me") == ""
		c.Writer().Header().Set("X-Server", "secret")
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Remove-Me", "should be removed")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !removedFromRequest {
		t.Error("expected request header to be removed")
	}
	if rec.Header().Get("X-Server") != "" {
		t.Error("expected response header to be removed")
	}
}

func TestSet(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Set("X-Test", "value"))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("X-Test") != "value" {
		t.Errorf("expected 'value', got %q", rec.Header().Get("X-Test"))
	}
}

func TestSetRequest(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(SetRequest("X-Request-Custom", "custom"))

	var capturedHeader string
	app.Get("/", func(c *mizu.Ctx) error {
		capturedHeader = c.Request().Header.Get("X-Request-Custom")
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if capturedHeader != "custom" {
		t.Errorf("expected 'custom', got %q", capturedHeader)
	}
}

func TestRemove(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Remove("Server", "X-Powered-By"))

	app.Get("/", func(c *mizu.Ctx) error {
		c.Writer().Header().Set("Server", "MyServer")
		c.Writer().Header().Set("X-Powered-By", "Go")
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Server") != "" {
		t.Error("expected Server header to be removed")
	}
	if rec.Header().Get("X-Powered-By") != "" {
		t.Error("expected X-Powered-By header to be removed")
	}
}

func TestRemoveRequest(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(RemoveRequest("Cookie"))

	var hasCookie bool
	app.Get("/", func(c *mizu.Ctx) error {
		hasCookie = c.Request().Header.Get("Cookie") != ""
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Cookie", "session=abc")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if hasCookie {
		t.Error("expected Cookie header to be removed")
	}
}

func TestXSSProtection(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(XSSProtection())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("X-XSS-Protection") != "1; mode=block" {
		t.Errorf("expected XSS protection header, got %q", rec.Header().Get("X-XSS-Protection"))
	}
}

func TestNoSniff(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(NoSniff())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("expected nosniff header")
	}
}

func TestFrameDeny(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(FrameDeny())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("X-Frame-Options") != "DENY" {
		t.Error("expected frame deny header")
	}
}

func TestFrameSameOrigin(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(FrameSameOrigin())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("X-Frame-Options") != "SAMEORIGIN" {
		t.Error("expected frame sameorigin header")
	}
}

func TestHSTS(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(HSTS(31536000, true, true))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	expected := "max-age=31536000; includeSubDomains; preload"
	if rec.Header().Get("Strict-Transport-Security") != expected {
		t.Errorf("expected %q, got %q", expected, rec.Header().Get("Strict-Transport-Security"))
	}
}

func TestCSP(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(CSP("default-src 'self'"))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Security-Policy") != "default-src 'self'" {
		t.Error("expected CSP header")
	}
}

func TestReferrerPolicy(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(ReferrerPolicy("strict-origin"))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Referrer-Policy") != "strict-origin" {
		t.Error("expected referrer policy header")
	}
}

func TestJSON(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(JSON())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "{}")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Errorf("expected JSON content type, got %q", rec.Header().Get("Content-Type"))
	}
}

func TestHTML(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(HTML())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "<html></html>")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Errorf("expected HTML content type, got %q", rec.Header().Get("Content-Type"))
	}
}

func TestText(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Text())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "hello")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Type") != "text/plain; charset=utf-8" {
		t.Errorf("expected text content type, got %q", rec.Header().Get("Content-Type"))
	}
}

func TestXML(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(XML())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "<xml></xml>")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Type") != "application/xml; charset=utf-8" {
		t.Errorf("expected XML content type, got %q", rec.Header().Get("Content-Type"))
	}
}
