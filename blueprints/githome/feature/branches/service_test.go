package branches_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"github.com/go-mizu/blueprints/githome/feature/branches"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
	pkggit "github.com/go-mizu/blueprints/githome/pkg/git"
	"github.com/go-mizu/blueprints/githome/store/duckdb"
)

func setupTestService(t *testing.T) (*branches.Service, *duckdb.Store, func()) {
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

	branchesStore := duckdb.NewBranchesStore(db)
	service := branches.NewService(branchesStore, store.Repos(), "https://api.example.com", "")

	cleanup := func() {
		store.Close()
	}

	return service, store, cleanup
}

// setupTestServiceWithGit creates a test service with a real git repository
func setupTestServiceWithGit(t *testing.T) (*branches.Service, *duckdb.Store, string, func()) {
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
	reposDir, err := os.MkdirTemp("", "branches-test-*")
	if err != nil {
		store.Close()
		t.Fatalf("failed to create temp dir: %v", err)
	}

	branchesStore := duckdb.NewBranchesStore(db)
	service := branches.NewService(branchesStore, store.Repos(), "https://api.example.com", reposDir)

	cleanup := func() {
		store.Close()
		os.RemoveAll(reposDir)
	}

	return service, store, reposDir, cleanup
}

// createGitRepo creates a bare git repository with an initial commit
func createGitRepo(t *testing.T, reposDir, owner, repoName string) string {
	t.Helper()

	// Create owner directory
	ownerDir := filepath.Join(reposDir, owner)
	if err := os.MkdirAll(ownerDir, 0755); err != nil {
		t.Fatalf("failed to create owner dir: %v", err)
	}

	// Create bare repo at owner/repo.git
	repoPath := filepath.Join(ownerDir, repoName+".git")
	_, _, err := pkggit.InitWithCommit(repoPath, pkggit.Signature{
		Name:  "Test Author",
		Email: "test@example.com",
		When:  time.Now(),
	}, "Initial commit")
	if err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	return repoPath
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

// Branch Listing Tests (Mock behavior)

func TestService_List_ReturnsDefaultBranch(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	repo := createTestRepo(t, store, user, "testrepo")
	repo.DefaultBranch = "main"

	list, err := service.List(context.Background(), "testowner", "testrepo", nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != 1 {
		t.Errorf("expected 1 branch, got %d", len(list))
	}
	if list[0].Name != "main" {
		t.Errorf("expected branch name 'main', got %q", list[0].Name)
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

func TestService_Get_ReturnsBranch(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")

	branch, err := service.Get(context.Background(), "testowner", "testrepo", "main")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if branch.Name != "main" {
		t.Errorf("expected branch name 'main', got %q", branch.Name)
	}
	if branch.Commit == nil {
		t.Error("expected commit to be set")
	}
	// Without a real git repo, SHA is empty (placeholder)
	if branch.Commit.SHA != "" {
		t.Errorf("expected empty SHA (no git repo), got %q", branch.Commit.SHA)
	}
}

func TestService_Get_RepoNotFound(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	_, err := service.Get(context.Background(), "unknown", "repo", "main")
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

func TestService_Get_WithProtection(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")

	// Set protection on the branch
	_, _ = service.UpdateProtection(context.Background(), "testowner", "testrepo", "main", &branches.UpdateProtectionIn{
		EnforceAdmins: true,
	})

	branch, err := service.Get(context.Background(), "testowner", "testrepo", "main")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if !branch.Protected {
		t.Error("expected branch to be protected")
	}
}

func TestService_Rename_Success(t *testing.T) {
	service, store, reposDir, cleanup := setupTestServiceWithGit(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	repo := createTestRepo(t, store, user, "testrepo")

	// Create git repository with main branch
	createGitRepo(t, reposDir, user.Login, repo.Name)

	// Rename main to master
	renamed, err := service.Rename(context.Background(), user.Login, repo.Name, "main", "master")
	if err != nil {
		t.Fatalf("Rename failed: %v", err)
	}

	if renamed.Name != "master" {
		t.Errorf("expected branch name 'master', got %q", renamed.Name)
	}
	if renamed.Commit == nil {
		t.Error("expected commit to be set")
	}
	if renamed.Commit.SHA == "" {
		t.Error("expected commit SHA to be set")
	}

	// Verify old branch no longer exists
	_, err = service.Get(context.Background(), user.Login, repo.Name, "main")
	if err != branches.ErrNotFound {
		t.Errorf("expected ErrNotFound for old branch, got %v", err)
	}

	// Verify new branch exists
	newBranch, err := service.Get(context.Background(), user.Login, repo.Name, "master")
	if err != nil {
		t.Fatalf("Get new branch failed: %v", err)
	}
	if newBranch.Name != "master" {
		t.Errorf("expected 'master', got %q", newBranch.Name)
	}
}

func TestService_Rename_BranchNotFound(t *testing.T) {
	service, store, reposDir, cleanup := setupTestServiceWithGit(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	repo := createTestRepo(t, store, user, "testrepo")

	// Create git repository
	createGitRepo(t, reposDir, user.Login, repo.Name)

	// Try to rename non-existent branch
	_, err := service.Rename(context.Background(), user.Login, repo.Name, "nonexistent", "newname")
	if err != branches.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_Rename_TargetExists(t *testing.T) {
	service, store, reposDir, cleanup := setupTestServiceWithGit(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	repo := createTestRepo(t, store, user, "testrepo")

	// Create git repository with main branch
	repoPath := createGitRepo(t, reposDir, user.Login, repo.Name)

	// Create a second branch
	gitRepo, _ := pkggit.Open(repoPath)
	ref, _ := gitRepo.GetRef("refs/heads/main")
	_ = gitRepo.CreateRef("refs/heads/develop", ref.SHA)

	// Try to rename main to develop (which already exists)
	_, err := service.Rename(context.Background(), user.Login, repo.Name, "main", "develop")
	if err != branches.ErrBranchExists {
		t.Errorf("expected ErrBranchExists, got %v", err)
	}
}

func TestService_Rename_ProtectedBranch(t *testing.T) {
	service, store, reposDir, cleanup := setupTestServiceWithGit(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	repo := createTestRepo(t, store, user, "testrepo")

	// Create git repository
	createGitRepo(t, reposDir, user.Login, repo.Name)

	// Protect the main branch
	_, _ = service.UpdateProtection(context.Background(), user.Login, repo.Name, "main", &branches.UpdateProtectionIn{
		EnforceAdmins: true,
	})

	// Try to rename protected branch
	_, err := service.Rename(context.Background(), user.Login, repo.Name, "main", "master")
	if err != branches.ErrProtected {
		t.Errorf("expected ErrProtected, got %v", err)
	}
}

func TestService_Rename_RepoNotFound(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	_, err := service.Rename(context.Background(), "unknown", "repo", "main", "master")
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

// Branch Protection Tests

func TestService_GetProtection_NotFound(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")

	_, err := service.GetProtection(context.Background(), "testowner", "testrepo", "main")
	if err != branches.ErrNotFound {
		t.Errorf("expected branches.ErrNotFound, got %v", err)
	}
}

func TestService_GetProtection_RepoNotFound(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	_, err := service.GetProtection(context.Background(), "unknown", "repo", "main")
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

func TestService_UpdateProtection_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")

	protection, err := service.UpdateProtection(context.Background(), "testowner", "testrepo", "main", &branches.UpdateProtectionIn{
		EnforceAdmins:         true,
		RequiredLinearHistory: true,
		AllowDeletions:        false,
	})
	if err != nil {
		t.Fatalf("UpdateProtection failed: %v", err)
	}

	if !protection.Enabled {
		t.Error("expected protection to be enabled")
	}
	if protection.EnforceAdmins == nil || !protection.EnforceAdmins.Enabled {
		t.Error("expected enforce_admins to be enabled")
	}
	if protection.RequiredLinearHistory == nil || !protection.RequiredLinearHistory.Enabled {
		t.Error("expected required_linear_history to be enabled")
	}
	if protection.URL == "" {
		t.Error("expected URL to be populated")
	}
}

func TestService_UpdateProtection_RepoNotFound(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	_, err := service.UpdateProtection(context.Background(), "unknown", "repo", "main", &branches.UpdateProtectionIn{
		EnforceAdmins: true,
	})
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

func TestService_UpdateProtection_WithStatusChecks(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")

	protection, err := service.UpdateProtection(context.Background(), "testowner", "testrepo", "main", &branches.UpdateProtectionIn{
		RequiredStatusChecks: &branches.RequiredStatusChecksIn{
			Strict:   true,
			Contexts: []string{"ci/build", "ci/test"},
		},
	})
	if err != nil {
		t.Fatalf("UpdateProtection failed: %v", err)
	}

	if protection.RequiredStatusChecks == nil {
		t.Fatal("expected required_status_checks to be set")
	}
	if !protection.RequiredStatusChecks.Strict {
		t.Error("expected strict to be true")
	}
	if len(protection.RequiredStatusChecks.Contexts) != 2 {
		t.Errorf("expected 2 contexts, got %d", len(protection.RequiredStatusChecks.Contexts))
	}
}

func TestService_UpdateProtection_WithPRReviews(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")

	protection, err := service.UpdateProtection(context.Background(), "testowner", "testrepo", "main", &branches.UpdateProtectionIn{
		RequiredPullRequestReviews: &branches.RequiredPullRequestReviewsIn{
			DismissStaleReviews:          true,
			RequireCodeOwnerReviews:      true,
			RequiredApprovingReviewCount: 2,
		},
	})
	if err != nil {
		t.Fatalf("UpdateProtection failed: %v", err)
	}

	if protection.RequiredPullRequestReviews == nil {
		t.Fatal("expected required_pull_request_reviews to be set")
	}
	if !protection.RequiredPullRequestReviews.DismissStaleReviews {
		t.Error("expected dismiss_stale_reviews to be true")
	}
	if protection.RequiredPullRequestReviews.RequiredApprovingReviewCount != 2 {
		t.Errorf("expected 2 approvals, got %d", protection.RequiredPullRequestReviews.RequiredApprovingReviewCount)
	}
}

func TestService_UpdateProtection_WithForcePushes(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")

	allowForce := true
	protection, err := service.UpdateProtection(context.Background(), "testowner", "testrepo", "main", &branches.UpdateProtectionIn{
		AllowForcePushes: &allowForce,
	})
	if err != nil {
		t.Fatalf("UpdateProtection failed: %v", err)
	}

	if protection.AllowForcePushes == nil || !protection.AllowForcePushes.Enabled {
		t.Error("expected allow_force_pushes to be enabled")
	}
}

func TestService_DeleteProtection_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")

	// First create protection
	_, _ = service.UpdateProtection(context.Background(), "testowner", "testrepo", "main", &branches.UpdateProtectionIn{
		EnforceAdmins: true,
	})

	// Then delete it
	err := service.DeleteProtection(context.Background(), "testowner", "testrepo", "main")
	if err != nil {
		t.Fatalf("DeleteProtection failed: %v", err)
	}

	// Verify it's gone
	_, err = service.GetProtection(context.Background(), "testowner", "testrepo", "main")
	if err != branches.ErrNotFound {
		t.Errorf("expected branches.ErrNotFound, got %v", err)
	}
}

func TestService_DeleteProtection_RepoNotFound(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	err := service.DeleteProtection(context.Background(), "unknown", "repo", "main")
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

// Required Status Checks Tests

func TestService_GetRequiredStatusChecks_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")

	// Create protection with status checks
	_, _ = service.UpdateProtection(context.Background(), "testowner", "testrepo", "main", &branches.UpdateProtectionIn{
		RequiredStatusChecks: &branches.RequiredStatusChecksIn{
			Strict:   true,
			Contexts: []string{"ci/test"},
		},
	})

	checks, err := service.GetRequiredStatusChecks(context.Background(), "testowner", "testrepo", "main")
	if err != nil {
		t.Fatalf("GetRequiredStatusChecks failed: %v", err)
	}

	if !checks.Strict {
		t.Error("expected strict to be true")
	}
}

func TestService_GetRequiredStatusChecks_NotFound(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")

	// Create protection without status checks
	_, _ = service.UpdateProtection(context.Background(), "testowner", "testrepo", "main", &branches.UpdateProtectionIn{
		EnforceAdmins: true,
	})

	_, err := service.GetRequiredStatusChecks(context.Background(), "testowner", "testrepo", "main")
	if err != branches.ErrNotFound {
		t.Errorf("expected branches.ErrNotFound, got %v", err)
	}
}

func TestService_UpdateRequiredStatusChecks_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")

	// Create initial protection
	_, _ = service.UpdateProtection(context.Background(), "testowner", "testrepo", "main", &branches.UpdateProtectionIn{
		EnforceAdmins: true,
	})

	checks, err := service.UpdateRequiredStatusChecks(context.Background(), "testowner", "testrepo", "main", &branches.RequiredStatusChecksIn{
		Strict:   true,
		Contexts: []string{"ci/build", "ci/test"},
	})
	if err != nil {
		t.Fatalf("UpdateRequiredStatusChecks failed: %v", err)
	}

	if !checks.Strict {
		t.Error("expected strict to be true")
	}
	if len(checks.Contexts) != 2 {
		t.Errorf("expected 2 contexts, got %d", len(checks.Contexts))
	}
	if checks.URL == "" {
		t.Error("expected URL to be populated")
	}
}

func TestService_RemoveRequiredStatusChecks_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")

	// Create protection with status checks
	_, _ = service.UpdateProtection(context.Background(), "testowner", "testrepo", "main", &branches.UpdateProtectionIn{
		RequiredStatusChecks: &branches.RequiredStatusChecksIn{
			Strict:   true,
			Contexts: []string{"ci/test"},
		},
	})

	err := service.RemoveRequiredStatusChecks(context.Background(), "testowner", "testrepo", "main")
	if err != nil {
		t.Fatalf("RemoveRequiredStatusChecks failed: %v", err)
	}

	_, err = service.GetRequiredStatusChecks(context.Background(), "testowner", "testrepo", "main")
	if err != branches.ErrNotFound {
		t.Errorf("expected branches.ErrNotFound, got %v", err)
	}
}

// Required Signatures Tests

func TestService_GetRequiredSignatures_Default(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")

	// Create protection without signatures
	_, _ = service.UpdateProtection(context.Background(), "testowner", "testrepo", "main", &branches.UpdateProtectionIn{
		EnforceAdmins: true,
	})

	setting, err := service.GetRequiredSignatures(context.Background(), "testowner", "testrepo", "main")
	if err != nil {
		t.Fatalf("GetRequiredSignatures failed: %v", err)
	}

	if setting.Enabled {
		t.Error("expected signatures to be disabled by default")
	}
}

func TestService_CreateRequiredSignatures_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")

	// Create initial protection
	_, _ = service.UpdateProtection(context.Background(), "testowner", "testrepo", "main", &branches.UpdateProtectionIn{
		EnforceAdmins: true,
	})

	setting, err := service.CreateRequiredSignatures(context.Background(), "testowner", "testrepo", "main")
	if err != nil {
		t.Fatalf("CreateRequiredSignatures failed: %v", err)
	}

	if !setting.Enabled {
		t.Error("expected signatures to be enabled")
	}
}

func TestService_DeleteRequiredSignatures_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")

	// Create protection with signatures enabled
	_, _ = service.UpdateProtection(context.Background(), "testowner", "testrepo", "main", &branches.UpdateProtectionIn{
		EnforceAdmins: true,
	})
	_, _ = service.CreateRequiredSignatures(context.Background(), "testowner", "testrepo", "main")

	err := service.DeleteRequiredSignatures(context.Background(), "testowner", "testrepo", "main")
	if err != nil {
		t.Fatalf("DeleteRequiredSignatures failed: %v", err)
	}

	setting, _ := service.GetRequiredSignatures(context.Background(), "testowner", "testrepo", "main")
	if setting.Enabled {
		t.Error("expected signatures to be disabled")
	}
}

// URL Population Tests

func TestService_PopulateProtectionURLs(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")

	protection, err := service.UpdateProtection(context.Background(), "testowner", "testrepo", "main", &branches.UpdateProtectionIn{
		EnforceAdmins: true,
		RequiredStatusChecks: &branches.RequiredStatusChecksIn{
			Strict: true,
		},
		RequiredPullRequestReviews: &branches.RequiredPullRequestReviewsIn{
			RequiredApprovingReviewCount: 1,
		},
		Restrictions: &branches.RestrictionsIn{
			Users: []string{},
			Teams: []string{},
		},
	})
	if err != nil {
		t.Fatalf("UpdateProtection failed: %v", err)
	}

	expectedURL := "https://api.example.com/api/v3/repos/testowner/testrepo/branches/main/protection"
	if protection.URL != expectedURL {
		t.Errorf("expected URL %q, got %q", expectedURL, protection.URL)
	}

	if protection.EnforceAdmins != nil && protection.EnforceAdmins.URL == "" {
		t.Error("expected enforce_admins URL to be populated")
	}

	if protection.RequiredStatusChecks != nil && protection.RequiredStatusChecks.URL == "" {
		t.Error("expected required_status_checks URL to be populated")
	}

	if protection.RequiredPullRequestReviews != nil && protection.RequiredPullRequestReviews.URL == "" {
		t.Error("expected required_pull_request_reviews URL to be populated")
	}

	if protection.Restrictions != nil && protection.Restrictions.URL == "" {
		t.Error("expected restrictions URL to be populated")
	}
}

// Multi-branch Tests

func TestService_Protection_MultipleBranches(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")

	// Protect main
	_, _ = service.UpdateProtection(context.Background(), "testowner", "testrepo", "main", &branches.UpdateProtectionIn{
		EnforceAdmins: true,
	})

	// Protect develop
	_, _ = service.UpdateProtection(context.Background(), "testowner", "testrepo", "develop", &branches.UpdateProtectionIn{
		RequiredLinearHistory: true,
	})

	// Verify main protection
	mainProtection, _ := service.GetProtection(context.Background(), "testowner", "testrepo", "main")
	if mainProtection.EnforceAdmins == nil || !mainProtection.EnforceAdmins.Enabled {
		t.Error("expected main branch to have enforce_admins enabled")
	}

	// Verify develop protection
	devProtection, _ := service.GetProtection(context.Background(), "testowner", "testrepo", "develop")
	if devProtection.RequiredLinearHistory == nil || !devProtection.RequiredLinearHistory.Enabled {
		t.Error("expected develop branch to have required_linear_history enabled")
	}
}
