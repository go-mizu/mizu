package flight

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/apache/arrow-go/v18/arrow/flight"
	"github.com/go-mizu/blueprints/localbase/pkg/storage"
	_ "github.com/go-mizu/blueprints/localbase/pkg/storage/driver/memory" // Register memory driver
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// testServer holds test server state.
type testServer struct {
	server       *Server
	store        storage.Storage
	grpcServer   *grpc.Server
	flightClient flight.Client
	addr         string
	lis          net.Listener
}

// setupTestServer creates a test Flight server backed by memory storage.
func setupTestServer(t *testing.T) *testServer {
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

	// Create Flight client
	addr := lis.Addr().String()
	flightClient, err := flight.NewClientWithMiddleware(addr, nil, nil,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		lis.Close()
		store.Close()
		t.Fatalf("create flight client: %v", err)
	}

	return &testServer{
		server:       server,
		store:        store,
		flightClient: flightClient,
		addr:         addr,
		lis:          lis,
	}
}

func (ts *testServer) cleanup() {
	ts.flightClient.Close()
	ts.server.Shutdown()
	ts.store.Close()
}

// TestFlightCreateBucket tests creating a bucket via DoAction.
func TestFlightCreateBucket(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Create bucket
	req := CreateBucketRequest{Name: "test-bucket"}
	reqBytes, _ := json.Marshal(req)

	stream, err := ts.flightClient.DoAction(ctx, &flight.Action{
		Type: ActionCreateBucket,
		Body: reqBytes,
	})
	if err != nil {
		t.Fatalf("DoAction CreateBucket: %v", err)
	}

	result, err := stream.Recv()
	if err != nil {
		t.Fatalf("Recv: %v", err)
	}

	info, err := DecodeBucketInfo(result.Body)
	if err != nil {
		t.Fatalf("DecodeBucketInfo: %v", err)
	}

	if info.Name != "test-bucket" {
		t.Errorf("expected bucket name 'test-bucket', got %q", info.Name)
	}
}

// TestFlightListBuckets tests listing buckets via ListFlights.
func TestFlightListBuckets(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Create buckets
	for _, name := range []string{"bucket-a", "bucket-b", "bucket-c"} {
		req := CreateBucketRequest{Name: name}
		reqBytes, _ := json.Marshal(req)

		stream, err := ts.flightClient.DoAction(ctx, &flight.Action{
			Type: ActionCreateBucket,
			Body: reqBytes,
		})
		if err != nil {
			t.Fatalf("DoAction CreateBucket %s: %v", name, err)
		}
		stream.Recv() // Drain response
	}

	// List buckets
	criteria := &Criteria{}
	criteriaBytes, _ := EncodeCriteria(criteria)

	stream, err := ts.flightClient.ListFlights(ctx, &flight.Criteria{Expression: criteriaBytes})
	if err != nil {
		t.Fatalf("ListFlights: %v", err)
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
		if len(info.FlightDescriptor.Path) > 0 {
			names = append(names, info.FlightDescriptor.Path[0])
		}
	}

	if len(names) != 3 {
		t.Errorf("expected 3 buckets, got %d: %v", len(names), names)
	}
}

// TestFlightDeleteBucket tests deleting a bucket via DoAction.
func TestFlightDeleteBucket(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Create bucket
	createReq := CreateBucketRequest{Name: "to-delete"}
	createReqBytes, _ := json.Marshal(createReq)

	stream, err := ts.flightClient.DoAction(ctx, &flight.Action{
		Type: ActionCreateBucket,
		Body: createReqBytes,
	})
	if err != nil {
		t.Fatalf("DoAction CreateBucket: %v", err)
	}
	stream.Recv()

	// Delete bucket
	deleteReq := DeleteBucketRequest{Name: "to-delete"}
	deleteReqBytes, _ := json.Marshal(deleteReq)

	stream, err = ts.flightClient.DoAction(ctx, &flight.Action{
		Type: ActionDeleteBucket,
		Body: deleteReqBytes,
	})
	if err != nil {
		t.Fatalf("DoAction DeleteBucket: %v", err)
	}
	_, err = stream.Recv()
	if err != nil && err != io.EOF {
		t.Fatalf("Recv: %v", err)
	}

	// Verify deleted
	criteria := &Criteria{}
	criteriaBytes, _ := EncodeCriteria(criteria)

	listStream, err := ts.flightClient.ListFlights(ctx, &flight.Criteria{Expression: criteriaBytes})
	if err != nil {
		t.Fatalf("ListFlights: %v", err)
	}

	count := 0
	for {
		_, err := listStream.Recv()
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

// TestFlightWriteReadObject tests writing and reading an object.
func TestFlightWriteReadObject(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Create bucket
	createReq := CreateBucketRequest{Name: "test-bucket"}
	createReqBytes, _ := json.Marshal(createReq)
	stream, _ := ts.flightClient.DoAction(ctx, &flight.Action{
		Type: ActionCreateBucket,
		Body: createReqBytes,
	})
	stream.Recv()

	// Write object using DoPut
	content := []byte("Hello, Flight Storage World!")

	putStream, err := ts.flightClient.DoPut(ctx)
	if err != nil {
		t.Fatalf("DoPut: %v", err)
	}

	// Create upload descriptor
	uploadDesc := &UploadDescriptor{
		Bucket:      "test-bucket",
		Key:         "hello.txt",
		Size:        int64(len(content)),
		ContentType: "text/plain",
	}
	descBytes, _ := EncodeUploadDescriptor(uploadDesc)

	writer := flight.NewRecordWriter(putStream)
	writer.SetFlightDescriptor(&flight.FlightDescriptor{
		Type: flight.DescriptorCMD,
		Cmd:  descBytes,
	})

	// Build and write record with data
	builder := NewObjectDataBuilder(nil)
	rec := builder.Build(content)
	if err := writer.Write(rec); err != nil {
		rec.Release()
		t.Fatalf("Write: %v", err)
	}
	rec.Release()
	writer.Close()

	// Close the send side of the stream
	if err := putStream.CloseSend(); err != nil {
		t.Fatalf("CloseSend: %v", err)
	}

	// Get result
	result, err := putStream.Recv()
	if err != nil {
		t.Fatalf("Recv result: %v", err)
	}

	objInfo, err := DecodeObjectInfo(result.AppMetadata)
	if err != nil {
		t.Fatalf("DecodeObjectInfo: %v", err)
	}

	if objInfo.Size != int64(len(content)) {
		t.Errorf("expected size %d, got %d", len(content), objInfo.Size)
	}

	// Read object using DoGet
	ticket := &Ticket{Bucket: "test-bucket", Key: "hello.txt"}
	ticketBytes, _ := EncodeTicket(ticket)

	getStream, err := ts.flightClient.DoGet(ctx, &flight.Ticket{Ticket: ticketBytes})
	if err != nil {
		t.Fatalf("DoGet: %v", err)
	}

	// First message contains metadata
	data, err := getStream.Recv()
	if err != nil {
		t.Fatalf("Recv metadata: %v", err)
	}

	if len(data.AppMetadata) > 0 {
		meta, _ := DecodeObjectInfo(data.AppMetadata)
		if meta != nil && meta.Size != int64(len(content)) {
			t.Errorf("metadata size mismatch: got %d, want %d", meta.Size, len(content))
		}
	}

	// Read data using Arrow reader
	reader, err := flight.NewRecordReader(getStream)
	if err != nil {
		t.Fatalf("NewRecordReader: %v", err)
	}
	defer reader.Release()

	var buf bytes.Buffer
	for reader.Next() {
		rec := reader.Record()
		if rec.NumCols() > 0 {
			col := rec.Column(0)
			if binArr, ok := col.(interface{ Value(int) []byte }); ok {
				for i := 0; i < col.Len(); i++ {
					buf.Write(binArr.Value(i))
				}
			}
		}
	}

	if !bytes.Equal(buf.Bytes(), content) {
		t.Errorf("content mismatch: got %q, want %q", buf.String(), string(content))
	}
}

// TestFlightStatObject tests getting object metadata via DoAction.
func TestFlightStatObject(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Create bucket
	createReq := CreateBucketRequest{Name: "stat-bucket"}
	createReqBytes, _ := json.Marshal(createReq)
	stream, _ := ts.flightClient.DoAction(ctx, &flight.Action{
		Type: ActionCreateBucket,
		Body: createReqBytes,
	})
	stream.Recv()

	// Write object directly via storage
	content := []byte("test content")
	bucket := ts.store.Bucket("stat-bucket")
	_, err := bucket.Write(ctx, "test.txt", bytes.NewReader(content), int64(len(content)), "text/plain", nil)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Stat object
	statReq := StatRequest{
		Bucket: "stat-bucket",
		Key:    "test.txt",
	}
	statReqBytes, _ := json.Marshal(statReq)

	statStream, err := ts.flightClient.DoAction(ctx, &flight.Action{
		Type: ActionStat,
		Body: statReqBytes,
	})
	if err != nil {
		t.Fatalf("DoAction Stat: %v", err)
	}

	result, err := statStream.Recv()
	if err != nil {
		t.Fatalf("Recv: %v", err)
	}

	info, err := DecodeObjectInfo(result.Body)
	if err != nil {
		t.Fatalf("DecodeObjectInfo: %v", err)
	}

	if info.Size != int64(len(content)) {
		t.Errorf("expected size %d, got %d", len(content), info.Size)
	}
	if info.ContentType != "text/plain" {
		t.Errorf("expected content type 'text/plain', got %q", info.ContentType)
	}
}

// TestFlightDeleteObject tests deleting an object via DoAction.
func TestFlightDeleteObject(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Create bucket and object
	createReq := CreateBucketRequest{Name: "delete-bucket"}
	createReqBytes, _ := json.Marshal(createReq)
	stream, _ := ts.flightClient.DoAction(ctx, &flight.Action{
		Type: ActionCreateBucket,
		Body: createReqBytes,
	})
	stream.Recv()

	content := []byte("to be deleted")
	bucket := ts.store.Bucket("delete-bucket")
	_, err := bucket.Write(ctx, "delete-me.txt", bytes.NewReader(content), int64(len(content)), "", nil)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Delete object
	deleteReq := DeleteObjectRequest{
		Bucket: "delete-bucket",
		Key:    "delete-me.txt",
	}
	deleteReqBytes, _ := json.Marshal(deleteReq)

	deleteStream, err := ts.flightClient.DoAction(ctx, &flight.Action{
		Type: ActionDeleteObject,
		Body: deleteReqBytes,
	})
	if err != nil {
		t.Fatalf("DoAction DeleteObject: %v", err)
	}
	deleteStream.Recv()

	// Verify deleted
	_, err = bucket.Stat(ctx, "delete-me.txt", nil)
	if err == nil {
		t.Error("expected error after delete, got nil")
	}
}

// TestFlightListObjects tests listing objects via ListFlights.
func TestFlightListObjects(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Create bucket
	createReq := CreateBucketRequest{Name: "list-bucket"}
	createReqBytes, _ := json.Marshal(createReq)
	stream, _ := ts.flightClient.DoAction(ctx, &flight.Action{
		Type: ActionCreateBucket,
		Body: createReqBytes,
	})
	stream.Recv()

	// Create objects
	bucket := ts.store.Bucket("list-bucket")
	for _, key := range []string{"file1.txt", "file2.txt", "dir/file3.txt"} {
		_, err := bucket.Write(ctx, key, bytes.NewReader([]byte("test")), 4, "", nil)
		if err != nil {
			t.Fatalf("Write %s: %v", key, err)
		}
	}

	// List all objects
	criteria := &Criteria{Bucket: "list-bucket"}
	criteriaBytes, _ := EncodeCriteria(criteria)

	listStream, err := ts.flightClient.ListFlights(ctx, &flight.Criteria{Expression: criteriaBytes})
	if err != nil {
		t.Fatalf("ListFlights: %v", err)
	}

	var keys []string
	for {
		info, err := listStream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Recv: %v", err)
		}
		if len(info.Endpoint) > 0 && len(info.Endpoint[0].AppMetadata) > 0 {
			obj, _ := DecodeObjectInfo(info.Endpoint[0].AppMetadata)
			if obj != nil {
				keys = append(keys, obj.Key)
			}
		}
	}

	if len(keys) != 3 {
		t.Errorf("expected 3 objects, got %d: %v", len(keys), keys)
	}

	// List with prefix
	criteria = &Criteria{Bucket: "list-bucket", Prefix: "dir/"}
	criteriaBytes, _ = EncodeCriteria(criteria)

	listStream, err = ts.flightClient.ListFlights(ctx, &flight.Criteria{Expression: criteriaBytes})
	if err != nil {
		t.Fatalf("ListFlights with prefix: %v", err)
	}

	var dirKeys []string
	for {
		info, err := listStream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Recv: %v", err)
		}
		if len(info.Endpoint) > 0 && len(info.Endpoint[0].AppMetadata) > 0 {
			obj, _ := DecodeObjectInfo(info.Endpoint[0].AppMetadata)
			if obj != nil {
				dirKeys = append(dirKeys, obj.Key)
			}
		}
	}

	if len(dirKeys) != 1 {
		t.Errorf("expected 1 object with prefix 'dir/', got %d: %v", len(dirKeys), dirKeys)
	}
}

// TestFlightCopyObject tests copying an object via DoAction.
func TestFlightCopyObject(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Create bucket
	createReq := CreateBucketRequest{Name: "copy-bucket"}
	createReqBytes, _ := json.Marshal(createReq)
	stream, _ := ts.flightClient.DoAction(ctx, &flight.Action{
		Type: ActionCreateBucket,
		Body: createReqBytes,
	})
	stream.Recv()

	// Create source object
	content := []byte("copy me!")
	bucket := ts.store.Bucket("copy-bucket")
	_, err := bucket.Write(ctx, "source.txt", bytes.NewReader(content), int64(len(content)), "", nil)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Copy object
	copyReq := CopyObjectRequest{
		SrcBucket: "copy-bucket",
		SrcKey:    "source.txt",
		DstBucket: "copy-bucket",
		DstKey:    "dest.txt",
	}
	copyReqBytes, _ := json.Marshal(copyReq)

	copyStream, err := ts.flightClient.DoAction(ctx, &flight.Action{
		Type: ActionCopyObject,
		Body: copyReqBytes,
	})
	if err != nil {
		t.Fatalf("DoAction CopyObject: %v", err)
	}

	result, err := copyStream.Recv()
	if err != nil {
		t.Fatalf("Recv: %v", err)
	}

	copyInfo, err := DecodeObjectInfo(result.Body)
	if err != nil {
		t.Fatalf("DecodeObjectInfo: %v", err)
	}

	if copyInfo.Key != "dest.txt" {
		t.Errorf("expected key 'dest.txt', got %q", copyInfo.Key)
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

// TestFlightMoveObject tests moving an object via DoAction.
func TestFlightMoveObject(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Create bucket
	createReq := CreateBucketRequest{Name: "move-bucket"}
	createReqBytes, _ := json.Marshal(createReq)
	stream, _ := ts.flightClient.DoAction(ctx, &flight.Action{
		Type: ActionCreateBucket,
		Body: createReqBytes,
	})
	stream.Recv()

	// Create source object
	content := []byte("move me!")
	bucket := ts.store.Bucket("move-bucket")
	_, err := bucket.Write(ctx, "old.txt", bytes.NewReader(content), int64(len(content)), "", nil)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Move object
	moveReq := MoveObjectRequest{
		SrcBucket: "move-bucket",
		SrcKey:    "old.txt",
		DstBucket: "move-bucket",
		DstKey:    "new.txt",
	}
	moveReqBytes, _ := json.Marshal(moveReq)

	moveStream, err := ts.flightClient.DoAction(ctx, &flight.Action{
		Type: ActionMoveObject,
		Body: moveReqBytes,
	})
	if err != nil {
		t.Fatalf("DoAction MoveObject: %v", err)
	}

	result, err := moveStream.Recv()
	if err != nil {
		t.Fatalf("Recv: %v", err)
	}

	moveInfo, err := DecodeObjectInfo(result.Body)
	if err != nil {
		t.Fatalf("DecodeObjectInfo: %v", err)
	}

	if moveInfo.Key != "new.txt" {
		t.Errorf("expected key 'new.txt', got %q", moveInfo.Key)
	}

	// Verify source is gone
	_, err = bucket.Stat(ctx, "old.txt", nil)
	if err == nil {
		t.Error("expected error for deleted source, got nil")
	}

	// Verify dest exists
	_, err = bucket.Stat(ctx, "new.txt", nil)
	if err != nil {
		t.Errorf("StatObject dest: %v", err)
	}
}

// TestFlightGetFeatures tests getting storage features via DoAction.
func TestFlightGetFeatures(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup()

	ctx := context.Background()

	stream, err := ts.flightClient.DoAction(ctx, &flight.Action{
		Type: ActionGetFeatures,
		Body: []byte("{}"),
	})
	if err != nil {
		t.Fatalf("DoAction GetFeatures: %v", err)
	}

	result, err := stream.Recv()
	if err != nil {
		t.Fatalf("Recv: %v", err)
	}

	features, err := DecodeFeatures(result.Body)
	if err != nil {
		t.Fatalf("DecodeFeatures: %v", err)
	}

	// Memory storage should have some features
	if features == nil {
		t.Error("expected non-nil features map")
	}
}

// TestFlightListActions tests listing available actions.
func TestFlightListActions(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup()

	ctx := context.Background()

	stream, err := ts.flightClient.ListActions(ctx, &flight.Empty{})
	if err != nil {
		t.Fatalf("ListActions: %v", err)
	}

	actionTypes := make(map[string]bool)
	for {
		action, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Recv action: %v", err)
		}
		actionTypes[action.Type] = true
	}

	expectedActions := []string{
		ActionCreateBucket,
		ActionDeleteBucket,
		ActionDeleteObject,
		ActionCopyObject,
		ActionMoveObject,
		ActionInitMultipart,
		ActionCompleteMultipart,
		ActionAbortMultipart,
		ActionSignedURL,
		ActionGetFeatures,
		ActionStat,
	}

	for _, expected := range expectedActions {
		if !actionTypes[expected] {
			t.Errorf("expected action %q not found", expected)
		}
	}
}

// TestFlightGetFlightInfo tests getting flight info.
func TestFlightGetFlightInfo(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Create bucket and object
	createReq := CreateBucketRequest{Name: "info-bucket"}
	createReqBytes, _ := json.Marshal(createReq)
	stream, _ := ts.flightClient.DoAction(ctx, &flight.Action{
		Type: ActionCreateBucket,
		Body: createReqBytes,
	})
	stream.Recv()

	content := []byte("info content")
	bucket := ts.store.Bucket("info-bucket")
	_, err := bucket.Write(ctx, "info.txt", bytes.NewReader(content), int64(len(content)), "text/plain", nil)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Get flight info for object
	desc := &flight.FlightDescriptor{
		Type: flight.DescriptorPATH,
		Path: []string{"info-bucket", "info.txt"},
	}

	info, err := ts.flightClient.GetFlightInfo(ctx, desc)
	if err != nil {
		t.Fatalf("GetFlightInfo: %v", err)
	}

	if info.TotalBytes != int64(len(content)) {
		t.Errorf("expected total bytes %d, got %d", len(content), info.TotalBytes)
	}

	if len(info.Endpoint) == 0 {
		t.Error("expected at least one endpoint")
	}
}

// TestFlightLargeObject tests writing and reading a large object.
func TestFlightLargeObject(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Create bucket
	createReq := CreateBucketRequest{Name: "large-bucket"}
	createReqBytes, _ := json.Marshal(createReq)
	stream, _ := ts.flightClient.DoAction(ctx, &flight.Action{
		Type: ActionCreateBucket,
		Body: createReqBytes,
	})
	stream.Recv()

	// Create large content (512KB)
	content := bytes.Repeat([]byte("X"), 512*1024)

	// Write object using storage directly
	bucket := ts.store.Bucket("large-bucket")
	_, err := bucket.Write(ctx, "large.bin", bytes.NewReader(content), int64(len(content)), "application/octet-stream", nil)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Read object using DoGet
	ticket := &Ticket{Bucket: "large-bucket", Key: "large.bin"}
	ticketBytes, _ := EncodeTicket(ticket)

	getStream, err := ts.flightClient.DoGet(ctx, &flight.Ticket{Ticket: ticketBytes})
	if err != nil {
		t.Fatalf("DoGet: %v", err)
	}

	// Skip first message (metadata)
	_, err = getStream.Recv()
	if err != nil {
		t.Fatalf("Recv metadata: %v", err)
	}

	// Read data
	reader, err := flight.NewRecordReader(getStream)
	if err != nil {
		t.Fatalf("NewRecordReader: %v", err)
	}
	defer reader.Release()

	var buf bytes.Buffer
	for reader.Next() {
		rec := reader.Record()
		if rec.NumCols() > 0 {
			col := rec.Column(0)
			if binArr, ok := col.(interface{ Value(int) []byte }); ok {
				for i := 0; i < col.Len(); i++ {
					buf.Write(binArr.Value(i))
				}
			}
		}
	}

	if buf.Len() != len(content) {
		t.Errorf("content length mismatch: got %d bytes, want %d bytes", buf.Len(), len(content))
	}

	if !bytes.Equal(buf.Bytes(), content) {
		t.Error("content mismatch")
	}
}

// TestFlightRangeRead tests reading a range of an object.
func TestFlightRangeRead(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Create bucket
	createReq := CreateBucketRequest{Name: "range-bucket"}
	createReqBytes, _ := json.Marshal(createReq)
	stream, _ := ts.flightClient.DoAction(ctx, &flight.Action{
		Type: ActionCreateBucket,
		Body: createReqBytes,
	})
	stream.Recv()

	// Create object
	content := []byte("0123456789abcdefghij")
	bucket := ts.store.Bucket("range-bucket")
	_, err := bucket.Write(ctx, "range.txt", bytes.NewReader(content), int64(len(content)), "", nil)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Read range
	ticket := &Ticket{
		Bucket: "range-bucket",
		Key:    "range.txt",
		Offset: 5,
		Length: 10,
	}
	ticketBytes, _ := EncodeTicket(ticket)

	getStream, err := ts.flightClient.DoGet(ctx, &flight.Ticket{Ticket: ticketBytes})
	if err != nil {
		t.Fatalf("DoGet: %v", err)
	}

	// Skip metadata
	_, err = getStream.Recv()
	if err != nil {
		t.Fatalf("Recv metadata: %v", err)
	}

	// Read data
	reader, err := flight.NewRecordReader(getStream)
	if err != nil {
		t.Fatalf("NewRecordReader: %v", err)
	}
	defer reader.Release()

	var buf bytes.Buffer
	for reader.Next() {
		rec := reader.Record()
		if rec.NumCols() > 0 {
			col := rec.Column(0)
			if binArr, ok := col.(interface{ Value(int) []byte }); ok {
				for i := 0; i < col.Len(); i++ {
					buf.Write(binArr.Value(i))
				}
			}
		}
	}

	expected := content[5 : 5+10]
	if !bytes.Equal(buf.Bytes(), expected) {
		t.Errorf("content mismatch: got %q, want %q", buf.String(), string(expected))
	}
}

// TestFlightAuthenticationRequired tests that authentication is enforced.
func TestFlightAuthenticationRequired(t *testing.T) {
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

	// Create listener
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer lis.Close()

	go server.ServeListener(lis)
	defer server.Shutdown()

	time.Sleep(50 * time.Millisecond)

	addr := lis.Addr().String()

	// Test without token - should fail
	client, err := flight.NewClientWithMiddleware(addr, nil, nil,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	defer client.Close()

	req := CreateBucketRequest{Name: "test"}
	reqBytes, _ := json.Marshal(req)
	stream, err := client.DoAction(ctx, &flight.Action{
		Type: ActionCreateBucket,
		Body: reqBytes,
	})
	if err != nil {
		// Error on initial call is also acceptable
		if !strings.Contains(err.Error(), "Unauthenticated") && !strings.Contains(err.Error(), "missing") {
			t.Errorf("expected Unauthenticated error on call, got: %v", err)
		}
	} else {
		// If no error on call, should get error on Recv
		_, err = stream.Recv()
		if err == nil {
			t.Error("expected error without token, got nil")
		} else if !strings.Contains(err.Error(), "Unauthenticated") && !strings.Contains(err.Error(), "missing") {
			t.Errorf("expected Unauthenticated error, got: %v", err)
		}
	}

	// Test with valid token - should succeed
	client2, err := flight.NewClientWithMiddleware(addr, nil, nil,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithPerRPCCredentials(&TokenCredentials{Token: "valid-token", Insecure: true}),
	)
	if err != nil {
		t.Fatalf("create client with token: %v", err)
	}
	defer client2.Close()

	stream2, err := client2.DoAction(ctx, &flight.Action{
		Type: ActionCreateBucket,
		Body: reqBytes,
	})
	if err != nil {
		t.Errorf("expected success with valid token, got: %v", err)
	} else {
		stream2.Recv() // Drain
	}
}

// TestFlightGetSchema tests getting the schema.
func TestFlightGetSchema(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup()

	ctx := context.Background()

	desc := &flight.FlightDescriptor{
		Type: flight.DescriptorPATH,
		Path: []string{"test"},
	}

	result, err := ts.flightClient.GetSchema(ctx, desc)
	if err != nil {
		t.Fatalf("GetSchema: %v", err)
	}

	if len(result.Schema) == 0 {
		t.Error("expected non-empty schema")
	}
}
