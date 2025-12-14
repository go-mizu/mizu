package slash

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestAdd(t *testing.T) {
	app := mizu.NewRouter()
	app.UseGlobal(Add())

	app.Get("/test/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("adds trailing slash", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusMovedPermanently {
			t.Errorf("expected %d, got %d", http.StatusMovedPermanently, rec.Code)
		}
		if rec.Header().Get("Location") != "/test/" {
			t.Errorf("expected /test/, got %q", rec.Header().Get("Location"))
		}
	})

	t.Run("preserves query", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test?foo=bar", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Header().Get("Location") != "/test/?foo=bar" {
			t.Errorf("expected /test/?foo=bar, got %q", rec.Header().Get("Location"))
		}
	})

	t.Run("skips root", func(t *testing.T) {
		app2 := mizu.NewRouter()
		app2.UseGlobal(Add())
		app2.Get("/", func(c *mizu.Ctx) error {
			return c.Text(http.StatusOK, "root")
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		app2.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("root should not redirect")
		}
	})
}

func TestAddCode(t *testing.T) {
	app := mizu.NewRouter()
	app.UseGlobal(AddCode(http.StatusTemporaryRedirect))

	app.Get("/test/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusTemporaryRedirect {
		t.Errorf("expected %d, got %d", http.StatusTemporaryRedirect, rec.Code)
	}
}

func TestRemove(t *testing.T) {
	app := mizu.NewRouter()
	app.UseGlobal(Remove())

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("removes trailing slash", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusMovedPermanently {
			t.Errorf("expected %d, got %d", http.StatusMovedPermanently, rec.Code)
		}
		if rec.Header().Get("Location") != "/test" {
			t.Errorf("expected /test, got %q", rec.Header().Get("Location"))
		}
	})

	t.Run("preserves query", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test/?foo=bar", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Header().Get("Location") != "/test?foo=bar" {
			t.Errorf("expected /test?foo=bar, got %q", rec.Header().Get("Location"))
		}
	})

	t.Run("skips root", func(t *testing.T) {
		app2 := mizu.NewRouter()
		app2.UseGlobal(Remove())
		app2.Get("/", func(c *mizu.Ctx) error {
			return c.Text(http.StatusOK, "root")
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		app2.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("root should not redirect")
		}
	})
}

func TestRemoveCode(t *testing.T) {
	app := mizu.NewRouter()
	app.UseGlobal(RemoveCode(http.StatusFound))

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Errorf("expected %d, got %d", http.StatusFound, rec.Code)
	}
}
