package store_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	crawl "github.com/go-mizu/mizu/blueprints/search/pkg/crawl"
	"github.com/go-mizu/mizu/blueprints/search/pkg/crawl/store"
)

func TestResultDB_AddFlushClose(t *testing.T) {
	dir := t.TempDir()
	rdb, err := store.NewResultDB(dir, 2, 10, 64)
	if err != nil {
		t.Fatalf("NewResultDB: %v", err)
	}

	rdb.Add(crawl.Result{
		URL:        "https://example.com/",
		StatusCode: 200,
		Domain:     "example.com",
		CrawledAt:  time.Now(),
	})

	if err := rdb.Flush(context.Background()); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if err := rdb.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Verify file was created
	entries, _ := os.ReadDir(dir)
	var dbs int
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".duckdb" {
			dbs++
		}
	}
	if dbs == 0 {
		t.Error("expected at least one .duckdb shard file")
	}
}

func TestResultDB_ImplementsResultWriter(t *testing.T) {
	dir := t.TempDir()
	rdb, err := store.NewResultDB(dir, 1, 10, 64)
	if err != nil {
		t.Fatalf("NewResultDB: %v", err)
	}
	defer rdb.Close()

	// store.ResultDB must satisfy crawl.ResultWriter directly
	var _ crawl.ResultWriter = rdb
}
