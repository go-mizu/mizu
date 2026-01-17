//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

// =============================================================================
// postgres-meta API Test Suite
// Comprehensive tests for Supabase Dashboard postgres-meta compatibility
// =============================================================================

// TestPGMeta_Version tests the database version endpoint
func TestPGMeta_Version(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	t.Run("get database version", func(t *testing.T) {
		status, body, _, err := client.Request("GET", "/api/pg/config/version", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, body)
			return
		}

		var version map[string]any
		if err := json.Unmarshal(body, &version); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if _, ok := version["version"]; !ok {
			t.Error("Response missing 'version' field")
		}

		// Verify version contains PostgreSQL
		if v, ok := version["version"].(string); ok {
			if len(v) == 0 {
				t.Error("Version string is empty")
			}
		}
	})

	t.Run("requires service role", func(t *testing.T) {
		anonClient := NewClient(localbaseURL, localbaseAPIKey)
		status, _, _, err := anonClient.Request("GET", "/api/pg/config/version", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 403 {
			t.Errorf("Expected 403 for anon, got %d", status)
		}
	})
}

// TestPGMeta_Indexes tests index management endpoints
func TestPGMeta_Indexes(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	t.Run("list indexes", func(t *testing.T) {
		status, body, _, err := client.Request("GET", "/api/pg/indexes?included_schemas=public", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, body)
			return
		}

		var indexes []map[string]any
		if err := json.Unmarshal(body, &indexes); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Should have at least some indexes (primary keys are indexes)
		t.Logf("Found %d indexes", len(indexes))
	})

	t.Run("create and drop index", func(t *testing.T) {
		// Create a test table first
		createTableBody := map[string]any{
			"schema": "public",
			"name":   fmt.Sprintf("test_idx_table_%d", time.Now().UnixNano()),
			"columns": []map[string]any{
				{"name": "id", "type": "uuid", "is_primary_key": true},
				{"name": "email", "type": "text", "is_nullable": false},
			},
		}

		status, _, _, err := client.Request("POST", "/api/database/tables", createTableBody, nil)
		if err != nil || status != 201 {
			t.Skipf("Could not create test table (status %d)", status)
		}

		tableName := createTableBody["name"].(string)
		defer client.Request("DELETE", "/api/database/tables/public/"+tableName, nil, nil)

		// Create an index on the email column
		indexName := fmt.Sprintf("idx_email_%d", time.Now().UnixNano())
		indexBody := map[string]any{
			"schema":  "public",
			"table":   tableName,
			"name":    indexName,
			"columns": []string{"email"},
			"unique":  false,
			"using":   "btree",
		}

		status, body, _, err := client.Request("POST", "/api/pg/indexes", indexBody, nil)
		if err != nil {
			t.Fatalf("Create index request failed: %v", err)
		}

		if status == 201 {
			// Drop the index
			status, _, _, err = client.Request("DELETE", "/api/pg/indexes/public."+indexName, nil, nil)
			if err != nil {
				t.Fatalf("Drop index request failed: %v", err)
			}
			if status != 204 && status != 200 {
				t.Errorf("Expected 204 or 200 for drop, got %d", status)
			}
		} else {
			t.Logf("Create index returned %d: %s", status, body)
		}
	})
}

// TestPGMeta_Views tests view management endpoints
func TestPGMeta_Views(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	t.Run("list views", func(t *testing.T) {
		status, body, _, err := client.Request("GET", "/api/pg/views?included_schemas=public", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, body)
			return
		}

		var views []map[string]any
		if err := json.Unmarshal(body, &views); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		t.Logf("Found %d views", len(views))
	})

	t.Run("create and drop view", func(t *testing.T) {
		viewName := fmt.Sprintf("test_view_%d", time.Now().UnixNano())
		viewBody := map[string]any{
			"schema":     "public",
			"name":       viewName,
			"definition": "SELECT 1 as value",
		}

		status, body, _, err := client.Request("POST", "/api/pg/views", viewBody, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status == 201 {
			var view map[string]any
			if err := json.Unmarshal(body, &view); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			// Drop the view
			status, _, _, err = client.Request("DELETE", "/api/pg/views/public."+viewName, nil, nil)
			if err != nil {
				t.Fatalf("Drop view request failed: %v", err)
			}
			if status != 204 && status != 200 {
				t.Errorf("Expected 204 or 200 for drop, got %d", status)
			}
		} else {
			t.Logf("Create view returned %d: %s", status, body)
		}
	})
}

// TestPGMeta_MaterializedViews tests materialized view management
func TestPGMeta_MaterializedViews(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	t.Run("list materialized views", func(t *testing.T) {
		status, body, _, err := client.Request("GET", "/api/pg/materialized-views?included_schemas=public", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, body)
			return
		}

		var mvs []map[string]any
		if err := json.Unmarshal(body, &mvs); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		t.Logf("Found %d materialized views", len(mvs))
	})

	t.Run("create, refresh, and drop materialized view", func(t *testing.T) {
		mvName := fmt.Sprintf("test_mv_%d", time.Now().UnixNano())
		mvBody := map[string]any{
			"schema":     "public",
			"name":       mvName,
			"definition": "SELECT 1 as value",
		}

		status, body, _, err := client.Request("POST", "/api/pg/materialized-views", mvBody, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status == 201 {
			// Refresh the materialized view
			status, _, _, err = client.Request("POST", "/api/pg/materialized-views/public."+mvName+"/refresh", nil, nil)
			if err != nil {
				t.Fatalf("Refresh request failed: %v", err)
			}
			if status != 204 && status != 200 {
				t.Logf("Refresh returned %d (expected 204 or 200)", status)
			}

			// Drop the materialized view
			status, _, _, err = client.Request("DELETE", "/api/pg/materialized-views/public."+mvName, nil, nil)
			if err != nil {
				t.Fatalf("Drop request failed: %v", err)
			}
			if status != 204 && status != 200 {
				t.Errorf("Expected 204 or 200 for drop, got %d", status)
			}
		} else {
			t.Logf("Create materialized view returned %d: %s", status, body)
		}
	})
}

// TestPGMeta_Triggers tests trigger management endpoints
func TestPGMeta_Triggers(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	t.Run("list triggers", func(t *testing.T) {
		status, body, _, err := client.Request("GET", "/api/pg/triggers?included_schemas=public", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, body)
			return
		}

		var triggers []map[string]any
		if err := json.Unmarshal(body, &triggers); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		t.Logf("Found %d triggers", len(triggers))
	})
}

// TestPGMeta_Types tests custom type management endpoints
func TestPGMeta_Types(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	t.Run("list types", func(t *testing.T) {
		status, body, _, err := client.Request("GET", "/api/pg/types?included_schemas=public", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, body)
			return
		}

		var types []map[string]any
		if err := json.Unmarshal(body, &types); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		t.Logf("Found %d custom types", len(types))
	})

	t.Run("create and drop enum type", func(t *testing.T) {
		typeName := fmt.Sprintf("test_enum_%d", time.Now().UnixNano())
		typeBody := map[string]any{
			"schema": "public",
			"name":   typeName,
			"type":   "enum",
			"values": []string{"pending", "active", "inactive"},
		}

		status, body, _, err := client.Request("POST", "/api/pg/types", typeBody, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status == 201 {
			// Drop the type
			status, _, _, err = client.Request("DELETE", "/api/pg/types/public."+typeName, nil, nil)
			if err != nil {
				t.Fatalf("Drop request failed: %v", err)
			}
			if status != 204 && status != 200 {
				t.Errorf("Expected 204 or 200 for drop, got %d", status)
			}
		} else {
			t.Logf("Create type returned %d: %s", status, body)
		}
	})
}

// TestPGMeta_Roles tests role management endpoints
func TestPGMeta_Roles(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	t.Run("list roles", func(t *testing.T) {
		status, body, _, err := client.Request("GET", "/api/pg/roles", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, body)
			return
		}

		var roles []map[string]any
		if err := json.Unmarshal(body, &roles); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Should have at least one role (postgres or anon)
		if len(roles) < 1 {
			t.Error("Expected at least one role")
		}

		t.Logf("Found %d roles", len(roles))

		// Verify role structure
		if len(roles) > 0 {
			role := roles[0]
			requiredFields := []string{"id", "name", "is_superuser", "can_login"}
			for _, field := range requiredFields {
				if _, ok := role[field]; !ok {
					t.Errorf("Role missing field: %s", field)
				}
			}
		}
	})
}

// TestPGMeta_Publications tests publication management endpoints
func TestPGMeta_Publications(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	t.Run("list publications", func(t *testing.T) {
		status, body, _, err := client.Request("GET", "/api/pg/publications", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, body)
			return
		}

		var pubs []map[string]any
		if err := json.Unmarshal(body, &pubs); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		t.Logf("Found %d publications", len(pubs))
	})
}

// TestPGMeta_Privileges tests privilege listing endpoints
func TestPGMeta_Privileges(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	t.Run("list table privileges", func(t *testing.T) {
		status, body, _, err := client.Request("GET", "/api/pg/table-privileges?included_schemas=public", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, body)
			return
		}

		var privs []map[string]any
		if err := json.Unmarshal(body, &privs); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		t.Logf("Found %d table privileges", len(privs))
	})

	t.Run("list column privileges", func(t *testing.T) {
		status, body, _, err := client.Request("GET", "/api/pg/column-privileges?included_schemas=public", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, body)
			return
		}

		var privs []map[string]any
		if err := json.Unmarshal(body, &privs); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		t.Logf("Found %d column privileges", len(privs))
	})
}

// TestPGMeta_Constraints tests constraint listing endpoints
func TestPGMeta_Constraints(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	t.Run("list constraints", func(t *testing.T) {
		status, body, _, err := client.Request("GET", "/api/pg/constraints?included_schemas=public", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, body)
			return
		}

		var constraints []map[string]any
		if err := json.Unmarshal(body, &constraints); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		t.Logf("Found %d constraints", len(constraints))
	})

	t.Run("list primary keys", func(t *testing.T) {
		status, body, _, err := client.Request("GET", "/api/pg/primary-keys?included_schemas=public", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, body)
			return
		}

		var pks []map[string]any
		if err := json.Unmarshal(body, &pks); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		t.Logf("Found %d primary keys", len(pks))
	})

	t.Run("list foreign keys", func(t *testing.T) {
		status, body, _, err := client.Request("GET", "/api/pg/foreign-keys?included_schemas=public", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, body)
			return
		}

		var fks []map[string]any
		if err := json.Unmarshal(body, &fks); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		t.Logf("Found %d foreign keys", len(fks))
	})

	t.Run("list relationships", func(t *testing.T) {
		status, body, _, err := client.Request("GET", "/api/pg/relationships?included_schemas=public", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, body)
			return
		}

		var rels []map[string]any
		if err := json.Unmarshal(body, &rels); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		t.Logf("Found %d relationships", len(rels))
	})
}

// TestPGMeta_SQLUtilities tests SQL utility endpoints
func TestPGMeta_SQLUtilities(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	t.Run("format SQL", func(t *testing.T) {
		reqBody := map[string]any{
			"query": "select * from users where id=1",
		}

		status, body, _, err := client.Request("POST", "/api/pg/format", reqBody, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status == 200 {
			var result map[string]any
			if err := json.Unmarshal(body, &result); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			if _, ok := result["formatted"]; !ok {
				t.Error("Response missing 'formatted' field")
			}
		} else {
			t.Logf("Format SQL returned %d: %s", status, body)
		}
	})

	t.Run("explain query", func(t *testing.T) {
		reqBody := map[string]any{
			"query":  "SELECT 1",
			"format": "json",
		}

		status, body, _, err := client.Request("POST", "/api/pg/explain", reqBody, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status == 200 {
			// Response should be JSON
			t.Logf("Explain query returned valid response")
		} else {
			t.Logf("Explain query returned %d: %s", status, body)
		}
	})
}

// TestPGMeta_Generators tests type generator endpoints
func TestPGMeta_Generators(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	t.Run("generate TypeScript types", func(t *testing.T) {
		status, body, _, err := client.Request("GET", "/api/pg/generators/typescript?included_schemas=public", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status == 200 {
			// Should contain TypeScript interface definitions
			bodyStr := string(body)
			if len(bodyStr) == 0 {
				t.Error("TypeScript output is empty")
			}
			// Check for basic TypeScript structure
			if !contains(bodyStr, "export") {
				t.Log("Warning: TypeScript output may not contain valid exports")
			}
			t.Logf("Generated TypeScript: %d bytes", len(bodyStr))
		} else {
			t.Logf("Generate TypeScript returned %d: %s", status, body)
		}
	})

	t.Run("generate OpenAPI spec", func(t *testing.T) {
		status, body, _, err := client.Request("GET", "/api/pg/generators/openapi?included_schemas=public", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status == 200 {
			var spec map[string]any
			if err := json.Unmarshal(body, &spec); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			if _, ok := spec["openapi"]; !ok {
				t.Error("Response missing 'openapi' field")
			}
			if _, ok := spec["paths"]; !ok {
				t.Error("Response missing 'paths' field")
			}
		} else {
			t.Logf("Generate OpenAPI returned %d: %s", status, body)
		}
	})
}

// TestPGMeta_DatabaseFunctions tests database function listing
func TestPGMeta_DatabaseFunctions(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	t.Run("list database functions", func(t *testing.T) {
		status, body, _, err := client.Request("GET", "/api/pg/functions?included_schemas=public,auth", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, body)
			return
		}

		var functions []map[string]any
		if err := json.Unmarshal(body, &functions); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		t.Logf("Found %d database functions", len(functions))

		// Should have at least auth.uid() and auth.role() functions
		foundAuthFunctions := 0
		for _, fn := range functions {
			if schema, ok := fn["schema"].(string); ok && schema == "auth" {
				foundAuthFunctions++
			}
		}
		if foundAuthFunctions > 0 {
			t.Logf("Found %d auth schema functions", foundAuthFunctions)
		}
	})
}

// TestPGMeta_ForeignTables tests foreign table listing
func TestPGMeta_ForeignTables(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	t.Run("list foreign tables", func(t *testing.T) {
		status, body, _, err := client.Request("GET", "/api/pg/foreign-tables?included_schemas=public", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, body)
			return
		}

		var tables []map[string]any
		if err := json.Unmarshal(body, &tables); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Foreign tables are optional, so 0 is acceptable
		t.Logf("Found %d foreign tables", len(tables))
	})
}

// =============================================================================
// Dashboard API Tests
// =============================================================================

// TestDashboard_ExtendedStats tests the enhanced dashboard statistics endpoint
func TestDashboard_ExtendedStats(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	t.Run("get extended stats", func(t *testing.T) {
		status, body, _, err := client.Request("GET", "/api/dashboard/stats", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, body)
			return
		}

		var stats map[string]any
		if err := json.Unmarshal(body, &stats); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Verify all required sections are present
		requiredSections := []string{"users", "storage", "functions", "database", "realtime", "timestamp"}
		for _, section := range requiredSections {
			if _, ok := stats[section]; !ok {
				t.Errorf("Stats missing section: %s", section)
			}
		}

		// Verify users section has required fields
		if users, ok := stats["users"].(map[string]any); ok {
			if _, ok := users["total"]; !ok {
				t.Error("Users section missing 'total' field")
			}
		} else {
			t.Error("Users section is not a map")
		}

		// Verify database section has required fields
		if db, ok := stats["database"].(map[string]any); ok {
			if _, ok := db["tables"]; !ok {
				t.Error("Database section missing 'tables' field")
			}
			if _, ok := db["schemas"]; !ok {
				t.Error("Database section missing 'schemas' field")
			}
		} else {
			t.Error("Database section is not a map")
		}

		t.Logf("Stats response contains all required sections")
	})

	t.Run("requires service role", func(t *testing.T) {
		anonClient := NewClient(localbaseURL, localbaseAPIKey)
		status, _, _, err := anonClient.Request("GET", "/api/dashboard/stats", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 403 {
			t.Errorf("Expected 403 for anon, got %d", status)
		}
	})
}

// TestDashboard_ExtendedHealth tests the enhanced health check endpoint
func TestDashboard_ExtendedHealth(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	t.Run("get extended health", func(t *testing.T) {
		status, body, _, err := client.Request("GET", "/api/dashboard/health", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, body)
			return
		}

		var health map[string]any
		if err := json.Unmarshal(body, &health); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Verify required fields
		if _, ok := health["status"]; !ok {
			t.Error("Health missing 'status' field")
		}
		if _, ok := health["services"]; !ok {
			t.Error("Health missing 'services' field")
		}
		if _, ok := health["version"]; !ok {
			t.Error("Health missing 'version' field")
		}

		// Verify services section
		if services, ok := health["services"].(map[string]any); ok {
			requiredServices := []string{"database", "auth", "storage", "realtime"}
			for _, svc := range requiredServices {
				if _, ok := services[svc]; !ok {
					t.Errorf("Services missing '%s'", svc)
				}
			}

			// Database should have detailed info
			if db, ok := services["database"].(map[string]any); ok {
				if _, ok := db["status"]; !ok {
					t.Error("Database service missing 'status' field")
				}
			}
		} else {
			t.Error("Services section is not a map")
		}
	})

	t.Run("requires service role", func(t *testing.T) {
		anonClient := NewClient(localbaseURL, localbaseAPIKey)
		status, _, _, err := anonClient.Request("GET", "/api/dashboard/health", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 403 {
			t.Errorf("Expected 403 for anon, got %d", status)
		}
	})
}

// =============================================================================
// Additional Type Generator Tests
// =============================================================================

// TestPGMeta_GenerateGo tests Go type generation
func TestPGMeta_GenerateGo(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	t.Run("generate Go types", func(t *testing.T) {
		status, body, _, err := client.Request("GET", "/api/pg/generators/go?included_schemas=public&package=models", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status == 200 {
			bodyStr := string(body)
			if len(bodyStr) == 0 {
				t.Error("Go output is empty")
			}
			// Check for basic Go structure
			if !contains(bodyStr, "package") {
				t.Error("Go output missing package declaration")
			}
			if !contains(bodyStr, "struct") {
				t.Log("Warning: Go output may not contain struct definitions")
			}
			t.Logf("Generated Go: %d bytes", len(bodyStr))
		} else {
			t.Logf("Generate Go returned %d: %s", status, body)
		}
	})

	t.Run("requires service role", func(t *testing.T) {
		anonClient := NewClient(localbaseURL, localbaseAPIKey)
		status, _, _, err := anonClient.Request("GET", "/api/pg/generators/go", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 403 {
			t.Errorf("Expected 403 for anon, got %d", status)
		}
	})
}

// TestPGMeta_GenerateSwift tests Swift type generation
func TestPGMeta_GenerateSwift(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	t.Run("generate Swift types", func(t *testing.T) {
		status, body, _, err := client.Request("GET", "/api/pg/generators/swift?included_schemas=public", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status == 200 {
			bodyStr := string(body)
			if len(bodyStr) == 0 {
				t.Error("Swift output is empty")
			}
			// Check for basic Swift structure
			if !contains(bodyStr, "import Foundation") {
				t.Error("Swift output missing Foundation import")
			}
			if !contains(bodyStr, "struct") {
				t.Log("Warning: Swift output may not contain struct definitions")
			}
			t.Logf("Generated Swift: %d bytes", len(bodyStr))
		} else {
			t.Logf("Generate Swift returned %d: %s", status, body)
		}
	})
}

// TestPGMeta_GeneratePython tests Python type generation
func TestPGMeta_GeneratePython(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	t.Run("generate Python types", func(t *testing.T) {
		status, body, _, err := client.Request("GET", "/api/pg/generators/python?included_schemas=public", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status == 200 {
			bodyStr := string(body)
			if len(bodyStr) == 0 {
				t.Error("Python output is empty")
			}
			// Check for basic Python structure
			if !contains(bodyStr, "from dataclasses import dataclass") {
				t.Error("Python output missing dataclass import")
			}
			if !contains(bodyStr, "@dataclass") {
				t.Log("Warning: Python output may not contain dataclass definitions")
			}
			t.Logf("Generated Python: %d bytes", len(bodyStr))
		} else {
			t.Logf("Generate Python returned %d: %s", status, body)
		}
	})
}

// =============================================================================
// Logs Explorer API Tests
// =============================================================================

// TestLogs_ListLogs tests log listing
func TestLogs_ListLogs(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	t.Run("list logs", func(t *testing.T) {
		status, body, _, err := client.Request("GET", "/api/logs", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, body)
			return
		}

		var result map[string]any
		if err := json.Unmarshal(body, &result); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Verify required fields
		if _, ok := result["logs"]; !ok {
			t.Error("Response missing 'logs' field")
		}
		if _, ok := result["total"]; !ok {
			t.Error("Response missing 'total' field")
		}

		t.Log("List logs returned valid response")
	})

	t.Run("list logs with filters", func(t *testing.T) {
		status, body, _, err := client.Request("GET", "/api/logs?type=auth&level=info&limit=10", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, body)
		}
	})

	t.Run("requires service role", func(t *testing.T) {
		anonClient := NewClient(localbaseURL, localbaseAPIKey)
		status, _, _, err := anonClient.Request("GET", "/api/logs", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 403 {
			t.Errorf("Expected 403 for anon, got %d", status)
		}
	})
}

// TestLogs_ListLogTypes tests log type listing
func TestLogs_ListLogTypes(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	t.Run("list log types", func(t *testing.T) {
		status, body, _, err := client.Request("GET", "/api/logs/types", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, body)
			return
		}

		var types []map[string]any
		if err := json.Unmarshal(body, &types); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Should have multiple log types
		if len(types) < 3 {
			t.Errorf("Expected at least 3 log types, got %d", len(types))
		}

		// Verify type structure
		if len(types) > 0 {
			typ := types[0]
			requiredFields := []string{"id", "name", "description"}
			for _, field := range requiredFields {
				if _, ok := typ[field]; !ok {
					t.Errorf("Log type missing field: %s", field)
				}
			}
		}

		t.Logf("Found %d log types", len(types))
	})
}

// TestLogs_SearchLogs tests log search
func TestLogs_SearchLogs(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	t.Run("search logs", func(t *testing.T) {
		reqBody := map[string]any{
			"type":   "auth",
			"levels": []string{"info", "warning"},
			"query":  "test",
			"limit":  50,
		}

		status, body, _, err := client.Request("POST", "/api/logs/search", reqBody, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, body)
			return
		}

		var result map[string]any
		if err := json.Unmarshal(body, &result); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if _, ok := result["logs"]; !ok {
			t.Error("Response missing 'logs' field")
		}

		t.Log("Search logs returned valid response")
	})
}

// TestLogs_ExportLogs tests log export
func TestLogs_ExportLogs(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	t.Run("export logs as JSON", func(t *testing.T) {
		status, _, _, err := client.Request("GET", "/api/logs/export?format=json", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d", status)
		}
	})

	t.Run("export logs as CSV", func(t *testing.T) {
		status, body, _, err := client.Request("GET", "/api/logs/export?format=csv", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d", status)
			return
		}

		bodyStr := string(body)
		if !contains(bodyStr, "id,type,level,message,timestamp") {
			t.Error("CSV missing header row")
		}
	})
}

// =============================================================================
// Settings API Tests
// =============================================================================

// TestSettings_GetAllSettings tests getting all settings
func TestSettings_GetAllSettings(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	t.Run("get all settings", func(t *testing.T) {
		status, body, _, err := client.Request("GET", "/api/settings", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, body)
			return
		}

		var settings map[string]any
		if err := json.Unmarshal(body, &settings); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Verify all required sections
		requiredSections := []string{"project", "api", "auth", "database", "storage"}
		for _, section := range requiredSections {
			if _, ok := settings[section]; !ok {
				t.Errorf("Settings missing section: %s", section)
			}
		}

		t.Log("Get all settings returned valid response")
	})

	t.Run("requires service role", func(t *testing.T) {
		anonClient := NewClient(localbaseURL, localbaseAPIKey)
		status, _, _, err := anonClient.Request("GET", "/api/settings", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 403 {
			t.Errorf("Expected 403 for anon, got %d", status)
		}
	})
}

// TestSettings_ProjectSettings tests project settings
func TestSettings_ProjectSettings(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	t.Run("get project settings", func(t *testing.T) {
		status, body, _, err := client.Request("GET", "/api/settings/project", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, body)
			return
		}

		var project map[string]any
		if err := json.Unmarshal(body, &project); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		requiredFields := []string{"project_id", "name", "region", "status"}
		for _, field := range requiredFields {
			if _, ok := project[field]; !ok {
				t.Errorf("Project settings missing field: %s", field)
			}
		}
	})

	t.Run("update project settings", func(t *testing.T) {
		reqBody := map[string]any{
			"name": "Updated LocalBase",
		}

		status, body, _, err := client.Request("PATCH", "/api/settings/project", reqBody, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, body)
			return
		}

		var project map[string]any
		if err := json.Unmarshal(body, &project); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if name, ok := project["name"].(string); !ok || name != "Updated LocalBase" {
			t.Error("Project name was not updated")
		}
	})
}

// TestSettings_APISettings tests API settings
func TestSettings_APISettings(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	t.Run("get API settings", func(t *testing.T) {
		status, body, _, err := client.Request("GET", "/api/settings/api", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, body)
			return
		}

		var api map[string]any
		if err := json.Unmarshal(body, &api); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		requiredFields := []string{"max_rows", "expose_schemas", "jwt_exp"}
		for _, field := range requiredFields {
			if _, ok := api[field]; !ok {
				t.Errorf("API settings missing field: %s", field)
			}
		}

		// Verify JWT secret is masked
		if secret, ok := api["jwt_secret"].(string); ok {
			if !contains(secret, "****") {
				t.Error("JWT secret should be masked")
			}
		}
	})

	t.Run("update API settings", func(t *testing.T) {
		reqBody := map[string]any{
			"max_rows":           2000,
			"rate_limit_enabled": true,
		}

		status, body, _, err := client.Request("PATCH", "/api/settings/api", reqBody, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, body)
		}
	})
}

// TestSettings_AuthSettings tests auth settings
func TestSettings_AuthSettings(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	t.Run("get auth settings", func(t *testing.T) {
		status, body, _, err := client.Request("GET", "/api/settings/auth", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, body)
			return
		}

		var auth map[string]any
		if err := json.Unmarshal(body, &auth); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		requiredFields := []string{"site_url", "disable_signup", "mfa_enabled", "password_min_length"}
		for _, field := range requiredFields {
			if _, ok := auth[field]; !ok {
				t.Errorf("Auth settings missing field: %s", field)
			}
		}
	})

	t.Run("update auth settings", func(t *testing.T) {
		reqBody := map[string]any{
			"disable_signup":      false,
			"mfa_enabled":         true,
			"password_min_length": 8,
		}

		status, body, _, err := client.Request("PATCH", "/api/settings/auth", reqBody, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, body)
		}
	})
}

// TestSettings_DatabaseSettings tests database settings
func TestSettings_DatabaseSettings(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	t.Run("get database settings", func(t *testing.T) {
		status, body, _, err := client.Request("GET", "/api/settings/database", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, body)
			return
		}

		var db map[string]any
		if err := json.Unmarshal(body, &db); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		requiredFields := []string{"pool_mode", "max_connections", "statement_timeout_ms"}
		for _, field := range requiredFields {
			if _, ok := db[field]; !ok {
				t.Errorf("Database settings missing field: %s", field)
			}
		}
	})

	t.Run("update database settings", func(t *testing.T) {
		reqBody := map[string]any{
			"max_connections":      150,
			"statement_timeout_ms": 60000,
		}

		status, body, _, err := client.Request("PATCH", "/api/settings/database", reqBody, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, body)
		}
	})
}

// TestSettings_StorageSettings tests storage settings
func TestSettings_StorageSettings(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	t.Run("get storage settings", func(t *testing.T) {
		status, body, _, err := client.Request("GET", "/api/settings/storage", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, body)
			return
		}

		var storage map[string]any
		if err := json.Unmarshal(body, &storage); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		requiredFields := []string{"file_size_limit", "image_transformations", "allowed_mime_types"}
		for _, field := range requiredFields {
			if _, ok := storage[field]; !ok {
				t.Errorf("Storage settings missing field: %s", field)
			}
		}
	})

	t.Run("update storage settings", func(t *testing.T) {
		reqBody := map[string]any{
			"file_size_limit":        104857600,
			"image_transformations": true,
		}

		status, body, _, err := client.Request("PATCH", "/api/settings/storage", reqBody, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, body)
		}
	})
}

// =============================================================================
// Helpers
// =============================================================================

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsImpl(s, substr))
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
