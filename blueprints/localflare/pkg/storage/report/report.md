# Storage Benchmark Report

**Generated:** 2026-01-21T16:18:28+07:00

**Go Version:** go1.25.5

**Platform:** darwin/arm64

## Executive Summary

### Summary

**Overall Winner:** rabbit_s3 (won 21/39 benchmarks, 54%)

| Rank | Driver | Wins | Win Rate |
|------|--------|------|----------|
| 1 | rabbit_s3 | 21 | 54% |
| 2 | liteio | 11 | 28% |
| 3 | usagi_s3 | 7 | 18% |

### Performance Leaders

| Operation | Leader | Performance | Margin |
|-----------|--------|-------------|--------|
| Small Read (1KB) | liteio | 4.2 MB/s | close |
| Small Write (1KB) | usagi_s3 | 1.7 MB/s | close |
| Large Read (100MB) | rabbit_s3 | 174.5 MB/s | close |
| Large Write (100MB) | liteio | 132.0 MB/s | close |
| Delete | rabbit_s3 | 4.1K ops/s | close |
| Stat | rabbit_s3 | 4.3K ops/s | close |
| List (100 objects) | rabbit_s3 | 900 ops/s | close |
| Copy | liteio | 1.2 MB/s | close |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (100MB+) | **liteio** | 132 MB/s | Best for media, backups |
| Large File Downloads (100MB) | **rabbit_s3** | 174 MB/s | Best for streaming, CDN |
| Small File Operations | **rabbit_s3** | 2968 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **usagi_s3** | - | Best for multi-user apps |
| Memory Constrained | **minio** | 903 MB RAM | Best for edge/embedded |

### Large File Performance (100MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| liteio | 132.0 | 170.8 | 757.9ms | 583.8ms |
| minio | 72.9 | 148.8 | 1.34s | 668.8ms |
| rabbit_s3 | 120.4 | 174.5 | 829.1ms | 568.1ms |
| usagi_s3 | 122.7 | 167.5 | 835.2ms | 604.5ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| liteio | 466 | 4267 | 1.9ms | 221.0us |
| minio | 233 | 1215 | 2.5ms | 727.6us |
| rabbit_s3 | 1697 | 4239 | 561.9us | 222.3us |
| usagi_s3 | 1783 | 4098 | 505.1us | 227.1us |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| liteio | 4043 | 893 | 3932 |
| minio | 1379 | 295 | 1220 |
| rabbit_s3 | 4259 | 900 | 4058 |
| usagi_s3 | 3681 | 801 | 3946 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| liteio | 1.37 | 0.30 | 0.11 | 0.06 | 0.04 | 0.02 |
| minio | 0.37 | 0.13 | 0.08 | 0.05 | 0.02 | 0.01 |
| rabbit_s3 | 1.45 | 0.38 | 0.17 | 0.10 | 0.03 | 0.02 |
| usagi_s3 | 1.17 | 0.39 | 0.16 | 0.06 | 0.04 | 0.02 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| liteio | 3.24 | 1.00 | 0.54 | 0.31 | 0.16 | 0.09 |
| minio | 1.28 | 0.50 | 0.26 | 0.18 | 0.09 | 0.05 |
| rabbit_s3 | 3.54 | 1.02 | 0.57 | 0.37 | 0.18 | 0.08 |
| usagi_s3 | 3.34 | 1.02 | 0.56 | 0.33 | 0.16 | 0.09 |

*\* indicates errors occurred*

### Scale Performance

Performance with varying numbers of objects (1KB each).

**Write N Files (total time)**

| Driver | 10 |
|--------|------|
| liteio | 8.6ms |
| minio | 35.9ms |
| rabbit_s3 | 7.7ms |
| usagi_s3 | 7.4ms |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 10 |
|--------|------|
| liteio | 667.4us |
| minio | 1.6ms |
| rabbit_s3 | 705.7us |
| usagi_s3 | 1.2ms |

*\* indicates errors occurred*

### Resource Usage Summary

| Driver | Memory | CPU |
|--------|--------|-----|
| liteio | 1053.7 MB | 2.8% |
| minio | 903.2 MB | 3.0% |
| rabbit_s3 | 1647.6 MB | 1.7% |
| usagi_s3 | 1514.5 MB | 1.7% |

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

- **liteio** (39 benchmarks)
- **minio** (39 benchmarks)
- **rabbit_s3** (39 benchmarks)
- **usagi_s3** (39 benchmarks)

## Detailed Results

### Copy/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.23 MB/s | 646.0us | 1.4ms | 1.9ms | 0 |
| usagi_s3 | 1.18 MB/s | 700.5us | 1.5ms | 2.0ms | 0 |
| rabbit_s3 | 1.13 MB/s | 768.4us | 1.4ms | 1.9ms | 0 |
| minio | 0.25 MB/s | 3.1ms | 6.6ms | 10.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.23 MB/s
usagi_s3     ████████████████████████████ 1.18 MB/s
rabbit_s3    ███████████████████████████ 1.13 MB/s
minio        ██████ 0.25 MB/s
```

**Latency (P50)**
```
liteio       ██████ 646.0us
usagi_s3     ██████ 700.5us
rabbit_s3    ███████ 768.4us
minio        ██████████████████████████████ 3.1ms
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit_s3 | 4058 ops/s | 225.8us | 347.4us | 631.0us | 0 |
| usagi_s3 | 3946 ops/s | 232.3us | 371.9us | 626.5us | 0 |
| liteio | 3932 ops/s | 235.3us | 357.6us | 607.5us | 0 |
| minio | 1220 ops/s | 739.9us | 1.5ms | 2.5ms | 0 |

**Throughput**
```
rabbit_s3    ██████████████████████████████ 4058 ops/s
usagi_s3     █████████████████████████████ 3946 ops/s
liteio       █████████████████████████████ 3932 ops/s
minio        █████████ 1220 ops/s
```

**Latency (P50)**
```
rabbit_s3    █████████ 225.8us
usagi_s3     █████████ 232.3us
liteio       █████████ 235.3us
minio        ██████████████████████████████ 739.9us
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.16 MB/s | 549.4us | 880.0us | 1.2ms | 0 |
| rabbit_s3 | 0.14 MB/s | 631.1us | 1.1ms | 1.5ms | 0 |
| liteio | 0.14 MB/s | 665.5us | 1.0ms | 1.4ms | 0 |
| minio | 0.03 MB/s | 2.9ms | 6.1ms | 10.1ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.16 MB/s
rabbit_s3    ██████████████████████████ 0.14 MB/s
liteio       █████████████████████████ 0.14 MB/s
minio        █████ 0.03 MB/s
```

**Latency (P50)**
```
usagi_s3     █████ 549.4us
rabbit_s3    ██████ 631.1us
liteio       ██████ 665.5us
minio        ██████████████████████████████ 2.9ms
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1649 ops/s | 545.9us | 896.9us | 1.1ms | 0 |
| rabbit_s3 | 1510 ops/s | 622.6us | 987.2us | 1.4ms | 0 |
| usagi_s3 | 1398 ops/s | 655.5us | 1.1ms | 1.5ms | 0 |
| minio | 263 ops/s | 3.2ms | 6.7ms | 12.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1649 ops/s
rabbit_s3    ███████████████████████████ 1510 ops/s
usagi_s3     █████████████████████████ 1398 ops/s
minio        ████ 263 ops/s
```

**Latency (P50)**
```
liteio       █████ 545.9us
rabbit_s3    █████ 622.6us
usagi_s3     ██████ 655.5us
minio        ██████████████████████████████ 3.2ms
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit_s3 | 0.14 MB/s | 597.4us | 1.1ms | 1.5ms | 0 |
| liteio | 0.13 MB/s | 572.4us | 1.5ms | 2.2ms | 0 |
| usagi_s3 | 0.13 MB/s | 660.5us | 1.1ms | 1.6ms | 0 |
| minio | 0.02 MB/s | 3.3ms | 7.8ms | 11.3ms | 0 |

**Throughput**
```
rabbit_s3    ██████████████████████████████ 0.14 MB/s
liteio       ████████████████████████████ 0.13 MB/s
usagi_s3     ███████████████████████████ 0.13 MB/s
minio        ████ 0.02 MB/s
```

**Latency (P50)**
```
rabbit_s3    █████ 597.4us
liteio       █████ 572.4us
usagi_s3     ██████ 660.5us
minio        ██████████████████████████████ 3.3ms
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit_s3 | 900 ops/s | 1.1ms | 1.5ms | 1.7ms | 0 |
| liteio | 893 ops/s | 1.1ms | 1.5ms | 1.7ms | 0 |
| usagi_s3 | 801 ops/s | 1.2ms | 1.7ms | 1.8ms | 0 |
| minio | 295 ops/s | 3.0ms | 5.7ms | 9.2ms | 0 |

**Throughput**
```
rabbit_s3    ██████████████████████████████ 900 ops/s
liteio       █████████████████████████████ 893 ops/s
usagi_s3     ██████████████████████████ 801 ops/s
minio        █████████ 295 ops/s
```

**Latency (P50)**
```
rabbit_s3    ██████████ 1.1ms
liteio       ██████████ 1.1ms
usagi_s3     ███████████ 1.2ms
minio        ██████████████████████████████ 3.0ms
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.42 MB/s | 25.5ms | 68.9ms | 333.3ms | 0 |
| usagi_s3 | 0.38 MB/s | 25.2ms | 48.4ms | 706.8ms | 0 |
| rabbit_s3 | 0.36 MB/s | 15.6ms | 104.8ms | 132.8ms | 0 |
| minio | 0.23 MB/s | 41.8ms | 177.3ms | 236.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.42 MB/s
usagi_s3     ███████████████████████████ 0.38 MB/s
rabbit_s3    █████████████████████████ 0.36 MB/s
minio        ████████████████ 0.23 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████ 25.5ms
usagi_s3     ██████████████████ 25.2ms
rabbit_s3    ███████████ 15.6ms
minio        ██████████████████████████████ 41.8ms
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.62 MB/s | 24.2ms | 34.9ms | 44.8ms | 0 |
| rabbit_s3 | 0.58 MB/s | 26.1ms | 44.6ms | 66.0ms | 0 |
| usagi_s3 | 0.57 MB/s | 26.7ms | 45.1ms | 58.1ms | 0 |
| minio | 0.48 MB/s | 28.0ms | 66.9ms | 102.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.62 MB/s
rabbit_s3    ███████████████████████████ 0.58 MB/s
usagi_s3     ███████████████████████████ 0.57 MB/s
minio        ███████████████████████ 0.48 MB/s
```

**Latency (P50)**
```
liteio       █████████████████████████ 24.2ms
rabbit_s3    ███████████████████████████ 26.1ms
usagi_s3     ████████████████████████████ 26.7ms
minio        ██████████████████████████████ 28.0ms
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.35 MB/s | 28.4ms | 66.4ms | 832.6ms | 0 |
| usagi_s3 | 0.32 MB/s | 45.7ms | 96.7ms | 109.2ms | 0 |
| rabbit_s3 | 0.26 MB/s | 77.1ms | 96.4ms | 101.8ms | 0 |
| minio | 0.16 MB/s | 61.8ms | 290.6ms | 856.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.35 MB/s
usagi_s3     ███████████████████████████ 0.32 MB/s
rabbit_s3    ██████████████████████ 0.26 MB/s
minio        █████████████ 0.16 MB/s
```

**Latency (P50)**
```
liteio       ███████████ 28.4ms
usagi_s3     █████████████████ 45.7ms
rabbit_s3    ██████████████████████████████ 77.1ms
minio        ████████████████████████ 61.8ms
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 112.02 MB/s | 132.0ms | 141.8ms | 141.8ms | 0 |
| rabbit_s3 | 95.91 MB/s | 152.5ms | 177.3ms | 177.3ms | 0 |
| usagi_s3 | 87.28 MB/s | 175.5ms | 179.6ms | 179.6ms | 0 |
| minio | 69.24 MB/s | 224.7ms | 296.1ms | 296.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 112.02 MB/s
rabbit_s3    █████████████████████████ 95.91 MB/s
usagi_s3     ███████████████████████ 87.28 MB/s
minio        ██████████████████ 69.24 MB/s
```

**Latency (P50)**
```
liteio       █████████████████ 132.0ms
rabbit_s3    ████████████████████ 152.5ms
usagi_s3     ███████████████████████ 175.5ms
minio        ██████████████████████████████ 224.7ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| rabbit_s3 | 3.54 MB/s | 275.6us | 348.0us | 261.9us | 348.1us | 579.3us | 0 |
| usagi_s3 | 3.34 MB/s | 292.5us | 428.0us | 272.0us | 428.2us | 622.1us | 0 |
| liteio | 3.24 MB/s | 301.6us | 423.9us | 281.1us | 424.0us | 650.1us | 0 |
| minio | 1.28 MB/s | 759.8us | 1.3ms | 709.6us | 1.3ms | 2.1ms | 0 |

**Throughput**
```
rabbit_s3    ██████████████████████████████ 3.54 MB/s
usagi_s3     ████████████████████████████ 3.34 MB/s
liteio       ███████████████████████████ 3.24 MB/s
minio        ██████████ 1.28 MB/s
```

**Latency (P50)**
```
rabbit_s3    ███████████ 261.9us
usagi_s3     ███████████ 272.0us
liteio       ███████████ 281.1us
minio        ██████████████████████████████ 709.6us
```

### ParallelRead/1KB/C10

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| usagi_s3 | 1.02 MB/s | 954.1us | 1.5ms | 902.0us | 1.5ms | 1.9ms | 0 |
| rabbit_s3 | 1.02 MB/s | 956.5us | 1.5ms | 900.0us | 1.5ms | 1.9ms | 0 |
| liteio | 1.00 MB/s | 978.1us | 1.6ms | 914.5us | 1.6ms | 2.1ms | 0 |
| minio | 0.50 MB/s | 2.0ms | 3.5ms | 1.8ms | 3.5ms | 6.0ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 1.02 MB/s
rabbit_s3    █████████████████████████████ 1.02 MB/s
liteio       █████████████████████████████ 1.00 MB/s
minio        ██████████████ 0.50 MB/s
```

**Latency (P50)**
```
usagi_s3     ███████████████ 902.0us
rabbit_s3    ███████████████ 900.0us
liteio       ███████████████ 914.5us
minio        ██████████████████████████████ 1.8ms
```

### ParallelRead/1KB/C100

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| rabbit_s3 | 0.18 MB/s | 5.5ms | 9.0ms | 5.3ms | 9.0ms | 11.5ms | 0 |
| liteio | 0.16 MB/s | 5.9ms | 9.5ms | 5.8ms | 9.5ms | 11.7ms | 0 |
| usagi_s3 | 0.16 MB/s | 6.0ms | 9.6ms | 5.8ms | 9.6ms | 12.8ms | 0 |
| minio | 0.09 MB/s | 10.7ms | 17.3ms | 10.2ms | 17.3ms | 26.4ms | 0 |

**Throughput**
```
rabbit_s3    ██████████████████████████████ 0.18 MB/s
liteio       ███████████████████████████ 0.16 MB/s
usagi_s3     ███████████████████████████ 0.16 MB/s
minio        ███████████████ 0.09 MB/s
```

**Latency (P50)**
```
rabbit_s3    ███████████████ 5.3ms
liteio       █████████████████ 5.8ms
usagi_s3     █████████████████ 5.8ms
minio        ██████████████████████████████ 10.2ms
```

### ParallelRead/1KB/C200

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| usagi_s3 | 0.09 MB/s | 10.7ms | 16.5ms | 10.8ms | 16.5ms | 20.9ms | 0 |
| liteio | 0.09 MB/s | 11.3ms | 16.3ms | 11.4ms | 16.3ms | 21.7ms | 0 |
| rabbit_s3 | 0.08 MB/s | 11.5ms | 16.8ms | 11.9ms | 16.8ms | 20.0ms | 0 |
| minio | 0.05 MB/s | 19.4ms | 41.2ms | 17.6ms | 41.2ms | 58.4ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.09 MB/s
liteio       ████████████████████████████ 0.09 MB/s
rabbit_s3    ███████████████████████████ 0.08 MB/s
minio        ████████████████ 0.05 MB/s
```

**Latency (P50)**
```
usagi_s3     ██████████████████ 10.8ms
liteio       ███████████████████ 11.4ms
rabbit_s3    ████████████████████ 11.9ms
minio        ██████████████████████████████ 17.6ms
```

### ParallelRead/1KB/C25

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| rabbit_s3 | 0.57 MB/s | 1.7ms | 2.7ms | 1.6ms | 2.7ms | 3.4ms | 0 |
| usagi_s3 | 0.56 MB/s | 1.7ms | 2.9ms | 1.6ms | 2.9ms | 3.8ms | 0 |
| liteio | 0.54 MB/s | 1.8ms | 3.0ms | 1.7ms | 3.0ms | 3.9ms | 0 |
| minio | 0.26 MB/s | 3.7ms | 6.4ms | 3.4ms | 6.4ms | 9.5ms | 0 |

**Throughput**
```
rabbit_s3    ██████████████████████████████ 0.57 MB/s
usagi_s3     █████████████████████████████ 0.56 MB/s
liteio       ████████████████████████████ 0.54 MB/s
minio        █████████████ 0.26 MB/s
```

**Latency (P50)**
```
rabbit_s3    ██████████████ 1.6ms
usagi_s3     ██████████████ 1.6ms
liteio       ██████████████ 1.7ms
minio        ██████████████████████████████ 3.4ms
```

### ParallelRead/1KB/C50

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| rabbit_s3 | 0.37 MB/s | 2.7ms | 5.2ms | 2.4ms | 5.2ms | 7.0ms | 0 |
| usagi_s3 | 0.33 MB/s | 2.9ms | 5.3ms | 2.7ms | 5.3ms | 6.8ms | 0 |
| liteio | 0.31 MB/s | 3.1ms | 5.1ms | 2.9ms | 5.1ms | 6.7ms | 0 |
| minio | 0.18 MB/s | 5.5ms | 9.6ms | 5.1ms | 9.6ms | 14.2ms | 0 |

**Throughput**
```
rabbit_s3    ██████████████████████████████ 0.37 MB/s
usagi_s3     ███████████████████████████ 0.33 MB/s
liteio       █████████████████████████ 0.31 MB/s
minio        ██████████████ 0.18 MB/s
```

**Latency (P50)**
```
rabbit_s3    █████████████ 2.4ms
usagi_s3     ███████████████ 2.7ms
liteio       █████████████████ 2.9ms
minio        ██████████████████████████████ 5.1ms
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit_s3 | 1.45 MB/s | 629.4us | 863.8us | 1.2ms | 0 |
| liteio | 1.37 MB/s | 653.9us | 1.0ms | 1.6ms | 0 |
| usagi_s3 | 1.17 MB/s | 766.5us | 1.2ms | 1.6ms | 0 |
| minio | 0.37 MB/s | 2.3ms | 3.8ms | 6.0ms | 0 |

**Throughput**
```
rabbit_s3    ██████████████████████████████ 1.45 MB/s
liteio       ████████████████████████████ 1.37 MB/s
usagi_s3     ████████████████████████ 1.17 MB/s
minio        ███████ 0.37 MB/s
```

**Latency (P50)**
```
rabbit_s3    ████████ 629.4us
liteio       ████████ 653.9us
usagi_s3     █████████ 766.5us
minio        ██████████████████████████████ 2.3ms
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.39 MB/s | 2.5ms | 3.5ms | 4.7ms | 0 |
| rabbit_s3 | 0.38 MB/s | 2.4ms | 3.9ms | 6.3ms | 0 |
| liteio | 0.30 MB/s | 2.9ms | 5.4ms | 6.5ms | 0 |
| minio | 0.13 MB/s | 6.4ms | 13.4ms | 23.3ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.39 MB/s
rabbit_s3    █████████████████████████████ 0.38 MB/s
liteio       ███████████████████████ 0.30 MB/s
minio        ██████████ 0.13 MB/s
```

**Latency (P50)**
```
usagi_s3     ███████████ 2.5ms
rabbit_s3    ███████████ 2.4ms
liteio       █████████████ 2.9ms
minio        ██████████████████████████████ 6.4ms
```

### ParallelWrite/1KB/C100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.04 MB/s | 26.1ms | 42.7ms | 99.7ms | 0 |
| usagi_s3 | 0.04 MB/s | 29.7ms | 41.3ms | 47.8ms | 0 |
| rabbit_s3 | 0.03 MB/s | 28.8ms | 52.6ms | 90.0ms | 0 |
| minio | 0.02 MB/s | 48.2ms | 83.0ms | 99.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.04 MB/s
usagi_s3     ████████████████████████████ 0.04 MB/s
rabbit_s3    █████████████████████████ 0.03 MB/s
minio        ███████████████ 0.02 MB/s
```

**Latency (P50)**
```
liteio       ████████████████ 26.1ms
usagi_s3     ██████████████████ 29.7ms
rabbit_s3    █████████████████ 28.8ms
minio        ██████████████████████████████ 48.2ms
```

### ParallelWrite/1KB/C200

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit_s3 | 0.02 MB/s | 53.8ms | 78.1ms | 92.0ms | 0 |
| liteio | 0.02 MB/s | 64.9ms | 88.1ms | 100.7ms | 0 |
| usagi_s3 | 0.02 MB/s | 67.2ms | 127.6ms | 156.4ms | 0 |
| minio | 0.01 MB/s | 99.3ms | 239.9ms | 310.0ms | 0 |

**Throughput**
```
rabbit_s3    ██████████████████████████████ 0.02 MB/s
liteio       █████████████████████████ 0.02 MB/s
usagi_s3     ██████████████████████ 0.02 MB/s
minio        ████████████ 0.01 MB/s
```

**Latency (P50)**
```
rabbit_s3    ████████████████ 53.8ms
liteio       ███████████████████ 64.9ms
usagi_s3     ████████████████████ 67.2ms
minio        ██████████████████████████████ 99.3ms
```

### ParallelWrite/1KB/C25

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit_s3 | 0.17 MB/s | 4.9ms | 9.9ms | 17.9ms | 0 |
| usagi_s3 | 0.16 MB/s | 5.7ms | 12.0ms | 16.8ms | 0 |
| liteio | 0.11 MB/s | 8.2ms | 19.0ms | 24.1ms | 0 |
| minio | 0.08 MB/s | 10.1ms | 19.3ms | 87.1ms | 0 |

**Throughput**
```
rabbit_s3    ██████████████████████████████ 0.17 MB/s
usagi_s3     ██████████████████████████ 0.16 MB/s
liteio       ██████████████████ 0.11 MB/s
minio        █████████████ 0.08 MB/s
```

**Latency (P50)**
```
rabbit_s3    ██████████████ 4.9ms
usagi_s3     ████████████████ 5.7ms
liteio       ████████████████████████ 8.2ms
minio        ██████████████████████████████ 10.1ms
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit_s3 | 0.10 MB/s | 9.0ms | 18.9ms | 26.3ms | 0 |
| usagi_s3 | 0.06 MB/s | 16.5ms | 23.6ms | 33.9ms | 0 |
| liteio | 0.06 MB/s | 17.6ms | 30.2ms | 40.3ms | 0 |
| minio | 0.05 MB/s | 18.6ms | 40.3ms | 57.9ms | 0 |

**Throughput**
```
rabbit_s3    ██████████████████████████████ 0.10 MB/s
usagi_s3     ███████████████████ 0.06 MB/s
liteio       █████████████████ 0.06 MB/s
minio        ██████████████ 0.05 MB/s
```

**Latency (P50)**
```
rabbit_s3    ██████████████ 9.0ms
usagi_s3     ██████████████████████████ 16.5ms
liteio       ████████████████████████████ 17.6ms
minio        ██████████████████████████████ 18.6ms
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 150.28 MB/s | 1.6ms | 2.1ms | 2.4ms | 0 |
| rabbit_s3 | 148.14 MB/s | 1.6ms | 2.2ms | 2.6ms | 0 |
| liteio | 146.94 MB/s | 1.7ms | 2.1ms | 2.4ms | 0 |
| minio | 78.91 MB/s | 3.0ms | 4.9ms | 7.1ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 150.28 MB/s
rabbit_s3    █████████████████████████████ 148.14 MB/s
liteio       █████████████████████████████ 146.94 MB/s
minio        ███████████████ 78.91 MB/s
```

**Latency (P50)**
```
usagi_s3     ████████████████ 1.6ms
rabbit_s3    ████████████████ 1.6ms
liteio       ████████████████ 1.7ms
minio        ██████████████████████████████ 3.0ms
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit_s3 | 149.91 MB/s | 1.6ms | 2.1ms | 2.3ms | 0 |
| liteio | 148.07 MB/s | 1.6ms | 2.1ms | 2.4ms | 0 |
| usagi_s3 | 147.81 MB/s | 1.6ms | 2.2ms | 2.4ms | 0 |
| minio | 82.13 MB/s | 2.9ms | 4.0ms | 6.6ms | 0 |

**Throughput**
```
rabbit_s3    ██████████████████████████████ 149.91 MB/s
liteio       █████████████████████████████ 148.07 MB/s
usagi_s3     █████████████████████████████ 147.81 MB/s
minio        ████████████████ 82.13 MB/s
```

**Latency (P50)**
```
rabbit_s3    █████████████████ 1.6ms
liteio       █████████████████ 1.6ms
usagi_s3     ████████████████ 1.6ms
minio        ██████████████████████████████ 2.9ms
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 131.21 MB/s | 1.8ms | 2.6ms | 3.1ms | 0 |
| rabbit_s3 | 131.07 MB/s | 1.8ms | 2.5ms | 3.3ms | 0 |
| usagi_s3 | 127.40 MB/s | 1.9ms | 2.7ms | 3.4ms | 0 |
| minio | 73.86 MB/s | 3.1ms | 4.6ms | 8.8ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 131.21 MB/s
rabbit_s3    █████████████████████████████ 131.07 MB/s
usagi_s3     █████████████████████████████ 127.40 MB/s
minio        ████████████████ 73.86 MB/s
```

**Latency (P50)**
```
liteio       █████████████████ 1.8ms
rabbit_s3    █████████████████ 1.8ms
usagi_s3     █████████████████ 1.9ms
minio        ██████████████████████████████ 3.1ms
```

### Read/100MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| rabbit_s3 | 174.49 MB/s | 2.5ms | 2.5ms | 568.1ms | 568.1ms | 568.1ms | 0 |
| liteio | 170.84 MB/s | 2.0ms | 2.0ms | 583.8ms | 583.8ms | 583.8ms | 0 |
| usagi_s3 | 167.46 MB/s | 2.5ms | 2.3ms | 604.5ms | 604.5ms | 604.5ms | 0 |
| minio | 148.80 MB/s | 2.8ms | 2.8ms | 668.8ms | 668.8ms | 668.8ms | 0 |

**Throughput**
```
rabbit_s3    ██████████████████████████████ 174.49 MB/s
liteio       █████████████████████████████ 170.84 MB/s
usagi_s3     ████████████████████████████ 167.46 MB/s
minio        █████████████████████████ 148.80 MB/s
```

**Latency (P50)**
```
rabbit_s3    █████████████████████████ 568.1ms
liteio       ██████████████████████████ 583.8ms
usagi_s3     ███████████████████████████ 604.5ms
minio        ██████████████████████████████ 668.8ms
```

### Read/10MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| rabbit_s3 | 177.38 MB/s | 2.2ms | 2.8ms | 54.1ms | 64.9ms | 73.7ms | 0 |
| liteio | 173.37 MB/s | 2.5ms | 4.1ms | 58.1ms | 60.7ms | 62.4ms | 0 |
| usagi_s3 | 168.91 MB/s | 2.0ms | 2.5ms | 60.0ms | 62.8ms | 65.6ms | 0 |
| minio | 139.84 MB/s | 2.0ms | 3.6ms | 70.1ms | 83.3ms | 83.3ms | 0 |

**Throughput**
```
rabbit_s3    ██████████████████████████████ 177.38 MB/s
liteio       █████████████████████████████ 173.37 MB/s
usagi_s3     ████████████████████████████ 168.91 MB/s
minio        ███████████████████████ 139.84 MB/s
```

**Latency (P50)**
```
rabbit_s3    ███████████████████████ 54.1ms
liteio       ████████████████████████ 58.1ms
usagi_s3     █████████████████████████ 60.0ms
minio        ██████████████████████████████ 70.1ms
```

### Read/1KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 4.17 MB/s | 234.2us | 316.0us | 221.0us | 316.0us | 464.0us | 0 |
| rabbit_s3 | 4.14 MB/s | 235.8us | 318.7us | 222.3us | 318.8us | 457.4us | 0 |
| usagi_s3 | 4.00 MB/s | 243.9us | 345.2us | 227.1us | 345.3us | 522.7us | 0 |
| minio | 1.19 MB/s | 822.8us | 1.4ms | 727.6us | 1.4ms | 2.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4.17 MB/s
rabbit_s3    █████████████████████████████ 4.14 MB/s
usagi_s3     ████████████████████████████ 4.00 MB/s
minio        ████████ 1.19 MB/s
```

**Latency (P50)**
```
liteio       █████████ 221.0us
rabbit_s3    █████████ 222.3us
usagi_s3     █████████ 227.1us
minio        ██████████████████████████████ 727.6us
```

### Read/1MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| rabbit_s3 | 179.56 MB/s | 501.0us | 811.2us | 5.4ms | 6.4ms | 8.6ms | 0 |
| liteio | 174.05 MB/s | 559.1us | 958.4us | 5.6ms | 7.0ms | 7.9ms | 0 |
| usagi_s3 | 172.21 MB/s | 520.3us | 788.1us | 5.7ms | 6.8ms | 7.6ms | 0 |
| minio | 113.54 MB/s | 2.0ms | 3.8ms | 8.2ms | 12.5ms | 14.0ms | 0 |

**Throughput**
```
rabbit_s3    ██████████████████████████████ 179.56 MB/s
liteio       █████████████████████████████ 174.05 MB/s
usagi_s3     ████████████████████████████ 172.21 MB/s
minio        ██████████████████ 113.54 MB/s
```

**Latency (P50)**
```
rabbit_s3    ███████████████████ 5.4ms
liteio       ████████████████████ 5.6ms
usagi_s3     ████████████████████ 5.7ms
minio        ██████████████████████████████ 8.2ms
```

### Read/64KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| rabbit_s3 | 105.88 MB/s | 292.2us | 444.8us | 571.6us | 755.4us | 945.6us | 0 |
| liteio | 103.10 MB/s | 307.6us | 536.1us | 577.7us | 828.2us | 1.1ms | 0 |
| usagi_s3 | 98.50 MB/s | 316.2us | 575.4us | 598.8us | 910.0us | 1.1ms | 0 |
| minio | 43.36 MB/s | 1.1ms | 2.0ms | 1.3ms | 2.4ms | 4.1ms | 0 |

**Throughput**
```
rabbit_s3    ██████████████████████████████ 105.88 MB/s
liteio       █████████████████████████████ 103.10 MB/s
usagi_s3     ███████████████████████████ 98.50 MB/s
minio        ████████████ 43.36 MB/s
```

**Latency (P50)**
```
rabbit_s3    █████████████ 571.6us
liteio       █████████████ 577.7us
usagi_s3     █████████████ 598.8us
minio        ██████████████████████████████ 1.3ms
```

### Scale/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit_s3 | 401 ops/s | 2.5ms | 2.5ms | 2.5ms | 0 |
| liteio | 392 ops/s | 2.6ms | 2.6ms | 2.6ms | 0 |
| usagi_s3 | 359 ops/s | 2.8ms | 2.8ms | 2.8ms | 0 |
| minio | 83 ops/s | 12.1ms | 12.1ms | 12.1ms | 0 |

**Throughput**
```
rabbit_s3    ██████████████████████████████ 401 ops/s
liteio       █████████████████████████████ 392 ops/s
usagi_s3     ██████████████████████████ 359 ops/s
minio        ██████ 83 ops/s
```

**Latency (P50)**
```
rabbit_s3    ██████ 2.5ms
liteio       ██████ 2.6ms
usagi_s3     ██████ 2.8ms
minio        ██████████████████████████████ 12.1ms
```

### Scale/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1498 ops/s | 667.4us | 667.4us | 667.4us | 0 |
| rabbit_s3 | 1417 ops/s | 705.7us | 705.7us | 705.7us | 0 |
| usagi_s3 | 834 ops/s | 1.2ms | 1.2ms | 1.2ms | 0 |
| minio | 611 ops/s | 1.6ms | 1.6ms | 1.6ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1498 ops/s
rabbit_s3    ████████████████████████████ 1417 ops/s
usagi_s3     ████████████████ 834 ops/s
minio        ████████████ 611 ops/s
```

**Latency (P50)**
```
liteio       ████████████ 667.4us
rabbit_s3    ████████████ 705.7us
usagi_s3     ██████████████████████ 1.2ms
minio        ██████████████████████████████ 1.6ms
```

### Scale/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 1.32 MB/s | 7.4ms | 7.4ms | 7.4ms | 0 |
| rabbit_s3 | 1.27 MB/s | 7.7ms | 7.7ms | 7.7ms | 0 |
| liteio | 1.13 MB/s | 8.6ms | 8.6ms | 8.6ms | 0 |
| minio | 0.27 MB/s | 35.9ms | 35.9ms | 35.9ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 1.32 MB/s
rabbit_s3    ████████████████████████████ 1.27 MB/s
liteio       █████████████████████████ 1.13 MB/s
minio        ██████ 0.27 MB/s
```

**Latency (P50)**
```
usagi_s3     ██████ 7.4ms
rabbit_s3    ██████ 7.7ms
liteio       ███████ 8.6ms
minio        ██████████████████████████████ 35.9ms
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit_s3 | 4259 ops/s | 217.9us | 322.8us | 530.6us | 0 |
| liteio | 4043 ops/s | 233.0us | 344.9us | 465.1us | 0 |
| usagi_s3 | 3681 ops/s | 257.8us | 392.0us | 554.4us | 0 |
| minio | 1379 ops/s | 638.1us | 1.3ms | 2.3ms | 0 |

**Throughput**
```
rabbit_s3    ██████████████████████████████ 4259 ops/s
liteio       ████████████████████████████ 4043 ops/s
usagi_s3     █████████████████████████ 3681 ops/s
minio        █████████ 1379 ops/s
```

**Latency (P50)**
```
rabbit_s3    ██████████ 217.9us
liteio       ██████████ 233.0us
usagi_s3     ████████████ 257.8us
minio        ██████████████████████████████ 638.1us
```

### Write/100MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 131.95 MB/s | 757.9ms | 757.9ms | 757.9ms | 0 |
| usagi_s3 | 122.67 MB/s | 835.2ms | 835.2ms | 835.2ms | 0 |
| rabbit_s3 | 120.41 MB/s | 829.1ms | 829.1ms | 829.1ms | 0 |
| minio | 72.92 MB/s | 1.34s | 1.34s | 1.34s | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 131.95 MB/s
usagi_s3     ███████████████████████████ 122.67 MB/s
rabbit_s3    ███████████████████████████ 120.41 MB/s
minio        ████████████████ 72.92 MB/s
```

**Latency (P50)**
```
liteio       ████████████████ 757.9ms
usagi_s3     ██████████████████ 835.2ms
rabbit_s3    ██████████████████ 829.1ms
minio        ██████████████████████████████ 1.34s
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit_s3 | 122.44 MB/s | 80.1ms | 85.5ms | 85.5ms | 0 |
| usagi_s3 | 115.53 MB/s | 84.5ms | 102.4ms | 102.4ms | 0 |
| minio | 72.25 MB/s | 134.3ms | 168.5ms | 168.5ms | 0 |
| liteio | 50.30 MB/s | 231.9ms | 261.1ms | 261.1ms | 0 |

**Throughput**
```
rabbit_s3    ██████████████████████████████ 122.44 MB/s
usagi_s3     ████████████████████████████ 115.53 MB/s
minio        █████████████████ 72.25 MB/s
liteio       ████████████ 50.30 MB/s
```

**Latency (P50)**
```
rabbit_s3    ██████████ 80.1ms
usagi_s3     ██████████ 84.5ms
minio        █████████████████ 134.3ms
liteio       ██████████████████████████████ 231.9ms
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 1.74 MB/s | 505.1us | 847.5us | 1.2ms | 0 |
| rabbit_s3 | 1.66 MB/s | 561.9us | 845.4us | 1.1ms | 0 |
| liteio | 0.45 MB/s | 1.9ms | 3.3ms | 4.9ms | 0 |
| minio | 0.23 MB/s | 2.5ms | 12.3ms | 14.1ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 1.74 MB/s
rabbit_s3    ████████████████████████████ 1.66 MB/s
liteio       ███████ 0.45 MB/s
minio        ███ 0.23 MB/s
```

**Latency (P50)**
```
usagi_s3     ██████ 505.1us
rabbit_s3    ██████ 561.9us
liteio       ██████████████████████ 1.9ms
minio        ██████████████████████████████ 2.5ms
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit_s3 | 129.27 MB/s | 7.5ms | 10.3ms | 11.8ms | 0 |
| usagi_s3 | 113.15 MB/s | 8.6ms | 11.2ms | 13.0ms | 0 |
| liteio | 64.90 MB/s | 13.9ms | 19.7ms | 21.7ms | 0 |
| minio | 50.40 MB/s | 17.7ms | 28.9ms | 53.6ms | 0 |

**Throughput**
```
rabbit_s3    ██████████████████████████████ 129.27 MB/s
usagi_s3     ██████████████████████████ 113.15 MB/s
liteio       ███████████████ 64.90 MB/s
minio        ███████████ 50.40 MB/s
```

**Latency (P50)**
```
rabbit_s3    ████████████ 7.5ms
usagi_s3     ██████████████ 8.6ms
liteio       ███████████████████████ 13.9ms
minio        ██████████████████████████████ 17.7ms
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rabbit_s3 | 68.09 MB/s | 846.3us | 1.2ms | 1.7ms | 0 |
| usagi_s3 | 65.81 MB/s | 860.6us | 1.3ms | 1.8ms | 0 |
| liteio | 21.32 MB/s | 2.7ms | 4.0ms | 5.4ms | 0 |
| minio | 15.78 MB/s | 3.4ms | 7.2ms | 9.4ms | 0 |

**Throughput**
```
rabbit_s3    ██████████████████████████████ 68.09 MB/s
usagi_s3     ████████████████████████████ 65.81 MB/s
liteio       █████████ 21.32 MB/s
minio        ██████ 15.78 MB/s
```

**Latency (P50)**
```
rabbit_s3    ███████ 846.3us
usagi_s3     ███████ 860.6us
liteio       ███████████████████████ 2.7ms
minio        ██████████████████████████████ 3.4ms
```

## Resource Usage

| Driver | Memory | RSS | Cache | CPU | Volume | Block I/O |
|--------|--------|-----|-------|-----|--------|----------|
| liteio | 1.029GiB / 7.653GiB | 1053.7 MB | - | 2.8% | 1495.0 MB | 4.46MB / 1.84GB |
| minio | 887.5MiB / 7.653GiB | 887.5 MB | - | 3.0% | 1730.0 MB | 4.49MB / 1.77GB |
| rabbit_s3 | 1.609GiB / 7.653GiB | 1647.6 MB | - | 1.7% | 14807.0 MB | 19.4MB / 14.9GB |
| usagi_s3 | 1.479GiB / 7.653GiB | 1514.5 MB | - | 1.7% | 1715.2 MB | 1.08MB / 2.03GB |

> **Note:** RSS = actual application memory. Cache = OS page cache (reclaimable).

## Recommendations

- **Write-heavy workloads:** liteio
- **Read-heavy workloads:** rabbit_s3

---

*Generated by storage benchmark CLI*
