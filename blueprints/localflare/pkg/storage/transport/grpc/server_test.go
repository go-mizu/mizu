// File: lib/storage/transport/grpc/server_test.go

package grpc

import (
	"bytes"
	"context"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/storage"
	_ "github.com/go-mizu/blueprints/localflare/pkg/storage/driver/memory" // Register memory driver
	pb "github.com/go-mizu/blueprints/localflare/pkg/storage/transport/grpc/storagepb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// testServer holds test server state.
type testServer struct {
	server     *Server
	store      storage.Storage
	grpcServer *grpc.Server
	client     pb.StorageServiceClient
	conn       *grpc.ClientConn
	addr       string
}

// setupTestServer creates a test gRPC server backed by memory storage.
func setupTestServer(t *testing.T) *testServer {
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

	// Create gRPC client
	addr := ln.Addr().String()
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		grpcServer.Stop()
		store.Close()
		t.Fatalf("create client: %v", err)
	}

	client := pb.NewStorageServiceClient(conn)

	return &testServer{
		server:     server,
		store:      store,
		grpcServer: grpcServer,
		client:     client,
		conn:       conn,
		addr:       addr,
	}
}

func (ts *testServer) cleanup() {
	ts.conn.Close()
	ts.grpcServer.Stop()
	ts.store.Close()
}

// TestCreateBucket tests creating a bucket.
func TestCreateBucket(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Create bucket
	info, err := ts.client.CreateBucket(ctx, &pb.CreateBucketRequest{
		Name: "test-bucket",
	})
	if err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}

	if info.Name != "test-bucket" {
		t.Errorf("expected bucket name 'test-bucket', got %q", info.Name)
	}
}

// TestListBuckets tests listing buckets.
func TestListBuckets(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Create buckets
	for _, name := range []string{"bucket-a", "bucket-b", "bucket-c"} {
		_, err := ts.client.CreateBucket(ctx, &pb.CreateBucketRequest{Name: name})
		if err != nil {
			t.Fatalf("CreateBucket %s: %v", name, err)
		}
	}

	// List buckets
	stream, err := ts.client.ListBuckets(ctx, &pb.ListBucketsRequest{})
	if err != nil {
		t.Fatalf("ListBuckets: %v", err)
	}

	var names []string
	for {
		info, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Recv: %v", err)
		}
		names = append(names, info.Name)
	}

	if len(names) != 3 {
		t.Errorf("expected 3 buckets, got %d", len(names))
	}
}

// TestDeleteBucket tests deleting a bucket.
func TestDeleteBucket(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Create bucket
	_, err := ts.client.CreateBucket(ctx, &pb.CreateBucketRequest{Name: "to-delete"})
	if err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}

	// Delete bucket
	_, err = ts.client.DeleteBucket(ctx, &pb.DeleteBucketRequest{Name: "to-delete"})
	if err != nil {
		t.Fatalf("DeleteBucket: %v", err)
	}

	// Verify deleted
	stream, err := ts.client.ListBuckets(ctx, &pb.ListBucketsRequest{})
	if err != nil {
		t.Fatalf("ListBuckets: %v", err)
	}

	count := 0
	for {
		_, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Recv: %v", err)
		}
		count++
	}

	if count != 0 {
		t.Errorf("expected 0 buckets after delete, got %d", count)
	}
}

// TestWriteReadObject tests writing and reading an object.
func TestWriteReadObject(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Create bucket
	_, err := ts.client.CreateBucket(ctx, &pb.CreateBucketRequest{Name: "test-bucket"})
	if err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}

	// Write object
	content := []byte("Hello, gRPC Storage World!")
	writeStream, err := ts.client.WriteObject(ctx)
	if err != nil {
		t.Fatalf("WriteObject: %v", err)
	}

	// Send metadata
	err = writeStream.Send(&pb.WriteObjectRequest{
		Payload: &pb.WriteObjectRequest_Metadata{
			Metadata: &pb.WriteObjectMetadata{
				Bucket:      "test-bucket",
				Key:         "hello.txt",
				Size:        int64(len(content)),
				ContentType: "text/plain",
			},
		},
	})
	if err != nil {
		t.Fatalf("Send metadata: %v", err)
	}

	// Send data
	err = writeStream.Send(&pb.WriteObjectRequest{
		Payload: &pb.WriteObjectRequest_Data{
			Data: content,
		},
	})
	if err != nil {
		t.Fatalf("Send data: %v", err)
	}

	objInfo, err := writeStream.CloseAndRecv()
	if err != nil {
		t.Fatalf("CloseAndRecv: %v", err)
	}

	if objInfo.Size != int64(len(content)) {
		t.Errorf("expected size %d, got %d", len(content), objInfo.Size)
	}

	// Read object
	readStream, err := ts.client.ReadObject(ctx, &pb.ReadObjectRequest{
		Bucket: "test-bucket",
		Key:    "hello.txt",
	})
	if err != nil {
		t.Fatalf("ReadObject: %v", err)
	}

	// First message should be metadata
	msg, err := readStream.Recv()
	if err != nil {
		t.Fatalf("Recv metadata: %v", err)
	}

	meta := msg.GetMetadata()
	if meta == nil {
		t.Fatal("expected metadata in first message")
	}
	if meta.Size != int64(len(content)) {
		t.Errorf("expected size %d, got %d", len(content), meta.Size)
	}

	// Collect data
	var buf bytes.Buffer
	for {
		msg, err := readStream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Recv data: %v", err)
		}
		buf.Write(msg.GetData())
	}

	if !bytes.Equal(buf.Bytes(), content) {
		t.Errorf("content mismatch: got %q, want %q", buf.String(), string(content))
	}
}

// TestStatObject tests getting object metadata.
func TestStatObject(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Create bucket and object
	_, err := ts.client.CreateBucket(ctx, &pb.CreateBucketRequest{Name: "stat-bucket"})
	if err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}

	content := []byte("test content")
	writeStream, err := ts.client.WriteObject(ctx)
	if err != nil {
		t.Fatalf("WriteObject: %v", err)
	}

	err = writeStream.Send(&pb.WriteObjectRequest{
		Payload: &pb.WriteObjectRequest_Metadata{
			Metadata: &pb.WriteObjectMetadata{
				Bucket:      "stat-bucket",
				Key:         "test.txt",
				Size:        int64(len(content)),
				ContentType: "text/plain",
			},
		},
	})
	if err != nil {
		t.Fatalf("Send metadata: %v", err)
	}

	err = writeStream.Send(&pb.WriteObjectRequest{
		Payload: &pb.WriteObjectRequest_Data{Data: content},
	})
	if err != nil {
		t.Fatalf("Send data: %v", err)
	}

	_, err = writeStream.CloseAndRecv()
	if err != nil {
		t.Fatalf("CloseAndRecv: %v", err)
	}

	// Stat object
	info, err := ts.client.StatObject(ctx, &pb.StatObjectRequest{
		Bucket: "stat-bucket",
		Key:    "test.txt",
	})
	if err != nil {
		t.Fatalf("StatObject: %v", err)
	}

	if info.Size != int64(len(content)) {
		t.Errorf("expected size %d, got %d", len(content), info.Size)
	}
	if info.ContentType != "text/plain" {
		t.Errorf("expected content type 'text/plain', got %q", info.ContentType)
	}
}

// TestDeleteObject tests deleting an object.
func TestDeleteObject(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Create bucket and object
	_, err := ts.client.CreateBucket(ctx, &pb.CreateBucketRequest{Name: "delete-bucket"})
	if err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}

	content := []byte("to be deleted")
	writeStream, err := ts.client.WriteObject(ctx)
	if err != nil {
		t.Fatalf("WriteObject: %v", err)
	}

	err = writeStream.Send(&pb.WriteObjectRequest{
		Payload: &pb.WriteObjectRequest_Metadata{
			Metadata: &pb.WriteObjectMetadata{
				Bucket: "delete-bucket",
				Key:    "delete-me.txt",
				Size:   int64(len(content)),
			},
		},
	})
	if err != nil {
		t.Fatalf("Send metadata: %v", err)
	}

	err = writeStream.Send(&pb.WriteObjectRequest{
		Payload: &pb.WriteObjectRequest_Data{Data: content},
	})
	if err != nil {
		t.Fatalf("Send data: %v", err)
	}

	_, err = writeStream.CloseAndRecv()
	if err != nil {
		t.Fatalf("CloseAndRecv: %v", err)
	}

	// Delete object
	_, err = ts.client.DeleteObject(ctx, &pb.DeleteObjectRequest{
		Bucket: "delete-bucket",
		Key:    "delete-me.txt",
	})
	if err != nil {
		t.Fatalf("DeleteObject: %v", err)
	}

	// Verify deleted
	_, err = ts.client.StatObject(ctx, &pb.StatObjectRequest{
		Bucket: "delete-bucket",
		Key:    "delete-me.txt",
	})
	if err == nil {
		t.Error("expected error after delete, got nil")
	}
}

// TestListObjects tests listing objects.
func TestListObjects(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Create bucket
	_, err := ts.client.CreateBucket(ctx, &pb.CreateBucketRequest{Name: "list-bucket"})
	if err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}

	// Create objects
	for _, key := range []string{"file1.txt", "file2.txt", "dir/file3.txt"} {
		writeStream, err := ts.client.WriteObject(ctx)
		if err != nil {
			t.Fatalf("WriteObject: %v", err)
		}

		err = writeStream.Send(&pb.WriteObjectRequest{
			Payload: &pb.WriteObjectRequest_Metadata{
				Metadata: &pb.WriteObjectMetadata{
					Bucket: "list-bucket",
					Key:    key,
					Size:   4,
				},
			},
		})
		if err != nil {
			t.Fatalf("Send metadata: %v", err)
		}

		err = writeStream.Send(&pb.WriteObjectRequest{
			Payload: &pb.WriteObjectRequest_Data{Data: []byte("test")},
		})
		if err != nil {
			t.Fatalf("Send data: %v", err)
		}

		_, err = writeStream.CloseAndRecv()
		if err != nil {
			t.Fatalf("CloseAndRecv: %v", err)
		}
	}

	// List all objects
	stream, err := ts.client.ListObjects(ctx, &pb.ListObjectsRequest{
		Bucket: "list-bucket",
	})
	if err != nil {
		t.Fatalf("ListObjects: %v", err)
	}

	var keys []string
	for {
		obj, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Recv: %v", err)
		}
		keys = append(keys, obj.Key)
	}

	if len(keys) != 3 {
		t.Errorf("expected 3 objects, got %d: %v", len(keys), keys)
	}

	// List with prefix
	stream, err = ts.client.ListObjects(ctx, &pb.ListObjectsRequest{
		Bucket: "list-bucket",
		Prefix: "dir/",
	})
	if err != nil {
		t.Fatalf("ListObjects with prefix: %v", err)
	}

	var dirKeys []string
	for {
		obj, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Recv: %v", err)
		}
		dirKeys = append(dirKeys, obj.Key)
	}

	if len(dirKeys) != 1 {
		t.Errorf("expected 1 object with prefix 'dir/', got %d: %v", len(dirKeys), dirKeys)
	}
}

// TestCopyObject tests copying an object.
func TestCopyObject(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Create bucket
	_, err := ts.client.CreateBucket(ctx, &pb.CreateBucketRequest{Name: "copy-bucket"})
	if err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}

	// Create source object
	content := []byte("copy me!")
	writeStream, err := ts.client.WriteObject(ctx)
	if err != nil {
		t.Fatalf("WriteObject: %v", err)
	}

	err = writeStream.Send(&pb.WriteObjectRequest{
		Payload: &pb.WriteObjectRequest_Metadata{
			Metadata: &pb.WriteObjectMetadata{
				Bucket: "copy-bucket",
				Key:    "source.txt",
				Size:   int64(len(content)),
			},
		},
	})
	if err != nil {
		t.Fatalf("Send metadata: %v", err)
	}

	err = writeStream.Send(&pb.WriteObjectRequest{
		Payload: &pb.WriteObjectRequest_Data{Data: content},
	})
	if err != nil {
		t.Fatalf("Send data: %v", err)
	}

	_, err = writeStream.CloseAndRecv()
	if err != nil {
		t.Fatalf("CloseAndRecv: %v", err)
	}

	// Copy object
	copyInfo, err := ts.client.CopyObject(ctx, &pb.CopyObjectRequest{
		SrcBucket: "copy-bucket",
		SrcKey:    "source.txt",
		DstBucket: "copy-bucket",
		DstKey:    "dest.txt",
	})
	if err != nil {
		t.Fatalf("CopyObject: %v", err)
	}

	if copyInfo.Key != "dest.txt" {
		t.Errorf("expected key 'dest.txt', got %q", copyInfo.Key)
	}

	// Verify copy
	readStream, err := ts.client.ReadObject(ctx, &pb.ReadObjectRequest{
		Bucket: "copy-bucket",
		Key:    "dest.txt",
	})
	if err != nil {
		t.Fatalf("ReadObject: %v", err)
	}

	// Skip metadata
	_, err = readStream.Recv()
	if err != nil {
		t.Fatalf("Recv metadata: %v", err)
	}

	var buf bytes.Buffer
	for {
		msg, err := readStream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Recv: %v", err)
		}
		buf.Write(msg.GetData())
	}

	if !bytes.Equal(buf.Bytes(), content) {
		t.Errorf("content mismatch after copy")
	}
}

// TestMoveObject tests moving an object.
func TestMoveObject(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Create bucket
	_, err := ts.client.CreateBucket(ctx, &pb.CreateBucketRequest{Name: "move-bucket"})
	if err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}

	// Create source object
	content := []byte("move me!")
	writeStream, err := ts.client.WriteObject(ctx)
	if err != nil {
		t.Fatalf("WriteObject: %v", err)
	}

	err = writeStream.Send(&pb.WriteObjectRequest{
		Payload: &pb.WriteObjectRequest_Metadata{
			Metadata: &pb.WriteObjectMetadata{
				Bucket: "move-bucket",
				Key:    "old.txt",
				Size:   int64(len(content)),
			},
		},
	})
	if err != nil {
		t.Fatalf("Send metadata: %v", err)
	}

	err = writeStream.Send(&pb.WriteObjectRequest{
		Payload: &pb.WriteObjectRequest_Data{Data: content},
	})
	if err != nil {
		t.Fatalf("Send data: %v", err)
	}

	_, err = writeStream.CloseAndRecv()
	if err != nil {
		t.Fatalf("CloseAndRecv: %v", err)
	}

	// Move object
	moveInfo, err := ts.client.MoveObject(ctx, &pb.MoveObjectRequest{
		SrcBucket: "move-bucket",
		SrcKey:    "old.txt",
		DstBucket: "move-bucket",
		DstKey:    "new.txt",
	})
	if err != nil {
		t.Fatalf("MoveObject: %v", err)
	}

	if moveInfo.Key != "new.txt" {
		t.Errorf("expected key 'new.txt', got %q", moveInfo.Key)
	}

	// Verify source is gone
	_, err = ts.client.StatObject(ctx, &pb.StatObjectRequest{
		Bucket: "move-bucket",
		Key:    "old.txt",
	})
	if err == nil {
		t.Error("expected error for deleted source, got nil")
	}

	// Verify dest exists
	_, err = ts.client.StatObject(ctx, &pb.StatObjectRequest{
		Bucket: "move-bucket",
		Key:    "new.txt",
	})
	if err != nil {
		t.Fatalf("StatObject dest: %v", err)
	}
}

// TestGetFeatures tests getting storage features.
func TestGetFeatures(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup()

	ctx := context.Background()

	features, err := ts.client.GetFeatures(ctx, &pb.GetFeaturesRequest{})
	if err != nil {
		t.Fatalf("GetFeatures: %v", err)
	}

	// Memory storage should have some features
	if features.Features == nil {
		t.Error("expected non-nil features map")
	}
}

// TestLargeObject tests writing and reading a large object.
func TestLargeObject(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Create bucket
	_, err := ts.client.CreateBucket(ctx, &pb.CreateBucketRequest{Name: "large-bucket"})
	if err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}

	// Create large content (1MB)
	content := bytes.Repeat([]byte("X"), 1024*1024)

	writeStream, err := ts.client.WriteObject(ctx)
	if err != nil {
		t.Fatalf("WriteObject: %v", err)
	}

	// Send metadata
	err = writeStream.Send(&pb.WriteObjectRequest{
		Payload: &pb.WriteObjectRequest_Metadata{
			Metadata: &pb.WriteObjectMetadata{
				Bucket:      "large-bucket",
				Key:         "large.bin",
				Size:        int64(len(content)),
				ContentType: "application/octet-stream",
			},
		},
	})
	if err != nil {
		t.Fatalf("Send metadata: %v", err)
	}

	// Send data in chunks
	chunkSize := 64 * 1024
	for i := 0; i < len(content); i += chunkSize {
		end := i + chunkSize
		if end > len(content) {
			end = len(content)
		}
		err = writeStream.Send(&pb.WriteObjectRequest{
			Payload: &pb.WriteObjectRequest_Data{
				Data: content[i:end],
			},
		})
		if err != nil {
			t.Fatalf("Send data chunk: %v", err)
		}
	}

	objInfo, err := writeStream.CloseAndRecv()
	if err != nil {
		t.Fatalf("CloseAndRecv: %v", err)
	}

	if objInfo.Size != int64(len(content)) {
		t.Errorf("expected size %d, got %d", len(content), objInfo.Size)
	}

	// Read object
	readStream, err := ts.client.ReadObject(ctx, &pb.ReadObjectRequest{
		Bucket: "large-bucket",
		Key:    "large.bin",
	})
	if err != nil {
		t.Fatalf("ReadObject: %v", err)
	}

	// Skip metadata
	_, err = readStream.Recv()
	if err != nil {
		t.Fatalf("Recv metadata: %v", err)
	}

	// Collect data
	var buf bytes.Buffer
	for {
		msg, err := readStream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Recv data: %v", err)
		}
		buf.Write(msg.GetData())
	}

	if !bytes.Equal(buf.Bytes(), content) {
		t.Errorf("content mismatch: got %d bytes, want %d bytes", buf.Len(), len(content))
	}
}

// TestRangeRead tests reading a range of an object.
func TestRangeRead(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Create bucket
	_, err := ts.client.CreateBucket(ctx, &pb.CreateBucketRequest{Name: "range-bucket"})
	if err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}

	// Create object
	content := []byte("0123456789abcdefghij")
	writeStream, err := ts.client.WriteObject(ctx)
	if err != nil {
		t.Fatalf("WriteObject: %v", err)
	}

	err = writeStream.Send(&pb.WriteObjectRequest{
		Payload: &pb.WriteObjectRequest_Metadata{
			Metadata: &pb.WriteObjectMetadata{
				Bucket: "range-bucket",
				Key:    "range.txt",
				Size:   int64(len(content)),
			},
		},
	})
	if err != nil {
		t.Fatalf("Send metadata: %v", err)
	}

	err = writeStream.Send(&pb.WriteObjectRequest{
		Payload: &pb.WriteObjectRequest_Data{Data: content},
	})
	if err != nil {
		t.Fatalf("Send data: %v", err)
	}

	_, err = writeStream.CloseAndRecv()
	if err != nil {
		t.Fatalf("CloseAndRecv: %v", err)
	}

	// Read range
	readStream, err := ts.client.ReadObject(ctx, &pb.ReadObjectRequest{
		Bucket: "range-bucket",
		Key:    "range.txt",
		Offset: 5,
		Length: 10,
	})
	if err != nil {
		t.Fatalf("ReadObject: %v", err)
	}

	// Skip metadata
	_, err = readStream.Recv()
	if err != nil {
		t.Fatalf("Recv metadata: %v", err)
	}

	// Collect data
	var buf bytes.Buffer
	for {
		msg, err := readStream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Recv data: %v", err)
		}
		buf.Write(msg.GetData())
	}

	expected := content[5 : 5+10]
	if !bytes.Equal(buf.Bytes(), expected) {
		t.Errorf("content mismatch: got %q, want %q", buf.String(), string(expected))
	}
}

// TestAuthenticationRequired tests that authentication is enforced.
func TestAuthenticationRequired(t *testing.T) {
	ctx := context.Background()

	// Create memory storage
	store, err := storage.Open(ctx, "mem://")
	if err != nil {
		t.Fatalf("create memory storage: %v", err)
	}
	defer store.Close()

	// Configure server with authentication
	cfg := &Config{
		Auth: &AuthConfig{
			TokenValidator: func(token string) (map[string]any, error) {
				if token == "valid-token" {
					return map[string]any{"sub": "user123"}, nil
				}
				return nil, storage.ErrPermission
			},
		},
	}
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

	// Test without token - should fail
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	defer conn.Close()

	client := pb.NewStorageServiceClient(conn)
	_, err = client.CreateBucket(ctx, &pb.CreateBucketRequest{Name: "test"})
	if err == nil {
		t.Error("expected error without token, got nil")
	}
	if !strings.Contains(err.Error(), "Unauthenticated") {
		t.Errorf("expected Unauthenticated error, got: %v", err)
	}

	// Test with valid token - should succeed
	conn2, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithPerRPCCredentials(&TokenCredentials{Token: "valid-token", Insecure: true}),
	)
	if err != nil {
		t.Fatalf("create client with token: %v", err)
	}
	defer conn2.Close()

	client2 := pb.NewStorageServiceClient(conn2)
	_, err = client2.CreateBucket(ctx, &pb.CreateBucketRequest{Name: "test"})
	if err != nil {
		t.Errorf("expected success with valid token, got: %v", err)
	}
}
