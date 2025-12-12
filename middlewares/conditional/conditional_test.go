package conditional

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "hello")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("ETag") == "" {
		t.Error("expected ETag header")
	}
}

func TestIfNoneMatch_NotModified(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "consistent content")
	})

	// First request to get ETag
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	etag := rec.Header().Get("ETag")

	// Second request with If-None-Match
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("If-None-Match", etag)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotModified {
		t.Errorf("expected %d, got %d", http.StatusNotModified, rec.Code)
	}
}

func TestIfNoneMatch_Modified(t *testing.T) {
	var content string

	app := mizu.NewRouter()
	app.Use(New())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, content)
	})

	// First request
	content = "content v1"
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	etag := rec.Header().Get("ETag")

	// Content changed
	content = "content v2"
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("If-None-Match", etag)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d for modified content, got %d", http.StatusOK, rec.Code)
	}
}

func TestWithOptions_WeakETag(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		ETag:     true,
		WeakETag: true,
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "content")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	etag := rec.Header().Get("ETag")
	if len(etag) < 2 || etag[:2] != "W/" {
		t.Errorf("expected weak ETag, got %q", etag)
	}
}

func TestWithOptions_LastModified(t *testing.T) {
	modTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		ETag:         true,
		LastModified: true,
		ModTimeFunc: func(_ *mizu.Ctx) time.Time {
			return modTime
		},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "content")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Last-Modified") == "" {
		t.Error("expected Last-Modified header")
	}
}

func TestIfModifiedSince(t *testing.T) {
	modTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		ETag:         false, // Disable ETag to test Last-Modified alone
		LastModified: true,
		ModTimeFunc: func(_ *mizu.Ctx) time.Time {
			return modTime
		},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "content")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("If-Modified-Since", modTime.Format(http.TimeFormat))
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotModified {
		t.Errorf("expected %d, got %d", http.StatusNotModified, rec.Code)
	}
}

func TestSkipNonGET(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Post("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("ETag") != "" {
		t.Error("expected no ETag for POST")
	}
}

func TestETagOnly(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(ETagOnly())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "content")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("ETag") == "" {
		t.Error("expected ETag header")
	}
}

func TestLastModifiedOnly(t *testing.T) {
	modTime := time.Now()

	app := mizu.NewRouter()
	app.Use(LastModifiedOnly(func(_ *mizu.Ctx) time.Time {
		return modTime
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "content")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Last-Modified") == "" {
		t.Error("expected Last-Modified header")
	}
}

func TestWithModTime(t *testing.T) {
	modTime := time.Now()

	app := mizu.NewRouter()
	app.Use(WithModTime(func(_ *mizu.Ctx) time.Time {
		return modTime
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "content")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("ETag") == "" {
		t.Error("expected ETag header")
	}
	if rec.Header().Get("Last-Modified") == "" {
		t.Error("expected Last-Modified header")
	}
}

func TestCustomETagFunc(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		ETag: true,
		ETagFunc: func(_ []byte) string {
			return "custom-etag-value"
		},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "content")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	etag := rec.Header().Get("ETag")
	if etag != `"custom-etag-value"` {
		t.Errorf("expected custom ETag, got %q", etag)
	}
}
