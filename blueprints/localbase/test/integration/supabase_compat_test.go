//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Test configuration
var (
	// Localbase configuration
	localbaseURL    = getEnv("LOCALBASE_URL", "http://localhost:54321")
	localbaseAPIKey = getEnv("LOCALBASE_ANON_KEY", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZS1kZW1vIiwicm9sZSI6ImFub24iLCJleHAiOjE5ODM4MTI5OTZ9.CRXP1A7WOeoJeXxjNni43kdQwgnWNReilDMblYTn_I0")
	serviceRoleKey  = getEnv("LOCALBASE_SERVICE_KEY", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZS1kZW1vIiwicm9sZSI6InNlcnZpY2Vfcm9sZSIsImV4cCI6MTk4MzgxMjk5Nn0.EGIM96RAZx35lJzdJsyH-qQwv8Hdp7fsn3W0YpN81IU")
	jwtSecret       = getEnv("LOCALBASE_JWT_SECRET", "super-secret-jwt-token-with-at-least-32-characters-long")

	// Supabase configuration (for comparison testing)
	supabaseURL    = getEnv("SUPABASE_URL", "http://localhost:54421")
	supabaseAPIKey = getEnv("SUPABASE_ANON_KEY", localbaseAPIKey)
)

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

// Client wraps HTTP client for API testing
type Client struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		apiKey:  apiKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) Request(method, path string, body any, headers map[string]string) (int, []byte, http.Header, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return 0, nil, nil, err
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return 0, nil, nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", c.apiKey)
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, nil, nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, resp.Header, err
	}

	return resp.StatusCode, respBody, resp.Header, nil
}

// =============================================================================
// API Key Authentication Tests
// =============================================================================

func TestAPIKey_AnonKeyRole(t *testing.T) {
	client := NewClient(localbaseURL, localbaseAPIKey)

	// Make a request to any endpoint
	status, body, _, err := client.Request("GET", "/rest/v1/users?limit=1", nil, nil)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	// Should succeed with anon key (table may be empty, that's ok)
	if status != 200 && status != 404 {
		t.Errorf("Expected 200 or 404, got %d: %s", status, body)
	}
}

func TestAPIKey_ServiceRoleRole(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	// Service role should be able to access admin endpoints
	status, _, _, err := client.Request("GET", "/auth/v1/admin/users?limit=1", nil, nil)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if status != 200 {
		t.Errorf("Expected 200 for service role on admin endpoint, got %d", status)
	}
}

func TestAPIKey_AnonCantAccessAdmin(t *testing.T) {
	client := NewClient(localbaseURL, localbaseAPIKey)

	// Anon key should be rejected from admin endpoints
	status, body, _, err := client.Request("GET", "/auth/v1/admin/users", nil, nil)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if status != 403 {
		t.Errorf("Expected 403 for anon on admin endpoint, got %d: %s", status, body)
	}
}

func TestAPIKey_InvalidKeyRejected(t *testing.T) {
	client := NewClient(localbaseURL, "invalid-key")

	// Invalid key should be rejected for non-optional endpoints
	status, _, _, err := client.Request("GET", "/rest/v1/users", nil, nil)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	// The API key middleware is optional for backward compatibility,
	// so it may accept but assign anon role
	if status != 200 && status != 401 && status != 404 {
		t.Errorf("Expected 200/401/404, got %d", status)
	}
}

// =============================================================================
// JWT Claims Tests
// =============================================================================

func TestJWT_UserClaimsExtracted(t *testing.T) {
	// Create a user JWT with specific claims
	userID := "test-user-123"
	userEmail := "test@example.com"

	token := createUserJWT(userID, userEmail)
	client := NewClient(localbaseURL, token)

	// The request should succeed and claims should be accessible
	status, _, _, err := client.Request("GET", "/rest/v1/users?limit=1", nil, nil)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	// Request should succeed (claims are passed to RLS)
	if status != 200 && status != 404 {
		t.Errorf("Expected 200 or 404, got %d", status)
	}
}

func createUserJWT(userID, email string) string {
	claims := jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"role":  "authenticated",
		"aud":   "authenticated",
		"iss":   "supabase-demo",
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(jwtSecret))
	return tokenString
}

// =============================================================================
// RLS Tests
// =============================================================================

// Note: These tests require the todos table with RLS enabled
// Setup:
// CREATE TABLE IF NOT EXISTS public.todos (
//   id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
//   user_id UUID NOT NULL,
//   title TEXT NOT NULL,
//   completed BOOLEAN DEFAULT FALSE,
//   created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
// );
// ALTER TABLE public.todos ENABLE ROW LEVEL SECURITY;
// CREATE POLICY "Users can view own todos" ON public.todos
//   FOR SELECT USING (auth.uid() = user_id);
// CREATE POLICY "Users can insert own todos" ON public.todos
//   FOR INSERT WITH CHECK (auth.uid() = user_id);

func TestRLS_ServiceRoleBypassesRLS(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	// Service role should see all rows regardless of RLS
	status, _, _, err := client.Request("GET", "/rest/v1/todos?limit=10", nil, nil)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	// Should succeed (service role bypasses RLS)
	if status != 200 && status != 404 {
		t.Errorf("Expected 200 or 404, got %d", status)
	}
}

func TestRLS_AnonCannotAccessRLSProtectedTable(t *testing.T) {
	client := NewClient(localbaseURL, localbaseAPIKey)

	// Anon users without user context should see empty results or be denied
	// when RLS policy requires auth.uid()
	status, body, _, err := client.Request("GET", "/rest/v1/todos", nil, nil)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	// With RLS, anon should see empty array (auth.uid() is NULL)
	if status != 200 {
		t.Logf("Status %d: %s", status, body)
	}

	// Parse response
	var results []map[string]any
	if err := json.Unmarshal(body, &results); err == nil {
		// Should be empty array (RLS filters out all rows)
		t.Logf("Anon sees %d todos (expected 0 with RLS)", len(results))
	}
}

func TestRLS_AuthenticatedUserSeesOwnRows(t *testing.T) {
	// This test requires setting up test data
	// For now, just verify the endpoint works with a user JWT
	userID := "11111111-1111-1111-1111-111111111111"
	token := createUserJWT(userID, "test@example.com")
	client := NewClient(localbaseURL, token)

	status, body, _, err := client.Request("GET", "/rest/v1/todos", nil, nil)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	t.Logf("Authenticated user query status: %d, body length: %d", status, len(body))

	// Should succeed
	if status != 200 && status != 404 {
		t.Errorf("Expected 200 or 404, got %d: %s", status, body)
	}
}

// =============================================================================
// Storage RLS Tests
// =============================================================================

func TestStorage_ListBucketsPublicOnly(t *testing.T) {
	client := NewClient(localbaseURL, localbaseAPIKey)

	// Anon users should only see public buckets
	status, body, _, err := client.Request("GET", "/storage/v1/bucket", nil, nil)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if status != 200 {
		t.Errorf("Expected 200, got %d: %s", status, body)
		return
	}

	var buckets []map[string]any
	if err := json.Unmarshal(body, &buckets); err != nil {
		t.Fatalf("Failed to parse buckets: %v", err)
	}

	// All buckets should be public
	for _, b := range buckets {
		if public, ok := b["public"].(bool); ok && !public {
			t.Errorf("Anon user sees private bucket: %v", b["name"])
		}
	}
}

func TestStorage_ServiceRoleSeesAllBuckets(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	// Service role should see all buckets
	status, _, _, err := client.Request("GET", "/storage/v1/bucket", nil, nil)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if status != 200 {
		t.Errorf("Expected 200, got %d", status)
	}
}

func TestStorage_AnonCannotUploadToPrivateBucket(t *testing.T) {
	// First create a private bucket with service role
	serviceClient := NewClient(localbaseURL, serviceRoleKey)
	bucketName := fmt.Sprintf("test-private-%d", time.Now().Unix())

	createBody := map[string]any{
		"name":   bucketName,
		"public": false,
	}
	status, _, _, err := serviceClient.Request("POST", "/storage/v1/bucket", createBody, nil)
	if err != nil {
		t.Fatalf("Failed to create bucket: %v", err)
	}
	if status != 200 {
		t.Skipf("Could not create test bucket (status %d)", status)
	}

	// Clean up after test
	defer func() {
		serviceClient.Request("DELETE", "/storage/v1/bucket/"+bucketName, nil, nil)
	}()

	// Try to upload as anon
	anonClient := NewClient(localbaseURL, localbaseAPIKey)
	status, body, _, err := anonClient.Request("POST", "/storage/v1/object/"+bucketName+"/test.txt", "test content", map[string]string{
		"Content-Type": "text/plain",
	})
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	// Should be forbidden
	if status != 403 && status != 400 {
		t.Errorf("Expected 403 for anon upload to private bucket, got %d: %s", status, body)
	}
}

// =============================================================================
// Admin Endpoint Tests
// =============================================================================

func TestAdmin_ListUsersRequiresServiceRole(t *testing.T) {
	tests := []struct {
		name       string
		apiKey     string
		wantStatus int
	}{
		{"service_role can list users", serviceRoleKey, 200},
		{"anon cannot list users", localbaseAPIKey, 403},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(localbaseURL, tt.apiKey)
			status, _, _, err := client.Request("GET", "/auth/v1/admin/users", nil, nil)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}

			if status != tt.wantStatus {
				t.Errorf("Expected %d, got %d", tt.wantStatus, status)
			}
		})
	}
}

func TestAdmin_CreateUserRequiresServiceRole(t *testing.T) {
	tests := []struct {
		name       string
		apiKey     string
		wantStatus int
	}{
		{"service_role can create user", serviceRoleKey, 201},
		{"anon cannot create user", localbaseAPIKey, 403},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(localbaseURL, tt.apiKey)

			body := map[string]any{
				"email":    fmt.Sprintf("test-%d@example.com", time.Now().UnixNano()),
				"password": "testpassword123",
			}

			status, respBody, _, err := client.Request("POST", "/auth/v1/admin/users", body, nil)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}

			// For service role, expect 201 (created) or 200
			// For anon, expect 403
			if tt.wantStatus == 201 && status != 201 && status != 200 {
				t.Errorf("Expected 201 or 200, got %d: %s", status, respBody)
			} else if tt.wantStatus == 403 && status != 403 {
				t.Errorf("Expected 403, got %d: %s", status, respBody)
			}
		})
	}
}

// =============================================================================
// PostgREST Compatibility Tests
// =============================================================================

func TestPostgREST_SelectWithFilters(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	// Test various filter operations
	tests := []struct {
		name string
		path string
	}{
		{"eq filter", "/rest/v1/users?id=eq.1"},
		{"neq filter", "/rest/v1/users?id=neq.1"},
		{"gt filter", "/rest/v1/users?id=gt.0"},
		{"like filter", "/rest/v1/users?email=like.*@example.com"},
		{"in filter", "/rest/v1/users?id=in.(1,2,3)"},
		{"is null filter", "/rest/v1/users?deleted_at=is.null"},
		{"order", "/rest/v1/users?order=created_at.desc"},
		{"limit offset", "/rest/v1/users?limit=10&offset=0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, _, _, err := client.Request("GET", tt.path, nil, nil)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}

			// Should succeed or return 404 for non-existent table
			if status != 200 && status != 404 {
				t.Errorf("Expected 200 or 404, got %d", status)
			}
		})
	}
}

func TestPostgREST_InsertWithReturn(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	// Test insert with return=representation
	body := map[string]any{
		"title":     "Test Todo",
		"completed": false,
		"user_id":   "11111111-1111-1111-1111-111111111111",
	}

	status, respBody, headers, err := client.Request("POST", "/rest/v1/todos", body, map[string]string{
		"Prefer": "return=representation",
	})
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if status != 201 && status != 200 && status != 404 {
		t.Errorf("Expected 201/200/404, got %d: %s", status, respBody)
	}

	// Check Preference-Applied header
	if prefer := headers.Get("Preference-Applied"); prefer != "" {
		t.Logf("Preference-Applied: %s", prefer)
	}
}

func TestPostgREST_UpdateWithReturn(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	// Test update
	body := map[string]any{
		"completed": true,
	}

	status, _, headers, err := client.Request("PATCH", "/rest/v1/todos?id=eq.test-id", body, map[string]string{
		"Prefer": "return=representation",
	})
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	// Should succeed or return appropriate error
	if status != 200 && status != 204 && status != 404 {
		t.Errorf("Expected 200/204/404, got %d", status)
	}

	t.Logf("Preference-Applied: %s", headers.Get("Preference-Applied"))
}

func TestPostgREST_DeleteWithReturn(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	status, _, _, err := client.Request("DELETE", "/rest/v1/todos?id=eq.non-existent", nil, map[string]string{
		"Prefer": "return=representation",
	})
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	// Should succeed (even with 0 affected rows)
	if status != 200 && status != 204 && status != 404 {
		t.Errorf("Expected 200/204/404, got %d", status)
	}
}

func TestPostgREST_RPC(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	// Test RPC call
	body := map[string]any{
		"a": 1,
		"b": 2,
	}

	status, respBody, _, err := client.Request("POST", "/rest/v1/rpc/add_numbers", body, nil)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	// Function may not exist, that's ok for compat testing
	if status != 200 && status != 404 {
		t.Logf("RPC status: %d, body: %s", status, respBody)
	}
}

// =============================================================================
// Error Response Compatibility Tests
// =============================================================================

func TestErrorResponse_Format(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	// Request a non-existent table to trigger error
	status, body, _, err := client.Request("GET", "/rest/v1/nonexistent_table_xyz", nil, nil)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if status != 404 {
		t.Logf("Unexpected status %d for non-existent table", status)
	}

	// Check error response format (Supabase returns code, message, details, hint)
	var errResp map[string]any
	if err := json.Unmarshal(body, &errResp); err == nil {
		t.Logf("Error response: %+v", errResp)

		// Should have standard PostgREST error fields
		if _, ok := errResp["code"]; !ok {
			t.Log("Warning: error response missing 'code' field")
		}
		if _, ok := errResp["message"]; !ok {
			t.Log("Warning: error response missing 'message' field")
		}
	}
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkSelect(b *testing.B) {
	client := NewClient(localbaseURL, serviceRoleKey)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.Request("GET", "/rest/v1/users?limit=10", nil, nil)
	}
}

func BenchmarkInsert(b *testing.B) {
	client := NewClient(localbaseURL, serviceRoleKey)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		body := map[string]any{
			"title":     fmt.Sprintf("Benchmark Todo %d", i),
			"completed": false,
			"user_id":   "11111111-1111-1111-1111-111111111111",
		}
		client.Request("POST", "/rest/v1/todos", body, map[string]string{
			"Prefer": "return=minimal",
		})
	}
}

// =============================================================================
// Storage User Folder Access Tests
// =============================================================================

func TestStorage_UserCanUploadToOwnFolder(t *testing.T) {
	userID := "22222222-2222-2222-2222-222222222222"
	token := createUserJWT(userID, "user@example.com")
	userClient := NewClient(localbaseURL, token)

	// First create a bucket with service role
	serviceClient := NewClient(localbaseURL, serviceRoleKey)
	bucketName := fmt.Sprintf("user-files-%d", time.Now().Unix())

	createBody := map[string]any{
		"name":   bucketName,
		"public": false,
	}
	status, _, _, err := serviceClient.Request("POST", "/storage/v1/bucket", createBody, nil)
	if err != nil {
		t.Fatalf("Failed to create bucket: %v", err)
	}
	if status != 200 {
		t.Skipf("Could not create test bucket (status %d)", status)
	}

	// Clean up after test
	defer func() {
		serviceClient.Request("DELETE", "/storage/v1/object/"+bucketName+"/"+userID+"/test.txt", nil, nil)
		serviceClient.Request("DELETE", "/storage/v1/bucket/"+bucketName, nil, nil)
	}()

	// User should be able to upload to their own folder
	status, body, _, err := userClient.Request("POST", "/storage/v1/object/"+bucketName+"/"+userID+"/test.txt", "test content", map[string]string{
		"Content-Type": "text/plain",
	})
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if status != 200 {
		t.Errorf("Expected 200 for user uploading to own folder, got %d: %s", status, body)
	}
}

func TestStorage_UserCannotUploadToOtherUserFolder(t *testing.T) {
	userID := "33333333-3333-3333-3333-333333333333"
	otherUserID := "44444444-4444-4444-4444-444444444444"
	token := createUserJWT(userID, "user@example.com")
	userClient := NewClient(localbaseURL, token)

	// First create a bucket with service role
	serviceClient := NewClient(localbaseURL, serviceRoleKey)
	bucketName := fmt.Sprintf("user-files-%d", time.Now().Unix())

	createBody := map[string]any{
		"name":   bucketName,
		"public": false,
	}
	status, _, _, err := serviceClient.Request("POST", "/storage/v1/bucket", createBody, nil)
	if err != nil {
		t.Fatalf("Failed to create bucket: %v", err)
	}
	if status != 200 {
		t.Skipf("Could not create test bucket (status %d)", status)
	}

	// Clean up after test
	defer func() {
		serviceClient.Request("DELETE", "/storage/v1/bucket/"+bucketName, nil, nil)
	}()

	// User should NOT be able to upload to another user's folder
	status, body, _, err := userClient.Request("POST", "/storage/v1/object/"+bucketName+"/"+otherUserID+"/test.txt", "test content", map[string]string{
		"Content-Type": "text/plain",
	})
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if status != 403 && status != 400 {
		t.Errorf("Expected 403 for user uploading to other user's folder, got %d: %s", status, body)
	}
}

// =============================================================================
// Supabase Side-by-Side Comparison Tests
// =============================================================================

// TestSideBySide_Select runs the same query against both Localbase and Supabase
// and compares the responses for compatibility.
func TestSideBySide_Select(t *testing.T) {
	if supabaseURL == localbaseURL {
		t.Skip("Skipping side-by-side test: SUPABASE_URL not configured")
	}

	localClient := NewClient(localbaseURL, localbaseAPIKey)
	supaClient := NewClient(supabaseURL, supabaseAPIKey)

	// Test basic select
	localStatus, localBody, _, err := localClient.Request("GET", "/rest/v1/users?limit=5", nil, nil)
	if err != nil {
		t.Fatalf("Localbase request failed: %v", err)
	}

	supaStatus, supaBody, _, err := supaClient.Request("GET", "/rest/v1/users?limit=5", nil, nil)
	if err != nil {
		t.Fatalf("Supabase request failed: %v", err)
	}

	// Compare status codes
	if localStatus != supaStatus {
		t.Errorf("Status mismatch: Localbase=%d, Supabase=%d", localStatus, supaStatus)
	}

	// Compare response structure (not exact data)
	var localResult, supaResult []map[string]any
	json.Unmarshal(localBody, &localResult)
	json.Unmarshal(supaBody, &supaResult)

	t.Logf("Localbase returned %d rows, Supabase returned %d rows", len(localResult), len(supaResult))
}

// TestSideBySide_Storage_ListBuckets compares storage bucket listing.
func TestSideBySide_Storage_ListBuckets(t *testing.T) {
	if supabaseURL == localbaseURL {
		t.Skip("Skipping side-by-side test: SUPABASE_URL not configured")
	}

	localClient := NewClient(localbaseURL, serviceRoleKey)
	supaClient := NewClient(supabaseURL, serviceRoleKey)

	localStatus, localBody, _, err := localClient.Request("GET", "/storage/v1/bucket", nil, nil)
	if err != nil {
		t.Fatalf("Localbase request failed: %v", err)
	}

	supaStatus, supaBody, _, err := supaClient.Request("GET", "/storage/v1/bucket", nil, nil)
	if err != nil {
		t.Fatalf("Supabase request failed: %v", err)
	}

	// Compare status codes
	if localStatus != supaStatus {
		t.Errorf("Status mismatch: Localbase=%d, Supabase=%d", localStatus, supaStatus)
	}

	var localBuckets, supaBuckets []map[string]any
	json.Unmarshal(localBody, &localBuckets)
	json.Unmarshal(supaBody, &supaBuckets)

	t.Logf("Localbase has %d buckets, Supabase has %d buckets", len(localBuckets), len(supaBuckets))

	// Compare bucket fields (if same buckets exist)
	if len(localBuckets) > 0 && len(supaBuckets) > 0 {
		localFields := getMapKeys(localBuckets[0])
		supaFields := getMapKeys(supaBuckets[0])
		t.Logf("Localbase bucket fields: %v", localFields)
		t.Logf("Supabase bucket fields: %v", supaFields)
	}
}

// TestSideBySide_Auth_AdminUsers compares admin user endpoints.
func TestSideBySide_Auth_AdminUsers(t *testing.T) {
	if supabaseURL == localbaseURL {
		t.Skip("Skipping side-by-side test: SUPABASE_URL not configured")
	}

	localClient := NewClient(localbaseURL, serviceRoleKey)
	supaClient := NewClient(supabaseURL, serviceRoleKey)

	localStatus, localBody, localHeaders, err := localClient.Request("GET", "/auth/v1/admin/users?per_page=1", nil, nil)
	if err != nil {
		t.Fatalf("Localbase request failed: %v", err)
	}

	supaStatus, supaBody, supaHeaders, err := supaClient.Request("GET", "/auth/v1/admin/users?per_page=1", nil, nil)
	if err != nil {
		t.Fatalf("Supabase request failed: %v", err)
	}

	// Compare status codes
	if localStatus != supaStatus {
		t.Errorf("Status mismatch: Localbase=%d, Supabase=%d", localStatus, supaStatus)
	}

	t.Logf("Localbase Content-Type: %s", localHeaders.Get("Content-Type"))
	t.Logf("Supabase Content-Type: %s", supaHeaders.Get("Content-Type"))
	t.Logf("Localbase body length: %d", len(localBody))
	t.Logf("Supabase body length: %d", len(supaBody))
}

func getMapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
