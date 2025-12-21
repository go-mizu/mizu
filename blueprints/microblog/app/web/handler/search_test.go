package handler_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/microblog/app/web/handler"
	"github.com/go-mizu/blueprints/microblog/feature/accounts"
	"github.com/go-mizu/blueprints/microblog/feature/posts"
	"github.com/go-mizu/blueprints/microblog/feature/search"
	"github.com/go-mizu/blueprints/microblog/feature/trending"
	"github.com/go-mizu/blueprints/microblog/store/duckdb"
)

func setupSearchTestEnv(t *testing.T) (*sql.DB, accounts.API, posts.API, search.API, trending.API, func()) {
	t.Helper()

	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	store, err := duckdb.New(db)
	if err != nil {
		db.Close()
		t.Fatalf("failed to create store: %v", err)
	}

	if err := store.Ensure(context.Background()); err != nil {
		db.Close()
		t.Fatalf("failed to initialize schema: %v", err)
	}

	accountsStore := duckdb.NewAccountsStore(db)
	postsStore := duckdb.NewPostsStore(db)
	searchStore := duckdb.NewSearchStore(db)
	trendingStore := duckdb.NewTrendingStore(db)

	accountsSvc := accounts.NewService(accountsStore)
	postsSvc := posts.NewService(postsStore, accountsSvc)
	searchSvc := search.NewService(searchStore)
	trendingSvc := trending.NewService(trendingStore)

	cleanup := func() {
		db.Close()
	}

	return db, accountsSvc, postsSvc, searchSvc, trendingSvc, cleanup
}

func TestSearch_Search(t *testing.T) {
	_, accountsSvc, postsSvc, searchSvc, trendingSvc, cleanup := setupSearchTestEnv(t)
	defer cleanup()

	account, _ := accountsSvc.Create(context.Background(), &accounts.CreateIn{
		Username: "searchuser",
		Email:    "search@example.com",
		Password: "password123",
	})

	// Create a post with searchable content
	postsSvc.Create(context.Background(), account.ID, &posts.CreateIn{
		Content: "This is a test post about #golang",
	})

	optionalAuth := func(c *mizu.Ctx) string {
		return ""
	}

	h := handler.NewSearch(searchSvc, trendingSvc, postsSvc, optionalAuth)

	rec, ctx := testRequest("GET", "/api/v1/search?q=golang", nil, "")
	ctx.Request().URL.RawQuery = "q=golang"

	if err := h.Search(ctx); err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if rec.Code != 200 {
		t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatal("expected data field to be object")
	}

	// Check for expected fields
	if _, ok := data["accounts"]; !ok {
		t.Error("expected accounts field")
	}
	if _, ok := data["hashtags"]; !ok {
		t.Error("expected hashtags field")
	}
	if _, ok := data["posts"]; !ok {
		t.Error("expected posts field")
	}
}

func TestSearch_SearchEmpty(t *testing.T) {
	_, _, postsSvc, searchSvc, trendingSvc, cleanup := setupSearchTestEnv(t)
	defer cleanup()

	optionalAuth := func(c *mizu.Ctx) string {
		return ""
	}

	h := handler.NewSearch(searchSvc, trendingSvc, postsSvc, optionalAuth)

	rec, ctx := testRequest("GET", "/api/v1/search?q=nonexistent", nil, "")
	ctx.Request().URL.RawQuery = "q=nonexistent"

	if err := h.Search(ctx); err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if rec.Code != 200 {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestSearch_SearchWithLimit(t *testing.T) {
	_, accountsSvc, postsSvc, searchSvc, trendingSvc, cleanup := setupSearchTestEnv(t)
	defer cleanup()

	account, _ := accountsSvc.Create(context.Background(), &accounts.CreateIn{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	})

	// Create multiple posts
	for i := 0; i < 10; i++ {
		postsSvc.Create(context.Background(), account.ID, &posts.CreateIn{
			Content: "Test post with keyword searchable",
		})
	}

	optionalAuth := func(c *mizu.Ctx) string {
		return ""
	}

	h := handler.NewSearch(searchSvc, trendingSvc, postsSvc, optionalAuth)

	rec, ctx := testRequest("GET", "/api/v1/search?q=searchable&limit=5", nil, "")
	ctx.Request().URL.RawQuery = "q=searchable&limit=5"

	if err := h.Search(ctx); err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if rec.Code != 200 {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestSearch_TrendingTags(t *testing.T) {
	_, accountsSvc, postsSvc, searchSvc, trendingSvc, cleanup := setupSearchTestEnv(t)
	defer cleanup()

	account, _ := accountsSvc.Create(context.Background(), &accounts.CreateIn{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	})

	// Create posts with hashtags
	postsSvc.Create(context.Background(), account.ID, &posts.CreateIn{
		Content: "Post with #trending hashtag",
	})

	optionalAuth := func(c *mizu.Ctx) string {
		return ""
	}

	h := handler.NewSearch(searchSvc, trendingSvc, postsSvc, optionalAuth)

	rec, ctx := testRequest("GET", "/api/v1/trends/tags", nil, "")

	if err := h.TrendingTags(ctx); err != nil {
		t.Fatalf("TrendingTags() error = %v", err)
	}

	if rec.Code != 200 {
		t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	data, ok := resp["data"]
	if !ok {
		t.Fatal("expected data field in response")
	}

	// Data should be an array (even if empty)
	if data == nil {
		t.Log("no trending tags (expected)")
	}
}

func TestSearch_TrendingTagsWithLimit(t *testing.T) {
	_, _, postsSvc, searchSvc, trendingSvc, cleanup := setupSearchTestEnv(t)
	defer cleanup()

	optionalAuth := func(c *mizu.Ctx) string {
		return ""
	}

	h := handler.NewSearch(searchSvc, trendingSvc, postsSvc, optionalAuth)

	rec, ctx := testRequest("GET", "/api/v1/trends/tags?limit=5", nil, "")
	ctx.Request().URL.RawQuery = "limit=5"

	if err := h.TrendingTags(ctx); err != nil {
		t.Fatalf("TrendingTags() error = %v", err)
	}

	if rec.Code != 200 {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestSearch_TrendingPosts(t *testing.T) {
	_, accountsSvc, postsSvc, searchSvc, trendingSvc, cleanup := setupSearchTestEnv(t)
	defer cleanup()

	account, _ := accountsSvc.Create(context.Background(), &accounts.CreateIn{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	})

	// Create some posts
	postsSvc.Create(context.Background(), account.ID, &posts.CreateIn{
		Content: "Trending post",
	})

	optionalAuth := func(c *mizu.Ctx) string {
		return ""
	}

	h := handler.NewSearch(searchSvc, trendingSvc, postsSvc, optionalAuth)

	rec, ctx := testRequest("GET", "/api/v1/trends/posts", nil, "")

	if err := h.TrendingPosts(ctx); err != nil {
		t.Fatalf("TrendingPosts() error = %v", err)
	}

	if rec.Code != 200 {
		t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	data, ok := resp["data"]
	if !ok {
		t.Fatal("expected data field in response")
	}

	// Data should be an array (even if empty)
	if data == nil {
		t.Log("no trending posts (expected)")
	}
}

func TestSearch_TrendingPostsWithLimit(t *testing.T) {
	_, _, postsSvc, searchSvc, trendingSvc, cleanup := setupSearchTestEnv(t)
	defer cleanup()

	optionalAuth := func(c *mizu.Ctx) string {
		return ""
	}

	h := handler.NewSearch(searchSvc, trendingSvc, postsSvc, optionalAuth)

	rec, ctx := testRequest("GET", "/api/v1/trends/posts?limit=10", nil, "")
	ctx.Request().URL.RawQuery = "limit=10"

	if err := h.TrendingPosts(ctx); err != nil {
		t.Fatalf("TrendingPosts() error = %v", err)
	}

	if rec.Code != 200 {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}
