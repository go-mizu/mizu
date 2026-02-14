package factory

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/go-mizu/mizu/blueprints/book/store/sqlite"
	"github.com/go-mizu/mizu/blueprints/book/types"
)

func TestOpenAutoMigratesSQLiteToDuckDB(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmp := t.TempDir()
	sqlitePath := filepath.Join(tmp, "book.db")
	duckdbPath := filepath.Join(tmp, "book.duckdb")

	src, err := sqlite.New(sqlitePath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := src.Ensure(ctx); err != nil {
		t.Fatalf("ensure sqlite schema: %v", err)
	}
	book := &types.Book{
		Title:       "Factory Migration Book",
		AuthorNames: "Factory Author",
		ISBN13:      "9780671027032",
		Language:    "en",
	}
	if err := src.Book().Create(ctx, book); err != nil {
		t.Fatalf("create sqlite book: %v", err)
	}
	if err := src.Close(); err != nil {
		t.Fatalf("close sqlite: %v", err)
	}

	s, err := Open(ctx, duckdbPath)
	if err != nil {
		t.Fatalf("open factory store: %v", err)
	}
	defer s.Close()

	got, err := s.Book().GetByISBN(ctx, book.ISBN13)
	if err != nil {
		t.Fatalf("get migrated book: %v", err)
	}
	if got == nil {
		t.Fatal("book not migrated")
	}
	if got.Title != book.Title {
		t.Fatalf("title mismatch: got %q want %q", got.Title, book.Title)
	}
}
