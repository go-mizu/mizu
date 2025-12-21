package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/microblog/feature/accounts"
	"github.com/go-mizu/blueprints/microblog/feature/posts"
)

func TestPostsStore_Insert(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create account first
	acctStore := NewAccountsStore(db)
	acct := &accounts.Account{
		ID:        "acct-01",
		Username:  "testuser",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := acctStore.Insert(context.Background(), acct, "hash"); err != nil {
		t.Fatalf("failed to create account: %v", err)
	}

	store := NewPostsStore(db)
	ctx := context.Background()

	post := &posts.Post{
		ID:         "post-01",
		AccountID:  acct.ID,
		Content:    "Hello, world!",
		Visibility: "public",
		CreatedAt:  time.Now(),
	}

	if err := store.Insert(ctx, post); err != nil {
		t.Fatalf("Insert() error = %v", err)
	}

	// Verify insert
	got, err := store.GetByID(ctx, post.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if got.Content != post.Content {
		t.Errorf("Content = %s, want %s", got.Content, post.Content)
	}
	if got.AccountID != post.AccountID {
		t.Errorf("AccountID = %s, want %s", got.AccountID, post.AccountID)
	}
	if got.Visibility != post.Visibility {
		t.Errorf("Visibility = %s, want %s", got.Visibility, post.Visibility)
	}
}

func TestPostsStore_Insert_WithReply(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create account
	acctStore := NewAccountsStore(db)
	acct := &accounts.Account{
		ID:        "acct-01",
		Username:  "testuser",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := acctStore.Insert(context.Background(), acct, "hash"); err != nil {
		t.Fatalf("failed to create account: %v", err)
	}

	store := NewPostsStore(db)
	ctx := context.Background()

	// Create parent post
	parent := &posts.Post{
		ID:         "parent-01",
		AccountID:  acct.ID,
		Content:    "Original post",
		Visibility: "public",
		ThreadID:   "parent-01", // Thread ID is self for root posts
		CreatedAt:  time.Now(),
	}
	if err := store.Insert(ctx, parent); err != nil {
		t.Fatalf("Insert parent error = %v", err)
	}

	// Create reply
	reply := &posts.Post{
		ID:         "reply-01",
		AccountID:  acct.ID,
		Content:    "This is a reply",
		Visibility: "public",
		ReplyToID:  parent.ID,
		ThreadID:   parent.ID,
		CreatedAt:  time.Now(),
	}
	if err := store.Insert(ctx, reply); err != nil {
		t.Fatalf("Insert reply error = %v", err)
	}

	// Verify reply
	got, err := store.GetByID(ctx, reply.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if got.ReplyToID != parent.ID {
		t.Errorf("ReplyToID = %s, want %s", got.ReplyToID, parent.ID)
	}
	if got.ThreadID != parent.ID {
		t.Errorf("ThreadID = %s, want %s", got.ThreadID, parent.ID)
	}
}

func TestPostsStore_GetByID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create account
	acctStore := NewAccountsStore(db)
	acct := &accounts.Account{
		ID:        "acct-01",
		Username:  "testuser",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	acctStore.Insert(context.Background(), acct, "hash")

	store := NewPostsStore(db)
	ctx := context.Background()

	post := &posts.Post{
		ID:             "post-01",
		AccountID:      acct.ID,
		Content:        "Hello, world!",
		ContentWarning: "Contains greeting",
		Visibility:     "public",
		Language:       "en",
		Sensitive:      true,
		CreatedAt:      time.Now(),
	}
	store.Insert(ctx, post)

	got, err := store.GetByID(ctx, post.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if got.ID != post.ID {
		t.Errorf("ID = %s, want %s", got.ID, post.ID)
	}
	if got.ContentWarning != post.ContentWarning {
		t.Errorf("ContentWarning = %s, want %s", got.ContentWarning, post.ContentWarning)
	}
	if got.Language != post.Language {
		t.Errorf("Language = %s, want %s", got.Language, post.Language)
	}
	if got.Sensitive != post.Sensitive {
		t.Errorf("Sensitive = %v, want %v", got.Sensitive, post.Sensitive)
	}
}

func TestPostsStore_GetByID_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewPostsStore(db)
	ctx := context.Background()

	_, err := store.GetByID(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent post")
	}
}

func TestPostsStore_Update(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create account
	acctStore := NewAccountsStore(db)
	acct := &accounts.Account{
		ID:        "acct-01",
		Username:  "testuser",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	acctStore.Insert(context.Background(), acct, "hash")

	store := NewPostsStore(db)
	ctx := context.Background()

	post := &posts.Post{
		ID:         "post-01",
		AccountID:  acct.ID,
		Content:    "Original content",
		Visibility: "public",
		CreatedAt:  time.Now(),
	}
	store.Insert(ctx, post)

	// Update content
	newContent := "Updated content"
	newCW := "Content warning"
	newSensitive := true
	err := store.Update(ctx, post.ID, &posts.UpdateIn{
		Content:        &newContent,
		ContentWarning: &newCW,
		Sensitive:      &newSensitive,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Verify update
	got, _ := store.GetByID(ctx, post.ID)
	if got.Content != newContent {
		t.Errorf("Content = %s, want %s", got.Content, newContent)
	}
	if got.ContentWarning != newCW {
		t.Errorf("ContentWarning = %s, want %s", got.ContentWarning, newCW)
	}
	if got.Sensitive != newSensitive {
		t.Errorf("Sensitive = %v, want %v", got.Sensitive, newSensitive)
	}
	if got.EditedAt == nil {
		t.Error("EditedAt should be set after update")
	}
}

func TestPostsStore_Delete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create account
	acctStore := NewAccountsStore(db)
	acct := &accounts.Account{
		ID:        "acct-01",
		Username:  "testuser",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	acctStore.Insert(context.Background(), acct, "hash")

	store := NewPostsStore(db)
	ctx := context.Background()

	post := &posts.Post{
		ID:         "post-01",
		AccountID:  acct.ID,
		Content:    "To be deleted",
		Visibility: "public",
		CreatedAt:  time.Now(),
	}
	store.Insert(ctx, post)

	// Delete
	if err := store.Delete(ctx, post.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify deleted
	_, err := store.GetByID(ctx, post.ID)
	if err == nil {
		t.Error("expected error for deleted post")
	}
}

func TestPostsStore_GetOwner(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create account
	acctStore := NewAccountsStore(db)
	acct := &accounts.Account{
		ID:        "acct-01",
		Username:  "testuser",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	acctStore.Insert(context.Background(), acct, "hash")

	store := NewPostsStore(db)
	ctx := context.Background()

	// Create parent and reply
	parent := &posts.Post{
		ID:         "parent-01",
		AccountID:  acct.ID,
		Content:    "Parent post",
		Visibility: "public",
		CreatedAt:  time.Now(),
	}
	store.Insert(ctx, parent)

	reply := &posts.Post{
		ID:         "reply-01",
		AccountID:  acct.ID,
		Content:    "Reply post",
		Visibility: "public",
		ReplyToID:  parent.ID,
		CreatedAt:  time.Now(),
	}
	store.Insert(ctx, reply)

	// Check owner of reply
	accountID, replyToID, err := store.GetOwner(ctx, reply.ID)
	if err != nil {
		t.Fatalf("GetOwner() error = %v", err)
	}
	if accountID != acct.ID {
		t.Errorf("AccountID = %s, want %s", accountID, acct.ID)
	}
	if replyToID != parent.ID {
		t.Errorf("ReplyToID = %s, want %s", replyToID, parent.ID)
	}
}

func TestPostsStore_GetThreadID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create account
	acctStore := NewAccountsStore(db)
	acct := &accounts.Account{
		ID:        "acct-01",
		Username:  "testuser",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	acctStore.Insert(context.Background(), acct, "hash")

	store := NewPostsStore(db)
	ctx := context.Background()

	post := &posts.Post{
		ID:         "post-01",
		AccountID:  acct.ID,
		Content:    "Post with thread",
		Visibility: "public",
		ThreadID:   "thread-id",
		CreatedAt:  time.Now(),
	}
	store.Insert(ctx, post)

	threadID, err := store.GetThreadID(ctx, post.ID)
	if err != nil {
		t.Fatalf("GetThreadID() error = %v", err)
	}
	if threadID != "thread-id" {
		t.Errorf("ThreadID = %s, want thread-id", threadID)
	}
}

func TestPostsStore_GetDescendants(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create account
	acctStore := NewAccountsStore(db)
	acct := &accounts.Account{
		ID:        "acct-01",
		Username:  "testuser",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	acctStore.Insert(context.Background(), acct, "hash")

	store := NewPostsStore(db)
	ctx := context.Background()

	// Create post chain: parent -> reply1 -> reply2
	parent := &posts.Post{
		ID:         "parent",
		AccountID:  acct.ID,
		Content:    "Parent",
		Visibility: "public",
		ThreadID:   "parent",
		CreatedAt:  time.Now(),
	}
	store.Insert(ctx, parent)

	reply1 := &posts.Post{
		ID:         "reply1",
		AccountID:  acct.ID,
		Content:    "Reply 1",
		Visibility: "public",
		ReplyToID:  "parent",
		ThreadID:   "parent",
		CreatedAt:  time.Now().Add(1 * time.Second),
	}
	store.Insert(ctx, reply1)

	reply2 := &posts.Post{
		ID:         "reply2",
		AccountID:  acct.ID,
		Content:    "Reply 2",
		Visibility: "public",
		ReplyToID:  "reply1",
		ThreadID:   "parent",
		CreatedAt:  time.Now().Add(2 * time.Second),
	}
	store.Insert(ctx, reply2)

	// Get descendants of parent
	descendants, err := store.GetDescendants(ctx, "parent", 10)
	if err != nil {
		t.Fatalf("GetDescendants() error = %v", err)
	}

	if len(descendants) != 2 {
		t.Errorf("len(descendants) = %d, want 2", len(descendants))
	}
}

func TestPostsStore_IncrementDecrementReplies(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create account
	acctStore := NewAccountsStore(db)
	acct := &accounts.Account{
		ID:        "acct-01",
		Username:  "testuser",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	acctStore.Insert(context.Background(), acct, "hash")

	store := NewPostsStore(db)
	ctx := context.Background()

	post := &posts.Post{
		ID:         "post-01",
		AccountID:  acct.ID,
		Content:    "Test",
		Visibility: "public",
		CreatedAt:  time.Now(),
	}
	store.Insert(ctx, post)

	// Initial count should be 0
	got, _ := store.GetByID(ctx, post.ID)
	if got.RepliesCount != 0 {
		t.Errorf("initial RepliesCount = %d, want 0", got.RepliesCount)
	}

	// Increment
	store.IncrementReplies(ctx, post.ID)
	got, _ = store.GetByID(ctx, post.ID)
	if got.RepliesCount != 1 {
		t.Errorf("RepliesCount after increment = %d, want 1", got.RepliesCount)
	}

	// Increment again
	store.IncrementReplies(ctx, post.ID)
	got, _ = store.GetByID(ctx, post.ID)
	if got.RepliesCount != 2 {
		t.Errorf("RepliesCount after second increment = %d, want 2", got.RepliesCount)
	}

	// Decrement
	store.DecrementReplies(ctx, post.ID)
	got, _ = store.GetByID(ctx, post.ID)
	if got.RepliesCount != 1 {
		t.Errorf("RepliesCount after decrement = %d, want 1", got.RepliesCount)
	}
}

func TestPostsStore_CheckInteractions(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create accounts
	acctStore := NewAccountsStore(db)
	acct := &accounts.Account{
		ID:        "acct-01",
		Username:  "testuser",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	acctStore.Insert(context.Background(), acct, "hash")

	store := NewPostsStore(db)
	ctx := context.Background()

	post := &posts.Post{
		ID:         "post-01",
		AccountID:  acct.ID,
		Content:    "Test",
		Visibility: "public",
		CreatedAt:  time.Now(),
	}
	store.Insert(ctx, post)

	// Initially not liked, reposted, or bookmarked
	liked, _ := store.CheckLiked(ctx, acct.ID, post.ID)
	if liked {
		t.Error("CheckLiked should be false initially")
	}

	reposted, _ := store.CheckReposted(ctx, acct.ID, post.ID)
	if reposted {
		t.Error("CheckReposted should be false initially")
	}

	bookmarked, _ := store.CheckBookmarked(ctx, acct.ID, post.ID)
	if bookmarked {
		t.Error("CheckBookmarked should be false initially")
	}
}

func TestPostsStore_SaveHashtag(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create account
	acctStore := NewAccountsStore(db)
	acct := &accounts.Account{
		ID:        "acct-01",
		Username:  "testuser",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	acctStore.Insert(context.Background(), acct, "hash")

	store := NewPostsStore(db)
	ctx := context.Background()

	post := &posts.Post{
		ID:         "post-01",
		AccountID:  acct.ID,
		Content:    "#golang",
		Visibility: "public",
		CreatedAt:  time.Now(),
	}
	store.Insert(ctx, post)

	// Save hashtag
	err := store.SaveHashtag(ctx, post.ID, "golang")
	if err != nil {
		t.Fatalf("SaveHashtag() error = %v", err)
	}

	// Save same hashtag again (should update count)
	post2 := &posts.Post{
		ID:         "post-02",
		AccountID:  acct.ID,
		Content:    "#golang",
		Visibility: "public",
		CreatedAt:  time.Now(),
	}
	store.Insert(ctx, post2)
	err = store.SaveHashtag(ctx, post2.ID, "golang")
	if err != nil {
		t.Fatalf("SaveHashtag() second call error = %v", err)
	}
}

func TestPostsStore_SaveMention(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create accounts
	acctStore := NewAccountsStore(db)
	author := &accounts.Account{
		ID:        "acct-author",
		Username:  "author",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	acctStore.Insert(context.Background(), author, "hash")

	mentioned := &accounts.Account{
		ID:        "acct-mentioned",
		Username:  "mentioned",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	acctStore.Insert(context.Background(), mentioned, "hash")

	store := NewPostsStore(db)
	ctx := context.Background()

	post := &posts.Post{
		ID:         "post-01",
		AccountID:  author.ID,
		Content:    "Hello @mentioned",
		Visibility: "public",
		CreatedAt:  time.Now(),
	}
	store.Insert(ctx, post)

	// Save mention
	err := store.SaveMention(ctx, post.ID, "mentioned")
	if err != nil {
		t.Fatalf("SaveMention() error = %v", err)
	}

	// Save mention for nonexistent user should not error
	err = store.SaveMention(ctx, post.ID, "nobody")
	if err != nil {
		t.Fatalf("SaveMention() for nonexistent user error = %v", err)
	}
}

func TestPostsStore_SaveEditHistory(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create account
	acctStore := NewAccountsStore(db)
	acct := &accounts.Account{
		ID:        "acct-01",
		Username:  "testuser",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	acctStore.Insert(context.Background(), acct, "hash")

	store := NewPostsStore(db)
	ctx := context.Background()

	post := &posts.Post{
		ID:         "post-01",
		AccountID:  acct.ID,
		Content:    "Original content",
		Visibility: "public",
		CreatedAt:  time.Now(),
	}
	store.Insert(ctx, post)

	// Save edit history
	err := store.SaveEditHistory(ctx, post.ID, "Original content", "", false)
	if err != nil {
		t.Fatalf("SaveEditHistory() error = %v", err)
	}
}
