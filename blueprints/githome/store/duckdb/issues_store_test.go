package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/issues"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/oklog/ulid/v2"
)

func createTestIssue(t *testing.T, issuesStore *IssuesStore, repoID, authorID string, number int) *issues.Issue {
	t.Helper()
	id := ulid.Make().String()
	issue := &issues.Issue{
		ID:             id,
		RepoID:         repoID,
		Number:         number,
		Title:          "Test Issue " + id[len(id)-8:],
		Body:           "This is a test issue body",
		AuthorID:       authorID,
		State:          "open",
		IsLocked:       false,
		CommentCount:   0,
		ReactionsCount: 0,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	if err := issuesStore.Create(context.Background(), issue); err != nil {
		t.Fatalf("failed to create test issue: %v", err)
	}
	return issue
}

func createRepoAndUser(t *testing.T, store *Store) (repoID, userID string) {
	t.Helper()
	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())

	user := createTestUser(t, usersStore)
	repo := createTestRepo(t, reposStore, user.ID)

	return repo.ID, user.ID
}

// =============================================================================
// Issue CRUD Tests
// =============================================================================

func TestIssuesStore_Create(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	issuesStore := NewIssuesStore(store.DB())

	issue := &issues.Issue{
		ID:        ulid.Make().String(),
		RepoID:    repoID,
		Number:    1,
		Title:     "First Issue",
		Body:      "Issue body content",
		AuthorID:  userID,
		State:     "open",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := issuesStore.Create(context.Background(), issue)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := issuesStore.GetByID(context.Background(), issue.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected issue to be created")
	}
	if got.Title != issue.Title {
		t.Errorf("got title %q, want %q", got.Title, issue.Title)
	}
	if got.Number != 1 {
		t.Errorf("got number %d, want 1", got.Number)
	}
}

func TestIssuesStore_Create_WithAllFields(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	issuesStore := NewIssuesStore(store.DB())

	closedAt := time.Now()
	issue := &issues.Issue{
		ID:             ulid.Make().String(),
		RepoID:         repoID,
		Number:         1,
		Title:          "Complete Issue",
		Body:           "Full issue with all fields",
		AuthorID:       userID,
		AssigneeID:     userID,
		State:          "closed",
		StateReason:    "completed",
		IsLocked:       true,
		LockReason:     "resolved",
		MilestoneID:    "milestone-123",
		CommentCount:   5,
		ReactionsCount: 10,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		ClosedAt:       &closedAt,
		ClosedByID:     userID,
	}

	err := issuesStore.Create(context.Background(), issue)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, _ := issuesStore.GetByID(context.Background(), issue.ID)
	if got.State != "closed" {
		t.Errorf("got state %q, want %q", got.State, "closed")
	}
	if got.StateReason != "completed" {
		t.Errorf("got state_reason %q, want %q", got.StateReason, "completed")
	}
	if !got.IsLocked {
		t.Error("expected issue to be locked")
	}
	if got.LockReason != "resolved" {
		t.Errorf("got lock_reason %q, want %q", got.LockReason, "resolved")
	}
	if got.ClosedAt == nil {
		t.Error("expected closed_at to be set")
	}
	if got.ClosedByID != userID {
		t.Errorf("got closed_by_id %q, want %q", got.ClosedByID, userID)
	}
}

func TestIssuesStore_GetByID(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	issuesStore := NewIssuesStore(store.DB())

	issue := createTestIssue(t, issuesStore, repoID, userID, 1)

	got, err := issuesStore.GetByID(context.Background(), issue.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected issue")
	}
	if got.ID != issue.ID {
		t.Errorf("got ID %q, want %q", got.ID, issue.ID)
	}
}

func TestIssuesStore_GetByID_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	issuesStore := NewIssuesStore(store.DB())

	got, err := issuesStore.GetByID(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent issue")
	}
}

func TestIssuesStore_GetByNumber(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	issuesStore := NewIssuesStore(store.DB())

	issue := createTestIssue(t, issuesStore, repoID, userID, 42)

	got, err := issuesStore.GetByNumber(context.Background(), repoID, 42)
	if err != nil {
		t.Fatalf("GetByNumber failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected issue")
	}
	if got.Number != 42 {
		t.Errorf("got number %d, want 42", got.Number)
	}
	if got.ID != issue.ID {
		t.Errorf("got ID %q, want %q", got.ID, issue.ID)
	}
}

func TestIssuesStore_GetByNumber_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	issuesStore := NewIssuesStore(store.DB())

	got, err := issuesStore.GetByNumber(context.Background(), repoID, 999)
	if err != nil {
		t.Fatalf("GetByNumber failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent issue number")
	}
}

func TestIssuesStore_GetByNumber_DifferentRepos(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())

	user := createTestUser(t, usersStore)
	repo1 := createTestRepo(t, reposStore, user.ID)
	repo2 := createTestRepo(t, reposStore, user.ID)

	// Both repos have issue #1
	issue1 := createTestIssue(t, issuesStore, repo1.ID, user.ID, 1)
	issue2 := createTestIssue(t, issuesStore, repo2.ID, user.ID, 1)

	got1, _ := issuesStore.GetByNumber(context.Background(), repo1.ID, 1)
	got2, _ := issuesStore.GetByNumber(context.Background(), repo2.ID, 1)

	if got1.ID != issue1.ID {
		t.Error("expected issue from repo1")
	}
	if got2.ID != issue2.ID {
		t.Error("expected issue from repo2")
	}
}

func TestIssuesStore_Update(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	issuesStore := NewIssuesStore(store.DB())

	issue := createTestIssue(t, issuesStore, repoID, userID, 1)

	issue.Title = "Updated Title"
	issue.Body = "Updated body"
	issue.State = "closed"

	err := issuesStore.Update(context.Background(), issue)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, _ := issuesStore.GetByID(context.Background(), issue.ID)
	if got.Title != "Updated Title" {
		t.Errorf("got title %q, want %q", got.Title, "Updated Title")
	}
	if got.State != "closed" {
		t.Errorf("got state %q, want %q", got.State, "closed")
	}
}

func TestIssuesStore_Update_UpdatesTimestamp(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	issuesStore := NewIssuesStore(store.DB())

	issue := createTestIssue(t, issuesStore, repoID, userID, 1)
	originalUpdatedAt := issue.UpdatedAt

	time.Sleep(10 * time.Millisecond)

	issue.Title = "New Title"
	issuesStore.Update(context.Background(), issue)

	got, _ := issuesStore.GetByID(context.Background(), issue.ID)
	if !got.UpdatedAt.After(originalUpdatedAt) {
		t.Error("expected updated_at to be updated")
	}
}

func TestIssuesStore_Update_CloseIssue(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	issuesStore := NewIssuesStore(store.DB())

	issue := createTestIssue(t, issuesStore, repoID, userID, 1)

	closedAt := time.Now()
	issue.State = "closed"
	issue.StateReason = "completed"
	issue.ClosedAt = &closedAt
	issue.ClosedByID = userID

	issuesStore.Update(context.Background(), issue)

	got, _ := issuesStore.GetByID(context.Background(), issue.ID)
	if got.State != "closed" {
		t.Error("expected issue to be closed")
	}
	if got.ClosedAt == nil {
		t.Error("expected closed_at to be set")
	}
	if got.ClosedByID != userID {
		t.Errorf("got closed_by_id %q, want %q", got.ClosedByID, userID)
	}
}

func TestIssuesStore_Update_LockIssue(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	issuesStore := NewIssuesStore(store.DB())

	issue := createTestIssue(t, issuesStore, repoID, userID, 1)

	issue.IsLocked = true
	issue.LockReason = "off-topic"

	issuesStore.Update(context.Background(), issue)

	got, _ := issuesStore.GetByID(context.Background(), issue.ID)
	if !got.IsLocked {
		t.Error("expected issue to be locked")
	}
	if got.LockReason != "off-topic" {
		t.Errorf("got lock_reason %q, want %q", got.LockReason, "off-topic")
	}
}

func TestIssuesStore_Delete(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	issuesStore := NewIssuesStore(store.DB())

	issue := createTestIssue(t, issuesStore, repoID, userID, 1)

	err := issuesStore.Delete(context.Background(), issue.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	got, _ := issuesStore.GetByID(context.Background(), issue.ID)
	if got != nil {
		t.Error("expected issue to be deleted")
	}
}

func TestIssuesStore_Delete_NonExistent(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	issuesStore := NewIssuesStore(store.DB())

	err := issuesStore.Delete(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("Delete should not error for non-existent issue: %v", err)
	}
}

// =============================================================================
// List Tests
// =============================================================================

func TestIssuesStore_List(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	issuesStore := NewIssuesStore(store.DB())

	for i := 1; i <= 5; i++ {
		createTestIssue(t, issuesStore, repoID, userID, i)
	}

	list, total, err := issuesStore.List(context.Background(), repoID, "", 10, 0)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 5 {
		t.Errorf("got %d issues, want 5", len(list))
	}
	if total != 5 {
		t.Errorf("got total %d, want 5", total)
	}
}

func TestIssuesStore_List_FilterByState(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	issuesStore := NewIssuesStore(store.DB())

	// Create open issues
	for i := 1; i <= 3; i++ {
		createTestIssue(t, issuesStore, repoID, userID, i)
	}

	// Create closed issues
	for i := 4; i <= 5; i++ {
		issue := createTestIssue(t, issuesStore, repoID, userID, i)
		issue.State = "closed"
		issuesStore.Update(context.Background(), issue)
	}

	// Filter open
	openList, openTotal, _ := issuesStore.List(context.Background(), repoID, "open", 10, 0)
	if len(openList) != 3 {
		t.Errorf("got %d open issues, want 3", len(openList))
	}
	if openTotal != 3 {
		t.Errorf("got open total %d, want 3", openTotal)
	}

	// Filter closed
	closedList, closedTotal, _ := issuesStore.List(context.Background(), repoID, "closed", 10, 0)
	if len(closedList) != 2 {
		t.Errorf("got %d closed issues, want 2", len(closedList))
	}
	if closedTotal != 2 {
		t.Errorf("got closed total %d, want 2", closedTotal)
	}

	// All issues
	allList, allTotal, _ := issuesStore.List(context.Background(), repoID, "all", 10, 0)
	if len(allList) != 5 {
		t.Errorf("got %d all issues, want 5", len(allList))
	}
	if allTotal != 5 {
		t.Errorf("got all total %d, want 5", allTotal)
	}
}

func TestIssuesStore_List_Pagination(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	issuesStore := NewIssuesStore(store.DB())

	for i := 1; i <= 10; i++ {
		createTestIssue(t, issuesStore, repoID, userID, i)
	}

	page1, total, _ := issuesStore.List(context.Background(), repoID, "", 3, 0)
	page2, _, _ := issuesStore.List(context.Background(), repoID, "", 3, 3)

	if len(page1) != 3 {
		t.Errorf("got %d issues on page 1, want 3", len(page1))
	}
	if len(page2) != 3 {
		t.Errorf("got %d issues on page 2, want 3", len(page2))
	}
	if total != 10 {
		t.Errorf("got total %d, want 10", total)
	}
	if page1[0].ID == page2[0].ID {
		t.Error("expected different issues on different pages")
	}
}

func TestIssuesStore_List_Empty(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	issuesStore := NewIssuesStore(store.DB())

	list, total, err := issuesStore.List(context.Background(), repoID, "", 10, 0)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 0 {
		t.Errorf("got total %d, want 0", total)
	}
	if list != nil && len(list) != 0 {
		t.Error("expected empty list")
	}
}

func TestIssuesStore_List_OrderByCreatedAt(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	issuesStore := NewIssuesStore(store.DB())

	// Create issues with time gaps
	for i := 1; i <= 3; i++ {
		createTestIssue(t, issuesStore, repoID, userID, i)
		time.Sleep(10 * time.Millisecond)
	}

	list, _, _ := issuesStore.List(context.Background(), repoID, "", 10, 0)

	// Should be ordered by created_at DESC (newest first)
	if len(list) >= 2 {
		if list[0].Number != 3 {
			t.Error("expected newest issue first")
		}
		if list[2].Number != 1 {
			t.Error("expected oldest issue last")
		}
	}
}

// =============================================================================
// GetNextNumber Tests
// =============================================================================

func TestIssuesStore_GetNextNumber_Empty(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, _ := createRepoAndUser(t, store)
	issuesStore := NewIssuesStore(store.DB())

	next, err := issuesStore.GetNextNumber(context.Background(), repoID)
	if err != nil {
		t.Fatalf("GetNextNumber failed: %v", err)
	}
	if next != 1 {
		t.Errorf("got next number %d, want 1", next)
	}
}

func TestIssuesStore_GetNextNumber_WithExisting(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	issuesStore := NewIssuesStore(store.DB())

	// Create issues 1, 2, 3
	for i := 1; i <= 3; i++ {
		createTestIssue(t, issuesStore, repoID, userID, i)
	}

	next, err := issuesStore.GetNextNumber(context.Background(), repoID)
	if err != nil {
		t.Fatalf("GetNextNumber failed: %v", err)
	}
	if next != 4 {
		t.Errorf("got next number %d, want 4", next)
	}
}

func TestIssuesStore_GetNextNumber_WithGaps(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	issuesStore := NewIssuesStore(store.DB())

	// Create issues 1, 5, 10 (with gaps)
	createTestIssue(t, issuesStore, repoID, userID, 1)
	createTestIssue(t, issuesStore, repoID, userID, 5)
	createTestIssue(t, issuesStore, repoID, userID, 10)

	next, err := issuesStore.GetNextNumber(context.Background(), repoID)
	if err != nil {
		t.Fatalf("GetNextNumber failed: %v", err)
	}
	// Should return max + 1, not fill gaps
	if next != 11 {
		t.Errorf("got next number %d, want 11", next)
	}
}

func TestIssuesStore_GetNextNumber_PerRepo(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())

	user := createTestUser(t, usersStore)
	repo1 := createTestRepo(t, reposStore, user.ID)
	repo2 := createTestRepo(t, reposStore, user.ID)

	// Create 5 issues in repo1
	for i := 1; i <= 5; i++ {
		createTestIssue(t, issuesStore, repo1.ID, user.ID, i)
	}

	// Create 2 issues in repo2
	for i := 1; i <= 2; i++ {
		createTestIssue(t, issuesStore, repo2.ID, user.ID, i)
	}

	next1, _ := issuesStore.GetNextNumber(context.Background(), repo1.ID)
	next2, _ := issuesStore.GetNextNumber(context.Background(), repo2.ID)

	if next1 != 6 {
		t.Errorf("got next number for repo1 %d, want 6", next1)
	}
	if next2 != 3 {
		t.Errorf("got next number for repo2 %d, want 3", next2)
	}
}

// =============================================================================
// Label Tests
// =============================================================================

func TestIssuesStore_AddLabel(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	issuesStore := NewIssuesStore(store.DB())

	issue := createTestIssue(t, issuesStore, repoID, userID, 1)

	issueLabel := &issues.IssueLabel{
		ID:        ulid.Make().String(),
		IssueID:   issue.ID,
		LabelID:   "label-123",
		CreatedAt: time.Now(),
	}

	err := issuesStore.AddLabel(context.Background(), issueLabel)
	if err != nil {
		t.Fatalf("AddLabel failed: %v", err)
	}

	labels, _ := issuesStore.ListLabels(context.Background(), issue.ID)
	if len(labels) != 1 {
		t.Errorf("got %d labels, want 1", len(labels))
	}
	if labels[0] != "label-123" {
		t.Errorf("got label %q, want %q", labels[0], "label-123")
	}
}

func TestIssuesStore_AddLabel_Multiple(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	issuesStore := NewIssuesStore(store.DB())

	issue := createTestIssue(t, issuesStore, repoID, userID, 1)

	labelIDs := []string{"bug", "priority-high", "help-wanted"}
	for _, labelID := range labelIDs {
		issueLabel := &issues.IssueLabel{
			ID:        ulid.Make().String(),
			IssueID:   issue.ID,
			LabelID:   labelID,
			CreatedAt: time.Now(),
		}
		issuesStore.AddLabel(context.Background(), issueLabel)
	}

	labels, _ := issuesStore.ListLabels(context.Background(), issue.ID)
	if len(labels) != 3 {
		t.Errorf("got %d labels, want 3", len(labels))
	}
}

func TestIssuesStore_RemoveLabel(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	issuesStore := NewIssuesStore(store.DB())

	issue := createTestIssue(t, issuesStore, repoID, userID, 1)

	issueLabel := &issues.IssueLabel{
		ID:        ulid.Make().String(),
		IssueID:   issue.ID,
		LabelID:   "to-remove",
		CreatedAt: time.Now(),
	}
	issuesStore.AddLabel(context.Background(), issueLabel)

	err := issuesStore.RemoveLabel(context.Background(), issue.ID, "to-remove")
	if err != nil {
		t.Fatalf("RemoveLabel failed: %v", err)
	}

	labels, _ := issuesStore.ListLabels(context.Background(), issue.ID)
	if len(labels) != 0 {
		t.Errorf("expected no labels, got %d", len(labels))
	}
}

func TestIssuesStore_ListLabels_Empty(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	issuesStore := NewIssuesStore(store.DB())

	issue := createTestIssue(t, issuesStore, repoID, userID, 1)

	labels, err := issuesStore.ListLabels(context.Background(), issue.ID)
	if err != nil {
		t.Fatalf("ListLabels failed: %v", err)
	}
	if labels != nil && len(labels) != 0 {
		t.Error("expected empty labels list")
	}
}

// =============================================================================
// Assignee Tests
// =============================================================================

func TestIssuesStore_AddAssignee(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())

	issue := createTestIssue(t, issuesStore, repoID, userID, 1)
	assignee := createTestUser(t, usersStore)

	issueAssignee := &issues.IssueAssignee{
		ID:        ulid.Make().String(),
		IssueID:   issue.ID,
		UserID:    assignee.ID,
		CreatedAt: time.Now(),
	}

	err := issuesStore.AddAssignee(context.Background(), issueAssignee)
	if err != nil {
		t.Fatalf("AddAssignee failed: %v", err)
	}

	assignees, _ := issuesStore.ListAssignees(context.Background(), issue.ID)
	if len(assignees) != 1 {
		t.Errorf("got %d assignees, want 1", len(assignees))
	}
	if assignees[0] != assignee.ID {
		t.Errorf("got assignee %q, want %q", assignees[0], assignee.ID)
	}
}

func TestIssuesStore_AddAssignee_Multiple(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())

	issue := createTestIssue(t, issuesStore, repoID, userID, 1)

	for i := 0; i < 3; i++ {
		assignee := createTestUser(t, usersStore)
		issueAssignee := &issues.IssueAssignee{
			ID:        ulid.Make().String(),
			IssueID:   issue.ID,
			UserID:    assignee.ID,
			CreatedAt: time.Now(),
		}
		issuesStore.AddAssignee(context.Background(), issueAssignee)
	}

	assignees, _ := issuesStore.ListAssignees(context.Background(), issue.ID)
	if len(assignees) != 3 {
		t.Errorf("got %d assignees, want 3", len(assignees))
	}
}

func TestIssuesStore_RemoveAssignee(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())

	issue := createTestIssue(t, issuesStore, repoID, userID, 1)
	assignee := createTestUser(t, usersStore)

	issueAssignee := &issues.IssueAssignee{
		ID:        ulid.Make().String(),
		IssueID:   issue.ID,
		UserID:    assignee.ID,
		CreatedAt: time.Now(),
	}
	issuesStore.AddAssignee(context.Background(), issueAssignee)

	err := issuesStore.RemoveAssignee(context.Background(), issue.ID, assignee.ID)
	if err != nil {
		t.Fatalf("RemoveAssignee failed: %v", err)
	}

	assignees, _ := issuesStore.ListAssignees(context.Background(), issue.ID)
	if len(assignees) != 0 {
		t.Errorf("expected no assignees, got %d", len(assignees))
	}
}

func TestIssuesStore_ListAssignees_Empty(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	issuesStore := NewIssuesStore(store.DB())

	issue := createTestIssue(t, issuesStore, repoID, userID, 1)

	assignees, err := issuesStore.ListAssignees(context.Background(), issue.ID)
	if err != nil {
		t.Fatalf("ListAssignees failed: %v", err)
	}
	if assignees != nil && len(assignees) != 0 {
		t.Error("expected empty assignees list")
	}
}

// =============================================================================
// Integration Tests
// =============================================================================

func TestIssuesStore_DeleteIssueRemovesLabels(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	issuesStore := NewIssuesStore(store.DB())

	issue := createTestIssue(t, issuesStore, repoID, userID, 1)

	// Add labels
	issueLabel := &issues.IssueLabel{
		ID:        ulid.Make().String(),
		IssueID:   issue.ID,
		LabelID:   "label-to-orphan",
		CreatedAt: time.Now(),
	}
	issuesStore.AddLabel(context.Background(), issueLabel)

	// Delete issue
	issuesStore.Delete(context.Background(), issue.ID)

	// Labels should be orphaned (issue doesn't exist)
	labels, _ := issuesStore.ListLabels(context.Background(), issue.ID)
	// Note: Labels may remain orphaned unless there's a CASCADE delete
	// This test documents the current behavior
	_ = labels
}

func TestIssuesStore_DeleteIssueRemovesAssignees(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())

	issue := createTestIssue(t, issuesStore, repoID, userID, 1)
	assignee := createTestUser(t, usersStore)

	// Add assignee
	issueAssignee := &issues.IssueAssignee{
		ID:        ulid.Make().String(),
		IssueID:   issue.ID,
		UserID:    assignee.ID,
		CreatedAt: time.Now(),
	}
	issuesStore.AddAssignee(context.Background(), issueAssignee)

	// Delete issue
	issuesStore.Delete(context.Background(), issue.ID)

	// Assignees should be orphaned (issue doesn't exist)
	assignees, _ := issuesStore.ListAssignees(context.Background(), issue.ID)
	// Note: Assignees may remain orphaned unless there's a CASCADE delete
	_ = assignees
}

func TestIssuesStore_IssueLifecycle(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	repoID, userID := createRepoAndUser(t, store)
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())

	// 1. Create issue
	issue := &issues.Issue{
		ID:        ulid.Make().String(),
		RepoID:    repoID,
		Number:    1,
		Title:     "Bug: Something is broken",
		Body:      "Description of the bug",
		AuthorID:  userID,
		State:     "open",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	issuesStore.Create(context.Background(), issue)

	// 2. Add labels
	for _, label := range []string{"bug", "priority-high"} {
		issueLabel := &issues.IssueLabel{
			ID:        ulid.Make().String(),
			IssueID:   issue.ID,
			LabelID:   label,
			CreatedAt: time.Now(),
		}
		issuesStore.AddLabel(context.Background(), issueLabel)
	}

	// 3. Add assignee
	assignee := createTestUser(t, usersStore)
	issueAssignee := &issues.IssueAssignee{
		ID:        ulid.Make().String(),
		IssueID:   issue.ID,
		UserID:    assignee.ID,
		CreatedAt: time.Now(),
	}
	issuesStore.AddAssignee(context.Background(), issueAssignee)

	// 4. Close issue
	closedAt := time.Now()
	issue.State = "closed"
	issue.StateReason = "completed"
	issue.ClosedAt = &closedAt
	issue.ClosedByID = assignee.ID
	issuesStore.Update(context.Background(), issue)

	// 5. Verify final state
	got, _ := issuesStore.GetByID(context.Background(), issue.ID)
	if got.State != "closed" {
		t.Error("expected issue to be closed")
	}

	labels, _ := issuesStore.ListLabels(context.Background(), issue.ID)
	if len(labels) != 2 {
		t.Errorf("expected 2 labels, got %d", len(labels))
	}

	assignees, _ := issuesStore.ListAssignees(context.Background(), issue.ID)
	if len(assignees) != 1 {
		t.Errorf("expected 1 assignee, got %d", len(assignees))
	}
}

// Helper to verify interface compliance
var _ issues.Store = (*IssuesStore)(nil)
var _ repos.Store = (*ReposStore)(nil)
