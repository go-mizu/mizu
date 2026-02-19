# Docker Cluster Deployments

True multi-node cluster configurations for benchmarking. Each product runs 4 S3 endpoints behind HAProxy.

## Quick Start

```bash
# MinIO: 4-node distributed erasure coding
cd minio && docker compose up -d
# S3: http://localhost:9000  Creds: minioadmin/minioadmin

# RustFS: 4-node distributed (alpha)
cd rustfs && docker compose up -d
# S3: http://localhost:9100  Creds: rustfsadmin/rustfsadmin

# SeaweedFS: 1 master + 3 volumes + 4 filer/S3
cd seaweedfs && docker compose up -d
# S3: http://localhost:8333  Creds: admin/adminpassword

# Herd: 4 independent S3+storage servers
cd herd && docker build -t herd:latest -f Dockerfile ../../.. && docker compose up -d
# S3: http://localhost:9230  Creds: herd/herd123
```

## Architecture

| Product | Nodes | Internal Protocol | LB Strategy |
|---------|-------|-------------------|-------------|
| MinIO | 4 (erasure coded) | HTTP between nodes | leastconn |
| RustFS | 4 (erasure coded) | gRPC between nodes | leastconn |
| SeaweedFS | 4 S3 + 3 vol + 1 master | HTTP/gRPC | leastconn |
| Herd | 4 independent stores | None | uri hash |

## Native Binary Benchmark (No Docker)

For lower overhead benchmarking without Docker:

```bash
# Install dependencies
./tools/cluster_bench/install.sh

# Run benchmark (starts all clusters, benchmarks, stops)
./tools/cluster_bench/run.sh

# Or run directly with options
go run ./tools/cluster_bench/ -systems minio,herd -quick
```
