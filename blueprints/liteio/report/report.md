# Storage Benchmark Report

**Generated:** 2026-02-18T23:29:57+07:00

**Go Version:** go1.26.0

**Platform:** darwin/arm64

## Executive Summary

### Summary

**Overall Winner:** liteio (won 40/40 benchmarks, 100%)

| Rank | Driver | Wins | Win Rate |
|------|--------|------|----------|
| 1 | liteio | 40 | 100% |

### Performance Leaders

| Operation | Leader | Performance | Margin |
|-----------|--------|-------------|--------|
| Small Read (1KB) | liteio | 1.3 MB/s | +96% vs minio |
| Small Write (1KB) | liteio | 1.4 MB/s | 8.7x vs minio |
| Large Read (10MB) | liteio | 114.5 MB/s | +18% vs minio |
| Large Write (10MB) | liteio | 63.3 MB/s | close |
| Delete | liteio | 1.1K ops/s | +97% vs minio |
| Stat | liteio | 1.5K ops/s | +71% vs minio |
| List (100 objects) | liteio | 384 ops/s | +96% vs minio |
| Copy | liteio | 0.9 MB/s | 4.1x vs minio |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (10MB+) | **liteio** | 63 MB/s | Best for media, backups |
| Large File Downloads (10MB) | **liteio** | 114 MB/s | Best for streaming, CDN |
| Small File Operations | **liteio** | 1370 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **liteio** | - | Best for multi-user apps |
| Memory Constrained | **minio** | 888 MB RAM | Best for edge/embedded |

### Large File Performance (10MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| liteio | 63.3 | 114.5 | 144.8ms | 86.1ms |
| minio | 58.1 | 97.3 | 174.3ms | 97.8ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| liteio | 1436 | 1304 | 644.5us | 660.5us |
| minio | 165 | 665 | 5.5ms | 1.3ms |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| liteio | 1456 | 384 | 1143 |
| minio | 850 | 196 | 579 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| liteio | 0.93 | 0.70 | 0.21 |
| minio | 0.19 | 0.07 | 0.01 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| liteio | 1.22 | 0.92 | 0.25 |
| minio | 0.57 | 0.38 | 0.08 |

*\* indicates errors occurred*

### Scale Performance

Performance with varying numbers of objects (256B each).

**Write N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 |
|--------|------|------|------|------|
| liteio | 2.5ms | 7.8ms | 90.4ms | 856.3ms |
| minio | 6.8ms | 59.2ms | 397.4ms | 4.24s |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 |
|--------|------|------|------|------|
| liteio | 1.7ms | 879.6us | 2.3ms | 12.9ms |
| minio | 6.2ms | 3.8ms | 8.0ms | 32.8ms |

*\* indicates errors occurred*

### Resource Usage Summary

| Driver | Memory | CPU |
|--------|--------|-----|
| liteio | 947.6 MB | 2.6% |
| minio | 888.5 MB | 0.0% |

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
| liteio | 0.86 MB/s | 902.7us | 2.3ms | 3.6ms | 0 |
| minio | 0.21 MB/s | 4.1ms | 8.1ms | 11.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.86 MB/s
minio        ███████ 0.21 MB/s
```

**Latency (P50)**
```
liteio       ██████ 902.7us
minio        ██████████████████████████████ 4.1ms
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1143 ops/s | 735.8us | 1.6ms | 2.8ms | 0 |
| minio | 579 ops/s | 1.4ms | 2.4ms | 6.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1143 ops/s
minio        ███████████████ 579 ops/s
```

**Latency (P50)**
```
liteio       ███████████████ 735.8us
minio        ██████████████████████████████ 1.4ms
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.11 MB/s | 783.9us | 1.5ms | 2.7ms | 0 |
| minio | 0.02 MB/s | 5.1ms | 13.2ms | 20.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.11 MB/s
minio        ████ 0.02 MB/s
```

**Latency (P50)**
```
liteio       ████ 783.9us
minio        ██████████████████████████████ 5.1ms
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1009 ops/s | 746.9us | 2.3ms | 4.9ms | 0 |
| minio | 224 ops/s | 4.3ms | 7.1ms | 9.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1009 ops/s
minio        ██████ 224 ops/s
```

**Latency (P50)**
```
liteio       █████ 746.9us
minio        ██████████████████████████████ 4.3ms
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.10 MB/s | 791.0us | 1.6ms | 2.8ms | 0 |
| minio | 0.02 MB/s | 4.6ms | 7.5ms | 10.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.10 MB/s
minio        █████ 0.02 MB/s
```

**Latency (P50)**
```
liteio       █████ 791.0us
minio        ██████████████████████████████ 4.6ms
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 384 ops/s | 2.3ms | 4.1ms | 6.3ms | 0 |
| minio | 196 ops/s | 5.1ms | 6.0ms | 6.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 384 ops/s
minio        ███████████████ 196 ops/s
```

**Latency (P50)**
```
liteio       █████████████ 2.3ms
minio        ██████████████████████████████ 5.1ms
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.49 MB/s | 22.5ms | 72.3ms | 242.6ms | 0 |
| minio | 0.11 MB/s | 116.0ms | 316.0ms | 393.8ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.49 MB/s
minio        ███████ 0.11 MB/s
```

**Latency (P50)**
```
liteio       █████ 22.5ms
minio        ██████████████████████████████ 116.0ms
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.52 MB/s | 27.5ms | 63.1ms | 114.2ms | 0 |
| minio | 0.22 MB/s | 61.9ms | 164.3ms | 292.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.52 MB/s
minio        ████████████ 0.22 MB/s
```

**Latency (P50)**
```
liteio       █████████████ 27.5ms
minio        ██████████████████████████████ 61.9ms
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.32 MB/s | 23.0ms | 103.8ms | 834.4ms | 0 |
| minio | 0.07 MB/s | 194.1ms | 424.3ms | 464.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.32 MB/s
minio        ███████ 0.07 MB/s
```

**Latency (P50)**
```
liteio       ███ 23.0ms
minio        ██████████████████████████████ 194.1ms
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 58.12 MB/s | 247.9ms | 247.9ms | 247.9ms | 0 |
| minio | 52.29 MB/s | 283.1ms | 283.1ms | 283.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 58.12 MB/s
minio        ██████████████████████████ 52.29 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████ 247.9ms
minio        ██████████████████████████████ 283.1ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.22 MB/s | 768.0us | 1.0ms | 1.3ms | 0 |
| minio | 0.57 MB/s | 1.4ms | 3.2ms | 6.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.22 MB/s
minio        ██████████████ 0.57 MB/s
```

**Latency (P50)**
```
liteio       ████████████████ 768.0us
minio        ██████████████████████████████ 1.4ms
```

### ParallelRead/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.92 MB/s | 968.7us | 1.8ms | 2.4ms | 0 |
| minio | 0.38 MB/s | 2.3ms | 4.3ms | 8.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.92 MB/s
minio        ████████████ 0.38 MB/s
```

**Latency (P50)**
```
liteio       ████████████ 968.7us
minio        ██████████████████████████████ 2.3ms
```

### ParallelRead/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.25 MB/s | 3.0ms | 7.6ms | 28.7ms | 0 |
| minio | 0.08 MB/s | 10.7ms | 28.4ms | 51.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.25 MB/s
minio        █████████ 0.08 MB/s
```

**Latency (P50)**
```
liteio       ████████ 3.0ms
minio        ██████████████████████████████ 10.7ms
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.93 MB/s | 936.0us | 1.6ms | 2.7ms | 0 |
| minio | 0.19 MB/s | 4.7ms | 8.8ms | 10.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.93 MB/s
minio        ██████ 0.19 MB/s
```

**Latency (P50)**
```
liteio       █████ 936.0us
minio        ██████████████████████████████ 4.7ms
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.70 MB/s | 1.3ms | 2.4ms | 3.6ms | 0 |
| minio | 0.07 MB/s | 12.1ms | 31.6ms | 42.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.70 MB/s
minio        ██ 0.07 MB/s
```

**Latency (P50)**
```
liteio       ███ 1.3ms
minio        ██████████████████████████████ 12.1ms
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.21 MB/s | 3.5ms | 9.7ms | 26.8ms | 0 |
| minio | 0.01 MB/s | 81.8ms | 466.0ms | 525.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.21 MB/s
minio        █ 0.01 MB/s
```

**Latency (P50)**
```
liteio       █ 3.5ms
minio        ██████████████████████████████ 81.8ms
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 59.48 MB/s | 3.9ms | 5.9ms | 7.2ms | 0 |
| minio | 37.95 MB/s | 6.2ms | 8.4ms | 11.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 59.48 MB/s
minio        ███████████████████ 37.95 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████ 3.9ms
minio        ██████████████████████████████ 6.2ms
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 60.17 MB/s | 3.9ms | 5.3ms | 7.8ms | 0 |
| minio | 40.79 MB/s | 5.8ms | 8.1ms | 12.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 60.17 MB/s
minio        ████████████████████ 40.79 MB/s
```

**Latency (P50)**
```
liteio       ████████████████████ 3.9ms
minio        ██████████████████████████████ 5.8ms
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 73.39 MB/s | 3.4ms | 4.8ms | 6.3ms | 0 |
| minio | 41.72 MB/s | 5.6ms | 7.6ms | 15.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 73.39 MB/s
minio        █████████████████ 41.72 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████ 3.4ms
minio        ██████████████████████████████ 5.6ms
```

### Read/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 114.47 MB/s | 86.1ms | 99.2ms | 99.2ms | 0 |
| minio | 97.25 MB/s | 97.8ms | 119.9ms | 119.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 114.47 MB/s
minio        █████████████████████████ 97.25 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████ 86.1ms
minio        ██████████████████████████████ 97.8ms
```

### Read/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.27 MB/s | 660.5us | 1.5ms | 2.3ms | 0 |
| minio | 0.65 MB/s | 1.3ms | 2.5ms | 4.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.27 MB/s
minio        ███████████████ 0.65 MB/s
```

**Latency (P50)**
```
liteio       ██████████████ 660.5us
minio        ██████████████████████████████ 1.3ms
```

### Read/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 76.44 MB/s | 12.8ms | 16.1ms | 17.9ms | 0 |
| minio | 75.02 MB/s | 12.9ms | 16.0ms | 17.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 76.44 MB/s
minio        █████████████████████████████ 75.02 MB/s
```

**Latency (P50)**
```
liteio       █████████████████████████████ 12.8ms
minio        ██████████████████████████████ 12.9ms
```

### Read/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 39.66 MB/s | 1.5ms | 2.2ms | 2.4ms | 0 |
| minio | 25.65 MB/s | 2.2ms | 3.8ms | 6.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 39.66 MB/s
minio        ███████████████████ 25.65 MB/s
```

**Latency (P50)**
```
liteio       ████████████████████ 1.5ms
minio        ██████████████████████████████ 2.2ms
```

### Scale/Delete/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 461 ops/s | 2.2ms | 2.2ms | 2.2ms | 0 |
| minio | 322 ops/s | 3.1ms | 3.1ms | 3.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 461 ops/s
minio        ████████████████████ 322 ops/s
```

**Latency (P50)**
```
liteio       ████████████████████ 2.2ms
minio        ██████████████████████████████ 3.1ms
```

### Scale/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 140 ops/s | 7.1ms | 7.1ms | 7.1ms | 0 |
| minio | 45 ops/s | 22.4ms | 22.4ms | 22.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 140 ops/s
minio        █████████ 45 ops/s
```

**Latency (P50)**
```
liteio       █████████ 7.1ms
minio        ██████████████████████████████ 22.4ms
```

### Scale/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 6 ops/s | 155.2ms | 155.2ms | 155.2ms | 0 |
| minio | 6 ops/s | 178.4ms | 178.4ms | 178.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 6 ops/s
minio        ██████████████████████████ 6 ops/s
```

**Latency (P50)**
```
liteio       ██████████████████████████ 155.2ms
minio        ██████████████████████████████ 178.4ms
```

### Scale/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1 ops/s | 823.4ms | 823.4ms | 823.4ms | 0 |
| minio | 1 ops/s | 1.85s | 1.85s | 1.85s | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1 ops/s
minio        █████████████ 1 ops/s
```

**Latency (P50)**
```
liteio       █████████████ 823.4ms
minio        ██████████████████████████████ 1.85s
```

### Scale/List/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 577 ops/s | 1.7ms | 1.7ms | 1.7ms | 0 |
| minio | 161 ops/s | 6.2ms | 6.2ms | 6.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 577 ops/s
minio        ████████ 161 ops/s
```

**Latency (P50)**
```
liteio       ████████ 1.7ms
minio        ██████████████████████████████ 6.2ms
```

### Scale/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1137 ops/s | 879.6us | 879.6us | 879.6us | 0 |
| minio | 263 ops/s | 3.8ms | 3.8ms | 3.8ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1137 ops/s
minio        ██████ 263 ops/s
```

**Latency (P50)**
```
liteio       ██████ 879.6us
minio        ██████████████████████████████ 3.8ms
```

### Scale/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 438 ops/s | 2.3ms | 2.3ms | 2.3ms | 0 |
| minio | 125 ops/s | 8.0ms | 8.0ms | 8.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 438 ops/s
minio        ████████ 125 ops/s
```

**Latency (P50)**
```
liteio       ████████ 2.3ms
minio        ██████████████████████████████ 8.0ms
```

### Scale/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 77 ops/s | 12.9ms | 12.9ms | 12.9ms | 0 |
| minio | 30 ops/s | 32.8ms | 32.8ms | 32.8ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 77 ops/s
minio        ███████████ 30 ops/s
```

**Latency (P50)**
```
liteio       ███████████ 12.9ms
minio        ██████████████████████████████ 32.8ms
```

### Scale/Write/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.10 MB/s | 2.5ms | 2.5ms | 2.5ms | 0 |
| minio | 0.04 MB/s | 6.8ms | 6.8ms | 6.8ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.10 MB/s
minio        ██████████ 0.04 MB/s
```

**Latency (P50)**
```
liteio       ██████████ 2.5ms
minio        ██████████████████████████████ 6.8ms
```

### Scale/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.31 MB/s | 7.8ms | 7.8ms | 7.8ms | 0 |
| minio | 0.04 MB/s | 59.2ms | 59.2ms | 59.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.31 MB/s
minio        ███ 0.04 MB/s
```

**Latency (P50)**
```
liteio       ███ 7.8ms
minio        ██████████████████████████████ 59.2ms
```

### Scale/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.27 MB/s | 90.4ms | 90.4ms | 90.4ms | 0 |
| minio | 0.06 MB/s | 397.4ms | 397.4ms | 397.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.27 MB/s
minio        ██████ 0.06 MB/s
```

**Latency (P50)**
```
liteio       ██████ 90.4ms
minio        ██████████████████████████████ 397.4ms
```

### Scale/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.29 MB/s | 856.3ms | 856.3ms | 856.3ms | 0 |
| minio | 0.06 MB/s | 4.24s | 4.24s | 4.24s | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.29 MB/s
minio        ██████ 0.06 MB/s
```

**Latency (P50)**
```
liteio       ██████ 856.3ms
minio        ██████████████████████████████ 4.24s
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1456 ops/s | 637.7us | 1.2ms | 1.6ms | 0 |
| minio | 850 ops/s | 1.0ms | 2.1ms | 3.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1456 ops/s
minio        █████████████████ 850 ops/s
```

**Latency (P50)**
```
liteio       ██████████████████ 637.7us
minio        ██████████████████████████████ 1.0ms
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 63.28 MB/s | 144.8ms | 164.6ms | 164.6ms | 0 |
| minio | 58.11 MB/s | 174.3ms | 174.3ms | 174.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 63.28 MB/s
minio        ███████████████████████████ 58.11 MB/s
```

**Latency (P50)**
```
liteio       ████████████████████████ 144.8ms
minio        ██████████████████████████████ 174.3ms
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.40 MB/s | 644.5us | 1.1ms | 1.7ms | 0 |
| minio | 0.16 MB/s | 5.5ms | 8.3ms | 10.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.40 MB/s
minio        ███ 0.16 MB/s
```

**Latency (P50)**
```
liteio       ███ 644.5us
minio        ██████████████████████████████ 5.5ms
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 56.72 MB/s | 16.9ms | 22.8ms | 23.2ms | 0 |
| minio | 32.91 MB/s | 27.4ms | 42.6ms | 43.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 56.72 MB/s
minio        █████████████████ 32.91 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████ 16.9ms
minio        ██████████████████████████████ 27.4ms
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 30.03 MB/s | 1.9ms | 3.4ms | 4.5ms | 0 |
| minio | 8.17 MB/s | 7.4ms | 13.9ms | 15.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 30.03 MB/s
minio        ████████ 8.17 MB/s
```

**Latency (P50)**
```
liteio       ███████ 1.9ms
minio        ██████████████████████████████ 7.4ms
```

## Resource Usage

| Driver | Memory | RSS | Cache | CPU | Volume | Block I/O |
|--------|--------|-----|-------|-----|--------|----------|
| liteio | 944.6MiB / 7.653GiB | 944.6 MB | - | 2.6% | (no data) | 0B / 0B |
| minio | 888.8MiB / 7.653GiB | 888.8 MB | - | 0.0% | 717.0 MB | 11.4MB / 2.37GB |

> **Note:** RSS = actual application memory. Cache = OS page cache (reclaimable).

## Recommendations

- **Write-heavy workloads:** liteio
- **Read-heavy workloads:** liteio

---

*Generated by storage benchmark CLI*
