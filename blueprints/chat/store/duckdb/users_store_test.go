package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/chat/feature/accounts"
)

func TestUsersStore_Insert(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewUsersStore(db)
	ctx := context.Background()

	user := &accounts.User{
		ID:            "user1",
		Username:      "testuser",
		Discriminator: "0001",
		DisplayName:   "Test User",
		Email:         "test@example.com",
		Status:        accounts.StatusOnline,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	err := store.Insert(ctx, user, "hashedpassword")
	if err != nil {
		t.Fatalf("Insert() error = %v", err)
	}

	// Verify user was inserted
	got, err := store.GetByID(ctx, "user1")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if got.Username != user.Username {
		t.Errorf("Username = %v, want %v", got.Username, user.Username)
	}
	if got.Email != user.Email {
		t.Errorf("Email = %v, want %v", got.Email, user.Email)
	}
}

func TestUsersStore_GetByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewUsersStore(db)
	ctx := context.Background()

	// Create test user
	user := createTestUser(t, store, "testuser")

	// Test getting existing user
	got, err := store.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if got.ID != user.ID {
		t.Errorf("ID = %v, want %v", got.ID, user.ID)
	}

	// Test getting non-existent user
	_, err = store.GetByID(ctx, "nonexistent")
	if err != accounts.ErrNotFound {
		t.Errorf("GetByID() error = %v, want ErrNotFound", err)
	}
}

func TestUsersStore_GetByIDs(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewUsersStore(db)
	ctx := context.Background()

	// Create test users
	user1 := createTestUser(t, store, "user1")
	user2 := createTestUser(t, store, "user2")

	// Test getting multiple users
	users, err := store.GetByIDs(ctx, []string{user1.ID, user2.ID})
	if err != nil {
		t.Fatalf("GetByIDs() error = %v", err)
	}

	if len(users) != 2 {
		t.Errorf("len(users) = %d, want 2", len(users))
	}

	// Test empty slice
	users, err = store.GetByIDs(ctx, []string{})
	if err != nil {
		t.Fatalf("GetByIDs() with empty slice error = %v", err)
	}
	if users != nil {
		t.Errorf("expected nil for empty slice, got %v", users)
	}
}

func TestUsersStore_GetByUsername(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewUsersStore(db)
	ctx := context.Background()

	user := createTestUser(t, store, "testuser")

	got, err := store.GetByUsername(ctx, "testuser")
	if err != nil {
		t.Fatalf("GetByUsername() error = %v", err)
	}

	if got.ID != user.ID {
		t.Errorf("ID = %v, want %v", got.ID, user.ID)
	}

	// Test non-existent
	_, err = store.GetByUsername(ctx, "nonexistent")
	if err != accounts.ErrNotFound {
		t.Errorf("GetByUsername() error = %v, want ErrNotFound", err)
	}
}

func TestUsersStore_GetByEmail(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewUsersStore(db)
	ctx := context.Background()

	user := createTestUser(t, store, "testuser")

	got, err := store.GetByEmail(ctx, "testuser@example.com")
	if err != nil {
		t.Fatalf("GetByEmail() error = %v", err)
	}

	if got.ID != user.ID {
		t.Errorf("ID = %v, want %v", got.ID, user.ID)
	}

	// Test non-existent
	_, err = store.GetByEmail(ctx, "nonexistent@example.com")
	if err != accounts.ErrNotFound {
		t.Errorf("GetByEmail() error = %v, want ErrNotFound", err)
	}
}

func TestUsersStore_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewUsersStore(db)
	ctx := context.Background()

	user := createTestUser(t, store, "testuser")

	// Update display name
	newName := "Updated Name"
	err := store.Update(ctx, user.ID, &accounts.UpdateIn{
		DisplayName: &newName,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got, _ := store.GetByID(ctx, user.ID)
	if got.DisplayName != newName {
		t.Errorf("DisplayName = %v, want %v", got.DisplayName, newName)
	}

	// Update with empty input should be no-op
	err = store.Update(ctx, user.ID, &accounts.UpdateIn{})
	if err != nil {
		t.Errorf("Update() with empty input error = %v", err)
	}
}

func TestUsersStore_ExistsUsername(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewUsersStore(db)
	ctx := context.Background()

	createTestUser(t, store, "existinguser")

	tests := []struct {
		username string
		want     bool
	}{
		{"existinguser", true},
		{"nonexistent", false},
	}

	for _, tt := range tests {
		exists, err := store.ExistsUsername(ctx, tt.username)
		if err != nil {
			t.Errorf("ExistsUsername(%s) error = %v", tt.username, err)
		}
		if exists != tt.want {
			t.Errorf("ExistsUsername(%s) = %v, want %v", tt.username, exists, tt.want)
		}
	}
}

func TestUsersStore_ExistsEmail(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewUsersStore(db)
	ctx := context.Background()

	createTestUser(t, store, "testuser")

	tests := []struct {
		email string
		want  bool
	}{
		{"testuser@example.com", true},
		{"nonexistent@example.com", false},
	}

	for _, tt := range tests {
		exists, err := store.ExistsEmail(ctx, tt.email)
		if err != nil {
			t.Errorf("ExistsEmail(%s) error = %v", tt.email, err)
		}
		if exists != tt.want {
			t.Errorf("ExistsEmail(%s) = %v, want %v", tt.email, exists, tt.want)
		}
	}
}

func TestUsersStore_GetPasswordHash(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewUsersStore(db)
	ctx := context.Background()

	user := createTestUser(t, store, "testuser")

	// Test by username
	id, hash, err := store.GetPasswordHash(ctx, "testuser")
	if err != nil {
		t.Fatalf("GetPasswordHash() error = %v", err)
	}
	if id != user.ID {
		t.Errorf("ID = %v, want %v", id, user.ID)
	}
	if hash != "hashedpassword123" {
		t.Errorf("hash = %v, want hashedpassword123", hash)
	}

	// Test by email
	id, _, err = store.GetPasswordHash(ctx, "testuser@example.com")
	if err != nil {
		t.Fatalf("GetPasswordHash() by email error = %v", err)
	}
	if id != user.ID {
		t.Errorf("ID = %v, want %v", id, user.ID)
	}

	// Test non-existent
	_, _, err = store.GetPasswordHash(ctx, "nonexistent")
	if err != accounts.ErrNotFound {
		t.Errorf("GetPasswordHash() error = %v, want ErrNotFound", err)
	}
}

func TestUsersStore_Search(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewUsersStore(db)
	ctx := context.Background()

	createTestUser(t, store, "alice")
	createTestUser(t, store, "bob")
	createTestUser(t, store, "charlie")

	// Search for 'ali'
	users, err := store.Search(ctx, "ali", 10)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(users) != 1 {
		t.Errorf("len(users) = %d, want 1", len(users))
	}

	if len(users) > 0 && users[0].Username != "alice" {
		t.Errorf("Username = %v, want alice", users[0].Username)
	}
}

func TestUsersStore_GetNextDiscriminator(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewUsersStore(db)
	ctx := context.Background()

	// First user with username
	disc, err := store.GetNextDiscriminator(ctx, "testuser")
	if err != nil {
		t.Fatalf("GetNextDiscriminator() error = %v", err)
	}
	if disc != "0001" {
		t.Errorf("disc = %v, want 0001", disc)
	}

	// Create user with that discriminator
	createTestUser(t, store, "testuser")

	// Next should be 0002
	disc, err = store.GetNextDiscriminator(ctx, "testuser")
	if err != nil {
		t.Fatalf("GetNextDiscriminator() error = %v", err)
	}
	if disc != "0002" {
		t.Errorf("disc = %v, want 0002", disc)
	}
}

func TestUsersStore_Session(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewUsersStore(db)
	ctx := context.Background()

	user := createTestUser(t, store, "testuser")

	// Create session
	session := &accounts.Session{
		ID:        "sess1",
		UserID:    user.ID,
		Token:     "token123",
		UserAgent: "Test Agent",
		IPAddress: "127.0.0.1",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}

	err := store.CreateSession(ctx, session)
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	// Get session
	got, err := store.GetSession(ctx, "token123")
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}

	if got.UserID != user.ID {
		t.Errorf("UserID = %v, want %v", got.UserID, user.ID)
	}

	// Get invalid session
	_, err = store.GetSession(ctx, "invalidtoken")
	if err != accounts.ErrInvalidSession {
		t.Errorf("GetSession() error = %v, want ErrInvalidSession", err)
	}

	// Delete session
	err = store.DeleteSession(ctx, "token123")
	if err != nil {
		t.Fatalf("DeleteSession() error = %v", err)
	}

	// Verify deleted
	_, err = store.GetSession(ctx, "token123")
	if err != accounts.ErrInvalidSession {
		t.Errorf("GetSession() after delete error = %v, want ErrInvalidSession", err)
	}
}

func TestUsersStore_DeleteExpiredSessions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewUsersStore(db)
	ctx := context.Background()

	user := createTestUser(t, store, "testuser")

	// Create expired session
	expiredSession := &accounts.Session{
		ID:        "sess-expired",
		UserID:    user.ID,
		Token:     "expired-token",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
		CreatedAt: time.Now().Add(-2 * time.Hour),
	}
	store.CreateSession(ctx, expiredSession)

	// Create valid session
	validSession := &accounts.Session{
		ID:        "sess-valid",
		UserID:    user.ID,
		Token:     "valid-token",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}
	store.CreateSession(ctx, validSession)

	// Delete expired
	err := store.DeleteExpiredSessions(ctx)
	if err != nil {
		t.Fatalf("DeleteExpiredSessions() error = %v", err)
	}

	// Valid should still work
	_, err = store.GetSession(ctx, "valid-token")
	if err != nil {
		t.Errorf("valid session should still exist: %v", err)
	}
}

func TestUsersStore_UpdateStatus(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewUsersStore(db)
	ctx := context.Background()

	user := createTestUser(t, store, "testuser")

	err := store.UpdateStatus(ctx, user.ID, "idle")
	if err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}

	got, _ := store.GetByID(ctx, user.ID)
	if string(got.Status) != "idle" {
		t.Errorf("Status = %v, want idle", got.Status)
	}
}
