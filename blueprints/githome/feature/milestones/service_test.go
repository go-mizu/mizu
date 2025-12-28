package milestones_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"github.com/go-mizu/blueprints/githome/feature/milestones"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
	"github.com/go-mizu/blueprints/githome/store/duckdb"
)

func setupTestService(t *testing.T) (*milestones.Service, *duckdb.Store, func()) {
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

	milestonesStore := duckdb.NewMilestonesStore(db)
	service := milestones.NewService(milestonesStore, store.Repos(), store.Users(), "https://api.example.com")

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

func createTestMilestone(t *testing.T, service *milestones.Service, owner, repo string, creatorID int64, title string) *milestones.Milestone {
	t.Helper()
	m, err := service.Create(context.Background(), owner, repo, creatorID, &milestones.CreateIn{
		Title:       title,
		Description: "Test milestone",
	})
	if err != nil {
		t.Fatalf("failed to create test milestone: %v", err)
	}
	return m
}

// Milestone Creation Tests

func TestService_Create_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")

	dueOn := time.Now().Add(7 * 24 * time.Hour)
	m, err := service.Create(context.Background(), "owner", "testrepo", owner.ID, &milestones.CreateIn{
		Title:       "v1.0",
		Description: "First release",
		DueOn:       &dueOn,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if m.Title != "v1.0" {
		t.Errorf("got title %q, want v1.0", m.Title)
	}
	if m.Description != "First release" {
		t.Errorf("got description %q, want First release", m.Description)
	}
	if m.State != "open" {
		t.Errorf("got state %q, want open", m.State)
	}
	if m.Number != 1 {
		t.Errorf("got number %d, want 1", m.Number)
	}
	if m.ID == 0 {
		t.Error("expected ID to be assigned")
	}
	if m.OpenIssues != 0 {
		t.Errorf("expected open_issues 0, got %d", m.OpenIssues)
	}
	if m.ClosedIssues != 0 {
		t.Errorf("expected closed_issues 0, got %d", m.ClosedIssues)
	}
	if m.Creator == nil {
		t.Error("expected creator to be set")
	}
	if m.URL == "" {
		t.Error("expected URL to be populated")
	}
}

func TestService_Create_DuplicateTitle(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")

	createTestMilestone(t, service, "owner", "testrepo", owner.ID, "v1.0")

	_, err := service.Create(context.Background(), "owner", "testrepo", owner.ID, &milestones.CreateIn{
		Title: "v1.0",
	})
	if err != milestones.ErrMilestoneExists {
		t.Errorf("expected ErrMilestoneExists, got %v", err)
	}
}

func TestService_Create_RepoNotFound(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")

	_, err := service.Create(context.Background(), "owner", "unknown", owner.ID, &milestones.CreateIn{
		Title: "v1.0",
	})
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

func TestService_Create_WithState(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")

	m, err := service.Create(context.Background(), "owner", "testrepo", owner.ID, &milestones.CreateIn{
		Title: "v1.0",
		State: "closed",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if m.State != "closed" {
		t.Errorf("got state %q, want closed", m.State)
	}
}

// Milestone Retrieval Tests

func TestService_Get_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")
	created := createTestMilestone(t, service, "owner", "testrepo", owner.ID, "v1.0")

	m, err := service.Get(context.Background(), "owner", "testrepo", created.Number)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if m.ID != created.ID {
		t.Errorf("got ID %d, want %d", m.ID, created.ID)
	}
	if m.Title != "v1.0" {
		t.Errorf("got title %q, want v1.0", m.Title)
	}
}

func TestService_Get_NotFound(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")

	_, err := service.Get(context.Background(), "owner", "testrepo", 999)
	if err != milestones.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_Get_RepoNotFound(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	_, err := service.Get(context.Background(), "unknown", "repo", 1)
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

func TestService_GetByID_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")
	created := createTestMilestone(t, service, "owner", "testrepo", owner.ID, "v1.0")

	m, err := service.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if m.ID != created.ID {
		t.Errorf("got ID %d, want %d", m.ID, created.ID)
	}
}

func TestService_GetByID_NotFound(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	_, err := service.GetByID(context.Background(), 99999)
	if err != milestones.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_List(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")

	createTestMilestone(t, service, "owner", "testrepo", owner.ID, "v1.0")
	createTestMilestone(t, service, "owner", "testrepo", owner.ID, "v1.1")
	createTestMilestone(t, service, "owner", "testrepo", owner.ID, "v2.0")

	list, err := service.List(context.Background(), "owner", "testrepo", nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != 3 {
		t.Errorf("expected 3 milestones, got %d", len(list))
	}
}

func TestService_List_Pagination(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")

	for i := 0; i < 5; i++ {
		createTestMilestone(t, service, "owner", "testrepo", owner.ID, "v"+string(rune('a'+i)))
	}

	list, err := service.List(context.Background(), "owner", "testrepo", &milestones.ListOpts{
		Page:    1,
		PerPage: 2,
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("expected 2 milestones, got %d", len(list))
	}
}

func TestService_List_ByState(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")

	// Create open milestones
	createTestMilestone(t, service, "owner", "testrepo", owner.ID, "v1.0")
	createTestMilestone(t, service, "owner", "testrepo", owner.ID, "v1.1")

	// Create closed milestone
	_, _ = service.Create(context.Background(), "owner", "testrepo", owner.ID, &milestones.CreateIn{
		Title: "v0.9",
		State: "closed",
	})

	// List open only (default)
	openList, err := service.List(context.Background(), "owner", "testrepo", &milestones.ListOpts{State: "open"})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(openList) != 2 {
		t.Errorf("expected 2 open milestones, got %d", len(openList))
	}

	// List closed only
	closedList, err := service.List(context.Background(), "owner", "testrepo", &milestones.ListOpts{State: "closed"})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(closedList) != 1 {
		t.Errorf("expected 1 closed milestone, got %d", len(closedList))
	}

	// List all
	allList, err := service.List(context.Background(), "owner", "testrepo", &milestones.ListOpts{State: "all"})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(allList) != 3 {
		t.Errorf("expected 3 total milestones, got %d", len(allList))
	}
}

func TestService_List_RepoNotFound(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	_, err := service.List(context.Background(), "unknown", "repo", nil)
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

// Milestone Update Tests

func TestService_Update_Title(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")
	created := createTestMilestone(t, service, "owner", "testrepo", owner.ID, "v1.0")

	newTitle := "v1.0-final"
	updated, err := service.Update(context.Background(), "owner", "testrepo", created.Number, &milestones.UpdateIn{
		Title: &newTitle,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if updated.Title != "v1.0-final" {
		t.Errorf("got title %q, want v1.0-final", updated.Title)
	}
}

func TestService_Update_Description(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")
	created := createTestMilestone(t, service, "owner", "testrepo", owner.ID, "v1.0")

	newDesc := "Updated description"
	updated, err := service.Update(context.Background(), "owner", "testrepo", created.Number, &milestones.UpdateIn{
		Description: &newDesc,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if updated.Description != "Updated description" {
		t.Errorf("got description %q, want Updated description", updated.Description)
	}
}

func TestService_Update_State(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")
	created := createTestMilestone(t, service, "owner", "testrepo", owner.ID, "v1.0")

	newState := "closed"
	updated, err := service.Update(context.Background(), "owner", "testrepo", created.Number, &milestones.UpdateIn{
		State: &newState,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if updated.State != "closed" {
		t.Errorf("got state %q, want closed", updated.State)
	}
}

func TestService_Update_NotFound(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")

	newTitle := "newname"
	_, err := service.Update(context.Background(), "owner", "testrepo", 999, &milestones.UpdateIn{
		Title: &newTitle,
	})
	if err != milestones.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// Milestone Delete Tests

func TestService_Delete_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")
	created := createTestMilestone(t, service, "owner", "testrepo", owner.ID, "v1.0")

	err := service.Delete(context.Background(), "owner", "testrepo", created.Number)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deleted
	_, err = service.Get(context.Background(), "owner", "testrepo", created.Number)
	if err != milestones.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestService_Delete_NotFound(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")

	err := service.Delete(context.Background(), "owner", "testrepo", 999)
	if err != milestones.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// Counter Tests

func TestService_IncrementOpenIssues(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")
	created := createTestMilestone(t, service, "owner", "testrepo", owner.ID, "v1.0")

	// Increment open issues
	err := service.IncrementOpenIssues(context.Background(), created.ID, 1)
	if err != nil {
		t.Fatalf("IncrementOpenIssues failed: %v", err)
	}

	// Verify
	m, _ := service.GetByID(context.Background(), created.ID)
	if m.OpenIssues != 1 {
		t.Errorf("expected open_issues 1, got %d", m.OpenIssues)
	}

	// Increment again
	_ = service.IncrementOpenIssues(context.Background(), created.ID, 2)
	m, _ = service.GetByID(context.Background(), created.ID)
	if m.OpenIssues != 3 {
		t.Errorf("expected open_issues 3, got %d", m.OpenIssues)
	}
}

func TestService_IncrementClosedIssues(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")
	created := createTestMilestone(t, service, "owner", "testrepo", owner.ID, "v1.0")

	// Increment closed issues
	err := service.IncrementClosedIssues(context.Background(), created.ID, 1)
	if err != nil {
		t.Fatalf("IncrementClosedIssues failed: %v", err)
	}

	// Verify
	m, _ := service.GetByID(context.Background(), created.ID)
	if m.ClosedIssues != 1 {
		t.Errorf("expected closed_issues 1, got %d", m.ClosedIssues)
	}
}

func TestService_DecrementOpenIssues(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")
	created := createTestMilestone(t, service, "owner", "testrepo", owner.ID, "v1.0")

	// First increment
	_ = service.IncrementOpenIssues(context.Background(), created.ID, 5)

	// Then decrement
	err := service.IncrementOpenIssues(context.Background(), created.ID, -2)
	if err != nil {
		t.Fatalf("DecrementOpenIssues failed: %v", err)
	}

	// Verify
	m, _ := service.GetByID(context.Background(), created.ID)
	if m.OpenIssues != 3 {
		t.Errorf("expected open_issues 3, got %d", m.OpenIssues)
	}
}

// URL Population Tests

func TestService_PopulateURLs(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")
	m := createTestMilestone(t, service, "owner", "testrepo", owner.ID, "v1.0")

	if m.URL != "https://api.example.com/api/v3/repos/owner/testrepo/milestones/1" {
		t.Errorf("unexpected URL: %s", m.URL)
	}
	if m.HTMLURL != "https://api.example.com/owner/testrepo/milestone/1" {
		t.Errorf("unexpected HTMLURL: %s", m.HTMLURL)
	}
	if m.LabelsURL != "https://api.example.com/api/v3/repos/owner/testrepo/milestones/1/labels" {
		t.Errorf("unexpected LabelsURL: %s", m.LabelsURL)
	}
	if m.NodeID == "" {
		t.Error("expected NodeID to be set")
	}
}

// Number Sequence Tests

func TestService_NumberSequence(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "testrepo")

	m1 := createTestMilestone(t, service, "owner", "testrepo", owner.ID, "v1.0")
	m2 := createTestMilestone(t, service, "owner", "testrepo", owner.ID, "v1.1")
	m3 := createTestMilestone(t, service, "owner", "testrepo", owner.ID, "v2.0")

	if m1.Number != 1 {
		t.Errorf("expected milestone 1 number 1, got %d", m1.Number)
	}
	if m2.Number != 2 {
		t.Errorf("expected milestone 2 number 2, got %d", m2.Number)
	}
	if m3.Number != 3 {
		t.Errorf("expected milestone 3 number 3, got %d", m3.Number)
	}
}

// Integration Test - Milestones Across Repos

func TestService_MilestonesAcrossRepos(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, store, "owner", "owner@example.com")
	createTestRepo(t, store, owner, "repo1")
	createTestRepo(t, store, owner, "repo2")

	// Same title in different repos should work
	m1 := createTestMilestone(t, service, "owner", "repo1", owner.ID, "v1.0")
	m2 := createTestMilestone(t, service, "owner", "repo2", owner.ID, "v1.0")

	if m1.ID == m2.ID {
		t.Error("milestones in different repos should have different IDs")
	}

	// Numbers should be per-repo
	if m1.Number != 1 {
		t.Errorf("repo1 milestone should have number 1, got %d", m1.Number)
	}
	if m2.Number != 1 {
		t.Errorf("repo2 milestone should have number 1, got %d", m2.Number)
	}

	// Each repo should have its own milestones
	list1, _ := service.List(context.Background(), "owner", "repo1", &milestones.ListOpts{State: "all"})
	list2, _ := service.List(context.Background(), "owner", "repo2", &milestones.ListOpts{State: "all"})

	if len(list1) != 1 {
		t.Errorf("repo1 should have 1 milestone, got %d", len(list1))
	}
	if len(list2) != 1 {
		t.Errorf("repo2 should have 1 milestone, got %d", len(list2))
	}
}
