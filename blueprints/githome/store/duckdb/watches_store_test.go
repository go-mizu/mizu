package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/watches"
)

func createTestWatch(t *testing.T, store *WatchesStore, userID, repoID string) *watches.Watch {
	t.Helper()
	w := &watches.Watch{
		UserID:    userID,
		RepoID:    repoID,
		Level:     "watching",
		CreatedAt: time.Now(),
	}
	if err := store.Create(context.Background(), w); err != nil {
		t.Fatalf("failed to create test watch: %v", err)
	}
	return w
}

// =============================================================================
// Watch CRUD Tests
// =============================================================================

func TestWatchesStore_Create(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	watchesStore := NewWatchesStore(store.DB())

	w := &watches.Watch{
		UserID:    userID,
		RepoID:    repoID,
		Level:     "watching",
		CreatedAt: time.Now(),
	}

	err := watchesStore.Create(context.Background(), w)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := watchesStore.Get(context.Background(), userID, repoID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected watch to be created")
	}
	if got.Level != "watching" {
		t.Errorf("got level %q, want %q", got.Level, "watching")
	}
}

func TestWatchesStore_Create_WithDifferentLevels(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())
	watchesStore := NewWatchesStore(store.DB())

	user := createTestUser(t, usersStore)
	actorID := createActorForUser(t, store.DB(), user.ID)
	repo1 := createTestRepo(t, reposStore, actorID)
	repo2 := createTestRepo(t, reposStore, actorID)
	repo3 := createTestRepo(t, reposStore, actorID)

	levels := []string{"watching", "ignoring", "releases_only"}
	repos := []string{repo1.ID, repo2.ID, repo3.ID}

	for i, level := range levels {
		w := &watches.Watch{
			UserID:    user.ID,
			RepoID:    repos[i],
			Level:     level,
			CreatedAt: time.Now(),
		}
		if err := watchesStore.Create(context.Background(), w); err != nil {
			t.Fatalf("Create failed for level %s: %v", level, err)
		}
	}

	for i, level := range levels {
		got, err := watchesStore.Get(context.Background(), user.ID, repos[i])
		if err != nil {
			t.Fatalf("Get failed for repo %s: %v", repos[i], err)
		}
		if got == nil {
			t.Fatalf("expected watch for level %s, got nil", level)
		}
		if got.Level != level {
			t.Errorf("got level %q, want %q", got.Level, level)
		}
	}
}

func TestWatchesStore_Get(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	watchesStore := NewWatchesStore(store.DB())

	createTestWatch(t, watchesStore, userID, repoID)

	got, err := watchesStore.Get(context.Background(), userID, repoID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected watch")
	}
}

func TestWatchesStore_Get_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	watchesStore := NewWatchesStore(store.DB())

	got, err := watchesStore.Get(context.Background(), "user-nonexistent", "repo-nonexistent")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent watch")
	}
}

func TestWatchesStore_Update(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	watchesStore := NewWatchesStore(store.DB())

	w := createTestWatch(t, watchesStore, userID, repoID)

	w.Level = "releases_only"

	err := watchesStore.Update(context.Background(), w)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, _ := watchesStore.Get(context.Background(), userID, repoID)
	if got.Level != "releases_only" {
		t.Errorf("got level %q, want %q", got.Level, "releases_only")
	}
}

func TestWatchesStore_Delete(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	watchesStore := NewWatchesStore(store.DB())

	createTestWatch(t, watchesStore, userID, repoID)

	err := watchesStore.Delete(context.Background(), userID, repoID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	got, _ := watchesStore.Get(context.Background(), userID, repoID)
	if got != nil {
		t.Error("expected watch to be deleted")
	}
}

func TestWatchesStore_Delete_NonExistent(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	watchesStore := NewWatchesStore(store.DB())

	err := watchesStore.Delete(context.Background(), "user-nonexistent", "repo-nonexistent")
	if err != nil {
		t.Fatalf("Delete should not error for non-existent watch: %v", err)
	}
}

// =============================================================================
// ListByRepo Tests
// =============================================================================

func TestWatchesStore_ListByRepo(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	usersStore := NewUsersStore(store.DB())
	watchesStore := NewWatchesStore(store.DB())

	for i := 0; i < 5; i++ {
		user := createTestUser(t, usersStore)
		createTestWatch(t, watchesStore, user.ID, repoID)
	}

	list, _, err := watchesStore.ListByRepo(context.Background(), repoID, 10, 0)
	if err != nil {
		t.Fatalf("ListByRepo failed: %v", err)
	}
	if len(list) != 5 {
		t.Errorf("got %d watches, want 5", len(list))
	}
}

func TestWatchesStore_ListByRepo_Pagination(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	usersStore := NewUsersStore(store.DB())
	watchesStore := NewWatchesStore(store.DB())

	for i := 0; i < 10; i++ {
		user := createTestUser(t, usersStore)
		createTestWatch(t, watchesStore, user.ID, repoID)
	}

	page1, _, _ := watchesStore.ListByRepo(context.Background(), repoID, 3, 0)
	page2, _, _ := watchesStore.ListByRepo(context.Background(), repoID, 3, 3)

	if len(page1) != 3 {
		t.Errorf("got %d watches on page 1, want 3", len(page1))
	}
	if len(page2) != 3 {
		t.Errorf("got %d watches on page 2, want 3", len(page2))
	}
}

// =============================================================================
// ListByUser Tests
// =============================================================================

func TestWatchesStore_ListByUser(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())
	watchesStore := NewWatchesStore(store.DB())

	user := createTestUser(t, usersStore)

	for i := 0; i < 5; i++ {
		repo := createTestRepo(t, reposStore, user.ID)
		createTestWatch(t, watchesStore, user.ID, repo.ID)
	}

	list, _, err := watchesStore.ListByUser(context.Background(), user.ID, 10, 0)
	if err != nil {
		t.Fatalf("ListByUser failed: %v", err)
	}
	if len(list) != 5 {
		t.Errorf("got %d watches, want 5", len(list))
	}
}

func TestWatchesStore_ListByUser_Pagination(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())
	watchesStore := NewWatchesStore(store.DB())

	user := createTestUser(t, usersStore)

	for i := 0; i < 10; i++ {
		repo := createTestRepo(t, reposStore, user.ID)
		createTestWatch(t, watchesStore, user.ID, repo.ID)
	}

	page1, _, _ := watchesStore.ListByUser(context.Background(), user.ID, 3, 0)
	page2, _, _ := watchesStore.ListByUser(context.Background(), user.ID, 3, 3)

	if len(page1) != 3 {
		t.Errorf("got %d watches on page 1, want 3", len(page1))
	}
	if len(page2) != 3 {
		t.Errorf("got %d watches on page 2, want 3", len(page2))
	}
}

// =============================================================================
// Count Tests
// =============================================================================

func TestWatchesStore_Count(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	usersStore := NewUsersStore(store.DB())
	watchesStore := NewWatchesStore(store.DB())

	for i := 0; i < 7; i++ {
		user := createTestUser(t, usersStore)
		createTestWatch(t, watchesStore, user.ID, repoID)
	}

	count, err := watchesStore.Count(context.Background(), repoID)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 7 {
		t.Errorf("got count %d, want 7", count)
	}
}

func TestWatchesStore_Count_Empty(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	watchesStore := NewWatchesStore(store.DB())

	count, err := watchesStore.Count(context.Background(), repoID)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 0 {
		t.Errorf("got count %d, want 0", count)
	}
}

// Verify interface compliance
var _ watches.Store = (*WatchesStore)(nil)
