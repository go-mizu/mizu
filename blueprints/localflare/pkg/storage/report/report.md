# Storage Benchmark Report

**Generated:** 2026-01-15T11:35:12+07:00

**Go Version:** go1.25.5

**Platform:** darwin/arm64

## Executive Summary

### Summary

**Overall Winner:** liteio (won 33/43 benchmarks, 77%)

| Rank | Driver | Wins | Win Rate |
|------|--------|------|----------|
| 1 | liteio | 33 | 77% |
| 2 | minio | 10 | 23% |

### Performance Leaders

| Operation | Leader | Performance | Margin |
|-----------|--------|-------------|--------|
| Small Read (1KB) | liteio | 5.2 MB/s | +58% vs minio |
| Small Write (1KB) | liteio | 1.4 MB/s | close |
| Large Read (10MB) | minio | 298.3 MB/s | close |
| Large Write (10MB) | liteio | 187.0 MB/s | +16% vs minio |
| Delete | liteio | 5.3K ops/s | +62% vs minio |
| Stat | liteio | 5.7K ops/s | +43% vs minio |
| List (100 objects) | liteio | 1.3K ops/s | 2.6x vs minio |
| Copy | minio | 1.3 MB/s | 3.6x vs liteio |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (10MB+) | **liteio** | 187 MB/s | Best for media, backups |
| Large File Downloads (10MB) | **minio** | 298 MB/s | Best for streaming, CDN |
| Small File Operations | **liteio** | 3393 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **minio** | - | Best for multi-user apps |
| Memory Constrained | **liteio** | 23 MB RAM | Best for edge/embedded |

### Large File Performance (10MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| liteio | 187.0 | 291.3 | 52.9ms | 33.7ms |
| minio | 161.7 | 298.3 | 62.7ms | 33.1ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| liteio | 1454 | 5331 | 646.2us | 175.8us |
| minio | 1330 | 3366 | 710.4us | 297.2us |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| liteio | 5739 | 1337 | 5251 |
| minio | 4023 | 520 | 3246 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| liteio | 1.72 | 0.39 | 0.25 |
| minio | 1.39 | 0.26 | 0.20 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| liteio | 4.73 | 0.46 | 1.03 |
| minio | 2.90 | 0.81 | 0.52 |

*\* indicates errors occurred*

### File Count Performance

Performance with varying numbers of files (1KB each).

**Write N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|------|
| liteio | 569.8us | 4.7ms | 49.5ms | 485.0ms | 5.09s |
| minio | 661.0us | 6.9ms | 72.4ms | 849.6ms | 8.39s |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|------|
| liteio | 276.6us | 374.9us | 975.0us | 5.2ms | 186.6ms |
| minio | 388.0us | 588.8us | 1.8ms | 15.7ms | 162.7ms |

*\* indicates errors occurred*

### Resource Usage Summary

| Driver | Memory | CPU |
|--------|--------|-----|
| liteio | 22.8 MB | 0.0% |
| minio | 341.0 MB | 0.0% |

---

## Configuration

| Parameter | Value |
|-----------|-------|
| Iterations | 20 |
| Warmup | 5 |
| Concurrency | 200 |
| Timeout | 1m0s |

## Drivers Tested

- **liteio** (43 benchmarks)
- **minio** (43 benchmarks)

## Detailed Results

### Copy/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 1.29 MB/s | 667.6us | 887.6us | 887.6us | 0 |
| liteio | 0.36 MB/s | 506.6us | 3.5ms | 3.5ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 1.29 MB/s
liteio       ████████ 0.36 MB/s
```

**Latency (P50)**
```
minio        ██████████████████████████████ 667.6us
liteio       ██████████████████████ 506.6us
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 5251 ops/s | 176.5us | 252.5us | 252.5us | 0 |
| minio | 3246 ops/s | 295.7us | 354.9us | 354.9us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 5251 ops/s
minio        ██████████████████ 3246 ops/s
```

**Latency (P50)**
```
liteio       █████████████████ 176.5us
minio        ██████████████████████████████ 295.7us
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 0.14 MB/s | 664.5us | 737.2us | 737.2us | 0 |
| liteio | 0.11 MB/s | 532.2us | 1.0ms | 1.0ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 0.14 MB/s
liteio       ████████████████████████ 0.11 MB/s
```

**Latency (P50)**
```
minio        ██████████████████████████████ 664.5us
liteio       ████████████████████████ 532.2us
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 1436 ops/s | 637.3us | 805.9us | 805.9us | 0 |
| liteio | 937 ops/s | 689.8us | 1.3ms | 1.3ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 1436 ops/s
liteio       ███████████████████ 937 ops/s
```

**Latency (P50)**
```
minio        ███████████████████████████ 637.3us
liteio       ██████████████████████████████ 689.8us
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.17 MB/s | 532.6us | 608.5us | 608.5us | 0 |
| minio | 0.06 MB/s | 1.1ms | 1.4ms | 1.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.17 MB/s
minio        ██████████ 0.06 MB/s
```

**Latency (P50)**
```
liteio       ██████████████ 532.6us
minio        ██████████████████████████████ 1.1ms
```

### FileCount/Delete/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 4667 ops/s | 214.3us | 214.3us | 214.3us | 0 |
| minio | 2680 ops/s | 373.2us | 373.2us | 373.2us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4667 ops/s
minio        █████████████████ 2680 ops/s
```

**Latency (P50)**
```
liteio       █████████████████ 214.3us
minio        ██████████████████████████████ 373.2us
```

### FileCount/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 435 ops/s | 2.3ms | 2.3ms | 2.3ms | 0 |
| minio | 308 ops/s | 3.2ms | 3.2ms | 3.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 435 ops/s
minio        █████████████████████ 308 ops/s
```

**Latency (P50)**
```
liteio       █████████████████████ 2.3ms
minio        ██████████████████████████████ 3.2ms
```

### FileCount/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 46 ops/s | 21.8ms | 21.8ms | 21.8ms | 0 |
| minio | 28 ops/s | 36.2ms | 36.2ms | 36.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 46 ops/s
minio        ██████████████████ 28 ops/s
```

**Latency (P50)**
```
liteio       ██████████████████ 21.8ms
minio        ██████████████████████████████ 36.2ms
```

### FileCount/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 5 ops/s | 197.4ms | 197.4ms | 197.4ms | 0 |
| minio | 3 ops/s | 360.1ms | 360.1ms | 360.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 5 ops/s
minio        ████████████████ 3 ops/s
```

**Latency (P50)**
```
liteio       ████████████████ 197.4ms
minio        ██████████████████████████████ 360.1ms
```

### FileCount/Delete/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1 ops/s | 1.89s | 1.89s | 1.89s | 0 |
| minio | 0 ops/s | 3.56s | 3.56s | 3.56s | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1 ops/s
minio        ███████████████ 0 ops/s
```

**Latency (P50)**
```
liteio       ███████████████ 1.89s
minio        ██████████████████████████████ 3.56s
```

### FileCount/List/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 3616 ops/s | 276.6us | 276.6us | 276.6us | 0 |
| minio | 2577 ops/s | 388.0us | 388.0us | 388.0us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3616 ops/s
minio        █████████████████████ 2577 ops/s
```

**Latency (P50)**
```
liteio       █████████████████████ 276.6us
minio        ██████████████████████████████ 388.0us
```

### FileCount/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 2667 ops/s | 374.9us | 374.9us | 374.9us | 0 |
| minio | 1698 ops/s | 588.8us | 588.8us | 588.8us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 2667 ops/s
minio        ███████████████████ 1698 ops/s
```

**Latency (P50)**
```
liteio       ███████████████████ 374.9us
minio        ██████████████████████████████ 588.8us
```

### FileCount/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1026 ops/s | 975.0us | 975.0us | 975.0us | 0 |
| minio | 549 ops/s | 1.8ms | 1.8ms | 1.8ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1026 ops/s
minio        ████████████████ 549 ops/s
```

**Latency (P50)**
```
liteio       ████████████████ 975.0us
minio        ██████████████████████████████ 1.8ms
```

### FileCount/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 193 ops/s | 5.2ms | 5.2ms | 5.2ms | 0 |
| minio | 64 ops/s | 15.7ms | 15.7ms | 15.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 193 ops/s
minio        █████████ 64 ops/s
```

**Latency (P50)**
```
liteio       █████████ 5.2ms
minio        ██████████████████████████████ 15.7ms
```

### FileCount/List/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 6 ops/s | 162.7ms | 162.7ms | 162.7ms | 0 |
| liteio | 5 ops/s | 186.6ms | 186.6ms | 186.6ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 6 ops/s
liteio       ██████████████████████████ 5 ops/s
```

**Latency (P50)**
```
minio        ██████████████████████████ 162.7ms
liteio       ██████████████████████████████ 186.6ms
```

### FileCount/Write/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.71 MB/s | 569.8us | 569.8us | 569.8us | 0 |
| minio | 1.48 MB/s | 661.0us | 661.0us | 661.0us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.71 MB/s
minio        █████████████████████████ 1.48 MB/s
```

**Latency (P50)**
```
liteio       █████████████████████████ 569.8us
minio        ██████████████████████████████ 661.0us
```

### FileCount/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 2.09 MB/s | 4.7ms | 4.7ms | 4.7ms | 0 |
| minio | 1.41 MB/s | 6.9ms | 6.9ms | 6.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 2.09 MB/s
minio        ████████████████████ 1.41 MB/s
```

**Latency (P50)**
```
liteio       ████████████████████ 4.7ms
minio        ██████████████████████████████ 6.9ms
```

### FileCount/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.97 MB/s | 49.5ms | 49.5ms | 49.5ms | 0 |
| minio | 1.35 MB/s | 72.4ms | 72.4ms | 72.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.97 MB/s
minio        ████████████████████ 1.35 MB/s
```

**Latency (P50)**
```
liteio       ████████████████████ 49.5ms
minio        ██████████████████████████████ 72.4ms
```

### FileCount/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 2.01 MB/s | 485.0ms | 485.0ms | 485.0ms | 0 |
| minio | 1.15 MB/s | 849.6ms | 849.6ms | 849.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 2.01 MB/s
minio        █████████████████ 1.15 MB/s
```

**Latency (P50)**
```
liteio       █████████████████ 485.0ms
minio        ██████████████████████████████ 849.6ms
```

### FileCount/Write/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.92 MB/s | 5.09s | 5.09s | 5.09s | 0 |
| minio | 1.16 MB/s | 8.39s | 8.39s | 8.39s | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.92 MB/s
minio        ██████████████████ 1.16 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████ 5.09s
minio        ██████████████████████████████ 8.39s
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1337 ops/s | 715.2us | 879.5us | 879.5us | 0 |
| minio | 520 ops/s | 1.8ms | 2.3ms | 2.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1337 ops/s
minio        ███████████ 520 ops/s
```

**Latency (P50)**
```
liteio       ███████████ 715.2us
minio        ██████████████████████████████ 1.8ms
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 8.81 MB/s | 1.6ms | 2.5ms | 2.5ms | 0 |
| liteio | 6.55 MB/s | 2.6ms | 3.3ms | 3.3ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 8.81 MB/s
liteio       ██████████████████████ 6.55 MB/s
```

**Latency (P50)**
```
minio        ██████████████████ 1.6ms
liteio       ██████████████████████████████ 2.6ms
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 6.62 MB/s | 2.5ms | 2.7ms | 2.7ms | 0 |
| minio | 6.58 MB/s | 2.4ms | 2.6ms | 2.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 6.62 MB/s
minio        █████████████████████████████ 6.58 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████████ 2.5ms
minio        █████████████████████████████ 2.4ms
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 5.12 MB/s | 3.5ms | 3.9ms | 3.9ms | 0 |
| minio | 4.07 MB/s | 4.4ms | 5.0ms | 5.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 5.12 MB/s
minio        ███████████████████████ 4.07 MB/s
```

**Latency (P50)**
```
liteio       ███████████████████████ 3.5ms
minio        ██████████████████████████████ 4.4ms
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 158.99 MB/s | 93.3ms | 98.7ms | 98.7ms | 0 |
| liteio | 150.62 MB/s | 98.9ms | 100.8ms | 100.8ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 158.99 MB/s
liteio       ████████████████████████████ 150.62 MB/s
```

**Latency (P50)**
```
minio        ████████████████████████████ 93.3ms
liteio       ██████████████████████████████ 98.9ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 4.73 MB/s | 204.3us | 252.7us | 197.4us | 252.8us | 252.8us | 0 |
| minio | 2.90 MB/s | 336.9us | 379.7us | 322.2us | 379.8us | 379.8us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4.73 MB/s
minio        ██████████████████ 2.90 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████ 197.4us
minio        ██████████████████████████████ 322.2us
```

### ParallelRead/1KB/C10

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 0.81 MB/s | 1.2ms | 1.6ms | 1.0ms | 1.6ms | 1.6ms | 0 |
| liteio | 0.46 MB/s | 2.1ms | 4.0ms | 2.0ms | 4.0ms | 4.0ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 0.81 MB/s
liteio       █████████████████ 0.46 MB/s
```

**Latency (P50)**
```
minio        ███████████████ 1.0ms
liteio       ██████████████████████████████ 2.0ms
```

### ParallelRead/1KB/C50

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 1.03 MB/s | 940.6us | 1.2ms | 934.0us | 1.2ms | 1.2ms | 0 |
| minio | 0.52 MB/s | 1.9ms | 2.2ms | 1.9ms | 2.2ms | 2.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.03 MB/s
minio        ███████████████ 0.52 MB/s
```

**Latency (P50)**
```
liteio       ███████████████ 934.0us
minio        ██████████████████████████████ 1.9ms
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.72 MB/s | 563.6us | 694.0us | 694.0us | 0 |
| minio | 1.39 MB/s | 684.9us | 799.8us | 799.8us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.72 MB/s
minio        ████████████████████████ 1.39 MB/s
```

**Latency (P50)**
```
liteio       ████████████████████████ 563.6us
minio        ██████████████████████████████ 684.9us
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.39 MB/s | 1.9ms | 4.0ms | 4.0ms | 0 |
| minio | 0.26 MB/s | 3.1ms | 5.5ms | 5.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.39 MB/s
minio        ████████████████████ 0.26 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████ 1.9ms
minio        ██████████████████████████████ 3.1ms
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.25 MB/s | 3.9ms | 4.7ms | 4.7ms | 0 |
| minio | 0.20 MB/s | 5.0ms | 5.5ms | 5.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.25 MB/s
minio        ████████████████████████ 0.20 MB/s
```

**Latency (P50)**
```
liteio       ███████████████████████ 3.9ms
minio        ██████████████████████████████ 5.0ms
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 184.86 MB/s | 1.3ms | 1.9ms | 1.9ms | 0 |
| minio | 166.53 MB/s | 1.4ms | 2.2ms | 2.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 184.86 MB/s
minio        ███████████████████████████ 166.53 MB/s
```

**Latency (P50)**
```
liteio       ███████████████████████████ 1.3ms
minio        ██████████████████████████████ 1.4ms
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 140.78 MB/s | 1.6ms | 2.8ms | 2.8ms | 0 |
| minio | 96.18 MB/s | 1.7ms | 9.0ms | 9.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 140.78 MB/s
minio        ████████████████████ 96.18 MB/s
```

**Latency (P50)**
```
liteio       ████████████████████████████ 1.6ms
minio        ██████████████████████████████ 1.7ms
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 131.41 MB/s | 1.8ms | 2.6ms | 2.6ms | 0 |
| liteio | 126.88 MB/s | 1.9ms | 3.0ms | 3.0ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 131.41 MB/s
liteio       ████████████████████████████ 126.88 MB/s
```

**Latency (P50)**
```
minio        ████████████████████████████ 1.8ms
liteio       ██████████████████████████████ 1.9ms
```

### Read/10MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 298.35 MB/s | 1.2ms | 1.3ms | 33.1ms | 34.6ms | 34.6ms | 0 |
| liteio | 291.27 MB/s | 614.5us | 793.3us | 33.7ms | 36.3ms | 36.3ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 298.35 MB/s
liteio       █████████████████████████████ 291.27 MB/s
```

**Latency (P50)**
```
minio        █████████████████████████████ 33.1ms
liteio       ██████████████████████████████ 33.7ms
```

### Read/1KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 5.21 MB/s | 187.5us | 222.0us | 175.8us | 222.2us | 222.2us | 0 |
| minio | 3.29 MB/s | 297.0us | 328.9us | 297.2us | 329.0us | 329.0us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 5.21 MB/s
minio        ██████████████████ 3.29 MB/s
```

**Latency (P50)**
```
liteio       █████████████████ 175.8us
minio        ██████████████████████████████ 297.2us
```

### Read/1MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 279.16 MB/s | 383.0us | 579.6us | 3.5ms | 3.8ms | 3.8ms | 0 |
| minio | 243.19 MB/s | 1.0ms | 1.2ms | 4.1ms | 4.4ms | 4.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 279.16 MB/s
minio        ██████████████████████████ 243.19 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████ 3.5ms
minio        ██████████████████████████████ 4.1ms
```

### Read/64KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 137.39 MB/s | 269.1us | 441.5us | 412.7us | 601.1us | 709.0us | 0 |
| minio | 95.51 MB/s | 476.3us | 568.2us | 649.0us | 739.9us | 787.5us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 137.39 MB/s
minio        ████████████████████ 95.51 MB/s
```

**Latency (P50)**
```
liteio       ███████████████████ 412.7us
minio        ██████████████████████████████ 649.0us
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 5739 ops/s | 159.3us | 213.0us | 213.0us | 0 |
| minio | 4023 ops/s | 238.0us | 323.4us | 323.4us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 5739 ops/s
minio        █████████████████████ 4023 ops/s
```

**Latency (P50)**
```
liteio       ████████████████████ 159.3us
minio        ██████████████████████████████ 238.0us
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 187.01 MB/s | 52.9ms | 55.7ms | 55.7ms | 0 |
| minio | 161.72 MB/s | 62.7ms | 65.9ms | 65.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 187.01 MB/s
minio        █████████████████████████ 161.72 MB/s
```

**Latency (P50)**
```
liteio       █████████████████████████ 52.9ms
minio        ██████████████████████████████ 62.7ms
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.42 MB/s | 646.2us | 789.1us | 789.1us | 0 |
| minio | 1.30 MB/s | 710.4us | 1.0ms | 1.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.42 MB/s
minio        ███████████████████████████ 1.30 MB/s
```

**Latency (P50)**
```
liteio       ███████████████████████████ 646.2us
minio        ██████████████████████████████ 710.4us
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 160.44 MB/s | 6.1ms | 6.6ms | 6.6ms | 0 |
| minio | 138.26 MB/s | 7.2ms | 7.9ms | 7.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 160.44 MB/s
minio        █████████████████████████ 138.26 MB/s
```

**Latency (P50)**
```
liteio       █████████████████████████ 6.1ms
minio        ██████████████████████████████ 7.2ms
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 59.46 MB/s | 1.0ms | 1.2ms | 1.3ms | 0 |
| liteio | 59.21 MB/s | 954.9us | 1.6ms | 1.8ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 59.46 MB/s
liteio       █████████████████████████████ 59.21 MB/s
```

**Latency (P50)**
```
minio        ██████████████████████████████ 1.0ms
liteio       ███████████████████████████ 954.9us
```

## Resource Usage

| Driver | Memory | RSS | Cache | CPU | Volume | Block I/O |
|--------|--------|-----|-------|-----|--------|----------|
| liteio | 22.67MiB / 7.653GiB | 22.7 MB | - | 0.0% | 121.6 MB | 0B / 466MB |
| minio | 341MiB / 7.653GiB | 341.0 MB | - | 0.0% | 3533.8 MB | 54MB / 484MB |

> **Note:** RSS = actual application memory. Cache = OS page cache (reclaimable).

## Recommendations

- **Write-heavy workloads:** liteio
- **Read-heavy workloads:** minio

---

*Generated by storage benchmark CLI*
