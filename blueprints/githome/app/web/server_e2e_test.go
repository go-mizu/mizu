package web

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-mizu/blueprints/githome/app/web/handler/api"
	"github.com/go-mizu/blueprints/githome/store/duckdb"

	_ "github.com/duckdb/duckdb-go/v2"
)

// testServer creates a test server with an in-memory database
func testServer(t *testing.T) (*Server, func()) {
	t.Helper()

	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "githome-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	cfg := Config{
		Addr:     ":0",
		DataDir:  tmpDir,
		ReposDir: tmpDir + "/repos",
		Dev:      true,
	}

	srv, err := New(cfg)
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

// request makes an HTTP request to the test server
type request struct {
	method  string
	path    string
	body    interface{}
	cookies []*http.Cookie
}

func (r request) do(t *testing.T, srv *Server) *httptest.ResponseRecorder {
	t.Helper()

	var body io.Reader
	if r.body != nil {
		b, err := json.Marshal(r.body)
		if err != nil {
			t.Fatalf("failed to marshal body: %v", err)
		}
		body = bytes.NewReader(b)
	}

	req := httptest.NewRequest(r.method, r.path, body)
	if r.body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for _, c := range r.cookies {
		req.AddCookie(c)
	}

	w := httptest.NewRecorder()
	srv.app.ServeHTTP(w, req)
	return w
}

// parseResponse parses the JSON response
func parseResponse(t *testing.T, w *httptest.ResponseRecorder) api.Response {
	t.Helper()

	var resp api.Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to parse response: %v, body: %s", err, w.Body.String())
	}
	return resp
}

// ============================================
// Auth Tests
// ============================================

func TestAuth_Register(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	// Test successful registration
	w := request{
		method: "POST",
		path:   "/api/v1/auth/register",
		body: map[string]string{
			"username": "testuser",
			"email":    "test@example.com",
			"password": "password123",
		},
	}.do(t, srv)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	if !resp.Success {
		t.Errorf("expected success, got error: %s", resp.Error)
	}

	// Check cookie was set
	cookies := w.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "session" {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Error("session cookie not set")
	}
}

func TestAuth_Register_DuplicateUsername(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	// Register first user
	request{
		method: "POST",
		path:   "/api/v1/auth/register",
		body: map[string]string{
			"username": "testuser",
			"email":    "test@example.com",
			"password": "password123",
		},
	}.do(t, srv)

	// Try to register with same username
	w := request{
		method: "POST",
		path:   "/api/v1/auth/register",
		body: map[string]string{
			"username": "testuser",
			"email":    "other@example.com",
			"password": "password123",
		},
	}.do(t, srv)

	if w.Code != http.StatusConflict {
		t.Errorf("expected status 409, got %d", w.Code)
	}
}

func TestAuth_Login(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	// Register user first
	request{
		method: "POST",
		path:   "/api/v1/auth/register",
		body: map[string]string{
			"username": "testuser",
			"email":    "test@example.com",
			"password": "password123",
		},
	}.do(t, srv)

	// Login
	w := request{
		method: "POST",
		path:   "/api/v1/auth/login",
		body: map[string]string{
			"login":    "testuser",
			"password": "password123",
		},
	}.do(t, srv)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuth_Login_InvalidCredentials(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	w := request{
		method: "POST",
		path:   "/api/v1/auth/login",
		body: map[string]string{
			"login":    "nonexistent",
			"password": "wrongpassword",
		},
	}.do(t, srv)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestAuth_Me(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	// Register and get session
	w := request{
		method: "POST",
		path:   "/api/v1/auth/register",
		body: map[string]string{
			"username": "testuser",
			"email":    "test@example.com",
			"password": "password123",
		},
	}.do(t, srv)

	var sessionCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "session" {
			sessionCookie = c
			break
		}
	}

	// Get current user
	w = request{
		method:  "GET",
		path:    "/api/v1/auth/me",
		cookies: []*http.Cookie{sessionCookie},
	}.do(t, srv)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuth_Logout(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	// Register and get session
	w := request{
		method: "POST",
		path:   "/api/v1/auth/register",
		body: map[string]string{
			"username": "testuser",
			"email":    "test@example.com",
			"password": "password123",
		},
	}.do(t, srv)

	var sessionCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "session" {
			sessionCookie = c
			break
		}
	}

	// Logout
	w = request{
		method:  "POST",
		path:    "/api/v1/auth/logout",
		cookies: []*http.Cookie{sessionCookie},
	}.do(t, srv)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Check session cookie was cleared
	for _, c := range w.Result().Cookies() {
		if c.Name == "session" && c.MaxAge == -1 {
			return
		}
	}
	t.Error("session cookie was not cleared")
}

// ============================================
// User Tests
// ============================================

func TestUser_GetCurrent(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	// Register user
	w := request{
		method: "POST",
		path:   "/api/v1/auth/register",
		body: map[string]string{
			"username": "testuser",
			"email":    "test@example.com",
			"password": "password123",
		},
	}.do(t, srv)

	var sessionCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "session" {
			sessionCookie = c
			break
		}
	}

	// Get current user
	w = request{
		method:  "GET",
		path:    "/api/v1/user",
		cookies: []*http.Cookie{sessionCookie},
	}.do(t, srv)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestUser_GetCurrent_Unauthorized(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	w := request{
		method: "GET",
		path:   "/api/v1/user",
	}.do(t, srv)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestUser_Update(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	// Register user
	w := request{
		method: "POST",
		path:   "/api/v1/auth/register",
		body: map[string]string{
			"username": "testuser",
			"email":    "test@example.com",
			"password": "password123",
		},
	}.do(t, srv)

	var sessionCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "session" {
			sessionCookie = c
			break
		}
	}

	// Update user
	w = request{
		method: "PATCH",
		path:   "/api/v1/user",
		body: map[string]string{
			"full_name": "Test User",
			"bio":       "Hello world",
		},
		cookies: []*http.Cookie{sessionCookie},
	}.do(t, srv)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUser_GetByUsername(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	// Register user
	request{
		method: "POST",
		path:   "/api/v1/auth/register",
		body: map[string]string{
			"username": "testuser",
			"email":    "test@example.com",
			"password": "password123",
		},
	}.do(t, srv)

	// Get user by username
	w := request{
		method: "GET",
		path:   "/api/v1/users/testuser",
	}.do(t, srv)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestUser_GetByUsername_NotFound(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	w := request{
		method: "GET",
		path:   "/api/v1/users/nonexistent",
	}.do(t, srv)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

// ============================================
// Repository Tests
// ============================================

func TestRepo_Create(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	// Register user
	w := request{
		method: "POST",
		path:   "/api/v1/auth/register",
		body: map[string]string{
			"username": "testuser",
			"email":    "test@example.com",
			"password": "password123",
		},
	}.do(t, srv)

	var sessionCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "session" {
			sessionCookie = c
			break
		}
	}

	// Create repo
	w = request{
		method: "POST",
		path:   "/api/v1/repos",
		body: map[string]interface{}{
			"name":        "testrepo",
			"description": "A test repo",
			"is_private":  false,
		},
		cookies: []*http.Cookie{sessionCookie},
	}.do(t, srv)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRepo_Create_Unauthorized(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	w := request{
		method: "POST",
		path:   "/api/v1/repos",
		body: map[string]interface{}{
			"name": "testrepo",
		},
	}.do(t, srv)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestRepo_ListPublic(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	// Register user and create a public repo
	w := request{
		method: "POST",
		path:   "/api/v1/auth/register",
		body: map[string]string{
			"username": "testuser",
			"email":    "test@example.com",
			"password": "password123",
		},
	}.do(t, srv)

	var sessionCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "session" {
			sessionCookie = c
			break
		}
	}

	request{
		method: "POST",
		path:   "/api/v1/repos",
		body: map[string]interface{}{
			"name":       "testrepo",
			"is_private": false,
		},
		cookies: []*http.Cookie{sessionCookie},
	}.do(t, srv)

	// List public repos (no auth needed)
	w = request{
		method: "GET",
		path:   "/api/v1/repos",
	}.do(t, srv)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestRepo_Get(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	// Register user and create repo
	w := request{
		method: "POST",
		path:   "/api/v1/auth/register",
		body: map[string]string{
			"username": "testuser",
			"email":    "test@example.com",
			"password": "password123",
		},
	}.do(t, srv)

	var sessionCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "session" {
			sessionCookie = c
			break
		}
	}

	request{
		method: "POST",
		path:   "/api/v1/repos",
		body: map[string]interface{}{
			"name":       "testrepo",
			"is_private": false,
		},
		cookies: []*http.Cookie{sessionCookie},
	}.do(t, srv)

	// Get repo
	w = request{
		method: "GET",
		path:   "/api/v1/repos/testuser/testrepo",
	}.do(t, srv)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRepo_Update(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	// Register user and create repo
	w := request{
		method: "POST",
		path:   "/api/v1/auth/register",
		body: map[string]string{
			"username": "testuser",
			"email":    "test@example.com",
			"password": "password123",
		},
	}.do(t, srv)

	var sessionCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "session" {
			sessionCookie = c
			break
		}
	}

	request{
		method: "POST",
		path:   "/api/v1/repos",
		body: map[string]interface{}{
			"name":       "testrepo",
			"is_private": false,
		},
		cookies: []*http.Cookie{sessionCookie},
	}.do(t, srv)

	// Update repo
	w = request{
		method: "PATCH",
		path:   "/api/v1/repos/testuser/testrepo",
		body: map[string]interface{}{
			"description": "Updated description",
		},
		cookies: []*http.Cookie{sessionCookie},
	}.do(t, srv)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRepo_Delete(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	// Register user and create repo
	w := request{
		method: "POST",
		path:   "/api/v1/auth/register",
		body: map[string]string{
			"username": "testuser",
			"email":    "test@example.com",
			"password": "password123",
		},
	}.do(t, srv)

	var sessionCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "session" {
			sessionCookie = c
			break
		}
	}

	request{
		method: "POST",
		path:   "/api/v1/repos",
		body: map[string]interface{}{
			"name":       "testrepo",
			"is_private": false,
		},
		cookies: []*http.Cookie{sessionCookie},
	}.do(t, srv)

	// Delete repo
	w = request{
		method:  "DELETE",
		path:    "/api/v1/repos/testuser/testrepo",
		cookies: []*http.Cookie{sessionCookie},
	}.do(t, srv)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRepo_Star_Unstar(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	// Register user and create repo
	w := request{
		method: "POST",
		path:   "/api/v1/auth/register",
		body: map[string]string{
			"username": "testuser",
			"email":    "test@example.com",
			"password": "password123",
		},
	}.do(t, srv)

	var sessionCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "session" {
			sessionCookie = c
			break
		}
	}

	request{
		method: "POST",
		path:   "/api/v1/repos",
		body: map[string]interface{}{
			"name":       "testrepo",
			"is_private": false,
		},
		cookies: []*http.Cookie{sessionCookie},
	}.do(t, srv)

	// Star repo
	w = request{
		method:  "PUT",
		path:    "/api/v1/user/starred/testuser/testrepo",
		cookies: []*http.Cookie{sessionCookie},
	}.do(t, srv)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204 for star, got %d", w.Code)
	}

	// Unstar repo
	w = request{
		method:  "DELETE",
		path:    "/api/v1/user/starred/testuser/testrepo",
		cookies: []*http.Cookie{sessionCookie},
	}.do(t, srv)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204 for unstar, got %d", w.Code)
	}
}

// ============================================
// Issue Tests
// ============================================

func createTestUserAndRepo(t *testing.T, srv *Server) *http.Cookie {
	t.Helper()

	// Register user
	w := request{
		method: "POST",
		path:   "/api/v1/auth/register",
		body: map[string]string{
			"username": "testuser",
			"email":    "test@example.com",
			"password": "password123",
		},
	}.do(t, srv)

	var sessionCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "session" {
			sessionCookie = c
			break
		}
	}

	// Create repo
	request{
		method: "POST",
		path:   "/api/v1/repos",
		body: map[string]interface{}{
			"name":       "testrepo",
			"is_private": false,
		},
		cookies: []*http.Cookie{sessionCookie},
	}.do(t, srv)

	return sessionCookie
}

func TestIssue_Create(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	sessionCookie := createTestUserAndRepo(t, srv)

	// Create issue
	w := request{
		method: "POST",
		path:   "/api/v1/repos/testuser/testrepo/issues",
		body: map[string]string{
			"title": "Test Issue",
			"body":  "This is a test issue",
		},
		cookies: []*http.Cookie{sessionCookie},
	}.do(t, srv)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestIssue_List(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	sessionCookie := createTestUserAndRepo(t, srv)

	// Create issue
	request{
		method: "POST",
		path:   "/api/v1/repos/testuser/testrepo/issues",
		body: map[string]string{
			"title": "Test Issue",
			"body":  "This is a test issue",
		},
		cookies: []*http.Cookie{sessionCookie},
	}.do(t, srv)

	// List issues
	w := request{
		method: "GET",
		path:   "/api/v1/repos/testuser/testrepo/issues",
	}.do(t, srv)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestIssue_Get(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	sessionCookie := createTestUserAndRepo(t, srv)

	// Create issue
	request{
		method: "POST",
		path:   "/api/v1/repos/testuser/testrepo/issues",
		body: map[string]string{
			"title": "Test Issue",
			"body":  "This is a test issue",
		},
		cookies: []*http.Cookie{sessionCookie},
	}.do(t, srv)

	// Get issue
	w := request{
		method: "GET",
		path:   "/api/v1/repos/testuser/testrepo/issues/1",
	}.do(t, srv)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestIssue_Update(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	sessionCookie := createTestUserAndRepo(t, srv)

	// Create issue
	request{
		method: "POST",
		path:   "/api/v1/repos/testuser/testrepo/issues",
		body: map[string]string{
			"title": "Test Issue",
			"body":  "This is a test issue",
		},
		cookies: []*http.Cookie{sessionCookie},
	}.do(t, srv)

	// Update issue
	w := request{
		method: "PATCH",
		path:   "/api/v1/repos/testuser/testrepo/issues/1",
		body: map[string]string{
			"title": "Updated Issue Title",
		},
		cookies: []*http.Cookie{sessionCookie},
	}.do(t, srv)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestIssue_LockUnlock(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	sessionCookie := createTestUserAndRepo(t, srv)

	// Create issue
	request{
		method: "POST",
		path:   "/api/v1/repos/testuser/testrepo/issues",
		body: map[string]string{
			"title": "Test Issue",
			"body":  "This is a test issue",
		},
		cookies: []*http.Cookie{sessionCookie},
	}.do(t, srv)

	// Lock issue
	w := request{
		method:  "PUT",
		path:    "/api/v1/repos/testuser/testrepo/issues/1/lock",
		cookies: []*http.Cookie{sessionCookie},
	}.do(t, srv)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204 for lock, got %d", w.Code)
	}

	// Unlock issue
	w = request{
		method:  "DELETE",
		path:    "/api/v1/repos/testuser/testrepo/issues/1/lock",
		cookies: []*http.Cookie{sessionCookie},
	}.do(t, srv)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204 for unlock, got %d", w.Code)
	}
}

func TestIssue_Comments(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	sessionCookie := createTestUserAndRepo(t, srv)

	// Create issue
	request{
		method: "POST",
		path:   "/api/v1/repos/testuser/testrepo/issues",
		body: map[string]string{
			"title": "Test Issue",
			"body":  "This is a test issue",
		},
		cookies: []*http.Cookie{sessionCookie},
	}.do(t, srv)

	// Add comment
	w := request{
		method: "POST",
		path:   "/api/v1/repos/testuser/testrepo/issues/1/comments",
		body: map[string]string{
			"body": "This is a test comment",
		},
		cookies: []*http.Cookie{sessionCookie},
	}.do(t, srv)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201 for add comment, got %d: %s", w.Code, w.Body.String())
	}

	// List comments
	w = request{
		method: "GET",
		path:   "/api/v1/repos/testuser/testrepo/issues/1/comments",
	}.do(t, srv)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 for list comments, got %d", w.Code)
	}
}

// ============================================
// Health Check Tests
// ============================================

func TestHealthCheck_Livez(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	w := request{
		method: "GET",
		path:   "/livez",
	}.do(t, srv)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestHealthCheck_Readyz(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	w := request{
		method: "GET",
		path:   "/readyz",
	}.do(t, srv)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

// ============================================
// Label Tests
// ============================================

func TestLabel_CRUD(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	sessionCookie := createTestUserAndRepo(t, srv)

	// Create label
	w := request{
		method: "POST",
		path:   "/api/v1/repos/testuser/testrepo/labels",
		body: map[string]string{
			"name":        "bug",
			"color":       "ff0000",
			"description": "Bug reports",
		},
		cookies: []*http.Cookie{sessionCookie},
	}.do(t, srv)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	// List labels
	w = request{
		method: "GET",
		path:   "/api/v1/repos/testuser/testrepo/labels",
	}.do(t, srv)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Get label
	w = request{
		method: "GET",
		path:   "/api/v1/repos/testuser/testrepo/labels/bug",
	}.do(t, srv)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Update label
	w = request{
		method: "PATCH",
		path:   "/api/v1/repos/testuser/testrepo/labels/bug",
		body: map[string]string{
			"color": "00ff00",
		},
		cookies: []*http.Cookie{sessionCookie},
	}.do(t, srv)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Delete label
	w = request{
		method:  "DELETE",
		path:    "/api/v1/repos/testuser/testrepo/labels/bug",
		cookies: []*http.Cookie{sessionCookie},
	}.do(t, srv)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", w.Code)
	}
}

// ============================================
// Milestone Tests
// ============================================

func TestMilestone_CRUD(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	sessionCookie := createTestUserAndRepo(t, srv)

	// Create milestone
	w := request{
		method: "POST",
		path:   "/api/v1/repos/testuser/testrepo/milestones",
		body: map[string]string{
			"title":       "v1.0",
			"description": "First release",
		},
		cookies: []*http.Cookie{sessionCookie},
	}.do(t, srv)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	// List milestones
	w = request{
		method: "GET",
		path:   "/api/v1/repos/testuser/testrepo/milestones",
	}.do(t, srv)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Get milestone
	w = request{
		method: "GET",
		path:   "/api/v1/repos/testuser/testrepo/milestones/1",
	}.do(t, srv)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Update milestone
	w = request{
		method: "PATCH",
		path:   "/api/v1/repos/testuser/testrepo/milestones/1",
		body: map[string]string{
			"description": "Updated description",
		},
		cookies: []*http.Cookie{sessionCookie},
	}.do(t, srv)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

// ============================================
// Organization Tests
// ============================================

func TestOrg_Create(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	// Register user
	w := request{
		method: "POST",
		path:   "/api/v1/auth/register",
		body: map[string]string{
			"username": "testuser",
			"email":    "test@example.com",
			"password": "password123",
		},
	}.do(t, srv)

	var sessionCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "session" {
			sessionCookie = c
			break
		}
	}

	// Create org
	w = request{
		method: "POST",
		path:   "/api/v1/orgs",
		body: map[string]string{
			"name":         "testorg",
			"display_name": "Test Organization",
		},
		cookies: []*http.Cookie{sessionCookie},
	}.do(t, srv)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestOrg_List(t *testing.T) {
	srv, cleanup := testServer(t)
	defer cleanup()

	w := request{
		method: "GET",
		path:   "/api/v1/orgs",
	}.do(t, srv)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

// ============================================
// Database Helper Test
// ============================================

func TestDatabase_Connection(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "githome-db-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := fmt.Sprintf("%s/test.db", tmpDir)
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	store, err := duckdb.New(db)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	if err := store.Ensure(context.Background()); err != nil {
		t.Fatalf("failed to ensure schema: %v", err)
	}

	if err := db.Ping(); err != nil {
		t.Fatalf("failed to ping database: %v", err)
	}
}
