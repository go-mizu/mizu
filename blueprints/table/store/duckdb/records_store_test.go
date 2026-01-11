package duckdb_test

import (
	"context"
	"testing"

	"github.com/go-mizu/blueprints/table/feature/attachments"
	"github.com/go-mizu/blueprints/table/feature/comments"
	"github.com/go-mizu/blueprints/table/feature/fields"
	"github.com/go-mizu/blueprints/table/feature/records"
	"github.com/go-mizu/blueprints/table/pkg/ulid"
)

func TestRecordsStoreBehaviors(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()
	user := createTestUser(t, store)
	ws := createTestWorkspace(t, store, user)
	base := createTestBase(t, store, ws, user)
	tbl := createTestTable(t, store, base, user)
	titleField := createTestField(t, store, tbl, "Title", fields.TypeSingleLineText, user)
	linkField := createTestField(t, store, tbl, "Link", fields.TypeLink, user)

	t.Run("CreateBatch and GetByIDs", func(t *testing.T) {
		rec1 := &records.Record{
			ID:        ulid.New(),
			TableID:   tbl.ID,
			Cells:     map[string]any{titleField.ID: "One"},
			CreatedBy: user.ID,
		}
		rec2 := &records.Record{
			ID:        ulid.New(),
			TableID:   tbl.ID,
			Cells:     map[string]any{titleField.ID: "Two"},
			CreatedBy: user.ID,
		}

		if err := store.Records().CreateBatch(ctx, []*records.Record{rec1, rec2}); err != nil {
			t.Fatalf("CreateBatch failed: %v", err)
		}

		got, err := store.Records().GetByIDs(ctx, []string{rec1.ID, rec2.ID})
		if err != nil {
			t.Fatalf("GetByIDs failed: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("Expected 2 records, got %d", len(got))
		}
	})

	t.Run("Update and DeleteBatch", func(t *testing.T) {
		rec := createTestRecord(t, store, tbl, user, map[string]any{titleField.ID: "Original"})
		rec.Cells[titleField.ID] = "Updated"
		rec.Position = 10
		rec.UpdatedBy = user.ID

		if err := store.Records().Update(ctx, rec); err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		updated, err := store.Records().GetByID(ctx, rec.ID)
		if err != nil {
			t.Fatalf("GetByID failed: %v", err)
		}
		if updated.Position != 10 {
			t.Errorf("Expected position 10, got %d", updated.Position)
		}
		if updated.Cells[titleField.ID] != "Updated" {
			t.Errorf("Expected updated cell, got %v", updated.Cells[titleField.ID])
		}

		rec2 := createTestRecord(t, store, tbl, user, map[string]any{titleField.ID: "Delete"})
		if err := store.Records().DeleteBatch(ctx, []string{rec.ID, rec2.ID}); err != nil {
			t.Fatalf("DeleteBatch failed: %v", err)
		}
		if _, err := store.Records().GetByID(ctx, rec.ID); err != records.ErrNotFound {
			t.Errorf("Expected ErrNotFound, got %v", err)
		}
	})

	t.Run("Record links lifecycle", func(t *testing.T) {
		source := createTestRecord(t, store, tbl, user, map[string]any{titleField.ID: "Source"})
		target := createTestRecord(t, store, tbl, user, map[string]any{titleField.ID: "Target"})

		link := &records.RecordLink{
			SourceRecordID: source.ID,
			SourceFieldID:  linkField.ID,
			TargetRecordID: target.ID,
			Position:       0,
		}
		if err := store.Records().CreateLink(ctx, link); err != nil {
			t.Fatalf("CreateLink failed: %v", err)
		}
		if link.ID == "" {
			t.Fatalf("Expected link ID to be set")
		}

		bySource, err := store.Records().ListLinksBySource(ctx, source.ID, linkField.ID)
		if err != nil {
			t.Fatalf("ListLinksBySource failed: %v", err)
		}
		if len(bySource) != 1 {
			t.Fatalf("Expected 1 link, got %d", len(bySource))
		}

		byTarget, err := store.Records().ListLinksByTarget(ctx, target.ID)
		if err != nil {
			t.Fatalf("ListLinksByTarget failed: %v", err)
		}
		if len(byTarget) != 1 {
			t.Fatalf("Expected 1 link, got %d", len(byTarget))
		}

		if err := store.Records().DeleteLinksBySource(ctx, source.ID, linkField.ID); err != nil {
			t.Fatalf("DeleteLinksBySource failed: %v", err)
		}
		bySource, _ = store.Records().ListLinksBySource(ctx, source.ID, linkField.ID)
		if len(bySource) != 0 {
			t.Errorf("Expected links to be deleted")
		}
	})

	t.Run("Delete cleans related data", func(t *testing.T) {
		rec := createTestRecord(t, store, tbl, user, map[string]any{titleField.ID: "Has data"})

		comment := &comments.Comment{
			ID:       ulid.New(),
			RecordID: rec.ID,
			UserID:   user.ID,
			Content:  "Note",
		}
		if err := store.Comments().Create(ctx, comment); err != nil {
			t.Fatalf("Create comment failed: %v", err)
		}

		att := &attachments.Attachment{
			ID:       ulid.New(),
			RecordID: rec.ID,
			FieldID:  titleField.ID,
			Filename: "file.txt",
			Size:     12,
			MimeType: "text/plain",
			URL:      "https://example.com/file.txt",
		}
		if err := store.Attachments().Create(ctx, att); err != nil {
			t.Fatalf("Create attachment failed: %v", err)
		}

		link := &records.RecordLink{
			SourceRecordID: rec.ID,
			SourceFieldID:  linkField.ID,
			TargetRecordID: rec.ID,
		}
		if err := store.Records().CreateLink(ctx, link); err != nil {
			t.Fatalf("CreateLink failed: %v", err)
		}

		if err := store.Records().Delete(ctx, rec.ID); err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
		if _, err := store.Comments().GetByID(ctx, comment.ID); err != comments.ErrNotFound {
			t.Errorf("Expected comment to be deleted, got %v", err)
		}
		if _, err := store.Attachments().GetByID(ctx, att.ID); err != attachments.ErrNotFound {
			t.Errorf("Expected attachment to be deleted, got %v", err)
		}
		links, _ := store.Records().ListLinksByTarget(ctx, rec.ID)
		if len(links) != 0 {
			t.Errorf("Expected links to be deleted")
		}
	})
}
