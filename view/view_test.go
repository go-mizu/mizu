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
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}
	})
}

func TestEngine_Component(t *testing.T) {
	e := New(Config{
		Dir:         "testdata/views",
		Development: true,
	})

	t.Run("simple component", func(t *testing.T) {
		var buf bytes.Buffer
		err := e.Component(&buf, "button", Data{
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
		err := e.Component(&buf, "nonexistent", nil)
		if err == nil {
			t.Fatal("expected error for nonexistent component")
		}
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got: %v", err)
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

func TestEngine_Slots(t *testing.T) {
	e := New(Config{
		Dir:         "testdata/views",
		Development: true,
	})

	var buf bytes.Buffer
	err := e.Render(&buf, "home", Data{"Name": "Bob"})
	if err != nil {
		t.Fatalf("render error: %v", err)
	}
	output := buf.String()

	if !strings.Contains(output, "<title>Home Page</title>") {
		t.Errorf("expected title slot filled, got: %s", output)
	}
	if !strings.Contains(output, "Default Header") {
		t.Errorf("expected default header, got: %s", output)
	}
	if !strings.Contains(output, "Default Footer") {
		t.Errorf("expected default footer, got: %s", output)
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

func TestHandler(t *testing.T) {
	e := New(Config{
		Dir:         "testdata/views",
		Development: true,
	})

	app := mizu.New()
	app.Use(e.Handler())

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

func TestRender_Status(t *testing.T) {
	e := New(Config{
		Dir:         "testdata/views",
		Development: true,
	})

	app := mizu.New()
	app.Use(e.Handler())

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

func TestComponent_Handler(t *testing.T) {
	e := New(Config{
		Dir:         "testdata/views",
		Development: true,
	})

	app := mizu.New()
	app.Use(e.Handler())

	app.Get("/button", func(c *mizu.Ctx) error {
		return Component(c, "button", Data{
			"Label":   "Submit",
			"Variant": "success",
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/button", nil)
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if !strings.Contains(rec.Body.String(), "btn-success") {
		t.Errorf("expected btn-success, got: %s", rec.Body.String())
	}
}

func TestFrom(t *testing.T) {
	e := New(Config{
		Dir:         "testdata/views",
		Development: true,
	})

	app := mizu.New()
	app.Use(e.Handler())

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

		err = &Error{Kind: "component", Name: "test"}
		if !errors.Is(err, ErrNotFound) {
			t.Error("expected component Error to match ErrNotFound")
		}

		err = &Error{Kind: "partial", Name: "test"}
		if !errors.Is(err, ErrNotFound) {
			t.Error("expected partial Error to match ErrNotFound")
		}
	})

	t.Run("Error with wrapped error", func(t *testing.T) {
		inner := errors.New("parse error")
		err := &Error{Kind: "page", Name: "test", Line: 10, Err: inner}

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

func TestDefaultFunc(t *testing.T) {
	t.Run("returns default for nil", func(t *testing.T) {
		result := defaultFunc("default", nil)
		if result != "default" {
			t.Errorf("expected 'default', got %v", result)
		}
	})

	t.Run("returns default for empty string", func(t *testing.T) {
		result := defaultFunc("default", "")
		if result != "default" {
			t.Errorf("expected 'default', got %v", result)
		}
	})

	t.Run("returns value when not empty", func(t *testing.T) {
		result := defaultFunc("default", "actual")
		if result != "actual" {
			t.Errorf("expected 'actual', got %v", result)
		}
	})

	t.Run("returns default for zero int", func(t *testing.T) {
		result := defaultFunc(42, 0)
		if result != 42 {
			t.Errorf("expected 42, got %v", result)
		}
	})
}

func TestEmptyFunc(t *testing.T) {
	tests := []struct {
		name     string
		val      any
		expected bool
	}{
		{"nil", nil, true},
		{"empty string", "", true},
		{"non-empty string", "hello", false},
		{"zero int", 0, true},
		{"non-zero int", 42, false},
		{"false bool", false, true},
		{"true bool", true, false},
		{"empty slice", []string{}, true},
		{"non-empty slice", []string{"a"}, false},
		{"empty map", map[string]int{}, true},
		{"non-empty map", map[string]int{"a": 1}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := emptyFunc(tt.val)
			if result != tt.expected {
				t.Errorf("empty(%v) = %v, want %v", tt.val, result, tt.expected)
			}
		})
	}
}

func TestTernaryFunc(t *testing.T) {
	t.Run("true condition", func(t *testing.T) {
		result := ternaryFunc(true, "yes", "no")
		if result != "yes" {
			t.Errorf("expected 'yes', got %v", result)
		}
	})

	t.Run("false condition", func(t *testing.T) {
		result := ternaryFunc(false, "yes", "no")
		if result != "no" {
			t.Errorf("expected 'no', got %v", result)
		}
	})
}

func TestCoalesceFunc(t *testing.T) {
	t.Run("returns first non-empty", func(t *testing.T) {
		result := coalesceFunc("", nil, "value", "other")
		if result != "value" {
			t.Errorf("expected 'value', got %v", result)
		}
	})

	t.Run("returns nil if all empty", func(t *testing.T) {
		result := coalesceFunc("", nil, 0)
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})
}

func TestComparisonFuncs(t *testing.T) {
	funcs := baseFuncs()
	eq := funcs["eq"].(func(any, any) bool)
	ne := funcs["ne"].(func(any, any) bool)
	lt := funcs["lt"].(func(any, any) bool)
	le := funcs["le"].(func(any, any) bool)
	gt := funcs["gt"].(func(any, any) bool)
	ge := funcs["ge"].(func(any, any) bool)

	t.Run("eq", func(t *testing.T) {
		if !eq(5, 5) {
			t.Error("expected 5 == 5")
		}
		if eq(5, 6) {
			t.Error("expected 5 != 6")
		}
	})

	t.Run("ne", func(t *testing.T) {
		if !ne(5, 6) {
			t.Error("expected 5 != 6")
		}
		if ne(5, 5) {
			t.Error("expected 5 == 5")
		}
	})

	t.Run("lt", func(t *testing.T) {
		if !lt(5, 6) {
			t.Error("expected 5 < 6")
		}
		if lt(6, 5) {
			t.Error("expected not 6 < 5")
		}
	})

	t.Run("le", func(t *testing.T) {
		if !le(5, 6) {
			t.Error("expected 5 <= 6")
		}
		if !le(5, 5) {
			t.Error("expected 5 <= 5")
		}
	})

	t.Run("gt", func(t *testing.T) {
		if !gt(6, 5) {
			t.Error("expected 6 > 5")
		}
		if gt(5, 6) {
			t.Error("expected not 5 > 6")
		}
	})

	t.Run("ge", func(t *testing.T) {
		if !ge(6, 5) {
			t.Error("expected 6 >= 5")
		}
		if !ge(5, 5) {
			t.Error("expected 5 >= 5")
		}
	})
}

func TestMathFuncs(t *testing.T) {
	funcs := baseFuncs()
	add := funcs["add"].(func(any, any) any)
	sub := funcs["sub"].(func(any, any) any)
	mul := funcs["mul"].(func(any, any) any)
	div := funcs["div"].(func(any, any) any)
	mod := funcs["mod"].(func(any, any) any)

	t.Run("add", func(t *testing.T) {
		result := add(5, 3)
		if result != 8.0 {
			t.Errorf("expected 8, got %v", result)
		}
	})

	t.Run("sub", func(t *testing.T) {
		result := sub(5, 3)
		if result != 2.0 {
			t.Errorf("expected 2, got %v", result)
		}
	})

	t.Run("mul", func(t *testing.T) {
		result := mul(5, 3)
		if result != 15.0 {
			t.Errorf("expected 15, got %v", result)
		}
	})

	t.Run("div", func(t *testing.T) {
		result := div(6, 2)
		if result != 3.0 {
			t.Errorf("expected 3, got %v", result)
		}
	})

	t.Run("div by zero", func(t *testing.T) {
		result := div(6, 0)
		if result != 0.0 {
			t.Errorf("expected 0, got %v", result)
		}
	})

	t.Run("mod", func(t *testing.T) {
		result := mod(7, 3)
		if result != int64(1) {
			t.Errorf("expected 1, got %v", result)
		}
	})

	t.Run("mod by zero", func(t *testing.T) {
		result := mod(7, 0)
		if result != int64(0) {
			t.Errorf("expected 0, got %v", result)
		}
	})
}
