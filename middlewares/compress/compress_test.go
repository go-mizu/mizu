package compress

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestGzip(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Gzip())

	largeBody := strings.Repeat("Hello, World! ", 200)
	app.Get("/test", func(c *mizu.Ctx) error {
		c.Header().Set("Content-Type", "text/plain")
		return c.Text(http.StatusOK, largeBody)
	})

	t.Run("compresses response", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}

		if rec.Header().Get("Content-Encoding") != "gzip" {
			t.Error("expected Content-Encoding: gzip")
		}

		// Decompress and verify
		gr, err := gzip.NewReader(rec.Body)
		if err != nil {
			t.Fatalf("failed to create gzip reader: %v", err)
		}
		defer func() { _ = gr.Close() }()

		body, err := io.ReadAll(gr)
		if err != nil {
			t.Fatalf("failed to read gzip body: %v", err)
		}

		if string(body) != largeBody {
			t.Errorf("decompressed body mismatch")
		}
	})

	t.Run("skips without accept-encoding", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Header().Get("Content-Encoding") == "gzip" {
			t.Error("should not compress without Accept-Encoding")
		}
	})

	t.Run("sets vary header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Header().Get("Vary") != "Accept-Encoding" {
			t.Error("expected Vary: Accept-Encoding")
		}
	})
}

func TestGzipLevel(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(GzipLevel(9)) // Best compression

	largeBody := strings.Repeat("Hello, World! ", 200)
	app.Get("/test", func(c *mizu.Ctx) error {
		c.Header().Set("Content-Type", "text/plain")
		return c.Text(http.StatusOK, largeBody)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Error("expected Content-Encoding: gzip")
	}
}

func TestDeflate(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Deflate())

	largeBody := strings.Repeat("Hello, World! ", 200)
	app.Get("/test", func(c *mizu.Ctx) error {
		c.Header().Set("Content-Type", "text/plain")
		return c.Text(http.StatusOK, largeBody)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Accept-Encoding", "deflate")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Encoding") != "deflate" {
		t.Error("expected Content-Encoding: deflate")
	}
}

func TestNew_SmallResponse(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(Options{MinSize: 1000}))

	app.Get("/small", func(c *mizu.Ctx) error {
		c.Header().Set("Content-Type", "text/plain")
		return c.Text(http.StatusOK, "small")
	})

	req := httptest.NewRequest(http.MethodGet, "/small", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Encoding") == "gzip" {
		t.Error("should not compress small responses")
	}
}

func TestNew_NonCompressibleType(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(Options{
		MinSize:      10,
		ContentTypes: []string{"text/plain"},
	}))

	app.Get("/binary", func(c *mizu.Ctx) error {
		return c.Bytes(http.StatusOK, []byte(strings.Repeat("x", 2000)), "application/octet-stream")
	})

	req := httptest.NewRequest(http.MethodGet, "/binary", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Encoding") == "gzip" {
		t.Error("should not compress non-compressible content types")
	}
}

func TestNew_AlreadyEncoded(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(Options{MinSize: 10}))

	largeBody := strings.Repeat("x", 2000)
	app.Get("/encoded", func(c *mizu.Ctx) error {
		c.Header().Set("Content-Type", "text/plain")
		c.Header().Set("Content-Encoding", "br") // Brotli
		return c.Text(http.StatusOK, largeBody)
	})

	req := httptest.NewRequest(http.MethodGet, "/encoded", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Encoding") != "br" {
		t.Error("should preserve existing Content-Encoding")
	}
}

func TestNew_AutoSelectEncoding(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(Options{MinSize: 10}))

	largeBody := strings.Repeat("Hello ", 500)
	app.Get("/test", func(c *mizu.Ctx) error {
		c.Header().Set("Content-Type", "text/plain")
		return c.Text(http.StatusOK, largeBody)
	})

	tests := []struct {
		acceptEncoding   string
		expectedEncoding string
	}{
		{"gzip", "gzip"},
		{"deflate", "deflate"},
		{"gzip, deflate", "gzip"}, // Prefers gzip
		{"br, gzip", "gzip"},
		{"identity", ""},
	}

	for _, tt := range tests {
		t.Run(tt.acceptEncoding, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Accept-Encoding", tt.acceptEncoding)
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)

			got := rec.Header().Get("Content-Encoding")
			if got != tt.expectedEncoding {
				t.Errorf("Accept-Encoding %q: expected %q, got %q",
					tt.acceptEncoding, tt.expectedEncoding, got)
			}
		})
	}
}
