# Storage Benchmark Report

**Generated:** 2026-01-15T01:18:25+07:00

**Go Version:** go1.25.5

**Platform:** darwin/arm64

## Executive Summary

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (1MB+) | **rustfs** | 112 MB/s | Best for media, backups |
| Large File Downloads | **minio** | 179 MB/s | Best for streaming, CDN |
| Small File Operations | **minio** | 1532 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **rustfs** | - | Best for multi-user apps |

### Large File Performance (1MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| liteio | 24.4 | 132.4 | 12.8ms | 7.4ms |
| liteio_mem | 64.7 | 142.4 | 15.4ms | 6.9ms |
| localstack | 89.8 | 140.7 | 10.3ms | 6.2ms |
| minio | 72.4 | 178.9 | 10.7ms | 5.4ms |
| rustfs | 111.5 | 132.3 | 8.0ms | 6.7ms |
| seaweedfs | 79.5 | 171.9 | 12.4ms | 5.5ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| liteio | 153 | 521 | 1.7ms | 1.7ms |
| liteio_mem | 265 | 541 | 3.7ms | 1.7ms |
| localstack | 665 | 661 | 1.0ms | 1.1ms |
| minio | 518 | 2546 | 1.8ms | 378.9us |
| rustfs | 492 | 1570 | 1.8ms | 624.8us |
| seaweedfs | 740 | 1504 | 844.4us | 586.4us |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| liteio | 555 | 349 | 632 |
| liteio_mem | 516 | 336 | 549 |
| localstack | 667 | 214 | 1055 |
| minio | 1822 | 475 | 1732 |
| rustfs | 1119 | 106 | 797 |
| seaweedfs | 1966 | 409 | 1822 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| liteio | 0.26 | 0.15 | 0.09 | 0.05 | 0.02 | 0.03 |
| liteio_mem | 0.25 | 0.14 | 0.09 | 0.05 | 0.03 | 0.03 |
| localstack | 0.69 | 0.13 | 0.06 | 0.03 | 0.02 | 0.01 |
| minio | 0.52 | 0.18 | 0.11 | 0.07 | 0.04 | 0.03 |
| rustfs | 1.13 | 0.31 | - | - | - | - |
| seaweedfs | 0.91 | 0.34 | 0.20 | 0.12 | 0.06 | 0.07 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| liteio | 0.53 | 0.35 | 0.24 | 0.16 | 0.11 | 0.12 |
| liteio_mem | 0.54 | 0.35 | 0.24 | 0.20 | 0.08 | 0.10 |
| localstack | 0.40 | 0.11 | 0.05 | 0.02 | 0.01 | 0.01 |
| minio | 2.03 | 0.72 | 0.36 | 0.18 | 0.16 | 0.14 |
| rustfs | 1.38 | 0.69 | - | - | - | - |
| seaweedfs | 1.18 | 0.39 | 0.18 | 0.17 | 0.19 | 0.16 |

*\* indicates errors occurred*

---

## Configuration

| Parameter | Value |
|-----------|-------|
| Iterations | 100 |
| Warmup | 10 |
| Concurrency | 200 |
| Timeout | 30s |

## Drivers Tested

- liteio (36 benchmarks)
- liteio_mem (36 benchmarks)
- localstack (36 benchmarks)
- minio (36 benchmarks)
- rustfs (28 benchmarks)
- seaweedfs (36 benchmarks)

## Performance Comparison

### Copy/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| localstack | 1.01 MB/s | 868.2us | 1.5ms | 2.5ms | 0 |
| rustfs | 0.62 MB/s | 1.5ms | 2.3ms | 3.0ms | 0 |
| seaweedfs | 0.26 MB/s | 1.4ms | 3.4ms | 4.6ms | 0 |
| liteio_mem | 0.25 MB/s | 3.8ms | 5.2ms | 5.5ms | 0 |
| liteio | 0.22 MB/s | 4.0ms | 7.2ms | 8.7ms | 0 |
| minio | 0.17 MB/s | 2.8ms | 4.1ms | 21.1ms | 0 |

```
  localstack   ████████████████████████████████████████ 1.01 MB/s
  rustfs       ████████████████████████ 0.62 MB/s
  seaweedfs    ██████████ 0.26 MB/s
  liteio_mem   ██████████ 0.25 MB/s
  liteio       ████████ 0.22 MB/s
  minio        ██████ 0.17 MB/s
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 1822 ops/s | 511.7us | 836.2us | 1.0ms | 0 |
| minio | 1732 ops/s | 490.9us | 1.0ms | 1.9ms | 0 |
| localstack | 1055 ops/s | 751.2us | 1.6ms | 3.4ms | 0 |
| rustfs | 797 ops/s | 1.1ms | 2.0ms | 2.5ms | 0 |
| liteio | 632 ops/s | 1.7ms | 2.6ms | 2.9ms | 0 |
| liteio_mem | 549 ops/s | 1.8ms | 2.4ms | 2.7ms | 0 |

```
  seaweedfs    ████████████████████████████████████████ 1822 ops/s
  minio        ██████████████████████████████████████ 1732 ops/s
  localstack   ███████████████████████ 1055 ops/s
  rustfs       █████████████████ 797 ops/s
  liteio       █████████████ 632 ops/s
  liteio_mem   ████████████ 549 ops/s
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 0.11 MB/s | 772.0us | 1.1ms | 1.5ms | 0 |
| localstack | 0.10 MB/s | 892.0us | 1.2ms | 1.3ms | 0 |
| liteio_mem | 0.08 MB/s | 1.2ms | 1.7ms | 1.7ms | 0 |
| rustfs | 0.08 MB/s | 1.1ms | 1.8ms | 2.6ms | 0 |
| minio | 0.03 MB/s | 3.2ms | 4.7ms | 6.4ms | 0 |
| liteio | 0.03 MB/s | 3.8ms | 5.3ms | 6.1ms | 0 |

```
  seaweedfs    ████████████████████████████████████████ 0.11 MB/s
  localstack   ███████████████████████████████████ 0.10 MB/s
  liteio_mem   ███████████████████████████ 0.08 MB/s
  rustfs       ███████████████████████████ 0.08 MB/s
  minio        ██████████ 0.03 MB/s
  liteio       █████████ 0.03 MB/s
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| localstack | 971 ops/s | 967.9us | 1.2ms | 1.7ms | 0 |
| seaweedfs | 838 ops/s | 816.4us | 1.5ms | 1.9ms | 0 |
| liteio_mem | 747 ops/s | 1.1ms | 2.5ms | 3.5ms | 0 |
| rustfs | 666 ops/s | 1.4ms | 2.1ms | 2.1ms | 0 |
| minio | 323 ops/s | 3.4ms | 4.2ms | 4.7ms | 0 |
| liteio | 237 ops/s | 4.0ms | 5.6ms | 5.9ms | 0 |

```
  localstack   ████████████████████████████████████████ 971 ops/s
  seaweedfs    ██████████████████████████████████ 838 ops/s
  liteio_mem   ██████████████████████████████ 747 ops/s
  rustfs       ███████████████████████████ 666 ops/s
  minio        █████████████ 323 ops/s
  liteio       █████████ 237 ops/s
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| localstack | 0.10 MB/s | 920.8us | 1.5ms | 1.5ms | 0 |
| liteio_mem | 0.08 MB/s | 1.1ms | 1.8ms | 2.1ms | 0 |
| seaweedfs | 0.06 MB/s | 1.4ms | 2.7ms | 3.0ms | 0 |
| rustfs | 0.06 MB/s | 1.5ms | 2.3ms | 3.1ms | 0 |
| liteio | 0.02 MB/s | 3.6ms | 5.1ms | 6.9ms | 0 |
| minio | 0.02 MB/s | 5.0ms | 9.0ms | 14.6ms | 0 |

```
  localstack   ████████████████████████████████████████ 0.10 MB/s
  liteio_mem   ████████████████████████████████ 0.08 MB/s
  seaweedfs    ██████████████████████████ 0.06 MB/s
  rustfs       ████████████████████████ 0.06 MB/s
  liteio       ██████████ 0.02 MB/s
  minio        ███████ 0.02 MB/s
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 475 ops/s | 2.0ms | 2.7ms | 3.3ms | 0 |
| seaweedfs | 409 ops/s | 2.2ms | 4.1ms | 5.2ms | 0 |
| liteio | 349 ops/s | 2.8ms | 3.9ms | 4.8ms | 0 |
| liteio_mem | 336 ops/s | 2.9ms | 4.0ms | 4.2ms | 0 |
| localstack | 214 ops/s | 4.2ms | 7.4ms | 9.1ms | 0 |
| rustfs | 106 ops/s | 8.6ms | 13.2ms | 23.2ms | 0 |

```
  minio        ████████████████████████████████████████ 475 ops/s
  seaweedfs    ██████████████████████████████████ 409 ops/s
  liteio       █████████████████████████████ 349 ops/s
  liteio_mem   ████████████████████████████ 336 ops/s
  localstack   ██████████████████ 214 ops/s
  rustfs       ████████ 106 ops/s
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 6.45 MB/s | 2.4ms | 3.6ms | 6.3ms | 0 |
| seaweedfs | 1.23 MB/s | 13.5ms | 17.3ms | 17.3ms | 0 |
| minio | 0.82 MB/s | 17.4ms | 25.4ms | 26.1ms | 0 |
| liteio | 0.72 MB/s | 25.0ms | 32.0ms | 32.3ms | 0 |
| liteio_mem | 0.71 MB/s | 21.0ms | 35.9ms | 37.6ms | 0 |
| localstack | 0.18 MB/s | 95.2ms | 115.0ms | 115.6ms | 0 |

```
  rustfs       ████████████████████████████████████████ 6.45 MB/s
  seaweedfs    ███████ 1.23 MB/s
  minio        █████ 0.82 MB/s
  liteio       ████ 0.72 MB/s
  liteio_mem   ████ 0.71 MB/s
  localstack   █ 0.18 MB/s
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 6.88 MB/s | 1.8ms | 6.7ms | 7.2ms | 0 |
| seaweedfs | 1.13 MB/s | 14.2ms | 14.9ms | 15.2ms | 0 |
| minio | 0.93 MB/s | 17.6ms | 21.0ms | 21.1ms | 0 |
| liteio_mem | 0.80 MB/s | 20.2ms | 26.1ms | 26.6ms | 0 |
| liteio | 0.71 MB/s | 24.4ms | 29.1ms | 29.5ms | 0 |
| localstack | 0.14 MB/s | 128.6ms | 144.5ms | 145.7ms | 0 |

```
  rustfs       ████████████████████████████████████████ 6.88 MB/s
  seaweedfs    ██████ 1.13 MB/s
  minio        █████ 0.93 MB/s
  liteio_mem   ████ 0.80 MB/s
  liteio       ████ 0.71 MB/s
  localstack   █ 0.14 MB/s
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 4.38 MB/s | 3.4ms | 5.6ms | 6.8ms | 0 |
| seaweedfs | 1.04 MB/s | 15.8ms | 19.5ms | 19.7ms | 0 |
| liteio | 0.52 MB/s | 28.4ms | 50.7ms | 50.8ms | 0 |
| liteio_mem | 0.49 MB/s | 32.6ms | 45.3ms | 48.2ms | 0 |
| minio | 0.48 MB/s | 36.3ms | 43.2ms | 44.0ms | 0 |
| localstack | 0.13 MB/s | 117.0ms | 121.0ms | 121.8ms | 0 |

```
  rustfs       ████████████████████████████████████████ 4.38 MB/s
  seaweedfs    █████████ 1.04 MB/s
  liteio       ████ 0.52 MB/s
  liteio_mem   ████ 0.49 MB/s
  minio        ████ 0.48 MB/s
  localstack   █ 0.13 MB/s
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 101.35 MB/s | 153.2ms | 182.8ms | 182.8ms | 0 |
| localstack | 88.80 MB/s | 160.9ms | 203.5ms | 203.5ms | 0 |
| liteio_mem | 85.28 MB/s | 154.3ms | 274.7ms | 274.7ms | 0 |
| minio | 83.65 MB/s | 176.1ms | 221.6ms | 221.6ms | 0 |
| seaweedfs | 50.11 MB/s | 333.7ms | 484.8ms | 484.8ms | 0 |
| liteio | 47.67 MB/s | 225.2ms | 316.3ms | 316.3ms | 0 |

```
  rustfs       ████████████████████████████████████████ 101.35 MB/s
  localstack   ███████████████████████████████████ 88.80 MB/s
  liteio_mem   █████████████████████████████████ 85.28 MB/s
  minio        █████████████████████████████████ 83.65 MB/s
  seaweedfs    ███████████████████ 50.11 MB/s
  liteio       ██████████████████ 47.67 MB/s
```

### ParallelRead/1KB/C1

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 2.03 MB/s | 481.9us | 676.0us | 457.0us | 676.0us | 899.9us | 0 |
| rustfs | 1.38 MB/s | 706.8us | 1.1ms | 637.8us | 1.1ms | 1.7ms | 0 |
| seaweedfs | 1.18 MB/s | 824.1us | 1.4ms | 737.7us | 1.4ms | 2.2ms | 0 |
| liteio_mem | 0.54 MB/s | 1.8ms | 2.6ms | 1.9ms | 2.6ms | 3.2ms | 0 |
| liteio | 0.53 MB/s | 1.8ms | 2.3ms | 1.8ms | 2.3ms | 2.8ms | 0 |
| localstack | 0.40 MB/s | 2.4ms | 5.6ms | 2.0ms | 5.6ms | 8.7ms | 0 |

```
  minio        ████████████████████████████████████████ 2.03 MB/s
  rustfs       ███████████████████████████ 1.38 MB/s
  seaweedfs    ███████████████████████ 1.18 MB/s
  liteio_mem   ██████████ 0.54 MB/s
  liteio       ██████████ 0.53 MB/s
  localstack   ███████ 0.40 MB/s
```

### ParallelRead/1KB/C10

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 0.72 MB/s | 1.4ms | 2.1ms | 1.2ms | 2.1ms | 3.0ms | 0 |
| rustfs | 0.69 MB/s | 1.4ms | 2.9ms | 1.2ms | 2.9ms | 3.7ms | 0 |
| seaweedfs | 0.39 MB/s | 2.5ms | 4.2ms | 2.1ms | 4.2ms | 4.5ms | 0 |
| liteio | 0.35 MB/s | 2.7ms | 4.0ms | 2.4ms | 4.0ms | 4.8ms | 0 |
| liteio_mem | 0.35 MB/s | 2.8ms | 4.1ms | 2.6ms | 4.1ms | 5.1ms | 0 |
| localstack | 0.11 MB/s | 9.2ms | 14.9ms | 9.0ms | 14.9ms | 18.7ms | 0 |

```
  minio        ████████████████████████████████████████ 0.72 MB/s
  rustfs       ██████████████████████████████████████ 0.69 MB/s
  seaweedfs    █████████████████████ 0.39 MB/s
  liteio       ███████████████████ 0.35 MB/s
  liteio_mem   ███████████████████ 0.35 MB/s
  localstack   █████ 0.11 MB/s
```

### ParallelRead/1KB/C100

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| seaweedfs | 0.19 MB/s | 5.1ms | 7.0ms | 5.0ms | 7.0ms | 7.5ms | 0 |
| minio | 0.16 MB/s | 6.2ms | 7.5ms | 6.2ms | 7.5ms | 7.7ms | 0 |
| liteio | 0.11 MB/s | 9.1ms | 13.5ms | 8.5ms | 13.5ms | 13.8ms | 0 |
| liteio_mem | 0.08 MB/s | 11.9ms | 15.7ms | 12.2ms | 15.7ms | 16.7ms | 0 |
| localstack | 0.01 MB/s | 77.3ms | 83.3ms | 79.8ms | 83.3ms | 83.7ms | 0 |

```
  seaweedfs    ████████████████████████████████████████ 0.19 MB/s
  minio        ████████████████████████████████ 0.16 MB/s
  liteio       ██████████████████████ 0.11 MB/s
  liteio_mem   █████████████████ 0.08 MB/s
  localstack   ██ 0.01 MB/s
```

### ParallelRead/1KB/C200

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| seaweedfs | 0.16 MB/s | 6.1ms | 7.3ms | 6.1ms | 7.3ms | 7.5ms | 0 |
| minio | 0.14 MB/s | 6.9ms | 8.2ms | 7.1ms | 8.2ms | 8.3ms | 0 |
| liteio | 0.12 MB/s | 8.1ms | 10.7ms | 8.1ms | 10.7ms | 10.9ms | 0 |
| liteio_mem | 0.10 MB/s | 10.2ms | 13.8ms | 10.4ms | 13.8ms | 14.0ms | 0 |
| localstack | 0.01 MB/s | 72.0ms | 87.5ms | 60.7ms | 87.5ms | 87.7ms | 0 |

```
  seaweedfs    ████████████████████████████████████████ 0.16 MB/s
  minio        ███████████████████████████████████ 0.14 MB/s
  liteio       █████████████████████████████ 0.12 MB/s
  liteio_mem   ███████████████████████ 0.10 MB/s
  localstack   ███ 0.01 MB/s
```

### ParallelRead/1KB/C25

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 0.36 MB/s | 2.7ms | 3.6ms | 2.5ms | 3.6ms | 6.9ms | 0 |
| liteio | 0.24 MB/s | 4.0ms | 7.6ms | 3.9ms | 7.6ms | 7.8ms | 0 |
| liteio_mem | 0.24 MB/s | 4.1ms | 7.4ms | 3.8ms | 7.4ms | 8.4ms | 0 |
| seaweedfs | 0.18 MB/s | 5.4ms | 14.4ms | 2.6ms | 14.4ms | 14.8ms | 0 |
| localstack | 0.05 MB/s | 18.2ms | 32.2ms | 15.1ms | 32.2ms | 35.4ms | 0 |

```
  minio        ████████████████████████████████████████ 0.36 MB/s
  liteio       ██████████████████████████ 0.24 MB/s
  liteio_mem   ██████████████████████████ 0.24 MB/s
  seaweedfs    ███████████████████ 0.18 MB/s
  localstack   █████ 0.05 MB/s
```

### ParallelRead/1KB/C50

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio_mem | 0.20 MB/s | 4.8ms | 7.4ms | 4.6ms | 7.4ms | 7.6ms | 0 |
| minio | 0.18 MB/s | 5.4ms | 8.1ms | 5.0ms | 8.1ms | 8.2ms | 0 |
| seaweedfs | 0.17 MB/s | 5.8ms | 8.1ms | 5.7ms | 8.1ms | 8.2ms | 0 |
| liteio | 0.16 MB/s | 6.3ms | 11.5ms | 5.4ms | 11.5ms | 12.4ms | 0 |
| localstack | 0.02 MB/s | 46.4ms | 57.8ms | 41.8ms | 57.8ms | 57.9ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 0.20 MB/s
  minio        ███████████████████████████████████ 0.18 MB/s
  seaweedfs    █████████████████████████████████ 0.17 MB/s
  liteio       ██████████████████████████████ 0.16 MB/s
  localstack   ████ 0.02 MB/s
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 1.13 MB/s | 787.8us | 1.5ms | 1.9ms | 0 |
| seaweedfs | 0.91 MB/s | 899.2us | 1.9ms | 2.5ms | 0 |
| localstack | 0.69 MB/s | 1.2ms | 3.0ms | 3.9ms | 0 |
| minio | 0.52 MB/s | 1.8ms | 3.3ms | 4.3ms | 0 |
| liteio | 0.26 MB/s | 3.7ms | 5.0ms | 5.9ms | 0 |
| liteio_mem | 0.25 MB/s | 3.7ms | 5.2ms | 5.7ms | 0 |

```
  rustfs       ████████████████████████████████████████ 1.13 MB/s
  seaweedfs    ████████████████████████████████ 0.91 MB/s
  localstack   ████████████████████████ 0.69 MB/s
  minio        ██████████████████ 0.52 MB/s
  liteio       █████████ 0.26 MB/s
  liteio_mem   █████████ 0.25 MB/s
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 0.34 MB/s | 2.8ms | 4.2ms | 5.0ms | 0 |
| rustfs | 0.31 MB/s | 3.0ms | 4.7ms | 5.6ms | 0 |
| minio | 0.18 MB/s | 5.0ms | 8.4ms | 9.6ms | 0 |
| liteio | 0.15 MB/s | 6.1ms | 10.2ms | 11.2ms | 0 |
| liteio_mem | 0.14 MB/s | 6.5ms | 10.9ms | 12.0ms | 0 |
| localstack | 0.13 MB/s | 7.5ms | 11.9ms | 14.1ms | 0 |

```
  seaweedfs    ████████████████████████████████████████ 0.34 MB/s
  rustfs       ████████████████████████████████████ 0.31 MB/s
  minio        █████████████████████ 0.18 MB/s
  liteio       █████████████████ 0.15 MB/s
  liteio_mem   ████████████████ 0.14 MB/s
  localstack   ██████████████ 0.13 MB/s
```

### ParallelWrite/1KB/C100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 0.06 MB/s | 14.6ms | 21.3ms | 21.5ms | 0 |
| minio | 0.04 MB/s | 23.7ms | 32.3ms | 33.1ms | 0 |
| liteio_mem | 0.03 MB/s | 38.9ms | 57.2ms | 59.0ms | 0 |
| liteio | 0.02 MB/s | 51.0ms | 67.2ms | 69.3ms | 0 |
| localstack | 0.02 MB/s | 66.4ms | 69.7ms | 71.9ms | 0 |

```
  seaweedfs    ████████████████████████████████████████ 0.06 MB/s
  minio        █████████████████████████ 0.04 MB/s
  liteio_mem   ███████████████ 0.03 MB/s
  liteio       ████████████ 0.02 MB/s
  localstack   ██████████ 0.02 MB/s
```

### ParallelWrite/1KB/C200

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 0.07 MB/s | 13.9ms | 18.4ms | 19.2ms | 0 |
| liteio_mem | 0.03 MB/s | 28.5ms | 52.6ms | 53.9ms | 0 |
| liteio | 0.03 MB/s | 29.4ms | 53.0ms | 53.5ms | 0 |
| minio | 0.03 MB/s | 24.9ms | 51.6ms | 52.0ms | 0 |
| localstack | 0.01 MB/s | 110.0ms | 116.1ms | 117.1ms | 0 |

```
  seaweedfs    ████████████████████████████████████████ 0.07 MB/s
  liteio_mem   ██████████████████ 0.03 MB/s
  liteio       █████████████████ 0.03 MB/s
  minio        █████████████████ 0.03 MB/s
  localstack   █████ 0.01 MB/s
```

### ParallelWrite/1KB/C25

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 0.20 MB/s | 4.7ms | 7.8ms | 8.4ms | 0 |
| minio | 0.11 MB/s | 8.4ms | 15.9ms | 20.1ms | 0 |
| liteio | 0.09 MB/s | 10.0ms | 18.8ms | 20.7ms | 0 |
| liteio_mem | 0.09 MB/s | 10.0ms | 17.2ms | 18.3ms | 0 |
| localstack | 0.06 MB/s | 14.9ms | 36.2ms | 36.4ms | 0 |

```
  seaweedfs    ████████████████████████████████████████ 0.20 MB/s
  minio        ████████████████████ 0.11 MB/s
  liteio       █████████████████ 0.09 MB/s
  liteio_mem   █████████████████ 0.09 MB/s
  localstack   ███████████ 0.06 MB/s
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 0.12 MB/s | 7.8ms | 12.0ms | 12.7ms | 0 |
| minio | 0.07 MB/s | 13.5ms | 24.4ms | 30.2ms | 0 |
| liteio | 0.05 MB/s | 15.7ms | 40.7ms | 41.9ms | 0 |
| liteio_mem | 0.05 MB/s | 20.4ms | 41.6ms | 45.7ms | 0 |
| localstack | 0.03 MB/s | 33.1ms | 80.4ms | 80.7ms | 0 |

```
  seaweedfs    ████████████████████████████████████████ 0.12 MB/s
  minio        ███████████████████████ 0.07 MB/s
  liteio       ███████████████ 0.05 MB/s
  liteio_mem   ███████████████ 0.05 MB/s
  localstack   ████████ 0.03 MB/s
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 108.61 MB/s | 2.2ms | 3.2ms | 3.9ms | 0 |
| minio | 103.42 MB/s | 2.1ms | 4.6ms | 6.1ms | 0 |
| liteio | 83.49 MB/s | 2.9ms | 3.9ms | 4.4ms | 0 |
| liteio_mem | 72.16 MB/s | 3.3ms | 4.8ms | 5.1ms | 0 |
| rustfs | 71.91 MB/s | 3.1ms | 6.2ms | 7.9ms | 0 |
| localstack | 70.99 MB/s | 3.3ms | 5.5ms | 6.3ms | 0 |

```
  seaweedfs    ████████████████████████████████████████ 108.61 MB/s
  minio        ██████████████████████████████████████ 103.42 MB/s
  liteio       ██████████████████████████████ 83.49 MB/s
  liteio_mem   ██████████████████████████ 72.16 MB/s
  rustfs       ██████████████████████████ 71.91 MB/s
  localstack   ██████████████████████████ 70.99 MB/s
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 94.05 MB/s | 2.2ms | 4.7ms | 9.8ms | 0 |
| seaweedfs | 91.65 MB/s | 2.5ms | 4.5ms | 6.0ms | 0 |
| liteio_mem | 83.38 MB/s | 3.0ms | 3.6ms | 3.8ms | 0 |
| rustfs | 80.47 MB/s | 2.9ms | 4.7ms | 5.3ms | 0 |
| liteio | 79.42 MB/s | 2.9ms | 4.8ms | 5.2ms | 0 |
| localstack | 76.83 MB/s | 2.7ms | 6.9ms | 8.4ms | 0 |

```
  minio        ████████████████████████████████████████ 94.05 MB/s
  seaweedfs    ██████████████████████████████████████ 91.65 MB/s
  liteio_mem   ███████████████████████████████████ 83.38 MB/s
  rustfs       ██████████████████████████████████ 80.47 MB/s
  liteio       █████████████████████████████████ 79.42 MB/s
  localstack   ████████████████████████████████ 76.83 MB/s
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 114.48 MB/s | 1.9ms | 4.5ms | 6.5ms | 0 |
| minio | 106.85 MB/s | 2.1ms | 4.1ms | 6.1ms | 0 |
| liteio | 85.86 MB/s | 2.8ms | 3.7ms | 4.4ms | 0 |
| liteio_mem | 77.38 MB/s | 3.1ms | 4.1ms | 5.4ms | 0 |
| rustfs | 70.53 MB/s | 3.3ms | 5.7ms | 7.0ms | 0 |
| localstack | 39.47 MB/s | 2.2ms | 6.5ms | 22.4ms | 0 |

```
  seaweedfs    ████████████████████████████████████████ 114.48 MB/s
  minio        █████████████████████████████████████ 106.85 MB/s
  liteio       ██████████████████████████████ 85.86 MB/s
  liteio_mem   ███████████████████████████ 77.38 MB/s
  rustfs       ████████████████████████ 70.53 MB/s
  localstack   █████████████ 39.47 MB/s
```

### Read/100MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| seaweedfs | 206.74 MB/s | 4.9ms | 4.5ms | 486.8ms | 490.0ms | 490.0ms | 0 |
| minio | 206.31 MB/s | 2.4ms | 2.4ms | 475.5ms | 492.6ms | 492.6ms | 0 |
| liteio | 194.37 MB/s | 2.6ms | 2.8ms | 517.9ms | 523.0ms | 523.0ms | 0 |
| liteio_mem | 184.17 MB/s | 2.4ms | 2.6ms | 538.3ms | 548.3ms | 548.3ms | 0 |
| rustfs | 174.69 MB/s | 6.1ms | 5.4ms | 478.5ms | 488.3ms | 488.3ms | 0 |
| localstack | 170.30 MB/s | 4.5ms | 3.2ms | 569.3ms | 621.0ms | 621.0ms | 0 |

```
  seaweedfs    ████████████████████████████████████████ 206.74 MB/s
  minio        ███████████████████████████████████████ 206.31 MB/s
  liteio       █████████████████████████████████████ 194.37 MB/s
  liteio_mem   ███████████████████████████████████ 184.17 MB/s
  rustfs       █████████████████████████████████ 174.69 MB/s
  localstack   ████████████████████████████████ 170.30 MB/s
```

### Read/10MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 235.89 MB/s | 1.2ms | 1.3ms | 42.2ms | 43.0ms | 43.0ms | 0 |
| seaweedfs | 200.06 MB/s | 3.2ms | 3.5ms | 50.3ms | 52.8ms | 52.8ms | 0 |
| liteio_mem | 191.21 MB/s | 2.0ms | 2.4ms | 50.3ms | 55.5ms | 55.5ms | 0 |
| liteio | 189.45 MB/s | 2.2ms | 2.9ms | 50.9ms | 60.9ms | 60.9ms | 0 |
| rustfs | 181.69 MB/s | 8.2ms | 8.7ms | 52.6ms | 58.2ms | 58.2ms | 0 |
| localstack | 159.86 MB/s | 3.7ms | 4.6ms | 59.6ms | 77.9ms | 77.9ms | 0 |

```
  minio        ████████████████████████████████████████ 235.89 MB/s
  seaweedfs    █████████████████████████████████ 200.06 MB/s
  liteio_mem   ████████████████████████████████ 191.21 MB/s
  liteio       ████████████████████████████████ 189.45 MB/s
  rustfs       ██████████████████████████████ 181.69 MB/s
  localstack   ███████████████████████████ 159.86 MB/s
```

### Read/1KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 2.49 MB/s | 392.5us | 476.4us | 378.9us | 476.6us | 508.7us | 0 |
| rustfs | 1.53 MB/s | 636.7us | 777.2us | 624.8us | 777.4us | 848.4us | 0 |
| seaweedfs | 1.47 MB/s | 664.8us | 1.3ms | 586.4us | 1.3ms | 1.6ms | 0 |
| localstack | 0.65 MB/s | 1.5ms | 2.4ms | 1.1ms | 2.4ms | 9.2ms | 0 |
| liteio_mem | 0.53 MB/s | 1.8ms | 2.6ms | 1.7ms | 2.6ms | 3.0ms | 0 |
| liteio | 0.51 MB/s | 1.9ms | 3.3ms | 1.7ms | 3.3ms | 3.9ms | 0 |

```
  minio        ████████████████████████████████████████ 2.49 MB/s
  rustfs       ████████████████████████ 1.53 MB/s
  seaweedfs    ███████████████████████ 1.47 MB/s
  localstack   ██████████ 0.65 MB/s
  liteio_mem   ████████ 0.53 MB/s
  liteio       ████████ 0.51 MB/s
```

### Read/1MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 178.90 MB/s | 1.3ms | 1.9ms | 5.4ms | 6.7ms | 6.7ms | 0 |
| seaweedfs | 171.86 MB/s | 1.9ms | 2.8ms | 5.5ms | 7.0ms | 7.0ms | 0 |
| liteio_mem | 142.42 MB/s | 1.7ms | 2.3ms | 6.9ms | 9.1ms | 9.1ms | 0 |
| localstack | 140.72 MB/s | 2.0ms | 3.1ms | 6.2ms | 12.0ms | 12.0ms | 0 |
| liteio | 132.38 MB/s | 1.8ms | 2.5ms | 7.4ms | 9.5ms | 9.5ms | 0 |
| rustfs | 132.29 MB/s | 3.2ms | 6.9ms | 6.7ms | 11.0ms | 11.0ms | 0 |

```
  minio        ████████████████████████████████████████ 178.90 MB/s
  seaweedfs    ██████████████████████████████████████ 171.86 MB/s
  liteio_mem   ███████████████████████████████ 142.42 MB/s
  localstack   ███████████████████████████████ 140.72 MB/s
  liteio       █████████████████████████████ 132.38 MB/s
  rustfs       █████████████████████████████ 132.29 MB/s
```

### Read/64KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 74.37 MB/s | 650.8us | 927.8us | 764.2us | 1.2ms | 1.4ms | 0 |
| seaweedfs | 66.18 MB/s | 746.9us | 920.8us | 919.8us | 1.1ms | 1.3ms | 0 |
| rustfs | 56.15 MB/s | 968.8us | 2.1ms | 975.1us | 2.1ms | 2.2ms | 0 |
| localstack | 43.67 MB/s | 1.2ms | 1.5ms | 1.3ms | 2.1ms | 2.4ms | 0 |
| liteio | 31.24 MB/s | 1.8ms | 2.2ms | 1.9ms | 2.6ms | 3.2ms | 0 |
| liteio_mem | 30.01 MB/s | 1.9ms | 2.3ms | 2.0ms | 2.6ms | 2.8ms | 0 |

```
  minio        ████████████████████████████████████████ 74.37 MB/s
  seaweedfs    ███████████████████████████████████ 66.18 MB/s
  rustfs       ██████████████████████████████ 56.15 MB/s
  localstack   ███████████████████████ 43.67 MB/s
  liteio       ████████████████ 31.24 MB/s
  liteio_mem   ████████████████ 30.01 MB/s
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 1966 ops/s | 457.7us | 948.6us | 1.1ms | 0 |
| minio | 1822 ops/s | 474.1us | 1.3ms | 1.7ms | 0 |
| rustfs | 1119 ops/s | 754.8us | 1.8ms | 2.3ms | 0 |
| localstack | 667 ops/s | 1.0ms | 3.0ms | 7.2ms | 0 |
| liteio | 555 ops/s | 1.7ms | 2.4ms | 2.5ms | 0 |
| liteio_mem | 516 ops/s | 1.8ms | 2.9ms | 3.3ms | 0 |

```
  seaweedfs    ████████████████████████████████████████ 1966 ops/s
  minio        █████████████████████████████████████ 1822 ops/s
  rustfs       ██████████████████████ 1119 ops/s
  localstack   █████████████ 667 ops/s
  liteio       ███████████ 555 ops/s
  liteio_mem   ██████████ 516 ops/s
```

### Write/100MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 120.17 MB/s | 771.6ms | 935.5ms | 935.5ms | 0 |
| seaweedfs | 113.67 MB/s | 830.1ms | 946.7ms | 946.7ms | 0 |
| minio | 102.34 MB/s | 996.2ms | 1.01s | 1.01s | 0 |
| localstack | 97.24 MB/s | 1.03s | 1.08s | 1.08s | 0 |
| liteio_mem | 91.67 MB/s | 1.07s | 1.11s | 1.11s | 0 |
| liteio | 89.13 MB/s | 1.12s | 1.14s | 1.14s | 0 |

```
  rustfs       ████████████████████████████████████████ 120.17 MB/s
  seaweedfs    █████████████████████████████████████ 113.67 MB/s
  minio        ██████████████████████████████████ 102.34 MB/s
  localstack   ████████████████████████████████ 97.24 MB/s
  liteio_mem   ██████████████████████████████ 91.67 MB/s
  liteio       █████████████████████████████ 89.13 MB/s
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 142.36 MB/s | 66.6ms | 88.1ms | 88.1ms | 0 |
| minio | 107.51 MB/s | 82.6ms | 123.3ms | 123.3ms | 0 |
| localstack | 105.81 MB/s | 92.8ms | 101.6ms | 101.6ms | 0 |
| seaweedfs | 91.04 MB/s | 111.8ms | 116.8ms | 116.8ms | 0 |
| liteio | 84.81 MB/s | 85.2ms | 125.3ms | 125.3ms | 0 |
| liteio_mem | 71.10 MB/s | 110.8ms | 175.3ms | 175.3ms | 0 |

```
  rustfs       ████████████████████████████████████████ 142.36 MB/s
  minio        ██████████████████████████████ 107.51 MB/s
  localstack   █████████████████████████████ 105.81 MB/s
  seaweedfs    █████████████████████████ 91.04 MB/s
  liteio       ███████████████████████ 84.81 MB/s
  liteio_mem   ███████████████████ 71.10 MB/s
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 0.72 MB/s | 844.4us | 3.3ms | 6.7ms | 0 |
| localstack | 0.65 MB/s | 1.0ms | 3.5ms | 6.6ms | 0 |
| minio | 0.51 MB/s | 1.8ms | 2.8ms | 3.5ms | 0 |
| rustfs | 0.48 MB/s | 1.8ms | 3.5ms | 6.2ms | 0 |
| liteio_mem | 0.26 MB/s | 3.7ms | 5.0ms | 5.8ms | 0 |
| liteio | 0.15 MB/s | 1.7ms | 5.5ms | 7.7ms | 0 |

```
  seaweedfs    ████████████████████████████████████████ 0.72 MB/s
  localstack   ███████████████████████████████████ 0.65 MB/s
  minio        ████████████████████████████ 0.51 MB/s
  rustfs       ██████████████████████████ 0.48 MB/s
  liteio_mem   ██████████████ 0.26 MB/s
  liteio       ████████ 0.15 MB/s
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 111.54 MB/s | 8.0ms | 13.5ms | 13.5ms | 0 |
| localstack | 89.81 MB/s | 10.3ms | 14.1ms | 14.1ms | 0 |
| seaweedfs | 79.50 MB/s | 12.4ms | 14.9ms | 14.9ms | 0 |
| minio | 72.39 MB/s | 10.7ms | 23.6ms | 23.6ms | 0 |
| liteio_mem | 64.72 MB/s | 15.4ms | 16.9ms | 16.9ms | 0 |
| liteio | 24.44 MB/s | 12.8ms | 40.5ms | 40.5ms | 0 |

```
  rustfs       ████████████████████████████████████████ 111.54 MB/s
  localstack   ████████████████████████████████ 89.81 MB/s
  seaweedfs    ████████████████████████████ 79.50 MB/s
  minio        █████████████████████████ 72.39 MB/s
  liteio_mem   ███████████████████████ 64.72 MB/s
  liteio       ████████ 24.44 MB/s
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| localstack | 39.97 MB/s | 1.3ms | 2.4ms | 2.6ms | 0 |
| rustfs | 37.71 MB/s | 1.5ms | 2.5ms | 3.3ms | 0 |
| seaweedfs | 26.87 MB/s | 2.1ms | 4.0ms | 4.0ms | 0 |
| minio | 13.53 MB/s | 2.5ms | 14.5ms | 23.9ms | 0 |
| liteio_mem | 12.74 MB/s | 4.5ms | 8.0ms | 9.6ms | 0 |
| liteio | 9.33 MB/s | 6.6ms | 8.9ms | 11.7ms | 0 |

```
  localstack   ████████████████████████████████████████ 39.97 MB/s
  rustfs       █████████████████████████████████████ 37.71 MB/s
  seaweedfs    ██████████████████████████ 26.87 MB/s
  minio        █████████████ 13.53 MB/s
  liteio_mem   ████████████ 12.74 MB/s
  liteio       █████████ 9.33 MB/s
```

## Recommendations

- **Best for write-heavy workloads:** rustfs
- **Best for read-heavy workloads:** minio

---

*Report generated by storage benchmark CLI*
