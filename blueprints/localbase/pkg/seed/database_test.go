package seed

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

// Test configuration
var (
	// Supabase Local endpoints (runs on port 54421 to avoid conflict with Localbase)
	supabaseRESTURL = getEnv("SUPABASE_REST_URL", "http://127.0.0.1:54421/rest/v1")
	supabaseAPIKey  = getEnv("SUPABASE_API_KEY", "sb_publishable_ACJWlzQHlZjBrEguHvfOxg_3BJgxAaH")
	supabaseDBURL   = getEnv("SUPABASE_DB_URL", "postgresql://postgres:postgres@127.0.0.1:54322/postgres")

	// Localbase endpoints (runs on port 54321 for main API)
	// Use the same Supabase API key format for compatibility testing
	localbaseRESTURL = getEnv("LOCALBASE_REST_URL", "http://localhost:54321/rest/v1")
	localbaseAPIKey  = getEnv("LOCALBASE_API_KEY", "sb_publishable_ACJWlzQHlZjBrEguHvfOxg_3BJgxAaH")
	localbaseDBURL   = getEnv("LOCALBASE_DB_URL", "postgresql://localbase:localbase@localhost:5432/localbase")
)

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

// TestClient wraps HTTP client for API testing
type TestClient struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

func NewTestClient(baseURL, apiKey string) *TestClient {
	return &TestClient{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		apiKey:  apiKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *TestClient) Do(method, path string, body interface{}, headers map[string]string) (*http.Response, []byte, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, nil, err
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", c.apiKey)
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, err
	}

	return resp, respBody, nil
}

// Comparison result
type ComparisonResult struct {
	TestName          string
	SupabaseStatus    int
	LocalbaseStatus   int
	StatusMatch       bool
	SupabaseBody      interface{}
	LocalbaseBody     interface{}
	BodyMatch         bool
	SupabaseHeaders   map[string]string
	LocalbaseHeaders  map[string]string
	HeadersMatch      bool
	ErrorCodeMatch    bool
	SupabaseErrorCode string
	LocalbaseErrorCode string
	Notes             string
}

func (r *ComparisonResult) String() string {
	status := "PASS"
	if !r.StatusMatch || !r.ErrorCodeMatch {
		status = "FAIL"
	}
	return fmt.Sprintf("[%s] %s - Status: %d/%d, ErrorCode: %s/%s",
		status, r.TestName,
		r.SupabaseStatus, r.LocalbaseStatus,
		r.SupabaseErrorCode, r.LocalbaseErrorCode)
}

// Compare runs the same request against both endpoints and compares results
func Compare(t *testing.T, name, method, path string, body interface{}, headers map[string]string) *ComparisonResult {
	t.Helper()

	supabase := NewTestClient(supabaseRESTURL, supabaseAPIKey)
	localbase := NewTestClient(localbaseRESTURL, localbaseAPIKey)

	result := &ComparisonResult{TestName: name}

	// Run against Supabase
	sResp, sBody, sErr := supabase.Do(method, path, body, headers)
	if sErr != nil {
		t.Logf("Supabase request error: %v", sErr)
		result.Notes = fmt.Sprintf("Supabase error: %v", sErr)
	} else {
		result.SupabaseStatus = sResp.StatusCode
		result.SupabaseHeaders = extractHeaders(sResp)
		if err := json.Unmarshal(sBody, &result.SupabaseBody); err != nil {
			result.SupabaseBody = string(sBody)
		}
		result.SupabaseErrorCode = extractErrorCode(sBody)
	}

	// Run against Localbase
	lResp, lBody, lErr := localbase.Do(method, path, body, headers)
	if lErr != nil {
		t.Logf("Localbase request error: %v", lErr)
		result.Notes += fmt.Sprintf(" Localbase error: %v", lErr)
	} else {
		result.LocalbaseStatus = lResp.StatusCode
		result.LocalbaseHeaders = extractHeaders(lResp)
		if err := json.Unmarshal(lBody, &result.LocalbaseBody); err != nil {
			result.LocalbaseBody = string(lBody)
		}
		result.LocalbaseErrorCode = extractErrorCode(lBody)
	}

	// Compare results
	result.StatusMatch = result.SupabaseStatus == result.LocalbaseStatus
	result.ErrorCodeMatch = result.SupabaseErrorCode == result.LocalbaseErrorCode

	// Log result
	t.Log(result.String())
	if !result.StatusMatch {
		t.Errorf("Status mismatch: Supabase=%d, Localbase=%d", result.SupabaseStatus, result.LocalbaseStatus)
	}
	if !result.ErrorCodeMatch && (result.SupabaseErrorCode != "" || result.LocalbaseErrorCode != "") {
		t.Errorf("Error code mismatch: Supabase=%s, Localbase=%s", result.SupabaseErrorCode, result.LocalbaseErrorCode)
	}

	return result
}

func extractHeaders(resp *http.Response) map[string]string {
	headers := make(map[string]string)
	for _, key := range []string{"Content-Type", "Content-Range", "Location", "Preference-Applied"} {
		if v := resp.Header.Get(key); v != "" {
			headers[key] = v
		}
	}
	return headers
}

func extractErrorCode(body []byte) string {
	var errResp struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Code != "" {
		return errResp.Code
	}
	return ""
}

// =============================================================================
// CRUD Operations Tests
// =============================================================================

func TestSelect_Basic(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		method string
	}{
		{"SELECT-001: Select all rows", "/test_users", "GET"},
		{"SELECT-002: Select with limit", "/test_users?limit=10", "GET"},
		{"SELECT-003: Select with offset", "/test_users?offset=5&limit=10", "GET"},
		{"SELECT-004: Select non-existent table", "/nonexistent_table", "GET"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Compare(t, tt.name, tt.method, tt.path, nil, nil)
		})
	}
}

func TestSelect_VerticalFiltering(t *testing.T) {
	tests := []struct {
		name   string
		path   string
	}{
		{"SELECT-010: Select specific columns", "/test_users?select=id,email,name"},
		{"SELECT-011: Select with alias", "/test_users?select=user_id:id,user_email:email"},
		{"SELECT-012: Select all columns", "/test_users?select=*"},
		{"SELECT-014: Select non-existent column", "/test_users?select=nonexistent_column"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Compare(t, tt.name, "GET", tt.path, nil, nil)
		})
	}
}

func TestSelect_HorizontalFiltering(t *testing.T) {
	tests := []struct {
		name   string
		path   string
	}{
		{"FILTER-001: eq operator", "/test_users?status=eq.active"},
		{"FILTER-002: neq operator", "/test_users?status=neq.deleted"},
		{"FILTER-003: gt operator", "/test_users?age=gt.25"},
		{"FILTER-004: gte operator", "/test_users?age=gte.25"},
		{"FILTER-005: lt operator", "/test_users?age=lt.30"},
		{"FILTER-006: lte operator", "/test_users?age=lte.30"},
		{"FILTER-007: like operator", "/test_users?name=like.*John*"},
		{"FILTER-008: ilike operator", "/test_users?name=ilike.*john*"},
		{"FILTER-030: is.null", "/posts?published_at=is.null"},
		{"FILTER-031: is.true", "/posts?published=is.true"},
		{"FILTER-032: is.false", "/posts?published=is.false"},
		{"FILTER-034: not.is.null", "/posts?published_at=not.is.null"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Compare(t, tt.name, "GET", tt.path, nil, nil)
		})
	}
}

func TestSelect_ArrayOperators(t *testing.T) {
	tests := []struct {
		name   string
		path   string
	}{
		// Use status column (VARCHAR) for IN operator test instead of id (UUID) to avoid type conversion issues
		{"FILTER-020: in operator", "/test_users?status=in.(active,inactive,pending)"},
		{"FILTER-021: cs (contains) operator", "/test_users?tags=cs.{premium}"},
		{"FILTER-022: cd (contained by) operator", "/test_users?tags=cd.{premium,standard,trial}"},
		{"FILTER-023: ov (overlap) operator", "/test_users?tags=ov.{premium,newsletter}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Compare(t, tt.name, "GET", tt.path, nil, nil)
		})
	}
}

func TestSelect_LogicalOperators(t *testing.T) {
	tests := []struct {
		name   string
		path   string
	}{
		{"FILTER-050: not operator", "/test_users?status=not.eq.deleted"},
		{"FILTER-051: and operator", "/test_users?and=(age.gt.18,status.eq.active)"},
		{"FILTER-052: or operator", "/test_users?or=(status.eq.pending,status.eq.active)"},
		{"FILTER-054: multiple conditions (implicit AND)", "/test_users?age=gt.18&status=eq.active"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Compare(t, tt.name, "GET", tt.path, nil, nil)
		})
	}
}

func TestSelect_Ordering(t *testing.T) {
	tests := []struct {
		name   string
		path   string
	}{
		{"ORDER-001: Order ascending", "/test_users?order=created_at.asc"},
		{"ORDER-002: Order descending", "/test_users?order=created_at.desc"},
		{"ORDER-003: Multi-column order", "/test_users?order=status.asc,created_at.desc"},
		{"ORDER-004: Nulls first", "/posts?order=published_at.asc.nullsfirst"},
		{"ORDER-005: Nulls last", "/posts?order=published_at.asc.nullslast"},
		{"ORDER-006: Order by non-existent", "/test_users?order=nonexistent.asc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Compare(t, tt.name, "GET", tt.path, nil, nil)
		})
	}
}

func TestSelect_JSONBOperators(t *testing.T) {
	tests := []struct {
		name   string
		path   string
	}{
		// Use ->> for text extraction (-> returns JSON which causes type mismatch)
		{"FILTER-060: JSON path access", "/test_users?metadata->>role=eq.admin"},
		{"FILTER-063: JSON contains", "/test_users?metadata=cs.{\"role\":\"admin\"}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Compare(t, tt.name, "GET", tt.path, nil, nil)
		})
	}
}

func TestInsert_Basic(t *testing.T) {
	t.Run("INSERT-001: Insert single row", func(t *testing.T) {
		body := map[string]interface{}{
			"email": fmt.Sprintf("test_%d@example.com", time.Now().UnixNano()),
			"name":  "Test User",
			"age":   25,
		}
		Compare(t, "INSERT-001", "POST", "/test_users", body, nil)
	})

	t.Run("INSERT-010: Bulk insert array", func(t *testing.T) {
		body := []map[string]interface{}{
			{"email": fmt.Sprintf("bulk1_%d@example.com", time.Now().UnixNano()), "name": "Bulk User 1"},
			{"email": fmt.Sprintf("bulk2_%d@example.com", time.Now().UnixNano()), "name": "Bulk User 2"},
		}
		Compare(t, "INSERT-010", "POST", "/test_users", body, nil)
	})
}

func TestInsert_ReturnPreferences(t *testing.T) {
	t.Run("INSERT-020: Return minimal", func(t *testing.T) {
		body := map[string]interface{}{
			"email": fmt.Sprintf("minimal_%d@example.com", time.Now().UnixNano()),
			"name":  "Minimal User",
		}
		headers := map[string]string{"Prefer": "return=minimal"}
		Compare(t, "INSERT-020", "POST", "/test_users", body, headers)
	})

	t.Run("INSERT-021: Return representation", func(t *testing.T) {
		body := map[string]interface{}{
			"email": fmt.Sprintf("rep_%d@example.com", time.Now().UnixNano()),
			"name":  "Representation User",
		}
		headers := map[string]string{"Prefer": "return=representation"}
		Compare(t, "INSERT-021", "POST", "/test_users", body, headers)
	})
}

func TestInsert_Upsert(t *testing.T) {
	// First insert a row
	email := fmt.Sprintf("upsert_%d@example.com", time.Now().UnixNano())
	body := map[string]interface{}{
		"email": email,
		"name":  "Original Name",
	}

	t.Run("INSERT-030: Initial insert", func(t *testing.T) {
		Compare(t, "INSERT-030-prep", "POST", "/test_users", body, nil)
	})

	t.Run("INSERT-031: Upsert merge duplicates", func(t *testing.T) {
		body["name"] = "Updated Name"
		headers := map[string]string{
			"Prefer": "resolution=merge-duplicates",
		}
		Compare(t, "INSERT-031", "POST", "/test_users?on_conflict=email", body, headers)
	})
}

func TestUpdate_Basic(t *testing.T) {
	t.Run("UPDATE-001: Update with filter", func(t *testing.T) {
		body := map[string]interface{}{
			"status": "updated",
		}
		Compare(t, "UPDATE-001", "PATCH", "/test_users?status=eq.pending&limit=1", body, nil)
	})

	t.Run("UPDATE-003: Update all (should be blocked)", func(t *testing.T) {
		body := map[string]interface{}{
			"status": "mass_update",
		}
		Compare(t, "UPDATE-003", "PATCH", "/test_users", body, nil)
	})
}

func TestUpdate_ReturnPreferences(t *testing.T) {
	t.Run("UPDATE-010: Return representation", func(t *testing.T) {
		body := map[string]interface{}{
			"status": "active",
		}
		headers := map[string]string{"Prefer": "return=representation"}
		Compare(t, "UPDATE-010", "PATCH", "/test_users?status=eq.inactive&limit=1", body, headers)
	})
}

func TestDelete_Basic(t *testing.T) {
	// Create a test record first
	email := fmt.Sprintf("delete_%d@example.com", time.Now().UnixNano())
	body := map[string]interface{}{
		"email": email,
		"name":  "Delete Test",
	}

	t.Run("DELETE-prep: Create record", func(t *testing.T) {
		Compare(t, "DELETE-prep", "POST", "/test_users", body, nil)
	})

	t.Run("DELETE-001: Delete single row", func(t *testing.T) {
		Compare(t, "DELETE-001", "DELETE", "/test_users?email=eq."+url.QueryEscape(email), nil, nil)
	})

	t.Run("DELETE-003: Delete all (should be blocked)", func(t *testing.T) {
		Compare(t, "DELETE-003", "DELETE", "/test_users", nil, nil)
	})
}

// =============================================================================
// Resource Embedding Tests
// =============================================================================

func TestEmbedding_ManyToOne(t *testing.T) {
	tests := []struct {
		name   string
		path   string
	}{
		{"EMBED-001: Embed parent", "/posts?select=*,author:test_users(*)"},
		{"EMBED-002: Embed specific columns", "/posts?select=*,author:test_users(id,name,email)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Compare(t, tt.name, "GET", tt.path, nil, nil)
		})
	}
}

func TestEmbedding_OneToMany(t *testing.T) {
	tests := []struct {
		name   string
		path   string
	}{
		{"EMBED-010: Embed children", "/test_users?select=*,posts(*)"},
		{"EMBED-012: Filter embedded", "/test_users?select=*,posts(*)&posts.published=eq.true"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Compare(t, tt.name, "GET", tt.path, nil, nil)
		})
	}
}

func TestEmbedding_ManyToMany(t *testing.T) {
	tests := []struct {
		name   string
		path   string
	}{
		{"EMBED-020: Through junction", "/posts?select=*,tags(*)"},
		{"EMBED-021: Junction with data", "/posts?select=*,post_tags(*,tags(*))"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Compare(t, tt.name, "GET", tt.path, nil, nil)
		})
	}
}

func TestEmbedding_Nested(t *testing.T) {
	tests := []struct {
		name   string
		path   string
	}{
		{"EMBED-030: Deep nesting", "/test_users?select=*,posts(id,title,comments(*))"},
		{"EMBED-031: Multiple embeds", "/posts?select=*,author:test_users(*),comments(*)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Compare(t, tt.name, "GET", tt.path, nil, nil)
		})
	}
}

// =============================================================================
// RPC (Stored Procedures) Tests
// =============================================================================

func TestRPC_Basic(t *testing.T) {
	t.Run("RPC-001: Call simple function", func(t *testing.T) {
		body := map[string]interface{}{
			"a": 5,
			"b": 3,
		}
		Compare(t, "RPC-001", "POST", "/rpc/add_numbers", body, nil)
	})

	t.Run("RPC-005: Function returning set", func(t *testing.T) {
		Compare(t, "RPC-005", "POST", "/rpc/get_active_users", nil, nil)
	})

	t.Run("RPC-007: Void function", func(t *testing.T) {
		// Get a post ID first
		Compare(t, "RPC-007", "POST", "/rpc/update_post_view_count", map[string]interface{}{
			"post_uuid": "00000000-0000-0000-0000-000000000000",
		}, nil)
	})
}

func TestRPC_WithFilters(t *testing.T) {
	t.Run("RPC-010: Filter result", func(t *testing.T) {
		Compare(t, "RPC-010", "POST", "/rpc/get_active_users?age=gt.25", nil, nil)
	})

	t.Run("RPC-011: Order result", func(t *testing.T) {
		Compare(t, "RPC-011", "POST", "/rpc/get_active_users?order=name", nil, nil)
	})

	t.Run("RPC-012: Select columns", func(t *testing.T) {
		Compare(t, "RPC-012", "POST", "/rpc/get_active_users?select=id,name", nil, nil)
	})
}

// =============================================================================
// Range and Count Tests
// =============================================================================

func TestRange_Headers(t *testing.T) {
	t.Run("RANGE-002: Range with exact count", func(t *testing.T) {
		headers := map[string]string{
			"Range":  "0-9",
			"Prefer": "count=exact",
		}
		Compare(t, "RANGE-002", "GET", "/test_users", nil, headers)
	})

	t.Run("RANGE-003: Planned count", func(t *testing.T) {
		headers := map[string]string{
			"Prefer": "count=planned",
		}
		Compare(t, "RANGE-003", "GET", "/test_users", nil, headers)
	})
}

func TestCount_Aggregates(t *testing.T) {
	t.Run("AGG-001: Count via HEAD", func(t *testing.T) {
		headers := map[string]string{
			"Prefer": "count=exact",
		}
		Compare(t, "AGG-001", "HEAD", "/test_users", nil, headers)
	})
}

// =============================================================================
// Error Handling Tests
// =============================================================================

func TestErrors_BadRequest(t *testing.T) {
	tests := []struct {
		name   string
		path   string
	}{
		{"ERR-002: Invalid filter", "/test_users?invalid_filter"},
		{"ERR-003: Invalid column in select", "/test_users?select=nonexistent_col"},
		{"ERR-006: Invalid order", "/test_users?order=nonexistent.asc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Compare(t, tt.name, "GET", tt.path, nil, nil)
		})
	}
}

func TestErrors_ConstraintViolations(t *testing.T) {
	t.Run("ERR-007: Unique violation", func(t *testing.T) {
		// Insert duplicate email
		body := map[string]interface{}{
			"email": "admin@localbase.dev",
			"name":  "Duplicate",
		}
		Compare(t, "ERR-007", "POST", "/test_users", body, nil)
	})

	t.Run("ERR-008: Not null violation", func(t *testing.T) {
		body := map[string]interface{}{
			"name": "No Email",
			// Missing required email field
		}
		Compare(t, "ERR-008", "POST", "/test_users", body, nil)
	})
}

// =============================================================================
// Security Tests
// =============================================================================

func TestSecurity_SQLInjection(t *testing.T) {
	tests := []struct {
		name   string
		path   string
	}{
		{"SEC-001: Filter injection", "/test_users?name=eq.'; DROP TABLE test_users;--"},
		{"SEC-002: Column injection", "/test_users?select=id;DROP TABLE test_users"},
		{"SEC-003: Order injection", "/test_users?order=name;DROP TABLE test_users"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Compare(t, tt.name, "GET", tt.path, nil, nil)
		})
	}
}

// =============================================================================
// Edge Cases Tests
// =============================================================================

func TestEdgeCases_Data(t *testing.T) {
	t.Run("EDGE-001: Empty string", func(t *testing.T) {
		body := map[string]interface{}{
			"email": fmt.Sprintf("empty_%d@example.com", time.Now().UnixNano()),
			"name":  "",
		}
		Compare(t, "EDGE-001", "POST", "/test_users", body, nil)
	})

	t.Run("EDGE-003: Unicode characters", func(t *testing.T) {
		body := map[string]interface{}{
			"email": fmt.Sprintf("unicode_%d@example.com", time.Now().UnixNano()),
			"name":  "日本語ユーザー",
		}
		Compare(t, "EDGE-003", "POST", "/test_users", body, nil)
	})

	t.Run("EDGE-004: Special characters", func(t *testing.T) {
		body := map[string]interface{}{
			"email": fmt.Sprintf("special_%d@example.com", time.Now().UnixNano()),
			"name":  "O'Brien & Associates <test>",
		}
		Compare(t, "EDGE-004", "POST", "/test_users", body, nil)
	})

	t.Run("EDGE-005: Zero value vs null", func(t *testing.T) {
		body := map[string]interface{}{
			"email": fmt.Sprintf("zero_%d@example.com", time.Now().UnixNano()),
			"name":  "Zero Age",
			"age":   0,
		}
		Compare(t, "EDGE-005", "POST", "/test_users", body, nil)
	})
}

func TestEdgeCases_Queries(t *testing.T) {
	tests := []struct {
		name   string
		path   string
	}{
		{"EDGE-010: Many filters", "/test_users?status=eq.active&age=gt.18&age=lt.65"},
		// Use status column (VARCHAR) for IN operator test instead of id (UUID) to avoid type conversion issues
		{"EDGE-012: Large IN list", "/test_users?status=in.(active,inactive,pending,suspended)"},
		{"EDGE-013: Nested logic", "/test_users?and=(or(status.eq.active,status.eq.pending),age.gt.18)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Compare(t, tt.name, "GET", tt.path, nil, nil)
		})
	}
}

// =============================================================================
// Views Tests
// =============================================================================

func TestViews(t *testing.T) {
	tests := []struct {
		name   string
		path   string
	}{
		{"VIEW-001: Select from view", "/published_posts"},
		{"VIEW-002: Select from aggregation view", "/user_stats"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Compare(t, tt.name, "GET", tt.path, nil, nil)
		})
	}
}

// =============================================================================
// Setup and Teardown
// =============================================================================

func TestMain(m *testing.M) {
	// Setup: Seed both databases
	ctx := context.Background()

	// Seed Supabase
	fmt.Println("Seeding Supabase database...")
	if seeder, err := NewFromConnString(ctx, supabaseDBURL); err == nil {
		if err := seeder.SeedAll(ctx); err != nil {
			fmt.Printf("Warning: Failed to seed Supabase: %v\n", err)
		}
		seeder.Close()
	}

	// Seed Localbase
	fmt.Println("Seeding Localbase database...")
	if seeder, err := NewFromConnString(ctx, localbaseDBURL); err == nil {
		if err := seeder.SeedAll(ctx); err != nil {
			fmt.Printf("Warning: Failed to seed Localbase: %v\n", err)
		}
		seeder.Close()
	}

	// Run tests
	code := m.Run()

	os.Exit(code)
}
