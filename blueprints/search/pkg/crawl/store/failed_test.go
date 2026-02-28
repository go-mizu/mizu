package store_test

import (
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
