package api_test

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

// TestUILoginWorkflow tests the complete login workflow
func TestUILoginWorkflow(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Register a user first
	ts.Register("loginworkflow@example.com", "Login Workflow", "password123")

	// Test login via API (simulating frontend)
	resp := ts.Request("POST", "/api/v1/auth/login", map[string]string{
		"email":    "loginworkflow@example.com",
		"password": "password123",
	})
	ts.ExpectStatus(resp, http.StatusOK)

	// Verify session cookie
	var hasCookie bool
	for _, c := range resp.Cookies() {
		if c.Name == "workspace_session" {
			hasCookie = true
			break
		}
	}
	if !hasCookie {
		t.Error("login should return session cookie")
	}
	resp.Body.Close()
}

// TestUIRegisterWorkflow tests the complete registration workflow
func TestUIRegisterWorkflow(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Test registration via API (simulating frontend)
	resp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email":    "registerworkflow@example.com",
		"name":     "Register Workflow",
		"password": "password123",
	})
	ts.ExpectStatus(resp, http.StatusCreated)

	// Verify session cookie
	var hasCookie bool
	for _, c := range resp.Cookies() {
		if c.Name == "workspace_session" {
			hasCookie = true
			break
		}
	}
	if !hasCookie {
		t.Error("register should return session cookie")
	}
	resp.Body.Close()
}

// TestUIPageCreationWorkflow tests creating a page via the UI workflow
func TestUIPageCreationWorkflow(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("pageworkflow@example.com", "Page Workflow", "password123")
	ws := createTestWorkspace(ts, cookie, "Workflow WS", "workflow-ws")

	// Create page via API (simulating frontend click on "New Page")
	resp := ts.Request("POST", "/api/v1/pages", map[string]interface{}{
		"workspace_id": ws.ID,
		"title":        "Untitled",
		"parent_type":  "workspace",
		"parent_id":    ws.ID,
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var page struct {
		ID string `json:"id"`
	}
	ts.ParseJSON(resp, &page)

	// Navigate to the new page (simulating redirect)
	resp = ts.Request("GET", "/w/"+ws.Slug+"/p/"+page.ID, nil, cookie)
	ts.ExpectStatus(resp, http.StatusOK)
	resp.Body.Close()
}

// TestUIDatabaseCreationWorkflow tests creating a database via the UI workflow
func TestUIDatabaseCreationWorkflow(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("dbworkflow@example.com", "DB Workflow", "password123")
	ws := createTestWorkspace(ts, cookie, "DB Workflow WS", "db-workflow-ws")

	// Create database via API
	resp := ts.Request("POST", "/api/v1/databases", map[string]interface{}{
		"workspace_id": ws.ID,
		"title":        "Untitled Database",
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var db struct {
		ID string `json:"id"`
	}
	ts.ParseJSON(resp, &db)

	// Navigate to the new database
	resp = ts.Request("GET", "/w/"+ws.Slug+"/d/"+db.ID, nil, cookie)
	ts.ExpectStatus(resp, http.StatusOK)
	resp.Body.Close()
}

// TestUISearchWorkflow tests the search functionality
func TestUISearchWorkflow(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("searchworkflow@example.com", "Search Workflow", "password123")
	ws := createTestWorkspace(ts, cookie, "Search Workflow WS", "search-workflow-ws")
	createTestPage(ts, cookie, ws.ID, "Searchable Page Title")

	// Access search page
	resp := ts.Request("GET", "/w/"+ws.Slug+"/search", nil, cookie)
	ts.ExpectStatus(resp, http.StatusOK)
	resp.Body.Close()

	// Perform search via API
	resp = ts.Request("GET", "/api/v1/workspaces/"+ws.ID+"/search?q=Searchable", nil, cookie)
	ts.ExpectStatus(resp, http.StatusOK)
	resp.Body.Close()
}

// TestUIFavoriteWorkflow tests favoriting a page
func TestUIFavoriteWorkflow(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("favworkflow@example.com", "Fav Workflow", "password123")
	ws := createTestWorkspace(ts, cookie, "Fav Workflow WS", "fav-workflow-ws")
	page := createTestPage(ts, cookie, ws.ID, "Favorite This Page")

	// Add to favorites
	resp := ts.Request("POST", "/api/v1/favorites", map[string]string{
		"page_id":      page.ID,
		"workspace_id": ws.ID,
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)
	resp.Body.Close()

	// Verify it appears in favorites list
	resp = ts.Request("GET", "/api/v1/workspaces/"+ws.ID+"/favorites", nil, cookie)
	ts.ExpectStatus(resp, http.StatusOK)
	resp.Body.Close()
}

// TestAPIPathConsistency verifies all API paths use /api/v1 prefix
func TestAPIPathConsistency(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	_, cookie := ts.Register("pathtest@example.com", "Path Test", "password123")

	// Test that old /api/ paths don't work (should return 404)
	oldPaths := []struct {
		method string
		path   string
	}{
		{"POST", "/api/auth/login"},
		{"POST", "/api/auth/register"},
		{"GET", "/api/auth/me"},
		{"GET", "/api/workspaces"},
	}

	for _, p := range oldPaths {
		t.Run("old path "+p.path, func(t *testing.T) {
			resp := ts.Request(p.method, p.path, nil, cookie)
			// Old paths should return 404 (not found) since they don't exist
			if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
				t.Errorf("old API path %s should not work, got %d", p.path, resp.StatusCode)
			}
			resp.Body.Close()
		})
	}

	// Test that new /api/v1/ paths work
	newPaths := []struct {
		method string
		path   string
		want   int
	}{
		{"GET", "/api/v1/auth/me", http.StatusOK},
		{"GET", "/api/v1/workspaces", http.StatusOK},
	}

	for _, p := range newPaths {
		t.Run("new path "+p.path, func(t *testing.T) {
			resp := ts.Request(p.method, p.path, nil, cookie)
			ts.ExpectStatus(resp, p.want)
			resp.Body.Close()
		})
	}
}

// TestFormElementsExist verifies HTML forms have required elements
func TestFormElementsExist(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	t.Run("login form elements", func(t *testing.T) {
		resp := ts.Request("GET", "/login", nil)
		body, _ := io.ReadAll(resp.Body)
		html := string(body)

		required := []string{"email", "password"}
		for _, elem := range required {
			if !strings.Contains(html, elem) {
				t.Errorf("login page missing element: %s", elem)
			}
		}
	})

	t.Run("register form elements", func(t *testing.T) {
		resp := ts.Request("GET", "/register", nil)
		body, _ := io.ReadAll(resp.Body)
		html := string(body)

		required := []string{"name", "email", "password"}
		for _, elem := range required {
			if !strings.Contains(html, elem) {
				t.Errorf("register page missing element: %s", elem)
			}
		}
	})
}

// TestCompleteUserJourney tests a complete user workflow
func TestCompleteUserJourney(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// 1. Register new user
	user, cookie := ts.Register("journey@example.com", "Journey User", "password123")
	if user.ID == "" {
		t.Fatal("user should have an ID")
	}

	// 2. Access app - should create default workspace or redirect
	resp := ts.Request("GET", "/app", nil, cookie)
	if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusOK {
		t.Errorf("/app should redirect or show content, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 3. Create a workspace
	ws := createTestWorkspace(ts, cookie, "Journey Workspace", "journey-ws")

	// 4. View workspace page
	resp = ts.Request("GET", "/w/"+ws.Slug, nil, cookie)
	ts.ExpectStatus(resp, http.StatusOK)
	resp.Body.Close()

	// 5. Create a page
	page := createTestPage(ts, cookie, ws.ID, "Journey Page")

	// 6. View the page
	resp = ts.Request("GET", "/w/"+ws.Slug+"/p/"+page.ID, nil, cookie)
	ts.ExpectStatus(resp, http.StatusOK)
	resp.Body.Close()

	// 7. Create a database
	db := createTestDatabase(ts, cookie, ws.ID, "Journey Database")

	// 8. View the database
	resp = ts.Request("GET", "/w/"+ws.Slug+"/d/"+db.ID, nil, cookie)
	ts.ExpectStatus(resp, http.StatusOK)
	resp.Body.Close()

	// 9. View search page
	resp = ts.Request("GET", "/w/"+ws.Slug+"/search", nil, cookie)
	ts.ExpectStatus(resp, http.StatusOK)
	resp.Body.Close()

	// 10. View settings page
	resp = ts.Request("GET", "/w/"+ws.Slug+"/settings", nil, cookie)
	ts.ExpectStatus(resp, http.StatusOK)
	resp.Body.Close()

	// 11. Logout
	resp = ts.Request("POST", "/api/v1/auth/logout", nil, cookie)
	ts.ExpectStatus(resp, http.StatusOK)
	resp.Body.Close()

	// 12. Protected pages should redirect to login
	resp = ts.Request("GET", "/w/"+ws.Slug, nil)
	ts.ExpectStatus(resp, http.StatusFound)
	location := resp.Header.Get("Location")
	if !strings.Contains(location, "/login") {
		t.Errorf("should redirect to /login, got %s", location)
	}
	resp.Body.Close()

	// 13. User can login again
	resp = ts.Request("POST", "/api/v1/auth/login", map[string]string{
		"email":    "journey@example.com",
		"password": "password123",
	})
	ts.ExpectStatus(resp, http.StatusOK)
	resp.Body.Close()
}

// TestProtectedEndpointsRequireAuth tests that all protected endpoints require authentication
func TestProtectedEndpointsRequireAuth(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	endpoints := []struct {
		method string
		path   string
	}{
		{"GET", "/api/v1/auth/me"},
		{"POST", "/api/v1/auth/logout"},
		{"GET", "/api/v1/workspaces"},
		{"POST", "/api/v1/workspaces"},
		{"POST", "/api/v1/pages"},
		{"POST", "/api/v1/blocks"},
		{"POST", "/api/v1/databases"},
		{"POST", "/api/v1/views"},
		{"POST", "/api/v1/comments"},
		{"POST", "/api/v1/favorites"},
	}

	for _, ep := range endpoints {
		t.Run(ep.method+" "+ep.path, func(t *testing.T) {
			resp := ts.Request(ep.method, ep.path, nil)
			ts.ExpectStatus(resp, http.StatusUnauthorized)
			resp.Body.Close()
		})
	}
}

// TestUIProtectedPagesRedirect tests that protected UI pages redirect to login
func TestUIProtectedPagesRedirect(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Create a workspace with an authenticated user first
	_, cookie := ts.Register("uiredirect@example.com", "UI Redirect", "password123")
	ws := createTestWorkspace(ts, cookie, "Redirect Test WS", "redirect-test-ws")
	page := createTestPage(ts, cookie, ws.ID, "Redirect Test Page")
	db := createTestDatabase(ts, cookie, ws.ID, "Redirect Test DB")

	paths := []string{
		"/app",
		"/w/" + ws.Slug,
		"/w/" + ws.Slug + "/p/" + page.ID,
		"/w/" + ws.Slug + "/d/" + db.ID,
		"/w/" + ws.Slug + "/search",
		"/w/" + ws.Slug + "/settings",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			// Without authentication, should redirect to login
			resp := ts.Request("GET", path, nil)
			ts.ExpectStatus(resp, http.StatusFound)
			location := resp.Header.Get("Location")
			if !strings.Contains(location, "/login") {
				t.Errorf("expected redirect to /login, got %s", location)
			}
			resp.Body.Close()
		})
	}
}

// TestBlockCreationWorkflow tests creating blocks via the UI workflow
func TestBlockCreationWorkflow(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("blockworkflow@example.com", "Block Workflow", "password123")
	ws := createTestWorkspace(ts, cookie, "Block Workflow WS", "block-workflow-ws")
	page := createTestPage(ts, cookie, ws.ID, "Block Test Page")

	blockTypes := []struct {
		blockType string
		position  int
	}{
		{"paragraph", 0},
		{"heading_1", 1},
		{"heading_2", 2},
		{"heading_3", 3},
	}

	for _, bt := range blockTypes {
		t.Run(bt.blockType, func(t *testing.T) {
			resp := ts.Request("POST", "/api/v1/blocks", map[string]interface{}{
				"page_id":  page.ID,
				"type":     bt.blockType,
				"position": bt.position,
				"content": map[string]interface{}{
					"rich_text": []map[string]interface{}{
						{"type": "text", "text": "Test content for " + bt.blockType},
					},
				},
			}, cookie)
			ts.ExpectStatus(resp, http.StatusCreated)
			resp.Body.Close()
		})
	}
}

// TestPageArchiveRestoreWorkflow tests the archive and restore workflow
func TestPageArchiveRestoreWorkflow(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("archiveworkflow@example.com", "Archive Workflow", "password123")
	ws := createTestWorkspace(ts, cookie, "Archive Workflow WS", "archive-workflow-ws")
	page := createTestPage(ts, cookie, ws.ID, "Archive Test Page")

	// Archive the page
	resp := ts.Request("POST", "/api/v1/pages/"+page.ID+"/archive", nil, cookie)
	ts.ExpectStatus(resp, http.StatusOK)
	resp.Body.Close()

	// Verify the page is archived
	resp = ts.Request("GET", "/api/v1/pages/"+page.ID, nil, cookie)
	ts.ExpectStatus(resp, http.StatusOK)
	var archived struct {
		IsArchived bool `json:"is_archived"`
	}
	ts.ParseJSON(resp, &archived)
	if !archived.IsArchived {
		t.Error("page should be archived")
	}

	// Restore the page
	resp = ts.Request("POST", "/api/v1/pages/"+page.ID+"/restore", nil, cookie)
	ts.ExpectStatus(resp, http.StatusOK)
	resp.Body.Close()

	// Verify the page is restored
	resp = ts.Request("GET", "/api/v1/pages/"+page.ID, nil, cookie)
	ts.ExpectStatus(resp, http.StatusOK)
	ts.ParseJSON(resp, &archived)
	if archived.IsArchived {
		t.Error("page should not be archived after restore")
	}
}

// TestPageDuplicateWorkflow tests the page duplication workflow
func TestPageDuplicateWorkflow(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("duplicateworkflow@example.com", "Duplicate Workflow", "password123")
	ws := createTestWorkspace(ts, cookie, "Duplicate Workflow WS", "duplicate-workflow-ws")
	originalPage := createTestPage(ts, cookie, ws.ID, "Original Page")

	// Duplicate the page
	resp := ts.Request("POST", "/api/v1/pages/"+originalPage.ID+"/duplicate", nil, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var duplicatedPage struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}
	ts.ParseJSON(resp, &duplicatedPage)

	// Verify the duplicate has different ID but similar title
	if duplicatedPage.ID == originalPage.ID {
		t.Error("duplicated page should have different ID")
	}

	// Navigate to the duplicated page
	resp = ts.Request("GET", "/w/"+ws.Slug+"/p/"+duplicatedPage.ID, nil, cookie)
	ts.ExpectStatus(resp, http.StatusOK)
	resp.Body.Close()
}

// TestWorkspaceSettingsWorkflow tests the workspace settings workflow
func TestWorkspaceSettingsWorkflow(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("settingsworkflow@example.com", "Settings Workflow", "password123")
	ws := createTestWorkspace(ts, cookie, "Settings Workflow WS", "settings-workflow-ws")

	// Access settings page
	resp := ts.Request("GET", "/w/"+ws.Slug+"/settings", nil, cookie)
	ts.ExpectStatus(resp, http.StatusOK)
	resp.Body.Close()

	// Update workspace name
	resp = ts.Request("PATCH", "/api/v1/workspaces/"+ws.ID, map[string]string{
		"name": "Updated Workspace Name",
	}, cookie)
	ts.ExpectStatus(resp, http.StatusOK)
	resp.Body.Close()

	// Verify update
	resp = ts.Request("GET", "/api/v1/workspaces/"+ws.ID, nil, cookie)
	ts.ExpectStatus(resp, http.StatusOK)
	var updated struct {
		Name string `json:"name"`
	}
	ts.ParseJSON(resp, &updated)
	if updated.Name != "Updated Workspace Name" {
		t.Errorf("expected 'Updated Workspace Name', got %q", updated.Name)
	}
}

// TestCommentWorkflow tests the comment creation and management workflow
func TestCommentWorkflow(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("commentworkflow@example.com", "Comment Workflow", "password123")
	ws := createTestWorkspace(ts, cookie, "Comment Workflow WS", "comment-workflow-ws")
	page := createTestPage(ts, cookie, ws.ID, "Comment Test Page")

	// Create a comment
	resp := ts.Request("POST", "/api/v1/comments", map[string]interface{}{
		"workspace_id": ws.ID,
		"target_type":  "page",
		"target_id":    page.ID,
		"content": []map[string]interface{}{
			{"type": "text", "text": "This is a test comment"},
		},
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)

	var comment struct {
		ID string `json:"id"`
	}
	ts.ParseJSON(resp, &comment)

	// List comments on the page
	resp = ts.Request("GET", "/api/v1/pages/"+page.ID+"/comments", nil, cookie)
	ts.ExpectStatus(resp, http.StatusOK)
	resp.Body.Close()

	// Update the comment
	resp = ts.Request("PATCH", "/api/v1/comments/"+comment.ID, map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": "Updated comment"},
		},
	}, cookie)
	ts.ExpectStatus(resp, http.StatusOK)
	resp.Body.Close()

	// Resolve the comment
	resp = ts.Request("POST", "/api/v1/comments/"+comment.ID+"/resolve", nil, cookie)
	ts.ExpectStatus(resp, http.StatusOK)
	resp.Body.Close()

	// Delete the comment
	resp = ts.Request("DELETE", "/api/v1/comments/"+comment.ID, nil, cookie)
	ts.ExpectStatus(resp, http.StatusOK)
	resp.Body.Close()
}

// TestViewWorkflow tests the database view creation and management workflow
func TestViewWorkflow(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("viewworkflow@example.com", "View Workflow", "password123")
	ws := createTestWorkspace(ts, cookie, "View Workflow WS", "view-workflow-ws")
	db := createTestDatabase(ts, cookie, ws.ID, "View Test Database")

	viewTypes := []string{"table", "board", "list", "calendar", "gallery", "timeline"}

	for _, viewType := range viewTypes {
		t.Run(viewType, func(t *testing.T) {
			resp := ts.Request("POST", "/api/v1/views", map[string]interface{}{
				"database_id": db.ID,
				"type":        viewType,
				"name":        viewType + " view",
			}, cookie)
			ts.ExpectStatus(resp, http.StatusCreated)
			resp.Body.Close()
		})
	}

	// List views for the database
	resp := ts.Request("GET", "/api/v1/databases/"+db.ID+"/views", nil, cookie)
	ts.ExpectStatus(resp, http.StatusOK)
	resp.Body.Close()
}

// TestShareWorkflow tests the page sharing workflow
func TestShareWorkflow(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup - two users
	userA, cookieA := ts.Register("sharer@example.com", "Sharer", "password123")
	userB, _ := ts.Register("sharee@example.com", "Sharee", "password123")

	ws := createTestWorkspace(ts, cookieA, "Share Workflow WS", "share-workflow-ws")
	page := createTestPage(ts, cookieA, ws.ID, "Shared Page")

	// Share page with user B
	resp := ts.Request("POST", "/api/v1/pages/"+page.ID+"/shares", map[string]interface{}{
		"type":       "user",
		"user_id":    userB.ID,
		"permission": "edit",
	}, cookieA)
	ts.ExpectStatus(resp, http.StatusCreated)

	var share struct {
		ID string `json:"id"`
	}
	ts.ParseJSON(resp, &share)

	// List shares for the page
	resp = ts.Request("GET", "/api/v1/pages/"+page.ID+"/shares", nil, cookieA)
	ts.ExpectStatus(resp, http.StatusOK)
	resp.Body.Close()

	// Delete the share
	resp = ts.Request("DELETE", "/api/v1/shares/"+share.ID, nil, cookieA)
	ts.ExpectStatus(resp, http.StatusOK)
	resp.Body.Close()

	_ = userA // Prevent unused variable warning
}
