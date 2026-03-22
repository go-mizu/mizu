package storage

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSHA256Hex(t *testing.T) {
	// SHA-256 of empty string
	got := SHA256Hex([]byte(""))
	want := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if got != want {
		t.Errorf("SHA256Hex('') = %s, want %s", got, want)
	}

	// SHA-256 of "hello"
	got = SHA256Hex([]byte("hello"))
	want = "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if got != want {
		t.Errorf("SHA256Hex('hello') = %s, want %s", got, want)
	}

	// Same data always produces the same hash
	data := []byte("test file content for dedup checking")
	h1 := SHA256Hex(data)
	h2 := SHA256Hex(data)
	if h1 != h2 {
		t.Error("SHA256Hex not deterministic")
	}
	if len(h1) != 64 {
		t.Errorf("SHA256Hex length = %d, want 64", len(h1))
	}
}

func TestUploadPresigned_Deduplicated(t *testing.T) {
	// Mock server: initiate returns deduplicated=true immediately
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/files/uploads" {
			body, _ := io.ReadAll(r.Body)
			var req map[string]string
			json.Unmarshal(body, &req)

			if req["content_hash"] == "" {
				t.Error("initiate request missing content_hash")
			}
			if req["path"] == "" {
				t.Error("initiate request missing path")
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"deduplicated": true,
				"path":         req["path"],
				"name":         "test.txt",
				"size":         42,
				"tx":           7,
				"time":         1700000000000,
			})
			return
		}
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		http.Error(w, "not found", 404)
	}))
	defer srv.Close()

	c := &Client{
		Endpoint:   srv.URL,
		Token:      "test-token",
		HTTPClient: srv.Client(),
	}

	data := []byte("hello world")
	result, err := c.UploadPresigned("docs/test.txt", data, "text/plain")
	if err != nil {
		t.Fatalf("UploadPresigned error: %v", err)
	}

	if !result.Deduplicated {
		t.Error("expected Deduplicated=true")
	}
	if result.Path != "docs/test.txt" {
		t.Errorf("Path = %q, want %q", result.Path, "docs/test.txt")
	}
	if result.Size != 42 {
		t.Errorf("Size = %d, want 42", result.Size)
	}
	if result.ContentHash == "" {
		t.Error("ContentHash should not be empty")
	}
}

func TestUploadPresigned_FullFlow(t *testing.T) {
	var (
		initCalled     bool
		putCalled      bool
		completeCalled bool
		putBody        []byte
	)

	// Mock presigned upload target
	presignSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" {
			putCalled = true
			putBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(200)
			return
		}
		http.Error(w, "method not allowed", 405)
	}))
	defer presignSrv.Close()

	// Mock storage API
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == "POST" && r.URL.Path == "/files/uploads":
			initCalled = true
			body, _ := io.ReadAll(r.Body)
			var req map[string]string
			json.Unmarshal(body, &req)

			// Not deduplicated — return presigned URL
			json.NewEncoder(w).Encode(map[string]any{
				"url":          presignSrv.URL + "/upload-target",
				"content_type": req["content_type"],
				"content_hash": req["content_hash"],
				"expires_in":   3600,
			})

		case r.Method == "POST" && r.URL.Path == "/files/uploads/complete":
			completeCalled = true
			body, _ := io.ReadAll(r.Body)
			var req map[string]string
			json.Unmarshal(body, &req)

			json.NewEncoder(w).Encode(map[string]any{
				"path": req["path"],
				"name": "test.txt",
				"size": 13,
				"tx":   1,
				"time": 1700000000000,
			})

		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.Error(w, "not found", 404)
		}
	}))
	defer srv.Close()

	c := &Client{
		Endpoint:   srv.URL,
		Token:      "test-token",
		HTTPClient: &http.Client{},
	}

	data := []byte("hello presign!")
	result, err := c.UploadPresigned("docs/test.txt", data, "text/plain")
	if err != nil {
		t.Fatalf("UploadPresigned error: %v", err)
	}

	if !initCalled {
		t.Error("initiate not called")
	}
	if !putCalled {
		t.Error("presigned PUT not called")
	}
	if !completeCalled {
		t.Error("complete not called")
	}
	if string(putBody) != "hello presign!" {
		t.Errorf("PUT body = %q, want %q", string(putBody), "hello presign!")
	}
	if result.Deduplicated {
		t.Error("expected Deduplicated=false")
	}
	if result.Size != 13 {
		t.Errorf("Size = %d, want 13", result.Size)
	}
	if result.ContentHash == "" {
		t.Error("ContentHash should be set")
	}
}

func TestUploadPresigned_ContentHashConsistency(t *testing.T) {
	// Verify the same file content produces the same hash each time,
	// ensuring dedup works across multiple uploads of the same file.
	content := []byte("identical file content for dedup test")
	hash1 := SHA256Hex(content)
	hash2 := SHA256Hex(content)

	if hash1 != hash2 {
		t.Error("same content produced different hashes")
	}

	// Different content produces different hash
	differentContent := []byte("different file content")
	hash3 := SHA256Hex(differentContent)
	if hash1 == hash3 {
		t.Error("different content produced same hash")
	}
}

func TestDownload_FollowsRedirect(t *testing.T) {
	// Simulate the /files/{path} → 302 → presigned R2 URL flow
	fileContent := "downloaded file content"

	// R2 presigned URL target
	r2Srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(fileContent))
	}))
	defer r2Srv.Close()

	// Storage API: redirects to presigned URL
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/files/") {
			http.Redirect(w, r, r2Srv.URL+"/presigned-blob", 302)
			return
		}
		http.Error(w, "not found", 404)
	}))
	defer srv.Close()

	c := &Client{
		Endpoint:   srv.URL,
		Token:      "test-token",
		HTTPClient: &http.Client{},
	}

	var buf strings.Builder
	err := c.Download("/files/docs/test.txt", &buf)
	if err != nil {
		t.Fatalf("Download error: %v", err)
	}

	if buf.String() != fileContent {
		t.Errorf("downloaded content = %q, want %q", buf.String(), fileContent)
	}
}

func TestDownload_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(404)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "not_found",
			"message": "File not found",
		})
	}))
	defer srv.Close()

	c := &Client{
		Endpoint:   srv.URL,
		Token:      "test-token",
		HTTPClient: &http.Client{},
	}

	var buf strings.Builder
	err := c.Download("/files/nonexistent.txt", &buf)
	if err == nil {
		t.Fatal("expected error for 404")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
}

func TestUploadPresigned_AuthHeader(t *testing.T) {
	var gotAuth string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/files/uploads" {
			gotAuth = r.Header.Get("Authorization")
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"deduplicated": true,
				"path":         "test.txt",
				"name":         "test.txt",
				"size":         5,
				"tx":           1,
				"time":         1700000000000,
			})
			return
		}
		http.Error(w, "not found", 404)
	}))
	defer srv.Close()

	c := &Client{
		Endpoint:   srv.URL,
		Token:      "sk_mytoken123",
		HTTPClient: srv.Client(),
	}

	_, err := c.UploadPresigned("test.txt", []byte("hello"), "text/plain")
	if err != nil {
		t.Fatalf("UploadPresigned error: %v", err)
	}

	if gotAuth != "Bearer sk_mytoken123" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer sk_mytoken123")
	}
}

func TestUploadPresigned_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "internal server error",
		})
	}))
	defer srv.Close()

	c := &Client{
		Endpoint:   srv.URL,
		Token:      "test-token",
		HTTPClient: srv.Client(),
	}

	_, err := c.UploadPresigned("test.txt", []byte("data"), "text/plain")
	if err == nil {
		t.Fatal("expected error for 500")
	}
}

func TestUploadPresigned_PresignedPutFailure(t *testing.T) {
	// Presigned target returns error
	presignSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
	}))
	defer presignSrv.Close()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/files/uploads" {
			json.NewEncoder(w).Encode(map[string]any{
				"url":          presignSrv.URL + "/upload",
				"content_type": "text/plain",
				"expires_in":   3600,
			})
			return
		}
		http.Error(w, "not found", 404)
	}))
	defer srv.Close()

	c := &Client{
		Endpoint:   srv.URL,
		Token:      "test-token",
		HTTPClient: &http.Client{},
	}

	_, err := c.UploadPresigned("test.txt", []byte("data"), "text/plain")
	if err == nil {
		t.Fatal("expected error when presigned PUT fails")
	}
	if !strings.Contains(err.Error(), "upload failed") {
		t.Errorf("error = %q, expected 'upload failed'", err.Error())
	}
}
