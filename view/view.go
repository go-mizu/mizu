package view

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"github.com/go-mizu/mizu"
)

// ErrNotFound is returned when a template is not found.
var ErrNotFound = errors.New("view: not found")

// Error wraps a template error with context.
type Error struct {
	Kind string // "page", "layout", "component", "partial"
	Name string
	Line int
	Err  error
}

func (e *Error) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("view: %s %q at line %d: %v", e.Kind, e.Name, e.Line, e.Err)
	}
	if e.Err != nil {
		return fmt.Sprintf("view: %s %q: %v", e.Kind, e.Name, e.Err)
	}
	return fmt.Sprintf("view: %s %q not found", e.Kind, e.Name)
}

func (e *Error) Unwrap() error { return e.Err }

func (e *Error) Is(target error) bool {
	return target == ErrNotFound && e.Err == nil
}

// Config configures the view engine.
type Config struct {
	Dir           string           // Views directory path. Default: "views"
	FS            fs.FS            // Optional filesystem (for embed.FS in production)
	Extension     string           // Template file extension. Default: ".html"
	DefaultLayout string           // Default layout name. Default: "default"
	Funcs         template.FuncMap // Custom template functions
	Delims        [2]string        // Custom delimiters. Default: "{{", "}}"
	Development   bool             // Enable dev mode (reload, detailed errors)
	StrictMode    bool             // Fail on missing slots/components
	DedupeStacks  bool             // Remove duplicate stack entries. Default: true
}

func (c *Config) defaults() {
	if c.Dir == "" {
		c.Dir = "views"
	}
	if c.Extension == "" {
		c.Extension = ".html"
	}
	if c.DefaultLayout == "" {
		c.DefaultLayout = "default"
	}
	if c.Delims[0] == "" {
		c.Delims[0] = "{{"
	}
	if c.Delims[1] == "" {
		c.Delims[1] = "}}"
	}
}

// Data is a convenience type for template data.
type Data = map[string]any

// Engine is the view template engine.
type Engine struct {
	cfg   Config
	fs    fs.FS
	mu    sync.RWMutex
	cache map[string]string
	funcs template.FuncMap
}

// New creates a new view engine.
func New(cfg Config) *Engine {
	cfg.defaults()
	e := &Engine{
		cfg:   cfg,
		cache: make(map[string]string),
		funcs: baseFuncs(),
	}
	for k, v := range cfg.Funcs {
		e.funcs[k] = v
	}
	if cfg.FS != nil {
		e.fs = cfg.FS
	} else {
		e.fs = os.DirFS(cfg.Dir)
	}
	return e
}

// Load preloads all templates. Call at startup in production.
func (e *Engine) Load() error {
	for _, dir := range []string{"layouts", "pages", "components", "partials"} {
		if err := e.loadDir(dir); err != nil {
			return fmt.Errorf("load %s: %w", dir, err)
		}
	}
	return nil
}

// Clear clears the template cache.
func (e *Engine) Clear() {
	e.mu.Lock()
	e.cache = make(map[string]string)
	e.mu.Unlock()
}

// Render renders a page template.
func (e *Engine) Render(w io.Writer, page string, data any, opts ...option) error {
	cfg := &renderCfg{layout: e.cfg.DefaultLayout}
	for _, opt := range opts {
		opt(cfg)
	}

	ctx := newRenderCtx(e)
	pd := &pageData{
		Page: pageMeta{Name: page, Layout: cfg.layout},
		Data: data,
	}

	content, err := e.content("pages", page)
	if err != nil {
		return err
	}

	tmpl, err := e.parseWithCtx(page, content, ctx, pd)
	if err != nil {
		return err
	}

	// Extract layout from template if defined
	if def := tmpl.Lookup("layout"); def != nil {
		var buf bytes.Buffer
		if err := def.Execute(&buf, nil); err == nil {
			if name := strings.TrimSpace(buf.String()); name != "" {
				cfg.layout = name
				pd.Page.Layout = name
			}
		}
	}

	// Render page content
	var pageBuf bytes.Buffer
	if def := tmpl.Lookup("content"); def != nil {
		if err := def.Execute(&pageBuf, pd); err != nil {
			return &Error{Kind: "page", Name: page, Err: err}
		}
	} else {
		if err := tmpl.Execute(&pageBuf, pd); err != nil {
			return &Error{Kind: "page", Name: page, Err: err}
		}
	}

	ctx.setSlot("content", pageBuf.String())

	// Extract slot definitions
	for _, t := range tmpl.Templates() {
		name := t.Name()
		if name != "" && name != page && name != "layout" && name != "content" {
			var buf bytes.Buffer
			if err := t.Execute(&buf, pd); err != nil {
				return &Error{Kind: "page", Name: page, Err: fmt.Errorf("slot %q: %w", name, err)}
			}
			ctx.setSlot(name, buf.String())
		}
	}

	if cfg.noLayout {
		_, err := w.Write(pageBuf.Bytes())
		return err
	}

	layoutContent, err := e.content("layouts", cfg.layout)
	if err != nil {
		return err
	}

	layoutTmpl, err := e.parseWithCtx(cfg.layout, layoutContent, ctx, pd)
	if err != nil {
		return err
	}

	if err := layoutTmpl.Execute(w, pd); err != nil {
		return &Error{Kind: "layout", Name: cfg.layout, Err: err}
	}
	return nil
}

// Component renders a component template.
func (e *Engine) Component(w io.Writer, name string, data any) error {
	ctx := newRenderCtx(e)
	return e.component(w, name, data, ctx)
}

// Partial renders a partial template.
func (e *Engine) Partial(w io.Writer, name string, data any) error {
	ctx := newRenderCtx(e)
	return e.partial(w, name, data, ctx)
}

func (e *Engine) component(w io.Writer, name string, data any, ctx *renderCtx) error {
	content, err := e.content("components", name)
	if err != nil {
		return err
	}
	tmpl, err := e.parseWithCtx(name, content, ctx, data)
	if err != nil {
		return err
	}
	if err := tmpl.Execute(w, data); err != nil {
		return &Error{Kind: "component", Name: name, Err: err}
	}
	return nil
}

func (e *Engine) partial(w io.Writer, name string, data any, ctx *renderCtx) error {
	content, err := e.content("partials", name)
	if err != nil {
		return err
	}
	tmpl, err := e.parseWithCtx(name, content, ctx, data)
	if err != nil {
		return err
	}
	if err := tmpl.Execute(w, data); err != nil {
		return &Error{Kind: "partial", Name: name, Err: err}
	}
	return nil
}

func (e *Engine) content(kind, name string) (string, error) {
	key := kind + "/" + name
	if e.cfg.Development {
		return e.load(kind, name)
	}
	e.mu.RLock()
	content, ok := e.cache[key]
	e.mu.RUnlock()
	if ok {
		return content, nil
	}
	content, err := e.load(kind, name)
	if err != nil {
		return "", err
	}
	e.mu.Lock()
	e.cache[key] = content
	e.mu.Unlock()
	return content, nil
}

func (e *Engine) load(kind, name string) (string, error) {
	p := path.Join(kind, name+e.cfg.Extension)
	data, err := fs.ReadFile(e.fs, p)
	if err != nil {
		if os.IsNotExist(err) {
			k := "page"
			switch kind {
			case "layouts":
				k = "layout"
			case "components":
				k = "component"
			case "partials":
				k = "partial"
			}
			return "", &Error{Kind: k, Name: name}
		}
		return "", fmt.Errorf("read template %s: %w", p, err)
	}
	return string(data), nil
}

func (e *Engine) loadDir(dir string) error {
	return e.walkDir(dir, func(name string) error {
		key := dir + "/" + name
		content, err := e.load(dir, name)
		if err != nil {
			return err
		}
		if _, err := e.parse(name, content); err != nil {
			return err
		}
		e.mu.Lock()
		e.cache[key] = content
		e.mu.Unlock()
		return nil
	})
}

func (e *Engine) walkDir(dir string, fn func(string) error) error {
	return fs.WalkDir(e.fs, dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return fs.SkipDir
			}
			return err
		}
		if d.IsDir() || !strings.HasSuffix(p, e.cfg.Extension) {
			return nil
		}
		name := strings.TrimPrefix(p, dir+"/")
		name = strings.TrimSuffix(name, e.cfg.Extension)
		name = filepath.ToSlash(name)
		return fn(name)
	})
}

func (e *Engine) parse(name, content string) (*template.Template, error) {
	tmpl := template.New(name).Delims(e.cfg.Delims[0], e.cfg.Delims[1]).Funcs(e.funcs)
	tmpl, err := tmpl.Parse(content)
	if err != nil {
		return nil, &Error{Kind: "template", Name: name, Err: err}
	}
	return tmpl, nil
}

func (e *Engine) parseWithCtx(name, content string, ctx *renderCtx, data any) (*template.Template, error) {
	funcs := make(template.FuncMap)
	for k, v := range e.funcs {
		funcs[k] = v
	}
	funcs["slot"] = func(n string, defs ...any) template.HTML { return ctx.slot(n, defs...) }
	funcs["stack"] = func(n string) template.HTML { return ctx.stack(n, e.cfg.DedupeStacks) }
	funcs["push"] = func(string) string { return "" }
	funcs["component"] = func(n string, d ...any) template.HTML {
		var v any
		if len(d) > 0 {
			v = d[0]
		}
		var buf bytes.Buffer
		if err := e.component(&buf, n, v, ctx); err != nil {
			if e.cfg.Development {
				return template.HTML(fmt.Sprintf("<!-- component error: %v -->", err))
			}
			return ""
		}
		return template.HTML(buf.String())
	}
	funcs["partial"] = func(n string, d ...any) template.HTML {
		v := data
		if len(d) > 0 {
			v = d[0]
		}
		var buf bytes.Buffer
		if err := e.partial(&buf, n, v, ctx); err != nil {
			if e.cfg.Development {
				return template.HTML(fmt.Sprintf("<!-- partial error: %v -->", err))
			}
			return ""
		}
		return template.HTML(buf.String())
	}
	funcs["children"] = func() template.HTML { return ctx.children() }

	tmpl := template.New(name).Delims(e.cfg.Delims[0], e.cfg.Delims[1]).Funcs(funcs)
	tmpl, err := tmpl.Parse(content)
	if err != nil {
		return nil, &Error{Kind: "template", Name: name, Err: err}
	}
	return tmpl, nil
}

// renderCtx holds state for a single render operation.
type renderCtx struct {
	engine   *Engine
	mu       sync.Mutex
	slots    map[string]string
	stacks   map[string][]string
	childBuf string
}

func newRenderCtx(e *Engine) *renderCtx {
	return &renderCtx{
		engine: e,
		slots:  make(map[string]string),
		stacks: make(map[string][]string),
	}
}

func (c *renderCtx) setSlot(name, content string) {
	c.mu.Lock()
	c.slots[name] = content
	c.mu.Unlock()
}

func (c *renderCtx) slot(name string, defs ...any) template.HTML {
	c.mu.Lock()
	defer c.mu.Unlock()
	if v, ok := c.slots[name]; ok {
		return template.HTML(v)
	}
	if len(defs) > 0 {
		switch d := defs[0].(type) {
		case string:
			return template.HTML(d)
		case template.HTML:
			return d
		}
	}
	return ""
}

func (c *renderCtx) push(name, content string) {
	c.mu.Lock()
	c.stacks[name] = append(c.stacks[name], content)
	c.mu.Unlock()
}

func (c *renderCtx) stack(name string, dedupe bool) template.HTML {
	c.mu.Lock()
	defer c.mu.Unlock()
	items := c.stacks[name]
	if len(items) == 0 {
		return ""
	}
	if !dedupe {
		return template.HTML(strings.Join(items, ""))
	}
	seen := make(map[string]bool)
	var result strings.Builder
	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result.WriteString(item)
		}
	}
	return template.HTML(result.String())
}

func (c *renderCtx) setChildren(content string) {
	c.mu.Lock()
	c.childBuf = content
	c.mu.Unlock()
}

func (c *renderCtx) children() template.HTML {
	c.mu.Lock()
	defer c.mu.Unlock()
	return template.HTML(c.childBuf)
}

// pageData is the data passed to page templates.
type pageData struct {
	Page pageMeta
	Data any
	CSRF string
}

type pageMeta struct {
	Name   string
	Title  string
	Layout string
}

// option configures a render call.
type option func(*renderCfg)

type renderCfg struct {
	status   int
	layout   string
	noLayout bool
}

// Status sets the HTTP status code.
func Status(code int) option {
	return func(c *renderCfg) { c.status = code }
}

// Layout sets the layout name.
func Layout(name string) option {
	return func(c *renderCfg) { c.layout = name }
}

// NoLayout disables layout rendering.
func NoLayout() option {
	return func(c *renderCfg) { c.noLayout = true }
}

// Handler returns a Mizu middleware that adds the engine to context.
func (e *Engine) Handler() mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			ctx := context.WithValue(c.Context(), engineKey{}, e)
			*c.Request() = *c.Request().WithContext(ctx)
			return next(c)
		}
	}
}

type engineKey struct{}

// From returns the engine from context.
func From(c *mizu.Ctx) *Engine {
	if e, ok := c.Context().Value(engineKey{}).(*Engine); ok {
		return e
	}
	return nil
}

// Render renders a page using the engine from context.
func Render(c *mizu.Ctx, page string, data any, opts ...option) error {
	e := From(c)
	if e == nil {
		return ErrNotFound
	}
	cfg := &renderCfg{status: 200, layout: e.cfg.DefaultLayout}
	for _, opt := range opts {
		opt(cfg)
	}
	c.Writer().Header().Set("Content-Type", "text/html; charset=utf-8")
	c.Writer().WriteHeader(cfg.status)
	return e.Render(c.Writer(), page, data, opts...)
}

// Component renders a component using the engine from context.
func Component(c *mizu.Ctx, page string, data any) error {
	e := From(c)
	if e == nil {
		return ErrNotFound
	}
	c.Writer().Header().Set("Content-Type", "text/html; charset=utf-8")
	return e.Component(c.Writer(), page, data)
}

// baseFuncs returns the base template functions.
func baseFuncs() template.FuncMap {
	return template.FuncMap{
		// Placeholders (replaced at render time)
		"slot":      func(string, ...any) template.HTML { return "" },
		"stack":     func(string) template.HTML { return "" },
		"push":      func(string) string { return "" },
		"component": func(string, ...any) template.HTML { return "" },
		"partial":   func(string, ...any) template.HTML { return "" },
		"children":  func() template.HTML { return "" },

		// Data helpers
		"dict":    dictFunc,
		"list":    listFunc,
		"default": defaultFunc,
		"empty":   emptyFunc,

		// Safe content
		"safeHTML": func(s string) template.HTML { return template.HTML(s) },       //nolint:gosec
		"safeCSS":  func(s string) template.CSS { return template.CSS(s) },         //nolint:gosec
		"safeJS":   func(s string) template.JS { return template.JS(s) },           //nolint:gosec
		"safeURL":  func(s string) template.URL { return template.URL(s) },         //nolint:gosec

		// String helpers
		"upper":     strings.ToUpper,
		"lower":     strings.ToLower,
		"title":     strings.Title, //nolint:staticcheck
		"trim":      strings.TrimSpace,
		"contains":  strings.Contains,
		"replace":   strings.ReplaceAll,
		"split":     strings.Split,
		"join":      strings.Join,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,

		// Conditionals
		"ternary":  ternaryFunc,
		"coalesce": coalesceFunc,

		// Comparisons
		"eq": func(a, b any) bool { return reflect.DeepEqual(a, b) },
		"ne": func(a, b any) bool { return !reflect.DeepEqual(a, b) },
		"lt": func(a, b any) bool { return toFloat64(a) < toFloat64(b) },
		"le": func(a, b any) bool { return toFloat64(a) <= toFloat64(b) },
		"gt": func(a, b any) bool { return toFloat64(a) > toFloat64(b) },
		"ge": func(a, b any) bool { return toFloat64(a) >= toFloat64(b) },

		// Math
		"add": func(a, b any) any { return toFloat64(a) + toFloat64(b) },
		"sub": func(a, b any) any { return toFloat64(a) - toFloat64(b) },
		"mul": func(a, b any) any { return toFloat64(a) * toFloat64(b) },
		"div": func(a, b any) any {
			if v := toFloat64(b); v != 0 {
				return toFloat64(a) / v
			}
			return 0.0
		},
		"mod": func(a, b any) any {
			if v := toInt64(b); v != 0 {
				return toInt64(a) % v
			}
			return int64(0)
		},
	}
}

func dictFunc(pairs ...any) (map[string]any, error) {
	if len(pairs)%2 != 0 {
		return nil, fmt.Errorf("dict requires even number of arguments")
	}
	m := make(map[string]any, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		k, ok := pairs[i].(string)
		if !ok {
			return nil, fmt.Errorf("dict keys must be strings, got %T", pairs[i])
		}
		m[k] = pairs[i+1]
	}
	return m, nil
}

func listFunc(items ...any) []any { return items }

func defaultFunc(def, val any) any {
	if emptyFunc(val) {
		return def
	}
	return val
}

func emptyFunc(val any) bool {
	if val == nil {
		return true
	}
	v := reflect.ValueOf(val)
	switch v.Kind() {
	case reflect.String, reflect.Array, reflect.Slice, reflect.Map:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	}
	return false
}

func ternaryFunc(cond bool, t, f any) any {
	if cond {
		return t
	}
	return f
}

func coalesceFunc(vals ...any) any {
	for _, v := range vals {
		if !emptyFunc(v) {
			return v
		}
	}
	return nil
}

func toFloat64(v any) float64 {
	switch n := v.(type) {
	case int:
		return float64(n)
	case int8:
		return float64(n)
	case int16:
		return float64(n)
	case int32:
		return float64(n)
	case int64:
		return float64(n)
	case uint:
		return float64(n)
	case uint8:
		return float64(n)
	case uint16:
		return float64(n)
	case uint32:
		return float64(n)
	case uint64:
		return float64(n)
	case float32:
		return float64(n)
	case float64:
		return n
	}
	return 0
}

func toInt64(v any) int64 {
	switch n := v.(type) {
	case int:
		return int64(n)
	case int8:
		return int64(n)
	case int16:
		return int64(n)
	case int32:
		return int64(n)
	case int64:
		return n
	case uint:
		return int64(n)
	case uint8:
		return int64(n)
	case uint16:
		return int64(n)
	case uint32:
		return int64(n)
	case uint64:
		return int64(n)
	case float32:
		return int64(n)
	case float64:
		return int64(n)
	}
	return 0
}
