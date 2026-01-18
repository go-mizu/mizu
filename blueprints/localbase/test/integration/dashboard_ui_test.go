//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

// =============================================================================
// Dashboard UI Comprehensive Test Suite
// Tests for Supabase Dashboard API compatibility
// =============================================================================

// TestDashboardUI_TableEditor tests table editor functionality
func TestDashboardUI_TableEditor(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	t.Run("list tables in public schema", func(t *testing.T) {
		status, body, _, err := client.Request("GET", "/api/database/tables?schema=public", nil, nil)
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

		t.Logf("Found %d tables in public schema", len(tables))

		// Verify table structure
		if len(tables) > 0 {
			table := tables[0]
			requiredFields := []string{"name", "schema", "rls_enabled"}
			for _, field := range requiredFields {
				if _, ok := table[field]; !ok {
					t.Errorf("Table missing field: %s", field)
				}
			}
		}
	})

	t.Run("create and delete table", func(t *testing.T) {
		tableName := fmt.Sprintf("test_table_%d", time.Now().UnixNano())
		createBody := map[string]any{
			"schema": "public",
			"name":   tableName,
			"columns": []map[string]any{
				{"name": "id", "type": "uuid", "is_primary_key": true, "default_value": "gen_random_uuid()"},
				{"name": "name", "type": "text", "is_nullable": false},
				{"name": "created_at", "type": "timestamptz", "default_value": "now()"},
			},
		}

		status, _, _, err := client.Request("POST", "/api/database/tables", createBody, nil)
		if err != nil {
			t.Fatalf("Create request failed: %v", err)
		}

		if status != 201 {
			t.Errorf("Expected 201 for create, got %d", status)
			return
		}

		// Delete the table
		status, _, _, err = client.Request("DELETE", "/api/database/tables/public/"+tableName, nil, nil)
		if err != nil {
			t.Fatalf("Delete request failed: %v", err)
		}

		if status != 200 && status != 204 {
			t.Errorf("Expected 200 or 204 for delete, got %d", status)
		}
	})

	t.Run("list columns for table", func(t *testing.T) {
		// Use a seeded table
		status, body, _, err := client.Request("GET", "/api/database/tables/public/todos/columns", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Logf("Skipping column test - todos table may not exist: %d", status)
			return
		}

		var columns []map[string]any
		if err := json.Unmarshal(body, &columns); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		t.Logf("Found %d columns", len(columns))

		// Verify column structure
		if len(columns) > 0 {
			col := columns[0]
			requiredFields := []string{"name", "type", "is_nullable"}
			for _, field := range requiredFields {
				if _, ok := col[field]; !ok {
					t.Errorf("Column missing field: %s", field)
				}
			}
		}
	})
}

// TestDashboardUI_SQLEditor tests SQL editor functionality
func TestDashboardUI_SQLEditor(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	t.Run("execute SELECT query", func(t *testing.T) {
		reqBody := map[string]any{
			"query": "SELECT 1 as value, 'test' as name",
		}

		status, body, _, err := client.Request("POST", "/api/database/query", reqBody, nil)
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

		// Verify result structure
		if _, ok := result["rows"]; !ok {
			t.Error("Response missing 'rows' field")
		}
		if _, ok := result["columns"]; !ok {
			t.Error("Response missing 'columns' field")
		}
	})

	t.Run("execute multiple statements", func(t *testing.T) {
		reqBody := map[string]any{
			"query": "SELECT 1; SELECT 2;",
		}

		status, _, _, err := client.Request("POST", "/api/database/query", reqBody, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		// Multiple statements may succeed or fail depending on implementation
		t.Logf("Multiple statements returned status: %d", status)
	})

	t.Run("handle syntax error", func(t *testing.T) {
		reqBody := map[string]any{
			"query": "SELEKT invalid syntax",
		}

		status, _, _, err := client.Request("POST", "/api/database/query", reqBody, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		// Should return error status
		if status == 200 {
			t.Error("Expected error status for invalid SQL")
		}
	})
}

// TestDashboardUI_PolicyManagement tests RLS policy functionality
func TestDashboardUI_PolicyManagement(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	t.Run("list policies for table", func(t *testing.T) {
		status, body, _, err := client.Request("GET", "/api/database/policies/public/todos", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Logf("Policy list returned %d (table may not exist)", status)
			return
		}

		var policies []map[string]any
		if err := json.Unmarshal(body, &policies); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		t.Logf("Found %d policies", len(policies))
	})

	t.Run("create and delete policy", func(t *testing.T) {
		// First create a test table
		tableName := fmt.Sprintf("test_policy_table_%d", time.Now().UnixNano())
		createTableBody := map[string]any{
			"schema": "public",
			"name":   tableName,
			"columns": []map[string]any{
				{"name": "id", "type": "uuid", "is_primary_key": true},
				{"name": "user_id", "type": "uuid"},
			},
		}

		status, _, _, err := client.Request("POST", "/api/database/tables", createTableBody, nil)
		if err != nil || status != 201 {
			t.Skipf("Could not create test table: %d", status)
		}
		defer client.Request("DELETE", "/api/database/tables/public/"+tableName, nil, nil)

		// Create a policy
		policyName := "select_own"
		policyBody := map[string]any{
			"name":       policyName,
			"schema":     "public",
			"table":      tableName,
			"command":    "SELECT",
			"definition": "(auth.uid() = user_id)",
		}

		status, body, _, err := client.Request("POST", "/api/database/policies", policyBody, nil)
		if err != nil {
			t.Fatalf("Create policy request failed: %v", err)
		}

		if status == 201 || status == 200 {
			// Delete the policy
			status, _, _, err = client.Request("DELETE", fmt.Sprintf("/api/database/policies/public/%s/%s", tableName, policyName), nil, nil)
			if err != nil {
				t.Fatalf("Delete policy request failed: %v", err)
			}
			if status != 200 && status != 204 {
				t.Logf("Delete policy returned %d", status)
			}
		} else {
			t.Logf("Create policy returned %d: %s", status, body)
		}
	})
}

// TestDashboardUI_IndexManagement tests index functionality
func TestDashboardUI_IndexManagement(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	t.Run("list all indexes", func(t *testing.T) {
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

		t.Logf("Found %d indexes", len(indexes))
	})

	t.Run("create btree index", func(t *testing.T) {
		// Create a test table
		tableName := fmt.Sprintf("test_idx_%d", time.Now().UnixNano())
		createTableBody := map[string]any{
			"schema": "public",
			"name":   tableName,
			"columns": []map[string]any{
				{"name": "id", "type": "uuid", "is_primary_key": true},
				{"name": "email", "type": "text"},
			},
		}

		status, _, _, err := client.Request("POST", "/api/database/tables", createTableBody, nil)
		if err != nil || status != 201 {
			t.Skipf("Could not create test table: %d", status)
		}
		defer client.Request("DELETE", "/api/database/tables/public/"+tableName, nil, nil)

		// Create an index
		indexName := fmt.Sprintf("idx_%s_email", tableName)
		indexBody := map[string]any{
			"schema":  "public",
			"table":   tableName,
			"name":    indexName,
			"columns": []string{"email"},
			"using":   "btree",
		}

		status, body, _, err := client.Request("POST", "/api/pg/indexes", indexBody, nil)
		if err != nil {
			t.Fatalf("Create index request failed: %v", err)
		}

		if status == 201 {
			// Clean up
			client.Request("DELETE", "/api/pg/indexes/public."+indexName, nil, nil)
		} else {
			t.Logf("Create index returned %d: %s", status, body)
		}
	})
}

// TestDashboardUI_ViewManagement tests view functionality
func TestDashboardUI_ViewManagement(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	t.Run("list all views", func(t *testing.T) {
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
			"definition": "SELECT 1 as value, now() as created_at",
		}

		status, body, _, err := client.Request("POST", "/api/pg/views", viewBody, nil)
		if err != nil {
			t.Fatalf("Create view request failed: %v", err)
		}

		if status == 201 {
			// Drop the view
			status, _, _, err = client.Request("DELETE", "/api/pg/views/public."+viewName, nil, nil)
			if err != nil {
				t.Fatalf("Drop view request failed: %v", err)
			}
			if status != 200 && status != 204 {
				t.Logf("Drop view returned %d", status)
			}
		} else {
			t.Logf("Create view returned %d: %s", status, body)
		}
	})

	t.Run("create and refresh materialized view", func(t *testing.T) {
		mvName := fmt.Sprintf("test_mv_%d", time.Now().UnixNano())
		mvBody := map[string]any{
			"schema":     "public",
			"name":       mvName,
			"definition": "SELECT 1 as value",
		}

		status, body, _, err := client.Request("POST", "/api/pg/materialized-views", mvBody, nil)
		if err != nil {
			t.Fatalf("Create MV request failed: %v", err)
		}

		if status == 201 {
			// Refresh
			status, _, _, err = client.Request("POST", "/api/pg/materialized-views/public."+mvName+"/refresh", nil, nil)
			if err != nil {
				t.Logf("Refresh MV request failed: %v", err)
			}

			// Drop
			client.Request("DELETE", "/api/pg/materialized-views/public."+mvName, nil, nil)
		} else {
			t.Logf("Create MV returned %d: %s", status, body)
		}
	})
}

// TestDashboardUI_TriggerManagement tests trigger functionality
func TestDashboardUI_TriggerManagement(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	t.Run("list all triggers", func(t *testing.T) {
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

// TestDashboardUI_RoleManagement tests role functionality
func TestDashboardUI_RoleManagement(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	t.Run("list all roles", func(t *testing.T) {
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

		// Should have at least postgres role
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

	t.Run("create and delete role", func(t *testing.T) {
		roleName := fmt.Sprintf("test_role_%d", time.Now().UnixNano())
		roleBody := map[string]any{
			"name":      roleName,
			"can_login": true,
			"inherit":   true,
		}

		status, body, _, err := client.Request("POST", "/api/pg/roles", roleBody, nil)
		if err != nil {
			t.Fatalf("Create role request failed: %v", err)
		}

		if status == 201 || status == 200 {
			// Delete the role
			status, _, _, err = client.Request("DELETE", "/api/pg/roles/"+roleName, nil, nil)
			if err != nil {
				t.Fatalf("Delete role request failed: %v", err)
			}
			if status != 200 && status != 204 {
				t.Logf("Delete role returned %d", status)
			}
		} else {
			t.Logf("Create role returned %d: %s", status, body)
		}
	})
}

// TestDashboardUI_TypeManagement tests custom type functionality
func TestDashboardUI_TypeManagement(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	t.Run("list all types", func(t *testing.T) {
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

	t.Run("create and delete enum type", func(t *testing.T) {
		typeName := fmt.Sprintf("test_enum_%d", time.Now().UnixNano())
		typeBody := map[string]any{
			"schema": "public",
			"name":   typeName,
			"type":   "enum",
			"values": []string{"pending", "active", "completed"},
		}

		status, body, _, err := client.Request("POST", "/api/pg/types", typeBody, nil)
		if err != nil {
			t.Fatalf("Create type request failed: %v", err)
		}

		if status == 201 || status == 200 {
			// Delete the type
			status, _, _, err = client.Request("DELETE", "/api/pg/types/public."+typeName, nil, nil)
			if err != nil {
				t.Fatalf("Delete type request failed: %v", err)
			}
			if status != 200 && status != 204 {
				t.Logf("Delete type returned %d", status)
			}
		} else {
			t.Logf("Create type returned %d: %s", status, body)
		}
	})
}

// TestDashboardUI_LogsExplorer tests logs explorer functionality
func TestDashboardUI_LogsExplorer(t *testing.T) {
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

		t.Logf("Found %d log types", len(types))
	})

	t.Run("list logs with filters", func(t *testing.T) {
		status, body, _, err := client.Request("GET", "/api/logs?type=auth&level=info&limit=10", nil, nil)
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
		if _, ok := result["total"]; !ok {
			t.Error("Response missing 'total' field")
		}
	})

	t.Run("search logs", func(t *testing.T) {
		reqBody := map[string]any{
			"type":   "auth",
			"levels": []string{"info", "error"},
			"limit":  50,
		}

		status, body, _, err := client.Request("POST", "/api/logs/search", reqBody, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, body)
		}
	})

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

		// CSV should have header
		bodyStr := string(body)
		if len(bodyStr) > 0 && !containsString(bodyStr, "id") {
			t.Log("Warning: CSV may be missing expected headers")
		}
	})
}

// TestDashboardUI_PublicationManagement tests publication functionality
func TestDashboardUI_PublicationManagement(t *testing.T) {
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

// TestDashboardUI_PrivilegeManagement tests privilege listing
func TestDashboardUI_PrivilegeManagement(t *testing.T) {
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

// TestDashboardUI_ConstraintManagement tests constraint listing
func TestDashboardUI_ConstraintManagement(t *testing.T) {
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

// TestDashboardUI_DatabaseFunctions tests database function listing
func TestDashboardUI_DatabaseFunctions(t *testing.T) {
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

		// Verify function structure
		if len(functions) > 0 {
			fn := functions[0]
			requiredFields := []string{"schema", "name", "return_type"}
			for _, field := range requiredFields {
				if _, ok := fn[field]; !ok {
					t.Errorf("Function missing field: %s", field)
				}
			}
		}
	})
}

// TestDashboardUI_TypeGenerators tests type generation
func TestDashboardUI_TypeGenerators(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	t.Run("generate TypeScript types", func(t *testing.T) {
		status, body, _, err := client.Request("GET", "/api/pg/generators/typescript?included_schemas=public", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status == 200 {
			t.Logf("Generated TypeScript: %d bytes", len(body))
		} else {
			t.Logf("Generate TypeScript returned %d", status)
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
		} else {
			t.Logf("Generate OpenAPI returned %d", status)
		}
	})

	t.Run("generate Go types", func(t *testing.T) {
		status, body, _, err := client.Request("GET", "/api/pg/generators/go?included_schemas=public&package=models", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status == 200 {
			bodyStr := string(body)
			if !containsString(bodyStr, "package") {
				t.Error("Go output missing package declaration")
			}
			t.Logf("Generated Go: %d bytes", len(body))
		} else {
			t.Logf("Generate Go returned %d", status)
		}
	})

	t.Run("generate Swift types", func(t *testing.T) {
		status, body, _, err := client.Request("GET", "/api/pg/generators/swift?included_schemas=public", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status == 200 {
			t.Logf("Generated Swift: %d bytes", len(body))
		} else {
			t.Logf("Generate Swift returned %d", status)
		}
	})

	t.Run("generate Python types", func(t *testing.T) {
		status, body, _, err := client.Request("GET", "/api/pg/generators/python?included_schemas=public", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status == 200 {
			t.Logf("Generated Python: %d bytes", len(body))
		} else {
			t.Logf("Generate Python returned %d", status)
		}
	})
}

// TestDashboardUI_SQLUtilities tests SQL utility functions
func TestDashboardUI_SQLUtilities(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	t.Run("format SQL", func(t *testing.T) {
		reqBody := map[string]any{
			"query": "select*from users where id=1 and name='test'",
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
			t.Logf("Format SQL returned %d", status)
		}
	})

	t.Run("explain query", func(t *testing.T) {
		reqBody := map[string]any{
			"query":  "SELECT 1",
			"format": "json",
		}

		status, _, _, err := client.Request("POST", "/api/pg/explain", reqBody, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status == 200 {
			t.Log("Explain query succeeded")
		} else {
			t.Logf("Explain query returned %d", status)
		}
	})
}

// TestDashboardUI_SchemaOperations tests schema management
func TestDashboardUI_SchemaOperations(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	t.Run("list schemas", func(t *testing.T) {
		status, body, _, err := client.Request("GET", "/api/database/schemas", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, body)
			return
		}

		var schemas []string
		if err := json.Unmarshal(body, &schemas); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Should have at least public schema
		foundPublic := false
		for _, s := range schemas {
			if s == "public" {
				foundPublic = true
				break
			}
		}

		if !foundPublic {
			t.Error("Expected to find 'public' schema")
		}

		t.Logf("Found %d schemas", len(schemas))
	})

	t.Run("create and drop schema", func(t *testing.T) {
		schemaName := fmt.Sprintf("test_schema_%d", time.Now().UnixNano())

		status, _, _, err := client.Request("POST", "/api/database/schemas", map[string]any{"name": schemaName}, nil)
		if err != nil {
			t.Fatalf("Create schema request failed: %v", err)
		}

		if status == 201 || status == 200 {
			// Schema created, now drop it (would need a drop endpoint)
			t.Logf("Schema %s created successfully", schemaName)
		} else {
			t.Logf("Create schema returned %d", status)
		}
	})
}

// TestDashboardUI_ExtensionOperations tests extension management
func TestDashboardUI_ExtensionOperations(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	t.Run("list extensions", func(t *testing.T) {
		status, body, _, err := client.Request("GET", "/api/database/extensions", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, body)
			return
		}

		var extensions []map[string]any
		if err := json.Unmarshal(body, &extensions); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		t.Logf("Found %d extensions", len(extensions))

		// Verify extension structure
		if len(extensions) > 0 {
			ext := extensions[0]
			if _, ok := ext["name"]; !ok {
				t.Error("Extension missing 'name' field")
			}
		}
	})
}

// =============================================================================
// Security Tests
// =============================================================================

// TestDashboardUI_RequiresServiceRole verifies that dashboard endpoints require service_role
func TestDashboardUI_RequiresServiceRole(t *testing.T) {
	anonClient := NewClient(localbaseURL, localbaseAPIKey)

	endpoints := []string{
		"/api/pg/config/version",
		"/api/pg/indexes",
		"/api/pg/views",
		"/api/pg/triggers",
		"/api/pg/roles",
		"/api/pg/types",
		"/api/pg/publications",
		"/api/logs",
		"/api/logs/types",
		"/api/settings",
		"/api/dashboard/stats",
		"/api/dashboard/health",
	}

	for _, endpoint := range endpoints {
		t.Run(endpoint, func(t *testing.T) {
			status, _, _, err := anonClient.Request("GET", endpoint, nil, nil)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}

			if status != 403 {
				t.Errorf("Expected 403 for anon on %s, got %d", endpoint, status)
			}
		})
	}
}

// =============================================================================
// Enhanced Table Editor API Tests
// =============================================================================

// TestDashboardUI_TableEditorEnhanced tests the enhanced table editor API endpoints
func TestDashboardUI_TableEditorEnhanced(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	// Create a test table first
	tableName := fmt.Sprintf("test_editor_%d", time.Now().UnixNano())
	createTableBody := map[string]any{
		"schema": "public",
		"name":   tableName,
		"columns": []map[string]any{
			{"name": "id", "type": "uuid", "is_primary_key": true, "default_value": "gen_random_uuid()"},
			{"name": "name", "type": "text", "is_nullable": false},
			{"name": "email", "type": "text"},
			{"name": "age", "type": "integer"},
			{"name": "active", "type": "boolean", "default_value": "true"},
			{"name": "metadata", "type": "jsonb"},
			{"name": "created_at", "type": "timestamptz", "default_value": "now()"},
		},
	}

	status, _, _, err := client.Request("POST", "/api/database/tables", createTableBody, nil)
	if err != nil || status != 201 {
		t.Fatalf("Failed to create test table: status=%d, err=%v", status, err)
	}
	defer client.Request("DELETE", "/api/database/tables/public/"+tableName, nil, nil)

	// Insert some test data
	for i := 1; i <= 15; i++ {
		insertBody := map[string]any{
			"name":     fmt.Sprintf("User %d", i),
			"email":    fmt.Sprintf("user%d@test.com", i),
			"age":      20 + i,
			"active":   i%2 == 0,
			"metadata": map[string]any{"index": i},
		}
		_, _, _, _ = client.Request("POST", fmt.Sprintf("/rest/v1/%s", tableName), insertBody, map[string]string{
			"Prefer": "return=minimal",
		})
	}

	t.Run("get table data with pagination", func(t *testing.T) {
		status, body, headers, err := client.Request("GET", fmt.Sprintf("/api/database/tables/public/%s/data?limit=5&offset=0&count=true", tableName), nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, body)
			return
		}

		var rows []map[string]any
		if err := json.Unmarshal(body, &rows); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if len(rows) > 5 {
			t.Errorf("Expected at most 5 rows, got %d", len(rows))
		}

		// Check for total count header
		totalCount := headers.Get("X-Total-Count")
		t.Logf("Got %d rows, total count header: %s", len(rows), totalCount)
	})

	t.Run("get table data with sorting", func(t *testing.T) {
		status, body, _, err := client.Request("GET", fmt.Sprintf("/api/database/tables/public/%s/data?order=age.desc&limit=5", tableName), nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, body)
			return
		}

		var rows []map[string]any
		if err := json.Unmarshal(body, &rows); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Verify descending order
		if len(rows) >= 2 {
			age1, ok1 := rows[0]["age"].(float64)
			age2, ok2 := rows[1]["age"].(float64)
			if ok1 && ok2 && age1 < age2 {
				t.Error("Expected descending order by age")
			}
		}
		t.Logf("Got %d sorted rows", len(rows))
	})

	t.Run("get table data with filter", func(t *testing.T) {
		status, body, _, err := client.Request("GET", fmt.Sprintf("/api/database/tables/public/%s/data?active=eq.true", tableName), nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, body)
			return
		}

		var rows []map[string]any
		if err := json.Unmarshal(body, &rows); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Verify all rows have active=true
		for _, row := range rows {
			if active, ok := row["active"].(bool); ok && !active {
				t.Error("Expected all rows to have active=true")
			}
		}
		t.Logf("Got %d filtered rows with active=true", len(rows))
	})

	t.Run("export table data as JSON", func(t *testing.T) {
		status, body, headers, err := client.Request("GET", fmt.Sprintf("/api/database/tables/public/%s/export?format=json", tableName), nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, body)
			return
		}

		contentType := headers.Get("Content-Type")
		if !containsString(contentType, "application/json") {
			t.Errorf("Expected JSON content type, got %s", contentType)
		}

		contentDisposition := headers.Get("Content-Disposition")
		if !containsString(contentDisposition, "attachment") {
			t.Errorf("Expected attachment content-disposition, got %s", contentDisposition)
		}

		t.Logf("Export JSON content type: %s", contentType)
	})

	t.Run("export table data as CSV", func(t *testing.T) {
		status, body, headers, err := client.Request("GET", fmt.Sprintf("/api/database/tables/public/%s/export?format=csv", tableName), nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, body)
			return
		}

		contentType := headers.Get("Content-Type")
		if !containsString(contentType, "text/csv") {
			t.Errorf("Expected CSV content type, got %s", contentType)
		}

		// Check CSV has header row
		bodyStr := string(body)
		if !containsString(bodyStr, "id,") && !containsString(bodyStr, "name,") {
			t.Error("CSV should contain column headers")
		}

		t.Logf("Export CSV content type: %s, length: %d", contentType, len(body))
	})

	t.Run("export table data as SQL", func(t *testing.T) {
		status, body, headers, err := client.Request("GET", fmt.Sprintf("/api/database/tables/public/%s/export?format=sql", tableName), nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, body)
			return
		}

		contentType := headers.Get("Content-Type")
		if !containsString(contentType, "text/plain") {
			t.Errorf("Expected text/plain content type, got %s", contentType)
		}

		// Check SQL has INSERT statements
		bodyStr := string(body)
		if !containsString(bodyStr, "INSERT INTO") {
			t.Error("SQL export should contain INSERT statements")
		}

		t.Logf("Export SQL content type: %s, length: %d", contentType, len(body))
	})

	t.Run("bulk delete rows", func(t *testing.T) {
		// First get some IDs to delete
		status, body, _, err := client.Request("GET", fmt.Sprintf("/api/database/tables/public/%s/data?limit=2", tableName), nil, nil)
		if err != nil || status != 200 {
			t.Fatalf("Failed to get rows: %v", err)
		}

		var rows []map[string]any
		if err := json.Unmarshal(body, &rows); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if len(rows) < 2 {
			t.Skip("Not enough rows for bulk delete test")
		}

		ids := []any{rows[0]["id"], rows[1]["id"]}

		bulkBody := map[string]any{
			"operation": "delete",
			"ids":       ids,
			"column":    "id",
		}

		status, body, _, err = client.Request("POST", fmt.Sprintf("/api/database/tables/public/%s/bulk", tableName), bulkBody, nil)
		if err != nil {
			t.Fatalf("Bulk delete request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, body)
			return
		}

		var result map[string]any
		if err := json.Unmarshal(body, &result); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if result["operation"] != "delete" {
			t.Errorf("Expected operation=delete, got %v", result["operation"])
		}

		rowsAffected, _ := result["rows_affected"].(float64)
		if rowsAffected < 1 {
			t.Errorf("Expected at least 1 row affected, got %v", rowsAffected)
		}

		t.Logf("Bulk deleted %v rows", rowsAffected)
	})

	t.Run("bulk update rows", func(t *testing.T) {
		// First get some IDs to update
		status, body, _, err := client.Request("GET", fmt.Sprintf("/api/database/tables/public/%s/data?limit=3", tableName), nil, nil)
		if err != nil || status != 200 {
			t.Fatalf("Failed to get rows: %v", err)
		}

		var rows []map[string]any
		if err := json.Unmarshal(body, &rows); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if len(rows) < 2 {
			t.Skip("Not enough rows for bulk update test")
		}

		ids := make([]any, 0)
		for _, row := range rows[:2] {
			ids = append(ids, row["id"])
		}

		bulkBody := map[string]any{
			"operation": "update",
			"ids":       ids,
			"column":    "id",
			"data": map[string]any{
				"name": "Bulk Updated",
			},
		}

		status, body, _, err = client.Request("POST", fmt.Sprintf("/api/database/tables/public/%s/bulk", tableName), bulkBody, nil)
		if err != nil {
			t.Fatalf("Bulk update request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, body)
			return
		}

		var result map[string]any
		if err := json.Unmarshal(body, &result); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if result["operation"] != "update" {
			t.Errorf("Expected operation=update, got %v", result["operation"])
		}

		rowsAffected, _ := result["rows_affected"].(float64)
		t.Logf("Bulk updated %v rows", rowsAffected)
	})
}

// =============================================================================
// Helpers
// =============================================================================

func containsString(s, substr string) bool {
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
