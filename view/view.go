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
	"strings"
	"sync"

	"github.com/go-mizu/mizu"
)

// ErrNotFound is returned when a template is not found.
var ErrNotFound = errors.New("view: not found")

// Error wraps a template error with context.
type Error struct {
	Kind string // "page", "layout", "template"
	Name string
	Err  error
}

func (e *Error) Error() string {
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
	cache map[string]*template.Template
	funcs template.FuncMap
}

// New creates a new view engine.
func New(cfg Config) *Engine {
	cfg.defaults()
	e := &Engine{
		cfg:   cfg,
		cache: make(map[string]*template.Template),
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
	for _, dir := range []string{"layouts", "pages"} {
		if err := e.loadDir(dir); err != nil {
			return fmt.Errorf("load %s: %w", dir, err)
		}
	}
	return nil
}

// Clear clears the template cache.
func (e *Engine) Clear() {
	e.mu.Lock()
	e.cache = make(map[string]*template.Template)
	e.mu.Unlock()
}

// Render renders a page template with an optional layout.
func (e *Engine) Render(w io.Writer, page string, data any, opts ...option) error {
	cfg := &renderCfg{layout: e.cfg.DefaultLayout}
	for _, opt := range opts {
		opt(cfg)
	}

	pd := &pageData{
		Page: pageMeta{Name: page, Layout: cfg.layout},
		Data: data,
	}

	// Get page template
	tmpl, err := e.template("pages", page)
	if err != nil {
		return err
	}

	// Render page content
	var pageBuf bytes.Buffer
	if err := tmpl.Execute(&pageBuf, pd); err != nil {
		return &Error{Kind: "page", Name: page, Err: err}
	}

	if cfg.noLayout {
		_, err := w.Write(pageBuf.Bytes())
		return err
	}

	// Set rendered content for layout
	pd.Content = template.HTML(pageBuf.String()) //nolint:gosec

	// Get layout template
	layoutTmpl, err := e.template("layouts", cfg.layout)
	if err != nil {
		return err
	}

	if err := layoutTmpl.Execute(w, pd); err != nil {
		return &Error{Kind: "layout", Name: cfg.layout, Err: err}
	}
	return nil
}

func (e *Engine) template(kind, name string) (*template.Template, error) {
	key := kind + "/" + name

	// In development mode, always parse fresh
	if e.cfg.Development {
		content, err := e.load(kind, name)
		if err != nil {
			return nil, err
		}
		return e.parse(name, content)
	}

	// Check cache first
	e.mu.RLock()
	tmpl, ok := e.cache[key]
	e.mu.RUnlock()
	if ok {
		return tmpl, nil
	}

	// Load and parse template
	content, err := e.load(kind, name)
	if err != nil {
		return nil, err
	}
	tmpl, err = e.parse(name, content)
	if err != nil {
		return nil, err
	}

	// Cache the parsed template
	e.mu.Lock()
	e.cache[key] = tmpl
	e.mu.Unlock()

	return tmpl, nil
}

func (e *Engine) load(kind, name string) (string, error) {
	p := path.Join(kind, name+e.cfg.Extension)
	data, err := fs.ReadFile(e.fs, p)
	if err != nil {
		if os.IsNotExist(err) {
			k := "page"
			if kind == "layouts" {
				k = "layout"
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
		tmpl, err := e.parse(name, content)
		if err != nil {
			return err
		}
		e.mu.Lock()
		e.cache[key] = tmpl
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

// pageData is the data passed to page and layout templates.
type pageData struct {
	Page    pageMeta
	Data    any
	Content template.HTML // Rendered page content (for layouts)
}

type pageMeta struct {
	Name   string
	Layout string
}

// option configures a render call.
type option func(*renderCfg)

type renderCfg struct {
	layout   string
	noLayout bool
}

// Layout sets the layout name.
func Layout(name string) option {
	return func(c *renderCfg) { c.layout = name }
}

// NoLayout disables layout rendering.
func NoLayout() option {
	return func(c *renderCfg) { c.noLayout = true }
}

// Middleware returns a Mizu middleware that adds the engine to context.
func (e *Engine) Middleware() mizu.Middleware {
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
	c.Writer().Header().Set("Content-Type", "text/html; charset=utf-8")
	c.Writer().WriteHeader(200)
	return e.Render(c.Writer(), page, data, opts...)
}

// baseFuncs returns the base template functions.
func baseFuncs() template.FuncMap {
	return template.FuncMap{
		// Data helpers
		"dict": dictFunc,
		"list": listFunc,

		// String helpers
		"upper":     strings.ToUpper,
		"lower":     strings.ToLower,
		"trim":      strings.TrimSpace,
		"contains":  strings.Contains,
		"replace":   strings.ReplaceAll,
		"split":     strings.Split,
		"join":      strings.Join,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,
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

