package honeypot

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "home")
	})
	app.Get("/admin", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "admin")
	})

	t.Run("normal path", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("honeypot path", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected %d for honeypot, got %d", http.StatusNotFound, rec.Code)
		}
	})
}

func TestWithOptions_CustomPaths(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Paths: []string{"/trap", "/secret"},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("custom trap path", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/trap", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected %d for honeypot, got %d", http.StatusNotFound, rec.Code)
		}
	})

	t.Run("default path not trapped", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		// /admin is not in custom paths, so should pass
		if rec.Code == http.StatusNotFound {
			t.Error("expected /admin to not be a honeypot with custom paths")
		}
	})
}

func TestWithOptions_OnTrap(t *testing.T) {
	var trappedIP, trappedPath string

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Paths: []string{"/honeypot"},
		OnTrap: func(ip, path string) {
			trappedIP = ip
			trappedPath = path
		},
	}))

	app.Get("/honeypot", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/honeypot", nil)
	req.RemoteAddr = "1.2.3.4:12345"
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if trappedIP == "" {
		t.Error("expected OnTrap to be called")
	}
	if trappedPath != "/honeypot" {
		t.Errorf("expected path '/honeypot', got %q", trappedPath)
	}
}

func TestWithOptions_CustomResponse(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Paths: []string{"/trap"},
		Response: func(c *mizu.Ctx) error {
			return c.JSON(http.StatusTeapot, map[string]string{"message": "gotcha"})
		},
	}))

	app.Get("/trap", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/trap", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusTeapot {
		t.Errorf("expected %d, got %d", http.StatusTeapot, rec.Code)
	}
}

func TestBlockedIP(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	// First, trigger honeypot
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Now try normal request from same IP
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected %d for blocked IP, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestPaths(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Paths("/custom1", "/custom2"))

	app.Get("/custom1", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/custom1", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected %d for custom honeypot, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestAdminPaths(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(AdminPaths())

	app.Get("/adminpanel", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/adminpanel", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected %d for admin honeypot, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestConfigPaths(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(ConfigPaths())

	app.Get("/.env", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/.env", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected %d for config honeypot, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestDatabasePaths(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(DatabasePaths())

	app.Get("/phpmyadmin", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/phpmyadmin", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected %d for database honeypot, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestForm(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Form("honeypot_field"))

	app.Post("/submit", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "submitted")
	})

	t.Run("empty honeypot field", func(t *testing.T) {
		form := url.Values{}
		form.Set("name", "John")
		form.Set("honeypot_field", "")

		req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d for empty honeypot, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("filled honeypot field", func(t *testing.T) {
		form := url.Values{}
		form.Set("name", "John")
		form.Set("honeypot_field", "bot filled this")

		req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected %d for filled honeypot, got %d", http.StatusBadRequest, rec.Code)
		}
	})
}
