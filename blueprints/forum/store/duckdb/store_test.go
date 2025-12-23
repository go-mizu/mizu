package duckdb

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/oklog/ulid/v2"
)

// setupTestStore creates an in-memory DuckDB store for testing.
func setupTestStore(t *testing.T) *Store {
	t.Helper()

	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("failed to open in-memory duckdb: %v", err)
	}

	// Initialize schema
	if _, err := db.ExecContext(context.Background(), schema); err != nil {
		db.Close()
		t.Fatalf("failed to initialize schema: %v", err)
	}

	store := &Store{
		db:            db,
		accounts:      NewAccountsStore(db),
		boards:        NewBoardsStore(db),
		threads:       NewThreadsStore(db),
		comments:      NewCommentsStore(db),
		votes:         NewVotesStore(db),
		bookmarks:     NewBookmarksStore(db),
		notifications: NewNotificationsStore(db),
	}

	t.Cleanup(func() {
		db.Close()
	})

	return store
}

// newTestID generates a new ULID for testing.
func newTestID() string {
	return ulid.Make().String()
}

// testTime returns a fixed time for testing.
func testTime() time.Time {
	return time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
}

// ptr returns a pointer to the given value.
func ptr[T any](v T) *T {
	return &v
}
