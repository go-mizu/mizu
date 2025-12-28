package watches_test

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
	"github.com/go-mizu/blueprints/githome/feature/watches"
	"github.com/go-mizu/blueprints/githome/store/duckdb"
)

func setupTestService(t *testing.T) (*watches.Service, *duckdb.Store, func()) {
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

	watchesStore := duckdb.NewWatchesStore(db)
	service := watches.NewService(watchesStore, store.Repos(), store.Users(), "https://api.example.com")

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

// Subscription Tests

func TestService_SetSubscription_Subscribe(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	watcher := createTestUser(t, store, "watcher", "watcher@example.com")
	repo := createTestRepo(t, store, owner, "testrepo")

	sub, err := service.SetSubscription(context.Background(), watcher.ID, "owner", "testrepo", true, false)
	if err != nil {
		t.Fatalf("SetSubscription failed: %v", err)
	}

	if !sub.Subscribed {
		t.Error("expected subscribed to be true")
	}
	if sub.Ignored {
		t.Error("expected ignored to be false")
	}
	if sub.URL == "" {
		t.Error("expected URL to be set")
	}

	// Verify counter incremented
	updatedRepo, _ := store.Repos().GetByID(context.Background(), repo.ID)
	if updatedRepo.WatchersCount != 1 {
		t.Errorf("expected watchers_count 1, got %d", updatedRepo.WatchersCount)
	}
}

func TestService_SetSubscription_Ignore(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	watcher := createTestUser(t, store, "watcher", "watcher@example.com")
	createTestRepo(t, store, owner, "testrepo")

	sub, err := service.SetSubscription(context.Background(), watcher.ID, "owner", "testrepo", false, true)
	if err != nil {
		t.Fatalf("SetSubscription failed: %v", err)
	}

	if sub.Subscribed {
		t.Error("expected subscribed to be false")
	}
	if !sub.Ignored {
		t.Error("expected ignored to be true")
	}
}

func TestService_SetSubscription_Update(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	watcher := createTestUser(t, store, "watcher", "watcher@example.com")
	repo := createTestRepo(t, store, owner, "testrepo")

	// First subscribe
	_, _ = service.SetSubscription(context.Background(), watcher.ID, "owner", "testrepo", true, false)

	// Then update to ignore
	sub, err := service.SetSubscription(context.Background(), watcher.ID, "owner", "testrepo", false, true)
	if err != nil {
		t.Fatalf("SetSubscription update failed: %v", err)
	}

	if sub.Subscribed {
		t.Error("expected subscribed to be false after update")
	}
	if !sub.Ignored {
		t.Error("expected ignored to be true after update")
	}

	// Counter should be decremented
	updatedRepo, _ := store.Repos().GetByID(context.Background(), repo.ID)
	if updatedRepo.WatchersCount != 0 {
		t.Errorf("expected watchers_count 0, got %d", updatedRepo.WatchersCount)
	}
}

func TestService_SetSubscription_RepoNotFound(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	watcher := createTestUser(t, store, "watcher", "watcher@example.com")

	_, err := service.SetSubscription(context.Background(), watcher.ID, "unknown", "repo", true, false)
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

// Get Subscription Tests

func TestService_GetSubscription_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	watcher := createTestUser(t, store, "watcher", "watcher@example.com")
	createTestRepo(t, store, owner, "testrepo")

	// Set subscription first
	_, _ = service.SetSubscription(context.Background(), watcher.ID, "owner", "testrepo", true, false)

	sub, err := service.GetSubscription(context.Background(), watcher.ID, "owner", "testrepo")
	if err != nil {
		t.Fatalf("GetSubscription failed: %v", err)
	}

	if !sub.Subscribed {
		t.Error("expected subscribed to be true")
	}
}

func TestService_GetSubscription_NotFound(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	watcher := createTestUser(t, store, "watcher", "watcher@example.com")
	createTestRepo(t, store, owner, "testrepo")

	_, err := service.GetSubscription(context.Background(), watcher.ID, "owner", "testrepo")
	if err != watches.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// Delete Subscription Tests

func TestService_DeleteSubscription_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	watcher := createTestUser(t, store, "watcher", "watcher@example.com")
	repo := createTestRepo(t, store, owner, "testrepo")

	// Subscribe first
	_, _ = service.SetSubscription(context.Background(), watcher.ID, "owner", "testrepo", true, false)

	// Delete subscription
	err := service.DeleteSubscription(context.Background(), watcher.ID, "owner", "testrepo")
	if err != nil {
		t.Fatalf("DeleteSubscription failed: %v", err)
	}

	// Verify not subscribed
	_, err = service.GetSubscription(context.Background(), watcher.ID, "owner", "testrepo")
	if err != watches.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}

	// Counter should be decremented
	updatedRepo, _ := store.Repos().GetByID(context.Background(), repo.ID)
	if updatedRepo.WatchersCount != 0 {
		t.Errorf("expected watchers_count 0, got %d", updatedRepo.WatchersCount)
	}
}

func TestService_DeleteSubscription_NotSubscribed(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	watcher := createTestUser(t, store, "watcher", "watcher@example.com")
	createTestRepo(t, store, owner, "testrepo")

	// Delete without subscribing should be idempotent
	err := service.DeleteSubscription(context.Background(), watcher.ID, "owner", "testrepo")
	if err != nil {
		t.Fatalf("DeleteSubscription should succeed (idempotent): %v", err)
	}
}

// List Watchers Tests

func TestService_ListWatchers(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	watcher1 := createTestUser(t, store, "watcher1", "watcher1@example.com")
	watcher2 := createTestUser(t, store, "watcher2", "watcher2@example.com")
	createTestRepo(t, store, owner, "testrepo")

	_, _ = service.SetSubscription(context.Background(), watcher1.ID, "owner", "testrepo", true, false)
	_, _ = service.SetSubscription(context.Background(), watcher2.ID, "owner", "testrepo", true, false)

	watchers, err := service.ListWatchers(context.Background(), "owner", "testrepo", nil)
	if err != nil {
		t.Fatalf("ListWatchers failed: %v", err)
	}

	if len(watchers) != 2 {
		t.Errorf("expected 2 watchers, got %d", len(watchers))
	}
}

func TestService_ListWatchers_Pagination(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")

	// Create multiple watchers
	for i := 0; i < 5; i++ {
		watcher := createTestUser(t, store, "watcher"+string(rune('a'+i)), "watcher"+string(rune('a'+i))+"@example.com")
		_, _ = service.SetSubscription(context.Background(), watcher.ID, "owner", "testrepo", true, false)
	}

	watchers, err := service.ListWatchers(context.Background(), "owner", "testrepo", &watches.ListOpts{
		Page:    1,
		PerPage: 2,
	})
	if err != nil {
		t.Fatalf("ListWatchers failed: %v", err)
	}

	if len(watchers) != 2 {
		t.Errorf("expected 2 watchers, got %d", len(watchers))
	}
}

func TestService_ListWatchers_RepoNotFound(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	_, err := service.ListWatchers(context.Background(), "unknown", "repo", nil)
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

// List Watched Repos Tests

func TestService_ListForUser(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	watcher := createTestUser(t, store, "watcher", "watcher@example.com")
	createTestRepo(t, store, owner, "repo1")
	createTestRepo(t, store, owner, "repo2")

	_, _ = service.SetSubscription(context.Background(), watcher.ID, "owner", "repo1", true, false)
	_, _ = service.SetSubscription(context.Background(), watcher.ID, "owner", "repo2", true, false)

	watched, err := service.ListForUser(context.Background(), "watcher", nil)
	if err != nil {
		t.Fatalf("ListForUser failed: %v", err)
	}

	if len(watched) != 2 {
		t.Errorf("expected 2 watched repos, got %d", len(watched))
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
	watcher := createTestUser(t, store, "watcher", "watcher@example.com")
	createTestRepo(t, store, owner, "repo1")

	_, _ = service.SetSubscription(context.Background(), watcher.ID, "owner", "repo1", true, false)

	watched, err := service.ListForAuthenticatedUser(context.Background(), watcher.ID, nil)
	if err != nil {
		t.Fatalf("ListForAuthenticatedUser failed: %v", err)
	}

	if len(watched) != 1 {
		t.Errorf("expected 1 watched repo, got %d", len(watched))
	}
}

// URL Population Tests

func TestService_PopulateURLs(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	watcher := createTestUser(t, store, "watcher", "watcher@example.com")
	createTestRepo(t, store, owner, "testrepo")

	sub, _ := service.SetSubscription(context.Background(), watcher.ID, "owner", "testrepo", true, false)

	if sub.URL != "https://api.example.com/api/v3/repos/owner/testrepo/subscription" {
		t.Errorf("unexpected URL: %s", sub.URL)
	}
	if sub.RepositoryURL != "https://api.example.com/api/v3/repos/owner/testrepo" {
		t.Errorf("unexpected RepositoryURL: %s", sub.RepositoryURL)
	}
}

// Integration Test - Multiple Users Watching

func TestService_MultipleUsersWatching(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	repo := createTestRepo(t, store, owner, "testrepo")

	// Multiple users watch the repo
	for i := 0; i < 3; i++ {
		watcher := createTestUser(t, store, "watcher"+string(rune('a'+i)), "watcher"+string(rune('a'+i))+"@example.com")
		_, _ = service.SetSubscription(context.Background(), watcher.ID, "owner", "testrepo", true, false)
	}

	// Verify counter
	updatedRepo, _ := store.Repos().GetByID(context.Background(), repo.ID)
	if updatedRepo.WatchersCount != 3 {
		t.Errorf("expected watchers_count 3, got %d", updatedRepo.WatchersCount)
	}

	// Verify list
	watchers, _ := service.ListWatchers(context.Background(), "owner", "testrepo", nil)
	if len(watchers) != 3 {
		t.Errorf("expected 3 watchers, got %d", len(watchers))
	}
}
