package jsonrpc

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestServer_Register(t *testing.T) {
	server := NewServer()
	server.Register("add", func(params map[string]any) (any, error) {
		a := params["a"].(float64)
		b := params["b"].(float64)
		return a + b, nil
	})

	app := mizu.NewRouter()
	app.Post("/rpc", server.Handler())

	body := `{"jsonrpc": "2.0", "method": "add", "params": {"a": 1, "b": 2}, "id": 1}`
	req := httptest.NewRequest(http.MethodPost, "/rpc", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	var resp Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Result != float64(3) {
		t.Errorf("expected result 3, got %v", resp.Result)
	}
	if resp.Error != nil {
		t.Errorf("expected no error, got %v", resp.Error)
	}
}

func TestServer_MethodNotFound(t *testing.T) {
	server := NewServer()

	app := mizu.NewRouter()
	app.Post("/rpc", server.Handler())

	body := `{"jsonrpc": "2.0", "method": "unknown", "id": 1}`
	req := httptest.NewRequest(http.MethodPost, "/rpc", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	var resp Response
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)

	if resp.Error == nil {
		t.Fatal("expected error")
	}
	if resp.Error.Code != MethodNotFound {
		t.Errorf("expected code %d, got %d", MethodNotFound, resp.Error.Code)
	}
}

func TestServer_InvalidJSONRPCVersion(t *testing.T) {
	server := NewServer()
	server.Register("test", func(params map[string]any) (any, error) {
		return nil, nil
	})

	app := mizu.NewRouter()
	app.Post("/rpc", server.Handler())

	body := `{"jsonrpc": "1.0", "method": "test", "id": 1}`
	req := httptest.NewRequest(http.MethodPost, "/rpc", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	var resp Response
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)

	if resp.Error == nil || resp.Error.Code != InvalidRequest {
		t.Errorf("expected InvalidRequest error")
	}
}

func TestServer_Notification(t *testing.T) {
	called := false
	server := NewServer()
	server.Register("notify", func(params map[string]any) (any, error) {
		called = true
		return nil, nil
	})

	app := mizu.NewRouter()
	app.Post("/rpc", server.Handler())

	// No ID = notification
	body := `{"jsonrpc": "2.0", "method": "notify"}`
	req := httptest.NewRequest(http.MethodPost, "/rpc", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !called {
		t.Error("expected handler to be called")
	}
	if rec.Code != http.StatusNoContent {
		t.Errorf("expected %d for notification, got %d", http.StatusNoContent, rec.Code)
	}
}

func TestServer_BatchRequest(t *testing.T) {
	server := NewServer()
	server.Register("double", func(params map[string]any) (any, error) {
		n := params["n"].(float64)
		return n * 2, nil
	})

	app := mizu.NewRouter()
	app.Post("/rpc", server.Handler())

	body := `[
		{"jsonrpc": "2.0", "method": "double", "params": {"n": 5}, "id": 1},
		{"jsonrpc": "2.0", "method": "double", "params": {"n": 10}, "id": 2}
	]`
	req := httptest.NewRequest(http.MethodPost, "/rpc", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	var responses []Response
	if err := json.Unmarshal(rec.Body.Bytes(), &responses); err != nil {
		t.Fatalf("failed to parse batch response: %v", err)
	}

	if len(responses) != 2 {
		t.Errorf("expected 2 responses, got %d", len(responses))
	}
}

func TestServer_ErrorInHandler(t *testing.T) {
	server := NewServer()
	server.Register("fail", func(params map[string]any) (any, error) {
		return nil, NewError(InternalError, "something went wrong")
	})

	app := mizu.NewRouter()
	app.Post("/rpc", server.Handler())

	body := `{"jsonrpc": "2.0", "method": "fail", "id": 1}`
	req := httptest.NewRequest(http.MethodPost, "/rpc", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	var resp Response
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)

	if resp.Error == nil {
		t.Fatal("expected error")
	}
	if resp.Error.Message != "something went wrong" {
		t.Errorf("expected error message, got %q", resp.Error.Message)
	}
}

func TestMiddleware(t *testing.T) {
	server := NewServer()

	app := mizu.NewRouter()
	app.Use(Middleware())
	app.Post("/rpc", server.Handler())

	t.Run("wrong method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/rpc", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		var resp Response
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)

		if resp.Error == nil {
			t.Error("expected error for GET request")
		}
	})

	t.Run("wrong content type", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/rpc", strings.NewReader("{}"))
		req.Header.Set("Content-Type", "text/plain")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		var resp Response
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)

		if resp.Error == nil {
			t.Error("expected error for wrong content type")
		}
	})
}

func TestParseError(t *testing.T) {
	server := NewServer()

	app := mizu.NewRouter()
	app.Post("/rpc", server.Handler())

	body := `{invalid json`
	req := httptest.NewRequest(http.MethodPost, "/rpc", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	var resp Response
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)

	if resp.Error == nil || resp.Error.Code != ParseError {
		t.Errorf("expected ParseError")
	}
}

func TestNewError(t *testing.T) {
	err := NewError(123, "test error")
	if err.Code != 123 {
		t.Errorf("expected code 123, got %d", err.Code)
	}
	if err.Message != "test error" {
		t.Errorf("expected message 'test error', got %q", err.Message)
	}
	if err.Error() != "test error" {
		t.Errorf("expected Error() to return message")
	}
}
