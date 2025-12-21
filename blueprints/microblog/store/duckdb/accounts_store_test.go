package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/microblog/feature/accounts"
)

func TestAccountsStore_Insert(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewAccountsStore(db)
	ctx := context.Background()

	now := time.Now().Truncate(time.Microsecond)
	acct := &accounts.Account{
		ID:          "01ABCDEFGHJK",
		Username:    "testuser",
		DisplayName: "Test User",
		Email:       "test@example.com",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	err := store.Insert(ctx, acct, "hashed_password")
	if err != nil {
		t.Fatalf("Insert() error = %v", err)
	}

	// Verify insert
	got, err := store.GetByID(ctx, acct.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if got.Username != acct.Username {
		t.Errorf("Username = %s, want %s", got.Username, acct.Username)
	}
	if got.DisplayName != acct.DisplayName {
		t.Errorf("DisplayName = %s, want %s", got.DisplayName, acct.DisplayName)
	}
	if got.Email != acct.Email {
		t.Errorf("Email = %s, want %s", got.Email, acct.Email)
	}
}

func TestAccountsStore_Insert_Duplicate(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewAccountsStore(db)
	ctx := context.Background()

	now := time.Now()
	acct := &accounts.Account{
		ID:        "01ABCDEFGHJK",
		Username:  "testuser",
		CreatedAt: now,
		UpdatedAt: now,
	}

	// First insert should succeed
	if err := store.Insert(ctx, acct, "hash"); err != nil {
		t.Fatalf("First Insert() error = %v", err)
	}

	// Second insert with same username should fail
	acct2 := &accounts.Account{
		ID:        "01DIFFERENT",
		Username:  "testuser", // Same username
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := store.Insert(ctx, acct2, "hash"); err == nil {
		t.Error("Expected duplicate username error, got nil")
	}
}

func TestAccountsStore_GetByID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewAccountsStore(db)
	ctx := context.Background()

	// Insert test account with basic fields (bio is set via Update)
	acct := &accounts.Account{
		ID:          "01ABCDEFGHJK",
		Username:    "testuser",
		DisplayName: "Test User",
		Email:       "getbyid@example.com",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := store.Insert(ctx, acct, "hash"); err != nil {
		t.Fatalf("Insert() error = %v", err)
	}

	// Update bio since Insert only sets basic fields
	bio := "Hello world"
	if err := store.Update(ctx, acct.ID, &accounts.UpdateIn{Bio: &bio}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Set verified via SetVerified
	if err := store.SetVerified(ctx, acct.ID, true); err != nil {
		t.Fatalf("SetVerified() error = %v", err)
	}

	// Test GetByID
	got, err := store.GetByID(ctx, acct.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if got.ID != acct.ID {
		t.Errorf("ID = %s, want %s", got.ID, acct.ID)
	}
	if got.Username != acct.Username {
		t.Errorf("Username = %s, want %s", got.Username, acct.Username)
	}
	if got.Bio != bio {
		t.Errorf("Bio = %s, want %s", got.Bio, bio)
	}
	if !got.Verified {
		t.Errorf("Verified = %v, want true", got.Verified)
	}
}

func TestAccountsStore_GetByID_NotFound(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewAccountsStore(db)
	ctx := context.Background()

	_, err := store.GetByID(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent account, got nil")
	}
}

func TestAccountsStore_GetByUsername(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewAccountsStore(db)
	ctx := context.Background()

	// Insert test account
	acct := &accounts.Account{
		ID:        "01ABCDEFGHJK",
		Username:  "testuser",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := store.Insert(ctx, acct, "hash"); err != nil {
		t.Fatalf("Insert() error = %v", err)
	}

	tests := []struct {
		name     string
		username string
		wantID   string
		wantErr  bool
	}{
		{"exact match", "testuser", "01ABCDEFGHJK", false},
		{"case insensitive", "TestUser", "01ABCDEFGHJK", false},
		{"uppercase", "TESTUSER", "01ABCDEFGHJK", false},
		{"not found", "nobody", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := store.GetByUsername(ctx, tt.username)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("GetByUsername() error = %v", err)
			}
			if got.ID != tt.wantID {
				t.Errorf("ID = %s, want %s", got.ID, tt.wantID)
			}
		})
	}
}

func TestAccountsStore_GetByEmail(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewAccountsStore(db)
	ctx := context.Background()

	// Insert test account
	acct := &accounts.Account{
		ID:        "01ABCDEFGHJK",
		Username:  "testuser",
		Email:     "test@example.com",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := store.Insert(ctx, acct, "hash"); err != nil {
		t.Fatalf("Insert() error = %v", err)
	}

	tests := []struct {
		name    string
		email   string
		wantID  string
		wantErr bool
	}{
		{"exact match", "test@example.com", "01ABCDEFGHJK", false},
		{"case insensitive", "Test@Example.COM", "01ABCDEFGHJK", false},
		{"not found", "other@example.com", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := store.GetByEmail(ctx, tt.email)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("GetByEmail() error = %v", err)
			}
			if got.ID != tt.wantID {
				t.Errorf("ID = %s, want %s", got.ID, tt.wantID)
			}
		})
	}
}

func TestAccountsStore_Update(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewAccountsStore(db)
	ctx := context.Background()

	// Insert test account
	acct := &accounts.Account{
		ID:          "01ABCDEFGHJK",
		Username:    "testuser",
		DisplayName: "Original Name",
		Bio:         "Original bio",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := store.Insert(ctx, acct, "hash"); err != nil {
		t.Fatalf("Insert() error = %v", err)
	}

	// Update display name and bio
	newDisplayName := "New Display Name"
	newBio := "Updated bio"
	err := store.Update(ctx, acct.ID, &accounts.UpdateIn{
		DisplayName: &newDisplayName,
		Bio:         &newBio,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Verify update
	got, err := store.GetByID(ctx, acct.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if got.DisplayName != newDisplayName {
		t.Errorf("DisplayName = %s, want %s", got.DisplayName, newDisplayName)
	}
	if got.Bio != newBio {
		t.Errorf("Bio = %s, want %s", got.Bio, newBio)
	}
}

func TestAccountsStore_Update_EmptyNoOp(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewAccountsStore(db)
	ctx := context.Background()

	// Insert test account
	acct := &accounts.Account{
		ID:        "01ABCDEFGHJK",
		Username:  "testuser",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := store.Insert(ctx, acct, "hash"); err != nil {
		t.Fatalf("Insert() error = %v", err)
	}

	// Update with nothing should not error
	err := store.Update(ctx, acct.ID, &accounts.UpdateIn{})
	if err != nil {
		t.Errorf("Update() with empty input error = %v", err)
	}
}

func TestAccountsStore_ExistsUsername(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewAccountsStore(db)
	ctx := context.Background()

	// Insert test account
	acct := &accounts.Account{
		ID:        "01ABCDEFGHJK",
		Username:  "testuser",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := store.Insert(ctx, acct, "hash"); err != nil {
		t.Fatalf("Insert() error = %v", err)
	}

	tests := []struct {
		username string
		want     bool
	}{
		{"testuser", true},
		{"TestUser", true}, // case insensitive
		{"nobody", false},
	}

	for _, tt := range tests {
		t.Run(tt.username, func(t *testing.T) {
			got, err := store.ExistsUsername(ctx, tt.username)
			if err != nil {
				t.Fatalf("ExistsUsername() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("ExistsUsername(%s) = %v, want %v", tt.username, got, tt.want)
			}
		})
	}
}

func TestAccountsStore_ExistsEmail(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewAccountsStore(db)
	ctx := context.Background()

	// Insert test account
	acct := &accounts.Account{
		ID:        "01ABCDEFGHJK",
		Username:  "testuser",
		Email:     "test@example.com",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := store.Insert(ctx, acct, "hash"); err != nil {
		t.Fatalf("Insert() error = %v", err)
	}

	tests := []struct {
		email string
		want  bool
	}{
		{"test@example.com", true},
		{"TEST@EXAMPLE.COM", true}, // case insensitive
		{"other@example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			got, err := store.ExistsEmail(ctx, tt.email)
			if err != nil {
				t.Fatalf("ExistsEmail() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("ExistsEmail(%s) = %v, want %v", tt.email, got, tt.want)
			}
		})
	}
}

func TestAccountsStore_GetPasswordHash(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewAccountsStore(db)
	ctx := context.Background()

	// Insert test account
	passwordHash := "$argon2id$v=19$m=65536,t=1,p=4$salt$hash"
	acct := &accounts.Account{
		ID:        "01ABCDEFGHJK",
		Username:  "testuser",
		Email:     "test@example.com",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := store.Insert(ctx, acct, passwordHash); err != nil {
		t.Fatalf("Insert() error = %v", err)
	}

	// Test by username
	id, hash, suspended, err := store.GetPasswordHash(ctx, "testuser")
	if err != nil {
		t.Fatalf("GetPasswordHash(username) error = %v", err)
	}
	if id != acct.ID {
		t.Errorf("ID = %s, want %s", id, acct.ID)
	}
	if hash != passwordHash {
		t.Errorf("hash = %s, want %s", hash, passwordHash)
	}
	if suspended {
		t.Error("suspended = true, want false")
	}

	// Test by email
	id, hash, suspended, err = store.GetPasswordHash(ctx, "test@example.com")
	if err != nil {
		t.Fatalf("GetPasswordHash(email) error = %v", err)
	}
	if id != acct.ID {
		t.Errorf("ID = %s, want %s", id, acct.ID)
	}
}

func TestAccountsStore_List(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewAccountsStore(db)
	ctx := context.Background()

	// Insert test accounts with unique emails
	for i, username := range []string{"alice", "bob", "charlie"} {
		acct := &accounts.Account{
			ID:        "id-" + string(rune('a'+i)),
			Username:  username,
			Email:     username + "@example.com",
			CreatedAt: time.Now().Add(time.Duration(i) * time.Second),
			UpdatedAt: time.Now(),
		}
		if err := store.Insert(ctx, acct, "hash"); err != nil {
			t.Fatalf("Insert() error = %v", err)
		}
	}

	// List all
	list, total, err := store.List(ctx, 10, 0)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
	if len(list) != 3 {
		t.Errorf("len(list) = %d, want 3", len(list))
	}

	// List with pagination
	list, total, err = store.List(ctx, 2, 0)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
	if len(list) != 2 {
		t.Errorf("len(list) = %d, want 2", len(list))
	}
}

func TestAccountsStore_Search(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewAccountsStore(db)
	ctx := context.Background()

	// Insert test accounts
	testAccounts := []struct {
		id          string
		username    string
		displayName string
	}{
		{"id-1", "alice", "Alice Wonder"},
		{"id-2", "bob", "Bob Builder"},
		{"id-3", "charlie", "Charlie Brown"},
		{"id-4", "alice_wonderland", "Alice Wonderland"},
	}
	for _, a := range testAccounts {
		acct := &accounts.Account{
			ID:          a.id,
			Username:    a.username,
			DisplayName: a.displayName,
			Email:       a.username + "@example.com",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		if err := store.Insert(ctx, acct, "hash"); err != nil {
			t.Fatalf("Insert() error = %v", err)
		}
	}

	tests := []struct {
		query     string
		wantCount int
	}{
		{"alice", 2},    // matches alice and alice_wonderland
		{"bob", 1},      // matches bob
		{"wonder", 2},   // matches display names
		{"nobody", 0},   // no matches
		{"Charlie", 1},  // case insensitive
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			results, err := store.Search(ctx, tt.query, 10)
			if err != nil {
				t.Fatalf("Search() error = %v", err)
			}
			if len(results) != tt.wantCount {
				t.Errorf("Search(%s) returned %d results, want %d", tt.query, len(results), tt.wantCount)
			}
		})
	}
}

func TestAccountsStore_SetVerified(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewAccountsStore(db)
	ctx := context.Background()

	// Insert test account
	acct := &accounts.Account{
		ID:        "01ABCDEFGHJK",
		Username:  "testuser",
		Verified:  false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := store.Insert(ctx, acct, "hash"); err != nil {
		t.Fatalf("Insert() error = %v", err)
	}

	// Set verified
	if err := store.SetVerified(ctx, acct.ID, true); err != nil {
		t.Fatalf("SetVerified() error = %v", err)
	}

	// Verify
	got, _ := store.GetByID(ctx, acct.ID)
	if !got.Verified {
		t.Error("Verified = false, want true")
	}

	// Unset verified
	if err := store.SetVerified(ctx, acct.ID, false); err != nil {
		t.Fatalf("SetVerified() error = %v", err)
	}

	got, _ = store.GetByID(ctx, acct.ID)
	if got.Verified {
		t.Error("Verified = true, want false")
	}
}

func TestAccountsStore_SetSuspended(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewAccountsStore(db)
	ctx := context.Background()

	// Insert test account
	acct := &accounts.Account{
		ID:        "01ABCDEFGHJK",
		Username:  "testuser",
		Suspended: false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := store.Insert(ctx, acct, "hash"); err != nil {
		t.Fatalf("Insert() error = %v", err)
	}

	// Suspend
	if err := store.SetSuspended(ctx, acct.ID, true); err != nil {
		t.Fatalf("SetSuspended() error = %v", err)
	}

	got, _ := store.GetByID(ctx, acct.ID)
	if !got.Suspended {
		t.Error("Suspended = false, want true")
	}

	// Suspended accounts should not appear in list
	list, total, _ := store.List(ctx, 10, 0)
	if total != 0 {
		t.Errorf("List total = %d, want 0 (suspended excluded)", total)
	}
	if len(list) != 0 {
		t.Errorf("List len = %d, want 0", len(list))
	}
}

func TestAccountsStore_SetAdmin(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewAccountsStore(db)
	ctx := context.Background()

	// Insert test account
	acct := &accounts.Account{
		ID:        "01ABCDEFGHJK",
		Username:  "testuser",
		Admin:     false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := store.Insert(ctx, acct, "hash"); err != nil {
		t.Fatalf("Insert() error = %v", err)
	}

	// Set admin
	if err := store.SetAdmin(ctx, acct.ID, true); err != nil {
		t.Fatalf("SetAdmin() error = %v", err)
	}

	got, _ := store.GetByID(ctx, acct.ID)
	if !got.Admin {
		t.Error("Admin = false, want true")
	}
}

func TestAccountsStore_Sessions(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewAccountsStore(db)
	ctx := context.Background()

	// Insert test account
	acct := &accounts.Account{
		ID:        "01ABCDEFGHJK",
		Username:  "testuser",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := store.Insert(ctx, acct, "hash"); err != nil {
		t.Fatalf("Insert() error = %v", err)
	}

	// Create session
	sess := &accounts.Session{
		ID:        "sess-id",
		AccountID: acct.ID,
		Token:     "test-token-12345",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}
	if err := store.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	// Get session
	got, err := store.GetSession(ctx, sess.Token)
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if got.ID != sess.ID {
		t.Errorf("ID = %s, want %s", got.ID, sess.ID)
	}
	if got.AccountID != sess.AccountID {
		t.Errorf("AccountID = %s, want %s", got.AccountID, sess.AccountID)
	}

	// Delete session
	if err := store.DeleteSession(ctx, sess.Token); err != nil {
		t.Fatalf("DeleteSession() error = %v", err)
	}

	// Session should be gone
	_, err = store.GetSession(ctx, sess.Token)
	if err == nil {
		t.Error("expected error for deleted session, got nil")
	}
}

func TestAccountsStore_Sessions_Expired(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewAccountsStore(db)
	ctx := context.Background()

	// Insert test account
	acct := &accounts.Account{
		ID:        "01ABCDEFGHJK",
		Username:  "testuser",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := store.Insert(ctx, acct, "hash"); err != nil {
		t.Fatalf("Insert() error = %v", err)
	}

	// Create expired session
	sess := &accounts.Session{
		ID:        "sess-id",
		AccountID: acct.ID,
		Token:     "expired-token",
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Already expired
		CreatedAt: time.Now(),
	}
	if err := store.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	// Expired session should not be returned
	_, err := store.GetSession(ctx, sess.Token)
	if err == nil {
		t.Error("expected error for expired session, got nil")
	}
}

func TestAccountsStore_FullLifecycle(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewAccountsStore(db)
	ctx := context.Background()

	// Create account
	acct := &accounts.Account{
		ID:          "01LIFECYCLE",
		Username:    "lifecycle",
		DisplayName: "Lifecycle Test",
		Email:       "lifecycle@example.com",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := store.Insert(ctx, acct, "initial_hash"); err != nil {
		t.Fatalf("Insert() error = %v", err)
	}

	// Read back
	got, err := store.GetByID(ctx, acct.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.Username != acct.Username {
		t.Errorf("Username = %s, want %s", got.Username, acct.Username)
	}

	// Update profile
	newBio := "Updated bio for lifecycle test"
	if err := store.Update(ctx, acct.ID, &accounts.UpdateIn{Bio: &newBio}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got, _ = store.GetByID(ctx, acct.ID)
	if got.Bio != newBio {
		t.Errorf("Bio = %s, want %s", got.Bio, newBio)
	}

	// Verify account
	if err := store.SetVerified(ctx, acct.ID, true); err != nil {
		t.Fatalf("SetVerified() error = %v", err)
	}

	// Make admin
	if err := store.SetAdmin(ctx, acct.ID, true); err != nil {
		t.Fatalf("SetAdmin() error = %v", err)
	}

	// Check final state
	got, _ = store.GetByID(ctx, acct.ID)
	if !got.Verified {
		t.Error("Verified = false, want true")
	}
	if !got.Admin {
		t.Error("Admin = false, want true")
	}
	if got.Bio != newBio {
		t.Errorf("Bio = %s, want %s", got.Bio, newBio)
	}

	// Create session with explicit future expiry
	sess := &accounts.Session{
		ID:        "lifecycle-sess",
		AccountID: acct.ID,
		Token:     "lifecycle-token",
		ExpiresAt: time.Now().Add(24 * time.Hour), // 24 hours in future
		CreatedAt: time.Now(),
	}
	if err := store.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	// Verify session works
	gotSess, err := store.GetSession(ctx, sess.Token)
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if gotSess.AccountID != acct.ID {
		t.Errorf("Session AccountID = %s, want %s", gotSess.AccountID, acct.ID)
	}

	// List should include our account
	list, total, _ := store.List(ctx, 10, 0)
	if total != 1 {
		t.Errorf("List total = %d, want 1", total)
	}
	if len(list) != 1 {
		t.Errorf("List len = %d, want 1", len(list))
	}
}
