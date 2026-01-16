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

	"github.com/go-mizu/blueprints/localbase/pkg/storage/driver/local"
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
