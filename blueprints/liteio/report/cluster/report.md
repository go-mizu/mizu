# Storage Benchmark Report

**Generated:** 0001-01-01T00:00:00Z

**Go Version:** go1.26.0

**Platform:** darwin/arm64

## Executive Summary

### Summary

**Overall Winner:** herd_cluster (won 35/40 benchmarks, 88%)

| Rank | Driver | Wins | Win Rate |
|------|--------|------|----------|
| 1 | herd_cluster | 35 | 88% |
| 2 | seaweedfs_cluster | 4 | 10% |
| 3 | minio_cluster | 1 | 2% |

### Performance Leaders

| Operation | Leader | Performance | Margin |
|-----------|--------|-------------|--------|
| Small Read (1KB) | herd_cluster | 14.0 MB/s | 2.6x vs seaweedfs_cluster |
| Small Write (1KB) | herd_cluster | 9.6 MB/s | 3.5x vs seaweedfs_cluster |
| Large Read (10MB) | minio_cluster | 2.8 GB/s | +22% vs herd_cluster |
| Large Write (10MB) | herd_cluster | 868.2 MB/s | 2.6x vs minio_cluster |
| Delete | herd_cluster | 11.5K ops/s | +55% vs seaweedfs_cluster |
| Stat | herd_cluster | 11.4K ops/s | +43% vs seaweedfs_cluster |
| List (100 objects) | herd_cluster | 2.0K ops/s | +85% vs seaweedfs_cluster |
| Copy | herd_cluster | 8.2 MB/s | 4.4x vs seaweedfs_cluster |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (10MB+) | **herd_cluster** | 868 MB/s | Best for media, backups |
| Large File Downloads (10MB) | **minio_cluster** | 2782 MB/s | Best for streaming, CDN |
| Small File Operations | **herd_cluster** | 12056 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **herd_cluster** | - | Best for multi-user apps |

### Large File Performance (10MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| herd_cluster | 868.2 | 2273.0 | 11.4ms | 4.3ms |
| minio_cluster | 338.7 | 2782.3 | 29.5ms | 3.5ms |
| seaweedfs_cluster | 304.0 | 2115.7 | 32.5ms | 4.6ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| herd_cluster | 9807 | 14305 | 95.7us | 66.4us |
| minio_cluster | 92 | 2039 | 1.1ms | 485.0us |
| seaweedfs_cluster | 2783 | 5399 | 345.8us | 181.7us |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| herd_cluster | 11412 | 1975 | 11522 |
| minio_cluster | 3009 | 378 | 1227 |
| seaweedfs_cluster | 7956 | 1065 | 7431 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| herd_cluster | 7.31 | 2.93 | 0.86 |
| minio_cluster | 0.06 | 0.02 | 0.03 |
| seaweedfs_cluster | 2.58 | 1.01 | 0.28 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| herd_cluster | 9.36 | 4.03 | 1.19 |
| minio_cluster | 1.80 | 0.67 | 0.17 |
| seaweedfs_cluster | 4.57 | 1.78 | 0.52 |

*\* indicates errors occurred*

### Scale Performance

Performance with varying numbers of objects (256B each).

**Write N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 |
|--------|------|------|------|------|
| herd_cluster | 116.6us | 1.0ms | 10.2ms | 97.0ms |
| minio_cluster | 1.1ms | 190.0ms | 443.3ms | 12.15s |
| seaweedfs_cluster | 373.3us | 3.3ms | 32.8ms | 333.9ms |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 |
|--------|------|------|------|------|
| herd_cluster | 225.9us | 229.5us | 567.7us | 3.9ms |
| minio_cluster | 840.0us | 1.5ms | 5.5ms | 45.4ms |
| seaweedfs_cluster | 368.2us | 390.4us | 1.0ms | 6.2ms |

*\* indicates errors occurred*

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

- **herd_cluster** (40 benchmarks)
- **minio_cluster** (40 benchmarks)
- **seaweedfs_cluster** (40 benchmarks)

## Detailed Results

### Copy/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 8.21 MB/s | 113.1us | 147.4us | 207.8us | 0 |
| seaweedfs_cluster | 1.87 MB/s | 505.0us | 600.0us | 875.1us | 0 |
| minio_cluster | 0.95 MB/s | 1.0ms | 1.3ms | 1.4ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 8.21 MB/s
seaweedfs_cluster ██████ 1.87 MB/s
minio_cluster ███ 0.95 MB/s
```

**Latency (P50)**
```
herd_cluster ███ 113.1us
seaweedfs_cluster ███████████████ 505.0us
minio_cluster ██████████████████████████████ 1.0ms
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 11522 ops/s | 82.9us | 107.2us | 155.4us | 0 |
| seaweedfs_cluster | 7431 ops/s | 130.4us | 161.3us | 205.1us | 0 |
| minio_cluster | 1227 ops/s | 815.8us | 975.4us | 1.1ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 11522 ops/s
seaweedfs_cluster ███████████████████ 7431 ops/s
minio_cluster ███ 1227 ops/s
```

**Latency (P50)**
```
herd_cluster ███ 82.9us
seaweedfs_cluster ████ 130.4us
minio_cluster ██████████████████████████████ 815.8us
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 0.99 MB/s | 93.5us | 113.1us | 138.0us | 0 |
| seaweedfs_cluster | 0.28 MB/s | 330.2us | 380.6us | 467.0us | 0 |
| minio_cluster | 0.01 MB/s | 1.2ms | 1.9ms | 242.1ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 0.99 MB/s
seaweedfs_cluster ████████ 0.28 MB/s
minio_cluster █ 0.01 MB/s
```

**Latency (P50)**
```
herd_cluster ██ 93.5us
seaweedfs_cluster ████████ 330.2us
minio_cluster ██████████████████████████████ 1.2ms
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 10215 ops/s | 93.5us | 126.3us | 163.6us | 0 |
| seaweedfs_cluster | 6322 ops/s | 153.0us | 189.7us | 232.9us | 0 |
| minio_cluster | 71 ops/s | 1.1ms | 67.9ms | 279.0ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 10215 ops/s
seaweedfs_cluster ██████████████████ 6322 ops/s
minio_cluster █ 71 ops/s
```

**Latency (P50)**
```
herd_cluster ██ 93.5us
seaweedfs_cluster ████ 153.0us
minio_cluster ██████████████████████████████ 1.1ms
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 0.93 MB/s | 99.2us | 119.8us | 156.5us | 0 |
| seaweedfs_cluster | 0.26 MB/s | 363.7us | 424.5us | 559.3us | 0 |
| minio_cluster | 0.01 MB/s | 1.2ms | 2.2ms | 265.8ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 0.93 MB/s
seaweedfs_cluster ████████ 0.26 MB/s
minio_cluster █ 0.01 MB/s
```

**Latency (P50)**
```
herd_cluster ██ 99.2us
seaweedfs_cluster ████████ 363.7us
minio_cluster ██████████████████████████████ 1.2ms
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 1975 ops/s | 493.5us | 561.5us | 827.0us | 0 |
| seaweedfs_cluster | 1065 ops/s | 924.5us | 1.0ms | 1.3ms | 0 |
| minio_cluster | 378 ops/s | 2.6ms | 2.9ms | 4.0ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 1975 ops/s
seaweedfs_cluster ████████████████ 1065 ops/s
minio_cluster █████ 378 ops/s
```

**Latency (P50)**
```
herd_cluster █████ 493.5us
seaweedfs_cluster ██████████ 924.5us
minio_cluster ██████████████████████████████ 2.6ms
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 13.43 MB/s | 876.5us | 3.1ms | 4.9ms | 0 |
| seaweedfs_cluster | 5.10 MB/s | 2.9ms | 5.9ms | 7.7ms | 0 |
| minio_cluster | 0.65 MB/s | 11.1ms | 72.9ms | 313.9ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 13.43 MB/s
seaweedfs_cluster ███████████ 5.10 MB/s
minio_cluster █ 0.65 MB/s
```

**Latency (P50)**
```
herd_cluster ██ 876.5us
seaweedfs_cluster ███████ 2.9ms
minio_cluster ██████████████████████████████ 11.1ms
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 17.09 MB/s | 694.2us | 2.4ms | 3.8ms | 0 |
| seaweedfs_cluster | 6.93 MB/s | 2.0ms | 4.1ms | 5.9ms | 0 |
| minio_cluster | 1.69 MB/s | 6.4ms | 13.1ms | 105.1ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 17.09 MB/s
seaweedfs_cluster ████████████ 6.93 MB/s
minio_cluster ██ 1.69 MB/s
```

**Latency (P50)**
```
herd_cluster ███ 694.2us
seaweedfs_cluster █████████ 2.0ms
minio_cluster ██████████████████████████████ 6.4ms
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 11.43 MB/s | 1.0ms | 3.5ms | 5.9ms | 0 |
| seaweedfs_cluster | 4.27 MB/s | 3.6ms | 5.7ms | 7.4ms | 0 |
| minio_cluster | 0.42 MB/s | 15.2ms | 218.5ms | 330.2ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 11.43 MB/s
seaweedfs_cluster ███████████ 4.27 MB/s
minio_cluster █ 0.42 MB/s
```

**Latency (P50)**
```
herd_cluster ██ 1.0ms
seaweedfs_cluster ███████ 3.6ms
minio_cluster ██████████████████████████████ 15.2ms
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 368.94 MB/s | 40.5ms | 43.0ms | 44.3ms | 0 |
| minio_cluster | 287.31 MB/s | 51.9ms | 54.6ms | 54.6ms | 0 |
| seaweedfs_cluster | 231.24 MB/s | 64.9ms | 65.9ms | 65.9ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 368.94 MB/s
minio_cluster ███████████████████████ 287.31 MB/s
seaweedfs_cluster ██████████████████ 231.24 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████████████ 40.5ms
minio_cluster ███████████████████████ 51.9ms
seaweedfs_cluster ██████████████████████████████ 64.9ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 9.36 MB/s | 101.6us | 119.3us | 143.2us | 0 |
| seaweedfs_cluster | 4.57 MB/s | 210.5us | 237.8us | 276.2us | 0 |
| minio_cluster | 1.80 MB/s | 526.3us | 679.4us | 1.0ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 9.36 MB/s
seaweedfs_cluster ██████████████ 4.57 MB/s
minio_cluster █████ 1.80 MB/s
```

**Latency (P50)**
```
herd_cluster █████ 101.6us
seaweedfs_cluster ███████████ 210.5us
minio_cluster ██████████████████████████████ 526.3us
```

### ParallelRead/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 4.03 MB/s | 227.2us | 404.7us | 599.8us | 0 |
| seaweedfs_cluster | 1.78 MB/s | 515.3us | 775.9us | 1.1ms | 0 |
| minio_cluster | 0.67 MB/s | 1.3ms | 2.1ms | 3.8ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 4.03 MB/s
seaweedfs_cluster █████████████ 1.78 MB/s
minio_cluster ████ 0.67 MB/s
```

**Latency (P50)**
```
herd_cluster █████ 227.2us
seaweedfs_cluster ███████████ 515.3us
minio_cluster ██████████████████████████████ 1.3ms
```

### ParallelRead/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 1.19 MB/s | 610.0us | 2.2ms | 3.4ms | 0 |
| seaweedfs_cluster | 0.52 MB/s | 1.7ms | 3.1ms | 4.7ms | 0 |
| minio_cluster | 0.17 MB/s | 5.0ms | 11.7ms | 21.7ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 1.19 MB/s
seaweedfs_cluster █████████████ 0.52 MB/s
minio_cluster ████ 0.17 MB/s
```

**Latency (P50)**
```
herd_cluster ███ 610.0us
seaweedfs_cluster ██████████ 1.7ms
minio_cluster ██████████████████████████████ 5.0ms
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 7.31 MB/s | 129.7us | 153.8us | 203.4us | 0 |
| seaweedfs_cluster | 2.58 MB/s | 368.4us | 444.3us | 602.0us | 0 |
| minio_cluster | 0.06 MB/s | 1.2ms | 104.5ms | 278.0ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 7.31 MB/s
seaweedfs_cluster ██████████ 2.58 MB/s
minio_cluster █ 0.06 MB/s
```

**Latency (P50)**
```
herd_cluster ███ 129.7us
seaweedfs_cluster ████████ 368.4us
minio_cluster ██████████████████████████████ 1.2ms
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 2.93 MB/s | 311.7us | 540.0us | 775.8us | 0 |
| seaweedfs_cluster | 1.01 MB/s | 930.6us | 1.3ms | 1.7ms | 0 |
| minio_cluster | 0.02 MB/s | 4.4ms | 318.8ms | 334.3ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 2.93 MB/s
seaweedfs_cluster ██████████ 1.01 MB/s
minio_cluster █ 0.02 MB/s
```

**Latency (P50)**
```
herd_cluster ██ 311.7us
seaweedfs_cluster ██████ 930.6us
minio_cluster ██████████████████████████████ 4.4ms
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 0.86 MB/s | 832.0us | 2.8ms | 4.4ms | 0 |
| seaweedfs_cluster | 0.28 MB/s | 3.3ms | 5.5ms | 8.4ms | 0 |
| minio_cluster | 0.03 MB/s | 18.7ms | 166.2ms | 341.0ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 0.86 MB/s
seaweedfs_cluster █████████ 0.28 MB/s
minio_cluster █ 0.03 MB/s
```

**Latency (P50)**
```
herd_cluster █ 832.0us
seaweedfs_cluster █████ 3.3ms
minio_cluster ██████████████████████████████ 18.7ms
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs_cluster | 1022.68 MB/s | 239.3us | 275.5us | 327.7us | 0 |
| herd_cluster | 581.79 MB/s | 398.6us | 572.7us | 889.5us | 0 |
| minio_cluster | 295.01 MB/s | 842.1us | 1.1ms | 1.9ms | 0 |

**Throughput**
```
seaweedfs_cluster ██████████████████████████████ 1022.68 MB/s
herd_cluster █████████████████ 581.79 MB/s
minio_cluster ████████ 295.01 MB/s
```

**Latency (P50)**
```
seaweedfs_cluster ████████ 239.3us
herd_cluster ██████████████ 398.6us
minio_cluster ██████████████████████████████ 842.1us
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs_cluster | 970.75 MB/s | 243.6us | 329.9us | 549.9us | 0 |
| herd_cluster | 582.80 MB/s | 397.9us | 559.5us | 869.3us | 0 |
| minio_cluster | 292.79 MB/s | 859.8us | 1.1ms | 1.9ms | 0 |

**Throughput**
```
seaweedfs_cluster ██████████████████████████████ 970.75 MB/s
herd_cluster ██████████████████ 582.80 MB/s
minio_cluster █████████ 292.79 MB/s
```

**Latency (P50)**
```
seaweedfs_cluster ████████ 243.6us
herd_cluster █████████████ 397.9us
minio_cluster ██████████████████████████████ 859.8us
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs_cluster | 1010.76 MB/s | 240.2us | 290.9us | 368.5us | 0 |
| herd_cluster | 480.34 MB/s | 471.8us | 752.2us | 1.3ms | 0 |
| minio_cluster | 277.47 MB/s | 875.8us | 1.2ms | 2.2ms | 0 |

**Throughput**
```
seaweedfs_cluster ██████████████████████████████ 1010.76 MB/s
herd_cluster ██████████████ 480.34 MB/s
minio_cluster ████████ 277.47 MB/s
```

**Latency (P50)**
```
seaweedfs_cluster ████████ 240.2us
herd_cluster ████████████████ 471.8us
minio_cluster ██████████████████████████████ 875.8us
```

### Read/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio_cluster | 2782.29 MB/s | 3.5ms | 4.3ms | 5.5ms | 0 |
| herd_cluster | 2273.01 MB/s | 4.3ms | 5.7ms | 6.2ms | 0 |
| seaweedfs_cluster | 2115.65 MB/s | 4.6ms | 5.6ms | 5.9ms | 0 |

**Throughput**
```
minio_cluster ██████████████████████████████ 2782.29 MB/s
herd_cluster ████████████████████████ 2273.01 MB/s
seaweedfs_cluster ██████████████████████ 2115.65 MB/s
```

**Latency (P50)**
```
minio_cluster ██████████████████████ 3.5ms
herd_cluster ████████████████████████████ 4.3ms
seaweedfs_cluster ██████████████████████████████ 4.6ms
```

### Read/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 13.97 MB/s | 66.4us | 87.5us | 133.1us | 0 |
| seaweedfs_cluster | 5.27 MB/s | 181.7us | 209.4us | 269.9us | 0 |
| minio_cluster | 1.99 MB/s | 485.0us | 625.5us | 744.7us | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 13.97 MB/s
seaweedfs_cluster ███████████ 5.27 MB/s
minio_cluster ████ 1.99 MB/s
```

**Latency (P50)**
```
herd_cluster ████ 66.4us
seaweedfs_cluster ███████████ 181.7us
minio_cluster ██████████████████████████████ 485.0us
```

### Read/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs_cluster | 2053.76 MB/s | 472.8us | 583.7us | 690.6us | 0 |
| herd_cluster | 1963.60 MB/s | 499.3us | 727.8us | 1.0ms | 0 |
| minio_cluster | 946.02 MB/s | 1.0ms | 1.3ms | 2.3ms | 0 |

**Throughput**
```
seaweedfs_cluster ██████████████████████████████ 2053.76 MB/s
herd_cluster ████████████████████████████ 1963.60 MB/s
minio_cluster █████████████ 946.02 MB/s
```

**Latency (P50)**
```
seaweedfs_cluster █████████████ 472.8us
herd_cluster ██████████████ 499.3us
minio_cluster ██████████████████████████████ 1.0ms
```

### Read/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 860.08 MB/s | 69.4us | 88.5us | 132.6us | 0 |
| seaweedfs_cluster | 294.61 MB/s | 206.2us | 241.7us | 366.2us | 0 |
| minio_cluster | 113.40 MB/s | 533.5us | 733.8us | 1.1ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 860.08 MB/s
seaweedfs_cluster ██████████ 294.61 MB/s
minio_cluster ███ 113.40 MB/s
```

**Latency (P50)**
```
herd_cluster ███ 69.4us
seaweedfs_cluster ███████████ 206.2us
minio_cluster ██████████████████████████████ 533.5us
```

### Scale/Delete/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 9288 ops/s | 107.7us | 107.7us | 107.7us | 0 |
| seaweedfs_cluster | 6348 ops/s | 157.5us | 157.5us | 157.5us | 0 |
| minio_cluster | 1144 ops/s | 874.4us | 874.4us | 874.4us | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 9288 ops/s
seaweedfs_cluster ████████████████████ 6348 ops/s
minio_cluster ███ 1144 ops/s
```

**Latency (P50)**
```
herd_cluster ███ 107.7us
seaweedfs_cluster █████ 157.5us
minio_cluster ██████████████████████████████ 874.4us
```

### Scale/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 1080 ops/s | 925.5us | 925.5us | 925.5us | 0 |
| seaweedfs_cluster | 677 ops/s | 1.5ms | 1.5ms | 1.5ms | 0 |
| minio_cluster | 100 ops/s | 10.0ms | 10.0ms | 10.0ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 1080 ops/s
seaweedfs_cluster ██████████████████ 677 ops/s
minio_cluster ██ 100 ops/s
```

**Latency (P50)**
```
herd_cluster ██ 925.5us
seaweedfs_cluster ████ 1.5ms
minio_cluster ██████████████████████████████ 10.0ms
```

### Scale/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 117 ops/s | 8.5ms | 8.5ms | 8.5ms | 0 |
| seaweedfs_cluster | 73 ops/s | 13.8ms | 13.8ms | 13.8ms | 0 |
| minio_cluster | 12 ops/s | 80.9ms | 80.9ms | 80.9ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 117 ops/s
seaweedfs_cluster ██████████████████ 73 ops/s
minio_cluster ███ 12 ops/s
```

**Latency (P50)**
```
herd_cluster ███ 8.5ms
seaweedfs_cluster █████ 13.8ms
minio_cluster ██████████████████████████████ 80.9ms
```

### Scale/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 11 ops/s | 87.9ms | 87.9ms | 87.9ms | 0 |
| seaweedfs_cluster | 7 ops/s | 136.3ms | 136.3ms | 136.3ms | 0 |
| minio_cluster | 1 ops/s | 836.3ms | 836.3ms | 836.3ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 11 ops/s
seaweedfs_cluster ███████████████████ 7 ops/s
minio_cluster ███ 1 ops/s
```

**Latency (P50)**
```
herd_cluster ███ 87.9ms
seaweedfs_cluster ████ 136.3ms
minio_cluster ██████████████████████████████ 836.3ms
```

### Scale/List/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 4427 ops/s | 225.9us | 225.9us | 225.9us | 0 |
| seaweedfs_cluster | 2716 ops/s | 368.2us | 368.2us | 368.2us | 0 |
| minio_cluster | 1191 ops/s | 840.0us | 840.0us | 840.0us | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 4427 ops/s
seaweedfs_cluster ██████████████████ 2716 ops/s
minio_cluster ████████ 1191 ops/s
```

**Latency (P50)**
```
herd_cluster ████████ 225.9us
seaweedfs_cluster █████████████ 368.2us
minio_cluster ██████████████████████████████ 840.0us
```

### Scale/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 4357 ops/s | 229.5us | 229.5us | 229.5us | 0 |
| seaweedfs_cluster | 2562 ops/s | 390.4us | 390.4us | 390.4us | 0 |
| minio_cluster | 679 ops/s | 1.5ms | 1.5ms | 1.5ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 4357 ops/s
seaweedfs_cluster █████████████████ 2562 ops/s
minio_cluster ████ 679 ops/s
```

**Latency (P50)**
```
herd_cluster ████ 229.5us
seaweedfs_cluster ███████ 390.4us
minio_cluster ██████████████████████████████ 1.5ms
```

### Scale/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 1762 ops/s | 567.7us | 567.7us | 567.7us | 0 |
| seaweedfs_cluster | 981 ops/s | 1.0ms | 1.0ms | 1.0ms | 0 |
| minio_cluster | 181 ops/s | 5.5ms | 5.5ms | 5.5ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 1762 ops/s
seaweedfs_cluster ████████████████ 981 ops/s
minio_cluster ███ 181 ops/s
```

**Latency (P50)**
```
herd_cluster ███ 567.7us
seaweedfs_cluster █████ 1.0ms
minio_cluster ██████████████████████████████ 5.5ms
```

### Scale/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 255 ops/s | 3.9ms | 3.9ms | 3.9ms | 0 |
| seaweedfs_cluster | 161 ops/s | 6.2ms | 6.2ms | 6.2ms | 0 |
| minio_cluster | 22 ops/s | 45.4ms | 45.4ms | 45.4ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 255 ops/s
seaweedfs_cluster ██████████████████ 161 ops/s
minio_cluster ██ 22 ops/s
```

**Latency (P50)**
```
herd_cluster ██ 3.9ms
seaweedfs_cluster ████ 6.2ms
minio_cluster ██████████████████████████████ 45.4ms
```

### Scale/Write/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 2.09 MB/s | 116.6us | 116.6us | 116.6us | 0 |
| seaweedfs_cluster | 0.65 MB/s | 373.3us | 373.3us | 373.3us | 0 |
| minio_cluster | 0.23 MB/s | 1.1ms | 1.1ms | 1.1ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 2.09 MB/s
seaweedfs_cluster █████████ 0.65 MB/s
minio_cluster ███ 0.23 MB/s
```

**Latency (P50)**
```
herd_cluster ███ 116.6us
seaweedfs_cluster ██████████ 373.3us
minio_cluster ██████████████████████████████ 1.1ms
```

### Scale/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 2.35 MB/s | 1.0ms | 1.0ms | 1.0ms | 0 |
| seaweedfs_cluster | 0.74 MB/s | 3.3ms | 3.3ms | 3.3ms | 0 |
| minio_cluster | 0.01 MB/s | 190.0ms | 190.0ms | 190.0ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 2.35 MB/s
seaweedfs_cluster █████████ 0.74 MB/s
minio_cluster █ 0.01 MB/s
```

**Latency (P50)**
```
herd_cluster █ 1.0ms
seaweedfs_cluster █ 3.3ms
minio_cluster ██████████████████████████████ 190.0ms
```

### Scale/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 2.40 MB/s | 10.2ms | 10.2ms | 10.2ms | 0 |
| seaweedfs_cluster | 0.74 MB/s | 32.8ms | 32.8ms | 32.8ms | 0 |
| minio_cluster | 0.06 MB/s | 443.3ms | 443.3ms | 443.3ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 2.40 MB/s
seaweedfs_cluster █████████ 0.74 MB/s
minio_cluster █ 0.06 MB/s
```

**Latency (P50)**
```
herd_cluster █ 10.2ms
seaweedfs_cluster ██ 32.8ms
minio_cluster ██████████████████████████████ 443.3ms
```

### Scale/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 2.52 MB/s | 97.0ms | 97.0ms | 97.0ms | 0 |
| seaweedfs_cluster | 0.73 MB/s | 333.9ms | 333.9ms | 333.9ms | 0 |
| minio_cluster | 0.02 MB/s | 12.15s | 12.15s | 12.15s | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 2.52 MB/s
seaweedfs_cluster ████████ 0.73 MB/s
minio_cluster █ 0.02 MB/s
```

**Latency (P50)**
```
herd_cluster █ 97.0ms
seaweedfs_cluster █ 333.9ms
minio_cluster ██████████████████████████████ 12.15s
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 11412 ops/s | 84.1us | 106.3us | 148.8us | 0 |
| seaweedfs_cluster | 7956 ops/s | 122.4us | 145.9us | 189.0us | 0 |
| minio_cluster | 3009 ops/s | 325.5us | 403.0us | 486.3us | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 11412 ops/s
seaweedfs_cluster ████████████████████ 7956 ops/s
minio_cluster ███████ 3009 ops/s
```

**Latency (P50)**
```
herd_cluster ███████ 84.1us
seaweedfs_cluster ███████████ 122.4us
minio_cluster ██████████████████████████████ 325.5us
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 868.23 MB/s | 11.4ms | 13.3ms | 13.5ms | 0 |
| minio_cluster | 338.71 MB/s | 29.5ms | 30.9ms | 31.2ms | 0 |
| seaweedfs_cluster | 304.05 MB/s | 32.5ms | 34.1ms | 34.9ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 868.23 MB/s
minio_cluster ███████████ 338.71 MB/s
seaweedfs_cluster ██████████ 304.05 MB/s
```

**Latency (P50)**
```
herd_cluster ██████████ 11.4ms
minio_cluster ███████████████████████████ 29.5ms
seaweedfs_cluster ██████████████████████████████ 32.5ms
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 9.58 MB/s | 95.7us | 135.9us | 210.8us | 0 |
| seaweedfs_cluster | 2.72 MB/s | 345.8us | 418.0us | 655.8us | 0 |
| minio_cluster | 0.09 MB/s | 1.1ms | 2.3ms | 213.2ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 9.58 MB/s
seaweedfs_cluster ████████ 2.72 MB/s
minio_cluster █ 0.09 MB/s
```

**Latency (P50)**
```
herd_cluster ██ 95.7us
seaweedfs_cluster █████████ 345.8us
minio_cluster ██████████████████████████████ 1.1ms
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 875.25 MB/s | 1.1ms | 1.5ms | 1.6ms | 0 |
| seaweedfs_cluster | 225.49 MB/s | 4.4ms | 4.7ms | 4.8ms | 0 |
| minio_cluster | 208.04 MB/s | 4.9ms | 5.3ms | 5.5ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 875.25 MB/s
seaweedfs_cluster ███████ 225.49 MB/s
minio_cluster ███████ 208.04 MB/s
```

**Latency (P50)**
```
herd_cluster ██████ 1.1ms
seaweedfs_cluster ███████████████████████████ 4.4ms
minio_cluster ██████████████████████████████ 4.9ms
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 374.54 MB/s | 157.2us | 191.9us | 467.1us | 0 |
| seaweedfs_cluster | 101.09 MB/s | 599.9us | 671.3us | 977.5us | 0 |
| minio_cluster | 38.92 MB/s | 1.4ms | 3.3ms | 4.5ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 374.54 MB/s
seaweedfs_cluster ████████ 101.09 MB/s
minio_cluster ███ 38.92 MB/s
```

**Latency (P50)**
```
herd_cluster ███ 157.2us
seaweedfs_cluster █████████████ 599.9us
minio_cluster ██████████████████████████████ 1.4ms
```

## Recommendations

- **Write-heavy workloads:** herd_cluster
- **Read-heavy workloads:** minio_cluster

---

*Generated by storage benchmark CLI*
