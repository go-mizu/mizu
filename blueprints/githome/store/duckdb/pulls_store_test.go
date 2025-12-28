package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/pulls"
	"github.com/oklog/ulid/v2"
)

func createTestPullRequest(t *testing.T, store *PullsStore, repoID, authorID string, number int) *pulls.PullRequest {
	t.Helper()
	id := ulid.Make().String()
	pr := &pulls.PullRequest{
		ID:         id,
		RepoID:     repoID,
		Number:     number,
		Title:      "PR " + id[len(id)-8:],
		Body:       "Test pull request body",
		AuthorID:   authorID,
		HeadBranch: "feature-branch",
		HeadSHA:    "abc123",
		BaseBranch: "main",
		BaseSHA:    "def456",
		State:      "open",
		IsDraft:    false,
		IsLocked:   false,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := store.Create(context.Background(), pr); err != nil {
		t.Fatalf("failed to create test pull request: %v", err)
	}
	return pr
}

// =============================================================================
// Pull Request CRUD Tests
// =============================================================================

func TestPullsStore_Create(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	pullsStore := NewPullsStore(store.DB())

	pr := &pulls.PullRequest{
		ID:           ulid.Make().String(),
		RepoID:       repoID,
		Number:       1,
		Title:        "Add new feature",
		Body:         "This PR adds a new feature",
		AuthorID:     userID,
		HeadBranch:   "feature/new-feature",
		HeadSHA:      "abc123def456",
		BaseBranch:   "main",
		BaseSHA:      "789xyz",
		State:        "open",
		IsDraft:      false,
		IsLocked:     false,
		Additions:    100,
		Deletions:    50,
		ChangedFiles: 10,
		Commits:      3,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err := pullsStore.Create(context.Background(), pr)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := pullsStore.GetByID(context.Background(), pr.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected pull request to be created")
	}
	if got.Title != "Add new feature" {
		t.Errorf("got title %q, want %q", got.Title, "Add new feature")
	}
	if got.Number != 1 {
		t.Errorf("got number %d, want 1", got.Number)
	}
	if got.Additions != 100 {
		t.Errorf("got additions %d, want 100", got.Additions)
	}
}

func TestPullsStore_Create_Draft(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	pullsStore := NewPullsStore(store.DB())

	pr := &pulls.PullRequest{
		ID:         ulid.Make().String(),
		RepoID:     repoID,
		Number:     1,
		Title:      "Draft PR",
		AuthorID:   userID,
		HeadBranch: "feature",
		HeadSHA:    "abc123",
		BaseBranch: "main",
		BaseSHA:    "def456",
		State:      "open",
		IsDraft:    true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	pullsStore.Create(context.Background(), pr)

	got, _ := pullsStore.GetByID(context.Background(), pr.ID)
	if !got.IsDraft {
		t.Error("expected PR to be draft")
	}
}

func TestPullsStore_GetByID(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	pullsStore := NewPullsStore(store.DB())

	pr := createTestPullRequest(t, pullsStore, repoID, userID, 1)

	got, err := pullsStore.GetByID(context.Background(), pr.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected pull request")
	}
	if got.ID != pr.ID {
		t.Errorf("got ID %q, want %q", got.ID, pr.ID)
	}
}

func TestPullsStore_GetByID_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	pullsStore := NewPullsStore(store.DB())

	got, err := pullsStore.GetByID(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent PR")
	}
}

func TestPullsStore_GetByNumber(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	pullsStore := NewPullsStore(store.DB())

	pr := createTestPullRequest(t, pullsStore, repoID, userID, 42)

	got, err := pullsStore.GetByNumber(context.Background(), repoID, 42)
	if err != nil {
		t.Fatalf("GetByNumber failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected pull request")
	}
	if got.Number != 42 {
		t.Errorf("got number %d, want 42", got.Number)
	}
	if got.ID != pr.ID {
		t.Errorf("got ID %q, want %q", got.ID, pr.ID)
	}
}

func TestPullsStore_GetByNumber_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	pullsStore := NewPullsStore(store.DB())

	got, err := pullsStore.GetByNumber(context.Background(), repoID, 999)
	if err != nil {
		t.Fatalf("GetByNumber failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent PR number")
	}
}

func TestPullsStore_Update(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	pullsStore := NewPullsStore(store.DB())

	pr := createTestPullRequest(t, pullsStore, repoID, userID, 1)

	pr.Title = "Updated Title"
	pr.Body = "Updated body"
	pr.State = "closed"

	err := pullsStore.Update(context.Background(), pr)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, _ := pullsStore.GetByID(context.Background(), pr.ID)
	if got.Title != "Updated Title" {
		t.Errorf("got title %q, want %q", got.Title, "Updated Title")
	}
	if got.State != "closed" {
		t.Errorf("got state %q, want %q", got.State, "closed")
	}
}

func TestPullsStore_Update_Merge(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	pullsStore := NewPullsStore(store.DB())

	pr := createTestPullRequest(t, pullsStore, repoID, userID, 1)

	mergedAt := time.Now()
	pr.State = "closed"
	pr.MergedAt = &mergedAt
	pr.MergedByID = userID
	pr.MergeCommitSHA = "merged123"

	pullsStore.Update(context.Background(), pr)

	got, _ := pullsStore.GetByID(context.Background(), pr.ID)
	if got.MergedAt == nil {
		t.Error("expected merged_at to be set")
	}
	if got.MergedByID != userID {
		t.Errorf("got merged_by_id %q, want %q", got.MergedByID, userID)
	}
	if got.MergeCommitSHA != "merged123" {
		t.Errorf("got merge_commit_sha %q, want %q", got.MergeCommitSHA, "merged123")
	}
}

func TestPullsStore_Delete(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	pullsStore := NewPullsStore(store.DB())

	pr := createTestPullRequest(t, pullsStore, repoID, userID, 1)

	err := pullsStore.Delete(context.Background(), pr.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	got, _ := pullsStore.GetByID(context.Background(), pr.ID)
	if got != nil {
		t.Error("expected PR to be deleted")
	}
}

// =============================================================================
// List Tests
// =============================================================================

func TestPullsStore_List(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	pullsStore := NewPullsStore(store.DB())

	for i := 1; i <= 5; i++ {
		createTestPullRequest(t, pullsStore, repoID, userID, i)
	}

	list, total, err := pullsStore.List(context.Background(), repoID, "", 10, 0)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 5 {
		t.Errorf("got %d PRs, want 5", len(list))
	}
	if total != 5 {
		t.Errorf("got total %d, want 5", total)
	}
}

func TestPullsStore_List_FilterByState(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	pullsStore := NewPullsStore(store.DB())

	// Create open PRs
	for i := 1; i <= 3; i++ {
		createTestPullRequest(t, pullsStore, repoID, userID, i)
	}

	// Create closed PRs
	for i := 4; i <= 5; i++ {
		pr := createTestPullRequest(t, pullsStore, repoID, userID, i)
		pr.State = "closed"
		pullsStore.Update(context.Background(), pr)
	}

	// Filter open
	openList, openTotal, _ := pullsStore.List(context.Background(), repoID, "open", 10, 0)
	if len(openList) != 3 {
		t.Errorf("got %d open PRs, want 3", len(openList))
	}
	if openTotal != 3 {
		t.Errorf("got open total %d, want 3", openTotal)
	}

	// Filter closed
	closedList, closedTotal, _ := pullsStore.List(context.Background(), repoID, "closed", 10, 0)
	if len(closedList) != 2 {
		t.Errorf("got %d closed PRs, want 2", len(closedList))
	}
	if closedTotal != 2 {
		t.Errorf("got closed total %d, want 2", closedTotal)
	}

	// All PRs
	allList, allTotal, _ := pullsStore.List(context.Background(), repoID, "all", 10, 0)
	if len(allList) != 5 {
		t.Errorf("got %d all PRs, want 5", len(allList))
	}
	if allTotal != 5 {
		t.Errorf("got all total %d, want 5", allTotal)
	}
}

func TestPullsStore_List_Pagination(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	pullsStore := NewPullsStore(store.DB())

	for i := 1; i <= 10; i++ {
		createTestPullRequest(t, pullsStore, repoID, userID, i)
	}

	page1, total, _ := pullsStore.List(context.Background(), repoID, "", 3, 0)
	page2, _, _ := pullsStore.List(context.Background(), repoID, "", 3, 3)

	if len(page1) != 3 {
		t.Errorf("got %d PRs on page 1, want 3", len(page1))
	}
	if len(page2) != 3 {
		t.Errorf("got %d PRs on page 2, want 3", len(page2))
	}
	if total != 10 {
		t.Errorf("got total %d, want 10", total)
	}
}

// =============================================================================
// GetNextNumber Tests
// =============================================================================

func TestPullsStore_GetNextNumber_Empty(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	pullsStore := NewPullsStore(store.DB())

	next, err := pullsStore.GetNextNumber(context.Background(), repoID)
	if err != nil {
		t.Fatalf("GetNextNumber failed: %v", err)
	}
	if next != 1 {
		t.Errorf("got next number %d, want 1", next)
	}
}

func TestPullsStore_GetNextNumber_WithExisting(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	pullsStore := NewPullsStore(store.DB())

	for i := 1; i <= 3; i++ {
		createTestPullRequest(t, pullsStore, repoID, userID, i)
	}

	next, err := pullsStore.GetNextNumber(context.Background(), repoID)
	if err != nil {
		t.Fatalf("GetNextNumber failed: %v", err)
	}
	if next != 4 {
		t.Errorf("got next number %d, want 4", next)
	}
}

// =============================================================================
// Label Tests
// =============================================================================

func TestPullsStore_AddLabel(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	pullsStore := NewPullsStore(store.DB())

	pr := createTestPullRequest(t, pullsStore, repoID, userID, 1)

	label := &pulls.PRLabel{
		PRID:      pr.ID,
		LabelID:   "label-123",
		CreatedAt: time.Now(),
	}

	err := pullsStore.AddLabel(context.Background(), label)
	if err != nil {
		t.Fatalf("AddLabel failed: %v", err)
	}

	labels, _ := pullsStore.ListLabels(context.Background(), pr.ID)
	if len(labels) != 1 {
		t.Errorf("got %d labels, want 1", len(labels))
	}
	if labels[0] != "label-123" {
		t.Errorf("got label %q, want %q", labels[0], "label-123")
	}
}

func TestPullsStore_RemoveLabel(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	pullsStore := NewPullsStore(store.DB())

	pr := createTestPullRequest(t, pullsStore, repoID, userID, 1)

	label := &pulls.PRLabel{
		PRID:      pr.ID,
		LabelID:   "to-remove",
		CreatedAt: time.Now(),
	}
	pullsStore.AddLabel(context.Background(), label)

	err := pullsStore.RemoveLabel(context.Background(), pr.ID, "to-remove")
	if err != nil {
		t.Fatalf("RemoveLabel failed: %v", err)
	}

	labels, _ := pullsStore.ListLabels(context.Background(), pr.ID)
	if len(labels) != 0 {
		t.Errorf("expected no labels, got %d", len(labels))
	}
}

// =============================================================================
// Assignee Tests
// =============================================================================

func TestPullsStore_AddAssignee(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	usersStore := NewUsersStore(store.DB())
	pullsStore := NewPullsStore(store.DB())

	pr := createTestPullRequest(t, pullsStore, repoID, userID, 1)
	assignee := createTestUser(t, usersStore)

	prAssignee := &pulls.PRAssignee{
		PRID:      pr.ID,
		UserID:    assignee.ID,
		CreatedAt: time.Now(),
	}

	err := pullsStore.AddAssignee(context.Background(), prAssignee)
	if err != nil {
		t.Fatalf("AddAssignee failed: %v", err)
	}

	assignees, _ := pullsStore.ListAssignees(context.Background(), pr.ID)
	if len(assignees) != 1 {
		t.Errorf("got %d assignees, want 1", len(assignees))
	}
}

func TestPullsStore_RemoveAssignee(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	usersStore := NewUsersStore(store.DB())
	pullsStore := NewPullsStore(store.DB())

	pr := createTestPullRequest(t, pullsStore, repoID, userID, 1)
	assignee := createTestUser(t, usersStore)

	prAssignee := &pulls.PRAssignee{
		PRID:      pr.ID,
		UserID:    assignee.ID,
		CreatedAt: time.Now(),
	}
	pullsStore.AddAssignee(context.Background(), prAssignee)

	err := pullsStore.RemoveAssignee(context.Background(), pr.ID, assignee.ID)
	if err != nil {
		t.Fatalf("RemoveAssignee failed: %v", err)
	}

	assignees, _ := pullsStore.ListAssignees(context.Background(), pr.ID)
	if len(assignees) != 0 {
		t.Errorf("expected no assignees, got %d", len(assignees))
	}
}

// =============================================================================
// Reviewer Tests
// =============================================================================

func TestPullsStore_AddReviewer(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	usersStore := NewUsersStore(store.DB())
	pullsStore := NewPullsStore(store.DB())

	pr := createTestPullRequest(t, pullsStore, repoID, userID, 1)
	reviewer := createTestUser(t, usersStore)

	prReviewer := &pulls.PRReviewer{
		PRID:      pr.ID,
		UserID:    reviewer.ID,
		State:     "pending",
		CreatedAt: time.Now(),
	}

	err := pullsStore.AddReviewer(context.Background(), prReviewer)
	if err != nil {
		t.Fatalf("AddReviewer failed: %v", err)
	}

	reviewers, _ := pullsStore.ListReviewers(context.Background(), pr.ID)
	if len(reviewers) != 1 {
		t.Errorf("got %d reviewers, want 1", len(reviewers))
	}
	if reviewers[0].State != "pending" {
		t.Errorf("got state %q, want %q", reviewers[0].State, "pending")
	}
}

func TestPullsStore_RemoveReviewer(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	usersStore := NewUsersStore(store.DB())
	pullsStore := NewPullsStore(store.DB())

	pr := createTestPullRequest(t, pullsStore, repoID, userID, 1)
	reviewer := createTestUser(t, usersStore)

	prReviewer := &pulls.PRReviewer{
		PRID:      pr.ID,
		UserID:    reviewer.ID,
		State:     "pending",
		CreatedAt: time.Now(),
	}
	pullsStore.AddReviewer(context.Background(), prReviewer)

	err := pullsStore.RemoveReviewer(context.Background(), pr.ID, reviewer.ID)
	if err != nil {
		t.Fatalf("RemoveReviewer failed: %v", err)
	}

	reviewers, _ := pullsStore.ListReviewers(context.Background(), pr.ID)
	if len(reviewers) != 0 {
		t.Errorf("expected no reviewers, got %d", len(reviewers))
	}
}

// =============================================================================
// Review Tests
// =============================================================================

func TestPullsStore_CreateReview(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	pullsStore := NewPullsStore(store.DB())

	pr := createTestPullRequest(t, pullsStore, repoID, userID, 1)

	review := &pulls.Review{
		ID:        ulid.Make().String(),
		PRID:      pr.ID,
		UserID:    userID,
		Body:      "LGTM!",
		State:     "approved",
		CommitSHA: "abc123",
		CreatedAt: time.Now(),
	}

	err := pullsStore.CreateReview(context.Background(), review)
	if err != nil {
		t.Fatalf("CreateReview failed: %v", err)
	}

	got, _ := pullsStore.GetReview(context.Background(), review.ID)
	if got == nil {
		t.Fatal("expected review")
	}
	if got.State != "approved" {
		t.Errorf("got state %q, want %q", got.State, "approved")
	}
}

func TestPullsStore_UpdateReview(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	pullsStore := NewPullsStore(store.DB())

	pr := createTestPullRequest(t, pullsStore, repoID, userID, 1)

	review := &pulls.Review{
		ID:        ulid.Make().String(),
		PRID:      pr.ID,
		UserID:    userID,
		Body:      "Initial review",
		State:     "pending",
		CommitSHA: "abc123",
		CreatedAt: time.Now(),
	}
	pullsStore.CreateReview(context.Background(), review)

	submittedAt := time.Now()
	review.Body = "Updated review"
	review.State = "approved"
	review.SubmittedAt = &submittedAt

	err := pullsStore.UpdateReview(context.Background(), review)
	if err != nil {
		t.Fatalf("UpdateReview failed: %v", err)
	}

	got, _ := pullsStore.GetReview(context.Background(), review.ID)
	if got.Body != "Updated review" {
		t.Errorf("got body %q, want %q", got.Body, "Updated review")
	}
	if got.State != "approved" {
		t.Errorf("got state %q, want %q", got.State, "approved")
	}
	if got.SubmittedAt == nil {
		t.Error("expected submitted_at to be set")
	}
}

func TestPullsStore_ListReviews(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	pullsStore := NewPullsStore(store.DB())

	pr := createTestPullRequest(t, pullsStore, repoID, userID, 1)

	for i := 0; i < 3; i++ {
		review := &pulls.Review{
			ID:        ulid.Make().String(),
			PRID:      pr.ID,
			UserID:    userID,
			Body:      "Review " + string(rune('a'+i)),
			State:     "approved",
			CommitSHA: "abc123",
			CreatedAt: time.Now(),
		}
		pullsStore.CreateReview(context.Background(), review)
	}

	reviews, err := pullsStore.ListReviews(context.Background(), pr.ID)
	if err != nil {
		t.Fatalf("ListReviews failed: %v", err)
	}
	if len(reviews) != 3 {
		t.Errorf("got %d reviews, want 3", len(reviews))
	}
}

// =============================================================================
// Review Comment Tests
// =============================================================================

func TestPullsStore_CreateReviewComment(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	pullsStore := NewPullsStore(store.DB())

	pr := createTestPullRequest(t, pullsStore, repoID, userID, 1)

	review := &pulls.Review{
		ID:        ulid.Make().String(),
		PRID:      pr.ID,
		UserID:    userID,
		Body:      "Review",
		State:     "commented",
		CommitSHA: "abc123",
		CreatedAt: time.Now(),
	}
	pullsStore.CreateReview(context.Background(), review)

	comment := &pulls.ReviewComment{
		ID:        ulid.Make().String(),
		ReviewID:  review.ID,
		UserID:    userID,
		Path:      "src/main.go",
		Position:  10,
		Line:      42,
		Side:      "RIGHT",
		Body:      "This needs a fix",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := pullsStore.CreateReviewComment(context.Background(), comment)
	if err != nil {
		t.Fatalf("CreateReviewComment failed: %v", err)
	}

	got, _ := pullsStore.GetReviewComment(context.Background(), comment.ID)
	if got == nil {
		t.Fatal("expected review comment")
	}
	if got.Path != "src/main.go" {
		t.Errorf("got path %q, want %q", got.Path, "src/main.go")
	}
	if got.Line != 42 {
		t.Errorf("got line %d, want 42", got.Line)
	}
}

func TestPullsStore_UpdateReviewComment(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	pullsStore := NewPullsStore(store.DB())

	pr := createTestPullRequest(t, pullsStore, repoID, userID, 1)

	review := &pulls.Review{
		ID:        ulid.Make().String(),
		PRID:      pr.ID,
		UserID:    userID,
		Body:      "Review",
		State:     "commented",
		CommitSHA: "abc123",
		CreatedAt: time.Now(),
	}
	pullsStore.CreateReview(context.Background(), review)

	comment := &pulls.ReviewComment{
		ID:        ulid.Make().String(),
		ReviewID:  review.ID,
		UserID:    userID,
		Path:      "src/main.go",
		Line:      42,
		Body:      "Original comment",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	pullsStore.CreateReviewComment(context.Background(), comment)

	comment.Body = "Updated comment"
	err := pullsStore.UpdateReviewComment(context.Background(), comment)
	if err != nil {
		t.Fatalf("UpdateReviewComment failed: %v", err)
	}

	got, _ := pullsStore.GetReviewComment(context.Background(), comment.ID)
	if got.Body != "Updated comment" {
		t.Errorf("got body %q, want %q", got.Body, "Updated comment")
	}
}

func TestPullsStore_DeleteReviewComment(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	pullsStore := NewPullsStore(store.DB())

	pr := createTestPullRequest(t, pullsStore, repoID, userID, 1)

	review := &pulls.Review{
		ID:        ulid.Make().String(),
		PRID:      pr.ID,
		UserID:    userID,
		Body:      "Review",
		State:     "commented",
		CommitSHA: "abc123",
		CreatedAt: time.Now(),
	}
	pullsStore.CreateReview(context.Background(), review)

	comment := &pulls.ReviewComment{
		ID:        ulid.Make().String(),
		ReviewID:  review.ID,
		UserID:    userID,
		Path:      "src/main.go",
		Line:      42,
		Body:      "Comment to delete",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	pullsStore.CreateReviewComment(context.Background(), comment)

	err := pullsStore.DeleteReviewComment(context.Background(), comment.ID)
	if err != nil {
		t.Fatalf("DeleteReviewComment failed: %v", err)
	}

	got, _ := pullsStore.GetReviewComment(context.Background(), comment.ID)
	if got != nil {
		t.Error("expected review comment to be deleted")
	}
}

func TestPullsStore_ListReviewComments(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	pullsStore := NewPullsStore(store.DB())

	pr := createTestPullRequest(t, pullsStore, repoID, userID, 1)

	review := &pulls.Review{
		ID:        ulid.Make().String(),
		PRID:      pr.ID,
		UserID:    userID,
		Body:      "Review",
		State:     "commented",
		CommitSHA: "abc123",
		CreatedAt: time.Now(),
	}
	pullsStore.CreateReview(context.Background(), review)

	for i := 0; i < 3; i++ {
		comment := &pulls.ReviewComment{
			ID:        ulid.Make().String(),
			ReviewID:  review.ID,
			UserID:    userID,
			Path:      "src/main.go",
			Line:      10 + i,
			Body:      "Comment " + string(rune('a'+i)),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		pullsStore.CreateReviewComment(context.Background(), comment)
	}

	comments, err := pullsStore.ListReviewComments(context.Background(), pr.ID)
	if err != nil {
		t.Fatalf("ListReviewComments failed: %v", err)
	}
	if len(comments) != 3 {
		t.Errorf("got %d review comments, want 3", len(comments))
	}
}

// Verify interface compliance
var _ pulls.Store = (*PullsStore)(nil)
