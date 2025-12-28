package duckdb

import (
	"context"
	"testing"

	"github.com/go-mizu/blueprints/githome/feature/issues"
	"github.com/go-mizu/blueprints/githome/feature/repos"
)

func createTestIssue(t *testing.T, store *IssuesStore, repoID, creatorID int64, number int) *issues.Issue {
	t.Helper()
	i := &issues.Issue{
		RepoID:    repoID,
		Number:    number,
		State:     "open",
		Title:     "Test Issue",
		Body:      "Test body",
		CreatorID: creatorID,
	}
	if err := store.Create(context.Background(), i); err != nil {
		t.Fatalf("failed to create test issue: %v", err)
	}
	return i
}

func TestIssuesStore_Create(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())

	u := createTestUser(t, usersStore, "issueowner")
	r := createTestRepo(t, reposStore, u.ID, "issuerepo")

	i := &issues.Issue{
		RepoID:    r.ID,
		Number:    1,
		State:     "open",
		Title:     "Test Issue",
		Body:      "Test body",
		CreatorID: u.ID,
	}

	err := issuesStore.Create(context.Background(), i)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if i.ID == 0 {
		t.Error("expected ID to be set")
	}

	got, err := issuesStore.GetByID(context.Background(), i.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected issue to be created")
	}
	if got.Title != i.Title {
		t.Errorf("got title %q, want %q", got.Title, i.Title)
	}
}

func TestIssuesStore_GetByNumber(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())

	u := createTestUser(t, usersStore, "getnumowner")
	r := createTestRepo(t, reposStore, u.ID, "getnumrepo")
	i := createTestIssue(t, issuesStore, r.ID, u.ID, 42)

	got, err := issuesStore.GetByNumber(context.Background(), r.ID, 42)
	if err != nil {
		t.Fatalf("GetByNumber failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected issue")
	}
	if got.Number != i.Number {
		t.Errorf("got number %d, want %d", got.Number, i.Number)
	}
}

func TestIssuesStore_Update(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())

	u := createTestUser(t, usersStore, "updateissueowner")
	r := createTestRepo(t, reposStore, u.ID, "updateissuerepo")
	i := createTestIssue(t, issuesStore, r.ID, u.ID, 1)

	newTitle := "Updated Title"
	err := issuesStore.Update(context.Background(), i.ID, &issues.UpdateIn{
		Title: &newTitle,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, _ := issuesStore.GetByID(context.Background(), i.ID)
	if got.Title != newTitle {
		t.Errorf("got title %q, want %q", got.Title, newTitle)
	}
}

func TestIssuesStore_Delete(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())

	u := createTestUser(t, usersStore, "delissueowner")
	r := createTestRepo(t, reposStore, u.ID, "delissuerepo")
	i := createTestIssue(t, issuesStore, r.ID, u.ID, 1)

	err := issuesStore.Delete(context.Background(), i.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	got, _ := issuesStore.GetByID(context.Background(), i.ID)
	if got != nil {
		t.Error("expected issue to be deleted")
	}
}

func TestIssuesStore_List(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())

	u := createTestUser(t, usersStore, "listissuesowner")
	r := createTestRepo(t, reposStore, u.ID, "listissuesrepo")
	createTestIssue(t, issuesStore, r.ID, u.ID, 1)
	createTestIssue(t, issuesStore, r.ID, u.ID, 2)
	createTestIssue(t, issuesStore, r.ID, u.ID, 3)

	list, err := issuesStore.List(context.Background(), r.ID, nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("got %d issues, want 3", len(list))
	}
}

func TestIssuesStore_NextNumber(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())

	u := createTestUser(t, usersStore, "nextnumowner")
	r := createTestRepo(t, reposStore, u.ID, "nextnumrepo")

	// First issue should be 1
	num, err := issuesStore.NextNumber(context.Background(), r.ID)
	if err != nil {
		t.Fatalf("NextNumber failed: %v", err)
	}
	if num != 1 {
		t.Errorf("got %d, want 1", num)
	}

	createTestIssue(t, issuesStore, r.ID, u.ID, 1)
	createTestIssue(t, issuesStore, r.ID, u.ID, 2)

	num, err = issuesStore.NextNumber(context.Background(), r.ID)
	if err != nil {
		t.Fatalf("NextNumber failed: %v", err)
	}
	if num != 3 {
		t.Errorf("got %d, want 3", num)
	}
}

func TestIssuesStore_SetLocked(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())

	u := createTestUser(t, usersStore, "lockowner")
	r := createTestRepo(t, reposStore, u.ID, "lockrepo")
	i := createTestIssue(t, issuesStore, r.ID, u.ID, 1)

	err := issuesStore.SetLocked(context.Background(), i.ID, true, "off-topic")
	if err != nil {
		t.Fatalf("SetLocked failed: %v", err)
	}

	got, _ := issuesStore.GetByID(context.Background(), i.ID)
	if !got.Locked {
		t.Error("expected issue to be locked")
	}
	if got.ActiveLockReason != "off-topic" {
		t.Errorf("got lock reason %q, want 'off-topic'", got.ActiveLockReason)
	}
}

func TestIssuesStore_Assignees(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())

	u1 := createTestUser(t, usersStore, "assignowner")
	u2 := createTestUser(t, usersStore, "assignee1")
	u3 := createTestUser(t, usersStore, "assignee2")
	r := createTestRepo(t, reposStore, u1.ID, "assignrepo")
	i := createTestIssue(t, issuesStore, r.ID, u1.ID, 1)

	// Add assignees
	issuesStore.AddAssignee(context.Background(), i.ID, u2.ID)
	issuesStore.AddAssignee(context.Background(), i.ID, u3.ID)

	list, err := issuesStore.ListAssignees(context.Background(), i.ID)
	if err != nil {
		t.Fatalf("ListAssignees failed: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("got %d assignees, want 2", len(list))
	}

	// Remove one
	issuesStore.RemoveAssignee(context.Background(), i.ID, u2.ID)
	list, _ = issuesStore.ListAssignees(context.Background(), i.ID)
	if len(list) != 1 {
		t.Errorf("got %d assignees, want 1", len(list))
	}
}

func TestIssuesStore_IncrementComments(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	usersStore := NewUsersStore(store.DB())
	reposStore := NewReposStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())

	u := createTestUser(t, usersStore, "commentowner")
	r := createTestRepo(t, reposStore, u.ID, "commentrepo")
	i := createTestIssue(t, issuesStore, r.ID, u.ID, 1)

	err := issuesStore.IncrementComments(context.Background(), i.ID, 5)
	if err != nil {
		t.Fatalf("IncrementComments failed: %v", err)
	}

	got, _ := issuesStore.GetByID(context.Background(), i.ID)
	if got.Comments != 5 {
		t.Errorf("got comments %d, want 5", got.Comments)
	}
}

// ensure repos.Repository is available
var _ = (*repos.Repository)(nil)
