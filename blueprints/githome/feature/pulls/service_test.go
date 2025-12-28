package pulls_test

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"github.com/go-mizu/blueprints/githome/feature/pulls"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
	"github.com/go-mizu/blueprints/githome/store/duckdb"
)

func setupTestService(t *testing.T) (*pulls.Service, *duckdb.UsersStore, *duckdb.ReposStore, func()) {
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

	usersStore := duckdb.NewUsersStore(db)
	reposStore := duckdb.NewReposStore(db)
	pullsStore := duckdb.NewPullsStore(db)
	service := pulls.NewService(pullsStore, reposStore, usersStore, "https://api.example.com", "")

	cleanup := func() {
		store.Close()
	}

	return service, usersStore, reposStore, cleanup
}

func createTestUser(t *testing.T, usersStore *duckdb.UsersStore, login, email string) *users.User {
	t.Helper()
	user := &users.User{
		Login:        login,
		Email:        email,
		Name:         "Test User",
		PasswordHash: "hash",
		Type:         "User",
	}
	if err := usersStore.Create(context.Background(), user); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return user
}

func createTestRepo(t *testing.T, reposStore *duckdb.ReposStore, owner *users.User, name string) *repos.Repository {
	t.Helper()
	repo := &repos.Repository{
		Name:          name,
		FullName:      owner.Login + "/" + name,
		OwnerID:       owner.ID,
		OwnerType:     "User",
		Visibility:    "public",
		DefaultBranch: "main",
	}
	if err := reposStore.Create(context.Background(), repo); err != nil {
		t.Fatalf("failed to create test repo: %v", err)
	}
	return repo
}

func createTestPR(t *testing.T, service *pulls.Service, owner, repo string, creatorID int64, title string) *pulls.PullRequest {
	t.Helper()
	pr, err := service.Create(context.Background(), owner, repo, creatorID, &pulls.CreateIn{
		Title: title,
		Body:  "PR body",
		Head:  "feature",
		Base:  "main",
	})
	if err != nil {
		t.Fatalf("failed to create test PR: %v", err)
	}
	return pr
}

// PR Creation Tests

func TestService_Create_Success(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")

	pr, err := service.Create(context.Background(), "testowner", "testrepo", user.ID, &pulls.CreateIn{
		Title: "Test PR",
		Body:  "This is a test PR",
		Head:  "feature",
		Base:  "main",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if pr.Title != "Test PR" {
		t.Errorf("got title %q, want Test PR", pr.Title)
	}
	if pr.Body != "This is a test PR" {
		t.Errorf("got body %q, want This is a test PR", pr.Body)
	}
	if pr.Number != 1 {
		t.Errorf("got number %d, want 1", pr.Number)
	}
	if pr.State != "open" {
		t.Errorf("got state %q, want open", pr.State)
	}
	if pr.User == nil {
		t.Error("expected user to be set")
	}
	if pr.ID == 0 {
		t.Error("expected ID to be assigned")
	}
	if pr.URL == "" {
		t.Error("expected URL to be populated")
	}
	if pr.Head == nil || pr.Head.Ref != "feature" {
		t.Error("expected head ref to be 'feature'")
	}
	if pr.Base == nil || pr.Base.Ref != "main" {
		t.Error("expected base ref to be 'main'")
	}
}

func TestService_Create_RepoNotFound(t *testing.T) {
	service, usersStore, _, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")

	_, err := service.Create(context.Background(), "unknown", "repo", user.ID, &pulls.CreateIn{
		Title: "Test PR",
		Head:  "feature",
		Base:  "main",
	})
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

func TestService_Create_UserNotFound(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")

	_, err := service.Create(context.Background(), "testowner", "testrepo", 99999, &pulls.CreateIn{
		Title: "Test PR",
		Head:  "feature",
		Base:  "main",
	})
	if err != users.ErrNotFound {
		t.Errorf("expected users.ErrNotFound, got %v", err)
	}
}

func TestService_Create_NumbersIncrement(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")

	pr1 := createTestPR(t, service, "testowner", "testrepo", user.ID, "PR 1")
	pr2 := createTestPR(t, service, "testowner", "testrepo", user.ID, "PR 2")
	pr3 := createTestPR(t, service, "testowner", "testrepo", user.ID, "PR 3")

	if pr1.Number != 1 {
		t.Errorf("expected pr1 number 1, got %d", pr1.Number)
	}
	if pr2.Number != 2 {
		t.Errorf("expected pr2 number 2, got %d", pr2.Number)
	}
	if pr3.Number != 3 {
		t.Errorf("expected pr3 number 3, got %d", pr3.Number)
	}
}

func TestService_Create_Draft(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")

	pr, err := service.Create(context.Background(), "testowner", "testrepo", user.ID, &pulls.CreateIn{
		Title: "Draft PR",
		Head:  "feature",
		Base:  "main",
		Draft: true,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if !pr.Draft {
		t.Error("expected PR to be draft")
	}
}

func TestService_Create_MaintainerCanModify(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")

	pr, err := service.Create(context.Background(), "testowner", "testrepo", user.ID, &pulls.CreateIn{
		Title:               "Modifiable PR",
		Head:                "feature",
		Base:                "main",
		MaintainerCanModify: true,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if !pr.MaintainerCanModify {
		t.Error("expected maintainer_can_modify to be true")
	}
}

// PR Retrieval Tests

func TestService_Get_Success(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")
	created := createTestPR(t, service, "testowner", "testrepo", user.ID, "Test PR")

	pr, err := service.Get(context.Background(), "testowner", "testrepo", created.Number)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if pr.Number != created.Number {
		t.Errorf("got number %d, want %d", pr.Number, created.Number)
	}
	if pr.Title != "Test PR" {
		t.Errorf("got title %q, want Test PR", pr.Title)
	}
}

func TestService_Get_NotFound(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")

	_, err := service.Get(context.Background(), "testowner", "testrepo", 999)
	if err != pulls.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_Get_RepoNotFound(t *testing.T) {
	service, _, _, cleanup := setupTestService(t)
	defer cleanup()

	_, err := service.Get(context.Background(), "unknown", "repo", 1)
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

func TestService_List_Success(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")
	createTestPR(t, service, "testowner", "testrepo", user.ID, "PR 1")
	createTestPR(t, service, "testowner", "testrepo", user.ID, "PR 2")
	createTestPR(t, service, "testowner", "testrepo", user.ID, "PR 3")

	list, err := service.List(context.Background(), "testowner", "testrepo", nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != 3 {
		t.Errorf("expected 3 PRs, got %d", len(list))
	}
}

func TestService_List_FilterByState(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")
	createTestPR(t, service, "testowner", "testrepo", user.ID, "Open PR 1")
	createTestPR(t, service, "testowner", "testrepo", user.ID, "Open PR 2")
	pr3 := createTestPR(t, service, "testowner", "testrepo", user.ID, "To Close")

	// Close one PR
	closedState := "closed"
	_, _ = service.Update(context.Background(), "testowner", "testrepo", pr3.Number, &pulls.UpdateIn{
		State: &closedState,
	})

	// List open only
	openList, _ := service.List(context.Background(), "testowner", "testrepo", &pulls.ListOpts{
		State: "open",
	})
	if len(openList) != 2 {
		t.Errorf("expected 2 open PRs, got %d", len(openList))
	}

	// List closed only
	closedList, _ := service.List(context.Background(), "testowner", "testrepo", &pulls.ListOpts{
		State: "closed",
	})
	if len(closedList) != 1 {
		t.Errorf("expected 1 closed PR, got %d", len(closedList))
	}

	// List all
	allList, _ := service.List(context.Background(), "testowner", "testrepo", &pulls.ListOpts{
		State: "all",
	})
	if len(allList) != 3 {
		t.Errorf("expected 3 total PRs, got %d", len(allList))
	}
}

func TestService_List_Pagination(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")
	for i := 0; i < 5; i++ {
		createTestPR(t, service, "testowner", "testrepo", user.ID, "PR")
	}

	list, err := service.List(context.Background(), "testowner", "testrepo", &pulls.ListOpts{
		Page:    1,
		PerPage: 2,
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("expected 2 PRs, got %d", len(list))
	}
}

// PR Update Tests

func TestService_Update_Title(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")
	created := createTestPR(t, service, "testowner", "testrepo", user.ID, "Original Title")

	newTitle := "Updated Title"
	updated, err := service.Update(context.Background(), "testowner", "testrepo", created.Number, &pulls.UpdateIn{
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
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")
	created := createTestPR(t, service, "testowner", "testrepo", user.ID, "Test PR")

	newBody := "Updated body content"
	updated, err := service.Update(context.Background(), "testowner", "testrepo", created.Number, &pulls.UpdateIn{
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
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")
	created := createTestPR(t, service, "testowner", "testrepo", user.ID, "Test PR")

	closedState := "closed"
	updated, err := service.Update(context.Background(), "testowner", "testrepo", created.Number, &pulls.UpdateIn{
		State: &closedState,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if updated.State != "closed" {
		t.Errorf("got state %q, want closed", updated.State)
	}
}

func TestService_Update_Reopen(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")
	created := createTestPR(t, service, "testowner", "testrepo", user.ID, "Test PR")

	// Close
	closedState := "closed"
	_, _ = service.Update(context.Background(), "testowner", "testrepo", created.Number, &pulls.UpdateIn{
		State: &closedState,
	})

	// Reopen
	openState := "open"
	updated, err := service.Update(context.Background(), "testowner", "testrepo", created.Number, &pulls.UpdateIn{
		State: &openState,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if updated.State != "open" {
		t.Errorf("got state %q, want open", updated.State)
	}
}

func TestService_Update_NotFound(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")

	newTitle := "Updated"
	_, err := service.Update(context.Background(), "testowner", "testrepo", 999, &pulls.UpdateIn{
		Title: &newTitle,
	})
	if err != pulls.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// PR Merge Tests

func TestService_IsMerged_False(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")
	created := createTestPR(t, service, "testowner", "testrepo", user.ID, "Test PR")

	merged, err := service.IsMerged(context.Background(), "testowner", "testrepo", created.Number)
	if err != nil {
		t.Fatalf("IsMerged failed: %v", err)
	}

	if merged {
		t.Error("expected PR to not be merged")
	}
}

func TestService_IsMerged_True(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")
	created := createTestPR(t, service, "testowner", "testrepo", user.ID, "Test PR")

	// Merge the PR
	_, _ = service.Merge(context.Background(), "testowner", "testrepo", created.Number, &pulls.MergeIn{})

	merged, err := service.IsMerged(context.Background(), "testowner", "testrepo", created.Number)
	if err != nil {
		t.Fatalf("IsMerged failed: %v", err)
	}

	if !merged {
		t.Error("expected PR to be merged")
	}
}

func TestService_IsMerged_NotFound(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")

	_, err := service.IsMerged(context.Background(), "testowner", "testrepo", 999)
	if err != pulls.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_Merge_Success(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")
	created := createTestPR(t, service, "testowner", "testrepo", user.ID, "Test PR")

	result, err := service.Merge(context.Background(), "testowner", "testrepo", created.Number, &pulls.MergeIn{})
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	if !result.Merged {
		t.Error("expected merged to be true")
	}
	if result.SHA == "" {
		t.Error("expected SHA to be set")
	}
	if result.Message == "" {
		t.Error("expected message to be set")
	}
}

func TestService_Merge_AlreadyMerged(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")
	created := createTestPR(t, service, "testowner", "testrepo", user.ID, "Test PR")

	// First merge
	_, _ = service.Merge(context.Background(), "testowner", "testrepo", created.Number, &pulls.MergeIn{})

	// Second merge attempt
	_, err := service.Merge(context.Background(), "testowner", "testrepo", created.Number, &pulls.MergeIn{})
	if err != pulls.ErrAlreadyMerged {
		t.Errorf("expected ErrAlreadyMerged, got %v", err)
	}
}

func TestService_Merge_NotOpen(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")
	created := createTestPR(t, service, "testowner", "testrepo", user.ID, "Test PR")

	// Close the PR
	closedState := "closed"
	_, _ = service.Update(context.Background(), "testowner", "testrepo", created.Number, &pulls.UpdateIn{
		State: &closedState,
	})

	// Try to merge
	_, err := service.Merge(context.Background(), "testowner", "testrepo", created.Number, &pulls.MergeIn{})
	if err != pulls.ErrNotMergeable {
		t.Errorf("expected ErrNotMergeable, got %v", err)
	}
}

func TestService_Merge_NotFound(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")

	_, err := service.Merge(context.Background(), "testowner", "testrepo", 999, &pulls.MergeIn{})
	if err != pulls.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// Review Tests

func TestService_CreateReview_Success(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	reviewer := createTestUser(t, usersStore, "reviewer", "reviewer@example.com")
	createTestRepo(t, reposStore, user, "testrepo")
	pr := createTestPR(t, service, "testowner", "testrepo", user.ID, "Test PR")

	review, err := service.CreateReview(context.Background(), "testowner", "testrepo", pr.Number, reviewer.ID, &pulls.CreateReviewIn{
		Body:  "Looks good!",
		Event: "APPROVE",
	})
	if err != nil {
		t.Fatalf("CreateReview failed: %v", err)
	}

	if review.Body != "Looks good!" {
		t.Errorf("got body %q, want Looks good!", review.Body)
	}
	if review.State != "APPROVE" {
		t.Errorf("got state %q, want APPROVE", review.State)
	}
	if review.User == nil {
		t.Error("expected user to be set")
	}
}

func TestService_CreateReview_PRNotFound(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")

	_, err := service.CreateReview(context.Background(), "testowner", "testrepo", 999, user.ID, &pulls.CreateReviewIn{
		Body:  "Review",
		Event: "COMMENT",
	})
	if err != pulls.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_CreateReview_UserNotFound(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")
	pr := createTestPR(t, service, "testowner", "testrepo", user.ID, "Test PR")

	_, err := service.CreateReview(context.Background(), "testowner", "testrepo", pr.Number, 99999, &pulls.CreateReviewIn{
		Body:  "Review",
		Event: "COMMENT",
	})
	if err != users.ErrNotFound {
		t.Errorf("expected users.ErrNotFound, got %v", err)
	}
}

func TestService_CreateReview_DefaultState(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")
	pr := createTestPR(t, service, "testowner", "testrepo", user.ID, "Test PR")

	review, err := service.CreateReview(context.Background(), "testowner", "testrepo", pr.Number, user.ID, &pulls.CreateReviewIn{
		Body: "Comment only",
		// No event specified
	})
	if err != nil {
		t.Fatalf("CreateReview failed: %v", err)
	}

	if review.State != "PENDING" {
		t.Errorf("expected state 'PENDING', got %q", review.State)
	}
}

func TestService_GetReview_Success(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")
	pr := createTestPR(t, service, "testowner", "testrepo", user.ID, "Test PR")

	created, err := service.CreateReview(context.Background(), "testowner", "testrepo", pr.Number, user.ID, &pulls.CreateReviewIn{
		Body:  "Review",
		Event: "COMMENT",
	})
	if err != nil {
		t.Fatalf("CreateReview failed: %v", err)
	}

	review, err := service.GetReview(context.Background(), "testowner", "testrepo", pr.Number, created.ID)
	if err != nil {
		t.Fatalf("GetReview failed: %v", err)
	}

	if review == nil {
		t.Fatal("expected review to be non-nil")
	}
	if review.ID != created.ID {
		t.Errorf("got ID %d, want %d", review.ID, created.ID)
	}
}

func TestService_ListReviews_Success(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	reviewer := createTestUser(t, usersStore, "reviewer", "reviewer@example.com")
	createTestRepo(t, reposStore, user, "testrepo")
	pr := createTestPR(t, service, "testowner", "testrepo", user.ID, "Test PR")

	_, _ = service.CreateReview(context.Background(), "testowner", "testrepo", pr.Number, user.ID, &pulls.CreateReviewIn{
		Body:  "Review 1",
		Event: "COMMENT",
	})
	_, _ = service.CreateReview(context.Background(), "testowner", "testrepo", pr.Number, reviewer.ID, &pulls.CreateReviewIn{
		Body:  "Review 2",
		Event: "APPROVE",
	})

	reviews, err := service.ListReviews(context.Background(), "testowner", "testrepo", pr.Number, nil)
	if err != nil {
		t.Fatalf("ListReviews failed: %v", err)
	}

	if len(reviews) != 2 {
		t.Errorf("expected 2 reviews, got %d", len(reviews))
	}
}

func TestService_UpdateReview_Success(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")
	pr := createTestPR(t, service, "testowner", "testrepo", user.ID, "Test PR")

	created, _ := service.CreateReview(context.Background(), "testowner", "testrepo", pr.Number, user.ID, &pulls.CreateReviewIn{
		Body:  "Original",
		Event: "COMMENT",
	})

	updated, err := service.UpdateReview(context.Background(), "testowner", "testrepo", pr.Number, created.ID, "Updated body")
	if err != nil {
		t.Fatalf("UpdateReview failed: %v", err)
	}

	if updated.Body != "Updated body" {
		t.Errorf("got body %q, want Updated body", updated.Body)
	}
}

func TestService_SubmitReview_Success(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")
	pr := createTestPR(t, service, "testowner", "testrepo", user.ID, "Test PR")

	// Create pending review
	created, _ := service.CreateReview(context.Background(), "testowner", "testrepo", pr.Number, user.ID, &pulls.CreateReviewIn{
		Body: "Pending review",
	})

	// Submit it
	submitted, err := service.SubmitReview(context.Background(), "testowner", "testrepo", pr.Number, created.ID, &pulls.SubmitReviewIn{
		Event: "APPROVE",
	})
	if err != nil {
		t.Fatalf("SubmitReview failed: %v", err)
	}

	if submitted.State != "APPROVE" {
		t.Errorf("got state %q, want APPROVE", submitted.State)
	}
}

func TestService_DismissReview_Success(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")
	pr := createTestPR(t, service, "testowner", "testrepo", user.ID, "Test PR")

	created, _ := service.CreateReview(context.Background(), "testowner", "testrepo", pr.Number, user.ID, &pulls.CreateReviewIn{
		Body:  "To dismiss",
		Event: "CHANGES_REQUESTED",
	})

	dismissed, err := service.DismissReview(context.Background(), "testowner", "testrepo", pr.Number, created.ID, "Dismissing because...")
	if err != nil {
		t.Fatalf("DismissReview failed: %v", err)
	}

	if dismissed.State != "DISMISSED" {
		t.Errorf("got state %q, want DISMISSED", dismissed.State)
	}
}

// Review Comment Tests

func TestService_CreateReviewComment_Success(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")
	pr := createTestPR(t, service, "testowner", "testrepo", user.ID, "Test PR")

	comment, err := service.CreateReviewComment(context.Background(), "testowner", "testrepo", pr.Number, user.ID, &pulls.CreateReviewCommentIn{
		Body:     "Nice code!",
		CommitID: "abc123",
		Path:     "main.go",
		Line:     10,
	})
	if err != nil {
		t.Fatalf("CreateReviewComment failed: %v", err)
	}

	if comment.Body != "Nice code!" {
		t.Errorf("got body %q, want Nice code!", comment.Body)
	}
	if comment.Path != "main.go" {
		t.Errorf("got path %q, want main.go", comment.Path)
	}
	if comment.Line != 10 {
		t.Errorf("got line %d, want 10", comment.Line)
	}
}

func TestService_ListReviewComments_Success(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")
	pr := createTestPR(t, service, "testowner", "testrepo", user.ID, "Test PR")

	_, _ = service.CreateReviewComment(context.Background(), "testowner", "testrepo", pr.Number, user.ID, &pulls.CreateReviewCommentIn{
		Body:     "Comment 1",
		CommitID: "abc123",
		Path:     "main.go",
	})
	_, _ = service.CreateReviewComment(context.Background(), "testowner", "testrepo", pr.Number, user.ID, &pulls.CreateReviewCommentIn{
		Body:     "Comment 2",
		CommitID: "abc123",
		Path:     "test.go",
	})

	comments, err := service.ListReviewComments(context.Background(), "testowner", "testrepo", pr.Number, nil)
	if err != nil {
		t.Fatalf("ListReviewComments failed: %v", err)
	}

	if len(comments) != 2 {
		t.Errorf("expected 2 comments, got %d", len(comments))
	}
}

// Reviewer Tests

func TestService_RequestReviewers_Success(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	reviewer := createTestUser(t, usersStore, "reviewer", "reviewer@example.com")
	createTestRepo(t, reposStore, user, "testrepo")
	pr := createTestPR(t, service, "testowner", "testrepo", user.ID, "Test PR")

	updated, err := service.RequestReviewers(context.Background(), "testowner", "testrepo", pr.Number, []string{"reviewer"}, nil)
	if err != nil {
		t.Fatalf("RequestReviewers failed: %v", err)
	}

	// PR should be returned without error
	if updated == nil {
		t.Error("expected PR to be returned")
	}
	_ = reviewer // used to create the reviewer
}

func TestService_RequestReviewers_SkipsUnknownUsers(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")
	pr := createTestPR(t, service, "testowner", "testrepo", user.ID, "Test PR")

	// Should not error, just skip unknown users
	_, err := service.RequestReviewers(context.Background(), "testowner", "testrepo", pr.Number, []string{"unknown"}, nil)
	if err != nil {
		t.Fatalf("RequestReviewers failed: %v", err)
	}
}

func TestService_RemoveReviewers_Success(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	reviewer := createTestUser(t, usersStore, "reviewer", "reviewer@example.com")
	createTestRepo(t, reposStore, user, "testrepo")
	pr := createTestPR(t, service, "testowner", "testrepo", user.ID, "Test PR")

	// Add reviewer
	_, _ = service.RequestReviewers(context.Background(), "testowner", "testrepo", pr.Number, []string{"reviewer"}, nil)

	// Remove reviewer
	updated, err := service.RemoveReviewers(context.Background(), "testowner", "testrepo", pr.Number, []string{"reviewer"}, nil)
	if err != nil {
		t.Fatalf("RemoveReviewers failed: %v", err)
	}

	if updated == nil {
		t.Error("expected PR to be returned")
	}
	_ = reviewer // used
}

// Author Association Tests

func TestService_AuthorAssociation_Owner(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")
	pr := createTestPR(t, service, "testowner", "testrepo", user.ID, "Test PR")

	if pr.AuthorAssociation != "OWNER" {
		t.Errorf("expected author_association 'OWNER', got %q", pr.AuthorAssociation)
	}
}

func TestService_AuthorAssociation_None(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, usersStore, "testowner", "owner@example.com")
	contributor := createTestUser(t, usersStore, "contributor", "contributor@example.com")
	createTestRepo(t, reposStore, owner, "testrepo")
	pr := createTestPR(t, service, "testowner", "testrepo", contributor.ID, "Test PR")

	if pr.AuthorAssociation != "NONE" {
		t.Errorf("expected author_association 'NONE', got %q", pr.AuthorAssociation)
	}
}

// Mock Behavior Tests

func TestService_ListCommits_ReturnsEmpty(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")
	pr := createTestPR(t, service, "testowner", "testrepo", user.ID, "Test PR")

	commits, err := service.ListCommits(context.Background(), "testowner", "testrepo", pr.Number, nil)
	if err != nil {
		t.Fatalf("ListCommits failed: %v", err)
	}

	// Mock implementation returns empty list
	if len(commits) != 0 {
		t.Errorf("expected 0 commits (mock), got %d", len(commits))
	}
}

func TestService_ListFiles_ReturnsEmpty(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")
	pr := createTestPR(t, service, "testowner", "testrepo", user.ID, "Test PR")

	files, err := service.ListFiles(context.Background(), "testowner", "testrepo", pr.Number, nil)
	if err != nil {
		t.Fatalf("ListFiles failed: %v", err)
	}

	// Mock implementation returns empty list
	if len(files) != 0 {
		t.Errorf("expected 0 files (mock), got %d", len(files))
	}
}

func TestService_UpdateBranch_NoOp(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")
	pr := createTestPR(t, service, "testowner", "testrepo", user.ID, "Test PR")

	// Without a real git repository, UpdateBranch returns ErrNotMergeable
	err := service.UpdateBranch(context.Background(), "testowner", "testrepo", pr.Number)
	if err != pulls.ErrNotMergeable {
		t.Errorf("expected ErrNotMergeable (no git repo), got %v", err)
	}
}

// URL Population Tests

func TestService_PopulateURLs_PR(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")
	pr := createTestPR(t, service, "testowner", "testrepo", user.ID, "Test PR")

	if pr.URL == "" {
		t.Error("expected URL to be populated")
	}
	if pr.HTMLURL == "" {
		t.Error("expected HTMLURL to be populated")
	}
	if pr.DiffURL == "" {
		t.Error("expected DiffURL to be populated")
	}
	if pr.PatchURL == "" {
		t.Error("expected PatchURL to be populated")
	}
	if pr.IssueURL == "" {
		t.Error("expected IssueURL to be populated")
	}
	if pr.CommitsURL == "" {
		t.Error("expected CommitsURL to be populated")
	}
	if pr.NodeID == "" {
		t.Error("expected NodeID to be populated")
	}
}

func TestService_PopulateURLs_Review(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")
	pr := createTestPR(t, service, "testowner", "testrepo", user.ID, "Test PR")

	review, _ := service.CreateReview(context.Background(), "testowner", "testrepo", pr.Number, user.ID, &pulls.CreateReviewIn{
		Body:  "Review",
		Event: "COMMENT",
	})

	if review.HTMLURL == "" {
		t.Error("expected HTMLURL to be populated")
	}
	if review.PullRequestURL == "" {
		t.Error("expected PullRequestURL to be populated")
	}
	if review.NodeID == "" {
		t.Error("expected NodeID to be populated")
	}
}

func TestService_PopulateURLs_ReviewComment(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "testrepo")
	pr := createTestPR(t, service, "testowner", "testrepo", user.ID, "Test PR")

	comment, _ := service.CreateReviewComment(context.Background(), "testowner", "testrepo", pr.Number, user.ID, &pulls.CreateReviewCommentIn{
		Body:     "Comment",
		CommitID: "abc123",
		Path:     "main.go",
	})

	if comment.URL == "" {
		t.Error("expected URL to be populated")
	}
	if comment.HTMLURL == "" {
		t.Error("expected HTMLURL to be populated")
	}
	if comment.PullRequestURL == "" {
		t.Error("expected PullRequestURL to be populated")
	}
	if comment.NodeID == "" {
		t.Error("expected NodeID to be populated")
	}
}

// Multi-repo Isolation Tests

func TestService_MultipleRepos(t *testing.T) {
	service, usersStore, reposStore, cleanup := setupTestService(t)
	defer cleanup()

	user := createTestUser(t, usersStore, "testowner", "owner@example.com")
	createTestRepo(t, reposStore, user, "repo1")
	createTestRepo(t, reposStore, user, "repo2")

	// Create PRs in different repos
	pr1 := createTestPR(t, service, "testowner", "repo1", user.ID, "PR in repo1")
	pr2 := createTestPR(t, service, "testowner", "repo2", user.ID, "PR in repo2")

	// Both should have number 1 (scoped to repo)
	if pr1.Number != 1 {
		t.Errorf("expected repo1 PR number 1, got %d", pr1.Number)
	}
	if pr2.Number != 1 {
		t.Errorf("expected repo2 PR number 1, got %d", pr2.Number)
	}

	// List should be isolated
	list1, _ := service.List(context.Background(), "testowner", "repo1", nil)
	list2, _ := service.List(context.Background(), "testowner", "repo2", nil)

	if len(list1) != 1 {
		t.Errorf("repo1 should have 1 PR, got %d", len(list1))
	}
	if len(list2) != 1 {
		t.Errorf("repo2 should have 1 PR, got %d", len(list2))
	}
}
