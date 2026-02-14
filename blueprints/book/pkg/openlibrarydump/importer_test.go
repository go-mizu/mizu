package openlibrarydump

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePathsPicksLatestByPattern(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	mustTouch(t, filepath.Join(dir, "ol_dump_authors_2025-10-30.txt.gz"))
	mustTouch(t, filepath.Join(dir, "ol_dump_authors_2025-10-31.txt.gz"))
	mustTouch(t, filepath.Join(dir, "ol_dump_works_2025-10-31.txt.gz"))
	mustTouch(t, filepath.Join(dir, "ol_dump_editions_2025-10-29.txt.gz"))
	mustTouch(t, filepath.Join(dir, "ol_dump_editions_2025-10-31.txt.gz"))

	got, err := ResolvePaths(Options{Dir: dir})
	if err != nil {
		t.Fatalf("resolve paths: %v", err)
	}

	if filepath.Base(got.AuthorsPath) != "ol_dump_authors_2025-10-31.txt.gz" {
		t.Fatalf("authors latest mismatch: %s", got.AuthorsPath)
	}
	if filepath.Base(got.WorksPath) != "ol_dump_works_2025-10-31.txt.gz" {
		t.Fatalf("works latest mismatch: %s", got.WorksPath)
	}
	if filepath.Base(got.EditionsPath) != "ol_dump_editions_2025-10-31.txt.gz" {
		t.Fatalf("editions latest mismatch: %s", got.EditionsPath)
	}
}

func mustTouch(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("touch %s: %v", path, err)
	}
}
