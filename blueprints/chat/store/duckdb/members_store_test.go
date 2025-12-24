package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/chat/feature/members"
)

func TestMembersStore_Insert(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	serversStore := NewServersStore(db)
	store := NewMembersStore(db)
	ctx := context.Background()

	owner := createTestUser(t, usersStore, "owner")
	srv := createTestServer(t, serversStore, owner.ID, "testserver")

	m := &members.Member{
		ServerID: srv.ID,
		UserID:   owner.ID,
		Nickname: "Admin",
		JoinedAt: time.Now(),
	}

	err := store.Insert(ctx, m)
	if err != nil {
		t.Fatalf("Insert() error = %v", err)
	}

	got, err := store.Get(ctx, srv.ID, owner.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got.Nickname != m.Nickname {
		t.Errorf("Nickname = %v, want %v", got.Nickname, m.Nickname)
	}
}

func TestMembersStore_Get(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	serversStore := NewServersStore(db)
	store := NewMembersStore(db)
	ctx := context.Background()

	owner := createTestUser(t, usersStore, "owner")
	srv := createTestServer(t, serversStore, owner.ID, "testserver")
	createTestMember(t, store, srv.ID, owner.ID)

	got, err := store.Get(ctx, srv.ID, owner.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got.UserID != owner.ID {
		t.Errorf("UserID = %v, want %v", got.UserID, owner.ID)
	}

	// Non-existent
	_, err = store.Get(ctx, srv.ID, "nonexistent")
	if err != members.ErrNotFound {
		t.Errorf("Get() error = %v, want ErrNotFound", err)
	}
}

func TestMembersStore_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	serversStore := NewServersStore(db)
	store := NewMembersStore(db)
	ctx := context.Background()

	owner := createTestUser(t, usersStore, "owner")
	srv := createTestServer(t, serversStore, owner.ID, "testserver")
	createTestMember(t, store, srv.ID, owner.ID)

	newNick := "ServerOwner"
	err := store.Update(ctx, srv.ID, owner.ID, &members.UpdateIn{
		Nickname: &newNick,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got, _ := store.Get(ctx, srv.ID, owner.ID)
	if got.Nickname != newNick {
		t.Errorf("Nickname = %v, want %v", got.Nickname, newNick)
	}
}

func TestMembersStore_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	serversStore := NewServersStore(db)
	store := NewMembersStore(db)
	ctx := context.Background()

	owner := createTestUser(t, usersStore, "owner")
	srv := createTestServer(t, serversStore, owner.ID, "testserver")
	createTestMember(t, store, srv.ID, owner.ID)

	err := store.Delete(ctx, srv.ID, owner.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err = store.Get(ctx, srv.ID, owner.ID)
	if err != members.ErrNotFound {
		t.Errorf("Get() after delete error = %v, want ErrNotFound", err)
	}
}

func TestMembersStore_List(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	serversStore := NewServersStore(db)
	store := NewMembersStore(db)
	ctx := context.Background()

	owner := createTestUser(t, usersStore, "owner")
	user2 := createTestUser(t, usersStore, "user2")
	user3 := createTestUser(t, usersStore, "user3")
	srv := createTestServer(t, serversStore, owner.ID, "testserver")

	createTestMember(t, store, srv.ID, owner.ID)
	createTestMember(t, store, srv.ID, user2.ID)
	createTestMember(t, store, srv.ID, user3.ID)

	mems, err := store.List(ctx, srv.ID, 10, 0)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(mems) != 3 {
		t.Errorf("len(mems) = %d, want 3", len(mems))
	}
}

func TestMembersStore_Count(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	serversStore := NewServersStore(db)
	store := NewMembersStore(db)
	ctx := context.Background()

	owner := createTestUser(t, usersStore, "owner")
	user2 := createTestUser(t, usersStore, "user2")
	srv := createTestServer(t, serversStore, owner.ID, "testserver")

	createTestMember(t, store, srv.ID, owner.ID)
	createTestMember(t, store, srv.ID, user2.ID)

	count, err := store.Count(ctx, srv.ID)
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}

	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestMembersStore_IsMember(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	serversStore := NewServersStore(db)
	store := NewMembersStore(db)
	ctx := context.Background()

	owner := createTestUser(t, usersStore, "owner")
	user2 := createTestUser(t, usersStore, "user2")
	srv := createTestServer(t, serversStore, owner.ID, "testserver")

	createTestMember(t, store, srv.ID, owner.ID)

	// Should be member
	isMember, err := store.IsMember(ctx, srv.ID, owner.ID)
	if err != nil {
		t.Fatalf("IsMember() error = %v", err)
	}
	if !isMember {
		t.Error("owner should be member")
	}

	// Should not be member
	isMember, err = store.IsMember(ctx, srv.ID, user2.ID)
	if err != nil {
		t.Fatalf("IsMember() error = %v", err)
	}
	if isMember {
		t.Error("user2 should not be member")
	}
}

func TestMembersStore_Roles(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	serversStore := NewServersStore(db)
	membersStore := NewMembersStore(db)
	rolesStore := NewRolesStore(db)
	ctx := context.Background()

	owner := createTestUser(t, usersStore, "owner")
	srv := createTestServer(t, serversStore, owner.ID, "testserver")
	createTestMember(t, membersStore, srv.ID, owner.ID)
	role := createTestRole(t, rolesStore, srv.ID, "Admin")

	// Add role
	err := membersStore.AddRole(ctx, srv.ID, owner.ID, role.ID)
	if err != nil {
		t.Fatalf("AddRole() error = %v", err)
	}

	// Get member with roles
	got, _ := membersStore.Get(ctx, srv.ID, owner.ID)
	if len(got.RoleIDs) != 1 {
		t.Errorf("len(RoleIDs) = %d, want 1", len(got.RoleIDs))
	}

	// Remove role
	err = membersStore.RemoveRole(ctx, srv.ID, owner.ID, role.ID)
	if err != nil {
		t.Fatalf("RemoveRole() error = %v", err)
	}

	got, _ = membersStore.Get(ctx, srv.ID, owner.ID)
	if len(got.RoleIDs) != 0 {
		t.Errorf("len(RoleIDs) after remove = %d, want 0", len(got.RoleIDs))
	}
}

func TestMembersStore_ListByRole(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	serversStore := NewServersStore(db)
	membersStore := NewMembersStore(db)
	rolesStore := NewRolesStore(db)
	ctx := context.Background()

	owner := createTestUser(t, usersStore, "owner")
	user2 := createTestUser(t, usersStore, "user2")
	user3 := createTestUser(t, usersStore, "user3")
	srv := createTestServer(t, serversStore, owner.ID, "testserver")

	createTestMember(t, membersStore, srv.ID, owner.ID)
	createTestMember(t, membersStore, srv.ID, user2.ID)
	createTestMember(t, membersStore, srv.ID, user3.ID)

	role := createTestRole(t, rolesStore, srv.ID, "Moderator")

	// Add role to two members
	membersStore.AddRole(ctx, srv.ID, owner.ID, role.ID)
	membersStore.AddRole(ctx, srv.ID, user2.ID, role.ID)

	mems, err := membersStore.ListByRole(ctx, srv.ID, role.ID)
	if err != nil {
		t.Fatalf("ListByRole() error = %v", err)
	}

	if len(mems) != 2 {
		t.Errorf("len(mems) = %d, want 2", len(mems))
	}
}

func TestMembersStore_Search(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	serversStore := NewServersStore(db)
	membersStore := NewMembersStore(db)
	ctx := context.Background()

	alice := createTestUser(t, usersStore, "alice")
	bob := createTestUser(t, usersStore, "bob")
	srv := createTestServer(t, serversStore, alice.ID, "testserver")

	// Create members with nicknames
	m1 := &members.Member{ServerID: srv.ID, UserID: alice.ID, Nickname: "SuperAlice", JoinedAt: time.Now()}
	m2 := &members.Member{ServerID: srv.ID, UserID: bob.ID, Nickname: "RegularBob", JoinedAt: time.Now()}
	membersStore.Insert(ctx, m1)
	membersStore.Insert(ctx, m2)

	// Search by username
	mems, err := membersStore.Search(ctx, srv.ID, "alice", 10)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(mems) != 1 {
		t.Errorf("len(mems) = %d, want 1", len(mems))
	}

	// Search by nickname
	mems, err = membersStore.Search(ctx, srv.ID, "Super", 10)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(mems) != 1 {
		t.Errorf("len(mems) = %d, want 1", len(mems))
	}
}

func TestMembersStore_Ban(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	serversStore := NewServersStore(db)
	membersStore := NewMembersStore(db)
	ctx := context.Background()

	owner := createTestUser(t, usersStore, "owner")
	badUser := createTestUser(t, usersStore, "baduser")
	srv := createTestServer(t, serversStore, owner.ID, "testserver")

	createTestMember(t, membersStore, srv.ID, owner.ID)
	createTestMember(t, membersStore, srv.ID, badUser.ID)

	// Ban user
	err := membersStore.Ban(ctx, srv.ID, badUser.ID, owner.ID, "Breaking rules")
	if err != nil {
		t.Fatalf("Ban() error = %v", err)
	}

	// User should no longer be member
	_, err = membersStore.Get(ctx, srv.ID, badUser.ID)
	if err != members.ErrNotFound {
		t.Error("banned user should be removed from members")
	}

	// User should be banned
	isBanned, err := membersStore.IsBanned(ctx, srv.ID, badUser.ID)
	if err != nil {
		t.Fatalf("IsBanned() error = %v", err)
	}
	if !isBanned {
		t.Error("user should be banned")
	}

	// List bans
	bans, err := membersStore.ListBans(ctx, srv.ID, 10, 0)
	if err != nil {
		t.Fatalf("ListBans() error = %v", err)
	}

	if len(bans) != 1 {
		t.Errorf("len(bans) = %d, want 1", len(bans))
	}

	if bans[0].Reason != "Breaking rules" {
		t.Errorf("Reason = %v, want 'Breaking rules'", bans[0].Reason)
	}

	// Unban
	err = membersStore.Unban(ctx, srv.ID, badUser.ID)
	if err != nil {
		t.Fatalf("Unban() error = %v", err)
	}

	isBanned, _ = membersStore.IsBanned(ctx, srv.ID, badUser.ID)
	if isBanned {
		t.Error("user should not be banned after unban")
	}
}
