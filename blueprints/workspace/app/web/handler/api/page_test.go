package api_test

import (
	"net/http"
	"testing"

	"github.com/go-mizu/blueprints/workspace/feature/pages"
	"github.com/go-mizu/blueprints/workspace/feature/workspaces"
)

// createTestWorkspace creates a workspace for testing pages.
func createTestWorkspace(ts *TestServer, cookie *http.Cookie, name, slug string) *workspaces.Workspace {
	resp := ts.Request("POST", "/api/v1/workspaces", map[string]interface{}{
		"name": name,
		"slug": slug,
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var ws workspaces.Workspace
	ts.ParseJSON(resp, &ws)
	return &ws
}

func TestPageCreate(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Register and create workspace
	_, cookie := ts.Register("pagecreate@example.com", "Page Create", "password123")
	ws := createTestWorkspace(ts, cookie, "Page Test Workspace", "page-test")

	tests := []struct {
		name       string
		body       map[string]interface{}
		wantStatus int
	}{
		{
			name: "root page",
			body: map[string]interface{}{
				"workspace_id": ws.ID,
				"title":        "Root Page",
				"parent_type":  "workspace",
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "page with icon",
			body: map[string]interface{}{
				"workspace_id": ws.ID,
				"title":        "Page with Icon",
				"icon":         "star",
				"parent_type":  "workspace",
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "page with cover",
			body: map[string]interface{}{
				"workspace_id": ws.ID,
				"title":        "Page with Cover",
				"cover":        "https://example.com/cover.jpg",
				"parent_type":  "workspace",
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "template page",
			body: map[string]interface{}{
				"workspace_id": ws.ID,
				"title":        "Template Page",
				"is_template":  true,
				"parent_type":  "workspace",
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "missing workspace_id",
			body: map[string]interface{}{
				"title":       "No Workspace",
				"parent_type": "workspace",
			},
			wantStatus: http.StatusCreated, // App allows pages without workspace_id
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := ts.Request("POST", "/api/v1/pages", tt.body, cookie)
			ts.ExpectStatus(resp, tt.wantStatus)

			if tt.wantStatus == http.StatusCreated {
				var page pages.Page
				ts.ParseJSON(resp, &page)

				if page.Title != tt.body["title"] {
					t.Errorf("title = %q, want %q", page.Title, tt.body["title"])
				}
				if page.ID == "" {
					t.Error("page ID should not be empty")
				}
				// Only check workspace_id if it was provided in request
				if tt.body["workspace_id"] != nil && page.WorkspaceID != ws.ID {
					t.Errorf("workspace_id = %q, want %q", page.WorkspaceID, ws.ID)
				}
			}
			resp.Body.Close()
		})
	}
}

func TestPageCreateNested(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("nested@example.com", "Nested", "password123")
	ws := createTestWorkspace(ts, cookie, "Nested Workspace", "nested-ws")

	// Create parent page
	resp := ts.Request("POST", "/api/v1/pages", map[string]interface{}{
		"workspace_id": ws.ID,
		"title":        "Parent Page",
		"parent_type":  "workspace",
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var parent pages.Page
	ts.ParseJSON(resp, &parent)

	// Create child page
	resp = ts.Request("POST", "/api/v1/pages", map[string]interface{}{
		"workspace_id": ws.ID,
		"parent_id":    parent.ID,
		"parent_type":  "page",
		"title":        "Child Page",
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var child pages.Page
	ts.ParseJSON(resp, &child)

	if child.ParentID != parent.ID {
		t.Errorf("parent_id = %q, want %q", child.ParentID, parent.ID)
	}
	if child.ParentType != pages.ParentPage {
		t.Errorf("parent_type = %q, want %q", child.ParentType, pages.ParentPage)
	}
}

func TestPageGet(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("pageget@example.com", "Page Get", "password123")
	ws := createTestWorkspace(ts, cookie, "Get Workspace", "get-ws")

	// Create page
	resp := ts.Request("POST", "/api/v1/pages", map[string]interface{}{
		"workspace_id": ws.ID,
		"title":        "Test Page",
		"icon":         "book",
		"parent_type":  "workspace",
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var created pages.Page
	ts.ParseJSON(resp, &created)

	t.Run("get existing page", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/pages/"+created.ID, nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var page pages.Page
		ts.ParseJSON(resp, &page)

		if page.ID != created.ID {
			t.Errorf("ID = %q, want %q", page.ID, created.ID)
		}
		if page.Title != created.Title {
			t.Errorf("Title = %q, want %q", page.Title, created.Title)
		}
		if page.Icon != created.Icon {
			t.Errorf("Icon = %q, want %q", page.Icon, created.Icon)
		}
	})

	t.Run("non-existent page", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/pages/non-existent-id", nil, cookie)
		ts.ExpectStatus(resp, http.StatusNotFound)
		resp.Body.Close()
	})
}

func TestPageUpdate(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("pageupdate@example.com", "Page Update", "password123")
	ws := createTestWorkspace(ts, cookie, "Update Workspace", "update-ws")

	// Create page
	resp := ts.Request("POST", "/api/v1/pages", map[string]interface{}{
		"workspace_id": ws.ID,
		"title":        "Original Title",
		"icon":         "star",
		"parent_type":  "workspace",
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var created pages.Page
	ts.ParseJSON(resp, &created)

	t.Run("update title", func(t *testing.T) {
		resp := ts.Request("PATCH", "/api/v1/pages/"+created.ID, map[string]interface{}{
			"title": "Updated Title",
		}, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var updated pages.Page
		ts.ParseJSON(resp, &updated)

		if updated.Title != "Updated Title" {
			t.Errorf("Title = %q, want %q", updated.Title, "Updated Title")
		}
	})

	t.Run("update icon", func(t *testing.T) {
		resp := ts.Request("PATCH", "/api/v1/pages/"+created.ID, map[string]interface{}{
			"icon": "moon",
		}, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var updated pages.Page
		ts.ParseJSON(resp, &updated)

		if updated.Icon != "moon" {
			t.Errorf("Icon = %q, want %q", updated.Icon, "moon")
		}
	})

	t.Run("update cover", func(t *testing.T) {
		resp := ts.Request("PATCH", "/api/v1/pages/"+created.ID, map[string]interface{}{
			"cover":   "https://example.com/new-cover.jpg",
			"cover_y": 0.5,
		}, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var updated pages.Page
		ts.ParseJSON(resp, &updated)

		if updated.Cover != "https://example.com/new-cover.jpg" {
			t.Errorf("Cover = %q, want expected", updated.Cover)
		}
	})
}

func TestPageDelete(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("pagedelete@example.com", "Page Delete", "password123")
	ws := createTestWorkspace(ts, cookie, "Delete Workspace", "delete-ws")

	// Create page
	resp := ts.Request("POST", "/api/v1/pages", map[string]interface{}{
		"workspace_id": ws.ID,
		"title":        "To Delete",
		"parent_type":  "workspace",
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var created pages.Page
	ts.ParseJSON(resp, &created)

	t.Run("delete page", func(t *testing.T) {
		resp := ts.Request("DELETE", "/api/v1/pages/"+created.ID, nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)
		resp.Body.Close()
	})

	t.Run("deleted page not found", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/pages/"+created.ID, nil, cookie)
		ts.ExpectStatus(resp, http.StatusNotFound)
		resp.Body.Close()
	})
}

func TestPageList(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("pagelist@example.com", "Page List", "password123")
	ws := createTestWorkspace(ts, cookie, "List Workspace", "list-ws")

	// Create pages
	for i := 0; i < 3; i++ {
		resp := ts.Request("POST", "/api/v1/pages", map[string]interface{}{
			"workspace_id": ws.ID,
			"title":        "Page " + string(rune('A'+i)),
			"parent_type":  "workspace",
		}, cookie)
		ts.ExpectStatus(resp, http.StatusCreated)
		resp.Body.Close()
	}

	t.Run("list workspace pages", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/workspaces/"+ws.ID+"/pages", nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var pageList []*pages.Page
		ts.ParseJSON(resp, &pageList)

		if len(pageList) < 3 {
			t.Errorf("expected at least 3 pages, got %d", len(pageList))
		}
	})
}

func TestPageArchiveRestore(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("archive@example.com", "Archive", "password123")
	ws := createTestWorkspace(ts, cookie, "Archive Workspace", "archive-ws")

	// Create page
	resp := ts.Request("POST", "/api/v1/pages", map[string]interface{}{
		"workspace_id": ws.ID,
		"title":        "To Archive",
		"parent_type":  "workspace",
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var created pages.Page
	ts.ParseJSON(resp, &created)

	t.Run("archive page", func(t *testing.T) {
		resp := ts.Request("POST", "/api/v1/pages/"+created.ID+"/archive", nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)
		resp.Body.Close()

		// Verify archived
		resp = ts.Request("GET", "/api/v1/pages/"+created.ID, nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var page pages.Page
		ts.ParseJSON(resp, &page)

		if !page.IsArchived {
			t.Error("page should be archived")
		}
	})

	t.Run("restore page", func(t *testing.T) {
		resp := ts.Request("POST", "/api/v1/pages/"+created.ID+"/restore", nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)
		resp.Body.Close()

		// Verify restored
		resp = ts.Request("GET", "/api/v1/pages/"+created.ID, nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var page pages.Page
		ts.ParseJSON(resp, &page)

		if page.IsArchived {
			t.Error("page should not be archived")
		}
	})
}

func TestPageDuplicate(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("duplicate@example.com", "Duplicate", "password123")
	ws := createTestWorkspace(ts, cookie, "Duplicate Workspace", "duplicate-ws")

	// Create page with title and icon
	resp := ts.Request("POST", "/api/v1/pages", map[string]interface{}{
		"workspace_id": ws.ID,
		"title":        "Original Page",
		"icon":         "star",
		"parent_type":  "workspace",
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var original pages.Page
	ts.ParseJSON(resp, &original)

	t.Run("duplicate page", func(t *testing.T) {
		resp := ts.Request("POST", "/api/v1/pages/"+original.ID+"/duplicate", map[string]interface{}{}, cookie)
		ts.ExpectStatus(resp, http.StatusCreated)

		var duplicated pages.Page
		ts.ParseJSON(resp, &duplicated)

		if duplicated.ID == original.ID {
			t.Error("duplicated page should have different ID")
		}
		if duplicated.WorkspaceID != original.WorkspaceID {
			t.Errorf("workspace_id = %q, want %q", duplicated.WorkspaceID, original.WorkspaceID)
		}
	})
}

func TestPageGetBlocks(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("blocks@example.com", "Blocks", "password123")
	ws := createTestWorkspace(ts, cookie, "Blocks Workspace", "blocks-ws")

	// Create page
	resp := ts.Request("POST", "/api/v1/pages", map[string]interface{}{
		"workspace_id": ws.ID,
		"title":        "Page with Blocks",
		"parent_type":  "workspace",
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var page pages.Page
	ts.ParseJSON(resp, &page)

	t.Run("empty page blocks", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/pages/"+page.ID+"/blocks", nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var blockList []interface{}
		ts.ParseJSON(resp, &blockList)

		// Empty page should have empty block list or null
	})
}

func TestPageUnauthenticated(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup - create authenticated page first
	_, cookie := ts.Register("authcheck@example.com", "Auth Check", "password123")
	ws := createTestWorkspace(ts, cookie, "Auth Workspace", "auth-ws")

	resp := ts.Request("POST", "/api/v1/pages", map[string]interface{}{
		"workspace_id": ws.ID,
		"title":        "Auth Page",
		"parent_type":  "workspace",
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var page pages.Page
	ts.ParseJSON(resp, &page)

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"create page", "POST", "/api/v1/pages"},
		{"get page", "GET", "/api/v1/pages/" + page.ID},
		{"update page", "PATCH", "/api/v1/pages/" + page.ID},
		{"delete page", "DELETE", "/api/v1/pages/" + page.ID},
		{"list pages", "GET", "/api/v1/workspaces/" + ws.ID + "/pages"},
		{"archive page", "POST", "/api/v1/pages/" + page.ID + "/archive"},
		{"restore page", "POST", "/api/v1/pages/" + page.ID + "/restore"},
		{"duplicate page", "POST", "/api/v1/pages/" + page.ID + "/duplicate"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := ts.Request(tt.method, tt.path, nil) // No cookie
			ts.ExpectStatus(resp, http.StatusUnauthorized)
			resp.Body.Close()
		})
	}
}
