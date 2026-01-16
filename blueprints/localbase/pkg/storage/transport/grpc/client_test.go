// File: lib/storage/transport/grpc/client_test.go

package grpc

import (
	"bytes"
	"context"
	"io"
	"net"
	"testing"
	"time"

	"github.com/go-mizu/mizu/blueprints/localbase/pkg/storage"
	_ "github.com/go-mizu/mizu/blueprints/localbase/pkg/storage/driver/memory" // Register memory driver
	"google.golang.org/grpc"
)

// setupClientTest creates a test server and returns a Client that implements storage.Storage.
func setupClientTest(t *testing.T) (*Client, func()) {
	t.Helper()

	ctx := context.Background()

	// Create memory storage
	store, err := storage.Open(ctx, "mem://")
	if err != nil {
		t.Fatalf("create memory storage: %v", err)
	}

	// Configure server
	cfg := &Config{}
	server := New(store, cfg)

	// Create gRPC server
	grpcServer := grpc.NewServer(server.ServerOptions()...)
	server.Register(grpcServer)

	// Create listener
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		store.Close()
		t.Fatalf("listen: %v", err)
	}

	// Start server in background
	go grpcServer.Serve(ln)

	// Wait a bit for server to start
	time.Sleep(50 * time.Millisecond)

	// Create client
	addr := ln.Addr().String()
	client, err := Open(ctx, addr)
	if err != nil {
		grpcServer.Stop()
		store.Close()
		t.Fatalf("Open client: %v", err)
	}

	cleanup := func() {
		client.Close()
		grpcServer.Stop()
		store.Close()
	}

	return client, cleanup
}

// TestClientBucket tests that client returns a Bucket handle.
func TestClientBucket(t *testing.T) {
	client, cleanup := setupClientTest(t)
	defer cleanup()

	bucket := client.Bucket("test")
	if bucket == nil {
		t.Fatal("expected non-nil bucket")
	}
	if bucket.Name() != "test" {
		t.Errorf("expected bucket name 'test', got %q", bucket.Name())
	}
}

// TestClientCreateBucket tests creating a bucket via client.
func TestClientCreateBucket(t *testing.T) {
	client, cleanup := setupClientTest(t)
	defer cleanup()

	ctx := context.Background()

	info, err := client.CreateBucket(ctx, "my-bucket", nil)
	if err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}

	if info.Name != "my-bucket" {
		t.Errorf("expected bucket name 'my-bucket', got %q", info.Name)
	}
}

// TestClientBuckets tests listing buckets via client.
func TestClientBuckets(t *testing.T) {
	client, cleanup := setupClientTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create buckets
	for _, name := range []string{"bucket1", "bucket2"} {
		_, err := client.CreateBucket(ctx, name, nil)
		if err != nil {
			t.Fatalf("CreateBucket: %v", err)
		}
	}

	// List buckets
	iter, err := client.Buckets(ctx, 0, 0, nil)
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

	if len(names) != 2 {
		t.Errorf("expected 2 buckets, got %d", len(names))
	}
}

// TestClientDeleteBucket tests deleting a bucket via client.
func TestClientDeleteBucket(t *testing.T) {
	client, cleanup := setupClientTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create bucket
	_, err := client.CreateBucket(ctx, "delete-me", nil)
	if err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}

	// Delete bucket
	err = client.DeleteBucket(ctx, "delete-me", nil)
	if err != nil {
		t.Fatalf("DeleteBucket: %v", err)
	}

	// Verify deleted
	iter, err := client.Buckets(ctx, 0, 0, nil)
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
		t.Errorf("expected 0 buckets, got %d", count)
	}
}

// TestClientFeatures tests getting features via client.
func TestClientFeatures(t *testing.T) {
	client, cleanup := setupClientTest(t)
	defer cleanup()

	features := client.Features()
	// Should not panic, may be nil or have some features
	_ = features
}

// TestClientBucketWrite tests writing an object via client bucket.
func TestClientBucketWrite(t *testing.T) {
	client, cleanup := setupClientTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create bucket
	_, err := client.CreateBucket(ctx, "write-bucket", nil)
	if err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}

	// Write object
	bucket := client.Bucket("write-bucket")
	content := []byte("Hello, Client!")

	obj, err := bucket.Write(ctx, "hello.txt", bytes.NewReader(content), int64(len(content)), "text/plain", nil)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	if obj.Size != int64(len(content)) {
		t.Errorf("expected size %d, got %d", len(content), obj.Size)
	}
}

// TestClientBucketOpen tests reading an object via client bucket.
func TestClientBucketOpen(t *testing.T) {
	client, cleanup := setupClientTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create bucket and object
	_, err := client.CreateBucket(ctx, "read-bucket", nil)
	if err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}

	bucket := client.Bucket("read-bucket")
	content := []byte("Read me!")

	_, err = bucket.Write(ctx, "read.txt", bytes.NewReader(content), int64(len(content)), "text/plain", nil)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Open and read
	rc, obj, err := bucket.Open(ctx, "read.txt", 0, 0, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer rc.Close()

	if obj.Size != int64(len(content)) {
		t.Errorf("expected size %d, got %d", len(content), obj.Size)
	}

	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	if !bytes.Equal(data, content) {
		t.Errorf("content mismatch: got %q, want %q", string(data), string(content))
	}
}

// TestClientBucketStat tests getting object metadata via client bucket.
func TestClientBucketStat(t *testing.T) {
	client, cleanup := setupClientTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create bucket and object
	_, err := client.CreateBucket(ctx, "stat-bucket", nil)
	if err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}

	bucket := client.Bucket("stat-bucket")
	content := []byte("Stat me!")

	_, err = bucket.Write(ctx, "stat.txt", bytes.NewReader(content), int64(len(content)), "text/plain", nil)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Stat object
	obj, err := bucket.Stat(ctx, "stat.txt", nil)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}

	if obj.Size != int64(len(content)) {
		t.Errorf("expected size %d, got %d", len(content), obj.Size)
	}
	if obj.ContentType != "text/plain" {
		t.Errorf("expected content type 'text/plain', got %q", obj.ContentType)
	}
}

// TestClientBucketDelete tests deleting an object via client bucket.
func TestClientBucketDelete(t *testing.T) {
	client, cleanup := setupClientTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create bucket and object
	_, err := client.CreateBucket(ctx, "delete-bucket", nil)
	if err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}

	bucket := client.Bucket("delete-bucket")
	content := []byte("Delete me!")

	_, err = bucket.Write(ctx, "delete.txt", bytes.NewReader(content), int64(len(content)), "text/plain", nil)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Delete object
	err = bucket.Delete(ctx, "delete.txt", nil)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Verify deleted
	_, err = bucket.Stat(ctx, "delete.txt", nil)
	if err == nil {
		t.Error("expected error after delete, got nil")
	}
}

// TestClientBucketList tests listing objects via client bucket.
func TestClientBucketList(t *testing.T) {
	client, cleanup := setupClientTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create bucket and objects
	_, err := client.CreateBucket(ctx, "list-bucket", nil)
	if err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}

	bucket := client.Bucket("list-bucket")

	for _, key := range []string{"a.txt", "b.txt", "c.txt"} {
		_, err = bucket.Write(ctx, key, bytes.NewReader([]byte("test")), 4, "text/plain", nil)
		if err != nil {
			t.Fatalf("Write %s: %v", key, err)
		}
	}

	// List objects
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
		t.Errorf("expected 3 objects, got %d", len(keys))
	}
}

// TestClientBucketCopy tests copying an object via client bucket.
func TestClientBucketCopy(t *testing.T) {
	client, cleanup := setupClientTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create bucket and object
	_, err := client.CreateBucket(ctx, "copy-bucket", nil)
	if err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}

	bucket := client.Bucket("copy-bucket")
	content := []byte("Copy me!")

	_, err = bucket.Write(ctx, "src.txt", bytes.NewReader(content), int64(len(content)), "text/plain", nil)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Copy object
	obj, err := bucket.Copy(ctx, "dst.txt", "copy-bucket", "src.txt", nil)
	if err != nil {
		t.Fatalf("Copy: %v", err)
	}

	if obj.Key != "dst.txt" {
		t.Errorf("expected key 'dst.txt', got %q", obj.Key)
	}

	// Verify copy content
	rc, _, err := bucket.Open(ctx, "dst.txt", 0, 0, nil)
	if err != nil {
		t.Fatalf("Open dst: %v", err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	if !bytes.Equal(data, content) {
		t.Errorf("content mismatch after copy")
	}
}

// TestClientBucketMove tests moving an object via client bucket.
func TestClientBucketMove(t *testing.T) {
	client, cleanup := setupClientTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create bucket and object
	_, err := client.CreateBucket(ctx, "move-bucket", nil)
	if err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}

	bucket := client.Bucket("move-bucket")
	content := []byte("Move me!")

	_, err = bucket.Write(ctx, "old.txt", bytes.NewReader(content), int64(len(content)), "text/plain", nil)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Move object
	obj, err := bucket.Move(ctx, "new.txt", "move-bucket", "old.txt", nil)
	if err != nil {
		t.Fatalf("Move: %v", err)
	}

	if obj.Key != "new.txt" {
		t.Errorf("expected key 'new.txt', got %q", obj.Key)
	}

	// Verify source is gone
	_, err = bucket.Stat(ctx, "old.txt", nil)
	if err == nil {
		t.Error("expected error for old key, got nil")
	}

	// Verify dest exists
	_, err = bucket.Stat(ctx, "new.txt", nil)
	if err != nil {
		t.Errorf("expected no error for new key, got: %v", err)
	}
}

// TestClientBucketInfo tests getting bucket info via client bucket.
func TestClientBucketInfo(t *testing.T) {
	client, cleanup := setupClientTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create bucket
	_, err := client.CreateBucket(ctx, "info-bucket", nil)
	if err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}

	bucket := client.Bucket("info-bucket")

	info, err := bucket.Info(ctx)
	if err != nil {
		t.Fatalf("Info: %v", err)
	}

	if info.Name != "info-bucket" {
		t.Errorf("expected bucket name 'info-bucket', got %q", info.Name)
	}
}

// TestClientBucketFeatures tests getting bucket features via client bucket.
func TestClientBucketFeatures(t *testing.T) {
	client, cleanup := setupClientTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create bucket
	_, err := client.CreateBucket(ctx, "features-bucket", nil)
	if err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}

	bucket := client.Bucket("features-bucket")

	features := bucket.Features()
	// Should not panic, may be nil or have some features
	_ = features
}

// TestClientImplementsStorage verifies Client implements storage.Storage interface.
func TestClientImplementsStorage(t *testing.T) {
	client, cleanup := setupClientTest(t)
	defer cleanup()

	// This is a compile-time check
	var _ storage.Storage = client
}

// TestClientBucketImplementsBucket verifies clientBucket implements storage.Bucket interface.
func TestClientBucketImplementsBucket(t *testing.T) {
	client, cleanup := setupClientTest(t)
	defer cleanup()

	bucket := client.Bucket("test")

	// This is a compile-time check
	var _ storage.Bucket = bucket
}

// TestClientWithOptions tests client creation with options.
func TestClientWithOptions(t *testing.T) {
	ctx := context.Background()

	// Create memory storage
	store, err := storage.Open(ctx, "mem://")
	if err != nil {
		t.Fatalf("create memory storage: %v", err)
	}
	defer store.Close()

	// Configure server
	cfg := &Config{}
	server := New(store, cfg)

	// Create gRPC server
	grpcServer := grpc.NewServer(server.ServerOptions()...)
	server.Register(grpcServer)

	// Create listener
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	go grpcServer.Serve(ln)
	defer grpcServer.Stop()

	time.Sleep(50 * time.Millisecond)

	addr := ln.Addr().String()

	// Create client with options
	client, err := Open(ctx, addr,
		WithMaxRecvMsgSize(32*1024*1024),
		WithMaxSendMsgSize(32*1024*1024),
		WithChunkSize(128*1024),
	)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer client.Close()

	// Verify client works
	_, err = client.CreateBucket(ctx, "test-bucket", nil)
	if err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}
}

// TestClientRangeRead tests reading a range of an object via client.
func TestClientRangeRead(t *testing.T) {
	client, cleanup := setupClientTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create bucket and object
	_, err := client.CreateBucket(ctx, "range-bucket", nil)
	if err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}

	bucket := client.Bucket("range-bucket")
	content := []byte("0123456789abcdefghij")

	_, err = bucket.Write(ctx, "range.txt", bytes.NewReader(content), int64(len(content)), "text/plain", nil)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Read range
	rc, _, err := bucket.Open(ctx, "range.txt", 5, 10, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	expected := content[5 : 5+10]
	if !bytes.Equal(data, expected) {
		t.Errorf("content mismatch: got %q, want %q", string(data), string(expected))
	}
}

// TestClientLargeObject tests writing and reading a large object via client.
func TestClientLargeObject(t *testing.T) {
	client, cleanup := setupClientTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create bucket
	_, err := client.CreateBucket(ctx, "large-bucket", nil)
	if err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}

	bucket := client.Bucket("large-bucket")

	// Create large content (1MB)
	content := bytes.Repeat([]byte("Y"), 1024*1024)

	_, err = bucket.Write(ctx, "large.bin", bytes.NewReader(content), int64(len(content)), "application/octet-stream", nil)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Read back
	rc, obj, err := bucket.Open(ctx, "large.bin", 0, 0, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer rc.Close()

	if obj.Size != int64(len(content)) {
		t.Errorf("expected size %d, got %d", len(content), obj.Size)
	}

	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	if !bytes.Equal(data, content) {
		t.Errorf("content mismatch: got %d bytes, want %d bytes", len(data), len(content))
	}
}
