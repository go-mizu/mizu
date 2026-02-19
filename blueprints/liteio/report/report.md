# Storage Benchmark Report

**Generated:** 2026-02-19T09:43:11+07:00

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
| Small Read (1KB) | liteio | 16.6 MB/s | +29% vs garage |
| Small Write (1KB) | liteio | 15.2 MB/s | 3.3x vs garage |
| Large Read (10MB) | liteio | 8.3 GB/s | 2.1x vs garage |
| Large Write (10MB) | liteio | 1.3 GB/s | 3.4x vs minio |
| Delete | liteio | 18.7K ops/s | 2.5x vs seaweedfs |
| Stat | liteio | 18.5K ops/s | +36% vs garage |
| List (100 objects) | liteio | 2.5K ops/s | 2.5x vs seaweedfs |
| Copy | liteio | 15.1 MB/s | 3.9x vs garage |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (10MB+) | **liteio** | 1343 MB/s | Best for media, backups |
| Large File Downloads (10MB) | **liteio** | 8329 MB/s | Best for streaming, CDN |
| Small File Operations | **liteio** | 16270 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **liteio** | - | Best for multi-user apps |

### Large File Performance (10MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| garage | 268.0 | 4039.9 | 37.3ms | 2.3ms |
| liteio | 1342.8 | 8328.7 | 7.1ms | 1.1ms |
| minio | 396.2 | 3283.9 | 25.2ms | 3.0ms |
| rustfs | 340.3 | 1789.3 | 28.9ms | 5.6ms |
| seaweedfs | 250.8 | 2170.5 | 37.3ms | 4.5ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| garage | 4743 | 13184 | 205.8us | 73.0us |
| liteio | 15551 | 16989 | 60.4us | 55.3us |
| minio | 2102 | 5600 | 459.8us | 175.9us |
| rustfs | 2131 | 4681 | 455.2us | 208.8us |
| seaweedfs | 2630 | 5693 | 351.8us | 161.2us |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| garage | 13639 | 525 | 3211 |
| liteio | 18534 | 2519 | 18735 |
| minio | 6511 | 432 | 2290 |
| rustfs | 6081 | 318 | 1802 |
| seaweedfs | 8538 | 1005 | 7488 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| garage | 4.06 | 1.04 | 0.20 |
| liteio | 9.83 | 3.79 | 0.92 |
| minio | 1.96 | 0.39 | 0.09 |
| rustfs | 1.91 | 0.40 | 0.09 |
| seaweedfs | 2.60 | 0.88 | 0.23 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| garage | 8.77 | 2.31 | 0.58 |
| liteio | 10.47 | 3.71 | 1.01 |
| minio | 4.45 | 0.99 | 0.26 |
| rustfs | 3.82 | 0.44 | 0.10 |
| seaweedfs | 4.68 | 1.68 | 0.42 |

*\* indicates errors occurred*

### Scale Performance

Performance with varying numbers of objects (256B each).

**Write N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 |
|--------|------|------|------|------|
| garage | 197.6us | 2.7ms | 27.4ms | 264.6ms |
| liteio | 73.4us | 620.0us | 6.1ms | 62.3ms |
| minio | 500.8us | 4.8ms | 46.2ms | 464.7ms |
| rustfs | 678.8us | 5.4ms | 50.0ms | 548.8ms |
| seaweedfs | 453.5us | 3.8ms | 38.4ms | 332.2ms |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 |
|--------|------|------|------|------|
| garage | 3.6ms | 1.9ms | 2.2ms | 6.7ms |
| liteio | 100.2us | 120.2us | 435.7us | 3.6ms |
| minio | 445.6us | 724.2us | 3.7ms | 26.2ms |
| rustfs | 782.9us | 908.0us | 4.4ms | 41.0ms |
| seaweedfs | 358.0us | 457.2us | 1.2ms | 6.2ms |

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

- **garage** (40 benchmarks)
- **liteio** (40 benchmarks)
- **minio** (40 benchmarks)
- **rustfs** (40 benchmarks)
- **seaweedfs** (40 benchmarks)

## Detailed Results

### Copy/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 15.07 MB/s | 60.1us | 76.8us | 128.0us | 0 |
| garage | 3.83 MB/s | 239.3us | 370.3us | 529.3us | 0 |
| seaweedfs | 2.01 MB/s | 457.6us | 611.4us | 1.1ms | 0 |
| minio | 1.91 MB/s | 494.2us | 619.2us | 683.1us | 0 |
| rustfs | 1.66 MB/s | 551.8us | 717.5us | 1.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 15.07 MB/s
garage       ███████ 3.83 MB/s
seaweedfs    ███ 2.01 MB/s
minio        ███ 1.91 MB/s
rustfs       ███ 1.66 MB/s
```

**Latency (P50)**
```
liteio       ███ 60.1us
garage       █████████████ 239.3us
seaweedfs    ████████████████████████ 457.6us
minio        ██████████████████████████ 494.2us
rustfs       ██████████████████████████████ 551.8us
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 18735 ops/s | 50.6us | 61.0us | 92.3us | 0 |
| seaweedfs | 7488 ops/s | 121.4us | 183.2us | 270.7us | 0 |
| garage | 3211 ops/s | 277.9us | 529.7us | 732.2us | 0 |
| minio | 2290 ops/s | 429.2us | 508.3us | 539.8us | 0 |
| rustfs | 1802 ops/s | 546.8us | 619.6us | 664.7us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 18735 ops/s
seaweedfs    ███████████ 7488 ops/s
garage       █████ 3211 ops/s
minio        ███ 2290 ops/s
rustfs       ██ 1802 ops/s
```

**Latency (P50)**
```
liteio       ██ 50.6us
seaweedfs    ██████ 121.4us
garage       ███████████████ 277.9us
minio        ███████████████████████ 429.2us
rustfs       ██████████████████████████████ 546.8us
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.54 MB/s | 59.2us | 70.0us | 94.5us | 0 |
| garage | 0.36 MB/s | 243.3us | 375.6us | 545.5us | 0 |
| seaweedfs | 0.29 MB/s | 307.8us | 438.5us | 642.2us | 0 |
| minio | 0.19 MB/s | 478.4us | 573.0us | 647.4us | 0 |
| rustfs | 0.17 MB/s | 519.2us | 680.7us | 908.6us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.54 MB/s
garage       ██████ 0.36 MB/s
seaweedfs    █████ 0.29 MB/s
minio        ███ 0.19 MB/s
rustfs       ███ 0.17 MB/s
```

**Latency (P50)**
```
liteio       ███ 59.2us
garage       ██████████████ 243.3us
seaweedfs    █████████████████ 307.8us
minio        ███████████████████████████ 478.4us
rustfs       ██████████████████████████████ 519.2us
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 16251 ops/s | 58.3us | 74.0us | 96.9us | 0 |
| seaweedfs | 6549 ops/s | 141.9us | 196.8us | 275.8us | 0 |
| garage | 3884 ops/s | 241.4us | 382.3us | 532.0us | 0 |
| minio | 2259 ops/s | 430.7us | 509.6us | 547.2us | 0 |
| rustfs | 2085 ops/s | 463.6us | 572.8us | 680.4us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 16251 ops/s
seaweedfs    ████████████ 6549 ops/s
garage       ███████ 3884 ops/s
minio        ████ 2259 ops/s
rustfs       ███ 2085 ops/s
```

**Latency (P50)**
```
liteio       ███ 58.3us
seaweedfs    █████████ 141.9us
garage       ███████████████ 241.4us
minio        ███████████████████████████ 430.7us
rustfs       ██████████████████████████████ 463.6us
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.42 MB/s | 63.5us | 76.5us | 113.7us | 0 |
| garage | 0.36 MB/s | 246.8us | 377.6us | 572.7us | 0 |
| seaweedfs | 0.26 MB/s | 343.8us | 489.4us | 711.2us | 0 |
| minio | 0.18 MB/s | 517.6us | 587.6us | 631.9us | 0 |
| rustfs | 0.17 MB/s | 517.5us | 668.2us | 847.4us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.42 MB/s
garage       ███████ 0.36 MB/s
seaweedfs    █████ 0.26 MB/s
minio        ███ 0.18 MB/s
rustfs       ███ 0.17 MB/s
```

**Latency (P50)**
```
liteio       ███ 63.5us
garage       ██████████████ 246.8us
seaweedfs    ███████████████████ 343.8us
minio        ██████████████████████████████ 517.6us
rustfs       █████████████████████████████ 517.5us
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 2519 ops/s | 387.2us | 419.9us | 694.5us | 0 |
| seaweedfs | 1005 ops/s | 925.4us | 1.4ms | 1.7ms | 0 |
| garage | 525 ops/s | 1.8ms | 2.2ms | 2.7ms | 0 |
| minio | 432 ops/s | 2.3ms | 2.5ms | 2.7ms | 0 |
| rustfs | 318 ops/s | 3.1ms | 3.4ms | 3.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 2519 ops/s
seaweedfs    ███████████ 1005 ops/s
garage       ██████ 525 ops/s
minio        █████ 432 ops/s
rustfs       ███ 318 ops/s
```

**Latency (P50)**
```
liteio       ███ 387.2us
seaweedfs    ████████ 925.4us
garage       █████████████████ 1.8ms
minio        ██████████████████████ 2.3ms
rustfs       ██████████████████████████████ 3.1ms
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 4.00 MB/s | 2.5ms | 12.0ms | 21.6ms | 0 |
| seaweedfs | 1.17 MB/s | 12.2ms | 28.2ms | 36.6ms | 0 |
| minio | 0.78 MB/s | 14.7ms | 52.9ms | 89.4ms | 0 |
| rustfs | 0.35 MB/s | 37.5ms | 64.3ms | 66.5ms | 0 |
| garage | 0.33 MB/s | 26.6ms | 111.4ms | 160.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4.00 MB/s
seaweedfs    ████████ 1.17 MB/s
minio        █████ 0.78 MB/s
rustfs       ██ 0.35 MB/s
garage       ██ 0.33 MB/s
```

**Latency (P50)**
```
liteio       ██ 2.5ms
seaweedfs    █████████ 12.2ms
minio        ███████████ 14.7ms
rustfs       ██████████████████████████████ 37.5ms
garage       █████████████████████ 26.6ms
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 4.86 MB/s | 2.3ms | 9.3ms | 15.7ms | 0 |
| seaweedfs | 1.56 MB/s | 8.7ms | 23.7ms | 33.8ms | 0 |
| minio | 1.55 MB/s | 6.9ms | 32.4ms | 52.7ms | 0 |
| garage | 0.81 MB/s | 6.7ms | 140.9ms | 169.3ms | 0 |
| rustfs | 0.36 MB/s | 41.9ms | 72.8ms | 79.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4.86 MB/s
seaweedfs    █████████ 1.56 MB/s
minio        █████████ 1.55 MB/s
garage       ████ 0.81 MB/s
rustfs       ██ 0.36 MB/s
```

**Latency (P50)**
```
liteio       █ 2.3ms
seaweedfs    ██████ 8.7ms
minio        ████ 6.9ms
garage       ████ 6.7ms
rustfs       ██████████████████████████████ 41.9ms
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 4.09 MB/s | 2.5ms | 11.8ms | 22.0ms | 0 |
| seaweedfs | 0.90 MB/s | 16.6ms | 32.1ms | 41.6ms | 0 |
| minio | 0.48 MB/s | 23.9ms | 95.9ms | 126.5ms | 0 |
| rustfs | 0.33 MB/s | 49.9ms | 72.7ms | 105.1ms | 0 |
| garage | 0.21 MB/s | 86.6ms | 104.9ms | 148.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4.09 MB/s
seaweedfs    ██████ 0.90 MB/s
minio        ███ 0.48 MB/s
rustfs       ██ 0.33 MB/s
garage       █ 0.21 MB/s
```

**Latency (P50)**
```
liteio       █ 2.5ms
seaweedfs    █████ 16.6ms
minio        ████████ 23.9ms
rustfs       █████████████████ 49.9ms
garage       ██████████████████████████████ 86.6ms
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 371.02 MB/s | 39.0ms | 42.8ms | 42.8ms | 0 |
| minio | 330.65 MB/s | 45.2ms | 47.8ms | 47.8ms | 0 |
| rustfs | 287.64 MB/s | 49.0ms | 70.3ms | 70.3ms | 0 |
| garage | 244.70 MB/s | 60.6ms | 65.0ms | 65.0ms | 0 |
| seaweedfs | 210.57 MB/s | 68.8ms | 74.8ms | 74.8ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 371.02 MB/s
minio        ██████████████████████████ 330.65 MB/s
rustfs       ███████████████████████ 287.64 MB/s
garage       ███████████████████ 244.70 MB/s
seaweedfs    █████████████████ 210.57 MB/s
```

**Latency (P50)**
```
liteio       ████████████████ 39.0ms
minio        ███████████████████ 45.2ms
rustfs       █████████████████████ 49.0ms
garage       ██████████████████████████ 60.6ms
seaweedfs    ██████████████████████████████ 68.8ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 10.47 MB/s | 90.2us | 105.0us | 131.5us | 0 |
| garage | 8.77 MB/s | 109.2us | 122.9us | 146.6us | 0 |
| seaweedfs | 4.68 MB/s | 197.8us | 255.8us | 388.0us | 0 |
| minio | 4.45 MB/s | 215.5us | 249.2us | 274.3us | 0 |
| rustfs | 3.82 MB/s | 247.5us | 302.9us | 352.8us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 10.47 MB/s
garage       █████████████████████████ 8.77 MB/s
seaweedfs    █████████████ 4.68 MB/s
minio        ████████████ 4.45 MB/s
rustfs       ██████████ 3.82 MB/s
```

**Latency (P50)**
```
liteio       ██████████ 90.2us
garage       █████████████ 109.2us
seaweedfs    ███████████████████████ 197.8us
minio        ██████████████████████████ 215.5us
rustfs       ██████████████████████████████ 247.5us
```

### ParallelRead/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 3.71 MB/s | 233.3us | 471.3us | 814.8us | 0 |
| garage | 2.31 MB/s | 400.5us | 699.4us | 963.2us | 0 |
| seaweedfs | 1.68 MB/s | 544.0us | 922.8us | 1.3ms | 0 |
| minio | 0.99 MB/s | 626.4us | 2.4ms | 5.3ms | 0 |
| rustfs | 0.44 MB/s | 2.2ms | 3.2ms | 3.8ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3.71 MB/s
garage       ██████████████████ 2.31 MB/s
seaweedfs    █████████████ 1.68 MB/s
minio        ███████ 0.99 MB/s
rustfs       ███ 0.44 MB/s
```

**Latency (P50)**
```
liteio       ███ 233.3us
garage       █████ 400.5us
seaweedfs    ███████ 544.0us
minio        ████████ 626.4us
rustfs       ██████████████████████████████ 2.2ms
```

### ParallelRead/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.01 MB/s | 722.0us | 2.5ms | 4.3ms | 0 |
| garage | 0.58 MB/s | 1.5ms | 3.3ms | 5.4ms | 0 |
| seaweedfs | 0.42 MB/s | 2.0ms | 5.1ms | 9.1ms | 0 |
| minio | 0.26 MB/s | 2.6ms | 9.4ms | 20.2ms | 0 |
| rustfs | 0.10 MB/s | 10.2ms | 12.6ms | 13.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.01 MB/s
garage       █████████████████ 0.58 MB/s
seaweedfs    ████████████ 0.42 MB/s
minio        ███████ 0.26 MB/s
rustfs       ██ 0.10 MB/s
```

**Latency (P50)**
```
liteio       ██ 722.0us
garage       ████ 1.5ms
seaweedfs    █████ 2.0ms
minio        ███████ 2.6ms
rustfs       ██████████████████████████████ 10.2ms
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 9.83 MB/s | 95.0us | 112.1us | 190.6us | 0 |
| garage | 4.06 MB/s | 230.0us | 301.7us | 458.1us | 0 |
| seaweedfs | 2.60 MB/s | 350.8us | 504.2us | 753.2us | 0 |
| minio | 1.96 MB/s | 479.9us | 580.5us | 641.5us | 0 |
| rustfs | 1.91 MB/s | 495.9us | 605.8us | 698.5us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 9.83 MB/s
garage       ████████████ 4.06 MB/s
seaweedfs    ███████ 2.60 MB/s
minio        █████ 1.96 MB/s
rustfs       █████ 1.91 MB/s
```

**Latency (P50)**
```
liteio       █████ 95.0us
garage       █████████████ 230.0us
seaweedfs    █████████████████████ 350.8us
minio        █████████████████████████████ 479.9us
rustfs       ██████████████████████████████ 495.9us
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 3.79 MB/s | 236.5us | 442.5us | 727.8us | 0 |
| garage | 1.04 MB/s | 860.6us | 1.6ms | 2.3ms | 0 |
| seaweedfs | 0.88 MB/s | 1.0ms | 1.6ms | 2.2ms | 0 |
| rustfs | 0.40 MB/s | 2.3ms | 3.8ms | 6.0ms | 0 |
| minio | 0.39 MB/s | 1.6ms | 5.3ms | 21.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3.79 MB/s
garage       ████████ 1.04 MB/s
seaweedfs    ██████ 0.88 MB/s
rustfs       ███ 0.40 MB/s
minio        ███ 0.39 MB/s
```

**Latency (P50)**
```
liteio       ███ 236.5us
garage       ███████████ 860.6us
seaweedfs    █████████████ 1.0ms
rustfs       ██████████████████████████████ 2.3ms
minio        ████████████████████ 1.6ms
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.92 MB/s | 767.6us | 2.9ms | 4.5ms | 0 |
| seaweedfs | 0.23 MB/s | 3.9ms | 7.1ms | 10.7ms | 0 |
| garage | 0.20 MB/s | 4.9ms | 8.0ms | 9.3ms | 0 |
| minio | 0.09 MB/s | 7.5ms | 29.8ms | 50.7ms | 0 |
| rustfs | 0.09 MB/s | 10.8ms | 14.8ms | 17.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.92 MB/s
seaweedfs    ███████ 0.23 MB/s
garage       ██████ 0.20 MB/s
minio        ███ 0.09 MB/s
rustfs       ██ 0.09 MB/s
```

**Latency (P50)**
```
liteio       ██ 767.6us
seaweedfs    ██████████ 3.9ms
garage       █████████████ 4.9ms
minio        ████████████████████ 7.5ms
rustfs       ██████████████████████████████ 10.8ms
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 2813.36 MB/s | 80.5us | 124.4us | 180.2us | 0 |
| seaweedfs | 1012.85 MB/s | 230.3us | 321.3us | 505.0us | 0 |
| minio | 552.33 MB/s | 429.3us | 497.7us | 816.6us | 0 |
| garage | 510.52 MB/s | 482.9us | 577.2us | 697.0us | 0 |
| rustfs | 383.72 MB/s | 618.4us | 773.6us | 1.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 2813.36 MB/s
seaweedfs    ██████████ 1012.85 MB/s
minio        █████ 552.33 MB/s
garage       █████ 510.52 MB/s
rustfs       ████ 383.72 MB/s
```

**Latency (P50)**
```
liteio       ███ 80.5us
seaweedfs    ███████████ 230.3us
minio        ████████████████████ 429.3us
garage       ███████████████████████ 482.9us
rustfs       ██████████████████████████████ 618.4us
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 2773.89 MB/s | 82.8us | 122.1us | 175.3us | 0 |
| seaweedfs | 1008.74 MB/s | 232.5us | 326.2us | 428.8us | 0 |
| minio | 583.69 MB/s | 406.5us | 488.5us | 895.4us | 0 |
| garage | 556.72 MB/s | 439.0us | 547.1us | 655.2us | 0 |
| rustfs | 397.82 MB/s | 613.0us | 727.1us | 825.5us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 2773.89 MB/s
seaweedfs    ██████████ 1008.74 MB/s
minio        ██████ 583.69 MB/s
garage       ██████ 556.72 MB/s
rustfs       ████ 397.82 MB/s
```

**Latency (P50)**
```
liteio       ████ 82.8us
seaweedfs    ███████████ 232.5us
minio        ███████████████████ 406.5us
garage       █████████████████████ 439.0us
rustfs       ██████████████████████████████ 613.0us
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 2602.51 MB/s | 86.6us | 136.4us | 212.3us | 0 |
| seaweedfs | 1017.86 MB/s | 230.4us | 312.8us | 482.0us | 0 |
| garage | 643.13 MB/s | 371.2us | 475.3us | 581.9us | 0 |
| minio | 602.94 MB/s | 392.5us | 481.0us | 957.8us | 0 |
| rustfs | 382.00 MB/s | 619.3us | 818.5us | 1.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 2602.51 MB/s
seaweedfs    ███████████ 1017.86 MB/s
garage       ███████ 643.13 MB/s
minio        ██████ 602.94 MB/s
rustfs       ████ 382.00 MB/s
```

**Latency (P50)**
```
liteio       ████ 86.6us
seaweedfs    ███████████ 230.4us
garage       █████████████████ 371.2us
minio        ███████████████████ 392.5us
rustfs       ██████████████████████████████ 619.3us
```

### Read/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 8328.67 MB/s | 1.1ms | 1.9ms | 2.4ms | 0 |
| garage | 4039.92 MB/s | 2.3ms | 3.1ms | 3.7ms | 0 |
| minio | 3283.86 MB/s | 3.0ms | 3.5ms | 3.8ms | 0 |
| seaweedfs | 2170.46 MB/s | 4.5ms | 5.3ms | 5.7ms | 0 |
| rustfs | 1789.27 MB/s | 5.6ms | 6.2ms | 6.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 8328.67 MB/s
garage       ██████████████ 4039.92 MB/s
minio        ███████████ 3283.86 MB/s
seaweedfs    ███████ 2170.46 MB/s
rustfs       ██████ 1789.27 MB/s
```

**Latency (P50)**
```
liteio       ██████ 1.1ms
garage       ████████████ 2.3ms
minio        ████████████████ 3.0ms
seaweedfs    ████████████████████████ 4.5ms
rustfs       ██████████████████████████████ 5.6ms
```

### Read/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 16.59 MB/s | 55.3us | 72.8us | 100.0us | 0 |
| garage | 12.87 MB/s | 73.0us | 88.6us | 111.3us | 0 |
| seaweedfs | 5.56 MB/s | 161.2us | 248.4us | 356.2us | 0 |
| minio | 5.47 MB/s | 175.9us | 205.8us | 235.8us | 0 |
| rustfs | 4.57 MB/s | 208.8us | 247.6us | 284.9us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 16.59 MB/s
garage       ███████████████████████ 12.87 MB/s
seaweedfs    ██████████ 5.56 MB/s
minio        █████████ 5.47 MB/s
rustfs       ████████ 4.57 MB/s
```

**Latency (P50)**
```
liteio       ███████ 55.3us
garage       ██████████ 73.0us
seaweedfs    ███████████████████████ 161.2us
minio        █████████████████████████ 175.9us
rustfs       ██████████████████████████████ 208.8us
```

### Read/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 5757.69 MB/s | 163.5us | 238.0us | 291.5us | 0 |
| seaweedfs | 2057.02 MB/s | 463.4us | 609.4us | 757.8us | 0 |
| garage | 1872.20 MB/s | 528.0us | 619.7us | 701.3us | 0 |
| minio | 1601.06 MB/s | 611.6us | 730.6us | 1.0ms | 0 |
| rustfs | 1179.48 MB/s | 835.1us | 993.3us | 1.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 5757.69 MB/s
seaweedfs    ██████████ 2057.02 MB/s
garage       █████████ 1872.20 MB/s
minio        ████████ 1601.06 MB/s
rustfs       ██████ 1179.48 MB/s
```

**Latency (P50)**
```
liteio       █████ 163.5us
seaweedfs    ████████████████ 463.4us
garage       ██████████████████ 528.0us
minio        █████████████████████ 611.6us
rustfs       ██████████████████████████████ 835.1us
```

### Read/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 925.90 MB/s | 62.8us | 85.5us | 137.3us | 0 |
| minio | 323.63 MB/s | 187.8us | 211.5us | 240.6us | 0 |
| seaweedfs | 318.68 MB/s | 182.5us | 251.0us | 430.9us | 0 |
| garage | 318.05 MB/s | 190.5us | 233.8us | 289.2us | 0 |
| rustfs | 251.60 MB/s | 242.9us | 280.7us | 334.8us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 925.90 MB/s
minio        ██████████ 323.63 MB/s
seaweedfs    ██████████ 318.68 MB/s
garage       ██████████ 318.05 MB/s
rustfs       ████████ 251.60 MB/s
```

**Latency (P50)**
```
liteio       ███████ 62.8us
minio        ███████████████████████ 187.8us
seaweedfs    ██████████████████████ 182.5us
garage       ███████████████████████ 190.5us
rustfs       ██████████████████████████████ 242.9us
```

### Scale/Delete/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 15947 ops/s | 62.7us | 62.7us | 62.7us | 0 |
| seaweedfs | 5463 ops/s | 183.0us | 183.0us | 183.0us | 0 |
| garage | 2276 ops/s | 439.4us | 439.4us | 439.4us | 0 |
| rustfs | 899 ops/s | 1.1ms | 1.1ms | 1.1ms | 0 |
| minio | 600 ops/s | 1.7ms | 1.7ms | 1.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 15947 ops/s
seaweedfs    ██████████ 5463 ops/s
garage       ████ 2276 ops/s
rustfs       █ 899 ops/s
minio        █ 600 ops/s
```

**Latency (P50)**
```
liteio       █ 62.7us
seaweedfs    ███ 183.0us
garage       ███████ 439.4us
rustfs       ████████████████████ 1.1ms
minio        ██████████████████████████████ 1.7ms
```

### Scale/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1856 ops/s | 538.7us | 538.7us | 538.7us | 0 |
| seaweedfs | 664 ops/s | 1.5ms | 1.5ms | 1.5ms | 0 |
| garage | 362 ops/s | 2.8ms | 2.8ms | 2.8ms | 0 |
| minio | 234 ops/s | 4.3ms | 4.3ms | 4.3ms | 0 |
| rustfs | 180 ops/s | 5.5ms | 5.5ms | 5.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1856 ops/s
seaweedfs    ██████████ 664 ops/s
garage       █████ 362 ops/s
minio        ███ 234 ops/s
rustfs       ██ 180 ops/s
```

**Latency (P50)**
```
liteio       ██ 538.7us
seaweedfs    ████████ 1.5ms
garage       ██████████████ 2.8ms
minio        ███████████████████████ 4.3ms
rustfs       ██████████████████████████████ 5.5ms
```

### Scale/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 149 ops/s | 6.7ms | 6.7ms | 6.7ms | 0 |
| seaweedfs | 68 ops/s | 14.7ms | 14.7ms | 14.7ms | 0 |
| garage | 30 ops/s | 33.8ms | 33.8ms | 33.8ms | 0 |
| minio | 24 ops/s | 42.1ms | 42.1ms | 42.1ms | 0 |
| rustfs | 17 ops/s | 60.3ms | 60.3ms | 60.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 149 ops/s
seaweedfs    █████████████ 68 ops/s
garage       █████ 30 ops/s
minio        ████ 24 ops/s
rustfs       ███ 17 ops/s
```

**Latency (P50)**
```
liteio       ███ 6.7ms
seaweedfs    ███████ 14.7ms
garage       ████████████████ 33.8ms
minio        ████████████████████ 42.1ms
rustfs       ██████████████████████████████ 60.3ms
```

### Scale/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 19 ops/s | 52.5ms | 52.5ms | 52.5ms | 0 |
| seaweedfs | 8 ops/s | 128.7ms | 128.7ms | 128.7ms | 0 |
| garage | 3 ops/s | 346.7ms | 346.7ms | 346.7ms | 0 |
| minio | 2 ops/s | 454.7ms | 454.7ms | 454.7ms | 0 |
| rustfs | 1 ops/s | 670.9ms | 670.9ms | 670.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 19 ops/s
seaweedfs    ████████████ 8 ops/s
garage       ████ 3 ops/s
minio        ███ 2 ops/s
rustfs       ██ 1 ops/s
```

**Latency (P50)**
```
liteio       ██ 52.5ms
seaweedfs    █████ 128.7ms
garage       ███████████████ 346.7ms
minio        ████████████████████ 454.7ms
rustfs       ██████████████████████████████ 670.9ms
```

### Scale/List/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 9975 ops/s | 100.2us | 100.2us | 100.2us | 0 |
| seaweedfs | 2793 ops/s | 358.0us | 358.0us | 358.0us | 0 |
| minio | 2244 ops/s | 445.6us | 445.6us | 445.6us | 0 |
| rustfs | 1277 ops/s | 782.9us | 782.9us | 782.9us | 0 |
| garage | 278 ops/s | 3.6ms | 3.6ms | 3.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 9975 ops/s
seaweedfs    ████████ 2793 ops/s
minio        ██████ 2244 ops/s
rustfs       ███ 1277 ops/s
garage       █ 278 ops/s
```

**Latency (P50)**
```
liteio       █ 100.2us
seaweedfs    ██ 358.0us
minio        ███ 445.6us
rustfs       ██████ 782.9us
garage       ██████████████████████████████ 3.6ms
```

### Scale/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 8319 ops/s | 120.2us | 120.2us | 120.2us | 0 |
| seaweedfs | 2187 ops/s | 457.2us | 457.2us | 457.2us | 0 |
| minio | 1381 ops/s | 724.2us | 724.2us | 724.2us | 0 |
| rustfs | 1101 ops/s | 908.0us | 908.0us | 908.0us | 0 |
| garage | 523 ops/s | 1.9ms | 1.9ms | 1.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 8319 ops/s
seaweedfs    ███████ 2187 ops/s
minio        ████ 1381 ops/s
rustfs       ███ 1101 ops/s
garage       █ 523 ops/s
```

**Latency (P50)**
```
liteio       █ 120.2us
seaweedfs    ███████ 457.2us
minio        ███████████ 724.2us
rustfs       ██████████████ 908.0us
garage       ██████████████████████████████ 1.9ms
```

### Scale/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 2295 ops/s | 435.7us | 435.7us | 435.7us | 0 |
| seaweedfs | 861 ops/s | 1.2ms | 1.2ms | 1.2ms | 0 |
| garage | 455 ops/s | 2.2ms | 2.2ms | 2.2ms | 0 |
| minio | 272 ops/s | 3.7ms | 3.7ms | 3.7ms | 0 |
| rustfs | 226 ops/s | 4.4ms | 4.4ms | 4.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 2295 ops/s
seaweedfs    ███████████ 861 ops/s
garage       █████ 455 ops/s
minio        ███ 272 ops/s
rustfs       ██ 226 ops/s
```

**Latency (P50)**
```
liteio       ██ 435.7us
seaweedfs    ███████ 1.2ms
garage       ██████████████ 2.2ms
minio        ████████████████████████ 3.7ms
rustfs       ██████████████████████████████ 4.4ms
```

### Scale/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 280 ops/s | 3.6ms | 3.6ms | 3.6ms | 0 |
| seaweedfs | 161 ops/s | 6.2ms | 6.2ms | 6.2ms | 0 |
| garage | 150 ops/s | 6.7ms | 6.7ms | 6.7ms | 0 |
| minio | 38 ops/s | 26.2ms | 26.2ms | 26.2ms | 0 |
| rustfs | 24 ops/s | 41.0ms | 41.0ms | 41.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 280 ops/s
seaweedfs    █████████████████ 161 ops/s
garage       ████████████████ 150 ops/s
minio        ████ 38 ops/s
rustfs       ██ 24 ops/s
```

**Latency (P50)**
```
liteio       ██ 3.6ms
seaweedfs    ████ 6.2ms
garage       ████ 6.7ms
minio        ███████████████████ 26.2ms
rustfs       ██████████████████████████████ 41.0ms
```

### Scale/Write/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 3.33 MB/s | 73.4us | 73.4us | 73.4us | 0 |
| garage | 1.24 MB/s | 197.6us | 197.6us | 197.6us | 0 |
| seaweedfs | 0.54 MB/s | 453.5us | 453.5us | 453.5us | 0 |
| minio | 0.49 MB/s | 500.8us | 500.8us | 500.8us | 0 |
| rustfs | 0.36 MB/s | 678.8us | 678.8us | 678.8us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3.33 MB/s
garage       ███████████ 1.24 MB/s
seaweedfs    ████ 0.54 MB/s
minio        ████ 0.49 MB/s
rustfs       ███ 0.36 MB/s
```

**Latency (P50)**
```
liteio       ███ 73.4us
garage       ████████ 197.6us
seaweedfs    ████████████████████ 453.5us
minio        ██████████████████████ 500.8us
rustfs       ██████████████████████████████ 678.8us
```

### Scale/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 3.94 MB/s | 620.0us | 620.0us | 620.0us | 0 |
| garage | 0.89 MB/s | 2.7ms | 2.7ms | 2.7ms | 0 |
| seaweedfs | 0.64 MB/s | 3.8ms | 3.8ms | 3.8ms | 0 |
| minio | 0.51 MB/s | 4.8ms | 4.8ms | 4.8ms | 0 |
| rustfs | 0.45 MB/s | 5.4ms | 5.4ms | 5.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3.94 MB/s
garage       ██████ 0.89 MB/s
seaweedfs    ████ 0.64 MB/s
minio        ███ 0.51 MB/s
rustfs       ███ 0.45 MB/s
```

**Latency (P50)**
```
liteio       ███ 620.0us
garage       ███████████████ 2.7ms
seaweedfs    █████████████████████ 3.8ms
minio        ██████████████████████████ 4.8ms
rustfs       ██████████████████████████████ 5.4ms
```

### Scale/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 3.98 MB/s | 6.1ms | 6.1ms | 6.1ms | 0 |
| garage | 0.89 MB/s | 27.4ms | 27.4ms | 27.4ms | 0 |
| seaweedfs | 0.64 MB/s | 38.4ms | 38.4ms | 38.4ms | 0 |
| minio | 0.53 MB/s | 46.2ms | 46.2ms | 46.2ms | 0 |
| rustfs | 0.49 MB/s | 50.0ms | 50.0ms | 50.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3.98 MB/s
garage       ██████ 0.89 MB/s
seaweedfs    ████ 0.64 MB/s
minio        ███ 0.53 MB/s
rustfs       ███ 0.49 MB/s
```

**Latency (P50)**
```
liteio       ███ 6.1ms
garage       ████████████████ 27.4ms
seaweedfs    ███████████████████████ 38.4ms
minio        ███████████████████████████ 46.2ms
rustfs       ██████████████████████████████ 50.0ms
```

### Scale/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 3.92 MB/s | 62.3ms | 62.3ms | 62.3ms | 0 |
| garage | 0.92 MB/s | 264.6ms | 264.6ms | 264.6ms | 0 |
| seaweedfs | 0.73 MB/s | 332.2ms | 332.2ms | 332.2ms | 0 |
| minio | 0.53 MB/s | 464.7ms | 464.7ms | 464.7ms | 0 |
| rustfs | 0.44 MB/s | 548.8ms | 548.8ms | 548.8ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3.92 MB/s
garage       ███████ 0.92 MB/s
seaweedfs    █████ 0.73 MB/s
minio        ████ 0.53 MB/s
rustfs       ███ 0.44 MB/s
```

**Latency (P50)**
```
liteio       ███ 62.3ms
garage       ██████████████ 264.6ms
seaweedfs    ██████████████████ 332.2ms
minio        █████████████████████████ 464.7ms
rustfs       ██████████████████████████████ 548.8ms
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 18534 ops/s | 51.4us | 62.4us | 86.5us | 0 |
| garage | 13639 ops/s | 70.9us | 83.4us | 102.6us | 0 |
| seaweedfs | 8538 ops/s | 109.8us | 151.6us | 209.8us | 0 |
| minio | 6511 ops/s | 152.4us | 173.5us | 190.7us | 0 |
| rustfs | 6081 ops/s | 160.4us | 190.2us | 216.9us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 18534 ops/s
garage       ██████████████████████ 13639 ops/s
seaweedfs    █████████████ 8538 ops/s
minio        ██████████ 6511 ops/s
rustfs       █████████ 6081 ops/s
```

**Latency (P50)**
```
liteio       █████████ 51.4us
garage       █████████████ 70.9us
seaweedfs    ████████████████████ 109.8us
minio        ████████████████████████████ 152.4us
rustfs       ██████████████████████████████ 160.4us
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1342.79 MB/s | 7.1ms | 8.6ms | 9.5ms | 0 |
| minio | 396.19 MB/s | 25.2ms | 25.7ms | 25.8ms | 0 |
| rustfs | 340.31 MB/s | 28.9ms | 31.3ms | 31.3ms | 0 |
| garage | 268.05 MB/s | 37.3ms | 37.6ms | 37.6ms | 0 |
| seaweedfs | 250.78 MB/s | 37.3ms | 43.8ms | 43.8ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1342.79 MB/s
minio        ████████ 396.19 MB/s
rustfs       ███████ 340.31 MB/s
garage       █████ 268.05 MB/s
seaweedfs    █████ 250.78 MB/s
```

**Latency (P50)**
```
liteio       █████ 7.1ms
minio        ████████████████████ 25.2ms
rustfs       ███████████████████████ 28.9ms
garage       █████████████████████████████ 37.3ms
seaweedfs    ██████████████████████████████ 37.3ms
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 15.19 MB/s | 60.4us | 80.8us | 114.7us | 0 |
| garage | 4.63 MB/s | 205.8us | 277.2us | 362.1us | 0 |
| seaweedfs | 2.57 MB/s | 351.8us | 524.1us | 899.5us | 0 |
| rustfs | 2.08 MB/s | 455.2us | 549.5us | 643.3us | 0 |
| minio | 2.05 MB/s | 459.8us | 572.1us | 725.5us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 15.19 MB/s
garage       █████████ 4.63 MB/s
seaweedfs    █████ 2.57 MB/s
rustfs       ████ 2.08 MB/s
minio        ████ 2.05 MB/s
```

**Latency (P50)**
```
liteio       ███ 60.4us
garage       █████████████ 205.8us
seaweedfs    ██████████████████████ 351.8us
rustfs       █████████████████████████████ 455.2us
minio        ██████████████████████████████ 459.8us
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1334.56 MB/s | 702.9us | 997.4us | 1.1ms | 0 |
| minio | 285.48 MB/s | 3.5ms | 3.7ms | 3.9ms | 0 |
| rustfs | 264.97 MB/s | 3.8ms | 4.1ms | 4.3ms | 0 |
| garage | 247.22 MB/s | 4.0ms | 4.3ms | 4.8ms | 0 |
| seaweedfs | 158.73 MB/s | 4.5ms | 11.3ms | 35.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1334.56 MB/s
minio        ██████ 285.48 MB/s
rustfs       █████ 264.97 MB/s
garage       █████ 247.22 MB/s
seaweedfs    ███ 158.73 MB/s
```

**Latency (P50)**
```
liteio       ████ 702.9us
minio        ███████████████████████ 3.5ms
rustfs       █████████████████████████ 3.8ms
garage       ██████████████████████████ 4.0ms
seaweedfs    ██████████████████████████████ 4.5ms
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 604.71 MB/s | 95.5us | 129.3us | 248.5us | 0 |
| seaweedfs | 94.72 MB/s | 589.1us | 845.2us | 2.0ms | 0 |
| minio | 91.61 MB/s | 654.5us | 757.0us | 1.1ms | 0 |
| garage | 90.71 MB/s | 686.4us | 882.1us | 1.0ms | 0 |
| rustfs | 90.64 MB/s | 644.6us | 813.9us | 1.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 604.71 MB/s
seaweedfs    ████ 94.72 MB/s
minio        ████ 91.61 MB/s
garage       ████ 90.71 MB/s
rustfs       ████ 90.64 MB/s
```

**Latency (P50)**
```
liteio       ████ 95.5us
seaweedfs    █████████████████████████ 589.1us
minio        ████████████████████████████ 654.5us
garage       ██████████████████████████████ 686.4us
rustfs       ████████████████████████████ 644.6us
```

## Recommendations

- **Write-heavy workloads:** liteio
- **Read-heavy workloads:** liteio

---

*Generated by storage benchmark CLI*
