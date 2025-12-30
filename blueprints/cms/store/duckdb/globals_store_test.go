package duckdb

import (
	"context"
	"testing"
	"time"
)

func TestGlobalsStore_Get(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	t.Run("Exists", func(t *testing.T) {
		// First create a global
		_, err := store.Globals.Update(ctx, "site-settings", map[string]any{
			"siteName":    "Test Site",
			"tagline":     "A test site",
			"contactEmail": "test@example.com",
		})
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		// Then get it
		global, err := store.Globals.Get(ctx, "site-settings")
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		if global == nil {
			t.Fatal("Expected global to be found")
		}
		if global.Slug != "site-settings" {
			t.Errorf("Expected slug='site-settings', got %s", global.Slug)
		}
		if global.Data["siteName"] != "Test Site" {
			t.Errorf("Expected siteName='Test Site', got %v", global.Data["siteName"])
		}
	})

	t.Run("NotExists", func(t *testing.T) {
		global, err := store.Globals.Get(ctx, "nonexistent-global")
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		if global != nil {
			t.Error("Expected nil for non-existent global")
		}
	})

	t.Run("DataDeserialization", func(t *testing.T) {
		// Create a global with complex nested data
		_, err := store.Globals.Update(ctx, "complex-settings", map[string]any{
			"nested": map[string]any{
				"level1": map[string]any{
					"value": "deep",
				},
			},
			"array": []any{"one", "two", "three"},
			"number": 42,
			"float":  3.14,
			"bool":   true,
		})
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		global, err := store.Globals.Get(ctx, "complex-settings")
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		if global.Data["number"] != float64(42) {
			t.Errorf("Expected number=42, got %v (type %T)", global.Data["number"], global.Data["number"])
		}
		if global.Data["bool"] != true {
			t.Errorf("Expected bool=true, got %v", global.Data["bool"])
		}

		arr, ok := global.Data["array"].([]any)
		if !ok {
			t.Fatalf("Expected array to be []any, got %T", global.Data["array"])
		}
		if len(arr) != 3 {
			t.Errorf("Expected array length=3, got %d", len(arr))
		}
	})
}

func TestGlobalsStore_Update(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	t.Run("CreateNew", func(t *testing.T) {
		global, err := store.Globals.Update(ctx, "new-global", map[string]any{
			"key": "value",
		})
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		if global.ID == "" {
			t.Error("Expected ID to be set")
		}
		if global.Slug != "new-global" {
			t.Errorf("Expected slug='new-global', got %s", global.Slug)
		}
		if global.CreatedAt.IsZero() {
			t.Error("Expected CreatedAt to be set")
		}
	})

	t.Run("UpdateExisting", func(t *testing.T) {
		// Create
		original, err := store.Globals.Update(ctx, "update-test", map[string]any{
			"version": 1,
		})
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		originalID := original.ID

		// Update
		updated, err := store.Globals.Update(ctx, "update-test", map[string]any{
			"version": 2,
		})
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		if updated.ID != originalID {
			t.Errorf("Expected ID to remain %s, got %s", originalID, updated.ID)
		}
		if updated.Data["version"] != float64(2) {
			t.Errorf("Expected version=2, got %v", updated.Data["version"])
		}
	})

	t.Run("PreservesCreatedAt", func(t *testing.T) {
		// Create
		original, err := store.Globals.Update(ctx, "preserve-created", map[string]any{
			"value": "initial",
		})
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		time.Sleep(10 * time.Millisecond)

		// Update
		updated, err := store.Globals.Update(ctx, "preserve-created", map[string]any{
			"value": "updated",
		})
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		if !updated.CreatedAt.Equal(original.CreatedAt) {
			t.Error("Expected CreatedAt to remain unchanged")
		}
	})

	t.Run("UpdatesTimestamp", func(t *testing.T) {
		// Create
		original, err := store.Globals.Update(ctx, "update-timestamp", map[string]any{
			"value": "initial",
		})
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		time.Sleep(10 * time.Millisecond)

		// Update
		updated, err := store.Globals.Update(ctx, "update-timestamp", map[string]any{
			"value": "updated",
		})
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		if !updated.UpdatedAt.After(original.UpdatedAt) {
			t.Error("Expected UpdatedAt to be updated")
		}
	})

	t.Run("ComplexData", func(t *testing.T) {
		global, err := store.Globals.Update(ctx, "complex-data", map[string]any{
			"settings": map[string]any{
				"theme": "dark",
				"notifications": map[string]any{
					"email":   true,
					"push":    false,
					"desktop": true,
				},
			},
			"features": []any{"feature1", "feature2"},
		})
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		// Verify by getting
		fetched, err := store.Globals.Get(ctx, "complex-data")
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		settings, ok := fetched.Data["settings"].(map[string]any)
		if !ok {
			t.Fatalf("Expected settings to be map, got %T", fetched.Data["settings"])
		}
		if settings["theme"] != "dark" {
			t.Errorf("Expected theme='dark', got %v", settings["theme"])
		}

		if global.ID == "" {
			t.Error("Expected ID to be set")
		}
	})
}

func TestGlobalsStore_List(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	t.Run("Empty", func(t *testing.T) {
		globals, err := store.Globals.List(ctx)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}

		if globals == nil {
			t.Error("Expected empty slice, got nil")
		}
	})

	t.Run("Multiple", func(t *testing.T) {
		// Create multiple globals
		slugs := []string{"charlie-global", "alpha-global", "bravo-global"}
		for _, slug := range slugs {
			_, err := store.Globals.Update(ctx, slug, map[string]any{"name": slug})
			if err != nil {
				t.Fatalf("Update failed for %s: %v", slug, err)
			}
		}

		globals, err := store.Globals.List(ctx)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}

		if len(globals) < 3 {
			t.Errorf("Expected at least 3 globals, got %d", len(globals))
		}
	})

	t.Run("OrderBySlug", func(t *testing.T) {
		// Create globals with specific slugs
		slugs := []string{"zzz-last", "aaa-first", "mmm-middle"}
		for _, slug := range slugs {
			_, err := store.Globals.Update(ctx, slug, map[string]any{"name": slug})
			if err != nil {
				t.Fatalf("Update failed: %v", err)
			}
		}

		globals, err := store.Globals.List(ctx)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}

		// Check ordering (should be alphabetical by slug)
		var foundFirst, foundLast bool
		var firstIdx, lastIdx int
		for i, g := range globals {
			if g.Slug == "aaa-first" {
				foundFirst = true
				firstIdx = i
			}
			if g.Slug == "zzz-last" {
				foundLast = true
				lastIdx = i
			}
		}

		if !foundFirst || !foundLast {
			t.Fatal("Expected to find aaa-first and zzz-last globals")
		}
		if firstIdx >= lastIdx {
			t.Error("Expected aaa-first to come before zzz-last")
		}
	})
}
