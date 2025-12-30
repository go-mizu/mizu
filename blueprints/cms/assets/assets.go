// Package assets provides embedded static files and templates for the CMS.
package assets

import (
	"crypto/md5"
	"embed"
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
