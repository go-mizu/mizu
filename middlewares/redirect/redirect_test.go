package redirect

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestHTTPSRedirect(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(HTTPSRedirect())

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("redirects HTTP to HTTPS", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusMovedPermanently {
			t.Errorf("expected status %d, got %d", http.StatusMovedPermanently, rec.Code)
		}

		location := rec.Header().Get("Location")
		if location != "https://example.com/test" {
			t.Errorf("expected https redirect, got %q", location)
		}
	})

	t.Run("allows HTTPS", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "https://example.com/test", nil)
		req.Header.Set("X-Forwarded-Proto", "https")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
	})
}

func TestHTTPSRedirectCode(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(HTTPSRedirectCode(http.StatusTemporaryRedirect))

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusTemporaryRedirect {
		t.Errorf("expected status %d, got %d", http.StatusTemporaryRedirect, rec.Code)
	}
}

func TestWWWRedirect(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WWWRedirect())

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("redirects to www", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusMovedPermanently {
			t.Errorf("expected status %d, got %d", http.StatusMovedPermanently, rec.Code)
		}

		location := rec.Header().Get("Location")
		if location != "http://www.example.com/test" {
			t.Errorf("expected www redirect, got %q", location)
		}
	})

	t.Run("allows www", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://www.example.com/test", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
	})
}

func TestNonWWWRedirect(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(NonWWWRedirect())

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("redirects from www", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://www.example.com/test", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusMovedPermanently {
			t.Errorf("expected status %d, got %d", http.StatusMovedPermanently, rec.Code)
		}

		location := rec.Header().Get("Location")
		if location != "http://example.com/test" {
			t.Errorf("expected non-www redirect, got %q", location)
		}
	})

	t.Run("allows non-www", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
	})
}

func TestNew(t *testing.T) {
	rules := []Rule{
		{From: "/old", To: "/new", Code: http.StatusMovedPermanently},
		{From: "/legacy", To: "/modern", Code: http.StatusFound},
	}

	app := mizu.NewRouter()
	app.Use(New(rules))

	app.Get("/new", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "new")
	})

	t.Run("redirects matching rule", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/old", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusMovedPermanently {
			t.Errorf("expected status %d, got %d", http.StatusMovedPermanently, rec.Code)
		}

		if rec.Header().Get("Location") != "/new" {
			t.Errorf("expected /new, got %q", rec.Header().Get("Location"))
		}
	})

	t.Run("preserves query string", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/old?foo=bar", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		location := rec.Header().Get("Location")
		if location != "/new?foo=bar" {
			t.Errorf("expected /new?foo=bar, got %q", location)
		}
	})
}

func TestNew_Regex(t *testing.T) {
	rules := []Rule{
		{From: `/users/(\d+)`, To: "/profile/$1", Regex: true},
		{From: `/posts/(\d+)/comments/(\d+)`, To: "/c/$2", Regex: true},
	}

	app := mizu.NewRouter()
	app.Use(New(rules))

	app.Get("/profile/{id}", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "profile")
	})

	t.Run("regex redirect with capture", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusMovedPermanently {
			t.Errorf("expected status %d, got %d", http.StatusMovedPermanently, rec.Code)
		}

		location := rec.Header().Get("Location")
		if location != "/profile/123" {
			t.Errorf("expected /profile/123, got %q", location)
		}
	})
}

func TestTrailingSlashRedirect(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(TrailingSlashRedirect())

	app.Get("/test/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("adds trailing slash", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusMovedPermanently {
			t.Errorf("expected status %d, got %d", http.StatusMovedPermanently, rec.Code)
		}

		if rec.Header().Get("Location") != "/test/" {
			t.Errorf("expected /test/, got %q", rec.Header().Get("Location"))
		}
	})

	t.Run("preserves query on redirect", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test?a=1", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Header().Get("Location") != "/test/?a=1" {
			t.Errorf("expected /test/?a=1, got %q", rec.Header().Get("Location"))
		}
	})

	t.Run("skips root path", func(t *testing.T) {
		app2 := mizu.NewRouter()
		app2.Use(TrailingSlashRedirect())
		app2.Get("/", func(c *mizu.Ctx) error {
			return c.Text(http.StatusOK, "root")
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		app2.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("root should not redirect, got status %d", rec.Code)
		}
	})
}
