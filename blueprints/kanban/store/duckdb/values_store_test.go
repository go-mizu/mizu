package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/kanban/feature/fields"
	"github.com/go-mizu/blueprints/kanban/feature/issues"
	"github.com/go-mizu/blueprints/kanban/feature/users"
	"github.com/go-mizu/blueprints/kanban/feature/values"
	"github.com/oklog/ulid/v2"
)

func ptr[T any](v T) *T {
	return &v
}

func TestValuesStore_Set(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())
	fieldsStore := NewFieldsStore(store.DB())
	valuesStore := NewValuesStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "values@example.com",
		Username:     "valuser",
		DisplayName:  "Values User",
		PasswordHash: "hashed",
	}
	usersStore.Create(context.Background(), u)

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)
	c := createTestColumn(t, columnsStore, p.ID, true)
	f := createTestField(t, fieldsStore, p.ID, "priority", fields.KindText)

	i := &issues.Issue{
		ID:        ulid.Make().String(),
		ProjectID: p.ID,
		Number:    1,
		Key:       "VAL-1",
		Title:     "Test Issue",
		ColumnID:  c.ID,
		Position:  0,
		CreatorID: u.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	issuesStore.Create(context.Background(), i)

	v := &values.Value{
		IssueID:   i.ID,
		FieldID:   f.ID,
		ValueText: ptr("high"),
	}

	err := valuesStore.Set(context.Background(), v)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	got, err := valuesStore.Get(context.Background(), i.ID, f.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected value")
	}
	if got.ValueText == nil || *got.ValueText != "high" {
		t.Errorf("got ValueText %v, want %q", got.ValueText, "high")
	}
}

func TestValuesStore_Set_Upsert(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())
	fieldsStore := NewFieldsStore(store.DB())
	valuesStore := NewValuesStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "upsert@example.com",
		Username:     "upsertuser",
		DisplayName:  "Upsert User",
		PasswordHash: "hashed",
	}
	usersStore.Create(context.Background(), u)

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)
	c := createTestColumn(t, columnsStore, p.ID, true)
	f := createTestField(t, fieldsStore, p.ID, "status", fields.KindText)

	i := &issues.Issue{
		ID:        ulid.Make().String(),
		ProjectID: p.ID,
		Number:    1,
		Key:       "UPS-1",
		Title:     "Test Issue",
		ColumnID:  c.ID,
		Position:  0,
		CreatorID: u.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	issuesStore.Create(context.Background(), i)

	// First set
	v1 := &values.Value{
		IssueID:   i.ID,
		FieldID:   f.ID,
		ValueText: ptr("todo"),
	}
	valuesStore.Set(context.Background(), v1)

	// Update (upsert)
	v2 := &values.Value{
		IssueID:   i.ID,
		FieldID:   f.ID,
		ValueText: ptr("done"),
	}
	err := valuesStore.Set(context.Background(), v2)
	if err != nil {
		t.Fatalf("Set (upsert) failed: %v", err)
	}

	got, _ := valuesStore.Get(context.Background(), i.ID, f.ID)
	if got.ValueText == nil || *got.ValueText != "done" {
		t.Errorf("got ValueText %v, want %q", got.ValueText, "done")
	}
}

func TestValuesStore_Set_AllTypes(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())
	fieldsStore := NewFieldsStore(store.DB())
	valuesStore := NewValuesStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "alltypes@example.com",
		Username:     "alltypes",
		DisplayName:  "All Types",
		PasswordHash: "hashed",
	}
	usersStore.Create(context.Background(), u)

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)
	c := createTestColumn(t, columnsStore, p.ID, true)

	fText := createTestField(t, fieldsStore, p.ID, "text_field", fields.KindText)
	fNum := createTestField(t, fieldsStore, p.ID, "num_field", fields.KindNumber)
	fBool := createTestField(t, fieldsStore, p.ID, "bool_field", fields.KindBool)
	fDate := createTestField(t, fieldsStore, p.ID, "date_field", fields.KindDate)
	fTS := createTestField(t, fieldsStore, p.ID, "ts_field", fields.KindTS)
	fRef := createTestField(t, fieldsStore, p.ID, "ref_field", fields.KindUser)
	fJSON := createTestField(t, fieldsStore, p.ID, "json_field", fields.KindJSON)

	i := &issues.Issue{
		ID:        ulid.Make().String(),
		ProjectID: p.ID,
		Number:    1,
		Key:       "TYP-1",
		Title:     "Test Issue",
		ColumnID:  c.ID,
		Position:  0,
		CreatorID: u.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	issuesStore.Create(context.Background(), i)

	now := time.Now().Truncate(time.Second)
	dateOnly := time.Date(2024, 12, 25, 0, 0, 0, 0, time.UTC)

	// Test text value
	valuesStore.Set(context.Background(), &values.Value{IssueID: i.ID, FieldID: fText.ID, ValueText: ptr("hello")})
	gotText, _ := valuesStore.Get(context.Background(), i.ID, fText.ID)
	if gotText.ValueText == nil || *gotText.ValueText != "hello" {
		t.Errorf("text value: got %v, want %q", gotText.ValueText, "hello")
	}

	// Test number value
	valuesStore.Set(context.Background(), &values.Value{IssueID: i.ID, FieldID: fNum.ID, ValueNum: ptr(42.5)})
	gotNum, _ := valuesStore.Get(context.Background(), i.ID, fNum.ID)
	if gotNum.ValueNum == nil || *gotNum.ValueNum != 42.5 {
		t.Errorf("num value: got %v, want %v", gotNum.ValueNum, 42.5)
	}

	// Test bool value
	valuesStore.Set(context.Background(), &values.Value{IssueID: i.ID, FieldID: fBool.ID, ValueBool: ptr(true)})
	gotBool, _ := valuesStore.Get(context.Background(), i.ID, fBool.ID)
	if gotBool.ValueBool == nil || *gotBool.ValueBool != true {
		t.Errorf("bool value: got %v, want %v", gotBool.ValueBool, true)
	}

	// Test date value
	valuesStore.Set(context.Background(), &values.Value{IssueID: i.ID, FieldID: fDate.ID, ValueDate: &dateOnly})
	gotDate, _ := valuesStore.Get(context.Background(), i.ID, fDate.ID)
	if gotDate.ValueDate == nil {
		t.Error("date value: got nil")
	}

	// Test timestamp value
	valuesStore.Set(context.Background(), &values.Value{IssueID: i.ID, FieldID: fTS.ID, ValueTS: &now})
	gotTS, _ := valuesStore.Get(context.Background(), i.ID, fTS.ID)
	if gotTS.ValueTS == nil {
		t.Error("ts value: got nil")
	}

	// Test ref value
	valuesStore.Set(context.Background(), &values.Value{IssueID: i.ID, FieldID: fRef.ID, ValueRef: ptr(u.ID)})
	gotRef, _ := valuesStore.Get(context.Background(), i.ID, fRef.ID)
	if gotRef.ValueRef == nil || *gotRef.ValueRef != u.ID {
		t.Errorf("ref value: got %v, want %v", gotRef.ValueRef, u.ID)
	}

	// Test JSON value
	valuesStore.Set(context.Background(), &values.Value{IssueID: i.ID, FieldID: fJSON.ID, ValueJSON: ptr(`{"foo": "bar"}`)})
	gotJSON, _ := valuesStore.Get(context.Background(), i.ID, fJSON.ID)
	if gotJSON.ValueJSON == nil || *gotJSON.ValueJSON != `{"foo": "bar"}` {
		t.Errorf("json value: got %v, want %q", gotJSON.ValueJSON, `{"foo": "bar"}`)
	}
}

func TestValuesStore_Get_NotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	valuesStore := NewValuesStore(store.DB())

	got, err := valuesStore.Get(context.Background(), "nonexistent", "nonexistent")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent value")
	}
}

func TestValuesStore_ListByIssue(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())
	fieldsStore := NewFieldsStore(store.DB())
	valuesStore := NewValuesStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "listbyissue@example.com",
		Username:     "listbyissue",
		DisplayName:  "List By Issue",
		PasswordHash: "hashed",
	}
	usersStore.Create(context.Background(), u)

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)
	c := createTestColumn(t, columnsStore, p.ID, true)

	f1 := createTestField(t, fieldsStore, p.ID, "field1", fields.KindText)
	f2 := createTestField(t, fieldsStore, p.ID, "field2", fields.KindNumber)

	i := &issues.Issue{
		ID:        ulid.Make().String(),
		ProjectID: p.ID,
		Number:    1,
		Key:       "LBI-1",
		Title:     "Test Issue",
		ColumnID:  c.ID,
		Position:  0,
		CreatorID: u.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	issuesStore.Create(context.Background(), i)

	valuesStore.Set(context.Background(), &values.Value{IssueID: i.ID, FieldID: f1.ID, ValueText: ptr("text")})
	valuesStore.Set(context.Background(), &values.Value{IssueID: i.ID, FieldID: f2.ID, ValueNum: ptr(123.0)})

	list, err := valuesStore.ListByIssue(context.Background(), i.ID)
	if err != nil {
		t.Fatalf("ListByIssue failed: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("got %d values, want 2", len(list))
	}
}

func TestValuesStore_ListByIssue_Empty(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	valuesStore := NewValuesStore(store.DB())

	list, err := valuesStore.ListByIssue(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("ListByIssue failed: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("got %d values, want 0", len(list))
	}
}

func TestValuesStore_ListByField(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())
	fieldsStore := NewFieldsStore(store.DB())
	valuesStore := NewValuesStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "listbyfield@example.com",
		Username:     "listbyfield",
		DisplayName:  "List By Field",
		PasswordHash: "hashed",
	}
	usersStore.Create(context.Background(), u)

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)
	c := createTestColumn(t, columnsStore, p.ID, true)

	f := createTestField(t, fieldsStore, p.ID, "shared_field", fields.KindText)

	i1 := &issues.Issue{
		ID:        ulid.Make().String(),
		ProjectID: p.ID,
		Number:    1,
		Key:       "LBF-1",
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
		Key:       "LBF-2",
		Title:     "Issue 2",
		ColumnID:  c.ID,
		Position:  0,
		CreatorID: u.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	issuesStore.Create(context.Background(), i1)
	issuesStore.Create(context.Background(), i2)

	valuesStore.Set(context.Background(), &values.Value{IssueID: i1.ID, FieldID: f.ID, ValueText: ptr("val1")})
	valuesStore.Set(context.Background(), &values.Value{IssueID: i2.ID, FieldID: f.ID, ValueText: ptr("val2")})

	list, err := valuesStore.ListByField(context.Background(), f.ID)
	if err != nil {
		t.Fatalf("ListByField failed: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("got %d values, want 2", len(list))
	}
}

func TestValuesStore_Delete(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())
	fieldsStore := NewFieldsStore(store.DB())
	valuesStore := NewValuesStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "delete@example.com",
		Username:     "deleteuser",
		DisplayName:  "Delete User",
		PasswordHash: "hashed",
	}
	usersStore.Create(context.Background(), u)

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)
	c := createTestColumn(t, columnsStore, p.ID, true)
	f := createTestField(t, fieldsStore, p.ID, "deleteme", fields.KindText)

	i := &issues.Issue{
		ID:        ulid.Make().String(),
		ProjectID: p.ID,
		Number:    1,
		Key:       "DEL-1",
		Title:     "Test Issue",
		ColumnID:  c.ID,
		Position:  0,
		CreatorID: u.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	issuesStore.Create(context.Background(), i)

	valuesStore.Set(context.Background(), &values.Value{IssueID: i.ID, FieldID: f.ID, ValueText: ptr("delete me")})

	err := valuesStore.Delete(context.Background(), i.ID, f.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	got, _ := valuesStore.Get(context.Background(), i.ID, f.ID)
	if got != nil {
		t.Error("expected value to be deleted")
	}
}

func TestValuesStore_DeleteByIssue(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())
	fieldsStore := NewFieldsStore(store.DB())
	valuesStore := NewValuesStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "deletebyissue@example.com",
		Username:     "deletebyissue",
		DisplayName:  "Delete By Issue",
		PasswordHash: "hashed",
	}
	usersStore.Create(context.Background(), u)

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)
	c := createTestColumn(t, columnsStore, p.ID, true)

	f1 := createTestField(t, fieldsStore, p.ID, "field1", fields.KindText)
	f2 := createTestField(t, fieldsStore, p.ID, "field2", fields.KindNumber)

	i := &issues.Issue{
		ID:        ulid.Make().String(),
		ProjectID: p.ID,
		Number:    1,
		Key:       "DBI-1",
		Title:     "Test Issue",
		ColumnID:  c.ID,
		Position:  0,
		CreatorID: u.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	issuesStore.Create(context.Background(), i)

	valuesStore.Set(context.Background(), &values.Value{IssueID: i.ID, FieldID: f1.ID, ValueText: ptr("text")})
	valuesStore.Set(context.Background(), &values.Value{IssueID: i.ID, FieldID: f2.ID, ValueNum: ptr(123.0)})

	err := valuesStore.DeleteByIssue(context.Background(), i.ID)
	if err != nil {
		t.Fatalf("DeleteByIssue failed: %v", err)
	}

	list, _ := valuesStore.ListByIssue(context.Background(), i.ID)
	if len(list) != 0 {
		t.Errorf("got %d values, want 0 after delete by issue", len(list))
	}
}

func TestValuesStore_BulkSet(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())
	fieldsStore := NewFieldsStore(store.DB())
	valuesStore := NewValuesStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "bulkset@example.com",
		Username:     "bulkset",
		DisplayName:  "Bulk Set",
		PasswordHash: "hashed",
	}
	usersStore.Create(context.Background(), u)

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)
	c := createTestColumn(t, columnsStore, p.ID, true)

	f1 := createTestField(t, fieldsStore, p.ID, "bulk1", fields.KindText)
	f2 := createTestField(t, fieldsStore, p.ID, "bulk2", fields.KindNumber)

	i := &issues.Issue{
		ID:        ulid.Make().String(),
		ProjectID: p.ID,
		Number:    1,
		Key:       "BLK-1",
		Title:     "Test Issue",
		ColumnID:  c.ID,
		Position:  0,
		CreatorID: u.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	issuesStore.Create(context.Background(), i)

	vs := []*values.Value{
		{IssueID: i.ID, FieldID: f1.ID, ValueText: ptr("bulk text")},
		{IssueID: i.ID, FieldID: f2.ID, ValueNum: ptr(999.0)},
	}

	err := valuesStore.BulkSet(context.Background(), vs)
	if err != nil {
		t.Fatalf("BulkSet failed: %v", err)
	}

	list, _ := valuesStore.ListByIssue(context.Background(), i.ID)
	if len(list) != 2 {
		t.Errorf("got %d values, want 2", len(list))
	}
}

func TestValuesStore_BulkSet_Empty(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	valuesStore := NewValuesStore(store.DB())

	err := valuesStore.BulkSet(context.Background(), []*values.Value{})
	if err != nil {
		t.Fatalf("BulkSet empty should not error: %v", err)
	}
}

func TestValuesStore_BulkGetByIssues(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	wsStore := NewWorkspacesStore(store.DB())
	teamsStore := NewTeamsStore(store.DB())
	projectsStore := NewProjectsStore(store.DB())
	columnsStore := NewColumnsStore(store.DB())
	usersStore := NewUsersStore(store.DB())
	issuesStore := NewIssuesStore(store.DB())
	fieldsStore := NewFieldsStore(store.DB())
	valuesStore := NewValuesStore(store.DB())

	u := &users.User{
		ID:           ulid.Make().String(),
		Email:        "bulkget@example.com",
		Username:     "bulkget",
		DisplayName:  "Bulk Get",
		PasswordHash: "hashed",
	}
	usersStore.Create(context.Background(), u)

	w := createTestWorkspace(t, wsStore)
	team := createTestTeam(t, teamsStore, w.ID)
	p := createTestProject(t, projectsStore, team.ID)
	c := createTestColumn(t, columnsStore, p.ID, true)
	f := createTestField(t, fieldsStore, p.ID, "bulkgetfield", fields.KindText)

	i1 := &issues.Issue{
		ID:        ulid.Make().String(),
		ProjectID: p.ID,
		Number:    1,
		Key:       "BGI-1",
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
		Key:       "BGI-2",
		Title:     "Issue 2",
		ColumnID:  c.ID,
		Position:  0,
		CreatorID: u.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	issuesStore.Create(context.Background(), i1)
	issuesStore.Create(context.Background(), i2)

	valuesStore.Set(context.Background(), &values.Value{IssueID: i1.ID, FieldID: f.ID, ValueText: ptr("val1")})
	valuesStore.Set(context.Background(), &values.Value{IssueID: i2.ID, FieldID: f.ID, ValueText: ptr("val2")})

	result, err := valuesStore.BulkGetByIssues(context.Background(), []string{i1.ID, i2.ID})
	if err != nil {
		t.Fatalf("BulkGetByIssues failed: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("got %d issues with values, want 2", len(result))
	}
	if len(result[i1.ID]) != 1 {
		t.Errorf("got %d values for i1, want 1", len(result[i1.ID]))
	}
	if len(result[i2.ID]) != 1 {
		t.Errorf("got %d values for i2, want 1", len(result[i2.ID]))
	}
}

func TestValuesStore_BulkGetByIssues_Empty(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	valuesStore := NewValuesStore(store.DB())

	result, err := valuesStore.BulkGetByIssues(context.Background(), []string{})
	if err != nil {
		t.Fatalf("BulkGetByIssues empty should not error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("got %d results, want 0", len(result))
	}
}
