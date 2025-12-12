package helmet

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-mizu/mizu"
)

func TestDefault(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Default())

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	tests := []struct {
		header   string
		expected string
	}{
		{"X-Content-Type-Options", "nosniff"},
		{"X-Frame-Options", "SAMEORIGIN"},
		{"X-DNS-Prefetch-Control", "off"},
		{"X-Download-Options", "noopen"},
		{"X-Permitted-Cross-Domain-Policies", "none"},
		{"Referrer-Policy", "strict-origin-when-cross-origin"},
		{"Cross-Origin-Opener-Policy", "same-origin"},
		{"Cross-Origin-Resource-Policy", "same-origin"},
		{"Origin-Agent-Cluster", "?1"},
	}

	for _, tt := range tests {
		t.Run(tt.header, func(t *testing.T) {
			got := rec.Header().Get(tt.header)
			if got != tt.expected {
				t.Errorf("%s = %q, want %q", tt.header, got, tt.expected)
			}
		})
	}
}

func TestContentSecurityPolicy(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(ContentSecurityPolicy("default-src 'self'"))

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Security-Policy") != "default-src 'self'" {
		t.Error("expected CSP header")
	}
}

func TestXFrameOptions(t *testing.T) {
	tests := []struct {
		value    string
		expected string
	}{
		{"DENY", "DENY"},
		{"SAMEORIGIN", "SAMEORIGIN"},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			app := mizu.NewRouter()
			app.Use(XFrameOptions(tt.value))

			app.Get("/test", func(c *mizu.Ctx) error {
				return c.Text(http.StatusOK, "ok")
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)

			if rec.Header().Get("X-Frame-Options") != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, rec.Header().Get("X-Frame-Options"))
			}
		})
	}
}

func TestStrictTransportSecurity(t *testing.T) {
	tests := []struct {
		name              string
		maxAge            time.Duration
		includeSubDomains bool
		preload           bool
		expected          string
	}{
		{
			"basic",
			365 * 24 * time.Hour,
			false,
			false,
			"max-age=31536000",
		},
		{
			"with subdomains",
			365 * 24 * time.Hour,
			true,
			false,
			"max-age=31536000; includeSubDomains",
		},
		{
			"with preload",
			365 * 24 * time.Hour,
			true,
			true,
			"max-age=31536000; includeSubDomains; preload",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := mizu.NewRouter()
			app.Use(StrictTransportSecurity(tt.maxAge, tt.includeSubDomains, tt.preload))

			app.Get("/test", func(c *mizu.Ctx) error {
				return c.Text(http.StatusOK, "ok")
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)

			got := rec.Header().Get("Strict-Transport-Security")
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestReferrerPolicy(t *testing.T) {
	policies := []string{
		"no-referrer",
		"no-referrer-when-downgrade",
		"origin",
		"origin-when-cross-origin",
		"same-origin",
		"strict-origin",
		"strict-origin-when-cross-origin",
		"unsafe-url",
	}

	for _, policy := range policies {
		t.Run(policy, func(t *testing.T) {
			app := mizu.NewRouter()
			app.Use(ReferrerPolicy(policy))

			app.Get("/test", func(c *mizu.Ctx) error {
				return c.Text(http.StatusOK, "ok")
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)

			if rec.Header().Get("Referrer-Policy") != policy {
				t.Errorf("expected %q", policy)
			}
		})
	}
}

func TestPermissionsPolicy(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(PermissionsPolicy("geolocation=(), microphone=()"))

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Permissions-Policy") != "geolocation=(), microphone=()" {
		t.Error("expected Permissions-Policy header")
	}
}

func TestCrossOriginPolicies(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(CrossOriginOpenerPolicy("same-origin"))
	app.Use(CrossOriginEmbedderPolicy("require-corp"))
	app.Use(CrossOriginResourcePolicy("same-site"))

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	tests := []struct {
		header   string
		expected string
	}{
		{"Cross-Origin-Opener-Policy", "same-origin"},
		{"Cross-Origin-Embedder-Policy", "require-corp"},
		{"Cross-Origin-Resource-Policy", "same-site"},
	}

	for _, tt := range tests {
		if rec.Header().Get(tt.header) != tt.expected {
			t.Errorf("%s = %q, want %q", tt.header, rec.Header().Get(tt.header), tt.expected)
		}
	}
}

func TestOriginAgentCluster(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(OriginAgentCluster())

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Origin-Agent-Cluster") != "?1" {
		t.Error("expected Origin-Agent-Cluster: ?1")
	}
}

func TestXDNSPrefetchControl(t *testing.T) {
	tests := []struct {
		on       bool
		expected string
	}{
		{true, "on"},
		{false, "off"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			app := mizu.NewRouter()
			app.Use(XDNSPrefetchControl(tt.on))

			app.Get("/test", func(c *mizu.Ctx) error {
				return c.Text(http.StatusOK, "ok")
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)

			if rec.Header().Get("X-DNS-Prefetch-Control") != tt.expected {
				t.Errorf("expected %q", tt.expected)
			}
		})
	}
}

func TestXDownloadOptions(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(XDownloadOptions())

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("X-Download-Options") != "noopen" {
		t.Error("expected X-Download-Options: noopen")
	}
}

func TestXPermittedCrossDomainPolicies(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(XPermittedCrossDomainPolicies("master-only"))

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("X-Permitted-Cross-Domain-Policies") != "master-only" {
		t.Error("expected X-Permitted-Cross-Domain-Policies: master-only")
	}
}

func TestXContentTypeOptions(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(XContentTypeOptions())

	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("expected X-Content-Type-Options: nosniff")
	}
}
