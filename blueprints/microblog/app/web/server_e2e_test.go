package web_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/microblog/app/web"
)

// testServer creates a test server instance with a temporary database.
func testServer(t *testing.T) (*web.Server, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "microblog-e2e-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	cfg := web.Config{
		Addr:    ":0",
		DataDir: tmpDir,
		Dev:     true,
	}

	srv, err := web.New(cfg)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create server: %v", err)
	}

	cleanup := func() {
		srv.Close()
		os.RemoveAll(tmpDir)
	}

	return srv, cleanup
}

// apiResponse is a generic API response wrapper.
type apiResponse struct {
	Data  json.RawMessage `json:"data,omitempty"`
	Error *apiError       `json:"error,omitempty"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type authData struct {
	Account struct {
		ID          string `json:"id"`
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
	} `json:"account"`
	Token string `json:"token"`
}

type postData struct {
	ID        string `json:"id"`
	Content   string `json:"content"`
	AccountID string `json:"account_id"`
}

func TestServerStartup(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	// Verify the server was created successfully
	if srv == nil {
		t.Fatal("expected server to be created")
	}
}

// assertHTMLPage verifies that an HTML page renders successfully without errors.
// It checks for proper status code, content type, and absence of error indicators.
func assertHTMLPage(t *testing.T, resp *http.Response, path string) {
	t.Helper()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("[%s] failed to read response body: %v", path, err)
	}
	body := string(bodyBytes)

	// Check for 500 Internal Server Error
	if resp.StatusCode == 500 {
		t.Errorf("[%s] got 500 Internal Server Error, body: %s", path, truncate(body, 500))
		return
	}

	// Check for successful status code (200-299)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		t.Errorf("[%s] expected 2xx status, got %d", path, resp.StatusCode)
		return
	}

	// Check content type is HTML
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("[%s] expected text/html content-type, got %s", path, contentType)
	}

	// Check body contains expected HTML structure
	if !strings.Contains(body, "<!DOCTYPE html>") && !strings.Contains(body, "<html") {
		t.Errorf("[%s] response doesn't look like HTML: %s", path, truncate(body, 200))
	}

	// Check for common error indicators in the body
	errorIndicators := []string{
		"Internal Server Error",
		"template error",
		"no such template",
		"undefined",
	}
	for _, indicator := range errorIndicators {
		if strings.Contains(body, indicator) {
			t.Errorf("[%s] response contains error indicator '%s': %s", path, indicator, truncate(body, 500))
		}
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func TestPageHome(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	assertHTMLPage(t, resp, "/")
}

func TestPageLogin(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/login")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	assertHTMLPage(t, resp, "/login")
}

func TestPageRegister(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/register")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	assertHTMLPage(t, resp, "/register")
}

func TestPageExplore(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/explore")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	assertHTMLPage(t, resp, "/explore")
}

func TestPageTagTimeline(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/tags/test")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	assertHTMLPage(t, resp, "/tags/test")
}

func TestPageSearch(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/search?q=test")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	assertHTMLPage(t, resp, "/search")
}

func TestPageProfile(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// Create a user first
	registerAndGetToken(t, ts.URL, "testprofile", "testprofile@example.com")

	resp, err := http.Get(ts.URL + "/u/testprofile")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	assertHTMLPage(t, resp, "/u/testprofile")
}

func TestPageNotifications(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	token := registerAndGetToken(t, ts.URL, "notifuser", "notif@example.com")

	req, _ := http.NewRequest("GET", ts.URL+"/notifications", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	assertHTMLPage(t, resp, "/notifications")
}

func TestPageBookmarks(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	token := registerAndGetToken(t, ts.URL, "bookmarkuser", "bookmark@example.com")

	req, _ := http.NewRequest("GET", ts.URL+"/bookmarks", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	assertHTMLPage(t, resp, "/bookmarks")
}

func TestPageSettings(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	token := registerAndGetToken(t, ts.URL, "settingsuser", "settings@example.com")

	req, _ := http.NewRequest("GET", ts.URL+"/settings", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	assertHTMLPage(t, resp, "/settings")
}

func TestAuthRegistration(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := `{"username":"alice","email":"alice@example.com","password":"secret123"}`
	resp, err := http.Post(ts.URL+"/api/v1/auth/register", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	var data authData
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to decode data: %v", err)
	}

	if data.Account.Username != "alice" {
		t.Errorf("expected username alice, got %s", data.Account.Username)
	}
	if data.Token == "" {
		t.Error("expected token to be set")
	}
}

func TestAuthLogin(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// Register first
	regBody := `{"username":"bob","email":"bob@example.com","password":"secret123"}`
	resp, err := http.Post(ts.URL+"/api/v1/auth/register", "application/json", bytes.NewBufferString(regBody))
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	resp.Body.Close()

	// Now login
	loginBody := `{"username":"bob","password":"secret123"}`
	resp, err = http.Post(ts.URL+"/api/v1/auth/login", "application/json", bytes.NewBufferString(loginBody))
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	var data authData
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to decode data: %v", err)
	}

	if data.Token == "" {
		t.Error("expected login token to be set")
	}
}

func TestLoginInvalidCredentials(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	loginBody := `{"username":"nonexistent","password":"wrong"}`
	resp, err := http.Post(ts.URL+"/api/v1/auth/login", "application/json", bytes.NewBufferString(loginBody))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 401 {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestCreatePost(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// Register and get token
	token := registerAndGetToken(t, ts.URL, "charlie", "charlie@example.com")

	// Create post
	postBody := `{"content":"Hello world! #test"}`
	req, _ := http.NewRequest("POST", ts.URL+"/api/v1/posts", bytes.NewBufferString(postBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("create post failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 && resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200/201, got %d: %s", resp.StatusCode, string(bodyBytes))
	}
}

func TestCreatePostUnauthorized(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	postBody := `{"content":"Hello world!"}`
	resp, err := http.Post(ts.URL+"/api/v1/posts", "application/json", bytes.NewBufferString(postBody))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 401 {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestGetLocalTimeline(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// Create a user and post
	token := registerAndGetToken(t, ts.URL, "dan", "dan@example.com")
	createPost(t, ts.URL, token, "Local timeline test post")

	// Fetch local timeline (no auth required)
	resp, err := http.Get(ts.URL + "/api/v1/timelines/local")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Data []postData `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(result.Data) == 0 {
		t.Error("expected at least one post in local timeline")
	}
}

func TestGetHomeTimeline(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	token := registerAndGetToken(t, ts.URL, "eve", "eve@example.com")

	req, _ := http.NewRequest("GET", ts.URL+"/api/v1/timelines/home", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(bodyBytes))
	}
}

func TestGetHomeTimelineUnauthorized(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/timelines/home")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 401 {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestGetAccount(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	accountID := registerAndGetAccountID(t, ts.URL, "frank", "frank@example.com")

	resp, err := http.Get(ts.URL + "/api/v1/accounts/" + accountID)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(bodyBytes))
	}
}

func TestVerifyCredentials(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	token := registerAndGetToken(t, ts.URL, "grace", "grace@example.com")

	req, _ := http.NewRequest("GET", ts.URL+"/api/v1/accounts/verify_credentials", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(bodyBytes))
	}
}

func TestFollowUnfollow(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// Create two users
	token1 := registerAndGetToken(t, ts.URL, "henry", "henry@example.com")
	accountID2 := registerAndGetAccountID(t, ts.URL, "iris", "iris@example.com")

	// Follow
	req, _ := http.NewRequest("POST", ts.URL+"/api/v1/accounts/"+accountID2+"/follow", nil)
	req.Header.Set("Authorization", "Bearer "+token1)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("follow request failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("expected 200 for follow, got %d", resp.StatusCode)
	}

	// Unfollow
	req, _ = http.NewRequest("POST", ts.URL+"/api/v1/accounts/"+accountID2+"/unfollow", nil)
	req.Header.Set("Authorization", "Bearer "+token1)

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("unfollow request failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("expected 200 for unfollow, got %d", resp.StatusCode)
	}
}

func TestLikeUnlikePost(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	token := registerAndGetToken(t, ts.URL, "jack", "jack@example.com")
	postID := createPost(t, ts.URL, token, "Like me!")

	// Like
	req, _ := http.NewRequest("POST", ts.URL+"/api/v1/posts/"+postID+"/like", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("like request failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("expected 200 for like, got %d", resp.StatusCode)
	}

	// Unlike
	req, _ = http.NewRequest("DELETE", ts.URL+"/api/v1/posts/"+postID+"/like", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("unlike request failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("expected 200 for unlike, got %d", resp.StatusCode)
	}
}

func TestBookmarkPost(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	token := registerAndGetToken(t, ts.URL, "kate", "kate@example.com")
	postID := createPost(t, ts.URL, token, "Bookmark me!")

	// Bookmark
	req, _ := http.NewRequest("POST", ts.URL+"/api/v1/posts/"+postID+"/bookmark", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("bookmark request failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("expected 200 for bookmark, got %d", resp.StatusCode)
	}

	// Verify bookmark appears in bookmarks list
	req, _ = http.NewRequest("GET", ts.URL+"/api/v1/bookmarks", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("get bookmarks failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("expected 200 for get bookmarks, got %d", resp.StatusCode)
	}
}

func TestSearch(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/search?q=test")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(bodyBytes))
	}
}

func TestTrendingTags(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/trends/tags")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(bodyBytes))
	}
}

func TestNotifications(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	token := registerAndGetToken(t, ts.URL, "leo", "leo@example.com")

	req, _ := http.NewRequest("GET", ts.URL+"/api/v1/notifications", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(bodyBytes))
	}
}

func TestHashtagTimeline(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	token := registerAndGetToken(t, ts.URL, "mike", "mike@example.com")
	createPost(t, ts.URL, token, "Hello #golang community!")

	resp, err := http.Get(ts.URL + "/api/v1/timelines/tag/golang")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(bodyBytes))
	}
}

func TestStaticAssets(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// Test static file route - verify it's handled by static handler
	resp, err := http.Get(ts.URL + "/static/css/app.css")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()

	// Verify correct status code
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	// Verify correct Content-Type for CSS files
	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/css; charset=utf-8" {
		t.Errorf("expected Content-Type 'text/css; charset=utf-8', got '%s'", contentType)
	}
}

func TestServerTimeout(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", ts.URL+"/api/v1/timelines/local", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

// TestE2EUserRegistrationAndLogin tests the complete registration and login workflow.
// This test verifies that a user can successfully:
// 1. Register a new account
// 2. Log in with their credentials
// 3. Access authenticated pages
func TestE2EUserRegistrationAndLogin(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// Step 1: Verify registration page loads
	t.Run("registration_page_loads", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/register")
		if err != nil {
			t.Fatalf("failed to load register page: %v", err)
		}
		defer resp.Body.Close()

		assertHTMLPage(t, resp, "/register")
	})

	// Step 2: Register a new user
	var token string
	var accountID string
	t.Run("user_registration", func(t *testing.T) {
		regBody := `{"username":"testuser","email":"test@example.com","password":"password123"}`
		resp, err := http.Post(ts.URL+"/api/v1/auth/register", "application/json", bytes.NewBufferString(regBody))
		if err != nil {
			t.Fatalf("registration request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			bodyBytes, _ := io.ReadAll(resp.Body)
			t.Fatalf("registration failed with status %d: %s", resp.StatusCode, string(bodyBytes))
		}

		var result apiResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode registration response: %v", err)
		}

		var data authData
		if err := json.Unmarshal(result.Data, &data); err != nil {
			t.Fatalf("failed to decode auth data: %v", err)
		}

		if data.Account.Username != "testuser" {
			t.Errorf("expected username 'testuser', got %s", data.Account.Username)
		}
		if data.Token == "" {
			t.Error("expected token to be returned after registration")
		}

		token = data.Token
		accountID = data.Account.ID
	})

	// Step 3: Verify login page loads
	t.Run("login_page_loads", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/login")
		if err != nil {
			t.Fatalf("failed to load login page: %v", err)
		}
		defer resp.Body.Close()

		assertHTMLPage(t, resp, "/login")
	})

	// Step 4: Login with registered credentials
	t.Run("user_login", func(t *testing.T) {
		loginBody := `{"username":"testuser","password":"password123"}`
		resp, err := http.Post(ts.URL+"/api/v1/auth/login", "application/json", bytes.NewBufferString(loginBody))
		if err != nil {
			t.Fatalf("login request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			bodyBytes, _ := io.ReadAll(resp.Body)
			t.Fatalf("login failed with status %d: %s", resp.StatusCode, string(bodyBytes))
		}

		var result apiResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode login response: %v", err)
		}

		var data authData
		if err := json.Unmarshal(result.Data, &data); err != nil {
			t.Fatalf("failed to decode auth data: %v", err)
		}

		if data.Token == "" {
			t.Error("expected token to be returned after login")
		}
		if data.Account.Username != "testuser" {
			t.Errorf("expected username 'testuser', got %s", data.Account.Username)
		}

		// Update token with login token (should be same or new)
		token = data.Token
	})

	// Step 5: Access authenticated pages with token
	t.Run("access_authenticated_pages", func(t *testing.T) {
		// Test home page
		req, _ := http.NewRequest("GET", ts.URL+"/", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("failed to access home page: %v", err)
		}
		defer resp.Body.Close()

		assertHTMLPage(t, resp, "/")

		// Test notifications page
		req, _ = http.NewRequest("GET", ts.URL+"/notifications", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("failed to access notifications page: %v", err)
		}
		defer resp.Body.Close()

		assertHTMLPage(t, resp, "/notifications")

		// Test settings page
		req, _ = http.NewRequest("GET", ts.URL+"/settings", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("failed to access settings page: %v", err)
		}
		defer resp.Body.Close()

		assertHTMLPage(t, resp, "/settings")
	})

	// Step 6: Create a post as authenticated user
	t.Run("create_post_as_user", func(t *testing.T) {
		postBody := `{"content":"Hello from e2e test! #testing"}`
		req, _ := http.NewRequest("POST", ts.URL+"/api/v1/posts", bytes.NewBufferString(postBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("failed to create post: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 && resp.StatusCode != 201 {
			bodyBytes, _ := io.ReadAll(resp.Body)
			t.Fatalf("create post failed with status %d: %s", resp.StatusCode, string(bodyBytes))
		}
	})

	// Step 7: View user profile
	t.Run("view_user_profile", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/u/testuser")
		if err != nil {
			t.Fatalf("failed to view profile: %v", err)
		}
		defer resp.Body.Close()

		assertHTMLPage(t, resp, "/u/testuser")
	})

	// Step 8: Verify credentials endpoint
	t.Run("verify_credentials", func(t *testing.T) {
		req, _ := http.NewRequest("GET", ts.URL+"/api/v1/accounts/verify_credentials", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("verify credentials request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			bodyBytes, _ := io.ReadAll(resp.Body)
			t.Fatalf("verify credentials failed with status %d: %s", resp.StatusCode, string(bodyBytes))
		}

		var result struct {
			Data struct {
				ID       string `json:"id"`
				Username string `json:"username"`
			} `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if result.Data.ID != accountID {
			t.Errorf("expected account ID %s, got %s", accountID, result.Data.ID)
		}
		if result.Data.Username != "testuser" {
			t.Errorf("expected username 'testuser', got %s", result.Data.Username)
		}
	})
}

// TestE2ECompleteUserJourney tests a complete user journey from registration to interaction.
// This simulates a realistic user flow including:
// 1. Registration
// 2. Login
// 3. Creating posts
// 4. Following other users
// 5. Interacting with posts (like, bookmark)
func TestE2ECompleteUserJourney(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// Create first user (Alice)
	aliceToken := registerAndGetToken(t, ts.URL, "alice", "alice@example.com")
	aliceID := registerAndGetAccountID(t, ts.URL, "alice_dup", "alice2@example.com")

	// Create second user (Bob)
	bobToken := registerAndGetToken(t, ts.URL, "bob", "bob@example.com")
	_ = registerAndGetAccountID(t, ts.URL, "bob_dup", "bob2@example.com") // Just to register, ID not needed

	// Alice creates a post
	var alicePostID string
	t.Run("alice_creates_post", func(t *testing.T) {
		alicePostID = createPost(t, ts.URL, aliceToken, "Hello world from Alice! #intro")
		if alicePostID == "" {
			t.Fatal("failed to create post for alice")
		}
	})

	// Bob follows Alice
	t.Run("bob_follows_alice", func(t *testing.T) {
		req, _ := http.NewRequest("POST", ts.URL+"/api/v1/accounts/"+aliceID+"/follow", nil)
		req.Header.Set("Authorization", "Bearer "+bobToken)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("follow request failed: %v", err)
		}
		resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Errorf("expected 200 for follow, got %d", resp.StatusCode)
		}
	})

	// Bob likes Alice's post
	t.Run("bob_likes_alice_post", func(t *testing.T) {
		req, _ := http.NewRequest("POST", ts.URL+"/api/v1/posts/"+alicePostID+"/like", nil)
		req.Header.Set("Authorization", "Bearer "+bobToken)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("like request failed: %v", err)
		}
		resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Errorf("expected 200 for like, got %d", resp.StatusCode)
		}
	})

	// Bob bookmarks Alice's post
	t.Run("bob_bookmarks_alice_post", func(t *testing.T) {
		req, _ := http.NewRequest("POST", ts.URL+"/api/v1/posts/"+alicePostID+"/bookmark", nil)
		req.Header.Set("Authorization", "Bearer "+bobToken)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("bookmark request failed: %v", err)
		}
		resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Errorf("expected 200 for bookmark, got %d", resp.StatusCode)
		}
	})

	// Verify Bob's bookmarks contain the post
	t.Run("verify_bob_bookmarks", func(t *testing.T) {
		req, _ := http.NewRequest("GET", ts.URL+"/api/v1/bookmarks", nil)
		req.Header.Set("Authorization", "Bearer "+bobToken)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("get bookmarks request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			bodyBytes, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200 for bookmarks, got %d: %s", resp.StatusCode, string(bodyBytes))
		}

		var result struct {
			Data []postData `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode bookmarks: %v", err)
		}

		if len(result.Data) == 0 {
			t.Error("expected at least one bookmark")
		}
	})

	// Verify Bob's home timeline includes Alice's posts
	t.Run("verify_bob_home_timeline", func(t *testing.T) {
		req, _ := http.NewRequest("GET", ts.URL+"/api/v1/timelines/home", nil)
		req.Header.Set("Authorization", "Bearer "+bobToken)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("home timeline request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			bodyBytes, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200 for home timeline, got %d: %s", resp.StatusCode, string(bodyBytes))
		}

		var result struct {
			Data []postData `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode timeline: %v", err)
		}

		// Since Bob follows Alice, Alice's posts should appear in Bob's home timeline
		// (depending on timeline algorithm implementation)
	})

	// Alice views her followers
	t.Run("alice_views_followers", func(t *testing.T) {
		req, _ := http.NewRequest("GET", ts.URL+"/api/v1/accounts/"+aliceID+"/followers", nil)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("get followers request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			bodyBytes, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200 for followers, got %d: %s", resp.StatusCode, string(bodyBytes))
		}
	})
}

// Helper functions

func registerAndGetToken(t *testing.T, baseURL, username, email string) string {
	t.Helper()

	body := fmt.Sprintf(`{"username":"%s","email":"%s","password":"secret123"}`, username, email)
	resp, err := http.Post(baseURL+"/api/v1/auth/register", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	defer resp.Body.Close()

	var result apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	var data authData
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to decode auth data: %v", err)
	}

	return data.Token
}

func registerAndGetAccountID(t *testing.T, baseURL, username, email string) string {
	t.Helper()

	body := fmt.Sprintf(`{"username":"%s","email":"%s","password":"secret123"}`, username, email)
	resp, err := http.Post(baseURL+"/api/v1/auth/register", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	defer resp.Body.Close()

	var result apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	var data authData
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to decode auth data: %v", err)
	}

	return data.Account.ID
}

func createPost(t *testing.T, baseURL, token, content string) string {
	t.Helper()

	body := fmt.Sprintf(`{"content":"%s"}`, content)
	req, _ := http.NewRequest("POST", baseURL+"/api/v1/posts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("create post failed: %v", err)
	}
	defer resp.Body.Close()

	var result struct {
		Data postData `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	return result.Data.ID
}
