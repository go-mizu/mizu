package frontend

import (
	"encoding/json"
	"html/template"
	"io/fs"
	"strings"
)

// Manifest provides access to build artifacts and asset mappings.
type Manifest struct {
	entries  map[string]*ManifestEntry
	assets   map[string]string
	preloads map[string][]string
}

// ManifestEntry represents a single entry in the build manifest.
type ManifestEntry struct {
	File           string   `json:"file"`
	Name           string   `json:"name,omitempty"`
	Src            string   `json:"src,omitempty"`
	IsEntry        bool     `json:"isEntry,omitempty"`
	IsDynamicEntry bool     `json:"isDynamicEntry,omitempty"`
	Imports        []string `json:"imports,omitempty"`
	DynamicImports []string `json:"dynamicImports,omitempty"`
	CSS            []string `json:"css,omitempty"`
	Assets         []string `json:"assets,omitempty"`
}

// LoadManifest loads a build manifest from filesystem.
// Automatically detects Vite or Webpack manifest format.
func LoadManifest(fsys fs.FS, path string) (*Manifest, error) {
	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		return nil, err
	}

	// Try Vite manifest format first
	var viteManifest map[string]*ManifestEntry
	if err := json.Unmarshal(data, &viteManifest); err == nil {
		// Check if it looks like a Vite manifest
		for _, entry := range viteManifest {
			if entry.File != "" {
				return parseViteManifest(viteManifest), nil
			}
		}
	}

	// Try Webpack manifest format
	var webpackManifest map[string]string
	if err := json.Unmarshal(data, &webpackManifest); err == nil {
		return parseWebpackManifest(webpackManifest), nil
	}

	return &Manifest{
		entries:  make(map[string]*ManifestEntry),
		assets:   make(map[string]string),
		preloads: make(map[string][]string),
	}, nil
}

// parseViteManifest parses a Vite-style manifest.
func parseViteManifest(raw map[string]*ManifestEntry) *Manifest {
	m := &Manifest{
		entries:  raw,
		assets:   make(map[string]string),
		preloads: make(map[string][]string),
	}

	// Build asset map and preload chains
	for src, entry := range raw {
		m.assets[src] = entry.File

		if entry.IsEntry || entry.IsDynamicEntry {
			preloads := buildPreloadChain(raw, src, make(map[string]bool))
			m.preloads[src] = preloads
		}
	}

	return m
}

// buildPreloadChain recursively builds the list of modules to preload.
func buildPreloadChain(manifest map[string]*ManifestEntry, src string, visited map[string]bool) []string {
	if visited[src] {
		return nil
	}
	visited[src] = true

	entry, ok := manifest[src]
	if !ok {
		return nil
	}

	var preloads []string

	// Add imports
	for _, imp := range entry.Imports {
		if impEntry, ok := manifest[imp]; ok {
			preloads = append(preloads, impEntry.File)
			preloads = append(preloads, buildPreloadChain(manifest, imp, visited)...)
		}
	}

	return preloads
}

// parseWebpackManifest parses a Webpack-style manifest.
func parseWebpackManifest(raw map[string]string) *Manifest {
	m := &Manifest{
		entries:  make(map[string]*ManifestEntry),
		assets:   raw,
		preloads: make(map[string][]string),
	}

	// Convert to entries
	for src, file := range raw {
		m.entries[src] = &ManifestEntry{
			File: file,
			Src:  src,
		}
	}

	return m
}

// Entry returns the output file for an entry point.
func (m *Manifest) Entry(name string) string {
	if entry, ok := m.entries[name]; ok {
		return entry.File
	}
	return ""
}

// Asset returns the output path for an asset.
func (m *Manifest) Asset(path string) string {
	if output, ok := m.assets[path]; ok {
		return "/" + output
	}
	return "/" + path
}

// Preloads returns module preload paths for an entry.
func (m *Manifest) Preloads(entry string) []string {
	return m.preloads[entry]
}

// CSS returns CSS files associated with an entry.
func (m *Manifest) CSS(entry string) []string {
	if e, ok := m.entries[entry]; ok {
		return e.CSS
	}
	return nil
}

// ScriptTag generates a script tag for an entry point.
func (m *Manifest) ScriptTag(entry string) template.HTML {
	e, ok := m.entries[entry]
	if !ok {
		return ""
	}

	return template.HTML(`<script type="module" src="/` + e.File + `"></script>`) //nolint:gosec // G203: File path from trusted manifest
}

// PreloadTags generates modulepreload link tags for an entry.
func (m *Manifest) PreloadTags(entry string) template.HTML {
	preloads := m.Preloads(entry)
	if len(preloads) == 0 {
		return ""
	}

	var b strings.Builder
	for _, path := range preloads {
		b.WriteString(`<link rel="modulepreload" href="/`)
		b.WriteString(path)
		b.WriteString(`">`)
		b.WriteByte('\n')
	}

	return template.HTML(b.String()) //nolint:gosec // G203: Paths from trusted manifest
}

// CSSTags generates stylesheet link tags for an entry.
func (m *Manifest) CSSTags(entry string) template.HTML {
	css := m.CSS(entry)
	if len(css) == 0 {
		return ""
	}

	var b strings.Builder
	for _, path := range css {
		b.WriteString(`<link rel="stylesheet" href="/`)
		b.WriteString(path)
		b.WriteString(`">`)
		b.WriteByte('\n')
	}

	return template.HTML(b.String()) //nolint:gosec // G203: Paths from trusted manifest
}

// EntryTags generates all necessary tags for an entry point.
// This includes CSS, preloads, and the main script.
func (m *Manifest) EntryTags(entry string) template.HTML {
	e, ok := m.entries[entry]
	if !ok {
		return ""
	}

	var b strings.Builder

	// CSS first
	for _, css := range e.CSS {
		b.WriteString(`<link rel="stylesheet" href="/`)
		b.WriteString(css)
		b.WriteString(`">`)
		b.WriteByte('\n')
	}

	// Module preloads
	preloads := m.Preloads(entry)
	for _, path := range preloads {
		b.WriteString(`<link rel="modulepreload" href="/`)
		b.WriteString(path)
		b.WriteString(`">`)
		b.WriteByte('\n')
	}

	// Main script
	b.WriteString(`<script type="module" src="/`)
	b.WriteString(e.File)
	b.WriteString(`"></script>`)

	return template.HTML(b.String()) //nolint:gosec // G203: Paths from trusted manifest
}

// Entries returns all entry point names.
func (m *Manifest) Entries() []string {
	var entries []string
	for src, entry := range m.entries {
		if entry.IsEntry {
			entries = append(entries, src)
		}
	}
	return entries
}

// ViewHelpers returns template functions for view integration.
func (m *Manifest) ViewHelpers() template.FuncMap {
	return template.FuncMap{
		"vite_entry": func(name string) template.HTML {
			return m.EntryTags(name)
		},
		"vite_asset": func(path string) string {
			return m.Asset(path)
		},
		"vite_script": func(name string) template.HTML {
			return m.ScriptTag(name)
		},
		"vite_css": func(name string) template.HTML {
			return m.CSSTags(name)
		},
		"vite_preload": func(name string) template.HTML {
			return m.PreloadTags(name)
		},
	}
}
