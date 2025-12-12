package transformer

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestAddHeader(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Request(AddHeader("X-Custom", "value")))

	app.Get("/", func(c *mizu.Ctx) error {
		val := c.Request().Header.Get("X-Custom")
		return c.Text(http.StatusOK, val)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "value" {
		t.Errorf("expected header value, got %q", rec.Body.String())
	}
}

func TestSetHeader(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Request(SetHeader("X-Custom", "new-value")))

	app.Get("/", func(c *mizu.Ctx) error {
		val := c.Request().Header.Get("X-Custom")
		return c.Text(http.StatusOK, val)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Custom", "old-value")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "new-value" {
		t.Errorf("expected new-value, got %q", rec.Body.String())
	}
}

func TestRemoveHeader(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Request(RemoveHeader("X-Remove")))

	app.Get("/", func(c *mizu.Ctx) error {
		val := c.Request().Header.Get("X-Remove")
		if val == "" {
			return c.Text(http.StatusOK, "removed")
		}
		return c.Text(http.StatusOK, "present")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Remove", "value")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "removed" {
		t.Errorf("expected header to be removed")
	}
}

func TestRewritePath(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Request(RewritePath("/old", "/new")))

	app.Get("/new/path", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, c.Request().URL.Path)
	})

	req := httptest.NewRequest(http.MethodGet, "/old/path", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "/new/path" {
		t.Errorf("expected /new/path, got %q", rec.Body.String())
	}
}

func TestAddQueryParam(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Request(AddQueryParam("added", "true")))

	app.Get("/", func(c *mizu.Ctx) error {
		val := c.Request().URL.Query().Get("added")
		return c.Text(http.StatusOK, val)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "true" {
		t.Errorf("expected true, got %q", rec.Body.String())
	}
}

func TestTransformBody(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Request(TransformBody(func(b []byte) ([]byte, error) {
		return bytes.ToUpper(b), nil
	})))

	app.Post("/", func(c *mizu.Ctx) error {
		body, _ := io.ReadAll(c.Request().Body)
		return c.Text(http.StatusOK, string(body))
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("hello"))
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "HELLO" {
		t.Errorf("expected HELLO, got %q", rec.Body.String())
	}
}

func TestAddResponseHeader(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Response(AddResponseHeader("X-Response", "value")))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("X-Response") != "value" {
		t.Errorf("expected response header, got %q", rec.Header().Get("X-Response"))
	}
}

func TestSetResponseHeader(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Response(SetResponseHeader("X-Override", "new")))

	app.Get("/", func(c *mizu.Ctx) error {
		c.Header().Set("X-Override", "old")
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("X-Override") != "new" {
		t.Errorf("expected new, got %q", rec.Header().Get("X-Override"))
	}
}

func TestRemoveResponseHeader(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Response(RemoveResponseHeader("X-Remove")))

	app.Get("/", func(c *mizu.Ctx) error {
		c.Header().Set("X-Remove", "value")
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("X-Remove") != "" {
		t.Error("expected header to be removed")
	}
}

func TestTransformResponseBody(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Response(TransformResponseBody(func(b []byte) ([]byte, error) {
		return bytes.ToUpper(b), nil
	})))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "hello")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "HELLO" {
		t.Errorf("expected HELLO, got %q", rec.Body.String())
	}
}

func TestMapStatusCode(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Response(MapStatusCode(http.StatusNotFound, http.StatusOK)))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusNotFound, "not found")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestReplaceBody(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Response(ReplaceBody(http.StatusNotFound, []byte("custom not found"))))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusNotFound, "original")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "custom not found" {
		t.Errorf("expected custom not found, got %q", rec.Body.String())
	}
}

func TestMultipleTransformers(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		RequestTransformers: []RequestTransformer{
			AddHeader("X-First", "1"),
			AddHeader("X-Second", "2"),
		},
		ResponseTransformers: []ResponseTransformer{
			AddResponseHeader("X-Resp-First", "a"),
			AddResponseHeader("X-Resp-Second", "b"),
		},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		first := c.Request().Header.Get("X-First")
		second := c.Request().Header.Get("X-Second")
		return c.Text(http.StatusOK, first+second)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "12" {
		t.Errorf("expected 12, got %q", rec.Body.String())
	}
	if rec.Header().Get("X-Resp-First") != "a" {
		t.Error("expected X-Resp-First header")
	}
	if rec.Header().Get("X-Resp-Second") != "b" {
		t.Error("expected X-Resp-Second header")
	}
}
