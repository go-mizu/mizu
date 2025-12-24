package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/social/feature/posts"
)

func TestTrendingStore_GetTrendingTags(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	postsStore := NewPostsStore(db)
	trendingStore := NewTrendingStore(db)

	// Create recent posts with hashtags
	for i := 0; i < 3; i++ {
		post := &posts.Post{
			ID:         newTestID(),
			AccountID:  account.ID,
			Content:    "Trending test",
			Visibility: posts.VisibilityPublic,
			CreatedAt:  time.Now().Add(-time.Hour), // Recent
		}
		postsStore.Insert(ctx, post)
		hashtagID, _ := postsStore.UpsertHashtag(ctx, "trending")
		postsStore.LinkPostHashtag(ctx, post.ID, hashtagID)
	}

	// Get trending tags
	tags, err := trendingStore.GetTrendingTags(ctx, 10, 0)
	if err != nil {
		t.Fatalf("GetTrendingTags failed: %v", err)
	}

	if len(tags) < 1 {
		t.Errorf("GetTrendingTags count: got %d, want at least 1", len(tags))
	}

	if tags[0].Name != "trending" {
		t.Errorf("Expected trending tag, got %s", tags[0].Name)
	}

	if tags[0].PostsCount != 3 {
		t.Errorf("PostsCount: got %d, want 3", tags[0].PostsCount)
	}
}

func TestTrendingStore_GetTrendingTags_ExcludesOldPosts(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	postsStore := NewPostsStore(db)
	trendingStore := NewTrendingStore(db)

	// Create old post with hashtag (more than 24 hours old)
	oldPost := &posts.Post{
		ID:         newTestID(),
		AccountID:  account.ID,
		Content:    "Old post",
		Visibility: posts.VisibilityPublic,
		CreatedAt:  time.Now().Add(-48 * time.Hour), // 2 days old
	}
	postsStore.Insert(ctx, oldPost)
	oldHashtagID, _ := postsStore.UpsertHashtag(ctx, "oldtag")
	postsStore.LinkPostHashtag(ctx, oldPost.ID, oldHashtagID)

	// Create recent post with different hashtag
	recentPost := &posts.Post{
		ID:         newTestID(),
		AccountID:  account.ID,
		Content:    "Recent post",
		Visibility: posts.VisibilityPublic,
		CreatedAt:  time.Now().Add(-time.Hour), // 1 hour old
	}
	postsStore.Insert(ctx, recentPost)
	recentHashtagID, _ := postsStore.UpsertHashtag(ctx, "recenttag")
	postsStore.LinkPostHashtag(ctx, recentPost.ID, recentHashtagID)

	// Get trending tags - should only show recent
	tags, err := trendingStore.GetTrendingTags(ctx, 10, 0)
	if err != nil {
		t.Fatalf("GetTrendingTags failed: %v", err)
	}

	// Should only have the recent tag
	for _, tag := range tags {
		if tag.Name == "oldtag" {
			t.Error("Old tags should not appear in trending")
		}
	}
}

func TestTrendingStore_GetTrendingTags_OnlyPublic(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	postsStore := NewPostsStore(db)
	trendingStore := NewTrendingStore(db)

	// Create private post with hashtag
	privatePost := &posts.Post{
		ID:         newTestID(),
		AccountID:  account.ID,
		Content:    "Private post",
		Visibility: "followers",
		CreatedAt:  time.Now().Add(-time.Hour),
	}
	postsStore.Insert(ctx, privatePost)
	privateHashtagID, _ := postsStore.UpsertHashtag(ctx, "privatetag")
	postsStore.LinkPostHashtag(ctx, privatePost.ID, privateHashtagID)

	// Create public post with hashtag
	publicPost := &posts.Post{
		ID:         newTestID(),
		AccountID:  account.ID,
		Content:    "Public post",
		Visibility: posts.VisibilityPublic,
		CreatedAt:  time.Now().Add(-time.Hour),
	}
	postsStore.Insert(ctx, publicPost)
	publicHashtagID, _ := postsStore.UpsertHashtag(ctx, "publictag")
	postsStore.LinkPostHashtag(ctx, publicPost.ID, publicHashtagID)

	// Get trending tags
	tags, err := trendingStore.GetTrendingTags(ctx, 10, 0)
	if err != nil {
		t.Fatalf("GetTrendingTags failed: %v", err)
	}

	// Should only have the public tag
	for _, tag := range tags {
		if tag.Name == "privatetag" {
			t.Error("Private post hashtags should not appear in trending")
		}
	}
}

func TestTrendingStore_GetTrendingTags_OrderByCount(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	postsStore := NewPostsStore(db)
	trendingStore := NewTrendingStore(db)

	// Create posts with different hashtag frequencies
	popularHashtagID, _ := postsStore.UpsertHashtag(ctx, "populartag")
	lessPopularHashtagID, _ := postsStore.UpsertHashtag(ctx, "lesspopulartag")

	// 5 posts with popular tag
	for i := 0; i < 5; i++ {
		post := &posts.Post{
			ID:         newTestID(),
			AccountID:  account.ID,
			Content:    "Popular content",
			Visibility: posts.VisibilityPublic,
			CreatedAt:  time.Now().Add(-time.Hour),
		}
		postsStore.Insert(ctx, post)
		postsStore.LinkPostHashtag(ctx, post.ID, popularHashtagID)
	}

	// 2 posts with less popular tag
	for i := 0; i < 2; i++ {
		post := &posts.Post{
			ID:         newTestID(),
			AccountID:  account.ID,
			Content:    "Less popular content",
			Visibility: posts.VisibilityPublic,
			CreatedAt:  time.Now().Add(-time.Hour),
		}
		postsStore.Insert(ctx, post)
		postsStore.LinkPostHashtag(ctx, post.ID, lessPopularHashtagID)
	}

	// Get trending tags
	tags, err := trendingStore.GetTrendingTags(ctx, 10, 0)
	if err != nil {
		t.Fatalf("GetTrendingTags failed: %v", err)
	}

	if len(tags) < 2 {
		t.Fatalf("Expected at least 2 tags")
	}

	// First should be more popular
	if tags[0].Name != "populartag" {
		t.Errorf("Expected populartag first, got %s", tags[0].Name)
	}
}

func TestTrendingStore_GetTrendingTags_Pagination(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	postsStore := NewPostsStore(db)
	trendingStore := NewTrendingStore(db)

	// Create multiple hashtags with posts
	for i := 0; i < 5; i++ {
		hashtagID, _ := postsStore.UpsertHashtag(ctx, "pagetag"+string(rune('a'+i)))
		post := &posts.Post{
			ID:         newTestID(),
			AccountID:  account.ID,
			Content:    "Page test",
			Visibility: posts.VisibilityPublic,
			CreatedAt:  time.Now().Add(-time.Hour),
		}
		postsStore.Insert(ctx, post)
		postsStore.LinkPostHashtag(ctx, post.ID, hashtagID)
	}

	// Get first page
	page1, err := trendingStore.GetTrendingTags(ctx, 2, 0)
	if err != nil {
		t.Fatalf("GetTrendingTags (page 1) failed: %v", err)
	}

	if len(page1) != 2 {
		t.Errorf("Page 1 count: got %d, want 2", len(page1))
	}

	// Get second page
	page2, err := trendingStore.GetTrendingTags(ctx, 2, 2)
	if err != nil {
		t.Fatalf("GetTrendingTags (page 2) failed: %v", err)
	}

	if len(page2) != 2 {
		t.Errorf("Page 2 count: got %d, want 2", len(page2))
	}
}

func TestTrendingStore_GetTrendingPosts(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	postsStore := NewPostsStore(db)
	interactionsStore := NewInteractionsStore(db)
	trendingStore := NewTrendingStore(db)

	// Create recent post with engagement
	post := &posts.Post{
		ID:         newTestID(),
		AccountID:  account.ID,
		Content:    "Trending post",
		Visibility: posts.VisibilityPublic,
		CreatedAt:  time.Now().Add(-time.Hour),
	}
	postsStore.Insert(ctx, post)

	// Add likes
	for i := 0; i < 5; i++ {
		interactionsStore.IncrementLikesCount(ctx, post.ID)
	}

	// Get trending posts
	trending, err := trendingStore.GetTrendingPosts(ctx, 10, 0)
	if err != nil {
		t.Fatalf("GetTrendingPosts failed: %v", err)
	}

	if len(trending) < 1 {
		t.Errorf("GetTrendingPosts count: got %d, want at least 1", len(trending))
	}
}

func TestTrendingStore_GetTrendingPosts_ExcludesReplies(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	postsStore := NewPostsStore(db)
	trendingStore := NewTrendingStore(db)

	// Create parent post
	parent := &posts.Post{
		ID:         newTestID(),
		AccountID:  account.ID,
		Content:    "Parent post",
		Visibility: posts.VisibilityPublic,
		CreatedAt:  time.Now().Add(-time.Hour),
	}
	postsStore.Insert(ctx, parent)

	// Create reply
	reply := &posts.Post{
		ID:         newTestID(),
		AccountID:  account.ID,
		Content:    "Reply post",
		ReplyToID:  parent.ID,
		Visibility: posts.VisibilityPublic,
		CreatedAt:  time.Now().Add(-time.Hour),
	}
	postsStore.Insert(ctx, reply)

	// Get trending posts
	trending, err := trendingStore.GetTrendingPosts(ctx, 10, 0)
	if err != nil {
		t.Fatalf("GetTrendingPosts failed: %v", err)
	}

	// Should only have parent, not reply
	for _, p := range trending {
		if p.ID == reply.ID {
			t.Error("Replies should not appear in trending posts")
		}
	}
}

func TestTrendingStore_GetTrendingPosts_OnlyPublic(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	postsStore := NewPostsStore(db)
	trendingStore := NewTrendingStore(db)

	// Create private post
	privatePost := &posts.Post{
		ID:         newTestID(),
		AccountID:  account.ID,
		Content:    "Private post",
		Visibility: "followers",
		CreatedAt:  time.Now().Add(-time.Hour),
	}
	postsStore.Insert(ctx, privatePost)

	// Create public post
	publicPost := &posts.Post{
		ID:         newTestID(),
		AccountID:  account.ID,
		Content:    "Public post",
		Visibility: posts.VisibilityPublic,
		CreatedAt:  time.Now().Add(-time.Hour),
	}
	postsStore.Insert(ctx, publicPost)

	// Get trending posts
	trending, err := trendingStore.GetTrendingPosts(ctx, 10, 0)
	if err != nil {
		t.Fatalf("GetTrendingPosts failed: %v", err)
	}

	// Should only have public post
	for _, p := range trending {
		if p.ID == privatePost.ID {
			t.Error("Private posts should not appear in trending")
		}
	}
}

func TestTrendingStore_GetTrendingPosts_OrderByEngagement(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	postsStore := NewPostsStore(db)
	interactionsStore := NewInteractionsStore(db)
	trendingStore := NewTrendingStore(db)

	// Create less engaging post
	lessPopular := &posts.Post{
		ID:         newTestID(),
		AccountID:  account.ID,
		Content:    "Less popular",
		Visibility: posts.VisibilityPublic,
		CreatedAt:  time.Now().Add(-time.Hour),
	}
	postsStore.Insert(ctx, lessPopular)
	interactionsStore.IncrementLikesCount(ctx, lessPopular.ID)

	// Create more engaging post
	morePopular := &posts.Post{
		ID:         newTestID(),
		AccountID:  account.ID,
		Content:    "More popular",
		Visibility: posts.VisibilityPublic,
		CreatedAt:  time.Now().Add(-time.Hour),
	}
	postsStore.Insert(ctx, morePopular)
	for i := 0; i < 10; i++ {
		interactionsStore.IncrementLikesCount(ctx, morePopular.ID)
	}

	// Get trending posts
	trending, err := trendingStore.GetTrendingPosts(ctx, 10, 0)
	if err != nil {
		t.Fatalf("GetTrendingPosts failed: %v", err)
	}

	if len(trending) < 2 {
		t.Fatalf("Expected at least 2 posts")
	}

	// More popular should be first
	if trending[0].ID != morePopular.ID {
		t.Errorf("Expected more popular post first")
	}
}

func TestTrendingStore_ComputeTrendingTags(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	postsStore := NewPostsStore(db)
	trendingStore := NewTrendingStore(db)

	// Create posts with hashtags
	hashtagID, _ := postsStore.UpsertHashtag(ctx, "computetag")
	post := &posts.Post{
		ID:         newTestID(),
		AccountID:  account.ID,
		Content:    "Compute test",
		Visibility: posts.VisibilityPublic,
		CreatedAt:  time.Now().Add(-time.Hour),
	}
	postsStore.Insert(ctx, post)
	postsStore.LinkPostHashtag(ctx, post.ID, hashtagID)

	// Compute trending tags
	tags, err := trendingStore.ComputeTrendingTags(ctx, 24*time.Hour, 10)
	if err != nil {
		t.Fatalf("ComputeTrendingTags failed: %v", err)
	}

	if len(tags) < 1 {
		t.Errorf("ComputeTrendingTags count: got %d, want at least 1", len(tags))
	}
}

func TestTrendingStore_ComputeTrendingPosts(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	postsStore := NewPostsStore(db)
	trendingStore := NewTrendingStore(db)

	// Create post
	post := &posts.Post{
		ID:         newTestID(),
		AccountID:  account.ID,
		Content:    "Compute post test",
		Visibility: posts.VisibilityPublic,
		CreatedAt:  time.Now().Add(-time.Hour),
	}
	postsStore.Insert(ctx, post)

	// Compute trending posts
	trending, err := trendingStore.ComputeTrendingPosts(ctx, 24*time.Hour, 10)
	if err != nil {
		t.Fatalf("ComputeTrendingPosts failed: %v", err)
	}

	if len(trending) < 1 {
		t.Errorf("ComputeTrendingPosts count: got %d, want at least 1", len(trending))
	}
}

func TestTrendingStore_GetTrendingTags_Empty(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	trendingStore := NewTrendingStore(db)

	// Get trending tags from empty database
	tags, err := trendingStore.GetTrendingTags(ctx, 10, 0)
	if err != nil {
		t.Fatalf("GetTrendingTags failed: %v", err)
	}

	if len(tags) != 0 {
		t.Errorf("GetTrendingTags from empty db: got %d, want 0", len(tags))
	}
}

func TestTrendingStore_GetTrendingPosts_Empty(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	trendingStore := NewTrendingStore(db)

	// Get trending posts from empty database
	trending, err := trendingStore.GetTrendingPosts(ctx, 10, 0)
	if err != nil {
		t.Fatalf("GetTrendingPosts failed: %v", err)
	}

	if len(trending) != 0 {
		t.Errorf("GetTrendingPosts from empty db: got %d, want 0", len(trending))
	}
}
