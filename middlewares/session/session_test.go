package session

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(Options{}))

	app.Get("/set", func(c *mizu.Ctx) error {
		sess := Get(c)
		sess.Set("user", "john")
		return c.Text(http.StatusOK, "set")
	})

	app.Get("/get", func(c *mizu.Ctx) error {
		sess := Get(c)
		user, _ := sess.Get("user").(string)
		return c.Text(http.StatusOK, user)
	})

	t.Run("set and get session value", func(t *testing.T) {
		// Set value
		req := httptest.NewRequest(http.MethodGet, "/set", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}

		// Get cookie
		cookies := rec.Result().Cookies()
		var sessionCookie *http.Cookie
		for _, c := range cookies {
			if c.Name == "session_id" {
				sessionCookie = c
				break
			}
		}
		if sessionCookie == nil {
			t.Fatal("expected session cookie")
		}

		// Get value with same session
		req = httptest.NewRequest(http.MethodGet, "/get", nil)
		req.AddCookie(sessionCookie)
		rec = httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Body.String() != "john" {
			t.Errorf("expected 'john', got %q", rec.Body.String())
		}
	})
}

func TestSession_Clear(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(Options{}))

	app.Get("/set", func(c *mizu.Ctx) error {
		sess := Get(c)
		sess.Set("a", "1")
		sess.Set("b", "2")
		return c.Text(http.StatusOK, "set")
	})

	app.Get("/clear", func(c *mizu.Ctx) error {
		sess := Get(c)
		sess.Clear()
		return c.Text(http.StatusOK, "cleared")
	})

	app.Get("/check", func(c *mizu.Ctx) error {
		sess := Get(c)
		if sess.Get("a") != nil || sess.Get("b") != nil {
			return c.Text(http.StatusOK, "not empty")
		}
		return c.Text(http.StatusOK, "empty")
	})

	// Set values
	req := httptest.NewRequest(http.MethodGet, "/set", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	var sessionCookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == "session_id" {
			sessionCookie = c
			break
		}
	}

	// Clear
	req = httptest.NewRequest(http.MethodGet, "/clear", nil)
	req.AddCookie(sessionCookie)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Check
	req = httptest.NewRequest(http.MethodGet, "/check", nil)
	req.AddCookie(sessionCookie)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "empty" {
		t.Errorf("expected 'empty', got %q", rec.Body.String())
	}
}

func TestSession_Delete(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(Options{}))

	app.Get("/set", func(c *mizu.Ctx) error {
		sess := Get(c)
		sess.Set("key", "value")
		return c.Text(http.StatusOK, "set")
	})

	app.Get("/delete", func(c *mizu.Ctx) error {
		sess := Get(c)
		sess.Delete("key")
		return c.Text(http.StatusOK, "deleted")
	})

	app.Get("/check", func(c *mizu.Ctx) error {
		sess := Get(c)
		if sess.Get("key") == nil {
			return c.Text(http.StatusOK, "nil")
		}
		return c.Text(http.StatusOK, "exists")
	})

	// Set
	req := httptest.NewRequest(http.MethodGet, "/set", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	var sessionCookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == "session_id" {
			sessionCookie = c
			break
		}
	}

	// Delete
	req = httptest.NewRequest(http.MethodGet, "/delete", nil)
	req.AddCookie(sessionCookie)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Check
	req = httptest.NewRequest(http.MethodGet, "/check", nil)
	req.AddCookie(sessionCookie)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "nil" {
		t.Errorf("expected 'nil', got %q", rec.Body.String())
	}
}

func TestWithStore(t *testing.T) {
	store := NewMemoryStore()
	app := mizu.NewRouter()
	app.Use(WithStore(store, Options{}))

	app.Get("/test", func(c *mizu.Ctx) error {
		sess := Get(c)
		sess.Set("test", true)
		return c.Text(http.StatusOK, sess.ID)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestSession_CustomCookie(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(Options{
		CookieName:   "my_session",
		CookiePath:   "/api",
		CookieSecure: true,
		SameSite:     http.SameSiteStrictMode,
	}))

	app.Get("/api/test", func(c *mizu.Ctx) error {
		sess := Get(c)
		sess.Set("x", 1)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	var found bool
	for _, c := range rec.Result().Cookies() {
		if c.Name == "my_session" {
			found = true
			if c.Path != "/api" {
				t.Errorf("expected path /api, got %s", c.Path)
			}
			if !c.Secure {
				t.Error("expected secure cookie")
			}
		}
	}
	if !found {
		t.Error("expected my_session cookie")
	}
}

func TestFromContext(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(Options{}))

	var s1, s2 *Session
	app.Get("/test", func(c *mizu.Ctx) error {
		s1 = Get(c)
		s2 = FromContext(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if s1 != s2 {
		t.Error("Get and FromContext should return same session")
	}
}

func TestGenerateSessionID(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := generateSessionID()
		if len(id) != 64 { // 32 bytes = 64 hex chars
			t.Errorf("expected 64 char ID, got %d", len(id))
		}
		if ids[id] {
			t.Error("duplicate ID generated")
		}
		ids[id] = true
	}
}
