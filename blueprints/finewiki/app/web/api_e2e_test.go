//go:build e2e

package web_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-mizu/blueprints/finewiki/app/web"
	"github.com/go-mizu/blueprints/finewiki/cli"
	"github.com/go-mizu/blueprints/finewiki/feature/search"
	"github.com/go-mizu/blueprints/finewiki/feature/view"
	"github.com/go-mizu/blueprints/finewiki/store/duckdb"

	_ "github.com/duckdb/duckdb-go/v2"
)

// TestAPI_E2E tests the full API stack with a real DuckDB store.
// Run with: E2E_TEST=1 go test -tags=e2e ./app/web -run E2E -v
func TestAPI_E2E(t *testing.T) {
	if os.Getenv("E2E_TEST") != "1" {
		t.Skip("Skipping E2E test; set E2E_TEST=1 to run")
	}

	ctx := context.Background()

	// Find parquet files for Vietnamese wiki
	dataDir := cli.DefaultDataDir()
	lang := "vi"

	if !parquetExists(dataDir, lang) {
		t.Skipf("Parquet files not found at %s/%s; run 'finewiki import vi' first", dataDir, lang)
	}

	// Create in-memory DuckDB for testing
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("failed to open duckdb: %v", err)
	}
	defer db.Close()

	store, err := duckdb.New(db)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	parquetGlob := cli.ParquetGlob(dataDir, lang)
	if err := store.Ensure(ctx, duckdb.Config{
		ParquetGlob: parquetGlob,
		EnableFTS:   false,
	}, duckdb.EnsureOptions{
		SeedIfEmpty: true,
		BuildIndex:  true,
		BuildFTS:    false,
	}); err != nil {
		t.Fatalf("failed to ensure store: %v", err)
	}

	// Create services
	searchSvc := search.New(store)
	viewSvc := view.New(store)

	// Create embedded templates
	tmpl, err := cli.NewTemplates()
	if err != nil {
		t.Fatalf("failed to create templates: %v", err)
	}

	// Create server
	srv := web.New(viewSvc, searchSvc, tmpl)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	t.Run("OpenAPI spec", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/openapi.json")
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}

		contentType := resp.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("expected application/json, got %s", contentType)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}

		var spec map[string]any
		if err := json.Unmarshal(body, &spec); err != nil {
			t.Errorf("invalid JSON: %v", err)
		}

		if _, ok := spec["openapi"]; !ok {
			t.Error("missing 'openapi' field")
		}
		if _, ok := spec["paths"]; !ok {
			t.Error("missing 'paths' field")
		}
	})

	t.Run("Docs UI", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/docs")
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}

		if len(body) == 0 {
			t.Error("empty body")
		}

		// Check for Scalar reference
		if !contains(string(body), "scalar") && !contains(string(body), "Scalar") {
			t.Error("missing Scalar UI reference")
		}
	})

	t.Run("Search API", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/search?q=vietnam")
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}

		var result struct {
			Results []struct {
				ID       string `json:"id"`
				WikiName string `json:"wikiname"`
				Title    string `json:"title"`
			} `json:"results"`
			Count int `json:"count"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode: %v", err)
		}

		if result.Count == 0 {
			t.Error("expected results, got 0")
		}

		if len(result.Results) == 0 {
			t.Error("empty results array")
		}
	})

	t.Run("Search API empty query", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/search?q=")
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}

		var result struct {
			Results []any `json:"results"`
			Count   int   `json:"count"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode: %v", err)
		}

		if result.Count != 0 {
			t.Errorf("expected 0 results for empty query, got %d", result.Count)
		}
	})

	t.Run("GetPage by ID", func(t *testing.T) {
		// First search for a page to get an ID
		searchResp, err := http.Get(ts.URL + "/api/search?q=vietnam&limit=1")
		if err != nil {
			t.Fatalf("search request failed: %v", err)
		}
		defer searchResp.Body.Close()

		var searchResult struct {
			Results []struct {
				ID string `json:"id"`
			} `json:"results"`
		}
		if err := json.NewDecoder(searchResp.Body).Decode(&searchResult); err != nil {
			t.Fatalf("failed to decode search: %v", err)
		}

		if len(searchResult.Results) == 0 {
			t.Skip("no results to test GetPage")
		}

		pageID := searchResult.Results[0].ID

		resp, err := http.Get(ts.URL + "/api/pages?id=" + pageID)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
		}

		var page struct {
			ID       string `json:"id"`
			WikiName string `json:"wikiname"`
			Title    string `json:"title"`
			Text     string `json:"text"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
			t.Fatalf("failed to decode: %v", err)
		}

		if page.ID != pageID {
			t.Errorf("expected id %s, got %s", pageID, page.ID)
		}

		if page.Title == "" {
			t.Error("expected non-empty title")
		}

		if page.Text == "" {
			t.Error("expected non-empty text")
		}
	})

	t.Run("GetPage missing params", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/pages")
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		// Should return an error (either 400 or 500 with error message)
		if resp.StatusCode == 200 {
			t.Error("expected error status for missing params")
		}
	})

	t.Run("Healthz endpoint", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/healthz")
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}

		if string(body) != "ok" {
			t.Errorf("expected 'ok', got %s", string(body))
		}
	})
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

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsRune(s, substr))
}

func containsRune(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
