# Storage Benchmark Report

**Generated:** 0001-01-01T00:00:00Z

**Go Version:** go1.26.0

**Platform:** darwin/arm64

## Executive Summary

### Performance Leaders

| Operation | Leader | Performance | Margin |
|-----------|--------|-------------|--------|
| Small Read (1KB) | herd_cluster | 10.1 MB/s |  |
| Small Write (1KB) | herd_cluster | 7.7 MB/s |  |
| Large Read (100MB) | herd_cluster | 2.3 GB/s |  |
| Large Write (100MB) | herd_cluster | 720.3 MB/s |  |
| Delete | herd_cluster | 9.5K ops/s |  |
| Stat | herd_cluster | 9.2K ops/s |  |
| List (100 objects) | herd_cluster | 1.9K ops/s |  |
| Copy | herd_cluster | 7.2 MB/s |  |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (100MB+) | **herd_cluster** | 720 MB/s | Best for media, backups |
| Large File Downloads (100MB) | **herd_cluster** | 2273 MB/s | Best for streaming, CDN |
| Small File Operations | **herd_cluster** | 9118 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **herd_cluster** | - | Best for multi-user apps |

### Large File Performance (100MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| herd_cluster | 720.3 | 2272.7 | 138.9ms | 44.7ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| herd_cluster | 7916 | 10320 | 122.5us | 93.0us |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| herd_cluster | 9249 | 1935 | 9513 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| herd_cluster | 6.24 | 2.33 | 1.20 | 0.72 | 0.45 | 0.28 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| herd_cluster | 7.38 | 3.02 | 1.48 | 0.93 | 0.55 | 0.33 |

*\* indicates errors occurred*

### Scale Performance

Performance with varying numbers of objects (256B each).

**Write N Files (total time)**

| Driver | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|
| herd_cluster | 1.3ms | 11.6ms | 137.7ms | 1.22s |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|
| herd_cluster | 262.5us | 691.7us | 4.0ms | 73.4ms |

*\* indicates errors occurred*

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
| herd_cluster | 7.17 MB/s | 132.9us | 168.5us | 246.4us | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 7.17 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 132.9us
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 9513 ops/s | 104.6us | 128.1us | 162.2us | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 9513 ops/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 104.6us
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 0.78 MB/s | 118.5us | 150.2us | 225.0us | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 0.78 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 118.5us
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 8359 ops/s | 116.1us | 151.4us | 236.1us | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 8359 ops/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 116.1us
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 0.73 MB/s | 124.2us | 153.0us | 221.9us | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 0.73 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 124.2us
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 1935 ops/s | 507.3us | 568.5us | 684.7us | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 1935 ops/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 507.3us
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 7.75 MB/s | 1.3ms | 4.9ms | 10.1ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 7.75 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 1.3ms
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 11.60 MB/s | 1.1ms | 3.4ms | 5.5ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 11.60 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 1.1ms
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 8.10 MB/s | 1.4ms | 5.2ms | 9.0ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 8.10 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 1.4ms
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 313.87 MB/s | 47.2ms | 54.4ms | 54.6ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 313.87 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 47.2ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 7.38 MB/s | 128.8us | 153.4us | 198.0us | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 7.38 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 128.8us
```

### ParallelRead/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 3.02 MB/s | 302.1us | 498.5us | 754.7us | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 3.02 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 302.1us
```

### ParallelRead/1KB/C100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 0.55 MB/s | 1.2ms | 4.9ms | 8.4ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 0.55 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 1.2ms
```

### ParallelRead/1KB/C200

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 0.33 MB/s | 2.1ms | 8.4ms | 14.9ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 0.33 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 2.1ms
```

### ParallelRead/1KB/C25

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 1.48 MB/s | 552.1us | 1.4ms | 2.2ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 1.48 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 552.1us
```

### ParallelRead/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 0.93 MB/s | 846.8us | 2.5ms | 3.9ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 0.93 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 846.8us
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 6.24 MB/s | 154.3us | 182.2us | 230.7us | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 6.24 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 154.3us
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 2.33 MB/s | 388.0us | 675.8us | 995.8us | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 2.33 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 388.0us
```

### ParallelWrite/1KB/C100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 0.45 MB/s | 1.5ms | 6.1ms | 10.6ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 0.45 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 1.5ms
```

### ParallelWrite/1KB/C200

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 0.28 MB/s | 2.3ms | 10.2ms | 17.6ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 0.28 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 2.3ms
```

### ParallelWrite/1KB/C25

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 1.20 MB/s | 666.4us | 1.8ms | 3.1ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 1.20 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 666.4us
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 0.72 MB/s | 1.0ms | 3.4ms | 5.4ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 0.72 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 1.0ms
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 561.81 MB/s | 465.6us | 687.8us | 1.2ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 561.81 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 465.6us
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 527.27 MB/s | 485.7us | 808.6us | 1.4ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 527.27 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 485.7us
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 541.79 MB/s | 480.8us | 721.1us | 1.3ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 541.79 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 480.8us
```

### Read/100MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 2272.69 MB/s | 44.7ms | 67.2ms | 72.3ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 2272.69 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 44.7ms
```

### Read/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 2431.48 MB/s | 4.3ms | 5.6ms | 6.9ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 2431.48 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 4.3ms
```

### Read/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 10.08 MB/s | 93.0us | 121.1us | 157.5us | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 10.08 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 93.0us
```

### Read/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 1809.41 MB/s | 508.2us | 871.2us | 2.2ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 1809.41 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 508.2us
```

### Read/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 603.55 MB/s | 98.8us | 132.1us | 189.9us | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 603.55 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 98.8us
```

### Scale/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 923 ops/s | 1.1ms | 1.1ms | 1.1ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 923 ops/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 1.1ms
```

### Scale/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 89 ops/s | 11.3ms | 11.3ms | 11.3ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 89 ops/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 11.3ms
```

### Scale/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 9 ops/s | 110.4ms | 110.4ms | 110.4ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 9 ops/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 110.4ms
```

### Scale/Delete/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 1 ops/s | 1.10s | 1.10s | 1.10s | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 1 ops/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 1.10s
```

### Scale/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 3810 ops/s | 262.5us | 262.5us | 262.5us | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 3810 ops/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 262.5us
```

### Scale/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 1446 ops/s | 691.7us | 691.7us | 691.7us | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 1446 ops/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 691.7us
```

### Scale/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 253 ops/s | 4.0ms | 4.0ms | 4.0ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 253 ops/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 4.0ms
```

### Scale/List/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 14 ops/s | 73.4ms | 73.4ms | 73.4ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 14 ops/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 73.4ms
```

### Scale/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 1.90 MB/s | 1.3ms | 1.3ms | 1.3ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 1.90 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 1.3ms
```

### Scale/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 2.10 MB/s | 11.6ms | 11.6ms | 11.6ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 2.10 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 11.6ms
```

### Scale/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 1.77 MB/s | 137.7ms | 137.7ms | 137.7ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 1.77 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 137.7ms
```

### Scale/Write/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 2.00 MB/s | 1.22s | 1.22s | 1.22s | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 2.00 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 1.22s
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 9249 ops/s | 106.6us | 132.1us | 184.2us | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 9249 ops/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 106.6us
```

### Write/100MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 720.29 MB/s | 138.9ms | 162.0ms | 162.0ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 720.29 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 138.9ms
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 682.09 MB/s | 14.9ms | 18.5ms | 19.2ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 682.09 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 14.9ms
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 7.73 MB/s | 122.5us | 160.7us | 254.1us | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 7.73 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 122.5us
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 745.40 MB/s | 1.3ms | 1.8ms | 2.0ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 745.40 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 1.3ms
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 318.58 MB/s | 194.0us | 243.4us | 425.5us | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 318.58 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████████████████ 194.0us
```

## Recommendations

- **Write-heavy workloads:** herd_cluster
- **Read-heavy workloads:** herd_cluster

---

*Generated by storage benchmark CLI*
