package e2e

import (
	"context"
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/go-mizu/blueprints/githome/pkg/seed/local"
	"github.com/go-mizu/blueprints/githome/store/duckdb"
)

// setupTestDB creates an in-memory DuckDB database for testing
func setupTestDB(t *testing.T) (*sql.DB, *duckdb.Store, func()) {
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
		db.Close()
		t.Fatalf("failed to ensure schema: %v", err)
	}

	return db, store, func() {
		db.Close()
	}
}

// setupTestRepos creates a temporary directory with mock git repositories
func setupTestRepos(t *testing.T) (string, func()) {
	t.Helper()

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "githome-e2e-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Create org/repo structure
	orgs := []struct {
		name  string
		repos []string
	}{
		{
			name:  "test-org1",
			repos: []string{"repo-a", "repo-b"},
		},
		{
			name:  "test-org2",
			repos: []string{"repo-c"},
		},
		{
			name:  "test-org3",
			repos: []string{"repo-d", "repo-e", "repo-f"},
		},
	}

	for _, org := range orgs {
		orgPath := filepath.Join(tempDir, org.name)
		if err := os.MkdirAll(orgPath, 0755); err != nil {
			os.RemoveAll(tempDir)
			t.Fatalf("failed to create org dir: %v", err)
		}

		for _, repoName := range org.repos {
			repoPath := filepath.Join(orgPath, repoName)
			if err := os.MkdirAll(repoPath, 0755); err != nil {
				os.RemoveAll(tempDir)
				t.Fatalf("failed to create repo dir: %v", err)
			}

			// Initialize git repo
			cmd := exec.Command("git", "init")
			cmd.Dir = repoPath
			if err := cmd.Run(); err != nil {
				os.RemoveAll(tempDir)
				t.Fatalf("failed to git init: %v", err)
			}

			// Create a sample file
			sampleFile := filepath.Join(repoPath, "README.md")
			if err := os.WriteFile(sampleFile, []byte("# "+repoName+"\n\nTest repository"), 0644); err != nil {
				os.RemoveAll(tempDir)
				t.Fatalf("failed to create README: %v", err)
			}

			// Git add and commit
			cmd = exec.Command("git", "add", ".")
			cmd.Dir = repoPath
			cmd.Run() // Ignore errors, git add may fail without config

			cmd = exec.Command("git", "-c", "user.email=test@test.com", "-c", "user.name=Test", "commit", "-m", "Initial commit")
			cmd.Dir = repoPath
			cmd.Run() // Ignore errors, commit may fail without config
		}
	}

	return tempDir, func() {
		os.RemoveAll(tempDir)
	}
}

func TestLocalSeeder_Seed(t *testing.T) {
	db, _, cleanupDB := setupTestDB(t)
	defer cleanupDB()

	tempDir, cleanupRepos := setupTestRepos(t)
	defer cleanupRepos()

	ctx := context.Background()

	// Ensure admin user
	usersStore := duckdb.NewUsersStore(db)
	actorsStore := duckdb.NewActorsStore(db)
	adminUserID, adminActorID, err := local.EnsureAdminUser(ctx, usersStore, actorsStore)
	if err != nil {
		t.Fatalf("failed to ensure admin user: %v", err)
	}

	// Create seeder
	seeder := local.NewSeeder(db, local.Config{
		ScanDir:      tempDir,
		AdminUserID:  adminUserID,
		AdminActorID: adminActorID,
		IsPublic:     true,
	})

	// Run seeding
	result, err := seeder.Seed(ctx)
	if err != nil {
		t.Fatalf("Seed failed: %v", err)
	}

	// Verify results
	if result.OrgsCreated != 3 {
		t.Errorf("got %d orgs created, want 3", result.OrgsCreated)
	}
	if result.ReposCreated != 6 {
		t.Errorf("got %d repos created, want 6", result.ReposCreated)
	}
	if result.ReposSkipped != 0 {
		t.Errorf("got %d repos skipped, want 0", result.ReposSkipped)
	}
	if len(result.Errors) != 0 {
		t.Errorf("got %d errors, want 0: %v", len(result.Errors), result.Errors)
	}
}

func TestLocalSeeder_Seed_Idempotent(t *testing.T) {
	db, _, cleanupDB := setupTestDB(t)
	defer cleanupDB()

	tempDir, cleanupRepos := setupTestRepos(t)
	defer cleanupRepos()

	ctx := context.Background()

	// Ensure admin user
	usersStore := duckdb.NewUsersStore(db)
	actorsStore := duckdb.NewActorsStore(db)
	adminUserID, adminActorID, err := local.EnsureAdminUser(ctx, usersStore, actorsStore)
	if err != nil {
		t.Fatalf("failed to ensure admin user: %v", err)
	}

	// Create seeder
	seeder := local.NewSeeder(db, local.Config{
		ScanDir:      tempDir,
		AdminUserID:  adminUserID,
		AdminActorID: adminActorID,
		IsPublic:     true,
	})

	// Run seeding twice
	result1, err := seeder.Seed(ctx)
	if err != nil {
		t.Fatalf("first Seed failed: %v", err)
	}

	result2, err := seeder.Seed(ctx)
	if err != nil {
		t.Fatalf("second Seed failed: %v", err)
	}

	// First run should create repos
	if result1.ReposCreated != 6 {
		t.Errorf("first run: got %d repos created, want 6", result1.ReposCreated)
	}

	// Second run should skip all (already exist)
	if result2.ReposCreated != 0 {
		t.Errorf("second run: got %d repos created, want 0", result2.ReposCreated)
	}
	if result2.ReposSkipped != 6 {
		t.Errorf("second run: got %d repos skipped, want 6", result2.ReposSkipped)
	}
}

func TestLocalSeeder_PublicReposVisibleInExplorer(t *testing.T) {
	db, _, cleanupDB := setupTestDB(t)
	defer cleanupDB()

	tempDir, cleanupRepos := setupTestRepos(t)
	defer cleanupRepos()

	ctx := context.Background()

	// Ensure admin user
	usersStore := duckdb.NewUsersStore(db)
	actorsStore := duckdb.NewActorsStore(db)
	adminUserID, adminActorID, err := local.EnsureAdminUser(ctx, usersStore, actorsStore)
	if err != nil {
		t.Fatalf("failed to ensure admin user: %v", err)
	}

	// Create seeder with public repos
	seeder := local.NewSeeder(db, local.Config{
		ScanDir:      tempDir,
		AdminUserID:  adminUserID,
		AdminActorID: adminActorID,
		IsPublic:     true,
	})

	// Run seeding
	_, err = seeder.Seed(ctx)
	if err != nil {
		t.Fatalf("Seed failed: %v", err)
	}

	// Verify repos are visible via ListPublic (explorer endpoint)
	reposStore := duckdb.NewReposStore(db)
	publicRepos, err := reposStore.ListPublic(ctx, 100, 0)
	if err != nil {
		t.Fatalf("ListPublic failed: %v", err)
	}

	if len(publicRepos) != 6 {
		t.Errorf("got %d public repos, want 6", len(publicRepos))
	}

	// Verify all repos are not private
	for _, repo := range publicRepos {
		if repo.IsPrivate {
			t.Errorf("repo %s should be public", repo.Name)
		}
	}
}

func TestLocalSeeder_PrivateReposNotVisibleInExplorer(t *testing.T) {
	db, _, cleanupDB := setupTestDB(t)
	defer cleanupDB()

	tempDir, cleanupRepos := setupTestRepos(t)
	defer cleanupRepos()

	ctx := context.Background()

	// Ensure admin user
	usersStore := duckdb.NewUsersStore(db)
	actorsStore := duckdb.NewActorsStore(db)
	adminUserID, adminActorID, err := local.EnsureAdminUser(ctx, usersStore, actorsStore)
	if err != nil {
		t.Fatalf("failed to ensure admin user: %v", err)
	}

	// Create seeder with private repos
	seeder := local.NewSeeder(db, local.Config{
		ScanDir:      tempDir,
		AdminUserID:  adminUserID,
		AdminActorID: adminActorID,
		IsPublic:     false,
	})

	// Run seeding
	_, err = seeder.Seed(ctx)
	if err != nil {
		t.Fatalf("Seed failed: %v", err)
	}

	// Verify repos are NOT visible via ListPublic (explorer endpoint)
	reposStore := duckdb.NewReposStore(db)
	publicRepos, err := reposStore.ListPublic(ctx, 100, 0)
	if err != nil {
		t.Fatalf("ListPublic failed: %v", err)
	}

	if len(publicRepos) != 0 {
		t.Errorf("got %d public repos, want 0 (all should be private)", len(publicRepos))
	}
}

func TestLocalSeeder_OrgsCreatedWithAdminMember(t *testing.T) {
	db, _, cleanupDB := setupTestDB(t)
	defer cleanupDB()

	tempDir, cleanupRepos := setupTestRepos(t)
	defer cleanupRepos()

	ctx := context.Background()

	// Ensure admin user
	usersStore := duckdb.NewUsersStore(db)
	actorsStore := duckdb.NewActorsStore(db)
	adminUserID, adminActorID, err := local.EnsureAdminUser(ctx, usersStore, actorsStore)
	if err != nil {
		t.Fatalf("failed to ensure admin user: %v", err)
	}

	// Create seeder
	seeder := local.NewSeeder(db, local.Config{
		ScanDir:      tempDir,
		AdminUserID:  adminUserID,
		AdminActorID: adminActorID,
		IsPublic:     true,
	})

	// Run seeding
	_, err = seeder.Seed(ctx)
	if err != nil {
		t.Fatalf("Seed failed: %v", err)
	}

	// Verify orgs exist and admin is a member
	orgsStore := duckdb.NewOrgsStore(db)

	for _, orgName := range []string{"test-org1", "test-org2", "test-org3"} {
		org, err := orgsStore.GetBySlug(ctx, orgName)
		if err != nil {
			t.Fatalf("failed to get org %s: %v", orgName, err)
		}
		if org == nil {
			t.Errorf("org %s not found", orgName)
			continue
		}

		// Verify admin is a member
		member, err := orgsStore.GetMember(ctx, org.ID, adminUserID)
		if err != nil {
			t.Fatalf("failed to get member: %v", err)
		}
		if member == nil {
			t.Errorf("admin should be a member of org %s", orgName)
			continue
		}
		if member.Role != "owner" {
			t.Errorf("admin should be owner of org %s, got role %s", orgName, member.Role)
		}
	}
}

func TestLocalSeeder_RepoStorageLinked(t *testing.T) {
	db, _, cleanupDB := setupTestDB(t)
	defer cleanupDB()

	tempDir, cleanupRepos := setupTestRepos(t)
	defer cleanupRepos()

	ctx := context.Background()

	// Ensure admin user
	usersStore := duckdb.NewUsersStore(db)
	actorsStore := duckdb.NewActorsStore(db)
	adminUserID, adminActorID, err := local.EnsureAdminUser(ctx, usersStore, actorsStore)
	if err != nil {
		t.Fatalf("failed to ensure admin user: %v", err)
	}

	// Create seeder
	seeder := local.NewSeeder(db, local.Config{
		ScanDir:      tempDir,
		AdminUserID:  adminUserID,
		AdminActorID: adminActorID,
		IsPublic:     true,
	})

	// Run seeding
	_, err = seeder.Seed(ctx)
	if err != nil {
		t.Fatalf("Seed failed: %v", err)
	}

	// Verify repo_storage has entries
	reposStore := duckdb.NewReposStore(db)
	repoStorageStore := duckdb.NewRepoStorageStore(db)

	publicRepos, _ := reposStore.ListPublic(ctx, 100, 0)
	for _, repo := range publicRepos {
		storage, err := repoStorageStore.GetByRepoID(ctx, repo.ID)
		if err != nil {
			t.Fatalf("failed to get storage for repo %s: %v", repo.Name, err)
		}
		if storage == nil {
			t.Errorf("no storage entry for repo %s", repo.Name)
			continue
		}
		if storage.StorageBackend != "fs" {
			t.Errorf("expected fs backend for repo %s, got %s", repo.Name, storage.StorageBackend)
		}
		if storage.StoragePath == "" {
			t.Errorf("expected non-empty storage path for repo %s", repo.Name)
		}

		// Verify path exists
		if _, err := os.Stat(storage.StoragePath); os.IsNotExist(err) {
			t.Errorf("storage path does not exist for repo %s: %s", repo.Name, storage.StoragePath)
		}
	}
}

func TestLocalSeeder_SkipsNonGitDirectories(t *testing.T) {
	db, _, cleanupDB := setupTestDB(t)
	defer cleanupDB()

	// Create temp directory with mixed content
	tempDir, err := os.MkdirTemp("", "githome-e2e-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create org with git and non-git directories
	orgPath := filepath.Join(tempDir, "mixed-org")
	os.MkdirAll(orgPath, 0755)

	// Create a git repo
	gitRepoPath := filepath.Join(orgPath, "real-repo")
	os.MkdirAll(gitRepoPath, 0755)
	exec.Command("git", "init").Dir = gitRepoPath
	cmd := exec.Command("git", "init")
	cmd.Dir = gitRepoPath
	cmd.Run()

	// Create a non-git directory
	nonGitPath := filepath.Join(orgPath, "not-a-repo")
	os.MkdirAll(nonGitPath, 0755)
	os.WriteFile(filepath.Join(nonGitPath, "file.txt"), []byte("just a file"), 0644)

	// Create hidden directory (should be skipped)
	hiddenPath := filepath.Join(orgPath, ".hidden")
	os.MkdirAll(hiddenPath, 0755)

	ctx := context.Background()

	// Ensure admin user
	usersStore := duckdb.NewUsersStore(db)
	actorsStore := duckdb.NewActorsStore(db)
	adminUserID, adminActorID, err := local.EnsureAdminUser(ctx, usersStore, actorsStore)
	if err != nil {
		t.Fatalf("failed to ensure admin user: %v", err)
	}

	// Create seeder
	seeder := local.NewSeeder(db, local.Config{
		ScanDir:      tempDir,
		AdminUserID:  adminUserID,
		AdminActorID: adminActorID,
		IsPublic:     true,
	})

	// Run seeding
	result, err := seeder.Seed(ctx)
	if err != nil {
		t.Fatalf("Seed failed: %v", err)
	}

	// Should only have created 1 repo (the git one)
	if result.ReposCreated != 1 {
		t.Errorf("got %d repos created, want 1", result.ReposCreated)
	}
}

func TestLocalSeeder_EmptyDirectory(t *testing.T) {
	db, _, cleanupDB := setupTestDB(t)
	defer cleanupDB()

	// Create empty temp directory
	tempDir, err := os.MkdirTemp("", "githome-e2e-empty-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	ctx := context.Background()

	// Ensure admin user
	usersStore := duckdb.NewUsersStore(db)
	actorsStore := duckdb.NewActorsStore(db)
	adminUserID, adminActorID, err := local.EnsureAdminUser(ctx, usersStore, actorsStore)
	if err != nil {
		t.Fatalf("failed to ensure admin user: %v", err)
	}

	// Create seeder
	seeder := local.NewSeeder(db, local.Config{
		ScanDir:      tempDir,
		AdminUserID:  adminUserID,
		AdminActorID: adminActorID,
		IsPublic:     true,
	})

	// Run seeding
	result, err := seeder.Seed(ctx)
	if err != nil {
		t.Fatalf("Seed failed: %v", err)
	}

	// Should have created nothing
	if result.OrgsCreated != 0 {
		t.Errorf("got %d orgs created, want 0", result.OrgsCreated)
	}
	if result.ReposCreated != 0 {
		t.Errorf("got %d repos created, want 0", result.ReposCreated)
	}
}

func TestLocalSeeder_NonExistentDirectory(t *testing.T) {
	db, _, cleanupDB := setupTestDB(t)
	defer cleanupDB()

	ctx := context.Background()

	// Ensure admin user
	usersStore := duckdb.NewUsersStore(db)
	actorsStore := duckdb.NewActorsStore(db)
	adminUserID, adminActorID, err := local.EnsureAdminUser(ctx, usersStore, actorsStore)
	if err != nil {
		t.Fatalf("failed to ensure admin user: %v", err)
	}

	// Create seeder with non-existent directory
	seeder := local.NewSeeder(db, local.Config{
		ScanDir:      "/nonexistent/path/that/does/not/exist",
		AdminUserID:  adminUserID,
		AdminActorID: adminActorID,
		IsPublic:     true,
	})

	// Run seeding - should fail
	_, err = seeder.Seed(ctx)
	if err == nil {
		t.Error("expected error for non-existent directory")
	}
}
