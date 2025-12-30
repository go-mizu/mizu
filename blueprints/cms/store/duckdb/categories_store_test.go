package duckdb

import (
	"context"
	"testing"

	"github.com/go-mizu/blueprints/cms/feature/categories"
)

func TestCategoriesStore_Create(t *testing.T) {
	db := setupTestDB(t)
	store := NewCategoriesStore(db)
	ctx := context.Background()

	cat := &categories.Category{
		ID:              "cat-001",
		Name:            "Technology",
		Slug:            "technology",
		Description:     "Tech related posts",
		FeaturedImageID: "img-001",
		Meta:            `{"icon":"tech"}`,
		SortOrder:       1,
		PostCount:       0,
		CreatedAt:       testTime,
		UpdatedAt:       testTime,
	}

	err := store.Create(ctx, cat)
	assertNoError(t, err)

	got, err := store.GetByID(ctx, cat.ID)
	assertNoError(t, err)
	assertEqual(t, "ID", got.ID, cat.ID)
	assertEqual(t, "Name", got.Name, cat.Name)
	assertEqual(t, "Slug", got.Slug, cat.Slug)
	assertEqual(t, "Description", got.Description, cat.Description)
	assertEqual(t, "SortOrder", got.SortOrder, cat.SortOrder)
}

func TestCategoriesStore_Create_WithParent(t *testing.T) {
	db := setupTestDB(t)
	store := NewCategoriesStore(db)
	ctx := context.Background()

	// Create parent category
	parent := &categories.Category{
		ID:        "cat-parent",
		Name:      "Parent Category",
		Slug:      "parent-category",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.Create(ctx, parent))

	// Create child category
	child := &categories.Category{
		ID:        "cat-child",
		ParentID:  parent.ID,
		Name:      "Child Category",
		Slug:      "child-category",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	err := store.Create(ctx, child)
	assertNoError(t, err)

	got, _ := store.GetByID(ctx, child.ID)
	assertEqual(t, "ParentID", got.ParentID, parent.ID)
}

func TestCategoriesStore_Create_DuplicateSlug(t *testing.T) {
	db := setupTestDB(t)
	store := NewCategoriesStore(db)
	ctx := context.Background()

	cat1 := &categories.Category{
		ID:        "cat-dup-1",
		Name:      "Category 1",
		Slug:      "duplicate-slug",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.Create(ctx, cat1))

	cat2 := &categories.Category{
		ID:        "cat-dup-2",
		Name:      "Category 2",
		Slug:      "duplicate-slug",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	err := store.Create(ctx, cat2)
	assertError(t, err)
}

func TestCategoriesStore_GetByID(t *testing.T) {
	db := setupTestDB(t)
	store := NewCategoriesStore(db)
	ctx := context.Background()

	cat := &categories.Category{
		ID:        "cat-get",
		Name:      "Get Category",
		Slug:      "get-category",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.Create(ctx, cat))

	got, err := store.GetByID(ctx, cat.ID)
	assertNoError(t, err)
	assertEqual(t, "ID", got.ID, cat.ID)
}

func TestCategoriesStore_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	store := NewCategoriesStore(db)
	ctx := context.Background()

	got, err := store.GetByID(ctx, "nonexistent")
	assertNoError(t, err)
	if got != nil {
		t.Error("expected nil for non-existent category")
	}
}

func TestCategoriesStore_GetBySlug(t *testing.T) {
	db := setupTestDB(t)
	store := NewCategoriesStore(db)
	ctx := context.Background()

	cat := &categories.Category{
		ID:        "cat-slug",
		Name:      "Slug Category",
		Slug:      "my-unique-slug",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.Create(ctx, cat))

	got, err := store.GetBySlug(ctx, "my-unique-slug")
	assertNoError(t, err)
	assertEqual(t, "ID", got.ID, cat.ID)
}

func TestCategoriesStore_List(t *testing.T) {
	db := setupTestDB(t)
	store := NewCategoriesStore(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		cat := &categories.Category{
			ID:        "cat-list-" + string(rune('a'+i)),
			Name:      "List Category " + string(rune('A'+i)),
			Slug:      "list-cat-" + string(rune('a'+i)),
			CreatedAt: testTime,
			UpdatedAt: testTime,
		}
		assertNoError(t, store.Create(ctx, cat))
	}

	list, total, err := store.List(ctx, &categories.ListIn{Limit: 10})
	assertNoError(t, err)
	assertEqual(t, "total", total, 5)
	assertLen(t, list, 5)
}

func TestCategoriesStore_List_FilterByParent(t *testing.T) {
	db := setupTestDB(t)
	store := NewCategoriesStore(db)
	ctx := context.Background()

	// Create parent
	parent := &categories.Category{
		ID:        "cat-parent-list",
		Name:      "Parent List",
		Slug:      "parent-list",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.Create(ctx, parent))

	// Create children
	for i := 0; i < 3; i++ {
		child := &categories.Category{
			ID:        "cat-child-list-" + string(rune('a'+i)),
			ParentID:  parent.ID,
			Name:      "Child " + string(rune('A'+i)),
			Slug:      "child-list-" + string(rune('a'+i)),
			CreatedAt: testTime,
			UpdatedAt: testTime,
		}
		assertNoError(t, store.Create(ctx, child))
	}

	// Create orphan
	orphan := &categories.Category{
		ID:        "cat-orphan",
		Name:      "Orphan",
		Slug:      "orphan",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.Create(ctx, orphan))

	list, total, err := store.List(ctx, &categories.ListIn{ParentID: parent.ID, Limit: 10})
	assertNoError(t, err)
	assertEqual(t, "total", total, 3)
	assertLen(t, list, 3)
}

func TestCategoriesStore_List_Search(t *testing.T) {
	db := setupTestDB(t)
	store := NewCategoriesStore(db)
	ctx := context.Background()

	names := []string{"Technology", "Science", "Tech News"}
	for i, name := range names {
		cat := &categories.Category{
			ID:        "cat-search-" + string(rune('a'+i)),
			Name:      name,
			Slug:      "search-" + string(rune('a'+i)),
			CreatedAt: testTime,
			UpdatedAt: testTime,
		}
		assertNoError(t, store.Create(ctx, cat))
	}

	list, total, err := store.List(ctx, &categories.ListIn{Search: "Tech", Limit: 10})
	assertNoError(t, err)
	assertEqual(t, "total", total, 2)
	assertLen(t, list, 2)
}

func TestCategoriesStore_GetTree(t *testing.T) {
	db := setupTestDB(t)
	store := NewCategoriesStore(db)
	ctx := context.Background()

	// Create categories with different sort orders
	for i := 0; i < 3; i++ {
		cat := &categories.Category{
			ID:        "cat-tree-" + string(rune('a'+i)),
			Name:      "Tree Category " + string(rune('C'-i)), // C, B, A
			Slug:      "tree-cat-" + string(rune('a'+i)),
			SortOrder: i,
			CreatedAt: testTime,
			UpdatedAt: testTime,
		}
		assertNoError(t, store.Create(ctx, cat))
	}

	list, err := store.GetTree(ctx)
	assertNoError(t, err)
	assertLen(t, list, 3)
	// Should be ordered by sort_order, then name
	assertEqual(t, "Name[0]", list[0].Name, "Tree Category C")
}

func TestCategoriesStore_GetTree_Ordering(t *testing.T) {
	db := setupTestDB(t)
	store := NewCategoriesStore(db)
	ctx := context.Background()

	// Same sort order, should then order by name
	catsData := []struct {
		name      string
		sortOrder int
	}{
		{"Zebra", 0},
		{"Apple", 0},
		{"Banana", 0},
	}
	for i, c := range catsData {
		cat := &categories.Category{
			ID:        "cat-order-" + string(rune('a'+i)),
			Name:      c.name,
			Slug:      "order-" + string(rune('a'+i)),
			SortOrder: c.sortOrder,
			CreatedAt: testTime,
			UpdatedAt: testTime,
		}
		assertNoError(t, store.Create(ctx, cat))
	}

	list, _ := store.GetTree(ctx)
	// Same sort_order, so ordered by name
	assertEqual(t, "Name[0]", list[0].Name, "Apple")
	assertEqual(t, "Name[1]", list[1].Name, "Banana")
	assertEqual(t, "Name[2]", list[2].Name, "Zebra")
}

func TestCategoriesStore_Update(t *testing.T) {
	db := setupTestDB(t)
	store := NewCategoriesStore(db)
	ctx := context.Background()

	cat := &categories.Category{
		ID:        "cat-update",
		Name:      "Original Name",
		Slug:      "update-cat",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.Create(ctx, cat))

	err := store.Update(ctx, cat.ID, &categories.UpdateIn{
		Name:        ptr("Updated Name"),
		Description: ptr("New description"),
	})
	assertNoError(t, err)

	got, _ := store.GetByID(ctx, cat.ID)
	assertEqual(t, "Name", got.Name, "Updated Name")
	assertEqual(t, "Description", got.Description, "New description")
}

func TestCategoriesStore_Update_ChangeParent(t *testing.T) {
	db := setupTestDB(t)
	store := NewCategoriesStore(db)
	ctx := context.Background()

	// Create two parents
	parent1 := &categories.Category{
		ID:        "cat-parent-1",
		Name:      "Parent 1",
		Slug:      "parent-1",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	parent2 := &categories.Category{
		ID:        "cat-parent-2",
		Name:      "Parent 2",
		Slug:      "parent-2",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.Create(ctx, parent1))
	assertNoError(t, store.Create(ctx, parent2))

	// Create child under parent1
	child := &categories.Category{
		ID:        "cat-movable",
		ParentID:  parent1.ID,
		Name:      "Movable Child",
		Slug:      "movable-child",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.Create(ctx, child))

	// Move to parent2
	err := store.Update(ctx, child.ID, &categories.UpdateIn{
		ParentID: ptr(parent2.ID),
	})
	assertNoError(t, err)

	got, _ := store.GetByID(ctx, child.ID)
	assertEqual(t, "ParentID", got.ParentID, parent2.ID)
}

func TestCategoriesStore_Delete(t *testing.T) {
	db := setupTestDB(t)
	store := NewCategoriesStore(db)
	ctx := context.Background()

	cat := &categories.Category{
		ID:        "cat-delete",
		Name:      "Delete Category",
		Slug:      "delete-cat",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.Create(ctx, cat))

	err := store.Delete(ctx, cat.ID)
	assertNoError(t, err)

	got, _ := store.GetByID(ctx, cat.ID)
	if got != nil {
		t.Error("expected category to be deleted")
	}
}

func TestCategoriesStore_IncrementPostCount(t *testing.T) {
	db := setupTestDB(t)
	store := NewCategoriesStore(db)
	ctx := context.Background()

	cat := &categories.Category{
		ID:        "cat-inc",
		Name:      "Increment Category",
		Slug:      "inc-cat",
		PostCount: 5,
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.Create(ctx, cat))

	err := store.IncrementPostCount(ctx, cat.ID)
	assertNoError(t, err)

	got, _ := store.GetByID(ctx, cat.ID)
	assertEqual(t, "PostCount", got.PostCount, 6)
}

func TestCategoriesStore_DecrementPostCount(t *testing.T) {
	db := setupTestDB(t)
	store := NewCategoriesStore(db)
	ctx := context.Background()

	cat := &categories.Category{
		ID:        "cat-dec",
		Name:      "Decrement Category",
		Slug:      "dec-cat",
		PostCount: 5,
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.Create(ctx, cat))

	err := store.DecrementPostCount(ctx, cat.ID)
	assertNoError(t, err)

	got, _ := store.GetByID(ctx, cat.ID)
	assertEqual(t, "PostCount", got.PostCount, 4)
}

func TestCategoriesStore_DecrementPostCount_Floor(t *testing.T) {
	db := setupTestDB(t)
	store := NewCategoriesStore(db)
	ctx := context.Background()

	cat := &categories.Category{
		ID:        "cat-floor",
		Name:      "Floor Category",
		Slug:      "floor-cat",
		PostCount: 0,
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.Create(ctx, cat))

	// Try to decrement below 0
	err := store.DecrementPostCount(ctx, cat.ID)
	assertNoError(t, err)

	got, _ := store.GetByID(ctx, cat.ID)
	assertEqual(t, "PostCount", got.PostCount, 0) // Should not go below 0
}
