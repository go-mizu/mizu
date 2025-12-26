package duckdb

import (
	"context"
	"testing"

	"github.com/go-mizu/blueprints/kanban/feature/columns"
	"github.com/oklog/ulid/v2"
)

func createTestColumn(t *testing.T, store *ColumnsStore, projectID string, isDefault bool) *columns.Column {
	t.Helper()
	id := ulid.Make().String()
	c := &columns.Column{
		ID:         id,
		ProjectID:  projectID,
		Name:       "Col-" + id,
		Position:   0,
		IsDefault:  isDefault,
		IsArchived: false,
	}
	if err := store.Create(context.Background(), c); err != nil {
		t.Fatalf("failed to create test column: %v", err)
	}
	return c
}

func TestColumnsStore_Create(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)

	c := &columns.Column{
		ID:         ulid.Make().String(),
		ProjectID:  p.ID,
		Name:       "Todo",
		Position:   0,
		IsDefault:  true,
		IsArchived: false,
	}

	err := columnsStore.Create(context.Background(), c)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := columnsStore.GetByID(context.Background(), c.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected column to be created")
	}
	if got.Name != c.Name {
		t.Errorf("got name %q, want %q", got.Name, c.Name)
	}
}

func TestColumnsStore_Create_DuplicateName(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)

	c1 := &columns.Column{
		ID:        ulid.Make().String(),
		ProjectID: p.ID,
		Name:      "Duplicate",
		Position:  0,
	}
	c2 := &columns.Column{
		ID:        ulid.Make().String(),
		ProjectID: p.ID,
		Name:      "Duplicate", // same name
		Position:  1,
	}

	if err := columnsStore.Create(context.Background(), c1); err != nil {
		t.Fatalf("first Create failed: %v", err)
	}

	err := columnsStore.Create(context.Background(), c2)
	if err == nil {
		t.Error("expected error for duplicate name")
	}
}

func TestColumnsStore_GetByID(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)
	c := createTestColumn(t, columnsStore, p.ID, true)

	got, err := columnsStore.GetByID(context.Background(), c.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected column")
	}
	if got.ID != c.ID {
		t.Errorf("got ID %q, want %q", got.ID, c.ID)
	}
}

func TestColumnsStore_GetByID_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	columnsStore := NewColumnsStore(store.DB())

	got, err := columnsStore.GetByID(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent column")
	}
}

func TestColumnsStore_ListByProject(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)
	createTestColumn(t, columnsStore, p.ID, true)
	createTestColumn(t, columnsStore, p.ID, false)

	list, err := columnsStore.ListByProject(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("ListByProject failed: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("got %d columns, want 2", len(list))
	}
}

func TestColumnsStore_ListByProject_ExcludesArchived(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)

	c1 := createTestColumn(t, columnsStore, p.ID, true)
	c2 := createTestColumn(t, columnsStore, p.ID, false)

	// Archive one column
	columnsStore.Archive(context.Background(), c2.ID)

	list, err := columnsStore.ListByProject(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("ListByProject failed: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("got %d columns, want 1 (archived should be excluded)", len(list))
	}
	if list[0].ID != c1.ID {
		t.Errorf("expected non-archived column")
	}
}

func TestColumnsStore_Update(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)
	c := createTestColumn(t, columnsStore, p.ID, true)

	newName := "Updated Column"
	err := columnsStore.Update(context.Background(), c.ID, &columns.UpdateIn{
		Name: &newName,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, _ := columnsStore.GetByID(context.Background(), c.ID)
	if got.Name != newName {
		t.Errorf("got name %q, want %q", got.Name, newName)
	}
}

func TestColumnsStore_UpdatePosition(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)
	c := createTestColumn(t, columnsStore, p.ID, true)

	err := columnsStore.UpdatePosition(context.Background(), c.ID, 5)
	if err != nil {
		t.Fatalf("UpdatePosition failed: %v", err)
	}

	got, _ := columnsStore.GetByID(context.Background(), c.ID)
	if got.Position != 5 {
		t.Errorf("got position %d, want 5", got.Position)
	}
}

func TestColumnsStore_SetDefault(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)

	c1 := createTestColumn(t, columnsStore, p.ID, true)
	c2 := createTestColumn(t, columnsStore, p.ID, false)

	// Set c2 as default
	err := columnsStore.SetDefault(context.Background(), p.ID, c2.ID)
	if err != nil {
		t.Fatalf("SetDefault failed: %v", err)
	}

	// c1 should no longer be default
	got1, _ := columnsStore.GetByID(context.Background(), c1.ID)
	if got1.IsDefault {
		t.Error("expected c1 to not be default")
	}

	// c2 should be default
	got2, _ := columnsStore.GetByID(context.Background(), c2.ID)
	if !got2.IsDefault {
		t.Error("expected c2 to be default")
	}
}

func TestColumnsStore_Archive(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)
	c := createTestColumn(t, columnsStore, p.ID, true)

	err := columnsStore.Archive(context.Background(), c.ID)
	if err != nil {
		t.Fatalf("Archive failed: %v", err)
	}

	got, _ := columnsStore.GetByID(context.Background(), c.ID)
	if !got.IsArchived {
		t.Error("expected column to be archived")
	}
}

func TestColumnsStore_Unarchive(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)
	c := createTestColumn(t, columnsStore, p.ID, true)

	columnsStore.Archive(context.Background(), c.ID)

	err := columnsStore.Unarchive(context.Background(), c.ID)
	if err != nil {
		t.Fatalf("Unarchive failed: %v", err)
	}

	got, _ := columnsStore.GetByID(context.Background(), c.ID)
	if got.IsArchived {
		t.Error("expected column to be unarchived")
	}
}

func TestColumnsStore_Delete(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)
	c := createTestColumn(t, columnsStore, p.ID, true)

	err := columnsStore.Delete(context.Background(), c.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	got, _ := columnsStore.GetByID(context.Background(), c.ID)
	if got != nil {
		t.Error("expected column to be deleted")
	}
}

func TestColumnsStore_GetDefault(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)
	c := createTestColumn(t, columnsStore, p.ID, true)

	got, err := columnsStore.GetDefault(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("GetDefault failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected default column")
	}
	if got.ID != c.ID {
		t.Errorf("got ID %q, want %q", got.ID, c.ID)
	}
}

func TestColumnsStore_GetDefault_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)

	// Create column without default
	createTestColumn(t, columnsStore, p.ID, false)

	got, err := columnsStore.GetDefault(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("GetDefault failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil when no default column")
	}
}
