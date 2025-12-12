package vary

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New("Accept-Encoding", "Accept-Language"))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	vary := rec.Header().Get("Vary")
	if !strings.Contains(vary, "Accept-Encoding") {
		t.Errorf("expected Accept-Encoding in Vary, got %q", vary)
	}
	if !strings.Contains(vary, "Accept-Language") {
		t.Errorf("expected Accept-Language in Vary, got %q", vary)
	}
}

func TestNoDuplicates(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New("Accept-Encoding", "Accept-Encoding"))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	vary := rec.Header().Get("Vary")
	count := strings.Count(vary, "Accept-Encoding")
	if count != 1 {
		t.Errorf("expected no duplicates, got %d occurrences in %q", count, vary)
	}
}

func TestWithOptions_Auto(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{Auto: true}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	vary := rec.Header().Get("Vary")
	if !strings.Contains(vary, "Accept") {
		t.Errorf("expected Accept in auto Vary, got %q", vary)
	}
	if !strings.Contains(vary, "Accept-Encoding") {
		t.Errorf("expected Accept-Encoding in auto Vary, got %q", vary)
	}
}

func TestAdd(t *testing.T) {
	app := mizu.NewRouter()

	app.Get("/", func(c *mizu.Ctx) error {
		Add(c, "X-Custom-Header")
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	vary := rec.Header().Get("Vary")
	if vary != "X-Custom-Header" {
		t.Errorf("expected X-Custom-Header, got %q", vary)
	}
}

func TestAcceptEncoding(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(AcceptEncoding())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Vary") != "Accept-Encoding" {
		t.Errorf("expected Accept-Encoding, got %q", rec.Header().Get("Vary"))
	}
}

func TestAccept(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Accept())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Vary") != "Accept" {
		t.Errorf("expected Accept, got %q", rec.Header().Get("Vary"))
	}
}

func TestAcceptLanguage(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(AcceptLanguage())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Vary") != "Accept-Language" {
		t.Errorf("expected Accept-Language, got %q", rec.Header().Get("Vary"))
	}
}

func TestOrigin(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Origin())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Vary") != "Origin" {
		t.Errorf("expected Origin, got %q", rec.Header().Get("Vary"))
	}
}

func TestAll(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(All())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	vary := rec.Header().Get("Vary")
	if !strings.Contains(vary, "Accept") {
		t.Error("expected Accept in All()")
	}
	if !strings.Contains(vary, "Accept-Encoding") {
		t.Error("expected Accept-Encoding in All()")
	}
	if !strings.Contains(vary, "Accept-Language") {
		t.Error("expected Accept-Language in All()")
	}
}

func TestAuto(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Auto())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Language", "en-US")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	vary := rec.Header().Get("Vary")
	if !strings.Contains(vary, "Accept-Language") {
		t.Errorf("expected auto-detected Accept-Language, got %q", vary)
	}
}

func TestCombineWithExisting(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New("Accept-Encoding"))

	app.Get("/", func(c *mizu.Ctx) error {
		c.Header().Set("Vary", "X-Custom")
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	vary := rec.Header().Get("Vary")
	if !strings.Contains(vary, "X-Custom") {
		t.Error("expected X-Custom to be preserved")
	}
	if !strings.Contains(vary, "Accept-Encoding") {
		t.Error("expected Accept-Encoding to be added")
	}
}
