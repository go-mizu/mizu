package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/kanban/feature/comments"
	"github.com/go-mizu/blueprints/kanban/feature/issues"
	"github.com/go-mizu/blueprints/kanban/feature/users"
	"github.com/oklog/ulid/v2"
)

func TestCommentsStore_Create(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())
	commentsStore := NewCommentsStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "comment@example.com",
		Username:     "commenter",
		DisplayName:  "Commenter",
		PasswordHash: "hashed",
	}
	usersStore.Create(context.Background(), u)

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)
	col := createTestColumn(t, columnsStore, p.ID, true)

	i := &issues.Issue{
		ID:        ulid.Make().String(),
		ProjectID: p.ID,
		Number:    1,
		Key:       "CMT-1",
		Title:     "Test Issue",
		ColumnID:  col.ID,
		Position:  0,
		CreatorID: u.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	issuesStore.Create(context.Background(), i)

	c := &comments.Comment{
		ID:        ulid.Make().String(),
		IssueID:   i.ID,
		AuthorID:  u.ID,
		Content:   "This is a comment",
		CreatedAt: time.Now(),
	}

	err := commentsStore.Create(context.Background(), c)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := commentsStore.GetByID(context.Background(), c.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected comment to be created")
	}
	if got.Content != c.Content {
		t.Errorf("got content %q, want %q", got.Content, c.Content)
	}
}

func TestCommentsStore_GetByID(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())
	commentsStore := NewCommentsStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "getcomment@example.com",
		Username:     "getcomment",
		DisplayName:  "Get Comment",
		PasswordHash: "hashed",
	}
	usersStore.Create(context.Background(), u)

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)
	col := createTestColumn(t, columnsStore, p.ID, true)

	i := &issues.Issue{
		ID:        ulid.Make().String(),
		ProjectID: p.ID,
		Number:    1,
		Key:       "GET-1",
		Title:     "Test Issue",
		ColumnID:  col.ID,
		Position:  0,
		CreatorID: u.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	issuesStore.Create(context.Background(), i)

	c := &comments.Comment{
		ID:        ulid.Make().String(),
		IssueID:   i.ID,
		AuthorID:  u.ID,
		Content:   "Test comment",
		CreatedAt: time.Now(),
	}
	commentsStore.Create(context.Background(), c)

	got, err := commentsStore.GetByID(context.Background(), c.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected comment")
	}
	if got.ID != c.ID {
		t.Errorf("got ID %q, want %q", got.ID, c.ID)
	}
}

func TestCommentsStore_GetByID_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	commentsStore := NewCommentsStore(store.DB())

	got, err := commentsStore.GetByID(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent comment")
	}
}

func TestCommentsStore_ListByIssue(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())
	commentsStore := NewCommentsStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "listcomment@example.com",
		Username:     "listcomment",
		DisplayName:  "List Comment",
		PasswordHash: "hashed",
	}
	usersStore.Create(context.Background(), u)

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)
	col := createTestColumn(t, columnsStore, p.ID, true)

	i := &issues.Issue{
		ID:        ulid.Make().String(),
		ProjectID: p.ID,
		Number:    1,
		Key:       "LST-1",
		Title:     "Test Issue",
		ColumnID:  col.ID,
		Position:  0,
		CreatorID: u.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	issuesStore.Create(context.Background(), i)

	c1 := &comments.Comment{
		ID:        ulid.Make().String(),
		IssueID:   i.ID,
		AuthorID:  u.ID,
		Content:   "Comment 1",
		CreatedAt: time.Now(),
	}
	c2 := &comments.Comment{
		ID:        ulid.Make().String(),
		IssueID:   i.ID,
		AuthorID:  u.ID,
		Content:   "Comment 2",
		CreatedAt: time.Now().Add(1 * time.Second),
	}
	commentsStore.Create(context.Background(), c1)
	commentsStore.Create(context.Background(), c2)

	list, err := commentsStore.ListByIssue(context.Background(), i.ID)
	if err != nil {
		t.Fatalf("ListByIssue failed: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("got %d comments, want 2", len(list))
	}
}

func TestCommentsStore_ListByIssue_Empty(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())
	commentsStore := NewCommentsStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "emptycomment@example.com",
		Username:     "emptycomment",
		DisplayName:  "Empty Comment",
		PasswordHash: "hashed",
	}
	usersStore.Create(context.Background(), u)

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)
	col := createTestColumn(t, columnsStore, p.ID, true)

	i := &issues.Issue{
		ID:        ulid.Make().String(),
		ProjectID: p.ID,
		Number:    1,
		Key:       "EMP-1",
		Title:     "Test Issue",
		ColumnID:  col.ID,
		Position:  0,
		CreatorID: u.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	issuesStore.Create(context.Background(), i)

	list, err := commentsStore.ListByIssue(context.Background(), i.ID)
	if err != nil {
		t.Fatalf("ListByIssue failed: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("got %d comments, want 0", len(list))
	}
}

func TestCommentsStore_Update(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())
	commentsStore := NewCommentsStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "updatecomment@example.com",
		Username:     "updatecomment",
		DisplayName:  "Update Comment",
		PasswordHash: "hashed",
	}
	usersStore.Create(context.Background(), u)

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)
	col := createTestColumn(t, columnsStore, p.ID, true)

	i := &issues.Issue{
		ID:        ulid.Make().String(),
		ProjectID: p.ID,
		Number:    1,
		Key:       "UPD-1",
		Title:     "Test Issue",
		ColumnID:  col.ID,
		Position:  0,
		CreatorID: u.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	issuesStore.Create(context.Background(), i)

	c := &comments.Comment{
		ID:        ulid.Make().String(),
		IssueID:   i.ID,
		AuthorID:  u.ID,
		Content:   "Original content",
		CreatedAt: time.Now(),
	}
	commentsStore.Create(context.Background(), c)

	newContent := "Updated content"
	err := commentsStore.Update(context.Background(), c.ID, newContent)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, _ := commentsStore.GetByID(context.Background(), c.ID)
	if got.Content != newContent {
		t.Errorf("got content %q, want %q", got.Content, newContent)
	}
	if got.EditedAt == nil {
		t.Error("expected edited_at to be set")
	}
}

func TestCommentsStore_Delete(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())
	commentsStore := NewCommentsStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "deletecomment@example.com",
		Username:     "deletecomment",
		DisplayName:  "Delete Comment",
		PasswordHash: "hashed",
	}
	usersStore.Create(context.Background(), u)

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)
	col := createTestColumn(t, columnsStore, p.ID, true)

	i := &issues.Issue{
		ID:        ulid.Make().String(),
		ProjectID: p.ID,
		Number:    1,
		Key:       "DEL-1",
		Title:     "Test Issue",
		ColumnID:  col.ID,
		Position:  0,
		CreatorID: u.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	issuesStore.Create(context.Background(), i)

	c := &comments.Comment{
		ID:        ulid.Make().String(),
		IssueID:   i.ID,
		AuthorID:  u.ID,
		Content:   "To be deleted",
		CreatedAt: time.Now(),
	}
	commentsStore.Create(context.Background(), c)

	err := commentsStore.Delete(context.Background(), c.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	got, _ := commentsStore.GetByID(context.Background(), c.ID)
	if got != nil {
		t.Error("expected comment to be deleted")
	}
}

func TestCommentsStore_CountByIssue(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())
	commentsStore := NewCommentsStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "countcomment@example.com",
		Username:     "countcomment",
		DisplayName:  "Count Comment",
		PasswordHash: "hashed",
	}
	usersStore.Create(context.Background(), u)

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)
	col := createTestColumn(t, columnsStore, p.ID, true)

	i := &issues.Issue{
		ID:        ulid.Make().String(),
		ProjectID: p.ID,
		Number:    1,
		Key:       "CNT-1",
		Title:     "Test Issue",
		ColumnID:  col.ID,
		Position:  0,
		CreatorID: u.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	issuesStore.Create(context.Background(), i)

	// Add 3 comments
	for n := 0; n < 3; n++ {
		c := &comments.Comment{
			ID:        ulid.Make().String(),
			IssueID:   i.ID,
			AuthorID:  u.ID,
			Content:   "Comment",
			CreatedAt: time.Now(),
		}
		commentsStore.Create(context.Background(), c)
	}

	count, err := commentsStore.CountByIssue(context.Background(), i.ID)
	if err != nil {
		t.Fatalf("CountByIssue failed: %v", err)
	}
	if count != 3 {
		t.Errorf("got count %d, want 3", count)
	}
}

func TestCommentsStore_CountByIssue_Zero(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())
	commentsStore := NewCommentsStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "zerocount@example.com",
		Username:     "zerocount",
		DisplayName:  "Zero Count",
		PasswordHash: "hashed",
	}
	usersStore.Create(context.Background(), u)

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)
	col := createTestColumn(t, columnsStore, p.ID, true)

	i := &issues.Issue{
		ID:        ulid.Make().String(),
		ProjectID: p.ID,
		Number:    1,
		Key:       "ZRO-1",
		Title:     "Test Issue",
		ColumnID:  col.ID,
		Position:  0,
		CreatorID: u.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	issuesStore.Create(context.Background(), i)

	count, err := commentsStore.CountByIssue(context.Background(), i.ID)
	if err != nil {
		t.Fatalf("CountByIssue failed: %v", err)
	}
	if count != 0 {
		t.Errorf("got count %d, want 0", count)
	}
}
