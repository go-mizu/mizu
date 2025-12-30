package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/cms/feature/users"
)

func TestUsersStore_Create(t *testing.T) {
	db := setupTestDB(t)
	store := NewUsersStore(db)
	ctx := context.Background()

	user := &users.User{
		ID:           "user-001",
		Email:        "test@example.com",
		PasswordHash: "hash123",
		Name:         "Test User",
		Slug:         "test-user",
		Bio:          "A test user bio",
		AvatarURL:    "https://example.com/avatar.jpg",
		Role:         "author",
		Status:       "active",
		Meta:         `{"key":"value"}`,
		CreatedAt:    testTime,
		UpdatedAt:    testTime,
	}

	err := store.Create(ctx, user)
	assertNoError(t, err)

	// Verify user was created
	got, err := store.GetByID(ctx, user.ID)
	assertNoError(t, err)
	assertEqual(t, "ID", got.ID, user.ID)
	assertEqual(t, "Email", got.Email, user.Email)
	assertEqual(t, "Name", got.Name, user.Name)
	assertEqual(t, "Slug", got.Slug, user.Slug)
	assertEqual(t, "Bio", got.Bio, user.Bio)
	assertEqual(t, "AvatarURL", got.AvatarURL, user.AvatarURL)
	assertEqual(t, "Role", got.Role, user.Role)
	assertEqual(t, "Status", got.Status, user.Status)
	assertEqual(t, "Meta", got.Meta, user.Meta)
}

func TestUsersStore_Create_MinimalFields(t *testing.T) {
	db := setupTestDB(t)
	store := NewUsersStore(db)
	ctx := context.Background()

	user := &users.User{
		ID:           "user-002",
		Email:        "minimal@example.com",
		PasswordHash: "hash123",
		Name:         "Minimal User",
		Slug:         "minimal-user",
		Role:         "author",
		Status:       "active",
		CreatedAt:    testTime,
		UpdatedAt:    testTime,
	}

	err := store.Create(ctx, user)
	assertNoError(t, err)

	got, err := store.GetByID(ctx, user.ID)
	assertNoError(t, err)
	assertEqual(t, "ID", got.ID, user.ID)
	assertEqual(t, "Bio", got.Bio, "")
	assertEqual(t, "AvatarURL", got.AvatarURL, "")
}

func TestUsersStore_Create_DuplicateEmail(t *testing.T) {
	db := setupTestDB(t)
	store := NewUsersStore(db)
	ctx := context.Background()

	user1 := &users.User{
		ID:           "user-003",
		Email:        "duplicate@example.com",
		PasswordHash: "hash123",
		Name:         "User 1",
		Slug:         "user-1",
		Role:         "author",
		Status:       "active",
		CreatedAt:    testTime,
		UpdatedAt:    testTime,
	}
	assertNoError(t, store.Create(ctx, user1))

	user2 := &users.User{
		ID:           "user-004",
		Email:        "duplicate@example.com", // Same email
		PasswordHash: "hash456",
		Name:         "User 2",
		Slug:         "user-2",
		Role:         "author",
		Status:       "active",
		CreatedAt:    testTime,
		UpdatedAt:    testTime,
	}
	err := store.Create(ctx, user2)
	assertError(t, err) // Should fail due to unique constraint
}

func TestUsersStore_Create_DuplicateSlug(t *testing.T) {
	db := setupTestDB(t)
	store := NewUsersStore(db)
	ctx := context.Background()

	user1 := &users.User{
		ID:           "user-005",
		Email:        "user5@example.com",
		PasswordHash: "hash123",
		Name:         "User 1",
		Slug:         "same-slug",
		Role:         "author",
		Status:       "active",
		CreatedAt:    testTime,
		UpdatedAt:    testTime,
	}
	assertNoError(t, store.Create(ctx, user1))

	user2 := &users.User{
		ID:           "user-006",
		Email:        "user6@example.com",
		PasswordHash: "hash456",
		Name:         "User 2",
		Slug:         "same-slug", // Same slug
		Role:         "author",
		Status:       "active",
		CreatedAt:    testTime,
		UpdatedAt:    testTime,
	}
	err := store.Create(ctx, user2)
	assertError(t, err) // Should fail due to unique constraint
}

func TestUsersStore_GetByID(t *testing.T) {
	db := setupTestDB(t)
	store := NewUsersStore(db)
	ctx := context.Background()

	user := &users.User{
		ID:           "user-get-001",
		Email:        "getby@example.com",
		PasswordHash: "hash123",
		Name:         "Get By ID User",
		Slug:         "getby-user",
		Role:         "author",
		Status:       "active",
		CreatedAt:    testTime,
		UpdatedAt:    testTime,
	}
	assertNoError(t, store.Create(ctx, user))

	got, err := store.GetByID(ctx, user.ID)
	assertNoError(t, err)
	assertEqual(t, "ID", got.ID, user.ID)
	assertEqual(t, "Email", got.Email, user.Email)
}

func TestUsersStore_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	store := NewUsersStore(db)
	ctx := context.Background()

	got, err := store.GetByID(ctx, "nonexistent")
	assertNoError(t, err) // Should not error
	if got != nil {
		t.Error("expected nil for non-existent user")
	}
}

func TestUsersStore_GetByEmail(t *testing.T) {
	db := setupTestDB(t)
	store := NewUsersStore(db)
	ctx := context.Background()

	user := &users.User{
		ID:           "user-email-001",
		Email:        "findme@example.com",
		PasswordHash: "hash123",
		Name:         "Find By Email",
		Slug:         "find-email",
		Role:         "author",
		Status:       "active",
		CreatedAt:    testTime,
		UpdatedAt:    testTime,
	}
	assertNoError(t, store.Create(ctx, user))

	got, err := store.GetByEmail(ctx, "findme@example.com")
	assertNoError(t, err)
	assertEqual(t, "ID", got.ID, user.ID)
	assertEqual(t, "Email", got.Email, user.Email)
}

func TestUsersStore_GetByEmail_NotFound(t *testing.T) {
	db := setupTestDB(t)
	store := NewUsersStore(db)
	ctx := context.Background()

	got, err := store.GetByEmail(ctx, "notfound@example.com")
	assertNoError(t, err)
	if got != nil {
		t.Error("expected nil for non-existent email")
	}
}

func TestUsersStore_GetBySlug(t *testing.T) {
	db := setupTestDB(t)
	store := NewUsersStore(db)
	ctx := context.Background()

	user := &users.User{
		ID:           "user-slug-001",
		Email:        "slug@example.com",
		PasswordHash: "hash123",
		Name:         "Find By Slug",
		Slug:         "find-by-slug",
		Role:         "author",
		Status:       "active",
		CreatedAt:    testTime,
		UpdatedAt:    testTime,
	}
	assertNoError(t, store.Create(ctx, user))

	got, err := store.GetBySlug(ctx, "find-by-slug")
	assertNoError(t, err)
	assertEqual(t, "ID", got.ID, user.ID)
	assertEqual(t, "Slug", got.Slug, user.Slug)
}

func TestUsersStore_GetByIDs(t *testing.T) {
	db := setupTestDB(t)
	store := NewUsersStore(db)
	ctx := context.Background()

	// Create multiple users
	for i, email := range []string{"multi1@ex.com", "multi2@ex.com", "multi3@ex.com"} {
		user := &users.User{
			ID:           "user-multi-" + string(rune('0'+i)),
			Email:        email,
			PasswordHash: "hash",
			Name:         "Multi User " + string(rune('0'+i)),
			Slug:         "multi-" + string(rune('0'+i)),
			Role:         "author",
			Status:       "active",
			CreatedAt:    testTime,
			UpdatedAt:    testTime,
		}
		assertNoError(t, store.Create(ctx, user))
	}

	got, err := store.GetByIDs(ctx, []string{"user-multi-0", "user-multi-2"})
	assertNoError(t, err)
	assertLen(t, got, 2)
}

func TestUsersStore_GetByIDs_Empty(t *testing.T) {
	db := setupTestDB(t)
	store := NewUsersStore(db)
	ctx := context.Background()

	got, err := store.GetByIDs(ctx, []string{})
	assertNoError(t, err)
	if got != nil {
		t.Errorf("expected nil for empty IDs, got %v", got)
	}
}

func TestUsersStore_GetByIDs_PartialMatch(t *testing.T) {
	db := setupTestDB(t)
	store := NewUsersStore(db)
	ctx := context.Background()

	user := &users.User{
		ID:           "user-partial-001",
		Email:        "partial@example.com",
		PasswordHash: "hash",
		Name:         "Partial User",
		Slug:         "partial-user",
		Role:         "author",
		Status:       "active",
		CreatedAt:    testTime,
		UpdatedAt:    testTime,
	}
	assertNoError(t, store.Create(ctx, user))

	got, err := store.GetByIDs(ctx, []string{"user-partial-001", "nonexistent"})
	assertNoError(t, err)
	assertLen(t, got, 1)
	assertEqual(t, "ID", got[0].ID, "user-partial-001")
}

func TestUsersStore_List(t *testing.T) {
	db := setupTestDB(t)
	store := NewUsersStore(db)
	ctx := context.Background()

	// Create users
	for i := 0; i < 5; i++ {
		user := &users.User{
			ID:           "user-list-" + string(rune('a'+i)),
			Email:        "list" + string(rune('a'+i)) + "@example.com",
			PasswordHash: "hash",
			Name:         "List User " + string(rune('A'+i)),
			Slug:         "list-user-" + string(rune('a'+i)),
			Role:         "author",
			Status:       "active",
			CreatedAt:    testTime,
			UpdatedAt:    testTime,
		}
		assertNoError(t, store.Create(ctx, user))
	}

	list, total, err := store.List(ctx, &users.ListIn{Limit: 10, Offset: 0})
	assertNoError(t, err)
	assertEqual(t, "total", total, 5)
	assertLen(t, list, 5)
}

func TestUsersStore_List_FilterByRole(t *testing.T) {
	db := setupTestDB(t)
	store := NewUsersStore(db)
	ctx := context.Background()

	// Create users with different roles
	roles := []string{"admin", "author", "author", "editor"}
	for i, role := range roles {
		user := &users.User{
			ID:           "user-role-" + string(rune('a'+i)),
			Email:        "role" + string(rune('a'+i)) + "@example.com",
			PasswordHash: "hash",
			Name:         "Role User",
			Slug:         "role-user-" + string(rune('a'+i)),
			Role:         role,
			Status:       "active",
			CreatedAt:    testTime,
			UpdatedAt:    testTime,
		}
		assertNoError(t, store.Create(ctx, user))
	}

	list, total, err := store.List(ctx, &users.ListIn{Role: "author", Limit: 10})
	assertNoError(t, err)
	assertEqual(t, "total", total, 2)
	assertLen(t, list, 2)
}

func TestUsersStore_List_FilterByStatus(t *testing.T) {
	db := setupTestDB(t)
	store := NewUsersStore(db)
	ctx := context.Background()

	statuses := []string{"active", "active", "suspended"}
	for i, status := range statuses {
		user := &users.User{
			ID:           "user-status-" + string(rune('a'+i)),
			Email:        "status" + string(rune('a'+i)) + "@example.com",
			PasswordHash: "hash",
			Name:         "Status User",
			Slug:         "status-user-" + string(rune('a'+i)),
			Role:         "author",
			Status:       status,
			CreatedAt:    testTime,
			UpdatedAt:    testTime,
		}
		assertNoError(t, store.Create(ctx, user))
	}

	list, total, err := store.List(ctx, &users.ListIn{Status: "suspended", Limit: 10})
	assertNoError(t, err)
	assertEqual(t, "total", total, 1)
	assertLen(t, list, 1)
}

func TestUsersStore_List_Search(t *testing.T) {
	db := setupTestDB(t)
	store := NewUsersStore(db)
	ctx := context.Background()

	usersData := []struct {
		name  string
		email string
	}{
		{"Alice Smith", "alice@example.com"},
		{"Bob Jones", "bob@example.com"},
		{"Alice Wonder", "wonder@example.com"},
	}
	for i, u := range usersData {
		user := &users.User{
			ID:           "user-search-" + string(rune('a'+i)),
			Email:        u.email,
			PasswordHash: "hash",
			Name:         u.name,
			Slug:         "search-" + string(rune('a'+i)),
			Role:         "author",
			Status:       "active",
			CreatedAt:    testTime,
			UpdatedAt:    testTime,
		}
		assertNoError(t, store.Create(ctx, user))
	}

	// Search by name
	list, total, err := store.List(ctx, &users.ListIn{Search: "Alice", Limit: 10})
	assertNoError(t, err)
	assertEqual(t, "total", total, 2)

	// Search by email
	list, total, err = store.List(ctx, &users.ListIn{Search: "bob@", Limit: 10})
	assertNoError(t, err)
	assertEqual(t, "total", total, 1)
	assertEqual(t, "Name", list[0].Name, "Bob Jones")
}

func TestUsersStore_List_Empty(t *testing.T) {
	db := setupTestDB(t)
	store := NewUsersStore(db)
	ctx := context.Background()

	list, total, err := store.List(ctx, &users.ListIn{Limit: 10})
	assertNoError(t, err)
	assertEqual(t, "total", total, 0)
	assertLen(t, list, 0)
}

func TestUsersStore_Update(t *testing.T) {
	db := setupTestDB(t)
	store := NewUsersStore(db)
	ctx := context.Background()

	user := &users.User{
		ID:           "user-update-001",
		Email:        "update@example.com",
		PasswordHash: "hash",
		Name:         "Original Name",
		Slug:         "update-user",
		Role:         "author",
		Status:       "active",
		CreatedAt:    testTime,
		UpdatedAt:    testTime,
	}
	assertNoError(t, store.Create(ctx, user))

	err := store.Update(ctx, user.ID, &users.UpdateIn{
		Name: ptr("Updated Name"),
		Bio:  ptr("New bio"),
	})
	assertNoError(t, err)

	got, _ := store.GetByID(ctx, user.ID)
	assertEqual(t, "Name", got.Name, "Updated Name")
	assertEqual(t, "Bio", got.Bio, "New bio")
}

func TestUsersStore_Update_PartialFields(t *testing.T) {
	db := setupTestDB(t)
	store := NewUsersStore(db)
	ctx := context.Background()

	user := &users.User{
		ID:           "user-partial-update",
		Email:        "partialup@example.com",
		PasswordHash: "hash",
		Name:         "Original",
		Slug:         "partial-up",
		Bio:          "Original Bio",
		Role:         "author",
		Status:       "active",
		CreatedAt:    testTime,
		UpdatedAt:    testTime,
	}
	assertNoError(t, store.Create(ctx, user))

	// Only update name, bio should remain
	err := store.Update(ctx, user.ID, &users.UpdateIn{
		Name: ptr("New Name"),
	})
	assertNoError(t, err)

	got, _ := store.GetByID(ctx, user.ID)
	assertEqual(t, "Name", got.Name, "New Name")
	assertEqual(t, "Bio", got.Bio, "Original Bio") // Should remain unchanged
}

func TestUsersStore_Update_NoFields(t *testing.T) {
	db := setupTestDB(t)
	store := NewUsersStore(db)
	ctx := context.Background()

	user := &users.User{
		ID:           "user-noop",
		Email:        "noop@example.com",
		PasswordHash: "hash",
		Name:         "NoOp User",
		Slug:         "noop-user",
		Role:         "author",
		Status:       "active",
		CreatedAt:    testTime,
		UpdatedAt:    testTime,
	}
	assertNoError(t, store.Create(ctx, user))

	// Update with no fields should be no-op
	err := store.Update(ctx, user.ID, &users.UpdateIn{})
	assertNoError(t, err)
}

func TestUsersStore_UpdatePassword(t *testing.T) {
	db := setupTestDB(t)
	store := NewUsersStore(db)
	ctx := context.Background()

	user := &users.User{
		ID:           "user-pwd",
		Email:        "pwd@example.com",
		PasswordHash: "oldhash",
		Name:         "Password User",
		Slug:         "pwd-user",
		Role:         "author",
		Status:       "active",
		CreatedAt:    testTime,
		UpdatedAt:    testTime,
	}
	assertNoError(t, store.Create(ctx, user))

	err := store.UpdatePassword(ctx, user.ID, "newhash")
	assertNoError(t, err)

	got, _ := store.GetByID(ctx, user.ID)
	assertEqual(t, "PasswordHash", got.PasswordHash, "newhash")
}

func TestUsersStore_UpdateLastLogin(t *testing.T) {
	db := setupTestDB(t)
	store := NewUsersStore(db)
	ctx := context.Background()

	user := &users.User{
		ID:           "user-login",
		Email:        "login@example.com",
		PasswordHash: "hash",
		Name:         "Login User",
		Slug:         "login-user",
		Role:         "author",
		Status:       "active",
		CreatedAt:    testTime,
		UpdatedAt:    testTime,
	}
	assertNoError(t, store.Create(ctx, user))

	err := store.UpdateLastLogin(ctx, user.ID)
	assertNoError(t, err)

	got, _ := store.GetByID(ctx, user.ID)
	if got.LastLoginAt == nil {
		t.Error("expected LastLoginAt to be set")
	}
}

func TestUsersStore_Delete(t *testing.T) {
	db := setupTestDB(t)
	store := NewUsersStore(db)
	ctx := context.Background()

	user := &users.User{
		ID:           "user-delete",
		Email:        "delete@example.com",
		PasswordHash: "hash",
		Name:         "Delete User",
		Slug:         "delete-user",
		Role:         "author",
		Status:       "active",
		CreatedAt:    testTime,
		UpdatedAt:    testTime,
	}
	assertNoError(t, store.Create(ctx, user))

	err := store.Delete(ctx, user.ID)
	assertNoError(t, err)

	got, err := store.GetByID(ctx, user.ID)
	assertNoError(t, err)
	if got != nil {
		t.Error("expected user to be deleted")
	}
}

func TestUsersStore_Delete_NotFound(t *testing.T) {
	db := setupTestDB(t)
	store := NewUsersStore(db)
	ctx := context.Background()

	err := store.Delete(ctx, "nonexistent")
	assertNoError(t, err) // Should not error for non-existent
}

// Session tests

func TestUsersStore_CreateSession(t *testing.T) {
	db := setupTestDB(t)
	store := NewUsersStore(db)
	ctx := context.Background()

	// Create user first
	user := &users.User{
		ID:           "user-sess",
		Email:        "sess@example.com",
		PasswordHash: "hash",
		Name:         "Session User",
		Slug:         "sess-user",
		Role:         "author",
		Status:       "active",
		CreatedAt:    testTime,
		UpdatedAt:    testTime,
	}
	assertNoError(t, store.Create(ctx, user))

	sess := &users.Session{
		ID:           "sess-001",
		UserID:       user.ID,
		RefreshToken: "refresh-token-123",
		UserAgent:    "Mozilla/5.0",
		IPAddress:    "192.168.1.1",
		ExpiresAt:    testTime.Add(24 * time.Hour),
		CreatedAt:    testTime,
	}

	err := store.CreateSession(ctx, sess)
	assertNoError(t, err)

	got, err := store.GetSession(ctx, sess.ID)
	assertNoError(t, err)
	assertEqual(t, "ID", got.ID, sess.ID)
	assertEqual(t, "UserID", got.UserID, sess.UserID)
	assertEqual(t, "RefreshToken", got.RefreshToken, sess.RefreshToken)
}

func TestUsersStore_GetSession_NotFound(t *testing.T) {
	db := setupTestDB(t)
	store := NewUsersStore(db)
	ctx := context.Background()

	got, err := store.GetSession(ctx, "nonexistent")
	assertNoError(t, err)
	if got != nil {
		t.Error("expected nil for non-existent session")
	}
}

func TestUsersStore_DeleteSession(t *testing.T) {
	db := setupTestDB(t)
	store := NewUsersStore(db)
	ctx := context.Background()

	// Create user and session
	user := &users.User{
		ID:           "user-del-sess",
		Email:        "delsess@example.com",
		PasswordHash: "hash",
		Name:         "Del Session User",
		Slug:         "del-sess-user",
		Role:         "author",
		Status:       "active",
		CreatedAt:    testTime,
		UpdatedAt:    testTime,
	}
	assertNoError(t, store.Create(ctx, user))

	sess := &users.Session{
		ID:        "sess-del",
		UserID:    user.ID,
		ExpiresAt: testTime.Add(24 * time.Hour),
		CreatedAt: testTime,
	}
	assertNoError(t, store.CreateSession(ctx, sess))

	err := store.DeleteSession(ctx, sess.ID)
	assertNoError(t, err)

	got, _ := store.GetSession(ctx, sess.ID)
	if got != nil {
		t.Error("expected session to be deleted")
	}
}

func TestUsersStore_DeleteExpiredSessions(t *testing.T) {
	db := setupTestDB(t)
	store := NewUsersStore(db)
	ctx := context.Background()

	// Create user
	user := &users.User{
		ID:           "user-exp-sess",
		Email:        "expsess@example.com",
		PasswordHash: "hash",
		Name:         "Expired Session User",
		Slug:         "exp-sess-user",
		Role:         "author",
		Status:       "active",
		CreatedAt:    testTime,
		UpdatedAt:    testTime,
	}
	assertNoError(t, store.Create(ctx, user))

	// Create expired session
	expiredSess := &users.Session{
		ID:        "sess-expired",
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
		CreatedAt: testTime,
	}
	assertNoError(t, store.CreateSession(ctx, expiredSess))

	// Create valid session
	validSess := &users.Session{
		ID:        "sess-valid",
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(24 * time.Hour), // Valid
		CreatedAt: testTime,
	}
	assertNoError(t, store.CreateSession(ctx, validSess))

	err := store.DeleteExpiredSessions(ctx)
	assertNoError(t, err)

	// Expired should be deleted
	got, _ := store.GetSession(ctx, "sess-expired")
	if got != nil {
		t.Error("expected expired session to be deleted")
	}

	// Valid should remain
	got, _ = store.GetSession(ctx, "sess-valid")
	if got == nil {
		t.Error("expected valid session to remain")
	}
}

func TestUsersStore_GetUserBySession(t *testing.T) {
	db := setupTestDB(t)
	store := NewUsersStore(db)
	ctx := context.Background()

	// Create user
	user := &users.User{
		ID:           "user-by-sess",
		Email:        "bysess@example.com",
		PasswordHash: "hash",
		Name:         "By Session User",
		Slug:         "by-sess-user",
		Role:         "author",
		Status:       "active",
		CreatedAt:    testTime,
		UpdatedAt:    testTime,
	}
	assertNoError(t, store.Create(ctx, user))

	// Create valid session
	sess := &users.Session{
		ID:        "sess-get-user",
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: testTime,
	}
	assertNoError(t, store.CreateSession(ctx, sess))

	got, err := store.GetUserBySession(ctx, sess.ID)
	assertNoError(t, err)
	assertEqual(t, "ID", got.ID, user.ID)
	assertEqual(t, "Email", got.Email, user.Email)
}

func TestUsersStore_GetUserBySession_Expired(t *testing.T) {
	db := setupTestDB(t)
	store := NewUsersStore(db)
	ctx := context.Background()

	// Create user
	user := &users.User{
		ID:           "user-exp-get",
		Email:        "expget@example.com",
		PasswordHash: "hash",
		Name:         "Expired Get User",
		Slug:         "exp-get-user",
		Role:         "author",
		Status:       "active",
		CreatedAt:    testTime,
		UpdatedAt:    testTime,
	}
	assertNoError(t, store.Create(ctx, user))

	// Create expired session
	sess := &users.Session{
		ID:        "sess-exp-get",
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
		CreatedAt: testTime,
	}
	assertNoError(t, store.CreateSession(ctx, sess))

	got, err := store.GetUserBySession(ctx, sess.ID)
	assertNoError(t, err)
	if got != nil {
		t.Error("expected nil for expired session")
	}
}

func TestUsersStore_GetUserBySession_NotFound(t *testing.T) {
	db := setupTestDB(t)
	store := NewUsersStore(db)
	ctx := context.Background()

	got, err := store.GetUserBySession(ctx, "nonexistent")
	assertNoError(t, err)
	if got != nil {
		t.Error("expected nil for non-existent session")
	}
}
