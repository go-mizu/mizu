package duckdb

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

// ============================================================
// Share CRUD Tests
// ============================================================

func TestCreateShare_UserShare(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user1 := newTestUser("user1", "alice@example.com")
	user2 := newTestUser("user2", "bob@example.com")
	store.CreateUser(ctx, user1)
	store.CreateUser(ctx, user2)

	file := newTestFile("file1", "user1", "document.pdf")
	store.CreateFile(ctx, file)

	share := newTestShare("share1", "user1", "file", "file1")
	share.SharedWithID = sql.NullString{String: "user2", Valid: true}
	share.Permission = "editor"

	if err := store.CreateShare(ctx, share); err != nil {
		t.Fatalf("create share failed: %v", err)
	}

	got, err := store.GetShareByID(ctx, "share1")
	if err != nil {
		t.Fatalf("get share failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected share, got nil")
	}
	if got.SharedWithID.String != "user2" {
		t.Errorf("expected shared_with_id user2, got %s", got.SharedWithID.String)
	}
	if got.Permission != "editor" {
		t.Errorf("expected permission editor, got %s", got.Permission)
	}
}

func TestCreateShare_LinkShare(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "alice@example.com")
	store.CreateUser(ctx, user)

	file := newTestFile("file1", "user1", "document.pdf")
	store.CreateFile(ctx, file)

	share := newTestShare("share1", "user1", "file", "file1")
	share.LinkToken = sql.NullString{String: "abc123token", Valid: true}
	share.Permission = "viewer"

	if err := store.CreateShare(ctx, share); err != nil {
		t.Fatalf("create share failed: %v", err)
	}

	got, _ := store.GetShareByID(ctx, "share1")
	if !got.LinkToken.Valid || got.LinkToken.String != "abc123token" {
		t.Errorf("expected link_token abc123token, got %v", got.LinkToken)
	}
}

func TestGetShareByID_Exists(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "alice@example.com")
	store.CreateUser(ctx, user)

	file := newTestFile("file1", "user1", "document.pdf")
	store.CreateFile(ctx, file)

	share := newTestShare("share1", "user1", "file", "file1")
	store.CreateShare(ctx, share)

	got, err := store.GetShareByID(ctx, "share1")
	if err != nil {
		t.Fatalf("get share failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected share, got nil")
	}
	if got.ID != "share1" {
		t.Errorf("expected ID share1, got %s", got.ID)
	}
}

func TestGetShareByID_NotFound(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	got, err := store.GetShareByID(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("get share failed: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestGetShareByToken_Exists(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "alice@example.com")
	store.CreateUser(ctx, user)

	file := newTestFile("file1", "user1", "document.pdf")
	store.CreateFile(ctx, file)

	share := newTestShare("share1", "user1", "file", "file1")
	share.LinkToken = sql.NullString{String: "secrettoken", Valid: true}
	store.CreateShare(ctx, share)

	got, err := store.GetShareByToken(ctx, "secrettoken")
	if err != nil {
		t.Fatalf("get share by token failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected share, got nil")
	}
	if got.LinkToken.String != "secrettoken" {
		t.Errorf("expected token secrettoken, got %s", got.LinkToken.String)
	}
}

func TestGetShareByToken_NotFound(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	got, err := store.GetShareByToken(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("get share by token failed: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestUpdateShare_ChangePermission(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "alice@example.com")
	store.CreateUser(ctx, user)

	file := newTestFile("file1", "user1", "document.pdf")
	store.CreateFile(ctx, file)

	share := newTestShare("share1", "user1", "file", "file1")
	share.Permission = "viewer"
	store.CreateShare(ctx, share)

	share.Permission = "editor"
	share.UpdatedAt = time.Now().Truncate(time.Microsecond)
	if err := store.UpdateShare(ctx, share); err != nil {
		t.Fatalf("update share failed: %v", err)
	}

	got, _ := store.GetShareByID(ctx, "share1")
	if got.Permission != "editor" {
		t.Errorf("expected permission editor, got %s", got.Permission)
	}
}

func TestDeleteShare_Success(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "alice@example.com")
	store.CreateUser(ctx, user)

	file := newTestFile("file1", "user1", "document.pdf")
	store.CreateFile(ctx, file)

	share := newTestShare("share1", "user1", "file", "file1")
	store.CreateShare(ctx, share)

	if err := store.DeleteShare(ctx, "share1"); err != nil {
		t.Fatalf("delete share failed: %v", err)
	}

	got, _ := store.GetShareByID(ctx, "share1")
	if got != nil {
		t.Errorf("expected nil after delete, got %+v", got)
	}
}

// ============================================================
// Share Listing Tests
// ============================================================

func TestListSharesByOwner(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user1 := newTestUser("user1", "alice@example.com")
	user2 := newTestUser("user2", "bob@example.com")
	store.CreateUser(ctx, user1)
	store.CreateUser(ctx, user2)

	file1 := newTestFile("file1", "user1", "doc1.pdf")
	file2 := newTestFile("file2", "user1", "doc2.pdf")
	file3 := newTestFile("file3", "user2", "doc3.pdf")
	store.CreateFile(ctx, file1)
	store.CreateFile(ctx, file2)
	store.CreateFile(ctx, file3)

	share1 := newTestShare("share1", "user1", "file", "file1")
	share2 := newTestShare("share2", "user1", "file", "file2")
	share3 := newTestShare("share3", "user2", "file", "file3")
	store.CreateShare(ctx, share1)
	store.CreateShare(ctx, share2)
	store.CreateShare(ctx, share3)

	shares, err := store.ListSharesByOwner(ctx, "user1")
	if err != nil {
		t.Fatalf("list shares by owner failed: %v", err)
	}
	if len(shares) != 2 {
		t.Errorf("expected 2 shares for user1, got %d", len(shares))
	}
}

func TestListSharesWithUser(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user1 := newTestUser("user1", "alice@example.com")
	user2 := newTestUser("user2", "bob@example.com")
	user3 := newTestUser("user3", "charlie@example.com")
	store.CreateUser(ctx, user1)
	store.CreateUser(ctx, user2)
	store.CreateUser(ctx, user3)

	file := newTestFile("file1", "user1", "document.pdf")
	store.CreateFile(ctx, file)

	share1 := newTestShare("share1", "user1", "file", "file1")
	share1.SharedWithID = sql.NullString{String: "user2", Valid: true}
	share2 := newTestShare("share2", "user1", "file", "file1")
	share2.SharedWithID = sql.NullString{String: "user2", Valid: true}
	share3 := newTestShare("share3", "user1", "file", "file1")
	share3.SharedWithID = sql.NullString{String: "user3", Valid: true}

	store.CreateShare(ctx, share1)
	store.CreateShare(ctx, share2)
	store.CreateShare(ctx, share3)

	shares, err := store.ListSharesWithUser(ctx, "user2")
	if err != nil {
		t.Fatalf("list shares with user failed: %v", err)
	}
	if len(shares) != 2 {
		t.Errorf("expected 2 shares for user2, got %d", len(shares))
	}
}

func TestListSharesForResource(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "alice@example.com")
	store.CreateUser(ctx, user)

	file := newTestFile("file1", "user1", "document.pdf")
	store.CreateFile(ctx, file)

	// Create multiple shares for same file
	share1 := newTestShare("share1", "user1", "file", "file1")
	share1.SharedWithID = sql.NullString{String: "userA", Valid: true}
	share2 := newTestShare("share2", "user1", "file", "file1")
	share2.SharedWithID = sql.NullString{String: "userB", Valid: true}
	share3 := newTestShare("share3", "user1", "file", "file1")
	share3.LinkToken = sql.NullString{String: "publiclink", Valid: true}

	store.CreateShare(ctx, share1)
	store.CreateShare(ctx, share2)
	store.CreateShare(ctx, share3)

	shares, err := store.ListSharesForResource(ctx, "file", "file1")
	if err != nil {
		t.Fatalf("list shares for resource failed: %v", err)
	}
	if len(shares) != 3 {
		t.Errorf("expected 3 shares for file, got %d", len(shares))
	}
}

// ============================================================
// Share Features Tests
// ============================================================

func TestGetShareForUserAndResource(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user1 := newTestUser("user1", "alice@example.com")
	user2 := newTestUser("user2", "bob@example.com")
	store.CreateUser(ctx, user1)
	store.CreateUser(ctx, user2)

	file := newTestFile("file1", "user1", "document.pdf")
	store.CreateFile(ctx, file)

	share := newTestShare("share1", "user1", "file", "file1")
	share.SharedWithID = sql.NullString{String: "user2", Valid: true}
	store.CreateShare(ctx, share)

	got, err := store.GetShareForUserAndResource(ctx, "user2", "file", "file1")
	if err != nil {
		t.Fatalf("get share for user and resource failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected share, got nil")
	}
	if got.SharedWithID.String != "user2" {
		t.Errorf("expected shared_with_id user2, got %s", got.SharedWithID.String)
	}
}

func TestIncrementDownloadCount(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "alice@example.com")
	store.CreateUser(ctx, user)

	file := newTestFile("file1", "user1", "document.pdf")
	store.CreateFile(ctx, file)

	share := newTestShare("share1", "user1", "file", "file1")
	share.DownloadCount = 0
	store.CreateShare(ctx, share)

	// Increment multiple times
	for i := 0; i < 5; i++ {
		if err := store.IncrementDownloadCount(ctx, "share1"); err != nil {
			t.Fatalf("increment download count failed: %v", err)
		}
	}

	got, _ := store.GetShareByID(ctx, "share1")
	if got.DownloadCount != 5 {
		t.Errorf("expected download_count 5, got %d", got.DownloadCount)
	}
}

func TestDeleteSharesForResource(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "alice@example.com")
	store.CreateUser(ctx, user)

	file := newTestFile("file1", "user1", "document.pdf")
	store.CreateFile(ctx, file)

	// Create multiple shares for the file
	for i := 1; i <= 3; i++ {
		share := newTestShare("share"+string(rune('0'+i)), "user1", "file", "file1")
		store.CreateShare(ctx, share)
	}

	if err := store.DeleteSharesForResource(ctx, "file", "file1"); err != nil {
		t.Fatalf("delete shares for resource failed: %v", err)
	}

	shares, _ := store.ListSharesForResource(ctx, "file", "file1")
	if len(shares) != 0 {
		t.Errorf("expected 0 shares after delete, got %d", len(shares))
	}
}

func TestCleanupExpiredShares(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "alice@example.com")
	store.CreateUser(ctx, user)

	file := newTestFile("file1", "user1", "document.pdf")
	store.CreateFile(ctx, file)

	// Create expired share
	expired := newTestShare("expired", "user1", "file", "file1")
	expired.ExpiresAt = sql.NullTime{Time: time.Now().Add(-24 * time.Hour), Valid: true}
	store.CreateShare(ctx, expired)

	// Create valid share
	valid := newTestShare("valid", "user1", "file", "file1")
	valid.ExpiresAt = sql.NullTime{Time: time.Now().Add(24 * time.Hour), Valid: true}
	store.CreateShare(ctx, valid)

	// Create share with no expiry
	noExpiry := newTestShare("noexpiry", "user1", "file", "file1")
	store.CreateShare(ctx, noExpiry)

	if err := store.CleanupExpiredShares(ctx); err != nil {
		t.Fatalf("cleanup expired shares failed: %v", err)
	}

	shares, _ := store.ListSharesForResource(ctx, "file", "file1")
	if len(shares) != 2 {
		t.Errorf("expected 2 shares after cleanup, got %d", len(shares))
	}
}

// ============================================================
// Business Use Cases - Sharing Scenarios
// ============================================================

func TestShareFlow_UserCollaboration(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	// Setup: Alice owns a file, shares with Bob
	alice := newTestUser("alice", "alice@example.com")
	bob := newTestUser("bob", "bob@example.com")
	store.CreateUser(ctx, alice)
	store.CreateUser(ctx, bob)

	file := newTestFile("file1", "alice", "project_plan.docx")
	store.CreateFile(ctx, file)

	// Alice shares with Bob as editor
	share := newTestShare("share1", "alice", "file", "file1")
	share.SharedWithID = sql.NullString{String: "bob", Valid: true}
	share.Permission = "editor"
	store.CreateShare(ctx, share)

	// Bob can find the share
	bobShares, _ := store.ListSharesWithUser(ctx, "bob")
	if len(bobShares) != 1 {
		t.Fatal("Bob should see 1 shared file")
	}
	if bobShares[0].Permission != "editor" {
		t.Errorf("Bob should have editor permission, got %s", bobShares[0].Permission)
	}

	// Alice can see who she shared with
	aliceShares, _ := store.ListSharesByOwner(ctx, "alice")
	if len(aliceShares) != 1 {
		t.Fatal("Alice should see 1 shared file")
	}
}

func TestShareFlow_PublicLink(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "alice@example.com")
	store.CreateUser(ctx, user)

	file := newTestFile("file1", "user1", "presentation.pptx")
	store.CreateFile(ctx, file)

	// Create public link
	share := newTestShare("share1", "user1", "file", "file1")
	share.LinkToken = sql.NullString{String: "public_abc123", Valid: true}
	share.Permission = "viewer"
	store.CreateShare(ctx, share)

	// Anyone with the link can access
	got, _ := store.GetShareByToken(ctx, "public_abc123")
	if got == nil {
		t.Fatal("should be able to access via link token")
	}
	if got.ResourceID != "file1" {
		t.Errorf("link should point to file1, got %s", got.ResourceID)
	}
}

func TestShareFlow_PasswordProtected(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "alice@example.com")
	store.CreateUser(ctx, user)

	file := newTestFile("file1", "user1", "confidential.pdf")
	store.CreateFile(ctx, file)

	share := newTestShare("share1", "user1", "file", "file1")
	share.LinkToken = sql.NullString{String: "protected_link", Valid: true}
	share.LinkPasswordHash = sql.NullString{String: "hashed_password123", Valid: true}
	store.CreateShare(ctx, share)

	got, _ := store.GetShareByToken(ctx, "protected_link")
	if got == nil {
		t.Fatal("should retrieve password-protected share")
	}
	if !got.LinkPasswordHash.Valid {
		t.Error("share should have password protection")
	}
}

func TestShareFlow_ExpiringLink(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "alice@example.com")
	store.CreateUser(ctx, user)

	file := newTestFile("file1", "user1", "temporary.pdf")
	store.CreateFile(ctx, file)

	share := newTestShare("share1", "user1", "file", "file1")
	share.LinkToken = sql.NullString{String: "temp_link", Valid: true}
	share.ExpiresAt = sql.NullTime{Time: time.Now().Add(7 * 24 * time.Hour), Valid: true} // 7 days

	store.CreateShare(ctx, share)

	got, _ := store.GetShareByToken(ctx, "temp_link")
	if !got.ExpiresAt.Valid {
		t.Error("share should have expiration")
	}

	// Simulate expiration and cleanup
	share.ExpiresAt = sql.NullTime{Time: time.Now().Add(-1 * time.Hour), Valid: true}
	store.UpdateShare(ctx, share)
	store.CleanupExpiredShares(ctx)

	got, _ = store.GetShareByToken(ctx, "temp_link")
	if got != nil {
		t.Error("expired share should be cleaned up")
	}
}

func TestShareFlow_DownloadLimit(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "alice@example.com")
	store.CreateUser(ctx, user)

	file := newTestFile("file1", "user1", "limited.zip")
	store.CreateFile(ctx, file)

	share := newTestShare("share1", "user1", "file", "file1")
	share.LinkToken = sql.NullString{String: "limited_link", Valid: true}
	share.DownloadLimit = sql.NullInt64{Int64: 5, Valid: true}
	share.DownloadCount = 0
	store.CreateShare(ctx, share)

	// Simulate 5 downloads
	for i := 0; i < 5; i++ {
		store.IncrementDownloadCount(ctx, "share1")
	}

	got, _ := store.GetShareByID(ctx, "share1")
	if got.DownloadCount != 5 {
		t.Errorf("expected 5 downloads, got %d", got.DownloadCount)
	}
	if got.DownloadCount >= int(got.DownloadLimit.Int64) {
		// Application logic would disable the link at this point
		t.Log("Download limit reached - link should be disabled")
	}
}

func TestShareFlow_PreventDownload(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	user := newTestUser("user1", "alice@example.com")
	store.CreateUser(ctx, user)

	file := newTestFile("file1", "user1", "viewonly.pdf")
	store.CreateFile(ctx, file)

	share := newTestShare("share1", "user1", "file", "file1")
	share.LinkToken = sql.NullString{String: "viewonly_link", Valid: true}
	share.PreventDownload = true
	store.CreateShare(ctx, share)

	got, _ := store.GetShareByToken(ctx, "viewonly_link")
	if !got.PreventDownload {
		t.Error("share should have download prevention enabled")
	}
}

func TestSharePermissions_ViewerVsEditor(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	owner := newTestUser("owner", "owner@example.com")
	viewer := newTestUser("viewer", "viewer@example.com")
	editor := newTestUser("editor", "editor@example.com")
	store.CreateUser(ctx, owner)
	store.CreateUser(ctx, viewer)
	store.CreateUser(ctx, editor)

	file := newTestFile("file1", "owner", "document.docx")
	store.CreateFile(ctx, file)

	// Share with viewer (read-only)
	viewerShare := newTestShare("share1", "owner", "file", "file1")
	viewerShare.SharedWithID = sql.NullString{String: "viewer", Valid: true}
	viewerShare.Permission = "viewer"
	store.CreateShare(ctx, viewerShare)

	// Share with editor (read-write)
	editorShare := newTestShare("share2", "owner", "file", "file1")
	editorShare.SharedWithID = sql.NullString{String: "editor", Valid: true}
	editorShare.Permission = "editor"
	store.CreateShare(ctx, editorShare)

	// Verify permissions
	viewerAccess, _ := store.GetShareForUserAndResource(ctx, "viewer", "file", "file1")
	if viewerAccess.Permission != "viewer" {
		t.Errorf("viewer should have viewer permission, got %s", viewerAccess.Permission)
	}

	editorAccess, _ := store.GetShareForUserAndResource(ctx, "editor", "file", "file1")
	if editorAccess.Permission != "editor" {
		t.Errorf("editor should have editor permission, got %s", editorAccess.Permission)
	}
}

func TestShareHierarchy_FolderShare(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	owner := newTestUser("owner", "owner@example.com")
	collaborator := newTestUser("collab", "collab@example.com")
	store.CreateUser(ctx, owner)
	store.CreateUser(ctx, collaborator)

	folder := newTestFolder("folder1", "owner", "SharedFolder")
	store.CreateFolder(ctx, folder)

	// Add files to folder
	file1 := newTestFile("file1", "owner", "doc1.pdf")
	file1.ParentID = sql.NullString{String: "folder1", Valid: true}
	file2 := newTestFile("file2", "owner", "doc2.pdf")
	file2.ParentID = sql.NullString{String: "folder1", Valid: true}
	store.CreateFile(ctx, file1)
	store.CreateFile(ctx, file2)

	// Share the folder (implies access to contents)
	share := newTestShare("share1", "owner", "folder", "folder1")
	share.SharedWithID = sql.NullString{String: "collab", Valid: true}
	share.Permission = "editor"
	store.CreateShare(ctx, share)

	// Verify folder share exists
	folderShare, _ := store.GetShareForUserAndResource(ctx, "collab", "folder", "folder1")
	if folderShare == nil {
		t.Fatal("collaborator should have access to shared folder")
	}
	if folderShare.Permission != "editor" {
		t.Errorf("expected editor permission on folder, got %s", folderShare.Permission)
	}
}
