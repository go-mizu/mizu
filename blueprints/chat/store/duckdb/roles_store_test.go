package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/chat/feature/roles"
)

func TestRolesStore_Insert(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	serversStore := NewServersStore(db)
	store := NewRolesStore(db)
	ctx := context.Background()

	owner := createTestUser(t, usersStore, "owner")
	srv := createTestServer(t, serversStore, owner.ID, "testserver")

	role := &roles.Role{
		ID:          "role1",
		ServerID:    srv.ID,
		Name:        "Admin",
		Color:       0xFF0000,
		Position:    10,
		Permissions: roles.PermissionAdministrator,
		CreatedAt:   time.Now(),
	}

	err := store.Insert(ctx, role)
	if err != nil {
		t.Fatalf("Insert() error = %v", err)
	}

	got, err := store.GetByID(ctx, "role1")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if got.Name != role.Name {
		t.Errorf("Name = %v, want %v", got.Name, role.Name)
	}
}

func TestRolesStore_GetByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	serversStore := NewServersStore(db)
	store := NewRolesStore(db)
	ctx := context.Background()

	owner := createTestUser(t, usersStore, "owner")
	srv := createTestServer(t, serversStore, owner.ID, "testserver")
	role := createTestRole(t, store, srv.ID, "Admin")

	got, err := store.GetByID(ctx, role.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if got.ID != role.ID {
		t.Errorf("ID = %v, want %v", got.ID, role.ID)
	}

	// Non-existent
	_, err = store.GetByID(ctx, "nonexistent")
	if err != roles.ErrNotFound {
		t.Errorf("GetByID() error = %v, want ErrNotFound", err)
	}
}

func TestRolesStore_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	serversStore := NewServersStore(db)
	store := NewRolesStore(db)
	ctx := context.Background()

	owner := createTestUser(t, usersStore, "owner")
	srv := createTestServer(t, serversStore, owner.ID, "testserver")
	role := createTestRole(t, store, srv.ID, "Admin")

	newName := "Super Admin"
	newColor := 0x00FF00
	err := store.Update(ctx, role.ID, &roles.UpdateIn{
		Name:  &newName,
		Color: &newColor,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got, _ := store.GetByID(ctx, role.ID)
	if got.Name != newName {
		t.Errorf("Name = %v, want %v", got.Name, newName)
	}
	if got.Color != newColor {
		t.Errorf("Color = %v, want %v", got.Color, newColor)
	}
}

func TestRolesStore_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	serversStore := NewServersStore(db)
	store := NewRolesStore(db)
	ctx := context.Background()

	owner := createTestUser(t, usersStore, "owner")
	srv := createTestServer(t, serversStore, owner.ID, "testserver")
	role := createTestRole(t, store, srv.ID, "Admin")

	err := store.Delete(ctx, role.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err = store.GetByID(ctx, role.ID)
	if err != roles.ErrNotFound {
		t.Errorf("GetByID() after delete error = %v, want ErrNotFound", err)
	}
}

func TestRolesStore_ListByServer(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	serversStore := NewServersStore(db)
	store := NewRolesStore(db)
	ctx := context.Background()

	owner := createTestUser(t, usersStore, "owner")
	srv := createTestServer(t, serversStore, owner.ID, "testserver")
	createTestRole(t, store, srv.ID, "Admin")
	createTestRole(t, store, srv.ID, "Moderator")
	createTestRole(t, store, srv.ID, "Member")

	rs, err := store.ListByServer(ctx, srv.ID)
	if err != nil {
		t.Fatalf("ListByServer() error = %v", err)
	}

	if len(rs) != 3 {
		t.Errorf("len(rs) = %d, want 3", len(rs))
	}
}

func TestRolesStore_GetByIDs(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	serversStore := NewServersStore(db)
	store := NewRolesStore(db)
	ctx := context.Background()

	owner := createTestUser(t, usersStore, "owner")
	srv := createTestServer(t, serversStore, owner.ID, "testserver")
	role1 := createTestRole(t, store, srv.ID, "Admin")
	role2 := createTestRole(t, store, srv.ID, "Mod")

	rs, err := store.GetByIDs(ctx, []string{role1.ID, role2.ID})
	if err != nil {
		t.Fatalf("GetByIDs() error = %v", err)
	}

	if len(rs) != 2 {
		t.Errorf("len(rs) = %d, want 2", len(rs))
	}

	// Empty slice
	rs, err = store.GetByIDs(ctx, []string{})
	if err != nil {
		t.Fatalf("GetByIDs() with empty slice error = %v", err)
	}
	if rs != nil {
		t.Errorf("expected nil for empty slice, got %v", rs)
	}
}

func TestRolesStore_CreateDefaultRole(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	serversStore := NewServersStore(db)
	store := NewRolesStore(db)
	ctx := context.Background()

	owner := createTestUser(t, usersStore, "owner")
	srv := createTestServer(t, serversStore, owner.ID, "testserver")

	role, err := store.CreateDefaultRole(ctx, srv.ID)
	if err != nil {
		t.Fatalf("CreateDefaultRole() error = %v", err)
	}

	if role.Name != "@everyone" {
		t.Errorf("Name = %v, want @everyone", role.Name)
	}
	if !role.IsDefault {
		t.Error("IsDefault should be true")
	}
	if role.ID != srv.ID {
		t.Errorf("ID = %v, want %v (same as server ID)", role.ID, srv.ID)
	}

	// Get default role
	got, err := store.GetDefaultRole(ctx, srv.ID)
	if err != nil {
		t.Fatalf("GetDefaultRole() error = %v", err)
	}

	if got.ID != role.ID {
		t.Errorf("ID = %v, want %v", got.ID, role.ID)
	}
}

func TestRolesStore_UpdatePositions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	serversStore := NewServersStore(db)
	store := NewRolesStore(db)
	ctx := context.Background()

	owner := createTestUser(t, usersStore, "owner")
	srv := createTestServer(t, serversStore, owner.ID, "testserver")
	role1 := createTestRole(t, store, srv.ID, "Admin")
	role2 := createTestRole(t, store, srv.ID, "Mod")

	positions := map[string]int{
		role1.ID: 5,
		role2.ID: 10,
	}

	err := store.UpdatePositions(ctx, srv.ID, positions)
	if err != nil {
		t.Fatalf("UpdatePositions() error = %v", err)
	}

	got1, _ := store.GetByID(ctx, role1.ID)
	got2, _ := store.GetByID(ctx, role2.ID)

	if got1.Position != 5 {
		t.Errorf("role1.Position = %d, want 5", got1.Position)
	}
	if got2.Position != 10 {
		t.Errorf("role2.Position = %d, want 10", got2.Position)
	}
}

func TestRolesStore_ChannelPermissions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	usersStore := NewUsersStore(db)
	serversStore := NewServersStore(db)
	channelsStore := NewChannelsStore(db)
	store := NewRolesStore(db)
	ctx := context.Background()

	owner := createTestUser(t, usersStore, "owner")
	srv := createTestServer(t, serversStore, owner.ID, "testserver")
	ch := createTestChannel(t, channelsStore, srv.ID, "general")
	role := createTestRole(t, store, srv.ID, "Admin")

	// Insert channel permission
	cp := &roles.ChannelPermission{
		ChannelID:  ch.ID,
		TargetID:   role.ID,
		TargetType: "role",
		Allow:      roles.PermissionSendMessages,
		Deny:       roles.PermissionManageMessages,
	}

	err := store.InsertChannelPermission(ctx, cp)
	if err != nil {
		t.Fatalf("InsertChannelPermission() error = %v", err)
	}

	// Get channel permissions
	perms, err := store.GetChannelPermissions(ctx, ch.ID)
	if err != nil {
		t.Fatalf("GetChannelPermissions() error = %v", err)
	}

	if len(perms) != 1 {
		t.Errorf("len(perms) = %d, want 1", len(perms))
	}

	if perms[0].Allow != roles.PermissionSendMessages {
		t.Errorf("Allow = %d, want %d", perms[0].Allow, roles.PermissionSendMessages)
	}

	// Update (upsert)
	cp.Allow = roles.PermissionViewChannel
	err = store.InsertChannelPermission(ctx, cp)
	if err != nil {
		t.Fatalf("InsertChannelPermission() upsert error = %v", err)
	}

	perms, _ = store.GetChannelPermissions(ctx, ch.ID)
	if perms[0].Allow != roles.PermissionViewChannel {
		t.Errorf("Allow after upsert = %d, want %d", perms[0].Allow, roles.PermissionViewChannel)
	}

	// Delete
	err = store.DeleteChannelPermission(ctx, ch.ID, role.ID)
	if err != nil {
		t.Fatalf("DeleteChannelPermission() error = %v", err)
	}

	perms, _ = store.GetChannelPermissions(ctx, ch.ID)
	if len(perms) != 0 {
		t.Errorf("len(perms) after delete = %d, want 0", len(perms))
	}
}
