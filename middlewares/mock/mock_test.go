package mock

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(map[string]*Response{
		"/api/users": JSON(http.StatusOK, []string{"user1", "user2"}),
	}))

	app.Get("/api/users", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "real response")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "application/json") {
		t.Error("expected JSON content type")
	}
}

func TestMock_Register(t *testing.T) {
	m := NewMock()
	m.Register("/test", Text(http.StatusOK, "mocked"))

	app := mizu.NewRouter()
	app.Use(m.Middleware())

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "real")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "mocked" {
		t.Errorf("expected 'mocked', got %q", rec.Body.String())
	}
}

func TestMock_RegisterMethod(t *testing.T) {
	m := NewMock()
	m.RegisterMethod(http.MethodPost, "/submit", Text(http.StatusCreated, "created"))

	app := mizu.NewRouter()
	app.Use(m.Middleware())

	app.Post("/submit", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "real")
	})

	req := httptest.NewRequest(http.MethodPost, "/submit", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected %d, got %d", http.StatusCreated, rec.Code)
	}
}

func TestMock_Clear(t *testing.T) {
	m := NewMock()
	m.Register("/test", Text(http.StatusOK, "mocked"))
	m.Clear()

	app := mizu.NewRouter()
	app.Use(m.Middleware())

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "real")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "real" {
		t.Errorf("expected 'real' after clear, got %q", rec.Body.String())
	}
}

func TestWithOptions_DefaultResponse(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		DefaultResponse: Text(http.StatusServiceUnavailable, "maintenance"),
		Passthrough:     false,
	}))

	app.Get("/anything", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "real")
	})

	req := httptest.NewRequest(http.MethodGet, "/anything", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}
}

func TestJSON(t *testing.T) {
	resp := JSON(http.StatusOK, map[string]string{"key": "value"})

	if resp.Status != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.Status)
	}
	if resp.Headers["Content-Type"] != "application/json" {
		t.Error("expected JSON content type")
	}
}

func TestText(t *testing.T) {
	resp := Text(http.StatusOK, "hello")

	if string(resp.Body) != "hello" {
		t.Errorf("expected 'hello', got %q", resp.Body)
	}
	if resp.Headers["Content-Type"] != "text/plain" {
		t.Error("expected text/plain content type")
	}
}

func TestHTML(t *testing.T) {
	resp := HTML(http.StatusOK, "<h1>Hello</h1>")

	if resp.Headers["Content-Type"] != "text/html" {
		t.Error("expected text/html content type")
	}
}

func TestRedirect(t *testing.T) {
	resp := Redirect("/new-location", http.StatusFound)

	if resp.Status != http.StatusFound {
		t.Errorf("expected %d, got %d", http.StatusFound, resp.Status)
	}
	if resp.Headers["Location"] != "/new-location" {
		t.Error("expected Location header")
	}
}

func TestError(t *testing.T) {
	resp := Error(http.StatusBadRequest, "invalid input")

	if resp.Status != http.StatusBadRequest {
		t.Errorf("expected %d, got %d", http.StatusBadRequest, resp.Status)
	}
}

func TestPrefix(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Prefix("/api/v2", Error(http.StatusNotImplemented, "V2 not implemented")))

	app.Get("/api/v2/users", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "real")
	})
	app.Get("/api/v1/users", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "v1 real")
	})

	t.Run("prefixed path", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v2/users", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotImplemented {
			t.Errorf("expected %d, got %d", http.StatusNotImplemented, rec.Code)
		}
	})

	t.Run("non-prefixed path", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
	})
}

func TestPassthrough(t *testing.T) {
	m := NewMock()
	m.opts.Passthrough = true

	app := mizu.NewRouter()
	app.Use(m.Middleware())

	app.Get("/real", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "real response")
	})

	req := httptest.NewRequest(http.MethodGet, "/real", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "real response" {
		t.Errorf("expected passthrough, got %q", rec.Body.String())
	}
}
