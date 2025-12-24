package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/chat/feature/presence"
)

func TestPresenceStore_Upsert(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	store := NewPresenceStore(db)
	ctx := context.Background()

	user := createTestUser(t, usersStore, "testuser")

	p := &presence.Presence{
		UserID:       user.ID,
		Status:       presence.StatusOnline,
		CustomStatus: "Working",
		Activities: []presence.Activity{
			{Type: "playing", Name: "Test Game"},
		},
		ClientStatus: presence.ClientStatus{
			Desktop: "online",
		},
		LastSeenAt: time.Now(),
	}

	err := store.Upsert(ctx, p)
	if err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}

	got, err := store.Get(ctx, user.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got.Status != presence.StatusOnline {
		t.Errorf("Status = %v, want online", got.Status)
	}
	if got.CustomStatus != "Working" {
		t.Errorf("CustomStatus = %v, want Working", got.CustomStatus)
	}

	// Update (upsert existing)
	p.Status = presence.StatusIdle
	err = store.Upsert(ctx, p)
	if err != nil {
		t.Fatalf("Upsert() update error = %v", err)
	}

	got, _ = store.Get(ctx, user.ID)
	if got.Status != presence.StatusIdle {
		t.Errorf("Status after update = %v, want idle", got.Status)
	}
}

func TestPresenceStore_Get(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	store := NewPresenceStore(db)
	ctx := context.Background()

	user := createTestUser(t, usersStore, "testuser")

	// Get non-existent returns offline
	got, err := store.Get(ctx, user.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got.Status != presence.StatusOffline {
		t.Errorf("Status for non-existent = %v, want offline", got.Status)
	}
	if got.UserID != user.ID {
		t.Errorf("UserID = %v, want %v", got.UserID, user.ID)
	}

	// Create presence
	p := &presence.Presence{
		UserID:     user.ID,
		Status:     presence.StatusDND,
		LastSeenAt: time.Now(),
	}
	store.Upsert(ctx, p)

	got, _ = store.Get(ctx, user.ID)
	if got.Status != presence.StatusDND {
		t.Errorf("Status = %v, want dnd", got.Status)
	}
}

func TestPresenceStore_GetBulk(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	store := NewPresenceStore(db)
	ctx := context.Background()

	user1 := createTestUser(t, usersStore, "user1")
	user2 := createTestUser(t, usersStore, "user2")
	user3 := createTestUser(t, usersStore, "user3")

	// Set presence for user1 and user2
	store.Upsert(ctx, &presence.Presence{
		UserID:     user1.ID,
		Status:     presence.StatusOnline,
		LastSeenAt: time.Now(),
	})
	store.Upsert(ctx, &presence.Presence{
		UserID:     user2.ID,
		Status:     presence.StatusIdle,
		LastSeenAt: time.Now(),
	})
	// user3 has no presence

	presences, err := store.GetBulk(ctx, []string{user1.ID, user2.ID, user3.ID})
	if err != nil {
		t.Fatalf("GetBulk() error = %v", err)
	}

	if len(presences) != 3 {
		t.Errorf("len(presences) = %d, want 3", len(presences))
	}

	// Verify statuses
	statusMap := make(map[string]presence.Status)
	for _, p := range presences {
		statusMap[p.UserID] = p.Status
	}

	if statusMap[user1.ID] != presence.StatusOnline {
		t.Errorf("user1 status = %v, want online", statusMap[user1.ID])
	}
	if statusMap[user2.ID] != presence.StatusIdle {
		t.Errorf("user2 status = %v, want idle", statusMap[user2.ID])
	}
	if statusMap[user3.ID] != presence.StatusOffline {
		t.Errorf("user3 status = %v, want offline (default)", statusMap[user3.ID])
	}

	// Empty slice
	presences, err = store.GetBulk(ctx, []string{})
	if err != nil {
		t.Fatalf("GetBulk() with empty slice error = %v", err)
	}
	if presences != nil {
		t.Errorf("expected nil for empty slice, got %v", presences)
	}
}

func TestPresenceStore_UpdateStatus(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	store := NewPresenceStore(db)
	ctx := context.Background()

	user := createTestUser(t, usersStore, "testuser")

	// UpdateStatus creates if not exists
	err := store.UpdateStatus(ctx, user.ID, presence.StatusOnline)
	if err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}

	got, _ := store.Get(ctx, user.ID)
	if got.Status != presence.StatusOnline {
		t.Errorf("Status = %v, want online", got.Status)
	}

	// Update existing
	err = store.UpdateStatus(ctx, user.ID, presence.StatusDND)
	if err != nil {
		t.Fatalf("UpdateStatus() update error = %v", err)
	}

	got, _ = store.Get(ctx, user.ID)
	if got.Status != presence.StatusDND {
		t.Errorf("Status after update = %v, want dnd", got.Status)
	}
}

func TestPresenceStore_SetOffline(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	store := NewPresenceStore(db)
	ctx := context.Background()

	user := createTestUser(t, usersStore, "testuser")

	// Set online first
	store.UpdateStatus(ctx, user.ID, presence.StatusOnline)

	// Set offline
	err := store.SetOffline(ctx, user.ID)
	if err != nil {
		t.Fatalf("SetOffline() error = %v", err)
	}

	got, _ := store.Get(ctx, user.ID)
	if got.Status != presence.StatusOffline {
		t.Errorf("Status = %v, want offline", got.Status)
	}
}

func TestPresenceStore_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	store := NewPresenceStore(db)
	ctx := context.Background()

	user := createTestUser(t, usersStore, "testuser")

	// Create presence
	store.Upsert(ctx, &presence.Presence{
		UserID:     user.ID,
		Status:     presence.StatusOnline,
		LastSeenAt: time.Now(),
	})

	// Delete
	err := store.Delete(ctx, user.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Should return default offline
	got, _ := store.Get(ctx, user.ID)
	if got.Status != presence.StatusOffline {
		t.Errorf("Status after delete = %v, want offline", got.Status)
	}
}

func TestPresenceStore_CleanupStale(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	store := NewPresenceStore(db)
	ctx := context.Background()

	user1 := createTestUser(t, usersStore, "user1")
	user2 := createTestUser(t, usersStore, "user2")

	// User1 has stale presence (last seen 1 hour ago)
	store.Upsert(ctx, &presence.Presence{
		UserID:     user1.ID,
		Status:     presence.StatusOnline,
		LastSeenAt: time.Now().Add(-1 * time.Hour),
	})

	// User2 has recent presence
	store.Upsert(ctx, &presence.Presence{
		UserID:     user2.ID,
		Status:     presence.StatusOnline,
		LastSeenAt: time.Now(),
	})

	// Cleanup stale (older than 30 minutes)
	err := store.CleanupStale(ctx, time.Now().Add(-30*time.Minute))
	if err != nil {
		t.Fatalf("CleanupStale() error = %v", err)
	}

	// User1 should be offline
	got1, _ := store.Get(ctx, user1.ID)
	if got1.Status != presence.StatusOffline {
		t.Errorf("user1 status = %v, want offline", got1.Status)
	}

	// User2 should still be online
	got2, _ := store.Get(ctx, user2.ID)
	if got2.Status != presence.StatusOnline {
		t.Errorf("user2 status = %v, want online", got2.Status)
	}
}

func TestPresenceStore_WithActivities(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	store := NewPresenceStore(db)
	ctx := context.Background()

	user := createTestUser(t, usersStore, "testuser")

	activities := []presence.Activity{
		{Type: "playing", Name: "Game 1", Details: "Level 5"},
		{Type: "listening", Name: "Spotify", Details: "Song Name"},
	}

	p := &presence.Presence{
		UserID:     user.ID,
		Status:     presence.StatusOnline,
		Activities: activities,
		LastSeenAt: time.Now(),
	}

	err := store.Upsert(ctx, p)
	if err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}

	got, _ := store.Get(ctx, user.ID)
	if len(got.Activities) != 2 {
		t.Errorf("len(Activities) = %d, want 2", len(got.Activities))
	}

	if got.Activities[0].Name != "Game 1" {
		t.Errorf("Activities[0].Name = %v, want Game 1", got.Activities[0].Name)
	}
}

func TestPresenceStore_WithClientStatus(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	store := NewPresenceStore(db)
	ctx := context.Background()

	user := createTestUser(t, usersStore, "testuser")

	clientStatus := presence.ClientStatus{
		Desktop: "online",
		Mobile:  "idle",
		Web:     "offline",
	}

	p := &presence.Presence{
		UserID:       user.ID,
		Status:       presence.StatusOnline,
		ClientStatus: clientStatus,
		LastSeenAt:   time.Now(),
	}

	err := store.Upsert(ctx, p)
	if err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}

	got, _ := store.Get(ctx, user.ID)
	if got.ClientStatus.Desktop == "" && got.ClientStatus.Mobile == "" && got.ClientStatus.Web == "" {
		t.Fatal("ClientStatus is empty")
	}

	if got.ClientStatus.Desktop != "online" {
		t.Errorf("Desktop = %v, want online", got.ClientStatus.Desktop)
	}
	if got.ClientStatus.Mobile != "idle" {
		t.Errorf("Mobile = %v, want idle", got.ClientStatus.Mobile)
	}
}
