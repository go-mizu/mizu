# Storage Benchmark Report

**Generated:** 2026-01-21T18:54:52+07:00

**Go Version:** go1.25.5

**Platform:** darwin/arm64

## Executive Summary

### Summary

**Overall Winner:** usagi_s3 (won 36/48 benchmarks, 75%)

| Rank | Driver | Wins | Win Rate |
|------|--------|------|----------|
| 1 | usagi_s3 | 36 | 75% |
| 2 | devnull_s3 | 7 | 15% |
| 3 | minio | 5 | 10% |

### Performance Leaders

| Operation | Leader | Performance | Margin |
|-----------|--------|-------------|--------|
| Small Read (1KB) | usagi_s3 | 4.5 MB/s | +13% vs devnull_s3 |
| Small Write (1KB) | usagi_s3 | 1.8 MB/s | +19% vs devnull_s3 |
| Large Read (100MB) | minio | 237.4 MB/s | +36% vs usagi_s3 |
| Large Write (100MB) | usagi_s3 | 152.3 MB/s | close |
| Delete | usagi_s3 | 4.2K ops/s | close |
| Stat | devnull_s3 | 4.2K ops/s | +16% vs usagi_s3 |
| List (100 objects) | usagi_s3 | 1.0K ops/s | close |
| Copy | usagi_s3 | 1.4 MB/s | close |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (100MB+) | **usagi_s3** | 152 MB/s | Best for media, backups |
| Large File Downloads (100MB) | **minio** | 237 MB/s | Best for streaming, CDN |
| Small File Operations | **usagi_s3** | 3248 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **usagi_s3** | - | Best for multi-user apps |
| Memory Constrained | **minio** | 651 MB RAM | Best for edge/embedded |

### Large File Performance (100MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| devnull_s3 | 123.7 | 165.2 | 818.0ms | 612.7ms |
| minio | 146.8 | 237.4 | 681.8ms | 417.7ms |
| usagi_s3 | 152.3 | 174.0 | 654.3ms | 577.9ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| devnull_s3 | 1565 | 4097 | 589.5us | 231.0us |
| minio | 1128 | 2511 | 819.1us | 328.0us |
| usagi_s3 | 1857 | 4639 | 512.2us | 202.1us |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| devnull_s3 | 4229 | 999 | 4095 |
| minio | 3203 | 461 | 2705 |
| usagi_s3 | 3634 | 1049 | 4228 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| devnull_s3 | 1.38 | 0.40 | 0.19 | 0.11 | 0.06 | 0.03 |
| minio | 1.25 | 0.25 | 0.09 | 0.05 | 0.02 | 0.01 |
| usagi_s3 | 1.59 | 0.43 | 0.20 | 0.11 | 0.05 | 0.03 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| devnull_s3 | 3.52 | 1.15 | 0.60 | 0.34 | 0.19 | 0.10 |
| minio | 2.61 | 0.85 | 0.40 | 0.20 | 0.10 | 0.05 |
| usagi_s3 | 3.89 | 1.18 | 0.61 | 0.37 | 0.19 | 0.10 |

*\* indicates errors occurred*

### Scale Performance

Performance with varying numbers of objects (256B each).

**Write N Files (total time)**

| Driver | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|
| devnull_s3 | 7.0ms | 59.0ms | 602.4ms | 5.95s |
| minio | 12.9ms | 118.8ms | 1.30s | 10.95s |
| usagi_s3 | 14.1ms | 81.5ms | 545.7ms | 5.77s |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|
| devnull_s3 | 449.4us | 1.0ms | 6.1ms | 208.4ms |
| minio | 1.2ms | 2.1ms | 15.5ms | 179.5ms |
| usagi_s3 | 487.2us | 1.0ms | 5.8ms | 234.4ms |

*\* indicates errors occurred*

### Resource Usage Summary

| Driver | Memory | CPU |
|--------|--------|-----|
| devnull_s3 | 1409.0 MB | 2.1% |
| minio | 650.6 MB | 7.4% |
| usagi_s3 | 1628.2 MB | 3.7% |

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
| usagi_s3 | 1.40 MB/s | 562.2us | 1.4ms | 2.3ms | 0 |
| devnull_s3 | 1.36 MB/s | 631.3us | 1.3ms | 1.9ms | 0 |
| minio | 0.80 MB/s | 1.1ms | 1.9ms | 2.5ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 1.40 MB/s
devnull_s3   █████████████████████████████ 1.36 MB/s
minio        █████████████████ 0.80 MB/s
```

**Latency (P50)**
```
usagi_s3     ██████████████ 562.2us
devnull_s3   ████████████████ 631.3us
minio        ██████████████████████████████ 1.1ms
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 4228 ops/s | 217.8us | 329.0us | 519.6us | 0 |
| devnull_s3 | 4095 ops/s | 227.3us | 329.8us | 507.8us | 0 |
| minio | 2705 ops/s | 352.7us | 472.3us | 667.8us | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 4228 ops/s
devnull_s3   █████████████████████████████ 4095 ops/s
minio        ███████████████████ 2705 ops/s
```

**Latency (P50)**
```
usagi_s3     ██████████████████ 217.8us
devnull_s3   ███████████████████ 227.3us
minio        ██████████████████████████████ 352.7us
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.17 MB/s | 526.3us | 680.2us | 931.8us | 0 |
| devnull_s3 | 0.15 MB/s | 604.5us | 896.0us | 1.2ms | 0 |
| minio | 0.07 MB/s | 1.2ms | 1.8ms | 3.0ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.17 MB/s
devnull_s3   █████████████████████████ 0.15 MB/s
minio        ████████████ 0.07 MB/s
```

**Latency (P50)**
```
usagi_s3     █████████████ 526.3us
devnull_s3   ███████████████ 604.5us
minio        ██████████████████████████████ 1.2ms
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 2009 ops/s | 462.9us | 665.2us | 1.2ms | 0 |
| devnull_s3 | 1771 ops/s | 485.0us | 774.0us | 1.1ms | 0 |
| minio | 845 ops/s | 1.0ms | 2.0ms | 3.4ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 2009 ops/s
devnull_s3   ██████████████████████████ 1771 ops/s
minio        ████████████ 845 ops/s
```

**Latency (P50)**
```
usagi_s3     █████████████ 462.9us
devnull_s3   ██████████████ 485.0us
minio        ██████████████████████████████ 1.0ms
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.17 MB/s | 531.3us | 733.0us | 991.2us | 0 |
| devnull_s3 | 0.15 MB/s | 577.5us | 937.4us | 1.6ms | 0 |
| minio | 0.07 MB/s | 1.2ms | 1.9ms | 3.1ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.17 MB/s
devnull_s3   ███████████████████████████ 0.15 MB/s
minio        ████████████ 0.07 MB/s
```

**Latency (P50)**
```
usagi_s3     ████████████ 531.3us
devnull_s3   █████████████ 577.5us
minio        ██████████████████████████████ 1.2ms
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 1049 ops/s | 922.4us | 1.2ms | 1.7ms | 0 |
| devnull_s3 | 999 ops/s | 986.4us | 1.1ms | 1.4ms | 0 |
| minio | 461 ops/s | 1.9ms | 4.0ms | 6.1ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 1049 ops/s
devnull_s3   ████████████████████████████ 999 ops/s
minio        █████████████ 461 ops/s
```

**Latency (P50)**
```
usagi_s3     ██████████████ 922.4us
devnull_s3   ███████████████ 986.4us
minio        ██████████████████████████████ 1.9ms
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.53 MB/s | 23.8ms | 49.8ms | 64.0ms | 0 |
| devnull_s3 | 0.48 MB/s | 23.7ms | 50.2ms | 249.1ms | 0 |
| minio | 0.26 MB/s | 42.2ms | 140.3ms | 197.5ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.53 MB/s
devnull_s3   ██████████████████████████ 0.48 MB/s
minio        ███████████████ 0.26 MB/s
```

**Latency (P50)**
```
usagi_s3     ████████████████ 23.8ms
devnull_s3   ████████████████ 23.7ms
minio        ██████████████████████████████ 42.2ms
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.63 MB/s | 24.4ms | 36.2ms | 50.4ms | 0 |
| devnull_s3 | 0.61 MB/s | 25.2ms | 37.7ms | 44.0ms | 0 |
| minio | 0.41 MB/s | 29.9ms | 102.4ms | 157.1ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.63 MB/s
devnull_s3   █████████████████████████████ 0.61 MB/s
minio        ███████████████████ 0.41 MB/s
```

**Latency (P50)**
```
usagi_s3     ████████████████████████ 24.4ms
devnull_s3   █████████████████████████ 25.2ms
minio        ██████████████████████████████ 29.9ms
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.42 MB/s | 31.3ms | 61.7ms | 101.4ms | 0 |
| devnull_s3 | 0.39 MB/s | 31.5ms | 59.0ms | 300.1ms | 0 |
| minio | 0.14 MB/s | 102.6ms | 247.7ms | 325.4ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.42 MB/s
devnull_s3   ███████████████████████████ 0.39 MB/s
minio        █████████ 0.14 MB/s
```

**Latency (P50)**
```
usagi_s3     █████████ 31.3ms
devnull_s3   █████████ 31.5ms
minio        ██████████████████████████████ 102.6ms
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 125.88 MB/s | 119.1ms | 128.7ms | 128.7ms | 0 |
| minio | 112.85 MB/s | 127.5ms | 148.8ms | 148.8ms | 0 |
| devnull_s3 | 104.72 MB/s | 141.8ms | 149.9ms | 149.9ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 125.88 MB/s
minio        ██████████████████████████ 112.85 MB/s
devnull_s3   ████████████████████████ 104.72 MB/s
```

**Latency (P50)**
```
usagi_s3     █████████████████████████ 119.1ms
minio        ██████████████████████████ 127.5ms
devnull_s3   ██████████████████████████████ 141.8ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 3.89 MB/s | 240.3us | 318.6us | 408.6us | 0 |
| devnull_s3 | 3.52 MB/s | 262.7us | 361.0us | 464.8us | 0 |
| minio | 2.61 MB/s | 350.4us | 471.9us | 909.3us | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 3.89 MB/s
devnull_s3   ███████████████████████████ 3.52 MB/s
minio        ████████████████████ 2.61 MB/s
```

**Latency (P50)**
```
usagi_s3     ████████████████████ 240.3us
devnull_s3   ██████████████████████ 262.7us
minio        ██████████████████████████████ 350.4us
```

### ParallelRead/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 1.18 MB/s | 788.2us | 1.2ms | 1.7ms | 0 |
| devnull_s3 | 1.15 MB/s | 825.5us | 1.2ms | 1.5ms | 0 |
| minio | 0.85 MB/s | 1.1ms | 1.9ms | 2.9ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 1.18 MB/s
devnull_s3   █████████████████████████████ 1.15 MB/s
minio        █████████████████████ 0.85 MB/s
```

**Latency (P50)**
```
usagi_s3     ██████████████████████ 788.2us
devnull_s3   ███████████████████████ 825.5us
minio        ██████████████████████████████ 1.1ms
```

### ParallelRead/1KB/C100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.19 MB/s | 4.8ms | 7.9ms | 11.3ms | 0 |
| devnull_s3 | 0.19 MB/s | 5.0ms | 8.3ms | 11.4ms | 0 |
| minio | 0.10 MB/s | 9.0ms | 20.5ms | 31.4ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.19 MB/s
devnull_s3   ████████████████████████████ 0.19 MB/s
minio        ██████████████ 0.10 MB/s
```

**Latency (P50)**
```
usagi_s3     ████████████████ 4.8ms
devnull_s3   ████████████████ 5.0ms
minio        ██████████████████████████████ 9.0ms
```

### ParallelRead/1KB/C200

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.10 MB/s | 9.8ms | 15.1ms | 21.1ms | 0 |
| devnull_s3 | 0.10 MB/s | 10.2ms | 14.8ms | 18.4ms | 0 |
| minio | 0.05 MB/s | 18.4ms | 35.8ms | 48.0ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.10 MB/s
devnull_s3   █████████████████████████████ 0.10 MB/s
minio        ███████████████ 0.05 MB/s
```

**Latency (P50)**
```
usagi_s3     ███████████████ 9.8ms
devnull_s3   ████████████████ 10.2ms
minio        ██████████████████████████████ 18.4ms
```

### ParallelRead/1KB/C25

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.61 MB/s | 1.5ms | 2.6ms | 3.7ms | 0 |
| devnull_s3 | 0.60 MB/s | 1.6ms | 2.6ms | 3.5ms | 0 |
| minio | 0.40 MB/s | 2.2ms | 4.1ms | 5.7ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.61 MB/s
devnull_s3   █████████████████████████████ 0.60 MB/s
minio        ███████████████████ 0.40 MB/s
```

**Latency (P50)**
```
usagi_s3     ████████████████████ 1.5ms
devnull_s3   ████████████████████ 1.6ms
minio        ██████████████████████████████ 2.2ms
```

### ParallelRead/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.37 MB/s | 2.5ms | 4.0ms | 5.3ms | 0 |
| devnull_s3 | 0.34 MB/s | 2.7ms | 4.8ms | 7.9ms | 0 |
| minio | 0.20 MB/s | 4.4ms | 9.3ms | 15.6ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.37 MB/s
devnull_s3   ███████████████████████████ 0.34 MB/s
minio        ████████████████ 0.20 MB/s
```

**Latency (P50)**
```
usagi_s3     █████████████████ 2.5ms
devnull_s3   █████████████████ 2.7ms
minio        ██████████████████████████████ 4.4ms
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 1.59 MB/s | 586.6us | 772.2us | 1.0ms | 0 |
| devnull_s3 | 1.38 MB/s | 649.9us | 990.7us | 1.6ms | 0 |
| minio | 1.25 MB/s | 764.5us | 943.3us | 1.1ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 1.59 MB/s
devnull_s3   ██████████████████████████ 1.38 MB/s
minio        ███████████████████████ 1.25 MB/s
```

**Latency (P50)**
```
usagi_s3     ███████████████████████ 586.6us
devnull_s3   █████████████████████████ 649.9us
minio        ██████████████████████████████ 764.5us
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.43 MB/s | 2.1ms | 3.5ms | 7.9ms | 0 |
| devnull_s3 | 0.40 MB/s | 2.2ms | 3.4ms | 7.0ms | 0 |
| minio | 0.25 MB/s | 3.6ms | 6.6ms | 9.2ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.43 MB/s
devnull_s3   ████████████████████████████ 0.40 MB/s
minio        █████████████████ 0.25 MB/s
```

**Latency (P50)**
```
usagi_s3     █████████████████ 2.1ms
devnull_s3   ██████████████████ 2.2ms
minio        ██████████████████████████████ 3.6ms
```

### ParallelWrite/1KB/C100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 0.06 MB/s | 16.0ms | 33.9ms | 45.1ms | 0 |
| usagi_s3 | 0.05 MB/s | 15.9ms | 37.7ms | 54.6ms | 0 |
| minio | 0.02 MB/s | 48.2ms | 96.1ms | 125.1ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 0.06 MB/s
usagi_s3     █████████████████████████████ 0.05 MB/s
minio        ██████████ 0.02 MB/s
```

**Latency (P50)**
```
devnull_s3   █████████ 16.0ms
usagi_s3     █████████ 15.9ms
minio        ██████████████████████████████ 48.2ms
```

### ParallelWrite/1KB/C200

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.03 MB/s | 33.6ms | 64.2ms | 79.7ms | 0 |
| devnull_s3 | 0.03 MB/s | 29.3ms | 87.3ms | 117.0ms | 0 |
| minio | 0.01 MB/s | 82.3ms | 188.4ms | 286.7ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.03 MB/s
devnull_s3   ████████████████████████████ 0.03 MB/s
minio        ███████████ 0.01 MB/s
```

**Latency (P50)**
```
usagi_s3     ████████████ 33.6ms
devnull_s3   ██████████ 29.3ms
minio        ██████████████████████████████ 82.3ms
```

### ParallelWrite/1KB/C25

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.20 MB/s | 4.4ms | 9.1ms | 15.9ms | 0 |
| devnull_s3 | 0.19 MB/s | 4.9ms | 9.2ms | 11.8ms | 0 |
| minio | 0.09 MB/s | 9.5ms | 20.6ms | 32.6ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.20 MB/s
devnull_s3   ████████████████████████████ 0.19 MB/s
minio        █████████████ 0.09 MB/s
```

**Latency (P50)**
```
usagi_s3     █████████████ 4.4ms
devnull_s3   ███████████████ 4.9ms
minio        ██████████████████████████████ 9.5ms
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.11 MB/s | 8.0ms | 15.1ms | 19.3ms | 0 |
| devnull_s3 | 0.11 MB/s | 7.7ms | 18.0ms | 33.3ms | 0 |
| minio | 0.05 MB/s | 18.6ms | 40.3ms | 49.2ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.11 MB/s
devnull_s3   ████████████████████████████ 0.11 MB/s
minio        ████████████ 0.05 MB/s
```

**Latency (P50)**
```
usagi_s3     ████████████ 8.0ms
devnull_s3   ████████████ 7.7ms
minio        ██████████████████████████████ 18.6ms
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 155.02 MB/s | 1.6ms | 1.9ms | 2.6ms | 0 |
| devnull_s3 | 150.84 MB/s | 1.6ms | 2.0ms | 2.4ms | 0 |
| minio | 107.28 MB/s | 2.2ms | 3.2ms | 3.9ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 155.02 MB/s
devnull_s3   █████████████████████████████ 150.84 MB/s
minio        ████████████████████ 107.28 MB/s
```

**Latency (P50)**
```
usagi_s3     █████████████████████ 1.6ms
devnull_s3   ██████████████████████ 1.6ms
minio        ██████████████████████████████ 2.2ms
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 156.09 MB/s | 1.6ms | 1.8ms | 2.1ms | 0 |
| devnull_s3 | 145.68 MB/s | 1.7ms | 2.1ms | 2.7ms | 0 |
| minio | 108.97 MB/s | 2.1ms | 3.2ms | 4.7ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 156.09 MB/s
devnull_s3   ███████████████████████████ 145.68 MB/s
minio        ████████████████████ 108.97 MB/s
```

**Latency (P50)**
```
usagi_s3     ██████████████████████ 1.6ms
devnull_s3   ███████████████████████ 1.7ms
minio        ██████████████████████████████ 2.1ms
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 135.10 MB/s | 1.7ms | 2.3ms | 4.3ms | 0 |
| devnull_s3 | 127.87 MB/s | 1.8ms | 2.8ms | 3.6ms | 0 |
| minio | 87.58 MB/s | 2.5ms | 5.1ms | 8.7ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 135.10 MB/s
devnull_s3   ████████████████████████████ 127.87 MB/s
minio        ███████████████████ 87.58 MB/s
```

**Latency (P50)**
```
usagi_s3     ████████████████████ 1.7ms
devnull_s3   ██████████████████████ 1.8ms
minio        ██████████████████████████████ 2.5ms
```

### Read/100MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 237.37 MB/s | 417.7ms | 417.7ms | 417.7ms | 0 |
| usagi_s3 | 173.97 MB/s | 577.9ms | 577.9ms | 577.9ms | 0 |
| devnull_s3 | 165.22 MB/s | 612.7ms | 612.7ms | 612.7ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 237.37 MB/s
usagi_s3     █████████████████████ 173.97 MB/s
devnull_s3   ████████████████████ 165.22 MB/s
```

**Latency (P50)**
```
minio        ████████████████████ 417.7ms
usagi_s3     ████████████████████████████ 577.9ms
devnull_s3   ██████████████████████████████ 612.7ms
```

### Read/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 242.38 MB/s | 40.6ms | 42.6ms | 44.9ms | 0 |
| usagi_s3 | 182.26 MB/s | 53.4ms | 61.6ms | 61.7ms | 0 |
| devnull_s3 | 168.17 MB/s | 58.9ms | 64.1ms | 67.6ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 242.38 MB/s
usagi_s3     ██████████████████████ 182.26 MB/s
devnull_s3   ████████████████████ 168.17 MB/s
```

**Latency (P50)**
```
minio        ████████████████████ 40.6ms
usagi_s3     ███████████████████████████ 53.4ms
devnull_s3   ██████████████████████████████ 58.9ms
```

### Read/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 4.53 MB/s | 202.1us | 303.5us | 411.8us | 0 |
| devnull_s3 | 4.00 MB/s | 231.0us | 329.8us | 433.0us | 0 |
| minio | 2.45 MB/s | 328.0us | 660.9us | 1.7ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 4.53 MB/s
devnull_s3   ██████████████████████████ 4.00 MB/s
minio        ████████████████ 2.45 MB/s
```

**Latency (P50)**
```
usagi_s3     ██████████████████ 202.1us
devnull_s3   █████████████████████ 231.0us
minio        ██████████████████████████████ 328.0us
```

### Read/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 198.42 MB/s | 4.7ms | 6.9ms | 9.4ms | 0 |
| usagi_s3 | 187.54 MB/s | 5.0ms | 7.6ms | 9.1ms | 0 |
| devnull_s3 | 173.12 MB/s | 5.7ms | 6.6ms | 7.2ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 198.42 MB/s
usagi_s3     ████████████████████████████ 187.54 MB/s
devnull_s3   ██████████████████████████ 173.12 MB/s
```

**Latency (P50)**
```
minio        ████████████████████████ 4.7ms
usagi_s3     ██████████████████████████ 5.0ms
devnull_s3   ██████████████████████████████ 5.7ms
```

### Read/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 116.08 MB/s | 503.8us | 781.6us | 1.2ms | 0 |
| devnull_s3 | 101.89 MB/s | 593.7us | 783.7us | 1.0ms | 0 |
| minio | 98.99 MB/s | 576.6us | 905.6us | 1.6ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 116.08 MB/s
devnull_s3   ██████████████████████████ 101.89 MB/s
minio        █████████████████████████ 98.99 MB/s
```

**Latency (P50)**
```
usagi_s3     █████████████████████████ 503.8us
devnull_s3   ██████████████████████████████ 593.7us
minio        █████████████████████████████ 576.6us
```

### Scale/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 388 ops/s | 2.6ms | 2.6ms | 2.6ms | 0 |
| usagi_s3 | 355 ops/s | 2.8ms | 2.8ms | 2.8ms | 0 |
| minio | 152 ops/s | 6.6ms | 6.6ms | 6.6ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 388 ops/s
usagi_s3     ███████████████████████████ 355 ops/s
minio        ███████████ 152 ops/s
```

**Latency (P50)**
```
devnull_s3   ███████████ 2.6ms
usagi_s3     ████████████ 2.8ms
minio        ██████████████████████████████ 6.6ms
```

### Scale/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 43 ops/s | 23.4ms | 23.4ms | 23.4ms | 0 |
| devnull_s3 | 41 ops/s | 24.3ms | 24.3ms | 24.3ms | 0 |
| minio | 21 ops/s | 47.3ms | 47.3ms | 47.3ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 43 ops/s
devnull_s3   ████████████████████████████ 41 ops/s
minio        ██████████████ 21 ops/s
```

**Latency (P50)**
```
usagi_s3     ██████████████ 23.4ms
devnull_s3   ███████████████ 24.3ms
minio        ██████████████████████████████ 47.3ms
```

### Scale/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 4 ops/s | 237.9ms | 237.9ms | 237.9ms | 0 |
| usagi_s3 | 4 ops/s | 257.2ms | 257.2ms | 257.2ms | 0 |
| minio | 2 ops/s | 582.7ms | 582.7ms | 582.7ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 4 ops/s
usagi_s3     ███████████████████████████ 4 ops/s
minio        ████████████ 2 ops/s
```

**Latency (P50)**
```
devnull_s3   ████████████ 237.9ms
usagi_s3     █████████████ 257.2ms
minio        ██████████████████████████████ 582.7ms
```

### Scale/Delete/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0 ops/s | 2.36s | 2.36s | 2.36s | 0 |
| devnull_s3 | 0 ops/s | 2.44s | 2.44s | 2.44s | 0 |
| minio | 0 ops/s | 3.99s | 3.99s | 3.99s | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0 ops/s
devnull_s3   █████████████████████████████ 0 ops/s
minio        █████████████████ 0 ops/s
```

**Latency (P50)**
```
usagi_s3     █████████████████ 2.36s
devnull_s3   ██████████████████ 2.44s
minio        ██████████████████████████████ 3.99s
```

### Scale/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 2225 ops/s | 449.4us | 449.4us | 449.4us | 0 |
| usagi_s3 | 2053 ops/s | 487.2us | 487.2us | 487.2us | 0 |
| minio | 832 ops/s | 1.2ms | 1.2ms | 1.2ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 2225 ops/s
usagi_s3     ███████████████████████████ 2053 ops/s
minio        ███████████ 832 ops/s
```

**Latency (P50)**
```
devnull_s3   ███████████ 449.4us
usagi_s3     ████████████ 487.2us
minio        ██████████████████████████████ 1.2ms
```

### Scale/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 959 ops/s | 1.0ms | 1.0ms | 1.0ms | 0 |
| devnull_s3 | 958 ops/s | 1.0ms | 1.0ms | 1.0ms | 0 |
| minio | 474 ops/s | 2.1ms | 2.1ms | 2.1ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 959 ops/s
devnull_s3   █████████████████████████████ 958 ops/s
minio        ██████████████ 474 ops/s
```

**Latency (P50)**
```
usagi_s3     ██████████████ 1.0ms
devnull_s3   ██████████████ 1.0ms
minio        ██████████████████████████████ 2.1ms
```

### Scale/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 174 ops/s | 5.8ms | 5.8ms | 5.8ms | 0 |
| devnull_s3 | 163 ops/s | 6.1ms | 6.1ms | 6.1ms | 0 |
| minio | 65 ops/s | 15.5ms | 15.5ms | 15.5ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 174 ops/s
devnull_s3   ████████████████████████████ 163 ops/s
minio        ███████████ 65 ops/s
```

**Latency (P50)**
```
usagi_s3     ███████████ 5.8ms
devnull_s3   ███████████ 6.1ms
minio        ██████████████████████████████ 15.5ms
```

### Scale/List/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 6 ops/s | 179.5ms | 179.5ms | 179.5ms | 0 |
| devnull_s3 | 5 ops/s | 208.4ms | 208.4ms | 208.4ms | 0 |
| usagi_s3 | 4 ops/s | 234.4ms | 234.4ms | 234.4ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 6 ops/s
devnull_s3   █████████████████████████ 5 ops/s
usagi_s3     ██████████████████████ 4 ops/s
```

**Latency (P50)**
```
minio        ██████████████████████ 179.5ms
devnull_s3   ██████████████████████████ 208.4ms
usagi_s3     ██████████████████████████████ 234.4ms
```

### Scale/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 0.35 MB/s | 7.0ms | 7.0ms | 7.0ms | 0 |
| minio | 0.19 MB/s | 12.9ms | 12.9ms | 12.9ms | 0 |
| usagi_s3 | 0.17 MB/s | 14.1ms | 14.1ms | 14.1ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 0.35 MB/s
minio        ████████████████ 0.19 MB/s
usagi_s3     ██████████████ 0.17 MB/s
```

**Latency (P50)**
```
devnull_s3   ██████████████ 7.0ms
minio        ███████████████████████████ 12.9ms
usagi_s3     ██████████████████████████████ 14.1ms
```

### Scale/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 0.41 MB/s | 59.0ms | 59.0ms | 59.0ms | 0 |
| usagi_s3 | 0.30 MB/s | 81.5ms | 81.5ms | 81.5ms | 0 |
| minio | 0.21 MB/s | 118.8ms | 118.8ms | 118.8ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 0.41 MB/s
usagi_s3     █████████████████████ 0.30 MB/s
minio        ██████████████ 0.21 MB/s
```

**Latency (P50)**
```
devnull_s3   ██████████████ 59.0ms
usagi_s3     ████████████████████ 81.5ms
minio        ██████████████████████████████ 118.8ms
```

### Scale/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.45 MB/s | 545.7ms | 545.7ms | 545.7ms | 0 |
| devnull_s3 | 0.41 MB/s | 602.4ms | 602.4ms | 602.4ms | 0 |
| minio | 0.19 MB/s | 1.30s | 1.30s | 1.30s | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.45 MB/s
devnull_s3   ███████████████████████████ 0.41 MB/s
minio        ████████████ 0.19 MB/s
```

**Latency (P50)**
```
usagi_s3     ████████████ 545.7ms
devnull_s3   █████████████ 602.4ms
minio        ██████████████████████████████ 1.30s
```

### Scale/Write/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.42 MB/s | 5.77s | 5.77s | 5.77s | 0 |
| devnull_s3 | 0.41 MB/s | 5.95s | 5.95s | 5.95s | 0 |
| minio | 0.22 MB/s | 10.95s | 10.95s | 10.95s | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.42 MB/s
devnull_s3   █████████████████████████████ 0.41 MB/s
minio        ███████████████ 0.22 MB/s
```

**Latency (P50)**
```
usagi_s3     ███████████████ 5.77s
devnull_s3   ████████████████ 5.95s
minio        ██████████████████████████████ 10.95s
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 4229 ops/s | 221.2us | 331.2us | 486.0us | 0 |
| usagi_s3 | 3634 ops/s | 235.5us | 518.3us | 788.8us | 0 |
| minio | 3203 ops/s | 298.9us | 396.0us | 594.9us | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 4229 ops/s
usagi_s3     █████████████████████████ 3634 ops/s
minio        ██████████████████████ 3203 ops/s
```

**Latency (P50)**
```
devnull_s3   ██████████████████████ 221.2us
usagi_s3     ███████████████████████ 235.5us
minio        ██████████████████████████████ 298.9us
```

### Write/100MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 152.32 MB/s | 654.3ms | 654.3ms | 654.3ms | 0 |
| minio | 146.82 MB/s | 681.8ms | 681.8ms | 681.8ms | 0 |
| devnull_s3 | 123.74 MB/s | 818.0ms | 818.0ms | 818.0ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 152.32 MB/s
minio        ████████████████████████████ 146.82 MB/s
devnull_s3   ████████████████████████ 123.74 MB/s
```

**Latency (P50)**
```
usagi_s3     ███████████████████████ 654.3ms
minio        █████████████████████████ 681.8ms
devnull_s3   ██████████████████████████████ 818.0ms
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 143.83 MB/s | 68.2ms | 83.1ms | 83.1ms | 0 |
| usagi_s3 | 142.54 MB/s | 69.8ms | 76.4ms | 76.6ms | 0 |
| devnull_s3 | 128.90 MB/s | 77.8ms | 82.7ms | 82.7ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 143.83 MB/s
usagi_s3     █████████████████████████████ 142.54 MB/s
devnull_s3   ██████████████████████████ 128.90 MB/s
```

**Latency (P50)**
```
minio        ██████████████████████████ 68.2ms
usagi_s3     ██████████████████████████ 69.8ms
devnull_s3   ██████████████████████████████ 77.8ms
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 1.81 MB/s | 512.2us | 676.1us | 1.1ms | 0 |
| devnull_s3 | 1.53 MB/s | 589.5us | 953.7us | 1.6ms | 0 |
| minio | 1.10 MB/s | 819.1us | 1.3ms | 1.8ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 1.81 MB/s
devnull_s3   █████████████████████████ 1.53 MB/s
minio        ██████████████████ 1.10 MB/s
```

**Latency (P50)**
```
usagi_s3     ██████████████████ 512.2us
devnull_s3   █████████████████████ 589.5us
minio        ██████████████████████████████ 819.1us
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 144.07 MB/s | 6.6ms | 9.5ms | 10.7ms | 0 |
| devnull_s3 | 112.37 MB/s | 8.4ms | 12.1ms | 17.5ms | 0 |
| minio | 111.30 MB/s | 8.6ms | 11.9ms | 14.4ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 144.07 MB/s
devnull_s3   ███████████████████████ 112.37 MB/s
minio        ███████████████████████ 111.30 MB/s
```

**Latency (P50)**
```
usagi_s3     ██████████████████████ 6.6ms
devnull_s3   █████████████████████████████ 8.4ms
minio        ██████████████████████████████ 8.6ms
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 65.99 MB/s | 912.8us | 1.2ms | 1.6ms | 0 |
| devnull_s3 | 51.41 MB/s | 1.1ms | 1.8ms | 3.0ms | 0 |
| minio | 38.25 MB/s | 1.4ms | 3.2ms | 5.1ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 65.99 MB/s
devnull_s3   ███████████████████████ 51.41 MB/s
minio        █████████████████ 38.25 MB/s
```

**Latency (P50)**
```
usagi_s3     ███████████████████ 912.8us
devnull_s3   ███████████████████████ 1.1ms
minio        ██████████████████████████████ 1.4ms
```

## Resource Usage

| Driver | Memory | RSS | Cache | CPU | Volume | Block I/O |
|--------|--------|-----|-------|-----|--------|----------|
| devnull_s3 | 1.376GiB / 7.653GiB | 1409.0 MB | - | 2.1% | (no data) | 15.4MB / 1.52GB |
| minio | 650.4MiB / 7.653GiB | 650.4 MB | - | 7.4% | 1889.0 MB | 61MB / 2.11GB |
| usagi_s3 | 1.59GiB / 7.653GiB | 1628.2 MB | - | 3.7% | (no data) | 3.94MB / 2.34GB |

> **Note:** RSS = actual application memory. Cache = OS page cache (reclaimable).

## Recommendations

- **Write-heavy workloads:** usagi_s3
- **Read-heavy workloads:** minio

---

*Generated by storage benchmark CLI*
