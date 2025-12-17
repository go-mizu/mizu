package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-mizu/mizu/contract/v1"
)

// TestService is a simple service for testing.
type TestService struct{}

type CreateInput struct {
	Title string `json:"title"`
}

type Todo struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

func (s *TestService) Create(ctx context.Context, in *CreateInput) (*Todo, error) {
	return &Todo{ID: "1", Title: in.Title}, nil
}

func (s *TestService) List(ctx context.Context) (*Todo, error) {
	return &Todo{ID: "1", Title: "test"}, nil
}

func (s *TestService) Health(ctx context.Context) error {
	return nil
}

func setupHandler(t *testing.T) *Handler {
	svc, err := contract.Register("test", &TestService{})
	if err != nil {
		t.Fatalf("failed to register service: %v", err)
	}
	return NewHandler(svc)
}

func TestHandler_Initialize(t *testing.T) {
	h := setupHandler(t)

	tests := []struct {
		name            string
		protocolVersion string
		wantSuccess     bool
	}{
		{"latest version", ProtocolLatest, true},
		{"fallback version", ProtocolFallback, true},
		{"legacy version", ProtocolLegacy, true},
		{"empty version", "", true},
		{"unsupported version", "1999-01-01", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := rpcRequest{
				JSONRPC: "2.0",
				ID:      json.RawMessage(`1`),
				Method:  "initialize",
				Params:  json.RawMessage(`{"protocolVersion":"` + tt.protocolVersion + `"}`),
			}
			body, _ := json.Marshal(req)

			r := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
			w := httptest.NewRecorder()

			h.ServeHTTP(w, r)

			if w.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d", w.Code)
			}

			var resp rpcResponse
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if tt.wantSuccess {
				if resp.Error != nil {
					t.Errorf("expected success, got error: %v", resp.Error)
				}
				if resp.Result == nil {
					t.Error("expected result, got nil")
				}
			} else {
				if resp.Error == nil {
					t.Error("expected error, got success")
				}
			}
		})
	}
}

func TestHandler_ToolsList(t *testing.T) {
	h := setupHandler(t)

	req := rpcRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "tools/list",
	}
	body, _ := json.Marshal(req)

	r := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp rpcResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("expected success, got error: %v", resp.Error)
	}

	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatal("expected result to be a map")
	}

	tools, ok := result["tools"].([]any)
	if !ok {
		t.Fatal("expected tools to be an array")
	}

	if len(tools) != 3 {
		t.Errorf("expected 3 tools, got %d", len(tools))
	}
}

func TestHandler_ToolsCall(t *testing.T) {
	h := setupHandler(t)

	req := rpcRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"test.Create","arguments":{"title":"hello"}}`),
	}
	body, _ := json.Marshal(req)

	r := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp rpcResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("expected success, got error: %v", resp.Error)
	}

	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatal("expected result to be a map")
	}

	isError, ok := result["isError"].(bool)
	if !ok || isError {
		t.Error("expected isError to be false")
	}
}

func TestHandler_ToolsCall_MethodNotFound(t *testing.T) {
	h := setupHandler(t)

	req := rpcRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"unknown.Method"}`),
	}
	body, _ := json.Marshal(req)

	r := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	var resp rpcResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error == nil {
		t.Error("expected error, got success")
	}

	if resp.Error.Code != codeMethodNotFound {
		t.Errorf("expected code %d, got %d", codeMethodNotFound, resp.Error.Code)
	}
}

func TestHandler_Notification(t *testing.T) {
	h := setupHandler(t)

	// Notification has no ID
	body := []byte(`{"jsonrpc":"2.0","method":"tools/list"}`)

	r := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected status 202, got %d", w.Code)
	}
}

func TestHandler_SSE(t *testing.T) {
	h := setupHandler(t)

	r := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "text/event-stream" {
		t.Errorf("expected content-type text/event-stream, got %s", contentType)
	}
}

func TestHandler_MethodNotAllowed(t *testing.T) {
	h := setupHandler(t)

	r := httptest.NewRequest(http.MethodPut, "/mcp", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

func TestHandler_InvalidJSON(t *testing.T) {
	h := setupHandler(t)

	r := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	var resp rpcResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error == nil {
		t.Error("expected error, got success")
	}

	if resp.Error.Code != codeParseError {
		t.Errorf("expected code %d, got %d", codeParseError, resp.Error.Code)
	}
}

func TestProtocolNegotiation(t *testing.T) {
	tests := []struct {
		requested string
		expected  string
	}{
		{"", ProtocolLatest},
		{ProtocolLatest, ProtocolLatest},
		{ProtocolFallback, ProtocolFallback},
		{ProtocolLegacy, ProtocolLegacy},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.requested, func(t *testing.T) {
			result := negotiateProtocol(tt.requested)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestMount(t *testing.T) {
	svc, err := contract.Register("test", &TestService{})
	if err != nil {
		t.Fatalf("failed to register service: %v", err)
	}

	mux := http.NewServeMux()
	Mount(mux, "/mcp", svc)

	// Test that the handler is mounted
	req := rpcRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "tools/list",
	}
	body, _ := json.Marshal(req)

	r := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}
