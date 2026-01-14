# Storage Benchmark Report

**Generated:** 2026-01-15T01:08:28+07:00

**Go Version:** go1.25.5

**Platform:** darwin/arm64

## Executive Summary

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (1MB+) | **liteio_mem** | 161 MB/s | Best for media, backups |
| Large File Downloads | **liteio** | 303 MB/s | Best for streaming, CDN |
| Small File Operations | **liteio_mem** | 2911 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **liteio** | - | Best for multi-user apps |

### Large File Performance (1MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| liteio | 160.5 | 303.0 | 6.0ms | 3.3ms |
| liteio_mem | 161.4 | 294.1 | 6.0ms | 3.4ms |
| localstack | 124.7 | 242.9 | 7.9ms | 4.1ms |
| minio | 139.6 | 257.7 | 6.9ms | 3.8ms |
| rustfs | 148.7 | 207.4 | 6.1ms | 4.7ms |
| seaweedfs | 128.0 | 252.0 | 7.8ms | 3.9ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| liteio | 733 | 4773 | 1.2ms | 206.1us |
| liteio_mem | 533 | 5289 | 764.4us | 192.9us |
| localstack | 1307 | 1455 | 755.8us | 678.7us |
| minio | 717 | 3178 | 1.4ms | 306.0us |
| rustfs | 1260 | 2235 | 758.3us | 440.8us |
| seaweedfs | 1360 | 2433 | 732.0us | 403.3us |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| liteio | 5714 | 1363 | 6886 |
| liteio_mem | 5908 | 1284 | 6542 |
| localstack | 1518 | 335 | 1689 |
| minio | 4285 | 614 | 3165 |
| rustfs | 3473 | 167 | 1233 |
| seaweedfs | 3793 | 650 | 3307 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| liteio | 1.36 | 0.31 | 0.19 |
| liteio_mem | 1.32 | 0.30 | 0.18 |
| localstack | 1.26 | 0.17 | 0.10 |
| minio | 1.03 | 0.27 | 0.19 |
| rustfs | 1.37 | 0.34 | - |
| seaweedfs | 1.43 | 0.30 | 0.23 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| liteio | 3.73 | 1.13 | 0.53 |
| liteio_mem | 4.32 | 0.80 | 0.51 |
| localstack | 1.32 | 0.19 | 0.09 |
| minio | 3.11 | 0.96 | 0.66 |
| rustfs | 1.90 | 0.76 | - |
| seaweedfs | 2.33 | 0.75 | 0.46 |

*\* indicates errors occurred*

---

## Configuration

| Parameter | Value |
|-----------|-------|
| Iterations | 20 |
| Warmup | 5 |
| Concurrency | 50 |
| Timeout | 1m0s |

## Drivers Tested

- liteio (28 benchmarks)
- liteio_mem (28 benchmarks)
- localstack (28 benchmarks)
- minio (28 benchmarks)
- rustfs (26 benchmarks)
- seaweedfs (28 benchmarks)

## Performance Comparison

### Copy/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.51 MB/s | 585.4us | 865.4us | 865.4us | 0 |
| liteio_mem | 1.43 MB/s | 654.9us | 802.3us | 802.3us | 0 |
| localstack | 1.34 MB/s | 717.5us | 873.2us | 873.2us | 0 |
| seaweedfs | 0.98 MB/s | 973.7us | 1.2ms | 1.2ms | 0 |
| rustfs | 0.94 MB/s | 991.2us | 1.4ms | 1.4ms | 0 |
| minio | 0.56 MB/s | 1.2ms | 3.2ms | 3.2ms | 0 |

```
  liteio       ████████████████████████████████████████ 1.51 MB/s
  liteio_mem   █████████████████████████████████████ 1.43 MB/s
  localstack   ███████████████████████████████████ 1.34 MB/s
  seaweedfs    █████████████████████████ 0.98 MB/s
  rustfs       ████████████████████████ 0.94 MB/s
  minio        ██████████████ 0.56 MB/s
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 6886 ops/s | 142.7us | 174.3us | 174.3us | 0 |
| liteio_mem | 6542 ops/s | 150.5us | 179.1us | 179.1us | 0 |
| seaweedfs | 3307 ops/s | 300.8us | 338.2us | 338.2us | 0 |
| minio | 3165 ops/s | 310.3us | 420.8us | 420.8us | 0 |
| localstack | 1689 ops/s | 563.2us | 665.1us | 665.1us | 0 |
| rustfs | 1233 ops/s | 803.0us | 935.5us | 935.5us | 0 |

```
  liteio       ████████████████████████████████████████ 6886 ops/s
  liteio_mem   ██████████████████████████████████████ 6542 ops/s
  seaweedfs    ███████████████████ 3307 ops/s
  minio        ██████████████████ 3165 ops/s
  localstack   █████████ 1689 ops/s
  rustfs       ███████ 1233 ops/s
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 0.16 MB/s | 560.7us | 607.8us | 607.8us | 0 |
| seaweedfs | 0.15 MB/s | 628.4us | 667.0us | 667.0us | 0 |
| liteio_mem | 0.14 MB/s | 623.6us | 917.2us | 917.2us | 0 |
| localstack | 0.14 MB/s | 670.6us | 772.0us | 772.0us | 0 |
| liteio | 0.10 MB/s | 911.3us | 1.2ms | 1.2ms | 0 |
| minio | 0.09 MB/s | 1.0ms | 1.2ms | 1.2ms | 0 |

```
  rustfs       ████████████████████████████████████████ 0.16 MB/s
  seaweedfs    ██████████████████████████████████████ 0.15 MB/s
  liteio_mem   ███████████████████████████████████ 0.14 MB/s
  localstack   ███████████████████████████████████ 0.14 MB/s
  liteio       ████████████████████████ 0.10 MB/s
  minio        ██████████████████████ 0.09 MB/s
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 2114 ops/s | 471.6us | 542.8us | 542.8us | 0 |
| liteio_mem | 1327 ops/s | 748.8us | 966.0us | 966.0us | 0 |
| rustfs | 1255 ops/s | 781.7us | 860.6us | 860.6us | 0 |
| localstack | 1188 ops/s | 794.9us | 946.0us | 946.0us | 0 |
| minio | 995 ops/s | 952.6us | 1.2ms | 1.2ms | 0 |
| liteio | 248 ops/s | 1.1ms | 1.6ms | 1.6ms | 0 |

```
  seaweedfs    ████████████████████████████████████████ 2114 ops/s
  liteio_mem   █████████████████████████ 1327 ops/s
  rustfs       ███████████████████████ 1255 ops/s
  localstack   ██████████████████████ 1188 ops/s
  minio        ██████████████████ 995 ops/s
  liteio       ████ 248 ops/s
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 0.16 MB/s | 547.2us | 717.5us | 717.5us | 0 |
| liteio_mem | 0.14 MB/s | 684.0us | 773.5us | 773.5us | 0 |
| liteio | 0.12 MB/s | 749.5us | 1.0ms | 1.0ms | 0 |
| seaweedfs | 0.11 MB/s | 784.0us | 1.2ms | 1.2ms | 0 |
| localstack | 0.09 MB/s | 827.3us | 1.2ms | 1.2ms | 0 |
| minio | 0.06 MB/s | 1.4ms | 1.6ms | 1.6ms | 0 |

```
  rustfs       ████████████████████████████████████████ 0.16 MB/s
  liteio_mem   █████████████████████████████████ 0.14 MB/s
  liteio       █████████████████████████████ 0.12 MB/s
  seaweedfs    ████████████████████████████ 0.11 MB/s
  localstack   ███████████████████████ 0.09 MB/s
  minio        ███████████████ 0.06 MB/s
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1363 ops/s | 698.0us | 826.1us | 826.1us | 0 |
| liteio_mem | 1284 ops/s | 714.2us | 1.0ms | 1.0ms | 0 |
| seaweedfs | 650 ops/s | 1.5ms | 2.0ms | 2.0ms | 0 |
| minio | 614 ops/s | 1.6ms | 1.8ms | 1.8ms | 0 |
| localstack | 335 ops/s | 2.9ms | 3.5ms | 3.5ms | 0 |
| rustfs | 167 ops/s | 6.1ms | 6.3ms | 6.3ms | 0 |

```
  liteio       ████████████████████████████████████████ 1363 ops/s
  liteio_mem   █████████████████████████████████████ 1284 ops/s
  seaweedfs    ███████████████████ 650 ops/s
  minio        ██████████████████ 614 ops/s
  localstack   █████████ 335 ops/s
  rustfs       ████ 167 ops/s
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 8.93 MB/s | 1.7ms | 2.3ms | 2.3ms | 0 |
| liteio | 7.50 MB/s | 2.2ms | 2.7ms | 2.7ms | 0 |
| liteio_mem | 7.17 MB/s | 2.0ms | 3.0ms | 3.0ms | 0 |
| seaweedfs | 5.51 MB/s | 2.7ms | 3.6ms | 3.6ms | 0 |
| minio | 3.98 MB/s | 4.0ms | 4.6ms | 4.6ms | 0 |
| localstack | 1.44 MB/s | 11.8ms | 12.7ms | 12.7ms | 0 |

```
  rustfs       ████████████████████████████████████████ 8.93 MB/s
  liteio       █████████████████████████████████ 7.50 MB/s
  liteio_mem   ████████████████████████████████ 7.17 MB/s
  seaweedfs    ████████████████████████ 5.51 MB/s
  minio        █████████████████ 3.98 MB/s
  localstack   ██████ 1.44 MB/s
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 7.96 MB/s | 1.7ms | 2.7ms | 2.7ms | 0 |
| liteio_mem | 6.27 MB/s | 2.4ms | 3.1ms | 3.1ms | 0 |
| liteio | 5.90 MB/s | 2.7ms | 3.1ms | 3.1ms | 0 |
| seaweedfs | 4.15 MB/s | 3.6ms | 4.3ms | 4.3ms | 0 |
| minio | 3.49 MB/s | 4.5ms | 5.0ms | 5.0ms | 0 |
| localstack | 1.22 MB/s | 13.5ms | 13.9ms | 13.9ms | 0 |

```
  rustfs       ████████████████████████████████████████ 7.96 MB/s
  liteio_mem   ███████████████████████████████ 6.27 MB/s
  liteio       █████████████████████████████ 5.90 MB/s
  seaweedfs    ████████████████████ 4.15 MB/s
  minio        █████████████████ 3.49 MB/s
  localstack   ██████ 1.22 MB/s
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 4.59 MB/s | 3.9ms | 4.5ms | 4.5ms | 0 |
| seaweedfs | 4.34 MB/s | 3.4ms | 7.3ms | 7.3ms | 0 |
| rustfs | 4.34 MB/s | 3.0ms | 6.0ms | 6.0ms | 0 |
| liteio | 3.14 MB/s | 4.4ms | 7.5ms | 7.5ms | 0 |
| minio | 2.23 MB/s | 8.8ms | 9.8ms | 9.8ms | 0 |
| localstack | 1.21 MB/s | 13.0ms | 13.7ms | 13.7ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 4.59 MB/s
  seaweedfs    █████████████████████████████████████ 4.34 MB/s
  rustfs       █████████████████████████████████████ 4.34 MB/s
  liteio       ███████████████████████████ 3.14 MB/s
  minio        ███████████████████ 2.23 MB/s
  localstack   ██████████ 1.21 MB/s
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 165.46 MB/s | 88.7ms | 89.3ms | 89.3ms | 0 |
| rustfs | 164.94 MB/s | 90.7ms | 93.3ms | 93.3ms | 0 |
| liteio | 157.82 MB/s | 93.3ms | 96.9ms | 96.9ms | 0 |
| minio | 154.53 MB/s | 96.3ms | 100.5ms | 100.5ms | 0 |
| seaweedfs | 129.19 MB/s | 111.5ms | 118.9ms | 118.9ms | 0 |
| localstack | 115.88 MB/s | 130.2ms | 131.4ms | 131.4ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 165.46 MB/s
  rustfs       ███████████████████████████████████████ 164.94 MB/s
  liteio       ██████████████████████████████████████ 157.82 MB/s
  minio        █████████████████████████████████████ 154.53 MB/s
  seaweedfs    ███████████████████████████████ 129.19 MB/s
  localstack   ████████████████████████████ 115.88 MB/s
```

### ParallelRead/1KB/C1

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio_mem | 4.32 MB/s | 223.2us | 300.0us | 207.8us | 300.1us | 300.1us | 0 |
| liteio | 3.73 MB/s | 257.0us | 246.1us | 196.5us | 251.4us | 251.4us | 0 |
| minio | 3.11 MB/s | 314.2us | 361.9us | 306.1us | 362.0us | 362.0us | 0 |
| seaweedfs | 2.33 MB/s | 419.2us | 496.7us | 405.6us | 496.8us | 496.8us | 0 |
| rustfs | 1.90 MB/s | 514.9us | 560.5us | 497.4us | 560.8us | 560.8us | 0 |
| localstack | 1.32 MB/s | 739.9us | 838.8us | 701.0us | 839.2us | 839.2us | 0 |

```
  liteio_mem   ████████████████████████████████████████ 4.32 MB/s
  liteio       ██████████████████████████████████ 3.73 MB/s
  minio        ████████████████████████████ 3.11 MB/s
  seaweedfs    █████████████████████ 2.33 MB/s
  rustfs       █████████████████ 1.90 MB/s
  localstack   ████████████ 1.32 MB/s
```

### ParallelRead/1KB/C10

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 1.13 MB/s | 846.0us | 1.1ms | 779.3us | 1.2ms | 1.2ms | 0 |
| minio | 0.96 MB/s | 1.0ms | 1.4ms | 943.2us | 1.4ms | 1.4ms | 0 |
| liteio_mem | 0.80 MB/s | 1.2ms | 1.7ms | 1.2ms | 1.7ms | 1.7ms | 0 |
| rustfs | 0.76 MB/s | 1.3ms | 1.5ms | 1.2ms | 1.5ms | 1.5ms | 0 |
| seaweedfs | 0.75 MB/s | 1.3ms | 1.9ms | 1.1ms | 1.9ms | 1.9ms | 0 |
| localstack | 0.19 MB/s | 5.1ms | 8.0ms | 5.0ms | 8.0ms | 8.0ms | 0 |

```
  liteio       ████████████████████████████████████████ 1.13 MB/s
  minio        ██████████████████████████████████ 0.96 MB/s
  liteio_mem   ████████████████████████████ 0.80 MB/s
  rustfs       ███████████████████████████ 0.76 MB/s
  seaweedfs    ██████████████████████████ 0.75 MB/s
  localstack   ██████ 0.19 MB/s
```

### ParallelRead/1KB/C50

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 0.66 MB/s | 1.5ms | 1.7ms | 1.5ms | 1.7ms | 1.7ms | 0 |
| liteio | 0.53 MB/s | 1.9ms | 3.9ms | 1.4ms | 3.9ms | 3.9ms | 0 |
| liteio_mem | 0.51 MB/s | 1.9ms | 2.6ms | 2.0ms | 2.6ms | 2.6ms | 0 |
| seaweedfs | 0.46 MB/s | 2.1ms | 2.5ms | 2.1ms | 2.5ms | 2.5ms | 0 |
| localstack | 0.09 MB/s | 10.5ms | 12.8ms | 12.0ms | 12.8ms | 12.8ms | 0 |

```
  minio        ████████████████████████████████████████ 0.66 MB/s
  liteio       ███████████████████████████████ 0.53 MB/s
  liteio_mem   ██████████████████████████████ 0.51 MB/s
  seaweedfs    ███████████████████████████ 0.46 MB/s
  localstack   █████ 0.09 MB/s
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 1.43 MB/s | 668.2us | 827.0us | 827.0us | 0 |
| rustfs | 1.37 MB/s | 682.0us | 954.0us | 954.0us | 0 |
| liteio | 1.36 MB/s | 671.8us | 944.2us | 944.2us | 0 |
| liteio_mem | 1.32 MB/s | 658.8us | 972.7us | 972.7us | 0 |
| localstack | 1.26 MB/s | 734.5us | 1.2ms | 1.2ms | 0 |
| minio | 1.03 MB/s | 874.8us | 1.3ms | 1.3ms | 0 |

```
  seaweedfs    ████████████████████████████████████████ 1.43 MB/s
  rustfs       ██████████████████████████████████████ 1.37 MB/s
  liteio       █████████████████████████████████████ 1.36 MB/s
  liteio_mem   ████████████████████████████████████ 1.32 MB/s
  localstack   ███████████████████████████████████ 1.26 MB/s
  minio        ████████████████████████████ 1.03 MB/s
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 0.34 MB/s | 2.5ms | 5.3ms | 5.3ms | 0 |
| liteio | 0.31 MB/s | 2.9ms | 4.7ms | 4.7ms | 0 |
| seaweedfs | 0.30 MB/s | 2.5ms | 5.0ms | 5.0ms | 0 |
| liteio_mem | 0.30 MB/s | 2.3ms | 5.6ms | 5.6ms | 0 |
| minio | 0.27 MB/s | 2.9ms | 6.2ms | 6.2ms | 0 |
| localstack | 0.17 MB/s | 4.6ms | 9.5ms | 9.5ms | 0 |

```
  rustfs       ████████████████████████████████████████ 0.34 MB/s
  liteio       █████████████████████████████████████ 0.31 MB/s
  seaweedfs    ████████████████████████████████████ 0.30 MB/s
  liteio_mem   ███████████████████████████████████ 0.30 MB/s
  minio        ███████████████████████████████ 0.27 MB/s
  localstack   ████████████████████ 0.17 MB/s
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 0.23 MB/s | 4.4ms | 5.1ms | 5.1ms | 0 |
| minio | 0.19 MB/s | 5.1ms | 6.0ms | 6.0ms | 0 |
| liteio | 0.19 MB/s | 5.0ms | 6.3ms | 6.3ms | 0 |
| liteio_mem | 0.18 MB/s | 5.2ms | 6.8ms | 6.8ms | 0 |
| localstack | 0.10 MB/s | 11.5ms | 12.5ms | 12.5ms | 0 |

```
  seaweedfs    ████████████████████████████████████████ 0.23 MB/s
  minio        ██████████████████████████████████ 0.19 MB/s
  liteio       ██████████████████████████████████ 0.19 MB/s
  liteio_mem   ███████████████████████████████ 0.18 MB/s
  localstack   █████████████████ 0.10 MB/s
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 211.91 MB/s | 1.0ms | 1.6ms | 1.6ms | 0 |
| liteio_mem | 211.35 MB/s | 1.1ms | 1.4ms | 1.4ms | 0 |
| seaweedfs | 192.91 MB/s | 1.3ms | 1.4ms | 1.4ms | 0 |
| localstack | 154.78 MB/s | 1.5ms | 1.9ms | 1.9ms | 0 |
| minio | 147.58 MB/s | 1.5ms | 3.3ms | 3.3ms | 0 |
| rustfs | 122.45 MB/s | 2.0ms | 2.4ms | 2.4ms | 0 |

```
  liteio       ████████████████████████████████████████ 211.91 MB/s
  liteio_mem   ███████████████████████████████████████ 211.35 MB/s
  seaweedfs    ████████████████████████████████████ 192.91 MB/s
  localstack   █████████████████████████████ 154.78 MB/s
  minio        ███████████████████████████ 147.58 MB/s
  rustfs       ███████████████████████ 122.45 MB/s
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 224.74 MB/s | 1.0ms | 1.2ms | 1.2ms | 0 |
| liteio | 223.77 MB/s | 1.0ms | 1.4ms | 1.4ms | 0 |
| seaweedfs | 190.83 MB/s | 1.3ms | 1.5ms | 1.5ms | 0 |
| localstack | 162.74 MB/s | 1.5ms | 1.7ms | 1.7ms | 0 |
| minio | 151.87 MB/s | 1.6ms | 1.9ms | 1.9ms | 0 |
| rustfs | 118.08 MB/s | 2.1ms | 2.3ms | 2.3ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 224.74 MB/s
  liteio       ███████████████████████████████████████ 223.77 MB/s
  seaweedfs    █████████████████████████████████ 190.83 MB/s
  localstack   ████████████████████████████ 162.74 MB/s
  minio        ███████████████████████████ 151.87 MB/s
  rustfs       █████████████████████ 118.08 MB/s
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 230.99 MB/s | 1.0ms | 1.3ms | 1.3ms | 0 |
| liteio | 225.95 MB/s | 1.0ms | 1.5ms | 1.5ms | 0 |
| seaweedfs | 187.34 MB/s | 1.2ms | 1.6ms | 1.6ms | 0 |
| minio | 157.81 MB/s | 1.6ms | 1.8ms | 1.8ms | 0 |
| localstack | 157.54 MB/s | 1.6ms | 1.8ms | 1.8ms | 0 |
| rustfs | 124.84 MB/s | 2.0ms | 2.2ms | 2.2ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 230.99 MB/s
  liteio       ███████████████████████████████████████ 225.95 MB/s
  seaweedfs    ████████████████████████████████ 187.34 MB/s
  minio        ███████████████████████████ 157.81 MB/s
  localstack   ███████████████████████████ 157.54 MB/s
  rustfs       █████████████████████ 124.84 MB/s
```

### Read/10MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 313.03 MB/s | 461.1us | 688.0us | 31.2ms | 34.4ms | 34.4ms | 0 |
| liteio_mem | 311.91 MB/s | 428.4us | 481.5us | 31.4ms | 34.8ms | 34.8ms | 0 |
| minio | 304.02 MB/s | 1.1ms | 1.3ms | 32.2ms | 34.6ms | 34.6ms | 0 |
| localstack | 285.14 MB/s | 1.4ms | 1.8ms | 34.8ms | 37.4ms | 37.4ms | 0 |
| seaweedfs | 279.77 MB/s | 2.2ms | 2.5ms | 34.6ms | 39.9ms | 39.9ms | 0 |
| rustfs | 270.00 MB/s | 6.1ms | 8.3ms | 36.5ms | 40.1ms | 40.1ms | 0 |

```
  liteio       ████████████████████████████████████████ 313.03 MB/s
  liteio_mem   ███████████████████████████████████████ 311.91 MB/s
  minio        ██████████████████████████████████████ 304.02 MB/s
  localstack   ████████████████████████████████████ 285.14 MB/s
  seaweedfs    ███████████████████████████████████ 279.77 MB/s
  rustfs       ██████████████████████████████████ 270.00 MB/s
```

### Read/1KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio_mem | 5.16 MB/s | 185.2us | 206.2us | 192.9us | 214.2us | 214.2us | 0 |
| liteio | 4.66 MB/s | 209.4us | 239.6us | 206.1us | 239.7us | 239.7us | 0 |
| minio | 3.10 MB/s | 314.5us | 347.3us | 306.0us | 347.4us | 347.4us | 0 |
| seaweedfs | 2.38 MB/s | 411.0us | 447.0us | 403.3us | 447.1us | 447.1us | 0 |
| rustfs | 2.18 MB/s | 447.3us | 500.6us | 440.8us | 500.7us | 500.7us | 0 |
| localstack | 1.42 MB/s | 687.2us | 755.0us | 678.7us | 755.1us | 755.1us | 0 |

```
  liteio_mem   ████████████████████████████████████████ 5.16 MB/s
  liteio       ████████████████████████████████████ 4.66 MB/s
  minio        ████████████████████████ 3.10 MB/s
  seaweedfs    ██████████████████ 2.38 MB/s
  rustfs       ████████████████ 2.18 MB/s
  localstack   ███████████ 1.42 MB/s
```

### Read/1MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 302.97 MB/s | 285.9us | 470.3us | 3.3ms | 3.4ms | 3.4ms | 0 |
| liteio_mem | 294.05 MB/s | 277.0us | 396.8us | 3.4ms | 3.6ms | 3.6ms | 0 |
| minio | 257.74 MB/s | 753.9us | 886.9us | 3.8ms | 4.1ms | 4.1ms | 0 |
| seaweedfs | 252.00 MB/s | 908.7us | 1.1ms | 3.9ms | 4.3ms | 4.3ms | 0 |
| localstack | 242.87 MB/s | 1.1ms | 1.3ms | 4.1ms | 4.5ms | 4.5ms | 0 |
| rustfs | 207.42 MB/s | 1.8ms | 2.0ms | 4.7ms | 5.2ms | 5.2ms | 0 |

```
  liteio       ████████████████████████████████████████ 302.97 MB/s
  liteio_mem   ██████████████████████████████████████ 294.05 MB/s
  minio        ██████████████████████████████████ 257.74 MB/s
  seaweedfs    █████████████████████████████████ 252.00 MB/s
  localstack   ████████████████████████████████ 242.87 MB/s
  rustfs       ███████████████████████████ 207.42 MB/s
```

### Read/64KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 161.32 MB/s | 209.6us | 253.4us | 375.7us | 451.8us | 451.8us | 0 |
| liteio_mem | 161.03 MB/s | 206.3us | 244.8us | 383.5us | 443.0us | 443.0us | 0 |
| minio | 107.48 MB/s | 404.9us | 447.5us | 580.3us | 626.5us | 626.5us | 0 |
| seaweedfs | 94.85 MB/s | 461.7us | 533.7us | 650.8us | 719.2us | 719.2us | 0 |
| rustfs | 87.04 MB/s | 596.5us | 739.5us | 709.2us | 757.8us | 757.8us | 0 |
| localstack | 67.12 MB/s | 799.7us | 842.5us | 891.8us | 979.5us | 979.5us | 0 |

```
  liteio       ████████████████████████████████████████ 161.32 MB/s
  liteio_mem   ███████████████████████████████████████ 161.03 MB/s
  minio        ██████████████████████████ 107.48 MB/s
  seaweedfs    ███████████████████████ 94.85 MB/s
  rustfs       █████████████████████ 87.04 MB/s
  localstack   ████████████████ 67.12 MB/s
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 5908 ops/s | 173.7us | 201.0us | 201.0us | 0 |
| liteio | 5714 ops/s | 161.2us | 218.9us | 218.9us | 0 |
| minio | 4285 ops/s | 229.9us | 272.7us | 272.7us | 0 |
| seaweedfs | 3793 ops/s | 257.9us | 299.6us | 299.6us | 0 |
| rustfs | 3473 ops/s | 283.7us | 332.6us | 332.6us | 0 |
| localstack | 1518 ops/s | 648.9us | 739.0us | 739.0us | 0 |

```
  liteio_mem   ████████████████████████████████████████ 5908 ops/s
  liteio       ██████████████████████████████████████ 5714 ops/s
  minio        █████████████████████████████ 4285 ops/s
  seaweedfs    █████████████████████████ 3793 ops/s
  rustfs       ███████████████████████ 3473 ops/s
  localstack   ██████████ 1518 ops/s
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 187.92 MB/s | 53.2ms | 55.3ms | 55.3ms | 0 |
| rustfs | 184.83 MB/s | 52.7ms | 62.7ms | 62.7ms | 0 |
| liteio | 182.53 MB/s | 53.9ms | 59.0ms | 59.0ms | 0 |
| minio | 170.19 MB/s | 56.5ms | 67.6ms | 67.6ms | 0 |
| seaweedfs | 147.99 MB/s | 64.8ms | 80.4ms | 80.4ms | 0 |
| localstack | 145.30 MB/s | 68.5ms | 72.1ms | 72.1ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 187.92 MB/s
  rustfs       ███████████████████████████████████████ 184.83 MB/s
  liteio       ██████████████████████████████████████ 182.53 MB/s
  minio        ████████████████████████████████████ 170.19 MB/s
  seaweedfs    ███████████████████████████████ 147.99 MB/s
  localstack   ██████████████████████████████ 145.30 MB/s
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 1.33 MB/s | 732.0us | 830.9us | 830.9us | 0 |
| localstack | 1.28 MB/s | 755.8us | 831.0us | 831.0us | 0 |
| rustfs | 1.23 MB/s | 758.3us | 934.5us | 934.5us | 0 |
| liteio | 0.72 MB/s | 1.2ms | 2.0ms | 2.0ms | 0 |
| minio | 0.70 MB/s | 1.4ms | 1.6ms | 1.6ms | 0 |
| liteio_mem | 0.52 MB/s | 764.4us | 5.8ms | 5.8ms | 0 |

```
  seaweedfs    ████████████████████████████████████████ 1.33 MB/s
  localstack   ██████████████████████████████████████ 1.28 MB/s
  rustfs       █████████████████████████████████████ 1.23 MB/s
  liteio       █████████████████████ 0.72 MB/s
  minio        █████████████████████ 0.70 MB/s
  liteio_mem   ███████████████ 0.52 MB/s
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 161.41 MB/s | 6.0ms | 6.6ms | 6.6ms | 0 |
| liteio | 160.53 MB/s | 6.0ms | 8.2ms | 8.2ms | 0 |
| rustfs | 148.67 MB/s | 6.1ms | 8.0ms | 8.0ms | 0 |
| minio | 139.63 MB/s | 6.9ms | 8.4ms | 8.4ms | 0 |
| seaweedfs | 128.03 MB/s | 7.8ms | 8.1ms | 8.1ms | 0 |
| localstack | 124.68 MB/s | 7.9ms | 8.6ms | 8.6ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 161.41 MB/s
  liteio       ███████████████████████████████████████ 160.53 MB/s
  rustfs       ████████████████████████████████████ 148.67 MB/s
  minio        ██████████████████████████████████ 139.63 MB/s
  seaweedfs    ███████████████████████████████ 128.03 MB/s
  localstack   ██████████████████████████████ 124.68 MB/s
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 65.27 MB/s | 933.9us | 1.2ms | 1.2ms | 0 |
| localstack | 59.19 MB/s | 1.0ms | 1.1ms | 1.1ms | 0 |
| rustfs | 58.53 MB/s | 1.0ms | 1.3ms | 1.3ms | 0 |
| seaweedfs | 58.08 MB/s | 1.1ms | 1.2ms | 1.2ms | 0 |
| liteio | 55.34 MB/s | 1.1ms | 1.4ms | 1.4ms | 0 |
| minio | 40.48 MB/s | 1.5ms | 2.0ms | 2.0ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 65.27 MB/s
  localstack   ████████████████████████████████████ 59.19 MB/s
  rustfs       ███████████████████████████████████ 58.53 MB/s
  seaweedfs    ███████████████████████████████████ 58.08 MB/s
  liteio       █████████████████████████████████ 55.34 MB/s
  minio        ████████████████████████ 40.48 MB/s
```

## Recommendations

- **Best for write-heavy workloads:** liteio_mem
- **Best for read-heavy workloads:** liteio

---

*Report generated by storage benchmark CLI*
