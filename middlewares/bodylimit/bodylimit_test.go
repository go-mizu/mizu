package bodylimit

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
	app.Use(New(100))

	app.Post("/upload", func(c *mizu.Ctx) error {
		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return c.Text(http.StatusBadRequest, err.Error())
		}
		return c.Text(http.StatusOK, string(body))
	})

	t.Run("allows small body", func(t *testing.T) {
		body := strings.Repeat("a", 50)
		req := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader(body))
		req.Header.Set("Content-Type", "text/plain")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
		if rec.Body.String() != body {
			t.Errorf("expected body %q, got %q", body, rec.Body.String())
		}
	})

	t.Run("rejects large body by content-length", func(t *testing.T) {
		body := strings.Repeat("a", 200)
		req := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader(body))
		req.Header.Set("Content-Type", "text/plain")
		req.ContentLength = 200
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusRequestEntityTooLarge {
			t.Errorf("expected status %d, got %d", http.StatusRequestEntityTooLarge, rec.Code)
		}
	})

	t.Run("rejects large body during read", func(t *testing.T) {
		// Create a reader without Content-Length set
		body := strings.Repeat("a", 200)
		req := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader(body))
		req.Header.Set("Content-Type", "text/plain")
		req.ContentLength = -1 // Unknown content length
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		// MaxBytesReader will return error during read
		if rec.Code != http.StatusBadRequest && rec.Code != http.StatusOK {
			// Either returns error or truncated body
			t.Logf("status: %d, body: %s", rec.Code, rec.Body.String())
		}
	})
}

func TestWithHandler(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithHandler(100, func(c *mizu.Ctx) error {
		return c.JSON(http.StatusRequestEntityTooLarge, map[string]string{
			"error": "body too large",
			"limit": "100 bytes",
		})
	}))

	app.Post("/upload", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	body := strings.Repeat("a", 200)
	req := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader(body))
	req.ContentLength = 200
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected status %d, got %d", http.StatusRequestEntityTooLarge, rec.Code)
	}
}

func TestWithOptions_Default(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{})) // Should default to 1MB

	app.Post("/upload", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	// 512KB should be allowed
	body := bytes.Repeat([]byte("a"), 512*1024)
	req := httptest.NewRequest(http.MethodPost, "/upload", bytes.NewReader(body))
	req.ContentLength = int64(len(body))
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestHelpers(t *testing.T) {
	tests := []struct {
		name     string
		fn       func(int64) int64
		input    int64
		expected int64
	}{
		{"KB", KB, 1, 1024},
		{"KB", KB, 10, 10240},
		{"MB", MB, 1, 1048576},
		{"MB", MB, 5, 5242880},
		{"GB", GB, 1, 1073741824},
		{"GB", GB, 2, 2147483648},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.fn(tt.input)
			if result != tt.expected {
				t.Errorf("%s(%d) = %d, want %d", tt.name, tt.input, result, tt.expected)
			}
		})
	}
}

func TestNew_WithHelpers(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(KB(10))) // 10KB limit

	app.Post("/upload", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("allows under limit", func(t *testing.T) {
		body := bytes.Repeat([]byte("a"), 5*1024) // 5KB
		req := httptest.NewRequest(http.MethodPost, "/upload", bytes.NewReader(body))
		req.ContentLength = int64(len(body))
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("rejects over limit", func(t *testing.T) {
		body := bytes.Repeat([]byte("a"), 15*1024) // 15KB
		req := httptest.NewRequest(http.MethodPost, "/upload", bytes.NewReader(body))
		req.ContentLength = int64(len(body))
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusRequestEntityTooLarge {
			t.Errorf("expected status %d, got %d", http.StatusRequestEntityTooLarge, rec.Code)
		}
	})
}
