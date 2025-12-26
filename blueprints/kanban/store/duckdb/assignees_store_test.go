package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/kanban/feature/issues"
	"github.com/go-mizu/blueprints/kanban/feature/users"
	"github.com/oklog/ulid/v2"
)

func TestAssigneesStore_Add(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())
	assigneesStore := NewAssigneesStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "assignee@example.com",
		Username:     "assignee",
		DisplayName:  "Assignee",
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
		Key:       "ASN-1",
		Title:     "Test Issue",
		ColumnID:  c.ID,
		Position:  0,
		CreatorID: u.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	issuesStore.Create(context.Background(), i)

	err := assigneesStore.Add(context.Background(), i.ID, u.ID)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	list, _ := assigneesStore.List(context.Background(), i.ID)
	if len(list) != 1 {
		t.Errorf("got %d assignees, want 1", len(list))
	}
	if len(list) > 0 && list[0] != u.ID {
		t.Errorf("got assignee %q, want %q", list[0], u.ID)
	}
}

func TestAssigneesStore_Add_Duplicate(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())
	assigneesStore := NewAssigneesStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "dupasn@example.com",
		Username:     "dupasn",
		DisplayName:  "Dup Assignee",
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
		Key:       "DUP-1",
		Title:     "Test Issue",
		ColumnID:  c.ID,
		Position:  0,
		CreatorID: u.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	issuesStore.Create(context.Background(), i)

	// Add twice - should not error
	assigneesStore.Add(context.Background(), i.ID, u.ID)
	err := assigneesStore.Add(context.Background(), i.ID, u.ID)
	if err != nil {
		t.Fatalf("Add duplicate should not error: %v", err)
	}

	list, _ := assigneesStore.List(context.Background(), i.ID)
	if len(list) != 1 {
		t.Errorf("got %d assignees, want 1 (no duplicates)", len(list))
	}
}

func TestAssigneesStore_Remove(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())
	assigneesStore := NewAssigneesStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "removeasn@example.com",
		Username:     "removeasn",
		DisplayName:  "Remove Assignee",
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
		Key:       "REM-1",
		Title:     "Test Issue",
		ColumnID:  c.ID,
		Position:  0,
		CreatorID: u.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	issuesStore.Create(context.Background(), i)

	assigneesStore.Add(context.Background(), i.ID, u.ID)

	err := assigneesStore.Remove(context.Background(), i.ID, u.ID)
	if err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	list, _ := assigneesStore.List(context.Background(), i.ID)
	if len(list) != 0 {
		t.Errorf("got %d assignees, want 0", len(list))
	}
}

func TestAssigneesStore_Remove_NotAssigned(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())
	assigneesStore := NewAssigneesStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "notasn@example.com",
		Username:     "notasn",
		DisplayName:  "Not Assigned",
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
		Key:       "NOT-1",
		Title:     "Test Issue",
		ColumnID:  c.ID,
		Position:  0,
		CreatorID: u.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	issuesStore.Create(context.Background(), i)

	// Remove non-assigned user should not error
	err := assigneesStore.Remove(context.Background(), i.ID, u.ID)
	if err != nil {
		t.Fatalf("Remove non-assigned should not error: %v", err)
	}
}

func TestAssigneesStore_List(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())
	assigneesStore := NewAssigneesStore(store.DB())

	u1 := &users.User{
		ID:           ulid.Make().String(),
		Email:        "asn1@example.com",
		Username:     "asn1",
		DisplayName:  "Assignee 1",
		PasswordHash: "hashed",
	}
	u2 := &users.User{
		ID:           ulid.Make().String(),
		Email:        "asn2@example.com",
		Username:     "asn2",
		DisplayName:  "Assignee 2",
		PasswordHash: "hashed",
	}
	usersStore.Create(context.Background(), u1)
	usersStore.Create(context.Background(), u2)

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)
	c := createTestColumn(t, columnsStore, p.ID, true)

	i := &issues.Issue{
		ID:        ulid.Make().String(),
		ProjectID: p.ID,
		Number:    1,
		Key:       "LST-1",
		Title:     "Test Issue",
		ColumnID:  c.ID,
		Position:  0,
		CreatorID: u1.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	issuesStore.Create(context.Background(), i)

	assigneesStore.Add(context.Background(), i.ID, u1.ID)
	assigneesStore.Add(context.Background(), i.ID, u2.ID)

	list, err := assigneesStore.List(context.Background(), i.ID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("got %d assignees, want 2", len(list))
	}
}

func TestAssigneesStore_List_Empty(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())
	assigneesStore := NewAssigneesStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "empty@example.com",
		Username:     "emptyasn",
		DisplayName:  "Empty",
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
		Key:       "EMP-1",
		Title:     "Test Issue",
		ColumnID:  c.ID,
		Position:  0,
		CreatorID: u.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	issuesStore.Create(context.Background(), i)

	list, err := assigneesStore.List(context.Background(), i.ID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("got %d assignees, want 0", len(list))
	}
}

func TestAssigneesStore_ListByUser(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())
	assigneesStore := NewAssigneesStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "byuser@example.com",
		Username:     "byuser",
		DisplayName:  "By User",
		PasswordHash: "hashed",
	}
	usersStore.Create(context.Background(), u)

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)
	c := createTestColumn(t, columnsStore, p.ID, true)

	i1 := &issues.Issue{
		ID:        ulid.Make().String(),
		ProjectID: p.ID,
		Number:    1,
		Key:       "BYU-1",
		Title:     "Issue 1",
		ColumnID:  c.ID,
		Position:  0,
		CreatorID: u.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	i2 := &issues.Issue{
		ID:        ulid.Make().String(),
		ProjectID: p.ID,
		Number:    2,
		Key:       "BYU-2",
		Title:     "Issue 2",
		ColumnID:  c.ID,
		Position:  0,
		CreatorID: u.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	issuesStore.Create(context.Background(), i1)
	issuesStore.Create(context.Background(), i2)

	assigneesStore.Add(context.Background(), i1.ID, u.ID)
	assigneesStore.Add(context.Background(), i2.ID, u.ID)

	list, err := assigneesStore.ListByUser(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("ListByUser failed: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("got %d issues, want 2", len(list))
	}
}

func TestAssigneesStore_ListByUser_Empty(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	assigneesStore := NewAssigneesStore(store.DB())

	list, err := assigneesStore.ListByUser(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("ListByUser failed: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("got %d issues, want 0", len(list))
	}
}
