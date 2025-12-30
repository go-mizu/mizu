package duckdb

import (
	"context"
	"testing"
	"time"
)

func TestPreferencesStore_Get(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	t.Run("Exists", func(t *testing.T) {
		// First set a preference
		_, err := store.Preferences.Set(ctx, "user1", "theme", "dark")
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		pref, err := store.Preferences.Get(ctx, "user1", "theme")
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		if pref == nil {
			t.Fatal("Expected preference to be found")
		}
		if pref.Key != "theme" {
			t.Errorf("Expected Key='theme', got %s", pref.Key)
		}
		if pref.Value != "dark" {
			t.Errorf("Expected Value='dark', got %v", pref.Value)
		}
	})

	t.Run("NotExists", func(t *testing.T) {
		pref, err := store.Preferences.Get(ctx, "nonexistent-user", "nonexistent-key")
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		if pref != nil {
			t.Error("Expected nil for non-existent preference")
		}
	})

	t.Run("JSONValue", func(t *testing.T) {
		value := map[string]any{
			"sidebar":    "collapsed",
			"fontSize":   14,
			"showHidden": true,
		}
		_, err := store.Preferences.Set(ctx, "user2", "layout", value)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		pref, err := store.Preferences.Get(ctx, "user2", "layout")
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		if pref == nil {
			t.Fatal("Expected preference to be found")
		}

		prefMap, ok := pref.Value.(map[string]any)
		if !ok {
			t.Fatalf("Expected map value, got %T", pref.Value)
		}
		if prefMap["sidebar"] != "collapsed" {
			t.Errorf("Expected sidebar='collapsed', got %v", prefMap["sidebar"])
		}
	})

	t.Run("StringValue", func(t *testing.T) {
		_, err := store.Preferences.Set(ctx, "user3", "language", "en-US")
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		pref, err := store.Preferences.Get(ctx, "user3", "language")
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		if pref.Value != "en-US" {
			t.Errorf("Expected Value='en-US', got %v", pref.Value)
		}
	})
}

func TestPreferencesStore_Set(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	t.Run("CreateNew", func(t *testing.T) {
		pref, err := store.Preferences.Set(ctx, "user-new-1", "newKey", "newValue")
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		if pref.ID == "" {
			t.Error("Expected ID to be set")
		}
		if pref.UserID != "user-new-1" {
			t.Errorf("Expected UserID='user-new-1', got %s", pref.UserID)
		}
		if pref.Key != "newKey" {
			t.Errorf("Expected Key='newKey', got %s", pref.Key)
		}
		if pref.CreatedAt.IsZero() {
			t.Error("Expected CreatedAt to be set")
		}
	})

	t.Run("UpdateExisting", func(t *testing.T) {
		// Create
		original, err := store.Preferences.Set(ctx, "user-update-1", "updateKey", "original")
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		// Update
		updated, err := store.Preferences.Set(ctx, "user-update-1", "updateKey", "updated")
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		if updated.ID != original.ID {
			t.Errorf("Expected ID to remain %s, got %s", original.ID, updated.ID)
		}
		if updated.Value != "updated" {
			t.Errorf("Expected Value='updated', got %v", updated.Value)
		}
	})

	t.Run("PreservesCreatedAt", func(t *testing.T) {
		original, err := store.Preferences.Set(ctx, "user-preserve-1", "preserveKey", "v1")
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		time.Sleep(10 * time.Millisecond)

		updated, err := store.Preferences.Set(ctx, "user-preserve-1", "preserveKey", "v2")
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		if !updated.CreatedAt.Equal(original.CreatedAt) {
			t.Error("Expected CreatedAt to remain unchanged")
		}
	})

	t.Run("UpdatesTimestamp", func(t *testing.T) {
		original, err := store.Preferences.Set(ctx, "user-timestamp-1", "tsKey", "v1")
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		time.Sleep(10 * time.Millisecond)

		updated, err := store.Preferences.Set(ctx, "user-timestamp-1", "tsKey", "v2")
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		if !updated.UpdatedAt.After(original.UpdatedAt) {
			t.Error("Expected UpdatedAt to be updated")
		}
	})

	t.Run("StringValue", func(t *testing.T) {
		pref, err := store.Preferences.Set(ctx, "user-type-1", "stringKey", "simple string")
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		if pref.Value != "simple string" {
			t.Errorf("Expected Value='simple string', got %v", pref.Value)
		}
	})

	t.Run("ObjectValue", func(t *testing.T) {
		value := map[string]any{
			"nested": map[string]any{
				"key": "value",
			},
		}
		pref, err := store.Preferences.Set(ctx, "user-type-2", "objectKey", value)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		// Fetch and verify
		fetched, _ := store.Preferences.Get(ctx, "user-type-2", "objectKey")
		if fetched == nil {
			t.Fatal("Expected preference to be found")
		}
		fetchedMap, ok := fetched.Value.(map[string]any)
		if !ok {
			t.Fatalf("Expected map value, got %T", fetched.Value)
		}
		nested, ok := fetchedMap["nested"].(map[string]any)
		if !ok {
			t.Fatalf("Expected nested map, got %T", fetchedMap["nested"])
		}
		if nested["key"] != "value" {
			t.Errorf("Expected nested.key='value', got %v", nested["key"])
		}

		if pref.ID == "" {
			t.Error("Expected ID to be set")
		}
	})

	t.Run("ArrayValue", func(t *testing.T) {
		value := []any{"item1", "item2", "item3"}
		pref, err := store.Preferences.Set(ctx, "user-type-3", "arrayKey", value)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		// Fetch and verify
		fetched, _ := store.Preferences.Get(ctx, "user-type-3", "arrayKey")
		if fetched == nil {
			t.Fatal("Expected preference to be found")
		}
		arr, ok := fetched.Value.([]any)
		if !ok {
			t.Fatalf("Expected array value, got %T", fetched.Value)
		}
		if len(arr) != 3 {
			t.Errorf("Expected 3 items, got %d", len(arr))
		}

		if pref.ID == "" {
			t.Error("Expected ID to be set")
		}
	})

	t.Run("BoolValue", func(t *testing.T) {
		pref, err := store.Preferences.Set(ctx, "user-type-4", "boolKey", true)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		fetched, _ := store.Preferences.Get(ctx, "user-type-4", "boolKey")
		if fetched == nil {
			t.Fatal("Expected preference to be found")
		}
		if fetched.Value != true {
			t.Errorf("Expected Value=true, got %v", fetched.Value)
		}

		if pref.ID == "" {
			t.Error("Expected ID to be set")
		}
	})

	t.Run("NumberValue", func(t *testing.T) {
		pref, err := store.Preferences.Set(ctx, "user-type-5", "numberKey", 42)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		fetched, _ := store.Preferences.Get(ctx, "user-type-5", "numberKey")
		if fetched == nil {
			t.Fatal("Expected preference to be found")
		}
		// JSON numbers are float64
		if fetched.Value != float64(42) {
			t.Errorf("Expected Value=42, got %v (type %T)", fetched.Value, fetched.Value)
		}

		if pref.ID == "" {
			t.Error("Expected ID to be set")
		}
	})
}

func TestPreferencesStore_Delete(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	t.Run("Exists", func(t *testing.T) {
		store.Preferences.Set(ctx, "user-del-1", "delKey", "value")

		err := store.Preferences.Delete(ctx, "user-del-1", "delKey")
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
	})

	t.Run("NotExists", func(t *testing.T) {
		err := store.Preferences.Delete(ctx, "nonexistent", "nonexistent")
		if err != nil {
			t.Fatalf("Delete failed for non-existent: %v", err)
		}
	})

	t.Run("Verify", func(t *testing.T) {
		store.Preferences.Set(ctx, "user-del-2", "verifyKey", "value")
		store.Preferences.Delete(ctx, "user-del-2", "verifyKey")

		pref, _ := store.Preferences.Get(ctx, "user-del-2", "verifyKey")
		if pref != nil {
			t.Error("Expected preference to be deleted")
		}
	})
}

func TestPreferencesStore_ListByUser(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	t.Run("Multiple", func(t *testing.T) {
		userID := "user-list-multi"
		keys := []string{"pref-c", "pref-a", "pref-b"}
		for _, key := range keys {
			store.Preferences.Set(ctx, userID, key, "value-"+key)
		}

		prefs, err := store.Preferences.ListByUser(ctx, userID)
		if err != nil {
			t.Fatalf("ListByUser failed: %v", err)
		}

		if len(prefs) != 3 {
			t.Errorf("Expected 3 preferences, got %d", len(prefs))
		}
	})

	t.Run("Empty", func(t *testing.T) {
		prefs, err := store.Preferences.ListByUser(ctx, "user-with-no-prefs")
		if err != nil {
			t.Fatalf("ListByUser failed: %v", err)
		}

		if prefs == nil {
			t.Error("Expected empty slice, got nil")
		}
		if len(prefs) != 0 {
			t.Errorf("Expected 0 preferences, got %d", len(prefs))
		}
	})

	t.Run("OrderByKey", func(t *testing.T) {
		userID := "user-list-order"
		keys := []string{"zulu", "alpha", "mike"}
		for _, key := range keys {
			store.Preferences.Set(ctx, userID, key, "value")
		}

		prefs, err := store.Preferences.ListByUser(ctx, userID)
		if err != nil {
			t.Fatalf("ListByUser failed: %v", err)
		}

		expected := []string{"alpha", "mike", "zulu"}
		for i, pref := range prefs {
			if pref.Key != expected[i] {
				t.Errorf("At index %d: expected %s, got %s", i, expected[i], pref.Key)
			}
		}
	})

	t.Run("IsolatedByUser", func(t *testing.T) {
		store.Preferences.Set(ctx, "user-isolated-1", "key1", "value1")
		store.Preferences.Set(ctx, "user-isolated-2", "key2", "value2")

		prefs, err := store.Preferences.ListByUser(ctx, "user-isolated-1")
		if err != nil {
			t.Fatalf("ListByUser failed: %v", err)
		}

		// Should only have user-isolated-1's preferences
		for _, pref := range prefs {
			if pref.UserID != "user-isolated-1" {
				t.Errorf("Expected only user-isolated-1's prefs, got %s", pref.UserID)
			}
		}
	})
}
