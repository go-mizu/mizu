package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

// MigrateFromSQLite copies data from an existing SQLite DB into a DuckDB DB.
// It expects the DuckDB schema to already exist.
func MigrateFromSQLite(ctx context.Context, sqlitePath, duckdbPath string) error {
	src, err := sql.Open("sqlite", sqlitePath)
	if err != nil {
		return fmt.Errorf("open sqlite source: %w", err)
	}
	defer src.Close()

	dstStore, err := New(duckdbPath)
	if err != nil {
		return fmt.Errorf("open duckdb target: %w", err)
	}
	defer dstStore.Close()

	if err := dstStore.Ensure(ctx); err != nil {
		return fmt.Errorf("ensure duckdb schema: %w", err)
	}

	dst := dstStore.db
	tx, err := dst.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin duckdb tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	order := []string{
		"authors",
		"books",
		"shelves",
		"shelf_books",
		"reading_challenges",
		"book_lists",
		"book_list_items",
		"reviews",
		"review_comments",
		"reading_progress",
		"quotes",
		"feed",
	}

	for _, table := range order {
		cols, err := sqliteColumns(ctx, src, table)
		if err != nil {
			return fmt.Errorf("read source columns for %s: %w", table, err)
		}
		if len(cols) == 0 {
			continue
		}
		if err := copyTable(ctx, src, tx, table, cols); err != nil {
			return fmt.Errorf("copy table %s: %w", table, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit duckdb tx: %w", err)
	}
	return nil
}

func sqliteColumns(ctx context.Context, db *sql.DB, table string) ([]string, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []string
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return nil, err
		}
		cols = append(cols, name)
	}
	return cols, rows.Err()
}

func copyTable(ctx context.Context, src *sql.DB, dst *sql.Tx, table string, cols []string) error {
	colCSV := strings.Join(cols, ", ")
	query := fmt.Sprintf("SELECT %s FROM %s", colCSV, table)
	rows, err := src.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	ph := make([]string, len(cols))
	for i := range ph {
		ph[i] = "?"
	}
	insertSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, colCSV, strings.Join(ph, ", "))
	stmt, err := dst.PrepareContext(ctx, insertSQL)
	if err != nil {
		return err
	}
	defer stmt.Close()

	vals := make([]any, len(cols))
	ptrs := make([]any, len(cols))
	for i := range vals {
		ptrs[i] = &vals[i]
	}

	for rows.Next() {
		if err := rows.Scan(ptrs...); err != nil {
			return err
		}
		args := make([]any, len(vals))
		for i, v := range vals {
			switch x := v.(type) {
			case []byte:
				args[i] = string(x)
			default:
				args[i] = x
			}
		}
		if _, err := stmt.ExecContext(ctx, args...); err != nil {
			return err
		}
	}
	return rows.Err()
}

// DefaultSQLitePathFromDuckDB infers a legacy SQLite path from a DuckDB path.
func DefaultSQLitePathFromDuckDB(duckdbPath string) string {
	base := filepath.Base(duckdbPath)
	if strings.HasSuffix(base, ".duckdb") {
		return filepath.Join(filepath.Dir(duckdbPath), strings.TrimSuffix(base, ".duckdb")+".db")
	}
	return filepath.Join(filepath.Dir(duckdbPath), "book.db")
}

// ShouldAutoMigrate reports whether auto-migration should run.
func ShouldAutoMigrate(duckdbPath string) (string, bool) {
	if _, err := os.Stat(duckdbPath); err == nil {
		return "", false
	}
	sqlitePath := DefaultSQLitePathFromDuckDB(duckdbPath)
	if _, err := os.Stat(sqlitePath); err == nil {
		return sqlitePath, true
	}
	return "", false
}
