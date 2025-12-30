package duckdb

import (
	"context"
	"testing"
	"time"
)

func setupTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}
	t.Cleanup(func() {
		store.Close()
	})
	return store
}

func createTestCollection(t *testing.T, store *Store, name string) {
	t.Helper()
	ctx := context.Background()
	err := store.CreateCollection(ctx, name, []ColumnDef{
		{Name: "title", Type: "VARCHAR(255)", Nullable: true},
		{Name: "slug", Type: "VARCHAR(255)", Nullable: true, Unique: true},
		{Name: "content", Type: "TEXT", Nullable: true},
		{Name: "status", Type: "VARCHAR(50)", Nullable: true},
		{Name: "age", Type: "INTEGER", Nullable: true},
		{Name: "price", Type: "DOUBLE", Nullable: true},
		{Name: "active", Type: "BOOLEAN", Nullable: true},
		{Name: "featured_image", Type: "VARCHAR(26)", Nullable: true},
		{Name: "tags", Type: "TEXT", Nullable: true},
	})
	if err != nil {
		t.Fatalf("failed to create test collection: %v", err)
	}
}

func TestCollectionsStore_Create(t *testing.T) {
	store := setupTestStore(t)
	createTestCollection(t, store, "articles")
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		data := map[string]any{
			"title":   "Test Article",
			"slug":    "test-article",
			"content": "This is test content",
			"status":  "draft",
		}

		doc, err := store.Collections.Create(ctx, "articles", data)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		if doc.ID == "" {
			t.Error("Expected ID to be set")
		}
		if doc.CreatedAt.IsZero() {
			t.Error("Expected CreatedAt to be set")
		}
		if doc.UpdatedAt.IsZero() {
			t.Error("Expected UpdatedAt to be set")
		}
		if doc.Data["title"] != "Test Article" {
			t.Errorf("Expected title to be 'Test Article', got %v", doc.Data["title"])
		}
	})

	t.Run("WithAllFieldTypes", func(t *testing.T) {
		data := map[string]any{
			"title":   "All Types Article",
			"slug":    "all-types",
			"age":     25,
			"price":   99.99,
			"active":  true,
			"tags":    []string{"go", "test"},
			"content": "Content here",
		}

		doc, err := store.Collections.Create(ctx, "articles", data)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		if doc.ID == "" {
			t.Error("Expected ID to be set")
		}
	})

	t.Run("WithSnakeCaseConversion", func(t *testing.T) {
		data := map[string]any{
			"title":         "Snake Case Test",
			"slug":          "snake-case-test",
			"featuredImage": "img123",
		}

		doc, err := store.Collections.Create(ctx, "articles", data)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		if doc.ID == "" {
			t.Error("Expected ID to be set")
		}
	})
}

func TestCollectionsStore_FindByID(t *testing.T) {
	store := setupTestStore(t)
	createTestCollection(t, store, "articles")
	ctx := context.Background()

	t.Run("Exists", func(t *testing.T) {
		data := map[string]any{
			"title":  "Find Me",
			"slug":   "find-me",
			"status": "published",
		}

		created, err := store.Collections.Create(ctx, "articles", data)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		found, err := store.Collections.FindByID(ctx, "articles", created.ID)
		if err != nil {
			t.Fatalf("FindByID failed: %v", err)
		}

		if found == nil {
			t.Fatal("Expected document to be found")
		}
		if found.ID != created.ID {
			t.Errorf("Expected ID %s, got %s", created.ID, found.ID)
		}
	})

	t.Run("NotExists", func(t *testing.T) {
		found, err := store.Collections.FindByID(ctx, "articles", "nonexistent123456789012345")
		if err != nil {
			t.Fatalf("FindByID failed: %v", err)
		}
		if found != nil {
			t.Error("Expected nil for non-existent document")
		}
	})
}

func TestCollectionsStore_Find(t *testing.T) {
	store := setupTestStore(t)
	createTestCollection(t, store, "posts")
	ctx := context.Background()

	// Create test data
	for i := 0; i < 25; i++ {
		status := "draft"
		if i%2 == 0 {
			status = "published"
		}
		_, err := store.Collections.Create(ctx, "posts", map[string]any{
			"title":  "Post " + string(rune('A'+i%26)),
			"slug":   "post-" + string(rune('a'+i%26)) + "-" + time.Now().Format("150405.000000"),
			"status": status,
			"age":    i + 10,
		})
		if err != nil {
			t.Fatalf("Failed to create test data: %v", err)
		}
	}

	t.Run("AllDocuments", func(t *testing.T) {
		result, err := store.Collections.Find(ctx, "posts", nil)
		if err != nil {
			t.Fatalf("Find failed: %v", err)
		}

		if result.TotalDocs != 25 {
			t.Errorf("Expected TotalDocs=25, got %d", result.TotalDocs)
		}
	})

	t.Run("DefaultPagination", func(t *testing.T) {
		result, err := store.Collections.Find(ctx, "posts", &FindOptions{})
		if err != nil {
			t.Fatalf("Find failed: %v", err)
		}

		if result.Limit != 10 {
			t.Errorf("Expected Limit=10, got %d", result.Limit)
		}
		if result.Page != 1 {
			t.Errorf("Expected Page=1, got %d", result.Page)
		}
		if len(result.Docs) != 10 {
			t.Errorf("Expected 10 docs, got %d", len(result.Docs))
		}
	})

	t.Run("CustomPagination", func(t *testing.T) {
		result, err := store.Collections.Find(ctx, "posts", &FindOptions{
			Limit: 5,
			Page:  2,
		})
		if err != nil {
			t.Fatalf("Find failed: %v", err)
		}

		if result.Limit != 5 {
			t.Errorf("Expected Limit=5, got %d", result.Limit)
		}
		if result.Page != 2 {
			t.Errorf("Expected Page=2, got %d", result.Page)
		}
		if len(result.Docs) != 5 {
			t.Errorf("Expected 5 docs, got %d", len(result.Docs))
		}
		if result.PagingCounter != 6 {
			t.Errorf("Expected PagingCounter=6, got %d", result.PagingCounter)
		}
	})

	t.Run("TotalPages", func(t *testing.T) {
		result, err := store.Collections.Find(ctx, "posts", &FindOptions{
			Limit: 10,
		})
		if err != nil {
			t.Fatalf("Find failed: %v", err)
		}

		if result.TotalPages != 3 {
			t.Errorf("Expected TotalPages=3, got %d", result.TotalPages)
		}
	})

	t.Run("HasNextPage", func(t *testing.T) {
		result, err := store.Collections.Find(ctx, "posts", &FindOptions{
			Limit: 10,
			Page:  1,
		})
		if err != nil {
			t.Fatalf("Find failed: %v", err)
		}

		if !result.HasNextPage {
			t.Error("Expected HasNextPage=true")
		}
		if result.NextPage == nil || *result.NextPage != 2 {
			t.Errorf("Expected NextPage=2, got %v", result.NextPage)
		}
	})

	t.Run("HasPrevPage", func(t *testing.T) {
		result, err := store.Collections.Find(ctx, "posts", &FindOptions{
			Limit: 10,
			Page:  2,
		})
		if err != nil {
			t.Fatalf("Find failed: %v", err)
		}

		if !result.HasPrevPage {
			t.Error("Expected HasPrevPage=true")
		}
		if result.PrevPage == nil || *result.PrevPage != 1 {
			t.Errorf("Expected PrevPage=1, got %v", result.PrevPage)
		}
	})

	t.Run("LastPage", func(t *testing.T) {
		result, err := store.Collections.Find(ctx, "posts", &FindOptions{
			Limit: 10,
			Page:  3,
		})
		if err != nil {
			t.Fatalf("Find failed: %v", err)
		}

		if result.HasNextPage {
			t.Error("Expected HasNextPage=false on last page")
		}
		if result.NextPage != nil {
			t.Errorf("Expected NextPage=nil, got %v", result.NextPage)
		}
	})
}

func TestCollectionsStore_FindWithWhere(t *testing.T) {
	store := setupTestStore(t)
	createTestCollection(t, store, "items")
	ctx := context.Background()

	// Create test data
	testData := []map[string]any{
		{"title": "Item A", "slug": "item-a", "status": "draft", "age": 15, "price": 50.0},
		{"title": "Item B", "slug": "item-b", "status": "published", "age": 25, "price": 100.0},
		{"title": "Item C", "slug": "item-c", "status": "published", "age": 35, "price": 150.0},
		{"title": "Hello World", "slug": "hello-world", "status": "archived", "age": 20, "price": 75.0},
		{"title": "Test Item", "slug": "test-item", "status": "draft", "age": 18, "price": 80.0},
	}

	for _, data := range testData {
		_, err := store.Collections.Create(ctx, "items", data)
		if err != nil {
			t.Fatalf("Failed to create test data: %v", err)
		}
	}

	t.Run("WhereEquals", func(t *testing.T) {
		result, err := store.Collections.Find(ctx, "items", &FindOptions{
			Where: map[string]any{"status": "published"},
		})
		if err != nil {
			t.Fatalf("Find failed: %v", err)
		}

		if result.TotalDocs != 2 {
			t.Errorf("Expected 2 published docs, got %d", result.TotalDocs)
		}
	})

	t.Run("WhereNotEquals", func(t *testing.T) {
		result, err := store.Collections.Find(ctx, "items", &FindOptions{
			Where: map[string]any{"status": map[string]any{"not_equals": "draft"}},
		})
		if err != nil {
			t.Fatalf("Find failed: %v", err)
		}

		if result.TotalDocs != 3 {
			t.Errorf("Expected 3 non-draft docs, got %d", result.TotalDocs)
		}
	})

	t.Run("WhereGreaterThan", func(t *testing.T) {
		result, err := store.Collections.Find(ctx, "items", &FindOptions{
			Where: map[string]any{"age": map[string]any{"greater_than": 18}},
		})
		if err != nil {
			t.Fatalf("Find failed: %v", err)
		}

		if result.TotalDocs != 3 {
			t.Errorf("Expected 3 docs with age > 18, got %d", result.TotalDocs)
		}
	})

	t.Run("WhereGreaterThanEqual", func(t *testing.T) {
		result, err := store.Collections.Find(ctx, "items", &FindOptions{
			Where: map[string]any{"age": map[string]any{"greater_than_equal": 18}},
		})
		if err != nil {
			t.Fatalf("Find failed: %v", err)
		}

		if result.TotalDocs != 4 {
			t.Errorf("Expected 4 docs with age >= 18, got %d", result.TotalDocs)
		}
	})

	t.Run("WhereLessThan", func(t *testing.T) {
		result, err := store.Collections.Find(ctx, "items", &FindOptions{
			Where: map[string]any{"price": map[string]any{"less_than": 100}},
		})
		if err != nil {
			t.Fatalf("Find failed: %v", err)
		}

		if result.TotalDocs != 3 {
			t.Errorf("Expected 3 docs with price < 100, got %d", result.TotalDocs)
		}
	})

	t.Run("WhereLessThanEqual", func(t *testing.T) {
		result, err := store.Collections.Find(ctx, "items", &FindOptions{
			Where: map[string]any{"price": map[string]any{"less_than_equal": 100}},
		})
		if err != nil {
			t.Fatalf("Find failed: %v", err)
		}

		if result.TotalDocs != 4 {
			t.Errorf("Expected 4 docs with price <= 100, got %d", result.TotalDocs)
		}
	})

	t.Run("WhereLike", func(t *testing.T) {
		result, err := store.Collections.Find(ctx, "items", &FindOptions{
			Where: map[string]any{"title": map[string]any{"like": "HELLO"}},
		})
		if err != nil {
			t.Fatalf("Find failed: %v", err)
		}

		if result.TotalDocs != 1 {
			t.Errorf("Expected 1 doc with title like 'HELLO', got %d", result.TotalDocs)
		}
	})

	t.Run("WhereContains", func(t *testing.T) {
		result, err := store.Collections.Find(ctx, "items", &FindOptions{
			Where: map[string]any{"title": map[string]any{"contains": "Item"}},
		})
		if err != nil {
			t.Fatalf("Find failed: %v", err)
		}

		if result.TotalDocs != 4 {
			t.Errorf("Expected 4 docs containing 'Item', got %d", result.TotalDocs)
		}
	})

	t.Run("WhereIn", func(t *testing.T) {
		result, err := store.Collections.Find(ctx, "items", &FindOptions{
			Where: map[string]any{"status": map[string]any{"in": []any{"draft", "published"}}},
		})
		if err != nil {
			t.Fatalf("Find failed: %v", err)
		}

		if result.TotalDocs != 4 {
			t.Errorf("Expected 4 docs in draft/published, got %d", result.TotalDocs)
		}
	})

	t.Run("WhereNotIn", func(t *testing.T) {
		result, err := store.Collections.Find(ctx, "items", &FindOptions{
			Where: map[string]any{"status": map[string]any{"not_in": []any{"archived"}}},
		})
		if err != nil {
			t.Fatalf("Find failed: %v", err)
		}

		if result.TotalDocs != 4 {
			t.Errorf("Expected 4 docs not archived, got %d", result.TotalDocs)
		}
	})
}

func TestCollectionsStore_FindWithSort(t *testing.T) {
	store := setupTestStore(t)
	createTestCollection(t, store, "sorted")
	ctx := context.Background()

	// Create test data
	titles := []string{"Charlie", "Alpha", "Bravo"}
	for _, title := range titles {
		_, err := store.Collections.Create(ctx, "sorted", map[string]any{
			"title": title,
			"slug":  title,
		})
		if err != nil {
			t.Fatalf("Failed to create test data: %v", err)
		}
		time.Sleep(10 * time.Millisecond) // Ensure different created_at
	}

	t.Run("SortAscending", func(t *testing.T) {
		result, err := store.Collections.Find(ctx, "sorted", &FindOptions{
			Sort: []SortField{{Field: "title", Desc: false}},
		})
		if err != nil {
			t.Fatalf("Find failed: %v", err)
		}

		if len(result.Docs) != 3 {
			t.Fatalf("Expected 3 docs, got %d", len(result.Docs))
		}

		expected := []string{"Alpha", "Bravo", "Charlie"}
		for i, doc := range result.Docs {
			if doc.Data["title"] != expected[i] {
				t.Errorf("At index %d: expected %s, got %v", i, expected[i], doc.Data["title"])
			}
		}
	})

	t.Run("SortDescending", func(t *testing.T) {
		result, err := store.Collections.Find(ctx, "sorted", &FindOptions{
			Sort: []SortField{{Field: "title", Desc: true}},
		})
		if err != nil {
			t.Fatalf("Find failed: %v", err)
		}

		if len(result.Docs) != 3 {
			t.Fatalf("Expected 3 docs, got %d", len(result.Docs))
		}

		expected := []string{"Charlie", "Bravo", "Alpha"}
		for i, doc := range result.Docs {
			if doc.Data["title"] != expected[i] {
				t.Errorf("At index %d: expected %s, got %v", i, expected[i], doc.Data["title"])
			}
		}
	})
}

func TestCollectionsStore_Count(t *testing.T) {
	store := setupTestStore(t)
	createTestCollection(t, store, "countable")
	ctx := context.Background()

	// Create test data
	for i := 0; i < 5; i++ {
		status := "draft"
		if i%2 == 0 {
			status = "published"
		}
		_, err := store.Collections.Create(ctx, "countable", map[string]any{
			"title":  "Count Item",
			"slug":   "count-" + time.Now().Format("150405.000000"),
			"status": status,
		})
		if err != nil {
			t.Fatalf("Failed to create test data: %v", err)
		}
	}

	t.Run("All", func(t *testing.T) {
		count, err := store.Collections.Count(ctx, "countable", nil)
		if err != nil {
			t.Fatalf("Count failed: %v", err)
		}

		if count != 5 {
			t.Errorf("Expected count=5, got %d", count)
		}
	})

	t.Run("WithWhere", func(t *testing.T) {
		count, err := store.Collections.Count(ctx, "countable", map[string]any{"status": "published"})
		if err != nil {
			t.Fatalf("Count failed: %v", err)
		}

		if count != 3 {
			t.Errorf("Expected count=3 for published, got %d", count)
		}
	})
}

func TestCollectionsStore_UpdateByID(t *testing.T) {
	store := setupTestStore(t)
	createTestCollection(t, store, "updateable")
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		doc, _ := store.Collections.Create(ctx, "updateable", map[string]any{
			"title":  "Original Title",
			"slug":   "original",
			"status": "draft",
		})

		updated, err := store.Collections.UpdateByID(ctx, "updateable", doc.ID, map[string]any{
			"title":  "Updated Title",
			"status": "published",
		})
		if err != nil {
			t.Fatalf("UpdateByID failed: %v", err)
		}

		if updated == nil {
			t.Fatal("Expected updated document")
		}
		if updated.Data["title"] != "Updated Title" {
			t.Errorf("Expected title='Updated Title', got %v", updated.Data["title"])
		}
	})

	t.Run("PartialUpdate", func(t *testing.T) {
		doc, _ := store.Collections.Create(ctx, "updateable", map[string]any{
			"title":  "Keep This",
			"slug":   "partial",
			"status": "draft",
		})

		updated, err := store.Collections.UpdateByID(ctx, "updateable", doc.ID, map[string]any{
			"status": "published",
		})
		if err != nil {
			t.Fatalf("UpdateByID failed: %v", err)
		}

		if updated.Data["title"] != "Keep This" {
			t.Errorf("Expected title to remain 'Keep This', got %v", updated.Data["title"])
		}
		if updated.Data["status"] != "published" {
			t.Errorf("Expected status='published', got %v", updated.Data["status"])
		}
	})

	t.Run("UpdatedAt", func(t *testing.T) {
		doc, _ := store.Collections.Create(ctx, "updateable", map[string]any{
			"title": "Timestamp Test",
			"slug":  "timestamp",
		})

		time.Sleep(10 * time.Millisecond)

		updated, err := store.Collections.UpdateByID(ctx, "updateable", doc.ID, map[string]any{
			"title": "Updated",
		})
		if err != nil {
			t.Fatalf("UpdateByID failed: %v", err)
		}

		if !updated.UpdatedAt.After(doc.UpdatedAt) {
			t.Error("Expected UpdatedAt to be updated")
		}
	})

	t.Run("NotExists", func(t *testing.T) {
		updated, err := store.Collections.UpdateByID(ctx, "updateable", "nonexistent12345678901234", map[string]any{
			"title": "Should Not Work",
		})
		if err != nil {
			t.Fatalf("UpdateByID failed: %v", err)
		}

		if updated != nil {
			t.Error("Expected nil for non-existent document")
		}
	})
}

func TestCollectionsStore_Update(t *testing.T) {
	store := setupTestStore(t)
	createTestCollection(t, store, "bulkupdate")
	ctx := context.Background()

	// Create test data
	for i := 0; i < 5; i++ {
		_, err := store.Collections.Create(ctx, "bulkupdate", map[string]any{
			"title":  "Bulk Item",
			"slug":   "bulk-" + time.Now().Format("150405.000000"),
			"status": "draft",
		})
		if err != nil {
			t.Fatalf("Failed to create test data: %v", err)
		}
	}

	t.Run("MultipleDocuments", func(t *testing.T) {
		affected, err := store.Collections.Update(ctx, "bulkupdate",
			map[string]any{"status": "draft"},
			map[string]any{"status": "published"},
		)
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		if affected != 5 {
			t.Errorf("Expected 5 affected rows, got %d", affected)
		}

		count, _ := store.Collections.Count(ctx, "bulkupdate", map[string]any{"status": "published"})
		if count != 5 {
			t.Errorf("Expected 5 published docs, got %d", count)
		}
	})
}

func TestCollectionsStore_DeleteByID(t *testing.T) {
	store := setupTestStore(t)
	createTestCollection(t, store, "deleteable")
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		doc, _ := store.Collections.Create(ctx, "deleteable", map[string]any{
			"title": "Delete Me",
			"slug":  "delete-me",
		})

		deleted, err := store.Collections.DeleteByID(ctx, "deleteable", doc.ID)
		if err != nil {
			t.Fatalf("DeleteByID failed: %v", err)
		}

		if !deleted {
			t.Error("Expected deleted=true")
		}
	})

	t.Run("NotExists", func(t *testing.T) {
		deleted, err := store.Collections.DeleteByID(ctx, "deleteable", "nonexistent12345678901234")
		if err != nil {
			t.Fatalf("DeleteByID failed: %v", err)
		}

		if deleted {
			t.Error("Expected deleted=false for non-existent")
		}
	})

	t.Run("Verify", func(t *testing.T) {
		doc, _ := store.Collections.Create(ctx, "deleteable", map[string]any{
			"title": "Verify Delete",
			"slug":  "verify-delete",
		})

		store.Collections.DeleteByID(ctx, "deleteable", doc.ID)

		found, _ := store.Collections.FindByID(ctx, "deleteable", doc.ID)
		if found != nil {
			t.Error("Expected document to be deleted")
		}
	})
}

func TestCollectionsStore_Delete(t *testing.T) {
	store := setupTestStore(t)
	createTestCollection(t, store, "bulkdelete")
	ctx := context.Background()

	// Create test data
	for i := 0; i < 5; i++ {
		status := "draft"
		if i < 2 {
			status = "archived"
		}
		_, err := store.Collections.Create(ctx, "bulkdelete", map[string]any{
			"title":  "Bulk Delete",
			"slug":   "bulkdel-" + time.Now().Format("150405.000000"),
			"status": status,
		})
		if err != nil {
			t.Fatalf("Failed to create test data: %v", err)
		}
	}

	t.Run("WithWhere", func(t *testing.T) {
		affected, err := store.Collections.Delete(ctx, "bulkdelete", map[string]any{"status": "archived"})
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		if affected != 2 {
			t.Errorf("Expected 2 affected rows, got %d", affected)
		}

		count, _ := store.Collections.Count(ctx, "bulkdelete", nil)
		if count != 3 {
			t.Errorf("Expected 3 remaining docs, got %d", count)
		}
	})
}

func TestCollectionsStore_EmptyCollection(t *testing.T) {
	store := setupTestStore(t)
	createTestCollection(t, store, "empty")
	ctx := context.Background()

	t.Run("FindEmpty", func(t *testing.T) {
		result, err := store.Collections.Find(ctx, "empty", nil)
		if err != nil {
			t.Fatalf("Find failed: %v", err)
		}

		if result.TotalDocs != 0 {
			t.Errorf("Expected TotalDocs=0, got %d", result.TotalDocs)
		}
		if len(result.Docs) != 0 {
			t.Errorf("Expected empty docs slice, got %d docs", len(result.Docs))
		}
	})

	t.Run("CountEmpty", func(t *testing.T) {
		count, err := store.Collections.Count(ctx, "empty", nil)
		if err != nil {
			t.Fatalf("Count failed: %v", err)
		}

		if count != 0 {
			t.Errorf("Expected count=0, got %d", count)
		}
	})
}
