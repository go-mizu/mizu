// Package duckdb provides a DuckDB-backed store for FineWiki.
// It reads Parquet files directly and maintains a local title index for fast search.
package duckdb

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"strings"

	"github.com/go-mizu/blueprints/finewiki/feature/search"
	"github.com/go-mizu/blueprints/finewiki/feature/view"
)

//go:embed schema.sql
var schemaDDL string

//go:embed seed.sql
var seedSQL string

// Config holds configuration for the DuckDB store.
type Config struct {
	// ParquetGlob is the glob pattern or path to Parquet files.
	ParquetGlob string

	// EnableFTS enables DuckDB FTS extension for fuzzy search fallback.
	EnableFTS bool
}

// EnsureOptions controls what to build during Ensure.
type EnsureOptions struct {
	// SeedIfEmpty extracts titles from Parquet if titles table is empty.
	SeedIfEmpty bool

	// BuildIndex creates indexes on titles table.
	BuildIndex bool

	// BuildFTS creates FTS index on titles table.
	BuildFTS bool
}

// Store implements search.Store and view.Store using DuckDB.
type Store struct {
	db   *sql.DB
	glob string
	fts  bool
}

// New creates a new Store with the given database connection.
func New(db *sql.DB) (*Store, error) {
	if db == nil {
		return nil, errors.New("duckdb: nil db")
	}
	return &Store{db: db}, nil
}

// Ensure initializes the database schema and optionally seeds data.
func (s *Store) Ensure(ctx context.Context, cfg Config, opts EnsureOptions) error {
	s.glob = cfg.ParquetGlob
	s.fts = cfg.EnableFTS

	// Create schema
	if _, err := s.db.ExecContext(ctx, schemaDDL); err != nil {
		return fmt.Errorf("duckdb: schema: %w", err)
	}

	// Seed titles if empty
	if opts.SeedIfEmpty {
		var count int64
		row := s.db.QueryRowContext(ctx, "SELECT count(*) FROM titles")
		if err := row.Scan(&count); err != nil {
			return fmt.Errorf("duckdb: count titles: %w", err)
		}

		if count == 0 && s.glob != "" {
			seed := strings.ReplaceAll(seedSQL, "__PARQUET_GLOB__", s.glob)
			if _, err := s.db.ExecContext(ctx, seed); err != nil {
				return fmt.Errorf("duckdb: seed: %w", err)
			}
		}
	}

	// Build indexes
	if opts.BuildIndex {
		indexes := []string{
			"CREATE INDEX IF NOT EXISTS idx_titles_title_lc ON titles(title_lc)",
			"CREATE INDEX IF NOT EXISTS idx_titles_wikiname ON titles(wikiname)",
			"CREATE INDEX IF NOT EXISTS idx_titles_lang ON titles(in_language)",
		}
		for _, ddl := range indexes {
			if _, err := s.db.ExecContext(ctx, ddl); err != nil {
				return fmt.Errorf("duckdb: index: %w", err)
			}
		}
	}

	// Build FTS index
	if opts.BuildFTS && s.fts {
		ftsSetup := []string{
			"INSTALL fts",
			"LOAD fts",
			`PRAGMA create_fts_index('titles', 'id', 'title', overwrite=1)`,
		}
		for _, stmt := range ftsSetup {
			if _, err := s.db.ExecContext(ctx, stmt); err != nil {
				// FTS may fail on some systems; log but don't fail
				continue
			}
		}
	}

	return nil
}

// Search implements search.Store.
func (s *Store) Search(ctx context.Context, q search.Query) ([]search.Result, error) {
	if q.Text == "" {
		return []search.Result{}, nil
	}

	textLower := strings.ToLower(q.Text)

	// Build WHERE clause
	var conditions []string
	var args []any
	argIdx := 1

	// Add text condition (exact or prefix)
	conditions = append(conditions, fmt.Sprintf("(title_lc = $%d OR title_lc LIKE $%d)", argIdx, argIdx+1))
	args = append(args, textLower, textLower+"%")
	argIdx += 2

	// Add optional filters
	if q.WikiName != "" {
		conditions = append(conditions, fmt.Sprintf("wikiname = $%d", argIdx))
		args = append(args, q.WikiName)
		argIdx++
	}
	if q.InLanguage != "" {
		conditions = append(conditions, fmt.Sprintf("in_language = $%d", argIdx))
		args = append(args, q.InLanguage)
		argIdx++
	}

	// Build query with ORDER BY to prioritize exact matches
	query := fmt.Sprintf(`
		SELECT id, wikiname, in_language, title
		FROM titles
		WHERE %s
		ORDER BY
			CASE WHEN title_lc = $1 THEN 0 ELSE 1 END,
			length(title),
			title
		LIMIT $%d
	`, strings.Join(conditions, " AND "), argIdx)
	args = append(args, q.Limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("duckdb: search: %w", err)
	}
	defer rows.Close()

	var results []search.Result
	for rows.Next() {
		var r search.Result
		if err := rows.Scan(&r.ID, &r.WikiName, &r.InLanguage, &r.Title); err != nil {
			return nil, fmt.Errorf("duckdb: scan: %w", err)
		}
		results = append(results, r)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("duckdb: rows: %w", err)
	}

	// If no results and FTS enabled, try FTS fallback
	if len(results) == 0 && q.EnableFTS && s.fts {
		return s.searchFTS(ctx, q)
	}

	return results, nil
}

// searchFTS performs a full-text search using DuckDB FTS extension.
func (s *Store) searchFTS(ctx context.Context, q search.Query) ([]search.Result, error) {
	var conditions []string
	var args []any
	argIdx := 1

	// FTS match
	conditions = append(conditions, fmt.Sprintf("fts_main_titles.match_bm25(id, $%d) IS NOT NULL", argIdx))
	args = append(args, q.Text)
	argIdx++

	if q.WikiName != "" {
		conditions = append(conditions, fmt.Sprintf("wikiname = $%d", argIdx))
		args = append(args, q.WikiName)
		argIdx++
	}
	if q.InLanguage != "" {
		conditions = append(conditions, fmt.Sprintf("in_language = $%d", argIdx))
		args = append(args, q.InLanguage)
		argIdx++
	}

	query := fmt.Sprintf(`
		SELECT id, wikiname, in_language, title
		FROM titles
		WHERE %s
		ORDER BY fts_main_titles.match_bm25(id, $1) DESC
		LIMIT $%d
	`, strings.Join(conditions, " AND "), argIdx)
	args = append(args, q.Limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		// FTS may not be available; return empty
		return []search.Result{}, nil
	}
	defer rows.Close()

	var results []search.Result
	for rows.Next() {
		var r search.Result
		if err := rows.Scan(&r.ID, &r.WikiName, &r.InLanguage, &r.Title); err != nil {
			return nil, fmt.Errorf("duckdb: fts scan: %w", err)
		}
		results = append(results, r)
	}

	return results, rows.Err()
}

// GetByID implements view.Store.
func (s *Store) GetByID(ctx context.Context, id string) (*view.Page, error) {
	if s.glob == "" {
		return nil, errors.New("duckdb: parquet glob not configured")
	}

	query := fmt.Sprintf(`
		SELECT
			id,
			wikiname,
			page_id,
			title,
			url,
			COALESCE(date_modified, '') as date_modified,
			in_language,
			COALESCE(text, '') as text,
			COALESCE(wikidata_id, '') as wikidata_id,
			COALESCE(bytes_html, 0) as bytes_html,
			COALESCE(has_math, false) as has_math,
			COALESCE(wikitext, '') as wikitext,
			COALESCE(version, '') as version,
			COALESCE(infoboxes::VARCHAR, '[]') as infoboxes
		FROM read_parquet('%s')
		WHERE id = $1
		LIMIT 1
	`, s.glob)

	row := s.db.QueryRowContext(ctx, query, id)
	return s.scanPage(row)
}

// GetByTitle implements view.Store.
func (s *Store) GetByTitle(ctx context.Context, wikiname, title string) (*view.Page, error) {
	if s.glob == "" {
		return nil, errors.New("duckdb: parquet glob not configured")
	}

	// First try exact match, then case-insensitive match
	query := fmt.Sprintf(`
		SELECT
			id,
			wikiname,
			page_id,
			title,
			url,
			COALESCE(date_modified, '') as date_modified,
			in_language,
			COALESCE(text, '') as text,
			COALESCE(wikidata_id, '') as wikidata_id,
			COALESCE(bytes_html, 0) as bytes_html,
			COALESCE(has_math, false) as has_math,
			COALESCE(wikitext, '') as wikitext,
			COALESCE(version, '') as version,
			COALESCE(infoboxes::VARCHAR, '[]') as infoboxes
		FROM read_parquet('%s')
		WHERE wikiname = $1 AND LOWER(title) = LOWER($2)
		LIMIT 1
	`, s.glob)

	row := s.db.QueryRowContext(ctx, query, wikiname, title)
	return s.scanPage(row)
}

// scanPage scans a row into a Page struct.
func (s *Store) scanPage(row *sql.Row) (*view.Page, error) {
	var p view.Page
	err := row.Scan(
		&p.ID,
		&p.WikiName,
		&p.PageID,
		&p.Title,
		&p.URL,
		&p.DateModified,
		&p.InLanguage,
		&p.Text,
		&p.WikidataID,
		&p.BytesHTML,
		&p.HasMath,
		&p.WikiText,
		&p.Version,
		&p.InfoboxesJSON,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("page not found")
		}
		return nil, fmt.Errorf("duckdb: scan page: %w", err)
	}
	return &p, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// Stats returns basic statistics about the store.
func (s *Store) Stats(ctx context.Context) (map[string]any, error) {
	stats := make(map[string]any)

	// Count titles
	var titleCount int64
	row := s.db.QueryRowContext(ctx, "SELECT count(*) FROM titles")
	if err := row.Scan(&titleCount); err == nil {
		stats["titles"] = titleCount
	}

	// Count wikis
	rows, err := s.db.QueryContext(ctx, "SELECT wikiname, count(*) FROM titles GROUP BY wikiname")
	if err == nil {
		defer rows.Close()
		wikis := make(map[string]int64)
		for rows.Next() {
			var wiki string
			var count int64
			if rows.Scan(&wiki, &count) == nil {
				wikis[wiki] = count
			}
		}
		stats["wikis"] = wikis
	}

	// Get seeded_at
	var seededAt string
	row = s.db.QueryRowContext(ctx, "SELECT v FROM meta WHERE k = 'seeded_at'")
	if row.Scan(&seededAt) == nil {
		stats["seeded_at"] = seededAt
	}

	return stats, nil
}
