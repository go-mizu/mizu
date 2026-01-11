package importexport

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-mizu/blueprints/table/feature/bases"
	"github.com/go-mizu/blueprints/table/feature/fields"
	"github.com/go-mizu/blueprints/table/feature/records"
	"github.com/go-mizu/blueprints/table/feature/tables"
	"github.com/go-mizu/blueprints/table/feature/views"
	"github.com/go-mizu/blueprints/table/store/duckdb"
)

func setupTestServices(t *testing.T) (*Service, *bases.Service, *tables.Service, *fields.Service, *records.Service, *views.Service, func()) {
	t.Helper()

	// Create temp directory for test database
	tempDir, err := os.MkdirTemp("", "importexport-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}

	// Open store
	store, err := duckdb.Open(tempDir)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("open store: %v", err)
	}

	// Create services
	basesSvc := bases.NewService(store.Bases())
	tablesSvc := tables.NewService(store.Tables())
	fieldsSvc := fields.NewService(store.Fields())
	recordsSvc := records.NewService(store.Records())
	viewsSvc := views.NewService(store.Views())
	importexportSvc := NewService(basesSvc, tablesSvc, fieldsSvc, recordsSvc, viewsSvc)

	cleanup := func() {
		store.Close()
		os.RemoveAll(tempDir)
	}

	return importexportSvc, basesSvc, tablesSvc, fieldsSvc, recordsSvc, viewsSvc, cleanup
}

func TestExportMeta(t *testing.T) {
	svc, basesSvc, tablesSvc, fieldsSvc, _, viewsSvc, cleanup := setupTestServices(t)
	defer cleanup()

	ctx := context.Background()
	userID := "test-user-001"

	// Create a base
	base, err := basesSvc.Create(ctx, userID, bases.CreateIn{
		WorkspaceID: "test-workspace-001",
		Name:        "Test Base",
		Color:       "#FF0000",
	})
	if err != nil {
		t.Fatalf("create base: %v", err)
	}

	// Create a table
	table, err := tablesSvc.Create(ctx, userID, tables.CreateIn{
		BaseID: base.ID,
		Name:   "Test Table",
	})
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	// Create fields
	_, err = fieldsSvc.Create(ctx, userID, fields.CreateIn{
		TableID: table.ID,
		Name:    "Name",
		Type:    "single_line_text",
	})
	if err != nil {
		t.Fatalf("create text field: %v", err)
	}

	choicesJSON, _ := json.Marshal(map[string]any{
		"choices": []map[string]any{
			{"id": "choice-1", "name": "Option A", "color": "#FF0000"},
			{"id": "choice-2", "name": "Option B", "color": "#00FF00"},
		},
	})
	_, err = fieldsSvc.Create(ctx, userID, fields.CreateIn{
		TableID: table.ID,
		Name:    "Status",
		Type:    "single_select",
		Options: choicesJSON,
	})
	if err != nil {
		t.Fatalf("create select field: %v", err)
	}

	// Create a view
	_, err = viewsSvc.Create(ctx, userID, views.CreateIn{
		TableID: table.ID,
		Name:    "All Records",
		Type:    "grid",
	})
	if err != nil {
		t.Fatalf("create view: %v", err)
	}

	// Export metadata
	meta, err := svc.ExportMeta(ctx, base.ID)
	if err != nil {
		t.Fatalf("export meta: %v", err)
	}

	// Verify metadata
	if meta.Version != Version {
		t.Errorf("version = %s, want %s", meta.Version, Version)
	}
	if meta.Base.Name != "Test Base" {
		t.Errorf("base name = %s, want Test Base", meta.Base.Name)
	}
	if len(meta.Tables) != 1 {
		t.Errorf("tables count = %d, want 1", len(meta.Tables))
	}
	if meta.Tables[0].Table.Name != "Test Table" {
		t.Errorf("table name = %s, want Test Table", meta.Tables[0].Table.Name)
	}
	if len(meta.Tables[0].Fields) != 2 {
		t.Errorf("fields count = %d, want 2", len(meta.Tables[0].Fields))
	}
	if len(meta.Tables[0].Views) != 1 {
		t.Errorf("views count = %d, want 1", len(meta.Tables[0].Views))
	}
}

func TestImportMeta(t *testing.T) {
	svc, basesSvc, tablesSvc, fieldsSvc, _, viewsSvc, cleanup := setupTestServices(t)
	defer cleanup()

	ctx := context.Background()
	userID := "test-user-001"
	workspaceID := "test-workspace-001"

	// Create test metadata
	meta := &Meta{
		Version: Version,
		Base: bases.Base{
			ID:          "old-base-id",
			WorkspaceID: "old-workspace-id",
			Name:        "Imported Base",
			Color:       "#0000FF",
		},
		Tables: []TableMeta{
			{
				Table: tables.Table{
					ID:     "old-table-id",
					BaseID: "old-base-id",
					Name:   "Imported Table",
				},
				Fields: []*fields.Field{
					{
						ID:      "old-field-1",
						TableID: "old-table-id",
						Name:    "Title",
						Type:    "single_line_text",
					},
					{
						ID:      "old-field-2",
						TableID: "old-table-id",
						Name:    "Category",
						Type:    "single_select",
						Options: json.RawMessage(`{"choices":[{"id":"old-choice-1","name":"Cat A","color":"#FF0000"},{"id":"old-choice-2","name":"Cat B","color":"#00FF00"}]}`),
					},
				},
				Choices: map[string][]*fields.SelectChoice{
					"old-field-2": {
						{ID: "old-choice-1", FieldID: "old-field-2", Name: "Cat A", Color: "#FF0000"},
						{ID: "old-choice-2", FieldID: "old-field-2", Name: "Cat B", Color: "#00FF00"},
					},
				},
				Views: []*views.View{
					{
						ID:      "old-view-id",
						TableID: "old-table-id",
						Name:    "Main View",
						Type:    "grid",
					},
				},
			},
		},
	}

	// Import metadata
	newBase, err := svc.ImportMeta(ctx, workspaceID, userID, meta)
	if err != nil {
		t.Fatalf("import meta: %v", err)
	}

	// Verify new base was created with new ID
	if newBase.ID == "old-base-id" {
		t.Error("base ID should be different from original")
	}
	if newBase.Name != "Imported Base" {
		t.Errorf("base name = %s, want Imported Base", newBase.Name)
	}
	if newBase.WorkspaceID != workspaceID {
		t.Errorf("workspace ID = %s, want %s", newBase.WorkspaceID, workspaceID)
	}

	// Verify tables
	tbls, err := tablesSvc.ListByBase(ctx, newBase.ID)
	if err != nil {
		t.Fatalf("list tables: %v", err)
	}
	if len(tbls) != 1 {
		t.Fatalf("tables count = %d, want 1", len(tbls))
	}
	if tbls[0].ID == "old-table-id" {
		t.Error("table ID should be different from original")
	}
	if tbls[0].Name != "Imported Table" {
		t.Errorf("table name = %s, want Imported Table", tbls[0].Name)
	}

	// Verify fields
	flds, err := fieldsSvc.ListByTable(ctx, tbls[0].ID)
	if err != nil {
		t.Fatalf("list fields: %v", err)
	}
	if len(flds) != 2 {
		t.Errorf("fields count = %d, want 2", len(flds))
	}

	// Verify views
	vws, err := viewsSvc.ListByTable(ctx, tbls[0].ID)
	if err != nil {
		t.Fatalf("list views: %v", err)
	}
	if len(vws) != 1 {
		t.Errorf("views count = %d, want 1", len(vws))
	}

	// Verify we can retrieve via bases service
	fetchedBase, err := basesSvc.GetByID(ctx, newBase.ID)
	if err != nil {
		t.Fatalf("get base by id: %v", err)
	}
	if fetchedBase.Name != "Imported Base" {
		t.Errorf("fetched base name = %s, want Imported Base", fetchedBase.Name)
	}
}

func TestExportImportRoundTrip(t *testing.T) {
	svc, basesSvc, tablesSvc, fieldsSvc, recordsSvc, viewsSvc, cleanup := setupTestServices(t)
	defer cleanup()

	ctx := context.Background()
	userID := "test-user-001"
	workspaceID := "test-workspace-001"

	// Create a base with data
	base, err := basesSvc.Create(ctx, userID, bases.CreateIn{
		WorkspaceID: workspaceID,
		Name:        "Round Trip Base",
		Color:       "#FF00FF",
	})
	if err != nil {
		t.Fatalf("create base: %v", err)
	}

	// Create table
	table, err := tablesSvc.Create(ctx, userID, tables.CreateIn{
		BaseID: base.ID,
		Name:   "Tasks",
	})
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	// Create fields
	nameField, err := fieldsSvc.Create(ctx, userID, fields.CreateIn{
		TableID: table.ID,
		Name:    "Name",
		Type:    "single_line_text",
	})
	if err != nil {
		t.Fatalf("create name field: %v", err)
	}

	choicesJSON, _ := json.Marshal(map[string]any{
		"choices": []map[string]any{
			{"id": "status-1", "name": "Todo", "color": "#6B7280"},
			{"id": "status-2", "name": "Done", "color": "#10B981"},
		},
	})
	statusField, err := fieldsSvc.Create(ctx, userID, fields.CreateIn{
		TableID: table.ID,
		Name:    "Status",
		Type:    "single_select",
		Options: choicesJSON,
	})
	if err != nil {
		t.Fatalf("create status field: %v", err)
	}

	// Get the created choice IDs
	choices, _ := fieldsSvc.ListSelectChoices(ctx, statusField.ID)
	var todoChoiceID, doneChoiceID string
	for _, c := range choices {
		if c.Name == "Todo" {
			todoChoiceID = c.ID
		} else if c.Name == "Done" {
			doneChoiceID = c.ID
		}
	}

	// Create records
	_, err = recordsSvc.CreateBatch(ctx, table.ID, []map[string]interface{}{
		{nameField.ID: "Task 1", statusField.ID: todoChoiceID},
		{nameField.ID: "Task 2", statusField.ID: doneChoiceID},
		{nameField.ID: "Task 3", statusField.ID: todoChoiceID},
	}, userID)
	if err != nil {
		t.Fatalf("create records: %v", err)
	}

	// Create view
	_, err = viewsSvc.Create(ctx, userID, views.CreateIn{
		TableID: table.ID,
		Name:    "All Tasks",
		Type:    "grid",
	})
	if err != nil {
		t.Fatalf("create view: %v", err)
	}

	// Export to directory
	exportDir, err := os.MkdirTemp("", "export-test-*")
	if err != nil {
		t.Fatalf("create export dir: %v", err)
	}
	defer os.RemoveAll(exportDir)

	err = svc.Export(ctx, base.ID, exportDir)
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	// Verify export files exist
	if _, err := os.Stat(filepath.Join(exportDir, "base.json")); os.IsNotExist(err) {
		t.Error("base.json not created")
	}
	if _, err := os.Stat(filepath.Join(exportDir, "data", "tasks.csv")); os.IsNotExist(err) {
		t.Error("tasks.csv not created")
	}

	// Create a new test environment for import
	svc2, basesSvc2, tablesSvc2, fieldsSvc2, recordsSvc2, viewsSvc2, cleanup2 := setupTestServices(t)
	defer cleanup2()

	// Import into new environment
	newBase, err := svc2.Import(ctx, "new-workspace", userID, exportDir)
	if err != nil {
		t.Fatalf("import: %v", err)
	}

	// Verify imported data
	if newBase.Name != "Round Trip Base" {
		t.Errorf("base name = %s, want Round Trip Base", newBase.Name)
	}

	// Verify tables
	tbls, err := tablesSvc2.ListByBase(ctx, newBase.ID)
	if err != nil {
		t.Fatalf("list tables: %v", err)
	}
	if len(tbls) != 1 {
		t.Fatalf("tables count = %d, want 1", len(tbls))
	}
	if tbls[0].Name != "Tasks" {
		t.Errorf("table name = %s, want Tasks", tbls[0].Name)
	}

	// Verify fields
	flds, err := fieldsSvc2.ListByTable(ctx, tbls[0].ID)
	if err != nil {
		t.Fatalf("list fields: %v", err)
	}
	if len(flds) != 2 {
		t.Errorf("fields count = %d, want 2", len(flds))
	}

	// Verify records
	recs, err := recordsSvc2.List(ctx, tbls[0].ID, records.ListOpts{})
	if err != nil {
		t.Fatalf("list records: %v", err)
	}
	if recs.Total != 3 {
		t.Errorf("records count = %d, want 3", recs.Total)
	}

	// Verify views
	vws, err := viewsSvc2.ListByTable(ctx, tbls[0].ID)
	if err != nil {
		t.Fatalf("list views: %v", err)
	}
	if len(vws) != 1 {
		t.Errorf("views count = %d, want 1", len(vws))
	}

	// Suppress unused variable warnings
	_ = basesSvc2
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Tasks", "tasks"},
		{"Team Members", "team_members"},
		{"Project's Data", "projects_data"},
		{"Data (2024)", "data_2024"},
		{"Test/File", "testfile"},
		{"UPPERCASE", "uppercase"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatAndParseCellValues(t *testing.T) {
	svc := &Service{}

	// Test checkbox
	t.Run("checkbox", func(t *testing.T) {
		fld := &fields.Field{Type: "checkbox"}
		formatted := svc.formatCellValue(true, fld, nil)
		if formatted != "true" {
			t.Errorf("format checkbox true = %q, want true", formatted)
		}

		parsed := svc.parseCellValue("true", fld, nil)
		if parsed != true {
			t.Errorf("parse checkbox true = %v, want true", parsed)
		}

		parsed = svc.parseCellValue("false", fld, nil)
		if parsed != false {
			t.Errorf("parse checkbox false = %v, want false", parsed)
		}
	})

	// Test number
	t.Run("number", func(t *testing.T) {
		fld := &fields.Field{Type: "number"}
		formatted := svc.formatCellValue(float64(42), fld, nil)
		if formatted != "42" {
			t.Errorf("format number = %q, want 42", formatted)
		}

		parsed := svc.parseCellValue("42", fld, nil)
		if parsed != float64(42) {
			t.Errorf("parse number = %v, want 42", parsed)
		}
	})

	// Test single_select
	t.Run("single_select", func(t *testing.T) {
		fld := &fields.Field{Type: "single_select"}
		choiceNames := map[string]string{
			"choice-1": "Option A",
			"choice-2": "Option B",
		}
		choiceIDs := map[string]string{
			"Option A": "new-choice-1",
			"Option B": "new-choice-2",
		}

		formatted := svc.formatCellValue("choice-1", fld, choiceNames)
		if formatted != "Option A" {
			t.Errorf("format single_select = %q, want Option A", formatted)
		}

		parsed := svc.parseCellValue("Option A", fld, choiceIDs)
		if parsed != "new-choice-1" {
			t.Errorf("parse single_select = %v, want new-choice-1", parsed)
		}
	})

	// Test multi_select
	t.Run("multi_select", func(t *testing.T) {
		fld := &fields.Field{Type: "multi_select"}
		choiceNames := map[string]string{
			"tag-1": "Frontend",
			"tag-2": "Backend",
		}
		choiceIDs := map[string]string{
			"Frontend": "new-tag-1",
			"Backend":  "new-tag-2",
		}

		formatted := svc.formatCellValue([]interface{}{"tag-1", "tag-2"}, fld, choiceNames)
		if formatted != "Frontend,Backend" {
			t.Errorf("format multi_select = %q, want Frontend,Backend", formatted)
		}

		parsed := svc.parseCellValue("Frontend,Backend", fld, choiceIDs)
		if parsedSlice, ok := parsed.([]string); ok {
			if len(parsedSlice) != 2 {
				t.Errorf("parse multi_select length = %d, want 2", len(parsedSlice))
			}
		} else {
			t.Errorf("parse multi_select type = %T, want []string", parsed)
		}
	})

	// Test empty value
	t.Run("empty", func(t *testing.T) {
		fld := &fields.Field{Type: "single_line_text"}
		formatted := svc.formatCellValue(nil, fld, nil)
		if formatted != "" {
			t.Errorf("format nil = %q, want empty", formatted)
		}

		parsed := svc.parseCellValue("", fld, nil)
		if parsed != nil {
			t.Errorf("parse empty = %v, want nil", parsed)
		}
	})
}

func TestImportNonExistentDirectory(t *testing.T) {
	svc, _, _, _, _, _, cleanup := setupTestServices(t)
	defer cleanup()

	ctx := context.Background()
	_, err := svc.Import(ctx, "workspace-id", "user-id", "/nonexistent/path")
	if err != ErrDirNotExist {
		t.Errorf("error = %v, want ErrDirNotExist", err)
	}
}

func TestExportNonExistentBase(t *testing.T) {
	svc, _, _, _, _, _, cleanup := setupTestServices(t)
	defer cleanup()

	ctx := context.Background()
	_, err := svc.ExportMeta(ctx, "nonexistent-base-id")
	if err != ErrBaseNotFound {
		t.Errorf("error = %v, want ErrBaseNotFound", err)
	}
}
