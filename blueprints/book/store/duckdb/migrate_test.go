package duckdb

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/go-mizu/mizu/blueprints/book/store/sqlite"
	"github.com/go-mizu/mizu/blueprints/book/types"
)

func TestMigrateFromSQLite(t *testing.T) {
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
	if err := src.Shelf().SeedDefaults(ctx); err != nil {
		t.Fatalf("seed sqlite defaults: %v", err)
	}

	book := &types.Book{
		Title:       "Migration Test Book",
		AuthorNames: "Test Author",
		ISBN13:      "9780316769488",
		Language:    "en",
		PageCount:   321,
		Subjects:    []string{"fiction", "test"},
	}
	if err := src.Book().Create(ctx, book); err != nil {
		t.Fatalf("create sqlite book: %v", err)
	}

	readShelf, err := src.Shelf().GetBySlug(ctx, "read")
	if err != nil {
		t.Fatalf("get sqlite read shelf: %v", err)
	}
	if readShelf == nil {
		t.Fatal("read shelf not found")
	}
	if err := src.Shelf().AddBook(ctx, readShelf.ID, book.ID); err != nil {
		t.Fatalf("add sqlite shelf book: %v", err)
	}

	if err := src.Close(); err != nil {
		t.Fatalf("close sqlite: %v", err)
	}

	if err := MigrateFromSQLite(ctx, sqlitePath, duckdbPath); err != nil {
		t.Fatalf("migrate sqlite->duckdb: %v", err)
	}

	dst, err := New(duckdbPath)
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	defer dst.Close()
	if err := dst.Ensure(ctx); err != nil {
		t.Fatalf("ensure duckdb schema: %v", err)
	}

	got, err := dst.Book().GetByISBN(ctx, book.ISBN13)
	if err != nil {
		t.Fatalf("get migrated book: %v", err)
	}
	if got == nil {
		t.Fatal("migrated book missing")
	}
	if got.Title != book.Title {
		t.Fatalf("title mismatch: got %q want %q", got.Title, book.Title)
	}
	if got.Language != "en" {
		t.Fatalf("language mismatch: got %q", got.Language)
	}

	shelfBooks, total, err := dst.Shelf().GetBooks(ctx, readShelf.ID, "date_added", 1, 10)
	if err != nil {
		t.Fatalf("list migrated shelf books: %v", err)
	}
	if total != 1 || len(shelfBooks) != 1 {
		t.Fatalf("migrated shelf books mismatch: total=%d len=%d", total, len(shelfBooks))
	}
}

func TestShouldAutoMigrate(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	duckdbPath := filepath.Join(tmp, "book.duckdb")
	sqlitePath := filepath.Join(tmp, "book.db")

	if got, ok := ShouldAutoMigrate(duckdbPath); ok || got != "" {
		t.Fatalf("expected no migration without sqlite file, got ok=%v path=%q", ok, got)
	}

	src, err := sqlite.New(sqlitePath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := src.Close(); err != nil {
		t.Fatalf("close sqlite: %v", err)
	}

	got, ok := ShouldAutoMigrate(duckdbPath)
	if !ok {
		t.Fatal("expected migration to be required")
	}
	if got != sqlitePath {
		t.Fatalf("sqlite path mismatch: got %q want %q", got, sqlitePath)
	}
}
