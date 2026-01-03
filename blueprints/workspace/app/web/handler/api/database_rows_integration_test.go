package api_test

import (
	"net/http"
	"testing"

	"github.com/go-mizu/blueprints/workspace/feature/databases"
	"github.com/go-mizu/blueprints/workspace/feature/rows"
)

// TestDatabaseRowsIntegration tests the full flow of database and row operations
func TestDatabaseRowsIntegration(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Register and setup
	_, cookie := ts.Register("dbrows@example.com", "DB Rows User", "password123")
	ws := createTestWorkspace(ts, cookie, "Rows Workspace", "rows-ws")

	// Create a database with properties
	var db databases.Database
	resp := ts.Request("POST", "/api/v1/databases", map[string]interface{}{
		"workspace_id": ws.ID,
		"page_id":      "page123",
		"title":        "Task Database",
		"properties": []map[string]interface{}{
			{"name": "Title", "type": "title"},
			{"name": "Status", "type": "select", "config": map[string]interface{}{
				"options": []map[string]string{
					{"name": "Todo", "color": "gray"},
					{"name": "In Progress", "color": "blue"},
					{"name": "Done", "color": "green"},
				},
			}},
			{"name": "Priority", "type": "number"},
		},
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)
	ts.ParseJSON(resp, &db)

	if db.ID == "" {
		t.Fatal("expected database ID to be set")
	}

	if len(db.Properties) < 3 {
		t.Errorf("expected at least 3 properties, got %d", len(db.Properties))
	}

	t.Run("create rows", func(t *testing.T) {
		testRows := []map[string]interface{}{
			{"title": "Task 1", "status": "Todo", "priority": 1},
			{"title": "Task 2", "status": "In Progress", "priority": 2},
			{"title": "Task 3", "status": "Done", "priority": 3},
			{"title": "Task 4", "status": "Todo", "priority": 1},
			{"title": "Task 5", "status": "In Progress", "priority": 2},
		}

		for _, props := range testRows {
			resp := ts.Request("POST", "/api/v1/databases/"+db.ID+"/rows", map[string]interface{}{
				"properties": props,
			}, cookie)
			ts.ExpectStatus(resp, http.StatusCreated)

			var row rows.Row
			ts.ParseJSON(resp, &row)

			if row.ID == "" {
				t.Error("expected row ID to be set")
			}
			if row.DatabaseID != db.ID {
				t.Errorf("expected database_id %s, got %s", db.ID, row.DatabaseID)
			}
		}
	})

	t.Run("list all rows", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/databases/"+db.ID+"/rows", nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var result rows.ListResult
		ts.ParseJSON(resp, &result)

		if len(result.Rows) != 5 {
			t.Errorf("expected 5 rows, got %d", len(result.Rows))
		}
	})

	t.Run("list rows with filter", func(t *testing.T) {
		// Filter for Todo status
		filters := `[{"property":"status","operator":"is","value":"Todo"}]`
		resp := ts.Request("GET", "/api/v1/databases/"+db.ID+"/rows?filters="+filters, nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var result rows.ListResult
		ts.ParseJSON(resp, &result)

		if len(result.Rows) != 2 {
			t.Errorf("expected 2 rows with Todo status, got %d", len(result.Rows))
		}
	})

	t.Run("list rows with sort", func(t *testing.T) {
		sorts := `[{"property":"priority","direction":"desc"}]`
		resp := ts.Request("GET", "/api/v1/databases/"+db.ID+"/rows?sorts="+sorts, nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var result rows.ListResult
		ts.ParseJSON(resp, &result)

		if len(result.Rows) != 5 {
			t.Errorf("expected 5 rows, got %d", len(result.Rows))
		}
	})

	var firstRowID string
	t.Run("get single row", func(t *testing.T) {
		// First, get list to find a row ID
		resp := ts.Request("GET", "/api/v1/databases/"+db.ID+"/rows", nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var result rows.ListResult
		ts.ParseJSON(resp, &result)

		if len(result.Rows) == 0 {
			t.Skip("no rows to test")
		}

		firstRowID = result.Rows[0].ID

		// Get single row
		resp = ts.Request("GET", "/api/v1/rows/"+firstRowID, nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var row rows.Row
		ts.ParseJSON(resp, &row)

		if row.ID != firstRowID {
			t.Errorf("expected row ID %s, got %s", firstRowID, row.ID)
		}
	})

	t.Run("update row", func(t *testing.T) {
		if firstRowID == "" {
			t.Skip("no row to update")
		}

		resp := ts.Request("PATCH", "/api/v1/rows/"+firstRowID, map[string]interface{}{
			"properties": map[string]interface{}{
				"status":   "Done",
				"priority": 10,
			},
		}, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var row rows.Row
		ts.ParseJSON(resp, &row)

		if row.Properties["status"] != "Done" {
			t.Errorf("expected status 'Done', got %v", row.Properties["status"])
		}
	})

	t.Run("duplicate row", func(t *testing.T) {
		if firstRowID == "" {
			t.Skip("no row to duplicate")
		}

		resp := ts.Request("POST", "/api/v1/rows/"+firstRowID+"/duplicate", nil, cookie)
		ts.ExpectStatus(resp, http.StatusCreated)

		var row rows.Row
		ts.ParseJSON(resp, &row)

		if row.ID == firstRowID {
			t.Error("duplicated row should have different ID")
		}
		if row.DatabaseID != db.ID {
			t.Errorf("expected database_id %s, got %s", db.ID, row.DatabaseID)
		}
	})

	t.Run("delete row", func(t *testing.T) {
		if firstRowID == "" {
			t.Skip("no row to delete")
		}

		resp := ts.Request("DELETE", "/api/v1/rows/"+firstRowID, nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		// Verify deletion
		resp = ts.Request("GET", "/api/v1/rows/"+firstRowID, nil, cookie)
		ts.ExpectStatus(resp, http.StatusNotFound)
	})

	t.Run("create view for database", func(t *testing.T) {
		// Test creating view via /databases/{id}/views path
		resp := ts.Request("POST", "/api/v1/databases/"+db.ID+"/views", map[string]interface{}{
			"name": "Kanban Board",
			"type": "board",
		}, cookie)
		ts.ExpectStatus(resp, http.StatusCreated)
	})
}
