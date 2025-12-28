package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/notifications"
	"github.com/oklog/ulid/v2"
)

func createTestNotification(t *testing.T, store *NotificationsStore, userID, repoID string) *notifications.Notification {
	t.Helper()
	id := ulid.Make().String()
	n := &notifications.Notification{
		ID:         id,
		UserID:     userID,
		RepoID:     repoID,
		Type:       "issue",
		TargetType: "issue",
		TargetID:   "issue-" + id[len(id)-8:],
		Title:      "Test notification " + id[len(id)-8:],
		Reason:     "subscribed",
		Unread:     true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := store.Create(context.Background(), n); err != nil {
		t.Fatalf("failed to create test notification: %v", err)
	}
	return n
}

// =============================================================================
// Notification CRUD Tests
// =============================================================================

func TestNotificationsStore_Create(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	notificationsStore := NewNotificationsStore(store.DB())

	n := &notifications.Notification{
		ID:         ulid.Make().String(),
		UserID:     userID,
		RepoID:     repoID,
		Type:       "pull_request",
		TargetType: "pull_request",
		TargetID:   "pr-1",
		Title:      "New PR opened",
		Reason:     "mention",
		Unread:     true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	err := notificationsStore.Create(context.Background(), n)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := notificationsStore.GetByID(context.Background(), n.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected notification to be created")
	}
	if got.Type != "pull_request" {
		t.Errorf("got type %q, want %q", got.Type, "pull_request")
	}
	if got.Reason != "mention" {
		t.Errorf("got reason %q, want %q", got.Reason, "mention")
	}
}

func TestNotificationsStore_GetByID(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	notificationsStore := NewNotificationsStore(store.DB())

	n := createTestNotification(t, notificationsStore, userID, repoID)

	got, err := notificationsStore.GetByID(context.Background(), n.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected notification")
	}
	if got.ID != n.ID {
		t.Errorf("got ID %q, want %q", got.ID, n.ID)
	}
}

func TestNotificationsStore_GetByID_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	notificationsStore := NewNotificationsStore(store.DB())

	got, err := notificationsStore.GetByID(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent notification")
	}
}

func TestNotificationsStore_Delete(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	notificationsStore := NewNotificationsStore(store.DB())

	n := createTestNotification(t, notificationsStore, userID, repoID)

	err := notificationsStore.Delete(context.Background(), n.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	got, _ := notificationsStore.GetByID(context.Background(), n.ID)
	if got != nil {
		t.Error("expected notification to be deleted")
	}
}

// =============================================================================
// List Tests
// =============================================================================

func TestNotificationsStore_List(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	notificationsStore := NewNotificationsStore(store.DB())

	for i := 0; i < 5; i++ {
		createTestNotification(t, notificationsStore, userID, repoID)
	}

	list, _, err := notificationsStore.List(context.Background(), userID, true, 10, 0)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 5 {
		t.Errorf("got %d notifications, want 5", len(list))
	}
}

func TestNotificationsStore_List_FilterByUnread(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	notificationsStore := NewNotificationsStore(store.DB())

	// Create unread notifications
	for i := 0; i < 3; i++ {
		createTestNotification(t, notificationsStore, userID, repoID)
	}

	// Create read notifications
	for i := 0; i < 2; i++ {
		n := createTestNotification(t, notificationsStore, userID, repoID)
		notificationsStore.MarkAsRead(context.Background(), n.ID)
	}

	// List unread only
	unreadList, _, _ := notificationsStore.List(context.Background(), userID, true, 10, 0)
	if len(unreadList) != 3 {
		t.Errorf("got %d unread notifications, want 3", len(unreadList))
	}

	// List all
	allList, _, _ := notificationsStore.List(context.Background(), userID, false, 10, 0)
	if len(allList) != 5 {
		t.Errorf("got %d all notifications, want 5", len(allList))
	}
}

func TestNotificationsStore_List_Pagination(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	notificationsStore := NewNotificationsStore(store.DB())

	for i := 0; i < 10; i++ {
		createTestNotification(t, notificationsStore, userID, repoID)
	}

	page1, _, _ := notificationsStore.List(context.Background(), userID, false, 3, 0)
	page2, _, _ := notificationsStore.List(context.Background(), userID, false, 3, 3)

	if len(page1) != 3 {
		t.Errorf("got %d notifications on page 1, want 3", len(page1))
	}
	if len(page2) != 3 {
		t.Errorf("got %d notifications on page 2, want 3", len(page2))
	}
}

// =============================================================================
// Mark As Read Tests
// =============================================================================

func TestNotificationsStore_MarkAsRead(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	notificationsStore := NewNotificationsStore(store.DB())

	n := createTestNotification(t, notificationsStore, userID, repoID)

	if !n.Unread {
		t.Fatal("expected notification to be unread initially")
	}

	err := notificationsStore.MarkAsRead(context.Background(), n.ID)
	if err != nil {
		t.Fatalf("MarkAsRead failed: %v", err)
	}

	got, _ := notificationsStore.GetByID(context.Background(), n.ID)
	if got.Unread {
		t.Error("expected notification to be marked as read")
	}
	if got.LastReadAt == nil {
		t.Error("expected last_read_at to be set")
	}
}

func TestNotificationsStore_MarkAllAsRead(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	notificationsStore := NewNotificationsStore(store.DB())

	for i := 0; i < 5; i++ {
		createTestNotification(t, notificationsStore, userID, repoID)
	}

	err := notificationsStore.MarkAllAsRead(context.Background(), userID)
	if err != nil {
		t.Fatalf("MarkAllAsRead failed: %v", err)
	}

	unreadList, _, _ := notificationsStore.List(context.Background(), userID, true, 10, 0)
	if len(unreadList) != 0 {
		t.Errorf("expected 0 unread notifications, got %d", len(unreadList))
	}
}

func TestNotificationsStore_MarkRepoAsRead(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())
	notificationsStore := NewNotificationsStore(store.DB())

	user := createTestUser(t, usersStore)
	repo1 := createTestRepo(t, reposStore, user.ID)
	repo2 := createTestRepo(t, reposStore, user.ID)

	// Create notifications for repo1
	for i := 0; i < 3; i++ {
		createTestNotification(t, notificationsStore, user.ID, repo1.ID)
	}

	// Create notifications for repo2
	for i := 0; i < 2; i++ {
		createTestNotification(t, notificationsStore, user.ID, repo2.ID)
	}

	err := notificationsStore.MarkRepoAsRead(context.Background(), user.ID, repo1.ID)
	if err != nil {
		t.Fatalf("MarkRepoAsRead failed: %v", err)
	}

	// Repo1 notifications should be read
	list1, _, _ := notificationsStore.List(context.Background(), user.ID, true, 10, 0)
	// Only repo2 notifications should be unread
	if len(list1) != 2 {
		t.Errorf("expected 2 unread notifications, got %d", len(list1))
	}
}

// =============================================================================
// CountUnread Tests
// =============================================================================

func TestNotificationsStore_CountUnread(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	notificationsStore := NewNotificationsStore(store.DB())

	for i := 0; i < 7; i++ {
		createTestNotification(t, notificationsStore, userID, repoID)
	}

	count, err := notificationsStore.CountUnread(context.Background(), userID)
	if err != nil {
		t.Fatalf("CountUnread failed: %v", err)
	}
	if count != 7 {
		t.Errorf("got count %d, want 7", count)
	}
}

func TestNotificationsStore_CountUnread_AfterRead(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	notificationsStore := NewNotificationsStore(store.DB())

	for i := 0; i < 5; i++ {
		n := createTestNotification(t, notificationsStore, userID, repoID)
		if i < 2 {
			notificationsStore.MarkAsRead(context.Background(), n.ID)
		}
	}

	count, _ := notificationsStore.CountUnread(context.Background(), userID)
	if count != 3 {
		t.Errorf("got count %d, want 3", count)
	}
}

func TestNotificationsStore_CountUnread_Empty(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	_, userID := createRepoAndUser(t, store)
	notificationsStore := NewNotificationsStore(store.DB())

	count, err := notificationsStore.CountUnread(context.Background(), userID)
	if err != nil {
		t.Fatalf("CountUnread failed: %v", err)
	}
	if count != 0 {
		t.Errorf("got count %d, want 0", count)
	}
}

// Verify interface compliance
var _ notifications.Store = (*NotificationsStore)(nil)
