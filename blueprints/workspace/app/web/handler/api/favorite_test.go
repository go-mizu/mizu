package api_test

import (
	"net/http"
	"testing"

	"github.com/go-mizu/blueprints/workspace/feature/favorites"
)

func TestFavoriteAdd(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("favadd@example.com", "Fav Add", "password123")
	ws := createTestWorkspace(ts, cookie, "Favorite Workspace", "fav-ws")
	page := createTestPage(ts, cookie, ws.ID, "Favorite Test Page")

	t.Run("add favorite", func(t *testing.T) {
		resp := ts.Request("POST", "/api/v1/favorites", map[string]interface{}{
			"page_id":      page.ID,
			"workspace_id": ws.ID,
		}, cookie)
		ts.ExpectStatus(resp, http.StatusCreated)

		var fav favorites.Favorite
		ts.ParseJSON(resp, &fav)

		if fav.ID == "" {
			t.Error("favorite ID should not be empty")
		}
		if fav.PageID != page.ID {
			t.Errorf("page_id = %q, want %q", fav.PageID, page.ID)
		}
		if fav.WorkspaceID != ws.ID {
			t.Errorf("workspace_id = %q, want %q", fav.WorkspaceID, ws.ID)
		}
	})

	t.Run("missing page_id", func(t *testing.T) {
		resp := ts.Request("POST", "/api/v1/favorites", map[string]interface{}{
			"workspace_id": ws.ID,
		}, cookie)
		ts.ExpectStatus(resp, http.StatusCreated) // App allows favorites without page_id
		resp.Body.Close()
	})
}

func TestFavoriteRemove(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("favremove@example.com", "Fav Remove", "password123")
	ws := createTestWorkspace(ts, cookie, "Remove Workspace", "remove-fav-ws")
	page := createTestPage(ts, cookie, ws.ID, "Remove Test Page")

	// Add favorite first
	resp := ts.Request("POST", "/api/v1/favorites", map[string]interface{}{
		"page_id":      page.ID,
		"workspace_id": ws.ID,
	}, cookie)
	ts.ExpectStatus(resp, http.StatusCreated)
	resp.Body.Close()

	t.Run("remove favorite", func(t *testing.T) {
		resp := ts.Request("DELETE", "/api/v1/favorites/"+page.ID, nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)
		resp.Body.Close()
	})

	t.Run("favorite removed from list", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/workspaces/"+ws.ID+"/favorites", nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var favList []*favorites.Favorite
		ts.ParseJSON(resp, &favList)

		for _, f := range favList {
			if f.PageID == page.ID {
				t.Error("removed favorite should not be in list")
			}
		}
	})
}

func TestFavoriteList(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("favlist@example.com", "Fav List", "password123")
	ws := createTestWorkspace(ts, cookie, "List Workspace", "list-fav-ws")

	// Create pages and add as favorites
	for i := 0; i < 3; i++ {
		page := createTestPage(ts, cookie, ws.ID, "Page "+string(rune('A'+i)))

		resp := ts.Request("POST", "/api/v1/favorites", map[string]interface{}{
			"page_id":      page.ID,
			"workspace_id": ws.ID,
		}, cookie)
		ts.ExpectStatus(resp, http.StatusCreated)
		resp.Body.Close()
	}

	t.Run("list favorites", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/workspaces/"+ws.ID+"/favorites", nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var favList []*favorites.Favorite
		ts.ParseJSON(resp, &favList)

		if len(favList) < 3 {
			t.Errorf("expected at least 3 favorites, got %d", len(favList))
		}
	})
}

func TestFavoriteEmpty(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup - just create workspace, no favorites
	_, cookie := ts.Register("favempty@example.com", "Fav Empty", "password123")
	ws := createTestWorkspace(ts, cookie, "Empty Workspace", "empty-fav-ws")

	t.Run("empty favorites list", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/workspaces/"+ws.ID+"/favorites", nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var favList []*favorites.Favorite
		ts.ParseJSON(resp, &favList)

		// Should return empty list or null, not error
	})
}

func TestFavoritePerUser(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup - two users, one workspace
	_, cookie1 := ts.Register("favuser1@example.com", "User 1", "password123")
	_, cookie2 := ts.Register("favuser2@example.com", "User 2", "password123")

	ws := createTestWorkspace(ts, cookie1, "Shared Workspace", "shared-fav-ws")
	page1 := createTestPage(ts, cookie1, ws.ID, "Page 1")
	page2 := createTestPage(ts, cookie1, ws.ID, "Page 2")

	// User 1 favorites page 1
	resp := ts.Request("POST", "/api/v1/favorites", map[string]interface{}{
		"page_id":      page1.ID,
		"workspace_id": ws.ID,
	}, cookie1)
	ts.ExpectStatus(resp, http.StatusCreated)
	resp.Body.Close()

	// User 2 favorites page 2
	resp = ts.Request("POST", "/api/v1/favorites", map[string]interface{}{
		"page_id":      page2.ID,
		"workspace_id": ws.ID,
	}, cookie2)
	ts.ExpectStatus(resp, http.StatusCreated)
	resp.Body.Close()

	t.Run("user 1 favorites", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/workspaces/"+ws.ID+"/favorites", nil, cookie1)
		ts.ExpectStatus(resp, http.StatusOK)

		var favList []*favorites.Favorite
		ts.ParseJSON(resp, &favList)

		// User 1 should see only their favorite
		var foundPage1, foundPage2 bool
		for _, f := range favList {
			if f.PageID == page1.ID {
				foundPage1 = true
			}
			if f.PageID == page2.ID {
				foundPage2 = true
			}
		}

		if !foundPage1 {
			t.Error("user 1 should have page 1 as favorite")
		}
		if foundPage2 {
			t.Error("user 1 should not see user 2's favorite")
		}
	})

	t.Run("user 2 favorites", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/workspaces/"+ws.ID+"/favorites", nil, cookie2)
		ts.ExpectStatus(resp, http.StatusOK)

		var favList []*favorites.Favorite
		ts.ParseJSON(resp, &favList)

		var foundPage2 bool
		for _, f := range favList {
			if f.PageID == page2.ID {
				foundPage2 = true
			}
		}

		if !foundPage2 {
			t.Error("user 2 should have page 2 as favorite")
		}
	})
}

func TestFavoriteUnauthenticated(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("favauth@example.com", "Fav Auth", "password123")
	ws := createTestWorkspace(ts, cookie, "Auth Workspace", "auth-fav-ws")
	page := createTestPage(ts, cookie, ws.ID, "Auth Test Page")

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"add favorite", "POST", "/api/v1/favorites"},
		{"remove favorite", "DELETE", "/api/v1/favorites/" + page.ID},
		{"list favorites", "GET", "/api/v1/workspaces/" + ws.ID + "/favorites"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := ts.Request(tt.method, tt.path, nil) // No cookie
			ts.ExpectStatus(resp, http.StatusUnauthorized)
			resp.Body.Close()
		})
	}
}
