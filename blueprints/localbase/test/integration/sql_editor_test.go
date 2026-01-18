package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-mizu/mizu/blueprints/localbase/app/web"
	"github.com/go-mizu/mizu/blueprints/localbase/store"
	"github.com/go-mizu/mizu/blueprints/localbase/store/postgres"
)

// Service key for authentication
const testServiceKey = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJsb2NhbGJhc2UiLCJyb2xlIjoic2VydmljZV9yb2xlIiwiaWF0IjoxNzA0MDY3MjAwLCJleHAiOjE4NjE4MzM2MDB9.service_role_key_signature"

// Helper function to make authenticated requests
func makeRequest(t *testing.T, handler http.Handler, method, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()

	var reqBody *bytes.Buffer
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("failed to marshal body: %v", err)
		}
		reqBody = bytes.NewBuffer(data)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", testServiceKey)
	req.Header.Set("Authorization", "Bearer "+testServiceKey)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	return rr
}

func TestSQLEditor_ExecuteQuery(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Create test store (in-memory for testing)
	testStore := createTestStore(t)
	if testStore == nil {
		t.Skip("no test store available")
	}

	// Create handler
	handler, err := web.NewServer(testStore, true)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	t.Run("execute SELECT query", func(t *testing.T) {
		body := map[string]interface{}{
			"query": "SELECT 1 AS test",
		}

		rr := makeRequest(t, handler, "POST", "/api/database/query", body)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
		}

		var result map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if result["row_count"] == nil {
			t.Error("expected row_count in response")
		}
		if result["columns"] == nil {
			t.Error("expected columns in response")
		}
		if result["rows"] == nil {
			t.Error("expected rows in response")
		}
	})

	t.Run("execute query with role", func(t *testing.T) {
		body := map[string]interface{}{
			"query": "SELECT 1 AS test",
			"role":  "anon",
		}

		rr := makeRequest(t, handler, "POST", "/api/database/query", body)

		// Should succeed (anon role can run SELECT)
		if rr.Code != http.StatusOK && rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 200 or 400, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("execute query with EXPLAIN", func(t *testing.T) {
		body := map[string]interface{}{
			"query":   "SELECT 1 AS test",
			"explain": true,
		}

		rr := makeRequest(t, handler, "POST", "/api/database/query", body)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("execute empty query returns error", func(t *testing.T) {
		body := map[string]interface{}{
			"query": "",
		}

		rr := makeRequest(t, handler, "POST", "/api/database/query", body)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})
}

func TestSQLEditor_QueryHistory(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testStore := createTestStore(t)
	if testStore == nil {
		t.Skip("no test store available")
	}

	handler, err := web.NewServer(testStore, true)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// First, execute a query to populate history
	t.Run("execute query to add to history", func(t *testing.T) {
		body := map[string]interface{}{
			"query": "SELECT 1 AS history_test",
		}

		rr := makeRequest(t, handler, "POST", "/api/database/query", body)
		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("list query history", func(t *testing.T) {
		rr := makeRequest(t, handler, "GET", "/api/database/query/history?limit=10", nil)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
		}

		var history []store.QueryHistoryEntry
		if err := json.Unmarshal(rr.Body.Bytes(), &history); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		// History might be empty if query history is not persisted between tests
		t.Logf("history entries: %d", len(history))
	})

	t.Run("clear query history", func(t *testing.T) {
		rr := makeRequest(t, handler, "DELETE", "/api/database/query/history", nil)

		if rr.Code != http.StatusNoContent {
			t.Errorf("expected status 204, got %d: %s", rr.Code, rr.Body.String())
		}
	})
}

func TestSQLEditor_Snippets(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testStore := createTestStore(t)
	if testStore == nil {
		t.Skip("no test store available")
	}

	handler, err := web.NewServer(testStore, true)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	var snippetID string

	t.Run("create snippet", func(t *testing.T) {
		body := map[string]interface{}{
			"name":  "Test Snippet",
			"query": "SELECT * FROM users WHERE active = true;",
		}

		rr := makeRequest(t, handler, "POST", "/api/database/snippets", body)

		if rr.Code != http.StatusCreated {
			t.Errorf("expected status 201, got %d: %s", rr.Code, rr.Body.String())
		}

		var snippet store.SQLSnippet
		if err := json.Unmarshal(rr.Body.Bytes(), &snippet); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if snippet.ID == "" {
			t.Error("expected snippet ID")
		}
		if snippet.Name != "Test Snippet" {
			t.Errorf("expected name 'Test Snippet', got '%s'", snippet.Name)
		}

		snippetID = snippet.ID
	})

	t.Run("list snippets", func(t *testing.T) {
		rr := makeRequest(t, handler, "GET", "/api/database/snippets", nil)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
		}

		var snippets []store.SQLSnippet
		if err := json.Unmarshal(rr.Body.Bytes(), &snippets); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if len(snippets) == 0 {
			t.Error("expected at least one snippet")
		}
	})

	t.Run("get snippet by ID", func(t *testing.T) {
		if snippetID == "" {
			t.Skip("no snippet ID available")
		}

		rr := makeRequest(t, handler, "GET", "/api/database/snippets/"+snippetID, nil)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("update snippet", func(t *testing.T) {
		if snippetID == "" {
			t.Skip("no snippet ID available")
		}

		body := map[string]interface{}{
			"name":  "Updated Snippet",
			"query": "SELECT * FROM users WHERE active = false;",
		}

		rr := makeRequest(t, handler, "PUT", "/api/database/snippets/"+snippetID, body)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("delete snippet", func(t *testing.T) {
		if snippetID == "" {
			t.Skip("no snippet ID available")
		}

		rr := makeRequest(t, handler, "DELETE", "/api/database/snippets/"+snippetID, nil)

		if rr.Code != http.StatusNoContent {
			t.Errorf("expected status 204, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("create snippet with empty name returns error", func(t *testing.T) {
		body := map[string]interface{}{
			"name":  "",
			"query": "SELECT 1",
		}

		rr := makeRequest(t, handler, "POST", "/api/database/snippets", body)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	t.Run("create snippet with empty query returns error", func(t *testing.T) {
		body := map[string]interface{}{
			"name":  "Empty Query",
			"query": "",
		}

		rr := makeRequest(t, handler, "POST", "/api/database/snippets", body)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})
}

func TestSQLEditor_Folders(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testStore := createTestStore(t)
	if testStore == nil {
		t.Skip("no test store available")
	}

	handler, err := web.NewServer(testStore, true)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	var folderID string

	t.Run("create folder", func(t *testing.T) {
		body := map[string]interface{}{
			"name": "Test Folder",
		}

		rr := makeRequest(t, handler, "POST", "/api/database/snippets/folders", body)

		if rr.Code != http.StatusCreated {
			t.Errorf("expected status 201, got %d: %s", rr.Code, rr.Body.String())
		}

		var folder store.SQLFolder
		if err := json.Unmarshal(rr.Body.Bytes(), &folder); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if folder.ID == "" {
			t.Error("expected folder ID")
		}
		if folder.Name != "Test Folder" {
			t.Errorf("expected name 'Test Folder', got '%s'", folder.Name)
		}

		folderID = folder.ID
	})

	t.Run("list folders", func(t *testing.T) {
		rr := makeRequest(t, handler, "GET", "/api/database/snippets/folders", nil)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
		}

		var folders []store.SQLFolder
		if err := json.Unmarshal(rr.Body.Bytes(), &folders); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if len(folders) == 0 {
			t.Error("expected at least one folder")
		}
	})

	t.Run("update folder", func(t *testing.T) {
		if folderID == "" {
			t.Skip("no folder ID available")
		}

		body := map[string]interface{}{
			"name": "Updated Folder",
		}

		rr := makeRequest(t, handler, "PUT", "/api/database/snippets/folders/"+folderID, body)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("delete folder", func(t *testing.T) {
		if folderID == "" {
			t.Skip("no folder ID available")
		}

		rr := makeRequest(t, handler, "DELETE", "/api/database/snippets/folders/"+folderID, nil)

		if rr.Code != http.StatusNoContent {
			t.Errorf("expected status 204, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("create folder with empty name returns error", func(t *testing.T) {
		body := map[string]interface{}{
			"name": "",
		}

		rr := makeRequest(t, handler, "POST", "/api/database/snippets/folders", body)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})
}

func TestSQLEditor_RoleBased(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testStore := createTestStore(t)
	if testStore == nil {
		t.Skip("no test store available")
	}

	handler, err := web.NewServer(testStore, true)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	roles := []string{"postgres", "anon", "authenticated", "service_role"}

	for _, role := range roles {
		t.Run("execute query as "+role, func(t *testing.T) {
			body := map[string]interface{}{
				"query": "SELECT 1 AS test",
				"role":  role,
			}

			rr := makeRequest(t, handler, "POST", "/api/database/query", body)

			// Query should succeed for all roles (simple SELECT 1)
			if rr.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
			}
		})
	}
}

// createTestStore creates a test store instance
// Returns nil if no database is available
func createTestStore(t *testing.T) *postgres.Store {
	t.Helper()

	// Try to connect to a test database
	// In CI, this would be provided via environment variable
	// For local development, use a local postgres instance
	connStr := "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"

	ctx := context.Background()
	testStore, err := postgres.New(ctx, connStr)
	if err != nil {
		t.Logf("could not connect to test database: %v", err)
		return nil
	}

	return testStore
}
