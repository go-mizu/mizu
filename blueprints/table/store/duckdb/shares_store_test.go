package duckdb_test

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/table/feature/shares"
	"github.com/go-mizu/blueprints/table/pkg/ulid"
)

func TestSharesStore(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()
	user := createTestUser(t, store)
	ws := createTestWorkspace(t, store, user)
	base := createTestBase(t, store, ws, user)
	tbl := createTestTable(t, store, base, user)
	view := createTestView(t, store, tbl, user, "Shared")

	t.Run("Create and GetByToken", func(t *testing.T) {
		expires := time.Now().Add(24 * time.Hour)
		share := &shares.Share{
			ID:         ulid.New(),
			BaseID:     base.ID,
			TableID:    tbl.ID,
			ViewID:     view.ID,
			Type:       shares.TypeLink,
			Permission: shares.PermRead,
			Token:      "token-" + ulid.New(),
			ExpiresAt:  &expires,
			CreatedBy:  user.ID,
		}

		if err := store.Shares().Create(ctx, share); err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		got, err := store.Shares().GetByToken(ctx, share.Token)
		if err != nil {
			t.Fatalf("GetByToken failed: %v", err)
		}
		if got.ID != share.ID {
			t.Errorf("Expected share ID %s, got %s", share.ID, got.ID)
		}
	})

	t.Run("ListByBase and ListByUser", func(t *testing.T) {
		share := &shares.Share{
			ID:         ulid.New(),
			BaseID:     base.ID,
			Type:       shares.TypeUser,
			Permission: shares.PermEdit,
			UserID:     user.ID,
			CreatedBy:  user.ID,
		}
		if err := store.Shares().Create(ctx, share); err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		byBase, err := store.Shares().ListByBase(ctx, base.ID)
		if err != nil {
			t.Fatalf("ListByBase failed: %v", err)
		}
		if len(byBase) == 0 {
			t.Errorf("Expected shares for base")
		}

		byUser, err := store.Shares().ListByUser(ctx, user.ID)
		if err != nil {
			t.Fatalf("ListByUser failed: %v", err)
		}
		if len(byUser) == 0 {
			t.Errorf("Expected shares for user")
		}
	})

	t.Run("Update and Delete", func(t *testing.T) {
		share := &shares.Share{
			ID:         ulid.New(),
			BaseID:     base.ID,
			Type:       shares.TypeEmail,
			Permission: shares.PermRead,
			Email:      "share@example.com",
			CreatedBy:  user.ID,
		}
		if err := store.Shares().Create(ctx, share); err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		expires := time.Now().Add(48 * time.Hour)
		share.Permission = shares.PermComment
		share.ExpiresAt = &expires
		if err := store.Shares().Update(ctx, share); err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		got, _ := store.Shares().GetByID(ctx, share.ID)
		if got.Permission != shares.PermComment {
			t.Errorf("Expected permission update, got %s", got.Permission)
		}
		if got.ExpiresAt == nil {
			t.Errorf("Expected expires_at to be set")
		}

		if err := store.Shares().Delete(ctx, share.ID); err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
		if _, err := store.Shares().GetByID(ctx, share.ID); err != shares.ErrNotFound {
			t.Errorf("Expected ErrNotFound, got %v", err)
		}
	})
}
