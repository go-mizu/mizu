package duckdb

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

// ============================================================
// Comment CRUD Tests
// ============================================================

func TestCreateComment_Success(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	file := newTestFile("file1", "user1", "document.pdf")
	store.CreateFile(ctx, file)

	comment := newTestComment("comment1", "file1", "user1", "This is a great document!")

	if err := store.CreateComment(ctx, comment); err != nil {
		t.Fatalf("create comment failed: %v", err)
	}

	got, err := store.GetCommentByID(ctx, "comment1")
	if err != nil {
		t.Fatalf("get comment failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected comment, got nil")
	}
	if got.Content != "This is a great document!" {
		t.Errorf("expected content, got %s", got.Content)
	}
	if got.FileID != "file1" {
		t.Errorf("expected file_id file1, got %s", got.FileID)
	}
	if got.UserID != "user1" {
		t.Errorf("expected user_id user1, got %s", got.UserID)
	}
}

func TestCreateComment_Reply(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	file := newTestFile("file1", "user1", "document.pdf")
	store.CreateFile(ctx, file)

	// Create parent comment
	parent := newTestComment("parent", "file1", "user1", "What do you think?")
	store.CreateComment(ctx, parent)

	// Create reply
	reply := newTestComment("reply", "file1", "user1", "Looks good to me!")
	reply.ParentID = sql.NullString{String: "parent", Valid: true}

	if err := store.CreateComment(ctx, reply); err != nil {
		t.Fatalf("create reply failed: %v", err)
	}

	got, _ := store.GetCommentByID(ctx, "reply")
	if !got.ParentID.Valid || got.ParentID.String != "parent" {
		t.Errorf("expected parent_id parent, got %v", got.ParentID)
	}
}

func TestGetCommentByID_Exists(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	file := newTestFile("file1", "user1", "document.pdf")
	store.CreateFile(ctx, file)

	comment := newTestComment("comment1", "file1", "user1", "Test comment")
	store.CreateComment(ctx, comment)

	got, err := store.GetCommentByID(ctx, "comment1")
	if err != nil {
		t.Fatalf("get comment failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected comment, got nil")
	}
	if got.ID != "comment1" {
		t.Errorf("expected ID comment1, got %s", got.ID)
	}
}

func TestGetCommentByID_NotFound(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	got, err := store.GetCommentByID(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("get comment failed: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestUpdateComment_Success(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	file := newTestFile("file1", "user1", "document.pdf")
	store.CreateFile(ctx, file)

	comment := newTestComment("comment1", "file1", "user1", "Original content")
	store.CreateComment(ctx, comment)

	comment.Content = "Updated content"
	comment.UpdatedAt = time.Now().Truncate(time.Microsecond)

	if err := store.UpdateComment(ctx, comment); err != nil {
		t.Fatalf("update comment failed: %v", err)
	}

	got, _ := store.GetCommentByID(ctx, "comment1")
	if got.Content != "Updated content" {
		t.Errorf("expected content Updated content, got %s", got.Content)
	}
}

func TestDeleteComment_Success(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	file := newTestFile("file1", "user1", "document.pdf")
	store.CreateFile(ctx, file)

	comment := newTestComment("comment1", "file1", "user1", "Test comment")
	store.CreateComment(ctx, comment)

	if err := store.DeleteComment(ctx, "comment1"); err != nil {
		t.Fatalf("delete comment failed: %v", err)
	}

	got, _ := store.GetCommentByID(ctx, "comment1")
	if got != nil {
		t.Errorf("expected nil after delete, got %+v", got)
	}
}

// ============================================================
// Comment Listing Tests
// ============================================================

func TestListCommentsByFile(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	file := newTestFile("file1", "user1", "document.pdf")
	store.CreateFile(ctx, file)

	// Create comments with different timestamps
	for i := 1; i <= 3; i++ {
		comment := newTestComment("comment"+string(rune('0'+i)), "file1", "user1", "Comment "+string(rune('0'+i)))
		comment.CreatedAt = time.Now().Add(time.Duration(i) * time.Minute).Truncate(time.Microsecond)
		store.CreateComment(ctx, comment)
	}

	comments, err := store.ListCommentsByFile(ctx, "file1")
	if err != nil {
		t.Fatalf("list comments failed: %v", err)
	}
	if len(comments) != 3 {
		t.Errorf("expected 3 comments, got %d", len(comments))
	}

	// Should be ordered by created_at ASC (oldest first)
	if comments[0].Content != "Comment 1" {
		t.Errorf("expected first comment to be oldest, got %s", comments[0].Content)
	}
}

func TestListTopLevelCommentsByFile(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	file := newTestFile("file1", "user1", "document.pdf")
	store.CreateFile(ctx, file)

	// Create top-level comments
	parent1 := newTestComment("parent1", "file1", "user1", "Parent 1")
	parent2 := newTestComment("parent2", "file1", "user1", "Parent 2")
	store.CreateComment(ctx, parent1)
	store.CreateComment(ctx, parent2)

	// Create replies
	reply1 := newTestComment("reply1", "file1", "user1", "Reply to Parent 1")
	reply1.ParentID = sql.NullString{String: "parent1", Valid: true}
	reply2 := newTestComment("reply2", "file1", "user1", "Another reply")
	reply2.ParentID = sql.NullString{String: "parent1", Valid: true}
	store.CreateComment(ctx, reply1)
	store.CreateComment(ctx, reply2)

	comments, err := store.ListTopLevelCommentsByFile(ctx, "file1")
	if err != nil {
		t.Fatalf("list top-level comments failed: %v", err)
	}
	if len(comments) != 2 {
		t.Errorf("expected 2 top-level comments, got %d", len(comments))
	}

	// All comments should have no parent
	for _, c := range comments {
		if c.ParentID.Valid {
			t.Errorf("expected top-level comment, got parent_id=%s", c.ParentID.String)
		}
	}
}

func TestListReplies(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	file := newTestFile("file1", "user1", "document.pdf")
	store.CreateFile(ctx, file)

	// Create parent comment
	parent := newTestComment("parent", "file1", "user1", "What do you think?")
	store.CreateComment(ctx, parent)

	// Create multiple replies
	for i := 1; i <= 3; i++ {
		reply := newTestComment("reply"+string(rune('0'+i)), "file1", "user1", "Reply "+string(rune('0'+i)))
		reply.ParentID = sql.NullString{String: "parent", Valid: true}
		reply.CreatedAt = time.Now().Add(time.Duration(i) * time.Minute).Truncate(time.Microsecond)
		store.CreateComment(ctx, reply)
	}

	replies, err := store.ListReplies(ctx, "parent")
	if err != nil {
		t.Fatalf("list replies failed: %v", err)
	}
	if len(replies) != 3 {
		t.Errorf("expected 3 replies, got %d", len(replies))
	}

	// Should be ordered by created_at ASC
	if replies[0].Content != "Reply 1" {
		t.Errorf("expected first reply to be oldest, got %s", replies[0].Content)
	}
}

// ============================================================
// Comment Features Tests
// ============================================================

func TestResolveComment(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	file := newTestFile("file1", "user1", "document.pdf")
	store.CreateFile(ctx, file)

	comment := newTestComment("comment1", "file1", "user1", "Please fix this issue")
	store.CreateComment(ctx, comment)

	// Initially not resolved
	got, _ := store.GetCommentByID(ctx, "comment1")
	if got.IsResolved {
		t.Error("comment should not be resolved initially")
	}

	if err := store.ResolveComment(ctx, "comment1"); err != nil {
		t.Fatalf("resolve comment failed: %v", err)
	}

	got, _ = store.GetCommentByID(ctx, "comment1")
	if !got.IsResolved {
		t.Error("expected comment to be resolved")
	}
}

func TestUnresolveComment(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	file := newTestFile("file1", "user1", "document.pdf")
	store.CreateFile(ctx, file)

	comment := newTestComment("comment1", "file1", "user1", "Issue that was fixed")
	comment.IsResolved = true
	store.CreateComment(ctx, comment)

	if err := store.UnresolveComment(ctx, "comment1"); err != nil {
		t.Fatalf("unresolve comment failed: %v", err)
	}

	got, _ := store.GetCommentByID(ctx, "comment1")
	if got.IsResolved {
		t.Error("expected comment to be unresolved")
	}
}

func TestDeleteCommentsByFile(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	file1 := newTestFile("file1", "user1", "doc1.pdf")
	file2 := newTestFile("file2", "user1", "doc2.pdf")
	store.CreateFile(ctx, file1)
	store.CreateFile(ctx, file2)

	// Create comments for both files
	for i := 1; i <= 3; i++ {
		c1 := newTestComment("c1_"+string(rune('0'+i)), "file1", "user1", "Comment")
		c2 := newTestComment("c2_"+string(rune('0'+i)), "file2", "user1", "Comment")
		store.CreateComment(ctx, c1)
		store.CreateComment(ctx, c2)
	}

	if err := store.DeleteCommentsByFile(ctx, "file1"); err != nil {
		t.Fatalf("delete comments by file failed: %v", err)
	}

	// file1 comments should be deleted
	file1Comments, _ := store.ListCommentsByFile(ctx, "file1")
	if len(file1Comments) != 0 {
		t.Errorf("expected 0 comments for file1, got %d", len(file1Comments))
	}

	// file2 comments should remain
	file2Comments, _ := store.ListCommentsByFile(ctx, "file2")
	if len(file2Comments) != 3 {
		t.Errorf("expected 3 comments for file2, got %d", len(file2Comments))
	}
}

func TestCountCommentsByFile(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	file := newTestFile("file1", "user1", "document.pdf")
	store.CreateFile(ctx, file)

	// Initially no comments
	count, err := store.CountCommentsByFile(ctx, "file1")
	if err != nil {
		t.Fatalf("count comments failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 comments, got %d", count)
	}

	// Add comments
	for i := 1; i <= 5; i++ {
		comment := newTestComment("comment"+string(rune('0'+i)), "file1", "user1", "Comment")
		store.CreateComment(ctx, comment)
	}

	count, _ = store.CountCommentsByFile(ctx, "file1")
	if count != 5 {
		t.Errorf("expected 5 comments, got %d", count)
	}
}

// ============================================================
// Business Use Cases - Collaboration
// ============================================================

func TestCollaboration_AddFeedback(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	alice := newTestUser("alice", "alice@example.com")
	bob := newTestUser("bob", "bob@example.com")
	store.CreateUser(ctx, alice)
	store.CreateUser(ctx, bob)

	// Alice creates a document
	file := newTestFile("file1", "alice", "proposal.docx")
	store.CreateFile(ctx, file)

	// Bob reviews and adds feedback
	feedback := newTestComment("feedback1", "file1", "bob", "I think section 3 needs more detail on the implementation timeline.")
	if err := store.CreateComment(ctx, feedback); err != nil {
		t.Fatalf("add feedback failed: %v", err)
	}

	// Verify feedback is visible
	comments, _ := store.ListCommentsByFile(ctx, "file1")
	if len(comments) != 1 {
		t.Fatal("expected 1 feedback comment")
	}
	if comments[0].UserID != "bob" {
		t.Errorf("expected comment from bob, got %s", comments[0].UserID)
	}
}

func TestCollaboration_ThreadedDiscussion(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	alice := newTestUser("alice", "alice@example.com")
	bob := newTestUser("bob", "bob@example.com")
	charlie := newTestUser("charlie", "charlie@example.com")
	store.CreateUser(ctx, alice)
	store.CreateUser(ctx, bob)
	store.CreateUser(ctx, charlie)

	file := newTestFile("file1", "alice", "design.pdf")
	store.CreateFile(ctx, file)

	// Bob starts a discussion
	thread := newTestComment("thread", "file1", "bob", "What font should we use for headings?")
	store.CreateComment(ctx, thread)

	// Multiple people reply
	replies := []struct {
		id      string
		userID  string
		content string
	}{
		{"reply1", "alice", "I think Helvetica looks clean"},
		{"reply2", "charlie", "What about using a serif font for contrast?"},
		{"reply3", "bob", "Good point Charlie. Let's try Georgia"},
		{"reply4", "alice", "Georgia works well. Let's go with that!"},
	}

	for i, r := range replies {
		reply := newTestComment(r.id, "file1", r.userID, r.content)
		reply.ParentID = sql.NullString{String: "thread", Valid: true}
		reply.CreatedAt = time.Now().Add(time.Duration(i) * time.Minute).Truncate(time.Microsecond)
		store.CreateComment(ctx, reply)
	}

	// Verify thread structure
	topLevel, _ := store.ListTopLevelCommentsByFile(ctx, "file1")
	if len(topLevel) != 1 {
		t.Fatal("expected 1 top-level comment")
	}

	threadReplies, _ := store.ListReplies(ctx, "thread")
	if len(threadReplies) != 4 {
		t.Errorf("expected 4 replies, got %d", len(threadReplies))
	}

	// Verify order (chronological)
	if threadReplies[0].Content != "I think Helvetica looks clean" {
		t.Errorf("expected first reply to be Alice's, got %s", threadReplies[0].Content)
	}
	if threadReplies[3].Content != "Georgia works well. Let's go with that!" {
		t.Errorf("expected last reply to be Alice's approval, got %s", threadReplies[3].Content)
	}
}

func TestCollaboration_ResolveIssue(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	alice := newTestUser("alice", "alice@example.com")
	bob := newTestUser("bob", "bob@example.com")
	store.CreateUser(ctx, alice)
	store.CreateUser(ctx, bob)

	file := newTestFile("file1", "alice", "code_review.go")
	store.CreateFile(ctx, file)

	// Bob raises an issue
	issue := newTestComment("issue1", "file1", "bob", "This function has a memory leak")
	store.CreateComment(ctx, issue)

	// Alice acknowledges
	ack := newTestComment("ack1", "file1", "alice", "Good catch! I'll fix it.")
	ack.ParentID = sql.NullString{String: "issue1", Valid: true}
	store.CreateComment(ctx, ack)

	// Alice fixes and marks resolved
	fixed := newTestComment("fixed1", "file1", "alice", "Fixed in latest commit")
	fixed.ParentID = sql.NullString{String: "issue1", Valid: true}
	store.CreateComment(ctx, fixed)
	store.ResolveComment(ctx, "issue1")

	// Verify issue is resolved
	got, _ := store.GetCommentByID(ctx, "issue1")
	if !got.IsResolved {
		t.Error("expected issue to be resolved")
	}

	// Count should include all comments
	count, _ := store.CountCommentsByFile(ctx, "file1")
	if count != 3 {
		t.Errorf("expected 3 comments total, got %d", count)
	}
}

func TestCollaboration_FileCleanup(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	file := newTestFile("file1", "user1", "old_document.pdf")
	store.CreateFile(ctx, file)

	// Add multiple comments
	for i := 1; i <= 5; i++ {
		comment := newTestComment("comment"+string(rune('0'+i)), "file1", "user1", "Comment "+string(rune('0'+i)))
		store.CreateComment(ctx, comment)
	}

	// Verify comments exist
	count, _ := store.CountCommentsByFile(ctx, "file1")
	if count != 5 {
		t.Fatalf("expected 5 comments before cleanup, got %d", count)
	}

	// Delete file - comments should be cleaned up separately
	store.DeleteCommentsByFile(ctx, "file1")
	store.DeleteFile(ctx, "file1")

	// Verify no orphan comments
	comments, _ := store.ListCommentsByFile(ctx, "file1")
	if len(comments) != 0 {
		t.Errorf("expected 0 comments after file deletion, got %d", len(comments))
	}
}

func TestCollaboration_MultipleFiles(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	// Create multiple files with comments
	for i := 1; i <= 3; i++ {
		file := newTestFile("file"+string(rune('0'+i)), "user1", "doc"+string(rune('0'+i))+".pdf")
		store.CreateFile(ctx, file)

		for j := 1; j <= i+1; j++ {
			comment := newTestComment("c"+string(rune('0'+i))+string(rune('0'+j)), "file"+string(rune('0'+i)), "user1", "Comment")
			store.CreateComment(ctx, comment)
		}
	}

	// Verify comment counts per file
	count1, _ := store.CountCommentsByFile(ctx, "file1")
	count2, _ := store.CountCommentsByFile(ctx, "file2")
	count3, _ := store.CountCommentsByFile(ctx, "file3")

	if count1 != 2 {
		t.Errorf("expected 2 comments for file1, got %d", count1)
	}
	if count2 != 3 {
		t.Errorf("expected 3 comments for file2, got %d", count2)
	}
	if count3 != 4 {
		t.Errorf("expected 4 comments for file3, got %d", count3)
	}
}

func TestCollaboration_EditComment(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	file := newTestFile("file1", "user1", "document.pdf")
	store.CreateFile(ctx, file)

	comment := newTestComment("comment1", "file1", "user1", "Original tyop here")
	store.CreateComment(ctx, comment)

	// Fix typo
	comment.Content = "Original typo here - fixed!"
	comment.UpdatedAt = time.Now().Truncate(time.Microsecond)
	store.UpdateComment(ctx, comment)

	got, _ := store.GetCommentByID(ctx, "comment1")
	if got.Content != "Original typo here - fixed!" {
		t.Errorf("expected edited content, got %s", got.Content)
	}
}
