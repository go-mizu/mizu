# 0564: Cluster S3 Benchmark — MinIO vs RustFS vs SeaweedFS vs Herd

## Objective

Benchmark four S3-compatible storage systems in 3-node cluster mode to compare throughput, latency, and concurrency characteristics. All systems accessed exclusively via S3 HTTP API for fair comparison.

## Systems Under Test

| System | Version | Architecture | Cluster Mode | S3 Port |
|--------|---------|-------------|-------------|---------|
| **MinIO** | latest | Erasure coding, distributed hash | 3 nodes, 2 dirs/node (6 drives total) | 9000 |
| **RustFS** | latest | MinIO-compatible Rust rewrite | 3 nodes, 2 dirs/node (6 drives total) | 9100 |
| **SeaweedFS** | latest | Master + 3 volume servers + filer + S3 gw | 1 master, 3 volumes, 1 filer+S3 | 8333 |
| **Herd** | built from source | Rendezvous hash, embedded multi-node | 3 embedded nodes, S3 gateway | 9230 |

### Why 3-Node Cluster via S3 Only

- **Fair comparison**: All tested through identical S3 HTTP interface (aws-sdk-go-v2)
- **Cluster mode**: Tests distributed write/read path, not single-node shortcuts
- **Real-world**: Most production deployments are 3+ node clusters accessed via S3

### Port Allocation

| System | S3 API | Internal Ports | Data Directories |
|--------|--------|---------------|-----------------|
| MinIO node 1/2/3 | 9000/9001/9002 | Console: 9010/9011/9012 | /tmp/cluster-bench/minio/node{1,2,3} |
| RustFS node 1/2/3 | 9100/9101/9102 | Console: 9110/9111/9112 | /tmp/cluster-bench/rustfs/node{1,2,3}/vol{1,2} |
| SeaweedFS | 8333 (S3) | Master:9333, Vol:8080-8082, Filer:8888 | /tmp/cluster-bench/seaweedfs/{master,vol1,vol2,vol3} |
| Herd | 9230 | Embedded (no extra ports) | /tmp/cluster-bench/herd |

## Benchmark Suite

Reuses the existing `bench/` framework from `cmd/bench/`. The cluster benchmark runs the same S3 benchmark suite with `--drivers` filtering to only cluster S3 endpoints.

### Benchmarks Executed

1. **Sequential Write** — 1KB, 64KB, 1MB, 10MB sizes
2. **Sequential Read** — same sizes, read from pre-populated pool
3. **Stat (HEAD)** — metadata-only latency
4. **List** — list 100 objects with prefix
5. **Delete** — create+delete cycle
6. **Parallel Write** — C1, C10, C25, C50, C100, C200 at 1KB
7. **Parallel Read** — same concurrency levels
8. **Range Read** — 256KB range reads from 1MB object (start/middle/end)
9. **Copy** — server-side copy at 1KB
10. **Mixed Workload** — 90/10, 50/50, 10/90 read/write at 16KB
11. **Multipart Upload** — 15MB via 3×5MB parts
12. **Scale** — write/list/delete at 10, 100, 1000, 10000 objects

### Metrics Captured

- Throughput (MB/s)
- Ops/sec
- Latency: min, avg, p50, p95, p99, max
- Error count and messages
- Concurrency scaling curve

## Implementation

### Directory Structure

```
tools/cluster_bench/
├── main.go              # Orchestrator: install, start, bench, stop, report
├── cluster.go           # Cluster lifecycle management (start/stop/health)
├── install.sh           # Install all dependencies
├── run.sh               # One-command: install → start → bench → stop → report
└── README.md            # Usage instructions
```

### Cluster Lifecycle (cluster.go)

Each system implements a `Cluster` interface:

```go
type Cluster interface {
    Name() string
    Install() error           // Install binary if missing
    Start() error             // Start 3-node cluster
    WaitReady(timeout) error  // Poll health endpoint
    S3Endpoint() string       // e.g., "http://localhost:9000"
    S3Credentials() (ak, sk)  // Access/secret key
    Stop() error              // SIGTERM all processes
    Cleanup() error           // Remove data dirs
}
```

### MinIO 3-Node Cluster

MinIO distributed mode requires drives to be a multiple of erasure set size (2–16). With 3 nodes × 2 dirs = 6 drives, erasure set size = 6, parity = 3 (can survive up to 2 drive failures).

```bash
# All three nodes get identical server URL list:
export MINIO_ROOT_USER=minioadmin MINIO_ROOT_PASSWORD=minioadmin
minio server --address :900{i} --console-address :901{i} \
  http://127.0.0.1:9000/tmp/cluster-bench/minio/node1 \
  http://127.0.0.1:9001/tmp/cluster-bench/minio/node2 \
  http://127.0.0.1:9002/tmp/cluster-bench/minio/node3
```

Health: `GET http://localhost:9000/minio/health/live` → 200
Credentials: `minioadmin` / `minioadmin`

### RustFS 3-Node Cluster

RustFS mirrors MinIO's distributed mode. Same 3×2=6 drives pattern.

```bash
export RUSTFS_ROOT_USER=rustfsadmin RUSTFS_ROOT_PASSWORD=rustfsadmin
rustfs server --address :910{i} --console-address :911{i} \
  http://127.0.0.1:9100/tmp/cluster-bench/rustfs/node1/vol1 \
  http://127.0.0.1:9100/tmp/cluster-bench/rustfs/node1/vol2 \
  http://127.0.0.1:9101/tmp/cluster-bench/rustfs/node2/vol1 \
  http://127.0.0.1:9101/tmp/cluster-bench/rustfs/node2/vol2 \
  http://127.0.0.1:9102/tmp/cluster-bench/rustfs/node3/vol1 \
  http://127.0.0.1:9102/tmp/cluster-bench/rustfs/node3/vol2
```

Health: `GET http://localhost:9100/minio/health/live` → 200
Credentials: `rustfsadmin` / `rustfsadmin`

### SeaweedFS 3-Node Cluster

SeaweedFS has a layered architecture. For cluster mode we run 1 master + 3 volume servers + 1 filer with S3 gateway.

```bash
weed master -mdir=/tmp/cluster-bench/seaweedfs/master -port=9333

weed volume -mserver=localhost:9333 -dir=/tmp/cluster-bench/seaweedfs/vol1 -port=8080
weed volume -mserver=localhost:9333 -dir=/tmp/cluster-bench/seaweedfs/vol2 -port=8081
weed volume -mserver=localhost:9333 -dir=/tmp/cluster-bench/seaweedfs/vol3 -port=8082

weed filer -master=localhost:9333 -port=8888 -s3 -s3.port=8333 \
  -s3.config=/tmp/cluster-bench/seaweedfs/s3.json
```

S3 config (`s3.json`):
```json
{
  "identities": [{
    "name": "admin",
    "credentials": [{"accessKey": "admin", "secretKey": "adminpassword"}],
    "actions": ["*"]
  }]
}
```

Health: `GET http://localhost:9333/cluster/status` → JSON
Credentials: `admin` / `adminpassword`

### Herd 3-Node Embedded Cluster

Herd runs 3 embedded nodes in a single process with S3 gateway. Zero network overhead between nodes — rendezvous hashing routes directly to in-memory stores.

```bash
herd -nodes 3 -listen :9230 -data-dir /tmp/cluster-bench/herd \
  -stripes 16 -sync none -inline-kb 8 -prealloc 1024 \
  -access-key herd -secret-key herd123
```

Health: `GET http://localhost:9230/` → response (any HTTP response = alive)
Credentials: `herd` / `herd123`

## Execution Plan

### Phase 1: Install Dependencies
1. Check/install `minio` binary
2. Check/install `rustfs` binary
3. Check/install `weed` (seaweedfs) binary
4. Build `herd` binary from source

### Phase 2: Start Clusters
1. Create data directories under `/tmp/cluster-bench/`
2. Start each cluster's processes in background
3. Wait for health checks (30s timeout each)
4. Create test buckets via aws-sdk-go-v2

### Phase 3: Run Benchmarks
1. Run `cmd/bench` with `--drivers minio_cluster,rustfs_cluster,seaweedfs_cluster,herd_cluster`
2. Add cluster driver configs to benchmark (or run standalone via tools/cluster_bench/main.go)
3. Capture results as JSON + CSV + Markdown

### Phase 4: Generate Report
1. Parse benchmark results
2. Generate comparison tables (throughput, latency, concurrency scaling)
3. Identify performance leaders per category
4. Save to `report/cluster_bench_report.md`

### Phase 5: Cleanup
1. SIGTERM all cluster processes
2. Remove `/tmp/cluster-bench/`

## Expected Outcomes

### Performance Hypotheses

- **Herd** will likely dominate small-object writes (inline caching, zero network between nodes)
- **MinIO** is battle-tested, expected strong all-around performance
- **RustFS** should match or exceed MinIO (Rust vs Go, same algorithm)
- **SeaweedFS** has extra layers (master → volume → filer → S3), may have higher latency but potentially better large-object throughput

### Key Comparisons

1. **Small object throughput** (1KB write/read) — tests per-request overhead
2. **Large object throughput** (10MB write/read) — tests streaming efficiency
3. **Concurrency scaling** (C1 → C200) — tests lock contention and connection handling
4. **Metadata ops** (Stat, List) — tests index/namespace performance
5. **Mixed workload** — tests real-world usage patterns

## Success Criteria

- All 4 systems benchmarked in cluster mode
- Results reproducible (adaptive benchmarking with statistical significance)
- Report includes throughput comparison, latency percentiles, and concurrency scaling charts
- No system crashes or data corruption during benchmarks
