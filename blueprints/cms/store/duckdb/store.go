// Package duckdb provides the DuckDB store implementation.
package duckdb

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/duckdb/duckdb-go/v2"
)

//go:embed schema.sql
var schema string

// Store is the main database store.
type Store struct {
	db *sql.DB

	// Sub-stores
	Collections *CollectionsStore
	Globals     *GlobalsStore
	Sessions    *SessionsStore
	Preferences *PreferencesStore
	Uploads     *UploadsStore
	Versions    *VersionsStore
}

// New creates a new Store with the given database path.
func New(dbPath string) (*Store, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Initialize schema
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}

	s := &Store{
		db:          db,
		Collections: NewCollectionsStore(db),
		Globals:     NewGlobalsStore(db),
		Sessions:    NewSessionsStore(db),
		Preferences: NewPreferencesStore(db),
		Uploads:     NewUploadsStore(db),
		Versions:    NewVersionsStore(db),
	}

	return s, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// DB returns the underlying database connection.
func (s *Store) DB() *sql.DB {
	return s.db
}

// Tx executes a function within a transaction.
func (s *Store) Tx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// CreateCollection creates a new collection table dynamically.
func (s *Store) CreateCollection(ctx context.Context, slug string, columns []ColumnDef) error {
	query := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id VARCHAR(26) PRIMARY KEY,
		%s
		_status VARCHAR(20) DEFAULT 'draft',
		_version INTEGER DEFAULT 1,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`, slug, buildColumns(columns))

	_, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("create collection table: %w", err)
	}

	return nil
}

// ColumnDef defines a column for dynamic table creation.
type ColumnDef struct {
	Name     string
	Type     string
	Nullable bool
	Unique   bool
	Index    bool
}

func buildColumns(columns []ColumnDef) string {
	var result string
	for _, col := range columns {
		nullable := "NULL"
		if !col.Nullable {
			nullable = ""
		}
		unique := ""
		if col.Unique {
			unique = "UNIQUE"
		}
		result += fmt.Sprintf("%s %s %s %s,\n\t\t", col.Name, col.Type, nullable, unique)
	}
	return result
}
