package assets

import (
	"embed"
	"encoding/json"
	"html/template"
	"io/fs"
	"sync"
)

//go:embed views static
var content embed.FS

// ViteManifestEntry represents a single entry in Vite's manifest.json
type ViteManifestEntry struct {
	File    string   `json:"file"`
	Src     string   `json:"src,omitempty"`
	IsEntry bool     `json:"isEntry,omitempty"`
	CSS     []string `json:"css,omitempty"`
	Imports []string `json:"imports,omitempty"`
}

var (
	manifest     map[string]ViteManifestEntry
	manifestOnce sync.Once
)

// loadManifest reads and parses the Vite manifest.json
func loadManifest() map[string]ViteManifestEntry {
	manifestOnce.Do(func() {
		data, err := content.ReadFile("static/dist/.vite/manifest.json")
		if err != nil {
			// Fallback for dev mode or missing manifest
			manifest = make(map[string]ViteManifestEntry)
			return
		}
		if err := json.Unmarshal(data, &manifest); err != nil {
			manifest = make(map[string]ViteManifestEntry)
		}
	})
	return manifest
}

// Asset returns the versioned path for a given source file.
// Usage in templates: {{ asset "src/main.tsx" }}
func Asset(src string) string {
	m := loadManifest()
	if entry, ok := m[src]; ok {
		return "/static/dist/" + entry.File
	}
	// Fallback: return original path for dev mode
	return "/static/dist/" + src
}

// AssetCSS returns the CSS files associated with an entry point.
// Usage in templates: {{ range assetCSS "src/main.tsx" }}<link rel="stylesheet" href="{{ . }}">{{ end }}
func AssetCSS(src string) []string {
	m := loadManifest()
	if entry, ok := m[src]; ok {
		paths := make([]string, len(entry.CSS))
		for i, css := range entry.CSS {
			paths[i] = "/static/dist/" + css
		}
		return paths
	}
	return nil
}

// Templates parses and returns all templates.
func Templates() (*template.Template, error) {
	// Load manifest on startup
	loadManifest()

	// Define template functions
	funcs := template.FuncMap{
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
		"safeJS": func(s string) template.JS {
			return template.JS(s)
		},
		"safeCSS": func(s string) template.CSS {
			return template.CSS(s)
		},
		"asset":    Asset,
		"assetCSS": AssetCSS,
	}

	// Parse all templates
	tmpl := template.New("").Funcs(funcs)

	// Parse layouts
	layouts, err := fs.Glob(content, "views/layouts/*.html")
	if err != nil {
		return nil, err
	}
	for _, layout := range layouts {
		data, err := content.ReadFile(layout)
		if err != nil {
			return nil, err
		}
		_, err = tmpl.New(layout).Parse(string(data))
		if err != nil {
			return nil, err
		}
	}

	// Parse pages
	pages, err := fs.Glob(content, "views/pages/*.html")
	if err != nil {
		return nil, err
	}
	for _, page := range pages {
		data, err := content.ReadFile(page)
		if err != nil {
			return nil, err
		}
		// Use just the filename as the template name
		name := page[len("views/pages/"):]
		_, err = tmpl.New(name).Parse(string(data))
		if err != nil {
			return nil, err
		}
	}

	return tmpl, nil
}

// Static returns the static file system.
func Static() fs.FS {
	static, _ := fs.Sub(content, "static")
	return static
}
