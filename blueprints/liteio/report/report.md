# Storage Benchmark Report

**Generated:** 2026-02-19T09:25:59+07:00

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
| Small Read (1KB) | liteio | 15.2 MB/s | 2.8x vs minio |
| Small Write (1KB) | liteio | 13.2 MB/s | 6.2x vs minio |
| Large Read (10MB) | liteio | 8.2 GB/s | 2.5x vs minio |
| Large Write (10MB) | liteio | 1.3 GB/s | 3.4x vs minio |
| Delete | liteio | 17.0K ops/s | 7.4x vs minio |
| Stat | liteio | 16.6K ops/s | 2.6x vs minio |
| List (100 objects) | liteio | 2.5K ops/s | 5.8x vs minio |
| Copy | liteio | 15.6 MB/s | 8.2x vs minio |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (10MB+) | **liteio** | 1342 MB/s | Best for media, backups |
| Large File Downloads (10MB) | **liteio** | 8167 MB/s | Best for streaming, CDN |
| Small File Operations | **liteio** | 14533 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **liteio** | - | Best for multi-user apps |

### Large File Performance (10MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| liteio | 1341.7 | 8166.7 | 7.5ms | 1.1ms |
| minio | 395.1 | 3274.7 | 25.4ms | 3.0ms |
| rustfs | 343.8 | 1812.2 | 28.7ms | 5.5ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| liteio | 13531 | 15536 | 71.0us | 62.8us |
| minio | 2177 | 5497 | 443.1us | 179.2us |
| rustfs | 2173 | 4659 | 432.9us | 210.0us |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| liteio | 16592 | 2491 | 16951 |
| minio | 6377 | 433 | 2301 |
| rustfs | 6009 | 323 | 1834 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| liteio | 9.21 | 4.03 | 1.03 |
| minio | 1.96 | 0.56 | 0.11 |
| rustfs | 1.94 | 0.41 | 0.09 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| liteio | 9.82 | 3.95 | 1.13 |
| minio | 4.45 | 2.39 | 0.54 |
| rustfs | 3.80 | 0.46 | 0.10 |

*\* indicates errors occurred*

### Scale Performance

Performance with varying numbers of objects (256B each).

**Write N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 |
|--------|------|------|------|------|
| liteio | 62.6us | 632.6us | 6.2ms | 62.0ms |
| minio | 501.2us | 4.6ms | 46.8ms | 483.8ms |
| rustfs | 522.2us | 5.4ms | 47.5ms | 594.4ms |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 |
|--------|------|------|------|------|
| liteio | 95.9us | 115.4us | 466.0us | 3.5ms |
| minio | 528.5us | 668.2us | 3.0ms | 25.9ms |
| rustfs | 465.8us | 1.1ms | 4.5ms | 37.9ms |

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
- **rustfs** (40 benchmarks)

## Detailed Results

### Copy/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 15.56 MB/s | 59.3us | 73.2us | 119.0us | 0 |
| minio | 1.91 MB/s | 492.8us | 596.4us | 638.9us | 0 |
| rustfs | 1.73 MB/s | 534.0us | 662.4us | 784.7us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 15.56 MB/s
minio        ███ 1.91 MB/s
rustfs       ███ 1.73 MB/s
```

**Latency (P50)**
```
liteio       ███ 59.3us
minio        ███████████████████████████ 492.8us
rustfs       ██████████████████████████████ 534.0us
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 16951 ops/s | 57.6us | 67.5us | 80.9us | 0 |
| minio | 2301 ops/s | 427.2us | 505.4us | 552.9us | 0 |
| rustfs | 1834 ops/s | 537.3us | 614.7us | 660.3us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 16951 ops/s
minio        ████ 2301 ops/s
rustfs       ███ 1834 ops/s
```

**Latency (P50)**
```
liteio       ███ 57.6us
minio        ███████████████████████ 427.2us
rustfs       ██████████████████████████████ 537.3us
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.53 MB/s | 59.7us | 71.2us | 100.5us | 0 |
| rustfs | 0.20 MB/s | 475.9us | 560.2us | 665.1us | 0 |
| minio | 0.17 MB/s | 498.2us | 875.2us | 1.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.53 MB/s
rustfs       ███ 0.20 MB/s
minio        ███ 0.17 MB/s
```

**Latency (P50)**
```
liteio       ███ 59.7us
rustfs       ████████████████████████████ 475.9us
minio        ██████████████████████████████ 498.2us
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 16267 ops/s | 58.5us | 75.7us | 99.8us | 0 |
| rustfs | 2103 ops/s | 465.7us | 548.3us | 647.8us | 0 |
| minio | 1372 ops/s | 731.3us | 944.0us | 1.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 16267 ops/s
rustfs       ███ 2103 ops/s
minio        ██ 1372 ops/s
```

**Latency (P50)**
```
liteio       ██ 58.5us
rustfs       ███████████████████ 465.7us
minio        ██████████████████████████████ 731.3us
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.43 MB/s | 63.8us | 76.4us | 100.5us | 0 |
| rustfs | 0.19 MB/s | 484.7us | 598.2us | 760.2us | 0 |
| minio | 0.12 MB/s | 810.3us | 1.0ms | 1.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.43 MB/s
rustfs       ███ 0.19 MB/s
minio        ██ 0.12 MB/s
```

**Latency (P50)**
```
liteio       ██ 63.8us
rustfs       █████████████████ 484.7us
minio        ██████████████████████████████ 810.3us
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 2491 ops/s | 394.4us | 420.8us | 655.8us | 0 |
| minio | 433 ops/s | 2.3ms | 2.4ms | 2.7ms | 0 |
| rustfs | 323 ops/s | 3.1ms | 3.3ms | 3.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 2491 ops/s
minio        █████ 433 ops/s
rustfs       ███ 323 ops/s
```

**Latency (P50)**
```
liteio       ███ 394.4us
minio        ██████████████████████ 2.3ms
rustfs       ██████████████████████████████ 3.1ms
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 4.51 MB/s | 2.5ms | 10.1ms | 16.0ms | 0 |
| minio | 0.68 MB/s | 18.2ms | 59.8ms | 108.4ms | 0 |
| rustfs | 0.36 MB/s | 39.7ms | 66.0ms | 68.8ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4.51 MB/s
minio        ████ 0.68 MB/s
rustfs       ██ 0.36 MB/s
```

**Latency (P50)**
```
liteio       █ 2.5ms
minio        █████████████ 18.2ms
rustfs       ██████████████████████████████ 39.7ms
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 5.30 MB/s | 2.1ms | 8.5ms | 15.7ms | 0 |
| minio | 1.68 MB/s | 6.5ms | 28.0ms | 46.4ms | 0 |
| rustfs | 0.38 MB/s | 39.9ms | 69.6ms | 75.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 5.30 MB/s
minio        █████████ 1.68 MB/s
rustfs       ██ 0.38 MB/s
```

**Latency (P50)**
```
liteio       █ 2.1ms
minio        ████ 6.5ms
rustfs       ██████████████████████████████ 39.9ms
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 4.44 MB/s | 2.5ms | 10.1ms | 17.2ms | 0 |
| minio | 0.48 MB/s | 25.9ms | 84.6ms | 142.5ms | 0 |
| rustfs | 0.34 MB/s | 46.7ms | 73.8ms | 100.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4.44 MB/s
minio        ███ 0.48 MB/s
rustfs       ██ 0.34 MB/s
```

**Latency (P50)**
```
liteio       █ 2.5ms
minio        ████████████████ 25.9ms
rustfs       ██████████████████████████████ 46.7ms
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 352.33 MB/s | 40.9ms | 49.3ms | 50.5ms | 0 |
| rustfs | 314.53 MB/s | 47.6ms | 48.7ms | 48.7ms | 0 |
| minio | 299.15 MB/s | 46.8ms | 63.3ms | 63.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 352.33 MB/s
rustfs       ██████████████████████████ 314.53 MB/s
minio        █████████████████████████ 299.15 MB/s
```

**Latency (P50)**
```
liteio       █████████████████████████ 40.9ms
rustfs       ██████████████████████████████ 47.6ms
minio        █████████████████████████████ 46.8ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 9.82 MB/s | 97.8us | 109.4us | 122.1us | 0 |
| minio | 4.45 MB/s | 217.2us | 243.4us | 272.8us | 0 |
| rustfs | 3.80 MB/s | 249.8us | 300.9us | 348.9us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 9.82 MB/s
minio        █████████████ 4.45 MB/s
rustfs       ███████████ 3.80 MB/s
```

**Latency (P50)**
```
liteio       ███████████ 97.8us
minio        ██████████████████████████ 217.2us
rustfs       ██████████████████████████████ 249.8us
```

### ParallelRead/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 3.95 MB/s | 226.0us | 420.8us | 660.2us | 0 |
| minio | 2.39 MB/s | 377.7us | 676.4us | 1.0ms | 0 |
| rustfs | 0.46 MB/s | 2.1ms | 3.1ms | 3.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3.95 MB/s
minio        ██████████████████ 2.39 MB/s
rustfs       ███ 0.46 MB/s
```

**Latency (P50)**
```
liteio       ███ 226.0us
minio        █████ 377.7us
rustfs       ██████████████████████████████ 2.1ms
```

### ParallelRead/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.13 MB/s | 639.7us | 2.3ms | 3.7ms | 0 |
| minio | 0.54 MB/s | 1.4ms | 4.5ms | 8.7ms | 0 |
| rustfs | 0.10 MB/s | 10.1ms | 12.0ms | 12.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.13 MB/s
minio        ██████████████ 0.54 MB/s
rustfs       ██ 0.10 MB/s
```

**Latency (P50)**
```
liteio       █ 639.7us
minio        ████ 1.4ms
rustfs       ██████████████████████████████ 10.1ms
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 9.21 MB/s | 103.9us | 116.3us | 137.2us | 0 |
| minio | 1.96 MB/s | 478.0us | 611.3us | 709.2us | 0 |
| rustfs | 1.94 MB/s | 487.5us | 588.0us | 733.0us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 9.21 MB/s
minio        ██████ 1.96 MB/s
rustfs       ██████ 1.94 MB/s
```

**Latency (P50)**
```
liteio       ██████ 103.9us
minio        █████████████████████████████ 478.0us
rustfs       ██████████████████████████████ 487.5us
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 4.03 MB/s | 224.9us | 415.0us | 641.0us | 0 |
| minio | 0.56 MB/s | 1.5ms | 3.0ms | 6.4ms | 0 |
| rustfs | 0.41 MB/s | 2.3ms | 3.9ms | 5.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4.03 MB/s
minio        ████ 0.56 MB/s
rustfs       ███ 0.41 MB/s
```

**Latency (P50)**
```
liteio       ██ 224.9us
minio        ████████████████████ 1.5ms
rustfs       ██████████████████████████████ 2.3ms
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.03 MB/s | 698.6us | 2.5ms | 3.9ms | 0 |
| minio | 0.11 MB/s | 6.0ms | 27.6ms | 58.0ms | 0 |
| rustfs | 0.09 MB/s | 10.8ms | 14.6ms | 17.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.03 MB/s
minio        ███ 0.11 MB/s
rustfs       ██ 0.09 MB/s
```

**Latency (P50)**
```
liteio       █ 698.6us
minio        ████████████████ 6.0ms
rustfs       ██████████████████████████████ 10.8ms
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 3015.62 MB/s | 79.9us | 96.4us | 123.1us | 0 |
| minio | 551.21 MB/s | 443.1us | 499.5us | 765.2us | 0 |
| rustfs | 352.07 MB/s | 645.9us | 1.3ms | 1.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3015.62 MB/s
minio        █████ 551.21 MB/s
rustfs       ███ 352.07 MB/s
```

**Latency (P50)**
```
liteio       ███ 79.9us
minio        ████████████████████ 443.1us
rustfs       ██████████████████████████████ 645.9us
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 2817.83 MB/s | 83.1us | 115.2us | 151.8us | 0 |
| minio | 556.92 MB/s | 426.1us | 499.0us | 722.0us | 0 |
| rustfs | 404.80 MB/s | 607.0us | 712.0us | 858.5us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 2817.83 MB/s
minio        █████ 556.92 MB/s
rustfs       ████ 404.80 MB/s
```

**Latency (P50)**
```
liteio       ████ 83.1us
minio        █████████████████████ 426.1us
rustfs       ██████████████████████████████ 607.0us
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 2702.07 MB/s | 84.0us | 128.1us | 218.8us | 0 |
| minio | 627.80 MB/s | 372.2us | 456.1us | 742.0us | 0 |
| rustfs | 400.15 MB/s | 612.7us | 727.0us | 816.9us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 2702.07 MB/s
minio        ██████ 627.80 MB/s
rustfs       ████ 400.15 MB/s
```

**Latency (P50)**
```
liteio       ████ 84.0us
minio        ██████████████████ 372.2us
rustfs       ██████████████████████████████ 612.7us
```

### Read/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 8166.70 MB/s | 1.1ms | 1.7ms | 2.3ms | 0 |
| minio | 3274.67 MB/s | 3.0ms | 3.4ms | 3.6ms | 0 |
| rustfs | 1812.17 MB/s | 5.5ms | 6.1ms | 6.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 8166.70 MB/s
minio        ████████████ 3274.67 MB/s
rustfs       ██████ 1812.17 MB/s
```

**Latency (P50)**
```
liteio       ██████ 1.1ms
minio        ████████████████ 3.0ms
rustfs       ██████████████████████████████ 5.5ms
```

### Read/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 15.17 MB/s | 62.8us | 74.3us | 90.9us | 0 |
| minio | 5.37 MB/s | 179.2us | 206.2us | 226.3us | 0 |
| rustfs | 4.55 MB/s | 210.0us | 249.8us | 269.8us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 15.17 MB/s
minio        ██████████ 5.37 MB/s
rustfs       ████████ 4.55 MB/s
```

**Latency (P50)**
```
liteio       ████████ 62.8us
minio        █████████████████████████ 179.2us
rustfs       ██████████████████████████████ 210.0us
```

### Read/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 5501.43 MB/s | 174.8us | 234.6us | 287.0us | 0 |
| minio | 1613.27 MB/s | 607.6us | 718.6us | 938.2us | 0 |
| rustfs | 1201.32 MB/s | 818.9us | 963.6us | 1.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 5501.43 MB/s
minio        ████████ 1613.27 MB/s
rustfs       ██████ 1201.32 MB/s
```

**Latency (P50)**
```
liteio       ██████ 174.8us
minio        ██████████████████████ 607.6us
rustfs       ██████████████████████████████ 818.9us
```

### Read/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 824.09 MB/s | 68.8us | 118.7us | 217.0us | 0 |
| minio | 326.61 MB/s | 186.3us | 212.2us | 286.0us | 0 |
| rustfs | 257.99 MB/s | 239.5us | 264.3us | 290.4us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 824.09 MB/s
minio        ███████████ 326.61 MB/s
rustfs       █████████ 257.99 MB/s
```

**Latency (P50)**
```
liteio       ████████ 68.8us
minio        ███████████████████████ 186.3us
rustfs       ██████████████████████████████ 239.5us
```

### Scale/Delete/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 15000 ops/s | 66.7us | 66.7us | 66.7us | 0 |
| rustfs | 1588 ops/s | 629.8us | 629.8us | 629.8us | 0 |
| minio | 671 ops/s | 1.5ms | 1.5ms | 1.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 15000 ops/s
rustfs       ███ 1588 ops/s
minio        █ 671 ops/s
```

**Latency (P50)**
```
liteio       █ 66.7us
rustfs       ████████████ 629.8us
minio        ██████████████████████████████ 1.5ms
```

### Scale/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1805 ops/s | 554.1us | 554.1us | 554.1us | 0 |
| minio | 249 ops/s | 4.0ms | 4.0ms | 4.0ms | 0 |
| rustfs | 166 ops/s | 6.0ms | 6.0ms | 6.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1805 ops/s
minio        ████ 249 ops/s
rustfs       ██ 166 ops/s
```

**Latency (P50)**
```
liteio       ██ 554.1us
minio        ████████████████████ 4.0ms
rustfs       ██████████████████████████████ 6.0ms
```

### Scale/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 191 ops/s | 5.2ms | 5.2ms | 5.2ms | 0 |
| minio | 24 ops/s | 42.5ms | 42.5ms | 42.5ms | 0 |
| rustfs | 16 ops/s | 63.2ms | 63.2ms | 63.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 191 ops/s
minio        ███ 24 ops/s
rustfs       ██ 16 ops/s
```

**Latency (P50)**
```
liteio       ██ 5.2ms
minio        ████████████████████ 42.5ms
rustfs       ██████████████████████████████ 63.2ms
```

### Scale/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 19 ops/s | 53.1ms | 53.1ms | 53.1ms | 0 |
| minio | 2 ops/s | 442.8ms | 442.8ms | 442.8ms | 0 |
| rustfs | 2 ops/s | 641.1ms | 641.1ms | 641.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 19 ops/s
minio        ███ 2 ops/s
rustfs       ██ 2 ops/s
```

**Latency (P50)**
```
liteio       ██ 53.1ms
minio        ████████████████████ 442.8ms
rustfs       ██████████████████████████████ 641.1ms
```

### Scale/List/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 10426 ops/s | 95.9us | 95.9us | 95.9us | 0 |
| rustfs | 2147 ops/s | 465.8us | 465.8us | 465.8us | 0 |
| minio | 1892 ops/s | 528.5us | 528.5us | 528.5us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 10426 ops/s
rustfs       ██████ 2147 ops/s
minio        █████ 1892 ops/s
```

**Latency (P50)**
```
liteio       █████ 95.9us
rustfs       ██████████████████████████ 465.8us
minio        ██████████████████████████████ 528.5us
```

### Scale/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 8667 ops/s | 115.4us | 115.4us | 115.4us | 0 |
| minio | 1497 ops/s | 668.2us | 668.2us | 668.2us | 0 |
| rustfs | 902 ops/s | 1.1ms | 1.1ms | 1.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 8667 ops/s
minio        █████ 1497 ops/s
rustfs       ███ 902 ops/s
```

**Latency (P50)**
```
liteio       ███ 115.4us
minio        ██████████████████ 668.2us
rustfs       ██████████████████████████████ 1.1ms
```

### Scale/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 2146 ops/s | 466.0us | 466.0us | 466.0us | 0 |
| minio | 330 ops/s | 3.0ms | 3.0ms | 3.0ms | 0 |
| rustfs | 224 ops/s | 4.5ms | 4.5ms | 4.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 2146 ops/s
minio        ████ 330 ops/s
rustfs       ███ 224 ops/s
```

**Latency (P50)**
```
liteio       ███ 466.0us
minio        ████████████████████ 3.0ms
rustfs       ██████████████████████████████ 4.5ms
```

### Scale/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 284 ops/s | 3.5ms | 3.5ms | 3.5ms | 0 |
| minio | 39 ops/s | 25.9ms | 25.9ms | 25.9ms | 0 |
| rustfs | 26 ops/s | 37.9ms | 37.9ms | 37.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 284 ops/s
minio        ████ 39 ops/s
rustfs       ██ 26 ops/s
```

**Latency (P50)**
```
liteio       ██ 3.5ms
minio        ████████████████████ 25.9ms
rustfs       ██████████████████████████████ 37.9ms
```

### Scale/Write/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 3.90 MB/s | 62.6us | 62.6us | 62.6us | 0 |
| minio | 0.49 MB/s | 501.2us | 501.2us | 501.2us | 0 |
| rustfs | 0.47 MB/s | 522.2us | 522.2us | 522.2us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3.90 MB/s
minio        ███ 0.49 MB/s
rustfs       ███ 0.47 MB/s
```

**Latency (P50)**
```
liteio       ███ 62.6us
minio        ████████████████████████████ 501.2us
rustfs       ██████████████████████████████ 522.2us
```

### Scale/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 3.86 MB/s | 632.6us | 632.6us | 632.6us | 0 |
| minio | 0.53 MB/s | 4.6ms | 4.6ms | 4.6ms | 0 |
| rustfs | 0.45 MB/s | 5.4ms | 5.4ms | 5.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3.86 MB/s
minio        ████ 0.53 MB/s
rustfs       ███ 0.45 MB/s
```

**Latency (P50)**
```
liteio       ███ 632.6us
minio        █████████████████████████ 4.6ms
rustfs       ██████████████████████████████ 5.4ms
```

### Scale/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 3.93 MB/s | 6.2ms | 6.2ms | 6.2ms | 0 |
| minio | 0.52 MB/s | 46.8ms | 46.8ms | 46.8ms | 0 |
| rustfs | 0.51 MB/s | 47.5ms | 47.5ms | 47.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3.93 MB/s
minio        ███ 0.52 MB/s
rustfs       ███ 0.51 MB/s
```

**Latency (P50)**
```
liteio       ███ 6.2ms
minio        █████████████████████████████ 46.8ms
rustfs       ██████████████████████████████ 47.5ms
```

### Scale/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 3.94 MB/s | 62.0ms | 62.0ms | 62.0ms | 0 |
| minio | 0.50 MB/s | 483.8ms | 483.8ms | 483.8ms | 0 |
| rustfs | 0.41 MB/s | 594.4ms | 594.4ms | 594.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3.94 MB/s
minio        ███ 0.50 MB/s
rustfs       ███ 0.41 MB/s
```

**Latency (P50)**
```
liteio       ███ 62.0ms
minio        ████████████████████████ 483.8ms
rustfs       ██████████████████████████████ 594.4ms
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 16592 ops/s | 58.7us | 68.6us | 81.8us | 0 |
| minio | 6377 ops/s | 154.5us | 178.2us | 200.5us | 0 |
| rustfs | 6009 ops/s | 162.0us | 191.0us | 233.0us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 16592 ops/s
minio        ███████████ 6377 ops/s
rustfs       ██████████ 6009 ops/s
```

**Latency (P50)**
```
liteio       ██████████ 58.7us
minio        ████████████████████████████ 154.5us
rustfs       ██████████████████████████████ 162.0us
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1341.70 MB/s | 7.5ms | 8.9ms | 9.1ms | 0 |
| minio | 395.13 MB/s | 25.4ms | 25.7ms | 26.2ms | 0 |
| rustfs | 343.77 MB/s | 28.7ms | 30.9ms | 30.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1341.70 MB/s
minio        ████████ 395.13 MB/s
rustfs       ███████ 343.77 MB/s
```

**Latency (P50)**
```
liteio       ███████ 7.5ms
minio        ██████████████████████████ 25.4ms
rustfs       ██████████████████████████████ 28.7ms
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 13.21 MB/s | 71.0us | 89.1us | 152.0us | 0 |
| minio | 2.13 MB/s | 443.1us | 563.4us | 604.1us | 0 |
| rustfs | 2.12 MB/s | 432.9us | 536.2us | 616.0us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 13.21 MB/s
minio        ████ 2.13 MB/s
rustfs       ████ 2.12 MB/s
```

**Latency (P50)**
```
liteio       ████ 71.0us
minio        ██████████████████████████████ 443.1us
rustfs       █████████████████████████████ 432.9us
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1308.02 MB/s | 734.6us | 1.0ms | 1.1ms | 0 |
| rustfs | 262.51 MB/s | 3.7ms | 4.1ms | 4.4ms | 0 |
| minio | 258.93 MB/s | 3.7ms | 5.3ms | 6.8ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1308.02 MB/s
rustfs       ██████ 262.51 MB/s
minio        █████ 258.93 MB/s
```

**Latency (P50)**
```
liteio       █████ 734.6us
rustfs       ██████████████████████████████ 3.7ms
minio        █████████████████████████████ 3.7ms
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 583.74 MB/s | 104.2us | 123.1us | 151.5us | 0 |
| rustfs | 96.90 MB/s | 597.2us | 901.9us | 1.1ms | 0 |
| minio | 92.03 MB/s | 631.7us | 943.3us | 1.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 583.74 MB/s
rustfs       ████ 96.90 MB/s
minio        ████ 92.03 MB/s
```

**Latency (P50)**
```
liteio       ████ 104.2us
rustfs       ████████████████████████████ 597.2us
minio        ██████████████████████████████ 631.7us
```

## Recommendations

- **Write-heavy workloads:** liteio
- **Read-heavy workloads:** liteio

---

*Generated by storage benchmark CLI*
