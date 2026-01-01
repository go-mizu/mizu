package files

import (
	"path/filepath"
	"strings"
)

// PreviewType represents the type of preview for a file.
type PreviewType string

const (
	PreviewTypeImage        PreviewType = "image"
	PreviewTypeVideo        PreviewType = "video"
	PreviewTypeAudio        PreviewType = "audio"
	PreviewTypePDF          PreviewType = "pdf"
	PreviewTypeCode         PreviewType = "code"
	PreviewTypeText         PreviewType = "text"
	PreviewTypeMarkdown     PreviewType = "markdown"
	PreviewTypeSpreadsheet  PreviewType = "spreadsheet"
	PreviewTypeDocument     PreviewType = "document"
	PreviewTypePresentation PreviewType = "presentation"
	PreviewType3D           PreviewType = "3d"
	PreviewTypeArchive      PreviewType = "archive"
	PreviewTypeFont         PreviewType = "font"
	PreviewTypeUnsupported  PreviewType = "unsupported"
)

// PreviewInfo contains preview metadata.
type PreviewInfo struct {
	ID           string      `json:"id"`
	Name         string      `json:"name"`
	MimeType     string      `json:"mime_type"`
	Size         int64       `json:"size"`
	PreviewType  PreviewType `json:"preview_type"`
	PreviewURL   string      `json:"preview_url"`
	ThumbnailURL string      `json:"thumbnail_url,omitempty"`
	CanPreview   bool        `json:"can_preview"`
	Language     string      `json:"language,omitempty"`
}

// SiblingFile represents a sibling file for navigation.
type SiblingFile struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// PreviewResponse contains preview data with navigation.
type PreviewResponse struct {
	*PreviewInfo
	Siblings struct {
		Prev *SiblingFile `json:"prev,omitempty"`
		Next *SiblingFile `json:"next,omitempty"`
	} `json:"siblings"`
}

// DetectPreviewType determines the preview type based on MIME type and extension.
func DetectPreviewType(mimeType, filename string) PreviewType {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext != "" {
		ext = ext[1:] // Remove leading dot
	}

	// Check MIME type first
	if strings.HasPrefix(mimeType, "image/") {
		return PreviewTypeImage
	}
	if strings.HasPrefix(mimeType, "video/") {
		return PreviewTypeVideo
	}
	if strings.HasPrefix(mimeType, "audio/") {
		return PreviewTypeAudio
	}
	if mimeType == "application/pdf" {
		return PreviewTypePDF
	}

	// Check by extension for code files
	codeExts := map[string]bool{
		"go": true, "js": true, "ts": true, "jsx": true, "tsx": true,
		"py": true, "rb": true, "rs": true, "java": true, "c": true,
		"cpp": true, "h": true, "hpp": true, "cs": true, "php": true,
		"swift": true, "kt": true, "scala": true, "r": true, "lua": true,
		"sh": true, "bash": true, "zsh": true, "fish": true, "ps1": true,
		"html": true, "css": true, "scss": true, "sass": true, "less": true,
		"xml": true, "svg": true, "vue": true, "svelte": true,
		"sql": true, "graphql": true, "gql": true,
		"dockerfile": true, "makefile": true, "cmake": true,
		"ini": true, "cfg": true, "conf": true,
	}
	if codeExts[ext] {
		return PreviewTypeCode
	}

	// Data formats (also treated as code for syntax highlighting)
	dataExts := map[string]bool{
		"json": true, "yaml": true, "yml": true, "toml": true,
		"env": true, "properties": true,
	}
	if dataExts[ext] {
		return PreviewTypeCode
	}

	// Markdown
	if ext == "md" || ext == "markdown" || ext == "mdx" {
		return PreviewTypeMarkdown
	}

	// Plain text
	textExts := map[string]bool{
		"txt": true, "log": true, "text": true, "readme": true,
		"license": true, "authors": true, "changelog": true,
		"gitignore": true, "gitattributes": true, "editorconfig": true,
	}
	if textExts[ext] {
		return PreviewTypeText
	}

	// Office documents
	spreadsheetExts := map[string]bool{
		"xlsx": true, "xls": true, "csv": true, "tsv": true, "ods": true,
	}
	if spreadsheetExts[ext] {
		return PreviewTypeSpreadsheet
	}

	documentExts := map[string]bool{
		"docx": true, "doc": true, "odt": true, "rtf": true,
	}
	if documentExts[ext] {
		return PreviewTypeDocument
	}

	presentationExts := map[string]bool{
		"pptx": true, "ppt": true, "odp": true,
	}
	if presentationExts[ext] {
		return PreviewTypePresentation
	}

	// 3D models
	model3DExts := map[string]bool{
		"obj": true, "stl": true, "gltf": true, "glb": true,
		"fbx": true, "dae": true, "3ds": true,
	}
	if model3DExts[ext] {
		return PreviewType3D
	}

	// Archives
	archiveExts := map[string]bool{
		"zip": true, "tar": true, "gz": true, "bz2": true, "xz": true,
		"7z": true, "rar": true, "tgz": true,
	}
	if archiveExts[ext] {
		return PreviewTypeArchive
	}

	// Fonts
	fontExts := map[string]bool{
		"ttf": true, "otf": true, "woff": true, "woff2": true, "eot": true,
	}
	if fontExts[ext] {
		return PreviewTypeFont
	}

	// Check text MIME types
	if strings.HasPrefix(mimeType, "text/") {
		return PreviewTypeText
	}

	return PreviewTypeUnsupported
}

// GetLanguage returns the programming language for syntax highlighting.
func GetLanguage(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext != "" {
		ext = ext[1:]
	}

	// Map extensions to highlight.js language names
	langMap := map[string]string{
		"go":         "go",
		"js":         "javascript",
		"jsx":        "javascript",
		"ts":         "typescript",
		"tsx":        "typescript",
		"py":         "python",
		"rb":         "ruby",
		"rs":         "rust",
		"java":       "java",
		"c":          "c",
		"cpp":        "cpp",
		"h":          "c",
		"hpp":        "cpp",
		"cs":         "csharp",
		"php":        "php",
		"swift":      "swift",
		"kt":         "kotlin",
		"scala":      "scala",
		"r":          "r",
		"lua":        "lua",
		"sh":         "bash",
		"bash":       "bash",
		"zsh":        "bash",
		"fish":       "bash",
		"ps1":        "powershell",
		"html":       "html",
		"css":        "css",
		"scss":       "scss",
		"sass":       "scss",
		"less":       "less",
		"xml":        "xml",
		"svg":        "xml",
		"vue":        "html",
		"svelte":     "html",
		"json":       "json",
		"yaml":       "yaml",
		"yml":        "yaml",
		"toml":       "toml",
		"ini":        "ini",
		"sql":        "sql",
		"graphql":    "graphql",
		"gql":        "graphql",
		"md":         "markdown",
		"markdown":   "markdown",
		"dockerfile": "dockerfile",
		"makefile":   "makefile",
	}

	if lang, ok := langMap[ext]; ok {
		return lang
	}

	// Check filename patterns
	name := strings.ToLower(filepath.Base(filename))
	if strings.Contains(name, "dockerfile") {
		return "dockerfile"
	}
	if strings.Contains(name, "makefile") || name == "gnumakefile" {
		return "makefile"
	}

	return "plaintext"
}

// CanPreview returns whether a file can be previewed.
func CanPreview(mimeType, filename string) bool {
	pt := DetectPreviewType(mimeType, filename)
	return pt != PreviewTypeUnsupported
}

// GetPreviewInfo creates preview info for a file.
func GetPreviewInfo(f *File, baseURL string) *PreviewInfo {
	previewType := DetectPreviewType(f.MimeType, f.Name)

	info := &PreviewInfo{
		ID:          f.ID,
		Name:        f.Name,
		MimeType:    f.MimeType,
		Size:        f.Size,
		PreviewType: previewType,
		PreviewURL:  baseURL + "/api/v1/files/" + f.ID + "/content",
		CanPreview:  previewType != PreviewTypeUnsupported,
	}

	// Add thumbnail URL for supported types
	if previewType == PreviewTypeImage || previewType == PreviewTypeVideo || previewType == PreviewTypePDF {
		info.ThumbnailURL = baseURL + "/api/v1/files/" + f.ID + "/thumbnail"
	}

	// Add language for code files
	if previewType == PreviewTypeCode {
		info.Language = GetLanguage(f.Name)
	}

	return info
}
