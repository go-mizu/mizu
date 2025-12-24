//go:build e2e

package web_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-mizu/blueprints/social/app/web"
	"github.com/go-mizu/blueprints/social/store/duckdb"

	_ "github.com/duckdb/duckdb-go/v2"
)

// setupTestServer creates a test server with a temp directory database.
func setupTestServer(t *testing.T) *httptest.Server {
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

// setupTestServerWithStore creates a test server and returns access to store for test data setup.
func setupTestServerWithStore(t *testing.T) (*httptest.Server, *duckdb.AccountsStore) {
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

	// Open store directly for test data setup
	store, err := duckdb.Open(tempDir)
	if err != nil {
		ts.Close()
		srv.Close()
		t.Fatalf("open store: %v", err)
	}

	t.Cleanup(func() {
		ts.Close()
		store.Close()
		srv.Close()
	})

	return ts, duckdb.NewAccountsStore(store.DB())
}

// authRequest makes an HTTP request with optional auth token.
func authRequest(t *testing.T, method, url, token string, body io.Reader) (*http.Response, string) {
	t.Helper()

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}

	respBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	return resp, string(respBody)
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

// registerAndLogin registers a user and returns the session token.
func registerAndLogin(t *testing.T, ts *httptest.Server, username string) string {
	t.Helper()

	// Register
	regBody := map[string]string{
		"username": username,
		"email":    username + "@example.com",
		"password": "password123",
	}
	jsonBody, _ := json.Marshal(regBody)
	resp, _ := authRequest(t, "POST", ts.URL+"/api/v1/auth/register", "", bytes.NewReader(jsonBody))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register failed: %d", resp.StatusCode)
	}

	// Login
	loginBody := map[string]string{
		"username": username,
		"password": "password123",
	}
	jsonBody, _ = json.Marshal(loginBody)
	resp, respBody := authRequest(t, "POST", ts.URL+"/api/v1/auth/login", "", bytes.NewReader(jsonBody))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login failed: %d - %s", resp.StatusCode, respBody)
	}

	// Extract token from response
	var loginResp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal([]byte(respBody), &loginResp); err == nil && loginResp.Token != "" {
		return loginResp.Token
	}

	// Try cookie
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "session" {
			return cookie.Value
		}
	}

	t.Fatal("no token in login response")
	return ""
}

// TestE2E_Auth tests authentication endpoints.
func TestE2E_Auth(t *testing.T) {
	if os.Getenv("E2E_TEST") != "1" {
		t.Skip("set E2E_TEST=1 to run")
	}

	ts := setupTestServer(t)

	t.Run("Register", func(t *testing.T) {
		body := `{"username":"newuser","email":"new@example.com","password":"password123"}`
		resp, respBody := authRequest(t, "POST", ts.URL+"/api/v1/auth/register", "", strings.NewReader(body))

		assertStatus(t, resp, http.StatusCreated)
		assertContains(t, respBody, "newuser")
	})

	t.Run("Register_DuplicateUsername", func(t *testing.T) {
		body := `{"username":"dupuser","email":"dup1@example.com","password":"password123"}`
		authRequest(t, "POST", ts.URL+"/api/v1/auth/register", "", strings.NewReader(body))

		// Try duplicate
		body = `{"username":"dupuser","email":"dup2@example.com","password":"password123"}`
		resp, _ := authRequest(t, "POST", ts.URL+"/api/v1/auth/register", "", strings.NewReader(body))

		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
			t.Error("expected error for duplicate username")
		}
	})

	t.Run("Login", func(t *testing.T) {
		// Register first
		body := `{"username":"logintest","email":"login@example.com","password":"password123"}`
		authRequest(t, "POST", ts.URL+"/api/v1/auth/register", "", strings.NewReader(body))

		// Login
		loginBody := `{"username":"logintest","password":"password123"}`
		resp, respBody := authRequest(t, "POST", ts.URL+"/api/v1/auth/login", "", strings.NewReader(loginBody))

		assertStatus(t, resp, http.StatusOK)
		assertContains(t, respBody, "logintest")
	})

	t.Run("Login_InvalidPassword", func(t *testing.T) {
		// Register first
		body := `{"username":"badpass","email":"badpass@example.com","password":"password123"}`
		authRequest(t, "POST", ts.URL+"/api/v1/auth/register", "", strings.NewReader(body))

		// Login with wrong password
		loginBody := `{"username":"badpass","password":"wrongpassword"}`
		resp, _ := authRequest(t, "POST", ts.URL+"/api/v1/auth/login", "", strings.NewReader(loginBody))

		if resp.StatusCode == http.StatusOK {
			t.Error("expected login to fail with wrong password")
		}
	})

	t.Run("Logout", func(t *testing.T) {
		token := registerAndLogin(t, ts, "logoutuser")

		resp, _ := authRequest(t, "POST", ts.URL+"/api/v1/auth/logout", token, nil)
		assertStatus(t, resp, http.StatusNoContent)
	})
}

// TestE2E_Accounts tests account endpoints.
func TestE2E_Accounts(t *testing.T) {
	if os.Getenv("E2E_TEST") != "1" {
		t.Skip("set E2E_TEST=1 to run")
	}

	ts := setupTestServer(t)

	token := registerAndLogin(t, ts, "accountuser")

	t.Run("VerifyCredentials", func(t *testing.T) {
		resp, respBody := authRequest(t, "GET", ts.URL+"/api/v1/accounts/verify_credentials", token, nil)

		assertStatus(t, resp, http.StatusOK)
		assertContains(t, respBody, "accountuser")
	})

	t.Run("UpdateCredentials", func(t *testing.T) {
		body := `{"display_name":"Updated Name","bio":"My updated bio"}`
		resp, respBody := authRequest(t, "PATCH", ts.URL+"/api/v1/accounts/update_credentials", token, strings.NewReader(body))

		assertStatus(t, resp, http.StatusOK)
		assertContains(t, respBody, "Updated Name")
	})

	t.Run("GetAccount", func(t *testing.T) {
		// Get own credentials to find ID
		_, respBody := authRequest(t, "GET", ts.URL+"/api/v1/accounts/verify_credentials", token, nil)
		var account struct {
			ID string `json:"id"`
		}
		json.Unmarshal([]byte(respBody), &account)

		resp, respBody := get(t, ts.URL+"/api/v1/accounts/"+account.ID)
		assertStatus(t, resp, http.StatusOK)
		assertContains(t, respBody, "accountuser")
	})

	t.Run("SearchAccounts", func(t *testing.T) {
		resp, respBody := get(t, ts.URL+"/api/v1/accounts/search?q=account")

		assertStatus(t, resp, http.StatusOK)
		assertContains(t, respBody, "[")
	})
}

// TestE2E_Posts tests post endpoints.
func TestE2E_Posts(t *testing.T) {
	if os.Getenv("E2E_TEST") != "1" {
		t.Skip("set E2E_TEST=1 to run")
	}

	ts := setupTestServer(t)

	token := registerAndLogin(t, ts, "postuser")

	t.Run("CreatePost", func(t *testing.T) {
		body := `{"content":"Hello world! #test","visibility":"public"}`
		resp, respBody := authRequest(t, "POST", ts.URL+"/api/v1/posts", token, strings.NewReader(body))

		assertStatus(t, resp, http.StatusCreated)
		assertContains(t, respBody, "Hello world!")
	})

	t.Run("GetPost", func(t *testing.T) {
		// Create post
		body := `{"content":"Get this post","visibility":"public"}`
		resp, respBody := authRequest(t, "POST", ts.URL+"/api/v1/posts", token, strings.NewReader(body))

		var post struct {
			ID string `json:"id"`
		}
		json.Unmarshal([]byte(respBody), &post)

		// Get post
		resp, respBody = get(t, ts.URL+"/api/v1/posts/"+post.ID)
		assertStatus(t, resp, http.StatusOK)
		assertContains(t, respBody, "Get this post")
	})

	t.Run("UpdatePost", func(t *testing.T) {
		// Create post
		body := `{"content":"Original content","visibility":"public"}`
		resp, respBody := authRequest(t, "POST", ts.URL+"/api/v1/posts", token, strings.NewReader(body))

		var post struct {
			ID string `json:"id"`
		}
		json.Unmarshal([]byte(respBody), &post)

		// Update post
		updateBody := `{"content":"Updated content"}`
		resp, respBody = authRequest(t, "PUT", ts.URL+"/api/v1/posts/"+post.ID, token, strings.NewReader(updateBody))
		assertStatus(t, resp, http.StatusOK)
		assertContains(t, respBody, "Updated content")
	})

	t.Run("DeletePost", func(t *testing.T) {
		// Create post
		body := `{"content":"To be deleted","visibility":"public"}`
		resp, respBody := authRequest(t, "POST", ts.URL+"/api/v1/posts", token, strings.NewReader(body))

		var post struct {
			ID string `json:"id"`
		}
		json.Unmarshal([]byte(respBody), &post)

		// Delete post
		resp, _ = authRequest(t, "DELETE", ts.URL+"/api/v1/posts/"+post.ID, token, nil)
		assertStatus(t, resp, http.StatusNoContent)

		// Verify deleted
		resp, _ = get(t, ts.URL+"/api/v1/posts/"+post.ID)
		if resp.StatusCode == http.StatusOK {
			t.Error("expected post to be deleted")
		}
	})

	t.Run("CreateReply", func(t *testing.T) {
		// Create parent post
		body := `{"content":"Parent post","visibility":"public"}`
		_, respBody := authRequest(t, "POST", ts.URL+"/api/v1/posts", token, strings.NewReader(body))

		var parent struct {
			ID string `json:"id"`
		}
		json.Unmarshal([]byte(respBody), &parent)

		// Create reply
		replyBody := `{"content":"This is a reply","visibility":"public","reply_to_id":"` + parent.ID + `"}`
		resp, respBody := authRequest(t, "POST", ts.URL+"/api/v1/posts", token, strings.NewReader(replyBody))
		assertStatus(t, resp, http.StatusCreated)
		assertContains(t, respBody, "This is a reply")
	})

	t.Run("GetPostContext", func(t *testing.T) {
		// Create post
		body := `{"content":"Context post","visibility":"public"}`
		_, respBody := authRequest(t, "POST", ts.URL+"/api/v1/posts", token, strings.NewReader(body))

		var post struct {
			ID string `json:"id"`
		}
		json.Unmarshal([]byte(respBody), &post)

		// Get context
		resp, _ := get(t, ts.URL+"/api/v1/posts/"+post.ID+"/context")
		assertStatus(t, resp, http.StatusOK)
	})
}

// TestE2E_Interactions tests like, repost, and bookmark endpoints.
func TestE2E_Interactions(t *testing.T) {
	if os.Getenv("E2E_TEST") != "1" {
		t.Skip("set E2E_TEST=1 to run")
	}

	ts := setupTestServer(t)

	token := registerAndLogin(t, ts, "interactuser")

	// Create post for interactions
	body := `{"content":"Post for interactions","visibility":"public"}`
	_, respBody := authRequest(t, "POST", ts.URL+"/api/v1/posts", token, strings.NewReader(body))

	var post struct {
		ID string `json:"id"`
	}
	json.Unmarshal([]byte(respBody), &post)

	t.Run("Like", func(t *testing.T) {
		resp, _ := authRequest(t, "POST", ts.URL+"/api/v1/posts/"+post.ID+"/like", token, nil)
		assertStatus(t, resp, http.StatusOK)
	})

	t.Run("Unlike", func(t *testing.T) {
		// Like first
		authRequest(t, "POST", ts.URL+"/api/v1/posts/"+post.ID+"/like", token, nil)

		resp, _ := authRequest(t, "DELETE", ts.URL+"/api/v1/posts/"+post.ID+"/like", token, nil)
		assertStatus(t, resp, http.StatusOK)
	})

	t.Run("LikedBy", func(t *testing.T) {
		// Like post
		authRequest(t, "POST", ts.URL+"/api/v1/posts/"+post.ID+"/like", token, nil)

		resp, respBody := get(t, ts.URL+"/api/v1/posts/"+post.ID+"/liked_by")
		assertStatus(t, resp, http.StatusOK)
		assertContains(t, respBody, "[")
	})

	t.Run("Repost", func(t *testing.T) {
		resp, _ := authRequest(t, "POST", ts.URL+"/api/v1/posts/"+post.ID+"/repost", token, nil)
		assertStatus(t, resp, http.StatusOK)
	})

	t.Run("Unrepost", func(t *testing.T) {
		// Repost first
		authRequest(t, "POST", ts.URL+"/api/v1/posts/"+post.ID+"/repost", token, nil)

		resp, _ := authRequest(t, "DELETE", ts.URL+"/api/v1/posts/"+post.ID+"/repost", token, nil)
		assertStatus(t, resp, http.StatusOK)
	})

	t.Run("RepostedBy", func(t *testing.T) {
		// Repost
		authRequest(t, "POST", ts.URL+"/api/v1/posts/"+post.ID+"/repost", token, nil)

		resp, respBody := get(t, ts.URL+"/api/v1/posts/"+post.ID+"/reposted_by")
		assertStatus(t, resp, http.StatusOK)
		assertContains(t, respBody, "[")
	})

	t.Run("Bookmark", func(t *testing.T) {
		resp, _ := authRequest(t, "POST", ts.URL+"/api/v1/posts/"+post.ID+"/bookmark", token, nil)
		assertStatus(t, resp, http.StatusOK)
	})

	t.Run("Unbookmark", func(t *testing.T) {
		// Bookmark first
		authRequest(t, "POST", ts.URL+"/api/v1/posts/"+post.ID+"/bookmark", token, nil)

		resp, _ := authRequest(t, "DELETE", ts.URL+"/api/v1/posts/"+post.ID+"/bookmark", token, nil)
		assertStatus(t, resp, http.StatusOK)
	})

	t.Run("GetBookmarks", func(t *testing.T) {
		// Bookmark post
		authRequest(t, "POST", ts.URL+"/api/v1/posts/"+post.ID+"/bookmark", token, nil)

		resp, respBody := authRequest(t, "GET", ts.URL+"/api/v1/bookmarks", token, nil)
		assertStatus(t, resp, http.StatusOK)
		assertContains(t, respBody, "[")
	})
}

// TestE2E_Relationships tests follow, block, mute endpoints.
func TestE2E_Relationships(t *testing.T) {
	if os.Getenv("E2E_TEST") != "1" {
		t.Skip("set E2E_TEST=1 to run")
	}

	ts := setupTestServer(t)

	token1 := registerAndLogin(t, ts, "reluser1")

	// Create second user
	body := `{"username":"reluser2","email":"rel2@example.com","password":"password123"}`
	authRequest(t, "POST", ts.URL+"/api/v1/auth/register", "", strings.NewReader(body))

	// Get second user's ID
	_, respBody := get(t, ts.URL+"/api/v1/accounts/search?q=reluser2")
	var searchResults []struct {
		ID string `json:"id"`
	}
	json.Unmarshal([]byte(respBody), &searchResults)

	if len(searchResults) == 0 {
		t.Fatal("could not find reluser2")
	}
	user2ID := searchResults[0].ID

	t.Run("Follow", func(t *testing.T) {
		resp, _ := authRequest(t, "POST", ts.URL+"/api/v1/accounts/"+user2ID+"/follow", token1, nil)
		assertStatus(t, resp, http.StatusOK)
	})

	t.Run("Unfollow", func(t *testing.T) {
		// Follow first
		authRequest(t, "POST", ts.URL+"/api/v1/accounts/"+user2ID+"/follow", token1, nil)

		resp, _ := authRequest(t, "POST", ts.URL+"/api/v1/accounts/"+user2ID+"/unfollow", token1, nil)
		assertStatus(t, resp, http.StatusOK)
	})

	t.Run("GetFollowers", func(t *testing.T) {
		// Follow first
		authRequest(t, "POST", ts.URL+"/api/v1/accounts/"+user2ID+"/follow", token1, nil)

		resp, respBody := get(t, ts.URL+"/api/v1/accounts/"+user2ID+"/followers")
		assertStatus(t, resp, http.StatusOK)
		assertContains(t, respBody, "[")
	})

	t.Run("GetFollowing", func(t *testing.T) {
		// Get user1's ID
		_, respBody := authRequest(t, "GET", ts.URL+"/api/v1/accounts/verify_credentials", token1, nil)
		var user1 struct {
			ID string `json:"id"`
		}
		json.Unmarshal([]byte(respBody), &user1)

		// Follow someone
		authRequest(t, "POST", ts.URL+"/api/v1/accounts/"+user2ID+"/follow", token1, nil)

		resp, respBody := get(t, ts.URL+"/api/v1/accounts/"+user1.ID+"/following")
		assertStatus(t, resp, http.StatusOK)
		assertContains(t, respBody, "[")
	})

	t.Run("Block", func(t *testing.T) {
		resp, _ := authRequest(t, "POST", ts.URL+"/api/v1/accounts/"+user2ID+"/block", token1, nil)
		assertStatus(t, resp, http.StatusOK)
	})

	t.Run("Unblock", func(t *testing.T) {
		// Block first
		authRequest(t, "POST", ts.URL+"/api/v1/accounts/"+user2ID+"/block", token1, nil)

		resp, _ := authRequest(t, "POST", ts.URL+"/api/v1/accounts/"+user2ID+"/unblock", token1, nil)
		assertStatus(t, resp, http.StatusOK)
	})

	t.Run("Mute", func(t *testing.T) {
		resp, _ := authRequest(t, "POST", ts.URL+"/api/v1/accounts/"+user2ID+"/mute", token1, nil)
		assertStatus(t, resp, http.StatusOK)
	})

	t.Run("Unmute", func(t *testing.T) {
		// Mute first
		authRequest(t, "POST", ts.URL+"/api/v1/accounts/"+user2ID+"/mute", token1, nil)

		resp, _ := authRequest(t, "POST", ts.URL+"/api/v1/accounts/"+user2ID+"/unmute", token1, nil)
		assertStatus(t, resp, http.StatusOK)
	})

	t.Run("GetRelationships", func(t *testing.T) {
		resp, respBody := authRequest(t, "GET", ts.URL+"/api/v1/accounts/relationships?id[]="+user2ID, token1, nil)
		assertStatus(t, resp, http.StatusOK)
		assertContains(t, respBody, "[")
	})
}

// TestE2E_Timelines tests timeline endpoints.
func TestE2E_Timelines(t *testing.T) {
	if os.Getenv("E2E_TEST") != "1" {
		t.Skip("set E2E_TEST=1 to run")
	}

	ts := setupTestServer(t)

	token := registerAndLogin(t, ts, "timelineuser")

	// Create some posts
	for i := 0; i < 3; i++ {
		body := `{"content":"Timeline post #test","visibility":"public"}`
		authRequest(t, "POST", ts.URL+"/api/v1/posts", token, strings.NewReader(body))
	}

	t.Run("HomeTimeline", func(t *testing.T) {
		resp, respBody := authRequest(t, "GET", ts.URL+"/api/v1/timelines/home", token, nil)
		assertStatus(t, resp, http.StatusOK)
		assertContains(t, respBody, "[")
	})

	t.Run("PublicTimeline", func(t *testing.T) {
		resp, respBody := get(t, ts.URL+"/api/v1/timelines/public")
		assertStatus(t, resp, http.StatusOK)
		assertContains(t, respBody, "[")
	})

	t.Run("HashtagTimeline", func(t *testing.T) {
		resp, respBody := get(t, ts.URL+"/api/v1/timelines/tag/test")
		assertStatus(t, resp, http.StatusOK)
		assertContains(t, respBody, "[")
	})
}

// TestE2E_Notifications tests notification endpoints.
func TestE2E_Notifications(t *testing.T) {
	if os.Getenv("E2E_TEST") != "1" {
		t.Skip("set E2E_TEST=1 to run")
	}

	ts := setupTestServer(t)

	token := registerAndLogin(t, ts, "notifuser")

	t.Run("ListNotifications", func(t *testing.T) {
		resp, respBody := authRequest(t, "GET", ts.URL+"/api/v1/notifications", token, nil)
		assertStatus(t, resp, http.StatusOK)
		// Empty array or null is acceptable for no notifications
		if !strings.Contains(respBody, "[") && !strings.Contains(respBody, "null") {
			t.Errorf("expected array or null, got %s", respBody)
		}
	})

	t.Run("UnreadCount", func(t *testing.T) {
		resp, respBody := authRequest(t, "GET", ts.URL+"/api/v1/notifications/unread_count", token, nil)
		assertStatus(t, resp, http.StatusOK)
		assertContains(t, respBody, "count")
	})

	t.Run("ClearNotifications", func(t *testing.T) {
		resp, _ := authRequest(t, "POST", ts.URL+"/api/v1/notifications/clear", token, nil)
		assertStatus(t, resp, http.StatusNoContent)
	})
}

// TestE2E_Search tests search endpoint.
func TestE2E_Search(t *testing.T) {
	if os.Getenv("E2E_TEST") != "1" {
		t.Skip("set E2E_TEST=1 to run")
	}

	ts := setupTestServer(t)

	token := registerAndLogin(t, ts, "searchuser")

	// Create some content
	body := `{"content":"Searchable content here","visibility":"public"}`
	authRequest(t, "POST", ts.URL+"/api/v1/posts", token, strings.NewReader(body))

	t.Run("SearchAll", func(t *testing.T) {
		resp, respBody := get(t, ts.URL+"/api/v1/search?q=search")
		assertStatus(t, resp, http.StatusOK)
		assertContains(t, respBody, "{")
	})

	t.Run("SearchAccounts", func(t *testing.T) {
		resp, respBody := get(t, ts.URL+"/api/v1/search?q=search&type=accounts")
		assertStatus(t, resp, http.StatusOK)
		assertContains(t, respBody, "accounts")
	})

	t.Run("SearchPosts", func(t *testing.T) {
		resp, respBody := get(t, ts.URL+"/api/v1/search?q=searchable&type=posts")
		assertStatus(t, resp, http.StatusOK)
		assertContains(t, respBody, "posts")
	})
}

// TestE2E_Trends tests trending endpoints.
func TestE2E_Trends(t *testing.T) {
	if os.Getenv("E2E_TEST") != "1" {
		t.Skip("set E2E_TEST=1 to run")
	}

	ts := setupTestServer(t)

	t.Run("TrendingTags", func(t *testing.T) {
		resp, respBody := get(t, ts.URL+"/api/v1/trends/tags")
		assertStatus(t, resp, http.StatusOK)
		// Empty array or null is acceptable
		if !strings.Contains(respBody, "[") && !strings.Contains(respBody, "null") {
			t.Errorf("expected array or null, got %s", respBody)
		}
	})

	t.Run("TrendingPosts", func(t *testing.T) {
		resp, respBody := get(t, ts.URL+"/api/v1/trends/posts")
		assertStatus(t, resp, http.StatusOK)
		// Empty array or null is acceptable
		if !strings.Contains(respBody, "[") && !strings.Contains(respBody, "null") {
			t.Errorf("expected array or null, got %s", respBody)
		}
	})
}

// TestE2E_Lists tests list endpoints.
func TestE2E_Lists(t *testing.T) {
	if os.Getenv("E2E_TEST") != "1" {
		t.Skip("set E2E_TEST=1 to run")
	}

	ts := setupTestServer(t)

	token := registerAndLogin(t, ts, "listuser")

	// Create another user to add to list
	body := `{"username":"listmember","email":"member@example.com","password":"password123"}`
	authRequest(t, "POST", ts.URL+"/api/v1/auth/register", "", strings.NewReader(body))

	_, respBody := get(t, ts.URL+"/api/v1/accounts/search?q=listmember")
	var searchResults []struct {
		ID string `json:"id"`
	}
	json.Unmarshal([]byte(respBody), &searchResults)
	memberID := ""
	if len(searchResults) > 0 {
		memberID = searchResults[0].ID
	}

	t.Run("CreateList", func(t *testing.T) {
		body := `{"title":"My List"}`
		resp, respBody := authRequest(t, "POST", ts.URL+"/api/v1/lists", token, strings.NewReader(body))

		assertStatus(t, resp, http.StatusCreated)
		assertContains(t, respBody, "My List")
	})

	t.Run("GetLists", func(t *testing.T) {
		resp, respBody := authRequest(t, "GET", ts.URL+"/api/v1/lists", token, nil)
		assertStatus(t, resp, http.StatusOK)
		assertContains(t, respBody, "[")
	})

	t.Run("UpdateList", func(t *testing.T) {
		// Create list
		body := `{"title":"Original"}`
		resp, respBody := authRequest(t, "POST", ts.URL+"/api/v1/lists", token, strings.NewReader(body))

		var list struct {
			ID string `json:"id"`
		}
		json.Unmarshal([]byte(respBody), &list)

		// Update
		updateBody := `{"title":"Updated"}`
		resp, respBody = authRequest(t, "PUT", ts.URL+"/api/v1/lists/"+list.ID, token, strings.NewReader(updateBody))
		assertStatus(t, resp, http.StatusOK)
		assertContains(t, respBody, "Updated")
	})

	t.Run("DeleteList", func(t *testing.T) {
		// Create list
		body := `{"title":"To Delete"}`
		resp, respBody := authRequest(t, "POST", ts.URL+"/api/v1/lists", token, strings.NewReader(body))

		var list struct {
			ID string `json:"id"`
		}
		json.Unmarshal([]byte(respBody), &list)

		// Delete
		resp, _ = authRequest(t, "DELETE", ts.URL+"/api/v1/lists/"+list.ID, token, nil)
		assertStatus(t, resp, http.StatusNoContent)
	})

	t.Run("AddMember", func(t *testing.T) {
		if memberID == "" {
			t.Skip("no member to add")
		}

		// Create list
		body := `{"title":"Members List"}`
		resp, respBody := authRequest(t, "POST", ts.URL+"/api/v1/lists", token, strings.NewReader(body))

		var list struct {
			ID string `json:"id"`
		}
		json.Unmarshal([]byte(respBody), &list)

		// Add member
		addBody := `{"account_ids":["` + memberID + `"]}`
		resp, _ = authRequest(t, "POST", ts.URL+"/api/v1/lists/"+list.ID+"/accounts", token, strings.NewReader(addBody))
		assertStatus(t, resp, http.StatusNoContent)
	})

	t.Run("GetMembers", func(t *testing.T) {
		// Create list
		body := `{"title":"Get Members List"}`
		resp, respBody := authRequest(t, "POST", ts.URL+"/api/v1/lists", token, strings.NewReader(body))

		var list struct {
			ID string `json:"id"`
		}
		json.Unmarshal([]byte(respBody), &list)

		resp, respBody = get(t, ts.URL+"/api/v1/lists/"+list.ID+"/accounts")
		assertStatus(t, resp, http.StatusOK)
		assertContains(t, respBody, "[")
	})

	t.Run("ListTimeline", func(t *testing.T) {
		// Create list
		body := `{"title":"Timeline List"}`
		resp, respBody := authRequest(t, "POST", ts.URL+"/api/v1/lists", token, strings.NewReader(body))

		var list struct {
			ID string `json:"id"`
		}
		json.Unmarshal([]byte(respBody), &list)

		resp, respBody = authRequest(t, "GET", ts.URL+"/api/v1/timelines/list/"+list.ID, token, nil)
		assertStatus(t, resp, http.StatusOK)
		// Empty array or null is acceptable for empty timeline
		if !strings.Contains(respBody, "[") && !strings.Contains(respBody, "null") {
			t.Errorf("expected array or null, got %s", respBody)
		}
	})
}

// TestE2E_HTMLPages tests HTML page rendering.
func TestE2E_HTMLPages(t *testing.T) {
	if os.Getenv("E2E_TEST") != "1" {
		t.Skip("set E2E_TEST=1 to run")
	}

	ts := setupTestServer(t)

	token := registerAndLogin(t, ts, "pageuser")

	// Create a post
	body := `{"content":"Page test post","visibility":"public"}`
	authRequest(t, "POST", ts.URL+"/api/v1/posts", token, strings.NewReader(body))

	t.Run("HomePage", func(t *testing.T) {
		resp, body := get(t, ts.URL+"/")
		assertStatus(t, resp, http.StatusOK)
		assertContains(t, body, "<!DOCTYPE html")
	})

	t.Run("LoginPage", func(t *testing.T) {
		resp, body := get(t, ts.URL+"/login")
		assertStatus(t, resp, http.StatusOK)
		assertContains(t, body, "<!DOCTYPE html")
	})

	t.Run("RegisterPage", func(t *testing.T) {
		resp, body := get(t, ts.URL+"/register")
		assertStatus(t, resp, http.StatusOK)
		assertContains(t, body, "<!DOCTYPE html")
	})

	t.Run("ExplorePage", func(t *testing.T) {
		resp, body := get(t, ts.URL+"/explore")
		assertStatus(t, resp, http.StatusOK)
		assertContains(t, body, "<!DOCTYPE html")
	})

	t.Run("SearchPage", func(t *testing.T) {
		resp, body := get(t, ts.URL+"/search?q=test")
		assertStatus(t, resp, http.StatusOK)
		assertContains(t, body, "<!DOCTYPE html")
	})

	t.Run("ProfilePage", func(t *testing.T) {
		resp, body := get(t, ts.URL+"/u/pageuser")
		assertStatus(t, resp, http.StatusOK)
		assertContains(t, body, "pageuser")
	})

	t.Run("TagPage", func(t *testing.T) {
		resp, body := get(t, ts.URL+"/tags/test")
		assertStatus(t, resp, http.StatusOK)
		assertContains(t, body, "<!DOCTYPE html")
	})
}

// TestE2E_Scenario_UserJourney tests a complete user journey.
func TestE2E_Scenario_UserJourney(t *testing.T) {
	if os.Getenv("E2E_TEST") != "1" {
		t.Skip("set E2E_TEST=1 to run")
	}

	ts := setupTestServer(t)

	// 1. Register
	registerBody := `{"username":"journeyuser","email":"journey@example.com","password":"password123"}`
	resp, _ := authRequest(t, "POST", ts.URL+"/api/v1/auth/register", "", strings.NewReader(registerBody))
	assertStatus(t, resp, http.StatusCreated)

	// 2. Login
	token := registerAndLogin(t, ts, "journeylogin")

	// 3. Update profile
	updateBody := `{"display_name":"Journey User","bio":"Testing the journey"}`
	resp, _ = authRequest(t, "PATCH", ts.URL+"/api/v1/accounts/update_credentials", token, strings.NewReader(updateBody))
	assertStatus(t, resp, http.StatusOK)

	// 4. Create post
	postBody := `{"content":"My first post! #hello","visibility":"public"}`
	resp, respBody := authRequest(t, "POST", ts.URL+"/api/v1/posts", token, strings.NewReader(postBody))
	assertStatus(t, resp, http.StatusCreated)

	var post struct {
		ID string `json:"id"`
	}
	json.Unmarshal([]byte(respBody), &post)

	// 5. Create another user and follow them
	body := `{"username":"journey2","email":"j2@example.com","password":"password123"}`
	authRequest(t, "POST", ts.URL+"/api/v1/auth/register", "", strings.NewReader(body))

	resp, respBody = get(t, ts.URL+"/api/v1/accounts/search?q=journey2")
	var searchResults []struct {
		ID string `json:"id"`
	}
	json.Unmarshal([]byte(respBody), &searchResults)

	if len(searchResults) > 0 {
		user2ID := searchResults[0].ID
		resp, _ = authRequest(t, "POST", ts.URL+"/api/v1/accounts/"+user2ID+"/follow", token, nil)
		assertStatus(t, resp, http.StatusOK)
	}

	// 6. Like own post
	resp, _ = authRequest(t, "POST", ts.URL+"/api/v1/posts/"+post.ID+"/like", token, nil)
	assertStatus(t, resp, http.StatusOK)

	// 7. Bookmark post
	resp, _ = authRequest(t, "POST", ts.URL+"/api/v1/posts/"+post.ID+"/bookmark", token, nil)
	assertStatus(t, resp, http.StatusOK)

	// 8. Create list
	listBody := `{"title":"My List"}`
	resp, respBody = authRequest(t, "POST", ts.URL+"/api/v1/lists", token, strings.NewReader(listBody))
	assertStatus(t, resp, http.StatusCreated)

	// 9. Check home timeline
	resp, respBody = authRequest(t, "GET", ts.URL+"/api/v1/timelines/home", token, nil)
	assertStatus(t, resp, http.StatusOK)

	// 10. Check notifications
	resp, respBody = authRequest(t, "GET", ts.URL+"/api/v1/notifications", token, nil)
	assertStatus(t, resp, http.StatusOK)

	// 11. Search
	resp, respBody = get(t, ts.URL+"/api/v1/search?q=hello")
	assertStatus(t, resp, http.StatusOK)

	// 12. Logout
	resp, _ = authRequest(t, "POST", ts.URL+"/api/v1/auth/logout", token, nil)
	assertStatus(t, resp, http.StatusNoContent)
}

// TestE2E_Unauthorized tests unauthorized access.
func TestE2E_Unauthorized(t *testing.T) {
	if os.Getenv("E2E_TEST") != "1" {
		t.Skip("set E2E_TEST=1 to run")
	}

	ts := setupTestServer(t)

	t.Run("CreatePost_NoAuth", func(t *testing.T) {
		body := `{"content":"No auth post","visibility":"public"}`
		resp, _ := authRequest(t, "POST", ts.URL+"/api/v1/posts", "", strings.NewReader(body))

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", resp.StatusCode)
		}
	})

	t.Run("HomeTimeline_NoAuth", func(t *testing.T) {
		resp, _ := authRequest(t, "GET", ts.URL+"/api/v1/timelines/home", "", nil)

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", resp.StatusCode)
		}
	})

	t.Run("Like_NoAuth", func(t *testing.T) {
		resp, _ := authRequest(t, "POST", ts.URL+"/api/v1/posts/someid/like", "", nil)

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", resp.StatusCode)
		}
	})

	t.Run("Follow_NoAuth", func(t *testing.T) {
		resp, _ := authRequest(t, "POST", ts.URL+"/api/v1/accounts/someid/follow", "", nil)

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", resp.StatusCode)
		}
	})

	t.Run("Notifications_NoAuth", func(t *testing.T) {
		resp, _ := authRequest(t, "GET", ts.URL+"/api/v1/notifications", "", nil)

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", resp.StatusCode)
		}
	})

	t.Run("Lists_NoAuth", func(t *testing.T) {
		resp, _ := authRequest(t, "GET", ts.URL+"/api/v1/lists", "", nil)

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", resp.StatusCode)
		}
	})
}

// TestE2E_NotFound tests 404 responses.
func TestE2E_NotFound(t *testing.T) {
	if os.Getenv("E2E_TEST") != "1" {
		t.Skip("set E2E_TEST=1 to run")
	}

	ts := setupTestServer(t)

	t.Run("GetPost_NotFound", func(t *testing.T) {
		resp, _ := get(t, ts.URL+"/api/v1/posts/nonexistent")
		if resp.StatusCode == http.StatusOK {
			t.Error("expected error for non-existent post")
		}
	})

	t.Run("GetAccount_NotFound", func(t *testing.T) {
		resp, _ := get(t, ts.URL+"/api/v1/accounts/nonexistent")
		if resp.StatusCode == http.StatusOK {
			t.Error("expected error for non-existent account")
		}
	})
}
