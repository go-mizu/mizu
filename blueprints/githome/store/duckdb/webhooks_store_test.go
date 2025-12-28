package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/webhooks"
	"github.com/oklog/ulid/v2"
)

func createTestWebhook(t *testing.T, store *WebhooksStore, repoID string, events []string) *webhooks.Webhook {
	t.Helper()
	id := ulid.Make().String()
	w := &webhooks.Webhook{
		ID:          id,
		RepoID:      repoID,
		URL:         "https://example.com/webhook/" + id[len(id)-8:],
		Secret:      "secret123",
		ContentType: "application/json",
		Events:      events,
		Active:      true,
		InsecureSSL: false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := store.Create(context.Background(), w); err != nil {
		t.Fatalf("failed to create test webhook: %v", err)
	}
	return w
}

// =============================================================================
// Webhook CRUD Tests
// =============================================================================

func TestWebhooksStore_Create(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	webhooksStore := NewWebhooksStore(store.DB())

	w := &webhooks.Webhook{
		ID:          ulid.Make().String(),
		RepoID:      repoID,
		URL:         "https://example.com/webhook",
		Secret:      "mysecret",
		ContentType: "application/json",
		Events:      []string{"push", "pull_request"},
		Active:      true,
		InsecureSSL: false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := webhooksStore.Create(context.Background(), w)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := webhooksStore.GetByID(context.Background(), w.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected webhook to be created")
	}
	if got.URL != "https://example.com/webhook" {
		t.Errorf("got URL %q, want %q", got.URL, "https://example.com/webhook")
	}
	if len(got.Events) != 2 {
		t.Errorf("got %d events, want 2", len(got.Events))
	}
}

func TestWebhooksStore_Create_WithOrgID(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgsStore := NewOrgsStore(store.DB())
	webhooksStore := NewWebhooksStore(store.DB())

	org := createTestOrg(t, orgsStore)

	w := &webhooks.Webhook{
		ID:          ulid.Make().String(),
		OrgID:       org.ID,
		URL:         "https://example.com/org-webhook",
		Secret:      "orgsecret",
		ContentType: "application/json",
		Events:      []string{"*"},
		Active:      true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	webhooksStore.Create(context.Background(), w)

	got, _ := webhooksStore.GetByID(context.Background(), w.ID)
	if got.OrgID != org.ID {
		t.Errorf("got org_id %q, want %q", got.OrgID, org.ID)
	}
}

func TestWebhooksStore_GetByID(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	webhooksStore := NewWebhooksStore(store.DB())

	w := createTestWebhook(t, webhooksStore, repoID, []string{"push"})

	got, err := webhooksStore.GetByID(context.Background(), w.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected webhook")
	}
	if got.ID != w.ID {
		t.Errorf("got ID %q, want %q", got.ID, w.ID)
	}
}

func TestWebhooksStore_GetByID_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	webhooksStore := NewWebhooksStore(store.DB())

	got, err := webhooksStore.GetByID(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent webhook")
	}
}

func TestWebhooksStore_Update(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	webhooksStore := NewWebhooksStore(store.DB())

	w := createTestWebhook(t, webhooksStore, repoID, []string{"push"})

	w.URL = "https://updated.example.com/webhook"
	w.Events = []string{"push", "pull_request", "issues"}
	w.Active = false

	err := webhooksStore.Update(context.Background(), w)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, _ := webhooksStore.GetByID(context.Background(), w.ID)
	if got.URL != "https://updated.example.com/webhook" {
		t.Errorf("got URL %q, want %q", got.URL, "https://updated.example.com/webhook")
	}
	if len(got.Events) != 3 {
		t.Errorf("got %d events, want 3", len(got.Events))
	}
	if got.Active {
		t.Error("expected webhook to be inactive")
	}
}

func TestWebhooksStore_Delete(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	webhooksStore := NewWebhooksStore(store.DB())

	w := createTestWebhook(t, webhooksStore, repoID, []string{"push"})

	err := webhooksStore.Delete(context.Background(), w.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	got, _ := webhooksStore.GetByID(context.Background(), w.ID)
	if got != nil {
		t.Error("expected webhook to be deleted")
	}
}

// =============================================================================
// List Tests
// =============================================================================

func TestWebhooksStore_ListByRepo(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	webhooksStore := NewWebhooksStore(store.DB())

	for i := 0; i < 5; i++ {
		createTestWebhook(t, webhooksStore, repoID, []string{"push"})
	}

	list, err := webhooksStore.ListByRepo(context.Background(), repoID, 10, 0)
	if err != nil {
		t.Fatalf("ListByRepo failed: %v", err)
	}
	if len(list) != 5 {
		t.Errorf("got %d webhooks, want 5", len(list))
	}
}

func TestWebhooksStore_ListByRepo_Pagination(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	webhooksStore := NewWebhooksStore(store.DB())

	for i := 0; i < 10; i++ {
		createTestWebhook(t, webhooksStore, repoID, []string{"push"})
	}

	page1, _ := webhooksStore.ListByRepo(context.Background(), repoID, 3, 0)
	page2, _ := webhooksStore.ListByRepo(context.Background(), repoID, 3, 3)

	if len(page1) != 3 {
		t.Errorf("got %d webhooks on page 1, want 3", len(page1))
	}
	if len(page2) != 3 {
		t.Errorf("got %d webhooks on page 2, want 3", len(page2))
	}
}

func TestWebhooksStore_ListByOrg(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgsStore := NewOrgsStore(store.DB())
	webhooksStore := NewWebhooksStore(store.DB())

	org := createTestOrg(t, orgsStore)

	for i := 0; i < 3; i++ {
		w := &webhooks.Webhook{
			ID:          ulid.Make().String(),
			OrgID:       org.ID,
			URL:         "https://example.com/webhook",
			Secret:      "secret",
			ContentType: "application/json",
			Events:      []string{"*"},
			Active:      true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		webhooksStore.Create(context.Background(), w)
	}

	list, err := webhooksStore.ListByOrg(context.Background(), org.ID, 10, 0)
	if err != nil {
		t.Fatalf("ListByOrg failed: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("got %d webhooks, want 3", len(list))
	}
}

func TestWebhooksStore_ListByEvent(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	webhooksStore := NewWebhooksStore(store.DB())

	// Create webhooks with different events
	createTestWebhook(t, webhooksStore, repoID, []string{"push"})
	createTestWebhook(t, webhooksStore, repoID, []string{"push", "pull_request"})
	createTestWebhook(t, webhooksStore, repoID, []string{"issues"})
	createTestWebhook(t, webhooksStore, repoID, []string{"*"}) // Wildcard

	list, err := webhooksStore.ListByEvent(context.Background(), repoID, "push")
	if err != nil {
		t.Fatalf("ListByEvent failed: %v", err)
	}
	// Should match: push, push+pull_request, and wildcard
	if len(list) != 3 {
		t.Errorf("got %d webhooks for push event, want 3", len(list))
	}
}

func TestWebhooksStore_ListByEvent_Wildcard(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	webhooksStore := NewWebhooksStore(store.DB())

	// Create webhook with wildcard
	createTestWebhook(t, webhooksStore, repoID, []string{"*"})

	list, _ := webhooksStore.ListByEvent(context.Background(), repoID, "any_event")
	if len(list) != 1 {
		t.Errorf("wildcard should match any event, got %d", len(list))
	}
}

func TestWebhooksStore_ListByEvent_OnlyActive(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	webhooksStore := NewWebhooksStore(store.DB())

	// Create active webhook
	createTestWebhook(t, webhooksStore, repoID, []string{"push"})

	// Create inactive webhook
	inactiveWebhook := createTestWebhook(t, webhooksStore, repoID, []string{"push"})
	inactiveWebhook.Active = false
	webhooksStore.Update(context.Background(), inactiveWebhook)

	list, _ := webhooksStore.ListByEvent(context.Background(), repoID, "push")
	if len(list) != 1 {
		t.Errorf("should only return active webhooks, got %d", len(list))
	}
}

// =============================================================================
// Delivery Tests
// =============================================================================

func TestWebhooksStore_CreateDelivery(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	webhooksStore := NewWebhooksStore(store.DB())

	w := createTestWebhook(t, webhooksStore, repoID, []string{"push"})

	d := &webhooks.Delivery{
		ID:             ulid.Make().String(),
		WebhookID:      w.ID,
		Event:          "push",
		GUID:           "abc-123",
		Payload:        `{"ref": "refs/heads/main"}`,
		RequestHeaders: "Content-Type: application/json",
		Delivered:      false,
		DurationMS:     0,
		CreatedAt:      time.Now(),
	}

	err := webhooksStore.CreateDelivery(context.Background(), d)
	if err != nil {
		t.Fatalf("CreateDelivery failed: %v", err)
	}

	got, _ := webhooksStore.GetDelivery(context.Background(), d.ID)
	if got == nil {
		t.Fatal("expected delivery")
	}
	if got.Event != "push" {
		t.Errorf("got event %q, want %q", got.Event, "push")
	}
}

func TestWebhooksStore_GetDelivery(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	webhooksStore := NewWebhooksStore(store.DB())

	w := createTestWebhook(t, webhooksStore, repoID, []string{"push"})

	d := &webhooks.Delivery{
		ID:             ulid.Make().String(),
		WebhookID:      w.ID,
		Event:          "push",
		GUID:           "xyz-789",
		Payload:        `{}`,
		RequestHeaders: "",
		CreatedAt:      time.Now(),
	}
	webhooksStore.CreateDelivery(context.Background(), d)

	got, err := webhooksStore.GetDelivery(context.Background(), d.ID)
	if err != nil {
		t.Fatalf("GetDelivery failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected delivery")
	}
	if got.GUID != "xyz-789" {
		t.Errorf("got GUID %q, want %q", got.GUID, "xyz-789")
	}
}

func TestWebhooksStore_GetDelivery_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	webhooksStore := NewWebhooksStore(store.DB())

	got, err := webhooksStore.GetDelivery(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("GetDelivery failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent delivery")
	}
}

func TestWebhooksStore_UpdateDelivery(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	webhooksStore := NewWebhooksStore(store.DB())

	w := createTestWebhook(t, webhooksStore, repoID, []string{"push"})

	d := &webhooks.Delivery{
		ID:             ulid.Make().String(),
		WebhookID:      w.ID,
		Event:          "push",
		GUID:           "test-guid",
		Payload:        `{}`,
		RequestHeaders: "",
		Delivered:      false,
		CreatedAt:      time.Now(),
	}
	webhooksStore.CreateDelivery(context.Background(), d)

	// Update with response
	d.ResponseHeaders = "Content-Type: application/json"
	d.ResponseBody = `{"ok": true}`
	d.StatusCode = 200
	d.Delivered = true
	d.DurationMS = 150

	err := webhooksStore.UpdateDelivery(context.Background(), d)
	if err != nil {
		t.Fatalf("UpdateDelivery failed: %v", err)
	}

	got, _ := webhooksStore.GetDelivery(context.Background(), d.ID)
	if got.StatusCode != 200 {
		t.Errorf("got status_code %d, want 200", got.StatusCode)
	}
	if !got.Delivered {
		t.Error("expected delivery to be marked as delivered")
	}
	if got.DurationMS != 150 {
		t.Errorf("got duration_ms %d, want 150", got.DurationMS)
	}
}

func TestWebhooksStore_ListDeliveries(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	webhooksStore := NewWebhooksStore(store.DB())

	w := createTestWebhook(t, webhooksStore, repoID, []string{"push"})

	for i := 0; i < 5; i++ {
		d := &webhooks.Delivery{
			ID:             ulid.Make().String(),
			WebhookID:      w.ID,
			Event:          "push",
			GUID:           "guid-" + string(rune('0'+i)),
			Payload:        `{}`,
			RequestHeaders: "",
			CreatedAt:      time.Now(),
		}
		webhooksStore.CreateDelivery(context.Background(), d)
	}

	list, err := webhooksStore.ListDeliveries(context.Background(), w.ID, 10, 0)
	if err != nil {
		t.Fatalf("ListDeliveries failed: %v", err)
	}
	if len(list) != 5 {
		t.Errorf("got %d deliveries, want 5", len(list))
	}
}

func TestWebhooksStore_ListDeliveries_Pagination(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	webhooksStore := NewWebhooksStore(store.DB())

	w := createTestWebhook(t, webhooksStore, repoID, []string{"push"})

	for i := 0; i < 10; i++ {
		d := &webhooks.Delivery{
			ID:             ulid.Make().String(),
			WebhookID:      w.ID,
			Event:          "push",
			GUID:           "guid-" + string(rune('0'+i)),
			Payload:        `{}`,
			RequestHeaders: "",
			CreatedAt:      time.Now(),
		}
		webhooksStore.CreateDelivery(context.Background(), d)
	}

	page1, _ := webhooksStore.ListDeliveries(context.Background(), w.ID, 3, 0)
	page2, _ := webhooksStore.ListDeliveries(context.Background(), w.ID, 3, 3)

	if len(page1) != 3 {
		t.Errorf("got %d deliveries on page 1, want 3", len(page1))
	}
	if len(page2) != 3 {
		t.Errorf("got %d deliveries on page 2, want 3", len(page2))
	}
}

func TestWebhooksStore_DeleteWebhookDeletesDeliveries(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	webhooksStore := NewWebhooksStore(store.DB())

	w := createTestWebhook(t, webhooksStore, repoID, []string{"push"})

	// Create deliveries
	for i := 0; i < 3; i++ {
		d := &webhooks.Delivery{
			ID:             ulid.Make().String(),
			WebhookID:      w.ID,
			Event:          "push",
			GUID:           "guid-" + string(rune('0'+i)),
			Payload:        `{}`,
			RequestHeaders: "",
			CreatedAt:      time.Now(),
		}
		webhooksStore.CreateDelivery(context.Background(), d)
	}

	// Delete webhook
	webhooksStore.Delete(context.Background(), w.ID)

	// Deliveries should also be deleted
	list, _ := webhooksStore.ListDeliveries(context.Background(), w.ID, 10, 0)
	if len(list) != 0 {
		t.Errorf("expected deliveries to be deleted, got %d", len(list))
	}
}

// Verify interface compliance
var _ webhooks.Store = (*WebhooksStore)(nil)
