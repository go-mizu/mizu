package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/activities"
	"github.com/oklog/ulid/v2"
)

func createTestActivity(t *testing.T, store *ActivitiesStore, actorID, repoID, eventType string) *activities.Activity {
	t.Helper()
	id := ulid.Make().String()
	a := &activities.Activity{
		ID:        id,
		ActorID:   actorID,
		RepoID:    repoID,
		EventType: eventType,
		Payload:   `{"test": true}`,
		IsPublic:  true,
		CreatedAt: time.Now(),
	}
	if err := store.Create(context.Background(), a); err != nil {
		t.Fatalf("failed to create test activity: %v", err)
	}
	return a
}

// =============================================================================
// Activity CRUD Tests
// =============================================================================

func TestActivitiesStore_Create(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	activitiesStore := NewActivitiesStore(store.DB())

	a := &activities.Activity{
		ID:        ulid.Make().String(),
		ActorID:   userID,
		RepoID:    repoID,
		EventType: "push",
		Payload:   `{"commits": 3}`,
		IsPublic:  true,
		CreatedAt: time.Now(),
	}

	err := activitiesStore.Create(context.Background(), a)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := activitiesStore.GetByID(context.Background(), a.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected activity to be created")
	}
	if got.EventType != "push" {
		t.Errorf("got event_type %q, want %q", got.EventType, "push")
	}
}

func TestActivitiesStore_GetByID(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	activitiesStore := NewActivitiesStore(store.DB())

	a := createTestActivity(t, activitiesStore, userID, repoID, "star")

	got, err := activitiesStore.GetByID(context.Background(), a.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected activity")
	}
	if got.ID != a.ID {
		t.Errorf("got ID %q, want %q", got.ID, a.ID)
	}
}

func TestActivitiesStore_GetByID_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	activitiesStore := NewActivitiesStore(store.DB())

	got, err := activitiesStore.GetByID(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent activity")
	}
}

func TestActivitiesStore_Delete(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	activitiesStore := NewActivitiesStore(store.DB())

	a := createTestActivity(t, activitiesStore, userID, repoID, "push")

	err := activitiesStore.Delete(context.Background(), a.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	got, _ := activitiesStore.GetByID(context.Background(), a.ID)
	if got != nil {
		t.Error("expected activity to be deleted")
	}
}

// =============================================================================
// ListByUser Tests
// =============================================================================

func TestActivitiesStore_ListByUser(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	activitiesStore := NewActivitiesStore(store.DB())

	for i := 0; i < 5; i++ {
		createTestActivity(t, activitiesStore, userID, repoID, "push")
	}

	list, err := activitiesStore.ListByUser(context.Background(), userID, 10, 0)
	if err != nil {
		t.Fatalf("ListByUser failed: %v", err)
	}
	if len(list) != 5 {
		t.Errorf("got %d activities, want 5", len(list))
	}
}

func TestActivitiesStore_ListByUser_Pagination(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	activitiesStore := NewActivitiesStore(store.DB())

	for i := 0; i < 10; i++ {
		createTestActivity(t, activitiesStore, userID, repoID, "push")
	}

	page1, _ := activitiesStore.ListByUser(context.Background(), userID, 3, 0)
	page2, _ := activitiesStore.ListByUser(context.Background(), userID, 3, 3)

	if len(page1) != 3 {
		t.Errorf("got %d activities on page 1, want 3", len(page1))
	}
	if len(page2) != 3 {
		t.Errorf("got %d activities on page 2, want 3", len(page2))
	}
	if page1[0].ID == page2[0].ID {
		t.Error("expected different activities on different pages")
	}
}

func TestActivitiesStore_ListByUser_Empty(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	_, userID := createRepoAndUser(t, store)
	activitiesStore := NewActivitiesStore(store.DB())

	list, err := activitiesStore.ListByUser(context.Background(), userID, 10, 0)
	if err != nil {
		t.Fatalf("ListByUser failed: %v", err)
	}
	if list != nil && len(list) != 0 {
		t.Error("expected empty list")
	}
}

// =============================================================================
// ListByRepo Tests
// =============================================================================

func TestActivitiesStore_ListByRepo(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	activitiesStore := NewActivitiesStore(store.DB())

	for i := 0; i < 5; i++ {
		createTestActivity(t, activitiesStore, userID, repoID, "push")
	}

	list, err := activitiesStore.ListByRepo(context.Background(), repoID, 10, 0)
	if err != nil {
		t.Fatalf("ListByRepo failed: %v", err)
	}
	if len(list) != 5 {
		t.Errorf("got %d activities, want 5", len(list))
	}
}

func TestActivitiesStore_ListByRepo_PerRepo(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())
	activitiesStore := NewActivitiesStore(store.DB())

	user := createTestUser(t, usersStore)
	repo1 := createTestRepo(t, reposStore, user.ID)
	repo2 := createTestRepo(t, reposStore, user.ID)

	for i := 0; i < 3; i++ {
		createTestActivity(t, activitiesStore, user.ID, repo1.ID, "push")
	}
	for i := 0; i < 2; i++ {
		createTestActivity(t, activitiesStore, user.ID, repo2.ID, "push")
	}

	list1, _ := activitiesStore.ListByRepo(context.Background(), repo1.ID, 10, 0)
	list2, _ := activitiesStore.ListByRepo(context.Background(), repo2.ID, 10, 0)

	if len(list1) != 3 {
		t.Errorf("got %d activities for repo1, want 3", len(list1))
	}
	if len(list2) != 2 {
		t.Errorf("got %d activities for repo2, want 2", len(list2))
	}
}

// =============================================================================
// ListPublic Tests
// =============================================================================

func TestActivitiesStore_ListPublic(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	activitiesStore := NewActivitiesStore(store.DB())

	// Create public activities
	for i := 0; i < 3; i++ {
		createTestActivity(t, activitiesStore, userID, repoID, "push")
	}

	// Create private activity
	privateActivity := &activities.Activity{
		ID:        ulid.Make().String(),
		ActorID:   userID,
		RepoID:    repoID,
		EventType: "push",
		Payload:   `{}`,
		IsPublic:  false,
		CreatedAt: time.Now(),
	}
	activitiesStore.Create(context.Background(), privateActivity)

	list, err := activitiesStore.ListPublic(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("ListPublic failed: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("got %d public activities, want 3", len(list))
	}
}

func TestActivitiesStore_ListPublic_Pagination(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	activitiesStore := NewActivitiesStore(store.DB())

	for i := 0; i < 10; i++ {
		createTestActivity(t, activitiesStore, userID, repoID, "push")
	}

	page1, _ := activitiesStore.ListPublic(context.Background(), 3, 0)
	page2, _ := activitiesStore.ListPublic(context.Background(), 3, 3)

	if len(page1) != 3 {
		t.Errorf("got %d activities on page 1, want 3", len(page1))
	}
	if len(page2) != 3 {
		t.Errorf("got %d activities on page 2, want 3", len(page2))
	}
}

// Verify interface compliance
var _ activities.Store = (*ActivitiesStore)(nil)
