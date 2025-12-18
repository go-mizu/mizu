package jsonrpc_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-mizu/mizu"
	contract "github.com/go-mizu/mizu/contract/v2"
	"github.com/go-mizu/mizu/contract/v2/transport/jsonrpc"
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

var errNotFound = errors.New("not found")

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
		return nil, errNotFound
	}
	return t, nil
}

func (s *todoService) Update(ctx context.Context, in *UpdateInput) (*Todo, error) {
	if s.err != nil {
		return nil, s.err
	}
	t, ok := s.todos[in.ID]
	if !ok {
		return nil, errNotFound
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

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
	ID      any    `json:"id,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func doRPC(ts *httptest.Server, req rpcRequest) (*rpcResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(ts.URL+"/rpc", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return nil, nil // notification
	}

	var rpcResp rpcResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, err
	}
	return &rpcResp, nil
}

func doBatchRPC(ts *httptest.Server, reqs []rpcRequest) ([]rpcResponse, error) {
	body, err := json.Marshal(reqs)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(ts.URL+"/rpc", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return nil, nil // all notifications
	}

	var rpcResps []rpcResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResps); err != nil {
		return nil, err
	}
	return rpcResps, nil
}

// --- Handler Tests ---

func TestHandler(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	handler, err := jsonrpc.Handler(inv)
	if err != nil {
		t.Fatalf("Handler failed: %v", err)
	}
	if handler == nil {
		t.Fatal("Handler returned nil")
	}
}

func TestHandlerNilInvoker(t *testing.T) {
	_, err := jsonrpc.Handler(nil)
	if err == nil {
		t.Fatal("Expected error for nil invoker")
	}
}

// --- Mount Tests ---

func TestMount(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	if err := jsonrpc.Mount(r, "/rpc", inv); err != nil {
		t.Fatalf("Mount failed: %v", err)
	}

	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, err := doRPC(ts, rpcRequest{
		JSONRPC: "2.0",
		Method:  "todos.create",
		Params:  map[string]any{"title": "Test Todo"},
		ID:      1,
	})
	if err != nil {
		t.Fatalf("RPC failed: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("RPC error: %+v", resp.Error)
	}

	var todo Todo
	if err := json.Unmarshal(resp.Result, &todo); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if todo.Title != "Test Todo" {
		t.Errorf("Expected title 'Test Todo', got '%s'", todo.Title)
	}
}

func TestMountEmptyPath(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	if err := jsonrpc.Mount(r, "", inv); err != nil {
		t.Fatalf("Mount failed: %v", err)
	}

	ts := httptest.NewServer(r)
	defer ts.Close()

	body, _ := json.Marshal(rpcRequest{
		JSONRPC: "2.0",
		Method:  "todos.list",
		ID:      1,
	})

	resp, err := http.Post(ts.URL+"/", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestMountNilRouter(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	if err := jsonrpc.Mount(nil, "/rpc", inv); err == nil {
		t.Fatal("Expected error for nil router")
	}
}

func TestMountNilInvoker(t *testing.T) {
	r := mizu.NewRouter()
	if err := jsonrpc.Mount(r, "/rpc", nil); err == nil {
		t.Fatal("Expected error for nil invoker")
	}
}

// --- Single Request Tests ---

func TestSingleRequest(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	jsonrpc.Mount(r, "/rpc", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Create a todo
	resp, err := doRPC(ts, rpcRequest{
		JSONRPC: "2.0",
		Method:  "todos.create",
		Params:  map[string]any{"title": "Test", "body": "Body"},
		ID:      1,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("Create error: %+v", resp.Error)
	}

	var todo Todo
	json.Unmarshal(resp.Result, &todo)
	if todo.ID == "" {
		t.Error("Expected todo ID")
	}

	// Get the todo
	resp, err = doRPC(ts, rpcRequest{
		JSONRPC: "2.0",
		Method:  "todos.get",
		Params:  map[string]any{"id": todo.ID},
		ID:      2,
	})
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("Get error: %+v", resp.Error)
	}

	var fetched Todo
	json.Unmarshal(resp.Result, &fetched)
	if fetched.Title != "Test" {
		t.Errorf("Expected title 'Test', got '%s'", fetched.Title)
	}
}

// --- Batch Request Tests ---

func TestBatchRequest(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	jsonrpc.Mount(r, "/rpc", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	resps, err := doBatchRPC(ts, []rpcRequest{
		{JSONRPC: "2.0", Method: "todos.create", Params: map[string]any{"title": "First"}, ID: 1},
		{JSONRPC: "2.0", Method: "todos.list", ID: 2},
	})
	if err != nil {
		t.Fatalf("Batch failed: %v", err)
	}

	if len(resps) != 2 {
		t.Fatalf("Expected 2 responses, got %d", len(resps))
	}

	for _, resp := range resps {
		if resp.Error != nil {
			t.Errorf("Unexpected error: %+v", resp.Error)
		}
	}
}

func TestEmptyBatch(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	jsonrpc.Mount(r, "/rpc", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/rpc", "application/json", strings.NewReader("[]"))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	var rpcResp rpcResponse
	json.NewDecoder(resp.Body).Decode(&rpcResp)
	if rpcResp.Error == nil {
		t.Error("Expected error for empty batch")
	}
	if rpcResp.Error.Code != -32600 {
		t.Errorf("Expected code -32600, got %d", rpcResp.Error.Code)
	}
}

// --- Notification Tests ---

func TestNotification(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	jsonrpc.Mount(r, "/rpc", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Request without ID is a notification
	resp, err := doRPC(ts, rpcRequest{
		JSONRPC: "2.0",
		Method:  "todos.create",
		Params:  map[string]any{"title": "Notification"},
		// No ID field
	})
	if err != nil {
		t.Fatalf("Notification failed: %v", err)
	}
	if resp != nil {
		t.Error("Expected no response for notification")
	}

	// Verify the todo was created
	listResp, _ := doRPC(ts, rpcRequest{
		JSONRPC: "2.0",
		Method:  "todos.list",
		ID:      1,
	})
	var list ListOutput
	json.Unmarshal(listResp.Result, &list)
	if list.Total != 1 {
		t.Errorf("Expected 1 todo, got %d", list.Total)
	}
}

func TestBatchAllNotifications(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	jsonrpc.Mount(r, "/rpc", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	resps, err := doBatchRPC(ts, []rpcRequest{
		{JSONRPC: "2.0", Method: "todos.create", Params: map[string]any{"title": "First"}},
		{JSONRPC: "2.0", Method: "todos.create", Params: map[string]any{"title": "Second"}},
	})
	if err != nil {
		t.Fatalf("Batch failed: %v", err)
	}
	if resps != nil {
		t.Error("Expected no response for all-notification batch")
	}
}

// --- Error Tests ---

func TestParseError(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	jsonrpc.Mount(r, "/rpc", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/rpc", "application/json", strings.NewReader("{invalid json"))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	var rpcResp rpcResponse
	json.NewDecoder(resp.Body).Decode(&rpcResp)
	if rpcResp.Error == nil {
		t.Fatal("Expected parse error")
	}
	if rpcResp.Error.Code != -32700 {
		t.Errorf("Expected code -32700, got %d", rpcResp.Error.Code)
	}
}

func TestInvalidRequest(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	jsonrpc.Mount(r, "/rpc", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Missing jsonrpc field
	resp, _ := doRPC(ts, rpcRequest{
		Method: "todos.list",
		ID:     1,
	})
	if resp.Error == nil {
		t.Fatal("Expected invalid request error")
	}
	if resp.Error.Code != -32600 {
		t.Errorf("Expected code -32600, got %d", resp.Error.Code)
	}

	// Missing method field
	resp, _ = doRPC(ts, rpcRequest{
		JSONRPC: "2.0",
		ID:      1,
	})
	if resp.Error == nil {
		t.Fatal("Expected invalid request error")
	}
	if resp.Error.Code != -32600 {
		t.Errorf("Expected code -32600, got %d", resp.Error.Code)
	}
}

func TestMethodNotFound(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	jsonrpc.Mount(r, "/rpc", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Unknown method
	resp, _ := doRPC(ts, rpcRequest{
		JSONRPC: "2.0",
		Method:  "todos.unknown",
		ID:      1,
	})
	if resp.Error == nil {
		t.Fatal("Expected method not found error")
	}
	if resp.Error.Code != -32601 {
		t.Errorf("Expected code -32601, got %d", resp.Error.Code)
	}

	// Invalid method format (no dot)
	resp, _ = doRPC(ts, rpcRequest{
		JSONRPC: "2.0",
		Method:  "noResourceDot",
		ID:      1,
	})
	if resp.Error == nil {
		t.Fatal("Expected method not found error")
	}
	if resp.Error.Code != -32601 {
		t.Errorf("Expected code -32601, got %d", resp.Error.Code)
	}
}

func TestInvalidParams(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	jsonrpc.Mount(r, "/rpc", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Array params not supported
	body, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"method":  "todos.create",
		"params":  []string{"title", "body"}, // array instead of object
		"id":      1,
	})
	resp, err := http.Post(ts.URL+"/rpc", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	var rpcResp rpcResponse
	json.NewDecoder(resp.Body).Decode(&rpcResp)
	if rpcResp.Error == nil {
		t.Fatal("Expected invalid params error")
	}
	if rpcResp.Error.Code != -32602 {
		t.Errorf("Expected code -32602, got %d", rpcResp.Error.Code)
	}
}

func TestServerError(t *testing.T) {
	svc := newTodoService()
	svc.err = errors.New("service error")
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	jsonrpc.Mount(r, "/rpc", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, _ := doRPC(ts, rpcRequest{
		JSONRPC: "2.0",
		Method:  "todos.list",
		ID:      1,
	})
	if resp.Error == nil {
		t.Fatal("Expected server error")
	}
	if resp.Error.Code != -32000 {
		t.Errorf("Expected code -32000, got %d", resp.Error.Code)
	}
}

// --- Option Tests ---

func TestErrorMapper(t *testing.T) {
	svc := newTodoService()
	svc.err = errNotFound
	inv := newTestInvoker(svc)

	customMapper := func(err error) (int, string, any) {
		if errors.Is(err, errNotFound) {
			return -32001, "not found", nil
		}
		return -32000, "server error", err.Error()
	}

	r := mizu.NewRouter()
	jsonrpc.Mount(r, "/rpc", inv, jsonrpc.WithErrorMapper(customMapper))
	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, _ := doRPC(ts, rpcRequest{
		JSONRPC: "2.0",
		Method:  "todos.list",
		ID:      1,
	})
	if resp.Error == nil {
		t.Fatal("Expected error")
	}
	if resp.Error.Code != -32001 {
		t.Errorf("Expected code -32001, got %d", resp.Error.Code)
	}
	if resp.Error.Message != "not found" {
		t.Errorf("Expected message 'not found', got '%s'", resp.Error.Message)
	}
}

func TestMaxBodySize(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	jsonrpc.Mount(r, "/rpc", inv, jsonrpc.WithMaxBodySize(100))
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Create a large request
	largeBody := strings.Repeat("x", 200)
	body, _ := json.Marshal(rpcRequest{
		JSONRPC: "2.0",
		Method:  "todos.create",
		Params:  map[string]any{"title": largeBody},
		ID:      1,
	})

	resp, err := http.Post(ts.URL+"/rpc", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	var rpcResp rpcResponse
	json.NewDecoder(resp.Body).Decode(&rpcResp)
	if rpcResp.Error == nil {
		t.Fatal("Expected body too large error")
	}
}

// --- HTTP Method Tests ---

func TestMethodNotAllowed(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	jsonrpc.Mount(r, "/rpc", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/rpc")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected 405, got %d", resp.StatusCode)
	}
}

func TestEmptyBody(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	jsonrpc.Mount(r, "/rpc", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/rpc", "application/json", strings.NewReader(""))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	var rpcResp rpcResponse
	json.NewDecoder(resp.Body).Decode(&rpcResp)
	if rpcResp.Error == nil {
		t.Fatal("Expected error for empty body")
	}
	if rpcResp.Error.Code != -32600 {
		t.Errorf("Expected code -32600, got %d", rpcResp.Error.Code)
	}
}

// --- No Input/Output Tests ---

func TestNoInput(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	jsonrpc.Mount(r, "/rpc", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	// List has no input
	resp, _ := doRPC(ts, rpcRequest{
		JSONRPC: "2.0",
		Method:  "todos.list",
		ID:      1,
	})
	if resp.Error != nil {
		t.Fatalf("List error: %+v", resp.Error)
	}
}

func TestNoOutput(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	jsonrpc.Mount(r, "/rpc", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	// First create a todo
	doRPC(ts, rpcRequest{
		JSONRPC: "2.0",
		Method:  "todos.create",
		Params:  map[string]any{"title": "Test"},
		ID:      1,
	})

	// Delete has no output
	resp, _ := doRPC(ts, rpcRequest{
		JSONRPC: "2.0",
		Method:  "todos.delete",
		Params:  map[string]any{"id": "todo-1"},
		ID:      2,
	})
	if resp.Error != nil {
		t.Fatalf("Delete error: %+v", resp.Error)
	}
	// Result should be null
	if string(resp.Result) != "null" {
		t.Errorf("Expected null result, got %s", resp.Result)
	}
}

// --- Middleware Test ---

func TestWithMiddleware(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	middlewareCalled := false
	middleware := func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			middlewareCalled = true
			return next(c)
		}
	}

	r := mizu.NewRouter()
	rpc := r.Prefix("/rpc").With(middleware)
	jsonrpc.Mount(rpc, "", inv)

	ts := httptest.NewServer(r)
	defer ts.Close()

	doRPC(ts, rpcRequest{
		JSONRPC: "2.0",
		Method:  "todos.list",
		ID:      1,
	})

	if !middlewareCalled {
		t.Error("Middleware was not called")
	}
}

// --- OpenRPC Test ---

func TestOpenRPC(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)
	desc := inv.Descriptor()

	spec, err := jsonrpc.OpenRPC(desc)
	if err != nil {
		t.Fatalf("OpenRPC failed: %v", err)
	}
	if len(spec) == 0 {
		t.Fatal("OpenRPC returned empty spec")
	}

	var doc map[string]any
	if err := json.Unmarshal(spec, &doc); err != nil {
		t.Fatalf("OpenRPC unmarshal failed: %v", err)
	}

	if doc["openrpc"] != "1.2.6" {
		t.Errorf("Expected openrpc version 1.2.6, got %v", doc["openrpc"])
	}

	methods, ok := doc["methods"].([]any)
	if !ok {
		t.Fatal("Expected methods array")
	}
	if len(methods) == 0 {
		t.Error("Expected at least one method")
	}
}

// --- Integration Test ---

func TestTodoAPI(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	jsonrpc.Mount(r, "/rpc", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Create
	createResp, _ := doRPC(ts, rpcRequest{
		JSONRPC: "2.0",
		Method:  "todos.create",
		Params:  map[string]any{"title": "Buy milk", "body": "From the store"},
		ID:      1,
	})
	if createResp.Error != nil {
		t.Fatalf("Create error: %+v", createResp.Error)
	}
	var created Todo
	json.Unmarshal(createResp.Result, &created)

	// List
	listResp, _ := doRPC(ts, rpcRequest{
		JSONRPC: "2.0",
		Method:  "todos.list",
		ID:      2,
	})
	if listResp.Error != nil {
		t.Fatalf("List error: %+v", listResp.Error)
	}
	var list ListOutput
	json.Unmarshal(listResp.Result, &list)
	if list.Total != 1 {
		t.Errorf("Expected 1 todo, got %d", list.Total)
	}

	// Get
	getResp, _ := doRPC(ts, rpcRequest{
		JSONRPC: "2.0",
		Method:  "todos.get",
		Params:  map[string]any{"id": created.ID},
		ID:      3,
	})
	if getResp.Error != nil {
		t.Fatalf("Get error: %+v", getResp.Error)
	}
	var fetched Todo
	json.Unmarshal(getResp.Result, &fetched)
	if fetched.Title != "Buy milk" {
		t.Errorf("Expected title 'Buy milk', got '%s'", fetched.Title)
	}

	// Update
	updateResp, _ := doRPC(ts, rpcRequest{
		JSONRPC: "2.0",
		Method:  "todos.update",
		Params:  map[string]any{"id": created.ID, "title": "Buy eggs"},
		ID:      4,
	})
	if updateResp.Error != nil {
		t.Fatalf("Update error: %+v", updateResp.Error)
	}
	var updated Todo
	json.Unmarshal(updateResp.Result, &updated)
	if updated.Title != "Buy eggs" {
		t.Errorf("Expected title 'Buy eggs', got '%s'", updated.Title)
	}

	// Delete
	deleteResp, _ := doRPC(ts, rpcRequest{
		JSONRPC: "2.0",
		Method:  "todos.delete",
		Params:  map[string]any{"id": created.ID},
		ID:      5,
	})
	if deleteResp.Error != nil {
		t.Fatalf("Delete error: %+v", deleteResp.Error)
	}

	// Verify deleted
	listResp, _ = doRPC(ts, rpcRequest{
		JSONRPC: "2.0",
		Method:  "todos.list",
		ID:      6,
	})
	json.Unmarshal(listResp.Result, &list)
	if list.Total != 0 {
		t.Errorf("Expected 0 todos after delete, got %d", list.Total)
	}
}

// --- Client Test ---

func TestClient(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	jsonrpc.Mount(r, "/rpc", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	client, err := jsonrpc.NewClient(inv.Descriptor())
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	client.Endpoint = ts.URL + "/rpc"

	// Create a todo
	var created Todo
	err = client.Call(context.Background(), "todos", "create", map[string]any{"title": "Client Test"}, &created)
	if err != nil {
		t.Fatalf("Client.Call failed: %v", err)
	}
	if created.Title != "Client Test" {
		t.Errorf("Expected title 'Client Test', got '%s'", created.Title)
	}

	// List todos
	var list ListOutput
	err = client.Call(context.Background(), "todos", "list", nil, &list)
	if err != nil {
		t.Fatalf("Client.Call failed: %v", err)
	}
	if list.Total != 1 {
		t.Errorf("Expected 1 todo, got %d", list.Total)
	}
}

func TestClientBatch(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	jsonrpc.Mount(r, "/rpc", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	client, _ := jsonrpc.NewClient(inv.Descriptor())
	client.Endpoint = ts.URL + "/rpc"

	results, err := client.BatchCall(context.Background(), []jsonrpc.BatchItem{
		{Resource: "todos", Method: "create", In: map[string]any{"title": "First"}},
		{Resource: "todos", Method: "create", In: map[string]any{"title": "Second"}},
		{Resource: "todos", Method: "list"},
	})
	if err != nil {
		t.Fatalf("BatchCall failed: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(results))
	}

	for i, res := range results {
		if res.Err != nil {
			t.Errorf("Result %d error: %v", i, res.Err)
		}
	}

	// Check list result
	var list ListOutput
	if err := json.Unmarshal(results[2].Result, &list); err != nil {
		t.Fatalf("Unmarshal list result: %v", err)
	}
	// Note: both creates add to the same ID "todo-1", so total is 1
	if list.Total != 1 {
		t.Errorf("Expected 1 todo, got %d", list.Total)
	}
}

// --- Response ID Type Tests ---

func TestStringID(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	jsonrpc.Mount(r, "/rpc", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, _ := doRPC(ts, rpcRequest{
		JSONRPC: "2.0",
		Method:  "todos.list",
		ID:      "string-id-123",
	})
	if resp.Error != nil {
		t.Fatalf("Error: %+v", resp.Error)
	}
	if resp.ID != "string-id-123" {
		t.Errorf("Expected ID 'string-id-123', got %v", resp.ID)
	}
}

func TestNullParams(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	jsonrpc.Mount(r, "/rpc", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Explicit null params for method without input
	body := []byte(`{"jsonrpc":"2.0","method":"todos.list","params":null,"id":1}`)
	resp, err := http.Post(ts.URL+"/rpc", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	var rpcResp rpcResponse
	json.NewDecoder(resp.Body).Decode(&rpcResp)
	if rpcResp.Error != nil {
		t.Errorf("Unexpected error: %+v", rpcResp.Error)
	}
}
