package assets

import (
	"embed"
	"html/template"
	"io/fs"
	"path/filepath"
)

//go:embed views static
var content embed.FS

// Templates parses and returns all templates.
func Templates() (map[string]*template.Template, error) {
	templates := make(map[string]*template.Template)

	// Define template functions
	funcs := template.FuncMap{
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
		"safeJS": func(s string) template.JS {
			return template.JS(s)
		},
		"safeCSS": func(s string) template.CSS {
			return template.CSS(s)
		},
	}

	// Parse layouts
	defaultLayout, err := content.ReadFile("views/layouts/default.html")
	if err != nil {
		return nil, err
	}

	authLayout, err := content.ReadFile("views/layouts/auth.html")
	if err != nil {
		return nil, err
	}

	// Parse pages and combine with layouts
	pages, err := fs.Glob(content, "views/pages/*.html")
	if err != nil {
		return nil, err
	}

	for _, page := range pages {
		name := filepath.Base(page)
		name = name[:len(name)-5] // Remove .html

		pageContent, err := content.ReadFile(page)
		if err != nil {
			return nil, err
		}

		// Choose layout based on page type
		var layout []byte
		if name == "login" || name == "register" {
			layout = authLayout
		} else {
			layout = defaultLayout
		}

		// Parse layout + page
		tmpl, err := template.New("layout").Funcs(funcs).Parse(string(layout))
		if err != nil {
			return nil, err
		}

		tmpl, err = tmpl.Parse(string(pageContent))
		if err != nil {
			return nil, err
		}

		templates[name] = tmpl
	}

	return templates, nil
}

// Static returns the static file system.
func Static() fs.FS {
	static, _ := fs.Sub(content, "static")
	return static
}
