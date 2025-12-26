package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/kanban/feature/users"
	"github.com/oklog/ulid/v2"
)

func createTestUser(t *testing.T, store *UsersStore) *users.User {
	t.Helper()
	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "test" + ulid.Make().String() + "@example.com",
		Username:     "user" + ulid.Make().String()[:8],
		DisplayName:  "Test User",
		PasswordHash: "hashed_password",
	}
	if err := store.Create(context.Background(), u); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return u
}

func TestUsersStore_Create(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "test@example.com",
		Username:     "testuser",
		DisplayName:  "Test User",
		PasswordHash: "hashed",
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
}

func TestUsersStore_Create_DuplicateEmail(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	u1 := &users.User{
		ID:           ulid.Make().String(),
		Email:        "dup@example.com",
		Username:     "user1",
		DisplayName:  "User 1",
		PasswordHash: "hashed",
	}
	u2 := &users.User{
		ID:           ulid.Make().String(),
		Email:        "dup@example.com", // same email
		Username:     "user2",
		DisplayName:  "User 2",
		PasswordHash: "hashed",
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
		Email:        "user1@example.com",
		Username:     "dupuser",
		DisplayName:  "User 1",
		PasswordHash: "hashed",
	}
	u2 := &users.User{
		ID:           ulid.Make().String(),
		Email:        "user2@example.com",
		Username:     "dupuser", // same username
		DisplayName:  "User 2",
		PasswordHash: "hashed",
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

func TestUsersStore_Update(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	u := createTestUser(t, usersStore)

	newName := "Updated Name"
	err := usersStore.Update(context.Background(), u.ID, &users.UpdateIn{
		DisplayName: &newName,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, _ := usersStore.GetByID(context.Background(), u.ID)
	if got.DisplayName != newName {
		t.Errorf("got display name %q, want %q", got.DisplayName, newName)
	}
}

func TestUsersStore_UpdatePassword(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	u := createTestUser(t, usersStore)

	newHash := "new_password_hash"
	err := usersStore.UpdatePassword(context.Background(), u.ID, newHash)
	if err != nil {
		t.Fatalf("UpdatePassword failed: %v", err)
	}

	got, _ := usersStore.GetByID(context.Background(), u.ID)
	if got.PasswordHash != newHash {
		t.Errorf("password hash not updated")
	}
}

func TestUsersStore_CreateSession(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	u := createTestUser(t, usersStore)

	sess := &users.Session{
		ID:        ulid.Make().String(),
		UserID:    u.ID,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
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
		ID:        ulid.Make().String(),
		UserID:    u.ID,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
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

func TestUsersStore_DeleteExpiredSessions(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	u := createTestUser(t, usersStore)

	// Create expired session
	expiredSess := &users.Session{
		ID:        ulid.Make().String(),
		UserID:    u.ID,
		ExpiresAt: time.Now().Add(-1 * time.Hour), // expired
		CreatedAt: time.Now().Add(-2 * time.Hour),
	}
	usersStore.CreateSession(context.Background(), expiredSess)

	// Create valid session
	validSess := &users.Session{
		ID:        ulid.Make().String(),
		UserID:    u.ID,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
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
