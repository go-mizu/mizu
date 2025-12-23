package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/forum/feature/accounts"
	"github.com/go-mizu/mizu/blueprints/forum/feature/boards"
	"github.com/go-mizu/mizu/blueprints/forum/feature/comments"
	"github.com/go-mizu/mizu/blueprints/forum/feature/threads"
)

func createTestThread(t *testing.T, store *Store, board *boards.Board, author *accounts.Account) *threads.Thread {
	t.Helper()
	thread := &threads.Thread{
		ID:        newTestID(),
		BoardID:   board.ID,
		AuthorID:  author.ID,
		Title:     "Test Thread",
		Type:      threads.ThreadTypeText,
		CreatedAt: testTime(),
		UpdatedAt: testTime(),
	}
	if err := store.Threads().Create(context.Background(), thread); err != nil {
		t.Fatalf("createTestThread failed: %v", err)
	}
	return thread
}

func TestCommentsStore_Create(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	author := createTestAccount(t, store, "author")
	board := createTestBoard(t, store, author)
	thread := createTestThread(t, store, board, author)

	comment := &comments.Comment{
		ID:          newTestID(),
		ThreadID:    thread.ID,
		AuthorID:    author.ID,
		Content:     "This is a comment",
		ContentHTML: "<p>This is a comment</p>",
		Score:       0,
		Depth:       0,
		Path:        "",
		CreatedAt:   testTime(),
		UpdatedAt:   testTime(),
	}
	comment.Path = comment.ID

	if err := store.Comments().Create(ctx, comment); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify
	got, err := store.Comments().GetByID(ctx, comment.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if got.Content != comment.Content {
		t.Errorf("Content: got %q, want %q", got.Content, comment.Content)
	}
	if got.ThreadID != comment.ThreadID {
		t.Errorf("ThreadID: got %q, want %q", got.ThreadID, comment.ThreadID)
	}
	if got.Depth != 0 {
		t.Errorf("Depth: got %d, want 0", got.Depth)
	}
}

func TestCommentsStore_Create_Nested(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	author := createTestAccount(t, store, "author")
	board := createTestBoard(t, store, author)
	thread := createTestThread(t, store, board, author)

	// Create parent comment
	parent := &comments.Comment{
		ID:        newTestID(),
		ThreadID:  thread.ID,
		AuthorID:  author.ID,
		Content:   "Parent comment",
		Depth:     0,
		CreatedAt: testTime(),
		UpdatedAt: testTime(),
	}
	parent.Path = parent.ID

	if err := store.Comments().Create(ctx, parent); err != nil {
		t.Fatalf("Create parent failed: %v", err)
	}

	// Create child comment
	child := &comments.Comment{
		ID:        newTestID(),
		ThreadID:  thread.ID,
		ParentID:  parent.ID,
		AuthorID:  author.ID,
		Content:   "Child comment",
		Depth:     1,
		Path:      parent.Path + "/" + newTestID(),
		CreatedAt: testTime(),
		UpdatedAt: testTime(),
	}
	child.Path = parent.Path + "/" + child.ID

	if err := store.Comments().Create(ctx, child); err != nil {
		t.Fatalf("Create child failed: %v", err)
	}

	// Verify child
	got, err := store.Comments().GetByID(ctx, child.ID)
	if err != nil {
		t.Fatalf("GetByID child failed: %v", err)
	}

	if got.ParentID != parent.ID {
		t.Errorf("ParentID: got %q, want %q", got.ParentID, parent.ID)
	}
	if got.Depth != 1 {
		t.Errorf("Depth: got %d, want 1", got.Depth)
	}
}

func TestCommentsStore_GetByID(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	author := createTestAccount(t, store, "author")
	board := createTestBoard(t, store, author)
	thread := createTestThread(t, store, board, author)

	comment := &comments.Comment{
		ID:        newTestID(),
		ThreadID:  thread.ID,
		AuthorID:  author.ID,
		Content:   "Test comment",
		Depth:     0,
		CreatedAt: testTime(),
		UpdatedAt: testTime(),
	}
	comment.Path = comment.ID

	if err := store.Comments().Create(ctx, comment); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := store.Comments().GetByID(ctx, comment.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if got.ID != comment.ID {
		t.Errorf("ID: got %q, want %q", got.ID, comment.ID)
	}
}

func TestCommentsStore_GetByID_NotFound(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	_, err := store.Comments().GetByID(ctx, "nonexistent")
	if err != comments.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestCommentsStore_Update(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	author := createTestAccount(t, store, "author")
	board := createTestBoard(t, store, author)
	thread := createTestThread(t, store, board, author)

	comment := &comments.Comment{
		ID:        newTestID(),
		ThreadID:  thread.ID,
		AuthorID:  author.ID,
		Content:   "Original content",
		Depth:     0,
		CreatedAt: testTime(),
		UpdatedAt: testTime(),
	}
	comment.Path = comment.ID

	if err := store.Comments().Create(ctx, comment); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Update
	comment.Content = "Updated content"
	comment.ContentHTML = "<p>Updated content</p>"
	comment.Score = 50
	editedAt := testTime().Add(time.Hour)
	comment.EditedAt = &editedAt
	comment.UpdatedAt = testTime().Add(time.Hour)

	if err := store.Comments().Update(ctx, comment); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify
	got, err := store.Comments().GetByID(ctx, comment.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if got.Content != "Updated content" {
		t.Errorf("Content: got %q, want %q", got.Content, "Updated content")
	}
	if got.Score != 50 {
		t.Errorf("Score: got %d, want 50", got.Score)
	}
	if got.EditedAt == nil {
		t.Error("EditedAt: got nil, want non-nil")
	}
}

func TestCommentsStore_Delete(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	author := createTestAccount(t, store, "author")
	board := createTestBoard(t, store, author)
	thread := createTestThread(t, store, board, author)

	comment := &comments.Comment{
		ID:        newTestID(),
		ThreadID:  thread.ID,
		AuthorID:  author.ID,
		Content:   "Test comment",
		Depth:     0,
		CreatedAt: testTime(),
		UpdatedAt: testTime(),
	}
	comment.Path = comment.ID

	if err := store.Comments().Create(ctx, comment); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := store.Comments().Delete(ctx, comment.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err := store.Comments().GetByID(ctx, comment.ID)
	if err != comments.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestCommentsStore_ListByThread_Top(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	author := createTestAccount(t, store, "author")
	board := createTestBoard(t, store, author)
	thread := createTestThread(t, store, board, author)

	// Create comments with different scores
	for i := 0; i < 5; i++ {
		comment := &comments.Comment{
			ID:        newTestID(),
			ThreadID:  thread.ID,
			AuthorID:  author.ID,
			Content:   "Comment " + string(rune('A'+i)),
			Score:     int64(i * 100),
			Depth:     0,
			CreatedAt: testTime(),
			UpdatedAt: testTime(),
		}
		comment.Path = comment.ID

		if err := store.Comments().Create(ctx, comment); err != nil {
			t.Fatalf("Create comment %d failed: %v", i, err)
		}
	}

	list, err := store.Comments().ListByThread(ctx, thread.ID, comments.ListOpts{
		Limit:  10,
		SortBy: comments.CommentSortTop,
	})
	if err != nil {
		t.Fatalf("ListByThread failed: %v", err)
	}

	if len(list) != 5 {
		t.Errorf("List count: got %d, want 5", len(list))
	}

	// First should have highest score
	if list[0].Score < list[len(list)-1].Score {
		t.Error("List should be ordered by score DESC")
	}
}

func TestCommentsStore_ListByThread_New(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	author := createTestAccount(t, store, "author")
	board := createTestBoard(t, store, author)
	thread := createTestThread(t, store, board, author)

	// Create comments at different times
	for i := 0; i < 5; i++ {
		comment := &comments.Comment{
			ID:        newTestID(),
			ThreadID:  thread.ID,
			AuthorID:  author.ID,
			Content:   "Comment " + string(rune('A'+i)),
			Depth:     0,
			CreatedAt: testTime().Add(time.Duration(i) * time.Hour),
			UpdatedAt: testTime().Add(time.Duration(i) * time.Hour),
		}
		comment.Path = comment.ID

		if err := store.Comments().Create(ctx, comment); err != nil {
			t.Fatalf("Create comment %d failed: %v", i, err)
		}
	}

	list, err := store.Comments().ListByThread(ctx, thread.ID, comments.ListOpts{
		Limit:  10,
		SortBy: comments.CommentSortNew,
	})
	if err != nil {
		t.Fatalf("ListByThread failed: %v", err)
	}

	if len(list) != 5 {
		t.Errorf("List count: got %d, want 5", len(list))
	}

	// First should be newest
	if list[0].CreatedAt.Before(list[len(list)-1].CreatedAt) {
		t.Error("List should be ordered by created_at DESC")
	}
}

func TestCommentsStore_ListByThread_Old(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	author := createTestAccount(t, store, "author")
	board := createTestBoard(t, store, author)
	thread := createTestThread(t, store, board, author)

	// Create comments at different times
	for i := 0; i < 5; i++ {
		comment := &comments.Comment{
			ID:        newTestID(),
			ThreadID:  thread.ID,
			AuthorID:  author.ID,
			Content:   "Comment " + string(rune('A'+i)),
			Depth:     0,
			CreatedAt: testTime().Add(time.Duration(i) * time.Hour),
			UpdatedAt: testTime().Add(time.Duration(i) * time.Hour),
		}
		comment.Path = comment.ID

		if err := store.Comments().Create(ctx, comment); err != nil {
			t.Fatalf("Create comment %d failed: %v", i, err)
		}
	}

	list, err := store.Comments().ListByThread(ctx, thread.ID, comments.ListOpts{
		Limit:  10,
		SortBy: comments.CommentSortOld,
	})
	if err != nil {
		t.Fatalf("ListByThread failed: %v", err)
	}

	if len(list) != 5 {
		t.Errorf("List count: got %d, want 5", len(list))
	}

	// First should be oldest
	if list[0].CreatedAt.After(list[len(list)-1].CreatedAt) {
		t.Error("List should be ordered by created_at ASC")
	}
}

func TestCommentsStore_ListByParent(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	author := createTestAccount(t, store, "author")
	board := createTestBoard(t, store, author)
	thread := createTestThread(t, store, board, author)

	// Create parent comment
	parent := &comments.Comment{
		ID:        newTestID(),
		ThreadID:  thread.ID,
		AuthorID:  author.ID,
		Content:   "Parent",
		Depth:     0,
		CreatedAt: testTime(),
		UpdatedAt: testTime(),
	}
	parent.Path = parent.ID

	if err := store.Comments().Create(ctx, parent); err != nil {
		t.Fatalf("Create parent failed: %v", err)
	}

	// Create child comments
	for i := 0; i < 3; i++ {
		child := &comments.Comment{
			ID:        newTestID(),
			ThreadID:  thread.ID,
			ParentID:  parent.ID,
			AuthorID:  author.ID,
			Content:   "Child " + string(rune('A'+i)),
			Depth:     1,
			CreatedAt: testTime(),
			UpdatedAt: testTime(),
		}
		child.Path = parent.Path + "/" + child.ID

		if err := store.Comments().Create(ctx, child); err != nil {
			t.Fatalf("Create child %d failed: %v", i, err)
		}
	}

	// Create another root-level comment
	other := &comments.Comment{
		ID:        newTestID(),
		ThreadID:  thread.ID,
		AuthorID:  author.ID,
		Content:   "Other",
		Depth:     0,
		CreatedAt: testTime(),
		UpdatedAt: testTime(),
	}
	other.Path = other.ID

	if err := store.Comments().Create(ctx, other); err != nil {
		t.Fatalf("Create other failed: %v", err)
	}

	// List by parent
	children, err := store.Comments().ListByParent(ctx, parent.ID, comments.ListOpts{Limit: 10})
	if err != nil {
		t.Fatalf("ListByParent failed: %v", err)
	}

	if len(children) != 3 {
		t.Errorf("Children count: got %d, want 3", len(children))
	}
}

func TestCommentsStore_ListByAuthor(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	author1 := createTestAccount(t, store, "author1")
	author2 := createTestAccount(t, store, "author2")
	board := createTestBoard(t, store, author1)
	thread := createTestThread(t, store, board, author1)

	// Create comments by different authors
	for i := 0; i < 3; i++ {
		comment := &comments.Comment{
			ID:        newTestID(),
			ThreadID:  thread.ID,
			AuthorID:  author1.ID,
			Content:   "Author1 Comment " + string(rune('A'+i)),
			Depth:     0,
			CreatedAt: testTime(),
			UpdatedAt: testTime(),
		}
		comment.Path = comment.ID

		if err := store.Comments().Create(ctx, comment); err != nil {
			t.Fatalf("Create comment %d failed: %v", i, err)
		}
	}

	for i := 0; i < 2; i++ {
		comment := &comments.Comment{
			ID:        newTestID(),
			ThreadID:  thread.ID,
			AuthorID:  author2.ID,
			Content:   "Author2 Comment " + string(rune('A'+i)),
			Depth:     0,
			CreatedAt: testTime(),
			UpdatedAt: testTime(),
		}
		comment.Path = comment.ID

		if err := store.Comments().Create(ctx, comment); err != nil {
			t.Fatalf("Create comment %d failed: %v", i, err)
		}
	}

	// List by author1
	list1, err := store.Comments().ListByAuthor(ctx, author1.ID, comments.ListOpts{Limit: 10})
	if err != nil {
		t.Fatalf("ListByAuthor failed: %v", err)
	}

	if len(list1) != 3 {
		t.Errorf("List author1 count: got %d, want 3", len(list1))
	}

	// List by author2
	list2, err := store.Comments().ListByAuthor(ctx, author2.ID, comments.ListOpts{Limit: 10})
	if err != nil {
		t.Fatalf("ListByAuthor failed: %v", err)
	}

	if len(list2) != 2 {
		t.Errorf("List author2 count: got %d, want 2", len(list2))
	}
}

func TestCommentsStore_IncrementChildCount(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	author := createTestAccount(t, store, "author")
	board := createTestBoard(t, store, author)
	thread := createTestThread(t, store, board, author)

	comment := &comments.Comment{
		ID:         newTestID(),
		ThreadID:   thread.ID,
		AuthorID:   author.ID,
		Content:    "Parent",
		Depth:      0,
		ChildCount: 0,
		CreatedAt:  testTime(),
		UpdatedAt:  testTime(),
	}
	comment.Path = comment.ID

	if err := store.Comments().Create(ctx, comment); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Increment
	if err := store.Comments().IncrementChildCount(ctx, comment.ID, 5); err != nil {
		t.Fatalf("IncrementChildCount failed: %v", err)
	}

	got, err := store.Comments().GetByID(ctx, comment.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if got.ChildCount != 5 {
		t.Errorf("ChildCount: got %d, want 5", got.ChildCount)
	}

	// Decrement
	if err := store.Comments().IncrementChildCount(ctx, comment.ID, -2); err != nil {
		t.Fatalf("IncrementChildCount (negative) failed: %v", err)
	}

	got2, err := store.Comments().GetByID(ctx, comment.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if got2.ChildCount != 3 {
		t.Errorf("ChildCount: got %d, want 3", got2.ChildCount)
	}
}

func TestCommentsStore_Removed_Excluded(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	author := createTestAccount(t, store, "author")
	board := createTestBoard(t, store, author)
	thread := createTestThread(t, store, board, author)

	// Create normal comment
	normal := &comments.Comment{
		ID:        newTestID(),
		ThreadID:  thread.ID,
		AuthorID:  author.ID,
		Content:   "Normal",
		Depth:     0,
		IsRemoved: false,
		CreatedAt: testTime(),
		UpdatedAt: testTime(),
	}
	normal.Path = normal.ID

	if err := store.Comments().Create(ctx, normal); err != nil {
		t.Fatalf("Create normal failed: %v", err)
	}

	// Create removed comment
	removed := &comments.Comment{
		ID:           newTestID(),
		ThreadID:     thread.ID,
		AuthorID:     author.ID,
		Content:      "Removed",
		Depth:        0,
		IsRemoved:    true,
		RemoveReason: "Spam",
		CreatedAt:    testTime(),
		UpdatedAt:    testTime(),
	}
	removed.Path = removed.ID

	if err := store.Comments().Create(ctx, removed); err != nil {
		t.Fatalf("Create removed failed: %v", err)
	}

	// List should exclude removed
	list, err := store.Comments().ListByThread(ctx, thread.ID, comments.ListOpts{Limit: 10})
	if err != nil {
		t.Fatalf("ListByThread failed: %v", err)
	}

	if len(list) != 1 {
		t.Errorf("List count: got %d, want 1 (excluding removed)", len(list))
	}
}

func TestCommentsStore_Deleted(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	author := createTestAccount(t, store, "author")
	board := createTestBoard(t, store, author)
	thread := createTestThread(t, store, board, author)

	comment := &comments.Comment{
		ID:        newTestID(),
		ThreadID:  thread.ID,
		AuthorID:  author.ID,
		Content:   "Original content",
		Depth:     0,
		IsDeleted: false,
		CreatedAt: testTime(),
		UpdatedAt: testTime(),
	}
	comment.Path = comment.ID

	if err := store.Comments().Create(ctx, comment); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Mark as deleted
	comment.Content = "[deleted]"
	comment.IsDeleted = true
	comment.UpdatedAt = testTime().Add(time.Hour)

	if err := store.Comments().Update(ctx, comment); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, err := store.Comments().GetByID(ctx, comment.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if !got.IsDeleted {
		t.Error("IsDeleted: got false, want true")
	}
	if got.Content != "[deleted]" {
		t.Errorf("Content: got %q, want %q", got.Content, "[deleted]")
	}
}
