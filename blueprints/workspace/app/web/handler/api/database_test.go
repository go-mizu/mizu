package api_test

import (
	"net/http"
	"testing"

	"github.com/go-mizu/blueprints/workspace/feature/databases"
)

func TestDatabaseCreate(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("dbcreate@example.com", "DB Create", "password123")
	ws := createTestWorkspace(ts, cookie, "DB Workspace", "db-ws")

	tests := []struct {
		name       string
		body       map[string]interface{}
		wantStatus int
	}{
		{
			name: "basic database",
			body: map[string]interface{}{
				"workspace_id": ws.ID,
				"title":        "My Database",
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "database with icon",
			body: map[string]interface{}{
				"workspace_id": ws.ID,
				"title":        "Database with Icon",
				"icon":         "table",
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "inline database",
			body: map[string]interface{}{
				"workspace_id": ws.ID,
				"title":        "Inline Database",
				"is_inline":    true,
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "database with properties",
			body: map[string]interface{}{
				"workspace_id": ws.ID,
				"title":        "Database with Properties",
				"properties": []map[string]interface{}{
					{
						"id":   "title",
						"name": "Name",
						"type": "title",
					},
					{
						"id":   "status",
						"name": "Status",
						"type": "select",
						"config": map[string]interface{}{
							"options": []map[string]interface{}{
								{"name": "Todo", "color": "gray"},
								{"name": "In Progress", "color": "blue"},
								{"name": "Done", "color": "green"},
							},
						},
					},
				},
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "missing workspace_id",
			body: map[string]interface{}{
				"title": "No Workspace",
			},
			wantStatus: http.StatusCreated, // App allows databases without workspace_id
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := ts.Request("POST", "/api/v1/databases", tt.body, cookie)
			ts.ExpectStatus(resp, tt.wantStatus)

			if tt.wantStatus == http.StatusCreated {
				var db databases.Database
				ts.ParseJSON(resp, &db)

				if db.Title != tt.body["title"] {
					t.Errorf("title = %q, want %q", db.Title, tt.body["title"])
				}
				if db.ID == "" {
					t.Error("database ID should not be empty")
				}
				// Only check workspace_id if it was provided in request
				if tt.body["workspace_id"] != nil && db.WorkspaceID != ws.ID {
					t.Errorf("workspace_id = %q, want %q", db.WorkspaceID, ws.ID)
				}
			}
			resp.Body.Close()
		})
	}
}

func TestDatabaseGet(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("dbget@example.com", "DB Get", "password123")
	ws := createTestWorkspace(ts, cookie, "DB Get Workspace", "db-get-ws")

	// Create database
	resp := ts.Request("POST", "/api/v1/databases", map[string]interface{}{
		"workspace_id": ws.ID,
		"title":        "Get Test Database",
		"icon":         "database",
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var created databases.Database
	ts.ParseJSON(resp, &created)

	t.Run("get existing database", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/databases/"+created.ID, nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var db databases.Database
		ts.ParseJSON(resp, &db)

		if db.ID != created.ID {
			t.Errorf("ID = %q, want %q", db.ID, created.ID)
		}
		if db.Title != created.Title {
			t.Errorf("Title = %q, want %q", db.Title, created.Title)
		}
	})

	t.Run("non-existent database", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/databases/non-existent-id", nil, cookie)
		ts.ExpectStatus(resp, http.StatusNotFound)
		resp.Body.Close()
	})
}

func TestDatabaseUpdate(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("dbupdate@example.com", "DB Update", "password123")
	ws := createTestWorkspace(ts, cookie, "DB Update Workspace", "db-update-ws")

	// Create database
	resp := ts.Request("POST", "/api/v1/databases", map[string]interface{}{
		"workspace_id": ws.ID,
		"title":        "Original Title",
		"icon":         "star",
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var created databases.Database
	ts.ParseJSON(resp, &created)

	t.Run("update title", func(t *testing.T) {
		resp := ts.Request("PATCH", "/api/v1/databases/"+created.ID, map[string]interface{}{
			"title": "Updated Title",
		}, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var updated databases.Database
		ts.ParseJSON(resp, &updated)

		if updated.Title != "Updated Title" {
			t.Errorf("Title = %q, want %q", updated.Title, "Updated Title")
		}
	})

	t.Run("update icon", func(t *testing.T) {
		resp := ts.Request("PATCH", "/api/v1/databases/"+created.ID, map[string]interface{}{
			"icon": "moon",
		}, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var updated databases.Database
		ts.ParseJSON(resp, &updated)

		if updated.Icon != "moon" {
			t.Errorf("Icon = %q, want %q", updated.Icon, "moon")
		}
	})
}

func TestDatabaseDelete(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("dbdelete@example.com", "DB Delete", "password123")
	ws := createTestWorkspace(ts, cookie, "DB Delete Workspace", "db-delete-ws")

	// Create database
	resp := ts.Request("POST", "/api/v1/databases", map[string]interface{}{
		"workspace_id": ws.ID,
		"title":        "To Delete",
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var created databases.Database
	ts.ParseJSON(resp, &created)

	t.Run("delete database", func(t *testing.T) {
		resp := ts.Request("DELETE", "/api/v1/databases/"+created.ID, nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)
		resp.Body.Close()
	})

	t.Run("deleted database not found", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/databases/"+created.ID, nil, cookie)
		ts.ExpectStatus(resp, http.StatusNotFound)
		resp.Body.Close()
	})
}

func TestDatabaseAddProperty(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("dbprop@example.com", "DB Prop", "password123")
	ws := createTestWorkspace(ts, cookie, "DB Prop Workspace", "db-prop-ws")

	// Create database
	resp := ts.Request("POST", "/api/v1/databases", map[string]interface{}{
		"workspace_id": ws.ID,
		"title":        "Property Test",
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var db databases.Database
	ts.ParseJSON(resp, &db)

	propertyTests := []struct {
		name string
		prop map[string]interface{}
	}{
		{
			name: "rich_text property",
			prop: map[string]interface{}{
				"id":   "description",
				"name": "Description",
				"type": "rich_text",
			},
		},
		{
			name: "number property",
			prop: map[string]interface{}{
				"id":   "amount",
				"name": "Amount",
				"type": "number",
				"config": map[string]interface{}{
					"format": "dollar",
				},
			},
		},
		{
			name: "select property",
			prop: map[string]interface{}{
				"id":   "priority",
				"name": "Priority",
				"type": "select",
				"config": map[string]interface{}{
					"options": []map[string]interface{}{
						{"name": "Low", "color": "gray"},
						{"name": "Medium", "color": "yellow"},
						{"name": "High", "color": "red"},
					},
				},
			},
		},
		{
			name: "multi_select property",
			prop: map[string]interface{}{
				"id":   "tags",
				"name": "Tags",
				"type": "multi_select",
				"config": map[string]interface{}{
					"options": []map[string]interface{}{
						{"name": "Bug", "color": "red"},
						{"name": "Feature", "color": "blue"},
						{"name": "Enhancement", "color": "green"},
					},
				},
			},
		},
		{
			name: "date property",
			prop: map[string]interface{}{
				"id":   "due_date",
				"name": "Due Date",
				"type": "date",
			},
		},
		{
			name: "checkbox property",
			prop: map[string]interface{}{
				"id":   "completed",
				"name": "Completed",
				"type": "checkbox",
			},
		},
		{
			name: "url property",
			prop: map[string]interface{}{
				"id":   "website",
				"name": "Website",
				"type": "url",
			},
		},
		{
			name: "email property",
			prop: map[string]interface{}{
				"id":   "email",
				"name": "Email",
				"type": "email",
			},
		},
		{
			name: "phone_number property",
			prop: map[string]interface{}{
				"id":   "phone",
				"name": "Phone",
				"type": "phone_number",
			},
		},
	}

	for _, tt := range propertyTests {
		t.Run(tt.name, func(t *testing.T) {
			resp := ts.Request("POST", "/api/v1/databases/"+db.ID+"/properties", tt.prop, cookie)
			ts.ExpectStatus(resp, http.StatusOK)

			var updated databases.Database
			ts.ParseJSON(resp, &updated)

			// Check that the property was added
			var found bool
			for _, p := range updated.Properties {
				if p.ID == tt.prop["id"] {
					found = true
					if string(p.Type) != tt.prop["type"] {
						t.Errorf("property type = %q, want %q", p.Type, tt.prop["type"])
					}
				}
			}
			if !found {
				t.Errorf("property %q not found", tt.prop["id"])
			}
		})
	}
}

func TestDatabaseUpdateProperty(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("dbupdateprop@example.com", "DB Update Prop", "password123")
	ws := createTestWorkspace(ts, cookie, "DB Update Prop Workspace", "db-update-prop-ws")

	// Create database with property
	resp := ts.Request("POST", "/api/v1/databases", map[string]interface{}{
		"workspace_id": ws.ID,
		"title":        "Update Prop Test",
		"properties": []map[string]interface{}{
			{
				"id":   "status",
				"name": "Status",
				"type": "select",
			},
		},
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var db databases.Database
	ts.ParseJSON(resp, &db)

	t.Run("update property name", func(t *testing.T) {
		resp := ts.Request("PATCH", "/api/v1/databases/"+db.ID+"/properties/status", map[string]interface{}{
			"id":   "status",
			"name": "Updated Status",
			"type": "select",
		}, cookie)
		ts.ExpectStatus(resp, http.StatusOK)
		resp.Body.Close()
	})
}

func TestDatabaseDeleteProperty(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("dbdeleteprop@example.com", "DB Delete Prop", "password123")
	ws := createTestWorkspace(ts, cookie, "DB Delete Prop Workspace", "db-delete-prop-ws")

	// Create database with property
	resp := ts.Request("POST", "/api/v1/databases", map[string]interface{}{
		"workspace_id": ws.ID,
		"title":        "Delete Prop Test",
		"properties": []map[string]interface{}{
			{
				"id":   "to_delete",
				"name": "To Delete",
				"type": "rich_text",
			},
		},
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var db databases.Database
	ts.ParseJSON(resp, &db)

	t.Run("delete property", func(t *testing.T) {
		resp := ts.Request("DELETE", "/api/v1/databases/"+db.ID+"/properties/to_delete", nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)
		resp.Body.Close()
	})
}

func TestDatabaseUnauthenticated(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("dbauth@example.com", "DB Auth", "password123")
	ws := createTestWorkspace(ts, cookie, "DB Auth Workspace", "db-auth-ws")

	// Create database
	resp := ts.Request("POST", "/api/v1/databases", map[string]interface{}{
		"workspace_id": ws.ID,
		"title":        "Auth Test Database",
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var db databases.Database
	ts.ParseJSON(resp, &db)

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"create database", "POST", "/api/v1/databases"},
		{"get database", "GET", "/api/v1/databases/" + db.ID},
		{"update database", "PATCH", "/api/v1/databases/" + db.ID},
		{"delete database", "DELETE", "/api/v1/databases/" + db.ID},
		{"add property", "POST", "/api/v1/databases/" + db.ID + "/properties"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := ts.Request(tt.method, tt.path, nil) // No cookie
			ts.ExpectStatus(resp, http.StatusUnauthorized)
			resp.Body.Close()
		})
	}
}
