// Package assets provides embedded static files and templates.
package assets

import (
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

// Templates parses and returns all templates.
func Templates() (map[string]*template.Template, error) {
	templates := make(map[string]*template.Template)

	// Template functions
	funcMap := template.FuncMap{
		"lower": strings.ToLower,
		"upper": strings.ToUpper,
		"title": strings.Title,
		"slice": func(s string, start, end int) string {
			if start >= len(s) {
				return ""
			}
			if end > len(s) {
				end = len(s)
			}
			return s[start:end]
		},
		"formatSize": formatFileSize,
		"formatTime": formatTimeAgo,
		"formatDate": formatDate,
		"fileIcon":   getFileIcon,
		"mimeIcon":   getMimeTypeIcon,
		"isImage":    isImageMime,
		"isVideo":    isVideoMime,
		"isAudio":    isAudioMime,
		"isDocument": isDocumentMime,
		"truncate": func(s string, n int) string {
			if len(s) <= n {
				return s
			}
			return s[:n-3] + "..."
		},
		"join":     strings.Join,
		"contains": strings.Contains,
		"add":      func(a, b int) int { return a + b },
		"sub":      func(a, b int) int { return a - b },
		"mul":      func(a, b int) int { return a * b },
		"div": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a / b
		},
		"percentage": func(used, total int64) int {
			if total == 0 {
				return 0
			}
			return int(used * 100 / total)
		},
		"eq": func(a, b interface{}) bool {
			return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
		},
		"ne": func(a, b interface{}) bool {
			return fmt.Sprintf("%v", a) != fmt.Sprintf("%v", b)
		},
		"default": func(def, val interface{}) interface{} {
			if val == nil || val == "" {
				return def
			}
			return val
		},
	}

	// Read the main layout
	layoutBytes, err := viewsFS.ReadFile("views/default/layouts/default.html")
	if err != nil {
		return nil, fmt.Errorf("read default layout: %w", err)
	}
	layoutContent := string(layoutBytes)

	// Read auth layout
	authLayoutBytes, err := viewsFS.ReadFile("views/default/layouts/auth.html")
	if err != nil {
		return nil, fmt.Errorf("read auth layout: %w", err)
	}
	authLayoutContent := string(authLayoutBytes)

	// Pages using the main layout
	mainPages := []string{
		"files", "shared", "recent", "starred",
		"trash", "search", "settings", "activity", "preview",
	}
	for _, name := range mainPages {
		pageBytes, err := viewsFS.ReadFile("views/default/pages/" + name + ".html")
		if err != nil {
			continue // Skip missing pages
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

	// Pages using the auth layout
	authPages := []string{"login", "register"}
	for _, name := range authPages {
		pageBytes, err := viewsFS.ReadFile("views/default/pages/" + name + ".html")
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

	// Shared link page (no auth, minimal layout)
	sharePageBytes, err := viewsFS.ReadFile("views/default/pages/share.html")
	if err == nil {
		tmpl, err := template.New("share").Funcs(funcMap).Parse(authLayoutContent)
		if err == nil {
			tmpl, err = tmpl.Parse(string(sharePageBytes))
			if err == nil {
				templates["share"] = tmpl
			}
		}
	}

	return templates, nil
}

// formatFileSize formats bytes into human-readable format.
func formatFileSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case size >= TB:
		return fmt.Sprintf("%.1f TB", float64(size)/TB)
	case size >= GB:
		return fmt.Sprintf("%.1f GB", float64(size)/GB)
	case size >= MB:
		return fmt.Sprintf("%.1f MB", float64(size)/MB)
	case size >= KB:
		return fmt.Sprintf("%.1f KB", float64(size)/KB)
	default:
		return fmt.Sprintf("%d B", size)
	}
}

// formatTimeAgo formats a time as relative (e.g., "2 hours ago").
func formatTimeAgo(t time.Time) string {
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
		return fmt.Sprintf("%d minutes ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "yesterday"
		}
		return fmt.Sprintf("%d days ago", days)
	case diff < 30*24*time.Hour:
		weeks := int(diff.Hours() / 24 / 7)
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	case diff < 365*24*time.Hour:
		months := int(diff.Hours() / 24 / 30)
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	default:
		return t.Format("Jan 2, 2006")
	}
}

// formatDate formats a time as a date string.
func formatDate(t time.Time) string {
	return t.Format("Jan 2, 2006")
}

// getFileIcon returns the icon name for a file extension.
func getFileIcon(filename string) string {
	ext := strings.ToLower(strings.TrimPrefix(filename, "."))
	if idx := strings.LastIndex(filename, "."); idx != -1 {
		ext = strings.ToLower(filename[idx+1:])
	}

	switch ext {
	case "jpg", "jpeg", "png", "gif", "webp", "svg", "bmp", "ico":
		return "image"
	case "mp4", "webm", "mov", "avi", "mkv", "m4v":
		return "video"
	case "mp3", "wav", "flac", "ogg", "m4a", "aac":
		return "audio"
	case "pdf":
		return "file-text"
	case "doc", "docx", "odt", "rtf":
		return "file-text"
	case "xls", "xlsx", "ods", "csv":
		return "file-spreadsheet"
	case "ppt", "pptx", "odp":
		return "file-presentation"
	case "zip", "rar", "7z", "tar", "gz", "bz2":
		return "file-archive"
	case "js", "ts", "jsx", "tsx", "go", "py", "rb", "java", "c", "cpp", "h", "rs", "swift":
		return "file-code"
	case "html", "css", "scss", "sass", "less":
		return "file-code"
	case "json", "xml", "yaml", "yml", "toml":
		return "file-json"
	case "md", "txt", "log":
		return "file-text"
	case "sql", "db", "sqlite":
		return "database"
	default:
		return "file"
	}
}

// getMimeTypeIcon returns the icon name for a MIME type.
func getMimeTypeIcon(mimeType string) string {
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return "image"
	case strings.HasPrefix(mimeType, "video/"):
		return "video"
	case strings.HasPrefix(mimeType, "audio/"):
		return "audio"
	case strings.HasPrefix(mimeType, "text/"):
		return "file-text"
	case mimeType == "application/pdf":
		return "file-text"
	case strings.Contains(mimeType, "zip") || strings.Contains(mimeType, "archive") || strings.Contains(mimeType, "compressed"):
		return "file-archive"
	case strings.Contains(mimeType, "json"):
		return "file-json"
	case strings.Contains(mimeType, "javascript") || strings.Contains(mimeType, "typescript"):
		return "file-code"
	default:
		return "file"
	}
}

// isImageMime checks if the MIME type is an image.
func isImageMime(mimeType string) bool {
	return strings.HasPrefix(mimeType, "image/")
}

// isVideoMime checks if the MIME type is a video.
func isVideoMime(mimeType string) bool {
	return strings.HasPrefix(mimeType, "video/")
}

// isAudioMime checks if the MIME type is audio.
func isAudioMime(mimeType string) bool {
	return strings.HasPrefix(mimeType, "audio/")
}

// isDocumentMime checks if the MIME type is a document.
func isDocumentMime(mimeType string) bool {
	docTypes := []string{
		"application/pdf",
		"application/msword",
		"application/vnd.openxmlformats-officedocument",
		"text/plain",
		"text/markdown",
	}
	for _, t := range docTypes {
		if strings.HasPrefix(mimeType, t) {
			return true
		}
	}
	return false
}
