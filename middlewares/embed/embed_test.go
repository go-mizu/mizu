package embed

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	testFS := fstest.MapFS{
		"index.html":     {Data: []byte("<html>index</html>")},
		"style.css":      {Data: []byte("body{}")},
		"js/app.js":      {Data: []byte("console.log('app')")},
		"sub/index.html": {Data: []byte("<html>sub</html>")},
	}

	app := mizu.NewRouter()
	app.Use(New(testFS))

	app.Get("/{path...}", func(c *mizu.Ctx) error {
		return c.Text(http.StatusNotFound, "not found")
	})

	t.Run("serve root", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
		}
		if rec.Body.String() != "<html>index</html>" {
			t.Fatalf("unexpected body %q", rec.Body.String())
		}
	})

	t.Run("serve nested", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/js/app.js", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
		}
	})
}

func TestWithOptions_Root(t *testing.T) {
	testFS := fstest.MapFS{
		"public/index.html": {Data: []byte("<html>public</html>")},
		"public/app.js":     {Data: []byte("app()")},
	}

	app := mizu.NewRouter()
	app.Use(WithOptions(testFS, Options{Root: "public"}))

	app.Get("/{path...}", func(c *mizu.Ctx) error {
		return c.Text(http.StatusNotFound, "not found")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
	if rec.Body.String() != "<html>public</html>" {
		t.Fatalf("unexpected body %q", rec.Body.String())
	}
}

func TestWithOptions_MaxAge(t *testing.T) {
	testFS := fstest.MapFS{
		"file.txt": {Data: []byte("content")},
	}

	app := mizu.NewRouter()
	app.Use(WithOptions(testFS, Options{MaxAge: 3600}))

	app.Get("/{path...}", func(c *mizu.Ctx) error {
		return c.Text(http.StatusNotFound, "not found")
	})

	req := httptest.NewRequest(http.MethodGet, "/file.txt", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Cache-Control") != "max-age=3600" {
		t.Fatalf("unexpected cache header %q", rec.Header().Get("Cache-Control"))
	}
}

func TestWithOptions_NotFoundHandler(t *testing.T) {
	testFS := fstest.MapFS{
		"existing.txt": {Data: []byte("exists")},
	}

	app := mizu.NewRouter()
	app.Use(WithOptions(testFS, Options{
		NotFoundHandler: func(c *mizu.Ctx) error {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "file not found"})
		},
	}))

	app.Get("/{path...}", func(c *mizu.Ctx) error {
		return c.Text(http.StatusNotFound, "fallback")
	})

	req := httptest.NewRequest(http.MethodGet, "/missing.txt", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected %d, got %d", http.StatusNotFound, rec.Code)
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("expected JSON content type")
	}
}

func TestHandler(t *testing.T) {
	testFS := fstest.MapFS{
		"index.html": {Data: []byte("<html>handler</html>")},
	}

	app := mizu.NewRouter()
	app.Get("/{path...}", Handler(testFS))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestStatic(t *testing.T) {
	testFS := fstest.MapFS{
		"assets/style.css": {Data: []byte("body{}")},
	}

	app := mizu.NewRouter()
	app.Use(Static(testFS, "assets"))

	app.Get("/{path...}", func(c *mizu.Ctx) error {
		return c.Text(http.StatusNotFound, "not found")
	})

	req := httptest.NewRequest(http.MethodGet, "/style.css", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestWithCaching(t *testing.T) {
	testFS := fstest.MapFS{
		"file.txt": {Data: []byte("content")},
	}

	app := mizu.NewRouter()
	app.Use(WithCaching(testFS, 86400))

	app.Get("/{path...}", func(c *mizu.Ctx) error {
		return c.Text(http.StatusNotFound, "not found")
	})

	req := httptest.NewRequest(http.MethodGet, "/file.txt", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Cache-Control") != "max-age=86400" {
		t.Fatalf("unexpected cache header %q", rec.Header().Get("Cache-Control"))
	}
}

func TestFallthrough(t *testing.T) {
	testFS := fstest.MapFS{}

	app := mizu.NewRouter()
	app.Use(New(testFS))

	app.Get("/api/data", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "api response")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
	if rec.Body.String() != "api response" {
		t.Fatalf("unexpected body %q", rec.Body.String())
	}
}

func TestSPA(t *testing.T) {
	testFS := fstest.MapFS{
		"index.html": {Data: []byte("<html>spa</html>")},
		"style.css":  {Data: []byte("body{}")},
	}

	app := mizu.NewRouter()
	app.Use(SPA(testFS, "index.html"))

	app.Get("/{path...}", func(c *mizu.Ctx) error {
		return c.Text(http.StatusTeapot, "should not reach")
	})

	t.Run("existing file", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/style.css", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("spa fallback", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/app/dashboard", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
		}
	})
}

func TestSPA_DefaultIndex(t *testing.T) {
	testFS := fstest.MapFS{
		"index.html": {Data: []byte("<html>default</html>")},
	}

	app := mizu.NewRouter()
	app.Use(SPA(testFS, ""))

	app.Get("/{path...}", func(c *mizu.Ctx) error {
		return c.Text(http.StatusTeapot, "should not reach")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestHandlerWithOptions_Root(t *testing.T) {
	testFS := fstest.MapFS{
		"static/file.txt": {Data: []byte("static content")},
	}

	app := mizu.NewRouter()
	app.Get("/{path...}", HandlerWithOptions(testFS, Options{Root: "static"}))

	req := httptest.NewRequest(http.MethodGet, "/file.txt", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestWithOptions_PathWithoutLeadingSlash(t *testing.T) {
	testFS := fstest.MapFS{
		"file.txt": {Data: []byte("content")},
	}

	app := mizu.NewRouter()
	app.Use(New(testFS))

	app.Get("/{path...}", func(c *mizu.Ctx) error {
		return c.Text(http.StatusNotFound, "not found")
	})

	req := httptest.NewRequest(http.MethodGet, "/file.txt", nil)
	req.URL.Path = "file.txt"
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestItoa(t *testing.T) {
	cases := []struct {
		in  int
		out string
	}{
		{0, "0"},
		{1, "1"},
		{10, "10"},
		{123, "123"},
		{9999, "9999"},
	}

	for _, c := range cases {
		if got := itoa(c.in); got != c.out {
			t.Fatalf("itoa(%d) = %q, want %q", c.in, got, c.out)
		}
	}
}

// compile-time check
var _ fs.FS = fstest.MapFS{}
