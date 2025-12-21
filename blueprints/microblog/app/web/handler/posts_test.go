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
	"github.com/go-mizu/blueprints/microblog/store/duckdb"
)

func setupPostsTestEnv(t *testing.T) (*sql.DB, accounts.API, posts.API, func()) {
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

	accountsSvc := accounts.NewService(accountsStore)
	postsSvc := posts.NewService(postsStore, accountsSvc)

	cleanup := func() {
		db.Close()
	}

	return db, accountsSvc, postsSvc, cleanup
}

func TestPost_Create(t *testing.T) {
	_, accountsSvc, postsSvc, cleanup := setupPostsTestEnv(t)
	defer cleanup()

	account, err := accountsSvc.Create(context.Background(), &accounts.CreateIn{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("failed to create account: %v", err)
	}

	getAccountID := func(c *mizu.Ctx) string {
		return account.ID
	}
	optionalAuth := func(c *mizu.Ctx) string {
		return ""
	}

	h := handler.NewPost(postsSvc, getAccountID, optionalAuth)

	postBody := []byte(`{"content":"Hello world! #test"}`)
	rec, ctx := testRequest("POST", "/api/v1/posts", postBody, account.ID)

	if err := h.Create(ctx); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if rec.Code != 201 {
		t.Errorf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatal("expected data field in response")
	}

	if data["content"] != "Hello world! #test" {
		t.Errorf("expected content 'Hello world! #test', got %v", data["content"])
	}
}

func TestPost_CreateInvalidJSON(t *testing.T) {
	_, accountsSvc, postsSvc, cleanup := setupPostsTestEnv(t)
	defer cleanup()

	account, err := accountsSvc.Create(context.Background(), &accounts.CreateIn{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("failed to create account: %v", err)
	}

	getAccountID := func(c *mizu.Ctx) string {
		return account.ID
	}
	optionalAuth := func(c *mizu.Ctx) string {
		return ""
	}

	h := handler.NewPost(postsSvc, getAccountID, optionalAuth)

	postBody := []byte(`{invalid json}`)
	rec, ctx := testRequest("POST", "/api/v1/posts", postBody, account.ID)

	if err := h.Create(ctx); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if rec.Code != 400 {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func TestPost_Get(t *testing.T) {
	_, accountsSvc, postsSvc, cleanup := setupPostsTestEnv(t)
	defer cleanup()

	account, err := accountsSvc.Create(context.Background(), &accounts.CreateIn{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("failed to create account: %v", err)
	}

	post, err := postsSvc.Create(context.Background(), account.ID, &posts.CreateIn{
		Content: "Test post content",
	})
	if err != nil {
		t.Fatalf("failed to create post: %v", err)
	}

	getAccountID := func(c *mizu.Ctx) string {
		return ""
	}
	optionalAuth := func(c *mizu.Ctx) string {
		return ""
	}

	h := handler.NewPost(postsSvc, getAccountID, optionalAuth)

	rec, ctx := testRequest("GET", "/api/v1/posts/"+post.ID, nil, "")
	ctx.Request().SetPathValue("id", post.ID)

	if err := h.Get(ctx); err != nil {
		t.Fatalf("Get() error = %v", err)
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

	if data["content"] != "Test post content" {
		t.Errorf("expected content 'Test post content', got %v", data["content"])
	}
}

func TestPost_GetNotFound(t *testing.T) {
	_, _, postsSvc, cleanup := setupPostsTestEnv(t)
	defer cleanup()

	getAccountID := func(c *mizu.Ctx) string {
		return ""
	}
	optionalAuth := func(c *mizu.Ctx) string {
		return ""
	}

	h := handler.NewPost(postsSvc, getAccountID, optionalAuth)

	rec, ctx := testRequest("GET", "/api/v1/posts/nonexistent", nil, "")
	ctx.Request().SetPathValue("id", "nonexistent")

	if err := h.Get(ctx); err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if rec.Code != 404 {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func TestPost_Update(t *testing.T) {
	_, accountsSvc, postsSvc, cleanup := setupPostsTestEnv(t)
	defer cleanup()

	account, err := accountsSvc.Create(context.Background(), &accounts.CreateIn{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("failed to create account: %v", err)
	}

	post, err := postsSvc.Create(context.Background(), account.ID, &posts.CreateIn{
		Content: "Original content",
	})
	if err != nil {
		t.Fatalf("failed to create post: %v", err)
	}

	getAccountID := func(c *mizu.Ctx) string {
		return account.ID
	}
	optionalAuth := func(c *mizu.Ctx) string {
		return ""
	}

	h := handler.NewPost(postsSvc, getAccountID, optionalAuth)

	updateBody := []byte(`{"content":"Updated content"}`)
	rec, ctx := testRequest("PUT", "/api/v1/posts/"+post.ID, updateBody, account.ID)
	ctx.Request().SetPathValue("id", post.ID)

	if err := h.Update(ctx); err != nil {
		t.Fatalf("Update() error = %v", err)
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

	if data["content"] != "Updated content" {
		t.Errorf("expected content 'Updated content', got %v", data["content"])
	}
}

func TestPost_Delete(t *testing.T) {
	_, accountsSvc, postsSvc, cleanup := setupPostsTestEnv(t)
	defer cleanup()

	account, err := accountsSvc.Create(context.Background(), &accounts.CreateIn{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("failed to create account: %v", err)
	}

	post, err := postsSvc.Create(context.Background(), account.ID, &posts.CreateIn{
		Content: "Post to delete",
	})
	if err != nil {
		t.Fatalf("failed to create post: %v", err)
	}

	getAccountID := func(c *mizu.Ctx) string {
		return account.ID
	}
	optionalAuth := func(c *mizu.Ctx) string {
		return ""
	}

	h := handler.NewPost(postsSvc, getAccountID, optionalAuth)

	rec, ctx := testRequest("DELETE", "/api/v1/posts/"+post.ID, nil, account.ID)
	ctx.Request().SetPathValue("id", post.ID)

	if err := h.Delete(ctx); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if rec.Code != 200 {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	// Verify post is deleted
	_, err = postsSvc.GetByID(context.Background(), post.ID, "")
	if err == nil {
		t.Error("expected post to be deleted")
	}
}

func TestPost_GetContext(t *testing.T) {
	_, accountsSvc, postsSvc, cleanup := setupPostsTestEnv(t)
	defer cleanup()

	account, err := accountsSvc.Create(context.Background(), &accounts.CreateIn{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("failed to create account: %v", err)
	}

	// Create parent post
	parentPost, err := postsSvc.Create(context.Background(), account.ID, &posts.CreateIn{
		Content: "Parent post",
	})
	if err != nil {
		t.Fatalf("failed to create parent post: %v", err)
	}

	// Create reply
	replyPost, err := postsSvc.Create(context.Background(), account.ID, &posts.CreateIn{
		Content:   "Reply post",
		ReplyToID: parentPost.ID,
	})
	if err != nil {
		t.Fatalf("failed to create reply post: %v", err)
	}

	getAccountID := func(c *mizu.Ctx) string {
		return ""
	}
	optionalAuth := func(c *mizu.Ctx) string {
		return ""
	}

	h := handler.NewPost(postsSvc, getAccountID, optionalAuth)

	rec, ctx := testRequest("GET", "/api/v1/posts/"+replyPost.ID+"/context", nil, "")
	ctx.Request().SetPathValue("id", replyPost.ID)

	if err := h.GetContext(ctx); err != nil {
		t.Fatalf("GetContext() error = %v", err)
	}

	if rec.Code != 200 {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}
