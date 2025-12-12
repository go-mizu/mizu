package canary

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(50)) // 50% canary

	var canaryCount int
	var totalCount int
	app.Get("/", func(c *mizu.Ctx) error {
		totalCount++
		if IsCanary(c) {
			canaryCount++
		}
		return c.Text(http.StatusOK, "ok")
	})

	// Send 100 requests
	for i := 0; i < 100; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}

	// Should be roughly 50% canary (allow some variance)
	if canaryCount < 40 || canaryCount > 60 {
		t.Errorf("expected ~50%% canary, got %d%%", canaryCount)
	}
}

func TestWithOptions_Header(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Percentage: 0, // No random canary
		Header:     "X-Canary",
	}))

	var isCanary bool
	app.Get("/", func(c *mizu.Ctx) error {
		isCanary = IsCanary(c)
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("with canary header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Canary", "true")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if !isCanary {
			t.Error("expected canary with header")
		}
	})

	t.Run("without canary header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if isCanary {
			t.Error("expected non-canary without header")
		}
	})
}

func TestWithOptions_Cookie(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Percentage: 0,
		Cookie:     "canary",
	}))

	var isCanary bool
	app.Get("/", func(c *mizu.Ctx) error {
		isCanary = IsCanary(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "canary", Value: "true"})
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !isCanary {
		t.Error("expected canary with cookie")
	}
}

func TestWithOptions_Selector(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Percentage: 0,
		Selector: func(c *mizu.Ctx) bool {
			return c.Request().Header.Get("User-Agent") == "Canary"
		},
	}))

	var isCanary bool
	app.Get("/", func(c *mizu.Ctx) error {
		isCanary = IsCanary(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("User-Agent", "Canary")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !isCanary {
		t.Error("expected canary with custom selector")
	}
}

func TestRoute(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(0)) // Force non-canary

	app.Get("/", Route(
		func(c *mizu.Ctx) error {
			return c.Text(http.StatusOK, "canary")
		},
		func(c *mizu.Ctx) error {
			return c.Text(http.StatusOK, "stable")
		},
	))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "stable" {
		t.Errorf("expected 'stable', got %q", rec.Body.String())
	}
}

func TestRoute_Canary(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Percentage: 0,
		Header:     "X-Canary",
	}))

	app.Get("/", Route(
		func(c *mizu.Ctx) error {
			return c.Text(http.StatusOK, "canary")
		},
		func(c *mizu.Ctx) error {
			return c.Text(http.StatusOK, "stable")
		},
	))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Canary", "true")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "canary" {
		t.Errorf("expected 'canary', got %q", rec.Body.String())
	}
}

func TestMiddleware(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Percentage: 0,
		Header:     "X-Canary",
	}))

	canaryMw := func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			c.Writer().Header().Set("X-Version", "canary")
			return next(c)
		}
	}
	stableMw := func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			c.Writer().Header().Set("X-Version", "stable")
			return next(c)
		}
	}

	app.Use(Middleware(canaryMw, stableMw))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("stable middleware", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Header().Get("X-Version") != "stable" {
			t.Error("expected stable middleware")
		}
	})

	t.Run("canary middleware", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Canary", "true")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Header().Get("X-Version") != "canary" {
			t.Error("expected canary middleware")
		}
	})
}

func TestReleaseManager(t *testing.T) {
	manager := NewReleaseManager()
	manager.Set("feature-x", 50)

	if manager.Get("feature-x") == nil {
		t.Error("expected release")
	}
	if manager.Get("feature-y") != nil {
		t.Error("expected nil for non-existent release")
	}
}

func TestReleaseManager_ShouldUseCanary(t *testing.T) {
	manager := NewReleaseManager()
	manager.Set("feature", 50)

	var canaryCount int
	for i := 0; i < 100; i++ {
		if manager.ShouldUseCanary("feature") {
			canaryCount++
		}
	}

	// Should be roughly 50%
	if canaryCount < 40 || canaryCount > 60 {
		t.Errorf("expected ~50%% canary, got %d%%", canaryCount)
	}
}

func TestHeaderSelector(t *testing.T) {
	selector := HeaderSelector("X-Beta", "true")

	app := mizu.NewRouter()
	var selected bool
	app.Get("/", func(c *mizu.Ctx) error {
		selected = selector(c)
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("matching header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Beta", "true")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if !selected {
			t.Error("expected selection with matching header")
		}
	})

	t.Run("non-matching header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Beta", "false")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if selected {
			t.Error("expected no selection with non-matching header")
		}
	})
}

func TestCookieSelector(t *testing.T) {
	selector := CookieSelector("beta_user", "yes")

	app := mizu.NewRouter()
	var selected bool
	app.Get("/", func(c *mizu.Ctx) error {
		selected = selector(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "beta_user", Value: "yes"})
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !selected {
		t.Error("expected selection with matching cookie")
	}
}
