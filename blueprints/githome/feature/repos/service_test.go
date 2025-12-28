package repos_test

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"github.com/go-mizu/blueprints/githome/feature/orgs"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
	"github.com/go-mizu/blueprints/githome/store/duckdb"
)

func setupTestService(t *testing.T) (*repos.Service, *duckdb.Store, func()) {
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

	service := repos.NewService(store.Repos(), store.Users(), nil, "https://api.example.com", "")

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

func createTestOrg(t *testing.T, store *duckdb.Store, login string) *orgs.Organization {
	t.Helper()

	db := store.DB()
	result, err := db.ExecContext(context.Background(),
		`INSERT INTO organizations (login, email, description, created_at, updated_at)
		 VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		login, login+"@example.com", "Test org")
	if err != nil {
		t.Fatalf("failed to create test org: %v", err)
	}
	id, _ := result.LastInsertId()
	return &orgs.Organization{
		ID:    id,
		Login: login,
	}
}

func createTestRepo(t *testing.T, service *repos.Service, ownerID int64, name string) *repos.Repository {
	t.Helper()
	repo, err := service.Create(context.Background(), ownerID, &repos.CreateIn{
		Name:        name,
		Description: "Test repo",
	})
	if err != nil {
		t.Fatalf("failed to create test repo: %v", err)
	}
	return repo
}

// Repository Creation Tests

func TestService_Create_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")

	repo, err := service.Create(context.Background(), user.ID, &repos.CreateIn{
		Name:        "testrepo",
		Description: "A test repository",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if repo.Name != "testrepo" {
		t.Errorf("got name %q, want testrepo", repo.Name)
	}
	if repo.FullName != "testowner/testrepo" {
		t.Errorf("got full_name %q, want testowner/testrepo", repo.FullName)
	}
	if repo.Description != "A test repository" {
		t.Errorf("got description %q, want A test repository", repo.Description)
	}
	if repo.ID == 0 {
		t.Error("expected ID to be assigned")
	}
	if repo.Private {
		t.Error("expected public by default")
	}
	if repo.DefaultBranch != "main" {
		t.Errorf("got default branch %q, want main", repo.DefaultBranch)
	}
	if repo.URL == "" {
		t.Error("expected URL to be populated")
	}
}

func TestService_Create_DuplicateName(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")

	createTestRepo(t, service, user.ID, "testrepo")

	_, err := service.Create(context.Background(), user.ID, &repos.CreateIn{
		Name: "testrepo",
	})
	if err != repos.ErrRepoExists {
		t.Errorf("expected ErrRepoExists, got %v", err)
	}
}

func TestService_Create_VisibilityPublic(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")

	repo, err := service.Create(context.Background(), user.ID, &repos.CreateIn{
		Name:       "testrepo",
		Visibility: "public",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if repo.Private {
		t.Error("expected public visibility")
	}
	if repo.Visibility != "public" {
		t.Errorf("got visibility %q, want public", repo.Visibility)
	}
}

func TestService_Create_VisibilityPrivate(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")

	repo, err := service.Create(context.Background(), user.ID, &repos.CreateIn{
		Name:    "testrepo",
		Private: true,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if !repo.Private {
		t.Error("expected private visibility")
	}
	if repo.Visibility != "private" {
		t.Errorf("got visibility %q, want private", repo.Visibility)
	}
}

func TestService_Create_DefaultFeatures(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")

	repo, err := service.Create(context.Background(), user.ID, &repos.CreateIn{
		Name: "testrepo",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if !repo.HasIssues {
		t.Error("expected HasIssues to be true by default")
	}
	if !repo.HasProjects {
		t.Error("expected HasProjects to be true by default")
	}
	if !repo.HasWiki {
		t.Error("expected HasWiki to be true by default")
	}
	if repo.HasDiscussions {
		t.Error("expected HasDiscussions to be false by default")
	}
}

// Repository Retrieval Tests

func TestService_Get_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	created := createTestRepo(t, service, user.ID, "testrepo")

	repo, err := service.Get(context.Background(), "testowner", "testrepo")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if repo.ID != created.ID {
		t.Errorf("got ID %d, want %d", repo.ID, created.ID)
	}
	if repo.Name != "testrepo" {
		t.Errorf("got name %q, want testrepo", repo.Name)
	}
}

func TestService_Get_NotFound(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	_, err := service.Get(context.Background(), "unknown", "repo")
	if err != repos.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_GetByID_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	created := createTestRepo(t, service, user.ID, "testrepo")

	repo, err := service.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if repo.ID != created.ID {
		t.Errorf("got ID %d, want %d", repo.ID, created.ID)
	}
}

func TestService_GetByID_NotFound(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	_, err := service.GetByID(context.Background(), 99999)
	if err != repos.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_ListForUser(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, service, user.ID, "repo1")
	createTestRepo(t, service, user.ID, "repo2")
	createTestRepo(t, service, user.ID, "repo3")

	list, err := service.ListForUser(context.Background(), "testowner", nil)
	if err != nil {
		t.Fatalf("ListForUser failed: %v", err)
	}

	if len(list) != 3 {
		t.Errorf("expected 3 repos, got %d", len(list))
	}
}

func TestService_ListForUser_Pagination(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	for i := 0; i < 5; i++ {
		createTestRepo(t, service, user.ID, "repo"+string(rune('a'+i)))
	}

	list, err := service.ListForUser(context.Background(), "testowner", &repos.ListOpts{
		Page:    1,
		PerPage: 2,
	})
	if err != nil {
		t.Fatalf("ListForUser failed: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("expected 2 repos, got %d", len(list))
	}
}

func TestService_ListForAuthenticatedUser(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, service, user.ID, "repo1")
	createTestRepo(t, service, user.ID, "repo2")

	list, err := service.ListForAuthenticatedUser(context.Background(), user.ID, nil)
	if err != nil {
		t.Fatalf("ListForAuthenticatedUser failed: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("expected 2 repos, got %d", len(list))
	}
}

// Repository Update Tests

func TestService_Update_Description(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, service, user.ID, "testrepo")

	newDesc := "Updated description"
	updated, err := service.Update(context.Background(), "testowner", "testrepo", &repos.UpdateIn{
		Description: &newDesc,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if updated.Description != "Updated description" {
		t.Errorf("got description %q, want Updated description", updated.Description)
	}
}

func TestService_Update_Rename(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, service, user.ID, "testrepo")

	newName := "newname"
	updated, err := service.Update(context.Background(), "testowner", "testrepo", &repos.UpdateIn{
		Name: &newName,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if updated.Name != "newname" {
		t.Errorf("got name %q, want newname", updated.Name)
	}

	// Old name should not exist
	_, err = service.Get(context.Background(), "testowner", "testrepo")
	if err != repos.ErrNotFound {
		t.Error("old repo name should not exist")
	}
}

func TestService_Update_NotFound(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	newDesc := "Updated description"
	_, err := service.Update(context.Background(), "unknown", "repo", &repos.UpdateIn{
		Description: &newDesc,
	})
	if err != repos.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_Delete_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, service, user.ID, "testrepo")

	err := service.Delete(context.Background(), "testowner", "testrepo")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deleted
	_, err = service.Get(context.Background(), "testowner", "testrepo")
	if err != repos.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestService_Delete_NotFound(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	err := service.Delete(context.Background(), "unknown", "repo")
	if err != repos.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// Topics Tests

func TestService_ListTopics(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	repo := createTestRepo(t, service, user.ID, "testrepo")

	// Set topics via store
	_ = store.Repos().SetTopics(context.Background(), repo.ID, []string{"go", "api", "rest"})

	topics, err := service.ListTopics(context.Background(), "testowner", "testrepo")
	if err != nil {
		t.Fatalf("ListTopics failed: %v", err)
	}

	if len(topics) != 3 {
		t.Errorf("expected 3 topics, got %d", len(topics))
	}
}

func TestService_ReplaceTopics(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, service, user.ID, "testrepo")

	topics, err := service.ReplaceTopics(context.Background(), "testowner", "testrepo", []string{"go", "api"})
	if err != nil {
		t.Fatalf("ReplaceTopics failed: %v", err)
	}

	if len(topics) != 2 {
		t.Errorf("expected 2 topics, got %d", len(topics))
	}

	// Verify persisted
	stored, _ := service.ListTopics(context.Background(), "testowner", "testrepo")
	if len(stored) != 2 {
		t.Errorf("expected 2 stored topics, got %d", len(stored))
	}
}

// Counter Management Tests

func TestService_IncrementOpenIssues(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	repo := createTestRepo(t, service, user.ID, "testrepo")

	err := service.IncrementOpenIssues(context.Background(), repo.ID, 1)
	if err != nil {
		t.Fatalf("IncrementOpenIssues failed: %v", err)
	}

	updated, _ := service.Get(context.Background(), "testowner", "testrepo")
	if updated.OpenIssuesCount != 1 {
		t.Errorf("expected open_issues_count 1, got %d", updated.OpenIssuesCount)
	}

	// Increment again
	_ = service.IncrementOpenIssues(context.Background(), repo.ID, 2)
	updated, _ = service.Get(context.Background(), "testowner", "testrepo")
	if updated.OpenIssuesCount != 3 {
		t.Errorf("expected open_issues_count 3, got %d", updated.OpenIssuesCount)
	}

	// Decrement
	_ = service.IncrementOpenIssues(context.Background(), repo.ID, -1)
	updated, _ = service.Get(context.Background(), "testowner", "testrepo")
	if updated.OpenIssuesCount != 2 {
		t.Errorf("expected open_issues_count 2, got %d", updated.OpenIssuesCount)
	}
}

func TestService_IncrementStargazers(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	repo := createTestRepo(t, service, user.ID, "testrepo")

	err := service.IncrementStargazers(context.Background(), repo.ID, 1)
	if err != nil {
		t.Fatalf("IncrementStargazers failed: %v", err)
	}

	updated, _ := service.Get(context.Background(), "testowner", "testrepo")
	if updated.StargazersCount != 1 {
		t.Errorf("expected stargazers_count 1, got %d", updated.StargazersCount)
	}
}

func TestService_IncrementWatchers(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	repo := createTestRepo(t, service, user.ID, "testrepo")

	err := service.IncrementWatchers(context.Background(), repo.ID, 1)
	if err != nil {
		t.Fatalf("IncrementWatchers failed: %v", err)
	}

	updated, _ := service.Get(context.Background(), "testowner", "testrepo")
	if updated.WatchersCount != 1 {
		t.Errorf("expected watchers_count 1, got %d", updated.WatchersCount)
	}
}

func TestService_IncrementForks(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	repo := createTestRepo(t, service, user.ID, "testrepo")

	err := service.IncrementForks(context.Background(), repo.ID, 1)
	if err != nil {
		t.Fatalf("IncrementForks failed: %v", err)
	}

	updated, _ := service.Get(context.Background(), "testowner", "testrepo")
	if updated.ForksCount != 1 {
		t.Errorf("expected forks_count 1, got %d", updated.ForksCount)
	}
}

// URL Population Tests

func TestService_PopulateURLs(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	repo := createTestRepo(t, service, user.ID, "testrepo")

	if repo.URL != "https://api.example.com/api/v3/repos/testowner/testrepo" {
		t.Errorf("unexpected URL: %s", repo.URL)
	}
	if repo.HTMLURL != "https://api.example.com/testowner/testrepo" {
		t.Errorf("unexpected HTMLURL: %s", repo.HTMLURL)
	}
	if repo.IssuesURL == "" {
		t.Error("expected IssuesURL to be set")
	}
	if repo.PullsURL == "" {
		t.Error("expected PullsURL to be set")
	}
	if repo.NodeID == "" {
		t.Error("expected NodeID to be set")
	}
}

// Integration Test - Multiple Repos Isolation

func TestService_MultipleRepos(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user1 := createTestUser(t, store, "user1", "user1@example.com")
	user2 := createTestUser(t, store, "user2", "user2@example.com")

	repo1 := createTestRepo(t, service, user1.ID, "repo1")
	repo2 := createTestRepo(t, service, user2.ID, "repo2")

	// Can create same-named repo for different owners
	repo1Again, err := service.Create(context.Background(), user2.ID, &repos.CreateIn{
		Name: "repo1",
	})
	if err != nil {
		t.Fatalf("Should allow same name for different owner: %v", err)
	}

	if repo1Again.FullName != "user2/repo1" {
		t.Errorf("got full_name %q, want user2/repo1", repo1Again.FullName)
	}

	// List for each user should be isolated
	list1, _ := service.ListForUser(context.Background(), "user1", nil)
	list2, _ := service.ListForUser(context.Background(), "user2", nil)

	if len(list1) != 1 {
		t.Errorf("user1 should have 1 repo, got %d", len(list1))
	}
	if len(list2) != 2 {
		t.Errorf("user2 should have 2 repos, got %d", len(list2))
	}

	// Verify different IDs
	if repo1.ID == repo2.ID {
		t.Error("repos should have different IDs")
	}
}
