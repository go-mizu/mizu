package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/kanban/feature/users"
	"github.com/go-mizu/blueprints/kanban/feature/workspaces"
	"github.com/oklog/ulid/v2"
)

func createTestWorkspace(t *testing.T, store *WorkspacesStore) *workspaces.Workspace {
	t.Helper()
	w := &workspaces.Workspace{
		ID:   ulid.Make().String(),
		Slug: "ws-" + ulid.Make().String()[:8],
		Name: "Test Workspace",
	}
	if err := store.Create(context.Background(), w); err != nil {
		t.Fatalf("failed to create test workspace: %v", err)
	}
	return w
}

func TestWorkspacesStore_Create(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	w := &workspaces.Workspace{
		ID:   ulid.Make().String(),
		Slug: "test-workspace",
		Name: "Test Workspace",
	}

	err := wsStore.Create(context.Background(), w)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := wsStore.GetByID(context.Background(), w.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected workspace to be created")
	}
	if got.Slug != w.Slug {
		t.Errorf("got slug %q, want %q", got.Slug, w.Slug)
	}
}

func TestWorkspacesStore_Create_DuplicateSlug(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	w1 := &workspaces.Workspace{
		ID:   ulid.Make().String(),
		Slug: "dup-slug",
		Name: "Workspace 1",
	}
	w2 := &workspaces.Workspace{
		ID:   ulid.Make().String(),
		Slug: "dup-slug", // same slug
		Name: "Workspace 2",
	}

	if err := wsStore.Create(context.Background(), w1); err != nil {
		t.Fatalf("first Create failed: %v", err)
	}

	err := wsStore.Create(context.Background(), w2)
	if err == nil {
		t.Error("expected error for duplicate slug")
	}
}

func TestWorkspacesStore_GetByID(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	w := createTestWorkspace(t, wsStore)

	got, err := wsStore.GetByID(context.Background(), w.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected workspace")
	}
	if got.ID != w.ID {
		t.Errorf("got ID %q, want %q", got.ID, w.ID)
	}
}

func TestWorkspacesStore_GetByID_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())

	got, err := wsStore.GetByID(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent workspace")
	}
}

func TestWorkspacesStore_GetBySlug(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	w := createTestWorkspace(t, wsStore)

	got, err := wsStore.GetBySlug(context.Background(), w.Slug)
	if err != nil {
		t.Fatalf("GetBySlug failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected workspace")
	}
	if got.Slug != w.Slug {
		t.Errorf("got slug %q, want %q", got.Slug, w.Slug)
	}
}

func TestWorkspacesStore_GetBySlug_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())

	got, err := wsStore.GetBySlug(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("GetBySlug failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent slug")
	}
}

func TestWorkspacesStore_ListByUser(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	usersStore := NewUsersStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "listuser@example.com",
		Username:     "listuser",
		DisplayName:  "List User",
		PasswordHash: "hashed",
	}
	usersStore.Create(context.Background(), u)

	w := createTestWorkspace(t, wsStore)
	wsStore.AddMember(context.Background(), &workspaces.Member{
		WorkspaceID: w.ID,
		UserID:      u.ID,
		Role:        "member",
		JoinedAt:    time.Now(),
	})

	list, err := wsStore.ListByUser(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("ListByUser failed: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("got %d workspaces, want 1", len(list))
	}
}

func TestWorkspacesStore_ListByUser_Empty(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())

	list, err := wsStore.ListByUser(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("ListByUser failed: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("got %d workspaces, want 0", len(list))
	}
}

func TestWorkspacesStore_Update(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	w := createTestWorkspace(t, wsStore)

	newName := "Updated Workspace"
	err := wsStore.Update(context.Background(), w.ID, &workspaces.UpdateIn{
		Name: &newName,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, _ := wsStore.GetByID(context.Background(), w.ID)
	if got.Name != newName {
		t.Errorf("got name %q, want %q", got.Name, newName)
	}
}

func TestWorkspacesStore_Delete(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	w := createTestWorkspace(t, wsStore)

	err := wsStore.Delete(context.Background(), w.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	got, _ := wsStore.GetByID(context.Background(), w.ID)
	if got != nil {
		t.Error("expected workspace to be deleted")
	}
}

func TestWorkspacesStore_AddMember(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	usersStore := NewUsersStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "member@example.com",
		Username:     "memberuser",
		DisplayName:  "Member User",
		PasswordHash: "hashed",
	}
	usersStore.Create(context.Background(), u)

	w := createTestWorkspace(t, wsStore)

	m := &workspaces.Member{
		WorkspaceID: w.ID,
		UserID:      u.ID,
		Role:        "member",
		JoinedAt:    time.Now(),
	}

	err := wsStore.AddMember(context.Background(), m)
	if err != nil {
		t.Fatalf("AddMember failed: %v", err)
	}

	got, err := wsStore.GetMember(context.Background(), w.ID, u.ID)
	if err != nil {
		t.Fatalf("GetMember failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected member")
	}
	if got.Role != "member" {
		t.Errorf("got role %q, want %q", got.Role, "member")
	}
}

func TestWorkspacesStore_GetMember_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	w := createTestWorkspace(t, wsStore)

	got, err := wsStore.GetMember(context.Background(), w.ID, "nonexistent")
	if err != nil {
		t.Fatalf("GetMember failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-member")
	}
}

func TestWorkspacesStore_ListMembers(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	usersStore := NewUsersStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "listmember@example.com",
		Username:     "listmemberuser",
		DisplayName:  "List Member",
		PasswordHash: "hashed",
	}
	usersStore.Create(context.Background(), u)

	w := createTestWorkspace(t, wsStore)
	wsStore.AddMember(context.Background(), &workspaces.Member{
		WorkspaceID: w.ID,
		UserID:      u.ID,
		Role:        "member",
		JoinedAt:    time.Now(),
	})

	list, err := wsStore.ListMembers(context.Background(), w.ID)
	if err != nil {
		t.Fatalf("ListMembers failed: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("got %d members, want 1", len(list))
	}
}

func TestWorkspacesStore_UpdateMemberRole(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	usersStore := NewUsersStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "roleupdate@example.com",
		Username:     "roleupdateuser",
		DisplayName:  "Role Update",
		PasswordHash: "hashed",
	}
	usersStore.Create(context.Background(), u)

	w := createTestWorkspace(t, wsStore)
	wsStore.AddMember(context.Background(), &workspaces.Member{
		WorkspaceID: w.ID,
		UserID:      u.ID,
		Role:        "member",
		JoinedAt:    time.Now(),
	})

	err := wsStore.UpdateMemberRole(context.Background(), w.ID, u.ID, "admin")
	if err != nil {
		t.Fatalf("UpdateMemberRole failed: %v", err)
	}

	got, _ := wsStore.GetMember(context.Background(), w.ID, u.ID)
	if got.Role != "admin" {
		t.Errorf("got role %q, want %q", got.Role, "admin")
	}
}

func TestWorkspacesStore_RemoveMember(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	usersStore := NewUsersStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "removemember@example.com",
		Username:     "removememberuser",
		DisplayName:  "Remove Member",
		PasswordHash: "hashed",
	}
	usersStore.Create(context.Background(), u)

	w := createTestWorkspace(t, wsStore)
	wsStore.AddMember(context.Background(), &workspaces.Member{
		WorkspaceID: w.ID,
		UserID:      u.ID,
		Role:        "member",
		JoinedAt:    time.Now(),
	})

	err := wsStore.RemoveMember(context.Background(), w.ID, u.ID)
	if err != nil {
		t.Fatalf("RemoveMember failed: %v", err)
	}

	got, _ := wsStore.GetMember(context.Background(), w.ID, u.ID)
	if got != nil {
		t.Error("expected member to be removed")
	}
}
