# Storage Benchmark Report

**Generated:** 2026-01-15T11:00:18+07:00

**Go Version:** go1.25.5

**Platform:** darwin/arm64

## Executive Summary

### Summary

**Overall Winner:** liteio_mem (won 21/51 benchmarks, 41%)

| Rank | Driver | Wins | Win Rate |
|------|--------|------|----------|
| 1 | liteio_mem | 21 | 41% |
| 2 | liteio | 11 | 22% |
| 3 | rustfs | 9 | 18% |
| 4 | seaweedfs | 5 | 10% |
| 5 | minio | 5 | 10% |

### Performance Leaders

| Operation | Leader | Performance | Margin |
|-----------|--------|-------------|--------|
| Small Read (1KB) | liteio | 4.8 MB/s | close |
| Small Write (1KB) | liteio_mem | 1.5 MB/s | close |
| Large Read (10MB) | minio | 298.3 MB/s | close |
| Large Write (10MB) | liteio | 189.2 MB/s | close |
| Delete | liteio_mem | 5.7K ops/s | +41% vs liteio |
| Stat | minio | 4.4K ops/s | close |
| List (100 objects) | liteio_mem | 1.2K ops/s | close |
| Copy | liteio_mem | 1.5 MB/s | +11% vs liteio |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (100MB+) | **liteio** | 193 MB/s | Best for media, backups |
| Large File Downloads (100MB) | **minio** | 324 MB/s | Best for streaming, CDN |
| Small File Operations | **liteio_mem** | 3087 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **minio** | - | Best for multi-user apps |
| Memory Constrained | **liteio_mem** | 71 MB RAM | Best for edge/embedded |

### Large File Performance (100MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| liteio | 193.2 | 252.2 | 518.6ms | 394.9ms |
| liteio_mem | 178.7 | 249.7 | 561.2ms | 398.0ms |
| localstack | 142.9 | 262.1 | 670.7ms | 382.4ms |
| minio | 170.8 | 324.3 | 561.7ms | 308.6ms |
| rustfs | 165.9 | 297.5 | 603.1ms | 328.6ms |
| seaweedfs | 190.9 | 254.0 | 528.5ms | 393.8ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| liteio | 1239 | 4922 | 728.2us | 201.4us |
| liteio_mem | 1578 | 4597 | 538.4us | 203.8us |
| localstack | 1160 | 1434 | 727.2us | 669.5us |
| minio | 972 | 2940 | 823.0us | 334.2us |
| rustfs | 1521 | 2296 | 646.9us | 438.4us |
| seaweedfs | 1489 | 2638 | 663.4us | 372.6us |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| liteio | 4316 | 1129 | 4022 |
| liteio_mem | 4076 | 1240 | 5681 |
| localstack | 1464 | 339 | 1658 |
| minio | 4399 | 607 | 2947 |
| rustfs | 3266 | 169 | 1217 |
| seaweedfs | 2997 | 692 | 2130 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| liteio | 1.33 | 0.39 | 0.25 | 0.08 | 0.07 | 0.10 |
| liteio_mem | 1.48 | 0.35 | 0.23 | 0.08 | 0.09 | 0.09 |
| localstack | 1.26 | 0.16 | 0.02 | 0.02 | 0.02 | 0.01 |
| minio | 0.96 | 0.28 | 0.18 | 0.10 | 0.06 | 0.07 |
| rustfs | 1.36 | 0.48 | - | - | - | - |
| seaweedfs | 0.86 | 0.39 | 0.22 | 0.11 | 0.11 | 0.13 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| liteio | 2.83 | 0.87 | 0.61 | 0.46 | 0.29 | 0.30 |
| liteio_mem | 2.57 | 0.80 | 0.61 | 0.50 | 0.34 | 0.57 |
| localstack | 1.27 | 0.16 | 0.04 | 0.03 | 0.02 | 0.02 |
| minio | 2.78 | 1.10 | 0.64 | 0.35 | 0.21 | 0.17 |
| rustfs | 1.85 | 0.76 | - | - | - | - |
| seaweedfs | 2.18 | 0.76 | 0.38 | 0.18 | 0.17 | 0.20 |

*\* indicates errors occurred*

### File Count Performance

Performance with varying numbers of files (1KB each).

**Write N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|------|
| liteio | 673.6us | 7.3ms | 92.2ms | 823.7ms | 5.90s |
| liteio_mem | 661.7us | 6.3ms | 69.1ms | 717.2ms | 5.73s |
| localstack | 850.3us | 8.7ms | 105.6ms | 829.5ms | 7.95s |
| minio | 1.3ms | 9.0ms | 94.5ms | 1.11s | 8.10s |
| rustfs | 803.9us | 10.2ms | 64.8ms | 639.7ms | 6.56s |
| seaweedfs | 802.2us | 7.2ms | 76.4ms | 783.2ms | 7.73s |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|------|
| liteio | 259.6us | 360.5us | 1.3ms | 6.8ms | 214.8ms |
| liteio_mem | 291.2us | 313.8us | 1.3ms | 6.2ms | 185.1ms |
| localstack | 1.1ms | 1.3ms | 5.2ms | 23.0ms | 290.7ms |
| minio | 511.7us | 634.0us | 1.8ms | 12.6ms | 161.4ms |
| rustfs | 1.4ms | 1.9ms | 6.4ms | 52.6ms | 707.3ms |
| seaweedfs | 691.7us | 812.0us | 1.7ms | 8.9ms | 93.3ms |

*\* indicates errors occurred*

### Skipped Benchmarks

Some benchmarks were skipped due to driver limitations:

- **rustfs**: 8 skipped
  - ParallelWrite/1KB/C25 (exceeds max concurrency 10)
  - ParallelRead/1KB/C25 (exceeds max concurrency 10)
  - ParallelWrite/1KB/C50 (exceeds max concurrency 10)
  - ParallelRead/1KB/C50 (exceeds max concurrency 10)
  - ParallelWrite/1KB/C100 (exceeds max concurrency 10)
  - ParallelRead/1KB/C100 (exceeds max concurrency 10)
  - ParallelWrite/1KB/C200 (exceeds max concurrency 10)
  - ParallelRead/1KB/C200 (exceeds max concurrency 10)

### Resource Usage Summary

| Driver | Memory | CPU |
|--------|--------|-----|
| liteio | 97.7 MB | 0.0% |
| liteio_mem | 70.8 MB | 1.1% |
| localstack | 388.8 MB | 0.0% |
| minio | 396.1 MB | 0.0% |
| rustfs | 360.3 MB | 0.1% |
| seaweedfs | 126.7 MB | 0.0% |

---

## Configuration

| Parameter | Value |
|-----------|-------|
| Iterations | 100 |
| Warmup | 10 |
| Concurrency | 200 |
| Timeout | 30s |

## Drivers Tested

- **liteio** (51 benchmarks)
- **liteio_mem** (51 benchmarks)
- **localstack** (51 benchmarks)
- **minio** (51 benchmarks)
- **rustfs** (43 benchmarks)
- **seaweedfs** (51 benchmarks)

*Reference baseline: devnull (excluded from comparisons)*

## Detailed Results

### Copy/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1.47 MB/s | 594.2us | 964.6us | 1.3ms | 0 |
| liteio | 1.32 MB/s | 708.8us | 955.0us | 1.1ms | 0 |
| localstack | 1.17 MB/s | 781.8us | 1.1ms | 1.3ms | 0 |
| rustfs | 1.09 MB/s | 902.0us | 1.1ms | 1.2ms | 0 |
| minio | 0.98 MB/s | 836.9us | 1.6ms | 1.6ms | 0 |
| seaweedfs | 0.48 MB/s | 1.1ms | 2.7ms | 6.5ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 1.47 MB/s
liteio       ██████████████████████████ 1.32 MB/s
localstack   ███████████████████████ 1.17 MB/s
rustfs       ██████████████████████ 1.09 MB/s
minio        ████████████████████ 0.98 MB/s
seaweedfs    █████████ 0.48 MB/s
```

**Latency (P50)**
```
liteio_mem   ████████████████ 594.2us
liteio       ███████████████████ 708.8us
localstack   █████████████████████ 781.8us
rustfs       █████████████████████████ 902.0us
minio        ███████████████████████ 836.9us
seaweedfs    ██████████████████████████████ 1.1ms
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 5681 ops/s | 160.4us | 225.9us | 243.7us | 0 |
| liteio | 4022 ops/s | 203.5us | 460.8us | 552.8us | 0 |
| minio | 2947 ops/s | 328.8us | 387.0us | 482.8us | 0 |
| seaweedfs | 2130 ops/s | 337.0us | 1.2ms | 1.3ms | 0 |
| localstack | 1658 ops/s | 586.0us | 663.1us | 1.1ms | 0 |
| rustfs | 1217 ops/s | 782.6us | 1.1ms | 1.5ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 5681 ops/s
liteio       █████████████████████ 4022 ops/s
minio        ███████████████ 2947 ops/s
seaweedfs    ███████████ 2130 ops/s
localstack   ████████ 1658 ops/s
rustfs       ██████ 1217 ops/s
```

**Latency (P50)**
```
liteio_mem   ██████ 160.4us
liteio       ███████ 203.5us
minio        ████████████ 328.8us
seaweedfs    ████████████ 337.0us
localstack   ██████████████████████ 586.0us
rustfs       ██████████████████████████████ 782.6us
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.15 MB/s | 623.5us | 749.7us | 850.2us | 0 |
| liteio_mem | 0.14 MB/s | 621.9us | 960.9us | 1.3ms | 0 |
| seaweedfs | 0.13 MB/s | 672.1us | 819.3us | 1.2ms | 0 |
| localstack | 0.11 MB/s | 814.1us | 1.1ms | 1.1ms | 0 |
| rustfs | 0.10 MB/s | 800.3us | 1.4ms | 1.7ms | 0 |
| minio | 0.10 MB/s | 805.2us | 1.6ms | 1.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.15 MB/s
liteio_mem   ███████████████████████████ 0.14 MB/s
seaweedfs    ███████████████████████████ 0.13 MB/s
localstack   ██████████████████████ 0.11 MB/s
rustfs       ████████████████████ 0.10 MB/s
minio        ███████████████████ 0.10 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████████ 623.5us
liteio_mem   ██████████████████████ 621.9us
seaweedfs    ████████████████████████ 672.1us
localstack   ██████████████████████████████ 814.1us
rustfs       █████████████████████████████ 800.3us
minio        █████████████████████████████ 805.2us
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 2668 ops/s | 366.0us | 453.6us | 533.3us | 0 |
| liteio_mem | 1559 ops/s | 573.3us | 914.8us | 1.1ms | 0 |
| liteio | 1165 ops/s | 603.5us | 900.9us | 1.9ms | 0 |
| localstack | 1138 ops/s | 831.4us | 1.1ms | 1.5ms | 0 |
| minio | 1039 ops/s | 776.3us | 1.5ms | 1.5ms | 0 |
| rustfs | 961 ops/s | 782.6us | 1.9ms | 3.5ms | 0 |

**Throughput**
```
seaweedfs    ██████████████████████████████ 2668 ops/s
liteio_mem   █████████████████ 1559 ops/s
liteio       █████████████ 1165 ops/s
localstack   ████████████ 1138 ops/s
minio        ███████████ 1039 ops/s
rustfs       ██████████ 961 ops/s
```

**Latency (P50)**
```
seaweedfs    █████████████ 366.0us
liteio_mem   ████████████████████ 573.3us
liteio       █████████████████████ 603.5us
localstack   ██████████████████████████████ 831.4us
minio        ████████████████████████████ 776.3us
rustfs       ████████████████████████████ 782.6us
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.15 MB/s | 614.9us | 797.0us | 906.8us | 0 |
| liteio_mem | 0.14 MB/s | 670.5us | 902.5us | 948.9us | 0 |
| seaweedfs | 0.12 MB/s | 742.5us | 965.8us | 1.1ms | 0 |
| rustfs | 0.10 MB/s | 738.5us | 1.0ms | 3.6ms | 0 |
| minio | 0.10 MB/s | 842.1us | 1.6ms | 1.7ms | 0 |
| localstack | 0.10 MB/s | 868.7us | 1.6ms | 1.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.15 MB/s
liteio_mem   ███████████████████████████ 0.14 MB/s
seaweedfs    ████████████████████████ 0.12 MB/s
rustfs       ████████████████████ 0.10 MB/s
minio        ████████████████████ 0.10 MB/s
localstack   ███████████████████ 0.10 MB/s
```

**Latency (P50)**
```
liteio       █████████████████████ 614.9us
liteio_mem   ███████████████████████ 670.5us
seaweedfs    █████████████████████████ 742.5us
rustfs       █████████████████████████ 738.5us
minio        █████████████████████████████ 842.1us
localstack   ██████████████████████████████ 868.7us
```

### FileCount/Delete/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 4572 ops/s | 218.7us | 218.7us | 218.7us | 0 |
| liteio_mem | 3479 ops/s | 287.5us | 287.5us | 287.5us | 0 |
| minio | 2150 ops/s | 465.2us | 465.2us | 465.2us | 0 |
| seaweedfs | 1404 ops/s | 712.0us | 712.0us | 712.0us | 0 |
| localstack | 1395 ops/s | 716.7us | 716.7us | 716.7us | 0 |
| rustfs | 955 ops/s | 1.0ms | 1.0ms | 1.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4572 ops/s
liteio_mem   ██████████████████████ 3479 ops/s
minio        ██████████████ 2150 ops/s
seaweedfs    █████████ 1404 ops/s
localstack   █████████ 1395 ops/s
rustfs       ██████ 955 ops/s
```

**Latency (P50)**
```
liteio       ██████ 218.7us
liteio_mem   ████████ 287.5us
minio        █████████████ 465.2us
seaweedfs    ████████████████████ 712.0us
localstack   ████████████████████ 716.7us
rustfs       ██████████████████████████████ 1.0ms
```

### FileCount/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 502 ops/s | 2.0ms | 2.0ms | 2.0ms | 0 |
| liteio | 476 ops/s | 2.1ms | 2.1ms | 2.1ms | 0 |
| minio | 276 ops/s | 3.6ms | 3.6ms | 3.6ms | 0 |
| seaweedfs | 270 ops/s | 3.7ms | 3.7ms | 3.7ms | 0 |
| rustfs | 142 ops/s | 7.0ms | 7.0ms | 7.0ms | 0 |
| localstack | 120 ops/s | 8.3ms | 8.3ms | 8.3ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 502 ops/s
liteio       ████████████████████████████ 476 ops/s
minio        ████████████████ 276 ops/s
seaweedfs    ████████████████ 270 ops/s
rustfs       ████████ 142 ops/s
localstack   ███████ 120 ops/s
```

**Latency (P50)**
```
liteio_mem   ███████ 2.0ms
liteio       ███████ 2.1ms
minio        █████████████ 3.6ms
seaweedfs    █████████████ 3.7ms
rustfs       █████████████████████████ 7.0ms
localstack   ██████████████████████████████ 8.3ms
```

### FileCount/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 45 ops/s | 22.1ms | 22.1ms | 22.1ms | 0 |
| seaweedfs | 30 ops/s | 33.2ms | 33.2ms | 33.2ms | 0 |
| minio | 28 ops/s | 35.3ms | 35.3ms | 35.3ms | 0 |
| liteio | 27 ops/s | 36.4ms | 36.4ms | 36.4ms | 0 |
| rustfs | 13 ops/s | 79.3ms | 79.3ms | 79.3ms | 0 |
| localstack | 12 ops/s | 80.4ms | 80.4ms | 80.4ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 45 ops/s
seaweedfs    ███████████████████ 30 ops/s
minio        ██████████████████ 28 ops/s
liteio       ██████████████████ 27 ops/s
rustfs       ████████ 13 ops/s
localstack   ████████ 12 ops/s
```

**Latency (P50)**
```
liteio_mem   ████████ 22.1ms
seaweedfs    ████████████ 33.2ms
minio        █████████████ 35.3ms
liteio       █████████████ 36.4ms
rustfs       █████████████████████████████ 79.3ms
localstack   ██████████████████████████████ 80.4ms
```

### FileCount/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 5 ops/s | 209.1ms | 209.1ms | 209.1ms | 0 |
| liteio | 4 ops/s | 227.4ms | 227.4ms | 227.4ms | 0 |
| seaweedfs | 3 ops/s | 355.8ms | 355.8ms | 355.8ms | 0 |
| minio | 3 ops/s | 357.2ms | 357.2ms | 357.2ms | 0 |
| localstack | 2 ops/s | 664.3ms | 664.3ms | 664.3ms | 0 |
| rustfs | 1 ops/s | 767.5ms | 767.5ms | 767.5ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 5 ops/s
liteio       ███████████████████████████ 4 ops/s
seaweedfs    █████████████████ 3 ops/s
minio        █████████████████ 3 ops/s
localstack   █████████ 2 ops/s
rustfs       ████████ 1 ops/s
```

**Latency (P50)**
```
liteio_mem   ████████ 209.1ms
liteio       ████████ 227.4ms
seaweedfs    █████████████ 355.8ms
minio        █████████████ 357.2ms
localstack   █████████████████████████ 664.3ms
rustfs       ██████████████████████████████ 767.5ms
```

### FileCount/Delete/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 0 ops/s | 2.09s | 2.09s | 2.09s | 0 |
| liteio | 0 ops/s | 2.18s | 2.18s | 2.18s | 0 |
| seaweedfs | 0 ops/s | 3.48s | 3.48s | 3.48s | 0 |
| minio | 0 ops/s | 3.57s | 3.57s | 3.57s | 0 |
| localstack | 0 ops/s | 6.50s | 6.50s | 6.50s | 0 |
| rustfs | 0 ops/s | 8.69s | 8.69s | 8.69s | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 0 ops/s
liteio       ████████████████████████████ 0 ops/s
seaweedfs    █████████████████ 0 ops/s
minio        █████████████████ 0 ops/s
localstack   █████████ 0 ops/s
rustfs       ███████ 0 ops/s
```

**Latency (P50)**
```
liteio_mem   ███████ 2.09s
liteio       ███████ 2.18s
seaweedfs    ████████████ 3.48s
minio        ████████████ 3.57s
localstack   ██████████████████████ 6.50s
rustfs       ██████████████████████████████ 8.69s
```

### FileCount/List/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 3852 ops/s | 259.6us | 259.6us | 259.6us | 0 |
| liteio_mem | 3434 ops/s | 291.2us | 291.2us | 291.2us | 0 |
| minio | 1954 ops/s | 511.7us | 511.7us | 511.7us | 0 |
| seaweedfs | 1446 ops/s | 691.7us | 691.7us | 691.7us | 0 |
| localstack | 893 ops/s | 1.1ms | 1.1ms | 1.1ms | 0 |
| rustfs | 690 ops/s | 1.4ms | 1.4ms | 1.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 3852 ops/s
liteio_mem   ██████████████████████████ 3434 ops/s
minio        ███████████████ 1954 ops/s
seaweedfs    ███████████ 1446 ops/s
localstack   ██████ 893 ops/s
rustfs       █████ 690 ops/s
```

**Latency (P50)**
```
liteio       █████ 259.6us
liteio_mem   ██████ 291.2us
minio        ██████████ 511.7us
seaweedfs    ██████████████ 691.7us
localstack   ███████████████████████ 1.1ms
rustfs       ██████████████████████████████ 1.4ms
```

### FileCount/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 3187 ops/s | 313.8us | 313.8us | 313.8us | 0 |
| liteio | 2774 ops/s | 360.5us | 360.5us | 360.5us | 0 |
| minio | 1577 ops/s | 634.0us | 634.0us | 634.0us | 0 |
| seaweedfs | 1231 ops/s | 812.0us | 812.0us | 812.0us | 0 |
| localstack | 749 ops/s | 1.3ms | 1.3ms | 1.3ms | 0 |
| rustfs | 526 ops/s | 1.9ms | 1.9ms | 1.9ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 3187 ops/s
liteio       ██████████████████████████ 2774 ops/s
minio        ██████████████ 1577 ops/s
seaweedfs    ███████████ 1231 ops/s
localstack   ███████ 749 ops/s
rustfs       ████ 526 ops/s
```

**Latency (P50)**
```
liteio_mem   ████ 313.8us
liteio       █████ 360.5us
minio        █████████ 634.0us
seaweedfs    ████████████ 812.0us
localstack   █████████████████████ 1.3ms
rustfs       ██████████████████████████████ 1.9ms
```

### FileCount/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 800 ops/s | 1.3ms | 1.3ms | 1.3ms | 0 |
| liteio | 784 ops/s | 1.3ms | 1.3ms | 1.3ms | 0 |
| seaweedfs | 575 ops/s | 1.7ms | 1.7ms | 1.7ms | 0 |
| minio | 569 ops/s | 1.8ms | 1.8ms | 1.8ms | 0 |
| localstack | 193 ops/s | 5.2ms | 5.2ms | 5.2ms | 0 |
| rustfs | 156 ops/s | 6.4ms | 6.4ms | 6.4ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 800 ops/s
liteio       █████████████████████████████ 784 ops/s
seaweedfs    █████████████████████ 575 ops/s
minio        █████████████████████ 569 ops/s
localstack   ███████ 193 ops/s
rustfs       █████ 156 ops/s
```

**Latency (P50)**
```
liteio_mem   █████ 1.3ms
liteio       █████ 1.3ms
seaweedfs    ████████ 1.7ms
minio        ████████ 1.8ms
localstack   ████████████████████████ 5.2ms
rustfs       ██████████████████████████████ 6.4ms
```

### FileCount/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 161 ops/s | 6.2ms | 6.2ms | 6.2ms | 0 |
| liteio | 147 ops/s | 6.8ms | 6.8ms | 6.8ms | 0 |
| seaweedfs | 112 ops/s | 8.9ms | 8.9ms | 8.9ms | 0 |
| minio | 79 ops/s | 12.6ms | 12.6ms | 12.6ms | 0 |
| localstack | 43 ops/s | 23.0ms | 23.0ms | 23.0ms | 0 |
| rustfs | 19 ops/s | 52.6ms | 52.6ms | 52.6ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 161 ops/s
liteio       ███████████████████████████ 147 ops/s
seaweedfs    ████████████████████ 112 ops/s
minio        ██████████████ 79 ops/s
localstack   ████████ 43 ops/s
rustfs       ███ 19 ops/s
```

**Latency (P50)**
```
liteio_mem   ███ 6.2ms
liteio       ███ 6.8ms
seaweedfs    █████ 8.9ms
minio        ███████ 12.6ms
localstack   █████████████ 23.0ms
rustfs       ██████████████████████████████ 52.6ms
```

### FileCount/List/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 11 ops/s | 93.3ms | 93.3ms | 93.3ms | 0 |
| minio | 6 ops/s | 161.4ms | 161.4ms | 161.4ms | 0 |
| liteio_mem | 5 ops/s | 185.1ms | 185.1ms | 185.1ms | 0 |
| liteio | 5 ops/s | 214.8ms | 214.8ms | 214.8ms | 0 |
| localstack | 3 ops/s | 290.7ms | 290.7ms | 290.7ms | 0 |
| rustfs | 1 ops/s | 707.3ms | 707.3ms | 707.3ms | 0 |

**Throughput**
```
seaweedfs    ██████████████████████████████ 11 ops/s
minio        █████████████████ 6 ops/s
liteio_mem   ███████████████ 5 ops/s
liteio       █████████████ 5 ops/s
localstack   █████████ 3 ops/s
rustfs       ███ 1 ops/s
```

**Latency (P50)**
```
seaweedfs    ███ 93.3ms
minio        ██████ 161.4ms
liteio_mem   ███████ 185.1ms
liteio       █████████ 214.8ms
localstack   ████████████ 290.7ms
rustfs       ██████████████████████████████ 707.3ms
```

### FileCount/Write/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1.48 MB/s | 661.7us | 661.7us | 661.7us | 0 |
| liteio | 1.45 MB/s | 673.6us | 673.6us | 673.6us | 0 |
| seaweedfs | 1.22 MB/s | 802.2us | 802.2us | 802.2us | 0 |
| rustfs | 1.21 MB/s | 803.9us | 803.9us | 803.9us | 0 |
| localstack | 1.15 MB/s | 850.3us | 850.3us | 850.3us | 0 |
| minio | 0.73 MB/s | 1.3ms | 1.3ms | 1.3ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 1.48 MB/s
liteio       █████████████████████████████ 1.45 MB/s
seaweedfs    ████████████████████████ 1.22 MB/s
rustfs       ████████████████████████ 1.21 MB/s
localstack   ███████████████████████ 1.15 MB/s
minio        ██████████████ 0.73 MB/s
```

**Latency (P50)**
```
liteio_mem   ██████████████ 661.7us
liteio       ███████████████ 673.6us
seaweedfs    ██████████████████ 802.2us
rustfs       ██████████████████ 803.9us
localstack   ███████████████████ 850.3us
minio        ██████████████████████████████ 1.3ms
```

### FileCount/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1.56 MB/s | 6.3ms | 6.3ms | 6.3ms | 0 |
| seaweedfs | 1.36 MB/s | 7.2ms | 7.2ms | 7.2ms | 0 |
| liteio | 1.34 MB/s | 7.3ms | 7.3ms | 7.3ms | 0 |
| localstack | 1.12 MB/s | 8.7ms | 8.7ms | 8.7ms | 0 |
| minio | 1.08 MB/s | 9.0ms | 9.0ms | 9.0ms | 0 |
| rustfs | 0.95 MB/s | 10.2ms | 10.2ms | 10.2ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 1.56 MB/s
seaweedfs    ██████████████████████████ 1.36 MB/s
liteio       █████████████████████████ 1.34 MB/s
localstack   █████████████████████ 1.12 MB/s
minio        ████████████████████ 1.08 MB/s
rustfs       ██████████████████ 0.95 MB/s
```

**Latency (P50)**
```
liteio_mem   ██████████████████ 6.3ms
seaweedfs    █████████████████████ 7.2ms
liteio       █████████████████████ 7.3ms
localstack   █████████████████████████ 8.7ms
minio        ██████████████████████████ 9.0ms
rustfs       ██████████████████████████████ 10.2ms
```

### FileCount/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 1.51 MB/s | 64.8ms | 64.8ms | 64.8ms | 0 |
| liteio_mem | 1.41 MB/s | 69.1ms | 69.1ms | 69.1ms | 0 |
| seaweedfs | 1.28 MB/s | 76.4ms | 76.4ms | 76.4ms | 0 |
| liteio | 1.06 MB/s | 92.2ms | 92.2ms | 92.2ms | 0 |
| minio | 1.03 MB/s | 94.5ms | 94.5ms | 94.5ms | 0 |
| localstack | 0.93 MB/s | 105.6ms | 105.6ms | 105.6ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 1.51 MB/s
liteio_mem   ████████████████████████████ 1.41 MB/s
seaweedfs    █████████████████████████ 1.28 MB/s
liteio       █████████████████████ 1.06 MB/s
minio        ████████████████████ 1.03 MB/s
localstack   ██████████████████ 0.93 MB/s
```

**Latency (P50)**
```
rustfs       ██████████████████ 64.8ms
liteio_mem   ███████████████████ 69.1ms
seaweedfs    █████████████████████ 76.4ms
liteio       ██████████████████████████ 92.2ms
minio        ██████████████████████████ 94.5ms
localstack   ██████████████████████████████ 105.6ms
```

### FileCount/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 1.53 MB/s | 639.7ms | 639.7ms | 639.7ms | 0 |
| liteio_mem | 1.36 MB/s | 717.2ms | 717.2ms | 717.2ms | 0 |
| seaweedfs | 1.25 MB/s | 783.2ms | 783.2ms | 783.2ms | 0 |
| liteio | 1.19 MB/s | 823.7ms | 823.7ms | 823.7ms | 0 |
| localstack | 1.18 MB/s | 829.5ms | 829.5ms | 829.5ms | 0 |
| minio | 0.88 MB/s | 1.11s | 1.11s | 1.11s | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 1.53 MB/s
liteio_mem   ██████████████████████████ 1.36 MB/s
seaweedfs    ████████████████████████ 1.25 MB/s
liteio       ███████████████████████ 1.19 MB/s
localstack   ███████████████████████ 1.18 MB/s
minio        █████████████████ 0.88 MB/s
```

**Latency (P50)**
```
rustfs       █████████████████ 639.7ms
liteio_mem   ███████████████████ 717.2ms
seaweedfs    █████████████████████ 783.2ms
liteio       ██████████████████████ 823.7ms
localstack   ██████████████████████ 829.5ms
minio        ██████████████████████████████ 1.11s
```

### FileCount/Write/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1.70 MB/s | 5.73s | 5.73s | 5.73s | 0 |
| liteio | 1.65 MB/s | 5.90s | 5.90s | 5.90s | 0 |
| rustfs | 1.49 MB/s | 6.56s | 6.56s | 6.56s | 0 |
| seaweedfs | 1.26 MB/s | 7.73s | 7.73s | 7.73s | 0 |
| localstack | 1.23 MB/s | 7.95s | 7.95s | 7.95s | 0 |
| minio | 1.21 MB/s | 8.10s | 8.10s | 8.10s | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 1.70 MB/s
liteio       █████████████████████████████ 1.65 MB/s
rustfs       ██████████████████████████ 1.49 MB/s
seaweedfs    ██████████████████████ 1.26 MB/s
localstack   █████████████████████ 1.23 MB/s
minio        █████████████████████ 1.21 MB/s
```

**Latency (P50)**
```
liteio_mem   █████████████████████ 5.73s
liteio       █████████████████████ 5.90s
rustfs       ████████████████████████ 6.56s
seaweedfs    ████████████████████████████ 7.73s
localstack   █████████████████████████████ 7.95s
minio        ██████████████████████████████ 8.10s
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1240 ops/s | 763.6us | 971.8us | 1.0ms | 0 |
| liteio | 1129 ops/s | 825.5us | 1.2ms | 1.4ms | 0 |
| seaweedfs | 692 ops/s | 1.4ms | 1.6ms | 2.0ms | 0 |
| minio | 607 ops/s | 1.6ms | 1.8ms | 2.0ms | 0 |
| localstack | 339 ops/s | 2.6ms | 5.0ms | 6.4ms | 0 |
| rustfs | 169 ops/s | 6.0ms | 6.5ms | 7.4ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 1240 ops/s
liteio       ███████████████████████████ 1129 ops/s
seaweedfs    ████████████████ 692 ops/s
minio        ██████████████ 607 ops/s
localstack   ████████ 339 ops/s
rustfs       ████ 169 ops/s
```

**Latency (P50)**
```
liteio_mem   ███ 763.6us
liteio       ████ 825.5us
seaweedfs    ███████ 1.4ms
minio        ███████ 1.6ms
localstack   █████████████ 2.6ms
rustfs       ██████████████████████████████ 6.0ms
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 6.76 MB/s | 2.1ms | 3.4ms | 10.1ms | 0 |
| seaweedfs | 1.43 MB/s | 11.4ms | 13.5ms | 13.8ms | 0 |
| minio | 1.35 MB/s | 10.9ms | 18.9ms | 19.2ms | 0 |
| liteio_mem | 1.31 MB/s | 11.9ms | 19.0ms | 19.6ms | 0 |
| liteio | 1.19 MB/s | 12.0ms | 34.2ms | 34.7ms | 0 |
| localstack | 0.27 MB/s | 55.2ms | 62.6ms | 63.5ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 6.76 MB/s
seaweedfs    ██████ 1.43 MB/s
minio        █████ 1.35 MB/s
liteio_mem   █████ 1.31 MB/s
liteio       █████ 1.19 MB/s
localstack   █ 0.27 MB/s
```

**Latency (P50)**
```
rustfs       █ 2.1ms
seaweedfs    ██████ 11.4ms
minio        █████ 10.9ms
liteio_mem   ██████ 11.9ms
liteio       ██████ 12.0ms
localstack   ██████████████████████████████ 55.2ms
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 8.51 MB/s | 1.7ms | 3.3ms | 3.8ms | 0 |
| seaweedfs | 1.58 MB/s | 9.4ms | 11.9ms | 12.1ms | 0 |
| minio | 1.44 MB/s | 11.2ms | 11.9ms | 13.0ms | 0 |
| liteio | 1.37 MB/s | 12.0ms | 13.8ms | 13.9ms | 0 |
| liteio_mem | 1.15 MB/s | 14.9ms | 16.1ms | 16.4ms | 0 |
| localstack | 0.24 MB/s | 65.1ms | 66.3ms | 66.4ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 8.51 MB/s
seaweedfs    █████ 1.58 MB/s
minio        █████ 1.44 MB/s
liteio       ████ 1.37 MB/s
liteio_mem   ████ 1.15 MB/s
localstack   █ 0.24 MB/s
```

**Latency (P50)**
```
rustfs       █ 1.7ms
seaweedfs    ████ 9.4ms
minio        █████ 11.2ms
liteio       █████ 12.0ms
liteio_mem   ██████ 14.9ms
localstack   ██████████████████████████████ 65.1ms
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 4.36 MB/s | 3.0ms | 7.4ms | 8.2ms | 0 |
| liteio | 1.42 MB/s | 11.3ms | 17.1ms | 19.0ms | 0 |
| seaweedfs | 0.97 MB/s | 17.6ms | 20.3ms | 21.5ms | 0 |
| minio | 0.69 MB/s | 23.5ms | 31.2ms | 31.4ms | 0 |
| liteio_mem | 0.61 MB/s | 27.6ms | 34.0ms | 34.8ms | 0 |
| localstack | 0.23 MB/s | 71.7ms | 74.9ms | 76.0ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 4.36 MB/s
liteio       █████████ 1.42 MB/s
seaweedfs    ██████ 0.97 MB/s
minio        ████ 0.69 MB/s
liteio_mem   ████ 0.61 MB/s
localstack   █ 0.23 MB/s
```

**Latency (P50)**
```
rustfs       █ 3.0ms
liteio       ████ 11.3ms
seaweedfs    ███████ 17.6ms
minio        █████████ 23.5ms
liteio_mem   ███████████ 27.6ms
localstack   ██████████████████████████████ 71.7ms
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 175.76 MB/s | 85.4ms | 98.4ms | 98.4ms | 0 |
| minio | 162.88 MB/s | 89.5ms | 105.8ms | 105.8ms | 0 |
| liteio_mem | 143.71 MB/s | 101.2ms | 113.8ms | 113.8ms | 0 |
| liteio | 143.16 MB/s | 100.8ms | 111.3ms | 111.3ms | 0 |
| seaweedfs | 124.55 MB/s | 116.9ms | 130.5ms | 130.5ms | 0 |
| localstack | 119.16 MB/s | 123.1ms | 140.1ms | 140.1ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 175.76 MB/s
minio        ███████████████████████████ 162.88 MB/s
liteio_mem   ████████████████████████ 143.71 MB/s
liteio       ████████████████████████ 143.16 MB/s
seaweedfs    █████████████████████ 124.55 MB/s
localstack   ████████████████████ 119.16 MB/s
```

**Latency (P50)**
```
rustfs       ████████████████████ 85.4ms
minio        █████████████████████ 89.5ms
liteio_mem   ████████████████████████ 101.2ms
liteio       ████████████████████████ 100.8ms
seaweedfs    ████████████████████████████ 116.9ms
localstack   ██████████████████████████████ 123.1ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 2.83 MB/s | 344.7us | 427.7us | 303.8us | 427.8us | 1.6ms | 0 |
| minio | 2.78 MB/s | 350.6us | 405.3us | 345.3us | 405.7us | 423.4us | 0 |
| liteio_mem | 2.57 MB/s | 380.2us | 472.3us | 334.6us | 472.4us | 1.5ms | 0 |
| seaweedfs | 2.18 MB/s | 447.8us | 520.5us | 434.3us | 521.4us | 721.1us | 0 |
| rustfs | 1.85 MB/s | 527.6us | 800.0us | 494.1us | 800.1us | 1.0ms | 0 |
| localstack | 1.27 MB/s | 771.7us | 948.5us | 740.8us | 948.7us | 1.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 2.83 MB/s
minio        █████████████████████████████ 2.78 MB/s
liteio_mem   ███████████████████████████ 2.57 MB/s
seaweedfs    ███████████████████████ 2.18 MB/s
rustfs       ███████████████████ 1.85 MB/s
localstack   █████████████ 1.27 MB/s
```

**Latency (P50)**
```
liteio       ████████████ 303.8us
minio        █████████████ 345.3us
liteio_mem   █████████████ 334.6us
seaweedfs    █████████████████ 434.3us
rustfs       ████████████████████ 494.1us
localstack   ██████████████████████████████ 740.8us
```

### ParallelRead/1KB/C10

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 1.10 MB/s | 885.2us | 1.5ms | 853.5us | 1.5ms | 1.7ms | 0 |
| liteio | 0.87 MB/s | 1.1ms | 2.2ms | 894.5us | 2.2ms | 3.0ms | 0 |
| liteio_mem | 0.80 MB/s | 1.2ms | 3.2ms | 818.5us | 3.2ms | 3.7ms | 0 |
| seaweedfs | 0.76 MB/s | 1.3ms | 1.9ms | 1.2ms | 1.9ms | 2.1ms | 0 |
| rustfs | 0.76 MB/s | 1.3ms | 1.8ms | 1.2ms | 1.8ms | 2.1ms | 0 |
| localstack | 0.16 MB/s | 6.1ms | 9.6ms | 5.8ms | 9.6ms | 11.7ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 1.10 MB/s
liteio       ███████████████████████ 0.87 MB/s
liteio_mem   █████████████████████ 0.80 MB/s
seaweedfs    ████████████████████ 0.76 MB/s
rustfs       ████████████████████ 0.76 MB/s
localstack   ████ 0.16 MB/s
```

**Latency (P50)**
```
minio        ████ 853.5us
liteio       ████ 894.5us
liteio_mem   ████ 818.5us
seaweedfs    ██████ 1.2ms
rustfs       ██████ 1.2ms
localstack   ██████████████████████████████ 5.8ms
```

### ParallelRead/1KB/C100

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio_mem | 0.34 MB/s | 2.8ms | 3.5ms | 2.8ms | 3.5ms | 3.8ms | 0 |
| liteio | 0.29 MB/s | 3.4ms | 4.8ms | 3.3ms | 4.8ms | 5.0ms | 0 |
| minio | 0.21 MB/s | 4.6ms | 5.8ms | 4.5ms | 5.8ms | 6.0ms | 0 |
| seaweedfs | 0.17 MB/s | 5.6ms | 7.0ms | 5.7ms | 7.0ms | 7.2ms | 0 |
| localstack | 0.02 MB/s | 48.1ms | 65.0ms | 35.1ms | 65.0ms | 66.2ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 0.34 MB/s
liteio       █████████████████████████ 0.29 MB/s
minio        ██████████████████ 0.21 MB/s
seaweedfs    ███████████████ 0.17 MB/s
localstack   █ 0.02 MB/s
```

**Latency (P50)**
```
liteio_mem   ██ 2.8ms
liteio       ██ 3.3ms
minio        ███ 4.5ms
seaweedfs    ████ 5.7ms
localstack   ██████████████████████████████ 35.1ms
```

### ParallelRead/1KB/C200

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio_mem | 0.57 MB/s | 1.7ms | 2.8ms | 1.6ms | 2.8ms | 3.3ms | 0 |
| liteio | 0.30 MB/s | 3.2ms | 4.6ms | 3.9ms | 4.6ms | 5.3ms | 0 |
| seaweedfs | 0.20 MB/s | 4.9ms | 6.0ms | 5.0ms | 6.0ms | 6.2ms | 0 |
| minio | 0.17 MB/s | 5.9ms | 7.5ms | 5.9ms | 7.5ms | 7.6ms | 0 |
| localstack | 0.02 MB/s | 61.7ms | 65.8ms | 64.1ms | 65.8ms | 67.4ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 0.57 MB/s
liteio       ███████████████ 0.30 MB/s
seaweedfs    ██████████ 0.20 MB/s
minio        ████████ 0.17 MB/s
localstack   █ 0.02 MB/s
```

**Latency (P50)**
```
liteio_mem   █ 1.6ms
liteio       █ 3.9ms
seaweedfs    ██ 5.0ms
minio        ██ 5.9ms
localstack   ██████████████████████████████ 64.1ms
```

### ParallelRead/1KB/C25

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 0.64 MB/s | 1.5ms | 2.2ms | 1.5ms | 2.2ms | 2.3ms | 0 |
| liteio_mem | 0.61 MB/s | 1.6ms | 2.4ms | 1.7ms | 2.4ms | 2.8ms | 0 |
| liteio | 0.61 MB/s | 1.6ms | 2.2ms | 1.7ms | 2.2ms | 2.3ms | 0 |
| seaweedfs | 0.38 MB/s | 2.5ms | 4.0ms | 2.4ms | 4.0ms | 4.2ms | 0 |
| localstack | 0.04 MB/s | 22.1ms | 37.1ms | 18.9ms | 37.1ms | 47.4ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 0.64 MB/s
liteio_mem   ████████████████████████████ 0.61 MB/s
liteio       ████████████████████████████ 0.61 MB/s
seaweedfs    █████████████████ 0.38 MB/s
localstack   ██ 0.04 MB/s
```

**Latency (P50)**
```
minio        ██ 1.5ms
liteio_mem   ██ 1.7ms
liteio       ██ 1.7ms
seaweedfs    ███ 2.4ms
localstack   ██████████████████████████████ 18.9ms
```

### ParallelRead/1KB/C50

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio_mem | 0.50 MB/s | 1.9ms | 3.0ms | 1.9ms | 3.0ms | 3.5ms | 0 |
| liteio | 0.46 MB/s | 2.1ms | 3.4ms | 2.0ms | 3.4ms | 3.7ms | 0 |
| minio | 0.35 MB/s | 2.8ms | 3.5ms | 2.7ms | 3.5ms | 3.6ms | 0 |
| seaweedfs | 0.18 MB/s | 5.5ms | 14.4ms | 3.9ms | 14.4ms | 18.3ms | 0 |
| localstack | 0.03 MB/s | 33.3ms | 44.3ms | 28.1ms | 44.3ms | 44.5ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 0.50 MB/s
liteio       ███████████████████████████ 0.46 MB/s
minio        █████████████████████ 0.35 MB/s
seaweedfs    ██████████ 0.18 MB/s
localstack   █ 0.03 MB/s
```

**Latency (P50)**
```
liteio_mem   ██ 1.9ms
liteio       ██ 2.0ms
minio        ██ 2.7ms
seaweedfs    ████ 3.9ms
localstack   ██████████████████████████████ 28.1ms
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1.48 MB/s | 606.8us | 892.9us | 1.3ms | 0 |
| rustfs | 1.36 MB/s | 698.8us | 922.7us | 1.1ms | 0 |
| liteio | 1.33 MB/s | 658.9us | 1.2ms | 1.7ms | 0 |
| localstack | 1.26 MB/s | 745.4us | 993.2us | 1.2ms | 0 |
| minio | 0.96 MB/s | 832.9us | 1.7ms | 2.2ms | 0 |
| seaweedfs | 0.86 MB/s | 784.0us | 2.1ms | 3.7ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 1.48 MB/s
rustfs       ███████████████████████████ 1.36 MB/s
liteio       ██████████████████████████ 1.33 MB/s
localstack   █████████████████████████ 1.26 MB/s
minio        ███████████████████ 0.96 MB/s
seaweedfs    █████████████████ 0.86 MB/s
```

**Latency (P50)**
```
liteio_mem   █████████████████████ 606.8us
rustfs       █████████████████████████ 698.8us
liteio       ███████████████████████ 658.9us
localstack   ██████████████████████████ 745.4us
minio        ██████████████████████████████ 832.9us
seaweedfs    ████████████████████████████ 784.0us
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 0.48 MB/s | 1.8ms | 4.5ms | 4.7ms | 0 |
| seaweedfs | 0.39 MB/s | 2.1ms | 4.3ms | 4.9ms | 0 |
| liteio | 0.39 MB/s | 2.3ms | 4.1ms | 5.1ms | 0 |
| liteio_mem | 0.35 MB/s | 2.8ms | 4.6ms | 4.8ms | 0 |
| minio | 0.28 MB/s | 3.1ms | 5.8ms | 6.4ms | 0 |
| localstack | 0.16 MB/s | 5.5ms | 9.8ms | 12.3ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 0.48 MB/s
seaweedfs    ████████████████████████ 0.39 MB/s
liteio       ████████████████████████ 0.39 MB/s
liteio_mem   █████████████████████ 0.35 MB/s
minio        █████████████████ 0.28 MB/s
localstack   ██████████ 0.16 MB/s
```

**Latency (P50)**
```
rustfs       █████████ 1.8ms
seaweedfs    ███████████ 2.1ms
liteio       ████████████ 2.3ms
liteio_mem   ███████████████ 2.8ms
minio        ████████████████ 3.1ms
localstack   ██████████████████████████████ 5.5ms
```

### ParallelWrite/1KB/C100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 0.11 MB/s | 7.8ms | 11.9ms | 12.2ms | 0 |
| liteio_mem | 0.09 MB/s | 10.2ms | 14.5ms | 15.2ms | 0 |
| liteio | 0.07 MB/s | 14.1ms | 21.3ms | 21.8ms | 0 |
| minio | 0.06 MB/s | 15.0ms | 24.9ms | 25.3ms | 0 |
| localstack | 0.02 MB/s | 44.9ms | 70.8ms | 71.6ms | 0 |

**Throughput**
```
seaweedfs    ██████████████████████████████ 0.11 MB/s
liteio_mem   ████████████████████████ 0.09 MB/s
liteio       █████████████████ 0.07 MB/s
minio        ████████████████ 0.06 MB/s
localstack   ████ 0.02 MB/s
```

**Latency (P50)**
```
seaweedfs    █████ 7.8ms
liteio_mem   ██████ 10.2ms
liteio       █████████ 14.1ms
minio        ██████████ 15.0ms
localstack   ██████████████████████████████ 44.9ms
```

### ParallelWrite/1KB/C200

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 0.13 MB/s | 8.0ms | 9.2ms | 9.8ms | 0 |
| liteio | 0.10 MB/s | 9.9ms | 15.2ms | 15.7ms | 0 |
| liteio_mem | 0.09 MB/s | 11.2ms | 17.1ms | 17.4ms | 0 |
| minio | 0.07 MB/s | 15.7ms | 21.8ms | 23.3ms | 0 |
| localstack | 0.01 MB/s | 74.5ms | 79.9ms | 80.3ms | 0 |

**Throughput**
```
seaweedfs    ██████████████████████████████ 0.13 MB/s
liteio       ███████████████████████ 0.10 MB/s
liteio_mem   ████████████████████ 0.09 MB/s
minio        ███████████████ 0.07 MB/s
localstack   ███ 0.01 MB/s
```

**Latency (P50)**
```
seaweedfs    ███ 8.0ms
liteio       ███ 9.9ms
liteio_mem   ████ 11.2ms
minio        ██████ 15.7ms
localstack   ██████████████████████████████ 74.5ms
```

### ParallelWrite/1KB/C25

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.25 MB/s | 3.7ms | 6.4ms | 7.5ms | 0 |
| liteio_mem | 0.23 MB/s | 4.4ms | 6.8ms | 7.1ms | 0 |
| seaweedfs | 0.22 MB/s | 4.1ms | 6.7ms | 7.4ms | 0 |
| minio | 0.18 MB/s | 5.3ms | 7.8ms | 9.8ms | 0 |
| localstack | 0.02 MB/s | 39.1ms | 113.1ms | 121.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.25 MB/s
liteio_mem   ███████████████████████████ 0.23 MB/s
seaweedfs    ██████████████████████████ 0.22 MB/s
minio        ██████████████████████ 0.18 MB/s
localstack   ██ 0.02 MB/s
```

**Latency (P50)**
```
liteio       ██ 3.7ms
liteio_mem   ███ 4.4ms
seaweedfs    ███ 4.1ms
minio        ████ 5.3ms
localstack   ██████████████████████████████ 39.1ms
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 0.11 MB/s | 7.9ms | 15.5ms | 16.2ms | 0 |
| minio | 0.10 MB/s | 8.5ms | 19.4ms | 21.1ms | 0 |
| liteio_mem | 0.08 MB/s | 9.6ms | 24.0ms | 25.3ms | 0 |
| liteio | 0.08 MB/s | 10.0ms | 23.1ms | 24.1ms | 0 |
| localstack | 0.02 MB/s | 36.7ms | 94.1ms | 94.6ms | 0 |

**Throughput**
```
seaweedfs    ██████████████████████████████ 0.11 MB/s
minio        ██████████████████████████ 0.10 MB/s
liteio_mem   ██████████████████████ 0.08 MB/s
liteio       ██████████████████████ 0.08 MB/s
localstack   █████ 0.02 MB/s
```

**Latency (P50)**
```
seaweedfs    ██████ 7.9ms
minio        ██████ 8.5ms
liteio_mem   ███████ 9.6ms
liteio       ████████ 10.0ms
localstack   ██████████████████████████████ 36.7ms
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 195.14 MB/s | 1.2ms | 1.9ms | 2.2ms | 0 |
| minio | 184.04 MB/s | 1.3ms | 1.6ms | 1.7ms | 0 |
| seaweedfs | 174.75 MB/s | 1.4ms | 1.8ms | 2.0ms | 0 |
| liteio | 164.53 MB/s | 1.4ms | 2.0ms | 2.5ms | 0 |
| localstack | 146.42 MB/s | 1.6ms | 2.3ms | 2.4ms | 0 |
| rustfs | 119.17 MB/s | 2.0ms | 2.8ms | 3.5ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 195.14 MB/s
minio        ████████████████████████████ 184.04 MB/s
seaweedfs    ██████████████████████████ 174.75 MB/s
liteio       █████████████████████████ 164.53 MB/s
localstack   ██████████████████████ 146.42 MB/s
rustfs       ██████████████████ 119.17 MB/s
```

**Latency (P50)**
```
liteio_mem   █████████████████ 1.2ms
minio        ████████████████████ 1.3ms
seaweedfs    ████████████████████ 1.4ms
liteio       █████████████████████ 1.4ms
localstack   ████████████████████████ 1.6ms
rustfs       ██████████████████████████████ 2.0ms
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 200.40 MB/s | 1.2ms | 1.6ms | 1.7ms | 0 |
| seaweedfs | 167.13 MB/s | 1.4ms | 2.0ms | 2.4ms | 0 |
| minio | 160.70 MB/s | 1.4ms | 2.0ms | 2.8ms | 0 |
| liteio | 148.10 MB/s | 1.7ms | 2.1ms | 2.4ms | 0 |
| localstack | 122.21 MB/s | 1.7ms | 4.5ms | 5.1ms | 0 |
| rustfs | 121.57 MB/s | 2.0ms | 2.5ms | 3.4ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 200.40 MB/s
seaweedfs    █████████████████████████ 167.13 MB/s
minio        ████████████████████████ 160.70 MB/s
liteio       ██████████████████████ 148.10 MB/s
localstack   ██████████████████ 122.21 MB/s
rustfs       ██████████████████ 121.57 MB/s
```

**Latency (P50)**
```
liteio_mem   ██████████████████ 1.2ms
seaweedfs    ██████████████████████ 1.4ms
minio        ██████████████████████ 1.4ms
liteio       █████████████████████████ 1.7ms
localstack   ██████████████████████████ 1.7ms
rustfs       ██████████████████████████████ 2.0ms
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 197.94 MB/s | 1.2ms | 1.8ms | 1.9ms | 0 |
| liteio_mem | 190.85 MB/s | 1.3ms | 1.6ms | 1.8ms | 0 |
| seaweedfs | 165.41 MB/s | 1.4ms | 2.1ms | 3.9ms | 0 |
| minio | 152.20 MB/s | 1.6ms | 2.0ms | 2.4ms | 0 |
| rustfs | 116.84 MB/s | 1.9ms | 3.3ms | 4.0ms | 0 |
| localstack | 100.38 MB/s | 1.6ms | 2.7ms | 2.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 197.94 MB/s
liteio_mem   ████████████████████████████ 190.85 MB/s
seaweedfs    █████████████████████████ 165.41 MB/s
minio        ███████████████████████ 152.20 MB/s
rustfs       █████████████████ 116.84 MB/s
localstack   ███████████████ 100.38 MB/s
```

**Latency (P50)**
```
liteio       ██████████████████ 1.2ms
liteio_mem   ███████████████████ 1.3ms
seaweedfs    █████████████████████ 1.4ms
minio        ████████████████████████ 1.6ms
rustfs       ██████████████████████████████ 1.9ms
localstack   ████████████████████████ 1.6ms
```

### Read/100MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 324.26 MB/s | 869.9us | 931.5us | 308.6ms | 309.3ms | 309.3ms | 0 |
| rustfs | 297.45 MB/s | 2.6ms | 2.8ms | 328.6ms | 344.3ms | 344.3ms | 0 |
| localstack | 262.11 MB/s | 2.2ms | 2.1ms | 382.4ms | 388.5ms | 388.5ms | 0 |
| seaweedfs | 253.96 MB/s | 2.6ms | 3.1ms | 393.8ms | 394.6ms | 394.6ms | 0 |
| liteio | 252.18 MB/s | 959.5us | 1.1ms | 394.9ms | 404.2ms | 404.2ms | 0 |
| liteio_mem | 249.65 MB/s | 562.7us | 587.5us | 398.0ms | 399.2ms | 399.2ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 324.26 MB/s
rustfs       ███████████████████████████ 297.45 MB/s
localstack   ████████████████████████ 262.11 MB/s
seaweedfs    ███████████████████████ 253.96 MB/s
liteio       ███████████████████████ 252.18 MB/s
liteio_mem   ███████████████████████ 249.65 MB/s
```

**Latency (P50)**
```
minio        ███████████████████████ 308.6ms
rustfs       ████████████████████████ 328.6ms
localstack   ████████████████████████████ 382.4ms
seaweedfs    █████████████████████████████ 393.8ms
liteio       █████████████████████████████ 394.9ms
liteio_mem   ██████████████████████████████ 398.0ms
```

### Read/10MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 298.28 MB/s | 970.2us | 1.1ms | 31.1ms | 38.6ms | 38.6ms | 0 |
| localstack | 295.77 MB/s | 1.6ms | 1.8ms | 32.9ms | 35.7ms | 35.7ms | 0 |
| liteio_mem | 258.99 MB/s | 570.6us | 585.1us | 37.6ms | 42.3ms | 42.3ms | 0 |
| rustfs | 257.87 MB/s | 5.5ms | 8.8ms | 35.5ms | 40.8ms | 40.8ms | 0 |
| seaweedfs | 252.46 MB/s | 2.4ms | 2.8ms | 40.0ms | 41.9ms | 41.9ms | 0 |
| liteio | 238.06 MB/s | 677.1us | 1.1ms | 40.6ms | 49.6ms | 49.6ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 298.28 MB/s
localstack   █████████████████████████████ 295.77 MB/s
liteio_mem   ██████████████████████████ 258.99 MB/s
rustfs       █████████████████████████ 257.87 MB/s
seaweedfs    █████████████████████████ 252.46 MB/s
liteio       ███████████████████████ 238.06 MB/s
```

**Latency (P50)**
```
minio        ██████████████████████ 31.1ms
localstack   ████████████████████████ 32.9ms
liteio_mem   ███████████████████████████ 37.6ms
rustfs       ██████████████████████████ 35.5ms
seaweedfs    █████████████████████████████ 40.0ms
liteio       ██████████████████████████████ 40.6ms
```

### Read/1KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 4.81 MB/s | 202.9us | 248.5us | 201.4us | 248.5us | 253.8us | 0 |
| liteio_mem | 4.49 MB/s | 216.8us | 265.2us | 203.8us | 265.3us | 498.1us | 0 |
| minio | 2.87 MB/s | 340.0us | 393.5us | 334.2us | 393.5us | 477.4us | 0 |
| seaweedfs | 2.58 MB/s | 379.0us | 451.1us | 372.6us | 451.5us | 555.0us | 0 |
| rustfs | 2.24 MB/s | 435.4us | 484.7us | 438.4us | 484.8us | 556.2us | 0 |
| localstack | 1.40 MB/s | 697.2us | 804.8us | 669.5us | 805.0us | 1.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4.81 MB/s
liteio_mem   ████████████████████████████ 4.49 MB/s
minio        █████████████████ 2.87 MB/s
seaweedfs    ████████████████ 2.58 MB/s
rustfs       █████████████ 2.24 MB/s
localstack   ████████ 1.40 MB/s
```

**Latency (P50)**
```
liteio       █████████ 201.4us
liteio_mem   █████████ 203.8us
minio        ██████████████ 334.2us
seaweedfs    ████████████████ 372.6us
rustfs       ███████████████████ 438.4us
localstack   ██████████████████████████████ 669.5us
```

### Read/1MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio_mem | 271.70 MB/s | 299.9us | 380.7us | 3.7ms | 3.8ms | 3.8ms | 0 |
| liteio | 242.17 MB/s | 422.3us | 705.9us | 3.7ms | 5.6ms | 5.6ms | 0 |
| minio | 233.19 MB/s | 1.3ms | 1.5ms | 4.0ms | 4.3ms | 4.3ms | 0 |
| localstack | 204.44 MB/s | 1.2ms | 1.6ms | 4.7ms | 6.2ms | 6.2ms | 0 |
| seaweedfs | 192.51 MB/s | 1.5ms | 3.5ms | 4.0ms | 13.0ms | 13.0ms | 0 |
| rustfs | 191.55 MB/s | 2.3ms | 3.8ms | 4.8ms | 6.9ms | 6.9ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 271.70 MB/s
liteio       ██████████████████████████ 242.17 MB/s
minio        █████████████████████████ 233.19 MB/s
localstack   ██████████████████████ 204.44 MB/s
seaweedfs    █████████████████████ 192.51 MB/s
rustfs       █████████████████████ 191.55 MB/s
```

**Latency (P50)**
```
liteio_mem   ███████████████████████ 3.7ms
liteio       ██████████████████████ 3.7ms
minio        ████████████████████████ 4.0ms
localstack   ████████████████████████████ 4.7ms
seaweedfs    ████████████████████████ 4.0ms
rustfs       ██████████████████████████████ 4.8ms
```

### Read/64KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 119.22 MB/s | 282.9us | 434.1us | 511.5us | 654.2us | 812.5us | 0 |
| liteio_mem | 108.25 MB/s | 343.4us | 568.6us | 511.7us | 815.0us | 1.5ms | 0 |
| seaweedfs | 100.32 MB/s | 462.4us | 598.7us | 610.0us | 708.8us | 726.5us | 0 |
| minio | 95.77 MB/s | 462.0us | 630.3us | 610.5us | 776.9us | 1.3ms | 0 |
| rustfs | 86.08 MB/s | 624.1us | 707.5us | 694.5us | 827.5us | 1.0ms | 0 |
| localstack | 70.31 MB/s | 782.5us | 852.8us | 870.7us | 950.0us | 981.2us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 119.22 MB/s
liteio_mem   ███████████████████████████ 108.25 MB/s
seaweedfs    █████████████████████████ 100.32 MB/s
minio        ████████████████████████ 95.77 MB/s
rustfs       █████████████████████ 86.08 MB/s
localstack   █████████████████ 70.31 MB/s
```

**Latency (P50)**
```
liteio       █████████████████ 511.5us
liteio_mem   █████████████████ 511.7us
seaweedfs    █████████████████████ 610.0us
minio        █████████████████████ 610.5us
rustfs       ███████████████████████ 694.5us
localstack   ██████████████████████████████ 870.7us
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 4399 ops/s | 222.2us | 285.3us | 326.8us | 0 |
| liteio | 4316 ops/s | 221.2us | 331.8us | 369.6us | 0 |
| liteio_mem | 4076 ops/s | 238.6us | 318.0us | 499.9us | 0 |
| rustfs | 3266 ops/s | 293.8us | 371.6us | 432.5us | 0 |
| seaweedfs | 2997 ops/s | 295.8us | 423.3us | 633.1us | 0 |
| localstack | 1464 ops/s | 647.8us | 911.0us | 1.2ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 4399 ops/s
liteio       █████████████████████████████ 4316 ops/s
liteio_mem   ███████████████████████████ 4076 ops/s
rustfs       ██████████████████████ 3266 ops/s
seaweedfs    ████████████████████ 2997 ops/s
localstack   █████████ 1464 ops/s
```

**Latency (P50)**
```
minio        ██████████ 222.2us
liteio       ██████████ 221.2us
liteio_mem   ███████████ 238.6us
rustfs       █████████████ 293.8us
seaweedfs    █████████████ 295.8us
localstack   ██████████████████████████████ 647.8us
```

### Write/100MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 193.20 MB/s | 518.6ms | 521.1ms | 521.1ms | 0 |
| seaweedfs | 190.92 MB/s | 528.5ms | 538.3ms | 538.3ms | 0 |
| liteio_mem | 178.73 MB/s | 561.2ms | 569.4ms | 569.4ms | 0 |
| minio | 170.85 MB/s | 561.7ms | 571.8ms | 571.8ms | 0 |
| rustfs | 165.85 MB/s | 603.1ms | 604.9ms | 604.9ms | 0 |
| localstack | 142.85 MB/s | 670.7ms | 732.1ms | 732.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 193.20 MB/s
seaweedfs    █████████████████████████████ 190.92 MB/s
liteio_mem   ███████████████████████████ 178.73 MB/s
minio        ██████████████████████████ 170.85 MB/s
rustfs       █████████████████████████ 165.85 MB/s
localstack   ██████████████████████ 142.85 MB/s
```

**Latency (P50)**
```
liteio       ███████████████████████ 518.6ms
seaweedfs    ███████████████████████ 528.5ms
liteio_mem   █████████████████████████ 561.2ms
minio        █████████████████████████ 561.7ms
rustfs       ██████████████████████████ 603.1ms
localstack   ██████████████████████████████ 670.7ms
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 189.15 MB/s | 51.7ms | 54.7ms | 54.7ms | 0 |
| minio | 183.38 MB/s | 53.5ms | 58.1ms | 58.1ms | 0 |
| rustfs | 174.89 MB/s | 57.1ms | 61.0ms | 61.0ms | 0 |
| liteio_mem | 164.64 MB/s | 56.3ms | 69.3ms | 69.3ms | 0 |
| seaweedfs | 159.91 MB/s | 60.9ms | 67.7ms | 67.7ms | 0 |
| localstack | 146.83 MB/s | 66.9ms | 72.5ms | 72.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 189.15 MB/s
minio        █████████████████████████████ 183.38 MB/s
rustfs       ███████████████████████████ 174.89 MB/s
liteio_mem   ██████████████████████████ 164.64 MB/s
seaweedfs    █████████████████████████ 159.91 MB/s
localstack   ███████████████████████ 146.83 MB/s
```

**Latency (P50)**
```
liteio       ███████████████████████ 51.7ms
minio        ████████████████████████ 53.5ms
rustfs       █████████████████████████ 57.1ms
liteio_mem   █████████████████████████ 56.3ms
seaweedfs    ███████████████████████████ 60.9ms
localstack   ██████████████████████████████ 66.9ms
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1.54 MB/s | 538.4us | 1.1ms | 1.2ms | 0 |
| rustfs | 1.49 MB/s | 646.9us | 857.4us | 1.0ms | 0 |
| seaweedfs | 1.45 MB/s | 663.4us | 835.4us | 972.2us | 0 |
| liteio | 1.21 MB/s | 728.2us | 1.2ms | 1.9ms | 0 |
| localstack | 1.13 MB/s | 727.2us | 1.4ms | 1.8ms | 0 |
| minio | 0.95 MB/s | 823.0us | 1.7ms | 2.2ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 1.54 MB/s
rustfs       ████████████████████████████ 1.49 MB/s
seaweedfs    ████████████████████████████ 1.45 MB/s
liteio       ███████████████████████ 1.21 MB/s
localstack   ██████████████████████ 1.13 MB/s
minio        ██████████████████ 0.95 MB/s
```

**Latency (P50)**
```
liteio_mem   ███████████████████ 538.4us
rustfs       ███████████████████████ 646.9us
seaweedfs    ████████████████████████ 663.4us
liteio       ██████████████████████████ 728.2us
localstack   ██████████████████████████ 727.2us
minio        ██████████████████████████████ 823.0us
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 166.52 MB/s | 5.5ms | 7.9ms | 7.9ms | 0 |
| liteio | 154.34 MB/s | 6.3ms | 7.0ms | 7.0ms | 0 |
| liteio_mem | 149.23 MB/s | 6.1ms | 9.2ms | 9.2ms | 0 |
| localstack | 129.38 MB/s | 7.5ms | 8.5ms | 8.5ms | 0 |
| minio | 125.96 MB/s | 7.7ms | 9.7ms | 9.7ms | 0 |
| seaweedfs | 122.95 MB/s | 7.9ms | 9.3ms | 9.3ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 166.52 MB/s
liteio       ███████████████████████████ 154.34 MB/s
liteio_mem   ██████████████████████████ 149.23 MB/s
localstack   ███████████████████████ 129.38 MB/s
minio        ██████████████████████ 125.96 MB/s
seaweedfs    ██████████████████████ 122.95 MB/s
```

**Latency (P50)**
```
rustfs       ████████████████████ 5.5ms
liteio       ███████████████████████ 6.3ms
liteio_mem   ███████████████████████ 6.1ms
localstack   ████████████████████████████ 7.5ms
minio        █████████████████████████████ 7.7ms
seaweedfs    ██████████████████████████████ 7.9ms
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 60.19 MB/s | 990.4us | 1.4ms | 1.8ms | 0 |
| liteio | 56.43 MB/s | 1.0ms | 1.5ms | 1.8ms | 0 |
| liteio_mem | 54.35 MB/s | 987.3us | 2.1ms | 2.5ms | 0 |
| seaweedfs | 53.42 MB/s | 1.1ms | 1.4ms | 1.4ms | 0 |
| minio | 47.41 MB/s | 1.2ms | 1.9ms | 2.0ms | 0 |
| localstack | 37.96 MB/s | 1.1ms | 2.5ms | 3.7ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 60.19 MB/s
liteio       ████████████████████████████ 56.43 MB/s
liteio_mem   ███████████████████████████ 54.35 MB/s
seaweedfs    ██████████████████████████ 53.42 MB/s
minio        ███████████████████████ 47.41 MB/s
localstack   ██████████████████ 37.96 MB/s
```

**Latency (P50)**
```
rustfs       ████████████████████████ 990.4us
liteio       █████████████████████████ 1.0ms
liteio_mem   ████████████████████████ 987.3us
seaweedfs    ███████████████████████████ 1.1ms
minio        ██████████████████████████████ 1.2ms
localstack   ████████████████████████████ 1.1ms
```

## Resource Usage

| Driver | Memory | RSS | Cache | CPU | Volume | Block I/O |
|--------|--------|-----|-------|-----|--------|----------|
| liteio | 98.01MiB / 7.653GiB | 98.0 MB | - | 0.0% | 1580.0 MB | 8.19kB / 2.23GB |
| liteio_mem | 70.83MiB / 7.653GiB | 70.8 MB | - | 1.1% | 1580.0 MB | 1.11MB / 2.23GB |
| localstack | 388.8MiB / 7.653GiB | 388.8 MB | - | 0.0% | 0.0 MB | 15.9MB / 1.7GB |
| minio | 396.1MiB / 7.653GiB | 396.1 MB | - | 0.0% | 1924.1 MB | 1.17MB / 2GB |
| rustfs | 360.6MiB / 7.653GiB | 360.6 MB | - | 0.1% | 1923.1 MB | 2.95MB / 1.87GB |
| seaweedfs | 126.7MiB / 7.653GiB | 126.7 MB | - | 0.0% | (no data) | 1.04MB / 0B |

> **Note:** RSS = actual application memory. Cache = OS page cache (reclaimable).

## Recommendations

- **Write-heavy workloads:** liteio
- **Read-heavy workloads:** minio

---

*Generated by storage benchmark CLI*
