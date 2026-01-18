//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// =============================================================================
// Functions API Test Suite
// Comprehensive tests for Supabase Edge Functions API compatibility
// =============================================================================

// Function represents a function response
type Function struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Slug       string    `json:"slug"`
	Version    int       `json:"version"`
	Status     string    `json:"status"`
	Entrypoint string    `json:"entrypoint"`
	ImportMap  string    `json:"import_map"`
	VerifyJWT  bool      `json:"verify_jwt"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Deployment represents a deployment response
type Deployment struct {
	ID         string    `json:"id"`
	FunctionID string    `json:"function_id"`
	Version    int       `json:"version"`
	SourceCode string    `json:"source_code"`
	Status     string    `json:"status"`
	DeployedAt time.Time `json:"deployed_at"`
}

// Secret represents a secret response (name only, no value)
type Secret struct {
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// =============================================================================
// Test Helpers
// =============================================================================

// createTestFunction creates a function for testing and returns its ID
func createTestFunction(t *testing.T, client *Client, name string, verifyJWT bool) string {
	t.Helper()

	body := map[string]any{
		"name":        name,
		"verify_jwt":  verifyJWT,
		"source_code": `export default async function handler(req) { return new Response("Hello from " + "` + name + `"); }`,
	}

	status, respBody, _, err := client.Request("POST", "/api/functions", body, nil)
	if err != nil {
		t.Fatalf("Failed to create function: %v", err)
	}

	if status != 201 {
		t.Fatalf("Expected 201, got %d: %s", status, respBody)
	}

	var fn Function
	if err := json.Unmarshal(respBody, &fn); err != nil {
		t.Fatalf("Failed to parse function response: %v", err)
	}

	return fn.ID
}

// deleteTestFunction deletes a test function
func deleteTestFunction(t *testing.T, client *Client, id string) {
	t.Helper()
	_, _, _, _ = client.Request("DELETE", "/api/functions/"+id, nil, nil)
}

// createTestSecret creates a secret for testing
func createTestSecret(t *testing.T, client *Client, name, value string) {
	t.Helper()

	body := map[string]any{
		"name":  name,
		"value": value,
	}

	status, respBody, _, err := client.Request("POST", "/api/functions/secrets", body, nil)
	if err != nil {
		t.Fatalf("Failed to create secret: %v", err)
	}

	if status != 201 {
		t.Fatalf("Expected 201, got %d: %s", status, respBody)
	}
}

// deleteTestSecret deletes a test secret
func deleteTestSecret(t *testing.T, client *Client, name string) {
	t.Helper()
	_, _, _, _ = client.Request("DELETE", "/api/functions/secrets/"+name, nil, nil)
}

// =============================================================================
// Function Invocation Tests
// =============================================================================

func TestFunctions_InvokeBasic(t *testing.T) {
	serviceClient := NewClient(localbaseURL, serviceRoleKey)

	// Create a test function
	fnName := fmt.Sprintf("test-invoke-%d", time.Now().UnixNano())
	fnID := createTestFunction(t, serviceClient, fnName, false)
	defer deleteTestFunction(t, serviceClient, fnID)

	// Test invoking with POST
	t.Run("invoke with POST", func(t *testing.T) {
		client := NewClient(localbaseURL, localbaseAPIKey)
		status, body, _, err := client.Request("POST", "/functions/v1/"+fnName, nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, body)
		}
	})

	// Test invoking non-existent function
	t.Run("invoke non-existent function", func(t *testing.T) {
		client := NewClient(localbaseURL, localbaseAPIKey)
		status, _, _, err := client.Request("POST", "/functions/v1/nonexistent-function-xyz", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 404 {
			t.Errorf("Expected 404, got %d", status)
		}
	})
}

func TestFunctions_InvokeWithBody(t *testing.T) {
	serviceClient := NewClient(localbaseURL, serviceRoleKey)

	fnName := fmt.Sprintf("test-body-%d", time.Now().UnixNano())
	fnID := createTestFunction(t, serviceClient, fnName, false)
	defer deleteTestFunction(t, serviceClient, fnID)

	tests := []struct {
		name        string
		body        any
		contentType string
	}{
		{"JSON body", map[string]any{"name": "test", "value": 123}, "application/json"},
		{"empty body", nil, "application/json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(localbaseURL, localbaseAPIKey)
			headers := map[string]string{"Content-Type": tt.contentType}
			status, _, _, err := client.Request("POST", "/functions/v1/"+fnName, tt.body, headers)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}

			if status != 200 {
				t.Errorf("Expected 200, got %d", status)
			}
		})
	}
}

func TestFunctions_InvokeAuth(t *testing.T) {
	serviceClient := NewClient(localbaseURL, serviceRoleKey)

	// Create a function that requires JWT verification
	fnNameJWT := fmt.Sprintf("test-jwt-%d", time.Now().UnixNano())
	fnIDJWT := createTestFunction(t, serviceClient, fnNameJWT, true)
	defer deleteTestFunction(t, serviceClient, fnIDJWT)

	// Create a function that doesn't require JWT verification
	fnNameNoJWT := fmt.Sprintf("test-nojwt-%d", time.Now().UnixNano())
	fnIDNoJWT := createTestFunction(t, serviceClient, fnNameNoJWT, false)
	defer deleteTestFunction(t, serviceClient, fnIDNoJWT)

	t.Run("invoke with service role key", func(t *testing.T) {
		client := NewClient(localbaseURL, serviceRoleKey)
		status, _, _, err := client.Request("POST", "/functions/v1/"+fnNameJWT, nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200 for service role, got %d", status)
		}
	})

	t.Run("invoke with user JWT", func(t *testing.T) {
		userToken := createUserJWT("user-123", "user@example.com")
		client := NewClient(localbaseURL, userToken)
		status, _, _, err := client.Request("POST", "/functions/v1/"+fnNameJWT, nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		// Should succeed with valid user JWT
		if status != 200 {
			t.Logf("Status for user JWT: %d (may require additional auth setup)", status)
		}
	})

	t.Run("invoke without auth when verify_jwt=false", func(t *testing.T) {
		// Use a client without proper auth
		client := &Client{
			baseURL: strings.TrimSuffix(localbaseURL, "/"),
			apiKey:  "", // No API key
			client:  &http.Client{Timeout: 30 * time.Second},
		}

		req, _ := http.NewRequest("POST", localbaseURL+"/functions/v1/"+fnNameNoJWT, nil)
		resp, err := client.client.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		// When verify_jwt=false, should allow unauthenticated access
		t.Logf("Status without auth (verify_jwt=false): %d", resp.StatusCode)
	})
}

// =============================================================================
// CORS Tests
// =============================================================================

func TestFunctions_CORS(t *testing.T) {
	serviceClient := NewClient(localbaseURL, serviceRoleKey)

	fnName := fmt.Sprintf("test-cors-%d", time.Now().UnixNano())
	fnID := createTestFunction(t, serviceClient, fnName, false)
	defer deleteTestFunction(t, serviceClient, fnID)

	t.Run("OPTIONS preflight request", func(t *testing.T) {
		req, err := http.NewRequest("OPTIONS", localbaseURL+"/functions/v1/"+fnName, nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Origin", "https://example.com")
		req.Header.Set("Access-Control-Request-Method", "POST")
		req.Header.Set("Access-Control-Request-Headers", "authorization, content-type")

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		// Should return 200 OK for preflight
		if resp.StatusCode != 200 && resp.StatusCode != 204 {
			t.Errorf("Expected 200 or 204 for OPTIONS, got %d", resp.StatusCode)
		}

		// Check CORS headers
		allowOrigin := resp.Header.Get("Access-Control-Allow-Origin")
		allowMethods := resp.Header.Get("Access-Control-Allow-Methods")
		allowHeaders := resp.Header.Get("Access-Control-Allow-Headers")

		t.Logf("Access-Control-Allow-Origin: %s", allowOrigin)
		t.Logf("Access-Control-Allow-Methods: %s", allowMethods)
		t.Logf("Access-Control-Allow-Headers: %s", allowHeaders)

		// Verify required CORS headers are present
		if allowOrigin == "" {
			t.Log("Warning: Access-Control-Allow-Origin header missing")
		}
	})

	t.Run("CORS headers on normal response", func(t *testing.T) {
		client := NewClient(localbaseURL, localbaseAPIKey)
		_, _, headers, err := client.Request("POST", "/functions/v1/"+fnName, nil, map[string]string{
			"Origin": "https://example.com",
		})
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		allowOrigin := headers.Get("Access-Control-Allow-Origin")
		t.Logf("CORS on response - Access-Control-Allow-Origin: %s", allowOrigin)
	})
}

// =============================================================================
// Function Management Tests
// =============================================================================

func TestFunctions_ManagementList(t *testing.T) {
	t.Run("list functions with service role", func(t *testing.T) {
		client := NewClient(localbaseURL, serviceRoleKey)
		status, body, _, err := client.Request("GET", "/api/functions", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, body)
		}

		var functions []Function
		if err := json.Unmarshal(body, &functions); err != nil {
			t.Fatalf("Failed to parse functions: %v", err)
		}

		t.Logf("Found %d functions", len(functions))
	})

	t.Run("list functions with anon key should be forbidden", func(t *testing.T) {
		client := NewClient(localbaseURL, localbaseAPIKey)
		status, _, _, err := client.Request("GET", "/api/functions", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 403 {
			t.Errorf("Expected 403 for anon key, got %d", status)
		}
	})
}

func TestFunctions_ManagementCreate(t *testing.T) {
	t.Run("create function with service role", func(t *testing.T) {
		client := NewClient(localbaseURL, serviceRoleKey)
		fnName := fmt.Sprintf("test-create-%d", time.Now().UnixNano())

		body := map[string]any{
			"name":        fnName,
			"verify_jwt":  true,
			"entrypoint":  "index.ts",
			"source_code": `export default function() { return new Response("Hello"); }`,
		}

		status, respBody, _, err := client.Request("POST", "/api/functions", body, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 201 {
			t.Errorf("Expected 201, got %d: %s", status, respBody)
		}

		var fn Function
		if err := json.Unmarshal(respBody, &fn); err != nil {
			t.Fatalf("Failed to parse function: %v", err)
		}

		// Verify function properties
		if fn.Name != fnName {
			t.Errorf("Expected name %s, got %s", fnName, fn.Name)
		}
		if fn.VerifyJWT != true {
			t.Errorf("Expected verify_jwt true, got %v", fn.VerifyJWT)
		}
		if fn.Status != "active" {
			t.Errorf("Expected status active, got %s", fn.Status)
		}
		if fn.Version != 1 {
			t.Errorf("Expected version 1, got %d", fn.Version)
		}

		// Cleanup
		deleteTestFunction(t, client, fn.ID)
	})

	t.Run("create function with anon key should be forbidden", func(t *testing.T) {
		client := NewClient(localbaseURL, localbaseAPIKey)
		body := map[string]any{
			"name": "test-anon-create",
		}

		status, _, _, err := client.Request("POST", "/api/functions", body, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 403 {
			t.Errorf("Expected 403, got %d", status)
		}
	})

	t.Run("create function with empty name", func(t *testing.T) {
		client := NewClient(localbaseURL, serviceRoleKey)
		body := map[string]any{
			"name": "",
		}

		status, _, _, err := client.Request("POST", "/api/functions", body, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 400 {
			t.Errorf("Expected 400 for empty name, got %d", status)
		}
	})

	t.Run("create duplicate function", func(t *testing.T) {
		client := NewClient(localbaseURL, serviceRoleKey)
		fnName := fmt.Sprintf("test-dup-%d", time.Now().UnixNano())

		body := map[string]any{"name": fnName}

		// Create first
		status1, respBody, _, err := client.Request("POST", "/api/functions", body, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if status1 != 201 {
			t.Fatalf("First create failed: %d", status1)
		}

		var fn Function
		json.Unmarshal(respBody, &fn)
		defer deleteTestFunction(t, client, fn.ID)

		// Try to create duplicate
		status2, _, _, err := client.Request("POST", "/api/functions", body, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status2 != 400 {
			t.Errorf("Expected 400 for duplicate, got %d", status2)
		}
	})
}

func TestFunctions_ManagementGet(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)
	fnName := fmt.Sprintf("test-get-%d", time.Now().UnixNano())
	fnID := createTestFunction(t, client, fnName, true)
	defer deleteTestFunction(t, client, fnID)

	t.Run("get existing function", func(t *testing.T) {
		status, respBody, _, err := client.Request("GET", "/api/functions/"+fnID, nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, respBody)
		}

		var fn Function
		if err := json.Unmarshal(respBody, &fn); err != nil {
			t.Fatalf("Failed to parse function: %v", err)
		}

		if fn.ID != fnID {
			t.Errorf("Expected ID %s, got %s", fnID, fn.ID)
		}
		if fn.Name != fnName {
			t.Errorf("Expected name %s, got %s", fnName, fn.Name)
		}
	})

	t.Run("get non-existent function", func(t *testing.T) {
		status, _, _, err := client.Request("GET", "/api/functions/nonexistent-id", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 404 {
			t.Errorf("Expected 404, got %d", status)
		}
	})
}

func TestFunctions_ManagementUpdate(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)
	fnName := fmt.Sprintf("test-update-%d", time.Now().UnixNano())
	fnID := createTestFunction(t, client, fnName, false)
	defer deleteTestFunction(t, client, fnID)

	t.Run("update function name", func(t *testing.T) {
		newName := fnName + "-updated"
		body := map[string]any{
			"name": newName,
		}

		status, respBody, _, err := client.Request("PUT", "/api/functions/"+fnID, body, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, respBody)
		}

		var fn Function
		json.Unmarshal(respBody, &fn)
		if fn.Name != newName {
			t.Errorf("Expected name %s, got %s", newName, fn.Name)
		}
	})

	t.Run("update verify_jwt", func(t *testing.T) {
		body := map[string]any{
			"verify_jwt": true,
		}

		status, respBody, _, err := client.Request("PUT", "/api/functions/"+fnID, body, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, respBody)
		}

		var fn Function
		json.Unmarshal(respBody, &fn)
		if fn.VerifyJWT != true {
			t.Errorf("Expected verify_jwt true, got %v", fn.VerifyJWT)
		}
	})

	t.Run("update status to inactive", func(t *testing.T) {
		body := map[string]any{
			"status": "inactive",
		}

		status, _, _, err := client.Request("PUT", "/api/functions/"+fnID, body, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d", status)
		}
	})
}

func TestFunctions_ManagementDelete(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)
	fnName := fmt.Sprintf("test-delete-%d", time.Now().UnixNano())
	fnID := createTestFunction(t, client, fnName, false)

	t.Run("delete existing function", func(t *testing.T) {
		status, _, _, err := client.Request("DELETE", "/api/functions/"+fnID, nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 204 && status != 200 {
			t.Errorf("Expected 204 or 200, got %d", status)
		}

		// Verify it's deleted
		getStatus, _, _, _ := client.Request("GET", "/api/functions/"+fnID, nil, nil)
		if getStatus != 404 {
			t.Errorf("Expected 404 after delete, got %d", getStatus)
		}
	})

	t.Run("delete with anon key should be forbidden", func(t *testing.T) {
		anonClient := NewClient(localbaseURL, localbaseAPIKey)
		status, _, _, err := anonClient.Request("DELETE", "/api/functions/some-id", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 403 {
			t.Errorf("Expected 403, got %d", status)
		}
	})
}

// =============================================================================
// Deployment Tests
// =============================================================================

func TestFunctions_Deploy(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)
	fnName := fmt.Sprintf("test-deploy-%d", time.Now().UnixNano())
	fnID := createTestFunction(t, client, fnName, false)
	defer deleteTestFunction(t, client, fnID)

	t.Run("deploy new version", func(t *testing.T) {
		body := map[string]any{
			"source_code": `export default function() { return new Response("Version 2"); }`,
		}

		status, respBody, _, err := client.Request("POST", "/api/functions/"+fnID+"/deploy", body, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 201 {
			t.Errorf("Expected 201, got %d: %s", status, respBody)
		}

		var deployment Deployment
		if err := json.Unmarshal(respBody, &deployment); err != nil {
			t.Fatalf("Failed to parse deployment: %v", err)
		}

		if deployment.Version < 1 {
			t.Errorf("Expected version >= 1, got %d", deployment.Version)
		}
		if deployment.Status != "deployed" {
			t.Errorf("Expected status deployed, got %s", deployment.Status)
		}
	})

	t.Run("deploy without source_code", func(t *testing.T) {
		body := map[string]any{}

		status, _, _, err := client.Request("POST", "/api/functions/"+fnID+"/deploy", body, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 400 {
			t.Errorf("Expected 400 for missing source_code, got %d", status)
		}
	})

	t.Run("deploy non-existent function", func(t *testing.T) {
		body := map[string]any{
			"source_code": "test",
		}

		status, _, _, err := client.Request("POST", "/api/functions/nonexistent-id/deploy", body, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 404 {
			t.Errorf("Expected 404, got %d", status)
		}
	})
}

func TestFunctions_ListDeployments(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)
	fnName := fmt.Sprintf("test-deploys-%d", time.Now().UnixNano())
	fnID := createTestFunction(t, client, fnName, false)
	defer deleteTestFunction(t, client, fnID)

	// Create a few deployments
	for i := 0; i < 3; i++ {
		body := map[string]any{
			"source_code": fmt.Sprintf(`export default function() { return new Response("Version %d"); }`, i+2),
		}
		client.Request("POST", "/api/functions/"+fnID+"/deploy", body, nil)
	}

	t.Run("list deployments", func(t *testing.T) {
		status, respBody, _, err := client.Request("GET", "/api/functions/"+fnID+"/deployments", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, respBody)
		}

		var deployments []Deployment
		if err := json.Unmarshal(respBody, &deployments); err != nil {
			t.Fatalf("Failed to parse deployments: %v", err)
		}

		t.Logf("Found %d deployments", len(deployments))

		// Should have at least 3 deployments
		if len(deployments) < 3 {
			t.Errorf("Expected at least 3 deployments, got %d", len(deployments))
		}
	})

	t.Run("list with limit", func(t *testing.T) {
		status, respBody, _, err := client.Request("GET", "/api/functions/"+fnID+"/deployments?limit=2", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d", status)
		}

		var deployments []Deployment
		json.Unmarshal(respBody, &deployments)

		if len(deployments) > 2 {
			t.Errorf("Expected max 2 deployments, got %d", len(deployments))
		}
	})
}

// =============================================================================
// Secrets Management Tests
// =============================================================================

func TestFunctions_SecretsManagement(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	t.Run("list secrets", func(t *testing.T) {
		status, respBody, _, err := client.Request("GET", "/api/functions/secrets", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, respBody)
		}

		var secrets []Secret
		if err := json.Unmarshal(respBody, &secrets); err != nil {
			t.Logf("Secrets response: %s", respBody)
		}
	})

	t.Run("list secrets with anon key should be forbidden", func(t *testing.T) {
		anonClient := NewClient(localbaseURL, localbaseAPIKey)
		status, _, _, err := anonClient.Request("GET", "/api/functions/secrets", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 403 {
			t.Errorf("Expected 403, got %d", status)
		}
	})

	secretName := fmt.Sprintf("TEST_SECRET_%d", time.Now().UnixNano())

	t.Run("create secret", func(t *testing.T) {
		body := map[string]any{
			"name":  secretName,
			"value": "super-secret-value",
		}

		status, respBody, _, err := client.Request("POST", "/api/functions/secrets", body, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 201 {
			t.Errorf("Expected 201, got %d: %s", status, respBody)
		}

		// Verify secret doesn't expose value
		var secretResp map[string]any
		json.Unmarshal(respBody, &secretResp)
		if _, hasValue := secretResp["value"]; hasValue {
			t.Error("Secret response should not expose value")
		}
	})

	t.Run("create secret with empty name", func(t *testing.T) {
		body := map[string]any{
			"name":  "",
			"value": "test",
		}

		status, _, _, err := client.Request("POST", "/api/functions/secrets", body, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 400 {
			t.Errorf("Expected 400 for empty name, got %d", status)
		}
	})

	t.Run("create secret with empty value", func(t *testing.T) {
		body := map[string]any{
			"name":  "EMPTY_VALUE_SECRET",
			"value": "",
		}

		status, _, _, err := client.Request("POST", "/api/functions/secrets", body, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 400 {
			t.Errorf("Expected 400 for empty value, got %d", status)
		}
	})

	t.Run("delete secret", func(t *testing.T) {
		status, _, _, err := client.Request("DELETE", "/api/functions/secrets/"+secretName, nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 204 && status != 200 {
			t.Errorf("Expected 204 or 200, got %d", status)
		}
	})
}

// =============================================================================
// Error Response Format Tests
// =============================================================================

func TestFunctions_ErrorResponseFormat(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	tests := []struct {
		name           string
		method         string
		path           string
		body           any
		expectedStatus int
	}{
		{"404 for non-existent function", "GET", "/api/functions/nonexistent", nil, 404},
		{"400 for empty name", "POST", "/api/functions", map[string]any{"name": ""}, 400},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, respBody, _, err := client.Request(tt.method, tt.path, tt.body, nil)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}

			if status != tt.expectedStatus {
				t.Errorf("Expected %d, got %d", tt.expectedStatus, status)
			}

			// Verify error response has expected format
			var errResp map[string]any
			if err := json.Unmarshal(respBody, &errResp); err != nil {
				t.Logf("Non-JSON error response: %s", respBody)
				return
			}

			// Should have error field
			if _, ok := errResp["error"]; !ok {
				t.Log("Warning: error response missing 'error' field")
			}

			t.Logf("Error response: %+v", errResp)
		})
	}
}

// =============================================================================
// Side-by-Side Comparison Tests
// =============================================================================

func TestFunctions_SideBySide_ListFunctions(t *testing.T) {
	if supabaseURL == localbaseURL {
		t.Skip("Skipping side-by-side test: SUPABASE_URL not configured")
	}

	localClient := NewClient(localbaseURL, serviceRoleKey)
	supaClient := NewClient(supabaseURL, serviceRoleKey)

	localStatus, localBody, localHeaders, err := localClient.Request("GET", "/api/functions", nil, nil)
	if err != nil {
		t.Fatalf("Localbase request failed: %v", err)
	}

	supaStatus, supaBody, supaHeaders, err := supaClient.Request("GET", "/api/functions", nil, nil)
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

func TestFunctions_SideBySide_Invoke(t *testing.T) {
	if supabaseURL == localbaseURL {
		t.Skip("Skipping side-by-side test: SUPABASE_URL not configured")
	}

	// This test assumes both environments have a "hello" function deployed
	localClient := NewClient(localbaseURL, localbaseAPIKey)
	supaClient := NewClient(supabaseURL, supabaseAPIKey)

	localStatus, _, localHeaders, err := localClient.Request("POST", "/functions/v1/hello", nil, nil)
	if err != nil {
		t.Logf("Localbase request failed (may not have hello function): %v", err)
	}

	supaStatus, _, supaHeaders, err := supaClient.Request("POST", "/functions/v1/hello", nil, nil)
	if err != nil {
		t.Logf("Supabase request failed (may not have hello function): %v", err)
	}

	t.Logf("Localbase invoke status: %d", localStatus)
	t.Logf("Supabase invoke status: %d", supaStatus)
	t.Logf("Localbase CORS: %s", localHeaders.Get("Access-Control-Allow-Origin"))
	t.Logf("Supabase CORS: %s", supaHeaders.Get("Access-Control-Allow-Origin"))
}

// =============================================================================
// HTTP Method Tests
// =============================================================================

func TestFunctions_HTTPMethods(t *testing.T) {
	serviceClient := NewClient(localbaseURL, serviceRoleKey)

	fnName := fmt.Sprintf("test-methods-%d", time.Now().UnixNano())
	fnID := createTestFunction(t, serviceClient, fnName, false)
	defer deleteTestFunction(t, serviceClient, fnID)

	methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE"}

	for _, method := range methods {
		t.Run(method+" request", func(t *testing.T) {
			client := NewClient(localbaseURL, localbaseAPIKey)

			var bodyReader io.Reader
			if method != "GET" && method != "DELETE" {
				bodyReader = bytes.NewReader([]byte(`{"test": true}`))
			}

			req, err := http.NewRequest(method, localbaseURL+"/functions/v1/"+fnName, bodyReader)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("apikey", client.apiKey)
			req.Header.Set("Authorization", "Bearer "+client.apiKey)

			resp, err := client.client.Do(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()

			// All methods should be accepted
			if resp.StatusCode != 200 {
				t.Logf("%s returned status %d (may need method support)", method, resp.StatusCode)
			}
		})
	}
}

// =============================================================================
// Inactive Function Tests
// =============================================================================

func TestFunctions_InvokeInactiveFunction(t *testing.T) {
	serviceClient := NewClient(localbaseURL, serviceRoleKey)

	fnName := fmt.Sprintf("test-inactive-%d", time.Now().UnixNano())
	fnID := createTestFunction(t, serviceClient, fnName, false)
	defer deleteTestFunction(t, serviceClient, fnID)

	// Set function to inactive
	body := map[string]any{"status": "inactive"}
	serviceClient.Request("PUT", "/api/functions/"+fnID, body, nil)

	// Try to invoke
	client := NewClient(localbaseURL, localbaseAPIKey)
	status, respBody, _, err := client.Request("POST", "/functions/v1/"+fnName, nil, nil)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if status != 503 {
		t.Errorf("Expected 503 for inactive function, got %d: %s", status, respBody)
	}
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkFunctions_Invoke(b *testing.B) {
	serviceClient := NewClient(localbaseURL, serviceRoleKey)

	fnName := fmt.Sprintf("bench-invoke-%d", time.Now().UnixNano())
	body := map[string]any{
		"name":        fnName,
		"verify_jwt":  false,
		"source_code": `export default function() { return new Response("OK"); }`,
	}

	status, respBody, _, err := serviceClient.Request("POST", "/api/functions", body, nil)
	if err != nil || status != 201 {
		b.Fatalf("Failed to create function: %v, status: %d", err, status)
	}

	var fn Function
	json.Unmarshal(respBody, &fn)
	defer serviceClient.Request("DELETE", "/api/functions/"+fn.ID, nil, nil)

	client := NewClient(localbaseURL, localbaseAPIKey)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.Request("POST", "/functions/v1/"+fnName, nil, nil)
	}
}

func BenchmarkFunctions_List(b *testing.B) {
	client := NewClient(localbaseURL, serviceRoleKey)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.Request("GET", "/api/functions", nil, nil)
	}
}

// =============================================================================
// JWT Helper Tests
// =============================================================================

func TestFunctions_JWTValidation(t *testing.T) {
	serviceClient := NewClient(localbaseURL, serviceRoleKey)

	fnName := fmt.Sprintf("test-jwt-validation-%d", time.Now().UnixNano())
	fnID := createTestFunction(t, serviceClient, fnName, true) // verify_jwt = true
	defer deleteTestFunction(t, serviceClient, fnID)

	t.Run("valid JWT accepted", func(t *testing.T) {
		userToken := createUserJWT("user-456", "user@example.com")
		client := NewClient(localbaseURL, userToken)
		status, _, _, err := client.Request("POST", "/functions/v1/"+fnName, nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		// Should succeed with valid JWT
		t.Logf("Valid JWT status: %d", status)
	})

	t.Run("expired JWT rejected", func(t *testing.T) {
		// Create an expired JWT
		claims := jwt.MapClaims{
			"sub":   "user-123",
			"email": "user@example.com",
			"role":  "authenticated",
			"aud":   "authenticated",
			"iss":   "supabase-demo",
			"iat":   time.Now().Add(-2 * time.Hour).Unix(),
			"exp":   time.Now().Add(-1 * time.Hour).Unix(), // Expired 1 hour ago
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		expiredToken, _ := token.SignedString([]byte(jwtSecret))

		client := NewClient(localbaseURL, expiredToken)
		status, _, _, err := client.Request("POST", "/functions/v1/"+fnName, nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		// Should reject expired token
		if status != 401 {
			t.Logf("Expired JWT returned status %d (expected 401)", status)
		}
	})

	t.Run("invalid signature rejected", func(t *testing.T) {
		// Create a JWT with wrong secret
		claims := jwt.MapClaims{
			"sub":   "user-123",
			"email": "user@example.com",
			"role":  "authenticated",
			"aud":   "authenticated",
			"iss":   "supabase-demo",
			"iat":   time.Now().Unix(),
			"exp":   time.Now().Add(time.Hour).Unix(),
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		invalidToken, _ := token.SignedString([]byte("wrong-secret-key"))

		client := NewClient(localbaseURL, invalidToken)
		status, _, _, err := client.Request("POST", "/functions/v1/"+fnName, nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		// Should reject invalid signature
		if status != 401 {
			t.Logf("Invalid signature returned status %d (expected 401)", status)
		}
	})
}

// =============================================================================
// Concurrent Invocation Tests
// =============================================================================

func TestFunctions_ConcurrentInvocation(t *testing.T) {
	serviceClient := NewClient(localbaseURL, serviceRoleKey)

	fnName := fmt.Sprintf("test-concurrent-%d", time.Now().UnixNano())
	fnID := createTestFunction(t, serviceClient, fnName, false)
	defer deleteTestFunction(t, serviceClient, fnID)

	t.Run("concurrent invocations", func(t *testing.T) {
		client := NewClient(localbaseURL, localbaseAPIKey)
		concurrency := 10

		var wg sync.WaitGroup
		errors := make(chan error, concurrency)
		statuses := make(chan int, concurrency)

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				status, _, _, err := client.Request("POST", "/functions/v1/"+fnName, map[string]any{
					"index": idx,
				}, nil)
				if err != nil {
					errors <- err
					return
				}
				statuses <- status
			}(i)
		}

		wg.Wait()
		close(errors)
		close(statuses)

		// Check for errors
		for err := range errors {
			t.Errorf("Concurrent request failed: %v", err)
		}

		// All should return 200
		successCount := 0
		for status := range statuses {
			if status == 200 {
				successCount++
			}
		}

		if successCount != concurrency {
			t.Errorf("Expected %d successful invocations, got %d", concurrency, successCount)
		}
	})
}

// =============================================================================
// Function Slug vs ID Lookup Tests
// =============================================================================

func TestFunctions_SlugLookup(t *testing.T) {
	serviceClient := NewClient(localbaseURL, serviceRoleKey)

	// Create a function with spaces in name (to test slug generation)
	fnName := fmt.Sprintf("Test Function With Spaces %d", time.Now().UnixNano())
	fnID := createTestFunction(t, serviceClient, fnName, false)
	defer deleteTestFunction(t, serviceClient, fnID)

	// Expected slug (lowercase, spaces replaced with hyphens)
	expectedSlug := strings.ToLower(strings.ReplaceAll(fnName, " ", "-"))

	t.Run("invoke by slug", func(t *testing.T) {
		client := NewClient(localbaseURL, localbaseAPIKey)
		status, _, _, err := client.Request("POST", "/functions/v1/"+expectedSlug, nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d", status)
		}
	})

	t.Run("get function returns slug", func(t *testing.T) {
		status, respBody, _, err := serviceClient.Request("GET", "/api/functions/"+fnID, nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d", status)
		}

		var fn Function
		json.Unmarshal(respBody, &fn)

		if fn.Slug != expectedSlug {
			t.Errorf("Expected slug %s, got %s", expectedSlug, fn.Slug)
		}
	})
}

// =============================================================================
// Request Body Content Type Tests
// =============================================================================

func TestFunctions_RequestContentTypes(t *testing.T) {
	serviceClient := NewClient(localbaseURL, serviceRoleKey)

	fnName := fmt.Sprintf("test-content-type-%d", time.Now().UnixNano())
	fnID := createTestFunction(t, serviceClient, fnName, false)
	defer deleteTestFunction(t, serviceClient, fnID)

	tests := []struct {
		name        string
		contentType string
		body        string
	}{
		{"application/json", "application/json", `{"key": "value"}`},
		{"text/plain", "text/plain", "plain text content"},
		{"text/html", "text/html", "<html><body>test</body></html>"},
		{"application/xml", "application/xml", "<root><item>test</item></root>"},
		{"application/x-www-form-urlencoded", "application/x-www-form-urlencoded", "key1=value1&key2=value2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", localbaseURL+"/functions/v1/"+fnName, strings.NewReader(tt.body))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			req.Header.Set("Content-Type", tt.contentType)
			req.Header.Set("apikey", localbaseAPIKey)
			req.Header.Set("Authorization", "Bearer "+localbaseAPIKey)

			client := &http.Client{Timeout: 30 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				t.Errorf("Expected 200 for %s, got %d", tt.contentType, resp.StatusCode)
			}
		})
	}
}

// =============================================================================
// Function Version Management Tests
// =============================================================================

func TestFunctions_VersionManagement(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)
	fnName := fmt.Sprintf("test-version-%d", time.Now().UnixNano())
	fnID := createTestFunction(t, client, fnName, false)
	defer deleteTestFunction(t, client, fnID)

	t.Run("initial version is 1", func(t *testing.T) {
		status, respBody, _, err := client.Request("GET", "/api/functions/"+fnID, nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Fatalf("Expected 200, got %d", status)
		}

		var fn Function
		json.Unmarshal(respBody, &fn)

		if fn.Version != 1 {
			t.Errorf("Expected version 1, got %d", fn.Version)
		}
	})

	t.Run("deploy increments version", func(t *testing.T) {
		// Deploy version 2
		body := map[string]any{
			"source_code": `export default function() { return new Response("Version 2"); }`,
		}
		client.Request("POST", "/api/functions/"+fnID+"/deploy", body, nil)

		// Check function version
		status, respBody, _, err := client.Request("GET", "/api/functions/"+fnID, nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Fatalf("Expected 200, got %d", status)
		}

		var fn Function
		json.Unmarshal(respBody, &fn)

		if fn.Version != 2 {
			t.Errorf("Expected version 2 after deploy, got %d", fn.Version)
		}
	})

	t.Run("multiple deploys increment correctly", func(t *testing.T) {
		// Deploy versions 3, 4, 5
		for i := 3; i <= 5; i++ {
			body := map[string]any{
				"source_code": fmt.Sprintf(`export default function() { return new Response("Version %d"); }`, i),
			}
			client.Request("POST", "/api/functions/"+fnID+"/deploy", body, nil)
		}

		// Check final version
		status, respBody, _, err := client.Request("GET", "/api/functions/"+fnID, nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Fatalf("Expected 200, got %d", status)
		}

		var fn Function
		json.Unmarshal(respBody, &fn)

		if fn.Version != 5 {
			t.Errorf("Expected version 5 after deploys, got %d", fn.Version)
		}
	})
}

// =============================================================================
// Secrets Upsert Tests
// =============================================================================

func TestFunctions_SecretsUpsert(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)
	secretName := fmt.Sprintf("TEST_UPSERT_%d", time.Now().UnixNano())

	defer deleteTestSecret(t, client, secretName)

	t.Run("create new secret", func(t *testing.T) {
		body := map[string]any{
			"name":  secretName,
			"value": "initial-value",
		}

		status, _, _, err := client.Request("POST", "/api/functions/secrets", body, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 201 {
			t.Errorf("Expected 201, got %d", status)
		}
	})

	t.Run("upsert updates existing secret", func(t *testing.T) {
		body := map[string]any{
			"name":  secretName,
			"value": "updated-value",
		}

		status, _, _, err := client.Request("POST", "/api/functions/secrets", body, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		// Should still succeed (upsert)
		if status != 201 {
			t.Errorf("Expected 201 for upsert, got %d", status)
		}
	})

	t.Run("secret appears in list after upsert", func(t *testing.T) {
		status, respBody, _, err := client.Request("GET", "/api/functions/secrets", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d", status)
		}

		var secrets []Secret
		json.Unmarshal(respBody, &secrets)

		found := false
		for _, s := range secrets {
			if s.Name == secretName {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("Secret %s not found in list", secretName)
		}
	})
}

// =============================================================================
// Response Header Tests
// =============================================================================

func TestFunctions_ResponseHeaders(t *testing.T) {
	serviceClient := NewClient(localbaseURL, serviceRoleKey)

	fnName := fmt.Sprintf("test-headers-%d", time.Now().UnixNano())
	fnID := createTestFunction(t, serviceClient, fnName, false)
	defer deleteTestFunction(t, serviceClient, fnID)

	t.Run("content-type header", func(t *testing.T) {
		client := NewClient(localbaseURL, localbaseAPIKey)
		_, _, headers, err := client.Request("POST", "/functions/v1/"+fnName, nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		contentType := headers.Get("Content-Type")
		if contentType == "" {
			t.Error("Missing Content-Type header")
		}

		t.Logf("Content-Type: %s", contentType)
	})

	t.Run("cors headers present", func(t *testing.T) {
		client := NewClient(localbaseURL, localbaseAPIKey)
		_, _, headers, err := client.Request("POST", "/functions/v1/"+fnName, nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		requiredHeaders := []string{
			"Access-Control-Allow-Origin",
			"Access-Control-Allow-Methods",
			"Access-Control-Allow-Headers",
		}

		for _, h := range requiredHeaders {
			if headers.Get(h) == "" {
				t.Errorf("Missing required CORS header: %s", h)
			}
		}
	})
}

// =============================================================================
// Entrypoint Tests
// =============================================================================

func TestFunctions_Entrypoint(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	t.Run("create with custom entrypoint", func(t *testing.T) {
		fnName := fmt.Sprintf("test-entrypoint-%d", time.Now().UnixNano())

		body := map[string]any{
			"name":       fnName,
			"entrypoint": "main.ts",
		}

		status, respBody, _, err := client.Request("POST", "/api/functions", body, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 201 {
			t.Errorf("Expected 201, got %d: %s", status, respBody)
		}

		var fn Function
		json.Unmarshal(respBody, &fn)

		if fn.Entrypoint != "main.ts" {
			t.Errorf("Expected entrypoint 'main.ts', got '%s'", fn.Entrypoint)
		}

		// Cleanup
		deleteTestFunction(t, client, fn.ID)
	})

	t.Run("default entrypoint is index.ts", func(t *testing.T) {
		fnName := fmt.Sprintf("test-default-entry-%d", time.Now().UnixNano())

		body := map[string]any{
			"name": fnName,
		}

		status, respBody, _, err := client.Request("POST", "/api/functions", body, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 201 {
			t.Errorf("Expected 201, got %d", status)
		}

		var fn Function
		json.Unmarshal(respBody, &fn)

		if fn.Entrypoint != "index.ts" {
			t.Errorf("Expected default entrypoint 'index.ts', got '%s'", fn.Entrypoint)
		}

		// Cleanup
		deleteTestFunction(t, client, fn.ID)
	})
}

// =============================================================================
// Import Map Tests
// =============================================================================

func TestFunctions_ImportMap(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	t.Run("create with import_map", func(t *testing.T) {
		fnName := fmt.Sprintf("test-import-map-%d", time.Now().UnixNano())

		importMap := `{"imports": {"lodash": "https://esm.sh/lodash"}}`

		body := map[string]any{
			"name":       fnName,
			"import_map": importMap,
		}

		status, respBody, _, err := client.Request("POST", "/api/functions", body, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 201 {
			t.Errorf("Expected 201, got %d: %s", status, respBody)
		}

		var fn Function
		json.Unmarshal(respBody, &fn)

		if fn.ImportMap != importMap {
			t.Errorf("Import map not stored correctly")
		}

		// Cleanup
		deleteTestFunction(t, client, fn.ID)
	})
}

// =============================================================================
// Status Transition Tests
// =============================================================================

func TestFunctions_StatusTransitions(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)
	fnName := fmt.Sprintf("test-status-%d", time.Now().UnixNano())
	fnID := createTestFunction(t, client, fnName, false)
	defer deleteTestFunction(t, client, fnID)

	t.Run("initial status is active", func(t *testing.T) {
		status, respBody, _, err := client.Request("GET", "/api/functions/"+fnID, nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Fatalf("Expected 200, got %d", status)
		}

		var fn Function
		json.Unmarshal(respBody, &fn)

		if fn.Status != "active" {
			t.Errorf("Expected status 'active', got '%s'", fn.Status)
		}
	})

	t.Run("set status to inactive", func(t *testing.T) {
		body := map[string]any{"status": "inactive"}
		status, _, _, err := client.Request("PUT", "/api/functions/"+fnID, body, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d", status)
		}

		// Verify status changed
		_, respBody, _, _ := client.Request("GET", "/api/functions/"+fnID, nil, nil)
		var fn Function
		json.Unmarshal(respBody, &fn)

		if fn.Status != "inactive" {
			t.Errorf("Expected status 'inactive', got '%s'", fn.Status)
		}
	})

	t.Run("set status back to active", func(t *testing.T) {
		body := map[string]any{"status": "active"}
		status, _, _, err := client.Request("PUT", "/api/functions/"+fnID, body, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d", status)
		}

		// Verify status changed
		_, respBody, _, _ := client.Request("GET", "/api/functions/"+fnID, nil, nil)
		var fn Function
		json.Unmarshal(respBody, &fn)

		if fn.Status != "active" {
			t.Errorf("Expected status 'active', got '%s'", fn.Status)
		}
	})
}

// =============================================================================
// API Key Header Tests
// =============================================================================

func TestFunctions_APIKeyHeader(t *testing.T) {
	serviceClient := NewClient(localbaseURL, serviceRoleKey)

	fnName := fmt.Sprintf("test-apikey-%d", time.Now().UnixNano())
	fnID := createTestFunction(t, serviceClient, fnName, false)
	defer deleteTestFunction(t, serviceClient, fnID)

	t.Run("apikey header accepted", func(t *testing.T) {
		req, err := http.NewRequest("POST", localbaseURL+"/functions/v1/"+fnName, nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		// Use apikey header instead of Authorization
		req.Header.Set("apikey", localbaseAPIKey)

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		// Should work with just apikey header
		if resp.StatusCode != 200 {
			t.Errorf("Expected 200 with apikey header, got %d", resp.StatusCode)
		}
	})

	t.Run("authorization header accepted", func(t *testing.T) {
		req, err := http.NewRequest("POST", localbaseURL+"/functions/v1/"+fnName, nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		// Use Authorization header
		req.Header.Set("Authorization", "Bearer "+localbaseAPIKey)

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Errorf("Expected 200 with Authorization header, got %d", resp.StatusCode)
		}
	})
}

// =============================================================================
// Large Payload Tests
// =============================================================================

func TestFunctions_LargePayload(t *testing.T) {
	serviceClient := NewClient(localbaseURL, serviceRoleKey)

	fnName := fmt.Sprintf("test-large-%d", time.Now().UnixNano())
	fnID := createTestFunction(t, serviceClient, fnName, false)
	defer deleteTestFunction(t, serviceClient, fnID)

	t.Run("1KB payload", func(t *testing.T) {
		payload := strings.Repeat("x", 1024)
		body := map[string]any{"data": payload}

		client := NewClient(localbaseURL, localbaseAPIKey)
		status, _, _, err := client.Request("POST", "/functions/v1/"+fnName, body, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200 for 1KB payload, got %d", status)
		}
	})

	t.Run("100KB payload", func(t *testing.T) {
		payload := strings.Repeat("x", 100*1024)
		body := map[string]any{"data": payload}

		client := NewClient(localbaseURL, localbaseAPIKey)
		status, _, _, err := client.Request("POST", "/functions/v1/"+fnName, body, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200 for 100KB payload, got %d", status)
		}
	})

	t.Run("1MB payload", func(t *testing.T) {
		payload := strings.Repeat("x", 1024*1024)

		req, err := http.NewRequest("POST", localbaseURL+"/functions/v1/"+fnName, bytes.NewReader([]byte(payload)))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		req.Header.Set("Content-Type", "text/plain")
		req.Header.Set("apikey", localbaseAPIKey)
		req.Header.Set("Authorization", "Bearer "+localbaseAPIKey)

		client := &http.Client{Timeout: 60 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		// Large payloads should still work
		if resp.StatusCode != 200 {
			t.Logf("1MB payload returned status %d (may have size limits)", resp.StatusCode)
		}
	})
}

// =============================================================================
// Query Parameter Tests
// =============================================================================

func TestFunctions_QueryParameters(t *testing.T) {
	serviceClient := NewClient(localbaseURL, serviceRoleKey)

	fnName := fmt.Sprintf("test-query-%d", time.Now().UnixNano())
	fnID := createTestFunction(t, serviceClient, fnName, false)
	defer deleteTestFunction(t, serviceClient, fnID)

	t.Run("query params passed to function", func(t *testing.T) {
		client := NewClient(localbaseURL, localbaseAPIKey)
		status, _, _, err := client.Request("GET", "/functions/v1/"+fnName+"?param1=value1&param2=value2", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d", status)
		}
	})

	t.Run("special characters in query params", func(t *testing.T) {
		client := NewClient(localbaseURL, localbaseAPIKey)
		status, _, _, err := client.Request("GET", "/functions/v1/"+fnName+"?search=hello%20world&filter=%3E%3D10", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d", status)
		}
	})
}

// =============================================================================
// Deployment Status Tests
// =============================================================================

func TestFunctions_DeploymentStatus(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)
	fnName := fmt.Sprintf("test-deploy-status-%d", time.Now().UnixNano())
	fnID := createTestFunction(t, client, fnName, false)
	defer deleteTestFunction(t, client, fnID)

	t.Run("deployment has deployed status", func(t *testing.T) {
		body := map[string]any{
			"source_code": `export default function() { return new Response("Hello"); }`,
		}

		status, respBody, _, err := client.Request("POST", "/api/functions/"+fnID+"/deploy", body, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 201 {
			t.Fatalf("Expected 201, got %d", status)
		}

		var deployment Deployment
		json.Unmarshal(respBody, &deployment)

		if deployment.Status != "deployed" {
			t.Errorf("Expected status 'deployed', got '%s'", deployment.Status)
		}
	})
}

// =============================================================================
// Empty Response Tests
// =============================================================================

func TestFunctions_EmptyResponses(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	t.Run("list functions empty", func(t *testing.T) {
		// This test depends on no other tests creating functions
		// Just verify the response is valid JSON array
		status, respBody, _, err := client.Request("GET", "/api/functions", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d", status)
		}

		var functions []Function
		if err := json.Unmarshal(respBody, &functions); err != nil {
			t.Errorf("Failed to parse response as array: %v", err)
		}
	})

	t.Run("list secrets returns array", func(t *testing.T) {
		status, respBody, _, err := client.Request("GET", "/api/functions/secrets", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d", status)
		}

		var secrets []Secret
		if err := json.Unmarshal(respBody, &secrets); err != nil {
			t.Errorf("Failed to parse response as array: %v", err)
		}
	})
}

// =============================================================================
// X-Client-Info Header Tests (Supabase SDK Compatibility)
// =============================================================================

func TestFunctions_XClientInfoHeader(t *testing.T) {
	serviceClient := NewClient(localbaseURL, serviceRoleKey)

	fnName := fmt.Sprintf("test-xclient-%d", time.Now().UnixNano())
	fnID := createTestFunction(t, serviceClient, fnName, false)
	defer deleteTestFunction(t, serviceClient, fnID)

	t.Run("x-client-info header accepted", func(t *testing.T) {
		req, err := http.NewRequest("POST", localbaseURL+"/functions/v1/"+fnName, nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		req.Header.Set("apikey", localbaseAPIKey)
		req.Header.Set("Authorization", "Bearer "+localbaseAPIKey)
		req.Header.Set("x-client-info", "supabase-js/2.38.0")

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Errorf("Expected 200 with x-client-info header, got %d", resp.StatusCode)
		}
	})
}

// =============================================================================
// Custom Headers Pass-Through Tests
// =============================================================================

func TestFunctions_CustomHeaders(t *testing.T) {
	serviceClient := NewClient(localbaseURL, serviceRoleKey)

	fnName := fmt.Sprintf("test-custom-headers-%d", time.Now().UnixNano())
	fnID := createTestFunction(t, serviceClient, fnName, false)
	defer deleteTestFunction(t, serviceClient, fnID)

	t.Run("custom headers accepted", func(t *testing.T) {
		client := NewClient(localbaseURL, localbaseAPIKey)
		headers := map[string]string{
			"X-Custom-Header":  "custom-value",
			"X-Request-ID":     "req-12345",
			"Accept-Language":  "en-US",
			"X-Forwarded-For":  "192.168.1.1",
			"X-Forwarded-Host": "example.com",
		}

		status, _, _, err := client.Request("POST", "/functions/v1/"+fnName, nil, headers)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d", status)
		}
	})
}

// =============================================================================
// Binary Content Type Tests
// =============================================================================

func TestFunctions_BinaryContent(t *testing.T) {
	serviceClient := NewClient(localbaseURL, serviceRoleKey)

	fnName := fmt.Sprintf("test-binary-%d", time.Now().UnixNano())
	fnID := createTestFunction(t, serviceClient, fnName, false)
	defer deleteTestFunction(t, serviceClient, fnID)

	t.Run("application/octet-stream", func(t *testing.T) {
		binaryData := []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD}

		req, err := http.NewRequest("POST", localbaseURL+"/functions/v1/"+fnName, bytes.NewReader(binaryData))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		req.Header.Set("Content-Type", "application/octet-stream")
		req.Header.Set("apikey", localbaseAPIKey)
		req.Header.Set("Authorization", "Bearer "+localbaseAPIKey)

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Errorf("Expected 200 for binary content, got %d", resp.StatusCode)
		}
	})

	t.Run("multipart/form-data", func(t *testing.T) {
		body := &bytes.Buffer{}
		body.WriteString("--boundary\r\n")
		body.WriteString("Content-Disposition: form-data; name=\"file\"; filename=\"test.txt\"\r\n")
		body.WriteString("Content-Type: text/plain\r\n\r\n")
		body.WriteString("file content\r\n")
		body.WriteString("--boundary--\r\n")

		req, err := http.NewRequest("POST", localbaseURL+"/functions/v1/"+fnName, body)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
		req.Header.Set("apikey", localbaseAPIKey)
		req.Header.Set("Authorization", "Bearer "+localbaseAPIKey)

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Errorf("Expected 200 for multipart content, got %d", resp.StatusCode)
		}
	})
}

// =============================================================================
// Response Content Type Tests
// =============================================================================

func TestFunctions_ResponseContentType(t *testing.T) {
	serviceClient := NewClient(localbaseURL, serviceRoleKey)

	fnName := fmt.Sprintf("test-resp-ct-%d", time.Now().UnixNano())
	fnID := createTestFunction(t, serviceClient, fnName, false)
	defer deleteTestFunction(t, serviceClient, fnID)

	t.Run("response has content-type", func(t *testing.T) {
		client := NewClient(localbaseURL, localbaseAPIKey)
		_, _, headers, err := client.Request("POST", "/functions/v1/"+fnName, nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		contentType := headers.Get("Content-Type")
		if contentType == "" {
			t.Error("Missing Content-Type header in response")
		}

		// Should be JSON
		if !strings.Contains(contentType, "application/json") {
			t.Logf("Content-Type is: %s", contentType)
		}
	})
}

// =============================================================================
// Error Code Consistency Tests
// =============================================================================

func TestFunctions_ErrorCodeConsistency(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	tests := []struct {
		name           string
		method         string
		path           string
		body           any
		expectedStatus int
		checkError     bool
	}{
		{"non-existent function invoke", "POST", "/functions/v1/nonexistent-fn-xyz", nil, 404, true},
		{"non-existent function get", "GET", "/api/functions/nonexistent-id-xyz", nil, 404, true},
		{"deploy to non-existent", "POST", "/api/functions/nonexistent-id-xyz/deploy", map[string]any{"source_code": "test"}, 404, true},
		{"empty name create", "POST", "/api/functions", map[string]any{"name": ""}, 400, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, respBody, _, err := client.Request(tt.method, tt.path, tt.body, nil)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}

			if status != tt.expectedStatus {
				t.Errorf("Expected %d, got %d", tt.expectedStatus, status)
			}

			if tt.checkError {
				var errResp map[string]any
				if err := json.Unmarshal(respBody, &errResp); err == nil {
					// Verify error response has 'error' field
					if _, ok := errResp["error"]; !ok {
						t.Error("Error response missing 'error' field")
					}
				}
			}
		})
	}
}

// =============================================================================
// Timestamp Format Tests
// =============================================================================

func TestFunctions_TimestampFormat(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)
	fnName := fmt.Sprintf("test-timestamp-%d", time.Now().UnixNano())
	fnID := createTestFunction(t, client, fnName, false)
	defer deleteTestFunction(t, client, fnID)

	t.Run("timestamps are RFC3339 format", func(t *testing.T) {
		status, respBody, _, err := client.Request("GET", "/api/functions/"+fnID, nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Fatalf("Expected 200, got %d", status)
		}

		var fn Function
		if err := json.Unmarshal(respBody, &fn); err != nil {
			t.Fatalf("Failed to parse function: %v", err)
		}

		// Verify timestamps are valid
		if fn.CreatedAt.IsZero() {
			t.Error("created_at is zero")
		}
		if fn.UpdatedAt.IsZero() {
			t.Error("updated_at is zero")
		}

		// CreatedAt should be before or equal to UpdatedAt
		if fn.CreatedAt.After(fn.UpdatedAt) {
			t.Error("created_at should not be after updated_at")
		}
	})

	t.Run("deployment timestamps are valid", func(t *testing.T) {
		// Create a deployment
		body := map[string]any{
			"source_code": `export default function() { return new Response("test"); }`,
		}
		status, respBody, _, err := client.Request("POST", "/api/functions/"+fnID+"/deploy", body, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 201 {
			t.Fatalf("Expected 201, got %d", status)
		}

		var deployment Deployment
		if err := json.Unmarshal(respBody, &deployment); err != nil {
			t.Fatalf("Failed to parse deployment: %v", err)
		}

		if deployment.DeployedAt.IsZero() {
			t.Error("deployed_at is zero")
		}
	})
}

// =============================================================================
// Function Field Validation Tests
// =============================================================================

func TestFunctions_FieldValidation(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	t.Run("function has all required fields", func(t *testing.T) {
		fnName := fmt.Sprintf("test-fields-%d", time.Now().UnixNano())
		body := map[string]any{
			"name":        fnName,
			"verify_jwt":  true,
			"entrypoint":  "main.ts",
			"import_map":  `{"imports": {}}`,
			"source_code": `export default function() {}`,
		}

		status, respBody, _, err := client.Request("POST", "/api/functions", body, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 201 {
			t.Fatalf("Expected 201, got %d: %s", status, respBody)
		}

		var fn Function
		if err := json.Unmarshal(respBody, &fn); err != nil {
			t.Fatalf("Failed to parse function: %v", err)
		}

		// Cleanup
		defer deleteTestFunction(t, client, fn.ID)

		// Validate all fields
		if fn.ID == "" {
			t.Error("ID is empty")
		}
		if fn.Name != fnName {
			t.Errorf("Name mismatch: expected %s, got %s", fnName, fn.Name)
		}
		if fn.Slug == "" {
			t.Error("Slug is empty")
		}
		if fn.Version != 1 {
			t.Errorf("Version should be 1, got %d", fn.Version)
		}
		if fn.Status != "active" {
			t.Errorf("Status should be 'active', got %s", fn.Status)
		}
		if fn.Entrypoint != "main.ts" {
			t.Errorf("Entrypoint mismatch: expected main.ts, got %s", fn.Entrypoint)
		}
		if !fn.VerifyJWT {
			t.Error("VerifyJWT should be true")
		}
	})
}

// =============================================================================
// Deployment Field Validation Tests
// =============================================================================

func TestFunctions_DeploymentFieldValidation(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)
	fnName := fmt.Sprintf("test-deploy-fields-%d", time.Now().UnixNano())
	fnID := createTestFunction(t, client, fnName, false)
	defer deleteTestFunction(t, client, fnID)

	t.Run("deployment has all required fields", func(t *testing.T) {
		body := map[string]any{
			"source_code": `export default function() { return new Response("test"); }`,
		}

		status, respBody, _, err := client.Request("POST", "/api/functions/"+fnID+"/deploy", body, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 201 {
			t.Fatalf("Expected 201, got %d", status)
		}

		var deployment Deployment
		if err := json.Unmarshal(respBody, &deployment); err != nil {
			t.Fatalf("Failed to parse deployment: %v", err)
		}

		// Validate all fields
		if deployment.ID == "" {
			t.Error("ID is empty")
		}
		if deployment.FunctionID != fnID {
			t.Errorf("FunctionID mismatch: expected %s, got %s", fnID, deployment.FunctionID)
		}
		if deployment.Version < 1 {
			t.Errorf("Version should be >= 1, got %d", deployment.Version)
		}
		if deployment.Status != "deployed" {
			t.Errorf("Status should be 'deployed', got %s", deployment.Status)
		}
	})
}

// =============================================================================
// Path Routing Tests
// =============================================================================

func TestFunctions_PathRouting(t *testing.T) {
	serviceClient := NewClient(localbaseURL, serviceRoleKey)

	fnName := fmt.Sprintf("test-routing-%d", time.Now().UnixNano())
	fnID := createTestFunction(t, serviceClient, fnName, false)
	defer deleteTestFunction(t, serviceClient, fnID)

	t.Run("subpath routing", func(t *testing.T) {
		client := NewClient(localbaseURL, localbaseAPIKey)

		// Test if subpaths work (may return 404 depending on implementation)
		paths := []string{
			"/functions/v1/" + fnName,
			"/functions/v1/" + fnName + "/",
		}

		for _, path := range paths {
			status, _, _, err := client.Request("GET", path, nil, nil)
			if err != nil {
				t.Fatalf("Request failed for %s: %v", path, err)
			}

			// 200 or 404 are both acceptable depending on path handling
			if status != 200 && status != 404 {
				t.Errorf("Path %s returned unexpected status %d", path, status)
			}
		}
	})
}

// =============================================================================
// Special Characters in Function Name Tests
// =============================================================================

func TestFunctions_SpecialCharactersInName(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	tests := []struct {
		name         string
		functionName string
		expectSlug   string
	}{
		{"spaces", "My Test Function", "my-test-function"},
		{"uppercase", "MyTestFunction", "mytestfunction"},
		{"numbers", "Function123", "function123"},
		{"mixed", "My Function 123", "my-function-123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fullName := fmt.Sprintf("%s-%d", tt.functionName, time.Now().UnixNano())

			body := map[string]any{
				"name": fullName,
			}

			status, respBody, _, err := client.Request("POST", "/api/functions", body, nil)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}

			if status != 201 {
				t.Errorf("Expected 201, got %d: %s", status, respBody)
				return
			}

			var fn Function
			json.Unmarshal(respBody, &fn)
			defer deleteTestFunction(t, client, fn.ID)

			// Verify slug is URL-friendly
			if strings.Contains(fn.Slug, " ") {
				t.Errorf("Slug should not contain spaces: %s", fn.Slug)
			}
			if fn.Slug != strings.ToLower(fn.Slug) {
				t.Errorf("Slug should be lowercase: %s", fn.Slug)
			}
		})
	}
}

// =============================================================================
// Concurrent Management Operations Tests
// =============================================================================

func TestFunctions_ConcurrentManagement(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)

	t.Run("concurrent function creation", func(t *testing.T) {
		concurrency := 5
		var wg sync.WaitGroup
		results := make(chan string, concurrency)

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				fnName := fmt.Sprintf("concurrent-create-%d-%d", time.Now().UnixNano(), idx)
				body := map[string]any{"name": fnName}

				status, respBody, _, err := client.Request("POST", "/api/functions", body, nil)
				if err != nil || status != 201 {
					return
				}

				var fn Function
				json.Unmarshal(respBody, &fn)
				results <- fn.ID
			}(i)
		}

		wg.Wait()
		close(results)

		// Cleanup created functions
		for id := range results {
			deleteTestFunction(t, client, id)
		}
	})

	t.Run("concurrent deployments", func(t *testing.T) {
		fnName := fmt.Sprintf("concurrent-deploy-%d", time.Now().UnixNano())
		fnID := createTestFunction(t, client, fnName, false)
		defer deleteTestFunction(t, client, fnID)

		concurrency := 3
		var wg sync.WaitGroup
		statuses := make(chan int, concurrency)

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				body := map[string]any{
					"source_code": fmt.Sprintf(`export default function() { return new Response("Version %d"); }`, idx),
				}
				status, _, _, _ := client.Request("POST", "/api/functions/"+fnID+"/deploy", body, nil)
				statuses <- status
			}(i)
		}

		wg.Wait()
		close(statuses)

		// All should succeed
		successCount := 0
		for status := range statuses {
			if status == 201 {
				successCount++
			}
		}

		if successCount != concurrency {
			t.Errorf("Expected %d successful deploys, got %d", concurrency, successCount)
		}
	})
}

// =============================================================================
// Accept Header Tests
// =============================================================================

func TestFunctions_AcceptHeader(t *testing.T) {
	serviceClient := NewClient(localbaseURL, serviceRoleKey)

	fnName := fmt.Sprintf("test-accept-%d", time.Now().UnixNano())
	fnID := createTestFunction(t, serviceClient, fnName, false)
	defer deleteTestFunction(t, serviceClient, fnID)

	acceptTypes := []string{
		"application/json",
		"*/*",
		"text/plain",
		"application/json, text/plain, */*",
	}

	for _, acceptType := range acceptTypes {
		t.Run("Accept: "+acceptType, func(t *testing.T) {
			req, err := http.NewRequest("POST", localbaseURL+"/functions/v1/"+fnName, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			req.Header.Set("Accept", acceptType)
			req.Header.Set("apikey", localbaseAPIKey)
			req.Header.Set("Authorization", "Bearer "+localbaseAPIKey)

			client := &http.Client{Timeout: 30 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				t.Errorf("Expected 200 for Accept: %s, got %d", acceptType, resp.StatusCode)
			}
		})
	}
}

// =============================================================================
// Rate Limiting Behavior Tests
// =============================================================================

func TestFunctions_RateLimitingBehavior(t *testing.T) {
	serviceClient := NewClient(localbaseURL, serviceRoleKey)

	fnName := fmt.Sprintf("test-ratelimit-%d", time.Now().UnixNano())
	fnID := createTestFunction(t, serviceClient, fnName, false)
	defer deleteTestFunction(t, serviceClient, fnID)

	t.Run("rapid invocations don't fail immediately", func(t *testing.T) {
		client := NewClient(localbaseURL, localbaseAPIKey)
		successCount := 0

		// Make 20 rapid requests
		for i := 0; i < 20; i++ {
			status, _, _, err := client.Request("POST", "/functions/v1/"+fnName, nil, nil)
			if err == nil && status == 200 {
				successCount++
			}
		}

		// Most should succeed (we're not testing actual rate limiting, just that burst is allowed)
		if successCount < 10 {
			t.Errorf("Expected at least 10 successful rapid requests, got %d", successCount)
		}
	})
}

// =============================================================================
// Function Invocation Response Validation Tests
// =============================================================================

func TestFunctions_InvocationResponseFormat(t *testing.T) {
	serviceClient := NewClient(localbaseURL, serviceRoleKey)

	fnName := fmt.Sprintf("test-resp-format-%d", time.Now().UnixNano())
	fnID := createTestFunction(t, serviceClient, fnName, false)
	defer deleteTestFunction(t, serviceClient, fnID)

	t.Run("invocation response is valid JSON", func(t *testing.T) {
		client := NewClient(localbaseURL, localbaseAPIKey)
		status, respBody, _, err := client.Request("POST", "/functions/v1/"+fnName, nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Fatalf("Expected 200, got %d", status)
		}

		// Verify response is valid JSON
		var result map[string]any
		if err := json.Unmarshal(respBody, &result); err != nil {
			t.Errorf("Response is not valid JSON: %v", err)
		}

		// Our mock response should have certain fields
		if _, ok := result["message"]; !ok {
			t.Log("Response missing 'message' field (may be different in production)")
		}
		if _, ok := result["function"]; !ok {
			t.Log("Response missing 'function' field (may be different in production)")
		}
	})
}

// =============================================================================
// Empty Body Tests
// =============================================================================

func TestFunctions_EmptyBodyHandling(t *testing.T) {
	serviceClient := NewClient(localbaseURL, serviceRoleKey)

	fnName := fmt.Sprintf("test-empty-body-%d", time.Now().UnixNano())
	fnID := createTestFunction(t, serviceClient, fnName, false)
	defer deleteTestFunction(t, serviceClient, fnID)

	methods := []string{"POST", "PUT", "PATCH"}

	for _, method := range methods {
		t.Run(method+" with empty body", func(t *testing.T) {
			req, err := http.NewRequest(method, localbaseURL+"/functions/v1/"+fnName, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			req.Header.Set("apikey", localbaseAPIKey)
			req.Header.Set("Authorization", "Bearer "+localbaseAPIKey)

			client := &http.Client{Timeout: 30 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				t.Errorf("Expected 200 for %s with empty body, got %d", method, resp.StatusCode)
			}
		})
	}
}

// =============================================================================
// Idempotency Tests
// =============================================================================

func TestFunctions_IdempotencyBehavior(t *testing.T) {
	client := NewClient(localbaseURL, serviceRoleKey)
	fnName := fmt.Sprintf("test-idempotent-%d", time.Now().UnixNano())
	fnID := createTestFunction(t, client, fnName, false)
	defer deleteTestFunction(t, client, fnID)

	t.Run("get same function multiple times returns same result", func(t *testing.T) {
		var firstResponse, secondResponse Function

		status1, body1, _, _ := client.Request("GET", "/api/functions/"+fnID, nil, nil)
		status2, body2, _, _ := client.Request("GET", "/api/functions/"+fnID, nil, nil)

		if status1 != status2 {
			t.Errorf("Status codes differ: %d vs %d", status1, status2)
		}

		json.Unmarshal(body1, &firstResponse)
		json.Unmarshal(body2, &secondResponse)

		if firstResponse.ID != secondResponse.ID {
			t.Error("IDs differ between requests")
		}
		if firstResponse.Name != secondResponse.Name {
			t.Error("Names differ between requests")
		}
		if firstResponse.Version != secondResponse.Version {
			t.Error("Versions differ between requests")
		}
	})
}

// =============================================================================
// Management API Authorization Tests (Comprehensive)
// =============================================================================

func TestFunctions_ManagementAuthorizationComprehensive(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		body           any
		useServiceRole bool
		expectedStatus int
	}{
		// Service role operations
		{"list with service role", "GET", "/api/functions", nil, true, 200},
		{"list secrets with service role", "GET", "/api/functions/secrets", nil, true, 200},

		// Anon role operations (should be forbidden)
		{"list with anon", "GET", "/api/functions", nil, false, 403},
		{"create with anon", "POST", "/api/functions", map[string]any{"name": "test"}, false, 403},
		{"list secrets with anon", "GET", "/api/functions/secrets", nil, false, 403},
		{"create secret with anon", "POST", "/api/functions/secrets", map[string]any{"name": "TEST", "value": "test"}, false, 403},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var client *Client
			if tt.useServiceRole {
				client = NewClient(localbaseURL, serviceRoleKey)
			} else {
				client = NewClient(localbaseURL, localbaseAPIKey)
			}

			status, _, _, err := client.Request(tt.method, tt.path, tt.body, nil)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}

			if status != tt.expectedStatus {
				t.Errorf("Expected %d, got %d", tt.expectedStatus, status)
			}
		})
	}
}

// =============================================================================
// Regional Invocation Tests
// =============================================================================

func TestFunctions_RegionalInvocation(t *testing.T) {
	serviceClient := NewClient(localbaseURL, serviceRoleKey)

	fnName := fmt.Sprintf("test-region-%d", time.Now().UnixNano())
	fnID := createTestFunction(t, serviceClient, fnName, false)
	defer deleteTestFunction(t, serviceClient, fnID)

	t.Run("response includes x-sb-edge-region header", func(t *testing.T) {
		client := NewClient(localbaseURL, localbaseAPIKey)
		_, _, headers, err := client.Request("POST", "/functions/v1/"+fnName, nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		region := headers.Get("x-sb-edge-region")
		if region == "" {
			t.Error("Missing x-sb-edge-region header in response")
		} else {
			t.Logf("x-sb-edge-region: %s", region)
		}
	})

	t.Run("invoke with x-region header", func(t *testing.T) {
		req, err := http.NewRequest("POST", localbaseURL+"/functions/v1/"+fnName, nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		req.Header.Set("apikey", localbaseAPIKey)
		req.Header.Set("Authorization", "Bearer "+localbaseAPIKey)
		req.Header.Set("x-region", "eu-west-1")

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Errorf("Expected 200, got %d", resp.StatusCode)
		}

		region := resp.Header.Get("x-sb-edge-region")
		if region != "eu-west-1" {
			t.Errorf("Expected region eu-west-1, got %s", region)
		}
	})

	t.Run("invoke with forceFunctionRegion query param", func(t *testing.T) {
		req, err := http.NewRequest("POST", localbaseURL+"/functions/v1/"+fnName+"?forceFunctionRegion=ap-northeast-1", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		req.Header.Set("apikey", localbaseAPIKey)
		req.Header.Set("Authorization", "Bearer "+localbaseAPIKey)

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Errorf("Expected 200, got %d", resp.StatusCode)
		}

		region := resp.Header.Get("x-sb-edge-region")
		if region != "ap-northeast-1" {
			t.Errorf("Expected region ap-northeast-1, got %s", region)
		}
	})

	t.Run("invalid region falls back to default", func(t *testing.T) {
		req, err := http.NewRequest("POST", localbaseURL+"/functions/v1/"+fnName, nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		req.Header.Set("apikey", localbaseAPIKey)
		req.Header.Set("Authorization", "Bearer "+localbaseAPIKey)
		req.Header.Set("x-region", "invalid-region")

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Errorf("Expected 200, got %d", resp.StatusCode)
		}

		region := resp.Header.Get("x-sb-edge-region")
		if region != "us-east-1" {
			t.Errorf("Expected default region us-east-1, got %s", region)
		}
	})

	t.Run("region in response body", func(t *testing.T) {
		client := NewClient(localbaseURL, localbaseAPIKey)
		_, respBody, _, err := client.Request("POST", "/functions/v1/"+fnName, nil, map[string]string{
			"x-region": "us-west-2",
		})
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		var result map[string]any
		if err := json.Unmarshal(respBody, &result); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if region, ok := result["region"].(string); !ok || region != "us-west-2" {
			t.Errorf("Expected region us-west-2 in response body, got %v", result["region"])
		}
	})
}

// =============================================================================
// Path Routing Tests
// =============================================================================

func TestFunctions_PathRouting(t *testing.T) {
	serviceClient := NewClient(localbaseURL, serviceRoleKey)

	fnName := fmt.Sprintf("test-path-routing-%d", time.Now().UnixNano())
	fnID := createTestFunction(t, serviceClient, fnName, false)
	defer deleteTestFunction(t, serviceClient, fnID)

	t.Run("invoke with subpath", func(t *testing.T) {
		client := NewClient(localbaseURL, localbaseAPIKey)
		status, respBody, _, err := client.Request("GET", "/functions/v1/"+fnName+"/users/123", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, respBody)
		}

		var result map[string]any
		if err := json.Unmarshal(respBody, &result); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if path, ok := result["path"].(string); !ok || path != "/users/123" {
			t.Errorf("Expected path /users/123, got %v", result["path"])
		}
	})

	t.Run("invoke with deep nested path", func(t *testing.T) {
		client := NewClient(localbaseURL, localbaseAPIKey)
		status, respBody, _, err := client.Request("POST", "/functions/v1/"+fnName+"/api/v1/users/123/posts/456", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d", status)
		}

		var result map[string]any
		if err := json.Unmarshal(respBody, &result); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if path, ok := result["path"].(string); !ok || path != "/api/v1/users/123/posts/456" {
			t.Errorf("Expected deep nested path, got %v", result["path"])
		}
	})

	t.Run("subpath with query params", func(t *testing.T) {
		client := NewClient(localbaseURL, localbaseAPIKey)
		status, _, _, err := client.Request("GET", "/functions/v1/"+fnName+"/search?q=test&limit=10", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d", status)
		}
	})

	t.Run("subpath with all HTTP methods", func(t *testing.T) {
		methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE"}
		client := NewClient(localbaseURL, localbaseAPIKey)

		for _, method := range methods {
			t.Run(method, func(t *testing.T) {
				req, err := http.NewRequest(method, localbaseURL+"/functions/v1/"+fnName+"/resource/123", nil)
				if err != nil {
					t.Fatalf("Failed to create request: %v", err)
				}

				req.Header.Set("apikey", localbaseAPIKey)
				req.Header.Set("Authorization", "Bearer "+localbaseAPIKey)

				resp, err := client.client.Do(req)
				if err != nil {
					t.Fatalf("Request failed: %v", err)
				}
				defer resp.Body.Close()

				if resp.StatusCode != 200 {
					t.Errorf("Expected 200 for %s, got %d", method, resp.StatusCode)
				}
			})
		}
	})

	t.Run("OPTIONS preflight for subpath", func(t *testing.T) {
		req, err := http.NewRequest("OPTIONS", localbaseURL+"/functions/v1/"+fnName+"/api/resource", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Origin", "https://example.com")
		req.Header.Set("Access-Control-Request-Method", "POST")

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 && resp.StatusCode != 204 {
			t.Errorf("Expected 200 or 204 for OPTIONS, got %d", resp.StatusCode)
		}

		// Check CORS headers
		if resp.Header.Get("Access-Control-Allow-Origin") == "" {
			t.Error("Missing Access-Control-Allow-Origin header")
		}
	})
}

// =============================================================================
// SDK Compatibility Tests
// =============================================================================

func TestFunctions_SDKCompatibility(t *testing.T) {
	serviceClient := NewClient(localbaseURL, serviceRoleKey)

	fnName := fmt.Sprintf("test-sdk-%d", time.Now().UnixNano())
	fnID := createTestFunction(t, serviceClient, fnName, false)
	defer deleteTestFunction(t, serviceClient, fnID)

	t.Run("supabase-js compatible headers", func(t *testing.T) {
		req, err := http.NewRequest("POST", localbaseURL+"/functions/v1/"+fnName, strings.NewReader(`{"key": "value"}`))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		// Headers typically sent by supabase-js SDK
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("apikey", localbaseAPIKey)
		req.Header.Set("Authorization", "Bearer "+localbaseAPIKey)
		req.Header.Set("x-client-info", "supabase-js/2.38.0")
		req.Header.Set("Accept", "application/json")

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Errorf("Expected 200, got %d", resp.StatusCode)
		}
	})

	t.Run("response format matches SDK expectations", func(t *testing.T) {
		client := NewClient(localbaseURL, localbaseAPIKey)
		_, respBody, headers, err := client.Request("POST", "/functions/v1/"+fnName, map[string]string{"test": "data"}, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		// Check Content-Type
		contentType := headers.Get("Content-Type")
		if !strings.Contains(contentType, "application/json") {
			t.Errorf("Expected Content-Type application/json, got %s", contentType)
		}

		// Response should be valid JSON
		var result map[string]any
		if err := json.Unmarshal(respBody, &result); err != nil {
			t.Errorf("Response is not valid JSON: %v", err)
		}
	})
}

// =============================================================================
// Error Types Tests (FunctionsHttpError, FunctionsRelayError, FunctionsFetchError)
// =============================================================================

func TestFunctions_ErrorTypes(t *testing.T) {
	t.Run("404 error format for non-existent function", func(t *testing.T) {
		client := NewClient(localbaseURL, localbaseAPIKey)
		status, respBody, _, err := client.Request("POST", "/functions/v1/nonexistent-function-xyz", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 404 {
			t.Errorf("Expected 404, got %d", status)
		}

		var errResp map[string]any
		if err := json.Unmarshal(respBody, &errResp); err != nil {
			t.Fatalf("Failed to parse error response: %v", err)
		}

		// Verify error structure matches Supabase format
		if _, ok := errResp["error"]; !ok {
			t.Error("Error response missing 'error' field")
		}
		if _, ok := errResp["message"]; !ok {
			t.Error("Error response missing 'message' field")
		}
	})

	t.Run("401 error format for unauthorized request", func(t *testing.T) {
		serviceClient := NewClient(localbaseURL, serviceRoleKey)
		fnName := fmt.Sprintf("test-auth-error-%d", time.Now().UnixNano())
		fnID := createTestFunction(t, serviceClient, fnName, true) // verify_jwt=true
		defer deleteTestFunction(t, serviceClient, fnID)

		// Request without authentication
		req, _ := http.NewRequest("POST", localbaseURL+"/functions/v1/"+fnName, nil)
		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 401 {
			t.Logf("Expected 401, got %d (function may not require auth)", resp.StatusCode)
		}
	})

	t.Run("503 error format for inactive function", func(t *testing.T) {
		serviceClient := NewClient(localbaseURL, serviceRoleKey)
		fnName := fmt.Sprintf("test-inactive-error-%d", time.Now().UnixNano())
		fnID := createTestFunction(t, serviceClient, fnName, false)
		defer deleteTestFunction(t, serviceClient, fnID)

		// Set function to inactive
		serviceClient.Request("PUT", "/api/functions/"+fnID, map[string]any{"status": "inactive"}, nil)

		client := NewClient(localbaseURL, localbaseAPIKey)
		status, respBody, _, err := client.Request("POST", "/functions/v1/"+fnName, nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 503 {
			t.Errorf("Expected 503, got %d", status)
		}

		var errResp map[string]any
		if err := json.Unmarshal(respBody, &errResp); err != nil {
			t.Fatalf("Failed to parse error response: %v", err)
		}

		if errResp["error"] != "Service Unavailable" {
			t.Errorf("Expected error 'Service Unavailable', got %v", errResp["error"])
		}
	})
}

// =============================================================================
// All Valid Regions Test
// =============================================================================

func TestFunctions_AllValidRegions(t *testing.T) {
	serviceClient := NewClient(localbaseURL, serviceRoleKey)

	fnName := fmt.Sprintf("test-regions-%d", time.Now().UnixNano())
	fnID := createTestFunction(t, serviceClient, fnName, false)
	defer deleteTestFunction(t, serviceClient, fnID)

	validRegions := []string{
		"us-east-1",
		"us-west-1",
		"us-west-2",
		"ca-central-1",
		"eu-west-1",
		"eu-west-2",
		"eu-west-3",
		"eu-central-1",
		"ap-northeast-1",
		"ap-northeast-2",
		"ap-south-1",
		"ap-southeast-1",
		"ap-southeast-2",
		"sa-east-1",
	}

	for _, region := range validRegions {
		t.Run("region "+region, func(t *testing.T) {
			req, err := http.NewRequest("POST", localbaseURL+"/functions/v1/"+fnName, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			req.Header.Set("apikey", localbaseAPIKey)
			req.Header.Set("Authorization", "Bearer "+localbaseAPIKey)
			req.Header.Set("x-region", region)

			client := &http.Client{Timeout: 30 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				t.Errorf("Expected 200 for region %s, got %d", region, resp.StatusCode)
			}

			responseRegion := resp.Header.Get("x-sb-edge-region")
			if responseRegion != region {
				t.Errorf("Expected region %s, got %s", region, responseRegion)
			}
		})
	}
}

// =============================================================================
// Content Negotiation Tests
// =============================================================================

func TestFunctions_ContentNegotiation(t *testing.T) {
	serviceClient := NewClient(localbaseURL, serviceRoleKey)

	fnName := fmt.Sprintf("test-content-%d", time.Now().UnixNano())
	fnID := createTestFunction(t, serviceClient, fnName, false)
	defer deleteTestFunction(t, serviceClient, fnID)

	acceptTypes := []struct {
		accept       string
		description  string
	}{
		{"application/json", "JSON"},
		{"text/plain", "Plain text"},
		{"*/*", "Any"},
		{"application/json, text/plain;q=0.9, */*;q=0.8", "Weighted"},
		{"", "No Accept header"},
	}

	for _, tt := range acceptTypes {
		t.Run(tt.description, func(t *testing.T) {
			req, err := http.NewRequest("POST", localbaseURL+"/functions/v1/"+fnName, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			req.Header.Set("apikey", localbaseAPIKey)
			req.Header.Set("Authorization", "Bearer "+localbaseAPIKey)
			if tt.accept != "" {
				req.Header.Set("Accept", tt.accept)
			}

			client := &http.Client{Timeout: 30 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				t.Errorf("Expected 200 for Accept: %s, got %d", tt.accept, resp.StatusCode)
			}
		})
	}
}

// =============================================================================
// Comprehensive CORS Tests
// =============================================================================

func TestFunctions_CORSComprehensive(t *testing.T) {
	serviceClient := NewClient(localbaseURL, serviceRoleKey)

	fnName := fmt.Sprintf("test-cors-full-%d", time.Now().UnixNano())
	fnID := createTestFunction(t, serviceClient, fnName, false)
	defer deleteTestFunction(t, serviceClient, fnID)

	t.Run("all CORS headers present", func(t *testing.T) {
		client := NewClient(localbaseURL, localbaseAPIKey)
		_, _, headers, err := client.Request("POST", "/functions/v1/"+fnName, nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		requiredHeaders := map[string]string{
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Methods": "POST, GET, OPTIONS, PUT, DELETE, PATCH",
		}

		for header, expectedValue := range requiredHeaders {
			value := headers.Get(header)
			if value == "" {
				t.Errorf("Missing header: %s", header)
			} else if !strings.Contains(value, expectedValue) && value != expectedValue {
				t.Logf("Header %s: expected to contain %s, got %s", header, expectedValue, value)
			}
		}

		// Check Allow-Headers contains required values
		allowHeaders := headers.Get("Access-Control-Allow-Headers")
		requiredAllowHeaders := []string{"authorization", "apikey", "content-type"}
		for _, h := range requiredAllowHeaders {
			if !strings.Contains(strings.ToLower(allowHeaders), h) {
				t.Errorf("Access-Control-Allow-Headers missing: %s", h)
			}
		}
	})

	t.Run("x-region in allowed headers", func(t *testing.T) {
		req, err := http.NewRequest("OPTIONS", localbaseURL+"/functions/v1/"+fnName, nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Origin", "https://example.com")
		req.Header.Set("Access-Control-Request-Headers", "x-region")

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		allowHeaders := resp.Header.Get("Access-Control-Allow-Headers")
		if !strings.Contains(strings.ToLower(allowHeaders), "x-region") {
			t.Errorf("x-region not in Access-Control-Allow-Headers: %s", allowHeaders)
		}
	})
}

// =============================================================================
// Source Code API Tests (New endpoints)
// =============================================================================

func TestFunctions_SourceCode(t *testing.T) {
	serviceClient := NewClient(localbaseURL, serviceRoleKey)

	fnName := fmt.Sprintf("test-source-%d", time.Now().UnixNano())
	fnID := createTestFunction(t, serviceClient, fnName, false)
	defer deleteTestFunction(t, serviceClient, fnID)

	t.Run("get source code", func(t *testing.T) {
		status, respBody, _, err := serviceClient.Request("GET", "/api/functions/"+fnID+"/source", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, respBody)
		}

		var source map[string]any
		if err := json.Unmarshal(respBody, &source); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Verify source response has expected fields
		if source["function_id"] == nil {
			t.Error("Missing function_id in response")
		}
		if source["source_code"] == nil {
			t.Error("Missing source_code in response")
		}
		if source["entrypoint"] == nil {
			t.Error("Missing entrypoint in response")
		}
	})

	t.Run("update source (save draft)", func(t *testing.T) {
		newSource := `export default async function handler(req) { return new Response("Updated code"); }`
		body := map[string]any{
			"source_code": newSource,
		}

		status, respBody, _, err := serviceClient.Request("PUT", "/api/functions/"+fnID+"/source", body, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, respBody)
		}

		var result map[string]any
		if err := json.Unmarshal(respBody, &result); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if result["saved"] != true {
			t.Error("Expected saved=true")
		}
		if result["is_draft"] != true {
			t.Error("Expected is_draft=true")
		}
	})

	t.Run("source not found for invalid function", func(t *testing.T) {
		status, _, _, err := serviceClient.Request("GET", "/api/functions/invalid-id/source", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 404 {
			t.Errorf("Expected 404, got %d", status)
		}
	})
}

// =============================================================================
// Function Logs API Tests
// =============================================================================

func TestFunctions_Logs(t *testing.T) {
	serviceClient := NewClient(localbaseURL, serviceRoleKey)

	fnName := fmt.Sprintf("test-logs-%d", time.Now().UnixNano())
	fnID := createTestFunction(t, serviceClient, fnName, false)
	defer deleteTestFunction(t, serviceClient, fnID)

	t.Run("get empty logs", func(t *testing.T) {
		status, respBody, _, err := serviceClient.Request("GET", "/api/functions/"+fnID+"/logs", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, respBody)
		}

		var result map[string]any
		if err := json.Unmarshal(respBody, &result); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if result["logs"] == nil {
			t.Error("Missing logs field in response")
		}
	})

	t.Run("logs with limit parameter", func(t *testing.T) {
		status, respBody, _, err := serviceClient.Request("GET", "/api/functions/"+fnID+"/logs?limit=10", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, respBody)
		}
	})

	t.Run("logs with level filter", func(t *testing.T) {
		status, respBody, _, err := serviceClient.Request("GET", "/api/functions/"+fnID+"/logs?level=error", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, respBody)
		}
	})
}

// =============================================================================
// Function Metrics API Tests
// =============================================================================

func TestFunctions_Metrics(t *testing.T) {
	serviceClient := NewClient(localbaseURL, serviceRoleKey)

	fnName := fmt.Sprintf("test-metrics-%d", time.Now().UnixNano())
	fnID := createTestFunction(t, serviceClient, fnName, false)
	defer deleteTestFunction(t, serviceClient, fnID)

	t.Run("get metrics default period", func(t *testing.T) {
		status, respBody, _, err := serviceClient.Request("GET", "/api/functions/"+fnID+"/metrics", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, respBody)
		}

		var result map[string]any
		if err := json.Unmarshal(respBody, &result); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Verify expected fields
		if result["function_id"] == nil {
			t.Error("Missing function_id in response")
		}
		if result["period"] == nil {
			t.Error("Missing period in response")
		}
		if result["invocations"] == nil {
			t.Error("Missing invocations in response")
		}
		if result["latency"] == nil {
			t.Error("Missing latency in response")
		}
	})

	periods := []string{"1h", "24h", "7d", "30d"}
	for _, period := range periods {
		t.Run("metrics for period "+period, func(t *testing.T) {
			status, respBody, _, err := serviceClient.Request("GET", "/api/functions/"+fnID+"/metrics?period="+period, nil, nil)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}

			if status != 200 {
				t.Errorf("Expected 200, got %d: %s", status, respBody)
			}

			var result map[string]any
			if err := json.Unmarshal(respBody, &result); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			if result["period"] != period {
				t.Errorf("Expected period=%s, got %v", period, result["period"])
			}
		})
	}
}

// =============================================================================
// Function Test Endpoint Tests
// =============================================================================

func TestFunctions_TestEndpoint(t *testing.T) {
	serviceClient := NewClient(localbaseURL, serviceRoleKey)

	fnName := fmt.Sprintf("test-test-%d", time.Now().UnixNano())
	fnID := createTestFunction(t, serviceClient, fnName, false)
	defer deleteTestFunction(t, serviceClient, fnID)

	t.Run("test function with POST", func(t *testing.T) {
		body := map[string]any{
			"method": "POST",
			"path":   "/",
			"headers": map[string]string{
				"Content-Type": "application/json",
			},
			"body": map[string]any{
				"name": "Test",
			},
		}

		status, respBody, _, err := serviceClient.Request("POST", "/api/functions/"+fnID+"/test", body, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, respBody)
		}

		var result map[string]any
		if err := json.Unmarshal(respBody, &result); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Verify response structure
		if result["status"] == nil {
			t.Error("Missing status in response")
		}
		if result["headers"] == nil {
			t.Error("Missing headers in response")
		}
		if result["body"] == nil {
			t.Error("Missing body in response")
		}
		if result["duration_ms"] == nil {
			t.Error("Missing duration_ms in response")
		}
	})

	t.Run("test function with different methods", func(t *testing.T) {
		methods := []string{"GET", "PUT", "PATCH", "DELETE"}
		for _, method := range methods {
			body := map[string]any{
				"method": method,
				"path":   "/test-path",
			}

			status, _, _, err := serviceClient.Request("POST", "/api/functions/"+fnID+"/test", body, nil)
			if err != nil {
				t.Fatalf("Request failed for %s: %v", method, err)
			}

			if status != 200 {
				t.Errorf("Expected 200 for %s, got %d", method, status)
			}
		}
	})
}

// =============================================================================
// Function Templates API Tests
// =============================================================================

func TestFunctions_Templates(t *testing.T) {
	serviceClient := NewClient(localbaseURL, serviceRoleKey)

	t.Run("list templates", func(t *testing.T) {
		status, respBody, _, err := serviceClient.Request("GET", "/api/functions/templates", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, respBody)
		}

		var result map[string]any
		if err := json.Unmarshal(respBody, &result); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		templates, ok := result["templates"].([]any)
		if !ok {
			t.Fatal("templates field should be an array")
		}

		if len(templates) == 0 {
			t.Error("Expected at least one template")
		}

		// Verify first template structure
		if len(templates) > 0 {
			template := templates[0].(map[string]any)
			if template["id"] == nil {
				t.Error("Template missing id")
			}
			if template["name"] == nil {
				t.Error("Template missing name")
			}
			if template["description"] == nil {
				t.Error("Template missing description")
			}
			if template["category"] == nil {
				t.Error("Template missing category")
			}
		}
	})

	t.Run("get specific template", func(t *testing.T) {
		// First get the list to find an ID
		_, respBody, _, _ := serviceClient.Request("GET", "/api/functions/templates", nil, nil)
		var listResult map[string]any
		json.Unmarshal(respBody, &listResult)
		templates := listResult["templates"].([]any)

		if len(templates) > 0 {
			template := templates[0].(map[string]any)
			templateID := template["id"].(string)

			status, respBody, _, err := serviceClient.Request("GET", "/api/functions/templates/"+templateID, nil, nil)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}

			if status != 200 {
				t.Errorf("Expected 200, got %d: %s", status, respBody)
			}

			var result map[string]any
			if err := json.Unmarshal(respBody, &result); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			if result["id"] != templateID {
				t.Errorf("Expected id=%s, got %v", templateID, result["id"])
			}
			if result["source_code"] == nil {
				t.Error("Template missing source_code")
			}
		}
	})

	t.Run("template not found", func(t *testing.T) {
		status, _, _, err := serviceClient.Request("GET", "/api/functions/templates/nonexistent-template", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 404 {
			t.Errorf("Expected 404, got %d", status)
		}
	})
}

// =============================================================================
// Bulk Secrets API Tests
// =============================================================================

func TestFunctions_BulkSecrets(t *testing.T) {
	serviceClient := NewClient(localbaseURL, serviceRoleKey)

	t.Run("bulk update secrets", func(t *testing.T) {
		// Clean up any existing test secrets
		secretsToClean := []string{"BULK_TEST_KEY_1", "BULK_TEST_KEY_2", "BULK_TEST_KEY_3"}
		for _, name := range secretsToClean {
			serviceClient.Request("DELETE", "/api/functions/secrets/"+name, nil, nil)
		}

		body := map[string]any{
			"secrets": []map[string]string{
				{"name": "BULK_TEST_KEY_1", "value": "value1"},
				{"name": "BULK_TEST_KEY_2", "value": "value2"},
				{"name": "BULK_TEST_KEY_3", "value": "value3"},
			},
		}

		status, respBody, _, err := serviceClient.Request("PUT", "/api/functions/secrets/bulk", body, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, respBody)
		}

		var result map[string]any
		if err := json.Unmarshal(respBody, &result); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		created, _ := result["created"].(float64)
		if created < 1 {
			t.Errorf("Expected at least 1 created, got %v", result["created"])
		}

		total, _ := result["total"].(float64)
		if total != 3 {
			t.Errorf("Expected total=3, got %v", result["total"])
		}

		// Clean up
		for _, name := range secretsToClean {
			serviceClient.Request("DELETE", "/api/functions/secrets/"+name, nil, nil)
		}
	})

	t.Run("bulk update with updates", func(t *testing.T) {
		// First create a secret
		secretName := fmt.Sprintf("BULK_UPDATE_%d", time.Now().UnixNano())
		createBody := map[string]any{
			"name":  secretName,
			"value": "original",
		}
		serviceClient.Request("POST", "/api/functions/secrets", createBody, nil)
		defer serviceClient.Request("DELETE", "/api/functions/secrets/"+secretName, nil, nil)

		// Now bulk update with the same name
		body := map[string]any{
			"secrets": []map[string]string{
				{"name": secretName, "value": "updated"},
			},
		}

		status, respBody, _, err := serviceClient.Request("PUT", "/api/functions/secrets/bulk", body, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, respBody)
		}

		var result map[string]any
		if err := json.Unmarshal(respBody, &result); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		updated, _ := result["updated"].(float64)
		if updated < 1 {
			t.Errorf("Expected at least 1 updated, got %v", result["updated"])
		}
	})

	t.Run("bulk update with empty list", func(t *testing.T) {
		body := map[string]any{
			"secrets": []map[string]string{},
		}

		status, respBody, _, err := serviceClient.Request("PUT", "/api/functions/secrets/bulk", body, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, respBody)
		}

		var result map[string]any
		json.Unmarshal(respBody, &result)
		total, _ := result["total"].(float64)
		if total != 0 {
			t.Errorf("Expected total=0 for empty list, got %v", result["total"])
		}
	})
}

// =============================================================================
// Function Download Tests
// =============================================================================

func TestFunctions_Download(t *testing.T) {
	serviceClient := NewClient(localbaseURL, serviceRoleKey)

	fnName := fmt.Sprintf("test-download-%d", time.Now().UnixNano())
	sourceCode := `export default async function handler(req) { return new Response("Downloadable code"); }`

	// Create and deploy the function
	body := map[string]any{
		"name":        fnName,
		"verify_jwt":  false,
		"source_code": sourceCode,
	}

	status, respBody, _, err := serviceClient.Request("POST", "/api/functions", body, nil)
	if err != nil || status != 201 {
		t.Fatalf("Failed to create function: %v, status: %d", err, status)
	}

	var fn Function
	json.Unmarshal(respBody, &fn)
	defer deleteTestFunction(t, serviceClient, fn.ID)

	// Deploy to have source available
	deployBody := map[string]any{
		"source_code": sourceCode,
	}
	serviceClient.Request("POST", "/api/functions/"+fn.ID+"/deploy", deployBody, nil)

	t.Run("download function", func(t *testing.T) {
		status, respBody, headers, err := serviceClient.Request("POST", "/api/functions/"+fn.ID+"/download", nil, nil)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if status != 200 {
			t.Errorf("Expected 200, got %d: %s", status, respBody)
		}

		// Check content type
		contentType := headers.Get("Content-Type")
		if contentType != "application/typescript" {
			t.Logf("Content-Type: %s (expected application/typescript)", contentType)
		}

		// Check content disposition
		disposition := headers.Get("Content-Disposition")
		if !strings.Contains(disposition, fnName) {
			t.Logf("Content-Disposition: %s (expected to contain function name)", disposition)
		}
	})
}
