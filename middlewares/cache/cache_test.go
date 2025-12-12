package cache

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(1 * time.Hour))

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	cc := rec.Header().Get("Cache-Control")
	if !strings.Contains(cc, "max-age=3600") {
		t.Errorf("expected max-age=3600, got %q", cc)
	}
}

func TestWithOptions(t *testing.T) {
	tests := []struct {
		name     string
		opts     Options
		expected []string
	}{
		{
			"public with max-age",
			Options{Public: true, MaxAge: time.Hour},
			[]string{"public", "max-age=3600"},
		},
		{
			"private with max-age",
			Options{Private: true, MaxAge: 5 * time.Minute},
			[]string{"private", "max-age=300"},
		},
		{
			"no-cache",
			Options{NoCache: true},
			[]string{"no-cache"},
		},
		{
			"no-store",
			Options{NoStore: true},
			[]string{"no-store"},
		},
		{
			"must-revalidate",
			Options{MaxAge: time.Hour, MustRevalidate: true},
			[]string{"must-revalidate", "max-age=3600"},
		},
		{
			"immutable",
			Options{Public: true, MaxAge: 365 * 24 * time.Hour, Immutable: true},
			[]string{"public", "immutable"},
		},
		{
			"s-maxage",
			Options{MaxAge: time.Hour, SMaxAge: 2 * time.Hour},
			[]string{"max-age=3600", "s-maxage=7200"},
		},
		{
			"stale-while-revalidate",
			Options{MaxAge: time.Hour, StaleWhileRevalidate: 30 * time.Minute},
			[]string{"max-age=3600", "stale-while-revalidate=1800"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := mizu.NewRouter()
			app.Use(WithOptions(tt.opts))

			app.Get("/test", func(c *mizu.Ctx) error {
				return c.Text(http.StatusOK, "ok")
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)

			cc := rec.Header().Get("Cache-Control")
			for _, exp := range tt.expected {
				if !strings.Contains(cc, exp) {
					t.Errorf("expected %q in Cache-Control, got %q", exp, cc)
				}
			}
		})
	}
}

func TestPublic(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Public(24 * time.Hour))

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	cc := rec.Header().Get("Cache-Control")
	if !strings.Contains(cc, "public") {
		t.Error("expected public directive")
	}
}

func TestPrivate(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Private(time.Hour))

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	cc := rec.Header().Get("Cache-Control")
	if !strings.Contains(cc, "private") {
		t.Error("expected private directive")
	}
}

func TestImmutable(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Immutable(365 * 24 * time.Hour))

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	cc := rec.Header().Get("Cache-Control")
	if !strings.Contains(cc, "immutable") {
		t.Error("expected immutable directive")
	}
}

func TestStatic(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Static(30 * 24 * time.Hour))

	app.Get("/assets/main.js", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "js content")
	})

	req := httptest.NewRequest(http.MethodGet, "/assets/main.js", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	cc := rec.Header().Get("Cache-Control")
	if !strings.Contains(cc, "public") || !strings.Contains(cc, "immutable") {
		t.Error("expected public and immutable")
	}
}

func TestSWR(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(SWR(time.Hour, 30*time.Minute))

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	cc := rec.Header().Get("Cache-Control")
	if !strings.Contains(cc, "stale-while-revalidate=1800") {
		t.Errorf("expected SWR directive, got %q", cc)
	}
}

func TestWithOptions_DoesNotOverride(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(time.Hour))

	app.Get("/test", func(c *mizu.Ctx) error {
		c.Header().Set("Cache-Control", "no-store")
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	cc := rec.Header().Get("Cache-Control")
	if cc != "no-store" {
		t.Errorf("should not override existing Cache-Control, got %q", cc)
	}
}

func TestWithOptions_Empty(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{}))

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	cc := rec.Header().Get("Cache-Control")
	if cc != "no-cache" {
		t.Errorf("expected 'no-cache' for empty options, got %q", cc)
	}
}
