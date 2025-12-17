package view

import (
	"html/template"
	"io/fs"
)

// Options configures the view engine.
type Options struct {
	// Dir is the views directory path.
	// Default: "views"
	Dir string

	// FS is an optional filesystem (for embed.FS in production).
	// If nil, uses os filesystem with Dir.
	FS fs.FS

	// Extension is the template file extension.
	// Default: ".html"
	Extension string

	// DefaultLayout is the layout used when page doesn't specify one.
	// Default: "default"
	DefaultLayout string

	// Funcs adds custom template functions.
	Funcs template.FuncMap

	// Delims sets custom template delimiters.
	// Default: "{{", "}}"
	Delims [2]string

	// Development enables dev mode features:
	// - Template reload on every request
	// - Detailed error pages with source context
	// - No caching
	// Default: false
	Development bool

	// StrictMode fails on missing slots/components instead of empty output.
	// Default: false (production-friendly)
	StrictMode bool

	// DedupeStacks removes duplicate stack entries.
	// Default: true
	DedupeStacks bool
}

// applyDefaults fills in default values for unset options.
func (o *Options) applyDefaults() {
	if o.Dir == "" {
		o.Dir = "views"
	}
	if o.Extension == "" {
		o.Extension = ".html"
	}
	if o.DefaultLayout == "" {
		o.DefaultLayout = "default"
	}
	if o.Delims[0] == "" {
		o.Delims[0] = "{{"
	}
	if o.Delims[1] == "" {
		o.Delims[1] = "}}"
	}
}

// Data is a convenience type for template data.
type Data map[string]any

// PageData is the structured data passed to page templates.
type PageData struct {
	// Page contains page metadata.
	Page PageMeta

	// Data contains user-provided data.
	Data any

	// CSRF token if middleware is enabled.
	CSRF string
}

// PageMeta contains page metadata.
type PageMeta struct {
	// Name is the page name (e.g., "users/show").
	Name string

	// Title is the page title.
	Title string

	// Layout is the layout name.
	Layout string
}

// RenderOption configures a render call.
type RenderOption func(*renderConfig)

// renderConfig holds render configuration.
type renderConfig struct {
	status   int
	layout   string
	noLayout bool
}

// Status sets the HTTP status code for the response.
func Status(code int) RenderOption {
	return func(cfg *renderConfig) {
		cfg.status = code
	}
}

// Layout sets the layout to use for rendering.
func Layout(name string) RenderOption {
	return func(cfg *renderConfig) {
		cfg.layout = name
	}
}

// NoLayout disables layout rendering.
func NoLayout() RenderOption {
	return func(cfg *renderConfig) {
		cfg.noLayout = true
	}
}
