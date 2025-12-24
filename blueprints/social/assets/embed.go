// Package assets provides embedded static files and templates.
package assets

import (
	"embed"
	"html/template"
	"io/fs"
	"path/filepath"
	"time"
)

//go:embed static
var staticFS embed.FS

//go:embed views
var viewsFS embed.FS

// Static returns the static files filesystem.
func Static() fs.FS {
	fsys, _ := fs.Sub(staticFS, "static")
	return fsys
}

// PageTemplates holds compiled page templates.
var pageTemplates map[string]*template.Template

// Templates parses and returns all templates.
// Each page template is parsed separately with the base layout to avoid
// block definition conflicts between pages.
func Templates() (*template.Template, error) {
	funcs := template.FuncMap{
		"formatTime":     formatTime,
		"formatRelative": formatRelative,
		"truncate":       truncate,
		"add":            func(a, b int) int { return a + b },
		"sub":            func(a, b int) int { return a - b },
		"formatNumber":   formatNumber,
	}

	// Parse base layout and components first
	base, err := template.New("").Funcs(funcs).ParseFS(viewsFS, "views/default/layouts/*.html", "views/default/components/*.html")
	if err != nil {
		return nil, err
	}

	// Find all page files
	pageFiles, err := fs.Glob(viewsFS, "views/default/pages/*.html")
	if err != nil {
		return nil, err
	}

	// Initialize the page templates map
	pageTemplates = make(map[string]*template.Template)

	// For each page, clone the base and add the page template
	// This ensures each page has its own set of block definitions
	for _, pageFile := range pageFiles {
		pageName := filepath.Base(pageFile)

		// Clone the base template for this page
		pageTemplate, err := base.Clone()
		if err != nil {
			return nil, err
		}

		// Parse this page into the clone - the page's defines become part of this clone only
		pageTemplate, err = pageTemplate.ParseFS(viewsFS, pageFile)
		if err != nil {
			return nil, err
		}

		// Store the complete template for this page
		pageTemplates[pageName] = pageTemplate
	}

	// Return the base template (needed for initialization)
	// The actual page rendering uses GetPageTemplate
	return base, nil
}

// GetPageTemplate returns the compiled template for a specific page.
func GetPageTemplate(name string) *template.Template {
	if pageTemplates == nil {
		return nil
	}
	return pageTemplates[name]
}

func formatTime(t time.Time) string {
	return t.Format("Jan 2, 2006 3:04 PM")
}

func formatRelative(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return template.HTMLEscapeString(string(rune('0'+mins/10))) + string(rune('0'+mins%10)) + " minutes ago"
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return template.HTMLEscapeString(string(rune('0'+hours/10))) + string(rune('0'+hours%10)) + " hours ago"
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return template.HTMLEscapeString(string(rune('0'+days))) + " days ago"
	default:
		return t.Format("Jan 2, 2006")
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func formatNumber(n int) string {
	if n >= 1000000 {
		return template.HTMLEscapeString(string(rune('0'+n/1000000))) + "M"
	}
	if n >= 1000 {
		return template.HTMLEscapeString(string(rune('0'+n/1000))) + "K"
	}
	return template.HTMLEscapeString(string(rune('0'+n/100))) + template.HTMLEscapeString(string(rune('0'+n/10%10))) + template.HTMLEscapeString(string(rune('0'+n%10)))
}
