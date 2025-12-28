package duckdb

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"

	_ "github.com/marcboeker/go-duckdb"
)

//go:embed schema.sql
var schemaDDL string

// Store is the core DuckDB store with schema management
type Store struct {
	db *sql.DB
}

// New creates a new store
func New(db *sql.DB) (*Store, error) {
	return &Store{db: db}, nil
}

// DB returns the underlying database connection
func (s *Store) DB() *sql.DB {
	return s.db
}

// Ensure creates all tables and runs migrations
func (s *Store) Ensure(ctx context.Context) error {
	// Execute embedded schema
	if _, err := s.db.ExecContext(ctx, schemaDDL); err != nil {
		return fmt.Errorf("schema: %w", err)
	}

	// Run migrations
	if err := s.migrate(ctx); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	return nil
}

// migrate runs any necessary migrations
func (s *Store) migrate(ctx context.Context) error {
	// Add migration statements here as needed
	migrations := []string{
		// Example: "ALTER TABLE users ADD COLUMN IF NOT EXISTS new_field VARCHAR DEFAULT ''",
	}

	for _, m := range migrations {
		if _, err := s.db.ExecContext(ctx, m); err != nil {
			// Ignore "already exists" errors
			continue
		}
	}

	return nil
}

// Close closes the database connection
func (s *Store) Close() error {
	return s.db.Close()
}
