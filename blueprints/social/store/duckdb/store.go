// Package duckdb provides DuckDB-based storage implementations.
package duckdb

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"path/filepath"
)

//go:embed schema.sql
var schema string

// Store is the core store for database operations.
type Store struct {
	db *sql.DB
}

// Open opens or creates a DuckDB database in the given directory.
func Open(dataDir string) (*Store, error) {
	dbPath := filepath.Join(dataDir, "social.db")
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open duckdb: %w", err)
	}

	store := &Store{db: db}
	if err := store.Ensure(context.Background()); err != nil {
		db.Close()
		return nil, err
	}

	return store, nil
}

// New creates a new store.
func New(db *sql.DB) (*Store, error) {
	return &Store{db: db}, nil
}

// Ensure ensures the schema exists.
func (s *Store) Ensure(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, schema)
	if err != nil {
		return fmt.Errorf("exec schema: %w", err)
	}
	return nil
}

// DB returns the underlying database connection.
func (s *Store) DB() *sql.DB {
	return s.db
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// Tx executes a function within a transaction.
func (s *Store) Tx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}
