# Storage Benchmark Report

**Generated:** 0001-01-01T00:00:00Z

**Go Version:** go1.26.0

**Platform:** darwin/arm64

## Executive Summary

### Performance Leaders

| Operation | Leader | Performance | Margin |
|-----------|--------|-------------|--------|
| Small Write (1KB) | minio_cluster | 1.6 MB/s |  |
| Large Write (10MB) | minio_cluster | 373.8 MB/s |  |
| Delete | minio_cluster | 1.7K ops/s |  |
| Stat | minio_cluster | 5.0K ops/s |  |
| List (100 objects) | minio_cluster | 1.0K ops/s |  |
| Copy | minio_cluster | 1.5 MB/s |  |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (10MB+) | **minio_cluster** | 374 MB/s | Best for media, backups |
| Small File Operations | **minio_cluster** | 794 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **minio_cluster** | - | Best for multi-user apps |

### Large File Performance (10MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| minio_cluster | 373.8 | 0.0 | 26.6ms | 0ns |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| minio_cluster | 1589 | 0 | 594.0us | 0ns |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| minio_cluster | 5016 | 1033 | 1684 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| minio_cluster | 1.47 | 0.21 | 0.04 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| minio_cluster | 0.00* | 1.50* | 0.51* |

*\* indicates errors occurred*

### Scale Performance

Performance with varying numbers of objects (256B each).

**Write N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 |
|--------|------|------|------|------|
| minio_cluster | 602.1us | 5.9ms | 61.1ms | 626.4ms |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 |
|--------|------|------|------|------|
| minio_cluster | 0ns* | 0ns* | 0ns* | 0ns* |

*\* indicates errors occurred*

### Warnings

- **minio_cluster**: 54963 errors during benchmarks

---

## Configuration

| Parameter | Value |
|-----------|-------|
| BenchTime | 500ms |
| MinIterations | 3 |
| Warmup | 2 |
| Concurrency | 50 |
| Timeout | 30s |

## Drivers Tested

- **minio_cluster** (40 benchmarks)

## Detailed Results

### Copy/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio_cluster | 1.52 MB/s | 632.2us | 742.7us | 821.6us | 2112 |

**Throughput**
```
minio_cluster ██████████████████████████████ 1.52 MB/s
```

**Latency (P50)**
```
minio_cluster ██████████████████████████████ 632.2us
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio_cluster | 1684 ops/s | 562.6us | 689.0us | 789.5us | 0 |

**Throughput**
```
minio_cluster ██████████████████████████████ 1684 ops/s
```

**Latency (P50)**
```
minio_cluster ██████████████████████████████ 562.6us
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio_cluster | 0.15 MB/s | 603.3us | 859.6us | 986.9us | 0 |

**Throughput**
```
minio_cluster ██████████████████████████████ 0.15 MB/s
```

**Latency (P50)**
```
minio_cluster ██████████████████████████████ 603.3us
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio_cluster | 1697 ops/s | 555.4us | 778.4us | 908.2us | 0 |

**Throughput**
```
minio_cluster ██████████████████████████████ 1697 ops/s
```

**Latency (P50)**
```
minio_cluster ██████████████████████████████ 555.4us
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio_cluster | 0.14 MB/s | 640.8us | 906.3us | 1.0ms | 0 |

**Throughput**
```
minio_cluster ██████████████████████████████ 0.14 MB/s
```

**Latency (P50)**
```
minio_cluster ██████████████████████████████ 640.8us
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio_cluster | 1033 ops/s | 946.5us | 1.1ms | 1.3ms | 0 |

**Throughput**
```
minio_cluster ██████████████████████████████ 1033 ops/s
```

**Latency (P50)**
```
minio_cluster ██████████████████████████████ 946.5us
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio_cluster | 0.97 MB/s | 9.8ms | 56.2ms | 125.7ms | 1448 |

**Throughput**
```
minio_cluster ██████████████████████████████ 0.97 MB/s
```

**Latency (P50)**
```
minio_cluster ██████████████████████████████ 9.8ms
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio_cluster | 2.86 MB/s | 3.2ms | 10.8ms | 110.2ms | 8698 |

**Throughput**
```
minio_cluster ██████████████████████████████ 2.86 MB/s
```

**Latency (P50)**
```
minio_cluster ██████████████████████████████ 3.2ms
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio_cluster | 0.76 MB/s | 13.8ms | 75.3ms | 98.3ms | 178 |

**Throughput**
```
minio_cluster ██████████████████████████████ 0.76 MB/s
```

**Latency (P50)**
```
minio_cluster ██████████████████████████████ 13.8ms
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio_cluster | 0.00 MB/s | 0ns | 0ns | 0ns | 53 |

**Throughput**
```
minio_cluster  0.00 MB/s
```

**Latency (P50)**
```
```

### ParallelRead/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio_cluster | 0.00 MB/s | 0ns | 0ns | 0ns | 3727 |

**Throughput**
```
minio_cluster  0.00 MB/s
```

**Latency (P50)**
```
```

### ParallelRead/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio_cluster | 1.50 MB/s | 592.7us | 1.0ms | 1.8ms | 8595 |

**Throughput**
```
minio_cluster ██████████████████████████████ 1.50 MB/s
```

**Latency (P50)**
```
minio_cluster ██████████████████████████████ 592.7us
```

### ParallelRead/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio_cluster | 0.51 MB/s | 1.6ms | 4.6ms | 7.3ms | 8403 |

**Throughput**
```
minio_cluster ██████████████████████████████ 0.51 MB/s
```

**Latency (P50)**
```
minio_cluster ██████████████████████████████ 1.6ms
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio_cluster | 1.47 MB/s | 627.1us | 790.6us | 875.9us | 0 |

**Throughput**
```
minio_cluster ██████████████████████████████ 1.47 MB/s
```

**Latency (P50)**
```
minio_cluster ██████████████████████████████ 627.1us
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio_cluster | 0.21 MB/s | 3.9ms | 8.7ms | 28.3ms | 0 |

**Throughput**
```
minio_cluster ██████████████████████████████ 0.21 MB/s
```

**Latency (P50)**
```
minio_cluster ██████████████████████████████ 3.9ms
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio_cluster | 0.04 MB/s | 18.3ms | 77.7ms | 110.2ms | 0 |

**Throughput**
```
minio_cluster ██████████████████████████████ 0.04 MB/s
```

**Latency (P50)**
```
minio_cluster ██████████████████████████████ 18.3ms
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio_cluster | 396.08 MB/s | 633.2us | 709.6us | 911.5us | 2055 |

**Throughput**
```
minio_cluster ██████████████████████████████ 396.08 MB/s
```

**Latency (P50)**
```
minio_cluster ██████████████████████████████ 633.2us
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio_cluster | 460.81 MB/s | 532.9us | 667.2us | 730.2us | 2598 |

**Throughput**
```
minio_cluster ██████████████████████████████ 460.81 MB/s
```

**Latency (P50)**
```
minio_cluster ██████████████████████████████ 532.9us
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio_cluster | 545.14 MB/s | 438.9us | 546.2us | 848.4us | 2526 |

**Throughput**
```
minio_cluster ██████████████████████████████ 545.14 MB/s
```

**Latency (P50)**
```
minio_cluster ██████████████████████████████ 438.9us
```

### Read/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio_cluster | 0.00 MB/s | 0ns | 0ns | 0ns | 1111 |

**Throughput**
```
minio_cluster  0.00 MB/s
```

**Latency (P50)**
```
```

### Read/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio_cluster | 0.00 MB/s | 0ns | 0ns | 0ns | 4406 |

**Throughput**
```
minio_cluster  0.00 MB/s
```

**Latency (P50)**
```
```

### Read/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio_cluster | 1162.72 MB/s | 831.5us | 1.3ms | 1.4ms | 1230 |

**Throughput**
```
minio_cluster ██████████████████████████████ 1162.72 MB/s
```

**Latency (P50)**
```
minio_cluster ██████████████████████████████ 831.5us
```

### Read/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio_cluster | 0.00 MB/s | 0ns | 0ns | 0ns | 4257 |

**Throughput**
```
minio_cluster  0.00 MB/s
```

**Latency (P50)**
```
```

### Scale/Delete/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio_cluster | 4088 ops/s | 244.6us | 244.6us | 244.6us | 0 |

**Throughput**
```
minio_cluster ██████████████████████████████ 4088 ops/s
```

**Latency (P50)**
```
minio_cluster ██████████████████████████████ 244.6us
```

### Scale/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio_cluster | 396 ops/s | 2.5ms | 2.5ms | 2.5ms | 0 |

**Throughput**
```
minio_cluster ██████████████████████████████ 396 ops/s
```

**Latency (P50)**
```
minio_cluster ██████████████████████████████ 2.5ms
```

### Scale/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio_cluster | 42 ops/s | 23.7ms | 23.7ms | 23.7ms | 0 |

**Throughput**
```
minio_cluster ██████████████████████████████ 42 ops/s
```

**Latency (P50)**
```
minio_cluster ██████████████████████████████ 23.7ms
```

### Scale/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio_cluster | 5 ops/s | 189.2ms | 189.2ms | 189.2ms | 0 |

**Throughput**
```
minio_cluster ██████████████████████████████ 5 ops/s
```

**Latency (P50)**
```
minio_cluster ██████████████████████████████ 189.2ms
```

### Scale/List/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio_cluster | 0 ops/s | 0ns | 0ns | 0ns | 1 |

**Throughput**
```
minio_cluster  0 ops/s
```

**Latency (P50)**
```
```

### Scale/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio_cluster | 0 ops/s | 0ns | 0ns | 0ns | 1 |

**Throughput**
```
minio_cluster  0 ops/s
```

**Latency (P50)**
```
```

### Scale/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio_cluster | 0 ops/s | 0ns | 0ns | 0ns | 1 |

**Throughput**
```
minio_cluster  0 ops/s
```

**Latency (P50)**
```
```

### Scale/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio_cluster | 0 ops/s | 0ns | 0ns | 0ns | 1 |

**Throughput**
```
minio_cluster  0 ops/s
```

**Latency (P50)**
```
```

### Scale/Write/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio_cluster | 0.41 MB/s | 602.1us | 602.1us | 602.1us | 0 |

**Throughput**
```
minio_cluster ██████████████████████████████ 0.41 MB/s
```

**Latency (P50)**
```
minio_cluster ██████████████████████████████ 602.1us
```

### Scale/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio_cluster | 0.41 MB/s | 5.9ms | 5.9ms | 5.9ms | 0 |

**Throughput**
```
minio_cluster ██████████████████████████████ 0.41 MB/s
```

**Latency (P50)**
```
minio_cluster ██████████████████████████████ 5.9ms
```

### Scale/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio_cluster | 0.40 MB/s | 61.1ms | 61.1ms | 61.1ms | 0 |

**Throughput**
```
minio_cluster ██████████████████████████████ 0.40 MB/s
```

**Latency (P50)**
```
minio_cluster ██████████████████████████████ 61.1ms
```

### Scale/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio_cluster | 0.39 MB/s | 626.4ms | 626.4ms | 626.4ms | 0 |

**Throughput**
```
minio_cluster ██████████████████████████████ 0.39 MB/s
```

**Latency (P50)**
```
minio_cluster ██████████████████████████████ 626.4ms
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio_cluster | 5016 ops/s | 196.4us | 228.0us | 269.5us | 3562 |

**Throughput**
```
minio_cluster ██████████████████████████████ 5016 ops/s
```

**Latency (P50)**
```
minio_cluster ██████████████████████████████ 196.4us
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio_cluster | 373.80 MB/s | 26.6ms | 27.5ms | 27.5ms | 0 |

**Throughput**
```
minio_cluster ██████████████████████████████ 373.80 MB/s
```

**Latency (P50)**
```
minio_cluster ██████████████████████████████ 26.6ms
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio_cluster | 1.55 MB/s | 594.0us | 839.3us | 1.1ms | 0 |

**Throughput**
```
minio_cluster ██████████████████████████████ 1.55 MB/s
```

**Latency (P50)**
```
minio_cluster ██████████████████████████████ 594.0us
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio_cluster | 260.50 MB/s | 3.7ms | 4.0ms | 4.2ms | 0 |

**Throughput**
```
minio_cluster ██████████████████████████████ 260.50 MB/s
```

**Latency (P50)**
```
minio_cluster ██████████████████████████████ 3.7ms
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio_cluster | 68.77 MB/s | 867.5us | 1.1ms | 1.3ms | 0 |

**Throughput**
```
minio_cluster ██████████████████████████████ 68.77 MB/s
```

**Latency (P50)**
```
minio_cluster ██████████████████████████████ 867.5us
```

## Recommendations

- **Write-heavy workloads:** minio_cluster
- **Read-heavy workloads:** minio_cluster

---

*Generated by storage benchmark CLI*
