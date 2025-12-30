package web

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-mizu/blueprints/cms/feature/categories"
	"github.com/go-mizu/blueprints/cms/feature/comments"
	"github.com/go-mizu/blueprints/cms/feature/menus"
	"github.com/go-mizu/blueprints/cms/feature/pages"
	"github.com/go-mizu/blueprints/cms/feature/posts"
	"github.com/go-mizu/blueprints/cms/feature/settings"
	"github.com/go-mizu/blueprints/cms/feature/tags"
	"github.com/go-mizu/blueprints/cms/feature/users"
)

// TestWPAdminPages tests all WordPress Admin page rendering
func TestWPAdminPages(t *testing.T) {
	// Create test server with temp database
	tmpDir, err := os.MkdirTemp("", "wpadmin-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	server, err := New(Config{
		Addr:    ":0",
		DataDir: tmpDir,
	})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Create test user
	ctx := context.Background()
	testUser, _, err := server.users.Register(ctx, &users.RegisterIn{
		Email:    "admin@test.com",
		Password: "password123",
		Name:     "Test Admin",
	})
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	// Update user role to administrator
	adminRole := "administrator"
	_, err = server.users.Update(ctx, testUser.ID, &users.UpdateIn{
		Role: &adminRole,
	})
	if err != nil {
		t.Fatalf("failed to update user role: %v", err)
	}

	// Create test data
	seedTestData(t, server, testUser.ID)

	// Test login page (unauthenticated)
	t.Run("LoginPage", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/wp-login.php", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		assertContains(t, rec.Body.String(), "Log In")
		assertContains(t, rec.Body.String(), "Username or Email Address")
		assertContains(t, rec.Body.String(), "Password")
	})

	// Get session for authenticated requests
	_, session, err := server.users.Login(ctx, &users.LoginIn{
		Email:    "admin@test.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("failed to login: %v", err)
	}

	// Helper to create authenticated request
	authRequest := func(method, path string) *http.Request {
		req := httptest.NewRequest(method, path, nil)
		req.AddCookie(&http.Cookie{
			Name:  "session",
			Value: session.ID,
		})
		return req
	}

	// Test dashboard
	t.Run("Dashboard", func(t *testing.T) {
		req := authRequest("GET", "/wp-admin/")
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		assertContains(t, rec.Body.String(), "Dashboard")
		assertContains(t, rec.Body.String(), "At a Glance")
		assertContains(t, rec.Body.String(), "Activity")
		assertContains(t, rec.Body.String(), "Quick Draft")
	})

	t.Run("DashboardIndexPHP", func(t *testing.T) {
		req := authRequest("GET", "/wp-admin/index.php")
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		assertContains(t, rec.Body.String(), "Dashboard")
	})

	// Test Posts pages
	t.Run("PostsList", func(t *testing.T) {
		req := authRequest("GET", "/wp-admin/edit.php")
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		assertContains(t, rec.Body.String(), "Posts")
		assertContains(t, rec.Body.String(), "Add New")
		assertContains(t, rec.Body.String(), "Test Post")
	})

	t.Run("PostsListFiltered", func(t *testing.T) {
		req := authRequest("GET", "/wp-admin/edit.php?post_status=published")
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		assertContains(t, rec.Body.String(), "Published")
	})

	t.Run("PostNew", func(t *testing.T) {
		req := authRequest("GET", "/wp-admin/post-new.php")
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		assertContains(t, rec.Body.String(), "Add New Post")
		assertContains(t, rec.Body.String(), "Title")
		assertContains(t, rec.Body.String(), "Content")
		assertContains(t, rec.Body.String(), "Categories")
		assertContains(t, rec.Body.String(), "Tags")
	})

	// Test Pages
	t.Run("PagesList", func(t *testing.T) {
		req := authRequest("GET", "/wp-admin/edit.php?post_type=page")
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		assertContains(t, rec.Body.String(), "Pages")
		assertContains(t, rec.Body.String(), "Add New")
		assertContains(t, rec.Body.String(), "Test Page")
	})

	t.Run("PageNew", func(t *testing.T) {
		req := authRequest("GET", "/wp-admin/post-new.php?post_type=page")
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		assertContains(t, rec.Body.String(), "Add New Page")
		assertContains(t, rec.Body.String(), "Title")
	})

	// Test Media
	t.Run("MediaLibrary", func(t *testing.T) {
		req := authRequest("GET", "/wp-admin/upload.php")
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		assertContains(t, rec.Body.String(), "Media Library")
	})

	t.Run("MediaNew", func(t *testing.T) {
		req := authRequest("GET", "/wp-admin/media-new.php")
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		assertContains(t, rec.Body.String(), "Upload New Media")
	})

	// Test Comments
	t.Run("CommentsList", func(t *testing.T) {
		req := authRequest("GET", "/wp-admin/edit-comments.php")
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		assertContains(t, rec.Body.String(), "Comments")
	})

	// Test Users
	t.Run("UsersList", func(t *testing.T) {
		req := authRequest("GET", "/wp-admin/users.php")
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		assertContains(t, rec.Body.String(), "Users")
		assertContains(t, rec.Body.String(), "Test Admin")
	})

	t.Run("UserNew", func(t *testing.T) {
		req := authRequest("GET", "/wp-admin/user-new.php")
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		assertContains(t, rec.Body.String(), "Add New User")
		assertContains(t, rec.Body.String(), "Username")
		assertContains(t, rec.Body.String(), "Email")
	})

	t.Run("Profile", func(t *testing.T) {
		req := authRequest("GET", "/wp-admin/profile.php")
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		assertContains(t, rec.Body.String(), "Profile")
		assertContains(t, rec.Body.String(), "Personal Options")
	})

	// Test Categories
	t.Run("CategoriesList", func(t *testing.T) {
		req := authRequest("GET", "/wp-admin/edit-tags.php?taxonomy=category")
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		assertContains(t, rec.Body.String(), "Categories")
		assertContains(t, rec.Body.String(), "Test Category")
	})

	// Test Tags
	t.Run("TagsList", func(t *testing.T) {
		req := authRequest("GET", "/wp-admin/edit-tags.php?taxonomy=post_tag")
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		assertContains(t, rec.Body.String(), "Tags")
		assertContains(t, rec.Body.String(), "Test Tag")
	})

	// Test Menus
	t.Run("Menus", func(t *testing.T) {
		req := authRequest("GET", "/wp-admin/nav-menus.php")
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		assertContains(t, rec.Body.String(), "Menus")
	})

	// Test Settings pages
	t.Run("SettingsGeneral", func(t *testing.T) {
		req := authRequest("GET", "/wp-admin/options-general.php")
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		assertContains(t, rec.Body.String(), "General Settings")
		assertContains(t, rec.Body.String(), "Site Title")
		assertContains(t, rec.Body.String(), "Tagline")
	})

	t.Run("SettingsWriting", func(t *testing.T) {
		req := authRequest("GET", "/wp-admin/options-writing.php")
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		assertContains(t, rec.Body.String(), "Writing Settings")
		assertContains(t, rec.Body.String(), "Default Post Category")
	})

	t.Run("SettingsReading", func(t *testing.T) {
		req := authRequest("GET", "/wp-admin/options-reading.php")
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		assertContains(t, rec.Body.String(), "Reading Settings")
		assertContains(t, rec.Body.String(), "homepage displays")
	})

	t.Run("SettingsDiscussion", func(t *testing.T) {
		req := authRequest("GET", "/wp-admin/options-discussion.php")
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		assertContains(t, rec.Body.String(), "Discussion Settings")
		assertContains(t, rec.Body.String(), "Default post settings")
	})

	t.Run("SettingsMedia", func(t *testing.T) {
		req := authRequest("GET", "/wp-admin/options-media.php")
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		assertContains(t, rec.Body.String(), "Media Settings")
		assertContains(t, rec.Body.String(), "Image sizes")
	})

	t.Run("SettingsPermalinks", func(t *testing.T) {
		req := authRequest("GET", "/wp-admin/options-permalink.php")
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		assertContains(t, rec.Body.String(), "Permalink Settings")
		assertContains(t, rec.Body.String(), "Common Settings")
	})

	// Test static assets
	t.Run("CSSAssets", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/wp-admin/css/wpadmin.css", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		assertContains(t, rec.Body.String(), "#wpadminbar")
	})

	t.Run("JSAssets", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/wp-admin/js/wpadmin.js", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		assertContains(t, rec.Body.String(), "initMenuToggle")
	})
}

// TestWPAdminAuthentication tests authentication behavior
func TestWPAdminAuthentication(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wpadmin-auth-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	server, err := New(Config{
		Addr:    ":0",
		DataDir: tmpDir,
	})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Test that unauthenticated requests redirect to login
	adminPages := []string{
		"/wp-admin/",
		"/wp-admin/index.php",
		"/wp-admin/edit.php",
		"/wp-admin/post-new.php",
		"/wp-admin/upload.php",
		"/wp-admin/edit-comments.php",
		"/wp-admin/users.php",
		"/wp-admin/options-general.php",
	}

	for _, page := range adminPages {
		t.Run("RedirectToLogin_"+page, func(t *testing.T) {
			req := httptest.NewRequest("GET", page, nil)
			rec := httptest.NewRecorder()
			server.app.ServeHTTP(rec, req)

			// Should redirect to login
			if rec.Code != http.StatusFound && rec.Code != http.StatusSeeOther {
				t.Errorf("expected redirect status, got %d for %s", rec.Code, page)
			}

			location := rec.Header().Get("Location")
			if !strings.HasPrefix(location, "/wp-login.php") {
				t.Errorf("expected redirect to /wp-login.php, got %s", location)
			}
		})
	}
}

// TestWPAdminMenuHighlighting tests that the correct menu item is highlighted
func TestWPAdminMenuHighlighting(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wpadmin-menu-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	server, err := New(Config{
		Addr:    ":0",
		DataDir: tmpDir,
	})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	ctx := context.Background()
	_, session, _ := server.users.Register(ctx, &users.RegisterIn{
		Email:    "menu@test.com",
		Password: "password123",
		Name:     "Menu Test",
	})

	authRequest := func(path string) *http.Request {
		req := httptest.NewRequest("GET", path, nil)
		req.AddCookie(&http.Cookie{
			Name:  "session",
			Value: session.ID,
		})
		return req
	}

	tests := []struct {
		path        string
		activeClass string
	}{
		{"/wp-admin/", "menu-icon-dashboard"},
		{"/wp-admin/edit.php", "menu-icon-post"},
		{"/wp-admin/upload.php", "menu-icon-media"},
		{"/wp-admin/edit-comments.php", "menu-icon-comments"},
		{"/wp-admin/users.php", "menu-icon-users"},
		{"/wp-admin/options-general.php", "menu-icon-settings"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := authRequest(tt.path)
			rec := httptest.NewRecorder()
			server.app.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d", rec.Code)
			}
			// The active menu item should have the "current" class
			assertContains(t, rec.Body.String(), "current")
		})
	}
}

// TestWPAdminPagination tests pagination on list pages
func TestWPAdminPagination(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wpadmin-pagination-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	server, err := New(Config{
		Addr:    ":0",
		DataDir: tmpDir,
	})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	ctx := context.Background()
	user, session, _ := server.users.Register(ctx, &users.RegisterIn{
		Email:    "pagination@test.com",
		Password: "password123",
		Name:     "Pagination Test",
	})

	// Create 25 posts to trigger pagination
	for i := 0; i < 25; i++ {
		_, err := server.posts.Create(ctx, user.ID, &posts.CreateIn{
			Title:   fmt.Sprintf("Pagination Test Post %d", i+1),
			Content: "Test content",
			Status:  "published",
		})
		if err != nil {
			t.Fatalf("failed to create test post: %v", err)
		}
	}

	authRequest := func(path string) *http.Request {
		req := httptest.NewRequest("GET", path, nil)
		req.AddCookie(&http.Cookie{
			Name:  "session",
			Value: session.ID,
		})
		return req
	}

	t.Run("PostsListPage1", func(t *testing.T) {
		req := authRequest("/wp-admin/edit.php?paged=1")
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		// Should contain posts and pagination controls
		assertContains(t, rec.Body.String(), "items")
	})

	t.Run("PostsListPage2", func(t *testing.T) {
		req := authRequest("/wp-admin/edit.php?paged=2")
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
	})
}

// TestWPAdminSearch tests search functionality
func TestWPAdminSearch(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wpadmin-search-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	server, err := New(Config{
		Addr:    ":0",
		DataDir: tmpDir,
	})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	ctx := context.Background()
	user, session, _ := server.users.Register(ctx, &users.RegisterIn{
		Email:    "search@test.com",
		Password: "password123",
		Name:     "Search Test",
	})

	// Create posts with specific titles
	_, _ = server.posts.Create(ctx, user.ID, &posts.CreateIn{
		Title:   "Unique Searchable Post",
		Content: "Content here",
		Status:  "published",
	})
	_, _ = server.posts.Create(ctx, user.ID, &posts.CreateIn{
		Title:   "Another Regular Post",
		Content: "More content",
		Status:  "published",
	})

	authRequest := func(path string) *http.Request {
		req := httptest.NewRequest("GET", path, nil)
		req.AddCookie(&http.Cookie{
			Name:  "session",
			Value: session.ID,
		})
		return req
	}

	t.Run("SearchPosts", func(t *testing.T) {
		req := authRequest("/wp-admin/edit.php?s=Unique")
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		assertContains(t, rec.Body.String(), "Unique Searchable Post")
	})
}

// seedTestData creates test data for the WordPress Admin tests
func seedTestData(t *testing.T, server *Server, userID string) {
	ctx := context.Background()

	// Create test settings
	_, _ = server.settings.Set(ctx, &settings.SetIn{
		Key:   "site_title",
		Value: "Test WordPress Site",
	})
	_, _ = server.settings.Set(ctx, &settings.SetIn{
		Key:   "tagline",
		Value: "Just another WordPress site",
	})

	// Create test category
	cat, _ := server.categories.Create(ctx, &categories.CreateIn{
		Name:        "Test Category",
		Slug:        "test-category",
		Description: "A test category",
	})

	// Create test tag
	tag, _ := server.tags.Create(ctx, &tags.CreateIn{
		Name:        "Test Tag",
		Slug:        "test-tag",
		Description: "A test tag",
	})

	// Create test post
	post, _ := server.posts.Create(ctx, userID, &posts.CreateIn{
		Title:       "Test Post",
		Slug:        "test-post",
		Content:     "This is test content for the test post.",
		Status:      "published",
		CategoryIDs: []string{cat.ID},
		TagIDs:      []string{tag.ID},
	})

	// Create test page
	_, _ = server.pages.Create(ctx, userID, &pages.CreateIn{
		Title:   "Test Page",
		Slug:    "test-page",
		Content: "This is test content for the test page.",
		Status:  "published",
	})

	// Create test comment
	_, _ = server.comments.Create(ctx, &comments.CreateIn{
		PostID:      post.ID,
		AuthorName:  "Test Commenter",
		AuthorEmail: "commenter@test.com",
		Content:     "This is a test comment.",
	})

	// Create test menu
	menu, _ := server.menus.CreateMenu(ctx, &menus.CreateMenuIn{
		Name:     "Test Menu",
		Location: "primary",
	})
	_ = menu

	// Note: Media upload requires file reader, skipping for UI tests
	// The UI still renders correctly without media items
}

// assertContains checks if the body contains the expected string
func assertContains(t *testing.T, body, expected string) {
	t.Helper()
	if !containsString(body, expected) {
		t.Errorf("expected body to contain %q", expected)
	}
}

// containsString checks if s contains substr
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
