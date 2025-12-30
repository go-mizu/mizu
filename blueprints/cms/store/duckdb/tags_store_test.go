package duckdb

import (
	"context"
	"testing"

	"github.com/go-mizu/blueprints/cms/feature/tags"
)

func TestTagsStore_Create(t *testing.T) {
	db := setupTestDB(t)
	store := NewTagsStore(db)
	ctx := context.Background()

	tag := &tags.Tag{
		ID:              "tag-001",
		Name:            "Go",
		Slug:            "go",
		Description:     "Go programming language",
		FeaturedImageID: "img-001",
		Meta:            `{"color":"blue"}`,
		PostCount:       0,
		CreatedAt:       testTime,
		UpdatedAt:       testTime,
	}

	err := store.Create(ctx, tag)
	assertNoError(t, err)

	got, err := store.GetByID(ctx, tag.ID)
	assertNoError(t, err)
	assertEqual(t, "ID", got.ID, tag.ID)
	assertEqual(t, "Name", got.Name, tag.Name)
	assertEqual(t, "Slug", got.Slug, tag.Slug)
	assertEqual(t, "Description", got.Description, tag.Description)
}

func TestTagsStore_Create_DuplicateSlug(t *testing.T) {
	db := setupTestDB(t)
	store := NewTagsStore(db)
	ctx := context.Background()

	tag1 := &tags.Tag{
		ID:        "tag-dup-1",
		Name:      "Tag 1",
		Slug:      "duplicate-slug",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.Create(ctx, tag1))

	tag2 := &tags.Tag{
		ID:        "tag-dup-2",
		Name:      "Tag 2",
		Slug:      "duplicate-slug",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	err := store.Create(ctx, tag2)
	assertError(t, err)
}

func TestTagsStore_GetByID(t *testing.T) {
	db := setupTestDB(t)
	store := NewTagsStore(db)
	ctx := context.Background()

	tag := &tags.Tag{
		ID:        "tag-get",
		Name:      "Get Tag",
		Slug:      "get-tag",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.Create(ctx, tag))

	got, err := store.GetByID(ctx, tag.ID)
	assertNoError(t, err)
	assertEqual(t, "ID", got.ID, tag.ID)
}

func TestTagsStore_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	store := NewTagsStore(db)
	ctx := context.Background()

	got, err := store.GetByID(ctx, "nonexistent")
	assertNoError(t, err)
	if got != nil {
		t.Error("expected nil for non-existent tag")
	}
}

func TestTagsStore_GetBySlug(t *testing.T) {
	db := setupTestDB(t)
	store := NewTagsStore(db)
	ctx := context.Background()

	tag := &tags.Tag{
		ID:        "tag-slug",
		Name:      "Slug Tag",
		Slug:      "my-unique-tag-slug",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.Create(ctx, tag))

	got, err := store.GetBySlug(ctx, "my-unique-tag-slug")
	assertNoError(t, err)
	assertEqual(t, "ID", got.ID, tag.ID)
}

func TestTagsStore_GetByIDs(t *testing.T) {
	db := setupTestDB(t)
	store := NewTagsStore(db)
	ctx := context.Background()

	// Create multiple tags
	for i := 0; i < 3; i++ {
		tag := &tags.Tag{
			ID:        "tag-multi-" + string(rune('a'+i)),
			Name:      "Multi Tag " + string(rune('A'+i)),
			Slug:      "multi-" + string(rune('a'+i)),
			CreatedAt: testTime,
			UpdatedAt: testTime,
		}
		assertNoError(t, store.Create(ctx, tag))
	}

	got, err := store.GetByIDs(ctx, []string{"tag-multi-a", "tag-multi-c"})
	assertNoError(t, err)
	assertLen(t, got, 2)
}

func TestTagsStore_GetByIDs_Empty(t *testing.T) {
	db := setupTestDB(t)
	store := NewTagsStore(db)
	ctx := context.Background()

	got, err := store.GetByIDs(ctx, []string{})
	assertNoError(t, err)
	if got != nil {
		t.Errorf("expected nil for empty IDs, got %v", got)
	}
}

func TestTagsStore_List(t *testing.T) {
	db := setupTestDB(t)
	store := NewTagsStore(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		tag := &tags.Tag{
			ID:        "tag-list-" + string(rune('a'+i)),
			Name:      "List Tag " + string(rune('A'+i)),
			Slug:      "list-tag-" + string(rune('a'+i)),
			CreatedAt: testTime,
			UpdatedAt: testTime,
		}
		assertNoError(t, store.Create(ctx, tag))
	}

	list, total, err := store.List(ctx, &tags.ListIn{Limit: 10})
	assertNoError(t, err)
	assertEqual(t, "total", total, 5)
	assertLen(t, list, 5)
}

func TestTagsStore_List_Search(t *testing.T) {
	db := setupTestDB(t)
	store := NewTagsStore(db)
	ctx := context.Background()

	names := []string{"Golang", "JavaScript", "Go Testing"}
	for i, name := range names {
		tag := &tags.Tag{
			ID:        "tag-search-" + string(rune('a'+i)),
			Name:      name,
			Slug:      "search-" + string(rune('a'+i)),
			CreatedAt: testTime,
			UpdatedAt: testTime,
		}
		assertNoError(t, store.Create(ctx, tag))
	}

	list, total, err := store.List(ctx, &tags.ListIn{Search: "Go", Limit: 10})
	assertNoError(t, err)
	assertEqual(t, "total", total, 2)
	assertLen(t, list, 2)
}

func TestTagsStore_List_OrderBy(t *testing.T) {
	db := setupTestDB(t)
	store := NewTagsStore(db)
	ctx := context.Background()

	tagsData := []struct {
		name      string
		postCount int
	}{
		{"Tag A", 10},
		{"Tag B", 5},
		{"Tag C", 20},
	}
	for i, td := range tagsData {
		tag := &tags.Tag{
			ID:        "tag-order-" + string(rune('a'+i)),
			Name:      td.name,
			Slug:      "order-" + string(rune('a'+i)),
			PostCount: td.postCount,
			CreatedAt: testTime,
			UpdatedAt: testTime,
		}
		assertNoError(t, store.Create(ctx, tag))
	}

	// Order by post_count DESC
	list, _, err := store.List(ctx, &tags.ListIn{OrderBy: "post_count", Order: "DESC", Limit: 10})
	assertNoError(t, err)
	assertEqual(t, "Name[0]", list[0].Name, "Tag C") // 20 posts
	assertEqual(t, "Name[1]", list[1].Name, "Tag A") // 10 posts
	assertEqual(t, "Name[2]", list[2].Name, "Tag B") // 5 posts
}

func TestTagsStore_Update(t *testing.T) {
	db := setupTestDB(t)
	store := NewTagsStore(db)
	ctx := context.Background()

	tag := &tags.Tag{
		ID:        "tag-update",
		Name:      "Original Name",
		Slug:      "update-tag",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.Create(ctx, tag))

	err := store.Update(ctx, tag.ID, &tags.UpdateIn{
		Name:        ptr("Updated Name"),
		Description: ptr("New description"),
	})
	assertNoError(t, err)

	got, _ := store.GetByID(ctx, tag.ID)
	assertEqual(t, "Name", got.Name, "Updated Name")
	assertEqual(t, "Description", got.Description, "New description")
}

func TestTagsStore_Update_PartialFields(t *testing.T) {
	db := setupTestDB(t)
	store := NewTagsStore(db)
	ctx := context.Background()

	tag := &tags.Tag{
		ID:          "tag-partial",
		Name:        "Original Name",
		Slug:        "partial-tag",
		Description: "Original Description",
		CreatedAt:   testTime,
		UpdatedAt:   testTime,
	}
	assertNoError(t, store.Create(ctx, tag))

	// Only update name
	err := store.Update(ctx, tag.ID, &tags.UpdateIn{
		Name: ptr("New Name"),
	})
	assertNoError(t, err)

	got, _ := store.GetByID(ctx, tag.ID)
	assertEqual(t, "Name", got.Name, "New Name")
	assertEqual(t, "Description", got.Description, "Original Description") // Unchanged
}

func TestTagsStore_Delete(t *testing.T) {
	db := setupTestDB(t)
	store := NewTagsStore(db)
	ctx := context.Background()

	tag := &tags.Tag{
		ID:        "tag-delete",
		Name:      "Delete Tag",
		Slug:      "delete-tag",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.Create(ctx, tag))

	err := store.Delete(ctx, tag.ID)
	assertNoError(t, err)

	got, _ := store.GetByID(ctx, tag.ID)
	if got != nil {
		t.Error("expected tag to be deleted")
	}
}

func TestTagsStore_IncrementPostCount(t *testing.T) {
	db := setupTestDB(t)
	store := NewTagsStore(db)
	ctx := context.Background()

	tag := &tags.Tag{
		ID:        "tag-inc",
		Name:      "Increment Tag",
		Slug:      "inc-tag",
		PostCount: 5,
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.Create(ctx, tag))

	err := store.IncrementPostCount(ctx, tag.ID)
	assertNoError(t, err)

	got, _ := store.GetByID(ctx, tag.ID)
	assertEqual(t, "PostCount", got.PostCount, 6)
}

func TestTagsStore_DecrementPostCount(t *testing.T) {
	db := setupTestDB(t)
	store := NewTagsStore(db)
	ctx := context.Background()

	tag := &tags.Tag{
		ID:        "tag-dec",
		Name:      "Decrement Tag",
		Slug:      "dec-tag",
		PostCount: 5,
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.Create(ctx, tag))

	err := store.DecrementPostCount(ctx, tag.ID)
	assertNoError(t, err)

	got, _ := store.GetByID(ctx, tag.ID)
	assertEqual(t, "PostCount", got.PostCount, 4)
}

func TestTagsStore_DecrementPostCount_Floor(t *testing.T) {
	db := setupTestDB(t)
	store := NewTagsStore(db)
	ctx := context.Background()

	tag := &tags.Tag{
		ID:        "tag-floor",
		Name:      "Floor Tag",
		Slug:      "floor-tag",
		PostCount: 0,
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.Create(ctx, tag))

	// Try to decrement below 0
	err := store.DecrementPostCount(ctx, tag.ID)
	assertNoError(t, err)

	got, _ := store.GetByID(ctx, tag.ID)
	assertEqual(t, "PostCount", got.PostCount, 0) // Should not go below 0
}
