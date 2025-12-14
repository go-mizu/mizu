package slash

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestAdd(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Add())

	// Canonical GET lives at /test/
	app.Get("/test/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	// Non-idempotent method should NOT be redirected, even if it lacks a slash.
	// Provide a POST handler at /test so we can assert pass-through.
	app.Post("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "post-ok")
	})

	t.Run("adds trailing slash for GET", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusMovedPermanently {
			t.Fatalf("expected %d, got %d", http.StatusMovedPermanently, rec.Code)
		}
		if got := rec.Header().Get("Location"); got != "/test/" {
			t.Fatalf("expected Location %q, got %q", "/test/", got)
		}
	})

	t.Run("adds trailing slash for HEAD", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodHead, "/test", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusMovedPermanently {
			t.Fatalf("expected %d, got %d", http.StatusMovedPermanently, rec.Code)
		}
		if got := rec.Header().Get("Location"); got != "/test/" {
			t.Fatalf("expected Location %q, got %q", "/test/", got)
		}
	})

	t.Run("preserves query", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test?foo=bar", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusMovedPermanently {
			t.Fatalf("expected %d, got %d", http.StatusMovedPermanently, rec.Code)
		}
		if got := rec.Header().Get("Location"); got != "/test/?foo=bar" {
			t.Fatalf("expected Location %q, got %q", "/test/?foo=bar", got)
		}
	})

	t.Run("skips root", func(t *testing.T) {
		app2 := mizu.NewRouter()
		app2.Use(Add())
		app2.Get("/", func(c *mizu.Ctx) error {
			return c.Text(http.StatusOK, "root")
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		app2.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
		}
		if got := rec.Header().Get("Location"); got != "" {
			t.Fatalf("expected no Location header, got %q", got)
		}
	})

	t.Run("does not redirect POST", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
		}
		if got := rec.Header().Get("Location"); got != "" {
			t.Fatalf("expected no Location header, got %q", got)
		}
	})
}

func TestAddCode(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(AddCode(http.StatusTemporaryRedirect))

	app.Get("/test/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected %d, got %d", http.StatusTemporaryRedirect, rec.Code)
	}
	if got := rec.Header().Get("Location"); got != "/test/" {
		t.Fatalf("expected Location %q, got %q", "/test/", got)
	}
}

func TestRemove(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Remove())

	// Canonical GET lives at /test
	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	// Non-idempotent method should NOT be redirected, even if it has a slash.
	app.Post("/test/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "post-ok")
	})

	t.Run("removes trailing slash for GET", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusMovedPermanently {
			t.Fatalf("expected %d, got %d", http.StatusMovedPermanently, rec.Code)
		}
		if got := rec.Header().Get("Location"); got != "/test" {
			t.Fatalf("expected Location %q, got %q", "/test", got)
		}
	})

	t.Run("removes trailing slash for HEAD", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodHead, "/test/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusMovedPermanently {
			t.Fatalf("expected %d, got %d", http.StatusMovedPermanently, rec.Code)
		}
		if got := rec.Header().Get("Location"); got != "/test" {
			t.Fatalf("expected Location %q, got %q", "/test", got)
		}
	})

	t.Run("preserves query", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test/?foo=bar", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusMovedPermanently {
			t.Fatalf("expected %d, got %d", http.StatusMovedPermanently, rec.Code)
		}
		if got := rec.Header().Get("Location"); got != "/test?foo=bar" {
			t.Fatalf("expected Location %q, got %q", "/test?foo=bar", got)
		}
	})

	t.Run("skips root", func(t *testing.T) {
		app2 := mizu.NewRouter()
		app2.Use(Remove())
		app2.Get("/", func(c *mizu.Ctx) error {
			return c.Text(http.StatusOK, "root")
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		app2.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
		}
		if got := rec.Header().Get("Location"); got != "" {
			t.Fatalf("expected no Location header, got %q", got)
		}
	})

	t.Run("does not redirect POST", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/test/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
		}
		if got := rec.Header().Get("Location"); got != "" {
			t.Fatalf("expected no Location header, got %q", got)
		}
	})
}

func TestRemoveCode(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(RemoveCode(http.StatusFound))

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected %d, got %d", http.StatusFound, rec.Code)
	}
	if got := rec.Header().Get("Location"); got != "/test" {
		t.Fatalf("expected Location %q, got %q", "/test", got)
	}
}
