package etag

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "Hello, World!")
	})

	t.Run("generates ETag", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}

		etag := rec.Header().Get("ETag")
		if etag == "" {
			t.Error("expected ETag header")
		}
		if !strings.HasPrefix(etag, `"`) || !strings.HasSuffix(etag, `"`) {
			t.Errorf("expected quoted ETag, got %q", etag)
		}
	})

	t.Run("returns 304 for matching If-None-Match", func(t *testing.T) {
		// First request to get ETag
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		etag := rec.Header().Get("ETag")

		// Second request with If-None-Match
		req = httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("If-None-Match", etag)
		rec = httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotModified {
			t.Errorf("expected status %d, got %d", http.StatusNotModified, rec.Code)
		}
	})

	t.Run("returns 200 for non-matching If-None-Match", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("If-None-Match", `"non-matching"`)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("wildcard If-None-Match", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("If-None-Match", "*")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotModified {
			t.Errorf("expected status %d, got %d", http.StatusNotModified, rec.Code)
		}
	})
}

func TestWeak(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Weak())

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "Hello, World!")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	etag := rec.Header().Get("ETag")
	if !strings.HasPrefix(etag, `W/"`) {
		t.Errorf("expected weak ETag (W/\"...\"), got %q", etag)
	}
}

func TestWithOptions_CustomHash(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		HashFunc: func(b []byte) string {
			return "custom-hash"
		},
	}))

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "Hello!")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	etag := rec.Header().Get("ETag")
	if etag != `"custom-hash"` {
		t.Errorf("expected custom hash, got %q", etag)
	}
}

func TestNew_SkipsNonGetHead(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Post("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "posted")
	})

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("ETag") != "" {
		t.Error("should not generate ETag for POST")
	}
}

func TestNew_SkipsErrorResponses(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Get("/error", func(c *mizu.Ctx) error {
		return c.Text(http.StatusInternalServerError, "error")
	})

	req := httptest.NewRequest(http.MethodGet, "/error", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("ETag") != "" {
		t.Error("should not generate ETag for error responses")
	}
}

func TestNew_ConsistentETag(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "Same content")
	})

	// Two requests should get same ETag
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec1 := httptest.NewRecorder()
	app.ServeHTTP(rec1, req1)

	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec2 := httptest.NewRecorder()
	app.ServeHTTP(rec2, req2)

	if rec1.Header().Get("ETag") != rec2.Header().Get("ETag") {
		t.Error("same content should produce same ETag")
	}
}

func TestNew_DifferentContent(t *testing.T) {
	counter := 0
	app := mizu.NewRouter()
	app.Use(New())

	app.Get("/test", func(c *mizu.Ctx) error {
		counter++
		return c.Text(http.StatusOK, strings.Repeat("x", counter))
	})

	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec1 := httptest.NewRecorder()
	app.ServeHTTP(rec1, req1)

	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec2 := httptest.NewRecorder()
	app.ServeHTTP(rec2, req2)

	if rec1.Header().Get("ETag") == rec2.Header().Get("ETag") {
		t.Error("different content should produce different ETags")
	}
}

func TestNew_HEAD(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Head("/test", func(c *mizu.Ctx) error {
		c.Header().Set("Content-Length", "13")
		return c.NoContent()
	})

	req := httptest.NewRequest(http.MethodHead, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// HEAD requests should still get ETag processing
	// (though body may be empty)
	if rec.Code != http.StatusNoContent && rec.Code != http.StatusOK {
		t.Errorf("unexpected status: %d", rec.Code)
	}
}
