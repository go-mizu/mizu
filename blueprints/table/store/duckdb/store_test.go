package duckdb_test

import (
	"context"
	"testing"

	"github.com/go-mizu/blueprints/table/feature/bases"
	"github.com/go-mizu/blueprints/table/feature/comments"
	"github.com/go-mizu/blueprints/table/feature/fields"
	"github.com/go-mizu/blueprints/table/feature/records"
	"github.com/go-mizu/blueprints/table/feature/tables"
	"github.com/go-mizu/blueprints/table/feature/users"
	"github.com/go-mizu/blueprints/table/feature/views"
	"github.com/go-mizu/blueprints/table/feature/workspaces"
	"github.com/go-mizu/blueprints/table/pkg/ulid"
	"github.com/go-mizu/blueprints/table/store/duckdb"
)

func setupTestStore(t *testing.T) *duckdb.Store {
	t.Helper()
	dir := t.TempDir()
	store, err := duckdb.Open(dir)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	t.Cleanup(func() {
		store.Close()
	})
	return store
}

func createTestUser(t *testing.T, store *duckdb.Store) *users.User {
	t.Helper()
	user := &users.User{
		ID:           ulid.New(),
		Email:        "test-" + ulid.New() + "@example.com",
		Name:         "Test User",
		PasswordHash: "hashed",
	}
	if err := store.Users().Create(context.Background(), user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	return user
}

func createTestWorkspace(t *testing.T, store *duckdb.Store, owner *users.User) *workspaces.Workspace {
	t.Helper()
	ws := &workspaces.Workspace{
		ID:      ulid.New(),
		Name:    "Test Workspace",
		Slug:    "test-" + ulid.New(),
		OwnerID: owner.ID,
	}
	if err := store.Workspaces().Create(context.Background(), ws); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}
	return ws
}

func createTestBase(t *testing.T, store *duckdb.Store, ws *workspaces.Workspace, user *users.User) *bases.Base {
	t.Helper()
	base := &bases.Base{
		ID:          ulid.New(),
		WorkspaceID: ws.ID,
		Name:        "Test Base",
		CreatedBy:   user.ID,
	}
	if err := store.Bases().Create(context.Background(), base); err != nil {
		t.Fatalf("failed to create base: %v", err)
	}
	return base
}

func createTestTable(t *testing.T, store *duckdb.Store, base *bases.Base, user *users.User) *tables.Table {
	t.Helper()
	tbl := &tables.Table{
		ID:        ulid.New(),
		BaseID:    base.ID,
		Name:      "Test Table",
		CreatedBy: user.ID,
	}
	if err := store.Tables().Create(context.Background(), tbl); err != nil {
		t.Fatalf("failed to create table: %v", err)
	}
	return tbl
}

func createTestField(t *testing.T, store *duckdb.Store, tbl *tables.Table, name, fieldType string, user *users.User) *fields.Field {
	t.Helper()
	field := &fields.Field{
		ID:        ulid.New(),
		TableID:   tbl.ID,
		Name:      name,
		Type:      fieldType,
		CreatedBy: user.ID,
	}
	if err := store.Fields().Create(context.Background(), field); err != nil {
		t.Fatalf("failed to create field: %v", err)
	}
	return field
}

func createTestRecord(t *testing.T, store *duckdb.Store, tbl *tables.Table, user *users.User, cells map[string]any) *records.Record {
	t.Helper()
	rec := &records.Record{
		ID:        ulid.New(),
		TableID:   tbl.ID,
		Cells:     cells,
		CreatedBy: user.ID,
	}
	if err := store.Records().Create(context.Background(), rec); err != nil {
		t.Fatalf("failed to create record: %v", err)
	}
	return rec
}

func createTestView(t *testing.T, store *duckdb.Store, tbl *tables.Table, user *users.User, name string) *views.View {
	t.Helper()
	view := &views.View{
		ID:        ulid.New(),
		TableID:   tbl.ID,
		Name:      name,
		Type:      views.TypeGrid,
		CreatedBy: user.ID,
	}
	if err := store.Views().Create(context.Background(), view); err != nil {
		t.Fatalf("failed to create view: %v", err)
	}
	return view
}

// TestUsersStore tests user CRUD operations.
func TestUsersStore(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	t.Run("Create and GetByID", func(t *testing.T) {
		user := &users.User{
			ID:           ulid.New(),
			Email:        "test@example.com",
			Name:         "Test User",
			PasswordHash: "hashed_password",
		}

		if err := store.Users().Create(ctx, user); err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		got, err := store.Users().GetByID(ctx, user.ID)
		if err != nil {
			t.Fatalf("GetByID failed: %v", err)
		}

		if got.Email != user.Email {
			t.Errorf("Email mismatch: got %s, want %s", got.Email, user.Email)
		}
	})

	t.Run("GetByEmail case insensitive", func(t *testing.T) {
		user := &users.User{
			ID:           ulid.New(),
			Email:        "CaseTest@Example.Com",
			Name:         "Case Test",
			PasswordHash: "hash",
		}
		store.Users().Create(ctx, user)

		got, err := store.Users().GetByEmail(ctx, "casetest@example.com")
		if err != nil {
			t.Fatalf("GetByEmail failed: %v", err)
		}

		if got.ID != user.ID {
			t.Errorf("ID mismatch: got %s, want %s", got.ID, user.ID)
		}
	})

	t.Run("Update", func(t *testing.T) {
		user := createTestUser(t, store)

		user.Name = "Updated Name"
		if err := store.Users().Update(ctx, user); err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		got, _ := store.Users().GetByID(ctx, user.ID)
		if got.Name != "Updated Name" {
			t.Errorf("Name not updated: got %s", got.Name)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		user := createTestUser(t, store)

		if err := store.Users().Delete(ctx, user.ID); err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		_, err := store.Users().GetByID(ctx, user.ID)
		if err != users.ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})
}

// TestWorkspacesStore tests workspace CRUD and membership.
func TestWorkspacesStore(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()
	owner := createTestUser(t, store)

	t.Run("Create and GetByID", func(t *testing.T) {
		ws := &workspaces.Workspace{
			ID:      ulid.New(),
			Name:    "Test Workspace",
			Slug:    "test-workspace",
			OwnerID: owner.ID,
		}

		if err := store.Workspaces().Create(ctx, ws); err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		got, err := store.Workspaces().GetByID(ctx, ws.ID)
		if err != nil {
			t.Fatalf("GetByID failed: %v", err)
		}

		if got.Name != ws.Name {
			t.Errorf("Name mismatch: got %s, want %s", got.Name, ws.Name)
		}
	})

	t.Run("GetBySlug", func(t *testing.T) {
		ws := createTestWorkspace(t, store, owner)

		got, err := store.Workspaces().GetBySlug(ctx, ws.Slug)
		if err != nil {
			t.Fatalf("GetBySlug failed: %v", err)
		}

		if got.ID != ws.ID {
			t.Errorf("ID mismatch: got %s, want %s", got.ID, ws.ID)
		}
	})

	t.Run("Members", func(t *testing.T) {
		ws := createTestWorkspace(t, store, owner)
		member := createTestUser(t, store)

		// Add member
		err := store.Workspaces().AddMember(ctx, &workspaces.Member{
			WorkspaceID: ws.ID,
			UserID:      member.ID,
			Role:        "member",
		})
		if err != nil {
			t.Fatalf("AddMember failed: %v", err)
		}

		// List members
		members, err := store.Workspaces().ListMembers(ctx, ws.ID)
		if err != nil {
			t.Fatalf("ListMembers failed: %v", err)
		}
		if len(members) != 1 {
			t.Errorf("Expected 1 member, got %d", len(members))
		}

		// Update role
		if err := store.Workspaces().UpdateMemberRole(ctx, ws.ID, member.ID, "admin"); err != nil {
			t.Fatalf("UpdateMemberRole failed: %v", err)
		}

		// Remove member
		if err := store.Workspaces().RemoveMember(ctx, ws.ID, member.ID); err != nil {
			t.Fatalf("RemoveMember failed: %v", err)
		}

		members, _ = store.Workspaces().ListMembers(ctx, ws.ID)
		if len(members) != 0 {
			t.Errorf("Expected 0 members, got %d", len(members))
		}
	})
}

// TestTablesStore tests table CRUD.
func TestTablesStore(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()
	user := createTestUser(t, store)
	ws := createTestWorkspace(t, store, user)
	base := createTestBase(t, store, ws, user)

	t.Run("Create and GetByID", func(t *testing.T) {
		tbl := &tables.Table{
			ID:        ulid.New(),
			BaseID:    base.ID,
			Name:      "Tasks",
			CreatedBy: user.ID,
		}

		if err := store.Tables().Create(ctx, tbl); err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		got, err := store.Tables().GetByID(ctx, tbl.ID)
		if err != nil {
			t.Fatalf("GetByID failed: %v", err)
		}

		if got.Name != tbl.Name {
			t.Errorf("Name mismatch: got %s, want %s", got.Name, tbl.Name)
		}
	})

	t.Run("AutoNumber", func(t *testing.T) {
		tbl := createTestTable(t, store, base, user)

		n1, err := store.Tables().NextAutoNumber(ctx, tbl.ID)
		if err != nil {
			t.Fatalf("NextAutoNumber failed: %v", err)
		}
		if n1 != 1 {
			t.Errorf("Expected 1, got %d", n1)
		}

		n2, _ := store.Tables().NextAutoNumber(ctx, tbl.ID)
		if n2 != 2 {
			t.Errorf("Expected 2, got %d", n2)
		}
	})
}

// TestFieldsStore tests field CRUD and select choices.
func TestFieldsStore(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()
	user := createTestUser(t, store)
	ws := createTestWorkspace(t, store, user)
	base := createTestBase(t, store, ws, user)
	tbl := createTestTable(t, store, base, user)

	t.Run("Create and ListByTable", func(t *testing.T) {
		field := &fields.Field{
			ID:        ulid.New(),
			TableID:   tbl.ID,
			Name:      "Title",
			Type:      fields.TypeSingleLineText,
			CreatedBy: user.ID,
		}

		if err := store.Fields().Create(ctx, field); err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		list, err := store.Fields().ListByTable(ctx, tbl.ID)
		if err != nil {
			t.Fatalf("ListByTable failed: %v", err)
		}
		if len(list) == 0 {
			t.Error("Expected at least 1 field")
		}
	})

	t.Run("SelectChoices", func(t *testing.T) {
		field := createTestField(t, store, tbl, "Status", fields.TypeSingleSelect, user)

		// Add choices
		for _, name := range []string{"Todo", "In Progress", "Done"} {
			err := store.Fields().AddSelectChoice(ctx, &fields.SelectChoice{
				FieldID: field.ID,
				Name:    name,
				Color:   "#6B7280",
			})
			if err != nil {
				t.Fatalf("AddSelectChoice failed: %v", err)
			}
		}

		// List choices
		choices, err := store.Fields().ListSelectChoices(ctx, field.ID)
		if err != nil {
			t.Fatalf("ListSelectChoices failed: %v", err)
		}
		if len(choices) != 3 {
			t.Errorf("Expected 3 choices, got %d", len(choices))
		}

		// Update choice
		err = store.Fields().UpdateSelectChoice(ctx, choices[0].ID, fields.UpdateChoiceIn{Name: "Backlog"})
		if err != nil {
			t.Fatalf("UpdateSelectChoice failed: %v", err)
		}

		// Delete choice
		if err := store.Fields().DeleteSelectChoice(ctx, choices[2].ID); err != nil {
			t.Fatalf("DeleteSelectChoice failed: %v", err)
		}

		choices, _ = store.Fields().ListSelectChoices(ctx, field.ID)
		if len(choices) != 2 {
			t.Errorf("Expected 2 choices after delete, got %d", len(choices))
		}
	})
}

// TestRecordsStore tests record CRUD and cell operations.
func TestRecordsStore(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()
	user := createTestUser(t, store)
	ws := createTestWorkspace(t, store, user)
	base := createTestBase(t, store, ws, user)
	tbl := createTestTable(t, store, base, user)
	titleField := createTestField(t, store, tbl, "Title", fields.TypeSingleLineText, user)

	t.Run("Create and GetByID", func(t *testing.T) {
		rec := &records.Record{
			ID:        ulid.New(),
			TableID:   tbl.ID,
			Cells:     map[string]any{titleField.ID: "Task 1"},
			CreatedBy: user.ID,
		}

		if err := store.Records().Create(ctx, rec); err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		got, err := store.Records().GetByID(ctx, rec.ID)
		if err != nil {
			t.Fatalf("GetByID failed: %v", err)
		}

		if got.Cells[titleField.ID] != "Task 1" {
			t.Errorf("Cell value mismatch: got %v", got.Cells[titleField.ID])
		}
	})

	t.Run("UpdateCell", func(t *testing.T) {
		rec := &records.Record{
			ID:        ulid.New(),
			TableID:   tbl.ID,
			Cells:     map[string]any{titleField.ID: "Original"},
			CreatedBy: user.ID,
		}
		store.Records().Create(ctx, rec)

		if err := store.Records().UpdateCell(ctx, rec.ID, titleField.ID, "Updated"); err != nil {
			t.Fatalf("UpdateCell failed: %v", err)
		}

		got, _ := store.Records().GetByID(ctx, rec.ID)
		if got.Cells[titleField.ID] != "Updated" {
			t.Errorf("Cell not updated: got %v", got.Cells[titleField.ID])
		}
	})

	t.Run("ClearCell", func(t *testing.T) {
		rec := &records.Record{
			ID:        ulid.New(),
			TableID:   tbl.ID,
			Cells:     map[string]any{titleField.ID: "To Clear"},
			CreatedBy: user.ID,
		}
		store.Records().Create(ctx, rec)

		if err := store.Records().ClearCell(ctx, rec.ID, titleField.ID); err != nil {
			t.Fatalf("ClearCell failed: %v", err)
		}

		got, _ := store.Records().GetByID(ctx, rec.ID)
		if _, exists := got.Cells[titleField.ID]; exists {
			t.Errorf("Cell should be cleared")
		}
	})

	t.Run("List with pagination", func(t *testing.T) {
		// Create 10 records
		for i := 0; i < 10; i++ {
			rec := &records.Record{
				ID:        ulid.New(),
				TableID:   tbl.ID,
				Cells:     map[string]any{},
				CreatedBy: user.ID,
			}
			store.Records().Create(ctx, rec)
		}

		list, err := store.Records().List(ctx, tbl.ID, records.ListOpts{Limit: 5})
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if len(list.Records) != 5 {
			t.Errorf("Expected 5 records, got %d", len(list.Records))
		}
		if list.Total < 10 {
			t.Errorf("Expected total >= 10, got %d", list.Total)
		}
	})
}

// TestViewsStore tests view CRUD.
func TestViewsStore(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()
	user := createTestUser(t, store)
	ws := createTestWorkspace(t, store, user)
	base := createTestBase(t, store, ws, user)
	tbl := createTestTable(t, store, base, user)

	t.Run("Create and GetByID", func(t *testing.T) {
		view := &views.View{
			ID:        ulid.New(),
			TableID:   tbl.ID,
			Name:      "All Records",
			Type:      views.TypeGrid,
			IsDefault: true,
			CreatedBy: user.ID,
		}

		if err := store.Views().Create(ctx, view); err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		got, err := store.Views().GetByID(ctx, view.ID)
		if err != nil {
			t.Fatalf("GetByID failed: %v", err)
		}

		if got.Name != view.Name {
			t.Errorf("Name mismatch: got %s, want %s", got.Name, view.Name)
		}
		if !got.IsDefault {
			t.Error("Expected IsDefault to be true")
		}
	})

	t.Run("Update with filters and sorts", func(t *testing.T) {
		view := &views.View{
			ID:        ulid.New(),
			TableID:   tbl.ID,
			Name:      "Filtered View",
			Type:      views.TypeGrid,
			CreatedBy: user.ID,
		}
		store.Views().Create(ctx, view)

		// Update with filters and sorts
		view.Filters = []views.Filter{{FieldID: "fld_test", Operator: "equals", Value: "done"}}
		view.Sorts = []views.SortSpec{{FieldID: "fld_test", Direction: "asc"}}

		if err := store.Views().Update(ctx, view); err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		got, _ := store.Views().GetByID(ctx, view.ID)
		if len(got.Filters) != 1 {
			t.Errorf("Expected 1 filter, got %d", len(got.Filters))
		}
		if len(got.Sorts) != 1 {
			t.Errorf("Expected 1 sort, got %d", len(got.Sorts))
		}
	})
}

// TestCommentsStore tests comment CRUD.
func TestCommentsStore(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()
	user := createTestUser(t, store)
	ws := createTestWorkspace(t, store, user)
	base := createTestBase(t, store, ws, user)
	tbl := createTestTable(t, store, base, user)

	rec := &records.Record{
		ID:        ulid.New(),
		TableID:   tbl.ID,
		Cells:     map[string]any{},
		CreatedBy: user.ID,
	}
	store.Records().Create(ctx, rec)

	t.Run("Create and ListByRecord", func(t *testing.T) {
		comment := &comments.Comment{
			ID:       ulid.New(),
			RecordID: rec.ID,
			UserID:   user.ID,
			Content:  "This is a test comment",
		}

		if err := store.Comments().Create(ctx, comment); err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		list, err := store.Comments().ListByRecord(ctx, rec.ID)
		if err != nil {
			t.Fatalf("ListByRecord failed: %v", err)
		}
		if len(list) != 1 {
			t.Errorf("Expected 1 comment, got %d", len(list))
		}
	})

	t.Run("Resolve comment", func(t *testing.T) {
		comment := &comments.Comment{
			ID:       ulid.New(),
			RecordID: rec.ID,
			UserID:   user.ID,
			Content:  "To be resolved",
		}
		store.Comments().Create(ctx, comment)

		comment.IsResolved = true
		if err := store.Comments().Update(ctx, comment); err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		got, _ := store.Comments().GetByID(ctx, comment.ID)
		if !got.IsResolved {
			t.Error("Comment should be resolved")
		}
	})
}

// TestProjectManagementWorkflow tests a complete project management workflow.
func TestProjectManagementWorkflow(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	// Create team
	owner := createTestUser(t, store)

	// Create workspace
	ws := createTestWorkspace(t, store, owner)

	// Create base
	base := createTestBase(t, store, ws, owner)

	// Create Tasks table
	tasksTable := &tables.Table{
		ID:        ulid.New(),
		BaseID:    base.ID,
		Name:      "Tasks",
		CreatedBy: owner.ID,
	}
	store.Tables().Create(ctx, tasksTable)

	// Create fields
	titleField := createTestField(t, store, tasksTable, "Title", fields.TypeSingleLineText, owner)
	statusField := createTestField(t, store, tasksTable, "Status", fields.TypeSingleSelect, owner)
	priorityField := createTestField(t, store, tasksTable, "Priority", fields.TypeSingleSelect, owner)

	// Add status choices
	for _, status := range []string{"Backlog", "In Progress", "Done"} {
		store.Fields().AddSelectChoice(ctx, &fields.SelectChoice{
			FieldID: statusField.ID,
			Name:    status,
		})
	}

	// Add priority choices
	for _, priority := range []string{"Low", "Medium", "High"} {
		store.Fields().AddSelectChoice(ctx, &fields.SelectChoice{
			FieldID: priorityField.ID,
			Name:    priority,
		})
	}

	// Create tasks
	task1 := &records.Record{
		ID:      ulid.New(),
		TableID: tasksTable.ID,
		Cells: map[string]any{
			titleField.ID:    "Implement user auth",
			statusField.ID:   "Backlog",
			priorityField.ID: "High",
		},
		CreatedBy: owner.ID,
	}
	store.Records().Create(ctx, task1)

	task2 := &records.Record{
		ID:      ulid.New(),
		TableID: tasksTable.ID,
		Cells: map[string]any{
			titleField.ID:    "Add API rate limiting",
			statusField.ID:   "Backlog",
			priorityField.ID: "Medium",
		},
		CreatedBy: owner.ID,
	}
	store.Records().Create(ctx, task2)

	// Create views
	allTasksView := &views.View{
		ID:        ulid.New(),
		TableID:   tasksTable.ID,
		Name:      "All Tasks",
		Type:      views.TypeGrid,
		IsDefault: true,
		CreatedBy: owner.ID,
	}
	store.Views().Create(ctx, allTasksView)

	// Move task to "In Progress"
	store.Records().UpdateCell(ctx, task1.ID, statusField.ID, "In Progress")

	// Verify status update
	updatedTask, _ := store.Records().GetByID(ctx, task1.ID)
	if updatedTask.Cells[statusField.ID] != "In Progress" {
		t.Errorf("Status should be 'In Progress', got %v", updatedTask.Cells[statusField.ID])
	}

	// Add comment
	comment := &comments.Comment{
		ID:       ulid.New(),
		RecordID: task1.ID,
		UserID:   owner.ID,
		Content:  "Started working on this. Will need OAuth2 library.",
	}
	store.Comments().Create(ctx, comment)

	// Verify comment
	commentList, _ := store.Comments().ListByRecord(ctx, task1.ID)
	if len(commentList) != 1 {
		t.Errorf("Expected 1 comment, got %d", len(commentList))
	}

	// List all records
	recordList, _ := store.Records().List(ctx, tasksTable.ID, records.ListOpts{Limit: 100})
	if len(recordList.Records) < 2 {
		t.Errorf("Expected at least 2 records, got %d", len(recordList.Records))
	}

	t.Logf("Project management workflow completed successfully")
}
