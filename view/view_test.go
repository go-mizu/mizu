package view

import (
	"bytes"
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestNew(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		e := New(Config{})
		if e.cfg.Dir != "views" {
			t.Errorf("expected Dir='views', got %q", e.cfg.Dir)
		}
		if e.cfg.Extension != ".html" {
			t.Errorf("expected Extension='.html', got %q", e.cfg.Extension)
		}
		if e.cfg.DefaultLayout != "default" {
			t.Errorf("expected DefaultLayout='default', got %q", e.cfg.DefaultLayout)
		}
	})

	t.Run("custom config", func(t *testing.T) {
		e := New(Config{
			Dir:           "templates",
			Extension:     ".tmpl",
			DefaultLayout: "main",
			Development:   true,
		})
		if e.cfg.Dir != "templates" {
			t.Errorf("expected Dir='templates', got %q", e.cfg.Dir)
		}
		if e.cfg.Extension != ".tmpl" {
			t.Errorf("expected Extension='.tmpl', got %q", e.cfg.Extension)
		}
		if e.cfg.DefaultLayout != "main" {
			t.Errorf("expected DefaultLayout='main', got %q", e.cfg.DefaultLayout)
		}
		if !e.cfg.Development {
			t.Error("expected Development=true")
		}
	})

	t.Run("with custom funcs", func(t *testing.T) {
		e := New(Config{
			Funcs: template.FuncMap{
				"customFunc": func() string { return "custom" },
			},
		})
		if _, ok := e.funcs["customFunc"]; !ok {
			t.Error("custom function not added")
		}
	})
}

func TestEngine_Render(t *testing.T) {
	e := New(Config{
		Dir:         "testdata/views",
		Development: true,
	})

	t.Run("simple page", func(t *testing.T) {
		var buf bytes.Buffer
		err := e.Render(&buf, "simple", nil, NoLayout())
		if err != nil {
			t.Fatalf("render error: %v", err)
		}
		if !strings.Contains(buf.String(), "Simple page content") {
			t.Errorf("expected 'Simple page content' in output, got: %s", buf.String())
		}
	})

	t.Run("page with data", func(t *testing.T) {
		var buf bytes.Buffer
		err := e.Render(&buf, "home", Data{"Name": "Alice"})
		if err != nil {
			t.Fatalf("render error: %v", err)
		}
		output := buf.String()
		if !strings.Contains(output, "Welcome, Alice") {
			t.Errorf("expected 'Welcome, Alice' in output, got: %s", output)
		}
	})

	t.Run("page with layout override", func(t *testing.T) {
		var buf bytes.Buffer
		err := e.Render(&buf, "simple", nil, Layout("bare"))
		if err != nil {
			t.Fatalf("render error: %v", err)
		}
		output := buf.String()
		if strings.Contains(output, "<!DOCTYPE html>") {
			t.Error("expected bare layout (no DOCTYPE)")
		}
	})

	t.Run("page not found", func(t *testing.T) {
		var buf bytes.Buffer
		err := e.Render(&buf, "nonexistent", nil)
		if err == nil {
			t.Fatal("expected error for nonexistent page")
		}
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}
	})
}

func TestEngine_RenderWithContent(t *testing.T) {
	e := New(Config{
		Dir:         "testdata/views",
		Development: true,
	})

	t.Run("page renders into layout content", func(t *testing.T) {
		var buf bytes.Buffer
		err := e.Render(&buf, "home", Data{"Name": "Bob"})
		if err != nil {
			t.Fatalf("render error: %v", err)
		}
		output := buf.String()

		// Check that content is rendered in layout
		if !strings.Contains(output, "Welcome, Bob") {
			t.Errorf("expected page content in layout, got: %s", output)
		}
		if !strings.Contains(output, "Default Header") {
			t.Errorf("expected default header, got: %s", output)
		}
		if !strings.Contains(output, "Default Footer") {
			t.Errorf("expected default footer, got: %s", output)
		}
	})
}

func TestEngine_ComponentInPage(t *testing.T) {
	e := New(Config{
		Dir:         "testdata/views",
		Development: true,
	})

	var buf bytes.Buffer
	err := e.Render(&buf, "with-component", nil)
	if err != nil {
		t.Fatalf("render error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, `class="btn btn-primary"`) {
		t.Errorf("expected btn-primary class, got: %s", output)
	}
	if !strings.Contains(output, "Click Me") {
		t.Errorf("expected 'Click Me' label, got: %s", output)
	}
}

func TestEngine_PartialInPage(t *testing.T) {
	e := New(Config{
		Dir:         "testdata/views",
		Development: true,
	})

	var buf bytes.Buffer
	err := e.Render(&buf, "with-partial", nil)
	if err != nil {
		t.Fatalf("render error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, `class="sidebar"`) {
		t.Errorf("expected sidebar class, got: %s", output)
	}
	if !strings.Contains(output, "Sidebar Navigation") {
		t.Errorf("expected 'Sidebar Navigation', got: %s", output)
	}
}

func TestEngine_Load(t *testing.T) {
	e := New(Config{
		Dir:         "testdata/views",
		Development: false,
	})

	err := e.Load()
	if err != nil {
		t.Fatalf("preload error: %v", err)
	}

	e.mu.RLock()
	defer e.mu.RUnlock()
	if len(e.cache) == 0 {
		t.Error("expected templates to be cached")
	}
}

func TestEngine_Clear(t *testing.T) {
	e := New(Config{
		Dir:         "testdata/views",
		Development: false,
	})

	err := e.Load()
	if err != nil {
		t.Fatalf("preload error: %v", err)
	}

	e.Clear()

	e.mu.RLock()
	defer e.mu.RUnlock()
	if len(e.cache) != 0 {
		t.Error("expected cache to be empty")
	}
}

func TestEngine_EmbedFS(t *testing.T) {
	fsys := os.DirFS("testdata/views")
	e := New(Config{FS: fsys})

	var buf bytes.Buffer
	err := e.Render(&buf, "simple", nil, NoLayout())
	if err != nil {
		t.Fatalf("render error: %v", err)
	}
	if !strings.Contains(buf.String(), "Simple page content") {
		t.Errorf("expected content, got: %s", buf.String())
	}
}

func TestMiddleware(t *testing.T) {
	e := New(Config{
		Dir:         "testdata/views",
		Development: true,
	})

	app := mizu.New()
	app.Use(e.Middleware())

	app.Get("/", func(c *mizu.Ctx) error {
		return Render(c, "home", Data{"Name": "Test"})
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Welcome, Test") {
		t.Errorf("expected 'Welcome, Test', got: %s", rec.Body.String())
	}
}

func TestFrom(t *testing.T) {
	e := New(Config{
		Dir:         "testdata/views",
		Development: true,
	})

	app := mizu.New()
	app.Use(e.Middleware())

	var gotEngine *Engine
	app.Get("/", func(c *mizu.Ctx) error {
		gotEngine = From(c)
		return c.Text(200, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if gotEngine != e {
		t.Error("expected to retrieve engine from context")
	}
}

func TestErrors(t *testing.T) {
	t.Run("Error Is", func(t *testing.T) {
		err := &Error{Kind: "page", Name: "test"}
		if !errors.Is(err, ErrNotFound) {
			t.Error("expected page Error to match ErrNotFound")
		}

		err = &Error{Kind: "layout", Name: "test"}
		if !errors.Is(err, ErrNotFound) {
			t.Error("expected layout Error to match ErrNotFound")
		}
	})

	t.Run("Error with wrapped error", func(t *testing.T) {
		inner := errors.New("parse error")
		err := &Error{Kind: "page", Name: "test", Err: inner}

		if !strings.Contains(err.Error(), "test") {
			t.Error("expected template name in error")
		}
		if !errors.Is(err, inner) {
			t.Error("expected Unwrap to return inner error")
		}
	})
}

// Template function tests

func TestDictFunc(t *testing.T) {
	t.Run("valid pairs", func(t *testing.T) {
		m, err := dictFunc("key1", "value1", "key2", 42)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m["key1"] != "value1" {
			t.Errorf("expected key1='value1', got %v", m["key1"])
		}
		if m["key2"] != 42 {
			t.Errorf("expected key2=42, got %v", m["key2"])
		}
	})

	t.Run("odd number of args", func(t *testing.T) {
		_, err := dictFunc("key1", "value1", "key2")
		if err == nil {
			t.Error("expected error for odd number of arguments")
		}
	})

	t.Run("non-string key", func(t *testing.T) {
		_, err := dictFunc(123, "value1")
		if err == nil {
			t.Error("expected error for non-string key")
		}
	})
}

func TestListFunc(t *testing.T) {
	list := listFunc("a", "b", "c")
	if len(list) != 3 {
		t.Errorf("expected 3 items, got %d", len(list))
	}
	if list[0] != "a" || list[1] != "b" || list[2] != "c" {
		t.Errorf("unexpected list contents: %v", list)
	}
}
