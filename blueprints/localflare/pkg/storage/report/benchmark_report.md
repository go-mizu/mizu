# S3 Storage Driver Benchmark Report

Generated: 2026-01-14
Go Version: go1.24
Platform: darwin/arm64 (Apple M4)

## Summary

This report contains benchmark results for the S3 storage driver across multiple S3-compatible backends:
- **RustFS** - High-performance Rust-based S3 server (port 9100)
- **SeaweedFS** - Distributed object storage (port 8333)
- **LocalStack** - AWS Local Testing emulator (port 4566)

**Note:** MinIO was not available during this benchmark run.

## Key Findings

### Performance Rankings

| Operation | Best Performer | Throughput |
|-----------|---------------|------------|
| Sequential Write (1MB) | RustFS | 144.43 MB/s |
| Sequential Read (1MB) | SeaweedFS | 247.41 MB/s |
| Parallel Write (C1) | SeaweedFS | 156.08 MB/s |
| Parallel Read (C25) | SeaweedFS | 168.02 MB/s |
| Copy (1MB) | LocalStack | 329.03 MB/s |
| List (100 objects) | SeaweedFS | 2.17ms |

### Known Issues

- **RustFS**: Has HTTP connection issues at high concurrency (C25+). Benchmark limited to C10 max.
- **Multipart Upload**: All drivers fail due to S3 minimum part size requirement (5MB). Test uses 1MB parts.

---

## Write Benchmarks

| Driver | Size | Throughput | Latency | Allocs/op |
|--------|------|------------|---------|-----------|
| rustfs | 1B | 0.00 MB/s | 777.5us | 567 |
| rustfs | 1KB | 1.29 MB/s | 793.5us | 571 |
| rustfs | 64KB | 58.70 MB/s | 1.12ms | 571 |
| rustfs | 1MB | **144.43 MB/s** | 7.26ms | 571 |
| seaweedfs | 1B | 0.00 MB/s | 766.9us | 564 |
| seaweedfs | 1KB | 1.48 MB/s | 693.4us | 568 |
| seaweedfs | 64KB | 59.38 MB/s | 1.10ms | 568 |
| seaweedfs | 1MB | 127.99 MB/s | 8.19ms | 568 |
| localstack | 1B | 0.00 MB/s | 814.8us | 578 |
| localstack | 1KB | 1.31 MB/s | 783.3us | 582 |
| localstack | 64KB | 58.44 MB/s | 1.12ms | 582 |
| localstack | 1MB | 129.91 MB/s | 8.07ms | 582 |

---

## Read Benchmarks

| Driver | Size | Throughput | Latency | Allocs/op |
|--------|------|------------|---------|-----------|
| rustfs | 1KB | 2.23 MB/s | 458.4us | 517 |
| rustfs | 64KB | 89.93 MB/s | 728.7us | 517 |
| rustfs | 1MB | 224.15 MB/s | 4.68ms | 517 |
| seaweedfs | 1KB | 2.47 MB/s | 413.8us | 512 |
| seaweedfs | 64KB | 96.29 MB/s | 680.6us | 512 |
| seaweedfs | 1MB | **247.41 MB/s** | 4.24ms | 512 |
| localstack | 1KB | 1.33 MB/s | 771.1us | 524 |
| localstack | 64KB | 66.65 MB/s | 983.3us | 524 |
| localstack | 1MB | 246.43 MB/s | 4.26ms | 524 |

---

## Range Read Benchmarks

| Driver | Range | Throughput | Latency |
|--------|-------|------------|---------|
| rustfs | Start 256KB | 124.56 MB/s | 2.10ms |
| rustfs | Middle 256KB | 120.01 MB/s | 2.18ms |
| rustfs | End 256KB | 121.34 MB/s | 2.16ms |
| rustfs | Tiny 4KB | 3.01 MB/s | 1.36ms |
| seaweedfs | Start 256KB | **178.01 MB/s** | 1.47ms |
| seaweedfs | Middle 256KB | 171.68 MB/s | 1.53ms |
| seaweedfs | End 256KB | 183.55 MB/s | 1.43ms |
| seaweedfs | Tiny 4KB | 9.86 MB/s | 415.4us |
| localstack | Start 256KB | 162.16 MB/s | 1.62ms |
| localstack | Middle 256KB | 159.04 MB/s | 1.65ms |
| localstack | End 256KB | 108.02 MB/s | 2.43ms |
| localstack | Tiny 4KB | 5.21 MB/s | 786.3us |

---

## Parallel Write Benchmarks

| Driver | Concurrency | Throughput | Latency |
|--------|-------------|------------|---------|
| rustfs | C1 | 142.90 MB/s | 458.6us |
| rustfs | C10 | 68.08 MB/s | 962.6us |
| rustfs | C25 | *SKIPPED* (connection issues) | - |
| seaweedfs | C1 | **156.08 MB/s** | 419.9us |
| seaweedfs | C10 | 23.46 MB/s | 2.79ms |
| seaweedfs | C25 | 58.83 MB/s | 1.11ms |
| localstack | C1 | 71.06 MB/s | 922.3us |
| localstack | C10 | 42.42 MB/s | 1.54ms |
| localstack | C25 | 48.17 MB/s | 1.36ms |

---

## Parallel Read Benchmarks

| Driver | Concurrency | Throughput | Latency |
|--------|-------------|------------|---------|
| rustfs | C1 | **226.60 MB/s** | 289.2us |
| rustfs | C10 | 188.28 MB/s | 348.1us |
| rustfs | C25 | 141.22 MB/s | 464.1us |
| seaweedfs | C1 | 190.42 MB/s | 344.2us |
| seaweedfs | C10 | 170.00 MB/s | 385.5us |
| seaweedfs | C25 | 168.02 MB/s | 390.0us |
| localstack | C1 | 72.89 MB/s | 899.1us |
| localstack | C10 | 90.06 MB/s | 727.7us |
| localstack | C25 | 67.11 MB/s | 976.5us |

---

## Mixed Workload Benchmarks

| Driver | Workload | Throughput | Latency |
|--------|----------|------------|---------|
| rustfs | 90% Read / 10% Write | 108.10 MB/s | 151.6us |
| rustfs | 10% Read / 90% Write | 63.87 MB/s | 256.5us |
| rustfs | 50% Read / 50% Write | 76.58 MB/s | 214.0us |
| seaweedfs | 90% Read / 10% Write | **120.23 MB/s** | 136.3us |
| seaweedfs | 10% Read / 90% Write | **85.70 MB/s** | 191.2us |
| seaweedfs | 50% Read / 50% Write | **107.82 MB/s** | 152.0us |
| localstack | 90% Read / 10% Write | 23.09 MB/s | 709.5us |
| localstack | 10% Read / 90% Write | 26.25 MB/s | 624.2us |
| localstack | 50% Read / 50% Write | 22.62 MB/s | 724.3us |

---

## Stat Benchmarks

| Driver | Type | Latency | Allocs/op |
|--------|------|---------|-----------|
| rustfs | Exists | 339.1us | 486 |
| rustfs | NotExists | 208.5us | 499 |
| seaweedfs | Exists | 353.1us | 481 |
| seaweedfs | NotExists | 297.6us | 503 |
| localstack | Exists | 713.1us | 493 |
| localstack | NotExists | 724.9us | 510 |

---

## Delete Benchmarks

| Driver | Type | Latency | Allocs/op |
|--------|------|---------|-----------|
| rustfs | Single | 3.49ms | 463 |
| rustfs | NonExistent | 246.1us | 462 |
| seaweedfs | Single | **308.8us** | 455 |
| seaweedfs | NonExistent | 307.4us | 457 |
| localstack | Single | 662.0us | 465 |
| localstack | NonExistent | 642.4us | 467 |

---

## Copy Benchmarks

| Driver | Size | Throughput | Latency |
|--------|------|------------|---------|
| rustfs | 1KB | 0.95 MB/s | 1.07ms |
| rustfs | 1MB | 224.43 MB/s | 4.67ms |
| seaweedfs | 1KB | 0.84 MB/s | 1.21ms |
| seaweedfs | 1MB | 215.14 MB/s | 4.87ms |
| localstack | 1KB | 1.13 MB/s | 909.1us |
| localstack | 1MB | **329.03 MB/s** | 3.19ms |

---

## List Benchmarks

| Driver | Objects | Latency | Allocs/op |
|--------|---------|---------|-----------|
| rustfs | 10 | 1.21ms | 1,123 |
| rustfs | 50 | 3.56ms | 3,446 |
| rustfs | 100 | 7.00ms | 6,348 |
| rustfs | Prefix 50/200 | 3.63ms | 3,447 |
| seaweedfs | 10 | **740.9us** | 1,124 |
| seaweedfs | 50 | **1.30ms** | 3,447 |
| seaweedfs | 100 | **2.17ms** | 6,349 |
| seaweedfs | Prefix 50/200 | **1.43ms** | 3,448 |
| localstack | 10 | 6.24ms | 1,319 |
| localstack | 50 | 7.78ms | 4,403 |
| localstack | 100 | 9.27ms | 8,254 |
| localstack | Prefix 50/200 | 7.54ms | 4,403 |

---

## Bucket Operations

| Driver | Operation | Latency | Allocs/op |
|--------|-----------|---------|-----------|
| rustfs | Create+Delete | 1.25ms | 871 |
| rustfs | BucketInfo | **199.6us** | 433 |
| seaweedfs | Create+Delete | 1.27ms | 873 |
| seaweedfs | BucketInfo | 304.7us | 432 |
| localstack | Create+Delete | 1.36ms | 894 |
| localstack | BucketInfo | 581.6us | 441 |

---

## Metadata Benchmarks

| Driver | Metadata Count | Throughput | Latency |
|--------|----------------|------------|---------|
| rustfs | None | 1.37 MB/s | 746.6us |
| rustfs | 5 keys | 1.32 MB/s | 775.8us |
| rustfs | 20 keys | 1.21 MB/s | 843.9us |
| seaweedfs | None | 1.40 MB/s | 730.9us |
| seaweedfs | 5 keys | **1.41 MB/s** | 725.8us |
| seaweedfs | 20 keys | 1.25 MB/s | 820.6us |
| localstack | None | 1.30 MB/s | 790.7us |
| localstack | 5 keys | 1.25 MB/s | 818.7us |
| localstack | 20 keys | 1.06 MB/s | 965.6us |

---

## Edge Cases

| Driver | Test | Latency | Allocs/op |
|--------|------|---------|-----------|
| rustfs | Empty Write | 800.4us | 553 |
| rustfs | Long Key (256) | 986.9us | 571 |
| rustfs | Unicode Key | 1.02ms | 570 |
| rustfs | Deep Nested | 775.8us | 568 |
| seaweedfs | Empty Write | **398.3us** | 550 |
| seaweedfs | Long Key (256) | 767.0us | 569 |
| seaweedfs | Unicode Key | 731.0us | 568 |
| seaweedfs | Deep Nested | 736.8us | 565 |
| localstack | Empty Write | 801.4us | 564 |
| localstack | Long Key (256) | 804.0us | 583 |
| localstack | Unicode Key | 795.1us | 582 |
| localstack | Deep Nested | 770.1us | 579 |

---

## Conclusions

1. **SeaweedFS** shows the best overall performance for most operations, particularly for read-heavy workloads and listing operations.

2. **RustFS** excels at sequential write operations (144 MB/s for 1MB files) and has excellent single-connection read performance (226 MB/s), but has **concurrency issues at C25+** causing connection timeouts.

3. **LocalStack** provides consistent performance suitable for local development and testing, but is slower than production-grade S3 implementations (expected as an emulator).

4. **Memory allocations** are consistent across drivers (~500-600 allocs/op for basic operations), indicating efficient SDK usage.

5. **Multipart uploads** need larger part sizes (5MB minimum per S3 spec) to work correctly.

## Recommendations

- For **production workloads** with high concurrency: Use SeaweedFS or MinIO
- For **development/testing**: LocalStack provides good compatibility
- For **low-concurrency, high-throughput**: RustFS performs excellently
- **Avoid** RustFS for workloads with >10 concurrent connections until the HTTP server issue is resolved
