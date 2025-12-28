package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/users"
	"github.com/oklog/ulid/v2"
)

func createTestUser(t *testing.T, store *UsersStore) *users.User {
	t.Helper()
	id := ulid.Make().String()
	u := &users.User{
		ID:           id,
		Username:     "user" + id[len(id)-12:], // Use last 12 chars for uniqueness
		Email:        id + "@example.com",
		PasswordHash: "hashed_password",
		FullName:     "Test User",
		AvatarURL:    "https://example.com/avatar.png",
		Bio:          "A test user bio",
		Location:     "San Francisco",
		Website:      "https://example.com",
		Company:      "Test Corp",
		IsAdmin:      false,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := store.Create(context.Background(), u); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return u
}

// =============================================================================
// User CRUD Tests
// =============================================================================

func TestUsersStore_Create(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	u := &users.User{
		ID:           ulid.Make().String(),
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hashed",
		FullName:     "Test User",
		IsAdmin:      false,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err := usersStore.Create(context.Background(), u)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify user was created
	got, err := usersStore.GetByID(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected user to be created")
	}
	if got.Email != u.Email {
		t.Errorf("got email %q, want %q", got.Email, u.Email)
	}
	if got.Username != u.Username {
		t.Errorf("got username %q, want %q", got.Username, u.Username)
	}
}

func TestUsersStore_Create_WithAllFields(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	u := &users.User{
		ID:           ulid.Make().String(),
		Username:     "fulluser",
		Email:        "full@example.com",
		PasswordHash: "hashed_password_123",
		FullName:     "Full Name User",
		AvatarURL:    "https://example.com/avatar.png",
		Bio:          "This is my bio",
		Location:     "New York",
		Website:      "https://mysite.com",
		Company:      "Acme Inc",
		IsAdmin:      true,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err := usersStore.Create(context.Background(), u)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, _ := usersStore.GetByID(context.Background(), u.ID)
	if got.FullName != u.FullName {
		t.Errorf("got full_name %q, want %q", got.FullName, u.FullName)
	}
	if got.Bio != u.Bio {
		t.Errorf("got bio %q, want %q", got.Bio, u.Bio)
	}
	if got.Location != u.Location {
		t.Errorf("got location %q, want %q", got.Location, u.Location)
	}
	if got.Website != u.Website {
		t.Errorf("got website %q, want %q", got.Website, u.Website)
	}
	if got.Company != u.Company {
		t.Errorf("got company %q, want %q", got.Company, u.Company)
	}
	if got.IsAdmin != u.IsAdmin {
		t.Errorf("got is_admin %v, want %v", got.IsAdmin, u.IsAdmin)
	}
}

func TestUsersStore_Create_DuplicateEmail(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	u1 := &users.User{
		ID:           ulid.Make().String(),
		Username:     "user1",
		Email:        "dup@example.com",
		PasswordHash: "hashed",
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	u2 := &users.User{
		ID:           ulid.Make().String(),
		Username:     "user2",
		Email:        "dup@example.com", // same email
		PasswordHash: "hashed",
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := usersStore.Create(context.Background(), u1); err != nil {
		t.Fatalf("first Create failed: %v", err)
	}

	err := usersStore.Create(context.Background(), u2)
	if err == nil {
		t.Error("expected error for duplicate email")
	}
}

func TestUsersStore_Create_DuplicateUsername(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	u1 := &users.User{
		ID:           ulid.Make().String(),
		Username:     "dupuser",
		Email:        "user1@example.com",
		PasswordHash: "hashed",
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	u2 := &users.User{
		ID:           ulid.Make().String(),
		Username:     "dupuser", // same username
		Email:        "user2@example.com",
		PasswordHash: "hashed",
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := usersStore.Create(context.Background(), u1); err != nil {
		t.Fatalf("first Create failed: %v", err)
	}

	err := usersStore.Create(context.Background(), u2)
	if err == nil {
		t.Error("expected error for duplicate username")
	}
}

func TestUsersStore_GetByID(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	u := createTestUser(t, usersStore)

	got, err := usersStore.GetByID(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected user")
	}
	if got.ID != u.ID {
		t.Errorf("got ID %q, want %q", got.ID, u.ID)
	}
}

func TestUsersStore_GetByID_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())

	got, err := usersStore.GetByID(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent user")
	}
}

func TestUsersStore_GetByUsername(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	u := createTestUser(t, usersStore)

	got, err := usersStore.GetByUsername(context.Background(), u.Username)
	if err != nil {
		t.Fatalf("GetByUsername failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected user")
	}
	if got.Username != u.Username {
		t.Errorf("got username %q, want %q", got.Username, u.Username)
	}
}

func TestUsersStore_GetByUsername_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())

	got, err := usersStore.GetByUsername(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("GetByUsername failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent username")
	}
}

func TestUsersStore_GetByEmail(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	u := createTestUser(t, usersStore)

	got, err := usersStore.GetByEmail(context.Background(), u.Email)
	if err != nil {
		t.Fatalf("GetByEmail failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected user")
	}
	if got.Email != u.Email {
		t.Errorf("got email %q, want %q", got.Email, u.Email)
	}
}

func TestUsersStore_GetByEmail_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())

	got, err := usersStore.GetByEmail(context.Background(), "nonexistent@example.com")
	if err != nil {
		t.Fatalf("GetByEmail failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent email")
	}
}

func TestUsersStore_Update(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	u := createTestUser(t, usersStore)

	// Update fields
	u.FullName = "Updated Name"
	u.Bio = "Updated bio"
	u.Location = "Updated Location"

	err := usersStore.Update(context.Background(), u)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, _ := usersStore.GetByID(context.Background(), u.ID)
	if got.FullName != "Updated Name" {
		t.Errorf("got full_name %q, want %q", got.FullName, "Updated Name")
	}
	if got.Bio != "Updated bio" {
		t.Errorf("got bio %q, want %q", got.Bio, "Updated bio")
	}
}

func TestUsersStore_Update_UpdatesTimestamp(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	u := createTestUser(t, usersStore)
	originalUpdatedAt := u.UpdatedAt

	// Wait a bit to ensure timestamp difference
	time.Sleep(10 * time.Millisecond)

	u.FullName = "New Name"
	err := usersStore.Update(context.Background(), u)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, _ := usersStore.GetByID(context.Background(), u.ID)
	if !got.UpdatedAt.After(originalUpdatedAt) {
		t.Error("expected updated_at to be updated")
	}
}

func TestUsersStore_Update_AdminFlag(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	u := createTestUser(t, usersStore)

	if u.IsAdmin {
		t.Fatal("test user should not be admin initially")
	}

	u.IsAdmin = true
	err := usersStore.Update(context.Background(), u)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, _ := usersStore.GetByID(context.Background(), u.ID)
	if !got.IsAdmin {
		t.Error("expected user to be admin after update")
	}
}

func TestUsersStore_Delete(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	u := createTestUser(t, usersStore)

	err := usersStore.Delete(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	got, _ := usersStore.GetByID(context.Background(), u.ID)
	if got != nil {
		t.Error("expected user to be deleted")
	}
}

func TestUsersStore_Delete_NonExistent(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())

	// Should not error when deleting non-existent user
	err := usersStore.Delete(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("Delete should not error for non-existent user: %v", err)
	}
}

func TestUsersStore_List(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())

	// Create multiple users
	for i := 0; i < 5; i++ {
		createTestUser(t, usersStore)
	}

	users, err := usersStore.List(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(users) != 5 {
		t.Errorf("got %d users, want 5", len(users))
	}
}

func TestUsersStore_List_Pagination(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())

	// Create 10 users
	for i := 0; i < 10; i++ {
		createTestUser(t, usersStore)
	}

	// Get first page
	page1, err := usersStore.List(context.Background(), 3, 0)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(page1) != 3 {
		t.Errorf("got %d users on page 1, want 3", len(page1))
	}

	// Get second page
	page2, err := usersStore.List(context.Background(), 3, 3)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(page2) != 3 {
		t.Errorf("got %d users on page 2, want 3", len(page2))
	}

	// Ensure different users
	if len(page1) > 0 && len(page2) > 0 && page1[0].ID == page2[0].ID {
		t.Error("expected different users on different pages")
	}
}

func TestUsersStore_List_Empty(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())

	users, err := usersStore.List(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if users == nil {
		// nil is acceptable for empty list
	} else if len(users) != 0 {
		t.Errorf("expected empty list, got %d users", len(users))
	}
}

func TestUsersStore_List_OrderByCreatedAt(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())

	// Create users with slight time gaps
	user1 := createTestUser(t, usersStore)
	time.Sleep(10 * time.Millisecond)
	_ = createTestUser(t, usersStore) // user2
	time.Sleep(10 * time.Millisecond)
	user3 := createTestUser(t, usersStore)

	users, err := usersStore.List(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	// Should be ordered by created_at DESC (newest first)
	if len(users) >= 3 {
		if users[0].ID != user3.ID {
			t.Error("expected newest user first")
		}
		if users[2].ID != user1.ID {
			t.Error("expected oldest user last")
		}
	}
}

// =============================================================================
// Session Tests
// =============================================================================

func TestUsersStore_CreateSession(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	u := createTestUser(t, usersStore)

	sess := &users.Session{
		ID:           ulid.Make().String(),
		UserID:       u.ID,
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		UserAgent:    "Mozilla/5.0 Test Browser",
		IPAddress:    "192.168.1.1",
		CreatedAt:    time.Now(),
		LastActiveAt: time.Now(),
	}

	err := usersStore.CreateSession(context.Background(), sess)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	got, err := usersStore.GetSession(context.Background(), sess.ID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected session")
	}
	if got.UserID != u.ID {
		t.Errorf("got user ID %q, want %q", got.UserID, u.ID)
	}
	if got.UserAgent != sess.UserAgent {
		t.Errorf("got user_agent %q, want %q", got.UserAgent, sess.UserAgent)
	}
	if got.IPAddress != sess.IPAddress {
		t.Errorf("got ip_address %q, want %q", got.IPAddress, sess.IPAddress)
	}
}

func TestUsersStore_GetSession_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())

	got, err := usersStore.GetSession(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent session")
	}
}

func TestUsersStore_DeleteSession(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	u := createTestUser(t, usersStore)

	sess := &users.Session{
		ID:           ulid.Make().String(),
		UserID:       u.ID,
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		CreatedAt:    time.Now(),
		LastActiveAt: time.Now(),
	}
	usersStore.CreateSession(context.Background(), sess)

	err := usersStore.DeleteSession(context.Background(), sess.ID)
	if err != nil {
		t.Fatalf("DeleteSession failed: %v", err)
	}

	got, _ := usersStore.GetSession(context.Background(), sess.ID)
	if got != nil {
		t.Error("expected session to be deleted")
	}
}

func TestUsersStore_DeleteUserSessions(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	u := createTestUser(t, usersStore)

	// Create multiple sessions for the same user
	sess1 := &users.Session{
		ID:           ulid.Make().String(),
		UserID:       u.ID,
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		CreatedAt:    time.Now(),
		LastActiveAt: time.Now(),
	}
	sess2 := &users.Session{
		ID:           ulid.Make().String(),
		UserID:       u.ID,
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		CreatedAt:    time.Now(),
		LastActiveAt: time.Now(),
	}
	usersStore.CreateSession(context.Background(), sess1)
	usersStore.CreateSession(context.Background(), sess2)

	err := usersStore.DeleteUserSessions(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("DeleteUserSessions failed: %v", err)
	}

	// Both sessions should be deleted
	got1, _ := usersStore.GetSession(context.Background(), sess1.ID)
	got2, _ := usersStore.GetSession(context.Background(), sess2.ID)
	if got1 != nil || got2 != nil {
		t.Error("expected all user sessions to be deleted")
	}
}

func TestUsersStore_DeleteUserSessions_PreservesOtherUsers(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	u1 := createTestUser(t, usersStore)
	u2 := createTestUser(t, usersStore)

	sess1 := &users.Session{
		ID:           ulid.Make().String(),
		UserID:       u1.ID,
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		CreatedAt:    time.Now(),
		LastActiveAt: time.Now(),
	}
	sess2 := &users.Session{
		ID:           ulid.Make().String(),
		UserID:       u2.ID,
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		CreatedAt:    time.Now(),
		LastActiveAt: time.Now(),
	}
	usersStore.CreateSession(context.Background(), sess1)
	usersStore.CreateSession(context.Background(), sess2)

	// Delete only u1's sessions
	usersStore.DeleteUserSessions(context.Background(), u1.ID)

	// u2's session should still exist
	got2, _ := usersStore.GetSession(context.Background(), sess2.ID)
	if got2 == nil {
		t.Error("expected other user's session to be preserved")
	}
}

func TestUsersStore_DeleteExpiredSessions(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	u := createTestUser(t, usersStore)

	// Create expired session
	expiredSess := &users.Session{
		ID:           ulid.Make().String(),
		UserID:       u.ID,
		ExpiresAt:    time.Now().Add(-1 * time.Hour), // expired
		CreatedAt:    time.Now().Add(-2 * time.Hour),
		LastActiveAt: time.Now().Add(-2 * time.Hour),
	}
	usersStore.CreateSession(context.Background(), expiredSess)

	// Create valid session
	validSess := &users.Session{
		ID:           ulid.Make().String(),
		UserID:       u.ID,
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		CreatedAt:    time.Now(),
		LastActiveAt: time.Now(),
	}
	usersStore.CreateSession(context.Background(), validSess)

	err := usersStore.DeleteExpiredSessions(context.Background())
	if err != nil {
		t.Fatalf("DeleteExpiredSessions failed: %v", err)
	}

	// Expired session should be gone
	got, _ := usersStore.GetSession(context.Background(), expiredSess.ID)
	if got != nil {
		t.Error("expected expired session to be deleted")
	}

	// Valid session should still exist
	got, _ = usersStore.GetSession(context.Background(), validSess.ID)
	if got == nil {
		t.Error("expected valid session to still exist")
	}
}

func TestUsersStore_UpdateSessionActivity(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	u := createTestUser(t, usersStore)

	originalTime := time.Now().Add(-1 * time.Hour)
	sess := &users.Session{
		ID:           ulid.Make().String(),
		UserID:       u.ID,
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		CreatedAt:    originalTime,
		LastActiveAt: originalTime,
	}
	usersStore.CreateSession(context.Background(), sess)

	// Update activity
	err := usersStore.UpdateSessionActivity(context.Background(), sess.ID)
	if err != nil {
		t.Fatalf("UpdateSessionActivity failed: %v", err)
	}

	got, _ := usersStore.GetSession(context.Background(), sess.ID)
	if got == nil {
		t.Fatal("expected session")
	}
	if !got.LastActiveAt.After(originalTime) {
		t.Error("expected last_active_at to be updated")
	}
}
