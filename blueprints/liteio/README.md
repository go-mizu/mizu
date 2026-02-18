# LiteIO

Lightweight, S3-compatible object storage server. Single binary, zero config, drop-in replacement for Amazon S3 / MinIO in local development and testing.

```
╭─────────────────────────────────────────────────────────────╮
│                       LiteIO Server                        │
├─────────────────────────────────────────────────────────────┤
│  Endpoint:     http://0.0.0.0:9000                         │
│  Region:       us-east-1                                   │
│  Access Key:   liteio                                      │
│  Secret Key:   li****23                                    │
╰─────────────────────────────────────────────────────────────╯
```

## Features

- **100% S3 API compatible** — works with AWS CLI, AWS SDK, boto3, mc, s3cmd, any S3 client
- **Single static binary** — zero external dependencies, no database, no config files
- **5 storage drivers** — local filesystem, in-memory, rabbit (high-perf), usagi (append-log), devnull (benchmark)
- **AWS Signature V4** — full authentication with signing key cache
- **Multipart uploads** — create, upload parts, complete, abort, list parts
- **Range reads** — `bytes=start-end`, `bytes=start-`, `bytes=-suffix`
- **Batch deletes** — up to 1000 keys per request
- **Docker ready** — 10MB scratch-based image

## Quick Start

### From Source

```bash
# Build
make build

# Run (default: port 9000, data at ~/data/liteio)
liteio

# Custom port and data directory
liteio --port 8000 --data-dir /tmp/storage

# In-memory mode (ephemeral, data lost on restart)
liteio --driver "memory://"

# Custom credentials
liteio --access-key admin --secret-key supersecret
```

### Docker

```bash
# Build image
make docker

# Run
docker run -d --name liteio \
  -p 9000:9000 \
  -v ~/data/liteio:/data \
  liteio:latest

# Or use docker-compose
docker run -d \
  -p 9000:9000 \
  -e LITEIO_ACCESS_KEY=myadmin \
  -e LITEIO_SECRET_KEY=mysecret \
  -v liteio-data:/data \
  liteio:latest
```

## Usage with AWS CLI

```bash
# Configure
export AWS_ACCESS_KEY_ID=liteio
export AWS_SECRET_ACCESS_KEY=liteio123
export AWS_DEFAULT_REGION=us-east-1

# Create a bucket
aws --endpoint-url http://localhost:9000 s3 mb s3://my-bucket

# Upload a file
aws --endpoint-url http://localhost:9000 s3 cp myfile.txt s3://my-bucket/

# List objects
aws --endpoint-url http://localhost:9000 s3 ls s3://my-bucket/

# Download a file
aws --endpoint-url http://localhost:9000 s3 cp s3://my-bucket/myfile.txt ./downloaded.txt

# Delete a file
aws --endpoint-url http://localhost:9000 s3 rm s3://my-bucket/myfile.txt

# Sync a directory
aws --endpoint-url http://localhost:9000 s3 sync ./local-dir s3://my-bucket/prefix/

# Delete a bucket
aws --endpoint-url http://localhost:9000 s3 rb s3://my-bucket --force
```

## Usage with AWS SDK (Go)

```go
package main

import (
    "context"
    "fmt"
    "strings"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/credentials"
    "github.com/aws/aws-sdk-go-v2/service/s3"
)

func main() {
    cfg, _ := config.LoadDefaultConfig(context.TODO(),
        config.WithRegion("us-east-1"),
        config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
            "liteio", "liteio123", "",
        )),
    )

    client := s3.NewFromConfig(cfg, func(o *s3.Options) {
        o.BaseEndpoint = aws.String("http://localhost:9000")
        o.UsePathStyle = true
    })

    // Create bucket
    client.CreateBucket(context.TODO(), &s3.CreateBucketInput{
        Bucket: aws.String("my-bucket"),
    })

    // Put object
    client.PutObject(context.TODO(), &s3.PutObjectInput{
        Bucket: aws.String("my-bucket"),
        Key:    aws.String("hello.txt"),
        Body:   strings.NewReader("Hello, LiteIO!"),
    })

    // Get object
    out, _ := client.GetObject(context.TODO(), &s3.GetObjectInput{
        Bucket: aws.String("my-bucket"),
        Key:    aws.String("hello.txt"),
    })
    defer out.Body.Close()
    fmt.Println("Got object, size:", out.ContentLength)
}
```

## Usage with Python (boto3)

```python
import boto3

s3 = boto3.client('s3',
    endpoint_url='http://localhost:9000',
    aws_access_key_id='liteio',
    aws_secret_access_key='liteio123',
    region_name='us-east-1',
)

# Create bucket
s3.create_bucket(Bucket='my-bucket')

# Upload
s3.put_object(Bucket='my-bucket', Key='hello.txt', Body=b'Hello, LiteIO!')

# Download
obj = s3.get_object(Bucket='my-bucket', Key='hello.txt')
print(obj['Body'].read())

# List
response = s3.list_objects_v2(Bucket='my-bucket')
for obj in response.get('Contents', []):
    print(obj['Key'], obj['Size'])
```

## Usage with JavaScript (AWS SDK v3)

```javascript
import { S3Client, CreateBucketCommand, PutObjectCommand, GetObjectCommand } from "@aws-sdk/client-s3";

const client = new S3Client({
  endpoint: "http://localhost:9000",
  region: "us-east-1",
  credentials: { accessKeyId: "liteio", secretAccessKey: "liteio123" },
  forcePathStyle: true,
});

// Create bucket
await client.send(new CreateBucketCommand({ Bucket: "my-bucket" }));

// Put object
await client.send(new PutObjectCommand({
  Bucket: "my-bucket",
  Key: "hello.txt",
  Body: "Hello, LiteIO!",
}));

// Get object
const { Body } = await client.send(new GetObjectCommand({
  Bucket: "my-bucket",
  Key: "hello.txt",
}));
console.log(await Body.transformToString());
```

## Configuration

### CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--port, -p` | `9000` | Port to listen on |
| `--host` | `0.0.0.0` | Host to bind to |
| `--data-dir, -d` | `$HOME/data/liteio` | Data directory (local driver) |
| `--driver` | — | Storage driver DSN (overrides data-dir) |
| `--access-key` | `liteio` | S3 access key ID |
| `--secret-key` | `liteio123` | S3 secret access key |
| `--region` | `us-east-1` | S3 region |
| `--pprof` | `true` | Enable pprof profiling endpoints |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `LITEIO_PORT` | Port (overrides default) |
| `LITEIO_HOST` | Host (overrides default) |
| `LITEIO_DATA_DIR` | Data directory (converted to `local://` DSN) |
| `LITEIO_DRIVER` | Full driver DSN (overrides everything) |
| `LITEIO_ACCESS_KEY` | Access key ID |
| `LITEIO_SECRET_KEY` | Secret access key |
| `LITEIO_REGION` | S3 region |
| `LITEIO_IN_MEMORY` | `true` to enable in-memory mode for local driver |
| `LITEIO_NO_FSYNC` | `true` to skip fsync (benchmark mode, data may be lost on crash) |

## Storage Drivers

### Local (default)

Filesystem-backed storage. Each bucket is a directory, each object is a file.

```bash
liteio --data-dir /var/data/liteio
# or
liteio --driver "local:///var/data/liteio"
```

**Performance features:**
- Tiered write strategy (empty → tiny → small → large → very-large)
- Hot cache (lock-free atomic ring)
- Object cache (64-shard LRU, 256MB)
- mmap for medium files (64KB–1MB)
- Platform sendfile (macOS/Linux)
- Optional in-memory mode (`LITEIO_IN_MEMORY=true`)
- Optional NoFsync mode (`LITEIO_NO_FSYNC=true`)

### Memory

Pure in-memory storage. Data is lost on restart.

```bash
liteio --driver "memory://"
```

### Rabbit

High-performance filesystem driver with L1/L2 tiered caching.

```bash
liteio --driver "rabbit:///var/data/rabbit?nofsync=true"
```

### Usagi

Append-only log-structured storage with configurable segment files.

```bash
liteio --driver "usagi:///var/data/usagi?segment_size_mb=64&segment_shards=4"
```

### Devnull

No-op driver. All writes are discarded, all reads return zeros. Used for benchmarking infrastructure overhead.

```bash
liteio --driver "devnull://"
```

## S3 API Reference

### Bucket Operations

| Operation | Method | Path |
|-----------|--------|------|
| ListBuckets | `GET /` | `/` |
| CreateBucket | `PUT /{bucket}` | `PUT /my-bucket` |
| DeleteBucket | `DELETE /{bucket}` | `DELETE /my-bucket` |
| HeadBucket | `HEAD /{bucket}` | `HEAD /my-bucket` |
| GetBucketLocation | `GET /{bucket}?location` | `GET /my-bucket?location` |
| ListObjectsV2 | `GET /{bucket}` | `GET /my-bucket?prefix=foo/&delimiter=/&max-keys=100` |

### Object Operations

| Operation | Method | Path |
|-----------|--------|------|
| PutObject | `PUT /{bucket}/{key}` | `PUT /my-bucket/path/to/file.txt` |
| GetObject | `GET /{bucket}/{key}` | `GET /my-bucket/path/to/file.txt` |
| HeadObject | `HEAD /{bucket}/{key}` | `HEAD /my-bucket/path/to/file.txt` |
| DeleteObject | `DELETE /{bucket}/{key}` | `DELETE /my-bucket/path/to/file.txt` |
| CopyObject | `PUT /{bucket}/{key}` | `PUT /my-bucket/copy.txt` + `x-amz-copy-source: /src-bucket/src.txt` |
| DeleteObjects | `POST /{bucket}?delete` | Batch delete up to 1000 keys |

### Multipart Upload

| Operation | Method | Path |
|-----------|--------|------|
| CreateMultipartUpload | `POST /{bucket}/{key}?uploads` | Initiate upload |
| UploadPart | `PUT /{bucket}/{key}?partNumber=N&uploadId=ID` | Upload part (1–10000) |
| ListParts | `GET /{bucket}/{key}?uploadId=ID` | List uploaded parts |
| CompleteMultipartUpload | `POST /{bucket}/{key}?uploadId=ID` | Assemble final object |
| AbortMultipartUpload | `DELETE /{bucket}/{key}?uploadId=ID` | Cancel and cleanup |
| ListMultipartUploads | `GET /{bucket}?uploads` | List active uploads |

### Authentication

LiteIO supports AWS Signature Version 4:

```
Authorization: AWS4-HMAC-SHA256 Credential=liteio/20260218/us-east-1/s3/aws4_request, SignedHeaders=host;x-amz-date, Signature=...
```

Presigned URLs are also supported:

```
GET /my-bucket/file.txt?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=liteio/20260218/us-east-1/s3/aws4_request&X-Amz-Date=20260218T000000Z&X-Amz-Expires=3600&X-Amz-Signature=...
```

## Health Check

```bash
# HTTP health check
curl http://localhost:9000/healthz/ready
# {"status":"ok","server":"liteio"}

# CLI health check
liteio healthcheck --port 9000
```

## Profiling

When `--pprof` is enabled (default), the following endpoints are available:

```
http://localhost:9000/debug/pprof/
http://localhost:9000/debug/pprof/heap
http://localhost:9000/debug/pprof/goroutine
http://localhost:9000/debug/pprof/profile?seconds=30
http://localhost:9000/debug/pprof/trace?seconds=5
```

## Programmatic Usage

LiteIO can be embedded as a library:

```go
package main

import (
    "github.com/liteio-dev/liteio/pkg/storage/server"
    _ "github.com/liteio-dev/liteio/pkg/storage/driver/local"
    _ "github.com/liteio-dev/liteio/pkg/storage/driver/memory"
)

func main() {
    cfg := &server.Config{
        Port:            9000,
        DSN:             "memory://",
        AccessKeyID:     "test",
        SecretAccessKey: "test123",
    }

    srv, err := server.New(cfg)
    if err != nil {
        panic(err)
    }

    if err := srv.Start(); err != nil {
        panic(err)
    }
}
```

## Project Structure

```
liteio/
├── cmd/liteio/main.go              # CLI entry point
├── pkg/storage/
│   ├── storage.go                   # Core interfaces
│   ├── driver.go                    # Driver registry
│   ├── multipart.go                 # Multipart interfaces
│   ├── server/server.go             # HTTP server
│   ├── transport/s3/                # S3 protocol layer
│   │   ├── server.go                # Routes + SigV4 auth
│   │   ├── handle_bucket.go         # Bucket operations
│   │   ├── handle_object.go         # Object operations
│   │   ├── handle_multipart.go      # Multipart operations
│   │   ├── response.go              # XML response helpers
│   │   └── response_cache.go        # Response cache
│   └── driver/
│       ├── local/                   # Filesystem driver
│       ├── memory/                  # In-memory driver
│       ├── rabbit/                  # High-perf driver
│       ├── usagi/                   # Append-log driver
│       └── devnull/                 # No-op driver
├── go.mod
├── Makefile
├── Dockerfile
└── README.md
```

## Make Targets

```bash
make build        # Build binary to $HOME/bin/liteio
make install      # Alias for build
make run          # Run server with defaults
make run-memory   # Run with memory driver
make test         # Run all tests
make test-v       # Run tests with verbose output
make bench        # Run benchmarks
make docker       # Build Docker image
make docker-run   # Run in Docker
make docker-stop  # Stop Docker container
make tidy         # go mod tidy
make update       # Update dependencies
make help         # Show all targets
```

## License

MIT
