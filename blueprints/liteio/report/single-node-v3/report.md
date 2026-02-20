# Storage Benchmark Report

**Generated:** 2026-02-20T14:49:03+07:00

**Go Version:** go1.26.0

**Platform:** darwin/arm64

## Executive Summary

### Summary

**Overall Winner:** herd_s3 (won 27/48 benchmarks, 56%)

| Rank | Driver | Wins | Win Rate |
|------|--------|------|----------|
| 1 | herd_s3 | 27 | 56% |
| 2 | liteio | 21 | 44% |

### Performance Leaders

| Operation | Leader | Performance | Margin |
|-----------|--------|-------------|--------|
| Small Read (1KB) | liteio | 4.5 MB/s | +29% vs herd_s3 |
| Small Write (1KB) | herd_s3 | 4.4 MB/s | 3.0x vs liteio |
| Large Read (100MB) | liteio | 183.0 MB/s | close |
| Large Write (100MB) | herd_s3 | 172.2 MB/s | +38% vs liteio |
| Delete | herd_s3 | 5.1K ops/s | close |
| Stat | herd_s3 | 4.5K ops/s | +24% vs liteio |
| List (100 objects) | herd_s3 | 1.4K ops/s | +47% vs liteio |
| Copy | herd_s3 | 4.5 MB/s | 3.2x vs liteio |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (100MB+) | **herd_s3** | 172 MB/s | Best for media, backups |
| Large File Downloads (100MB) | **liteio** | 183 MB/s | Best for streaming, CDN |
| Small File Operations | **herd_s3** | 4075 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **herd_s3** | - | Best for multi-user apps |
| Memory Constrained | **minio** | 916 MB RAM | Best for edge/embedded |

### Large File Performance (100MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| herd_s3 | 172.2 | 169.4 | 560.8ms | 568.6ms |
| liteio | 125.2 | 183.0 | 780.0ms | 522.4ms |
| minio | 101.4 | 170.5 | 1.01s | 571.3ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| herd_s3 | 4544 | 3606 | 189.6us | 199.5us |
| liteio | 1521 | 4643 | 604.8us | 193.7us |
| minio | 632 | 2718 | 1.4ms | 347.8us |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| herd_s3 | 4520 | 1441 | 5097 |
| liteio | 3658 | 979 | 4883 |
| minio | 3202 | 598 | 2618 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| herd_s3 | 3.42 | 0.99 | 0.53 | 0.25 | 0.14 | 0.07 |
| liteio | 1.40 | 0.49 | 0.28 | 0.13 | 0.11 | 0.05 |
| minio | 0.62 | 0.17 | 0.08 | 0.03 | 0.01 | 0.01 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| herd_s3 | 3.80 | 1.06 | 0.56 | 0.31 | 0.19 | 0.09 |
| liteio | 4.12 | 1.19 | 0.63 | 0.34 | 0.18 | 0.10 |
| minio | 2.66 | 0.75 | 0.39 | 0.22 | 0.11 | 0.05 |

*\* indicates errors occurred*

### Scale Performance

Performance with varying numbers of objects (256B each).

**Write N Files (total time)**

| Driver | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|
| herd_s3 | 2.5ms | 23.8ms | 229.2ms | 2.22s |
| liteio | 6.2ms | 66.1ms | 603.6ms | 6.96s |
| minio | 9.5ms | 97.4ms | 1.19s | 12.04s |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|
| herd_s3 | 376.8us | 973.6us | 5.3ms | 104.3ms |
| liteio | 326.7us | 835.9us | 6.0ms | 237.9ms |
| minio | 901.4us | 2.0ms | 12.8ms | 185.9ms |

*\* indicates errors occurred*

### Resource Usage Summary

| Driver | Memory | CPU |
|--------|--------|-----|
| herd_s3 | 1821.7 MB | 5.0% |
| liteio | 1273.9 MB | 7.2% |
| minio | 915.6 MB | 0.0% |

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

- **herd_s3** (48 benchmarks)
- **liteio** (48 benchmarks)
- **minio** (48 benchmarks)

## Detailed Results

### Copy/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 4.45 MB/s | 202.6us | 301.0us | 483.9us | 0 |
| liteio | 1.37 MB/s | 633.7us | 1.2ms | 1.9ms | 0 |
| minio | 0.51 MB/s | 1.6ms | 3.0ms | 10.4ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 4.45 MB/s
liteio       █████████ 1.37 MB/s
minio        ███ 0.51 MB/s
```

**Latency (P50)**
```
herd_s3      ███ 202.6us
liteio       ███████████ 633.7us
minio        ██████████████████████████████ 1.6ms
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 5097 ops/s | 184.7us | 265.8us | 374.9us | 0 |
| liteio | 4883 ops/s | 179.5us | 323.5us | 502.5us | 0 |
| minio | 2618 ops/s | 352.5us | 506.6us | 845.5us | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 5097 ops/s
liteio       ████████████████████████████ 4883 ops/s
minio        ███████████████ 2618 ops/s
```

**Latency (P50)**
```
herd_s3      ███████████████ 184.7us
liteio       ███████████████ 179.5us
minio        ██████████████████████████████ 352.5us
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 0.35 MB/s | 241.7us | 416.9us | 682.9us | 0 |
| liteio | 0.15 MB/s | 600.1us | 874.3us | 1.4ms | 0 |
| minio | 0.06 MB/s | 1.5ms | 2.2ms | 7.1ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 0.35 MB/s
liteio       ████████████ 0.15 MB/s
minio        ████ 0.06 MB/s
```

**Latency (P50)**
```
herd_s3      ████ 241.7us
liteio       ████████████ 600.1us
minio        ██████████████████████████████ 1.5ms
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 3457 ops/s | 264.6us | 426.0us | 712.9us | 0 |
| liteio | 1721 ops/s | 534.2us | 804.2us | 1.3ms | 0 |
| minio | 805 ops/s | 1.1ms | 1.7ms | 4.9ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 3457 ops/s
liteio       ██████████████ 1721 ops/s
minio        ██████ 805 ops/s
```

**Latency (P50)**
```
herd_s3      ███████ 264.6us
liteio       ██████████████ 534.2us
minio        ██████████████████████████████ 1.1ms
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 0.30 MB/s | 286.9us | 499.5us | 812.8us | 0 |
| liteio | 0.15 MB/s | 587.0us | 804.0us | 1.2ms | 0 |
| minio | 0.05 MB/s | 1.5ms | 2.5ms | 4.9ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 0.30 MB/s
liteio       ██████████████ 0.15 MB/s
minio        █████ 0.05 MB/s
```

**Latency (P50)**
```
herd_s3      █████ 286.9us
liteio       ███████████ 587.0us
minio        ██████████████████████████████ 1.5ms
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 1441 ops/s | 674.5us | 798.3us | 1.1ms | 0 |
| liteio | 979 ops/s | 943.5us | 1.7ms | 2.2ms | 0 |
| minio | 598 ops/s | 1.6ms | 2.0ms | 3.2ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 1441 ops/s
liteio       ████████████████████ 979 ops/s
minio        ████████████ 598 ops/s
```

**Latency (P50)**
```
herd_s3      ████████████ 674.5us
liteio       █████████████████ 943.5us
minio        ██████████████████████████████ 1.6ms
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.62 MB/s | 19.7ms | 40.3ms | 83.1ms | 0 |
| herd_s3 | 0.56 MB/s | 14.5ms | 40.4ms | 254.1ms | 0 |
| minio | 0.19 MB/s | 38.3ms | 453.6ms | 559.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.62 MB/s
herd_s3      ███████████████████████████ 0.56 MB/s
minio        █████████ 0.19 MB/s
```

**Latency (P50)**
```
liteio       ███████████████ 19.7ms
herd_s3      ███████████ 14.5ms
minio        ██████████████████████████████ 38.3ms
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 0.62 MB/s | 25.7ms | 34.2ms | 38.0ms | 0 |
| liteio | 0.55 MB/s | 23.7ms | 58.7ms | 98.2ms | 0 |
| minio | 0.38 MB/s | 22.6ms | 168.4ms | 394.5ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 0.62 MB/s
liteio       ██████████████████████████ 0.55 MB/s
minio        ██████████████████ 0.38 MB/s
```

**Latency (P50)**
```
herd_s3      ██████████████████████████████ 25.7ms
liteio       ███████████████████████████ 23.7ms
minio        ██████████████████████████ 22.6ms
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.64 MB/s | 21.8ms | 39.8ms | 47.1ms | 0 |
| herd_s3 | 0.48 MB/s | 16.1ms | 213.2ms | 421.1ms | 0 |
| minio | 0.10 MB/s | 107.9ms | 450.5ms | 883.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.64 MB/s
herd_s3      ██████████████████████ 0.48 MB/s
minio        ████ 0.10 MB/s
```

**Latency (P50)**
```
liteio       ██████ 21.8ms
herd_s3      ████ 16.1ms
minio        ██████████████████████████████ 107.9ms
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 135.05 MB/s | 104.0ms | 118.4ms | 118.4ms | 0 |
| herd_s3 | 103.47 MB/s | 142.2ms | 157.7ms | 157.7ms | 0 |
| minio | 76.88 MB/s | 196.6ms | 202.1ms | 202.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 135.05 MB/s
herd_s3      ██████████████████████ 103.47 MB/s
minio        █████████████████ 76.88 MB/s
```

**Latency (P50)**
```
liteio       ███████████████ 104.0ms
herd_s3      █████████████████████ 142.2ms
minio        ██████████████████████████████ 196.6ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 4.12 MB/s | 217.9us | 323.0us | 567.8us | 0 |
| herd_s3 | 3.80 MB/s | 237.9us | 355.2us | 591.4us | 0 |
| minio | 2.66 MB/s | 351.3us | 434.1us | 777.8us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4.12 MB/s
herd_s3      ███████████████████████████ 3.80 MB/s
minio        ███████████████████ 2.66 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████ 217.9us
herd_s3      ████████████████████ 237.9us
minio        ██████████████████████████████ 351.3us
```

### ParallelRead/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.19 MB/s | 779.4us | 1.2ms | 1.7ms | 0 |
| herd_s3 | 1.06 MB/s | 878.5us | 1.4ms | 2.0ms | 0 |
| minio | 0.75 MB/s | 1.2ms | 2.1ms | 3.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.19 MB/s
herd_s3      ██████████████████████████ 1.06 MB/s
minio        ██████████████████ 0.75 MB/s
```

**Latency (P50)**
```
liteio       ███████████████████ 779.4us
herd_s3      ██████████████████████ 878.5us
minio        ██████████████████████████████ 1.2ms
```

### ParallelRead/1KB/C100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 0.19 MB/s | 5.1ms | 8.0ms | 11.3ms | 0 |
| liteio | 0.18 MB/s | 5.3ms | 9.1ms | 13.1ms | 0 |
| minio | 0.11 MB/s | 8.1ms | 19.8ms | 29.4ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 0.19 MB/s
liteio       ████████████████████████████ 0.18 MB/s
minio        ████████████████ 0.11 MB/s
```

**Latency (P50)**
```
herd_s3      ██████████████████ 5.1ms
liteio       ███████████████████ 5.3ms
minio        ██████████████████████████████ 8.1ms
```

### ParallelRead/1KB/C200

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.10 MB/s | 10.2ms | 14.8ms | 20.0ms | 0 |
| herd_s3 | 0.09 MB/s | 11.3ms | 18.2ms | 23.8ms | 0 |
| minio | 0.05 MB/s | 16.2ms | 41.3ms | 65.8ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.10 MB/s
herd_s3      ██████████████████████████ 0.09 MB/s
minio        ████████████████ 0.05 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████ 10.2ms
herd_s3      ████████████████████ 11.3ms
minio        ██████████████████████████████ 16.2ms
```

### ParallelRead/1KB/C25

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.63 MB/s | 1.4ms | 2.5ms | 4.4ms | 0 |
| herd_s3 | 0.56 MB/s | 1.7ms | 2.8ms | 3.9ms | 0 |
| minio | 0.39 MB/s | 2.4ms | 4.2ms | 5.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.63 MB/s
herd_s3      ██████████████████████████ 0.56 MB/s
minio        ██████████████████ 0.39 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████ 1.4ms
herd_s3      █████████████████████ 1.7ms
minio        ██████████████████████████████ 2.4ms
```

### ParallelRead/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.34 MB/s | 2.7ms | 4.6ms | 6.9ms | 0 |
| herd_s3 | 0.31 MB/s | 3.0ms | 5.2ms | 7.7ms | 0 |
| minio | 0.22 MB/s | 4.1ms | 8.2ms | 11.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.34 MB/s
herd_s3      ██████████████████████████ 0.31 MB/s
minio        ██████████████████ 0.22 MB/s
```

**Latency (P50)**
```
liteio       ███████████████████ 2.7ms
herd_s3      █████████████████████ 3.0ms
minio        ██████████████████████████████ 4.1ms
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 3.42 MB/s | 265.2us | 390.1us | 712.2us | 0 |
| liteio | 1.40 MB/s | 651.8us | 902.6us | 1.5ms | 0 |
| minio | 0.62 MB/s | 1.6ms | 2.1ms | 2.7ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 3.42 MB/s
liteio       ████████████ 1.40 MB/s
minio        █████ 0.62 MB/s
```

**Latency (P50)**
```
herd_s3      █████ 265.2us
liteio       ████████████ 651.8us
minio        ██████████████████████████████ 1.6ms
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 0.99 MB/s | 923.8us | 1.6ms | 2.4ms | 0 |
| liteio | 0.49 MB/s | 1.9ms | 2.6ms | 4.1ms | 0 |
| minio | 0.17 MB/s | 5.5ms | 7.6ms | 13.2ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 0.99 MB/s
liteio       ██████████████ 0.49 MB/s
minio        █████ 0.17 MB/s
```

**Latency (P50)**
```
herd_s3      █████ 923.8us
liteio       ██████████ 1.9ms
minio        ██████████████████████████████ 5.5ms
```

### ParallelWrite/1KB/C100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 0.14 MB/s | 7.0ms | 10.5ms | 14.0ms | 0 |
| liteio | 0.11 MB/s | 8.8ms | 12.9ms | 16.3ms | 0 |
| minio | 0.01 MB/s | 57.8ms | 132.8ms | 154.7ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 0.14 MB/s
liteio       ███████████████████████ 0.11 MB/s
minio        ███ 0.01 MB/s
```

**Latency (P50)**
```
herd_s3      ███ 7.0ms
liteio       ████ 8.8ms
minio        ██████████████████████████████ 57.8ms
```

### ParallelWrite/1KB/C200

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 0.07 MB/s | 13.8ms | 20.5ms | 27.4ms | 0 |
| liteio | 0.05 MB/s | 16.9ms | 40.6ms | 63.1ms | 0 |
| minio | 0.01 MB/s | 122.2ms | 292.9ms | 385.3ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 0.07 MB/s
liteio       █████████████████████ 0.05 MB/s
minio        ██ 0.01 MB/s
```

**Latency (P50)**
```
herd_s3      ███ 13.8ms
liteio       ████ 16.9ms
minio        ██████████████████████████████ 122.2ms
```

### ParallelWrite/1KB/C25

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 0.53 MB/s | 1.8ms | 2.7ms | 4.2ms | 0 |
| liteio | 0.28 MB/s | 3.4ms | 4.8ms | 6.5ms | 0 |
| minio | 0.08 MB/s | 11.7ms | 18.8ms | 24.2ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 0.53 MB/s
liteio       ███████████████ 0.28 MB/s
minio        ████ 0.08 MB/s
```

**Latency (P50)**
```
herd_s3      ████ 1.8ms
liteio       ████████ 3.4ms
minio        ██████████████████████████████ 11.7ms
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 0.25 MB/s | 3.7ms | 6.7ms | 10.8ms | 0 |
| liteio | 0.13 MB/s | 6.2ms | 15.3ms | 29.2ms | 0 |
| minio | 0.03 MB/s | 25.0ms | 59.0ms | 74.4ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 0.25 MB/s
liteio       ███████████████ 0.13 MB/s
minio        ████ 0.03 MB/s
```

**Latency (P50)**
```
herd_s3      ████ 3.7ms
liteio       ███████ 6.2ms
minio        ██████████████████████████████ 25.0ms
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 154.47 MB/s | 1.5ms | 2.3ms | 3.4ms | 0 |
| herd_s3 | 151.46 MB/s | 1.6ms | 2.2ms | 2.9ms | 0 |
| minio | 102.02 MB/s | 2.3ms | 3.5ms | 4.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 154.47 MB/s
herd_s3      █████████████████████████████ 151.46 MB/s
minio        ███████████████████ 102.02 MB/s
```

**Latency (P50)**
```
liteio       ███████████████████ 1.5ms
herd_s3      ████████████████████ 1.6ms
minio        ██████████████████████████████ 2.3ms
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 149.24 MB/s | 1.5ms | 2.6ms | 3.9ms | 0 |
| herd_s3 | 148.55 MB/s | 1.6ms | 2.4ms | 3.3ms | 0 |
| minio | 90.09 MB/s | 2.6ms | 3.8ms | 4.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 149.24 MB/s
herd_s3      █████████████████████████████ 148.55 MB/s
minio        ██████████████████ 90.09 MB/s
```

**Latency (P50)**
```
liteio       █████████████████ 1.5ms
herd_s3      █████████████████ 1.6ms
minio        ██████████████████████████████ 2.6ms
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 141.98 MB/s | 1.6ms | 2.7ms | 3.3ms | 0 |
| herd_s3 | 134.52 MB/s | 1.8ms | 2.5ms | 3.2ms | 0 |
| minio | 75.42 MB/s | 3.1ms | 5.1ms | 6.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 141.98 MB/s
herd_s3      ████████████████████████████ 134.52 MB/s
minio        ███████████████ 75.42 MB/s
```

**Latency (P50)**
```
liteio       ███████████████ 1.6ms
herd_s3      ████████████████ 1.8ms
minio        ██████████████████████████████ 3.1ms
```

### Read/100MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 183.03 MB/s | 522.4ms | 522.4ms | 522.4ms | 0 |
| minio | 170.53 MB/s | 571.3ms | 571.3ms | 571.3ms | 0 |
| herd_s3 | 169.37 MB/s | 568.6ms | 568.6ms | 568.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 183.03 MB/s
minio        ███████████████████████████ 170.53 MB/s
herd_s3      ███████████████████████████ 169.37 MB/s
```

**Latency (P50)**
```
liteio       ███████████████████████████ 522.4ms
minio        ██████████████████████████████ 571.3ms
herd_s3      █████████████████████████████ 568.6ms
```

### Read/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 203.26 MB/s | 48.7ms | 51.3ms | 51.8ms | 0 |
| herd_s3 | 183.14 MB/s | 53.7ms | 58.9ms | 62.9ms | 0 |
| minio | 160.66 MB/s | 57.5ms | 83.1ms | 86.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 203.26 MB/s
herd_s3      ███████████████████████████ 183.14 MB/s
minio        ███████████████████████ 160.66 MB/s
```

**Latency (P50)**
```
liteio       █████████████████████████ 48.7ms
herd_s3      ████████████████████████████ 53.7ms
minio        ██████████████████████████████ 57.5ms
```

### Read/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 4.53 MB/s | 193.7us | 338.5us | 556.3us | 0 |
| herd_s3 | 3.52 MB/s | 199.5us | 615.2us | 1.5ms | 0 |
| minio | 2.65 MB/s | 347.8us | 469.8us | 683.3us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4.53 MB/s
herd_s3      ███████████████████████ 3.52 MB/s
minio        █████████████████ 2.65 MB/s
```

**Latency (P50)**
```
liteio       ████████████████ 193.7us
herd_s3      █████████████████ 199.5us
minio        ██████████████████████████████ 347.8us
```

### Read/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 177.14 MB/s | 5.3ms | 7.8ms | 11.2ms | 0 |
| herd_s3 | 168.07 MB/s | 5.7ms | 7.6ms | 9.2ms | 0 |
| minio | 157.67 MB/s | 6.0ms | 8.5ms | 12.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 177.14 MB/s
herd_s3      ████████████████████████████ 168.07 MB/s
minio        ██████████████████████████ 157.67 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████████ 5.3ms
herd_s3      ████████████████████████████ 5.7ms
minio        ██████████████████████████████ 6.0ms
```

### Read/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 117.20 MB/s | 498.0us | 769.5us | 1.0ms | 0 |
| herd_s3 | 101.81 MB/s | 571.3us | 898.5us | 1.2ms | 0 |
| minio | 95.33 MB/s | 624.1us | 812.2us | 1.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 117.20 MB/s
herd_s3      ██████████████████████████ 101.81 MB/s
minio        ████████████████████████ 95.33 MB/s
```

**Latency (P50)**
```
liteio       ███████████████████████ 498.0us
herd_s3      ███████████████████████████ 571.3us
minio        ██████████████████████████████ 624.1us
```

### Scale/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 517 ops/s | 1.9ms | 1.9ms | 1.9ms | 0 |
| herd_s3 | 456 ops/s | 2.2ms | 2.2ms | 2.2ms | 0 |
| minio | 260 ops/s | 3.8ms | 3.8ms | 3.8ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 517 ops/s
herd_s3      ██████████████████████████ 456 ops/s
minio        ███████████████ 260 ops/s
```

**Latency (P50)**
```
liteio       ███████████████ 1.9ms
herd_s3      █████████████████ 2.2ms
minio        ██████████████████████████████ 3.8ms
```

### Scale/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 52 ops/s | 19.4ms | 19.4ms | 19.4ms | 0 |
| herd_s3 | 46 ops/s | 21.6ms | 21.6ms | 21.6ms | 0 |
| minio | 27 ops/s | 36.7ms | 36.7ms | 36.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 52 ops/s
herd_s3      ██████████████████████████ 46 ops/s
minio        ███████████████ 27 ops/s
```

**Latency (P50)**
```
liteio       ███████████████ 19.4ms
herd_s3      █████████████████ 21.6ms
minio        ██████████████████████████████ 36.7ms
```

### Scale/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 5 ops/s | 199.9ms | 199.9ms | 199.9ms | 0 |
| liteio | 5 ops/s | 209.9ms | 209.9ms | 209.9ms | 0 |
| minio | 3 ops/s | 383.0ms | 383.0ms | 383.0ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 5 ops/s
liteio       ████████████████████████████ 5 ops/s
minio        ███████████████ 3 ops/s
```

**Latency (P50)**
```
herd_s3      ███████████████ 199.9ms
liteio       ████████████████ 209.9ms
minio        ██████████████████████████████ 383.0ms
```

### Scale/Delete/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0 ops/s | 2.04s | 2.04s | 2.04s | 0 |
| herd_s3 | 0 ops/s | 2.40s | 2.40s | 2.40s | 0 |
| minio | 0 ops/s | 3.77s | 3.77s | 3.77s | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0 ops/s
herd_s3      █████████████████████████ 0 ops/s
minio        ████████████████ 0 ops/s
```

**Latency (P50)**
```
liteio       ████████████████ 2.04s
herd_s3      ███████████████████ 2.40s
minio        ██████████████████████████████ 3.77s
```

### Scale/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 3061 ops/s | 326.7us | 326.7us | 326.7us | 0 |
| herd_s3 | 2654 ops/s | 376.8us | 376.8us | 376.8us | 0 |
| minio | 1109 ops/s | 901.4us | 901.4us | 901.4us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3061 ops/s
herd_s3      ██████████████████████████ 2654 ops/s
minio        ██████████ 1109 ops/s
```

**Latency (P50)**
```
liteio       ██████████ 326.7us
herd_s3      ████████████ 376.8us
minio        ██████████████████████████████ 901.4us
```

### Scale/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1196 ops/s | 835.9us | 835.9us | 835.9us | 0 |
| herd_s3 | 1027 ops/s | 973.6us | 973.6us | 973.6us | 0 |
| minio | 496 ops/s | 2.0ms | 2.0ms | 2.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1196 ops/s
herd_s3      █████████████████████████ 1027 ops/s
minio        ████████████ 496 ops/s
```

**Latency (P50)**
```
liteio       ████████████ 835.9us
herd_s3      ██████████████ 973.6us
minio        ██████████████████████████████ 2.0ms
```

### Scale/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 188 ops/s | 5.3ms | 5.3ms | 5.3ms | 0 |
| liteio | 166 ops/s | 6.0ms | 6.0ms | 6.0ms | 0 |
| minio | 78 ops/s | 12.8ms | 12.8ms | 12.8ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 188 ops/s
liteio       ██████████████████████████ 166 ops/s
minio        ████████████ 78 ops/s
```

**Latency (P50)**
```
herd_s3      ████████████ 5.3ms
liteio       ██████████████ 6.0ms
minio        ██████████████████████████████ 12.8ms
```

### Scale/List/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 10 ops/s | 104.3ms | 104.3ms | 104.3ms | 0 |
| minio | 5 ops/s | 185.9ms | 185.9ms | 185.9ms | 0 |
| liteio | 4 ops/s | 237.9ms | 237.9ms | 237.9ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 10 ops/s
minio        ████████████████ 5 ops/s
liteio       █████████████ 4 ops/s
```

**Latency (P50)**
```
herd_s3      █████████████ 104.3ms
minio        ███████████████████████ 185.9ms
liteio       ██████████████████████████████ 237.9ms
```

### Scale/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 0.96 MB/s | 2.5ms | 2.5ms | 2.5ms | 0 |
| liteio | 0.39 MB/s | 6.2ms | 6.2ms | 6.2ms | 0 |
| minio | 0.26 MB/s | 9.5ms | 9.5ms | 9.5ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 0.96 MB/s
liteio       ████████████ 0.39 MB/s
minio        ███████ 0.26 MB/s
```

**Latency (P50)**
```
herd_s3      ███████ 2.5ms
liteio       ███████████████████ 6.2ms
minio        ██████████████████████████████ 9.5ms
```

### Scale/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 1.02 MB/s | 23.8ms | 23.8ms | 23.8ms | 0 |
| liteio | 0.37 MB/s | 66.1ms | 66.1ms | 66.1ms | 0 |
| minio | 0.25 MB/s | 97.4ms | 97.4ms | 97.4ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 1.02 MB/s
liteio       ██████████ 0.37 MB/s
minio        ███████ 0.25 MB/s
```

**Latency (P50)**
```
herd_s3      ███████ 23.8ms
liteio       ████████████████████ 66.1ms
minio        ██████████████████████████████ 97.4ms
```

### Scale/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 1.07 MB/s | 229.2ms | 229.2ms | 229.2ms | 0 |
| liteio | 0.40 MB/s | 603.6ms | 603.6ms | 603.6ms | 0 |
| minio | 0.21 MB/s | 1.19s | 1.19s | 1.19s | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 1.07 MB/s
liteio       ███████████ 0.40 MB/s
minio        █████ 0.21 MB/s
```

**Latency (P50)**
```
herd_s3      █████ 229.2ms
liteio       ███████████████ 603.6ms
minio        ██████████████████████████████ 1.19s
```

### Scale/Write/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 1.10 MB/s | 2.22s | 2.22s | 2.22s | 0 |
| liteio | 0.35 MB/s | 6.96s | 6.96s | 6.96s | 0 |
| minio | 0.20 MB/s | 12.04s | 12.04s | 12.04s | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 1.10 MB/s
liteio       █████████ 0.35 MB/s
minio        █████ 0.20 MB/s
```

**Latency (P50)**
```
herd_s3      █████ 2.22s
liteio       █████████████████ 6.96s
minio        ██████████████████████████████ 12.04s
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 4520 ops/s | 202.6us | 324.1us | 504.7us | 0 |
| liteio | 3658 ops/s | 240.8us | 452.3us | 743.5us | 0 |
| minio | 3202 ops/s | 297.3us | 407.8us | 554.4us | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 4520 ops/s
liteio       ████████████████████████ 3658 ops/s
minio        █████████████████████ 3202 ops/s
```

**Latency (P50)**
```
herd_s3      ████████████████████ 202.6us
liteio       ████████████████████████ 240.8us
minio        ██████████████████████████████ 297.3us
```

### Write/100MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 172.23 MB/s | 560.8ms | 560.8ms | 560.8ms | 0 |
| liteio | 125.22 MB/s | 780.0ms | 780.0ms | 780.0ms | 0 |
| minio | 101.39 MB/s | 1.01s | 1.01s | 1.01s | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 172.23 MB/s
liteio       █████████████████████ 125.22 MB/s
minio        █████████████████ 101.39 MB/s
```

**Latency (P50)**
```
herd_s3      ████████████████ 560.8ms
liteio       ███████████████████████ 780.0ms
minio        ██████████████████████████████ 1.01s
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 161.63 MB/s | 56.1ms | 85.3ms | 90.0ms | 0 |
| minio | 153.62 MB/s | 65.0ms | 75.2ms | 78.0ms | 0 |
| liteio | 138.25 MB/s | 71.9ms | 81.0ms | 81.3ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 161.63 MB/s
minio        ████████████████████████████ 153.62 MB/s
liteio       █████████████████████████ 138.25 MB/s
```

**Latency (P50)**
```
herd_s3      ███████████████████████ 56.1ms
minio        ███████████████████████████ 65.0ms
liteio       ██████████████████████████████ 71.9ms
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 4.44 MB/s | 189.6us | 369.8us | 705.3us | 0 |
| liteio | 1.48 MB/s | 604.8us | 846.9us | 1.4ms | 0 |
| minio | 0.62 MB/s | 1.4ms | 2.3ms | 3.6ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 4.44 MB/s
liteio       ██████████ 1.48 MB/s
minio        ████ 0.62 MB/s
```

**Latency (P50)**
```
herd_s3      ███ 189.6us
liteio       ████████████ 604.8us
minio        ██████████████████████████████ 1.4ms
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 189.08 MB/s | 4.9ms | 7.2ms | 9.6ms | 0 |
| liteio | 128.49 MB/s | 7.0ms | 13.2ms | 17.1ms | 0 |
| minio | 84.36 MB/s | 9.3ms | 22.4ms | 32.1ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 189.08 MB/s
liteio       ████████████████████ 128.49 MB/s
minio        █████████████ 84.36 MB/s
```

**Latency (P50)**
```
herd_s3      ███████████████ 4.9ms
liteio       ██████████████████████ 7.0ms
minio        ██████████████████████████████ 9.3ms
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_s3 | 129.14 MB/s | 450.1us | 663.3us | 1.2ms | 0 |
| liteio | 65.55 MB/s | 897.4us | 1.2ms | 1.7ms | 0 |
| minio | 26.36 MB/s | 2.2ms | 3.0ms | 4.5ms | 0 |

**Throughput**
```
herd_s3      ██████████████████████████████ 129.14 MB/s
liteio       ███████████████ 65.55 MB/s
minio        ██████ 26.36 MB/s
```

**Latency (P50)**
```
herd_s3      ██████ 450.1us
liteio       ███████████ 897.4us
minio        ██████████████████████████████ 2.2ms
```

## Resource Usage

| Driver | Memory | RSS | Cache | CPU | Volume | Block I/O |
|--------|--------|-----|-------|-----|--------|----------|
| herd_s3 | 1.778GiB / 7.653GiB | 1820.7 MB | - | 5.0% | 2437.0 MB | 1.34MB / 419MB |
| liteio | 1.244GiB / 7.653GiB | 1273.9 MB | - | 7.2% | 3799.0 MB | 49.2kB / 2.17GB |
| minio | 915.6MiB / 7.653GiB | 915.6 MB | - | 0.0% | 3124.0 MB | 403MB / 2.07GB |

> **Note:** RSS = actual application memory. Cache = OS page cache (reclaimable).

## Recommendations

- **Write-heavy workloads:** herd_s3
- **Read-heavy workloads:** liteio

---

*Generated by storage benchmark CLI*
