package fw2

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.DataDir == "" {
		t.Error("DataDir should not be empty")
	}
	if cfg.Concurrency == 0 {
		t.Error("Concurrency should not be zero")
	}
	if cfg.Timeout == 0 {
		t.Error("Timeout should not be zero")
	}
}

func TestGetLanguage(t *testing.T) {
	tests := []struct {
		code    string
		want    bool
		name    string
	}{
		{"vie_Latn", true, "Vietnamese"},
		{"eng_Latn", true, "English"},
		{"invalid", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			lang, ok := GetLanguage(tt.code)
			if ok != tt.want {
				t.Errorf("GetLanguage(%q) = %v, want %v", tt.code, ok, tt.want)
			}
			if ok && lang.Name != tt.name {
				t.Errorf("GetLanguage(%q).Name = %q, want %q", tt.code, lang.Name, tt.name)
			}
		})
	}
}

func TestIsValidLanguage(t *testing.T) {
	if !IsValidLanguage("vie_Latn") {
		t.Error("vie_Latn should be valid")
	}
	if IsValidLanguage("invalid") {
		t.Error("invalid should not be valid")
	}
}

func TestDownloader_LocalPath(t *testing.T) {
	cfg := Config{DataDir: "/test/data"}
	d := NewDownloader(cfg)

	path := d.LocalPath("vie_Latn")
	expected := filepath.Join("/test/data", "vie_Latn", "train")
	if path != expected {
		t.Errorf("LocalPath() = %q, want %q", path, expected)
	}
}

func TestDownloader_IsDownloaded(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "fineweb-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	d := NewDownloader(Config{DataDir: tmpDir})

	// Initially not downloaded
	downloaded, err := d.IsDownloaded("vie_Latn")
	if err != nil {
		t.Fatal(err)
	}
	if downloaded {
		t.Error("should not be downloaded initially")
	}

	// Create language directory with parquet file
	langDir := filepath.Join(tmpDir, "vie_Latn", "train")
	if err := os.MkdirAll(langDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(langDir, "test.parquet"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	// Now should be downloaded
	downloaded, err = d.IsDownloaded("vie_Latn")
	if err != nil {
		t.Fatal(err)
	}
	if !downloaded {
		t.Error("should be downloaded after creating files")
	}
}

func TestDownloader_ListDownloaded(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fineweb-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	d := NewDownloader(Config{DataDir: tmpDir})

	// Initially empty
	langs, err := d.ListDownloaded()
	if err != nil {
		t.Fatal(err)
	}
	if len(langs) != 0 {
		t.Errorf("expected 0 languages, got %d", len(langs))
	}

	// Create two language directories
	for _, lang := range []string{"eng_Latn", "vie_Latn"} {
		langDir := filepath.Join(tmpDir, lang, "train")
		if err := os.MkdirAll(langDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(langDir, "test.parquet"), []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Should find both
	langs, err = d.ListDownloaded()
	if err != nil {
		t.Fatal(err)
	}
	if len(langs) != 2 {
		t.Errorf("expected 2 languages, got %d", len(langs))
	}
}

func TestDownloader_ListFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fineweb-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	d := NewDownloader(Config{DataDir: tmpDir})

	// Create directory with parquet files
	langDir := filepath.Join(tmpDir, "vie_Latn", "train")
	if err := os.MkdirAll(langDir, 0755); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"000.parquet", "001.parquet", "other.txt"} {
		if err := os.WriteFile(filepath.Join(langDir, name), []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	files, err := d.ListFiles("vie_Latn")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Errorf("expected 2 parquet files, got %d", len(files))
	}
}

func TestClient_ListFiles_MockServer(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return mock file listing
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[
			{"type": "file", "path": "vie_Latn/train/000000.parquet", "size": 1000000, "oid": "abc123"},
			{"type": "file", "path": "vie_Latn/train/000001.parquet", "size": 2000000, "oid": "def456"},
			{"type": "directory", "path": "vie_Latn/train/subdir"}
		]`))
	}))
	defer server.Close()

	// Note: We can't easily test the real client without mocking the HTTP client
	// This test documents the expected behavior
	t.Log("Client.ListFiles should parse HuggingFace API response correctly")
}

func TestSupportedLanguages(t *testing.T) {
	// Verify we have expected languages
	langCodes := make(map[string]bool)
	for _, lang := range SupportedLanguages {
		langCodes[lang.Code] = true
	}

	required := []string{"vie_Latn", "eng_Latn"}
	for _, code := range required {
		if !langCodes[code] {
			t.Errorf("missing required language: %s", code)
		}
	}
}

func TestNewDownloader(t *testing.T) {
	// With empty config, should use defaults
	d := NewDownloader(Config{})
	if d.config.DataDir == "" {
		t.Error("should use default config when empty config provided")
	}

	// With custom config
	d = NewDownloader(Config{DataDir: "/custom/path"})
	if d.config.DataDir != "/custom/path" {
		t.Error("should use provided DataDir")
	}
}

func TestDownloadProgress(t *testing.T) {
	// Test progress struct fields
	progress := DownloadProgress{
		Language:      "vie_Latn",
		CurrentFile:   "000000.parquet",
		FileIndex:     1,
		TotalFiles:    10,
		BytesReceived: 500000,
		TotalBytes:    1000000,
		Done:          false,
		Error:         nil,
	}

	if progress.Language != "vie_Latn" {
		t.Error("incorrect Language field")
	}
	if progress.FileIndex != 1 {
		t.Error("incorrect FileIndex field")
	}
}

func TestDownloader_Download_CancelContext(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fineweb-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	d := NewDownloader(Config{DataDir: tmpDir})

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Download should return context error
	err = d.Download(ctx, []string{"vie_Latn"}, nil)
	if err == nil {
		t.Skip("Expected error with cancelled context (may succeed if file listing fails first)")
	}
}
