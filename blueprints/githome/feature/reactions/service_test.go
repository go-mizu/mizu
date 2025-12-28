package reactions_test

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"github.com/go-mizu/blueprints/githome/feature/comments"
	"github.com/go-mizu/blueprints/githome/feature/issues"
	"github.com/go-mizu/blueprints/githome/feature/reactions"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
	"github.com/go-mizu/blueprints/githome/store/duckdb"
)

// testContext holds all the stores and service for tests
type testContext struct {
	service       *reactions.Service
	store         *duckdb.Store
	usersStore    *duckdb.UsersStore
	reposStore    *duckdb.ReposStore
	issuesStore   *duckdb.IssuesStore
	commentsStore *duckdb.CommentsStore
	cleanup       func()
}

func setupTestService(t *testing.T) *testContext {
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

	reactionsStore := duckdb.NewReactionsStore(db)
	usersStore := duckdb.NewUsersStore(db)
	reposStore := duckdb.NewReposStore(db)
	issuesStore := duckdb.NewIssuesStore(db)
	commentsStore := duckdb.NewCommentsStore(db)
	service := reactions.NewService(reactionsStore, reposStore, issuesStore, commentsStore, "https://api.example.com")

	return &testContext{
		service:       service,
		store:         store,
		usersStore:    usersStore,
		reposStore:    reposStore,
		issuesStore:   issuesStore,
		commentsStore: commentsStore,
		cleanup: func() {
			store.Close()
		},
	}
}

func createTestUser(t *testing.T, tc *testContext, login, email string) *users.User {
	t.Helper()
	user := &users.User{
		Login:        login,
		Email:        email,
		Name:         "Test User",
		PasswordHash: "hash",
		Type:         "User",
	}
	if err := tc.usersStore.Create(context.Background(), user); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return user
}

func createTestRepo(t *testing.T, tc *testContext, owner *users.User, name string) *repos.Repository {
	t.Helper()
	repo := &repos.Repository{
		Name:          name,
		FullName:      owner.Login + "/" + name,
		OwnerID:       owner.ID,
		OwnerType:     "User",
		Visibility:    "public",
		DefaultBranch: "main",
	}
	if err := tc.reposStore.Create(context.Background(), repo); err != nil {
		t.Fatalf("failed to create test repo: %v", err)
	}
	return repo
}

func createTestIssue(t *testing.T, tc *testContext, repo *repos.Repository, creator *users.User, number int, title string) *issues.Issue {
	t.Helper()
	issue := &issues.Issue{
		Number:    number,
		Title:     title,
		Body:      "Test issue body",
		State:     "open",
		RepoID:    repo.ID,
		CreatorID: creator.ID,
	}
	if err := tc.issuesStore.Create(context.Background(), issue); err != nil {
		t.Fatalf("failed to create test issue: %v", err)
	}
	return issue
}

func createTestIssueComment(t *testing.T, tc *testContext, issue *issues.Issue, repo *repos.Repository, creator *users.User) *comments.IssueComment {
	t.Helper()
	comment := &comments.IssueComment{
		Body:      "Test comment",
		IssueID:   issue.ID,
		RepoID:    repo.ID,
		CreatorID: creator.ID,
	}
	if err := tc.commentsStore.CreateIssueComment(context.Background(), comment); err != nil {
		t.Fatalf("failed to create test comment: %v", err)
	}
	return comment
}

func createTestCommitComment(t *testing.T, tc *testContext, repo *repos.Repository, creator *users.User) *comments.CommitComment {
	t.Helper()
	comment := &comments.CommitComment{
		Body:      "Test commit comment",
		CommitID:  "abc123",
		RepoID:    repo.ID,
		CreatorID: creator.ID,
	}
	if err := tc.commentsStore.CreateCommitComment(context.Background(), comment); err != nil {
		t.Fatalf("failed to create test commit comment: %v", err)
	}
	return comment
}

// Issue Reaction Tests

func TestService_CreateForIssue_Success(t *testing.T) {
	tc := setupTestService(t)
	defer tc.cleanup()

	owner := createTestUser(t, tc, "owner", "owner@example.com")
	repo := createTestRepo(t, tc, owner, "testrepo")
	createTestIssue(t, tc, repo, owner, 1, "Test Issue")

	reaction, err := tc.service.CreateForIssue(context.Background(), "owner", "testrepo", 1, owner.ID, reactions.ContentPlusOne)
	if err != nil {
		t.Fatalf("CreateForIssue failed: %v", err)
	}

	if reaction.Content != reactions.ContentPlusOne {
		t.Errorf("got content %q, want %q", reaction.Content, reactions.ContentPlusOne)
	}
	if reaction.ID == 0 {
		t.Error("expected ID to be assigned")
	}
	if reaction.User == nil {
		t.Error("expected user to be set")
	}
}

func TestService_CreateForIssue_InvalidContent(t *testing.T) {
	tc := setupTestService(t)
	defer tc.cleanup()

	owner := createTestUser(t, tc, "owner", "owner@example.com")
	repo := createTestRepo(t, tc, owner, "testrepo")
	createTestIssue(t, tc, repo, owner, 1, "Test Issue")

	_, err := tc.service.CreateForIssue(context.Background(), "owner", "testrepo", 1, owner.ID, "invalid")
	if err != reactions.ErrInvalidContent {
		t.Errorf("expected ErrInvalidContent, got %v", err)
	}
}

func TestService_CreateForIssue_Idempotent(t *testing.T) {
	tc := setupTestService(t)
	defer tc.cleanup()

	owner := createTestUser(t, tc, "owner", "owner@example.com")
	repo := createTestRepo(t, tc, owner, "testrepo")
	createTestIssue(t, tc, repo, owner, 1, "Test Issue")

	// First reaction
	reaction1, _ := tc.service.CreateForIssue(context.Background(), "owner", "testrepo", 1, owner.ID, reactions.ContentPlusOne)

	// Second identical reaction should return existing
	reaction2, err := tc.service.CreateForIssue(context.Background(), "owner", "testrepo", 1, owner.ID, reactions.ContentPlusOne)
	if err != nil {
		t.Fatalf("CreateForIssue should be idempotent: %v", err)
	}

	if reaction2.ID != reaction1.ID {
		t.Error("expected same reaction to be returned")
	}
}

func TestService_CreateForIssue_DifferentContent(t *testing.T) {
	tc := setupTestService(t)
	defer tc.cleanup()

	owner := createTestUser(t, tc, "owner", "owner@example.com")
	repo := createTestRepo(t, tc, owner, "testrepo")
	createTestIssue(t, tc, repo, owner, 1, "Test Issue")

	// Add different reaction types
	r1, _ := tc.service.CreateForIssue(context.Background(), "owner", "testrepo", 1, owner.ID, reactions.ContentPlusOne)
	r2, _ := tc.service.CreateForIssue(context.Background(), "owner", "testrepo", 1, owner.ID, reactions.ContentHeart)

	if r1.ID == r2.ID {
		t.Error("different reaction types should have different IDs")
	}
}

func TestService_CreateForIssue_IssueNotFound(t *testing.T) {
	tc := setupTestService(t)
	defer tc.cleanup()

	owner := createTestUser(t, tc, "owner", "owner@example.com")
	createTestRepo(t, tc, owner, "testrepo")

	_, err := tc.service.CreateForIssue(context.Background(), "owner", "testrepo", 999, owner.ID, reactions.ContentPlusOne)
	if err != issues.ErrNotFound {
		t.Errorf("expected issues.ErrNotFound, got %v", err)
	}
}

func TestService_ListForIssue(t *testing.T) {
	tc := setupTestService(t)
	defer tc.cleanup()

	owner := createTestUser(t, tc, "owner", "owner@example.com")
	user2 := createTestUser(t, tc, "user2", "user2@example.com")
	repo := createTestRepo(t, tc, owner, "testrepo")
	createTestIssue(t, tc, repo, owner, 1, "Test Issue")

	_, _ = tc.service.CreateForIssue(context.Background(), "owner", "testrepo", 1, owner.ID, reactions.ContentPlusOne)
	_, _ = tc.service.CreateForIssue(context.Background(), "owner", "testrepo", 1, user2.ID, reactions.ContentHeart)

	list, err := tc.service.ListForIssue(context.Background(), "owner", "testrepo", 1, nil)
	if err != nil {
		t.Fatalf("ListForIssue failed: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("expected 2 reactions, got %d", len(list))
	}
}

func TestService_DeleteForIssue_Success(t *testing.T) {
	tc := setupTestService(t)
	defer tc.cleanup()

	owner := createTestUser(t, tc, "owner", "owner@example.com")
	repo := createTestRepo(t, tc, owner, "testrepo")
	createTestIssue(t, tc, repo, owner, 1, "Test Issue")

	reaction, _ := tc.service.CreateForIssue(context.Background(), "owner", "testrepo", 1, owner.ID, reactions.ContentPlusOne)

	err := tc.service.DeleteForIssue(context.Background(), "owner", "testrepo", 1, reaction.ID)
	if err != nil {
		t.Fatalf("DeleteForIssue failed: %v", err)
	}

	// Verify deleted
	list, _ := tc.service.ListForIssue(context.Background(), "owner", "testrepo", 1, nil)
	if len(list) != 0 {
		t.Error("expected reaction to be deleted")
	}
}

// Issue Comment Reaction Tests

func TestService_CreateForIssueComment_Success(t *testing.T) {
	tc := setupTestService(t)
	defer tc.cleanup()

	owner := createTestUser(t, tc, "owner", "owner@example.com")
	repo := createTestRepo(t, tc, owner, "testrepo")
	issue := createTestIssue(t, tc, repo, owner, 1, "Test Issue")
	comment := createTestIssueComment(t, tc, issue, repo, owner)

	reaction, err := tc.service.CreateForIssueComment(context.Background(), "owner", "testrepo", comment.ID, owner.ID, reactions.ContentRocket)
	if err != nil {
		t.Fatalf("CreateForIssueComment failed: %v", err)
	}

	if reaction.Content != reactions.ContentRocket {
		t.Errorf("got content %q, want %q", reaction.Content, reactions.ContentRocket)
	}
}

func TestService_CreateForIssueComment_CommentNotFound(t *testing.T) {
	tc := setupTestService(t)
	defer tc.cleanup()

	owner := createTestUser(t, tc, "owner", "owner@example.com")
	createTestRepo(t, tc, owner, "testrepo")

	_, err := tc.service.CreateForIssueComment(context.Background(), "owner", "testrepo", 99999, owner.ID, reactions.ContentPlusOne)
	if err != comments.ErrNotFound {
		t.Errorf("expected comments.ErrNotFound, got %v", err)
	}
}

func TestService_ListForIssueComment(t *testing.T) {
	tc := setupTestService(t)
	defer tc.cleanup()

	owner := createTestUser(t, tc, "owner", "owner@example.com")
	repo := createTestRepo(t, tc, owner, "testrepo")
	issue := createTestIssue(t, tc, repo, owner, 1, "Test Issue")
	comment := createTestIssueComment(t, tc, issue, repo, owner)

	_, _ = tc.service.CreateForIssueComment(context.Background(), "owner", "testrepo", comment.ID, owner.ID, reactions.ContentPlusOne)
	_, _ = tc.service.CreateForIssueComment(context.Background(), "owner", "testrepo", comment.ID, owner.ID, reactions.ContentLaugh)

	list, err := tc.service.ListForIssueComment(context.Background(), "owner", "testrepo", comment.ID, nil)
	if err != nil {
		t.Fatalf("ListForIssueComment failed: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("expected 2 reactions, got %d", len(list))
	}
}

func TestService_DeleteForIssueComment_Success(t *testing.T) {
	tc := setupTestService(t)
	defer tc.cleanup()

	owner := createTestUser(t, tc, "owner", "owner@example.com")
	repo := createTestRepo(t, tc, owner, "testrepo")
	issue := createTestIssue(t, tc, repo, owner, 1, "Test Issue")
	comment := createTestIssueComment(t, tc, issue, repo, owner)

	reaction, _ := tc.service.CreateForIssueComment(context.Background(), "owner", "testrepo", comment.ID, owner.ID, reactions.ContentPlusOne)

	err := tc.service.DeleteForIssueComment(context.Background(), "owner", "testrepo", comment.ID, reaction.ID)
	if err != nil {
		t.Fatalf("DeleteForIssueComment failed: %v", err)
	}
}

// Commit Comment Reaction Tests

func TestService_CreateForCommitComment_Success(t *testing.T) {
	tc := setupTestService(t)
	defer tc.cleanup()

	owner := createTestUser(t, tc, "owner", "owner@example.com")
	repo := createTestRepo(t, tc, owner, "testrepo")
	comment := createTestCommitComment(t, tc, repo, owner)

	reaction, err := tc.service.CreateForCommitComment(context.Background(), "owner", "testrepo", comment.ID, owner.ID, reactions.ContentEyes)
	if err != nil {
		t.Fatalf("CreateForCommitComment failed: %v", err)
	}

	if reaction.Content != reactions.ContentEyes {
		t.Errorf("got content %q, want %q", reaction.Content, reactions.ContentEyes)
	}
}

func TestService_ListForCommitComment(t *testing.T) {
	tc := setupTestService(t)
	defer tc.cleanup()

	owner := createTestUser(t, tc, "owner", "owner@example.com")
	repo := createTestRepo(t, tc, owner, "testrepo")
	comment := createTestCommitComment(t, tc, repo, owner)

	_, _ = tc.service.CreateForCommitComment(context.Background(), "owner", "testrepo", comment.ID, owner.ID, reactions.ContentPlusOne)

	list, err := tc.service.ListForCommitComment(context.Background(), "owner", "testrepo", comment.ID, nil)
	if err != nil {
		t.Fatalf("ListForCommitComment failed: %v", err)
	}

	if len(list) != 1 {
		t.Errorf("expected 1 reaction, got %d", len(list))
	}
}

func TestService_DeleteForCommitComment_Success(t *testing.T) {
	tc := setupTestService(t)
	defer tc.cleanup()

	owner := createTestUser(t, tc, "owner", "owner@example.com")
	repo := createTestRepo(t, tc, owner, "testrepo")
	comment := createTestCommitComment(t, tc, repo, owner)

	reaction, _ := tc.service.CreateForCommitComment(context.Background(), "owner", "testrepo", comment.ID, owner.ID, reactions.ContentPlusOne)

	err := tc.service.DeleteForCommitComment(context.Background(), "owner", "testrepo", comment.ID, reaction.ID)
	if err != nil {
		t.Fatalf("DeleteForCommitComment failed: %v", err)
	}
}

// Rollup Tests

func TestService_GetRollup(t *testing.T) {
	tc := setupTestService(t)
	defer tc.cleanup()

	owner := createTestUser(t, tc, "owner", "owner@example.com")
	user2 := createTestUser(t, tc, "user2", "user2@example.com")
	repo := createTestRepo(t, tc, owner, "testrepo")
	issue := createTestIssue(t, tc, repo, owner, 1, "Test Issue")

	// Add various reactions
	_, _ = tc.service.CreateForIssue(context.Background(), "owner", "testrepo", 1, owner.ID, reactions.ContentPlusOne)
	_, _ = tc.service.CreateForIssue(context.Background(), "owner", "testrepo", 1, user2.ID, reactions.ContentPlusOne)
	_, _ = tc.service.CreateForIssue(context.Background(), "owner", "testrepo", 1, owner.ID, reactions.ContentHeart)

	rollup, err := tc.service.GetRollup(context.Background(), "issue", issue.ID)
	if err != nil {
		t.Fatalf("GetRollup failed: %v", err)
	}

	if rollup.TotalCount != 3 {
		t.Errorf("expected total_count 3, got %d", rollup.TotalCount)
	}
	if rollup.PlusOne != 2 {
		t.Errorf("expected +1 count 2, got %d", rollup.PlusOne)
	}
	if rollup.Heart != 1 {
		t.Errorf("expected heart count 1, got %d", rollup.Heart)
	}
	if rollup.URL == "" {
		t.Error("expected URL to be set")
	}
}

// ValidContent Tests

func TestValidContent(t *testing.T) {
	validContents := []string{
		reactions.ContentPlusOne,
		reactions.ContentMinusOne,
		reactions.ContentLaugh,
		reactions.ContentConfused,
		reactions.ContentHeart,
		reactions.ContentHooray,
		reactions.ContentRocket,
		reactions.ContentEyes,
	}

	for _, content := range validContents {
		if !reactions.ValidContent(content) {
			t.Errorf("expected %q to be valid", content)
		}
	}

	invalidContents := []string{"", "invalid", "thumbsup", "like"}
	for _, content := range invalidContents {
		if reactions.ValidContent(content) {
			t.Errorf("expected %q to be invalid", content)
		}
	}
}

// All Reaction Types Test

func TestService_AllReactionTypes(t *testing.T) {
	tc := setupTestService(t)
	defer tc.cleanup()

	owner := createTestUser(t, tc, "owner", "owner@example.com")
	repo := createTestRepo(t, tc, owner, "testrepo")
	createTestIssue(t, tc, repo, owner, 1, "Test Issue")

	allTypes := []string{
		reactions.ContentPlusOne,
		reactions.ContentMinusOne,
		reactions.ContentLaugh,
		reactions.ContentConfused,
		reactions.ContentHeart,
		reactions.ContentHooray,
		reactions.ContentRocket,
		reactions.ContentEyes,
	}

	for _, content := range allTypes {
		r, err := tc.service.CreateForIssue(context.Background(), "owner", "testrepo", 1, owner.ID, content)
		if err != nil {
			t.Errorf("failed to create reaction with content %q: %v", content, err)
			continue
		}
		if r.Content != content {
			t.Errorf("got content %q, want %q", r.Content, content)
		}
	}

	// Verify all were created
	list, _ := tc.service.ListForIssue(context.Background(), "owner", "testrepo", 1, nil)
	if len(list) != len(allTypes) {
		t.Errorf("expected %d reactions, got %d", len(allTypes), len(list))
	}
}

// Integration Test - Reactions Across Different Subjects

func TestService_ReactionsAcrossSubjects(t *testing.T) {
	tc := setupTestService(t)
	defer tc.cleanup()

	owner := createTestUser(t, tc, "owner", "owner@example.com")
	repo := createTestRepo(t, tc, owner, "testrepo")
	issue := createTestIssue(t, tc, repo, owner, 1, "Test Issue")
	issueComment := createTestIssueComment(t, tc, issue, repo, owner)
	commitComment := createTestCommitComment(t, tc, repo, owner)

	// Add reactions to different subjects
	_, _ = tc.service.CreateForIssue(context.Background(), "owner", "testrepo", 1, owner.ID, reactions.ContentPlusOne)
	_, _ = tc.service.CreateForIssueComment(context.Background(), "owner", "testrepo", issueComment.ID, owner.ID, reactions.ContentPlusOne)
	_, _ = tc.service.CreateForCommitComment(context.Background(), "owner", "testrepo", commitComment.ID, owner.ID, reactions.ContentPlusOne)

	// Verify each has its own reactions
	issueReactions, _ := tc.service.ListForIssue(context.Background(), "owner", "testrepo", 1, nil)
	issueCommentReactions, _ := tc.service.ListForIssueComment(context.Background(), "owner", "testrepo", issueComment.ID, nil)
	commitCommentReactions, _ := tc.service.ListForCommitComment(context.Background(), "owner", "testrepo", commitComment.ID, nil)

	if len(issueReactions) != 1 {
		t.Errorf("expected 1 issue reaction, got %d", len(issueReactions))
	}
	if len(issueCommentReactions) != 1 {
		t.Errorf("expected 1 issue comment reaction, got %d", len(issueCommentReactions))
	}
	if len(commitCommentReactions) != 1 {
		t.Errorf("expected 1 commit comment reaction, got %d", len(commitCommentReactions))
	}
}
