// Package duckdb provides database access using DuckDB.
package duckdb

import (
	"context"
	"database/sql"
	_ "embed"

	_ "github.com/duckdb/duckdb-go/v2"
)

//go:embed schema.sql
var schema string

// Store is the main database store.
type Store struct {
	db *sql.DB
}

// New creates a new Store.
func New(db *sql.DB) (*Store, error) {
	return &Store{db: db}, nil
}

// Ensure creates the database schema if it doesn't exist.
func (s *Store) Ensure(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, schema)
	return err
}

// DB returns the underlying database connection.
func (s *Store) DB() *sql.DB {
	return s.db
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// Open opens a new database connection.
func Open(path string) (*sql.DB, error) {
	return sql.Open("duckdb", path)
}
