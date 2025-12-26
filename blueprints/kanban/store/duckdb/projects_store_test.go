package duckdb

import (
	"context"
	"testing"

	"github.com/go-mizu/blueprints/kanban/feature/projects"
	"github.com/oklog/ulid/v2"
)

func createTestProject(t *testing.T, store *ProjectsStore, teamID string) *projects.Project {
	t.Helper()
	id := ulid.Make().String()
	p := &projects.Project{
		ID:           id,
		TeamID:       teamID,
		Key:          "P" + id,
		Name:         "Project " + id,
		IssueCounter: 0,
	}
	if err := store.Create(context.Background(), p); err != nil {
		t.Fatalf("failed to create test project: %v", err)
	}
	return p
}

func TestProjectsStore_Create(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)

	p := &projects.Project{
		ID:           ulid.Make().String(),
		TeamID:       team.ID,
		Key:          "PROJ",
		Name:         "Test Project",
		IssueCounter: 0,
	}

	err := projectsStore.Create(context.Background(), p)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := projectsStore.GetByID(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected project to be created")
	}
	if got.Key != p.Key {
		t.Errorf("got key %q, want %q", got.Key, p.Key)
	}
}

func TestProjectsStore_Create_DuplicateKey(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)

	p1 := &projects.Project{
		ID:     ulid.Make().String(),
		TeamID: team.ID,
		Key:    "DUP",
		Name:   "Project 1",
	}
	p2 := &projects.Project{
		ID:     ulid.Make().String(),
		TeamID: team.ID,
		Key:    "DUP", // same key
		Name:   "Project 2",
	}

	if err := projectsStore.Create(context.Background(), p1); err != nil {
		t.Fatalf("first Create failed: %v", err)
	}

	err := projectsStore.Create(context.Background(), p2)
	if err == nil {
		t.Error("expected error for duplicate key")
	}
}

func TestProjectsStore_GetByID(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)

	got, err := projectsStore.GetByID(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected project")
	}
	if got.ID != p.ID {
		t.Errorf("got ID %q, want %q", got.ID, p.ID)
	}
}

func TestProjectsStore_GetByID_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	projectsStore := NewProjectsStore(store.DB())

	got, err := projectsStore.GetByID(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent project")
	}
}

func TestProjectsStore_GetByKey(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)

	got, err := projectsStore.GetByKey(context.Background(), team.ID, p.Key)
	if err != nil {
		t.Fatalf("GetByKey failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected project")
	}
	if got.Key != p.Key {
		t.Errorf("got key %q, want %q", got.Key, p.Key)
	}
}

func TestProjectsStore_GetByKey_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)

	got, err := projectsStore.GetByKey(context.Background(), team.ID, "NONEXIST")
	if err != nil {
		t.Fatalf("GetByKey failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent key")
	}
}

func TestProjectsStore_ListByTeam(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	createTestProject(t, projectsStore, team.ID)
	createTestProject(t, projectsStore, team.ID)

	list, err := projectsStore.ListByTeam(context.Background(), team.ID)
	if err != nil {
		t.Fatalf("ListByTeam failed: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("got %d projects, want 2", len(list))
	}
}

func TestProjectsStore_ListByTeam_Empty(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)

	list, err := projectsStore.ListByTeam(context.Background(), team.ID)
	if err != nil {
		t.Fatalf("ListByTeam failed: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("got %d projects, want 0", len(list))
	}
}

func TestProjectsStore_Update(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)

	newName := "Updated Project"
	err := projectsStore.Update(context.Background(), p.ID, &projects.UpdateIn{
		Name: &newName,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, _ := projectsStore.GetByID(context.Background(), p.ID)
	if got.Name != newName {
		t.Errorf("got name %q, want %q", got.Name, newName)
	}
}

func TestProjectsStore_Delete(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)

	err := projectsStore.Delete(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	got, _ := projectsStore.GetByID(context.Background(), p.ID)
	if got != nil {
		t.Error("expected project to be deleted")
	}
}

func TestProjectsStore_IncrementIssueCounter(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)

	counter1, err := projectsStore.IncrementIssueCounter(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("IncrementIssueCounter failed: %v", err)
	}
	if counter1 != 1 {
		t.Errorf("got counter %d, want 1", counter1)
	}

	counter2, err := projectsStore.IncrementIssueCounter(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("IncrementIssueCounter failed: %v", err)
	}
	if counter2 != 2 {
		t.Errorf("got counter %d, want 2", counter2)
	}
}

func TestProjectsStore_IncrementIssueCounter_Multiple(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)

	// Increment multiple times
	for i := 1; i <= 5; i++ {
		counter, err := projectsStore.IncrementIssueCounter(context.Background(), p.ID)
		if err != nil {
			t.Fatalf("IncrementIssueCounter failed: %v", err)
		}
		if counter != i {
			t.Errorf("iteration %d: got counter %d, want %d", i, counter, i)
		}
	}

	// Verify final state
	got, _ := projectsStore.GetByID(context.Background(), p.ID)
	if got.IssueCounter != 5 {
		t.Errorf("got issue counter %d, want 5", got.IssueCounter)
	}
}
