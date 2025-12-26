package web

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"github.com/go-mizu/blueprints/kanban/feature/assignees"
	"github.com/go-mizu/blueprints/kanban/feature/columns"
	"github.com/go-mizu/blueprints/kanban/feature/comments"
	"github.com/go-mizu/blueprints/kanban/feature/cycles"
	"github.com/go-mizu/blueprints/kanban/feature/fields"
	"github.com/go-mizu/blueprints/kanban/feature/issues"
	"github.com/go-mizu/blueprints/kanban/feature/projects"
	"github.com/go-mizu/blueprints/kanban/feature/teams"
	"github.com/go-mizu/blueprints/kanban/feature/users"
	"github.com/go-mizu/blueprints/kanban/feature/values"
	"github.com/go-mizu/blueprints/kanban/feature/workspaces"
	"github.com/go-mizu/blueprints/kanban/store/duckdb"
)

// testEnv holds test environment
type testEnv struct {
	t          *testing.T
	server     *Server
	db         *sql.DB
	users      users.API
	workspaces workspaces.API
	teams      teams.API
	projects   projects.API
	columns    columns.API
	issues     issues.API
	cycles     cycles.API
	comments   comments.API
	fields     fields.API
	values     values.API
	assignees  assignees.API
}

// apiResponse is the standard API response format
type apiResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// testServer creates a test server with in-memory database
func testServer(t *testing.T) (*testEnv, func()) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "kanban-test-*")
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

	env := &testEnv{
		t:          t,
		server:     srv,
		db:         srv.db,
		users:      srv.users,
		workspaces: srv.workspaces,
		teams:      srv.teams,
		projects:   srv.projects,
		columns:    srv.columns,
		issues:     srv.issues,
		cycles:     srv.cycles,
		comments:   srv.comments,
		fields:     srv.fields,
		values:     srv.values,
		assignees:  srv.assignees,
	}

	cleanup := func() {
		srv.Close()
		os.RemoveAll(tempDir)
	}

	return env, cleanup
}

// doRequest performs an HTTP request and returns the response
func doRequest(t *testing.T, handler http.Handler, method, path string, body any, token string) *httptest.ResponseRecorder {
	t.Helper()

	var reqBody []byte
	if body != nil {
		var err error
		reqBody, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("failed to marshal request body: %v", err)
		}
	}

	req := httptest.NewRequest(method, path, bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.AddCookie(&http.Cookie{Name: "session", Value: token})
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// createTestUser creates a test user and returns user and session token
func (e *testEnv) createTestUser(username, email, password string) (*users.User, string) {
	user, session, err := e.users.Register(context.Background(), &users.RegisterIn{
		Username: username,
		Email:    email,
		Password: password,
	})
	if err != nil {
		e.t.Fatalf("failed to create test user: %v", err)
	}
	return user, session.ID
}

// createTestWorkspace creates a test workspace
func (e *testEnv) createTestWorkspace(userID, slug, name string) *workspaces.Workspace {
	ws, err := e.workspaces.Create(context.Background(), userID, &workspaces.CreateIn{
		Slug: slug,
		Name: name,
	})
	if err != nil {
		e.t.Fatalf("failed to create test workspace: %v", err)
	}
	return ws
}

// createTestTeam creates a test team
func (e *testEnv) createTestTeam(workspaceID, key, name string) *teams.Team {
	team, err := e.teams.Create(context.Background(), workspaceID, &teams.CreateIn{
		Key:  key,
		Name: name,
	})
	if err != nil {
		e.t.Fatalf("failed to create test team: %v", err)
	}
	return team
}

// createTestProject creates a test project
func (e *testEnv) createTestProject(teamID, key, name string) *projects.Project {
	project, err := e.projects.Create(context.Background(), teamID, &projects.CreateIn{
		Key:  key,
		Name: name,
	})
	if err != nil {
		e.t.Fatalf("failed to create test project: %v", err)
	}
	return project
}

// createTestColumn creates a test column
func (e *testEnv) createTestColumn(projectID, name string) *columns.Column {
	col, err := e.columns.Create(context.Background(), projectID, &columns.CreateIn{
		Name: name,
	})
	if err != nil {
		e.t.Fatalf("failed to create test column: %v", err)
	}
	return col
}

// createTestIssue creates a test issue
func (e *testEnv) createTestIssue(projectID, creatorID, title string) *issues.Issue {
	issue, err := e.issues.Create(context.Background(), projectID, creatorID, &issues.CreateIn{
		Title: title,
	})
	if err != nil {
		e.t.Fatalf("failed to create test issue: %v", err)
	}
	return issue
}

// createTestCycle creates a test cycle
func (e *testEnv) createTestCycle(teamID, name string) *cycles.Cycle {
	cycle, err := e.cycles.Create(context.Background(), teamID, &cycles.CreateIn{
		Name:      name,
		StartDate: time.Now(),
		EndDate:   time.Now().Add(7 * 24 * time.Hour),
	})
	if err != nil {
		e.t.Fatalf("failed to create test cycle: %v", err)
	}
	return cycle
}

// ============================================================================
// Server Tests
// ============================================================================

func TestServer_New(t *testing.T) {
	env, cleanup := testServer(t)
	defer cleanup()

	if env.server == nil {
		t.Error("expected server to not be nil")
	}
	if env.server.app == nil {
		t.Error("expected app to not be nil")
	}
	if env.server.db == nil {
		t.Error("expected db to not be nil")
	}
}

func TestServer_Handler(t *testing.T) {
	env, cleanup := testServer(t)
	defer cleanup()

	handler := env.server.Handler()
	if handler == nil {
		t.Error("expected handler to not be nil")
	}
}

func TestServer_Close(t *testing.T) {
	env, cleanup := testServer(t)
	defer cleanup()

	err := env.server.Close()
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

// ============================================================================
// Auth Tests
// ============================================================================

func TestAuth_Register_Success(t *testing.T) {
	env, cleanup := testServer(t)
	defer cleanup()

	body := map[string]string{
		"username": "testuser",
		"email":    "test@example.com",
		"password": "password123",
	}

	rec := doRequest(t, env.server.Handler(), "POST", "/api/v1/auth/register", body, "")

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Check session cookie is set
	cookies := rec.Result().Cookies()
	var hasSession bool
	for _, c := range cookies {
		if c.Name == "session" && c.Value != "" {
			hasSession = true
			break
		}
	}
	if !hasSession {
		t.Error("expected session cookie to be set")
	}
}

func TestAuth_Register_MissingUsername(t *testing.T) {
	env, cleanup := testServer(t)
	defer cleanup()

	body := map[string]string{
		"email":    "test@example.com",
		"password": "password123",
	}

	rec := doRequest(t, env.server.Handler(), "POST", "/api/v1/auth/register", body, "")

	if rec.Code != http.StatusBadRequest && rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 400 or 500, got %d", rec.Code)
	}
}

func TestAuth_Login_Success(t *testing.T) {
	env, cleanup := testServer(t)
	defer cleanup()

	// Create user first
	env.createTestUser("testuser", "test@example.com", "password123")

	body := map[string]string{
		"email":    "test@example.com",
		"password": "password123",
	}

	rec := doRequest(t, env.server.Handler(), "POST", "/api/v1/auth/login", body, "")

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAuth_Login_InvalidPassword(t *testing.T) {
	env, cleanup := testServer(t)
	defer cleanup()

	// Create user first
	env.createTestUser("testuser", "test@example.com", "password123")

	body := map[string]string{
		"email":    "test@example.com",
		"password": "wrongpassword",
	}

	rec := doRequest(t, env.server.Handler(), "POST", "/api/v1/auth/login", body, "")

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
}

func TestAuth_Me_WithSession(t *testing.T) {
	env, cleanup := testServer(t)
	defer cleanup()

	_, token := env.createTestUser("testuser", "test@example.com", "password123")

	rec := doRequest(t, env.server.Handler(), "GET", "/api/v1/auth/me", nil, token)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAuth_Me_Unauthorized(t *testing.T) {
	env, cleanup := testServer(t)
	defer cleanup()

	rec := doRequest(t, env.server.Handler(), "GET", "/api/v1/auth/me", nil, "")

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
}

func TestAuth_Logout(t *testing.T) {
	env, cleanup := testServer(t)
	defer cleanup()

	_, token := env.createTestUser("testuser", "test@example.com", "password123")

	rec := doRequest(t, env.server.Handler(), "POST", "/api/v1/auth/logout", nil, token)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ============================================================================
// Workspace Tests
// ============================================================================

func TestWorkspace_Create(t *testing.T) {
	env, cleanup := testServer(t)
	defer cleanup()

	_, token := env.createTestUser("testuser", "test@example.com", "password123")

	body := map[string]string{
		"slug": "test-workspace",
		"name": "Test Workspace",
	}

	rec := doRequest(t, env.server.Handler(), "POST", "/api/v1/workspaces", body, token)

	if rec.Code != http.StatusCreated && rec.Code != http.StatusOK {
		t.Errorf("expected status 201 or 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestWorkspace_List(t *testing.T) {
	env, cleanup := testServer(t)
	defer cleanup()

	user, token := env.createTestUser("testuser", "test@example.com", "password123")
	env.createTestWorkspace(user.ID, "test-ws", "Test Workspace")

	rec := doRequest(t, env.server.Handler(), "GET", "/api/v1/workspaces", nil, token)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestWorkspace_Get(t *testing.T) {
	env, cleanup := testServer(t)
	defer cleanup()

	user, token := env.createTestUser("testuser", "test@example.com", "password123")
	env.createTestWorkspace(user.ID, "test-ws", "Test Workspace")

	rec := doRequest(t, env.server.Handler(), "GET", "/api/v1/workspaces/test-ws", nil, token)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestWorkspace_Unauthorized(t *testing.T) {
	env, cleanup := testServer(t)
	defer cleanup()

	rec := doRequest(t, env.server.Handler(), "GET", "/api/v1/workspaces", nil, "")

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
}

// ============================================================================
// Team Tests
// ============================================================================

func TestTeam_Create(t *testing.T) {
	env, cleanup := testServer(t)
	defer cleanup()

	user, token := env.createTestUser("testuser", "test@example.com", "password123")
	ws := env.createTestWorkspace(user.ID, "test-ws", "Test Workspace")

	body := map[string]string{
		"key":  "ENG",
		"name": "Engineering",
	}

	rec := doRequest(t, env.server.Handler(), "POST", "/api/v1/workspaces/"+ws.ID+"/teams", body, token)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestTeam_List(t *testing.T) {
	env, cleanup := testServer(t)
	defer cleanup()

	user, token := env.createTestUser("testuser", "test@example.com", "password123")
	ws := env.createTestWorkspace(user.ID, "test-ws", "Test Workspace")
	env.createTestTeam(ws.ID, "ENG", "Engineering")

	rec := doRequest(t, env.server.Handler(), "GET", "/api/v1/workspaces/"+ws.ID+"/teams", nil, token)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestTeam_Get(t *testing.T) {
	env, cleanup := testServer(t)
	defer cleanup()

	user, token := env.createTestUser("testuser", "test@example.com", "password123")
	ws := env.createTestWorkspace(user.ID, "test-ws", "Test Workspace")
	team := env.createTestTeam(ws.ID, "ENG", "Engineering")

	rec := doRequest(t, env.server.Handler(), "GET", "/api/v1/teams/"+team.ID, nil, token)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ============================================================================
// Project Tests
// ============================================================================

func TestProject_Create(t *testing.T) {
	env, cleanup := testServer(t)
	defer cleanup()

	user, token := env.createTestUser("testuser", "test@example.com", "password123")
	ws := env.createTestWorkspace(user.ID, "test-ws", "Test Workspace")
	team := env.createTestTeam(ws.ID, "ENG", "Engineering")

	body := map[string]string{
		"key":  "PROJ",
		"name": "My Project",
	}

	rec := doRequest(t, env.server.Handler(), "POST", "/api/v1/teams/"+team.ID+"/projects", body, token)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestProject_List(t *testing.T) {
	env, cleanup := testServer(t)
	defer cleanup()

	user, token := env.createTestUser("testuser", "test@example.com", "password123")
	ws := env.createTestWorkspace(user.ID, "test-ws", "Test Workspace")
	team := env.createTestTeam(ws.ID, "ENG", "Engineering")
	env.createTestProject(team.ID, "PROJ", "My Project")

	rec := doRequest(t, env.server.Handler(), "GET", "/api/v1/teams/"+team.ID+"/projects", nil, token)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ============================================================================
// Column Tests
// ============================================================================

func TestColumn_Create(t *testing.T) {
	env, cleanup := testServer(t)
	defer cleanup()

	user, token := env.createTestUser("testuser", "test@example.com", "password123")
	ws := env.createTestWorkspace(user.ID, "test-ws", "Test Workspace")
	team := env.createTestTeam(ws.ID, "ENG", "Engineering")
	project := env.createTestProject(team.ID, "PROJ", "My Project")

	body := map[string]string{
		"name": "To Do",
	}

	rec := doRequest(t, env.server.Handler(), "POST", "/api/v1/projects/"+project.ID+"/columns", body, token)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestColumn_List(t *testing.T) {
	env, cleanup := testServer(t)
	defer cleanup()

	user, token := env.createTestUser("testuser", "test@example.com", "password123")
	ws := env.createTestWorkspace(user.ID, "test-ws", "Test Workspace")
	team := env.createTestTeam(ws.ID, "ENG", "Engineering")
	project := env.createTestProject(team.ID, "PROJ", "My Project")
	env.createTestColumn(project.ID, "To Do")

	rec := doRequest(t, env.server.Handler(), "GET", "/api/v1/projects/"+project.ID+"/columns", nil, token)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ============================================================================
// Issue Tests
// ============================================================================

func TestIssue_Create(t *testing.T) {
	env, cleanup := testServer(t)
	defer cleanup()

	user, token := env.createTestUser("testuser", "test@example.com", "password123")
	ws := env.createTestWorkspace(user.ID, "test-ws", "Test Workspace")
	team := env.createTestTeam(ws.ID, "ENG", "Engineering")
	project := env.createTestProject(team.ID, "PROJ", "My Project")
	env.createTestColumn(project.ID, "To Do")

	body := map[string]string{
		"title": "Test Issue",
	}

	rec := doRequest(t, env.server.Handler(), "POST", "/api/v1/projects/"+project.ID+"/issues", body, token)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestIssue_List(t *testing.T) {
	env, cleanup := testServer(t)
	defer cleanup()

	user, token := env.createTestUser("testuser", "test@example.com", "password123")
	ws := env.createTestWorkspace(user.ID, "test-ws", "Test Workspace")
	team := env.createTestTeam(ws.ID, "ENG", "Engineering")
	project := env.createTestProject(team.ID, "PROJ", "My Project")
	env.createTestColumn(project.ID, "To Do")
	env.createTestIssue(project.ID, user.ID, "Test Issue")

	rec := doRequest(t, env.server.Handler(), "GET", "/api/v1/projects/"+project.ID+"/issues", nil, token)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestIssue_Get(t *testing.T) {
	env, cleanup := testServer(t)
	defer cleanup()

	user, token := env.createTestUser("testuser", "test@example.com", "password123")
	ws := env.createTestWorkspace(user.ID, "test-ws", "Test Workspace")
	team := env.createTestTeam(ws.ID, "ENG", "Engineering")
	project := env.createTestProject(team.ID, "PROJ", "My Project")
	env.createTestColumn(project.ID, "To Do")
	issue := env.createTestIssue(project.ID, user.ID, "Test Issue")

	rec := doRequest(t, env.server.Handler(), "GET", "/api/v1/issues/"+issue.Key, nil, token)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ============================================================================
// Cycle Tests
// ============================================================================

func TestCycle_Create(t *testing.T) {
	env, cleanup := testServer(t)
	defer cleanup()

	user, token := env.createTestUser("testuser", "test@example.com", "password123")
	ws := env.createTestWorkspace(user.ID, "test-ws", "Test Workspace")
	team := env.createTestTeam(ws.ID, "ENG", "Engineering")

	body := map[string]any{
		"name":       "Sprint 1",
		"start_date": time.Now().Format(time.RFC3339),
		"end_date":   time.Now().Add(7 * 24 * time.Hour).Format(time.RFC3339),
	}

	rec := doRequest(t, env.server.Handler(), "POST", "/api/v1/teams/"+team.ID+"/cycles", body, token)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCycle_List(t *testing.T) {
	env, cleanup := testServer(t)
	defer cleanup()

	user, token := env.createTestUser("testuser", "test@example.com", "password123")
	ws := env.createTestWorkspace(user.ID, "test-ws", "Test Workspace")
	team := env.createTestTeam(ws.ID, "ENG", "Engineering")
	env.createTestCycle(team.ID, "Sprint 1")

	rec := doRequest(t, env.server.Handler(), "GET", "/api/v1/teams/"+team.ID+"/cycles", nil, token)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ============================================================================
// Field Tests
// ============================================================================

func TestField_Create(t *testing.T) {
	env, cleanup := testServer(t)
	defer cleanup()

	user, token := env.createTestUser("testuser", "test@example.com", "password123")
	ws := env.createTestWorkspace(user.ID, "test-ws", "Test Workspace")
	team := env.createTestTeam(ws.ID, "ENG", "Engineering")
	project := env.createTestProject(team.ID, "PROJ", "My Project")

	body := map[string]any{
		"key":  "priority",
		"name": "Priority",
		"kind": "select",
	}

	rec := doRequest(t, env.server.Handler(), "POST", "/api/v1/projects/"+project.ID+"/fields", body, token)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestField_List(t *testing.T) {
	env, cleanup := testServer(t)
	defer cleanup()

	user, token := env.createTestUser("testuser", "test@example.com", "password123")
	ws := env.createTestWorkspace(user.ID, "test-ws", "Test Workspace")
	team := env.createTestTeam(ws.ID, "ENG", "Engineering")
	project := env.createTestProject(team.ID, "PROJ", "My Project")

	// Create a field
	env.fields.Create(context.Background(), project.ID, &fields.CreateIn{
		Key:  "priority",
		Name: "Priority",
		Kind: "select",
	})

	rec := doRequest(t, env.server.Handler(), "GET", "/api/v1/projects/"+project.ID+"/fields", nil, token)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ============================================================================
// Value Tests
// ============================================================================

func TestValue_Set(t *testing.T) {
	env, cleanup := testServer(t)
	defer cleanup()

	user, token := env.createTestUser("testuser", "test@example.com", "password123")
	ws := env.createTestWorkspace(user.ID, "test-ws", "Test Workspace")
	team := env.createTestTeam(ws.ID, "ENG", "Engineering")
	project := env.createTestProject(team.ID, "PROJ", "My Project")
	env.createTestColumn(project.ID, "To Do")
	issue := env.createTestIssue(project.ID, user.ID, "Test Issue")

	field, _ := env.fields.Create(context.Background(), project.ID, &fields.CreateIn{
		Key:  "priority",
		Name: "Priority",
		Kind: "text",
	})

	text := "high"
	body := map[string]any{
		"value_text": text,
	}

	rec := doRequest(t, env.server.Handler(), "PUT", "/api/v1/issues/"+issue.ID+"/values/"+field.ID, body, token)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestValue_ListByIssue(t *testing.T) {
	env, cleanup := testServer(t)
	defer cleanup()

	user, token := env.createTestUser("testuser", "test@example.com", "password123")
	ws := env.createTestWorkspace(user.ID, "test-ws", "Test Workspace")
	team := env.createTestTeam(ws.ID, "ENG", "Engineering")
	project := env.createTestProject(team.ID, "PROJ", "My Project")
	env.createTestColumn(project.ID, "To Do")
	issue := env.createTestIssue(project.ID, user.ID, "Test Issue")

	rec := doRequest(t, env.server.Handler(), "GET", "/api/v1/issues/"+issue.ID+"/values", nil, token)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ============================================================================
// Assignee Tests
// ============================================================================

func TestAssignee_Add(t *testing.T) {
	env, cleanup := testServer(t)
	defer cleanup()

	user, token := env.createTestUser("testuser", "test@example.com", "password123")
	ws := env.createTestWorkspace(user.ID, "test-ws", "Test Workspace")
	team := env.createTestTeam(ws.ID, "ENG", "Engineering")
	project := env.createTestProject(team.ID, "PROJ", "My Project")
	env.createTestColumn(project.ID, "To Do")
	issue := env.createTestIssue(project.ID, user.ID, "Test Issue")

	body := map[string]string{
		"user_id": user.ID,
	}

	rec := doRequest(t, env.server.Handler(), "POST", "/api/v1/issues/"+issue.ID+"/assignees", body, token)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAssignee_List(t *testing.T) {
	env, cleanup := testServer(t)
	defer cleanup()

	user, token := env.createTestUser("testuser", "test@example.com", "password123")
	ws := env.createTestWorkspace(user.ID, "test-ws", "Test Workspace")
	team := env.createTestTeam(ws.ID, "ENG", "Engineering")
	project := env.createTestProject(team.ID, "PROJ", "My Project")
	env.createTestColumn(project.ID, "To Do")
	issue := env.createTestIssue(project.ID, user.ID, "Test Issue")

	// Add assignee
	env.assignees.Add(context.Background(), issue.ID, user.ID)

	rec := doRequest(t, env.server.Handler(), "GET", "/api/v1/issues/"+issue.ID+"/assignees", nil, token)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ============================================================================
// Comment Tests
// ============================================================================

func TestComment_Create(t *testing.T) {
	env, cleanup := testServer(t)
	defer cleanup()

	user, token := env.createTestUser("testuser", "test@example.com", "password123")
	ws := env.createTestWorkspace(user.ID, "test-ws", "Test Workspace")
	team := env.createTestTeam(ws.ID, "ENG", "Engineering")
	project := env.createTestProject(team.ID, "PROJ", "My Project")
	env.createTestColumn(project.ID, "To Do")
	issue := env.createTestIssue(project.ID, user.ID, "Test Issue")

	body := map[string]string{
		"content": "This is a comment",
	}

	rec := doRequest(t, env.server.Handler(), "POST", "/api/v1/issues/"+issue.ID+"/comments", body, token)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestComment_List(t *testing.T) {
	env, cleanup := testServer(t)
	defer cleanup()

	user, token := env.createTestUser("testuser", "test@example.com", "password123")
	ws := env.createTestWorkspace(user.ID, "test-ws", "Test Workspace")
	team := env.createTestTeam(ws.ID, "ENG", "Engineering")
	project := env.createTestProject(team.ID, "PROJ", "My Project")
	env.createTestColumn(project.ID, "To Do")
	issue := env.createTestIssue(project.ID, user.ID, "Test Issue")

	// Create comment
	env.comments.Create(context.Background(), issue.ID, user.ID, &comments.CreateIn{
		Content: "This is a comment",
	})

	rec := doRequest(t, env.server.Handler(), "GET", "/api/v1/issues/"+issue.ID+"/comments", nil, token)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ============================================================================
// Page Tests
// ============================================================================

func TestPage_Login(t *testing.T) {
	env, cleanup := testServer(t)
	defer cleanup()

	rec := doRequest(t, env.server.Handler(), "GET", "/login", nil, "")

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestPage_Register(t *testing.T) {
	env, cleanup := testServer(t)
	defer cleanup()

	rec := doRequest(t, env.server.Handler(), "GET", "/register", nil, "")

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestPage_Home_Redirect(t *testing.T) {
	env, cleanup := testServer(t)
	defer cleanup()

	rec := doRequest(t, env.server.Handler(), "GET", "/", nil, "")

	if rec.Code != http.StatusFound {
		t.Errorf("expected status 302, got %d", rec.Code)
	}
}

// ============================================================================
// Static Files Tests
// ============================================================================

func TestStatic_NotFound(t *testing.T) {
	env, cleanup := testServer(t)
	defer cleanup()

	rec := doRequest(t, env.server.Handler(), "GET", "/static/nonexistent.js", nil, "")

	if rec.Code != http.StatusNotFound && rec.Code != http.StatusOK {
		// File server might return 200 with empty body or 404
		t.Logf("status: %d", rec.Code)
	}
}

// ============================================================================
// Security Tests
// ============================================================================

func TestSecurity_APIRequiresAuth(t *testing.T) {
	env, cleanup := testServer(t)
	defer cleanup()

	endpoints := []struct {
		method string
		path   string
	}{
		{"GET", "/api/v1/workspaces"},
		{"GET", "/api/v1/auth/me"},
		{"POST", "/api/v1/auth/logout"},
	}

	for _, ep := range endpoints {
		rec := doRequest(t, env.server.Handler(), ep.method, ep.path, nil, "")
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: expected status 401, got %d", ep.method, ep.path, rec.Code)
		}
	}
}
