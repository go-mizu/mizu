package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/comments"
	"github.com/oklog/ulid/v2"
)

func createTestComment(t *testing.T, store *CommentsStore, targetType, targetID, userID string) *comments.Comment {
	t.Helper()
	id := ulid.Make().String()
	c := &comments.Comment{
		ID:         id,
		TargetType: targetType,
		TargetID:   targetID,
		UserID:     userID,
		Body:       "Test comment " + id[len(id)-8:],
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := store.Create(context.Background(), c); err != nil {
		t.Fatalf("failed to create test comment: %v", err)
	}
	return c
}

// =============================================================================
// Comment CRUD Tests
// =============================================================================

func TestCommentsStore_Create(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	_, userID := createRepoAndUser(t, store)
	commentsStore := NewCommentsStore(store.DB())

	c := &comments.Comment{
		ID:         ulid.Make().String(),
		TargetType: "issue",
		TargetID:   "issue-123",
		UserID:     userID,
		Body:       "This is a comment",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	err := commentsStore.Create(context.Background(), c)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := commentsStore.GetByID(context.Background(), c.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected comment to be created")
	}
	if got.Body != "This is a comment" {
		t.Errorf("got body %q, want %q", got.Body, "This is a comment")
	}
	if got.TargetType != "issue" {
		t.Errorf("got target_type %q, want %q", got.TargetType, "issue")
	}
}

func TestCommentsStore_GetByID(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	_, userID := createRepoAndUser(t, store)
	commentsStore := NewCommentsStore(store.DB())

	c := createTestComment(t, commentsStore, "issue", "issue-123", userID)

	got, err := commentsStore.GetByID(context.Background(), c.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected comment")
	}
	if got.ID != c.ID {
		t.Errorf("got ID %q, want %q", got.ID, c.ID)
	}
}

func TestCommentsStore_GetByID_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	commentsStore := NewCommentsStore(store.DB())

	got, err := commentsStore.GetByID(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent comment")
	}
}

func TestCommentsStore_Update(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	_, userID := createRepoAndUser(t, store)
	commentsStore := NewCommentsStore(store.DB())

	c := createTestComment(t, commentsStore, "issue", "issue-123", userID)

	c.Body = "Updated comment body"

	err := commentsStore.Update(context.Background(), c)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, _ := commentsStore.GetByID(context.Background(), c.ID)
	if got.Body != "Updated comment body" {
		t.Errorf("got body %q, want %q", got.Body, "Updated comment body")
	}
}

func TestCommentsStore_Update_UpdatesTimestamp(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	_, userID := createRepoAndUser(t, store)
	commentsStore := NewCommentsStore(store.DB())

	c := createTestComment(t, commentsStore, "issue", "issue-123", userID)
	originalUpdatedAt := c.UpdatedAt

	time.Sleep(10 * time.Millisecond)

	c.Body = "New body"
	c.UpdatedAt = time.Now() // Set new timestamp before update
	commentsStore.Update(context.Background(), c)

	got, _ := commentsStore.GetByID(context.Background(), c.ID)
	if !got.UpdatedAt.After(originalUpdatedAt) {
		t.Error("expected updated_at to be updated")
	}
}

func TestCommentsStore_Delete(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	_, userID := createRepoAndUser(t, store)
	commentsStore := NewCommentsStore(store.DB())

	c := createTestComment(t, commentsStore, "issue", "issue-123", userID)

	err := commentsStore.Delete(context.Background(), c.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	got, _ := commentsStore.GetByID(context.Background(), c.ID)
	if got != nil {
		t.Error("expected comment to be deleted")
	}
}

func TestCommentsStore_Delete_NonExistent(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	commentsStore := NewCommentsStore(store.DB())

	err := commentsStore.Delete(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("Delete should not error for non-existent comment: %v", err)
	}
}

// =============================================================================
// List Tests
// =============================================================================

func TestCommentsStore_List(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	_, userID := createRepoAndUser(t, store)
	commentsStore := NewCommentsStore(store.DB())

	for i := 0; i < 5; i++ {
		createTestComment(t, commentsStore, "issue", "issue-123", userID)
	}

	list, total, err := commentsStore.List(context.Background(), "issue", "issue-123", 10, 0)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 5 {
		t.Errorf("got %d comments, want 5", len(list))
	}
	if total != 5 {
		t.Errorf("got total %d, want 5", total)
	}
}

func TestCommentsStore_List_Pagination(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	_, userID := createRepoAndUser(t, store)
	commentsStore := NewCommentsStore(store.DB())

	for i := 0; i < 10; i++ {
		createTestComment(t, commentsStore, "issue", "issue-123", userID)
	}

	page1, total, _ := commentsStore.List(context.Background(), "issue", "issue-123", 3, 0)
	page2, _, _ := commentsStore.List(context.Background(), "issue", "issue-123", 3, 3)

	if len(page1) != 3 {
		t.Errorf("got %d comments on page 1, want 3", len(page1))
	}
	if len(page2) != 3 {
		t.Errorf("got %d comments on page 2, want 3", len(page2))
	}
	if total != 10 {
		t.Errorf("got total %d, want 10", total)
	}
	if page1[0].ID == page2[0].ID {
		t.Error("expected different comments on different pages")
	}
}

func TestCommentsStore_List_Empty(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	commentsStore := NewCommentsStore(store.DB())

	list, total, err := commentsStore.List(context.Background(), "issue", "issue-nonexistent", 10, 0)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 0 {
		t.Errorf("got total %d, want 0", total)
	}
	if list != nil && len(list) != 0 {
		t.Error("expected empty list")
	}
}

func TestCommentsStore_List_FilterByTarget(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	_, userID := createRepoAndUser(t, store)
	commentsStore := NewCommentsStore(store.DB())

	// Create comments for different targets
	for i := 0; i < 3; i++ {
		createTestComment(t, commentsStore, "issue", "issue-1", userID)
	}
	for i := 0; i < 2; i++ {
		createTestComment(t, commentsStore, "issue", "issue-2", userID)
	}
	for i := 0; i < 4; i++ {
		createTestComment(t, commentsStore, "pull_request", "pr-1", userID)
	}

	list1, total1, _ := commentsStore.List(context.Background(), "issue", "issue-1", 10, 0)
	list2, total2, _ := commentsStore.List(context.Background(), "issue", "issue-2", 10, 0)
	list3, total3, _ := commentsStore.List(context.Background(), "pull_request", "pr-1", 10, 0)

	if len(list1) != 3 || total1 != 3 {
		t.Errorf("got %d comments for issue-1, want 3", len(list1))
	}
	if len(list2) != 2 || total2 != 2 {
		t.Errorf("got %d comments for issue-2, want 2", len(list2))
	}
	if len(list3) != 4 || total3 != 4 {
		t.Errorf("got %d comments for pr-1, want 4", len(list3))
	}
}

// =============================================================================
// CountByTarget Tests
// =============================================================================

func TestCommentsStore_CountByTarget(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	_, userID := createRepoAndUser(t, store)
	commentsStore := NewCommentsStore(store.DB())

	for i := 0; i < 7; i++ {
		createTestComment(t, commentsStore, "issue", "issue-123", userID)
	}

	count, err := commentsStore.CountByTarget(context.Background(), "issue", "issue-123")
	if err != nil {
		t.Fatalf("CountByTarget failed: %v", err)
	}
	if count != 7 {
		t.Errorf("got count %d, want 7", count)
	}
}

func TestCommentsStore_CountByTarget_Empty(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	commentsStore := NewCommentsStore(store.DB())

	count, err := commentsStore.CountByTarget(context.Background(), "issue", "nonexistent")
	if err != nil {
		t.Fatalf("CountByTarget failed: %v", err)
	}
	if count != 0 {
		t.Errorf("got count %d, want 0", count)
	}
}

func TestCommentsStore_CountByTarget_PerTarget(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	_, userID := createRepoAndUser(t, store)
	commentsStore := NewCommentsStore(store.DB())

	for i := 0; i < 5; i++ {
		createTestComment(t, commentsStore, "issue", "issue-1", userID)
	}
	for i := 0; i < 3; i++ {
		createTestComment(t, commentsStore, "issue", "issue-2", userID)
	}

	count1, _ := commentsStore.CountByTarget(context.Background(), "issue", "issue-1")
	count2, _ := commentsStore.CountByTarget(context.Background(), "issue", "issue-2")

	if count1 != 5 {
		t.Errorf("got count for issue-1 %d, want 5", count1)
	}
	if count2 != 3 {
		t.Errorf("got count for issue-2 %d, want 3", count2)
	}
}

// Verify interface compliance
var _ comments.Store = (*CommentsStore)(nil)
