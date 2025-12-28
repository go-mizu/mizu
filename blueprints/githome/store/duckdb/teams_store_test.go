package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/teams"
	"github.com/oklog/ulid/v2"
)

func createTestTeam(t *testing.T, store *TeamsStore, orgID string) *teams.Team {
	t.Helper()
	id := ulid.Make().String()
	team := &teams.Team{
		ID:          id,
		OrgID:       orgID,
		Name:        "Team " + id[len(id)-8:],
		Slug:        "team-" + id[len(id)-8:],
		Description: "Test team description",
		Permission:  "read",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := store.Create(context.Background(), team); err != nil {
		t.Fatalf("failed to create test team: %v", err)
	}
	return team
}

// =============================================================================
// Team CRUD Tests
// =============================================================================

func TestTeamsStore_Create(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgsStore := NewOrgsStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())

	org := createTestOrg(t, orgsStore)

	team := &teams.Team{
		ID:          ulid.Make().String(),
		OrgID:       org.ID,
		Name:        "Engineering",
		Slug:        "engineering",
		Description: "Engineering team",
		Permission:  "write",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
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
	if got.Name != "Engineering" {
		t.Errorf("got name %q, want %q", got.Name, "Engineering")
	}
	if got.Slug != "engineering" {
		t.Errorf("got slug %q, want %q", got.Slug, "engineering")
	}
}

func TestTeamsStore_Create_WithParent(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgsStore := NewOrgsStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())

	org := createTestOrg(t, orgsStore)
	parentTeam := createTestTeam(t, teamsStore, org.ID)

	childTeam := &teams.Team{
		ID:          ulid.Make().String(),
		OrgID:       org.ID,
		Name:        "Frontend",
		Slug:        "frontend",
		Description: "Frontend team",
		Permission:  "write",
		ParentID:    parentTeam.ID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	teamsStore.Create(context.Background(), childTeam)

	got, _ := teamsStore.GetByID(context.Background(), childTeam.ID)
	if got.ParentID != parentTeam.ID {
		t.Errorf("got parent_id %q, want %q", got.ParentID, parentTeam.ID)
	}
}

func TestTeamsStore_GetByID(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgsStore := NewOrgsStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())

	org := createTestOrg(t, orgsStore)
	team := createTestTeam(t, teamsStore, org.ID)

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

func TestTeamsStore_GetBySlug(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgsStore := NewOrgsStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())

	org := createTestOrg(t, orgsStore)
	team := createTestTeam(t, teamsStore, org.ID)

	got, err := teamsStore.GetBySlug(context.Background(), org.ID, team.Slug)
	if err != nil {
		t.Fatalf("GetBySlug failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected team")
	}
	if got.Slug != team.Slug {
		t.Errorf("got slug %q, want %q", got.Slug, team.Slug)
	}
}

func TestTeamsStore_Update(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgsStore := NewOrgsStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())

	org := createTestOrg(t, orgsStore)
	team := createTestTeam(t, teamsStore, org.ID)

	team.Name = "Updated Name"
	team.Description = "Updated description"
	team.Permission = "admin"

	err := teamsStore.Update(context.Background(), team)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, _ := teamsStore.GetByID(context.Background(), team.ID)
	if got.Name != "Updated Name" {
		t.Errorf("got name %q, want %q", got.Name, "Updated Name")
	}
	if got.Permission != "admin" {
		t.Errorf("got permission %q, want %q", got.Permission, "admin")
	}
}

func TestTeamsStore_Delete(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgsStore := NewOrgsStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())

	org := createTestOrg(t, orgsStore)
	team := createTestTeam(t, teamsStore, org.ID)

	err := teamsStore.Delete(context.Background(), team.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	got, _ := teamsStore.GetByID(context.Background(), team.ID)
	if got != nil {
		t.Error("expected team to be deleted")
	}
}

// =============================================================================
// List Tests
// =============================================================================

func TestTeamsStore_List(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgsStore := NewOrgsStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())

	org := createTestOrg(t, orgsStore)

	for i := 0; i < 5; i++ {
		createTestTeam(t, teamsStore, org.ID)
	}

	list, err := teamsStore.List(context.Background(), org.ID, 10, 0)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 5 {
		t.Errorf("got %d teams, want 5", len(list))
	}
}

func TestTeamsStore_ListChildren(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgsStore := NewOrgsStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())

	org := createTestOrg(t, orgsStore)
	parentTeam := createTestTeam(t, teamsStore, org.ID)

	// Create child teams
	for i := 0; i < 3; i++ {
		child := &teams.Team{
			ID:          ulid.Make().String(),
			OrgID:       org.ID,
			Name:        "Child " + ulid.Make().String()[20:],
			Slug:        "child-" + ulid.Make().String()[20:],
			Description: "Child team",
			Permission:  "read",
			ParentID:    parentTeam.ID,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		teamsStore.Create(context.Background(), child)
	}

	list, err := teamsStore.ListChildren(context.Background(), parentTeam.ID)
	if err != nil {
		t.Fatalf("ListChildren failed: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("got %d child teams, want 3", len(list))
	}
}

// =============================================================================
// Member Tests
// =============================================================================

func TestTeamsStore_AddMember(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	orgsStore := NewOrgsStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())

	user := createTestUser(t, usersStore)
	org := createTestOrg(t, orgsStore)
	team := createTestTeam(t, teamsStore, org.ID)

	member := &teams.TeamMember{
		TeamID:    team.ID,
		UserID:    user.ID,
		Role:      "member",
		CreatedAt: time.Now(),
	}

	err := teamsStore.AddMember(context.Background(), member)
	if err != nil {
		t.Fatalf("AddMember failed: %v", err)
	}

	got, _ := teamsStore.GetMember(context.Background(), team.ID, user.ID)
	if got == nil {
		t.Fatal("expected member")
	}
	if got.Role != "member" {
		t.Errorf("got role %q, want %q", got.Role, "member")
	}
}

func TestTeamsStore_UpdateMember(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	orgsStore := NewOrgsStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())

	user := createTestUser(t, usersStore)
	org := createTestOrg(t, orgsStore)
	team := createTestTeam(t, teamsStore, org.ID)

	member := &teams.TeamMember{
		TeamID:    team.ID,
		UserID:    user.ID,
		Role:      "member",
		CreatedAt: time.Now(),
	}
	teamsStore.AddMember(context.Background(), member)

	member.Role = "maintainer"
	err := teamsStore.UpdateMember(context.Background(), member)
	if err != nil {
		t.Fatalf("UpdateMember failed: %v", err)
	}

	got, _ := teamsStore.GetMember(context.Background(), team.ID, user.ID)
	if got.Role != "maintainer" {
		t.Errorf("got role %q, want %q", got.Role, "maintainer")
	}
}

func TestTeamsStore_RemoveMember(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	orgsStore := NewOrgsStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())

	user := createTestUser(t, usersStore)
	org := createTestOrg(t, orgsStore)
	team := createTestTeam(t, teamsStore, org.ID)

	member := &teams.TeamMember{
		TeamID:    team.ID,
		UserID:    user.ID,
		Role:      "member",
		CreatedAt: time.Now(),
	}
	teamsStore.AddMember(context.Background(), member)

	err := teamsStore.RemoveMember(context.Background(), team.ID, user.ID)
	if err != nil {
		t.Fatalf("RemoveMember failed: %v", err)
	}

	got, _ := teamsStore.GetMember(context.Background(), team.ID, user.ID)
	if got != nil {
		t.Error("expected member to be removed")
	}
}

func TestTeamsStore_ListMembers(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	orgsStore := NewOrgsStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())

	org := createTestOrg(t, orgsStore)
	team := createTestTeam(t, teamsStore, org.ID)

	for i := 0; i < 5; i++ {
		user := createTestUser(t, usersStore)
		member := &teams.TeamMember{
			TeamID:    team.ID,
			UserID:    user.ID,
			Role:      "member",
			CreatedAt: time.Now(),
		}
		teamsStore.AddMember(context.Background(), member)
	}

	list, err := teamsStore.ListMembers(context.Background(), team.ID, 10, 0)
	if err != nil {
		t.Fatalf("ListMembers failed: %v", err)
	}
	if len(list) != 5 {
		t.Errorf("got %d members, want 5", len(list))
	}
}

func TestTeamsStore_ListByUser(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	orgsStore := NewOrgsStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())

	user := createTestUser(t, usersStore)
	org := createTestOrg(t, orgsStore)

	for i := 0; i < 3; i++ {
		team := createTestTeam(t, teamsStore, org.ID)
		member := &teams.TeamMember{
			TeamID:    team.ID,
			UserID:    user.ID,
			Role:      "member",
			CreatedAt: time.Now(),
		}
		teamsStore.AddMember(context.Background(), member)
	}

	list, err := teamsStore.ListByUser(context.Background(), org.ID, user.ID)
	if err != nil {
		t.Fatalf("ListByUser failed: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("got %d teams, want 3", len(list))
	}
}

// =============================================================================
// Repo Tests
// =============================================================================

func TestTeamsStore_AddRepo(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())
	orgsStore := NewOrgsStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())

	user := createTestUser(t, usersStore)
	repo := createTestRepo(t, reposStore, user.ID)
	org := createTestOrg(t, orgsStore)
	team := createTestTeam(t, teamsStore, org.ID)

	teamRepo := &teams.TeamRepo{
		TeamID:     team.ID,
		RepoID:     repo.ID,
		Permission: "write",
		CreatedAt:  time.Now(),
	}

	err := teamsStore.AddRepo(context.Background(), teamRepo)
	if err != nil {
		t.Fatalf("AddRepo failed: %v", err)
	}

	got, _ := teamsStore.GetRepo(context.Background(), team.ID, repo.ID)
	if got == nil {
		t.Fatal("expected team repo")
	}
	if got.Permission != "write" {
		t.Errorf("got permission %q, want %q", got.Permission, "write")
	}
}

func TestTeamsStore_UpdateRepo(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())
	orgsStore := NewOrgsStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())

	user := createTestUser(t, usersStore)
	repo := createTestRepo(t, reposStore, user.ID)
	org := createTestOrg(t, orgsStore)
	team := createTestTeam(t, teamsStore, org.ID)

	teamRepo := &teams.TeamRepo{
		TeamID:     team.ID,
		RepoID:     repo.ID,
		Permission: "read",
		CreatedAt:  time.Now(),
	}
	teamsStore.AddRepo(context.Background(), teamRepo)

	teamRepo.Permission = "admin"
	err := teamsStore.UpdateRepo(context.Background(), teamRepo)
	if err != nil {
		t.Fatalf("UpdateRepo failed: %v", err)
	}

	got, _ := teamsStore.GetRepo(context.Background(), team.ID, repo.ID)
	if got.Permission != "admin" {
		t.Errorf("got permission %q, want %q", got.Permission, "admin")
	}
}

func TestTeamsStore_RemoveRepo(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())
	orgsStore := NewOrgsStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())

	user := createTestUser(t, usersStore)
	repo := createTestRepo(t, reposStore, user.ID)
	org := createTestOrg(t, orgsStore)
	team := createTestTeam(t, teamsStore, org.ID)

	teamRepo := &teams.TeamRepo{
		TeamID:     team.ID,
		RepoID:     repo.ID,
		Permission: "write",
		CreatedAt:  time.Now(),
	}
	teamsStore.AddRepo(context.Background(), teamRepo)

	err := teamsStore.RemoveRepo(context.Background(), team.ID, repo.ID)
	if err != nil {
		t.Fatalf("RemoveRepo failed: %v", err)
	}

	got, _ := teamsStore.GetRepo(context.Background(), team.ID, repo.ID)
	if got != nil {
		t.Error("expected team repo to be removed")
	}
}

func TestTeamsStore_ListRepos(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())
	orgsStore := NewOrgsStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())

	user := createTestUser(t, usersStore)
	org := createTestOrg(t, orgsStore)
	team := createTestTeam(t, teamsStore, org.ID)

	for i := 0; i < 5; i++ {
		repo := createTestRepo(t, reposStore, user.ID)
		teamRepo := &teams.TeamRepo{
			TeamID:     team.ID,
			RepoID:     repo.ID,
			Permission: "read",
			CreatedAt:  time.Now(),
		}
		teamsStore.AddRepo(context.Background(), teamRepo)
	}

	list, err := teamsStore.ListRepos(context.Background(), team.ID, 10, 0)
	if err != nil {
		t.Fatalf("ListRepos failed: %v", err)
	}
	if len(list) != 5 {
		t.Errorf("got %d repos, want 5", len(list))
	}
}

// Verify interface compliance
var _ teams.Store = (*TeamsStore)(nil)
