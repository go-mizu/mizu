# S3 Benchmark Results

**Date:** 2026-01-16 01:52:22

## Configuration

- Threads: 8 - 10
- Payload sizes: 4 MB - 8 MB
- Samples per test: 20

- Drivers: liteio, minio, rustfs

## Summary

- **Best Throughput:** liteio (38.9 MB/s with 4 MB, 8 threads)
- **Best TTFB:** minio (7 ms avg)
- **Best TTLB:** liteio (102 ms avg)

## 4 MB Objects

| Driver | Threads | Throughput | TTFB p50 | TTFB p99 | TTLB p50 | TTLB p99 |
|--------|---------|------------|----------|----------|----------|----------|
| liteio | 8 | 38.9 MB/s | 10 ms | 17 ms | 84 ms | 168 ms |
| liteio | 9 | 34.3 MB/s | 11 ms | 15 ms | 106 ms | 239 ms |
| liteio | 10 | 28.8 MB/s | 12 ms | 18 ms | 108 ms | 280 ms |
| minio | 8 | 33.8 MB/s | 9 ms | 13 ms | 106 ms | 183 ms |
| minio | 9 | 31.5 MB/s | 8 ms | 12 ms | 118 ms | 174 ms |
| minio | 10 | 28.0 MB/s | 5 ms | 12 ms | 120 ms | 233 ms |
| rustfs | 8 | 34.5 MB/s | 13 ms | 17 ms | 101 ms | 171 ms |
| rustfs | 9 | 31.9 MB/s | 12 ms | 14 ms | 115 ms | 186 ms |
| rustfs | 10 | 27.2 MB/s | 13 ms | 19 ms | 141 ms | 220 ms |

## 8 MB Objects

| Driver | Threads | Throughput | TTFB p50 | TTFB p99 | TTLB p50 | TTLB p99 |
|--------|---------|------------|----------|----------|----------|----------|
| liteio | 8 | 36.9 MB/s | 16 ms | 27 ms | 175 ms | 420 ms |
| liteio | 9 | 30.9 MB/s | 15 ms | 27 ms | 229 ms | 540 ms |
| liteio | 10 | 30.0 MB/s | 12 ms | 28 ms | 212 ms | 543 ms |
| minio | 8 | 33.4 MB/s | 9 ms | 14 ms | 200 ms | 389 ms |
| minio | 9 | 29.9 MB/s | 9 ms | 14 ms | 250 ms | 381 ms |
| minio | 10 | 22.4 MB/s | 5 ms | 17 ms | 296 ms | 554 ms |
| rustfs | 8 | 33.0 MB/s | 17 ms | 20 ms | 226 ms | 359 ms |
| rustfs | 9 | 31.0 MB/s | 20 ms | 31 ms | 225 ms | 428 ms |
| rustfs | 10 | 27.3 MB/s | 20 ms | 24 ms | 270 ms | 426 ms |

