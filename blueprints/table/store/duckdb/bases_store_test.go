package duckdb_test

import (
	"context"
	"testing"

	"github.com/go-mizu/blueprints/table/feature/bases"
	"github.com/go-mizu/blueprints/table/feature/fields"
	"github.com/go-mizu/blueprints/table/feature/records"
	"github.com/go-mizu/blueprints/table/feature/tables"
	"github.com/go-mizu/blueprints/table/feature/views"
	"github.com/go-mizu/blueprints/table/pkg/ulid"
)

func TestBasesStore(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()
	user := createTestUser(t, store)
	ws := createTestWorkspace(t, store, user)

	t.Run("Create defaults and Update", func(t *testing.T) {
		base := &bases.Base{
			ID:          ulid.New(),
			WorkspaceID: ws.ID,
			Name:        "Marketing",
			CreatedBy:   user.ID,
		}

		if err := store.Bases().Create(ctx, base); err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		got, err := store.Bases().GetByID(ctx, base.ID)
		if err != nil {
			t.Fatalf("GetByID failed: %v", err)
		}
		if got.Color != "#2563EB" {
			t.Errorf("Expected default color, got %s", got.Color)
		}

		got.Name = "Marketing Ops"
		got.Description = "Ops planning"
		if err := store.Bases().Update(ctx, got); err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		updated, _ := store.Bases().GetByID(ctx, base.ID)
		if updated.Name != "Marketing Ops" {
			t.Errorf("Expected updated name, got %s", updated.Name)
		}
	})

	t.Run("ListByWorkspace ordering", func(t *testing.T) {
		baseA := &bases.Base{
			ID:          ulid.New(),
			WorkspaceID: ws.ID,
			Name:        "A Base",
			CreatedBy:   user.ID,
		}
		baseB := &bases.Base{
			ID:          ulid.New(),
			WorkspaceID: ws.ID,
			Name:        "B Base",
			CreatedBy:   user.ID,
		}
		if err := store.Bases().Create(ctx, baseB); err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		if err := store.Bases().Create(ctx, baseA); err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		list, err := store.Bases().ListByWorkspace(ctx, ws.ID)
		if err != nil {
			t.Fatalf("ListByWorkspace failed: %v", err)
		}
		if len(list) < 2 {
			t.Fatalf("Expected at least 2 bases, got %d", len(list))
		}
		if list[0].Name != "A Base" || list[1].Name != "B Base" {
			t.Errorf("Expected name ordering, got %s then %s", list[0].Name, list[1].Name)
		}
	})

	t.Run("Delete cascades table data", func(t *testing.T) {
		base := createTestBase(t, store, ws, user)
		tbl := createTestTable(t, store, base, user)
		field := createTestField(t, store, tbl, "Title", fields.TypeSingleLineText, user)
		rec := createTestRecord(t, store, tbl, user, map[string]any{field.ID: "Row 1"})
		view := createTestView(t, store, tbl, user, "All")

		if err := store.Bases().Delete(ctx, base.ID); err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		if _, err := store.Tables().GetByID(ctx, tbl.ID); err != tables.ErrNotFound {
			t.Errorf("Expected table to be deleted, got %v", err)
		}
		if _, err := store.Fields().GetByID(ctx, field.ID); err != fields.ErrNotFound {
			t.Errorf("Expected field to be deleted, got %v", err)
		}
		if _, err := store.Records().GetByID(ctx, rec.ID); err != records.ErrNotFound {
			t.Errorf("Expected record to be deleted, got %v", err)
		}
		if _, err := store.Views().GetByID(ctx, view.ID); err != views.ErrNotFound {
			t.Errorf("Expected view to be deleted, got %v", err)
		}
	})
}
