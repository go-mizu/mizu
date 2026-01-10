package api_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-mizu/blueprints/table/app/web"
)

type testServer struct {
	t       *testing.T
	server  *web.Server
	httpSrv *httptest.Server
	baseURL string
	dataDir string
	client  *http.Client
}

func newTestServer(t *testing.T) *testServer {
	t.Helper()
	dataDir := t.TempDir()
	srv, err := web.New(web.Config{
		Addr:    "127.0.0.1:0",
		DataDir: dataDir,
	})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	httpSrv := httptest.NewServer(srv.Handler())
	t.Cleanup(func() {
		httpSrv.Close()
		if err := srv.Close(); err != nil {
			t.Logf("server close: %v", err)
		}
	})

	return &testServer{
		t:       t,
		server:  srv,
		httpSrv: httpSrv,
		baseURL: httpSrv.URL,
		dataDir: dataDir,
		client:  httpSrv.Client(),
	}
}

func (ts *testServer) apiPath(path string) string {
	if strings.HasPrefix(path, "/api/") {
		return path
	}
	return "/api/v1" + path
}

func (ts *testServer) do(method, path string, body any, token string) (int, []byte) {
	ts.t.Helper()

	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			ts.t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, ts.baseURL+ts.apiPath(path), reader)
	if err != nil {
		ts.t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := ts.client.Do(req)
	if err != nil {
		ts.t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		ts.t.Fatalf("read response: %v", err)
	}

	return resp.StatusCode, respBody
}

func (ts *testServer) doJSON(method, path string, body any, token string) (int, map[string]any) {
	ts.t.Helper()
	status, respBody := ts.do(method, path, body, token)
	if len(respBody) == 0 {
		return status, nil
	}

	var payload map[string]any
	if err := json.Unmarshal(respBody, &payload); err != nil {
		ts.t.Fatalf("unmarshal response: %v", err)
	}
	return status, payload
}

func requireMap(t *testing.T, payload map[string]any, key string) map[string]any {
	t.Helper()
	value, ok := payload[key].(map[string]any)
	if !ok {
		t.Fatalf("expected map for %q, got %T", key, payload[key])
	}
	return value
}

func requireSlice(t *testing.T, payload map[string]any, key string) []any {
	t.Helper()
	if payload[key] == nil {
		return []any{}
	}
	value, ok := payload[key].([]any)
	if !ok {
		t.Fatalf("expected slice for %q, got %T", key, payload[key])
	}
	return value
}

func requireString(t *testing.T, payload map[string]any, key string) string {
	t.Helper()
	value, ok := payload[key].(string)
	if !ok || value == "" {
		t.Fatalf("expected non-empty string for %q, got %v", key, payload[key])
	}
	return value
}

func requireBool(t *testing.T, payload map[string]any, key string) bool {
	t.Helper()
	value, ok := payload[key].(bool)
	if !ok {
		t.Fatalf("expected bool for %q, got %T", key, payload[key])
	}
	return value
}

func registerUser(t *testing.T, ts *testServer, email string) (string, map[string]any) {
	t.Helper()
	status, data := ts.doJSON(http.MethodPost, "/auth/register", map[string]any{
		"email":    email,
		"name":     "Test User",
		"password": "secret123!",
	}, "")
	if status != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", status)
	}
	token := requireString(t, data, "token")
	user := requireMap(t, data, "user")
	return token, user
}

func createWorkspace(t *testing.T, ts *testServer, token, name, slug string) map[string]any {
	t.Helper()
	status, data := ts.doJSON(http.MethodPost, "/workspaces", map[string]any{
		"name": name,
		"slug": slug,
	}, token)
	if status != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", status)
	}
	return requireMap(t, data, "workspace")
}

func createBase(t *testing.T, ts *testServer, token, workspaceID, name string) map[string]any {
	t.Helper()
	status, data := ts.doJSON(http.MethodPost, "/bases", map[string]any{
		"workspace_id": workspaceID,
		"name":         name,
	}, token)
	if status != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", status)
	}
	return requireMap(t, data, "base")
}

func createTable(t *testing.T, ts *testServer, token, baseID, name string) map[string]any {
	t.Helper()
	status, data := ts.doJSON(http.MethodPost, "/tables", map[string]any{
		"base_id": baseID,
		"name":    name,
	}, token)
	if status != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", status)
	}
	return requireMap(t, data, "table")
}

func createField(t *testing.T, ts *testServer, token, tableID, name, fieldType string) map[string]any {
	t.Helper()
	status, data := ts.doJSON(http.MethodPost, "/fields", map[string]any{
		"table_id": tableID,
		"name":     name,
		"type":     fieldType,
	}, token)
	if status != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", status)
	}
	return requireMap(t, data, "field")
}

func createView(t *testing.T, ts *testServer, token, tableID, name, viewType string) map[string]any {
	t.Helper()
	status, data := ts.doJSON(http.MethodPost, "/views", map[string]any{
		"table_id": tableID,
		"name":     name,
		"type":     viewType,
	}, token)
	if status != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", status)
	}
	return requireMap(t, data, "view")
}

func createRecord(t *testing.T, ts *testServer, token, tableID string, fields map[string]any) map[string]any {
	t.Helper()
	status, data := ts.doJSON(http.MethodPost, "/records", map[string]any{
		"table_id": tableID,
		"fields":   fields,
	}, token)
	if status != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", status)
	}
	return requireMap(t, data, "record")
}

func createComment(t *testing.T, ts *testServer, token, recordID, content string) map[string]any {
	t.Helper()
	status, data := ts.doJSON(http.MethodPost, "/comments", map[string]any{
		"record_id": recordID,
		"content":   content,
	}, token)
	if status != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", status)
	}
	return requireMap(t, data, "comment")
}

func createShare(t *testing.T, ts *testServer, token, baseID, shareType, permission string) map[string]any {
	t.Helper()
	status, data := ts.doJSON(http.MethodPost, "/shares", map[string]any{
		"base_id":    baseID,
		"type":       shareType,
		"permission": permission,
	}, token)
	if status != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", status)
	}
	return requireMap(t, data, "share")
}
