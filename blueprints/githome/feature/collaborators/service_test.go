package collaborators_test

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"github.com/go-mizu/blueprints/githome/feature/collaborators"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
	"github.com/go-mizu/blueprints/githome/store/duckdb"
)

func setupTestService(t *testing.T) (*collaborators.Service, *duckdb.Store, func()) {
	t.Helper()

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

	collaboratorsStore := duckdb.NewCollaboratorsStore(db)
	service := collaborators.NewService(collaboratorsStore, store.Repos(), store.Users(), "https://api.example.com")

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

func createTestRepo(t *testing.T, store *duckdb.Store, owner *users.User, name string, private bool) *repos.Repository {
	t.Helper()
	repo := &repos.Repository{
		Name:          name,
		FullName:      owner.Login + "/" + name,
		OwnerID:       owner.ID,
		OwnerType:     "User",
		Private:       private,
		Visibility:    "public",
		DefaultBranch: "main",
	}
	if private {
		repo.Visibility = "private"
	}
	if err := store.Repos().Create(context.Background(), repo); err != nil {
		t.Fatalf("failed to create test repo: %v", err)
	}
	return repo
}

// Permission Tests

func TestService_GetPermission_Owner(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo", false)

	perm, err := service.GetPermission(context.Background(), "owner", "testrepo", "owner")
	if err != nil {
		t.Fatalf("GetPermission failed: %v", err)
	}

	if perm.Permission != "admin" {
		t.Errorf("got permission %q, want admin", perm.Permission)
	}
	if perm.RoleName != "admin" {
		t.Errorf("got role_name %q, want admin", perm.RoleName)
	}
	if perm.User == nil {
		t.Error("expected user to be set")
	}
}

func TestService_GetPermission_PublicRepo(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	_ = createTestUser(t, store, "other", "other@example.com")
	createTestRepo(t, store, owner, "testrepo", false) // public

	perm, err := service.GetPermission(context.Background(), "owner", "testrepo", "other")
	if err != nil {
		t.Fatalf("GetPermission failed: %v", err)
	}

	if perm.Permission != "read" {
		t.Errorf("got permission %q, want read for public repo", perm.Permission)
	}
}

func TestService_GetPermission_PrivateRepo(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	_ = createTestUser(t, store, "other", "other@example.com")
	createTestRepo(t, store, owner, "testrepo", true) // private

	perm, err := service.GetPermission(context.Background(), "owner", "testrepo", "other")
	if err != nil {
		t.Fatalf("GetPermission failed: %v", err)
	}

	if perm.Permission != "none" {
		t.Errorf("got permission %q, want none for private repo", perm.Permission)
	}
}

func TestService_GetPermission_Collaborator(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	collab := createTestUser(t, store, "collab", "collab@example.com")
	createTestRepo(t, store, owner, "testrepo", true)

	// Add as collaborator directly
	_ = duckdb.NewCollaboratorsStore(store.DB()).Add(context.Background(), 1, collab.ID, "push")

	perm, err := service.GetPermission(context.Background(), "owner", "testrepo", "collab")
	if err != nil {
		t.Fatalf("GetPermission failed: %v", err)
	}

	if perm.Permission != "push" {
		t.Errorf("got permission %q, want push", perm.Permission)
	}
}

func TestService_GetPermission_RepoNotFound(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	createTestUser(t, store, "owner", "owner@example.com")

	_, err := service.GetPermission(context.Background(), "owner", "unknown", "owner")
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

func TestService_GetPermission_UserNotFound(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo", false)

	_, err := service.GetPermission(context.Background(), "owner", "testrepo", "unknown")
	if err != users.ErrNotFound {
		t.Errorf("expected users.ErrNotFound, got %v", err)
	}
}

// IsCollaborator Tests

func TestService_IsCollaborator_True(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	collab := createTestUser(t, store, "collab", "collab@example.com")
	createTestRepo(t, store, owner, "testrepo", false)

	// Add as collaborator directly
	_ = duckdb.NewCollaboratorsStore(store.DB()).Add(context.Background(), 1, collab.ID, "push")

	isCollab, err := service.IsCollaborator(context.Background(), "owner", "testrepo", "collab")
	if err != nil {
		t.Fatalf("IsCollaborator failed: %v", err)
	}

	if !isCollab {
		t.Error("expected IsCollaborator to return true")
	}
}

func TestService_IsCollaborator_False(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestUser(t, store, "other", "other@example.com")
	createTestRepo(t, store, owner, "testrepo", false)

	isCollab, err := service.IsCollaborator(context.Background(), "owner", "testrepo", "other")
	if err != nil {
		t.Fatalf("IsCollaborator failed: %v", err)
	}

	if isCollab {
		t.Error("expected IsCollaborator to return false")
	}
}

func TestService_IsCollaborator_UserNotFound(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo", false)

	isCollab, err := service.IsCollaborator(context.Background(), "owner", "testrepo", "unknown")
	if err != nil {
		t.Fatalf("IsCollaborator failed: %v", err)
	}

	if isCollab {
		t.Error("expected IsCollaborator to return false for unknown user")
	}
}

// Add/Remove Collaborator Tests

func TestService_Add_CreatesInvitation(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	invitee := createTestUser(t, store, "invitee", "invitee@example.com")
	createTestRepo(t, store, owner, "testrepo", false)

	inv, err := service.Add(context.Background(), "owner", "testrepo", "invitee", "push")
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	if inv == nil {
		t.Fatal("expected invitation to be created")
	}
	if inv.ID == 0 {
		t.Error("expected invitation ID to be assigned")
	}
	if inv.Permissions != "push" {
		t.Errorf("got permissions %q, want push", inv.Permissions)
	}
	if inv.Invitee.ID != invitee.ID {
		t.Error("expected invitee to be set correctly")
	}
	if inv.URL == "" {
		t.Error("expected URL to be populated")
	}
}

func TestService_Add_UpdatesExistingCollaborator(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	collab := createTestUser(t, store, "collab", "collab@example.com")
	createTestRepo(t, store, owner, "testrepo", false)

	// Add as collaborator first
	_ = duckdb.NewCollaboratorsStore(store.DB()).Add(context.Background(), 1, collab.ID, "pull")

	// Try to add again with different permission - should update
	inv, err := service.Add(context.Background(), "owner", "testrepo", "collab", "push")
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Should return nil invitation (no new invite needed)
	if inv != nil {
		t.Error("expected nil invitation for existing collaborator")
	}

	// Verify permission updated
	perm, _ := service.GetPermission(context.Background(), "owner", "testrepo", "collab")
	if perm.Permission != "push" {
		t.Errorf("expected permission to be updated to push, got %q", perm.Permission)
	}
}

func TestService_Add_DefaultPermission(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestUser(t, store, "invitee", "invitee@example.com")
	createTestRepo(t, store, owner, "testrepo", false)

	inv, err := service.Add(context.Background(), "owner", "testrepo", "invitee", "")
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	if inv.Permissions != "push" {
		t.Errorf("expected default permission to be push, got %q", inv.Permissions)
	}
}

func TestService_Add_RepoNotFound(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	createTestUser(t, store, "owner", "owner@example.com")
	createTestUser(t, store, "invitee", "invitee@example.com")

	_, err := service.Add(context.Background(), "owner", "unknown", "invitee", "push")
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

func TestService_Add_UserNotFound(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo", false)

	_, err := service.Add(context.Background(), "owner", "testrepo", "unknown", "push")
	if err != users.ErrNotFound {
		t.Errorf("expected users.ErrNotFound, got %v", err)
	}
}

func TestService_Remove_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	collab := createTestUser(t, store, "collab", "collab@example.com")
	createTestRepo(t, store, owner, "testrepo", false)

	// Add first
	_ = duckdb.NewCollaboratorsStore(store.DB()).Add(context.Background(), 1, collab.ID, "push")

	err := service.Remove(context.Background(), "owner", "testrepo", "collab")
	if err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	// Verify removed
	isCollab, _ := service.IsCollaborator(context.Background(), "owner", "testrepo", "collab")
	if isCollab {
		t.Error("expected collaborator to be removed")
	}
}

func TestService_Remove_RepoNotFound(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	createTestUser(t, store, "owner", "owner@example.com")
	createTestUser(t, store, "collab", "collab@example.com")

	err := service.Remove(context.Background(), "owner", "unknown", "collab")
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

func TestService_Remove_UserNotFound(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo", false)

	err := service.Remove(context.Background(), "owner", "testrepo", "unknown")
	if err != users.ErrNotFound {
		t.Errorf("expected users.ErrNotFound, got %v", err)
	}
}

// List Tests

func TestService_List(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	collab1 := createTestUser(t, store, "collab1", "collab1@example.com")
	collab2 := createTestUser(t, store, "collab2", "collab2@example.com")
	createTestRepo(t, store, owner, "testrepo", false)

	_ = duckdb.NewCollaboratorsStore(store.DB()).Add(context.Background(), 1, collab1.ID, "push")
	_ = duckdb.NewCollaboratorsStore(store.DB()).Add(context.Background(), 1, collab2.ID, "admin")

	list, err := service.List(context.Background(), "owner", "testrepo", nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("expected 2 collaborators, got %d", len(list))
	}
}

func TestService_List_Pagination(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo", false)

	for i := 0; i < 5; i++ {
		collab := createTestUser(t, store, "collab"+string(rune('a'+i)), "collab"+string(rune('a'+i))+"@example.com")
		_ = duckdb.NewCollaboratorsStore(store.DB()).Add(context.Background(), 1, collab.ID, "push")
	}

	list, err := service.List(context.Background(), "owner", "testrepo", &collaborators.ListOpts{
		Page:    1,
		PerPage: 2,
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("expected 2 collaborators, got %d", len(list))
	}
}

func TestService_List_RepoNotFound(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	_, err := service.List(context.Background(), "unknown", "repo", nil)
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

// Invitation Tests

func TestService_ListInvitations(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestUser(t, store, "invitee1", "invitee1@example.com")
	createTestUser(t, store, "invitee2", "invitee2@example.com")
	createTestRepo(t, store, owner, "testrepo", false)

	_, _ = service.Add(context.Background(), "owner", "testrepo", "invitee1", "push")
	_, _ = service.Add(context.Background(), "owner", "testrepo", "invitee2", "admin")

	invitations, err := service.ListInvitations(context.Background(), "owner", "testrepo", nil)
	if err != nil {
		t.Fatalf("ListInvitations failed: %v", err)
	}

	if len(invitations) != 2 {
		t.Errorf("expected 2 invitations, got %d", len(invitations))
	}
}

func TestService_UpdateInvitation(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestUser(t, store, "invitee", "invitee@example.com")
	createTestRepo(t, store, owner, "testrepo", false)

	inv, _ := service.Add(context.Background(), "owner", "testrepo", "invitee", "pull")

	updated, err := service.UpdateInvitation(context.Background(), "owner", "testrepo", inv.ID, "admin")
	if err != nil {
		t.Fatalf("UpdateInvitation failed: %v", err)
	}

	if updated.Permissions != "admin" {
		t.Errorf("got permissions %q, want admin", updated.Permissions)
	}
}

func TestService_UpdateInvitation_NotFound(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo", false)

	_, err := service.UpdateInvitation(context.Background(), "owner", "testrepo", 99999, "admin")
	if err != collaborators.ErrInvitationNotFound {
		t.Errorf("expected ErrInvitationNotFound, got %v", err)
	}
}

func TestService_DeleteInvitation(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestUser(t, store, "invitee", "invitee@example.com")
	createTestRepo(t, store, owner, "testrepo", false)

	inv, _ := service.Add(context.Background(), "owner", "testrepo", "invitee", "push")

	err := service.DeleteInvitation(context.Background(), "owner", "testrepo", inv.ID)
	if err != nil {
		t.Fatalf("DeleteInvitation failed: %v", err)
	}

	// Verify deleted
	invitations, _ := service.ListInvitations(context.Background(), "owner", "testrepo", nil)
	if len(invitations) != 0 {
		t.Error("expected invitation to be deleted")
	}
}

func TestService_DeleteInvitation_NotFound(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo", false)

	err := service.DeleteInvitation(context.Background(), "owner", "testrepo", 99999)
	if err != collaborators.ErrInvitationNotFound {
		t.Errorf("expected ErrInvitationNotFound, got %v", err)
	}
}

// Accept/Decline Invitation Tests

func TestService_AcceptInvitation(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	invitee := createTestUser(t, store, "invitee", "invitee@example.com")
	createTestRepo(t, store, owner, "testrepo", false)

	inv, _ := service.Add(context.Background(), "owner", "testrepo", "invitee", "push")

	err := service.AcceptInvitation(context.Background(), invitee.ID, inv.ID)
	if err != nil {
		t.Fatalf("AcceptInvitation failed: %v", err)
	}

	// Verify now a collaborator
	isCollab, _ := service.IsCollaborator(context.Background(), "owner", "testrepo", "invitee")
	if !isCollab {
		t.Error("expected user to be a collaborator after accepting")
	}

	// Verify invitation deleted
	invitations, _ := service.ListInvitations(context.Background(), "owner", "testrepo", nil)
	if len(invitations) != 0 {
		t.Error("expected invitation to be deleted after accepting")
	}
}

func TestService_AcceptInvitation_NotFound(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	invitee := createTestUser(t, store, "invitee", "invitee@example.com")

	err := service.AcceptInvitation(context.Background(), invitee.ID, 99999)
	if err != collaborators.ErrInvitationNotFound {
		t.Errorf("expected ErrInvitationNotFound, got %v", err)
	}
}

func TestService_AcceptInvitation_WrongUser(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	_ = createTestUser(t, store, "invitee", "invitee@example.com")
	other := createTestUser(t, store, "other", "other@example.com")
	createTestRepo(t, store, owner, "testrepo", false)

	inv, _ := service.Add(context.Background(), "owner", "testrepo", "invitee", "push")

	// Try to accept as wrong user
	err := service.AcceptInvitation(context.Background(), other.ID, inv.ID)
	if err != collaborators.ErrInvitationNotFound {
		t.Errorf("expected ErrInvitationNotFound for wrong user, got %v", err)
	}
}

func TestService_DeclineInvitation(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	invitee := createTestUser(t, store, "invitee", "invitee@example.com")
	createTestRepo(t, store, owner, "testrepo", false)

	inv, _ := service.Add(context.Background(), "owner", "testrepo", "invitee", "push")

	err := service.DeclineInvitation(context.Background(), invitee.ID, inv.ID)
	if err != nil {
		t.Fatalf("DeclineInvitation failed: %v", err)
	}

	// Verify not a collaborator
	isCollab, _ := service.IsCollaborator(context.Background(), "owner", "testrepo", "invitee")
	if isCollab {
		t.Error("expected user not to be a collaborator after declining")
	}

	// Verify invitation deleted
	invitations, _ := service.ListInvitations(context.Background(), "owner", "testrepo", nil)
	if len(invitations) != 0 {
		t.Error("expected invitation to be deleted after declining")
	}
}

func TestService_ListUserInvitations(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	invitee := createTestUser(t, store, "invitee", "invitee@example.com")
	createTestRepo(t, store, owner, "repo1", false)
	createTestRepo(t, store, owner, "repo2", false)

	_, _ = service.Add(context.Background(), "owner", "repo1", "invitee", "push")
	_, _ = service.Add(context.Background(), "owner", "repo2", "invitee", "admin")

	invitations, err := service.ListUserInvitations(context.Background(), invitee.ID, nil)
	if err != nil {
		t.Fatalf("ListUserInvitations failed: %v", err)
	}

	if len(invitations) != 2 {
		t.Errorf("expected 2 invitations, got %d", len(invitations))
	}
}

// Helper Function Tests

func TestPermissionToLevel(t *testing.T) {
	tests := []struct {
		permission string
		want       int
	}{
		{"admin", 4},
		{"maintain", 3},
		{"write", 2},
		{"push", 2},
		{"triage", 1},
		{"read", 0},
		{"pull", 0},
		{"unknown", -1},
	}

	for _, tt := range tests {
		got := collaborators.PermissionToLevel(tt.permission)
		if got != tt.want {
			t.Errorf("PermissionToLevel(%q) = %d, want %d", tt.permission, got, tt.want)
		}
	}
}

func TestPermissionToPermissions(t *testing.T) {
	// Test admin has all permissions
	p := collaborators.PermissionToPermissions("admin")
	if !p.Pull || !p.Triage || !p.Push || !p.Maintain || !p.Admin {
		t.Error("admin should have all permissions")
	}

	// Test read only has pull
	p = collaborators.PermissionToPermissions("read")
	if !p.Pull || p.Triage || p.Push || p.Maintain || p.Admin {
		t.Error("read should only have pull permission")
	}

	// Test push has pull, triage, push
	p = collaborators.PermissionToPermissions("push")
	if !p.Pull || !p.Triage || !p.Push || p.Maintain || p.Admin {
		t.Error("push should have pull, triage, push permissions")
	}
}
