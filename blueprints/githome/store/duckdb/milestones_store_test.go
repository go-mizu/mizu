package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/milestones"
	"github.com/oklog/ulid/v2"
)

func createTestMilestone(t *testing.T, store *MilestonesStore, repoID string, number int) *milestones.Milestone {
	t.Helper()
	id := ulid.Make().String()
	m := &milestones.Milestone{
		ID:           id,
		RepoID:       repoID,
		Number:       number,
		Title:        "Milestone " + id[len(id)-8:],
		Description:  "Test milestone description",
		State:        "open",
		OpenIssues:   0,
		ClosedIssues: 0,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := store.Create(context.Background(), m); err != nil {
		t.Fatalf("failed to create test milestone: %v", err)
	}
	return m
}

// =============================================================================
// Milestone CRUD Tests
// =============================================================================

func TestMilestonesStore_Create(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	milestonesStore := NewMilestonesStore(store.DB())

	dueDate := time.Now().Add(30 * 24 * time.Hour)
	m := &milestones.Milestone{
		ID:          ulid.Make().String(),
		RepoID:      repoID,
		Number:      1,
		Title:       "v1.0.0",
		Description: "First release",
		State:       "open",
		DueDate:     &dueDate,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := milestonesStore.Create(context.Background(), m)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := milestonesStore.GetByID(context.Background(), m.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected milestone to be created")
	}
	if got.Title != "v1.0.0" {
		t.Errorf("got title %q, want %q", got.Title, "v1.0.0")
	}
	if got.Number != 1 {
		t.Errorf("got number %d, want 1", got.Number)
	}
}

func TestMilestonesStore_Create_WithAllFields(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	milestonesStore := NewMilestonesStore(store.DB())

	dueDate := time.Now().Add(30 * 24 * time.Hour)
	closedAt := time.Now()
	m := &milestones.Milestone{
		ID:           ulid.Make().String(),
		RepoID:       repoID,
		Number:       1,
		Title:        "Complete Milestone",
		Description:  "Full milestone with all fields",
		State:        "closed",
		DueDate:      &dueDate,
		OpenIssues:   5,
		ClosedIssues: 10,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		ClosedAt:     &closedAt,
	}

	err := milestonesStore.Create(context.Background(), m)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, _ := milestonesStore.GetByID(context.Background(), m.ID)
	if got.State != "closed" {
		t.Errorf("got state %q, want %q", got.State, "closed")
	}
	// Note: OpenIssues and ClosedIssues are computed fields, not stored in DB
	if got.DueDate == nil {
		t.Error("expected due_date to be set")
	}
	if got.ClosedAt == nil {
		t.Error("expected closed_at to be set")
	}
}

func TestMilestonesStore_GetByID(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	milestonesStore := NewMilestonesStore(store.DB())

	m := createTestMilestone(t, milestonesStore, repoID, 1)

	got, err := milestonesStore.GetByID(context.Background(), m.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected milestone")
	}
	if got.ID != m.ID {
		t.Errorf("got ID %q, want %q", got.ID, m.ID)
	}
}

func TestMilestonesStore_GetByID_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	milestonesStore := NewMilestonesStore(store.DB())

	got, err := milestonesStore.GetByID(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent milestone")
	}
}

func TestMilestonesStore_GetByNumber(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	milestonesStore := NewMilestonesStore(store.DB())

	m := createTestMilestone(t, milestonesStore, repoID, 42)

	got, err := milestonesStore.GetByNumber(context.Background(), repoID, 42)
	if err != nil {
		t.Fatalf("GetByNumber failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected milestone")
	}
	if got.Number != 42 {
		t.Errorf("got number %d, want 42", got.Number)
	}
	if got.ID != m.ID {
		t.Errorf("got ID %q, want %q", got.ID, m.ID)
	}
}

func TestMilestonesStore_GetByNumber_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	milestonesStore := NewMilestonesStore(store.DB())

	got, err := milestonesStore.GetByNumber(context.Background(), repoID, 999)
	if err != nil {
		t.Fatalf("GetByNumber failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent milestone number")
	}
}

func TestMilestonesStore_Update(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	milestonesStore := NewMilestonesStore(store.DB())

	m := createTestMilestone(t, milestonesStore, repoID, 1)

	m.Title = "Updated Title"
	m.Description = "Updated description"
	m.State = "closed"

	err := milestonesStore.Update(context.Background(), m)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, _ := milestonesStore.GetByID(context.Background(), m.ID)
	if got.Title != "Updated Title" {
		t.Errorf("got title %q, want %q", got.Title, "Updated Title")
	}
	if got.State != "closed" {
		t.Errorf("got state %q, want %q", got.State, "closed")
	}
}

func TestMilestonesStore_Update_DueDate(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	milestonesStore := NewMilestonesStore(store.DB())

	m := createTestMilestone(t, milestonesStore, repoID, 1)

	dueDate := time.Now().Add(60 * 24 * time.Hour)
	m.DueDate = &dueDate

	milestonesStore.Update(context.Background(), m)

	got, _ := milestonesStore.GetByID(context.Background(), m.ID)
	if got.DueDate == nil {
		t.Error("expected due_date to be set")
	}
}

func TestMilestonesStore_Update_CloseMilestone(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	milestonesStore := NewMilestonesStore(store.DB())

	m := createTestMilestone(t, milestonesStore, repoID, 1)

	closedAt := time.Now()
	m.State = "closed"
	m.ClosedAt = &closedAt

	milestonesStore.Update(context.Background(), m)

	got, _ := milestonesStore.GetByID(context.Background(), m.ID)
	if got.State != "closed" {
		t.Error("expected milestone to be closed")
	}
	if got.ClosedAt == nil {
		t.Error("expected closed_at to be set")
	}
}

func TestMilestonesStore_Delete(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	milestonesStore := NewMilestonesStore(store.DB())

	m := createTestMilestone(t, milestonesStore, repoID, 1)

	err := milestonesStore.Delete(context.Background(), m.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	got, _ := milestonesStore.GetByID(context.Background(), m.ID)
	if got != nil {
		t.Error("expected milestone to be deleted")
	}
}

func TestMilestonesStore_Delete_NonExistent(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	milestonesStore := NewMilestonesStore(store.DB())

	err := milestonesStore.Delete(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("Delete should not error for non-existent milestone: %v", err)
	}
}

// =============================================================================
// List Tests
// =============================================================================

func TestMilestonesStore_List(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	milestonesStore := NewMilestonesStore(store.DB())

	for i := 1; i <= 5; i++ {
		createTestMilestone(t, milestonesStore, repoID, i)
	}

	list, err := milestonesStore.List(context.Background(), repoID, "")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 5 {
		t.Errorf("got %d milestones, want 5", len(list))
	}
}

func TestMilestonesStore_List_FilterByState(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	milestonesStore := NewMilestonesStore(store.DB())

	// Create open milestones
	for i := 1; i <= 3; i++ {
		createTestMilestone(t, milestonesStore, repoID, i)
	}

	// Create closed milestones
	for i := 4; i <= 5; i++ {
		m := createTestMilestone(t, milestonesStore, repoID, i)
		m.State = "closed"
		milestonesStore.Update(context.Background(), m)
	}

	// Filter open
	openList, _ := milestonesStore.List(context.Background(), repoID, "open")
	if len(openList) != 3 {
		t.Errorf("got %d open milestones, want 3", len(openList))
	}

	// Filter closed
	closedList, _ := milestonesStore.List(context.Background(), repoID, "closed")
	if len(closedList) != 2 {
		t.Errorf("got %d closed milestones, want 2", len(closedList))
	}

	// All milestones
	allList, _ := milestonesStore.List(context.Background(), repoID, "")
	if len(allList) != 5 {
		t.Errorf("got %d all milestones, want 5", len(allList))
	}
}

func TestMilestonesStore_List_Empty(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	milestonesStore := NewMilestonesStore(store.DB())

	list, err := milestonesStore.List(context.Background(), repoID, "")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if list != nil && len(list) != 0 {
		t.Error("expected empty list")
	}
}

// =============================================================================
// GetNextNumber Tests
// =============================================================================

func TestMilestonesStore_GetNextNumber_Empty(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	milestonesStore := NewMilestonesStore(store.DB())

	next, err := milestonesStore.GetNextNumber(context.Background(), repoID)
	if err != nil {
		t.Fatalf("GetNextNumber failed: %v", err)
	}
	if next != 1 {
		t.Errorf("got next number %d, want 1", next)
	}
}

func TestMilestonesStore_GetNextNumber_WithExisting(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	milestonesStore := NewMilestonesStore(store.DB())

	for i := 1; i <= 3; i++ {
		createTestMilestone(t, milestonesStore, repoID, i)
	}

	next, err := milestonesStore.GetNextNumber(context.Background(), repoID)
	if err != nil {
		t.Fatalf("GetNextNumber failed: %v", err)
	}
	if next != 4 {
		t.Errorf("got next number %d, want 4", next)
	}
}

func TestMilestonesStore_GetNextNumber_WithGaps(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	milestonesStore := NewMilestonesStore(store.DB())

	// Create milestones with gaps
	createTestMilestone(t, milestonesStore, repoID, 1)
	createTestMilestone(t, milestonesStore, repoID, 5)
	createTestMilestone(t, milestonesStore, repoID, 10)

	next, err := milestonesStore.GetNextNumber(context.Background(), repoID)
	if err != nil {
		t.Fatalf("GetNextNumber failed: %v", err)
	}
	// Should return max + 1, not fill gaps
	if next != 11 {
		t.Errorf("got next number %d, want 11", next)
	}
}

func TestMilestonesStore_GetNextNumber_PerRepo(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())
	milestonesStore := NewMilestonesStore(store.DB())

	user := createTestUser(t, usersStore)
	repo1 := createTestRepo(t, reposStore, user.ID)
	repo2 := createTestRepo(t, reposStore, user.ID)

	// Create 5 milestones in repo1
	for i := 1; i <= 5; i++ {
		createTestMilestone(t, milestonesStore, repo1.ID, i)
	}

	// Create 2 milestones in repo2
	for i := 1; i <= 2; i++ {
		createTestMilestone(t, milestonesStore, repo2.ID, i)
	}

	next1, _ := milestonesStore.GetNextNumber(context.Background(), repo1.ID)
	next2, _ := milestonesStore.GetNextNumber(context.Background(), repo2.ID)

	if next1 != 6 {
		t.Errorf("got next number for repo1 %d, want 6", next1)
	}
	if next2 != 3 {
		t.Errorf("got next number for repo2 %d, want 3", next2)
	}
}

// Verify interface compliance
var _ milestones.Store = (*MilestonesStore)(nil)
