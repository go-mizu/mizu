package duckdb_test

import (
	"context"
	"testing"

	"github.com/go-mizu/blueprints/table/feature/fields"
	"github.com/go-mizu/blueprints/table/feature/records"
	"github.com/go-mizu/blueprints/table/feature/tables"
	"github.com/go-mizu/blueprints/table/feature/views"
	"github.com/go-mizu/blueprints/table/pkg/ulid"
)

func TestTablesStoreBehaviors(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()
	user := createTestUser(t, store)
	ws := createTestWorkspace(t, store, user)
	base := createTestBase(t, store, ws, user)

	t.Run("Positions and ListByBase ordering", func(t *testing.T) {
		first := &tables.Table{
			ID:        ulid.New(),
			BaseID:    base.ID,
			Name:      "First",
			CreatedBy: user.ID,
		}
		second := &tables.Table{
			ID:        ulid.New(),
			BaseID:    base.ID,
			Name:      "Second",
			CreatedBy: user.ID,
		}
		if err := store.Tables().Create(ctx, first); err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		if err := store.Tables().Create(ctx, second); err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		if second.Position <= first.Position {
			t.Errorf("Expected positions to increase, got %d then %d", first.Position, second.Position)
		}

		list, err := store.Tables().ListByBase(ctx, base.ID)
		if err != nil {
			t.Fatalf("ListByBase failed: %v", err)
		}
		if len(list) < 2 {
			t.Fatalf("Expected at least 2 tables, got %d", len(list))
		}
		if list[0].ID != first.ID || list[1].ID != second.ID {
			t.Errorf("Unexpected ordering in ListByBase")
		}
	})

	t.Run("SetPrimaryField and Update", func(t *testing.T) {
		tbl := createTestTable(t, store, base, user)
		field := createTestField(t, store, tbl, "Name", fields.TypeSingleLineText, user)

		if err := store.Tables().SetPrimaryField(ctx, tbl.ID, field.ID); err != nil {
			t.Fatalf("SetPrimaryField failed: %v", err)
		}

		got, err := store.Tables().GetByID(ctx, tbl.ID)
		if err != nil {
			t.Fatalf("GetByID failed: %v", err)
		}
		if got.PrimaryFieldID != field.ID {
			t.Errorf("Expected primary field %s, got %s", field.ID, got.PrimaryFieldID)
		}

		got.Description = "Updated description"
		if err := store.Tables().Update(ctx, got); err != nil {
			t.Fatalf("Update failed: %v", err)
		}
		updated, _ := store.Tables().GetByID(ctx, tbl.ID)
		if updated.Description != "Updated description" {
			t.Errorf("Expected description update, got %s", updated.Description)
		}
	})

	t.Run("Delete cascades table data", func(t *testing.T) {
		tbl := createTestTable(t, store, base, user)
		field := createTestField(t, store, tbl, "Title", fields.TypeSingleLineText, user)
		rec := createTestRecord(t, store, tbl, user, map[string]any{field.ID: "Row"})
		view := createTestView(t, store, tbl, user, "All")

		if err := store.Tables().Delete(ctx, tbl.ID); err != nil {
			t.Fatalf("Delete failed: %v", err)
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
