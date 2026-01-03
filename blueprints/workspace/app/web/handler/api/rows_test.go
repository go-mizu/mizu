package api_test

import (
	"net/http"
	"testing"

	"github.com/go-mizu/blueprints/workspace/feature/databases"
	"github.com/go-mizu/blueprints/workspace/feature/rows"
)

// createTestRow creates a row in a database and returns it.
func createTestRow(ts *TestServer, cookie *http.Cookie, dbID string, props map[string]interface{}) *rows.Row {
	resp := ts.Request("POST", "/api/v1/databases/"+dbID+"/rows", map[string]interface{}{
		"properties": props,
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var row rows.Row
	ts.ParseJSON(resp, &row)
	return &row
}

func TestRowHandler_Create(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("rowcreate@example.com", "Row Create", "password123")
	ws := createTestWorkspace(ts, cookie, "Row Workspace", "row-ws")
	db := createTestDatabase(ts, cookie, ws.ID, "Row Test Database")

	tests := []struct {
		name       string
		props      map[string]interface{}
		wantStatus int
	}{
		{
			name:       "basic row",
			props:      map[string]interface{}{"title": "Test Row", "status": "active"},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "row with multiple properties",
			props:      map[string]interface{}{"title": "Multi Prop", "count": 42, "done": true},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "empty properties",
			props:      map[string]interface{}{},
			wantStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := ts.Request("POST", "/api/v1/databases/"+db.ID+"/rows", map[string]interface{}{
				"properties": tt.props,
			}, cookie)
			ts.ExpectStatus(resp, tt.wantStatus)

			if tt.wantStatus == http.StatusCreated {
				var row rows.Row
				ts.ParseJSON(resp, &row)

				if row.ID == "" {
					t.Error("row ID should not be empty")
				}
				if row.DatabaseID != db.ID {
					t.Errorf("database_id = %q, want %q", row.DatabaseID, db.ID)
				}
			} else {
				resp.Body.Close()
			}
		})
	}
}

func TestRowHandler_Get(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("rowget@example.com", "Row Get", "password123")
	ws := createTestWorkspace(ts, cookie, "Row Get Workspace", "row-get-ws")
	db := createTestDatabase(ts, cookie, ws.ID, "Row Get Database")
	row := createTestRow(ts, cookie, db.ID, map[string]interface{}{"title": "Test Row"})

	t.Run("get existing row", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/rows/"+row.ID, nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var fetched rows.Row
		ts.ParseJSON(resp, &fetched)

		if fetched.ID != row.ID {
			t.Errorf("id = %q, want %q", fetched.ID, row.ID)
		}
	})

	t.Run("get non-existent row", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/rows/non-existent-id", nil, cookie)
		ts.ExpectStatus(resp, http.StatusNotFound)
		resp.Body.Close()
	})
}

func TestRowHandler_Update(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("rowupdate@example.com", "Row Update", "password123")
	ws := createTestWorkspace(ts, cookie, "Row Update Workspace", "row-update-ws")
	db := createTestDatabase(ts, cookie, ws.ID, "Row Update Database")
	row := createTestRow(ts, cookie, db.ID, map[string]interface{}{"title": "Original", "status": "todo"})

	t.Run("update properties", func(t *testing.T) {
		resp := ts.Request("PATCH", "/api/v1/rows/"+row.ID, map[string]interface{}{
			"properties": map[string]interface{}{"title": "Updated", "status": "done"},
		}, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var updated rows.Row
		ts.ParseJSON(resp, &updated)

		if updated.Properties["title"] != "Updated" {
			t.Errorf("title = %v, want 'Updated'", updated.Properties["title"])
		}
		if updated.Properties["status"] != "done" {
			t.Errorf("status = %v, want 'done'", updated.Properties["status"])
		}
	})

	t.Run("update non-existent row", func(t *testing.T) {
		resp := ts.Request("PATCH", "/api/v1/rows/non-existent-id", map[string]interface{}{
			"properties": map[string]interface{}{"title": "Updated"},
		}, cookie)
		ts.ExpectStatus(resp, http.StatusInternalServerError)
		resp.Body.Close()
	})
}

func TestRowHandler_Delete(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("rowdelete@example.com", "Row Delete", "password123")
	ws := createTestWorkspace(ts, cookie, "Row Delete Workspace", "row-delete-ws")
	db := createTestDatabase(ts, cookie, ws.ID, "Row Delete Database")
	row := createTestRow(ts, cookie, db.ID, map[string]interface{}{"title": "To Delete"})

	t.Run("delete row", func(t *testing.T) {
		resp := ts.Request("DELETE", "/api/v1/rows/"+row.ID, nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)
		resp.Body.Close()

		// Verify row is gone
		resp = ts.Request("GET", "/api/v1/rows/"+row.ID, nil, cookie)
		ts.ExpectStatus(resp, http.StatusNotFound)
		resp.Body.Close()
	})
}

func TestRowHandler_List(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("rowlist@example.com", "Row List", "password123")
	ws := createTestWorkspace(ts, cookie, "Row List Workspace", "row-list-ws")
	db := createTestDatabase(ts, cookie, ws.ID, "Row List Database")

	// Create multiple rows
	for i := 0; i < 5; i++ {
		createTestRow(ts, cookie, db.ID, map[string]interface{}{"title": "Row " + string(rune('A'+i))})
	}

	t.Run("list all rows", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/databases/"+db.ID+"/rows", nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var result rows.ListResult
		ts.ParseJSON(resp, &result)

		if len(result.Rows) != 5 {
			t.Errorf("expected 5 rows, got %d", len(result.Rows))
		}
	})
}

func TestRowHandler_ListWithFilters(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("rowfilter@example.com", "Row Filter", "password123")
	ws := createTestWorkspace(ts, cookie, "Row Filter Workspace", "row-filter-ws")
	db := createTestDatabase(ts, cookie, ws.ID, "Row Filter Database")

	// Create rows with different statuses
	createTestRow(ts, cookie, db.ID, map[string]interface{}{"title": "Row 1", "status": "active"})
	createTestRow(ts, cookie, db.ID, map[string]interface{}{"title": "Row 2", "status": "done"})
	createTestRow(ts, cookie, db.ID, map[string]interface{}{"title": "Row 3", "status": "active"})

	t.Run("filter by status", func(t *testing.T) {
		filters := `[{"property":"status","operator":"is","value":"active"}]`
		resp := ts.Request("GET", "/api/v1/databases/"+db.ID+"/rows?filters="+filters, nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var result rows.ListResult
		ts.ParseJSON(resp, &result)

		if len(result.Rows) != 2 {
			t.Errorf("expected 2 rows with status=active, got %d", len(result.Rows))
		}
	})
}

func TestRowHandler_Duplicate(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("rowdup@example.com", "Row Dup", "password123")
	ws := createTestWorkspace(ts, cookie, "Row Dup Workspace", "row-dup-ws")
	db := createTestDatabase(ts, cookie, ws.ID, "Row Dup Database")
	row := createTestRow(ts, cookie, db.ID, map[string]interface{}{"title": "Original", "count": 42})

	t.Run("duplicate row", func(t *testing.T) {
		resp := ts.Request("POST", "/api/v1/rows/"+row.ID+"/duplicate", nil, cookie)
		ts.ExpectStatus(resp, http.StatusCreated)

		var dup rows.Row
		ts.ParseJSON(resp, &dup)

		if dup.ID == row.ID {
			t.Error("duplicated row should have different ID")
		}
		if dup.DatabaseID != row.DatabaseID {
			t.Errorf("database_id = %q, want %q", dup.DatabaseID, row.DatabaseID)
		}
	})
}

// Tests for all property types to ensure data persistence

func TestRowHandler_PropertyTypes_Text(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("rowtext@example.com", "Row Text", "password123")
	ws := createTestWorkspace(ts, cookie, "Row Text Workspace", "row-text-ws")
	db := createTestDatabase(ts, cookie, ws.ID, "Row Text Database")

	// Create with text properties
	row := createTestRow(ts, cookie, db.ID, map[string]interface{}{
		"name":        "John Doe",
		"description": "A long text description",
	})

	if row.Properties["name"] != "John Doe" {
		t.Errorf("expected name 'John Doe', got %v", row.Properties["name"])
	}

	// Update text property
	resp := ts.Request("PATCH", "/api/v1/rows/"+row.ID, map[string]interface{}{
		"properties": map[string]interface{}{"name": "Jane Doe"},
	}, cookie)
	ts.ExpectStatus(resp, http.StatusOK)

	var updated rows.Row
	ts.ParseJSON(resp, &updated)

	if updated.Properties["name"] != "Jane Doe" {
		t.Errorf("expected name 'Jane Doe', got %v", updated.Properties["name"])
	}
}

func TestRowHandler_PropertyTypes_Number(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("rownum@example.com", "Row Num", "password123")
	ws := createTestWorkspace(ts, cookie, "Row Num Workspace", "row-num-ws")
	db := createTestDatabase(ts, cookie, ws.ID, "Row Num Database")

	// Create with number properties
	row := createTestRow(ts, cookie, db.ID, map[string]interface{}{
		"age":   25,
		"price": 99.99,
		"count": 100,
	})

	if row.Properties["age"] != float64(25) {
		t.Errorf("expected age 25, got %v", row.Properties["age"])
	}
	if row.Properties["price"] != float64(99.99) {
		t.Errorf("expected price 99.99, got %v", row.Properties["price"])
	}

	// Update number property
	resp := ts.Request("PATCH", "/api/v1/rows/"+row.ID, map[string]interface{}{
		"properties": map[string]interface{}{"age": 30},
	}, cookie)
	ts.ExpectStatus(resp, http.StatusOK)
	resp.Body.Close()
}

func TestRowHandler_PropertyTypes_Checkbox(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("rowcheck@example.com", "Row Check", "password123")
	ws := createTestWorkspace(ts, cookie, "Row Check Workspace", "row-check-ws")
	db := createTestDatabase(ts, cookie, ws.ID, "Row Check Database")

	// Create with checkbox property
	row := createTestRow(ts, cookie, db.ID, map[string]interface{}{
		"completed": true,
		"archived":  false,
	})

	if row.Properties["completed"] != true {
		t.Errorf("expected completed true, got %v", row.Properties["completed"])
	}
	if row.Properties["archived"] != false {
		t.Errorf("expected archived false, got %v", row.Properties["archived"])
	}

	// Toggle checkbox
	resp := ts.Request("PATCH", "/api/v1/rows/"+row.ID, map[string]interface{}{
		"properties": map[string]interface{}{"completed": false},
	}, cookie)
	ts.ExpectStatus(resp, http.StatusOK)

	var updated rows.Row
	ts.ParseJSON(resp, &updated)

	if updated.Properties["completed"] != false {
		t.Errorf("expected completed false, got %v", updated.Properties["completed"])
	}
}

func TestRowHandler_PropertyTypes_Date(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("rowdate@example.com", "Row Date", "password123")
	ws := createTestWorkspace(ts, cookie, "Row Date Workspace", "row-date-ws")
	db := createTestDatabase(ts, cookie, ws.ID, "Row Date Database")

	// Create with date property
	row := createTestRow(ts, cookie, db.ID, map[string]interface{}{
		"due_date":   "2025-12-31T23:59:59Z",
		"created_at": "2025-01-01T00:00:00Z",
	})

	if row.Properties["due_date"] != "2025-12-31T23:59:59Z" {
		t.Errorf("expected due_date '2025-12-31T23:59:59Z', got %v", row.Properties["due_date"])
	}

	// Update date
	resp := ts.Request("PATCH", "/api/v1/rows/"+row.ID, map[string]interface{}{
		"properties": map[string]interface{}{"due_date": "2026-01-15T12:00:00Z"},
	}, cookie)
	ts.ExpectStatus(resp, http.StatusOK)
	resp.Body.Close()
}

func TestRowHandler_PropertyTypes_Select(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("rowselect@example.com", "Row Select", "password123")
	ws := createTestWorkspace(ts, cookie, "Row Select Workspace", "row-select-ws")
	db := createTestDatabase(ts, cookie, ws.ID, "Row Select Database")

	// Create with select property
	row := createTestRow(ts, cookie, db.ID, map[string]interface{}{
		"status":   "opt_active",
		"priority": "opt_high",
	})

	if row.Properties["status"] != "opt_active" {
		t.Errorf("expected status 'opt_active', got %v", row.Properties["status"])
	}

	// Change select value
	resp := ts.Request("PATCH", "/api/v1/rows/"+row.ID, map[string]interface{}{
		"properties": map[string]interface{}{"status": "opt_done"},
	}, cookie)
	ts.ExpectStatus(resp, http.StatusOK)

	var updated rows.Row
	ts.ParseJSON(resp, &updated)

	if updated.Properties["status"] != "opt_done" {
		t.Errorf("expected status 'opt_done', got %v", updated.Properties["status"])
	}
}

func TestRowHandler_PropertyTypes_MultiSelect(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("rowmulti@example.com", "Row Multi", "password123")
	ws := createTestWorkspace(ts, cookie, "Row Multi Workspace", "row-multi-ws")
	db := createTestDatabase(ts, cookie, ws.ID, "Row Multi Database")

	// Create with multi-select property
	row := createTestRow(ts, cookie, db.ID, map[string]interface{}{
		"tags": []string{"tag_work", "tag_urgent"},
	})

	tags, ok := row.Properties["tags"].([]interface{})
	if !ok {
		t.Fatalf("expected tags to be array, got %T", row.Properties["tags"])
	}
	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags))
	}

	// Update multi-select
	resp := ts.Request("PATCH", "/api/v1/rows/"+row.ID, map[string]interface{}{
		"properties": map[string]interface{}{"tags": []string{"tag_work", "tag_personal", "tag_important"}},
	}, cookie)
	ts.ExpectStatus(resp, http.StatusOK)

	var updated rows.Row
	ts.ParseJSON(resp, &updated)

	updatedTags, _ := updated.Properties["tags"].([]interface{})
	if len(updatedTags) != 3 {
		t.Errorf("expected 3 tags after update, got %d", len(updatedTags))
	}
}

func TestRowHandler_PropertyTypes_URL(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("rowurl@example.com", "Row URL", "password123")
	ws := createTestWorkspace(ts, cookie, "Row URL Workspace", "row-url-ws")
	db := createTestDatabase(ts, cookie, ws.ID, "Row URL Database")

	// Create with URL property
	row := createTestRow(ts, cookie, db.ID, map[string]interface{}{
		"website": "https://example.com",
		"docs":    "https://docs.example.com/api",
	})

	if row.Properties["website"] != "https://example.com" {
		t.Errorf("expected website 'https://example.com', got %v", row.Properties["website"])
	}

	// Update URL
	resp := ts.Request("PATCH", "/api/v1/rows/"+row.ID, map[string]interface{}{
		"properties": map[string]interface{}{"website": "https://new-example.com"},
	}, cookie)
	ts.ExpectStatus(resp, http.StatusOK)
	resp.Body.Close()
}

func TestRowHandler_PropertyTypes_Email(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("rowemail@example.com", "Row Email", "password123")
	ws := createTestWorkspace(ts, cookie, "Row Email Workspace", "row-email-ws")
	db := createTestDatabase(ts, cookie, ws.ID, "Row Email Database")

	// Create with email property
	row := createTestRow(ts, cookie, db.ID, map[string]interface{}{
		"email":   "user@example.com",
		"contact": "support@company.com",
	})

	if row.Properties["email"] != "user@example.com" {
		t.Errorf("expected email 'user@example.com', got %v", row.Properties["email"])
	}
}

func TestRowHandler_PropertyTypes_Phone(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("rowphone@example.com", "Row Phone", "password123")
	ws := createTestWorkspace(ts, cookie, "Row Phone Workspace", "row-phone-ws")
	db := createTestDatabase(ts, cookie, ws.ID, "Row Phone Database")

	// Create with phone property
	row := createTestRow(ts, cookie, db.ID, map[string]interface{}{
		"phone": "+1-555-123-4567",
		"fax":   "+1-555-987-6543",
	})

	if row.Properties["phone"] != "+1-555-123-4567" {
		t.Errorf("expected phone '+1-555-123-4567', got %v", row.Properties["phone"])
	}
}

func TestRowHandler_PropertyTypes_AllTypesIntegration(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("rowallprop@example.com", "Row All Props", "password123")
	ws := createTestWorkspace(ts, cookie, "Row All Workspace", "row-all-ws")
	db := createTestDatabase(ts, cookie, ws.ID, "Row All Database")

	// Create row with all property types
	row := createTestRow(ts, cookie, db.ID, map[string]interface{}{
		"title":     "Test Item",
		"count":     42,
		"completed": true,
		"due_date":  "2025-12-31T00:00:00Z",
		"status":    "opt_active",
		"tags":      []string{"tag_a", "tag_b"},
		"website":   "https://example.com",
		"email":     "test@example.com",
		"phone":     "+1-555-000-0000",
	})

	// Verify GET returns same data
	resp := ts.Request("GET", "/api/v1/rows/"+row.ID, nil, cookie)
	ts.ExpectStatus(resp, http.StatusOK)

	var fetched rows.Row
	ts.ParseJSON(resp, &fetched)

	if fetched.Properties["title"] != "Test Item" {
		t.Errorf("title mismatch after fetch")
	}
	if fetched.Properties["count"] != float64(42) {
		t.Errorf("count mismatch after fetch")
	}
	if fetched.Properties["completed"] != true {
		t.Errorf("completed mismatch after fetch")
	}
	if fetched.Properties["status"] != "opt_active" {
		t.Errorf("status mismatch after fetch")
	}

	// Update multiple properties at once
	resp = ts.Request("PATCH", "/api/v1/rows/"+row.ID, map[string]interface{}{
		"properties": map[string]interface{}{
			"title":     "Updated Item",
			"count":     100,
			"completed": false,
		},
	}, cookie)
	ts.ExpectStatus(resp, http.StatusOK)

	var updated rows.Row
	ts.ParseJSON(resp, &updated)

	if updated.Properties["title"] != "Updated Item" {
		t.Errorf("title not updated")
	}
	if updated.Properties["count"] != float64(100) {
		t.Errorf("count not updated")
	}
	if updated.Properties["completed"] != false {
		t.Errorf("completed not updated")
	}
}

func TestRowHandler_Unauthenticated(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup - create data with authenticated user
	_, cookie := ts.Register("rowauth@example.com", "Row Auth", "password123")
	ws := createTestWorkspace(ts, cookie, "Row Auth Workspace", "row-auth-ws")
	db := createTestDatabase(ts, cookie, ws.ID, "Row Auth Database")
	row := createTestRow(ts, cookie, db.ID, map[string]interface{}{"title": "Test"})

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"create row", "POST", "/api/v1/databases/" + db.ID + "/rows"},
		{"get row", "GET", "/api/v1/rows/" + row.ID},
		{"update row", "PATCH", "/api/v1/rows/" + row.ID},
		{"delete row", "DELETE", "/api/v1/rows/" + row.ID},
		{"list rows", "GET", "/api/v1/databases/" + db.ID + "/rows"},
		{"duplicate row", "POST", "/api/v1/rows/" + row.ID + "/duplicate"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := ts.Request(tt.method, tt.path, nil) // No cookie
			ts.ExpectStatus(resp, http.StatusUnauthorized)
			resp.Body.Close()
		})
	}
}

// Row Comments Tests

func TestRowHandler_Comments(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("rowcomment@example.com", "Row Comment", "password123")
	ws := createTestWorkspace(ts, cookie, "Row Comment Workspace", "row-comment-ws")
	db := createTestDatabase(ts, cookie, ws.ID, "Row Comment Database")
	row := createTestRow(ts, cookie, db.ID, map[string]interface{}{"title": "Commentable Row"})

	t.Run("create comment on row", func(t *testing.T) {
		resp := ts.Request("POST", "/api/v1/rows/"+row.ID+"/comments", map[string]interface{}{
			"content": []map[string]interface{}{
				{"type": "text", "text": "This is a comment on a row"},
			},
		}, cookie)
		ts.ExpectStatus(resp, http.StatusCreated)
		resp.Body.Close()
	})

	t.Run("list row comments", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/rows/"+row.ID+"/comments", nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)
		resp.Body.Close()
	})
}

// Row Blocks Tests

func TestRowHandler_Blocks(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("rowblock@example.com", "Row Block", "password123")
	ws := createTestWorkspace(ts, cookie, "Row Block Workspace", "row-block-ws")
	db := createTestDatabase(ts, cookie, ws.ID, "Row Block Database")
	row := createTestRow(ts, cookie, db.ID, map[string]interface{}{"title": "Row with Blocks"})

	t.Run("create block in row", func(t *testing.T) {
		resp := ts.Request("POST", "/api/v1/rows/"+row.ID+"/blocks", map[string]interface{}{
			"type": "paragraph",
			"content": map[string]interface{}{
				"rich_text": []map[string]interface{}{
					{"type": "text", "text": "Block content"},
				},
			},
		}, cookie)
		ts.ExpectStatus(resp, http.StatusCreated)
		resp.Body.Close()
	})

	t.Run("list row blocks", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/rows/"+row.ID+"/blocks", nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)
		resp.Body.Close()
	})
}

// Helper to create database with properties for filter tests
func createDatabaseWithProperties(ts *TestServer, cookie *http.Cookie, wsID, title string) *databases.Database {
	resp := ts.Request("POST", "/api/v1/databases", map[string]interface{}{
		"workspace_id": wsID,
		"title":        title,
		"properties": []map[string]interface{}{
			{"id": "title", "name": "Title", "type": "title"},
			{"id": "status", "name": "Status", "type": "select"},
			{"id": "priority", "name": "Priority", "type": "number"},
		},
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var db databases.Database
	ts.ParseJSON(resp, &db)
	return &db
}
