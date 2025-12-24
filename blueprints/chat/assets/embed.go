// Package assets provides embedded static files and templates.
package assets

import (
	"embed"
	"html/template"
	"io/fs"
)

//go:embed static/* views/**/*.html
var embedded embed.FS

// Static returns the static files filesystem.
func Static() fs.FS {
	sub, _ := fs.Sub(embedded, "static")
	return sub
}

// Templates parses and returns all HTML templates.
func Templates() (*template.Template, error) {
	return template.ParseFS(embedded, "views/**/*.html")
}
