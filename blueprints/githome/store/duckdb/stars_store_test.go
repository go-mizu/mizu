package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/stars"
)

func createTestStar(t *testing.T, store *StarsStore, userID, repoID string) *stars.Star {
	t.Helper()
	s := &stars.Star{
		UserID:    userID,
		RepoID:    repoID,
		CreatedAt: time.Now(),
	}
	if err := store.Create(context.Background(), s); err != nil {
		t.Fatalf("failed to create test star: %v", err)
	}
	return s
}

// =============================================================================
// Star CRUD Tests
// =============================================================================

func TestStarsStore_Create(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	starsStore := NewStarsStore(store.DB())

	s := &stars.Star{
		UserID:    userID,
		RepoID:    repoID,
		CreatedAt: time.Now(),
	}

	err := starsStore.Create(context.Background(), s)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := starsStore.Get(context.Background(), userID, repoID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected star to be created")
	}
	if got.UserID != userID {
		t.Errorf("got user_id %q, want %q", got.UserID, userID)
	}
	if got.RepoID != repoID {
		t.Errorf("got repo_id %q, want %q", got.RepoID, repoID)
	}
}

func TestStarsStore_Get(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	starsStore := NewStarsStore(store.DB())

	createTestStar(t, starsStore, userID, repoID)

	got, err := starsStore.Get(context.Background(), userID, repoID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected star")
	}
}

func TestStarsStore_Get_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	starsStore := NewStarsStore(store.DB())

	got, err := starsStore.Get(context.Background(), "user-nonexistent", "repo-nonexistent")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent star")
	}
}

func TestStarsStore_Delete(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	starsStore := NewStarsStore(store.DB())

	createTestStar(t, starsStore, userID, repoID)

	err := starsStore.Delete(context.Background(), userID, repoID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	got, _ := starsStore.Get(context.Background(), userID, repoID)
	if got != nil {
		t.Error("expected star to be deleted")
	}
}

func TestStarsStore_Delete_NonExistent(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	starsStore := NewStarsStore(store.DB())

	err := starsStore.Delete(context.Background(), "user-nonexistent", "repo-nonexistent")
	if err != nil {
		t.Fatalf("Delete should not error for non-existent star: %v", err)
	}
}

// =============================================================================
// ListByRepo Tests
// =============================================================================

func TestStarsStore_ListByRepo(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	usersStore := NewUsersStore(store.DB())
	starsStore := NewStarsStore(store.DB())

	// Create multiple users who star the repo
	for i := 0; i < 5; i++ {
		user := createTestUser(t, usersStore)
		createTestStar(t, starsStore, user.ID, repoID)
	}

	list, _, err := starsStore.ListByRepo(context.Background(), repoID, 10, 0)
	if err != nil {
		t.Fatalf("ListByRepo failed: %v", err)
	}
	if len(list) != 5 {
		t.Errorf("got %d stars, want 5", len(list))
	}
}

func TestStarsStore_ListByRepo_Pagination(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	usersStore := NewUsersStore(store.DB())
	starsStore := NewStarsStore(store.DB())

	for i := 0; i < 10; i++ {
		user := createTestUser(t, usersStore)
		createTestStar(t, starsStore, user.ID, repoID)
	}

	page1, _, _ := starsStore.ListByRepo(context.Background(), repoID, 3, 0)
	page2, _, _ := starsStore.ListByRepo(context.Background(), repoID, 3, 3)

	if len(page1) != 3 {
		t.Errorf("got %d stars on page 1, want 3", len(page1))
	}
	if len(page2) != 3 {
		t.Errorf("got %d stars on page 2, want 3", len(page2))
	}
}

func TestStarsStore_ListByRepo_Empty(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	starsStore := NewStarsStore(store.DB())

	list, _, err := starsStore.ListByRepo(context.Background(), repoID, 10, 0)
	if err != nil {
		t.Fatalf("ListByRepo failed: %v", err)
	}
	if list != nil && len(list) != 0 {
		t.Error("expected empty list")
	}
}

// =============================================================================
// ListByUser Tests
// =============================================================================

func TestStarsStore_ListByUser(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())
	starsStore := NewStarsStore(store.DB())

	user := createTestUser(t, usersStore)

	// Create multiple repos and star them
	for i := 0; i < 5; i++ {
		repo := createTestRepo(t, reposStore, user.ID)
		createTestStar(t, starsStore, user.ID, repo.ID)
	}

	list, _, err := starsStore.ListByUser(context.Background(), user.ID, 10, 0)
	if err != nil {
		t.Fatalf("ListByUser failed: %v", err)
	}
	if len(list) != 5 {
		t.Errorf("got %d stars, want 5", len(list))
	}
}

func TestStarsStore_ListByUser_Pagination(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())
	starsStore := NewStarsStore(store.DB())

	user := createTestUser(t, usersStore)

	for i := 0; i < 10; i++ {
		repo := createTestRepo(t, reposStore, user.ID)
		createTestStar(t, starsStore, user.ID, repo.ID)
	}

	page1, _, _ := starsStore.ListByUser(context.Background(), user.ID, 3, 0)
	page2, _, _ := starsStore.ListByUser(context.Background(), user.ID, 3, 3)

	if len(page1) != 3 {
		t.Errorf("got %d stars on page 1, want 3", len(page1))
	}
	if len(page2) != 3 {
		t.Errorf("got %d stars on page 2, want 3", len(page2))
	}
}

// =============================================================================
// Count Tests
// =============================================================================

func TestStarsStore_Count(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	usersStore := NewUsersStore(store.DB())
	starsStore := NewStarsStore(store.DB())

	for i := 0; i < 7; i++ {
		user := createTestUser(t, usersStore)
		createTestStar(t, starsStore, user.ID, repoID)
	}

	count, err := starsStore.Count(context.Background(), repoID)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 7 {
		t.Errorf("got count %d, want 7", count)
	}
}

func TestStarsStore_Count_Empty(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	starsStore := NewStarsStore(store.DB())

	count, err := starsStore.Count(context.Background(), repoID)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 0 {
		t.Errorf("got count %d, want 0", count)
	}
}

func TestStarsStore_Count_PerRepo(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())
	starsStore := NewStarsStore(store.DB())

	user := createTestUser(t, usersStore)
	repo1 := createTestRepo(t, reposStore, user.ID)
	repo2 := createTestRepo(t, reposStore, user.ID)

	// Star repo1 with 5 users
	for i := 0; i < 5; i++ {
		u := createTestUser(t, usersStore)
		createTestStar(t, starsStore, u.ID, repo1.ID)
	}

	// Star repo2 with 3 users
	for i := 0; i < 3; i++ {
		u := createTestUser(t, usersStore)
		createTestStar(t, starsStore, u.ID, repo2.ID)
	}

	count1, _ := starsStore.Count(context.Background(), repo1.ID)
	count2, _ := starsStore.Count(context.Background(), repo2.ID)

	if count1 != 5 {
		t.Errorf("got count for repo1 %d, want 5", count1)
	}
	if count2 != 3 {
		t.Errorf("got count for repo2 %d, want 3", count2)
	}
}

// Verify interface compliance
var _ stars.Store = (*StarsStore)(nil)
