package duckdb

import (
	"context"
	"database/sql"
	"testing"

	"github.com/go-mizu/blueprints/social/feature/posts"
)

func createTestPost(t *testing.T, store *PostsStore, accountID string) *posts.Post {
	t.Helper()
	ctx := context.Background()

	post := &posts.Post{
		ID:           newTestID(),
		AccountID:    accountID,
		Content:      "Test post content",
		Visibility:   posts.VisibilityPublic,
		Sensitive:    false,
		CreatedAt:    testTime(),
		LikesCount:   0,
		RepostsCount: 0,
		RepliesCount: 0,
		QuotesCount:  0,
	}

	if err := store.Insert(ctx, post); err != nil {
		t.Fatalf("createTestPost failed: %v", err)
	}
	return post
}

func TestPostsStore_Insert(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	store := NewPostsStore(db)

	post := &posts.Post{
		ID:             newTestID(),
		AccountID:      account.ID,
		Content:        "Hello, world!",
		ContentWarning: "Test warning",
		Visibility:     posts.VisibilityPublic,
		Language:       "en",
		Sensitive:      true,
		CreatedAt:      testTime(),
		LikesCount:     0,
		RepostsCount:   0,
		RepliesCount:   0,
		QuotesCount:    0,
	}

	if err := store.Insert(ctx, post); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Verify
	got, err := store.GetByID(ctx, post.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if got.Content != post.Content {
		t.Errorf("Content: got %q, want %q", got.Content, post.Content)
	}
	if got.ContentWarning != post.ContentWarning {
		t.Errorf("ContentWarning: got %q, want %q", got.ContentWarning, post.ContentWarning)
	}
	if got.Visibility != post.Visibility {
		t.Errorf("Visibility: got %q, want %q", got.Visibility, post.Visibility)
	}
	if got.Language != post.Language {
		t.Errorf("Language: got %q, want %q", got.Language, post.Language)
	}
	if !got.Sensitive {
		t.Error("Sensitive: got false, want true")
	}
	if got.AccountID != post.AccountID {
		t.Errorf("AccountID: got %q, want %q", got.AccountID, post.AccountID)
	}
}

func TestPostsStore_GetByID(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	store := NewPostsStore(db)
	post := createTestPost(t, store, account.ID)

	got, err := store.GetByID(ctx, post.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if got.ID != post.ID {
		t.Errorf("ID: got %q, want %q", got.ID, post.ID)
	}
}

func TestPostsStore_GetByID_NotFound(t *testing.T) {
	db := setupTestStore(t)
	store := NewPostsStore(db)
	ctx := context.Background()

	_, err := store.GetByID(ctx, "nonexistent")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestPostsStore_GetByIDs(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	store := NewPostsStore(db)

	post1 := createTestPost(t, store, account.ID)
	post2 := createTestPost(t, store, account.ID)
	post3 := createTestPost(t, store, account.ID)

	// Get all three
	got, err := store.GetByIDs(ctx, []string{post1.ID, post2.ID, post3.ID})
	if err != nil {
		t.Fatalf("GetByIDs failed: %v", err)
	}

	if len(got) != 3 {
		t.Errorf("GetByIDs count: got %d, want 3", len(got))
	}

	// Get empty slice
	empty, err := store.GetByIDs(ctx, []string{})
	if err != nil {
		t.Fatalf("GetByIDs (empty) failed: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("GetByIDs (empty) count: got %d, want 0", len(empty))
	}
}

func TestPostsStore_Update(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	store := NewPostsStore(db)
	post := createTestPost(t, store, account.ID)

	// Update
	update := &posts.UpdateIn{
		Content:        ptr("Updated content"),
		ContentWarning: ptr("Updated warning"),
		Sensitive:      ptr(true),
	}

	if err := store.Update(ctx, post.ID, update); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify
	got, err := store.GetByID(ctx, post.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if got.Content != "Updated content" {
		t.Errorf("Content: got %q, want %q", got.Content, "Updated content")
	}
	if got.ContentWarning != "Updated warning" {
		t.Errorf("ContentWarning: got %q, want %q", got.ContentWarning, "Updated warning")
	}
	if !got.Sensitive {
		t.Error("Sensitive: got false, want true")
	}
	if got.EditedAt == nil {
		t.Error("EditedAt: got nil, want non-nil")
	}
}

func TestPostsStore_Delete(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	store := NewPostsStore(db)
	post := createTestPost(t, store, account.ID)

	// Delete
	if err := store.Delete(ctx, post.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deleted
	_, err := store.GetByID(ctx, post.ID)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows after delete, got %v", err)
	}
}

func TestPostsStore_List(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	store := NewPostsStore(db)

	// Create posts
	for i := 0; i < 5; i++ {
		createTestPost(t, store, account.ID)
	}

	// List all
	list, err := store.List(ctx, posts.ListOpts{Limit: 10})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != 5 {
		t.Errorf("List count: got %d, want 5", len(list))
	}

	// List with limit
	list2, err := store.List(ctx, posts.ListOpts{Limit: 2})
	if err != nil {
		t.Fatalf("List (limit) failed: %v", err)
	}
	if len(list2) != 2 {
		t.Errorf("List (limit) count: got %d, want 2", len(list2))
	}
}

func TestPostsStore_List_ByAccountID(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	user1 := createTestAccount(t, db, "user1")
	user2 := createTestAccount(t, db, "user2")
	store := NewPostsStore(db)

	// Create posts for user1
	for i := 0; i < 3; i++ {
		createTestPost(t, store, user1.ID)
	}
	// Create posts for user2
	for i := 0; i < 2; i++ {
		createTestPost(t, store, user2.ID)
	}

	// List by user1
	list, err := store.List(ctx, posts.ListOpts{Limit: 10, AccountID: user1.ID})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != 3 {
		t.Errorf("List count: got %d, want 3", len(list))
	}
}

func TestPostsStore_List_ExcludeReplies(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	store := NewPostsStore(db)

	// Create regular post
	regularPost := createTestPost(t, store, account.ID)

	// Create reply
	reply := &posts.Post{
		ID:         newTestID(),
		AccountID:  account.ID,
		Content:    "This is a reply",
		ReplyToID:  regularPost.ID,
		Visibility: posts.VisibilityPublic,
		CreatedAt:  testTime(),
	}
	if err := store.Insert(ctx, reply); err != nil {
		t.Fatalf("Insert reply failed: %v", err)
	}

	// List excluding replies
	list, err := store.List(ctx, posts.ListOpts{Limit: 10, ExcludeReplies: true})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != 1 {
		t.Errorf("List count: got %d, want 1", len(list))
	}
}

func TestPostsStore_GetReplies(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	store := NewPostsStore(db)

	// Create parent post
	parent := createTestPost(t, store, account.ID)

	// Create replies
	for i := 0; i < 3; i++ {
		reply := &posts.Post{
			ID:         newTestID(),
			AccountID:  account.ID,
			Content:    "Reply " + string(rune('A'+i)),
			ReplyToID:  parent.ID,
			Visibility: posts.VisibilityPublic,
			CreatedAt:  testTime(),
		}
		if err := store.Insert(ctx, reply); err != nil {
			t.Fatalf("Insert reply failed: %v", err)
		}
	}

	// Get replies
	replies, err := store.GetReplies(ctx, parent.ID, 10, 0)
	if err != nil {
		t.Fatalf("GetReplies failed: %v", err)
	}

	if len(replies) != 3 {
		t.Errorf("GetReplies count: got %d, want 3", len(replies))
	}
}

func TestPostsStore_GetAncestors(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	store := NewPostsStore(db)

	// Create chain: root -> reply1 -> reply2
	root := createTestPost(t, store, account.ID)

	reply1 := &posts.Post{
		ID:         newTestID(),
		AccountID:  account.ID,
		Content:    "Reply 1",
		ReplyToID:  root.ID,
		Visibility: posts.VisibilityPublic,
		CreatedAt:  testTime(),
	}
	if err := store.Insert(ctx, reply1); err != nil {
		t.Fatalf("Insert reply1 failed: %v", err)
	}

	reply2 := &posts.Post{
		ID:         newTestID(),
		AccountID:  account.ID,
		Content:    "Reply 2",
		ReplyToID:  reply1.ID,
		Visibility: posts.VisibilityPublic,
		CreatedAt:  testTime(),
	}
	if err := store.Insert(ctx, reply2); err != nil {
		t.Fatalf("Insert reply2 failed: %v", err)
	}

	// Get ancestors of reply2
	ancestors, err := store.GetAncestors(ctx, reply2.ID)
	if err != nil {
		t.Fatalf("GetAncestors failed: %v", err)
	}

	if len(ancestors) != 2 {
		t.Errorf("GetAncestors count: got %d, want 2", len(ancestors))
	}

	// First ancestor should be root
	if ancestors[0].ID != root.ID {
		t.Errorf("First ancestor: got %q, want %q", ancestors[0].ID, root.ID)
	}
	// Second ancestor should be reply1
	if ancestors[1].ID != reply1.ID {
		t.Errorf("Second ancestor: got %q, want %q", ancestors[1].ID, reply1.ID)
	}
}

func TestPostsStore_GetDescendants(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	store := NewPostsStore(db)

	// Create parent
	parent := createTestPost(t, store, account.ID)

	// Create replies with thread_id
	for i := 0; i < 3; i++ {
		reply := &posts.Post{
			ID:         newTestID(),
			AccountID:  account.ID,
			Content:    "Reply " + string(rune('A'+i)),
			ReplyToID:  parent.ID,
			ThreadID:   parent.ID,
			Visibility: posts.VisibilityPublic,
			CreatedAt:  testTime(),
		}
		if err := store.Insert(ctx, reply); err != nil {
			t.Fatalf("Insert reply failed: %v", err)
		}
	}

	// Get descendants
	descendants, err := store.GetDescendants(ctx, parent.ID, 10)
	if err != nil {
		t.Fatalf("GetDescendants failed: %v", err)
	}

	if len(descendants) != 3 {
		t.Errorf("GetDescendants count: got %d, want 3", len(descendants))
	}
}

func TestPostsStore_IncrementRepliesCount(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	store := NewPostsStore(db)
	post := createTestPost(t, store, account.ID)

	// Increment
	if err := store.IncrementRepliesCount(ctx, post.ID); err != nil {
		t.Fatalf("IncrementRepliesCount failed: %v", err)
	}
	if err := store.IncrementRepliesCount(ctx, post.ID); err != nil {
		t.Fatalf("IncrementRepliesCount (2) failed: %v", err)
	}

	// Verify
	got, _ := store.GetByID(ctx, post.ID)
	if got.RepliesCount != 2 {
		t.Errorf("RepliesCount: got %d, want 2", got.RepliesCount)
	}
}

func TestPostsStore_DecrementRepliesCount(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	store := NewPostsStore(db)
	post := createTestPost(t, store, account.ID)

	// Increment first
	store.IncrementRepliesCount(ctx, post.ID)
	store.IncrementRepliesCount(ctx, post.ID)

	// Decrement
	if err := store.DecrementRepliesCount(ctx, post.ID); err != nil {
		t.Fatalf("DecrementRepliesCount failed: %v", err)
	}

	// Verify
	got, _ := store.GetByID(ctx, post.ID)
	if got.RepliesCount != 1 {
		t.Errorf("RepliesCount: got %d, want 1", got.RepliesCount)
	}

	// Should not go below 0
	store.DecrementRepliesCount(ctx, post.ID)
	store.DecrementRepliesCount(ctx, post.ID)

	got2, _ := store.GetByID(ctx, post.ID)
	if got2.RepliesCount < 0 {
		t.Errorf("RepliesCount should not go below 0: got %d", got2.RepliesCount)
	}
}

func TestPostsStore_IncrementQuotesCount(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	store := NewPostsStore(db)
	post := createTestPost(t, store, account.ID)

	// Increment
	if err := store.IncrementQuotesCount(ctx, post.ID); err != nil {
		t.Fatalf("IncrementQuotesCount failed: %v", err)
	}

	// Verify
	got, _ := store.GetByID(ctx, post.ID)
	if got.QuotesCount != 1 {
		t.Errorf("QuotesCount: got %d, want 1", got.QuotesCount)
	}
}

func TestPostsStore_Media_CRUD(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	postsStore := NewPostsStore(db)
	post := createTestPost(t, postsStore, account.ID)

	// Insert media
	media := &posts.Media{
		ID:         newTestID(),
		PostID:     post.ID,
		Type:       "image",
		URL:        "https://example.com/image.jpg",
		PreviewURL: "https://example.com/image_thumb.jpg",
		AltText:    "Test image",
		Width:      800,
		Height:     600,
		Position:   0,
	}

	if err := postsStore.InsertMedia(ctx, media); err != nil {
		t.Fatalf("InsertMedia failed: %v", err)
	}

	// Get media by post ID
	mediaList, err := postsStore.GetMediaByPostID(ctx, post.ID)
	if err != nil {
		t.Fatalf("GetMediaByPostID failed: %v", err)
	}

	if len(mediaList) != 1 {
		t.Errorf("Media count: got %d, want 1", len(mediaList))
	}

	m := mediaList[0]
	if m.Type != "image" {
		t.Errorf("Type: got %q, want %q", m.Type, "image")
	}
	if m.URL != media.URL {
		t.Errorf("URL: got %q, want %q", m.URL, media.URL)
	}
	if m.AltText != media.AltText {
		t.Errorf("AltText: got %q, want %q", m.AltText, media.AltText)
	}

	// Delete media
	if err := postsStore.DeleteMediaByPostID(ctx, post.ID); err != nil {
		t.Fatalf("DeleteMediaByPostID failed: %v", err)
	}

	// Verify deleted
	mediaList2, _ := postsStore.GetMediaByPostID(ctx, post.ID)
	if len(mediaList2) != 0 {
		t.Errorf("Media should be deleted, got %d", len(mediaList2))
	}
}

func TestPostsStore_Hashtags(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	store := NewPostsStore(db)
	post := createTestPost(t, store, account.ID)

	// Upsert hashtags
	hashtagID1, err := store.UpsertHashtag(ctx, "golang")
	if err != nil {
		t.Fatalf("UpsertHashtag failed: %v", err)
	}
	if hashtagID1 == "" {
		t.Error("UpsertHashtag should return ID")
	}

	hashtagID2, err := store.UpsertHashtag(ctx, "testing")
	if err != nil {
		t.Fatalf("UpsertHashtag (2) failed: %v", err)
	}

	// Link to post
	if err := store.LinkPostHashtag(ctx, post.ID, hashtagID1); err != nil {
		t.Fatalf("LinkPostHashtag failed: %v", err)
	}
	if err := store.LinkPostHashtag(ctx, post.ID, hashtagID2); err != nil {
		t.Fatalf("LinkPostHashtag (2) failed: %v", err)
	}

	// Get hashtags by post ID
	tags, err := store.GetHashtagsByPostID(ctx, post.ID)
	if err != nil {
		t.Fatalf("GetHashtagsByPostID failed: %v", err)
	}

	if len(tags) != 2 {
		t.Errorf("Hashtags count: got %d, want 2", len(tags))
	}

	// Upsert same hashtag should not create duplicate
	hashtagID1Again, err := store.UpsertHashtag(ctx, "golang")
	if err != nil {
		t.Fatalf("UpsertHashtag (again) failed: %v", err)
	}
	if hashtagID1Again != hashtagID1 {
		t.Errorf("UpsertHashtag should return same ID for existing hashtag")
	}
}

func TestPostsStore_Mentions(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	mentioned1 := createTestAccount(t, db, "mentioned1")
	mentioned2 := createTestAccount(t, db, "mentioned2")

	store := NewPostsStore(db)
	post := createTestPost(t, store, account.ID)

	// Insert mentions
	if err := store.InsertMention(ctx, post.ID, mentioned1.ID); err != nil {
		t.Fatalf("InsertMention failed: %v", err)
	}
	if err := store.InsertMention(ctx, post.ID, mentioned2.ID); err != nil {
		t.Fatalf("InsertMention (2) failed: %v", err)
	}

	// Get mentions
	mentions, err := store.GetMentionsByPostID(ctx, post.ID)
	if err != nil {
		t.Fatalf("GetMentionsByPostID failed: %v", err)
	}

	if len(mentions) != 2 {
		t.Errorf("Mentions count: got %d, want 2", len(mentions))
	}
}

func TestPostsStore_EditHistory(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	store := NewPostsStore(db)
	post := createTestPost(t, store, account.ID)

	// Insert edit history
	if err := store.InsertEditHistory(ctx, post.ID, "Original content", "", false); err != nil {
		t.Fatalf("InsertEditHistory failed: %v", err)
	}

	// Insert another version
	if err := store.InsertEditHistory(ctx, post.ID, "Second version", "Added CW", true); err != nil {
		t.Fatalf("InsertEditHistory (2) failed: %v", err)
	}

	// Verify by counting rows
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM edit_history WHERE post_id = $1", post.ID).Scan(&count)
	if err != nil {
		t.Fatalf("Query edit_history failed: %v", err)
	}
	if count != 2 {
		t.Errorf("EditHistory count: got %d, want 2", count)
	}
}

func TestPostsStore_Reply(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	store := NewPostsStore(db)

	// Create parent post
	parent := createTestPost(t, store, account.ID)

	// Create reply
	reply := &posts.Post{
		ID:         newTestID(),
		AccountID:  account.ID,
		Content:    "This is a reply",
		ReplyToID:  parent.ID,
		ThreadID:   parent.ID,
		Visibility: posts.VisibilityPublic,
		CreatedAt:  testTime(),
	}
	if err := store.Insert(ctx, reply); err != nil {
		t.Fatalf("Insert reply failed: %v", err)
	}

	// Verify
	got, err := store.GetByID(ctx, reply.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if got.ReplyToID != parent.ID {
		t.Errorf("ReplyToID: got %q, want %q", got.ReplyToID, parent.ID)
	}
	if got.ThreadID != parent.ID {
		t.Errorf("ThreadID: got %q, want %q", got.ThreadID, parent.ID)
	}
}

func TestPostsStore_QuotePost(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	store := NewPostsStore(db)

	// Create original post
	original := createTestPost(t, store, account.ID)

	// Create quote
	quote := &posts.Post{
		ID:         newTestID(),
		AccountID:  account.ID,
		Content:    "Quoting this post",
		QuoteOfID:  original.ID,
		Visibility: posts.VisibilityPublic,
		CreatedAt:  testTime(),
	}
	if err := store.Insert(ctx, quote); err != nil {
		t.Fatalf("Insert quote failed: %v", err)
	}

	// Verify
	got, err := store.GetByID(ctx, quote.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if got.QuoteOfID != original.ID {
		t.Errorf("QuoteOfID: got %q, want %q", got.QuoteOfID, original.ID)
	}
}
