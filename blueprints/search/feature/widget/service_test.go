package widget

import (
	"context"
	"testing"

	"github.com/go-mizu/mizu/blueprints/search/types"
)

// mockWidgetStore implements store.WidgetStore for testing.
type mockWidgetStore struct {
	settings     map[string][]*types.WidgetSetting
	cheatSheets  map[string]*types.CheatSheet
	relatedCache map[string][]string
}

func newMockWidgetStore() *mockWidgetStore {
	return &mockWidgetStore{
		settings:     make(map[string][]*types.WidgetSetting),
		cheatSheets:  make(map[string]*types.CheatSheet),
		relatedCache: make(map[string][]string),
	}
}

func (m *mockWidgetStore) GetWidgetSettings(ctx context.Context, userID string) ([]*types.WidgetSetting, error) {
	return m.settings[userID], nil
}

func (m *mockWidgetStore) SetWidgetSetting(ctx context.Context, setting *types.WidgetSetting) error {
	m.settings[setting.UserID] = append(m.settings[setting.UserID], setting)
	return nil
}

func (m *mockWidgetStore) GetCheatSheet(ctx context.Context, language string) (*types.CheatSheet, error) {
	return m.cheatSheets[language], nil
}

func (m *mockWidgetStore) SaveCheatSheet(ctx context.Context, sheet *types.CheatSheet) error {
	m.cheatSheets[sheet.Language] = sheet
	return nil
}

func (m *mockWidgetStore) ListCheatSheets(ctx context.Context) ([]*types.CheatSheet, error) {
	var sheets []*types.CheatSheet
	for _, s := range m.cheatSheets {
		sheets = append(sheets, s)
	}
	return sheets, nil
}

func (m *mockWidgetStore) SeedCheatSheets(ctx context.Context) error {
	return nil
}

func (m *mockWidgetStore) GetRelatedSearches(ctx context.Context, queryHash string) ([]string, error) {
	return m.relatedCache[queryHash], nil
}

func (m *mockWidgetStore) SaveRelatedSearches(ctx context.Context, queryHash, query string, related []string) error {
	m.relatedCache[queryHash] = related
	return nil
}

func TestService_GetSettings(t *testing.T) {
	store := newMockWidgetStore()
	store.settings["user1"] = []*types.WidgetSetting{
		{UserID: "user1", WidgetType: types.WidgetCheatSheet, Enabled: true, Position: 0},
		{UserID: "user1", WidgetType: types.WidgetRelatedSearches, Enabled: true, Position: 1},
	}

	svc := NewService(store)
	ctx := context.Background()

	settings, err := svc.GetSettings(ctx, "user1")
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}

	if len(settings) != 2 {
		t.Errorf("GetSettings() returned %d settings, want 2", len(settings))
	}
}

func TestService_GetCheatSheet(t *testing.T) {
	store := newMockWidgetStore()
	store.cheatSheets["go"] = &types.CheatSheet{
		Language: "go",
		Title:    "Go Cheat Sheet",
		Sections: []types.CheatSection{
			{
				Name: "Variables",
				Items: []types.CheatItem{
					{Code: "var x int", Description: "Declare variable"},
					{Code: "x := 10", Description: "Short declaration"},
				},
			},
		},
	}

	svc := NewService(store)
	ctx := context.Background()

	sheet, err := svc.GetCheatSheet(ctx, "go")
	if err != nil {
		t.Fatalf("GetCheatSheet() error = %v", err)
	}

	if sheet == nil {
		t.Fatal("GetCheatSheet() returned nil")
	}

	if sheet.Language != "go" {
		t.Errorf("GetCheatSheet() language = %q, want %q", sheet.Language, "go")
	}

	if len(sheet.Sections) != 1 {
		t.Errorf("GetCheatSheet() sections = %d, want 1", len(sheet.Sections))
	}
}

func TestService_GetCheatSheet_NotFound(t *testing.T) {
	store := newMockWidgetStore()
	svc := NewService(store)
	ctx := context.Background()

	sheet, err := svc.GetCheatSheet(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("GetCheatSheet() error = %v", err)
	}

	if sheet != nil {
		t.Errorf("GetCheatSheet() = %v, want nil", sheet)
	}
}

func TestService_RelatedSearches(t *testing.T) {
	store := newMockWidgetStore()
	svc := NewService(store)
	ctx := context.Background()

	// Save related searches
	err := svc.SaveRelatedSearches(ctx, "golang tutorial", []string{
		"go programming",
		"golang beginner",
		"go vs rust",
	})
	if err != nil {
		t.Fatalf("SaveRelatedSearches() error = %v", err)
	}

	// Get related searches
	related, err := svc.GetRelatedSearches(ctx, "golang tutorial")
	if err != nil {
		t.Fatalf("GetRelatedSearches() error = %v", err)
	}

	if len(related) != 3 {
		t.Errorf("GetRelatedSearches() returned %d results, want 3", len(related))
	}
}

func TestService_GenerateWidgets_CheatSheet(t *testing.T) {
	store := newMockWidgetStore()
	store.cheatSheets["go"] = &types.CheatSheet{
		Language: "go",
		Title:    "Go Cheat Sheet",
		Sections: []types.CheatSection{},
	}

	svc := NewService(store)
	ctx := context.Background()

	// Test queries that should trigger Go cheat sheet
	queries := []string{
		"golang for loop",
		"go programming tutorial",
		"go slice operations",
		"go map example",
	}

	for _, query := range queries {
		t.Run(query, func(t *testing.T) {
			widgets := svc.GenerateWidgets(ctx, query, nil)

			found := false
			for _, w := range widgets {
				if w.Type == types.WidgetCheatSheet {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("GenerateWidgets(%q) did not include cheat sheet widget", query)
			}
		})
	}
}

func TestService_GenerateWidgets_NoCheatSheet(t *testing.T) {
	store := newMockWidgetStore()
	svc := NewService(store)
	ctx := context.Background()

	// Test queries that should not trigger cheat sheet
	queries := []string{
		"weather today",
		"news headlines",
		"best restaurants near me",
	}

	for _, query := range queries {
		t.Run(query, func(t *testing.T) {
			widgets := svc.GenerateWidgets(ctx, query, nil)

			for _, w := range widgets {
				if w.Type == types.WidgetCheatSheet {
					t.Errorf("GenerateWidgets(%q) unexpectedly included cheat sheet widget", query)
				}
			}
		})
	}
}

func TestDetectProgrammingLanguage(t *testing.T) {
	tests := []struct {
		query string
		want  string
	}{
		{"golang for loop", "go"},
		{"go programming", "go"},
		{"go slice", "go"},
		{"python list comprehension", "python"},
		{"python3 tutorial", "python"},
		{"javascript array", "javascript"},
		{"js tutorial", "javascript"},
		{"typescript interface", "typescript"},
		{"rust lang memory", "rust"},
		{"weather today", ""},
		{"news headlines", ""},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			got := detectProgrammingLanguage(tt.query)
			if got != tt.want {
				t.Errorf("detectProgrammingLanguage(%q) = %q, want %q", tt.query, got, tt.want)
			}
		})
	}
}

func TestHashQuery(t *testing.T) {
	// Same query should produce same hash
	hash1 := hashQuery("test query")
	hash2 := hashQuery("test query")
	if hash1 != hash2 {
		t.Errorf("hashQuery() produced different hashes for same query: %q vs %q", hash1, hash2)
	}

	// Case insensitive
	hash3 := hashQuery("Test Query")
	if hash1 != hash3 {
		t.Errorf("hashQuery() should be case insensitive: %q vs %q", hash1, hash3)
	}

	// Trimmed whitespace
	hash4 := hashQuery("  test query  ")
	if hash1 != hash4 {
		t.Errorf("hashQuery() should trim whitespace: %q vs %q", hash1, hash4)
	}

	// Different queries should produce different hashes
	hash5 := hashQuery("different query")
	if hash1 == hash5 {
		t.Errorf("hashQuery() produced same hash for different queries")
	}

	// Hash should be 16 characters
	if len(hash1) != 16 {
		t.Errorf("hashQuery() length = %d, want 16", len(hash1))
	}
}
