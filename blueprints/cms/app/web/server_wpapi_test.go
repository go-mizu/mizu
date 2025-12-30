package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-mizu/blueprints/cms/app/web/handler/wpapi"
)

// testServer creates a test server with a temporary database.
func testServer(t *testing.T) (*Server, func()) {
	t.Helper()

	// Create temp directory for test data
	tempDir, err := os.MkdirTemp("", "wpapi-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	cfg := Config{
		Addr:    ":0",
		DataDir: tempDir,
		Dev:     true,
	}

	srv, err := New(cfg)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("failed to create server: %v", err)
	}

	cleanup := func() {
		srv.Close()
		os.RemoveAll(tempDir)
	}

	return srv, cleanup
}

// makeRequest performs an HTTP request against the test server.
func makeRequest(t *testing.T, srv *Server, method, path string, body any, sessionCookie string) *httptest.ResponseRecorder {
	t.Helper()

	var reqBody io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("failed to marshal body: %v", err)
		}
		reqBody = bytes.NewReader(jsonBytes)
	}

	req := httptest.NewRequest(method, path, reqBody)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if sessionCookie != "" {
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionCookie})
	}

	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	return rr
}

// registerUser registers a test user and returns the session cookie.
func registerUser(t *testing.T, srv *Server, email, password, name string) string {
	t.Helper()

	body := map[string]string{
		"email":    email,
		"password": password,
		"name":     name,
	}

	rr := makeRequest(t, srv, "POST", "/api/v1/auth/register", body, "")
	// Accept both 200 and 201 status codes
	if rr.Code != http.StatusOK && rr.Code != http.StatusCreated {
		t.Fatalf("failed to register user: %d - %s", rr.Code, rr.Body.String())
	}

	// Get session cookie
	for _, cookie := range rr.Result().Cookies() {
		if cookie.Name == "session" {
			return cookie.Value
		}
	}

	t.Fatal("no session cookie returned")
	return ""
}

// TestWPAPI_Discovery tests the API discovery endpoints.
func TestWPAPI_Discovery(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	t.Run("GET /wp-json", func(t *testing.T) {
		rr := makeRequest(t, srv, "GET", "/wp-json", nil, "")

		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}

		var discovery wpapi.WPDiscovery
		if err := json.NewDecoder(rr.Body).Decode(&discovery); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(discovery.Namespaces) == 0 {
			t.Error("expected namespaces to be populated")
		}

		if len(discovery.Routes) == 0 {
			t.Error("expected routes to be populated")
		}
	})

	t.Run("GET /wp-json/wp/v2", func(t *testing.T) {
		rr := makeRequest(t, srv, "GET", "/wp-json/wp/v2", nil, "")

		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}

		var result map[string]any
		if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if result["namespace"] != "wp/v2" {
			t.Errorf("expected namespace wp/v2, got %v", result["namespace"])
		}
	})
}

// TestWPAPI_Posts tests the posts endpoints.
func TestWPAPI_Posts(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	// Register user for authenticated requests
	session := registerUser(t, srv, "admin@test.com", "password123", "Admin User")

	var createdPostID int64

	t.Run("Create Post", func(t *testing.T) {
		body := map[string]any{
			"title":   "Test Post",
			"content": "<p>This is test content</p>",
			"excerpt": "Test excerpt",
			"status":  "publish",
		}

		rr := makeRequest(t, srv, "POST", "/wp-json/wp/v2/posts", body, session)

		if rr.Code != http.StatusCreated {
			t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, rr.Code, rr.Body.String())
		}

		var post wpapi.WPPost
		if err := json.NewDecoder(rr.Body).Decode(&post); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if post.Title.Rendered != "Test Post" {
			t.Errorf("expected title 'Test Post', got '%s'", post.Title.Rendered)
		}

		if post.Status != "publish" {
			t.Errorf("expected status 'publish', got '%s'", post.Status)
		}

		if post.Type != "post" {
			t.Errorf("expected type 'post', got '%s'", post.Type)
		}

		createdPostID = post.ID
	})

	t.Run("List Posts", func(t *testing.T) {
		rr := makeRequest(t, srv, "GET", "/wp-json/wp/v2/posts", nil, "")

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
		}

		// Check pagination headers
		total := rr.Header().Get("X-WP-Total")
		if total == "" {
			t.Error("expected X-WP-Total header")
		}

		totalPages := rr.Header().Get("X-WP-TotalPages")
		if totalPages == "" {
			t.Error("expected X-WP-TotalPages header")
		}

		var posts []wpapi.WPPost
		if err := json.NewDecoder(rr.Body).Decode(&posts); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(posts) == 0 {
			t.Error("expected at least one post")
		}
	})

	t.Run("Get Post", func(t *testing.T) {
		if createdPostID == 0 {
			t.Skip("no post created")
		}

		// Note: We use the ULID, not the numeric ID for our API
		// First get the list to find the actual ID
		rr := makeRequest(t, srv, "GET", "/wp-json/wp/v2/posts", nil, "")
		var posts []wpapi.WPPost
		json.NewDecoder(rr.Body).Decode(&posts)

		if len(posts) == 0 {
			t.Skip("no posts found")
		}

		// Get using slug filter since we can't easily map numeric to ULID
		rr = makeRequest(t, srv, "GET", "/wp-json/wp/v2/posts?slug=test-post", nil, "")

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
		}

		var filteredPosts []wpapi.WPPost
		if err := json.NewDecoder(rr.Body).Decode(&filteredPosts); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(filteredPosts) != 1 {
			t.Errorf("expected 1 post, got %d", len(filteredPosts))
		}
	})

	t.Run("Update Post", func(t *testing.T) {
		// Get post ID from the list
		rr := makeRequest(t, srv, "GET", "/wp-json/wp/v2/posts?slug=test-post", nil, "")
		var posts []wpapi.WPPost
		json.NewDecoder(rr.Body).Decode(&posts)

		if len(posts) == 0 {
			t.Skip("no posts found")
		}

		// We need the internal ID, let's use the REST API to get it
		listRR := makeRequest(t, srv, "GET", "/api/v1/posts", nil, session)
		var restResp struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		json.NewDecoder(listRR.Body).Decode(&restResp)

		if len(restResp.Data) == 0 {
			t.Skip("no posts in REST API")
		}

		postID := restResp.Data[0].ID

		body := map[string]any{
			"title": "Updated Post Title",
		}

		rr = makeRequest(t, srv, "POST", "/wp-json/wp/v2/posts/"+postID, body, session)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
		}

		var post wpapi.WPPost
		if err := json.NewDecoder(rr.Body).Decode(&post); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if post.Title.Rendered != "Updated Post Title" {
			t.Errorf("expected title 'Updated Post Title', got '%s'", post.Title.Rendered)
		}
	})

	t.Run("Unauthorized Create", func(t *testing.T) {
		body := map[string]any{
			"title":   "Unauthorized Post",
			"content": "Content",
		}

		rr := makeRequest(t, srv, "POST", "/wp-json/wp/v2/posts", body, "")

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
		}

		var wpErr wpapi.WPError
		if err := json.NewDecoder(rr.Body).Decode(&wpErr); err != nil {
			t.Fatalf("failed to decode error: %v", err)
		}

		if wpErr.Code != "rest_not_logged_in" {
			t.Errorf("expected error code 'rest_not_logged_in', got '%s'", wpErr.Code)
		}
	})
}

// TestWPAPI_Pages tests the pages endpoints.
func TestWPAPI_Pages(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	session := registerUser(t, srv, "admin@test.com", "password123", "Admin User")

	t.Run("Create Page", func(t *testing.T) {
		body := map[string]any{
			"title":   "About Us",
			"content": "<p>About us page content</p>",
			"status":  "publish",
		}

		rr := makeRequest(t, srv, "POST", "/wp-json/wp/v2/pages", body, session)

		if rr.Code != http.StatusCreated {
			t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, rr.Code, rr.Body.String())
		}

		var page wpapi.WPPage
		if err := json.NewDecoder(rr.Body).Decode(&page); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if page.Title.Rendered != "About Us" {
			t.Errorf("expected title 'About Us', got '%s'", page.Title.Rendered)
		}

		if page.Type != "page" {
			t.Errorf("expected type 'page', got '%s'", page.Type)
		}
	})

	t.Run("List Pages", func(t *testing.T) {
		rr := makeRequest(t, srv, "GET", "/wp-json/wp/v2/pages", nil, "")

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
		}

		var pages []wpapi.WPPage
		if err := json.NewDecoder(rr.Body).Decode(&pages); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(pages) == 0 {
			t.Error("expected at least one page")
		}
	})
}

// TestWPAPI_Users tests the users endpoints.
func TestWPAPI_Users(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	session := registerUser(t, srv, "admin@test.com", "password123", "Admin User")

	t.Run("List Users", func(t *testing.T) {
		rr := makeRequest(t, srv, "GET", "/wp-json/wp/v2/users", nil, "")

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
		}

		var users []wpapi.WPUser
		if err := json.NewDecoder(rr.Body).Decode(&users); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(users) == 0 {
			t.Error("expected at least one user")
		}
	})

	t.Run("Get Current User", func(t *testing.T) {
		rr := makeRequest(t, srv, "GET", "/wp-json/wp/v2/users/me", nil, session)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
		}

		var user wpapi.WPUser
		if err := json.NewDecoder(rr.Body).Decode(&user); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if user.Name != "Admin User" {
			t.Errorf("expected name 'Admin User', got '%s'", user.Name)
		}

		// Check that avatar URLs are populated
		if len(user.AvatarURLs) == 0 {
			t.Error("expected avatar URLs to be populated")
		}
	})

	t.Run("Update Current User", func(t *testing.T) {
		body := map[string]any{
			"name":        "Updated Admin",
			"description": "Updated bio",
		}

		rr := makeRequest(t, srv, "POST", "/wp-json/wp/v2/users/me", body, session)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
		}

		var user wpapi.WPUser
		if err := json.NewDecoder(rr.Body).Decode(&user); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if user.Name != "Updated Admin" {
			t.Errorf("expected name 'Updated Admin', got '%s'", user.Name)
		}
	})

	t.Run("Unauthorized Current User", func(t *testing.T) {
		rr := makeRequest(t, srv, "GET", "/wp-json/wp/v2/users/me", nil, "")

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
		}
	})
}

// TestWPAPI_Categories tests the categories endpoints.
func TestWPAPI_Categories(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	session := registerUser(t, srv, "admin@test.com", "password123", "Admin User")

	var createdCategorySlug string

	t.Run("Create Category", func(t *testing.T) {
		body := map[string]any{
			"name":        "Technology",
			"description": "Tech articles",
		}

		rr := makeRequest(t, srv, "POST", "/wp-json/wp/v2/categories", body, session)

		if rr.Code != http.StatusCreated {
			t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, rr.Code, rr.Body.String())
		}

		var category wpapi.WPCategory
		if err := json.NewDecoder(rr.Body).Decode(&category); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if category.Name != "Technology" {
			t.Errorf("expected name 'Technology', got '%s'", category.Name)
		}

		if category.Taxonomy != "category" {
			t.Errorf("expected taxonomy 'category', got '%s'", category.Taxonomy)
		}

		createdCategorySlug = category.Slug
	})

	t.Run("List Categories", func(t *testing.T) {
		rr := makeRequest(t, srv, "GET", "/wp-json/wp/v2/categories", nil, "")

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
		}

		var categories []wpapi.WPCategory
		if err := json.NewDecoder(rr.Body).Decode(&categories); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(categories) == 0 {
			t.Error("expected at least one category")
		}
	})

	t.Run("Get Category by Slug", func(t *testing.T) {
		if createdCategorySlug == "" {
			t.Skip("no category created")
		}

		rr := makeRequest(t, srv, "GET", "/wp-json/wp/v2/categories?slug="+createdCategorySlug, nil, "")

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
		}

		var categories []wpapi.WPCategory
		if err := json.NewDecoder(rr.Body).Decode(&categories); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(categories) != 1 {
			t.Errorf("expected 1 category, got %d", len(categories))
		}
	})
}

// TestWPAPI_Tags tests the tags endpoints.
func TestWPAPI_Tags(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	session := registerUser(t, srv, "admin@test.com", "password123", "Admin User")

	t.Run("Create Tag", func(t *testing.T) {
		body := map[string]any{
			"name":        "golang",
			"description": "Go programming language",
		}

		rr := makeRequest(t, srv, "POST", "/wp-json/wp/v2/tags", body, session)

		if rr.Code != http.StatusCreated {
			t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, rr.Code, rr.Body.String())
		}

		var tag wpapi.WPTag
		if err := json.NewDecoder(rr.Body).Decode(&tag); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if tag.Name != "golang" {
			t.Errorf("expected name 'golang', got '%s'", tag.Name)
		}

		if tag.Taxonomy != "post_tag" {
			t.Errorf("expected taxonomy 'post_tag', got '%s'", tag.Taxonomy)
		}
	})

	t.Run("List Tags", func(t *testing.T) {
		rr := makeRequest(t, srv, "GET", "/wp-json/wp/v2/tags", nil, "")

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
		}

		var tags []wpapi.WPTag
		if err := json.NewDecoder(rr.Body).Decode(&tags); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(tags) == 0 {
			t.Error("expected at least one tag")
		}
	})
}

// TestWPAPI_Comments tests the comments endpoints.
func TestWPAPI_Comments(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	session := registerUser(t, srv, "admin@test.com", "password123", "Admin User")

	// Create a post first
	postBody := map[string]any{
		"title":   "Post with Comments",
		"content": "Content",
		"status":  "publish",
	}
	rr := makeRequest(t, srv, "POST", "/wp-json/wp/v2/posts", postBody, session)
	if rr.Code != http.StatusCreated {
		t.Fatalf("failed to create post: %s", rr.Body.String())
	}

	// Get post ID from WP API response (numeric ID)
	var wpPost wpapi.WPPost
	if err := json.NewDecoder(rr.Body).Decode(&wpPost); err != nil {
		t.Fatalf("failed to decode post: %v", err)
	}
	postID := wpPost.ID // This is the numeric ID

	t.Run("Create Comment", func(t *testing.T) {
		body := map[string]any{
			"post":         postID,
			"content":      "Great post!",
			"author_name":  "John Doe",
			"author_email": "john@example.com",
		}

		rr := makeRequest(t, srv, "POST", "/wp-json/wp/v2/comments", body, "")

		if rr.Code != http.StatusCreated {
			t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, rr.Code, rr.Body.String())
		}

		var comment wpapi.WPComment
		if err := json.NewDecoder(rr.Body).Decode(&comment); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if comment.AuthorName != "John Doe" {
			t.Errorf("expected author name 'John Doe', got '%s'", comment.AuthorName)
		}

		if comment.Type != "comment" {
			t.Errorf("expected type 'comment', got '%s'", comment.Type)
		}
	})

	t.Run("List Comments", func(t *testing.T) {
		rr := makeRequest(t, srv, "GET", "/wp-json/wp/v2/comments", nil, session)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
		}

		var comments []wpapi.WPComment
		if err := json.NewDecoder(rr.Body).Decode(&comments); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		// Note: Comments may be in pending status, so list may be empty for unauthenticated
	})
}

// TestWPAPI_Media tests the media endpoints.
func TestWPAPI_Media(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	session := registerUser(t, srv, "admin@test.com", "password123", "Admin User")

	t.Run("Upload Media", func(t *testing.T) {
		// Create a test image file
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)

		// Create a simple 1x1 pixel PNG
		pngData := []byte{
			0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D,
			0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
			0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xDE, 0x00, 0x00, 0x00,
			0x0C, 0x49, 0x44, 0x41, 0x54, 0x08, 0xD7, 0x63, 0xF8, 0xFF, 0xFF, 0x3F,
			0x00, 0x05, 0xFE, 0x02, 0xFE, 0xDC, 0xCC, 0x59, 0xE7, 0x00, 0x00, 0x00,
			0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82,
		}

		part, err := writer.CreateFormFile("file", "test.png")
		if err != nil {
			t.Fatalf("failed to create form file: %v", err)
		}
		part.Write(pngData)

		writer.WriteField("alt_text", "Test image")
		writer.WriteField("caption", "A test caption")
		writer.Close()

		req := httptest.NewRequest("POST", "/wp-json/wp/v2/media", &buf)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.AddCookie(&http.Cookie{Name: "session", Value: session})

		rr := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, rr.Code, rr.Body.String())
		}

		var media wpapi.WPMedia
		if err := json.NewDecoder(rr.Body).Decode(&media); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if media.Type != "attachment" {
			t.Errorf("expected type 'attachment', got '%s'", media.Type)
		}

		if media.MimeType != "image/png" {
			t.Errorf("expected mime type 'image/png', got '%s'", media.MimeType)
		}

		if media.AltText != "Test image" {
			t.Errorf("expected alt text 'Test image', got '%s'", media.AltText)
		}
	})

	t.Run("List Media", func(t *testing.T) {
		rr := makeRequest(t, srv, "GET", "/wp-json/wp/v2/media", nil, session)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
		}

		var media []wpapi.WPMedia
		if err := json.NewDecoder(rr.Body).Decode(&media); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(media) == 0 {
			t.Error("expected at least one media item")
		}
	})
}

// TestWPAPI_Settings tests the settings endpoint.
func TestWPAPI_Settings(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	session := registerUser(t, srv, "admin@test.com", "password123", "Admin User")

	t.Run("Get Settings", func(t *testing.T) {
		rr := makeRequest(t, srv, "GET", "/wp-json/wp/v2/settings", nil, session)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
		}

		var settings wpapi.WPSettings
		if err := json.NewDecoder(rr.Body).Decode(&settings); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		// Check default values
		if settings.PostsPerPage == 0 {
			t.Error("expected posts_per_page to have a value")
		}
	})

	t.Run("Update Settings", func(t *testing.T) {
		body := map[string]any{
			"title":       "My Test Site",
			"description": "A test site description",
		}

		rr := makeRequest(t, srv, "POST", "/wp-json/wp/v2/settings", body, session)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
		}

		var settings wpapi.WPSettings
		if err := json.NewDecoder(rr.Body).Decode(&settings); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if settings.Title != "My Test Site" {
			t.Errorf("expected title 'My Test Site', got '%s'", settings.Title)
		}
	})

	t.Run("Unauthorized Settings", func(t *testing.T) {
		rr := makeRequest(t, srv, "GET", "/wp-json/wp/v2/settings", nil, "")

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
		}
	})
}

// TestWPAPI_FullWorkflow tests a complete WordPress-style workflow.
func TestWPAPI_FullWorkflow(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	// 1. Register and login
	session := registerUser(t, srv, "blogger@test.com", "password123", "Test Blogger")

	// 2. Create a category
	categoryBody := map[string]any{
		"name":        "Development",
		"description": "Software development articles",
	}
	rr := makeRequest(t, srv, "POST", "/wp-json/wp/v2/categories", categoryBody, session)
	if rr.Code != http.StatusCreated {
		t.Fatalf("failed to create category: %s", rr.Body.String())
	}

	// Get category from REST API
	catListRR := makeRequest(t, srv, "GET", "/api/v1/categories", nil, session)
	var catRestResp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.NewDecoder(catListRR.Body).Decode(&catRestResp)

	if len(catRestResp.Data) == 0 {
		t.Fatal("no categories found")
	}
	categoryID := catRestResp.Data[0].ID

	// 3. Create a tag
	tagBody := map[string]any{
		"name":        "tutorial",
		"description": "Tutorial posts",
	}
	rr = makeRequest(t, srv, "POST", "/wp-json/wp/v2/tags", tagBody, session)
	if rr.Code != http.StatusCreated {
		t.Fatalf("failed to create tag: %s", rr.Body.String())
	}

	// Get tag from REST API
	tagListRR := makeRequest(t, srv, "GET", "/api/v1/tags", nil, session)
	var tagRestResp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.NewDecoder(tagListRR.Body).Decode(&tagRestResp)

	if len(tagRestResp.Data) == 0 {
		t.Fatal("no tags found")
	}
	tagID := tagRestResp.Data[0].ID

	// 4. Create a post with category and tag
	postBody := map[string]any{
		"title":   "Getting Started with Go",
		"content": "<p>This is a comprehensive guide to Go programming.</p>",
		"excerpt": "Learn Go programming from scratch",
		"status":  "publish",
	}
	rr = makeRequest(t, srv, "POST", "/wp-json/wp/v2/posts", postBody, session)
	if rr.Code != http.StatusCreated {
		t.Fatalf("failed to create post: %s", rr.Body.String())
	}

	var createdPost wpapi.WPPost
	json.NewDecoder(rr.Body).Decode(&createdPost)

	// Get post internal ID from REST API for update
	postListRR := makeRequest(t, srv, "GET", "/api/v1/posts", nil, session)
	var postRestResp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.NewDecoder(postListRR.Body).Decode(&postRestResp)

	if len(postRestResp.Data) == 0 {
		t.Fatal("no posts found")
	}
	postInternalID := postRestResp.Data[0].ID

	// 5. Update post with categories/tags through REST API
	updateBody := map[string]any{
		"category_ids": []string{categoryID},
		"tag_ids":      []string{tagID},
	}
	rr = makeRequest(t, srv, "PUT", "/api/v1/posts/"+postInternalID, updateBody, session)
	if rr.Code != http.StatusOK {
		t.Logf("Note: category/tag assignment returned: %s", rr.Body.String())
	}

	// 6. Add a comment to the post (use WP API numeric ID)
	commentBody := map[string]any{
		"post":         createdPost.ID,
		"content":      "This is very helpful, thank you!",
		"author_name":  "Reader",
		"author_email": "reader@example.com",
	}
	rr = makeRequest(t, srv, "POST", "/wp-json/wp/v2/comments", commentBody, "")
	if rr.Code != http.StatusCreated {
		t.Fatalf("failed to create comment: %s", rr.Body.String())
	}

	// 7. Verify the post appears in listings
	rr = makeRequest(t, srv, "GET", "/wp-json/wp/v2/posts?slug=getting-started-with-go", nil, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("failed to list posts: %s", rr.Body.String())
	}

	var posts []wpapi.WPPost
	json.NewDecoder(rr.Body).Decode(&posts)

	if len(posts) != 1 {
		t.Errorf("expected 1 post, got %d", len(posts))
	}

	// 8. Verify content is correct
	if len(posts) > 0 {
		post := posts[0]
		if post.Title.Rendered != "Getting Started with Go" {
			t.Errorf("unexpected title: %s", post.Title.Rendered)
		}
		if post.Status != "publish" {
			t.Errorf("unexpected status: %s", post.Status)
		}
	}

	// 9. Create a page
	pageBody := map[string]any{
		"title":   "Contact Us",
		"content": "<p>Contact information here</p>",
		"status":  "publish",
	}
	rr = makeRequest(t, srv, "POST", "/wp-json/wp/v2/pages", pageBody, session)
	if rr.Code != http.StatusCreated {
		t.Fatalf("failed to create page: %s", rr.Body.String())
	}

	// 10. Verify page
	rr = makeRequest(t, srv, "GET", "/wp-json/wp/v2/pages?slug=contact-us", nil, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("failed to list pages: %s", rr.Body.String())
	}

	var pages []wpapi.WPPage
	json.NewDecoder(rr.Body).Decode(&pages)

	if len(pages) != 1 {
		t.Errorf("expected 1 page, got %d", len(pages))
	}

	t.Log("Full workflow test completed successfully")
}

// TestWPAPI_ErrorFormats tests that errors are returned in WordPress format.
func TestWPAPI_ErrorFormats(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	t.Run("404 Not Found", func(t *testing.T) {
		rr := makeRequest(t, srv, "GET", "/wp-json/wp/v2/posts/nonexistent", nil, "")

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, rr.Code)
		}

		var wpErr wpapi.WPError
		if err := json.NewDecoder(rr.Body).Decode(&wpErr); err != nil {
			t.Fatalf("failed to decode error: %v", err)
		}

		if wpErr.Code == "" {
			t.Error("expected error code to be set")
		}

		if wpErr.Data.Status != 404 {
			t.Errorf("expected error status 404, got %d", wpErr.Data.Status)
		}
	})

	t.Run("401 Unauthorized", func(t *testing.T) {
		rr := makeRequest(t, srv, "POST", "/wp-json/wp/v2/posts", map[string]any{"title": "Test"}, "")

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
		}

		var wpErr wpapi.WPError
		if err := json.NewDecoder(rr.Body).Decode(&wpErr); err != nil {
			t.Fatalf("failed to decode error: %v", err)
		}

		if wpErr.Code != "rest_not_logged_in" {
			t.Errorf("expected error code 'rest_not_logged_in', got '%s'", wpErr.Code)
		}

		if wpErr.Data.Status != 401 {
			t.Errorf("expected error status 401, got %d", wpErr.Data.Status)
		}
	})

	t.Run("400 Bad Request", func(t *testing.T) {
		session := registerUser(t, srv, "admin@test.com", "password123", "Admin")

		// Create post without required title
		rr := makeRequest(t, srv, "POST", "/wp-json/wp/v2/posts", map[string]any{"content": "no title"}, session)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
		}

		var wpErr wpapi.WPError
		if err := json.NewDecoder(rr.Body).Decode(&wpErr); err != nil {
			t.Fatalf("failed to decode error: %v", err)
		}

		if wpErr.Data.Status != 400 {
			t.Errorf("expected error status 400, got %d", wpErr.Data.Status)
		}
	})
}

// TestWPAPI_Pagination tests pagination behavior.
func TestWPAPI_Pagination(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	session := registerUser(t, srv, "admin@test.com", "password123", "Admin")

	// Create multiple posts
	for i := 1; i <= 15; i++ {
		body := map[string]any{
			"title":   fmt.Sprintf("Post %d", i),
			"content": "Content",
			"status":  "publish",
		}
		rr := makeRequest(t, srv, "POST", "/wp-json/wp/v2/posts", body, session)
		if rr.Code != http.StatusCreated {
			t.Fatalf("failed to create post %d: %s", i, rr.Body.String())
		}
	}

	t.Run("Default Pagination", func(t *testing.T) {
		rr := makeRequest(t, srv, "GET", "/wp-json/wp/v2/posts", nil, "")

		total := rr.Header().Get("X-WP-Total")
		if total != "15" {
			t.Errorf("expected X-WP-Total '15', got '%s'", total)
		}

		totalPages := rr.Header().Get("X-WP-TotalPages")
		if totalPages != "2" {
			t.Errorf("expected X-WP-TotalPages '2', got '%s'", totalPages)
		}

		var posts []wpapi.WPPost
		json.NewDecoder(rr.Body).Decode(&posts)

		if len(posts) != 10 { // Default per_page is 10
			t.Errorf("expected 10 posts, got %d", len(posts))
		}
	})

	t.Run("Custom Per Page", func(t *testing.T) {
		rr := makeRequest(t, srv, "GET", "/wp-json/wp/v2/posts?per_page=5", nil, "")

		totalPages := rr.Header().Get("X-WP-TotalPages")
		if totalPages != "3" {
			t.Errorf("expected X-WP-TotalPages '3', got '%s'", totalPages)
		}

		var posts []wpapi.WPPost
		json.NewDecoder(rr.Body).Decode(&posts)

		if len(posts) != 5 {
			t.Errorf("expected 5 posts, got %d", len(posts))
		}
	})

	t.Run("Page 2", func(t *testing.T) {
		rr := makeRequest(t, srv, "GET", "/wp-json/wp/v2/posts?per_page=5&page=2", nil, "")

		var posts []wpapi.WPPost
		json.NewDecoder(rr.Body).Decode(&posts)

		if len(posts) != 5 {
			t.Errorf("expected 5 posts on page 2, got %d", len(posts))
		}
	})

	t.Run("Max Per Page Limit", func(t *testing.T) {
		// WordPress limits per_page to 100
		rr := makeRequest(t, srv, "GET", "/wp-json/wp/v2/posts?per_page=200", nil, "")

		var posts []wpapi.WPPost
		json.NewDecoder(rr.Body).Decode(&posts)

		if len(posts) > 100 {
			t.Errorf("expected at most 100 posts (per_page limit), got %d", len(posts))
		}
	})
}

// TestWPAPI_ResponseFormat tests that responses match WordPress format.
func TestWPAPI_ResponseFormat(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	session := registerUser(t, srv, "admin@test.com", "password123", "Admin")

	// Create a post
	body := map[string]any{
		"title":   "Format Test",
		"content": "<p>Testing format</p>",
		"status":  "publish",
	}
	rr := makeRequest(t, srv, "POST", "/wp-json/wp/v2/posts", body, session)

	var post wpapi.WPPost
	json.NewDecoder(rr.Body).Decode(&post)

	t.Run("Has Required Fields", func(t *testing.T) {
		if post.ID == 0 {
			t.Error("expected ID to be set")
		}
		if post.Date == "" {
			t.Error("expected date to be set")
		}
		if post.DateGMT == "" {
			t.Error("expected date_gmt to be set")
		}
		if post.Slug == "" {
			t.Error("expected slug to be set")
		}
		if post.Type != "post" {
			t.Errorf("expected type 'post', got '%s'", post.Type)
		}
		if post.Link == "" {
			t.Error("expected link to be set")
		}
	})

	t.Run("Has Rendered Fields", func(t *testing.T) {
		if post.Title.Rendered == "" {
			t.Error("expected title.rendered to be set")
		}
		if post.Content.Rendered == "" {
			t.Error("expected content.rendered to be set")
		}
		if post.GUID.Rendered == "" {
			t.Error("expected guid.rendered to be set")
		}
	})

	t.Run("Has _links", func(t *testing.T) {
		if post.Links == nil {
			t.Error("expected _links to be set")
		}
		if _, ok := post.Links["self"]; !ok {
			t.Error("expected _links.self to be set")
		}
		if _, ok := post.Links["collection"]; !ok {
			t.Error("expected _links.collection to be set")
		}
	})

	t.Run("Categories and Tags are Arrays", func(t *testing.T) {
		if post.Categories == nil {
			t.Error("expected categories to be array, got nil")
		}
		if post.Tags == nil {
			t.Error("expected tags to be array, got nil")
		}
	})

	t.Run("Meta is Array", func(t *testing.T) {
		if post.Meta == nil {
			t.Error("expected meta to be array, got nil")
		}
	})
}

// TestWPAPI_ContextParameter tests the context query parameter.
func TestWPAPI_ContextParameter(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	session := registerUser(t, srv, "admin@test.com", "password123", "Admin")

	// Create a post
	body := map[string]any{
		"title":   "Context Test",
		"content": "<p>Testing context</p>",
		"status":  "publish",
	}
	makeRequest(t, srv, "POST", "/wp-json/wp/v2/posts", body, session)

	t.Run("Edit Context Shows Raw", func(t *testing.T) {
		rr := makeRequest(t, srv, "GET", "/wp-json/wp/v2/posts?context=edit", nil, session)

		var posts []wpapi.WPPost
		json.NewDecoder(rr.Body).Decode(&posts)

		if len(posts) == 0 {
			t.Skip("no posts found")
		}

		post := posts[0]
		if post.Title.Raw == "" {
			t.Error("expected title.raw in edit context")
		}
		if post.Content.Raw == "" {
			t.Error("expected content.raw in edit context")
		}
	})

	t.Run("View Context Omits Raw", func(t *testing.T) {
		rr := makeRequest(t, srv, "GET", "/wp-json/wp/v2/posts?context=view", nil, "")

		var posts []wpapi.WPPost
		json.NewDecoder(rr.Body).Decode(&posts)

		if len(posts) == 0 {
			t.Skip("no posts found")
		}

		post := posts[0]
		if post.Title.Raw != "" {
			t.Error("expected title.raw to be empty in view context")
		}
		if post.Content.Raw != "" {
			t.Error("expected content.raw to be empty in view context")
		}
	})
}

// TestWPAPI_StatusFiltering tests filtering by post status.
func TestWPAPI_StatusFiltering(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	session := registerUser(t, srv, "admin@test.com", "password123", "Admin")

	// Create published post
	makeRequest(t, srv, "POST", "/wp-json/wp/v2/posts", map[string]any{
		"title":  "Published Post",
		"status": "publish",
	}, session)

	// Create draft post
	makeRequest(t, srv, "POST", "/wp-json/wp/v2/posts", map[string]any{
		"title":  "Draft Post",
		"status": "draft",
	}, session)

	t.Run("Unauthenticated Only Sees Published", func(t *testing.T) {
		rr := makeRequest(t, srv, "GET", "/wp-json/wp/v2/posts", nil, "")

		var posts []wpapi.WPPost
		json.NewDecoder(rr.Body).Decode(&posts)

		for _, post := range posts {
			if post.Status != "publish" {
				t.Errorf("unauthenticated user saw non-published post: %s", post.Status)
			}
		}
	})

	t.Run("Authenticated Can See Drafts", func(t *testing.T) {
		rr := makeRequest(t, srv, "GET", "/wp-json/wp/v2/posts?status=draft", nil, session)

		var posts []wpapi.WPPost
		json.NewDecoder(rr.Body).Decode(&posts)

		hasDraft := false
		for _, post := range posts {
			if post.Status == "draft" {
				hasDraft = true
				break
			}
		}

		if !hasDraft {
			t.Error("expected to see draft posts when authenticated")
		}
	})
}

// TestWPAPI_RealDatabase tests with the real database path.
func TestWPAPI_RealDatabase(t *testing.T) {
	// Skip if HOME is not set
	home := os.Getenv("HOME")
	if home == "" {
		t.Skip("HOME not set")
	}

	dataDir := filepath.Join(home, "data", "blueprint", "cms")

	// Check if database exists
	dbPath := filepath.Join(dataDir, "cms.duckdb")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Skip("real database not found at " + dbPath)
	}

	// Use a copy to avoid modifying real data
	tempDir, err := os.MkdirTemp("", "wpapi-realdb-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Copy database file
	srcDB, err := os.ReadFile(dbPath)
	if err != nil {
		t.Fatalf("failed to read database: %v", err)
	}
	dstPath := filepath.Join(tempDir, "cms.duckdb")
	if err := os.WriteFile(dstPath, srcDB, 0644); err != nil {
		t.Fatalf("failed to write database copy: %v", err)
	}

	// Create uploads dir
	os.MkdirAll(filepath.Join(tempDir, "uploads"), 0755)

	cfg := Config{
		Addr:    ":0",
		DataDir: tempDir,
		Dev:     true,
	}

	srv, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer srv.Close()

	t.Run("Discovery Works", func(t *testing.T) {
		rr := makeRequest(t, srv, "GET", "/wp-json", nil, "")

		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("List Existing Posts", func(t *testing.T) {
		rr := makeRequest(t, srv, "GET", "/wp-json/wp/v2/posts", nil, "")

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
		}

		total := rr.Header().Get("X-WP-Total")
		t.Logf("Found %s posts in real database", total)
	})

	t.Run("List Existing Users", func(t *testing.T) {
		rr := makeRequest(t, srv, "GET", "/wp-json/wp/v2/users", nil, "")

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
		}

		total := rr.Header().Get("X-WP-Total")
		t.Logf("Found %s users in real database", total)
	})

	t.Run("List Existing Categories", func(t *testing.T) {
		rr := makeRequest(t, srv, "GET", "/wp-json/wp/v2/categories", nil, "")

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
		}

		total := rr.Header().Get("X-WP-Total")
		t.Logf("Found %s categories in real database", total)
	})
}
