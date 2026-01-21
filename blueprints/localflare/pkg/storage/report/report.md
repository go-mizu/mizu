# Storage Benchmark Report

**Generated:** 2026-01-21T18:43:40+07:00

**Go Version:** go1.25.5

**Platform:** darwin/arm64

## Executive Summary

### Summary

**Overall Winner:** devnull_s3 (won 33/48 benchmarks, 69%)

| Rank | Driver | Wins | Win Rate |
|------|--------|------|----------|
| 1 | devnull_s3 | 33 | 69% |
| 2 | liteio | 13 | 27% |
| 3 | minio | 2 | 4% |

### Performance Leaders

| Operation | Leader | Performance | Margin |
|-----------|--------|-------------|--------|
| Small Read (1KB) | devnull_s3 | 4.1 MB/s | close |
| Small Write (1KB) | devnull_s3 | 1.8 MB/s | close |
| Large Read (100MB) | devnull_s3 | 175.0 MB/s | close |
| Large Write (100MB) | devnull_s3 | 136.8 MB/s | +10% vs liteio |
| Delete | devnull_s3 | 4.1K ops/s | close |
| Stat | devnull_s3 | 4.1K ops/s | close |
| List (100 objects) | devnull_s3 | 952 ops/s | +15% vs liteio |
| Copy | devnull_s3 | 1.4 MB/s | close |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (100MB+) | **devnull_s3** | 137 MB/s | Best for media, backups |
| Large File Downloads (100MB) | **devnull_s3** | 175 MB/s | Best for streaming, CDN |
| Small File Operations | **devnull_s3** | 3061 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **devnull_s3** | - | Best for multi-user apps |
| Memory Constrained | **minio** | 768 MB RAM | Best for edge/embedded |

### Large File Performance (100MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| devnull_s3 | 136.8 | 175.0 | 734.0ms | 582.6ms |
| liteio | 124.1 | 158.1 | 786.5ms | 621.8ms |
| minio | 117.4 | 160.8 | 860.4ms | 634.5ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| devnull_s3 | 1881 | 4241 | 512.8us | 223.1us |
| liteio | 1801 | 4063 | 516.2us | 229.2us |
| minio | 1190 | 2749 | 769.3us | 345.8us |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| devnull_s3 | 4105 | 952 | 4115 |
| liteio | 3981 | 828 | 3979 |
| minio | 2804 | 488 | 2392 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| devnull_s3 | 1.55 | 0.44 | 0.21 | 0.11 | 0.06 | 0.03 |
| liteio | 1.29 | 0.40 | 0.18 | 0.11 | 0.05 | 0.03 |
| minio | 1.10 | 0.24 | 0.10 | 0.05 | 0.02 | 0.01 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| devnull_s3 | 3.40 | 1.12 | 0.59 | 0.38 | 0.20 | 0.10 |
| liteio | 3.24 | 1.07 | 0.59 | 0.35 | 0.20 | 0.10 |
| minio | 2.31 | 0.80 | 0.39 | 0.20 | 0.11 | 0.06 |

*\* indicates errors occurred*

### Scale Performance

Performance with varying numbers of objects (256B each).

**Write N Files (total time)**

| Driver | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|
| devnull_s3 | 6.0ms | 56.5ms | 637.1ms | 5.85s |
| liteio | 5.7ms | 55.8ms | 569.6ms | 5.91s |
| minio | 15.0ms | 121.7ms | 1.01s | 10.03s |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|
| devnull_s3 | 401.1us | 1.1ms | 6.0ms | 207.1ms |
| liteio | 427.5us | 1.1ms | 6.0ms | 247.7ms |
| minio | 1.1ms | 2.2ms | 14.1ms | 182.4ms |

*\* indicates errors occurred*

### Resource Usage Summary

| Driver | Memory | CPU |
|--------|--------|-----|
| devnull_s3 | 1387.5 MB | 3.8% |
| liteio | 1675.3 MB | 2.6% |
| minio | 767.9 MB | 7.7% |

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

- **devnull_s3** (48 benchmarks)
- **liteio** (48 benchmarks)
- **minio** (48 benchmarks)

## Detailed Results

### Copy/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 1.38 MB/s | 582.7us | 1.2ms | 1.8ms | 0 |
| liteio | 1.25 MB/s | 618.3us | 1.4ms | 2.1ms | 0 |
| minio | 0.88 MB/s | 987.5us | 1.8ms | 2.8ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 1.38 MB/s
liteio       ███████████████████████████ 1.25 MB/s
minio        ███████████████████ 0.88 MB/s
```

**Latency (P50)**
```
devnull_s3   █████████████████ 582.7us
liteio       ██████████████████ 618.3us
minio        ██████████████████████████████ 987.5us
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 4115 ops/s | 226.7us | 338.1us | 517.1us | 0 |
| liteio | 3979 ops/s | 230.5us | 357.9us | 597.8us | 0 |
| minio | 2392 ops/s | 393.1us | 521.5us | 780.6us | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 4115 ops/s
liteio       █████████████████████████████ 3979 ops/s
minio        █████████████████ 2392 ops/s
```

**Latency (P50)**
```
devnull_s3   █████████████████ 226.7us
liteio       █████████████████ 230.5us
minio        ██████████████████████████████ 393.1us
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 0.16 MB/s | 545.9us | 791.1us | 1.5ms | 0 |
| liteio | 0.16 MB/s | 549.5us | 779.3us | 936.8us | 0 |
| minio | 0.08 MB/s | 1.0ms | 1.5ms | 2.4ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 0.16 MB/s
liteio       █████████████████████████████ 0.16 MB/s
minio        ███████████████ 0.08 MB/s
```

**Latency (P50)**
```
devnull_s3   ███████████████ 545.9us
liteio       ███████████████ 549.5us
minio        ██████████████████████████████ 1.0ms
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 2121 ops/s | 453.2us | 582.5us | 900.3us | 0 |
| liteio | 1846 ops/s | 512.8us | 722.3us | 1.1ms | 0 |
| minio | 940 ops/s | 898.3us | 1.8ms | 2.9ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 2121 ops/s
liteio       ██████████████████████████ 1846 ops/s
minio        █████████████ 940 ops/s
```

**Latency (P50)**
```
devnull_s3   ███████████████ 453.2us
liteio       █████████████████ 512.8us
minio        ██████████████████████████████ 898.3us
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 0.16 MB/s | 586.5us | 847.2us | 1.3ms | 0 |
| liteio | 0.14 MB/s | 633.3us | 973.6us | 1.9ms | 0 |
| minio | 0.08 MB/s | 1.0ms | 1.8ms | 3.7ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 0.16 MB/s
liteio       ███████████████████████████ 0.14 MB/s
minio        ███████████████ 0.08 MB/s
```

**Latency (P50)**
```
devnull_s3   █████████████████ 586.5us
liteio       ██████████████████ 633.3us
minio        ██████████████████████████████ 1.0ms
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 952 ops/s | 1.0ms | 1.3ms | 1.7ms | 0 |
| liteio | 828 ops/s | 1.1ms | 1.7ms | 2.7ms | 0 |
| minio | 488 ops/s | 1.9ms | 2.6ms | 4.7ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 952 ops/s
liteio       ██████████████████████████ 828 ops/s
minio        ███████████████ 488 ops/s
```

**Latency (P50)**
```
devnull_s3   ███████████████ 1.0ms
liteio       █████████████████ 1.1ms
minio        ██████████████████████████████ 1.9ms
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.55 MB/s | 23.6ms | 47.7ms | 63.0ms | 0 |
| devnull_s3 | 0.50 MB/s | 25.0ms | 54.5ms | 69.9ms | 0 |
| minio | 0.28 MB/s | 36.4ms | 141.2ms | 226.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.55 MB/s
devnull_s3   ██████████████████████████ 0.50 MB/s
minio        ███████████████ 0.28 MB/s
```

**Latency (P50)**
```
liteio       ███████████████████ 23.6ms
devnull_s3   ████████████████████ 25.0ms
minio        ██████████████████████████████ 36.4ms
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.67 MB/s | 22.7ms | 33.0ms | 43.6ms | 0 |
| devnull_s3 | 0.57 MB/s | 26.3ms | 39.5ms | 48.4ms | 0 |
| minio | 0.49 MB/s | 21.9ms | 109.0ms | 191.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.67 MB/s
devnull_s3   █████████████████████████ 0.57 MB/s
minio        ██████████████████████ 0.49 MB/s
```

**Latency (P50)**
```
liteio       █████████████████████████ 22.7ms
devnull_s3   ██████████████████████████████ 26.3ms
minio        ████████████████████████ 21.9ms
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.45 MB/s | 28.7ms | 54.2ms | 242.2ms | 0 |
| devnull_s3 | 0.36 MB/s | 37.2ms | 74.0ms | 255.6ms | 0 |
| minio | 0.15 MB/s | 82.2ms | 226.6ms | 816.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.45 MB/s
devnull_s3   ████████████████████████ 0.36 MB/s
minio        ██████████ 0.15 MB/s
```

**Latency (P50)**
```
liteio       ██████████ 28.7ms
devnull_s3   █████████████ 37.2ms
minio        ██████████████████████████████ 82.2ms
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 121.62 MB/s | 123.6ms | 136.3ms | 136.3ms | 0 |
| devnull_s3 | 110.32 MB/s | 135.7ms | 139.1ms | 139.1ms | 0 |
| liteio | 109.80 MB/s | 125.8ms | 149.5ms | 149.5ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 121.62 MB/s
devnull_s3   ███████████████████████████ 110.32 MB/s
liteio       ███████████████████████████ 109.80 MB/s
```

**Latency (P50)**
```
minio        ███████████████████████████ 123.6ms
devnull_s3   ██████████████████████████████ 135.7ms
liteio       ███████████████████████████ 125.8ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 3.40 MB/s | 265.5us | 384.3us | 616.6us | 0 |
| liteio | 3.24 MB/s | 272.0us | 433.1us | 852.7us | 0 |
| minio | 2.31 MB/s | 395.4us | 550.2us | 891.1us | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 3.40 MB/s
liteio       ████████████████████████████ 3.24 MB/s
minio        ████████████████████ 2.31 MB/s
```

**Latency (P50)**
```
devnull_s3   ████████████████████ 265.5us
liteio       ████████████████████ 272.0us
minio        ██████████████████████████████ 395.4us
```

### ParallelRead/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 1.12 MB/s | 836.8us | 1.3ms | 1.8ms | 0 |
| liteio | 1.07 MB/s | 874.4us | 1.4ms | 1.9ms | 0 |
| minio | 0.80 MB/s | 1.2ms | 1.8ms | 2.7ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 1.12 MB/s
liteio       ████████████████████████████ 1.07 MB/s
minio        █████████████████████ 0.80 MB/s
```

**Latency (P50)**
```
devnull_s3   █████████████████████ 836.8us
liteio       ██████████████████████ 874.4us
minio        ██████████████████████████████ 1.2ms
```

### ParallelRead/1KB/C100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 0.20 MB/s | 4.6ms | 7.5ms | 10.0ms | 0 |
| liteio | 0.20 MB/s | 4.9ms | 7.3ms | 9.2ms | 0 |
| minio | 0.11 MB/s | 8.4ms | 17.4ms | 25.7ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 0.20 MB/s
liteio       ████████████████████████████ 0.20 MB/s
minio        ███████████████ 0.11 MB/s
```

**Latency (P50)**
```
devnull_s3   ████████████████ 4.6ms
liteio       █████████████████ 4.9ms
minio        ██████████████████████████████ 8.4ms
```

### ParallelRead/1KB/C200

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 0.10 MB/s | 10.0ms | 14.1ms | 17.4ms | 0 |
| liteio | 0.10 MB/s | 10.1ms | 14.5ms | 19.3ms | 0 |
| minio | 0.06 MB/s | 15.6ms | 35.2ms | 54.1ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 0.10 MB/s
liteio       █████████████████████████████ 0.10 MB/s
minio        █████████████████ 0.06 MB/s
```

**Latency (P50)**
```
devnull_s3   ███████████████████ 10.0ms
liteio       ███████████████████ 10.1ms
minio        ██████████████████████████████ 15.6ms
```

### ParallelRead/1KB/C25

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.59 MB/s | 1.6ms | 2.6ms | 3.6ms | 0 |
| devnull_s3 | 0.59 MB/s | 1.6ms | 2.5ms | 3.5ms | 0 |
| minio | 0.39 MB/s | 2.3ms | 4.2ms | 6.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.59 MB/s
devnull_s3   █████████████████████████████ 0.59 MB/s
minio        ███████████████████ 0.39 MB/s
```

**Latency (P50)**
```
liteio       ████████████████████ 1.6ms
devnull_s3   ████████████████████ 1.6ms
minio        ██████████████████████████████ 2.3ms
```

### ParallelRead/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 0.38 MB/s | 2.5ms | 3.9ms | 5.0ms | 0 |
| liteio | 0.35 MB/s | 2.6ms | 4.6ms | 7.3ms | 0 |
| minio | 0.20 MB/s | 4.2ms | 10.1ms | 15.5ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 0.38 MB/s
liteio       ███████████████████████████ 0.35 MB/s
minio        ███████████████ 0.20 MB/s
```

**Latency (P50)**
```
devnull_s3   █████████████████ 2.5ms
liteio       ██████████████████ 2.6ms
minio        ██████████████████████████████ 4.2ms
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 1.55 MB/s | 605.1us | 767.1us | 943.9us | 0 |
| liteio | 1.29 MB/s | 708.8us | 1.1ms | 1.7ms | 0 |
| minio | 1.10 MB/s | 842.2us | 1.0ms | 1.2ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 1.55 MB/s
liteio       ████████████████████████ 1.29 MB/s
minio        █████████████████████ 1.10 MB/s
```

**Latency (P50)**
```
devnull_s3   █████████████████████ 605.1us
liteio       █████████████████████████ 708.8us
minio        ██████████████████████████████ 842.2us
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 0.44 MB/s | 2.1ms | 3.0ms | 4.4ms | 0 |
| liteio | 0.40 MB/s | 2.4ms | 3.5ms | 5.1ms | 0 |
| minio | 0.24 MB/s | 3.6ms | 7.0ms | 10.8ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 0.44 MB/s
liteio       ██████████████████████████ 0.40 MB/s
minio        ████████████████ 0.24 MB/s
```

**Latency (P50)**
```
devnull_s3   █████████████████ 2.1ms
liteio       ███████████████████ 2.4ms
minio        ██████████████████████████████ 3.6ms
```

### ParallelWrite/1KB/C100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 0.06 MB/s | 15.6ms | 36.6ms | 47.0ms | 0 |
| liteio | 0.05 MB/s | 16.3ms | 40.4ms | 56.4ms | 0 |
| minio | 0.02 MB/s | 39.6ms | 73.4ms | 97.5ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 0.06 MB/s
liteio       ███████████████████████████ 0.05 MB/s
minio        ████████████ 0.02 MB/s
```

**Latency (P50)**
```
devnull_s3   ███████████ 15.6ms
liteio       ████████████ 16.3ms
minio        ██████████████████████████████ 39.6ms
```

### ParallelWrite/1KB/C200

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.03 MB/s | 29.8ms | 70.5ms | 103.3ms | 0 |
| devnull_s3 | 0.03 MB/s | 32.4ms | 71.3ms | 101.1ms | 0 |
| minio | 0.01 MB/s | 64.5ms | 129.8ms | 338.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.03 MB/s
devnull_s3   ████████████████████████████ 0.03 MB/s
minio        █████████████ 0.01 MB/s
```

**Latency (P50)**
```
liteio       █████████████ 29.8ms
devnull_s3   ███████████████ 32.4ms
minio        ██████████████████████████████ 64.5ms
```

### ParallelWrite/1KB/C25

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 0.21 MB/s | 4.4ms | 8.4ms | 13.5ms | 0 |
| liteio | 0.18 MB/s | 4.8ms | 10.5ms | 15.9ms | 0 |
| minio | 0.10 MB/s | 9.3ms | 16.3ms | 20.4ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 0.21 MB/s
liteio       ██████████████████████████ 0.18 MB/s
minio        ██████████████ 0.10 MB/s
```

**Latency (P50)**
```
devnull_s3   ██████████████ 4.4ms
liteio       ███████████████ 4.8ms
minio        ██████████████████████████████ 9.3ms
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 0.11 MB/s | 8.1ms | 17.0ms | 27.3ms | 0 |
| liteio | 0.11 MB/s | 8.0ms | 18.4ms | 25.3ms | 0 |
| minio | 0.05 MB/s | 16.0ms | 40.2ms | 55.7ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 0.11 MB/s
liteio       █████████████████████████████ 0.11 MB/s
minio        ██████████████ 0.05 MB/s
```

**Latency (P50)**
```
devnull_s3   ███████████████ 8.1ms
liteio       ██████████████ 8.0ms
minio        ██████████████████████████████ 16.0ms
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 151.22 MB/s | 1.6ms | 2.1ms | 2.5ms | 0 |
| liteio | 145.06 MB/s | 1.6ms | 2.2ms | 3.0ms | 0 |
| minio | 113.30 MB/s | 2.1ms | 2.9ms | 4.8ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 151.22 MB/s
liteio       ████████████████████████████ 145.06 MB/s
minio        ██████████████████████ 113.30 MB/s
```

**Latency (P50)**
```
devnull_s3   ███████████████████████ 1.6ms
liteio       ███████████████████████ 1.6ms
minio        ██████████████████████████████ 2.1ms
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 151.06 MB/s | 1.6ms | 2.0ms | 2.6ms | 0 |
| liteio | 144.65 MB/s | 1.6ms | 2.2ms | 2.9ms | 0 |
| minio | 119.07 MB/s | 2.0ms | 2.5ms | 3.1ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 151.06 MB/s
liteio       ████████████████████████████ 144.65 MB/s
minio        ███████████████████████ 119.07 MB/s
```

**Latency (P50)**
```
devnull_s3   ███████████████████████ 1.6ms
liteio       ████████████████████████ 1.6ms
minio        ██████████████████████████████ 2.0ms
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 133.52 MB/s | 1.8ms | 2.5ms | 3.7ms | 0 |
| liteio | 131.99 MB/s | 1.8ms | 2.7ms | 3.7ms | 0 |
| minio | 103.65 MB/s | 2.3ms | 3.2ms | 3.6ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 133.52 MB/s
liteio       █████████████████████████████ 131.99 MB/s
minio        ███████████████████████ 103.65 MB/s
```

**Latency (P50)**
```
devnull_s3   ███████████████████████ 1.8ms
liteio       ███████████████████████ 1.8ms
minio        ██████████████████████████████ 2.3ms
```

### Read/100MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 174.98 MB/s | 582.6ms | 582.6ms | 582.6ms | 0 |
| minio | 160.78 MB/s | 634.5ms | 634.5ms | 634.5ms | 0 |
| liteio | 158.05 MB/s | 621.8ms | 621.8ms | 621.8ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 174.98 MB/s
minio        ███████████████████████████ 160.78 MB/s
liteio       ███████████████████████████ 158.05 MB/s
```

**Latency (P50)**
```
devnull_s3   ███████████████████████████ 582.6ms
minio        ██████████████████████████████ 634.5ms
liteio       █████████████████████████████ 621.8ms
```

### Read/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 184.46 MB/s | 53.7ms | 56.5ms | 59.8ms | 0 |
| minio | 178.06 MB/s | 54.7ms | 60.3ms | 61.8ms | 0 |
| liteio | 162.76 MB/s | 61.2ms | 72.1ms | 72.1ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 184.46 MB/s
minio        ████████████████████████████ 178.06 MB/s
liteio       ██████████████████████████ 162.76 MB/s
```

**Latency (P50)**
```
devnull_s3   ██████████████████████████ 53.7ms
minio        ██████████████████████████ 54.7ms
liteio       ██████████████████████████████ 61.2ms
```

### Read/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 4.14 MB/s | 223.1us | 322.6us | 383.5us | 0 |
| liteio | 3.97 MB/s | 229.2us | 357.9us | 481.2us | 0 |
| minio | 2.68 MB/s | 345.8us | 457.8us | 679.9us | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 4.14 MB/s
liteio       ████████████████████████████ 3.97 MB/s
minio        ███████████████████ 2.68 MB/s
```

**Latency (P50)**
```
devnull_s3   ███████████████████ 223.1us
liteio       ███████████████████ 229.2us
minio        ██████████████████████████████ 345.8us
```

### Read/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 173.26 MB/s | 5.7ms | 6.7ms | 8.1ms | 0 |
| minio | 157.38 MB/s | 6.2ms | 7.1ms | 10.4ms | 0 |
| liteio | 150.46 MB/s | 6.3ms | 8.7ms | 11.0ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 173.26 MB/s
minio        ███████████████████████████ 157.38 MB/s
liteio       ██████████████████████████ 150.46 MB/s
```

**Latency (P50)**
```
devnull_s3   ██████████████████████████ 5.7ms
minio        █████████████████████████████ 6.2ms
liteio       ██████████████████████████████ 6.3ms
```

### Read/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 97.86 MB/s | 587.1us | 933.8us | 1.5ms | 0 |
| liteio | 96.46 MB/s | 605.6us | 923.0us | 1.3ms | 0 |
| minio | 78.64 MB/s | 724.0us | 1.1ms | 1.9ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 97.86 MB/s
liteio       █████████████████████████████ 96.46 MB/s
minio        ████████████████████████ 78.64 MB/s
```

**Latency (P50)**
```
devnull_s3   ████████████████████████ 587.1us
liteio       █████████████████████████ 605.6us
minio        ██████████████████████████████ 724.0us
```

### Scale/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 417 ops/s | 2.4ms | 2.4ms | 2.4ms | 0 |
| devnull_s3 | 417 ops/s | 2.4ms | 2.4ms | 2.4ms | 0 |
| minio | 118 ops/s | 8.5ms | 8.5ms | 8.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 417 ops/s
devnull_s3   █████████████████████████████ 417 ops/s
minio        ████████ 118 ops/s
```

**Latency (P50)**
```
liteio       ████████ 2.4ms
devnull_s3   ████████ 2.4ms
minio        ██████████████████████████████ 8.5ms
```

### Scale/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 41 ops/s | 24.4ms | 24.4ms | 24.4ms | 0 |
| devnull_s3 | 40 ops/s | 25.1ms | 25.1ms | 25.1ms | 0 |
| minio | 24 ops/s | 41.4ms | 41.4ms | 41.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 41 ops/s
devnull_s3   █████████████████████████████ 40 ops/s
minio        █████████████████ 24 ops/s
```

**Latency (P50)**
```
liteio       █████████████████ 24.4ms
devnull_s3   ██████████████████ 25.1ms
minio        ██████████████████████████████ 41.4ms
```

### Scale/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 4 ops/s | 237.9ms | 237.9ms | 237.9ms | 0 |
| devnull_s3 | 4 ops/s | 253.7ms | 253.7ms | 253.7ms | 0 |
| minio | 2 ops/s | 428.1ms | 428.1ms | 428.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4 ops/s
devnull_s3   ████████████████████████████ 4 ops/s
minio        ████████████████ 2 ops/s
```

**Latency (P50)**
```
liteio       ████████████████ 237.9ms
devnull_s3   █████████████████ 253.7ms
minio        ██████████████████████████████ 428.1ms
```

### Scale/Delete/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 0 ops/s | 2.43s | 2.43s | 2.43s | 0 |
| liteio | 0 ops/s | 2.49s | 2.49s | 2.49s | 0 |
| minio | 0 ops/s | 4.08s | 4.08s | 4.08s | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 0 ops/s
liteio       █████████████████████████████ 0 ops/s
minio        █████████████████ 0 ops/s
```

**Latency (P50)**
```
devnull_s3   █████████████████ 2.43s
liteio       ██████████████████ 2.49s
minio        ██████████████████████████████ 4.08s
```

### Scale/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 2493 ops/s | 401.1us | 401.1us | 401.1us | 0 |
| liteio | 2339 ops/s | 427.5us | 427.5us | 427.5us | 0 |
| minio | 882 ops/s | 1.1ms | 1.1ms | 1.1ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 2493 ops/s
liteio       ████████████████████████████ 2339 ops/s
minio        ██████████ 882 ops/s
```

**Latency (P50)**
```
devnull_s3   ██████████ 401.1us
liteio       ███████████ 427.5us
minio        ██████████████████████████████ 1.1ms
```

### Scale/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 902 ops/s | 1.1ms | 1.1ms | 1.1ms | 0 |
| liteio | 897 ops/s | 1.1ms | 1.1ms | 1.1ms | 0 |
| minio | 453 ops/s | 2.2ms | 2.2ms | 2.2ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 902 ops/s
liteio       █████████████████████████████ 897 ops/s
minio        ███████████████ 453 ops/s
```

**Latency (P50)**
```
devnull_s3   ███████████████ 1.1ms
liteio       ███████████████ 1.1ms
minio        ██████████████████████████████ 2.2ms
```

### Scale/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 166 ops/s | 6.0ms | 6.0ms | 6.0ms | 0 |
| devnull_s3 | 166 ops/s | 6.0ms | 6.0ms | 6.0ms | 0 |
| minio | 71 ops/s | 14.1ms | 14.1ms | 14.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 166 ops/s
devnull_s3   ██████████████████████████████ 166 ops/s
minio        ████████████ 71 ops/s
```

**Latency (P50)**
```
liteio       ████████████ 6.0ms
devnull_s3   ████████████ 6.0ms
minio        ██████████████████████████████ 14.1ms
```

### Scale/List/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 5 ops/s | 182.4ms | 182.4ms | 182.4ms | 0 |
| devnull_s3 | 5 ops/s | 207.1ms | 207.1ms | 207.1ms | 0 |
| liteio | 4 ops/s | 247.7ms | 247.7ms | 247.7ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 5 ops/s
devnull_s3   ██████████████████████████ 5 ops/s
liteio       ██████████████████████ 4 ops/s
```

**Latency (P50)**
```
minio        ██████████████████████ 182.4ms
devnull_s3   █████████████████████████ 207.1ms
liteio       ██████████████████████████████ 247.7ms
```

### Scale/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.43 MB/s | 5.7ms | 5.7ms | 5.7ms | 0 |
| devnull_s3 | 0.41 MB/s | 6.0ms | 6.0ms | 6.0ms | 0 |
| minio | 0.16 MB/s | 15.0ms | 15.0ms | 15.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.43 MB/s
devnull_s3   ████████████████████████████ 0.41 MB/s
minio        ███████████ 0.16 MB/s
```

**Latency (P50)**
```
liteio       ███████████ 5.7ms
devnull_s3   ███████████ 6.0ms
minio        ██████████████████████████████ 15.0ms
```

### Scale/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.44 MB/s | 55.8ms | 55.8ms | 55.8ms | 0 |
| devnull_s3 | 0.43 MB/s | 56.5ms | 56.5ms | 56.5ms | 0 |
| minio | 0.20 MB/s | 121.7ms | 121.7ms | 121.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.44 MB/s
devnull_s3   █████████████████████████████ 0.43 MB/s
minio        █████████████ 0.20 MB/s
```

**Latency (P50)**
```
liteio       █████████████ 55.8ms
devnull_s3   █████████████ 56.5ms
minio        ██████████████████████████████ 121.7ms
```

### Scale/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.43 MB/s | 569.6ms | 569.6ms | 569.6ms | 0 |
| devnull_s3 | 0.38 MB/s | 637.1ms | 637.1ms | 637.1ms | 0 |
| minio | 0.24 MB/s | 1.01s | 1.01s | 1.01s | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.43 MB/s
devnull_s3   ██████████████████████████ 0.38 MB/s
minio        ████████████████ 0.24 MB/s
```

**Latency (P50)**
```
liteio       ████████████████ 569.6ms
devnull_s3   ██████████████████ 637.1ms
minio        ██████████████████████████████ 1.01s
```

### Scale/Write/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 0.42 MB/s | 5.85s | 5.85s | 5.85s | 0 |
| liteio | 0.41 MB/s | 5.91s | 5.91s | 5.91s | 0 |
| minio | 0.24 MB/s | 10.03s | 10.03s | 10.03s | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 0.42 MB/s
liteio       █████████████████████████████ 0.41 MB/s
minio        █████████████████ 0.24 MB/s
```

**Latency (P50)**
```
devnull_s3   █████████████████ 5.85s
liteio       █████████████████ 5.91s
minio        ██████████████████████████████ 10.03s
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 4105 ops/s | 223.0us | 348.9us | 632.7us | 0 |
| liteio | 3981 ops/s | 234.0us | 363.0us | 511.9us | 0 |
| minio | 2804 ops/s | 334.7us | 486.9us | 679.3us | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 4105 ops/s
liteio       █████████████████████████████ 3981 ops/s
minio        ████████████████████ 2804 ops/s
```

**Latency (P50)**
```
devnull_s3   ███████████████████ 223.0us
liteio       ████████████████████ 234.0us
minio        ██████████████████████████████ 334.7us
```

### Write/100MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 136.76 MB/s | 734.0ms | 734.0ms | 734.0ms | 0 |
| liteio | 124.11 MB/s | 786.5ms | 786.5ms | 786.5ms | 0 |
| minio | 117.43 MB/s | 860.4ms | 860.4ms | 860.4ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 136.76 MB/s
liteio       ███████████████████████████ 124.11 MB/s
minio        █████████████████████████ 117.43 MB/s
```

**Latency (P50)**
```
devnull_s3   █████████████████████████ 734.0ms
liteio       ███████████████████████████ 786.5ms
minio        ██████████████████████████████ 860.4ms
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 136.36 MB/s | 73.6ms | 77.0ms | 77.0ms | 0 |
| devnull_s3 | 127.40 MB/s | 77.5ms | 87.1ms | 87.1ms | 0 |
| minio | 114.72 MB/s | 86.3ms | 94.7ms | 94.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 136.36 MB/s
devnull_s3   ████████████████████████████ 127.40 MB/s
minio        █████████████████████████ 114.72 MB/s
```

**Latency (P50)**
```
liteio       █████████████████████████ 73.6ms
devnull_s3   ██████████████████████████ 77.5ms
minio        ██████████████████████████████ 86.3ms
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 1.84 MB/s | 512.8us | 653.2us | 964.4us | 0 |
| liteio | 1.76 MB/s | 516.2us | 778.8us | 1.2ms | 0 |
| minio | 1.16 MB/s | 769.3us | 1.1ms | 2.3ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 1.84 MB/s
liteio       ████████████████████████████ 1.76 MB/s
minio        ██████████████████ 1.16 MB/s
```

**Latency (P50)**
```
devnull_s3   ███████████████████ 512.8us
liteio       ████████████████████ 516.2us
minio        ██████████████████████████████ 769.3us
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 114.44 MB/s | 8.2ms | 11.3ms | 12.6ms | 0 |
| liteio | 112.58 MB/s | 8.3ms | 12.4ms | 14.8ms | 0 |
| minio | 110.54 MB/s | 8.5ms | 12.7ms | 17.9ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 114.44 MB/s
liteio       █████████████████████████████ 112.58 MB/s
minio        ████████████████████████████ 110.54 MB/s
```

**Latency (P50)**
```
devnull_s3   █████████████████████████████ 8.2ms
liteio       █████████████████████████████ 8.3ms
minio        ██████████████████████████████ 8.5ms
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 56.91 MB/s | 1.0ms | 1.5ms | 2.5ms | 0 |
| liteio | 56.84 MB/s | 998.7us | 1.7ms | 2.7ms | 0 |
| minio | 43.40 MB/s | 1.4ms | 1.9ms | 2.3ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 56.91 MB/s
liteio       █████████████████████████████ 56.84 MB/s
minio        ██████████████████████ 43.40 MB/s
```

**Latency (P50)**
```
devnull_s3   ██████████████████████ 1.0ms
liteio       █████████████████████ 998.7us
minio        ██████████████████████████████ 1.4ms
```

## Resource Usage

| Driver | Memory | RSS | Cache | CPU | Volume | Block I/O |
|--------|--------|-----|-------|-----|--------|----------|
| devnull_s3 | 1.355GiB / 7.653GiB | 1387.5 MB | - | 3.8% | 5425.2 MB | 2.72MB / 2.16GB |
| liteio | 1.636GiB / 7.653GiB | 1675.3 MB | - | 2.6% | (no data) | 291kB / 3.28GB |
| minio | 768.1MiB / 7.653GiB | 768.1 MB | - | 7.7% | 1939.0 MB | 60.2MB / 2.17GB |

> **Note:** RSS = actual application memory. Cache = OS page cache (reclaimable).

## Recommendations

- **Write-heavy workloads:** devnull_s3
- **Read-heavy workloads:** devnull_s3

---

*Generated by storage benchmark CLI*
