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
	"github.com/go-mizu/blueprints/microblog/store/duckdb"
)

func setupInteractionsTestEnv(t *testing.T) (*sql.DB, accounts.API, posts.API, interactions.API, func()) {
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
	interactionsStore := duckdb.NewInteractionsStore(db)

	accountsSvc := accounts.NewService(accountsStore)
	postsSvc := posts.NewService(postsStore, accountsSvc)
	interactionsSvc := interactions.NewService(interactionsStore)

	cleanup := func() {
		db.Close()
	}

	return db, accountsSvc, postsSvc, interactionsSvc, cleanup
}

func TestInteraction_Like(t *testing.T) {
	_, accountsSvc, postsSvc, interactionsSvc, cleanup := setupInteractionsTestEnv(t)
	defer cleanup()

	account, _ := accountsSvc.Create(context.Background(), &accounts.CreateIn{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	})

	post, _ := postsSvc.Create(context.Background(), account.ID, &posts.CreateIn{
		Content: "Test post",
	})

	getAccountID := func(c *mizu.Ctx) string {
		return account.ID
	}

	h := handler.NewInteraction(interactionsSvc, postsSvc, accountsSvc, getAccountID)

	rec, ctx := testRequest("POST", "/api/v1/posts/"+post.ID+"/like", nil, account.ID)
	ctx.Request().SetPathValue("id", post.ID)

	if err := h.Like(ctx); err != nil {
		t.Fatalf("Like() error = %v", err)
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
		t.Fatal("expected data field in response")
	}

	if data["liked"] != true {
		t.Error("expected post to be liked")
	}
}

func TestInteraction_Unlike(t *testing.T) {
	_, accountsSvc, postsSvc, interactionsSvc, cleanup := setupInteractionsTestEnv(t)
	defer cleanup()

	account, _ := accountsSvc.Create(context.Background(), &accounts.CreateIn{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	})

	post, _ := postsSvc.Create(context.Background(), account.ID, &posts.CreateIn{
		Content: "Test post",
	})

	// Like first
	interactionsSvc.Like(context.Background(), account.ID, post.ID)

	getAccountID := func(c *mizu.Ctx) string {
		return account.ID
	}

	h := handler.NewInteraction(interactionsSvc, postsSvc, accountsSvc, getAccountID)

	rec, ctx := testRequest("DELETE", "/api/v1/posts/"+post.ID+"/like", nil, account.ID)
	ctx.Request().SetPathValue("id", post.ID)

	if err := h.Unlike(ctx); err != nil {
		t.Fatalf("Unlike() error = %v", err)
	}

	if rec.Code != 200 {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatal("expected data field in response")
	}

	if data["liked"] == true {
		t.Error("expected post to be unliked")
	}
}

func TestInteraction_Repost(t *testing.T) {
	_, accountsSvc, postsSvc, interactionsSvc, cleanup := setupInteractionsTestEnv(t)
	defer cleanup()

	account, _ := accountsSvc.Create(context.Background(), &accounts.CreateIn{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	})

	post, _ := postsSvc.Create(context.Background(), account.ID, &posts.CreateIn{
		Content: "Test post",
	})

	getAccountID := func(c *mizu.Ctx) string {
		return account.ID
	}

	h := handler.NewInteraction(interactionsSvc, postsSvc, accountsSvc, getAccountID)

	rec, ctx := testRequest("POST", "/api/v1/posts/"+post.ID+"/repost", nil, account.ID)
	ctx.Request().SetPathValue("id", post.ID)

	if err := h.Repost(ctx); err != nil {
		t.Fatalf("Repost() error = %v", err)
	}

	if rec.Code != 200 {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatal("expected data field in response")
	}

	if data["reposted"] != true {
		t.Error("expected post to be reposted")
	}
}

func TestInteraction_Unrepost(t *testing.T) {
	_, accountsSvc, postsSvc, interactionsSvc, cleanup := setupInteractionsTestEnv(t)
	defer cleanup()

	account, _ := accountsSvc.Create(context.Background(), &accounts.CreateIn{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	})

	post, _ := postsSvc.Create(context.Background(), account.ID, &posts.CreateIn{
		Content: "Test post",
	})

	// Repost first
	interactionsSvc.Repost(context.Background(), account.ID, post.ID)

	getAccountID := func(c *mizu.Ctx) string {
		return account.ID
	}

	h := handler.NewInteraction(interactionsSvc, postsSvc, accountsSvc, getAccountID)

	rec, ctx := testRequest("DELETE", "/api/v1/posts/"+post.ID+"/repost", nil, account.ID)
	ctx.Request().SetPathValue("id", post.ID)

	if err := h.Unrepost(ctx); err != nil {
		t.Fatalf("Unrepost() error = %v", err)
	}

	if rec.Code != 200 {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestInteraction_Bookmark(t *testing.T) {
	_, accountsSvc, postsSvc, interactionsSvc, cleanup := setupInteractionsTestEnv(t)
	defer cleanup()

	account, _ := accountsSvc.Create(context.Background(), &accounts.CreateIn{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	})

	post, _ := postsSvc.Create(context.Background(), account.ID, &posts.CreateIn{
		Content: "Test post",
	})

	getAccountID := func(c *mizu.Ctx) string {
		return account.ID
	}

	h := handler.NewInteraction(interactionsSvc, postsSvc, accountsSvc, getAccountID)

	rec, ctx := testRequest("POST", "/api/v1/posts/"+post.ID+"/bookmark", nil, account.ID)
	ctx.Request().SetPathValue("id", post.ID)

	if err := h.Bookmark(ctx); err != nil {
		t.Fatalf("Bookmark() error = %v", err)
	}

	if rec.Code != 200 {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatal("expected data field in response")
	}

	if data["bookmarked"] != true {
		t.Error("expected post to be bookmarked")
	}
}

func TestInteraction_Unbookmark(t *testing.T) {
	_, accountsSvc, postsSvc, interactionsSvc, cleanup := setupInteractionsTestEnv(t)
	defer cleanup()

	account, _ := accountsSvc.Create(context.Background(), &accounts.CreateIn{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	})

	post, _ := postsSvc.Create(context.Background(), account.ID, &posts.CreateIn{
		Content: "Test post",
	})

	// Bookmark first
	interactionsSvc.Bookmark(context.Background(), account.ID, post.ID)

	getAccountID := func(c *mizu.Ctx) string {
		return account.ID
	}

	h := handler.NewInteraction(interactionsSvc, postsSvc, accountsSvc, getAccountID)

	rec, ctx := testRequest("DELETE", "/api/v1/posts/"+post.ID+"/bookmark", nil, account.ID)
	ctx.Request().SetPathValue("id", post.ID)

	if err := h.Unbookmark(ctx); err != nil {
		t.Fatalf("Unbookmark() error = %v", err)
	}

	if rec.Code != 200 {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestInteraction_LikedBy(t *testing.T) {
	_, accountsSvc, postsSvc, interactionsSvc, cleanup := setupInteractionsTestEnv(t)
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

	post, _ := postsSvc.Create(context.Background(), account1.ID, &posts.CreateIn{
		Content: "Test post",
	})

	// account2 likes the post
	interactionsSvc.Like(context.Background(), account2.ID, post.ID)

	getAccountID := func(c *mizu.Ctx) string {
		return ""
	}

	h := handler.NewInteraction(interactionsSvc, postsSvc, accountsSvc, getAccountID)

	rec, ctx := testRequest("GET", "/api/v1/posts/"+post.ID+"/liked_by", nil, "")
	ctx.Request().SetPathValue("id", post.ID)

	if err := h.LikedBy(ctx); err != nil {
		t.Fatalf("LikedBy() error = %v", err)
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

	if len(data) != 1 {
		t.Errorf("expected 1 account, got %d", len(data))
	}
}

func TestInteraction_RepostedBy(t *testing.T) {
	_, accountsSvc, postsSvc, interactionsSvc, cleanup := setupInteractionsTestEnv(t)
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

	post, _ := postsSvc.Create(context.Background(), account1.ID, &posts.CreateIn{
		Content: "Test post",
	})

	// account2 reposts the post
	interactionsSvc.Repost(context.Background(), account2.ID, post.ID)

	getAccountID := func(c *mizu.Ctx) string {
		return ""
	}

	h := handler.NewInteraction(interactionsSvc, postsSvc, accountsSvc, getAccountID)

	rec, ctx := testRequest("GET", "/api/v1/posts/"+post.ID+"/reposted_by", nil, "")
	ctx.Request().SetPathValue("id", post.ID)

	if err := h.RepostedBy(ctx); err != nil {
		t.Fatalf("RepostedBy() error = %v", err)
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

	if len(data) != 1 {
		t.Errorf("expected 1 account, got %d", len(data))
	}
}
