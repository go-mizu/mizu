package openlibrarydump

import (
	"compress/gzip"
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestExportParquetAndCleanup(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmp := t.TempDir()

	authorsPath := filepath.Join(tmp, "authors.txt.gz")
	worksPath := filepath.Join(tmp, "works.txt.gz")
	editionsPath := filepath.Join(tmp, "editions.txt.gz")
	dbPath := filepath.Join(tmp, "book.duckdb")
	parquetDir := filepath.Join(tmp, "parquet")

	writeGzipLines(t, authorsPath, []string{
		"/type/author\t/authors/OL23919A\t1\t2020-01-01T00:00:00.000000\t{\"name\":\"Jane Austen\",\"bio\":\"English novelist\",\"birth_date\":\"1775\",\"death_date\":\"1817\",\"work_count\":120}",
	})
	writeGzipLines(t, worksPath, []string{
		"/type/work\t/works/OL14986754W\t1\t2020-01-01T00:00:00.000000\t{\"title\":\"Pride and Prejudice\",\"description\":\"A classic novel\",\"subjects\":[\"Fiction\",\"Classic\"],\"covers\":[8231856],\"first_publish_date\":\"1813\",\"authors\":[{\"author\":{\"key\":\"/authors/OL23919A\"}}]}",
	})
	writeGzipLines(t, editionsPath, []string{
		"/type/edition\t/books/OL1M\t1\t2020-01-01T00:00:00.000000\t{\"works\":[{\"key\":\"/works/OL14986754W\"}],\"isbn_13\":[\"9780141439518\"],\"isbn_10\":[\"0141439513\"],\"publishers\":[\"Penguin\"],\"publish_date\":\"2002\",\"number_of_pages\":279,\"languages\":[{\"key\":\"/languages/eng\"}]}",
	})

	if _, err := ImportToDuckDB(ctx, dbPath, Options{
		AuthorsPath:  authorsPath,
		WorksPath:    worksPath,
		EditionsPath: editionsPath,
		ReplaceBooks: true,
	}); err != nil {
		t.Fatalf("import to duckdb: %v", err)
	}

	paths, err := ExportParquet(ctx, dbPath, parquetDir)
	if err != nil {
		t.Fatalf("export parquet: %v", err)
	}
	if len(paths) != 2 {
		t.Fatalf("unexpected parquet paths count: %d", len(paths))
	}
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			t.Fatalf("stat parquet file %s: %v", p, err)
		}
		if info.Size() <= 0 {
			t.Fatalf("empty parquet file: %s", p)
		}
	}

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	defer db.Close()
	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM read_parquet(?)", paths[0]).Scan(&count); err != nil {
		t.Fatalf("query parquet books: %v", err)
	}
	if count < 1 {
		t.Fatalf("expected at least one book row, got %d", count)
	}

	if err := DeleteSourceFiles(authorsPath, worksPath, editionsPath); err != nil {
		t.Fatalf("delete source files: %v", err)
	}
	if _, err := os.Stat(authorsPath); !os.IsNotExist(err) {
		t.Fatalf("authors source file still exists")
	}
}

func writeGzipLines(t *testing.T, path string, lines []string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create %s: %v", path, err)
	}
	defer f.Close()

	zw := gzip.NewWriter(f)
	for _, line := range lines {
		if _, err := zw.Write([]byte(line + "\n")); err != nil {
			t.Fatalf("write gzip %s: %v", path, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close gzip %s: %v", path, err)
	}
}
