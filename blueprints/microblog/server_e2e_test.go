package microblog_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
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

	// Test CSS file
	resp, err := http.Get(ts.URL + "/static/css/app.css")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()

	// May return 200 or 404 depending on static file presence
	if resp.StatusCode != 200 && resp.StatusCode != 404 {
		t.Errorf("expected 200 or 404 for static file, got %d", resp.StatusCode)
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
