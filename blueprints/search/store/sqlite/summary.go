package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/store"
	"github.com/go-mizu/mizu/blueprints/search/types"
)

// SummaryStore handles URL/text summarization cache.
type SummaryStore struct {
	db *sql.DB
}

// GetSummary retrieves a cached summary.
func (s *SummaryStore) GetSummary(ctx context.Context, urlHash, engine, summaryType, lang string) (*store.SummaryCache, error) {
	var summary store.SummaryCache
	var targetLang sql.NullString
	var expiresAt sql.NullTime

	err := s.db.QueryRowContext(ctx, `
		SELECT id, url_hash, url, engine, summary_type, target_language, output, tokens, created_at, expires_at
		FROM summaries_cache
		WHERE url_hash = ? AND engine = ? AND summary_type = ? AND (target_language = ? OR (target_language IS NULL AND ? = ''))
		AND (expires_at IS NULL OR expires_at > datetime('now'))
	`, urlHash, engine, summaryType, lang, lang).Scan(
		&summary.ID, &summary.URLHash, &summary.URL, &summary.Engine, &summary.SummaryType,
		&targetLang, &summary.Output, &summary.Tokens, &summary.CreatedAt, &expiresAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if targetLang.Valid {
		summary.TargetLanguage = targetLang.String
	}
	if expiresAt.Valid {
		summary.ExpiresAt = expiresAt.Time
	}

	return &summary, nil
}

// SaveSummary caches a summary.
func (s *SummaryStore) SaveSummary(ctx context.Context, summary *store.SummaryCache) error {
	summary.CreatedAt = time.Now()
	// Cache for 24 hours by default
	summary.ExpiresAt = summary.CreatedAt.Add(24 * time.Hour)

	var targetLang interface{}
	if summary.TargetLanguage != "" {
		targetLang = summary.TargetLanguage
	}

	result, err := s.db.ExecContext(ctx, `
		INSERT INTO summaries_cache (url_hash, url, engine, summary_type, target_language, output, tokens, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(url_hash, engine, summary_type, target_language) DO UPDATE SET
			output = excluded.output,
			tokens = excluded.tokens,
			created_at = excluded.created_at,
			expires_at = excluded.expires_at
	`, summary.URLHash, summary.URL, summary.Engine, summary.SummaryType,
		targetLang, summary.Output, summary.Tokens, summary.CreatedAt, summary.ExpiresAt)

	if err != nil {
		return err
	}

	id, _ := result.LastInsertId()
	summary.ID = id
	return nil
}

// DeleteExpiredSummaries removes expired cache entries.
func (s *SummaryStore) DeleteExpiredSummaries(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM summaries_cache WHERE expires_at < datetime('now')")
	return err
}

// Ensure types are correctly used
var (
	_ types.SummaryEngine = types.EngineCecil
	_ types.SummaryType   = types.SummaryTypeSummary
)
