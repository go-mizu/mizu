package lastmodified

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	lastMod := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	app := mizu.NewRouter()
	app.Use(New(func(_ *mizu.Ctx) time.Time {
		return lastMod
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Last-Modified") == "" {
		t.Error("expected Last-Modified header")
	}
}

func TestIfModifiedSince_NotModified(t *testing.T) {
	lastMod := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	app := mizu.NewRouter()
	app.Use(New(func(_ *mizu.Ctx) time.Time {
		return lastMod
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("If-Modified-Since", lastMod.UTC().Format(http.TimeFormat))
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotModified {
		t.Errorf("expected %d, got %d", http.StatusNotModified, rec.Code)
	}
}

func TestIfModifiedSince_Modified(t *testing.T) {
	lastMod := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	app := mizu.NewRouter()
	app.Use(New(func(_ *mizu.Ctx) time.Time {
		return lastMod
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	olderTime := lastMod.Add(-time.Hour)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("If-Modified-Since", olderTime.UTC().Format(http.TimeFormat))
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestWithOptions_SkipPaths(t *testing.T) {
	lastMod := time.Now()

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		TimeFunc:  func(_ *mizu.Ctx) time.Time { return lastMod },
		SkipPaths: []string{"/skip"},
	}))

	app.Get("/skip", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/skip", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Last-Modified") != "" {
		t.Error("expected no Last-Modified for skipped path")
	}
}

func TestSkipPostMethod(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(func(_ *mizu.Ctx) time.Time {
		return time.Now()
	}))

	app.Post("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// POST should not get Last-Modified
	if rec.Header().Get("Last-Modified") != "" {
		t.Error("expected no Last-Modified for POST")
	}
}

func TestStatic(t *testing.T) {
	fixedTime := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)

	app := mizu.NewRouter()
	app.Use(Static(fixedTime))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	expected := fixedTime.UTC().Format(http.TimeFormat)
	if rec.Header().Get("Last-Modified") != expected {
		t.Errorf("expected %q, got %q", expected, rec.Header().Get("Last-Modified"))
	}
}

func TestNow(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Now())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Last-Modified") == "" {
		t.Error("expected Last-Modified header")
	}
}

func TestStartupTime(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(StartupTime())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	// First request
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	firstTime := rec.Header().Get("Last-Modified")

	// Second request should have same time
	time.Sleep(10 * time.Millisecond)
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	secondTime := rec.Header().Get("Last-Modified")

	if firstTime != secondTime {
		t.Errorf("startup time should be consistent: %q vs %q", firstTime, secondTime)
	}
}

func TestFromHeader(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(FromHeader("X-Resource-Modified"))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	lastMod := time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Resource-Modified", lastMod.Format(http.TimeFormat))
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Last-Modified") == "" {
		t.Error("expected Last-Modified from custom header")
	}
}

func TestZeroTime(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(func(_ *mizu.Ctx) time.Time {
		return time.Time{} // Zero time
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Should not set header for zero time
	if rec.Header().Get("Last-Modified") != "" {
		t.Error("expected no header for zero time")
	}
}
