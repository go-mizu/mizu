package local

import (
	"context"
	"testing"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine"
)

func TestAdapterImplementsEngine(t *testing.T) {
	// Verify that Adapter implements engine.Engine interface
	var _ engine.Engine = (*Adapter)(nil)
}

func TestNewAdapterWithDefaults(t *testing.T) {
	adapter := NewAdapterWithDefaults()
	if adapter == nil {
		t.Fatal("NewAdapterWithDefaults returned nil")
	}

	if adapter.MetaSearch() == nil {
		t.Fatal("Adapter.MetaSearch() returned nil")
	}
}

func TestAdapterName(t *testing.T) {
	adapter := NewAdapterWithDefaults()
	if adapter.Name() != "local" {
		t.Errorf("Expected name 'local', got '%s'", adapter.Name())
	}
}

func TestAdapterCategories(t *testing.T) {
	adapter := NewAdapterWithDefaults()
	categories := adapter.Categories()

	expectedCategories := []engine.Category{
		engine.CategoryGeneral,
		engine.CategoryImages,
		engine.CategoryVideos,
		engine.CategoryNews,
		engine.CategoryMusic,
		engine.CategoryFiles,
		engine.CategoryIT,
		engine.CategoryScience,
		engine.CategorySocial,
		engine.CategoryMaps,
	}

	if len(categories) != len(expectedCategories) {
		t.Errorf("Expected %d categories, got %d", len(expectedCategories), len(categories))
	}

	for i, expected := range expectedCategories {
		if categories[i] != expected {
			t.Errorf("Category %d: expected '%s', got '%s'", i, expected, categories[i])
		}
	}
}

func TestAdapterHealthz(t *testing.T) {
	adapter := NewAdapterWithDefaults()
	ctx := context.Background()

	err := adapter.Healthz(ctx)
	if err != nil {
		t.Errorf("Healthz should return nil for local adapter, got: %v", err)
	}
}

func TestAdapterConvertSearchOptions(t *testing.T) {
	adapter := NewAdapterWithDefaults()

	tests := []struct {
		name     string
		input    engine.SearchOptions
		expected SearchOptions
	}{
		{
			name: "basic options",
			input: engine.SearchOptions{
				Page:    2,
				PerPage: 20,
			},
			expected: SearchOptions{
				Page:    2,
				PerPage: 20,
			},
		},
		{
			name: "with category",
			input: engine.SearchOptions{
				Category: engine.CategoryImages,
				Page:     1,
			},
			expected: SearchOptions{
				Categories: []Category{CategoryImages},
				Page:       1,
			},
		},
		{
			name: "with time range",
			input: engine.SearchOptions{
				TimeRange: "week",
			},
			expected: SearchOptions{
				TimeRange: TimeRangeWeek,
			},
		},
		{
			name: "with safe search moderate",
			input: engine.SearchOptions{
				SafeSearch: 1,
			},
			expected: SearchOptions{
				SafeSearch: SafeSearchModerate,
			},
		},
		{
			name: "with safe search strict",
			input: engine.SearchOptions{
				SafeSearch: 2,
			},
			expected: SearchOptions{
				SafeSearch: SafeSearchStrict,
			},
		},
		{
			name: "with language and region",
			input: engine.SearchOptions{
				Language: "en",
				Region:   "us",
			},
			expected: SearchOptions{
				Language: "en",
				Locale:   "us",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := adapter.convertSearchOptions(tt.input)

			if result.Page != tt.expected.Page {
				t.Errorf("Page: expected %d, got %d", tt.expected.Page, result.Page)
			}
			if result.PerPage != tt.expected.PerPage {
				t.Errorf("PerPage: expected %d, got %d", tt.expected.PerPage, result.PerPage)
			}
			if result.TimeRange != tt.expected.TimeRange {
				t.Errorf("TimeRange: expected %s, got %s", tt.expected.TimeRange, result.TimeRange)
			}
			if result.SafeSearch != tt.expected.SafeSearch {
				t.Errorf("SafeSearch: expected %d, got %d", tt.expected.SafeSearch, result.SafeSearch)
			}
			if result.Language != tt.expected.Language {
				t.Errorf("Language: expected %s, got %s", tt.expected.Language, result.Language)
			}
			if result.Locale != tt.expected.Locale {
				t.Errorf("Locale: expected %s, got %s", tt.expected.Locale, result.Locale)
			}
		})
	}
}

func TestAdapterConvertToLocalCategory(t *testing.T) {
	adapter := NewAdapterWithDefaults()

	tests := []struct {
		input    engine.Category
		expected Category
	}{
		{engine.CategoryGeneral, CategoryGeneral},
		{engine.CategoryImages, CategoryImages},
		{engine.CategoryVideos, CategoryVideos},
		{engine.CategoryNews, CategoryNews},
		{engine.CategoryMusic, CategoryMusic},
		{engine.CategoryFiles, CategoryFiles},
		{engine.CategoryIT, CategoryIT},
		{engine.CategoryScience, CategoryScience},
		{engine.CategorySocial, CategorySocial},
		{engine.CategoryMaps, CategoryMaps},
		{"unknown", CategoryGeneral}, // Default fallback
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			result := adapter.convertToLocalCategory(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestAdapterConvertToEngineCategory(t *testing.T) {
	adapter := NewAdapterWithDefaults()

	tests := []struct {
		input    Category
		expected engine.Category
	}{
		{CategoryGeneral, engine.CategoryGeneral},
		{CategoryWeb, engine.CategoryGeneral}, // Web maps to General
		{CategoryImages, engine.CategoryImages},
		{CategoryVideos, engine.CategoryVideos},
		{CategoryNews, engine.CategoryNews},
		{CategoryMusic, engine.CategoryMusic},
		{CategoryFiles, engine.CategoryFiles},
		{CategoryIT, engine.CategoryIT},
		{CategoryScience, engine.CategoryScience},
		{CategorySocial, engine.CategorySocial},
		{CategoryMaps, engine.CategoryMaps},
		{"unknown", engine.CategoryGeneral}, // Default fallback
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			result := adapter.convertToEngineCategory(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestAdapterSearch(t *testing.T) {
	adapter := NewAdapterWithDefaults()
	ctx := context.Background()

	// Test basic search with empty query (should return empty results)
	opts := engine.SearchOptions{
		Category: engine.CategoryGeneral,
		Page:     1,
		PerPage:  10,
	}

	resp, err := adapter.Search(ctx, "test query", opts)
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}

	if resp == nil {
		t.Fatal("Search returned nil response")
	}

	if resp.Query != "test query" {
		t.Errorf("Expected query 'test query', got '%s'", resp.Query)
	}

	if resp.Page != 1 {
		t.Errorf("Expected page 1, got %d", resp.Page)
	}
}

func TestNewAdapterWithConfig(t *testing.T) {
	config := &Config{
		RequestTimeout:    10000000000,
		MaxRequestTimeout: 30000000000,
		DefaultPageSize:   20,
		DefaultLanguage:   "en",
		DefaultLocale:     "en-US",
		DefaultCategories: []Category{CategoryGeneral},
	}

	adapter := NewAdapterWithConfig(config)
	if adapter == nil {
		t.Fatal("NewAdapterWithConfig returned nil")
	}

	ms := adapter.MetaSearch()
	if ms == nil {
		t.Fatal("MetaSearch is nil")
	}
}
