// Package assets provides embedded static files and templates.
package assets

import (
	"embed"
	"html/template"
	"io/fs"
)

// toFloat64 converts various numeric types to float64.
func toFloat64(v interface{}) float64 {
	switch n := v.(type) {
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case float64:
		return n
	case float32:
		return float64(n)
	default:
		return 0
	}
}

//go:embed static/*
var staticFS embed.FS

//go:embed views/*
var viewsFS embed.FS

// Available themes
var Themes = []string{"default", "trello"}

// Static returns the static files filesystem.
func Static() fs.FS {
	sub, _ := fs.Sub(staticFS, "static")
	return sub
}

// Templates parses and returns all templates for the default theme.
func Templates() (map[string]*template.Template, error) {
	return TemplatesForTheme("default")
}

// TemplatesForTheme parses and returns all templates for a specific theme.
func TemplatesForTheme(theme string) (map[string]*template.Template, error) {
	templates := make(map[string]*template.Template)

	// Template functions
	funcMap := template.FuncMap{
		"lower": func(s string) string {
			// Simple lowercase for status classes
			result := ""
			for _, c := range s {
				if c >= 'A' && c <= 'Z' {
					result += string(c + 32)
				} else if c == ' ' {
					result += "-"
				} else {
					result += string(c)
				}
			}
			return result
		},
		"slice": func(s string, start, end int) string {
			if start >= len(s) {
				return ""
			}
			if end > len(s) {
				end = len(s)
			}
			return s[start:end]
		},
		"div": func(a, b interface{}) float64 {
			af := toFloat64(a)
			bf := toFloat64(b)
			if bf == 0 {
				return 0
			}
			return af / bf
		},
		"mul": func(a, b interface{}) float64 {
			return toFloat64(a) * toFloat64(b)
		},
		"float64": func(i interface{}) float64 {
			return toFloat64(i)
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"mod": func(a, b int) int {
			return a % b
		},
	}

	// Read the layout for the theme
	layoutBytes, err := viewsFS.ReadFile("views/" + theme + "/layouts/default.html")
	if err != nil {
		// Fall back to default theme
		layoutBytes, err = viewsFS.ReadFile("views/default/layouts/default.html")
		if err != nil {
			return nil, err
		}
	}
	layoutContent := string(layoutBytes)

	// Read auth layout
	authLayoutBytes, err := viewsFS.ReadFile("views/" + theme + "/layouts/auth.html")
	if err != nil {
		authLayoutBytes, err = viewsFS.ReadFile("views/default/layouts/auth.html")
		if err != nil {
			// Use default layout for auth if no auth layout exists
			authLayoutBytes = layoutBytes
		}
	}
	authLayoutContent := string(authLayoutBytes)

	// Pages using the main layout
	mainPages := []string{"home", "inbox", "board", "issues", "issue", "cycles", "activities", "team", "workspace-settings", "project-settings", "project-fields", "calendar", "gantt"}
	for _, name := range mainPages {
		pageBytes, err := viewsFS.ReadFile("views/" + theme + "/pages/" + name + ".html")
		if err != nil {
			pageBytes, err = viewsFS.ReadFile("views/default/pages/" + name + ".html")
			if err != nil {
				continue
			}
		}

		tmpl, err := template.New(name).Funcs(funcMap).Parse(layoutContent)
		if err != nil {
			return nil, err
		}

		tmpl, err = tmpl.Parse(string(pageBytes))
		if err != nil {
			return nil, err
		}

		templates[name] = tmpl
	}

	// Pages using the auth layout
	authPages := []string{"login", "register"}
	for _, name := range authPages {
		pageBytes, err := viewsFS.ReadFile("views/" + theme + "/pages/" + name + ".html")
		if err != nil {
			pageBytes, err = viewsFS.ReadFile("views/default/pages/" + name + ".html")
			if err != nil {
				continue
			}
		}

		tmpl, err := template.New(name).Funcs(funcMap).Parse(authLayoutContent)
		if err != nil {
			return nil, err
		}

		tmpl, err = tmpl.Parse(string(pageBytes))
		if err != nil {
			return nil, err
		}

		templates[name] = tmpl
	}

	// Trello theme templates (if theme is trello or loading all)
	if theme == "trello" {
		// Trello auth layout
		trelloAuthLayoutBytes, err := viewsFS.ReadFile("views/trello/layouts/auth.html")
		if err == nil {
			trelloAuthLayoutContent := string(trelloAuthLayoutBytes)

			// Trello auth pages
			trelloAuthPages := []string{"login", "register"}
			for _, name := range trelloAuthPages {
				pageBytes, err := viewsFS.ReadFile("views/trello/pages/" + name + ".html")
				if err != nil {
					continue
				}

				tmpl, err := template.New("trello-" + name).Funcs(funcMap).Parse(trelloAuthLayoutContent)
				if err != nil {
					return nil, err
				}

				tmpl, err = tmpl.Parse(string(pageBytes))
				if err != nil {
					return nil, err
				}

				templates["trello-"+name] = tmpl
			}
		}

		// Trello main layout
		trelloLayoutBytes, err := viewsFS.ReadFile("views/trello/layouts/default.html")
		if err == nil {
			trelloLayoutContent := string(trelloLayoutBytes)

			// Trello main pages
			trelloMainPages := []string{"boards", "board", "card"}
			for _, name := range trelloMainPages {
				pageBytes, err := viewsFS.ReadFile("views/trello/pages/" + name + ".html")
				if err != nil {
					continue
				}

				tmpl, err := template.New("trello-" + name).Funcs(funcMap).Parse(trelloLayoutContent)
				if err != nil {
					return nil, err
				}

				tmpl, err = tmpl.Parse(string(pageBytes))
				if err != nil {
					return nil, err
				}

				templates["trello-"+name] = tmpl
			}
		}
	}

	return templates, nil
}

// AllTemplates parses and returns templates for all themes.
func AllTemplates() (map[string]map[string]*template.Template, error) {
	allTemplates := make(map[string]map[string]*template.Template)

	for _, theme := range Themes {
		templates, err := TemplatesForTheme(theme)
		if err != nil {
			return nil, err
		}
		allTemplates[theme] = templates
	}

	return allTemplates, nil
}
