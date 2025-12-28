package duckdb

import (
	"context"
	"testing"

	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
)

func createTestRepo(t *testing.T, store *ReposStore, ownerID int64, name string) *repos.Repository {
	t.Helper()
	r := &repos.Repository{
		Name:          name,
		FullName:      "testowner/" + name,
		OwnerID:       ownerID,
		OwnerType:     "User",
		DefaultBranch: "main",
		Visibility:    "public",
	}
	if err := store.Create(context.Background(), r); err != nil {
		t.Fatalf("failed to create test repo: %v", err)
	}
	return r
}

func TestReposStore_Create(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())

	u := createTestUser(t, usersStore, "repoowner")

	r := &repos.Repository{
		Name:          "testrepo",
		FullName:      u.Login + "/testrepo",
		OwnerID:       u.ID,
		OwnerType:     "User",
		DefaultBranch: "main",
		Visibility:    "public",
	}

	err := reposStore.Create(context.Background(), r)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if r.ID == 0 {
		t.Error("expected ID to be set")
	}
	if r.NodeID == "" {
		t.Error("expected NodeID to be set")
	}

	got, err := reposStore.GetByID(context.Background(), r.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected repo to be created")
	}
	if got.Name != r.Name {
		t.Errorf("got name %q, want %q", got.Name, r.Name)
	}
}

func TestReposStore_GetByID(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())

	u := createTestUser(t, usersStore, "getbyidowner")
	r := createTestRepo(t, reposStore, u.ID, "getbyidrepo")

	got, err := reposStore.GetByID(context.Background(), r.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected repo")
	}
	if got.ID != r.ID {
		t.Errorf("got ID %d, want %d", got.ID, r.ID)
	}
}

func TestReposStore_GetByOwnerAndName(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())

	u := createTestUser(t, usersStore, "ownernameowner")
	r := createTestRepo(t, reposStore, u.ID, "ownernameproj")

	got, err := reposStore.GetByOwnerAndName(context.Background(), u.ID, r.Name)
	if err != nil {
		t.Fatalf("GetByOwnerAndName failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected repo")
	}
	if got.Name != r.Name {
		t.Errorf("got name %q, want %q", got.Name, r.Name)
	}
}

func TestReposStore_GetByFullName(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())

	u := createTestUser(t, usersStore, "fullnameowner")
	r := &repos.Repository{
		Name:          "fullnamerepo",
		FullName:      u.Login + "/fullnamerepo",
		OwnerID:       u.ID,
		OwnerType:     "User",
		DefaultBranch: "main",
		Visibility:    "public",
	}
	reposStore.Create(context.Background(), r)

	got, err := reposStore.GetByFullName(context.Background(), u.Login, r.Name)
	if err != nil {
		t.Fatalf("GetByFullName failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected repo")
	}
	if got.FullName != r.FullName {
		t.Errorf("got full_name %q, want %q", got.FullName, r.FullName)
	}
}

func TestReposStore_Update(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())

	u := createTestUser(t, usersStore, "updateowner")
	r := createTestRepo(t, reposStore, u.ID, "updaterepo")

	newDesc := "Updated description"
	err := reposStore.Update(context.Background(), r.ID, &repos.UpdateIn{
		Description: &newDesc,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, _ := reposStore.GetByID(context.Background(), r.ID)
	if got.Description != newDesc {
		t.Errorf("got description %q, want %q", got.Description, newDesc)
	}
}

func TestReposStore_Delete(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())

	u := createTestUser(t, usersStore, "deleteowner")
	r := createTestRepo(t, reposStore, u.ID, "deleterepo")

	err := reposStore.Delete(context.Background(), r.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	got, _ := reposStore.GetByID(context.Background(), r.ID)
	if got != nil {
		t.Error("expected repo to be deleted")
	}
}

func TestReposStore_ListByOwner(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())

	u := createTestUser(t, usersStore, "listowner")
	createTestRepo(t, reposStore, u.ID, "listrepo1")
	createTestRepo(t, reposStore, u.ID, "listrepo2")

	list, err := reposStore.ListByOwner(context.Background(), u.ID, nil)
	if err != nil {
		t.Fatalf("ListByOwner failed: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("got %d repos, want 2", len(list))
	}
}

func TestReposStore_Topics(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())

	u := createTestUser(t, usersStore, "topicsowner")
	r := createTestRepo(t, reposStore, u.ID, "topicsrepo")

	topics := []string{"go", "web", "api"}
	err := reposStore.SetTopics(context.Background(), r.ID, topics)
	if err != nil {
		t.Fatalf("SetTopics failed: %v", err)
	}

	got, err := reposStore.GetTopics(context.Background(), r.ID)
	if err != nil {
		t.Fatalf("GetTopics failed: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("got %d topics, want 3", len(got))
	}
}

func TestReposStore_Languages(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())

	u := createTestUser(t, usersStore, "langsowner")
	r := createTestRepo(t, reposStore, u.ID, "langsrepo")

	langs := map[string]int{"Go": 10000, "JavaScript": 5000}
	err := reposStore.SetLanguages(context.Background(), r.ID, langs)
	if err != nil {
		t.Fatalf("SetLanguages failed: %v", err)
	}

	got, err := reposStore.GetLanguages(context.Background(), r.ID)
	if err != nil {
		t.Fatalf("GetLanguages failed: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("got %d languages, want 2", len(got))
	}
	if got["Go"] != 10000 {
		t.Errorf("got Go bytes %d, want 10000", got["Go"])
	}
}

func TestReposStore_IncrementStargazers(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())

	u := createTestUser(t, usersStore, "starsowner")
	r := createTestRepo(t, reposStore, u.ID, "starsrepo")

	err := reposStore.IncrementStargazers(context.Background(), r.ID, 5)
	if err != nil {
		t.Fatalf("IncrementStargazers failed: %v", err)
	}

	got, _ := reposStore.GetByID(context.Background(), r.ID)
	if got.StargazersCount != 5 {
		t.Errorf("got stargazers_count %d, want 5", got.StargazersCount)
	}
}

func TestReposStore_IncrementOpenIssues(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())

	u := createTestUser(t, usersStore, "issuesowner")
	r := createTestRepo(t, reposStore, u.ID, "issuesrepo")

	err := reposStore.IncrementOpenIssues(context.Background(), r.ID, 3)
	if err != nil {
		t.Fatalf("IncrementOpenIssues failed: %v", err)
	}

	got, _ := reposStore.GetByID(context.Background(), r.ID)
	if got.OpenIssuesCount != 3 {
		t.Errorf("got open_issues_count %d, want 3", got.OpenIssuesCount)
	}
}

// ensure users.User is available
var _ = (*users.User)(nil)
