# 0552: LiteIO Benchmark System

## Overview

The benchmark system (`liteio/bench/`) provides comprehensive performance testing for S3-compatible storage backends. It supports both Go's `testing.B` style benchmarks and a standalone CLI runner for controlled comparison testing.

## Architecture

### Files

| File | Lines | Purpose |
|------|-------|---------|
| `bench_test.go` | ~1400 | Go `testing.B` benchmarks with auto-detection of available backends |
| `config.go` | ~377 | Configuration, driver configs, all tuning constants |
| `runner.go` | ~1950 | Standalone CLI runner with adaptive iteration scaling |
| `metrics.go` | ~297 | Latency percentiles, throughput, TTFB collection |
| `docker.go` | ~623 | Docker container stats (CPU, memory, block I/O, volume size) |
| `report.go` | ~2221 | Markdown/JSON/CSV report generation with charts |
| `profiler.go` | ~328 | pprof integration (CPU, heap, goroutine, mutex profiles) |
| `progress.go` | ~167 | Live progress bar with ETA and throughput display |

### Test Modes

**1. Go Benchmark Mode (`go test -bench`)**

Standard Go benchmarks using `testing.B`. Auto-detects available backends by probing ports:

```
go test -bench=. -benchmem -benchtime=3s ./bench/...
```

Auto-detected backends:
- `memory` — in-process memory driver (always available)
- `local` — in-process filesystem driver (always available)
- `minio` — port 9000 (Docker)
- `rustfs` — port 9100 (Docker)
- `seaweedfs` — port 8333 (Docker)
- `localstack` — port 4566 (Docker)
- `liteio` — port 9200 (Docker)
- `rabbit_s3` — port 9300 (Docker)
- `usagi_s3` — port 9301 (Docker)
- `devnull_s3` — port 9302 (Docker)

**2. CLI Runner Mode**

Standalone runner with Go-style adaptive iteration scaling:

```go
runner := bench.NewRunner(bench.DefaultConfig())
runner.Run(context.Background())
```

Features:
- Adaptive iteration scaling (`predictN` algorithm, same as Go's `testing.B`)
- Warmup phase with adaptive iteration count per object size
- Docker container stats collection between runs
- Live progress display with spinner and ops/sec
- Container cleanup between drivers

### Benchmark Suite

| Benchmark | Description |
|-----------|-------------|
| **Write** | Sequential writes at various sizes (1KB–100MB) |
| **Read** | Sequential reads at various sizes |
| **RangeRead** | Partial reads with byte range offsets |
| **Stat** | Object metadata queries (HEAD) |
| **Delete** | Sequential deletes |
| **Copy** | Server-side object copy |
| **List** | List objects with prefix filtering |
| **ParallelWrite** | Concurrent writes at [1, 10, 25, 50, 100, 200] concurrency |
| **ParallelRead** | Concurrent reads at multiple concurrency levels |
| **MixedWorkload** | 40% write, 40% read, 20% stat interleaved |
| **Multipart** | Multipart upload (10MB, 5 parts) |
| **EdgeCases** | Empty objects, special characters, large keys |
| **Metadata** | User metadata read/write |
| **BucketOps** | Create/delete/head bucket operations |
| **Scale** | Object count scaling at [10, 100, 1000, 10000] |

### Object Sizes

| Label | Size | Use Case |
|-------|------|----------|
| Tiny | 256B | Metadata-like objects |
| Small | 1KB | Config files, small JSON |
| Medium | 64KB | Typical web assets |
| Large | 1MB | Images, documents |
| XLarge | 10MB | Videos, archives |
| XXLarge | 100MB | Large media (opt-in via `--large`) |

### Configuration

```go
type Config struct {
    BenchTime          time.Duration // Target duration per benchmark (default: 1s)
    WarmupIterations   int          // Warmup iterations before timing (default: 10)
    Concurrency        int          // Default parallel concurrency (default: 200)
    ConcurrencyLevels  []int        // Levels to test: [1, 10, 25, 50, 100, 200]
    ObjectSizes        []int        // Sizes to benchmark: [1KB, 64KB, 1MB, 10MB, 100MB]
    DockerStats        bool         // Collect Docker container stats (default: true)
    LowOverhead        bool         // Minimize client-side overhead (default: true)
    EnableTTFB         bool         // Capture time-to-first-byte
    ScaleCounts        []int        // Object counts: [10, 100, 1000, 10000]
    ScaleMaxBytes      int64        // Safety cap: 2GB
    CleanupDataPaths   bool         // Remove local data after each driver
    CleanupDockerData  bool         // Clear Docker volumes after each driver
}
```

### Adaptive Iteration Scaling

The runner implements Go's `predictN` algorithm:

1. Start with `n = 1` iteration
2. Time `n` iterations
3. If elapsed < `BenchTime`, compute `predictN = n * BenchTime / elapsed`
4. Apply safety bounds: `predictN` ≤ `100 * n` and ≤ `MaxBenchIterations`
5. Repeat until elapsed ≥ `BenchTime`

This ensures statistically significant results while keeping wall-clock time bounded.

### Metrics Collection

```go
type Metrics struct {
    Operation  string
    Driver     string
    ObjectSize int
    Iterations int
    Duration   time.Duration
    P50, P95, P99 time.Duration  // Latency percentiles
    Throughput    float64         // MB/s
    OpsPerSec     float64         // Operations per second
    TTFB          TTFBMetrics     // Time to first byte (optional)
    DockerStats   DockerStats     // Container resource usage
}
```

### Docker Integration

The `DockerStatsCollector` captures:
- **Memory**: RSS, cache, working set
- **CPU**: User/system time, percentage
- **Block I/O**: Read/write bytes
- **Volume Size**: Data path size inside container
- **PIDs**: Process count

Between driver runs, containers are restarted to ensure clean state.

### Report Generation

Reports are generated in markdown, JSON, and CSV formats:
- Executive summary with performance leaders
- Per-operation comparison tables with ASCII bar charts
- Throughput and latency visualizations
- Docker resource usage comparisons
- Baseline regression detection (optional)

## Docker Setup

### docker-compose.yaml

Located at `liteio/docker/s3/all/docker-compose.yaml`:

| Service | Port | Credentials | Language |
|---------|------|-------------|----------|
| MinIO | 9000/9001 | minioadmin/minioadmin | Go |
| RustFS | 9100/9101 | rustfsadmin/rustfsadmin | Rust |
| SeaweedFS | 8333 | admin/adminpassword | Go |
| LocalStack | 4566 | test/test | Python |
| LiteIO | 9200 | liteio/liteio123 | Go |
| LiteIO (mem) | 9201 | liteio/liteio123 | Go |
| Rabbit S3 | 9300 | rabbit/rabbit123 | Go |
| Usagi S3 | 9301 | usagi/usagi123 | Go |
| Devnull S3 | 9302 | devnull/devnull123 | Go |

Each service includes:
- Health check with retry (5s interval, 10 retries)
- Init container to create `test-bucket` using AWS CLI
- Named volume for data persistence

### LiteIO Dockerfile

Two-stage build: `golang:1.25-alpine` → `scratch`
- `CGO_ENABLED=0` static binary
- Runs as `liteio --port 9000 --host 0.0.0.0 --data-dir /data`

## Running Benchmarks

### Quick Start

```bash
# Start all S3 services
cd liteio/docker/s3/all
docker compose up -d

# Wait for all services to be healthy
docker compose ps

# Run benchmarks (all detected backends)
cd liteio
go test -bench=. -benchtime=3s ./bench/...

# Run specific benchmark
go test -bench=BenchmarkWrite -benchtime=5s ./bench/...

# Run only MinIO and LiteIO
go test -bench=. -benchtime=3s -run="minio|liteio" ./bench/...
```

### CLI Runner

```go
cfg := bench.DefaultConfig()
cfg.Drivers = []string{"liteio", "minio"} // Compare these two
cfg.BenchTime = 3 * time.Second
cfg.DockerStats = true

runner := bench.NewRunner(cfg)
results, err := runner.Run(context.Background())
```

## LiteIO vs MinIO Performance Profile

### Where LiteIO Wins (Expected)

1. **Small Object Writes (≤128KB)**: LiteIO uses `os.WriteFile` (single syscall) with NoFsync mode. MinIO has multi-layer overhead (erasure coding, metadata, etc.)

2. **Small Object Reads (≤64KB)**: Hot object cache + zero-copy readers serve from memory. MinIO reads from disk every time.

3. **Stat/HEAD Operations**: LiteIO serves from object cache when available. MinIO always hits metadata storage.

4. **Delete Operations**: LiteIO uses `unlinkat` syscall directly. MinIO has erasure coding cleanup overhead.

5. **Single-Node Throughput**: LiteIO is purpose-built for single-node local storage. MinIO is designed for distributed multi-node.

### Where MinIO Wins (Expected)

1. **Very Large Files (≥100MB)**: MinIO's erasure coding can parallelize across disks. LiteIO's parallel write helps but single disk is the bottleneck.

2. **High Concurrency (200+)**: MinIO's connection handling is battle-tested at scale. LiteIO may have lock contention in directory cache or buffer pools.

3. **List Operations**: MinIO maintains an indexed metadata layer. LiteIO walks the filesystem.

### Key Bottlenecks in LiteIO

1. **S3 Transport Layer**: SigV4 verification on every request (HMAC-SHA256 computation)
2. **XML Serialization**: `encoding/xml` for list responses
3. **Directory Cache Contention**: Single `sync.RWMutex` under high concurrency
4. **Buffer Pool Contention**: Even sharded pools have contention at 200+ goroutines
5. **Filesystem Metadata**: `os.Stat` calls for every read operation
