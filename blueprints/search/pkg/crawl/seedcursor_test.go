package crawl_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"strconv"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
	crawl "github.com/go-mizu/mizu/blueprints/search/pkg/crawl"
)

func makeSeedDB(t *testing.T, rows int) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "seeds.duckdb")
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE docs (url VARCHAR, domain VARCHAR)`); err != nil {
		t.Fatal(err)
	}
	for i := range rows {
		if _, err := db.Exec("INSERT INTO docs VALUES (?, ?)",
			"http://example.com/"+strconv.Itoa(i), "example.com"); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

func TestSeedCursorPageThrough(t *testing.T) {
	path := makeSeedDB(t, 25)
	c, err := crawl.NewSeedCursor(path, 10)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	total := 0
	for {
		page, err := c.Next(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if len(page) == 0 {
			break
		}
		total += len(page)
	}
	if total != 25 {
		t.Fatalf("got %d rows, want 25", total)
	}
}

func TestSeedCursorEmpty(t *testing.T) {
	path := makeSeedDB(t, 0)
	c, err := crawl.NewSeedCursor(path, 10)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	page, err := c.Next(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(page) != 0 {
		t.Fatalf("expected empty page, got %d rows", len(page))
	}
}
