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
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Tags        []string      `json:"tags"`
	SubTemplates []subTemplate `json:"subTemplates,omitempty"`
}

// subTemplate describes a sub-template option within a parent template.
type subTemplate struct {
	Name        string `json:"name"`
	Description string `json:"description"`
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

// listTemplates returns all available templates, including nested templates.
func listTemplates() ([]templateMeta, error) {
	var templates []templateMeta
	if err := listTemplatesRecursive("templates", "", &templates); err != nil {
		return nil, err
	}

	sort.Slice(templates, func(i, j int) bool {
		return templates[i].Name < templates[j].Name
	})

	return templates, nil
}

// listTemplatesRecursive recursively finds all templates in a directory.
// It identifies templates by the presence of template.json files.
// When a template has subTemplates defined, its sub-directories are not listed separately.
func listTemplatesRecursive(basePath, prefix string, templates *[]templateMeta) error {
	entries, err := fs.ReadDir(templatesFS, basePath)
	if err != nil {
		return fmt.Errorf("read directory %s: %w", basePath, err)
	}

	for _, entry := range entries {
		// Skip non-directories and underscore-prefixed directories
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), "_") || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		dirName := entry.Name()
		templateName := dirName
		if prefix != "" {
			templateName = prefix + "/" + dirName
		}
		dirPath := path.Join(basePath, dirName)

		// Check if this directory has a template.json (it's a template)
		metaPath := path.Join(dirPath, "template.json")
		if _, err := templatesFS.ReadFile(metaPath); err == nil {
			meta, err := loadTemplateMeta(templateName)
			if err == nil {
				*templates = append(*templates, meta)
				// If this template has subTemplates, don't list them separately
				if len(meta.SubTemplates) > 0 {
					continue
				}
			}
		}

		// Recursively check subdirectories for nested templates
		if err := listTemplatesRecursive(dirPath, templateName, templates); err != nil {
			// Ignore errors from subdirectories
			continue
		}
	}

	return nil
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
// For nested templates (e.g., "frontend/react"), it loads files from:
// 1. templates/_common/ (root common files)
// 2. templates/frontend/_common/ (category common files)
// 3. templates/frontend/react/ (template-specific files)
// Each level overrides files from the previous level.
func loadTemplateFiles(name string) ([]templateFile, error) {
	fileMap := make(map[string]templateFile)

	// Load root common files first
	commonFiles, err := loadFilesFromDir("templates/_common")
	if err == nil {
		for _, f := range commonFiles {
			fileMap[f.path] = f
		}
	}

	// For nested templates, load each level's _common directory
	// e.g., for "frontend/react":
	//   - templates/frontend/_common
	parts := strings.Split(name, "/")
	for i := 1; i < len(parts); i++ {
		commonPath := path.Join("templates", path.Join(parts[:i]...), "_common")
		files, err := loadFilesFromDir(commonPath)
		if err == nil {
			for _, f := range files {
				fileMap[f.path] = f
			}
		}
	}

	// Load template-specific files (override common files)
	templateFiles, err := loadFilesFromDir(path.Join("templates", name))
	if err != nil {
		return nil, fmt.Errorf("load template %s: %w", name, err)
	}
	for _, f := range templateFiles {
		fileMap[f.path] = f
	}

	// Convert map to slice
	files := make([]templateFile, 0, len(fileMap))
	for _, f := range fileMap {
		files = append(files, f)
	}

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
// For nested templates (e.g., "frontend/react"), it checks
// for the presence of template.json in the nested path.
func templateExists(name string) bool {
	templatePath := path.Join("templates", name)

	// Check if template.json exists (preferred way to identify templates)
	_, err := templatesFS.ReadFile(path.Join(templatePath, "template.json"))
	if err == nil {
		return true
	}

	// Also check if directory exists with actual template files (not just _common)
	entries, err := fs.ReadDir(templatesFS, templatePath)
	if err != nil {
		return false
	}

	// Must have at least one non-hidden, non-metadata file or directory
	for _, entry := range entries {
		name := entry.Name()
		if name == "template.json" || strings.HasPrefix(name, "_") || strings.HasPrefix(name, ".") {
			continue
		}
		return true
	}

	return false
}
