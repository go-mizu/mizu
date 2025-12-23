package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/forum/feature/accounts"
	"github.com/go-mizu/mizu/blueprints/forum/feature/boards"
)

func createTestAccount(t *testing.T, store *Store, username string) *accounts.Account {
	t.Helper()
	account := &accounts.Account{
		ID:           newTestID(),
		Username:     username,
		Email:        username + "@example.com",
		PasswordHash: "hash",
		CreatedAt:    testTime(),
		UpdatedAt:    testTime(),
	}
	if err := store.Accounts().Create(context.Background(), account); err != nil {
		t.Fatalf("createTestAccount failed: %v", err)
	}
	return account
}

func TestBoardsStore_Create(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	creator := createTestAccount(t, store, "creator")

	board := &boards.Board{
		ID:          newTestID(),
		Name:        "testboard",
		Title:       "Test Board",
		Description: "A test board",
		Sidebar:     "Welcome to test board",
		SidebarHTML: "<p>Welcome to test board</p>",
		IconURL:     "https://example.com/icon.png",
		BannerURL:   "https://example.com/banner.png",
		IsNSFW:      false,
		IsPrivate:   false,
		MemberCount: 0,
		ThreadCount: 0,
		CreatedAt:   testTime(),
		CreatedBy:   creator.ID,
		UpdatedAt:   testTime(),
	}

	if err := store.Boards().Create(ctx, board); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify
	got, err := store.Boards().GetByID(ctx, board.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if got.Name != board.Name {
		t.Errorf("Name: got %q, want %q", got.Name, board.Name)
	}
	if got.Title != board.Title {
		t.Errorf("Title: got %q, want %q", got.Title, board.Title)
	}
	if got.Description != board.Description {
		t.Errorf("Description: got %q, want %q", got.Description, board.Description)
	}
}

func TestBoardsStore_Create_DuplicateName(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	creator := createTestAccount(t, store, "creator")

	board1 := &boards.Board{
		ID:        newTestID(),
		Name:      "samename",
		Title:     "Board 1",
		CreatedAt: testTime(),
		CreatedBy: creator.ID,
		UpdatedAt: testTime(),
	}

	board2 := &boards.Board{
		ID:        newTestID(),
		Name:      "samename",
		Title:     "Board 2",
		CreatedAt: testTime(),
		CreatedBy: creator.ID,
		UpdatedAt: testTime(),
	}

	if err := store.Boards().Create(ctx, board1); err != nil {
		t.Fatalf("Create first board failed: %v", err)
	}

	err := store.Boards().Create(ctx, board2)
	if err == nil {
		t.Error("expected error for duplicate name, got nil")
	}
}

func TestBoardsStore_GetByName(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	creator := createTestAccount(t, store, "creator")

	board := &boards.Board{
		ID:        newTestID(),
		Name:      "TestBoard",
		Title:     "Test Board",
		CreatedAt: testTime(),
		CreatedBy: creator.ID,
		UpdatedAt: testTime(),
	}

	if err := store.Boards().Create(ctx, board); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Test case-insensitive lookup
	got, err := store.Boards().GetByName(ctx, "testboard")
	if err != nil {
		t.Fatalf("GetByName failed: %v", err)
	}

	if got.ID != board.ID {
		t.Errorf("ID: got %q, want %q", got.ID, board.ID)
	}

	// Test with different case
	got2, err := store.Boards().GetByName(ctx, "TESTBOARD")
	if err != nil {
		t.Fatalf("GetByName (uppercase) failed: %v", err)
	}

	if got2.ID != board.ID {
		t.Errorf("ID: got %q, want %q", got2.ID, board.ID)
	}
}

func TestBoardsStore_GetByName_NotFound(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	_, err := store.Boards().GetByName(ctx, "nonexistent")
	if err != boards.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestBoardsStore_Update(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	creator := createTestAccount(t, store, "creator")

	board := &boards.Board{
		ID:          newTestID(),
		Name:        "testboard",
		Title:       "Original Title",
		Description: "Original description",
		CreatedAt:   testTime(),
		CreatedBy:   creator.ID,
		UpdatedAt:   testTime(),
	}

	if err := store.Boards().Create(ctx, board); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Update
	board.Title = "Updated Title"
	board.Description = "Updated description"
	board.MemberCount = 100
	board.UpdatedAt = testTime().Add(time.Hour)

	if err := store.Boards().Update(ctx, board); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify
	got, err := store.Boards().GetByID(ctx, board.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if got.Title != "Updated Title" {
		t.Errorf("Title: got %q, want %q", got.Title, "Updated Title")
	}
	if got.Description != "Updated description" {
		t.Errorf("Description: got %q, want %q", got.Description, "Updated description")
	}
	if got.MemberCount != 100 {
		t.Errorf("MemberCount: got %d, want %d", got.MemberCount, 100)
	}
}

func TestBoardsStore_Delete(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	creator := createTestAccount(t, store, "creator")

	board := &boards.Board{
		ID:        newTestID(),
		Name:      "testboard",
		Title:     "Test Board",
		CreatedAt: testTime(),
		CreatedBy: creator.ID,
		UpdatedAt: testTime(),
	}

	if err := store.Boards().Create(ctx, board); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := store.Boards().Delete(ctx, board.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err := store.Boards().GetByID(ctx, board.ID)
	if err != boards.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestBoardsStore_Members(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	creator := createTestAccount(t, store, "creator")
	member := createTestAccount(t, store, "member")

	board := &boards.Board{
		ID:        newTestID(),
		Name:      "testboard",
		Title:     "Test Board",
		CreatedAt: testTime(),
		CreatedBy: creator.ID,
		UpdatedAt: testTime(),
	}

	if err := store.Boards().Create(ctx, board); err != nil {
		t.Fatalf("Create board failed: %v", err)
	}

	// Add member
	boardMember := &boards.BoardMember{
		BoardID:   board.ID,
		AccountID: member.ID,
		JoinedAt:  testTime(),
	}

	if err := store.Boards().AddMember(ctx, boardMember); err != nil {
		t.Fatalf("AddMember failed: %v", err)
	}

	// Get member
	got, err := store.Boards().GetMember(ctx, board.ID, member.ID)
	if err != nil {
		t.Fatalf("GetMember failed: %v", err)
	}

	if got.AccountID != member.ID {
		t.Errorf("AccountID: got %q, want %q", got.AccountID, member.ID)
	}

	// List members
	members, err := store.Boards().ListMembers(ctx, board.ID, boards.ListOpts{Limit: 10})
	if err != nil {
		t.Fatalf("ListMembers failed: %v", err)
	}

	if len(members) != 1 {
		t.Errorf("Members count: got %d, want 1", len(members))
	}

	// Remove member
	if err := store.Boards().RemoveMember(ctx, board.ID, member.ID); err != nil {
		t.Fatalf("RemoveMember failed: %v", err)
	}

	// Verify removed
	_, err = store.Boards().GetMember(ctx, board.ID, member.ID)
	if err != boards.ErrNotMember {
		t.Errorf("expected ErrNotMember after remove, got %v", err)
	}
}

func TestBoardsStore_Members_Idempotent(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	creator := createTestAccount(t, store, "creator")
	member := createTestAccount(t, store, "member")

	board := &boards.Board{
		ID:        newTestID(),
		Name:      "testboard",
		Title:     "Test Board",
		CreatedAt: testTime(),
		CreatedBy: creator.ID,
		UpdatedAt: testTime(),
	}

	if err := store.Boards().Create(ctx, board); err != nil {
		t.Fatalf("Create board failed: %v", err)
	}

	boardMember := &boards.BoardMember{
		BoardID:   board.ID,
		AccountID: member.ID,
		JoinedAt:  testTime(),
	}

	// Add member twice - should not error
	if err := store.Boards().AddMember(ctx, boardMember); err != nil {
		t.Fatalf("AddMember first failed: %v", err)
	}

	if err := store.Boards().AddMember(ctx, boardMember); err != nil {
		t.Fatalf("AddMember second failed: %v", err)
	}

	// Should still only have one member
	members, err := store.Boards().ListMembers(ctx, board.ID, boards.ListOpts{Limit: 10})
	if err != nil {
		t.Fatalf("ListMembers failed: %v", err)
	}

	if len(members) != 1 {
		t.Errorf("Members count: got %d, want 1", len(members))
	}
}

func TestBoardsStore_Moderators(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	creator := createTestAccount(t, store, "creator")
	mod := createTestAccount(t, store, "mod")

	board := &boards.Board{
		ID:        newTestID(),
		Name:      "testboard",
		Title:     "Test Board",
		CreatedAt: testTime(),
		CreatedBy: creator.ID,
		UpdatedAt: testTime(),
	}

	if err := store.Boards().Create(ctx, board); err != nil {
		t.Fatalf("Create board failed: %v", err)
	}

	// Add moderator
	moderator := &boards.BoardModerator{
		BoardID:   board.ID,
		AccountID: mod.ID,
		Permissions: boards.ModPerms{
			ManagePosts:    true,
			ManageComments: true,
			ManageUsers:    false,
			ManageMods:     false,
			ManageSettings: false,
		},
		AddedAt: testTime(),
		AddedBy: creator.ID,
	}

	if err := store.Boards().AddModerator(ctx, moderator); err != nil {
		t.Fatalf("AddModerator failed: %v", err)
	}

	// Get moderator
	got, err := store.Boards().GetModerator(ctx, board.ID, mod.ID)
	if err != nil {
		t.Fatalf("GetModerator failed: %v", err)
	}

	if got.AccountID != mod.ID {
		t.Errorf("AccountID: got %q, want %q", got.AccountID, mod.ID)
	}
	if !got.Permissions.ManagePosts {
		t.Error("ManagePosts: got false, want true")
	}
	if got.Permissions.ManageUsers {
		t.Error("ManageUsers: got true, want false")
	}

	// List moderators
	mods, err := store.Boards().ListModerators(ctx, board.ID)
	if err != nil {
		t.Fatalf("ListModerators failed: %v", err)
	}

	if len(mods) != 1 {
		t.Errorf("Mods count: got %d, want 1", len(mods))
	}

	// Remove moderator
	if err := store.Boards().RemoveModerator(ctx, board.ID, mod.ID); err != nil {
		t.Fatalf("RemoveModerator failed: %v", err)
	}

	// Verify removed
	_, err = store.Boards().GetModerator(ctx, board.ID, mod.ID)
	if err != boards.ErrNotModerator {
		t.Errorf("expected ErrNotModerator after remove, got %v", err)
	}
}

func TestBoardsStore_ListJoinedBoards(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	creator := createTestAccount(t, store, "creator")
	member := createTestAccount(t, store, "member")

	// Create multiple boards
	for i := 0; i < 3; i++ {
		board := &boards.Board{
			ID:        newTestID(),
			Name:      "board" + string(rune('a'+i)),
			Title:     "Board " + string(rune('A'+i)),
			CreatedAt: testTime(),
			CreatedBy: creator.ID,
			UpdatedAt: testTime(),
		}
		if err := store.Boards().Create(ctx, board); err != nil {
			t.Fatalf("Create board %d failed: %v", i, err)
		}

		// Join first two boards
		if i < 2 {
			boardMember := &boards.BoardMember{
				BoardID:   board.ID,
				AccountID: member.ID,
				JoinedAt:  testTime(),
			}
			if err := store.Boards().AddMember(ctx, boardMember); err != nil {
				t.Fatalf("AddMember %d failed: %v", i, err)
			}
		}
	}

	// List joined boards
	joined, err := store.Boards().ListJoinedBoards(ctx, member.ID)
	if err != nil {
		t.Fatalf("ListJoinedBoards failed: %v", err)
	}

	if len(joined) != 2 {
		t.Errorf("Joined count: got %d, want 2", len(joined))
	}
}

func TestBoardsStore_ListModeratedBoards(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	creator := createTestAccount(t, store, "creator")
	mod := createTestAccount(t, store, "mod")

	// Create multiple boards
	for i := 0; i < 3; i++ {
		board := &boards.Board{
			ID:        newTestID(),
			Name:      "board" + string(rune('a'+i)),
			Title:     "Board " + string(rune('A'+i)),
			CreatedAt: testTime(),
			CreatedBy: creator.ID,
			UpdatedAt: testTime(),
		}
		if err := store.Boards().Create(ctx, board); err != nil {
			t.Fatalf("Create board %d failed: %v", i, err)
		}

		// Moderate first two boards
		if i < 2 {
			moderator := &boards.BoardModerator{
				BoardID:     board.ID,
				AccountID:   mod.ID,
				Permissions: boards.FullPerms(),
				AddedAt:     testTime(),
				AddedBy:     creator.ID,
			}
			if err := store.Boards().AddModerator(ctx, moderator); err != nil {
				t.Fatalf("AddModerator %d failed: %v", i, err)
			}
		}
	}

	// List moderated boards
	moderated, err := store.Boards().ListModeratedBoards(ctx, mod.ID)
	if err != nil {
		t.Fatalf("ListModeratedBoards failed: %v", err)
	}

	if len(moderated) != 2 {
		t.Errorf("Moderated count: got %d, want 2", len(moderated))
	}
}

func TestBoardsStore_List(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	creator := createTestAccount(t, store, "creator")

	// Create boards with different member counts
	for i := 0; i < 5; i++ {
		board := &boards.Board{
			ID:          newTestID(),
			Name:        "board" + string(rune('a'+i)),
			Title:       "Board " + string(rune('A'+i)),
			MemberCount: int64((5 - i) * 100), // Descending member count
			IsPrivate:   i == 4,               // Last one is private
			CreatedAt:   testTime(),
			CreatedBy:   creator.ID,
			UpdatedAt:   testTime(),
		}
		if err := store.Boards().Create(ctx, board); err != nil {
			t.Fatalf("Create board %d failed: %v", i, err)
		}
	}

	// List public boards (should exclude private)
	list, err := store.Boards().List(ctx, boards.ListOpts{Limit: 10})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != 4 {
		t.Errorf("List count: got %d, want 4 (excluding private)", len(list))
	}

	// Should be ordered by member count (descending)
	if list[0].MemberCount < list[len(list)-1].MemberCount {
		t.Error("List should be ordered by member_count DESC")
	}
}

func TestBoardsStore_Search(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	creator := createTestAccount(t, store, "creator")

	// Create boards with searchable names
	testBoards := []struct {
		name  string
		title string
	}{
		{"golang", "Go Programming"},
		{"rustlang", "Rust Programming"},
		{"programming", "General Programming"},
		{"webdev", "Web Development"},
	}

	for _, b := range testBoards {
		board := &boards.Board{
			ID:        newTestID(),
			Name:      b.name,
			Title:     b.title,
			CreatedAt: testTime(),
			CreatedBy: creator.ID,
			UpdatedAt: testTime(),
		}
		if err := store.Boards().Create(ctx, board); err != nil {
			t.Fatalf("Create board %s failed: %v", b.name, err)
		}
	}

	// Search for "programming"
	results, err := store.Boards().Search(ctx, "programming", 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Search 'programming' count: got %d, want 3", len(results))
	}

	// Search for "go"
	results2, err := store.Boards().Search(ctx, "go", 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results2) != 1 {
		t.Errorf("Search 'go' count: got %d, want 1", len(results2))
	}
}

func TestBoardsStore_ListPopular(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	creator := createTestAccount(t, store, "creator")

	// Create boards with different member counts
	for i := 0; i < 5; i++ {
		board := &boards.Board{
			ID:          newTestID(),
			Name:        "board" + string(rune('a'+i)),
			Title:       "Board " + string(rune('A'+i)),
			MemberCount: int64(i * 100),
			CreatedAt:   testTime(),
			CreatedBy:   creator.ID,
			UpdatedAt:   testTime(),
		}
		if err := store.Boards().Create(ctx, board); err != nil {
			t.Fatalf("Create board %d failed: %v", i, err)
		}
	}

	popular, err := store.Boards().ListPopular(ctx, 3)
	if err != nil {
		t.Fatalf("ListPopular failed: %v", err)
	}

	if len(popular) != 3 {
		t.Errorf("Popular count: got %d, want 3", len(popular))
	}

	// First should be most popular
	if popular[0].MemberCount < popular[len(popular)-1].MemberCount {
		t.Error("ListPopular should be ordered by member_count DESC")
	}
}

func TestBoardsStore_ListNew(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	creator := createTestAccount(t, store, "creator")

	// Create boards at different times
	for i := 0; i < 5; i++ {
		board := &boards.Board{
			ID:        newTestID(),
			Name:      "board" + string(rune('a'+i)),
			Title:     "Board " + string(rune('A'+i)),
			CreatedAt: testTime().Add(time.Duration(i) * time.Hour),
			CreatedBy: creator.ID,
			UpdatedAt: testTime().Add(time.Duration(i) * time.Hour),
		}
		if err := store.Boards().Create(ctx, board); err != nil {
			t.Fatalf("Create board %d failed: %v", i, err)
		}
	}

	newBoards, err := store.Boards().ListNew(ctx, 3)
	if err != nil {
		t.Fatalf("ListNew failed: %v", err)
	}

	if len(newBoards) != 3 {
		t.Errorf("New count: got %d, want 3", len(newBoards))
	}

	// First should be newest
	if newBoards[0].CreatedAt.Before(newBoards[len(newBoards)-1].CreatedAt) {
		t.Error("ListNew should be ordered by created_at DESC")
	}
}
