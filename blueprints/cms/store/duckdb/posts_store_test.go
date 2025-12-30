package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/cms/feature/posts"
)

func TestPostsStore_Create(t *testing.T) {
	db := setupTestDB(t)
	store := NewPostsStore(db)
	ctx := context.Background()

	publishedAt := testTime.Add(time.Hour)
	post := &posts.Post{
		ID:              "post-001",
		AuthorID:        "author-001",
		Title:           "Test Post",
		Slug:            "test-post",
		Excerpt:         "A test excerpt",
		Content:         "# Test Content\nThis is test content.",
		ContentFormat:   "markdown",
		FeaturedImageID: "img-001",
		Status:          "published",
		Visibility:      "public",
		Password:        "secret",
		PublishedAt:     &publishedAt,
		Meta:            `{"seo":"keywords"}`,
		ReadingTime:     5,
		WordCount:       100,
		AllowComments:   true,
		IsFeatured:      true,
		IsSticky:        false,
		SortOrder:       1,
		CreatedAt:       testTime,
		UpdatedAt:       testTime,
	}

	err := store.Create(ctx, post)
	assertNoError(t, err)

	got, err := store.GetByID(ctx, post.ID)
	assertNoError(t, err)
	assertEqual(t, "ID", got.ID, post.ID)
	assertEqual(t, "Title", got.Title, post.Title)
	assertEqual(t, "Slug", got.Slug, post.Slug)
	assertEqual(t, "Excerpt", got.Excerpt, post.Excerpt)
	assertEqual(t, "Content", got.Content, post.Content)
	assertEqual(t, "Status", got.Status, post.Status)
	assertEqual(t, "IsFeatured", got.IsFeatured, post.IsFeatured)
	if got.PublishedAt == nil {
		t.Error("expected PublishedAt to be set")
	}
}

func TestPostsStore_Create_MinimalFields(t *testing.T) {
	db := setupTestDB(t)
	store := NewPostsStore(db)
	ctx := context.Background()

	post := &posts.Post{
		ID:            "post-002",
		AuthorID:      "author-002",
		Title:         "Minimal Post",
		Slug:          "minimal-post",
		ContentFormat: "markdown",
		Status:        "draft",
		Visibility:    "public",
		CreatedAt:     testTime,
		UpdatedAt:     testTime,
	}

	err := store.Create(ctx, post)
	assertNoError(t, err)

	got, _ := store.GetByID(ctx, post.ID)
	assertEqual(t, "Excerpt", got.Excerpt, "")
	assertEqual(t, "Content", got.Content, "")
	if got.PublishedAt != nil {
		t.Error("expected PublishedAt to be nil")
	}
}

func TestPostsStore_Create_DuplicateSlug(t *testing.T) {
	db := setupTestDB(t)
	store := NewPostsStore(db)
	ctx := context.Background()

	post1 := &posts.Post{
		ID:            "post-dup-1",
		AuthorID:      "author-001",
		Title:         "Post 1",
		Slug:          "duplicate-slug",
		ContentFormat: "markdown",
		Status:        "draft",
		Visibility:    "public",
		CreatedAt:     testTime,
		UpdatedAt:     testTime,
	}
	assertNoError(t, store.Create(ctx, post1))

	post2 := &posts.Post{
		ID:            "post-dup-2",
		AuthorID:      "author-001",
		Title:         "Post 2",
		Slug:          "duplicate-slug",
		ContentFormat: "markdown",
		Status:        "draft",
		Visibility:    "public",
		CreatedAt:     testTime,
		UpdatedAt:     testTime,
	}
	err := store.Create(ctx, post2)
	assertError(t, err)
}

func TestPostsStore_GetByID(t *testing.T) {
	db := setupTestDB(t)
	store := NewPostsStore(db)
	ctx := context.Background()

	post := &posts.Post{
		ID:            "post-get-001",
		AuthorID:      "author-001",
		Title:         "Get Post",
		Slug:          "get-post",
		ContentFormat: "markdown",
		Status:        "draft",
		Visibility:    "public",
		CreatedAt:     testTime,
		UpdatedAt:     testTime,
	}
	assertNoError(t, store.Create(ctx, post))

	got, err := store.GetByID(ctx, post.ID)
	assertNoError(t, err)
	assertEqual(t, "ID", got.ID, post.ID)
}

func TestPostsStore_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	store := NewPostsStore(db)
	ctx := context.Background()

	got, err := store.GetByID(ctx, "nonexistent")
	assertNoError(t, err)
	if got != nil {
		t.Error("expected nil for non-existent post")
	}
}

func TestPostsStore_GetBySlug(t *testing.T) {
	db := setupTestDB(t)
	store := NewPostsStore(db)
	ctx := context.Background()

	post := &posts.Post{
		ID:            "post-slug-001",
		AuthorID:      "author-001",
		Title:         "Slug Post",
		Slug:          "my-unique-slug",
		ContentFormat: "markdown",
		Status:        "draft",
		Visibility:    "public",
		CreatedAt:     testTime,
		UpdatedAt:     testTime,
	}
	assertNoError(t, store.Create(ctx, post))

	got, err := store.GetBySlug(ctx, "my-unique-slug")
	assertNoError(t, err)
	assertEqual(t, "ID", got.ID, post.ID)
}

func TestPostsStore_GetBySlug_NotFound(t *testing.T) {
	db := setupTestDB(t)
	store := NewPostsStore(db)
	ctx := context.Background()

	got, err := store.GetBySlug(ctx, "nonexistent-slug")
	assertNoError(t, err)
	if got != nil {
		t.Error("expected nil for non-existent slug")
	}
}

func TestPostsStore_List(t *testing.T) {
	db := setupTestDB(t)
	store := NewPostsStore(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		post := &posts.Post{
			ID:            "post-list-" + string(rune('a'+i)),
			AuthorID:      "author-001",
			Title:         "List Post " + string(rune('A'+i)),
			Slug:          "list-post-" + string(rune('a'+i)),
			ContentFormat: "markdown",
			Status:        "published",
			Visibility:    "public",
			CreatedAt:     testTime.Add(time.Duration(i) * time.Hour),
			UpdatedAt:     testTime,
		}
		assertNoError(t, store.Create(ctx, post))
	}

	list, total, err := store.List(ctx, &posts.ListIn{Limit: 10})
	assertNoError(t, err)
	assertEqual(t, "total", total, 5)
	assertLen(t, list, 5)
}

func TestPostsStore_List_FilterByAuthor(t *testing.T) {
	db := setupTestDB(t)
	store := NewPostsStore(db)
	ctx := context.Background()

	authors := []string{"author-a", "author-a", "author-b"}
	for i, author := range authors {
		post := &posts.Post{
			ID:            "post-auth-" + string(rune('a'+i)),
			AuthorID:      author,
			Title:         "Author Post",
			Slug:          "auth-post-" + string(rune('a'+i)),
			ContentFormat: "markdown",
			Status:        "published",
			Visibility:    "public",
			CreatedAt:     testTime,
			UpdatedAt:     testTime,
		}
		assertNoError(t, store.Create(ctx, post))
	}

	list, total, err := store.List(ctx, &posts.ListIn{AuthorID: "author-a", Limit: 10})
	assertNoError(t, err)
	assertEqual(t, "total", total, 2)
	assertLen(t, list, 2)
}

func TestPostsStore_List_FilterByStatus(t *testing.T) {
	db := setupTestDB(t)
	store := NewPostsStore(db)
	ctx := context.Background()

	statuses := []string{"draft", "published", "published"}
	for i, status := range statuses {
		post := &posts.Post{
			ID:            "post-stat-" + string(rune('a'+i)),
			AuthorID:      "author-001",
			Title:         "Status Post",
			Slug:          "stat-post-" + string(rune('a'+i)),
			ContentFormat: "markdown",
			Status:        status,
			Visibility:    "public",
			CreatedAt:     testTime,
			UpdatedAt:     testTime,
		}
		assertNoError(t, store.Create(ctx, post))
	}

	list, total, err := store.List(ctx, &posts.ListIn{Status: "draft", Limit: 10})
	assertNoError(t, err)
	assertEqual(t, "total", total, 1)
	assertLen(t, list, 1)
}

func TestPostsStore_List_FilterByVisibility(t *testing.T) {
	db := setupTestDB(t)
	store := NewPostsStore(db)
	ctx := context.Background()

	visibilities := []string{"public", "private", "public"}
	for i, vis := range visibilities {
		post := &posts.Post{
			ID:            "post-vis-" + string(rune('a'+i)),
			AuthorID:      "author-001",
			Title:         "Visibility Post",
			Slug:          "vis-post-" + string(rune('a'+i)),
			ContentFormat: "markdown",
			Status:        "published",
			Visibility:    vis,
			CreatedAt:     testTime,
			UpdatedAt:     testTime,
		}
		assertNoError(t, store.Create(ctx, post))
	}

	list, total, err := store.List(ctx, &posts.ListIn{Visibility: "private", Limit: 10})
	assertNoError(t, err)
	assertEqual(t, "total", total, 1)
	assertLen(t, list, 1)
}

func TestPostsStore_List_FilterByCategory(t *testing.T) {
	db := setupTestDB(t)
	postsStore := NewPostsStore(db)
	ctx := context.Background()

	// Create posts
	post1 := &posts.Post{
		ID:            "post-cat-1",
		AuthorID:      "author-001",
		Title:         "Category Post 1",
		Slug:          "cat-post-1",
		ContentFormat: "markdown",
		Status:        "published",
		Visibility:    "public",
		CreatedAt:     testTime,
		UpdatedAt:     testTime,
	}
	post2 := &posts.Post{
		ID:            "post-cat-2",
		AuthorID:      "author-001",
		Title:         "Category Post 2",
		Slug:          "cat-post-2",
		ContentFormat: "markdown",
		Status:        "published",
		Visibility:    "public",
		CreatedAt:     testTime,
		UpdatedAt:     testTime,
	}
	assertNoError(t, postsStore.Create(ctx, post1))
	assertNoError(t, postsStore.Create(ctx, post2))

	// Set categories
	assertNoError(t, postsStore.SetCategories(ctx, post1.ID, []string{"cat-tech"}))

	list, total, err := postsStore.List(ctx, &posts.ListIn{CategoryID: "cat-tech", Limit: 10})
	assertNoError(t, err)
	assertEqual(t, "total", total, 1)
	assertEqual(t, "ID", list[0].ID, post1.ID)
}

func TestPostsStore_List_FilterByTag(t *testing.T) {
	db := setupTestDB(t)
	postsStore := NewPostsStore(db)
	ctx := context.Background()

	// Create posts
	post1 := &posts.Post{
		ID:            "post-tag-1",
		AuthorID:      "author-001",
		Title:         "Tag Post 1",
		Slug:          "tag-post-1",
		ContentFormat: "markdown",
		Status:        "published",
		Visibility:    "public",
		CreatedAt:     testTime,
		UpdatedAt:     testTime,
	}
	post2 := &posts.Post{
		ID:            "post-tag-2",
		AuthorID:      "author-001",
		Title:         "Tag Post 2",
		Slug:          "tag-post-2",
		ContentFormat: "markdown",
		Status:        "published",
		Visibility:    "public",
		CreatedAt:     testTime,
		UpdatedAt:     testTime,
	}
	assertNoError(t, postsStore.Create(ctx, post1))
	assertNoError(t, postsStore.Create(ctx, post2))

	// Set tags
	assertNoError(t, postsStore.SetTags(ctx, post1.ID, []string{"tag-golang"}))

	list, total, err := postsStore.List(ctx, &posts.ListIn{TagID: "tag-golang", Limit: 10})
	assertNoError(t, err)
	assertEqual(t, "total", total, 1)
	assertEqual(t, "ID", list[0].ID, post1.ID)
}

func TestPostsStore_List_FilterByFeatured(t *testing.T) {
	db := setupTestDB(t)
	store := NewPostsStore(db)
	ctx := context.Background()

	featured := []bool{true, false, true}
	for i, f := range featured {
		post := &posts.Post{
			ID:            "post-feat-" + string(rune('a'+i)),
			AuthorID:      "author-001",
			Title:         "Featured Post",
			Slug:          "feat-post-" + string(rune('a'+i)),
			ContentFormat: "markdown",
			Status:        "published",
			Visibility:    "public",
			IsFeatured:    f,
			CreatedAt:     testTime,
			UpdatedAt:     testTime,
		}
		assertNoError(t, store.Create(ctx, post))
	}

	featuredTrue := true
	list, total, err := store.List(ctx, &posts.ListIn{IsFeatured: &featuredTrue, Limit: 10})
	assertNoError(t, err)
	assertEqual(t, "total", total, 2)
	assertLen(t, list, 2)
}

func TestPostsStore_List_Search(t *testing.T) {
	db := setupTestDB(t)
	store := NewPostsStore(db)
	ctx := context.Background()

	postsData := []struct {
		title   string
		content string
	}{
		{"Go Programming", "Learn about Go language"},
		{"Python Tutorial", "Python is great"},
		{"Advanced Go Patterns", "Go patterns for experts"},
	}
	for i, p := range postsData {
		post := &posts.Post{
			ID:            "post-search-" + string(rune('a'+i)),
			AuthorID:      "author-001",
			Title:         p.title,
			Slug:          "search-post-" + string(rune('a'+i)),
			Content:       p.content,
			ContentFormat: "markdown",
			Status:        "published",
			Visibility:    "public",
			CreatedAt:     testTime,
			UpdatedAt:     testTime,
		}
		assertNoError(t, store.Create(ctx, post))
	}

	// Search by title
	list, total, err := store.List(ctx, &posts.ListIn{Search: "Go", Limit: 10})
	assertNoError(t, err)
	assertEqual(t, "total", total, 2)

	// Search by content
	list, total, err = store.List(ctx, &posts.ListIn{Search: "Python", Limit: 10})
	assertNoError(t, err)
	assertEqual(t, "total", total, 1)
	assertEqual(t, "Title", list[0].Title, "Python Tutorial")
}

func TestPostsStore_List_OrderBy(t *testing.T) {
	db := setupTestDB(t)
	store := NewPostsStore(db)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		post := &posts.Post{
			ID:            "post-order-" + string(rune('a'+i)),
			AuthorID:      "author-001",
			Title:         "Order Post " + string(rune('C'-i)), // C, B, A
			Slug:          "order-post-" + string(rune('a'+i)),
			ContentFormat: "markdown",
			Status:        "published",
			Visibility:    "public",
			CreatedAt:     testTime.Add(time.Duration(i) * time.Hour),
			UpdatedAt:     testTime,
		}
		assertNoError(t, store.Create(ctx, post))
	}

	// Order by title ASC
	list, _, err := store.List(ctx, &posts.ListIn{OrderBy: "title", Order: "ASC", Limit: 10})
	assertNoError(t, err)
	assertEqual(t, "Title[0]", list[0].Title, "Order Post A")
	assertEqual(t, "Title[2]", list[2].Title, "Order Post C")
}

func TestPostsStore_Update(t *testing.T) {
	db := setupTestDB(t)
	store := NewPostsStore(db)
	ctx := context.Background()

	post := &posts.Post{
		ID:            "post-update",
		AuthorID:      "author-001",
		Title:         "Original Title",
		Slug:          "update-post",
		ContentFormat: "markdown",
		Status:        "draft",
		Visibility:    "public",
		CreatedAt:     testTime,
		UpdatedAt:     testTime,
	}
	assertNoError(t, store.Create(ctx, post))

	err := store.Update(ctx, post.ID, &posts.UpdateIn{
		Title:   ptr("Updated Title"),
		Content: ptr("New content"),
		Status:  ptr("published"),
	})
	assertNoError(t, err)

	got, _ := store.GetByID(ctx, post.ID)
	assertEqual(t, "Title", got.Title, "Updated Title")
	assertEqual(t, "Content", got.Content, "New content")
	assertEqual(t, "Status", got.Status, "published")
}

func TestPostsStore_Update_PartialFields(t *testing.T) {
	db := setupTestDB(t)
	store := NewPostsStore(db)
	ctx := context.Background()

	post := &posts.Post{
		ID:            "post-partial",
		AuthorID:      "author-001",
		Title:         "Original Title",
		Slug:          "partial-post",
		Excerpt:       "Original Excerpt",
		ContentFormat: "markdown",
		Status:        "draft",
		Visibility:    "public",
		CreatedAt:     testTime,
		UpdatedAt:     testTime,
	}
	assertNoError(t, store.Create(ctx, post))

	// Only update title
	err := store.Update(ctx, post.ID, &posts.UpdateIn{
		Title: ptr("New Title"),
	})
	assertNoError(t, err)

	got, _ := store.GetByID(ctx, post.ID)
	assertEqual(t, "Title", got.Title, "New Title")
	assertEqual(t, "Excerpt", got.Excerpt, "Original Excerpt") // Unchanged
}

func TestPostsStore_Update_NoFields(t *testing.T) {
	db := setupTestDB(t)
	store := NewPostsStore(db)
	ctx := context.Background()

	post := &posts.Post{
		ID:            "post-noop",
		AuthorID:      "author-001",
		Title:         "NoOp Post",
		Slug:          "noop-post",
		ContentFormat: "markdown",
		Status:        "draft",
		Visibility:    "public",
		CreatedAt:     testTime,
		UpdatedAt:     testTime,
	}
	assertNoError(t, store.Create(ctx, post))

	err := store.Update(ctx, post.ID, &posts.UpdateIn{})
	assertNoError(t, err) // No-op should succeed
}

func TestPostsStore_Update_PublishedAt(t *testing.T) {
	db := setupTestDB(t)
	store := NewPostsStore(db)
	ctx := context.Background()

	post := &posts.Post{
		ID:            "post-pub-time",
		AuthorID:      "author-001",
		Title:         "Publish Time Post",
		Slug:          "pub-time-post",
		ContentFormat: "markdown",
		Status:        "draft",
		Visibility:    "public",
		CreatedAt:     testTime,
		UpdatedAt:     testTime,
	}
	assertNoError(t, store.Create(ctx, post))

	pubTime := testTime.Add(48 * time.Hour)
	err := store.Update(ctx, post.ID, &posts.UpdateIn{
		PublishedAt: &pubTime,
	})
	assertNoError(t, err)

	got, _ := store.GetByID(ctx, post.ID)
	if got.PublishedAt == nil {
		t.Error("expected PublishedAt to be set")
	}
}

func TestPostsStore_Delete(t *testing.T) {
	db := setupTestDB(t)
	store := NewPostsStore(db)
	ctx := context.Background()

	post := &posts.Post{
		ID:            "post-delete",
		AuthorID:      "author-001",
		Title:         "Delete Post",
		Slug:          "delete-post",
		ContentFormat: "markdown",
		Status:        "draft",
		Visibility:    "public",
		CreatedAt:     testTime,
		UpdatedAt:     testTime,
	}
	assertNoError(t, store.Create(ctx, post))

	err := store.Delete(ctx, post.ID)
	assertNoError(t, err)

	got, _ := store.GetByID(ctx, post.ID)
	if got != nil {
		t.Error("expected post to be deleted")
	}
}

func TestPostsStore_Delete_WithRelationships(t *testing.T) {
	db := setupTestDB(t)
	postsStore := NewPostsStore(db)
	ctx := context.Background()

	post := &posts.Post{
		ID:            "post-del-rel",
		AuthorID:      "author-001",
		Title:         "Delete With Relations",
		Slug:          "del-rel-post",
		ContentFormat: "markdown",
		Status:        "published",
		Visibility:    "public",
		CreatedAt:     testTime,
		UpdatedAt:     testTime,
	}
	assertNoError(t, postsStore.Create(ctx, post))

	// Add categories and tags
	assertNoError(t, postsStore.SetCategories(ctx, post.ID, []string{"cat-1", "cat-2"}))
	assertNoError(t, postsStore.SetTags(ctx, post.ID, []string{"tag-1", "tag-2"}))

	// Delete should cascade
	err := postsStore.Delete(ctx, post.ID)
	assertNoError(t, err)

	// Verify relationships are removed
	catIDs, _ := postsStore.GetCategoryIDs(ctx, post.ID)
	assertLen(t, catIDs, 0)

	tagIDs, _ := postsStore.GetTagIDs(ctx, post.ID)
	assertLen(t, tagIDs, 0)
}

func TestPostsStore_GetCategoryIDs(t *testing.T) {
	db := setupTestDB(t)
	store := NewPostsStore(db)
	ctx := context.Background()

	post := &posts.Post{
		ID:            "post-get-cats",
		AuthorID:      "author-001",
		Title:         "Get Categories Post",
		Slug:          "get-cats-post",
		ContentFormat: "markdown",
		Status:        "published",
		Visibility:    "public",
		CreatedAt:     testTime,
		UpdatedAt:     testTime,
	}
	assertNoError(t, store.Create(ctx, post))
	assertNoError(t, store.SetCategories(ctx, post.ID, []string{"cat-a", "cat-b", "cat-c"}))

	ids, err := store.GetCategoryIDs(ctx, post.ID)
	assertNoError(t, err)
	assertLen(t, ids, 3)
	// Should be in order
	assertEqual(t, "ids[0]", ids[0], "cat-a")
	assertEqual(t, "ids[1]", ids[1], "cat-b")
	assertEqual(t, "ids[2]", ids[2], "cat-c")
}

func TestPostsStore_GetCategoryIDs_Empty(t *testing.T) {
	db := setupTestDB(t)
	store := NewPostsStore(db)
	ctx := context.Background()

	post := &posts.Post{
		ID:            "post-no-cats",
		AuthorID:      "author-001",
		Title:         "No Categories Post",
		Slug:          "no-cats-post",
		ContentFormat: "markdown",
		Status:        "published",
		Visibility:    "public",
		CreatedAt:     testTime,
		UpdatedAt:     testTime,
	}
	assertNoError(t, store.Create(ctx, post))

	ids, err := store.GetCategoryIDs(ctx, post.ID)
	assertNoError(t, err)
	assertLen(t, ids, 0)
}

func TestPostsStore_GetTagIDs(t *testing.T) {
	db := setupTestDB(t)
	store := NewPostsStore(db)
	ctx := context.Background()

	post := &posts.Post{
		ID:            "post-get-tags",
		AuthorID:      "author-001",
		Title:         "Get Tags Post",
		Slug:          "get-tags-post",
		ContentFormat: "markdown",
		Status:        "published",
		Visibility:    "public",
		CreatedAt:     testTime,
		UpdatedAt:     testTime,
	}
	assertNoError(t, store.Create(ctx, post))
	assertNoError(t, store.SetTags(ctx, post.ID, []string{"tag-x", "tag-y"}))

	ids, err := store.GetTagIDs(ctx, post.ID)
	assertNoError(t, err)
	assertLen(t, ids, 2)
}

func TestPostsStore_SetCategories(t *testing.T) {
	db := setupTestDB(t)
	store := NewPostsStore(db)
	ctx := context.Background()

	post := &posts.Post{
		ID:            "post-set-cats",
		AuthorID:      "author-001",
		Title:         "Set Categories Post",
		Slug:          "set-cats-post",
		ContentFormat: "markdown",
		Status:        "published",
		Visibility:    "public",
		CreatedAt:     testTime,
		UpdatedAt:     testTime,
	}
	assertNoError(t, store.Create(ctx, post))

	err := store.SetCategories(ctx, post.ID, []string{"cat-1", "cat-2"})
	assertNoError(t, err)

	ids, _ := store.GetCategoryIDs(ctx, post.ID)
	assertLen(t, ids, 2)
}

func TestPostsStore_SetCategories_Replace(t *testing.T) {
	db := setupTestDB(t)
	store := NewPostsStore(db)
	ctx := context.Background()

	post := &posts.Post{
		ID:            "post-rep-cats",
		AuthorID:      "author-001",
		Title:         "Replace Categories Post",
		Slug:          "rep-cats-post",
		ContentFormat: "markdown",
		Status:        "published",
		Visibility:    "public",
		CreatedAt:     testTime,
		UpdatedAt:     testTime,
	}
	assertNoError(t, store.Create(ctx, post))
	assertNoError(t, store.SetCategories(ctx, post.ID, []string{"old-cat-1", "old-cat-2"}))

	// Replace with new categories
	err := store.SetCategories(ctx, post.ID, []string{"new-cat-1"})
	assertNoError(t, err)

	ids, _ := store.GetCategoryIDs(ctx, post.ID)
	assertLen(t, ids, 1)
	assertEqual(t, "ids[0]", ids[0], "new-cat-1")
}

func TestPostsStore_SetCategories_Empty(t *testing.T) {
	db := setupTestDB(t)
	store := NewPostsStore(db)
	ctx := context.Background()

	post := &posts.Post{
		ID:            "post-clear-cats",
		AuthorID:      "author-001",
		Title:         "Clear Categories Post",
		Slug:          "clear-cats-post",
		ContentFormat: "markdown",
		Status:        "published",
		Visibility:    "public",
		CreatedAt:     testTime,
		UpdatedAt:     testTime,
	}
	assertNoError(t, store.Create(ctx, post))
	assertNoError(t, store.SetCategories(ctx, post.ID, []string{"cat-1"}))

	// Clear categories
	err := store.SetCategories(ctx, post.ID, []string{})
	assertNoError(t, err)

	ids, _ := store.GetCategoryIDs(ctx, post.ID)
	assertLen(t, ids, 0)
}

func TestPostsStore_SetTags(t *testing.T) {
	db := setupTestDB(t)
	store := NewPostsStore(db)
	ctx := context.Background()

	post := &posts.Post{
		ID:            "post-set-tags",
		AuthorID:      "author-001",
		Title:         "Set Tags Post",
		Slug:          "set-tags-post",
		ContentFormat: "markdown",
		Status:        "published",
		Visibility:    "public",
		CreatedAt:     testTime,
		UpdatedAt:     testTime,
	}
	assertNoError(t, store.Create(ctx, post))

	err := store.SetTags(ctx, post.ID, []string{"tag-1", "tag-2", "tag-3"})
	assertNoError(t, err)

	ids, _ := store.GetTagIDs(ctx, post.ID)
	assertLen(t, ids, 3)
}

func TestPostsStore_SetTags_Replace(t *testing.T) {
	db := setupTestDB(t)
	store := NewPostsStore(db)
	ctx := context.Background()

	post := &posts.Post{
		ID:            "post-rep-tags",
		AuthorID:      "author-001",
		Title:         "Replace Tags Post",
		Slug:          "rep-tags-post",
		ContentFormat: "markdown",
		Status:        "published",
		Visibility:    "public",
		CreatedAt:     testTime,
		UpdatedAt:     testTime,
	}
	assertNoError(t, store.Create(ctx, post))
	assertNoError(t, store.SetTags(ctx, post.ID, []string{"old-tag"}))

	err := store.SetTags(ctx, post.ID, []string{"new-tag-1", "new-tag-2"})
	assertNoError(t, err)

	ids, _ := store.GetTagIDs(ctx, post.ID)
	assertLen(t, ids, 2)
}

func TestPostsStore_SetTags_Empty(t *testing.T) {
	db := setupTestDB(t)
	store := NewPostsStore(db)
	ctx := context.Background()

	post := &posts.Post{
		ID:            "post-clear-tags",
		AuthorID:      "author-001",
		Title:         "Clear Tags Post",
		Slug:          "clear-tags-post",
		ContentFormat: "markdown",
		Status:        "published",
		Visibility:    "public",
		CreatedAt:     testTime,
		UpdatedAt:     testTime,
	}
	assertNoError(t, store.Create(ctx, post))
	assertNoError(t, store.SetTags(ctx, post.ID, []string{"tag-1"}))

	err := store.SetTags(ctx, post.ID, []string{})
	assertNoError(t, err)

	ids, _ := store.GetTagIDs(ctx, post.ID)
	assertLen(t, ids, 0)
}
