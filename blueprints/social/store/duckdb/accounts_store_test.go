package duckdb

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/social/feature/accounts"
	"github.com/go-mizu/blueprints/social/feature/relationships"
)

func createTestAccount(t *testing.T, db *sql.DB, username string) *accounts.Account {
	t.Helper()
	store := NewAccountsStore(db)
	ctx := context.Background()

	account := &accounts.Account{
		ID:           newTestID(),
		Username:     username,
		DisplayName:  username + " User",
		Email:        username + "@example.com",
		Bio:          "Bio for " + username,
		AvatarURL:    "https://example.com/avatar/" + username,
		HeaderURL:    "https://example.com/header/" + username,
		Location:     "Test City",
		Website:      "https://example.com/" + username,
		Verified:     false,
		Admin:        false,
		Suspended:    false,
		Private:      false,
		Discoverable: true,
		CreatedAt:    testTime(),
		UpdatedAt:    testTime(),
	}

	if err := store.Insert(ctx, account, "hashedpassword"); err != nil {
		t.Fatalf("createTestAccount failed: %v", err)
	}
	return account
}

func TestAccountsStore_Insert(t *testing.T) {
	db := setupTestStore(t)
	store := NewAccountsStore(db)
	ctx := context.Background()

	account := &accounts.Account{
		ID:           newTestID(),
		Username:     "testuser",
		DisplayName:  "Test User",
		Email:        "test@example.com",
		Bio:          "A test user bio",
		AvatarURL:    "https://example.com/avatar.png",
		HeaderURL:    "https://example.com/header.png",
		Location:     "Test City",
		Website:      "https://example.com",
		Fields:       []accounts.Field{{Name: "Twitter", Value: "@test"}},
		Verified:     true,
		Admin:        false,
		Suspended:    false,
		Private:      false,
		Discoverable: true,
		CreatedAt:    testTime(),
		UpdatedAt:    testTime(),
	}

	err := store.Insert(ctx, account, "hashedpassword")
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Verify by retrieving
	got, err := store.GetByID(ctx, account.ID)
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
	if got.AvatarURL != account.AvatarURL {
		t.Errorf("AvatarURL: got %q, want %q", got.AvatarURL, account.AvatarURL)
	}
	if got.HeaderURL != account.HeaderURL {
		t.Errorf("HeaderURL: got %q, want %q", got.HeaderURL, account.HeaderURL)
	}
	if got.Location != account.Location {
		t.Errorf("Location: got %q, want %q", got.Location, account.Location)
	}
	if got.Website != account.Website {
		t.Errorf("Website: got %q, want %q", got.Website, account.Website)
	}
	if !got.Verified {
		t.Error("Verified: got false, want true")
	}
	if got.Discoverable != account.Discoverable {
		t.Errorf("Discoverable: got %v, want %v", got.Discoverable, account.Discoverable)
	}
	if len(got.Fields) != 1 {
		t.Errorf("Fields: got %d, want 1", len(got.Fields))
	}
}

func TestAccountsStore_Insert_DuplicateUsername(t *testing.T) {
	db := setupTestStore(t)
	store := NewAccountsStore(db)
	ctx := context.Background()

	account1 := &accounts.Account{
		ID:        newTestID(),
		Username:  "sameuser",
		Email:     "user1@example.com",
		CreatedAt: testTime(),
		UpdatedAt: testTime(),
	}

	account2 := &accounts.Account{
		ID:        newTestID(),
		Username:  "sameuser",
		Email:     "user2@example.com",
		CreatedAt: testTime(),
		UpdatedAt: testTime(),
	}

	if err := store.Insert(ctx, account1, "hash1"); err != nil {
		t.Fatalf("Insert first account failed: %v", err)
	}

	err := store.Insert(ctx, account2, "hash2")
	if err == nil {
		t.Error("expected error for duplicate username, got nil")
	}
}

func TestAccountsStore_Insert_DuplicateEmail(t *testing.T) {
	db := setupTestStore(t)
	store := NewAccountsStore(db)
	ctx := context.Background()

	account1 := &accounts.Account{
		ID:        newTestID(),
		Username:  "user1",
		Email:     "same@example.com",
		CreatedAt: testTime(),
		UpdatedAt: testTime(),
	}

	account2 := &accounts.Account{
		ID:        newTestID(),
		Username:  "user2",
		Email:     "same@example.com",
		CreatedAt: testTime(),
		UpdatedAt: testTime(),
	}

	if err := store.Insert(ctx, account1, "hash1"); err != nil {
		t.Fatalf("Insert first account failed: %v", err)
	}

	err := store.Insert(ctx, account2, "hash2")
	if err == nil {
		t.Error("expected error for duplicate email, got nil")
	}
}

func TestAccountsStore_GetByID(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	store := NewAccountsStore(db)

	got, err := store.GetByID(ctx, account.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if got.ID != account.ID {
		t.Errorf("ID: got %q, want %q", got.ID, account.ID)
	}
}

func TestAccountsStore_GetByID_NotFound(t *testing.T) {
	db := setupTestStore(t)
	store := NewAccountsStore(db)
	ctx := context.Background()

	_, err := store.GetByID(ctx, "nonexistent")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestAccountsStore_GetByIDs(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account1 := createTestAccount(t, db, "user1")
	account2 := createTestAccount(t, db, "user2")
	account3 := createTestAccount(t, db, "user3")

	store := NewAccountsStore(db)

	// Get all three
	got, err := store.GetByIDs(ctx, []string{account1.ID, account2.ID, account3.ID})
	if err != nil {
		t.Fatalf("GetByIDs failed: %v", err)
	}

	if len(got) != 3 {
		t.Errorf("GetByIDs count: got %d, want 3", len(got))
	}

	// Get empty slice
	empty, err := store.GetByIDs(ctx, []string{})
	if err != nil {
		t.Fatalf("GetByIDs (empty) failed: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("GetByIDs (empty) count: got %d, want 0", len(empty))
	}
}

func TestAccountsStore_GetByUsername(t *testing.T) {
	db := setupTestStore(t)
	store := NewAccountsStore(db)
	ctx := context.Background()

	account := &accounts.Account{
		ID:        newTestID(),
		Username:  "TestUser",
		Email:     "test@example.com",
		CreatedAt: testTime(),
		UpdatedAt: testTime(),
	}

	if err := store.Insert(ctx, account, "hash"); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Test case-insensitive lookup
	got, err := store.GetByUsername(ctx, "testuser")
	if err != nil {
		t.Fatalf("GetByUsername failed: %v", err)
	}

	if got.ID != account.ID {
		t.Errorf("ID: got %q, want %q", got.ID, account.ID)
	}

	// Test with different case
	got2, err := store.GetByUsername(ctx, "TESTUSER")
	if err != nil {
		t.Fatalf("GetByUsername (uppercase) failed: %v", err)
	}

	if got2.ID != account.ID {
		t.Errorf("ID: got %q, want %q", got2.ID, account.ID)
	}
}

func TestAccountsStore_GetByEmail(t *testing.T) {
	db := setupTestStore(t)
	store := NewAccountsStore(db)
	ctx := context.Background()

	account := &accounts.Account{
		ID:        newTestID(),
		Username:  "testuser",
		Email:     "Test@Example.com",
		CreatedAt: testTime(),
		UpdatedAt: testTime(),
	}

	if err := store.Insert(ctx, account, "hash"); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Test case-insensitive lookup
	got, err := store.GetByEmail(ctx, "test@example.com")
	if err != nil {
		t.Fatalf("GetByEmail failed: %v", err)
	}

	if got.ID != account.ID {
		t.Errorf("ID: got %q, want %q", got.ID, account.ID)
	}
}

func TestAccountsStore_Update(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	store := NewAccountsStore(db)

	// Update fields
	update := &accounts.UpdateIn{
		DisplayName:  ptr("Updated Name"),
		Bio:          ptr("Updated bio"),
		AvatarURL:    ptr("https://example.com/new-avatar.png"),
		Location:     ptr("New City"),
		Private:      ptr(true),
		Discoverable: ptr(false),
	}

	if err := store.Update(ctx, account.ID, update); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify
	got, err := store.GetByID(ctx, account.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if got.DisplayName != "Updated Name" {
		t.Errorf("DisplayName: got %q, want %q", got.DisplayName, "Updated Name")
	}
	if got.Bio != "Updated bio" {
		t.Errorf("Bio: got %q, want %q", got.Bio, "Updated bio")
	}
	if got.AvatarURL != "https://example.com/new-avatar.png" {
		t.Errorf("AvatarURL: got %q, want %q", got.AvatarURL, "https://example.com/new-avatar.png")
	}
	if got.Location != "New City" {
		t.Errorf("Location: got %q, want %q", got.Location, "New City")
	}
	if !got.Private {
		t.Error("Private: got false, want true")
	}
	if got.Discoverable {
		t.Error("Discoverable: got true, want false")
	}
}

func TestAccountsStore_ExistsUsername(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	createTestAccount(t, db, "existinguser")
	store := NewAccountsStore(db)

	// Should exist
	exists, err := store.ExistsUsername(ctx, "existinguser")
	if err != nil {
		t.Fatalf("ExistsUsername failed: %v", err)
	}
	if !exists {
		t.Error("ExistsUsername: got false, want true")
	}

	// Should exist (case-insensitive)
	exists2, err := store.ExistsUsername(ctx, "EXISTINGUSER")
	if err != nil {
		t.Fatalf("ExistsUsername (uppercase) failed: %v", err)
	}
	if !exists2 {
		t.Error("ExistsUsername (uppercase): got false, want true")
	}

	// Should not exist
	exists3, err := store.ExistsUsername(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("ExistsUsername (nonexistent) failed: %v", err)
	}
	if exists3 {
		t.Error("ExistsUsername (nonexistent): got true, want false")
	}
}

func TestAccountsStore_ExistsEmail(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	createTestAccount(t, db, "testuser")
	store := NewAccountsStore(db)

	// Should exist
	exists, err := store.ExistsEmail(ctx, "testuser@example.com")
	if err != nil {
		t.Fatalf("ExistsEmail failed: %v", err)
	}
	if !exists {
		t.Error("ExistsEmail: got false, want true")
	}

	// Should not exist
	exists2, err := store.ExistsEmail(ctx, "nonexistent@example.com")
	if err != nil {
		t.Fatalf("ExistsEmail (nonexistent) failed: %v", err)
	}
	if exists2 {
		t.Error("ExistsEmail (nonexistent): got true, want false")
	}
}

func TestAccountsStore_GetPasswordHash(t *testing.T) {
	db := setupTestStore(t)
	store := NewAccountsStore(db)
	ctx := context.Background()

	account := &accounts.Account{
		ID:        newTestID(),
		Username:  "hashtest",
		Email:     "hash@example.com",
		Suspended: false,
		CreatedAt: testTime(),
		UpdatedAt: testTime(),
	}

	if err := store.Insert(ctx, account, "secrethash"); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Get by username
	id, hash, suspended, err := store.GetPasswordHash(ctx, "hashtest")
	if err != nil {
		t.Fatalf("GetPasswordHash failed: %v", err)
	}
	if id != account.ID {
		t.Errorf("ID: got %q, want %q", id, account.ID)
	}
	if hash != "secrethash" {
		t.Errorf("hash: got %q, want %q", hash, "secrethash")
	}
	if suspended {
		t.Error("suspended: got true, want false")
	}

	// Get by email
	id2, hash2, _, err := store.GetPasswordHash(ctx, "hash@example.com")
	if err != nil {
		t.Fatalf("GetPasswordHash (email) failed: %v", err)
	}
	if id2 != account.ID {
		t.Errorf("ID (email): got %q, want %q", id2, account.ID)
	}
	if hash2 != "secrethash" {
		t.Errorf("hash (email): got %q, want %q", hash2, "secrethash")
	}
}

func TestAccountsStore_List(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	// Create multiple accounts
	for i := 0; i < 5; i++ {
		createTestAccount(t, db, "user"+string(rune('a'+i)))
	}

	store := NewAccountsStore(db)

	// List all
	list, total, err := store.List(ctx, 10, 0)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != 5 {
		t.Errorf("List count: got %d, want 5", len(list))
	}
	if total != 5 {
		t.Errorf("Total: got %d, want 5", total)
	}

	// List with limit
	list2, total2, err := store.List(ctx, 2, 0)
	if err != nil {
		t.Fatalf("List (limit) failed: %v", err)
	}
	if len(list2) != 2 {
		t.Errorf("List (limit) count: got %d, want 2", len(list2))
	}
	if total2 != 5 {
		t.Errorf("Total (limit): got %d, want 5", total2)
	}

	// List with offset
	list3, _, err := store.List(ctx, 10, 3)
	if err != nil {
		t.Fatalf("List (offset) failed: %v", err)
	}
	if len(list3) != 2 {
		t.Errorf("List (offset) count: got %d, want 2", len(list3))
	}
}

func TestAccountsStore_Search(t *testing.T) {
	db := setupTestStore(t)
	store := NewAccountsStore(db)
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
			DisplayName:  a.displayName,
			Discoverable: true,
			Suspended:    false,
			CreatedAt:    testTime(),
			UpdatedAt:    testTime(),
		}
		if err := store.Insert(ctx, account, "hash"); err != nil {
			t.Fatalf("Insert account %s failed: %v", a.username, err)
		}
	}

	// Search for "al" - should find alice and alex
	results, err := store.Search(ctx, "al", 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Search 'al' count: got %d, want 2", len(results))
	}

	// Search for "smith" - should find alice (by display name)
	results2, err := store.Search(ctx, "smith", 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results2) != 1 {
		t.Errorf("Search 'smith' count: got %d, want 1", len(results2))
	}
}

func TestAccountsStore_SetVerified(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	store := NewAccountsStore(db)

	// Set verified to true
	if err := store.SetVerified(ctx, account.ID, true); err != nil {
		t.Fatalf("SetVerified failed: %v", err)
	}

	got, _ := store.GetByID(ctx, account.ID)
	if !got.Verified {
		t.Error("Verified: got false, want true")
	}

	// Set verified to false
	if err := store.SetVerified(ctx, account.ID, false); err != nil {
		t.Fatalf("SetVerified (false) failed: %v", err)
	}

	got2, _ := store.GetByID(ctx, account.ID)
	if got2.Verified {
		t.Error("Verified: got true, want false")
	}
}

func TestAccountsStore_SetSuspended(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	store := NewAccountsStore(db)

	// Set suspended to true
	if err := store.SetSuspended(ctx, account.ID, true); err != nil {
		t.Fatalf("SetSuspended failed: %v", err)
	}

	got, _ := store.GetByID(ctx, account.ID)
	if !got.Suspended {
		t.Error("Suspended: got false, want true")
	}

	// Set suspended to false
	if err := store.SetSuspended(ctx, account.ID, false); err != nil {
		t.Fatalf("SetSuspended (false) failed: %v", err)
	}

	got2, _ := store.GetByID(ctx, account.ID)
	if got2.Suspended {
		t.Error("Suspended: got true, want false")
	}
}

func TestAccountsStore_SetAdmin(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	store := NewAccountsStore(db)

	// Set admin to true
	if err := store.SetAdmin(ctx, account.ID, true); err != nil {
		t.Fatalf("SetAdmin failed: %v", err)
	}

	got, _ := store.GetByID(ctx, account.ID)
	if !got.Admin {
		t.Error("Admin: got false, want true")
	}
}

func TestAccountsStore_GetFollowersCount(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	user1 := createTestAccount(t, db, "user1")
	user2 := createTestAccount(t, db, "user2")
	user3 := createTestAccount(t, db, "user3")

	// Create follows (user2 and user3 follow user1)
	relStore := NewRelationshipsStore(db)
	relStore.InsertFollow(ctx, &relationships.Follow{
		ID:          newTestID(),
		FollowerID:  user2.ID,
		FollowingID: user1.ID,
		Pending:     false,
		CreatedAt:   testTime(),
	})
	relStore.InsertFollow(ctx, &relationships.Follow{
		ID:          newTestID(),
		FollowerID:  user3.ID,
		FollowingID: user1.ID,
		Pending:     false,
		CreatedAt:   testTime(),
	})

	store := NewAccountsStore(db)
	count, err := store.GetFollowersCount(ctx, user1.ID)
	if err != nil {
		t.Fatalf("GetFollowersCount failed: %v", err)
	}

	if count != 2 {
		t.Errorf("FollowersCount: got %d, want 2", count)
	}
}

func TestAccountsStore_GetFollowingCount(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	user1 := createTestAccount(t, db, "user1")
	user2 := createTestAccount(t, db, "user2")
	user3 := createTestAccount(t, db, "user3")

	// Create follows (user1 follows user2 and user3)
	relStore := NewRelationshipsStore(db)
	relStore.InsertFollow(ctx, &relationships.Follow{
		ID:          newTestID(),
		FollowerID:  user1.ID,
		FollowingID: user2.ID,
		Pending:     false,
		CreatedAt:   testTime(),
	})
	relStore.InsertFollow(ctx, &relationships.Follow{
		ID:          newTestID(),
		FollowerID:  user1.ID,
		FollowingID: user3.ID,
		Pending:     false,
		CreatedAt:   testTime(),
	})

	store := NewAccountsStore(db)
	count, err := store.GetFollowingCount(ctx, user1.ID)
	if err != nil {
		t.Fatalf("GetFollowingCount failed: %v", err)
	}

	if count != 2 {
		t.Errorf("FollowingCount: got %d, want 2", count)
	}
}

func TestAccountsStore_GetPostsCount(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	user := createTestAccount(t, db, "testuser")
	postsStore := NewPostsStore(db)

	// Create posts
	for i := 0; i < 3; i++ {
		createTestPost(t, postsStore, user.ID)
	}

	store := NewAccountsStore(db)
	count, err := store.GetPostsCount(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetPostsCount failed: %v", err)
	}

	if count != 3 {
		t.Errorf("PostsCount: got %d, want 3", count)
	}
}

func TestAccountsStore_Session_CRUD(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	store := NewAccountsStore(db)

	// Create session
	session := &accounts.Session{
		ID:        newTestID(),
		AccountID: account.ID,
		Token:     "test-token-123",
		UserAgent: "TestBrowser/1.0",
		IPAddress: "127.0.0.1",
		ExpiresAt: testTime().Add(24 * time.Hour),
		CreatedAt: testTime(),
	}

	if err := store.CreateSession(ctx, session); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Get session by token
	got, err := store.GetSession(ctx, session.Token)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
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
	if got.IPAddress != session.IPAddress {
		t.Errorf("IPAddress: got %q, want %q", got.IPAddress, session.IPAddress)
	}

	// Delete session
	if err := store.DeleteSession(ctx, session.Token); err != nil {
		t.Fatalf("DeleteSession failed: %v", err)
	}

	// Verify deleted
	_, err = store.GetSession(ctx, session.Token)
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows after delete, got %v", err)
	}
}

func TestAccountsStore_DeleteExpiredSessions(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	store := NewAccountsStore(db)

	// Create expired session
	expired := &accounts.Session{
		ID:        newTestID(),
		AccountID: account.ID,
		Token:     "expired-token",
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Already expired
		CreatedAt: testTime(),
	}
	if err := store.CreateSession(ctx, expired); err != nil {
		t.Fatalf("CreateSession (expired) failed: %v", err)
	}

	// Create valid session
	valid := &accounts.Session{
		ID:        newTestID(),
		AccountID: account.ID,
		Token:     "valid-token",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: testTime(),
	}
	if err := store.CreateSession(ctx, valid); err != nil {
		t.Fatalf("CreateSession (valid) failed: %v", err)
	}

	// Delete expired sessions
	if err := store.DeleteExpiredSessions(ctx); err != nil {
		t.Fatalf("DeleteExpiredSessions failed: %v", err)
	}

	// Expired should be gone
	_, err := store.GetSession(ctx, expired.Token)
	if err != sql.ErrNoRows {
		t.Error("expired session should have been deleted")
	}

	// Valid should still exist
	_, err = store.GetSession(ctx, valid.Token)
	if err != nil {
		t.Error("valid session should still exist")
	}
}
