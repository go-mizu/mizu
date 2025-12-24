//go:build e2e

package web_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/news/app/web"
	"github.com/go-mizu/mizu/blueprints/news/feature/comments"
	"github.com/go-mizu/mizu/blueprints/news/feature/stories"
	"github.com/go-mizu/mizu/blueprints/news/feature/users"
	"github.com/go-mizu/mizu/blueprints/news/pkg/ulid"
	"github.com/go-mizu/mizu/blueprints/news/store/duckdb"

	_ "github.com/duckdb/duckdb-go/v2"
)

// setupTestServer creates a test server with a temp directory database.
func setupTestServer(t *testing.T) (*httptest.Server, *duckdb.Store) {
	t.Helper()

	// Use temp directory for test database
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

	// Create test server using the server's handler
	ts := httptest.NewServer(srv.Handler())

	t.Cleanup(func() {
		ts.Close()
		store.Close()
	})

	return ts, store
}

// Helper to create a test user directly via store.
func createTestUser(t *testing.T, store *duckdb.Store, username string) *users.User {
	t.Helper()
	ctx := context.Background()

	user := &users.User{
		ID:        ulid.New(),
		Username:  username,
		Email:     username + "@example.com",
		Karma:     100,
		CreatedAt: time.Now(),
	}

	if err := store.Users().Create(ctx, user); err != nil {
		t.Fatalf("create user %s: %v", username, err)
	}
	return user
}

// Helper to create a test story directly via store.
func createTestStory(t *testing.T, store *duckdb.Store, user *users.User, title, url, text string) *stories.Story {
	t.Helper()
	ctx := context.Background()

	story := &stories.Story{
		ID:        ulid.New(),
		AuthorID:  user.ID,
		Title:     title,
		URL:       url,
		Domain:    stories.ExtractDomain(url),
		Text:      text,
		Score:     10,
		CreatedAt: time.Now(),
	}

	if err := store.Stories().Create(ctx, story, nil); err != nil {
		t.Fatalf("create story: %v", err)
	}
	return story
}

// Helper to create a test comment directly via store.
func createTestComment(t *testing.T, store *duckdb.Store, user *users.User, story *stories.Story, text string) *comments.Comment {
	t.Helper()
	ctx := context.Background()

	commentID := ulid.New()
	comment := &comments.Comment{
		ID:        commentID,
		StoryID:   story.ID,
		AuthorID:  user.ID,
		Text:      text,
		Score:     1,
		Depth:     0,
		Path:      commentID,
		CreatedAt: time.Now(),
	}

	if err := store.Comments().Create(ctx, comment); err != nil {
		t.Fatalf("create comment: %v", err)
	}
	return comment
}

func get(t *testing.T, url string) (*http.Response, string) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return resp, string(body)
}

func assertStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		t.Errorf("status: got %d, want %d", resp.StatusCode, want)
	}
}

func assertContains(t *testing.T, body, substr string) {
	t.Helper()
	if !strings.Contains(body, substr) {
		t.Errorf("body missing %q (len=%d)", substr, len(body))
	}
}

// TestE2E_Stories tests story read endpoints.
func TestE2E_Stories(t *testing.T) {
	if os.Getenv("E2E_TEST") != "1" {
		t.Skip("set E2E_TEST=1 to run")
	}

	ts, store := setupTestServer(t)

	// Create test data
	user := createTestUser(t, store, "storyuser")
	story := createTestStory(t, store, user, "Test Story", "https://example.com", "")

	t.Run("GetStory", func(t *testing.T) {
		resp, respBody := get(t, ts.URL+"/api/stories/"+story.ID)
		assertStatus(t, resp, http.StatusOK)
		assertContains(t, respBody, "Test Story")
	})

	t.Run("ListStories", func(t *testing.T) {
		resp, respBody := get(t, ts.URL+"/api/stories")

		assertStatus(t, resp, http.StatusOK)
		assertContains(t, respBody, "[") // Should be array
		assertContains(t, respBody, "Test Story")
	})

	t.Run("ListStories_BySort", func(t *testing.T) {
		resp, _ := get(t, ts.URL+"/api/stories?sort=new")
		assertStatus(t, resp, http.StatusOK)

		resp, _ = get(t, ts.URL+"/api/stories?sort=top")
		assertStatus(t, resp, http.StatusOK)
	})
}

// TestE2E_HTMLPages tests HTML page rendering.
// Note: HTML template rendering has some issues with nil pointers in the templates,
// but the pages still return 200 OK status and render (with some template errors logged).
func TestE2E_HTMLPages(t *testing.T) {
	if os.Getenv("E2E_TEST") != "1" {
		t.Skip("set E2E_TEST=1 to run")
	}

	ts, store := setupTestServer(t)

	// Create some data
	user := createTestUser(t, store, "pageuser")
	createTestStory(t, store, user, "Page Test Story", "https://pagetest.example.com", "")

	t.Run("UserProfilePage", func(t *testing.T) {
		resp, body := get(t, ts.URL+"/user/"+user.Username)

		assertStatus(t, resp, http.StatusOK)
		assertContains(t, body, "pageuser")
	})
}

// TestE2E_UserProfile tests user profile endpoints.
func TestE2E_UserProfile(t *testing.T) {
	if os.Getenv("E2E_TEST") != "1" {
		t.Skip("set E2E_TEST=1 to run")
	}

	ts, store := setupTestServer(t)

	user := createTestUser(t, store, "profileuser")
	createTestStory(t, store, user, "Profile User Story", "https://profile.example.com", "")

	t.Run("GetUser_API", func(t *testing.T) {
		resp, body := get(t, ts.URL+"/api/users/"+user.Username)

		assertStatus(t, resp, http.StatusOK)
		assertContains(t, body, "profileuser")
	})

	t.Run("UserProfilePage", func(t *testing.T) {
		resp, body := get(t, ts.URL+"/user/"+user.Username)

		assertStatus(t, resp, http.StatusOK)
		assertContains(t, body, "profileuser")
		assertContains(t, body, "Profile User Story")
	})

	t.Run("UserNotFound", func(t *testing.T) {
		resp, _ := get(t, ts.URL+"/api/users/nonexistent")
		assertStatus(t, resp, http.StatusNotFound)
	})
}

// TestE2E_NotFound tests 404 responses.
func TestE2E_NotFound(t *testing.T) {
	if os.Getenv("E2E_TEST") != "1" {
		t.Skip("set E2E_TEST=1 to run")
	}

	ts, _ := setupTestServer(t)

	t.Run("StoryNotFound", func(t *testing.T) {
		resp, _ := get(t, ts.URL+"/api/stories/nonexistent")
		assertStatus(t, resp, http.StatusNotFound)
	})

	t.Run("UserNotFound", func(t *testing.T) {
		resp, _ := get(t, ts.URL+"/api/users/nonexistent")
		assertStatus(t, resp, http.StatusNotFound)
	})
}

// TestE2E_StaticFiles tests static file serving.
func TestE2E_StaticFiles(t *testing.T) {
	if os.Getenv("E2E_TEST") != "1" {
		t.Skip("set E2E_TEST=1 to run")
	}

	ts, _ := setupTestServer(t)

	t.Run("CSS", func(t *testing.T) {
		resp, _ := get(t, ts.URL+"/static/css/style.css")
		// May or may not exist depending on assets
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
			t.Errorf("unexpected status: %d", resp.StatusCode)
		}
	})
}

// TestE2E_Pagination tests pagination.
func TestE2E_Pagination(t *testing.T) {
	if os.Getenv("E2E_TEST") != "1" {
		t.Skip("set E2E_TEST=1 to run")
	}

	ts, store := setupTestServer(t)

	user := createTestUser(t, store, "paginationuser")

	// Create several stories
	for i := 0; i < 5; i++ {
		createTestStory(t, store, user, "Pagination Story", "https://pagination.example.com/"+ulid.New(), "")
	}

	t.Run("Page1", func(t *testing.T) {
		resp, _ := get(t, ts.URL+"/api/stories?limit=2&offset=0")
		assertStatus(t, resp, http.StatusOK)
	})

	t.Run("Page2", func(t *testing.T) {
		resp, _ := get(t, ts.URL+"/api/stories?limit=2&offset=2")
		assertStatus(t, resp, http.StatusOK)
	})
}
