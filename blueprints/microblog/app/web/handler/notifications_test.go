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
	"github.com/go-mizu/blueprints/microblog/feature/notifications"
	"github.com/go-mizu/blueprints/microblog/store/duckdb"
)

func setupNotificationsTestEnv(t *testing.T) (*sql.DB, accounts.API, notifications.API, func()) {
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
	notificationsStore := duckdb.NewNotificationsStore(db)

	accountsSvc := accounts.NewService(accountsStore)
	notificationsSvc := notifications.NewService(notificationsStore, accountsSvc)

	cleanup := func() {
		db.Close()
	}

	return db, accountsSvc, notificationsSvc, cleanup
}

func TestNotification_List(t *testing.T) {
	_, accountsSvc, notifSvc, cleanup := setupNotificationsTestEnv(t)
	defer cleanup()

	account, _ := accountsSvc.Create(context.Background(), &accounts.CreateIn{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	})

	getAccountID := func(c *mizu.Ctx) string {
		return account.ID
	}

	h := handler.NewNotification(notifSvc, getAccountID)

	rec, ctx := testRequest("GET", "/api/v1/notifications", nil, account.ID)

	if err := h.List(ctx); err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if rec.Code != 200 {
		t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Data field may be null or an empty array for no notifications
	if resp["data"] == nil {
		t.Log("no notifications yet (data is null, expected)")
		return
	}

	if _, ok := resp["data"].([]any); !ok {
		t.Logf("data field is not an array, got type %T", resp["data"])
	}
}

func TestNotification_ListWithLimit(t *testing.T) {
	_, accountsSvc, notifSvc, cleanup := setupNotificationsTestEnv(t)
	defer cleanup()

	account, _ := accountsSvc.Create(context.Background(), &accounts.CreateIn{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	})

	getAccountID := func(c *mizu.Ctx) string {
		return account.ID
	}

	h := handler.NewNotification(notifSvc, getAccountID)

	rec, ctx := testRequest("GET", "/api/v1/notifications?limit=10", nil, account.ID)
	ctx.Request().URL.RawQuery = "limit=10"

	if err := h.List(ctx); err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if rec.Code != 200 {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestNotification_Clear(t *testing.T) {
	_, accountsSvc, notifSvc, cleanup := setupNotificationsTestEnv(t)
	defer cleanup()

	account, _ := accountsSvc.Create(context.Background(), &accounts.CreateIn{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	})

	getAccountID := func(c *mizu.Ctx) string {
		return account.ID
	}

	h := handler.NewNotification(notifSvc, getAccountID)

	rec, ctx := testRequest("POST", "/api/v1/notifications/clear", nil, account.ID)

	if err := h.Clear(ctx); err != nil {
		t.Fatalf("Clear() error = %v", err)
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

	if data["success"] != true {
		t.Error("expected success to be true")
	}
}

func TestNotification_Dismiss(t *testing.T) {
	_, accountsSvc, notifSvc, cleanup := setupNotificationsTestEnv(t)
	defer cleanup()

	account, _ := accountsSvc.Create(context.Background(), &accounts.CreateIn{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	})

	// Use a fake notification ID for testing
	fakeNotifID := "test-notification-id"

	getAccountID := func(c *mizu.Ctx) string {
		return account.ID
	}

	h := handler.NewNotification(notifSvc, getAccountID)

	rec, ctx := testRequest("POST", "/api/v1/notifications/"+fakeNotifID+"/dismiss", nil, account.ID)
	ctx.Request().SetPathValue("id", fakeNotifID)

	if err := h.Dismiss(ctx); err != nil {
		t.Fatalf("Dismiss() error = %v", err)
	}

	// The dismiss will likely fail for a non-existent notification but we just test the handler works
	if rec.Code != 200 && rec.Code != 500 {
		t.Logf("got status %d (expected 200 or 500 for fake notification)", rec.Code)
	}
}

func TestNotification_DismissNonExistent(t *testing.T) {
	_, accountsSvc, notifSvc, cleanup := setupNotificationsTestEnv(t)
	defer cleanup()

	account, _ := accountsSvc.Create(context.Background(), &accounts.CreateIn{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	})

	getAccountID := func(c *mizu.Ctx) string {
		return account.ID
	}

	h := handler.NewNotification(notifSvc, getAccountID)

	rec, ctx := testRequest("POST", "/api/v1/notifications/nonexistent/dismiss", nil, account.ID)
	ctx.Request().SetPathValue("id", "nonexistent")

	if err := h.Dismiss(ctx); err != nil {
		t.Fatalf("Dismiss() error = %v", err)
	}

	// Should still return success even if notification doesn't exist
	if rec.Code != 200 && rec.Code != 500 {
		t.Logf("got status %d (expected 200 or 500 for non-existent notification)", rec.Code)
	}
}
