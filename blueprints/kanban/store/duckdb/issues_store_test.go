package duckdb

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/kanban/feature/issues"
	"github.com/go-mizu/blueprints/kanban/feature/users"
	"github.com/oklog/ulid/v2"
)

func createTestIssue(t *testing.T, store *IssuesStore, projectID, columnID, creatorID string, number int) *issues.Issue {
	t.Helper()
	i := &issues.Issue{
		ID:        ulid.Make().String(),
		ProjectID: projectID,
		Number:    number,
		Key:       fmt.Sprintf("TEST-%d", number),
		Title:     fmt.Sprintf("Test Issue %d", number),
		ColumnID:  columnID,
		Position:  0,
		CreatorID: creatorID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := store.Create(context.Background(), i); err != nil {
		t.Fatalf("failed to create test issue: %v", err)
	}
	return i
}

func TestIssuesStore_Create(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "issuetest@example.com",
		Username:     "issuetest",
		DisplayName:  "Issue Test",
		PasswordHash: "hashed",
	}
	usersStore.Create(context.Background(), u)

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)
	c := createTestColumn(t, columnsStore, p.ID, true)

	i := &issues.Issue{
		ID:        ulid.Make().String(),
		ProjectID: p.ID,
		Number:    1,
		Key:       "TEST-1",
		Title:     "Test Issue",
		ColumnID:  c.ID,
		Position:  0,
		CreatorID: u.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := issuesStore.Create(context.Background(), i)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := issuesStore.GetByID(context.Background(), i.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected issue to be created")
	}
	if got.Key != i.Key {
		t.Errorf("got key %q, want %q", got.Key, i.Key)
	}
}

func TestIssuesStore_GetByID(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "getbyid@example.com",
		Username:     "getbyid",
		DisplayName:  "Get By ID",
		PasswordHash: "hashed",
	}
	usersStore.Create(context.Background(), u)

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)
	c := createTestColumn(t, columnsStore, p.ID, true)
	i := createTestIssue(t, issuesStore, p.ID, c.ID, u.ID, 1)

	got, err := issuesStore.GetByID(context.Background(), i.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected issue")
	}
	if got.ID != i.ID {
		t.Errorf("got ID %q, want %q", got.ID, i.ID)
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

func TestIssuesStore_GetByKey(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "getbykey@example.com",
		Username:     "getbykey",
		DisplayName:  "Get By Key",
		PasswordHash: "hashed",
	}
	usersStore.Create(context.Background(), u)

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)
	c := createTestColumn(t, columnsStore, p.ID, true)
	i := createTestIssue(t, issuesStore, p.ID, c.ID, u.ID, 1)

	got, err := issuesStore.GetByKey(context.Background(), i.Key)
	if err != nil {
		t.Fatalf("GetByKey failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected issue")
	}
	if got.Key != i.Key {
		t.Errorf("got key %q, want %q", got.Key, i.Key)
	}
}

func TestIssuesStore_GetByKey_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	issuesStore := NewIssuesStore(store.DB())

	got, err := issuesStore.GetByKey(context.Background(), "NONEXIST-999")
	if err != nil {
		t.Fatalf("GetByKey failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent key")
	}
}

func TestIssuesStore_ListByProject(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "listproj@example.com",
		Username:     "listproj",
		DisplayName:  "List By Project",
		PasswordHash: "hashed",
	}
	usersStore.Create(context.Background(), u)

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)
	c := createTestColumn(t, columnsStore, p.ID, true)
	createTestIssue(t, issuesStore, p.ID, c.ID, u.ID, 1)
	createTestIssue(t, issuesStore, p.ID, c.ID, u.ID, 2)

	list, err := issuesStore.ListByProject(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("ListByProject failed: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("got %d issues, want 2", len(list))
	}
}

func TestIssuesStore_ListByColumn(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "listcol@example.com",
		Username:     "listcol",
		DisplayName:  "List By Column",
		PasswordHash: "hashed",
	}
	usersStore.Create(context.Background(), u)

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)
	c1 := createTestColumn(t, columnsStore, p.ID, true)
	c2 := createTestColumn(t, columnsStore, p.ID, false)

	createTestIssue(t, issuesStore, p.ID, c1.ID, u.ID, 1)
	createTestIssue(t, issuesStore, p.ID, c2.ID, u.ID, 2)

	list, err := issuesStore.ListByColumn(context.Background(), c1.ID)
	if err != nil {
		t.Fatalf("ListByColumn failed: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("got %d issues, want 1", len(list))
	}
}

func TestIssuesStore_Update(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "updateissue@example.com",
		Username:     "updateissue",
		DisplayName:  "Update Issue",
		PasswordHash: "hashed",
	}
	usersStore.Create(context.Background(), u)

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)
	c := createTestColumn(t, columnsStore, p.ID, true)
	i := createTestIssue(t, issuesStore, p.ID, c.ID, u.ID, 1)

	newTitle := "Updated Issue Title"
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

func TestIssuesStore_Move(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "moveissue@example.com",
		Username:     "moveissue",
		DisplayName:  "Move Issue",
		PasswordHash: "hashed",
	}
	usersStore.Create(context.Background(), u)

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)
	c1 := createTestColumn(t, columnsStore, p.ID, true)
	c2 := createTestColumn(t, columnsStore, p.ID, false)

	i := createTestIssue(t, issuesStore, p.ID, c1.ID, u.ID, 1)

	err := issuesStore.Move(context.Background(), i.ID, c2.ID, 5)
	if err != nil {
		t.Fatalf("Move failed: %v", err)
	}

	got, _ := issuesStore.GetByID(context.Background(), i.ID)
	if got.ColumnID != c2.ID {
		t.Errorf("got column ID %q, want %q", got.ColumnID, c2.ID)
	}
	if got.Position != 5 {
		t.Errorf("got position %d, want 5", got.Position)
	}
}

func TestIssuesStore_AttachCycle(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())
	cyclesStore := NewCyclesStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "attachcycle@example.com",
		Username:     "attachcycle",
		DisplayName:  "Attach Cycle",
		PasswordHash: "hashed",
	}
	usersStore.Create(context.Background(), u)

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)
	c := createTestColumn(t, columnsStore, p.ID, true)
	cycle := createTestCycle(t, cyclesStore, team.ID, 1, "active")

	i := createTestIssue(t, issuesStore, p.ID, c.ID, u.ID, 1)

	err := issuesStore.AttachCycle(context.Background(), i.ID, cycle.ID)
	if err != nil {
		t.Fatalf("AttachCycle failed: %v", err)
	}

	got, _ := issuesStore.GetByID(context.Background(), i.ID)
	if got.CycleID != cycle.ID {
		t.Errorf("got cycle ID %q, want %q", got.CycleID, cycle.ID)
	}
}

func TestIssuesStore_DetachCycle(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())
	cyclesStore := NewCyclesStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "detachcycle@example.com",
		Username:     "detachcycle",
		DisplayName:  "Detach Cycle",
		PasswordHash: "hashed",
	}
	usersStore.Create(context.Background(), u)

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)
	c := createTestColumn(t, columnsStore, p.ID, true)
	cycle := createTestCycle(t, cyclesStore, team.ID, 1, "active")

	i := createTestIssue(t, issuesStore, p.ID, c.ID, u.ID, 1)
	issuesStore.AttachCycle(context.Background(), i.ID, cycle.ID)

	err := issuesStore.DetachCycle(context.Background(), i.ID)
	if err != nil {
		t.Fatalf("DetachCycle failed: %v", err)
	}

	got, _ := issuesStore.GetByID(context.Background(), i.ID)
	if got.CycleID != "" {
		t.Errorf("expected empty cycle ID, got %q", got.CycleID)
	}
}

func TestIssuesStore_Delete(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "deleteissue@example.com",
		Username:     "deleteissue",
		DisplayName:  "Delete Issue",
		PasswordHash: "hashed",
	}
	usersStore.Create(context.Background(), u)

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)
	c := createTestColumn(t, columnsStore, p.ID, true)
	i := createTestIssue(t, issuesStore, p.ID, c.ID, u.ID, 1)

	err := issuesStore.Delete(context.Background(), i.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	got, _ := issuesStore.GetByID(context.Background(), i.ID)
	if got != nil {
		t.Error("expected issue to be deleted")
	}
}

func TestIssuesStore_Search(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "searchissue@example.com",
		Username:     "searchissue",
		DisplayName:  "Search Issue",
		PasswordHash: "hashed",
	}
	usersStore.Create(context.Background(), u)

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)
	c := createTestColumn(t, columnsStore, p.ID, true)

	// Create issues with different titles
	i1 := &issues.Issue{
		ID:        ulid.Make().String(),
		ProjectID: p.ID,
		Number:    1,
		Key:       "SRCH-1",
		Title:     "Fix login bug",
		ColumnID:  c.ID,
		Position:  0,
		CreatorID: u.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	issuesStore.Create(context.Background(), i1)

	i2 := &issues.Issue{
		ID:        ulid.Make().String(),
		ProjectID: p.ID,
		Number:    2,
		Key:       "SRCH-2",
		Title:     "Add feature",
		ColumnID:  c.ID,
		Position:  0,
		CreatorID: u.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	issuesStore.Create(context.Background(), i2)

	// Search for "login"
	list, err := issuesStore.Search(context.Background(), p.ID, "login", 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("got %d results, want 1", len(list))
	}
	if len(list) > 0 && list[0].Title != "Fix login bug" {
		t.Errorf("got title %q, want 'Fix login bug'", list[0].Title)
	}
}

func TestIssuesStore_Search_ByKey(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "searchkey@example.com",
		Username:     "searchkey",
		DisplayName:  "Search Key",
		PasswordHash: "hashed",
	}
	usersStore.Create(context.Background(), u)

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)
	c := createTestColumn(t, columnsStore, p.ID, true)

	i := &issues.Issue{
		ID:        ulid.Make().String(),
		ProjectID: p.ID,
		Number:    123,
		Key:       "KEY-123",
		Title:     "Some Issue",
		ColumnID:  c.ID,
		Position:  0,
		CreatorID: u.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	issuesStore.Create(context.Background(), i)

	list, err := issuesStore.Search(context.Background(), p.ID, "KEY-123", 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("got %d results, want 1", len(list))
	}
}

func TestIssuesStore_Search_NoResults(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)

	list, err := issuesStore.Search(context.Background(), p.ID, "nonexistent", 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("got %d results, want 0", len(list))
	}
}
