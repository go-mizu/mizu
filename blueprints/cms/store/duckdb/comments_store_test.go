package duckdb

import (
	"context"
	"testing"

	"github.com/go-mizu/blueprints/cms/feature/comments"
)

func TestCommentsStore_Create(t *testing.T) {
	db := setupTestDB(t)
	store := NewCommentsStore(db)
	ctx := context.Background()

	comment := &comments.Comment{
		ID:          "comment-001",
		PostID:      "post-001",
		AuthorID:    "user-001",
		AuthorName:  "John Doe",
		AuthorEmail: "john@example.com",
		AuthorURL:   "https://johndoe.com",
		Content:     "This is a great post!",
		Status:      "approved",
		IPAddress:   "192.168.1.1",
		UserAgent:   "Mozilla/5.0",
		LikesCount:  5,
		Meta:        `{"sentiment":"positive"}`,
		CreatedAt:   testTime,
		UpdatedAt:   testTime,
	}

	err := store.Create(ctx, comment)
	assertNoError(t, err)

	got, err := store.GetByID(ctx, comment.ID)
	assertNoError(t, err)
	assertEqual(t, "ID", got.ID, comment.ID)
	assertEqual(t, "PostID", got.PostID, comment.PostID)
	assertEqual(t, "Content", got.Content, comment.Content)
	assertEqual(t, "Status", got.Status, comment.Status)
	assertEqual(t, "AuthorName", got.AuthorName, comment.AuthorName)
}

func TestCommentsStore_Create_WithParent(t *testing.T) {
	db := setupTestDB(t)
	store := NewCommentsStore(db)
	ctx := context.Background()

	// Create parent comment
	parent := &comments.Comment{
		ID:        "comment-parent",
		PostID:    "post-001",
		Content:   "Parent comment",
		Status:    "approved",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.Create(ctx, parent))

	// Create reply
	reply := &comments.Comment{
		ID:        "comment-reply",
		PostID:    "post-001",
		ParentID:  parent.ID,
		Content:   "Reply comment",
		Status:    "approved",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	err := store.Create(ctx, reply)
	assertNoError(t, err)

	got, _ := store.GetByID(ctx, reply.ID)
	assertEqual(t, "ParentID", got.ParentID, parent.ID)
}

func TestCommentsStore_Create_AnonymousAuthor(t *testing.T) {
	db := setupTestDB(t)
	store := NewCommentsStore(db)
	ctx := context.Background()

	comment := &comments.Comment{
		ID:          "comment-anon",
		PostID:      "post-001",
		AuthorName:  "Anonymous User",
		AuthorEmail: "anon@example.com",
		Content:     "Anonymous comment",
		Status:      "pending",
		CreatedAt:   testTime,
		UpdatedAt:   testTime,
	}

	err := store.Create(ctx, comment)
	assertNoError(t, err)

	got, _ := store.GetByID(ctx, comment.ID)
	assertEqual(t, "AuthorID", got.AuthorID, "") // No author ID
	assertEqual(t, "AuthorName", got.AuthorName, "Anonymous User")
	assertEqual(t, "AuthorEmail", got.AuthorEmail, "anon@example.com")
}

func TestCommentsStore_GetByID(t *testing.T) {
	db := setupTestDB(t)
	store := NewCommentsStore(db)
	ctx := context.Background()

	comment := &comments.Comment{
		ID:        "comment-get",
		PostID:    "post-001",
		Content:   "Get comment",
		Status:    "approved",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.Create(ctx, comment))

	got, err := store.GetByID(ctx, comment.ID)
	assertNoError(t, err)
	assertEqual(t, "ID", got.ID, comment.ID)
}

func TestCommentsStore_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	store := NewCommentsStore(db)
	ctx := context.Background()

	got, err := store.GetByID(ctx, "nonexistent")
	assertNoError(t, err)
	if got != nil {
		t.Error("expected nil for non-existent comment")
	}
}

func TestCommentsStore_List(t *testing.T) {
	db := setupTestDB(t)
	store := NewCommentsStore(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		comment := &comments.Comment{
			ID:        "comment-list-" + string(rune('a'+i)),
			PostID:    "post-001",
			Content:   "Comment " + string(rune('A'+i)),
			Status:    "approved",
			CreatedAt: testTime,
			UpdatedAt: testTime,
		}
		assertNoError(t, store.Create(ctx, comment))
	}

	list, total, err := store.List(ctx, &comments.ListIn{Limit: 10})
	assertNoError(t, err)
	assertEqual(t, "total", total, 5)
	assertLen(t, list, 5)
}

func TestCommentsStore_List_FilterByPost(t *testing.T) {
	db := setupTestDB(t)
	store := NewCommentsStore(db)
	ctx := context.Background()

	posts := []string{"post-a", "post-a", "post-b"}
	for i, postID := range posts {
		comment := &comments.Comment{
			ID:        "comment-post-" + string(rune('a'+i)),
			PostID:    postID,
			Content:   "Comment",
			Status:    "approved",
			CreatedAt: testTime,
			UpdatedAt: testTime,
		}
		assertNoError(t, store.Create(ctx, comment))
	}

	list, total, err := store.List(ctx, &comments.ListIn{PostID: "post-a", Limit: 10})
	assertNoError(t, err)
	assertEqual(t, "total", total, 2)
	assertLen(t, list, 2)
}

func TestCommentsStore_List_FilterByParent(t *testing.T) {
	db := setupTestDB(t)
	store := NewCommentsStore(db)
	ctx := context.Background()

	// Create parent
	parent := &comments.Comment{
		ID:        "comment-parent-list",
		PostID:    "post-001",
		Content:   "Parent",
		Status:    "approved",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.Create(ctx, parent))

	// Create replies
	for i := 0; i < 3; i++ {
		reply := &comments.Comment{
			ID:        "comment-reply-" + string(rune('a'+i)),
			PostID:    "post-001",
			ParentID:  parent.ID,
			Content:   "Reply " + string(rune('A'+i)),
			Status:    "approved",
			CreatedAt: testTime,
			UpdatedAt: testTime,
		}
		assertNoError(t, store.Create(ctx, reply))
	}

	// Create standalone comment
	standalone := &comments.Comment{
		ID:        "comment-standalone",
		PostID:    "post-001",
		Content:   "Standalone",
		Status:    "approved",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.Create(ctx, standalone))

	list, total, err := store.List(ctx, &comments.ListIn{ParentID: parent.ID, Limit: 10})
	assertNoError(t, err)
	assertEqual(t, "total", total, 3)
	assertLen(t, list, 3)
}

func TestCommentsStore_List_FilterByAuthor(t *testing.T) {
	db := setupTestDB(t)
	store := NewCommentsStore(db)
	ctx := context.Background()

	authors := []string{"user-a", "user-a", "user-b"}
	for i, author := range authors {
		comment := &comments.Comment{
			ID:        "comment-author-" + string(rune('a'+i)),
			PostID:    "post-001",
			AuthorID:  author,
			Content:   "Comment",
			Status:    "approved",
			CreatedAt: testTime,
			UpdatedAt: testTime,
		}
		assertNoError(t, store.Create(ctx, comment))
	}

	list, total, err := store.List(ctx, &comments.ListIn{AuthorID: "user-a", Limit: 10})
	assertNoError(t, err)
	assertEqual(t, "total", total, 2)
	assertLen(t, list, 2)
}

func TestCommentsStore_List_FilterByStatus(t *testing.T) {
	db := setupTestDB(t)
	store := NewCommentsStore(db)
	ctx := context.Background()

	statuses := []string{"pending", "approved", "approved", "spam"}
	for i, status := range statuses {
		comment := &comments.Comment{
			ID:        "comment-status-" + string(rune('a'+i)),
			PostID:    "post-001",
			Content:   "Comment",
			Status:    status,
			CreatedAt: testTime,
			UpdatedAt: testTime,
		}
		assertNoError(t, store.Create(ctx, comment))
	}

	// Get pending
	list, total, err := store.List(ctx, &comments.ListIn{Status: "pending", Limit: 10})
	assertNoError(t, err)
	assertEqual(t, "total", total, 1)

	// Get approved
	list, total, err = store.List(ctx, &comments.ListIn{Status: "approved", Limit: 10})
	assertNoError(t, err)
	assertEqual(t, "total", total, 2)
	assertLen(t, list, 2)
}

func TestCommentsStore_ListByPost(t *testing.T) {
	db := setupTestDB(t)
	store := NewCommentsStore(db)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		comment := &comments.Comment{
			ID:        "comment-bypost-" + string(rune('a'+i)),
			PostID:    "post-target",
			Content:   "Comment " + string(rune('A'+i)),
			Status:    "approved",
			CreatedAt: testTime,
			UpdatedAt: testTime,
		}
		assertNoError(t, store.Create(ctx, comment))
	}

	// Add comment for different post
	other := &comments.Comment{
		ID:        "comment-other",
		PostID:    "post-other",
		Content:   "Other",
		Status:    "approved",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.Create(ctx, other))

	list, total, err := store.ListByPost(ctx, "post-target", &comments.ListIn{Limit: 10})
	assertNoError(t, err)
	assertEqual(t, "total", total, 3)
	assertLen(t, list, 3)
}

func TestCommentsStore_Update(t *testing.T) {
	db := setupTestDB(t)
	store := NewCommentsStore(db)
	ctx := context.Background()

	comment := &comments.Comment{
		ID:        "comment-update",
		PostID:    "post-001",
		Content:   "Original content",
		Status:    "pending",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.Create(ctx, comment))

	err := store.Update(ctx, comment.ID, &comments.UpdateIn{
		Content: ptr("Updated content"),
		Status:  ptr("approved"),
	})
	assertNoError(t, err)

	got, _ := store.GetByID(ctx, comment.ID)
	assertEqual(t, "Content", got.Content, "Updated content")
	assertEqual(t, "Status", got.Status, "approved")
}

func TestCommentsStore_Update_ChangeStatus(t *testing.T) {
	db := setupTestDB(t)
	store := NewCommentsStore(db)
	ctx := context.Background()

	comment := &comments.Comment{
		ID:        "comment-status-change",
		PostID:    "post-001",
		Content:   "Comment",
		Status:    "pending",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.Create(ctx, comment))

	// Change to approved
	err := store.Update(ctx, comment.ID, &comments.UpdateIn{
		Status: ptr("approved"),
	})
	assertNoError(t, err)

	got, _ := store.GetByID(ctx, comment.ID)
	assertEqual(t, "Status", got.Status, "approved")

	// Change to spam
	err = store.Update(ctx, comment.ID, &comments.UpdateIn{
		Status: ptr("spam"),
	})
	assertNoError(t, err)

	got, _ = store.GetByID(ctx, comment.ID)
	assertEqual(t, "Status", got.Status, "spam")
}

func TestCommentsStore_Delete(t *testing.T) {
	db := setupTestDB(t)
	store := NewCommentsStore(db)
	ctx := context.Background()

	comment := &comments.Comment{
		ID:        "comment-delete",
		PostID:    "post-001",
		Content:   "Delete me",
		Status:    "approved",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.Create(ctx, comment))

	err := store.Delete(ctx, comment.ID)
	assertNoError(t, err)

	got, _ := store.GetByID(ctx, comment.ID)
	if got != nil {
		t.Error("expected comment to be deleted")
	}
}

func TestCommentsStore_CountByPost(t *testing.T) {
	db := setupTestDB(t)
	store := NewCommentsStore(db)
	ctx := context.Background()

	// Create mix of approved and non-approved
	commentsData := []string{"approved", "approved", "pending", "spam", "approved"}
	for i, status := range commentsData {
		comment := &comments.Comment{
			ID:        "comment-count-" + string(rune('a'+i)),
			PostID:    "post-count",
			Content:   "Comment",
			Status:    status,
			CreatedAt: testTime,
			UpdatedAt: testTime,
		}
		assertNoError(t, store.Create(ctx, comment))
	}

	count, err := store.CountByPost(ctx, "post-count")
	assertNoError(t, err)
	assertEqual(t, "count", count, 3) // Only approved comments
}

func TestCommentsStore_CountByPost_NoComments(t *testing.T) {
	db := setupTestDB(t)
	store := NewCommentsStore(db)
	ctx := context.Background()

	count, err := store.CountByPost(ctx, "post-no-comments")
	assertNoError(t, err)
	assertEqual(t, "count", count, 0)
}
