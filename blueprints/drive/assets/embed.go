// Package assets provides embedded static files and HTML templates.
package assets

import (
	"embed"
	"html/template"
	"io/fs"
)

//go:embed static
var staticFS embed.FS

//go:embed views
var viewsFS embed.FS

// Static returns the static file system.
func Static() fs.FS {
	sub, _ := fs.Sub(staticFS, "static")
	return sub
}

// Views returns the views file system.
func Views() fs.FS {
	sub, _ := fs.Sub(viewsFS, "views")
	return sub
}

// Templates parses all templates.
func Templates() (*template.Template, error) {
	return template.New("").Funcs(template.FuncMap{
		"lower": func(s string) string {
			return s
		},
	}).ParseFS(viewsFS, "views/layouts/*.html", "views/pages/*.html")
}
