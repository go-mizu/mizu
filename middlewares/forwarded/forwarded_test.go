package forwarded

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
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Host", "example.com")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if info == nil {
		t.Fatal("expected info")
	}
	if info.For != "1.2.3.4" {
		t.Errorf("expected For '1.2.3.4', got %q", info.For)
	}
	if info.Proto != "https" {
		t.Errorf("expected Proto 'https', got %q", info.Proto)
	}
	if info.Host != "example.com" {
		t.Errorf("expected Host 'example.com', got %q", info.Host)
	}
}

func TestWithOptions_TrustedProxies(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		TrustProxy:     true,
		TrustedProxies: []string{"192.168.1.0/24"},
	}))

	var info *Info
	app.Get("/", func(c *mizu.Ctx) error {
		info = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("trusted proxy", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		req.Header.Set("X-Forwarded-For", "10.0.0.1")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if info.For != "10.0.0.1" {
			t.Errorf("expected For '10.0.0.1', got %q", info.For)
		}
	})

	t.Run("untrusted proxy", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.50:12345"
		req.Header.Set("X-Forwarded-For", "1.1.1.1")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		// Should use RemoteAddr since proxy is not trusted
		if info.For != "10.0.0.50" {
			t.Errorf("expected For '10.0.0.50', got %q", info.For)
		}
	})
}

func TestWithOptions_XForwardedForMultiple(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	var info *Info
	app.Get("/", func(c *mizu.Ctx) error {
		info = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "1.2.3.4, 5.6.7.8, 9.10.11.12")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Should use the first (original client) IP
	if info.For != "1.2.3.4" {
		t.Errorf("expected For '1.2.3.4', got %q", info.For)
	}
}

func TestWithOptions_ForwardedHeader(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	var info *Info
	app.Get("/", func(c *mizu.Ctx) error {
		info = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Forwarded", `for=192.0.2.60;proto=https;host=example.org`)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if info.For != "192.0.2.60" {
		t.Errorf("expected For '192.0.2.60', got %q", info.For)
	}
	if info.Proto != "https" {
		t.Errorf("expected Proto 'https', got %q", info.Proto)
	}
	if info.Host != "example.org" {
		t.Errorf("expected Host 'example.org', got %q", info.Host)
	}
}

func TestFromContext(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	var info1, info2 *Info
	app.Get("/", func(c *mizu.Ctx) error {
		info1 = Get(c)
		info2 = FromContext(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if info1 != info2 {
		t.Error("Get and FromContext should return same info")
	}
}

func TestClientIP(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	var ip string
	app.Get("/", func(c *mizu.Ctx) error {
		clientIP := ClientIP(c)
		if clientIP != nil {
			ip = clientIP.String()
		}
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if ip != "1.2.3.4" {
		t.Errorf("expected '1.2.3.4', got %q", ip)
	}
}

func TestProto(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	var proto string
	app.Get("/", func(c *mizu.Ctx) error {
		proto = Proto(c)
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("with proto header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Forwarded-Proto", "https")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if proto != "https" {
			t.Errorf("expected 'https', got %q", proto)
		}
	})

	t.Run("without proto header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if proto != "http" {
			t.Errorf("expected 'http', got %q", proto)
		}
	})
}

func TestHost(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	var host string
	app.Get("/", func(c *mizu.Ctx) error {
		host = Host(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-Host", "api.example.com")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if host != "api.example.com" {
		t.Errorf("expected 'api.example.com', got %q", host)
	}
}

func TestWithOptions_DisableTrustProxy(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		TrustProxy: false,
	}))

	var info *Info
	app.Get("/", func(c *mizu.Ctx) error {
		info = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Should ignore X-Forwarded-For when TrustProxy is false
	if info.For != "192.168.1.1" {
		t.Errorf("expected For '192.168.1.1', got %q", info.For)
	}
}
