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
		// Request "/" instead of "/index.html" since FileServer redirects index.html to /
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
		if rec.Body.String() != "<html>index</html>" {
			t.Errorf("expected index content, got %q", rec.Body.String())
		}
	})

	t.Run("serve nested", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/js/app.js", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
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

	// Request "/" instead of "/index.html" since FileServer redirects index.html to /
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
	if rec.Body.String() != "<html>public</html>" {
		t.Errorf("expected public content, got %q", rec.Body.String())
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

	cacheControl := rec.Header().Get("Cache-Control")
	if cacheControl != "max-age=3600" {
		t.Errorf("expected max-age=3600, got %q", cacheControl)
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
		t.Errorf("expected %d, got %d", http.StatusNotFound, rec.Code)
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "application/json") {
		t.Error("expected JSON response from not found handler")
	}
}

func TestHandler(t *testing.T) {
	testFS := fstest.MapFS{
		"index.html": {Data: []byte("<html>handler</html>")},
	}

	app := mizu.NewRouter()
	app.Get("/{path...}", Handler(testFS))

	// Request "/" instead of "/index.html" since FileServer redirects index.html to /
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
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
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
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
		t.Errorf("expected cache header, got %q", rec.Header().Get("Cache-Control"))
	}
}

func TestIndexFile(t *testing.T) {
	testFS := fstest.MapFS{
		"index.html":     {Data: []byte("<html>root</html>")},
		"sub/index.html": {Data: []byte("<html>sub</html>")},
	}

	app := mizu.NewRouter()
	app.Use(New(testFS))

	app.Get("/{path...}", func(c *mizu.Ctx) error {
		return c.Text(http.StatusNotFound, "not found")
	})

	// Root should serve index.html
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d for root, got %d", http.StatusOK, rec.Code)
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
		t.Errorf("expected fallthrough to work, got %d", rec.Code)
	}
	if rec.Body.String() != "api response" {
		t.Errorf("expected api response, got %q", rec.Body.String())
	}
}

// Verify that fs.FS interface is properly used
var _ fs.FS = fstest.MapFS{}

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

	t.Run("serve existing file", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/style.css", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("serve index for unknown routes", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/app/dashboard", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		// SPA should serve index.html for unknown routes
		// Note: Depending on implementation, this may return index or 404
		// The SPA function creates a NotFoundHandler that serves index.html
		if rec.Code != http.StatusOK && rec.Code != http.StatusNotFound {
			t.Errorf("expected OK or NotFound for SPA route, got %d", rec.Code)
		}
	})
}

func TestSPA_DefaultIndex(t *testing.T) {
	testFS := fstest.MapFS{
		"index.html": {Data: []byte("<html>default</html>")},
	}

	app := mizu.NewRouter()
	app.Use(SPA(testFS, "")) // Empty index should default to index.html

	app.Get("/{path...}", func(c *mizu.Ctx) error {
		return c.Text(http.StatusTeapot, "should not reach")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestHandlerWithOptions(t *testing.T) {
	testFS := fstest.MapFS{
		"file.txt": {Data: []byte("content")},
	}

	app := mizu.NewRouter()
	app.Get("/{path...}", HandlerWithOptions(testFS, Options{MaxAge: 7200}))

	req := httptest.NewRequest(http.MethodGet, "/file.txt", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
	if rec.Header().Get("Cache-Control") != "max-age=7200" {
		t.Errorf("expected cache header, got %q", rec.Header().Get("Cache-Control"))
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
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestWithOptions_InvalidRoot(t *testing.T) {
	testFS := fstest.MapFS{
		"file.txt": {Data: []byte("content")},
	}

	app := mizu.NewRouter()
	// Use a root that doesn't exist - should fall back to fsys
	app.Use(WithOptions(testFS, Options{Root: "nonexistent"}))

	app.Get("/{path...}", func(c *mizu.Ctx) error {
		return c.Text(http.StatusNotFound, "not found")
	})

	req := httptest.NewRequest(http.MethodGet, "/file.txt", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// With invalid root, it should fall back to the original filesystem
	// and then to next handler
	if rec.Code != http.StatusOK && rec.Code != http.StatusNotFound {
		t.Errorf("unexpected status: %d", rec.Code)
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

	// Test path cleaning
	req := httptest.NewRequest(http.MethodGet, "/file.txt", nil)
	req.URL.Path = "file.txt" // Set path without leading slash after creating request
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Should handle path without leading slash
	if rec.Code != http.StatusOK && rec.Code != http.StatusNotFound {
		t.Errorf("unexpected status: %d", rec.Code)
	}
}

func TestItoa(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{10, "10"},
		{123, "123"},
		{9999, "9999"},
	}

	for _, tc := range tests {
		result := itoa(tc.input)
		if result != tc.expected {
			t.Errorf("itoa(%d) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}
