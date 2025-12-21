// Package assets provides embedded static files and templates.
package assets

import (
	"embed"
	"html/template"
	"io/fs"
)

//go:embed static views
var FS embed.FS

// Static returns the static files filesystem.
func Static() fs.FS {
	sub, _ := fs.Sub(FS, "static")
	return sub
}

// Views returns the views filesystem.
func Views() fs.FS {
	sub, _ := fs.Sub(FS, "views")
	return sub
}

// Templates parses all view templates.
func Templates() (*template.Template, error) {
	tmpl := template.New("")

	// Add custom functions
	tmpl = tmpl.Funcs(template.FuncMap{
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
	})

	return tmpl.ParseFS(Views(), "layouts/*.html", "pages/*.html", "components/*.html")
}
