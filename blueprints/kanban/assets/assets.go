// Package assets provides embedded static files and templates.
package assets

import (
	"embed"
	"html/template"
	"io/fs"
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

// Views returns the views filesystem.
func Views() fs.FS {
	sub, _ := fs.Sub(FS, "views")
	return sub
}

// Templates parses all view templates with custom functions.
func Templates() (*template.Template, error) {
	tmpl := template.New("")

	tmpl = tmpl.Funcs(template.FuncMap{
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"formatTime": func(t time.Time) string {
			now := time.Now()
			diff := now.Sub(t)
			switch {
			case diff < time.Minute:
				return "just now"
			case diff < time.Hour:
				mins := int(diff.Minutes())
				return itoa(mins) + "m"
			case diff < 24*time.Hour:
				hours := int(diff.Hours())
				return itoa(hours) + "h"
			case diff < 7*24*time.Hour:
				days := int(diff.Hours() / 24)
				return itoa(days) + "d"
			default:
				if t.Year() == now.Year() {
					return t.Format("Jan 2")
				}
				return t.Format("Jan 2, 2006")
			}
		},
		"formatDate": func(t *time.Time) string {
			if t == nil {
				return ""
			}
			return t.Format("Jan 2, 2006")
		},
		"eq":  func(a, b any) bool { return a == b },
		"ne":  func(a, b any) bool { return a != b },
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"default": func(defaultVal, val any) any {
			if val == nil || val == "" || val == 0 {
				return defaultVal
			}
			return val
		},
		"dict": func(values ...any) map[string]any {
			if len(values)%2 != 0 {
				return nil
			}
			dict := make(map[string]any, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					continue
				}
				dict[key] = values[i+1]
			}
			return dict
		},
		"truncate": func(s string, maxLen int) string {
			if len(s) <= maxLen {
				return s
			}
			return s[:maxLen-3] + "..."
		},
		"contains": func(s, substr string) bool {
			return strings.Contains(s, substr)
		},
		"statusLabel": func(status string) string {
			labels := map[string]string{
				"backlog":     "Backlog",
				"todo":        "Todo",
				"in_progress": "In Progress",
				"in_review":   "In Review",
				"done":        "Done",
				"cancelled":   "Cancelled",
			}
			if l, ok := labels[status]; ok {
				return l
			}
			return status
		},
		"priorityLabel": func(priority string) string {
			labels := map[string]string{
				"urgent": "Urgent",
				"high":   "High",
				"medium": "Medium",
				"low":    "Low",
				"none":   "No priority",
			}
			if l, ok := labels[priority]; ok {
				return l
			}
			return priority
		},
		"typeLabel": func(t string) string {
			labels := map[string]string{
				"epic":    "Epic",
				"story":   "Story",
				"task":    "Task",
				"bug":     "Bug",
				"subtask": "Subtask",
			}
			if l, ok := labels[t]; ok {
				return l
			}
			return t
		},
		"len": func(v any) int {
			switch val := v.(type) {
			case []any:
				return len(val)
			case string:
				return len(val)
			default:
				return 0
			}
		},
	})

	return tmpl.ParseFS(Views(), "layouts/*.html", "pages/*.html", "components/*.html")
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
