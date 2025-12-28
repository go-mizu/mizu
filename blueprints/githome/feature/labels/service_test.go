package labels_test

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"github.com/go-mizu/blueprints/githome/feature/labels"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
	"github.com/go-mizu/blueprints/githome/store/duckdb"
)

func setupTestService(t *testing.T) (*labels.Service, *duckdb.UsersStore, *duckdb.ReposStore, func()) {
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

	usersStore := duckdb.NewUsersStore(db)
	reposStore := duckdb.NewReposStore(db)
	labelsStore := duckdb.NewLabelsStore(db)
	issuesStore := duckdb.NewIssuesStore(db)
	milestonesStore := duckdb.NewMilestonesStore(db)
	service := labels.NewService(labelsStore, reposStore, issuesStore, milestonesStore, "https://api.example.com")

	cleanup := func() {
		store.Close()
	}

	return service, usersStore, reposStore, cleanup
}

func createTestUser(t *testing.T, usersStore *duckdb.UsersStore, login, email string) *users.User {
	t.Helper()
	user := &users.User{
		Login:        login,
		Email:        email,
		Name:         "Test User",
		PasswordHash: "hash",
		Type:         "User",
	}
	if err := usersStore.Create(context.Background(), user); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return user
}

func createTestRepo(t *testing.T, reposStore *duckdb.ReposStore, owner *users.User, name string) *repos.Repository {
	t.Helper()
	repo := &repos.Repository{
		Name:          name,
		FullName:      owner.Login + "/" + name,
		OwnerID:       owner.ID,
		OwnerType:     "User",
		Visibility:    "public",
		DefaultBranch: "main",
	}
	if err := reposStore.Create(context.Background(), repo); err != nil {
		t.Fatalf("failed to create test repo: %v", err)
	}
	return repo
}

func createTestLabel(t *testing.T, service *labels.Service, owner, repo, name, color string) *labels.Label {
	t.Helper()
	label, err := service.Create(context.Background(), owner, repo, &labels.CreateIn{
		Name:        name,
		Color:       color,
		Description: "Test label",
	})
	if err != nil {
		t.Fatalf("failed to create test label: %v", err)
	}
	return label
}

// Label Creation Tests

func TestService_Create_Success(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, usersStore, "owner", "owner@example.com")
	createTestRepo(t, reposStore, owner, "testrepo")

	label, err := service.Create(context.Background(), "owner", "testrepo", &labels.CreateIn{
		Name:        "bug",
		Color:       "d73a4a",
		Description: "Something isn't working",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if label.Name != "bug" {
		t.Errorf("got name %q, want bug", label.Name)
	}
	if label.Color != "d73a4a" {
		t.Errorf("got color %q, want d73a4a", label.Color)
	}
	if label.Description != "Something isn't working" {
		t.Errorf("got description %q, want Something isn't working", label.Description)
	}
	if label.ID == 0 {
		t.Error("expected ID to be assigned")
	}
	if label.URL == "" {
		t.Error("expected URL to be populated")
	}
}

func TestService_Create_DuplicateName(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, usersStore, "owner", "owner@example.com")
	createTestRepo(t, reposStore, owner, "testrepo")

	createTestLabel(t, service, "owner", "testrepo", "bug", "d73a4a")

	_, err := service.Create(context.Background(), "owner", "testrepo", &labels.CreateIn{
		Name:  "bug",
		Color: "ffffff",
	})
	if err != labels.ErrLabelExists {
		t.Errorf("expected ErrLabelExists, got %v", err)
	}
}

func TestService_Create_RepoNotFound(t *testing.T) {
	service, _, _, cleanup := setupTestService(t)
	defer cleanup()

	_, err := service.Create(context.Background(), "unknown", "repo", &labels.CreateIn{
		Name:  "bug",
		Color: "d73a4a",
	})
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

// Label Retrieval Tests

func TestService_Get_Success(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, usersStore, "owner", "owner@example.com")
	createTestRepo(t, reposStore, owner, "testrepo")
	createTestLabel(t, service, "owner", "testrepo", "bug", "d73a4a")

	label, err := service.Get(context.Background(), "owner", "testrepo", "bug")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if label.Name != "bug" {
		t.Errorf("got name %q, want bug", label.Name)
	}
}

func TestService_Get_NotFound(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, usersStore, "owner", "owner@example.com")
	createTestRepo(t, reposStore, owner, "testrepo")

	_, err := service.Get(context.Background(), "owner", "testrepo", "nonexistent")
	if err != labels.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_List(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, usersStore, "owner", "owner@example.com")
	createTestRepo(t, reposStore, owner, "testrepo")
	createTestLabel(t, service, "owner", "testrepo", "bug", "d73a4a")
	createTestLabel(t, service, "owner", "testrepo", "enhancement", "a2eeef")
	createTestLabel(t, service, "owner", "testrepo", "documentation", "0075ca")

	list, err := service.List(context.Background(), "owner", "testrepo", nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != 3 {
		t.Errorf("expected 3 labels, got %d", len(list))
	}
}

func TestService_List_Pagination(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, usersStore, "owner", "owner@example.com")
	createTestRepo(t, reposStore, owner, "testrepo")

	for i := 0; i < 5; i++ {
		createTestLabel(t, service, "owner", "testrepo", "label"+string(rune('a'+i)), "ffffff")
	}

	list, err := service.List(context.Background(), "owner", "testrepo", &labels.ListOpts{
		Page:    1,
		PerPage: 2,
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("expected 2 labels, got %d", len(list))
	}
}

// Label Update Tests

func TestService_Update_Name(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, usersStore, "owner", "owner@example.com")
	createTestRepo(t, reposStore, owner, "testrepo")
	createTestLabel(t, service, "owner", "testrepo", "bug", "d73a4a")

	newName := "bugfix"
	updated, err := service.Update(context.Background(), "owner", "testrepo", "bug", &labels.UpdateIn{
		NewName: &newName,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if updated.Name != "bugfix" {
		t.Errorf("got name %q, want bugfix", updated.Name)
	}

	// Old name should not exist
	_, err = service.Get(context.Background(), "owner", "testrepo", "bug")
	if err != labels.ErrNotFound {
		t.Error("old label name should not exist")
	}
}

func TestService_Update_Color(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, usersStore, "owner", "owner@example.com")
	createTestRepo(t, reposStore, owner, "testrepo")
	createTestLabel(t, service, "owner", "testrepo", "bug", "d73a4a")

	newColor := "ff0000"
	updated, err := service.Update(context.Background(), "owner", "testrepo", "bug", &labels.UpdateIn{
		Color: &newColor,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if updated.Color != "ff0000" {
		t.Errorf("got color %q, want ff0000", updated.Color)
	}
}

func TestService_Update_Description(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, usersStore, "owner", "owner@example.com")
	createTestRepo(t, reposStore, owner, "testrepo")
	createTestLabel(t, service, "owner", "testrepo", "bug", "d73a4a")

	newDesc := "Updated description"
	updated, err := service.Update(context.Background(), "owner", "testrepo", "bug", &labels.UpdateIn{
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
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, usersStore, "owner", "owner@example.com")
	createTestRepo(t, reposStore, owner, "testrepo")

	newName := "newname"
	_, err := service.Update(context.Background(), "owner", "testrepo", "nonexistent", &labels.UpdateIn{
		NewName: &newName,
	})
	if err != labels.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// Label Delete Tests

func TestService_Delete_Success(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, usersStore, "owner", "owner@example.com")
	createTestRepo(t, reposStore, owner, "testrepo")
	createTestLabel(t, service, "owner", "testrepo", "bug", "d73a4a")

	err := service.Delete(context.Background(), "owner", "testrepo", "bug")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deleted
	_, err = service.Get(context.Background(), "owner", "testrepo", "bug")
	if err != labels.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestService_Delete_NotFound(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, usersStore, "owner", "owner@example.com")
	createTestRepo(t, reposStore, owner, "testrepo")

	err := service.Delete(context.Background(), "owner", "testrepo", "nonexistent")
	if err != labels.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// URL Population Tests

func TestService_PopulateURLs(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, usersStore, "owner", "owner@example.com")
	createTestRepo(t, reposStore, owner, "testrepo")
	label := createTestLabel(t, service, "owner", "testrepo", "bug", "d73a4a")

	if label.URL != "https://api.example.com/api/v3/repos/owner/testrepo/labels/bug" {
		t.Errorf("unexpected URL: %s", label.URL)
	}
	if label.NodeID == "" {
		t.Error("expected NodeID to be set")
	}
}

// Integration Test - Labels Across Repos

func TestService_LabelsAcrossRepos(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, usersStore, "owner", "owner@example.com")
	createTestRepo(t, reposStore, owner, "repo1")
	createTestRepo(t, reposStore, owner, "repo2")

	// Same label name in different repos should work
	label1 := createTestLabel(t, service, "owner", "repo1", "bug", "d73a4a")
	label2 := createTestLabel(t, service, "owner", "repo2", "bug", "ff0000")

	if label1.ID == label2.ID {
		t.Error("labels in different repos should have different IDs")
	}

	// Each repo should have its own labels
	list1, _ := service.List(context.Background(), "owner", "repo1", nil)
	list2, _ := service.List(context.Background(), "owner", "repo2", nil)

	if len(list1) != 1 {
		t.Errorf("repo1 should have 1 label, got %d", len(list1))
	}
	if len(list2) != 1 {
		t.Errorf("repo2 should have 1 label, got %d", len(list2))
	}
}
