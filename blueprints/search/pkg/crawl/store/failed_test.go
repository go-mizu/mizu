package store_test

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	crawl "github.com/go-mizu/mizu/blueprints/search/pkg/crawl"
	"github.com/go-mizu/mizu/blueprints/search/pkg/crawl/store"
)

func TestFailedDB_AddURLAndLoadRetry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "failed.duckdb")
	fdb, err := store.OpenFailedDB(path)
	if err != nil {
		t.Fatalf("OpenFailedDB: %v", err)
	}

	runStart := time.Now()
	fdb.AddURL(crawl.FailedURL{
		URL:    "https://slow.example.com/",
		Domain: "slow.example.com",
		Reason: "http_timeout",
	})
	if err := fdb.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	seeds, err := store.LoadRetryURLsSince(path, runStart.Add(-time.Second))
	if err != nil {
		t.Fatalf("LoadRetryURLsSince: %v", err)
	}
	if len(seeds) != 1 {
		t.Fatalf("want 1 retry seed, got %d", len(seeds))
	}
	if seeds[0].URL != "https://slow.example.com/" {
		t.Errorf("unexpected URL: %s", seeds[0].URL)
	}
}

func TestFailedDB_ImplementsFailureWriter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "failed.duckdb")
	fdb, err := store.OpenFailedDB(path)
	if err != nil {
		t.Fatalf("OpenFailedDB: %v", err)
	}
	defer fdb.Close()

	var _ crawl.FailureWriter = fdb
}

func TestFailedDB_TopDomains(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "failed.duckdb")

	fdb, err := store.OpenFailedDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	// Add 3 failures for example.com, 1 for foo.net (unique URLs to avoid dedup on PK)
	for i := range 3 {
		fdb.AddURL(crawl.FailedURL{URL: fmt.Sprintf("http://example.com/p%d", i), Domain: "example.com", Reason: "http_timeout"})
	}
	fdb.AddURL(crawl.FailedURL{URL: "http://foo.net/p", Domain: "foo.net", Reason: "http_error"})
	if err := fdb.Close(); err != nil {
		t.Fatal(err)
	}

	top, err := store.FailedURLTopDomains(dbPath, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(top) != 2 {
		t.Fatalf("want 2 entries, got %d", len(top))
	}
	if top[0][0] != "example.com" || top[0][1] != "3" {
		t.Errorf("want example.com:3, got %v", top[0])
	}
	if top[1][0] != "foo.net" || top[1][1] != "1" {
		t.Errorf("want foo.net:1, got %v", top[1])
	}
}
