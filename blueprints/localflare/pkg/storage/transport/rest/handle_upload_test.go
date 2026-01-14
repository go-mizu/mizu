package rest

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/drive/lib/storage/driver/local"
	"github.com/go-mizu/mizu"
)

func newTUSTestServer(t *testing.T) (*httptest.Server, func()) {
	t.Helper()

	ctx := context.Background()
	store, err := local.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("open local storage: %v", err)
	}

	app := mizu.New()
	Register(app, "/storage/v1", store)

	srv := httptest.NewServer(app)

	cleanup := func() {
		srv.Close()
		_ = store.Close()
		ClearUploadStates()
	}

	return srv, cleanup
}

func TestTUSOptions(t *testing.T) {
	srv, cleanup := newTUSTestServer(t)
	defer cleanup()

	tests := []struct {
		name string
		path string
	}{
		{"base path", "/storage/v1/upload/resumable/"},
		{"with bucket and path", "/storage/v1/upload/resumable/test-bucket/test-file.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodOptions, srv.URL+tt.path, nil)
			if err != nil {
				t.Fatalf("create request: %v", err)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("do request: %v", err)
			}
			defer func() {
				_ = resp.Body.Close()
			}()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
			}

			// Check TUS headers
			if got := resp.Header.Get("Tus-Resumable"); got != tusVersion {
				t.Errorf("Tus-Resumable = %q, want %q", got, tusVersion)
			}
			if got := resp.Header.Get("Tus-Version"); got != tusVersion {
				t.Errorf("Tus-Version = %q, want %q", got, tusVersion)
			}
			if got := resp.Header.Get("Tus-Extension"); got != tusExtensions {
				t.Errorf("Tus-Extension = %q, want %q", got, tusExtensions)
			}
			if got := resp.Header.Get("Tus-Max-Size"); got != strconv.FormatInt(tusMaxSize, 10) {
				t.Errorf("Tus-Max-Size = %q, want %d", got, tusMaxSize)
			}

			// Check CORS headers
			if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "*" {
				t.Errorf("Access-Control-Allow-Origin = %q, want *", got)
			}
			if got := resp.Header.Get("Access-Control-Allow-Methods"); got == "" {
				t.Error("Access-Control-Allow-Methods is empty")
			}
		})
	}
}

func TestTUSCreateUpload(t *testing.T) {
	srv, cleanup := newTUSTestServer(t)
	defer cleanup()

	base := srv.URL + "/storage/v1"

	// Create bucket first
	createBucket(t, base, "test-bucket")

	tests := []struct {
		name           string
		path           string
		uploadLength   string
		deferLength    string
		metadata       string
		wantStatus     int
		checkHeaders   bool
		wantLocationOK bool
	}{
		{
			name:           "valid upload with length",
			path:           "/storage/v1/upload/resumable/test-bucket/test-file.txt",
			uploadLength:   "100",
			wantStatus:     http.StatusCreated,
			checkHeaders:   true,
			wantLocationOK: true,
		},
		{
			name:           "valid upload with metadata",
			path:           "/storage/v1/upload/resumable/test-bucket/file-with-metadata.txt",
			uploadLength:   "200",
			metadata:       "filename dGVzdC5wZGY=,content-type dGV4dC9wbGFpbg==",
			wantStatus:     http.StatusCreated,
			checkHeaders:   true,
			wantLocationOK: true,
		},
		{
			name:           "valid upload with deferred length",
			path:           "/storage/v1/upload/resumable/test-bucket/deferred.txt",
			deferLength:    "1",
			wantStatus:     http.StatusCreated,
			checkHeaders:   true,
			wantLocationOK: true,
		},
		{
			name:       "missing Tus-Resumable header",
			path:       "/storage/v1/upload/resumable/test-bucket/no-tus.txt",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:         "missing upload length and defer length",
			path:         "/storage/v1/upload/resumable/test-bucket/no-length.txt",
			uploadLength: "",
			wantStatus:   http.StatusBadRequest,
		},
		{
			name:         "invalid upload length",
			path:         "/storage/v1/upload/resumable/test-bucket/invalid-length.txt",
			uploadLength: "invalid",
			wantStatus:   http.StatusBadRequest,
		},
		{
			name:         "upload exceeds max size",
			path:         "/storage/v1/upload/resumable/test-bucket/too-large.txt",
			uploadLength: strconv.FormatInt(tusMaxSize+1, 10),
			wantStatus:   http.StatusRequestEntityTooLarge,
		},
		{
			name:         "bucket does not exist",
			path:         "/storage/v1/upload/resumable/nonexistent-bucket/file.txt",
			uploadLength: "100",
			wantStatus:   http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost, srv.URL+tt.path, nil)
			if err != nil {
				t.Fatalf("create request: %v", err)
			}

			// Set headers conditionally
			if tt.name != "missing Tus-Resumable header" {
				req.Header.Set("Tus-Resumable", tusVersion)
			}
			if tt.uploadLength != "" {
				req.Header.Set("Upload-Length", tt.uploadLength)
			}
			if tt.deferLength != "" {
				req.Header.Set("Upload-Defer-Length", tt.deferLength)
			}
			if tt.metadata != "" {
				req.Header.Set("Upload-Metadata", tt.metadata)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("do request: %v", err)
			}
			defer func() {
				_ = resp.Body.Close()
			}()

			if resp.StatusCode != tt.wantStatus {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("status = %d, want %d, body = %s", resp.StatusCode, tt.wantStatus, string(body))
			}

			if tt.checkHeaders {
				if got := resp.Header.Get("Tus-Resumable"); got != tusVersion {
					t.Errorf("Tus-Resumable = %q, want %q", got, tusVersion)
				}
				if got := resp.Header.Get("Upload-Offset"); got != "0" {
					t.Errorf("Upload-Offset = %q, want 0", got)
				}
				if tt.uploadLength != "" {
					if got := resp.Header.Get("Upload-Length"); got != tt.uploadLength {
						t.Errorf("Upload-Length = %q, want %q", got, tt.uploadLength)
					}
				}
			}

			if tt.wantLocationOK {
				location := resp.Header.Get("Location")
				if location == "" {
					t.Error("Location header is empty")
				}
				if !strings.HasPrefix(location, "/upload/resumable/") {
					t.Errorf("Location = %q, expected to start with /upload/resumable/", location)
				}
			}
		})
	}
}

func TestTUSPatchUpload(t *testing.T) {
	srv, cleanup := newTUSTestServer(t)
	defer cleanup()

	base := srv.URL + "/storage/v1"

	// Create bucket
	createBucket(t, base, "test-bucket")

	// Create upload
	objectPath := "test-file.txt"
	uploadPath := fmt.Sprintf("/storage/v1/upload/resumable/test-bucket/%s", objectPath)
	createUpload(t, srv.URL+uploadPath, "100")

	tests := []struct {
		name         string
		path         string
		uploadOffset string
		contentType  string
		body         string
		wantStatus   int
		wantOffset   string
	}{
		{
			name:         "valid first chunk",
			path:         uploadPath,
			uploadOffset: "0",
			contentType:  chunkContentType,
			body:         "hello",
			wantStatus:   http.StatusNoContent,
			wantOffset:   "5",
		},
		{
			name:         "valid second chunk",
			path:         uploadPath,
			uploadOffset: "5",
			contentType:  chunkContentType,
			body:         " world",
			wantStatus:   http.StatusNoContent,
			wantOffset:   "11",
		},
		{
			name:         "missing Upload-Offset",
			path:         uploadPath,
			uploadOffset: "",
			contentType:  chunkContentType,
			body:         "data",
			wantStatus:   http.StatusBadRequest,
		},
		{
			name:         "wrong Content-Type",
			path:         uploadPath,
			uploadOffset: "11",
			contentType:  "text/plain",
			body:         "data",
			wantStatus:   http.StatusUnsupportedMediaType,
		},
		{
			name:         "offset mismatch",
			path:         uploadPath,
			uploadOffset: "999",
			contentType:  chunkContentType,
			body:         "data",
			wantStatus:   http.StatusConflict,
		},
		{
			name:         "upload not found",
			path:         "/storage/v1/upload/resumable/test-bucket/nonexistent.txt",
			uploadOffset: "0",
			contentType:  chunkContentType,
			body:         "data",
			wantStatus:   http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPatch, srv.URL+tt.path, strings.NewReader(tt.body))
			if err != nil {
				t.Fatalf("create request: %v", err)
			}

			req.Header.Set("Tus-Resumable", tusVersion)
			if tt.uploadOffset != "" {
				req.Header.Set("Upload-Offset", tt.uploadOffset)
			}
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("do request: %v", err)
			}
			defer func() {
				_ = resp.Body.Close()
			}()

			if resp.StatusCode != tt.wantStatus {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("status = %d, want %d, body = %s", resp.StatusCode, tt.wantStatus, string(body))
			}

			if tt.wantStatus == http.StatusNoContent && tt.wantOffset != "" {
				if got := resp.Header.Get("Upload-Offset"); got != tt.wantOffset {
					t.Errorf("Upload-Offset = %q, want %q", got, tt.wantOffset)
				}
			}
		})
	}
}

func TestTUSCompleteUpload(t *testing.T) {
	srv, cleanup := newTUSTestServer(t)
	defer cleanup()

	base := srv.URL + "/storage/v1"

	// Create bucket
	createBucket(t, base, "test-bucket")

	objectPath := "complete-file.txt"
	uploadPath := fmt.Sprintf("/storage/v1/upload/resumable/test-bucket/%s", objectPath)

	// Create upload with known length
	fileContent := "Hello, World!"
	uploadLength := len(fileContent)
	createUpload(t, srv.URL+uploadPath, strconv.Itoa(uploadLength))

	// Upload the content
	req, err := http.NewRequest(http.MethodPatch, srv.URL+uploadPath, strings.NewReader(fileContent))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Tus-Resumable", tusVersion)
	req.Header.Set("Upload-Offset", "0")
	req.Header.Set("Content-Type", chunkContentType)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("upload status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}

	// Wait a bit for the file to be written
	time.Sleep(100 * time.Millisecond)

	// Verify the file was created in the bucket
	downloadPath := fmt.Sprintf("/storage/v1/object/test-bucket/%s", objectPath)
	downloadReq, err := http.NewRequest(http.MethodGet, srv.URL+downloadPath, nil)
	if err != nil {
		t.Fatalf("create download request: %v", err)
	}

	downloadResp, err := http.DefaultClient.Do(downloadReq)
	if err != nil {
		t.Fatalf("do download request: %v", err)
	}
	defer func() {
		_ = downloadResp.Body.Close()
	}()

	if downloadResp.StatusCode != http.StatusOK {
		t.Fatalf("download status = %d, want %d", downloadResp.StatusCode, http.StatusOK)
	}

	body, err := io.ReadAll(downloadResp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	if string(body) != fileContent {
		t.Errorf("downloaded content = %q, want %q", string(body), fileContent)
	}

	// Verify upload state is cleaned up
	state := GetUploadState("test-bucket", objectPath)
	if state != nil {
		t.Error("upload state should be cleaned up after completion")
	}
}

func TestTUSHead(t *testing.T) {
	srv, cleanup := newTUSTestServer(t)
	defer cleanup()

	base := srv.URL + "/storage/v1"

	// Create bucket
	createBucket(t, base, "test-bucket")

	objectPath := "head-test.txt"
	uploadPath := fmt.Sprintf("/storage/v1/upload/resumable/test-bucket/%s", objectPath)

	// Create upload
	createUpload(t, srv.URL+uploadPath, "100")

	// Upload partial data
	patchReq, err := http.NewRequest(http.MethodPatch, srv.URL+uploadPath, strings.NewReader("hello"))
	if err != nil {
		t.Fatalf("create patch request: %v", err)
	}
	patchReq.Header.Set("Tus-Resumable", tusVersion)
	patchReq.Header.Set("Upload-Offset", "0")
	patchReq.Header.Set("Content-Type", chunkContentType)

	patchResp, err := http.DefaultClient.Do(patchReq)
	if err != nil {
		t.Fatalf("do patch request: %v", err)
	}
	_ = patchResp.Body.Close()

	// HEAD request
	headReq, err := http.NewRequest(http.MethodHead, srv.URL+uploadPath, nil)
	if err != nil {
		t.Fatalf("create head request: %v", err)
	}
	headReq.Header.Set("Tus-Resumable", tusVersion)

	headResp, err := http.DefaultClient.Do(headReq)
	if err != nil {
		t.Fatalf("do head request: %v", err)
	}
	defer func() {
		_ = headResp.Body.Close()
	}()

	if headResp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", headResp.StatusCode, http.StatusOK)
	}

	// Check headers
	if got := headResp.Header.Get("Tus-Resumable"); got != tusVersion {
		t.Errorf("Tus-Resumable = %q, want %q", got, tusVersion)
	}
	if got := headResp.Header.Get("Upload-Offset"); got != "5" {
		t.Errorf("Upload-Offset = %q, want 5", got)
	}
	if got := headResp.Header.Get("Upload-Length"); got != "100" {
		t.Errorf("Upload-Length = %q, want 100", got)
	}
	if got := headResp.Header.Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", got)
	}

	// HEAD for nonexistent upload
	headReq2, err := http.NewRequest(http.MethodHead, srv.URL+"/storage/v1/upload/resumable/test-bucket/nonexistent.txt", nil)
	if err != nil {
		t.Fatalf("create head request 2: %v", err)
	}
	headReq2.Header.Set("Tus-Resumable", tusVersion)

	headResp2, err := http.DefaultClient.Do(headReq2)
	if err != nil {
		t.Fatalf("do head request 2: %v", err)
	}
	defer func() {
		_ = headResp2.Body.Close()
	}()

	if headResp2.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", headResp2.StatusCode, http.StatusNotFound)
	}
}

func TestTUSDelete(t *testing.T) {
	srv, cleanup := newTUSTestServer(t)
	defer cleanup()

	base := srv.URL + "/storage/v1"

	// Create bucket
	createBucket(t, base, "test-bucket")

	objectPath := "delete-test.txt"
	uploadPath := fmt.Sprintf("/storage/v1/upload/resumable/test-bucket/%s", objectPath)

	// Create upload
	createUpload(t, srv.URL+uploadPath, "100")

	// Get temp file path before deletion
	state := GetUploadState("test-bucket", objectPath)
	if state == nil {
		t.Fatal("upload state not found")
	}
	tempFile := state.TempFile

	// Verify temp file exists
	if _, err := os.Stat(tempFile); os.IsNotExist(err) {
		t.Fatal("temp file does not exist")
	}

	// Delete upload
	delReq, err := http.NewRequest(http.MethodDelete, srv.URL+uploadPath, nil)
	if err != nil {
		t.Fatalf("create delete request: %v", err)
	}
	delReq.Header.Set("Tus-Resumable", tusVersion)

	delResp, err := http.DefaultClient.Do(delReq)
	if err != nil {
		t.Fatalf("do delete request: %v", err)
	}
	defer func() {
		_ = delResp.Body.Close()
	}()

	if delResp.StatusCode != http.StatusNoContent {
		t.Errorf("status = %d, want %d", delResp.StatusCode, http.StatusNoContent)
	}

	// Verify upload state is cleaned up
	state = GetUploadState("test-bucket", objectPath)
	if state != nil {
		t.Error("upload state should be cleaned up after deletion")
	}

	// Verify temp file is deleted
	if _, err := os.Stat(tempFile); !os.IsNotExist(err) {
		t.Error("temp file should be deleted")
	}

	// Delete nonexistent upload
	delReq2, err := http.NewRequest(http.MethodDelete, srv.URL+uploadPath, nil)
	if err != nil {
		t.Fatalf("create delete request 2: %v", err)
	}
	delReq2.Header.Set("Tus-Resumable", tusVersion)

	delResp2, err := http.DefaultClient.Do(delReq2)
	if err != nil {
		t.Fatalf("do delete request 2: %v", err)
	}
	defer func() {
		_ = delResp2.Body.Close()
	}()

	if delResp2.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", delResp2.StatusCode, http.StatusNotFound)
	}
}

func TestTUSUpsert(t *testing.T) {
	srv, cleanup := newTUSTestServer(t)
	defer cleanup()

	base := srv.URL + "/storage/v1"

	// Create bucket
	createBucket(t, base, "test-bucket")

	objectPath := "upsert-test.txt"
	uploadPath := fmt.Sprintf("/storage/v1/upload/resumable/test-bucket/%s", objectPath)

	// First upload
	fileContent1 := "First version"
	createUploadWithUpsert(t, srv.URL+uploadPath, strconv.Itoa(len(fileContent1)), true)
	uploadChunks(t, srv.URL+uploadPath, []string{fileContent1})

	// Wait for upload to complete
	time.Sleep(100 * time.Millisecond)

	// Download and verify first version
	content1 := downloadObject(t, base, "test-bucket", objectPath)
	if content1 != fileContent1 {
		t.Errorf("first version = %q, want %q", content1, fileContent1)
	}

	// Second upload with upsert
	fileContent2 := "Second version!"
	createUploadWithUpsert(t, srv.URL+uploadPath, strconv.Itoa(len(fileContent2)), true)
	uploadChunks(t, srv.URL+uploadPath, []string{fileContent2})

	// Wait for upload to complete
	time.Sleep(100 * time.Millisecond)

	// Download and verify second version
	content2 := downloadObject(t, base, "test-bucket", objectPath)
	if content2 != fileContent2 {
		t.Errorf("second version = %q, want %q", content2, fileContent2)
	}
}

func TestTUSMultiChunkUpload(t *testing.T) {
	srv, cleanup := newTUSTestServer(t)
	defer cleanup()

	base := srv.URL + "/storage/v1"

	// Create bucket
	createBucket(t, base, "test-bucket")

	objectPath := "multi-chunk.txt"
	uploadPath := fmt.Sprintf("/storage/v1/upload/resumable/test-bucket/%s", objectPath)

	// Create upload
	chunks := []string{"Hello", ", ", "World", "!"}
	totalSize := 0
	for _, chunk := range chunks {
		totalSize += len(chunk)
	}

	createUpload(t, srv.URL+uploadPath, strconv.Itoa(totalSize))

	// Upload chunks sequentially
	uploadChunks(t, srv.URL+uploadPath, chunks)

	// Wait for upload to complete
	time.Sleep(100 * time.Millisecond)

	// Download and verify
	content := downloadObject(t, base, "test-bucket", objectPath)
	expected := strings.Join(chunks, "")
	if content != expected {
		t.Errorf("content = %q, want %q", content, expected)
	}
}

func TestParseUploadMetadata(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected map[string]string
	}{
		{
			name:     "empty",
			header:   "",
			expected: map[string]string{},
		},
		{
			name:   "single key-value",
			header: "filename dGVzdC50eHQ=",
			expected: map[string]string{
				"filename": "test.txt",
			},
		},
		{
			name:   "multiple key-values",
			header: "filename dGVzdC50eHQ=,content-type dGV4dC9wbGFpbg==",
			expected: map[string]string{
				"filename":     "test.txt",
				"content-type": "text/plain",
			},
		},
		{
			name:   "key without value",
			header: "empty-value",
			expected: map[string]string{
				"empty-value": "",
			},
		},
		{
			name:   "mixed",
			header: "filename dGVzdC5wZGY=,size,type YXBwbGljYXRpb24vcGRm",
			expected: map[string]string{
				"filename": "test.pdf",
				"size":     "",
				"type":     "application/pdf",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseUploadMetadata(tt.header)
			if len(result) != len(tt.expected) {
				t.Errorf("length = %d, want %d", len(result), len(tt.expected))
			}
			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("key %q: got %q, want %q", k, result[k], v)
				}
			}
		})
	}
}

func TestTUSVersionMismatch(t *testing.T) {
	srv, cleanup := newTUSTestServer(t)
	defer cleanup()

	base := srv.URL + "/storage/v1"

	// Create bucket
	createBucket(t, base, "test-bucket")

	uploadPath := "/storage/v1/upload/resumable/test-bucket/version-test.txt"

	// Try to create upload with wrong version
	req, err := http.NewRequest(http.MethodPost, srv.URL+uploadPath, nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Tus-Resumable", "2.0.0")
	req.Header.Set("Upload-Length", "100")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusPreconditionFailed {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusPreconditionFailed)
	}

	// Check Tus-Version header in error response
	if got := resp.Header.Get("Tus-Version"); got != tusVersion {
		t.Errorf("Tus-Version = %q, want %q", got, tusVersion)
	}
}

// Helper functions

func createBucket(t *testing.T, baseURL, bucketName string) {
	t.Helper()

	buf := &bytes.Buffer{}
	fmt.Fprintf(buf, `{"name": "%s"}`, bucketName)

	req, err := http.NewRequest(http.MethodPost, baseURL+"/bucket", buf)
	if err != nil {
		t.Fatalf("create bucket request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do bucket request: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("create bucket status = %d, body = %s", resp.StatusCode, string(body))
	}
}

func createUpload(t *testing.T, uploadURL, uploadLength string) {
	t.Helper()
	createUploadWithUpsert(t, uploadURL, uploadLength, false)
}

func createUploadWithUpsert(t *testing.T, uploadURL, uploadLength string, upsert bool) {
	t.Helper()

	req, err := http.NewRequest(http.MethodPost, uploadURL, nil)
	if err != nil {
		t.Fatalf("create upload request: %v", err)
	}
	req.Header.Set("Tus-Resumable", tusVersion)
	req.Header.Set("Upload-Length", uploadLength)
	if upsert {
		req.Header.Set("x-upsert", "true")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do upload request: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("create upload status = %d, body = %s", resp.StatusCode, string(body))
	}
}

func uploadChunks(t *testing.T, uploadURL string, chunks []string) {
	t.Helper()

	offset := 0
	for _, chunk := range chunks {
		req, err := http.NewRequest(http.MethodPatch, uploadURL, strings.NewReader(chunk))
		if err != nil {
			t.Fatalf("create patch request: %v", err)
		}
		req.Header.Set("Tus-Resumable", tusVersion)
		req.Header.Set("Upload-Offset", strconv.Itoa(offset))
		req.Header.Set("Content-Type", chunkContentType)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("do patch request: %v", err)
		}
		_ = resp.Body.Close()

		if resp.StatusCode != http.StatusNoContent {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("patch status = %d, want %d, body = %s", resp.StatusCode, http.StatusNoContent, string(body))
		}

		offset += len(chunk)
	}
}

func downloadObject(t *testing.T, baseURL, bucketName, objectPath string) string {
	t.Helper()

	downloadPath := fmt.Sprintf("%s/object/%s/%s", baseURL, bucketName, objectPath)
	req, err := http.NewRequest(http.MethodGet, downloadPath, nil)
	if err != nil {
		t.Fatalf("create download request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do download request: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("download status = %d, body = %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	return string(body)
}
