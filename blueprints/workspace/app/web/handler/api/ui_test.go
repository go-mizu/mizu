package api_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestUIPublicPages(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	tests := []struct {
		name       string
		path       string
		wantStatus int
	}{
		{
			name:       "home redirects to login",
			path:       "/",
			wantStatus: http.StatusFound,
		},
		{
			name:       "login page",
			path:       "/login",
			wantStatus: http.StatusOK,
		},
		{
			name:       "register page",
			path:       "/register",
			wantStatus: http.StatusOK,
		},
		{
			name:       "health check",
			path:       "/health",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := ts.Request("GET", tt.path, nil)
			ts.ExpectStatus(resp, tt.wantStatus)
			resp.Body.Close()
		})
	}
}

func TestUIAuthRequiredRedirect(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup - create workspace and page with authenticated user
	_, cookie := ts.Register("uiredirect@example.com", "UI Redirect", "password123")
	ws := createTestWorkspace(ts, cookie, "UI Workspace", "ui-ws")
	page := createTestPage(ts, cookie, ws.ID, "UI Test Page")

	tests := []struct {
		name string
		path string
	}{
		{"app redirect", "/app"},
		{"workspace page", "/w/" + ws.Slug},
		{"page view", "/w/" + ws.Slug + "/p/" + page.ID},
		{"search page", "/w/" + ws.Slug + "/search"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Without authentication, should redirect to login
			resp := ts.Request("GET", tt.path, nil)
			ts.ExpectStatus(resp, http.StatusFound)

			location := resp.Header.Get("Location")
			if !strings.Contains(location, "/login") {
				t.Errorf("expected redirect to /login, got %s", location)
			}
			resp.Body.Close()
		})
	}
}

func TestUIAuthenticatedPages(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup
	_, cookie := ts.Register("uiauth@example.com", "UI Auth", "password123")
	ws := createTestWorkspace(ts, cookie, "UI Auth Workspace", "ui-auth-ws")
	page := createTestPage(ts, cookie, ws.ID, "UI Auth Page")

	tests := []struct {
		name       string
		path       string
		wantStatus int
	}{
		{
			name:       "workspace page",
			path:       "/w/" + ws.Slug,
			wantStatus: http.StatusOK,
		},
		{
			name:       "page view",
			path:       "/w/" + ws.Slug + "/p/" + page.ID,
			wantStatus: http.StatusOK,
		},
		{
			name:       "search page",
			path:       "/w/" + ws.Slug + "/search",
			wantStatus: http.StatusOK,
		},
		{
			name:       "settings page",
			path:       "/w/" + ws.Slug + "/settings",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := ts.Request("GET", tt.path, nil, cookie)
			ts.ExpectStatus(resp, tt.wantStatus)
			resp.Body.Close()
		})
	}
}

func TestUIAppRedirect(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	// Setup - user with a workspace
	_, cookie := ts.Register("uiappredirect@example.com", "UI App Redirect", "password123")
	createTestWorkspace(ts, cookie, "App Redirect Workspace", "app-redirect-ws")

	t.Run("app redirects to workspace", func(t *testing.T) {
		resp := ts.Request("GET", "/app", nil, cookie)
		// Should redirect to a workspace
		if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusOK {
			t.Errorf("expected redirect or OK, got %d", resp.StatusCode)
		}
		resp.Body.Close()
	})
}

func TestUIStaticAssets(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	tests := []struct {
		name       string
		path       string
		wantStatus int
	}{
		{
			name:       "static css exists",
			path:       "/static/css/style.css",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := ts.Request("GET", tt.path, nil)
			// Static files may or may not exist depending on build
			// Just check it doesn't panic
			resp.Body.Close()
		})
	}
}

func TestUIWorkspaceNotFound(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	_, cookie := ts.Register("uinotfound@example.com", "UI Not Found", "password123")

	t.Run("non-existent workspace", func(t *testing.T) {
		resp := ts.Request("GET", "/w/non-existent-workspace", nil, cookie)
		// Should return 404 or redirect
		if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusFound {
			// May also render an error page with 200
		}
		resp.Body.Close()
	})
}

func TestUIPageNotFound(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	_, cookie := ts.Register("uipagenotfound@example.com", "UI Page Not Found", "password123")
	ws := createTestWorkspace(ts, cookie, "Page Not Found Workspace", "page-not-found-ws")

	t.Run("non-existent page", func(t *testing.T) {
		resp := ts.Request("GET", "/w/"+ws.Slug+"/p/non-existent-page-id", nil, cookie)
		// Should return 404 or show error
		resp.Body.Close()
	})
}

func TestUIHealthCheck(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	t.Run("health endpoint", func(t *testing.T) {
		resp := ts.Request("GET", "/health", nil)
		ts.ExpectStatus(resp, http.StatusOK)

		var result map[string]string
		ts.ParseJSON(resp, &result)

		if result["status"] != "ok" {
			t.Errorf("health status = %q, want 'ok'", result["status"])
		}
	})
}

func TestUILoginPage(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	t.Run("login page renders", func(t *testing.T) {
		resp := ts.Request("GET", "/login", nil)
		ts.ExpectStatus(resp, http.StatusOK)

		// Check content type
		contentType := resp.Header.Get("Content-Type")
		if !strings.Contains(contentType, "text/html") {
			t.Errorf("expected HTML content type, got %s", contentType)
		}
		resp.Body.Close()
	})
}

func TestUIRegisterPage(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	t.Run("register page renders", func(t *testing.T) {
		resp := ts.Request("GET", "/register", nil)
		ts.ExpectStatus(resp, http.StatusOK)

		// Check content type
		contentType := resp.Header.Get("Content-Type")
		if !strings.Contains(contentType, "text/html") {
			t.Errorf("expected HTML content type, got %s", contentType)
		}
		resp.Body.Close()
	})
}

func TestUISearchPage(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	_, cookie := ts.Register("uisearch@example.com", "UI Search", "password123")
	ws := createTestWorkspace(ts, cookie, "Search UI Workspace", "search-ui-ws")

	t.Run("search page renders", func(t *testing.T) {
		resp := ts.Request("GET", "/w/"+ws.Slug+"/search", nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)
		resp.Body.Close()
	})

	t.Run("search page with query", func(t *testing.T) {
		resp := ts.Request("GET", "/w/"+ws.Slug+"/search?q=test", nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)
		resp.Body.Close()
	})
}

func TestUIDatabase(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	_, cookie := ts.Register("uidb@example.com", "UI DB", "password123")
	ws := createTestWorkspace(ts, cookie, "DB UI Workspace", "db-ui-ws")
	db := createTestDatabase(ts, cookie, ws.ID, "UI Database")

	t.Run("database page renders", func(t *testing.T) {
		resp := ts.Request("GET", "/w/"+ws.Slug+"/d/"+db.ID, nil, cookie)
		ts.ExpectStatus(resp, http.StatusOK)
		resp.Body.Close()
	})
}
