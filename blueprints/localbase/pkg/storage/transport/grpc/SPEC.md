# gRPC Transport Layer Specification

This document specifies the gRPC transport layer for `storage.Storage` backends, enabling high-performance remote storage access over gRPC.

## Overview

The gRPC transport exposes `storage.Storage` implementations over gRPC, providing:

- High-performance binary protocol with HTTP/2
- Bidirectional streaming for efficient large object transfers
- Strong typing via Protocol Buffers
- Built-in authentication via gRPC interceptors
- Client library for transparent remote storage access

## Architecture

```
┌─────────────────┐         gRPC          ┌─────────────────┐
│  gRPC Client    │◄──────────────────────►│  gRPC Server    │
│  (storage.      │    (HTTP/2 + Proto)    │                 │
│   Storage)      │                        │                 │
└─────────────────┘                        └────────┬────────┘
                                                    │
                                                    ▼
                                           ┌─────────────────┐
                                           │ storage.Storage │
                                           │   (any driver)  │
                                           └─────────────────┘
```

## Path Mapping

Consistent with other transports (SFTP, WebDAV):

| Path Pattern        | Maps To                    |
|---------------------|----------------------------|
| `/`                 | List buckets               |
| `/<bucket>/`        | Bucket root                |
| `/<bucket>/<key>`   | Object                     |

## Service Definition

### StorageService

The main service exposing all storage operations.

```protobuf
service StorageService {
  // Bucket Operations
  rpc CreateBucket(CreateBucketRequest) returns (BucketInfo);
  rpc DeleteBucket(DeleteBucketRequest) returns (google.protobuf.Empty);
  rpc GetBucket(GetBucketRequest) returns (BucketInfo);
  rpc ListBuckets(ListBucketsRequest) returns (stream BucketInfo);

  // Object Operations
  rpc StatObject(StatObjectRequest) returns (ObjectInfo);
  rpc DeleteObject(DeleteObjectRequest) returns (google.protobuf.Empty);
  rpc CopyObject(CopyObjectRequest) returns (ObjectInfo);
  rpc MoveObject(MoveObjectRequest) returns (ObjectInfo);
  rpc ListObjects(ListObjectsRequest) returns (stream ObjectInfo);

  // Streaming Data Operations
  rpc ReadObject(ReadObjectRequest) returns (stream ReadObjectResponse);
  rpc WriteObject(stream WriteObjectRequest) returns (ObjectInfo);

  // Multipart Upload Operations
  rpc InitMultipart(InitMultipartRequest) returns (MultipartUpload);
  rpc UploadPart(stream UploadPartRequest) returns (PartInfo);
  rpc ListParts(ListPartsRequest) returns (stream PartInfo);
  rpc CompleteMultipart(CompleteMultipartRequest) returns (ObjectInfo);
  rpc AbortMultipart(AbortMultipartRequest) returns (google.protobuf.Empty);

  // Signed URL
  rpc SignedURL(SignedURLRequest) returns (SignedURLResponse);

  // Features
  rpc GetFeatures(GetFeaturesRequest) returns (Features);
}
```

## Message Types

### Core Types

```protobuf
message BucketInfo {
  string name = 1;
  google.protobuf.Timestamp created_at = 2;
  bool public = 3;
  map<string, string> metadata = 4;
}

message ObjectInfo {
  string bucket = 1;
  string key = 2;
  int64 size = 3;
  string content_type = 4;
  string etag = 5;
  string version = 6;
  google.protobuf.Timestamp created = 7;
  google.protobuf.Timestamp updated = 8;
  map<string, string> hash = 9;
  map<string, string> metadata = 10;
  bool is_dir = 11;
}

message MultipartUpload {
  string bucket = 1;
  string key = 2;
  string upload_id = 3;
  map<string, string> metadata = 4;
}

message PartInfo {
  int32 number = 1;
  int64 size = 2;
  string etag = 3;
  map<string, string> hash = 4;
  google.protobuf.Timestamp last_modified = 5;
}

message Features {
  map<string, bool> features = 1;
}
```

### Request/Response Types

```protobuf
// Bucket Operations
message CreateBucketRequest {
  string name = 1;
  map<string, bytes> options = 2;
}

message DeleteBucketRequest {
  string name = 1;
  map<string, bytes> options = 2;
}

message GetBucketRequest {
  string name = 1;
}

message ListBucketsRequest {
  int32 limit = 1;
  int32 offset = 2;
  map<string, bytes> options = 3;
}

// Object Operations
message StatObjectRequest {
  string bucket = 1;
  string key = 2;
  map<string, bytes> options = 3;
}

message DeleteObjectRequest {
  string bucket = 1;
  string key = 2;
  map<string, bytes> options = 3;
}

message CopyObjectRequest {
  string src_bucket = 1;
  string src_key = 2;
  string dst_bucket = 3;
  string dst_key = 4;
  map<string, bytes> options = 5;
}

message MoveObjectRequest {
  string src_bucket = 1;
  string src_key = 2;
  string dst_bucket = 3;
  string dst_key = 4;
  map<string, bytes> options = 5;
}

message ListObjectsRequest {
  string bucket = 1;
  string prefix = 2;
  int32 limit = 3;
  int32 offset = 4;
  map<string, bytes> options = 5;
}

// Streaming Read
message ReadObjectRequest {
  string bucket = 1;
  string key = 2;
  int64 offset = 3;
  int64 length = 4;  // 0 or negative means full object
  map<string, bytes> options = 5;
}

message ReadObjectResponse {
  oneof payload {
    ObjectInfo metadata = 1;  // First message contains metadata
    bytes data = 2;           // Subsequent messages contain data chunks
  }
}

// Streaming Write
message WriteObjectRequest {
  oneof payload {
    WriteObjectMetadata metadata = 1;  // First message must be metadata
    bytes data = 2;                     // Subsequent messages are data chunks
  }
}

message WriteObjectMetadata {
  string bucket = 1;
  string key = 2;
  int64 size = 3;  // -1 for unknown/streaming
  string content_type = 4;
  map<string, bytes> options = 5;
}

// Multipart Operations
message InitMultipartRequest {
  string bucket = 1;
  string key = 2;
  string content_type = 3;
  map<string, bytes> options = 4;
}

message UploadPartRequest {
  oneof payload {
    UploadPartMetadata metadata = 1;
    bytes data = 2;
  }
}

message UploadPartMetadata {
  string bucket = 1;
  string key = 2;
  string upload_id = 3;
  int32 part_number = 4;
  int64 size = 5;
  map<string, bytes> options = 6;
}

message ListPartsRequest {
  string bucket = 1;
  string key = 2;
  string upload_id = 3;
  int32 limit = 4;
  int32 offset = 5;
  map<string, bytes> options = 6;
}

message CompleteMultipartRequest {
  string bucket = 1;
  string key = 2;
  string upload_id = 3;
  repeated PartInfo parts = 4;
  map<string, bytes> options = 5;
}

message AbortMultipartRequest {
  string bucket = 1;
  string key = 2;
  string upload_id = 3;
  map<string, bytes> options = 4;
}

// Signed URL
message SignedURLRequest {
  string bucket = 1;
  string key = 2;
  string method = 3;
  google.protobuf.Duration expires = 4;
  map<string, bytes> options = 5;
}

message SignedURLResponse {
  string url = 1;
}

// Features
message GetFeaturesRequest {
  string bucket = 1;  // Empty for storage-level features
}
```

## Error Handling

gRPC status codes map to storage errors:

| Storage Error           | gRPC Status Code        |
|-------------------------|-------------------------|
| `storage.ErrNotExist`   | `NOT_FOUND` (5)         |
| `storage.ErrExist`      | `ALREADY_EXISTS` (6)    |
| `storage.ErrPermission` | `PERMISSION_DENIED` (7) |
| `storage.ErrUnsupported`| `UNIMPLEMENTED` (12)    |
| Other errors            | `INTERNAL` (13)         |

Error details are provided in the status message.

## Authentication

### Token-Based Authentication

The transport supports token-based authentication via gRPC metadata:

```go
// Client sets authorization metadata
md := metadata.Pairs("authorization", "Bearer <token>")
ctx := metadata.NewOutgoingContext(ctx, md)
```

### Server Interceptor

```go
type AuthConfig struct {
    // TokenValidator validates the token and returns claims.
    TokenValidator func(token string) (map[string]any, error)

    // AllowUnauthenticated permits requests without tokens.
    AllowUnauthenticated bool
}
```

## Streaming Protocol

### Object Read Streaming

1. Client sends `ReadObjectRequest`
2. Server streams `ReadObjectResponse`:
   - First message: `metadata` field with `ObjectInfo`
   - Subsequent messages: `data` field with byte chunks (default 64KB)

### Object Write Streaming

1. Client streams `WriteObjectRequest`:
   - First message: `metadata` field with `WriteObjectMetadata`
   - Subsequent messages: `data` field with byte chunks
2. Server responds with final `ObjectInfo`

### Part Upload Streaming

Same pattern as object write for multipart part uploads.

## Configuration

### Server Configuration

```go
type Config struct {
    // MaxRecvMsgSize is the max message size in bytes. Default 16MB.
    MaxRecvMsgSize int

    // MaxSendMsgSize is the max message size in bytes. Default 16MB.
    MaxSendMsgSize int

    // ChunkSize for streaming reads. Default 64KB.
    ChunkSize int

    // Auth configures authentication. Nil disables auth.
    Auth *AuthConfig

    // Logger for server events.
    Logger *slog.Logger
}
```

### Client Configuration

```go
type ClientConfig struct {
    // Target is the gRPC server address (e.g., "localhost:9000").
    Target string

    // TLS configures TLS. Nil uses insecure connection.
    TLS *tls.Config

    // Token for authentication.
    Token string

    // MaxRecvMsgSize is the max message size in bytes. Default 16MB.
    MaxRecvMsgSize int

    // MaxSendMsgSize is the max message size in bytes. Default 16MB.
    MaxSendMsgSize int

    // ChunkSize for streaming writes. Default 64KB.
    ChunkSize int
}
```

## Client Implementation

The client implements `storage.Storage` interface, enabling transparent remote access:

```go
// Open connects to a gRPC storage server.
func Open(ctx context.Context, target string, opts ...ClientOption) (storage.Storage, error)

// Example usage:
store, err := grpc.Open(ctx, "localhost:9000",
    grpc.WithToken("bearer-token"),
    grpc.WithTLS(tlsConfig),
)
if err != nil {
    return err
}
defer store.Close()

// Use like any other storage.Storage
bucket := store.Bucket("my-bucket")
obj, err := bucket.Write(ctx, "key", reader, size, "text/plain", nil)
```

## Server Implementation

```go
// New creates a new gRPC server.
func New(store storage.Storage, cfg *Config) *Server

// Register registers the storage service with a gRPC server.
func (s *Server) Register(grpcServer *grpc.Server)

// Example usage:
store, _ := storage.Open(ctx, "local:///data")
server := grpc.New(store, &grpc.Config{
    Auth: &grpc.AuthConfig{
        TokenValidator: validateToken,
    },
})

grpcServer := grpc.NewServer()
server.Register(grpcServer)
lis, _ := net.Listen("tcp", ":9000")
grpcServer.Serve(lis)
```

## Performance Considerations

### Chunk Size

- Default 64KB balances latency and throughput
- Larger chunks reduce overhead for large files
- Smaller chunks provide better progress granularity

### Connection Pooling

- gRPC maintains connection pools automatically
- Configure `MaxConcurrentStreams` for high throughput

### Compression

- Enable gRPC compression for compressible content:
  ```go
  grpc.UseCompressor(gzip.Name)
  ```

## Wire Protocol

Uses Protocol Buffers v3 with gRPC over HTTP/2:

- Service: `storage.v1.StorageService`
- Package: `storage.v1`
- Full proto path: `lib/storage/transport/grpc/proto/storage.proto`

## Buf Configuration

```yaml
# buf.yaml
version: v2
modules:
  - path: proto
    name: buf.build/open-lake/storage
lint:
  use:
    - DEFAULT
breaking:
  use:
    - FILE
```

```yaml
# buf.gen.yaml
version: v2
managed:
  enabled: true
  override:
    - file_option: go_package_prefix
      value: open-lake.dev/lib/storage/transport/grpc
plugins:
  - remote: buf.build/protocolbuffers/go
    out: .
    opt: paths=source_relative
  - remote: buf.build/grpc/go
    out: .
    opt: paths=source_relative
```

## Directory Structure

```
lib/storage/transport/grpc/
├── SPEC.md                 # This specification
├── buf.yaml                # Buf module configuration
├── buf.gen.yaml            # Buf code generation config
├── proto/
│   └── storage.proto       # Protocol buffer definitions
├── storagepb/              # Generated Go code
│   ├── storage.pb.go
│   └── storage_grpc.pb.go
├── server.go               # gRPC server implementation
├── server_test.go          # Server tests
├── client.go               # gRPC client (storage.Storage impl)
├── client_test.go          # Client tests
├── auth.go                 # Authentication interceptors
└── errors.go               # Error mapping utilities
```

## Compatibility

- Protocol Buffers 3.x
- gRPC 1.50+
- Go 1.21+

## Future Considerations

- Server reflection for debugging
- Health checking service
- Watch/notification streaming (when storage supports it)
- Batch operations for efficiency
- Copy/move across storage backends
