package mcp_test

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
	"github.com/go-mizu/mizu/contract/v2/transport/mcp"
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

func doMCP(ts *httptest.Server, req rpcRequest) (*rpcResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(ts.URL+"/mcp", "application/json", bytes.NewReader(body))
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

// --- Handler Tests ---

func TestHandler(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	handler, err := mcp.Handler(inv)
	if err != nil {
		t.Fatalf("Handler failed: %v", err)
	}
	if handler == nil {
		t.Fatal("Handler returned nil")
	}
}

func TestHandlerNilInvoker(t *testing.T) {
	_, err := mcp.Handler(nil)
	if err == nil {
		t.Fatal("Expected error for nil invoker")
	}
}

// --- Mount Tests ---

func TestMount(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	if err := mcp.Mount(r, "/mcp", inv); err != nil {
		t.Fatalf("Mount failed: %v", err)
	}

	ts := httptest.NewServer(r)
	defer ts.Close()

	// Test initialize
	resp, err := doMCP(ts, rpcRequest{
		JSONRPC: "2.0",
		Method:  "initialize",
		Params: map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "test"},
		},
		ID: 1,
	})
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("Initialize error: %+v", resp.Error)
	}
}

func TestMountEmptyPath(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	if err := mcp.Mount(r, "", inv); err != nil {
		t.Fatalf("Mount failed: %v", err)
	}

	ts := httptest.NewServer(r)
	defer ts.Close()

	body, _ := json.Marshal(rpcRequest{
		JSONRPC: "2.0",
		Method:  "tools/list",
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

	if err := mcp.Mount(nil, "/mcp", inv); err == nil {
		t.Fatal("Expected error for nil router")
	}
}

func TestMountNilInvoker(t *testing.T) {
	r := mizu.NewRouter()
	if err := mcp.Mount(r, "/mcp", nil); err == nil {
		t.Fatal("Expected error for nil invoker")
	}
}

// --- Initialize Tests ---

func TestInitialize(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	mcp.Mount(r, "/mcp", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, err := doMCP(ts, rpcRequest{
		JSONRPC: "2.0",
		Method:  "initialize",
		Params: map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
		},
		ID: 1,
	})
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("Initialize error: %+v", resp.Error)
	}

	var result map[string]any
	json.Unmarshal(resp.Result, &result)

	if result["protocolVersion"] != "2024-11-05" {
		t.Errorf("Expected protocolVersion '2024-11-05', got %v", result["protocolVersion"])
	}

	caps, ok := result["capabilities"].(map[string]any)
	if !ok {
		t.Fatal("Expected capabilities object")
	}
	if _, ok := caps["tools"]; !ok {
		t.Error("Expected tools capability")
	}
}

func TestInitializeWithServerInfo(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	mcp.Mount(r, "/mcp", inv, mcp.WithServerInfo("test-server", "1.0.0"))
	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, _ := doMCP(ts, rpcRequest{
		JSONRPC: "2.0",
		Method:  "initialize",
		Params:  map[string]any{"protocolVersion": "2024-11-05"},
		ID:      1,
	})

	var result map[string]any
	json.Unmarshal(resp.Result, &result)

	info, ok := result["serverInfo"].(map[string]any)
	if !ok {
		t.Fatal("Expected serverInfo")
	}
	if info["name"] != "test-server" {
		t.Errorf("Expected name 'test-server', got %v", info["name"])
	}
	if info["version"] != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got %v", info["version"])
	}
}

func TestInitializedNotification(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	mcp.Mount(r, "/mcp", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, err := doMCP(ts, rpcRequest{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
		// No ID - it's a notification
	})
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if resp != nil {
		t.Error("Expected no response for notification")
	}
}

// --- Tools List Tests ---

func TestToolsList(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	mcp.Mount(r, "/mcp", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, err := doMCP(ts, rpcRequest{
		JSONRPC: "2.0",
		Method:  "tools/list",
		ID:      1,
	})
	if err != nil {
		t.Fatalf("tools/list failed: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("tools/list error: %+v", resp.Error)
	}

	var result struct {
		Tools []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			InputSchema struct {
				Type       string         `json:"type"`
				Properties map[string]any `json:"properties"`
				Required   []string       `json:"required"`
			} `json:"inputSchema"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(result.Tools) != 5 {
		t.Errorf("Expected 5 tools, got %d", len(result.Tools))
	}

	// Check tool names
	names := make(map[string]bool)
	for _, tool := range result.Tools {
		names[tool.Name] = true
	}
	expected := []string{"todos_create", "todos_list", "todos_get", "todos_update", "todos_delete"}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("Missing tool: %s", name)
		}
	}
}

func TestToolsListInputSchema(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	mcp.Mount(r, "/mcp", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, _ := doMCP(ts, rpcRequest{
		JSONRPC: "2.0",
		Method:  "tools/list",
		ID:      1,
	})

	var result struct {
		Tools []struct {
			Name        string `json:"name"`
			InputSchema struct {
				Type       string                    `json:"type"`
				Properties map[string]map[string]any `json:"properties"`
				Required   []string                  `json:"required"`
			} `json:"inputSchema"`
		} `json:"tools"`
	}
	json.Unmarshal(resp.Result, &result)

	// Find todos_create tool
	var createTool *struct {
		Name        string `json:"name"`
		InputSchema struct {
			Type       string                    `json:"type"`
			Properties map[string]map[string]any `json:"properties"`
			Required   []string                  `json:"required"`
		} `json:"inputSchema"`
	}
	for i := range result.Tools {
		if result.Tools[i].Name == "todos_create" {
			createTool = &result.Tools[i]
			break
		}
	}
	if createTool == nil {
		t.Fatal("todos_create tool not found")
	}

	// Check schema
	if createTool.InputSchema.Type != "object" {
		t.Errorf("Expected type 'object', got '%s'", createTool.InputSchema.Type)
	}

	// Check properties
	if _, ok := createTool.InputSchema.Properties["title"]; !ok {
		t.Error("Expected 'title' property")
	}
	if _, ok := createTool.InputSchema.Properties["body"]; !ok {
		t.Error("Expected 'body' property")
	}

	// Check required fields
	hasTitle := false
	for _, r := range createTool.InputSchema.Required {
		if r == "title" {
			hasTitle = true
		}
	}
	if !hasTitle {
		t.Error("Expected 'title' to be required")
	}
}

// --- Tools Call Tests ---

func TestToolsCall(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	mcp.Mount(r, "/mcp", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, err := doMCP(ts, rpcRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params: map[string]any{
			"name":      "todos_create",
			"arguments": map[string]any{"title": "Test Todo", "body": "Test Body"},
		},
		ID: 1,
	})
	if err != nil {
		t.Fatalf("tools/call failed: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("tools/call error: %+v", resp.Error)
	}

	var result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	json.Unmarshal(resp.Result, &result)

	if result.IsError {
		t.Error("Expected isError=false")
	}
	if len(result.Content) != 1 {
		t.Fatalf("Expected 1 content item, got %d", len(result.Content))
	}
	if result.Content[0].Type != "text" {
		t.Errorf("Expected type 'text', got '%s'", result.Content[0].Type)
	}

	// Parse the text content as Todo
	var todo Todo
	if err := json.Unmarshal([]byte(result.Content[0].Text), &todo); err != nil {
		t.Fatalf("Failed to parse content: %v", err)
	}
	if todo.Title != "Test Todo" {
		t.Errorf("Expected title 'Test Todo', got '%s'", todo.Title)
	}
}

func TestToolsCallError(t *testing.T) {
	svc := newTodoService()
	svc.err = errors.New("service error")
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	mcp.Mount(r, "/mcp", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, _ := doMCP(ts, rpcRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params: map[string]any{
			"name": "todos_list",
		},
		ID: 1,
	})

	var result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	json.Unmarshal(resp.Result, &result)

	if !result.IsError {
		t.Error("Expected isError=true")
	}
	if len(result.Content) == 0 {
		t.Fatal("Expected content")
	}
	if !strings.Contains(result.Content[0].Text, "service error") {
		t.Errorf("Expected error message, got '%s'", result.Content[0].Text)
	}
}

func TestToolsCallUnknownTool(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	mcp.Mount(r, "/mcp", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, _ := doMCP(ts, rpcRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params: map[string]any{
			"name": "unknown_tool",
		},
		ID: 1,
	})

	if resp.Error == nil {
		t.Fatal("Expected error for unknown tool")
	}
	if resp.Error.Code != -32602 {
		t.Errorf("Expected code -32602, got %d", resp.Error.Code)
	}
}

func TestToolsCallInvalidToolName(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	mcp.Mount(r, "/mcp", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Tool name without underscore
	resp, _ := doMCP(ts, rpcRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params: map[string]any{
			"name": "invalidname",
		},
		ID: 1,
	})

	if resp.Error == nil {
		t.Fatal("Expected error for invalid tool name")
	}
}

// --- Error Tests ---

func TestParseError(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	mcp.Mount(r, "/mcp", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/mcp", "application/json", strings.NewReader("{invalid json"))
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
	mcp.Mount(r, "/mcp", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Missing jsonrpc field
	resp, _ := doMCP(ts, rpcRequest{
		Method: "tools/list",
		ID:     1,
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
	mcp.Mount(r, "/mcp", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, _ := doMCP(ts, rpcRequest{
		JSONRPC: "2.0",
		Method:  "unknown/method",
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
	mcp.Mount(r, "/mcp", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Invalid params for tools/call
	body := []byte(`{"jsonrpc":"2.0","method":"tools/call","params":"invalid","id":1}`)
	resp, err := http.Post(ts.URL+"/mcp", "application/json", bytes.NewReader(body))
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

// --- Option Tests ---

func TestErrorMapper(t *testing.T) {
	svc := newTodoService()
	svc.err = errNotFound
	inv := newTestInvoker(svc)

	customMapper := func(err error) (bool, string) {
		if errors.Is(err, errNotFound) {
			return true, "resource not found"
		}
		return true, err.Error()
	}

	r := mizu.NewRouter()
	mcp.Mount(r, "/mcp", inv, mcp.WithErrorMapper(customMapper))
	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, _ := doMCP(ts, rpcRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  map[string]any{"name": "todos_list"},
		ID:      1,
	})

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	json.Unmarshal(resp.Result, &result)

	if !result.IsError {
		t.Error("Expected isError=true")
	}
	if result.Content[0].Text != "resource not found" {
		t.Errorf("Expected 'resource not found', got '%s'", result.Content[0].Text)
	}
}

func TestMaxBodySize(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	mcp.Mount(r, "/mcp", inv, mcp.WithMaxBodySize(100))
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Create a large request
	largeBody := strings.Repeat("x", 200)
	body, _ := json.Marshal(rpcRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  map[string]any{"name": "todos_create", "arguments": map[string]any{"title": largeBody}},
		ID:      1,
	})

	resp, err := http.Post(ts.URL+"/mcp", "application/json", bytes.NewReader(body))
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
	mcp.Mount(r, "/mcp", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/mcp")
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
	mcp.Mount(r, "/mcp", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/mcp", "application/json", strings.NewReader(""))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	var rpcResp rpcResponse
	json.NewDecoder(resp.Body).Decode(&rpcResp)
	if rpcResp.Error == nil {
		t.Fatal("Expected error for empty body")
	}
}

// --- No Input/Output Tests ---

func TestNoInput(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	mcp.Mount(r, "/mcp", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	// List has no input
	resp, _ := doMCP(ts, rpcRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  map[string]any{"name": "todos_list"},
		ID:      1,
	})
	if resp.Error != nil {
		t.Fatalf("tools/call error: %+v", resp.Error)
	}
}

func TestNoOutput(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	mcp.Mount(r, "/mcp", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	// First create a todo
	doMCP(ts, rpcRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params: map[string]any{
			"name":      "todos_create",
			"arguments": map[string]any{"title": "Test"},
		},
		ID: 1,
	})

	// Delete has no output
	resp, _ := doMCP(ts, rpcRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params: map[string]any{
			"name":      "todos_delete",
			"arguments": map[string]any{"id": "todo-1"},
		},
		ID: 2,
	})
	if resp.Error != nil {
		t.Fatalf("Delete error: %+v", resp.Error)
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	json.Unmarshal(resp.Result, &result)

	// Result should be null
	if result.Content[0].Text != "null" {
		t.Errorf("Expected null result, got %s", result.Content[0].Text)
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
	api := r.Prefix("/mcp").With(middleware)
	mcp.Mount(api, "", inv)

	ts := httptest.NewServer(r)
	defer ts.Close()

	doMCP(ts, rpcRequest{
		JSONRPC: "2.0",
		Method:  "tools/list",
		ID:      1,
	})

	if !middlewareCalled {
		t.Error("Middleware was not called")
	}
}

// --- Integration Test ---

func TestTodoAPI(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	mcp.Mount(r, "/mcp", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Initialize
	doMCP(ts, rpcRequest{
		JSONRPC: "2.0",
		Method:  "initialize",
		Params:  map[string]any{"protocolVersion": "2024-11-05"},
		ID:      1,
	})

	// Create
	createResp, _ := doMCP(ts, rpcRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params: map[string]any{
			"name":      "todos_create",
			"arguments": map[string]any{"title": "Buy milk", "body": "From the store"},
		},
		ID: 2,
	})
	if createResp.Error != nil {
		t.Fatalf("Create error: %+v", createResp.Error)
	}

	var createResult struct {
		Content []struct{ Text string } `json:"content"`
	}
	json.Unmarshal(createResp.Result, &createResult)
	var created Todo
	json.Unmarshal([]byte(createResult.Content[0].Text), &created)

	// List
	listResp, _ := doMCP(ts, rpcRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  map[string]any{"name": "todos_list"},
		ID:      3,
	})
	if listResp.Error != nil {
		t.Fatalf("List error: %+v", listResp.Error)
	}

	var listResult struct {
		Content []struct{ Text string } `json:"content"`
	}
	json.Unmarshal(listResp.Result, &listResult)
	var list ListOutput
	json.Unmarshal([]byte(listResult.Content[0].Text), &list)
	if list.Total != 1 {
		t.Errorf("Expected 1 todo, got %d", list.Total)
	}

	// Get
	getResp, _ := doMCP(ts, rpcRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params: map[string]any{
			"name":      "todos_get",
			"arguments": map[string]any{"id": created.ID},
		},
		ID: 4,
	})
	if getResp.Error != nil {
		t.Fatalf("Get error: %+v", getResp.Error)
	}

	// Update
	updateResp, _ := doMCP(ts, rpcRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params: map[string]any{
			"name":      "todos_update",
			"arguments": map[string]any{"id": created.ID, "title": "Buy eggs"},
		},
		ID: 5,
	})
	if updateResp.Error != nil {
		t.Fatalf("Update error: %+v", updateResp.Error)
	}

	var updateResult struct {
		Content []struct{ Text string } `json:"content"`
	}
	json.Unmarshal(updateResp.Result, &updateResult)
	var updated Todo
	json.Unmarshal([]byte(updateResult.Content[0].Text), &updated)
	if updated.Title != "Buy eggs" {
		t.Errorf("Expected title 'Buy eggs', got '%s'", updated.Title)
	}

	// Delete
	deleteResp, _ := doMCP(ts, rpcRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params: map[string]any{
			"name":      "todos_delete",
			"arguments": map[string]any{"id": created.ID},
		},
		ID: 6,
	})
	if deleteResp.Error != nil {
		t.Fatalf("Delete error: %+v", deleteResp.Error)
	}

	// Verify deleted
	listResp, _ = doMCP(ts, rpcRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  map[string]any{"name": "todos_list"},
		ID:      7,
	})
	json.Unmarshal(listResp.Result, &listResult)
	json.Unmarshal([]byte(listResult.Content[0].Text), &list)
	if list.Total != 0 {
		t.Errorf("Expected 0 todos after delete, got %d", list.Total)
	}
}

// --- Tool Naming Tests ---

func TestToolNaming(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	mcp.Mount(r, "/mcp", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, _ := doMCP(ts, rpcRequest{
		JSONRPC: "2.0",
		Method:  "tools/list",
		ID:      1,
	})

	var result struct {
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}
	json.Unmarshal(resp.Result, &result)

	// All tool names should be in format resource_method
	for _, tool := range result.Tools {
		if !strings.Contains(tool.Name, "_") {
			t.Errorf("Tool name '%s' should contain underscore", tool.Name)
		}
		parts := strings.SplitN(tool.Name, "_", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			t.Errorf("Tool name '%s' has invalid format", tool.Name)
		}
	}
}

// --- Response ID Type Tests ---

func TestStringID(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	mcp.Mount(r, "/mcp", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, _ := doMCP(ts, rpcRequest{
		JSONRPC: "2.0",
		Method:  "tools/list",
		ID:      "string-id-123",
	})
	if resp.Error != nil {
		t.Fatalf("Error: %+v", resp.Error)
	}
	if resp.ID != "string-id-123" {
		t.Errorf("Expected ID 'string-id-123', got %v", resp.ID)
	}
}

func TestNullArguments(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	mcp.Mount(r, "/mcp", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Explicit null arguments for tool without input
	body := []byte(`{"jsonrpc":"2.0","method":"tools/call","params":{"name":"todos_list","arguments":null},"id":1}`)
	resp, err := http.Post(ts.URL+"/mcp", "application/json", bytes.NewReader(body))
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

// --- Unknown Notification Test ---

func TestUnknownNotification(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	mcp.Mount(r, "/mcp", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Unknown method as notification (no ID) should be silently ignored
	resp, err := doMCP(ts, rpcRequest{
		JSONRPC: "2.0",
		Method:  "unknown/notification",
		// No ID - it's a notification
	})
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if resp != nil {
		t.Error("Expected no response for unknown notification")
	}
}
