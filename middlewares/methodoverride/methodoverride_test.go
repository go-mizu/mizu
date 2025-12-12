package methodoverride

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	var capturedMethod string
	handler := func(c *mizu.Ctx) error {
		capturedMethod = c.Request().Method
		return c.Text(http.StatusOK, "ok")
	}

	app.Post("/test", handler)
	app.Put("/test", handler)
	app.Patch("/test", handler)
	app.Delete("/test", handler)

	t.Run("overrides via header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		req.Header.Set("X-Http-Method-Override", "PUT")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if capturedMethod != http.MethodPut {
			t.Errorf("expected PUT, got %s", capturedMethod)
		}
	})

	t.Run("overrides via query", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/test?_method=DELETE", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if capturedMethod != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", capturedMethod)
		}
	})

	t.Run("overrides via form", func(t *testing.T) {
		form := url.Values{}
		form.Set("_method", "PATCH")
		req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if capturedMethod != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", capturedMethod)
		}
	})

	t.Run("ignores non-POST", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Http-Method-Override", "DELETE")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		// Should remain GET (and likely 404 or whatever default)
	})

	t.Run("ignores invalid method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		req.Header.Set("X-Http-Method-Override", "INVALID")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if capturedMethod != http.MethodPost {
			t.Errorf("expected POST (unchanged), got %s", capturedMethod)
		}
	})
}

func TestWithOptions_CustomHeader(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Header: "X-Method",
	}))

	var capturedMethod string
	app.Post("/test", func(c *mizu.Ctx) error {
		capturedMethod = c.Request().Method
		return c.Text(http.StatusOK, "ok")
	})
	app.Put("/test", func(c *mizu.Ctx) error {
		capturedMethod = c.Request().Method
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("X-Method", "PUT")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if capturedMethod != http.MethodPut {
		t.Errorf("expected PUT, got %s", capturedMethod)
	}
}

func TestWithOptions_CustomMethods(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Methods: []string{"GET", "HEAD"},
	}))

	var capturedMethod string
	app.Post("/test", func(c *mizu.Ctx) error {
		capturedMethod = c.Request().Method
		return c.Text(http.StatusOK, "ok")
	})
	app.Get("/test", func(c *mizu.Ctx) error {
		capturedMethod = c.Request().Method
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("allows custom method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		req.Header.Set("X-Http-Method-Override", "GET")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if capturedMethod != http.MethodGet {
			t.Errorf("expected GET, got %s", capturedMethod)
		}
	})

	t.Run("blocks non-allowed method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		req.Header.Set("X-Http-Method-Override", "DELETE")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if capturedMethod != http.MethodPost {
			t.Errorf("expected POST (unchanged), got %s", capturedMethod)
		}
	})
}

func TestWithOptions_CaseInsensitive(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	var capturedMethod string
	app.Post("/test", func(c *mizu.Ctx) error {
		capturedMethod = c.Request().Method
		return c.Text(http.StatusOK, "ok")
	})
	app.Delete("/test", func(c *mizu.Ctx) error {
		capturedMethod = c.Request().Method
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("X-Http-Method-Override", "delete") // lowercase
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if capturedMethod != http.MethodDelete {
		t.Errorf("expected DELETE, got %s", capturedMethod)
	}
}
