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
	t.Run("default options", func(t *testing.T) {
		e := New(Options{})
		if e.opts.Dir != "views" {
			t.Errorf("expected Dir='views', got %q", e.opts.Dir)
		}
		if e.opts.Extension != ".html" {
			t.Errorf("expected Extension='.html', got %q", e.opts.Extension)
		}
		if e.opts.DefaultLayout != "default" {
			t.Errorf("expected DefaultLayout='default', got %q", e.opts.DefaultLayout)
		}
	})

	t.Run("custom options", func(t *testing.T) {
		e := New(Options{
			Dir:           "templates",
			Extension:     ".tmpl",
			DefaultLayout: "main",
			Development:   true,
		})
		if e.opts.Dir != "templates" {
			t.Errorf("expected Dir='templates', got %q", e.opts.Dir)
		}
		if e.opts.Extension != ".tmpl" {
			t.Errorf("expected Extension='.tmpl', got %q", e.opts.Extension)
		}
		if e.opts.DefaultLayout != "main" {
			t.Errorf("expected DefaultLayout='main', got %q", e.opts.DefaultLayout)
		}
		if !e.opts.Development {
			t.Error("expected Development=true")
		}
	})

	t.Run("with custom funcs", func(t *testing.T) {
		e := New(Options{
			Funcs: template.FuncMap{
				"customFunc": func() string {
					return "custom"
				},
			},
		})
		if _, ok := e.baseFuncs["customFunc"]; !ok {
			t.Error("custom function not added")
		}
	})
}

func TestEngine_Render(t *testing.T) {
	e := New(Options{
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
		if !strings.Contains(output, "Home Page") {
			t.Errorf("expected 'Home Page' title in output, got: %s", output)
		}
	})

	t.Run("page with custom layout", func(t *testing.T) {
		var buf bytes.Buffer
		err := e.Render(&buf, "with-layout", nil)
		if err != nil {
			t.Fatalf("render error: %v", err)
		}
		output := buf.String()
		// bare layout just outputs content
		if strings.Contains(output, "<!DOCTYPE html>") {
			t.Error("expected bare layout (no DOCTYPE)")
		}
		if !strings.Contains(output, "Using bare layout") {
			t.Errorf("expected 'Using bare layout' in output, got: %s", output)
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
		if !errors.Is(err, ErrTemplateNotFound) {
			t.Errorf("expected ErrTemplateNotFound, got: %v", err)
		}
	})
}

func TestEngine_RenderComponent(t *testing.T) {
	e := New(Options{
		Dir:         "testdata/views",
		Development: true,
	})

	t.Run("simple component", func(t *testing.T) {
		var buf bytes.Buffer
		err := e.RenderComponent(&buf, "button", Data{
			"Label":   "Click Me",
			"Variant": "primary",
		})
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
	})

	t.Run("component not found", func(t *testing.T) {
		var buf bytes.Buffer
		err := e.RenderComponent(&buf, "nonexistent", nil)
		if err == nil {
			t.Fatal("expected error for nonexistent component")
		}
		if !errors.Is(err, ErrComponentNotFound) {
			t.Errorf("expected ErrComponentNotFound, got: %v", err)
		}
	})
}

func TestEngine_ComponentInPage(t *testing.T) {
	e := New(Options{
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
	e := New(Options{
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

func TestEngine_Slots(t *testing.T) {
	e := New(Options{
		Dir:         "testdata/views",
		Development: true,
	})

	var buf bytes.Buffer
	err := e.Render(&buf, "home", Data{"Name": "Bob"})
	if err != nil {
		t.Fatalf("render error: %v", err)
	}
	output := buf.String()

	// Title slot should be filled
	if !strings.Contains(output, "<title>Home Page</title>") {
		t.Errorf("expected title slot filled, got: %s", output)
	}

	// Header slot should have default
	if !strings.Contains(output, "Default Header") {
		t.Errorf("expected default header, got: %s", output)
	}

	// Footer slot should have default
	if !strings.Contains(output, "Default Footer") {
		t.Errorf("expected default footer, got: %s", output)
	}
}

func TestEngine_Preload(t *testing.T) {
	e := New(Options{
		Dir:         "testdata/views",
		Development: false,
	})

	err := e.Preload()
	if err != nil {
		t.Fatalf("preload error: %v", err)
	}

	// Verify templates are cached
	e.mu.RLock()
	defer e.mu.RUnlock()

	if len(e.contentCache) == 0 {
		t.Error("expected templates to be cached")
	}
}

func TestEngine_ClearCache(t *testing.T) {
	e := New(Options{
		Dir:         "testdata/views",
		Development: false,
	})

	// Preload
	err := e.Preload()
	if err != nil {
		t.Fatalf("preload error: %v", err)
	}

	// Clear
	e.ClearCache()

	e.mu.RLock()
	defer e.mu.RUnlock()

	if len(e.contentCache) != 0 {
		t.Error("expected content cache to be empty")
	}
}

func TestEngine_EmbedFS(t *testing.T) {
	// Test with os.DirFS as a stand-in for embed.FS
	fsys := os.DirFS("testdata/views")
	e := New(Options{
		FS: fsys,
	})

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
	e := New(Options{
		Dir:         "testdata/views",
		Development: true,
	})

	app := mizu.New()
	app.Use(Middleware(e))

	app.Get("/", func(c *mizu.Ctx) error {
		return Render(c, "home", Data{"Name": "Test"})
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Welcome, Test") {
		t.Errorf("expected 'Welcome, Test', got: %s", body)
	}
}

func TestRender_Status(t *testing.T) {
	e := New(Options{
		Dir:         "testdata/views",
		Development: true,
	})

	app := mizu.New()
	app.Use(Middleware(e))

	app.Get("/not-found", func(c *mizu.Ctx) error {
		return Render(c, "simple", nil, Status(404), NoLayout())
	})

	req := httptest.NewRequest(http.MethodGet, "/not-found", nil)
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func TestRenderComponent_Handler(t *testing.T) {
	e := New(Options{
		Dir:         "testdata/views",
		Development: true,
	})

	app := mizu.New()
	app.Use(Middleware(e))

	app.Get("/button", func(c *mizu.Ctx) error {
		return RenderComponent(c, "button", Data{
			"Label":   "Submit",
			"Variant": "success",
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/button", nil)
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "btn-success") {
		t.Errorf("expected btn-success, got: %s", body)
	}
}

func TestGetEngine(t *testing.T) {
	e := New(Options{
		Dir:         "testdata/views",
		Development: true,
	})

	app := mizu.New()
	app.Use(Middleware(e))

	var gotEngine *Engine
	app.Get("/", func(c *mizu.Ctx) error {
		gotEngine = GetEngine(c)
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
	t.Run("NotFoundError Is", func(t *testing.T) {
		err := &NotFoundError{Type: "page", Name: "test"}
		if !errors.Is(err, ErrTemplateNotFound) {
			t.Error("expected page NotFoundError to match ErrTemplateNotFound")
		}

		err = &NotFoundError{Type: "layout", Name: "test"}
		if !errors.Is(err, ErrLayoutNotFound) {
			t.Error("expected layout NotFoundError to match ErrLayoutNotFound")
		}

		err = &NotFoundError{Type: "component", Name: "test"}
		if !errors.Is(err, ErrComponentNotFound) {
			t.Error("expected component NotFoundError to match ErrComponentNotFound")
		}

		err = &NotFoundError{Type: "partial", Name: "test"}
		if !errors.Is(err, ErrPartialNotFound) {
			t.Error("expected partial NotFoundError to match ErrPartialNotFound")
		}
	})

	t.Run("TemplateError", func(t *testing.T) {
		inner := errors.New("parse error")
		err := &TemplateError{Name: "test", Line: 10, Err: inner}

		if !strings.Contains(err.Error(), "test") {
			t.Error("expected template name in error")
		}
		if !strings.Contains(err.Error(), "10") {
			t.Error("expected line number in error")
		}
		if !errors.Is(err, inner) {
			t.Error("expected Unwrap to return inner error")
		}
	})
}
