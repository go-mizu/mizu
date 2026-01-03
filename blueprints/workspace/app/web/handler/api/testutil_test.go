package api_test

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
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"github.com/go-mizu/blueprints/workspace/app/web"
	"github.com/go-mizu/blueprints/workspace/feature/users"
)

// TestServer wraps a test HTTP server with helper methods.
type TestServer struct {
	*httptest.Server
	t       *testing.T
	dataDir string
}

// NewTestServer creates a new test server with a fresh database.
func NewTestServer(t *testing.T) *TestServer {
	t.Helper()

	// Create temp data directory
	dataDir, err := os.MkdirTemp("", "workspace-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}

	// Create server
	srv, err := web.New(web.Config{
		Addr:    ":0",
		DataDir: dataDir,
		Dev:     true,
	})
	if err != nil {
		os.RemoveAll(dataDir)
		t.Fatalf("create server: %v", err)
	}

	// Create test server
	ts := httptest.NewServer(srv.Handler())

	return &TestServer{
		Server:  ts,
		t:       t,
		dataDir: dataDir,
	}
}

// Close cleans up the test server.
func (ts *TestServer) Close() {
	ts.Server.Close()
	os.RemoveAll(ts.dataDir)
}

// Request makes an HTTP request and returns the response.
func (ts *TestServer) Request(method, path string, body interface{}, cookies ...*http.Cookie) *http.Response {
	ts.t.Helper()

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			ts.t.Fatalf("marshal body: %v", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, ts.URL+path, bodyReader)
	if err != nil {
		ts.t.Fatalf("create request: %v", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	for _, c := range cookies {
		req.AddCookie(c)
	}

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		ts.t.Fatalf("do request: %v", err)
	}

	return resp
}

// ParseJSON parses the response body as JSON.
func (ts *TestServer) ParseJSON(resp *http.Response, v interface{}) {
	ts.t.Helper()
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		ts.t.Fatalf("decode response: %v", err)
	}
}

// ExpectStatus checks that the response has the expected status code.
func (ts *TestServer) ExpectStatus(resp *http.Response, expected int) {
	ts.t.Helper()
	if resp.StatusCode != expected {
		body, _ := io.ReadAll(resp.Body)
		ts.t.Errorf("expected status %d, got %d: %s", expected, resp.StatusCode, string(body))
	}
}

// Register registers a new user and returns the session cookie.
func (ts *TestServer) Register(email, name, password string) (*users.User, *http.Cookie) {
	ts.t.Helper()

	resp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email":    email,
		"name":     name,
		"password": password,
	})

	ts.ExpectStatus(resp, http.StatusCreated)

	var user users.User
	ts.ParseJSON(resp, &user)

	var sessionCookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == "workspace_session" {
			sessionCookie = c
			break
		}
	}

	if sessionCookie == nil {
		ts.t.Fatal("no session cookie returned")
	}

	return &user, sessionCookie
}

// Login logs in a user and returns the session cookie.
func (ts *TestServer) Login(email, password string) (*users.User, *http.Cookie) {
	ts.t.Helper()

	resp := ts.Request("POST", "/api/v1/auth/login", map[string]string{
		"email":    email,
		"password": password,
	})

	ts.ExpectStatus(resp, http.StatusOK)

	var user users.User
	ts.ParseJSON(resp, &user)

	var sessionCookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == "workspace_session" {
			sessionCookie = c
			break
		}
	}

	if sessionCookie == nil {
		ts.t.Fatal("no session cookie returned")
	}

	return &user, sessionCookie
}

// Standalone test helper functions

// setupTestDB creates a test database and returns cleanup function.
func setupTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()

	// Create temp file
	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("test-%d.duckdb", os.Getpid()))

	db, err := sql.Open("duckdb", tmpFile)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.Remove(tmpFile)
	}

	return db, cleanup
}

// ptr returns a pointer to the value.
func ptr[T any](v T) *T {
	return &v
}

// mustMarshal marshals the value to JSON or panics.
func mustMarshal(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}

// testContext returns a background context for tests.
func testContext() context.Context {
	return context.Background()
}
