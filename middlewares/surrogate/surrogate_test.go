package surrogate

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Get("/", func(c *mizu.Ctx) error {
		Add(c, "page", "content")
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	surrogateKey := rec.Header().Get("Surrogate-Key")
	if !strings.Contains(surrogateKey, "page") {
		t.Errorf("expected 'page' in Surrogate-Key, got %q", surrogateKey)
	}
	if !strings.Contains(surrogateKey, "content") {
		t.Errorf("expected 'content' in Surrogate-Key, got %q", surrogateKey)
	}
}

func TestWithOptions_DefaultKeys(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		DefaultKeys: []string{"site", "global"},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	surrogateKey := rec.Header().Get("Surrogate-Key")
	if !strings.Contains(surrogateKey, "site") {
		t.Error("expected default key 'site'")
	}
	if !strings.Contains(surrogateKey, "global") {
		t.Error("expected default key 'global'")
	}
}

func TestWithOptions_CustomHeader(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Header: "xkey",
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		Add(c, "custom")
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("xkey") == "" {
		t.Error("expected xkey header")
	}
}

func TestGet(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	var keys *Keys
	app.Get("/", func(c *mizu.Ctx) error {
		keys = Get(c)
		keys.Add("test-key")
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if len(keys.Get()) != 1 || keys.Get()[0] != "test-key" {
		t.Errorf("expected [test-key], got %v", keys.Get())
	}
}

func TestAdd(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Get("/", func(c *mizu.Ctx) error {
		Add(c, "key1", "key2")
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	surrogateKey := rec.Header().Get("Surrogate-Key")
	if !strings.Contains(surrogateKey, "key1") || !strings.Contains(surrogateKey, "key2") {
		t.Errorf("expected key1 and key2, got %q", surrogateKey)
	}
}

func TestClear(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		DefaultKeys: []string{"default"},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		Clear(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	surrogateKey := rec.Header().Get("Surrogate-Key")
	if surrogateKey != "" {
		t.Errorf("expected empty after clear, got %q", surrogateKey)
	}
}

func TestWithKeys(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithKeys("site", "section"))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	surrogateKey := rec.Header().Get("Surrogate-Key")
	if !strings.Contains(surrogateKey, "site") {
		t.Error("expected 'site' key")
	}
}

func TestFastly(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Fastly())

	app.Get("/", func(c *mizu.Ctx) error {
		Add(c, "fastly-key")
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Surrogate-Key") == "" {
		t.Error("expected Surrogate-Key for Fastly")
	}
}

func TestVarnish(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Varnish())

	app.Get("/", func(c *mizu.Ctx) error {
		Add(c, "varnish-key")
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("xkey") == "" {
		t.Error("expected xkey for Varnish")
	}
}

func TestNoKeys(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Get("/", func(c *mizu.Ctx) error {
		// Don't add any keys
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Should not set header if no keys
	if rec.Header().Get("Surrogate-Key") != "" {
		t.Error("expected no Surrogate-Key header when no keys added")
	}
}

func TestKeysCleared(t *testing.T) {
	keys := &Keys{}
	keys.Add("a", "b")

	if len(keys.Get()) != 2 {
		t.Errorf("expected 2 keys, got %d", len(keys.Get()))
	}

	keys.Clear()

	if len(keys.Get()) != 0 {
		t.Errorf("expected 0 keys after clear, got %d", len(keys.Get()))
	}
}
