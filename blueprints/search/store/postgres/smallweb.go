package postgres

import (
	"context"
	"database/sql"

	"github.com/go-mizu/mizu/blueprints/search/store"
)

// SmallWebStore handles small web index for enrichment (stub implementation).
type SmallWebStore struct {
	db *sql.DB
}

func (s *SmallWebStore) IndexEntry(ctx context.Context, entry *store.SmallWebEntry) error {
	return nil
}

func (s *SmallWebStore) SearchWeb(ctx context.Context, query string, limit int) ([]*store.EnrichmentResult, error) {
	return nil, nil
}

func (s *SmallWebStore) SearchNews(ctx context.Context, query string, limit int) ([]*store.EnrichmentResult, error) {
	return nil, nil
}

func (s *SmallWebStore) SeedSmallWeb(ctx context.Context) error {
	return nil
}
