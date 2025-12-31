// Package assets provides embedded static files and templates for the CMS.
package assets

import (
	"crypto/md5"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"strings"
	"time"
)

//go:embed static/*
var staticFS embed.FS

//go:embed views/*
var viewsFS embed.FS

//go:embed theme/*
var themeFS embed.FS

// ThemeJSON represents the structure of theme.json files.
type ThemeJSON struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Screenshot  string `json:"screenshot"`
	Author      struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"author"`
	Colors     map[string]string `json:"colors"`
	DarkColors map[string]string `json:"dark_colors"`
	Fonts      map[string]string `json:"fonts"`
	Features   map[string]bool   `json:"features"`
	Config     map[string]any    `json:"config"`
}

// ListThemes returns information about all available themes.
func ListThemes() ([]*ThemeJSON, error) {
	var themes []*ThemeJSON

	entries, err := fs.ReadDir(themeFS, "theme")
	if err != nil {
		return nil, fmt.Errorf("read theme directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		themeJSONPath := "theme/" + entry.Name() + "/theme.json"
		data, err := themeFS.ReadFile(themeJSONPath)
		if err != nil {
			continue
		}

		var themeInfo ThemeJSON
		if err := json.Unmarshal(data, &themeInfo); err != nil {
			continue
		}

		themes = append(themes, &themeInfo)
	}

	return themes, nil
}

// GetTheme returns information about a specific theme by slug.
func GetTheme(slug string) (*ThemeJSON, error) {
	themeJSONPath := "theme/" + slug + "/theme.json"
	data, err := themeFS.ReadFile(themeJSONPath)
	if err != nil {
		return nil, fmt.Errorf("read theme.json: %w", err)
	}

	var themeInfo ThemeJSON
	if err := json.Unmarshal(data, &themeInfo); err != nil {
		return nil, fmt.Errorf("parse theme.json: %w", err)
	}

	return &themeInfo, nil
}

// Theme returns the theme files filesystem for a specific theme.
func Theme() fs.FS {
	sub, _ := fs.Sub(themeFS, "theme/default")
	return sub
}

// ThemeBySlug returns the theme files filesystem for a specific theme by slug.
func ThemeBySlug(slug string) fs.FS {
	sub, err := fs.Sub(themeFS, "theme/"+slug)
	if err != nil {
		sub, _ = fs.Sub(themeFS, "theme/default")
	}
	return sub
}

// ThemeAssets returns the theme assets filesystem for the default theme.
func ThemeAssets() fs.FS {
	sub, _ := fs.Sub(themeFS, "theme/default/assets")
	return sub
}

// ThemeAssetsBySlug returns the theme assets filesystem for a specific theme.
func ThemeAssetsBySlug(slug string) fs.FS {
	sub, err := fs.Sub(themeFS, "theme/"+slug+"/assets")
	if err != nil {
		sub, _ = fs.Sub(themeFS, "theme/default/assets")
	}
	return sub
}

// Static returns the static files filesystem.
func Static() fs.FS {
	sub, _ := fs.Sub(staticFS, "static")
	return sub
}

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

// WPAdminTemplates parses and returns all WordPress admin templates.
func WPAdminTemplates() (map[string]*template.Template, error) {
	templates := make(map[string]*template.Template)

	// Template functions
	funcMap := template.FuncMap{
		"lower": func(s string) string {
			return strings.ToLower(strings.ReplaceAll(s, " ", "-"))
		},
		"upper": func(s string) string {
			return strings.ToUpper(s)
		},
		"title": func(s string) string {
			return strings.Title(s)
		},
		"truncate": func(s string, length int) string {
			if len(s) <= length {
				return s
			}
			return s[:length] + "..."
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
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"mod": func(a, b int) int {
			return a % b
		},
		"float64": func(i interface{}) float64 {
			return toFloat64(i)
		},
		"formatDate": func(t time.Time) string {
			return t.Format("2006/01/02")
		},
		"formatDateTime": func(t time.Time) string {
			return t.Format("2006/01/02 at 3:04 pm")
		},
		"formatDateHuman": func(t time.Time) string {
			now := time.Now()
			diff := now.Sub(t)

			if diff < time.Minute {
				return "just now"
			} else if diff < time.Hour {
				mins := int(diff.Minutes())
				if mins == 1 {
					return "1 minute ago"
				}
				return strings.Replace("X minutes ago", "X", string(rune('0'+mins%10)), 1)
			} else if diff < 24*time.Hour {
				hours := int(diff.Hours())
				if hours == 1 {
					return "1 hour ago"
				}
				return t.Format("3:04 pm")
			} else if diff < 7*24*time.Hour {
				return t.Format("Mon 3:04 pm")
			}
			return t.Format("Jan 2, 2006")
		},
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
		"safeURL": func(s string) template.URL {
			return template.URL(s)
		},
		"contains": func(slice []string, item string) bool {
			for _, s := range slice {
				if s == item {
					return true
				}
			}
			return false
		},
		"join": func(slice []string, sep string) string {
			return strings.Join(slice, sep)
		},
		"seq": func(start, end int) []int {
			result := make([]int, end-start+1)
			for i := range result {
				result[i] = start + i
			}
			return result
		},
		"default": func(def, val interface{}) interface{} {
			if val == nil || val == "" {
				return def
			}
			return val
		},
		"eq": func(a, b interface{}) bool {
			return a == b
		},
		"ne": func(a, b interface{}) bool {
			return a != b
		},
		"gt": func(a, b int) bool {
			return a > b
		},
		"lt": func(a, b int) bool {
			return a < b
		},
		"gte": func(a, b int) bool {
			return a >= b
		},
		"lte": func(a, b int) bool {
			return a <= b
		},
		"gravatar": func(email string, size int) string {
			email = strings.ToLower(strings.TrimSpace(email))
			hash := md5.Sum([]byte(email))
			return fmt.Sprintf("https://www.gravatar.com/avatar/%x?s=%d&d=mm", hash, size)
		},
		"avatarURL": func(avatarURL, email string, size int) string {
			if avatarURL != "" {
				return avatarURL
			}
			email = strings.ToLower(strings.TrimSpace(email))
			hash := md5.Sum([]byte(email))
			return fmt.Sprintf("https://www.gravatar.com/avatar/%x?s=%d&d=mm", hash, size)
		},
	}

	// Read the admin layout
	layoutBytes, err := viewsFS.ReadFile("views/wpadmin/layouts/admin.html")
	if err != nil {
		return nil, err
	}
	layoutContent := string(layoutBytes)

	// Read the login layout
	loginLayoutBytes, err := viewsFS.ReadFile("views/wpadmin/layouts/login.html")
	if err != nil {
		return nil, err
	}
	loginLayoutContent := string(loginLayoutBytes)

	// Admin pages using the main layout
	adminPages := []string{
		"dashboard",
		"posts",
		"post-edit",
		"pages-list",
		"page-edit",
		"media",
		"media-edit",
		"comments",
		"comment-edit",
		"users",
		"user-edit",
		"profile",
		"categories",
		"tags",
		"menus",
		"themes",
		"settings-general",
		"settings-writing",
		"settings-reading",
		"settings-discussion",
		"settings-media",
		"settings-permalinks",
	}

	for _, name := range adminPages {
		pageBytes, err := viewsFS.ReadFile("views/wpadmin/pages/" + name + ".html")
		if err != nil {
			continue
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

	// Auth pages using the login layout
	authPages := []string{"login"}
	for _, name := range authPages {
		pageBytes, err := viewsFS.ReadFile("views/wpadmin/pages/" + name + ".html")
		if err != nil {
			continue
		}

		tmpl, err := template.New(name).Funcs(funcMap).Parse(loginLayoutContent)
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

// ObakeTemplates parses and returns all Ghost-compatible admin templates.
func ObakeTemplates() (map[string]*template.Template, error) {
	templates := make(map[string]*template.Template)

	// Template functions
	funcMap := template.FuncMap{
		"lower": func(s string) string {
			return strings.ToLower(strings.ReplaceAll(s, " ", "-"))
		},
		"upper": func(s string) string {
			return strings.ToUpper(s)
		},
		"title": func(s string) string {
			return strings.Title(s)
		},
		"truncate": func(s string, length int) string {
			if len(s) <= length {
				return s
			}
			return s[:length] + "..."
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
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"mod": func(a, b int) int {
			return a % b
		},
		"float64": func(i interface{}) float64 {
			return toFloat64(i)
		},
		"formatDate": func(t time.Time) string {
			return t.Format("2 Jan 2006")
		},
		"formatDateTime": func(t time.Time) string {
			return t.Format("2 Jan 2006, 15:04")
		},
		"formatDateRelative": func(t time.Time) string {
			now := time.Now()
			diff := now.Sub(t)

			if diff < time.Minute {
				return "Just now"
			} else if diff < time.Hour {
				mins := int(diff.Minutes())
				if mins == 1 {
					return "1 min ago"
				}
				return fmt.Sprintf("%d mins ago", mins)
			} else if diff < 24*time.Hour {
				hours := int(diff.Hours())
				if hours == 1 {
					return "1 hour ago"
				}
				return fmt.Sprintf("%d hours ago", hours)
			} else if diff < 7*24*time.Hour {
				days := int(diff.Hours() / 24)
				if days == 1 {
					return "Yesterday"
				}
				return fmt.Sprintf("%d days ago", days)
			}
			return t.Format("2 Jan 2006")
		},
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
		"safeURL": func(s string) template.URL {
			return template.URL(s)
		},
		"contains": func(slice []string, item string) bool {
			for _, s := range slice {
				if s == item {
					return true
				}
			}
			return false
		},
		"join": func(slice []string, sep string) string {
			return strings.Join(slice, sep)
		},
		"seq": func(start, end int) []int {
			result := make([]int, end-start+1)
			for i := range result {
				result[i] = start + i
			}
			return result
		},
		"default": func(def, val interface{}) interface{} {
			if val == nil || val == "" {
				return def
			}
			return val
		},
		"eq": func(a, b interface{}) bool {
			return a == b
		},
		"ne": func(a, b interface{}) bool {
			return a != b
		},
		"gt": func(a, b int) bool {
			return a > b
		},
		"lt": func(a, b int) bool {
			return a < b
		},
		"gte": func(a, b int) bool {
			return a >= b
		},
		"lte": func(a, b int) bool {
			return a <= b
		},
		"gravatar": func(email string, size int) string {
			email = strings.ToLower(strings.TrimSpace(email))
			hash := md5.Sum([]byte(email))
			return fmt.Sprintf("https://www.gravatar.com/avatar/%x?s=%d&d=mm", hash, size)
		},
		"avatarURL": func(avatarURL, email string, size int) string {
			if avatarURL != "" {
				return avatarURL
			}
			email = strings.ToLower(strings.TrimSpace(email))
			hash := md5.Sum([]byte(email))
			return fmt.Sprintf("https://www.gravatar.com/avatar/%x?s=%d&d=mm", hash, size)
		},
		"statusClass": func(status string) string {
			switch status {
			case "published":
				return "gh-status-published"
			case "draft":
				return "gh-status-draft"
			case "scheduled":
				return "gh-status-scheduled"
			default:
				return ""
			}
		},
	}

	// Read the admin layout
	layoutBytes, err := viewsFS.ReadFile("views/obake/layouts/admin.html")
	if err != nil {
		return nil, fmt.Errorf("read admin layout: %w", err)
	}
	layoutContent := string(layoutBytes)

	// Read the auth layout
	authLayoutBytes, err := viewsFS.ReadFile("views/obake/layouts/auth.html")
	if err != nil {
		return nil, fmt.Errorf("read auth layout: %w", err)
	}
	authLayoutContent := string(authLayoutBytes)

	// Admin pages using the main layout
	adminPages := []string{
		"dashboard",
		"posts",
		"post-edit",
		"pages-list",
		"page-edit",
		"tags",
		"tag-edit",
		"members",
		"member-detail",
		"staff",
		"staff-edit",
		"settings-general",
		"settings-design",
		"settings-membership",
		"settings-email",
		"settings-advanced",
	}

	for _, name := range adminPages {
		pageBytes, err := viewsFS.ReadFile("views/obake/pages/" + name + ".html")
		if err != nil {
			continue
		}

		tmpl, err := template.New(name).Funcs(funcMap).Parse(layoutContent)
		if err != nil {
			return nil, fmt.Errorf("parse layout for %s: %w", name, err)
		}

		tmpl, err = tmpl.Parse(string(pageBytes))
		if err != nil {
			return nil, fmt.Errorf("parse page %s: %w", name, err)
		}

		templates[name] = tmpl
	}

	// Auth pages using the auth layout
	authPages := []string{"login"}
	for _, name := range authPages {
		pageBytes, err := viewsFS.ReadFile("views/obake/pages/" + name + ".html")
		if err != nil {
			continue
		}

		tmpl, err := template.New(name).Funcs(funcMap).Parse(authLayoutContent)
		if err != nil {
			return nil, fmt.Errorf("parse auth layout for %s: %w", name, err)
		}

		tmpl, err = tmpl.Parse(string(pageBytes))
		if err != nil {
			return nil, fmt.Errorf("parse auth page %s: %w", name, err)
		}

		templates[name] = tmpl
	}

	return templates, nil
}

// SiteTemplates parses and returns all frontend theme templates.
func SiteTemplates() (map[string]*template.Template, error) {
	templates := make(map[string]*template.Template)

	// Template functions for the frontend theme
	funcMap := template.FuncMap{
		"safe": func(s string) template.HTML {
			return template.HTML(s)
		},
		"truncate": func(s string, length int) string {
			if len(s) <= length {
				return s
			}
			// Try to break at a word boundary
			if length > 3 {
				for i := length; i > length-20 && i > 0; i-- {
					if s[i] == ' ' {
						return s[:i] + "..."
					}
				}
			}
			return s[:length] + "..."
		},
		"stripHtml": func(s string) string {
			// Simple HTML tag stripper
			result := strings.Builder{}
			inTag := false
			for _, r := range s {
				if r == '<' {
					inTag = true
				} else if r == '>' {
					inTag = false
				} else if !inTag {
					result.WriteRune(r)
				}
			}
			return result.String()
		},
		"substr": func(s string, start, end int) string {
			runes := []rune(s)
			if start >= len(runes) {
				return ""
			}
			if end > len(runes) {
				end = len(runes)
			}
			return string(runes[start:end])
		},
		"contains": func(s, substr string) bool {
			return strings.Contains(s, substr)
		},
		"urlEncode": func(s string) string {
			return template.URLQueryEscaper(s)
		},
		"now": func() time.Time {
			return time.Now()
		},
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
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
		"gt": func(a, b int) bool {
			return a > b
		},
		"lt": func(a, b int) bool {
			return a < b
		},
		"gte": func(a, b int) bool {
			return a >= b
		},
		"lte": func(a, b int) bool {
			return a <= b
		},
		"default": func(def, val interface{}) interface{} {
			if val == nil || val == "" {
				return def
			}
			return val
		},
		"lower": func(s string) string {
			return strings.ToLower(s)
		},
		"upper": func(s string) string {
			return strings.ToUpper(s)
		},
		"title": func(s string) string {
			return strings.Title(s)
		},
		"replace": func(s, old, new string) string {
			return strings.ReplaceAll(s, old, new)
		},
		"split": func(s, sep string) []string {
			return strings.Split(s, sep)
		},
		"join": func(sep string, arr []string) string {
			return strings.Join(arr, sep)
		},
		"asset": func(path string) string {
			return "/theme/assets/" + path
		},
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
		"safeURL": func(s string) template.URL {
			return template.URL(s)
		},
		"safeCSS": func(s string) template.CSS {
			return template.CSS(s)
		},
	}

	// Read the base layout
	layoutBytes, err := themeFS.ReadFile("theme/default/layouts/base.html")
	if err != nil {
		return nil, fmt.Errorf("read base layout: %w", err)
	}
	layoutContent := string(layoutBytes)

	// Common partials used by base layout (always included)
	commonPartials := []string{"header", "footer"}
	commonPartialContents := make(map[string]string)
	for _, name := range commonPartials {
		content, err := themeFS.ReadFile("theme/default/partials/" + name + ".html")
		if err != nil {
			continue
		}
		commonPartialContents[name] = string(content)
	}

	// Map of template name to the partials it needs
	templatePartials := map[string][]string{
		"index":    {"post-card", "pagination", "sidebar"},
		"post":     {"post-meta", "social-share", "author-box", "post-navigation", "related-posts", "comments", "comment-form"},
		"page":     {},
		"archive":  {"post-card", "pagination", "sidebar"},
		"category": {"post-card", "pagination", "sidebar"},
		"tag":      {"post-card", "pagination", "sidebar"},
		"author":   {"post-card", "pagination", "sidebar"},
		"search":   {"pagination"},
		"error":    {},
	}

	// Page templates
	pageTemplates := []string{
		"index", "post", "page", "archive",
		"category", "tag", "author", "search", "error",
	}

	for _, name := range pageTemplates {
		pageBytes, err := themeFS.ReadFile("theme/default/templates/" + name + ".html")
		if err != nil {
			continue
		}

		// Build combined template
		// Parse the layout first as the base, then the page content as "content" template
		tmpl := template.New("layout").Funcs(funcMap)

		// Parse the layout first
		tmpl, err = tmpl.Parse(layoutContent)
		if err != nil {
			return nil, fmt.Errorf("parse layout for %s: %w", name, err)
		}

		// Parse the page content - it contains {{define "content"}} which the layout uses
		tmpl, err = tmpl.Parse(string(pageBytes))
		if err != nil {
			return nil, fmt.Errorf("parse page %s: %w", name, err)
		}

		// Parse common partials
		for partialName, partialContent := range commonPartialContents {
			tmpl, err = tmpl.New("partials/" + partialName + ".html").Parse(partialContent)
			if err != nil {
				return nil, fmt.Errorf("parse common partial %s for %s: %w", partialName, name, err)
			}
		}

		// Parse only the partials this template needs
		neededPartials := templatePartials[name]
		for _, partialName := range neededPartials {
			content, err := themeFS.ReadFile("theme/default/partials/" + partialName + ".html")
			if err != nil {
				continue
			}
			tmpl, err = tmpl.New("partials/" + partialName + ".html").Parse(string(content))
			if err != nil {
				return nil, fmt.Errorf("parse partial %s for %s: %w", partialName, name, err)
			}
		}

		templates[name] = tmpl
	}

	return templates, nil
}
