package async_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-mizu/mizu"
	contract "github.com/go-mizu/mizu/contract/v2"
	"github.com/go-mizu/mizu/contract/v2/transport/async"
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
	mu    sync.RWMutex
	todos map[string]*Todo
	err   error // inject error for testing
	delay time.Duration
}

func newTodoService() *todoService {
	return &todoService{
		todos: make(map[string]*Todo),
	}
}

func (s *todoService) Create(ctx context.Context, in *CreateInput) (*Todo, error) {
	if s.delay > 0 {
		time.Sleep(s.delay)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
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
	if s.delay > 0 {
		time.Sleep(s.delay)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
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
	if s.delay > 0 {
		time.Sleep(s.delay)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
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
	if s.delay > 0 {
		time.Sleep(s.delay)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
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
	if s.delay > 0 {
		time.Sleep(s.delay)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
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
		contract.WithName("TodoService"),
	)
}

type submitRequest struct {
	ID     string `json:"id"`
	Method string `json:"method"`
	Params any    `json:"params,omitempty"`
}

type submitResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type sseEvent struct {
	Type string
	Data json.RawMessage
}

type asyncResponse struct {
	ID     string          `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *asyncError     `json:"error,omitempty"`
}

type asyncError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func doSubmit(ts *httptest.Server, req submitRequest) (*submitResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(ts.URL+"/async", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, errors.New("submit failed: " + string(bodyBytes))
	}

	var submitResp submitResponse
	if err := json.NewDecoder(resp.Body).Decode(&submitResp); err != nil {
		return nil, err
	}
	return &submitResp, nil
}

func readSSEEvent(scanner *bufio.Scanner) (*sseEvent, error) {
	var event sseEvent
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			// End of event
			if event.Type != "" || len(event.Data) > 0 {
				return &event, nil
			}
			continue
		}
		if strings.HasPrefix(line, "event: ") {
			event.Type = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			event.Data = json.RawMessage(strings.TrimPrefix(line, "data: "))
		} else if strings.HasPrefix(line, ": ") {
			// Comment (ping), skip
			continue
		}
	}
	if scanner.Err() != nil {
		return nil, scanner.Err()
	}
	return nil, io.EOF
}

// --- Handler Tests ---

func TestHandler(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	handler, err := async.Handler(inv)
	if err != nil {
		t.Fatalf("Handler failed: %v", err)
	}
	if handler == nil {
		t.Fatal("Handler returned nil")
	}
}

func TestHandlerNilInvoker(t *testing.T) {
	_, err := async.Handler(nil)
	if err == nil {
		t.Fatal("Expected error for nil invoker")
	}
}

// --- Mount Tests ---

func TestMount(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	if err := async.Mount(r, "/async", inv); err != nil {
		t.Fatalf("Mount failed: %v", err)
	}

	ts := httptest.NewServer(r)
	defer ts.Close()

	// Test submit
	resp, err := doSubmit(ts, submitRequest{
		ID:     "test-1",
		Method: "todos.create",
		Params: map[string]any{"title": "Test Todo"},
	})
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}
	if resp.ID != "test-1" {
		t.Errorf("Expected ID 'test-1', got '%s'", resp.ID)
	}
	if resp.Status != "accepted" {
		t.Errorf("Expected status 'accepted', got '%s'", resp.Status)
	}
}

func TestMountEmptyPath(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	if err := async.Mount(r, "", inv); err != nil {
		t.Fatalf("Mount failed: %v", err)
	}

	ts := httptest.NewServer(r)
	defer ts.Close()

	body, _ := json.Marshal(submitRequest{
		ID:     "test-1",
		Method: "todos.list",
	})

	resp, err := http.Post(ts.URL+"/", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("Expected 202, got %d", resp.StatusCode)
	}
}

func TestMountNilRouter(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	if err := async.Mount(nil, "/async", inv); err == nil {
		t.Fatal("Expected error for nil router")
	}
}

func TestMountNilInvoker(t *testing.T) {
	r := mizu.NewRouter()
	if err := async.Mount(r, "/async", nil); err == nil {
		t.Fatal("Expected error for nil invoker")
	}
}

// --- Submit Tests ---

func TestSubmit(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	async.Mount(r, "/async", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, err := doSubmit(ts, submitRequest{
		ID:     "req-123",
		Method: "todos.create",
		Params: map[string]any{"title": "My Todo", "body": "Description"},
	})
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}
	if resp.ID != "req-123" {
		t.Errorf("Expected ID 'req-123', got '%s'", resp.ID)
	}
	if resp.Status != "accepted" {
		t.Errorf("Expected status 'accepted', got '%s'", resp.Status)
	}
}

func TestSubmitMissingID(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	async.Mount(r, "/async", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	body, _ := json.Marshal(submitRequest{
		Method: "todos.create",
	})

	resp, err := http.Post(ts.URL+"/async", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestSubmitMissingMethod(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	async.Mount(r, "/async", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	body, _ := json.Marshal(submitRequest{
		ID: "test-1",
	})

	resp, err := http.Post(ts.URL+"/async", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestSubmitInvalidMethod(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	async.Mount(r, "/async", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Method without dot
	body, _ := json.Marshal(submitRequest{
		ID:     "test-1",
		Method: "invalidmethod",
	})

	resp, err := http.Post(ts.URL+"/async", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestSubmitUnknownMethod(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	async.Mount(r, "/async", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	body, _ := json.Marshal(submitRequest{
		ID:     "test-1",
		Method: "unknown.method",
	})

	resp, err := http.Post(ts.URL+"/async", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestSubmitInvalidJSON(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	async.Mount(r, "/async", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/async", "application/json", strings.NewReader("{invalid json"))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

// --- SSE Stream Tests ---

func TestStream(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	async.Mount(r, "/async", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Connect to SSE stream
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", ts.URL+"/async", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/event-stream") {
		t.Errorf("Expected Content-Type text/event-stream, got %s", contentType)
	}
}

// --- Full Round-Trip Tests ---

func TestSubmitAndReceive(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	async.Mount(r, "/async", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Connect to SSE stream first
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", ts.URL+"/async", nil)
	streamResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer streamResp.Body.Close()

	// Give the stream connection a moment to establish
	time.Sleep(50 * time.Millisecond)

	// Submit request
	_, err = doSubmit(ts, submitRequest{
		ID:     "roundtrip-1",
		Method: "todos.create",
		Params: map[string]any{"title": "Round Trip Test"},
	})
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}

	// Read SSE event
	scanner := bufio.NewScanner(streamResp.Body)
	event, err := readSSEEvent(scanner)
	if err != nil {
		t.Fatalf("Failed to read SSE event: %v", err)
	}

	if event.Type != "result" {
		t.Errorf("Expected event type 'result', got '%s'", event.Type)
	}

	var response asyncResponse
	if err := json.Unmarshal(event.Data, &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.ID != "roundtrip-1" {
		t.Errorf("Expected ID 'roundtrip-1', got '%s'", response.ID)
	}

	if response.Error != nil {
		t.Errorf("Unexpected error: %+v", response.Error)
	}

	var todo Todo
	if err := json.Unmarshal(response.Result, &todo); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	if todo.Title != "Round Trip Test" {
		t.Errorf("Expected title 'Round Trip Test', got '%s'", todo.Title)
	}
}

func TestSubmitAndReceiveError(t *testing.T) {
	svc := newTodoService()
	svc.err = errors.New("service error")
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	async.Mount(r, "/async", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Connect to SSE stream
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", ts.URL+"/async", nil)
	streamResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer streamResp.Body.Close()

	time.Sleep(50 * time.Millisecond)

	// Submit request that will fail
	_, err = doSubmit(ts, submitRequest{
		ID:     "error-1",
		Method: "todos.create",
		Params: map[string]any{"title": "Will Fail"},
	})
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}

	// Read SSE event
	scanner := bufio.NewScanner(streamResp.Body)
	event, err := readSSEEvent(scanner)
	if err != nil {
		t.Fatalf("Failed to read SSE event: %v", err)
	}

	if event.Type != "error" {
		t.Errorf("Expected event type 'error', got '%s'", event.Type)
	}

	var response asyncResponse
	if err := json.Unmarshal(event.Data, &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.ID != "error-1" {
		t.Errorf("Expected ID 'error-1', got '%s'", response.ID)
	}

	if response.Error == nil {
		t.Fatal("Expected error in response")
	}

	if !strings.Contains(response.Error.Message, "service error") {
		t.Errorf("Expected error message to contain 'service error', got '%s'", response.Error.Message)
	}
}

// --- Option Tests ---

func TestErrorMapper(t *testing.T) {
	svc := newTodoService()
	svc.err = errNotFound
	inv := newTestInvoker(svc)

	customMapper := func(err error) (string, string) {
		if errors.Is(err, errNotFound) {
			return "not_found", "resource not found"
		}
		return "error", err.Error()
	}

	r := mizu.NewRouter()
	async.Mount(r, "/async", inv, async.WithErrorMapper(customMapper))
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Connect to SSE stream
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", ts.URL+"/async", nil)
	streamResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer streamResp.Body.Close()

	time.Sleep(50 * time.Millisecond)

	_, err = doSubmit(ts, submitRequest{
		ID:     "mapped-error-1",
		Method: "todos.list",
	})
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}

	scanner := bufio.NewScanner(streamResp.Body)
	event, err := readSSEEvent(scanner)
	if err != nil {
		t.Fatalf("Failed to read SSE event: %v", err)
	}

	var response asyncResponse
	json.Unmarshal(event.Data, &response)

	if response.Error == nil {
		t.Fatal("Expected error")
	}
	if response.Error.Code != "not_found" {
		t.Errorf("Expected code 'not_found', got '%s'", response.Error.Code)
	}
	if response.Error.Message != "resource not found" {
		t.Errorf("Expected message 'resource not found', got '%s'", response.Error.Message)
	}
}

func TestMaxBodySize(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	async.Mount(r, "/async", inv, async.WithMaxBodySize(50))
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Create a large request
	largeTitle := strings.Repeat("x", 100)
	body, _ := json.Marshal(submitRequest{
		ID:     "large-1",
		Method: "todos.create",
		Params: map[string]any{"title": largeTitle},
	})

	resp, err := http.Post(ts.URL+"/async", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Errorf("Expected 413, got %d", resp.StatusCode)
	}
}

func TestConnectDisconnectCallbacks(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	var (
		mu            sync.Mutex
		connectedIDs  []string
		disconnectIDs []string
	)

	onConnect := func(id string) {
		mu.Lock()
		connectedIDs = append(connectedIDs, id)
		mu.Unlock()
	}

	onDisconnect := func(id string) {
		mu.Lock()
		disconnectIDs = append(disconnectIDs, id)
		mu.Unlock()
	}

	r := mizu.NewRouter()
	async.Mount(r, "/async", inv,
		async.WithOnConnect(onConnect),
		async.WithOnDisconnect(onDisconnect),
	)
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Connect to SSE stream
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", ts.URL+"/async", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}

	// Wait a bit for connection
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	if len(connectedIDs) != 1 {
		t.Errorf("Expected 1 connection, got %d", len(connectedIDs))
	}
	mu.Unlock()

	// Close connection
	resp.Body.Close()
	cancel()

	// Wait for disconnect
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	if len(disconnectIDs) != 1 {
		t.Errorf("Expected 1 disconnect, got %d", len(disconnectIDs))
	}
	mu.Unlock()
}

// --- HTTP Method Tests ---

func TestMethodNotAllowed(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	async.Mount(r, "/async", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Try PUT
	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/async", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PUT failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected 405, got %d", resp.StatusCode)
	}
}

// --- No Input/Output Tests ---

func TestNoInput(t *testing.T) {
	svc := newTodoService()
	// Pre-populate
	svc.todos["1"] = &Todo{ID: "1", Title: "One"}
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	async.Mount(r, "/async", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Connect to SSE
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", ts.URL+"/async", nil)
	streamResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer streamResp.Body.Close()

	time.Sleep(50 * time.Millisecond)

	// List has no input
	_, err = doSubmit(ts, submitRequest{
		ID:     "no-input-1",
		Method: "todos.list",
	})
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}

	scanner := bufio.NewScanner(streamResp.Body)
	event, err := readSSEEvent(scanner)
	if err != nil {
		t.Fatalf("Failed to read SSE event: %v", err)
	}

	if event.Type != "result" {
		t.Errorf("Expected 'result', got '%s'", event.Type)
	}

	var response asyncResponse
	json.Unmarshal(event.Data, &response)

	if response.Error != nil {
		t.Errorf("Unexpected error: %+v", response.Error)
	}

	var list ListOutput
	json.Unmarshal(response.Result, &list)
	if list.Total != 1 {
		t.Errorf("Expected 1 item, got %d", list.Total)
	}
}

func TestNoOutput(t *testing.T) {
	svc := newTodoService()
	svc.todos["del-1"] = &Todo{ID: "del-1", Title: "Delete Me"}
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	async.Mount(r, "/async", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Connect to SSE
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", ts.URL+"/async", nil)
	streamResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer streamResp.Body.Close()

	time.Sleep(50 * time.Millisecond)

	// Delete has no output
	_, err = doSubmit(ts, submitRequest{
		ID:     "no-output-1",
		Method: "todos.delete",
		Params: map[string]any{"id": "del-1"},
	})
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}

	scanner := bufio.NewScanner(streamResp.Body)
	event, err := readSSEEvent(scanner)
	if err != nil {
		t.Fatalf("Failed to read SSE event: %v", err)
	}

	var response asyncResponse
	json.Unmarshal(event.Data, &response)

	if response.Error != nil {
		t.Errorf("Unexpected error: %+v", response.Error)
	}

	// Result should be null
	if string(response.Result) != "null" {
		t.Errorf("Expected null result, got %s", response.Result)
	}
}

// --- Multiple Clients Tests ---

func TestMultipleClients(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	async.Mount(r, "/async", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Connect two SSE clients
	ctx1, cancel1 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel1()
	req1, _ := http.NewRequestWithContext(ctx1, "GET", ts.URL+"/async", nil)
	streamResp1, _ := http.DefaultClient.Do(req1)
	defer streamResp1.Body.Close()

	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()
	req2, _ := http.NewRequestWithContext(ctx2, "GET", ts.URL+"/async", nil)
	streamResp2, _ := http.DefaultClient.Do(req2)
	defer streamResp2.Body.Close()

	time.Sleep(50 * time.Millisecond)

	// Submit request
	_, err := doSubmit(ts, submitRequest{
		ID:     "multi-client-1",
		Method: "todos.create",
		Params: map[string]any{"title": "Multi Client Test"},
	})
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}

	// Both clients should receive the event
	scanner1 := bufio.NewScanner(streamResp1.Body)
	scanner2 := bufio.NewScanner(streamResp2.Body)

	event1, err := readSSEEvent(scanner1)
	if err != nil {
		t.Fatalf("Client 1 failed to read event: %v", err)
	}

	event2, err := readSSEEvent(scanner2)
	if err != nil {
		t.Fatalf("Client 2 failed to read event: %v", err)
	}

	// Both should receive the same event
	var resp1, resp2 asyncResponse
	json.Unmarshal(event1.Data, &resp1)
	json.Unmarshal(event2.Data, &resp2)

	if resp1.ID != "multi-client-1" || resp2.ID != "multi-client-1" {
		t.Error("Both clients should receive the same event")
	}
}

// --- Concurrent Requests Test ---

func TestConcurrentRequests(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	async.Mount(r, "/async", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Connect to SSE
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", ts.URL+"/async", nil)
	streamResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer streamResp.Body.Close()

	time.Sleep(50 * time.Millisecond)

	// Submit multiple requests concurrently
	numRequests := 5
	var wg sync.WaitGroup
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			doSubmit(ts, submitRequest{
				ID:     "concurrent-" + string(rune('0'+n)),
				Method: "todos.list",
			})
		}(i)
	}
	wg.Wait()

	// Read all responses
	receivedIDs := make(map[string]bool)
	scanner := bufio.NewScanner(streamResp.Body)

	for i := 0; i < numRequests; i++ {
		event, err := readSSEEvent(scanner)
		if err != nil {
			break
		}
		var response asyncResponse
		json.Unmarshal(event.Data, &response)
		receivedIDs[response.ID] = true
	}

	if len(receivedIDs) != numRequests {
		t.Errorf("Expected %d responses, got %d", numRequests, len(receivedIDs))
	}
}

// --- AsyncAPI Tests ---

func TestAsyncAPI(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	spec, err := async.AsyncAPI(inv.Descriptor())
	if err != nil {
		t.Fatalf("AsyncAPI failed: %v", err)
	}

	if len(spec) == 0 {
		t.Fatal("Expected non-empty AsyncAPI spec")
	}

	// Parse and verify basic structure
	var doc map[string]any
	if err := json.Unmarshal(spec, &doc); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	if doc["asyncapi"] != "2.6.0" {
		t.Errorf("Expected asyncapi 2.6.0, got %v", doc["asyncapi"])
	}

	if doc["channels"] == nil {
		t.Error("Expected channels in AsyncAPI doc")
	}
}

// --- Integration Test ---

func TestTodoAPI(t *testing.T) {
	svc := newTodoService()
	inv := newTestInvoker(svc)

	r := mizu.NewRouter()
	async.Mount(r, "/async", inv)
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Connect to SSE
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", ts.URL+"/async", nil)
	streamResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer streamResp.Body.Close()

	scanner := bufio.NewScanner(streamResp.Body)
	time.Sleep(50 * time.Millisecond)

	// 1. Create
	doSubmit(ts, submitRequest{
		ID:     "api-create",
		Method: "todos.create",
		Params: map[string]any{"title": "API Test", "body": "Body"},
	})

	event, _ := readSSEEvent(scanner)
	var createResp asyncResponse
	json.Unmarshal(event.Data, &createResp)
	if createResp.Error != nil {
		t.Fatalf("Create failed: %+v", createResp.Error)
	}

	var created Todo
	json.Unmarshal(createResp.Result, &created)
	if created.Title != "API Test" {
		t.Errorf("Expected title 'API Test', got '%s'", created.Title)
	}

	// 2. List
	doSubmit(ts, submitRequest{
		ID:     "api-list",
		Method: "todos.list",
	})

	event, _ = readSSEEvent(scanner)
	var listResp asyncResponse
	json.Unmarshal(event.Data, &listResp)
	if listResp.Error != nil {
		t.Fatalf("List failed: %+v", listResp.Error)
	}

	var list ListOutput
	json.Unmarshal(listResp.Result, &list)
	if list.Total != 1 {
		t.Errorf("Expected 1 todo, got %d", list.Total)
	}

	// 3. Get
	doSubmit(ts, submitRequest{
		ID:     "api-get",
		Method: "todos.get",
		Params: map[string]any{"id": created.ID},
	})

	event, _ = readSSEEvent(scanner)
	var getResp asyncResponse
	json.Unmarshal(event.Data, &getResp)
	if getResp.Error != nil {
		t.Fatalf("Get failed: %+v", getResp.Error)
	}

	// 4. Update
	doSubmit(ts, submitRequest{
		ID:     "api-update",
		Method: "todos.update",
		Params: map[string]any{"id": created.ID, "title": "Updated"},
	})

	event, _ = readSSEEvent(scanner)
	var updateResp asyncResponse
	json.Unmarshal(event.Data, &updateResp)
	if updateResp.Error != nil {
		t.Fatalf("Update failed: %+v", updateResp.Error)
	}

	var updated Todo
	json.Unmarshal(updateResp.Result, &updated)
	if updated.Title != "Updated" {
		t.Errorf("Expected title 'Updated', got '%s'", updated.Title)
	}

	// 5. Delete
	doSubmit(ts, submitRequest{
		ID:     "api-delete",
		Method: "todos.delete",
		Params: map[string]any{"id": created.ID},
	})

	event, _ = readSSEEvent(scanner)
	var deleteResp asyncResponse
	json.Unmarshal(event.Data, &deleteResp)
	if deleteResp.Error != nil {
		t.Fatalf("Delete failed: %+v", deleteResp.Error)
	}
}
