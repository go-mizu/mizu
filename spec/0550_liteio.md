# 0550: LiteIO — Lightweight S3-Compatible Object Storage

## Overview

LiteIO is a self-contained, drop-in replacement for Amazon S3 / MinIO for local development, testing, and edge deployments. It provides 100% S3 API compatibility for the most commonly used operations, with multiple pluggable storage backends and extreme performance optimizations.

**Module:** `github.com/liteio-dev/liteio`
**Blueprint:** `blueprints/liteio/`
**Origin:** Extracted from localflare's `pkg/storage/` + `cmd/liteio/`

## Goals

1. **100% S3 API compatible** — works with AWS SDK, `aws` CLI, `mc`, `s3cmd`, boto3, any S3 client
2. **Zero external dependencies at runtime** — single static binary, no database, no config files
3. **Multiple storage drivers** — local filesystem, in-memory, rabbit (high-perf), usagi (append-log), devnull (benchmark)
4. **Sub-millisecond latency** — tiered caching, zero-copy I/O, platform-specific optimizations
5. **Docker-first** — 10MB scratch-based image, single `docker run` command

## Architecture

```
┌──────────────────────────────────────────────────────┐
│                   cmd/liteio/main.go                  │
│              Cobra CLI + signal handling               │
├──────────────────────────────────────────────────────┤
│              pkg/storage/server/server.go             │
│          HTTP server + pprof + healthcheck            │
├──────────────────────────────────────────────────────┤
│            pkg/storage/transport/s3/                  │
│   S3 API (SigV4 auth, XML responses, range reads)    │
├──────────────────────────────────────────────────────┤
│               pkg/storage/storage.go                  │
│        Storage / Bucket / Object interfaces           │
├──────────┬──────────┬─────────┬─────────┬────────────┤
│  local   │  memory  │ rabbit  │ usagi   │  devnull   │
│  driver  │  driver  │ driver  │ driver  │  driver    │
└──────────┴──────────┴─────────┴─────────┴────────────┘
```

### Component Breakdown

#### 1. CLI (`cmd/liteio/main.go`)
- Cobra-based CLI with root command (starts server) and `healthcheck` subcommand
- Configuration via flags and environment variables (`LITEIO_*`)
- Graceful shutdown on SIGTERM/SIGINT
- Build variables injected via ldflags (Version, Commit, BuildTime)

#### 2. Server (`pkg/storage/server/`)
- Wraps S3 transport + storage into a complete HTTP server
- Registers all storage drivers via blank imports
- Health endpoint at `/healthz/ready`
- Optional pprof endpoints at `/debug/pprof/*`
- Path normalization for trailing slashes (preserving original for SigV4)
- Configurable timeouts (read: 60s, write: 60s, idle: 120s)

#### 3. S3 Transport (`pkg/storage/transport/s3/`)
- Full AWS Signature V4 authentication with signing key cache
- Server-side response cache (256 shards, 256MB max, 128KB/item)
- Hand-optimized XML response generation (no reflection)
- Pooled I/O buffers (8MB streaming pool, XML buffer pool)

#### 4. Storage Interfaces (`pkg/storage/`)
- `Storage` — root backend (bucket management)
- `Bucket` — object CRUD operations
- `HasMultipart` — optional multipart upload support
- `HasDirectories` — optional directory operations
- `Driver` — plugin registry for storage backends
- `Options` / `Features` / `Hashes` — extensible maps

#### 5. Storage Drivers (`pkg/storage/driver/`)

| Driver | Scheme | Description | Persistence | Use Case |
|--------|--------|-------------|-------------|----------|
| `local` | `local://`, `file://` | Filesystem-backed | Yes | Default, production |
| `memory` | `mem://`, `memory://` | Pure in-memory | No | Testing, benchmarks |
| `rabbit` | `rabbit://` | High-perf filesystem | Yes | Performance-critical |
| `usagi` | `usagi://` | Append-log segments | Yes | Write-heavy workloads |
| `devnull` | `devnull://` | No-op (discard) | No | Benchmark baseline |

## S3 API Coverage

### Bucket Operations
| Operation | Method | Route | Status |
|-----------|--------|-------|--------|
| ListBuckets | `GET /` | `/` | Implemented |
| CreateBucket | `PUT /{bucket}` | `/{bucket}` | Implemented |
| DeleteBucket | `DELETE /{bucket}` | `/{bucket}` | Implemented |
| HeadBucket | `HEAD /{bucket}` | `/{bucket}` | Implemented |
| GetBucketLocation | `GET /{bucket}?location` | `/{bucket}` | Implemented |

### Object Operations
| Operation | Method | Route | Status |
|-----------|--------|-------|--------|
| GetObject | `GET /{bucket}/{key}` | `/{bucket}/{key...}` | Implemented (range support) |
| PutObject | `PUT /{bucket}/{key}` | `/{bucket}/{key...}` | Implemented |
| CopyObject | `PUT /{bucket}/{key}` + `x-amz-copy-source` | `/{bucket}/{key...}` | Implemented |
| DeleteObject | `DELETE /{bucket}/{key}` | `/{bucket}/{key...}` | Implemented (S3 204 semantics) |
| HeadObject | `HEAD /{bucket}/{key}` | `/{bucket}/{key...}` | Implemented |
| DeleteObjects | `POST /{bucket}?delete` | `/{bucket}` | Implemented (batch, max 1000) |
| ListObjectsV2 | `GET /{bucket}` | `/{bucket}` | Implemented (prefix, delimiter, pagination) |

### Multipart Upload Operations
| Operation | Method | Route | Status |
|-----------|--------|-------|--------|
| CreateMultipartUpload | `POST /{bucket}/{key}?uploads` | `/{bucket}/{key...}` | Implemented |
| UploadPart | `PUT /{bucket}/{key}?partNumber=N&uploadId=ID` | `/{bucket}/{key...}` | Implemented |
| ListParts | `GET /{bucket}/{key}?uploadId=ID` | `/{bucket}/{key...}` | Implemented |
| CompleteMultipartUpload | `POST /{bucket}/{key}?uploadId=ID` | `/{bucket}/{key...}` | Implemented |
| AbortMultipartUpload | `DELETE /{bucket}/{key}?uploadId=ID` | `/{bucket}/{key...}` | Implemented |
| ListMultipartUploads | `GET /{bucket}?uploads` | `/{bucket}` | Implemented |

### Authentication
- AWS Signature V4 (Authorization header)
- AWS Signature V4 (Presigned URLs with `X-Amz-Signature`)
- Configurable time skew tolerance (default 15 minutes)
- Signing key cache (keyed by credential prefix + date + region + service)

### Response Features
- Proper XML error responses matching S3 format
- S3 error codes: `AccessDenied`, `NoSuchBucket`, `NoSuchKey`, `BucketNotEmpty`, `InvalidRequest`, `EntityTooLarge`, `InternalError`, `NotImplemented`, `InvalidPart`, `InvalidPartOrder`, `NoSuchUpload`
- `Content-Range` header for partial reads
- `Accept-Ranges: bytes` for all objects
- `ETag` headers on all mutations
- URL encoding support (`encoding-type=url`)
- `x-amz-meta-*` custom metadata headers

## Storage Driver Details

### Local Driver (`driver/local/`)
**Write strategy by size:**
| Size | Strategy | Details |
|------|----------|---------|
| 0 bytes | `writeEmptyFile` | Single syscall |
| ≤8KB | `writeTinyFile` | Sharded pool, direct write |
| 8KB–128KB | `writeSmallFile` | Exact-size buffer selection |
| ≥32MB | `writeVeryLargeFile` | Parallel chunked `ParallelWriter` |
| Default | `writeLargeFile` | Temp file + atomic rename |

**Read optimizations:**
- Hot cache: lock-free atomic ring, zero-copy
- Object cache: 64-shard LRU, 256MB max, 128KB/item, lazy LRU update
- mmap: 64KB–1MB files (Unix/Windows)
- Buffered reader: 1MB–32MB
- Streaming reader: ≥32MB
- Platform sendfile: Darwin (`sendfile`), Linux (`sendfile`, `copy_file_range`)

**Special modes:**
- `EnableInMemoryMode()`: 512-shard in-memory hash map (bypass filesystem)
- `NoFsync = true`: Skip fsync for benchmark performance

### Memory Driver (`driver/memory/`)
- `sync.Map` per bucket for lock-free reads/writes
- 256-shard key index for sorted `List`
- Tiered buffer pools: 4KB, 64KB, 256KB, 1MB
- Entry object pool for small object reuse
- 10ms time cache (batched `time.Now()`)

### Rabbit Driver (`driver/rabbit/`)
- HotCache (L1) + WarmCache (L2) tiered caching
- `sync.Map` bucket registry (lock-free)
- Sharded key index per bucket
- DSN params: `?nofsync=true`

### Usagi Driver (`driver/usagi/`)
- Append-only segment files (`data.usagi`)
- Configurable segment size, shard count, manifest interval
- Small object cache
- DSN params: `?segment_size_mb=N&segment_shards=N&manifest_interval_s=N`

## Configuration

### CLI Flags
| Flag | Default | Description |
|------|---------|-------------|
| `--port, -p` | 9000 | Port to listen on |
| `--host` | 0.0.0.0 | Host to bind to |
| `--data-dir, -d` | `$HOME/data/liteio` | Data directory (local driver) |
| `--driver` | — | Storage driver DSN (overrides data-dir) |
| `--access-key` | liteio | S3 access key ID |
| `--secret-key` | liteio123 | S3 secret access key |
| `--region` | us-east-1 | S3 region |
| `--pprof` | true | Enable pprof endpoints |

### Environment Variables
| Variable | Description |
|----------|-------------|
| `LITEIO_PORT` | Port (overrides default) |
| `LITEIO_HOST` | Host (overrides default) |
| `LITEIO_DATA_DIR` | Data directory (converted to `local://` DSN) |
| `LITEIO_DRIVER` | Full driver DSN (overrides data-dir) |
| `LITEIO_ACCESS_KEY` | Access key ID |
| `LITEIO_SECRET_KEY` | Secret access key |
| `LITEIO_REGION` | S3 region |
| `LITEIO_IN_MEMORY` | Set `true` to enable in-memory mode for local driver |
| `LITEIO_NO_FSYNC` | Set `true` to skip fsync (benchmark mode) |

## Project Structure

```
liteio/
├── cmd/liteio/main.go          # CLI entry point
├── pkg/storage/
│   ├── storage.go              # Core interfaces (Storage, Bucket, Object)
│   ├── driver.go               # Driver registry
│   ├── multipart.go            # Multipart upload interfaces
│   ├── server/
│   │   └── server.go           # HTTP server wrapper
│   ├── transport/s3/
│   │   ├── server.go           # S3 route registration + SigV4 auth
│   │   ├── handle_bucket.go    # Bucket operations
│   │   ├── handle_object.go    # Object operations
│   │   ├── handle_multipart.go # Multipart operations
│   │   ├── response.go         # Fast XML response helpers
│   │   ├── response_cache.go   # 256-shard response cache
│   │   ├── debug.go            # Debug logging (build tag)
│   │   └── debug_on.go         # Debug logging implementation
│   └── driver/
│       ├── local/              # Filesystem driver (default)
│       ├── memory/             # In-memory driver
│       ├── rabbit/             # High-performance filesystem driver
│       ├── usagi/              # Append-log driver
│       └── devnull/            # No-op benchmark driver
├── go.mod
├── Makefile
├── Dockerfile
└── README.md
```

## Build & Run

```bash
# Build binary to $HOME/bin/liteio
make build

# Run directly
make run

# Run with memory driver
make run-memory

# Run tests
make test

# Docker
make docker
make docker-run
```

## Dependencies

**Required:**
- `github.com/go-mizu/mizu` — HTTP framework (routing, context, middleware)
- `github.com/spf13/cobra` — CLI framework

**Optional (test only):**
- `github.com/aws/aws-sdk-go-v2` — Used in integration tests

**Platform-specific:**
- `github.com/edsrzf/mmap-go` — Memory-mapped file I/O
- `golang.org/x/sys` — Unix syscalls (sendfile, copy_file_range)

## Not Implemented (vs full MinIO)

These S3 operations are not implemented in v1:
- Bucket versioning / object lock / legal hold
- Bucket lifecycle / replication / notification
- Bucket policy / ACL (beyond static credentials)
- Server-side encryption (SSE-S3, SSE-KMS, SSE-C)
- Object tagging API (`?tagging`)
- Select Object Content (S3 Select)
- Torrent (`?torrent`)
- Bucket analytics / metrics / inventory
- STS (Security Token Service)
- IAM (Identity and Access Management)

These may be added in future versions as needed.
