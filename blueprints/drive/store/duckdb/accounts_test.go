package duckdb

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

// ============================================================
// User CRUD Tests
// ============================================================

func TestCreateUser_Success(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	user.AvatarURL = sql.NullString{String: "https://example.com/avatar.png", Valid: true}
	user.IsAdmin = true

	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	got, err := store.GetUserByID(ctx, "user1")
	if err != nil {
		t.Fatalf("get user failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected user, got nil")
	}
	if got.Email != "test@example.com" {
		t.Errorf("expected email test@example.com, got %s", got.Email)
	}
	if got.IsAdmin != true {
		t.Error("expected is_admin true")
	}
	if !got.AvatarURL.Valid || got.AvatarURL.String != "https://example.com/avatar.png" {
		t.Errorf("expected avatar URL, got %v", got.AvatarURL)
	}
}

func TestCreateUser_DuplicateEmail(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user1 := newTestUser("user1", "test@example.com")
	if err := store.CreateUser(ctx, user1); err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	user2 := newTestUser("user2", "test@example.com")
	if err := store.CreateUser(ctx, user2); err == nil {
		t.Error("expected error for duplicate email, got nil")
	}
}

func TestGetUserByID_Exists(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	got, err := store.GetUserByID(ctx, "user1")
	if err != nil {
		t.Fatalf("get user failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected user, got nil")
	}
	if got.ID != "user1" {
		t.Errorf("expected ID user1, got %s", got.ID)
	}
}

func TestGetUserByID_NotFound(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	got, err := store.GetUserByID(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("get user failed: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestGetUserByEmail_Exists(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	got, err := store.GetUserByEmail(ctx, "test@example.com")
	if err != nil {
		t.Fatalf("get user failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected user, got nil")
	}
	if got.Email != "test@example.com" {
		t.Errorf("expected email test@example.com, got %s", got.Email)
	}
}

func TestGetUserByEmail_NotFound(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	got, err := store.GetUserByEmail(ctx, "nonexistent@example.com")
	if err != nil {
		t.Fatalf("get user failed: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestUpdateUser_Success(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	user.Name = "Updated Name"
	user.Email = "updated@example.com"
	user.IsAdmin = true
	user.StorageUsed = 5000
	user.UpdatedAt = time.Now().Truncate(time.Microsecond)

	if err := store.UpdateUser(ctx, user); err != nil {
		t.Fatalf("update user failed: %v", err)
	}

	got, err := store.GetUserByID(ctx, "user1")
	if err != nil {
		t.Fatalf("get user failed: %v", err)
	}
	if got.Name != "Updated Name" {
		t.Errorf("expected name Updated Name, got %s", got.Name)
	}
	if got.Email != "updated@example.com" {
		t.Errorf("expected email updated@example.com, got %s", got.Email)
	}
	if !got.IsAdmin {
		t.Error("expected is_admin true")
	}
	if got.StorageUsed != 5000 {
		t.Errorf("expected storage_used 5000, got %d", got.StorageUsed)
	}
}

func TestDeleteUser_Success(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	if err := store.DeleteUser(ctx, "user1"); err != nil {
		t.Fatalf("delete user failed: %v", err)
	}

	got, err := store.GetUserByID(ctx, "user1")
	if err != nil {
		t.Fatalf("get user failed: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil after delete, got %+v", got)
	}
}

func TestListUsers_Empty(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	users, err := store.ListUsers(ctx)
	if err != nil {
		t.Fatalf("list users failed: %v", err)
	}
	if len(users) != 0 {
		t.Errorf("expected 0 users, got %d", len(users))
	}
}

func TestListUsers_Multiple(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	// Create users with different timestamps
	user1 := newTestUser("user1", "a@example.com")
	user1.CreatedAt = time.Now().Add(-2 * time.Hour).Truncate(time.Microsecond)
	if err := store.CreateUser(ctx, user1); err != nil {
		t.Fatalf("create user1 failed: %v", err)
	}

	user2 := newTestUser("user2", "b@example.com")
	user2.CreatedAt = time.Now().Add(-1 * time.Hour).Truncate(time.Microsecond)
	if err := store.CreateUser(ctx, user2); err != nil {
		t.Fatalf("create user2 failed: %v", err)
	}

	user3 := newTestUser("user3", "c@example.com")
	user3.CreatedAt = time.Now().Truncate(time.Microsecond)
	if err := store.CreateUser(ctx, user3); err != nil {
		t.Fatalf("create user3 failed: %v", err)
	}

	users, err := store.ListUsers(ctx)
	if err != nil {
		t.Fatalf("list users failed: %v", err)
	}
	if len(users) != 3 {
		t.Fatalf("expected 3 users, got %d", len(users))
	}

	// Should be ordered by created_at DESC (newest first)
	if users[0].ID != "user3" {
		t.Errorf("expected first user to be user3, got %s", users[0].ID)
	}
	if users[2].ID != "user1" {
		t.Errorf("expected last user to be user1, got %s", users[2].ID)
	}
}

// ============================================================
// User Storage Tracking Tests
// ============================================================

func TestUpdateUserStorageUsed_Increment(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	if err := store.UpdateUserStorageUsed(ctx, "user1", 1000); err != nil {
		t.Fatalf("update storage failed: %v", err)
	}

	got, _ := store.GetUserByID(ctx, "user1")
	if got.StorageUsed != 1000 {
		t.Errorf("expected storage_used 1000, got %d", got.StorageUsed)
	}

	// Increment again
	if err := store.UpdateUserStorageUsed(ctx, "user1", 500); err != nil {
		t.Fatalf("update storage failed: %v", err)
	}

	got, _ = store.GetUserByID(ctx, "user1")
	if got.StorageUsed != 1500 {
		t.Errorf("expected storage_used 1500, got %d", got.StorageUsed)
	}
}

func TestUpdateUserStorageUsed_Decrement(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	user.StorageUsed = 5000
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	if err := store.UpdateUserStorageUsed(ctx, "user1", -1000); err != nil {
		t.Fatalf("update storage failed: %v", err)
	}

	got, _ := store.GetUserByID(ctx, "user1")
	if got.StorageUsed != 4000 {
		t.Errorf("expected storage_used 4000, got %d", got.StorageUsed)
	}
}

// ============================================================
// Session Management Tests
// ============================================================

func TestCreateSession_Success(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	sess := newTestSession("sess1", "user1", "token123")
	sess.IPAddress = sql.NullString{String: "192.168.1.1", Valid: true}
	sess.UserAgent = sql.NullString{String: "Mozilla/5.0", Valid: true}

	if err := store.CreateSession(ctx, sess); err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	got, err := store.GetSessionByID(ctx, "sess1")
	if err != nil {
		t.Fatalf("get session failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected session, got nil")
	}
	if got.UserID != "user1" {
		t.Errorf("expected user_id user1, got %s", got.UserID)
	}
	if got.IPAddress.String != "192.168.1.1" {
		t.Errorf("expected ip 192.168.1.1, got %s", got.IPAddress.String)
	}
}

func TestGetSessionByID_Exists(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	sess := newTestSession("sess1", "user1", "token123")
	store.CreateSession(ctx, sess)

	got, err := store.GetSessionByID(ctx, "sess1")
	if err != nil {
		t.Fatalf("get session failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected session, got nil")
	}
	if got.ID != "sess1" {
		t.Errorf("expected ID sess1, got %s", got.ID)
	}
}

func TestGetSessionByID_NotFound(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	got, err := store.GetSessionByID(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("get session failed: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestGetSessionByToken_ValidNotExpired(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	sess := newTestSession("sess1", "user1", "token123")
	sess.ExpiresAt = time.Now().Add(24 * time.Hour) // Not expired
	store.CreateSession(ctx, sess)

	got, err := store.GetSessionByToken(ctx, "token123")
	if err != nil {
		t.Fatalf("get session by token failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected session, got nil")
	}
	if got.TokenHash != "token123" {
		t.Errorf("expected token_hash token123, got %s", got.TokenHash)
	}
}

func TestGetSessionByToken_Expired(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	sess := newTestSession("sess1", "user1", "token123")
	sess.ExpiresAt = time.Now().Add(-1 * time.Hour) // Already expired
	store.CreateSession(ctx, sess)

	got, err := store.GetSessionByToken(ctx, "token123")
	if err != nil {
		t.Fatalf("get session by token failed: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for expired session, got %+v", got)
	}
}

func TestUpdateSessionActivity(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	sess := newTestSession("sess1", "user1", "token123")
	store.CreateSession(ctx, sess)

	if err := store.UpdateSessionActivity(ctx, "sess1"); err != nil {
		t.Fatalf("update session activity failed: %v", err)
	}

	got, _ := store.GetSessionByID(ctx, "sess1")
	if !got.LastActiveAt.Valid {
		t.Error("expected last_active_at to be set")
	}
}

func TestDeleteSession_Success(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	sess := newTestSession("sess1", "user1", "token123")
	store.CreateSession(ctx, sess)

	if err := store.DeleteSession(ctx, "sess1"); err != nil {
		t.Fatalf("delete session failed: %v", err)
	}

	got, _ := store.GetSessionByID(ctx, "sess1")
	if got != nil {
		t.Errorf("expected nil after delete, got %+v", got)
	}
}

func TestDeleteUserSessions(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	sess1 := newTestSession("sess1", "user1", "token1")
	sess2 := newTestSession("sess2", "user1", "token2")
	sess3 := newTestSession("sess3", "user1", "token3")
	store.CreateSession(ctx, sess1)
	store.CreateSession(ctx, sess2)
	store.CreateSession(ctx, sess3)

	if err := store.DeleteUserSessions(ctx, "user1"); err != nil {
		t.Fatalf("delete user sessions failed: %v", err)
	}

	sessions, _ := store.ListUserSessions(ctx, "user1")
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}
}

func TestListUserSessions_Empty(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	sessions, err := store.ListUserSessions(ctx, "user1")
	if err != nil {
		t.Fatalf("list sessions failed: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}
}

func TestListUserSessions_Multiple(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	// Create sessions with different timestamps
	sess1 := newTestSession("sess1", "user1", "token1")
	sess1.CreatedAt = time.Now().Add(-2 * time.Hour).Truncate(time.Microsecond)
	store.CreateSession(ctx, sess1)

	sess2 := newTestSession("sess2", "user1", "token2")
	sess2.CreatedAt = time.Now().Truncate(time.Microsecond)
	store.CreateSession(ctx, sess2)

	sessions, err := store.ListUserSessions(ctx, "user1")
	if err != nil {
		t.Fatalf("list sessions failed: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}

	// Should be ordered by created_at DESC
	if sessions[0].ID != "sess2" {
		t.Errorf("expected first session to be sess2, got %s", sessions[0].ID)
	}
}

func TestCleanupExpiredSessions(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "test@example.com")
	store.CreateUser(ctx, user)

	// Create expired session
	expired := newTestSession("expired", "user1", "token1")
	expired.ExpiresAt = time.Now().Add(-1 * time.Hour)
	store.CreateSession(ctx, expired)

	// Create valid session
	valid := newTestSession("valid", "user1", "token2")
	valid.ExpiresAt = time.Now().Add(24 * time.Hour)
	store.CreateSession(ctx, valid)

	if err := store.CleanupExpiredSessions(ctx); err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}

	sessions, _ := store.ListUserSessions(ctx, "user1")
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].ID != "valid" {
		t.Errorf("expected valid session, got %s", sessions[0].ID)
	}
}

// ============================================================
// Business Use Cases - Authentication Flow
// ============================================================

func TestAuthFlow_RegisterLogin(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	// Register user
	user := newTestUser("user1", "alice@example.com")
	user.PasswordHash = "hashed_password"
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	// Login - find user by email
	found, err := store.GetUserByEmail(ctx, "alice@example.com")
	if err != nil {
		t.Fatalf("login lookup failed: %v", err)
	}
	if found == nil {
		t.Fatal("expected user for login")
	}

	// Create session after successful password check
	sess := newTestSession("sess1", found.ID, "session_token_hash")
	if err := store.CreateSession(ctx, sess); err != nil {
		t.Fatalf("session creation failed: %v", err)
	}

	// Verify session is retrievable
	gotSess, _ := store.GetSessionByToken(ctx, "session_token_hash")
	if gotSess == nil {
		t.Fatal("expected session after login")
	}
	if gotSess.UserID != "user1" {
		t.Errorf("expected session for user1, got %s", gotSess.UserID)
	}
}

func TestAuthFlow_MultipleDevices(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "alice@example.com")
	store.CreateUser(ctx, user)

	// Login from multiple devices
	devices := []struct {
		id        string
		token     string
		ip        string
		userAgent string
	}{
		{"sess1", "token1", "192.168.1.1", "Chrome on Windows"},
		{"sess2", "token2", "192.168.1.2", "Safari on iPhone"},
		{"sess3", "token3", "192.168.1.3", "Firefox on Mac"},
	}

	for _, d := range devices {
		sess := newTestSession(d.id, "user1", d.token)
		sess.IPAddress = sql.NullString{String: d.ip, Valid: true}
		sess.UserAgent = sql.NullString{String: d.userAgent, Valid: true}
		store.CreateSession(ctx, sess)
	}

	sessions, _ := store.ListUserSessions(ctx, "user1")
	if len(sessions) != 3 {
		t.Errorf("expected 3 active sessions, got %d", len(sessions))
	}
}

func TestAuthFlow_LogoutAllDevices(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "alice@example.com")
	store.CreateUser(ctx, user)

	// Create multiple sessions
	for i := 1; i <= 5; i++ {
		sess := newTestSession("sess"+string(rune('0'+i)), "user1", "token"+string(rune('0'+i)))
		store.CreateSession(ctx, sess)
	}

	// Logout all devices
	if err := store.DeleteUserSessions(ctx, "user1"); err != nil {
		t.Fatalf("logout all failed: %v", err)
	}

	sessions, _ := store.ListUserSessions(ctx, "user1")
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions after logout all, got %d", len(sessions))
	}
}

func TestAuthFlow_SessionRefresh(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "alice@example.com")
	store.CreateUser(ctx, user)

	sess := newTestSession("sess1", "user1", "token123")
	store.CreateSession(ctx, sess)

	// Initial state - no last_active_at
	got, _ := store.GetSessionByID(ctx, "sess1")
	if got.LastActiveAt.Valid {
		t.Error("expected no last_active_at initially")
	}

	// User makes API request - refresh activity
	store.UpdateSessionActivity(ctx, "sess1")

	// Verify activity updated
	got, _ = store.GetSessionByID(ctx, "sess1")
	if !got.LastActiveAt.Valid {
		t.Error("expected last_active_at after refresh")
	}
}
