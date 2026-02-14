package factory

import (
	"context"
	"fmt"

	"github.com/go-mizu/mizu/blueprints/book/store"
	"github.com/go-mizu/mizu/blueprints/book/store/duckdb"
)

// Open opens the default store backend (DuckDB).
// If the target DuckDB file does not exist and a sibling legacy SQLite file exists,
// it performs a one-time migration before opening the DuckDB store.
func Open(ctx context.Context, dbPath string) (store.Store, error) {
	if sqlitePath, ok := duckdb.ShouldAutoMigrate(dbPath); ok {
		if err := duckdb.MigrateFromSQLite(ctx, sqlitePath, dbPath); err != nil {
			return nil, fmt.Errorf("auto-migrate sqlite to duckdb: %w", err)
		}
	}

	s, err := duckdb.New(dbPath)
	if err != nil {
		return nil, err
	}
	if err := s.Ensure(ctx); err != nil {
		s.Close() //nolint:errcheck
		return nil, fmt.Errorf("ensure schema: %w", err)
	}
	return s, nil
}
