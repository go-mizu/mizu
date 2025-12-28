// Package duckdb provides a DuckDB-backed store for GitHome.
package duckdb

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/base64"
	"errors"
	"fmt"
	"time"
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

// Ensure initializes the database schema.
func (s *Store) Ensure(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, schemaDDL); err != nil {
		return fmt.Errorf("duckdb: schema: %w", err)
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

	tables := []string{
		"users", "organizations", "repositories", "teams",
		"issues", "labels", "milestones", "issue_comments",
		"pull_requests", "pr_reviews", "stars", "watches",
		"collaborators", "reactions", "releases", "events",
		"notifications", "webhooks",
	}
	for _, table := range tables {
		var count int64
		row := s.db.QueryRowContext(ctx, fmt.Sprintf("SELECT count(*) FROM %s", table))
		if err := row.Scan(&count); err == nil {
			stats[table] = count
		}
	}

	return stats, nil
}

// Helper functions

// generateNodeID creates a GitHub-compatible node ID.
func generateNodeID(prefix string, id int64) string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s_%d", prefix, id)))
}

// nullString converts a string to sql.NullString.
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

// nullStringPtr converts a *string to sql.NullString.
func nullStringPtr(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

// nullInt64 converts an int64 to sql.NullInt64.
func nullInt64(i int64) sql.NullInt64 {
	if i == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: i, Valid: true}
}

// nullInt64Ptr converts a *int64 to sql.NullInt64.
func nullInt64Ptr(i *int64) sql.NullInt64 {
	if i == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *i, Valid: true}
}

// nullBoolPtr converts a *bool to sql.NullBool.
func nullBoolPtr(b *bool) sql.NullBool {
	if b == nil {
		return sql.NullBool{}
	}
	return sql.NullBool{Bool: *b, Valid: true}
}

// nullTime converts a *time.Time to sql.NullTime.
func nullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

// paginationParams returns offset and limit from page and perPage.
func paginationParams(page, perPage int) (offset, limit int) {
	if perPage <= 0 {
		perPage = 30
	}
	if perPage > 100 {
		perPage = 100
	}
	if page <= 0 {
		page = 1
	}
	return (page - 1) * perPage, perPage
}

// applyPagination adds LIMIT and OFFSET to a query.
func applyPagination(query string, page, perPage int) string {
	offset, limit := paginationParams(page, perPage)
	return fmt.Sprintf("%s LIMIT %d OFFSET %d", query, limit, offset)
}
