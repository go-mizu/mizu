package maintenance

import (
	"strings"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(true))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}
	if rec.Body.String() != "Service is under maintenance" {
		t.Errorf("expected maintenance message, got %q", rec.Body.String())
	}
}

func TestNew_Disabled(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(false))

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

func TestWithOptions_CustomMessage(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Enabled: true,
		Message: "We'll be back soon!",
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "We'll be back soon!" {
		t.Errorf("expected custom message, got %q", rec.Body.String())
	}
}

func TestWithOptions_RetryAfter(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Enabled:    true,
		RetryAfter: 7200,
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Retry-After") != "7200" {
		t.Errorf("expected Retry-After: 7200, got %q", rec.Header().Get("Retry-After"))
	}
}

func TestWithOptions_Whitelist(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Enabled:   true,
		Whitelist: []string{"192.168.1.1"},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("whitelisted IP", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Real-IP", "192.168.1.1")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d for whitelisted IP, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("non-whitelisted IP", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Real-IP", "10.0.0.1")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("expected %d for non-whitelisted IP, got %d", http.StatusServiceUnavailable, rec.Code)
		}
	})
}

func TestWithOptions_WhitelistPaths(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Enabled:        true,
		WhitelistPaths: []string{"/health", "/status"},
	}))

	app.Get("/health", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "healthy")
	})
	app.Get("/api", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "api")
	})

	t.Run("whitelisted path", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d for whitelisted path, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("non-whitelisted path", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("expected %d for non-whitelisted path, got %d", http.StatusServiceUnavailable, rec.Code)
		}
	})
}

func TestWithOptions_CustomHandler(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Enabled: true,
		Handler: func(c *mizu.Ctx) error {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{
				"status":  "maintenance",
				"message": "Custom handler",
			})
		},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !strings.Contains(rec.Header().Get("Content-Type"), "application/json") {
		t.Error("expected JSON response from custom handler")
	}
}

func TestMode(t *testing.T) {
	mode := NewMode(Options{})

	if mode.IsEnabled() {
		t.Error("expected disabled by default")
	}

	mode.Enable()
	if !mode.IsEnabled() {
		t.Error("expected enabled after Enable()")
	}

	mode.Disable()
	if mode.IsEnabled() {
		t.Error("expected disabled after Disable()")
	}

	mode.Toggle()
	if !mode.IsEnabled() {
		t.Error("expected enabled after Toggle()")
	}
}

func TestMode_Middleware(t *testing.T) {
	mode := NewMode(Options{})

	app := mizu.NewRouter()
	app.Use(mode.Middleware())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("disabled", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d when disabled, got %d", http.StatusOK, rec.Code)
		}
	})

	mode.Enable()

	t.Run("enabled", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("expected %d when enabled, got %d", http.StatusServiceUnavailable, rec.Code)
		}
	})
}

func TestScheduledMaintenance(t *testing.T) {
	// Schedule maintenance in the past (already over)
	past := time.Now().Add(-time.Hour)
	pastEnd := time.Now().Add(-time.Minute)

	app := mizu.NewRouter()
	app.Use(ScheduledMaintenance(past, pastEnd))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Should be OK since maintenance period has passed
	if rec.Code != http.StatusOK {
		t.Errorf("expected %d after maintenance period, got %d", http.StatusOK, rec.Code)
	}
}
