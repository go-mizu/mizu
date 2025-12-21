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
	"github.com/go-mizu/blueprints/microblog/feature/interactions"
	"github.com/go-mizu/blueprints/microblog/feature/posts"
	"github.com/go-mizu/blueprints/microblog/feature/relationships"
	"github.com/go-mizu/blueprints/microblog/feature/timelines"
	"github.com/go-mizu/blueprints/microblog/store/duckdb"
)

func setupTimelinesTestEnv(t *testing.T) (*sql.DB, accounts.API, posts.API, timelines.API, relationships.API, interactions.API, func()) {
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
	timelinesStore := duckdb.NewTimelinesStore(db)
	relationshipsStore := duckdb.NewRelationshipsStore(db)
	interactionsStore := duckdb.NewInteractionsStore(db)

	accountsSvc := accounts.NewService(accountsStore)
	postsSvc := posts.NewService(postsStore, accountsSvc)
	timelinesSvc := timelines.NewService(timelinesStore, accountsSvc)
	relationshipsSvc := relationships.NewService(relationshipsStore)
	interactionsSvc := interactions.NewService(interactionsStore)

	cleanup := func() {
		db.Close()
	}

	return db, accountsSvc, postsSvc, timelinesSvc, relationshipsSvc, interactionsSvc, cleanup
}

func TestTimeline_Home(t *testing.T) {
	_, accountsSvc, postsSvc, timelinesSvc, relSvc, _, cleanup := setupTimelinesTestEnv(t)
	defer cleanup()

	account1, _ := accountsSvc.Create(context.Background(), &accounts.CreateIn{
		Username: "user1",
		Email:    "user1@example.com",
		Password: "password123",
	})

	account2, _ := accountsSvc.Create(context.Background(), &accounts.CreateIn{
		Username: "user2",
		Email:    "user2@example.com",
		Password: "password123",
	})

	// account1 follows account2
	relSvc.Follow(context.Background(), account1.ID, account2.ID)

	// Create post by account2
	postsSvc.Create(context.Background(), account2.ID, &posts.CreateIn{
		Content: "Post from user2",
	})

	getAccountID := func(c *mizu.Ctx) string {
		return account1.ID
	}
	optionalAuth := func(c *mizu.Ctx) string {
		return ""
	}

	h := handler.NewTimeline(timelinesSvc, getAccountID, optionalAuth)

	rec, ctx := testRequest("GET", "/api/v1/timelines/home", nil, account1.ID)

	if err := h.Home(ctx); err != nil {
		t.Fatalf("Home() error = %v", err)
	}

	if rec.Code != 200 {
		t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	data, ok := resp["data"].([]any)
	if !ok {
		t.Fatal("expected data field to be array")
	}

	if len(data) == 0 {
		t.Error("expected at least one post in home timeline")
	}
}

func TestTimeline_Local(t *testing.T) {
	_, accountsSvc, postsSvc, timelinesSvc, _, _, cleanup := setupTimelinesTestEnv(t)
	defer cleanup()

	account, _ := accountsSvc.Create(context.Background(), &accounts.CreateIn{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	})

	// Create a local post
	postsSvc.Create(context.Background(), account.ID, &posts.CreateIn{
		Content: "Local post",
	})

	getAccountID := func(c *mizu.Ctx) string {
		return ""
	}
	optionalAuth := func(c *mizu.Ctx) string {
		return ""
	}

	h := handler.NewTimeline(timelinesSvc, getAccountID, optionalAuth)

	rec, ctx := testRequest("GET", "/api/v1/timelines/local", nil, "")

	if err := h.Local(ctx); err != nil {
		t.Fatalf("Local() error = %v", err)
	}

	if rec.Code != 200 {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	data, ok := resp["data"].([]any)
	if !ok {
		t.Fatal("expected data field to be array")
	}

	if len(data) == 0 {
		t.Error("expected at least one post in local timeline")
	}
}

func TestTimeline_Hashtag(t *testing.T) {
	_, accountsSvc, postsSvc, timelinesSvc, _, _, cleanup := setupTimelinesTestEnv(t)
	defer cleanup()

	account, _ := accountsSvc.Create(context.Background(), &accounts.CreateIn{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	})

	// Create post with hashtag
	postsSvc.Create(context.Background(), account.ID, &posts.CreateIn{
		Content: "Hello #golang world!",
	})

	getAccountID := func(c *mizu.Ctx) string {
		return ""
	}
	optionalAuth := func(c *mizu.Ctx) string {
		return ""
	}

	h := handler.NewTimeline(timelinesSvc, getAccountID, optionalAuth)

	rec, ctx := testRequest("GET", "/api/v1/timelines/tag/golang", nil, "")
	ctx.Request().SetPathValue("tag", "golang")

	if err := h.Hashtag(ctx); err != nil {
		t.Fatalf("Hashtag() error = %v", err)
	}

	if rec.Code != 200 {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	data, ok := resp["data"].([]any)
	if !ok {
		t.Fatal("expected data field to be array")
	}

	if len(data) == 0 {
		t.Error("expected at least one post in hashtag timeline")
	}
}

func TestTimeline_Bookmarks(t *testing.T) {
	_, accountsSvc, postsSvc, timelinesSvc, _, interactionsSvc, cleanup := setupTimelinesTestEnv(t)
	defer cleanup()

	account, _ := accountsSvc.Create(context.Background(), &accounts.CreateIn{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	})

	// Create post and bookmark it
	post, _ := postsSvc.Create(context.Background(), account.ID, &posts.CreateIn{
		Content: "Bookmarked post",
	})
	interactionsSvc.Bookmark(context.Background(), account.ID, post.ID)

	getAccountID := func(c *mizu.Ctx) string {
		return account.ID
	}
	optionalAuth := func(c *mizu.Ctx) string {
		return ""
	}

	h := handler.NewTimeline(timelinesSvc, getAccountID, optionalAuth)

	rec, ctx := testRequest("GET", "/api/v1/bookmarks", nil, account.ID)

	if err := h.Bookmarks(ctx); err != nil {
		t.Fatalf("Bookmarks() error = %v", err)
	}

	if rec.Code != 200 {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	data, ok := resp["data"].([]any)
	if !ok {
		t.Fatal("expected data field to be array")
	}

	if len(data) == 0 {
		t.Error("expected at least one bookmarked post")
	}
}

func TestTimeline_HomeWithLimit(t *testing.T) {
	_, accountsSvc, postsSvc, timelinesSvc, relSvc, _, cleanup := setupTimelinesTestEnv(t)
	defer cleanup()

	account1, _ := accountsSvc.Create(context.Background(), &accounts.CreateIn{
		Username: "user1",
		Email:    "user1@example.com",
		Password: "password123",
	})

	account2, _ := accountsSvc.Create(context.Background(), &accounts.CreateIn{
		Username: "user2",
		Email:    "user2@example.com",
		Password: "password123",
	})

	// account1 follows account2
	relSvc.Follow(context.Background(), account1.ID, account2.ID)

	// Create multiple posts
	for i := 0; i < 5; i++ {
		postsSvc.Create(context.Background(), account2.ID, &posts.CreateIn{
			Content: "Post #" + string(rune('0'+i)),
		})
	}

	getAccountID := func(c *mizu.Ctx) string {
		return account1.ID
	}
	optionalAuth := func(c *mizu.Ctx) string {
		return ""
	}

	h := handler.NewTimeline(timelinesSvc, getAccountID, optionalAuth)

	rec, ctx := testRequest("GET", "/api/v1/timelines/home?limit=2", nil, account1.ID)
	ctx.Request().URL.RawQuery = "limit=2"

	if err := h.Home(ctx); err != nil {
		t.Fatalf("Home() error = %v", err)
	}

	if rec.Code != 200 {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	data, ok := resp["data"].([]any)
	if !ok {
		t.Fatal("expected data field to be array")
	}

	if len(data) > 2 {
		t.Errorf("expected at most 2 posts with limit=2, got %d", len(data))
	}
}
