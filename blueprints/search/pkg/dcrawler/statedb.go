package dcrawler

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/duckdb/duckdb-go/v2"
)

// StateDB persists crawl state for restart/resume.
// Single DuckDB file at {domainDir}/state.duckdb.
type StateDB struct {
	db   *sql.DB
	path string
}

// OpenStateDB opens or creates the state database.
func OpenStateDB(domainDir string) (*StateDB, error) {
	if err := os.MkdirAll(domainDir, 0755); err != nil {
		return nil, err
	}
	path := filepath.Join(domainDir, "state.duckdb")
	db, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, fmt.Errorf("opening state db: %w", err)
	}
	sdb := &StateDB{db: db, path: path}
	if err := sdb.init(); err != nil {
		db.Close()
		return nil, err
	}
	return sdb, nil
}

func (s *StateDB) init() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS crawl_meta (
			key VARCHAR PRIMARY KEY,
			value VARCHAR
		)`,
		`CREATE TABLE IF NOT EXISTS frontier (
			url VARCHAR NOT NULL,
			depth INTEGER NOT NULL
		)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("init state db: %w", err)
		}
	}
	return nil
}

// SetMeta stores a key-value pair.
func (s *StateDB) SetMeta(key, value string) {
	s.db.Exec("INSERT OR REPLACE INTO crawl_meta (key, value) VALUES (?, ?)", key, value)
}

// GetMeta retrieves a value by key.
func (s *StateDB) GetMeta(key string) string {
	var val string
	row := s.db.QueryRow("SELECT value FROM crawl_meta WHERE key = ?", key)
	if row.Scan(&val) != nil {
		return ""
	}
	return val
}

// SaveFrontier saves pending frontier URLs for restart.
// Clears previous frontier state first.
func (s *StateDB) SaveFrontier(items []CrawlItem) error {
	s.db.Exec("DELETE FROM frontier")
	if len(items) == 0 {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT INTO frontier (url, depth) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, item := range items {
		stmt.Exec(item.URL, item.Depth)
	}
	return tx.Commit()
}

// LoadFrontier reads saved frontier URLs.
func (s *StateDB) LoadFrontier() ([]CrawlItem, error) {
	rows, err := s.db.Query("SELECT url, depth FROM frontier")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []CrawlItem
	for rows.Next() {
		var item CrawlItem
		if err := rows.Scan(&item.URL, &item.Depth); err == nil {
			items = append(items, item)
		}
	}
	return items, nil
}

// FrontierCount returns the number of saved frontier URLs.
func (s *StateDB) FrontierCount() int {
	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM frontier").Scan(&count)
	return count
}

// Close closes the state database.
func (s *StateDB) Close() error {
	return s.db.Close()
}
