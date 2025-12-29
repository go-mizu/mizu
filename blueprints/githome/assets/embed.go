// Package assets provides embedded static files and templates.
package assets

import (
	"embed"
	"html/template"
	"io/fs"
	"strings"
)

//go:embed static/*
var staticFS embed.FS

//go:embed views/*
var viewsFS embed.FS

// Available themes
var Themes = []string{"default"}

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
			return strings.ToLower(s)
		},
		"upper": func(s string) string {
			return strings.ToUpper(s)
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
		"firstChar": func(s string) string {
			if len(s) == 0 {
				return ""
			}
			return strings.ToUpper(string(s[0]))
		},
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"subtract": func(a, b int) int {
			return a - b
		},
		"mul": func(a, b int) int {
			return a * b
		},
		"div": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a / b
		},
		"mod": func(a, b int) int {
			return a % b
		},
		"eq": func(a, b interface{}) bool {
			return a == b
		},
		"ne": func(a, b interface{}) bool {
			return a != b
		},
		"default": func(def, val interface{}) interface{} {
			if val == nil || val == "" {
				return def
			}
			return val
		},
		"truncate": func(s string, n int) string {
			if len(s) <= n {
				return s
			}
			return s[:n] + "..."
		},
		"split": func(s, sep string) []string {
			return strings.Split(s, sep)
		},
	}

	// Read the main layout for the theme
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
	mainPages := []string{
		"home", "explore", "new_repo", "notifications",
		"user_profile",
		"repo_home", "repo_code", "repo_blob", "repo_blame", "repo_issues", "issue_view", "new_issue", "repo_settings",
		"repo_commits", "commit_detail",
	}
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
