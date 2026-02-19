# Docker Configurations

## Structure

```
docker/
  s3/          Single-node S3 configs (development/testing)
  cluster/     Multi-node cluster configs (benchmarking)
```

## Single-Node (`docker/s3/`)

Individual and combined S3 services for local development and integration testing.

```bash
cd s3/all && docker compose up -d    # All services
cd s3/minio && docker compose up -d  # Just MinIO
```

See [s3/README.md](s3/README.md) for details.

## Cluster Mode (`docker/cluster/`)

True multi-node clusters with HAProxy load balancing for benchmarking.

| Product | Nodes | S3 Port | Architecture |
|---------|-------|---------|-------------|
| MinIO | 4 distributed | 9000 | Erasure coded, HAProxy LB |
| RustFS | 4 distributed | 9100 | Erasure coded, HAProxy LB |
| SeaweedFS | 4 S3 + 3 vol + 1 master | 8333 | Shared storage, HAProxy LB |
| Herd | 4 independent | 9230 | Independent stores, HAProxy LB |

```bash
cd cluster/minio && docker compose up -d
cd cluster/herd && docker build -t herd:latest -f Dockerfile ../../.. && docker compose up -d
```

See [cluster/README.md](cluster/README.md) for details.
