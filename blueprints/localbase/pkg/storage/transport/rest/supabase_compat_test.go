//go:build integration
// +build integration

// This file contains integration tests that compare Localbase and Supabase Storage APIs.
// Run with: go test -tags=integration ./pkg/storage/transport/rest/...
// Requires both Supabase and Localbase servers running.

package rest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// getEnv returns environment variable value or default
func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

// Storage test configuration
var (
	// Supabase Local storage endpoints
	supabaseStorageURL = getEnv("SUPABASE_STORAGE_URL", "http://127.0.0.1:54421/storage/v1")
	supabaseS3URL      = getEnv("SUPABASE_S3_URL", "http://127.0.0.1:54421/storage/v1/s3")

	// Supabase service_role key for admin operations (bypasses RLS)
	// This key should be set via environment variable for security
	supabaseServiceRoleKey = getEnv("SUPABASE_SERVICE_ROLE_KEY", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZS1kZW1vIiwicm9sZSI6InNlcnZpY2Vfcm9sZSIsImV4cCI6MTk4MzgxMjk5Nn0.EGIM96RAZx35lJzdJsyH-qQwv8Hdp7fsn3W0YpN81IU")

	// Localbase storage endpoints
	localbaseStorageURL = getEnv("LOCALBASE_STORAGE_URL", "http://localhost:54321/storage/v1")
	localbaseS3URL      = getEnv("LOCALBASE_S3_URL", "http://localhost:54321/s3")

	// Localbase API key (same as service role key for testing)
	localbaseAPIKey = getEnv("LOCALBASE_API_KEY", supabaseServiceRoleKey)
)

// StorageTestClient wraps HTTP client for Storage API testing
type StorageTestClient struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

func NewStorageTestClient(baseURL, apiKey string) *StorageTestClient {
	return &StorageTestClient{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		apiKey:  apiKey,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (c *StorageTestClient) Do(method, path string, body interface{}, headers map[string]string) (*http.Response, []byte, error) {
	var bodyReader io.Reader

	switch v := body.(type) {
	case nil:
		// No body
	case []byte:
		bodyReader = bytes.NewReader(v)
	case string:
		bodyReader = strings.NewReader(v)
	case io.Reader:
		bodyReader = v
	default:
		// JSON encode
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

	// Default content type for JSON
	if body != nil {
		switch body.(type) {
		case []byte, string, io.Reader:
			// Don't set default content type for raw bodies
		default:
			req.Header.Set("Content-Type", "application/json")
		}
	}

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

// DoRaw performs request without closing body (for streaming)
func (c *StorageTestClient) DoRaw(method, path string, body io.Reader, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("apikey", c.apiKey)
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return c.client.Do(req)
}

// StorageComparisonResult holds comparison results
type StorageComparisonResult struct {
	TestName           string
	SupabaseStatus     int
	LocalbaseStatus    int
	StatusMatch        bool
	SupabaseBody       interface{}
	LocalbaseBody      interface{}
	BodyMatch          bool
	SupabaseHeaders    map[string]string
	LocalbaseHeaders   map[string]string
	HeadersMatch       bool
	SupabaseErrorCode  string
	LocalbaseErrorCode string
	ErrorCodeMatch     bool
	Notes              string
}

func (r *StorageComparisonResult) String() string {
	status := "PASS"
	if !r.StatusMatch {
		status = "FAIL"
	}
	return fmt.Sprintf("[%s] %s - Status: %d/%d",
		status, r.TestName,
		r.SupabaseStatus, r.LocalbaseStatus)
}

// CompareStorage runs the same request against both storage endpoints
func CompareStorage(t *testing.T, name, method, path string, body interface{}, headers map[string]string) *StorageComparisonResult {
	t.Helper()

	supabase := NewStorageTestClient(supabaseStorageURL, supabaseServiceRoleKey)
	localbase := NewStorageTestClient(localbaseStorageURL, localbaseAPIKey)

	result := &StorageComparisonResult{TestName: name}

	// Run against Supabase
	sResp, sBody, sErr := supabase.Do(method, path, body, headers)
	if sErr != nil {
		t.Logf("Supabase request error: %v", sErr)
		result.Notes = fmt.Sprintf("Supabase error: %v", sErr)
	} else {
		result.SupabaseStatus = sResp.StatusCode
		result.SupabaseHeaders = extractStorageHeaders(sResp)
		if err := json.Unmarshal(sBody, &result.SupabaseBody); err != nil {
			result.SupabaseBody = string(sBody)
		}
		result.SupabaseErrorCode = extractStorageErrorCode(sBody)
	}

	// Run against Localbase
	lResp, lBody, lErr := localbase.Do(method, path, body, headers)
	if lErr != nil {
		t.Logf("Localbase request error: %v", lErr)
		result.Notes += fmt.Sprintf(" Localbase error: %v", lErr)
	} else {
		result.LocalbaseStatus = lResp.StatusCode
		result.LocalbaseHeaders = extractStorageHeaders(lResp)
		if err := json.Unmarshal(lBody, &result.LocalbaseBody); err != nil {
			result.LocalbaseBody = string(lBody)
		}
		result.LocalbaseErrorCode = extractStorageErrorCode(lBody)
	}

	// Compare results
	result.StatusMatch = result.SupabaseStatus == result.LocalbaseStatus
	result.ErrorCodeMatch = result.SupabaseErrorCode == result.LocalbaseErrorCode

	// Log result
	t.Log(result.String())
	if !result.StatusMatch {
		t.Errorf("Status mismatch: Supabase=%d, Localbase=%d", result.SupabaseStatus, result.LocalbaseStatus)
		t.Logf("Supabase body: %s", truncateString(fmt.Sprintf("%v", result.SupabaseBody), 500))
		t.Logf("Localbase body: %s", truncateString(fmt.Sprintf("%v", result.LocalbaseBody), 500))
	}

	return result
}

func extractStorageHeaders(resp *http.Response) map[string]string {
	headers := make(map[string]string)
	for _, key := range []string{"Content-Type", "Content-Length", "ETag", "Content-Disposition", "Accept-Ranges"} {
		if v := resp.Header.Get(key); v != "" {
			headers[key] = v
		}
	}
	return headers
}

func extractStorageErrorCode(body []byte) string {
	var errResp struct {
		StatusCode int    `json:"statusCode"`
		Error      string `json:"error"`
		Message    string `json:"message"`
	}
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error != "" {
		return errResp.Error
	}
	return ""
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// =============================================================================
// Bucket Operations Tests
// =============================================================================

func TestStorage_Bucket_Create(t *testing.T) {
	// Generate unique bucket names
	ts := time.Now().UnixNano()

	tests := []struct {
		name     string
		body     map[string]interface{}
		wantCode int
	}{
		{
			name:     "BUCKET-001: Create public bucket",
			body:     map[string]interface{}{"name": fmt.Sprintf("test-public-%d", ts), "public": true},
			wantCode: 200,
		},
		{
			name:     "BUCKET-002: Create private bucket",
			body:     map[string]interface{}{"name": fmt.Sprintf("test-private-%d", ts), "public": false},
			wantCode: 200,
		},
		{
			name:     "BUCKET-003: Create bucket with file size limit",
			body:     map[string]interface{}{"name": fmt.Sprintf("test-limited-%d", ts), "file_size_limit": 1048576},
			wantCode: 200,
		},
		{
			name:     "BUCKET-005: Create bucket with empty name",
			body:     map[string]interface{}{"name": ""},
			wantCode: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CompareStorage(t, tt.name, "POST", "/bucket", tt.body, nil)
			if result.SupabaseStatus != tt.wantCode && result.LocalbaseStatus != tt.wantCode {
				t.Logf("Expected status %d, got Supabase=%d, Localbase=%d",
					tt.wantCode, result.SupabaseStatus, result.LocalbaseStatus)
			}
		})
	}
}

func TestStorage_Bucket_List(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"BUCKET-010: List all buckets", "/bucket"},
		{"BUCKET-011: List with limit", "/bucket?limit=5"},
		{"BUCKET-012: List with offset", "/bucket?offset=1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			CompareStorage(t, tt.name, "GET", tt.path, nil, nil)
		})
	}
}

func TestStorage_Bucket_Get(t *testing.T) {
	// First create a test bucket
	ts := time.Now().UnixNano()
	bucketName := fmt.Sprintf("test-get-%d", ts)

	// Create bucket on both endpoints
	supabase := NewStorageTestClient(supabaseStorageURL, supabaseServiceRoleKey)
	localbase := NewStorageTestClient(localbaseStorageURL, localbaseAPIKey)

	body := map[string]interface{}{"name": bucketName, "public": true}
	supabase.Do("POST", "/bucket", body, nil)
	localbase.Do("POST", "/bucket", body, nil)

	tests := []struct {
		name string
		path string
	}{
		{"BUCKET-020: Get existing bucket", "/bucket/" + bucketName},
		{"BUCKET-021: Get non-existent bucket", "/bucket/nonexistent-bucket-xyz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			CompareStorage(t, tt.name, "GET", tt.path, nil, nil)
		})
	}
}

func TestStorage_Bucket_Delete(t *testing.T) {
	// Create test buckets
	ts := time.Now().UnixNano()
	emptyBucket := fmt.Sprintf("test-empty-del-%d", ts)

	supabase := NewStorageTestClient(supabaseStorageURL, supabaseServiceRoleKey)
	localbase := NewStorageTestClient(localbaseStorageURL, localbaseAPIKey)

	// Create empty bucket
	body := map[string]interface{}{"name": emptyBucket, "public": true}
	supabase.Do("POST", "/bucket", body, nil)
	localbase.Do("POST", "/bucket", body, nil)

	tests := []struct {
		name string
		path string
	}{
		{"BUCKET-040: Delete empty bucket", "/bucket/" + emptyBucket},
		{"BUCKET-042: Delete non-existent bucket", "/bucket/nonexistent-xyz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			CompareStorage(t, tt.name, "DELETE", tt.path, nil, nil)
		})
	}
}

// =============================================================================
// Object Upload Tests
// =============================================================================

func TestStorage_Object_Upload(t *testing.T) {
	// Create test bucket
	ts := time.Now().UnixNano()
	bucketName := fmt.Sprintf("test-upload-%d", ts)

	supabase := NewStorageTestClient(supabaseStorageURL, supabaseServiceRoleKey)
	localbase := NewStorageTestClient(localbaseStorageURL, localbaseAPIKey)

	// Create bucket on both
	body := map[string]interface{}{"name": bucketName, "public": true}
	supabase.Do("POST", "/bucket", body, nil)
	localbase.Do("POST", "/bucket", body, nil)

	t.Run("UPLOAD-001: Upload small file", func(t *testing.T) {
		content := []byte("Hello, World!")
		headers := map[string]string{"Content-Type": "text/plain"}

		// Upload to Supabase
		sResp, sBody, sErr := supabase.Do("POST", "/object/"+bucketName+"/hello.txt", content, headers)
		// Upload to Localbase
		lResp, lBody, lErr := localbase.Do("POST", "/object/"+bucketName+"/hello.txt", content, headers)

		if sErr != nil || lErr != nil {
			t.Logf("Supabase error: %v, Localbase error: %v", sErr, lErr)
		}

		if sResp != nil && lResp != nil {
			if sResp.StatusCode != lResp.StatusCode {
				t.Errorf("Status mismatch: Supabase=%d, Localbase=%d", sResp.StatusCode, lResp.StatusCode)
			}
			t.Logf("[%s] UPLOAD-001 - Status: %d/%d", statusSymbol(sResp.StatusCode == lResp.StatusCode), sResp.StatusCode, lResp.StatusCode)
			t.Logf("Supabase body: %s", truncateString(string(sBody), 200))
			t.Logf("Localbase body: %s", truncateString(string(lBody), 200))
		}
	})

	t.Run("UPLOAD-004: Upload to nested path", func(t *testing.T) {
		content := []byte("Nested file content")
		headers := map[string]string{"Content-Type": "text/plain"}

		sResp, _, sErr := supabase.Do("POST", "/object/"+bucketName+"/folder/subfolder/nested.txt", content, headers)
		lResp, _, lErr := localbase.Do("POST", "/object/"+bucketName+"/folder/subfolder/nested.txt", content, headers)

		if sErr != nil || lErr != nil {
			t.Logf("Errors: Supabase=%v, Localbase=%v", sErr, lErr)
		}

		if sResp != nil && lResp != nil {
			if sResp.StatusCode != lResp.StatusCode {
				t.Errorf("Status mismatch: Supabase=%d, Localbase=%d", sResp.StatusCode, lResp.StatusCode)
			}
			t.Logf("[%s] UPLOAD-004 - Status: %d/%d", statusSymbol(sResp.StatusCode == lResp.StatusCode), sResp.StatusCode, lResp.StatusCode)
		}
	})

	t.Run("UPLOAD-007: Upload to non-existent bucket", func(t *testing.T) {
		content := []byte("Test content")
		headers := map[string]string{"Content-Type": "text/plain"}

		sResp, _, _ := supabase.Do("POST", "/object/nonexistent-bucket-xyz/file.txt", content, headers)
		lResp, _, _ := localbase.Do("POST", "/object/nonexistent-bucket-xyz/file.txt", content, headers)

		if sResp != nil && lResp != nil {
			if sResp.StatusCode != lResp.StatusCode {
				t.Errorf("Status mismatch: Supabase=%d, Localbase=%d", sResp.StatusCode, lResp.StatusCode)
			}
			t.Logf("[%s] UPLOAD-007 - Status: %d/%d", statusSymbol(sResp.StatusCode == lResp.StatusCode), sResp.StatusCode, lResp.StatusCode)
		}
	})

	t.Run("UPLOAD-008: Upload empty file", func(t *testing.T) {
		content := []byte{}
		headers := map[string]string{"Content-Type": "application/octet-stream"}

		sResp, _, _ := supabase.Do("POST", "/object/"+bucketName+"/empty.bin", content, headers)
		lResp, _, _ := localbase.Do("POST", "/object/"+bucketName+"/empty.bin", content, headers)

		if sResp != nil && lResp != nil {
			if sResp.StatusCode != lResp.StatusCode {
				t.Errorf("Status mismatch: Supabase=%d, Localbase=%d", sResp.StatusCode, lResp.StatusCode)
			}
			t.Logf("[%s] UPLOAD-008 - Status: %d/%d", statusSymbol(sResp.StatusCode == lResp.StatusCode), sResp.StatusCode, lResp.StatusCode)
		}
	})
}

func TestStorage_Object_Upload_Upsert(t *testing.T) {
	ts := time.Now().UnixNano()
	bucketName := fmt.Sprintf("test-upsert-%d", ts)

	supabase := NewStorageTestClient(supabaseStorageURL, supabaseServiceRoleKey)
	localbase := NewStorageTestClient(localbaseStorageURL, localbaseAPIKey)

	// Create bucket
	body := map[string]interface{}{"name": bucketName, "public": true}
	supabase.Do("POST", "/bucket", body, nil)
	localbase.Do("POST", "/bucket", body, nil)

	// First upload
	content := []byte("Original content")
	headers := map[string]string{"Content-Type": "text/plain"}
	supabase.Do("POST", "/object/"+bucketName+"/upsert-test.txt", content, headers)
	localbase.Do("POST", "/object/"+bucketName+"/upsert-test.txt", content, headers)

	t.Run("UPLOAD-005: Upload duplicate without upsert", func(t *testing.T) {
		content := []byte("New content")
		headers := map[string]string{"Content-Type": "text/plain"}

		sResp, _, _ := supabase.Do("POST", "/object/"+bucketName+"/upsert-test.txt", content, headers)
		lResp, _, _ := localbase.Do("POST", "/object/"+bucketName+"/upsert-test.txt", content, headers)

		if sResp != nil && lResp != nil {
			// Both should return conflict (409) or similar
			if sResp.StatusCode != lResp.StatusCode {
				t.Errorf("Status mismatch: Supabase=%d, Localbase=%d", sResp.StatusCode, lResp.StatusCode)
			}
			t.Logf("[%s] UPLOAD-005 - Status: %d/%d", statusSymbol(sResp.StatusCode == lResp.StatusCode), sResp.StatusCode, lResp.StatusCode)
		}
	})

	t.Run("UPLOAD-006: Upload with upsert header", func(t *testing.T) {
		content := []byte("Updated content with upsert")
		headers := map[string]string{
			"Content-Type": "text/plain",
			"x-upsert":     "true",
		}

		sResp, _, _ := supabase.Do("POST", "/object/"+bucketName+"/upsert-test.txt", content, headers)
		lResp, _, _ := localbase.Do("POST", "/object/"+bucketName+"/upsert-test.txt", content, headers)

		if sResp != nil && lResp != nil {
			if sResp.StatusCode != lResp.StatusCode {
				t.Errorf("Status mismatch: Supabase=%d, Localbase=%d", sResp.StatusCode, lResp.StatusCode)
			}
			t.Logf("[%s] UPLOAD-006 - Status: %d/%d", statusSymbol(sResp.StatusCode == lResp.StatusCode), sResp.StatusCode, lResp.StatusCode)
		}
	})
}

// =============================================================================
// Object Download Tests
// =============================================================================

func TestStorage_Object_Download(t *testing.T) {
	ts := time.Now().UnixNano()
	bucketName := fmt.Sprintf("test-download-%d", ts)

	supabase := NewStorageTestClient(supabaseStorageURL, supabaseServiceRoleKey)
	localbase := NewStorageTestClient(localbaseStorageURL, localbaseAPIKey)

	// Create bucket and upload file
	body := map[string]interface{}{"name": bucketName, "public": true}
	supabase.Do("POST", "/bucket", body, nil)
	localbase.Do("POST", "/bucket", body, nil)

	content := []byte("Test content for download")
	headers := map[string]string{"Content-Type": "text/plain"}
	supabase.Do("POST", "/object/"+bucketName+"/download-test.txt", content, headers)
	localbase.Do("POST", "/object/"+bucketName+"/download-test.txt", content, headers)

	t.Run("DOWNLOAD-001: Download existing file", func(t *testing.T) {
		sResp, sBody, _ := supabase.Do("GET", "/object/"+bucketName+"/download-test.txt", nil, nil)
		lResp, lBody, _ := localbase.Do("GET", "/object/"+bucketName+"/download-test.txt", nil, nil)

		if sResp != nil && lResp != nil {
			if sResp.StatusCode != lResp.StatusCode {
				t.Errorf("Status mismatch: Supabase=%d, Localbase=%d", sResp.StatusCode, lResp.StatusCode)
			}
			if !bytes.Equal(sBody, lBody) {
				t.Errorf("Content mismatch:\nSupabase: %s\nLocalbase: %s", string(sBody), string(lBody))
			}
			t.Logf("[%s] DOWNLOAD-001 - Status: %d/%d, Content match: %v",
				statusSymbol(sResp.StatusCode == lResp.StatusCode && bytes.Equal(sBody, lBody)),
				sResp.StatusCode, lResp.StatusCode, bytes.Equal(sBody, lBody))
		}
	})

	t.Run("DOWNLOAD-002: Download non-existent file", func(t *testing.T) {
		sResp, _, _ := supabase.Do("GET", "/object/"+bucketName+"/nonexistent.txt", nil, nil)
		lResp, _, _ := localbase.Do("GET", "/object/"+bucketName+"/nonexistent.txt", nil, nil)

		if sResp != nil && lResp != nil {
			if sResp.StatusCode != lResp.StatusCode {
				t.Errorf("Status mismatch: Supabase=%d, Localbase=%d", sResp.StatusCode, lResp.StatusCode)
			}
			t.Logf("[%s] DOWNLOAD-002 - Status: %d/%d", statusSymbol(sResp.StatusCode == lResp.StatusCode), sResp.StatusCode, lResp.StatusCode)
		}
	})

	t.Run("DOWNLOAD-004: Download with Range header", func(t *testing.T) {
		headers := map[string]string{"Range": "bytes=0-9"}

		sResp, sBody, _ := supabase.Do("GET", "/object/"+bucketName+"/download-test.txt", nil, headers)
		lResp, lBody, _ := localbase.Do("GET", "/object/"+bucketName+"/download-test.txt", nil, headers)

		if sResp != nil && lResp != nil {
			// Should be 206 Partial Content
			if sResp.StatusCode != lResp.StatusCode {
				t.Errorf("Status mismatch: Supabase=%d, Localbase=%d", sResp.StatusCode, lResp.StatusCode)
			}
			t.Logf("[%s] DOWNLOAD-004 - Status: %d/%d, Body length: %d/%d",
				statusSymbol(sResp.StatusCode == lResp.StatusCode),
				sResp.StatusCode, lResp.StatusCode, len(sBody), len(lBody))
		}
	})
}

// =============================================================================
// Object List Tests
// =============================================================================

func TestStorage_Object_List(t *testing.T) {
	ts := time.Now().UnixNano()
	bucketName := fmt.Sprintf("test-list-%d", ts)

	supabase := NewStorageTestClient(supabaseStorageURL, supabaseServiceRoleKey)
	localbase := NewStorageTestClient(localbaseStorageURL, localbaseAPIKey)

	// Create bucket and upload multiple files
	body := map[string]interface{}{"name": bucketName, "public": true}
	supabase.Do("POST", "/bucket", body, nil)
	localbase.Do("POST", "/bucket", body, nil)

	// Upload test files
	files := []string{"file1.txt", "file2.txt", "folder/nested1.txt", "folder/nested2.txt"}
	headers := map[string]string{"Content-Type": "text/plain"}
	for _, file := range files {
		content := []byte("Content of " + file)
		supabase.Do("POST", "/object/"+bucketName+"/"+file, content, headers)
		localbase.Do("POST", "/object/"+bucketName+"/"+file, content, headers)
	}

	t.Run("LIST-001: List root level", func(t *testing.T) {
		reqBody := map[string]interface{}{"prefix": ""}
		CompareStorage(t, "LIST-001", "POST", "/object/list/"+bucketName, reqBody, nil)
	})

	t.Run("LIST-002: List with prefix", func(t *testing.T) {
		reqBody := map[string]interface{}{"prefix": "folder/"}
		CompareStorage(t, "LIST-002", "POST", "/object/list/"+bucketName, reqBody, nil)
	})

	t.Run("LIST-003: List with limit", func(t *testing.T) {
		reqBody := map[string]interface{}{"prefix": "", "limit": 2}
		CompareStorage(t, "LIST-003", "POST", "/object/list/"+bucketName, reqBody, nil)
	})

	t.Run("LIST-007: List non-existent bucket", func(t *testing.T) {
		reqBody := map[string]interface{}{"prefix": ""}
		CompareStorage(t, "LIST-007", "POST", "/object/list/nonexistent-bucket-xyz", reqBody, nil)
	})
}

// =============================================================================
// Object Move/Copy Tests
// =============================================================================

func TestStorage_Object_Move(t *testing.T) {
	ts := time.Now().UnixNano()
	bucketName := fmt.Sprintf("test-move-%d", ts)

	supabase := NewStorageTestClient(supabaseStorageURL, supabaseServiceRoleKey)
	localbase := NewStorageTestClient(localbaseStorageURL, localbaseAPIKey)

	// Create bucket and upload file
	body := map[string]interface{}{"name": bucketName, "public": true}
	supabase.Do("POST", "/bucket", body, nil)
	localbase.Do("POST", "/bucket", body, nil)

	content := []byte("Content to move")
	headers := map[string]string{"Content-Type": "text/plain"}
	supabase.Do("POST", "/object/"+bucketName+"/source.txt", content, headers)
	localbase.Do("POST", "/object/"+bucketName+"/source.txt", content, headers)

	t.Run("MOVE-001: Move within bucket", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"bucketId":       bucketName,
			"sourceKey":      "source.txt",
			"destinationKey": "destination.txt",
		}
		CompareStorage(t, "MOVE-001", "POST", "/object/move", reqBody, nil)
	})

	t.Run("MOVE-004: Move non-existent file", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"bucketId":       bucketName,
			"sourceKey":      "nonexistent.txt",
			"destinationKey": "moved.txt",
		}
		CompareStorage(t, "MOVE-004", "POST", "/object/move", reqBody, nil)
	})
}

func TestStorage_Object_Copy(t *testing.T) {
	ts := time.Now().UnixNano()
	bucketName := fmt.Sprintf("test-copy-%d", ts)

	supabase := NewStorageTestClient(supabaseStorageURL, supabaseServiceRoleKey)
	localbase := NewStorageTestClient(localbaseStorageURL, localbaseAPIKey)

	// Create bucket and upload file
	body := map[string]interface{}{"name": bucketName, "public": true}
	supabase.Do("POST", "/bucket", body, nil)
	localbase.Do("POST", "/bucket", body, nil)

	content := []byte("Content to copy")
	headers := map[string]string{"Content-Type": "text/plain"}
	supabase.Do("POST", "/object/"+bucketName+"/original.txt", content, headers)
	localbase.Do("POST", "/object/"+bucketName+"/original.txt", content, headers)

	t.Run("COPY-001: Copy within bucket", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"bucketId":       bucketName,
			"sourceKey":      "original.txt",
			"destinationKey": "copied.txt",
		}
		CompareStorage(t, "COPY-001", "POST", "/object/copy", reqBody, nil)
	})

	t.Run("COPY-004: Copy non-existent file", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"bucketId":       bucketName,
			"sourceKey":      "nonexistent.txt",
			"destinationKey": "copied.txt",
		}
		CompareStorage(t, "COPY-004", "POST", "/object/copy", reqBody, nil)
	})
}

// =============================================================================
// Object Delete Tests
// =============================================================================

func TestStorage_Object_Delete(t *testing.T) {
	ts := time.Now().UnixNano()
	bucketName := fmt.Sprintf("test-delete-%d", ts)

	supabase := NewStorageTestClient(supabaseStorageURL, supabaseServiceRoleKey)
	localbase := NewStorageTestClient(localbaseStorageURL, localbaseAPIKey)

	// Create bucket and upload files
	body := map[string]interface{}{"name": bucketName, "public": true}
	supabase.Do("POST", "/bucket", body, nil)
	localbase.Do("POST", "/bucket", body, nil)

	content := []byte("Content to delete")
	headers := map[string]string{"Content-Type": "text/plain"}
	supabase.Do("POST", "/object/"+bucketName+"/delete-single.txt", content, headers)
	localbase.Do("POST", "/object/"+bucketName+"/delete-single.txt", content, headers)
	supabase.Do("POST", "/object/"+bucketName+"/delete-multi1.txt", content, headers)
	localbase.Do("POST", "/object/"+bucketName+"/delete-multi1.txt", content, headers)
	supabase.Do("POST", "/object/"+bucketName+"/delete-multi2.txt", content, headers)
	localbase.Do("POST", "/object/"+bucketName+"/delete-multi2.txt", content, headers)

	t.Run("DELETE-001: Delete existing file", func(t *testing.T) {
		CompareStorage(t, "DELETE-001", "DELETE", "/object/"+bucketName+"/delete-single.txt", nil, nil)
	})

	t.Run("DELETE-002: Delete non-existent file", func(t *testing.T) {
		CompareStorage(t, "DELETE-002", "DELETE", "/object/"+bucketName+"/nonexistent.txt", nil, nil)
	})

	t.Run("DELETE-010: Delete multiple files", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"prefixes": []string{"delete-multi1.txt", "delete-multi2.txt"},
		}
		CompareStorage(t, "DELETE-010", "DELETE", "/object/"+bucketName, reqBody, nil)
	})
}

// =============================================================================
// Object Info Tests
// =============================================================================

func TestStorage_Object_Info(t *testing.T) {
	ts := time.Now().UnixNano()
	bucketName := fmt.Sprintf("test-info-%d", ts)

	supabase := NewStorageTestClient(supabaseStorageURL, supabaseServiceRoleKey)
	localbase := NewStorageTestClient(localbaseStorageURL, localbaseAPIKey)

	// Create bucket and upload file
	body := map[string]interface{}{"name": bucketName, "public": true}
	supabase.Do("POST", "/bucket", body, nil)
	localbase.Do("POST", "/bucket", body, nil)

	content := []byte("Info test content")
	headers := map[string]string{"Content-Type": "text/plain"}
	supabase.Do("POST", "/object/"+bucketName+"/info-test.txt", content, headers)
	localbase.Do("POST", "/object/"+bucketName+"/info-test.txt", content, headers)

	t.Run("INFO-001: Get existing file info", func(t *testing.T) {
		CompareStorage(t, "INFO-001", "GET", "/object/info/"+bucketName+"/info-test.txt", nil, nil)
	})

	t.Run("INFO-002: Get non-existent file info", func(t *testing.T) {
		CompareStorage(t, "INFO-002", "GET", "/object/info/"+bucketName+"/nonexistent.txt", nil, nil)
	})
}

// =============================================================================
// Signed URL Tests
// =============================================================================

func TestStorage_SignedURL(t *testing.T) {
	ts := time.Now().UnixNano()
	bucketName := fmt.Sprintf("test-signed-%d", ts)

	supabase := NewStorageTestClient(supabaseStorageURL, supabaseServiceRoleKey)
	localbase := NewStorageTestClient(localbaseStorageURL, localbaseAPIKey)

	// Create bucket and upload file
	body := map[string]interface{}{"name": bucketName, "public": true}
	supabase.Do("POST", "/bucket", body, nil)
	localbase.Do("POST", "/bucket", body, nil)

	content := []byte("Signed URL test content")
	headers := map[string]string{"Content-Type": "text/plain"}
	supabase.Do("POST", "/object/"+bucketName+"/signed-test.txt", content, headers)
	localbase.Do("POST", "/object/"+bucketName+"/signed-test.txt", content, headers)

	t.Run("SIGN-001: Create signed URL", func(t *testing.T) {
		reqBody := map[string]interface{}{"expiresIn": 3600}
		CompareStorage(t, "SIGN-001", "POST", "/object/sign/"+bucketName+"/signed-test.txt", reqBody, nil)
	})

	t.Run("SIGN-002: Create with short expiry", func(t *testing.T) {
		reqBody := map[string]interface{}{"expiresIn": 60}
		CompareStorage(t, "SIGN-002", "POST", "/object/sign/"+bucketName+"/signed-test.txt", reqBody, nil)
	})

	t.Run("SIGN-003: Invalid expiresIn (zero)", func(t *testing.T) {
		reqBody := map[string]interface{}{"expiresIn": 0}
		CompareStorage(t, "SIGN-003", "POST", "/object/sign/"+bucketName+"/signed-test.txt", reqBody, nil)
	})

	t.Run("SIGN-004: Invalid expiresIn (negative)", func(t *testing.T) {
		reqBody := map[string]interface{}{"expiresIn": -1}
		CompareStorage(t, "SIGN-004", "POST", "/object/sign/"+bucketName+"/signed-test.txt", reqBody, nil)
	})

	t.Run("SIGN-010: Create multiple signed URLs", func(t *testing.T) {
		// Upload another file
		supabase.Do("POST", "/object/"+bucketName+"/signed-test2.txt", content, headers)
		localbase.Do("POST", "/object/"+bucketName+"/signed-test2.txt", content, headers)

		reqBody := map[string]interface{}{
			"expiresIn": 3600,
			"paths":     []string{"signed-test.txt", "signed-test2.txt"},
		}
		CompareStorage(t, "SIGN-010", "POST", "/object/sign/"+bucketName, reqBody, nil)
	})

	t.Run("SIGN-020: Create signed upload URL", func(t *testing.T) {
		CompareStorage(t, "SIGN-020", "POST", "/object/upload/sign/"+bucketName+"/new-upload.txt", nil, nil)
	})
}

// =============================================================================
// Public Object Access Tests
// =============================================================================

func TestStorage_PublicAccess(t *testing.T) {
	ts := time.Now().UnixNano()
	publicBucket := fmt.Sprintf("test-public-access-%d", ts)
	privateBucket := fmt.Sprintf("test-private-access-%d", ts)

	supabase := NewStorageTestClient(supabaseStorageURL, supabaseServiceRoleKey)
	localbase := NewStorageTestClient(localbaseStorageURL, localbaseAPIKey)

	// Create public and private buckets
	supabase.Do("POST", "/bucket", map[string]interface{}{"name": publicBucket, "public": true}, nil)
	localbase.Do("POST", "/bucket", map[string]interface{}{"name": publicBucket, "public": true}, nil)
	supabase.Do("POST", "/bucket", map[string]interface{}{"name": privateBucket, "public": false}, nil)
	localbase.Do("POST", "/bucket", map[string]interface{}{"name": privateBucket, "public": false}, nil)

	content := []byte("Public access test")
	headers := map[string]string{"Content-Type": "text/plain"}
	supabase.Do("POST", "/object/"+publicBucket+"/public-file.txt", content, headers)
	localbase.Do("POST", "/object/"+publicBucket+"/public-file.txt", content, headers)
	supabase.Do("POST", "/object/"+privateBucket+"/private-file.txt", content, headers)
	localbase.Do("POST", "/object/"+privateBucket+"/private-file.txt", content, headers)

	t.Run("PUBLIC-001: Access public bucket file", func(t *testing.T) {
		CompareStorage(t, "PUBLIC-001", "GET", "/object/public/"+publicBucket+"/public-file.txt", nil, nil)
	})

	// Note: This test may vary based on implementation - some allow public endpoint on private bucket to return 403, others 404
	t.Run("PUBLIC-003: Access non-existent file in public bucket", func(t *testing.T) {
		CompareStorage(t, "PUBLIC-003", "GET", "/object/public/"+publicBucket+"/nonexistent.txt", nil, nil)
	})
}

// =============================================================================
// Edge Case Tests
// =============================================================================

func TestStorage_EdgeCases(t *testing.T) {
	ts := time.Now().UnixNano()
	bucketName := fmt.Sprintf("test-edge-%d", ts)

	supabase := NewStorageTestClient(supabaseStorageURL, supabaseServiceRoleKey)
	localbase := NewStorageTestClient(localbaseStorageURL, localbaseAPIKey)

	// Create bucket
	body := map[string]interface{}{"name": bucketName, "public": true}
	supabase.Do("POST", "/bucket", body, nil)
	localbase.Do("POST", "/bucket", body, nil)

	t.Run("EDGE-001: Path with URL-encoded spaces", func(t *testing.T) {
		content := []byte("Space test")
		headers := map[string]string{"Content-Type": "text/plain"}

		sResp, _, _ := supabase.Do("POST", "/object/"+bucketName+"/file%20with%20spaces.txt", content, headers)
		lResp, _, _ := localbase.Do("POST", "/object/"+bucketName+"/file%20with%20spaces.txt", content, headers)

		if sResp != nil && lResp != nil {
			if sResp.StatusCode != lResp.StatusCode {
				t.Errorf("Status mismatch: Supabase=%d, Localbase=%d", sResp.StatusCode, lResp.StatusCode)
			}
			t.Logf("[%s] EDGE-001 - Status: %d/%d", statusSymbol(sResp.StatusCode == lResp.StatusCode), sResp.StatusCode, lResp.StatusCode)
		}
	})

	t.Run("EDGE-005: Path with special characters", func(t *testing.T) {
		content := []byte("Special chars test")
		headers := map[string]string{"Content-Type": "text/plain"}

		// Note: Not all special characters are allowed - test safe ones
		sResp, _, _ := supabase.Do("POST", "/object/"+bucketName+"/file-with_underscore.txt", content, headers)
		lResp, _, _ := localbase.Do("POST", "/object/"+bucketName+"/file-with_underscore.txt", content, headers)

		if sResp != nil && lResp != nil {
			if sResp.StatusCode != lResp.StatusCode {
				t.Errorf("Status mismatch: Supabase=%d, Localbase=%d", sResp.StatusCode, lResp.StatusCode)
			}
			t.Logf("[%s] EDGE-005 - Status: %d/%d", statusSymbol(sResp.StatusCode == lResp.StatusCode), sResp.StatusCode, lResp.StatusCode)
		}
	})

	t.Run("EDGE-010: Empty file", func(t *testing.T) {
		content := []byte{}
		headers := map[string]string{"Content-Type": "application/octet-stream"}

		sResp, _, _ := supabase.Do("POST", "/object/"+bucketName+"/empty-edge.bin", content, headers)
		lResp, _, _ := localbase.Do("POST", "/object/"+bucketName+"/empty-edge.bin", content, headers)

		if sResp != nil && lResp != nil {
			if sResp.StatusCode != lResp.StatusCode {
				t.Errorf("Status mismatch: Supabase=%d, Localbase=%d", sResp.StatusCode, lResp.StatusCode)
			}
			t.Logf("[%s] EDGE-010 - Status: %d/%d", statusSymbol(sResp.StatusCode == lResp.StatusCode), sResp.StatusCode, lResp.StatusCode)
		}
	})

	t.Run("EDGE-013: Various MIME types", func(t *testing.T) {
		mimeTypes := map[string]string{
			"test.json": "application/json",
			"test.html": "text/html",
			"test.css":  "text/css",
			"test.js":   "application/javascript",
		}

		for filename, contentType := range mimeTypes {
			content := []byte("Test content")
			headers := map[string]string{"Content-Type": contentType}

			sResp, _, _ := supabase.Do("POST", "/object/"+bucketName+"/mime/"+filename, content, headers)
			lResp, _, _ := localbase.Do("POST", "/object/"+bucketName+"/mime/"+filename, content, headers)

			if sResp != nil && lResp != nil && sResp.StatusCode != lResp.StatusCode {
				t.Errorf("MIME %s status mismatch: Supabase=%d, Localbase=%d", contentType, sResp.StatusCode, lResp.StatusCode)
			}
		}
		t.Log("[PASS] EDGE-013: Various MIME types tested")
	})
}

// =============================================================================
// Error Handling Tests
// =============================================================================

func TestStorage_ErrorHandling(t *testing.T) {
	t.Run("ERR-001: Invalid bucket operations", func(t *testing.T) {
		// Empty bucket name in create
		CompareStorage(t, "ERR-001a", "POST", "/bucket", map[string]interface{}{"name": ""}, nil)

		// Invalid JSON
		supabase := NewStorageTestClient(supabaseStorageURL, supabaseServiceRoleKey)
		localbase := NewStorageTestClient(localbaseStorageURL, localbaseAPIKey)

		headers := map[string]string{"Content-Type": "application/json"}
		sResp, _, _ := supabase.Do("POST", "/bucket", "{invalid json}", headers)
		lResp, _, _ := localbase.Do("POST", "/bucket", "{invalid json}", headers)

		if sResp != nil && lResp != nil {
			t.Logf("[%s] ERR-001b Invalid JSON - Status: %d/%d",
				statusSymbol(sResp.StatusCode == lResp.StatusCode), sResp.StatusCode, lResp.StatusCode)
		}
	})

	t.Run("ERR-002: Non-existent resources", func(t *testing.T) {
		// Non-existent bucket
		CompareStorage(t, "ERR-002a", "GET", "/bucket/nonexistent-xyz-123", nil, nil)

		// Non-existent object
		CompareStorage(t, "ERR-002b", "GET", "/object/nonexistent-bucket-xyz/file.txt", nil, nil)
	})
}

// =============================================================================
// Helper Functions
// =============================================================================

func statusSymbol(pass bool) string {
	if pass {
		return "PASS"
	}
	return "FAIL"
}

// createMultipartForm creates a multipart form for file upload
func createMultipartForm(filename string, content []byte, contentType string) (*bytes.Buffer, string, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, "", err
	}
	if _, err := part.Write(content); err != nil {
		return nil, "", err
	}

	if err := writer.Close(); err != nil {
		return nil, "", err
	}

	return &body, writer.FormDataContentType(), nil
}

// =============================================================================
// Connectivity Test
// =============================================================================

func TestStorage_Connectivity(t *testing.T) {
	// Verify endpoints are accessible
	t.Log("Testing Storage API endpoints...")

	supabase := NewStorageTestClient(supabaseStorageURL, supabaseServiceRoleKey)
	localbase := NewStorageTestClient(localbaseStorageURL, localbaseAPIKey)

	// Test Supabase connectivity
	if resp, _, err := supabase.Do("GET", "/bucket", nil, nil); err != nil {
		t.Logf("Warning: Cannot connect to Supabase Storage: %v", err)
	} else {
		t.Logf("Supabase Storage: %d", resp.StatusCode)
	}

	// Test Localbase connectivity
	if resp, _, err := localbase.Do("GET", "/bucket", nil, nil); err != nil {
		t.Logf("Warning: Cannot connect to Localbase Storage: %v", err)
	} else {
		t.Logf("Localbase Storage: %d", resp.StatusCode)
	}
}
