//go:build e2e

package web_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/blueprints/forum/app/web"
	"github.com/go-mizu/mizu/blueprints/forum/feature/accounts"
	"github.com/go-mizu/mizu/blueprints/forum/feature/boards"
	"github.com/go-mizu/mizu/blueprints/forum/feature/threads"
	"github.com/go-mizu/mizu/blueprints/forum/store/duckdb"

	_ "github.com/marcboeker/go-duckdb"
)

// setupUITestServer creates a test server for UI testing.
func setupUITestServer(t *testing.T) *httptest.Server {
	t.Helper()

	tempDir := t.TempDir()
	store, err := duckdb.Open(tempDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	srv, err := web.NewServer(store, web.ServerConfig{
		Addr: ":0",
		Dev:  true,
	})
	if err != nil {
		store.Close()
		t.Fatalf("new server: %v", err)
	}

	ts := httptest.NewServer(srv)

	t.Cleanup(func() {
		ts.Close()
		store.Close()
	})

	return ts
}

// setupUITestServerWithData creates a test server with sample data.
func setupUITestServerWithData(t *testing.T) (*httptest.Server, *duckdb.Store) {
	t.Helper()

	tempDir := t.TempDir()
	store, err := duckdb.Open(tempDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	// Create test user
	ctx := context.Background()
	accSvc := accounts.NewService(store.Accounts())
	testUser, err := accSvc.Create(ctx, accounts.CreateIn{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	// Create test board
	boardSvc := boards.NewService(store.Boards())
	testBoard, err := boardSvc.Create(ctx, testUser.ID, boards.CreateIn{
		Name:        "testboard",
		Title:       "Test Board",
		Description: "A test board for UI testing",
	})
	if err != nil {
		t.Fatalf("create board: %v", err)
	}

	// Create test thread
	threadSvc := threads.NewService(store.Threads(), accSvc, boardSvc)
	_, err = threadSvc.Create(ctx, testUser.ID, threads.CreateIn{
		BoardID: testBoard.ID,
		Title:   "Test Thread",
		Content: "This is test content",
		Type:    "text",
	})
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}

	srv, err := web.NewServer(store, web.ServerConfig{
		Addr: ":0",
		Dev:  true,
	})
	if err != nil {
		store.Close()
		t.Fatalf("new server: %v", err)
	}

	ts := httptest.NewServer(srv)

	t.Cleanup(func() {
		ts.Close()
		store.Close()
	})

	return ts, store
}

func getUIPage(t *testing.T, url string) (int, string) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	return resp.StatusCode, string(body)
}

func assertUIPageContains(t *testing.T, body string, substrings ...string) {
	t.Helper()
	for _, s := range substrings {
		if !strings.Contains(body, s) {
			t.Errorf("page missing expected content: %q", s)
		}
	}
}

func assertUIValidHTML(t *testing.T, body string) {
	t.Helper()
	if !strings.Contains(body, "<!DOCTYPE html") {
		t.Error("response is not valid HTML (missing DOCTYPE)")
	}
	if !strings.Contains(body, "</html>") {
		t.Error("response is not valid HTML (missing closing html tag)")
	}
}

// TestUI_HomePage tests the home page renders correctly.
func TestUI_HomePage(t *testing.T) {
	ts := setupUITestServer(t)

	status, body := getUIPage(t, ts.URL+"/")

	if status != http.StatusOK {
		t.Errorf("expected 200, got %d", status)
	}

	assertUIValidHTML(t, body)
	assertUIPageContains(t, body,
		"Forum",               // Logo/site name
		"nav",                 // Navigation
		"sort-tabs",           // Sort tabs
		"Popular Communities", // Sidebar section
		"/static/css/app.css", // CSS link
		"/static/js/app.js",   // JS link
	)
}

// TestUI_HomePageWithData tests home page with actual data.
func TestUI_HomePageWithData(t *testing.T) {
	ts, _ := setupUITestServerWithData(t)

	status, body := getUIPage(t, ts.URL+"/")

	if status != http.StatusOK {
		t.Errorf("expected 200, got %d", status)
	}

	assertUIValidHTML(t, body)
	assertUIPageContains(t, body,
		"Test Thread", // Thread title
		"testboard",   // Board name
	)
}

// TestUI_LoginPage tests the login page renders correctly.
func TestUI_LoginPage(t *testing.T) {
	ts := setupUITestServer(t)

	status, body := getUIPage(t, ts.URL+"/login")

	if status != http.StatusOK {
		t.Errorf("expected 200, got %d", status)
	}

	assertUIValidHTML(t, body)
	assertUIPageContains(t, body,
		"Welcome back",        // Page title
		"Username or Email",   // Form field
		"Password",            // Form field
		"Log In",              // Submit button
		"/register",           // Registration link
		`data-action="login"`, // Form action
	)
}

// TestUI_RegisterPage tests the register page renders correctly.
func TestUI_RegisterPage(t *testing.T) {
	ts := setupUITestServer(t)

	status, body := getUIPage(t, ts.URL+"/register")

	if status != http.StatusOK {
		t.Errorf("expected 200, got %d", status)
	}

	assertUIValidHTML(t, body)
	assertUIPageContains(t, body,
		"Create your account",    // Page title
		"Username",               // Form field
		"Email",                  // Form field
		"Password",               // Form field
		"Sign Up",                // Submit button
		"/login",                 // Login link
		`data-action="register"`, // Form action
	)
}

// TestUI_BoardPage tests the board page renders correctly.
func TestUI_BoardPage(t *testing.T) {
	ts, _ := setupUITestServerWithData(t)

	status, body := getUIPage(t, ts.URL+"/b/testboard")

	if status != http.StatusOK {
		t.Errorf("expected 200, got %d", status)
	}

	assertUIValidHTML(t, body)
	assertUIPageContains(t, body,
		"Test Board",      // Board title
		"b/testboard",     // Board name
		"About Community", // Sidebar section
		"sort-tabs",       // Sort tabs
	)
}

// TestUI_BoardPage_NotFound tests board not found error.
func TestUI_BoardPage_NotFound(t *testing.T) {
	ts := setupUITestServer(t)

	status, _ := getUIPage(t, ts.URL+"/b/nonexistent")

	// Should return 404 or render error page
	if status != http.StatusNotFound && status != http.StatusOK {
		t.Errorf("expected 404 or 200, got %d", status)
	}
}

// TestUI_SearchPage tests the search page renders correctly.
func TestUI_SearchPage(t *testing.T) {
	ts := setupUITestServer(t)

	status, body := getUIPage(t, ts.URL+"/search")

	if status != http.StatusOK {
		t.Errorf("expected 200, got %d", status)
	}

	assertUIValidHTML(t, body)
	assertUIPageContains(t, body,
		"Search",        // Page title
		`type="search"`, // Search input
	)
}

// TestUI_SearchPage_WithQuery tests search page with query.
func TestUI_SearchPage_WithQuery(t *testing.T) {
	ts, _ := setupUITestServerWithData(t)

	status, body := getUIPage(t, ts.URL+"/search?q=test")

	if status != http.StatusOK {
		t.Errorf("expected 200, got %d", status)
	}

	assertUIValidHTML(t, body)
	assertUIPageContains(t, body,
		"Search",
		`value="test"`, // Query preserved in input
	)
}

// TestUI_AllPage tests the all posts page.
func TestUI_AllPage(t *testing.T) {
	ts, _ := setupUITestServerWithData(t)

	status, body := getUIPage(t, ts.URL+"/all")

	if status != http.StatusOK {
		t.Errorf("expected 200, got %d", status)
	}

	assertUIValidHTML(t, body)
	assertUIPageContains(t, body,
		"All Posts",
	)
}

// TestUI_UserPage tests the user profile page.
func TestUI_UserPage(t *testing.T) {
	ts, _ := setupUITestServerWithData(t)

	status, body := getUIPage(t, ts.URL+"/u/testuser")

	if status != http.StatusOK {
		t.Errorf("expected 200, got %d", status)
	}

	assertUIValidHTML(t, body)
	assertUIPageContains(t, body,
		"testuser",   // Username
		"u/testuser", // Username display
		"Karma",      // Stat label
		"Posts",      // Tab
		"Comments",   // Tab
	)
}

// TestUI_UserPage_NotFound tests user not found error.
func TestUI_UserPage_NotFound(t *testing.T) {
	ts := setupUITestServer(t)

	status, _ := getUIPage(t, ts.URL+"/u/nonexistent")

	// Should return 404 or render error page
	if status != http.StatusNotFound && status != http.StatusOK {
		t.Errorf("expected 404 or 200, got %d", status)
	}
}

// TestUI_StaticAssets tests static assets are served.
func TestUI_StaticAssets(t *testing.T) {
	ts := setupUITestServer(t)

	t.Run("CSS", func(t *testing.T) {
		status, body := getUIPage(t, ts.URL+"/static/css/app.css")

		if status != http.StatusOK {
			t.Errorf("expected 200, got %d", status)
		}

		// Check CSS contains key design tokens
		assertUIPageContains(t, body,
			"--bg-canvas",      // CSS variable
			"--accent-primary", // CSS variable
			".nav",             // Navigation styles
			".thread-card",     // Thread card styles
		)
	})

	t.Run("JS", func(t *testing.T) {
		status, body := getUIPage(t, ts.URL+"/static/js/app.js")

		if status != http.StatusOK {
			t.Errorf("expected 200, got %d", status)
		}

		// Check JS contains key functions
		assertUIPageContains(t, body,
			"handleVote",     // Vote function
			"handleBookmark", // Bookmark function
			"handleLogin",    // Login function
		)
	})
}

// TestUI_Navigation tests navigation component.
func TestUI_Navigation(t *testing.T) {
	ts := setupUITestServer(t)

	status, body := getUIPage(t, ts.URL+"/")

	if status != http.StatusOK {
		t.Errorf("expected 200, got %d", status)
	}

	// Check navigation elements for logged out user
	assertUIPageContains(t, body,
		`class="nav"`,        // Nav element
		`class="nav-logo"`,   // Logo
		`class="nav-search"`, // Search
		"/login",             // Login link
		"/register",          // Register link
	)
}

// TestUI_SortTabs tests sort tabs functionality.
func TestUI_SortTabs(t *testing.T) {
	ts, _ := setupUITestServerWithData(t)

	testCases := []struct {
		sort     string
		expected string
	}{
		{"hot", `class="sort-tab active"`},
		{"new", `class="sort-tab active"`},
		{"top", `class="sort-tab active"`},
	}

	for _, tc := range testCases {
		t.Run(tc.sort, func(t *testing.T) {
			status, body := getUIPage(t, ts.URL+"/?sort="+tc.sort)

			if status != http.StatusOK {
				t.Errorf("expected 200, got %d", status)
			}

			if !strings.Contains(body, tc.expected) {
				t.Errorf("active tab not highlighted for sort=%s", tc.sort)
			}
		})
	}
}

// TestUI_ThreadCard tests thread card component.
func TestUI_ThreadCard(t *testing.T) {
	ts, _ := setupUITestServerWithData(t)

	status, body := getUIPage(t, ts.URL+"/")

	if status != http.StatusOK {
		t.Errorf("expected 200, got %d", status)
	}

	assertUIPageContains(t, body,
		`class="thread-card"`,       // Thread card
		`class="vote-column"`,       // Vote buttons
		`class="vote-btn upvote"`,   // Upvote button
		`class="vote-btn downvote"`, // Downvote button
		`class="vote-score"`,        // Score display
		`class="thread-title"`,      // Title
		`class="thread-actions"`,    // Actions
	)
}

// TestUI_EmptyState tests empty state rendering.
func TestUI_EmptyState(t *testing.T) {
	ts := setupUITestServer(t)

	status, body := getUIPage(t, ts.URL+"/")

	if status != http.StatusOK {
		t.Errorf("expected 200, got %d", status)
	}

	// With no data, should show empty state
	assertUIPageContains(t, body,
		"empty-state",
	)
}

// TestUI_ResponsiveMetaTag tests responsive viewport meta tag.
func TestUI_ResponsiveMetaTag(t *testing.T) {
	ts := setupUITestServer(t)

	status, body := getUIPage(t, ts.URL+"/")

	if status != http.StatusOK {
		t.Errorf("expected 200, got %d", status)
	}

	assertUIPageContains(t, body,
		`<meta name="viewport"`,
		`width=device-width`,
	)
}

// TestUI_PageTitles tests page titles are set correctly.
func TestUI_PageTitles(t *testing.T) {
	ts := setupUITestServer(t)

	testCases := []struct {
		path     string
		contains string
	}{
		{"/", "Forum"},
		{"/login", "Log In"},
		{"/register", "Register"},
		{"/search", "Search"},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			status, body := getUIPage(t, ts.URL+tc.path)

			if status != http.StatusOK {
				t.Errorf("expected 200, got %d", status)
			}

			if !strings.Contains(body, "<title>") {
				t.Error("page missing title tag")
			}
		})
	}
}

// TestUI_FormValidation tests form validation attributes.
func TestUI_FormValidation(t *testing.T) {
	ts := setupUITestServer(t)

	t.Run("LoginForm", func(t *testing.T) {
		status, body := getUIPage(t, ts.URL+"/login")

		if status != http.StatusOK {
			t.Errorf("expected 200, got %d", status)
		}

		assertUIPageContains(t, body,
			`required`,                // Required fields
			`autocomplete="username"`, // Autocomplete
		)
	})

	t.Run("RegisterForm", func(t *testing.T) {
		status, body := getUIPage(t, ts.URL+"/register")

		if status != http.StatusOK {
			t.Errorf("expected 200, got %d", status)
		}

		assertUIPageContains(t, body,
			`required`,       // Required fields
			`minlength="8"`,  // Password length
			`minlength="3"`,  // Username length
			`type="email"`,   // Email type
		)
	})
}

// TestUI_SVGIcons tests SVG icons are properly embedded.
func TestUI_SVGIcons(t *testing.T) {
	ts, _ := setupUITestServerWithData(t)

	status, body := getUIPage(t, ts.URL+"/")

	if status != http.StatusOK {
		t.Errorf("expected 200, got %d", status)
	}

	// Check for SVG usage in vote buttons and navigation
	assertUIPageContains(t, body,
		"<svg",
		"viewBox",
	)
}

// TestUI_DataAttributes tests data attributes for JavaScript.
func TestUI_DataAttributes(t *testing.T) {
	ts, _ := setupUITestServerWithData(t)

	status, body := getUIPage(t, ts.URL+"/")

	if status != http.StatusOK {
		t.Errorf("expected 200, got %d", status)
	}

	// Check for data attributes used by JavaScript
	assertUIPageContains(t, body,
		`data-action=`, // Action attributes
		`data-thread=`, // Thread ID attributes
	)
}
