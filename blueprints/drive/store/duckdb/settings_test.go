package duckdb

import (
	"context"
	"testing"
	"time"
)

// ============================================================
// Settings CRUD Tests
// ============================================================

func TestCreateSettings_Success(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	settings := newTestSettings("user1")
	settings.Theme = "dark"
	settings.Language = "es"
	settings.Timezone = "America/New_York"

	if err := store.CreateSettings(ctx, settings); err != nil {
		t.Fatalf("create settings failed: %v", err)
	}

	got, err := store.GetSettings(ctx, "user1")
	if err != nil {
		t.Fatalf("get settings failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected settings, got nil")
	}
	if got.Theme != "dark" {
		t.Errorf("expected theme dark, got %s", got.Theme)
	}
	if got.Language != "es" {
		t.Errorf("expected language es, got %s", got.Language)
	}
	if got.Timezone != "America/New_York" {
		t.Errorf("expected timezone America/New_York, got %s", got.Timezone)
	}
}

func TestGetSettings_Exists(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	settings := newTestSettings("user1")
	store.CreateSettings(ctx, settings)

	got, err := store.GetSettings(ctx, "user1")
	if err != nil {
		t.Fatalf("get settings failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected settings, got nil")
	}
	if got.UserID != "user1" {
		t.Errorf("expected user_id user1, got %s", got.UserID)
	}
}

func TestGetSettings_NotFound(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	got, err := store.GetSettings(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("get settings failed: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestUpdateSettings_Success(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	settings := newTestSettings("user1")
	store.CreateSettings(ctx, settings)

	// Update all fields
	settings.Theme = "light"
	settings.Language = "fr"
	settings.Timezone = "Europe/Paris"
	settings.ListView = "grid"
	settings.SortBy = "date"
	settings.SortOrder = "desc"
	settings.NotificationsEnabled = false
	settings.EmailNotifications = false
	settings.UpdatedAt = time.Now().Truncate(time.Microsecond)

	if err := store.UpdateSettings(ctx, settings); err != nil {
		t.Fatalf("update settings failed: %v", err)
	}

	got, _ := store.GetSettings(ctx, "user1")
	if got.Theme != "light" {
		t.Errorf("expected theme light, got %s", got.Theme)
	}
	if got.Language != "fr" {
		t.Errorf("expected language fr, got %s", got.Language)
	}
	if got.ListView != "grid" {
		t.Errorf("expected list_view grid, got %s", got.ListView)
	}
	if got.SortBy != "date" {
		t.Errorf("expected sort_by date, got %s", got.SortBy)
	}
	if got.SortOrder != "desc" {
		t.Errorf("expected sort_order desc, got %s", got.SortOrder)
	}
	if got.NotificationsEnabled {
		t.Error("expected notifications_enabled false")
	}
	if got.EmailNotifications {
		t.Error("expected email_notifications false")
	}
}

func TestDeleteSettings_Success(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	settings := newTestSettings("user1")
	store.CreateSettings(ctx, settings)

	if err := store.DeleteSettings(ctx, "user1"); err != nil {
		t.Fatalf("delete settings failed: %v", err)
	}

	got, _ := store.GetSettings(ctx, "user1")
	if got != nil {
		t.Errorf("expected nil after delete, got %+v", got)
	}
}

// ============================================================
// Settings Upsert Tests
// ============================================================

func TestUpsertSettings_Create(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	settings := newTestSettings("user1")
	settings.Theme = "dark"

	if err := store.UpsertSettings(ctx, settings); err != nil {
		t.Fatalf("upsert settings failed: %v", err)
	}

	got, _ := store.GetSettings(ctx, "user1")
	if got == nil {
		t.Fatal("expected settings to be created")
	}
	if got.Theme != "dark" {
		t.Errorf("expected theme dark, got %s", got.Theme)
	}
}

func TestUpsertSettings_Update(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	// Create initial settings
	settings := newTestSettings("user1")
	settings.Theme = "light"
	store.CreateSettings(ctx, settings)

	// Upsert with new values
	settings.Theme = "dark"
	settings.UpdatedAt = time.Now().Truncate(time.Microsecond)

	if err := store.UpsertSettings(ctx, settings); err != nil {
		t.Fatalf("upsert settings failed: %v", err)
	}

	got, _ := store.GetSettings(ctx, "user1")
	if got.Theme != "dark" {
		t.Errorf("expected theme to be updated to dark, got %s", got.Theme)
	}
}

// ============================================================
// Default Settings Tests
// ============================================================

func TestGetOrCreateDefaultSettings_Create(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	// No settings exist yet
	settings, err := store.GetOrCreateDefaultSettings(ctx, "user1")
	if err != nil {
		t.Fatalf("get or create default settings failed: %v", err)
	}
	if settings == nil {
		t.Fatal("expected settings, got nil")
	}

	// Verify defaults
	if settings.Theme != "system" {
		t.Errorf("expected default theme system, got %s", settings.Theme)
	}
	if settings.Language != "en" {
		t.Errorf("expected default language en, got %s", settings.Language)
	}
	if settings.Timezone != "UTC" {
		t.Errorf("expected default timezone UTC, got %s", settings.Timezone)
	}
	if settings.ListView != "list" {
		t.Errorf("expected default list_view list, got %s", settings.ListView)
	}
	if settings.SortBy != "name" {
		t.Errorf("expected default sort_by name, got %s", settings.SortBy)
	}
	if settings.SortOrder != "asc" {
		t.Errorf("expected default sort_order asc, got %s", settings.SortOrder)
	}
	if !settings.NotificationsEnabled {
		t.Error("expected default notifications_enabled true")
	}
	if !settings.EmailNotifications {
		t.Error("expected default email_notifications true")
	}
}

func TestGetOrCreateDefaultSettings_Get(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	// Create custom settings first
	custom := newTestSettings("user1")
	custom.Theme = "dark"
	custom.Language = "ja"
	store.CreateSettings(ctx, custom)

	// GetOrCreate should return existing settings, not overwrite
	settings, err := store.GetOrCreateDefaultSettings(ctx, "user1")
	if err != nil {
		t.Fatalf("get or create default settings failed: %v", err)
	}

	if settings.Theme != "dark" {
		t.Errorf("expected existing theme dark, got %s", settings.Theme)
	}
	if settings.Language != "ja" {
		t.Errorf("expected existing language ja, got %s", settings.Language)
	}
}

// ============================================================
// Business Use Cases - User Preferences
// ============================================================

func TestPreferences_ThemeChange(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "alice@example.com")
	store.CreateUser(ctx, user)

	// User gets default settings
	settings, _ := store.GetOrCreateDefaultSettings(ctx, "user1")
	if settings.Theme != "system" {
		t.Errorf("expected initial theme system, got %s", settings.Theme)
	}

	// User changes to dark theme
	settings.Theme = "dark"
	store.UpdateSettings(ctx, settings)

	updated, _ := store.GetSettings(ctx, "user1")
	if updated.Theme != "dark" {
		t.Errorf("expected theme dark after change, got %s", updated.Theme)
	}

	// User changes to light theme
	settings.Theme = "light"
	store.UpdateSettings(ctx, settings)

	updated, _ = store.GetSettings(ctx, "user1")
	if updated.Theme != "light" {
		t.Errorf("expected theme light after change, got %s", updated.Theme)
	}
}

func TestPreferences_NotificationToggle(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "alice@example.com")
	store.CreateUser(ctx, user)

	settings, _ := store.GetOrCreateDefaultSettings(ctx, "user1")

	// Default: both notifications enabled
	if !settings.NotificationsEnabled || !settings.EmailNotifications {
		t.Error("expected both notifications enabled by default")
	}

	// Disable push notifications
	settings.NotificationsEnabled = false
	store.UpdateSettings(ctx, settings)

	updated, _ := store.GetSettings(ctx, "user1")
	if updated.NotificationsEnabled {
		t.Error("expected push notifications disabled")
	}
	if !updated.EmailNotifications {
		t.Error("email notifications should still be enabled")
	}

	// Disable email notifications too
	settings.EmailNotifications = false
	store.UpdateSettings(ctx, settings)

	updated, _ = store.GetSettings(ctx, "user1")
	if updated.EmailNotifications {
		t.Error("expected email notifications disabled")
	}
}

func TestPreferences_SortPreferences(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "alice@example.com")
	store.CreateUser(ctx, user)

	settings, _ := store.GetOrCreateDefaultSettings(ctx, "user1")

	// Default: sorted by name ascending
	if settings.SortBy != "name" || settings.SortOrder != "asc" {
		t.Errorf("expected default sort name/asc, got %s/%s", settings.SortBy, settings.SortOrder)
	}

	// Change to sort by date descending
	settings.SortBy = "modified"
	settings.SortOrder = "desc"
	store.UpdateSettings(ctx, settings)

	updated, _ := store.GetSettings(ctx, "user1")
	if updated.SortBy != "modified" {
		t.Errorf("expected sort_by modified, got %s", updated.SortBy)
	}
	if updated.SortOrder != "desc" {
		t.Errorf("expected sort_order desc, got %s", updated.SortOrder)
	}

	// Change to sort by size
	settings.SortBy = "size"
	settings.SortOrder = "asc"
	store.UpdateSettings(ctx, settings)

	updated, _ = store.GetSettings(ctx, "user1")
	if updated.SortBy != "size" {
		t.Errorf("expected sort_by size, got %s", updated.SortBy)
	}
}

func TestPreferences_NewUserDefaults(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	// Create multiple new users
	users := []string{"user1", "user2", "user3"}
	for _, id := range users {
		user := newTestUser(id, id+"@example.com")
		store.CreateUser(ctx, user)
	}

	// Each new user should get default settings
	for _, id := range users {
		settings, err := store.GetOrCreateDefaultSettings(ctx, id)
		if err != nil {
			t.Fatalf("failed for user %s: %v", id, err)
		}

		// Verify all defaults
		if settings.Theme != "system" {
			t.Errorf("user %s: expected theme system, got %s", id, settings.Theme)
		}
		if settings.Language != "en" {
			t.Errorf("user %s: expected language en, got %s", id, settings.Language)
		}
		if settings.Timezone != "UTC" {
			t.Errorf("user %s: expected timezone UTC, got %s", id, settings.Timezone)
		}
		if !settings.NotificationsEnabled {
			t.Errorf("user %s: expected notifications enabled", id)
		}
	}
}

func TestPreferences_ListViewToggle(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "alice@example.com")
	store.CreateUser(ctx, user)

	settings, _ := store.GetOrCreateDefaultSettings(ctx, "user1")

	// Default: list view
	if settings.ListView != "list" {
		t.Errorf("expected default list view, got %s", settings.ListView)
	}

	// Switch to grid view
	settings.ListView = "grid"
	store.UpdateSettings(ctx, settings)

	updated, _ := store.GetSettings(ctx, "user1")
	if updated.ListView != "grid" {
		t.Errorf("expected grid view, got %s", updated.ListView)
	}

	// Switch to compact view
	settings.ListView = "compact"
	store.UpdateSettings(ctx, settings)

	updated, _ = store.GetSettings(ctx, "user1")
	if updated.ListView != "compact" {
		t.Errorf("expected compact view, got %s", updated.ListView)
	}
}

func TestPreferences_TimezoneChange(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "alice@example.com")
	store.CreateUser(ctx, user)

	settings, _ := store.GetOrCreateDefaultSettings(ctx, "user1")

	// Default: UTC
	if settings.Timezone != "UTC" {
		t.Errorf("expected default timezone UTC, got %s", settings.Timezone)
	}

	// User moves and changes timezone
	timezones := []string{"America/Los_Angeles", "Asia/Tokyo", "Europe/London"}
	for _, tz := range timezones {
		settings.Timezone = tz
		store.UpdateSettings(ctx, settings)

		updated, _ := store.GetSettings(ctx, "user1")
		if updated.Timezone != tz {
			t.Errorf("expected timezone %s, got %s", tz, updated.Timezone)
		}
	}
}

func TestPreferences_LanguageChange(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "alice@example.com")
	store.CreateUser(ctx, user)

	settings, _ := store.GetOrCreateDefaultSettings(ctx, "user1")

	// Default: English
	if settings.Language != "en" {
		t.Errorf("expected default language en, got %s", settings.Language)
	}

	// User changes language preferences
	languages := []string{"es", "fr", "de", "ja", "zh"}
	for _, lang := range languages {
		settings.Language = lang
		store.UpdateSettings(ctx, settings)

		updated, _ := store.GetSettings(ctx, "user1")
		if updated.Language != lang {
			t.Errorf("expected language %s, got %s", lang, updated.Language)
		}
	}
}
