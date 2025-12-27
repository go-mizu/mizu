// Package assets provides embedded static files and templates.
package assets

import (
	"crypto/md5"
	"embed"
	"encoding/hex"
	"html/template"
	"io/fs"
)

//go:embed static/*
var staticFS embed.FS

//go:embed views/*
var viewsFS embed.FS

// Available themes
var Themes = []string{"default", "aim1.0", "ymxp", "msn", "im26", "imos9", "imosx", "team11", "jarvis"}

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

	// Read the layout for the theme
	layoutBytes, err := viewsFS.ReadFile("views/" + theme + "/layouts/default.html")
	if err != nil {
		// Fall back to default theme
		layoutBytes, err = viewsFS.ReadFile("views/default/layouts/default.html")
		if err != nil {
			return nil, err
		}
	}
	layoutContent := string(layoutBytes)

	// Read all pages
	pages := []string{"home", "login", "register", "app", "settings"}
	for _, name := range pages {
		// Try theme-specific page first
		pageBytes, err := viewsFS.ReadFile("views/" + theme + "/pages/" + name + ".html")
		if err != nil {
			// Fall back to default theme
			pageBytes, err = viewsFS.ReadFile("views/default/pages/" + name + ".html")
			if err != nil {
				continue
			}
		}

		tmpl, err := template.New(name).Parse(layoutContent)
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
// Returns a map of theme -> page -> template.
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

// AssetHashes contains content-based hashes for cache busting.
type AssetHashes struct {
	AppJS string            // Hash for app.js
	CSS   map[string]string // Theme -> CSS hash
}

// ComputeAssetHashes calculates MD5 hashes of static assets for cache busting.
func ComputeAssetHashes() *AssetHashes {
	hashes := &AssetHashes{
		CSS: make(map[string]string),
	}

	// Hash app.js
	if data, err := staticFS.ReadFile("static/js/app.js"); err == nil {
		hashes.AppJS = hashContent(data)
	}

	// Hash CSS files for each theme
	cssFiles := map[string]string{
		"default": "static/css/default.css",
		"aim1.0":  "static/css/aim.css",
		"ymxp":    "static/css/ymxp.css",
		"msn":     "static/css/msn.css",
		"im26":    "static/css/imessage.css",
		"imos9":   "static/css/imos9.css",
		"imosx":   "static/css/imosx.css",
		"team11":  "static/css/team11.css",
		"jarvis":  "static/css/jarvis.css",
	}

	for theme, path := range cssFiles {
		if data, err := staticFS.ReadFile(path); err == nil {
			hashes.CSS[theme] = hashContent(data)
		}
	}

	return hashes
}

func hashContent(data []byte) string {
	h := md5.Sum(data)
	return hex.EncodeToString(h[:8]) // Use first 8 bytes (16 hex chars)
}
