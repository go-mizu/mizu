package api_test

import (
	"net/http"
	"testing"

	"github.com/go-mizu/blueprints/workspace/feature/pages"
	"github.com/go-mizu/blueprints/workspace/feature/search"
)

func TestSearch(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("search@example.com", "Search User", "password123")
	ws := createTestWorkspace(ts, cookie, "Search Workspace", "search-ws")

	// Create pages with searchable content
	pageData := []struct {
		title string
	}{
		{"Project Alpha Documentation"},
		{"Meeting Notes February"},
		{"Technical Specifications"},
		{"Alpha Release Planning"},
		{"Budget Report Q1"},
	}

	for _, pd := range pageData {
		createTestPage(ts, cookie, ws.ID, pd.title)
	}

	t.Run("search for Alpha", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/workspaces/"+ws.ID+"/search?q=Alpha", nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var result search.SearchResult
		ts.ParseJSON(resp, &result)

		// Should find pages with "Alpha" in title
		if len(result.Pages) < 2 {
			t.Errorf("expected at least 2 pages with 'Alpha', got %d", len(result.Pages))
		}
	})

	t.Run("search for Meeting", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/workspaces/"+ws.ID+"/search?q=Meeting", nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var result search.SearchResult
		ts.ParseJSON(resp, &result)

		// Should find at least the Meeting Notes page
		if len(result.Pages) < 1 {
			t.Errorf("expected at least 1 page with 'Meeting', got %d", len(result.Pages))
		}
	})

	t.Run("search with no results", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/workspaces/"+ws.ID+"/search?q=NonexistentTerm123", nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var result search.SearchResult
		ts.ParseJSON(resp, &result)

		// Should return empty or nil results, not error
		if len(result.Pages) != 0 {
			t.Errorf("expected 0 pages, got %d", len(result.Pages))
		}
	})
}

func TestQuickSearch(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("quicksearch@example.com", "Quick Search", "password123")
	ws := createTestWorkspace(ts, cookie, "Quick Search Workspace", "quick-ws")

	// Create pages
	for i := 0; i < 5; i++ {
		createTestPage(ts, cookie, ws.ID, "Quick Page "+string(rune('A'+i)))
	}

	t.Run("quick search", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/workspaces/"+ws.ID+"/quick-search?q=Quick", nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var results []*pages.PageRef
		ts.ParseJSON(resp, &results)

		if len(results) < 5 {
			t.Errorf("expected at least 5 quick search results, got %d", len(results))
		}

		// Verify results have required fields
		for _, r := range results {
			if r.ID == "" {
				t.Error("page ref should have ID")
			}
			if r.Title == "" {
				t.Error("page ref should have title")
			}
		}
	})

	t.Run("quick search limited results", func(t *testing.T) {
		// Create many pages
		for i := 0; i < 20; i++ {
			createTestPage(ts, cookie, ws.ID, "Many Page "+string(rune('A'+i%26)))
		}

		resp := ts.Request("GET", "/api/v1/workspaces/"+ws.ID+"/quick-search?q=Many", nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var results []*pages.PageRef
		ts.ParseJSON(resp, &results)

		// Quick search should return limited results (usually 10)
		if len(results) > 10 {
			t.Errorf("quick search should be limited, got %d results", len(results))
		}
	})
}

func TestRecent(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("recent@example.com", "Recent User", "password123")
	ws := createTestWorkspace(ts, cookie, "Recent Workspace", "recent-ws")

	// Create and access pages
	var pageIDs []string
	for i := 0; i < 5; i++ {
		page := createTestPage(ts, cookie, ws.ID, "Recent Page "+string(rune('A'+i)))
		pageIDs = append(pageIDs, page.ID)

		// Access the page to track it
		resp := ts.Request("GET", "/api/v1/pages/"+page.ID, nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)
		resp.Body.Close()
	}

	t.Run("get recent pages", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/workspaces/"+ws.ID+"/recent", nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var recentPages []*pages.Page
		ts.ParseJSON(resp, &recentPages)

		// Should have some recent pages
		// Note: depends on whether page access is tracked
	})
}

func TestSearchWithDatabases(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("searchdb@example.com", "Search DB", "password123")
	ws := createTestWorkspace(ts, cookie, "Search DB Workspace", "search-db-ws")

	// Create pages
	createTestPage(ts, cookie, ws.ID, "Project Tracker")

	// Create databases
	createTestDatabase(ts, cookie, ws.ID, "Task Database")
	createTestDatabase(ts, cookie, ws.ID, "Bug Tracker Database")

	t.Run("search returns databases", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/workspaces/"+ws.ID+"/search?q=Tracker", nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var result search.SearchResult
		ts.ParseJSON(resp, &result)

		// Should find both pages and databases with "Tracker"
		totalResults := len(result.Pages) + len(result.Databases)
		if totalResults < 2 {
			t.Errorf("expected at least 2 results, got %d", totalResults)
		}
	})
}

func TestSearchCaseSensitivity(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("searchcase@example.com", "Search Case", "password123")
	ws := createTestWorkspace(ts, cookie, "Case Workspace", "case-ws")

	createTestPage(ts, cookie, ws.ID, "UPPERCASE PAGE")
	createTestPage(ts, cookie, ws.ID, "lowercase page")
	createTestPage(ts, cookie, ws.ID, "MixedCase Page")

	t.Run("search is case insensitive", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/workspaces/"+ws.ID+"/search?q=page", nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var result search.SearchResult
		ts.ParseJSON(resp, &result)

		// Should find all pages regardless of case
		if len(result.Pages) < 3 {
			t.Errorf("expected at least 3 pages (case insensitive), got %d", len(result.Pages))
		}
	})
}

func TestSearchPartialMatch(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("searchpartial@example.com", "Search Partial", "password123")
	ws := createTestWorkspace(ts, cookie, "Partial Workspace", "partial-ws")

	createTestPage(ts, cookie, ws.ID, "Documentation")
	createTestPage(ts, cookie, ws.ID, "Document Template")
	createTestPage(ts, cookie, ws.ID, "My Documents")

	t.Run("partial match search", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/workspaces/"+ws.ID+"/search?q=Doc", nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)

		var result search.SearchResult
		ts.ParseJSON(resp, &result)

		// Should find pages with partial match
		if len(result.Pages) < 3 {
			t.Errorf("expected at least 3 pages (partial match), got %d", len(result.Pages))
		}
	})
}

func TestSearchUnauthenticated(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("searchauth@example.com", "Search Auth", "password123")
	ws := createTestWorkspace(ts, cookie, "Auth Workspace", "auth-search-ws")

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"search", "GET", "/api/v1/workspaces/" + ws.ID + "/search?q=test"},
		{"quick search", "GET", "/api/v1/workspaces/" + ws.ID + "/quick-search?q=test"},
		{"recent", "GET", "/api/v1/workspaces/" + ws.ID + "/recent"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := ts.Request(tt.method, tt.path, nil) // No cookie
			ts.ExpectStatus(resp, http.StatusUnauthorized)
			resp.Body.Close()
		})
	}
}
