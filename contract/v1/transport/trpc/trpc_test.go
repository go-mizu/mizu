package trpc

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-mizu/mizu/contract"
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
	return NewHandler("/trpc", svc)
}

func setupMux(t *testing.T) *http.ServeMux {
	svc, err := contract.Register("test", &TestService{})
	if err != nil {
		t.Fatalf("failed to register service: %v", err)
	}
	mux := http.NewServeMux()
	Mount(mux, "/trpc", svc)
	return mux
}

func TestHandler_Call(t *testing.T) {
	mux := setupMux(t)

	body := []byte(`{"title":"hello"}`)
	r := httptest.NewRequest(http.MethodPost, "/trpc/Create", bytes.NewReader(body))
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp Envelope
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("expected success, got error: %v", resp.Error)
	}

	if resp.Result == nil {
		t.Error("expected result, got nil")
	}

	data, ok := resp.Result.Data.(map[string]any)
	if !ok {
		t.Fatal("expected data to be a map")
	}

	if data["title"] != "hello" {
		t.Errorf("expected title 'hello', got %v", data["title"])
	}
}

func TestHandler_Call_ServicePrefix(t *testing.T) {
	mux := setupMux(t)

	body := []byte(`{"title":"hello"}`)
	r := httptest.NewRequest(http.MethodPost, "/trpc/test.Create", bytes.NewReader(body))
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp Envelope
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("expected success, got error: %v", resp.Error)
	}
}

func TestHandler_Call_NoInput(t *testing.T) {
	mux := setupMux(t)

	r := httptest.NewRequest(http.MethodPost, "/trpc/List", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp Envelope
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("expected success, got error: %v", resp.Error)
	}
}

func TestHandler_Call_VoidOutput(t *testing.T) {
	mux := setupMux(t)

	r := httptest.NewRequest(http.MethodPost, "/trpc/Health", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp Envelope
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("expected success, got error: %v", resp.Error)
	}

	if resp.Result == nil {
		t.Error("expected result, got nil")
	}
}

func TestHandler_Call_UnknownProcedure(t *testing.T) {
	mux := setupMux(t)

	r := httptest.NewRequest(http.MethodPost, "/trpc/Unknown", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var resp Envelope
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error == nil {
		t.Error("expected error, got success")
	}

	if resp.Error.Code != CodeBadRequest {
		t.Errorf("expected code %s, got %s", CodeBadRequest, resp.Error.Code)
	}
}

func TestHandler_Call_InvalidJSON(t *testing.T) {
	mux := setupMux(t)

	r := httptest.NewRequest(http.MethodPost, "/trpc/Create", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestHandler_Meta(t *testing.T) {
	mux := setupMux(t)

	r := httptest.NewRequest(http.MethodGet, "/trpc.meta", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var meta ServiceMeta
	if err := json.NewDecoder(w.Body).Decode(&meta); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if meta.Service != "test" {
		t.Errorf("expected service 'test', got %s", meta.Service)
	}

	if len(meta.Methods) != 3 {
		t.Errorf("expected 3 methods, got %d", len(meta.Methods))
	}
}

func TestHandler_Meta_MethodNotAllowed(t *testing.T) {
	mux := setupMux(t)

	r := httptest.NewRequest(http.MethodPost, "/trpc.meta", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

func TestHandler_Call_MethodNotAllowed(t *testing.T) {
	mux := setupMux(t)

	r := httptest.NewRequest(http.MethodGet, "/trpc/Create", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

func TestEnvelope_Success(t *testing.T) {
	env := SuccessEnvelope(map[string]string{"id": "1"})

	if env.Error != nil {
		t.Error("expected no error")
	}

	if env.Result == nil {
		t.Error("expected result")
	}

	data, ok := env.Result.Data.(map[string]string)
	if !ok {
		t.Fatal("expected data to be map[string]string")
	}

	if data["id"] != "1" {
		t.Errorf("expected id '1', got %v", data["id"])
	}
}

func TestEnvelope_Error(t *testing.T) {
	env := ErrorEnvelope(CodeInternalError, "something went wrong")

	if env.Result != nil {
		t.Error("expected no result")
	}

	if env.Error == nil {
		t.Error("expected error")
	}

	if env.Error.Code != CodeInternalError {
		t.Errorf("expected code %s, got %s", CodeInternalError, env.Error.Code)
	}

	if env.Error.Message != "something went wrong" {
		t.Errorf("expected message 'something went wrong', got %s", env.Error.Message)
	}
}

func TestMount(t *testing.T) {
	svc, err := contract.Register("test", &TestService{})
	if err != nil {
		t.Fatalf("failed to register service: %v", err)
	}

	mux := http.NewServeMux()
	Mount(mux, "/trpc", svc)

	// Test that meta handler is mounted
	r := httptest.NewRequest(http.MethodGet, "/trpc.meta", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Test that call handler is mounted
	r = httptest.NewRequest(http.MethodPost, "/trpc/List", nil)
	w = httptest.NewRecorder()

	mux.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestMount_DefaultPath(t *testing.T) {
	svc, err := contract.Register("test", &TestService{})
	if err != nil {
		t.Fatalf("failed to register service: %v", err)
	}

	mux := http.NewServeMux()
	Mount(mux, "", svc)

	r := httptest.NewRequest(http.MethodGet, "/trpc.meta", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}
