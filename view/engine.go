package view

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
)

// Engine is the view template engine.
type Engine struct {
	opts Options
	fs   fs.FS

	mu           sync.RWMutex
	contentCache map[string]string // caches raw template content
	baseFuncs    template.FuncMap
}

// New creates a new view engine with the given options.
func New(opts Options) *Engine {
	opts.applyDefaults()

	e := &Engine{
		opts:         opts,
		contentCache: make(map[string]string),
		baseFuncs:    baseFuncs(),
	}

	// Merge user funcs
	if opts.Funcs != nil {
		for k, v := range opts.Funcs {
			e.baseFuncs[k] = v
		}
	}

	// Setup filesystem
	if opts.FS != nil {
		// Use provided FS (e.g., embed.FS)
		e.fs = opts.FS
	} else {
		// Use OS filesystem
		e.fs = os.DirFS(opts.Dir)
	}

	return e
}

// Preload loads and caches all template content.
// Call this at startup in production to fail fast on template errors.
func (e *Engine) Preload() error {
	// Load all layouts
	if err := e.preloadDir("layouts"); err != nil {
		return fmt.Errorf("load layouts: %w", err)
	}

	// Load all pages
	if err := e.preloadDir("pages"); err != nil {
		return fmt.Errorf("load pages: %w", err)
	}

	// Load all components
	if err := e.preloadDir("components"); err != nil {
		return fmt.Errorf("load components: %w", err)
	}

	// Load all partials
	if err := e.preloadDir("partials"); err != nil {
		return fmt.Errorf("load partials: %w", err)
	}

	return nil
}

// preloadDir loads all template content from a directory.
func (e *Engine) preloadDir(dir string) error {
	return e.walkDir(dir, func(name string) error {
		cacheKey := dir + "/" + name
		content, err := e.loadContent(dir, name)
		if err != nil {
			return err
		}

		// Verify template parses correctly
		_, err = e.parseTemplate(name, content)
		if err != nil {
			return err
		}

		e.mu.Lock()
		e.contentCache[cacheKey] = content
		e.mu.Unlock()

		return nil
	})
}

// ClearCache clears the template cache.
func (e *Engine) ClearCache() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.contentCache = make(map[string]string)
}

// Render renders a page template to the writer.
func (e *Engine) Render(w io.Writer, name string, data any, opts ...RenderOption) error {
	cfg := &renderConfig{
		status: 200,
		layout: e.opts.DefaultLayout,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	// Create render context
	ctx := newRenderContext(e)

	// Build page data
	pageData := &PageData{
		Page: PageMeta{
			Name:   name,
			Layout: cfg.layout,
		},
		Data: data,
	}

	// Get page content and parse with context functions
	pageContent, err := e.getContent("pages", name)
	if err != nil {
		return err
	}

	pageTmpl, err := e.parseTemplateWithContext(name, pageContent, ctx, pageData)
	if err != nil {
		return err
	}

	// Extract layout from page template if defined
	if layoutDef := pageTmpl.Lookup("layout"); layoutDef != nil {
		var layoutBuf bytes.Buffer
		if err := layoutDef.Execute(&layoutBuf, nil); err == nil {
			layoutName := strings.TrimSpace(layoutBuf.String())
			if layoutName != "" {
				cfg.layout = layoutName
				pageData.Page.Layout = layoutName
			}
		}
	}

	// Render page content to buffer
	var pageBuf bytes.Buffer

	// Execute page template
	if contentDef := pageTmpl.Lookup("content"); contentDef != nil {
		if err := contentDef.Execute(&pageBuf, pageData); err != nil {
			return &TemplateError{Name: name, Err: err}
		}
	} else {
		if err := pageTmpl.Execute(&pageBuf, pageData); err != nil {
			return &TemplateError{Name: name, Err: err}
		}
	}

	// Store page content in slot
	ctx.setSlot("content", pageBuf.String())

	// Extract other slot definitions from page
	for _, t := range pageTmpl.Templates() {
		tname := t.Name()
		if tname != "" && tname != name && tname != "layout" && tname != "content" {
			var slotBuf bytes.Buffer
			if err := t.Execute(&slotBuf, pageData); err != nil {
				return &TemplateError{Name: name, Err: fmt.Errorf("slot %q: %w", tname, err)}
			}
			ctx.setSlot(tname, slotBuf.String())
		}
	}

	// If no layout, output page directly
	if cfg.noLayout {
		_, err := w.Write(pageBuf.Bytes())
		return err
	}

	// Get layout content and parse with context functions
	layoutContent, err := e.getContent("layouts", cfg.layout)
	if err != nil {
		return err
	}

	layoutTmpl, err := e.parseTemplateWithContext(cfg.layout, layoutContent, ctx, pageData)
	if err != nil {
		return err
	}

	// Execute layout
	if err := layoutTmpl.Execute(w, pageData); err != nil {
		return &TemplateError{Name: cfg.layout, Err: err}
	}

	return nil
}

// RenderComponent renders a component template directly.
func (e *Engine) RenderComponent(w io.Writer, name string, data any) error {
	ctx := newRenderContext(e)
	return e.renderComponent(w, name, data, ctx)
}

// renderComponent is the internal component renderer.
func (e *Engine) renderComponent(w io.Writer, name string, data any, ctx *renderContext) error {
	content, err := e.getContent("components", name)
	if err != nil {
		return err
	}

	tmpl, err := e.parseTemplateWithContext(name, content, ctx, data)
	if err != nil {
		return err
	}

	if err := tmpl.Execute(w, data); err != nil {
		return &TemplateError{Name: name, Err: err}
	}

	return nil
}

// renderPartial renders a partial template.
func (e *Engine) renderPartial(w io.Writer, name string, data any, ctx *renderContext) error {
	content, err := e.getContent("partials", name)
	if err != nil {
		return err
	}

	tmpl, err := e.parseTemplateWithContext(name, content, ctx, data)
	if err != nil {
		return err
	}

	if err := tmpl.Execute(w, data); err != nil {
		return &TemplateError{Name: name, Err: err}
	}

	return nil
}

// getContent retrieves template content from cache or loads from filesystem.
func (e *Engine) getContent(kind, name string) (string, error) {
	cacheKey := kind + "/" + name

	// In development mode, always reload from filesystem
	if e.opts.Development {
		return e.loadContent(kind, name)
	}

	// Check cache
	e.mu.RLock()
	content, ok := e.contentCache[cacheKey]
	e.mu.RUnlock()

	if ok {
		return content, nil
	}

	// Load and cache
	content, err := e.loadContent(kind, name)
	if err != nil {
		return "", err
	}

	e.mu.Lock()
	e.contentCache[cacheKey] = content
	e.mu.Unlock()

	return content, nil
}

// loadContent loads template content from the filesystem.
func (e *Engine) loadContent(kind, name string) (string, error) {
	filePath := path.Join(kind, name+e.opts.Extension)

	content, err := fs.ReadFile(e.fs, filePath)
	if err != nil {
		if os.IsNotExist(err) {
			typeStr := "page"
			switch kind {
			case "layouts":
				typeStr = "layout"
			case "components":
				typeStr = "component"
			case "partials":
				typeStr = "partial"
			}
			return "", &NotFoundError{Type: typeStr, Name: name}
		}
		return "", fmt.Errorf("read template %s: %w", filePath, err)
	}

	return string(content), nil
}

// parseTemplate parses a template with base functions.
func (e *Engine) parseTemplate(name, content string) (*template.Template, error) {
	tmpl := template.New(name).
		Delims(e.opts.Delims[0], e.opts.Delims[1]).
		Funcs(e.baseFuncs)

	tmpl, err := tmpl.Parse(content)
	if err != nil {
		return nil, &TemplateError{Name: name, Err: err}
	}

	return tmpl, nil
}

// parseTemplateWithContext parses a template with context-specific functions.
func (e *Engine) parseTemplateWithContext(name, content string, ctx *renderContext, data any) (*template.Template, error) {
	// Create context-aware functions
	contextFuncs := template.FuncMap{
		"slot": func(slotName string, defaults ...any) template.HTML {
			return ctx.slot(slotName, defaults...)
		},
		"stack": func(stackName string) template.HTML {
			return ctx.stack(stackName, e.opts.DedupeStacks)
		},
		"push": func(stackName string) string {
			return ""
		},
		"component": func(compName string, compData ...any) template.HTML {
			var d any
			if len(compData) > 0 {
				d = compData[0]
			}
			var buf bytes.Buffer
			if err := e.renderComponent(&buf, compName, d, ctx); err != nil {
				if e.opts.Development {
					return template.HTML(fmt.Sprintf("<!-- component error: %v -->", err))
				}
				return ""
			}
			return template.HTML(buf.String())
		},
		"partial": func(partialName string, partialData ...any) template.HTML {
			var d any = data
			if len(partialData) > 0 {
				d = partialData[0]
			}
			var buf bytes.Buffer
			if err := e.renderPartial(&buf, partialName, d, ctx); err != nil {
				if e.opts.Development {
					return template.HTML(fmt.Sprintf("<!-- partial error: %v -->", err))
				}
				return ""
			}
			return template.HTML(buf.String())
		},
		"children": func() template.HTML {
			return ctx.children()
		},
	}

	// Merge context funcs with base funcs
	funcs := make(template.FuncMap)
	for k, v := range e.baseFuncs {
		funcs[k] = v
	}
	for k, v := range contextFuncs {
		funcs[k] = v
	}

	tmpl := template.New(name).
		Delims(e.opts.Delims[0], e.opts.Delims[1]).
		Funcs(funcs)

	tmpl, err := tmpl.Parse(content)
	if err != nil {
		return nil, &TemplateError{Name: name, Err: err}
	}

	return tmpl, nil
}

// walkDir walks a directory and calls fn for each template file.
func (e *Engine) walkDir(dir string, fn func(name string) error) error {
	return fs.WalkDir(e.fs, dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				// Directory doesn't exist, skip silently
				return fs.SkipDir
			}
			return err
		}

		if d.IsDir() {
			return nil
		}

		// Skip non-template files
		if !strings.HasSuffix(p, e.opts.Extension) {
			return nil
		}

		// Calculate template name (relative path without extension)
		name := strings.TrimPrefix(p, dir+"/")
		name = strings.TrimSuffix(name, e.opts.Extension)
		// Normalize path separator
		name = filepath.ToSlash(name)

		return fn(name)
	})
}

