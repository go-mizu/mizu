package basicauth

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-mizu/mizu"
)

func basicAuthHeader(user, pass string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+pass))
}

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(map[string]string{
		"admin": "secret",
		"user":  "password",
	}))

	app.Get("/protected", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "protected")
	})

	t.Run("allows valid credentials", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", basicAuthHeader("admin", "secret"))
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("rejects invalid password", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", basicAuthHeader("admin", "wrong"))
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})

	t.Run("rejects unknown user", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", basicAuthHeader("unknown", "password"))
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})

	t.Run("rejects missing auth", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
		}

		if rec.Header().Get("WWW-Authenticate") == "" {
			t.Error("expected WWW-Authenticate header")
		}
	})
}

func TestWithValidator(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithValidator(func(user, pass string) bool {
		return user == "test" && pass == "test123"
	}))

	app.Get("/protected", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", basicAuthHeader("test", "test123"))
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestWithRealm(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithRealm("Admin Area", map[string]string{
		"admin": "pass",
	}))

	app.Get("/protected", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	wwwAuth := rec.Header().Get("WWW-Authenticate")
	if wwwAuth != `Basic realm="Admin Area"` {
		t.Errorf("expected realm 'Admin Area', got %q", wwwAuth)
	}
}

func TestWithOptions_ErrorHandler(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Validator: func(user, pass string) bool {
			return false
		},
		ErrorHandler: func(c *mizu.Ctx) error {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": "authentication required",
			})
		},
	}))

	app.Get("/protected", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestWithOptions_InvalidAuth(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(map[string]string{"admin": "pass"}))

	app.Get("/protected", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	tests := []struct {
		name string
		auth string
	}{
		{"not basic", "Bearer token"},
		{"invalid base64", "Basic !!!invalid"},
		{"no colon", "Basic " + base64.StdEncoding.EncodeToString([]byte("nocolon"))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			req.Header.Set("Authorization", tt.auth)
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
			}
		})
	}
}

func TestWithOptions_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic without validator")
		}
	}()
	WithOptions(Options{})
}

func TestSecureCompare(t *testing.T) {
	tests := []struct {
		a, b     string
		expected bool
	}{
		{"password", "password", true},
		{"password", "different", false},
		{"short", "longpassword", false},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.a+"_"+tt.b, func(t *testing.T) {
			if got := secureCompare(tt.a, tt.b); got != tt.expected {
				t.Errorf("secureCompare(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.expected)
			}
		})
	}
}
