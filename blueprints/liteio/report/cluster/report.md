# Storage Benchmark Report

**Generated:** 2026-02-20T01:08:45+07:00

**Go Version:** go1.26.0

**Platform:** darwin/arm64

## Executive Summary

### Summary

**Overall Winner:** herd_cluster (won 39/40 benchmarks, 98%)

| Rank | Driver | Wins | Win Rate |
|------|--------|------|----------|
| 1 | herd_cluster | 39 | 98% |
| 2 | minio_cluster | 1 | 2% |

### Performance Leaders

| Operation | Leader | Performance | Margin |
|-----------|--------|-------------|--------|
| Small Read (1KB) | herd_cluster | 10.0 MB/s | 2.1x vs minio_cluster |
| Small Write (1KB) | herd_cluster | 7.8 MB/s | 2.8x vs seaweedfs_cluster |
| Large Read (10MB) | herd_cluster | 4.8 GB/s | 2.0x vs minio_cluster |
| Large Write (10MB) | herd_cluster | 652.0 MB/s | +97% vs minio_cluster |
| Delete | herd_cluster | 13.7K ops/s | 2.4x vs seaweedfs_cluster |
| Stat | herd_cluster | 10.0K ops/s | +25% vs seaweedfs_cluster |
| List (100 objects) | herd_cluster | 1.9K ops/s | 2.0x vs seaweedfs_cluster |
| Copy | herd_cluster | 12.5 MB/s | 16.3x vs seaweedfs_cluster |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (10MB+) | **herd_cluster** | 652 MB/s | Best for media, backups |
| Large File Downloads (10MB) | **herd_cluster** | 4819 MB/s | Best for streaming, CDN |
| Small File Operations | **herd_cluster** | 9088 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **herd_cluster** | - | Best for multi-user apps |

### Large File Performance (10MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| herd_cluster | 652.0 | 4818.9 | 12.7ms | 2.0ms |
| minio_cluster | 330.9 | 2378.2 | 28.6ms | 3.5ms |
| rustfs_cluster | 280.4 | 1915.9 | 33.1ms | 5.1ms |
| seaweedfs_cluster | 186.8 | 2144.5 | 51.0ms | 4.6ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| herd_cluster | 7940 | 10236 | 108.8us | 84.5us |
| minio_cluster | 400 | 4911 | 1.7ms | 198.3us |
| rustfs_cluster | 538 | 2669 | 1.5ms | 313.8us |
| seaweedfs_cluster | 2839 | 2405 | 323.6us | 202.7us |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| herd_cluster | 9988 | 1876 | 13714 |
| minio_cluster | 4872 | 213 | 733 |
| rustfs_cluster | 4302 | 144 | 569 |
| seaweedfs_cluster | 7963 | 923 | 5641 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| herd_cluster | 8.33 | 3.08 | 0.82 |
| minio_cluster | 0.59 | 0.05 | 0.01 |
| rustfs_cluster | 0.63 | 0.03 | 0.01 |
| seaweedfs_cluster | 0.52 | 0.83 | 0.22 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| herd_cluster | 9.03 | 3.19 | 0.83 |
| minio_cluster | 3.82 | 1.29 | 0.12 |
| rustfs_cluster | 2.53 | 0.16 | 0.06 |
| seaweedfs_cluster | 4.24 | 1.30 | 0.38 |

*\* indicates errors occurred*

### Scale Performance

Performance with varying numbers of objects (256B each).

**Write N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 |
|--------|------|------|------|------|
| herd_cluster | 90.2us | 846.2us | 7.5ms | 76.2ms |
| minio_cluster | 2.2ms | 18.4ms | 200.4ms | 1.89s |
| rustfs_cluster | 2.1ms | 20.5ms | 286.3ms | 2.30s |
| seaweedfs_cluster | 610.7us | 6.5ms | 52.8ms | 576.3ms |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 |
|--------|------|------|------|------|
| herd_cluster | 126.2us | 137.0us | 471.4us | 3.7ms |
| minio_cluster | 892.8us | 2.0ms | 11.4ms | 99.6ms |
| rustfs_cluster | 1.8ms | 3.8ms | 21.4ms | 179.8ms |
| seaweedfs_cluster | 532.7us | 1.1ms | 1.2ms | 8.2ms |

*\* indicates errors occurred*

### Warnings

- **herd_cluster**: 1101 errors during benchmarks

---

## Configuration

| Parameter | Value |
|-----------|-------|
| BenchTime | 1s |
| MinIterations | 3 |
| Warmup | 5 |
| Concurrency | 200 |
| Timeout | 1m0s |

## Drivers Tested

- **herd_cluster** (40 benchmarks)
- **minio_cluster** (40 benchmarks)
- **rustfs_cluster** (40 benchmarks)
- **seaweedfs_cluster** (40 benchmarks)

## Detailed Results

### Copy/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 12.50 MB/s | 73.9us | 91.8us | 151.7us | 0 |
| seaweedfs_cluster | 0.77 MB/s | 690.2us | 4.0ms | 8.6ms | 0 |
| minio_cluster | 0.54 MB/s | 1.7ms | 2.3ms | 5.1ms | 0 |
| rustfs_cluster | 0.43 MB/s | 1.9ms | 5.5ms | 7.1ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 12.50 MB/s
seaweedfs_cluster █ 0.77 MB/s
minio_cluster █ 0.54 MB/s
rustfs_cluster █ 0.43 MB/s
```

**Latency (P50)**
```
herd_cluster █ 73.9us
seaweedfs_cluster ██████████ 690.2us
minio_cluster ██████████████████████████ 1.7ms
rustfs_cluster ██████████████████████████████ 1.9ms
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 13714 ops/s | 65.6us | 108.2us | 182.4us | 0 |
| seaweedfs_cluster | 5641 ops/s | 131.0us | 339.8us | 1.1ms | 0 |
| minio_cluster | 733 ops/s | 1.3ms | 1.7ms | 4.5ms | 0 |
| rustfs_cluster | 569 ops/s | 1.6ms | 2.5ms | 6.4ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 13714 ops/s
seaweedfs_cluster ████████████ 5641 ops/s
minio_cluster █ 733 ops/s
rustfs_cluster █ 569 ops/s
```

**Latency (P50)**
```
herd_cluster █ 65.6us
seaweedfs_cluster ██ 131.0us
minio_cluster ████████████████████████ 1.3ms
rustfs_cluster ██████████████████████████████ 1.6ms
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 1.21 MB/s | 75.6us | 91.5us | 131.4us | 0 |
| seaweedfs_cluster | 0.15 MB/s | 566.7us | 1.1ms | 1.5ms | 0 |
| minio_cluster | 0.05 MB/s | 1.7ms | 2.2ms | 5.8ms | 0 |
| rustfs_cluster | 0.05 MB/s | 2.0ms | 2.6ms | 5.2ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 1.21 MB/s
seaweedfs_cluster ███ 0.15 MB/s
minio_cluster █ 0.05 MB/s
rustfs_cluster █ 0.05 MB/s
```

**Latency (P50)**
```
herd_cluster █ 75.6us
seaweedfs_cluster ████████ 566.7us
minio_cluster █████████████████████████ 1.7ms
rustfs_cluster ██████████████████████████████ 2.0ms
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 13229 ops/s | 72.2us | 87.9us | 118.4us | 0 |
| seaweedfs_cluster | 2904 ops/s | 287.5us | 699.2us | 949.6us | 0 |
| minio_cluster | 241 ops/s | 4.0ms | 7.8ms | 11.7ms | 0 |
| rustfs_cluster | 184 ops/s | 5.4ms | 10.4ms | 12.1ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 13229 ops/s
seaweedfs_cluster ██████ 2904 ops/s
minio_cluster █ 241 ops/s
rustfs_cluster █ 184 ops/s
```

**Latency (P50)**
```
herd_cluster █ 72.2us
seaweedfs_cluster █ 287.5us
minio_cluster ██████████████████████ 4.0ms
rustfs_cluster ██████████████████████████████ 5.4ms
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 1.15 MB/s | 79.7us | 94.5us | 127.8us | 0 |
| seaweedfs_cluster | 0.09 MB/s | 935.0us | 2.1ms | 4.1ms | 0 |
| minio_cluster | 0.05 MB/s | 1.7ms | 3.2ms | 6.6ms | 0 |
| rustfs_cluster | 0.04 MB/s | 2.0ms | 2.7ms | 6.5ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 1.15 MB/s
seaweedfs_cluster ██ 0.09 MB/s
minio_cluster █ 0.05 MB/s
rustfs_cluster █ 0.04 MB/s
```

**Latency (P50)**
```
herd_cluster █ 79.7us
seaweedfs_cluster ██████████████ 935.0us
minio_cluster ██████████████████████████ 1.7ms
rustfs_cluster ██████████████████████████████ 2.0ms
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 1876 ops/s | 493.7us | 742.7us | 1.1ms | 0 |
| seaweedfs_cluster | 923 ops/s | 928.7us | 1.8ms | 2.4ms | 0 |
| minio_cluster | 213 ops/s | 4.6ms | 5.3ms | 6.0ms | 0 |
| rustfs_cluster | 144 ops/s | 6.5ms | 8.6ms | 12.7ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 1876 ops/s
seaweedfs_cluster ██████████████ 923 ops/s
minio_cluster ███ 213 ops/s
rustfs_cluster ██ 144 ops/s
```

**Latency (P50)**
```
herd_cluster ██ 493.7us
seaweedfs_cluster ████ 928.7us
minio_cluster █████████████████████ 4.6ms
rustfs_cluster ██████████████████████████████ 6.5ms
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 3.70 MB/s | 2.6ms | 13.8ms | 26.3ms | 0 |
| seaweedfs_cluster | 1.06 MB/s | 10.9ms | 28.3ms | 50.0ms | 0 |
| rustfs_cluster | 0.07 MB/s | 191.0ms | 362.7ms | 425.8ms | 0 |
| minio_cluster | 0.02 MB/s | 74.7ms | 2.76s | 3.01s | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 3.70 MB/s
seaweedfs_cluster ████████ 1.06 MB/s
rustfs_cluster █ 0.07 MB/s
minio_cluster █ 0.02 MB/s
```

**Latency (P50)**
```
herd_cluster █ 2.6ms
seaweedfs_cluster █ 10.9ms
rustfs_cluster ██████████████████████████████ 191.0ms
minio_cluster ███████████ 74.7ms
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 3.96 MB/s | 2.2ms | 13.0ms | 30.2ms | 0 |
| seaweedfs_cluster | 1.37 MB/s | 6.0ms | 19.7ms | 137.3ms | 0 |
| minio_cluster | 0.26 MB/s | 28.4ms | 257.4ms | 675.7ms | 0 |
| rustfs_cluster | 0.17 MB/s | 90.8ms | 160.3ms | 180.1ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 3.96 MB/s
seaweedfs_cluster ██████████ 1.37 MB/s
minio_cluster █ 0.26 MB/s
rustfs_cluster █ 0.17 MB/s
```

**Latency (P50)**
```
herd_cluster █ 2.2ms
seaweedfs_cluster █ 6.0ms
minio_cluster █████████ 28.4ms
rustfs_cluster ██████████████████████████████ 90.8ms
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 2.95 MB/s | 3.1ms | 16.7ms | 38.2ms | 0 |
| seaweedfs_cluster | 0.58 MB/s | 22.3ms | 61.6ms | 107.5ms | 0 |
| rustfs_cluster | 0.04 MB/s | 415.5ms | 708.5ms | 761.9ms | 0 |
| minio_cluster | 0.03 MB/s | 511.2ms | 827.5ms | 828.4ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 2.95 MB/s
seaweedfs_cluster █████ 0.58 MB/s
rustfs_cluster █ 0.04 MB/s
minio_cluster █ 0.03 MB/s
```

**Latency (P50)**
```
herd_cluster █ 3.1ms
seaweedfs_cluster █ 22.3ms
rustfs_cluster ████████████████████████ 415.5ms
minio_cluster ██████████████████████████████ 511.2ms
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio_cluster | 280.00 MB/s | 51.3ms | 70.4ms | 71.7ms | 0 |
| seaweedfs_cluster | 180.69 MB/s | 81.7ms | 96.0ms | 96.0ms | 0 |
| rustfs_cluster | 162.60 MB/s | 94.0ms | 118.8ms | 127.8ms | 0 |
| herd_cluster | 0.00 MB/s | 0ns | 0ns | 0ns | 1101 |

**Throughput**
```
minio_cluster ██████████████████████████████ 280.00 MB/s
seaweedfs_cluster ███████████████████ 180.69 MB/s
rustfs_cluster █████████████████ 162.60 MB/s
herd_cluster  0.00 MB/s
```

**Latency (P50)**
```
minio_cluster ████████████████ 51.3ms
seaweedfs_cluster ██████████████████████████ 81.7ms
rustfs_cluster ██████████████████████████████ 94.0ms
herd_cluster  0ns
```

### ParallelRead/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 9.03 MB/s | 104.1us | 124.0us | 183.1us | 0 |
| seaweedfs_cluster | 4.24 MB/s | 213.5us | 338.0us | 467.5us | 0 |
| minio_cluster | 3.82 MB/s | 244.9us | 316.1us | 460.9us | 0 |
| rustfs_cluster | 2.53 MB/s | 377.6us | 450.4us | 525.2us | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 9.03 MB/s
seaweedfs_cluster ██████████████ 4.24 MB/s
minio_cluster ████████████ 3.82 MB/s
rustfs_cluster ████████ 2.53 MB/s
```

**Latency (P50)**
```
herd_cluster ████████ 104.1us
seaweedfs_cluster ████████████████ 213.5us
minio_cluster ███████████████████ 244.9us
rustfs_cluster ██████████████████████████████ 377.6us
```

### ParallelRead/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 3.19 MB/s | 265.3us | 545.5us | 956.2us | 0 |
| seaweedfs_cluster | 1.30 MB/s | 650.7us | 1.3ms | 2.5ms | 0 |
| minio_cluster | 1.29 MB/s | 640.6us | 1.4ms | 3.0ms | 0 |
| rustfs_cluster | 0.16 MB/s | 4.0ms | 17.8ms | 40.7ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 3.19 MB/s
seaweedfs_cluster ████████████ 1.30 MB/s
minio_cluster ████████████ 1.29 MB/s
rustfs_cluster █ 0.16 MB/s
```

**Latency (P50)**
```
herd_cluster █ 265.3us
seaweedfs_cluster ████ 650.7us
minio_cluster ████ 640.6us
rustfs_cluster ██████████████████████████████ 4.0ms
```

### ParallelRead/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 0.83 MB/s | 853.9us | 3.2ms | 5.5ms | 0 |
| seaweedfs_cluster | 0.38 MB/s | 2.2ms | 4.6ms | 8.8ms | 0 |
| minio_cluster | 0.12 MB/s | 4.9ms | 28.2ms | 54.6ms | 0 |
| rustfs_cluster | 0.06 MB/s | 16.7ms | 20.7ms | 26.5ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 0.83 MB/s
seaweedfs_cluster █████████████ 0.38 MB/s
minio_cluster ████ 0.12 MB/s
rustfs_cluster ██ 0.06 MB/s
```

**Latency (P50)**
```
herd_cluster █ 853.9us
seaweedfs_cluster ███ 2.2ms
minio_cluster ████████ 4.9ms
rustfs_cluster ██████████████████████████████ 16.7ms
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 8.33 MB/s | 112.8us | 135.9us | 192.3us | 0 |
| rustfs_cluster | 0.63 MB/s | 1.5ms | 2.0ms | 4.5ms | 0 |
| minio_cluster | 0.59 MB/s | 1.6ms | 2.2ms | 4.5ms | 0 |
| seaweedfs_cluster | 0.52 MB/s | 1.5ms | 5.2ms | 7.1ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 8.33 MB/s
rustfs_cluster ██ 0.63 MB/s
minio_cluster ██ 0.59 MB/s
seaweedfs_cluster █ 0.52 MB/s
```

**Latency (P50)**
```
herd_cluster ██ 112.8us
rustfs_cluster ███████████████████████████ 1.5ms
minio_cluster ██████████████████████████████ 1.6ms
seaweedfs_cluster ████████████████████████████ 1.5ms
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 3.08 MB/s | 275.8us | 585.0us | 1.0ms | 0 |
| seaweedfs_cluster | 0.83 MB/s | 1.1ms | 1.7ms | 2.9ms | 0 |
| minio_cluster | 0.05 MB/s | 16.9ms | 44.7ms | 79.5ms | 0 |
| rustfs_cluster | 0.03 MB/s | 24.0ms | 59.9ms | 78.5ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 3.08 MB/s
seaweedfs_cluster ████████ 0.83 MB/s
minio_cluster █ 0.05 MB/s
rustfs_cluster █ 0.03 MB/s
```

**Latency (P50)**
```
herd_cluster █ 275.8us
seaweedfs_cluster █ 1.1ms
minio_cluster █████████████████████ 16.9ms
rustfs_cluster ██████████████████████████████ 24.0ms
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 0.82 MB/s | 872.8us | 3.2ms | 5.4ms | 0 |
| seaweedfs_cluster | 0.22 MB/s | 4.2ms | 6.8ms | 9.2ms | 0 |
| rustfs_cluster | 0.01 MB/s | 91.6ms | 147.5ms | 180.6ms | 0 |
| minio_cluster | 0.01 MB/s | 134.5ms | 300.5ms | 419.6ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 0.82 MB/s
seaweedfs_cluster ████████ 0.22 MB/s
rustfs_cluster █ 0.01 MB/s
minio_cluster █ 0.01 MB/s
```

**Latency (P50)**
```
herd_cluster █ 872.8us
seaweedfs_cluster █ 4.2ms
rustfs_cluster ████████████████████ 91.6ms
minio_cluster ██████████████████████████████ 134.5ms
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 2708.69 MB/s | 86.9us | 115.9us | 168.5us | 0 |
| seaweedfs_cluster | 1007.22 MB/s | 241.2us | 292.9us | 347.5us | 0 |
| minio_cluster | 631.99 MB/s | 367.3us | 440.9us | 1.1ms | 0 |
| rustfs_cluster | 167.92 MB/s | 1.5ms | 1.7ms | 1.8ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 2708.69 MB/s
seaweedfs_cluster ███████████ 1007.22 MB/s
minio_cluster ██████ 631.99 MB/s
rustfs_cluster █ 167.92 MB/s
```

**Latency (P50)**
```
herd_cluster █ 86.9us
seaweedfs_cluster ████ 241.2us
minio_cluster ███████ 367.3us
rustfs_cluster ██████████████████████████████ 1.5ms
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 2744.77 MB/s | 86.6us | 112.3us | 151.5us | 0 |
| seaweedfs_cluster | 961.50 MB/s | 244.0us | 333.7us | 564.2us | 0 |
| minio_cluster | 631.47 MB/s | 367.7us | 444.4us | 1.0ms | 0 |
| rustfs_cluster | 165.81 MB/s | 1.5ms | 1.7ms | 1.9ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 2744.77 MB/s
seaweedfs_cluster ██████████ 961.50 MB/s
minio_cluster ██████ 631.47 MB/s
rustfs_cluster █ 165.81 MB/s
```

**Latency (P50)**
```
herd_cluster █ 86.6us
seaweedfs_cluster ████ 244.0us
minio_cluster ███████ 367.7us
rustfs_cluster ██████████████████████████████ 1.5ms
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 2657.64 MB/s | 88.1us | 117.7us | 169.8us | 0 |
| seaweedfs_cluster | 979.56 MB/s | 244.7us | 316.2us | 405.2us | 0 |
| minio_cluster | 669.19 MB/s | 346.5us | 431.2us | 1.0ms | 0 |
| rustfs_cluster | 161.61 MB/s | 1.5ms | 1.8ms | 2.7ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 2657.64 MB/s
seaweedfs_cluster ███████████ 979.56 MB/s
minio_cluster ███████ 669.19 MB/s
rustfs_cluster █ 161.61 MB/s
```

**Latency (P50)**
```
herd_cluster █ 88.1us
seaweedfs_cluster ████ 244.7us
minio_cluster ██████ 346.5us
rustfs_cluster ██████████████████████████████ 1.5ms
```

### Read/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 4818.88 MB/s | 2.0ms | 3.0ms | 3.4ms | 0 |
| minio_cluster | 2378.21 MB/s | 3.5ms | 8.8ms | 9.9ms | 0 |
| seaweedfs_cluster | 2144.52 MB/s | 4.6ms | 5.2ms | 5.4ms | 0 |
| rustfs_cluster | 1915.92 MB/s | 5.1ms | 5.8ms | 6.1ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 4818.88 MB/s
minio_cluster ██████████████ 2378.21 MB/s
seaweedfs_cluster █████████████ 2144.52 MB/s
rustfs_cluster ███████████ 1915.92 MB/s
```

**Latency (P50)**
```
herd_cluster ███████████ 2.0ms
minio_cluster ████████████████████ 3.5ms
seaweedfs_cluster ██████████████████████████ 4.6ms
rustfs_cluster ██████████████████████████████ 5.1ms
```

### Read/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 10.00 MB/s | 84.5us | 166.0us | 280.0us | 0 |
| minio_cluster | 4.80 MB/s | 198.3us | 231.8us | 264.6us | 0 |
| rustfs_cluster | 2.61 MB/s | 313.8us | 558.1us | 1.2ms | 0 |
| seaweedfs_cluster | 2.35 MB/s | 202.7us | 1.0ms | 2.5ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 10.00 MB/s
minio_cluster ██████████████ 4.80 MB/s
rustfs_cluster ███████ 2.61 MB/s
seaweedfs_cluster ███████ 2.35 MB/s
```

**Latency (P50)**
```
herd_cluster ████████ 84.5us
minio_cluster ██████████████████ 198.3us
rustfs_cluster ██████████████████████████████ 313.8us
seaweedfs_cluster ███████████████████ 202.7us
```

### Read/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 2193.53 MB/s | 331.5us | 991.7us | 2.5ms | 0 |
| seaweedfs_cluster | 1974.40 MB/s | 487.7us | 604.0us | 897.4us | 0 |
| rustfs_cluster | 782.89 MB/s | 1.3ms | 1.5ms | 1.8ms | 0 |
| minio_cluster | 538.46 MB/s | 1.5ms | 3.9ms | 5.6ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 2193.53 MB/s
seaweedfs_cluster ███████████████████████████ 1974.40 MB/s
rustfs_cluster ██████████ 782.89 MB/s
minio_cluster ███████ 538.46 MB/s
```

**Latency (P50)**
```
herd_cluster ██████ 331.5us
seaweedfs_cluster █████████ 487.7us
rustfs_cluster ████████████████████████ 1.3ms
minio_cluster ██████████████████████████████ 1.5ms
```

### Read/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 535.58 MB/s | 98.6us | 222.8us | 326.8us | 0 |
| seaweedfs_cluster | 303.95 MB/s | 197.1us | 243.1us | 448.7us | 0 |
| minio_cluster | 227.84 MB/s | 213.8us | 632.5us | 1.0ms | 0 |
| rustfs_cluster | 157.49 MB/s | 375.0us | 528.2us | 697.8us | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 535.58 MB/s
seaweedfs_cluster █████████████████ 303.95 MB/s
minio_cluster ████████████ 227.84 MB/s
rustfs_cluster ████████ 157.49 MB/s
```

**Latency (P50)**
```
herd_cluster ███████ 98.6us
seaweedfs_cluster ███████████████ 197.1us
minio_cluster █████████████████ 213.8us
rustfs_cluster ██████████████████████████████ 375.0us
```

### Scale/Delete/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 12585 ops/s | 79.5us | 79.5us | 79.5us | 0 |
| seaweedfs_cluster | 3256 ops/s | 307.1us | 307.1us | 307.1us | 0 |
| rustfs_cluster | 348 ops/s | 2.9ms | 2.9ms | 2.9ms | 0 |
| minio_cluster | 158 ops/s | 6.3ms | 6.3ms | 6.3ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 12585 ops/s
seaweedfs_cluster ███████ 3256 ops/s
rustfs_cluster █ 348 ops/s
minio_cluster █ 158 ops/s
```

**Latency (P50)**
```
herd_cluster █ 79.5us
seaweedfs_cluster █ 307.1us
rustfs_cluster █████████████ 2.9ms
minio_cluster ██████████████████████████████ 6.3ms
```

### Scale/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 1414 ops/s | 707.5us | 707.5us | 707.5us | 0 |
| seaweedfs_cluster | 503 ops/s | 2.0ms | 2.0ms | 2.0ms | 0 |
| minio_cluster | 67 ops/s | 14.9ms | 14.9ms | 14.9ms | 0 |
| rustfs_cluster | 36 ops/s | 28.0ms | 28.0ms | 28.0ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 1414 ops/s
seaweedfs_cluster ██████████ 503 ops/s
minio_cluster █ 67 ops/s
rustfs_cluster █ 36 ops/s
```

**Latency (P50)**
```
herd_cluster █ 707.5us
seaweedfs_cluster ██ 2.0ms
minio_cluster ███████████████ 14.9ms
rustfs_cluster ██████████████████████████████ 28.0ms
```

### Scale/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 130 ops/s | 7.7ms | 7.7ms | 7.7ms | 0 |
| seaweedfs_cluster | 48 ops/s | 21.0ms | 21.0ms | 21.0ms | 0 |
| minio_cluster | 7 ops/s | 147.6ms | 147.6ms | 147.6ms | 0 |
| rustfs_cluster | 2 ops/s | 566.0ms | 566.0ms | 566.0ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 130 ops/s
seaweedfs_cluster ██████████ 48 ops/s
minio_cluster █ 7 ops/s
rustfs_cluster █ 2 ops/s
```

**Latency (P50)**
```
herd_cluster █ 7.7ms
seaweedfs_cluster █ 21.0ms
minio_cluster ███████ 147.6ms
rustfs_cluster ██████████████████████████████ 566.0ms
```

### Scale/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 15 ops/s | 66.8ms | 66.8ms | 66.8ms | 0 |
| seaweedfs_cluster | 2 ops/s | 414.8ms | 414.8ms | 414.8ms | 0 |
| minio_cluster | 0 ops/s | 2.03s | 2.03s | 2.03s | 0 |
| rustfs_cluster | 0 ops/s | 3.06s | 3.06s | 3.06s | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 15 ops/s
seaweedfs_cluster ████ 2 ops/s
minio_cluster █ 0 ops/s
rustfs_cluster █ 0 ops/s
```

**Latency (P50)**
```
herd_cluster █ 66.8ms
seaweedfs_cluster ████ 414.8ms
minio_cluster ███████████████████ 2.03s
rustfs_cluster ██████████████████████████████ 3.06s
```

### Scale/List/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 7926 ops/s | 126.2us | 126.2us | 126.2us | 0 |
| seaweedfs_cluster | 1877 ops/s | 532.7us | 532.7us | 532.7us | 0 |
| minio_cluster | 1120 ops/s | 892.8us | 892.8us | 892.8us | 0 |
| rustfs_cluster | 551 ops/s | 1.8ms | 1.8ms | 1.8ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 7926 ops/s
seaweedfs_cluster ███████ 1877 ops/s
minio_cluster ████ 1120 ops/s
rustfs_cluster ██ 551 ops/s
```

**Latency (P50)**
```
herd_cluster ██ 126.2us
seaweedfs_cluster ████████ 532.7us
minio_cluster ██████████████ 892.8us
rustfs_cluster ██████████████████████████████ 1.8ms
```

### Scale/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 7297 ops/s | 137.0us | 137.0us | 137.0us | 0 |
| seaweedfs_cluster | 896 ops/s | 1.1ms | 1.1ms | 1.1ms | 0 |
| minio_cluster | 498 ops/s | 2.0ms | 2.0ms | 2.0ms | 0 |
| rustfs_cluster | 260 ops/s | 3.8ms | 3.8ms | 3.8ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 7297 ops/s
seaweedfs_cluster ███ 896 ops/s
minio_cluster ██ 498 ops/s
rustfs_cluster █ 260 ops/s
```

**Latency (P50)**
```
herd_cluster █ 137.0us
seaweedfs_cluster ████████ 1.1ms
minio_cluster ███████████████ 2.0ms
rustfs_cluster ██████████████████████████████ 3.8ms
```

### Scale/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 2121 ops/s | 471.4us | 471.4us | 471.4us | 0 |
| seaweedfs_cluster | 834 ops/s | 1.2ms | 1.2ms | 1.2ms | 0 |
| minio_cluster | 88 ops/s | 11.4ms | 11.4ms | 11.4ms | 0 |
| rustfs_cluster | 47 ops/s | 21.4ms | 21.4ms | 21.4ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 2121 ops/s
seaweedfs_cluster ███████████ 834 ops/s
minio_cluster █ 88 ops/s
rustfs_cluster █ 47 ops/s
```

**Latency (P50)**
```
herd_cluster █ 471.4us
seaweedfs_cluster █ 1.2ms
minio_cluster ███████████████ 11.4ms
rustfs_cluster ██████████████████████████████ 21.4ms
```

### Scale/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 272 ops/s | 3.7ms | 3.7ms | 3.7ms | 0 |
| seaweedfs_cluster | 122 ops/s | 8.2ms | 8.2ms | 8.2ms | 0 |
| minio_cluster | 10 ops/s | 99.6ms | 99.6ms | 99.6ms | 0 |
| rustfs_cluster | 6 ops/s | 179.8ms | 179.8ms | 179.8ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 272 ops/s
seaweedfs_cluster █████████████ 122 ops/s
minio_cluster █ 10 ops/s
rustfs_cluster █ 6 ops/s
```

**Latency (P50)**
```
herd_cluster █ 3.7ms
seaweedfs_cluster █ 8.2ms
minio_cluster ████████████████ 99.6ms
rustfs_cluster ██████████████████████████████ 179.8ms
```

### Scale/Write/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 2.71 MB/s | 90.2us | 90.2us | 90.2us | 0 |
| seaweedfs_cluster | 0.40 MB/s | 610.7us | 610.7us | 610.7us | 0 |
| rustfs_cluster | 0.11 MB/s | 2.1ms | 2.1ms | 2.1ms | 0 |
| minio_cluster | 0.11 MB/s | 2.2ms | 2.2ms | 2.2ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 2.71 MB/s
seaweedfs_cluster ████ 0.40 MB/s
rustfs_cluster █ 0.11 MB/s
minio_cluster █ 0.11 MB/s
```

**Latency (P50)**
```
herd_cluster █ 90.2us
seaweedfs_cluster ████████ 610.7us
rustfs_cluster ████████████████████████████ 2.1ms
minio_cluster ██████████████████████████████ 2.2ms
```

### Scale/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 2.89 MB/s | 846.2us | 846.2us | 846.2us | 0 |
| seaweedfs_cluster | 0.38 MB/s | 6.5ms | 6.5ms | 6.5ms | 0 |
| minio_cluster | 0.13 MB/s | 18.4ms | 18.4ms | 18.4ms | 0 |
| rustfs_cluster | 0.12 MB/s | 20.5ms | 20.5ms | 20.5ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 2.89 MB/s
seaweedfs_cluster ███ 0.38 MB/s
minio_cluster █ 0.13 MB/s
rustfs_cluster █ 0.12 MB/s
```

**Latency (P50)**
```
herd_cluster █ 846.2us
seaweedfs_cluster █████████ 6.5ms
minio_cluster ██████████████████████████ 18.4ms
rustfs_cluster ██████████████████████████████ 20.5ms
```

### Scale/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 3.24 MB/s | 7.5ms | 7.5ms | 7.5ms | 0 |
| seaweedfs_cluster | 0.46 MB/s | 52.8ms | 52.8ms | 52.8ms | 0 |
| minio_cluster | 0.12 MB/s | 200.4ms | 200.4ms | 200.4ms | 0 |
| rustfs_cluster | 0.09 MB/s | 286.3ms | 286.3ms | 286.3ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 3.24 MB/s
seaweedfs_cluster ████ 0.46 MB/s
minio_cluster █ 0.12 MB/s
rustfs_cluster █ 0.09 MB/s
```

**Latency (P50)**
```
herd_cluster █ 7.5ms
seaweedfs_cluster █████ 52.8ms
minio_cluster █████████████████████ 200.4ms
rustfs_cluster ██████████████████████████████ 286.3ms
```

### Scale/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 3.20 MB/s | 76.2ms | 76.2ms | 76.2ms | 0 |
| seaweedfs_cluster | 0.42 MB/s | 576.3ms | 576.3ms | 576.3ms | 0 |
| minio_cluster | 0.13 MB/s | 1.89s | 1.89s | 1.89s | 0 |
| rustfs_cluster | 0.11 MB/s | 2.30s | 2.30s | 2.30s | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 3.20 MB/s
seaweedfs_cluster ███ 0.42 MB/s
minio_cluster █ 0.13 MB/s
rustfs_cluster █ 0.11 MB/s
```

**Latency (P50)**
```
herd_cluster █ 76.2ms
seaweedfs_cluster ███████ 576.3ms
minio_cluster ████████████████████████ 1.89s
rustfs_cluster ██████████████████████████████ 2.30s
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 9988 ops/s | 84.8us | 178.9us | 284.4us | 0 |
| seaweedfs_cluster | 7963 ops/s | 121.8us | 149.2us | 192.3us | 0 |
| minio_cluster | 4872 ops/s | 191.1us | 269.6us | 425.4us | 0 |
| rustfs_cluster | 4302 ops/s | 225.3us | 286.2us | 374.5us | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 9988 ops/s
seaweedfs_cluster ███████████████████████ 7963 ops/s
minio_cluster ██████████████ 4872 ops/s
rustfs_cluster ████████████ 4302 ops/s
```

**Latency (P50)**
```
herd_cluster ███████████ 84.8us
seaweedfs_cluster ████████████████ 121.8us
minio_cluster █████████████████████████ 191.1us
rustfs_cluster ██████████████████████████████ 225.3us
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 652.01 MB/s | 12.7ms | 29.3ms | 34.4ms | 0 |
| minio_cluster | 330.94 MB/s | 28.6ms | 34.0ms | 39.4ms | 0 |
| rustfs_cluster | 280.40 MB/s | 33.1ms | 43.5ms | 51.8ms | 0 |
| seaweedfs_cluster | 186.76 MB/s | 51.0ms | 68.6ms | 74.4ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 652.01 MB/s
minio_cluster ███████████████ 330.94 MB/s
rustfs_cluster ████████████ 280.40 MB/s
seaweedfs_cluster ████████ 186.76 MB/s
```

**Latency (P50)**
```
herd_cluster ███████ 12.7ms
minio_cluster ████████████████ 28.6ms
rustfs_cluster ███████████████████ 33.1ms
seaweedfs_cluster ██████████████████████████████ 51.0ms
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 7.75 MB/s | 108.8us | 234.8us | 349.8us | 0 |
| seaweedfs_cluster | 2.77 MB/s | 323.6us | 479.7us | 680.1us | 0 |
| rustfs_cluster | 0.53 MB/s | 1.5ms | 5.3ms | 7.1ms | 0 |
| minio_cluster | 0.39 MB/s | 1.7ms | 7.2ms | 8.3ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 7.75 MB/s
seaweedfs_cluster ██████████ 2.77 MB/s
rustfs_cluster ██ 0.53 MB/s
minio_cluster █ 0.39 MB/s
```

**Latency (P50)**
```
herd_cluster █ 108.8us
seaweedfs_cluster █████ 323.6us
rustfs_cluster █████████████████████████ 1.5ms
minio_cluster ██████████████████████████████ 1.7ms
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 618.53 MB/s | 1.1ms | 3.8ms | 5.4ms | 0 |
| seaweedfs_cluster | 209.70 MB/s | 4.5ms | 5.2ms | 10.6ms | 0 |
| rustfs_cluster | 148.55 MB/s | 6.0ms | 10.2ms | 16.6ms | 0 |
| minio_cluster | 96.85 MB/s | 9.6ms | 18.0ms | 27.6ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 618.53 MB/s
seaweedfs_cluster ██████████ 209.70 MB/s
rustfs_cluster ███████ 148.55 MB/s
minio_cluster ████ 96.85 MB/s
```

**Latency (P50)**
```
herd_cluster ███ 1.1ms
seaweedfs_cluster ██████████████ 4.5ms
rustfs_cluster ██████████████████ 6.0ms
minio_cluster ██████████████████████████████ 9.6ms
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| herd_cluster | 290.06 MB/s | 179.8us | 382.3us | 660.2us | 0 |
| seaweedfs_cluster | 103.10 MB/s | 560.2us | 707.3us | 1.1ms | 0 |
| rustfs_cluster | 28.13 MB/s | 1.7ms | 6.0ms | 7.5ms | 0 |
| minio_cluster | 19.83 MB/s | 2.0ms | 7.3ms | 8.2ms | 0 |

**Throughput**
```
herd_cluster ██████████████████████████████ 290.06 MB/s
seaweedfs_cluster ██████████ 103.10 MB/s
rustfs_cluster ██ 28.13 MB/s
minio_cluster ██ 19.83 MB/s
```

**Latency (P50)**
```
herd_cluster ██ 179.8us
seaweedfs_cluster ████████ 560.2us
rustfs_cluster █████████████████████████ 1.7ms
minio_cluster ██████████████████████████████ 2.0ms
```

## Recommendations

- **Write-heavy workloads:** herd_cluster
- **Read-heavy workloads:** herd_cluster

---

*Generated by storage benchmark CLI*
