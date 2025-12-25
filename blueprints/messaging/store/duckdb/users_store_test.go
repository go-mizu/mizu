package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/messaging/feature/accounts"
)

func testUsersStore(t *testing.T) *UsersStore {
	t.Helper()
	store := testStore(t)
	return NewUsersStore(store.DB())
}

func createTestUser(t *testing.T, s *UsersStore, suffix string) *accounts.User {
	t.Helper()
	now := time.Now()
	u := &accounts.User{
		ID:                  "user_" + suffix,
		Phone:               "+1234567890" + suffix,
		Email:               "test" + suffix + "@example.com",
		Username:            "testuser" + suffix,
		DisplayName:         "Test User " + suffix,
		Bio:                 "A test bio",
		AvatarURL:           "https://example.com/avatar.png",
		Status:              "Available",
		PrivacyLastSeen:     "everyone",
		PrivacyProfilePhoto: "everyone",
		PrivacyAbout:        "everyone",
		PrivacyGroups:       "everyone",
		PrivacyReadReceipts: true,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	if err := s.Insert(context.Background(), u, "hashed_password"); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return u
}

func TestNewUsersStore(t *testing.T) {
	store := testStore(t)
	us := NewUsersStore(store.DB())

	if us == nil {
		t.Fatal("NewUsersStore() returned nil")
	}
	if us.db == nil {
		t.Fatal("UsersStore.db is nil")
	}
}

func TestUsersStore_Insert(t *testing.T) {
	s := testUsersStore(t)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		now := time.Now()
		u := &accounts.User{
			ID:                  "user_insert_1",
			Phone:               "+1234567890",
			Email:               "insert@example.com",
			Username:            "insertuser",
			DisplayName:         "Insert User",
			Bio:                 "Bio here",
			AvatarURL:           "https://example.com/avatar.png",
			Status:              "Online",
			PrivacyLastSeen:     "contacts",
			PrivacyProfilePhoto: "contacts",
			PrivacyAbout:        "contacts",
			PrivacyGroups:       "contacts",
			PrivacyReadReceipts: false,
			CreatedAt:           now,
			UpdatedAt:           now,
		}

		err := s.Insert(ctx, u, "password_hash_123")
		if err != nil {
			t.Fatalf("Insert() returned error: %v", err)
		}

		// Verify user was inserted
		retrieved, err := s.GetByID(ctx, u.ID)
		if err != nil {
			t.Fatalf("failed to retrieve inserted user: %v", err)
		}
		if retrieved.Username != u.Username {
			t.Errorf("Username = %v, want %v", retrieved.Username, u.Username)
		}
		if retrieved.DisplayName != u.DisplayName {
			t.Errorf("DisplayName = %v, want %v", retrieved.DisplayName, u.DisplayName)
		}
	})

	t.Run("with optional fields empty", func(t *testing.T) {
		now := time.Now()
		u := &accounts.User{
			ID:                  "user_insert_2",
			Username:            "minimaluser",
			DisplayName:         "Minimal User",
			PrivacyLastSeen:     "everyone",
			PrivacyProfilePhoto: "everyone",
			PrivacyAbout:        "everyone",
			PrivacyGroups:       "everyone",
			PrivacyReadReceipts: true,
			CreatedAt:           now,
			UpdatedAt:           now,
		}

		err := s.Insert(ctx, u, "password_hash")
		if err != nil {
			t.Fatalf("Insert() with minimal fields returned error: %v", err)
		}
	})

	t.Run("duplicate username fails", func(t *testing.T) {
		now := time.Now()
		u1 := &accounts.User{
			ID:                  "user_dup_1",
			Username:            "duplicateuser",
			DisplayName:         "Duplicate User",
			PrivacyLastSeen:     "everyone",
			PrivacyProfilePhoto: "everyone",
			PrivacyAbout:        "everyone",
			PrivacyGroups:       "everyone",
			PrivacyReadReceipts: true,
			CreatedAt:           now,
			UpdatedAt:           now,
		}
		if err := s.Insert(ctx, u1, "hash"); err != nil {
			t.Fatalf("first Insert() failed: %v", err)
		}

		u2 := &accounts.User{
			ID:                  "user_dup_2",
			Username:            "duplicateuser",
			DisplayName:         "Another User",
			PrivacyLastSeen:     "everyone",
			PrivacyProfilePhoto: "everyone",
			PrivacyAbout:        "everyone",
			PrivacyGroups:       "everyone",
			PrivacyReadReceipts: true,
			CreatedAt:           now,
			UpdatedAt:           now,
		}
		err := s.Insert(ctx, u2, "hash")
		if err == nil {
			t.Fatal("Insert() with duplicate username should fail")
		}
	})
}

func TestUsersStore_GetByID(t *testing.T) {
	s := testUsersStore(t)
	ctx := context.Background()

	t.Run("existing user", func(t *testing.T) {
		u := createTestUser(t, s, "getbyid")
		retrieved, err := s.GetByID(ctx, u.ID)
		if err != nil {
			t.Fatalf("GetByID() returned error: %v", err)
		}

		if retrieved.ID != u.ID {
			t.Errorf("ID = %v, want %v", retrieved.ID, u.ID)
		}
		if retrieved.Username != u.Username {
			t.Errorf("Username = %v, want %v", retrieved.Username, u.Username)
		}
		if retrieved.Email != u.Email {
			t.Errorf("Email = %v, want %v", retrieved.Email, u.Email)
		}
		if retrieved.Phone != u.Phone {
			t.Errorf("Phone = %v, want %v", retrieved.Phone, u.Phone)
		}
	})

	t.Run("non-existing user", func(t *testing.T) {
		_, err := s.GetByID(ctx, "non_existing_id")
		if err != accounts.ErrNotFound {
			t.Errorf("GetByID() error = %v, want %v", err, accounts.ErrNotFound)
		}
	})
}

func TestUsersStore_GetByIDs(t *testing.T) {
	s := testUsersStore(t)
	ctx := context.Background()

	t.Run("multiple users", func(t *testing.T) {
		u1 := createTestUser(t, s, "getbyids1")
		u2 := createTestUser(t, s, "getbyids2")
		u3 := createTestUser(t, s, "getbyids3")

		users, err := s.GetByIDs(ctx, []string{u1.ID, u2.ID, u3.ID})
		if err != nil {
			t.Fatalf("GetByIDs() returned error: %v", err)
		}

		if len(users) != 3 {
			t.Errorf("len(users) = %v, want 3", len(users))
		}
	})

	t.Run("empty list", func(t *testing.T) {
		users, err := s.GetByIDs(ctx, []string{})
		if err != nil {
			t.Fatalf("GetByIDs() with empty list returned error: %v", err)
		}
		if users != nil {
			t.Errorf("GetByIDs() with empty list should return nil, got %v", users)
		}
	})

	t.Run("partial match", func(t *testing.T) {
		u := createTestUser(t, s, "getbyidspartial")
		users, err := s.GetByIDs(ctx, []string{u.ID, "nonexistent"})
		if err != nil {
			t.Fatalf("GetByIDs() returned error: %v", err)
		}
		if len(users) != 1 {
			t.Errorf("len(users) = %v, want 1", len(users))
		}
	})
}

func TestUsersStore_GetByUsername(t *testing.T) {
	s := testUsersStore(t)
	ctx := context.Background()

	t.Run("existing user", func(t *testing.T) {
		u := createTestUser(t, s, "getbyusername")
		retrieved, err := s.GetByUsername(ctx, u.Username)
		if err != nil {
			t.Fatalf("GetByUsername() returned error: %v", err)
		}
		if retrieved.ID != u.ID {
			t.Errorf("ID = %v, want %v", retrieved.ID, u.ID)
		}
	})

	t.Run("non-existing user", func(t *testing.T) {
		_, err := s.GetByUsername(ctx, "nonexistent_username")
		if err != accounts.ErrNotFound {
			t.Errorf("GetByUsername() error = %v, want %v", err, accounts.ErrNotFound)
		}
	})
}

func TestUsersStore_GetByPhone(t *testing.T) {
	s := testUsersStore(t)
	ctx := context.Background()

	t.Run("existing user", func(t *testing.T) {
		u := createTestUser(t, s, "getbyphone")
		retrieved, err := s.GetByPhone(ctx, u.Phone)
		if err != nil {
			t.Fatalf("GetByPhone() returned error: %v", err)
		}
		if retrieved.ID != u.ID {
			t.Errorf("ID = %v, want %v", retrieved.ID, u.ID)
		}
	})

	t.Run("non-existing phone", func(t *testing.T) {
		_, err := s.GetByPhone(ctx, "+9999999999")
		if err != accounts.ErrNotFound {
			t.Errorf("GetByPhone() error = %v, want %v", err, accounts.ErrNotFound)
		}
	})
}

func TestUsersStore_GetByEmail(t *testing.T) {
	s := testUsersStore(t)
	ctx := context.Background()

	t.Run("existing user", func(t *testing.T) {
		u := createTestUser(t, s, "getbyemail")
		retrieved, err := s.GetByEmail(ctx, u.Email)
		if err != nil {
			t.Fatalf("GetByEmail() returned error: %v", err)
		}
		if retrieved.ID != u.ID {
			t.Errorf("ID = %v, want %v", retrieved.ID, u.ID)
		}
	})

	t.Run("non-existing email", func(t *testing.T) {
		_, err := s.GetByEmail(ctx, "nonexistent@example.com")
		if err != accounts.ErrNotFound {
			t.Errorf("GetByEmail() error = %v, want %v", err, accounts.ErrNotFound)
		}
	})
}

func TestUsersStore_Update(t *testing.T) {
	s := testUsersStore(t)
	ctx := context.Background()

	t.Run("update display name", func(t *testing.T) {
		u := createTestUser(t, s, "updatename")
		newName := "Updated Name"
		err := s.Update(ctx, u.ID, &accounts.UpdateIn{
			DisplayName: &newName,
		})
		if err != nil {
			t.Fatalf("Update() returned error: %v", err)
		}

		retrieved, _ := s.GetByID(ctx, u.ID)
		if retrieved.DisplayName != newName {
			t.Errorf("DisplayName = %v, want %v", retrieved.DisplayName, newName)
		}
	})

	t.Run("update multiple fields", func(t *testing.T) {
		u := createTestUser(t, s, "updatemulti")
		newBio := "New bio"
		newStatus := "Busy"
		newPrivacy := "contacts"
		err := s.Update(ctx, u.ID, &accounts.UpdateIn{
			Bio:             &newBio,
			Status:          &newStatus,
			PrivacyLastSeen: &newPrivacy,
		})
		if err != nil {
			t.Fatalf("Update() returned error: %v", err)
		}

		retrieved, _ := s.GetByID(ctx, u.ID)
		if retrieved.Bio != newBio {
			t.Errorf("Bio = %v, want %v", retrieved.Bio, newBio)
		}
		if retrieved.Status != newStatus {
			t.Errorf("Status = %v, want %v", retrieved.Status, newStatus)
		}
		if retrieved.PrivacyLastSeen != newPrivacy {
			t.Errorf("PrivacyLastSeen = %v, want %v", retrieved.PrivacyLastSeen, newPrivacy)
		}
	})

	t.Run("empty update", func(t *testing.T) {
		u := createTestUser(t, s, "updateempty")
		err := s.Update(ctx, u.ID, &accounts.UpdateIn{})
		if err != nil {
			t.Fatalf("Update() with no changes returned error: %v", err)
		}
	})

	t.Run("update privacy settings", func(t *testing.T) {
		u := createTestUser(t, s, "updateprivacy")
		readReceipts := false
		err := s.Update(ctx, u.ID, &accounts.UpdateIn{
			PrivacyReadReceipts: &readReceipts,
		})
		if err != nil {
			t.Fatalf("Update() returned error: %v", err)
		}

		retrieved, _ := s.GetByID(ctx, u.ID)
		if retrieved.PrivacyReadReceipts != readReceipts {
			t.Errorf("PrivacyReadReceipts = %v, want %v", retrieved.PrivacyReadReceipts, readReceipts)
		}
	})
}

func TestUsersStore_Delete(t *testing.T) {
	s := testUsersStore(t)
	ctx := context.Background()

	t.Run("existing user", func(t *testing.T) {
		u := createTestUser(t, s, "delete")
		err := s.Delete(ctx, u.ID)
		if err != nil {
			t.Fatalf("Delete() returned error: %v", err)
		}

		_, err = s.GetByID(ctx, u.ID)
		if err != accounts.ErrNotFound {
			t.Errorf("GetByID() after delete error = %v, want %v", err, accounts.ErrNotFound)
		}
	})

	t.Run("non-existing user", func(t *testing.T) {
		err := s.Delete(ctx, "nonexistent")
		if err != nil {
			t.Fatalf("Delete() non-existing user should not error: %v", err)
		}
	})
}

func TestUsersStore_ExistsUsername(t *testing.T) {
	s := testUsersStore(t)
	ctx := context.Background()

	u := createTestUser(t, s, "existsuser")

	t.Run("existing username", func(t *testing.T) {
		exists, err := s.ExistsUsername(ctx, u.Username)
		if err != nil {
			t.Fatalf("ExistsUsername() returned error: %v", err)
		}
		if !exists {
			t.Error("ExistsUsername() = false, want true")
		}
	})

	t.Run("non-existing username", func(t *testing.T) {
		exists, err := s.ExistsUsername(ctx, "nonexistent_username")
		if err != nil {
			t.Fatalf("ExistsUsername() returned error: %v", err)
		}
		if exists {
			t.Error("ExistsUsername() = true, want false")
		}
	})
}

func TestUsersStore_ExistsPhone(t *testing.T) {
	s := testUsersStore(t)
	ctx := context.Background()

	u := createTestUser(t, s, "existsphone")

	t.Run("existing phone", func(t *testing.T) {
		exists, err := s.ExistsPhone(ctx, u.Phone)
		if err != nil {
			t.Fatalf("ExistsPhone() returned error: %v", err)
		}
		if !exists {
			t.Error("ExistsPhone() = false, want true")
		}
	})

	t.Run("non-existing phone", func(t *testing.T) {
		exists, err := s.ExistsPhone(ctx, "+0000000000")
		if err != nil {
			t.Fatalf("ExistsPhone() returned error: %v", err)
		}
		if exists {
			t.Error("ExistsPhone() = true, want false")
		}
	})
}

func TestUsersStore_ExistsEmail(t *testing.T) {
	s := testUsersStore(t)
	ctx := context.Background()

	u := createTestUser(t, s, "existsemail")

	t.Run("existing email", func(t *testing.T) {
		exists, err := s.ExistsEmail(ctx, u.Email)
		if err != nil {
			t.Fatalf("ExistsEmail() returned error: %v", err)
		}
		if !exists {
			t.Error("ExistsEmail() = false, want true")
		}
	})

	t.Run("non-existing email", func(t *testing.T) {
		exists, err := s.ExistsEmail(ctx, "nonexistent@example.com")
		if err != nil {
			t.Fatalf("ExistsEmail() returned error: %v", err)
		}
		if exists {
			t.Error("ExistsEmail() = true, want false")
		}
	})
}

func TestUsersStore_GetPasswordHash(t *testing.T) {
	s := testUsersStore(t)
	ctx := context.Background()

	now := time.Now()
	u := &accounts.User{
		ID:                  "user_pwdhash",
		Phone:               "+1111111111",
		Email:               "pwdhash@example.com",
		Username:            "pwdhashuser",
		DisplayName:         "Password Hash User",
		PrivacyLastSeen:     "everyone",
		PrivacyProfilePhoto: "everyone",
		PrivacyAbout:        "everyone",
		PrivacyGroups:       "everyone",
		PrivacyReadReceipts: true,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	expectedHash := "secret_hash_123"
	s.Insert(ctx, u, expectedHash)

	t.Run("by username", func(t *testing.T) {
		id, hash, err := s.GetPasswordHash(ctx, u.Username)
		if err != nil {
			t.Fatalf("GetPasswordHash() returned error: %v", err)
		}
		if id != u.ID {
			t.Errorf("id = %v, want %v", id, u.ID)
		}
		if hash != expectedHash {
			t.Errorf("hash = %v, want %v", hash, expectedHash)
		}
	})

	t.Run("by email", func(t *testing.T) {
		id, hash, err := s.GetPasswordHash(ctx, u.Email)
		if err != nil {
			t.Fatalf("GetPasswordHash() returned error: %v", err)
		}
		if id != u.ID {
			t.Errorf("id = %v, want %v", id, u.ID)
		}
		if hash != expectedHash {
			t.Errorf("hash = %v, want %v", hash, expectedHash)
		}
	})

	t.Run("by phone", func(t *testing.T) {
		id, hash, err := s.GetPasswordHash(ctx, u.Phone)
		if err != nil {
			t.Fatalf("GetPasswordHash() returned error: %v", err)
		}
		if id != u.ID {
			t.Errorf("id = %v, want %v", id, u.ID)
		}
		if hash != expectedHash {
			t.Errorf("hash = %v, want %v", hash, expectedHash)
		}
	})

	t.Run("non-existing user", func(t *testing.T) {
		_, _, err := s.GetPasswordHash(ctx, "nonexistent")
		if err != accounts.ErrNotFound {
			t.Errorf("GetPasswordHash() error = %v, want %v", err, accounts.ErrNotFound)
		}
	})
}

func TestUsersStore_Search(t *testing.T) {
	s := testUsersStore(t)
	ctx := context.Background()

	// Create test users for search
	createTestUser(t, s, "searchalpha")
	createTestUser(t, s, "searchbeta")
	createTestUser(t, s, "searchgamma")

	t.Run("by username prefix", func(t *testing.T) {
		users, err := s.Search(ctx, "testusersearch", 10)
		if err != nil {
			t.Fatalf("Search() returned error: %v", err)
		}
		if len(users) < 3 {
			t.Errorf("len(users) = %v, want >= 3", len(users))
		}
	})

	t.Run("by display name", func(t *testing.T) {
		users, err := s.Search(ctx, "Test User", 10)
		if err != nil {
			t.Fatalf("Search() returned error: %v", err)
		}
		if len(users) < 1 {
			t.Error("Search() should find users by display name")
		}
	})

	t.Run("no results", func(t *testing.T) {
		users, err := s.Search(ctx, "xyznonexistent123", 10)
		if err != nil {
			t.Fatalf("Search() returned error: %v", err)
		}
		if len(users) != 0 {
			t.Errorf("len(users) = %v, want 0", len(users))
		}
	})

	t.Run("with limit", func(t *testing.T) {
		users, err := s.Search(ctx, "testuser", 2)
		if err != nil {
			t.Fatalf("Search() returned error: %v", err)
		}
		if len(users) > 2 {
			t.Errorf("len(users) = %v, want <= 2", len(users))
		}
	})
}

func TestUsersStore_UpdateOnlineStatus(t *testing.T) {
	s := testUsersStore(t)
	ctx := context.Background()

	u := createTestUser(t, s, "onlinestatus")

	t.Run("set online", func(t *testing.T) {
		err := s.UpdateOnlineStatus(ctx, u.ID, true)
		if err != nil {
			t.Fatalf("UpdateOnlineStatus() returned error: %v", err)
		}

		retrieved, _ := s.GetByID(ctx, u.ID)
		if !retrieved.IsOnline {
			t.Error("IsOnline = false, want true")
		}
	})

	t.Run("set offline", func(t *testing.T) {
		err := s.UpdateOnlineStatus(ctx, u.ID, false)
		if err != nil {
			t.Fatalf("UpdateOnlineStatus() returned error: %v", err)
		}

		retrieved, _ := s.GetByID(ctx, u.ID)
		if retrieved.IsOnline {
			t.Error("IsOnline = true, want false")
		}
	})
}

func TestUsersStore_UpdateLastSeen(t *testing.T) {
	s := testUsersStore(t)
	ctx := context.Background()

	u := createTestUser(t, s, "lastseen")

	before := time.Now()
	err := s.UpdateLastSeen(ctx, u.ID)
	if err != nil {
		t.Fatalf("UpdateLastSeen() returned error: %v", err)
	}
	after := time.Now()

	retrieved, _ := s.GetByID(ctx, u.ID)
	if retrieved.LastSeenAt.Before(before) || retrieved.LastSeenAt.After(after) {
		t.Errorf("LastSeenAt = %v, want between %v and %v", retrieved.LastSeenAt, before, after)
	}
}

func TestUsersStore_Sessions(t *testing.T) {
	s := testUsersStore(t)
	ctx := context.Background()

	u := createTestUser(t, s, "sessions")

	now := time.Now()
	sess := &accounts.Session{
		ID:           "sess_1",
		UserID:       u.ID,
		Token:        "token_abc123",
		DeviceName:   "iPhone 15",
		DeviceType:   "mobile",
		PushToken:    "push_token_123",
		IPAddress:    "192.168.1.1",
		UserAgent:    "Mozilla/5.0",
		LastActiveAt: now,
		ExpiresAt:    now.Add(30 * 24 * time.Hour),
		CreatedAt:    now,
	}

	t.Run("create session", func(t *testing.T) {
		err := s.CreateSession(ctx, sess)
		if err != nil {
			t.Fatalf("CreateSession() returned error: %v", err)
		}
	})

	t.Run("get session", func(t *testing.T) {
		retrieved, err := s.GetSession(ctx, sess.Token)
		if err != nil {
			t.Fatalf("GetSession() returned error: %v", err)
		}
		if retrieved.ID != sess.ID {
			t.Errorf("ID = %v, want %v", retrieved.ID, sess.ID)
		}
		if retrieved.UserID != sess.UserID {
			t.Errorf("UserID = %v, want %v", retrieved.UserID, sess.UserID)
		}
		if retrieved.DeviceName != sess.DeviceName {
			t.Errorf("DeviceName = %v, want %v", retrieved.DeviceName, sess.DeviceName)
		}
	})

	t.Run("get non-existing session", func(t *testing.T) {
		_, err := s.GetSession(ctx, "nonexistent_token")
		if err != accounts.ErrInvalidSession {
			t.Errorf("GetSession() error = %v, want %v", err, accounts.ErrInvalidSession)
		}
	})

	t.Run("get expired session", func(t *testing.T) {
		expiredSess := &accounts.Session{
			ID:           "sess_expired",
			UserID:       u.ID,
			Token:        "expired_token",
			LastActiveAt: now,
			ExpiresAt:    now.Add(-1 * time.Hour), // Expired
			CreatedAt:    now,
		}
		s.CreateSession(ctx, expiredSess)

		_, err := s.GetSession(ctx, expiredSess.Token)
		if err != accounts.ErrInvalidSession {
			t.Errorf("GetSession() for expired session error = %v, want %v", err, accounts.ErrInvalidSession)
		}
	})

	t.Run("delete session", func(t *testing.T) {
		err := s.DeleteSession(ctx, sess.Token)
		if err != nil {
			t.Fatalf("DeleteSession() returned error: %v", err)
		}

		_, err = s.GetSession(ctx, sess.Token)
		if err != accounts.ErrInvalidSession {
			t.Error("session should be deleted")
		}
	})

	t.Run("delete all sessions", func(t *testing.T) {
		// Create multiple sessions
		for i := 0; i < 3; i++ {
			s.CreateSession(ctx, &accounts.Session{
				ID:           "sess_all_" + string(rune('a'+i)),
				UserID:       u.ID,
				Token:        "token_all_" + string(rune('a'+i)),
				LastActiveAt: now,
				ExpiresAt:    now.Add(24 * time.Hour),
				CreatedAt:    now,
			})
		}

		err := s.DeleteAllSessions(ctx, u.ID)
		if err != nil {
			t.Fatalf("DeleteAllSessions() returned error: %v", err)
		}

		// Verify all sessions are deleted
		for i := 0; i < 3; i++ {
			_, err := s.GetSession(ctx, "token_all_"+string(rune('a'+i)))
			if err != accounts.ErrInvalidSession {
				t.Errorf("session %d should be deleted", i)
			}
		}
	})

	t.Run("delete expired sessions", func(t *testing.T) {
		// Create expired and valid sessions
		s.CreateSession(ctx, &accounts.Session{
			ID:           "sess_exp_1",
			UserID:       u.ID,
			Token:        "token_exp_1",
			LastActiveAt: now,
			ExpiresAt:    now.Add(-1 * time.Hour),
			CreatedAt:    now,
		})
		s.CreateSession(ctx, &accounts.Session{
			ID:           "sess_valid_1",
			UserID:       u.ID,
			Token:        "token_valid_1",
			LastActiveAt: now,
			ExpiresAt:    now.Add(24 * time.Hour),
			CreatedAt:    now,
		})

		err := s.DeleteExpiredSessions(ctx)
		if err != nil {
			t.Fatalf("DeleteExpiredSessions() returned error: %v", err)
		}

		// Valid session should still exist
		_, err = s.GetSession(ctx, "token_valid_1")
		if err != nil {
			t.Error("valid session should still exist")
		}
	})
}

func TestNullString(t *testing.T) {
	t.Run("empty string", func(t *testing.T) {
		ns := nullString("")
		if ns.Valid {
			t.Error("nullString(\"\") should be invalid")
		}
	})

	t.Run("non-empty string", func(t *testing.T) {
		ns := nullString("hello")
		if !ns.Valid {
			t.Error("nullString(\"hello\") should be valid")
		}
		if ns.String != "hello" {
			t.Errorf("String = %v, want hello", ns.String)
		}
	})
}
