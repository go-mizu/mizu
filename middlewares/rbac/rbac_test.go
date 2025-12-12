package rbac

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-mizu/mizu"
)

func setupUserMiddleware(user *User) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			if user != nil {
				Set(c, user)
			}
			return next(c)
		}
	}
}

func TestRequireRole(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(setupUserMiddleware(&User{
		ID:    "1",
		Roles: []string{"user", "editor"},
	}))
	app.Use(RequireRole("editor"))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestRequireRole_Forbidden(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(setupUserMiddleware(&User{
		ID:    "1",
		Roles: []string{"user"},
	}))
	app.Use(RequireRole("admin"))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestRequireAnyRole(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(setupUserMiddleware(&User{
		ID:    "1",
		Roles: []string{"editor"},
	}))
	app.Use(RequireAnyRole("admin", "editor"))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestRequireAllRoles(t *testing.T) {
	t.Run("has all", func(t *testing.T) {
		app := mizu.NewRouter()
		app.Use(setupUserMiddleware(&User{
			ID:    "1",
			Roles: []string{"admin", "editor"},
		}))
		app.Use(RequireAllRoles("admin", "editor"))

		app.Get("/", func(c *mizu.Ctx) error {
			return c.Text(http.StatusOK, "ok")
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("missing one", func(t *testing.T) {
		app := mizu.NewRouter()
		app.Use(setupUserMiddleware(&User{
			ID:    "1",
			Roles: []string{"admin"},
		}))
		app.Use(RequireAllRoles("admin", "editor"))

		app.Get("/", func(c *mizu.Ctx) error {
			return c.Text(http.StatusOK, "ok")
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected %d, got %d", http.StatusForbidden, rec.Code)
		}
	})
}

func TestRequirePermission(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(setupUserMiddleware(&User{
		ID:          "1",
		Permissions: []string{"read:posts", "write:posts"},
	}))
	app.Use(RequirePermission("write:posts"))

	app.Post("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestAdmin(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(setupUserMiddleware(&User{
		ID:    "1",
		Roles: []string{"admin"},
	}))
	app.Use(Admin())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestAuthenticated(t *testing.T) {
	t.Run("authenticated", func(t *testing.T) {
		app := mizu.NewRouter()
		app.Use(setupUserMiddleware(&User{ID: "1"}))
		app.Use(Authenticated())

		app.Get("/", func(c *mizu.Ctx) error {
			return c.Text(http.StatusOK, "ok")
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		app := mizu.NewRouter()
		app.Use(setupUserMiddleware(nil))
		app.Use(Authenticated())

		app.Get("/", func(c *mizu.Ctx) error {
			return c.Text(http.StatusOK, "ok")
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})
}

func TestHasRole(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(setupUserMiddleware(&User{
		ID:    "1",
		Roles: []string{"user", "editor"},
	}))

	var hasUser, hasAdmin bool
	app.Get("/", func(c *mizu.Ctx) error {
		hasUser = HasRole(c, "user")
		hasAdmin = HasRole(c, "admin")
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !hasUser {
		t.Error("expected HasRole(user) to be true")
	}
	if hasAdmin {
		t.Error("expected HasRole(admin) to be false")
	}
}

func TestGet(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(setupUserMiddleware(&User{ID: "123"}))

	var user *User
	app.Get("/", func(c *mizu.Ctx) error {
		user = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if user == nil || user.ID != "123" {
		t.Errorf("expected user with ID 123, got %v", user)
	}
}

func TestNoUser(t *testing.T) {
	app := mizu.NewRouter()

	var hasRole bool
	app.Get("/", func(c *mizu.Ctx) error {
		hasRole = HasRole(c, "admin")
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if hasRole {
		t.Error("expected HasRole to be false without user")
	}
}
