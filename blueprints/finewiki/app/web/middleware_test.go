package web

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-mizu/mizu"
)

func TestLogging(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	handler := func(c *mizu.Ctx) error {
		c.Text(200, "ok")
		return nil
	}

	middleware := Logging()
	wrapped := middleware(handler)

	// Create a minimal test context
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	r := mizu.New()
	r.Get("/test", func(c *mizu.Ctx) error {
		return wrapped(c)
	})

	r.ServeHTTP(rec, req)

	logOutput := buf.String()
	if !strings.Contains(logOutput, "GET") {
		t.Errorf("log output should contain GET, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "/test") {
		t.Errorf("log output should contain /test, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "200") {
		t.Errorf("log output should contain 200, got: %s", logOutput)
	}
}

func TestRecovery(t *testing.T) {
	handler := func(c *mizu.Ctx) error {
		panic("test panic")
	}

	middleware := Recovery()
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	r := mizu.New()
	r.Get("/test", func(c *mizu.Ctx) error {
		return wrapped(c)
	})

	// Should not panic
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestRequestID(t *testing.T) {
	handler := func(c *mizu.Ctx) error {
		c.Text(200, "ok")
		return nil
	}

	middleware := RequestID()
	wrapped := middleware(handler)

	t.Run("generates ID when not present", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		r := mizu.New()
		r.Get("/test", func(c *mizu.Ctx) error {
			return wrapped(c)
		})
		r.ServeHTTP(rec, req)

		id := rec.Header().Get("X-Request-ID")
		if id == "" {
			t.Error("X-Request-ID header should be set")
		}
	})

	t.Run("uses existing ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Request-ID", "existing-id")
		rec := httptest.NewRecorder()

		r := mizu.New()
		r.Get("/test", func(c *mizu.Ctx) error {
			return wrapped(c)
		})
		r.ServeHTTP(rec, req)

		id := rec.Header().Get("X-Request-ID")
		if id != "existing-id" {
			t.Errorf("X-Request-ID = %q, want %q", id, "existing-id")
		}
	})
}

func TestCache(t *testing.T) {
	handler := func(c *mizu.Ctx) error {
		c.Text(200, "ok")
		return nil
	}

	middleware := Cache(time.Hour)
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	r := mizu.New()
	r.Get("/test", func(c *mizu.Ctx) error {
		return wrapped(c)
	})
	r.ServeHTTP(rec, req)

	cc := rec.Header().Get("Cache-Control")
	if !strings.Contains(cc, "public") {
		t.Errorf("Cache-Control should contain 'public', got: %s", cc)
	}
	if !strings.Contains(cc, "3600") {
		t.Errorf("Cache-Control should contain max-age=3600, got: %s", cc)
	}
}

func TestNoCache(t *testing.T) {
	handler := func(c *mizu.Ctx) error {
		c.Text(200, "ok")
		return nil
	}

	middleware := NoCache()
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	r := mizu.New()
	r.Get("/test", func(c *mizu.Ctx) error {
		return wrapped(c)
	})
	r.ServeHTTP(rec, req)

	cc := rec.Header().Get("Cache-Control")
	if !strings.Contains(cc, "no-cache") {
		t.Errorf("Cache-Control should contain 'no-cache', got: %s", cc)
	}

	pragma := rec.Header().Get("Pragma")
	if pragma != "no-cache" {
		t.Errorf("Pragma = %q, want %q", pragma, "no-cache")
	}

	expires := rec.Header().Get("Expires")
	if expires != "0" {
		t.Errorf("Expires = %q, want %q", expires, "0")
	}
}

func TestGenerateRequestID(t *testing.T) {
	id := generateRequestID()

	if id == "" {
		t.Error("generateRequestID() returned empty string")
	}

	// Verify the format is reasonable (timestamp format)
	if len(id) < 10 {
		t.Errorf("generateRequestID() = %q, seems too short", id)
	}
}

func TestFormatSeconds(t *testing.T) {
	tests := []struct {
		duration time.Duration
		want     string
	}{
		{time.Second, "1"},
		{time.Hour, "3600"},
		{30 * time.Minute, "1800"},
		{24 * time.Hour, "86400"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatSeconds(tt.duration)
			if got != tt.want {
				t.Errorf("formatSeconds(%v) = %q, want %q", tt.duration, got, tt.want)
			}
		})
	}
}
