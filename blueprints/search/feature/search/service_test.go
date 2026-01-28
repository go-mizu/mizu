package search_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/feature/search"
	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/searxng"
	"github.com/go-mizu/mizu/blueprints/search/store/sqlite"
	"github.com/go-mizu/mizu/blueprints/search/types"
)

func getSearXNGURL() string {
	if url := os.Getenv("SEARXNG_URL"); url != "" {
		return url
	}
	return "http://localhost:8888"
}

func setupTestService(t *testing.T) (*search.Service, *sqlite.Store) {
	t.Helper()

	// Create in-memory SQLite store
	st, err := sqlite.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Initialize schema
	ctx := context.Background()
	if err := st.Ensure(ctx); err != nil {
		t.Fatalf("failed to ensure schema: %v", err)
	}

	// Seed knowledge for testing
	if err := st.SeedKnowledge(ctx); err != nil {
		t.Fatalf("failed to seed knowledge: %v", err)
	}

	// Create SearXNG engine
	eng := searxng.New(getSearXNGURL())

	// Check if SearXNG is available
	if err := eng.Healthz(ctx); err != nil {
		t.Skipf("SearXNG not available: %v", err)
	}

	// Create cache
	cacheStore := st.Cache()
	cache := search.NewCacheWithDefaults(cacheStore)

	// Create service
	svc := search.NewService(search.ServiceConfig{
		Engine: eng,
		Cache:  cache,
		Store:  st,
	})

	return svc, st
}

func setupTestServiceWithoutEngine(t *testing.T) (*search.Service, *sqlite.Store) {
	t.Helper()

	// Create in-memory SQLite store
	st, err := sqlite.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Initialize schema
	ctx := context.Background()
	if err := st.Ensure(ctx); err != nil {
		t.Fatalf("failed to ensure schema: %v", err)
	}

	// Seed data
	if err := st.SeedDocuments(ctx); err != nil {
		t.Fatalf("failed to seed documents: %v", err)
	}
	if err := st.SeedKnowledge(ctx); err != nil {
		t.Fatalf("failed to seed knowledge: %v", err)
	}

	// Create service without engine (fallback mode)
	svc := search.NewServiceWithDefaults(st)

	return svc, st
}

func TestService_Search_WithSearXNG(t *testing.T) {
	svc, st := setupTestService(t)
	defer st.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := svc.Search(ctx, "golang", types.SearchOptions{
		Page:    1,
		PerPage: 10,
	})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if resp.Query != "golang" {
		t.Errorf("expected query 'golang', got %q", resp.Query)
	}

	if len(resp.Results) == 0 {
		t.Error("expected at least one result")
	}

	// Check first result has required fields
	if len(resp.Results) > 0 {
		r := resp.Results[0]
		if r.URL == "" {
			t.Error("expected result to have URL")
		}
		if r.Title == "" {
			t.Error("expected result to have title")
		}
	}

	t.Logf("Got %d results for 'golang'", len(resp.Results))
}

func TestService_Search_WithCache(t *testing.T) {
	svc, st := setupTestService(t)
	defer st.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// First search - should hit SearXNG
	start1 := time.Now()
	resp1, err := svc.Search(ctx, "cache-test-query-unique", types.SearchOptions{
		Page:    1,
		PerPage: 10,
	})
	if err != nil {
		t.Fatalf("first search failed: %v", err)
	}
	duration1 := time.Since(start1)

	// Second search - should hit cache
	start2 := time.Now()
	resp2, err := svc.Search(ctx, "cache-test-query-unique", types.SearchOptions{
		Page:    1,
		PerPage: 10,
	})
	if err != nil {
		t.Fatalf("second search failed: %v", err)
	}
	duration2 := time.Since(start2)

	// Cache should be faster
	if duration2 > duration1 {
		t.Logf("Cache might not be working: first=%v, second=%v", duration1, duration2)
	}

	// Results should be the same
	if resp1.Query != resp2.Query {
		t.Errorf("queries differ: %q vs %q", resp1.Query, resp2.Query)
	}
	if len(resp1.Results) != len(resp2.Results) {
		t.Errorf("result counts differ: %d vs %d", len(resp1.Results), len(resp2.Results))
	}

	t.Logf("First search: %v, Second search: %v", duration1, duration2)
}

func TestService_Search_FallbackToStore(t *testing.T) {
	svc, st := setupTestServiceWithoutEngine(t)
	defer st.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := svc.Search(ctx, "golang", types.SearchOptions{
		Page:    1,
		PerPage: 10,
	})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if resp.Query != "golang" {
		t.Errorf("expected query 'golang', got %q", resp.Query)
	}

	t.Logf("Got %d results from store fallback", len(resp.Results))
}

func TestService_Search_RecordsHistory(t *testing.T) {
	svc, st := setupTestService(t)
	defer st.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Perform search
	_, err := svc.Search(ctx, "history-test-query", types.SearchOptions{
		Page:    1,
		PerPage: 10,
	})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	// Check history was recorded
	history, err := st.History().GetHistory(ctx, 10, 0)
	if err != nil {
		t.Fatalf("failed to get history: %v", err)
	}

	var found bool
	for _, h := range history {
		if h.Query == "history-test-query" {
			found = true
			break
		}
	}

	if !found {
		t.Error("search query was not recorded in history")
	}
}

func TestService_Search_RecordsSuggestions(t *testing.T) {
	svc, st := setupTestService(t)
	defer st.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Perform multiple searches
	queries := []string{"suggest-test-1", "suggest-test-2", "suggest-test-1"}
	for _, q := range queries {
		_, err := svc.Search(ctx, q, types.SearchOptions{
			Page:    1,
			PerPage: 10,
		})
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
	}

	// Check suggestions
	suggestions, err := st.Suggest().GetSuggestions(ctx, "suggest-test", 10)
	if err != nil {
		t.Fatalf("failed to get suggestions: %v", err)
	}

	if len(suggestions) == 0 {
		t.Error("expected suggestions to be recorded")
	}

	t.Logf("Got %d suggestions for 'suggest-test'", len(suggestions))
}

func TestService_Search_WithInstantAnswer(t *testing.T) {
	svc, st := setupTestServiceWithoutEngine(t)
	defer st.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Search with a calculator query
	resp, err := svc.Search(ctx, "2+2", types.SearchOptions{
		Page:    1,
		PerPage: 10,
	})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if resp.InstantAnswer == nil {
		t.Log("No instant answer detected (may depend on instant service implementation)")
	} else {
		t.Logf("Instant answer: %+v", resp.InstantAnswer)
	}
}

func TestService_Search_WithKnowledgePanel(t *testing.T) {
	svc, st := setupTestService(t)
	defer st.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Search for an entity that should have a knowledge panel
	resp, err := svc.Search(ctx, "Go", types.SearchOptions{
		Page:    1,
		PerPage: 10,
	})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if resp.KnowledgePanel == nil {
		t.Log("No knowledge panel found (may depend on seeded entities)")
	} else {
		t.Logf("Knowledge panel: %s - %s", resp.KnowledgePanel.Title, resp.KnowledgePanel.Description)
	}
}

func TestService_SearchImages(t *testing.T) {
	svc, st := setupTestService(t)
	defer st.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	results, err := svc.SearchImages(ctx, "golang logo", types.SearchOptions{
		Page:    1,
		PerPage: 10,
	})
	if err != nil {
		t.Fatalf("image search failed: %v", err)
	}

	if len(results) == 0 {
		t.Skip("no image results returned")
	}

	// Check first result
	r := results[0]
	if r.URL == "" && r.ThumbnailURL == "" {
		t.Error("expected image result to have URL or thumbnail")
	}

	t.Logf("Got %d image results", len(results))
}

func TestService_SearchVideos(t *testing.T) {
	svc, st := setupTestService(t)
	defer st.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	results, err := svc.SearchVideos(ctx, "golang tutorial", types.SearchOptions{
		Page:    1,
		PerPage: 10,
	})
	if err != nil {
		t.Fatalf("video search failed: %v", err)
	}

	if len(results) == 0 {
		t.Skip("no video results returned")
	}

	t.Logf("Got %d video results", len(results))
}

func TestService_SearchNews(t *testing.T) {
	svc, st := setupTestService(t)
	defer st.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	results, err := svc.SearchNews(ctx, "technology", types.SearchOptions{
		Page:    1,
		PerPage: 10,
	})
	if err != nil {
		t.Fatalf("news search failed: %v", err)
	}

	if len(results) == 0 {
		t.Skip("no news results returned")
	}

	t.Logf("Got %d news results", len(results))
}

func TestService_Search_Pagination(t *testing.T) {
	svc, st := setupTestService(t)
	defer st.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Page 1
	resp1, err := svc.Search(ctx, "programming", types.SearchOptions{
		Page:    1,
		PerPage: 5,
	})
	if err != nil {
		t.Fatalf("page 1 search failed: %v", err)
	}

	// Page 2
	resp2, err := svc.Search(ctx, "programming", types.SearchOptions{
		Page:    2,
		PerPage: 5,
	})
	if err != nil {
		t.Fatalf("page 2 search failed: %v", err)
	}

	// Pages should have different results
	if len(resp1.Results) > 0 && len(resp2.Results) > 0 {
		if resp1.Results[0].URL == resp2.Results[0].URL {
			t.Error("page 1 and page 2 have the same first result")
		}
	}

	t.Logf("Page 1: %d results, Page 2: %d results", len(resp1.Results), len(resp2.Results))
}
