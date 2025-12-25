// Package assets provides embedded static files and templates.
package assets

import (
	"embed"
	"html/template"
	"io/fs"
)

//go:embed static/*
var staticFS embed.FS

//go:embed views/*
var viewsFS embed.FS

// Static returns the static files filesystem.
func Static() fs.FS {
	sub, _ := fs.Sub(staticFS, "static")
	return sub
}

// Templates parses and returns all templates.
func Templates() (map[string]*template.Template, error) {
	templates := make(map[string]*template.Template)

	// Read the layout
	layoutBytes, err := viewsFS.ReadFile("views/default/layouts/default.html")
	if err != nil {
		return nil, err
	}
	layoutContent := string(layoutBytes)

	// Read all pages
	pages := []string{"home", "login", "register", "app", "settings"}
	for _, name := range pages {
		pageBytes, err := viewsFS.ReadFile("views/default/pages/" + name + ".html")
		if err != nil {
			continue
		}

		tmpl, err := template.New(name).Parse(layoutContent)
		if err != nil {
			return nil, err
		}

		tmpl, err = tmpl.Parse(string(pageBytes))
		if err != nil {
			return nil, err
		}

		templates[name] = tmpl
	}

	return templates, nil
}
