package realip

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	var capturedIP string
	app.Get("/test", func(c *mizu.Ctx) error {
		capturedIP = FromContext(c)
		return c.Text(http.StatusOK, capturedIP)
	})

	t.Run("uses X-Forwarded-For", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		req.Header.Set("X-Forwarded-For", "203.0.113.195, 70.41.3.18, 150.172.238.178")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if capturedIP != "203.0.113.195" {
			t.Errorf("expected '203.0.113.195', got %q", capturedIP)
		}
	})

	t.Run("uses X-Real-IP", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		req.Header.Set("X-Real-IP", "198.51.100.178")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if capturedIP != "198.51.100.178" {
			t.Errorf("expected '198.51.100.178', got %q", capturedIP)
		}
	})

	t.Run("uses CF-Connecting-IP", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		req.Header.Set("CF-Connecting-IP", "192.0.2.1")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if capturedIP != "192.0.2.1" {
			t.Errorf("expected '192.0.2.1', got %q", capturedIP)
		}
	})

	t.Run("falls back to RemoteAddr", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.100:5678"
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if capturedIP != "192.168.1.100" {
			t.Errorf("expected '192.168.1.100', got %q", capturedIP)
		}
	})
}

func TestWithTrustedProxies(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithTrustedProxies("10.0.0.0/8"))

	var capturedIP string
	app.Get("/test", func(c *mizu.Ctx) error {
		capturedIP = FromContext(c)
		return c.Text(http.StatusOK, "ok")
	})

	t.Run("trusts headers from trusted proxy", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		req.Header.Set("X-Forwarded-For", "203.0.113.195")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if capturedIP != "203.0.113.195" {
			t.Errorf("expected '203.0.113.195', got %q", capturedIP)
		}
	})

	t.Run("ignores headers from untrusted source", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:1234" // Not in 10.0.0.0/8
		req.Header.Set("X-Forwarded-For", "203.0.113.195")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if capturedIP != "192.168.1.1" {
			t.Errorf("expected '192.168.1.1', got %q", capturedIP)
		}
	})
}

func TestWithTrustedProxies_SingleIP(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithTrustedProxies("10.0.0.1"))

	var capturedIP string
	app.Get("/test", func(c *mizu.Ctx) error {
		capturedIP = FromContext(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("X-Real-IP", "1.2.3.4")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if capturedIP != "1.2.3.4" {
		t.Errorf("expected '1.2.3.4', got %q", capturedIP)
	}
}

func TestWithOptions_CustomHeaders(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		TrustedHeaders: []string{"X-Custom-IP"},
	}))

	var capturedIP string
	app.Get("/test", func(c *mizu.Ctx) error {
		capturedIP = FromContext(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("X-Custom-IP", "5.6.7.8")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if capturedIP != "5.6.7.8" {
		t.Errorf("expected '5.6.7.8', got %q", capturedIP)
	}
}

func TestGet(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	var ip1, ip2 string
	app.Get("/test", func(c *mizu.Ctx) error {
		ip1 = FromContext(c)
		ip2 = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if ip1 != ip2 {
		t.Errorf("FromContext and Get should return same: %q vs %q", ip1, ip2)
	}
}

func TestExtractFirstIP(t *testing.T) {
	tests := []struct {
		header   string
		expected string
	}{
		{"203.0.113.195", "203.0.113.195"},
		{"203.0.113.195, 70.41.3.18, 150.172.238.178", "203.0.113.195"},
		{"  203.0.113.195  ", "203.0.113.195"},
		{"invalid", ""},
		{"", ""},
		{"2001:db8::1", "2001:db8::1"},
	}

	for _, tt := range tests {
		t.Run(tt.header, func(t *testing.T) {
			got := extractFirstIP(tt.header)
			if got != tt.expected {
				t.Errorf("extractFirstIP(%q) = %q, want %q", tt.header, got, tt.expected)
			}
		})
	}
}

func TestExtractIP(t *testing.T) {
	tests := []struct {
		addr     string
		expected string
	}{
		{"192.168.1.1:8080", "192.168.1.1"},
		{"192.168.1.1", "192.168.1.1"},
		{"[::1]:8080", "::1"},
	}

	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			got := extractIP(tt.addr)
			if got != tt.expected {
				t.Errorf("extractIP(%q) = %q, want %q", tt.addr, got, tt.expected)
			}
		})
	}
}

func TestIsTrusted(t *testing.T) {
	tests := []struct {
		ip       string
		networks []string
		expected bool
	}{
		{"10.0.0.1", []string{"10.0.0.0/8"}, true},
		{"10.255.255.255", []string{"10.0.0.0/8"}, true},
		{"11.0.0.1", []string{"10.0.0.0/8"}, false},
		{"192.168.1.1", []string{"10.0.0.0/8", "192.168.0.0/16"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			var networks []*net.IPNet
			for _, n := range tt.networks {
				_, network, _ := net.ParseCIDR(n)
				networks = append(networks, network)
			}
			got := isTrusted(tt.ip, networks)
			if got != tt.expected {
				t.Errorf("isTrusted(%q) = %v, want %v", tt.ip, got, tt.expected)
			}
		})
	}
}
