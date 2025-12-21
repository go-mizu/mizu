package handler_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/microblog/app/web/handler"
	"github.com/go-mizu/blueprints/microblog/feature/accounts"
	"github.com/go-mizu/blueprints/microblog/feature/relationships"
	"github.com/go-mizu/blueprints/microblog/feature/timelines"
	"github.com/go-mizu/blueprints/microblog/store/duckdb"
)

// setupTestEnv creates a test environment with in-memory DuckDB.
func setupTestEnv(t *testing.T) (*sql.DB, accounts.API, relationships.API, timelines.API, func()) {
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
	relationshipsStore := duckdb.NewRelationshipsStore(db)
	timelinesStore := duckdb.NewTimelinesStore(db)

	accountsSvc := accounts.NewService(accountsStore)
	relationshipsSvc := relationships.NewService(relationshipsStore)
	timelinesSvc := timelines.NewService(timelinesStore, accountsSvc)

	cleanup := func() {
		db.Close()
	}

	return db, accountsSvc, relationshipsSvc, timelinesSvc, cleanup
}

// testRequest creates a test request with Mizu context.
func testRequest(method, path string, body []byte, accountID string) (*httptest.ResponseRecorder, *mizu.Ctx) {
	var reqBody *bytes.Buffer
	if body != nil {
		reqBody = bytes.NewBuffer(body)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req := httptest.NewRequest(method, path, reqBody)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()

	ctx := mizu.NewCtx(rec, req, nil)

	return rec, ctx
}

func TestAccount_VerifyCredentials(t *testing.T) {
	_, accountsSvc, relSvc, timelinesSvc, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create test account
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

	h := handler.NewAccount(accountsSvc, relSvc, timelinesSvc, getAccountID, optionalAuth)

	rec, ctx := testRequest("GET", "/api/v1/accounts/verify_credentials", nil, account.ID)

	if err := h.VerifyCredentials(ctx); err != nil {
		t.Fatalf("VerifyCredentials() error = %v", err)
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

	if data["username"] != "testuser" {
		t.Errorf("expected username testuser, got %v", data["username"])
	}
}

func TestAccount_UpdateCredentials(t *testing.T) {
	_, accountsSvc, relSvc, timelinesSvc, cleanup := setupTestEnv(t)
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

	h := handler.NewAccount(accountsSvc, relSvc, timelinesSvc, getAccountID, optionalAuth)

	updateBody := []byte(`{"display_name":"Updated Name","bio":"Updated bio"}`)
	rec, ctx := testRequest("PATCH", "/api/v1/accounts/update_credentials", updateBody, account.ID)

	if err := h.UpdateCredentials(ctx); err != nil {
		t.Fatalf("UpdateCredentials() error = %v", err)
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

	if data["display_name"] != "Updated Name" {
		t.Errorf("expected display_name 'Updated Name', got %v", data["display_name"])
	}
}

func TestAccount_GetAccount(t *testing.T) {
	_, accountsSvc, relSvc, timelinesSvc, cleanup := setupTestEnv(t)
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
		return ""
	}
	optionalAuth := func(c *mizu.Ctx) string {
		return ""
	}

	h := handler.NewAccount(accountsSvc, relSvc, timelinesSvc, getAccountID, optionalAuth)

	// Test with account ID
	rec, ctx := testRequest("GET", "/api/v1/accounts/"+account.ID, nil, "")
	ctx.Request().SetPathValue("id", account.ID)

	if err := h.GetAccount(ctx); err != nil {
		t.Fatalf("GetAccount() error = %v", err)
	}

	if rec.Code != 200 {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	// Test with username
	rec2, ctx2 := testRequest("GET", "/api/v1/accounts/testuser", nil, "")
	ctx2.Request().SetPathValue("id", "testuser")

	if err := h.GetAccount(ctx2); err != nil {
		t.Fatalf("GetAccount() with username error = %v", err)
	}

	if rec2.Code != 200 {
		t.Errorf("expected status 200, got %d", rec2.Code)
	}
}

func TestAccount_GetAccountFollowers(t *testing.T) {
	_, accountsSvc, relSvc, timelinesSvc, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create two accounts
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

	// account2 follows account1
	relSvc.Follow(context.Background(), account2.ID, account1.ID)

	getAccountID := func(c *mizu.Ctx) string {
		return ""
	}
	optionalAuth := func(c *mizu.Ctx) string {
		return ""
	}

	h := handler.NewAccount(accountsSvc, relSvc, timelinesSvc, getAccountID, optionalAuth)

	rec, ctx := testRequest("GET", "/api/v1/accounts/"+account1.ID+"/followers", nil, "")
	ctx.Request().SetPathValue("id", account1.ID)

	if err := h.GetAccountFollowers(ctx); err != nil {
		t.Fatalf("GetAccountFollowers() error = %v", err)
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
		t.Errorf("expected 1 follower, got %d", len(data))
	}
}

func TestAccount_GetAccountFollowing(t *testing.T) {
	_, accountsSvc, relSvc, timelinesSvc, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create two accounts
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

	getAccountID := func(c *mizu.Ctx) string {
		return ""
	}
	optionalAuth := func(c *mizu.Ctx) string {
		return ""
	}

	h := handler.NewAccount(accountsSvc, relSvc, timelinesSvc, getAccountID, optionalAuth)

	rec, ctx := testRequest("GET", "/api/v1/accounts/"+account1.ID+"/following", nil, "")
	ctx.Request().SetPathValue("id", account1.ID)

	if err := h.GetAccountFollowing(ctx); err != nil {
		t.Fatalf("GetAccountFollowing() error = %v", err)
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
		t.Errorf("expected 1 following, got %d", len(data))
	}
}

func TestAccount_GetAccountNotFound(t *testing.T) {
	_, accountsSvc, relSvc, timelinesSvc, cleanup := setupTestEnv(t)
	defer cleanup()

	getAccountID := func(c *mizu.Ctx) string {
		return ""
	}
	optionalAuth := func(c *mizu.Ctx) string {
		return ""
	}

	h := handler.NewAccount(accountsSvc, relSvc, timelinesSvc, getAccountID, optionalAuth)

	rec, ctx := testRequest("GET", "/api/v1/accounts/nonexistent", nil, "")
	ctx.Request().SetPathValue("id", "nonexistent")

	if err := h.GetAccount(ctx); err != nil {
		t.Fatalf("GetAccount() error = %v", err)
	}

	if rec.Code != 404 {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}
