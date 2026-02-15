package perplexity

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/duckdb/duckdb-go/v2"
)

// DB wraps a DuckDB database for storing Perplexity search results.
type DB struct {
	db   *sql.DB
	path string
}

// OpenDB opens or creates a DuckDB database at the given path.
func OpenDB(path string) (*DB, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	db, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, fmt.Errorf("open duckdb %s: %w", path, err)
	}

	d := &DB{db: db, path: path}
	if err := d.initSchema(); err != nil {
		db.Close()
		return nil, err
	}

	return d, nil
}

func (d *DB) initSchema() error {
	_, err := d.db.Exec(`
		CREATE TABLE IF NOT EXISTS searches (
			id           INTEGER PRIMARY KEY DEFAULT nextval('search_seq'),
			query        TEXT NOT NULL,
			answer       TEXT,
			citations    TEXT,
			web_results  TEXT,
			chunks       TEXT,
			media_items  TEXT,
			related      TEXT,
			backend_uuid TEXT,
			mode         TEXT,
			model        TEXT,
			source       TEXT DEFAULT 'sse',
			searched_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		// Sequence might not exist, create it first
		d.db.Exec(`CREATE SEQUENCE IF NOT EXISTS search_seq START 1`)
		_, err = d.db.Exec(`
			CREATE TABLE IF NOT EXISTS searches (
				id           INTEGER PRIMARY KEY DEFAULT nextval('search_seq'),
				query        TEXT NOT NULL,
				answer       TEXT,
				citations    TEXT,
				web_results  TEXT,
				chunks       TEXT,
				media_items  TEXT,
				related      TEXT,
				backend_uuid TEXT,
				mode         TEXT,
				model        TEXT,
				source       TEXT DEFAULT 'sse',
				searched_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)
		`)
		if err != nil {
			return fmt.Errorf("init schema: %w", err)
		}
	}
	return nil
}

// SaveSearch stores a search result.
func (d *DB) SaveSearch(r *SearchResult) error {
	citationsJSON, _ := json.Marshal(r.Citations)
	webResultsJSON, _ := json.Marshal(r.WebResults)
	chunksJSON, _ := json.Marshal(r.Chunks)
	mediaJSON, _ := json.Marshal(r.MediaItems)
	relatedJSON, _ := json.Marshal(r.RelatedQ)

	_, err := d.db.Exec(`
		INSERT INTO searches (query, answer, citations, web_results, chunks, media_items, related, backend_uuid, mode, model, source, searched_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		r.Query, r.Answer,
		string(citationsJSON), string(webResultsJSON),
		string(chunksJSON), string(mediaJSON),
		string(relatedJSON), r.BackendUUID,
		r.Mode, r.Model, r.Source, r.SearchedAt,
	)
	return err
}

// Count returns the total number of stored searches.
func (d *DB) Count() (int, error) {
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM searches").Scan(&count)
	return count, err
}

// RecentSearches returns the N most recent searches.
func (d *DB) RecentSearches(limit int) ([]SearchResult, error) {
	rows, err := d.db.Query(`
		SELECT query, answer, citations, web_results, related, backend_uuid, mode, model, source, searched_at
		FROM searches
		ORDER BY searched_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		var citationsStr, webResultsStr, relatedStr sql.NullString
		err := rows.Scan(
			&r.Query, &r.Answer, &citationsStr, &webResultsStr,
			&relatedStr, &r.BackendUUID, &r.Mode, &r.Model, &r.Source, &r.SearchedAt,
		)
		if err != nil {
			continue
		}
		if citationsStr.Valid {
			json.Unmarshal([]byte(citationsStr.String), &r.Citations)
		}
		if webResultsStr.Valid {
			json.Unmarshal([]byte(webResultsStr.String), &r.WebResults)
		}
		if relatedStr.Valid {
			json.Unmarshal([]byte(relatedStr.String), &r.RelatedQ)
		}
		results = append(results, r)
	}
	return results, nil
}

// Close closes the database.
func (d *DB) Close() error {
	return d.db.Close()
}
