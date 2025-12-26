package duckdb

import (
	"context"
	"testing"

	"github.com/go-mizu/blueprints/kanban/feature/fields"
	"github.com/oklog/ulid/v2"
)

func createTestField(t *testing.T, store *FieldsStore, projectID string, key, kind string) *fields.Field {
	t.Helper()
	f := &fields.Field{
		ID:         ulid.Make().String(),
		ProjectID:  projectID,
		Key:        key,
		Name:       "Field " + key,
		Kind:       kind,
		Position:   0,
		IsRequired: false,
		IsArchived: false,
	}
	if err := store.Create(context.Background(), f); err != nil {
		t.Fatalf("failed to create test field: %v", err)
	}
	return f
}

func TestFieldsStore_Create(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	fieldsStore := NewFieldsStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)

	f := &fields.Field{
		ID:           ulid.Make().String(),
		ProjectID:    p.ID,
		Key:          "priority",
		Name:         "Priority",
		Kind:         fields.KindSelect,
		Position:     0,
		IsRequired:   true,
		IsArchived:   false,
		SettingsJSON: `{"options": ["low", "medium", "high"]}`,
	}

	err := fieldsStore.Create(context.Background(), f)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := fieldsStore.GetByID(context.Background(), f.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected field to be created")
	}
	if got.Key != "priority" {
		t.Errorf("got key %q, want %q", got.Key, "priority")
	}
	if got.Kind != fields.KindSelect {
		t.Errorf("got kind %q, want %q", got.Kind, fields.KindSelect)
	}
	if !got.IsRequired {
		t.Error("expected IsRequired to be true")
	}
	if got.SettingsJSON != f.SettingsJSON {
		t.Errorf("got settings %q, want %q", got.SettingsJSON, f.SettingsJSON)
	}
}

func TestFieldsStore_Create_DuplicateKey(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	fieldsStore := NewFieldsStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)

	createTestField(t, fieldsStore, p.ID, "duplicate", fields.KindText)

	f2 := &fields.Field{
		ID:        ulid.Make().String(),
		ProjectID: p.ID,
		Key:       "duplicate", // same key
		Name:      "Duplicate Field",
		Kind:      fields.KindText,
	}

	err := fieldsStore.Create(context.Background(), f2)
	if err == nil {
		t.Error("expected error for duplicate key")
	}
}

func TestFieldsStore_GetByID(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	fieldsStore := NewFieldsStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)
	f := createTestField(t, fieldsStore, p.ID, "test", fields.KindText)

	got, err := fieldsStore.GetByID(context.Background(), f.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected field")
	}
	if got.ID != f.ID {
		t.Errorf("got ID %q, want %q", got.ID, f.ID)
	}
}

func TestFieldsStore_GetByID_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	fieldsStore := NewFieldsStore(store.DB())

	got, err := fieldsStore.GetByID(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent field")
	}
}

func TestFieldsStore_GetByKey(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	fieldsStore := NewFieldsStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)
	f := createTestField(t, fieldsStore, p.ID, "mykey", fields.KindNumber)

	got, err := fieldsStore.GetByKey(context.Background(), p.ID, "mykey")
	if err != nil {
		t.Fatalf("GetByKey failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected field")
	}
	if got.ID != f.ID {
		t.Errorf("got ID %q, want %q", got.ID, f.ID)
	}
}

func TestFieldsStore_GetByKey_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	fieldsStore := NewFieldsStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)

	got, err := fieldsStore.GetByKey(context.Background(), p.ID, "nonexistent")
	if err != nil {
		t.Fatalf("GetByKey failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent key")
	}
}

func TestFieldsStore_ListByProject(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	fieldsStore := NewFieldsStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)

	createTestField(t, fieldsStore, p.ID, "field1", fields.KindText)
	createTestField(t, fieldsStore, p.ID, "field2", fields.KindNumber)

	list, err := fieldsStore.ListByProject(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("ListByProject failed: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("got %d fields, want 2", len(list))
	}
}

func TestFieldsStore_ListByProject_ExcludesArchived(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	fieldsStore := NewFieldsStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)

	f1 := createTestField(t, fieldsStore, p.ID, "active", fields.KindText)
	f2 := createTestField(t, fieldsStore, p.ID, "archived", fields.KindText)

	// Archive f2
	fieldsStore.Archive(context.Background(), f2.ID)

	list, err := fieldsStore.ListByProject(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("ListByProject failed: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("got %d fields, want 1 (archived should be excluded)", len(list))
	}
	if list[0].ID != f1.ID {
		t.Errorf("expected active field")
	}
}

func TestFieldsStore_ListByProject_Empty(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	fieldsStore := NewFieldsStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)

	list, err := fieldsStore.ListByProject(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("ListByProject failed: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("got %d fields, want 0", len(list))
	}
}

func TestFieldsStore_Update(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	fieldsStore := NewFieldsStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)
	f := createTestField(t, fieldsStore, p.ID, "updateme", fields.KindText)

	newName := "Updated Field"
	isRequired := true
	err := fieldsStore.Update(context.Background(), f.ID, &fields.UpdateIn{
		Name:       &newName,
		IsRequired: &isRequired,
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, _ := fieldsStore.GetByID(context.Background(), f.ID)
	if got.Name != newName {
		t.Errorf("got name %q, want %q", got.Name, newName)
	}
	if !got.IsRequired {
		t.Error("expected IsRequired to be true")
	}
}

func TestFieldsStore_UpdatePosition(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	fieldsStore := NewFieldsStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)
	f := createTestField(t, fieldsStore, p.ID, "posfield", fields.KindText)

	err := fieldsStore.UpdatePosition(context.Background(), f.ID, 5)
	if err != nil {
		t.Fatalf("UpdatePosition failed: %v", err)
	}

	got, _ := fieldsStore.GetByID(context.Background(), f.ID)
	if got.Position != 5 {
		t.Errorf("got position %d, want 5", got.Position)
	}
}

func TestFieldsStore_Archive(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	fieldsStore := NewFieldsStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)
	f := createTestField(t, fieldsStore, p.ID, "archiveme", fields.KindText)

	err := fieldsStore.Archive(context.Background(), f.ID)
	if err != nil {
		t.Fatalf("Archive failed: %v", err)
	}

	got, _ := fieldsStore.GetByID(context.Background(), f.ID)
	if !got.IsArchived {
		t.Error("expected field to be archived")
	}
}

func TestFieldsStore_Unarchive(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	fieldsStore := NewFieldsStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)
	f := createTestField(t, fieldsStore, p.ID, "unarchiveme", fields.KindText)

	fieldsStore.Archive(context.Background(), f.ID)

	err := fieldsStore.Unarchive(context.Background(), f.ID)
	if err != nil {
		t.Fatalf("Unarchive failed: %v", err)
	}

	got, _ := fieldsStore.GetByID(context.Background(), f.ID)
	if got.IsArchived {
		t.Error("expected field to be unarchived")
	}
}

func TestFieldsStore_Delete(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	fieldsStore := NewFieldsStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)
	f := createTestField(t, fieldsStore, p.ID, "deleteme", fields.KindText)

	err := fieldsStore.Delete(context.Background(), f.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	got, _ := fieldsStore.GetByID(context.Background(), f.ID)
	if got != nil {
		t.Error("expected field to be deleted")
	}
}

func TestFieldsStore_AllKinds(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	fieldsStore := NewFieldsStore(store.DB())

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)

	kinds := []string{
		fields.KindText,
		fields.KindNumber,
		fields.KindBool,
		fields.KindDate,
		fields.KindTS,
		fields.KindSelect,
		fields.KindUser,
		fields.KindJSON,
	}

	for _, kind := range kinds {
		f := &fields.Field{
			ID:        ulid.Make().String(),
			ProjectID: p.ID,
			Key:       kind + "_field",
			Name:      kind + " Field",
			Kind:      kind,
		}
		err := fieldsStore.Create(context.Background(), f)
		if err != nil {
			t.Errorf("Create failed for kind %q: %v", kind, err)
		}

		got, err := fieldsStore.GetByID(context.Background(), f.ID)
		if err != nil {
			t.Errorf("GetByID failed for kind %q: %v", kind, err)
		}
		if got.Kind != kind {
			t.Errorf("got kind %q, want %q", got.Kind, kind)
		}
	}
}
