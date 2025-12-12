package chaos

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Default chaos does nothing
	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestError(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Error(100, http.StatusServiceUnavailable)) // 100% error rate

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}
}

func TestLatency(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Latency(50*time.Millisecond, 100*time.Millisecond))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	start := time.Now()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	elapsed := time.Since(start)

	if elapsed < 50*time.Millisecond {
		t.Errorf("expected latency >= 50ms, got %v", elapsed)
	}
}

func TestWithOptions_Disabled(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Enabled:   false,
		ErrorRate: 100,
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Should not inject error when disabled
	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestWithOptions_Selector(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Enabled:   true,
		ErrorRate: 100,
		Selector: func(c *mizu.Ctx) bool {
			return c.Request().URL.Path == "/chaos"
		},
	}))

	app.Get("/chaos", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})
	app.Get("/normal", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("chaos path", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/chaos", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected error on chaos path, got %d", rec.Code)
		}
	})

	t.Run("normal path", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/normal", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d on normal path, got %d", http.StatusOK, rec.Code)
		}
	})
}

func TestController(t *testing.T) {
	ctrl := NewController()

	if ctrl.IsEnabled() {
		t.Error("expected disabled by default")
	}

	ctrl.Enable()
	if !ctrl.IsEnabled() {
		t.Error("expected enabled after Enable()")
	}

	ctrl.Disable()
	if ctrl.IsEnabled() {
		t.Error("expected disabled after Disable()")
	}
}

func TestController_Middleware(t *testing.T) {
	ctrl := NewController()
	ctrl.SetErrorRate(100)

	app := mizu.NewRouter()
	app.Use(ctrl.Middleware())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("disabled", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Error("expected OK when disabled")
		}
	})

	ctrl.Enable()

	t.Run("enabled", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Error("expected error when enabled")
		}
	})
}

func TestController_SetErrorCode(t *testing.T) {
	ctrl := NewController()
	ctrl.Enable()
	ctrl.SetErrorRate(100)
	ctrl.SetErrorCode(http.StatusBadGateway)

	app := mizu.NewRouter()
	app.Use(ctrl.Middleware())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Errorf("expected %d, got %d", http.StatusBadGateway, rec.Code)
	}
}

func TestPathSelector(t *testing.T) {
	selector := PathSelector("/api/chaos", "/test/chaos")

	app := mizu.NewRouter()
	var selected bool
	app.Get("/api/chaos", func(c *mizu.Ctx) error {
		selected = selector(c)
		return c.Text(http.StatusOK, "ok")
	})
	app.Get("/api/normal", func(c *mizu.Ctx) error {
		selected = selector(c)
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("matching path", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/chaos", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if !selected {
			t.Error("expected path to be selected")
		}
	})

	t.Run("non-matching path", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/normal", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if selected {
			t.Error("expected path to not be selected")
		}
	})
}

func TestHeaderSelector(t *testing.T) {
	selector := HeaderSelector("X-Chaos")

	app := mizu.NewRouter()
	var selected bool
	app.Get("/", func(c *mizu.Ctx) error {
		selected = selector(c)
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("with header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Chaos", "true")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if !selected {
			t.Error("expected selection with header")
		}
	})

	t.Run("without header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if selected {
			t.Error("expected no selection without header")
		}
	})
}

func TestMethodSelector(t *testing.T) {
	selector := MethodSelector("POST", "PUT")

	app := mizu.NewRouter()
	var selected bool
	app.Post("/", func(c *mizu.Ctx) error {
		selected = selector(c)
		return c.Text(http.StatusOK, "ok")
	})
	app.Get("/", func(c *mizu.Ctx) error {
		selected = selector(c)
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("matching method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if !selected {
			t.Error("expected POST to be selected")
		}
	})

	t.Run("non-matching method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if selected {
			t.Error("expected GET to not be selected")
		}
	})
}
