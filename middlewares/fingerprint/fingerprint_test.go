package fingerprint

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	var info *Info
	app.Get("/", func(c *mizu.Ctx) error {
		info = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("User-Agent", "TestAgent/1.0")
	req.Header.Set("Accept", "text/html")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if info.Hash == "" {
		t.Error("expected fingerprint hash to be generated")
	}
	if info.Components["User-Agent"] != "TestAgent/1.0" {
		t.Errorf("expected User-Agent in components, got %v", info.Components)
	}
}

func TestWithOptions_IncludeIP(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{IncludeIP: true}))

	var info *Info
	app.Get("/", func(c *mizu.Ctx) error {
		info = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if info.Components["IP"] != "192.168.1.1:12345" {
		t.Errorf("expected IP in components, got %q", info.Components["IP"])
	}
}

func TestWithOptions_IncludeMethod(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{IncludeMethod: true}))

	var info *Info
	app.Get("/", func(c *mizu.Ctx) error {
		info = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if info.Components["Method"] != "GET" {
		t.Errorf("expected Method in components, got %q", info.Components["Method"])
	}
}

func TestWithOptions_IncludePath(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{IncludePath: true}))

	var info *Info
	app.Get("/test/path", func(c *mizu.Ctx) error {
		info = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test/path", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if info.Components["Path"] != "/test/path" {
		t.Errorf("expected Path in components, got %q", info.Components["Path"])
	}
}

func TestWithOptions_Custom(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Custom: func(c *mizu.Ctx) map[string]string {
			return map[string]string{
				"CustomKey": "CustomValue",
			}
		},
	}))

	var info *Info
	app.Get("/", func(c *mizu.Ctx) error {
		info = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if info.Components["CustomKey"] != "CustomValue" {
		t.Errorf("expected custom component, got %v", info.Components)
	}
}

func TestHash(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	var hash string
	app.Get("/", func(c *mizu.Ctx) error {
		hash = Hash(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("User-Agent", "TestAgent/1.0")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if hash == "" {
		t.Error("expected hash to be returned")
	}
	if len(hash) != 64 { // SHA256 hex
		t.Errorf("expected 64-char hash, got %d", len(hash))
	}
}

func TestConsistentHash(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	var hash1, hash2 string
	app.Get("/", func(c *mizu.Ctx) error {
		h := Hash(c)
		if hash1 == "" {
			hash1 = h
		} else {
			hash2 = h
		}
		return c.Text(http.StatusOK, "ok")
	})

	// First request
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("User-Agent", "TestAgent/1.0")
	req.Header.Set("Accept", "text/html")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Second request with same headers
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("User-Agent", "TestAgent/1.0")
	req.Header.Set("Accept", "text/html")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if hash1 != hash2 {
		t.Errorf("expected consistent hash, got %q and %q", hash1, hash2)
	}
}

func TestHeadersOnly(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(HeadersOnly("X-Custom-Header"))

	var info *Info
	app.Get("/", func(c *mizu.Ctx) error {
		info = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Custom-Header", "custom-value")
	req.Header.Set("User-Agent", "should-not-be-included")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if info.Components["X-Custom-Header"] != "custom-value" {
		t.Error("expected custom header in components")
	}
	if _, exists := info.Components["User-Agent"]; exists {
		t.Error("expected User-Agent to not be in components")
	}
}

func TestWithIP(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithIP())

	var info *Info
	app.Get("/", func(c *mizu.Ctx) error {
		info = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:8080"
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if info.Components["IP"] == "" {
		t.Error("expected IP in components with WithIP()")
	}
}

func TestFull(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Full())

	var info *Info
	app.Get("/test", func(c *mizu.Ctx) error {
		info = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "10.0.0.1:8080"
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if info.Components["IP"] == "" {
		t.Error("expected IP in full fingerprint")
	}
	if info.Components["Method"] != "GET" {
		t.Error("expected Method in full fingerprint")
	}
	if info.Components["Path"] != "/test" {
		t.Error("expected Path in full fingerprint")
	}
}

func TestXForwardedFor(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{IncludeIP: true}))

	var info *Info
	app.Get("/", func(c *mizu.Ctx) error {
		info = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "1.2.3.4, 5.6.7.8")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if info.Components["IP"] != "1.2.3.4" {
		t.Errorf("expected first IP from X-Forwarded-For, got %q", info.Components["IP"])
	}
}
