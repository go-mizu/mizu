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
	"github.com/go-mizu/blueprints/microblog/feature/relationships"
	"github.com/go-mizu/blueprints/microblog/store/duckdb"
)

func setupRelationshipsTestEnv(t *testing.T) (*sql.DB, accounts.API, relationships.API, func()) {
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

	accountsSvc := accounts.NewService(accountsStore)
	relationshipsSvc := relationships.NewService(relationshipsStore)

	cleanup := func() {
		db.Close()
	}

	return db, accountsSvc, relationshipsSvc, cleanup
}

func TestRelationship_Follow(t *testing.T) {
	_, accountsSvc, relSvc, cleanup := setupRelationshipsTestEnv(t)
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

	getAccountID := func(c *mizu.Ctx) string {
		return account1.ID
	}

	h := handler.NewRelationship(relSvc, getAccountID)

	rec, ctx := testRequest("POST", "/api/v1/accounts/"+account2.ID+"/follow", nil, account1.ID)
	ctx.Request().SetPathValue("id", account2.ID)

	if err := h.Follow(ctx); err != nil {
		t.Fatalf("Follow() error = %v", err)
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

	if data["following"] != true {
		t.Error("expected following to be true")
	}
}

func TestRelationship_Unfollow(t *testing.T) {
	_, accountsSvc, relSvc, cleanup := setupRelationshipsTestEnv(t)
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

	// Follow first
	relSvc.Follow(context.Background(), account1.ID, account2.ID)

	getAccountID := func(c *mizu.Ctx) string {
		return account1.ID
	}

	h := handler.NewRelationship(relSvc, getAccountID)

	rec, ctx := testRequest("POST", "/api/v1/accounts/"+account2.ID+"/unfollow", nil, account1.ID)
	ctx.Request().SetPathValue("id", account2.ID)

	if err := h.Unfollow(ctx); err != nil {
		t.Fatalf("Unfollow() error = %v", err)
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

	if data["following"] == true {
		t.Error("expected following to be false")
	}
}

func TestRelationship_Block(t *testing.T) {
	_, accountsSvc, relSvc, cleanup := setupRelationshipsTestEnv(t)
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

	getAccountID := func(c *mizu.Ctx) string {
		return account1.ID
	}

	h := handler.NewRelationship(relSvc, getAccountID)

	rec, ctx := testRequest("POST", "/api/v1/accounts/"+account2.ID+"/block", nil, account1.ID)
	ctx.Request().SetPathValue("id", account2.ID)

	if err := h.Block(ctx); err != nil {
		t.Fatalf("Block() error = %v", err)
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

	if data["blocking"] != true {
		t.Error("expected blocking to be true")
	}
}

func TestRelationship_Unblock(t *testing.T) {
	_, accountsSvc, relSvc, cleanup := setupRelationshipsTestEnv(t)
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

	// Block first
	relSvc.Block(context.Background(), account1.ID, account2.ID)

	getAccountID := func(c *mizu.Ctx) string {
		return account1.ID
	}

	h := handler.NewRelationship(relSvc, getAccountID)

	rec, ctx := testRequest("POST", "/api/v1/accounts/"+account2.ID+"/unblock", nil, account1.ID)
	ctx.Request().SetPathValue("id", account2.ID)

	if err := h.Unblock(ctx); err != nil {
		t.Fatalf("Unblock() error = %v", err)
	}

	if rec.Code != 200 {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestRelationship_Mute(t *testing.T) {
	_, accountsSvc, relSvc, cleanup := setupRelationshipsTestEnv(t)
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

	getAccountID := func(c *mizu.Ctx) string {
		return account1.ID
	}

	h := handler.NewRelationship(relSvc, getAccountID)

	rec, ctx := testRequest("POST", "/api/v1/accounts/"+account2.ID+"/mute", nil, account1.ID)
	ctx.Request().SetPathValue("id", account2.ID)

	if err := h.Mute(ctx); err != nil {
		t.Fatalf("Mute() error = %v", err)
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

	if data["muting"] != true {
		t.Error("expected muting to be true")
	}
}

func TestRelationship_Unmute(t *testing.T) {
	_, accountsSvc, relSvc, cleanup := setupRelationshipsTestEnv(t)
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

	// Mute first
	relSvc.Mute(context.Background(), account1.ID, account2.ID, true, nil)

	getAccountID := func(c *mizu.Ctx) string {
		return account1.ID
	}

	h := handler.NewRelationship(relSvc, getAccountID)

	rec, ctx := testRequest("POST", "/api/v1/accounts/"+account2.ID+"/unmute", nil, account1.ID)
	ctx.Request().SetPathValue("id", account2.ID)

	if err := h.Unmute(ctx); err != nil {
		t.Fatalf("Unmute() error = %v", err)
	}

	if rec.Code != 200 {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestRelationship_GetRelationships(t *testing.T) {
	_, accountsSvc, relSvc, cleanup := setupRelationshipsTestEnv(t)
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

	// Follow
	relSvc.Follow(context.Background(), account1.ID, account2.ID)

	getAccountID := func(c *mizu.Ctx) string {
		return account1.ID
	}

	h := handler.NewRelationship(relSvc, getAccountID)

	rec, ctx := testRequest("GET", "/api/v1/accounts/relationships?id[]="+account2.ID, nil, account1.ID)
	ctx.Request().URL.RawQuery = "id[]=" + account2.ID

	if err := h.GetRelationships(ctx); err != nil {
		t.Fatalf("GetRelationships() error = %v", err)
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
		t.Error("expected at least one relationship")
	}
}

func TestRelationship_GetRelationshipsEmpty(t *testing.T) {
	_, accountsSvc, relSvc, cleanup := setupRelationshipsTestEnv(t)
	defer cleanup()

	account1, _ := accountsSvc.Create(context.Background(), &accounts.CreateIn{
		Username: "user1",
		Email:    "user1@example.com",
		Password: "password123",
	})

	getAccountID := func(c *mizu.Ctx) string {
		return account1.ID
	}

	h := handler.NewRelationship(relSvc, getAccountID)

	rec, ctx := testRequest("GET", "/api/v1/accounts/relationships", nil, account1.ID)

	if err := h.GetRelationships(ctx); err != nil {
		t.Fatalf("GetRelationships() error = %v", err)
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

	if len(data) != 0 {
		t.Errorf("expected empty array, got %d items", len(data))
	}
}
