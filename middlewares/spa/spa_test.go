package spa

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	tmpDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(tmpDir, "index.html"), []byte("<html>spa</html>"), 0600) //nolint:gosec // G306: Test file
	_ = os.WriteFile(filepath.Join(tmpDir, "app.js"), []byte("console.log('app')"), 0600)   //nolint:gosec // G306: Test file

	app := mizu.NewRouter()
	app.Use(New(tmpDir))

	// Register catch-all route for SPA
	app.Get("/{path...}", func(c *mizu.Ctx) error {
		return c.Text(http.StatusNotFound, "Not Found")
	})

	t.Run("serve static file", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/app.js", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
		if rec.Body.String() != "console.log('app')" {
			t.Errorf("expected js content, got %q", rec.Body.String())
		}
	})

	t.Run("fallback to index for route", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
		if rec.Body.String() != "<html>spa</html>" {
			t.Errorf("expected index.html, got %q", rec.Body.String())
		}
	})

	t.Run("root path", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
		if rec.Body.String() != "<html>spa</html>" {
			t.Errorf("expected index.html, got %q", rec.Body.String())
		}
	})
}

func TestWithFS(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html": {Data: []byte("spa index")},
		"main.js":    {Data: []byte("main()")},
	}

	app := mizu.NewRouter()
	app.Use(WithFS(fsys))

	// Register catch-all route for SPA
	app.Get("/{path...}", func(c *mizu.Ctx) error {
		return c.Text(http.StatusNotFound, "Not Found")
	})

	t.Run("serve from FS", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/main.js", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Body.String() != "main()" {
			t.Errorf("expected 'main()', got %q", rec.Body.String())
		}
	})

	t.Run("fallback from FS", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Body.String() != "spa index" {
			t.Errorf("expected 'spa index', got %q", rec.Body.String())
		}
	})
}

func TestWithOptions_IgnorePaths(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html": {Data: []byte("spa")},
	}

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		FS:          fsys,
		IgnorePaths: []string{"/api", "/health"},
	}))

	app.Get("/api/users", func(c *mizu.Ctx) error {
		return c.JSON(http.StatusOK, map[string]string{"users": "list"})
	})

	app.Get("/health", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	// Register catch-all route for SPA
	app.Get("/{path...}", func(c *mizu.Ctx) error {
		return c.Text(http.StatusNotFound, "Not Found")
	})

	t.Run("ignore api path", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
		if rec.Body.String() == "spa" {
			t.Error("expected API response, got SPA index")
		}
	})

	t.Run("ignore health path", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Body.String() != "ok" {
			t.Errorf("expected 'ok', got %q", rec.Body.String())
		}
	})

	t.Run("non-ignored path gets SPA", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/profile", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Body.String() != "spa" {
			t.Errorf("expected 'spa', got %q", rec.Body.String())
		}
	})
}

func TestWithOptions_Prefix(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html": {Data: []byte("spa")},
		"app.js":     {Data: []byte("app()")},
	}

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		FS:     fsys,
		Prefix: "/app",
	}))

	app.Get("/other", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "other")
	})

	// Register catch-all route for SPA
	app.Get("/{path...}", func(c *mizu.Ctx) error {
		return c.Text(http.StatusNotFound, "Not Found")
	})

	t.Run("with prefix", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/app/dashboard", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Body.String() != "spa" {
			t.Errorf("expected 'spa', got %q", rec.Body.String())
		}
	})

	t.Run("without prefix", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/other", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Body.String() != "other" {
			t.Errorf("expected 'other', got %q", rec.Body.String())
		}
	})
}

func TestWithOptions_CacheControl(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html": {Data: []byte("spa")},
		"static.js":  {Data: []byte("static")},
	}

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		FS:          fsys,
		MaxAge:      31536000,
		IndexMaxAge: 0,
	}))

	// Register catch-all route for SPA
	app.Get("/{path...}", func(c *mizu.Ctx) error {
		return c.Text(http.StatusNotFound, "Not Found")
	})

	t.Run("static asset cache", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/static.js", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Header().Get("Cache-Control") != "public, max-age=31536000" {
			t.Errorf("expected cache control for static, got %q", rec.Header().Get("Cache-Control"))
		}
	})

	t.Run("index no cache", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/route", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		cc := rec.Header().Get("Cache-Control")
		if cc != "no-cache, no-store, must-revalidate" {
			t.Errorf("expected no cache for index, got %q", cc)
		}
	})
}

func TestWithOptions_CustomIndex(t *testing.T) {
	fsys := fstest.MapFS{
		"app.html": {Data: []byte("custom index")},
	}

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		FS:    fsys,
		Index: "app.html",
	}))

	// Register catch-all route for SPA
	app.Get("/{path...}", func(c *mizu.Ctx) error {
		return c.Text(http.StatusNotFound, "Not Found")
	})

	req := httptest.NewRequest(http.MethodGet, "/any/route", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "custom index" {
		t.Errorf("expected 'custom index', got %q", rec.Body.String())
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
