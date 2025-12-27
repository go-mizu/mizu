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

	// Fix FK constraints for tables referencing issues (add ON UPDATE CASCADE ON DELETE CASCADE)
	// DuckDB doesn't support ALTER CONSTRAINT, so we need to recreate tables
	if err := s.migrateFKConstraints(ctx); err != nil {
		return fmt.Errorf("migrate FK constraints: %w", err)
	}

	return nil
}

// migrateFKConstraints removes FK constraints from tables that reference issues.
// This fixes the DuckDB issue where UPDATE operations fail due to FK checks.
// DuckDB internally treats some UPDATEs as DELETE+INSERT, triggering FK violations.
func (s *Store) migrateFKConstraints(ctx context.Context) error {
	// Check if migration is already done
	var markerExists int
	row := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.tables WHERE table_name = '_fk_migration_v1'`)
	if err := row.Scan(&markerExists); err == nil && markerExists > 0 {
		return nil // Already migrated
	}

	// Check if issue_assignees has FK constraints that need removal
	var hasFKConstraint int
	row = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM duckdb_constraints()
		WHERE table_name = 'issue_assignees'
		AND constraint_type = 'FOREIGN KEY'
	`)
	if err := row.Scan(&hasFKConstraint); err != nil || hasFKConstraint == 0 {
		// No FK constraints or can't check - create marker and return
		s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS _fk_migration_v1 (done BOOLEAN)`)
		return nil
	}

	// Migrate issue_assignees (remove FK constraints)
	if _, err := s.db.ExecContext(ctx, `
		CREATE TABLE issue_assignees_new (
			issue_id VARCHAR NOT NULL,
			user_id  VARCHAR NOT NULL,
			PRIMARY KEY (issue_id, user_id)
		)
	`); err == nil {
		s.db.ExecContext(ctx, `INSERT INTO issue_assignees_new SELECT * FROM issue_assignees`)
		s.db.ExecContext(ctx, `DROP TABLE issue_assignees`)
		s.db.ExecContext(ctx, `ALTER TABLE issue_assignees_new RENAME TO issue_assignees`)
	}

	// Migrate comments (remove FK constraints)
	if _, err := s.db.ExecContext(ctx, `
		CREATE TABLE comments_new (
			id         VARCHAR PRIMARY KEY,
			issue_id   VARCHAR NOT NULL,
			author_id  VARCHAR NOT NULL,
			content    VARCHAR NOT NULL,
			edited_at  TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`); err == nil {
		s.db.ExecContext(ctx, `INSERT INTO comments_new SELECT * FROM comments`)
		s.db.ExecContext(ctx, `DROP TABLE comments`)
		s.db.ExecContext(ctx, `ALTER TABLE comments_new RENAME TO comments`)
	}

	// Migrate field_values (remove FK constraints)
	if _, err := s.db.ExecContext(ctx, `
		CREATE TABLE field_values_new (
			issue_id   VARCHAR NOT NULL,
			field_id   VARCHAR NOT NULL,
			value_text VARCHAR,
			value_num  DOUBLE,
			value_bool BOOLEAN,
			value_date DATE,
			value_ts   TIMESTAMP,
			value_ref  VARCHAR,
			value_json VARCHAR,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (issue_id, field_id)
		)
	`); err == nil {
		s.db.ExecContext(ctx, `INSERT INTO field_values_new SELECT * FROM field_values`)
		s.db.ExecContext(ctx, `DROP TABLE field_values`)
		s.db.ExecContext(ctx, `ALTER TABLE field_values_new RENAME TO field_values`)
	}

	// Create marker to indicate migration is complete
	s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS _fk_migration_v1 (done BOOLEAN)`)

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
