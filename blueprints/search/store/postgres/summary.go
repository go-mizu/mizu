package postgres

import (
	"context"
	"database/sql"

	"github.com/go-mizu/mizu/blueprints/search/store"
)

// SummaryStore handles URL/text summarization cache (stub implementation).
type SummaryStore struct {
	db *sql.DB
}

func (s *SummaryStore) GetSummary(ctx context.Context, urlHash, engine, summaryType, lang string) (*store.SummaryCache, error) {
	return nil, nil
}

func (s *SummaryStore) SaveSummary(ctx context.Context, summary *store.SummaryCache) error {
	return nil
}

func (s *SummaryStore) DeleteExpiredSummaries(ctx context.Context) error {
	return nil
}
