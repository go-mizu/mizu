# Storage Benchmark Report

**Generated:** 2026-01-22T18:26:51+07:00

**Go Version:** go1.25.6

**Platform:** darwin/arm64

## Executive Summary

### Summary

**Overall Winner:** usagi (won 43/48 benchmarks, 90%)

| Rank | Driver | Wins | Win Rate |
|------|--------|------|----------|
| 1 | usagi | 43 | 90% |
| 2 | usagi_s3 | 4 | 8% |
| 3 | rustfs | 1 | 2% |

### Performance Leaders

| Operation | Leader | Performance | Margin |
|-----------|--------|-------------|--------|
| Small Read (1KB) | usagi | 3.4 GB/s | 941.2x vs usagi_s3 |
| Small Write (1KB) | usagi | 231.7 MB/s | 146.6x vs rustfs |
| Large Read (100MB) | usagi | 6.5 GB/s | 30.1x vs rustfs |
| Large Write (100MB) | usagi | 483.5 MB/s | 2.9x vs rustfs |
| Delete | usagi | 210.6K ops/s | 45.6x vs usagi_s3 |
| Stat | usagi | 4.9M ops/s | 1264.8x vs usagi_s3 |
| List (100 objects) | usagi | 61.7K ops/s | 61.3x vs usagi_s3 |
| Copy | usagi | 94.4 MB/s | 92.5x vs usagi_s3 |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (100MB+) | **usagi** | 483 MB/s | Best for media, backups |
| Large File Downloads (100MB) | **usagi** | 6533 MB/s | Best for streaming, CDN |
| Small File Operations | **usagi** | 1842877 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **usagi** | - | Best for multi-user apps |
| Memory Constrained | **usagi_s3** | 781 MB RAM | Best for edge/embedded |

### Large File Performance (100MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| rustfs | 164.4 | 216.8 | 632.8ms | 429.1ms |
| usagi | 483.5 | 6532.7 | 228.5ms | 12.8ms |
| usagi_s3 | 148.5 | 195.5 | 682.3ms | 503.0ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| rustfs | 1619 | 2522 | 586.5us | 381.2us |
| usagi | 237244 | 3448510 | 2.5us | 208ns |
| usagi_s3 | 1082 | 3664 | 745.3us | 226.6us |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| rustfs | 3426 | 193 | 1563 |
| usagi | 4893524 | 61720 | 210637 |
| usagi_s3 | 3869 | 1007 | 4621 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| rustfs | 1.39 | 0.45 | 0.19 | 0.09 | 0.05 | 0.01 |
| usagi | 93.11 | 21.51 | 0.00 | 4.05 | 2.31 | 0.78 |
| usagi_s3 | 1.15 | 0.36 | 0.18 | 0.08 | 0.06 | 0.02 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| rustfs | 2.07 | 0.87 | 0.36 | 0.15 | 0.10 | 0.04 |
| usagi | 713.91 | 81.76 | 28.41 | 14.48 | 7.04 | 4.09 |
| usagi_s3 | 4.03 | 1.29 | 0.56 | 0.34 | 0.16 | 0.09 |

*\* indicates errors occurred*

### Scale Performance

Performance with varying numbers of objects (256B each).

**Write N Files (total time)**

| Driver | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|
| rustfs | 5.4ms | 62.3ms | 690.8ms | 6.72s |
| usagi | 162.1us | 1.0ms | 11.3ms | 77.5ms |
| usagi_s3 | 8.7ms | 81.3ms | 805.0ms | 7.26s |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|
| rustfs | 2.9ms | 6.9ms | 49.8ms | 632.6ms |
| usagi | 1.04s | 1.32s | 1.17s | 945.2ms |
| usagi_s3 | 1.3ms | 1.2ms | 7.7ms | 221.6ms |

*\* indicates errors occurred*

### Resource Usage Summary

| Driver | Memory | CPU |
|--------|--------|-----|
| rustfs | 2167.8 MB | 0.1% |
| usagi_s3 | 781.2 MB | 1.7% |

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

- **rustfs** (48 benchmarks)
- **usagi** (48 benchmarks)
- **usagi_s3** (48 benchmarks)

## Detailed Results

### Copy/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 94.36 MB/s | 4.8us | 16.6us | 42.9us | 0 |
| usagi_s3 | 1.02 MB/s | 759.3us | 1.7ms | 3.6ms | 0 |
| rustfs | 0.88 MB/s | 1.0ms | 1.7ms | 3.0ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 94.36 MB/s
usagi_s3     █ 1.02 MB/s
rustfs       █ 0.88 MB/s
```

**Latency (P50)**
```
usagi        █ 4.8us
usagi_s3     ██████████████████████ 759.3us
rustfs       ██████████████████████████████ 1.0ms
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 210637 ops/s | 3.8us | 5.8us | 21.3us | 0 |
| usagi_s3 | 4621 ops/s | 197.4us | 323.8us | 434.1us | 0 |
| rustfs | 1563 ops/s | 587.6us | 772.2us | 1.1ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 210637 ops/s
usagi_s3     █ 4621 ops/s
rustfs       █ 1563 ops/s
```

**Latency (P50)**
```
usagi        █ 3.8us
usagi_s3     ██████████ 197.4us
rustfs       ██████████████████████████████ 587.6us
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 1.83 MB/s | 3.2us | 12.3us | 28.5us | 0 |
| rustfs | 0.15 MB/s | 608.2us | 797.9us | 973.5us | 0 |
| usagi_s3 | 0.14 MB/s | 639.1us | 903.8us | 1.2ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 1.83 MB/s
rustfs       ██ 0.15 MB/s
usagi_s3     ██ 0.14 MB/s
```

**Latency (P50)**
```
usagi        █ 3.2us
rustfs       ████████████████████████████ 608.2us
usagi_s3     ██████████████████████████████ 639.1us
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 276989 ops/s | 2.7us | 3.5us | 17.7us | 0 |
| rustfs | 1550 ops/s | 612.2us | 890.0us | 1.2ms | 0 |
| usagi_s3 | 1472 ops/s | 608.2us | 1.0ms | 1.6ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 276989 ops/s
rustfs       █ 1550 ops/s
usagi_s3     █ 1472 ops/s
```

**Latency (P50)**
```
usagi        █ 2.7us
rustfs       ██████████████████████████████ 612.2us
usagi_s3     █████████████████████████████ 608.2us
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 15.50 MB/s | 3.5us | 20.2us | 34.5us | 0 |
| rustfs | 0.14 MB/s | 629.4us | 1.1ms | 1.9ms | 0 |
| usagi_s3 | 0.13 MB/s | 671.5us | 1.2ms | 2.3ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 15.50 MB/s
rustfs       █ 0.14 MB/s
usagi_s3     █ 0.13 MB/s
```

**Latency (P50)**
```
usagi        █ 3.5us
rustfs       ████████████████████████████ 629.4us
usagi_s3     ██████████████████████████████ 671.5us
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 61720 ops/s | 11.7us | 24.0us | 79.8us | 0 |
| usagi_s3 | 1007 ops/s | 936.7us | 1.3ms | 2.2ms | 0 |
| rustfs | 193 ops/s | 4.9ms | 6.5ms | 8.2ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 61720 ops/s
usagi_s3     █ 1007 ops/s
rustfs       █ 193 ops/s
```

**Latency (P50)**
```
usagi        █ 11.7us
usagi_s3     █████ 936.7us
rustfs       ██████████████████████████████ 4.9ms
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 3.06 MB/s | 846.6us | 22.8ms | 46.6ms | 0 |
| usagi_s3 | 0.55 MB/s | 22.9ms | 59.6ms | 111.9ms | 0 |
| rustfs | 0.33 MB/s | 38.7ms | 70.7ms | 291.5ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 3.06 MB/s
usagi_s3     █████ 0.55 MB/s
rustfs       ███ 0.33 MB/s
```

**Latency (P50)**
```
usagi        █ 846.6us
usagi_s3     █████████████████ 22.9ms
rustfs       ██████████████████████████████ 38.7ms
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 8.71 MB/s | 184.1us | 6.2ms | 26.0ms | 0 |
| usagi_s3 | 0.70 MB/s | 21.8ms | 31.6ms | 41.1ms | 0 |
| rustfs | 0.45 MB/s | 32.8ms | 57.9ms | 73.4ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 8.71 MB/s
usagi_s3     ██ 0.70 MB/s
rustfs       █ 0.45 MB/s
```

**Latency (P50)**
```
usagi        █ 184.1us
usagi_s3     ███████████████████ 21.8ms
rustfs       ██████████████████████████████ 32.8ms
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 1.84 MB/s | 5.2ms | 26.5ms | 69.5ms | 0 |
| usagi_s3 | 0.31 MB/s | 47.9ms | 84.5ms | 121.4ms | 0 |
| rustfs | 0.29 MB/s | 47.3ms | 60.7ms | 318.0ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 1.84 MB/s
usagi_s3     █████ 0.31 MB/s
rustfs       ████ 0.29 MB/s
```

**Latency (P50)**
```
usagi        ███ 5.2ms
usagi_s3     ██████████████████████████████ 47.9ms
rustfs       █████████████████████████████ 47.3ms
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 190.20 MB/s | 74.0ms | 115.8ms | 115.8ms | 0 |
| usagi_s3 | 109.99 MB/s | 128.5ms | 139.4ms | 139.4ms | 0 |
| rustfs | 98.92 MB/s | 124.2ms | 232.4ms | 232.4ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 190.20 MB/s
usagi_s3     █████████████████ 109.99 MB/s
rustfs       ███████████████ 98.92 MB/s
```

**Latency (P50)**
```
usagi        █████████████████ 74.0ms
usagi_s3     ██████████████████████████████ 128.5ms
rustfs       ████████████████████████████ 124.2ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 713.91 MB/s | 541ns | 2.8us | 5.6us | 0 |
| usagi_s3 | 4.03 MB/s | 227.1us | 344.5us | 446.1us | 0 |
| rustfs | 2.07 MB/s | 417.2us | 692.4us | 1.6ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 713.91 MB/s
usagi_s3     █ 4.03 MB/s
rustfs       █ 2.07 MB/s
```

**Latency (P50)**
```
usagi        █ 541ns
usagi_s3     ████████████████ 227.1us
rustfs       ██████████████████████████████ 417.2us
```

### ParallelRead/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 81.76 MB/s | 792ns | 54.5us | 123.6us | 0 |
| usagi_s3 | 1.29 MB/s | 716.5us | 1.2ms | 1.7ms | 0 |
| rustfs | 0.87 MB/s | 1.1ms | 1.6ms | 1.9ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 81.76 MB/s
usagi_s3     █ 1.29 MB/s
rustfs       █ 0.87 MB/s
```

**Latency (P50)**
```
usagi        █ 792ns
usagi_s3     ███████████████████ 716.5us
rustfs       ██████████████████████████████ 1.1ms
```

### ParallelRead/1KB/C100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 7.04 MB/s | 1.2us | 657.6us | 1.3ms | 0 |
| usagi_s3 | 0.16 MB/s | 5.2ms | 10.8ms | 21.7ms | 0 |
| rustfs | 0.10 MB/s | 9.9ms | 13.7ms | 15.1ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 7.04 MB/s
usagi_s3     █ 0.16 MB/s
rustfs       █ 0.10 MB/s
```

**Latency (P50)**
```
usagi        █ 1.2us
usagi_s3     ███████████████ 5.2ms
rustfs       ██████████████████████████████ 9.9ms
```

### ParallelRead/1KB/C200

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 4.09 MB/s | 1.2us | 1.2ms | 2.5ms | 0 |
| usagi_s3 | 0.09 MB/s | 10.9ms | 16.0ms | 23.3ms | 0 |
| rustfs | 0.04 MB/s | 22.3ms | 27.8ms | 32.1ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 4.09 MB/s
usagi_s3     █ 0.09 MB/s
rustfs       █ 0.04 MB/s
```

**Latency (P50)**
```
usagi        █ 1.2us
usagi_s3     ██████████████ 10.9ms
rustfs       ██████████████████████████████ 22.3ms
```

### ParallelRead/1KB/C25

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 28.41 MB/s | 958ns | 176.0us | 387.0us | 0 |
| usagi_s3 | 0.56 MB/s | 1.6ms | 3.1ms | 4.6ms | 0 |
| rustfs | 0.36 MB/s | 2.6ms | 4.4ms | 6.1ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 28.41 MB/s
usagi_s3     █ 0.56 MB/s
rustfs       █ 0.36 MB/s
```

**Latency (P50)**
```
usagi        █ 958ns
usagi_s3     ██████████████████ 1.6ms
rustfs       ██████████████████████████████ 2.6ms
```

### ParallelRead/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 14.48 MB/s | 1.1us | 317.0us | 665.0us | 0 |
| usagi_s3 | 0.34 MB/s | 2.6ms | 5.3ms | 9.2ms | 0 |
| rustfs | 0.15 MB/s | 6.2ms | 11.0ms | 15.5ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 14.48 MB/s
usagi_s3     █ 0.34 MB/s
rustfs       █ 0.15 MB/s
```

**Latency (P50)**
```
usagi        █ 1.1us
usagi_s3     ████████████ 2.6ms
rustfs       ██████████████████████████████ 6.2ms
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 93.11 MB/s | 6.6us | 16.9us | 56.6us | 0 |
| rustfs | 1.39 MB/s | 661.4us | 974.0us | 1.5ms | 0 |
| usagi_s3 | 1.15 MB/s | 769.5us | 1.3ms | 2.3ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 93.11 MB/s
rustfs       █ 1.39 MB/s
usagi_s3     █ 1.15 MB/s
```

**Latency (P50)**
```
usagi        █ 6.6us
rustfs       █████████████████████████ 661.4us
usagi_s3     ██████████████████████████████ 769.5us
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 21.51 MB/s | 14.5us | 89.5us | 230.6us | 0 |
| rustfs | 0.45 MB/s | 1.9ms | 3.5ms | 6.1ms | 0 |
| usagi_s3 | 0.36 MB/s | 2.4ms | 4.4ms | 9.3ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 21.51 MB/s
rustfs       █ 0.45 MB/s
usagi_s3     █ 0.36 MB/s
```

**Latency (P50)**
```
usagi        █ 14.5us
rustfs       ████████████████████████ 1.9ms
usagi_s3     ██████████████████████████████ 2.4ms
```

### ParallelWrite/1KB/C100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 2.31 MB/s | 19.2us | 1.2ms | 9.0ms | 0 |
| usagi_s3 | 0.06 MB/s | 15.5ms | 34.4ms | 53.5ms | 0 |
| rustfs | 0.05 MB/s | 19.0ms | 31.4ms | 80.0ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 2.31 MB/s
usagi_s3     █ 0.06 MB/s
rustfs       █ 0.05 MB/s
```

**Latency (P50)**
```
usagi        █ 19.2us
usagi_s3     ████████████████████████ 15.5ms
rustfs       ██████████████████████████████ 19.0ms
```

### ParallelWrite/1KB/C200

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 0.78 MB/s | 294.9us | 4.5ms | 13.2ms | 0 |
| usagi_s3 | 0.02 MB/s | 35.3ms | 86.2ms | 105.1ms | 0 |
| rustfs | 0.01 MB/s | 37.0ms | 53.3ms | 968.4ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 0.78 MB/s
usagi_s3     █ 0.02 MB/s
rustfs       █ 0.01 MB/s
```

**Latency (P50)**
```
usagi        █ 294.9us
usagi_s3     ████████████████████████████ 35.3ms
rustfs       ██████████████████████████████ 37.0ms
```

### ParallelWrite/1KB/C25

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 0.19 MB/s | 4.5ms | 9.0ms | 12.6ms | 0 |
| usagi_s3 | 0.18 MB/s | 4.9ms | 9.7ms | 15.5ms | 0 |
| usagi | 0.00 MB/s | 224.3us | 224.3us | 224.3us | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 0.19 MB/s
usagi_s3     ████████████████████████████ 0.18 MB/s
usagi        █ 0.00 MB/s
```

**Latency (P50)**
```
rustfs       ████████████████████████████ 4.5ms
usagi_s3     ██████████████████████████████ 4.9ms
usagi        █ 224.3us
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 4.05 MB/s | 16.2us | 630.7us | 2.4ms | 0 |
| rustfs | 0.09 MB/s | 10.1ms | 16.3ms | 20.4ms | 0 |
| usagi_s3 | 0.08 MB/s | 10.4ms | 31.2ms | 44.5ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 4.05 MB/s
rustfs       █ 0.09 MB/s
usagi_s3     █ 0.08 MB/s
```

**Latency (P50)**
```
usagi        █ 16.2us
rustfs       █████████████████████████████ 10.1ms
usagi_s3     ██████████████████████████████ 10.4ms
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 10601.18 MB/s | 20.7us | 34.5us | 69.0us | 0 |
| usagi_s3 | 151.79 MB/s | 1.5ms | 2.0ms | 3.8ms | 0 |
| rustfs | 79.33 MB/s | 2.8ms | 4.8ms | 7.1ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 10601.18 MB/s
usagi_s3     █ 151.79 MB/s
rustfs       █ 79.33 MB/s
```

**Latency (P50)**
```
usagi        █ 20.7us
usagi_s3     ████████████████ 1.5ms
rustfs       ██████████████████████████████ 2.8ms
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 10690.63 MB/s | 20.8us | 33.2us | 62.2us | 0 |
| usagi_s3 | 144.70 MB/s | 1.6ms | 2.3ms | 2.9ms | 0 |
| rustfs | 76.80 MB/s | 3.0ms | 5.2ms | 7.0ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 10690.63 MB/s
usagi_s3     █ 144.70 MB/s
rustfs       █ 76.80 MB/s
```

**Latency (P50)**
```
usagi        █ 20.8us
usagi_s3     ████████████████ 1.6ms
rustfs       ██████████████████████████████ 3.0ms
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 8840.59 MB/s | 22.2us | 48.5us | 127.0us | 0 |
| usagi_s3 | 136.99 MB/s | 1.7ms | 2.7ms | 3.3ms | 0 |
| rustfs | 79.77 MB/s | 2.9ms | 4.7ms | 6.3ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 8840.59 MB/s
usagi_s3     █ 136.99 MB/s
rustfs       █ 79.77 MB/s
```

**Latency (P50)**
```
usagi        █ 22.2us
usagi_s3     █████████████████ 1.7ms
rustfs       ██████████████████████████████ 2.9ms
```

### Read/100MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 6532.68 MB/s | 12.8ms | 30.5ms | 45.4ms | 0 |
| rustfs | 216.78 MB/s | 429.1ms | 429.1ms | 429.1ms | 0 |
| usagi_s3 | 195.54 MB/s | 503.0ms | 503.0ms | 503.0ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 6532.68 MB/s
rustfs       █ 216.78 MB/s
usagi_s3     █ 195.54 MB/s
```

**Latency (P50)**
```
usagi        █ 12.8ms
rustfs       █████████████████████████ 429.1ms
usagi_s3     ██████████████████████████████ 503.0ms
```

### Read/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 8859.27 MB/s | 1.1ms | 1.4ms | 2.1ms | 0 |
| rustfs | 220.63 MB/s | 44.3ms | 51.8ms | 51.9ms | 0 |
| usagi_s3 | 172.16 MB/s | 51.1ms | 81.6ms | 82.5ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 8859.27 MB/s
rustfs       █ 220.63 MB/s
usagi_s3     █ 172.16 MB/s
```

**Latency (P50)**
```
usagi        █ 1.1ms
rustfs       ██████████████████████████ 44.3ms
usagi_s3     ██████████████████████████████ 51.1ms
```

### Read/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 3367.69 MB/s | 208ns | 500ns | 1.4us | 0 |
| usagi_s3 | 3.58 MB/s | 226.6us | 541.7us | 928.2us | 0 |
| rustfs | 2.46 MB/s | 381.2us | 498.0us | 619.7us | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 3367.69 MB/s
usagi_s3     █ 3.58 MB/s
rustfs       █ 2.46 MB/s
```

**Latency (P50)**
```
usagi        █ 208ns
usagi_s3     █████████████████ 226.6us
rustfs       ██████████████████████████████ 381.2us
```

### Read/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 8349.51 MB/s | 108.5us | 137.6us | 364.2us | 0 |
| usagi_s3 | 174.26 MB/s | 5.1ms | 9.3ms | 11.3ms | 0 |
| rustfs | 172.98 MB/s | 5.3ms | 8.1ms | 9.5ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 8349.51 MB/s
usagi_s3     █ 174.26 MB/s
rustfs       █ 172.98 MB/s
```

**Latency (P50)**
```
usagi        █ 108.5us
usagi_s3     ████████████████████████████ 5.1ms
rustfs       ██████████████████████████████ 5.3ms
```

### Read/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 16505.96 MB/s | 2.3us | 6.8us | 27.0us | 0 |
| usagi_s3 | 118.29 MB/s | 504.1us | 728.1us | 932.8us | 0 |
| rustfs | 76.48 MB/s | 708.0us | 1.3ms | 2.0ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 16505.96 MB/s
usagi_s3     █ 118.29 MB/s
rustfs       █ 76.48 MB/s
```

**Latency (P50)**
```
usagi        █ 2.3us
usagi_s3     █████████████████████ 504.1us
rustfs       ██████████████████████████████ 708.0us
```

### Scale/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 3966 ops/s | 252.2us | 252.2us | 252.2us | 0 |
| usagi_s3 | 125 ops/s | 8.0ms | 8.0ms | 8.0ms | 0 |
| rustfs | 103 ops/s | 9.7ms | 9.7ms | 9.7ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 3966 ops/s
usagi_s3     █ 125 ops/s
rustfs       █ 103 ops/s
```

**Latency (P50)**
```
usagi        █ 252.2us
usagi_s3     ████████████████████████ 8.0ms
rustfs       ██████████████████████████████ 9.7ms
```

### Scale/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 1355 ops/s | 737.9us | 737.9us | 737.9us | 0 |
| usagi_s3 | 45 ops/s | 22.3ms | 22.3ms | 22.3ms | 0 |
| rustfs | 14 ops/s | 73.1ms | 73.1ms | 73.1ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 1355 ops/s
usagi_s3     █ 45 ops/s
rustfs       █ 14 ops/s
```

**Latency (P50)**
```
usagi        █ 737.9us
usagi_s3     █████████ 22.3ms
rustfs       ██████████████████████████████ 73.1ms
```

### Scale/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 277 ops/s | 3.6ms | 3.6ms | 3.6ms | 0 |
| usagi_s3 | 4 ops/s | 228.7ms | 228.7ms | 228.7ms | 0 |
| rustfs | 1 ops/s | 835.8ms | 835.8ms | 835.8ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 277 ops/s
usagi_s3     █ 4 ops/s
rustfs       █ 1 ops/s
```

**Latency (P50)**
```
usagi        █ 3.6ms
usagi_s3     ████████ 228.7ms
rustfs       ██████████████████████████████ 835.8ms
```

### Scale/Delete/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 19 ops/s | 53.0ms | 53.0ms | 53.0ms | 0 |
| usagi_s3 | 0 ops/s | 2.74s | 2.74s | 2.74s | 0 |
| rustfs | 0 ops/s | 7.77s | 7.77s | 7.77s | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 19 ops/s
usagi_s3     █ 0 ops/s
rustfs       █ 0 ops/s
```

**Latency (P50)**
```
usagi        █ 53.0ms
usagi_s3     ██████████ 2.74s
rustfs       ██████████████████████████████ 7.77s
```

### Scale/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 744 ops/s | 1.3ms | 1.3ms | 1.3ms | 0 |
| rustfs | 347 ops/s | 2.9ms | 2.9ms | 2.9ms | 0 |
| usagi | 1 ops/s | 1.04s | 1.04s | 1.04s | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 744 ops/s
rustfs       █████████████ 347 ops/s
usagi        █ 1 ops/s
```

**Latency (P50)**
```
usagi_s3     █ 1.3ms
rustfs       █ 2.9ms
usagi        ██████████████████████████████ 1.04s
```

### Scale/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 842 ops/s | 1.2ms | 1.2ms | 1.2ms | 0 |
| rustfs | 145 ops/s | 6.9ms | 6.9ms | 6.9ms | 0 |
| usagi | 1 ops/s | 1.32s | 1.32s | 1.32s | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 842 ops/s
rustfs       █████ 145 ops/s
usagi        █ 1 ops/s
```

**Latency (P50)**
```
usagi_s3     █ 1.2ms
rustfs       █ 6.9ms
usagi        ██████████████████████████████ 1.32s
```

### Scale/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 130 ops/s | 7.7ms | 7.7ms | 7.7ms | 0 |
| rustfs | 20 ops/s | 49.8ms | 49.8ms | 49.8ms | 0 |
| usagi | 1 ops/s | 1.17s | 1.17s | 1.17s | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 130 ops/s
rustfs       ████ 20 ops/s
usagi        █ 1 ops/s
```

**Latency (P50)**
```
usagi_s3     █ 7.7ms
rustfs       █ 49.8ms
usagi        ██████████████████████████████ 1.17s
```

### Scale/List/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 5 ops/s | 221.6ms | 221.6ms | 221.6ms | 0 |
| rustfs | 2 ops/s | 632.6ms | 632.6ms | 632.6ms | 0 |
| usagi | 1 ops/s | 945.2ms | 945.2ms | 945.2ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 5 ops/s
rustfs       ██████████ 2 ops/s
usagi        ███████ 1 ops/s
```

**Latency (P50)**
```
usagi_s3     ███████ 221.6ms
rustfs       ████████████████████ 632.6ms
usagi        ██████████████████████████████ 945.2ms
```

### Scale/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 15.06 MB/s | 162.1us | 162.1us | 162.1us | 0 |
| rustfs | 0.45 MB/s | 5.4ms | 5.4ms | 5.4ms | 0 |
| usagi_s3 | 0.28 MB/s | 8.7ms | 8.7ms | 8.7ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 15.06 MB/s
rustfs       █ 0.45 MB/s
usagi_s3     █ 0.28 MB/s
```

**Latency (P50)**
```
usagi        █ 162.1us
rustfs       ██████████████████ 5.4ms
usagi_s3     ██████████████████████████████ 8.7ms
```

### Scale/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 24.28 MB/s | 1.0ms | 1.0ms | 1.0ms | 0 |
| rustfs | 0.39 MB/s | 62.3ms | 62.3ms | 62.3ms | 0 |
| usagi_s3 | 0.30 MB/s | 81.3ms | 81.3ms | 81.3ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 24.28 MB/s
rustfs       █ 0.39 MB/s
usagi_s3     █ 0.30 MB/s
```

**Latency (P50)**
```
usagi        █ 1.0ms
rustfs       ███████████████████████ 62.3ms
usagi_s3     ██████████████████████████████ 81.3ms
```

### Scale/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 21.68 MB/s | 11.3ms | 11.3ms | 11.3ms | 0 |
| rustfs | 0.35 MB/s | 690.8ms | 690.8ms | 690.8ms | 0 |
| usagi_s3 | 0.30 MB/s | 805.0ms | 805.0ms | 805.0ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 21.68 MB/s
rustfs       █ 0.35 MB/s
usagi_s3     █ 0.30 MB/s
```

**Latency (P50)**
```
usagi        █ 11.3ms
rustfs       █████████████████████████ 690.8ms
usagi_s3     ██████████████████████████████ 805.0ms
```

### Scale/Write/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 31.51 MB/s | 77.5ms | 77.5ms | 77.5ms | 0 |
| rustfs | 0.36 MB/s | 6.72s | 6.72s | 6.72s | 0 |
| usagi_s3 | 0.34 MB/s | 7.26s | 7.26s | 7.26s | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 31.51 MB/s
rustfs       █ 0.36 MB/s
usagi_s3     █ 0.34 MB/s
```

**Latency (P50)**
```
usagi        █ 77.5ms
rustfs       ███████████████████████████ 6.72s
usagi_s3     ██████████████████████████████ 7.26s
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 4893524 ops/s | 84ns | 125ns | 500ns | 0 |
| usagi_s3 | 3869 ops/s | 208.6us | 529.6us | 695.6us | 0 |
| rustfs | 3426 ops/s | 271.8us | 400.8us | 725.8us | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 4893524 ops/s
usagi_s3     █ 3869 ops/s
rustfs       █ 3426 ops/s
```

**Latency (P50)**
```
usagi        █ 84ns
usagi_s3     ███████████████████████ 208.6us
rustfs       ██████████████████████████████ 271.8us
```

### Write/100MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 483.50 MB/s | 228.5ms | 231.7ms | 231.7ms | 0 |
| rustfs | 164.38 MB/s | 632.8ms | 632.8ms | 632.8ms | 0 |
| usagi_s3 | 148.51 MB/s | 682.3ms | 682.3ms | 682.3ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 483.50 MB/s
rustfs       ██████████ 164.38 MB/s
usagi_s3     █████████ 148.51 MB/s
```

**Latency (P50)**
```
usagi        ██████████ 228.5ms
rustfs       ███████████████████████████ 632.8ms
usagi_s3     ██████████████████████████████ 682.3ms
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 398.53 MB/s | 14.5ms | 52.2ms | 61.5ms | 0 |
| rustfs | 169.85 MB/s | 56.6ms | 64.7ms | 66.5ms | 0 |
| usagi_s3 | 143.82 MB/s | 68.1ms | 83.5ms | 83.5ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 398.53 MB/s
rustfs       ████████████ 169.85 MB/s
usagi_s3     ██████████ 143.82 MB/s
```

**Latency (P50)**
```
usagi        ██████ 14.5ms
rustfs       ████████████████████████ 56.6ms
usagi_s3     ██████████████████████████████ 68.1ms
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 231.68 MB/s | 2.5us | 5.7us | 11.7us | 0 |
| rustfs | 1.58 MB/s | 586.5us | 819.3us | 1.2ms | 0 |
| usagi_s3 | 1.06 MB/s | 745.3us | 2.0ms | 2.7ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 231.68 MB/s
rustfs       █ 1.58 MB/s
usagi_s3     █ 1.06 MB/s
```

**Latency (P50)**
```
usagi        █ 2.5us
rustfs       ███████████████████████ 586.5us
usagi_s3     ██████████████████████████████ 745.3us
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 692.52 MB/s | 454.8us | 2.8ms | 13.8ms | 0 |
| rustfs | 137.60 MB/s | 6.2ms | 13.3ms | 16.5ms | 0 |
| usagi_s3 | 136.72 MB/s | 6.9ms | 10.6ms | 12.0ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 692.52 MB/s
rustfs       █████ 137.60 MB/s
usagi_s3     █████ 136.72 MB/s
```

**Latency (P50)**
```
usagi        █ 454.8us
rustfs       ██████████████████████████ 6.2ms
usagi_s3     ██████████████████████████████ 6.9ms
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi | 708.64 MB/s | 21.4us | 50.6us | 243.7us | 0 |
| rustfs | 55.36 MB/s | 986.3us | 1.6ms | 3.6ms | 0 |
| usagi_s3 | 45.88 MB/s | 1.2ms | 2.1ms | 3.4ms | 0 |

**Throughput**
```
usagi        ██████████████████████████████ 708.64 MB/s
rustfs       ██ 55.36 MB/s
usagi_s3     █ 45.88 MB/s
```

**Latency (P50)**
```
usagi        █ 21.4us
rustfs       ███████████████████████ 986.3us
usagi_s3     ██████████████████████████████ 1.2ms
```

## Resource Usage

| Driver | Memory | RSS | Cache | CPU | Volume | Block I/O |
|--------|--------|-----|-------|-----|--------|----------|
| rustfs | 2.117GiB / 7.653GiB | 2167.8 MB | - | 0.1% | 2210.0 MB | 39.4MB / 2.25GB |
| usagi_s3 | 781.1MiB / 7.653GiB | 781.1 MB | - | 1.7% | (no data) | 13.6MB / 2.04GB |

> **Note:** RSS = actual application memory. Cache = OS page cache (reclaimable).

## Recommendations

- **Write-heavy workloads:** usagi
- **Read-heavy workloads:** usagi

---

*Generated by storage benchmark CLI*
