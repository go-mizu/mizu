package jsonrpc

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/contract"
)

// Test service for JSON-RPC tests
type testService struct{}

type echoInput struct {
	Message string `json:"message"`
}

type echoOutput struct {
	Echo string `json:"echo"`
}

func (s *testService) Echo(ctx context.Context, in *echoInput) (*echoOutput, error) {
	return &echoOutput{Echo: in.Message}, nil
}

func (s *testService) Ping(ctx context.Context) error {
	return nil
}

func (s *testService) Error(ctx context.Context) error {
	return contract.ErrNotFound("resource not found")
}

func TestCodec_Decode(t *testing.T) {
	codec := NewCodec()

	tests := []struct {
		name      string
		input     string
		wantCount int
		wantErr   bool
	}{
		{
			name:      "single request",
			input:     `{"jsonrpc":"2.0","id":1,"method":"Echo","params":{"message":"hello"}}`,
			wantCount: 1,
		},
		{
			name:      "batch request",
			input:     `[{"jsonrpc":"2.0","id":1,"method":"Echo"},{"jsonrpc":"2.0","id":2,"method":"Ping"}]`,
			wantCount: 2,
		},
		{
			name:    "empty body",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			input:   `{invalid}`,
			wantErr: true,
		},
		{
			name:    "empty batch",
			input:   `[]`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests, err := codec.Decode(strings.NewReader(tt.input))
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(requests) != tt.wantCount {
				t.Errorf("got %d requests, want %d", len(requests), tt.wantCount)
			}
		})
	}
}

func TestCodec_Encode(t *testing.T) {
	codec := NewCodec()

	resp := NewResponse(json.RawMessage("1"), map[string]string{"key": "value"})

	var buf bytes.Buffer
	if err := codec.Encode(&buf, resp); err != nil {
		t.Fatalf("encode error: %v", err)
	}

	var decoded Response
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if decoded.JSONRPC != Version {
		t.Errorf("got jsonrpc=%q, want %q", decoded.JSONRPC, Version)
	}
	if decoded.Error != nil {
		t.Errorf("unexpected error: %v", decoded.Error)
	}
}

func TestCodec_EncodeError(t *testing.T) {
	codec := NewCodec()

	var buf bytes.Buffer
	err := codec.EncodeError(&buf, json.RawMessage("1"), MethodNotFound("Test"))

	if err != nil {
		t.Fatalf("encode error: %v", err)
	}

	var resp Response
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if resp.Error == nil {
		t.Fatal("expected error in response")
	}
	if resp.Error.Code != CodeMethodNotFound {
		t.Errorf("got code=%d, want %d", resp.Error.Code, CodeMethodNotFound)
	}
}

func TestRequest_IsNotification(t *testing.T) {
	tests := []struct {
		name string
		id   json.RawMessage
		want bool
	}{
		{"nil id", nil, true},
		{"empty id", json.RawMessage(""), true},
		{"null id", json.RawMessage("null"), true},
		{"number id", json.RawMessage("1"), false},
		{"string id", json.RawMessage(`"abc"`), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &Request{ID: tt.id}
			if got := req.IsNotification(); got != tt.want {
				t.Errorf("IsNotification() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestError_Methods(t *testing.T) {
	err := NewError(CodeMethodNotFound, "method not found")
	if err.Error() != "method not found" {
		t.Errorf("Error() = %q, want %q", err.Error(), "method not found")
	}

	err = NewError(CodeMethodNotFound, "")
	if err.Error() != "Method not found" {
		t.Errorf("Error() with default = %q, want %q", err.Error(), "Method not found")
	}

	err = err.WithData(map[string]string{"key": "value"})
	if err.Data == nil {
		t.Error("expected data to be set")
	}
}

func TestHandler_ServeHTTP(t *testing.T) {
	svc, err := contract.Register("test", &testService{})
	if err != nil {
		t.Fatalf("register error: %v", err)
	}

	handler := NewHandler(svc)
	mux := http.NewServeMux()
	mux.Handle("/rpc", handler)

	tests := []struct {
		name       string
		method     string
		body       string
		wantStatus int
		wantResult bool
		wantError  bool
	}{
		{
			name:       "valid request",
			method:     http.MethodPost,
			body:       `{"jsonrpc":"2.0","id":1,"method":"Echo","params":{"message":"hello"}}`,
			wantStatus: http.StatusOK,
			wantResult: true,
		},
		{
			name:       "method not found",
			method:     http.MethodPost,
			body:       `{"jsonrpc":"2.0","id":1,"method":"Unknown"}`,
			wantStatus: http.StatusOK,
			wantError:  true,
		},
		{
			name:       "invalid JSON",
			method:     http.MethodPost,
			body:       `{invalid}`,
			wantStatus: http.StatusOK,
			wantError:  true,
		},
		{
			name:       "wrong HTTP method",
			method:     http.MethodGet,
			body:       `{}`,
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "notification",
			method:     http.MethodPost,
			body:       `{"jsonrpc":"2.0","method":"Ping"}`,
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "service.method format",
			method:     http.MethodPost,
			body:       `{"jsonrpc":"2.0","id":1,"method":"test.Echo","params":{"message":"hi"}}`,
			wantStatus: http.StatusOK,
			wantResult: true,
		},
		{
			name:       "error from method",
			method:     http.MethodPost,
			body:       `{"jsonrpc":"2.0","id":1,"method":"Error"}`,
			wantStatus: http.StatusOK,
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/rpc", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			if tt.wantResult || tt.wantError {
				var resp Response
				if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
					t.Fatalf("decode response: %v", err)
				}

				if tt.wantResult && resp.Result == nil {
					t.Error("expected result")
				}
				if tt.wantError && resp.Error == nil {
					t.Error("expected error")
				}
				if !tt.wantError && resp.Error != nil {
					t.Errorf("unexpected error: %v", resp.Error)
				}
			}
		})
	}
}

func TestHandler_Batch(t *testing.T) {
	svc, _ := contract.Register("test", &testService{})
	handler := NewHandler(svc)

	body := `[
		{"jsonrpc":"2.0","id":1,"method":"Echo","params":{"message":"one"}},
		{"jsonrpc":"2.0","id":2,"method":"Echo","params":{"message":"two"}},
		{"jsonrpc":"2.0","method":"Ping"}
	]`

	req := httptest.NewRequest(http.MethodPost, "/rpc", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var responses []Response
	if err := json.Unmarshal(rec.Body.Bytes(), &responses); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	// Should have 2 responses (notification excluded)
	if len(responses) != 2 {
		t.Errorf("got %d responses, want 2", len(responses))
	}
}

func TestHandler_AllNotifications(t *testing.T) {
	svc, _ := contract.Register("test", &testService{})
	handler := NewHandler(svc)

	body := `[{"jsonrpc":"2.0","method":"Ping"},{"jsonrpc":"2.0","method":"Ping"}]`

	req := httptest.NewRequest(http.MethodPost, "/rpc", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

func TestMount(t *testing.T) {
	svc, _ := contract.Register("test", &testService{})
	mux := http.NewServeMux()

	Mount(mux, "/rpc", svc)

	body := `{"jsonrpc":"2.0","id":1,"method":"Ping"}`
	req := httptest.NewRequest(http.MethodPost, "/rpc", strings.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestMount_DefaultPath(t *testing.T) {
	svc, _ := contract.Register("test", &testService{})
	mux := http.NewServeMux()

	Mount(mux, "", svc)

	body := `{"jsonrpc":"2.0","id":1,"method":"Ping"}`
	req := httptest.NewRequest(http.MethodPost, "/jsonrpc", strings.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestMapError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode int
	}{
		{
			name:     "nil error",
			err:      nil,
			wantCode: 0,
		},
		{
			name:     "contract not found",
			err:      contract.ErrNotFound("not found"),
			wantCode: CodeMethodNotFound,
		},
		{
			name:     "contract invalid argument",
			err:      contract.ErrInvalidArgument("bad input"),
			wantCode: CodeInvalidParams,
		},
		{
			name:     "contract internal",
			err:      contract.ErrInternal("oops"),
			wantCode: CodeInternalError,
		},
		{
			name:     "json-rpc error passthrough",
			err:      NewError(CodeParseError, "parse error"),
			wantCode: CodeParseError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rpcErr := MapError(tt.err)
			if tt.err == nil {
				if rpcErr != nil {
					t.Error("expected nil")
				}
				return
			}
			if rpcErr == nil {
				t.Fatal("expected error")
			}
			if rpcErr.Code != tt.wantCode {
				t.Errorf("code = %d, want %d", rpcErr.Code, tt.wantCode)
			}
		})
	}
}
