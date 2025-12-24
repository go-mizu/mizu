package assets

import (
	"embed"
	"html/template"
	"path/filepath"
	"strings"
	"time"
)

//go:embed static/* views/*.html
var embedded embed.FS

// GetStatic returns a static file's content and content type.
func GetStatic(path string) ([]byte, string, error) {
	data, err := embedded.ReadFile("static/" + path)
	if err != nil {
		return nil, "", err
	}

	contentType := "application/octet-stream"
	ext := filepath.Ext(path)
	switch ext {
	case ".css":
		contentType = "text/css; charset=utf-8"
	case ".js":
		contentType = "application/javascript; charset=utf-8"
	case ".html":
		contentType = "text/html; charset=utf-8"
	case ".svg":
		contentType = "image/svg+xml"
	case ".png":
		contentType = "image/png"
	case ".ico":
		contentType = "image/x-icon"
	}

	return data, contentType, nil
}

// LoadTemplates loads and parses all HTML templates.
func LoadTemplates() (*template.Template, error) {
	funcMap := template.FuncMap{
		"timeago": timeAgo,
		"add":     func(a, b int) int { return a + b },
		"mul":     func(a, b int) int { return a * b },
		"safe":    func(s string) template.HTML { return template.HTML(s) },
		"dict":    dict,
	}

	tmpl := template.New("").Funcs(funcMap)

	// Read all template files
	entries, err := embedded.ReadDir("views")
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".html") {
			continue
		}

		data, err := embedded.ReadFile("views/" + entry.Name())
		if err != nil {
			return nil, err
		}

		_, err = tmpl.New(entry.Name()).Parse(string(data))
		if err != nil {
			return nil, err
		}
	}

	return tmpl, nil
}

// timeAgo formats a time as a human-readable relative time.
func timeAgo(t time.Time) string {
	diff := time.Since(t)

	if diff < time.Minute {
		return "just now"
	}
	if diff < time.Hour {
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return strings.Replace("X minutes ago", "X", itoa(mins), 1)
	}
	if diff < 24*time.Hour {
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return strings.Replace("X hours ago", "X", itoa(hours), 1)
	}
	if diff < 30*24*time.Hour {
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return strings.Replace("X days ago", "X", itoa(days), 1)
	}
	if diff < 365*24*time.Hour {
		months := int(diff.Hours() / 24 / 30)
		if months == 1 {
			return "1 month ago"
		}
		return strings.Replace("X months ago", "X", itoa(months), 1)
	}

	years := int(diff.Hours() / 24 / 365)
	if years == 1 {
		return "1 year ago"
	}
	return strings.Replace("X years ago", "X", itoa(years), 1)
}

func itoa(i int) string {
	return strings.TrimSpace(strings.Replace("   ", " ", string(rune('0'+i%10)), 1))
}

// dict creates a map from key-value pairs for use in templates.
func dict(values ...any) map[string]any {
	if len(values)%2 != 0 {
		return nil
	}
	m := make(map[string]any, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			continue
		}
		m[key] = values[i+1]
	}
	return m
}
