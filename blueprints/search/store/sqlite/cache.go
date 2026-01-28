package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/go-mizu/mizu/blueprints/search/feature/search"
)

// CacheStore implements search.CacheStore using SQLite with versioning.
type CacheStore struct {
	db *sql.DB
}

// NewCacheStore creates a new SQLite cache store.
func NewCacheStore(db *sql.DB) *CacheStore {
	return &CacheStore{db: db}
}

// Get retrieves the latest cache entry by hash.
func (s *CacheStore) Get(ctx context.Context, hash string) (*search.CacheEntry, error) {
	var entry search.CacheEntry
	err := s.db.QueryRowContext(ctx, `
		SELECT hash, query, category, options_json, results_json, version, created_at
		FROM search_cache
		WHERE hash = ?
		ORDER BY version DESC
		LIMIT 1
	`, hash).Scan(
		&entry.Hash,
		&entry.Query,
		&entry.Category,
		&entry.OptionsJSON,
		&entry.ResultsJSON,
		&entry.Version,
		&entry.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &entry, nil
}

// GetVersion retrieves a specific version of a cache entry.
func (s *CacheStore) GetVersion(ctx context.Context, hash string, version int) (*search.CacheEntry, error) {
	var entry search.CacheEntry
	err := s.db.QueryRowContext(ctx, `
		SELECT hash, query, category, options_json, results_json, version, created_at
		FROM search_cache
		WHERE hash = ? AND version = ?
	`, hash, version).Scan(
		&entry.Hash,
		&entry.Query,
		&entry.Category,
		&entry.OptionsJSON,
		&entry.ResultsJSON,
		&entry.Version,
		&entry.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &entry, nil
}

// GetVersions retrieves all versions of a cache entry.
func (s *CacheStore) GetVersions(ctx context.Context, hash string) ([]*search.CacheEntry, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT hash, query, category, options_json, results_json, version, created_at
		FROM search_cache
		WHERE hash = ?
		ORDER BY version DESC
	`, hash)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*search.CacheEntry
	for rows.Next() {
		var entry search.CacheEntry
		if err := rows.Scan(
			&entry.Hash,
			&entry.Query,
			&entry.Category,
			&entry.OptionsJSON,
			&entry.ResultsJSON,
			&entry.Version,
			&entry.CreatedAt,
		); err != nil {
			return nil, err
		}
		entries = append(entries, &entry)
	}

	return entries, rows.Err()
}

// Set stores a new version of a cache entry.
func (s *CacheStore) Set(ctx context.Context, entry *search.CacheEntry) error {
	// Get next version number
	var maxVersion sql.NullInt64
	err := s.db.QueryRowContext(ctx, `
		SELECT MAX(version) FROM search_cache WHERE hash = ?
	`, entry.Hash).Scan(&maxVersion)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	nextVersion := 1
	if maxVersion.Valid {
		nextVersion = int(maxVersion.Int64) + 1
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO search_cache (hash, query, category, options_json, results_json, version, created_at)
		VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`, entry.Hash, entry.Query, entry.Category, entry.OptionsJSON, entry.ResultsJSON, nextVersion)

	return err
}

// Delete removes all versions of a cache entry by hash.
func (s *CacheStore) Delete(ctx context.Context, hash string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM search_cache WHERE hash = ?
	`, hash)
	return err
}

// DeleteVersion removes a specific version of a cache entry.
func (s *CacheStore) DeleteVersion(ctx context.Context, hash string, version int) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM search_cache WHERE hash = ? AND version = ?
	`, hash, version)
	return err
}

// DeleteExpired is kept for interface compatibility but does nothing (no TTL).
func (s *CacheStore) DeleteExpired(ctx context.Context) (int64, error) {
	return 0, nil
}

// DeleteOldVersions keeps only the N most recent versions for each hash.
func (s *CacheStore) DeleteOldVersions(ctx context.Context, keepVersions int) (int64, error) {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM search_cache
		WHERE id NOT IN (
			SELECT id FROM (
				SELECT id, ROW_NUMBER() OVER (PARTITION BY hash ORDER BY version DESC) as rn
				FROM search_cache
			) ranked
			WHERE rn <= ?
		)
	`, keepVersions)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// GetStats returns cache statistics.
func (s *CacheStore) GetStats(ctx context.Context) (*search.CacheStats, error) {
	var stats search.CacheStats

	// Total entries
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM search_cache`).Scan(&stats.TotalEntries)
	if err != nil {
		return nil, err
	}

	// Unique queries
	err = s.db.QueryRowContext(ctx, `SELECT COUNT(DISTINCT hash) FROM search_cache`).Scan(&stats.UniqueQueries)
	if err != nil {
		return nil, err
	}

	// Total size (approximate)
	err = s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(LENGTH(results_json)), 0) FROM search_cache
	`).Scan(&stats.TotalSizeBytes)
	if err != nil {
		return nil, err
	}

	return &stats, nil
}

// SearchCacheHistory searches the cache history for a query pattern.
func (s *CacheStore) SearchCacheHistory(ctx context.Context, queryPattern string, limit int) ([]*search.CacheEntry, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT hash, query, category, options_json, results_json, version, created_at
		FROM search_cache
		WHERE query LIKE ?
		ORDER BY created_at DESC
		LIMIT ?
	`, "%"+queryPattern+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*search.CacheEntry
	for rows.Next() {
		var entry search.CacheEntry
		if err := rows.Scan(
			&entry.Hash,
			&entry.Query,
			&entry.Category,
			&entry.OptionsJSON,
			&entry.ResultsJSON,
			&entry.Version,
			&entry.CreatedAt,
		); err != nil {
			return nil, err
		}
		entries = append(entries, &entry)
	}

	return entries, rows.Err()
}

// MarshalOptions serializes search options to JSON.
func MarshalOptions(opts interface{}) string {
	data, _ := json.Marshal(opts)
	return string(data)
}
