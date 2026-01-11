package duckdb_test

import (
	"context"
	"testing"

	"github.com/go-mizu/blueprints/table/feature/comments"
	"github.com/go-mizu/blueprints/table/feature/fields"
	"github.com/go-mizu/blueprints/table/pkg/ulid"
)

func TestCommentsStoreBehaviors(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()
	user := createTestUser(t, store)
	ws := createTestWorkspace(t, store, user)
	base := createTestBase(t, store, ws, user)
	tbl := createTestTable(t, store, base, user)
	field := createTestField(t, store, tbl, "Title", fields.TypeSingleLineText, user)
	rec := createTestRecord(t, store, tbl, user, map[string]any{field.ID: "Row"})

	t.Run("Parent comments and update", func(t *testing.T) {
		parent := &comments.Comment{
			ID:       ulid.New(),
			RecordID: rec.ID,
			UserID:   user.ID,
			Content:  "Parent",
		}
		if err := store.Comments().Create(ctx, parent); err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		child := &comments.Comment{
			ID:       ulid.New(),
			RecordID: rec.ID,
			ParentID: parent.ID,
			UserID:   user.ID,
			Content:  "Child",
		}
		if err := store.Comments().Create(ctx, child); err != nil {
			t.Fatalf("Create child failed: %v", err)
		}

		child.Content = "Updated"
		if err := store.Comments().Update(ctx, child); err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		got, err := store.Comments().GetByID(ctx, child.ID)
		if err != nil {
			t.Fatalf("GetByID failed: %v", err)
		}
		if got.Content != "Updated" {
			t.Errorf("Expected updated content, got %s", got.Content)
		}
		if got.ParentID != parent.ID {
			t.Errorf("Expected parent_id %s, got %s", parent.ID, got.ParentID)
		}
	})

	t.Run("DeleteByRecord", func(t *testing.T) {
		comment := &comments.Comment{
			ID:       ulid.New(),
			RecordID: rec.ID,
			UserID:   user.ID,
			Content:  "To delete",
		}
		if err := store.Comments().Create(ctx, comment); err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		if err := store.Comments().DeleteByRecord(ctx, rec.ID); err != nil {
			t.Fatalf("DeleteByRecord failed: %v", err)
		}
		if _, err := store.Comments().GetByID(ctx, comment.ID); err != comments.ErrNotFound {
			t.Errorf("Expected ErrNotFound, got %v", err)
		}
	})
}
