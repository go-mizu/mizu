package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/go-mizu/mizu/blueprints/localbase/app/web"
	"github.com/go-mizu/mizu/blueprints/localbase/app/web/handler/api"
)

func TestDatabase_Overview(t *testing.T) {
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

	t.Run("get database overview", func(t *testing.T) {
		rr := makeRequest(t, handler, "GET", "/api/database/overview", nil)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
		}

		var overview api.DatabaseOverview
		if err := json.Unmarshal(rr.Body.Bytes(), &overview); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		// Verify overview structure
		if overview.Schemas == nil {
			t.Error("expected schemas in response")
		}
		if overview.DatabaseSize == "" {
			t.Error("expected database_size in response")
		}

		t.Logf("Overview: tables=%d, views=%d, functions=%d, size=%s",
			overview.TotalTables, overview.TotalViews, overview.TotalFunctions, overview.DatabaseSize)
	})
}

func TestDatabase_TableStats(t *testing.T) {
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

	t.Run("get table stats for public schema", func(t *testing.T) {
		rr := makeRequest(t, handler, "GET", "/api/database/tables/stats?schema=public", nil)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
		}

		var stats []api.TableStats
		if err := json.Unmarshal(rr.Body.Bytes(), &stats); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		t.Logf("Found %d tables in public schema", len(stats))
	})

	t.Run("get table stats for auth schema", func(t *testing.T) {
		rr := makeRequest(t, handler, "GET", "/api/database/tables/stats?schema=auth", nil)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})
}

func TestDatabase_Tables(t *testing.T) {
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

	tableName := "test_table_" + randomString(8)

	t.Run("create table", func(t *testing.T) {
		body := map[string]interface{}{
			"schema": "public",
			"name":   tableName,
			"columns": []map[string]interface{}{
				{
					"name":           "id",
					"type":           "uuid",
					"is_primary_key": true,
					"default_value":  "gen_random_uuid()",
				},
				{
					"name":        "name",
					"type":        "text",
					"is_nullable": false,
				},
				{
					"name":        "created_at",
					"type":        "timestamptz",
					"is_nullable": false,
					"default_value": "now()",
				},
			},
		}

		rr := makeRequest(t, handler, "POST", "/api/database/tables", body)

		if rr.Code != http.StatusCreated {
			t.Errorf("expected status 201, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("list tables", func(t *testing.T) {
		rr := makeRequest(t, handler, "GET", "/api/database/tables?schema=public", nil)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("get table", func(t *testing.T) {
		rr := makeRequest(t, handler, "GET", "/api/database/tables/public/"+tableName, nil)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("get table data", func(t *testing.T) {
		rr := makeRequest(t, handler, "GET", "/api/database/tables/public/"+tableName+"/data", nil)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("add column", func(t *testing.T) {
		body := map[string]interface{}{
			"name":        "email",
			"type":        "text",
			"is_nullable": true,
		}

		rr := makeRequest(t, handler, "POST", "/api/database/tables/public/"+tableName+"/columns", body)

		if rr.Code != http.StatusCreated {
			t.Errorf("expected status 201, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("list columns", func(t *testing.T) {
		rr := makeRequest(t, handler, "GET", "/api/database/tables/public/"+tableName+"/columns", nil)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("drop column", func(t *testing.T) {
		rr := makeRequest(t, handler, "DELETE", "/api/database/tables/public/"+tableName+"/columns/email", nil)

		if rr.Code != http.StatusNoContent {
			t.Errorf("expected status 204, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("drop table", func(t *testing.T) {
		rr := makeRequest(t, handler, "DELETE", "/api/database/tables/public/"+tableName, nil)

		if rr.Code != http.StatusNoContent {
			t.Errorf("expected status 204, got %d: %s", rr.Code, rr.Body.String())
		}
	})
}

func TestDatabase_Indexes(t *testing.T) {
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

	// Create a test table first
	tableName := "test_idx_table_" + randomString(8)
	createTableBody := map[string]interface{}{
		"schema": "public",
		"name":   tableName,
		"columns": []map[string]interface{}{
			{
				"name":           "id",
				"type":           "uuid",
				"is_primary_key": true,
				"default_value":  "gen_random_uuid()",
			},
			{
				"name":        "email",
				"type":        "text",
				"is_nullable": false,
			},
		},
	}
	rr := makeRequest(t, handler, "POST", "/api/database/tables", createTableBody)
	if rr.Code != http.StatusCreated {
		t.Fatalf("failed to create test table: %s", rr.Body.String())
	}

	indexName := "idx_" + randomString(8)

	t.Run("create index", func(t *testing.T) {
		body := map[string]interface{}{
			"name":      indexName,
			"schema":    "public",
			"table":     tableName,
			"columns":   []string{"email"},
			"type":      "btree",
			"is_unique": true,
		}

		rr := makeRequest(t, handler, "POST", "/api/database/indexes", body)

		if rr.Code != http.StatusCreated {
			t.Errorf("expected status 201, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("list indexes", func(t *testing.T) {
		rr := makeRequest(t, handler, "GET", "/api/database/indexes?schema=public&table="+tableName, nil)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("drop index", func(t *testing.T) {
		rr := makeRequest(t, handler, "DELETE", "/api/database/indexes/public/"+indexName, nil)

		if rr.Code != http.StatusNoContent {
			t.Errorf("expected status 204, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	// Cleanup: drop the test table
	makeRequest(t, handler, "DELETE", "/api/database/tables/public/"+tableName, nil)
}

func TestDatabase_RLSManagement(t *testing.T) {
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

	// Create a test table first
	tableName := "test_rls_table_" + randomString(8)
	createTableBody := map[string]interface{}{
		"schema": "public",
		"name":   tableName,
		"columns": []map[string]interface{}{
			{
				"name":           "id",
				"type":           "uuid",
				"is_primary_key": true,
				"default_value":  "gen_random_uuid()",
			},
			{
				"name":        "user_id",
				"type":        "uuid",
				"is_nullable": false,
			},
		},
	}
	rr := makeRequest(t, handler, "POST", "/api/database/tables", createTableBody)
	if rr.Code != http.StatusCreated {
		t.Fatalf("failed to create test table: %s", rr.Body.String())
	}

	t.Run("enable RLS", func(t *testing.T) {
		rr := makeRequest(t, handler, "POST", "/api/database/tables/public/"+tableName+"/rls/enable", nil)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("create policy", func(t *testing.T) {
		body := map[string]interface{}{
			"name":       "user_select_policy",
			"schema":     "public",
			"table":      tableName,
			"command":    "SELECT",
			"definition": "true",
		}

		rr := makeRequest(t, handler, "POST", "/api/database/policies", body)

		if rr.Code != http.StatusCreated {
			t.Errorf("expected status 201, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("list policies", func(t *testing.T) {
		rr := makeRequest(t, handler, "GET", "/api/database/policies/public/"+tableName, nil)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("drop policy", func(t *testing.T) {
		rr := makeRequest(t, handler, "DELETE", "/api/database/policies/public/"+tableName+"/user_select_policy", nil)

		if rr.Code != http.StatusNoContent {
			t.Errorf("expected status 204, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("disable RLS", func(t *testing.T) {
		rr := makeRequest(t, handler, "POST", "/api/database/tables/public/"+tableName+"/rls/disable", nil)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	// Cleanup: drop the test table
	makeRequest(t, handler, "DELETE", "/api/database/tables/public/"+tableName, nil)
}

func TestDatabase_Schemas(t *testing.T) {
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

	t.Run("list schemas", func(t *testing.T) {
		rr := makeRequest(t, handler, "GET", "/api/database/schemas", nil)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
		}

		var schemas []string
		if err := json.Unmarshal(rr.Body.Bytes(), &schemas); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		// Should have at least public schema
		found := false
		for _, s := range schemas {
			if s == "public" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected public schema in list")
		}

		t.Logf("Found %d schemas", len(schemas))
	})
}

func TestDatabase_Extensions(t *testing.T) {
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

	t.Run("list extensions", func(t *testing.T) {
		rr := makeRequest(t, handler, "GET", "/api/database/extensions", nil)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})
}

func TestDatabase_BulkOperations(t *testing.T) {
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

	// Create a test table
	tableName := "test_bulk_" + randomString(8)
	createTableBody := map[string]interface{}{
		"schema": "public",
		"name":   tableName,
		"columns": []map[string]interface{}{
			{
				"name":           "id",
				"type":           "serial",
				"is_primary_key": true,
			},
			{
				"name":        "name",
				"type":        "text",
				"is_nullable": false,
			},
		},
	}
	rr := makeRequest(t, handler, "POST", "/api/database/tables", createTableBody)
	if rr.Code != http.StatusCreated {
		t.Fatalf("failed to create test table: %s", rr.Body.String())
	}

	// Insert some test data using REST API
	for i := 0; i < 5; i++ {
		makeRequest(t, handler, "POST", "/rest/v1/"+tableName, map[string]interface{}{
			"name": "Test Item",
		})
	}

	t.Run("bulk delete", func(t *testing.T) {
		body := map[string]interface{}{
			"operation": "delete",
			"ids":       []int{1, 2, 3},
			"column":    "id",
		}

		rr := makeRequest(t, handler, "POST", "/api/database/tables/public/"+tableName+"/bulk", body)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
		}

		var result map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if result["operation"] != "delete" {
			t.Errorf("expected operation 'delete', got '%v'", result["operation"])
		}
	})

	// Cleanup
	makeRequest(t, handler, "DELETE", "/api/database/tables/public/"+tableName, nil)
}

func TestDatabase_Export(t *testing.T) {
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

	// Create a test table with some data
	tableName := "test_export_" + randomString(8)
	createTableBody := map[string]interface{}{
		"schema": "public",
		"name":   tableName,
		"columns": []map[string]interface{}{
			{
				"name":           "id",
				"type":           "serial",
				"is_primary_key": true,
			},
			{
				"name":        "name",
				"type":        "text",
				"is_nullable": false,
			},
		},
	}
	rr := makeRequest(t, handler, "POST", "/api/database/tables", createTableBody)
	if rr.Code != http.StatusCreated {
		t.Fatalf("failed to create test table: %s", rr.Body.String())
	}

	// Insert test data
	makeRequest(t, handler, "POST", "/rest/v1/"+tableName, map[string]interface{}{
		"name": "Export Test 1",
	})
	makeRequest(t, handler, "POST", "/rest/v1/"+tableName, map[string]interface{}{
		"name": "Export Test 2",
	})

	t.Run("export as JSON", func(t *testing.T) {
		rr := makeRequest(t, handler, "GET", "/api/database/tables/public/"+tableName+"/export?format=json", nil)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
		}

		contentType := rr.Header().Get("Content-Type")
		if contentType != "application/json; charset=utf-8" {
			t.Errorf("expected JSON content type, got %s", contentType)
		}
	})

	t.Run("export as CSV", func(t *testing.T) {
		rr := makeRequest(t, handler, "GET", "/api/database/tables/public/"+tableName+"/export?format=csv", nil)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
		}

		contentType := rr.Header().Get("Content-Type")
		if contentType != "text/csv; charset=utf-8" {
			t.Errorf("expected CSV content type, got %s", contentType)
		}
	})

	t.Run("export as SQL", func(t *testing.T) {
		rr := makeRequest(t, handler, "GET", "/api/database/tables/public/"+tableName+"/export?format=sql", nil)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
		}

		contentType := rr.Header().Get("Content-Type")
		if contentType != "text/plain; charset=utf-8" {
			t.Errorf("expected SQL content type, got %s", contentType)
		}
	})

	// Cleanup
	makeRequest(t, handler, "DELETE", "/api/database/tables/public/"+tableName, nil)
}

// randomString generates a random string for test table names
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[i%len(letters)]
	}
	return string(b)
}
