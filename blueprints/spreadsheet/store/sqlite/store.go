// Package sqlite provides a SQLite-backed store for Spreadsheet.
package sqlite

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
)

//go:embed schema.sql
var schemaDDL string

// Store implements the data access layer using SQLite.
type Store struct {
	db *sql.DB
}

// New creates a new Store with the given database connection.
func New(db *sql.DB) (*Store, error) {
	if db == nil {
		return nil, errors.New("sqlite: nil db")
	}
	return &Store{db: db}, nil
}

// DB returns the underlying database connection.
func (s *Store) DB() *sql.DB {
	return s.db
}

// Ensure initializes the database schema and configures SQLite for optimal performance.
func (s *Store) Ensure(ctx context.Context) error {
	// Configure SQLite for optimal spreadsheet workloads
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA cache_size=-64000",
		"PRAGMA temp_store=MEMORY",
		"PRAGMA mmap_size=268435456",
		"PRAGMA foreign_keys=ON",
	}

	for _, pragma := range pragmas {
		if _, err := s.db.ExecContext(ctx, pragma); err != nil {
			// Ignore pragma errors for in-memory databases
			continue
		}
	}

	if _, err := s.db.ExecContext(ctx, schemaDDL); err != nil {
		return fmt.Errorf("sqlite: schema: %w", err)
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

// Stats returns basic statistics about the store.
func (s *Store) Stats(ctx context.Context) (map[string]any, error) {
	stats := make(map[string]any)

	tables := []string{"users", "workbooks", "sheets", "cells", "charts", "comments", "shares"}
	for _, table := range tables {
		var count int64
		row := s.db.QueryRowContext(ctx, fmt.Sprintf("SELECT count(*) FROM %s", table))
		if err := row.Scan(&count); err == nil {
			stats[table] = count
		}
	}

	return stats, nil
}
