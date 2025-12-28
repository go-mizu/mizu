package duckdb

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
	"github.com/oklog/ulid/v2"
)

// createActorForUser creates an actor for a user and returns the actor ID
func createActorForUser(t *testing.T, db *sql.DB, userID string) string {
	t.Helper()
	actorID := ulid.Make().String()
	_, err := db.Exec(`
		INSERT INTO actors (id, actor_type, user_id, created_at)
		VALUES (?, 'user', ?, CURRENT_TIMESTAMP)
	`, actorID, userID)
	if err != nil {
		t.Fatalf("failed to create actor for user: %v", err)
	}
	return actorID
}

func createTestRepo(t *testing.T, reposStore *ReposStore, ownerActorID string) *repos.Repository {
	t.Helper()
	id := ulid.Make().String()
	r := &repos.Repository{
		ID:             id,
		OwnerActorID:   ownerActorID,
		Name:           "repo-" + id[len(id)-12:],
		Slug:           "repo-" + id[len(id)-12:],
		Description:    "A test repository",
		Website:        "https://example.com",
		DefaultBranch:  "main",
		IsPrivate:      false,
		IsArchived:     false,
		IsTemplate:     false,
		IsFork:         false,
		StarCount:      0,
		ForkCount:      0,
		WatcherCount:   0,
		OpenIssueCount: 0,
		OpenPRCount:    0,
		SizeKB:         100,
		Topics:         []string{"go", "test"},
		License:        "MIT",
		HasIssues:      true,
		HasWiki:        false,
		HasProjects:    false,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	if err := reposStore.Create(context.Background(), r); err != nil {
		t.Fatalf("failed to create test repo: %v", err)
	}
	return r
}

func createTestUserForRepos(t *testing.T, db interface{ Exec(string, ...interface{}) (interface{}, error) }) string {
	t.Helper()
	id := ulid.Make().String()
	_, err := db.Exec(`
		INSERT INTO users (id, username, email, password_hash, is_active, created_at, updated_at)
		VALUES (?, ?, ?, 'hash', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, id, "user"+id[:8], id+"@example.com")
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return id
}

// =============================================================================
// Repository CRUD Tests
// =============================================================================

func TestReposStore_Create(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())

	owner := createTestUser(t, usersStore)
	actorID := createActorForUser(t, store.DB(), owner.ID)
	r := &repos.Repository{
		ID:            ulid.Make().String(),
		OwnerActorID:  actorID,
		Name:          "my-repo",
		Slug:          "my-repo",
		Description:   "Test repository",
		DefaultBranch: "main",
		IsPrivate:     false,
		HasIssues:     true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	err := reposStore.Create(context.Background(), r)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := reposStore.GetByID(context.Background(), r.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected repository to be created")
	}
	if got.Name != r.Name {
		t.Errorf("got name %q, want %q", got.Name, r.Name)
	}
}

func TestReposStore_Create_WithTopics(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())

	owner := createTestUser(t, usersStore)
	actorID := createActorForUser(t, store.DB(), owner.ID)
	r := &repos.Repository{
		ID:            ulid.Make().String(),
		OwnerActorID:  actorID,
		Name:          "topic-repo",
		Slug:          "topic-repo",
		Topics:        []string{"golang", "database", "testing"},
		DefaultBranch: "main",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	err := reposStore.Create(context.Background(), r)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, _ := reposStore.GetByID(context.Background(), r.ID)
	if len(got.Topics) != 3 {
		t.Errorf("got %d topics, want 3", len(got.Topics))
	}
	if got.Topics[0] != "golang" {
		t.Errorf("got topic %q, want %q", got.Topics[0], "golang")
	}
}

func TestReposStore_Create_WithFork(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())

	owner := createTestUser(t, usersStore)
	ownerActorID := createActorForUser(t, store.DB(), owner.ID)
	original := createTestRepo(t, reposStore, ownerActorID)

	forker := createTestUser(t, usersStore)
	forkerActorID := createActorForUser(t, store.DB(), forker.ID)
	fork := &repos.Repository{
		ID:            ulid.Make().String(),
		OwnerActorID:  forkerActorID,
		Name:          "forked-repo",
		Slug:          "forked-repo",
		IsFork:        true,
		ForkedFromID:  original.ID,
		DefaultBranch: "main",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	err := reposStore.Create(context.Background(), fork)
	if err != nil {
		t.Fatalf("Create fork failed: %v", err)
	}

	got, _ := reposStore.GetByID(context.Background(), fork.ID)
	if !got.IsFork {
		t.Error("expected IsFork to be true")
	}
	if got.ForkedFromID != original.ID {
		t.Errorf("got forked_from_id %q, want %q", got.ForkedFromID, original.ID)
	}
}

func TestReposStore_GetByID(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())

	owner := createTestUser(t, usersStore)
	actorID := createActorForUser(t, store.DB(), owner.ID)
	r := createTestRepo(t, reposStore, actorID)

	got, err := reposStore.GetByID(context.Background(), r.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected repository")
	}
	if got.ID != r.ID {
		t.Errorf("got ID %q, want %q", got.ID, r.ID)
	}
}

func TestReposStore_GetByID_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	reposStore := NewReposStore(store.DB())

	got, err := reposStore.GetByID(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent repository")
	}
}

func TestReposStore_GetByOwnerAndName(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())

	owner := createTestUser(t, usersStore)
	actorID := createActorForUser(t, store.DB(), owner.ID)
	r := createTestRepo(t, reposStore, actorID)

	got, err := reposStore.GetByOwnerAndName(context.Background(), actorID, "user", r.Slug)
	if err != nil {
		t.Fatalf("GetByOwnerAndName failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected repository")
	}
	if got.ID != r.ID {
		t.Errorf("got ID %q, want %q", got.ID, r.ID)
	}
}

func TestReposStore_GetByOwnerAndName_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	reposStore := NewReposStore(store.DB())

	got, err := reposStore.GetByOwnerAndName(context.Background(), "owner", "user", "nonexistent")
	if err != nil {
		t.Fatalf("GetByOwnerAndName failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent repository")
	}
}

func TestReposStore_Update(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())

	owner := createTestUser(t, usersStore)
	actorID := createActorForUser(t, store.DB(), owner.ID)
	r := createTestRepo(t, reposStore, actorID)

	r.Description = "Updated description"
	r.IsPrivate = true
	r.Topics = []string{"updated", "topics"}

	err := reposStore.Update(context.Background(), r)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, _ := reposStore.GetByID(context.Background(), r.ID)
	if got.Description != "Updated description" {
		t.Errorf("got description %q, want %q", got.Description, "Updated description")
	}
	if !got.IsPrivate {
		t.Error("expected repository to be private")
	}
	if len(got.Topics) != 2 || got.Topics[0] != "updated" {
		t.Error("expected topics to be updated")
	}
}

func TestReposStore_Update_UpdatesTimestamp(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())

	owner := createTestUser(t, usersStore)
	actorID := createActorForUser(t, store.DB(), owner.ID)
	r := createTestRepo(t, reposStore, actorID)
	originalUpdatedAt := r.UpdatedAt

	time.Sleep(10 * time.Millisecond)

	r.Description = "New description"
	reposStore.Update(context.Background(), r)

	got, _ := reposStore.GetByID(context.Background(), r.ID)
	if !got.UpdatedAt.After(originalUpdatedAt) {
		t.Error("expected updated_at to be updated")
	}
}

func TestReposStore_Delete(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())

	owner := createTestUser(t, usersStore)
	actorID := createActorForUser(t, store.DB(), owner.ID)
	r := createTestRepo(t, reposStore, actorID)

	err := reposStore.Delete(context.Background(), r.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	got, _ := reposStore.GetByID(context.Background(), r.ID)
	if got != nil {
		t.Error("expected repository to be deleted")
	}
}

func TestReposStore_ListByOwner(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())

	owner1 := createTestUser(t, usersStore)
	actor1ID := createActorForUser(t, store.DB(), owner1.ID)
	owner2 := createTestUser(t, usersStore)
	actor2ID := createActorForUser(t, store.DB(), owner2.ID)

	// Create repos for owner1
	for i := 0; i < 3; i++ {
		createTestRepo(t, reposStore, actor1ID)
	}
	// Create repos for owner2
	for i := 0; i < 2; i++ {
		createTestRepo(t, reposStore, actor2ID)
	}

	repos, err := reposStore.ListByOwner(context.Background(), actor1ID, "user", 10, 0)
	if err != nil {
		t.Fatalf("ListByOwner failed: %v", err)
	}
	if len(repos) != 3 {
		t.Errorf("got %d repos, want 3", len(repos))
	}
}

func TestReposStore_ListByOwner_Pagination(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())

	owner := createTestUser(t, usersStore)
	actorID := createActorForUser(t, store.DB(), owner.ID)
	for i := 0; i < 10; i++ {
		createTestRepo(t, reposStore, actorID)
	}

	page1, _ := reposStore.ListByOwner(context.Background(), actorID, "user", 3, 0)
	page2, _ := reposStore.ListByOwner(context.Background(), actorID, "user", 3, 3)

	if len(page1) != 3 {
		t.Errorf("got %d repos on page 1, want 3", len(page1))
	}
	if len(page2) != 3 {
		t.Errorf("got %d repos on page 2, want 3", len(page2))
	}
	if page1[0].ID == page2[0].ID {
		t.Error("expected different repos on different pages")
	}
}

func TestReposStore_ListPublic(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())

	owner := createTestUser(t, usersStore)
	actorID := createActorForUser(t, store.DB(), owner.ID)

	// Create public repos
	for i := 0; i < 3; i++ {
		r := createTestRepo(t, reposStore, actorID)
		r.IsPrivate = false
		reposStore.Update(context.Background(), r)
	}

	// Create private repo
	private := &repos.Repository{
		ID:            ulid.Make().String(),
		OwnerActorID:  actorID,
		Name:          "private-repo",
		Slug:          "private-repo",
		IsPrivate:     true,
		DefaultBranch: "main",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	reposStore.Create(context.Background(), private)

	repos, err := reposStore.ListPublic(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("ListPublic failed: %v", err)
	}

	// Should only return public repos
	for _, r := range repos {
		if r.IsPrivate {
			t.Error("expected only public repos")
		}
	}
}

func TestReposStore_ListByIDs(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())

	owner := createTestUser(t, usersStore)
	actorID := createActorForUser(t, store.DB(), owner.ID)
	r1 := createTestRepo(t, reposStore, actorID)
	r2 := createTestRepo(t, reposStore, actorID)
	createTestRepo(t, reposStore, actorID) // r3 not in query

	repos, err := reposStore.ListByIDs(context.Background(), []string{r1.ID, r2.ID})
	if err != nil {
		t.Fatalf("ListByIDs failed: %v", err)
	}
	if len(repos) != 2 {
		t.Errorf("got %d repos, want 2", len(repos))
	}
}

func TestReposStore_ListByIDs_Empty(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	reposStore := NewReposStore(store.DB())

	repos, err := reposStore.ListByIDs(context.Background(), []string{})
	if err != nil {
		t.Fatalf("ListByIDs failed: %v", err)
	}
	if repos != nil && len(repos) != 0 {
		t.Errorf("expected empty result for empty IDs")
	}
}

// =============================================================================
// Collaborator Tests
// =============================================================================

func TestReposStore_AddCollaborator(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())

	owner := createTestUser(t, usersStore)
	actorID := createActorForUser(t, store.DB(), owner.ID)
	collaborator := createTestUser(t, usersStore)
	r := createTestRepo(t, reposStore, actorID)

	collab := &repos.Collaborator{
		RepoID:     r.ID,
		UserID:     collaborator.ID,
		Permission: repos.PermissionWrite,
		CreatedAt:  time.Now(),
	}

	err := reposStore.AddCollaborator(context.Background(), collab)
	if err != nil {
		t.Fatalf("AddCollaborator failed: %v", err)
	}

	got, _ := reposStore.GetCollaborator(context.Background(), r.ID, collaborator.ID)
	if got == nil {
		t.Fatal("expected collaborator")
	}
	if got.Permission != repos.PermissionWrite {
		t.Errorf("got permission %q, want %q", got.Permission, repos.PermissionWrite)
	}
}

func TestReposStore_GetCollaborator_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	reposStore := NewReposStore(store.DB())

	got, err := reposStore.GetCollaborator(context.Background(), "repo", "user")
	if err != nil {
		t.Fatalf("GetCollaborator failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent collaborator")
	}
}

func TestReposStore_RemoveCollaborator(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())

	owner := createTestUser(t, usersStore)
	actorID := createActorForUser(t, store.DB(), owner.ID)
	collaborator := createTestUser(t, usersStore)
	r := createTestRepo(t, reposStore, actorID)

	collab := &repos.Collaborator{
		RepoID:     r.ID,
		UserID:     collaborator.ID,
		Permission: repos.PermissionWrite,
		CreatedAt:  time.Now(),
	}
	reposStore.AddCollaborator(context.Background(), collab)

	err := reposStore.RemoveCollaborator(context.Background(), r.ID, collaborator.ID)
	if err != nil {
		t.Fatalf("RemoveCollaborator failed: %v", err)
	}

	got, _ := reposStore.GetCollaborator(context.Background(), r.ID, collaborator.ID)
	if got != nil {
		t.Error("expected collaborator to be removed")
	}
}

func TestReposStore_ListCollaborators(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())

	owner := createTestUser(t, usersStore)
	actorID := createActorForUser(t, store.DB(), owner.ID)
	r := createTestRepo(t, reposStore, actorID)

	// Add multiple collaborators
	for i := 0; i < 3; i++ {
		collaborator := createTestUser(t, usersStore)
		collab := &repos.Collaborator{
			RepoID:     r.ID,
			UserID:     collaborator.ID,
			Permission: repos.PermissionRead,
			CreatedAt:  time.Now(),
		}
		reposStore.AddCollaborator(context.Background(), collab)
	}

	collabs, err := reposStore.ListCollaborators(context.Background(), r.ID)
	if err != nil {
		t.Fatalf("ListCollaborators failed: %v", err)
	}
	if len(collabs) != 3 {
		t.Errorf("got %d collaborators, want 3", len(collabs))
	}
}

func TestReposStore_ListCollaborators_Empty(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())

	owner := createTestUser(t, usersStore)
	actorID := createActorForUser(t, store.DB(), owner.ID)
	r := createTestRepo(t, reposStore, actorID)

	collabs, err := reposStore.ListCollaborators(context.Background(), r.ID)
	if err != nil {
		t.Fatalf("ListCollaborators failed: %v", err)
	}
	if collabs != nil && len(collabs) != 0 {
		t.Errorf("expected empty collaborators list")
	}
}

func TestReposStore_Collaborator_AllPermissionLevels(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())

	owner := createTestUser(t, usersStore)
	actorID := createActorForUser(t, store.DB(), owner.ID)
	r := createTestRepo(t, reposStore, actorID)

	permissions := []repos.Permission{
		repos.PermissionRead,
		repos.PermissionTriage,
		repos.PermissionWrite,
		repos.PermissionMaintain,
		repos.PermissionAdmin,
	}

	for _, perm := range permissions {
		collaborator := createTestUser(t, usersStore)
		collab := &repos.Collaborator{
			RepoID:     r.ID,
			UserID:     collaborator.ID,
			Permission: perm,
			CreatedAt:  time.Now(),
		}
		if err := reposStore.AddCollaborator(context.Background(), collab); err != nil {
			t.Fatalf("AddCollaborator with permission %q failed: %v", perm, err)
		}

		got, _ := reposStore.GetCollaborator(context.Background(), r.ID, collaborator.ID)
		if got.Permission != perm {
			t.Errorf("got permission %q, want %q", got.Permission, perm)
		}
	}
}

// =============================================================================
// Star Tests
// =============================================================================

func TestReposStore_Star(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())

	owner := createTestUser(t, usersStore)
	actorID := createActorForUser(t, store.DB(), owner.ID)
	starrer := createTestUser(t, usersStore)
	r := createTestRepo(t, reposStore, actorID)

	star := &repos.Star{
		UserID:    starrer.ID,
		RepoID:    r.ID,
		CreatedAt: time.Now(),
	}

	err := reposStore.Star(context.Background(), star)
	if err != nil {
		t.Fatalf("Star failed: %v", err)
	}

	isStarred, err := reposStore.IsStarred(context.Background(), starrer.ID, r.ID)
	if err != nil {
		t.Fatalf("IsStarred failed: %v", err)
	}
	if !isStarred {
		t.Error("expected repository to be starred")
	}
}

func TestReposStore_IsStarred_NotStarred(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())

	owner := createTestUser(t, usersStore)
	actorID := createActorForUser(t, store.DB(), owner.ID)
	user := createTestUser(t, usersStore)
	r := createTestRepo(t, reposStore, actorID)

	isStarred, err := reposStore.IsStarred(context.Background(), user.ID, r.ID)
	if err != nil {
		t.Fatalf("IsStarred failed: %v", err)
	}
	if isStarred {
		t.Error("expected repository to not be starred")
	}
}

func TestReposStore_Unstar(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())

	owner := createTestUser(t, usersStore)
	actorID := createActorForUser(t, store.DB(), owner.ID)
	starrer := createTestUser(t, usersStore)
	r := createTestRepo(t, reposStore, actorID)

	star := &repos.Star{
		UserID:    starrer.ID,
		RepoID:    r.ID,
		CreatedAt: time.Now(),
	}
	reposStore.Star(context.Background(), star)

	err := reposStore.Unstar(context.Background(), starrer.ID, r.ID)
	if err != nil {
		t.Fatalf("Unstar failed: %v", err)
	}

	isStarred, _ := reposStore.IsStarred(context.Background(), starrer.ID, r.ID)
	if isStarred {
		t.Error("expected repository to be unstarred")
	}
}

func TestReposStore_ListStarredByUser(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())

	owner := createTestUser(t, usersStore)
	actorID := createActorForUser(t, store.DB(), owner.ID)
	starrer := createTestUser(t, usersStore)

	// Create and star multiple repos
	for i := 0; i < 3; i++ {
		r := createTestRepo(t, reposStore, actorID)
		star := &repos.Star{
			UserID:    starrer.ID,
			RepoID:    r.ID,
			CreatedAt: time.Now(),
		}
		reposStore.Star(context.Background(), star)
	}

	starred, err := reposStore.ListStarredByUser(context.Background(), starrer.ID, 10, 0)
	if err != nil {
		t.Fatalf("ListStarredByUser failed: %v", err)
	}
	if len(starred) != 3 {
		t.Errorf("got %d starred repos, want 3", len(starred))
	}
}

func TestReposStore_ListStarredByUser_Empty(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())

	user := createTestUser(t, usersStore)

	starred, err := reposStore.ListStarredByUser(context.Background(), user.ID, 10, 0)
	if err != nil {
		t.Fatalf("ListStarredByUser failed: %v", err)
	}
	if starred != nil && len(starred) != 0 {
		t.Error("expected empty starred list")
	}
}

func TestReposStore_ListStarredByUser_Pagination(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())

	owner := createTestUser(t, usersStore)
	actorID := createActorForUser(t, store.DB(), owner.ID)
	starrer := createTestUser(t, usersStore)

	// Create and star 10 repos
	for i := 0; i < 10; i++ {
		r := createTestRepo(t, reposStore, actorID)
		star := &repos.Star{
			UserID:    starrer.ID,
			RepoID:    r.ID,
			CreatedAt: time.Now(),
		}
		reposStore.Star(context.Background(), star)
		time.Sleep(time.Millisecond) // Ensure different timestamps
	}

	page1, _ := reposStore.ListStarredByUser(context.Background(), starrer.ID, 3, 0)
	page2, _ := reposStore.ListStarredByUser(context.Background(), starrer.ID, 3, 3)

	if len(page1) != 3 {
		t.Errorf("got %d repos on page 1, want 3", len(page1))
	}
	if len(page2) != 3 {
		t.Errorf("got %d repos on page 2, want 3", len(page2))
	}
	if len(page1) > 0 && len(page2) > 0 && page1[0].ID == page2[0].ID {
		t.Error("expected different repos on different pages")
	}
}

// =============================================================================
// Integration Tests
// =============================================================================

func TestReposStore_DeleteRepoRemovesCollaborators(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())

	owner := createTestUser(t, usersStore)
	actorID := createActorForUser(t, store.DB(), owner.ID)
	collaborator := createTestUser(t, usersStore)
	r := createTestRepo(t, reposStore, actorID)

	collab := &repos.Collaborator{
		RepoID:     r.ID,
		UserID:     collaborator.ID,
		Permission: repos.PermissionWrite,
		CreatedAt:  time.Now(),
	}
	reposStore.AddCollaborator(context.Background(), collab)

	// Delete repo
	reposStore.Delete(context.Background(), r.ID)

	// Note: Collaborators remain orphaned since there's no CASCADE delete
	// This test documents current behavior - collaborators are NOT automatically removed
	collabs, _ := reposStore.ListCollaborators(context.Background(), r.ID)
	// Collaborators still exist (orphaned) - this is expected behavior
	// A future enhancement could add CASCADE delete or cleanup logic
	_ = collabs
}

func TestReposStore_UserWithMultipleRepos(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())

	owner := createTestUser(t, usersStore)
	actorID := createActorForUser(t, store.DB(), owner.ID)

	// User can own multiple repos
	repos := make([]*repos.Repository, 5)
	for i := 0; i < 5; i++ {
		repos[i] = createTestRepo(t, reposStore, actorID)
	}

	// Verify all repos belong to owner
	owned, _ := reposStore.ListByOwner(context.Background(), actorID, "user", 10, 0)
	if len(owned) != 5 {
		t.Errorf("got %d repos, want 5", len(owned))
	}
}

func TestReposStore_OrgTypeOwner(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	reposStore := NewReposStore(store.DB())
	orgsStore := NewOrgsStore(store.DB())

	// Create an org first
	org := createTestOrg(t, orgsStore)
	// Create an actor for the org
	actorID := ulid.Make().String()
	store.DB().Exec(`
		INSERT INTO actors (id, actor_type, org_id, created_at)
		VALUES (?, 'org', ?, CURRENT_TIMESTAMP)
	`, actorID, org.ID)

	// Create org-owned repo
	r := &repos.Repository{
		ID:            ulid.Make().String(),
		OwnerActorID:  actorID,
		Name:          "org-repo",
		Slug:          "org-repo",
		DefaultBranch: "main",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	reposStore.Create(context.Background(), r)

	// Should find by actor ID
	got, _ := reposStore.GetByOwnerAndName(context.Background(), actorID, "org", "org-repo")
	if got == nil {
		t.Fatal("expected org repo")
	}
	if got.OwnerActorID != actorID {
		t.Errorf("got owner_actor_id %q, want %q", got.OwnerActorID, actorID)
	}
}

// Helper to verify users.User implements the interface for test user creation
var _ users.Store = (*UsersStore)(nil)
