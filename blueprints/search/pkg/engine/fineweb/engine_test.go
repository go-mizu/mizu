package fineweb

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.DataDir == "" {
		t.Error("DataDir should not be empty")
	}
	if cfg.SourceDir == "" {
		t.Error("SourceDir should not be empty")
	}
	if cfg.ResultLimit == 0 {
		t.Error("ResultLimit should not be zero")
	}
	if cfg.ContentSnippetLength == 0 {
		t.Error("ContentSnippetLength should not be zero")
	}
}

func TestExtractTitle(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		url      string
		expected string
	}{
		{
			name:     "first line as title",
			text:     "This is the title\nThis is the body text that follows.",
			url:      "",
			expected: "This is the title",
		},
		{
			name:     "truncate long first line",
			text:     "This is a very long title that should be truncated at some point because it exceeds the maximum length allowed\nBody text",
			url:      "",
			expected: "This is a very long title that should be truncated at some point because it exceeds the maximum...",
		},
		{
			name:     "use first chars when no newline",
			text:     "A very long text without any newlines that should be truncated at word boundary",
			url:      "",
			expected: "A very long text without any newlines that...",
		},
		{
			name:     "fallback to url",
			text:     "",
			url:      "https://example.com/page",
			expected: "example.com",
		},
		{
			name:     "untitled fallback",
			text:     "",
			url:      "",
			expected: "Untitled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTitle(tt.text, tt.url)
			if result != tt.expected {
				t.Errorf("extractTitle() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestTruncateText(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		maxLen   int
		expected string
	}{
		{
			name:     "short text unchanged",
			text:     "Hello world",
			maxLen:   100,
			expected: "Hello world",
		},
		{
			name:     "truncate at word boundary",
			text:     "Hello world this is a test",
			maxLen:   15,
			expected: "Hello world...",
		},
		{
			name:     "remove newlines",
			text:     "Hello\nworld\ntest",
			maxLen:   100,
			expected: "Hello world test",
		},
		{
			name:     "collapse spaces",
			text:     "Hello    world    test",
			maxLen:   100,
			expected: "Hello world test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateText(tt.text, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncateText() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestNewEngine(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fineweb-engine-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := Config{
		DataDir:   filepath.Join(tmpDir, "db"),
		SourceDir: filepath.Join(tmpDir, "data"),
	}

	engine, err := NewEngine(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()

	// Check base engine properties
	if engine.Name() != "fineweb" {
		t.Errorf("Name() = %q, want %q", engine.Name(), "fineweb")
	}
	if engine.Shortcut() != "fw" {
		t.Errorf("Shortcut() = %q, want %q", engine.Shortcut(), "fw")
	}
	if !engine.SupportsPaging() {
		t.Error("should support paging")
	}
	if !engine.SupportsLanguage() {
		t.Error("should support language")
	}
}

func TestEngine_Search_NoData(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fineweb-engine-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := Config{
		DataDir:   filepath.Join(tmpDir, "db"),
		SourceDir: filepath.Join(tmpDir, "data"),
	}

	engine, err := NewEngine(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()

	ctx := context.Background()
	results, err := engine.Search(ctx, "test query", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Should return empty results when no data
	if len(results.Results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results.Results))
	}
}

func TestEngine_GetLanguages_Empty(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fineweb-engine-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := Config{
		DataDir:   filepath.Join(tmpDir, "db"),
		SourceDir: filepath.Join(tmpDir, "data"),
	}

	engine, err := NewEngine(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()

	langs := engine.GetLanguages()
	if len(langs) != 0 {
		t.Errorf("expected 0 languages, got %d", len(langs))
	}
}

func TestEngine_GetDocumentCount_Empty(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fineweb-engine-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := Config{
		DataDir:   filepath.Join(tmpDir, "db"),
		SourceDir: filepath.Join(tmpDir, "data"),
	}

	engine, err := NewEngine(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()

	ctx := context.Background()
	count, err := engine.GetDocumentCount(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Errorf("expected 0 documents, got %d", count)
	}
}

func TestStore_NewStore(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fineweb-store-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewStore("vie_Latn", tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	// Check that database file was created
	dbPath := filepath.Join(tmpDir, "vie_Latn.duckdb")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("database file should be created")
	}
}

func TestStore_Count_Empty(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fineweb-store-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewStore("vie_Latn", tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	ctx := context.Background()
	count, err := store.Count(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Errorf("expected 0 documents, got %d", count)
	}
}

func TestStore_GetImportState_Empty(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fineweb-store-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewStore("vie_Latn", tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	ctx := context.Background()
	states, err := store.GetImportState(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(states) != 0 {
		t.Errorf("expected 0 import states, got %d", len(states))
	}
}

func TestStore_SearchSimple_Empty(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fineweb-store-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewStore("vie_Latn", tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	ctx := context.Background()
	docs, err := store.SearchSimple(ctx, "test", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 0 {
		t.Errorf("expected 0 documents, got %d", len(docs))
	}
}

func TestDocument_Struct(t *testing.T) {
	doc := Document{
		ID:            "test-id",
		URL:           "https://example.com",
		Text:          "Sample text",
		Dump:          "CC-MAIN-2024",
		Date:          "2024-01-01",
		Language:      "vie_Latn",
		LanguageScore: 0.99,
		Score:         1.5,
	}

	if doc.ID != "test-id" {
		t.Error("incorrect ID field")
	}
	if doc.Score != 1.5 {
		t.Error("incorrect Score field")
	}
}

func TestImportState_Struct(t *testing.T) {
	state := ImportState{
		ParquetFile: "000000.parquet",
		ImportedAt:  "2024-01-01 12:00:00",
		RowCount:    1000,
	}

	if state.ParquetFile != "000000.parquet" {
		t.Error("incorrect ParquetFile field")
	}
	if state.RowCount != 1000 {
		t.Error("incorrect RowCount field")
	}
}

func TestSortByScore(t *testing.T) {
	docs := []Document{
		{ID: "1", Score: 1.0},
		{ID: "3", Score: 3.0},
		{ID: "2", Score: 2.0},
	}

	sortByScore(docs)

	if docs[0].ID != "3" || docs[1].ID != "2" || docs[2].ID != "1" {
		t.Errorf("incorrect sort order: %v", docs)
	}
}
