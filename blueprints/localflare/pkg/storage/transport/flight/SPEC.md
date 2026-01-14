# Apache Arrow Flight Transport Layer Specification

This document specifies the Apache Arrow Flight transport layer for `storage.Storage` backends, enabling high-performance columnar data transfer using Arrow's zero-copy serialization.

## Overview

The Flight transport exposes `storage.Storage` implementations over Apache Arrow Flight RPC, providing:

- High-performance columnar data transfer using Arrow IPC format
- Zero-copy data serialization where possible
- Parallel data streams via multiple endpoints
- Bidirectional streaming for efficient large object transfers
- Built-in authentication via Flight middleware
- Client library for transparent remote storage access
- Full compatibility with Arrow ecosystem tools

## Architecture

```
┌─────────────────┐      Arrow Flight       ┌─────────────────┐
│  Flight Client  │◄──────────────────────►│  Flight Server  │
│  (storage.      │    (gRPC + Arrow IPC)   │                 │
│   Storage)      │                         │                 │
└─────────────────┘                         └────────┬────────┘
                                                     │
                                                     ▼
                                            ┌─────────────────┐
                                            │ storage.Storage │
                                            │   (any driver)  │
                                            └─────────────────┘
```

## Flight Protocol Mapping

The storage layer maps to Flight concepts as follows:

| Storage Concept     | Flight Concept                    |
|---------------------|-----------------------------------|
| Bucket              | FlightDescriptor (PATH type)      |
| Object              | FlightInfo + Ticket               |
| Object Data         | RecordBatch stream (binary column)|
| Object List         | ListFlights result stream         |
| Metadata Operations | DoAction                          |
| Upload              | DoPut                             |
| Download            | DoGet                             |
| Bidirectional       | DoExchange                        |

## FlightDescriptor Schema

### Path-Based Descriptors

Storage operations use PATH-type descriptors:

| Path Components         | Operation              |
|-------------------------|------------------------|
| `[]`                    | List all buckets       |
| `["bucket"]`            | Bucket operations      |
| `["bucket", "key/path"]`| Object operations      |

### Command-Based Descriptors

Extended operations use CMD-type descriptors with JSON payload:

```json
{
  "action": "create_bucket",
  "bucket": "my-bucket",
  "options": {}
}
```

## Arrow Schema Definitions

### Object Data Schema

Objects are transferred as Arrow RecordBatches with the following schema:

```
Schema: Object Data Stream
├── data: Binary (the object content as a single binary value)
└── (metadata fields in schema metadata)

Schema Metadata:
  - bucket: string
  - key: string
  - size: int64
  - content_type: string
  - etag: string
  - version: string
  - created: timestamp (RFC3339)
  - updated: timestamp (RFC3339)
  - hash.md5: string (if available)
  - hash.sha256: string (if available)
  - is_dir: bool
  - Custom metadata prefixed with "meta."
```

### Object List Schema

Bucket object listings use this schema:

```
Schema: Object List
├── bucket: Utf8 (not null)
├── key: Utf8 (not null)
├── size: Int64
├── content_type: Utf8
├── etag: Utf8
├── version: Utf8
├── created: Timestamp[ns, UTC]
├── updated: Timestamp[ns, UTC]
├── is_dir: Boolean
└── metadata: Map<Utf8, Utf8>
```

### Bucket List Schema

```
Schema: Bucket List
├── name: Utf8 (not null)
├── created_at: Timestamp[ns, UTC]
├── public: Boolean
└── metadata: Map<Utf8, Utf8>
```

### Multipart Upload Schema

```
Schema: Part List
├── number: Int32 (not null)
├── size: Int64
├── etag: Utf8
└── last_modified: Timestamp[ns, UTC]
```

## RPC Methods

### GetFlightInfo

Retrieves metadata about an object or bucket.

**Request**: `FlightDescriptor`
- PATH descriptor: `["bucket", "key"]` for object info
- PATH descriptor: `["bucket"]` for bucket info

**Response**: `FlightInfo`
- Schema with object metadata
- Single endpoint for data retrieval
- Total bytes and records

### ListFlights

Lists objects or buckets.

**Request**: `Criteria` (JSON encoded)
```json
{
  "bucket": "my-bucket",
  "prefix": "path/",
  "limit": 100,
  "offset": 0,
  "recursive": true
}
```

For bucket listing, omit "bucket" field.

**Response**: Stream of `FlightInfo`
- One FlightInfo per object/bucket
- Includes schema and endpoints

### DoGet

Downloads object data as Arrow RecordBatches.

**Request**: `Ticket` (JSON encoded)
```json
{
  "bucket": "my-bucket",
  "key": "path/to/object",
  "offset": 0,
  "length": -1,
  "options": {}
}
```

**Response**: Stream of `FlightData`
- First message contains schema
- Subsequent messages contain record batches with binary data chunks
- App metadata includes progress information

### DoPut

Uploads object data.

**Request**: Stream of `FlightData`
- First message: FlightDescriptor with upload metadata
  ```json
  {
    "bucket": "my-bucket",
    "key": "path/to/object",
    "size": 1024,
    "content_type": "application/octet-stream",
    "options": {}
  }
  ```
- Subsequent messages: RecordBatches with binary data

**Response**: Stream of `PutResult`
- Final result contains object metadata as app_metadata

### DoExchange

Bidirectional streaming for complex operations.

Used for:
- Multipart uploads with immediate acknowledgment
- Range reads with dynamic requests
- Server-side transformations

**Request/Response**: Bidirectional `FlightData` stream

### DoAction

Executes administrative operations.

**Supported Actions**:

| Action Type            | Body (JSON)                    | Result                |
|------------------------|--------------------------------|-----------------------|
| `CreateBucket`         | `{name, options}`              | BucketInfo            |
| `DeleteBucket`         | `{name, options}`              | Empty                 |
| `DeleteObject`         | `{bucket, key, options}`       | Empty                 |
| `CopyObject`           | `{src_bucket, src_key, dst_bucket, dst_key, options}` | ObjectInfo |
| `MoveObject`           | `{src_bucket, src_key, dst_bucket, dst_key, options}` | ObjectInfo |
| `InitMultipart`        | `{bucket, key, content_type, options}` | MultipartUpload |
| `CompleteMultipart`    | `{bucket, key, upload_id, parts, options}` | ObjectInfo |
| `AbortMultipart`       | `{bucket, key, upload_id, options}` | Empty |
| `SignedURL`            | `{bucket, key, method, expires, options}` | `{url}` |
| `GetFeatures`          | `{bucket}`                     | Features              |
| `Stat`                 | `{bucket, key, options}`       | ObjectInfo            |

### ListActions

Returns all supported action types.

**Response**: Stream of `ActionType`
- Type name and description for each action

## Error Handling

Flight status codes map to storage errors:

| Storage Error           | Flight Status Code     |
|-------------------------|------------------------|
| `storage.ErrNotExist`   | `NOT_FOUND`            |
| `storage.ErrExist`      | `ALREADY_EXISTS`       |
| `storage.ErrPermission` | `PERMISSION_DENIED`    |
| `storage.ErrUnsupported`| `UNIMPLEMENTED`        |
| Context canceled        | `CANCELLED`            |
| Context deadline        | `DEADLINE_EXCEEDED`    |
| Invalid argument        | `INVALID_ARGUMENT`     |
| Other errors            | `INTERNAL`             |

Error details are provided in the status description and may include JSON-encoded additional context.

## Authentication

### Token-Based Authentication (Recommended)

Uses Flight's built-in Handshake mechanism:

1. Client sends username/password or initial credentials
2. Server validates and returns bearer token
3. Client includes token in subsequent requests via gRPC metadata

```go
type AuthConfig struct {
    // TokenValidator validates bearer tokens and returns claims.
    TokenValidator func(token string) (map[string]any, error)

    // BasicAuthValidator validates username/password and returns token.
    BasicAuthValidator func(username, password string) (string, error)

    // AllowUnauthenticated permits requests without tokens.
    AllowUnauthenticated bool
}
```

### Middleware Authentication

Uses Flight server middleware for header-based auth:

```go
// Server-side
middleware := CreateAuthMiddleware(authConfig)
server := flight.NewServerWithMiddleware([]flight.ServerMiddleware{middleware})

// Client-side
client, err := flight.NewClientWithMiddleware(addr, authHandler, middlewares, opts...)
```

### mTLS Authentication

Configure gRPC TLS credentials for mutual TLS:

```go
// Server
tlsConfig := &tls.Config{
    Certificates: []tls.Certificate{serverCert},
    ClientAuth:   tls.RequireAndVerifyClientCert,
    ClientCAs:    clientCAPool,
}
creds := credentials.NewTLS(tlsConfig)
server.Init(addr, grpc.Creds(creds))

// Client
tlsConfig := &tls.Config{
    Certificates: []tls.Certificate{clientCert},
    RootCAs:      serverCAPool,
}
opts := grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))
```

## Configuration

### Server Configuration

```go
type Config struct {
    // Addr is the listen address (e.g., ":8080").
    Addr string

    // TLS configures TLS. Nil uses plaintext.
    TLS *tls.Config

    // Auth configures authentication. Nil disables auth.
    Auth *AuthConfig

    // MaxRecvMsgSize is the max message size in bytes. Default 64MB.
    MaxRecvMsgSize int

    // MaxSendMsgSize is the max message size in bytes. Default 64MB.
    MaxSendMsgSize int

    // ChunkSize for streaming data. Default 1MB.
    // Larger chunks improve throughput for large objects.
    ChunkSize int

    // MaxConcurrentStreams limits parallel streams per connection.
    MaxConcurrentStreams uint32

    // Logger for server events.
    Logger *slog.Logger

    // Allocator for Arrow memory. Default uses Go allocator.
    Allocator memory.Allocator
}
```

### Client Configuration

```go
type ClientConfig struct {
    // Target is the Flight server address.
    Target string

    // TLS configures TLS. Nil uses plaintext.
    TLS *tls.Config

    // Token for bearer authentication.
    Token string

    // Username/Password for basic auth handshake.
    Username string
    Password string

    // MaxRecvMsgSize is the max message size in bytes. Default 64MB.
    MaxRecvMsgSize int

    // MaxSendMsgSize is the max message size in bytes. Default 64MB.
    MaxSendMsgSize int

    // ChunkSize for streaming writes. Default 1MB.
    ChunkSize int

    // DialOptions are additional gRPC dial options.
    DialOptions []grpc.DialOption

    // Allocator for Arrow memory. Default uses Go allocator.
    Allocator memory.Allocator
}
```

## Streaming Protocol

### Object Download (DoGet)

```
Client                              Server
  │                                    │
  │──── DoGet(Ticket) ────────────────►│
  │                                    │
  │◄──── FlightData (schema) ─────────│
  │◄──── FlightData (batch 1) ────────│
  │◄──── FlightData (batch 2) ────────│
  │◄──── FlightData (batch N) ────────│
  │◄──── FlightData (EOS) ────────────│
  │                                    │
```

Each batch contains a Binary column with a chunk of object data (default 1MB chunks).

### Object Upload (DoPut)

```
Client                              Server
  │                                    │
  │──── FlightData (descriptor) ──────►│
  │──── FlightData (batch 1) ─────────►│
  │──── FlightData (batch 2) ─────────►│
  │──── FlightData (batch N) ─────────►│
  │──── FlightData (EOS) ─────────────►│
  │                                    │
  │◄──── PutResult (metadata) ────────│
  │                                    │
```

### Multipart Upload (DoExchange)

```
Client                              Server
  │                                    │
  │──── FlightData (init multipart) ──►│
  │◄──── FlightData (upload_id) ──────│
  │                                    │
  │──── FlightData (part 1 data) ─────►│
  │◄──── FlightData (part 1 info) ────│
  │                                    │
  │──── FlightData (part 2 data) ─────►│
  │◄──── FlightData (part 2 info) ────│
  │                                    │
  │──── FlightData (complete) ────────►│
  │◄──── FlightData (object info) ────│
  │                                    │
```

## URI Schemes

| Transport        | Scheme             |
|------------------|-------------------|
| Plaintext gRPC   | `grpc://`         |
| TLS gRPC         | `grpc+tls://`     |
| Unix socket      | `grpc+unix://`    |
| Reuse connection | `arrow-flight-reuse-connection://?` |

## Client Implementation

The client implements `storage.Storage` interface:

```go
// Open connects to a Flight storage server.
func Open(ctx context.Context, target string, opts ...ClientOption) (storage.Storage, error)

// Example usage:
store, err := flight.Open(ctx, "grpc://localhost:8080",
    flight.WithToken("bearer-token"),
    flight.WithTLS(tlsConfig),
    flight.WithChunkSize(2 * 1024 * 1024),
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
// New creates a new Flight server.
func New(store storage.Storage, cfg *Config) *Server

// Example usage:
store, _ := storage.Open(ctx, "local:///data")
server := flight.New(store, &flight.Config{
    Addr: ":8080",
    Auth: &flight.AuthConfig{
        TokenValidator: validateToken,
    },
})

if err := server.Serve(); err != nil {
    log.Fatal(err)
}
```

## Performance Considerations

### Chunk Size

- Default 1MB balances memory usage and throughput
- Increase for large objects (10MB+) to reduce overhead
- Decrease for memory-constrained environments or small objects

### Memory Management

The transport uses Arrow's memory allocator:

```go
// Use custom allocator for memory tracking
allocator := memory.NewCheckedAllocator(memory.DefaultAllocator)
cfg := &Config{Allocator: allocator}
```

### Parallel Endpoints

For large objects, servers may return multiple endpoints in FlightInfo:

```go
info, _ := client.GetFlightInfo(ctx, descriptor)
for _, endpoint := range info.Endpoint {
    // Can fetch each endpoint in parallel
    go fetchEndpoint(endpoint)
}
```

### Zero-Copy Optimization

When possible, the transport avoids copies:
- Direct buffer references in Arrow IPC
- Memory-mapped data sources
- Pooled buffer allocation

### Compression

Enable Arrow compression for compressible content:

```go
writer := flight.NewRecordWriter(stream,
    ipc.WithCompressConcurrency(4),
    ipc.WithLZ4())
```

## Directory Structure

```
lib/storage/transport/flight/
├── SPEC.md                 # This specification
├── server.go               # Flight server implementation
├── server_test.go          # Server tests
├── client.go               # Flight client (storage.Storage impl)
├── client_test.go          # Client tests
├── auth.go                 # Authentication handlers
├── convert.go              # Type conversion utilities
├── errors.go               # Error mapping utilities
└── schema.go               # Arrow schema definitions
```

## Compatibility

- Apache Arrow Go v18+
- gRPC 1.50+
- Go 1.21+

## Comparison with gRPC Transport

| Feature              | Flight Transport        | gRPC Transport          |
|----------------------|-------------------------|-------------------------|
| Protocol             | Arrow IPC + gRPC        | Protobuf + gRPC         |
| Schema               | Dynamic Arrow schemas   | Static protobuf schemas |
| Data serialization   | Arrow IPC (columnar)    | Protobuf (row-based)    |
| Zero-copy            | Yes (when possible)     | No                      |
| Ecosystem            | Arrow tools compatible  | gRPC tools compatible   |
| Metadata             | Schema metadata         | Protobuf messages       |
| Streaming            | Native Arrow streams    | gRPC streams            |
| Compression          | Arrow compression       | gRPC compression        |
| Max message size     | 64MB default            | 16MB default            |

## Use Cases

### Best For

- Large object transfers (>1MB)
- Integration with Arrow-based analytics tools
- High-throughput data pipelines
- Columnar data processing

### Consider gRPC Transport For

- Small objects with frequent metadata operations
- Systems already using Protocol Buffers
- Lower memory footprint requirements
- Simpler debugging (human-readable protos)

## Future Considerations

- Poll-based long-running operations (PollFlightInfo)
- Session management for stateful operations
- Endpoint renewal for long transfers
- Flight SQL compatibility for query operations
- Watch/notification streaming
- Server-side filtering and projection
