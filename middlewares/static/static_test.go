package static

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	// Create temp directory with test files
	tmpDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("hello"), 0600)                //nolint:gosec // G306: Test file
	_ = os.WriteFile(filepath.Join(tmpDir, "index.html"), []byte("<html>index</html>"), 0600) //nolint:gosec // G306: Test file

	app := mizu.NewRouter()
	app.Use(New(tmpDir))

	app.Get("/api", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "api")
	})

	t.Run("serve file", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test.txt", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
		if rec.Body.String() != "hello" {
			t.Errorf("expected 'hello', got %q", rec.Body.String())
		}
	})

	t.Run("serve index", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
		if rec.Body.String() != "<html>index</html>" {
			t.Errorf("expected index.html content, got %q", rec.Body.String())
		}
	})

	t.Run("file not found falls through", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Body.String() != "api" {
			t.Errorf("expected 'api', got %q", rec.Body.String())
		}
	})
}

func TestWithFS(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html":  {Data: []byte("index")},
		"css/app.css": {Data: []byte("body{}")},
	}

	app := mizu.NewRouter()
	app.Use(WithFS(fsys))

	t.Run("serve file from FS", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/css/app.css", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
		if rec.Body.String() != "body{}" {
			t.Errorf("expected 'body{}', got %q", rec.Body.String())
		}
	})
}

func TestWithOptions_Prefix(t *testing.T) {
	fsys := fstest.MapFS{
		"app.js": {Data: []byte("console.log('hello')")},
	}

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		FS:     fsys,
		Prefix: "/static",
	}))

	app.Get("/api", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "api")
	})

	t.Run("with prefix", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/static/app.js", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("without prefix falls through", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Body.String() != "api" {
			t.Errorf("expected 'api', got %q", rec.Body.String())
		}
	})
}

func TestWithOptions_MaxAge(t *testing.T) {
	fsys := fstest.MapFS{
		"style.css": {Data: []byte("body{}")},
	}

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		FS:     fsys,
		MaxAge: 3600,
	}))

	req := httptest.NewRequest(http.MethodGet, "/style.css", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Cache-Control") != "public, max-age=3600" {
		t.Errorf("expected cache control header, got %q", rec.Header().Get("Cache-Control"))
	}
}

func TestWithOptions_NotFoundHandler(t *testing.T) {
	fsys := fstest.MapFS{}

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		FS: fsys,
		NotFoundHandler: func(c *mizu.Ctx) error {
			return c.Text(http.StatusNotFound, "custom 404")
		},
	}))

	req := httptest.NewRequest(http.MethodGet, "/missing.txt", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected %d, got %d", http.StatusNotFound, rec.Code)
	}
	if rec.Body.String() != "custom 404" {
		t.Errorf("expected 'custom 404', got %q", rec.Body.String())
	}
}

func TestWithOptions_CustomIndex(t *testing.T) {
	fsys := fstest.MapFS{
		"default.htm": {Data: []byte("default page")},
	}

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		FS:    fsys,
		Index: "default.htm",
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
	if rec.Body.String() != "default page" {
		t.Errorf("expected 'default page', got %q", rec.Body.String())
	}
}

func TestWithOptions_Browse(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	_ = os.Mkdir(subDir, 0750)                                                   //nolint:gosec // G301: Test directory
	_ = os.WriteFile(filepath.Join(subDir, "file.txt"), []byte("content"), 0600) //nolint:gosec // G306: Test file

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Root:   tmpDir,
		Browse: true,
	}))

	req := httptest.NewRequest(http.MethodGet, "/subdir/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestWithOptions_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic")
		}
	}()
	WithOptions(Options{})
}

// Ensure fs.FS implementation
var _ fs.FS = fstest.MapFS{}
