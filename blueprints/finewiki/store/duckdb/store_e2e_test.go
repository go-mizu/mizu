//go:build e2e

package duckdb_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-mizu/blueprints/finewiki/cli"
	"github.com/go-mizu/blueprints/finewiki/feature/search"
	"github.com/go-mizu/blueprints/finewiki/store/duckdb"

	_ "github.com/duckdb/duckdb-go/v2"
)

// setupStore creates a store with real vi wiki data for testing.
func setupStore(t *testing.T) *duckdb.Store {
	t.Helper()

	if os.Getenv("E2E_TEST") != "1" {
		t.Skip("set E2E_TEST=1 to run")
	}

	dataDir := cli.DefaultDataDir()
	lang := "vi"

	if !parquetExists(dataDir, lang) {
		t.Skipf("Parquet not found at %s/%s; run 'finewiki import vi' first", dataDir, lang)
	}

	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	store, err := duckdb.New(db)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	ctx := context.Background()
	err = store.Ensure(ctx, duckdb.Config{
		ParquetGlob: cli.ParquetGlob(dataDir, lang),
		EnableFTS:   false,
	}, duckdb.EnsureOptions{
		SeedIfEmpty: true,
		BuildIndex:  true,
	})
	if err != nil {
		t.Fatalf("ensure: %v", err)
	}

	return store
}

func parquetExists(dataDir, lang string) bool {
	// Check single file
	if _, err := os.Stat(filepath.Join(dataDir, lang, "data.parquet")); err == nil {
		return true
	}
	// Check sharded files
	pattern := filepath.Join(dataDir, lang, "data-*.parquet")
	matches, _ := filepath.Glob(pattern)
	return len(matches) > 0
}

func TestStore_Search_E2E(t *testing.T) {
	store := setupStore(t)
	ctx := context.Background()

	t.Run("basic search", func(t *testing.T) {
		results, err := store.Search(ctx, search.Query{
			Text:  "vietnam",
			Limit: 10,
		})
		if err != nil {
			t.Fatalf("search: %v", err)
		}
		if len(results) == 0 {
			t.Error("expected results for 'vietnam'")
		}
		// Verify result structure
		for _, r := range results {
			if r.ID == "" {
				t.Error("result missing ID")
			}
			if r.WikiName == "" {
				t.Error("result missing WikiName")
			}
			if r.Title == "" {
				t.Error("result missing Title")
			}
			if r.InLanguage == "" {
				t.Error("result missing InLanguage")
			}
		}
	})

	t.Run("Vietnamese characters", func(t *testing.T) {
		results, err := store.Search(ctx, search.Query{
			Text:  "Việt",
			Limit: 10,
		})
		if err != nil {
			t.Fatalf("search: %v", err)
		}
		if len(results) == 0 {
			t.Error("expected results for 'Việt'")
		}
	})

	t.Run("empty query", func(t *testing.T) {
		results, err := store.Search(ctx, search.Query{
			Text:  "",
			Limit: 10,
		})
		if err != nil {
			t.Fatalf("search: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("expected 0 results for empty query, got %d", len(results))
		}
	})

	t.Run("WikiName filter", func(t *testing.T) {
		results, err := store.Search(ctx, search.Query{
			Text:     "vietnam",
			WikiName: "viwiki",
			Limit:    10,
		})
		if err != nil {
			t.Fatalf("search: %v", err)
		}
		for _, r := range results {
			if r.WikiName != "viwiki" {
				t.Errorf("result WikiName = %q, want 'viwiki'", r.WikiName)
			}
		}
	})

	t.Run("InLanguage filter", func(t *testing.T) {
		results, err := store.Search(ctx, search.Query{
			Text:       "vietnam",
			InLanguage: "vi",
			Limit:      10,
		})
		if err != nil {
			t.Fatalf("search: %v", err)
		}
		for _, r := range results {
			if r.InLanguage != "vi" {
				t.Errorf("result InLanguage = %q, want 'vi'", r.InLanguage)
			}
		}
	})

	t.Run("limit", func(t *testing.T) {
		results, err := store.Search(ctx, search.Query{
			Text:  "a",
			Limit: 5,
		})
		if err != nil {
			t.Fatalf("search: %v", err)
		}
		if len(results) > 5 {
			t.Errorf("got %d results, limit was 5", len(results))
		}
	})
}

func TestStore_GetByID_E2E(t *testing.T) {
	store := setupStore(t)
	ctx := context.Background()

	// First search to get a valid ID
	results, err := store.Search(ctx, search.Query{
		Text:  "vietnam",
		Limit: 1,
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) == 0 {
		t.Skip("no search results to test GetByID")
	}

	pageID := results[0].ID

	t.Run("valid ID", func(t *testing.T) {
		page, err := store.GetByID(ctx, pageID)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}

		if page.ID != pageID {
			t.Errorf("ID = %q, want %q", page.ID, pageID)
		}
		if page.Title == "" {
			t.Error("Title is empty")
		}
		if page.Text == "" {
			t.Error("Text is empty")
		}
		if page.WikiName == "" {
			t.Error("WikiName is empty")
		}
		if page.InLanguage == "" {
			t.Error("InLanguage is empty")
		}
		if page.URL == "" {
			t.Error("URL is empty")
		}
	})

	t.Run("non-existent ID", func(t *testing.T) {
		_, err := store.GetByID(ctx, "nonexistent/99999999")
		if err == nil {
			t.Error("expected error for non-existent ID")
		}
	})
}

func TestStore_GetByTitle_E2E(t *testing.T) {
	store := setupStore(t)
	ctx := context.Background()

	// First search to get a valid title
	results, err := store.Search(ctx, search.Query{
		Text:  "vietnam",
		Limit: 1,
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) == 0 {
		t.Skip("no search results to test GetByTitle")
	}

	wikiname := results[0].WikiName
	title := results[0].Title

	t.Run("valid title", func(t *testing.T) {
		page, err := store.GetByTitle(ctx, wikiname, title)
		if err != nil {
			t.Fatalf("GetByTitle: %v", err)
		}

		if page.Title != title {
			t.Errorf("Title = %q, want %q", page.Title, title)
		}
		if page.WikiName != wikiname {
			t.Errorf("WikiName = %q, want %q", page.WikiName, wikiname)
		}
		if page.Text == "" {
			t.Error("Text is empty")
		}
	})

	t.Run("non-existent title", func(t *testing.T) {
		_, err := store.GetByTitle(ctx, "viwiki", "NonExistentTitleXYZ12345")
		if err == nil {
			t.Error("expected error for non-existent title")
		}
	})
}

func TestStore_Stats_E2E(t *testing.T) {
	store := setupStore(t)
	ctx := context.Background()

	stats, err := store.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}

	// Check titles count
	titles, ok := stats["titles"]
	if !ok {
		t.Error("stats missing 'titles'")
	} else if titles.(int64) <= 0 {
		t.Errorf("titles = %d, want > 0", titles.(int64))
	}

	// Check pages count
	pages, ok := stats["pages"]
	if !ok {
		t.Error("stats missing 'pages'")
	} else if pages.(int64) <= 0 {
		t.Errorf("pages = %d, want > 0", pages.(int64))
	}

	// Check wikis
	wikis, ok := stats["wikis"]
	if !ok {
		t.Error("stats missing 'wikis'")
	} else {
		wikisMap := wikis.(map[string]int64)
		if _, ok := wikisMap["viwiki"]; !ok {
			t.Error("wikis missing 'viwiki'")
		}
	}

	// Check seeded_at
	seededAt, ok := stats["seeded_at"]
	if !ok {
		t.Error("stats missing 'seeded_at'")
	} else if seededAt.(string) == "" {
		t.Error("seeded_at is empty")
	}
}

func TestStore_Ensure_E2E(t *testing.T) {
	if os.Getenv("E2E_TEST") != "1" {
		t.Skip("set E2E_TEST=1 to run")
	}

	dataDir := cli.DefaultDataDir()
	lang := "vi"

	if !parquetExists(dataDir, lang) {
		t.Skipf("Parquet not found; run 'finewiki import vi' first")
	}

	ctx := context.Background()

	t.Run("idempotent", func(t *testing.T) {
		db, err := sql.Open("duckdb", "")
		if err != nil {
			t.Fatalf("open duckdb: %v", err)
		}
		defer db.Close()

		store, err := duckdb.New(db)
		if err != nil {
			t.Fatalf("new store: %v", err)
		}

		cfg := duckdb.Config{
			ParquetGlob: cli.ParquetGlob(dataDir, lang),
			EnableFTS:   false,
		}
		opts := duckdb.EnsureOptions{
			SeedIfEmpty: true,
			BuildIndex:  true,
		}

		// First call
		if err := store.Ensure(ctx, cfg, opts); err != nil {
			t.Fatalf("first ensure: %v", err)
		}

		// Second call should not fail
		if err := store.Ensure(ctx, cfg, opts); err != nil {
			t.Fatalf("second ensure: %v", err)
		}

		// Should still work
		results, err := store.Search(ctx, search.Query{Text: "vietnam", Limit: 1})
		if err != nil {
			t.Fatalf("search after double ensure: %v", err)
		}
		if len(results) == 0 {
			t.Error("no results after double ensure")
		}
	})
}
