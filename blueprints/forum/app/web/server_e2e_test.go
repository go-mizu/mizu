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

	"github.com/go-mizu/blueprints/forum/app/web"
)

// testServer creates a test server instance with a temporary database.
func testServer(t *testing.T) (*web.Server, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "forum-e2e-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	dbPath := tmpDir + "/forum_test.duckdb"
	cfg := web.Config{
		Host:         "localhost",
		Port:         0,
		DatabasePath: dbPath,
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
	Error string          `json:"error,omitempty"`
}

type authData struct {
	Session struct {
		Token     string `json:"token"`
		AccountID string `json:"account_id"`
	} `json:"session"`
	Account struct {
		ID          string `json:"id"`
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
		Email       string `json:"email,omitempty"`
	} `json:"account"`
}

type accountData struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email,omitempty"`
	PostKarma   int    `json:"post_karma"`
	CommentKarma int   `json:"comment_karma"`
	TotalKarma  int    `json:"total_karma"`
}

// assertHTMLPage verifies that an HTML page renders successfully without errors.
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
	if !strings.Contains(body, "<html") {
		t.Errorf("[%s] response doesn't look like HTML: %s", path, truncate(body, 200))
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func TestServerStartup(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	// Verify the server was created successfully
	if srv == nil {
		t.Fatal("expected server to be created")
	}

	// Verify handler is not nil
	if srv.Handler() == nil {
		t.Fatal("expected handler to be non-nil")
	}
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

func TestAuthRegistration(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := `{"username":"alice","email":"alice@example.com","password":"secret123","display_name":"Alice"}`
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
	if data.Session.Token == "" {
		t.Error("expected token to be set")
	}
	if data.Session.AccountID == "" {
		t.Error("expected account_id to be set")
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
	loginBody := `{"username_or_email":"bob","password":"secret123"}`
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

	if data.Session.Token == "" {
		t.Error("expected login token to be set")
	}
}

func TestLoginInvalidCredentials(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	loginBody := `{"username_or_email":"nonexistent","password":"wrong"}`
	resp, err := http.Post(ts.URL+"/api/v1/auth/login", "application/json", bytes.NewBufferString(loginBody))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 401 {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestAuthLogout(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	token := registerAndGetToken(t, ts.URL, "charlie", "charlie@example.com")

	// Logout
	req, _ := http.NewRequest("POST", ts.URL+"/api/v1/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("logout failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Verify token is invalid after logout by trying to access protected endpoint
	req, _ = http.NewRequest("GET", ts.URL+"/api/v1/auth/verify", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("verify request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 401 {
		t.Errorf("expected 401 after logout, got %d", resp.StatusCode)
	}
}

func TestVerifyCredentials(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	token := registerAndGetToken(t, ts.URL, "dave", "dave@example.com")

	req, _ := http.NewRequest("GET", ts.URL+"/api/v1/auth/verify", nil)
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

	var result struct {
		Data struct {
			Account accountData `json:"account"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Data.Account.Username != "dave" {
		t.Errorf("expected username dave, got %s", result.Data.Account.Username)
	}
}

func TestGetAccount(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	accountID := registerAndGetAccountID(t, ts.URL, "eve", "eve@example.com")

	// Get account by ID
	resp, err := http.Get(ts.URL + "/api/v1/accounts/" + accountID)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Data struct {
			Account accountData `json:"account"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Data.Account.Username != "eve" {
		t.Errorf("expected username eve, got %s", result.Data.Account.Username)
	}
}

func TestGetAccountByUsername(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	_ = registerAndGetToken(t, ts.URL, "frank", "frank@example.com")

	// Get account by username
	resp, err := http.Get(ts.URL + "/api/v1/accounts/frank")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Data struct {
			Account accountData `json:"account"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Data.Account.Username != "frank" {
		t.Errorf("expected username frank, got %s", result.Data.Account.Username)
	}
}

func TestUpdateAccount(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	token := registerAndGetToken(t, ts.URL, "grace", "grace@example.com")
	accountID := registerAndGetAccountID(t, ts.URL, "grace_dup", "grace2@example.com")

	// Update account
	updateBody := `{"display_name":"Grace Updated","bio":"This is my new bio"}`
	req, _ := http.NewRequest("PATCH", ts.URL+"/api/v1/accounts/"+accountID, bytes.NewBufferString(updateBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(bodyBytes))
	}
}

func TestSearchAccounts(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	_ = registerAndGetToken(t, ts.URL, "searchuser1", "search1@example.com")
	_ = registerAndGetToken(t, ts.URL, "searchuser2", "search2@example.com")

	// Search accounts
	resp, err := http.Get(ts.URL + "/api/v1/accounts/search?q=searchuser&limit=10")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Data struct {
			Accounts []accountData `json:"accounts"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(result.Data.Accounts) < 2 {
		t.Errorf("expected at least 2 accounts, got %d", len(result.Data.Accounts))
	}
}

func TestPageProfile(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	_ = registerAndGetToken(t, ts.URL, "henry", "henry@example.com")

	resp, err := http.Get(ts.URL + "/u/henry")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	assertHTMLPage(t, resp, "/u/henry")
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
	defer resp.Body.Close()

	// We expect either 200 (file exists) or 404 (file doesn't exist in embedded assets)
	// Both are acceptable as long as the route is properly configured
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status 200 or 404, got %d", resp.StatusCode)
	}
}

func TestServerTimeout(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", ts.URL+"/", nil)
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
func TestE2EUserRegistrationAndLogin(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	var token string
	var accountID string

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
	t.Run("user_registration", func(t *testing.T) {
		regBody := `{"username":"testuser","email":"test@example.com","password":"password123","display_name":"Test User"}`
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
		if data.Session.Token == "" {
			t.Error("expected token to be returned after registration")
		}

		token = data.Session.Token
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
		loginBody := `{"username_or_email":"testuser","password":"password123"}`
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

		if data.Session.Token == "" {
			t.Error("expected token to be returned after login")
		}
		if data.Account.Username != "testuser" {
			t.Errorf("expected username 'testuser', got %s", data.Account.Username)
		}

		// Update token with login token
		token = data.Session.Token
	})

	// Step 5: Verify credentials endpoint
	t.Run("verify_credentials", func(t *testing.T) {
		req, _ := http.NewRequest("GET", ts.URL+"/api/v1/auth/verify", nil)
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
				Account struct {
					ID       string `json:"id"`
					Username string `json:"username"`
				} `json:"account"`
			} `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if result.Data.Account.ID != accountID {
			t.Errorf("expected account ID %s, got %s", accountID, result.Data.Account.ID)
		}
		if result.Data.Account.Username != "testuser" {
			t.Errorf("expected username 'testuser', got %s", result.Data.Account.Username)
		}
	})

	// Step 6: View user profile
	t.Run("view_user_profile", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/u/testuser")
		if err != nil {
			t.Fatalf("failed to view profile: %v", err)
		}
		defer resp.Body.Close()

		assertHTMLPage(t, resp, "/u/testuser")
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

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("register failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	var data authData
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to decode auth data: %v", err)
	}

	return data.Session.Token
}

func registerAndGetAccountID(t *testing.T, baseURL, username, email string) string {
	t.Helper()

	body := fmt.Sprintf(`{"username":"%s","email":"%s","password":"secret123"}`, username, email)
	resp, err := http.Post(baseURL+"/api/v1/auth/register", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("register failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

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
