package stars_test

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/stars"
	"github.com/go-mizu/blueprints/githome/feature/users"
	"github.com/go-mizu/blueprints/githome/store/duckdb"
)

func setupTestService(t *testing.T) (*stars.Service, *duckdb.Store, func()) {
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

	starsStore := duckdb.NewStarsStore(db)
	service := stars.NewService(starsStore, store.Repos(), store.Users(), "https://api.example.com")

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

// Star/Unstar Tests

func TestService_Star_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	starrer := createTestUser(t, store, "starrer", "starrer@example.com")
	repo := createTestRepo(t, store, owner, "testrepo")

	err := service.Star(context.Background(), starrer.ID, "owner", "testrepo")
	if err != nil {
		t.Fatalf("Star failed: %v", err)
	}

	// Verify starred
	isStarred, err := service.IsStarred(context.Background(), starrer.ID, "owner", "testrepo")
	if err != nil {
		t.Fatalf("IsStarred failed: %v", err)
	}
	if !isStarred {
		t.Error("expected repo to be starred")
	}

	// Verify counter incremented
	updatedRepo, _ := store.Repos().GetByID(context.Background(), repo.ID)
	if updatedRepo.StargazersCount != 1 {
		t.Errorf("expected stargazers_count 1, got %d", updatedRepo.StargazersCount)
	}
}

func TestService_Star_AlreadyStarred(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	starrer := createTestUser(t, store, "starrer", "starrer@example.com")
	repo := createTestRepo(t, store, owner, "testrepo")

	// First star
	_ = service.Star(context.Background(), starrer.ID, "owner", "testrepo")

	// Second star should be idempotent
	err := service.Star(context.Background(), starrer.ID, "owner", "testrepo")
	if err != nil {
		t.Fatalf("Second star should succeed (idempotent): %v", err)
	}

	// Counter should still be 1
	updatedRepo, _ := store.Repos().GetByID(context.Background(), repo.ID)
	if updatedRepo.StargazersCount != 1 {
		t.Errorf("expected stargazers_count 1, got %d", updatedRepo.StargazersCount)
	}
}

func TestService_Star_RepoNotFound(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	starrer := createTestUser(t, store, "starrer", "starrer@example.com")

	err := service.Star(context.Background(), starrer.ID, "unknown", "repo")
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

func TestService_Unstar_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	starrer := createTestUser(t, store, "starrer", "starrer@example.com")
	repo := createTestRepo(t, store, owner, "testrepo")

	// First star
	_ = service.Star(context.Background(), starrer.ID, "owner", "testrepo")

	// Then unstar
	err := service.Unstar(context.Background(), starrer.ID, "owner", "testrepo")
	if err != nil {
		t.Fatalf("Unstar failed: %v", err)
	}

	// Verify unstarred
	isStarred, _ := service.IsStarred(context.Background(), starrer.ID, "owner", "testrepo")
	if isStarred {
		t.Error("expected repo to not be starred")
	}

	// Verify counter decremented
	updatedRepo, _ := store.Repos().GetByID(context.Background(), repo.ID)
	if updatedRepo.StargazersCount != 0 {
		t.Errorf("expected stargazers_count 0, got %d", updatedRepo.StargazersCount)
	}
}

func TestService_Unstar_NotStarred(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	starrer := createTestUser(t, store, "starrer", "starrer@example.com")
	createTestRepo(t, store, owner, "testrepo")

	// Unstar without starring should be idempotent
	err := service.Unstar(context.Background(), starrer.ID, "owner", "testrepo")
	if err != nil {
		t.Fatalf("Unstar should succeed (idempotent): %v", err)
	}
}

func TestService_Unstar_RepoNotFound(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	starrer := createTestUser(t, store, "starrer", "starrer@example.com")

	err := service.Unstar(context.Background(), starrer.ID, "unknown", "repo")
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

// IsStarred Tests

func TestService_IsStarred_True(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	starrer := createTestUser(t, store, "starrer", "starrer@example.com")
	createTestRepo(t, store, owner, "testrepo")

	_ = service.Star(context.Background(), starrer.ID, "owner", "testrepo")

	isStarred, err := service.IsStarred(context.Background(), starrer.ID, "owner", "testrepo")
	if err != nil {
		t.Fatalf("IsStarred failed: %v", err)
	}
	if !isStarred {
		t.Error("expected repo to be starred")
	}
}

func TestService_IsStarred_False(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	starrer := createTestUser(t, store, "starrer", "starrer@example.com")
	createTestRepo(t, store, owner, "testrepo")

	isStarred, err := service.IsStarred(context.Background(), starrer.ID, "owner", "testrepo")
	if err != nil {
		t.Fatalf("IsStarred failed: %v", err)
	}
	if isStarred {
		t.Error("expected repo to not be starred")
	}
}

func TestService_IsStarred_RepoNotFound(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	starrer := createTestUser(t, store, "starrer", "starrer@example.com")

	_, err := service.IsStarred(context.Background(), starrer.ID, "unknown", "repo")
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

// List Stargazers Tests

func TestService_ListStargazers(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	starrer1 := createTestUser(t, store, "starrer1", "starrer1@example.com")
	starrer2 := createTestUser(t, store, "starrer2", "starrer2@example.com")
	createTestRepo(t, store, owner, "testrepo")

	_ = service.Star(context.Background(), starrer1.ID, "owner", "testrepo")
	_ = service.Star(context.Background(), starrer2.ID, "owner", "testrepo")

	stargazers, err := service.ListStargazers(context.Background(), "owner", "testrepo", nil)
	if err != nil {
		t.Fatalf("ListStargazers failed: %v", err)
	}

	if len(stargazers) != 2 {
		t.Errorf("expected 2 stargazers, got %d", len(stargazers))
	}
}

func TestService_ListStargazers_Pagination(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")

	// Create multiple starrers
	for i := 0; i < 5; i++ {
		starrer := createTestUser(t, store, "starrer"+string(rune('a'+i)), "starrer"+string(rune('a'+i))+"@example.com")
		_ = service.Star(context.Background(), starrer.ID, "owner", "testrepo")
	}

	stargazers, err := service.ListStargazers(context.Background(), "owner", "testrepo", &stars.ListOpts{
		Page:    1,
		PerPage: 2,
	})
	if err != nil {
		t.Fatalf("ListStargazers failed: %v", err)
	}

	if len(stargazers) != 2 {
		t.Errorf("expected 2 stargazers, got %d", len(stargazers))
	}
}

func TestService_ListStargazers_RepoNotFound(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	_, err := service.ListStargazers(context.Background(), "unknown", "repo", nil)
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

func TestService_ListStargazersWithTimestamps(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	starrer := createTestUser(t, store, "starrer", "starrer@example.com")
	createTestRepo(t, store, owner, "testrepo")

	_ = service.Star(context.Background(), starrer.ID, "owner", "testrepo")

	stargazers, err := service.ListStargazersWithTimestamps(context.Background(), "owner", "testrepo", nil)
	if err != nil {
		t.Fatalf("ListStargazersWithTimestamps failed: %v", err)
	}

	if len(stargazers) != 1 {
		t.Errorf("expected 1 stargazer, got %d", len(stargazers))
	}
	if stargazers[0].StarredAt.IsZero() {
		t.Error("expected starred_at to be set")
	}
}

// List Starred Repos Tests

func TestService_ListForUser(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	starrer := createTestUser(t, store, "starrer", "starrer@example.com")
	createTestRepo(t, store, owner, "repo1")
	createTestRepo(t, store, owner, "repo2")

	_ = service.Star(context.Background(), starrer.ID, "owner", "repo1")
	_ = service.Star(context.Background(), starrer.ID, "owner", "repo2")

	starred, err := service.ListForUser(context.Background(), "starrer", nil)
	if err != nil {
		t.Fatalf("ListForUser failed: %v", err)
	}

	if len(starred) != 2 {
		t.Errorf("expected 2 starred repos, got %d", len(starred))
	}
}

func TestService_ListForUser_UserNotFound(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	_, err := service.ListForUser(context.Background(), "unknown", nil)
	if err == nil {
		t.Error("expected error for unknown user")
	}
}

func TestService_ListForAuthenticatedUser(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	starrer := createTestUser(t, store, "starrer", "starrer@example.com")
	createTestRepo(t, store, owner, "repo1")

	_ = service.Star(context.Background(), starrer.ID, "owner", "repo1")

	starred, err := service.ListForAuthenticatedUser(context.Background(), starrer.ID, nil)
	if err != nil {
		t.Fatalf("ListForAuthenticatedUser failed: %v", err)
	}

	if len(starred) != 1 {
		t.Errorf("expected 1 starred repo, got %d", len(starred))
	}
}

func TestService_ListForAuthenticatedUserWithTimestamps(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	starrer := createTestUser(t, store, "starrer", "starrer@example.com")
	createTestRepo(t, store, owner, "repo1")

	_ = service.Star(context.Background(), starrer.ID, "owner", "repo1")

	starred, err := service.ListForAuthenticatedUserWithTimestamps(context.Background(), starrer.ID, nil)
	if err != nil {
		t.Fatalf("ListForAuthenticatedUserWithTimestamps failed: %v", err)
	}

	if len(starred) != 1 {
		t.Errorf("expected 1 starred repo, got %d", len(starred))
	}
	if starred[0].StarredAt.IsZero() {
		t.Error("expected starred_at to be set")
	}
}

// Integration Test - Multiple Users Starring

func TestService_MultipleUsersStarring(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	repo := createTestRepo(t, store, owner, "testrepo")

	// Multiple users star the repo
	for i := 0; i < 3; i++ {
		starrer := createTestUser(t, store, "starrer"+string(rune('a'+i)), "starrer"+string(rune('a'+i))+"@example.com")
		_ = service.Star(context.Background(), starrer.ID, "owner", "testrepo")
	}

	// Verify counter
	updatedRepo, _ := store.Repos().GetByID(context.Background(), repo.ID)
	if updatedRepo.StargazersCount != 3 {
		t.Errorf("expected stargazers_count 3, got %d", updatedRepo.StargazersCount)
	}

	// Verify list
	stargazers, _ := service.ListStargazers(context.Background(), "owner", "testrepo", nil)
	if len(stargazers) != 3 {
		t.Errorf("expected 3 stargazers, got %d", len(stargazers))
	}
}
