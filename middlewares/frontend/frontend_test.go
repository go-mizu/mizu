package frontend

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	// Create temp directory with test files
	dir := t.TempDir()
	os.WriteFile(dir+"/index.html", []byte("<!DOCTYPE html><html><body>Hello</body></html>"), 0644)
	os.WriteFile(dir+"/app.js", []byte("console.log('app')"), 0644)

	// Set production mode
	os.Setenv("MIZU_ENV", "production")
	defer os.Unsetenv("MIZU_ENV")

	app := mizu.New()
	app.Use(New(dir))
	app.Get("/api/test", func(c *mizu.Ctx) error {
		return c.Text(200, "API")
	})

	t.Run("serves index.html", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != 200 {
			t.Errorf("expected 200, got %d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "Hello") {
			t.Error("expected body to contain Hello")
		}
	})

	t.Run("serves static files", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/app.js", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != 200 {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("SPA fallback", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/some/route", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != 200 {
			t.Errorf("expected 200, got %d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "Hello") {
			t.Error("expected SPA fallback to index.html")
		}
	})

	t.Run("ignores API paths", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/test", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != 200 {
			t.Errorf("expected 200, got %d", rec.Code)
		}
		if rec.Body.String() != "API" {
			t.Errorf("expected API, got %s", rec.Body.String())
		}
	})
}

func TestWithFS(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<!DOCTYPE html><html><body>Embedded</body></html>")},
		"assets/app.abc123.js": &fstest.MapFile{Data: []byte("// app")},
		"assets/styles.css":    &fstest.MapFile{Data: []byte("body {}")},
	}

	os.Setenv("MIZU_ENV", "production")
	defer os.Unsetenv("MIZU_ENV")

	app := mizu.New()
	app.Use(WithFS(fsys))

	t.Run("serves from embedded FS", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != 200 {
			t.Errorf("expected 200, got %d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "Embedded") {
			t.Error("expected body to contain Embedded")
		}
	})

	t.Run("serves assets", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/assets/app.abc123.js", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != 200 {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})
}

func TestWithOptions(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<!DOCTYPE html><html><head></head><body>Test</body></html>")},
	}

	t.Run("custom prefix", func(t *testing.T) {
		os.Setenv("MIZU_ENV", "production")
		defer os.Unsetenv("MIZU_ENV")

		app := mizu.New()
		app.Use(WithOptions(Options{
			FS:     fsys,
			Prefix: "/app",
		}))

		// Request with prefix should work
		req := httptest.NewRequest("GET", "/app/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != 200 {
			t.Errorf("expected 200, got %d", rec.Code)
		}

		// Request without prefix should 404
		req = httptest.NewRequest("GET", "/other", nil)
		rec = httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != 404 {
			t.Errorf("expected 404, got %d", rec.Code)
		}
	})

	t.Run("custom ignore paths", func(t *testing.T) {
		os.Setenv("MIZU_ENV", "production")
		defer os.Unsetenv("MIZU_ENV")

		app := mizu.New()
		app.Use(WithOptions(Options{
			FS:          fsys,
			IgnorePaths: []string{"/custom"},
		}))
		app.Get("/custom/route", func(c *mizu.Ctx) error {
			return c.Text(200, "Custom")
		})

		req := httptest.NewRequest("GET", "/custom/route", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != 200 {
			t.Errorf("expected 200, got %d", rec.Code)
		}
		if rec.Body.String() != "Custom" {
			t.Errorf("expected Custom, got %s", rec.Body.String())
		}
	})

	t.Run("env injection", func(t *testing.T) {
		os.Setenv("MIZU_ENV", "production")
		os.Setenv("TEST_VAR", "test_value")
		defer os.Unsetenv("MIZU_ENV")
		defer os.Unsetenv("TEST_VAR")

		// Need a proper HTML file with <head> tag for injection
		injectFS := fstest.MapFS{
			"index.html": &fstest.MapFile{Data: []byte("<!DOCTYPE html><html><head></head><body>Test</body></html>")},
		}

		app := mizu.New()
		app.Use(WithOptions(Options{
			FS:        injectFS,
			InjectEnv: []string{"TEST_VAR"},
		}))

		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if !strings.Contains(rec.Body.String(), "window.__ENV__") {
			t.Error("expected env injection")
		}
		if !strings.Contains(rec.Body.String(), "test_value") {
			t.Error("expected test value in env")
		}
	})

	t.Run("security headers", func(t *testing.T) {
		os.Setenv("MIZU_ENV", "production")
		defer os.Unsetenv("MIZU_ENV")

		app := mizu.New()
		app.Use(WithOptions(Options{
			FS:              fsys,
			SecurityHeaders: true,
		}))

		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Error("expected X-Content-Type-Options header")
		}
		if rec.Header().Get("X-Frame-Options") != "SAMEORIGIN" {
			t.Error("expected X-Frame-Options header")
		}
	})
}

func TestCacheHeaders(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html":           &fstest.MapFile{Data: []byte("<html></html>")},
		"app.a1b2c3d4.js":      &fstest.MapFile{Data: []byte("// hashed")},
		"vendor-abc123.css":    &fstest.MapFile{Data: []byte("/* hashed */")},
		"logo.png":             &fstest.MapFile{Data: []byte("PNG")},
		"app.js.map":           &fstest.MapFile{Data: []byte("{}")},
	}

	os.Setenv("MIZU_ENV", "production")
	defer os.Unsetenv("MIZU_ENV")

	app := mizu.New()
	app.Use(WithOptions(Options{
		FS:         fsys,
		SourceMaps: true,
		CacheControl: CacheConfig{
			HashedAssets:   365 * 24 * time.Hour,
			UnhashedAssets: 7 * 24 * time.Hour,
		},
	}))

	tests := []struct {
		path     string
		expected string
	}{
		{"/app.a1b2c3d4.js", "public, max-age=31536000, immutable"},
		{"/vendor-abc123.css", "public, max-age=31536000, immutable"},
		{"/logo.png", "public, max-age=604800"},
		{"/index.html", "no-cache, no-store, must-revalidate"},
		{"/app.js.map", "no-cache"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)

			if rec.Code != 200 {
				t.Errorf("expected 200, got %d", rec.Code)
			}

			cc := rec.Header().Get("Cache-Control")
			if cc != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, cc)
			}
		})
	}
}

func TestSourceMapsBlocked(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html></html>")},
		"app.js.map": &fstest.MapFile{Data: []byte("{}")},
	}

	os.Setenv("MIZU_ENV", "production")
	defer os.Unsetenv("MIZU_ENV")

	app := mizu.New()
	app.Use(WithOptions(Options{
		FS:         fsys,
		SourceMaps: false, // Block source maps
	}))

	req := httptest.NewRequest("GET", "/app.js.map", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 404 {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestDevProxy(t *testing.T) {
	// Create a mock dev server
	devServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/vite-hmr" {
			// Simulate WebSocket upgrade failure for testing
			w.WriteHeader(http.StatusOK)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		io.WriteString(w, "<!DOCTYPE html><html><body>Dev Server</body></html>")
	}))
	defer devServer.Close()

	os.Setenv("MIZU_ENV", "development")
	defer os.Unsetenv("MIZU_ENV")

	app := mizu.New()
	app.Use(Dev(devServer.URL))

	t.Run("proxies to dev server", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != 200 {
			t.Errorf("expected 200, got %d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "Dev Server") {
			t.Error("expected response from dev server")
		}
	})
}

func TestModeDetection(t *testing.T) {
	tests := []struct {
		envVar   string
		envValue string
		expected Mode
	}{
		{"MIZU_ENV", "production", ModeProduction},
		{"MIZU_ENV", "prod", ModeProduction},
		{"GO_ENV", "production", ModeProduction},
		{"ENV", "production", ModeProduction},
		{"MIZU_ENV", "development", ModeDev},
		{"MIZU_ENV", "", ModeDev},
	}

	for _, tt := range tests {
		t.Run(tt.envVar+"="+tt.envValue, func(t *testing.T) {
			// Clear all env vars
			os.Unsetenv("MIZU_ENV")
			os.Unsetenv("GO_ENV")
			os.Unsetenv("ENV")

			if tt.envValue != "" {
				os.Setenv(tt.envVar, tt.envValue)
				defer os.Unsetenv(tt.envVar)
			}

			mode := detectMode()
			if mode != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, mode)
			}
		})
	}
}

func TestAssetClassification(t *testing.T) {
	tests := []struct {
		path     string
		expected assetType
	}{
		{"app.a1b2c3d4.js", assetHashed},
		{"vendor-abc123.js", assetHashed},
		{"chunk.ABCDEF.css", assetHashed},
		{"app_1234567890abcdef.js", assetHashed},
		{"app.js", assetUnhashed},
		{"logo.png", assetUnhashed},
		{"fonts/roboto.woff2", assetUnhashed},
		{"index.html", assetHTML},
		{"about.htm", assetHTML},
		{"app.js.map", assetMap},
		{"styles.css.map", assetMap},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := classifyAsset(tt.path)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
