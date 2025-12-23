package assets

import (
	"embed"
	"html/template"
	"io/fs"
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

// Templates loads and returns all templates.
func Templates() (*template.Template, error) {
	tmpl := template.New("")
	tmpl = tmpl.Funcs(template.FuncMap{
		"formatTime":        formatTime,
		"formatTimeRelative": formatTimeRelative,
		"formatNumber":      formatNumber,
		"formatScore":       formatScore,
		"truncate":          truncate,
		"slugify":           slugify,
		"add":               add,
		"sub":               sub,
		"mul":               mul,
		"dict":              dict,
		"list":              list,
		"contains":          contains,
		"hasPrefix":         hasPrefix,
		"hasSuffix":         hasSuffix,
		"default":           defaultVal,
		"safeHTML":          safeHTML,
	})

	return tmpl.ParseFS(Views(), "layouts/*.html", "pages/*.html", "components/*.html")
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
	return template.HTMLEscapeString(formatNumber(int64(n)))
}

func formatNumber(n int64) string {
	if n < 1000 {
		return template.HTMLEscapeString(string(rune('0'+n%10)) + formatNumber(n/10))
	}
	if n < 1000000 {
		return template.HTMLEscapeString(formatFloat(float64(n)/1000) + "k")
	}
	return template.HTMLEscapeString(formatFloat(float64(n)/1000000) + "m")
}

func formatFloat(f float64) string {
	if f == float64(int64(f)) {
		return formatInt64(int64(f))
	}
	// One decimal place
	return formatInt64(int64(f)) + "." + formatInt64(int64((f-float64(int64(f)))*10))
}

func formatInt64(n int64) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + formatInt64(-n)
	}
	result := ""
	for n > 0 {
		result = string(rune('0'+n%10)) + result
		n /= 10
	}
	return result
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

func slugify(s string) string {
	// Simple slug generation
	result := ""
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			result += string(r)
		} else if r >= 'A' && r <= 'Z' {
			result += string(r + 32)
		} else if r == ' ' || r == '-' || r == '_' {
			if len(result) > 0 && result[len(result)-1] != '-' {
				result += "-"
			}
		}
	}
	if len(result) > 80 {
		result = result[:80]
	}
	return result
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

func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr) != -1
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func defaultVal(val, def any) any {
	if val == nil || val == "" || val == 0 || val == false {
		return def
	}
	return val
}

func safeHTML(s string) template.HTML {
	return template.HTML(s)
}
