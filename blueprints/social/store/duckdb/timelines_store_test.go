package duckdb

import (
	"context"
	"testing"

	"github.com/go-mizu/blueprints/social/feature/interactions"
	"github.com/go-mizu/blueprints/social/feature/lists"
	"github.com/go-mizu/blueprints/social/feature/posts"
	"github.com/go-mizu/blueprints/social/feature/relationships"
)

func TestTimelinesStore_GetHomeFeed(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	user := createTestAccount(t, db, "user")
	friend := createTestAccount(t, db, "friend")
	stranger := createTestAccount(t, db, "stranger")

	postsStore := NewPostsStore(db)
	relStore := NewRelationshipsStore(db)
	timelinesStore := NewTimelinesStore(db)

	// user follows friend
	relStore.InsertFollow(ctx, &relationships.Follow{
		ID:          newTestID(),
		FollowerID:  user.ID,
		FollowingID: friend.ID,
		Pending:     false,
		CreatedAt:   testTime(),
	})

	// Create posts
	for i := 0; i < 2; i++ {
		createTestPost(t, postsStore, user.ID)     // user's own posts
		createTestPost(t, postsStore, friend.ID)   // friend's posts
		createTestPost(t, postsStore, stranger.ID) // stranger's posts
	}

	// Get home feed
	feed, err := timelinesStore.GetHomeFeed(ctx, user.ID, 10, "", "")
	if err != nil {
		t.Fatalf("GetHomeFeed failed: %v", err)
	}

	// Should see own posts (2) + friend's posts (2) = 4
	if len(feed) != 4 {
		t.Errorf("GetHomeFeed count: got %d, want 4", len(feed))
	}

	// Verify no stranger's posts
	for _, post := range feed {
		if post.AccountID == stranger.ID {
			t.Error("Home feed should not contain stranger's posts")
		}
	}
}

func TestTimelinesStore_GetPublicFeed(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	postsStore := NewPostsStore(db)
	timelinesStore := NewTimelinesStore(db)

	// Create public posts
	for i := 0; i < 3; i++ {
		createTestPost(t, postsStore, account.ID)
	}

	// Create private post (followers only)
	privatePost := &posts.Post{
		ID:         newTestID(),
		AccountID:  account.ID,
		Content:    "Private post",
		Visibility: "followers",
		CreatedAt:  testTime(),
	}
	postsStore.Insert(ctx, privatePost)

	// Get public feed
	feed, err := timelinesStore.GetPublicFeed(ctx, 10, "", "", false)
	if err != nil {
		t.Fatalf("GetPublicFeed failed: %v", err)
	}

	// Should only see public posts
	if len(feed) != 3 {
		t.Errorf("GetPublicFeed count: got %d, want 3", len(feed))
	}
}

func TestTimelinesStore_GetPublicFeed_ExcludesReplies(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	postsStore := NewPostsStore(db)
	timelinesStore := NewTimelinesStore(db)

	// Create regular post
	parent := createTestPost(t, postsStore, account.ID)

	// Create reply
	reply := &posts.Post{
		ID:         newTestID(),
		AccountID:  account.ID,
		Content:    "Reply",
		ReplyToID:  parent.ID,
		Visibility: posts.VisibilityPublic,
		CreatedAt:  testTime(),
	}
	postsStore.Insert(ctx, reply)

	// Get public feed (should exclude replies)
	feed, err := timelinesStore.GetPublicFeed(ctx, 10, "", "", false)
	if err != nil {
		t.Fatalf("GetPublicFeed failed: %v", err)
	}

	if len(feed) != 1 {
		t.Errorf("GetPublicFeed count: got %d, want 1", len(feed))
	}
}

func TestTimelinesStore_GetUserFeed(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	user := createTestAccount(t, db, "testuser")
	other := createTestAccount(t, db, "other")
	postsStore := NewPostsStore(db)
	timelinesStore := NewTimelinesStore(db)

	// Create posts for user
	for i := 0; i < 3; i++ {
		createTestPost(t, postsStore, user.ID)
	}

	// Create posts for other
	createTestPost(t, postsStore, other.ID)

	// Get user feed
	feed, err := timelinesStore.GetUserFeed(ctx, user.ID, 10, "", "", false, false)
	if err != nil {
		t.Fatalf("GetUserFeed failed: %v", err)
	}

	if len(feed) != 3 {
		t.Errorf("GetUserFeed count: got %d, want 3", len(feed))
	}
}

func TestTimelinesStore_GetUserFeed_IncludeReplies(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	user := createTestAccount(t, db, "testuser")
	postsStore := NewPostsStore(db)
	timelinesStore := NewTimelinesStore(db)

	// Create regular post
	parent := createTestPost(t, postsStore, user.ID)

	// Create reply
	reply := &posts.Post{
		ID:         newTestID(),
		AccountID:  user.ID,
		Content:    "Reply",
		ReplyToID:  parent.ID,
		Visibility: posts.VisibilityPublic,
		CreatedAt:  testTime(),
	}
	postsStore.Insert(ctx, reply)

	// Without replies
	feedNoReplies, err := timelinesStore.GetUserFeed(ctx, user.ID, 10, "", "", false, false)
	if err != nil {
		t.Fatalf("GetUserFeed failed: %v", err)
	}
	if len(feedNoReplies) != 1 {
		t.Errorf("GetUserFeed (no replies) count: got %d, want 1", len(feedNoReplies))
	}

	// With replies
	feedWithReplies, err := timelinesStore.GetUserFeed(ctx, user.ID, 10, "", "", true, false)
	if err != nil {
		t.Fatalf("GetUserFeed (with replies) failed: %v", err)
	}
	if len(feedWithReplies) != 2 {
		t.Errorf("GetUserFeed (with replies) count: got %d, want 2", len(feedWithReplies))
	}
}

func TestTimelinesStore_GetHashtagFeed(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	postsStore := NewPostsStore(db)
	timelinesStore := NewTimelinesStore(db)

	// Create posts with hashtag
	for i := 0; i < 3; i++ {
		post := createTestPost(t, postsStore, account.ID)
		hashtagID, _ := postsStore.UpsertHashtag(ctx, "golang")
		postsStore.LinkPostHashtag(ctx, post.ID, hashtagID)
	}

	// Create post without hashtag
	createTestPost(t, postsStore, account.ID)

	// Get hashtag feed
	feed, err := timelinesStore.GetHashtagFeed(ctx, "golang", 10, "", "")
	if err != nil {
		t.Fatalf("GetHashtagFeed failed: %v", err)
	}

	if len(feed) != 3 {
		t.Errorf("GetHashtagFeed count: got %d, want 3", len(feed))
	}
}

func TestTimelinesStore_GetListFeed(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	owner := createTestAccount(t, db, "owner")
	member1 := createTestAccount(t, db, "member1")
	member2 := createTestAccount(t, db, "member2")
	nonMember := createTestAccount(t, db, "nonmember")

	postsStore := NewPostsStore(db)
	listsStore := NewListsStore(db)
	timelinesStore := NewTimelinesStore(db)

	// Create list
	list := &lists.List{
		ID:        newTestID(),
		AccountID: owner.ID,
		Title:     "Test List",
		CreatedAt: testTime(),
		UpdatedAt: testTime(),
	}
	listsStore.Insert(ctx, list)

	// Add members
	listsStore.InsertMember(ctx, &lists.ListMember{
		ListID:    list.ID,
		AccountID: member1.ID,
		CreatedAt: testTime(),
	})
	listsStore.InsertMember(ctx, &lists.ListMember{
		ListID:    list.ID,
		AccountID: member2.ID,
		CreatedAt: testTime(),
	})

	// Create posts
	createTestPost(t, postsStore, member1.ID)
	createTestPost(t, postsStore, member2.ID)
	createTestPost(t, postsStore, nonMember.ID)

	// Get list feed
	feed, err := timelinesStore.GetListFeed(ctx, list.ID, 10, "", "")
	if err != nil {
		t.Fatalf("GetListFeed failed: %v", err)
	}

	// Should only see posts from members
	if len(feed) != 2 {
		t.Errorf("GetListFeed count: got %d, want 2", len(feed))
	}
}

func TestTimelinesStore_GetBookmarksFeed(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	user := createTestAccount(t, db, "user")
	author := createTestAccount(t, db, "author")

	postsStore := NewPostsStore(db)
	interactionsStore := NewInteractionsStore(db)
	timelinesStore := NewTimelinesStore(db)

	// Create posts
	post1 := createTestPost(t, postsStore, author.ID)
	post2 := createTestPost(t, postsStore, author.ID)
	createTestPost(t, postsStore, author.ID) // not bookmarked

	// Bookmark posts
	interactionsStore.InsertBookmark(ctx, &interactions.Bookmark{
		ID:        newTestID(),
		AccountID: user.ID,
		PostID:    post1.ID,
		CreatedAt: testTime(),
	})
	interactionsStore.InsertBookmark(ctx, &interactions.Bookmark{
		ID:        newTestID(),
		AccountID: user.ID,
		PostID:    post2.ID,
		CreatedAt: testTime(),
	})

	// Get bookmarks feed
	feed, err := timelinesStore.GetBookmarksFeed(ctx, user.ID, 10, "", "")
	if err != nil {
		t.Fatalf("GetBookmarksFeed failed: %v", err)
	}

	if len(feed) != 2 {
		t.Errorf("GetBookmarksFeed count: got %d, want 2", len(feed))
	}
}

func TestTimelinesStore_GetLikesFeed(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	user := createTestAccount(t, db, "user")
	author := createTestAccount(t, db, "author")

	postsStore := NewPostsStore(db)
	interactionsStore := NewInteractionsStore(db)
	timelinesStore := NewTimelinesStore(db)

	// Create posts
	post1 := createTestPost(t, postsStore, author.ID)
	post2 := createTestPost(t, postsStore, author.ID)
	createTestPost(t, postsStore, author.ID) // not liked

	// Like posts
	interactionsStore.InsertLike(ctx, &interactions.Like{
		ID:        newTestID(),
		AccountID: user.ID,
		PostID:    post1.ID,
		CreatedAt: testTime(),
	})
	interactionsStore.InsertLike(ctx, &interactions.Like{
		ID:        newTestID(),
		AccountID: user.ID,
		PostID:    post2.ID,
		CreatedAt: testTime(),
	})

	// Get likes feed
	feed, err := timelinesStore.GetLikesFeed(ctx, user.ID, 10, "", "")
	if err != nil {
		t.Fatalf("GetLikesFeed failed: %v", err)
	}

	if len(feed) != 2 {
		t.Errorf("GetLikesFeed count: got %d, want 2", len(feed))
	}
}

func TestTimelinesStore_Pagination(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	postsStore := NewPostsStore(db)
	timelinesStore := NewTimelinesStore(db)

	// Create posts
	var postIDs []string
	for i := 0; i < 5; i++ {
		post := createTestPost(t, postsStore, account.ID)
		postIDs = append(postIDs, post.ID)
	}

	// Get first page
	page1, err := timelinesStore.GetPublicFeed(ctx, 2, "", "", false)
	if err != nil {
		t.Fatalf("GetPublicFeed failed: %v", err)
	}

	if len(page1) != 2 {
		t.Errorf("Page 1 count: got %d, want 2", len(page1))
	}

	// Get second page using maxID
	lastID := page1[len(page1)-1].ID
	page2, err := timelinesStore.GetPublicFeed(ctx, 2, lastID, "", false)
	if err != nil {
		t.Fatalf("GetPublicFeed (page 2) failed: %v", err)
	}

	if len(page2) != 2 {
		t.Errorf("Page 2 count: got %d, want 2", len(page2))
	}

	// Ensure no overlap
	for _, p1 := range page1 {
		for _, p2 := range page2 {
			if p1.ID == p2.ID {
				t.Error("Pages should not overlap")
			}
		}
	}
}
