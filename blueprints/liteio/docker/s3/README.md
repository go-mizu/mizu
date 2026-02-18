# S3-Compatible Storage Docker Configurations

This directory contains Docker Compose configurations for production-grade, open-source S3-compatible storage systems. These are used for testing the S3 storage driver.

## Quick Start

### Start MinIO (recommended for local development)

```bash
cd minio
docker compose up -d
```

Access:
- S3 API: `http://localhost:9000`
- Console: `http://localhost:9001`
- Credentials: `minioadmin` / `minioadmin`

### Start Multiple Services for Testing

```bash
cd all
docker compose up -d minio rustfs localstack
```

## S3 Implementations

| Service | Language | Port | Access Key | Secret Key | Notes |
|---------|----------|------|------------|------------|-------|
| **MinIO** | Go | 9000 | minioadmin | minioadmin | Industry reference, single binary |
| **RustFS** | Rust | 9000 | rustfsadmin | rustfsadmin | High-performance MinIO fork, 2.3x faster |
| **SeaweedFS** | Go | 8333 | (anonymous) | (anonymous) | Distributed volume + filer architecture |
| **Garage** | Rust | 3900 | (configure) | (configure) | Decentralized, erasure-coded |
| **Zenko** | Node.js | 8000 | accessKey1 | verySecretKey1 | Scality CloudServer, lightweight |
| **LocalStack** | Python | 4566 | test | test | AWS local testing environment |
| **Ceph RGW** | C++ | 8080 | demo | demosecret | Enterprise-grade, heavy |

## Testing with Different Implementations

Set environment variables to test against different S3 implementations:

### MinIO (Recommended)
```bash
export S3_TEST_ENDPOINT=localhost:9000
export S3_TEST_ACCESS_KEY=minioadmin
export S3_TEST_SECRET_KEY=minioadmin
export S3_TEST_BUCKET=test-bucket
export S3_TEST_REGION=us-east-1
export S3_TEST_INSECURE=true

go test -v ./pkg/storage/driver/exp/s3/...
```

### RustFS
```bash
export S3_TEST_ENDPOINT=localhost:9000
export S3_TEST_ACCESS_KEY=rustfsadmin
export S3_TEST_SECRET_KEY=rustfsadmin
export S3_TEST_BUCKET=test-bucket
export S3_TEST_INSECURE=true
```

### SeaweedFS
```bash
export S3_TEST_ENDPOINT=localhost:8333
export S3_TEST_ACCESS_KEY=admin
export S3_TEST_SECRET_KEY=adminpassword
export S3_TEST_BUCKET=test-bucket
export S3_TEST_INSECURE=true
```

### Zenko CloudServer
```bash
export S3_TEST_ENDPOINT=localhost:8000
export S3_TEST_ACCESS_KEY=accessKey1
export S3_TEST_SECRET_KEY=verySecretKey1
export S3_TEST_BUCKET=test-bucket
export S3_TEST_INSECURE=true
```

### LocalStack
```bash
export S3_TEST_ENDPOINT=localhost:4566
export S3_TEST_ACCESS_KEY=test
export S3_TEST_SECRET_KEY=test
export S3_TEST_BUCKET=test-bucket
export S3_TEST_INSECURE=true
```

### Ceph RGW
```bash
export S3_TEST_ENDPOINT=localhost:8080
export S3_TEST_ACCESS_KEY=demo
export S3_TEST_SECRET_KEY=demosecret
export S3_TEST_BUCKET=test-bucket
export S3_TEST_INSECURE=true
```

## DSN Format

The S3 driver uses the following DSN format:

```
s3://[access_key:secret_key@]endpoint/bucket?params
```

Examples:

```bash
# MinIO
s3://minioadmin:minioadmin@localhost:9000/mybucket?insecure=true

# AWS S3
s3://mybucket?region=us-west-2

# RustFS
s3://rustfsadmin:rustfsadmin@localhost:9000/mybucket?insecure=true

# Zenko
s3://accessKey1:verySecretKey1@localhost:8000/mybucket?insecure=true

# LocalStack
s3://test:test@localhost:4566/mybucket?insecure=true

# SeaweedFS
s3://admin:adminpassword@localhost:8333/mybucket?insecure=true
```

### Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `region` | AWS region | us-east-1 |
| `force_path_style` | Use path-style URLs | auto |
| `insecure` | Use HTTP instead of HTTPS | false |

## Architectural Notes

### Single-Binary (Easy to Deploy)
- **MinIO**: Most compatible, production-ready
- **RustFS**: High-performance MinIO fork in Rust
- **Garage**: Rust, decentralized, erasure-coded
- **Zenko**: Lightweight, Node.js

### Distributed (Multi-Component)
- **SeaweedFS**: Master + Volume + Filer + S3
- **Ceph RGW**: Full Ceph cluster required

### AWS Local Testing
- **LocalStack**: Full AWS API emulation including S3

## Running Tests

```bash
# Start MinIO
cd docker/s3/minio
docker compose up -d

# Wait for healthy
docker compose ps

# Run integration tests
cd ../../..
export S3_TEST_ENDPOINT=localhost:9000
export S3_TEST_ACCESS_KEY=minioadmin
export S3_TEST_SECRET_KEY=minioadmin
export S3_TEST_BUCKET=test-bucket
export S3_TEST_INSECURE=true

go test -v ./pkg/storage/driver/exp/s3/...
```

## Cleanup

```bash
# Stop services
docker compose down

# Remove volumes
docker compose down -v
```
