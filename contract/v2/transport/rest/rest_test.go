package rest_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-mizu/mizu"
	contract "github.com/go-mizu/mizu/contract/v2"
	"github.com/go-mizu/mizu/contract/v2/transport/rest"
)

// --- Test API Definition ---

type TodoAPI interface {
	Create(ctx context.Context, in *CreateInput) (*Todo, error)
	List(ctx context.Context) (*ListOutput, error)
	Get(ctx context.Context, in *GetInput) (*Todo, error)
	Update(ctx context.Context, in *UpdateInput) (*Todo, error)
	Delete(ctx context.Context, in *DeleteInput) error
}

type CreateInput struct {
	Title string `json:"title"`
	Body  string `json:"body,omitempty"`
}

type GetInput struct {
	ID string `json:"id"`
}

type UpdateInput struct {
	ID    string `json:"id"`
	Title string `json:"title,omitempty"`
	Body  string `json:"body,omitempty"`
}

type DeleteInput struct {
	ID string `json:"id"`
}

type Todo struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

type ListOutput struct {
	Items []*Todo `json:"items"`
	Total int     `json:"total"`
}

// --- Test Implementation ---

type todoService struct {
	todos map[string]*Todo
	err   error // inject error for testing
}

func newTodoService() *todoService {
	return &todoService{
		todos: make(map[string]*Todo),
	}
}

func (s *todoService) Create(ctx context.Context, in *CreateInput) (*Todo, error) {
	if s.err != nil {
		return nil, s.err
	}
	todo := &Todo{
		ID:    "todo-1",
		Title: in.Title,
		Body:  in.Body,
	}
	s.todos[todo.ID] = todo
	return todo, nil
}

func (s *todoService) List(ctx context.Context) (*ListOutput, error) {
	if s.err != nil {
		return nil, s.err
	}
	items := make([]*Todo, 0, len(s.todos))
	for _, t := range s.todos {
		items = append(items, t)
	}
	return &ListOutput{Items: items, Total: len(items)}, nil
}

func (s *todoService) Get(ctx context.Context, in *GetInput) (*Todo, error) {
	if s.err != nil {
		return nil, s.err
	}
	t, ok := s.todos[in.ID]
	if !ok {
		return nil, errors.New("not found")
	}
	return t, nil
}

func (s *todoService) Update(ctx context.Context, in *UpdateInput) (*Todo, error) {
	if s.err != nil {
		return nil, s.err
	}
	t, ok := s.todos[in.ID]
	if !ok {
		return nil, errors.New("not found")
	}
	if in.Title != "" {
		t.Title = in.Title
	}
	if in.Body != "" {
		t.Body = in.Body
	}
	return t, nil
}

func (s *todoService) Delete(ctx context.Context, in *DeleteInput) error {
	if s.err != nil {
		return s.err
	}
	delete(s.todos, in.ID)
	return nil
}

// --- Test Helpers ---

func newTestInvoker(svc *todoService) contract.Invoker {
	return contract.Register[TodoAPI](svc,
		contract.WithDefaultResource("todos"),
	)
}

// --- Mount Tests ---

func TestMount(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	if err := rest.Mount(r, inv); err != nil {
		t.Fatalf("Mount failed: %v", err)
	}

	ts := httptest.NewServer(r)
	defer ts.Close()

	// Test POST /todos (create)
	resp, err := http.Post(ts.URL+"/todos", "application/json", strings.NewReader(`{"title":"Test Todo"}`))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, body)
	}

	var todo Todo
	if err := json.NewDecoder(resp.Body).Decode(&todo); err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	if todo.Title != "Test Todo" {
		t.Errorf("Expected title 'Test Todo', got '%s'", todo.Title)
	}
}

func TestMountAt(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	if err := rest.MountAt(r, "/api/v1", inv); err != nil {
		t.Fatalf("MountAt failed: %v", err)
	}

	ts := httptest.NewServer(r)
	defer ts.Close()

	// Test POST /api/v1/todos (create)
	resp, err := http.Post(ts.URL+"/api/v1/todos", "application/json", strings.NewReader(`{"title":"Test"}`))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, body)
	}
}

func TestMountNilRouter(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	err := rest.Mount(nil, inv)
	if err == nil {
		t.Error("Expected error for nil router")
	}
}

func TestMountNilInvoker(t *testing.T) {
	r := mizu.NewRouter()
	err := rest.Mount(r, nil)
	if err == nil {
		t.Error("Expected error for nil invoker")
	}
}

// --- Handler Tests ---

func TestHandler(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	handler, err := rest.Handler(inv)
	if err != nil {
		t.Fatalf("Handler failed: %v", err)
	}

	r := mizu.NewRouter()
	r.Handle("POST", "/todos", handler)
	r.Handle("GET", "/todos", handler)
	r.Handle("GET", "/todos/{id}", handler)

	ts := httptest.NewServer(r)
	defer ts.Close()

	// Create a todo
	resp, err := http.Post(ts.URL+"/todos", "application/json", strings.NewReader(`{"title":"Handler Test"}`))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestHandlerNotFound(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	handler, err := rest.Handler(inv)
	if err != nil {
		t.Fatalf("Handler failed: %v", err)
	}

	r := mizu.NewRouter()
	r.Handle("GET", "/unknown", handler)

	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/unknown")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

// --- Routes Tests ---

func TestRoutes(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	routes, err := rest.Routes(inv)
	if err != nil {
		t.Fatalf("Routes failed: %v", err)
	}

	if len(routes) == 0 {
		t.Fatal("Expected routes, got empty")
	}

	// Check for expected routes
	expectedRoutes := map[string]bool{
		"POST /todos":       false,
		"GET /todos":        false,
		"GET /todos/{id}":   false,
		"PUT /todos/{id}":   false,
		"DELETE /todos/{id}": false,
	}

	for _, rt := range routes {
		key := rt.Method + " " + rt.Path
		if _, ok := expectedRoutes[key]; ok {
			expectedRoutes[key] = true
		}
		if rt.Handler == nil {
			t.Errorf("Route %s has nil handler", key)
		}
	}

	for key, found := range expectedRoutes {
		if !found {
			t.Errorf("Expected route %s not found", key)
		}
	}
}

// --- Path Params Tests ---

func TestPathParams(t *testing.T) {
	svc := newTodoService()
	// Pre-populate a todo
	svc.todos["abc123"] = &Todo{ID: "abc123", Title: "Existing", Body: "Body"}

	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	if err := rest.Mount(r, inv); err != nil {
		t.Fatalf("Mount failed: %v", err)
	}

	ts := httptest.NewServer(r)
	defer ts.Close()

	// Test GET /todos/{id}
	resp, err := http.Get(ts.URL + "/todos/abc123")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, body)
	}

	var todo Todo
	if err := json.NewDecoder(resp.Body).Decode(&todo); err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	if todo.ID != "abc123" {
		t.Errorf("Expected ID 'abc123', got '%s'", todo.ID)
	}
}

// --- Query Params Tests ---

func TestQueryParams(t *testing.T) {
	svc := newTodoService()
	// Pre-populate a todo
	svc.todos["qp-test"] = &Todo{ID: "qp-test", Title: "Query", Body: "Test"}

	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	if err := rest.Mount(r, inv); err != nil {
		t.Fatalf("Mount failed: %v", err)
	}

	ts := httptest.NewServer(r)
	defer ts.Close()

	// Test GET /todos/{id} - path param should work
	resp, err := http.Get(ts.URL + "/todos/qp-test")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, body)
	}
}

// --- JSON Body Tests ---

func TestJSONBody(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	if err := rest.Mount(r, inv); err != nil {
		t.Fatalf("Mount failed: %v", err)
	}

	ts := httptest.NewServer(r)
	defer ts.Close()

	// Test POST with JSON body
	body := `{"title":"JSON Test","body":"This is the body"}`
	resp, err := http.Post(ts.URL+"/todos", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, respBody)
	}

	var todo Todo
	if err := json.NewDecoder(resp.Body).Decode(&todo); err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	if todo.Title != "JSON Test" {
		t.Errorf("Expected title 'JSON Test', got '%s'", todo.Title)
	}
	if todo.Body != "This is the body" {
		t.Errorf("Expected body 'This is the body', got '%s'", todo.Body)
	}
}

func TestInvalidJSON(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	if err := rest.Mount(r, inv); err != nil {
		t.Fatalf("Mount failed: %v", err)
	}

	ts := httptest.NewServer(r)
	defer ts.Close()

	// Test POST with invalid JSON
	resp, err := http.Post(ts.URL+"/todos", "application/json", strings.NewReader(`{invalid}`))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400 for invalid JSON, got %d", resp.StatusCode)
	}
}

// --- Error Mapping Tests ---

func TestDefaultErrorMapper(t *testing.T) {
	svc := newTodoService()
	svc.err = errors.New("something went wrong")
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	if err := rest.Mount(r, inv); err != nil {
		t.Fatalf("Mount failed: %v", err)
	}

	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/todos", "application/json", strings.NewReader(`{"title":"Test"}`))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}

	var errResp struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	if errResp.Error != "request_error" {
		t.Errorf("Expected error 'request_error', got '%s'", errResp.Error)
	}
}

var ErrNotFound = errors.New("not found")

func TestCustomErrorMapper(t *testing.T) {
	svc := newTodoService()
	svc.err = ErrNotFound
	inv := newTestInvoker(svc)

	customMapper := func(err error) (int, string, string) {
		if errors.Is(err, ErrNotFound) {
			return http.StatusNotFound, "not_found", "Resource not found"
		}
		return http.StatusInternalServerError, "internal_error", err.Error()
	}

	r := mizu.NewRouter()
	if err := rest.Mount(r, inv, rest.WithErrorMapper(customMapper)); err != nil {
		t.Fatalf("Mount failed: %v", err)
	}

	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/todos", "application/json", strings.NewReader(`{"title":"Test"}`))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}

	var errResp struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	if errResp.Error != "not_found" {
		t.Errorf("Expected error 'not_found', got '%s'", errResp.Error)
	}
}

// --- No Input Tests ---

func TestNoInput(t *testing.T) {
	svc := newTodoService()
	// Pre-populate
	svc.todos["1"] = &Todo{ID: "1", Title: "One"}
	svc.todos["2"] = &Todo{ID: "2", Title: "Two"}

	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	if err := rest.Mount(r, inv); err != nil {
		t.Fatalf("Mount failed: %v", err)
	}

	ts := httptest.NewServer(r)
	defer ts.Close()

	// Test GET /todos (list - no input)
	resp, err := http.Get(ts.URL + "/todos")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, body)
	}

	var list ListOutput
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	if list.Total != 2 {
		t.Errorf("Expected total 2, got %d", list.Total)
	}
}

// --- No Output Tests ---

func TestNoOutput(t *testing.T) {
	svc := newTodoService()
	svc.todos["del-1"] = &Todo{ID: "del-1", Title: "Delete Me"}

	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	if err := rest.Mount(r, inv); err != nil {
		t.Fatalf("Mount failed: %v", err)
	}

	ts := httptest.NewServer(r)
	defer ts.Close()

	// Test DELETE /todos/{id} (no output)
	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/todos/del-1", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 204, got %d: %s", resp.StatusCode, body)
	}
}

// --- Middleware Tests ---

func TestWithMiddleware(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	headerMiddleware := func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			c.Header().Set("X-Custom-Header", "test-value")
			return next(c)
		}
	}

	r := mizu.NewRouter()
	api := r.Prefix("/api").With(headerMiddleware)
	if err := rest.Mount(api, inv); err != nil {
		t.Fatalf("Mount failed: %v", err)
	}

	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/api/todos", "application/json", strings.NewReader(`{"title":"Middleware Test"}`))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, body)
	}

	if v := resp.Header.Get("X-Custom-Header"); v != "test-value" {
		t.Errorf("Expected X-Custom-Header 'test-value', got '%s'", v)
	}
}

// --- Options Tests ---

func TestWithMaxBodySize(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	// Set very small max body size
	if err := rest.Mount(r, inv, rest.WithMaxBodySize(10)); err != nil {
		t.Fatalf("Mount failed: %v", err)
	}

	ts := httptest.NewServer(r)
	defer ts.Close()

	// Large body should fail
	largeBody := `{"title":"This is a very long title that exceeds the max body size limit"}`
	resp, err := http.Post(ts.URL+"/todos", "application/json", strings.NewReader(largeBody))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400 for large body, got %d", resp.StatusCode)
	}
}

// --- OpenAPI Tests ---

func TestOpenAPI(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	spec, err := rest.OpenAPI(inv.Descriptor())
	if err != nil {
		t.Fatalf("OpenAPI failed: %v", err)
	}

	if len(spec) == 0 {
		t.Fatal("Expected non-empty OpenAPI spec")
	}

	// Parse and verify basic structure
	var doc map[string]any
	if err := json.Unmarshal(spec, &doc); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	if doc["openapi"] != "3.0.3" {
		t.Errorf("Expected openapi 3.0.3, got %v", doc["openapi"])
	}

	if doc["paths"] == nil {
		t.Error("Expected paths in OpenAPI doc")
	}
}

// --- Integration Test ---

func TestCRUDOperations(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	if err := rest.Mount(r, inv); err != nil {
		t.Fatalf("Mount failed: %v", err)
	}

	ts := httptest.NewServer(r)
	defer ts.Close()

	// 1. Create
	resp, _ := http.Post(ts.URL+"/todos", "application/json", strings.NewReader(`{"title":"CRUD Test","body":"Test body"}`))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Create failed: %d", resp.StatusCode)
	}
	var created Todo
	json.NewDecoder(resp.Body).Decode(&created)
	resp.Body.Close()

	// 2. Read (List)
	resp, _ = http.Get(ts.URL + "/todos")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("List failed: %d", resp.StatusCode)
	}
	var list ListOutput
	json.NewDecoder(resp.Body).Decode(&list)
	resp.Body.Close()
	if list.Total != 1 {
		t.Errorf("Expected 1 todo, got %d", list.Total)
	}

	// 3. Read (Get)
	resp, _ = http.Get(ts.URL + "/todos/" + created.ID)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Get failed: %d", resp.StatusCode)
	}
	var fetched Todo
	json.NewDecoder(resp.Body).Decode(&fetched)
	resp.Body.Close()
	if fetched.ID != created.ID {
		t.Errorf("Expected ID %s, got %s", created.ID, fetched.ID)
	}

	// 4. Update
	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/todos/"+created.ID, strings.NewReader(`{"title":"Updated Title"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Update failed: %d", resp.StatusCode)
	}
	var updated Todo
	json.NewDecoder(resp.Body).Decode(&updated)
	resp.Body.Close()
	if updated.Title != "Updated Title" {
		t.Errorf("Expected title 'Updated Title', got '%s'", updated.Title)
	}

	// 5. Delete
	req, _ = http.NewRequest(http.MethodDelete, ts.URL+"/todos/"+created.ID, nil)
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("Delete failed: %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 6. Verify deleted
	resp, _ = http.Get(ts.URL + "/todos")
	json.NewDecoder(resp.Body).Decode(&list)
	resp.Body.Close()
	if list.Total != 0 {
		t.Errorf("Expected 0 todos after delete, got %d", list.Total)
	}
}
