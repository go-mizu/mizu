# S3 Benchmark Results

**Date:** 2026-01-16 01:59:17

## Configuration

- Threads: 8 - 12
- Payload sizes: 4 MB - 16 MB
- Samples per test: 100

- Drivers: liteio, liteio_mem, localstack, minio, rustfs, seaweedfs

## Summary

- **Best Throughput:** minio (34.8 MB/s with 8 MB, 8 threads)
- **Best TTFB:** minio (11 ms avg)
- **Best TTLB:** liteio (120 ms avg)

## 4 MB Objects

| Driver | Threads | Throughput | TTFB p50 | TTFB p99 | TTLB p50 | TTLB p99 |
|--------|---------|------------|----------|----------|----------|----------|
| liteio | 8 | 33.2 MB/s | 36 ms | 52 ms | 116 ms | 201 ms |
| liteio | 9 | 30.2 MB/s | 31 ms | 48 ms | 117 ms | 375 ms |
| liteio | 10 | 26.7 MB/s | 30 ms | 62 ms | 145 ms | 423 ms |
| liteio | 11 | 23.4 MB/s | 30 ms | 50 ms | 122 ms | 593 ms |
| liteio | 12 | 22.1 MB/s | 31 ms | 57 ms | 145 ms | 520 ms |
| liteio_mem | 8 | 30.5 MB/s | 31 ms | 47 ms | 123 ms | 216 ms |
| liteio_mem | 9 | 27.1 MB/s | 32 ms | 53 ms | 136 ms | 317 ms |
| liteio_mem | 10 | 25.4 MB/s | 32 ms | 55 ms | 135 ms | 384 ms |
| liteio_mem | 11 | 21.6 MB/s | 38 ms | 68 ms | 182 ms | 404 ms |
| liteio_mem | 12 | 20.3 MB/s | 38 ms | 67 ms | 175 ms | 379 ms |
| localstack | 8 | 26.7 MB/s | 33 ms | 53 ms | 134 ms | 324 ms |
| localstack | 9 | 24.6 MB/s | 20 ms | 37 ms | 138 ms | 518 ms |
| localstack | 10 | 22.1 MB/s | 19 ms | 30 ms | 156 ms | 466 ms |
| localstack | 11 | 20.9 MB/s | 34 ms | 56 ms | 175 ms | 494 ms |
| localstack | 12 | 19.3 MB/s | 26 ms | 44 ms | 181 ms | 631 ms |
| minio | 8 | 30.4 MB/s | 14 ms | 60 ms | 126 ms | 195 ms |
| minio | 9 | 26.9 MB/s | 12 ms | 21 ms | 141 ms | 378 ms |
| minio | 10 | 24.2 MB/s | 14 ms | 24 ms | 166 ms | 219 ms |
| minio | 11 | 21.5 MB/s | 13 ms | 19 ms | 178 ms | 356 ms |
| minio | 12 | 19.4 MB/s | 17 ms | 23 ms | 198 ms | 371 ms |
| rustfs | 8 | 27.5 MB/s | 22 ms | 51 ms | 142 ms | 206 ms |
| rustfs | 9 | 26.7 MB/s | 18 ms | 24 ms | 153 ms | 208 ms |
| rustfs | 10 | 23.6 MB/s | 19 ms | 27 ms | 164 ms | 270 ms |
| rustfs | 11 | 21.5 MB/s | 20 ms | 26 ms | 197 ms | 289 ms |
| rustfs | 12 | 20.1 MB/s | 22 ms | 32 ms | 197 ms | 323 ms |
| seaweedfs | 8 | 28.4 MB/s | 38 ms | 78 ms | 134 ms | 259 ms |
| seaweedfs | 9 | 26.4 MB/s | 32 ms | 49 ms | 122 ms | 442 ms |
| seaweedfs | 10 | 24.1 MB/s | 38 ms | 64 ms | 154 ms | 456 ms |
| seaweedfs | 11 | 22.4 MB/s | 35 ms | 59 ms | 143 ms | 446 ms |
| seaweedfs | 12 | 20.2 MB/s | 40 ms | 70 ms | 182 ms | 346 ms |

## 8 MB Objects

| Driver | Threads | Throughput | TTFB p50 | TTFB p99 | TTLB p50 | TTLB p99 |
|--------|---------|------------|----------|----------|----------|----------|
| liteio | 8 | 28.1 MB/s | 54 ms | 74 ms | 261 ms | 689 ms |
| liteio | 9 | 25.1 MB/s | 55 ms | 122 ms | 303 ms | 574 ms |
| liteio | 10 | 19.3 MB/s | 89 ms | 477 ms | 337 ms | 573 ms |
| liteio | 11 | 0.0 MB/s | 0 ms | 0 ms | 0 ms | 0 ms |
| liteio | 12 | 0.0 MB/s | 0 ms | 0 ms | 0 ms | 0 ms |
| liteio_mem | 8 | 30.2 MB/s | 43 ms | 66 ms | 266 ms | 375 ms |
| liteio_mem | 9 | 27.8 MB/s | 41 ms | 64 ms | 264 ms | 966 ms |
| liteio_mem | 10 | 24.5 MB/s | 46 ms | 64 ms | 289 ms | 734 ms |
| liteio_mem | 11 | 22.2 MB/s | 50 ms | 74 ms | 333 ms | 944 ms |
| liteio_mem | 12 | 20.9 MB/s | 46 ms | 71 ms | 320 ms | 911 ms |
| localstack | 8 | 25.9 MB/s | 33 ms | 45 ms | 216 ms | 1254 ms |
| localstack | 9 | 23.7 MB/s | 35 ms | 52 ms | 242 ms | 1475 ms |
| localstack | 10 | 21.5 MB/s | 28 ms | 38 ms | 205 ms | 1353 ms |
| localstack | 11 | 19.9 MB/s | 38 ms | 52 ms | 303 ms | 1080 ms |
| minio | 8 | 34.8 MB/s | 15 ms | 29 ms | 227 ms | 330 ms |
| minio | 9 | 29.8 MB/s | 12 ms | 17 ms | 259 ms | 395 ms |
| minio | 10 | 26.9 MB/s | 12 ms | 17 ms | 285 ms | 437 ms |
| minio | 11 | 23.7 MB/s | 13 ms | 22 ms | 340 ms | 454 ms |
| minio | 12 | 21.3 MB/s | 14 ms | 20 ms | 362 ms | 563 ms |
| rustfs | 8 | 27.9 MB/s | 27 ms | 60 ms | 279 ms | 417 ms |
| rustfs | 9 | 26.6 MB/s | 21 ms | 31 ms | 286 ms | 482 ms |
| rustfs | 10 | 23.0 MB/s | 24 ms | 38 ms | 365 ms | 449 ms |
| rustfs | 11 | 21.8 MB/s | 25 ms | 46 ms | 361 ms | 529 ms |
| rustfs | 12 | 19.5 MB/s | 26 ms | 34 ms | 400 ms | 614 ms |
| seaweedfs | 8 | 28.2 MB/s | 50 ms | 64 ms | 257 ms | 994 ms |
| seaweedfs | 9 | 25.8 MB/s | 45 ms | 62 ms | 258 ms | 704 ms |
| seaweedfs | 10 | 23.5 MB/s | 45 ms | 63 ms | 266 ms | 1119 ms |
| seaweedfs | 11 | 21.5 MB/s | 54 ms | 79 ms | 333 ms | 1252 ms |
| seaweedfs | 12 | 19.1 MB/s | 67 ms | 90 ms | 390 ms | 1111 ms |

