// Package widget provides widget generation for search results.
package widget

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/go-mizu/mizu/blueprints/search/store"
	"github.com/go-mizu/mizu/blueprints/search/types"
)

// Service handles widget generation and settings.
type Service struct {
	store store.WidgetStore
}

// NewService creates a new widget service.
func NewService(st store.WidgetStore) *Service {
	return &Service{store: st}
}

// GetSettings returns widget settings for a user.
func (s *Service) GetSettings(ctx context.Context, userID string) ([]*types.WidgetSetting, error) {
	return s.store.GetWidgetSettings(ctx, userID)
}

// UpdateSetting updates a widget setting.
func (s *Service) UpdateSetting(ctx context.Context, setting *types.WidgetSetting) error {
	return s.store.SetWidgetSetting(ctx, setting)
}

// GetCheatSheet returns a programming cheat sheet.
func (s *Service) GetCheatSheet(ctx context.Context, language string) (*types.CheatSheet, error) {
	return s.store.GetCheatSheet(ctx, language)
}

// ListCheatSheets returns all available cheat sheets.
func (s *Service) ListCheatSheets(ctx context.Context) ([]*types.CheatSheet, error) {
	return s.store.ListCheatSheets(ctx)
}

// GetRelatedSearches returns related searches for a query.
func (s *Service) GetRelatedSearches(ctx context.Context, query string) ([]string, error) {
	hash := hashQuery(query)
	return s.store.GetRelatedSearches(ctx, hash)
}

// SaveRelatedSearches caches related searches.
func (s *Service) SaveRelatedSearches(ctx context.Context, query string, related []string) error {
	hash := hashQuery(query)
	return s.store.SaveRelatedSearches(ctx, hash, query, related)
}

// GenerateWidgets generates widgets for a search query.
func (s *Service) GenerateWidgets(ctx context.Context, query string, results []types.SearchResult) []types.Widget {
	var widgets []types.Widget

	// Check if this is a programming-related query
	lang := detectProgrammingLanguage(query)
	if lang != "" {
		if sheet, err := s.store.GetCheatSheet(ctx, lang); err == nil && sheet != nil {
			widgets = append(widgets, types.Widget{
				Type:     types.WidgetCheatSheet,
				Title:    sheet.Title,
				Position: 0,
				Content:  sheet,
			})
		}
	}

	// Add related searches widget
	if related, err := s.GetRelatedSearches(ctx, query); err == nil && len(related) > 0 {
		widgets = append(widgets, types.Widget{
			Type:     types.WidgetRelatedSearches,
			Title:    "Related Searches",
			Position: -1, // Sidebar
			Content:  related,
		})
	}

	return widgets
}

// SeedCheatSheets seeds default cheat sheets.
func (s *Service) SeedCheatSheets(ctx context.Context) error {
	return s.store.SeedCheatSheets(ctx)
}

// detectProgrammingLanguage detects if query is about a programming language.
func detectProgrammingLanguage(query string) string {
	query = strings.ToLower(query)

	languages := map[string][]string{
		"go":         {"golang", "go lang", "go programming", "go tutorial", "go for loop", "go slice", "go map"},
		"python":     {"python", "python3", "python programming", "python tutorial", "python for loop", "python list"},
		"javascript": {"javascript", "js ", "js tutorial", "javascript tutorial", "js array", "javascript array"},
		"typescript": {"typescript", "ts ", "ts tutorial", "typescript tutorial"},
		"rust":       {"rust ", "rust lang", "rustlang", "rust programming", "rust tutorial"},
	}

	for lang, keywords := range languages {
		for _, keyword := range keywords {
			if strings.Contains(query, keyword) {
				return lang
			}
		}
	}

	return ""
}

// hashQuery generates a hash for a query.
func hashQuery(query string) string {
	h := sha256.New()
	h.Write([]byte(strings.ToLower(strings.TrimSpace(query))))
	return hex.EncodeToString(h.Sum(nil))[:16]
}
