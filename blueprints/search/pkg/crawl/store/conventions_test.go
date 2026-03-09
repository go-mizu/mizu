package store_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"sync"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// TestDuckDBSingleWriterRule verifies that SetMaxOpenConns(1) on a write-mode
// DuckDB connection prevents "conflicting lock" errors.
func TestDuckDBSingleWriterRule(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.duckdb")

	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if _, err := db.ExecContext(context.Background(), `CREATE TABLE IF NOT EXISTS t (id INTEGER)`); err != nil {
		t.Fatal(err)
	}

	// Concurrent inserts should not deadlock or error with MaxOpenConns=1
	var wg sync.WaitGroup
	errs := make([]error, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, err := db.ExecContext(context.Background(), `INSERT INTO t VALUES (?)`, idx)
			errs[idx] = err
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("insert %d failed: %v", i, err)
		}
	}

	var count int
	db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM t`).Scan(&count)
	if count != 10 {
		t.Errorf("expected 10 rows, got %d", count)
	}
}

// TestDuckDBReadOnlyMultiConn verifies that read-only connections allow >1 concurrent connection.
func TestDuckDBReadOnlyMultiConn(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test_ro.duckdb")

	// Create with write connection first
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.ExecContext(context.Background(), `CREATE TABLE t (id INTEGER); INSERT INTO t VALUES (1),(2),(3)`); err != nil {
		db.Close()
		t.Fatal(err)
	}
	db.Close()

	// Open read-only with MaxOpenConns=2
	roDB, err := sql.Open("duckdb", path+"?access_mode=READ_ONLY")
	if err != nil {
		t.Fatal(err)
	}
	defer roDB.Close()
	roDB.SetMaxOpenConns(2)

	var wg sync.WaitGroup
	errs := make([]error, 4)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			var count int
			errs[idx] = roDB.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM t`).Scan(&count)
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Errorf("read query %d failed: %v", i, err)
		}
	}
}
