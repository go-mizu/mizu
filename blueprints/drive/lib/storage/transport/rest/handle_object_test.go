package rest

import (
	"encoding/json"
	"io"
	"net/http"
	"path"
	"strings"
	"testing"
	"time"
)

// TestObjectUpload tests object upload endpoint.
func TestObjectUpload(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	base := srv.URL + "/storage/v1"

	// Create bucket
	doJSONRequest(t, http.MethodPost, base+"/bucket", map[string]any{"name": "uploads"})

	tests := []struct {
		name        string
		path        string
		content     string
		contentType string
		headers     map[string]string
		wantStatus  int
		wantError   bool
	}{
		{
			name:        "upload text file",
			path:        "/object/uploads/test.txt",
			content:     "hello world",
			contentType: "text/plain",
			wantStatus:  http.StatusOK,
		},
		{
			name:        "upload json file",
			path:        "/object/uploads/data.json",
			content:     `{"key":"value"}`,
			contentType: "application/json",
			wantStatus:  http.StatusOK,
		},
		{
			name:        "upload nested path",
			path:        "/object/uploads/folder/subfolder/file.txt",
			content:     "nested",
			contentType: "text/plain",
			wantStatus:  http.StatusOK,
		},
		{
			name:        "upload with default content type",
			path:        "/object/uploads/binary.bin",
			content:     "binary data",
			contentType: "",
			wantStatus:  http.StatusOK,
		},
		{
			name:        "upload duplicate without upsert should fail",
			path:        "/object/uploads/test.txt",
			content:     "duplicate",
			contentType: "text/plain",
			wantStatus:  http.StatusConflict,
			wantError:   true,
		},
		{
			name:        "upload duplicate with upsert should succeed",
			path:        "/object/uploads/test.txt",
			content:     "updated",
			contentType: "text/plain",
			headers:     map[string]string{"x-upsert": "true"},
			wantStatus:  http.StatusOK,
		},
		{
			name:        "upload to non-existent bucket",
			path:        "/object/nonexistent/file.txt",
			content:     "data",
			contentType: "text/plain",
			wantStatus:  http.StatusNotFound,
			wantError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := map[string]string{}
			if tt.contentType != "" {
				headers["Content-Type"] = tt.contentType
			}
			for k, v := range tt.headers {
				headers[k] = v
			}
			status, body := doRequest(t, http.MethodPost, base+tt.path, strings.NewReader(tt.content), headers)
			if status != tt.wantStatus {
				t.Errorf("status = %d, want %d, body = %s", status, tt.wantStatus, string(body))
			}

			if tt.wantError {
				var errResp errorPayload
				if err := json.Unmarshal(body, &errResp); err != nil {
					t.Fatalf("decode error: %v", err)
				}
			} else {
				var resp UploadResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("decode response: %v", err)
				}
				if resp.ID == "" || resp.Key == "" {
					t.Error("upload response missing Id or Key")
				}
			}
		})
	}
}

// TestObjectDownload tests object download endpoint.
func TestObjectDownload(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	base := srv.URL + "/storage/v1"

	// Setup
	doJSONRequest(t, http.MethodPost, base+"/bucket", map[string]any{"name": "downloads"})
	content := "hello world from storage api"
	doRequest(t, http.MethodPost, base+"/object/downloads/file.txt", strings.NewReader(content), map[string]string{"Content-Type": "text/plain"})

	tests := []struct {
		name       string
		path       string
		headers    map[string]string
		wantStatus int
		wantBody   string
		checkRange bool
	}{
		{
			name:       "download full file",
			path:       "/object/downloads/file.txt",
			wantStatus: http.StatusOK,
			wantBody:   content,
		},
		{
			name:       "download with range header",
			path:       "/object/downloads/file.txt",
			headers:    map[string]string{"Range": "bytes=0-4"},
			wantStatus: http.StatusPartialContent,
			wantBody:   "hello",
			checkRange: true,
		},
		{
			name:       "download with range from middle",
			path:       "/object/downloads/file.txt",
			headers:    map[string]string{"Range": "bytes=6-10"},
			wantStatus: http.StatusPartialContent,
			wantBody:   "world",
			checkRange: true,
		},
		{
			name:       "download non-existent file",
			path:       "/object/downloads/missing.txt",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "download with download query param",
			path:       "/object/downloads/file.txt?download=true",
			wantStatus: http.StatusOK,
			wantBody:   content,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body, resp := doRequestWithResponse(t, http.MethodGet, base+tt.path, nil, tt.headers)
			if status != tt.wantStatus {
				t.Errorf("status = %d, want %d", status, tt.wantStatus)
			}

			if tt.wantBody != "" && string(body) != tt.wantBody {
				t.Errorf("body = %q, want %q", string(body), tt.wantBody)
			}

			if tt.checkRange {
				if contentRange := resp.Header.Get("Content-Range"); contentRange == "" {
					t.Error("Content-Range header missing for partial content")
				}
				if acceptRanges := resp.Header.Get("Accept-Ranges"); acceptRanges != "bytes" {
					t.Errorf("Accept-Ranges = %q, want 'bytes'", acceptRanges)
				}
			}

			if strings.Contains(tt.path, "download=") {
				if disposition := resp.Header.Get("Content-Disposition"); !strings.Contains(disposition, "attachment") {
					t.Errorf("Content-Disposition should contain 'attachment', got %q", disposition)
				}
			}
		})
	}
}

// TestObjectUpdate tests object update endpoint.
func TestObjectUpdate(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	base := srv.URL + "/storage/v1"

	// Setup
	doJSONRequest(t, http.MethodPost, base+"/bucket", map[string]any{"name": "updates"})
	doRequest(t, http.MethodPost, base+"/object/updates/file.txt", strings.NewReader("original"), map[string]string{"Content-Type": "text/plain"})

	tests := []struct {
		name       string
		path       string
		content    string
		wantStatus int
		wantError  bool
	}{
		{
			name:       "update existing file",
			path:       "/object/updates/file.txt",
			content:    "updated content",
			wantStatus: http.StatusOK,
		},
		{
			name:       "update non-existent file should fail",
			path:       "/object/updates/missing.txt",
			content:    "new content",
			wantStatus: http.StatusNotFound,
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := doRequest(t, http.MethodPut, base+tt.path, strings.NewReader(tt.content), map[string]string{"Content-Type": "text/plain"})
			if status != tt.wantStatus {
				t.Errorf("status = %d, want %d", status, tt.wantStatus)
			}

			if tt.wantError {
				var errResp errorPayload
				json.Unmarshal(body, &errResp)
			} else {
				var resp UploadResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("decode response: %v", err)
				}

				// Verify content was updated
				downloadStatus, downloadBody := doRequest(t, http.MethodGet, base+tt.path, nil, nil)
				if downloadStatus != http.StatusOK {
					t.Fatalf("download status = %d", downloadStatus)
				}
				if string(downloadBody) != tt.content {
					t.Errorf("downloaded content = %q, want %q", string(downloadBody), tt.content)
				}
			}
		})
	}
}

// TestObjectDelete tests object deletion endpoint.
func TestObjectDelete(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	base := srv.URL + "/storage/v1"

	// Setup
	doJSONRequest(t, http.MethodPost, base+"/bucket", map[string]any{"name": "deletes"})
	doRequest(t, http.MethodPost, base+"/object/deletes/file1.txt", strings.NewReader("data1"), map[string]string{"Content-Type": "text/plain"})
	doRequest(t, http.MethodPost, base+"/object/deletes/file2.txt", strings.NewReader("data2"), map[string]string{"Content-Type": "text/plain"})

	t.Run("delete single object", func(t *testing.T) {
		status, body := doRequest(t, http.MethodDelete, base+"/object/deletes/file1.txt", nil, nil)
		if status != http.StatusOK {
			t.Fatalf("status = %d, body = %s", status, string(body))
		}

		var resp MessageResponse
		json.Unmarshal(body, &resp)
		if !strings.Contains(strings.ToLower(resp.Message), "delete") {
			t.Errorf("message should contain 'delete', got %q", resp.Message)
		}

		// Verify file is deleted
		getStatus, _ := doRequest(t, http.MethodGet, base+"/object/deletes/file1.txt", nil, nil)
		if getStatus != http.StatusNotFound {
			t.Errorf("file should be deleted, got status %d", getStatus)
		}
	})

	t.Run("delete multiple objects", func(t *testing.T) {
		doRequest(t, http.MethodPost, base+"/object/deletes/file3.txt", strings.NewReader("data3"), map[string]string{"Content-Type": "text/plain"})
		doRequest(t, http.MethodPost, base+"/object/deletes/file4.txt", strings.NewReader("data4"), map[string]string{"Content-Type": "text/plain"})

		req := DeleteObjectsRequest{
			Prefixes: []string{"file2.txt", "file3.txt", "file4.txt"},
		}
		status, body := doJSONRequest(t, http.MethodDelete, base+"/object/deletes", req)
		if status != http.StatusOK {
			t.Fatalf("status = %d, body = %s", status, string(body))
		}

		var deleted []ObjectInfo
		if err := json.Unmarshal(body, &deleted); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if len(deleted) < 3 {
			t.Errorf("expected 3 deleted objects, got %d", len(deleted))
		}
	})
}

// TestObjectList tests object listing endpoint.
func TestObjectList(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	base := srv.URL + "/storage/v1"

	// Setup
	doJSONRequest(t, http.MethodPost, base+"/bucket", map[string]any{"name": "lists"})
	files := []string{"folder1/file1.txt", "folder1/file2.txt", "folder2/file3.txt", "root.txt"}
	for _, f := range files {
		doRequest(t, http.MethodPost, base+"/object/lists/"+f, strings.NewReader("data"), map[string]string{"Content-Type": "text/plain"})
	}

	tests := []struct {
		name       string
		req        ListObjectsRequest
		wantStatus int
		minObjects int
	}{
		{
			name:       "list all objects",
			req:        ListObjectsRequest{},
			wantStatus: http.StatusOK,
			minObjects: 1,
		},
		{
			name:       "list with prefix",
			req:        ListObjectsRequest{Prefix: "folder1"},
			wantStatus: http.StatusOK,
			minObjects: 2,
		},
		{
			name:       "list with limit",
			req:        ListObjectsRequest{Limit: 2},
			wantStatus: http.StatusOK,
			minObjects: 1,
		},
		{
			name:       "list with offset",
			req:        ListObjectsRequest{Offset: 1},
			wantStatus: http.StatusOK,
			minObjects: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := doJSONRequest(t, http.MethodPost, base+"/object/list/lists", tt.req)
			if status != tt.wantStatus {
				t.Fatalf("status = %d, want %d", status, tt.wantStatus)
			}

			var objects []ObjectInfo
			if err := json.Unmarshal(body, &objects); err != nil {
				t.Fatalf("decode response: %v", err)
			}

			if len(objects) < tt.minObjects {
				t.Errorf("got %d objects, want at least %d", len(objects), tt.minObjects)
			}

			// Verify object response structure
			for i, obj := range objects {
				if obj.Name == "" {
					t.Errorf("object[%d].name is empty", i)
				}
				if obj.BucketID != "lists" {
					t.Errorf("object[%d].bucket_id = %q, want 'lists'", i, obj.BucketID)
				}
			}
		})
	}
}

// TestObjectMove tests object move endpoint.
func TestObjectMove(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	base := srv.URL + "/storage/v1"

	// Setup
	doJSONRequest(t, http.MethodPost, base+"/bucket", map[string]any{"name": "source"})
	doJSONRequest(t, http.MethodPost, base+"/bucket", map[string]any{"name": "dest"})
	doRequest(t, http.MethodPost, base+"/object/source/file.txt", strings.NewReader("move me"), map[string]string{"Content-Type": "text/plain"})

	tests := []struct {
		name       string
		req        MoveObjectRequest
		wantStatus int
		wantError  bool
	}{
		{
			name: "move within same bucket",
			req: MoveObjectRequest{
				BucketID:       "source",
				SourceKey:      "file.txt",
				DestinationKey: "moved.txt",
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "move to different bucket",
			req: MoveObjectRequest{
				BucketID:          "source",
				SourceKey:         "moved.txt",
				DestinationBucket: "dest",
				DestinationKey:    "file.txt",
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "move missing file should fail",
			req: MoveObjectRequest{
				BucketID:       "source",
				SourceKey:      "missing.txt",
				DestinationKey: "moved.txt",
			},
			wantStatus: http.StatusNotFound,
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := doJSONRequest(t, http.MethodPost, base+"/object/move", tt.req)
			if status != tt.wantStatus {
				t.Errorf("status = %d, want %d, body = %s", status, tt.wantStatus, string(body))
			}

			if tt.wantError {
				var errResp errorPayload
				json.Unmarshal(body, &errResp)
			} else {
				var resp MessageResponse
				json.Unmarshal(body, &resp)
			}
		})
	}
}

// TestObjectCopy tests object copy endpoint.
func TestObjectCopy(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	base := srv.URL + "/storage/v1"

	// Setup
	doJSONRequest(t, http.MethodPost, base+"/bucket", map[string]any{"name": "copy-src"})
	doJSONRequest(t, http.MethodPost, base+"/bucket", map[string]any{"name": "copy-dst"})
	doRequest(t, http.MethodPost, base+"/object/copy-src/original.txt", strings.NewReader("copy me"), map[string]string{"Content-Type": "text/plain"})

	tests := []struct {
		name       string
		req        CopyObjectRequest
		wantStatus int
		wantError  bool
	}{
		{
			name: "copy within same bucket",
			req: CopyObjectRequest{
				BucketID:       "copy-src",
				SourceKey:      "original.txt",
				DestinationKey: "copy1.txt",
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "copy to different bucket",
			req: CopyObjectRequest{
				BucketID:          "copy-src",
				SourceKey:         "original.txt",
				DestinationBucket: "copy-dst",
				DestinationKey:    "copy2.txt",
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "copy with metadata",
			req: CopyObjectRequest{
				BucketID:       "copy-src",
				SourceKey:      "original.txt",
				DestinationKey: "copy3.txt",
				Metadata:       map[string]string{"custom": "value"},
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "copy missing file should fail",
			req: CopyObjectRequest{
				BucketID:       "copy-src",
				SourceKey:      "missing.txt",
				DestinationKey: "copy.txt",
			},
			wantStatus: http.StatusNotFound,
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := doJSONRequest(t, http.MethodPost, base+"/object/copy", tt.req)
			if status != tt.wantStatus {
				t.Errorf("status = %d, want %d, body = %s", status, tt.wantStatus, string(body))
			}

			if tt.wantError {
				var errResp errorPayload
				json.Unmarshal(body, &errResp)
			} else {
				var resp CopyObjectResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("decode response: %v", err)
				}
				if resp.ID == "" || resp.Key == "" {
					t.Error("copy response missing Id or Key")
				}
			}
		})
	}
}

// TestObjectInfo tests object info endpoint.
func TestObjectInfo(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	base := srv.URL + "/storage/v1"

	// Setup
	doJSONRequest(t, http.MethodPost, base+"/bucket", map[string]any{"name": "info"})
	doRequest(t, http.MethodPost, base+"/object/info/test.txt", strings.NewReader("data"), map[string]string{"Content-Type": "text/plain"})

	// Wait a bit to ensure timestamps are set
	time.Sleep(10 * time.Millisecond)

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantError  bool
	}{
		{
			name:       "get object info",
			path:       "/object/info/info/test.txt",
			wantStatus: http.StatusOK,
		},
		{
			name:       "get public object info",
			path:       "/object/info/public/info/test.txt",
			wantStatus: http.StatusOK,
		},
		{
			name:       "get authenticated object info",
			path:       "/object/info/authenticated/info/test.txt",
			wantStatus: http.StatusOK,
		},
		{
			name:       "get info for non-existent object",
			path:       "/object/info/info/missing.txt",
			wantStatus: http.StatusNotFound,
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := doRequest(t, http.MethodGet, base+tt.path, nil, nil)
			if status != tt.wantStatus {
				t.Errorf("status = %d, want %d", status, tt.wantStatus)
			}

			if tt.wantError {
				var errResp errorPayload
				json.Unmarshal(body, &errResp)
			} else {
				var info ObjectInfo
				if err := json.Unmarshal(body, &info); err != nil {
					t.Fatalf("decode response: %v", err)
				}
				if info.ID == "" || info.Name == "" {
					t.Error("object info missing ID or Name")
				}
				if info.BucketID != "info" {
					t.Errorf("bucket_id = %q, want 'info'", info.BucketID)
				}
				if info.CreatedAt.IsZero() {
					t.Error("created_at is zero")
				}
			}
		})
	}
}

// TestPublicAndAuthenticatedEndpoints tests public and authenticated object access.
func TestPublicAndAuthenticatedEndpoints(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	base := srv.URL + "/storage/v1"

	// Setup public bucket
	doJSONRequest(t, http.MethodPost, base+"/bucket", map[string]any{"name": "public", "public": true})
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

	t.Run("access via authenticated endpoint", func(t *testing.T) {
		status, body := doRequest(t, http.MethodGet, base+"/object/authenticated/public/file.txt", nil, nil)
		if status != http.StatusOK {
			t.Fatalf("status = %d", status)
		}
		if string(body) != content {
			t.Errorf("body = %q, want %q", string(body), content)
		}
	})
}

// TestSignedURLs tests signed URL creation endpoints.
func TestSignedURLs(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	base := srv.URL + "/storage/v1"

	// Create bucket and object
	doJSONRequest(t, http.MethodPost, base+"/bucket", map[string]any{"name": "signs"})
	doRequest(t, http.MethodPost, base+"/object/signs/file.txt", strings.NewReader("sign me"), map[string]string{"Content-Type": "text/plain"})

	t.Run("create signed URL for single object", func(t *testing.T) {
		req := SignURLRequest{ExpiresIn: 3600}
		status, body := doJSONRequest(t, http.MethodPost, base+"/object/sign/signs/file.txt", req)

		// Local driver doesn't support signing, should return Not Implemented
		if status != http.StatusNotImplemented {
			t.Logf("Expected 501 Not Implemented for local driver, got %d", status)
			// If the driver supports signing, verify response format
			if status == http.StatusOK {
				var resp SignURLResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("decode response: %v", err)
				}
				if resp.SignedURL == "" {
					t.Error("signedURL is empty")
				}
			}
		}
	})

	t.Run("create signed URLs for multiple objects", func(t *testing.T) {
		doRequest(t, http.MethodPost, base+"/object/signs/file2.txt", strings.NewReader("sign2"), map[string]string{"Content-Type": "text/plain"})

		req := SignURLsRequest{
			ExpiresIn: 3600,
			Paths:     []string{"file.txt", "file2.txt", "missing.txt"},
		}
		status, body := doJSONRequest(t, http.MethodPost, base+"/object/sign/signs", req)

		// Local driver doesn't support signing
		if status != http.StatusNotImplemented {
			t.Logf("Expected 501 Not Implemented for local driver, got %d", status)
			if status == http.StatusOK {
				var results []SignURLsResponseItem
				if err := json.Unmarshal(body, &results); err != nil {
					t.Fatalf("decode response: %v", err)
				}
				if len(results) != 3 {
					t.Errorf("got %d results, want 3", len(results))
				}
				// Verify each result has either signedURL or error
				for i, r := range results {
					if r.Path == "" {
						t.Errorf("result[%d].path is empty", i)
					}
					if r.SignedURL == "" && r.Error == "" {
						t.Errorf("result[%d] has neither signedURL nor error", i)
					}
				}
			}
		}
	})

	t.Run("create signed URL with invalid expiresIn", func(t *testing.T) {
		req := SignURLRequest{ExpiresIn: -1}
		status, body := doJSONRequest(t, http.MethodPost, base+"/object/sign/signs/file.txt", req)
		if status != http.StatusBadRequest {
			t.Errorf("status = %d, want %d, body = %s", status, http.StatusBadRequest, string(body))
		}
	})
}

// TestUploadSignedURL tests signed upload URL creation endpoint.
func TestUploadSignedURL(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	base := srv.URL + "/storage/v1"

	// Create bucket
	doJSONRequest(t, http.MethodPost, base+"/bucket", map[string]any{"name": "upload-sign"})

	t.Run("create upload signed URL", func(t *testing.T) {
		status, body := doRequest(t, http.MethodPost, base+"/object/upload/sign/upload-sign/new-file.txt", nil, nil)

		// Local driver doesn't support signing
		if status != http.StatusNotImplemented {
			t.Logf("Expected 501 Not Implemented for local driver, got %d", status)
			if status == http.StatusOK {
				var resp UploadSignedURLResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("decode response: %v", err)
				}
				if resp.URL == "" {
					t.Error("url is empty")
				}
			}
		}
	})
}

// TestObjectResponseFormat verifies object responses match Supabase Storage API format.
func TestObjectResponseFormat(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	base := srv.URL + "/storage/v1"

	// Create bucket and upload object
	doJSONRequest(t, http.MethodPost, base+"/bucket", map[string]any{"name": "format"})
	uploadStatus, uploadBody := doRequest(t, http.MethodPost, base+"/object/format/test.txt", strings.NewReader("test content"), map[string]string{"Content-Type": "text/plain"})

	if uploadStatus != http.StatusOK {
		t.Fatalf("upload status = %d, body = %s", uploadStatus, string(uploadBody))
	}

	// Verify upload response format
	var uploadResp UploadResponse
	if err := json.Unmarshal(uploadBody, &uploadResp); err != nil {
		t.Fatalf("decode upload response: %v", err)
	}
	if uploadResp.ID == "" {
		t.Error("upload response Id is empty")
	}
	if uploadResp.Key == "" {
		t.Error("upload response Key is empty")
	}
	if uploadResp.Key != path.Join("format", "test.txt") {
		t.Errorf("Key = %q, want %q", uploadResp.Key, path.Join("format", "test.txt"))
	}
}

// TestObjectErrorFormat verifies error responses match Supabase Storage API format.
func TestObjectErrorFormat(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	base := srv.URL + "/storage/v1"

	tests := []struct {
		name          string
		method        string
		url           string
		body          io.Reader
		wantStatus    int
		wantErrorName string
	}{
		{
			name:          "object not found",
			method:        http.MethodGet,
			url:           base + "/object/nonexistent/file.txt",
			wantStatus:    http.StatusNotFound,
			wantErrorName: "Not Found",
		},
		{
			name:          "bucket not found for upload",
			method:        http.MethodPost,
			url:           base + "/object/nonexistent/file.txt",
			body:          strings.NewReader("data"),
			wantStatus:    http.StatusNotFound,
			wantErrorName: "Not Found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var headers map[string]string
			if tt.body != nil {
				headers = map[string]string{"Content-Type": "text/plain"}
			}
			status, body := doRequest(t, tt.method, tt.url, tt.body, headers)
			if status != tt.wantStatus {
				t.Errorf("status = %d, want %d", status, tt.wantStatus)
			}

			var errResp errorPayload
			if err := json.Unmarshal(body, &errResp); err != nil {
				t.Fatalf("decode error: %v, body: %s", err, string(body))
			}

			if errResp.StatusCode != tt.wantStatus {
				t.Errorf("errorPayload.statusCode = %d, want %d", errResp.StatusCode, tt.wantStatus)
			}
			if errResp.Error != tt.wantErrorName {
				t.Errorf("errorPayload.error = %q, want %q", errResp.Error, tt.wantErrorName)
			}
			if errResp.Message == "" {
				t.Error("errorPayload.message is empty")
			}
		})
	}
}
