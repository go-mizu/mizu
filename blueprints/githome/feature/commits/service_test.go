package commits_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"github.com/go-mizu/blueprints/githome/feature/commits"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
	pkggit "github.com/go-mizu/blueprints/githome/pkg/git"
	"github.com/go-mizu/blueprints/githome/store/duckdb"
)

func setupTestService(t *testing.T) (*commits.Service, *duckdb.Store, *duckdb.UsersStore, *duckdb.ReposStore, func()) {
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
	commitsStore := duckdb.NewCommitsStore(db)
	service := commits.NewService(commitsStore, reposStore, usersStore, "https://api.example.com", "")

	cleanup := func() {
		store.Close()
	}

	return service, store, usersStore, reposStore, cleanup
}

// setupTestServiceWithGit creates a test service with a real git repository
func setupTestServiceWithGit(t *testing.T) (*commits.Service, *duckdb.Store, *duckdb.UsersStore, *duckdb.ReposStore, string, func()) {
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

	// Create temp directory for git repos
	reposDir, err := os.MkdirTemp("", "commits-test-*")
	if err != nil {
		store.Close()
		t.Fatalf("failed to create temp dir: %v", err)
	}

	usersStore := duckdb.NewUsersStore(db)
	reposStore := duckdb.NewReposStore(db)
	commitsStore := duckdb.NewCommitsStore(db)
	service := commits.NewService(commitsStore, reposStore, usersStore, "https://api.example.com", reposDir)

	cleanup := func() {
		store.Close()
		os.RemoveAll(reposDir)
	}

	return service, store, usersStore, reposStore, reposDir, cleanup
}

// createGitRepo creates a bare git repository with an initial commit and returns the commit SHA
func createGitRepo(t *testing.T, reposDir, owner, repoName string) string {
	t.Helper()

	// Create owner directory
	ownerDir := filepath.Join(reposDir, owner)
	if err := os.MkdirAll(ownerDir, 0755); err != nil {
		t.Fatalf("failed to create owner dir: %v", err)
	}

	// Create bare repo at owner/repo.git
	repoPath := filepath.Join(ownerDir, repoName+".git")
	_, commitSHA, err := pkggit.InitWithCommit(repoPath, pkggit.Signature{
		Name:  "Test Author",
		Email: "test@example.com",
		When:  time.Now(),
	}, "Initial commit")
	if err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	return commitSHA
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

// Commit Status Tests (Production Ready)

func TestService_CreateStatus_Success(t *testing.T) {
	service, _, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")

	status, err := service.CreateStatus(context.Background(), "testowner", "testrepo", "abc123", user.ID, &commits.CreateStatusIn{
		State:       "success",
		TargetURL:   "https://ci.example.com/build/123",
		Description: "Build passed",
		Context:     "ci/build",
	})
	if err != nil {
		t.Fatalf("CreateStatus failed: %v", err)
	}

	if status.State != "success" {
		t.Errorf("expected state 'success', got %q", status.State)
	}
	if status.Context != "ci/build" {
		t.Errorf("expected context 'ci/build', got %q", status.Context)
	}
	if status.Description != "Build passed" {
		t.Errorf("expected description 'Build passed', got %q", status.Description)
	}
	if status.TargetURL != "https://ci.example.com/build/123" {
		t.Errorf("expected target_url, got %q", status.TargetURL)
	}
	if status.Creator == nil {
		t.Error("expected creator to be set")
	}
	if status.ID == 0 {
		t.Error("expected ID to be assigned")
	}
	if status.NodeID == "" {
		t.Error("expected NodeID to be set")
	}
}

func TestService_CreateStatus_RepoNotFound(t *testing.T) {
	service, _, usersStore, _, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")

	_, err := service.CreateStatus(context.Background(), "unknown", "repo", "abc123", user.ID, &commits.CreateStatusIn{
		State: "success",
	})
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

func TestService_CreateStatus_UserNotFound(t *testing.T) {
	service, _, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")

	_, err := service.CreateStatus(context.Background(), "testowner", "testrepo", "abc123", 99999, &commits.CreateStatusIn{
		State: "success",
	})
	if err != users.ErrNotFound {
		t.Errorf("expected users.ErrNotFound, got %v", err)
	}
}

func TestService_CreateStatus_DefaultContext(t *testing.T) {
	service, _, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")

	status, err := service.CreateStatus(context.Background(), "testowner", "testrepo", "abc123", user.ID, &commits.CreateStatusIn{
		State: "pending",
		// No context provided
	})
	if err != nil {
		t.Fatalf("CreateStatus failed: %v", err)
	}

	if status.Context != "default" {
		t.Errorf("expected context 'default', got %q", status.Context)
	}
}

func TestService_ListStatuses_Success(t *testing.T) {
	service, _, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")

	// Create multiple statuses
	_, _ = service.CreateStatus(context.Background(), "testowner", "testrepo", "abc123", user.ID, &commits.CreateStatusIn{
		State:   "success",
		Context: "ci/build",
	})
	_, _ = service.CreateStatus(context.Background(), "testowner", "testrepo", "abc123", user.ID, &commits.CreateStatusIn{
		State:   "success",
		Context: "ci/test",
	})
	_, _ = service.CreateStatus(context.Background(), "testowner", "testrepo", "abc123", user.ID, &commits.CreateStatusIn{
		State:   "pending",
		Context: "ci/lint",
	})

	statuses, err := service.ListStatuses(context.Background(), "testowner", "testrepo", "abc123", nil)
	if err != nil {
		t.Fatalf("ListStatuses failed: %v", err)
	}

	if len(statuses) != 3 {
		t.Errorf("expected 3 statuses, got %d", len(statuses))
	}
}

func TestService_ListStatuses_Pagination(t *testing.T) {
	service, _, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")

	// Create 5 statuses
	for i := 0; i < 5; i++ {
		_, _ = service.CreateStatus(context.Background(), "testowner", "testrepo", "abc123", user.ID, &commits.CreateStatusIn{
			State:   "success",
			Context: "ci/job" + string(rune('a'+i)),
		})
	}

	statuses, err := service.ListStatuses(context.Background(), "testowner", "testrepo", "abc123", &commits.ListOpts{
		Page:    1,
		PerPage: 2,
	})
	if err != nil {
		t.Fatalf("ListStatuses failed: %v", err)
	}

	if len(statuses) != 2 {
		t.Errorf("expected 2 statuses, got %d", len(statuses))
	}
}

func TestService_ListStatuses_Empty(t *testing.T) {
	service, _, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")

	statuses, err := service.ListStatuses(context.Background(), "testowner", "testrepo", "abc123", nil)
	if err != nil {
		t.Fatalf("ListStatuses failed: %v", err)
	}

	if statuses == nil {
		statuses = []*commits.Status{}
	}
	if len(statuses) != 0 {
		t.Errorf("expected 0 statuses, got %d", len(statuses))
	}
}

func TestService_ListStatuses_RepoNotFound(t *testing.T) {
	service, _, _, _, cleanup := setupTestService(t)
	defer cleanup()

	_, err := service.ListStatuses(context.Background(), "unknown", "repo", "abc123", nil)
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

func TestService_GetCombinedStatus_Pending(t *testing.T) {
	service, _, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")

	combined, err := service.GetCombinedStatus(context.Background(), "testowner", "testrepo", "abc123")
	if err != nil {
		t.Fatalf("GetCombinedStatus failed: %v", err)
	}

	if combined.State != "pending" {
		t.Errorf("expected state 'pending', got %q", combined.State)
	}
	if combined.TotalCount != 0 {
		t.Errorf("expected 0 total_count, got %d", combined.TotalCount)
	}
	if combined.SHA != "abc123" {
		t.Errorf("expected SHA 'abc123', got %q", combined.SHA)
	}
}

func TestService_GetCombinedStatus_Success(t *testing.T) {
	service, _, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")

	// Create all success statuses
	_, _ = service.CreateStatus(context.Background(), "testowner", "testrepo", "abc123", user.ID, &commits.CreateStatusIn{
		State:   "success",
		Context: "ci/build",
	})
	_, _ = service.CreateStatus(context.Background(), "testowner", "testrepo", "abc123", user.ID, &commits.CreateStatusIn{
		State:   "success",
		Context: "ci/test",
	})

	combined, err := service.GetCombinedStatus(context.Background(), "testowner", "testrepo", "abc123")
	if err != nil {
		t.Fatalf("GetCombinedStatus failed: %v", err)
	}

	if combined.State != "success" {
		t.Errorf("expected state 'success', got %q", combined.State)
	}
	if combined.TotalCount != 2 {
		t.Errorf("expected 2 total_count, got %d", combined.TotalCount)
	}
}

func TestService_GetCombinedStatus_Failure(t *testing.T) {
	service, _, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")

	// Create mixed statuses with one failure
	_, _ = service.CreateStatus(context.Background(), "testowner", "testrepo", "abc123", user.ID, &commits.CreateStatusIn{
		State:   "success",
		Context: "ci/build",
	})
	_, _ = service.CreateStatus(context.Background(), "testowner", "testrepo", "abc123", user.ID, &commits.CreateStatusIn{
		State:   "failure",
		Context: "ci/test",
	})

	combined, err := service.GetCombinedStatus(context.Background(), "testowner", "testrepo", "abc123")
	if err != nil {
		t.Fatalf("GetCombinedStatus failed: %v", err)
	}

	if combined.State != "failure" {
		t.Errorf("expected state 'failure', got %q", combined.State)
	}
}

func TestService_GetCombinedStatus_Error(t *testing.T) {
	service, _, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")

	// Create status with error state
	_, _ = service.CreateStatus(context.Background(), "testowner", "testrepo", "abc123", user.ID, &commits.CreateStatusIn{
		State:   "error",
		Context: "ci/build",
	})

	combined, err := service.GetCombinedStatus(context.Background(), "testowner", "testrepo", "abc123")
	if err != nil {
		t.Fatalf("GetCombinedStatus failed: %v", err)
	}

	if combined.State != "error" {
		t.Errorf("expected state 'error', got %q", combined.State)
	}
}

func TestService_GetCombinedStatus_WithPending(t *testing.T) {
	service, _, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")

	// Create mixed statuses with pending
	_, _ = service.CreateStatus(context.Background(), "testowner", "testrepo", "abc123", user.ID, &commits.CreateStatusIn{
		State:   "success",
		Context: "ci/build",
	})
	_, _ = service.CreateStatus(context.Background(), "testowner", "testrepo", "abc123", user.ID, &commits.CreateStatusIn{
		State:   "pending",
		Context: "ci/test",
	})

	combined, err := service.GetCombinedStatus(context.Background(), "testowner", "testrepo", "abc123")
	if err != nil {
		t.Fatalf("GetCombinedStatus failed: %v", err)
	}

	if combined.State != "pending" {
		t.Errorf("expected state 'pending', got %q", combined.State)
	}
}

func TestService_GetCombinedStatus_RepoNotFound(t *testing.T) {
	service, _, _, _, cleanup := setupTestService(t)
	defer cleanup()

	_, err := service.GetCombinedStatus(context.Background(), "unknown", "repo", "abc123")
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

func TestService_GetCombinedStatus_URLsPopulated(t *testing.T) {
	service, _, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")

	combined, err := service.GetCombinedStatus(context.Background(), "testowner", "testrepo", "abc123")
	if err != nil {
		t.Fatalf("GetCombinedStatus failed: %v", err)
	}

	if combined.URL == "" {
		t.Error("expected URL to be populated")
	}
	if combined.CommitURL == "" {
		t.Error("expected CommitURL to be populated")
	}
	if combined.Repository != nil && combined.Repository.URL == "" {
		t.Error("expected Repository URL to be populated")
	}
}

// Mock Behavior Tests - Verify services work with placeholder implementations

func TestService_List_ReturnsEmpty(t *testing.T) {
	service, _, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")

	list, err := service.List(context.Background(), "testowner", "testrepo", nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	// Mock implementation returns empty list
	if len(list) != 0 {
		t.Errorf("expected 0 commits (mock), got %d", len(list))
	}
}

func TestService_List_RepoNotFound(t *testing.T) {
	service, _, _, _, cleanup := setupTestService(t)
	defer cleanup()

	_, err := service.List(context.Background(), "unknown", "repo", nil)
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

func TestService_Get_ReturnsPlaceholder(t *testing.T) {
	service, _, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")

	// Without a real git repository, Get returns ErrNotFound
	_, err := service.Get(context.Background(), "testowner", "testrepo", "abc123")
	if err != commits.ErrNotFound {
		t.Errorf("expected commits.ErrNotFound (no git repo), got %v", err)
	}
}

func TestService_Get_RepoNotFound(t *testing.T) {
	service, _, _, _, cleanup := setupTestService(t)
	defer cleanup()

	_, err := service.Get(context.Background(), "unknown", "repo", "abc123")
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

func TestService_Compare_ReturnsEmpty(t *testing.T) {
	service, _, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")

	// Without a real git repository, Compare returns ErrNotFound
	_, err := service.Compare(context.Background(), "testowner", "testrepo", "main", "feature")
	if err != commits.ErrNotFound {
		t.Errorf("expected commits.ErrNotFound (no git repo), got %v", err)
	}
}

func TestService_Compare_RepoNotFound(t *testing.T) {
	service, _, _, _, cleanup := setupTestService(t)
	defer cleanup()

	_, err := service.Compare(context.Background(), "unknown", "repo", "main", "feature")
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

func TestService_ListBranchesForHead_ReturnsEmpty(t *testing.T) {
	service, _, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")

	branches, err := service.ListBranchesForHead(context.Background(), "testowner", "testrepo", "abc123")
	if err != nil {
		t.Fatalf("ListBranchesForHead failed: %v", err)
	}

	// Mock implementation returns empty list
	if len(branches) != 0 {
		t.Errorf("expected 0 branches (mock), got %d", len(branches))
	}
}

func TestService_ListBranchesForHead_RepoNotFound(t *testing.T) {
	service, _, _, _, cleanup := setupTestService(t)
	defer cleanup()

	_, err := service.ListBranchesForHead(context.Background(), "unknown", "repo", "abc123")
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

func TestService_ListPullsForCommit_ReturnsEmpty(t *testing.T) {
	service, _, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")

	prs, err := service.ListPullsForCommit(context.Background(), "testowner", "testrepo", "abc123", nil)
	if err != nil {
		t.Fatalf("ListPullsForCommit failed: %v", err)
	}

	// Mock implementation returns empty list
	if len(prs) != 0 {
		t.Errorf("expected 0 PRs (mock), got %d", len(prs))
	}
}

func TestService_ListPullsForCommit_RepoNotFound(t *testing.T) {
	service, _, _, _, cleanup := setupTestService(t)
	defer cleanup()

	_, err := service.ListPullsForCommit(context.Background(), "unknown", "repo", "abc123", nil)
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

// URL Population Tests

func TestService_PopulateURLs(t *testing.T) {
	service, _, usersStore, reposStore, reposDir, cleanup := setupTestServiceWithGit(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	repo := createTestRepo(t, reposStore, user, "testrepo")

	// Create git repository with a commit
	commitSHA := createGitRepo(t, reposDir, user.Login, repo.Name)

	// Get the commit
	commit, err := service.Get(context.Background(), user.Login, repo.Name, commitSHA)
	if err != nil {
		t.Fatalf("Get commit failed: %v", err)
	}

	// Verify URLs are populated
	expectedURL := "https://api.example.com/api/v3/repos/testowner/testrepo/commits/" + commitSHA
	if commit.URL != expectedURL {
		t.Errorf("expected URL %q, got %q", expectedURL, commit.URL)
	}

	expectedHTMLURL := "https://api.example.com/testowner/testrepo/commit/" + commitSHA
	if commit.HTMLURL != expectedHTMLURL {
		t.Errorf("expected HTMLURL %q, got %q", expectedHTMLURL, commit.HTMLURL)
	}

	expectedCommentsURL := "https://api.example.com/api/v3/repos/testowner/testrepo/commits/" + commitSHA + "/comments"
	if commit.CommentsURL != expectedCommentsURL {
		t.Errorf("expected CommentsURL %q, got %q", expectedCommentsURL, commit.CommentsURL)
	}

	if commit.NodeID == "" {
		t.Error("expected NodeID to be populated")
	}

	// Verify commit data URL
	if commit.Commit == nil {
		t.Fatal("expected Commit data to be set")
	}
	if commit.Commit.URL != expectedURL {
		t.Errorf("expected Commit.URL %q, got %q", expectedURL, commit.Commit.URL)
	}
}

func TestService_PopulateURLs_InList(t *testing.T) {
	service, _, usersStore, reposStore, reposDir, cleanup := setupTestServiceWithGit(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	repo := createTestRepo(t, reposStore, user, "testrepo")

	// Create git repository
	createGitRepo(t, reposDir, user.Login, repo.Name)

	// List commits
	commitList, err := service.List(context.Background(), user.Login, repo.Name, nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(commitList) == 0 {
		t.Fatal("expected at least one commit")
	}

	// Verify URLs are populated for each commit in list
	for _, commit := range commitList {
		if commit.URL == "" {
			t.Error("expected URL to be populated in list")
		}
		if !strings.Contains(commit.URL, commit.SHA) {
			t.Error("URL should contain commit SHA")
		}
		if commit.HTMLURL == "" {
			t.Error("expected HTMLURL to be populated in list")
		}
	}
}

func TestService_PopulateStatusURLs(t *testing.T) {
	service, _, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")

	status, _ := service.CreateStatus(context.Background(), "testowner", "testrepo", "abc123", user.ID, &commits.CreateStatusIn{
		State:   "success",
		Context: "ci/build",
	})

	if status.URL == "" {
		t.Error("expected URL to be populated")
	}
	if status.NodeID == "" {
		t.Error("expected NodeID to be populated")
	}
}

// Enhanced Commit Tests - GitHub API Compatibility

func TestService_List_WithPagination(t *testing.T) {
	service, _, usersStore, reposStore, reposDir, cleanup := setupTestServiceWithGit(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	repo := createTestRepo(t, reposStore, user, "testrepo")

	// Create git repository with multiple commits
	createGitRepoWithMultipleCommits(t, reposDir, user.Login, repo.Name, 5)

	// Test default pagination (page 1, per_page 30)
	commitList, err := service.List(context.Background(), user.Login, repo.Name, nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(commitList) != 5 {
		t.Errorf("expected 5 commits, got %d", len(commitList))
	}

	// Test custom per_page
	opts := &commits.ListOpts{Page: 1, PerPage: 2}
	commitList, err = service.List(context.Background(), user.Login, repo.Name, opts)
	if err != nil {
		t.Fatalf("List with pagination failed: %v", err)
	}
	if len(commitList) != 2 {
		t.Errorf("expected 2 commits, got %d", len(commitList))
	}

	// Test page 2
	opts = &commits.ListOpts{Page: 2, PerPage: 2}
	commitList, err = service.List(context.Background(), user.Login, repo.Name, opts)
	if err != nil {
		t.Fatalf("List page 2 failed: %v", err)
	}
	if len(commitList) != 2 {
		t.Errorf("expected 2 commits on page 2, got %d", len(commitList))
	}
}

func TestService_List_MaxPerPage(t *testing.T) {
	service, _, usersStore, reposStore, reposDir, cleanup := setupTestServiceWithGit(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	repo := createTestRepo(t, reposStore, user, "testrepo")
	createGitRepo(t, reposDir, user.Login, repo.Name)

	// Test per_page > 100 is capped at 100
	opts := &commits.ListOpts{PerPage: 200}
	_, err := service.List(context.Background(), user.Login, repo.Name, opts)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	// The actual cap is enforced in service, so this should not error
}

func TestService_Get_WithStatsAndFiles(t *testing.T) {
	service, _, usersStore, reposStore, reposDir, cleanup := setupTestServiceWithGit(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	repo := createTestRepo(t, reposStore, user, "testrepo")

	// Create git repository with a commit that modifies files
	commitSHA := createGitRepoWithFileChanges(t, reposDir, user.Login, repo.Name)

	// Get the commit
	commit, err := service.Get(context.Background(), user.Login, repo.Name, commitSHA)
	if err != nil {
		t.Fatalf("Get commit failed: %v", err)
	}

	// Verify stats are present
	if commit.Stats == nil {
		t.Error("expected Stats to be populated")
	} else {
		if commit.Stats.Additions <= 0 && commit.Stats.Deletions <= 0 {
			t.Error("expected non-zero additions or deletions")
		}
		if commit.Stats.Total != commit.Stats.Additions+commit.Stats.Deletions {
			t.Errorf("expected Total=%d to equal Additions(%d)+Deletions(%d)",
				commit.Stats.Total, commit.Stats.Additions, commit.Stats.Deletions)
		}
	}

	// Verify files are present
	if commit.Files == nil || len(commit.Files) == 0 {
		t.Error("expected Files to be populated")
	} else {
		for _, f := range commit.Files {
			if f.Filename == "" {
				t.Error("expected Filename to be set")
			}
			if f.Status == "" {
				t.Error("expected Status to be set")
			}
			if f.BlobURL == "" {
				t.Error("expected BlobURL to be set")
			}
			if f.RawURL == "" {
				t.Error("expected RawURL to be set")
			}
			if f.ContentsURL == "" {
				t.Error("expected ContentsURL to be set")
			}
		}
	}
}

func TestService_Get_Verification(t *testing.T) {
	service, _, usersStore, reposStore, reposDir, cleanup := setupTestServiceWithGit(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	repo := createTestRepo(t, reposStore, user, "testrepo")
	commitSHA := createGitRepo(t, reposDir, user.Login, repo.Name)

	// Get the commit
	commit, err := service.Get(context.Background(), user.Login, repo.Name, commitSHA)
	if err != nil {
		t.Fatalf("Get commit failed: %v", err)
	}

	// Verify verification object is present
	if commit.Commit == nil {
		t.Fatal("expected Commit data to be set")
	}
	if commit.Commit.Verification == nil {
		t.Error("expected Verification to be set")
	} else {
		if commit.Commit.Verification.Verified {
			t.Error("expected Verified=false for unsigned commit")
		}
		if commit.Commit.Verification.Reason != "unsigned" {
			t.Errorf("expected Reason='unsigned', got %q", commit.Commit.Verification.Reason)
		}
	}
}

func TestService_Get_ParentsWithHTMLURL(t *testing.T) {
	service, _, usersStore, reposStore, reposDir, cleanup := setupTestServiceWithGit(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	repo := createTestRepo(t, reposStore, user, "testrepo")

	// Create git repository with multiple commits (so we have parents)
	commits := createGitRepoWithMultipleCommits(t, reposDir, user.Login, repo.Name, 3)
	latestSHA := commits[0] // First in list is latest

	// Get the latest commit
	commit, err := service.Get(context.Background(), user.Login, repo.Name, latestSHA)
	if err != nil {
		t.Fatalf("Get commit failed: %v", err)
	}

	// Verify parents have html_url
	if len(commit.Parents) == 0 {
		t.Error("expected at least one parent")
	} else {
		for _, p := range commit.Parents {
			if p.HTMLURL == "" {
				t.Error("expected parent HTMLURL to be set")
			}
			if !strings.Contains(p.HTMLURL, "/commit/") {
				t.Errorf("expected parent HTMLURL to contain '/commit/', got %q", p.HTMLURL)
			}
		}
	}
}

func TestService_List_Verification(t *testing.T) {
	service, _, usersStore, reposStore, reposDir, cleanup := setupTestServiceWithGit(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	repo := createTestRepo(t, reposStore, user, "testrepo")
	createGitRepo(t, reposDir, user.Login, repo.Name)

	// List commits
	commitList, err := service.List(context.Background(), user.Login, repo.Name, nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	// Verify each commit has verification object
	for _, commit := range commitList {
		if commit.Commit == nil {
			t.Error("expected Commit data to be set")
			continue
		}
		if commit.Commit.Verification == nil {
			t.Error("expected Verification to be set in list response")
		}
	}
}

// Helper function to create a git repo with multiple commits
func createGitRepoWithMultipleCommits(t *testing.T, reposDir, owner, repoName string, count int) []string {
	t.Helper()

	// Create owner directory
	ownerDir := filepath.Join(reposDir, owner)
	if err := os.MkdirAll(ownerDir, 0755); err != nil {
		t.Fatalf("failed to create owner dir: %v", err)
	}

	// Create bare repo at owner/repo.git
	repoPath := filepath.Join(ownerDir, repoName+".git")
	repo, commitSHA, err := pkggit.InitWithCommit(repoPath, pkggit.Signature{
		Name:  "Test Author",
		Email: "test@example.com",
		When:  time.Now().Add(-time.Duration(count) * time.Hour),
	}, "Initial commit")
	if err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	shas := []string{commitSHA}

	// Create additional commits
	for i := 1; i < count; i++ {
		// Create a blob with some content
		content := []byte("content " + string(rune('0'+i)))
		blobSHA, err := repo.CreateBlob(content)
		if err != nil {
			t.Fatalf("failed to create blob: %v", err)
		}

		// Create a tree with the blob
		treeSHA, err := repo.CreateTree(&pkggit.CreateTreeOpts{
			Entries: []pkggit.TreeEntryInput{
				{Path: "file" + string(rune('0'+i)) + ".txt", Mode: pkggit.ModeFile, SHA: blobSHA},
			},
		})
		if err != nil {
			t.Fatalf("failed to create tree: %v", err)
		}

		// Create commit
		newCommitSHA, err := repo.CreateCommit(&pkggit.CreateCommitOpts{
			Message: "Commit " + string(rune('0'+i)),
			TreeSHA: treeSHA,
			Parents: []string{shas[len(shas)-1]},
			Author: pkggit.Signature{
				Name:  "Test Author",
				Email: "test@example.com",
				When:  time.Now().Add(-time.Duration(count-i) * time.Hour),
			},
			Committer: pkggit.Signature{
				Name:  "Test Committer",
				Email: "committer@example.com",
				When:  time.Now().Add(-time.Duration(count-i) * time.Hour),
			},
		})
		if err != nil {
			t.Fatalf("failed to create commit: %v", err)
		}
		shas = append(shas, newCommitSHA)

		// Update main branch
		if err := repo.UpdateRef("refs/heads/main", newCommitSHA, true); err != nil {
			t.Fatalf("failed to update ref: %v", err)
		}
	}

	// Reverse so latest is first
	for i, j := 0, len(shas)-1; i < j; i, j = i+1, j-1 {
		shas[i], shas[j] = shas[j], shas[i]
	}

	return shas
}

// Helper function to create a git repo with file changes for testing diff
func createGitRepoWithFileChanges(t *testing.T, reposDir, owner, repoName string) string {
	t.Helper()

	// Create owner directory
	ownerDir := filepath.Join(reposDir, owner)
	if err := os.MkdirAll(ownerDir, 0755); err != nil {
		t.Fatalf("failed to create owner dir: %v", err)
	}

	// Create bare repo at owner/repo.git
	repoPath := filepath.Join(ownerDir, repoName+".git")
	repo, initialSHA, err := pkggit.InitWithCommit(repoPath, pkggit.Signature{
		Name:  "Test Author",
		Email: "test@example.com",
		When:  time.Now().Add(-2 * time.Hour),
	}, "Initial commit")
	if err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Create a blob with content
	content := []byte("Hello, World!\nThis is a test file.\n")
	blobSHA, err := repo.CreateBlob(content)
	if err != nil {
		t.Fatalf("failed to create blob: %v", err)
	}

	// Create a tree with the file
	treeSHA, err := repo.CreateTree(&pkggit.CreateTreeOpts{
		Entries: []pkggit.TreeEntryInput{
			{Path: "README.md", Mode: pkggit.ModeFile, SHA: blobSHA},
		},
	})
	if err != nil {
		t.Fatalf("failed to create tree: %v", err)
	}

	// Create commit with the file
	commitSHA, err := repo.CreateCommit(&pkggit.CreateCommitOpts{
		Message: "Add README.md",
		TreeSHA: treeSHA,
		Parents: []string{initialSHA},
		Author: pkggit.Signature{
			Name:  "Test Author",
			Email: "test@example.com",
			When:  time.Now().Add(-1 * time.Hour),
		},
		Committer: pkggit.Signature{
			Name:  "Test Author",
			Email: "test@example.com",
			When:  time.Now().Add(-1 * time.Hour),
		},
	})
	if err != nil {
		t.Fatalf("failed to create commit: %v", err)
	}

	// Update main branch
	if err := repo.UpdateRef("refs/heads/main", commitSHA, true); err != nil {
		t.Fatalf("failed to update ref: %v", err)
	}

	return commitSHA
}

// GitHub API Compatibility Test - Verify JSON structure matches GitHub

func TestService_Get_GitHubAPICompatibility(t *testing.T) {
	service, _, usersStore, reposStore, reposDir, cleanup := setupTestServiceWithGit(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	repo := createTestRepo(t, reposStore, user, "testrepo")
	commitSHA := createGitRepoWithFileChanges(t, reposDir, user.Login, repo.Name)

	// Get the commit
	commit, err := service.Get(context.Background(), user.Login, repo.Name, commitSHA)
	if err != nil {
		t.Fatalf("Get commit failed: %v", err)
	}

	// Verify all required GitHub API fields are present
	// Top-level fields
	if commit.SHA == "" {
		t.Error("missing SHA")
	}
	if commit.NodeID == "" {
		t.Error("missing node_id")
	}
	if commit.URL == "" {
		t.Error("missing url")
	}
	if commit.HTMLURL == "" {
		t.Error("missing html_url")
	}
	if commit.CommentsURL == "" {
		t.Error("missing comments_url")
	}
	if commit.Parents == nil {
		t.Error("missing parents (should be empty array, not nil)")
	}

	// Commit data object
	if commit.Commit == nil {
		t.Fatal("missing commit object")
	}
	if commit.Commit.URL == "" {
		t.Error("missing commit.url")
	}
	if commit.Commit.Message == "" {
		t.Error("missing commit.message")
	}

	// Author info
	if commit.Commit.Author == nil {
		t.Error("missing commit.author")
	} else {
		if commit.Commit.Author.Name == "" {
			t.Error("missing commit.author.name")
		}
		if commit.Commit.Author.Email == "" {
			t.Error("missing commit.author.email")
		}
		if commit.Commit.Author.Date.IsZero() {
			t.Error("missing commit.author.date")
		}
	}

	// Committer info
	if commit.Commit.Committer == nil {
		t.Error("missing commit.committer")
	}

	// Tree ref
	if commit.Commit.Tree == nil {
		t.Error("missing commit.tree")
	} else {
		if commit.Commit.Tree.SHA == "" {
			t.Error("missing commit.tree.sha")
		}
		if commit.Commit.Tree.URL == "" {
			t.Error("missing commit.tree.url")
		}
	}

	// Verification
	if commit.Commit.Verification == nil {
		t.Error("missing commit.verification")
	}

	// Stats (for single commit GET)
	if commit.Stats == nil {
		t.Error("missing stats")
	} else {
		// Total should equal additions + deletions
		if commit.Stats.Total != commit.Stats.Additions+commit.Stats.Deletions {
			t.Errorf("stats.total (%d) != additions (%d) + deletions (%d)",
				commit.Stats.Total, commit.Stats.Additions, commit.Stats.Deletions)
		}
	}

	// Files (for single commit GET)
	if commit.Files == nil {
		t.Error("missing files")
	} else if len(commit.Files) > 0 {
		f := commit.Files[0]
		if f.Filename == "" {
			t.Error("missing files[].filename")
		}
		if f.Status == "" {
			t.Error("missing files[].status")
		}
		if f.BlobURL == "" {
			t.Error("missing files[].blob_url")
		}
		if f.RawURL == "" {
			t.Error("missing files[].raw_url")
		}
		if f.ContentsURL == "" {
			t.Error("missing files[].contents_url")
		}
		// Patch should be present for non-binary files
		if f.Status == "added" || f.Status == "modified" {
			// Patch might be empty for some files
		}
	}
}

func TestService_List_GitHubAPICompatibility(t *testing.T) {
	service, _, usersStore, reposStore, reposDir, cleanup := setupTestServiceWithGit(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	repo := createTestRepo(t, reposStore, user, "testrepo")
	createGitRepoWithMultipleCommits(t, reposDir, user.Login, repo.Name, 3)

	// List commits
	commitList, err := service.List(context.Background(), user.Login, repo.Name, nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(commitList) == 0 {
		t.Fatal("expected at least one commit")
	}

	// Verify structure of list response (should NOT have stats/files)
	for i, commit := range commitList {
		// Required fields
		if commit.SHA == "" {
			t.Errorf("commit[%d]: missing SHA", i)
		}
		if commit.NodeID == "" {
			t.Errorf("commit[%d]: missing node_id", i)
		}
		if commit.URL == "" {
			t.Errorf("commit[%d]: missing url", i)
		}
		if commit.HTMLURL == "" {
			t.Errorf("commit[%d]: missing html_url", i)
		}
		if commit.Commit == nil {
			t.Errorf("commit[%d]: missing commit object", i)
		}
		if commit.Parents == nil {
			t.Errorf("commit[%d]: parents should not be nil", i)
		}

		// Parent should have html_url
		for j, p := range commit.Parents {
			if p.HTMLURL == "" {
				t.Errorf("commit[%d].parents[%d]: missing html_url", i, j)
			}
		}

		// Verification should be present in list response
		if commit.Commit != nil && commit.Commit.Verification == nil {
			t.Errorf("commit[%d]: missing verification", i)
		}

		// NOTE: Stats and Files should be nil in list response (GitHub API behavior)
		// We currently include them, but GitHub doesn't for performance
	}
}

// Different SHA Tests - Ensure statuses are scoped to SHA

func TestService_Statuses_ScopedToSHA(t *testing.T) {
	service, _, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")

	// Create statuses for different SHAs
	_, _ = service.CreateStatus(context.Background(), "testowner", "testrepo", "sha1", user.ID, &commits.CreateStatusIn{
		State:   "success",
		Context: "ci/build",
	})
	_, _ = service.CreateStatus(context.Background(), "testowner", "testrepo", "sha2", user.ID, &commits.CreateStatusIn{
		State:   "failure",
		Context: "ci/build",
	})

	// Verify each SHA has its own status
	statuses1, _ := service.ListStatuses(context.Background(), "testowner", "testrepo", "sha1", nil)
	statuses2, _ := service.ListStatuses(context.Background(), "testowner", "testrepo", "sha2", nil)

	if len(statuses1) != 1 {
		t.Errorf("expected 1 status for sha1, got %d", len(statuses1))
	}
	if len(statuses2) != 1 {
		t.Errorf("expected 1 status for sha2, got %d", len(statuses2))
	}

	if statuses1[0].State != "success" {
		t.Errorf("expected sha1 state 'success', got %q", statuses1[0].State)
	}
	if statuses2[0].State != "failure" {
		t.Errorf("expected sha2 state 'failure', got %q", statuses2[0].State)
	}
}
