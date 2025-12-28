package search_test

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"github.com/go-mizu/blueprints/githome/feature/issues"
	"github.com/go-mizu/blueprints/githome/feature/labels"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/search"
	"github.com/go-mizu/blueprints/githome/feature/users"
	"github.com/go-mizu/blueprints/githome/store/duckdb"
)

func setupTestService(t *testing.T) (*search.Service, *duckdb.Store, func()) {
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

	searchStore := duckdb.NewSearchStore(db)
	service := search.NewService(searchStore, "https://api.example.com")

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
		Name:         "Test " + login,
		PasswordHash: "hash",
		Type:         "User",
	}
	if err := store.Users().Create(context.Background(), user); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return user
}

func createTestRepo(t *testing.T, store *duckdb.Store, owner *users.User, name, description string) *repos.Repository {
	t.Helper()
	repo := &repos.Repository{
		Name:          name,
		FullName:      owner.Login + "/" + name,
		OwnerID:       owner.ID,
		OwnerType:     "User",
		Visibility:    "public",
		Description:   description,
		DefaultBranch: "main",
	}
	if err := store.Repos().Create(context.Background(), repo); err != nil {
		t.Fatalf("failed to create test repo: %v", err)
	}
	return repo
}

func createTestIssue(t *testing.T, db *sql.DB, store *duckdb.Store, repo *repos.Repository, creator *users.User, title, body string) {
	t.Helper()
	issuesStore := duckdb.NewIssuesStore(db)
	issuesSvc := issues.NewService(issuesStore, store.Repos(), store.Users(), "https://api.example.com")
	_, err := issuesSvc.Create(context.Background(), repo.FullName[:len(repo.FullName)-len(repo.Name)-1], repo.Name, creator.ID, &issues.CreateIn{
		Title: title,
		Body:  body,
	})
	if err != nil {
		t.Fatalf("failed to create test issue: %v", err)
	}
}

func createTestLabel(t *testing.T, db *sql.DB, store *duckdb.Store, repo *repos.Repository, name, description string) {
	t.Helper()
	labelsStore := duckdb.NewLabelsStore(db)
	labelsSvc := labels.NewService(labelsStore, store.Repos(), duckdb.NewIssuesStore(db), "https://api.example.com")
	ownerLogin := repo.FullName[:len(repo.FullName)-len(repo.Name)-1]
	_, err := labelsSvc.Create(context.Background(), ownerLogin, repo.Name, &labels.CreateIn{
		Name:        name,
		Description: description,
		Color:       "ededed",
	})
	if err != nil {
		t.Fatalf("failed to create test label: %v", err)
	}
}

// Pagination Tests

func TestService_Code_Pagination(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	// With nil opts, should use defaults
	result, err := service.Code(context.Background(), "test", nil)
	if err != nil {
		t.Fatalf("Code failed: %v", err)
	}

	// Should return empty result (no code index)
	if result == nil {
		t.Error("expected result to be non-nil")
	}
}

func TestService_Code_MaxPerPage(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	// Request more than max (100)
	result, err := service.Code(context.Background(), "test", &search.SearchCodeOpts{
		PerPage: 200,
	})
	if err != nil {
		t.Fatalf("Code failed: %v", err)
	}

	// Should not error, just cap at 100
	if result == nil {
		t.Error("expected result to be non-nil")
	}
}

func TestService_Commits_Pagination(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	result, err := service.Commits(context.Background(), "test", nil)
	if err != nil {
		t.Fatalf("Commits failed: %v", err)
	}

	if result == nil {
		t.Error("expected result to be non-nil")
	}
}

func TestService_IssuesAndPullRequests_Pagination(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	result, err := service.IssuesAndPullRequests(context.Background(), "test", nil)
	if err != nil {
		t.Fatalf("IssuesAndPullRequests failed: %v", err)
	}

	if result == nil {
		t.Error("expected result to be non-nil")
	}
}

func TestService_Labels_Pagination(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	result, err := service.Labels(context.Background(), 1, "test", nil)
	if err != nil {
		t.Fatalf("Labels failed: %v", err)
	}

	if result == nil {
		t.Error("expected result to be non-nil")
	}
}

func TestService_Repositories_Pagination(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	result, err := service.Repositories(context.Background(), "test", nil)
	if err != nil {
		t.Fatalf("Repositories failed: %v", err)
	}

	if result == nil {
		t.Error("expected result to be non-nil")
	}
}

func TestService_Topics_Pagination(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	result, err := service.Topics(context.Background(), "test", nil)
	if err != nil {
		t.Fatalf("Topics failed: %v", err)
	}

	if result == nil {
		t.Error("expected result to be non-nil")
	}
}

func TestService_Users_Pagination(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	result, err := service.Users(context.Background(), "test", nil)
	if err != nil {
		t.Fatalf("Users failed: %v", err)
	}

	if result == nil {
		t.Error("expected result to be non-nil")
	}
}

// User Search Tests

func TestService_Users_ReturnsMatches(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	// Create test users
	createTestUser(t, store, "alice", "alice@example.com")
	createTestUser(t, store, "alicia", "alicia@example.com")
	createTestUser(t, store, "bob", "bob@example.com")

	result, err := service.Users(context.Background(), "ali", nil)
	if err != nil {
		t.Fatalf("Users failed: %v", err)
	}

	if result.TotalCount != 2 {
		t.Errorf("expected 2 matches, got %d", result.TotalCount)
	}
	if len(result.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(result.Items))
	}
}

func TestService_Users_CaseInsensitive(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	createTestUser(t, store, "TestUser", "test@example.com")

	result, err := service.Users(context.Background(), "testuser", nil)
	if err != nil {
		t.Fatalf("Users failed: %v", err)
	}

	if result.TotalCount != 1 {
		t.Errorf("expected 1 match, got %d", result.TotalCount)
	}
}

func TestService_Users_EmptyQuery(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	createTestUser(t, store, "user1", "user1@example.com")
	createTestUser(t, store, "user2", "user2@example.com")

	// Empty query should match all
	result, err := service.Users(context.Background(), "", nil)
	if err != nil {
		t.Fatalf("Users failed: %v", err)
	}

	if result.TotalCount < 2 {
		t.Errorf("expected at least 2 matches, got %d", result.TotalCount)
	}
}

func TestService_Users_Pagination_Works(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	// Create 5 users
	for i := 0; i < 5; i++ {
		createTestUser(t, store, "searchuser"+string(rune('a'+i)), "searchuser"+string(rune('a'+i))+"@example.com")
	}

	result, err := service.Users(context.Background(), "searchuser", &search.SearchUsersOpts{
		Page:    1,
		PerPage: 2,
	})
	if err != nil {
		t.Fatalf("Users failed: %v", err)
	}

	if len(result.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(result.Items))
	}
	if result.TotalCount != 5 {
		t.Errorf("expected total count 5, got %d", result.TotalCount)
	}
}

// Repository Search Tests

func TestService_Repositories_ReturnsMatches(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "awesome-project", "An awesome project")
	createTestRepo(t, store, user, "another-awesome", "Another awesome thing")
	createTestRepo(t, store, user, "boring-stuff", "Boring stuff")

	result, err := service.Repositories(context.Background(), "awesome", nil)
	if err != nil {
		t.Fatalf("Repositories failed: %v", err)
	}

	if result.TotalCount != 2 {
		t.Errorf("expected 2 matches, got %d", result.TotalCount)
	}
}

func TestService_Repositories_SearchesDescription(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "myproject", "A fantastic machine learning project")

	result, err := service.Repositories(context.Background(), "machine learning", nil)
	if err != nil {
		t.Fatalf("Repositories failed: %v", err)
	}

	if result.TotalCount != 1 {
		t.Errorf("expected 1 match, got %d", result.TotalCount)
	}
}

func TestService_Repositories_EmptyQuery(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "repo1", "Repo 1")
	createTestRepo(t, store, user, "repo2", "Repo 2")

	result, err := service.Repositories(context.Background(), "", nil)
	if err != nil {
		t.Fatalf("Repositories failed: %v", err)
	}

	if result.TotalCount < 2 {
		t.Errorf("expected at least 2 matches, got %d", result.TotalCount)
	}
}

// Issue Search Tests

func TestService_Issues_ReturnsMatches(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	db, _ := sql.Open("duckdb", "")
	defer db.Close()
	store2, _ := duckdb.New(db)
	_ = store2.Ensure(context.Background())

	user := createTestUser(t, store, "testowner", "owner@example.com")
	repo := createTestRepo(t, store, user, "testrepo", "Test repo")
	createTestIssue(t, store.DB(), store, repo, user, "Bug: login broken", "Users cannot log in")
	createTestIssue(t, store.DB(), store, repo, user, "Feature request", "Add dark mode")
	createTestIssue(t, store.DB(), store, repo, user, "Bug: logout issue", "Logout button missing")

	result, err := service.IssuesAndPullRequests(context.Background(), "bug", nil)
	if err != nil {
		t.Fatalf("IssuesAndPullRequests failed: %v", err)
	}

	if result.TotalCount != 2 {
		t.Errorf("expected 2 matches, got %d", result.TotalCount)
	}
}

func TestService_Issues_SearchesBody(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	repo := createTestRepo(t, store, user, "testrepo", "Test repo")
	createTestIssue(t, store.DB(), store, repo, user, "General issue", "There is a critical security vulnerability here")

	result, err := service.IssuesAndPullRequests(context.Background(), "security vulnerability", nil)
	if err != nil {
		t.Fatalf("IssuesAndPullRequests failed: %v", err)
	}

	if result.TotalCount != 1 {
		t.Errorf("expected 1 match, got %d", result.TotalCount)
	}
}

// Label Search Tests

func TestService_Labels_ReturnsMatches(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	repo := createTestRepo(t, store, user, "testrepo", "Test repo")
	createTestLabel(t, store.DB(), store, repo, "bug", "Something isn't working")
	createTestLabel(t, store.DB(), store, repo, "bugfix", "Fix for a bug")
	createTestLabel(t, store.DB(), store, repo, "enhancement", "New feature")

	result, err := service.Labels(context.Background(), repo.ID, "bug", nil)
	if err != nil {
		t.Fatalf("Labels failed: %v", err)
	}

	if result.TotalCount != 2 {
		t.Errorf("expected 2 matches, got %d", result.TotalCount)
	}
}

func TestService_Labels_SearchesDescription(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	repo := createTestRepo(t, store, user, "testrepo", "Test repo")
	createTestLabel(t, store.DB(), store, repo, "priority-high", "Critical priority item")

	result, err := service.Labels(context.Background(), repo.ID, "critical", nil)
	if err != nil {
		t.Fatalf("Labels failed: %v", err)
	}

	if result.TotalCount != 1 {
		t.Errorf("expected 1 match, got %d", result.TotalCount)
	}
}

// Topics Search Tests

func TestService_Topics_ReturnsMatches(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	repo := createTestRepo(t, store, user, "testrepo", "Test repo")

	// Add topics to the repo
	_, _ = store.Repos().SetTopics(context.Background(), repo.ID, []string{"golang", "go-library", "testing"})

	result, err := service.Topics(context.Background(), "go", nil)
	if err != nil {
		t.Fatalf("Topics failed: %v", err)
	}

	if result.TotalCount < 2 {
		t.Errorf("expected at least 2 matches (golang, go-library), got %d", result.TotalCount)
	}
}

// Code Search Tests (Mock behavior)

func TestService_Code_ReturnsEmpty(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	result, err := service.Code(context.Background(), "function", nil)
	if err != nil {
		t.Fatalf("Code failed: %v", err)
	}

	// Code search requires full-text index, returns empty
	if result.TotalCount != 0 {
		t.Errorf("expected 0 results (no code index), got %d", result.TotalCount)
	}
}

// Commits Search Tests (Mock behavior)

func TestService_Commits_ReturnsEmpty(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	result, err := service.Commits(context.Background(), "fix bug", nil)
	if err != nil {
		t.Fatalf("Commits failed: %v", err)
	}

	// Commit search requires git integration, returns empty
	if result.TotalCount != 0 {
		t.Errorf("expected 0 results (no git integration), got %d", result.TotalCount)
	}
}

// Result Structure Tests

func TestService_Users_ResultStructure(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	createTestUser(t, store, "testuser", "test@example.com")

	result, err := service.Users(context.Background(), "testuser", nil)
	if err != nil {
		t.Fatalf("Users failed: %v", err)
	}

	if len(result.Items) == 0 {
		t.Fatal("expected at least 1 item")
	}

	user := result.Items[0]
	if user.ID == 0 {
		t.Error("expected ID to be set")
	}
	if user.Login == "" {
		t.Error("expected Login to be set")
	}
	if user.Type == "" {
		t.Error("expected Type to be set")
	}
}

func TestService_Repositories_ResultStructure(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo", "A test repo")

	result, err := service.Repositories(context.Background(), "testrepo", nil)
	if err != nil {
		t.Fatalf("Repositories failed: %v", err)
	}

	if len(result.Items) == 0 {
		t.Fatal("expected at least 1 item")
	}

	repo := result.Items[0]
	if repo.ID == 0 {
		t.Error("expected ID to be set")
	}
	if repo.Name == "" {
		t.Error("expected Name to be set")
	}
	if repo.FullName == "" {
		t.Error("expected FullName to be set")
	}
	if repo.Owner == nil {
		t.Error("expected Owner to be set")
	}
}

// IncompleteResults Tests

func TestService_IncompleteResults_False(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	createTestUser(t, store, "testuser", "test@example.com")

	result, err := service.Users(context.Background(), "testuser", nil)
	if err != nil {
		t.Fatalf("Users failed: %v", err)
	}

	if result.IncompleteResults {
		t.Error("expected incomplete_results to be false")
	}
}
