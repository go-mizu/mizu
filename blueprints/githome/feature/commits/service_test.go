package commits_test

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"github.com/go-mizu/blueprints/githome/feature/commits"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
	"github.com/go-mizu/blueprints/githome/store/duckdb"
)

func setupTestService(t *testing.T) (*commits.Service, *duckdb.Store, func()) {
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

	commitsStore := duckdb.NewCommitsStore(db)
	service := commits.NewService(commitsStore, store.Repos(), store.Users(), "https://api.example.com", "")

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

func createTestRepo(t *testing.T, store *duckdb.Store, owner *users.User, name string) *repos.Repository {
	t.Helper()
	repo := &repos.Repository{
		Name:          name,
		FullName:      owner.Login + "/" + name,
		OwnerID:       owner.ID,
		OwnerType:     "User",
		Visibility:    "public",
		DefaultBranch: "main",
	}
	if err := store.Repos().Create(context.Background(), repo); err != nil {
		t.Fatalf("failed to create test repo: %v", err)
	}
	return repo
}

// Commit Status Tests (Production Ready)

func TestService_CreateStatus_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")

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
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")

	_, err := service.CreateStatus(context.Background(), "unknown", "repo", "abc123", user.ID, &commits.CreateStatusIn{
		State: "success",
	})
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

func TestService_CreateStatus_UserNotFound(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")

	_, err := service.CreateStatus(context.Background(), "testowner", "testrepo", "abc123", 99999, &commits.CreateStatusIn{
		State: "success",
	})
	if err != users.ErrNotFound {
		t.Errorf("expected users.ErrNotFound, got %v", err)
	}
}

func TestService_CreateStatus_DefaultContext(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")

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
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")

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
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")

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
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")

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
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	_, err := service.ListStatuses(context.Background(), "unknown", "repo", "abc123", nil)
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

func TestService_GetCombinedStatus_Pending(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")

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
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")

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
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")

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
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")

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
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")

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
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	_, err := service.GetCombinedStatus(context.Background(), "unknown", "repo", "abc123")
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

func TestService_GetCombinedStatus_URLsPopulated(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")

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
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")

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
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	_, err := service.List(context.Background(), "unknown", "repo", nil)
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

func TestService_Get_ReturnsPlaceholder(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")

	commit, err := service.Get(context.Background(), "testowner", "testrepo", "abc123")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if commit.SHA != "abc123" {
		t.Errorf("expected SHA 'abc123', got %q", commit.SHA)
	}
	if commit.Commit == nil {
		t.Error("expected commit data to be set")
	}
	if commit.URL == "" {
		t.Error("expected URL to be populated")
	}
}

func TestService_Get_RepoNotFound(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	_, err := service.Get(context.Background(), "unknown", "repo", "abc123")
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

func TestService_Compare_ReturnsEmpty(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")

	comparison, err := service.Compare(context.Background(), "testowner", "testrepo", "main", "feature")
	if err != nil {
		t.Fatalf("Compare failed: %v", err)
	}

	// Mock implementation returns empty comparison
	if len(comparison.Commits) != 0 {
		t.Errorf("expected 0 commits (mock), got %d", len(comparison.Commits))
	}
	if comparison.URL == "" {
		t.Error("expected URL to be populated")
	}
}

func TestService_Compare_RepoNotFound(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	_, err := service.Compare(context.Background(), "unknown", "repo", "main", "feature")
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

func TestService_ListBranchesForHead_ReturnsEmpty(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")

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
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	_, err := service.ListBranchesForHead(context.Background(), "unknown", "repo", "abc123")
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

func TestService_ListPullsForCommit_ReturnsEmpty(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")

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
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	_, err := service.ListPullsForCommit(context.Background(), "unknown", "repo", "abc123", nil)
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

// URL Population Tests

func TestService_PopulateURLs(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")

	commit, _ := service.Get(context.Background(), "testowner", "testrepo", "abc123")

	expectedURL := "https://api.example.com/api/v3/repos/testowner/testrepo/commits/abc123"
	if commit.URL != expectedURL {
		t.Errorf("expected URL %q, got %q", expectedURL, commit.URL)
	}

	expectedHTMLURL := "https://api.example.com/testowner/testrepo/commit/abc123"
	if commit.HTMLURL != expectedHTMLURL {
		t.Errorf("expected HTMLURL %q, got %q", expectedHTMLURL, commit.HTMLURL)
	}

	if commit.CommentsURL == "" {
		t.Error("expected CommentsURL to be populated")
	}
}

func TestService_PopulateStatusURLs(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")

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

// Different SHA Tests - Ensure statuses are scoped to SHA

func TestService_Statuses_ScopedToSHA(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")

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
