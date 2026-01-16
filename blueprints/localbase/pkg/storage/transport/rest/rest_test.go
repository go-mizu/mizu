package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/blueprints/localbase/pkg/storage/driver/local"
	"github.com/go-mizu/mizu"
)

func newTestServer(t *testing.T) (*httptest.Server, func()) {
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
	}

	return srv, cleanup
}

func TestBucketLifecycle(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	base := srv.URL + "/storage/v1"

	// Create bucket
	createStatus, createBody := doJSONRequest(t, http.MethodPost, base+"/bucket", map[string]any{"name": "test"})
	if createStatus != http.StatusOK {
		t.Fatalf("create bucket status = %d, body = %s", createStatus, string(createBody))
	}

	var createResp CreateBucketResponse
	if err := json.Unmarshal(createBody, &createResp); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if createResp.Name != "test" {
		t.Fatalf("unexpected bucket name %q", createResp.Name)
	}

	// Duplicate bucket should return conflict error payload
	dupStatus, dupBody := doJSONRequest(t, http.MethodPost, base+"/bucket", map[string]any{"name": "test"})
	if dupStatus != http.StatusConflict {
		t.Fatalf("duplicate status = %d, want %d", dupStatus, http.StatusConflict)
	}
	var dupErr errorPayload
	if err := json.Unmarshal(dupBody, &dupErr); err != nil {
		t.Fatalf("decode duplicate error: %v", err)
	}
	if dupErr.StatusCode != http.StatusConflict || dupErr.Error != "Conflict" {
		t.Fatalf("unexpected error payload %+v", dupErr)
	}

	// List buckets should include the created bucket
	listStatus, listBody := doRequest(t, http.MethodGet, base+"/bucket", nil, nil)
	if listStatus != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", listStatus, string(listBody))
	}
	var buckets []BucketResponse
	if err := json.Unmarshal(listBody, &buckets); err != nil {
		t.Fatalf("decode bucket list: %v", err)
	}
	if len(buckets) != 1 || buckets[0].Name != "test" {
		t.Fatalf("unexpected buckets %+v", buckets)
	}

	// Get bucket details
	getStatus, getBody := doRequest(t, http.MethodGet, base+"/bucket/test", nil, nil)
	if getStatus != http.StatusOK {
		t.Fatalf("get bucket status = %d, body = %s", getStatus, string(getBody))
	}
	var getResp BucketResponse
	if err := json.Unmarshal(getBody, &getResp); err != nil {
		t.Fatalf("decode bucket response: %v", err)
	}
	if getResp.Name != "test" || getResp.ID != "test" {
		t.Fatalf("unexpected bucket response %+v", getResp)
	}
}

func TestObjectCRUD(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	base := srv.URL + "/storage/v1"

	// Prepare bucket
	if status, body := doJSONRequest(t, http.MethodPost, base+"/bucket", map[string]any{"name": "files"}); status != http.StatusOK {
		t.Fatalf("create bucket status = %d, body = %s", status, string(body))
	}

	objectPath := "/object/files/folder/file.txt"
	content := "hello world"

	// Upload object
	headers := map[string]string{"Content-Type": "text/plain"}
	uploadStatus, uploadBody := doRequest(t, http.MethodPost, base+objectPath, strings.NewReader(content), headers)
	if uploadStatus != http.StatusOK {
		t.Fatalf("upload status = %d, body = %s", uploadStatus, string(uploadBody))
	}

	var uploadResp UploadResponse
	if err := json.Unmarshal(uploadBody, &uploadResp); err != nil {
		t.Fatalf("decode upload response: %v", err)
	}
	if uploadResp.Key != path.Join("files", "folder/file.txt") {
		t.Fatalf("unexpected upload key %q", uploadResp.Key)
	}

	// Download the object
	downloadStatus, downloadBody := doRequest(t, http.MethodGet, base+objectPath, nil, nil)
	if downloadStatus != http.StatusOK {
		t.Fatalf("download status = %d, body = %s", downloadStatus, string(downloadBody))
	}
	if string(downloadBody) != content {
		t.Fatalf("downloaded content = %q, want %q", string(downloadBody), content)
	}

	// Partial content using Range header
	rangeHeaders := map[string]string{"Range": "bytes=0-4"}
	rangeStatus, rangeBody, rangeResp := doRequestWithResponse(t, http.MethodGet, base+objectPath, nil, rangeHeaders)
	if rangeStatus != http.StatusPartialContent {
		t.Fatalf("range status = %d", rangeStatus)
	}
	if string(rangeBody) != "hello" {
		t.Fatalf("range body = %q", string(rangeBody))
	}
	if got := rangeResp.Header.Get("Content-Range"); got != "bytes 0-4/11" {
		t.Fatalf("content-range header = %q", got)
	}
	if got := rangeResp.Header.Get("Accept-Ranges"); got != "bytes" {
		t.Fatalf("accept-ranges header = %q", got)
	}

	// List objects
	listReq := ListObjectsRequest{Prefix: "folder"}
	listStatus, listBody := doJSONRequest(t, http.MethodPost, base+"/object/list/files", listReq)
	if listStatus != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", listStatus, string(listBody))
	}
	var objects []ObjectInfo
	if err := json.Unmarshal(listBody, &objects); err != nil {
		t.Fatalf("decode objects: %v", err)
	}
	if len(objects) != 1 || objects[0].Name != "file.txt" {
		t.Fatalf("unexpected objects %+v", objects)
	}

	// Signed URL should be unsupported by local driver
	signStatus, signBody := doJSONRequest(t, http.MethodPost, base+"/object/sign/files/folder/file.txt", SignURLRequest{ExpiresIn: 60})
	if signStatus != http.StatusNotImplemented {
		t.Fatalf("sign status = %d, body = %s", signStatus, string(signBody))
	}
	var signErr errorPayload
	if err := json.Unmarshal(signBody, &signErr); err != nil {
		t.Fatalf("decode sign error: %v", err)
	}
	if signErr.StatusCode != http.StatusNotImplemented || signErr.Error != "Not Implemented" {
		t.Fatalf("unexpected sign error %+v", signErr)
	}
}

func doJSONRequest(t *testing.T, method, url string, payload any) (int, []byte) {
	t.Helper()

	buf := &bytes.Buffer{}
	if payload != nil {
		if err := json.NewEncoder(buf).Encode(payload); err != nil {
			t.Fatalf("encode payload: %v", err)
		}
	}
	headers := map[string]string{"Content-Type": "application/json"}
	return doRequest(t, method, url, buf, headers)
}

func doRequest(t *testing.T, method, url string, body io.Reader, headers map[string]string) (int, []byte) {
	status, data, _ := doRequestWithResponse(t, method, url, body, headers)
	return status, data
}

func doRequestWithResponse(t *testing.T, method, url string, body io.Reader, headers map[string]string) (int, []byte, *http.Response) {
	t.Helper()

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return resp.StatusCode, data, resp
}

// Test secret for signed URLs
const testSigningSecret = "test-signing-secret-at-least-32-characters"

// newTestServerWithAuth creates a test server with JWT authentication enabled.
func newTestServerWithAuth(t *testing.T) (*httptest.Server, string, func()) {
	t.Helper()

	ctx := context.Background()
	store, err := local.Open(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("open local storage: %v", err)
	}

	app := mizu.New()
	authConfig := AuthConfig{
		JWTSecret:            testSigningSecret,
		AllowAnonymousPublic: true,
	}
	RegisterWithAuth(app, "/storage/v1", store, authConfig)

	srv := httptest.NewServer(app)

	// Create a valid token for testing
	token, err := createToken(&Claims{
		Sub:  "test-user",
		Role: "service_role",
		Exp:  9999999999,
	}, testSigningSecret)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	cleanup := func() {
		srv.Close()
		_ = store.Close()
	}

	return srv, token, cleanup
}

// TestSignedURLsWithAuth tests signed URL creation and access with authentication.
func TestSignedURLsWithAuth(t *testing.T) {
	srv, token, cleanup := newTestServerWithAuth(t)
	defer cleanup()

	base := srv.URL + "/storage/v1"

	// Helper to make authenticated requests
	authHeaders := map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "application/json",
	}
	authUploadHeaders := map[string]string{
		"Authorization": "Bearer " + token,
		"Content-Type":  "text/plain",
	}

	// Create bucket and object
	bucketReq, _ := json.Marshal(map[string]any{"name": "signed-test"})
	req, _ := http.NewRequest(http.MethodPost, base+"/bucket", bytes.NewReader(bucketReq))
	for k, v := range authHeaders {
		req.Header.Set(k, v)
	}
	resp, _ := http.DefaultClient.Do(req)
	_ = resp.Body.Close()

	// Upload test file
	uploadReq, _ := http.NewRequest(http.MethodPost, base+"/object/signed-test/hello.txt", strings.NewReader("Hello, World!"))
	for k, v := range authUploadHeaders {
		uploadReq.Header.Set(k, v)
	}
	uploadResp, _ := http.DefaultClient.Do(uploadReq)
	_ = uploadResp.Body.Close()

	t.Run("create signed URL for existing object", func(t *testing.T) {
		signReq := SignURLRequest{ExpiresIn: 3600}
		signBody, _ := json.Marshal(signReq)
		req, _ := http.NewRequest(http.MethodPost, base+"/object/sign/signed-test/hello.txt", bytes.NewReader(signBody))
		for k, v := range authHeaders {
			req.Header.Set(k, v)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("do request: %v", err)
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("status = %d, body = %s", resp.StatusCode, string(body))
		}

		var signResp SignURLResponse
		if err := json.NewDecoder(resp.Body).Decode(&signResp); err != nil {
			t.Fatalf("decode response: %v", err)
		}

		if signResp.SignedURL == "" {
			t.Fatal("signedURL is empty")
		}

		// Verify URL contains render endpoint and token
		if !strings.Contains(signResp.SignedURL, "/object/render/") {
			t.Errorf("signed URL should contain /object/render/, got %s", signResp.SignedURL)
		}
		if !strings.Contains(signResp.SignedURL, "token=") {
			t.Errorf("signed URL should contain token parameter, got %s", signResp.SignedURL)
		}
	})

	t.Run("access object via signed URL", func(t *testing.T) {
		// First, get a signed URL
		signReq := SignURLRequest{ExpiresIn: 3600}
		signBody, _ := json.Marshal(signReq)
		req, _ := http.NewRequest(http.MethodPost, base+"/object/sign/signed-test/hello.txt", bytes.NewReader(signBody))
		for k, v := range authHeaders {
			req.Header.Set(k, v)
		}

		resp, _ := http.DefaultClient.Do(req)
		var signResp SignURLResponse
		_ = json.NewDecoder(resp.Body).Decode(&signResp)
		_ = resp.Body.Close()

		// Access the object via signed URL (no auth required)
		getReq, _ := http.NewRequest(http.MethodGet, signResp.SignedURL, nil)
		getResp, err := http.DefaultClient.Do(getReq)
		if err != nil {
			t.Fatalf("do request: %v", err)
		}
		defer func() {
			_ = getResp.Body.Close()
		}()

		if getResp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(getResp.Body)
			t.Fatalf("status = %d, body = %s", getResp.StatusCode, string(body))
		}

		body, _ := io.ReadAll(getResp.Body)
		if string(body) != "Hello, World!" {
			t.Errorf("body = %q, want %q", string(body), "Hello, World!")
		}
	})

	t.Run("signed URL for multiple objects", func(t *testing.T) {
		// Upload another file
		uploadReq, _ := http.NewRequest(http.MethodPost, base+"/object/signed-test/world.txt", strings.NewReader("World!"))
		for k, v := range authUploadHeaders {
			uploadReq.Header.Set(k, v)
		}
		uploadResp, _ := http.DefaultClient.Do(uploadReq)
		_ = uploadResp.Body.Close()

		signReq := SignURLsRequest{
			ExpiresIn: 3600,
			Paths:     []string{"hello.txt", "world.txt"},
		}
		signBody, _ := json.Marshal(signReq)
		req, _ := http.NewRequest(http.MethodPost, base+"/object/sign/signed-test", bytes.NewReader(signBody))
		for k, v := range authHeaders {
			req.Header.Set(k, v)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("do request: %v", err)
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("status = %d, body = %s", resp.StatusCode, string(body))
		}

		var results []SignURLsResponseItem
		if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
			t.Fatalf("decode response: %v", err)
		}

		if len(results) != 2 {
			t.Fatalf("got %d results, want 2", len(results))
		}

		for _, r := range results {
			if r.SignedURL == "" {
				t.Errorf("result for %s has empty signedURL", r.Path)
			}
		}
	})

	t.Run("create upload signed URL", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, base+"/object/upload/sign/signed-test/upload-target.txt", nil)
		for k, v := range authHeaders {
			req.Header.Set(k, v)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("do request: %v", err)
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("status = %d, body = %s", resp.StatusCode, string(body))
		}

		var uploadSignResp UploadSignedURLResponse
		if err := json.NewDecoder(resp.Body).Decode(&uploadSignResp); err != nil {
			t.Fatalf("decode response: %v", err)
		}

		if uploadSignResp.URL == "" {
			t.Fatal("upload URL is empty")
		}
		if uploadSignResp.Token == "" {
			t.Fatal("upload token is empty")
		}
	})

	t.Run("invalid token rejected", func(t *testing.T) {
		// Try to access render endpoint with invalid token
		req, _ := http.NewRequest(http.MethodGet, base+"/object/render/signed-test/hello.txt?token=invalid-token", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("do request: %v", err)
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if resp.StatusCode != http.StatusUnauthorized {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("status = %d, want 401, body = %s", resp.StatusCode, string(body))
		}
	})

	t.Run("missing token rejected", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, base+"/object/render/signed-test/hello.txt", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("do request: %v", err)
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if resp.StatusCode != http.StatusUnauthorized {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("status = %d, want 401, body = %s", resp.StatusCode, string(body))
		}
	})
}
