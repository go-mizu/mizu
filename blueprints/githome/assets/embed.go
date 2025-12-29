// Package assets provides embedded static files and templates.
package assets

import (
	"crypto/md5"
	"embed"
	"encoding/hex"
	"html/template"
	"io/fs"
	"strconv"
	"strings"
	"sync"
)

//go:embed static/*
var staticFS embed.FS

//go:embed views/*
var viewsFS embed.FS

// Available themes
var Themes = []string{"default"}

// Asset hashes for cache busting
var (
	assetHashes     = make(map[string]string)
	assetHashesMu   sync.RWMutex
	assetHashesOnce sync.Once
)

// computeAssetHashes calculates MD5 hashes for static assets
func computeAssetHashes() {
	assetHashesMu.Lock()
	defer assetHashesMu.Unlock()

	// Compute hashes for known assets
	assets := []string{
		"css/main.css",
		"js/app.js",
	}

	for _, asset := range assets {
		data, err := staticFS.ReadFile("static/" + asset)
		if err != nil {
			continue
		}
		hash := md5.Sum(data)
		assetHashes[asset] = hex.EncodeToString(hash[:])[:8] // Use first 8 chars
	}
}

// GetAssetHash returns the hash for a given asset path
func GetAssetHash(path string) string {
	assetHashesOnce.Do(computeAssetHashes)
	assetHashesMu.RLock()
	defer assetHashesMu.RUnlock()
	return assetHashes[path]
}

// AssetURL returns the asset URL with cache busting hash
func AssetURL(path string) string {
	hash := GetAssetHash(path)
	if hash != "" {
		return "/_assets/" + path + "?v=" + hash
	}
	return "/_assets/" + path
}

// Static returns the static files filesystem.
func Static() fs.FS {
	sub, _ := fs.Sub(staticFS, "static")
	return sub
}

// Templates parses and returns all templates for the default theme.
func Templates() (map[string]*template.Template, error) {
	return TemplatesForTheme("default")
}

// TemplatesForTheme parses and returns all templates for a specific theme.
func TemplatesForTheme(theme string) (map[string]*template.Template, error) {
	templates := make(map[string]*template.Template)

	// Template functions
	funcMap := template.FuncMap{
		"lower": func(s string) string {
			return strings.ToLower(s)
		},
		"upper": func(s string) string {
			return strings.ToUpper(s)
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
		"firstChar": func(s string) string {
			if len(s) == 0 {
				return ""
			}
			return strings.ToUpper(string(s[0]))
		},
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"subtract": func(a, b int) int {
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
		"default": func(def, val interface{}) interface{} {
			if val == nil || val == "" {
				return def
			}
			return val
		},
		"truncate": func(s string, n int) string {
			if len(s) <= n {
				return s
			}
			return s[:n] + "..."
		},
		"split": func(s, sep string) []string {
			return strings.Split(s, sep)
		},
		"formatNumber": func(n int) string {
			// Format numbers with thousand separators (e.g., 1,234,567)
			str := strconv.Itoa(n)
			if n < 1000 {
				return str
			}
			// Insert commas from right to left
			var result []byte
			for i, c := range str {
				if i > 0 && (len(str)-i)%3 == 0 {
					result = append(result, ',')
				}
				result = append(result, byte(c))
			}
			return string(result)
		},
		"contrastColor": func(hexColor string) string {
			// Calculate the best contrasting text color (white or black) for a given background
			// Remove # prefix if present
			hexColor = strings.TrimPrefix(hexColor, "#")
			if len(hexColor) < 6 {
				return "#ffffff"
			}

			r, _ := strconv.ParseInt(hexColor[0:2], 16, 64)
			g, _ := strconv.ParseInt(hexColor[2:4], 16, 64)
			b, _ := strconv.ParseInt(hexColor[4:6], 16, 64)

			// Calculate relative luminance using sRGB
			// Using a simplified formula: (R*299 + G*587 + B*114) / 1000
			luminance := (float64(r)*299 + float64(g)*587 + float64(b)*114) / 1000

			// Return white for dark backgrounds, black for light backgrounds
			if luminance > 128 {
				return "#24292f" // Dark text
			}
			return "#ffffff" // White text
		},
		"assetURL": AssetURL,
	}

	// Read the main layout for the theme
	layoutBytes, err := viewsFS.ReadFile("views/" + theme + "/layouts/default.html")
	if err != nil {
		// Fall back to default theme
		layoutBytes, err = viewsFS.ReadFile("views/default/layouts/default.html")
		if err != nil {
			return nil, err
		}
	}
	layoutContent := string(layoutBytes)

	// Read auth layout
	authLayoutBytes, err := viewsFS.ReadFile("views/" + theme + "/layouts/auth.html")
	if err != nil {
		authLayoutBytes, err = viewsFS.ReadFile("views/default/layouts/auth.html")
		if err != nil {
			// Use default layout for auth if no auth layout exists
			authLayoutBytes = layoutBytes
		}
	}
	authLayoutContent := string(authLayoutBytes)

	// Pages using the main layout
	mainPages := []string{
		"home", "explore", "new_repo", "notifications",
		"user_profile",
		"repo_home", "repo_code", "repo_blob", "repo_blame", "repo_issues", "issue_view", "new_issue", "repo_settings",
		"repo_commits", "commit_detail",
	}
	for _, name := range mainPages {
		pageBytes, err := viewsFS.ReadFile("views/" + theme + "/pages/" + name + ".html")
		if err != nil {
			pageBytes, err = viewsFS.ReadFile("views/default/pages/" + name + ".html")
			if err != nil {
				continue
			}
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

	// Pages using the auth layout
	authPages := []string{"login", "register"}
	for _, name := range authPages {
		pageBytes, err := viewsFS.ReadFile("views/" + theme + "/pages/" + name + ".html")
		if err != nil {
			pageBytes, err = viewsFS.ReadFile("views/default/pages/" + name + ".html")
			if err != nil {
				continue
			}
		}

		tmpl, err := template.New(name).Funcs(funcMap).Parse(authLayoutContent)
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

// AllTemplates parses and returns templates for all themes.
func AllTemplates() (map[string]map[string]*template.Template, error) {
	allTemplates := make(map[string]map[string]*template.Template)

	for _, theme := range Themes {
		templates, err := TemplatesForTheme(theme)
		if err != nil {
			return nil, err
		}
		allTemplates[theme] = templates
	}

	return allTemplates, nil
}
