package duckdb

import (
	"context"
	"testing"

	"github.com/go-mizu/blueprints/cms/feature/settings"
)

func TestSettingsStore_Set_Create(t *testing.T) {
	db := setupTestDB(t)
	store := NewSettingsStore(db)
	ctx := context.Background()

	setting := &settings.Setting{
		ID:          "setting-001",
		Key:         "site_title",
		Value:       "My Website",
		ValueType:   "string",
		GroupName:   "general",
		Description: "The title of the site",
		IsPublic:    true,
		CreatedAt:   testTime,
		UpdatedAt:   testTime,
	}

	err := store.Set(ctx, setting)
	assertNoError(t, err)

	got, err := store.Get(ctx, setting.Key)
	assertNoError(t, err)
	assertEqual(t, "Key", got.Key, setting.Key)
	assertEqual(t, "Value", got.Value, setting.Value)
	assertEqual(t, "ValueType", got.ValueType, setting.ValueType)
	assertEqual(t, "GroupName", got.GroupName, setting.GroupName)
	assertEqual(t, "IsPublic", got.IsPublic, setting.IsPublic)
}

func TestSettingsStore_Set_Update(t *testing.T) {
	db := setupTestDB(t)
	store := NewSettingsStore(db)
	ctx := context.Background()

	// Create initial setting
	setting := &settings.Setting{
		ID:        "setting-upd-001",
		Key:       "site_name",
		Value:     "Original Name",
		ValueType: "string",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.Set(ctx, setting))

	// Update (upsert)
	updated := &settings.Setting{
		ID:        "setting-upd-002", // Different ID, same key
		Key:       "site_name",
		Value:     "Updated Name",
		ValueType: "string",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	err := store.Set(ctx, updated)
	assertNoError(t, err)

	got, _ := store.Get(ctx, "site_name")
	assertEqual(t, "Value", got.Value, "Updated Name")
}

func TestSettingsStore_Set_WithGroup(t *testing.T) {
	db := setupTestDB(t)
	store := NewSettingsStore(db)
	ctx := context.Background()

	setting := &settings.Setting{
		ID:        "setting-group",
		Key:       "theme_color",
		Value:     "#ff0000",
		ValueType: "string",
		GroupName: "appearance",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}

	err := store.Set(ctx, setting)
	assertNoError(t, err)

	got, _ := store.Get(ctx, setting.Key)
	assertEqual(t, "GroupName", got.GroupName, "appearance")
}

func TestSettingsStore_Set_Public(t *testing.T) {
	db := setupTestDB(t)
	store := NewSettingsStore(db)
	ctx := context.Background()

	setting := &settings.Setting{
		ID:        "setting-public",
		Key:       "public_key",
		Value:     "public_value",
		ValueType: "string",
		IsPublic:  true,
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}

	err := store.Set(ctx, setting)
	assertNoError(t, err)

	got, _ := store.Get(ctx, setting.Key)
	assertEqual(t, "IsPublic", got.IsPublic, true)
}

func TestSettingsStore_Get(t *testing.T) {
	db := setupTestDB(t)
	store := NewSettingsStore(db)
	ctx := context.Background()

	setting := &settings.Setting{
		ID:        "setting-get",
		Key:       "get_key",
		Value:     "get_value",
		ValueType: "string",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.Set(ctx, setting))

	got, err := store.Get(ctx, "get_key")
	assertNoError(t, err)
	assertEqual(t, "Key", got.Key, "get_key")
	assertEqual(t, "Value", got.Value, "get_value")
}

func TestSettingsStore_Get_NotFound(t *testing.T) {
	db := setupTestDB(t)
	store := NewSettingsStore(db)
	ctx := context.Background()

	got, err := store.Get(ctx, "nonexistent")
	assertNoError(t, err)
	if got != nil {
		t.Error("expected nil for non-existent setting")
	}
}

func TestSettingsStore_GetByGroup(t *testing.T) {
	db := setupTestDB(t)
	store := NewSettingsStore(db)
	ctx := context.Background()

	// Create settings in different groups
	settingsData := []struct {
		key   string
		group string
	}{
		{"general_1", "general"},
		{"general_2", "general"},
		{"appearance_1", "appearance"},
	}
	for i, sd := range settingsData {
		setting := &settings.Setting{
			ID:        "setting-group-" + string(rune('a'+i)),
			Key:       sd.key,
			Value:     "value",
			ValueType: "string",
			GroupName: sd.group,
			CreatedAt: testTime,
			UpdatedAt: testTime,
		}
		assertNoError(t, store.Set(ctx, setting))
	}

	list, err := store.GetByGroup(ctx, "general")
	assertNoError(t, err)
	assertLen(t, list, 2)
}

func TestSettingsStore_GetByGroup_Empty(t *testing.T) {
	db := setupTestDB(t)
	store := NewSettingsStore(db)
	ctx := context.Background()

	list, err := store.GetByGroup(ctx, "nonexistent_group")
	assertNoError(t, err)
	assertLen(t, list, 0)
}

func TestSettingsStore_GetAll(t *testing.T) {
	db := setupTestDB(t)
	store := NewSettingsStore(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		setting := &settings.Setting{
			ID:        "setting-all-" + string(rune('a'+i)),
			Key:       "all_key_" + string(rune('a'+i)),
			Value:     "value",
			ValueType: "string",
			CreatedAt: testTime,
			UpdatedAt: testTime,
		}
		assertNoError(t, store.Set(ctx, setting))
	}

	list, err := store.GetAll(ctx)
	assertNoError(t, err)
	assertLen(t, list, 5)
}

func TestSettingsStore_GetAll_Empty(t *testing.T) {
	db := setupTestDB(t)
	store := NewSettingsStore(db)
	ctx := context.Background()

	list, err := store.GetAll(ctx)
	assertNoError(t, err)
	assertLen(t, list, 0)
}

func TestSettingsStore_GetPublic(t *testing.T) {
	db := setupTestDB(t)
	store := NewSettingsStore(db)
	ctx := context.Background()

	// Create mix of public and private settings
	settingsData := []struct {
		key      string
		isPublic bool
	}{
		{"public_1", true},
		{"private_1", false},
		{"public_2", true},
		{"private_2", false},
	}
	for i, sd := range settingsData {
		setting := &settings.Setting{
			ID:        "setting-pub-" + string(rune('a'+i)),
			Key:       sd.key,
			Value:     "value",
			ValueType: "string",
			IsPublic:  sd.isPublic,
			CreatedAt: testTime,
			UpdatedAt: testTime,
		}
		assertNoError(t, store.Set(ctx, setting))
	}

	list, err := store.GetPublic(ctx)
	assertNoError(t, err)
	assertLen(t, list, 2)

	// All should be public
	for _, s := range list {
		if !s.IsPublic {
			t.Errorf("expected IsPublic=true for key %s", s.Key)
		}
	}
}

func TestSettingsStore_Delete(t *testing.T) {
	db := setupTestDB(t)
	store := NewSettingsStore(db)
	ctx := context.Background()

	setting := &settings.Setting{
		ID:        "setting-delete",
		Key:       "delete_key",
		Value:     "delete_value",
		ValueType: "string",
		CreatedAt: testTime,
		UpdatedAt: testTime,
	}
	assertNoError(t, store.Set(ctx, setting))

	err := store.Delete(ctx, "delete_key")
	assertNoError(t, err)

	got, _ := store.Get(ctx, "delete_key")
	if got != nil {
		t.Error("expected setting to be deleted")
	}
}

func TestSettingsStore_Delete_NotFound(t *testing.T) {
	db := setupTestDB(t)
	store := NewSettingsStore(db)
	ctx := context.Background()

	err := store.Delete(ctx, "nonexistent")
	assertNoError(t, err) // Should not error for non-existent
}
