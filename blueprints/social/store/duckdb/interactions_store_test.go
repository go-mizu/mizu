package duckdb

import (
	"context"
	"testing"

	"github.com/go-mizu/blueprints/social/feature/interactions"
)

func TestInteractionsStore_Like(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	postsStore := NewPostsStore(db)
	post := createTestPost(t, postsStore, account.ID)

	store := NewInteractionsStore(db)

	// Insert like
	like := &interactions.Like{
		ID:        newTestID(),
		AccountID: account.ID,
		PostID:    post.ID,
		CreatedAt: testTime(),
	}

	if err := store.InsertLike(ctx, like); err != nil {
		t.Fatalf("InsertLike failed: %v", err)
	}

	// Check exists
	exists, err := store.ExistsLike(ctx, account.ID, post.ID)
	if err != nil {
		t.Fatalf("ExistsLike failed: %v", err)
	}
	if !exists {
		t.Error("ExistsLike: got false, want true")
	}
}

func TestInteractionsStore_Unlike(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	postsStore := NewPostsStore(db)
	post := createTestPost(t, postsStore, account.ID)

	store := NewInteractionsStore(db)

	// Insert like
	like := &interactions.Like{
		ID:        newTestID(),
		AccountID: account.ID,
		PostID:    post.ID,
		CreatedAt: testTime(),
	}
	store.InsertLike(ctx, like)

	// Delete like
	if err := store.DeleteLike(ctx, account.ID, post.ID); err != nil {
		t.Fatalf("DeleteLike failed: %v", err)
	}

	// Check not exists
	exists, _ := store.ExistsLike(ctx, account.ID, post.ID)
	if exists {
		t.Error("ExistsLike after delete: got true, want false")
	}
}

func TestInteractionsStore_GetLikedBy(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	author := createTestAccount(t, db, "author")
	liker1 := createTestAccount(t, db, "liker1")
	liker2 := createTestAccount(t, db, "liker2")

	postsStore := NewPostsStore(db)
	post := createTestPost(t, postsStore, author.ID)

	store := NewInteractionsStore(db)

	// Add likes
	store.InsertLike(ctx, &interactions.Like{
		ID:        newTestID(),
		AccountID: liker1.ID,
		PostID:    post.ID,
		CreatedAt: testTime(),
	})
	store.InsertLike(ctx, &interactions.Like{
		ID:        newTestID(),
		AccountID: liker2.ID,
		PostID:    post.ID,
		CreatedAt: testTime(),
	})

	// Get liked by
	ids, err := store.GetLikedBy(ctx, post.ID, 10, 0)
	if err != nil {
		t.Fatalf("GetLikedBy failed: %v", err)
	}

	if len(ids) != 2 {
		t.Errorf("GetLikedBy count: got %d, want 2", len(ids))
	}
}

func TestInteractionsStore_GetLikedPosts(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	postsStore := NewPostsStore(db)
	post1 := createTestPost(t, postsStore, account.ID)
	post2 := createTestPost(t, postsStore, account.ID)
	post3 := createTestPost(t, postsStore, account.ID)

	store := NewInteractionsStore(db)

	// Like posts
	for _, post := range []*testPostHelper{{ID: post1.ID}, {ID: post2.ID}, {ID: post3.ID}} {
		store.InsertLike(ctx, &interactions.Like{
			ID:        newTestID(),
			AccountID: account.ID,
			PostID:    post.ID,
			CreatedAt: testTime(),
		})
	}

	// Get liked posts
	ids, err := store.GetLikedPosts(ctx, account.ID, 10, 0)
	if err != nil {
		t.Fatalf("GetLikedPosts failed: %v", err)
	}

	if len(ids) != 3 {
		t.Errorf("GetLikedPosts count: got %d, want 3", len(ids))
	}
}

type testPostHelper struct {
	ID string
}

func TestInteractionsStore_LikesCount(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	postsStore := NewPostsStore(db)
	post := createTestPost(t, postsStore, account.ID)

	store := NewInteractionsStore(db)

	// Increment
	if err := store.IncrementLikesCount(ctx, post.ID); err != nil {
		t.Fatalf("IncrementLikesCount failed: %v", err)
	}
	if err := store.IncrementLikesCount(ctx, post.ID); err != nil {
		t.Fatalf("IncrementLikesCount (2) failed: %v", err)
	}

	// Verify
	got, _ := postsStore.GetByID(ctx, post.ID)
	if got.LikesCount != 2 {
		t.Errorf("LikesCount: got %d, want 2", got.LikesCount)
	}

	// Decrement
	if err := store.DecrementLikesCount(ctx, post.ID); err != nil {
		t.Fatalf("DecrementLikesCount failed: %v", err)
	}

	got2, _ := postsStore.GetByID(ctx, post.ID)
	if got2.LikesCount != 1 {
		t.Errorf("LikesCount after decrement: got %d, want 1", got2.LikesCount)
	}
}

func TestInteractionsStore_Repost(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	postsStore := NewPostsStore(db)
	post := createTestPost(t, postsStore, account.ID)

	store := NewInteractionsStore(db)

	// Insert repost
	repost := &interactions.Repost{
		ID:        newTestID(),
		AccountID: account.ID,
		PostID:    post.ID,
		CreatedAt: testTime(),
	}

	if err := store.InsertRepost(ctx, repost); err != nil {
		t.Fatalf("InsertRepost failed: %v", err)
	}

	// Check exists
	exists, err := store.ExistsRepost(ctx, account.ID, post.ID)
	if err != nil {
		t.Fatalf("ExistsRepost failed: %v", err)
	}
	if !exists {
		t.Error("ExistsRepost: got false, want true")
	}
}

func TestInteractionsStore_Unrepost(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	postsStore := NewPostsStore(db)
	post := createTestPost(t, postsStore, account.ID)

	store := NewInteractionsStore(db)

	// Insert repost
	repost := &interactions.Repost{
		ID:        newTestID(),
		AccountID: account.ID,
		PostID:    post.ID,
		CreatedAt: testTime(),
	}
	store.InsertRepost(ctx, repost)

	// Delete repost
	if err := store.DeleteRepost(ctx, account.ID, post.ID); err != nil {
		t.Fatalf("DeleteRepost failed: %v", err)
	}

	// Check not exists
	exists, _ := store.ExistsRepost(ctx, account.ID, post.ID)
	if exists {
		t.Error("ExistsRepost after delete: got true, want false")
	}
}

func TestInteractionsStore_GetRepostedBy(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	author := createTestAccount(t, db, "author")
	reposter1 := createTestAccount(t, db, "reposter1")
	reposter2 := createTestAccount(t, db, "reposter2")

	postsStore := NewPostsStore(db)
	post := createTestPost(t, postsStore, author.ID)

	store := NewInteractionsStore(db)

	// Add reposts
	store.InsertRepost(ctx, &interactions.Repost{
		ID:        newTestID(),
		AccountID: reposter1.ID,
		PostID:    post.ID,
		CreatedAt: testTime(),
	})
	store.InsertRepost(ctx, &interactions.Repost{
		ID:        newTestID(),
		AccountID: reposter2.ID,
		PostID:    post.ID,
		CreatedAt: testTime(),
	})

	// Get reposted by
	ids, err := store.GetRepostedBy(ctx, post.ID, 10, 0)
	if err != nil {
		t.Fatalf("GetRepostedBy failed: %v", err)
	}

	if len(ids) != 2 {
		t.Errorf("GetRepostedBy count: got %d, want 2", len(ids))
	}
}

func TestInteractionsStore_RepostsCount(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	postsStore := NewPostsStore(db)
	post := createTestPost(t, postsStore, account.ID)

	store := NewInteractionsStore(db)

	// Increment
	if err := store.IncrementRepostsCount(ctx, post.ID); err != nil {
		t.Fatalf("IncrementRepostsCount failed: %v", err)
	}

	// Verify
	got, _ := postsStore.GetByID(ctx, post.ID)
	if got.RepostsCount != 1 {
		t.Errorf("RepostsCount: got %d, want 1", got.RepostsCount)
	}

	// Decrement
	if err := store.DecrementRepostsCount(ctx, post.ID); err != nil {
		t.Fatalf("DecrementRepostsCount failed: %v", err)
	}

	got2, _ := postsStore.GetByID(ctx, post.ID)
	if got2.RepostsCount != 0 {
		t.Errorf("RepostsCount after decrement: got %d, want 0", got2.RepostsCount)
	}
}

func TestInteractionsStore_Bookmark(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	postsStore := NewPostsStore(db)
	post := createTestPost(t, postsStore, account.ID)

	store := NewInteractionsStore(db)

	// Insert bookmark
	bookmark := &interactions.Bookmark{
		ID:        newTestID(),
		AccountID: account.ID,
		PostID:    post.ID,
		CreatedAt: testTime(),
	}

	if err := store.InsertBookmark(ctx, bookmark); err != nil {
		t.Fatalf("InsertBookmark failed: %v", err)
	}

	// Check exists
	exists, err := store.ExistsBookmark(ctx, account.ID, post.ID)
	if err != nil {
		t.Fatalf("ExistsBookmark failed: %v", err)
	}
	if !exists {
		t.Error("ExistsBookmark: got false, want true")
	}
}

func TestInteractionsStore_Unbookmark(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	postsStore := NewPostsStore(db)
	post := createTestPost(t, postsStore, account.ID)

	store := NewInteractionsStore(db)

	// Insert bookmark
	bookmark := &interactions.Bookmark{
		ID:        newTestID(),
		AccountID: account.ID,
		PostID:    post.ID,
		CreatedAt: testTime(),
	}
	store.InsertBookmark(ctx, bookmark)

	// Delete bookmark
	if err := store.DeleteBookmark(ctx, account.ID, post.ID); err != nil {
		t.Fatalf("DeleteBookmark failed: %v", err)
	}

	// Check not exists
	exists, _ := store.ExistsBookmark(ctx, account.ID, post.ID)
	if exists {
		t.Error("ExistsBookmark after delete: got true, want false")
	}
}

func TestInteractionsStore_GetBookmarkedPosts(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	postsStore := NewPostsStore(db)
	post1 := createTestPost(t, postsStore, account.ID)
	post2 := createTestPost(t, postsStore, account.ID)

	store := NewInteractionsStore(db)

	// Bookmark posts
	store.InsertBookmark(ctx, &interactions.Bookmark{
		ID:        newTestID(),
		AccountID: account.ID,
		PostID:    post1.ID,
		CreatedAt: testTime(),
	})
	store.InsertBookmark(ctx, &interactions.Bookmark{
		ID:        newTestID(),
		AccountID: account.ID,
		PostID:    post2.ID,
		CreatedAt: testTime(),
	})

	// Get bookmarked posts
	ids, err := store.GetBookmarkedPosts(ctx, account.ID, 10, 0)
	if err != nil {
		t.Fatalf("GetBookmarkedPosts failed: %v", err)
	}

	if len(ids) != 2 {
		t.Errorf("GetBookmarkedPosts count: got %d, want 2", len(ids))
	}
}

func TestInteractionsStore_GetPostState(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	postsStore := NewPostsStore(db)
	post := createTestPost(t, postsStore, account.ID)

	store := NewInteractionsStore(db)

	// Initially no interactions
	state, err := store.GetPostState(ctx, account.ID, post.ID)
	if err != nil {
		t.Fatalf("GetPostState failed: %v", err)
	}

	if state.Liked {
		t.Error("Liked: got true, want false")
	}
	if state.Reposted {
		t.Error("Reposted: got true, want false")
	}
	if state.Bookmarked {
		t.Error("Bookmarked: got true, want false")
	}

	// Add interactions
	store.InsertLike(ctx, &interactions.Like{
		ID:        newTestID(),
		AccountID: account.ID,
		PostID:    post.ID,
		CreatedAt: testTime(),
	})
	store.InsertRepost(ctx, &interactions.Repost{
		ID:        newTestID(),
		AccountID: account.ID,
		PostID:    post.ID,
		CreatedAt: testTime(),
	})
	store.InsertBookmark(ctx, &interactions.Bookmark{
		ID:        newTestID(),
		AccountID: account.ID,
		PostID:    post.ID,
		CreatedAt: testTime(),
	})

	// Now all should be true
	state2, err := store.GetPostState(ctx, account.ID, post.ID)
	if err != nil {
		t.Fatalf("GetPostState (2) failed: %v", err)
	}

	if !state2.Liked {
		t.Error("Liked: got false, want true")
	}
	if !state2.Reposted {
		t.Error("Reposted: got false, want true")
	}
	if !state2.Bookmarked {
		t.Error("Bookmarked: got false, want true")
	}
}

func TestInteractionsStore_DuplicateLike(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	postsStore := NewPostsStore(db)
	post := createTestPost(t, postsStore, account.ID)

	store := NewInteractionsStore(db)

	// Insert like
	like := &interactions.Like{
		ID:        newTestID(),
		AccountID: account.ID,
		PostID:    post.ID,
		CreatedAt: testTime(),
	}

	if err := store.InsertLike(ctx, like); err != nil {
		t.Fatalf("InsertLike failed: %v", err)
	}

	// Insert duplicate like should fail
	like2 := &interactions.Like{
		ID:        newTestID(),
		AccountID: account.ID,
		PostID:    post.ID,
		CreatedAt: testTime(),
	}

	err := store.InsertLike(ctx, like2)
	if err == nil {
		t.Error("expected error for duplicate like, got nil")
	}
}
