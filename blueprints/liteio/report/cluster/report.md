# Storage Benchmark Report

**Generated:** 0001-01-01T00:00:00Z

**Go Version:** go1.26.0

**Platform:** darwin/arm64

## Executive Summary

### Performance Leaders

| Operation | Leader | Performance | Margin |
|-----------|--------|-------------|--------|
| Small Read (1KB) | herd_cluster | 6.2 MB/s |  |
| Small Write (1KB) | herd_cluster | 7.6 MB/s |  |
| Large Read (100MB) | herd_cluster | 1.3 GB/s |  |
| Large Write (100MB) | herd_cluster | 577.9 MB/s |  |
| Delete | herd_cluster | 5.3K ops/s |  |
| Stat | herd_cluster | 6.0K ops/s |  |
| List (100 objects) | herd_cluster | 1.6K ops/s |  |
| Copy | herd_cluster | 4.7 MB/s |  |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (100MB+) | **herd_cluster** | 578 MB/s | Best for media, backups |
| Large File Downloads (100MB) | **herd_cluster** | 1327 MB/s | Best for streaming, CDN |
| Small File Operations | **herd_cluster** | 7064 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **herd_cluster** | - | Best for multi-user apps |

### Large File Performance (100MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| herd_cluster | 577.9 | 1327.0 | 185.8ms | 84.4ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| herd_cluster | 7823 | 6304 | 119.9us | 134.5us |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| herd_cluster | 6016 | 1635 | 5269 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| herd_cluster | 4.80 | 1.30 | 0.75 | 0.47 | 0.29 | 0.13 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| herd_cluster | 5.48 | 1.85 | 1.06 | 0.65 | 0.38 | 0.20 |

*\* indicates errors occurred*

### Scale Performance

Performance with varying numbers of objects (256B each).

**Write N Files (total time)**

| Driver | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|
| herd_cluster | 1.3ms | 13.0ms | 167.3ms | 1.66s |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|
| herd_cluster | 298.2us | 716.8us | 5.1ms | 90.0ms |

*\* indicates errors occurred*

### Warnings

- **herd_cluster**: 78 errors during benchmarks

---

## Configuration

| Parameter | Value |
|-----------|-------|
| BenchTime | 1s |
| MinIterations | 3 |
| Warmup | 2 |
| Concurrency | 50 |
| Timeout | 30s |

## Drivers Tested

- **herd_cluster** (48 benchmarks)

## Detailed Results

### Copy/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 4.74 MB/s | 170.3us | 417.8us | 815.8us | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 4.74 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 170.3us
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 5269 ops/s | 156.3us | 383.0us | 560.8us | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 5269 ops/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 156.3us
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 0.67 MB/s | 134.7us | 202.2us | 313.3us | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 0.67 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 134.7us
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 7196 ops/s | 125.6us | 218.3us | 382.6us | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 7196 ops/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 125.6us
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 0.69 MB/s | 132.9us | 180.6us | 263.3us | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 0.69 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 132.9us
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 1635 ops/s | 593.3us | 726.5us | 1.0ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 1635 ops/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 593.3us
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 5.61 MB/s | 1.9ms | 7.7ms | 15.6ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 5.61 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 1.9ms
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 5.89 MB/s | 1.7ms | 7.3ms | 18.8ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 5.89 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 1.7ms
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 5.53 MB/s | 2.1ms | 7.3ms | 13.3ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 5.53 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 2.1ms
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 0.00 MB/s | 0ns | 0ns | 0ns | 78 |

**Throughput**
```
herd_cluster  0.00 MB/s
```

**Latency (P50)**
```
```

### ParallelRead/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 5.48 MB/s | 158.2us | 290.2us | 474.2us | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 5.48 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 158.2us
```

### ParallelRead/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 1.85 MB/s | 445.5us | 967.2us | 1.7ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 1.85 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 445.5us
```

### ParallelRead/1KB/C100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 0.38 MB/s | 1.7ms | 7.5ms | 14.2ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 0.38 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 1.7ms
```

### ParallelRead/1KB/C200

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 0.20 MB/s | 3.3ms | 14.0ms | 29.0ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 0.20 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 3.3ms
```

### ParallelRead/1KB/C25

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 1.06 MB/s | 759.5us | 1.9ms | 3.3ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 1.06 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 759.5us
```

### ParallelRead/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 0.65 MB/s | 1.1ms | 4.0ms | 7.2ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 0.65 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 1.1ms
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 4.80 MB/s | 189.5us | 296.0us | 435.8us | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 4.80 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 189.5us
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 1.30 MB/s | 582.1us | 1.6ms | 3.3ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 1.30 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 582.1us
```

### ParallelWrite/1KB/C100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 0.29 MB/s | 2.2ms | 10.0ms | 19.8ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 0.29 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 2.2ms
```

### ParallelWrite/1KB/C200

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 0.13 MB/s | 4.7ms | 24.0ms | 36.7ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 0.13 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 4.7ms
```

### ParallelWrite/1KB/C25

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 0.75 MB/s | 1.1ms | 2.7ms | 5.2ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 0.75 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 1.1ms
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 0.47 MB/s | 1.6ms | 5.1ms | 9.8ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 0.47 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 1.6ms
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 439.96 MB/s | 579.8us | 902.5us | 1.5ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 439.96 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 579.8us
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 402.99 MB/s | 603.8us | 1.2ms | 1.7ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 402.99 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 603.8us
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 357.95 MB/s | 669.7us | 1.3ms | 1.9ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 357.95 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 669.7us
```

### Read/100MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 1327.01 MB/s | 84.4ms | 103.5ms | 106.6ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 1327.01 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 84.4ms
```

### Read/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 1924.61 MB/s | 5.4ms | 7.6ms | 9.8ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 1924.61 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 5.4ms
```

### Read/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 6.16 MB/s | 134.5us | 276.7us | 469.0us | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 6.16 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 134.5us
```

### Read/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 1525.72 MB/s | 631.5us | 1.1ms | 1.8ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 1525.72 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 631.5us
```

### Read/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 445.38 MB/s | 126.7us | 215.6us | 338.2us | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 445.38 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 126.7us
```

### Scale/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 845 ops/s | 1.2ms | 1.2ms | 1.2ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 845 ops/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 1.2ms
```

### Scale/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 71 ops/s | 14.2ms | 14.2ms | 14.2ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 71 ops/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 14.2ms
```

### Scale/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 7 ops/s | 148.0ms | 148.0ms | 148.0ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 7 ops/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 148.0ms
```

### Scale/Delete/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 1 ops/s | 1.39s | 1.39s | 1.39s | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 1 ops/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 1.39s
```

### Scale/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 3353 ops/s | 298.2us | 298.2us | 298.2us | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 3353 ops/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 298.2us
```

### Scale/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 1395 ops/s | 716.8us | 716.8us | 716.8us | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 1395 ops/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 716.8us
```

### Scale/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 198 ops/s | 5.1ms | 5.1ms | 5.1ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 198 ops/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 5.1ms
```

### Scale/List/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 11 ops/s | 90.0ms | 90.0ms | 90.0ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 11 ops/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 90.0ms
```

### Scale/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 1.95 MB/s | 1.3ms | 1.3ms | 1.3ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 1.95 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 1.3ms
```

### Scale/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 1.88 MB/s | 13.0ms | 13.0ms | 13.0ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 1.88 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 13.0ms
```

### Scale/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 1.46 MB/s | 167.3ms | 167.3ms | 167.3ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 1.46 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 167.3ms
```

### Scale/Write/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 1.47 MB/s | 1.66s | 1.66s | 1.66s | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 1.47 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 1.66s
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 6016 ops/s | 143.9us | 300.2us | 462.3us | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 6016 ops/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 143.9us
```

### Write/100MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 577.86 MB/s | 185.8ms | 196.1ms | 196.1ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 577.86 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 185.8ms
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 604.53 MB/s | 16.9ms | 23.0ms | 26.4ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 604.53 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 16.9ms
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 7.64 MB/s | 119.9us | 181.5us | 299.8us | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 7.64 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 119.9us
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 700.57 MB/s | 1.4ms | 2.0ms | 2.5ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 700.57 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 1.4ms
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 295.87 MB/s | 194.6us | 339.7us | 596.6us | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 295.87 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 194.6us
```

## Recommendations

- **Write-heavy workloads:** herd_cluster
- **Read-heavy workloads:** herd_cluster

---

*Generated by storage benchmark CLI*
