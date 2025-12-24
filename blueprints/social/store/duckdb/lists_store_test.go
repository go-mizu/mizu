package duckdb

import (
	"context"
	"database/sql"
	"testing"

	"github.com/go-mizu/blueprints/social/feature/lists"
)

func TestListsStore_Insert(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	store := NewListsStore(db)

	list := &lists.List{
		ID:        newTestID(),
		AccountID: account.ID,
		Title:     "My List",
		CreatedAt: testTime(),
		UpdatedAt: testTime(),
	}

	if err := store.Insert(ctx, list); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Verify
	got, err := store.GetByID(ctx, list.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if got.Title != list.Title {
		t.Errorf("Title: got %q, want %q", got.Title, list.Title)
	}
	if got.AccountID != list.AccountID {
		t.Errorf("AccountID: got %q, want %q", got.AccountID, list.AccountID)
	}
}

func TestListsStore_GetByID(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	store := NewListsStore(db)

	list := &lists.List{
		ID:        newTestID(),
		AccountID: account.ID,
		Title:     "Test List",
		CreatedAt: testTime(),
		UpdatedAt: testTime(),
	}
	store.Insert(ctx, list)

	got, err := store.GetByID(ctx, list.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if got.ID != list.ID {
		t.Errorf("ID: got %q, want %q", got.ID, list.ID)
	}
}

func TestListsStore_GetByID_NotFound(t *testing.T) {
	db := setupTestStore(t)
	store := NewListsStore(db)
	ctx := context.Background()

	_, err := store.GetByID(ctx, "nonexistent")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestListsStore_GetByAccount(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	store := NewListsStore(db)

	// Create multiple lists
	for _, title := range []string{"List A", "List B", "List C"} {
		store.Insert(ctx, &lists.List{
			ID:        newTestID(),
			AccountID: account.ID,
			Title:     title,
			CreatedAt: testTime(),
			UpdatedAt: testTime(),
		})
	}

	got, err := store.GetByAccount(ctx, account.ID)
	if err != nil {
		t.Fatalf("GetByAccount failed: %v", err)
	}

	if len(got) != 3 {
		t.Errorf("GetByAccount count: got %d, want 3", len(got))
	}

	// Should be sorted by title
	if got[0].Title != "List A" {
		t.Errorf("First list title: got %q, want %q", got[0].Title, "List A")
	}
}

func TestListsStore_Update(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	store := NewListsStore(db)

	list := &lists.List{
		ID:        newTestID(),
		AccountID: account.ID,
		Title:     "Original Title",
		CreatedAt: testTime(),
		UpdatedAt: testTime(),
	}
	store.Insert(ctx, list)

	// Update
	newTitle := "Updated Title"
	if err := store.Update(ctx, list.ID, &lists.UpdateIn{Title: &newTitle}); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify
	got, _ := store.GetByID(ctx, list.ID)
	if got.Title != "Updated Title" {
		t.Errorf("Title: got %q, want %q", got.Title, "Updated Title")
	}
}

func TestListsStore_Delete(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	account := createTestAccount(t, db, "testuser")
	store := NewListsStore(db)

	list := &lists.List{
		ID:        newTestID(),
		AccountID: account.ID,
		Title:     "To Delete",
		CreatedAt: testTime(),
		UpdatedAt: testTime(),
	}
	store.Insert(ctx, list)

	// Add a member
	member := createTestAccount(t, db, "member")
	store.InsertMember(ctx, &lists.ListMember{
		ListID:    list.ID,
		AccountID: member.ID,
		CreatedAt: testTime(),
	})

	// Delete (should cascade to members)
	if err := store.Delete(ctx, list.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify list deleted
	_, err := store.GetByID(ctx, list.ID)
	if err != sql.ErrNoRows {
		t.Error("List should be deleted")
	}

	// Verify member removed
	exists, _ := store.ExistsMember(ctx, list.ID, member.ID)
	if exists {
		t.Error("Member should be deleted with list")
	}
}

func TestListsStore_InsertMember(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	owner := createTestAccount(t, db, "owner")
	member := createTestAccount(t, db, "member")
	store := NewListsStore(db)

	list := &lists.List{
		ID:        newTestID(),
		AccountID: owner.ID,
		Title:     "Test List",
		CreatedAt: testTime(),
		UpdatedAt: testTime(),
	}
	store.Insert(ctx, list)

	// Add member
	listMember := &lists.ListMember{
		ListID:    list.ID,
		AccountID: member.ID,
		CreatedAt: testTime(),
	}
	if err := store.InsertMember(ctx, listMember); err != nil {
		t.Fatalf("InsertMember failed: %v", err)
	}

	// Verify
	exists, err := store.ExistsMember(ctx, list.ID, member.ID)
	if err != nil {
		t.Fatalf("ExistsMember failed: %v", err)
	}
	if !exists {
		t.Error("ExistsMember: got false, want true")
	}
}

func TestListsStore_DeleteMember(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	owner := createTestAccount(t, db, "owner")
	member := createTestAccount(t, db, "member")
	store := NewListsStore(db)

	list := &lists.List{
		ID:        newTestID(),
		AccountID: owner.ID,
		Title:     "Test List",
		CreatedAt: testTime(),
		UpdatedAt: testTime(),
	}
	store.Insert(ctx, list)

	// Add member
	store.InsertMember(ctx, &lists.ListMember{
		ListID:    list.ID,
		AccountID: member.ID,
		CreatedAt: testTime(),
	})

	// Delete member
	if err := store.DeleteMember(ctx, list.ID, member.ID); err != nil {
		t.Fatalf("DeleteMember failed: %v", err)
	}

	// Verify
	exists, _ := store.ExistsMember(ctx, list.ID, member.ID)
	if exists {
		t.Error("ExistsMember after delete: got true, want false")
	}
}

func TestListsStore_GetMembers(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	owner := createTestAccount(t, db, "owner")
	member1 := createTestAccount(t, db, "member1")
	member2 := createTestAccount(t, db, "member2")
	member3 := createTestAccount(t, db, "member3")
	store := NewListsStore(db)

	list := &lists.List{
		ID:        newTestID(),
		AccountID: owner.ID,
		Title:     "Test List",
		CreatedAt: testTime(),
		UpdatedAt: testTime(),
	}
	store.Insert(ctx, list)

	// Add members
	for _, memberID := range []string{member1.ID, member2.ID, member3.ID} {
		store.InsertMember(ctx, &lists.ListMember{
			ListID:    list.ID,
			AccountID: memberID,
			CreatedAt: testTime(),
		})
	}

	// Get members
	members, err := store.GetMembers(ctx, list.ID, 10, 0)
	if err != nil {
		t.Fatalf("GetMembers failed: %v", err)
	}

	if len(members) != 3 {
		t.Errorf("GetMembers count: got %d, want 3", len(members))
	}
}

func TestListsStore_GetMemberCount(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	owner := createTestAccount(t, db, "owner")
	member1 := createTestAccount(t, db, "member1")
	member2 := createTestAccount(t, db, "member2")
	store := NewListsStore(db)

	list := &lists.List{
		ID:        newTestID(),
		AccountID: owner.ID,
		Title:     "Test List",
		CreatedAt: testTime(),
		UpdatedAt: testTime(),
	}
	store.Insert(ctx, list)

	// Add members
	store.InsertMember(ctx, &lists.ListMember{
		ListID:    list.ID,
		AccountID: member1.ID,
		CreatedAt: testTime(),
	})
	store.InsertMember(ctx, &lists.ListMember{
		ListID:    list.ID,
		AccountID: member2.ID,
		CreatedAt: testTime(),
	})

	// Get count
	count, err := store.GetMemberCount(ctx, list.ID)
	if err != nil {
		t.Fatalf("GetMemberCount failed: %v", err)
	}

	if count != 2 {
		t.Errorf("GetMemberCount: got %d, want 2", count)
	}
}

func TestListsStore_GetListsContaining(t *testing.T) {
	db := setupTestStore(t)
	ctx := context.Background()

	owner := createTestAccount(t, db, "owner")
	target := createTestAccount(t, db, "target")
	store := NewListsStore(db)

	// Create lists
	list1 := &lists.List{
		ID:        newTestID(),
		AccountID: owner.ID,
		Title:     "List 1",
		CreatedAt: testTime(),
		UpdatedAt: testTime(),
	}
	store.Insert(ctx, list1)

	list2 := &lists.List{
		ID:        newTestID(),
		AccountID: owner.ID,
		Title:     "List 2",
		CreatedAt: testTime(),
		UpdatedAt: testTime(),
	}
	store.Insert(ctx, list2)

	list3 := &lists.List{
		ID:        newTestID(),
		AccountID: owner.ID,
		Title:     "List 3",
		CreatedAt: testTime(),
		UpdatedAt: testTime(),
	}
	store.Insert(ctx, list3)

	// Add target to list1 and list2, not list3
	store.InsertMember(ctx, &lists.ListMember{
		ListID:    list1.ID,
		AccountID: target.ID,
		CreatedAt: testTime(),
	})
	store.InsertMember(ctx, &lists.ListMember{
		ListID:    list2.ID,
		AccountID: target.ID,
		CreatedAt: testTime(),
	})

	// Get lists containing target
	containing, err := store.GetListsContaining(ctx, target.ID)
	if err != nil {
		t.Fatalf("GetListsContaining failed: %v", err)
	}

	if len(containing) != 2 {
		t.Errorf("GetListsContaining count: got %d, want 2", len(containing))
	}
}
