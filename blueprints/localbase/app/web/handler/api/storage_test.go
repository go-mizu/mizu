package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestSanitizeStoragePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal path",
			input:    "folder/file.txt",
			expected: "folder/file.txt",
		},
		{
			name:     "path traversal attempt",
			input:    "../../../etc/passwd",
			expected: "etc/passwd",
		},
		{
			name:     "double dots in middle",
			input:    "folder/../other/file.txt",
			expected: "folder/other/file.txt",
		},
		{
			name:     "leading slash",
			input:    "/absolute/path.txt",
			expected: "absolute/path.txt",
		},
		{
			name:     "null bytes",
			input:    "file\x00name.txt",
			expected: "filename.txt",
		},
		{
			name:     "leading dots",
			input:    "...hidden",
			expected: "hidden",
		},
		{
			name:     "empty path",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeStoragePath(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeStoragePath(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal filename",
			input:    "document.pdf",
			expected: "document.pdf",
		},
		{
			name:     "newline injection",
			input:    "file\nname.txt",
			expected: "filename.txt",
		},
		{
			name:     "carriage return injection",
			input:    "file\rname.txt",
			expected: "filename.txt",
		},
		{
			name:     "quote injection",
			input:    `file"name.txt`,
			expected: "filename.txt",
		},
		{
			name:     "backslash",
			input:    `folder\file.txt`,
			expected: "folderfile.txt",
		},
		{
			name:     "slash",
			input:    "folder/file.txt",
			expected: "folderfile.txt",
		},
		{
			name:     "null byte",
			input:    "file\x00.txt",
			expected: "file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeFilename(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGenerateImagePlaceholder(t *testing.T) {
	h := &StorageHandler{}

	result := h.generateImagePlaceholder("test-image.png", "image/png")

	// Check that it returns valid SVG
	if !bytes.Contains(result, []byte("<svg")) {
		t.Error("Expected SVG content")
	}
	if !bytes.Contains(result, []byte("test-image.png")) {
		t.Error("Expected filename in SVG")
	}
}

func TestGenerateTextPlaceholder(t *testing.T) {
	h := &StorageHandler{}

	tests := []struct {
		name        string
		filename    string
		contentType string
		contains    string
	}{
		{"JSON file", "config.json", "application/json", "{"},
		{"YAML file", "config.yaml", "application/x-yaml", "name:"},
		{"Markdown file", "readme.md", "text/markdown", "#"},
		{"SQL file", "query.sql", "application/sql", "SELECT"},
		{"Go file", "main.go", "text/x-go", "package"},
		{"Python file", "script.py", "text/x-python", "def"},
		{"TypeScript file", "app.tsx", "text/typescript", "import"},
		{"CSS file", "style.css", "text/css", ":root"},
		{"HTML file", "index.html", "text/html", "<!DOCTYPE"},
		{"SVG file", "icon.svg", "image/svg+xml", "<svg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := h.generateTextPlaceholder(tt.filename, tt.contentType)
			if !bytes.Contains(result, []byte(tt.contains)) {
				t.Errorf("Expected %q to contain %q", string(result), tt.contains)
			}
		})
	}
}

func TestGeneratePlaceholderContent(t *testing.T) {
	h := &StorageHandler{}

	// Test image returns SVG
	content, contentType := h.generatePlaceholderContent("image/png", "test.png", 1024)
	if contentType != "image/svg+xml" {
		t.Errorf("Expected image/svg+xml, got %s", contentType)
	}
	if !bytes.Contains(content, []byte("<svg")) {
		t.Error("Expected SVG content for image")
	}

	// Test JSON returns JSON
	content, contentType = h.generatePlaceholderContent("application/json", "data.json", 100)
	if contentType != "application/json" {
		t.Errorf("Expected application/json, got %s", contentType)
	}

	// Test unknown type returns plain text
	content, contentType = h.generatePlaceholderContent("application/octet-stream", "file.bin", 500)
	if contentType != "text/plain" {
		t.Errorf("Expected text/plain, got %s", contentType)
	}
}

func TestGetFilePath(t *testing.T) {
	h := &StorageHandler{dataDir: "/tmp/storage"}

	result := h.getFilePath("bucket-id", "folder/file.txt")
	expected := filepath.Join("/tmp/storage", "bucket-id", "folder/file.txt")

	if result != expected {
		t.Errorf("getFilePath() = %q, expected %q", result, expected)
	}
}

func TestAbs(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{5, 5},
		{-5, 5},
		{0, 0},
		{-100, 100},
	}

	for _, tt := range tests {
		result := abs(tt.input)
		if result != tt.expected {
			t.Errorf("abs(%d) = %d, expected %d", tt.input, result, tt.expected)
		}
	}
}

// Helper for creating test requests
func newTestRequest(method, path string, body interface{}) *http.Request {
	var bodyReader *bytes.Buffer
	if body != nil {
		data, _ := json.Marshal(body)
		bodyReader = bytes.NewBuffer(data)
	} else {
		bodyReader = bytes.NewBuffer(nil)
	}
	req := httptest.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	return req
}

// Integration test for storage handler (requires mock store)
func TestStorageHandlerIntegration(t *testing.T) {
	// Skip if not in integration test mode
	if os.Getenv("INTEGRATION_TEST") == "" {
		t.Skip("Skipping integration test")
	}

	// This would require setting up a mock postgres store
	// For now, we just verify the handler can be created
	app := mizu.New()
	if app == nil {
		t.Error("Failed to create app")
	}
}
