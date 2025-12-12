package rewrite

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(
		Prefix("/old", "/new"),
	))

	var rewrittenPath string
	app.Get("/new/path", func(c *mizu.Ctx) error {
		rewrittenPath = c.Request().URL.Path
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/old/path", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rewrittenPath != "/new/path" {
		t.Errorf("expected /new/path, got %s", rewrittenPath)
	}
}

func TestWithOptions_Regex(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		Rules: []Rule{
			Regex(`^/api/v(\d+)/(.*)$`, "/api/v$1/$2"),
		},
	}))

	var capturedPath string
	app.Get("/api/v2/users", func(c *mizu.Ctx) error {
		capturedPath = c.Request().URL.Path
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v2/users", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if capturedPath != "/api/v2/users" {
		t.Errorf("expected '/api/v2/users', got %q", capturedPath)
	}
}

func TestWithOptions_MultipleRules(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(
		Prefix("/blog", "/articles"),
		Prefix("/articles", "/posts"),
	))

	var capturedPath string
	app.Get("/articles/hello", func(c *mizu.Ctx) error {
		capturedPath = c.Request().URL.Path
		return c.Text(http.StatusOK, "ok")
	})

	// First rule should match and stop
	req := httptest.NewRequest(http.MethodGet, "/blog/hello", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if capturedPath != "/articles/hello" {
		t.Errorf("expected /articles/hello, got %s", capturedPath)
	}
}

func TestWithOptions_NoMatch(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(
		Prefix("/api", "/internal"),
	))

	var capturedPath string
	app.Get("/public/file", func(c *mizu.Ctx) error {
		capturedPath = c.Request().URL.Path
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/public/file", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if capturedPath != "/public/file" {
		t.Errorf("expected /public/file unchanged, got %s", capturedPath)
	}
}

func TestPrefix(t *testing.T) {
	rule := Prefix("/old", "/new")
	if rule.Match != "/old" {
		t.Errorf("expected Match '/old', got %q", rule.Match)
	}
	if rule.Rewrite != "/new" {
		t.Errorf("expected Rewrite '/new', got %q", rule.Rewrite)
	}
	if rule.Regex {
		t.Error("expected Regex false")
	}
}

func TestRegex(t *testing.T) {
	rule := Regex(`^/(\w+)$`, "/page/$1")
	if rule.Match != `^/(\w+)$` {
		t.Errorf("expected Match pattern, got %q", rule.Match)
	}
	if rule.Rewrite != "/page/$1" {
		t.Errorf("expected Rewrite '/page/$1', got %q", rule.Rewrite)
	}
	if !rule.Regex {
		t.Error("expected Regex true")
	}
}

func TestWithOptions_RegexCapture(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(
		Regex(`^/user/(\d+)/profile$`, "/profiles/$1"),
	))

	var capturedPath string
	app.Get("/profiles/123", func(c *mizu.Ctx) error {
		capturedPath = c.Request().URL.Path
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/user/123/profile", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if capturedPath != "/profiles/123" {
		t.Errorf("expected /profiles/123, got %s", capturedPath)
	}
}
