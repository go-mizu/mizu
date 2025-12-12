package contenttype

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestRequire(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Require("application/json", "text/plain"))

	app.Post("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("allows matching content type", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("allows with charset", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("rejects wrong content type", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("<xml>"))
		req.Header.Set("Content-Type", "application/xml")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnsupportedMediaType {
			t.Errorf("expected %d, got %d", http.StatusUnsupportedMediaType, rec.Code)
		}
	})

	t.Run("rejects missing content type", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("data"))
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnsupportedMediaType {
			t.Errorf("expected %d, got %d", http.StatusUnsupportedMediaType, rec.Code)
		}
	})

	t.Run("skips GET requests", func(t *testing.T) {
		app2 := mizu.NewRouter()
		app2.Use(Require("application/json"))
		app2.Get("/test", func(c *mizu.Ctx) error {
			return c.Text(http.StatusOK, "ok")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		app2.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("GET should be allowed without content-type")
		}
	})
}

func TestRequireJSON(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(RequireJSON())

	app.Post("/api", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("allows JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api", strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("rejects form", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api", strings.NewReader("a=b"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnsupportedMediaType {
			t.Errorf("expected %d, got %d", http.StatusUnsupportedMediaType, rec.Code)
		}
	})
}

func TestRequireForm(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(RequireForm())

	app.Post("/form", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("allows form-urlencoded", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/form", strings.NewReader("a=b"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("allows multipart", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/form", strings.NewReader(""))
		req.Header.Set("Content-Type", "multipart/form-data; boundary=----")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
	})
}

func TestDefault(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Default("application/json"))

	var capturedCT string
	app.Post("/test", func(c *mizu.Ctx) error {
		capturedCT = c.Request().Header.Get("Content-Type")
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("sets default when missing", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(""))
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if capturedCT != "application/json" {
			t.Errorf("expected 'application/json', got %q", capturedCT)
		}
	})

	t.Run("preserves existing", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(""))
		req.Header.Set("Content-Type", "text/plain")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if capturedCT != "text/plain" {
			t.Errorf("expected 'text/plain', got %q", capturedCT)
		}
	})
}

func TestSetResponse(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(SetResponse("application/json; charset=utf-8"))

	app.Get("/api", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, `{"ok":true}`)
	})

	req := httptest.NewRequest(http.MethodGet, "/api", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	ct := rec.Header().Get("Content-Type")
	if ct != "application/json; charset=utf-8" {
		t.Errorf("expected JSON content type, got %q", ct)
	}
}
