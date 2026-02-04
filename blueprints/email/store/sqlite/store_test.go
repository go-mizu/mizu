package sqlite

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/email/store"
	"github.com/go-mizu/mizu/blueprints/email/types"
	"github.com/google/uuid"
)

// setupTestStore creates a new in-memory SQLite store with schema applied.
func setupTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Ensure(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

// makeTestEmail creates a test email with the given subject and optional labels.
func makeTestEmail(subject string, labels ...string) *types.Email {
	id := uuid.New().String()
	threadID := uuid.New().String()
	return &types.Email{
		ID:          id,
		ThreadID:    threadID,
		MessageID:   fmt.Sprintf("<%s@test.local>", id),
		FromAddress: "sender@test.com",
		FromName:    "Test Sender",
		ToAddresses: []types.Recipient{{Address: "recipient@test.com", Name: "Test Recipient"}},
		Subject:     subject,
		BodyText:    "Test body text for " + subject,
		BodyHTML:    "<p>Test body HTML for " + subject + "</p>",
		Snippet:     "Test body text for " + subject,
		Labels:      labels,
		ReceivedAt:  time.Now(),
	}
}

// seedSystemLabels seeds system labels into the store for tests that need them.
func seedSystemLabels(t *testing.T, s *Store) {
	t.Helper()
	ctx := context.Background()
	labels := []types.Label{
		{ID: "inbox", Name: "Inbox", Type: types.LabelTypeSystem, Visible: true, Position: 0},
		{ID: "starred", Name: "Starred", Type: types.LabelTypeSystem, Visible: true, Position: 1},
		{ID: "important", Name: "Important", Type: types.LabelTypeSystem, Visible: true, Position: 2},
		{ID: "sent", Name: "Sent", Type: types.LabelTypeSystem, Visible: true, Position: 4},
		{ID: "drafts", Name: "Drafts", Type: types.LabelTypeSystem, Visible: true, Position: 5},
		{ID: "all", Name: "All Mail", Type: types.LabelTypeSystem, Visible: true, Position: 6},
		{ID: "spam", Name: "Spam", Type: types.LabelTypeSystem, Visible: true, Position: 7},
		{ID: "trash", Name: "Trash", Type: types.LabelTypeSystem, Visible: true, Position: 8},
	}
	for _, label := range labels {
		if err := s.CreateLabel(ctx, &label); err != nil {
			t.Fatalf("failed to seed label %s: %v", label.ID, err)
		}
	}
}

// ---------------------------------------------------------------------------
// Schema
// ---------------------------------------------------------------------------

func TestEnsure(t *testing.T) {
	t.Run("creates schema successfully", func(t *testing.T) {
		s, err := New(":memory:")
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()

		err = s.Ensure(context.Background())
		if err != nil {
			t.Fatalf("Ensure failed: %v", err)
		}
	})

	t.Run("idempotent - calling twice does not error", func(t *testing.T) {
		s, err := New(":memory:")
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()

		ctx := context.Background()
		if err := s.Ensure(ctx); err != nil {
			t.Fatalf("first Ensure failed: %v", err)
		}
		if err := s.Ensure(ctx); err != nil {
			t.Fatalf("second Ensure failed: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// Email CRUD
// ---------------------------------------------------------------------------

func TestCreateEmail(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	t.Run("creates email with provided ID", func(t *testing.T) {
		email := makeTestEmail("Test Subject")
		originalID := email.ID

		err := s.CreateEmail(ctx, email)
		if err != nil {
			t.Fatalf("CreateEmail failed: %v", err)
		}
		if email.ID != originalID {
			t.Errorf("expected ID %s, got %s", originalID, email.ID)
		}
	})

	t.Run("generates ID if empty", func(t *testing.T) {
		email := makeTestEmail("Auto ID")
		email.ID = ""
		email.MessageID = fmt.Sprintf("<%s@test.local>", uuid.New().String())

		err := s.CreateEmail(ctx, email)
		if err != nil {
			t.Fatalf("CreateEmail failed: %v", err)
		}
		if email.ID == "" {
			t.Error("expected non-empty ID after creation")
		}
	})

	t.Run("generates ThreadID if empty", func(t *testing.T) {
		email := makeTestEmail("Auto ThreadID")
		email.ThreadID = ""
		email.MessageID = fmt.Sprintf("<%s@test.local>", uuid.New().String())

		err := s.CreateEmail(ctx, email)
		if err != nil {
			t.Fatalf("CreateEmail failed: %v", err)
		}
		if email.ThreadID == "" {
			t.Error("expected non-empty ThreadID after creation")
		}
	})

	t.Run("calculates size bytes", func(t *testing.T) {
		email := makeTestEmail("Size Check")
		email.MessageID = fmt.Sprintf("<%s@test.local>", uuid.New().String())
		email.SizeBytes = 0

		err := s.CreateEmail(ctx, email)
		if err != nil {
			t.Fatalf("CreateEmail failed: %v", err)
		}
		if email.SizeBytes == 0 {
			t.Error("expected SizeBytes to be calculated")
		}
	})
}

func TestGetEmail(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()
	seedSystemLabels(t, s)

	t.Run("retrieves email with all fields", func(t *testing.T) {
		email := makeTestEmail("Get Test", "inbox")

		err := s.CreateEmail(ctx, email)
		if err != nil {
			t.Fatalf("CreateEmail failed: %v", err)
		}

		got, err := s.GetEmail(ctx, email.ID)
		if err != nil {
			t.Fatalf("GetEmail failed: %v", err)
		}

		if got.ID != email.ID {
			t.Errorf("ID: want %s, got %s", email.ID, got.ID)
		}
		if got.ThreadID != email.ThreadID {
			t.Errorf("ThreadID: want %s, got %s", email.ThreadID, got.ThreadID)
		}
		if got.MessageID != email.MessageID {
			t.Errorf("MessageID: want %s, got %s", email.MessageID, got.MessageID)
		}
		if got.FromAddress != email.FromAddress {
			t.Errorf("FromAddress: want %s, got %s", email.FromAddress, got.FromAddress)
		}
		if got.FromName != email.FromName {
			t.Errorf("FromName: want %s, got %s", email.FromName, got.FromName)
		}
		if got.Subject != email.Subject {
			t.Errorf("Subject: want %s, got %s", email.Subject, got.Subject)
		}
		if got.BodyText != email.BodyText {
			t.Errorf("BodyText: want %s, got %s", email.BodyText, got.BodyText)
		}
		if got.BodyHTML != email.BodyHTML {
			t.Errorf("BodyHTML: want %s, got %s", email.BodyHTML, got.BodyHTML)
		}
		if got.Snippet != email.Snippet {
			t.Errorf("Snippet: want %s, got %s", email.Snippet, got.Snippet)
		}
		if len(got.ToAddresses) != 1 {
			t.Fatalf("expected 1 ToAddress, got %d", len(got.ToAddresses))
		}
		if got.ToAddresses[0].Address != "recipient@test.com" {
			t.Errorf("ToAddress: want recipient@test.com, got %s", got.ToAddresses[0].Address)
		}
		if len(got.Labels) != 1 || got.Labels[0] != "inbox" {
			t.Errorf("Labels: want [inbox], got %v", got.Labels)
		}
	})

	t.Run("returns error for non-existent email", func(t *testing.T) {
		_, err := s.GetEmail(ctx, "non-existent-id")
		if err == nil {
			t.Error("expected error for non-existent email")
		}
	})
}

func TestListEmails(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()
	seedSystemLabels(t, s)

	// Create 5 emails with staggered received times
	for i := range 5 {
		email := makeTestEmail(fmt.Sprintf("Email %d", i), "inbox")
		email.ReceivedAt = time.Now().Add(-time.Duration(i) * time.Hour)
		if err := s.CreateEmail(ctx, email); err != nil {
			t.Fatalf("CreateEmail %d failed: %v", i, err)
		}
	}

	t.Run("returns all emails with default pagination", func(t *testing.T) {
		result, err := s.ListEmails(ctx, store.EmailFilter{})
		if err != nil {
			t.Fatalf("ListEmails failed: %v", err)
		}
		if result.Total != 5 {
			t.Errorf("Total: want 5, got %d", result.Total)
		}
		if len(result.Emails) != 5 {
			t.Errorf("Emails count: want 5, got %d", len(result.Emails))
		}
	})

	t.Run("respects pagination", func(t *testing.T) {
		result, err := s.ListEmails(ctx, store.EmailFilter{Page: 1, PerPage: 2})
		if err != nil {
			t.Fatalf("ListEmails failed: %v", err)
		}
		if len(result.Emails) != 2 {
			t.Errorf("Emails count: want 2, got %d", len(result.Emails))
		}
		if result.Total != 5 {
			t.Errorf("Total: want 5, got %d", result.Total)
		}
		if result.TotalPages != 3 {
			t.Errorf("TotalPages: want 3, got %d", result.TotalPages)
		}
		if result.Page != 1 {
			t.Errorf("Page: want 1, got %d", result.Page)
		}
	})

	t.Run("returns second page", func(t *testing.T) {
		result, err := s.ListEmails(ctx, store.EmailFilter{Page: 2, PerPage: 2})
		if err != nil {
			t.Fatalf("ListEmails failed: %v", err)
		}
		if len(result.Emails) != 2 {
			t.Errorf("Emails count: want 2, got %d", len(result.Emails))
		}
		if result.Page != 2 {
			t.Errorf("Page: want 2, got %d", result.Page)
		}
	})

	t.Run("returns last page with remaining items", func(t *testing.T) {
		result, err := s.ListEmails(ctx, store.EmailFilter{Page: 3, PerPage: 2})
		if err != nil {
			t.Fatalf("ListEmails failed: %v", err)
		}
		if len(result.Emails) != 1 {
			t.Errorf("Emails count: want 1, got %d", len(result.Emails))
		}
	})

	t.Run("returns emails ordered by received_at DESC", func(t *testing.T) {
		result, err := s.ListEmails(ctx, store.EmailFilter{})
		if err != nil {
			t.Fatalf("ListEmails failed: %v", err)
		}
		for i := 1; i < len(result.Emails); i++ {
			if result.Emails[i].ReceivedAt.After(result.Emails[i-1].ReceivedAt) {
				t.Error("emails are not ordered by received_at DESC")
				break
			}
		}
	})
}

func TestListEmailsWithLabelFilter(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()
	seedSystemLabels(t, s)

	// Create a user label
	workLabel := &types.Label{ID: "work", Name: "Work", Type: types.LabelTypeUser, Visible: true}
	if err := s.CreateLabel(ctx, workLabel); err != nil {
		t.Fatal(err)
	}

	// Create emails: 3 with "inbox", 2 with "work"
	for i := range 3 {
		email := makeTestEmail(fmt.Sprintf("Inbox email %d", i), "inbox")
		if err := s.CreateEmail(ctx, email); err != nil {
			t.Fatal(err)
		}
	}
	for i := range 2 {
		email := makeTestEmail(fmt.Sprintf("Work email %d", i), "work")
		if err := s.CreateEmail(ctx, email); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("filters by inbox label", func(t *testing.T) {
		result, err := s.ListEmails(ctx, store.EmailFilter{LabelID: "inbox"})
		if err != nil {
			t.Fatalf("ListEmails failed: %v", err)
		}
		if result.Total != 3 {
			t.Errorf("Total: want 3, got %d", result.Total)
		}
	})

	t.Run("filters by work label", func(t *testing.T) {
		result, err := s.ListEmails(ctx, store.EmailFilter{LabelID: "work"})
		if err != nil {
			t.Fatalf("ListEmails failed: %v", err)
		}
		if result.Total != 2 {
			t.Errorf("Total: want 2, got %d", result.Total)
		}
	})

	t.Run("returns empty for label with no emails", func(t *testing.T) {
		result, err := s.ListEmails(ctx, store.EmailFilter{LabelID: "trash"})
		if err != nil {
			t.Fatalf("ListEmails failed: %v", err)
		}
		if result.Total != 0 {
			t.Errorf("Total: want 0, got %d", result.Total)
		}
		if len(result.Emails) != 0 {
			t.Errorf("Emails: want empty, got %d", len(result.Emails))
		}
	})
}

func TestUpdateEmail(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()
	seedSystemLabels(t, s)

	t.Run("update is_read", func(t *testing.T) {
		email := makeTestEmail("Read Update", "inbox")
		if err := s.CreateEmail(ctx, email); err != nil {
			t.Fatal(err)
		}

		err := s.UpdateEmail(ctx, email.ID, map[string]any{"is_read": true})
		if err != nil {
			t.Fatalf("UpdateEmail failed: %v", err)
		}

		got, err := s.GetEmail(ctx, email.ID)
		if err != nil {
			t.Fatal(err)
		}
		if !got.IsRead {
			t.Error("expected is_read to be true")
		}
	})

	t.Run("update is_starred", func(t *testing.T) {
		email := makeTestEmail("Star Update", "inbox")
		if err := s.CreateEmail(ctx, email); err != nil {
			t.Fatal(err)
		}

		err := s.UpdateEmail(ctx, email.ID, map[string]any{"is_starred": true})
		if err != nil {
			t.Fatalf("UpdateEmail failed: %v", err)
		}

		got, err := s.GetEmail(ctx, email.ID)
		if err != nil {
			t.Fatal(err)
		}
		if !got.IsStarred {
			t.Error("expected is_starred to be true")
		}
	})

	t.Run("update is_important", func(t *testing.T) {
		email := makeTestEmail("Important Update", "inbox")
		if err := s.CreateEmail(ctx, email); err != nil {
			t.Fatal(err)
		}

		err := s.UpdateEmail(ctx, email.ID, map[string]any{"is_important": true})
		if err != nil {
			t.Fatalf("UpdateEmail failed: %v", err)
		}

		got, err := s.GetEmail(ctx, email.ID)
		if err != nil {
			t.Fatal(err)
		}
		if !got.IsImportant {
			t.Error("expected is_important to be true")
		}
	})

	t.Run("updates multiple fields at once", func(t *testing.T) {
		email := makeTestEmail("Multi Update", "inbox")
		if err := s.CreateEmail(ctx, email); err != nil {
			t.Fatal(err)
		}

		err := s.UpdateEmail(ctx, email.ID, map[string]any{
			"is_read":      true,
			"is_starred":   true,
			"is_important": true,
		})
		if err != nil {
			t.Fatalf("UpdateEmail failed: %v", err)
		}

		got, err := s.GetEmail(ctx, email.ID)
		if err != nil {
			t.Fatal(err)
		}
		if !got.IsRead {
			t.Error("expected is_read to be true")
		}
		if !got.IsStarred {
			t.Error("expected is_starred to be true")
		}
		if !got.IsImportant {
			t.Error("expected is_important to be true")
		}
	})

	t.Run("updates updated_at timestamp", func(t *testing.T) {
		email := makeTestEmail("Timestamp Update", "inbox")
		// Set a creation time clearly in the past so the update will produce a later timestamp.
		// RFC3339 has second-level granularity, so we need at least 1-second difference.
		email.ReceivedAt = time.Now().Add(-2 * time.Second)
		email.CreatedAt = time.Now().Add(-2 * time.Second)
		email.UpdatedAt = time.Now().Add(-2 * time.Second)
		if err := s.CreateEmail(ctx, email); err != nil {
			t.Fatal(err)
		}

		beforeUpdate, _ := s.GetEmail(ctx, email.ID)

		err := s.UpdateEmail(ctx, email.ID, map[string]any{"is_read": true})
		if err != nil {
			t.Fatal(err)
		}

		afterUpdate, _ := s.GetEmail(ctx, email.ID)
		if !afterUpdate.UpdatedAt.After(beforeUpdate.UpdatedAt) {
			t.Errorf("expected updated_at (%v) to be later than before (%v)",
				afterUpdate.UpdatedAt, beforeUpdate.UpdatedAt)
		}
	})
}

func TestDeleteEmail(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()
	seedSystemLabels(t, s)

	t.Run("soft delete moves to trash", func(t *testing.T) {
		email := makeTestEmail("Soft Delete", "inbox")
		if err := s.CreateEmail(ctx, email); err != nil {
			t.Fatal(err)
		}

		err := s.DeleteEmail(ctx, email.ID, false)
		if err != nil {
			t.Fatalf("DeleteEmail (soft) failed: %v", err)
		}

		// Email should still exist
		got, err := s.GetEmail(ctx, email.ID)
		if err != nil {
			t.Fatalf("GetEmail after soft delete failed: %v", err)
		}

		// Should have trash label
		hasTrash := false
		hasInbox := false
		for _, l := range got.Labels {
			if l == "trash" {
				hasTrash = true
			}
			if l == "inbox" {
				hasInbox = true
			}
		}
		if !hasTrash {
			t.Error("expected trash label after soft delete")
		}
		if hasInbox {
			t.Error("expected inbox label to be removed after soft delete")
		}
	})

	t.Run("permanent delete removes email", func(t *testing.T) {
		email := makeTestEmail("Permanent Delete", "inbox")
		if err := s.CreateEmail(ctx, email); err != nil {
			t.Fatal(err)
		}

		err := s.DeleteEmail(ctx, email.ID, true)
		if err != nil {
			t.Fatalf("DeleteEmail (permanent) failed: %v", err)
		}

		_, err = s.GetEmail(ctx, email.ID)
		if err == nil {
			t.Error("expected error when getting permanently deleted email")
		}
	})
}

func TestBatchUpdateEmails(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()
	seedSystemLabels(t, s)

	t.Run("batch archive removes inbox label", func(t *testing.T) {
		var ids []string
		for i := range 3 {
			email := makeTestEmail(fmt.Sprintf("Archive %d", i), "inbox")
			if err := s.CreateEmail(ctx, email); err != nil {
				t.Fatal(err)
			}
			ids = append(ids, email.ID)
		}

		err := s.BatchUpdateEmails(ctx, &types.BatchAction{
			IDs:    ids,
			Action: "archive",
		})
		if err != nil {
			t.Fatalf("BatchUpdateEmails (archive) failed: %v", err)
		}

		for _, id := range ids {
			got, err := s.GetEmail(ctx, id)
			if err != nil {
				t.Fatal(err)
			}
			for _, l := range got.Labels {
				if l == "inbox" {
					t.Errorf("email %s still has inbox label after archive", id)
				}
			}
		}
	})

	t.Run("batch trash moves to trash", func(t *testing.T) {
		var ids []string
		for i := range 2 {
			email := makeTestEmail(fmt.Sprintf("Trash %d", i), "inbox")
			if err := s.CreateEmail(ctx, email); err != nil {
				t.Fatal(err)
			}
			ids = append(ids, email.ID)
		}

		err := s.BatchUpdateEmails(ctx, &types.BatchAction{
			IDs:    ids,
			Action: "trash",
		})
		if err != nil {
			t.Fatalf("BatchUpdateEmails (trash) failed: %v", err)
		}

		for _, id := range ids {
			got, err := s.GetEmail(ctx, id)
			if err != nil {
				t.Fatal(err)
			}
			hasTrash := false
			for _, l := range got.Labels {
				if l == "trash" {
					hasTrash = true
				}
				if l == "inbox" {
					t.Errorf("email %s still has inbox label after trash", id)
				}
			}
			if !hasTrash {
				t.Errorf("email %s missing trash label", id)
			}
		}
	})

	t.Run("batch read marks as read", func(t *testing.T) {
		var ids []string
		for i := range 2 {
			email := makeTestEmail(fmt.Sprintf("Read %d", i), "inbox")
			if err := s.CreateEmail(ctx, email); err != nil {
				t.Fatal(err)
			}
			ids = append(ids, email.ID)
		}

		err := s.BatchUpdateEmails(ctx, &types.BatchAction{
			IDs:    ids,
			Action: "read",
		})
		if err != nil {
			t.Fatalf("BatchUpdateEmails (read) failed: %v", err)
		}

		for _, id := range ids {
			got, err := s.GetEmail(ctx, id)
			if err != nil {
				t.Fatal(err)
			}
			if !got.IsRead {
				t.Errorf("email %s is not marked as read", id)
			}
		}
	})

	t.Run("batch star marks as starred", func(t *testing.T) {
		var ids []string
		for i := range 2 {
			email := makeTestEmail(fmt.Sprintf("Star %d", i), "inbox")
			if err := s.CreateEmail(ctx, email); err != nil {
				t.Fatal(err)
			}
			ids = append(ids, email.ID)
		}

		err := s.BatchUpdateEmails(ctx, &types.BatchAction{
			IDs:    ids,
			Action: "star",
		})
		if err != nil {
			t.Fatalf("BatchUpdateEmails (star) failed: %v", err)
		}

		for _, id := range ids {
			got, err := s.GetEmail(ctx, id)
			if err != nil {
				t.Fatal(err)
			}
			if !got.IsStarred {
				t.Errorf("email %s is not starred", id)
			}
		}
	})

	t.Run("batch delete permanently removes emails", func(t *testing.T) {
		var ids []string
		for i := range 2 {
			email := makeTestEmail(fmt.Sprintf("Delete %d", i), "inbox")
			if err := s.CreateEmail(ctx, email); err != nil {
				t.Fatal(err)
			}
			ids = append(ids, email.ID)
		}

		err := s.BatchUpdateEmails(ctx, &types.BatchAction{
			IDs:    ids,
			Action: "delete",
		})
		if err != nil {
			t.Fatalf("BatchUpdateEmails (delete) failed: %v", err)
		}

		for _, id := range ids {
			_, err := s.GetEmail(ctx, id)
			if err == nil {
				t.Errorf("expected error getting deleted email %s", id)
			}
		}
	})
}

func TestSearchEmails(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	// Create emails with distinct subjects and bodies
	emails := []*types.Email{
		makeTestEmail("Quarterly Budget Report"),
		makeTestEmail("Weekly Team Standup Notes"),
		makeTestEmail("Database Migration Plan"),
	}
	emails[0].BodyText = "This is the quarterly budget report with financial details"
	emails[1].BodyText = "Weekly standup meeting notes from engineering team"
	emails[2].BodyText = "The database migration plan for the quarterly release"

	for _, email := range emails {
		if err := s.CreateEmail(ctx, email); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("search by subject keyword", func(t *testing.T) {
		result, err := s.SearchEmails(ctx, "budget", 1, 25)
		if err != nil {
			t.Fatalf("SearchEmails failed: %v", err)
		}
		if result.Total < 1 {
			t.Errorf("expected at least 1 result for 'budget', got %d", result.Total)
		}
	})

	t.Run("search by body keyword", func(t *testing.T) {
		result, err := s.SearchEmails(ctx, "migration", 1, 25)
		if err != nil {
			t.Fatalf("SearchEmails failed: %v", err)
		}
		if result.Total < 1 {
			t.Errorf("expected at least 1 result for 'migration', got %d", result.Total)
		}
	})

	t.Run("search returns no results for unmatched term", func(t *testing.T) {
		result, err := s.SearchEmails(ctx, "xyznonexistent", 1, 25)
		if err != nil {
			t.Fatalf("SearchEmails failed: %v", err)
		}
		if result.Total != 0 {
			t.Errorf("expected 0 results, got %d", result.Total)
		}
	})

	t.Run("search with term matching multiple emails", func(t *testing.T) {
		// "quarterly" appears in emails[0] subject and emails[2] body
		result, err := s.SearchEmails(ctx, "quarterly", 1, 25)
		if err != nil {
			t.Fatalf("SearchEmails failed: %v", err)
		}
		if result.Total < 2 {
			t.Errorf("expected at least 2 results for 'quarterly', got %d", result.Total)
		}
	})

	t.Run("search respects pagination", func(t *testing.T) {
		result, err := s.SearchEmails(ctx, "quarterly", 1, 1)
		if err != nil {
			t.Fatalf("SearchEmails failed: %v", err)
		}
		if len(result.Emails) != 1 {
			t.Errorf("expected 1 email on page, got %d", len(result.Emails))
		}
		if result.TotalPages < 2 {
			t.Errorf("expected at least 2 total pages, got %d", result.TotalPages)
		}
	})
}

// ---------------------------------------------------------------------------
// Thread operations
// ---------------------------------------------------------------------------

func TestGetThread(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()
	seedSystemLabels(t, s)

	threadID := uuid.New().String()

	// Create 3 emails in the same thread
	for i := range 3 {
		email := makeTestEmail(fmt.Sprintf("Thread Email %d", i), "inbox")
		email.ThreadID = threadID
		email.ReceivedAt = time.Now().Add(-time.Duration(3-i) * time.Hour) // oldest first
		if i > 0 {
			email.IsStarred = true
		}
		if i == 0 {
			email.IsImportant = true
		}
		if err := s.CreateEmail(ctx, email); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("returns thread with all emails", func(t *testing.T) {
		thread, err := s.GetThread(ctx, threadID)
		if err != nil {
			t.Fatalf("GetThread failed: %v", err)
		}

		if thread.ID != threadID {
			t.Errorf("ID: want %s, got %s", threadID, thread.ID)
		}
		if thread.EmailCount != 3 {
			t.Errorf("EmailCount: want 3, got %d", thread.EmailCount)
		}
		if len(thread.Emails) != 3 {
			t.Fatalf("Emails: want 3, got %d", len(thread.Emails))
		}
	})

	t.Run("thread subject comes from first email", func(t *testing.T) {
		thread, err := s.GetThread(ctx, threadID)
		if err != nil {
			t.Fatal(err)
		}
		if thread.Subject != "Thread Email 0" {
			t.Errorf("Subject: want 'Thread Email 0', got %s", thread.Subject)
		}
	})

	t.Run("thread snippet comes from last email", func(t *testing.T) {
		thread, err := s.GetThread(ctx, threadID)
		if err != nil {
			t.Fatal(err)
		}
		expected := "Test body text for Thread Email 2"
		if thread.Snippet != expected {
			t.Errorf("Snippet: want %q, got %q", expected, thread.Snippet)
		}
	})

	t.Run("thread aggregates starred flag", func(t *testing.T) {
		thread, err := s.GetThread(ctx, threadID)
		if err != nil {
			t.Fatal(err)
		}
		if !thread.IsStarred {
			t.Error("expected thread IsStarred to be true (at least one email is starred)")
		}
	})

	t.Run("thread aggregates important flag", func(t *testing.T) {
		thread, err := s.GetThread(ctx, threadID)
		if err != nil {
			t.Fatal(err)
		}
		if !thread.IsImportant {
			t.Error("expected thread IsImportant to be true (at least one email is important)")
		}
	})

	t.Run("thread counts unread emails", func(t *testing.T) {
		thread, err := s.GetThread(ctx, threadID)
		if err != nil {
			t.Fatal(err)
		}
		// All emails were created with IsRead=false by default
		if thread.UnreadCount != 3 {
			t.Errorf("UnreadCount: want 3, got %d", thread.UnreadCount)
		}
	})

	t.Run("thread emails are ordered by received_at ASC", func(t *testing.T) {
		thread, err := s.GetThread(ctx, threadID)
		if err != nil {
			t.Fatal(err)
		}
		for i := 1; i < len(thread.Emails); i++ {
			if thread.Emails[i].ReceivedAt.Before(thread.Emails[i-1].ReceivedAt) {
				t.Error("thread emails are not ordered by received_at ASC")
				break
			}
		}
	})

	t.Run("returns error for non-existent thread", func(t *testing.T) {
		_, err := s.GetThread(ctx, "non-existent-thread")
		if err == nil {
			t.Error("expected error for non-existent thread")
		}
	})
}

func TestListThreads(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()
	seedSystemLabels(t, s)

	// Create 3 threads with different numbers of emails
	threadIDs := []string{uuid.New().String(), uuid.New().String(), uuid.New().String()}

	for i, threadID := range threadIDs {
		for j := 0; j <= i; j++ {
			email := makeTestEmail(fmt.Sprintf("Thread %d Email %d", i, j), "inbox")
			email.ThreadID = threadID
			email.ReceivedAt = time.Now().Add(-time.Duration(len(threadIDs)-i)*time.Hour + time.Duration(j)*time.Minute)
			if err := s.CreateEmail(ctx, email); err != nil {
				t.Fatal(err)
			}
		}
	}

	t.Run("returns all threads", func(t *testing.T) {
		result, err := s.ListThreads(ctx, store.EmailFilter{})
		if err != nil {
			t.Fatalf("ListThreads failed: %v", err)
		}
		if result.Total != 3 {
			t.Errorf("Total: want 3, got %d", result.Total)
		}
		if len(result.Threads) != 3 {
			t.Errorf("Threads count: want 3, got %d", len(result.Threads))
		}
	})

	t.Run("threads are ordered by most recent email", func(t *testing.T) {
		result, err := s.ListThreads(ctx, store.EmailFilter{})
		if err != nil {
			t.Fatal(err)
		}
		for i := 1; i < len(result.Threads); i++ {
			if result.Threads[i].LastEmailAt.After(result.Threads[i-1].LastEmailAt) {
				t.Error("threads are not ordered by most recent email DESC")
				break
			}
		}
	})

	t.Run("respects pagination", func(t *testing.T) {
		result, err := s.ListThreads(ctx, store.EmailFilter{Page: 1, PerPage: 2})
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Threads) != 2 {
			t.Errorf("Threads count: want 2, got %d", len(result.Threads))
		}
		if result.Total != 3 {
			t.Errorf("Total: want 3, got %d", result.Total)
		}
		if result.TotalPages != 2 {
			t.Errorf("TotalPages: want 2, got %d", result.TotalPages)
		}
	})

	t.Run("each thread has correct email count", func(t *testing.T) {
		result, err := s.ListThreads(ctx, store.EmailFilter{})
		if err != nil {
			t.Fatal(err)
		}
		for _, thread := range result.Threads {
			if thread.EmailCount < 1 {
				t.Errorf("thread %s has EmailCount %d, expected >= 1", thread.ID, thread.EmailCount)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// Label CRUD
// ---------------------------------------------------------------------------

func TestCreateLabel(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	t.Run("creates label with all fields", func(t *testing.T) {
		label := &types.Label{
			ID:       "test-label",
			Name:     "Test Label",
			Color:    "#FF0000",
			Type:     types.LabelTypeUser,
			Visible:  true,
			Position: 10,
		}

		err := s.CreateLabel(ctx, label)
		if err != nil {
			t.Fatalf("CreateLabel failed: %v", err)
		}
	})

	t.Run("generates ID if empty", func(t *testing.T) {
		label := &types.Label{
			Name:    "Auto ID Label",
			Type:    types.LabelTypeUser,
			Visible: true,
		}

		err := s.CreateLabel(ctx, label)
		if err != nil {
			t.Fatalf("CreateLabel failed: %v", err)
		}
		if label.ID == "" {
			t.Error("expected non-empty ID after creation")
		}
	})

	t.Run("label appears in list", func(t *testing.T) {
		label := &types.Label{
			ID:      "visible-label",
			Name:    "Visible Label",
			Color:   "#00FF00",
			Type:    types.LabelTypeUser,
			Visible: true,
		}
		if err := s.CreateLabel(ctx, label); err != nil {
			t.Fatal(err)
		}

		labels, err := s.ListLabels(ctx)
		if err != nil {
			t.Fatal(err)
		}

		found := false
		for _, l := range labels {
			if l.ID == "visible-label" {
				found = true
				if l.Name != "Visible Label" {
					t.Errorf("Name: want 'Visible Label', got %s", l.Name)
				}
				if l.Color != "#00FF00" {
					t.Errorf("Color: want '#00FF00', got %s", l.Color)
				}
				break
			}
		}
		if !found {
			t.Error("created label not found in list")
		}
	})
}

func TestListLabels(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()
	seedSystemLabels(t, s)

	t.Run("returns system labels", func(t *testing.T) {
		labels, err := s.ListLabels(ctx)
		if err != nil {
			t.Fatalf("ListLabels failed: %v", err)
		}
		if len(labels) < 8 {
			t.Errorf("expected at least 8 labels (system), got %d", len(labels))
		}
	})

	t.Run("labels are ordered by position", func(t *testing.T) {
		labels, err := s.ListLabels(ctx)
		if err != nil {
			t.Fatal(err)
		}
		for i := 1; i < len(labels); i++ {
			if labels[i].Position < labels[i-1].Position {
				t.Error("labels are not ordered by position")
				break
			}
		}
	})

	t.Run("includes unread and total counts", func(t *testing.T) {
		// Create an unread email in inbox
		email := makeTestEmail("Count Test", "inbox")
		if err := s.CreateEmail(ctx, email); err != nil {
			t.Fatal(err)
		}

		labels, err := s.ListLabels(ctx)
		if err != nil {
			t.Fatal(err)
		}

		for _, l := range labels {
			if l.ID == "inbox" {
				if l.TotalCount < 1 {
					t.Errorf("inbox TotalCount: want >= 1, got %d", l.TotalCount)
				}
				if l.UnreadCount < 1 {
					t.Errorf("inbox UnreadCount: want >= 1, got %d", l.UnreadCount)
				}
				break
			}
		}
	})
}

func TestUpdateLabel(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	t.Run("update name", func(t *testing.T) {
		label := &types.Label{ID: "update-name", Name: "Original", Type: types.LabelTypeUser, Visible: true}
		if err := s.CreateLabel(ctx, label); err != nil {
			t.Fatal(err)
		}

		err := s.UpdateLabel(ctx, "update-name", map[string]any{"name": "Updated Name"})
		if err != nil {
			t.Fatalf("UpdateLabel failed: %v", err)
		}

		labels, err := s.ListLabels(ctx)
		if err != nil {
			t.Fatal(err)
		}
		for _, l := range labels {
			if l.ID == "update-name" {
				if l.Name != "Updated Name" {
					t.Errorf("Name: want 'Updated Name', got %s", l.Name)
				}
				return
			}
		}
		t.Error("updated label not found")
	})

	t.Run("update color", func(t *testing.T) {
		label := &types.Label{ID: "update-color", Name: "Color Test", Color: "#000000", Type: types.LabelTypeUser, Visible: true}
		if err := s.CreateLabel(ctx, label); err != nil {
			t.Fatal(err)
		}

		err := s.UpdateLabel(ctx, "update-color", map[string]any{"color": "#FF5500"})
		if err != nil {
			t.Fatalf("UpdateLabel failed: %v", err)
		}

		labels, err := s.ListLabels(ctx)
		if err != nil {
			t.Fatal(err)
		}
		for _, l := range labels {
			if l.ID == "update-color" {
				if l.Color != "#FF5500" {
					t.Errorf("Color: want '#FF5500', got %s", l.Color)
				}
				return
			}
		}
		t.Error("updated label not found")
	})

	t.Run("no-op when no valid fields", func(t *testing.T) {
		label := &types.Label{ID: "noop-update", Name: "NoOp", Type: types.LabelTypeUser, Visible: true}
		if err := s.CreateLabel(ctx, label); err != nil {
			t.Fatal(err)
		}

		err := s.UpdateLabel(ctx, "noop-update", map[string]any{"invalid_field": "value"})
		if err != nil {
			t.Fatalf("UpdateLabel with no valid fields should not error: %v", err)
		}
	})
}

func TestDeleteLabel(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()
	seedSystemLabels(t, s)

	t.Run("deletes user label", func(t *testing.T) {
		label := &types.Label{ID: "delete-me", Name: "Delete Me", Type: types.LabelTypeUser, Visible: true}
		if err := s.CreateLabel(ctx, label); err != nil {
			t.Fatal(err)
		}

		err := s.DeleteLabel(ctx, "delete-me")
		if err != nil {
			t.Fatalf("DeleteLabel failed: %v", err)
		}

		labels, err := s.ListLabels(ctx)
		if err != nil {
			t.Fatal(err)
		}
		for _, l := range labels {
			if l.ID == "delete-me" {
				t.Error("label should have been deleted")
			}
		}
	})

	t.Run("cannot delete system label", func(t *testing.T) {
		err := s.DeleteLabel(ctx, "inbox")
		if err == nil {
			t.Error("expected error when deleting system label")
		}
	})

	t.Run("returns error for non-existent label", func(t *testing.T) {
		err := s.DeleteLabel(ctx, "non-existent")
		if err == nil {
			t.Error("expected error for non-existent label")
		}
	})

	t.Run("removes label associations from emails", func(t *testing.T) {
		label := &types.Label{ID: "remove-assoc", Name: "Remove Assoc", Type: types.LabelTypeUser, Visible: true}
		if err := s.CreateLabel(ctx, label); err != nil {
			t.Fatal(err)
		}

		email := makeTestEmail("Label Assoc Test", "remove-assoc")
		if err := s.CreateEmail(ctx, email); err != nil {
			t.Fatal(err)
		}

		// Verify label is associated
		got, err := s.GetEmail(ctx, email.ID)
		if err != nil {
			t.Fatal(err)
		}
		hasLabel := false
		for _, l := range got.Labels {
			if l == "remove-assoc" {
				hasLabel = true
			}
		}
		if !hasLabel {
			t.Fatal("label should be associated before deletion")
		}

		// Delete the label
		if err := s.DeleteLabel(ctx, "remove-assoc"); err != nil {
			t.Fatal(err)
		}

		// Verify label is no longer associated
		got, err = s.GetEmail(ctx, email.ID)
		if err != nil {
			t.Fatal(err)
		}
		for _, l := range got.Labels {
			if l == "remove-assoc" {
				t.Error("label should have been removed from email after deletion")
			}
		}
	})
}

func TestAddRemoveEmailLabel(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()
	seedSystemLabels(t, s)

	workLabel := &types.Label{ID: "work", Name: "Work", Type: types.LabelTypeUser, Visible: true}
	if err := s.CreateLabel(ctx, workLabel); err != nil {
		t.Fatal(err)
	}

	t.Run("add label to email", func(t *testing.T) {
		email := makeTestEmail("Add Label Test", "inbox")
		if err := s.CreateEmail(ctx, email); err != nil {
			t.Fatal(err)
		}

		err := s.AddEmailLabel(ctx, email.ID, "work")
		if err != nil {
			t.Fatalf("AddEmailLabel failed: %v", err)
		}

		got, err := s.GetEmail(ctx, email.ID)
		if err != nil {
			t.Fatal(err)
		}

		hasWork := false
		for _, l := range got.Labels {
			if l == "work" {
				hasWork = true
			}
		}
		if !hasWork {
			t.Error("expected 'work' label on email")
		}
	})

	t.Run("add label is idempotent", func(t *testing.T) {
		email := makeTestEmail("Idempotent Label", "inbox")
		if err := s.CreateEmail(ctx, email); err != nil {
			t.Fatal(err)
		}

		// Add same label twice - should not error
		if err := s.AddEmailLabel(ctx, email.ID, "work"); err != nil {
			t.Fatal(err)
		}
		if err := s.AddEmailLabel(ctx, email.ID, "work"); err != nil {
			t.Fatal(err)
		}

		got, err := s.GetEmail(ctx, email.ID)
		if err != nil {
			t.Fatal(err)
		}

		workCount := 0
		for _, l := range got.Labels {
			if l == "work" {
				workCount++
			}
		}
		if workCount != 1 {
			t.Errorf("expected exactly 1 'work' label, got %d", workCount)
		}
	})

	t.Run("remove label from email", func(t *testing.T) {
		email := makeTestEmail("Remove Label Test", "inbox")
		if err := s.CreateEmail(ctx, email); err != nil {
			t.Fatal(err)
		}

		if err := s.AddEmailLabel(ctx, email.ID, "work"); err != nil {
			t.Fatal(err)
		}

		err := s.RemoveEmailLabel(ctx, email.ID, "work")
		if err != nil {
			t.Fatalf("RemoveEmailLabel failed: %v", err)
		}

		got, err := s.GetEmail(ctx, email.ID)
		if err != nil {
			t.Fatal(err)
		}

		for _, l := range got.Labels {
			if l == "work" {
				t.Error("expected 'work' label to be removed")
			}
		}
	})

	t.Run("remove non-existent label does not error", func(t *testing.T) {
		email := makeTestEmail("Remove Nonexistent", "inbox")
		if err := s.CreateEmail(ctx, email); err != nil {
			t.Fatal(err)
		}

		err := s.RemoveEmailLabel(ctx, email.ID, "nonexistent-label")
		if err != nil {
			t.Fatalf("RemoveEmailLabel should not error for non-existent label: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// Contact CRUD
// ---------------------------------------------------------------------------

func TestCreateContact(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	t.Run("creates contact", func(t *testing.T) {
		contact := &types.Contact{
			ID:    uuid.New().String(),
			Email: "alice@example.com",
			Name:  "Alice",
		}

		err := s.CreateContact(ctx, contact)
		if err != nil {
			t.Fatalf("CreateContact failed: %v", err)
		}
	})

	t.Run("generates ID if empty", func(t *testing.T) {
		contact := &types.Contact{
			Email: "autoid@example.com",
			Name:  "Auto ID",
		}

		err := s.CreateContact(ctx, contact)
		if err != nil {
			t.Fatalf("CreateContact failed: %v", err)
		}
		if contact.ID == "" {
			t.Error("expected non-empty ID after creation")
		}
	})

	t.Run("contact appears in list", func(t *testing.T) {
		contact := &types.Contact{
			ID:    uuid.New().String(),
			Email: "visible@example.com",
			Name:  "Visible Contact",
		}
		if err := s.CreateContact(ctx, contact); err != nil {
			t.Fatal(err)
		}

		contacts, err := s.ListContacts(ctx, "")
		if err != nil {
			t.Fatal(err)
		}

		found := false
		for _, c := range contacts {
			if c.Email == "visible@example.com" {
				found = true
				if c.Name != "Visible Contact" {
					t.Errorf("Name: want 'Visible Contact', got %s", c.Name)
				}
				break
			}
		}
		if !found {
			t.Error("created contact not found in list")
		}
	})

	t.Run("upserts on duplicate email", func(t *testing.T) {
		contact1 := &types.Contact{
			ID:           uuid.New().String(),
			Email:        "upsert@example.com",
			Name:         "First Name",
			ContactCount: 1,
		}
		if err := s.CreateContact(ctx, contact1); err != nil {
			t.Fatal(err)
		}

		contact2 := &types.Contact{
			ID:           uuid.New().String(),
			Email:        "upsert@example.com",
			Name:         "Updated Name",
			ContactCount: 1,
		}
		if err := s.CreateContact(ctx, contact2); err != nil {
			t.Fatal(err)
		}

		// Should update name and increment contact_count
		contacts, err := s.ListContacts(ctx, "")
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range contacts {
			if c.Email == "upsert@example.com" {
				if c.Name != "Updated Name" {
					t.Errorf("Name: want 'Updated Name', got %s", c.Name)
				}
				if c.ContactCount < 2 {
					t.Errorf("ContactCount: want >= 2, got %d", c.ContactCount)
				}
				return
			}
		}
		t.Error("upserted contact not found")
	})
}

func TestListContacts(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	// Create several contacts
	contacts := []*types.Contact{
		{ID: uuid.New().String(), Email: "alice@test.com", Name: "Alice Johnson", ContactCount: 5},
		{ID: uuid.New().String(), Email: "bob@test.com", Name: "Bob Smith", ContactCount: 10},
		{ID: uuid.New().String(), Email: "carol@test.com", Name: "Carol Williams", ContactCount: 3},
	}
	for _, c := range contacts {
		if err := s.CreateContact(ctx, c); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("lists all contacts", func(t *testing.T) {
		result, err := s.ListContacts(ctx, "")
		if err != nil {
			t.Fatalf("ListContacts failed: %v", err)
		}
		if len(result) < 3 {
			t.Errorf("expected at least 3 contacts, got %d", len(result))
		}
	})

	t.Run("contacts ordered by contact_count DESC", func(t *testing.T) {
		result, err := s.ListContacts(ctx, "")
		if err != nil {
			t.Fatal(err)
		}
		for i := 1; i < len(result); i++ {
			if result[i].ContactCount > result[i-1].ContactCount {
				t.Error("contacts are not ordered by contact_count DESC")
				break
			}
		}
	})

	t.Run("search by name using LIKE fallback", func(t *testing.T) {
		// Use a query that finds alice. FTS may or may not work; either way at least 1 result.
		result, err := s.ListContacts(ctx, "Alice")
		if err != nil {
			t.Fatalf("ListContacts (search) failed: %v", err)
		}
		if len(result) < 1 {
			t.Errorf("expected at least 1 contact for 'Alice', got %d", len(result))
		}
	})

	t.Run("search returns empty for no match", func(t *testing.T) {
		result, err := s.ListContacts(ctx, "zzzznoone")
		if err != nil {
			t.Fatalf("ListContacts (no match) failed: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("expected 0 contacts, got %d", len(result))
		}
	})
}

func TestUpdateContact(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	t.Run("update name", func(t *testing.T) {
		contact := &types.Contact{
			ID:    "update-contact",
			Email: "update-name@test.com",
			Name:  "Old Name",
		}
		if err := s.CreateContact(ctx, contact); err != nil {
			t.Fatal(err)
		}

		err := s.UpdateContact(ctx, "update-contact", map[string]any{"name": "New Name"})
		if err != nil {
			t.Fatalf("UpdateContact failed: %v", err)
		}

		contacts, err := s.ListContacts(ctx, "")
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range contacts {
			if c.ID == "update-contact" {
				if c.Name != "New Name" {
					t.Errorf("Name: want 'New Name', got %s", c.Name)
				}
				return
			}
		}
		t.Error("updated contact not found")
	})

	t.Run("no-op with empty updates", func(t *testing.T) {
		contact := &types.Contact{
			ID:    "noop-contact",
			Email: "noop@test.com",
			Name:  "NoOp",
		}
		if err := s.CreateContact(ctx, contact); err != nil {
			t.Fatal(err)
		}

		err := s.UpdateContact(ctx, "noop-contact", map[string]any{"invalid": "value"})
		if err != nil {
			t.Fatalf("UpdateContact with no valid fields should not error: %v", err)
		}
	})
}

func TestDeleteContact(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	t.Run("deletes contact", func(t *testing.T) {
		contact := &types.Contact{
			ID:    "delete-contact",
			Email: "delete@test.com",
			Name:  "Delete Me",
		}
		if err := s.CreateContact(ctx, contact); err != nil {
			t.Fatal(err)
		}

		err := s.DeleteContact(ctx, "delete-contact")
		if err != nil {
			t.Fatalf("DeleteContact failed: %v", err)
		}

		contacts, err := s.ListContacts(ctx, "")
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range contacts {
			if c.ID == "delete-contact" {
				t.Error("contact should have been deleted")
			}
		}
	})

	t.Run("deleting non-existent contact does not error", func(t *testing.T) {
		err := s.DeleteContact(ctx, "non-existent-contact")
		if err != nil {
			t.Fatalf("DeleteContact should not error for non-existent: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// Settings
// ---------------------------------------------------------------------------

func TestGetSettings(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	t.Run("returns default settings", func(t *testing.T) {
		settings, err := s.GetSettings(ctx)
		if err != nil {
			t.Fatalf("GetSettings failed: %v", err)
		}

		if settings.ID != 1 {
			t.Errorf("ID: want 1, got %d", settings.ID)
		}
		if settings.DisplayName != "Me" {
			t.Errorf("DisplayName: want 'Me', got %s", settings.DisplayName)
		}
		if settings.EmailAddress != "me@example.com" {
			t.Errorf("EmailAddress: want 'me@example.com', got %s", settings.EmailAddress)
		}
		if settings.Theme != "light" {
			t.Errorf("Theme: want 'light', got %s", settings.Theme)
		}
		if settings.Density != "default" {
			t.Errorf("Density: want 'default', got %s", settings.Density)
		}
		if !settings.ConversationView {
			t.Error("ConversationView: want true, got false")
		}
		if settings.AutoAdvance != "newer" {
			t.Errorf("AutoAdvance: want 'newer', got %s", settings.AutoAdvance)
		}
		if settings.UndoSendSeconds != 5 {
			t.Errorf("UndoSendSeconds: want 5, got %d", settings.UndoSendSeconds)
		}
	})
}

func TestUpdateSettings(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	t.Run("updates and persists settings", func(t *testing.T) {
		newSettings := &types.Settings{
			DisplayName:      "John Doe",
			EmailAddress:     "john@example.com",
			Signature:        "Best regards,\nJohn",
			Theme:            "dark",
			Density:          "compact",
			ConversationView: false,
			AutoAdvance:      "older",
			UndoSendSeconds:  10,
		}

		err := s.UpdateSettings(ctx, newSettings)
		if err != nil {
			t.Fatalf("UpdateSettings failed: %v", err)
		}

		got, err := s.GetSettings(ctx)
		if err != nil {
			t.Fatalf("GetSettings failed: %v", err)
		}

		if got.DisplayName != "John Doe" {
			t.Errorf("DisplayName: want 'John Doe', got %s", got.DisplayName)
		}
		if got.EmailAddress != "john@example.com" {
			t.Errorf("EmailAddress: want 'john@example.com', got %s", got.EmailAddress)
		}
		if got.Signature != "Best regards,\nJohn" {
			t.Errorf("Signature: want 'Best regards,\\nJohn', got %s", got.Signature)
		}
		if got.Theme != "dark" {
			t.Errorf("Theme: want 'dark', got %s", got.Theme)
		}
		if got.Density != "compact" {
			t.Errorf("Density: want 'compact', got %s", got.Density)
		}
		if got.ConversationView {
			t.Error("ConversationView: want false, got true")
		}
		if got.AutoAdvance != "older" {
			t.Errorf("AutoAdvance: want 'older', got %s", got.AutoAdvance)
		}
		if got.UndoSendSeconds != 10 {
			t.Errorf("UndoSendSeconds: want 10, got %d", got.UndoSendSeconds)
		}
	})

	t.Run("update is idempotent", func(t *testing.T) {
		settings := &types.Settings{
			DisplayName:      "Test User",
			EmailAddress:     "test@example.com",
			Theme:            "light",
			Density:          "default",
			ConversationView: true,
			AutoAdvance:      "newer",
			UndoSendSeconds:  5,
		}

		if err := s.UpdateSettings(ctx, settings); err != nil {
			t.Fatal(err)
		}
		if err := s.UpdateSettings(ctx, settings); err != nil {
			t.Fatal(err)
		}

		got, err := s.GetSettings(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if got.DisplayName != "Test User" {
			t.Errorf("DisplayName: want 'Test User', got %s", got.DisplayName)
		}
	})
}

// ---------------------------------------------------------------------------
// Attachments
// ---------------------------------------------------------------------------

func TestCreateAttachment(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	// Create an email first
	email := makeTestEmail("Attachment Email")
	if err := s.CreateEmail(ctx, email); err != nil {
		t.Fatal(err)
	}

	t.Run("creates attachment with data", func(t *testing.T) {
		att := &types.Attachment{
			ID:          uuid.New().String(),
			EmailID:     email.ID,
			Filename:    "document.pdf",
			ContentType: "application/pdf",
			SizeBytes:   1024,
		}
		data := []byte("fake pdf content here")

		err := s.CreateAttachment(ctx, att, data)
		if err != nil {
			t.Fatalf("CreateAttachment failed: %v", err)
		}
	})

	t.Run("sets has_attachments on email", func(t *testing.T) {
		got, err := s.GetEmail(ctx, email.ID)
		if err != nil {
			t.Fatal(err)
		}
		if !got.HasAttachments {
			t.Error("expected HasAttachments to be true after creating attachment")
		}
	})

	t.Run("generates ID if empty", func(t *testing.T) {
		att := &types.Attachment{
			EmailID:     email.ID,
			Filename:    "auto-id.txt",
			ContentType: "text/plain",
			SizeBytes:   100,
		}

		err := s.CreateAttachment(ctx, att, []byte("test data"))
		if err != nil {
			t.Fatalf("CreateAttachment failed: %v", err)
		}
		if att.ID == "" {
			t.Error("expected non-empty ID after creation")
		}
	})
}

func TestListAttachments(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	email := makeTestEmail("List Attachments Email")
	if err := s.CreateEmail(ctx, email); err != nil {
		t.Fatal(err)
	}

	// Create 3 attachments
	for i := range 3 {
		att := &types.Attachment{
			ID:          uuid.New().String(),
			EmailID:     email.ID,
			Filename:    fmt.Sprintf("file%d.txt", i),
			ContentType: "text/plain",
			SizeBytes:   int64(100 * (i + 1)),
		}
		if err := s.CreateAttachment(ctx, att, fmt.Appendf(nil, "content %d", i)); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("lists all attachments for email", func(t *testing.T) {
		attachments, err := s.ListAttachments(ctx, email.ID)
		if err != nil {
			t.Fatalf("ListAttachments failed: %v", err)
		}
		if len(attachments) != 3 {
			t.Errorf("expected 3 attachments, got %d", len(attachments))
		}
	})

	t.Run("returns empty for email with no attachments", func(t *testing.T) {
		otherEmail := makeTestEmail("No Attachments")
		if err := s.CreateEmail(ctx, otherEmail); err != nil {
			t.Fatal(err)
		}

		attachments, err := s.ListAttachments(ctx, otherEmail.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(attachments) != 0 {
			t.Errorf("expected 0 attachments, got %d", len(attachments))
		}
	})

	t.Run("attachments contain correct metadata", func(t *testing.T) {
		attachments, err := s.ListAttachments(ctx, email.ID)
		if err != nil {
			t.Fatal(err)
		}
		for _, att := range attachments {
			if att.EmailID != email.ID {
				t.Errorf("EmailID: want %s, got %s", email.ID, att.EmailID)
			}
			if att.Filename == "" {
				t.Error("expected non-empty Filename")
			}
			if att.ContentType == "" {
				t.Error("expected non-empty ContentType")
			}
		}
	})
}

func TestGetAttachment(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	email := makeTestEmail("Get Attachment Email")
	if err := s.CreateEmail(ctx, email); err != nil {
		t.Fatal(err)
	}

	originalData := []byte("this is the attachment content with some binary data: \x00\x01\x02")
	att := &types.Attachment{
		ID:          uuid.New().String(),
		EmailID:     email.ID,
		Filename:    "test-file.bin",
		ContentType: "application/octet-stream",
		SizeBytes:   int64(len(originalData)),
	}
	if err := s.CreateAttachment(ctx, att, originalData); err != nil {
		t.Fatal(err)
	}

	t.Run("returns attachment with data", func(t *testing.T) {
		gotAtt, gotData, err := s.GetAttachment(ctx, att.ID)
		if err != nil {
			t.Fatalf("GetAttachment failed: %v", err)
		}

		if gotAtt.ID != att.ID {
			t.Errorf("ID: want %s, got %s", att.ID, gotAtt.ID)
		}
		if gotAtt.EmailID != att.EmailID {
			t.Errorf("EmailID: want %s, got %s", att.EmailID, gotAtt.EmailID)
		}
		if gotAtt.Filename != "test-file.bin" {
			t.Errorf("Filename: want 'test-file.bin', got %s", gotAtt.Filename)
		}
		if gotAtt.ContentType != "application/octet-stream" {
			t.Errorf("ContentType: want 'application/octet-stream', got %s", gotAtt.ContentType)
		}
		if string(gotData) != string(originalData) {
			t.Error("attachment data does not match original")
		}
	})

	t.Run("returns error for non-existent attachment", func(t *testing.T) {
		_, _, err := s.GetAttachment(ctx, "non-existent-att")
		if err == nil {
			t.Error("expected error for non-existent attachment")
		}
	})
}

func TestDeleteAttachment(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	email := makeTestEmail("Delete Attachment Email")
	if err := s.CreateEmail(ctx, email); err != nil {
		t.Fatal(err)
	}

	t.Run("deletes attachment and updates has_attachments flag", func(t *testing.T) {
		att := &types.Attachment{
			ID:          uuid.New().String(),
			EmailID:     email.ID,
			Filename:    "delete-me.txt",
			ContentType: "text/plain",
			SizeBytes:   50,
		}
		if err := s.CreateAttachment(ctx, att, []byte("delete me")); err != nil {
			t.Fatal(err)
		}

		// Confirm has_attachments is true
		got, _ := s.GetEmail(ctx, email.ID)
		if !got.HasAttachments {
			t.Fatal("expected HasAttachments to be true before delete")
		}

		err := s.DeleteAttachment(ctx, att.ID)
		if err != nil {
			t.Fatalf("DeleteAttachment failed: %v", err)
		}

		// Attachment should no longer be retrievable
		_, _, err = s.GetAttachment(ctx, att.ID)
		if err == nil {
			t.Error("expected error getting deleted attachment")
		}

		// has_attachments should be false since no more attachments
		got, _ = s.GetEmail(ctx, email.ID)
		if got.HasAttachments {
			t.Error("expected HasAttachments to be false after deleting last attachment")
		}
	})

	t.Run("does not clear has_attachments if other attachments remain", func(t *testing.T) {
		email2 := makeTestEmail("Multi Attachment Email")
		if err := s.CreateEmail(ctx, email2); err != nil {
			t.Fatal(err)
		}

		att1 := &types.Attachment{
			ID: uuid.New().String(), EmailID: email2.ID,
			Filename: "keep.txt", ContentType: "text/plain", SizeBytes: 10,
		}
		att2 := &types.Attachment{
			ID: uuid.New().String(), EmailID: email2.ID,
			Filename: "remove.txt", ContentType: "text/plain", SizeBytes: 10,
		}
		if err := s.CreateAttachment(ctx, att1, []byte("keep")); err != nil {
			t.Fatal(err)
		}
		if err := s.CreateAttachment(ctx, att2, []byte("remove")); err != nil {
			t.Fatal(err)
		}

		// Delete one attachment
		if err := s.DeleteAttachment(ctx, att2.ID); err != nil {
			t.Fatal(err)
		}

		// has_attachments should still be true
		got, _ := s.GetEmail(ctx, email2.ID)
		if !got.HasAttachments {
			t.Error("expected HasAttachments to remain true when other attachments exist")
		}
	})
}

// ---------------------------------------------------------------------------
// Seed operations
// ---------------------------------------------------------------------------

func TestSeedLabels(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	t.Run("seeds all system and custom labels", func(t *testing.T) {
		err := s.SeedLabels(ctx)
		if err != nil {
			t.Fatalf("SeedLabels failed: %v", err)
		}

		labels, err := s.ListLabels(ctx)
		if err != nil {
			t.Fatal(err)
		}

		// 10 system + 4 custom = 14 labels total
		if len(labels) < 14 {
			t.Errorf("expected at least 14 labels, got %d", len(labels))
		}

		// Verify some specific system labels exist
		expectedSystem := map[string]bool{
			"inbox": false, "sent": false, "trash": false,
			"drafts": false, "spam": false, "starred": false,
		}
		for _, l := range labels {
			if _, ok := expectedSystem[l.ID]; ok {
				expectedSystem[l.ID] = true
				if l.Type != types.LabelTypeSystem {
					t.Errorf("label %s: want type system, got %s", l.ID, l.Type)
				}
			}
		}
		for id, found := range expectedSystem {
			if !found {
				t.Errorf("expected system label %s not found", id)
			}
		}

		// Verify some custom labels
		expectedCustom := map[string]bool{"work": false, "personal": false, "finance": false, "travel": false}
		for _, l := range labels {
			if _, ok := expectedCustom[l.ID]; ok {
				expectedCustom[l.ID] = true
				if l.Type != types.LabelTypeUser {
					t.Errorf("label %s: want type user, got %s", l.ID, l.Type)
				}
			}
		}
		for id, found := range expectedCustom {
			if !found {
				t.Errorf("expected custom label %s not found", id)
			}
		}
	})

	t.Run("is idempotent", func(t *testing.T) {
		err := s.SeedLabels(ctx)
		if err != nil {
			t.Fatalf("second SeedLabels failed: %v", err)
		}

		labels, err := s.ListLabels(ctx)
		if err != nil {
			t.Fatal(err)
		}
		// Should still be 14 (not duplicated)
		if len(labels) > 14 {
			t.Errorf("expected 14 labels after second seed, got %d", len(labels))
		}
	})
}

func TestSeedContacts(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	t.Run("seeds contacts", func(t *testing.T) {
		err := s.SeedContacts(ctx)
		if err != nil {
			t.Fatalf("SeedContacts failed: %v", err)
		}

		contacts, err := s.ListContacts(ctx, "")
		if err != nil {
			t.Fatal(err)
		}

		// There are 15 contacts in the seed data
		if len(contacts) < 15 {
			t.Errorf("expected at least 15 contacts, got %d", len(contacts))
		}
	})

	t.Run("seeded contacts have correct fields", func(t *testing.T) {
		contacts, err := s.ListContacts(ctx, "")
		if err != nil {
			t.Fatal(err)
		}

		for _, c := range contacts {
			if c.ID == "" {
				t.Error("contact has empty ID")
			}
			if c.Email == "" {
				t.Error("contact has empty email")
			}
			if c.Name == "" {
				t.Error("contact has empty name")
			}
		}
	})
}

func TestSeedEmails(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	// SeedEmails requires labels to exist for email_labels associations
	if err := s.SeedLabels(ctx); err != nil {
		t.Fatalf("SeedLabels prerequisite failed: %v", err)
	}

	t.Run("seeds emails", func(t *testing.T) {
		err := s.SeedEmails(ctx)
		if err != nil {
			t.Fatalf("SeedEmails failed: %v", err)
		}

		result, err := s.ListEmails(ctx, store.EmailFilter{PerPage: 100})
		if err != nil {
			t.Fatal(err)
		}

		// There are 32 emails in the seed data
		if result.Total < 32 {
			t.Errorf("expected at least 32 emails, got %d", result.Total)
		}
	})

	t.Run("seeded emails have labels", func(t *testing.T) {
		result, err := s.ListEmails(ctx, store.EmailFilter{LabelID: "inbox", PerPage: 100})
		if err != nil {
			t.Fatal(err)
		}
		if result.Total < 1 {
			t.Error("expected at least 1 email in inbox after seeding")
		}
	})

	t.Run("seeded emails form threads", func(t *testing.T) {
		threads, err := s.ListThreads(ctx, store.EmailFilter{PerPage: 100})
		if err != nil {
			t.Fatal(err)
		}
		if threads.Total < 5 {
			t.Errorf("expected at least 5 threads, got %d", threads.Total)
		}

		// Find a multi-email thread
		foundMulti := false
		for _, thread := range threads.Threads {
			if thread.EmailCount > 1 {
				foundMulti = true
				break
			}
		}
		if !foundMulti {
			t.Error("expected at least one thread with multiple emails")
		}
	})

	t.Run("seeded emails include drafts", func(t *testing.T) {
		isDraft := true
		result, err := s.ListEmails(ctx, store.EmailFilter{IsDraft: &isDraft, PerPage: 100})
		if err != nil {
			t.Fatal(err)
		}
		if result.Total < 1 {
			t.Error("expected at least 1 draft email after seeding")
		}
	})

	t.Run("seeded emails are searchable via FTS", func(t *testing.T) {
		result, err := s.SearchEmails(ctx, "migration", 1, 25)
		if err != nil {
			t.Fatalf("SearchEmails failed: %v", err)
		}
		if result.Total < 1 {
			t.Error("expected at least 1 search result for 'migration' in seeded data")
		}
	})
}

// ---------------------------------------------------------------------------
// Search query parser (unit tests for unexported parseSearchQuery)
// ---------------------------------------------------------------------------

func TestParseSearchQuery(t *testing.T) {
	t.Run("parses from: operator", func(t *testing.T) {
		sf := parseSearchQuery("from:alice")
		if sf.from != "alice" {
			t.Errorf("from: want 'alice', got %q", sf.from)
		}
		if sf.freeText != "" {
			t.Errorf("freeText: want empty, got %q", sf.freeText)
		}
	})

	t.Run("parses to: operator", func(t *testing.T) {
		sf := parseSearchQuery("to:bob@corp.com")
		if sf.to != "bob@corp.com" {
			t.Errorf("to: want 'bob@corp.com', got %q", sf.to)
		}
	})

	t.Run("parses subject: operator", func(t *testing.T) {
		sf := parseSearchQuery("subject:migration")
		if sf.subject != "migration" {
			t.Errorf("subject: want 'migration', got %q", sf.subject)
		}
	})

	t.Run("parses has:attachment", func(t *testing.T) {
		sf := parseSearchQuery("has:attachment")
		if !sf.hasAttachment {
			t.Error("expected hasAttachment to be true")
		}
	})

	t.Run("parses is:unread", func(t *testing.T) {
		sf := parseSearchQuery("is:unread")
		if sf.isUnread == nil || !*sf.isUnread {
			t.Error("expected isUnread to be true")
		}
	})

	t.Run("parses is:read", func(t *testing.T) {
		sf := parseSearchQuery("is:read")
		if sf.isUnread == nil || *sf.isUnread {
			t.Error("expected isUnread to be false for is:read")
		}
	})

	t.Run("parses is:starred", func(t *testing.T) {
		sf := parseSearchQuery("is:starred")
		if sf.isStarred == nil || !*sf.isStarred {
			t.Error("expected isStarred to be true")
		}
	})

	t.Run("parses is:important", func(t *testing.T) {
		sf := parseSearchQuery("is:important")
		if sf.isImportant == nil || !*sf.isImportant {
			t.Error("expected isImportant to be true")
		}
	})

	t.Run("parses before: date", func(t *testing.T) {
		sf := parseSearchQuery("before:2024/07/01")
		if sf.before == "" {
			t.Fatal("expected non-empty before value")
		}
		// Should parse to RFC3339 for 2024-07-01
		expected := "2024-07-01T00:00:00Z"
		if sf.before != expected {
			t.Errorf("before: want %q, got %q", expected, sf.before)
		}
	})

	t.Run("parses after: date with dashes", func(t *testing.T) {
		sf := parseSearchQuery("after:2024-06-15")
		if sf.after == "" {
			t.Fatal("expected non-empty after value")
		}
		expected := "2024-06-15T00:00:00Z"
		if sf.after != expected {
			t.Errorf("after: want %q, got %q", expected, sf.after)
		}
	})

	t.Run("parses label: operator", func(t *testing.T) {
		sf := parseSearchQuery("label:work")
		if sf.label != "work" {
			t.Errorf("label: want 'work', got %q", sf.label)
		}
	})

	t.Run("extracts free text alongside operators", func(t *testing.T) {
		sf := parseSearchQuery("from:alice budget report")
		if sf.from != "alice" {
			t.Errorf("from: want 'alice', got %q", sf.from)
		}
		if sf.freeText != "budget report" {
			t.Errorf("freeText: want 'budget report', got %q", sf.freeText)
		}
	})

	t.Run("handles quoted subject", func(t *testing.T) {
		sf := parseSearchQuery(`subject:"Budget Report"`)
		if sf.subject != "Budget Report" {
			t.Errorf("subject: want 'Budget Report', got %q", sf.subject)
		}
	})

	t.Run("handles multiple operators", func(t *testing.T) {
		sf := parseSearchQuery("from:alice is:unread has:attachment")
		if sf.from != "alice" {
			t.Errorf("from: want 'alice', got %q", sf.from)
		}
		if sf.isUnread == nil || !*sf.isUnread {
			t.Error("expected isUnread to be true")
		}
		if !sf.hasAttachment {
			t.Error("expected hasAttachment to be true")
		}
		if sf.freeText != "" {
			t.Errorf("freeText: want empty, got %q", sf.freeText)
		}
	})

	t.Run("unknown operator treated as free text", func(t *testing.T) {
		sf := parseSearchQuery("unknown:value plaintext")
		if sf.freeText != "unknown:value plaintext" {
			t.Errorf("freeText: want 'unknown:value plaintext', got %q", sf.freeText)
		}
	})

	t.Run("empty query returns empty filter", func(t *testing.T) {
		sf := parseSearchQuery("")
		if sf.from != "" || sf.to != "" || sf.subject != "" || sf.label != "" || sf.freeText != "" {
			t.Error("expected all fields empty for empty query")
		}
		if sf.hasAttachment || sf.isUnread != nil || sf.isStarred != nil || sf.isImportant != nil {
			t.Error("expected all boolean filters unset for empty query")
		}
	})
}

func TestTokenizeSearch(t *testing.T) {
	t.Run("splits on spaces", func(t *testing.T) {
		tokens := tokenizeSearch("hello world")
		if len(tokens) != 2 || tokens[0] != "hello" || tokens[1] != "world" {
			t.Errorf("want [hello world], got %v", tokens)
		}
	})

	t.Run("respects quoted strings", func(t *testing.T) {
		tokens := tokenizeSearch(`subject:"hello world" from:alice`)
		if len(tokens) != 2 {
			t.Fatalf("want 2 tokens, got %d: %v", len(tokens), tokens)
		}
		if tokens[0] != "subject:hello world" {
			t.Errorf("token[0]: want 'subject:hello world', got %q", tokens[0])
		}
		if tokens[1] != "from:alice" {
			t.Errorf("token[1]: want 'from:alice', got %q", tokens[1])
		}
	})

	t.Run("handles empty input", func(t *testing.T) {
		tokens := tokenizeSearch("")
		if len(tokens) != 0 {
			t.Errorf("want 0 tokens, got %d", len(tokens))
		}
	})

	t.Run("collapses multiple spaces", func(t *testing.T) {
		tokens := tokenizeSearch("hello   world")
		if len(tokens) != 2 {
			t.Errorf("want 2 tokens, got %d: %v", len(tokens), tokens)
		}
	})
}

func TestParseDateOperator(t *testing.T) {
	t.Run("parses YYYY/MM/DD format", func(t *testing.T) {
		result := parseDateOperator("2024/07/15")
		if result != "2024-07-15T00:00:00Z" {
			t.Errorf("want '2024-07-15T00:00:00Z', got %q", result)
		}
	})

	t.Run("parses YYYY-MM-DD format", func(t *testing.T) {
		result := parseDateOperator("2024-07-15")
		if result != "2024-07-15T00:00:00Z" {
			t.Errorf("want '2024-07-15T00:00:00Z', got %q", result)
		}
	})

	t.Run("parses single-digit month/day", func(t *testing.T) {
		result := parseDateOperator("2024-1-5")
		if result != "2024-01-05T00:00:00Z" {
			t.Errorf("want '2024-01-05T00:00:00Z', got %q", result)
		}
	})

	t.Run("returns empty for invalid date", func(t *testing.T) {
		result := parseDateOperator("not-a-date")
		if result != "" {
			t.Errorf("want empty, got %q", result)
		}
	})
}

// ---------------------------------------------------------------------------
// Advanced search integration (through SearchEmails)
// ---------------------------------------------------------------------------

func TestSearchEmailsWithOperators(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()
	seedSystemLabels(t, s)

	workLabel := &types.Label{ID: "work", Name: "Work", Type: types.LabelTypeUser, Visible: true}
	if err := s.CreateLabel(ctx, workLabel); err != nil {
		t.Fatal(err)
	}

	// Create test emails with specific characteristics
	alice := makeTestEmail("Budget Report Q3")
	alice.FromAddress = "alice@corp.com"
	alice.FromName = "Alice Johnson"
	alice.ToAddresses = []types.Recipient{{Address: "bob@corp.com", Name: "Bob Smith"}}
	alice.IsStarred = true
	alice.IsRead = true
	alice.HasAttachments = true
	alice.ReceivedAt = time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC)
	alice.Labels = []string{"inbox", "work"}

	bob := makeTestEmail("Team Standup Notes")
	bob.FromAddress = "bob@corp.com"
	bob.FromName = "Bob Smith"
	bob.ToAddresses = []types.Recipient{{Address: "alice@corp.com", Name: "Alice Johnson"}}
	bob.IsImportant = true
	bob.ReceivedAt = time.Date(2024, 7, 20, 14, 0, 0, 0, time.UTC)
	bob.Labels = []string{"inbox"}

	carol := makeTestEmail("Database Migration Plan")
	carol.FromAddress = "carol@corp.com"
	carol.FromName = "Carol Williams"
	carol.ToAddresses = []types.Recipient{{Address: "team@corp.com", Name: "Team"}}
	carol.ReceivedAt = time.Date(2024, 8, 1, 9, 0, 0, 0, time.UTC)
	carol.Labels = []string{"inbox", "work"}

	for _, email := range []*types.Email{alice, bob, carol} {
		if err := s.CreateEmail(ctx, email); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("from: operator filters by sender", func(t *testing.T) {
		result, err := s.SearchEmails(ctx, "from:alice", 1, 25)
		if err != nil {
			t.Fatalf("SearchEmails failed: %v", err)
		}
		if result.Total != 1 {
			t.Errorf("expected 1 result for from:alice, got %d", result.Total)
		}
		if result.Total > 0 && result.Emails[0].FromAddress != "alice@corp.com" {
			t.Errorf("expected email from alice, got from %s", result.Emails[0].FromAddress)
		}
	})

	t.Run("to: operator filters by recipient", func(t *testing.T) {
		result, err := s.SearchEmails(ctx, "to:team", 1, 25)
		if err != nil {
			t.Fatalf("SearchEmails failed: %v", err)
		}
		if result.Total != 1 {
			t.Errorf("expected 1 result for to:team, got %d", result.Total)
		}
	})

	t.Run("subject: operator filters by subject", func(t *testing.T) {
		result, err := s.SearchEmails(ctx, "subject:Migration", 1, 25)
		if err != nil {
			t.Fatalf("SearchEmails failed: %v", err)
		}
		if result.Total != 1 {
			t.Errorf("expected 1 result for subject:Migration, got %d", result.Total)
		}
	})

	t.Run("has:attachment filters emails with attachments", func(t *testing.T) {
		result, err := s.SearchEmails(ctx, "has:attachment", 1, 25)
		if err != nil {
			t.Fatalf("SearchEmails failed: %v", err)
		}
		if result.Total != 1 {
			t.Errorf("expected 1 result for has:attachment, got %d", result.Total)
		}
	})

	t.Run("is:unread filters unread emails", func(t *testing.T) {
		result, err := s.SearchEmails(ctx, "is:unread", 1, 25)
		if err != nil {
			t.Fatalf("SearchEmails failed: %v", err)
		}
		if result.Total != 2 {
			t.Errorf("expected 2 results for is:unread, got %d", result.Total)
		}
	})

	t.Run("is:read filters read emails", func(t *testing.T) {
		result, err := s.SearchEmails(ctx, "is:read", 1, 25)
		if err != nil {
			t.Fatalf("SearchEmails failed: %v", err)
		}
		if result.Total != 1 {
			t.Errorf("expected 1 result for is:read, got %d", result.Total)
		}
	})

	t.Run("is:starred filters starred emails", func(t *testing.T) {
		result, err := s.SearchEmails(ctx, "is:starred", 1, 25)
		if err != nil {
			t.Fatalf("SearchEmails failed: %v", err)
		}
		if result.Total != 1 {
			t.Errorf("expected 1 result for is:starred, got %d", result.Total)
		}
	})

	t.Run("is:important filters important emails", func(t *testing.T) {
		result, err := s.SearchEmails(ctx, "is:important", 1, 25)
		if err != nil {
			t.Fatalf("SearchEmails failed: %v", err)
		}
		if result.Total != 1 {
			t.Errorf("expected 1 result for is:important, got %d", result.Total)
		}
	})

	t.Run("before: filters by date", func(t *testing.T) {
		result, err := s.SearchEmails(ctx, "before:2024/07/01", 1, 25)
		if err != nil {
			t.Fatalf("SearchEmails failed: %v", err)
		}
		if result.Total != 1 {
			t.Errorf("expected 1 result for before:2024/07/01, got %d", result.Total)
		}
	})

	t.Run("after: filters by date", func(t *testing.T) {
		result, err := s.SearchEmails(ctx, "after:2024/07/01", 1, 25)
		if err != nil {
			t.Fatalf("SearchEmails failed: %v", err)
		}
		if result.Total != 2 {
			t.Errorf("expected 2 results for after:2024/07/01, got %d", result.Total)
		}
	})

	t.Run("label: filters by label", func(t *testing.T) {
		result, err := s.SearchEmails(ctx, "label:work", 1, 25)
		if err != nil {
			t.Fatalf("SearchEmails failed: %v", err)
		}
		if result.Total != 2 {
			t.Errorf("expected 2 results for label:work, got %d", result.Total)
		}
	})

	t.Run("combined operators narrow results", func(t *testing.T) {
		result, err := s.SearchEmails(ctx, "from:alice is:starred", 1, 25)
		if err != nil {
			t.Fatalf("SearchEmails failed: %v", err)
		}
		if result.Total != 1 {
			t.Errorf("expected 1 result for from:alice is:starred, got %d", result.Total)
		}
	})

	t.Run("operator with free text", func(t *testing.T) {
		result, err := s.SearchEmails(ctx, "from:carol migration", 1, 25)
		if err != nil {
			t.Fatalf("SearchEmails failed: %v", err)
		}
		if result.Total != 1 {
			t.Errorf("expected 1 result for 'from:carol migration', got %d", result.Total)
		}
	})

	t.Run("no results for non-matching operator", func(t *testing.T) {
		result, err := s.SearchEmails(ctx, "from:nobody", 1, 25)
		if err != nil {
			t.Fatalf("SearchEmails failed: %v", err)
		}
		if result.Total != 0 {
			t.Errorf("expected 0 results for from:nobody, got %d", result.Total)
		}
	})
}

// ---------------------------------------------------------------------------
// Mute via UpdateEmail and Batch
// ---------------------------------------------------------------------------

func TestUpdateEmailMute(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	t.Run("mute email", func(t *testing.T) {
		email := makeTestEmail("Mute Test")
		if err := s.CreateEmail(ctx, email); err != nil {
			t.Fatal(err)
		}

		err := s.UpdateEmail(ctx, email.ID, map[string]any{"is_muted": true})
		if err != nil {
			t.Fatalf("UpdateEmail failed: %v", err)
		}

		got, err := s.GetEmail(ctx, email.ID)
		if err != nil {
			t.Fatal(err)
		}
		if !got.IsMuted {
			t.Error("expected is_muted to be true")
		}
	})

	t.Run("unmute email", func(t *testing.T) {
		email := makeTestEmail("Unmute Test")
		email.IsMuted = true
		if err := s.CreateEmail(ctx, email); err != nil {
			t.Fatal(err)
		}

		err := s.UpdateEmail(ctx, email.ID, map[string]any{"is_muted": false})
		if err != nil {
			t.Fatalf("UpdateEmail failed: %v", err)
		}

		got, err := s.GetEmail(ctx, email.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.IsMuted {
			t.Error("expected is_muted to be false")
		}
	})
}

func TestBatchMuteUnmute(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	t.Run("batch mute marks emails as muted", func(t *testing.T) {
		var ids []string
		for i := range 3 {
			email := makeTestEmail(fmt.Sprintf("Mute Batch %d", i))
			if err := s.CreateEmail(ctx, email); err != nil {
				t.Fatal(err)
			}
			ids = append(ids, email.ID)
		}

		err := s.BatchUpdateEmails(ctx, &types.BatchAction{
			IDs:    ids,
			Action: "mute",
		})
		if err != nil {
			t.Fatalf("BatchUpdateEmails (mute) failed: %v", err)
		}

		for _, id := range ids {
			got, err := s.GetEmail(ctx, id)
			if err != nil {
				t.Fatal(err)
			}
			if !got.IsMuted {
				t.Errorf("email %s is not muted after batch mute", id)
			}
		}
	})

	t.Run("batch unmute clears muted flag", func(t *testing.T) {
		var ids []string
		for i := range 2 {
			email := makeTestEmail(fmt.Sprintf("Unmute Batch %d", i))
			email.IsMuted = true
			if err := s.CreateEmail(ctx, email); err != nil {
				t.Fatal(err)
			}
			ids = append(ids, email.ID)
		}

		err := s.BatchUpdateEmails(ctx, &types.BatchAction{
			IDs:    ids,
			Action: "unmute",
		})
		if err != nil {
			t.Fatalf("BatchUpdateEmails (unmute) failed: %v", err)
		}

		for _, id := range ids {
			got, err := s.GetEmail(ctx, id)
			if err != nil {
				t.Fatal(err)
			}
			if got.IsMuted {
				t.Errorf("email %s is still muted after batch unmute", id)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// Snooze and Schedule via UpdateEmail
// ---------------------------------------------------------------------------

func TestUpdateEmailSnooze(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	t.Run("set snoozed_until", func(t *testing.T) {
		email := makeTestEmail("Snooze Test")
		if err := s.CreateEmail(ctx, email); err != nil {
			t.Fatal(err)
		}

		snoozeTime := time.Now().Add(24 * time.Hour)
		err := s.UpdateEmail(ctx, email.ID, map[string]any{
			"snoozed_until": snoozeTime,
		})
		if err != nil {
			t.Fatalf("UpdateEmail failed: %v", err)
		}

		got, err := s.GetEmail(ctx, email.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.SnoozedUntil == nil {
			t.Fatal("expected snoozed_until to be set")
		}
		// Compare to second precision since SQLite stores RFC3339
		if got.SnoozedUntil.Unix() != snoozeTime.Unix() {
			t.Errorf("snoozed_until: want %v, got %v", snoozeTime, *got.SnoozedUntil)
		}
	})

	t.Run("clear snoozed_until", func(t *testing.T) {
		email := makeTestEmail("Unsnooze Test")
		snoozeTime := time.Now().Add(24 * time.Hour)
		email.SnoozedUntil = &snoozeTime
		if err := s.CreateEmail(ctx, email); err != nil {
			t.Fatal(err)
		}

		err := s.UpdateEmail(ctx, email.ID, map[string]any{
			"snoozed_until": (*time.Time)(nil),
		})
		if err != nil {
			t.Fatalf("UpdateEmail failed: %v", err)
		}

		got, err := s.GetEmail(ctx, email.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.SnoozedUntil != nil {
			t.Errorf("expected snoozed_until to be nil, got %v", *got.SnoozedUntil)
		}
	})
}

func TestUpdateEmailSchedule(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	t.Run("set scheduled_at", func(t *testing.T) {
		email := makeTestEmail("Schedule Test")
		email.IsDraft = true
		if err := s.CreateEmail(ctx, email); err != nil {
			t.Fatal(err)
		}

		schedTime := time.Now().Add(48 * time.Hour)
		err := s.UpdateEmail(ctx, email.ID, map[string]any{
			"scheduled_at": schedTime,
		})
		if err != nil {
			t.Fatalf("UpdateEmail failed: %v", err)
		}

		got, err := s.GetEmail(ctx, email.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.ScheduledAt == nil {
			t.Fatal("expected scheduled_at to be set")
		}
		if got.ScheduledAt.Unix() != schedTime.Unix() {
			t.Errorf("scheduled_at: want %v, got %v", schedTime, *got.ScheduledAt)
		}
	})

	t.Run("clear scheduled_at", func(t *testing.T) {
		email := makeTestEmail("Unschedule Test")
		email.IsDraft = true
		schedTime := time.Now().Add(48 * time.Hour)
		email.ScheduledAt = &schedTime
		if err := s.CreateEmail(ctx, email); err != nil {
			t.Fatal(err)
		}

		err := s.UpdateEmail(ctx, email.ID, map[string]any{
			"scheduled_at": (*time.Time)(nil),
		})
		if err != nil {
			t.Fatalf("UpdateEmail failed: %v", err)
		}

		got, err := s.GetEmail(ctx, email.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.ScheduledAt != nil {
			t.Errorf("expected scheduled_at to be nil, got %v", *got.ScheduledAt)
		}
	})
}
