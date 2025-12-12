package nonce

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

	var nonce string
	app.Get("/", func(c *mizu.Ctx) error {
		nonce = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if nonce == "" {
		t.Error("expected nonce to be generated")
	}

	csp := rec.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Error("expected CSP header to be set")
	}

	if !strings.Contains(csp, "nonce-"+nonce) {
		t.Errorf("expected CSP to contain nonce, got %q", csp)
	}
}

func TestWithOptions_CustomLength(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{Length: 32}))

	var nonce string
	app.Get("/", func(c *mizu.Ctx) error {
		nonce = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// 32 bytes -> ~43 base64 chars
	if len(nonce) < 40 {
		t.Errorf("expected longer nonce with 32 bytes, got %d chars", len(nonce))
	}
}

func TestWithOptions_CustomDirectives(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Directives: []string{"script-src"},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "script-src") {
		t.Error("expected script-src in CSP")
	}
	if strings.Contains(csp, "style-src") {
		t.Error("expected style-src to not be in CSP with custom directives")
	}
}

func TestWithOptions_BasePolicy(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		BasePolicy: "default-src 'self'; img-src *",
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "default-src 'self'") {
		t.Errorf("expected base policy in CSP, got %q", csp)
	}
	if !strings.Contains(csp, "img-src *") {
		t.Errorf("expected img-src in CSP, got %q", csp)
	}
}

func TestWithOptions_CustomGenerator(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Generator: func() (string, error) {
			return "fixed-nonce-value", nil
		},
	}))

	var nonce string
	app.Get("/", func(c *mizu.Ctx) error {
		nonce = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if nonce != "fixed-nonce-value" {
		t.Errorf("expected custom nonce, got %q", nonce)
	}
}

func TestScriptTag(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	var tag string
	app.Get("/", func(c *mizu.Ctx) error {
		tag = ScriptTag(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !strings.HasPrefix(tag, `nonce="`) {
		t.Errorf("expected nonce attribute, got %q", tag)
	}
	if !strings.HasSuffix(tag, `"`) {
		t.Errorf("expected nonce attribute to end with quote, got %q", tag)
	}
}

func TestStyleTag(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	var tag string
	app.Get("/", func(c *mizu.Ctx) error {
		tag = StyleTag(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if tag == "" {
		t.Error("expected style tag to have nonce")
	}
}

func TestForScripts(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(ForScripts())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "script-src") {
		t.Error("expected script-src in CSP")
	}
}

func TestForStyles(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(ForStyles())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "style-src") {
		t.Error("expected style-src in CSP")
	}
}

func TestReportOnly(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(ReportOnly())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy-Report-Only")
	if csp == "" {
		t.Error("expected CSP-Report-Only header")
	}
}

func TestUniqueNonce(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	var nonces []string
	app.Get("/", func(c *mizu.Ctx) error {
		nonces = append(nonces, Get(c))
		return c.Text(http.StatusOK, "ok")
	})

	// Make multiple requests
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}

	// Check all nonces are unique
	seen := make(map[string]bool)
	for _, n := range nonces {
		if seen[n] {
			t.Errorf("nonce %q was repeated", n)
		}
		seen[n] = true
	}
}

func TestGetWithoutMiddleware(t *testing.T) {
	app := mizu.NewRouter()

	var nonce string
	app.Get("/", func(c *mizu.Ctx) error {
		nonce = Get(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if nonce != "" {
		t.Error("expected empty nonce without middleware")
	}
}
