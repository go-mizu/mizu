package duckdb

import (
	"context"
	"testing"

	"github.com/go-mizu/blueprints/social/feature/accounts"
	"github.com/go-mizu/blueprints/social/feature/posts"
)

func TestSearchStore_SearchAccounts(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	accountsStore := NewAccountsStore(db)
	searchStore := NewSearchStore(db)

	// Create discoverable accounts
	_ = createTestAccount(t, db, "johndoe")
	_ = createTestAccount(t, db, "janedoe")
	acc3 := createTestAccount(t, db, "bobsmith")

	// Update one to be non-discoverable
	discoverable := false
	accountsStore.Update(ctx, acc3.ID, &accounts.UpdateIn{Discoverable: &discoverable})

	// Search for "doe"
	results, err := searchStore.SearchAccounts(ctx, "doe", 10, 0)
	if err != nil {
		t.Fatalf("SearchAccounts failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("SearchAccounts count: got %d, want 2", len(results))
	}
}

func TestSearchStore_SearchAccounts_ExcludesSuspended(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	accountsStore := NewAccountsStore(db)
	searchStore := NewSearchStore(db)

	// Create accounts
	acc1 := createTestAccount(t, db, "testuser1")
	acc2 := createTestAccount(t, db, "testuser2")

	// Suspend one
	accountsStore.SetSuspended(ctx, acc2.ID, true)

	// Search
	results, err := searchStore.SearchAccounts(ctx, "testuser", 10, 0)
	if err != nil {
		t.Fatalf("SearchAccounts failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("SearchAccounts count: got %d, want 1", len(results))
	}

	if results[0].ID != acc1.ID {
		t.Errorf("Expected non-suspended account, got %s", results[0].ID)
	}
}

func TestSearchStore_SearchAccounts_ExactMatchFirst(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	searchStore := NewSearchStore(db)

	// Create accounts - exact match should come first
	createTestAccount(t, db, "alice_test")
	exact := createTestAccount(t, db, "alice")

	// Search for exact match
	results, err := searchStore.SearchAccounts(ctx, "alice", 10, 0)
	if err != nil {
		t.Fatalf("SearchAccounts failed: %v", err)
	}

	if len(results) < 1 {
		t.Fatalf("Expected at least 1 result")
	}

	// Exact match should be first
	if results[0].ID != exact.ID {
		t.Errorf("Expected exact match first, got %s", results[0].Username)
	}
}

func TestSearchStore_SearchAccounts_Pagination(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	searchStore := NewSearchStore(db)

	// Create multiple accounts
	for i := 0; i < 5; i++ {
		createTestAccount(t, db, "searchtest"+string(rune('0'+i)))
	}

	// Get first page
	page1, err := searchStore.SearchAccounts(ctx, "searchtest", 2, 0)
	if err != nil {
		t.Fatalf("SearchAccounts (page 1) failed: %v", err)
	}

	if len(page1) != 2 {
		t.Errorf("Page 1 count: got %d, want 2", len(page1))
	}

	// Get second page
	page2, err := searchStore.SearchAccounts(ctx, "searchtest", 2, 2)
	if err != nil {
		t.Fatalf("SearchAccounts (page 2) failed: %v", err)
	}

	if len(page2) != 2 {
		t.Errorf("Page 2 count: got %d, want 2", len(page2))
	}

	// Ensure no overlap
	for _, a1 := range page1 {
		for _, a2 := range page2 {
			if a1.ID == a2.ID {
				t.Error("Pages should not overlap")
			}
		}
	}
}

func TestSearchStore_SearchPosts(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	postsStore := NewPostsStore(db)
	searchStore := NewSearchStore(db)

	// Create posts with different content
	postsStore.Insert(ctx, &posts.Post{
		ID:         newTestID(),
		AccountID:  account.ID,
		Content:    "Hello golang world",
		Visibility: posts.VisibilityPublic,
		CreatedAt:  testTime(),
	})
	postsStore.Insert(ctx, &posts.Post{
		ID:         newTestID(),
		AccountID:  account.ID,
		Content:    "Python is great",
		Visibility: posts.VisibilityPublic,
		CreatedAt:  testTime(),
	})
	postsStore.Insert(ctx, &posts.Post{
		ID:         newTestID(),
		AccountID:  account.ID,
		Content:    "Learning golang basics",
		Visibility: posts.VisibilityPublic,
		CreatedAt:  testTime(),
	})

	// Search for "golang"
	results, err := searchStore.SearchPosts(ctx, "golang", 10, 0, 0, 0, false)
	if err != nil {
		t.Fatalf("SearchPosts failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("SearchPosts count: got %d, want 2", len(results))
	}
}

func TestSearchStore_SearchPosts_OnlyPublic(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	postsStore := NewPostsStore(db)
	searchStore := NewSearchStore(db)

	// Create public post
	postsStore.Insert(ctx, &posts.Post{
		ID:         newTestID(),
		AccountID:  account.ID,
		Content:    "Public searchable post",
		Visibility: posts.VisibilityPublic,
		CreatedAt:  testTime(),
	})

	// Create private post
	postsStore.Insert(ctx, &posts.Post{
		ID:         newTestID(),
		AccountID:  account.ID,
		Content:    "Private searchable post",
		Visibility: "followers",
		CreatedAt:  testTime(),
	})

	// Search
	results, err := searchStore.SearchPosts(ctx, "searchable", 10, 0, 0, 0, false)
	if err != nil {
		t.Fatalf("SearchPosts failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("SearchPosts count: got %d, want 1", len(results))
	}
}

func TestSearchStore_SearchPosts_MinLikes(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	postsStore := NewPostsStore(db)
	interactionsStore := NewInteractionsStore(db)
	searchStore := NewSearchStore(db)

	// Create posts
	post1 := &posts.Post{
		ID:         newTestID(),
		AccountID:  account.ID,
		Content:    "Popular searchterm post",
		Visibility: posts.VisibilityPublic,
		CreatedAt:  testTime(),
	}
	postsStore.Insert(ctx, post1)

	post2 := &posts.Post{
		ID:         newTestID(),
		AccountID:  account.ID,
		Content:    "Unpopular searchterm post",
		Visibility: posts.VisibilityPublic,
		CreatedAt:  testTime(),
	}
	postsStore.Insert(ctx, post2)

	// Add likes to post1
	for i := 0; i < 5; i++ {
		interactionsStore.IncrementLikesCount(ctx, post1.ID)
	}

	// Search with min likes
	results, err := searchStore.SearchPosts(ctx, "searchterm", 10, 0, 3, 0, false)
	if err != nil {
		t.Fatalf("SearchPosts failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("SearchPosts with minLikes count: got %d, want 1", len(results))
	}
}

func TestSearchStore_SearchHashtags(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	postsStore := NewPostsStore(db)
	searchStore := NewSearchStore(db)

	// Create hashtags
	postsStore.UpsertHashtag(ctx, "golang")
	postsStore.UpsertHashtag(ctx, "goland")
	postsStore.UpsertHashtag(ctx, "python")

	// Link some posts to hashtags to create posts_count
	post := createTestPost(t, postsStore, account.ID)
	golangID, _ := postsStore.UpsertHashtag(ctx, "golang")
	postsStore.LinkPostHashtag(ctx, post.ID, golangID)

	// Search for "go"
	results, err := searchStore.SearchHashtags(ctx, "go", 10)
	if err != nil {
		t.Fatalf("SearchHashtags failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("SearchHashtags count: got %d, want 2", len(results))
	}
}

func TestSearchStore_SearchHashtags_OrderByPostsCount(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	postsStore := NewPostsStore(db)
	searchStore := NewSearchStore(db)

	// Create hashtags with different post counts
	tag1ID, _ := postsStore.UpsertHashtag(ctx, "testtag1")
	tag2ID, _ := postsStore.UpsertHashtag(ctx, "testtag2")

	// Link more posts to tag2
	for i := 0; i < 3; i++ {
		post := createTestPost(t, postsStore, account.ID)
		postsStore.LinkPostHashtag(ctx, post.ID, tag2ID)
	}

	// Link one post to tag1
	post := createTestPost(t, postsStore, account.ID)
	postsStore.LinkPostHashtag(ctx, post.ID, tag1ID)

	// Search - should return tag2 first (more posts)
	results, err := searchStore.SearchHashtags(ctx, "testtag", 10)
	if err != nil {
		t.Fatalf("SearchHashtags failed: %v", err)
	}

	if len(results) < 2 {
		t.Fatalf("Expected at least 2 results")
	}

	if results[0].Name != "testtag2" {
		t.Errorf("Expected testtag2 first (most posts), got %s", results[0].Name)
	}
}

func TestSearchStore_SuggestHashtags(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	postsStore := NewPostsStore(db)
	searchStore := NewSearchStore(db)

	// Create hashtags
	postsStore.UpsertHashtag(ctx, "programming")
	postsStore.UpsertHashtag(ctx, "productivity")
	postsStore.UpsertHashtag(ctx, "design")

	// Get suggestions for "pro"
	suggestions, err := searchStore.SuggestHashtags(ctx, "pro", 10)
	if err != nil {
		t.Fatalf("SuggestHashtags failed: %v", err)
	}

	if len(suggestions) != 2 {
		t.Errorf("SuggestHashtags count: got %d, want 2", len(suggestions))
	}
}

func TestSearchStore_SuggestHashtags_Limit(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	postsStore := NewPostsStore(db)
	searchStore := NewSearchStore(db)

	// Create many hashtags
	for i := 0; i < 10; i++ {
		postsStore.UpsertHashtag(ctx, "suggest"+string(rune('a'+i)))
	}

	// Get limited suggestions
	suggestions, err := searchStore.SuggestHashtags(ctx, "suggest", 3)
	if err != nil {
		t.Fatalf("SuggestHashtags failed: %v", err)
	}

	if len(suggestions) != 3 {
		t.Errorf("SuggestHashtags count: got %d, want 3", len(suggestions))
	}
}

func TestSearchStore_SearchAccounts_NoResults(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	searchStore := NewSearchStore(db)

	// Search for non-existent
	results, err := searchStore.SearchAccounts(ctx, "nonexistentuser123", 10, 0)
	if err != nil {
		t.Fatalf("SearchAccounts failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("SearchAccounts count: got %d, want 0", len(results))
	}
}

func TestSearchStore_SearchPosts_NoResults(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	searchStore := NewSearchStore(db)

	// Search for non-existent
	results, err := searchStore.SearchPosts(ctx, "nonexistentcontent123", 10, 0, 0, 0, false)
	if err != nil {
		t.Fatalf("SearchPosts failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("SearchPosts count: got %d, want 0", len(results))
	}
}

func TestSearchStore_SearchHashtags_NoResults(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	searchStore := NewSearchStore(db)

	// Search for non-existent
	results, err := searchStore.SearchHashtags(ctx, "nonexistenttag123", 10)
	if err != nil {
		t.Fatalf("SearchHashtags failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("SearchHashtags count: got %d, want 0", len(results))
	}
}

func TestSearchStore_SearchAccounts_CaseInsensitive(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	searchStore := NewSearchStore(db)

	// Create account
	createTestAccount(t, db, "CamelCase")

	// Search with different cases
	results1, _ := searchStore.SearchAccounts(ctx, "camelcase", 10, 0)
	results2, _ := searchStore.SearchAccounts(ctx, "CAMELCASE", 10, 0)
	results3, _ := searchStore.SearchAccounts(ctx, "CamelCase", 10, 0)

	if len(results1) != 1 || len(results2) != 1 || len(results3) != 1 {
		t.Error("Search should be case insensitive")
	}
}

func TestSearchStore_SearchPosts_CaseInsensitive(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	postsStore := NewPostsStore(db)
	searchStore := NewSearchStore(db)

	// Create post
	postsStore.Insert(ctx, &posts.Post{
		ID:         newTestID(),
		AccountID:  account.ID,
		Content:    "MixedCase Content Here",
		Visibility: posts.VisibilityPublic,
		CreatedAt:  testTime(),
	})

	// Search with different cases
	results1, _ := searchStore.SearchPosts(ctx, "mixedcase", 10, 0, 0, 0, false)
	results2, _ := searchStore.SearchPosts(ctx, "MIXEDCASE", 10, 0, 0, 0, false)
	results3, _ := searchStore.SearchPosts(ctx, "MixedCase", 10, 0, 0, 0, false)

	if len(results1) != 1 || len(results2) != 1 || len(results3) != 1 {
		t.Error("Search should be case insensitive")
	}
}
