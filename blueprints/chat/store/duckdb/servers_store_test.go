package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/chat/feature/servers"
)

func TestServersStore_Insert(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	store := NewServersStore(db)
	ctx := context.Background()

	// Create owner
	owner := createTestUser(t, usersStore, "owner")

	srv := &servers.Server{
		ID:          "srv1",
		Name:        "Test Server",
		Description: "A test server",
		OwnerID:     owner.ID,
		IsPublic:    true,
		InviteCode:  "abc123",
		MemberCount: 1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := store.Insert(ctx, srv)
	if err != nil {
		t.Fatalf("Insert() error = %v", err)
	}

	got, err := store.GetByID(ctx, "srv1")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if got.Name != srv.Name {
		t.Errorf("Name = %v, want %v", got.Name, srv.Name)
	}
}

func TestServersStore_GetByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	store := NewServersStore(db)
	ctx := context.Background()

	owner := createTestUser(t, usersStore, "owner")
	srv := createTestServer(t, store, owner.ID, "testserver")

	got, err := store.GetByID(ctx, srv.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if got.ID != srv.ID {
		t.Errorf("ID = %v, want %v", got.ID, srv.ID)
	}

	// Non-existent
	_, err = store.GetByID(ctx, "nonexistent")
	if err != servers.ErrNotFound {
		t.Errorf("GetByID() error = %v, want ErrNotFound", err)
	}
}

func TestServersStore_GetByInviteCode(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	store := NewServersStore(db)
	ctx := context.Background()

	owner := createTestUser(t, usersStore, "owner")
	srv := createTestServer(t, store, owner.ID, "testserver")

	got, err := store.GetByInviteCode(ctx, "invite-testserver")
	if err != nil {
		t.Fatalf("GetByInviteCode() error = %v", err)
	}

	if got.ID != srv.ID {
		t.Errorf("ID = %v, want %v", got.ID, srv.ID)
	}

	// Non-existent
	_, err = store.GetByInviteCode(ctx, "invalid-code")
	if err != servers.ErrNotFound {
		t.Errorf("GetByInviteCode() error = %v, want ErrNotFound", err)
	}
}

func TestServersStore_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	store := NewServersStore(db)
	ctx := context.Background()

	owner := createTestUser(t, usersStore, "owner")
	srv := createTestServer(t, store, owner.ID, "testserver")

	// Update name
	newName := "Updated Server"
	err := store.Update(ctx, srv.ID, &servers.UpdateIn{
		Name: &newName,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got, _ := store.GetByID(ctx, srv.ID)
	if got.Name != newName {
		t.Errorf("Name = %v, want %v", got.Name, newName)
	}

	// Empty update should be no-op
	err = store.Update(ctx, srv.ID, &servers.UpdateIn{})
	if err != nil {
		t.Errorf("Update() with empty input error = %v", err)
	}
}

func TestServersStore_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	store := NewServersStore(db)
	ctx := context.Background()

	owner := createTestUser(t, usersStore, "owner")
	srv := createTestServer(t, store, owner.ID, "testserver")

	err := store.Delete(ctx, srv.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err = store.GetByID(ctx, srv.ID)
	if err != servers.ErrNotFound {
		t.Errorf("GetByID() after delete error = %v, want ErrNotFound", err)
	}
}

func TestServersStore_ListByUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	serversStore := NewServersStore(db)
	membersStore := NewMembersStore(db)
	ctx := context.Background()

	owner := createTestUser(t, usersStore, "owner")
	srv1 := createTestServer(t, serversStore, owner.ID, "server1")
	srv2 := createTestServer(t, serversStore, owner.ID, "server2")

	// Add owner as member
	createTestMember(t, membersStore, srv1.ID, owner.ID)
	createTestMember(t, membersStore, srv2.ID, owner.ID)

	srvs, err := serversStore.ListByUser(ctx, owner.ID, 10, 0)
	if err != nil {
		t.Fatalf("ListByUser() error = %v", err)
	}

	if len(srvs) != 2 {
		t.Errorf("len(srvs) = %d, want 2", len(srvs))
	}
}

func TestServersStore_ListPublic(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	store := NewServersStore(db)
	ctx := context.Background()

	owner := createTestUser(t, usersStore, "owner")
	createTestServer(t, store, owner.ID, "public1")
	createTestServer(t, store, owner.ID, "public2")

	// Create a private server
	privateSrv := &servers.Server{
		ID:        "private-srv",
		Name:      "Private Server",
		OwnerID:   owner.ID,
		IsPublic:  false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	store.Insert(ctx, privateSrv)

	srvs, err := store.ListPublic(ctx, 10, 0)
	if err != nil {
		t.Fatalf("ListPublic() error = %v", err)
	}

	// Should only have public servers
	if len(srvs) != 2 {
		t.Errorf("len(srvs) = %d, want 2", len(srvs))
	}

	for _, srv := range srvs {
		if !srv.IsPublic {
			t.Errorf("got private server in ListPublic: %s", srv.Name)
		}
	}
}

func TestServersStore_UpdateMemberCount(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	store := NewServersStore(db)
	ctx := context.Background()

	owner := createTestUser(t, usersStore, "owner")
	srv := createTestServer(t, store, owner.ID, "testserver")

	initialCount := srv.MemberCount

	// Increment
	err := store.UpdateMemberCount(ctx, srv.ID, 5)
	if err != nil {
		t.Fatalf("UpdateMemberCount() error = %v", err)
	}

	got, _ := store.GetByID(ctx, srv.ID)
	if got.MemberCount != initialCount+5 {
		t.Errorf("MemberCount = %d, want %d", got.MemberCount, initialCount+5)
	}

	// Decrement
	err = store.UpdateMemberCount(ctx, srv.ID, -2)
	if err != nil {
		t.Fatalf("UpdateMemberCount() error = %v", err)
	}

	got, _ = store.GetByID(ctx, srv.ID)
	if got.MemberCount != initialCount+3 {
		t.Errorf("MemberCount = %d, want %d", got.MemberCount, initialCount+3)
	}
}

func TestServersStore_SetDefaultChannel(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	serversStore := NewServersStore(db)
	channelsStore := NewChannelsStore(db)
	ctx := context.Background()

	owner := createTestUser(t, usersStore, "owner")
	srv := createTestServer(t, serversStore, owner.ID, "testserver")
	ch := createTestChannel(t, channelsStore, srv.ID, "general")

	err := serversStore.SetDefaultChannel(ctx, srv.ID, ch.ID)
	if err != nil {
		t.Fatalf("SetDefaultChannel() error = %v", err)
	}

	got, _ := serversStore.GetByID(ctx, srv.ID)
	if got.DefaultChannel != ch.ID {
		t.Errorf("DefaultChannel = %v, want %v", got.DefaultChannel, ch.ID)
	}
}

func TestServersStore_Search(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	store := NewServersStore(db)
	ctx := context.Background()

	owner := createTestUser(t, usersStore, "owner")
	createTestServer(t, store, owner.ID, "gaming")
	createTestServer(t, store, owner.ID, "coding")
	createTestServer(t, store, owner.ID, "music")

	// Search for 'gam'
	srvs, err := store.Search(ctx, "gam", 10)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(srvs) != 1 {
		t.Errorf("len(srvs) = %d, want 1", len(srvs))
	}

	if len(srvs) > 0 && srvs[0].Name != "gaming" {
		t.Errorf("Name = %v, want gaming", srvs[0].Name)
	}
}
