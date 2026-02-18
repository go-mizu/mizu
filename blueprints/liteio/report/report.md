# Storage Benchmark Report

**Generated:** 2026-02-18T23:05:36+07:00

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
| Small Read (1KB) | liteio | 1.1 MB/s | +77% vs minio |
| Small Write (1KB) | liteio | 0.4 MB/s | +81% vs minio |
| Large Read (10MB) | liteio | 104.4 MB/s | close |
| Large Write (10MB) | minio | 53.7 MB/s | close |
| Delete | liteio | 1.2K ops/s | 2.1x vs minio |
| Stat | liteio | 1.1K ops/s | +38% vs minio |
| List (100 objects) | liteio | 368 ops/s | 2.0x vs minio |
| Copy | liteio | 0.3 MB/s | +32% vs minio |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (10MB+) | **minio** | 54 MB/s | Best for media, backups |
| Large File Downloads (10MB) | **liteio** | 104 MB/s | Best for streaming, CDN |
| Small File Operations | **liteio** | 763 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **liteio** | - | Best for multi-user apps |

### Large File Performance (10MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| liteio | 53.5 | 104.4 | 183.3ms | 91.1ms |
| minio | 53.7 | 103.7 | 193.2ms | 89.0ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| liteio | 390 | 1136 | 2.3ms | 777.0us |
| minio | 215 | 640 | 3.9ms | 1.3ms |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| liteio | 1136 | 368 | 1179 |
| minio | 823 | 184 | 569 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| liteio | 0.37 | 0.18 | 0.05 |
| minio | 0.24 | 0.08 | 0.02 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| liteio | 0.83 | 0.63 | 0.20 |
| minio | 0.60 | 0.36 | 0.08 |

*\* indicates errors occurred*

### Scale Performance

Performance with varying numbers of objects (256B each).

**Write N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 |
|--------|------|------|------|------|
| liteio | 2.6ms | 43.3ms | 292.8ms | 2.51s |
| minio | 4.0ms | 35.9ms | 315.7ms | 3.91s |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 |
|--------|------|------|------|------|
| liteio | 796.1us | 2.2ms | 2.7ms | 18.7ms |
| minio | 2.9ms | 2.3ms | 4.9ms | 53.3ms |

*\* indicates errors occurred*

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
| liteio | 0.35 MB/s | 2.4ms | 5.1ms | 6.6ms | 0 |
| minio | 0.26 MB/s | 3.6ms | 5.0ms | 5.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.35 MB/s
minio        ██████████████████████ 0.26 MB/s
```

**Latency (P50)**
```
liteio       ███████████████████ 2.4ms
minio        ██████████████████████████████ 3.6ms
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1179 ops/s | 744.7us | 1.4ms | 2.5ms | 0 |
| minio | 569 ops/s | 1.6ms | 2.9ms | 4.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1179 ops/s
minio        ██████████████ 569 ops/s
```

**Latency (P50)**
```
liteio       ██████████████ 744.7us
minio        ██████████████████████████████ 1.6ms
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.04 MB/s | 2.3ms | 4.6ms | 6.1ms | 0 |
| minio | 0.02 MB/s | 4.6ms | 9.6ms | 12.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.04 MB/s
minio        █████████████ 0.02 MB/s
```

**Latency (P50)**
```
liteio       ███████████████ 2.3ms
minio        ██████████████████████████████ 4.6ms
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 463 ops/s | 1.9ms | 3.6ms | 4.3ms | 0 |
| minio | 197 ops/s | 4.8ms | 8.1ms | 9.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 463 ops/s
minio        ████████████ 197 ops/s
```

**Latency (P50)**
```
liteio       ████████████ 1.9ms
minio        ██████████████████████████████ 4.8ms
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.04 MB/s | 2.3ms | 3.7ms | 5.4ms | 0 |
| minio | 0.02 MB/s | 4.8ms | 8.6ms | 10.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.04 MB/s
minio        ██████████████ 0.02 MB/s
```

**Latency (P50)**
```
liteio       ██████████████ 2.3ms
minio        ██████████████████████████████ 4.8ms
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 368 ops/s | 2.5ms | 4.0ms | 7.2ms | 0 |
| minio | 184 ops/s | 5.3ms | 6.4ms | 7.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 368 ops/s
minio        ██████████████ 184 ops/s
```

**Latency (P50)**
```
liteio       ██████████████ 2.5ms
minio        ██████████████████████████████ 5.3ms
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.23 MB/s | 41.2ms | 189.0ms | 347.5ms | 0 |
| minio | 0.07 MB/s | 84.0ms | 645.3ms | 827.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.23 MB/s
minio        █████████ 0.07 MB/s
```

**Latency (P50)**
```
liteio       ██████████████ 41.2ms
minio        ██████████████████████████████ 84.0ms
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.48 MB/s | 28.4ms | 68.2ms | 99.4ms | 0 |
| minio | 0.16 MB/s | 65.9ms | 376.2ms | 536.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.48 MB/s
minio        █████████ 0.16 MB/s
```

**Latency (P50)**
```
liteio       ████████████ 28.4ms
minio        ██████████████████████████████ 65.9ms
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.08 MB/s | 75.0ms | 856.2ms | 864.2ms | 0 |
| minio | 0.06 MB/s | 249.4ms | 495.9ms | 692.8ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.08 MB/s
minio        ██████████████████████ 0.06 MB/s
```

**Latency (P50)**
```
liteio       █████████ 75.0ms
minio        ██████████████████████████████ 249.4ms
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 46.92 MB/s | 313.6ms | 313.6ms | 313.6ms | 0 |
| minio | 43.94 MB/s | 323.5ms | 323.5ms | 323.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 46.92 MB/s
minio        ████████████████████████████ 43.94 MB/s
```

**Latency (P50)**
```
liteio       █████████████████████████████ 313.6ms
minio        ██████████████████████████████ 323.5ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.83 MB/s | 947.0us | 2.3ms | 4.7ms | 0 |
| minio | 0.60 MB/s | 1.5ms | 2.4ms | 3.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.83 MB/s
minio        █████████████████████ 0.60 MB/s
```

**Latency (P50)**
```
liteio       ███████████████████ 947.0us
minio        ██████████████████████████████ 1.5ms
```

### ParallelRead/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.63 MB/s | 1.3ms | 3.0ms | 5.0ms | 0 |
| minio | 0.36 MB/s | 2.3ms | 4.8ms | 9.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.63 MB/s
minio        █████████████████ 0.36 MB/s
```

**Latency (P50)**
```
liteio       ████████████████ 1.3ms
minio        ██████████████████████████████ 2.3ms
```

### ParallelRead/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.20 MB/s | 3.5ms | 11.0ms | 43.7ms | 0 |
| minio | 0.08 MB/s | 10.3ms | 25.1ms | 37.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.20 MB/s
minio        ████████████ 0.08 MB/s
```

**Latency (P50)**
```
liteio       ██████████ 3.5ms
minio        ██████████████████████████████ 10.3ms
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.37 MB/s | 2.5ms | 4.4ms | 5.6ms | 0 |
| minio | 0.24 MB/s | 3.6ms | 7.0ms | 9.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.37 MB/s
minio        ███████████████████ 0.24 MB/s
```

**Latency (P50)**
```
liteio       ████████████████████ 2.5ms
minio        ██████████████████████████████ 3.6ms
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.18 MB/s | 4.8ms | 9.4ms | 15.9ms | 0 |
| minio | 0.08 MB/s | 9.2ms | 29.0ms | 35.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.18 MB/s
minio        █████████████ 0.08 MB/s
```

**Latency (P50)**
```
liteio       ███████████████ 4.8ms
minio        ██████████████████████████████ 9.2ms
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.05 MB/s | 14.1ms | 46.6ms | 150.0ms | 0 |
| minio | 0.02 MB/s | 36.2ms | 147.5ms | 182.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.05 MB/s
minio        ██████████ 0.02 MB/s
```

**Latency (P50)**
```
liteio       ███████████ 14.1ms
minio        ██████████████████████████████ 36.2ms
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 58.85 MB/s | 3.9ms | 6.0ms | 9.2ms | 0 |
| minio | 46.08 MB/s | 5.2ms | 6.6ms | 7.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 58.85 MB/s
minio        ███████████████████████ 46.08 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████ 3.9ms
minio        ██████████████████████████████ 5.2ms
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 59.54 MB/s | 3.8ms | 6.4ms | 8.8ms | 0 |
| minio | 39.51 MB/s | 6.0ms | 8.5ms | 10.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 59.54 MB/s
minio        ███████████████████ 39.51 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████ 3.8ms
minio        ██████████████████████████████ 6.0ms
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 64.91 MB/s | 3.7ms | 5.7ms | 7.4ms | 0 |
| minio | 35.42 MB/s | 6.0ms | 10.1ms | 12.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 64.91 MB/s
minio        ████████████████ 35.42 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████ 3.7ms
minio        ██████████████████████████████ 6.0ms
```

### Read/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 104.39 MB/s | 91.1ms | 112.3ms | 112.3ms | 0 |
| minio | 103.70 MB/s | 89.0ms | 110.7ms | 110.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 104.39 MB/s
minio        █████████████████████████████ 103.70 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 91.1ms
minio        █████████████████████████████ 89.0ms
```

### Read/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.11 MB/s | 777.0us | 1.5ms | 2.3ms | 0 |
| minio | 0.62 MB/s | 1.3ms | 2.7ms | 5.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.11 MB/s
minio        ████████████████ 0.62 MB/s
```

**Latency (P50)**
```
liteio       █████████████████ 777.0us
minio        ██████████████████████████████ 1.3ms
```

### Read/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 83.98 MB/s | 11.2ms | 14.4ms | 20.3ms | 0 |
| minio | 71.99 MB/s | 13.7ms | 15.8ms | 16.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 83.98 MB/s
minio        █████████████████████████ 71.99 MB/s
```

**Latency (P50)**
```
liteio       ████████████████████████ 11.2ms
minio        ██████████████████████████████ 13.7ms
```

### Read/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 39.49 MB/s | 1.5ms | 2.1ms | 3.0ms | 0 |
| minio | 22.02 MB/s | 2.4ms | 4.4ms | 6.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 39.49 MB/s
minio        ████████████████ 22.02 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████ 1.5ms
minio        ██████████████████████████████ 2.4ms
```

### Scale/Delete/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1362 ops/s | 734.0us | 734.0us | 734.0us | 0 |
| minio | 398 ops/s | 2.5ms | 2.5ms | 2.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1362 ops/s
minio        ████████ 398 ops/s
```

**Latency (P50)**
```
liteio       ████████ 734.0us
minio        ██████████████████████████████ 2.5ms
```

### Scale/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 108 ops/s | 9.2ms | 9.2ms | 9.2ms | 0 |
| minio | 58 ops/s | 17.3ms | 17.3ms | 17.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 108 ops/s
minio        ████████████████ 58 ops/s
```

**Latency (P50)**
```
liteio       ████████████████ 9.2ms
minio        ██████████████████████████████ 17.3ms
```

### Scale/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 8 ops/s | 120.0ms | 120.0ms | 120.0ms | 0 |
| minio | 6 ops/s | 167.2ms | 167.2ms | 167.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 8 ops/s
minio        █████████████████████ 6 ops/s
```

**Latency (P50)**
```
liteio       █████████████████████ 120.0ms
minio        ██████████████████████████████ 167.2ms
```

### Scale/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1 ops/s | 897.6ms | 897.6ms | 897.6ms | 0 |
| minio | 1 ops/s | 1.95s | 1.95s | 1.95s | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1 ops/s
minio        █████████████ 1 ops/s
```

**Latency (P50)**
```
liteio       █████████████ 897.6ms
minio        ██████████████████████████████ 1.95s
```

### Scale/List/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1256 ops/s | 796.1us | 796.1us | 796.1us | 0 |
| minio | 344 ops/s | 2.9ms | 2.9ms | 2.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1256 ops/s
minio        ████████ 344 ops/s
```

**Latency (P50)**
```
liteio       ████████ 796.1us
minio        ██████████████████████████████ 2.9ms
```

### Scale/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 448 ops/s | 2.2ms | 2.2ms | 2.2ms | 0 |
| minio | 439 ops/s | 2.3ms | 2.3ms | 2.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 448 ops/s
minio        █████████████████████████████ 439 ops/s
```

**Latency (P50)**
```
liteio       █████████████████████████████ 2.2ms
minio        ██████████████████████████████ 2.3ms
```

### Scale/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 366 ops/s | 2.7ms | 2.7ms | 2.7ms | 0 |
| minio | 205 ops/s | 4.9ms | 4.9ms | 4.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 366 ops/s
minio        ████████████████ 205 ops/s
```

**Latency (P50)**
```
liteio       ████████████████ 2.7ms
minio        ██████████████████████████████ 4.9ms
```

### Scale/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 53 ops/s | 18.7ms | 18.7ms | 18.7ms | 0 |
| minio | 19 ops/s | 53.3ms | 53.3ms | 53.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 53 ops/s
minio        ██████████ 19 ops/s
```

**Latency (P50)**
```
liteio       ██████████ 18.7ms
minio        ██████████████████████████████ 53.3ms
```

### Scale/Write/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.09 MB/s | 2.6ms | 2.6ms | 2.6ms | 0 |
| minio | 0.06 MB/s | 4.0ms | 4.0ms | 4.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.09 MB/s
minio        ███████████████████ 0.06 MB/s
```

**Latency (P50)**
```
liteio       ███████████████████ 2.6ms
minio        ██████████████████████████████ 4.0ms
```

### Scale/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 0.07 MB/s | 35.9ms | 35.9ms | 35.9ms | 0 |
| liteio | 0.06 MB/s | 43.3ms | 43.3ms | 43.3ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 0.07 MB/s
liteio       ████████████████████████ 0.06 MB/s
```

**Latency (P50)**
```
minio        ████████████████████████ 35.9ms
liteio       ██████████████████████████████ 43.3ms
```

### Scale/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.08 MB/s | 292.8ms | 292.8ms | 292.8ms | 0 |
| minio | 0.08 MB/s | 315.7ms | 315.7ms | 315.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.08 MB/s
minio        ███████████████████████████ 0.08 MB/s
```

**Latency (P50)**
```
liteio       ███████████████████████████ 292.8ms
minio        ██████████████████████████████ 315.7ms
```

### Scale/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.10 MB/s | 2.51s | 2.51s | 2.51s | 0 |
| minio | 0.06 MB/s | 3.91s | 3.91s | 3.91s | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.10 MB/s
minio        ███████████████████ 0.06 MB/s
```

**Latency (P50)**
```
liteio       ███████████████████ 2.51s
minio        ██████████████████████████████ 3.91s
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1136 ops/s | 747.5us | 1.6ms | 2.6ms | 0 |
| minio | 823 ops/s | 1.1ms | 2.1ms | 3.8ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1136 ops/s
minio        █████████████████████ 823 ops/s
```

**Latency (P50)**
```
liteio       █████████████████████ 747.5us
minio        ██████████████████████████████ 1.1ms
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 53.70 MB/s | 193.2ms | 193.2ms | 193.2ms | 0 |
| liteio | 53.52 MB/s | 183.3ms | 183.3ms | 183.3ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 53.70 MB/s
liteio       █████████████████████████████ 53.52 MB/s
```

**Latency (P50)**
```
minio        ██████████████████████████████ 193.2ms
liteio       ████████████████████████████ 183.3ms
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.38 MB/s | 2.3ms | 4.0ms | 4.9ms | 0 |
| minio | 0.21 MB/s | 3.9ms | 8.7ms | 14.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.38 MB/s
minio        ████████████████ 0.21 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████ 2.3ms
minio        ██████████████████████████████ 3.9ms
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 46.61 MB/s | 20.4ms | 29.2ms | 29.6ms | 0 |
| minio | 36.75 MB/s | 26.4ms | 31.4ms | 36.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 46.61 MB/s
minio        ███████████████████████ 36.75 MB/s
```

**Latency (P50)**
```
liteio       ███████████████████████ 20.4ms
minio        ██████████████████████████████ 26.4ms
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 15.53 MB/s | 3.7ms | 6.6ms | 8.1ms | 0 |
| minio | 9.60 MB/s | 5.9ms | 10.2ms | 13.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 15.53 MB/s
minio        ██████████████████ 9.60 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████ 3.7ms
minio        ██████████████████████████████ 5.9ms
```

## Recommendations

- **Write-heavy workloads:** minio
- **Read-heavy workloads:** liteio

---

*Generated by storage benchmark CLI*
