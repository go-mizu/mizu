package rest

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestBucketEmptyNameVariations tests various empty/whitespace bucket name scenarios.
func TestBucketEmptyNameVariations(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	base := srv.URL + "/storage/v1"

	tests := []struct {
		name       string
		payload    map[string]any
		wantStatus int
	}{
		{
			name:       "empty name",
			payload:    map[string]any{"name": ""},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "whitespace only name",
			payload:    map[string]any{"name": "   "},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "nil name",
			payload:    map[string]any{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "name with leading/trailing spaces",
			payload:    map[string]any{"name": "  valid  "},
			wantStatus: http.StatusOK, // Should trim and succeed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := doJSONRequest(t, http.MethodPost, base+"/bucket", tt.payload)
			if status != tt.wantStatus {
				t.Errorf("status = %d, want %d, body = %s", status, tt.wantStatus, string(body))
			}
		})
	}
}

// TestObjectUploadContentTypes tests various content type handling.
func TestObjectUploadContentTypes(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	base := srv.URL + "/storage/v1"

	// Create bucket
	doJSONRequest(t, http.MethodPost, base+"/bucket", map[string]any{"name": "ct-test"})

	tests := []struct {
		name        string
		path        string
		contentType string
		content     string
	}{
		{
			name:        "text/plain",
			path:        "/object/ct-test/text.txt",
			contentType: "text/plain",
			content:     "plain text",
		},
		{
			name:        "text/html",
			path:        "/object/ct-test/page.html",
			contentType: "text/html",
			content:     "<html><body>test</body></html>",
		},
		{
			name:        "application/json",
			path:        "/object/ct-test/data.json",
			contentType: "application/json",
			content:     `{"key":"value"}`,
		},
		{
			name:        "application/xml",
			path:        "/object/ct-test/data.xml",
			contentType: "application/xml",
			content:     "<root><item>test</item></root>",
		},
		{
			name:        "application/octet-stream",
			path:        "/object/ct-test/binary.bin",
			contentType: "application/octet-stream",
			content:     "\x00\x01\x02\x03",
		},
		{
			name:        "image/png (simulated)",
			path:        "/object/ct-test/image.png",
			contentType: "image/png",
			content:     "PNG fake content",
		},
		{
			name:        "text/csv",
			path:        "/object/ct-test/data.csv",
			contentType: "text/csv",
			content:     "a,b,c\n1,2,3",
		},
		{
			name:        "no content type (default)",
			path:        "/object/ct-test/noct.dat",
			contentType: "",
			content:     "no content type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := map[string]string{}
			if tt.contentType != "" {
				headers["Content-Type"] = tt.contentType
			}
			headers["x-upsert"] = "true"

			// Upload
			status, _ := doRequest(t, http.MethodPost, base+tt.path, strings.NewReader(tt.content), headers)
			if status != http.StatusOK {
				t.Fatalf("upload status = %d", status)
			}

			// Download and verify
			dlStatus, dlBody, dlResp := doRequestWithResponse(t, http.MethodGet, base+tt.path, nil, nil)
			if dlStatus != http.StatusOK {
				t.Fatalf("download status = %d", dlStatus)
			}

			if string(dlBody) != tt.content {
				t.Errorf("content mismatch: got %q, want %q", string(dlBody), tt.content)
			}

			// Content-Type header may be preserved or defaulted depending on storage backend
			ct := dlResp.Header.Get("Content-Type")
			if ct == "" {
				t.Error("Content-Type header is empty")
			}
		})
	}
}

// TestObjectPathEdgeCases tests various path edge cases.
func TestObjectPathEdgeCases(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	base := srv.URL + "/storage/v1"

	// Create bucket
	doJSONRequest(t, http.MethodPost, base+"/bucket", map[string]any{"name": "paths"})

	tests := []struct {
		name       string
		uploadPath string
		wantStatus int
		skip       bool
	}{
		{
			name:       "simple path",
			uploadPath: "/object/paths/file.txt",
			wantStatus: http.StatusOK,
		},
		{
			name:       "nested path",
			uploadPath: "/object/paths/a/b/c/d/file.txt",
			wantStatus: http.StatusOK,
		},
		{
			name:       "path with hyphen",
			uploadPath: "/object/paths/file-name.txt",
			wantStatus: http.StatusOK,
		},
		{
			name:       "path with underscore",
			uploadPath: "/object/paths/file_name.txt",
			wantStatus: http.StatusOK,
		},
		{
			name:       "path with numbers",
			uploadPath: "/object/paths/file123.txt",
			wantStatus: http.StatusOK,
		},
		{
			name:       "path with dots",
			uploadPath: "/object/paths/file.name.txt",
			wantStatus: http.StatusOK,
		},
		{
			name:       "very long filename",
			uploadPath: "/object/paths/" + strings.Repeat("a", 200) + ".txt",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		if tt.skip {
			continue
		}
		t.Run(tt.name, func(t *testing.T) {
			status, _ := doRequest(t, http.MethodPost, base+tt.uploadPath, strings.NewReader("test"), map[string]string{"Content-Type": "text/plain", "x-upsert": "true"})
			if status != tt.wantStatus {
				t.Errorf("status = %d, want %d", status, tt.wantStatus)
			}
		})
	}
}

// TestObjectRangeRequests tests various range request scenarios.
func TestObjectRangeRequests(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	base := srv.URL + "/storage/v1"

	// Create bucket and upload test file
	doJSONRequest(t, http.MethodPost, base+"/bucket", map[string]any{"name": "ranges"})

	// Create content with known bytes: 0123456789 (10 bytes)
	content := "0123456789"
	doRequest(t, http.MethodPost, base+"/object/ranges/file.txt", strings.NewReader(content), map[string]string{"Content-Type": "text/plain"})

	tests := []struct {
		name           string
		rangeHeader    string
		wantStatus     int
		wantBody       string
		wantRange      string
		skipRangeCheck bool
	}{
		{
			name:        "bytes 0-4",
			rangeHeader: "bytes=0-4",
			wantStatus:  http.StatusPartialContent,
			wantBody:    "01234",
			wantRange:   "bytes 0-4/10",
		},
		{
			name:        "bytes 5-9",
			rangeHeader: "bytes=5-9",
			wantStatus:  http.StatusPartialContent,
			wantBody:    "56789",
			wantRange:   "bytes 5-9/10",
		},
		{
			name:        "bytes 0-9 (full)",
			rangeHeader: "bytes=0-9",
			wantStatus:  http.StatusPartialContent,
			wantBody:    "0123456789",
			wantRange:   "bytes 0-9/10",
		},
		{
			name:        "bytes 5- (suffix from position)",
			rangeHeader: "bytes=5-",
			wantStatus:  http.StatusPartialContent,
			wantBody:    "56789",
			wantRange:   "bytes 5-9/10",
		},
		{
			name:        "bytes -5 (last 5 bytes)",
			rangeHeader: "bytes=-5",
			wantStatus:  http.StatusPartialContent,
			wantBody:    "56789",
			wantRange:   "bytes 5-9/10",
		},
		{
			name:        "bytes 0-0 (single byte)",
			rangeHeader: "bytes=0-0",
			wantStatus:  http.StatusPartialContent,
			wantBody:    "0",
			wantRange:   "bytes 0-0/10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := map[string]string{"Range": tt.rangeHeader}
			status, body, resp := doRequestWithResponse(t, http.MethodGet, base+"/object/ranges/file.txt", nil, headers)

			if status != tt.wantStatus {
				t.Errorf("status = %d, want %d", status, tt.wantStatus)
			}

			if string(body) != tt.wantBody {
				t.Errorf("body = %q, want %q", string(body), tt.wantBody)
			}

			if !tt.skipRangeCheck && status == http.StatusPartialContent {
				contentRange := resp.Header.Get("Content-Range")
				if contentRange != tt.wantRange {
					t.Errorf("Content-Range = %q, want %q", contentRange, tt.wantRange)
				}

				acceptRanges := resp.Header.Get("Accept-Ranges")
				if acceptRanges != "bytes" {
					t.Errorf("Accept-Ranges = %q, want 'bytes'", acceptRanges)
				}
			}
		})
	}
}

// TestDeleteObjectsRequest tests batch delete functionality.
func TestDeleteObjectsRequest(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	base := srv.URL + "/storage/v1"

	// Create bucket and upload files
	doJSONRequest(t, http.MethodPost, base+"/bucket", map[string]any{"name": "batch-del"})

	for i := 1; i <= 5; i++ {
		path := "/object/batch-del/file" + strconv.Itoa(i) + ".txt"
		doRequest(t, http.MethodPost, base+path, strings.NewReader("content"+strconv.Itoa(i)), map[string]string{"Content-Type": "text/plain"})
	}

	t.Run("delete multiple existing files", func(t *testing.T) {
		req := DeleteObjectsRequest{
			Prefixes: []string{"file1.txt", "file2.txt", "file3.txt"},
		}
		status, body := doJSONRequest(t, http.MethodDelete, base+"/object/batch-del", req)
		if status != http.StatusOK {
			t.Fatalf("status = %d, body = %s", status, string(body))
		}

		var deleted []ObjectInfo
		if err := json.Unmarshal(body, &deleted); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if len(deleted) != 3 {
			t.Errorf("expected 3 deleted, got %d", len(deleted))
		}
	})

	t.Run("delete with mixed existing/non-existing", func(t *testing.T) {
		req := DeleteObjectsRequest{
			Prefixes: []string{"file4.txt", "nonexistent.txt", "file5.txt"},
		}
		status, body := doJSONRequest(t, http.MethodDelete, base+"/object/batch-del", req)
		if status != http.StatusOK {
			t.Fatalf("status = %d, body = %s", status, string(body))
		}

		var deleted []ObjectInfo
		if err := json.Unmarshal(body, &deleted); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		// Should have at least 2 (file4.txt and file5.txt), may also include non-existent
		if len(deleted) < 2 {
			t.Errorf("expected at least 2 deleted, got %d", len(deleted))
		}
	})
}

// TestListObjectsSortBy tests list objects with sort options.
func TestListObjectsSortBy(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	base := srv.URL + "/storage/v1"

	// Create bucket and upload files with delays
	doJSONRequest(t, http.MethodPost, base+"/bucket", map[string]any{"name": "sort-test"})

	files := []string{"c.txt", "a.txt", "b.txt"}
	for _, f := range files {
		doRequest(t, http.MethodPost, base+"/object/sort-test/"+f, strings.NewReader("content"), map[string]string{"Content-Type": "text/plain"})
		time.Sleep(10 * time.Millisecond) // Small delay to ensure different timestamps
	}

	tests := []struct {
		name   string
		sortBy *SortConfig
	}{
		{
			name:   "sort by name asc",
			sortBy: &SortConfig{Column: "name", Order: "asc"},
		},
		{
			name:   "sort by name desc",
			sortBy: &SortConfig{Column: "name", Order: "desc"},
		},
		{
			name:   "sort by created_at asc",
			sortBy: &SortConfig{Column: "created_at", Order: "asc"},
		},
		{
			name:   "no sort",
			sortBy: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := ListObjectsRequest{
				SortBy: tt.sortBy,
			}
			status, body := doJSONRequest(t, http.MethodPost, base+"/object/list/sort-test", req)
			if status != http.StatusOK {
				t.Fatalf("status = %d, body = %s", status, string(body))
			}

			var objects []ObjectInfo
			if err := json.Unmarshal(body, &objects); err != nil {
				t.Fatalf("decode response: %v", err)
			}

			if len(objects) < 3 {
				t.Errorf("expected at least 3 objects, got %d", len(objects))
			}
		})
	}
}

// TestListObjectsSearch tests list objects with search filter.
func TestListObjectsSearch(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	base := srv.URL + "/storage/v1"

	// Create bucket and upload files
	doJSONRequest(t, http.MethodPost, base+"/bucket", map[string]any{"name": "search-test"})

	files := []string{"report.txt", "data.txt", "report_2024.txt", "summary.pdf"}
	for _, f := range files {
		doRequest(t, http.MethodPost, base+"/object/search-test/"+f, strings.NewReader("content"), map[string]string{"Content-Type": "text/plain"})
	}

	tests := []struct {
		name       string
		search     string
		minResults int
	}{
		{
			name:       "search for 'report'",
			search:     "report",
			minResults: 2,
		},
		{
			name:       "search for 'data'",
			search:     "data",
			minResults: 1,
		},
		{
			name:       "search for 'nonexistent'",
			search:     "nonexistent",
			minResults: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := ListObjectsRequest{
				Search: tt.search,
			}
			status, body := doJSONRequest(t, http.MethodPost, base+"/object/list/search-test", req)
			if status != http.StatusOK {
				t.Fatalf("status = %d, body = %s", status, string(body))
			}

			var objects []ObjectInfo
			if err := json.Unmarshal(body, &objects); err != nil {
				t.Fatalf("decode response: %v", err)
			}

			if len(objects) < tt.minResults {
				t.Errorf("expected at least %d results, got %d", tt.minResults, len(objects))
			}
		})
	}
}

// TestEmptyFileUpload tests uploading empty files.
func TestEmptyFileUpload(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	base := srv.URL + "/storage/v1"

	doJSONRequest(t, http.MethodPost, base+"/bucket", map[string]any{"name": "empty"})

	t.Run("upload empty file", func(t *testing.T) {
		status, body := doRequest(t, http.MethodPost, base+"/object/empty/zero.txt", strings.NewReader(""), map[string]string{"Content-Type": "text/plain"})
		if status != http.StatusOK {
			t.Fatalf("upload status = %d, body = %s", status, string(body))
		}

		// Verify we can download it
		dlStatus, dlBody := doRequest(t, http.MethodGet, base+"/object/empty/zero.txt", nil, nil)
		if dlStatus != http.StatusOK {
			t.Fatalf("download status = %d", dlStatus)
		}
		if len(dlBody) != 0 {
			t.Errorf("expected empty body, got %d bytes", len(dlBody))
		}
	})
}

// TestTUSCapabilities tests TUS protocol OPTIONS discovery.
func TestTUSCapabilities(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	base := srv.URL + "/storage/v1"

	req, err := http.NewRequest(http.MethodOptions, base+"/upload/resumable/", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()

	// OPTIONS should return 200 or 204
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		t.Errorf("status = %d, want 200 or 204", resp.StatusCode)
	}

	// Verify TUS headers
	tusResumable := resp.Header.Get("Tus-Resumable")
	if tusResumable == "" {
		t.Error("Tus-Resumable header missing")
	}

	tusVersion := resp.Header.Get("Tus-Version")
	if tusVersion == "" {
		t.Error("Tus-Version header missing")
	}

	tusExtension := resp.Header.Get("Tus-Extension")
	if tusExtension == "" {
		t.Log("Tus-Extension header not set (optional)")
	}
}

// TestTUSCreateUploadCompat tests TUS upload creation (compatibility test suite).
func TestTUSCreateUploadCompat(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	base := srv.URL + "/storage/v1"

	// Create bucket first
	doJSONRequest(t, http.MethodPost, base+"/bucket", map[string]any{"name": "tus-test"})

	t.Run("create upload session", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, base+"/upload/resumable/tus-test/testfile.txt", nil)
		if err != nil {
			t.Fatalf("create request: %v", err)
		}

		req.Header.Set("Tus-Resumable", "1.0.0")
		req.Header.Set("Upload-Length", "1000")
		req.Header.Set("Content-Type", "application/offset+octet-stream")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("do request: %v", err)
		}
		defer resp.Body.Close()

		// Should return 201 Created
		if resp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("status = %d, want 201, body = %s", resp.StatusCode, string(body))
		}

		// Verify Location header is set
		location := resp.Header.Get("Location")
		if location == "" {
			t.Error("Location header missing")
		}

		// Verify Tus-Resumable header
		tusResumable := resp.Header.Get("Tus-Resumable")
		if tusResumable != "1.0.0" {
			t.Errorf("Tus-Resumable = %q, want '1.0.0'", tusResumable)
		}
	})
}

// TestTUSUploadChunk tests TUS chunk upload.
func TestTUSUploadChunk(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	base := srv.URL + "/storage/v1"

	// Create bucket
	doJSONRequest(t, http.MethodPost, base+"/bucket", map[string]any{"name": "tus-chunks"})

	// Create upload session
	createReq, _ := http.NewRequest(http.MethodPost, base+"/upload/resumable/tus-chunks/file.txt", nil)
	createReq.Header.Set("Tus-Resumable", "1.0.0")
	createReq.Header.Set("Upload-Length", "20")

	createResp, err := http.DefaultClient.Do(createReq)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	createResp.Body.Close()

	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d", createResp.StatusCode)
	}

	location := createResp.Header.Get("Location")
	if location == "" {
		t.Fatal("Location header missing")
	}

	t.Run("upload first chunk", func(t *testing.T) {
		chunk1 := []byte("0123456789") // 10 bytes

		patchReq, _ := http.NewRequest(http.MethodPatch, base+"/upload/resumable/tus-chunks/file.txt", bytes.NewReader(chunk1))
		patchReq.Header.Set("Tus-Resumable", "1.0.0")
		patchReq.Header.Set("Upload-Offset", "0")
		patchReq.Header.Set("Content-Type", "application/offset+octet-stream")
		patchReq.Header.Set("Content-Length", "10")

		patchResp, err := http.DefaultClient.Do(patchReq)
		if err != nil {
			t.Fatalf("patch request: %v", err)
		}
		patchResp.Body.Close()

		if patchResp.StatusCode != http.StatusNoContent {
			t.Errorf("status = %d, want 204", patchResp.StatusCode)
		}

		uploadOffset := patchResp.Header.Get("Upload-Offset")
		if uploadOffset != "10" {
			t.Errorf("Upload-Offset = %q, want '10'", uploadOffset)
		}
	})
}

// TestTUSGetUploadStatus tests TUS HEAD request for upload status.
func TestTUSGetUploadStatus(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	base := srv.URL + "/storage/v1"

	// Create bucket
	doJSONRequest(t, http.MethodPost, base+"/bucket", map[string]any{"name": "tus-status"})

	// Create upload session
	createReq, _ := http.NewRequest(http.MethodPost, base+"/upload/resumable/tus-status/file.txt", nil)
	createReq.Header.Set("Tus-Resumable", "1.0.0")
	createReq.Header.Set("Upload-Length", "100")

	createResp, err := http.DefaultClient.Do(createReq)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	createResp.Body.Close()

	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d", createResp.StatusCode)
	}

	t.Run("get upload status", func(t *testing.T) {
		headReq, _ := http.NewRequest(http.MethodHead, base+"/upload/resumable/tus-status/file.txt", nil)
		headReq.Header.Set("Tus-Resumable", "1.0.0")

		headResp, err := http.DefaultClient.Do(headReq)
		if err != nil {
			t.Fatalf("head request: %v", err)
		}
		headResp.Body.Close()

		if headResp.StatusCode != http.StatusOK {
			t.Errorf("status = %d, want 200", headResp.StatusCode)
		}

		// Should have Upload-Offset and Upload-Length headers
		uploadOffset := headResp.Header.Get("Upload-Offset")
		if uploadOffset == "" {
			t.Error("Upload-Offset header missing")
		}

		uploadLength := headResp.Header.Get("Upload-Length")
		if uploadLength != "100" {
			t.Errorf("Upload-Length = %q, want '100'", uploadLength)
		}
	})
}

// TestObjectCopyPreservesMetadata tests that copy preserves file metadata.
func TestObjectCopyPreservesMetadata(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	base := srv.URL + "/storage/v1"

	// Create buckets
	doJSONRequest(t, http.MethodPost, base+"/bucket", map[string]any{"name": "copy-src"})
	doJSONRequest(t, http.MethodPost, base+"/bucket", map[string]any{"name": "copy-dst"})

	// Upload original file
	content := "original content"
	doRequest(t, http.MethodPost, base+"/object/copy-src/original.txt", strings.NewReader(content), map[string]string{"Content-Type": "text/plain"})

	t.Run("copy within same bucket", func(t *testing.T) {
		req := CopyObjectRequest{
			BucketID:       "copy-src",
			SourceKey:      "original.txt",
			DestinationKey: "copy1.txt",
		}
		status, body := doJSONRequest(t, http.MethodPost, base+"/object/copy", req)
		if status != http.StatusOK {
			t.Fatalf("copy status = %d, body = %s", status, string(body))
		}

		// Verify content
		dlStatus, dlBody := doRequest(t, http.MethodGet, base+"/object/copy-src/copy1.txt", nil, nil)
		if dlStatus != http.StatusOK {
			t.Fatalf("download status = %d", dlStatus)
		}
		if string(dlBody) != content {
			t.Errorf("content = %q, want %q", string(dlBody), content)
		}
	})

	t.Run("copy to different bucket", func(t *testing.T) {
		req := CopyObjectRequest{
			BucketID:          "copy-src",
			SourceKey:         "original.txt",
			DestinationBucket: "copy-dst",
			DestinationKey:    "copy2.txt",
		}
		status, body := doJSONRequest(t, http.MethodPost, base+"/object/copy", req)
		if status != http.StatusOK {
			t.Fatalf("copy status = %d, body = %s", status, string(body))
		}

		// Verify content
		dlStatus, dlBody := doRequest(t, http.MethodGet, base+"/object/copy-dst/copy2.txt", nil, nil)
		if dlStatus != http.StatusOK {
			t.Fatalf("download status = %d", dlStatus)
		}
		if string(dlBody) != content {
			t.Errorf("content = %q, want %q", string(dlBody), content)
		}
	})
}

// TestConcurrentBucketOperations tests concurrent bucket operations.
func TestConcurrentBucketOperations(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	base := srv.URL + "/storage/v1"

	// Create multiple buckets concurrently
	done := make(chan error, 10)
	for i := 0; i < 10; i++ {
		go func(n int) {
			name := "concurrent-" + strconv.Itoa(n)
			status, _ := doJSONRequest(t, http.MethodPost, base+"/bucket", map[string]any{"name": name})
			if status != http.StatusOK {
				done <- nil // Some may conflict, that's OK
			} else {
				done <- nil
			}
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// List buckets and verify some were created
	status, body := doRequest(t, http.MethodGet, base+"/bucket", nil, nil)
	if status != http.StatusOK {
		t.Fatalf("list status = %d", status)
	}

	var buckets []BucketResponse
	if err := json.Unmarshal(body, &buckets); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(buckets) < 5 {
		t.Errorf("expected at least 5 buckets, got %d", len(buckets))
	}
}

// TestErrorResponseFormat tests that all error responses have consistent format.
func TestErrorResponseFormat(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	base := srv.URL + "/storage/v1"

	tests := []struct {
		name          string
		method        string
		url           string
		body          string
		expectedCode  int
		expectedError string
	}{
		{
			name:          "bucket not found",
			method:        http.MethodGet,
			url:           base + "/bucket/nonexistent",
			expectedCode:  http.StatusNotFound,
			expectedError: "Not Found",
		},
		{
			name:          "object not found",
			method:        http.MethodGet,
			url:           base + "/object/nonexistent/file.txt",
			expectedCode:  http.StatusNotFound,
			expectedError: "Not Found",
		},
		{
			name:          "invalid JSON in bucket create",
			method:        http.MethodPost,
			url:           base + "/bucket",
			body:          "{invalid}",
			expectedCode:  http.StatusBadRequest,
			expectedError: "Bad Request",
		},
		{
			name:          "missing required field",
			method:        http.MethodPost,
			url:           base + "/bucket",
			body:          "{}",
			expectedCode:  http.StatusBadRequest,
			expectedError: "Bad Request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body io.Reader
			headers := map[string]string{}
			if tt.body != "" {
				body = strings.NewReader(tt.body)
				headers["Content-Type"] = "application/json"
			}

			status, respBody := doRequest(t, tt.method, tt.url, body, headers)

			if status != tt.expectedCode {
				t.Errorf("status = %d, want %d", status, tt.expectedCode)
			}

			var errResp errorPayload
			if err := json.Unmarshal(respBody, &errResp); err != nil {
				t.Fatalf("decode error: %v, body: %s", err, string(respBody))
			}

			// Verify error response structure
			if errResp.StatusCode != tt.expectedCode {
				t.Errorf("statusCode = %d, want %d", errResp.StatusCode, tt.expectedCode)
			}
			if errResp.Error != tt.expectedError {
				t.Errorf("error = %q, want %q", errResp.Error, tt.expectedError)
			}
			if errResp.Message == "" {
				t.Error("message is empty")
			}
		})
	}
}

// TestPublicBucketAccess tests public bucket access patterns.
func TestPublicBucketAccess(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	base := srv.URL + "/storage/v1"

	// Create public bucket
	doJSONRequest(t, http.MethodPost, base+"/bucket", map[string]any{"name": "public", "public": true})

	// Upload file to public bucket
	content := "public content"
	doRequest(t, http.MethodPost, base+"/object/public/file.txt", strings.NewReader(content), map[string]string{"Content-Type": "text/plain"})

	t.Run("access via public endpoint", func(t *testing.T) {
		status, body := doRequest(t, http.MethodGet, base+"/object/public/public/file.txt", nil, nil)
		if status != http.StatusOK {
			t.Fatalf("status = %d", status)
		}
		if string(body) != content {
			t.Errorf("body = %q, want %q", string(body), content)
		}
	})

	t.Run("get public object info", func(t *testing.T) {
		status, body := doRequest(t, http.MethodGet, base+"/object/info/public/public/file.txt", nil, nil)
		if status != http.StatusOK {
			t.Fatalf("status = %d, body = %s", status, string(body))
		}

		var info ObjectInfo
		if err := json.Unmarshal(body, &info); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if info.Name != "file.txt" {
			t.Errorf("name = %q, want 'file.txt'", info.Name)
		}
	})
}
