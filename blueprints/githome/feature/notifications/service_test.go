package notifications_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"github.com/go-mizu/blueprints/githome/feature/notifications"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
	"github.com/go-mizu/blueprints/githome/store/duckdb"
)

func setupTestService(t *testing.T) (*notifications.Service, *duckdb.Store, func()) {
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

	notificationsStore := duckdb.NewNotificationsStore(db)
	service := notifications.NewService(notificationsStore, store.Repos(), "https://api.example.com")

	cleanup := func() {
		store.Close()
	}

	return service, store, cleanup
}

func createTestUser(t *testing.T, store *duckdb.Store, login string) *users.User {
	t.Helper()
	userService := users.NewService(store.Users(), "https://api.example.com")
	user, err := userService.Create(context.Background(), &users.CreateIn{
		Login:    login,
		Email:    login + "@example.com",
		Password: "password123",
		Name:     "Test User",
	})
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return user
}

func createTestRepo(t *testing.T, store *duckdb.Store, ownerID int64, name string) *repos.Repository {
	t.Helper()
	orgsStore := duckdb.NewOrgsStore(store.DB())
	repoService := repos.NewService(store.Repos(), store.Users(), orgsStore, "https://api.example.com", "")
	repo, err := repoService.Create(context.Background(), ownerID, &repos.CreateIn{
		Name:        name,
		Description: "Test repository",
	})
	if err != nil {
		t.Fatalf("failed to create test repo: %v", err)
	}
	return repo
}

// Create Tests

func TestService_Create_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testuser")
	repo := createTestRepo(t, store, user.ID, "testrepo")

	n, err := service.Create(context.Background(), user.ID, repo.ID, "assign", "Issue", "Test Issue #1", "https://api.example.com/repos/testuser/testrepo/issues/1")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if n.ID == "" {
		t.Error("expected ID to be assigned")
	}
	if !n.Unread {
		t.Error("expected notification to be unread")
	}
	if n.Reason != "assign" {
		t.Errorf("got reason %q, want assign", n.Reason)
	}
	if n.Subject == nil {
		t.Fatal("expected Subject to be set")
	}
	if n.Subject.Title != "Test Issue #1" {
		t.Errorf("got subject title %q, want Test Issue #1", n.Subject.Title)
	}
	if n.Subject.Type != "Issue" {
		t.Errorf("got subject type %q, want Issue", n.Subject.Type)
	}
	if n.Repository == nil {
		t.Fatal("expected Repository to be set")
	}
	if n.Repository.ID != repo.ID {
		t.Errorf("got repo ID %d, want %d", n.Repository.ID, repo.ID)
	}
}

func TestService_Create_NonExistentRepo(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testuser")

	_, err := service.Create(context.Background(), user.ID, 9999, "assign", "Issue", "Test Issue", "https://example.com")
	if err != repos.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_Create_PopulatesURLs(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testuser")
	repo := createTestRepo(t, store, user.ID, "testrepo")

	n, err := service.Create(context.Background(), user.ID, repo.ID, "mention", "PullRequest", "Fix bug", "https://api.example.com/repos/testuser/testrepo/pulls/1")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if n.URL == "" {
		t.Error("expected URL to be populated")
	}
	if n.SubscriptionURL == "" {
		t.Error("expected SubscriptionURL to be populated")
	}
	if n.Repository.URL == "" {
		t.Error("expected Repository.URL to be populated")
	}
	if n.Repository.HTMLURL == "" {
		t.Error("expected Repository.HTMLURL to be populated")
	}
}

// List Tests

func TestService_List_EmptyList(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testuser")

	list, err := service.List(context.Background(), user.ID, nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != 0 {
		t.Errorf("expected empty list, got %d items", len(list))
	}
}

func TestService_List_UnreadOnly(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testuser")
	repo := createTestRepo(t, store, user.ID, "testrepo")

	// Create multiple notifications
	for i := 1; i <= 3; i++ {
		_, err := service.Create(context.Background(), user.ID, repo.ID, "mention", "Issue", "Issue", "https://example.com")
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	// List only unread (default)
	list, err := service.List(context.Background(), user.ID, nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != 3 {
		t.Errorf("expected 3 notifications, got %d", len(list))
	}
}

func TestService_List_AllNotifications(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testuser")
	repo := createTestRepo(t, store, user.ID, "testrepo")

	// Create notifications
	n1, _ := service.Create(context.Background(), user.ID, repo.ID, "mention", "Issue", "Issue 1", "https://example.com")
	service.Create(context.Background(), user.ID, repo.ID, "assign", "Issue", "Issue 2", "https://example.com")

	// Mark one as read
	service.MarkThreadAsRead(context.Background(), user.ID, n1.ID)

	// List all (including read)
	list, err := service.List(context.Background(), user.ID, &notifications.ListOpts{All: true})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("expected 2 notifications with All=true, got %d", len(list))
	}

	// List unread only
	unreadList, err := service.List(context.Background(), user.ID, nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(unreadList) != 1 {
		t.Errorf("expected 1 unread notification, got %d", len(unreadList))
	}
}

func TestService_List_Pagination(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testuser")
	repo := createTestRepo(t, store, user.ID, "testrepo")

	// Create 5 notifications
	for i := 1; i <= 5; i++ {
		service.Create(context.Background(), user.ID, repo.ID, "mention", "Issue", "Issue", "https://example.com")
	}

	// List with pagination
	list, err := service.List(context.Background(), user.ID, &notifications.ListOpts{PerPage: 2, Page: 1})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("expected 2 notifications with PerPage=2, got %d", len(list))
	}
}

func TestService_List_PerPageMax(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testuser")

	// Request more than max - should succeed even with PerPage > 100 (capped internally)
	_, err := service.List(context.Background(), user.ID, &notifications.ListOpts{PerPage: 200})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	// Success - PerPage was capped to 100 internally
}

func TestService_List_PopulatesURLs(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testuser")
	repo := createTestRepo(t, store, user.ID, "testrepo")

	service.Create(context.Background(), user.ID, repo.ID, "mention", "Issue", "Issue 1", "https://example.com")

	list, err := service.List(context.Background(), user.ID, nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) == 0 {
		t.Fatal("expected at least one notification")
	}

	n := list[0]
	if n.URL == "" {
		t.Error("expected URL to be populated")
	}
	if n.SubscriptionURL == "" {
		t.Error("expected SubscriptionURL to be populated")
	}
}

// ListForRepo Tests

func TestService_ListForRepo_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testuser")
	repo1 := createTestRepo(t, store, user.ID, "repo1")
	repo2 := createTestRepo(t, store, user.ID, "repo2")

	// Create notifications for different repos
	service.Create(context.Background(), user.ID, repo1.ID, "mention", "Issue", "Issue in repo1", "https://example.com")
	service.Create(context.Background(), user.ID, repo1.ID, "assign", "Issue", "Another in repo1", "https://example.com")
	service.Create(context.Background(), user.ID, repo2.ID, "mention", "Issue", "Issue in repo2", "https://example.com")

	// List for repo1
	list, err := service.ListForRepo(context.Background(), user.ID, "testuser", "repo1", nil)
	if err != nil {
		t.Fatalf("ListForRepo failed: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("expected 2 notifications for repo1, got %d", len(list))
	}
}

func TestService_ListForRepo_NonExistentRepo(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testuser")

	_, err := service.ListForRepo(context.Background(), user.ID, "testuser", "nonexistent", nil)
	if err != repos.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_ListForRepo_Isolation(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user1 := createTestUser(t, store, "user1")
	user2 := createTestUser(t, store, "user2")
	repo := createTestRepo(t, store, user1.ID, "repo")

	// Create notification for user1
	service.Create(context.Background(), user1.ID, repo.ID, "mention", "Issue", "Issue 1", "https://example.com")

	// User2 should see no notifications
	list, err := service.ListForRepo(context.Background(), user2.ID, "user1", "repo", nil)
	if err != nil {
		t.Fatalf("ListForRepo failed: %v", err)
	}

	if len(list) != 0 {
		t.Errorf("expected 0 notifications for user2, got %d", len(list))
	}
}

// MarkAsRead Tests

func TestService_MarkAsRead_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testuser")
	repo := createTestRepo(t, store, user.ID, "testrepo")

	// Create notifications
	service.Create(context.Background(), user.ID, repo.ID, "mention", "Issue", "Issue 1", "https://example.com")
	service.Create(context.Background(), user.ID, repo.ID, "assign", "Issue", "Issue 2", "https://example.com")

	// Mark all as read
	err := service.MarkAsRead(context.Background(), user.ID, time.Time{})
	if err != nil {
		t.Fatalf("MarkAsRead failed: %v", err)
	}

	// List unread - should be empty
	list, err := service.List(context.Background(), user.ID, nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != 0 {
		t.Errorf("expected 0 unread notifications, got %d", len(list))
	}
}

func TestService_MarkAsRead_WithTimestamp(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testuser")
	repo := createTestRepo(t, store, user.ID, "testrepo")

	service.Create(context.Background(), user.ID, repo.ID, "mention", "Issue", "Issue 1", "https://example.com")

	timestamp := time.Now()
	err := service.MarkAsRead(context.Background(), user.ID, timestamp)
	if err != nil {
		t.Fatalf("MarkAsRead failed: %v", err)
	}

	// Verify it's marked as read
	list, err := service.List(context.Background(), user.ID, nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != 0 {
		t.Errorf("expected 0 unread notifications, got %d", len(list))
	}
}

// MarkRepoAsRead Tests

func TestService_MarkRepoAsRead_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testuser")
	repo1 := createTestRepo(t, store, user.ID, "repo1")
	repo2 := createTestRepo(t, store, user.ID, "repo2")

	// Create notifications for both repos
	service.Create(context.Background(), user.ID, repo1.ID, "mention", "Issue", "In repo1", "https://example.com")
	service.Create(context.Background(), user.ID, repo2.ID, "mention", "Issue", "In repo2", "https://example.com")

	// Mark only repo1 as read
	err := service.MarkRepoAsRead(context.Background(), user.ID, "testuser", "repo1", time.Time{})
	if err != nil {
		t.Fatalf("MarkRepoAsRead failed: %v", err)
	}

	// Repo1 should have no unread
	list1, _ := service.ListForRepo(context.Background(), user.ID, "testuser", "repo1", nil)
	if len(list1) != 0 {
		t.Errorf("expected 0 unread in repo1, got %d", len(list1))
	}

	// Repo2 should still have 1 unread
	list2, _ := service.ListForRepo(context.Background(), user.ID, "testuser", "repo2", nil)
	if len(list2) != 1 {
		t.Errorf("expected 1 unread in repo2, got %d", len(list2))
	}
}

func TestService_MarkRepoAsRead_NonExistentRepo(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testuser")

	err := service.MarkRepoAsRead(context.Background(), user.ID, "testuser", "nonexistent", time.Time{})
	if err != repos.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// GetThread Tests

func TestService_GetThread_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testuser")
	repo := createTestRepo(t, store, user.ID, "testrepo")

	created, _ := service.Create(context.Background(), user.ID, repo.ID, "mention", "Issue", "Test Issue", "https://example.com")

	n, err := service.GetThread(context.Background(), user.ID, created.ID)
	if err != nil {
		t.Fatalf("GetThread failed: %v", err)
	}

	if n.ID != created.ID {
		t.Errorf("got ID %q, want %q", n.ID, created.ID)
	}
	if n.Subject.Title != "Test Issue" {
		t.Errorf("got title %q, want Test Issue", n.Subject.Title)
	}
}

func TestService_GetThread_NotFound(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testuser")

	_, err := service.GetThread(context.Background(), user.ID, "nonexistent")
	if err != notifications.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_GetThread_WrongUser(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user1 := createTestUser(t, store, "user1")
	user2 := createTestUser(t, store, "user2")
	repo := createTestRepo(t, store, user1.ID, "repo")

	created, _ := service.Create(context.Background(), user1.ID, repo.ID, "mention", "Issue", "Test Issue", "https://example.com")

	// User2 trying to access user1's notification
	_, err := service.GetThread(context.Background(), user2.ID, created.ID)
	if err != notifications.ErrNotFound {
		t.Errorf("expected ErrNotFound for wrong user, got %v", err)
	}
}

func TestService_GetThread_PopulatesURLs(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testuser")
	repo := createTestRepo(t, store, user.ID, "testrepo")

	created, _ := service.Create(context.Background(), user.ID, repo.ID, "mention", "Issue", "Test Issue", "https://example.com")

	n, err := service.GetThread(context.Background(), user.ID, created.ID)
	if err != nil {
		t.Fatalf("GetThread failed: %v", err)
	}

	if n.URL == "" {
		t.Error("expected URL to be populated")
	}
	if n.SubscriptionURL == "" {
		t.Error("expected SubscriptionURL to be populated")
	}
}

// MarkThreadAsRead Tests

func TestService_MarkThreadAsRead_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testuser")
	repo := createTestRepo(t, store, user.ID, "testrepo")

	created, _ := service.Create(context.Background(), user.ID, repo.ID, "mention", "Issue", "Test Issue", "https://example.com")

	err := service.MarkThreadAsRead(context.Background(), user.ID, created.ID)
	if err != nil {
		t.Fatalf("MarkThreadAsRead failed: %v", err)
	}

	// Verify it's marked as read
	n, _ := service.GetThread(context.Background(), user.ID, created.ID)
	if n.Unread {
		t.Error("expected notification to be read")
	}
}

func TestService_MarkThreadAsRead_NotFound(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testuser")

	err := service.MarkThreadAsRead(context.Background(), user.ID, "nonexistent")
	if err != notifications.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_MarkThreadAsRead_Idempotent(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testuser")
	repo := createTestRepo(t, store, user.ID, "testrepo")

	created, _ := service.Create(context.Background(), user.ID, repo.ID, "mention", "Issue", "Test Issue", "https://example.com")

	// Mark as read twice
	err := service.MarkThreadAsRead(context.Background(), user.ID, created.ID)
	if err != nil {
		t.Fatalf("First MarkThreadAsRead failed: %v", err)
	}

	err = service.MarkThreadAsRead(context.Background(), user.ID, created.ID)
	if err != nil {
		t.Fatalf("Second MarkThreadAsRead failed: %v", err)
	}
}

// MarkThreadAsDone Tests

func TestService_MarkThreadAsDone_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testuser")
	repo := createTestRepo(t, store, user.ID, "testrepo")

	created, _ := service.Create(context.Background(), user.ID, repo.ID, "mention", "Issue", "Test Issue", "https://example.com")

	err := service.MarkThreadAsDone(context.Background(), user.ID, created.ID)
	if err != nil {
		t.Fatalf("MarkThreadAsDone failed: %v", err)
	}

	// Verify it's removed
	_, err = service.GetThread(context.Background(), user.ID, created.ID)
	if err != notifications.ErrNotFound {
		t.Errorf("expected ErrNotFound after done, got %v", err)
	}
}

func TestService_MarkThreadAsDone_NotFound(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testuser")

	err := service.MarkThreadAsDone(context.Background(), user.ID, "nonexistent")
	if err != notifications.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// GetThreadSubscription Tests

func TestService_GetThreadSubscription_Default(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testuser")
	repo := createTestRepo(t, store, user.ID, "testrepo")

	created, _ := service.Create(context.Background(), user.ID, repo.ID, "mention", "Issue", "Test Issue", "https://example.com")

	sub, err := service.GetThreadSubscription(context.Background(), user.ID, created.ID)
	if err != nil {
		t.Fatalf("GetThreadSubscription failed: %v", err)
	}

	// Default subscription
	if !sub.Subscribed {
		t.Error("expected Subscribed to be true by default")
	}
	if sub.Ignored {
		t.Error("expected Ignored to be false by default")
	}
	if sub.URL == "" {
		t.Error("expected URL to be populated")
	}
	if sub.ThreadURL == "" {
		t.Error("expected ThreadURL to be populated")
	}
}

func TestService_GetThreadSubscription_NotFound(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testuser")

	_, err := service.GetThreadSubscription(context.Background(), user.ID, "nonexistent")
	if err != notifications.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// SetThreadSubscription Tests

func TestService_SetThreadSubscription_IgnoreTrue(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testuser")
	repo := createTestRepo(t, store, user.ID, "testrepo")

	created, _ := service.Create(context.Background(), user.ID, repo.ID, "mention", "Issue", "Test Issue", "https://example.com")

	sub, err := service.SetThreadSubscription(context.Background(), user.ID, created.ID, true)
	if err != nil {
		t.Fatalf("SetThreadSubscription failed: %v", err)
	}

	if !sub.Ignored {
		t.Error("expected Ignored to be true")
	}
	if !sub.Subscribed {
		t.Error("expected Subscribed to be true")
	}
}

func TestService_SetThreadSubscription_IgnoreFalse(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testuser")
	repo := createTestRepo(t, store, user.ID, "testrepo")

	created, _ := service.Create(context.Background(), user.ID, repo.ID, "mention", "Issue", "Test Issue", "https://example.com")

	// First set to ignored
	service.SetThreadSubscription(context.Background(), user.ID, created.ID, true)

	// Then set to not ignored
	sub, err := service.SetThreadSubscription(context.Background(), user.ID, created.ID, false)
	if err != nil {
		t.Fatalf("SetThreadSubscription failed: %v", err)
	}

	if sub.Ignored {
		t.Error("expected Ignored to be false")
	}
}

func TestService_SetThreadSubscription_NotFound(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testuser")

	_, err := service.SetThreadSubscription(context.Background(), user.ID, "nonexistent", true)
	if err != notifications.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_SetThreadSubscription_PopulatesURLs(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testuser")
	repo := createTestRepo(t, store, user.ID, "testrepo")

	created, _ := service.Create(context.Background(), user.ID, repo.ID, "mention", "Issue", "Test Issue", "https://example.com")

	sub, err := service.SetThreadSubscription(context.Background(), user.ID, created.ID, false)
	if err != nil {
		t.Fatalf("SetThreadSubscription failed: %v", err)
	}

	if sub.URL == "" {
		t.Error("expected URL to be populated")
	}
	if sub.ThreadURL == "" {
		t.Error("expected ThreadURL to be populated")
	}
}

// DeleteThreadSubscription Tests

func TestService_DeleteThreadSubscription_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testuser")
	repo := createTestRepo(t, store, user.ID, "testrepo")

	created, _ := service.Create(context.Background(), user.ID, repo.ID, "mention", "Issue", "Test Issue", "https://example.com")

	// Set subscription first
	service.SetThreadSubscription(context.Background(), user.ID, created.ID, true)

	// Delete it
	err := service.DeleteThreadSubscription(context.Background(), user.ID, created.ID)
	if err != nil {
		t.Fatalf("DeleteThreadSubscription failed: %v", err)
	}

	// Get subscription should return default again
	sub, err := service.GetThreadSubscription(context.Background(), user.ID, created.ID)
	if err != nil {
		t.Fatalf("GetThreadSubscription failed: %v", err)
	}

	// Should have default values (subscribed=true, ignored=false)
	if !sub.Subscribed {
		t.Error("expected default Subscribed=true after delete")
	}
	if sub.Ignored {
		t.Error("expected default Ignored=false after delete")
	}
}

func TestService_DeleteThreadSubscription_NotFound(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testuser")

	err := service.DeleteThreadSubscription(context.Background(), user.ID, "nonexistent")
	if err != notifications.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// User Isolation Tests

func TestService_UserIsolation(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user1 := createTestUser(t, store, "user1")
	user2 := createTestUser(t, store, "user2")
	repo := createTestRepo(t, store, user1.ID, "repo")

	// Create notifications for both users
	n1, _ := service.Create(context.Background(), user1.ID, repo.ID, "mention", "Issue", "For user1", "https://example.com")
	n2, _ := service.Create(context.Background(), user2.ID, repo.ID, "assign", "Issue", "For user2", "https://example.com")

	// User1 should only see their notification
	list1, _ := service.List(context.Background(), user1.ID, nil)
	if len(list1) != 1 {
		t.Errorf("user1 expected 1 notification, got %d", len(list1))
	}
	if list1[0].ID != n1.ID {
		t.Error("user1 got wrong notification")
	}

	// User2 should only see their notification
	list2, _ := service.List(context.Background(), user2.ID, nil)
	if len(list2) != 1 {
		t.Errorf("user2 expected 1 notification, got %d", len(list2))
	}
	if list2[0].ID != n2.ID {
		t.Error("user2 got wrong notification")
	}
}

// Various Reason Types Tests

func TestService_Create_DifferentReasons(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testuser")
	repo := createTestRepo(t, store, user.ID, "testrepo")

	reasons := []string{"assign", "author", "comment", "invitation", "manual", "mention", "review_requested", "security_alert", "state_change", "subscribed", "team_mention"}

	for _, reason := range reasons {
		n, err := service.Create(context.Background(), user.ID, repo.ID, reason, "Issue", "Test", "https://example.com")
		if err != nil {
			t.Errorf("Create with reason %q failed: %v", reason, err)
		}
		if n.Reason != reason {
			t.Errorf("got reason %q, want %q", n.Reason, reason)
		}
	}
}

// Subject Types Tests

func TestService_Create_DifferentSubjectTypes(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testuser")
	repo := createTestRepo(t, store, user.ID, "testrepo")

	types := []string{"Issue", "PullRequest", "Commit", "Release"}

	for _, subjectType := range types {
		n, err := service.Create(context.Background(), user.ID, repo.ID, "mention", subjectType, "Test", "https://example.com")
		if err != nil {
			t.Errorf("Create with subject type %q failed: %v", subjectType, err)
		}
		if n.Subject.Type != subjectType {
			t.Errorf("got subject type %q, want %q", n.Subject.Type, subjectType)
		}
	}
}
