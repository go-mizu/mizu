package duckdb

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/go-mizu/blueprints/finewiki/feature/search"

	_ "github.com/duckdb/duckdb-go/v2"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		db      *sql.DB
		wantErr bool
	}{
		{
			name:    "nil db returns error",
			db:      nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.db)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStore_Ensure(t *testing.T) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("failed to open duckdb: %v", err)
	}
	defer db.Close()

	store, err := New(db)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	ctx := context.Background()
	cfg := Config{
		ParquetGlob: "testdata/*.parquet",
		EnableFTS:   false,
	}
	opts := EnsureOptions{
		SeedIfEmpty: false, // Don't try to seed from non-existent parquet
		BuildIndex:  true,
		BuildFTS:    false,
	}

	if err := store.Ensure(ctx, cfg, opts); err != nil {
		t.Errorf("Ensure() error = %v", err)
	}

	// Verify tables were created
	var count int
	err = db.QueryRowContext(ctx, "SELECT count(*) FROM information_schema.tables WHERE table_name = 'titles'").Scan(&count)
	if err != nil {
		t.Errorf("failed to check tables: %v", err)
	}
	if count != 1 {
		t.Errorf("titles table not created, count = %d", count)
	}
}

func TestStore_Search_Empty(t *testing.T) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("failed to open duckdb: %v", err)
	}
	defer db.Close()

	store, err := New(db)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	ctx := context.Background()
	if err := store.Ensure(ctx, Config{}, EnsureOptions{}); err != nil {
		t.Fatalf("failed to ensure: %v", err)
	}

	// Search with empty text should return empty
	results, err := store.Search(ctx, search.Query{Text: ""})
	if err != nil {
		t.Errorf("Search() error = %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Search() returned %d results, want 0", len(results))
	}
}

func TestStore_Search_WithData(t *testing.T) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("failed to open duckdb: %v", err)
	}
	defer db.Close()

	store, err := New(db)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	ctx := context.Background()
	if err := store.Ensure(ctx, Config{}, EnsureOptions{}); err != nil {
		t.Fatalf("failed to ensure: %v", err)
	}

	// Insert test data
	_, err = db.ExecContext(ctx, `
		INSERT INTO titles (id, wikiname, in_language, title, title_lc) VALUES
		('enwiki/1', 'enwiki', 'en', 'Alan Turing', 'alan turing'),
		('enwiki/2', 'enwiki', 'en', 'Alan Kay', 'alan kay'),
		('enwiki/3', 'enwiki', 'en', 'Albert Einstein', 'albert einstein'),
		('dewiki/1', 'dewiki', 'de', 'Alan Turing', 'alan turing')
	`)
	if err != nil {
		t.Fatalf("failed to insert test data: %v", err)
	}

	tests := []struct {
		name      string
		query     search.Query
		wantCount int
		wantFirst string
	}{
		{
			name:      "exact match",
			query:     search.Query{Text: "alan turing", Limit: 10},
			wantCount: 2,
			wantFirst: "Alan Turing",
		},
		{
			name:      "prefix match",
			query:     search.Query{Text: "alan", Limit: 10},
			wantCount: 3,
		},
		{
			name:      "case insensitive",
			query:     search.Query{Text: "ALAN", Limit: 10},
			wantCount: 3,
		},
		{
			name:      "filter by wiki",
			query:     search.Query{Text: "alan", WikiName: "enwiki", Limit: 10},
			wantCount: 2,
		},
		{
			name:      "filter by language",
			query:     search.Query{Text: "alan", InLanguage: "de", Limit: 10},
			wantCount: 1,
		},
		{
			name:      "no match",
			query:     search.Query{Text: "xyz", Limit: 10},
			wantCount: 0,
		},
		{
			name:      "limit enforced",
			query:     search.Query{Text: "alan", Limit: 1},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := store.Search(ctx, tt.query)
			if err != nil {
				t.Errorf("Search() error = %v", err)
				return
			}
			if len(results) != tt.wantCount {
				t.Errorf("Search() returned %d results, want %d", len(results), tt.wantCount)
			}
			if tt.wantFirst != "" && len(results) > 0 && results[0].Title != tt.wantFirst {
				t.Errorf("first result = %q, want %q", results[0].Title, tt.wantFirst)
			}
		})
	}
}

func TestStore_GetByID_NoGlob(t *testing.T) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("failed to open duckdb: %v", err)
	}
	defer db.Close()

	store, err := New(db)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	ctx := context.Background()
	if err := store.Ensure(ctx, Config{}, EnsureOptions{}); err != nil {
		t.Fatalf("failed to ensure: %v", err)
	}

	// Without parquet glob, GetByID should fail
	_, err = store.GetByID(ctx, "test/1")
	if err == nil {
		t.Error("GetByID() expected error without parquet glob")
	}
}

func TestStore_Stats(t *testing.T) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("failed to open duckdb: %v", err)
	}
	defer db.Close()

	store, err := New(db)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	ctx := context.Background()
	if err := store.Ensure(ctx, Config{}, EnsureOptions{}); err != nil {
		t.Fatalf("failed to ensure: %v", err)
	}

	// Insert test data
	_, err = db.ExecContext(ctx, `
		INSERT INTO titles (id, wikiname, in_language, title, title_lc) VALUES
		('enwiki/1', 'enwiki', 'en', 'Test 1', 'test 1'),
		('enwiki/2', 'enwiki', 'en', 'Test 2', 'test 2'),
		('dewiki/1', 'dewiki', 'de', 'Test 3', 'test 3')
	`)
	if err != nil {
		t.Fatalf("failed to insert test data: %v", err)
	}

	stats, err := store.Stats(ctx)
	if err != nil {
		t.Errorf("Stats() error = %v", err)
	}

	if stats["titles"] != int64(3) {
		t.Errorf("stats[titles] = %v, want 3", stats["titles"])
	}

	wikis, ok := stats["wikis"].(map[string]int64)
	if !ok {
		t.Errorf("stats[wikis] not a map")
	} else {
		if wikis["enwiki"] != 2 {
			t.Errorf("stats[wikis][enwiki] = %d, want 2", wikis["enwiki"])
		}
		if wikis["dewiki"] != 1 {
			t.Errorf("stats[wikis][dewiki] = %d, want 1", wikis["dewiki"])
		}
	}
}

func TestImportParquet_Empty(t *testing.T) {
	_, err := ImportParquet(context.Background(), "", "")
	if err == nil {
		t.Error("ImportParquet() expected error for empty src")
	}

	_, err = ImportParquet(context.Background(), "test.parquet", "")
	if err == nil {
		t.Error("ImportParquet() expected error for empty dir")
	}
}

func TestImportParquet_NonExistent(t *testing.T) {
	dir := t.TempDir()
	_, err := ImportParquet(context.Background(), "/nonexistent/file.parquet", dir)
	if err == nil {
		t.Error("ImportParquet() expected error for non-existent file")
	}
}

func TestListParquet_Empty(t *testing.T) {
	_, err := ListParquet(context.Background(), "")
	if err == nil {
		t.Error("ListParquet() expected error for empty dataset")
	}
}

func TestIsHTTP(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"http://example.com", true},
		{"https://example.com", true},
		{"HTTP://EXAMPLE.COM", true},
		{"HTTPS://EXAMPLE.COM", true},
		{"ftp://example.com", false},
		{"/path/to/file", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := isHTTP(tt.input); got != tt.want {
				t.Errorf("isHTTP(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestFileNameFromURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://example.com/file.parquet", "file.parquet"},
		{"https://example.com/path/to/file.parquet", "file.parquet"},
		{"https://example.com/file.parquet?query=1", "file.parquet"},
		{"https://example.com/", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := fileNameFromURL(tt.input); got != tt.want {
				t.Errorf("fileNameFromURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()

	// Create source file
	src := dir + "/source.txt"
	if err := os.WriteFile(src, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	// Copy file
	dst := dir + "/dest.txt"
	if err := copyFile(src, dst); err != nil {
		t.Errorf("copyFile() error = %v", err)
	}

	// Verify content
	content, err := os.ReadFile(dst)
	if err != nil {
		t.Errorf("failed to read dest file: %v", err)
	}
	if string(content) != "test content" {
		t.Errorf("dest content = %q, want %q", string(content), "test content")
	}
}
