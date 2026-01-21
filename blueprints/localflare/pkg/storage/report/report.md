# Storage Benchmark Report

**Generated:** 2026-01-21T17:57:03+07:00

**Go Version:** go1.25.5

**Platform:** darwin/arm64

## Executive Summary

### Summary

**Overall Winner:** devnull_s3 (won 25/48 benchmarks, 52%)

| Rank | Driver | Wins | Win Rate |
|------|--------|------|----------|
| 1 | devnull_s3 | 25 | 52% |
| 2 | usagi_s3 | 19 | 40% |
| 3 | minio | 4 | 8% |

### Performance Leaders

| Operation | Leader | Performance | Margin |
|-----------|--------|-------------|--------|
| Small Read (1KB) | usagi_s3 | 4.3 MB/s | close |
| Small Write (1KB) | usagi_s3 | 1.7 MB/s | close |
| Large Read (100MB) | minio | 242.3 MB/s | +14% vs usagi_s3 |
| Large Write (100MB) | usagi_s3 | 158.9 MB/s | close |
| Delete | devnull_s3 | 4.2K ops/s | close |
| Stat | usagi_s3 | 4.5K ops/s | close |
| List (100 objects) | devnull_s3 | 960 ops/s | close |
| Copy | devnull_s3 | 1.2 MB/s | +33% vs usagi_s3 |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (100MB+) | **usagi_s3** | 159 MB/s | Best for media, backups |
| Large File Downloads (100MB) | **minio** | 242 MB/s | Best for streaming, CDN |
| Small File Operations | **usagi_s3** | 3097 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **usagi_s3** | - | Best for multi-user apps |
| Memory Constrained | **minio** | 887 MB RAM | Best for edge/embedded |

### Large File Performance (100MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| devnull_s3 | 149.0 | 183.5 | 678.9ms | 550.8ms |
| minio | 152.6 | 242.3 | 660.8ms | 410.5ms |
| usagi_s3 | 158.9 | 212.0 | 637.3ms | 471.2ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| devnull_s3 | 1712 | 4360 | 540.0us | 208.4us |
| minio | 1262 | 2617 | 719.8us | 351.1us |
| usagi_s3 | 1761 | 4433 | 521.5us | 204.0us |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| devnull_s3 | 4331 | 960 | 4230 |
| minio | 3172 | 505 | 2246 |
| usagi_s3 | 4541 | 947 | 3971 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| devnull_s3 | 1.38 | 0.38 | 0.10 | 0.07 | 0.05 | 0.02 |
| minio | 1.06 | 0.24 | 0.12 | 0.05 | 0.02 | 0.01 |
| usagi_s3 | 1.42 | 0.40 | 0.13 | 0.08 | 0.04 | 0.01 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| devnull_s3 | 3.50 | 1.11 | 0.59 | 0.35 | 0.17 | 0.10 |
| minio | 2.46 | 0.81 | 0.38 | 0.19 | 0.12 | 0.06 |
| usagi_s3 | 3.53 | 1.13 | 0.64 | 0.35 | 0.15 | 0.06 |

*\* indicates errors occurred*

### Scale Performance

Performance with varying numbers of objects (256B each).

**Write N Files (total time)**

| Driver | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|
| devnull_s3 | 5.8ms | 68.7ms | 630.5ms | 6.21s |
| minio | 11.0ms | 98.0ms | 959.6ms | 10.78s |
| usagi_s3 | 6.1ms | 67.5ms | 789.1ms | 6.87s |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|
| devnull_s3 | 476.0us | 1.1ms | 6.9ms | 233.7ms |
| minio | 1.1ms | 2.3ms | 15.3ms | 174.6ms |
| usagi_s3 | 605.2us | 1.2ms | 8.8ms | 288.7ms |

*\* indicates errors occurred*

### Resource Usage Summary

| Driver | Memory | CPU |
|--------|--------|-----|
| devnull_s3 | 1541.1 MB | 3.6% |
| minio | 886.8 MB | 0.0% |
| usagi_s3 | 1587.2 MB | 3.2% |

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
- **minio** (48 benchmarks)
- **usagi_s3** (48 benchmarks)

## Detailed Results

### Copy/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 1.22 MB/s | 643.7us | 1.6ms | 2.1ms | 0 |
| usagi_s3 | 0.92 MB/s | 816.4us | 2.3ms | 4.1ms | 0 |
| minio | 0.88 MB/s | 992.4us | 1.8ms | 2.4ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 1.22 MB/s
usagi_s3     ██████████████████████ 0.92 MB/s
minio        █████████████████████ 0.88 MB/s
```

**Latency (P50)**
```
devnull_s3   ███████████████████ 643.7us
usagi_s3     ████████████████████████ 816.4us
minio        ██████████████████████████████ 992.4us
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 4230 ops/s | 214.6us | 362.2us | 597.1us | 0 |
| usagi_s3 | 3971 ops/s | 221.4us | 436.9us | 708.2us | 0 |
| minio | 2246 ops/s | 412.6us | 624.0us | 997.9us | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 4230 ops/s
usagi_s3     ████████████████████████████ 3971 ops/s
minio        ███████████████ 2246 ops/s
```

**Latency (P50)**
```
devnull_s3   ███████████████ 214.6us
usagi_s3     ████████████████ 221.4us
minio        ██████████████████████████████ 412.6us
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 0.14 MB/s | 647.8us | 1.0ms | 1.5ms | 0 |
| usagi_s3 | 0.14 MB/s | 638.5us | 1.2ms | 1.7ms | 0 |
| minio | 0.09 MB/s | 1.0ms | 1.5ms | 2.2ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 0.14 MB/s
usagi_s3     █████████████████████████████ 0.14 MB/s
minio        ███████████████████ 0.09 MB/s
```

**Latency (P50)**
```
devnull_s3   ███████████████████ 647.8us
usagi_s3     ███████████████████ 638.5us
minio        ██████████████████████████████ 1.0ms
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 1573 ops/s | 591.7us | 996.3us | 1.4ms | 0 |
| devnull_s3 | 1414 ops/s | 647.3us | 1.2ms | 1.7ms | 0 |
| minio | 924 ops/s | 940.0us | 1.9ms | 2.8ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 1573 ops/s
devnull_s3   ██████████████████████████ 1414 ops/s
minio        █████████████████ 924 ops/s
```

**Latency (P50)**
```
usagi_s3     ██████████████████ 591.7us
devnull_s3   ████████████████████ 647.3us
minio        ██████████████████████████████ 940.0us
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 0.13 MB/s | 643.4us | 1.1ms | 1.5ms | 0 |
| usagi_s3 | 0.13 MB/s | 615.9us | 1.3ms | 1.9ms | 0 |
| minio | 0.08 MB/s | 1.0ms | 2.1ms | 3.2ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 0.13 MB/s
usagi_s3     █████████████████████████████ 0.13 MB/s
minio        █████████████████ 0.08 MB/s
```

**Latency (P50)**
```
devnull_s3   ██████████████████ 643.4us
usagi_s3     █████████████████ 615.9us
minio        ██████████████████████████████ 1.0ms
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 960 ops/s | 980.2us | 1.4ms | 1.7ms | 0 |
| usagi_s3 | 947 ops/s | 1.0ms | 1.5ms | 1.7ms | 0 |
| minio | 505 ops/s | 1.9ms | 2.6ms | 4.5ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 960 ops/s
usagi_s3     █████████████████████████████ 947 ops/s
minio        ███████████████ 505 ops/s
```

**Latency (P50)**
```
devnull_s3   ███████████████ 980.2us
usagi_s3     ████████████████ 1.0ms
minio        ██████████████████████████████ 1.9ms
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 0.38 MB/s | 14.8ms | 112.8ms | 124.3ms | 0 |
| minio | 0.28 MB/s | 29.8ms | 155.1ms | 227.7ms | 0 |
| usagi_s3 | 0.26 MB/s | 14.3ms | 206.9ms | 236.6ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 0.38 MB/s
minio        ██████████████████████ 0.28 MB/s
usagi_s3     █████████████████████ 0.26 MB/s
```

**Latency (P50)**
```
devnull_s3   ██████████████ 14.8ms
minio        ██████████████████████████████ 29.8ms
usagi_s3     ██████████████ 14.3ms
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 0.64 MB/s | 23.1ms | 38.6ms | 58.9ms | 0 |
| usagi_s3 | 0.53 MB/s | 24.3ms | 72.7ms | 107.7ms | 0 |
| minio | 0.46 MB/s | 26.2ms | 85.1ms | 168.6ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 0.64 MB/s
usagi_s3     ████████████████████████ 0.53 MB/s
minio        █████████████████████ 0.46 MB/s
```

**Latency (P50)**
```
devnull_s3   ██████████████████████████ 23.1ms
usagi_s3     ███████████████████████████ 24.3ms
minio        ██████████████████████████████ 26.2ms
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 0.21 MB/s | 88.6ms | 125.5ms | 134.4ms | 0 |
| usagi_s3 | 0.21 MB/s | 89.3ms | 132.9ms | 141.5ms | 0 |
| minio | 0.20 MB/s | 76.9ms | 152.5ms | 209.9ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 0.21 MB/s
usagi_s3     █████████████████████████████ 0.21 MB/s
minio        ████████████████████████████ 0.20 MB/s
```

**Latency (P50)**
```
devnull_s3   █████████████████████████████ 88.6ms
usagi_s3     ██████████████████████████████ 89.3ms
minio        █████████████████████████ 76.9ms
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 132.99 MB/s | 113.6ms | 118.1ms | 118.1ms | 0 |
| usagi_s3 | 109.12 MB/s | 137.4ms | 144.0ms | 144.0ms | 0 |
| devnull_s3 | 107.84 MB/s | 138.5ms | 143.2ms | 143.2ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 132.99 MB/s
usagi_s3     ████████████████████████ 109.12 MB/s
devnull_s3   ████████████████████████ 107.84 MB/s
```

**Latency (P50)**
```
minio        ████████████████████████ 113.6ms
usagi_s3     █████████████████████████████ 137.4ms
devnull_s3   ██████████████████████████████ 138.5ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| usagi_s3 | 3.53 MB/s | 276.4us | 424.3us | 252.5us | 424.7us | 671.9us | 0 |
| devnull_s3 | 3.50 MB/s | 279.0us | 430.9us | 256.7us | 431.1us | 642.4us | 0 |
| minio | 2.46 MB/s | 396.2us | 521.4us | 369.4us | 521.5us | 926.6us | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 3.53 MB/s
devnull_s3   █████████████████████████████ 3.50 MB/s
minio        ████████████████████ 2.46 MB/s
```

**Latency (P50)**
```
usagi_s3     ████████████████████ 252.5us
devnull_s3   ████████████████████ 256.7us
minio        ██████████████████████████████ 369.4us
```

### ParallelRead/1KB/C10

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| usagi_s3 | 1.13 MB/s | 864.3us | 1.4ms | 796.9us | 1.4ms | 1.9ms | 0 |
| devnull_s3 | 1.11 MB/s | 883.4us | 1.4ms | 829.3us | 1.4ms | 1.8ms | 0 |
| minio | 0.81 MB/s | 1.2ms | 1.9ms | 1.1ms | 1.9ms | 3.1ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 1.13 MB/s
devnull_s3   █████████████████████████████ 1.11 MB/s
minio        █████████████████████ 0.81 MB/s
```

**Latency (P50)**
```
usagi_s3     █████████████████████ 796.9us
devnull_s3   ██████████████████████ 829.3us
minio        ██████████████████████████████ 1.1ms
```

### ParallelRead/1KB/C100

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| devnull_s3 | 0.17 MB/s | 5.9ms | 9.4ms | 5.7ms | 9.4ms | 11.8ms | 0 |
| usagi_s3 | 0.15 MB/s | 6.6ms | 12.8ms | 6.0ms | 12.8ms | 18.4ms | 0 |
| minio | 0.12 MB/s | 8.3ms | 16.4ms | 7.6ms | 16.4ms | 21.9ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 0.17 MB/s
usagi_s3     ██████████████████████████ 0.15 MB/s
minio        █████████████████████ 0.12 MB/s
```

**Latency (P50)**
```
devnull_s3   ██████████████████████ 5.7ms
usagi_s3     ███████████████████████ 6.0ms
minio        ██████████████████████████████ 7.6ms
```

### ParallelRead/1KB/C200

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| devnull_s3 | 0.10 MB/s | 9.9ms | 15.8ms | 10.3ms | 15.8ms | 23.8ms | 0 |
| usagi_s3 | 0.06 MB/s | 15.4ms | 26.2ms | 14.9ms | 26.2ms | 33.4ms | 0 |
| minio | 0.06 MB/s | 17.1ms | 35.4ms | 15.5ms | 35.4ms | 49.5ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 0.10 MB/s
usagi_s3     ███████████████████ 0.06 MB/s
minio        █████████████████ 0.06 MB/s
```

**Latency (P50)**
```
devnull_s3   ███████████████████ 10.3ms
usagi_s3     ████████████████████████████ 14.9ms
minio        ██████████████████████████████ 15.5ms
```

### ParallelRead/1KB/C25

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| usagi_s3 | 0.64 MB/s | 1.5ms | 2.7ms | 1.4ms | 2.7ms | 3.6ms | 0 |
| devnull_s3 | 0.59 MB/s | 1.6ms | 2.7ms | 1.5ms | 2.7ms | 3.6ms | 0 |
| minio | 0.38 MB/s | 2.6ms | 4.8ms | 2.3ms | 4.8ms | 7.0ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.64 MB/s
devnull_s3   ███████████████████████████ 0.59 MB/s
minio        █████████████████ 0.38 MB/s
```

**Latency (P50)**
```
usagi_s3     ██████████████████ 1.4ms
devnull_s3   ████████████████████ 1.5ms
minio        ██████████████████████████████ 2.3ms
```

### ParallelRead/1KB/C50

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| usagi_s3 | 0.35 MB/s | 2.8ms | 4.5ms | 2.7ms | 4.5ms | 6.3ms | 0 |
| devnull_s3 | 0.35 MB/s | 2.8ms | 5.1ms | 2.6ms | 5.1ms | 6.7ms | 0 |
| minio | 0.19 MB/s | 5.0ms | 10.3ms | 4.3ms | 10.3ms | 18.9ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.35 MB/s
devnull_s3   █████████████████████████████ 0.35 MB/s
minio        ████████████████ 0.19 MB/s
```

**Latency (P50)**
```
usagi_s3     ██████████████████ 2.7ms
devnull_s3   ██████████████████ 2.6ms
minio        ██████████████████████████████ 4.3ms
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 1.42 MB/s | 633.0us | 959.5us | 1.2ms | 0 |
| devnull_s3 | 1.38 MB/s | 619.8us | 1.1ms | 1.7ms | 0 |
| minio | 1.06 MB/s | 834.5us | 1.4ms | 2.5ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 1.42 MB/s
devnull_s3   █████████████████████████████ 1.38 MB/s
minio        ██████████████████████ 1.06 MB/s
```

**Latency (P50)**
```
usagi_s3     ██████████████████████ 633.0us
devnull_s3   ██████████████████████ 619.8us
minio        ██████████████████████████████ 834.5us
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.40 MB/s | 2.3ms | 4.0ms | 6.3ms | 0 |
| devnull_s3 | 0.38 MB/s | 2.4ms | 4.0ms | 6.8ms | 0 |
| minio | 0.24 MB/s | 3.8ms | 6.6ms | 8.8ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.40 MB/s
devnull_s3   ████████████████████████████ 0.38 MB/s
minio        ██████████████████ 0.24 MB/s
```

**Latency (P50)**
```
usagi_s3     ██████████████████ 2.3ms
devnull_s3   ██████████████████ 2.4ms
minio        ██████████████████████████████ 3.8ms
```

### ParallelWrite/1KB/C100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 0.05 MB/s | 16.5ms | 35.7ms | 47.4ms | 0 |
| usagi_s3 | 0.04 MB/s | 28.9ms | 37.5ms | 43.1ms | 0 |
| minio | 0.02 MB/s | 40.2ms | 91.8ms | 127.4ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 0.05 MB/s
usagi_s3     █████████████████████ 0.04 MB/s
minio        ████████████ 0.02 MB/s
```

**Latency (P50)**
```
devnull_s3   ████████████ 16.5ms
usagi_s3     █████████████████████ 28.9ms
minio        ██████████████████████████████ 40.2ms
```

### ParallelWrite/1KB/C200

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 0.02 MB/s | 45.5ms | 74.9ms | 92.8ms | 0 |
| usagi_s3 | 0.01 MB/s | 80.3ms | 145.1ms | 195.5ms | 0 |
| minio | 0.01 MB/s | 68.5ms | 145.6ms | 207.2ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 0.02 MB/s
usagi_s3     █████████████████ 0.01 MB/s
minio        ████████████████ 0.01 MB/s
```

**Latency (P50)**
```
devnull_s3   ████████████████ 45.5ms
usagi_s3     ██████████████████████████████ 80.3ms
minio        █████████████████████████ 68.5ms
```

### ParallelWrite/1KB/C25

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.13 MB/s | 8.3ms | 12.3ms | 15.8ms | 0 |
| minio | 0.12 MB/s | 7.5ms | 13.5ms | 19.5ms | 0 |
| devnull_s3 | 0.10 MB/s | 10.2ms | 14.0ms | 17.7ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.13 MB/s
minio        ████████████████████████████ 0.12 MB/s
devnull_s3   ████████████████████████ 0.10 MB/s
```

**Latency (P50)**
```
usagi_s3     ████████████████████████ 8.3ms
minio        ██████████████████████ 7.5ms
devnull_s3   ██████████████████████████████ 10.2ms
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.08 MB/s | 14.3ms | 19.6ms | 23.9ms | 0 |
| devnull_s3 | 0.07 MB/s | 15.2ms | 22.0ms | 35.7ms | 0 |
| minio | 0.05 MB/s | 16.0ms | 38.3ms | 48.6ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.08 MB/s
devnull_s3   ████████████████████████████ 0.07 MB/s
minio        █████████████████████ 0.05 MB/s
```

**Latency (P50)**
```
usagi_s3     ██████████████████████████ 14.3ms
devnull_s3   ████████████████████████████ 15.2ms
minio        ██████████████████████████████ 16.0ms
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 146.93 MB/s | 1.6ms | 2.2ms | 2.8ms | 0 |
| minio | 130.76 MB/s | 1.8ms | 2.6ms | 3.2ms | 0 |
| usagi_s3 | 100.89 MB/s | 2.2ms | 4.1ms | 6.0ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 146.93 MB/s
minio        ██████████████████████████ 130.76 MB/s
usagi_s3     ████████████████████ 100.89 MB/s
```

**Latency (P50)**
```
devnull_s3   ██████████████████████ 1.6ms
minio        ████████████████████████ 1.8ms
usagi_s3     ██████████████████████████████ 2.2ms
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 147.30 MB/s | 1.6ms | 2.3ms | 2.7ms | 0 |
| minio | 129.37 MB/s | 1.8ms | 2.5ms | 3.6ms | 0 |
| usagi_s3 | 90.70 MB/s | 2.6ms | 4.2ms | 6.3ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 147.30 MB/s
minio        ██████████████████████████ 129.37 MB/s
usagi_s3     ██████████████████ 90.70 MB/s
```

**Latency (P50)**
```
devnull_s3   ██████████████████ 1.6ms
minio        ████████████████████ 1.8ms
usagi_s3     ██████████████████████████████ 2.6ms
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 139.15 MB/s | 1.7ms | 2.5ms | 3.2ms | 0 |
| minio | 107.31 MB/s | 2.1ms | 3.8ms | 5.8ms | 0 |
| usagi_s3 | 101.35 MB/s | 2.3ms | 3.9ms | 5.7ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 139.15 MB/s
minio        ███████████████████████ 107.31 MB/s
usagi_s3     █████████████████████ 101.35 MB/s
```

**Latency (P50)**
```
devnull_s3   ██████████████████████ 1.7ms
minio        ███████████████████████████ 2.1ms
usagi_s3     ██████████████████████████████ 2.3ms
```

### Read/100MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 242.30 MB/s | 1.1ms | 1.1ms | 410.5ms | 410.5ms | 410.5ms | 0 |
| usagi_s3 | 212.02 MB/s | 1.9ms | 1.9ms | 471.2ms | 471.2ms | 471.2ms | 0 |
| devnull_s3 | 183.46 MB/s | 3.3ms | 2.1ms | 550.8ms | 550.8ms | 550.8ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 242.30 MB/s
usagi_s3     ██████████████████████████ 212.02 MB/s
devnull_s3   ██████████████████████ 183.46 MB/s
```

**Latency (P50)**
```
minio        ██████████████████████ 410.5ms
usagi_s3     █████████████████████████ 471.2ms
devnull_s3   ██████████████████████████████ 550.8ms
```

### Read/10MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 236.22 MB/s | 1.2ms | 1.3ms | 41.4ms | 49.8ms | 49.9ms | 0 |
| usagi_s3 | 204.91 MB/s | 2.1ms | 3.0ms | 47.9ms | 53.0ms | 57.5ms | 0 |
| devnull_s3 | 199.21 MB/s | 2.6ms | 4.4ms | 49.8ms | 53.1ms | 56.1ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 236.22 MB/s
usagi_s3     ██████████████████████████ 204.91 MB/s
devnull_s3   █████████████████████████ 199.21 MB/s
```

**Latency (P50)**
```
minio        ████████████████████████ 41.4ms
usagi_s3     ████████████████████████████ 47.9ms
devnull_s3   ██████████████████████████████ 49.8ms
```

### Read/1KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| usagi_s3 | 4.33 MB/s | 225.5us | 359.3us | 204.0us | 359.8us | 585.2us | 0 |
| devnull_s3 | 4.26 MB/s | 229.2us | 356.7us | 208.4us | 356.8us | 558.0us | 0 |
| minio | 2.56 MB/s | 382.1us | 528.5us | 351.1us | 528.7us | 868.3us | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 4.33 MB/s
devnull_s3   █████████████████████████████ 4.26 MB/s
minio        █████████████████ 2.56 MB/s
```

**Latency (P50)**
```
usagi_s3     █████████████████ 204.0us
devnull_s3   █████████████████ 208.4us
minio        ██████████████████████████████ 351.1us
```

### Read/1MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| usagi_s3 | 206.74 MB/s | 517.6us | 918.9us | 4.7ms | 6.0ms | 7.1ms | 0 |
| minio | 202.89 MB/s | 1.1ms | 1.5ms | 4.8ms | 6.1ms | 7.4ms | 0 |
| devnull_s3 | 201.99 MB/s | 512.9us | 865.3us | 4.8ms | 6.1ms | 7.0ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 206.74 MB/s
minio        █████████████████████████████ 202.89 MB/s
devnull_s3   █████████████████████████████ 201.99 MB/s
```

**Latency (P50)**
```
usagi_s3     █████████████████████████████ 4.7ms
minio        █████████████████████████████ 4.8ms
devnull_s3   ██████████████████████████████ 4.8ms
```

### Read/64KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| usagi_s3 | 113.06 MB/s | 296.7us | 607.2us | 510.4us | 867.5us | 1.1ms | 0 |
| devnull_s3 | 106.81 MB/s | 306.0us | 609.9us | 531.0us | 931.3us | 1.3ms | 0 |
| minio | 93.98 MB/s | 496.4us | 795.2us | 613.1us | 977.8us | 1.5ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 113.06 MB/s
devnull_s3   ████████████████████████████ 106.81 MB/s
minio        ████████████████████████ 93.98 MB/s
```

**Latency (P50)**
```
usagi_s3     ████████████████████████ 510.4us
devnull_s3   █████████████████████████ 531.0us
minio        ██████████████████████████████ 613.1us
```

### Scale/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 409 ops/s | 2.4ms | 2.4ms | 2.4ms | 0 |
| usagi_s3 | 378 ops/s | 2.6ms | 2.6ms | 2.6ms | 0 |
| minio | 229 ops/s | 4.4ms | 4.4ms | 4.4ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 409 ops/s
usagi_s3     ███████████████████████████ 378 ops/s
minio        ████████████████ 229 ops/s
```

**Latency (P50)**
```
devnull_s3   ████████████████ 2.4ms
usagi_s3     ██████████████████ 2.6ms
minio        ██████████████████████████████ 4.4ms
```

### Scale/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 37 ops/s | 26.9ms | 26.9ms | 26.9ms | 0 |
| devnull_s3 | 36 ops/s | 27.6ms | 27.6ms | 27.6ms | 0 |
| minio | 24 ops/s | 41.0ms | 41.0ms | 41.0ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 37 ops/s
devnull_s3   █████████████████████████████ 36 ops/s
minio        ███████████████████ 24 ops/s
```

**Latency (P50)**
```
usagi_s3     ███████████████████ 26.9ms
devnull_s3   ████████████████████ 27.6ms
minio        ██████████████████████████████ 41.0ms
```

### Scale/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 4 ops/s | 278.4ms | 278.4ms | 278.4ms | 0 |
| usagi_s3 | 3 ops/s | 295.1ms | 295.1ms | 295.1ms | 0 |
| minio | 2 ops/s | 440.4ms | 440.4ms | 440.4ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 4 ops/s
usagi_s3     ████████████████████████████ 3 ops/s
minio        ██████████████████ 2 ops/s
```

**Latency (P50)**
```
devnull_s3   ██████████████████ 278.4ms
usagi_s3     ████████████████████ 295.1ms
minio        ██████████████████████████████ 440.4ms
```

### Scale/Delete/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 0 ops/s | 2.74s | 2.74s | 2.74s | 0 |
| usagi_s3 | 0 ops/s | 3.02s | 3.02s | 3.02s | 0 |
| minio | 0 ops/s | 4.15s | 4.15s | 4.15s | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 0 ops/s
usagi_s3     ███████████████████████████ 0 ops/s
minio        ███████████████████ 0 ops/s
```

**Latency (P50)**
```
devnull_s3   ███████████████████ 2.74s
usagi_s3     █████████████████████ 3.02s
minio        ██████████████████████████████ 4.15s
```

### Scale/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 2101 ops/s | 476.0us | 476.0us | 476.0us | 0 |
| usagi_s3 | 1652 ops/s | 605.2us | 605.2us | 605.2us | 0 |
| minio | 873 ops/s | 1.1ms | 1.1ms | 1.1ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 2101 ops/s
usagi_s3     ███████████████████████ 1652 ops/s
minio        ████████████ 873 ops/s
```

**Latency (P50)**
```
devnull_s3   ████████████ 476.0us
usagi_s3     ███████████████ 605.2us
minio        ██████████████████████████████ 1.1ms
```

### Scale/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 901 ops/s | 1.1ms | 1.1ms | 1.1ms | 0 |
| usagi_s3 | 809 ops/s | 1.2ms | 1.2ms | 1.2ms | 0 |
| minio | 430 ops/s | 2.3ms | 2.3ms | 2.3ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 901 ops/s
usagi_s3     ██████████████████████████ 809 ops/s
minio        ██████████████ 430 ops/s
```

**Latency (P50)**
```
devnull_s3   ██████████████ 1.1ms
usagi_s3     ███████████████ 1.2ms
minio        ██████████████████████████████ 2.3ms
```

### Scale/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 145 ops/s | 6.9ms | 6.9ms | 6.9ms | 0 |
| usagi_s3 | 114 ops/s | 8.8ms | 8.8ms | 8.8ms | 0 |
| minio | 65 ops/s | 15.3ms | 15.3ms | 15.3ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 145 ops/s
usagi_s3     ███████████████████████ 114 ops/s
minio        █████████████ 65 ops/s
```

**Latency (P50)**
```
devnull_s3   █████████████ 6.9ms
usagi_s3     █████████████████ 8.8ms
minio        ██████████████████████████████ 15.3ms
```

### Scale/List/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 6 ops/s | 174.6ms | 174.6ms | 174.6ms | 0 |
| devnull_s3 | 4 ops/s | 233.7ms | 233.7ms | 233.7ms | 0 |
| usagi_s3 | 3 ops/s | 288.7ms | 288.7ms | 288.7ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 6 ops/s
devnull_s3   ██████████████████████ 4 ops/s
usagi_s3     ██████████████████ 3 ops/s
```

**Latency (P50)**
```
minio        ██████████████████ 174.6ms
devnull_s3   ████████████████████████ 233.7ms
usagi_s3     ██████████████████████████████ 288.7ms
```

### Scale/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 0.42 MB/s | 5.8ms | 5.8ms | 5.8ms | 0 |
| usagi_s3 | 0.40 MB/s | 6.1ms | 6.1ms | 6.1ms | 0 |
| minio | 0.22 MB/s | 11.0ms | 11.0ms | 11.0ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 0.42 MB/s
usagi_s3     ████████████████████████████ 0.40 MB/s
minio        ███████████████ 0.22 MB/s
```

**Latency (P50)**
```
devnull_s3   ███████████████ 5.8ms
usagi_s3     ████████████████ 6.1ms
minio        ██████████████████████████████ 11.0ms
```

### Scale/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.36 MB/s | 67.5ms | 67.5ms | 67.5ms | 0 |
| devnull_s3 | 0.36 MB/s | 68.7ms | 68.7ms | 68.7ms | 0 |
| minio | 0.25 MB/s | 98.0ms | 98.0ms | 98.0ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.36 MB/s
devnull_s3   █████████████████████████████ 0.36 MB/s
minio        ████████████████████ 0.25 MB/s
```

**Latency (P50)**
```
usagi_s3     ████████████████████ 67.5ms
devnull_s3   █████████████████████ 68.7ms
minio        ██████████████████████████████ 98.0ms
```

### Scale/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 0.39 MB/s | 630.5ms | 630.5ms | 630.5ms | 0 |
| usagi_s3 | 0.31 MB/s | 789.1ms | 789.1ms | 789.1ms | 0 |
| minio | 0.25 MB/s | 959.6ms | 959.6ms | 959.6ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 0.39 MB/s
usagi_s3     ███████████████████████ 0.31 MB/s
minio        ███████████████████ 0.25 MB/s
```

**Latency (P50)**
```
devnull_s3   ███████████████████ 630.5ms
usagi_s3     ████████████████████████ 789.1ms
minio        ██████████████████████████████ 959.6ms
```

### Scale/Write/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 0.39 MB/s | 6.21s | 6.21s | 6.21s | 0 |
| usagi_s3 | 0.36 MB/s | 6.87s | 6.87s | 6.87s | 0 |
| minio | 0.23 MB/s | 10.78s | 10.78s | 10.78s | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 0.39 MB/s
usagi_s3     ███████████████████████████ 0.36 MB/s
minio        █████████████████ 0.23 MB/s
```

**Latency (P50)**
```
devnull_s3   █████████████████ 6.21s
usagi_s3     ███████████████████ 6.87s
minio        ██████████████████████████████ 10.78s
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 4541 ops/s | 199.8us | 340.3us | 528.7us | 0 |
| devnull_s3 | 4331 ops/s | 208.6us | 350.7us | 583.1us | 0 |
| minio | 3172 ops/s | 286.5us | 463.3us | 869.7us | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 4541 ops/s
devnull_s3   ████████████████████████████ 4331 ops/s
minio        ████████████████████ 3172 ops/s
```

**Latency (P50)**
```
usagi_s3     ████████████████████ 199.8us
devnull_s3   █████████████████████ 208.6us
minio        ██████████████████████████████ 286.5us
```

### Write/100MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 158.89 MB/s | 637.3ms | 637.3ms | 637.3ms | 0 |
| minio | 152.64 MB/s | 660.8ms | 660.8ms | 660.8ms | 0 |
| devnull_s3 | 148.96 MB/s | 678.9ms | 678.9ms | 678.9ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 158.89 MB/s
minio        ████████████████████████████ 152.64 MB/s
devnull_s3   ████████████████████████████ 148.96 MB/s
```

**Latency (P50)**
```
usagi_s3     ████████████████████████████ 637.3ms
minio        █████████████████████████████ 660.8ms
devnull_s3   ██████████████████████████████ 678.9ms
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 158.06 MB/s | 62.9ms | 68.5ms | 68.5ms | 0 |
| devnull_s3 | 146.45 MB/s | 66.9ms | 72.9ms | 72.9ms | 0 |
| minio | 142.22 MB/s | 70.3ms | 78.7ms | 78.7ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 158.06 MB/s
devnull_s3   ███████████████████████████ 146.45 MB/s
minio        ██████████████████████████ 142.22 MB/s
```

**Latency (P50)**
```
usagi_s3     ██████████████████████████ 62.9ms
devnull_s3   ████████████████████████████ 66.9ms
minio        ██████████████████████████████ 70.3ms
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 1.72 MB/s | 521.5us | 844.1us | 1.5ms | 0 |
| devnull_s3 | 1.67 MB/s | 540.0us | 850.3us | 1.3ms | 0 |
| minio | 1.23 MB/s | 719.8us | 1.1ms | 2.2ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 1.72 MB/s
devnull_s3   █████████████████████████████ 1.67 MB/s
minio        █████████████████████ 1.23 MB/s
```

**Latency (P50)**
```
usagi_s3     █████████████████████ 521.5us
devnull_s3   ██████████████████████ 540.0us
minio        ██████████████████████████████ 719.8us
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 133.74 MB/s | 7.1ms | 10.4ms | 11.6ms | 0 |
| usagi_s3 | 126.95 MB/s | 7.1ms | 10.6ms | 18.6ms | 0 |
| minio | 118.28 MB/s | 8.1ms | 10.3ms | 12.4ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 133.74 MB/s
usagi_s3     ████████████████████████████ 126.95 MB/s
minio        ██████████████████████████ 118.28 MB/s
```

**Latency (P50)**
```
devnull_s3   ██████████████████████████ 7.1ms
usagi_s3     ██████████████████████████ 7.1ms
minio        ██████████████████████████████ 8.1ms
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 62.80 MB/s | 901.5us | 1.5ms | 2.0ms | 0 |
| devnull_s3 | 60.12 MB/s | 947.4us | 1.5ms | 2.3ms | 0 |
| minio | 43.63 MB/s | 1.3ms | 2.1ms | 3.5ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 62.80 MB/s
devnull_s3   ████████████████████████████ 60.12 MB/s
minio        ████████████████████ 43.63 MB/s
```

**Latency (P50)**
```
usagi_s3     ████████████████████ 901.5us
devnull_s3   █████████████████████ 947.4us
minio        ██████████████████████████████ 1.3ms
```

## Resource Usage

| Driver | Memory | RSS | Cache | CPU | Volume | Block I/O |
|--------|--------|-----|-------|-----|--------|----------|
| devnull_s3 | 1.505GiB / 7.653GiB | 1541.1 MB | - | 3.6% | 1825.8 MB | 11MB / 2.27GB |
| minio | 886.8MiB / 7.653GiB | 886.8 MB | - | 0.0% | 2108.0 MB | 47.7MB / 2.37GB |
| usagi_s3 | 1.55GiB / 7.653GiB | 1587.2 MB | - | 3.2% | 8680.4 MB | 123kB / 2.34GB |

> **Note:** RSS = actual application memory. Cache = OS page cache (reclaimable).

## Recommendations

- **Write-heavy workloads:** usagi_s3
- **Read-heavy workloads:** minio

---

*Generated by storage benchmark CLI*
