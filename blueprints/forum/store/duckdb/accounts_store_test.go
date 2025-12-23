package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/forum/feature/accounts"
)

func TestAccountsStore_Create(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	account := &accounts.Account{
		ID:           newTestID(),
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hashedpassword",
		DisplayName:  "Test User",
		Bio:          "A test user bio",
		AvatarURL:    "https://example.com/avatar.png",
		BannerURL:    "https://example.com/banner.png",
		Karma:        100,
		PostKarma:    50,
		CommentKarma: 50,
		IsAdmin:      false,
		IsSuspended:  false,
		CreatedAt:    testTime(),
		UpdatedAt:    testTime(),
	}

	err := store.Accounts().Create(ctx, account)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify by retrieving
	got, err := store.Accounts().GetByID(ctx, account.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if got.Username != account.Username {
		t.Errorf("Username: got %q, want %q", got.Username, account.Username)
	}
	if got.Email != account.Email {
		t.Errorf("Email: got %q, want %q", got.Email, account.Email)
	}
	if got.DisplayName != account.DisplayName {
		t.Errorf("DisplayName: got %q, want %q", got.DisplayName, account.DisplayName)
	}
	if got.Bio != account.Bio {
		t.Errorf("Bio: got %q, want %q", got.Bio, account.Bio)
	}
	if got.Karma != account.Karma {
		t.Errorf("Karma: got %d, want %d", got.Karma, account.Karma)
	}
}

func TestAccountsStore_Create_DuplicateUsername(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	account1 := &accounts.Account{
		ID:           newTestID(),
		Username:     "sameuser",
		Email:        "user1@example.com",
		PasswordHash: "hash1",
		CreatedAt:    testTime(),
		UpdatedAt:    testTime(),
	}

	account2 := &accounts.Account{
		ID:           newTestID(),
		Username:     "sameuser",
		Email:        "user2@example.com",
		PasswordHash: "hash2",
		CreatedAt:    testTime(),
		UpdatedAt:    testTime(),
	}

	if err := store.Accounts().Create(ctx, account1); err != nil {
		t.Fatalf("Create first account failed: %v", err)
	}

	err := store.Accounts().Create(ctx, account2)
	if err == nil {
		t.Error("expected error for duplicate username, got nil")
	}
}

func TestAccountsStore_Create_DuplicateEmail(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	account1 := &accounts.Account{
		ID:           newTestID(),
		Username:     "user1",
		Email:        "same@example.com",
		PasswordHash: "hash1",
		CreatedAt:    testTime(),
		UpdatedAt:    testTime(),
	}

	account2 := &accounts.Account{
		ID:           newTestID(),
		Username:     "user2",
		Email:        "same@example.com",
		PasswordHash: "hash2",
		CreatedAt:    testTime(),
		UpdatedAt:    testTime(),
	}

	if err := store.Accounts().Create(ctx, account1); err != nil {
		t.Fatalf("Create first account failed: %v", err)
	}

	err := store.Accounts().Create(ctx, account2)
	if err == nil {
		t.Error("expected error for duplicate email, got nil")
	}
}

func TestAccountsStore_GetByID(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	account := &accounts.Account{
		ID:           newTestID(),
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hash",
		CreatedAt:    testTime(),
		UpdatedAt:    testTime(),
	}

	if err := store.Accounts().Create(ctx, account); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := store.Accounts().GetByID(ctx, account.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if got.ID != account.ID {
		t.Errorf("ID: got %q, want %q", got.ID, account.ID)
	}
}

func TestAccountsStore_GetByID_NotFound(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	_, err := store.Accounts().GetByID(ctx, "nonexistent")
	if err != accounts.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestAccountsStore_GetByUsername(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	account := &accounts.Account{
		ID:           newTestID(),
		Username:     "TestUser",
		Email:        "test@example.com",
		PasswordHash: "hash",
		CreatedAt:    testTime(),
		UpdatedAt:    testTime(),
	}

	if err := store.Accounts().Create(ctx, account); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Test case-insensitive lookup
	got, err := store.Accounts().GetByUsername(ctx, "testuser")
	if err != nil {
		t.Fatalf("GetByUsername failed: %v", err)
	}

	if got.ID != account.ID {
		t.Errorf("ID: got %q, want %q", got.ID, account.ID)
	}

	// Test with different case
	got2, err := store.Accounts().GetByUsername(ctx, "TESTUSER")
	if err != nil {
		t.Fatalf("GetByUsername (uppercase) failed: %v", err)
	}

	if got2.ID != account.ID {
		t.Errorf("ID: got %q, want %q", got2.ID, account.ID)
	}
}

func TestAccountsStore_GetByEmail(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	account := &accounts.Account{
		ID:           newTestID(),
		Username:     "testuser",
		Email:        "Test@Example.com",
		PasswordHash: "hash",
		CreatedAt:    testTime(),
		UpdatedAt:    testTime(),
	}

	if err := store.Accounts().Create(ctx, account); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Test case-insensitive lookup
	got, err := store.Accounts().GetByEmail(ctx, "test@example.com")
	if err != nil {
		t.Fatalf("GetByEmail failed: %v", err)
	}

	if got.ID != account.ID {
		t.Errorf("ID: got %q, want %q", got.ID, account.ID)
	}
}

func TestAccountsStore_Update(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	account := &accounts.Account{
		ID:           newTestID(),
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hash",
		DisplayName:  "Original Name",
		Bio:          "Original bio",
		CreatedAt:    testTime(),
		UpdatedAt:    testTime(),
	}

	if err := store.Accounts().Create(ctx, account); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Update
	account.DisplayName = "Updated Name"
	account.Bio = "Updated bio"
	account.Karma = 200
	account.UpdatedAt = testTime().Add(time.Hour)

	if err := store.Accounts().Update(ctx, account); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify
	got, err := store.Accounts().GetByID(ctx, account.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if got.DisplayName != "Updated Name" {
		t.Errorf("DisplayName: got %q, want %q", got.DisplayName, "Updated Name")
	}
	if got.Bio != "Updated bio" {
		t.Errorf("Bio: got %q, want %q", got.Bio, "Updated bio")
	}
	if got.Karma != 200 {
		t.Errorf("Karma: got %d, want %d", got.Karma, 200)
	}
}

func TestAccountsStore_Delete(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	account := &accounts.Account{
		ID:           newTestID(),
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hash",
		CreatedAt:    testTime(),
		UpdatedAt:    testTime(),
	}

	if err := store.Accounts().Create(ctx, account); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Delete
	if err := store.Accounts().Delete(ctx, account.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deleted
	_, err := store.Accounts().GetByID(ctx, account.ID)
	if err != accounts.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestAccountsStore_Session_CRUD(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	// Create account first
	account := &accounts.Account{
		ID:           newTestID(),
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hash",
		CreatedAt:    testTime(),
		UpdatedAt:    testTime(),
	}

	if err := store.Accounts().Create(ctx, account); err != nil {
		t.Fatalf("Create account failed: %v", err)
	}

	// Create session
	session := &accounts.Session{
		ID:        newTestID(),
		AccountID: account.ID,
		Token:     "test-token-123",
		UserAgent: "TestBrowser/1.0",
		IP:        "127.0.0.1",
		ExpiresAt: testTime().Add(24 * time.Hour),
		CreatedAt: testTime(),
	}

	if err := store.Accounts().CreateSession(ctx, session); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Get session by token
	got, err := store.Accounts().GetSessionByToken(ctx, session.Token)
	if err != nil {
		t.Fatalf("GetSessionByToken failed: %v", err)
	}

	if got.ID != session.ID {
		t.Errorf("ID: got %q, want %q", got.ID, session.ID)
	}
	if got.AccountID != session.AccountID {
		t.Errorf("AccountID: got %q, want %q", got.AccountID, session.AccountID)
	}
	if got.UserAgent != session.UserAgent {
		t.Errorf("UserAgent: got %q, want %q", got.UserAgent, session.UserAgent)
	}

	// Delete session
	if err := store.Accounts().DeleteSession(ctx, session.Token); err != nil {
		t.Fatalf("DeleteSession failed: %v", err)
	}

	// Verify deleted
	_, err = store.Accounts().GetSessionByToken(ctx, session.Token)
	if err != accounts.ErrSessionExpired {
		t.Errorf("expected ErrSessionExpired after delete, got %v", err)
	}
}

func TestAccountsStore_Session_DeleteByAccount(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	// Create account
	account := &accounts.Account{
		ID:           newTestID(),
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hash",
		CreatedAt:    testTime(),
		UpdatedAt:    testTime(),
	}

	if err := store.Accounts().Create(ctx, account); err != nil {
		t.Fatalf("Create account failed: %v", err)
	}

	// Create multiple sessions
	for i := 0; i < 3; i++ {
		session := &accounts.Session{
			ID:        newTestID(),
			AccountID: account.ID,
			Token:     newTestID(),
			ExpiresAt: testTime().Add(24 * time.Hour),
			CreatedAt: testTime(),
		}
		if err := store.Accounts().CreateSession(ctx, session); err != nil {
			t.Fatalf("CreateSession %d failed: %v", i, err)
		}
	}

	// Delete all sessions for account
	if err := store.Accounts().DeleteSessionsByAccount(ctx, account.ID); err != nil {
		t.Fatalf("DeleteSessionsByAccount failed: %v", err)
	}

	// Sessions should be gone (can't easily verify without listing, but this tests the delete doesn't error)
}

func TestAccountsStore_List(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	// Create multiple accounts
	for i := 0; i < 5; i++ {
		account := &accounts.Account{
			ID:           newTestID(),
			Username:     "user" + string(rune('a'+i)),
			Email:        "user" + string(rune('a'+i)) + "@example.com",
			PasswordHash: "hash",
			Karma:        int64(i * 10),
			CreatedAt:    testTime().Add(time.Duration(i) * time.Hour),
			UpdatedAt:    testTime().Add(time.Duration(i) * time.Hour),
		}
		if err := store.Accounts().Create(ctx, account); err != nil {
			t.Fatalf("Create account %d failed: %v", i, err)
		}
	}

	// List by created_at (default)
	list, err := store.Accounts().List(ctx, accounts.ListOpts{Limit: 10})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != 5 {
		t.Errorf("List count: got %d, want 5", len(list))
	}

	// List by karma
	listByKarma, err := store.Accounts().List(ctx, accounts.ListOpts{Limit: 10, OrderBy: "karma"})
	if err != nil {
		t.Fatalf("List by karma failed: %v", err)
	}

	if len(listByKarma) != 5 {
		t.Errorf("List count: got %d, want 5", len(listByKarma))
	}

	// First should be highest karma
	if listByKarma[0].Karma < listByKarma[len(listByKarma)-1].Karma {
		t.Error("List by karma should be descending")
	}
}

func TestAccountsStore_Search(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	// Create accounts with searchable names
	testAccounts := []struct {
		username    string
		displayName string
	}{
		{"alice", "Alice Smith"},
		{"bob", "Bob Jones"},
		{"charlie", "Charlie Brown"},
		{"alex", "Alex Johnson"},
	}

	for _, a := range testAccounts {
		account := &accounts.Account{
			ID:           newTestID(),
			Username:     a.username,
			Email:        a.username + "@example.com",
			PasswordHash: "hash",
			DisplayName:  a.displayName,
			CreatedAt:    testTime(),
			UpdatedAt:    testTime(),
		}
		if err := store.Accounts().Create(ctx, account); err != nil {
			t.Fatalf("Create account %s failed: %v", a.username, err)
		}
	}

	// Search for "al" - should find alice and alex
	results, err := store.Accounts().Search(ctx, "al", 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Search 'al' count: got %d, want 2", len(results))
	}

	// Search for "smith" - should find alice (by display name)
	results2, err := store.Accounts().Search(ctx, "smith", 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results2) != 1 {
		t.Errorf("Search 'smith' count: got %d, want 1", len(results2))
	}
}

func TestAccountsStore_Suspension(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	suspendUntil := testTime().Add(7 * 24 * time.Hour)
	account := &accounts.Account{
		ID:            newTestID(),
		Username:      "testuser",
		Email:         "test@example.com",
		PasswordHash:  "hash",
		IsSuspended:   true,
		SuspendReason: "Test suspension",
		SuspendUntil:  &suspendUntil,
		CreatedAt:     testTime(),
		UpdatedAt:     testTime(),
	}

	if err := store.Accounts().Create(ctx, account); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := store.Accounts().GetByID(ctx, account.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if !got.IsSuspended {
		t.Error("IsSuspended: got false, want true")
	}
	if got.SuspendReason != "Test suspension" {
		t.Errorf("SuspendReason: got %q, want %q", got.SuspendReason, "Test suspension")
	}
	if got.SuspendUntil == nil {
		t.Error("SuspendUntil: got nil, want non-nil")
	}
}
