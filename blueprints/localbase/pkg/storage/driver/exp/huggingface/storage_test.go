package huggingface

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/localbase/pkg/storage"
)

// simpleReader implements io.Reader for testing
type simpleReader struct{}

func (simpleReader) Read(p []byte) (int, error) {
	return 0, io.EOF
}

func TestDriverOpenParsing(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name    string
		dsn     string
		wantErr string
	}{
		{"invalid_url", "http://%zz", "invalid dsn"},
		{"missing_scheme", "repo-only", "missing scheme"},
		{"missing_repo", "huggingface://", "missing repo"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := &driver{}
			_, err := d.Open(ctx, tc.dsn)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q got %v", tc.wantErr, err)
			}
		})
	}
}

func TestDriverOpenSuccessAndBucketClone(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r) // not used here
	}))
	t.Cleanup(server.Close)

	d := &driver{}
	dsn := "huggingface://hf_token@org/repo?repo_type=dataset&revision=dev&base_url=" + url.QueryEscape(server.URL) + "&timeout=2"
	stAny, err := d.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	st, ok := stAny.(*hfStorage)
	if !ok {
		t.Fatalf("unexpected storage type %T", stAny)
	}

	if st.repoID != "org/repo" || st.repoType != "dataset" || st.revision != "dev" {
		t.Fatalf("unexpected parsed values: %+v", st)
	}
	if st.baseURL != server.URL {
		t.Fatalf("baseURL not trimmed correctly: %q", st.baseURL)
	}
	if st.token != "hf_token" {
		t.Fatalf("token not parsed from userinfo: %q", st.token)
	}
	if st.client.Timeout != 2*time.Second {
		t.Fatalf("timeout not applied: %v", st.client.Timeout)
	}

	// Bucket with different name should clone storage with new repoID
	b := st.Bucket("other").(*hfBucket)
	if b.st.repoID != "other" {
		t.Fatalf("bucket did not clone repo id: %q", b.st.repoID)
	}

	// Bucket with same name keeps original repo id
	b2 := st.Bucket(st.repoID).(*hfBucket)
	if b2.st.repoID != st.repoID {
		t.Fatalf("bucket changed repo unexpectedly")
	}
}

func TestStorageUnsupportedOperationsAndFeatures(t *testing.T) {
	st := &hfStorage{}

	if _, err := st.Buckets(context.Background(), 0, 0, nil); !errors.Is(err, storage.ErrUnsupported) {
		t.Fatalf("Buckets should return ErrUnsupported")
	}
	if _, err := st.CreateBucket(context.Background(), "x", nil); !errors.Is(err, storage.ErrUnsupported) {
		t.Fatalf("CreateBucket should return ErrUnsupported")
	}
	if err := st.DeleteBucket(context.Background(), "x", nil); !errors.Is(err, storage.ErrUnsupported) {
		t.Fatalf("DeleteBucket should return ErrUnsupported")
	}

	features := st.Features()
	if !features["directories"] || !features["public_url"] {
		t.Fatalf("unexpected features: %+v", features)
	}
	if err := st.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
}

func TestBucketInfoAndUnsupported(t *testing.T) {
	st := &hfStorage{repoID: "org/repo", repoType: "model", revision: "main"}
	b := &hfBucket{st: st}

	info, err := b.Info(context.Background())
	if err != nil {
		t.Fatalf("Info error: %v", err)
	}
	if info.Name != "org/repo" || info.Metadata["repo_type"] != "model" {
		t.Fatalf("unexpected info: %+v", info)
	}

	if _, err := b.Write(context.Background(), "file", simpleReader{}, 1, "text/plain", nil); !errors.Is(err, storage.ErrUnsupported) {
		t.Fatalf("Write expected ErrUnsupported")
	}
	if err := b.Delete(context.Background(), "file", nil); !errors.Is(err, storage.ErrUnsupported) {
		t.Fatalf("Delete expected ErrUnsupported")
	}
	if _, err := b.Copy(context.Background(), "dst", "src", "key", nil); !errors.Is(err, storage.ErrUnsupported) {
		t.Fatalf("Copy expected ErrUnsupported")
	}
	if _, err := b.Move(context.Background(), "dst", "src", "key", nil); !errors.Is(err, storage.ErrUnsupported) {
		t.Fatalf("Move expected ErrUnsupported")
	}
}

func TestBucketOpenAndStatStatuses(t *testing.T) {
	ctx := context.Background()

	handler := func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok/resolve/main/file.txt":
			if r.Header.Get("Authorization") != "Bearer secret" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			if rng := r.Header.Get("Range"); rng != "bytes=5-9" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Length", "5")
			w.Header().Set("Content-Type", "text/plain")
			w.Header().Set("ETag", "\"etag\"")
			w.WriteHeader(http.StatusPartialContent)
			_, _ = w.Write([]byte("hello"))
		case "/ok/resolve/main/missing", "/stat/resolve/main/missing":
			w.WriteHeader(http.StatusNotFound)
		case "/ok/resolve/main/forbidden", "/stat/resolve/main/forbidden":
			w.WriteHeader(http.StatusForbidden)
		case "/ok/resolve/main/file", "/stat/resolve/main/file":
			w.Header().Set("Content-Length", "10")
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("ETag", "etag2")
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusTeapot)
		}
	}

	server := httptest.NewServer(http.HandlerFunc(handler))
	t.Cleanup(server.Close)

	st := &hfStorage{client: server.Client(), baseURL: server.URL, repoID: "ok", repoType: "model", revision: "main", token: "secret"}
	b := &hfBucket{st: st}

	// Open success with range
	rc, obj, err := b.Open(ctx, "/file.txt", 5, 5, nil)
	if err != nil {
		t.Fatalf("Open error: %v", err)
	}
	defer func() {
		_ = rc.Close()
	}()
	data, _ := io.ReadAll(rc)
	if string(data) != "hello" {
		t.Fatalf("unexpected data %q", data)
	}
	if obj.Size != 5 || obj.ContentType != "text/plain" || obj.ETag != "etag" {
		t.Fatalf("unexpected obj: %+v", obj)
	}

	if _, _, err := b.Open(ctx, "missing", 0, 0, nil); !errors.Is(err, storage.ErrNotExist) {
		t.Fatalf("expected ErrNotExist")
	}
	if _, _, err := b.Open(ctx, "forbidden", 0, 0, nil); !errors.Is(err, storage.ErrPermission) {
		t.Fatalf("expected ErrPermission")
	}
	if _, _, err := b.Open(ctx, "weird", 0, 0, nil); err == nil {
		t.Fatalf("expected error for unexpected status")
	}

	// Stat success
	obj, err = b.Stat(ctx, "file", nil)
	if err != nil {
		t.Fatalf("Stat error: %v", err)
	}
	if obj.Size != 10 || obj.ContentType != "application/octet-stream" || obj.ETag != "etag2" {
		t.Fatalf("unexpected stat object: %+v", obj)
	}

	if _, err := b.Stat(ctx, "missing", nil); !errors.Is(err, storage.ErrNotExist) {
		t.Fatalf("Stat missing expected ErrNotExist")
	}
	if _, err := b.Stat(ctx, "forbidden", nil); !errors.Is(err, storage.ErrPermission) {
		t.Fatalf("Stat forbidden expected ErrPermission")
	}
	if _, err := b.Stat(ctx, "unknown", nil); err == nil {
		t.Fatalf("Stat expected error for unexpected status")
	}
}

func TestBucketListAndURL(t *testing.T) {
	ctx := context.Background()
	requests := make([]*http.Request, 0, 2)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r)
		if strings.Contains(r.URL.Path, "tree") {
			resp := []map[string]any{
				{"path": "prefix/file1.txt", "type": "file", "size": 10},
				{"rfilename": "prefix/dir/", "type": "directory"},
				{"path": "prefix/file2.txt", "type": "file", "LFS": map[string]any{"size": json.Number("20")}},
				{"path": "other/file3.txt", "type": "file", "filesize": 30},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	st := &hfStorage{client: server.Client(), baseURL: server.URL, repoID: "repo", repoType: "space", revision: "main"}
	b := &hfBucket{st: st}

	iter, err := b.List(ctx, "prefix/", 1, 0, storage.Options{"recursive": false})
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	defer func() {
		_ = iter.Close()
	}()

	obj, err := iter.Next()
	if err != nil || obj == nil {
		t.Fatalf("expected first object, got %v, %v", obj, err)
	}
	if obj.Key != "prefix/file1.txt" || obj.Size != 10 {
		t.Fatalf("unexpected first object: %+v", obj)
	}
	obj, err = iter.Next()
	if obj != nil || err != nil {
		t.Fatalf("expected no more objects, got %v %v", obj, err)
	}

	if got := b.st.treeURL(); !strings.Contains(got, "/spaces/") {
		t.Fatalf("treeURL not using space segment: %s", got)
	}

	if _, err := b.SignedURL(ctx, "key", http.MethodPost, time.Second, nil); !errors.Is(err, storage.ErrUnsupported) {
		t.Fatalf("URL should be unsupported for non-GET")
	}
	urlStr, err := b.SignedURL(ctx, "key", http.MethodGet, time.Second, nil)
	if err != nil || !strings.Contains(urlStr, "resolve/main/key") {
		t.Fatalf("unexpected URL %q err=%v", urlStr, err)
	}

	if len(requests) == 0 {
		t.Fatalf("expected at least one request recorded")
	}
	if strings.Contains(requests[0].URL.RawQuery, "recursive") {
		t.Fatalf("recursive parameter should be absent when set to false")
	}
}

func TestHelpers(t *testing.T) {
	st := &hfStorage{baseURL: "https://example.com/", repoID: "org/repo", repoType: "dataset", revision: "dev"}
	if got := st.fileURL("/path/to/file"); !strings.Contains(got, "/datasets/org/repo/resolve/dev/path/to/file") {
		t.Fatalf("fileURL unexpected: %s", got)
	}
	st.repoType = "model"
	if got := st.fileURL("file"); !strings.Contains(got, "/org/repo/resolve/dev/file") {
		t.Fatalf("fileURL for model unexpected: %s", got)
	}

	st.repoType = "datasets"
	if got := st.treeURL(); !strings.Contains(got, "/datasets/") {
		t.Fatalf("treeURL unexpected: %s", got)
	}
	st.repoType = "unknown"
	if got := st.treeURL(); !strings.Contains(got, "/models/") {
		t.Fatalf("treeURL default unexpected: %s", got)
	}

	hdr := http.Header{}
	hdr.Set("Content-Length", "123")
	hdr.Set("Content-Type", "text/plain")
	hdr.Set("ETag", "\"quoted\"")
	obj := (&hfBucket{st: st}).objectFromHeaders("k", hdr)
	if obj.Size != 123 || obj.ETag != "quoted" || obj.ContentType != "text/plain" {
		t.Fatalf("objectFromHeaders unexpected: %+v", obj)
	}

	if got := entryString(map[string]any{"k": "v"}, "k"); got != "v" {
		t.Fatalf("entryString unexpected %q", got)
	}
	if got := entryString(map[string]any{"k": 123}, "k"); got != "" {
		t.Fatalf("entryString should be empty for non-string")
	}

	if got := entryInt64(map[string]any{"k": json.Number("5")}, "k"); got != 5 {
		t.Fatalf("entryInt64 unexpected %d", got)
	}
	if got := entryInt64(map[string]any{"k": "x"}, "k"); got != -1 {
		t.Fatalf("entryInt64 unexpected for non-number: %d", got)
	}
}
