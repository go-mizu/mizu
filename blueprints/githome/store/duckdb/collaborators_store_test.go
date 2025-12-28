package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/collaborators"
)

func createTestCollaborator(t *testing.T, store *CollaboratorsStore, repoID, userID string) *collaborators.Collaborator {
	t.Helper()
	c := &collaborators.Collaborator{
		RepoID:     repoID,
		UserID:     userID,
		Permission: "write",
		CreatedAt:  time.Now(),
	}
	if err := store.Create(context.Background(), c); err != nil {
		t.Fatalf("failed to create test collaborator: %v", err)
	}
	return c
}

// =============================================================================
// Collaborator CRUD Tests
// =============================================================================

func TestCollaboratorsStore_Create(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	collaboratorsStore := NewCollaboratorsStore(store.DB())

	repoID, _ := createRepoAndUser(t, store)
	collaborator := createTestUser(t, usersStore)

	c := &collaborators.Collaborator{
		RepoID:     repoID,
		UserID:     collaborator.ID,
		Permission: "write",
		CreatedAt:  time.Now(),
	}

	err := collaboratorsStore.Create(context.Background(), c)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := collaboratorsStore.Get(context.Background(), repoID, collaborator.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected collaborator to be created")
	}
	if got.Permission != "write" {
		t.Errorf("got permission %q, want %q", got.Permission, "write")
	}
}

func TestCollaboratorsStore_Create_DifferentPermissions(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	collaboratorsStore := NewCollaboratorsStore(store.DB())

	repoID, _ := createRepoAndUser(t, store)

	permissions := []string{"read", "triage", "write", "maintain", "admin"}

	for _, perm := range permissions {
		user := createTestUser(t, usersStore)
		c := &collaborators.Collaborator{
			RepoID:     repoID,
			UserID:     user.ID,
			Permission: perm,
			CreatedAt:  time.Now(),
		}
		collaboratorsStore.Create(context.Background(), c)

		got, _ := collaboratorsStore.Get(context.Background(), repoID, user.ID)
		if got.Permission != perm {
			t.Errorf("got permission %q, want %q", got.Permission, perm)
		}
	}
}

func TestCollaboratorsStore_Get(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	usersStore := NewUsersStore(store.DB())
	collaboratorsStore := NewCollaboratorsStore(store.DB())

	collaborator := createTestUser(t, usersStore)
	createTestCollaborator(t, collaboratorsStore, repoID, collaborator.ID)
	_ = userID

	got, err := collaboratorsStore.Get(context.Background(), repoID, collaborator.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected collaborator")
	}
}

func TestCollaboratorsStore_Get_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	collaboratorsStore := NewCollaboratorsStore(store.DB())

	got, err := collaboratorsStore.Get(context.Background(), "repo-nonexistent", "user-nonexistent")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent collaborator")
	}
}

func TestCollaboratorsStore_Update(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	usersStore := NewUsersStore(store.DB())
	collaboratorsStore := NewCollaboratorsStore(store.DB())

	collaborator := createTestUser(t, usersStore)
	c := createTestCollaborator(t, collaboratorsStore, repoID, collaborator.ID)

	c.Permission = "admin"

	err := collaboratorsStore.Update(context.Background(), c)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, _ := collaboratorsStore.Get(context.Background(), repoID, collaborator.ID)
	if got.Permission != "admin" {
		t.Errorf("got permission %q, want %q", got.Permission, "admin")
	}
}

func TestCollaboratorsStore_Delete(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	usersStore := NewUsersStore(store.DB())
	collaboratorsStore := NewCollaboratorsStore(store.DB())

	collaborator := createTestUser(t, usersStore)
	createTestCollaborator(t, collaboratorsStore, repoID, collaborator.ID)

	err := collaboratorsStore.Delete(context.Background(), repoID, collaborator.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	got, _ := collaboratorsStore.Get(context.Background(), repoID, collaborator.ID)
	if got != nil {
		t.Error("expected collaborator to be deleted")
	}
}

func TestCollaboratorsStore_Delete_NonExistent(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	collaboratorsStore := NewCollaboratorsStore(store.DB())

	err := collaboratorsStore.Delete(context.Background(), "repo-nonexistent", "user-nonexistent")
	if err != nil {
		t.Fatalf("Delete should not error for non-existent collaborator: %v", err)
	}
}

// =============================================================================
// List Tests
// =============================================================================

func TestCollaboratorsStore_List(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	usersStore := NewUsersStore(store.DB())
	collaboratorsStore := NewCollaboratorsStore(store.DB())

	for i := 0; i < 5; i++ {
		user := createTestUser(t, usersStore)
		createTestCollaborator(t, collaboratorsStore, repoID, user.ID)
	}

	list, err := collaboratorsStore.List(context.Background(), repoID, 10, 0)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 5 {
		t.Errorf("got %d collaborators, want 5", len(list))
	}
}

func TestCollaboratorsStore_List_Pagination(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	usersStore := NewUsersStore(store.DB())
	collaboratorsStore := NewCollaboratorsStore(store.DB())

	for i := 0; i < 10; i++ {
		user := createTestUser(t, usersStore)
		createTestCollaborator(t, collaboratorsStore, repoID, user.ID)
	}

	page1, _ := collaboratorsStore.List(context.Background(), repoID, 3, 0)
	page2, _ := collaboratorsStore.List(context.Background(), repoID, 3, 3)

	if len(page1) != 3 {
		t.Errorf("got %d collaborators on page 1, want 3", len(page1))
	}
	if len(page2) != 3 {
		t.Errorf("got %d collaborators on page 2, want 3", len(page2))
	}
}

func TestCollaboratorsStore_List_Empty(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	collaboratorsStore := NewCollaboratorsStore(store.DB())

	list, err := collaboratorsStore.List(context.Background(), repoID, 10, 0)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if list != nil && len(list) != 0 {
		t.Error("expected empty list")
	}
}

func TestCollaboratorsStore_List_PerRepo(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())
	collaboratorsStore := NewCollaboratorsStore(store.DB())

	owner := createTestUser(t, usersStore)
	actorID := createActorForUser(t, store.DB(), owner.ID)
	repo1 := createTestRepo(t, reposStore, actorID)
	repo2 := createTestRepo(t, reposStore, actorID)

	// Add 3 collaborators to repo1
	for i := 0; i < 3; i++ {
		user := createTestUser(t, usersStore)
		createTestCollaborator(t, collaboratorsStore, repo1.ID, user.ID)
	}

	// Add 2 collaborators to repo2
	for i := 0; i < 2; i++ {
		user := createTestUser(t, usersStore)
		createTestCollaborator(t, collaboratorsStore, repo2.ID, user.ID)
	}

	list1, _ := collaboratorsStore.List(context.Background(), repo1.ID, 10, 0)
	list2, _ := collaboratorsStore.List(context.Background(), repo2.ID, 10, 0)

	if len(list1) != 3 {
		t.Errorf("got %d collaborators for repo1, want 3", len(list1))
	}
	if len(list2) != 2 {
		t.Errorf("got %d collaborators for repo2, want 2", len(list2))
	}
}

// =============================================================================
// ListByUser Tests
// =============================================================================

func TestCollaboratorsStore_ListByUser(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())
	collaboratorsStore := NewCollaboratorsStore(store.DB())

	owner := createTestUser(t, usersStore)
	actorID := createActorForUser(t, store.DB(), owner.ID)
	collaborator := createTestUser(t, usersStore)

	// Add collaborator to 5 repos
	for i := 0; i < 5; i++ {
		repo := createTestRepo(t, reposStore, actorID)
		createTestCollaborator(t, collaboratorsStore, repo.ID, collaborator.ID)
	}

	list, err := collaboratorsStore.ListByUser(context.Background(), collaborator.ID, 10, 0)
	if err != nil {
		t.Fatalf("ListByUser failed: %v", err)
	}
	if len(list) != 5 {
		t.Errorf("got %d repos, want 5", len(list))
	}
}

func TestCollaboratorsStore_ListByUser_Pagination(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())
	collaboratorsStore := NewCollaboratorsStore(store.DB())

	owner := createTestUser(t, usersStore)
	actorID := createActorForUser(t, store.DB(), owner.ID)
	collaborator := createTestUser(t, usersStore)

	for i := 0; i < 10; i++ {
		repo := createTestRepo(t, reposStore, actorID)
		createTestCollaborator(t, collaboratorsStore, repo.ID, collaborator.ID)
	}

	page1, _ := collaboratorsStore.ListByUser(context.Background(), collaborator.ID, 3, 0)
	page2, _ := collaboratorsStore.ListByUser(context.Background(), collaborator.ID, 3, 3)

	if len(page1) != 3 {
		t.Errorf("got %d repos on page 1, want 3", len(page1))
	}
	if len(page2) != 3 {
		t.Errorf("got %d repos on page 2, want 3", len(page2))
	}
}

func TestCollaboratorsStore_ListByUser_Empty(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	collaboratorsStore := NewCollaboratorsStore(store.DB())

	user := createTestUser(t, usersStore)

	list, err := collaboratorsStore.ListByUser(context.Background(), user.ID, 10, 0)
	if err != nil {
		t.Fatalf("ListByUser failed: %v", err)
	}
	if list != nil && len(list) != 0 {
		t.Error("expected empty list")
	}
}

// Verify interface compliance
var _ collaborators.Store = (*CollaboratorsStore)(nil)
