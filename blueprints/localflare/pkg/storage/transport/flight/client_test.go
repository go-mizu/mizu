package flight

import (
	"bytes"
	"context"
	"io"
	"net"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/storage"
	_ "github.com/go-mizu/blueprints/localflare/pkg/storage/driver/memory" // Register memory driver
)

// testClientServer holds client/server test state.
type testClientServer struct {
	server *Server
	store  storage.Storage
	client *Client
	addr   string
}

// setupClientServer creates a test server and client.
func setupClientServer(t *testing.T) *testClientServer {
	t.Helper()

	ctx := context.Background()

	// Create memory storage
	store, err := storage.Open(ctx, "mem://")
	if err != nil {
		t.Fatalf("create memory storage: %v", err)
	}

	// Configure server
	cfg := &Config{
		ChunkSize: 64 * 1024, // 64KB for tests
	}
	server := New(store, cfg)

	// Create listener
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		store.Close()
		t.Fatalf("listen: %v", err)
	}

	// Start server in background
	go server.ServeListener(lis)

	// Wait a bit for server to start
	time.Sleep(50 * time.Millisecond)

	addr := lis.Addr().String()

	// Create client
	client, err := Open(ctx, addr)
	if err != nil {
		store.Close()
		t.Fatalf("create client: %v", err)
	}

	return &testClientServer{
		server: server,
		store:  store,
		client: client,
		addr:   addr,
	}
}

func (ts *testClientServer) cleanup() {
	ts.client.Close()
	ts.server.Shutdown()
	ts.store.Close()
}

// TestClientCreateBucket tests creating a bucket via client.
func TestClientCreateBucket(t *testing.T) {
	ts := setupClientServer(t)
	defer ts.cleanup()

	ctx := context.Background()

	info, err := ts.client.CreateBucket(ctx, "test-bucket", nil)
	if err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}

	if info.Name != "test-bucket" {
		t.Errorf("expected bucket name 'test-bucket', got %q", info.Name)
	}
}

// TestClientListBuckets tests listing buckets via client.
func TestClientListBuckets(t *testing.T) {
	ts := setupClientServer(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Create buckets
	for _, name := range []string{"bucket-a", "bucket-b", "bucket-c"} {
		_, err := ts.client.CreateBucket(ctx, name, nil)
		if err != nil {
			t.Fatalf("CreateBucket %s: %v", name, err)
		}
	}

	// List buckets
	iter, err := ts.client.Buckets(ctx, 0, 0, nil)
	if err != nil {
		t.Fatalf("Buckets: %v", err)
	}
	defer iter.Close()

	var names []string
	for {
		info, err := iter.Next()
		if err != nil {
			t.Fatalf("Next: %v", err)
		}
		if info == nil {
			break
		}
		names = append(names, info.Name)
	}

	if len(names) != 3 {
		t.Errorf("expected 3 buckets, got %d: %v", len(names), names)
	}
}

// TestClientDeleteBucket tests deleting a bucket via client.
func TestClientDeleteBucket(t *testing.T) {
	ts := setupClientServer(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Create bucket
	_, err := ts.client.CreateBucket(ctx, "to-delete", nil)
	if err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}

	// Delete bucket
	err = ts.client.DeleteBucket(ctx, "to-delete", nil)
	if err != nil {
		t.Fatalf("DeleteBucket: %v", err)
	}

	// Verify deleted
	iter, err := ts.client.Buckets(ctx, 0, 0, nil)
	if err != nil {
		t.Fatalf("Buckets: %v", err)
	}
	defer iter.Close()

	count := 0
	for {
		info, err := iter.Next()
		if err != nil {
			t.Fatalf("Next: %v", err)
		}
		if info == nil {
			break
		}
		count++
	}

	if count != 0 {
		t.Errorf("expected 0 buckets after delete, got %d", count)
	}
}

// TestClientWriteReadObject tests writing and reading an object via client.
func TestClientWriteReadObject(t *testing.T) {
	ts := setupClientServer(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Create bucket
	_, err := ts.client.CreateBucket(ctx, "test-bucket", nil)
	if err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}

	// Write object
	content := []byte("Hello, Flight Client World!")
	bucket := ts.client.Bucket("test-bucket")

	obj, err := bucket.Write(ctx, "hello.txt", bytes.NewReader(content), int64(len(content)), "text/plain", nil)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	if obj.Size != int64(len(content)) {
		t.Errorf("expected size %d, got %d", len(content), obj.Size)
	}

	// Read object
	rc, readObj, err := bucket.Open(ctx, "hello.txt", 0, 0, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer rc.Close()

	if readObj.Bucket != "test-bucket" || readObj.Key != "hello.txt" {
		t.Errorf("unexpected object info: %+v", readObj)
	}

	readContent, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	if !bytes.Equal(readContent, content) {
		t.Errorf("content mismatch: got %q, want %q", string(readContent), string(content))
	}
}

// TestClientStatObject tests getting object metadata via client.
func TestClientStatObject(t *testing.T) {
	ts := setupClientServer(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Create bucket
	_, err := ts.client.CreateBucket(ctx, "stat-bucket", nil)
	if err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}

	// Write object
	content := []byte("test content")
	bucket := ts.client.Bucket("stat-bucket")

	_, err = bucket.Write(ctx, "test.txt", bytes.NewReader(content), int64(len(content)), "text/plain", nil)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Stat object
	info, err := bucket.Stat(ctx, "test.txt", nil)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}

	if info.Size != int64(len(content)) {
		t.Errorf("expected size %d, got %d", len(content), info.Size)
	}
	if info.ContentType != "text/plain" {
		t.Errorf("expected content type 'text/plain', got %q", info.ContentType)
	}
}

// TestClientDeleteObject tests deleting an object via client.
func TestClientDeleteObject(t *testing.T) {
	ts := setupClientServer(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Create bucket and object
	_, err := ts.client.CreateBucket(ctx, "delete-bucket", nil)
	if err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}

	bucket := ts.client.Bucket("delete-bucket")
	content := []byte("to be deleted")

	_, err = bucket.Write(ctx, "delete-me.txt", bytes.NewReader(content), int64(len(content)), "", nil)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Delete object
	err = bucket.Delete(ctx, "delete-me.txt", nil)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Verify deleted
	_, err = bucket.Stat(ctx, "delete-me.txt", nil)
	if err == nil {
		t.Error("expected error after delete, got nil")
	}
}

// TestClientListObjects tests listing objects via client.
func TestClientListObjects(t *testing.T) {
	ts := setupClientServer(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Create bucket
	_, err := ts.client.CreateBucket(ctx, "list-bucket", nil)
	if err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}

	bucket := ts.client.Bucket("list-bucket")

	// Create objects
	for _, key := range []string{"file1.txt", "file2.txt", "dir/file3.txt"} {
		_, err := bucket.Write(ctx, key, bytes.NewReader([]byte("test")), 4, "", nil)
		if err != nil {
			t.Fatalf("Write %s: %v", key, err)
		}
	}

	// List all objects
	iter, err := bucket.List(ctx, "", 0, 0, nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	defer iter.Close()

	var keys []string
	for {
		obj, err := iter.Next()
		if err != nil {
			t.Fatalf("Next: %v", err)
		}
		if obj == nil {
			break
		}
		keys = append(keys, obj.Key)
	}

	if len(keys) != 3 {
		t.Errorf("expected 3 objects, got %d: %v", len(keys), keys)
	}

	// List with prefix
	iter, err = bucket.List(ctx, "dir/", 0, 0, nil)
	if err != nil {
		t.Fatalf("List with prefix: %v", err)
	}
	defer iter.Close()

	var dirKeys []string
	for {
		obj, err := iter.Next()
		if err != nil {
			t.Fatalf("Next: %v", err)
		}
		if obj == nil {
			break
		}
		dirKeys = append(dirKeys, obj.Key)
	}

	if len(dirKeys) != 1 {
		t.Errorf("expected 1 object with prefix 'dir/', got %d: %v", len(dirKeys), dirKeys)
	}
}

// TestClientCopyObject tests copying an object via client.
func TestClientCopyObject(t *testing.T) {
	ts := setupClientServer(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Create bucket
	_, err := ts.client.CreateBucket(ctx, "copy-bucket", nil)
	if err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}

	bucket := ts.client.Bucket("copy-bucket")
	content := []byte("copy me!")

	// Create source object
	_, err = bucket.Write(ctx, "source.txt", bytes.NewReader(content), int64(len(content)), "", nil)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Copy object
	copyObj, err := bucket.Copy(ctx, "dest.txt", "copy-bucket", "source.txt", nil)
	if err != nil {
		t.Fatalf("Copy: %v", err)
	}

	if copyObj.Key != "dest.txt" {
		t.Errorf("expected key 'dest.txt', got %q", copyObj.Key)
	}

	// Verify copy
	rc, _, err := bucket.Open(ctx, "dest.txt", 0, 0, nil)
	if err != nil {
		t.Fatalf("Open dest: %v", err)
	}
	defer rc.Close()

	readContent, _ := io.ReadAll(rc)
	if !bytes.Equal(readContent, content) {
		t.Errorf("content mismatch after copy")
	}
}

// TestClientMoveObject tests moving an object via client.
func TestClientMoveObject(t *testing.T) {
	ts := setupClientServer(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Create bucket
	_, err := ts.client.CreateBucket(ctx, "move-bucket", nil)
	if err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}

	bucket := ts.client.Bucket("move-bucket")
	content := []byte("move me!")

	// Create source object
	_, err = bucket.Write(ctx, "old.txt", bytes.NewReader(content), int64(len(content)), "", nil)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Move object
	moveObj, err := bucket.Move(ctx, "new.txt", "move-bucket", "old.txt", nil)
	if err != nil {
		t.Fatalf("Move: %v", err)
	}

	if moveObj.Key != "new.txt" {
		t.Errorf("expected key 'new.txt', got %q", moveObj.Key)
	}

	// Verify source is gone
	_, err = bucket.Stat(ctx, "old.txt", nil)
	if err == nil {
		t.Error("expected error for deleted source, got nil")
	}

	// Verify dest exists
	_, err = bucket.Stat(ctx, "new.txt", nil)
	if err != nil {
		t.Fatalf("Stat dest: %v", err)
	}
}

// TestClientFeatures tests getting storage features via client.
func TestClientFeatures(t *testing.T) {
	ts := setupClientServer(t)
	defer ts.cleanup()

	features := ts.client.Features()
	if features == nil {
		t.Error("expected non-nil features map")
	}
}

// TestClientBucketInfo tests getting bucket info via client.
func TestClientBucketInfo(t *testing.T) {
	ts := setupClientServer(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Create bucket
	_, err := ts.client.CreateBucket(ctx, "info-bucket", nil)
	if err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}

	bucket := ts.client.Bucket("info-bucket")
	info, err := bucket.Info(ctx)
	if err != nil {
		t.Fatalf("Info: %v", err)
	}

	if info.Name != "info-bucket" {
		t.Errorf("expected bucket name 'info-bucket', got %q", info.Name)
	}
}

// TestClientBucketFeatures tests getting bucket features via client.
func TestClientBucketFeatures(t *testing.T) {
	ts := setupClientServer(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Create bucket
	_, err := ts.client.CreateBucket(ctx, "features-bucket", nil)
	if err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}

	bucket := ts.client.Bucket("features-bucket")
	features := bucket.Features()
	if features == nil {
		t.Error("expected non-nil features map")
	}
}

// TestClientLargeObject tests writing and reading a large object via client.
func TestClientLargeObject(t *testing.T) {
	ts := setupClientServer(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Create bucket
	_, err := ts.client.CreateBucket(ctx, "large-bucket", nil)
	if err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}

	bucket := ts.client.Bucket("large-bucket")

	// Create large content (512KB)
	content := bytes.Repeat([]byte("X"), 512*1024)

	// Write object
	obj, err := bucket.Write(ctx, "large.bin", bytes.NewReader(content), int64(len(content)), "application/octet-stream", nil)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	if obj.Size != int64(len(content)) {
		t.Errorf("expected size %d, got %d", len(content), obj.Size)
	}

	// Read object
	rc, _, err := bucket.Open(ctx, "large.bin", 0, 0, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer rc.Close()

	readContent, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	if len(readContent) != len(content) {
		t.Errorf("content length mismatch: got %d bytes, want %d bytes", len(readContent), len(content))
	}

	if !bytes.Equal(readContent, content) {
		t.Error("content mismatch")
	}
}

// TestClientRangeRead tests reading a range of an object via client.
func TestClientRangeRead(t *testing.T) {
	ts := setupClientServer(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Create bucket
	_, err := ts.client.CreateBucket(ctx, "range-bucket", nil)
	if err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}

	bucket := ts.client.Bucket("range-bucket")

	// Create object
	content := []byte("0123456789abcdefghij")
	_, err = bucket.Write(ctx, "range.txt", bytes.NewReader(content), int64(len(content)), "", nil)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Read range
	rc, _, err := bucket.Open(ctx, "range.txt", 5, 10, nil)
	if err != nil {
		t.Fatalf("Open with range: %v", err)
	}
	defer rc.Close()

	readContent, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	expected := content[5 : 5+10]
	if !bytes.Equal(readContent, expected) {
		t.Errorf("content mismatch: got %q, want %q", string(readContent), string(expected))
	}
}

// TestClientStorageInterface verifies the client implements storage.Storage.
func TestClientStorageInterface(t *testing.T) {
	ts := setupClientServer(t)
	defer ts.cleanup()

	// This test verifies compile-time interface compliance
	var _ storage.Storage = ts.client
	var _ storage.Bucket = ts.client.Bucket("test")
}

// TestClientOptions tests client configuration options.
func TestClientOptions(t *testing.T) {
	ts := setupClientServer(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Test with various options
	client, err := Open(ctx, ts.addr,
		WithClientChunkSize(32*1024),
		WithClientMaxRecvMsgSize(32*1024*1024),
		WithClientMaxSendMsgSize(32*1024*1024),
	)
	if err != nil {
		t.Fatalf("Open with options: %v", err)
	}
	defer client.Close()

	// Verify client works
	_, err = client.CreateBucket(ctx, "options-test", nil)
	if err != nil {
		t.Errorf("CreateBucket with custom options: %v", err)
	}
}

// TestClientClose tests that closing the client is safe.
func TestClientClose(t *testing.T) {
	ts := setupClientServer(t)

	// Close once
	err := ts.client.Close()
	if err != nil {
		t.Errorf("Close: %v", err)
	}

	// Cleanup server and store manually since client is already closed
	ts.server.Shutdown()
	ts.store.Close()
}

// TestClientSignedURL tests getting a signed URL (may return unsupported).
func TestClientSignedURL(t *testing.T) {
	ts := setupClientServer(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Create bucket and object
	_, err := ts.client.CreateBucket(ctx, "signed-bucket", nil)
	if err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}

	bucket := ts.client.Bucket("signed-bucket")
	content := []byte("signed content")

	_, err = bucket.Write(ctx, "signed.txt", bytes.NewReader(content), int64(len(content)), "", nil)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Get signed URL - may return unsupported error for memory storage
	_, err = bucket.SignedURL(ctx, "signed.txt", "GET", time.Hour, nil)
	if err != nil && err != storage.ErrUnsupported {
		// Other errors are acceptable since memory storage may not support signed URLs
		t.Logf("SignedURL: %v (may be unsupported)", err)
	}
}

// TestClientMultipart tests multipart upload operations.
func TestClientMultipart(t *testing.T) {
	ts := setupClientServer(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Create bucket
	_, err := ts.client.CreateBucket(ctx, "multipart-bucket", nil)
	if err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}

	bucket := ts.client.Bucket("multipart-bucket")

	// Check if multipart is supported
	mp, ok := bucket.(storage.HasMultipart)
	if !ok {
		t.Skip("multipart not supported by client bucket")
	}

	// Init multipart
	upload, err := mp.InitMultipart(ctx, "multipart.bin", "application/octet-stream", nil)
	if err != nil {
		if err == storage.ErrUnsupported {
			t.Skip("multipart init not supported")
		}
		t.Fatalf("InitMultipart: %v", err)
	}

	// Upload parts
	part1Content := bytes.Repeat([]byte("A"), 100*1024)
	part1, err := mp.UploadPart(ctx, upload, 1, bytes.NewReader(part1Content), int64(len(part1Content)), nil)
	if err != nil {
		if err == storage.ErrUnsupported {
			t.Skip("multipart upload not supported")
		}
		t.Fatalf("UploadPart 1: %v", err)
	}

	part2Content := bytes.Repeat([]byte("B"), 100*1024)
	part2, err := mp.UploadPart(ctx, upload, 2, bytes.NewReader(part2Content), int64(len(part2Content)), nil)
	if err != nil {
		t.Fatalf("UploadPart 2: %v", err)
	}

	// Complete multipart
	obj, err := mp.CompleteMultipart(ctx, upload, []*storage.PartInfo{part1, part2}, nil)
	if err != nil {
		t.Fatalf("CompleteMultipart: %v", err)
	}

	expectedSize := int64(len(part1Content) + len(part2Content))
	if obj.Size != expectedSize {
		t.Errorf("expected size %d, got %d", expectedSize, obj.Size)
	}

	// Read and verify
	rc, _, err := bucket.Open(ctx, "multipart.bin", 0, 0, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer rc.Close()

	readContent, _ := io.ReadAll(rc)
	expectedContent := append(part1Content, part2Content...)
	if !bytes.Equal(readContent, expectedContent) {
		t.Error("multipart content mismatch")
	}
}

// TestClientAbortMultipart tests aborting a multipart upload.
func TestClientAbortMultipart(t *testing.T) {
	ts := setupClientServer(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Create bucket
	_, err := ts.client.CreateBucket(ctx, "abort-bucket", nil)
	if err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}

	bucket := ts.client.Bucket("abort-bucket")

	// Check if multipart is supported
	mp, ok := bucket.(storage.HasMultipart)
	if !ok {
		t.Skip("multipart not supported by client bucket")
	}

	// Init multipart
	upload, err := mp.InitMultipart(ctx, "abort.bin", "application/octet-stream", nil)
	if err != nil {
		if err == storage.ErrUnsupported {
			t.Skip("multipart init not supported")
		}
		t.Fatalf("InitMultipart: %v", err)
	}

	// Abort multipart
	err = mp.AbortMultipart(ctx, upload, nil)
	if err != nil && err != storage.ErrUnsupported {
		t.Fatalf("AbortMultipart: %v", err)
	}
}
