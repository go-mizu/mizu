package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/kanban/feature/teams"
	"github.com/go-mizu/blueprints/kanban/feature/users"
	"github.com/go-mizu/blueprints/kanban/feature/workspaces"
	"github.com/oklog/ulid/v2"
)

func createTestTeam(t *testing.T, store *TeamsStore, workspaceID string) *teams.Team {
	t.Helper()
	id := ulid.Make().String()
	team := &teams.Team{
		ID:          id,
		WorkspaceID: workspaceID,
		Key:         "T" + id,
		Name:        "Team " + id,
	}
	if err := store.Create(context.Background(), team); err != nil {
		t.Fatalf("failed to create test team: %v", err)
	}
	return team
}

func TestTeamsStore_Create(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())

	w := createTestWorkspace(t, wsStore)

	team := &teams.Team{
		ID:          ulid.Make().String(),
		WorkspaceID: w.ID,
		Key:         "ENG",
		Name:        "Engineering",
	}

	err := teamsStore.Create(context.Background(), team)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := teamsStore.GetByID(context.Background(), team.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected team to be created")
	}
	if got.Key != team.Key {
		t.Errorf("got key %q, want %q", got.Key, team.Key)
	}
}

func TestTeamsStore_Create_DuplicateKey(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())

	w := createTestWorkspace(t, wsStore)

	team1 := &teams.Team{
		ID:          ulid.Make().String(),
		WorkspaceID: w.ID,
		Key:         "DUP",
		Name:        "Team 1",
	}
	team2 := &teams.Team{
		ID:          ulid.Make().String(),
		WorkspaceID: w.ID,
		Key:         "DUP", // same key
		Name:        "Team 2",
	}

	if err := teamsStore.Create(context.Background(), team1); err != nil {
		t.Fatalf("first Create failed: %v", err)
	}

	err := teamsStore.Create(context.Background(), team2)
	if err == nil {
		t.Error("expected error for duplicate key")
	}
}

func TestTeamsStore_GetByID(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)

	got, err := teamsStore.GetByID(context.Background(), team.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected team")
	}
	if got.ID != team.ID {
		t.Errorf("got ID %q, want %q", got.ID, team.ID)
	}
}

func TestTeamsStore_GetByID_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	teamsStore := NewTeamsStore(store.DB())

	got, err := teamsStore.GetByID(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent team")
	}
}

func TestTeamsStore_GetByKey(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)

	got, err := teamsStore.GetByKey(context.Background(), w.ID, team.Key)
	if err != nil {
		t.Fatalf("GetByKey failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected team")
	}
	if got.Key != team.Key {
		t.Errorf("got key %q, want %q", got.Key, team.Key)
	}
}

func TestTeamsStore_GetByKey_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())

	w := createTestWorkspace(t, wsStore)

	got, err := teamsStore.GetByKey(context.Background(), w.ID, "NONEXIST")
	if err != nil {
		t.Fatalf("GetByKey failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent key")
	}
}

func TestTeamsStore_ListByWorkspace(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	createTestTeam(t, teamsStore, w.ID)
	createTestTeam(t, teamsStore, w.ID)

	list, err := teamsStore.ListByWorkspace(context.Background(), w.ID)
	if err != nil {
		t.Fatalf("ListByWorkspace failed: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("got %d teams, want 2", len(list))
	}
}

func TestTeamsStore_ListByWorkspace_Empty(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())

	w := createTestWorkspace(t, wsStore)

	list, err := teamsStore.ListByWorkspace(context.Background(), w.ID)
	if err != nil {
		t.Fatalf("ListByWorkspace failed: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("got %d teams, want 0", len(list))
	}
}

func TestTeamsStore_Update(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)

	newName := "Updated Team"
	err := teamsStore.Update(context.Background(), team.ID, &teams.UpdateIn{
		Name: &newName,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, _ := teamsStore.GetByID(context.Background(), team.ID)
	if got.Name != newName {
		t.Errorf("got name %q, want %q", got.Name, newName)
	}
}

func TestTeamsStore_Delete(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)

	err := teamsStore.Delete(context.Background(), team.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	got, _ := teamsStore.GetByID(context.Background(), team.ID)
	if got != nil {
		t.Error("expected team to be deleted")
	}
}

func TestTeamsStore_AddMember(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	usersStore := NewUsersStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "teammember@example.com",
		Username:     "teammember",
		DisplayName:  "Team Member",
		PasswordHash: "hashed",
	}
	usersStore.Create(context.Background(), u)
	wsStore.AddMember(context.Background(), &workspaces.Member{
		WorkspaceID: w.ID,
		UserID:      u.ID,
		Role:        "member",
		JoinedAt:    time.Now(),
	})

	m := &teams.Member{
		TeamID:   team.ID,
		UserID:   u.ID,
		Role:     "member",
		JoinedAt: time.Now(),
	}

	err := teamsStore.AddMember(context.Background(), m)
	if err != nil {
		t.Fatalf("AddMember failed: %v", err)
	}

	got, err := teamsStore.GetMember(context.Background(), team.ID, u.ID)
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

func TestTeamsStore_GetMember_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)

	got, err := teamsStore.GetMember(context.Background(), team.ID, "nonexistent")
	if err != nil {
		t.Fatalf("GetMember failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-member")
	}
}

func TestTeamsStore_ListMembers(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	usersStore := NewUsersStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "listteam@example.com",
		Username:     "listteam",
		DisplayName:  "List Team",
		PasswordHash: "hashed",
	}
	usersStore.Create(context.Background(), u)
	wsStore.AddMember(context.Background(), &workspaces.Member{
		WorkspaceID: w.ID,
		UserID:      u.ID,
		Role:        "member",
		JoinedAt:    time.Now(),
	})
	teamsStore.AddMember(context.Background(), &teams.Member{
		TeamID:   team.ID,
		UserID:   u.ID,
		Role:     "member",
		JoinedAt: time.Now(),
	})

	list, err := teamsStore.ListMembers(context.Background(), team.ID)
	if err != nil {
		t.Fatalf("ListMembers failed: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("got %d members, want 1", len(list))
	}
}

func TestTeamsStore_UpdateMemberRole(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	usersStore := NewUsersStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "teamrole@example.com",
		Username:     "teamrole",
		DisplayName:  "Team Role",
		PasswordHash: "hashed",
	}
	usersStore.Create(context.Background(), u)
	wsStore.AddMember(context.Background(), &workspaces.Member{
		WorkspaceID: w.ID,
		UserID:      u.ID,
		Role:        "member",
		JoinedAt:    time.Now(),
	})
	teamsStore.AddMember(context.Background(), &teams.Member{
		TeamID:   team.ID,
		UserID:   u.ID,
		Role:     "member",
		JoinedAt: time.Now(),
	})

	err := teamsStore.UpdateMemberRole(context.Background(), team.ID, u.ID, "lead")
	if err != nil {
		t.Fatalf("UpdateMemberRole failed: %v", err)
	}

	got, _ := teamsStore.GetMember(context.Background(), team.ID, u.ID)
	if got.Role != "lead" {
		t.Errorf("got role %q, want %q", got.Role, "lead")
	}
}

func TestTeamsStore_RemoveMember(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	usersStore := NewUsersStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "removeteam@example.com",
		Username:     "removeteam",
		DisplayName:  "Remove Team",
		PasswordHash: "hashed",
	}
	usersStore.Create(context.Background(), u)
	wsStore.AddMember(context.Background(), &workspaces.Member{
		WorkspaceID: w.ID,
		UserID:      u.ID,
		Role:        "member",
		JoinedAt:    time.Now(),
	})
	teamsStore.AddMember(context.Background(), &teams.Member{
		TeamID:   team.ID,
		UserID:   u.ID,
		Role:     "member",
		JoinedAt: time.Now(),
	})

	err := teamsStore.RemoveMember(context.Background(), team.ID, u.ID)
	if err != nil {
		t.Fatalf("RemoveMember failed: %v", err)
	}

	got, _ := teamsStore.GetMember(context.Background(), team.ID, u.ID)
	if got != nil {
		t.Error("expected member to be removed")
	}
}
