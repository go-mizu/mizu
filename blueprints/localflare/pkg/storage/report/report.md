# Storage Benchmark Report

**Generated:** 2026-01-21T17:42:23+07:00

**Go Version:** go1.25.5

**Platform:** darwin/arm64

## Executive Summary

### Summary

**Overall Winner:** usagi_s3 (won 40/48 benchmarks, 83%)

| Rank | Driver | Wins | Win Rate |
|------|--------|------|----------|
| 1 | usagi_s3 | 40 | 83% |
| 2 | minio | 8 | 17% |

### Performance Leaders

| Operation | Leader | Performance | Margin |
|-----------|--------|-------------|--------|
| Small Read (1KB) | usagi_s3 | 3.9 MB/s | +43% vs minio |
| Small Write (1KB) | usagi_s3 | 1.8 MB/s | +45% vs minio |
| Large Read (100MB) | minio | 196.9 MB/s | +20% vs usagi_s3 |
| Large Write (100MB) | minio | 130.5 MB/s | close |
| Delete | usagi_s3 | 3.8K ops/s | +51% vs minio |
| Stat | usagi_s3 | 3.8K ops/s | +17% vs minio |
| List (100 objects) | usagi_s3 | 869 ops/s | +65% vs minio |
| Copy | usagi_s3 | 1.0 MB/s | +20% vs minio |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (100MB+) | **minio** | 131 MB/s | Best for media, backups |
| Large File Downloads (100MB) | **minio** | 197 MB/s | Best for streaming, CDN |
| Small File Operations | **usagi_s3** | 2913 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **usagi_s3** | - | Best for multi-user apps |
| Memory Constrained | **minio** | 961 MB RAM | Best for edge/embedded |

### Large File Performance (100MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| minio | 130.5 | 196.9 | 727.6ms | 507.7ms |
| usagi_s3 | 125.0 | 164.5 | 804.0ms | 615.8ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| minio | 1269 | 2796 | 738.1us | 331.2us |
| usagi_s3 | 1838 | 3989 | 509.4us | 232.4us |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| minio | 3277 | 528 | 2480 |
| usagi_s3 | 3829 | 869 | 3756 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| minio | 1.06 | 0.30 | 0.10 | 0.05 | 0.02 | 0.01 |
| usagi_s3 | 1.31 | 0.29 | 0.10 | 0.06 | 0.03 | 0.02 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| minio | 2.32 | 0.69 | 0.34 | 0.21 | 0.10 | 0.05 |
| usagi_s3 | 3.09 | 1.00 | 0.52 | 0.32 | 0.18 | 0.08 |

*\* indicates errors occurred*

### Scale Performance

Performance with varying numbers of objects (256B each).

**Write N Files (total time)**

| Driver | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|
| minio | 10.1ms | 101.8ms | 1.07s | 10.57s |
| usagi_s3 | 6.6ms | 87.0ms | 729.1ms | 6.13s |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|
| minio | 1.0ms | 2.3ms | 14.7ms | 179.1ms |
| usagi_s3 | 1.1ms | 1.2ms | 8.0ms | 255.7ms |

*\* indicates errors occurred*

### Skipped Benchmarks

Some benchmarks were skipped due to driver limitations:

- **minio**: 2 skipped
  - Scale/100000 (requires longer timeout)
  - Scale/1000000 (requires longer timeout)
- **usagi_s3**: 2 skipped
  - Scale/100000 (requires longer timeout)
  - Scale/1000000 (requires longer timeout)

### Resource Usage Summary

| Driver | Memory | CPU |
|--------|--------|-----|
| minio | 961.1 MB | 0.0% |
| usagi_s3 | 1332.2 MB | 1.6% |

---

## Configuration

| Parameter | Value |
|-----------|-------|
| BenchTime | 1s |
| MinIterations | 3 |
| Warmup | 10 |
| Concurrency | 200 |
| Timeout | 30s |

## Drivers Tested

- **minio** (48 benchmarks)
- **usagi_s3** (48 benchmarks)

## Detailed Results

### Copy/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 1.03 MB/s | 743.1us | 1.8ms | 2.7ms | 0 |
| minio | 0.86 MB/s | 1.0ms | 1.8ms | 2.2ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 1.03 MB/s
minio        ████████████████████████ 0.86 MB/s
```

**Latency (P50)**
```
usagi_s3     █████████████████████ 743.1us
minio        ██████████████████████████████ 1.0ms
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 3756 ops/s | 239.4us | 404.3us | 717.2us | 0 |
| minio | 2480 ops/s | 380.8us | 527.1us | 732.5us | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 3756 ops/s
minio        ███████████████████ 2480 ops/s
```

**Latency (P50)**
```
usagi_s3     ██████████████████ 239.4us
minio        ██████████████████████████████ 380.8us
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.13 MB/s | 640.3us | 1.2ms | 1.7ms | 0 |
| minio | 0.09 MB/s | 1.0ms | 1.4ms | 1.9ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.13 MB/s
minio        ████████████████████ 0.09 MB/s
```

**Latency (P50)**
```
usagi_s3     ███████████████████ 640.3us
minio        ██████████████████████████████ 1.0ms
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 1584 ops/s | 564.4us | 1.0ms | 1.5ms | 0 |
| minio | 965 ops/s | 896.6us | 1.8ms | 2.4ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 1584 ops/s
minio        ██████████████████ 965 ops/s
```

**Latency (P50)**
```
usagi_s3     ██████████████████ 564.4us
minio        ██████████████████████████████ 896.6us
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.14 MB/s | 580.3us | 1.1ms | 1.5ms | 0 |
| minio | 0.08 MB/s | 997.0us | 1.9ms | 2.5ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.14 MB/s
minio        █████████████████ 0.08 MB/s
```

**Latency (P50)**
```
usagi_s3     █████████████████ 580.3us
minio        ██████████████████████████████ 997.0us
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 869 ops/s | 1.1ms | 1.5ms | 1.8ms | 0 |
| minio | 528 ops/s | 1.8ms | 2.3ms | 2.8ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 869 ops/s
minio        ██████████████████ 528 ops/s
```

**Latency (P50)**
```
usagi_s3     ██████████████████ 1.1ms
minio        ██████████████████████████████ 1.8ms
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.36 MB/s | 14.5ms | 106.3ms | 119.7ms | 0 |
| minio | 0.22 MB/s | 29.7ms | 230.3ms | 358.8ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.36 MB/s
minio        ██████████████████ 0.22 MB/s
```

**Latency (P50)**
```
usagi_s3     ██████████████ 14.5ms
minio        ██████████████████████████████ 29.7ms
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.61 MB/s | 22.9ms | 44.0ms | 78.4ms | 0 |
| minio | 0.44 MB/s | 24.4ms | 107.6ms | 223.7ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.61 MB/s
minio        █████████████████████ 0.44 MB/s
```

**Latency (P50)**
```
usagi_s3     ████████████████████████████ 22.9ms
minio        ██████████████████████████████ 24.4ms
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.25 MB/s | 74.4ms | 104.2ms | 119.6ms | 0 |
| minio | 0.17 MB/s | 66.7ms | 248.2ms | 302.3ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.25 MB/s
minio        ███████████████████ 0.17 MB/s
```

**Latency (P50)**
```
usagi_s3     ██████████████████████████████ 74.4ms
minio        ██████████████████████████ 66.7ms
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 118.30 MB/s | 123.5ms | 136.8ms | 136.8ms | 0 |
| usagi_s3 | 99.09 MB/s | 130.5ms | 168.3ms | 168.3ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 118.30 MB/s
usagi_s3     █████████████████████████ 99.09 MB/s
```

**Latency (P50)**
```
minio        ████████████████████████████ 123.5ms
usagi_s3     ██████████████████████████████ 130.5ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| usagi_s3 | 3.09 MB/s | 315.4us | 510.8us | 285.8us | 511.0us | 726.1us | 0 |
| minio | 2.32 MB/s | 420.2us | 574.5us | 391.5us | 574.6us | 856.4us | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 3.09 MB/s
minio        ██████████████████████ 2.32 MB/s
```

**Latency (P50)**
```
usagi_s3     █████████████████████ 285.8us
minio        ██████████████████████████████ 391.5us
```

### ParallelRead/1KB/C10

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| usagi_s3 | 1.00 MB/s | 976.4us | 1.5ms | 906.9us | 1.5ms | 2.1ms | 0 |
| minio | 0.69 MB/s | 1.4ms | 2.4ms | 1.3ms | 2.4ms | 4.5ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 1.00 MB/s
minio        ████████████████████ 0.69 MB/s
```

**Latency (P50)**
```
usagi_s3     █████████████████████ 906.9us
minio        ██████████████████████████████ 1.3ms
```

### ParallelRead/1KB/C100

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| usagi_s3 | 0.18 MB/s | 5.5ms | 9.7ms | 5.3ms | 9.7ms | 13.1ms | 0 |
| minio | 0.10 MB/s | 9.9ms | 20.2ms | 8.7ms | 20.2ms | 32.1ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.18 MB/s
minio        ████████████████ 0.10 MB/s
```

**Latency (P50)**
```
usagi_s3     ██████████████████ 5.3ms
minio        ██████████████████████████████ 8.7ms
```

### ParallelRead/1KB/C200

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| usagi_s3 | 0.08 MB/s | 12.3ms | 17.5ms | 12.2ms | 17.5ms | 25.2ms | 0 |
| minio | 0.05 MB/s | 21.2ms | 44.9ms | 19.0ms | 44.9ms | 63.8ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.08 MB/s
minio        █████████████████ 0.05 MB/s
```

**Latency (P50)**
```
usagi_s3     ███████████████████ 12.2ms
minio        ██████████████████████████████ 19.0ms
```

### ParallelRead/1KB/C25

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| usagi_s3 | 0.52 MB/s | 1.9ms | 3.2ms | 1.7ms | 3.2ms | 4.5ms | 0 |
| minio | 0.34 MB/s | 2.9ms | 5.3ms | 2.7ms | 5.3ms | 7.6ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.52 MB/s
minio        ███████████████████ 0.34 MB/s
```

**Latency (P50)**
```
usagi_s3     ███████████████████ 1.7ms
minio        ██████████████████████████████ 2.7ms
```

### ParallelRead/1KB/C50

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| usagi_s3 | 0.32 MB/s | 3.1ms | 5.3ms | 2.9ms | 5.3ms | 7.0ms | 0 |
| minio | 0.21 MB/s | 4.7ms | 8.3ms | 4.4ms | 8.3ms | 10.8ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.32 MB/s
minio        ███████████████████ 0.21 MB/s
```

**Latency (P50)**
```
usagi_s3     ███████████████████ 2.9ms
minio        ██████████████████████████████ 4.4ms
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 1.31 MB/s | 685.8us | 1.1ms | 1.6ms | 0 |
| minio | 1.06 MB/s | 849.0us | 1.2ms | 2.1ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 1.31 MB/s
minio        ████████████████████████ 1.06 MB/s
```

**Latency (P50)**
```
usagi_s3     ████████████████████████ 685.8us
minio        ██████████████████████████████ 849.0us
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 0.30 MB/s | 3.2ms | 4.6ms | 6.1ms | 0 |
| usagi_s3 | 0.29 MB/s | 3.1ms | 5.6ms | 7.1ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 0.30 MB/s
usagi_s3     █████████████████████████████ 0.29 MB/s
```

**Latency (P50)**
```
minio        ██████████████████████████████ 3.2ms
usagi_s3     █████████████████████████████ 3.1ms
```

### ParallelWrite/1KB/C100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.03 MB/s | 35.4ms | 44.6ms | 49.4ms | 0 |
| minio | 0.02 MB/s | 36.5ms | 72.1ms | 92.8ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.03 MB/s
minio        ██████████████████████ 0.02 MB/s
```

**Latency (P50)**
```
usagi_s3     █████████████████████████████ 35.4ms
minio        ██████████████████████████████ 36.5ms
```

### ParallelWrite/1KB/C200

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.02 MB/s | 72.0ms | 112.5ms | 122.2ms | 0 |
| minio | 0.01 MB/s | 77.1ms | 159.7ms | 264.1ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.02 MB/s
minio        ██████████████████████ 0.01 MB/s
```

**Latency (P50)**
```
usagi_s3     ████████████████████████████ 72.0ms
minio        ██████████████████████████████ 77.1ms
```

### ParallelWrite/1KB/C25

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.10 MB/s | 10.2ms | 15.3ms | 24.0ms | 0 |
| minio | 0.10 MB/s | 8.8ms | 17.9ms | 22.4ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.10 MB/s
minio        █████████████████████████████ 0.10 MB/s
```

**Latency (P50)**
```
usagi_s3     ██████████████████████████████ 10.2ms
minio        █████████████████████████ 8.8ms
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.06 MB/s | 14.6ms | 38.3ms | 51.3ms | 0 |
| minio | 0.05 MB/s | 18.9ms | 32.8ms | 48.8ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.06 MB/s
minio        █████████████████████████ 0.05 MB/s
```

**Latency (P50)**
```
usagi_s3     ███████████████████████ 14.6ms
minio        ██████████████████████████████ 18.9ms
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 146.78 MB/s | 1.6ms | 2.1ms | 2.4ms | 0 |
| minio | 118.19 MB/s | 2.0ms | 2.7ms | 3.6ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 146.78 MB/s
minio        ████████████████████████ 118.19 MB/s
```

**Latency (P50)**
```
usagi_s3     ████████████████████████ 1.6ms
minio        ██████████████████████████████ 2.0ms
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 144.33 MB/s | 1.7ms | 2.2ms | 2.9ms | 0 |
| minio | 118.62 MB/s | 2.0ms | 2.6ms | 3.2ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 144.33 MB/s
minio        ████████████████████████ 118.62 MB/s
```

**Latency (P50)**
```
usagi_s3     ████████████████████████ 1.7ms
minio        ██████████████████████████████ 2.0ms
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 131.15 MB/s | 1.8ms | 2.6ms | 3.2ms | 0 |
| minio | 91.89 MB/s | 2.4ms | 4.4ms | 7.2ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 131.15 MB/s
minio        █████████████████████ 91.89 MB/s
```

**Latency (P50)**
```
usagi_s3     ██████████████████████ 1.8ms
minio        ██████████████████████████████ 2.4ms
```

### Read/100MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 196.92 MB/s | 1.2ms | 1.2ms | 507.7ms | 507.7ms | 507.7ms | 0 |
| usagi_s3 | 164.55 MB/s | 2.0ms | 1.9ms | 615.8ms | 615.8ms | 615.8ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 196.92 MB/s
usagi_s3     █████████████████████████ 164.55 MB/s
```

**Latency (P50)**
```
minio        ████████████████████████ 507.7ms
usagi_s3     ██████████████████████████████ 615.8ms
```

### Read/10MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 199.19 MB/s | 1.3ms | 1.7ms | 49.6ms | 53.0ms | 54.6ms | 0 |
| usagi_s3 | 163.63 MB/s | 2.5ms | 3.3ms | 60.0ms | 67.1ms | 67.5ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 199.19 MB/s
usagi_s3     ████████████████████████ 163.63 MB/s
```

**Latency (P50)**
```
minio        ████████████████████████ 49.6ms
usagi_s3     ██████████████████████████████ 60.0ms
```

### Read/1KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| usagi_s3 | 3.90 MB/s | 250.6us | 362.9us | 232.4us | 363.0us | 552.6us | 0 |
| minio | 2.73 MB/s | 357.6us | 497.2us | 331.2us | 497.3us | 753.5us | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 3.90 MB/s
minio        █████████████████████ 2.73 MB/s
```

**Latency (P50)**
```
usagi_s3     █████████████████████ 232.4us
minio        ██████████████████████████████ 331.2us
```

### Read/1MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 171.97 MB/s | 1.2ms | 1.9ms | 5.5ms | 7.5ms | 9.8ms | 0 |
| usagi_s3 | 162.94 MB/s | 566.5us | 948.9us | 6.0ms | 7.3ms | 8.0ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 171.97 MB/s
usagi_s3     ████████████████████████████ 162.94 MB/s
```

**Latency (P50)**
```
minio        ███████████████████████████ 5.5ms
usagi_s3     ██████████████████████████████ 6.0ms
```

### Read/64KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| usagi_s3 | 95.23 MB/s | 334.9us | 599.1us | 615.4us | 948.8us | 1.2ms | 0 |
| minio | 90.61 MB/s | 482.6us | 787.5us | 639.2us | 986.1us | 1.5ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 95.23 MB/s
minio        ████████████████████████████ 90.61 MB/s
```

**Latency (P50)**
```
usagi_s3     ████████████████████████████ 615.4us
minio        ██████████████████████████████ 639.2us
```

### Scale/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 388 ops/s | 2.6ms | 2.6ms | 2.6ms | 0 |
| minio | 216 ops/s | 4.6ms | 4.6ms | 4.6ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 388 ops/s
minio        ████████████████ 216 ops/s
```

**Latency (P50)**
```
usagi_s3     ████████████████ 2.6ms
minio        ██████████████████████████████ 4.6ms
```

### Scale/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 41 ops/s | 24.5ms | 24.5ms | 24.5ms | 0 |
| minio | 24 ops/s | 41.6ms | 41.6ms | 41.6ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 41 ops/s
minio        █████████████████ 24 ops/s
```

**Latency (P50)**
```
usagi_s3     █████████████████ 24.5ms
minio        ██████████████████████████████ 41.6ms
```

### Scale/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 4 ops/s | 270.8ms | 270.8ms | 270.8ms | 0 |
| minio | 2 ops/s | 416.8ms | 416.8ms | 416.8ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 4 ops/s
minio        ███████████████████ 2 ops/s
```

**Latency (P50)**
```
usagi_s3     ███████████████████ 270.8ms
minio        ██████████████████████████████ 416.8ms
```

### Scale/Delete/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0 ops/s | 2.70s | 2.70s | 2.70s | 0 |
| minio | 0 ops/s | 4.35s | 4.35s | 4.35s | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0 ops/s
minio        ██████████████████ 0 ops/s
```

**Latency (P50)**
```
usagi_s3     ██████████████████ 2.70s
minio        ██████████████████████████████ 4.35s
```

### Scale/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 989 ops/s | 1.0ms | 1.0ms | 1.0ms | 0 |
| usagi_s3 | 946 ops/s | 1.1ms | 1.1ms | 1.1ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 989 ops/s
usagi_s3     ████████████████████████████ 946 ops/s
```

**Latency (P50)**
```
minio        ████████████████████████████ 1.0ms
usagi_s3     ██████████████████████████████ 1.1ms
```

### Scale/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 836 ops/s | 1.2ms | 1.2ms | 1.2ms | 0 |
| minio | 436 ops/s | 2.3ms | 2.3ms | 2.3ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 836 ops/s
minio        ███████████████ 436 ops/s
```

**Latency (P50)**
```
usagi_s3     ███████████████ 1.2ms
minio        ██████████████████████████████ 2.3ms
```

### Scale/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 125 ops/s | 8.0ms | 8.0ms | 8.0ms | 0 |
| minio | 68 ops/s | 14.7ms | 14.7ms | 14.7ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 125 ops/s
minio        ████████████████ 68 ops/s
```

**Latency (P50)**
```
usagi_s3     ████████████████ 8.0ms
minio        ██████████████████████████████ 14.7ms
```

### Scale/List/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 6 ops/s | 179.1ms | 179.1ms | 179.1ms | 0 |
| usagi_s3 | 4 ops/s | 255.7ms | 255.7ms | 255.7ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 6 ops/s
usagi_s3     █████████████████████ 4 ops/s
```

**Latency (P50)**
```
minio        █████████████████████ 179.1ms
usagi_s3     ██████████████████████████████ 255.7ms
```

### Scale/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.37 MB/s | 6.6ms | 6.6ms | 6.6ms | 0 |
| minio | 0.24 MB/s | 10.1ms | 10.1ms | 10.1ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.37 MB/s
minio        ███████████████████ 0.24 MB/s
```

**Latency (P50)**
```
usagi_s3     ███████████████████ 6.6ms
minio        ██████████████████████████████ 10.1ms
```

### Scale/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.28 MB/s | 87.0ms | 87.0ms | 87.0ms | 0 |
| minio | 0.24 MB/s | 101.8ms | 101.8ms | 101.8ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.28 MB/s
minio        █████████████████████████ 0.24 MB/s
```

**Latency (P50)**
```
usagi_s3     █████████████████████████ 87.0ms
minio        ██████████████████████████████ 101.8ms
```

### Scale/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.33 MB/s | 729.1ms | 729.1ms | 729.1ms | 0 |
| minio | 0.23 MB/s | 1.07s | 1.07s | 1.07s | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.33 MB/s
minio        ████████████████████ 0.23 MB/s
```

**Latency (P50)**
```
usagi_s3     ████████████████████ 729.1ms
minio        ██████████████████████████████ 1.07s
```

### Scale/Write/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.40 MB/s | 6.13s | 6.13s | 6.13s | 0 |
| minio | 0.23 MB/s | 10.57s | 10.57s | 10.57s | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.40 MB/s
minio        █████████████████ 0.23 MB/s
```

**Latency (P50)**
```
usagi_s3     █████████████████ 6.13s
minio        ██████████████████████████████ 10.57s
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 3829 ops/s | 242.9us | 381.9us | 537.8us | 0 |
| minio | 3277 ops/s | 289.3us | 411.2us | 539.5us | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 3829 ops/s
minio        █████████████████████████ 3277 ops/s
```

**Latency (P50)**
```
usagi_s3     █████████████████████████ 242.9us
minio        ██████████████████████████████ 289.3us
```

### Write/100MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 130.54 MB/s | 727.6ms | 727.6ms | 727.6ms | 0 |
| usagi_s3 | 125.03 MB/s | 804.0ms | 804.0ms | 804.0ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 130.54 MB/s
usagi_s3     ████████████████████████████ 125.03 MB/s
```

**Latency (P50)**
```
minio        ███████████████████████████ 727.6ms
usagi_s3     ██████████████████████████████ 804.0ms
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 130.54 MB/s | 76.3ms | 83.2ms | 83.2ms | 0 |
| minio | 126.03 MB/s | 79.5ms | 85.5ms | 85.5ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 130.54 MB/s
minio        ████████████████████████████ 126.03 MB/s
```

**Latency (P50)**
```
usagi_s3     ████████████████████████████ 76.3ms
minio        ██████████████████████████████ 79.5ms
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 1.79 MB/s | 509.4us | 768.2us | 1.2ms | 0 |
| minio | 1.24 MB/s | 738.1us | 1.1ms | 1.5ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 1.79 MB/s
minio        ████████████████████ 1.24 MB/s
```

**Latency (P50)**
```
usagi_s3     ████████████████████ 509.4us
minio        ██████████████████████████████ 738.1us
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 104.99 MB/s | 8.7ms | 13.4ms | 20.8ms | 0 |
| minio | 103.24 MB/s | 9.2ms | 12.6ms | 16.2ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 104.99 MB/s
minio        █████████████████████████████ 103.24 MB/s
```

**Latency (P50)**
```
usagi_s3     ████████████████████████████ 8.7ms
minio        ██████████████████████████████ 9.2ms
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 59.61 MB/s | 944.3us | 1.5ms | 2.2ms | 0 |
| minio | 43.07 MB/s | 1.3ms | 2.2ms | 3.8ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 59.61 MB/s
minio        █████████████████████ 43.07 MB/s
```

**Latency (P50)**
```
usagi_s3     █████████████████████ 944.3us
minio        ██████████████████████████████ 1.3ms
```

## Resource Usage

| Driver | Memory | RSS | Cache | CPU | Volume | Block I/O |
|--------|--------|-----|-------|-----|--------|----------|
| minio | 961.1MiB / 7.653GiB | 961.1 MB | - | 0.0% | 1986.0 MB | 46.1MB / 2.22GB |
| usagi_s3 | 1.301GiB / 7.653GiB | 1332.2 MB | - | 1.6% | 5024.8 MB | 1.25MB / 2.15GB |

> **Note:** RSS = actual application memory. Cache = OS page cache (reclaimable).

## Recommendations

- **Write-heavy workloads:** minio
- **Read-heavy workloads:** minio

---

*Generated by storage benchmark CLI*
