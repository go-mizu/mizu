package duckdb

import (
	"context"
	"testing"

	"github.com/go-mizu/blueprints/cms/feature/pages"
)

func TestPagesStore_Create(t *testing.T) {
	db := setupTestDB(t)
	store := NewPagesStore(db)
	ctx := context.Background()

	page := &pages.Page{
		ID:              "page-001",
		AuthorID:        "author-001",
		Title:           "Test Page",
		Slug:            "test-page",
		Content:         "# Page Content",
		ContentFormat:   "markdown",
		FeaturedImageID: "img-001",
		Template:        "default",
		Status:          "published",
		Visibility:      "public",
		Meta:            `{"seo":"keywords"}`,
		SortOrder:       1,
		CreatedAt:       testTime,
		UpdatedAt:       testTime,
	}

	err := store.Create(ctx, page)
	assertNoError(t, err)

	got, err := store.GetByID(ctx, page.ID)
	assertNoError(t, err)
	assertEqual(t, "ID", got.ID, page.ID)
	assertEqual(t, "Title", got.Title, page.Title)
	assertEqual(t, "Slug", got.Slug, page.Slug)
	assertEqual(t, "Content", got.Content, page.Content)
	assertEqual(t, "Template", got.Template, page.Template)
	assertEqual(t, "Status", got.Status, page.Status)
}

func TestPagesStore_Create_MinimalFields(t *testing.T) {
	db := setupTestDB(t)
	store := NewPagesStore(db)
	ctx := context.Background()

	page := &pages.Page{
		ID:            "page-002",
		AuthorID:      "author-002",
		Title:         "Minimal Page",
		Slug:          "minimal-page",
		ContentFormat: "markdown",
		Status:        "draft",
		Visibility:    "public",
		CreatedAt:     testTime,
		UpdatedAt:     testTime,
	}

	err := store.Create(ctx, page)
	assertNoError(t, err)

	got, _ := store.GetByID(ctx, page.ID)
	assertEqual(t, "Content", got.Content, "")
	assertEqual(t, "ParentID", got.ParentID, "")
	assertEqual(t, "Template", got.Template, "")
}

func TestPagesStore_Create_WithParent(t *testing.T) {
	db := setupTestDB(t)
	store := NewPagesStore(db)
	ctx := context.Background()

	// Create parent page
	parent := &pages.Page{
		ID:            "page-parent",
		AuthorID:      "author-001",
		Title:         "Parent Page",
		Slug:          "parent-page",
		ContentFormat: "markdown",
		Status:        "published",
		Visibility:    "public",
		CreatedAt:     testTime,
		UpdatedAt:     testTime,
	}
	assertNoError(t, store.Create(ctx, parent))

	// Create child page
	child := &pages.Page{
		ID:            "page-child",
		AuthorID:      "author-001",
		ParentID:      parent.ID,
		Title:         "Child Page",
		Slug:          "child-page",
		ContentFormat: "markdown",
		Status:        "published",
		Visibility:    "public",
		CreatedAt:     testTime,
		UpdatedAt:     testTime,
	}
	err := store.Create(ctx, child)
	assertNoError(t, err)

	got, _ := store.GetByID(ctx, child.ID)
	assertEqual(t, "ParentID", got.ParentID, parent.ID)
}

func TestPagesStore_Create_WithTemplate(t *testing.T) {
	db := setupTestDB(t)
	store := NewPagesStore(db)
	ctx := context.Background()

	page := &pages.Page{
		ID:            "page-template",
		AuthorID:      "author-001",
		Title:         "Template Page",
		Slug:          "template-page",
		ContentFormat: "markdown",
		Template:      "landing",
		Status:        "published",
		Visibility:    "public",
		CreatedAt:     testTime,
		UpdatedAt:     testTime,
	}

	err := store.Create(ctx, page)
	assertNoError(t, err)

	got, _ := store.GetByID(ctx, page.ID)
	assertEqual(t, "Template", got.Template, "landing")
}

func TestPagesStore_GetByID(t *testing.T) {
	db := setupTestDB(t)
	store := NewPagesStore(db)
	ctx := context.Background()

	page := &pages.Page{
		ID:            "page-get",
		AuthorID:      "author-001",
		Title:         "Get Page",
		Slug:          "get-page",
		ContentFormat: "markdown",
		Status:        "published",
		Visibility:    "public",
		CreatedAt:     testTime,
		UpdatedAt:     testTime,
	}
	assertNoError(t, store.Create(ctx, page))

	got, err := store.GetByID(ctx, page.ID)
	assertNoError(t, err)
	assertEqual(t, "ID", got.ID, page.ID)
}

func TestPagesStore_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	store := NewPagesStore(db)
	ctx := context.Background()

	got, err := store.GetByID(ctx, "nonexistent")
	assertNoError(t, err)
	if got != nil {
		t.Error("expected nil for non-existent page")
	}
}

func TestPagesStore_GetBySlug(t *testing.T) {
	db := setupTestDB(t)
	store := NewPagesStore(db)
	ctx := context.Background()

	page := &pages.Page{
		ID:            "page-slug",
		AuthorID:      "author-001",
		Title:         "Slug Page",
		Slug:          "my-unique-page-slug",
		ContentFormat: "markdown",
		Status:        "published",
		Visibility:    "public",
		CreatedAt:     testTime,
		UpdatedAt:     testTime,
	}
	assertNoError(t, store.Create(ctx, page))

	got, err := store.GetBySlug(ctx, "my-unique-page-slug")
	assertNoError(t, err)
	assertEqual(t, "ID", got.ID, page.ID)
}

func TestPagesStore_GetByParentAndSlug(t *testing.T) {
	db := setupTestDB(t)
	store := NewPagesStore(db)
	ctx := context.Background()

	// Create parent
	parent := &pages.Page{
		ID:            "page-parent-slug",
		AuthorID:      "author-001",
		Title:         "Parent",
		Slug:          "parent",
		ContentFormat: "markdown",
		Status:        "published",
		Visibility:    "public",
		CreatedAt:     testTime,
		UpdatedAt:     testTime,
	}
	assertNoError(t, store.Create(ctx, parent))

	// Create child with specific slug
	child := &pages.Page{
		ID:            "page-child-slug",
		AuthorID:      "author-001",
		ParentID:      parent.ID,
		Title:         "Child",
		Slug:          "child",
		ContentFormat: "markdown",
		Status:        "published",
		Visibility:    "public",
		CreatedAt:     testTime,
		UpdatedAt:     testTime,
	}
	assertNoError(t, store.Create(ctx, child))

	got, err := store.GetByParentAndSlug(ctx, parent.ID, "child")
	assertNoError(t, err)
	assertEqual(t, "ID", got.ID, child.ID)
}

func TestPagesStore_GetByParentAndSlug_RootLevel(t *testing.T) {
	db := setupTestDB(t)
	store := NewPagesStore(db)
	ctx := context.Background()

	page := &pages.Page{
		ID:            "page-root",
		AuthorID:      "author-001",
		Title:         "Root Page",
		Slug:          "root-page",
		ContentFormat: "markdown",
		Status:        "published",
		Visibility:    "public",
		CreatedAt:     testTime,
		UpdatedAt:     testTime,
	}
	assertNoError(t, store.Create(ctx, page))

	// Get root page (no parent)
	got, err := store.GetByParentAndSlug(ctx, "", "root-page")
	assertNoError(t, err)
	assertEqual(t, "ID", got.ID, page.ID)
}

func TestPagesStore_List(t *testing.T) {
	db := setupTestDB(t)
	store := NewPagesStore(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		page := &pages.Page{
			ID:            "page-list-" + string(rune('a'+i)),
			AuthorID:      "author-001",
			Title:         "List Page " + string(rune('A'+i)),
			Slug:          "list-page-" + string(rune('a'+i)),
			ContentFormat: "markdown",
			Status:        "published",
			Visibility:    "public",
			CreatedAt:     testTime,
			UpdatedAt:     testTime,
		}
		assertNoError(t, store.Create(ctx, page))
	}

	list, total, err := store.List(ctx, &pages.ListIn{Limit: 10})
	assertNoError(t, err)
	assertEqual(t, "total", total, 5)
	assertLen(t, list, 5)
}

func TestPagesStore_List_FilterByParent(t *testing.T) {
	db := setupTestDB(t)
	store := NewPagesStore(db)
	ctx := context.Background()

	// Create parent
	parent := &pages.Page{
		ID:            "page-parent-list",
		AuthorID:      "author-001",
		Title:         "Parent List",
		Slug:          "parent-list",
		ContentFormat: "markdown",
		Status:        "published",
		Visibility:    "public",
		CreatedAt:     testTime,
		UpdatedAt:     testTime,
	}
	assertNoError(t, store.Create(ctx, parent))

	// Create children
	for i := 0; i < 3; i++ {
		child := &pages.Page{
			ID:            "page-child-list-" + string(rune('a'+i)),
			AuthorID:      "author-001",
			ParentID:      parent.ID,
			Title:         "Child " + string(rune('A'+i)),
			Slug:          "child-list-" + string(rune('a'+i)),
			ContentFormat: "markdown",
			Status:        "published",
			Visibility:    "public",
			CreatedAt:     testTime,
			UpdatedAt:     testTime,
		}
		assertNoError(t, store.Create(ctx, child))
	}

	// Also create orphan page
	orphan := &pages.Page{
		ID:            "page-orphan",
		AuthorID:      "author-001",
		Title:         "Orphan",
		Slug:          "orphan",
		ContentFormat: "markdown",
		Status:        "published",
		Visibility:    "public",
		CreatedAt:     testTime,
		UpdatedAt:     testTime,
	}
	assertNoError(t, store.Create(ctx, orphan))

	list, total, err := store.List(ctx, &pages.ListIn{ParentID: parent.ID, Limit: 10})
	assertNoError(t, err)
	assertEqual(t, "total", total, 3)
	assertLen(t, list, 3)
}

func TestPagesStore_List_FilterByAuthor(t *testing.T) {
	db := setupTestDB(t)
	store := NewPagesStore(db)
	ctx := context.Background()

	authors := []string{"author-a", "author-a", "author-b"}
	for i, author := range authors {
		page := &pages.Page{
			ID:            "page-auth-" + string(rune('a'+i)),
			AuthorID:      author,
			Title:         "Author Page",
			Slug:          "auth-page-" + string(rune('a'+i)),
			ContentFormat: "markdown",
			Status:        "published",
			Visibility:    "public",
			CreatedAt:     testTime,
			UpdatedAt:     testTime,
		}
		assertNoError(t, store.Create(ctx, page))
	}

	list, total, err := store.List(ctx, &pages.ListIn{AuthorID: "author-a", Limit: 10})
	assertNoError(t, err)
	assertEqual(t, "total", total, 2)
	assertLen(t, list, 2)
}

func TestPagesStore_List_FilterByStatus(t *testing.T) {
	db := setupTestDB(t)
	store := NewPagesStore(db)
	ctx := context.Background()

	statuses := []string{"draft", "published", "published"}
	for i, status := range statuses {
		page := &pages.Page{
			ID:            "page-stat-" + string(rune('a'+i)),
			AuthorID:      "author-001",
			Title:         "Status Page",
			Slug:          "stat-page-" + string(rune('a'+i)),
			ContentFormat: "markdown",
			Status:        status,
			Visibility:    "public",
			CreatedAt:     testTime,
			UpdatedAt:     testTime,
		}
		assertNoError(t, store.Create(ctx, page))
	}

	list, total, err := store.List(ctx, &pages.ListIn{Status: "draft", Limit: 10})
	assertNoError(t, err)
	assertEqual(t, "total", total, 1)
	assertLen(t, list, 1)
}

func TestPagesStore_List_FilterByVisibility(t *testing.T) {
	db := setupTestDB(t)
	store := NewPagesStore(db)
	ctx := context.Background()

	visibilities := []string{"public", "private", "public"}
	for i, vis := range visibilities {
		page := &pages.Page{
			ID:            "page-vis-" + string(rune('a'+i)),
			AuthorID:      "author-001",
			Title:         "Visibility Page",
			Slug:          "vis-page-" + string(rune('a'+i)),
			ContentFormat: "markdown",
			Status:        "published",
			Visibility:    vis,
			CreatedAt:     testTime,
			UpdatedAt:     testTime,
		}
		assertNoError(t, store.Create(ctx, page))
	}

	list, total, err := store.List(ctx, &pages.ListIn{Visibility: "private", Limit: 10})
	assertNoError(t, err)
	assertEqual(t, "total", total, 1)
	assertLen(t, list, 1)
}

func TestPagesStore_List_Search(t *testing.T) {
	db := setupTestDB(t)
	store := NewPagesStore(db)
	ctx := context.Background()

	pagesData := []struct {
		title   string
		content string
	}{
		{"About Us", "Learn about our company"},
		{"Contact", "Contact information"},
		{"About Our Services", "Services we provide"},
	}
	for i, p := range pagesData {
		page := &pages.Page{
			ID:            "page-search-" + string(rune('a'+i)),
			AuthorID:      "author-001",
			Title:         p.title,
			Slug:          "search-page-" + string(rune('a'+i)),
			Content:       p.content,
			ContentFormat: "markdown",
			Status:        "published",
			Visibility:    "public",
			CreatedAt:     testTime,
			UpdatedAt:     testTime,
		}
		assertNoError(t, store.Create(ctx, page))
	}

	// Search by title
	list, total, err := store.List(ctx, &pages.ListIn{Search: "About", Limit: 10})
	assertNoError(t, err)
	assertEqual(t, "total", total, 2)
	assertLen(t, list, 2)
}

func TestPagesStore_GetTree(t *testing.T) {
	db := setupTestDB(t)
	store := NewPagesStore(db)
	ctx := context.Background()

	// Create pages with different sort orders
	for i := 0; i < 3; i++ {
		page := &pages.Page{
			ID:            "page-tree-" + string(rune('a'+i)),
			AuthorID:      "author-001",
			Title:         "Tree Page " + string(rune('C'-i)), // C, B, A
			Slug:          "tree-page-" + string(rune('a'+i)),
			ContentFormat: "markdown",
			Status:        "published",
			Visibility:    "public",
			SortOrder:     i,
			CreatedAt:     testTime,
			UpdatedAt:     testTime,
		}
		assertNoError(t, store.Create(ctx, page))
	}

	list, err := store.GetTree(ctx)
	assertNoError(t, err)
	assertLen(t, list, 3)
	// Should be ordered by sort_order
	assertEqual(t, "Title[0]", list[0].Title, "Tree Page C")
}

func TestPagesStore_GetTree_Empty(t *testing.T) {
	db := setupTestDB(t)
	store := NewPagesStore(db)
	ctx := context.Background()

	list, err := store.GetTree(ctx)
	assertNoError(t, err)
	assertLen(t, list, 0)
}

func TestPagesStore_Update(t *testing.T) {
	db := setupTestDB(t)
	store := NewPagesStore(db)
	ctx := context.Background()

	page := &pages.Page{
		ID:            "page-update",
		AuthorID:      "author-001",
		Title:         "Original Title",
		Slug:          "update-page",
		ContentFormat: "markdown",
		Status:        "draft",
		Visibility:    "public",
		CreatedAt:     testTime,
		UpdatedAt:     testTime,
	}
	assertNoError(t, store.Create(ctx, page))

	err := store.Update(ctx, page.ID, &pages.UpdateIn{
		Title:   ptr("Updated Title"),
		Content: ptr("New content"),
		Status:  ptr("published"),
	})
	assertNoError(t, err)

	got, _ := store.GetByID(ctx, page.ID)
	assertEqual(t, "Title", got.Title, "Updated Title")
	assertEqual(t, "Content", got.Content, "New content")
	assertEqual(t, "Status", got.Status, "published")
}

func TestPagesStore_Update_PartialFields(t *testing.T) {
	db := setupTestDB(t)
	store := NewPagesStore(db)
	ctx := context.Background()

	page := &pages.Page{
		ID:            "page-partial",
		AuthorID:      "author-001",
		Title:         "Original Title",
		Slug:          "partial-page",
		Content:       "Original Content",
		ContentFormat: "markdown",
		Status:        "draft",
		Visibility:    "public",
		CreatedAt:     testTime,
		UpdatedAt:     testTime,
	}
	assertNoError(t, store.Create(ctx, page))

	err := store.Update(ctx, page.ID, &pages.UpdateIn{
		Title: ptr("New Title"),
	})
	assertNoError(t, err)

	got, _ := store.GetByID(ctx, page.ID)
	assertEqual(t, "Title", got.Title, "New Title")
	assertEqual(t, "Content", got.Content, "Original Content") // Unchanged
}

func TestPagesStore_Update_ChangeParent(t *testing.T) {
	db := setupTestDB(t)
	store := NewPagesStore(db)
	ctx := context.Background()

	// Create two parents
	parent1 := &pages.Page{
		ID:            "page-parent-1",
		AuthorID:      "author-001",
		Title:         "Parent 1",
		Slug:          "parent-1",
		ContentFormat: "markdown",
		Status:        "published",
		Visibility:    "public",
		CreatedAt:     testTime,
		UpdatedAt:     testTime,
	}
	parent2 := &pages.Page{
		ID:            "page-parent-2",
		AuthorID:      "author-001",
		Title:         "Parent 2",
		Slug:          "parent-2",
		ContentFormat: "markdown",
		Status:        "published",
		Visibility:    "public",
		CreatedAt:     testTime,
		UpdatedAt:     testTime,
	}
	assertNoError(t, store.Create(ctx, parent1))
	assertNoError(t, store.Create(ctx, parent2))

	// Create child under parent1
	child := &pages.Page{
		ID:            "page-movable",
		AuthorID:      "author-001",
		ParentID:      parent1.ID,
		Title:         "Movable Child",
		Slug:          "movable-child",
		ContentFormat: "markdown",
		Status:        "published",
		Visibility:    "public",
		CreatedAt:     testTime,
		UpdatedAt:     testTime,
	}
	assertNoError(t, store.Create(ctx, child))

	// Move to parent2
	err := store.Update(ctx, child.ID, &pages.UpdateIn{
		ParentID: ptr(parent2.ID),
	})
	assertNoError(t, err)

	got, _ := store.GetByID(ctx, child.ID)
	assertEqual(t, "ParentID", got.ParentID, parent2.ID)
}

func TestPagesStore_Update_SetTemplate(t *testing.T) {
	db := setupTestDB(t)
	store := NewPagesStore(db)
	ctx := context.Background()

	page := &pages.Page{
		ID:            "page-set-template",
		AuthorID:      "author-001",
		Title:         "Template Page",
		Slug:          "set-template-page",
		ContentFormat: "markdown",
		Status:        "published",
		Visibility:    "public",
		CreatedAt:     testTime,
		UpdatedAt:     testTime,
	}
	assertNoError(t, store.Create(ctx, page))

	err := store.Update(ctx, page.ID, &pages.UpdateIn{
		Template: ptr("full-width"),
	})
	assertNoError(t, err)

	got, _ := store.GetByID(ctx, page.ID)
	assertEqual(t, "Template", got.Template, "full-width")
}

func TestPagesStore_Delete(t *testing.T) {
	db := setupTestDB(t)
	store := NewPagesStore(db)
	ctx := context.Background()

	page := &pages.Page{
		ID:            "page-delete",
		AuthorID:      "author-001",
		Title:         "Delete Page",
		Slug:          "delete-page",
		ContentFormat: "markdown",
		Status:        "draft",
		Visibility:    "public",
		CreatedAt:     testTime,
		UpdatedAt:     testTime,
	}
	assertNoError(t, store.Create(ctx, page))

	err := store.Delete(ctx, page.ID)
	assertNoError(t, err)

	got, _ := store.GetByID(ctx, page.ID)
	if got != nil {
		t.Error("expected page to be deleted")
	}
}
