// File: lib/storage/transport/webdav/server_test.go

package webdav

import (
	"bytes"
	"context"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/localbase/pkg/storage/driver/local"
)

func newTestServer(t *testing.T, cfg *Config) (*httptest.Server, func()) {
	t.Helper()

	ctx := context.Background()
	store, err := local.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("open local storage: %v", err)
	}

	if cfg == nil {
		cfg = &Config{}
	}

	server := New(store, cfg)
	srv := httptest.NewServer(server)

	cleanup := func() {
		srv.Close()
		_ = store.Close()
	}

	return srv, cleanup
}

func TestServer_OPTIONS(t *testing.T) {
	srv, cleanup := newTestServer(t, nil)
	defer cleanup()

	req, err := http.NewRequest("OPTIONS", srv.URL+"/", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Check DAV header - indicates WebDAV capability
	dav := resp.Header.Get("DAV")
	if dav == "" {
		t.Error("missing DAV header")
	}
	if !strings.Contains(dav, "1") {
		t.Errorf("DAV header %q does not contain '1'", dav)
	}

	// Check Allow header contains WebDAV methods
	// The golang.org/x/net/webdav package may not include GET/PUT/HEAD in Allow header
	// but those are standard HTTP methods that work anyway
	allow := resp.Header.Get("Allow")
	if allow == "" {
		t.Error("missing Allow header")
	}

	// Check for WebDAV-specific methods
	expectedMethods := []string{"DELETE", "PROPFIND", "COPY", "MOVE", "LOCK", "UNLOCK"}
	for _, method := range expectedMethods {
		if !strings.Contains(allow, method) {
			t.Errorf("Allow header %q does not contain %q", allow, method)
		}
	}
}

func TestServer_MKCOL_DeleteBucket(t *testing.T) {
	srv, cleanup := newTestServer(t, nil)
	defer cleanup()

	// Create bucket using MKCOL
	req, _ := http.NewRequest("MKCOL", srv.URL+"/test-bucket", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("MKCOL request: %v", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("MKCOL status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	// Verify bucket exists using PROPFIND
	req, _ = http.NewRequest("PROPFIND", srv.URL+"/test-bucket", nil)
	req.Header.Set("Depth", "0")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PROPFIND request: %v", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusMultiStatus {
		t.Errorf("PROPFIND status = %d, want %d", resp.StatusCode, http.StatusMultiStatus)
	}

	// Delete bucket
	req, _ = http.NewRequest("DELETE", srv.URL+"/test-bucket", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE request: %v", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		t.Errorf("DELETE status = %d, want %d or %d", resp.StatusCode, http.StatusNoContent, http.StatusOK)
	}

	// Verify bucket no longer exists
	req, _ = http.NewRequest("PROPFIND", srv.URL+"/test-bucket", nil)
	req.Header.Set("Depth", "0")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PROPFIND request: %v", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("PROPFIND after delete status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestServer_PUT_GET_DELETE(t *testing.T) {
	srv, cleanup := newTestServer(t, nil)
	defer cleanup()

	// Create bucket first
	req, _ := http.NewRequest("MKCOL", srv.URL+"/files", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("MKCOL request: %v", err)
	}
	_ = resp.Body.Close()

	content := "Hello, WebDAV!"

	// Upload file using PUT
	req, _ = http.NewRequest("PUT", srv.URL+"/files/test.txt", strings.NewReader(content))
	req.Header.Set("Content-Type", "text/plain")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PUT request: %v", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		t.Errorf("PUT status = %d, want 201, 204, or 200", resp.StatusCode)
	}

	// Download file using GET
	req, _ = http.NewRequest("GET", srv.URL+"/files/test.txt", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET request: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if string(body) != content {
		t.Errorf("GET body = %q, want %q", string(body), content)
	}

	// Delete file
	req, _ = http.NewRequest("DELETE", srv.URL+"/files/test.txt", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE request: %v", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		t.Errorf("DELETE status = %d, want 204 or 200", resp.StatusCode)
	}

	// Verify file no longer exists
	req, _ = http.NewRequest("GET", srv.URL+"/files/test.txt", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET request: %v", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("GET after delete status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestServer_PROPFIND_Root(t *testing.T) {
	srv, cleanup := newTestServer(t, nil)
	defer cleanup()

	// Create some buckets
	for _, name := range []string{"bucket1", "bucket2"} {
		req, _ := http.NewRequest("MKCOL", srv.URL+"/"+name, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("MKCOL request: %v", err)
		}
		_ = resp.Body.Close()
	}

	// PROPFIND on root
	req, _ := http.NewRequest("PROPFIND", srv.URL+"/", nil)
	req.Header.Set("Depth", "1")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PROPFIND request: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusMultiStatus {
		t.Errorf("PROPFIND status = %d, want %d", resp.StatusCode, http.StatusMultiStatus)
	}

	// Check response contains both buckets
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "bucket1") {
		t.Error("PROPFIND response does not contain bucket1")
	}
	if !strings.Contains(bodyStr, "bucket2") {
		t.Error("PROPFIND response does not contain bucket2")
	}
}

func TestServer_PROPFIND_File(t *testing.T) {
	srv, cleanup := newTestServer(t, nil)
	defer cleanup()

	// Create bucket and file
	req, _ := http.NewRequest("MKCOL", srv.URL+"/data", nil)
	resp, _ := http.DefaultClient.Do(req)
	_ = resp.Body.Close()

	content := "test content for propfind"
	req, _ = http.NewRequest("PUT", srv.URL+"/data/file.txt", strings.NewReader(content))
	resp, _ = http.DefaultClient.Do(req)
	_ = resp.Body.Close()

	// PROPFIND on file
	propfindBody := `<?xml version="1.0" encoding="utf-8"?>
<propfind xmlns="DAV:">
  <prop>
    <getcontentlength/>
    <getlastmodified/>
    <resourcetype/>
  </prop>
</propfind>`

	req, _ = http.NewRequest("PROPFIND", srv.URL+"/data/file.txt", strings.NewReader(propfindBody))
	req.Header.Set("Depth", "0")
	req.Header.Set("Content-Type", "application/xml")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PROPFIND request: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusMultiStatus {
		t.Errorf("PROPFIND status = %d, want %d, body: %s", resp.StatusCode, http.StatusMultiStatus, string(body))
	}

	bodyStr := string(body)
	if !strings.Contains(bodyStr, "getcontentlength") {
		t.Error("PROPFIND response does not contain getcontentlength")
	}
}

func TestServer_COPY(t *testing.T) {
	srv, cleanup := newTestServer(t, nil)
	defer cleanup()

	// Create bucket and source file
	req, _ := http.NewRequest("MKCOL", srv.URL+"/source", nil)
	resp, _ := http.DefaultClient.Do(req)
	_ = resp.Body.Close()

	content := "copy me"
	req, _ = http.NewRequest("PUT", srv.URL+"/source/original.txt", strings.NewReader(content))
	resp, _ = http.DefaultClient.Do(req)
	_ = resp.Body.Close()

	// COPY to new location in same bucket
	req, _ = http.NewRequest("COPY", srv.URL+"/source/original.txt", nil)
	req.Header.Set("Destination", srv.URL+"/source/copied.txt")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("COPY request: %v", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		t.Errorf("COPY status = %d, want 201 or 204", resp.StatusCode)
	}

	// Verify copied file exists
	req, _ = http.NewRequest("GET", srv.URL+"/source/copied.txt", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET request: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET copied file status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if string(body) != content {
		t.Errorf("copied content = %q, want %q", string(body), content)
	}

	// Verify original still exists
	req, _ = http.NewRequest("GET", srv.URL+"/source/original.txt", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET request: %v", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET original file status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestServer_MOVE(t *testing.T) {
	srv, cleanup := newTestServer(t, nil)
	defer cleanup()

	// Create bucket and source file
	req, _ := http.NewRequest("MKCOL", srv.URL+"/movebucket", nil)
	resp, _ := http.DefaultClient.Do(req)
	_ = resp.Body.Close()

	content := "move me"
	req, _ = http.NewRequest("PUT", srv.URL+"/movebucket/source.txt", strings.NewReader(content))
	resp, _ = http.DefaultClient.Do(req)
	_ = resp.Body.Close()

	// MOVE to new location
	req, _ = http.NewRequest("MOVE", srv.URL+"/movebucket/source.txt", nil)
	req.Header.Set("Destination", srv.URL+"/movebucket/dest.txt")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("MOVE request: %v", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		t.Errorf("MOVE status = %d, want 201 or 204", resp.StatusCode)
	}

	// Verify destination file exists
	req, _ = http.NewRequest("GET", srv.URL+"/movebucket/dest.txt", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET request: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET moved file status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if string(body) != content {
		t.Errorf("moved content = %q, want %q", string(body), content)
	}

	// Verify source no longer exists
	req, _ = http.NewRequest("GET", srv.URL+"/movebucket/source.txt", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET request: %v", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("GET source after move status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestServer_ReadOnly(t *testing.T) {
	srv, cleanup := newTestServer(t, &Config{ReadOnly: true})
	defer cleanup()

	// Try to create bucket - should fail
	req, _ := http.NewRequest("MKCOL", srv.URL+"/test", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("MKCOL request: %v", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("MKCOL in read-only status = %d, want %d", resp.StatusCode, http.StatusForbidden)
	}

	// Try to PUT file - should fail
	req, _ = http.NewRequest("PUT", srv.URL+"/test/file.txt", strings.NewReader("test"))
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PUT request: %v", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("PUT in read-only status = %d, want %d", resp.StatusCode, http.StatusForbidden)
	}
}

func TestServer_BasicAuth(t *testing.T) {
	srv, cleanup := newTestServer(t, &Config{
		Auth: AuthConfig{
			Type:  "basic",
			Realm: "Test",
			BasicAuth: func(username, password string) bool {
				return username == "admin" && password == "secret"
			},
		},
	})
	defer cleanup()

	// Request without auth - should fail
	req, _ := http.NewRequest("PROPFIND", srv.URL+"/", nil)
	req.Header.Set("Depth", "0")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PROPFIND request: %v", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("unauthenticated status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
	if resp.Header.Get("WWW-Authenticate") == "" {
		t.Error("missing WWW-Authenticate header")
	}

	// Request with wrong auth - should fail
	req, _ = http.NewRequest("PROPFIND", srv.URL+"/", nil)
	req.Header.Set("Depth", "0")
	req.SetBasicAuth("admin", "wrongpassword")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PROPFIND request: %v", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("wrong auth status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}

	// Request with correct auth - should succeed
	req, _ = http.NewRequest("PROPFIND", srv.URL+"/", nil)
	req.Header.Set("Depth", "0")
	req.SetBasicAuth("admin", "secret")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PROPFIND request: %v", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusMultiStatus {
		t.Errorf("authenticated status = %d, want %d", resp.StatusCode, http.StatusMultiStatus)
	}
}

func TestServer_JWTAuth(t *testing.T) {
	jwtSecret := "test-secret-key"

	srv, cleanup := newTestServer(t, &Config{
		Auth: AuthConfig{
			Type:      "jwt",
			JWTSecret: jwtSecret,
		},
	})
	defer cleanup()

	// Request without auth - should fail
	req, _ := http.NewRequest("PROPFIND", srv.URL+"/", nil)
	req.Header.Set("Depth", "0")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PROPFIND request: %v", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("unauthenticated status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}

	// Create valid token
	claims := map[string]any{
		"sub":  "user123",
		"name": "Test User",
		"exp":  float64(time.Now().Add(time.Hour).Unix()),
	}
	token, err := createToken(claims, jwtSecret)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	// Request with valid token - should succeed
	req, _ = http.NewRequest("PROPFIND", srv.URL+"/", nil)
	req.Header.Set("Depth", "0")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PROPFIND request: %v", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusMultiStatus {
		t.Errorf("authenticated status = %d, want %d", resp.StatusCode, http.StatusMultiStatus)
	}

	// Create expired token
	expiredClaims := map[string]any{
		"sub":  "user123",
		"name": "Test User",
		"exp":  float64(time.Now().Add(-time.Hour).Unix()),
	}
	expiredToken, _ := createToken(expiredClaims, jwtSecret)

	// Request with expired token - should fail
	req, _ = http.NewRequest("PROPFIND", srv.URL+"/", nil)
	req.Header.Set("Depth", "0")
	req.Header.Set("Authorization", "Bearer "+expiredToken)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PROPFIND request: %v", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expired token status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestServer_SingleBucketMode(t *testing.T) {
	ctx := context.Background()
	store, err := local.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("open local storage: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create bucket first
	_, err = store.CreateBucket(ctx, "mybucket", nil)
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}

	cfg := &Config{
		Bucket: "mybucket",
	}

	server := New(store, cfg)
	srv := httptest.NewServer(server)
	defer srv.Close()

	// PUT directly to root (maps to mybucket)
	content := "single bucket content"
	req, _ := http.NewRequest("PUT", srv.URL+"/file.txt", strings.NewReader(content))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PUT request: %v", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		t.Errorf("PUT status = %d, want 201, 204, or 200", resp.StatusCode)
	}

	// GET from root
	req, _ = http.NewRequest("GET", srv.URL+"/file.txt", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET request: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if string(body) != content {
		t.Errorf("GET body = %q, want %q", string(body), content)
	}
}

func TestServer_NestedDirectories(t *testing.T) {
	srv, cleanup := newTestServer(t, nil)
	defer cleanup()

	// Create bucket
	req, _ := http.NewRequest("MKCOL", srv.URL+"/nested", nil)
	resp, _ := http.DefaultClient.Do(req)
	_ = resp.Body.Close()

	// Create nested file
	content := "nested content"
	req, _ = http.NewRequest("PUT", srv.URL+"/nested/a/b/c/file.txt", strings.NewReader(content))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PUT request: %v", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		t.Errorf("PUT nested status = %d, want 201, 204, or 200", resp.StatusCode)
	}

	// PROPFIND on intermediate directory
	req, _ = http.NewRequest("PROPFIND", srv.URL+"/nested/a/b", nil)
	req.Header.Set("Depth", "1")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PROPFIND request: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusMultiStatus {
		t.Errorf("PROPFIND nested status = %d, want %d, body: %s", resp.StatusCode, http.StatusMultiStatus, string(body))
	}

	// Verify "c" directory is listed
	if !strings.Contains(string(body), "c") {
		t.Errorf("PROPFIND response does not contain 'c' directory: %s", string(body))
	}

	// GET nested file
	req, _ = http.NewRequest("GET", srv.URL+"/nested/a/b/c/file.txt", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET request: %v", err)
	}
	body, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET nested status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if string(body) != content {
		t.Errorf("GET nested body = %q, want %q", string(body), content)
	}
}

func TestServer_LargeFile(t *testing.T) {
	srv, cleanup := newTestServer(t, &Config{
		WriteBufferSize: 1024, // Small buffer to test temp file spilling
	})
	defer cleanup()

	// Create bucket
	req, _ := http.NewRequest("MKCOL", srv.URL+"/large", nil)
	resp, _ := http.DefaultClient.Do(req)
	_ = resp.Body.Close()

	// Create large content (larger than buffer)
	content := bytes.Repeat([]byte("X"), 2048)

	req, _ = http.NewRequest("PUT", srv.URL+"/large/bigfile.bin", bytes.NewReader(content))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PUT request: %v", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		t.Errorf("PUT large status = %d, want 201, 204, or 200", resp.StatusCode)
	}

	// GET and verify
	req, _ = http.NewRequest("GET", srv.URL+"/large/bigfile.bin", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET request: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET large status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if !bytes.Equal(body, content) {
		t.Errorf("GET large content mismatch: got %d bytes, want %d bytes", len(body), len(content))
	}
}

func TestServer_HideDotFiles(t *testing.T) {
	srv, cleanup := newTestServer(t, &Config{
		HideDotFiles: true,
	})
	defer cleanup()

	// Create bucket
	req, _ := http.NewRequest("MKCOL", srv.URL+"/dotfiles", nil)
	resp, _ := http.DefaultClient.Do(req)
	_ = resp.Body.Close()

	// Create visible and hidden files
	req, _ = http.NewRequest("PUT", srv.URL+"/dotfiles/visible.txt", strings.NewReader("visible"))
	resp, _ = http.DefaultClient.Do(req)
	_ = resp.Body.Close()

	req, _ = http.NewRequest("PUT", srv.URL+"/dotfiles/.hidden", strings.NewReader("hidden"))
	resp, _ = http.DefaultClient.Do(req)
	_ = resp.Body.Close()

	// PROPFIND should not show hidden file
	req, _ = http.NewRequest("PROPFIND", srv.URL+"/dotfiles", nil)
	req.Header.Set("Depth", "1")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PROPFIND request: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	bodyStr := string(body)
	if !strings.Contains(bodyStr, "visible.txt") {
		t.Error("PROPFIND should show visible.txt")
	}
	if strings.Contains(bodyStr, ".hidden") {
		t.Error("PROPFIND should not show .hidden when HideDotFiles is true")
	}

	// Hidden file should still be accessible directly
	req, _ = http.NewRequest("GET", srv.URL+"/dotfiles/.hidden", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET request: %v", err)
	}
	body, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET .hidden status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if string(body) != "hidden" {
		t.Errorf("GET .hidden body = %q, want %q", string(body), "hidden")
	}
}

// Multistat represents a WebDAV multi-status response
type Multistat struct {
	XMLName   xml.Name   `xml:"multistatus"`
	Responses []Response `xml:"response"`
}

type Response struct {
	Href     string    `xml:"href"`
	Propstat *Propstat `xml:"propstat"`
}

type Propstat struct {
	Prop   Prop   `xml:"prop"`
	Status string `xml:"status"`
}

type Prop struct {
	DisplayName      string `xml:"displayname"`
	ResourceType     string `xml:"resourcetype"`
	GetContentLength int64  `xml:"getcontentlength"`
}
