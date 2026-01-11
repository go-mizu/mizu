package duckdb_test

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/table/feature/webhooks"
	"github.com/go-mizu/blueprints/table/pkg/ulid"
)

func TestWebhooksStore(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()
	user := createTestUser(t, store)
	ws := createTestWorkspace(t, store, user)
	base := createTestBase(t, store, ws, user)
	tbl := createTestTable(t, store, base, user)

	t.Run("Create and List", func(t *testing.T) {
		hook := &webhooks.Webhook{
			ID:        ulid.New(),
			BaseID:    base.ID,
			TableID:   tbl.ID,
			URL:       "https://example.com/webhook",
			Events:    []string{webhooks.EventRecordCreated},
			CreatedBy: user.ID,
		}
		if err := store.Webhooks().Create(ctx, hook); err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		got, err := store.Webhooks().GetByID(ctx, hook.ID)
		if err != nil {
			t.Fatalf("GetByID failed: %v", err)
		}
		if !got.IsActive {
			t.Errorf("Expected webhook to be active by default")
		}

		byBase, err := store.Webhooks().ListByBase(ctx, base.ID)
		if err != nil {
			t.Fatalf("ListByBase failed: %v", err)
		}
		if len(byBase) == 0 {
			t.Errorf("Expected webhooks for base")
		}

		byTable, err := store.Webhooks().ListByTable(ctx, tbl.ID)
		if err != nil {
			t.Fatalf("ListByTable failed: %v", err)
		}
		if len(byTable) == 0 {
			t.Errorf("Expected webhooks for table")
		}
	})

	t.Run("Update and Deliveries", func(t *testing.T) {
		hook := &webhooks.Webhook{
			ID:        ulid.New(),
			BaseID:    base.ID,
			URL:       "https://example.com/old",
			Events:    []string{webhooks.EventRecordUpdated},
			CreatedBy: user.ID,
		}
		if err := store.Webhooks().Create(ctx, hook); err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		hook.URL = "https://example.com/new"
		hook.IsActive = false
		if err := store.Webhooks().Update(ctx, hook); err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		updated, _ := store.Webhooks().GetByID(ctx, hook.ID)
		if updated.URL != "https://example.com/new" || updated.IsActive {
			t.Errorf("Expected webhook updates to persist")
		}

		first := &webhooks.Delivery{
			ID:         ulid.New(),
			WebhookID:  hook.ID,
			Event:      webhooks.EventRecordUpdated,
			Payload:    "{}",
			StatusCode: 200,
			Response:   "ok",
			DurationMs: 10,
		}
		if err := store.Webhooks().CreateDelivery(ctx, first); err != nil {
			t.Fatalf("CreateDelivery failed: %v", err)
		}
		time.Sleep(5 * time.Millisecond)
		second := &webhooks.Delivery{
			ID:         ulid.New(),
			WebhookID:  hook.ID,
			Event:      webhooks.EventRecordUpdated,
			Payload:    "{}",
			StatusCode: 500,
			Response:   "error",
			DurationMs: 5,
		}
		if err := store.Webhooks().CreateDelivery(ctx, second); err != nil {
			t.Fatalf("CreateDelivery failed: %v", err)
		}

		list, err := store.Webhooks().ListDeliveries(ctx, hook.ID, webhooks.ListOpts{Limit: 1})
		if err != nil {
			t.Fatalf("ListDeliveries failed: %v", err)
		}
		if len(list) != 1 || list[0].ID != second.ID {
			t.Errorf("Expected latest delivery first")
		}

		if err := store.Webhooks().Delete(ctx, hook.ID); err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
		if _, err := store.Webhooks().GetDelivery(ctx, second.ID); err != webhooks.ErrNotFound {
			t.Errorf("Expected delivery to be deleted, got %v", err)
		}
	})
}
