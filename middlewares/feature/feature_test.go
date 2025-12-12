package feature

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(Flags{
		"dark_mode":   {Name: "dark_mode", Enabled: true},
		"new_feature": {Name: "new_feature", Enabled: false},
	}))

	var darkModeEnabled, newFeatureEnabled bool
	app.Get("/", func(c *mizu.Ctx) error {
		darkModeEnabled = IsEnabled(c, "dark_mode")
		newFeatureEnabled = IsEnabled(c, "new_feature")
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !darkModeEnabled {
		t.Error("expected dark_mode to be enabled")
	}
	if newFeatureEnabled {
		t.Error("expected new_feature to be disabled")
	}
}

func TestIsDisabled(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(Flags{
		"feature": {Name: "feature", Enabled: false},
	}))

	var disabled bool
	app.Get("/", func(c *mizu.Ctx) error {
		disabled = IsDisabled(c, "feature")
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !disabled {
		t.Error("expected feature to be disabled")
	}
}

func TestGetFlags(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(Flags{
		"a": {Name: "a", Enabled: true},
		"b": {Name: "b", Enabled: false},
	}))

	var flags Flags
	app.Get("/", func(c *mizu.Ctx) error {
		flags = GetFlags(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if len(flags) != 2 {
		t.Errorf("expected 2 flags, got %d", len(flags))
	}
}

func TestGet(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(Flags{
		"test": {Name: "test", Enabled: true, Description: "Test flag"},
	}))

	var flag *Flag
	app.Get("/", func(c *mizu.Ctx) error {
		flag = Get(c, "test")
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if flag == nil {
		t.Fatal("expected flag")
	}
	if flag.Description != "Test flag" {
		t.Errorf("expected description 'Test flag', got %q", flag.Description)
	}
}

func TestRequire(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(Flags{
		"enabled_feature":  {Name: "enabled_feature", Enabled: true},
		"disabled_feature": {Name: "disabled_feature", Enabled: false},
	}))

	app.Get("/enabled", Require("enabled_feature", nil)(func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "enabled")
	}))
	app.Get("/disabled", Require("disabled_feature", nil)(func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "disabled")
	}))

	t.Run("enabled feature", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/enabled", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("disabled feature", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/disabled", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, rec.Code)
		}
	})
}

func TestRequire_CustomHandler(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(Flags{}))

	app.Get("/", Require("missing", func(c *mizu.Ctx) error {
		return c.Text(http.StatusForbidden, "feature disabled")
	})(func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestRequireAll(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(Flags{
		"feature_a": {Name: "feature_a", Enabled: true},
		"feature_b": {Name: "feature_b", Enabled: true},
	}))

	app.Get("/", RequireAll([]string{"feature_a", "feature_b"}, nil)(func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestRequireAny(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(Flags{
		"feature_a": {Name: "feature_a", Enabled: false},
		"feature_b": {Name: "feature_b", Enabled: true},
	}))

	app.Get("/", RequireAny([]string{"feature_a", "feature_b"}, nil)(func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d (one enabled), got %d", http.StatusOK, rec.Code)
	}
}

func TestMemoryProvider(t *testing.T) {
	provider := NewMemoryProvider()

	provider.Enable("feature_a")
	provider.Set("feature_b", false)
	provider.SetFlag(&Flag{Name: "feature_c", Enabled: true, Description: "Custom"})

	app := mizu.NewRouter()
	app.Use(WithProvider(provider))

	var aEnabled, bEnabled, cEnabled bool
	var cDesc string
	app.Get("/", func(c *mizu.Ctx) error {
		aEnabled = IsEnabled(c, "feature_a")
		bEnabled = IsEnabled(c, "feature_b")
		cEnabled = IsEnabled(c, "feature_c")
		if flag := Get(c, "feature_c"); flag != nil {
			cDesc = flag.Description
		}
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !aEnabled {
		t.Error("expected feature_a to be enabled")
	}
	if bEnabled {
		t.Error("expected feature_b to be disabled")
	}
	if !cEnabled {
		t.Error("expected feature_c to be enabled")
	}
	if cDesc != "Custom" {
		t.Errorf("expected description 'Custom', got %q", cDesc)
	}
}

func TestMemoryProvider_Toggle(t *testing.T) {
	provider := NewMemoryProvider()
	provider.Enable("feature")

	app := mizu.NewRouter()
	app.Use(WithProvider(provider))

	var enabled bool
	app.Get("/", func(c *mizu.Ctx) error {
		enabled = IsEnabled(c, "feature")
		return c.Text(http.StatusOK, "ok")
	})

	// Should be enabled
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if !enabled {
		t.Error("expected feature to be enabled")
	}

	// Toggle
	provider.Toggle("feature")

	// Should be disabled
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if enabled {
		t.Error("expected feature to be disabled after toggle")
	}
}

func TestMemoryProvider_Delete(t *testing.T) {
	provider := NewMemoryProvider()
	provider.Enable("feature")
	provider.Delete("feature")

	app := mizu.NewRouter()
	app.Use(WithProvider(provider))

	var enabled bool
	app.Get("/", func(c *mizu.Ctx) error {
		enabled = IsEnabled(c, "feature")
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if enabled {
		t.Error("expected feature to not exist")
	}
}
