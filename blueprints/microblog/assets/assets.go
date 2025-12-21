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

	// Add custom functions
	tmpl = tmpl.Funcs(template.FuncMap{
		// safeHTML marks a string as safe HTML
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},

		// upper converts a string to uppercase
		"upper": strings.ToUpper,

		// lower converts a string to lowercase
		"lower": strings.ToLower,

		// formatTime formats a time.Time to a human-readable relative string
		"formatTime": func(t time.Time) string {
			now := time.Now()
			diff := now.Sub(t)

			switch {
			case diff < time.Minute:
				return "just now"
			case diff < time.Hour:
				mins := int(diff.Minutes())
				if mins == 1 {
					return "1m"
				}
				return string(rune('0'+mins/10)) + string(rune('0'+mins%10)) + "m"
			case diff < 24*time.Hour:
				hours := int(diff.Hours())
				if hours == 1 {
					return "1h"
				}
				return string(rune('0'+hours/10)) + string(rune('0'+hours%10)) + "h"
			case diff < 7*24*time.Hour:
				days := int(diff.Hours() / 24)
				if days == 1 {
					return "1d"
				}
				return string(rune('0'+days)) + "d"
			default:
				if t.Year() == now.Year() {
					return t.Format("Jan 2")
				}
				return t.Format("Jan 2, 2006")
			}
		},

		// formatNumber formats a number with K/M suffixes
		"formatNumber": func(n int) string {
			switch {
			case n >= 1000000:
				return string(rune('0'+n/1000000)) + "." + string(rune('0'+(n%1000000)/100000)) + "M"
			case n >= 1000:
				return string(rune('0'+n/1000)) + "." + string(rune('0'+(n%1000)/100)) + "K"
			default:
				return itoa(n)
			}
		},

		// percentage calculates percentage
		"percentage": func(part, total int) int {
			if total == 0 {
				return 0
			}
			return (part * 100) / total
		},

		// dict creates a map from key-value pairs
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

		// slice returns a substring
		"slice": func(s string, start, end int) string {
			if start < 0 {
				start = 0
			}
			if end > len(s) {
				end = len(s)
			}
			if start > end {
				return ""
			}
			return s[start:end]
		},

		// eq compares two values for equality
		"eq": func(a, b any) bool {
			return a == b
		},

		// ne compares two values for inequality
		"ne": func(a, b any) bool {
			return a != b
		},

		// add adds two integers
		"add": func(a, b int) int {
			return a + b
		},

		// sub subtracts two integers
		"sub": func(a, b int) int {
			return a - b
		},

		// mul multiplies two integers
		"mul": func(a, b int) int {
			return a * b
		},

		// div divides two integers
		"div": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a / b
		},

		// default returns the default value if the value is empty
		"default": func(defaultVal, val any) any {
			if val == nil || val == "" || val == 0 || val == false {
				return defaultVal
			}
			return val
		},

		// join joins strings with a separator
		"join": func(sep string, items []string) string {
			return strings.Join(items, sep)
		},

		// contains checks if a string contains a substring
		"contains": func(s, substr string) bool {
			return strings.Contains(s, substr)
		},

		// hasPrefix checks if a string starts with a prefix
		"hasPrefix": func(s, prefix string) bool {
			return strings.HasPrefix(s, prefix)
		},

		// hasSuffix checks if a string ends with a suffix
		"hasSuffix": func(s, suffix string) bool {
			return strings.HasSuffix(s, suffix)
		},

		// truncate truncates a string to a maximum length
		"truncate": func(s string, maxLen int) string {
			if len(s) <= maxLen {
				return s
			}
			return s[:maxLen-3] + "..."
		},

		// printf formats a string
		"printf": func(format string, args ...any) string {
			return sprintf(format, args...)
		},
	})

	return tmpl.ParseFS(Views(), "layouts/*.html", "pages/*.html", "components/*.html")
}

// itoa converts an integer to a string without using strconv
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + itoa(-n)
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

// sprintf is a simple printf implementation for common cases
func sprintf(format string, args ...any) string {
	if len(args) == 0 {
		return format
	}

	var result strings.Builder
	argIdx := 0

	for i := 0; i < len(format); i++ {
		if format[i] == '%' && i+1 < len(format) {
			switch format[i+1] {
			case 's':
				if argIdx < len(args) {
					if s, ok := args[argIdx].(string); ok {
						result.WriteString(s)
					}
					argIdx++
				}
				i++
				continue
			case 'd':
				if argIdx < len(args) {
					if n, ok := args[argIdx].(int); ok {
						result.WriteString(itoa(n))
					}
					argIdx++
				}
				i++
				continue
			case '%':
				result.WriteByte('%')
				i++
				continue
			}
		}
		result.WriteByte(format[i])
	}

	return result.String()
}
