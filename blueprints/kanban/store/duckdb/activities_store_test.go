package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/kanban/feature/activities"
	"github.com/go-mizu/blueprints/kanban/feature/issues"
	"github.com/go-mizu/blueprints/kanban/feature/users"
	"github.com/oklog/ulid/v2"
)

func TestActivitiesStore_Create(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())
	activitiesStore := NewActivitiesStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "activity@example.com",
		Username:     "activityuser",
		DisplayName:  "Activity User",
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
		Key:       "ACT-1",
		Title:     "Test Issue",
		ColumnID:  col.ID,
		Position:  0,
		CreatorID: u.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	issuesStore.Create(context.Background(), i)

	a := &activities.Activity{
		ID:        ulid.Make().String(),
		IssueID:   i.ID,
		ActorID:   u.ID,
		Action:    activities.ActionIssueCreated,
		CreatedAt: time.Now(),
	}

	err := activitiesStore.Create(context.Background(), a)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := activitiesStore.GetByID(context.Background(), a.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected activity to be created")
	}
	if got.Action != a.Action {
		t.Errorf("got action %q, want %q", got.Action, a.Action)
	}
}

func TestActivitiesStore_CreateWithValues(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())
	activitiesStore := NewActivitiesStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "activityval@example.com",
		Username:     "activityval",
		DisplayName:  "Activity Val",
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
		Key:       "ACV-1",
		Title:     "Test Issue",
		ColumnID:  col.ID,
		Position:  0,
		CreatorID: u.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	issuesStore.Create(context.Background(), i)

	a := &activities.Activity{
		ID:        ulid.Make().String(),
		IssueID:   i.ID,
		ActorID:   u.ID,
		Action:    activities.ActionStatusChanged,
		OldValue:  "Backlog",
		NewValue:  "In Progress",
		Metadata:  `{"column_id":"col123"}`,
		CreatedAt: time.Now(),
	}

	err := activitiesStore.Create(context.Background(), a)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := activitiesStore.GetByID(context.Background(), a.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got.OldValue != a.OldValue {
		t.Errorf("got old_value %q, want %q", got.OldValue, a.OldValue)
	}
	if got.NewValue != a.NewValue {
		t.Errorf("got new_value %q, want %q", got.NewValue, a.NewValue)
	}
	if got.Metadata != a.Metadata {
		t.Errorf("got metadata %q, want %q", got.Metadata, a.Metadata)
	}
}

func TestActivitiesStore_GetByID(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())
	activitiesStore := NewActivitiesStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "getact@example.com",
		Username:     "getact",
		DisplayName:  "Get Activity",
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

	a := &activities.Activity{
		ID:        ulid.Make().String(),
		IssueID:   i.ID,
		ActorID:   u.ID,
		Action:    activities.ActionPriorityChanged,
		OldValue:  "None",
		NewValue:  "High",
		CreatedAt: time.Now(),
	}
	activitiesStore.Create(context.Background(), a)

	got, err := activitiesStore.GetByID(context.Background(), a.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected activity")
	}
	if got.ID != a.ID {
		t.Errorf("got ID %q, want %q", got.ID, a.ID)
	}
}

func TestActivitiesStore_GetByID_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	activitiesStore := NewActivitiesStore(store.DB())

	got, err := activitiesStore.GetByID(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent activity")
	}
}

func TestActivitiesStore_ListByIssue(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())
	activitiesStore := NewActivitiesStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "listact@example.com",
		Username:     "listact",
		DisplayName:  "List Activity",
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

	a1 := &activities.Activity{
		ID:        ulid.Make().String(),
		IssueID:   i.ID,
		ActorID:   u.ID,
		Action:    activities.ActionIssueCreated,
		CreatedAt: time.Now(),
	}
	a2 := &activities.Activity{
		ID:        ulid.Make().String(),
		IssueID:   i.ID,
		ActorID:   u.ID,
		Action:    activities.ActionStatusChanged,
		OldValue:  "Backlog",
		NewValue:  "Todo",
		CreatedAt: time.Now().Add(1 * time.Second),
	}
	activitiesStore.Create(context.Background(), a1)
	activitiesStore.Create(context.Background(), a2)

	list, err := activitiesStore.ListByIssue(context.Background(), i.ID)
	if err != nil {
		t.Fatalf("ListByIssue failed: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("got %d activities, want 2", len(list))
	}
}

func TestActivitiesStore_ListByIssue_Empty(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())
	activitiesStore := NewActivitiesStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "emptyact@example.com",
		Username:     "emptyact",
		DisplayName:  "Empty Activity",
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

	list, err := activitiesStore.ListByIssue(context.Background(), i.ID)
	if err != nil {
		t.Fatalf("ListByIssue failed: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("got %d activities, want 0", len(list))
	}
}

func TestActivitiesStore_ListByWorkspace(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())
	activitiesStore := NewActivitiesStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "wsact@example.com",
		Username:     "wsact",
		DisplayName:  "Workspace Activity",
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
		Key:       "WSA-1",
		Title:     "Test Issue",
		ColumnID:  col.ID,
		Position:  0,
		CreatorID: u.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	issuesStore.Create(context.Background(), i)

	a := &activities.Activity{
		ID:        ulid.Make().String(),
		IssueID:   i.ID,
		ActorID:   u.ID,
		Action:    activities.ActionIssueCreated,
		CreatedAt: time.Now(),
	}
	activitiesStore.Create(context.Background(), a)

	list, err := activitiesStore.ListByWorkspace(context.Background(), w.ID, 10, 0)
	if err != nil {
		t.Fatalf("ListByWorkspace failed: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("got %d activities, want 1", len(list))
	}
	if list[0].ActorName != u.DisplayName {
		t.Errorf("got actor name %q, want %q", list[0].ActorName, u.DisplayName)
	}
	if list[0].IssueKey != i.Key {
		t.Errorf("got issue key %q, want %q", list[0].IssueKey, i.Key)
	}
}

func TestActivitiesStore_Delete(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())
	activitiesStore := NewActivitiesStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "delact@example.com",
		Username:     "delact",
		DisplayName:  "Delete Activity",
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

	a := &activities.Activity{
		ID:        ulid.Make().String(),
		IssueID:   i.ID,
		ActorID:   u.ID,
		Action:    activities.ActionIssueCreated,
		CreatedAt: time.Now(),
	}
	activitiesStore.Create(context.Background(), a)

	err := activitiesStore.Delete(context.Background(), a.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	got, _ := activitiesStore.GetByID(context.Background(), a.ID)
	if got != nil {
		t.Error("expected activity to be deleted")
	}
}

func TestActivitiesStore_CountByIssue(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())
	activitiesStore := NewActivitiesStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "cntact@example.com",
		Username:     "cntact",
		DisplayName:  "Count Activity",
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

	// Add 3 activities
	for n := 0; n < 3; n++ {
		a := &activities.Activity{
			ID:        ulid.Make().String(),
			IssueID:   i.ID,
			ActorID:   u.ID,
			Action:    activities.ActionIssueUpdated,
			CreatedAt: time.Now(),
		}
		activitiesStore.Create(context.Background(), a)
	}

	count, err := activitiesStore.CountByIssue(context.Background(), i.ID)
	if err != nil {
		t.Fatalf("CountByIssue failed: %v", err)
	}
	if count != 3 {
		t.Errorf("got count %d, want 3", count)
	}
}

func TestActivitiesStore_CountByIssue_Zero(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())
	activitiesStore := NewActivitiesStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "zerocnt@example.com",
		Username:     "zerocnt",
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

	count, err := activitiesStore.CountByIssue(context.Background(), i.ID)
	if err != nil {
		t.Fatalf("CountByIssue failed: %v", err)
	}
	if count != 0 {
		t.Errorf("got count %d, want 0", count)
	}
}
