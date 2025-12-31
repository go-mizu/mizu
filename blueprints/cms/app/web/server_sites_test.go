package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-mizu/blueprints/cms/feature/categories"
	"github.com/go-mizu/blueprints/cms/feature/pages"
	"github.com/go-mizu/blueprints/cms/feature/posts"
	"github.com/go-mizu/blueprints/cms/feature/tags"
	"github.com/go-mizu/blueprints/cms/feature/users"
)

// assertStatus checks the HTTP status code
func assertStatus(t *testing.T, got, want int) {
	t.Helper()
	if got != want {
		t.Errorf("expected status %d, got %d", want, got)
	}
}

// assertBodyContains checks the body contains a substring (local version)
func assertBodyContains(t *testing.T, body, expected string) {
	t.Helper()
	if !strings.Contains(body, expected) {
		t.Errorf("expected body to contain %q, got body of length %d", expected, len(body))
	}
}

// TestSitePages tests all public-facing site page rendering
func TestSitePages(t *testing.T) {
	// Create test server with temp database
	tmpDir, err := os.MkdirTemp("", "site-test-*")
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

	// Create test data
	ctx := context.Background()
	testUser, _, err := server.users.Register(ctx, &users.RegisterIn{
		Email:    "author@test.com",
		Password: "password123",
		Name:     "Test Author",
	})
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	// Seed test data
	seedSiteTestData(t, server, testUser.ID)

	// Test Homepage
	t.Run("Homepage", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		assertStatus(t, rec.Code, http.StatusOK)
		body := rec.Body.String()
		assertBodyContains(t, body, "<!DOCTYPE html>")
		assertBodyContains(t, body, "</html>")
	})

	t.Run("HomepageWithPagination", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/?page=1", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		assertStatus(t, rec.Code, http.StatusOK)
	})

	// Test Single Post Page
	t.Run("SinglePost", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test-post", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		assertStatus(t, rec.Code, http.StatusOK)
		body := rec.Body.String()
		assertBodyContains(t, body, "Test Post Title")
		assertBodyContains(t, body, "article")
	})

	t.Run("SinglePostNotFound", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/non-existent-post", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		assertStatus(t, rec.Code, http.StatusNotFound)
		body := rec.Body.String()
		assertBodyContains(t, body, "404")
		assertBodyContains(t, body, "page-error")
	})

	// Test Single Page
	t.Run("SinglePage", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/page/about-us", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		assertStatus(t, rec.Code, http.StatusOK)
		body := rec.Body.String()
		assertBodyContains(t, body, "About Us")
	})

	t.Run("SinglePageNotFound", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/page/non-existent-page", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		assertStatus(t, rec.Code, http.StatusNotFound)
		body := rec.Body.String()
		assertBodyContains(t, body, "404")
	})

	// Test Category Archive
	t.Run("CategoryArchive", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/category/technology", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		assertStatus(t, rec.Code, http.StatusOK)
		body := rec.Body.String()
		assertBodyContains(t, body, "Technology")
		assertBodyContains(t, body, "Category")
	})

	t.Run("CategoryArchiveWithPagination", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/category/technology?page=1", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		assertStatus(t, rec.Code, http.StatusOK)
	})

	t.Run("CategoryArchiveNotFound", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/category/non-existent", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		assertStatus(t, rec.Code, http.StatusNotFound)
		body := rec.Body.String()
		assertBodyContains(t, body, "404")
	})

	// Test Tag Archive
	t.Run("TagArchive", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/tag/golang", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		assertStatus(t, rec.Code, http.StatusOK)
		body := rec.Body.String()
		assertBodyContains(t, body, "golang")
		assertBodyContains(t, body, "Tag")
	})

	t.Run("TagArchiveWithPagination", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/tag/golang?page=1", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		assertStatus(t, rec.Code, http.StatusOK)
	})

	t.Run("TagArchiveNotFound", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/tag/non-existent-tag", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		assertStatus(t, rec.Code, http.StatusNotFound)
		body := rec.Body.String()
		assertBodyContains(t, body, "404")
	})

	// Test Author Archive
	t.Run("AuthorArchive", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/author/test-author", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		assertStatus(t, rec.Code, http.StatusOK)
		body := rec.Body.String()
		assertBodyContains(t, body, "Test Author")
	})

	t.Run("AuthorArchiveWithPagination", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/author/test-author?page=1", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		assertStatus(t, rec.Code, http.StatusOK)
	})

	t.Run("AuthorArchiveNotFound", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/author/non-existent-author", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		assertStatus(t, rec.Code, http.StatusNotFound)
		body := rec.Body.String()
		assertBodyContains(t, body, "404")
	})

	// Test General Archive
	t.Run("Archive", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/archive", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		assertStatus(t, rec.Code, http.StatusOK)
		body := rec.Body.String()
		assertBodyContains(t, body, "Archive")
	})

	t.Run("ArchiveWithPagination", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/archive?page=2", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		assertStatus(t, rec.Code, http.StatusOK)
	})

	// Test Search
	t.Run("SearchPage", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/search", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		assertStatus(t, rec.Code, http.StatusOK)
		body := rec.Body.String()
		assertBodyContains(t, body, "Search")
	})

	t.Run("SearchWithQuery", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/search?q=test", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		assertStatus(t, rec.Code, http.StatusOK)
		body := rec.Body.String()
		assertBodyContains(t, body, "test")
	})

	t.Run("SearchWithPagination", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/search?q=test&page=1", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		assertStatus(t, rec.Code, http.StatusOK)
	})

	// Test 404 Error Page
	t.Run("404ErrorPage", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/this-page-definitely-does-not-exist", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		assertStatus(t, rec.Code, http.StatusNotFound)
		body := rec.Body.String()
		assertBodyContains(t, body, "404")
		assertBodyContains(t, body, "page-error")
	})

	// Test RSS Feed
	t.Run("RSSFeed", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/feed", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		assertStatus(t, rec.Code, http.StatusOK)

		contentType := rec.Header().Get("Content-Type")
		if contentType != "application/rss+xml; charset=utf-8" {
			t.Errorf("expected Content-Type 'application/rss+xml; charset=utf-8', got %s", contentType)
		}

		body := rec.Body.String()
		assertBodyContains(t, body, "<?xml")
		assertBodyContains(t, body, "<rss")
		assertBodyContains(t, body, "<channel>")
	})

	// Test Theme Assets
	t.Run("ThemeCSS", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/theme/assets/css/style.css", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		assertStatus(t, rec.Code, http.StatusOK)

		body := rec.Body.String()
		assertBodyContains(t, body, ":root")
	})

	t.Run("ThemeJS", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/theme/assets/js/main.js", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		assertStatus(t, rec.Code, http.StatusOK)
	})
}

// TestSiteTemplateElements tests that key template elements are rendered
func TestSiteTemplateElements(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "site-elements-test-*")
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
	testUser, _, _ := server.users.Register(ctx, &users.RegisterIn{
		Email:    "elements@test.com",
		Password: "password123",
		Name:     "Elements Author",
	})
	seedSiteTestData(t, server, testUser.ID)

	t.Run("HomepageHasHeader", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		body := rec.Body.String()
		assertContains(t, body, "<header")
	})

	t.Run("HomepageHasFooter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		body := rec.Body.String()
		assertContains(t, body, "<footer")
	})

	t.Run("HomepageHasMain", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		body := rec.Body.String()
		assertContains(t, body, "<main")
		assertContains(t, body, "id=\"main-content\"")
	})

	t.Run("HomepageHasSkipLink", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		body := rec.Body.String()
		assertContains(t, body, "skip-link")
	})

	t.Run("PagesHaveMeta", func(t *testing.T) {
		pages := []string{"/", "/search", "/archive"}
		for _, page := range pages {
			t.Run(page, func(t *testing.T) {
				req := httptest.NewRequest("GET", page, nil)
				rec := httptest.NewRecorder()
				server.app.ServeHTTP(rec, req)

				if rec.Code != http.StatusOK {
					t.Errorf("expected status 200 for %s, got %d", page, rec.Code)
					return
				}

				body := rec.Body.String()
				assertContains(t, body, "<meta")
				assertContains(t, body, "viewport")
				assertContains(t, body, "charset")
			})
		}
	})

	t.Run("PagesHaveOpenGraphTags", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		body := rec.Body.String()
		assertContains(t, body, "og:site_name")
		assertContains(t, body, "og:type")
	})

	t.Run("PagesHaveTwitterCard", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		body := rec.Body.String()
		assertContains(t, body, "twitter:card")
	})

	t.Run("PagesHaveRSSLink", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		body := rec.Body.String()
		assertContains(t, body, "application/rss+xml")
		assertContains(t, body, "/feed")
	})

	t.Run("PagesLoadThemeStyles", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		body := rec.Body.String()
		assertContains(t, body, "/theme/assets/css/style.css")
	})

	t.Run("PagesLoadThemeScripts", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		body := rec.Body.String()
		assertContains(t, body, "/theme/assets/js/main.js")
	})
}

// TestSitePostTemplateElements tests single post page elements
func TestSitePostTemplateElements(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "site-post-test-*")
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
	testUser, _, _ := server.users.Register(ctx, &users.RegisterIn{
		Email:    "post-test@test.com",
		Password: "password123",
		Name:     "Post Test Author",
	})
	seedSiteTestData(t, server, testUser.ID)

	t.Run("PostHasArticleTag", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test-post", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		body := rec.Body.String()
		assertContains(t, body, "<article")
	})

	t.Run("PostHasTitle", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test-post", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		body := rec.Body.String()
		assertContains(t, body, "Test Post Title")
	})

	t.Run("PostHasContent", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test-post", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		body := rec.Body.String()
		assertContains(t, body, "Test post content")
	})

	t.Run("PostHasProperBodyClass", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test-post", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		body := rec.Body.String()
		assertContains(t, body, "page-post")
	})
}

// TestSitePageTemplateElements tests single page elements
func TestSitePageTemplateElements(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "site-page-test-*")
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
	testUser, _, _ := server.users.Register(ctx, &users.RegisterIn{
		Email:    "page-test@test.com",
		Password: "password123",
		Name:     "Page Test Author",
	})
	seedSiteTestData(t, server, testUser.ID)

	t.Run("PageHasArticleTag", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/page/about-us", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		body := rec.Body.String()
		assertContains(t, body, "<article")
	})

	t.Run("PageHasTitle", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/page/about-us", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		body := rec.Body.String()
		assertContains(t, body, "About Us")
	})

	t.Run("PageHasContent", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/page/about-us", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		body := rec.Body.String()
		assertContains(t, body, "This is the about page")
	})

	t.Run("PageHasProperBodyClass", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/page/about-us", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		body := rec.Body.String()
		assertContains(t, body, "page-page")
	})
}

// TestSiteArchiveTemplateElements tests archive page elements
func TestSiteArchiveTemplateElements(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "site-archive-test-*")
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
	testUser, _, _ := server.users.Register(ctx, &users.RegisterIn{
		Email:    "archive@test.com",
		Password: "password123",
		Name:     "Archive Author",
	})
	seedSiteTestData(t, server, testUser.ID)

	t.Run("CategoryArchiveHasHeader", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/category/technology", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		body := rec.Body.String()
		assertContains(t, body, "Technology")
	})

	t.Run("CategoryArchiveHasProperBodyClass", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/category/technology", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		body := rec.Body.String()
		assertContains(t, body, "page-category")
	})

	t.Run("TagArchiveHasTitle", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/tag/golang", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		body := rec.Body.String()
		assertContains(t, body, "golang")
	})

	t.Run("TagArchiveHasProperBodyClass", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/tag/golang", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		body := rec.Body.String()
		assertContains(t, body, "page-tag")
	})

	t.Run("AuthorArchiveHasName", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/author/archive-author", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		body := rec.Body.String()
		assertContains(t, body, "Archive Author")
	})

	t.Run("AuthorArchiveHasProperBodyClass", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/author/archive-author", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		body := rec.Body.String()
		assertContains(t, body, "page-author")
	})
}

// TestSiteSearchTemplateElements tests search page elements
func TestSiteSearchTemplateElements(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "site-search-test-*")
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
	testUser, _, _ := server.users.Register(ctx, &users.RegisterIn{
		Email:    "search@test.com",
		Password: "password123",
		Name:     "Search Author",
	})
	seedSiteTestData(t, server, testUser.ID)

	t.Run("SearchPageHasForm", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/search", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		body := rec.Body.String()
		assertContains(t, body, "<form")
		assertContains(t, body, "search")
	})

	t.Run("SearchPageHasProperBodyClass", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/search", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		body := rec.Body.String()
		assertContains(t, body, "page-search")
	})

	t.Run("SearchResultsShowQuery", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/search?q=test", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		body := rec.Body.String()
		assertContains(t, body, "test")
	})
}

// TestSiteErrorTemplateElements tests error page elements
func TestSiteErrorTemplateElements(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "site-error-test-*")
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

	t.Run("404PageHasErrorCode", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/definitely-not-found-page", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", rec.Code)
		}
		body := rec.Body.String()
		assertContains(t, body, "404")
	})

	t.Run("404PageHasBackLink", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/definitely-not-found-page", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		body := rec.Body.String()
		assertContains(t, body, "Back to Home")
	})

	t.Run("404PageHasProperBodyClass", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/definitely-not-found-page", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		body := rec.Body.String()
		assertContains(t, body, "page-error")
		assertContains(t, body, "error-404")
	})
}

// TestSiteRSSFeed tests RSS feed content
func TestSiteRSSFeed(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "site-rss-test-*")
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
	testUser, _, _ := server.users.Register(ctx, &users.RegisterIn{
		Email:    "rss@test.com",
		Password: "password123",
		Name:     "RSS Author",
	})
	seedSiteTestData(t, server, testUser.ID)

	t.Run("RSSFeedHasCorrectFormat", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/feed", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		body := rec.Body.String()
		assertContains(t, body, "<?xml version=\"1.0\"")
		assertContains(t, body, "<rss version=\"2.0\"")
		assertContains(t, body, "<channel>")
		assertContains(t, body, "<title>")
		assertContains(t, body, "<link>")
		assertContains(t, body, "</channel>")
		assertContains(t, body, "</rss>")
	})

	t.Run("RSSFeedHasItems", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/feed", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		body := rec.Body.String()
		assertContains(t, body, "<item>")
		assertContains(t, body, "Test Post Title")
	})

	t.Run("RSSFeedHasAtomLink", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/feed", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		body := rec.Body.String()
		assertContains(t, body, "atom:link")
	})
}

// TestSiteThemeConfig tests that theme configuration is applied
func TestSiteThemeConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "site-theme-test-*")
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

	t.Run("ThemeColorsApplied", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		body := rec.Body.String()
		// Check that CSS variables are set inline
		assertContains(t, body, "--color-primary")
	})

	t.Run("DarkModeSupport", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		body := rec.Body.String()
		// Check that dark mode data attribute is set
		assertContains(t, body, "data-theme")
		// Check dark mode CSS variables
		assertContains(t, body, "[data-theme=\"dark\"]")
	})

	t.Run("GoogleFontsLoaded", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		body := rec.Body.String()
		assertContains(t, body, "fonts.googleapis.com")
		assertContains(t, body, "Inter")
	})
}

// TestSiteStatusCodes specifically tests HTTP status codes for all endpoints
func TestSiteStatusCodes(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "site-status-test-*")
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
	testUser, _, err := server.users.Register(ctx, &users.RegisterIn{
		Email:    "status@test.com",
		Password: "password123",
		Name:     "Status Tester",
	})
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	seedSiteTestData(t, server, testUser.ID)

	// Test cases with expected status codes
	testCases := []struct {
		name           string
		path           string
		expectedStatus int
	}{
		// 200 OK - existing resources
		{"Homepage", "/", http.StatusOK},
		{"Archive", "/archive", http.StatusOK},
		{"Search", "/search", http.StatusOK},
		{"SearchWithQuery", "/search?q=test", http.StatusOK},
		{"Feed", "/feed", http.StatusOK},
		{"ExistingPost", "/test-post", http.StatusOK},
		{"ExistingPage", "/page/about-us", http.StatusOK},
		{"ExistingCategory", "/category/technology", http.StatusOK},
		{"ExistingTag", "/tag/golang", http.StatusOK},
		{"ExistingAuthor", "/author/status-tester", http.StatusOK},
		{"ThemeCSS", "/theme/assets/css/style.css", http.StatusOK},
		{"ThemeJS", "/theme/assets/js/main.js", http.StatusOK},

		// 404 Not Found - non-existing resources
		{"NonExistentPost", "/non-existent-post-slug", http.StatusNotFound},
		{"NonExistentPage", "/page/non-existent-page", http.StatusNotFound},
		{"NonExistentCategory", "/category/non-existent", http.StatusNotFound},
		{"NonExistentTag", "/tag/non-existent-tag", http.StatusNotFound},
		{"NonExistentAuthor", "/author/nobody", http.StatusNotFound},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.path, nil)
			rec := httptest.NewRecorder()
			server.app.ServeHTTP(rec, req)

			assertStatus(t, rec.Code, tc.expectedStatus)

			// For 404 responses, verify error page content
			if tc.expectedStatus == http.StatusNotFound {
				body := rec.Body.String()
				assertBodyContains(t, body, "404")
				assertBodyContains(t, body, "page-error")
			}
		})
	}
}

// TestSiteMarkdownRendering tests that markdown content is properly rendered to HTML
func TestSiteMarkdownRendering(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "site-markdown-test-*")
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
	testUser, _, err := server.users.Register(ctx, &users.RegisterIn{
		Email:    "markdown@test.com",
		Password: "password123",
		Name:     "Markdown Tester",
	})
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	// Create a post with markdown content
	markdownContent := `# Test Heading

This is a **bold** and *italic* text.

## Second Heading

- List item 1
- List item 2

` + "```go\nfunc main() {\n\tfmt.Println(\"Hello\")\n}\n```"

	post, err := server.posts.Create(ctx, testUser.ID, &posts.CreateIn{
		Title:         "Markdown Test Post",
		Slug:          "markdown-test",
		Content:       markdownContent,
		ContentFormat: "markdown",
		Status:        "published",
	})
	if err != nil {
		t.Fatalf("failed to create post: %v", err)
	}
	_ = post

	t.Run("MarkdownHeadingsRendered", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/markdown-test", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		assertStatus(t, rec.Code, http.StatusOK)
		body := rec.Body.String()
		// Check that markdown headings are rendered as HTML
		assertBodyContains(t, body, "<h1")
		assertBodyContains(t, body, "<h2")
		assertBodyContains(t, body, "Test Heading")
	})

	t.Run("MarkdownBoldItalicRendered", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/markdown-test", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		body := rec.Body.String()
		// Check that bold and italic are rendered
		assertBodyContains(t, body, "<strong>")
		assertBodyContains(t, body, "<em>")
	})

	t.Run("MarkdownListsRendered", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/markdown-test", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		body := rec.Body.String()
		// Check that lists are rendered
		assertBodyContains(t, body, "<ul>")
		assertBodyContains(t, body, "<li>")
	})

	t.Run("MarkdownCodeBlockRendered", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/markdown-test", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		body := rec.Body.String()
		// Check that code blocks are rendered (goldmark renders as <pre><code>)
		assertBodyContains(t, body, "<pre>")
		// Content of the code block should be there
		assertBodyContains(t, body, "fmt.Println")
	})
}

// TestSitePostCardElements tests that post cards on homepage have all required elements
func TestSitePostCardElements(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "site-postcard-test-*")
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
	testUser, _, err := server.users.Register(ctx, &users.RegisterIn{
		Email:    "postcard@test.com",
		Password: "password123",
		Name:     "Post Card Tester",
	})
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	seedSiteTestData(t, server, testUser.ID)

	t.Run("PostCardHasTitle", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		body := rec.Body.String()
		assertBodyContains(t, body, "post-card-title")
	})

	t.Run("PostCardHasExcerpt", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		body := rec.Body.String()
		assertBodyContains(t, body, "post-card-excerpt")
	})

	t.Run("PostCardHasAuthor", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		body := rec.Body.String()
		assertBodyContains(t, body, "post-card-author")
	})

	t.Run("PostCardHasFooter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		body := rec.Body.String()
		// Post card should have footer with info (date is conditional on PublishedAt)
		assertBodyContains(t, body, "post-card-footer")
	})

	t.Run("PostCardHasLink", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()
		server.app.ServeHTTP(rec, req)

		body := rec.Body.String()
		// Check post card has a link to the post
		assertBodyContains(t, body, "href=\"/test-post\"")
	})
}

// TestSiteTemplateIntegrity tests that all templates parse correctly
func TestSiteTemplateIntegrity(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "site-template-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Creating a server implicitly tests that all templates parse correctly
	server, err := New(Config{
		Addr:    ":0",
		DataDir: tmpDir,
	})
	if err != nil {
		t.Fatalf("server creation failed (template parsing error): %v", err)
	}

	// Test that basic requests don't cause template errors
	testPaths := []string{
		"/",
		"/search",
		"/archive",
		"/feed",
	}

	for _, path := range testPaths {
		t.Run("Path"+path, func(t *testing.T) {
			req := httptest.NewRequest("GET", path, nil)
			rec := httptest.NewRecorder()
			server.app.ServeHTTP(rec, req)

			// Should not return 500 (template error)
			if rec.Code == http.StatusInternalServerError {
				t.Errorf("template error for path %s: got status 500", path)
			}

			// Response should contain valid HTML
			body := rec.Body.String()
			if path != "/feed" {
				assertBodyContains(t, body, "<!DOCTYPE html>")
				assertBodyContains(t, body, "</html>")
			}
		})
	}
}

// TestSiteContentTypeHeaders tests that correct Content-Type headers are set
func TestSiteContentTypeHeaders(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "site-headers-test-*")
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

	testCases := []struct {
		name        string
		path        string
		contentType string
	}{
		{"Homepage", "/", "text/html; charset=utf-8"},
		{"Search", "/search", "text/html; charset=utf-8"},
		{"Feed", "/feed", "application/rss+xml; charset=utf-8"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.path, nil)
			rec := httptest.NewRecorder()
			server.app.ServeHTTP(rec, req)

			contentType := rec.Header().Get("Content-Type")
			if contentType != tc.contentType {
				t.Errorf("expected Content-Type %q, got %q", tc.contentType, contentType)
			}
		})
	}
}

// seedSiteTestData creates test data for site tests
func seedSiteTestData(t *testing.T, server *Server, userID string) {
	ctx := context.Background()

	// Create test category
	cat, _ := server.categories.Create(ctx, &categories.CreateIn{
		Name:        "Technology",
		Slug:        "technology",
		Description: "Tech related posts",
	})

	// Create test tag
	tag, _ := server.tags.Create(ctx, &tags.CreateIn{
		Name:        "golang",
		Slug:        "golang",
		Description: "Go programming language",
	})

	// Create test post
	catIDs := []string{}
	tagIDs := []string{}
	if cat != nil {
		catIDs = append(catIDs, cat.ID)
	}
	if tag != nil {
		tagIDs = append(tagIDs, tag.ID)
	}

	_, _ = server.posts.Create(ctx, userID, &posts.CreateIn{
		Title:       "Test Post Title",
		Slug:        "test-post",
		Content:     "<p>Test post content for e2e testing.</p>",
		Excerpt:     "A test post excerpt",
		Status:      "published",
		CategoryIDs: catIDs,
		TagIDs:      tagIDs,
	})

	// Create more posts for pagination testing
	for i := 1; i <= 15; i++ {
		_, _ = server.posts.Create(ctx, userID, &posts.CreateIn{
			Title:       "Additional Post " + string(rune('0'+i%10)),
			Content:     "<p>Content for additional post.</p>",
			Status:      "published",
			CategoryIDs: catIDs,
		})
	}

	// Create test page
	_, _ = server.pages.Create(ctx, userID, &pages.CreateIn{
		Title:   "About Us",
		Slug:    "about-us",
		Content: "<p>This is the about page content for testing.</p>",
		Status:  "published",
	})

	// Create another page
	_, _ = server.pages.Create(ctx, userID, &pages.CreateIn{
		Title:   "Contact",
		Slug:    "contact",
		Content: "<p>Contact us at test@example.com</p>",
		Status:  "published",
	})
}
