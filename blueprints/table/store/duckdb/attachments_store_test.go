package duckdb_test

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/table/feature/attachments"
	"github.com/go-mizu/blueprints/table/feature/fields"
	"github.com/go-mizu/blueprints/table/pkg/ulid"
)

func TestAttachmentsStore(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()
	user := createTestUser(t, store)
	ws := createTestWorkspace(t, store, user)
	base := createTestBase(t, store, ws, user)
	tbl := createTestTable(t, store, base, user)
	field := createTestField(t, store, tbl, "Files", fields.TypeAttachment, user)
	rec := createTestRecord(t, store, tbl, user, map[string]any{})

	t.Run("Create and GetByID", func(t *testing.T) {
		att := &attachments.Attachment{
			ID:           ulid.New(),
			RecordID:     rec.ID,
			FieldID:      field.ID,
			Filename:     "doc.txt",
			Size:         10,
			MimeType:     "text/plain",
			URL:          "https://example.com/doc.txt",
			ThumbnailURL: "https://example.com/doc-thumb.txt",
			Width:        640,
			Height:       480,
		}

		if err := store.Attachments().Create(ctx, att); err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		got, err := store.Attachments().GetByID(ctx, att.ID)
		if err != nil {
			t.Fatalf("GetByID failed: %v", err)
		}
		if got.Filename != att.Filename {
			t.Errorf("Expected filename %s, got %s", att.Filename, got.Filename)
		}
	})

	t.Run("ListByRecord and DeleteByRecord", func(t *testing.T) {
		first := &attachments.Attachment{
			ID:       ulid.New(),
			RecordID: rec.ID,
			FieldID:  field.ID,
			Filename: "first.txt",
			Size:     1,
			MimeType: "text/plain",
			URL:      "https://example.com/first.txt",
		}
		second := &attachments.Attachment{
			ID:       ulid.New(),
			RecordID: rec.ID,
			FieldID:  field.ID,
			Filename: "second.txt",
			Size:     2,
			MimeType: "text/plain",
			URL:      "https://example.com/second.txt",
		}

		if err := store.Attachments().Create(ctx, first); err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		time.Sleep(5 * time.Millisecond)
		if err := store.Attachments().Create(ctx, second); err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		list, err := store.Attachments().ListByRecord(ctx, rec.ID, field.ID)
		if err != nil {
			t.Fatalf("ListByRecord failed: %v", err)
		}
		if len(list) < 2 {
			t.Fatalf("Expected at least 2 attachments, got %d", len(list))
		}

		if err := store.Attachments().DeleteByRecord(ctx, rec.ID); err != nil {
			t.Fatalf("DeleteByRecord failed: %v", err)
		}
		if _, err := store.Attachments().GetByID(ctx, first.ID); err != attachments.ErrNotFound {
			t.Errorf("Expected ErrNotFound, got %v", err)
		}
	})
}
