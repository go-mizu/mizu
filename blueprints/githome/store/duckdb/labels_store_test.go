package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/labels"
	"github.com/oklog/ulid/v2"
)

func createTestLabel(t *testing.T, store *LabelsStore, repoID string) *labels.Label {
	t.Helper()
	id := ulid.Make().String()
	l := &labels.Label{
		ID:          id,
		RepoID:      repoID,
		Name:        "label-" + id[len(id)-8:],
		Color:       "ff0000",
		Description: "Test label description",
		CreatedAt:   time.Now(),
	}
	if err := store.Create(context.Background(), l); err != nil {
		t.Fatalf("failed to create test label: %v", err)
	}
	return l
}

// =============================================================================
// Label CRUD Tests
// =============================================================================

func TestLabelsStore_Create(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	labelsStore := NewLabelsStore(store.DB())

	l := &labels.Label{
		ID:          ulid.Make().String(),
		RepoID:      repoID,
		Name:        "bug",
		Color:       "d73a4a",
		Description: "Something isn't working",
		CreatedAt:   time.Now(),
	}

	err := labelsStore.Create(context.Background(), l)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := labelsStore.GetByID(context.Background(), l.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected label to be created")
	}
	if got.Name != "bug" {
		t.Errorf("got name %q, want %q", got.Name, "bug")
	}
	if got.Color != "d73a4a" {
		t.Errorf("got color %q, want %q", got.Color, "d73a4a")
	}
}

func TestLabelsStore_GetByID(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	labelsStore := NewLabelsStore(store.DB())

	l := createTestLabel(t, labelsStore, repoID)

	got, err := labelsStore.GetByID(context.Background(), l.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected label")
	}
	if got.ID != l.ID {
		t.Errorf("got ID %q, want %q", got.ID, l.ID)
	}
}

func TestLabelsStore_GetByID_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	labelsStore := NewLabelsStore(store.DB())

	got, err := labelsStore.GetByID(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent label")
	}
}

func TestLabelsStore_GetByName(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	labelsStore := NewLabelsStore(store.DB())

	l := &labels.Label{
		ID:        ulid.Make().String(),
		RepoID:    repoID,
		Name:      "enhancement",
		Color:     "a2eeef",
		CreatedAt: time.Now(),
	}
	labelsStore.Create(context.Background(), l)

	got, err := labelsStore.GetByName(context.Background(), repoID, "enhancement")
	if err != nil {
		t.Fatalf("GetByName failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected label")
	}
	if got.Name != "enhancement" {
		t.Errorf("got name %q, want %q", got.Name, "enhancement")
	}
}

func TestLabelsStore_GetByName_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	labelsStore := NewLabelsStore(store.DB())

	got, err := labelsStore.GetByName(context.Background(), repoID, "nonexistent")
	if err != nil {
		t.Fatalf("GetByName failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent label name")
	}
}

func TestLabelsStore_Update(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	labelsStore := NewLabelsStore(store.DB())

	l := createTestLabel(t, labelsStore, repoID)

	l.Name = "updated-name"
	l.Color = "0000ff"
	l.Description = "Updated description"

	err := labelsStore.Update(context.Background(), l)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, _ := labelsStore.GetByID(context.Background(), l.ID)
	if got.Name != "updated-name" {
		t.Errorf("got name %q, want %q", got.Name, "updated-name")
	}
	if got.Color != "0000ff" {
		t.Errorf("got color %q, want %q", got.Color, "0000ff")
	}
}

func TestLabelsStore_Delete(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	labelsStore := NewLabelsStore(store.DB())

	l := createTestLabel(t, labelsStore, repoID)

	err := labelsStore.Delete(context.Background(), l.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	got, _ := labelsStore.GetByID(context.Background(), l.ID)
	if got != nil {
		t.Error("expected label to be deleted")
	}
}

func TestLabelsStore_Delete_NonExistent(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	labelsStore := NewLabelsStore(store.DB())

	err := labelsStore.Delete(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("Delete should not error for non-existent label: %v", err)
	}
}

func TestLabelsStore_List(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	labelsStore := NewLabelsStore(store.DB())

	for i := 0; i < 5; i++ {
		createTestLabel(t, labelsStore, repoID)
	}

	list, err := labelsStore.List(context.Background(), repoID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 5 {
		t.Errorf("got %d labels, want 5", len(list))
	}
}

func TestLabelsStore_List_Empty(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	labelsStore := NewLabelsStore(store.DB())

	list, err := labelsStore.List(context.Background(), repoID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if list != nil && len(list) != 0 {
		t.Error("expected empty list")
	}
}

func TestLabelsStore_List_PerRepo(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())
	labelsStore := NewLabelsStore(store.DB())

	user := createTestUser(t, usersStore)
	repo1 := createTestRepo(t, reposStore, user.ID)
	repo2 := createTestRepo(t, reposStore, user.ID)

	// Create 3 labels in repo1
	for i := 0; i < 3; i++ {
		createTestLabel(t, labelsStore, repo1.ID)
	}

	// Create 2 labels in repo2
	for i := 0; i < 2; i++ {
		createTestLabel(t, labelsStore, repo2.ID)
	}

	list1, _ := labelsStore.List(context.Background(), repo1.ID)
	list2, _ := labelsStore.List(context.Background(), repo2.ID)

	if len(list1) != 3 {
		t.Errorf("got %d labels for repo1, want 3", len(list1))
	}
	if len(list2) != 2 {
		t.Errorf("got %d labels for repo2, want 2", len(list2))
	}
}

func TestLabelsStore_ListByIDs(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	labelsStore := NewLabelsStore(store.DB())

	l1 := createTestLabel(t, labelsStore, repoID)
	l2 := createTestLabel(t, labelsStore, repoID)
	_ = createTestLabel(t, labelsStore, repoID) // l3 not in query

	list, err := labelsStore.ListByIDs(context.Background(), []string{l1.ID, l2.ID})
	if err != nil {
		t.Fatalf("ListByIDs failed: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("got %d labels, want 2", len(list))
	}
}

func TestLabelsStore_ListByIDs_Empty(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	labelsStore := NewLabelsStore(store.DB())

	list, err := labelsStore.ListByIDs(context.Background(), []string{})
	if err != nil {
		t.Fatalf("ListByIDs failed: %v", err)
	}
	if list != nil && len(list) != 0 {
		t.Error("expected empty list for empty IDs")
	}
}

// Verify interface compliance
var _ labels.Store = (*LabelsStore)(nil)
