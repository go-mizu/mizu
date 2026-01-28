package postgres

import (
	"context"
	"database/sql"

	"github.com/go-mizu/mizu/blueprints/search/store"
)

// WidgetStore handles widget settings and cheat sheets (stub implementation).
type WidgetStore struct {
	db *sql.DB
}

func (s *WidgetStore) GetWidgetSettings(ctx context.Context, userID string) ([]*store.WidgetSetting, error) {
	return nil, nil
}

func (s *WidgetStore) SetWidgetSetting(ctx context.Context, setting *store.WidgetSetting) error {
	return nil
}

func (s *WidgetStore) GetCheatSheet(ctx context.Context, language string) (*store.CheatSheet, error) {
	return nil, nil
}

func (s *WidgetStore) SaveCheatSheet(ctx context.Context, sheet *store.CheatSheet) error {
	return nil
}

func (s *WidgetStore) ListCheatSheets(ctx context.Context) ([]*store.CheatSheet, error) {
	return nil, nil
}

func (s *WidgetStore) SeedCheatSheets(ctx context.Context) error {
	return nil
}

func (s *WidgetStore) GetRelatedSearches(ctx context.Context, queryHash string) ([]string, error) {
	return nil, nil
}

func (s *WidgetStore) SaveRelatedSearches(ctx context.Context, queryHash, query string, related []string) error {
	return nil
}
