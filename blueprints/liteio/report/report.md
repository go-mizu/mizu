# Storage Benchmark Report

**Generated:** 2026-02-18T23:34:32+07:00

**Go Version:** go1.26.0

**Platform:** darwin/arm64

## Executive Summary

### Summary

**Overall Winner:** liteio (won 38/40 benchmarks, 95%)

| Rank | Driver | Wins | Win Rate |
|------|--------|------|----------|
| 1 | liteio | 38 | 95% |
| 2 | minio | 2 | 5% |

### Performance Leaders

| Operation | Leader | Performance | Margin |
|-----------|--------|-------------|--------|
| Small Read (1KB) | liteio | 1.0 MB/s | +59% vs minio |
| Small Write (1KB) | liteio | 0.4 MB/s | +37% vs minio |
| Large Read (10MB) | minio | 109.4 MB/s | close |
| Large Write (10MB) | minio | 55.0 MB/s | close |
| Delete | liteio | 1.1K ops/s | 2.0x vs minio |
| Stat | liteio | 1.2K ops/s | +84% vs minio |
| List (100 objects) | liteio | 367 ops/s | 2.1x vs minio |
| Copy | liteio | 0.3 MB/s | +68% vs minio |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (10MB+) | **minio** | 55 MB/s | Best for media, backups |
| Large File Downloads (10MB) | **minio** | 109 MB/s | Best for streaming, CDN |
| Small File Operations | **liteio** | 714 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **liteio** | - | Best for multi-user apps |
| Memory Constrained | **liteio** | 453 MB RAM | Best for edge/embedded |

### Large File Performance (10MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| liteio | 54.9 | 107.1 | 182.3ms | 96.1ms |
| minio | 55.0 | 109.4 | 170.9ms | 94.4ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| liteio | 391 | 1038 | 2.3ms | 791.9us |
| minio | 286 | 653 | 3.2ms | 1.3ms |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| liteio | 1157 | 367 | 1102 |
| minio | 628 | 176 | 549 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| liteio | 0.36 | 0.12 | 0.03 |
| minio | 0.32 | 0.08 | 0.01 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| liteio | 0.87 | 0.62 | 0.22 |
| minio | 0.65 | 0.35 | 0.07 |

*\* indicates errors occurred*

### Scale Performance

Performance with varying numbers of objects (256B each).

**Write N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 |
|--------|------|------|------|------|
| liteio | 2.1ms | 21.2ms | 286.3ms | 2.62s |
| minio | 2.4ms | 35.6ms | 367.9ms | 4.21s |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 |
|--------|------|------|------|------|
| liteio | 861.1us | 918.2us | 4.9ms | 34.2ms |
| minio | 3.1ms | 2.4ms | 7.6ms | 50.6ms |

*\* indicates errors occurred*

### Resource Usage Summary

| Driver | Memory | CPU |
|--------|--------|-----|
| liteio | 452.8 MB | 2.7% |
| minio | 491.7 MB | 0.0% |

---

## Configuration

| Parameter | Value |
|-----------|-------|
| BenchTime | 500ms |
| MinIterations | 3 |
| Warmup | 5 |
| Concurrency | 200 |
| Timeout | 1m0s |

## Drivers Tested

- **liteio** (40 benchmarks)
- **minio** (40 benchmarks)

## Detailed Results

### Copy/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.32 MB/s | 2.7ms | 6.1ms | 7.5ms | 0 |
| minio | 0.19 MB/s | 4.5ms | 8.8ms | 15.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.32 MB/s
minio        █████████████████ 0.19 MB/s
```

**Latency (P50)**
```
liteio       █████████████████ 2.7ms
minio        ██████████████████████████████ 4.5ms
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1102 ops/s | 772.3us | 1.8ms | 3.6ms | 0 |
| minio | 549 ops/s | 1.6ms | 2.9ms | 4.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1102 ops/s
minio        ██████████████ 549 ops/s
```

**Latency (P50)**
```
liteio       ██████████████ 772.3us
minio        ██████████████████████████████ 1.6ms
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.04 MB/s | 2.2ms | 3.7ms | 4.7ms | 0 |
| minio | 0.02 MB/s | 3.8ms | 9.7ms | 17.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.04 MB/s
minio        ████████████ 0.02 MB/s
```

**Latency (P50)**
```
liteio       ████████████████ 2.2ms
minio        ██████████████████████████████ 3.8ms
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 451 ops/s | 1.9ms | 4.5ms | 6.8ms | 0 |
| minio | 108 ops/s | 6.6ms | 23.3ms | 38.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 451 ops/s
minio        ███████ 108 ops/s
```

**Latency (P50)**
```
liteio       ████████ 1.9ms
minio        ██████████████████████████████ 6.6ms
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.04 MB/s | 2.3ms | 4.4ms | 7.6ms | 0 |
| minio | 0.01 MB/s | 5.6ms | 16.5ms | 38.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.04 MB/s
minio        █████████ 0.01 MB/s
```

**Latency (P50)**
```
liteio       ████████████ 2.3ms
minio        ██████████████████████████████ 5.6ms
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 367 ops/s | 2.5ms | 3.8ms | 5.5ms | 0 |
| minio | 176 ops/s | 5.3ms | 8.1ms | 11.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 367 ops/s
minio        ██████████████ 176 ops/s
```

**Latency (P50)**
```
liteio       ██████████████ 2.5ms
minio        ██████████████████████████████ 5.3ms
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.24 MB/s | 39.9ms | 98.9ms | 911.3ms | 0 |
| minio | 0.13 MB/s | 92.7ms | 239.2ms | 298.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.24 MB/s
minio        ███████████████ 0.13 MB/s
```

**Latency (P50)**
```
liteio       ████████████ 39.9ms
minio        ██████████████████████████████ 92.7ms
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.36 MB/s | 34.7ms | 105.1ms | 163.3ms | 0 |
| minio | 0.16 MB/s | 73.0ms | 271.1ms | 390.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.36 MB/s
minio        █████████████ 0.16 MB/s
```

**Latency (P50)**
```
liteio       ██████████████ 34.7ms
minio        ██████████████████████████████ 73.0ms
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.24 MB/s | 52.3ms | 97.9ms | 309.4ms | 0 |
| minio | 0.07 MB/s | 203.5ms | 569.5ms | 647.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.24 MB/s
minio        ████████ 0.07 MB/s
```

**Latency (P50)**
```
liteio       ███████ 52.3ms
minio        ██████████████████████████████ 203.5ms
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 45.54 MB/s | 338.6ms | 338.6ms | 338.6ms | 0 |
| minio | 45.46 MB/s | 320.6ms | 320.6ms | 320.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 45.54 MB/s
minio        █████████████████████████████ 45.46 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 338.6ms
minio        ████████████████████████████ 320.6ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.87 MB/s | 902.1us | 2.1ms | 4.7ms | 0 |
| minio | 0.65 MB/s | 1.4ms | 2.0ms | 2.8ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.87 MB/s
minio        ██████████████████████ 0.65 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████ 902.1us
minio        ██████████████████████████████ 1.4ms
```

### ParallelRead/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.62 MB/s | 1.4ms | 2.8ms | 4.4ms | 0 |
| minio | 0.35 MB/s | 2.3ms | 5.4ms | 10.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.62 MB/s
minio        ████████████████ 0.35 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████ 1.4ms
minio        ██████████████████████████████ 2.3ms
```

### ParallelRead/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.22 MB/s | 3.6ms | 9.1ms | 16.7ms | 0 |
| minio | 0.07 MB/s | 11.2ms | 35.9ms | 49.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.22 MB/s
minio        █████████ 0.07 MB/s
```

**Latency (P50)**
```
liteio       █████████ 3.6ms
minio        ██████████████████████████████ 11.2ms
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.36 MB/s | 2.5ms | 4.2ms | 5.5ms | 0 |
| minio | 0.32 MB/s | 3.2ms | 4.5ms | 4.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.36 MB/s
minio        ██████████████████████████ 0.32 MB/s
```

**Latency (P50)**
```
liteio       ███████████████████████ 2.5ms
minio        ██████████████████████████████ 3.2ms
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.12 MB/s | 6.2ms | 18.8ms | 24.6ms | 0 |
| minio | 0.08 MB/s | 10.0ms | 35.3ms | 44.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.12 MB/s
minio        ██████████████████ 0.08 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████ 6.2ms
minio        ██████████████████████████████ 10.0ms
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.03 MB/s | 17.2ms | 107.3ms | 289.6ms | 0 |
| minio | 0.01 MB/s | 36.9ms | 235.9ms | 356.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.03 MB/s
minio        █████████████ 0.01 MB/s
```

**Latency (P50)**
```
liteio       █████████████ 17.2ms
minio        ██████████████████████████████ 36.9ms
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 61.94 MB/s | 3.9ms | 5.5ms | 6.5ms | 0 |
| minio | 33.46 MB/s | 6.7ms | 11.6ms | 14.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 61.94 MB/s
minio        ████████████████ 33.46 MB/s
```

**Latency (P50)**
```
liteio       █████████████████ 3.9ms
minio        ██████████████████████████████ 6.7ms
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 57.34 MB/s | 4.2ms | 5.7ms | 6.9ms | 0 |
| minio | 39.67 MB/s | 5.7ms | 8.5ms | 13.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 57.34 MB/s
minio        ████████████████████ 39.67 MB/s
```

**Latency (P50)**
```
liteio       █████████████████████ 4.2ms
minio        ██████████████████████████████ 5.7ms
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 70.16 MB/s | 3.2ms | 5.2ms | 8.7ms | 0 |
| minio | 37.58 MB/s | 6.1ms | 9.9ms | 14.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 70.16 MB/s
minio        ████████████████ 37.58 MB/s
```

**Latency (P50)**
```
liteio       ███████████████ 3.2ms
minio        ██████████████████████████████ 6.1ms
```

### Read/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 109.43 MB/s | 94.4ms | 99.3ms | 99.3ms | 0 |
| liteio | 107.06 MB/s | 96.1ms | 97.4ms | 97.4ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 109.43 MB/s
liteio       █████████████████████████████ 107.06 MB/s
```

**Latency (P50)**
```
minio        █████████████████████████████ 94.4ms
liteio       ██████████████████████████████ 96.1ms
```

### Read/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.01 MB/s | 791.9us | 2.1ms | 3.7ms | 0 |
| minio | 0.64 MB/s | 1.3ms | 2.7ms | 4.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.01 MB/s
minio        ██████████████████ 0.64 MB/s
```

**Latency (P50)**
```
liteio       █████████████████ 791.9us
minio        ██████████████████████████████ 1.3ms
```

### Read/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 89.22 MB/s | 11.4ms | 13.8ms | 14.3ms | 0 |
| minio | 58.24 MB/s | 16.0ms | 25.6ms | 27.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 89.22 MB/s
minio        ███████████████████ 58.24 MB/s
```

**Latency (P50)**
```
liteio       █████████████████████ 11.4ms
minio        ██████████████████████████████ 16.0ms
```

### Read/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 33.90 MB/s | 1.6ms | 3.5ms | 4.4ms | 0 |
| minio | 26.11 MB/s | 2.2ms | 3.5ms | 5.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 33.90 MB/s
minio        ███████████████████████ 26.11 MB/s
```

**Latency (P50)**
```
liteio       █████████████████████ 1.6ms
minio        ██████████████████████████████ 2.2ms
```

### Scale/Delete/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1224 ops/s | 816.7us | 816.7us | 816.7us | 0 |
| minio | 183 ops/s | 5.5ms | 5.5ms | 5.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1224 ops/s
minio        ████ 183 ops/s
```

**Latency (P50)**
```
liteio       ████ 816.7us
minio        ██████████████████████████████ 5.5ms
```

### Scale/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 142 ops/s | 7.1ms | 7.1ms | 7.1ms | 0 |
| minio | 68 ops/s | 14.7ms | 14.7ms | 14.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 142 ops/s
minio        ██████████████ 68 ops/s
```

**Latency (P50)**
```
liteio       ██████████████ 7.1ms
minio        ██████████████████████████████ 14.7ms
```

### Scale/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 7 ops/s | 135.3ms | 135.3ms | 135.3ms | 0 |
| minio | 7 ops/s | 152.2ms | 152.2ms | 152.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 7 ops/s
minio        ██████████████████████████ 7 ops/s
```

**Latency (P50)**
```
liteio       ██████████████████████████ 135.3ms
minio        ██████████████████████████████ 152.2ms
```

### Scale/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1 ops/s | 1.13s | 1.13s | 1.13s | 0 |
| minio | 1 ops/s | 1.93s | 1.93s | 1.93s | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1 ops/s
minio        █████████████████ 1 ops/s
```

**Latency (P50)**
```
liteio       █████████████████ 1.13s
minio        ██████████████████████████████ 1.93s
```

### Scale/List/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1161 ops/s | 861.1us | 861.1us | 861.1us | 0 |
| minio | 321 ops/s | 3.1ms | 3.1ms | 3.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1161 ops/s
minio        ████████ 321 ops/s
```

**Latency (P50)**
```
liteio       ████████ 861.1us
minio        ██████████████████████████████ 3.1ms
```

### Scale/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1089 ops/s | 918.2us | 918.2us | 918.2us | 0 |
| minio | 411 ops/s | 2.4ms | 2.4ms | 2.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1089 ops/s
minio        ███████████ 411 ops/s
```

**Latency (P50)**
```
liteio       ███████████ 918.2us
minio        ██████████████████████████████ 2.4ms
```

### Scale/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 205 ops/s | 4.9ms | 4.9ms | 4.9ms | 0 |
| minio | 131 ops/s | 7.6ms | 7.6ms | 7.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 205 ops/s
minio        ███████████████████ 131 ops/s
```

**Latency (P50)**
```
liteio       ███████████████████ 4.9ms
minio        ██████████████████████████████ 7.6ms
```

### Scale/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 29 ops/s | 34.2ms | 34.2ms | 34.2ms | 0 |
| minio | 20 ops/s | 50.6ms | 50.6ms | 50.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 29 ops/s
minio        ████████████████████ 20 ops/s
```

**Latency (P50)**
```
liteio       ████████████████████ 34.2ms
minio        ██████████████████████████████ 50.6ms
```

### Scale/Write/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.12 MB/s | 2.1ms | 2.1ms | 2.1ms | 0 |
| minio | 0.10 MB/s | 2.4ms | 2.4ms | 2.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.12 MB/s
minio        ██████████████████████████ 0.10 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████ 2.1ms
minio        ██████████████████████████████ 2.4ms
```

### Scale/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.11 MB/s | 21.2ms | 21.2ms | 21.2ms | 0 |
| minio | 0.07 MB/s | 35.6ms | 35.6ms | 35.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.11 MB/s
minio        █████████████████ 0.07 MB/s
```

**Latency (P50)**
```
liteio       █████████████████ 21.2ms
minio        ██████████████████████████████ 35.6ms
```

### Scale/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.09 MB/s | 286.3ms | 286.3ms | 286.3ms | 0 |
| minio | 0.07 MB/s | 367.9ms | 367.9ms | 367.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.09 MB/s
minio        ███████████████████████ 0.07 MB/s
```

**Latency (P50)**
```
liteio       ███████████████████████ 286.3ms
minio        ██████████████████████████████ 367.9ms
```

### Scale/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.09 MB/s | 2.62s | 2.62s | 2.62s | 0 |
| minio | 0.06 MB/s | 4.21s | 4.21s | 4.21s | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.09 MB/s
minio        ██████████████████ 0.06 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████ 2.62s
minio        ██████████████████████████████ 4.21s
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1157 ops/s | 700.1us | 1.8ms | 3.5ms | 0 |
| minio | 628 ops/s | 1.3ms | 3.5ms | 4.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1157 ops/s
minio        ████████████████ 628 ops/s
```

**Latency (P50)**
```
liteio       ████████████████ 700.1us
minio        ██████████████████████████████ 1.3ms
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 55.01 MB/s | 170.9ms | 170.9ms | 170.9ms | 0 |
| liteio | 54.90 MB/s | 182.3ms | 182.3ms | 182.3ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 55.01 MB/s
liteio       █████████████████████████████ 54.90 MB/s
```

**Latency (P50)**
```
minio        ████████████████████████████ 170.9ms
liteio       ██████████████████████████████ 182.3ms
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.38 MB/s | 2.3ms | 4.5ms | 7.8ms | 0 |
| minio | 0.28 MB/s | 3.2ms | 5.0ms | 6.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.38 MB/s
minio        █████████████████████ 0.28 MB/s
```

**Latency (P50)**
```
liteio       ████████████████████ 2.3ms
minio        ██████████████████████████████ 3.2ms
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 43.10 MB/s | 21.8ms | 33.7ms | 34.8ms | 0 |
| minio | 39.68 MB/s | 22.7ms | 33.6ms | 37.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 43.10 MB/s
minio        ███████████████████████████ 39.68 MB/s
```

**Latency (P50)**
```
liteio       ████████████████████████████ 21.8ms
minio        ██████████████████████████████ 22.7ms
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 12.46 MB/s | 4.2ms | 9.3ms | 12.9ms | 0 |
| minio | 10.78 MB/s | 4.8ms | 9.8ms | 18.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 12.46 MB/s
minio        █████████████████████████ 10.78 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████ 4.2ms
minio        ██████████████████████████████ 4.8ms
```

## Resource Usage

| Driver | Memory | RSS | Cache | CPU | Volume | Block I/O |
|--------|--------|-----|-------|-----|--------|----------|
| liteio | 453.4MiB / 7.653GiB | 453.4 MB | - | 2.7% | 1147.9 MB | 954kB / 404MB |
| minio | 491.7MiB / 7.653GiB | 491.7 MB | - | 0.0% | 359.0 MB | 36.9kB / 362MB |

> **Note:** RSS = actual application memory. Cache = OS page cache (reclaimable).

## Recommendations

- **Write-heavy workloads:** minio
- **Read-heavy workloads:** minio

---

*Generated by storage benchmark CLI*
