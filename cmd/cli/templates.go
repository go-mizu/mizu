package cli

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
	"text/template"
	"time"
)

//go:embed all:templates
var templatesFS embed.FS

// templateMeta holds template metadata.
type templateMeta struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

// templateVars holds variables available to templates.
type templateVars struct {
	Name    string
	Module  string
	License string
	Year    int
	Vars    map[string]string
}

// templateFile represents a file to be rendered.
type templateFile struct {
	path    string // relative path in output (without .tmpl)
	content []byte // raw template content
}

// listTemplates returns all available templates.
func listTemplates() ([]templateMeta, error) {
	entries, err := fs.ReadDir(templatesFS, "templates")
	if err != nil {
		return nil, fmt.Errorf("read templates directory: %w", err)
	}

	var templates []templateMeta
	for _, entry := range entries {
		// Skip non-directories, underscore-prefixed, and the common directory
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), "_") || entry.Name() == "common" {
			continue
		}

		meta, err := loadTemplateMeta(entry.Name())
		if err != nil {
			// Skip templates without metadata
			continue
		}
		templates = append(templates, meta)
	}

	sort.Slice(templates, func(i, j int) bool {
		return templates[i].Name < templates[j].Name
	})

	return templates, nil
}

// loadTemplateMeta loads metadata for a template.
func loadTemplateMeta(name string) (templateMeta, error) {
	metaPath := path.Join("templates", name, "template.json")
	data, err := templatesFS.ReadFile(metaPath)
	if err != nil {
		// Return default metadata if not found
		return templateMeta{
			Name:        name,
			Description: name + " template",
		}, nil
	}

	var meta templateMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return templateMeta{}, fmt.Errorf("parse template.json: %w", err)
	}

	if meta.Name == "" {
		meta.Name = name
	}

	return meta, nil
}

// loadTemplateFiles loads all files for a template.
func loadTemplateFiles(name string) ([]templateFile, error) {
	var files []templateFile

	// Load common files first
	commonFiles, err := loadFilesFromDir("templates/_common")
	if err == nil {
		files = append(files, commonFiles...)
	}

	// Load template-specific files
	templateFiles, err := loadFilesFromDir(path.Join("templates", name))
	if err != nil {
		return nil, fmt.Errorf("load template %s: %w", name, err)
	}
	files = append(files, templateFiles...)

	return files, nil
}

// loadFilesFromDir recursively loads template files from a directory.
func loadFilesFromDir(dir string) ([]templateFile, error) {
	var files []templateFile

	err := fs.WalkDir(templatesFS, dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		// Skip metadata files
		name := d.Name()
		if name == "template.json" {
			return nil
		}

		// Read file content
		content, err := templatesFS.ReadFile(p)
		if err != nil {
			return fmt.Errorf("read %s: %w", p, err)
		}

		// Calculate relative path (removing template dir prefix)
		relPath := strings.TrimPrefix(p, dir+"/")

		// Remove .tmpl extension from output path
		outPath := strings.TrimSuffix(relPath, ".tmpl")

		// Handle special filename mappings
		outPath = mapOutputFilename(outPath)

		files = append(files, templateFile{
			path:    outPath,
			content: content,
		})

		return nil
	})

	return files, err
}

// renderTemplateFile renders a template file with the given variables.
func renderTemplateFile(tf templateFile, vars templateVars) ([]byte, error) {
	// If not a template file, return as-is
	if !bytes.Contains(tf.content, []byte("{{")) {
		return tf.content, nil
	}

	tmpl, err := template.New(tf.path).Parse(string(tf.content))
	if err != nil {
		return nil, fmt.Errorf("parse template %s: %w", tf.path, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return nil, fmt.Errorf("execute template %s: %w", tf.path, err)
	}

	return buf.Bytes(), nil
}

// newTemplateVars creates template variables with defaults.
func newTemplateVars(name, module, license string, customVars map[string]string) templateVars {
	if license == "" {
		license = "MIT"
	}

	vars := templateVars{
		Name:    name,
		Module:  module,
		License: license,
		Year:    time.Now().Year(),
		Vars:    customVars,
	}

	if vars.Vars == nil {
		vars.Vars = make(map[string]string)
	}

	return vars
}

// mapOutputFilename maps template filenames to their output equivalents.
// This handles files that can't have leading dots in templates (like .gitignore).
func mapOutputFilename(name string) string {
	base := path.Base(name)
	dir := path.Dir(name)

	// Special mappings for dotfiles
	mappings := map[string]string{
		"gitignore":  ".gitignore",
		"dockerignore": ".dockerignore",
		"env":        ".env",
		"env.example": ".env.example",
	}

	if mapped, ok := mappings[base]; ok {
		if dir == "." {
			return mapped
		}
		return path.Join(dir, mapped)
	}

	return name
}

// templateExists checks if a template exists.
func templateExists(name string) bool {
	_, err := templatesFS.ReadFile(path.Join("templates", name, "template.json"))
	if err == nil {
		return true
	}

	// Also check if directory exists even without metadata
	entries, err := fs.ReadDir(templatesFS, path.Join("templates", name))
	if err != nil {
		return false
	}
	return len(entries) > 0
}
