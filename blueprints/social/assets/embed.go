// Package assets provides embedded static files and templates.
package assets

import (
	"embed"
	"html/template"
	"io/fs"
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

// Templates parses and returns all templates.
func Templates() (*template.Template, error) {
	funcs := template.FuncMap{
		"formatTime":     formatTime,
		"formatRelative": formatRelative,
		"truncate":       truncate,
		"add":            func(a, b int) int { return a + b },
		"sub":            func(a, b int) int { return a - b },
		"formatNumber":   formatNumber,
	}

	return template.New("").Funcs(funcs).ParseFS(viewsFS, "views/default/layouts/*.html", "views/default/pages/*.html", "views/default/components/*.html")
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
