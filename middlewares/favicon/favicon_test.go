package favicon

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/go-mizu/mizu"
)

// Test PNG header bytes
var testPNG = []byte{
	0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
	0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
}

func TestNew(t *testing.T) {
	tmpDir := t.TempDir()
	faviconPath := filepath.Join(tmpDir, "favicon.ico")
	os.WriteFile(faviconPath, testPNG, 0644)

	app := mizu.NewRouter()
	app.Use(New(faviconPath))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "home")
	})

	t.Run("serve favicon", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/favicon.ico", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
		if rec.Header().Get("Content-Type") != "image/png" {
			t.Errorf("expected image/png, got %q", rec.Header().Get("Content-Type"))
		}
	})

	t.Run("other paths pass through", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Body.String() != "home" {
			t.Errorf("expected 'home', got %q", rec.Body.String())
		}
	})
}

func TestFromData(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(FromData(testPNG))

	req := httptest.NewRequest(http.MethodGet, "/favicon.ico", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestWithOptions_CustomURL(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Data: testPNG,
		URL:  "/icon.png",
	}))

	t.Run("custom url serves", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/icon.png", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("default url does not serve", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/favicon.ico", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, rec.Code)
		}
	})
}

func TestWithOptions_FS(t *testing.T) {
	fsys := fstest.MapFS{
		"favicon.ico": {Data: testPNG},
	}

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		FS:   fsys,
		File: "favicon.ico",
	}))

	req := httptest.NewRequest(http.MethodGet, "/favicon.ico", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestWithOptions_MaxAge(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Data:   testPNG,
		MaxAge: 3600,
	}))

	req := httptest.NewRequest(http.MethodGet, "/favicon.ico", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Cache-Control") != "public, max-age=3600" {
		t.Errorf("expected cache control header, got %q", rec.Header().Get("Cache-Control"))
	}
}

func TestWithOptions_HeadMethod(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(FromData(testPNG))

	req := httptest.NewRequest(http.MethodHead, "/favicon.ico", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Error("HEAD should not return body")
	}
}

func TestEmpty(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Empty())

	req := httptest.NewRequest(http.MethodGet, "/favicon.ico", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected %d, got %d", http.StatusNoContent, rec.Code)
	}
}

func TestRedirect(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Redirect("https://cdn.example.com/favicon.ico"))

	req := httptest.NewRequest(http.MethodGet, "/favicon.ico", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusMovedPermanently {
		t.Errorf("expected %d, got %d", http.StatusMovedPermanently, rec.Code)
	}
	if rec.Header().Get("Location") != "https://cdn.example.com/favicon.ico" {
		t.Errorf("expected redirect location, got %q", rec.Header().Get("Location"))
	}
}

func TestSVG(t *testing.T) {
	svgData := []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><circle cx="50" cy="50" r="50"/></svg>`)

	app := mizu.NewRouter()
	app.Use(SVG(svgData))

	t.Run("favicon.ico serves SVG", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/favicon.ico", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
		if rec.Header().Get("Content-Type") != "image/svg+xml" {
			t.Errorf("expected image/svg+xml, got %q", rec.Header().Get("Content-Type"))
		}
	})

	t.Run("favicon.svg serves SVG", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/favicon.svg", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
	})
}

func TestWithOptions_NoData(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{}))

	req := httptest.NewRequest(http.MethodGet, "/favicon.ico", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected %d for empty favicon, got %d", http.StatusNoContent, rec.Code)
	}
}
