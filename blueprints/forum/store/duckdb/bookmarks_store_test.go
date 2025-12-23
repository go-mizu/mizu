package duckdb

import (
	"context"
	"testing"

	"github.com/go-mizu/mizu/blueprints/forum/feature/bookmarks"
)

func TestBookmarksStore_Create(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	author := createTestAccount(t, store, "author")

	bookmark := &bookmarks.Bookmark{
		ID:         newTestID(),
		AccountID:  author.ID,
		TargetType: bookmarks.TargetThread,
		TargetID:   newTestID(),
		CreatedAt:  testTime(),
	}

	if err := store.Bookmarks().Create(ctx, bookmark); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify
	got, err := store.Bookmarks().GetByTarget(ctx, bookmark.AccountID, bookmark.TargetType, bookmark.TargetID)
	if err != nil {
		t.Fatalf("GetByTarget failed: %v", err)
	}

	if got.ID != bookmark.ID {
		t.Errorf("ID: got %q, want %q", got.ID, bookmark.ID)
	}
}

func TestBookmarksStore_GetByTarget(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	author := createTestAccount(t, store, "author")
	targetID := newTestID()

	bookmark := &bookmarks.Bookmark{
		ID:         newTestID(),
		AccountID:  author.ID,
		TargetType: bookmarks.TargetThread,
		TargetID:   targetID,
		CreatedAt:  testTime(),
	}

	if err := store.Bookmarks().Create(ctx, bookmark); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := store.Bookmarks().GetByTarget(ctx, author.ID, bookmarks.TargetThread, targetID)
	if err != nil {
		t.Fatalf("GetByTarget failed: %v", err)
	}

	if got.AccountID != author.ID {
		t.Errorf("AccountID: got %q, want %q", got.AccountID, author.ID)
	}
	if got.TargetID != targetID {
		t.Errorf("TargetID: got %q, want %q", got.TargetID, targetID)
	}
}

func TestBookmarksStore_GetByTarget_NotFound(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	_, err := store.Bookmarks().GetByTarget(ctx, "account", "thread", "nonexistent")
	if err != bookmarks.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestBookmarksStore_Delete(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	author := createTestAccount(t, store, "author")
	targetID := newTestID()

	bookmark := &bookmarks.Bookmark{
		ID:         newTestID(),
		AccountID:  author.ID,
		TargetType: bookmarks.TargetThread,
		TargetID:   targetID,
		CreatedAt:  testTime(),
	}

	if err := store.Bookmarks().Create(ctx, bookmark); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := store.Bookmarks().Delete(ctx, author.ID, bookmarks.TargetThread, targetID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err := store.Bookmarks().GetByTarget(ctx, author.ID, bookmarks.TargetThread, targetID)
	if err != bookmarks.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestBookmarksStore_List(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	author := createTestAccount(t, store, "author")

	// Create multiple bookmarks
	for i := 0; i < 5; i++ {
		bookmark := &bookmarks.Bookmark{
			ID:         newTestID(),
			AccountID:  author.ID,
			TargetType: bookmarks.TargetThread,
			TargetID:   newTestID(),
			CreatedAt:  testTime(),
		}

		if err := store.Bookmarks().Create(ctx, bookmark); err != nil {
			t.Fatalf("Create bookmark %d failed: %v", i, err)
		}
	}

	// Also create comment bookmarks
	for i := 0; i < 3; i++ {
		bookmark := &bookmarks.Bookmark{
			ID:         newTestID(),
			AccountID:  author.ID,
			TargetType: bookmarks.TargetComment,
			TargetID:   newTestID(),
			CreatedAt:  testTime(),
		}

		if err := store.Bookmarks().Create(ctx, bookmark); err != nil {
			t.Fatalf("Create comment bookmark %d failed: %v", i, err)
		}
	}

	// List thread bookmarks
	threadBookmarks, err := store.Bookmarks().List(ctx, author.ID, bookmarks.TargetThread, bookmarks.ListOpts{Limit: 10})
	if err != nil {
		t.Fatalf("List threads failed: %v", err)
	}

	if len(threadBookmarks) != 5 {
		t.Errorf("Thread bookmarks count: got %d, want 5", len(threadBookmarks))
	}

	// List comment bookmarks
	commentBookmarks, err := store.Bookmarks().List(ctx, author.ID, bookmarks.TargetComment, bookmarks.ListOpts{Limit: 10})
	if err != nil {
		t.Fatalf("List comments failed: %v", err)
	}

	if len(commentBookmarks) != 3 {
		t.Errorf("Comment bookmarks count: got %d, want 3", len(commentBookmarks))
	}
}

func TestBookmarksStore_List_Limit(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	author := createTestAccount(t, store, "author")

	// Create 10 bookmarks
	for i := 0; i < 10; i++ {
		bookmark := &bookmarks.Bookmark{
			ID:         newTestID(),
			AccountID:  author.ID,
			TargetType: bookmarks.TargetThread,
			TargetID:   newTestID(),
			CreatedAt:  testTime(),
		}

		if err := store.Bookmarks().Create(ctx, bookmark); err != nil {
			t.Fatalf("Create bookmark %d failed: %v", i, err)
		}
	}

	// List with limit
	list, err := store.Bookmarks().List(ctx, author.ID, bookmarks.TargetThread, bookmarks.ListOpts{Limit: 5})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != 5 {
		t.Errorf("List count: got %d, want 5", len(list))
	}
}

func TestBookmarksStore_BookmarkComment(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	author := createTestAccount(t, store, "author")
	commentID := newTestID()

	bookmark := &bookmarks.Bookmark{
		ID:         newTestID(),
		AccountID:  author.ID,
		TargetType: bookmarks.TargetComment,
		TargetID:   commentID,
		CreatedAt:  testTime(),
	}

	if err := store.Bookmarks().Create(ctx, bookmark); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := store.Bookmarks().GetByTarget(ctx, author.ID, bookmarks.TargetComment, commentID)
	if err != nil {
		t.Fatalf("GetByTarget failed: %v", err)
	}

	if got.TargetType != bookmarks.TargetComment {
		t.Errorf("TargetType: got %q, want %q", got.TargetType, bookmarks.TargetComment)
	}
}

func TestBookmarksStore_DifferentUsers(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user1 := createTestAccount(t, store, "user1")
	user2 := createTestAccount(t, store, "user2")
	targetID := newTestID()

	// Both users bookmark same target
	bookmark1 := &bookmarks.Bookmark{
		ID:         newTestID(),
		AccountID:  user1.ID,
		TargetType: bookmarks.TargetThread,
		TargetID:   targetID,
		CreatedAt:  testTime(),
	}

	bookmark2 := &bookmarks.Bookmark{
		ID:         newTestID(),
		AccountID:  user2.ID,
		TargetType: bookmarks.TargetThread,
		TargetID:   targetID,
		CreatedAt:  testTime(),
	}

	if err := store.Bookmarks().Create(ctx, bookmark1); err != nil {
		t.Fatalf("Create bookmark1 failed: %v", err)
	}

	if err := store.Bookmarks().Create(ctx, bookmark2); err != nil {
		t.Fatalf("Create bookmark2 failed: %v", err)
	}

	// User1's bookmarks
	list1, err := store.Bookmarks().List(ctx, user1.ID, bookmarks.TargetThread, bookmarks.ListOpts{Limit: 10})
	if err != nil {
		t.Fatalf("List user1 failed: %v", err)
	}

	if len(list1) != 1 {
		t.Errorf("User1 bookmarks count: got %d, want 1", len(list1))
	}

	// User2's bookmarks
	list2, err := store.Bookmarks().List(ctx, user2.ID, bookmarks.TargetThread, bookmarks.ListOpts{Limit: 10})
	if err != nil {
		t.Fatalf("List user2 failed: %v", err)
	}

	if len(list2) != 1 {
		t.Errorf("User2 bookmarks count: got %d, want 1", len(list2))
	}
}

func TestBookmarksStore_DeleteNonExistent(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	// Deleting non-existent bookmark should not error
	err := store.Bookmarks().Delete(ctx, "account", "thread", "nonexistent")
	if err != nil {
		t.Errorf("Delete non-existent should not error, got %v", err)
	}
}
