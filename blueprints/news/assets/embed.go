package assets

import (
	"embed"
	"html/template"
	"io/fs"
	"path/filepath"
	"strings"
	"time"
)

//go:embed static views
var FS embed.FS

// Static returns the static files filesystem.
func Static() fs.FS {
	sub, _ := fs.Sub(FS, "static")
	return sub
}

// Views returns the views filesystem for the default theme.
func Views() fs.FS {
	sub, _ := fs.Sub(FS, "views/default")
	return sub
}

// ViewsForTheme returns the views filesystem for a specific theme.
// Themes inherit from default - theme files override default files.
func ViewsForTheme(theme string) fs.FS {
	if theme == "" || theme == "default" {
		return Views()
	}
	return &themeFS{
		theme:    theme,
		base:     FS,
		default_: "views/default",
		overlay:  "views/" + theme,
	}
}

// themeFS implements fs.FS with theme inheritance.
// Files from the overlay (theme) directory take precedence over the default.
type themeFS struct {
	theme    string
	base     fs.FS
	default_ string
	overlay  string
}

func (t *themeFS) Open(name string) (fs.File, error) {
	// Try overlay first
	if f, err := t.base.Open(t.overlay + "/" + name); err == nil {
		return f, nil
	}
	// Fall back to default
	return t.base.Open(t.default_ + "/" + name)
}

func (t *themeFS) ReadDir(name string) ([]fs.DirEntry, error) {
	// Merge directory listings from both default and overlay
	entries := make(map[string]fs.DirEntry)

	// Read default first
	if dir, ok := t.base.(fs.ReadDirFS); ok {
		defaultPath := t.default_
		if name != "." {
			defaultPath = t.default_ + "/" + name
		}
		if list, err := dir.ReadDir(defaultPath); err == nil {
			for _, e := range list {
				entries[e.Name()] = e
			}
		}
	}

	// Read overlay (overrides default entries)
	if dir, ok := t.base.(fs.ReadDirFS); ok {
		overlayPath := t.overlay
		if name != "." {
			overlayPath = t.overlay + "/" + name
		}
		if list, err := dir.ReadDir(overlayPath); err == nil {
			for _, e := range list {
				entries[e.Name()] = e
			}
		}
	}

	// Convert map to sorted slice
	result := make([]fs.DirEntry, 0, len(entries))
	for _, e := range entries {
		result = append(result, e)
	}
	return result, nil
}

func (t *themeFS) ReadFile(name string) ([]byte, error) {
	// Try overlay first
	if f, ok := t.base.(fs.ReadFileFS); ok {
		if data, err := f.ReadFile(t.overlay + "/" + name); err == nil {
			return data, nil
		}
	}
	// Fall back to default
	if f, ok := t.base.(fs.ReadFileFS); ok {
		return f.ReadFile(t.default_ + "/" + name)
	}
	return nil, fs.ErrNotExist
}

func (t *themeFS) Glob(pattern string) ([]string, error) {
	matches := make(map[string]bool)

	// Glob default - use fs.Glob which works with any fs.FS
	if list, err := fs.Glob(t.base, t.default_+"/"+pattern); err == nil {
		for _, m := range list {
			// Remove the prefix to get relative path
			rel := m[len(t.default_)+1:]
			matches[rel] = true
		}
	}

	// Glob overlay - use fs.Glob which works with any fs.FS
	if list, err := fs.Glob(t.base, t.overlay+"/"+pattern); err == nil {
		for _, m := range list {
			rel := m[len(t.overlay)+1:]
			matches[rel] = true
		}
	}

	result := make([]string, 0, len(matches))
	for m := range matches {
		result = append(result, m)
	}
	return result, nil
}

// templateFuncs returns the template function map.
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"formatTime":         formatTime,
		"formatTimeRelative": formatTimeRelative,
		"formatNumber":       formatNumber,
		"formatScore":        formatScore,
		"truncate":           truncate,
		"domain":             domain,
		"add":                add,
		"sub":                sub,
		"mul":                mul,
		"dict":               dict,
		"list":               list,
		"safeHTML":           safeHTML,
		"indent":             indent,
	}
}

// LoadTemplates loads and returns all templates as a map keyed by page name.
// Each page gets its own isolated template to avoid content block collisions.
func LoadTemplates() (map[string]*template.Template, error) {
	return LoadTemplatesForTheme("default")
}

// LoadTemplatesForTheme loads templates for a specific theme with inheritance from default.
func LoadTemplatesForTheme(theme string) (map[string]*template.Template, error) {
	views := ViewsForTheme(theme)

	// Create base template with layouts and components
	baseTmpl := template.New("").Funcs(templateFuncs())

	// Parse layout
	layoutContent, err := fs.ReadFile(views, "layouts/default.html")
	if err != nil {
		return nil, err
	}
	_, err = baseTmpl.New("layout").Parse(string(layoutContent))
	if err != nil {
		return nil, err
	}

	// Parse components - they define their own template names inside
	componentFiles := []string{"components/nav.html", "components/story_card.html", "components/comment.html"}
	for _, f := range componentFiles {
		content, err := fs.ReadFile(views, f)
		if err != nil {
			return nil, err
		}
		_, err = baseTmpl.Parse(string(content))
		if err != nil {
			return nil, err
		}
	}

	// Find all page files
	pageFiles, err := fs.Glob(views, "pages/*.html")
	if err != nil {
		return nil, err
	}

	// Create a map of templates, one per page
	templates := make(map[string]*template.Template)

	for _, pageFile := range pageFiles {
		pageName := filepath.Base(pageFile) // e.g., "home.html"

		// Clone the base template so each page is isolated
		pageTemplate, err := baseTmpl.Clone()
		if err != nil {
			return nil, err
		}

		// Parse the page file
		content, err := fs.ReadFile(views, pageFile)
		if err != nil {
			return nil, err
		}
		_, err = pageTemplate.New(pageName).Parse(string(content))
		if err != nil {
			return nil, err
		}

		templates[pageName] = pageTemplate
	}

	return templates, nil
}

// GetStatic returns static file content and content type.
func GetStatic(path string) ([]byte, string, error) {
	data, err := fs.ReadFile(Static(), path)
	if err != nil {
		return nil, "", err
	}

	contentType := "application/octet-stream"
	switch {
	case strings.HasSuffix(path, ".css"):
		contentType = "text/css; charset=utf-8"
	case strings.HasSuffix(path, ".js"):
		contentType = "application/javascript; charset=utf-8"
	case strings.HasSuffix(path, ".svg"):
		contentType = "image/svg+xml"
	case strings.HasSuffix(path, ".png"):
		contentType = "image/png"
	case strings.HasSuffix(path, ".ico"):
		contentType = "image/x-icon"
	}

	return data, contentType, nil
}

// Template functions

func formatTime(t time.Time) string {
	return t.Format("Jan 2, 2006 at 3:04 PM")
}

func formatTimeRelative(t time.Time) string {
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
		return formatInt(mins) + " minutes ago"
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return formatInt(hours) + " hours ago"
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return formatInt(days) + " days ago"
	case diff < 30*24*time.Hour:
		weeks := int(diff.Hours() / 24 / 7)
		if weeks == 1 {
			return "1 week ago"
		}
		return formatInt(weeks) + " weeks ago"
	case diff < 365*24*time.Hour:
		months := int(diff.Hours() / 24 / 30)
		if months == 1 {
			return "1 month ago"
		}
		return formatInt(months) + " months ago"
	default:
		years := int(diff.Hours() / 24 / 365)
		if years == 1 {
			return "1 year ago"
		}
		return formatInt(years) + " years ago"
	}
}

func formatInt(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + formatInt(-n)
	}
	result := ""
	for n > 0 {
		result = string(rune('0'+n%10)) + result
		n /= 10
	}
	return result
}

func formatNumber(n int64) string {
	if n < 1000 {
		return formatInt(int(n))
	}
	if n < 1000000 {
		return formatFloat(float64(n)/1000) + "k"
	}
	return formatFloat(float64(n)/1000000) + "m"
}

func formatFloat(f float64) string {
	if f == float64(int64(f)) {
		return formatInt(int(f))
	}
	// One decimal place
	return formatInt(int(f)) + "." + formatInt(int((f-float64(int(f)))*10))
}

func formatScore(score int64) string {
	if score < 0 {
		return "-" + formatNumber(-score)
	}
	return formatNumber(score)
}

func truncate(s string, length int) string {
	runes := []rune(s)
	if len(runes) <= length {
		return s
	}
	return string(runes[:length-3]) + "..."
}

func domain(url string) string {
	// Extract domain from URL
	s := url
	// Remove protocol
	if idx := strings.Index(s, "://"); idx != -1 {
		s = s[idx+3:]
	}
	// Remove path
	if idx := strings.Index(s, "/"); idx != -1 {
		s = s[:idx]
	}
	// Remove www.
	s = strings.TrimPrefix(s, "www.")
	return s
}

func add(a, b int) int {
	return a + b
}

func sub(a, b int) int {
	return a - b
}

func mul(a, b int) int {
	return a * b
}

func dict(values ...any) map[string]any {
	result := make(map[string]any)
	for i := 0; i < len(values); i += 2 {
		if i+1 < len(values) {
			key, ok := values[i].(string)
			if ok {
				result[key] = values[i+1]
			}
		}
	}
	return result
}

func list(values ...any) []any {
	return values
}

func safeHTML(s string) template.HTML {
	return template.HTML(s)
}

func indent(depth int) int {
	return depth * 40 // 40px per indent level
}
