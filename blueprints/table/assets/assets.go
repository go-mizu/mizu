package assets

import (
	"embed"
	"html/template"
	"io/fs"
)

//go:embed views static
var content embed.FS

// Templates parses and returns all templates.
func Templates() (*template.Template, error) {
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

	// Parse all templates
	tmpl := template.New("").Funcs(funcs)

	// Parse layouts
	layouts, err := fs.Glob(content, "views/layouts/*.html")
	if err != nil {
		return nil, err
	}
	for _, layout := range layouts {
		data, err := content.ReadFile(layout)
		if err != nil {
			return nil, err
		}
		_, err = tmpl.New(layout).Parse(string(data))
		if err != nil {
			return nil, err
		}
	}

	// Parse pages
	pages, err := fs.Glob(content, "views/pages/*.html")
	if err != nil {
		return nil, err
	}
	for _, page := range pages {
		data, err := content.ReadFile(page)
		if err != nil {
			return nil, err
		}
		// Use just the filename as the template name
		name := page[len("views/pages/"):]
		_, err = tmpl.New(name).Parse(string(data))
		if err != nil {
			return nil, err
		}
	}

	return tmpl, nil
}

// Static returns the static file system.
func Static() fs.FS {
	static, _ := fs.Sub(content, "static")
	return static
}
