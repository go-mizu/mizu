package comments_test

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"github.com/go-mizu/blueprints/githome/feature/comments"
	"github.com/go-mizu/blueprints/githome/feature/issues"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
	"github.com/go-mizu/blueprints/githome/store/duckdb"
)

type testEnv struct {
	service     *comments.Service
	store       *duckdb.Store
	issuesStore issues.Store
	db          *sql.DB
}

func setupTestService(t *testing.T) (*testEnv, func()) {
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

	commentsStore := duckdb.NewCommentsStore(db)
	issuesStore := duckdb.NewIssuesStore(db)
	service := comments.NewService(commentsStore, store.Repos(), issuesStore, store.Users(), "https://api.example.com")

	cleanup := func() {
		store.Close()
	}

	return &testEnv{
		service:     service,
		store:       store,
		issuesStore: issuesStore,
		db:          db,
	}, cleanup
}

func createTestUser(t *testing.T, env *testEnv, login, email string) *users.User {
	t.Helper()
	user := &users.User{
		Login:        login,
		Email:        email,
		Name:         "Test User",
		PasswordHash: "hash",
		Type:         "User",
	}
	if err := env.store.Users().Create(context.Background(), user); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return user
}

func createTestRepo(t *testing.T, env *testEnv, owner *users.User, name string) *repos.Repository {
	t.Helper()
	repo := &repos.Repository{
		Name:          name,
		FullName:      owner.Login + "/" + name,
		OwnerID:       owner.ID,
		OwnerType:     "User",
		Visibility:    "public",
		DefaultBranch: "main",
	}
	if err := env.store.Repos().Create(context.Background(), repo); err != nil {
		t.Fatalf("failed to create test repo: %v", err)
	}
	return repo
}

func createTestIssue(t *testing.T, env *testEnv, repo *repos.Repository, creator *users.User, title string) *issues.Issue {
	t.Helper()
	issue := &issues.Issue{
		Number:    1,
		Title:     title,
		Body:      "Test issue body",
		State:     "open",
		RepoID:    repo.ID,
		CreatorID: creator.ID,
	}
	if err := env.issuesStore.Create(context.Background(), issue); err != nil {
		t.Fatalf("failed to create test issue: %v", err)
	}
	return issue
}

// Issue Comment Tests

func TestService_CreateIssueComment_Success(t *testing.T) {
	env, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, env, "owner", "owner@example.com")
	repo := createTestRepo(t, env, owner, "testrepo")
	createTestIssue(t, env, repo, owner, "Test Issue")

	comment, err := env.service.CreateIssueComment(context.Background(), "owner", "testrepo", 1, owner.ID, "This is a comment")
	if err != nil {
		t.Fatalf("CreateIssueComment failed: %v", err)
	}

	if comment.Body != "This is a comment" {
		t.Errorf("got body %q, want This is a comment", comment.Body)
	}
	if comment.ID == 0 {
		t.Error("expected ID to be assigned")
	}
	if comment.User == nil {
		t.Error("expected user to be set")
	}
	if comment.AuthorAssociation != "OWNER" {
		t.Errorf("got author_association %q, want OWNER", comment.AuthorAssociation)
	}
	if comment.URL == "" {
		t.Error("expected URL to be populated")
	}
}

func TestService_CreateIssueComment_RepoNotFound(t *testing.T) {
	env, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, env, "owner", "owner@example.com")

	_, err := env.service.CreateIssueComment(context.Background(), "owner", "unknown", 1, owner.ID, "comment")
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

func TestService_CreateIssueComment_IssueNotFound(t *testing.T) {
	env, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, env, "owner", "owner@example.com")
	createTestRepo(t, env, owner, "testrepo")

	_, err := env.service.CreateIssueComment(context.Background(), "owner", "testrepo", 999, owner.ID, "comment")
	if err != issues.ErrNotFound {
		t.Errorf("expected issues.ErrNotFound, got %v", err)
	}
}

func TestService_CreateIssueComment_IncrementsCounter(t *testing.T) {
	env, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, env, "owner", "owner@example.com")
	repo := createTestRepo(t, env, owner, "testrepo")
	issue := createTestIssue(t, env, repo, owner, "Test Issue")

	_, _ = env.service.CreateIssueComment(context.Background(), "owner", "testrepo", 1, owner.ID, "Comment 1")
	_, _ = env.service.CreateIssueComment(context.Background(), "owner", "testrepo", 1, owner.ID, "Comment 2")

	// Verify issue comment count
	updatedIssue, _ := env.issuesStore.GetByID(context.Background(), issue.ID)
	if updatedIssue.Comments != 2 {
		t.Errorf("expected comments 2, got %d", updatedIssue.Comments)
	}
}

func TestService_GetIssueComment_Success(t *testing.T) {
	env, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, env, "owner", "owner@example.com")
	repo := createTestRepo(t, env, owner, "testrepo")
	createTestIssue(t, env, repo, owner, "Test Issue")

	created, _ := env.service.CreateIssueComment(context.Background(), "owner", "testrepo", 1, owner.ID, "Test comment")

	comment, err := env.service.GetIssueComment(context.Background(), "owner", "testrepo", created.ID)
	if err != nil {
		t.Fatalf("GetIssueComment failed: %v", err)
	}

	if comment.ID != created.ID {
		t.Errorf("got ID %d, want %d", comment.ID, created.ID)
	}
	if comment.Body != "Test comment" {
		t.Errorf("got body %q, want Test comment", comment.Body)
	}
}

func TestService_GetIssueComment_NotFound(t *testing.T) {
	env, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, env, "owner", "owner@example.com")
	createTestRepo(t, env, owner, "testrepo")

	_, err := env.service.GetIssueComment(context.Background(), "owner", "testrepo", 99999)
	if err != comments.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_UpdateIssueComment_Success(t *testing.T) {
	env, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, env, "owner", "owner@example.com")
	repo := createTestRepo(t, env, owner, "testrepo")
	createTestIssue(t, env, repo, owner, "Test Issue")

	created, _ := env.service.CreateIssueComment(context.Background(), "owner", "testrepo", 1, owner.ID, "Original")

	updated, err := env.service.UpdateIssueComment(context.Background(), "owner", "testrepo", created.ID, "Updated body")
	if err != nil {
		t.Fatalf("UpdateIssueComment failed: %v", err)
	}

	if updated.Body != "Updated body" {
		t.Errorf("got body %q, want Updated body", updated.Body)
	}
}

func TestService_UpdateIssueComment_NotFound(t *testing.T) {
	env, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, env, "owner", "owner@example.com")
	createTestRepo(t, env, owner, "testrepo")

	_, err := env.service.UpdateIssueComment(context.Background(), "owner", "testrepo", 99999, "body")
	if err != comments.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_DeleteIssueComment_Success(t *testing.T) {
	env, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, env, "owner", "owner@example.com")
	repo := createTestRepo(t, env, owner, "testrepo")
	createTestIssue(t, env, repo, owner, "Test Issue")

	created, _ := env.service.CreateIssueComment(context.Background(), "owner", "testrepo", 1, owner.ID, "To delete")

	err := env.service.DeleteIssueComment(context.Background(), "owner", "testrepo", created.ID)
	if err != nil {
		t.Fatalf("DeleteIssueComment failed: %v", err)
	}

	// Verify deleted
	_, err = env.service.GetIssueComment(context.Background(), "owner", "testrepo", created.ID)
	if err != comments.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestService_DeleteIssueComment_DecrementsCounter(t *testing.T) {
	env, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, env, "owner", "owner@example.com")
	repo := createTestRepo(t, env, owner, "testrepo")
	issue := createTestIssue(t, env, repo, owner, "Test Issue")

	comment1, _ := env.service.CreateIssueComment(context.Background(), "owner", "testrepo", 1, owner.ID, "Comment 1")
	_, _ = env.service.CreateIssueComment(context.Background(), "owner", "testrepo", 1, owner.ID, "Comment 2")

	// Delete one comment
	_ = env.service.DeleteIssueComment(context.Background(), "owner", "testrepo", comment1.ID)

	// Verify issue comment count
	updatedIssue, _ := env.issuesStore.GetByID(context.Background(), issue.ID)
	if updatedIssue.Comments != 1 {
		t.Errorf("expected comments 1, got %d", updatedIssue.Comments)
	}
}

func TestService_DeleteIssueComment_NotFound(t *testing.T) {
	env, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, env, "owner", "owner@example.com")
	createTestRepo(t, env, owner, "testrepo")

	err := env.service.DeleteIssueComment(context.Background(), "owner", "testrepo", 99999)
	if err != comments.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_ListForIssue(t *testing.T) {
	env, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, env, "owner", "owner@example.com")
	repo := createTestRepo(t, env, owner, "testrepo")
	createTestIssue(t, env, repo, owner, "Test Issue")

	_, _ = env.service.CreateIssueComment(context.Background(), "owner", "testrepo", 1, owner.ID, "Comment 1")
	_, _ = env.service.CreateIssueComment(context.Background(), "owner", "testrepo", 1, owner.ID, "Comment 2")
	_, _ = env.service.CreateIssueComment(context.Background(), "owner", "testrepo", 1, owner.ID, "Comment 3")

	list, err := env.service.ListForIssue(context.Background(), "owner", "testrepo", 1, nil)
	if err != nil {
		t.Fatalf("ListForIssue failed: %v", err)
	}

	if len(list) != 3 {
		t.Errorf("expected 3 comments, got %d", len(list))
	}
}

func TestService_ListForIssue_IssueNotFound(t *testing.T) {
	env, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, env, "owner", "owner@example.com")
	createTestRepo(t, env, owner, "testrepo")

	_, err := env.service.ListForIssue(context.Background(), "owner", "testrepo", 999, nil)
	if err != issues.ErrNotFound {
		t.Errorf("expected issues.ErrNotFound, got %v", err)
	}
}

func TestService_ListForIssue_Pagination(t *testing.T) {
	env, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, env, "owner", "owner@example.com")
	repo := createTestRepo(t, env, owner, "testrepo")
	createTestIssue(t, env, repo, owner, "Test Issue")

	for i := 0; i < 5; i++ {
		_, _ = env.service.CreateIssueComment(context.Background(), "owner", "testrepo", 1, owner.ID, "Comment")
	}

	list, err := env.service.ListForIssue(context.Background(), "owner", "testrepo", 1, &comments.ListOpts{
		Page:    1,
		PerPage: 2,
	})
	if err != nil {
		t.Fatalf("ListForIssue failed: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("expected 2 comments, got %d", len(list))
	}
}

func TestService_ListForRepo(t *testing.T) {
	env, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, env, "owner", "owner@example.com")
	repo := createTestRepo(t, env, owner, "testrepo")
	createTestIssue(t, env, repo, owner, "Test Issue")

	// Create second issue manually with number 2
	issue2 := &issues.Issue{
		Number:    2,
		Title:     "Test Issue 2",
		Body:      "Body",
		State:     "open",
		RepoID:    repo.ID,
		CreatorID: owner.ID,
	}
	_ = env.issuesStore.Create(context.Background(), issue2)

	_, _ = env.service.CreateIssueComment(context.Background(), "owner", "testrepo", 1, owner.ID, "Comment on issue 1")
	_, _ = env.service.CreateIssueComment(context.Background(), "owner", "testrepo", 2, owner.ID, "Comment on issue 2")

	list, err := env.service.ListForRepo(context.Background(), "owner", "testrepo", nil)
	if err != nil {
		t.Fatalf("ListForRepo failed: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("expected 2 comments, got %d", len(list))
	}
}

func TestService_ListForRepo_RepoNotFound(t *testing.T) {
	env, cleanup := setupTestService(t)
	defer cleanup()

	_, err := env.service.ListForRepo(context.Background(), "unknown", "repo", nil)
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

// Commit Comment Tests

func TestService_CreateCommitComment_Success(t *testing.T) {
	env, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, env, "owner", "owner@example.com")
	createTestRepo(t, env, owner, "testrepo")

	comment, err := env.service.CreateCommitComment(context.Background(), "owner", "testrepo", "abc123", owner.ID, &comments.CreateCommitCommentIn{
		Body:     "This is a commit comment",
		Path:     "file.go",
		Position: 10,
		Line:     5,
	})
	if err != nil {
		t.Fatalf("CreateCommitComment failed: %v", err)
	}

	if comment.Body != "This is a commit comment" {
		t.Errorf("got body %q, want This is a commit comment", comment.Body)
	}
	if comment.CommitID != "abc123" {
		t.Errorf("got commit_id %q, want abc123", comment.CommitID)
	}
	if comment.Path != "file.go" {
		t.Errorf("got path %q, want file.go", comment.Path)
	}
	if comment.Position != 10 {
		t.Errorf("got position %d, want 10", comment.Position)
	}
	if comment.Line != 5 {
		t.Errorf("got line %d, want 5", comment.Line)
	}
	if comment.ID == 0 {
		t.Error("expected ID to be assigned")
	}
	if comment.URL == "" {
		t.Error("expected URL to be populated")
	}
}

func TestService_CreateCommitComment_RepoNotFound(t *testing.T) {
	env, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, env, "owner", "owner@example.com")

	_, err := env.service.CreateCommitComment(context.Background(), "owner", "unknown", "abc123", owner.ID, &comments.CreateCommitCommentIn{
		Body: "comment",
	})
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

func TestService_GetCommitComment_Success(t *testing.T) {
	env, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, env, "owner", "owner@example.com")
	createTestRepo(t, env, owner, "testrepo")

	created, _ := env.service.CreateCommitComment(context.Background(), "owner", "testrepo", "abc123", owner.ID, &comments.CreateCommitCommentIn{
		Body: "Test commit comment",
	})

	comment, err := env.service.GetCommitComment(context.Background(), "owner", "testrepo", created.ID)
	if err != nil {
		t.Fatalf("GetCommitComment failed: %v", err)
	}

	if comment.ID != created.ID {
		t.Errorf("got ID %d, want %d", comment.ID, created.ID)
	}
	if comment.Body != "Test commit comment" {
		t.Errorf("got body %q, want Test commit comment", comment.Body)
	}
}

func TestService_GetCommitComment_NotFound(t *testing.T) {
	env, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, env, "owner", "owner@example.com")
	createTestRepo(t, env, owner, "testrepo")

	_, err := env.service.GetCommitComment(context.Background(), "owner", "testrepo", 99999)
	if err != comments.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_UpdateCommitComment_Success(t *testing.T) {
	env, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, env, "owner", "owner@example.com")
	createTestRepo(t, env, owner, "testrepo")

	created, _ := env.service.CreateCommitComment(context.Background(), "owner", "testrepo", "abc123", owner.ID, &comments.CreateCommitCommentIn{
		Body: "Original",
	})

	updated, err := env.service.UpdateCommitComment(context.Background(), "owner", "testrepo", created.ID, "Updated body")
	if err != nil {
		t.Fatalf("UpdateCommitComment failed: %v", err)
	}

	if updated.Body != "Updated body" {
		t.Errorf("got body %q, want Updated body", updated.Body)
	}
}

func TestService_UpdateCommitComment_NotFound(t *testing.T) {
	env, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, env, "owner", "owner@example.com")
	createTestRepo(t, env, owner, "testrepo")

	_, err := env.service.UpdateCommitComment(context.Background(), "owner", "testrepo", 99999, "body")
	if err != comments.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_DeleteCommitComment_Success(t *testing.T) {
	env, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, env, "owner", "owner@example.com")
	createTestRepo(t, env, owner, "testrepo")

	created, _ := env.service.CreateCommitComment(context.Background(), "owner", "testrepo", "abc123", owner.ID, &comments.CreateCommitCommentIn{
		Body: "To delete",
	})

	err := env.service.DeleteCommitComment(context.Background(), "owner", "testrepo", created.ID)
	if err != nil {
		t.Fatalf("DeleteCommitComment failed: %v", err)
	}

	// Verify deleted
	_, err = env.service.GetCommitComment(context.Background(), "owner", "testrepo", created.ID)
	if err != comments.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestService_DeleteCommitComment_NotFound(t *testing.T) {
	env, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, env, "owner", "owner@example.com")
	createTestRepo(t, env, owner, "testrepo")

	err := env.service.DeleteCommitComment(context.Background(), "owner", "testrepo", 99999)
	if err != comments.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_ListForCommit(t *testing.T) {
	env, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, env, "owner", "owner@example.com")
	createTestRepo(t, env, owner, "testrepo")

	_, _ = env.service.CreateCommitComment(context.Background(), "owner", "testrepo", "abc123", owner.ID, &comments.CreateCommitCommentIn{Body: "Comment 1"})
	_, _ = env.service.CreateCommitComment(context.Background(), "owner", "testrepo", "abc123", owner.ID, &comments.CreateCommitCommentIn{Body: "Comment 2"})
	_, _ = env.service.CreateCommitComment(context.Background(), "owner", "testrepo", "def456", owner.ID, &comments.CreateCommitCommentIn{Body: "Different commit"})

	list, err := env.service.ListForCommit(context.Background(), "owner", "testrepo", "abc123", nil)
	if err != nil {
		t.Fatalf("ListForCommit failed: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("expected 2 comments for abc123, got %d", len(list))
	}
}

func TestService_ListCommitCommentsForRepo(t *testing.T) {
	env, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, env, "owner", "owner@example.com")
	createTestRepo(t, env, owner, "testrepo")

	_, _ = env.service.CreateCommitComment(context.Background(), "owner", "testrepo", "abc123", owner.ID, &comments.CreateCommitCommentIn{Body: "Comment 1"})
	_, _ = env.service.CreateCommitComment(context.Background(), "owner", "testrepo", "def456", owner.ID, &comments.CreateCommitCommentIn{Body: "Comment 2"})

	list, err := env.service.ListCommitCommentsForRepo(context.Background(), "owner", "testrepo", nil)
	if err != nil {
		t.Fatalf("ListCommitCommentsForRepo failed: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("expected 2 commit comments, got %d", len(list))
	}
}

func TestService_ListCommitCommentsForRepo_RepoNotFound(t *testing.T) {
	env, cleanup := setupTestService(t)
	defer cleanup()

	_, err := env.service.ListCommitCommentsForRepo(context.Background(), "unknown", "repo", nil)
	if err != repos.ErrNotFound {
		t.Errorf("expected repos.ErrNotFound, got %v", err)
	}
}

// URL Population Tests

func TestService_PopulateIssueCommentURLs(t *testing.T) {
	env, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, env, "owner", "owner@example.com")
	repo := createTestRepo(t, env, owner, "testrepo")
	createTestIssue(t, env, repo, owner, "Test Issue")

	comment, _ := env.service.CreateIssueComment(context.Background(), "owner", "testrepo", 1, owner.ID, "Test")

	if comment.URL == "" {
		t.Error("expected URL to be set")
	}
	if comment.HTMLURL == "" {
		t.Error("expected HTMLURL to be set")
	}
	if comment.IssueURL == "" {
		t.Error("expected IssueURL to be set")
	}
	if comment.NodeID == "" {
		t.Error("expected NodeID to be set")
	}
}

func TestService_PopulateCommitCommentURLs(t *testing.T) {
	env, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, env, "owner", "owner@example.com")
	createTestRepo(t, env, owner, "testrepo")

	comment, _ := env.service.CreateCommitComment(context.Background(), "owner", "testrepo", "abc123", owner.ID, &comments.CreateCommitCommentIn{
		Body: "Test",
	})

	if comment.URL == "" {
		t.Error("expected URL to be set")
	}
	if comment.HTMLURL == "" {
		t.Error("expected HTMLURL to be set")
	}
	if comment.NodeID == "" {
		t.Error("expected NodeID to be set")
	}
}

// Author Association Tests

func TestService_AuthorAssociation_Owner(t *testing.T) {
	env, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, env, "owner", "owner@example.com")
	repo := createTestRepo(t, env, owner, "testrepo")
	createTestIssue(t, env, repo, owner, "Test Issue")

	comment, _ := env.service.CreateIssueComment(context.Background(), "owner", "testrepo", 1, owner.ID, "By owner")

	if comment.AuthorAssociation != "OWNER" {
		t.Errorf("expected author_association OWNER, got %q", comment.AuthorAssociation)
	}
}

func TestService_AuthorAssociation_None(t *testing.T) {
	env, cleanup := setupTestService(t)
	defer cleanup()

	owner := createTestUser(t, env, "owner", "owner@example.com")
	other := createTestUser(t, env, "other", "other@example.com")
	repo := createTestRepo(t, env, owner, "testrepo")
	createTestIssue(t, env, repo, owner, "Test Issue")

	comment, _ := env.service.CreateIssueComment(context.Background(), "owner", "testrepo", 1, other.ID, "By other")

	if comment.AuthorAssociation != "NONE" {
		t.Errorf("expected author_association NONE, got %q", comment.AuthorAssociation)
	}
}
