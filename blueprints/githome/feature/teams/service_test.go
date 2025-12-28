//go:build ignore
// +build ignore

// This test file is excluded from build until teams store is implemented in duckdb store.

package teams_test

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"github.com/go-mizu/blueprints/githome/feature/orgs"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/teams"
	"github.com/go-mizu/blueprints/githome/feature/users"
	"github.com/go-mizu/blueprints/githome/store/duckdb"
)

func setupTestService(t *testing.T) (*teams.Service, *duckdb.Store, func()) {
	t.Helper()
	t.Skip("teams store not yet implemented in duckdb store")

	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("failed to open duckdb: %v", err)
	}

	store, err := duckdb.New(db)
	if err != nil {
		db.Close()
		t.Fatalf("failed to create store: %v", err)
	}

	if err := store.Ensure(context.Background()); err != nil {
		store.Close()
		t.Fatalf("failed to ensure schema: %v", err)
	}

	service := teams.NewService(store.Teams(), store.Orgs(), store.Repos(), store.Users(), "https://api.example.com")

	cleanup := func() {
		store.Close()
	}

	return service, store, cleanup
}

func createTestUser(t *testing.T, store *duckdb.Store, login, email string) *users.User {
	t.Helper()
	user := &users.User{
		Login:        login,
		Email:        email,
		Name:         "Test User",
		PasswordHash: "hash",
		Type:         "User",
	}
	if err := store.Users().Create(context.Background(), user); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return user
}

func createTestOrg(t *testing.T, store *duckdb.Store, login string) *users.User {
	t.Helper()
	org := &users.User{
		Login:        login,
		Email:        login + "@example.com",
		Name:         "Test Organization",
		PasswordHash: "",
		Type:         "Organization",
	}
	if err := store.Users().Create(context.Background(), org); err != nil {
		t.Fatalf("failed to create test org: %v", err)
	}
	// Also create in orgs store if needed
	_ = store.Orgs().Create(context.Background(), &orgs.Organization{
		ID:    org.ID,
		Login: login,
		Email: login + "@example.com",
		Type:  "Organization",
	})
	return org
}

func createTestRepo(t *testing.T, store *duckdb.Store, owner *users.User, name string) *repos.Repository {
	t.Helper()
	repo := &repos.Repository{
		Name:          name,
		FullName:      owner.Login + "/" + name,
		OwnerID:       owner.ID,
		OwnerType:     owner.Type,
		Visibility:    "public",
		DefaultBranch: "main",
	}
	if err := store.Repos().Create(context.Background(), repo); err != nil {
		t.Fatalf("failed to create test repo: %v", err)
	}
	return repo
}

func createTestTeam(t *testing.T, service *teams.Service, org, name string) *teams.Team {
	t.Helper()
	team, err := service.Create(context.Background(), org, &teams.CreateIn{
		Name:        name,
		Description: "Test team",
	})
	if err != nil {
		t.Fatalf("failed to create test team: %v", err)
	}
	return team
}

// Team Creation Tests

func TestService_Create_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	createTestOrg(t, store, "testorg")

	team, err := service.Create(context.Background(), "testorg", &teams.CreateIn{
		Name:        "Engineering",
		Description: "The engineering team",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if team.Name != "Engineering" {
		t.Errorf("got name %q, want Engineering", team.Name)
	}
	if team.Slug != "engineering" {
		t.Errorf("got slug %q, want engineering", team.Slug)
	}
	if team.Description != "The engineering team" {
		t.Errorf("got description %q, want The engineering team", team.Description)
	}
	if team.ID == 0 {
		t.Error("expected ID to be assigned")
	}
	if team.Privacy != "secret" {
		t.Errorf("expected default privacy secret, got %q", team.Privacy)
	}
	if team.Permission != "pull" {
		t.Errorf("expected default permission pull, got %q", team.Permission)
	}
	if team.URL == "" {
		t.Error("expected URL to be populated")
	}
}

func TestService_Create_DuplicateName(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	createTestOrg(t, store, "testorg")
	createTestTeam(t, service, "testorg", "Engineering")

	_, err := service.Create(context.Background(), "testorg", &teams.CreateIn{
		Name: "Engineering",
	})
	if err != teams.ErrTeamExists {
		t.Errorf("expected ErrTeamExists, got %v", err)
	}
}

func TestService_Create_OrgNotFound(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	_, err := service.Create(context.Background(), "unknown", &teams.CreateIn{
		Name: "Engineering",
	})
	if err != orgs.ErrNotFound {
		t.Errorf("expected orgs.ErrNotFound, got %v", err)
	}
}

func TestService_Create_WithPrivacyAndPermission(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	createTestOrg(t, store, "testorg")

	team, err := service.Create(context.Background(), "testorg", &teams.CreateIn{
		Name:       "Engineering",
		Privacy:    "closed",
		Permission: "push",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if team.Privacy != "closed" {
		t.Errorf("got privacy %q, want closed", team.Privacy)
	}
	if team.Permission != "push" {
		t.Errorf("got permission %q, want push", team.Permission)
	}
}

func TestService_Create_WithMaintainers(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	createTestOrg(t, store, "testorg")
	createTestUser(t, store, "maintainer1", "m1@example.com")
	createTestUser(t, store, "maintainer2", "m2@example.com")

	team, err := service.Create(context.Background(), "testorg", &teams.CreateIn{
		Name:        "Engineering",
		Maintainers: []string{"maintainer1", "maintainer2"},
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if team.MembersCount != 2 {
		t.Errorf("expected members_count 2, got %d", team.MembersCount)
	}
}

// Team Retrieval Tests

func TestService_GetBySlug_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	createTestOrg(t, store, "testorg")
	created := createTestTeam(t, service, "testorg", "Engineering")

	team, err := service.GetBySlug(context.Background(), "testorg", "engineering")
	if err != nil {
		t.Fatalf("GetBySlug failed: %v", err)
	}

	if team.ID != created.ID {
		t.Errorf("got ID %d, want %d", team.ID, created.ID)
	}
}

func TestService_GetBySlug_NotFound(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	createTestOrg(t, store, "testorg")

	_, err := service.GetBySlug(context.Background(), "testorg", "nonexistent")
	if err != teams.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_GetByID_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	createTestOrg(t, store, "testorg")
	created := createTestTeam(t, service, "testorg", "Engineering")

	team, err := service.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if team.ID != created.ID {
		t.Errorf("got ID %d, want %d", team.ID, created.ID)
	}
}

func TestService_GetByID_NotFound(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	_, err := service.GetByID(context.Background(), 99999)
	if err != teams.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_List(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	createTestOrg(t, store, "testorg")
	createTestTeam(t, service, "testorg", "Engineering")
	createTestTeam(t, service, "testorg", "Design")
	createTestTeam(t, service, "testorg", "Product")

	list, err := service.List(context.Background(), "testorg", nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != 3 {
		t.Errorf("expected 3 teams, got %d", len(list))
	}
}

func TestService_List_Pagination(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	createTestOrg(t, store, "testorg")
	for i := 0; i < 5; i++ {
		createTestTeam(t, service, "testorg", "Team"+string(rune('a'+i)))
	}

	list, err := service.List(context.Background(), "testorg", &teams.ListOpts{
		Page:    1,
		PerPage: 2,
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("expected 2 teams, got %d", len(list))
	}
}

// Team Update Tests

func TestService_Update_Name(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	createTestOrg(t, store, "testorg")
	createTestTeam(t, service, "testorg", "Engineering")

	newName := "Platform"
	updated, err := service.Update(context.Background(), "testorg", "engineering", &teams.UpdateIn{
		Name: &newName,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if updated.Name != "Platform" {
		t.Errorf("got name %q, want Platform", updated.Name)
	}
	if updated.Slug != "platform" {
		t.Errorf("got slug %q, want platform", updated.Slug)
	}
}

func TestService_Update_Description(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	createTestOrg(t, store, "testorg")
	createTestTeam(t, service, "testorg", "Engineering")

	newDesc := "Updated description"
	updated, err := service.Update(context.Background(), "testorg", "engineering", &teams.UpdateIn{
		Description: &newDesc,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if updated.Description != "Updated description" {
		t.Errorf("got description %q, want Updated description", updated.Description)
	}
}

func TestService_Update_NotFound(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	createTestOrg(t, store, "testorg")

	newName := "newname"
	_, err := service.Update(context.Background(), "testorg", "nonexistent", &teams.UpdateIn{
		Name: &newName,
	})
	if err != teams.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// Team Delete Tests

func TestService_Delete_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	createTestOrg(t, store, "testorg")
	createTestTeam(t, service, "testorg", "Engineering")

	err := service.Delete(context.Background(), "testorg", "engineering")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deleted
	_, err = service.GetBySlug(context.Background(), "testorg", "engineering")
	if err != teams.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestService_Delete_NotFound(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	createTestOrg(t, store, "testorg")

	err := service.Delete(context.Background(), "testorg", "nonexistent")
	if err != teams.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// Membership Tests

func TestService_AddMembership_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	createTestOrg(t, store, "testorg")
	createTestTeam(t, service, "testorg", "Engineering")
	createTestUser(t, store, "member1", "member1@example.com")

	membership, err := service.AddMembership(context.Background(), "testorg", "engineering", "member1", "member")
	if err != nil {
		t.Fatalf("AddMembership failed: %v", err)
	}

	if membership.Role != "member" {
		t.Errorf("got role %q, want member", membership.Role)
	}
}

func TestService_AddMembership_Maintainer(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	createTestOrg(t, store, "testorg")
	createTestTeam(t, service, "testorg", "Engineering")
	createTestUser(t, store, "member1", "member1@example.com")

	membership, err := service.AddMembership(context.Background(), "testorg", "engineering", "member1", "maintainer")
	if err != nil {
		t.Fatalf("AddMembership failed: %v", err)
	}

	if membership.Role != "maintainer" {
		t.Errorf("got role %q, want maintainer", membership.Role)
	}
}

func TestService_AddMembership_DefaultRole(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	createTestOrg(t, store, "testorg")
	createTestTeam(t, service, "testorg", "Engineering")
	createTestUser(t, store, "member1", "member1@example.com")

	membership, err := service.AddMembership(context.Background(), "testorg", "engineering", "member1", "")
	if err != nil {
		t.Fatalf("AddMembership failed: %v", err)
	}

	if membership.Role != "member" {
		t.Errorf("expected default role member, got %q", membership.Role)
	}
}

func TestService_AddMembership_UpdatesCounter(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	createTestOrg(t, store, "testorg")
	team := createTestTeam(t, service, "testorg", "Engineering")
	createTestUser(t, store, "member1", "member1@example.com")
	createTestUser(t, store, "member2", "member2@example.com")

	_, _ = service.AddMembership(context.Background(), "testorg", "engineering", "member1", "member")
	_, _ = service.AddMembership(context.Background(), "testorg", "engineering", "member2", "member")

	// Verify counter
	updated, _ := service.GetByID(context.Background(), team.ID)
	if updated.MembersCount != 2 {
		t.Errorf("expected members_count 2, got %d", updated.MembersCount)
	}
}

func TestService_AddMembership_UserNotFound(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	createTestOrg(t, store, "testorg")
	createTestTeam(t, service, "testorg", "Engineering")

	_, err := service.AddMembership(context.Background(), "testorg", "engineering", "unknown", "member")
	if err != users.ErrNotFound {
		t.Errorf("expected users.ErrNotFound, got %v", err)
	}
}

func TestService_GetMembership_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	createTestOrg(t, store, "testorg")
	createTestTeam(t, service, "testorg", "Engineering")
	createTestUser(t, store, "member1", "member1@example.com")

	_, _ = service.AddMembership(context.Background(), "testorg", "engineering", "member1", "maintainer")

	membership, err := service.GetMembership(context.Background(), "testorg", "engineering", "member1")
	if err != nil {
		t.Fatalf("GetMembership failed: %v", err)
	}

	if membership.Role != "maintainer" {
		t.Errorf("got role %q, want maintainer", membership.Role)
	}
	if membership.URL == "" {
		t.Error("expected URL to be set")
	}
}

func TestService_GetMembership_NotMember(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	createTestOrg(t, store, "testorg")
	createTestTeam(t, service, "testorg", "Engineering")
	createTestUser(t, store, "nonmember", "nonmember@example.com")

	_, err := service.GetMembership(context.Background(), "testorg", "engineering", "nonmember")
	if err != teams.ErrNotMember {
		t.Errorf("expected ErrNotMember, got %v", err)
	}
}

func TestService_RemoveMembership_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	createTestOrg(t, store, "testorg")
	team := createTestTeam(t, service, "testorg", "Engineering")
	createTestUser(t, store, "member1", "member1@example.com")

	_, _ = service.AddMembership(context.Background(), "testorg", "engineering", "member1", "member")

	err := service.RemoveMembership(context.Background(), "testorg", "engineering", "member1")
	if err != nil {
		t.Fatalf("RemoveMembership failed: %v", err)
	}

	// Verify removed
	_, err = service.GetMembership(context.Background(), "testorg", "engineering", "member1")
	if err != teams.ErrNotMember {
		t.Errorf("expected ErrNotMember after remove, got %v", err)
	}

	// Verify counter decremented
	updated, _ := service.GetByID(context.Background(), team.ID)
	if updated.MembersCount != 0 {
		t.Errorf("expected members_count 0, got %d", updated.MembersCount)
	}
}

func TestService_ListMembers(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	createTestOrg(t, store, "testorg")
	createTestTeam(t, service, "testorg", "Engineering")
	createTestUser(t, store, "member1", "member1@example.com")
	createTestUser(t, store, "member2", "member2@example.com")

	_, _ = service.AddMembership(context.Background(), "testorg", "engineering", "member1", "member")
	_, _ = service.AddMembership(context.Background(), "testorg", "engineering", "member2", "maintainer")

	members, err := service.ListMembers(context.Background(), "testorg", "engineering", nil)
	if err != nil {
		t.Fatalf("ListMembers failed: %v", err)
	}

	if len(members) != 2 {
		t.Errorf("expected 2 members, got %d", len(members))
	}
}

// Repository Tests

func TestService_AddRepo_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	org := createTestOrg(t, store, "testorg")
	team := createTestTeam(t, service, "testorg", "Engineering")
	createTestRepo(t, store, org, "testrepo")

	err := service.AddRepo(context.Background(), "testorg", "engineering", "testorg", "testrepo", "push")
	if err != nil {
		t.Fatalf("AddRepo failed: %v", err)
	}

	// Verify counter incremented
	updated, _ := service.GetByID(context.Background(), team.ID)
	if updated.ReposCount != 1 {
		t.Errorf("expected repos_count 1, got %d", updated.ReposCount)
	}
}

func TestService_AddRepo_DefaultPermission(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	org := createTestOrg(t, store, "testorg")
	createTestTeam(t, service, "testorg", "Engineering")
	createTestRepo(t, store, org, "testrepo")

	err := service.AddRepo(context.Background(), "testorg", "engineering", "testorg", "testrepo", "")
	if err != nil {
		t.Fatalf("AddRepo failed: %v", err)
	}

	// Verify permission is default "pull"
	perm, err := service.CheckRepoPermission(context.Background(), "testorg", "engineering", "testorg", "testrepo")
	if err != nil {
		t.Fatalf("CheckRepoPermission failed: %v", err)
	}
	if perm.Permission != "pull" {
		t.Errorf("expected default permission pull, got %q", perm.Permission)
	}
}

func TestService_AddRepo_RepoNotFound(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	createTestOrg(t, store, "testorg")
	createTestTeam(t, service, "testorg", "Engineering")

	err := service.AddRepo(context.Background(), "testorg", "engineering", "testorg", "unknown", "push")
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

func TestService_RemoveRepo_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	org := createTestOrg(t, store, "testorg")
	team := createTestTeam(t, service, "testorg", "Engineering")
	createTestRepo(t, store, org, "testrepo")

	_ = service.AddRepo(context.Background(), "testorg", "engineering", "testorg", "testrepo", "push")

	err := service.RemoveRepo(context.Background(), "testorg", "engineering", "testorg", "testrepo")
	if err != nil {
		t.Fatalf("RemoveRepo failed: %v", err)
	}

	// Verify counter decremented
	updated, _ := service.GetByID(context.Background(), team.ID)
	if updated.ReposCount != 0 {
		t.Errorf("expected repos_count 0, got %d", updated.ReposCount)
	}
}

func TestService_CheckRepoPermission(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	org := createTestOrg(t, store, "testorg")
	createTestTeam(t, service, "testorg", "Engineering")
	createTestRepo(t, store, org, "testrepo")

	_ = service.AddRepo(context.Background(), "testorg", "engineering", "testorg", "testrepo", "admin")

	perm, err := service.CheckRepoPermission(context.Background(), "testorg", "engineering", "testorg", "testrepo")
	if err != nil {
		t.Fatalf("CheckRepoPermission failed: %v", err)
	}

	if perm.Permission != "admin" {
		t.Errorf("got permission %q, want admin", perm.Permission)
	}
}

func TestService_ListRepos(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	org := createTestOrg(t, store, "testorg")
	createTestTeam(t, service, "testorg", "Engineering")
	createTestRepo(t, store, org, "repo1")
	createTestRepo(t, store, org, "repo2")

	_ = service.AddRepo(context.Background(), "testorg", "engineering", "testorg", "repo1", "push")
	_ = service.AddRepo(context.Background(), "testorg", "engineering", "testorg", "repo2", "pull")

	repos, err := service.ListRepos(context.Background(), "testorg", "engineering", nil)
	if err != nil {
		t.Fatalf("ListRepos failed: %v", err)
	}

	if len(repos) != 2 {
		t.Errorf("expected 2 repos, got %d", len(repos))
	}
}

// Child Teams Tests

func TestService_ListChildren(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	createTestOrg(t, store, "testorg")
	parent := createTestTeam(t, service, "testorg", "Engineering")

	// Create child teams
	_, _ = service.Create(context.Background(), "testorg", &teams.CreateIn{
		Name:         "Frontend",
		ParentTeamID: &parent.ID,
	})
	_, _ = service.Create(context.Background(), "testorg", &teams.CreateIn{
		Name:         "Backend",
		ParentTeamID: &parent.ID,
	})

	children, err := service.ListChildren(context.Background(), "testorg", "engineering", nil)
	if err != nil {
		t.Fatalf("ListChildren failed: %v", err)
	}

	if len(children) != 2 {
		t.Errorf("expected 2 child teams, got %d", len(children))
	}
}

// URL Population Tests

func TestService_PopulateURLs(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	createTestOrg(t, store, "testorg")
	team := createTestTeam(t, service, "testorg", "Engineering")

	if team.URL != "https://api.example.com/api/v3/orgs/testorg/teams/engineering" {
		t.Errorf("unexpected URL: %s", team.URL)
	}
	if team.HTMLURL != "https://api.example.com/orgs/testorg/teams/engineering" {
		t.Errorf("unexpected HTMLURL: %s", team.HTMLURL)
	}
	if team.MembersURL == "" {
		t.Error("expected MembersURL to be set")
	}
	if team.RepositoriesURL == "" {
		t.Error("expected RepositoriesURL to be set")
	}
	if team.NodeID == "" {
		t.Error("expected NodeID to be set")
	}
}

// Integration Test - Teams Across Orgs

func TestService_TeamsAcrossOrgs(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	createTestOrg(t, store, "org1")
	createTestOrg(t, store, "org2")

	// Same team name in different orgs should work
	team1 := createTestTeam(t, service, "org1", "Engineering")
	team2 := createTestTeam(t, service, "org2", "Engineering")

	if team1.ID == team2.ID {
		t.Error("teams in different orgs should have different IDs")
	}

	// Each org should have its own teams
	list1, _ := service.List(context.Background(), "org1", nil)
	list2, _ := service.List(context.Background(), "org2", nil)

	if len(list1) != 1 {
		t.Errorf("org1 should have 1 team, got %d", len(list1))
	}
	if len(list2) != 1 {
		t.Errorf("org2 should have 1 team, got %d", len(list2))
	}
}
