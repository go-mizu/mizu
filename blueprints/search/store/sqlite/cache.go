package sqlite

import (
	"context"
	"database/sql"

	"github.com/go-mizu/mizu/blueprints/search/feature/search"
)

// CacheStore implements search.CacheStore using SQLite.
type CacheStore struct {
	db *sql.DB
}

// NewCacheStore creates a new SQLite cache store.
func NewCacheStore(db *sql.DB) *CacheStore {
	return &CacheStore{db: db}
}

// Get retrieves a cache entry by hash.
func (s *CacheStore) Get(ctx context.Context, hash string) (*search.CacheEntry, error) {
	var entry search.CacheEntry
	err := s.db.QueryRowContext(ctx, `
		SELECT hash, query, category, results_json, created_at, expires_at
		FROM search_cache
		WHERE hash = ?
	`, hash).Scan(
		&entry.Hash,
		&entry.Query,
		&entry.Category,
		&entry.ResultsJSON,
		&entry.CreatedAt,
		&entry.ExpiresAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &entry, nil
}

// Set stores a cache entry.
func (s *CacheStore) Set(ctx context.Context, entry *search.CacheEntry) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO search_cache (hash, query, category, results_json, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(hash) DO UPDATE SET
			results_json = excluded.results_json,
			created_at = excluded.created_at,
			expires_at = excluded.expires_at
	`, entry.Hash, entry.Query, entry.Category, entry.ResultsJSON, entry.CreatedAt, entry.ExpiresAt)

	return err
}

// Delete removes a cache entry by hash.
func (s *CacheStore) Delete(ctx context.Context, hash string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM search_cache WHERE hash = ?
	`, hash)
	return err
}

// DeleteExpired removes all expired cache entries.
func (s *CacheStore) DeleteExpired(ctx context.Context) (int64, error) {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM search_cache WHERE expires_at < CURRENT_TIMESTAMP
	`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
