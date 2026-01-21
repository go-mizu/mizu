# Storage Benchmark Report

**Generated:** 2026-01-21T17:00:41+07:00

**Go Version:** go1.25.5

**Platform:** darwin/arm64

## Executive Summary

### Summary

**Overall Winner:** usagi_s3 (won 26/39 benchmarks, 67%)

| Rank | Driver | Wins | Win Rate |
|------|--------|------|----------|
| 1 | usagi_s3 | 26 | 67% |
| 2 | minio | 13 | 33% |

### Performance Leaders

| Operation | Leader | Performance | Margin |
|-----------|--------|-------------|--------|
| Small Read (1KB) | minio | 2.9 MB/s | +32% vs usagi_s3 |
| Small Write (1KB) | minio | 1.1 MB/s | +98% vs usagi_s3 |
| Large Read (100MB) | minio | 250.8 MB/s | +39% vs usagi_s3 |
| Large Write (100MB) | minio | 155.3 MB/s | +10% vs usagi_s3 |
| Delete | usagi_s3 | 4.1K ops/s | +53% vs minio |
| Stat | minio | 3.6K ops/s | 3.0x vs usagi_s3 |
| List (100 objects) | minio | 562 ops/s | +84% vs usagi_s3 |
| Copy | usagi_s3 | 1.2 MB/s | +22% vs minio |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (100MB+) | **minio** | 155 MB/s | Best for media, backups |
| Large File Downloads (100MB) | **minio** | 251 MB/s | Best for streaming, CDN |
| Small File Operations | **minio** | 2068 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **usagi_s3** | - | Best for multi-user apps |
| Memory Constrained | **minio** | 1001 MB RAM | Best for edge/embedded |

### Large File Performance (100MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| minio | 155.3 | 250.8 | 640.4ms | 400.5ms |
| usagi_s3 | 140.9 | 180.9 | 702.1ms | 558.7ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| minio | 1134 | 3002 | 854.2us | 312.3us |
| usagi_s3 | 574 | 2277 | 1.6ms | 401.3us |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| minio | 3594 | 562 | 2679 |
| usagi_s3 | 1199 | 305 | 4102 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| minio | 1.19 | 0.27 | 0.11 | 0.05 | 0.03 | 0.01 |
| usagi_s3 | 1.22 | 0.43 | 0.14 | 0.07 | 0.04 | 0.02 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| minio | 2.61 | 0.88 | 0.43 | 0.24 | 0.12 | 0.07 |
| usagi_s3 | 3.61 | 1.30 | 0.64 | 0.37 | 0.17 | 0.09 |

*\* indicates errors occurred*

### Scale Performance

Performance with varying numbers of objects (1KB each).

**Write N Files (total time)**

| Driver | 10 |
|--------|------|
| minio | 10.2ms |
| usagi_s3 | 8.8ms |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 10 |
|--------|------|
| minio | 809.7us |
| usagi_s3 | 497.9us |

*\* indicates errors occurred*

### Resource Usage Summary

| Driver | Memory | CPU |
|--------|--------|-----|
| minio | 1001.0 MB | 2.0% |
| usagi_s3 | 1133.6 MB | 4.3% |

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

- **minio** (39 benchmarks)
- **usagi_s3** (39 benchmarks)

## Detailed Results

### Copy/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 1.17 MB/s | 674.4us | 1.9ms | 2.7ms | 0 |
| minio | 0.96 MB/s | 905.7us | 1.7ms | 2.3ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 1.17 MB/s
minio        ████████████████████████ 0.96 MB/s
```

**Latency (P50)**
```
usagi_s3     ██████████████████████ 674.4us
minio        ██████████████████████████████ 905.7us
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 4102 ops/s | 215.2us | 380.6us | 755.1us | 0 |
| minio | 2679 ops/s | 352.6us | 464.6us | 685.1us | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 4102 ops/s
minio        ███████████████████ 2679 ops/s
```

**Latency (P50)**
```
usagi_s3     ██████████████████ 215.2us
minio        ██████████████████████████████ 352.6us
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.14 MB/s | 648.3us | 1.1ms | 1.5ms | 0 |
| minio | 0.09 MB/s | 967.5us | 1.4ms | 1.8ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.14 MB/s
minio        ████████████████████ 0.09 MB/s
```

**Latency (P50)**
```
usagi_s3     ████████████████████ 648.3us
minio        ██████████████████████████████ 967.5us
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 1666 ops/s | 569.6us | 863.7us | 1.2ms | 0 |
| minio | 1016 ops/s | 860.6us | 1.7ms | 2.5ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 1666 ops/s
minio        ██████████████████ 1016 ops/s
```

**Latency (P50)**
```
usagi_s3     ███████████████████ 569.6us
minio        ██████████████████████████████ 860.6us
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.13 MB/s | 678.1us | 1.2ms | 1.6ms | 0 |
| minio | 0.10 MB/s | 916.4us | 1.5ms | 2.0ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.13 MB/s
minio        ██████████████████████ 0.10 MB/s
```

**Latency (P50)**
```
usagi_s3     ██████████████████████ 678.1us
minio        ██████████████████████████████ 916.4us
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 562 ops/s | 1.7ms | 2.1ms | 2.3ms | 0 |
| usagi_s3 | 305 ops/s | 2.7ms | 6.7ms | 10.9ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 562 ops/s
usagi_s3     ████████████████ 305 ops/s
```

**Latency (P50)**
```
minio        ███████████████████ 1.7ms
usagi_s3     ██████████████████████████████ 2.7ms
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.47 MB/s | 25.7ms | 59.0ms | 92.6ms | 0 |
| minio | 0.26 MB/s | 41.3ms | 172.8ms | 231.5ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.47 MB/s
minio        ████████████████ 0.26 MB/s
```

**Latency (P50)**
```
usagi_s3     ██████████████████ 25.7ms
minio        ██████████████████████████████ 41.3ms
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.63 MB/s | 24.4ms | 38.9ms | 52.4ms | 0 |
| minio | 0.51 MB/s | 21.7ms | 100.6ms | 159.2ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.63 MB/s
minio        ████████████████████████ 0.51 MB/s
```

**Latency (P50)**
```
usagi_s3     ██████████████████████████████ 24.4ms
minio        ██████████████████████████ 21.7ms
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.37 MB/s | 33.4ms | 76.5ms | 251.5ms | 0 |
| minio | 0.16 MB/s | 59.7ms | 249.1ms | 835.5ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.37 MB/s
minio        █████████████ 0.16 MB/s
```

**Latency (P50)**
```
usagi_s3     ████████████████ 33.4ms
minio        ██████████████████████████████ 59.7ms
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 132.20 MB/s | 114.5ms | 127.5ms | 127.5ms | 0 |
| usagi_s3 | 125.55 MB/s | 118.3ms | 124.4ms | 124.4ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 132.20 MB/s
usagi_s3     ████████████████████████████ 125.55 MB/s
```

**Latency (P50)**
```
minio        █████████████████████████████ 114.5ms
usagi_s3     ██████████████████████████████ 118.3ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| usagi_s3 | 3.61 MB/s | 270.3us | 394.8us | 248.5us | 394.8us | 629.6us | 0 |
| minio | 2.61 MB/s | 374.2us | 487.0us | 356.0us | 487.2us | 686.1us | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 3.61 MB/s
minio        █████████████████████ 2.61 MB/s
```

**Latency (P50)**
```
usagi_s3     ████████████████████ 248.5us
minio        ██████████████████████████████ 356.0us
```

### ParallelRead/1KB/C10

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| usagi_s3 | 1.30 MB/s | 750.5us | 1.2ms | 712.5us | 1.2ms | 1.5ms | 0 |
| minio | 0.88 MB/s | 1.1ms | 1.8ms | 1.0ms | 1.8ms | 2.6ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 1.30 MB/s
minio        ████████████████████ 0.88 MB/s
```

**Latency (P50)**
```
usagi_s3     ████████████████████ 712.5us
minio        ██████████████████████████████ 1.0ms
```

### ParallelRead/1KB/C100

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| usagi_s3 | 0.17 MB/s | 5.6ms | 9.9ms | 5.3ms | 9.9ms | 13.9ms | 0 |
| minio | 0.12 MB/s | 7.8ms | 15.6ms | 7.0ms | 15.6ms | 21.9ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.17 MB/s
minio        █████████████████████ 0.12 MB/s
```

**Latency (P50)**
```
usagi_s3     ██████████████████████ 5.3ms
minio        ██████████████████████████████ 7.0ms
```

### ParallelRead/1KB/C200

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| usagi_s3 | 0.09 MB/s | 10.5ms | 15.9ms | 10.7ms | 15.9ms | 18.4ms | 0 |
| minio | 0.07 MB/s | 14.9ms | 30.9ms | 13.3ms | 31.0ms | 43.2ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.09 MB/s
minio        █████████████████████ 0.07 MB/s
```

**Latency (P50)**
```
usagi_s3     ████████████████████████ 10.7ms
minio        ██████████████████████████████ 13.3ms
```

### ParallelRead/1KB/C25

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| usagi_s3 | 0.64 MB/s | 1.5ms | 2.6ms | 1.4ms | 2.6ms | 3.2ms | 0 |
| minio | 0.43 MB/s | 2.3ms | 3.9ms | 2.1ms | 3.9ms | 5.4ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.64 MB/s
minio        ████████████████████ 0.43 MB/s
```

**Latency (P50)**
```
usagi_s3     ████████████████████ 1.4ms
minio        ██████████████████████████████ 2.1ms
```

### ParallelRead/1KB/C50

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| usagi_s3 | 0.37 MB/s | 2.6ms | 4.6ms | 2.5ms | 4.6ms | 6.1ms | 0 |
| minio | 0.24 MB/s | 4.0ms | 7.4ms | 3.7ms | 7.4ms | 10.7ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.37 MB/s
minio        ███████████████████ 0.24 MB/s
```

**Latency (P50)**
```
usagi_s3     ████████████████████ 2.5ms
minio        ██████████████████████████████ 3.7ms
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 1.22 MB/s | 746.6us | 1.0ms | 1.6ms | 0 |
| minio | 1.19 MB/s | 763.3us | 1.1ms | 2.5ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 1.22 MB/s
minio        █████████████████████████████ 1.19 MB/s
```

**Latency (P50)**
```
usagi_s3     █████████████████████████████ 746.6us
minio        ██████████████████████████████ 763.3us
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.43 MB/s | 2.2ms | 3.3ms | 4.2ms | 0 |
| minio | 0.27 MB/s | 3.3ms | 5.9ms | 7.6ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.43 MB/s
minio        ██████████████████ 0.27 MB/s
```

**Latency (P50)**
```
usagi_s3     ███████████████████ 2.2ms
minio        ██████████████████████████████ 3.3ms
```

### ParallelWrite/1KB/C100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.04 MB/s | 29.2ms | 46.2ms | 60.2ms | 0 |
| minio | 0.03 MB/s | 32.4ms | 54.9ms | 70.2ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.04 MB/s
minio        ███████████████████████ 0.03 MB/s
```

**Latency (P50)**
```
usagi_s3     ███████████████████████████ 29.2ms
minio        ██████████████████████████████ 32.4ms
```

### ParallelWrite/1KB/C200

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.02 MB/s | 56.7ms | 75.7ms | 95.0ms | 0 |
| minio | 0.01 MB/s | 62.5ms | 120.1ms | 182.1ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.02 MB/s
minio        ██████████████████████ 0.01 MB/s
```

**Latency (P50)**
```
usagi_s3     ███████████████████████████ 56.7ms
minio        ██████████████████████████████ 62.5ms
```

### ParallelWrite/1KB/C25

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.14 MB/s | 7.0ms | 11.4ms | 13.5ms | 0 |
| minio | 0.11 MB/s | 8.5ms | 14.1ms | 18.3ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.14 MB/s
minio        ██████████████████████ 0.11 MB/s
```

**Latency (P50)**
```
usagi_s3     ████████████████████████ 7.0ms
minio        ██████████████████████████████ 8.5ms
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.07 MB/s | 13.1ms | 20.6ms | 39.2ms | 0 |
| minio | 0.05 MB/s | 15.5ms | 37.7ms | 54.1ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.07 MB/s
minio        █████████████████████ 0.05 MB/s
```

**Latency (P50)**
```
usagi_s3     █████████████████████████ 13.1ms
minio        ██████████████████████████████ 15.5ms
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 167.23 MB/s | 1.4ms | 1.9ms | 2.7ms | 0 |
| minio | 143.48 MB/s | 1.7ms | 2.2ms | 3.6ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 167.23 MB/s
minio        █████████████████████████ 143.48 MB/s
```

**Latency (P50)**
```
usagi_s3     ██████████████████████████ 1.4ms
minio        ██████████████████████████████ 1.7ms
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 168.44 MB/s | 1.4ms | 1.8ms | 2.2ms | 0 |
| minio | 144.49 MB/s | 1.7ms | 2.1ms | 2.7ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 168.44 MB/s
minio        █████████████████████████ 144.49 MB/s
```

**Latency (P50)**
```
usagi_s3     █████████████████████████ 1.4ms
minio        ██████████████████████████████ 1.7ms
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 144.25 MB/s | 1.6ms | 2.6ms | 3.2ms | 0 |
| minio | 123.21 MB/s | 1.8ms | 3.0ms | 4.8ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 144.25 MB/s
minio        █████████████████████████ 123.21 MB/s
```

**Latency (P50)**
```
usagi_s3     █████████████████████████ 1.6ms
minio        ██████████████████████████████ 1.8ms
```

### Read/100MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 250.82 MB/s | 1.0ms | 979.2us | 400.5ms | 400.5ms | 400.5ms | 0 |
| usagi_s3 | 180.88 MB/s | 1.7ms | 1.8ms | 558.7ms | 558.7ms | 558.7ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 250.82 MB/s
usagi_s3     █████████████████████ 180.88 MB/s
```

**Latency (P50)**
```
minio        █████████████████████ 400.5ms
usagi_s3     ██████████████████████████████ 558.7ms
```

### Read/10MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 252.47 MB/s | 971.2us | 1.1ms | 39.6ms | 40.6ms | 40.7ms | 0 |
| usagi_s3 | 181.82 MB/s | 1.8ms | 2.3ms | 54.1ms | 59.6ms | 61.3ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 252.47 MB/s
usagi_s3     █████████████████████ 181.82 MB/s
```

**Latency (P50)**
```
minio        █████████████████████ 39.6ms
usagi_s3     ██████████████████████████████ 54.1ms
```

### Read/1KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 2.93 MB/s | 333.0us | 427.1us | 312.3us | 427.2us | 721.6us | 0 |
| usagi_s3 | 2.22 MB/s | 438.9us | 671.2us | 401.3us | 671.6us | 943.2us | 0 |

**Throughput**
```
minio        ██████████████████████████████ 2.93 MB/s
usagi_s3     ██████████████████████ 2.22 MB/s
```

**Latency (P50)**
```
minio        ███████████████████████ 312.3us
usagi_s3     ██████████████████████████████ 401.3us
```

### Read/1MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 205.97 MB/s | 971.5us | 1.5ms | 4.6ms | 6.2ms | 9.0ms | 0 |
| usagi_s3 | 185.46 MB/s | 605.7us | 1.1ms | 5.3ms | 6.3ms | 7.1ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 205.97 MB/s
usagi_s3     ███████████████████████████ 185.46 MB/s
```

**Latency (P50)**
```
minio        ██████████████████████████ 4.6ms
usagi_s3     ██████████████████████████████ 5.3ms
```

### Read/64KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 97.19 MB/s | 449.7us | 740.7us | 599.5us | 901.9us | 1.3ms | 0 |
| usagi_s3 | 73.44 MB/s | 550.4us | 932.7us | 791.8us | 1.3ms | 1.7ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 97.19 MB/s
usagi_s3     ██████████████████████ 73.44 MB/s
```

**Latency (P50)**
```
minio        ██████████████████████ 599.5us
usagi_s3     ██████████████████████████████ 791.8us
```

### Scale/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 340 ops/s | 2.9ms | 2.9ms | 2.9ms | 0 |
| minio | 253 ops/s | 3.9ms | 3.9ms | 3.9ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 340 ops/s
minio        ██████████████████████ 253 ops/s
```

**Latency (P50)**
```
usagi_s3     ██████████████████████ 2.9ms
minio        ██████████████████████████████ 3.9ms
```

### Scale/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 2009 ops/s | 497.9us | 497.9us | 497.9us | 0 |
| minio | 1235 ops/s | 809.7us | 809.7us | 809.7us | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 2009 ops/s
minio        ██████████████████ 1235 ops/s
```

**Latency (P50)**
```
usagi_s3     ██████████████████ 497.9us
minio        ██████████████████████████████ 809.7us
```

### Scale/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 1.10 MB/s | 8.8ms | 8.8ms | 8.8ms | 0 |
| minio | 0.96 MB/s | 10.2ms | 10.2ms | 10.2ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 1.10 MB/s
minio        ██████████████████████████ 0.96 MB/s
```

**Latency (P50)**
```
usagi_s3     ██████████████████████████ 8.8ms
minio        ██████████████████████████████ 10.2ms
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 3594 ops/s | 261.2us | 368.0us | 610.7us | 0 |
| usagi_s3 | 1199 ops/s | 426.5us | 2.1ms | 6.2ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 3594 ops/s
usagi_s3     ██████████ 1199 ops/s
```

**Latency (P50)**
```
minio        ██████████████████ 261.2us
usagi_s3     ██████████████████████████████ 426.5us
```

### Write/100MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 155.28 MB/s | 640.4ms | 640.4ms | 640.4ms | 0 |
| usagi_s3 | 140.92 MB/s | 702.1ms | 702.1ms | 702.1ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 155.28 MB/s
usagi_s3     ███████████████████████████ 140.92 MB/s
```

**Latency (P50)**
```
minio        ███████████████████████████ 640.4ms
usagi_s3     ██████████████████████████████ 702.1ms
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 156.51 MB/s | 64.7ms | 67.6ms | 68.4ms | 0 |
| usagi_s3 | 107.11 MB/s | 89.6ms | 114.8ms | 114.8ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 156.51 MB/s
usagi_s3     ████████████████████ 107.11 MB/s
```

**Latency (P50)**
```
minio        █████████████████████ 64.7ms
usagi_s3     ██████████████████████████████ 89.6ms
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 1.11 MB/s | 854.2us | 1.1ms | 1.3ms | 0 |
| usagi_s3 | 0.56 MB/s | 1.6ms | 3.0ms | 5.3ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 1.11 MB/s
usagi_s3     ███████████████ 0.56 MB/s
```

**Latency (P50)**
```
minio        ████████████████ 854.2us
usagi_s3     ██████████████████████████████ 1.6ms
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 122.58 MB/s | 7.9ms | 10.3ms | 11.5ms | 0 |
| usagi_s3 | 98.29 MB/s | 8.8ms | 19.4ms | 30.2ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 122.58 MB/s
usagi_s3     ████████████████████████ 98.29 MB/s
```

**Latency (P50)**
```
minio        ███████████████████████████ 7.9ms
usagi_s3     ██████████████████████████████ 8.8ms
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 45.75 MB/s | 1.3ms | 1.7ms | 3.7ms | 0 |
| usagi_s3 | 26.57 MB/s | 2.0ms | 4.9ms | 8.2ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 45.75 MB/s
usagi_s3     █████████████████ 26.57 MB/s
```

**Latency (P50)**
```
minio        ██████████████████ 1.3ms
usagi_s3     ██████████████████████████████ 2.0ms
```

## Resource Usage

| Driver | Memory | RSS | Cache | CPU | Volume | Block I/O |
|--------|--------|-----|-------|-----|--------|----------|
| minio | 1001MiB / 7.653GiB | 1001.0 MB | - | 2.0% | 2169.0 MB | 5.76MB / 2.34GB |
| usagi_s3 | 1.107GiB / 7.653GiB | 1133.6 MB | - | 4.3% | 1626.1 MB | 0B / 2.08GB |

> **Note:** RSS = actual application memory. Cache = OS page cache (reclaimable).

## Recommendations

- **Write-heavy workloads:** minio
- **Read-heavy workloads:** minio

---

*Generated by storage benchmark CLI*
