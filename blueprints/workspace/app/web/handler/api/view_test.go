package api_test

import (
	"net/http"
	"testing"

	"github.com/go-mizu/blueprints/workspace/feature/databases"
	"github.com/go-mizu/blueprints/workspace/feature/views"
)

// createTestDatabase creates a database for testing views.
func createTestDatabase(ts *TestServer, cookie *http.Cookie, workspaceID, title string) *databases.Database {
	resp := ts.Request("POST", "/api/v1/databases", map[string]interface{}{
		"workspace_id": workspaceID,
		"title":        title,
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var db databases.Database
	ts.ParseJSON(resp, &db)
	return &db
}

func TestViewCreate(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("viewcreate@example.com", "View Create", "password123")
	ws := createTestWorkspace(ts, cookie, "View Workspace", "view-ws")
	db := createTestDatabase(ts, cookie, ws.ID, "View Test Database")

	tests := []struct {
		name       string
		body       map[string]interface{}
		wantStatus int
		wantType   views.ViewType
	}{
		{
			name: "table view",
			body: map[string]interface{}{
				"database_id": db.ID,
				"name":        "Table View",
				"type":        "table",
			},
			wantStatus: http.StatusCreated,
			wantType:   views.ViewTable,
		},
		{
			name: "board view",
			body: map[string]interface{}{
				"database_id": db.ID,
				"name":        "Board View",
				"type":        "board",
				"group_by":    "status",
			},
			wantStatus: http.StatusCreated,
			wantType:   views.ViewBoard,
		},
		{
			name: "list view",
			body: map[string]interface{}{
				"database_id": db.ID,
				"name":        "List View",
				"type":        "list",
			},
			wantStatus: http.StatusCreated,
			wantType:   views.ViewList,
		},
		{
			name: "calendar view",
			body: map[string]interface{}{
				"database_id": db.ID,
				"name":        "Calendar View",
				"type":        "calendar",
				"calendar_by": "due_date",
			},
			wantStatus: http.StatusCreated,
			wantType:   views.ViewCalendar,
		},
		{
			name: "gallery view",
			body: map[string]interface{}{
				"database_id": db.ID,
				"name":        "Gallery View",
				"type":        "gallery",
			},
			wantStatus: http.StatusCreated,
			wantType:   views.ViewGallery,
		},
		{
			name: "timeline view",
			body: map[string]interface{}{
				"database_id": db.ID,
				"name":        "Timeline View",
				"type":        "timeline",
			},
			wantStatus: http.StatusCreated,
			wantType:   views.ViewTimeline,
		},
		{
			name: "view with filter",
			body: map[string]interface{}{
				"database_id": db.ID,
				"name":        "Filtered View",
				"type":        "table",
				"filter": map[string]interface{}{
					"property_id": "status",
					"operator":    "equals",
					"value":       "Done",
				},
			},
			wantStatus: http.StatusCreated,
			wantType:   views.ViewTable,
		},
		{
			name: "view with sorts",
			body: map[string]interface{}{
				"database_id": db.ID,
				"name":        "Sorted View",
				"type":        "table",
				"sorts": []map[string]interface{}{
					{
						"property_id": "created_at",
						"direction":   "desc",
					},
				},
			},
			wantStatus: http.StatusCreated,
			wantType:   views.ViewTable,
		},
		{
			name: "missing database_id",
			body: map[string]interface{}{
				"name": "No Database",
				"type": "table",
			},
			wantStatus: http.StatusCreated, // App allows views without database_id
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := ts.Request("POST", "/api/v1/views", tt.body, cookie)
			ts.ExpectStatus(resp, tt.wantStatus)

			if tt.wantStatus == http.StatusCreated {
				var view views.View
				ts.ParseJSON(resp, &view)

				if view.ID == "" {
					t.Error("view ID should not be empty")
				}
				// Only check type and database_id if expected
				if tt.wantType != "" && view.Type != tt.wantType {
					t.Errorf("type = %q, want %q", view.Type, tt.wantType)
				}
				if tt.body["database_id"] != nil && view.DatabaseID != db.ID {
					t.Errorf("database_id = %q, want %q", view.DatabaseID, db.ID)
				}
			}
			resp.Body.Close()
		})
	}
}

func TestViewGet(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("viewget@example.com", "View Get", "password123")
	ws := createTestWorkspace(ts, cookie, "View Get Workspace", "view-get-ws")
	db := createTestDatabase(ts, cookie, ws.ID, "View Get Database")

	// Create view
	resp := ts.Request("POST", "/api/v1/views", map[string]interface{}{
		"database_id": db.ID,
		"name":        "Get Test View",
		"type":        "table",
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var created views.View
	ts.ParseJSON(resp, &created)

	t.Run("get existing view", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/views/"+created.ID, nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var view views.View
		ts.ParseJSON(resp, &view)

		if view.ID != created.ID {
			t.Errorf("ID = %q, want %q", view.ID, created.ID)
		}
		if view.Name != created.Name {
			t.Errorf("Name = %q, want %q", view.Name, created.Name)
		}
	})

	t.Run("non-existent view", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/views/non-existent-id", nil, cookie)
		ts.ExpectStatus(resp, http.StatusNotFound)
		resp.Body.Close()
	})
}

func TestViewUpdate(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("viewupdate@example.com", "View Update", "password123")
	ws := createTestWorkspace(ts, cookie, "View Update Workspace", "view-update-ws")
	db := createTestDatabase(ts, cookie, ws.ID, "View Update Database")

	// Create view
	resp := ts.Request("POST", "/api/v1/views", map[string]interface{}{
		"database_id": db.ID,
		"name":        "Original View",
		"type":        "table",
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var created views.View
	ts.ParseJSON(resp, &created)

	t.Run("update name", func(t *testing.T) {
		resp := ts.Request("PATCH", "/api/v1/views/"+created.ID, map[string]interface{}{
			"name": "Updated View",
		}, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var updated views.View
		ts.ParseJSON(resp, &updated)

		if updated.Name != "Updated View" {
			t.Errorf("Name = %q, want %q", updated.Name, "Updated View")
		}
	})

	t.Run("update filter", func(t *testing.T) {
		resp := ts.Request("PATCH", "/api/v1/views/"+created.ID, map[string]interface{}{
			"filter": map[string]interface{}{
				"property_id": "status",
				"operator":    "equals",
				"value":       "Active",
			},
		}, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var updated views.View
		ts.ParseJSON(resp, &updated)

		if updated.Filter == nil {
			t.Error("filter should not be nil")
		}
	})

	t.Run("update sorts", func(t *testing.T) {
		resp := ts.Request("PATCH", "/api/v1/views/"+created.ID, map[string]interface{}{
			"sorts": []map[string]interface{}{
				{
					"property_id": "name",
					"direction":   "asc",
				},
			},
		}, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var updated views.View
		ts.ParseJSON(resp, &updated)

		if len(updated.Sorts) == 0 {
			t.Error("sorts should not be empty")
		}
	})
}

func TestViewDelete(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("viewdelete@example.com", "View Delete", "password123")
	ws := createTestWorkspace(ts, cookie, "View Delete Workspace", "view-delete-ws")
	db := createTestDatabase(ts, cookie, ws.ID, "View Delete Database")

	// Create view
	resp := ts.Request("POST", "/api/v1/views", map[string]interface{}{
		"database_id": db.ID,
		"name":        "To Delete",
		"type":        "table",
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var created views.View
	ts.ParseJSON(resp, &created)

	t.Run("delete view", func(t *testing.T) {
		resp := ts.Request("DELETE", "/api/v1/views/"+created.ID, nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)
		resp.Body.Close()
	})

	t.Run("deleted view not found", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/views/"+created.ID, nil, cookie)
		ts.ExpectStatus(resp, http.StatusNotFound)
		resp.Body.Close()
	})
}

func TestViewList(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("viewlist@example.com", "View List", "password123")
	ws := createTestWorkspace(ts, cookie, "View List Workspace", "view-list-ws")
	db := createTestDatabase(ts, cookie, ws.ID, "View List Database")

	// Create views
	viewTypes := []string{"table", "board", "list"}
	for i, vt := range viewTypes {
		resp := ts.Request("POST", "/api/v1/views", map[string]interface{}{
			"database_id": db.ID,
			"name":        "View " + string(rune('A'+i)),
			"type":        vt,
		}, cookie)
		ts.ExpectStatus(resp, http.StatusCreated)
		resp.Body.Close()
	}

	t.Run("list database views", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/databases/"+db.ID+"/views", nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var viewList []*views.View
		ts.ParseJSON(resp, &viewList)

		if len(viewList) < 3 {
			t.Errorf("expected at least 3 views, got %d", len(viewList))
		}
	})
}

func TestViewQuery(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("viewquery@example.com", "View Query", "password123")
	ws := createTestWorkspace(ts, cookie, "View Query Workspace", "view-query-ws")
	db := createTestDatabase(ts, cookie, ws.ID, "View Query Database")

	// Create view
	resp := ts.Request("POST", "/api/v1/views", map[string]interface{}{
		"database_id": db.ID,
		"name":        "Query View",
		"type":        "table",
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var view views.View
	ts.ParseJSON(resp, &view)

	t.Run("basic query", func(t *testing.T) {
		resp := ts.Request("POST", "/api/v1/views/"+view.ID+"/query", map[string]interface{}{}, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var result views.QueryResult
		ts.ParseJSON(resp, &result)

		// Result may be empty (nil or empty slice is acceptable)
		if len(result.Items) != 0 {
			t.Errorf("expected 0 items for empty database, got %d", len(result.Items))
		}
	})

	t.Run("query with pagination", func(t *testing.T) {
		resp := ts.Request("POST", "/api/v1/views/"+view.ID+"/query", map[string]interface{}{
			"limit": 10,
		}, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var result views.QueryResult
		ts.ParseJSON(resp, &result)
		// Just verify the query works
	})
}

func TestViewComplexFilter(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("viewfilter@example.com", "View Filter", "password123")
	ws := createTestWorkspace(ts, cookie, "View Filter Workspace", "view-filter-ws")
	db := createTestDatabase(ts, cookie, ws.ID, "View Filter Database")

	t.Run("AND filter", func(t *testing.T) {
		resp := ts.Request("POST", "/api/v1/views", map[string]interface{}{
			"database_id": db.ID,
			"name":        "AND Filter View",
			"type":        "table",
			"filter": map[string]interface{}{
				"and": []map[string]interface{}{
					{
						"property_id": "status",
						"operator":    "equals",
						"value":       "Active",
					},
					{
						"property_id": "priority",
						"operator":    "equals",
						"value":       "High",
					},
				},
			},
		}, cookie)
		ts.ExpectStatus(resp, http.StatusCreated)

		var view views.View
		ts.ParseJSON(resp, &view)

		if view.Filter == nil || len(view.Filter.And) == 0 {
			t.Error("AND filter should be set")
		}
	})

	t.Run("OR filter", func(t *testing.T) {
		resp := ts.Request("POST", "/api/v1/views", map[string]interface{}{
			"database_id": db.ID,
			"name":        "OR Filter View",
			"type":        "table",
			"filter": map[string]interface{}{
				"or": []map[string]interface{}{
					{
						"property_id": "status",
						"operator":    "equals",
						"value":       "Active",
					},
					{
						"property_id": "status",
						"operator":    "equals",
						"value":       "Pending",
					},
				},
			},
		}, cookie)
		ts.ExpectStatus(resp, http.StatusCreated)

		var view views.View
		ts.ParseJSON(resp, &view)

		if view.Filter == nil || len(view.Filter.Or) == 0 {
			t.Error("OR filter should be set")
		}
	})
}

func TestViewUnauthenticated(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("viewauth@example.com", "View Auth", "password123")
	ws := createTestWorkspace(ts, cookie, "View Auth Workspace", "view-auth-ws")
	db := createTestDatabase(ts, cookie, ws.ID, "View Auth Database")

	// Create view
	resp := ts.Request("POST", "/api/v1/views", map[string]interface{}{
		"database_id": db.ID,
		"name":        "Auth Test View",
		"type":        "table",
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var view views.View
	ts.ParseJSON(resp, &view)

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"create view", "POST", "/api/v1/views"},
		{"get view", "GET", "/api/v1/views/" + view.ID},
		{"update view", "PATCH", "/api/v1/views/" + view.ID},
		{"delete view", "DELETE", "/api/v1/views/" + view.ID},
		{"list views", "GET", "/api/v1/databases/" + db.ID + "/views"},
		{"query view", "POST", "/api/v1/views/" + view.ID + "/query"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := ts.Request(tt.method, tt.path, nil) // No cookie
			ts.ExpectStatus(resp, http.StatusUnauthorized)
			resp.Body.Close()
		})
	}
}
