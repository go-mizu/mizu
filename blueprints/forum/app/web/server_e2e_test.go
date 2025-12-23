//go:build e2e

package web_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/blueprints/forum/app/web"
	"github.com/go-mizu/mizu/blueprints/forum/feature/accounts"
	"github.com/go-mizu/mizu/blueprints/forum/store/duckdb"

	_ "github.com/marcboeker/go-duckdb"
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
	ts := httptest.NewServer(srv)

	t.Cleanup(func() {
		ts.Close()
		store.Close()
	})

	return ts, store
}

// Helper to create a test user.
func createTestUser(t *testing.T, store *duckdb.Store, username string) *accounts.Account {
	t.Helper()
	ctx := context.Background()

	svc := accounts.NewService(store.Accounts())
	account, err := svc.Create(ctx, accounts.CreateIn{
		Username: username,
		Email:    username + "@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("create user %s: %v", username, err)
	}
	return account
}

// Helper to login and get session token.
func loginUser(t *testing.T, ts *httptest.Server, username, password string) string {
	t.Helper()

	body := map[string]string{
		"username": username,
		"password": password,
	}
	jsonBody, _ := json.Marshal(body)

	resp, err := http.Post(ts.URL+"/api/auth/login", "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		t.Fatalf("login request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("login failed: %d - %s", resp.StatusCode, string(body))
	}

	// Get session cookie
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "session" {
			return cookie.Value
		}
	}

	t.Fatal("no session cookie in login response")
	return ""
}

// Helper to make authenticated request.
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

// TestE2E_Auth tests authentication endpoints.
func TestE2E_Auth(t *testing.T) {
	if os.Getenv("E2E_TEST") != "1" {
		t.Skip("set E2E_TEST=1 to run")
	}

	ts, _ := setupTestServer(t)

	t.Run("Register", func(t *testing.T) {
		body := `{"username":"newuser","email":"new@example.com","password":"password123"}`
		resp, respBody := authRequest(t, "POST", ts.URL+"/api/auth/register", "", strings.NewReader(body))

		assertStatus(t, resp, http.StatusOK)
		assertContains(t, respBody, "newuser")
	})

	t.Run("Login", func(t *testing.T) {
		// First register
		body := `{"username":"logintest","email":"login@example.com","password":"password123"}`
		authRequest(t, "POST", ts.URL+"/api/auth/register", "", strings.NewReader(body))

		// Then login
		loginBody := `{"username":"logintest","password":"password123"}`
		resp, respBody := authRequest(t, "POST", ts.URL+"/api/auth/login", "", strings.NewReader(loginBody))

		assertStatus(t, resp, http.StatusOK)
		assertContains(t, respBody, "logintest")

		// Check for session cookie
		found := false
		for _, cookie := range resp.Cookies() {
			if cookie.Name == "session" {
				found = true
				break
			}
		}
		if !found {
			t.Error("no session cookie set")
		}
	})

	t.Run("Login_InvalidPassword", func(t *testing.T) {
		// Register first
		body := `{"username":"badpass","email":"badpass@example.com","password":"password123"}`
		authRequest(t, "POST", ts.URL+"/api/auth/register", "", strings.NewReader(body))

		// Login with wrong password
		loginBody := `{"username":"badpass","password":"wrongpassword"}`
		resp, _ := authRequest(t, "POST", ts.URL+"/api/auth/login", "", strings.NewReader(loginBody))

		if resp.StatusCode == http.StatusOK {
			t.Error("expected login to fail with wrong password")
		}
	})
}

// TestE2E_Boards tests board endpoints.
func TestE2E_Boards(t *testing.T) {
	if os.Getenv("E2E_TEST") != "1" {
		t.Skip("set E2E_TEST=1 to run")
	}

	ts, store := setupTestServer(t)

	// Create and login user
	user := createTestUser(t, store, "boarduser")
	token := loginUser(t, ts, user.Username, "password123")

	t.Run("CreateBoard", func(t *testing.T) {
		body := `{"name":"testboard","title":"Test Board","description":"A test board"}`
		resp, respBody := authRequest(t, "POST", ts.URL+"/api/boards", token, strings.NewReader(body))

		assertStatus(t, resp, http.StatusOK)
		assertContains(t, respBody, "testboard")
		assertContains(t, respBody, "Test Board")
	})

	t.Run("GetBoard", func(t *testing.T) {
		// Create board first
		body := `{"name":"getboard","title":"Get Board"}`
		authRequest(t, "POST", ts.URL+"/api/boards", token, strings.NewReader(body))

		resp, respBody := get(t, ts.URL+"/api/boards/getboard")

		assertStatus(t, resp, http.StatusOK)
		assertContains(t, respBody, "getboard")
	})

	t.Run("ListBoards", func(t *testing.T) {
		resp, respBody := get(t, ts.URL+"/api/boards")

		assertStatus(t, resp, http.StatusOK)
		assertContains(t, respBody, "[") // Should be array
	})

	t.Run("JoinBoard", func(t *testing.T) {
		// Create board
		body := `{"name":"joinboard","title":"Join Board"}`
		authRequest(t, "POST", ts.URL+"/api/boards", token, strings.NewReader(body))

		// Join
		resp, _ := authRequest(t, "POST", ts.URL+"/api/boards/joinboard/join", token, nil)
		assertStatus(t, resp, http.StatusOK)
	})

	t.Run("LeaveBoard", func(t *testing.T) {
		// Create and join board
		body := `{"name":"leaveboard","title":"Leave Board"}`
		authRequest(t, "POST", ts.URL+"/api/boards", token, strings.NewReader(body))
		authRequest(t, "POST", ts.URL+"/api/boards/leaveboard/join", token, nil)

		// Leave
		resp, _ := authRequest(t, "DELETE", ts.URL+"/api/boards/leaveboard/join", token, nil)
		assertStatus(t, resp, http.StatusOK)
	})
}

// TestE2E_Threads tests thread endpoints.
func TestE2E_Threads(t *testing.T) {
	if os.Getenv("E2E_TEST") != "1" {
		t.Skip("set E2E_TEST=1 to run")
	}

	ts, store := setupTestServer(t)

	user := createTestUser(t, store, "threaduser")
	token := loginUser(t, ts, user.Username, "password123")

	// Create board first
	boardBody := `{"name":"threadboard","title":"Thread Board"}`
	authRequest(t, "POST", ts.URL+"/api/boards", token, strings.NewReader(boardBody))

	t.Run("CreateThread", func(t *testing.T) {
		body := `{"title":"Test Thread","content":"This is test content","type":"text"}`
		resp, respBody := authRequest(t, "POST", ts.URL+"/api/boards/threadboard/threads", token, strings.NewReader(body))

		assertStatus(t, resp, http.StatusOK)
		assertContains(t, respBody, "Test Thread")
	})

	t.Run("GetThread", func(t *testing.T) {
		// Create thread
		body := `{"title":"Get Thread","content":"Content","type":"text"}`
		resp, respBody := authRequest(t, "POST", ts.URL+"/api/boards/threadboard/threads", token, strings.NewReader(body))

		var thread struct {
			ID string `json:"id"`
		}
		json.Unmarshal([]byte(respBody), &thread)

		// Get thread
		resp, respBody = get(t, ts.URL+"/api/threads/"+thread.ID)
		assertStatus(t, resp, http.StatusOK)
		assertContains(t, respBody, "Get Thread")
	})

	t.Run("ListThreads", func(t *testing.T) {
		resp, respBody := get(t, ts.URL+"/api/threads")

		assertStatus(t, resp, http.StatusOK)
		assertContains(t, respBody, "[")
	})
}

// TestE2E_Comments tests comment endpoints.
func TestE2E_Comments(t *testing.T) {
	if os.Getenv("E2E_TEST") != "1" {
		t.Skip("set E2E_TEST=1 to run")
	}

	ts, store := setupTestServer(t)

	user := createTestUser(t, store, "commentuser")
	token := loginUser(t, ts, user.Username, "password123")

	// Create board and thread
	boardBody := `{"name":"commentboard","title":"Comment Board"}`
	authRequest(t, "POST", ts.URL+"/api/boards", token, strings.NewReader(boardBody))

	threadBody := `{"title":"Comment Thread","content":"Content","type":"text"}`
	resp, respBody := authRequest(t, "POST", ts.URL+"/api/boards/commentboard/threads", token, strings.NewReader(threadBody))

	var thread struct {
		ID string `json:"id"`
	}
	json.Unmarshal([]byte(respBody), &thread)

	t.Run("CreateComment", func(t *testing.T) {
		body := `{"content":"This is a comment"}`
		resp, respBody := authRequest(t, "POST", ts.URL+"/api/threads/"+thread.ID+"/comments", token, strings.NewReader(body))

		assertStatus(t, resp, http.StatusOK)
		assertContains(t, respBody, "This is a comment")
	})

	t.Run("ListComments", func(t *testing.T) {
		resp, respBody := get(t, ts.URL+"/api/threads/"+thread.ID+"/comments")

		assertStatus(t, resp, http.StatusOK)
		assertContains(t, respBody, "[")
	})
}

// TestE2E_Voting tests voting endpoints.
func TestE2E_Voting(t *testing.T) {
	if os.Getenv("E2E_TEST") != "1" {
		t.Skip("set E2E_TEST=1 to run")
	}

	ts, store := setupTestServer(t)

	user := createTestUser(t, store, "voteuser")
	token := loginUser(t, ts, user.Username, "password123")

	// Create board and thread
	boardBody := `{"name":"voteboard","title":"Vote Board"}`
	authRequest(t, "POST", ts.URL+"/api/boards", token, strings.NewReader(boardBody))

	threadBody := `{"title":"Vote Thread","content":"Content","type":"text"}`
	resp, respBody := authRequest(t, "POST", ts.URL+"/api/boards/voteboard/threads", token, strings.NewReader(threadBody))

	var thread struct {
		ID string `json:"id"`
	}
	json.Unmarshal([]byte(respBody), &thread)

	t.Run("UpvoteThread", func(t *testing.T) {
		body := `{"value":1}`
		resp, _ := authRequest(t, "POST", ts.URL+"/api/threads/"+thread.ID+"/vote", token, strings.NewReader(body))
		assertStatus(t, resp, http.StatusOK)
	})

	t.Run("DownvoteThread", func(t *testing.T) {
		body := `{"value":-1}`
		resp, _ := authRequest(t, "POST", ts.URL+"/api/threads/"+thread.ID+"/vote", token, strings.NewReader(body))
		assertStatus(t, resp, http.StatusOK)
	})

	t.Run("UnvoteThread", func(t *testing.T) {
		resp, _ := authRequest(t, "DELETE", ts.URL+"/api/threads/"+thread.ID+"/vote", token, nil)
		assertStatus(t, resp, http.StatusOK)
	})
}

// TestE2E_HTMLPages tests HTML page rendering.
func TestE2E_HTMLPages(t *testing.T) {
	if os.Getenv("E2E_TEST") != "1" {
		t.Skip("set E2E_TEST=1 to run")
	}

	ts, store := setupTestServer(t)

	// Create some data
	user := createTestUser(t, store, "pageuser")
	token := loginUser(t, ts, user.Username, "password123")

	boardBody := `{"name":"pageboard","title":"Page Board"}`
	authRequest(t, "POST", ts.URL+"/api/boards", token, strings.NewReader(boardBody))

	t.Run("HomePage", func(t *testing.T) {
		resp, body := get(t, ts.URL+"/")

		assertStatus(t, resp, http.StatusOK)
		assertContains(t, body, "<!DOCTYPE html")
	})

	t.Run("AllPage", func(t *testing.T) {
		resp, body := get(t, ts.URL+"/all")

		assertStatus(t, resp, http.StatusOK)
		assertContains(t, body, "<!DOCTYPE html")
	})

	t.Run("BoardPage", func(t *testing.T) {
		resp, body := get(t, ts.URL+"/b/pageboard")

		assertStatus(t, resp, http.StatusOK)
		assertContains(t, body, "Page Board")
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

	t.Run("SearchPage", func(t *testing.T) {
		resp, body := get(t, ts.URL+"/search?q=test")

		assertStatus(t, resp, http.StatusOK)
		assertContains(t, body, "<!DOCTYPE html")
	})
}

// TestE2E_UserProfile tests user profile endpoints.
func TestE2E_UserProfile(t *testing.T) {
	if os.Getenv("E2E_TEST") != "1" {
		t.Skip("set E2E_TEST=1 to run")
	}

	ts, store := setupTestServer(t)

	user := createTestUser(t, store, "profileuser")

	t.Run("GetUser", func(t *testing.T) {
		resp, body := get(t, ts.URL+"/api/users/"+user.Username)

		assertStatus(t, resp, http.StatusOK)
		assertContains(t, body, "profileuser")
	})

	t.Run("GetUserThreads", func(t *testing.T) {
		resp, body := get(t, ts.URL+"/api/users/"+user.Username+"/threads")

		assertStatus(t, resp, http.StatusOK)
		assertContains(t, body, "[")
	})

	t.Run("GetUserComments", func(t *testing.T) {
		resp, body := get(t, ts.URL+"/api/users/"+user.Username+"/comments")

		assertStatus(t, resp, http.StatusOK)
		assertContains(t, body, "[")
	})

	t.Run("UserProfilePage", func(t *testing.T) {
		resp, body := get(t, ts.URL+"/u/"+user.Username)

		assertStatus(t, resp, http.StatusOK)
		assertContains(t, body, "profileuser")
	})
}

// TestE2E_Scenario_UserJourney tests a complete user journey.
func TestE2E_Scenario_UserJourney(t *testing.T) {
	if os.Getenv("E2E_TEST") != "1" {
		t.Skip("set E2E_TEST=1 to run")
	}

	ts, _ := setupTestServer(t)

	// 1. Register
	registerBody := `{"username":"journeyuser","email":"journey@example.com","password":"password123"}`
	resp, _ := authRequest(t, "POST", ts.URL+"/api/auth/register", "", strings.NewReader(registerBody))
	assertStatus(t, resp, http.StatusOK)

	// 2. Login
	loginBody := `{"username":"journeyuser","password":"password123"}`
	resp, respBody := authRequest(t, "POST", ts.URL+"/api/auth/login", "", strings.NewReader(loginBody))
	assertStatus(t, resp, http.StatusOK)

	var loginResp struct {
		Token string `json:"token"`
	}
	json.Unmarshal([]byte(respBody), &loginResp)
	token := loginResp.Token
	if token == "" {
		// Try cookie
		for _, c := range resp.Cookies() {
			if c.Name == "session" {
				token = c.Value
				break
			}
		}
	}

	// 3. Create board
	boardBody := `{"name":"journeyboard","title":"Journey Board","description":"My journey board"}`
	resp, _ = authRequest(t, "POST", ts.URL+"/api/boards", token, strings.NewReader(boardBody))
	assertStatus(t, resp, http.StatusOK)

	// 4. Create thread
	threadBody := `{"title":"My Journey Thread","content":"Hello world!","type":"text"}`
	resp, respBody = authRequest(t, "POST", ts.URL+"/api/boards/journeyboard/threads", token, strings.NewReader(threadBody))
	assertStatus(t, resp, http.StatusOK)

	var thread struct {
		ID string `json:"id"`
	}
	json.Unmarshal([]byte(respBody), &thread)

	// 5. Create comment
	commentBody := `{"content":"This is my comment"}`
	resp, respBody = authRequest(t, "POST", ts.URL+"/api/threads/"+thread.ID+"/comments", token, strings.NewReader(commentBody))
	assertStatus(t, resp, http.StatusOK)

	var comment struct {
		ID string `json:"id"`
	}
	json.Unmarshal([]byte(respBody), &comment)

	// 6. Vote on thread
	voteBody := `{"value":1}`
	resp, _ = authRequest(t, "POST", ts.URL+"/api/threads/"+thread.ID+"/vote", token, strings.NewReader(voteBody))
	assertStatus(t, resp, http.StatusOK)

	// 7. Bookmark thread
	resp, _ = authRequest(t, "POST", ts.URL+"/api/threads/"+thread.ID+"/bookmark", token, nil)
	assertStatus(t, resp, http.StatusOK)

	// 8. Check profile
	resp, respBody = get(t, ts.URL+"/api/users/journeyuser")
	assertStatus(t, resp, http.StatusOK)
	assertContains(t, respBody, "journeyuser")

	// 9. Logout
	resp, _ = authRequest(t, "POST", ts.URL+"/api/auth/logout", token, nil)
	assertStatus(t, resp, http.StatusOK)
}

// TestE2E_Unauthorized tests unauthorized access.
func TestE2E_Unauthorized(t *testing.T) {
	if os.Getenv("E2E_TEST") != "1" {
		t.Skip("set E2E_TEST=1 to run")
	}

	ts, _ := setupTestServer(t)

	t.Run("CreateBoard_NoAuth", func(t *testing.T) {
		body := `{"name":"noauthboard","title":"No Auth Board"}`
		resp, _ := authRequest(t, "POST", ts.URL+"/api/boards", "", strings.NewReader(body))

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", resp.StatusCode)
		}
	})

	t.Run("CreateThread_NoAuth", func(t *testing.T) {
		body := `{"title":"No Auth Thread","content":"Content"}`
		resp, _ := authRequest(t, "POST", ts.URL+"/api/boards/someboard/threads", "", strings.NewReader(body))

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", resp.StatusCode)
		}
	})

	t.Run("Vote_NoAuth", func(t *testing.T) {
		body := `{"value":1}`
		resp, _ := authRequest(t, "POST", ts.URL+"/api/threads/someid/vote", "", strings.NewReader(body))

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", resp.StatusCode)
		}
	})
}
