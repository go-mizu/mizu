// Package duckdb provides a DuckDB-backed store for Kanban.
package duckdb

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
)

//go:embed schema.sql
var schemaDDL string

// Store implements the data access layer using DuckDB.
type Store struct {
	db *sql.DB
}

// New creates a new Store with the given database connection.
func New(db *sql.DB) (*Store, error) {
	if db == nil {
		return nil, errors.New("duckdb: nil db")
	}
	return &Store{db: db}, nil
}

// DB returns the underlying database connection.
func (s *Store) DB() *sql.DB {
	return s.db
}

// Ensure initializes the database schema and runs migrations.
func (s *Store) Ensure(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, schemaDDL); err != nil {
		return fmt.Errorf("duckdb: schema: %w", err)
	}

	// Run migrations for existing databases
	if err := s.migrate(ctx); err != nil {
		return fmt.Errorf("duckdb: migrate: %w", err)
	}

	return nil
}

// migrate adds missing columns to existing tables.
func (s *Store) migrate(ctx context.Context) error {
	// Add missing columns to issues table (for databases created before these columns existed)
	migrations := []string{
		"ALTER TABLE issues ADD COLUMN IF NOT EXISTS description VARCHAR DEFAULT ''",
		"ALTER TABLE issues ADD COLUMN IF NOT EXISTS due_date DATE",
		"ALTER TABLE issues ADD COLUMN IF NOT EXISTS start_date DATE",
		"ALTER TABLE issues ADD COLUMN IF NOT EXISTS end_date DATE",
		"ALTER TABLE issues ADD COLUMN IF NOT EXISTS priority INTEGER NOT NULL DEFAULT 0",
	}

	for _, m := range migrations {
		if _, err := s.db.ExecContext(ctx, m); err != nil {
			// Ignore errors (column might already exist in a different form)
			continue
		}
	}

	return nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// Exec executes a query without returning rows.
func (s *Store) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return s.db.ExecContext(ctx, query, args...)
}

// Query executes a query that returns rows.
func (s *Store) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return s.db.QueryContext(ctx, query, args...)
}

// QueryRow executes a query that returns a single row.
func (s *Store) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return s.db.QueryRowContext(ctx, query, args...)
}

// Stats returns basic statistics about the store.
func (s *Store) Stats(ctx context.Context) (map[string]any, error) {
	stats := make(map[string]any)

	tables := []string{"users", "workspaces", "teams", "projects", "columns", "cycles", "issues", "comments", "fields", "field_values"}
	for _, table := range tables {
		var count int64
		row := s.db.QueryRowContext(ctx, fmt.Sprintf("SELECT count(*) FROM %s", table))
		if err := row.Scan(&count); err == nil {
			stats[table] = count
		}
	}

	return stats, nil
}
