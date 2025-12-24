package web_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-mizu/blueprints/social/app/web"

	_ "github.com/duckdb/duckdb-go/v2"
)

// setupUITestServer creates a test server for UI testing.
func setupUITestServer(t *testing.T) *httptest.Server {
	t.Helper()

	tempDir := t.TempDir()
	srv, err := web.New(web.Config{
		Addr:    ":0",
		DataDir: tempDir,
		Dev:     true,
	})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	ts := httptest.NewServer(srv.Handler())

	t.Cleanup(func() {
		ts.Close()
		srv.Close()
	})

	return ts
}

// setupUITestServerWithData creates a test server with sample data created via API.
func setupUITestServerWithData(t *testing.T) *httptest.Server {
	t.Helper()

	tempDir := t.TempDir()
	srv, err := web.New(web.Config{
		Addr:    ":0",
		DataDir: tempDir,
		Dev:     true,
	})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	ts := httptest.NewServer(srv.Handler())

	// Create test user via API
	registerBody := `{"username":"testuser","email":"test@example.com","password":"password123"}`
	resp, err := http.Post(ts.URL+"/api/v1/auth/register", "application/json", strings.NewReader(registerBody))
	if err != nil {
		ts.Close()
		srv.Close()
		t.Fatalf("register user: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		ts.Close()
		srv.Close()
		t.Fatalf("register user failed: %d", resp.StatusCode)
	}

	t.Cleanup(func() {
		ts.Close()
		srv.Close()
	})

	return ts
}

// assertNoTemplateErrors checks that the response body has no template rendering errors.
func assertNoTemplateErrors(t *testing.T, body string, path string) {
	t.Helper()

	errorPatterns := []string{
		"error calling slice",
		"nil pointer dereference",
		"runtime error:",
		"panic:",
	}

	for _, pattern := range errorPatterns {
		if strings.Contains(body, pattern) {
			t.Errorf("page %s contains template error: %q\nbody preview: %s", path, pattern, truncateForError(body))
		}
	}

	if strings.Contains(body, "template:") && strings.Contains(body, "error") {
		t.Errorf("page %s contains template error\nbody preview: %s", path, truncateForError(body))
	}
}

func truncateForError(s string) string {
	if len(s) > 500 {
		return s[:500] + "..."
	}
	return s
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
	assertNoTemplateErrors(t, body, "/")
	assertUIPageContains(t, body,
		"Social",                // Logo/site name
		"Home",                  // Page title
		"sidebar",               // Navigation
		"compose-box",           // Compose box
		"/static/css/style.css", // CSS link
		"/static/js/app.js",     // JS link
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
	assertNoTemplateErrors(t, body, "/login")
	assertUIPageContains(t, body,
		"Login",             // Page title
		"Username or Email", // Form field
		"Password",          // Form field
		"login-form",        // Form id
		"/register",         // Registration link
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
	assertNoTemplateErrors(t, body, "/register")
	assertUIPageContains(t, body,
		"Create Account", // Page title
		"Username",       // Form field
		"Email",          // Form field
		"Password",       // Form field
		"register-form",  // Form id
		"/login",         // Login link
	)
}

// TestUI_ExplorePage tests the explore page renders correctly.
func TestUI_ExplorePage(t *testing.T) {
	ts := setupUITestServer(t)

	status, body := getUIPage(t, ts.URL+"/explore")

	if status != http.StatusOK {
		t.Errorf("expected 200, got %d", status)
	}

	assertUIValidHTML(t, body)
	assertNoTemplateErrors(t, body, "/explore")
	assertUIPageContains(t, body,
		"Explore",        // Page title
		"Trending Tags",  // Section
		"Trending Posts", // Section
	)
}

// TestUI_SearchPage tests the search page renders correctly.
func TestUI_SearchPage(t *testing.T) {
	ts := setupUITestServer(t)

	status, body := getUIPage(t, ts.URL+"/search")

	if status != http.StatusOK {
		t.Errorf("expected 200, got %d", status)
	}

	assertUIValidHTML(t, body)
	assertNoTemplateErrors(t, body, "/search")
	assertUIPageContains(t, body,
		"Search",      // Page title
		"search-form", // Search form
		"search-tabs", // Search tabs
	)
}

// TestUI_SearchPage_WithQuery tests search page with query.
func TestUI_SearchPage_WithQuery(t *testing.T) {
	ts := setupUITestServer(t)

	status, body := getUIPage(t, ts.URL+"/search?q=test")

	if status != http.StatusOK {
		t.Errorf("expected 200, got %d", status)
	}

	assertUIValidHTML(t, body)
	assertNoTemplateErrors(t, body, "/search?q=test")
	assertUIPageContains(t, body,
		"Search",
		`value="test"`, // Query preserved in input
	)
}

// TestUI_ProfilePage tests the profile page renders correctly.
func TestUI_ProfilePage(t *testing.T) {
	ts := setupUITestServerWithData(t)

	status, body := getUIPage(t, ts.URL+"/u/testuser")

	if status != http.StatusOK {
		t.Errorf("expected 200, got %d", status)
	}

	assertUIValidHTML(t, body)
	assertNoTemplateErrors(t, body, "/u/testuser")
	assertUIPageContains(t, body,
		"testuser",       // Username
		"@testuser",      // Username display
		"profile-header", // Profile header
		"profile-stats",  // Stats section
		"Following",      // Stats label
		"Followers",      // Stats label
	)
}

// TestUI_ProfilePage_NotFound tests profile not found error.
func TestUI_ProfilePage_NotFound(t *testing.T) {
	ts := setupUITestServer(t)

	status, body := getUIPage(t, ts.URL+"/u/nonexistent")

	assertNoTemplateErrors(t, body, "/u/nonexistent")
	// Should render 404 page or show error
	if status != http.StatusOK && status != http.StatusNotFound {
		t.Errorf("expected 200 or 404, got %d", status)
	}
}

// TestUI_TagPage tests the tag page renders correctly.
func TestUI_TagPage(t *testing.T) {
	ts := setupUITestServer(t)

	status, body := getUIPage(t, ts.URL+"/tags/test")

	if status != http.StatusOK {
		t.Errorf("expected 200, got %d", status)
	}

	assertUIValidHTML(t, body)
	assertNoTemplateErrors(t, body, "/tags/test")
}

// TestUI_NotificationsPage tests the notifications page.
func TestUI_NotificationsPage(t *testing.T) {
	ts := setupUITestServer(t)

	status, body := getUIPage(t, ts.URL+"/notifications")

	if status != http.StatusOK {
		t.Errorf("expected 200, got %d", status)
	}

	assertUIValidHTML(t, body)
	assertNoTemplateErrors(t, body, "/notifications")
}

// TestUI_BookmarksPage tests the bookmarks page.
func TestUI_BookmarksPage(t *testing.T) {
	ts := setupUITestServer(t)

	status, body := getUIPage(t, ts.URL+"/bookmarks")

	if status != http.StatusOK {
		t.Errorf("expected 200, got %d", status)
	}

	assertUIValidHTML(t, body)
	assertNoTemplateErrors(t, body, "/bookmarks")
}

// TestUI_ListsPage tests the lists page.
func TestUI_ListsPage(t *testing.T) {
	ts := setupUITestServer(t)

	status, body := getUIPage(t, ts.URL+"/lists")

	if status != http.StatusOK {
		t.Errorf("expected 200, got %d", status)
	}

	assertUIValidHTML(t, body)
	assertNoTemplateErrors(t, body, "/lists")
}

// TestUI_SettingsPage tests the settings page.
func TestUI_SettingsPage(t *testing.T) {
	ts := setupUITestServer(t)

	status, body := getUIPage(t, ts.URL+"/settings")

	if status != http.StatusOK {
		t.Errorf("expected 200, got %d", status)
	}

	assertUIValidHTML(t, body)
	assertNoTemplateErrors(t, body, "/settings")
}

// TestUI_StaticAssets tests static assets are served.
func TestUI_StaticAssets(t *testing.T) {
	ts := setupUITestServer(t)

	t.Run("CSS", func(t *testing.T) {
		status, body := getUIPage(t, ts.URL+"/static/css/style.css")

		if status != http.StatusOK {
			t.Errorf("expected 200, got %d", status)
		}

		// Check CSS contains content
		if len(body) == 0 {
			t.Error("CSS file is empty")
		}
	})

	t.Run("JS", func(t *testing.T) {
		status, body := getUIPage(t, ts.URL+"/static/js/app.js")

		if status != http.StatusOK {
			t.Errorf("expected 200, got %d", status)
		}

		// Check JS contains content
		if len(body) == 0 {
			t.Error("JS file is empty")
		}
	})
}

// TestUI_Navigation tests navigation component.
func TestUI_Navigation(t *testing.T) {
	ts := setupUITestServer(t)

	status, body := getUIPage(t, ts.URL+"/")

	if status != http.StatusOK {
		t.Errorf("expected 200, got %d", status)
	}

	// Check navigation elements
	assertUIPageContains(t, body,
		`class="sidebar"`,   // Nav element
		`class="logo"`,      // Logo
		`class="nav-links"`, // Nav links
		"/explore",          // Explore link
		"/notifications",    // Notifications link
		"/search",           // Search link
		"/bookmarks",        // Bookmarks link
		"/lists",            // Lists link
		"/settings",         // Settings link
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
		{"/", "Home"},
		{"/login", "Login"},
		{"/register", "Register"},
		{"/explore", "Explore"},
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

			if !strings.Contains(body, tc.contains) {
				t.Errorf("page title missing expected content: %q", tc.contains)
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
			`required`, // Required fields
		)
	})

	t.Run("RegisterForm", func(t *testing.T) {
		status, body := getUIPage(t, ts.URL+"/register")

		if status != http.StatusOK {
			t.Errorf("expected 200, got %d", status)
		}

		assertUIPageContains(t, body,
			`required`,      // Required fields
			`minlength="8"`, // Password length
			`type="email"`,  // Email type
		)
	})
}

// TestUI_TemplateIsolation verifies that each page renders its own content block.
func TestUI_TemplateIsolation(t *testing.T) {
	ts := setupUITestServerWithData(t)

	tests := []struct {
		name        string
		path        string
		mustHave    []string // Content unique to this page
		mustNotHave []string // Content from OTHER pages that should NOT appear
	}{
		{
			name:        "HomePage_NotProfilePage",
			path:        "/",
			mustHave:    []string{"compose-box", "Home"},
			mustNotHave: []string{"profile-header", "profile-stats"},
		},
		{
			name:        "ExplorePage_NotHomePage",
			path:        "/explore",
			mustHave:    []string{"Explore", "Trending Tags", "Trending Posts"},
			mustNotHave: []string{"compose-box"},
		},
		{
			name:        "LoginPage_NotRegisterPage",
			path:        "/login",
			mustHave:    []string{"Login", "login-form"},
			mustNotHave: []string{"Create Account", "register-form"},
		},
		{
			name:        "RegisterPage_NotLoginPage",
			path:        "/register",
			mustHave:    []string{"Create Account", "register-form"},
			mustNotHave: []string{"login-form"},
		},
		{
			name:        "SearchPage_NotExplorePage",
			path:        "/search",
			mustHave:    []string{"Search", "search-form", "search-tabs"},
			mustNotHave: []string{"Trending Tags"},
		},
		{
			name:        "ProfilePage_HasProfileContent",
			path:        "/u/testuser",
			mustHave:    []string{"testuser", "profile-header", "Following", "Followers"},
			mustNotHave: []string{"compose-box"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := getUIPage(t, ts.URL+tt.path)

			if status != http.StatusOK {
				t.Fatalf("status: got %d, want 200", status)
			}

			// First, check for template errors
			assertNoTemplateErrors(t, body, tt.path)

			// Check for required content
			for _, want := range tt.mustHave {
				if !strings.Contains(body, want) {
					t.Errorf("page %s missing expected content: %q", tt.path, want)
				}
			}

			// Check that content from OTHER templates doesn't appear
			for _, notWant := range tt.mustNotHave {
				if strings.Contains(body, notWant) {
					t.Errorf("page %s incorrectly contains content from wrong template: %q (template collision bug!)", tt.path, notWant)
				}
			}
		})
	}
}

// TestUI_AllPagesRenderWithoutErrors visits every page and checks for template errors.
func TestUI_AllPagesRenderWithoutErrors(t *testing.T) {
	ts := setupUITestServerWithData(t)

	pages := []struct {
		path     string
		wantCode int
	}{
		{"/", 200},
		{"/login", 200},
		{"/register", 200},
		{"/explore", 200},
		{"/search", 200},
		{"/search?q=test", 200},
		{"/notifications", 200},
		{"/bookmarks", 200},
		{"/lists", 200},
		{"/settings", 200},
		{"/u/testuser", 200},
		{"/u/nonexistent", 200}, // Returns 404 page
		{"/tags/test", 200},
	}

	for _, page := range pages {
		t.Run(page.path, func(t *testing.T) {
			status, body := getUIPage(t, ts.URL+page.path)

			if status != page.wantCode {
				t.Errorf("status: got %d, want %d", status, page.wantCode)
			}

			assertNoTemplateErrors(t, body, page.path)
			assertUIValidHTML(t, body)
		})
	}
}
