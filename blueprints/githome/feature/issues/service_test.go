package issues_test

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"github.com/go-mizu/blueprints/githome/feature/issues"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
	"github.com/go-mizu/blueprints/githome/store/duckdb"
)

func setupTestService(t *testing.T) (*issues.Service, *duckdb.Store, func()) {
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

	issuesStore := duckdb.NewIssuesStore(db)
	orgsStore := duckdb.NewOrgsStore(db)
	collabStore := duckdb.NewCollaboratorsStore(db)
	service := issues.NewService(issuesStore, store.Repos(), store.Users(), orgsStore, collabStore, "https://api.example.com")

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
		HasIssues:     true,
	}
	if err := store.Repos().Create(context.Background(), repo); err != nil {
		t.Fatalf("failed to create test repo: %v", err)
	}
	return repo
}

func createTestIssue(t *testing.T, service *issues.Service, owner, repo string, creatorID int64, title string) *issues.Issue {
	t.Helper()
	issue, err := service.Create(context.Background(), owner, repo, creatorID, &issues.CreateIn{
		Title: title,
		Body:  "Issue body",
	})
	if err != nil {
		t.Fatalf("failed to create test issue: %v", err)
	}
	return issue
}

// Issue Creation Tests

func TestService_Create_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")

	issue, err := service.Create(context.Background(), "testowner", "testrepo", user.ID, &issues.CreateIn{
		Title: "Test Issue",
		Body:  "This is a test issue body",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if issue.Title != "Test Issue" {
		t.Errorf("got title %q, want Test Issue", issue.Title)
	}
	if issue.Body != "This is a test issue body" {
		t.Errorf("got body %q, want This is a test issue body", issue.Body)
	}
	if issue.Number != 1 {
		t.Errorf("got number %d, want 1", issue.Number)
	}
	if issue.State != "open" {
		t.Errorf("got state %q, want open", issue.State)
	}
	if issue.User == nil {
		t.Error("expected user to be set")
	}
	if issue.ID == 0 {
		t.Error("expected ID to be assigned")
	}
	if issue.URL == "" {
		t.Error("expected URL to be populated")
	}
}

func TestService_Create_RepoNotFound(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")

	_, err := service.Create(context.Background(), "unknown", "repo", user.ID, &issues.CreateIn{
		Title: "Test Issue",
	})
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

func TestService_Create_IncrementsOpenIssues(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	repo := createTestRepo(t, store, user, "testrepo")

	// Create issues
	createTestIssue(t, service, "testowner", "testrepo", user.ID, "Issue 1")
	createTestIssue(t, service, "testowner", "testrepo", user.ID, "Issue 2")

	// Check repo counter
	updatedRepo, _ := store.Repos().GetByID(context.Background(), repo.ID)
	if updatedRepo.OpenIssuesCount != 2 {
		t.Errorf("expected open_issues_count 2, got %d", updatedRepo.OpenIssuesCount)
	}
}

func TestService_Create_NumbersIncrement(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")

	issue1 := createTestIssue(t, service, "testowner", "testrepo", user.ID, "Issue 1")
	issue2 := createTestIssue(t, service, "testowner", "testrepo", user.ID, "Issue 2")
	issue3 := createTestIssue(t, service, "testowner", "testrepo", user.ID, "Issue 3")

	if issue1.Number != 1 {
		t.Errorf("expected issue1 number 1, got %d", issue1.Number)
	}
	if issue2.Number != 2 {
		t.Errorf("expected issue2 number 2, got %d", issue2.Number)
	}
	if issue3.Number != 3 {
		t.Errorf("expected issue3 number 3, got %d", issue3.Number)
	}
}

// Issue Retrieval Tests

func TestService_Get_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")
	created := createTestIssue(t, service, "testowner", "testrepo", user.ID, "Test Issue")

	issue, err := service.Get(context.Background(), "testowner", "testrepo", created.Number)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if issue.Number != created.Number {
		t.Errorf("got number %d, want %d", issue.Number, created.Number)
	}
	if issue.Title != "Test Issue" {
		t.Errorf("got title %q, want Test Issue", issue.Title)
	}
}

func TestService_Get_NotFound(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")

	_, err := service.Get(context.Background(), "testowner", "testrepo", 999)
	if err != issues.ErrNotFound {
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

func TestService_ListForRepo(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")
	createTestIssue(t, service, "testowner", "testrepo", user.ID, "Issue 1")
	createTestIssue(t, service, "testowner", "testrepo", user.ID, "Issue 2")
	createTestIssue(t, service, "testowner", "testrepo", user.ID, "Issue 3")

	list, err := service.ListForRepo(context.Background(), "testowner", "testrepo", nil)
	if err != nil {
		t.Fatalf("ListForRepo failed: %v", err)
	}

	if len(list) != 3 {
		t.Errorf("expected 3 issues, got %d", len(list))
	}
}

func TestService_ListForRepo_FilterByState(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")
	createTestIssue(t, service, "testowner", "testrepo", user.ID, "Open Issue 1")
	createTestIssue(t, service, "testowner", "testrepo", user.ID, "Open Issue 2")
	issue3 := createTestIssue(t, service, "testowner", "testrepo", user.ID, "To Close")

	// Close one issue
	closedState := "closed"
	_, _ = service.Update(context.Background(), "testowner", "testrepo", issue3.Number, &issues.UpdateIn{
		State: &closedState,
	})

	// List open only
	openList, _ := service.ListForRepo(context.Background(), "testowner", "testrepo", &issues.ListOpts{
		State: "open",
	})
	if len(openList) != 2 {
		t.Errorf("expected 2 open issues, got %d", len(openList))
	}

	// List closed only
	closedList, _ := service.ListForRepo(context.Background(), "testowner", "testrepo", &issues.ListOpts{
		State: "closed",
	})
	if len(closedList) != 1 {
		t.Errorf("expected 1 closed issue, got %d", len(closedList))
	}

	// List all
	allList, _ := service.ListForRepo(context.Background(), "testowner", "testrepo", &issues.ListOpts{
		State: "all",
	})
	if len(allList) != 3 {
		t.Errorf("expected 3 total issues, got %d", len(allList))
	}
}

func TestService_ListForRepo_Pagination(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")
	for i := 0; i < 5; i++ {
		createTestIssue(t, service, "testowner", "testrepo", user.ID, "Issue")
	}

	list, err := service.ListForRepo(context.Background(), "testowner", "testrepo", &issues.ListOpts{
		Page:    1,
		PerPage: 2,
	})
	if err != nil {
		t.Fatalf("ListForRepo failed: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("expected 2 issues, got %d", len(list))
	}
}

// Issue Update Tests

func TestService_Update_Title(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")
	created := createTestIssue(t, service, "testowner", "testrepo", user.ID, "Original Title")

	newTitle := "Updated Title"
	updated, err := service.Update(context.Background(), "testowner", "testrepo", created.Number, &issues.UpdateIn{
		Title: &newTitle,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if updated.Title != "Updated Title" {
		t.Errorf("got title %q, want Updated Title", updated.Title)
	}
}

func TestService_Update_Body(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")
	created := createTestIssue(t, service, "testowner", "testrepo", user.ID, "Test Issue")

	newBody := "Updated body content"
	updated, err := service.Update(context.Background(), "testowner", "testrepo", created.Number, &issues.UpdateIn{
		Body: &newBody,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if updated.Body != "Updated body content" {
		t.Errorf("got body %q, want Updated body content", updated.Body)
	}
}

func TestService_Update_Close(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	repo := createTestRepo(t, store, user, "testrepo")
	created := createTestIssue(t, service, "testowner", "testrepo", user.ID, "Test Issue")

	closedState := "closed"
	updated, err := service.Update(context.Background(), "testowner", "testrepo", created.Number, &issues.UpdateIn{
		State: &closedState,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if updated.State != "closed" {
		t.Errorf("got state %q, want closed", updated.State)
	}
	if updated.ClosedAt == nil {
		t.Error("expected ClosedAt to be set")
	}

	// Check repo counter decremented
	updatedRepo, _ := store.Repos().GetByID(context.Background(), repo.ID)
	if updatedRepo.OpenIssuesCount != 0 {
		t.Errorf("expected open_issues_count 0, got %d", updatedRepo.OpenIssuesCount)
	}
}

func TestService_Update_Reopen(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	repo := createTestRepo(t, store, user, "testrepo")
	created := createTestIssue(t, service, "testowner", "testrepo", user.ID, "Test Issue")

	// Close
	closedState := "closed"
	_, _ = service.Update(context.Background(), "testowner", "testrepo", created.Number, &issues.UpdateIn{
		State: &closedState,
	})

	// Reopen
	openState := "open"
	updated, err := service.Update(context.Background(), "testowner", "testrepo", created.Number, &issues.UpdateIn{
		State: &openState,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if updated.State != "open" {
		t.Errorf("got state %q, want open", updated.State)
	}

	// Check repo counter incremented back
	updatedRepo, _ := store.Repos().GetByID(context.Background(), repo.ID)
	if updatedRepo.OpenIssuesCount != 1 {
		t.Errorf("expected open_issues_count 1, got %d", updatedRepo.OpenIssuesCount)
	}
}

func TestService_Update_NotFound(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")

	newTitle := "Updated"
	_, err := service.Update(context.Background(), "testowner", "testrepo", 999, &issues.UpdateIn{
		Title: &newTitle,
	})
	if err != issues.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// Lock/Unlock Tests

func TestService_Lock_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")
	created := createTestIssue(t, service, "testowner", "testrepo", user.ID, "Test Issue")

	err := service.Lock(context.Background(), "testowner", "testrepo", created.Number, "off-topic")
	if err != nil {
		t.Fatalf("Lock failed: %v", err)
	}

	issue, _ := service.Get(context.Background(), "testowner", "testrepo", created.Number)
	if !issue.Locked {
		t.Error("expected issue to be locked")
	}
	if issue.ActiveLockReason != "off-topic" {
		t.Errorf("got lock reason %q, want off-topic", issue.ActiveLockReason)
	}
}

func TestService_Unlock_Success(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")
	created := createTestIssue(t, service, "testowner", "testrepo", user.ID, "Test Issue")

	// Lock first
	_ = service.Lock(context.Background(), "testowner", "testrepo", created.Number, "spam")

	// Then unlock
	err := service.Unlock(context.Background(), "testowner", "testrepo", created.Number)
	if err != nil {
		t.Fatalf("Unlock failed: %v", err)
	}

	issue, _ := service.Get(context.Background(), "testowner", "testrepo", created.Number)
	if issue.Locked {
		t.Error("expected issue to be unlocked")
	}
}

// URL Population Tests

func TestService_PopulateURLs(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "testrepo")
	issue := createTestIssue(t, service, "testowner", "testrepo", user.ID, "Test Issue")

	if issue.URL != "https://api.example.com/api/v3/repos/testowner/testrepo/issues/1" {
		t.Errorf("unexpected URL: %s", issue.URL)
	}
	if issue.HTMLURL != "https://api.example.com/testowner/testrepo/issues/1" {
		t.Errorf("unexpected HTMLURL: %s", issue.HTMLURL)
	}
	if issue.RepositoryURL == "" {
		t.Error("expected RepositoryURL to be set")
	}
	if issue.NodeID == "" {
		t.Error("expected NodeID to be set")
	}
}

// Integration Test - Multiple Repos Isolation

func TestService_MultipleRepos(t *testing.T) {
	service, store, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, store, "testowner", "owner@example.com")
	createTestRepo(t, store, user, "repo1")
	createTestRepo(t, store, user, "repo2")

	// Create issues in different repos
	issue1 := createTestIssue(t, service, "testowner", "repo1", user.ID, "Issue in repo1")
	issue2 := createTestIssue(t, service, "testowner", "repo2", user.ID, "Issue in repo2")

	// Both should have number 1 (scoped to repo)
	if issue1.Number != 1 {
		t.Errorf("expected repo1 issue number 1, got %d", issue1.Number)
	}
	if issue2.Number != 1 {
		t.Errorf("expected repo2 issue number 1, got %d", issue2.Number)
	}

	// List should be isolated
	list1, _ := service.ListForRepo(context.Background(), "testowner", "repo1", nil)
	list2, _ := service.ListForRepo(context.Background(), "testowner", "repo2", nil)

	if len(list1) != 1 {
		t.Errorf("repo1 should have 1 issue, got %d", len(list1))
	}
	if len(list2) != 1 {
		t.Errorf("repo2 should have 1 issue, got %d", len(list2))
	}
}
