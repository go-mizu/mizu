# Storage Benchmark Report

**Generated:** 2026-01-15T11:42:25+07:00

**Go Version:** go1.25.5

**Platform:** darwin/arm64

## Executive Summary

### Summary

**Overall Winner:** liteio_mem (won 22/51 benchmarks, 43%)

| Rank | Driver | Wins | Win Rate |
|------|--------|------|----------|
| 1 | liteio_mem | 22 | 43% |
| 2 | seaweedfs | 12 | 24% |
| 3 | liteio | 9 | 18% |
| 4 | rustfs | 5 | 10% |
| 5 | minio | 3 | 6% |

### Performance Leaders

| Operation | Leader | Performance | Margin |
|-----------|--------|-------------|--------|
| Small Read (1KB) | liteio | 5.0 MB/s | +18% vs liteio_mem |
| Small Write (1KB) | rustfs | 1.5 MB/s | close |
| Large Read (10MB) | minio | 316.9 MB/s | close |
| Large Write (10MB) | rustfs | 194.9 MB/s | +28% vs minio |
| Delete | liteio | 6.3K ops/s | close |
| Stat | liteio_mem | 4.2K ops/s | close |
| List (100 objects) | liteio | 1.3K ops/s | close |
| Copy | liteio_mem | 1.6 MB/s | close |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (100MB+) | **seaweedfs** | 195 MB/s | Best for media, backups |
| Large File Downloads (100MB) | **minio** | 331 MB/s | Best for streaming, CDN |
| Small File Operations | **liteio** | 3075 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **minio** | - | Best for multi-user apps |
| Memory Constrained | **liteio_mem** | 68 MB RAM | Best for edge/embedded |

### Large File Performance (100MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| liteio | 128.1 | 269.3 | 619.6ms | 382.2ms |
| liteio_mem | 87.0 | 281.5 | 784.7ms | 345.7ms |
| localstack | 137.3 | 316.3 | 666.8ms | 317.5ms |
| minio | 167.3 | 330.7 | 626.8ms | 301.3ms |
| rustfs | 182.3 | 316.2 | 590.0ms | 315.0ms |
| seaweedfs | 194.6 | 300.0 | 514.7ms | 336.9ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| liteio | 988 | 5163 | 944.0us | 173.9us |
| liteio_mem | 1297 | 4367 | 687.2us | 205.0us |
| localstack | 1180 | 1331 | 761.8us | 732.4us |
| minio | 1390 | 2684 | 660.1us | 370.5us |
| rustfs | 1487 | 1852 | 660.7us | 526.7us |
| seaweedfs | 1446 | 2227 | 676.5us | 432.9us |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| liteio | 4076 | 1317 | 6312 |
| liteio_mem | 4180 | 1224 | 5830 |
| localstack | 1609 | 329 | 1602 |
| minio | 4061 | 663 | 2298 |
| rustfs | 2537 | 133 | 1095 |
| seaweedfs | 1749 | 498 | 2647 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| liteio | 1.44 | 0.42 | 0.25 | 0.12 | 0.06 | 0.09 |
| liteio_mem | 1.52 | 0.43 | 0.23 | 0.12 | 0.07 | 0.09 |
| localstack | 1.03 | 0.19 | 0.08 | 0.03 | 0.03 | 0.02 |
| minio | 0.96 | 0.39 | 0.19 | 0.12 | 0.07 | 0.07 |
| rustfs | 1.00 | 0.28 | 0.16 | 0.08 | 0.16 | 0.10 |
| seaweedfs | 0.95 | 0.44 | 0.28 | 0.20 | 0.10 | 0.12 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| liteio | 4.02 | 1.08 | 0.83 | 0.40 | 0.35 | 0.27 |
| liteio_mem | 3.79 | 0.97 | 0.79 | 0.49 | 0.32 | 0.26 |
| localstack | 1.09 | 0.19 | 0.08 | 0.04 | 0.02 | 0.02 |
| minio | 2.79 | 1.23 | 0.65 | 0.36 | 0.24 | 0.20 |
| rustfs | 1.44 | 0.78 | 0.55 | 0.39 | 0.38 | 0.31 |
| seaweedfs | 1.69 | 0.91 | 0.56 | 0.48 | 0.46 | 0.32 |

*\* indicates errors occurred*

### File Count Performance

Performance with varying numbers of files (1KB each).

**Write N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|------|
| liteio | 1.8ms | 8.9ms | 166.7ms | 1.27s | 7.44s |
| liteio_mem | 628.2us | 5.8ms | 66.1ms | 635.6ms | 5.30s |
| localstack | 808.2us | 7.6ms | 82.8ms | 793.5ms | 8.15s |
| minio | 849.6us | 9.0ms | 92.3ms | 866.9ms | 10.72s |
| rustfs | 698.8us | 6.8ms | 68.0ms | 694.4ms | 6.99s |
| seaweedfs | 880.2us | 7.5ms | 70.3ms | 738.5ms | 7.80s |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|------|
| liteio | 539.2us | 520.2us | 932.0us | 6.0ms | 205.5ms |
| liteio_mem | 248.0us | 323.1us | 814.7us | 8.6ms | 201.7ms |
| localstack | 1.1ms | 1.4ms | 4.9ms | 29.5ms | 212.5ms |
| minio | 576.6us | 730.9us | 3.0ms | 19.8ms | 191.1ms |
| rustfs | 1.1ms | 1.5ms | 8.1ms | 60.0ms | 754.5ms |
| seaweedfs | 717.7us | 853.8us | 2.2ms | 15.1ms | 129.9ms |

*\* indicates errors occurred*

### Resource Usage Summary

| Driver | Memory | CPU |
|--------|--------|-----|
| liteio | 93.5 MB | 0.0% |
| liteio_mem | 68.4 MB | 0.0% |
| localstack | 387.0 MB | 0.1% |
| minio | 406.7 MB | 3.2% |
| rustfs | 701.3 MB | 0.1% |
| seaweedfs | 131.8 MB | 1.0% |

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
- **rustfs** (51 benchmarks)
- **seaweedfs** (51 benchmarks)

*Reference baseline: devnull (excluded from comparisons)*

## Detailed Results

### Copy/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1.61 MB/s | 582.6us | 791.2us | 972.3us | 0 |
| liteio | 1.49 MB/s | 606.7us | 847.8us | 1.2ms | 0 |
| localstack | 1.33 MB/s | 714.9us | 977.4us | 1.2ms | 0 |
| seaweedfs | 0.96 MB/s | 1.0ms | 1.2ms | 1.7ms | 0 |
| minio | 0.80 MB/s | 1.2ms | 1.6ms | 1.9ms | 0 |
| rustfs | 0.57 MB/s | 1.1ms | 4.2ms | 13.0ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 1.61 MB/s
liteio       ███████████████████████████ 1.49 MB/s
localstack   ████████████████████████ 1.33 MB/s
seaweedfs    █████████████████ 0.96 MB/s
minio        ██████████████ 0.80 MB/s
rustfs       ██████████ 0.57 MB/s
```

**Latency (P50)**
```
liteio_mem   ███████████████ 582.6us
liteio       ███████████████ 606.7us
localstack   ██████████████████ 714.9us
seaweedfs    ██████████████████████████ 1.0ms
minio        ██████████████████████████████ 1.2ms
rustfs       ████████████████████████████ 1.1ms
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 6312 ops/s | 153.3us | 203.1us | 228.8us | 0 |
| liteio_mem | 5830 ops/s | 162.1us | 212.2us | 310.1us | 0 |
| seaweedfs | 2647 ops/s | 370.4us | 443.3us | 497.7us | 0 |
| minio | 2298 ops/s | 429.7us | 518.2us | 568.6us | 0 |
| localstack | 1602 ops/s | 599.0us | 741.4us | 930.6us | 0 |
| rustfs | 1095 ops/s | 845.0us | 1.3ms | 1.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 6312 ops/s
liteio_mem   ███████████████████████████ 5830 ops/s
seaweedfs    ████████████ 2647 ops/s
minio        ██████████ 2298 ops/s
localstack   ███████ 1602 ops/s
rustfs       █████ 1095 ops/s
```

**Latency (P50)**
```
liteio       █████ 153.3us
liteio_mem   █████ 162.1us
seaweedfs    █████████████ 370.4us
minio        ███████████████ 429.7us
localstack   █████████████████████ 599.0us
rustfs       ██████████████████████████████ 845.0us
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 0.16 MB/s | 551.0us | 737.5us | 790.0us | 0 |
| rustfs | 0.16 MB/s | 636.2us | 705.4us | 777.9us | 0 |
| seaweedfs | 0.13 MB/s | 732.0us | 800.2us | 1.0ms | 0 |
| localstack | 0.11 MB/s | 798.1us | 1.0ms | 1.3ms | 0 |
| minio | 0.11 MB/s | 850.4us | 1.1ms | 1.2ms | 0 |
| liteio | 0.06 MB/s | 1.5ms | 2.1ms | 2.8ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 0.16 MB/s
rustfs       █████████████████████████████ 0.16 MB/s
seaweedfs    ███████████████████████ 0.13 MB/s
localstack   ████████████████████ 0.11 MB/s
minio        ███████████████████ 0.11 MB/s
liteio       ███████████ 0.06 MB/s
```

**Latency (P50)**
```
liteio_mem   ███████████ 551.0us
rustfs       ████████████ 636.2us
seaweedfs    ██████████████ 732.0us
localstack   ████████████████ 798.1us
minio        █████████████████ 850.4us
liteio       ██████████████████████████████ 1.5ms
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 2120 ops/s | 431.7us | 639.8us | 811.0us | 0 |
| liteio_mem | 1687 ops/s | 531.8us | 803.4us | 1.2ms | 0 |
| rustfs | 1519 ops/s | 633.1us | 715.2us | 841.1us | 0 |
| localstack | 1165 ops/s | 819.3us | 1.1ms | 1.4ms | 0 |
| minio | 1040 ops/s | 896.5us | 1.3ms | 2.0ms | 0 |
| liteio | 632 ops/s | 1.4ms | 2.0ms | 6.5ms | 0 |

**Throughput**
```
seaweedfs    ██████████████████████████████ 2120 ops/s
liteio_mem   ███████████████████████ 1687 ops/s
rustfs       █████████████████████ 1519 ops/s
localstack   ████████████████ 1165 ops/s
minio        ██████████████ 1040 ops/s
liteio       ████████ 632 ops/s
```

**Latency (P50)**
```
seaweedfs    █████████ 431.7us
liteio_mem   ███████████ 531.8us
rustfs       █████████████ 633.1us
localstack   █████████████████ 819.3us
minio        ██████████████████ 896.5us
liteio       ██████████████████████████████ 1.4ms
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 0.16 MB/s | 581.3us | 695.4us | 927.2us | 0 |
| seaweedfs | 0.12 MB/s | 757.5us | 913.3us | 1.1ms | 0 |
| localstack | 0.12 MB/s | 794.5us | 928.4us | 1.2ms | 0 |
| rustfs | 0.11 MB/s | 793.2us | 1.0ms | 1.4ms | 0 |
| minio | 0.10 MB/s | 930.2us | 1.1ms | 1.1ms | 0 |
| liteio | 0.09 MB/s | 789.4us | 1.8ms | 2.0ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 0.16 MB/s
seaweedfs    ███████████████████████ 0.12 MB/s
localstack   ██████████████████████ 0.12 MB/s
rustfs       █████████████████████ 0.11 MB/s
minio        ███████████████████ 0.10 MB/s
liteio       █████████████████ 0.09 MB/s
```

**Latency (P50)**
```
liteio_mem   ██████████████████ 581.3us
seaweedfs    ████████████████████████ 757.5us
localstack   █████████████████████████ 794.5us
rustfs       █████████████████████████ 793.2us
minio        ██████████████████████████████ 930.2us
liteio       █████████████████████████ 789.4us
```

### FileCount/Delete/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 2344 ops/s | 426.6us | 426.6us | 426.6us | 0 |
| seaweedfs | 2208 ops/s | 452.9us | 452.9us | 452.9us | 0 |
| minio | 2169 ops/s | 461.0us | 461.0us | 461.0us | 0 |
| liteio_mem | 1801 ops/s | 555.4us | 555.4us | 555.4us | 0 |
| localstack | 1239 ops/s | 807.2us | 807.2us | 807.2us | 0 |
| rustfs | 1017 ops/s | 983.0us | 983.0us | 983.0us | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 2344 ops/s
seaweedfs    ████████████████████████████ 2208 ops/s
minio        ███████████████████████████ 2169 ops/s
liteio_mem   ███████████████████████ 1801 ops/s
localstack   ███████████████ 1239 ops/s
rustfs       █████████████ 1017 ops/s
```

**Latency (P50)**
```
liteio       █████████████ 426.6us
seaweedfs    █████████████ 452.9us
minio        ██████████████ 461.0us
liteio_mem   ████████████████ 555.4us
localstack   ████████████████████████ 807.2us
rustfs       ██████████████████████████████ 983.0us
```

### FileCount/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 490 ops/s | 2.0ms | 2.0ms | 2.0ms | 0 |
| liteio | 391 ops/s | 2.6ms | 2.6ms | 2.6ms | 0 |
| minio | 258 ops/s | 3.9ms | 3.9ms | 3.9ms | 0 |
| seaweedfs | 191 ops/s | 5.2ms | 5.2ms | 5.2ms | 0 |
| localstack | 141 ops/s | 7.1ms | 7.1ms | 7.1ms | 0 |
| rustfs | 113 ops/s | 8.8ms | 8.8ms | 8.8ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 490 ops/s
liteio       ███████████████████████ 391 ops/s
minio        ███████████████ 258 ops/s
seaweedfs    ███████████ 191 ops/s
localstack   ████████ 141 ops/s
rustfs       ██████ 113 ops/s
```

**Latency (P50)**
```
liteio_mem   ██████ 2.0ms
liteio       ████████ 2.6ms
minio        █████████████ 3.9ms
seaweedfs    █████████████████ 5.2ms
localstack   ████████████████████████ 7.1ms
rustfs       ██████████████████████████████ 8.8ms
```

### FileCount/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 52 ops/s | 19.3ms | 19.3ms | 19.3ms | 0 |
| liteio | 49 ops/s | 20.5ms | 20.5ms | 20.5ms | 0 |
| seaweedfs | 22 ops/s | 45.6ms | 45.6ms | 45.6ms | 0 |
| minio | 21 ops/s | 46.7ms | 46.7ms | 46.7ms | 0 |
| localstack | 15 ops/s | 64.8ms | 64.8ms | 64.8ms | 0 |
| rustfs | 11 ops/s | 87.8ms | 87.8ms | 87.8ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 52 ops/s
liteio       ████████████████████████████ 49 ops/s
seaweedfs    ████████████ 22 ops/s
minio        ████████████ 21 ops/s
localstack   ████████ 15 ops/s
rustfs       ██████ 11 ops/s
```

**Latency (P50)**
```
liteio_mem   ██████ 19.3ms
liteio       ██████ 20.5ms
seaweedfs    ███████████████ 45.6ms
minio        ███████████████ 46.7ms
localstack   ██████████████████████ 64.8ms
rustfs       ██████████████████████████████ 87.8ms
```

### FileCount/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 5 ops/s | 190.9ms | 190.9ms | 190.9ms | 0 |
| liteio_mem | 5 ops/s | 202.0ms | 202.0ms | 202.0ms | 0 |
| minio | 2 ops/s | 432.8ms | 432.8ms | 432.8ms | 0 |
| seaweedfs | 2 ops/s | 454.2ms | 454.2ms | 454.2ms | 0 |
| localstack | 2 ops/s | 657.7ms | 657.7ms | 657.7ms | 0 |
| rustfs | 1 ops/s | 880.8ms | 880.8ms | 880.8ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 5 ops/s
liteio_mem   ████████████████████████████ 5 ops/s
minio        █████████████ 2 ops/s
seaweedfs    ████████████ 2 ops/s
localstack   ████████ 2 ops/s
rustfs       ██████ 1 ops/s
```

**Latency (P50)**
```
liteio       ██████ 190.9ms
liteio_mem   ██████ 202.0ms
minio        ██████████████ 432.8ms
seaweedfs    ███████████████ 454.2ms
localstack   ██████████████████████ 657.7ms
rustfs       ██████████████████████████████ 880.8ms
```

### FileCount/Delete/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1 ops/s | 1.90s | 1.90s | 1.90s | 0 |
| liteio | 1 ops/s | 1.98s | 1.98s | 1.98s | 0 |
| seaweedfs | 0 ops/s | 3.79s | 3.79s | 3.79s | 0 |
| minio | 0 ops/s | 4.05s | 4.05s | 4.05s | 0 |
| localstack | 0 ops/s | 6.68s | 6.68s | 6.68s | 0 |
| rustfs | 0 ops/s | 8.61s | 8.61s | 8.61s | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 1 ops/s
liteio       ████████████████████████████ 1 ops/s
seaweedfs    ███████████████ 0 ops/s
minio        ██████████████ 0 ops/s
localstack   ████████ 0 ops/s
rustfs       ██████ 0 ops/s
```

**Latency (P50)**
```
liteio_mem   ██████ 1.90s
liteio       ██████ 1.98s
seaweedfs    █████████████ 3.79s
minio        ██████████████ 4.05s
localstack   ███████████████████████ 6.68s
rustfs       ██████████████████████████████ 8.61s
```

### FileCount/List/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 4033 ops/s | 248.0us | 248.0us | 248.0us | 0 |
| liteio | 1855 ops/s | 539.2us | 539.2us | 539.2us | 0 |
| minio | 1734 ops/s | 576.6us | 576.6us | 576.6us | 0 |
| seaweedfs | 1393 ops/s | 717.7us | 717.7us | 717.7us | 0 |
| rustfs | 933 ops/s | 1.1ms | 1.1ms | 1.1ms | 0 |
| localstack | 908 ops/s | 1.1ms | 1.1ms | 1.1ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 4033 ops/s
liteio       █████████████ 1855 ops/s
minio        ████████████ 1734 ops/s
seaweedfs    ██████████ 1393 ops/s
rustfs       ██████ 933 ops/s
localstack   ██████ 908 ops/s
```

**Latency (P50)**
```
liteio_mem   ██████ 248.0us
liteio       ██████████████ 539.2us
minio        ███████████████ 576.6us
seaweedfs    ███████████████████ 717.7us
rustfs       █████████████████████████████ 1.1ms
localstack   ██████████████████████████████ 1.1ms
```

### FileCount/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 3095 ops/s | 323.1us | 323.1us | 323.1us | 0 |
| liteio | 1922 ops/s | 520.2us | 520.2us | 520.2us | 0 |
| minio | 1368 ops/s | 730.9us | 730.9us | 730.9us | 0 |
| seaweedfs | 1171 ops/s | 853.8us | 853.8us | 853.8us | 0 |
| localstack | 732 ops/s | 1.4ms | 1.4ms | 1.4ms | 0 |
| rustfs | 674 ops/s | 1.5ms | 1.5ms | 1.5ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 3095 ops/s
liteio       ██████████████████ 1922 ops/s
minio        █████████████ 1368 ops/s
seaweedfs    ███████████ 1171 ops/s
localstack   ███████ 732 ops/s
rustfs       ██████ 674 ops/s
```

**Latency (P50)**
```
liteio_mem   ██████ 323.1us
liteio       ██████████ 520.2us
minio        ██████████████ 730.9us
seaweedfs    █████████████████ 853.8us
localstack   ███████████████████████████ 1.4ms
rustfs       ██████████████████████████████ 1.5ms
```

### FileCount/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1227 ops/s | 814.7us | 814.7us | 814.7us | 0 |
| liteio | 1073 ops/s | 932.0us | 932.0us | 932.0us | 0 |
| seaweedfs | 464 ops/s | 2.2ms | 2.2ms | 2.2ms | 0 |
| minio | 331 ops/s | 3.0ms | 3.0ms | 3.0ms | 0 |
| localstack | 204 ops/s | 4.9ms | 4.9ms | 4.9ms | 0 |
| rustfs | 123 ops/s | 8.1ms | 8.1ms | 8.1ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 1227 ops/s
liteio       ██████████████████████████ 1073 ops/s
seaweedfs    ███████████ 464 ops/s
minio        ████████ 331 ops/s
localstack   ████ 204 ops/s
rustfs       ███ 123 ops/s
```

**Latency (P50)**
```
liteio_mem   ███ 814.7us
liteio       ███ 932.0us
seaweedfs    ███████ 2.2ms
minio        ███████████ 3.0ms
localstack   ██████████████████ 4.9ms
rustfs       ██████████████████████████████ 8.1ms
```

### FileCount/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 167 ops/s | 6.0ms | 6.0ms | 6.0ms | 0 |
| liteio_mem | 117 ops/s | 8.6ms | 8.6ms | 8.6ms | 0 |
| seaweedfs | 66 ops/s | 15.1ms | 15.1ms | 15.1ms | 0 |
| minio | 50 ops/s | 19.8ms | 19.8ms | 19.8ms | 0 |
| localstack | 34 ops/s | 29.5ms | 29.5ms | 29.5ms | 0 |
| rustfs | 17 ops/s | 60.0ms | 60.0ms | 60.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 167 ops/s
liteio_mem   ████████████████████ 117 ops/s
seaweedfs    ███████████ 66 ops/s
minio        █████████ 50 ops/s
localstack   ██████ 34 ops/s
rustfs       ██ 17 ops/s
```

**Latency (P50)**
```
liteio       ██ 6.0ms
liteio_mem   ████ 8.6ms
seaweedfs    ███████ 15.1ms
minio        █████████ 19.8ms
localstack   ██████████████ 29.5ms
rustfs       ██████████████████████████████ 60.0ms
```

### FileCount/List/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 8 ops/s | 129.9ms | 129.9ms | 129.9ms | 0 |
| minio | 5 ops/s | 191.1ms | 191.1ms | 191.1ms | 0 |
| liteio_mem | 5 ops/s | 201.7ms | 201.7ms | 201.7ms | 0 |
| liteio | 5 ops/s | 205.5ms | 205.5ms | 205.5ms | 0 |
| localstack | 5 ops/s | 212.5ms | 212.5ms | 212.5ms | 0 |
| rustfs | 1 ops/s | 754.5ms | 754.5ms | 754.5ms | 0 |

**Throughput**
```
seaweedfs    ██████████████████████████████ 8 ops/s
minio        ████████████████████ 5 ops/s
liteio_mem   ███████████████████ 5 ops/s
liteio       ██████████████████ 5 ops/s
localstack   ██████████████████ 5 ops/s
rustfs       █████ 1 ops/s
```

**Latency (P50)**
```
seaweedfs    █████ 129.9ms
minio        ███████ 191.1ms
liteio_mem   ████████ 201.7ms
liteio       ████████ 205.5ms
localstack   ████████ 212.5ms
rustfs       ██████████████████████████████ 754.5ms
```

### FileCount/Write/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1.55 MB/s | 628.2us | 628.2us | 628.2us | 0 |
| rustfs | 1.40 MB/s | 698.8us | 698.8us | 698.8us | 0 |
| localstack | 1.21 MB/s | 808.2us | 808.2us | 808.2us | 0 |
| minio | 1.15 MB/s | 849.6us | 849.6us | 849.6us | 0 |
| seaweedfs | 1.11 MB/s | 880.2us | 880.2us | 880.2us | 0 |
| liteio | 0.55 MB/s | 1.8ms | 1.8ms | 1.8ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 1.55 MB/s
rustfs       ██████████████████████████ 1.40 MB/s
localstack   ███████████████████████ 1.21 MB/s
minio        ██████████████████████ 1.15 MB/s
seaweedfs    █████████████████████ 1.11 MB/s
liteio       ██████████ 0.55 MB/s
```

**Latency (P50)**
```
liteio_mem   ██████████ 628.2us
rustfs       ███████████ 698.8us
localstack   █████████████ 808.2us
minio        ██████████████ 849.6us
seaweedfs    ██████████████ 880.2us
liteio       ██████████████████████████████ 1.8ms
```

### FileCount/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1.70 MB/s | 5.8ms | 5.8ms | 5.8ms | 0 |
| rustfs | 1.44 MB/s | 6.8ms | 6.8ms | 6.8ms | 0 |
| seaweedfs | 1.30 MB/s | 7.5ms | 7.5ms | 7.5ms | 0 |
| localstack | 1.29 MB/s | 7.6ms | 7.6ms | 7.6ms | 0 |
| liteio | 1.10 MB/s | 8.9ms | 8.9ms | 8.9ms | 0 |
| minio | 1.09 MB/s | 9.0ms | 9.0ms | 9.0ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 1.70 MB/s
rustfs       █████████████████████████ 1.44 MB/s
seaweedfs    ███████████████████████ 1.30 MB/s
localstack   ██████████████████████ 1.29 MB/s
liteio       ███████████████████ 1.10 MB/s
minio        ███████████████████ 1.09 MB/s
```

**Latency (P50)**
```
liteio_mem   ███████████████████ 5.8ms
rustfs       ██████████████████████ 6.8ms
seaweedfs    █████████████████████████ 7.5ms
localstack   █████████████████████████ 7.6ms
liteio       █████████████████████████████ 8.9ms
minio        ██████████████████████████████ 9.0ms
```

### FileCount/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1.48 MB/s | 66.1ms | 66.1ms | 66.1ms | 0 |
| rustfs | 1.44 MB/s | 68.0ms | 68.0ms | 68.0ms | 0 |
| seaweedfs | 1.39 MB/s | 70.3ms | 70.3ms | 70.3ms | 0 |
| localstack | 1.18 MB/s | 82.8ms | 82.8ms | 82.8ms | 0 |
| minio | 1.06 MB/s | 92.3ms | 92.3ms | 92.3ms | 0 |
| liteio | 0.59 MB/s | 166.7ms | 166.7ms | 166.7ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 1.48 MB/s
rustfs       █████████████████████████████ 1.44 MB/s
seaweedfs    ████████████████████████████ 1.39 MB/s
localstack   ███████████████████████ 1.18 MB/s
minio        █████████████████████ 1.06 MB/s
liteio       ███████████ 0.59 MB/s
```

**Latency (P50)**
```
liteio_mem   ███████████ 66.1ms
rustfs       ████████████ 68.0ms
seaweedfs    ████████████ 70.3ms
localstack   ██████████████ 82.8ms
minio        ████████████████ 92.3ms
liteio       ██████████████████████████████ 166.7ms
```

### FileCount/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1.54 MB/s | 635.6ms | 635.6ms | 635.6ms | 0 |
| rustfs | 1.41 MB/s | 694.4ms | 694.4ms | 694.4ms | 0 |
| seaweedfs | 1.32 MB/s | 738.5ms | 738.5ms | 738.5ms | 0 |
| localstack | 1.23 MB/s | 793.5ms | 793.5ms | 793.5ms | 0 |
| minio | 1.13 MB/s | 866.9ms | 866.9ms | 866.9ms | 0 |
| liteio | 0.77 MB/s | 1.27s | 1.27s | 1.27s | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 1.54 MB/s
rustfs       ███████████████████████████ 1.41 MB/s
seaweedfs    █████████████████████████ 1.32 MB/s
localstack   ████████████████████████ 1.23 MB/s
minio        █████████████████████ 1.13 MB/s
liteio       ███████████████ 0.77 MB/s
```

**Latency (P50)**
```
liteio_mem   ███████████████ 635.6ms
rustfs       ████████████████ 694.4ms
seaweedfs    █████████████████ 738.5ms
localstack   ██████████████████ 793.5ms
minio        ████████████████████ 866.9ms
liteio       ██████████████████████████████ 1.27s
```

### FileCount/Write/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1.84 MB/s | 5.30s | 5.30s | 5.30s | 0 |
| rustfs | 1.40 MB/s | 6.99s | 6.99s | 6.99s | 0 |
| liteio | 1.31 MB/s | 7.44s | 7.44s | 7.44s | 0 |
| seaweedfs | 1.25 MB/s | 7.80s | 7.80s | 7.80s | 0 |
| localstack | 1.20 MB/s | 8.15s | 8.15s | 8.15s | 0 |
| minio | 0.91 MB/s | 10.72s | 10.72s | 10.72s | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 1.84 MB/s
rustfs       ██████████████████████ 1.40 MB/s
liteio       █████████████████████ 1.31 MB/s
seaweedfs    ████████████████████ 1.25 MB/s
localstack   ███████████████████ 1.20 MB/s
minio        ██████████████ 0.91 MB/s
```

**Latency (P50)**
```
liteio_mem   ██████████████ 5.30s
rustfs       ███████████████████ 6.99s
liteio       ████████████████████ 7.44s
seaweedfs    █████████████████████ 7.80s
localstack   ██████████████████████ 8.15s
minio        ██████████████████████████████ 10.72s
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1317 ops/s | 745.4us | 886.4us | 903.9us | 0 |
| liteio_mem | 1224 ops/s | 753.4us | 1.2ms | 1.3ms | 0 |
| minio | 663 ops/s | 1.5ms | 1.7ms | 1.7ms | 0 |
| seaweedfs | 498 ops/s | 2.1ms | 2.7ms | 2.9ms | 0 |
| localstack | 329 ops/s | 2.7ms | 3.7ms | 3.8ms | 0 |
| rustfs | 133 ops/s | 6.9ms | 10.1ms | 15.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1317 ops/s
liteio_mem   ███████████████████████████ 1224 ops/s
minio        ███████████████ 663 ops/s
seaweedfs    ███████████ 498 ops/s
localstack   ███████ 329 ops/s
rustfs       ███ 133 ops/s
```

**Latency (P50)**
```
liteio       ███ 745.4us
liteio_mem   ███ 753.4us
minio        ██████ 1.5ms
seaweedfs    █████████ 2.1ms
localstack   ███████████ 2.7ms
rustfs       ██████████████████████████████ 6.9ms
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 2.33 MB/s | 7.2ms | 9.4ms | 10.2ms | 0 |
| rustfs | 1.62 MB/s | 10.9ms | 14.8ms | 15.4ms | 0 |
| liteio | 1.54 MB/s | 9.2ms | 15.8ms | 16.2ms | 0 |
| liteio_mem | 1.42 MB/s | 9.8ms | 16.6ms | 16.8ms | 0 |
| minio | 0.94 MB/s | 11.6ms | 28.6ms | 29.4ms | 0 |
| localstack | 0.32 MB/s | 59.7ms | 60.8ms | 61.0ms | 0 |

**Throughput**
```
seaweedfs    ██████████████████████████████ 2.33 MB/s
rustfs       ████████████████████ 1.62 MB/s
liteio       ███████████████████ 1.54 MB/s
liteio_mem   ██████████████████ 1.42 MB/s
minio        ████████████ 0.94 MB/s
localstack   ████ 0.32 MB/s
```

**Latency (P50)**
```
seaweedfs    ███ 7.2ms
rustfs       █████ 10.9ms
liteio       ████ 9.2ms
liteio_mem   ████ 9.8ms
minio        █████ 11.6ms
localstack   ██████████████████████████████ 59.7ms
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 3.73 MB/s | 4.0ms | 6.6ms | 7.8ms | 0 |
| rustfs | 2.03 MB/s | 7.2ms | 11.3ms | 23.7ms | 0 |
| liteio | 1.55 MB/s | 10.1ms | 11.5ms | 11.6ms | 0 |
| liteio_mem | 1.53 MB/s | 10.3ms | 11.0ms | 11.1ms | 0 |
| minio | 1.47 MB/s | 10.8ms | 12.0ms | 12.2ms | 0 |
| localstack | 0.29 MB/s | 61.5ms | 62.4ms | 62.5ms | 0 |

**Throughput**
```
seaweedfs    ██████████████████████████████ 3.73 MB/s
rustfs       ████████████████ 2.03 MB/s
liteio       ████████████ 1.55 MB/s
liteio_mem   ████████████ 1.53 MB/s
minio        ███████████ 1.47 MB/s
localstack   ██ 0.29 MB/s
```

**Latency (P50)**
```
seaweedfs    █ 4.0ms
rustfs       ███ 7.2ms
liteio       ████ 10.1ms
liteio_mem   █████ 10.3ms
minio        █████ 10.8ms
localstack   ██████████████████████████████ 61.5ms
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 1.87 MB/s | 8.5ms | 11.3ms | 12.1ms | 0 |
| rustfs | 1.46 MB/s | 10.9ms | 15.4ms | 16.1ms | 0 |
| liteio | 1.07 MB/s | 15.3ms | 20.9ms | 21.9ms | 0 |
| liteio_mem | 0.90 MB/s | 17.9ms | 22.7ms | 23.4ms | 0 |
| minio | 0.62 MB/s | 28.5ms | 32.2ms | 32.6ms | 0 |
| localstack | 0.27 MB/s | 61.8ms | 65.9ms | 66.3ms | 0 |

**Throughput**
```
seaweedfs    ██████████████████████████████ 1.87 MB/s
rustfs       ███████████████████████ 1.46 MB/s
liteio       █████████████████ 1.07 MB/s
liteio_mem   ██████████████ 0.90 MB/s
minio        █████████ 0.62 MB/s
localstack   ████ 0.27 MB/s
```

**Latency (P50)**
```
seaweedfs    ████ 8.5ms
rustfs       █████ 10.9ms
liteio       ███████ 15.3ms
liteio_mem   ████████ 17.9ms
minio        █████████████ 28.5ms
localstack   ██████████████████████████████ 61.8ms
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 173.85 MB/s | 84.4ms | 95.2ms | 95.2ms | 0 |
| localstack | 127.59 MB/s | 115.6ms | 125.3ms | 125.3ms | 0 |
| liteio_mem | 126.32 MB/s | 112.3ms | 159.9ms | 159.9ms | 0 |
| minio | 124.92 MB/s | 110.1ms | 151.7ms | 151.7ms | 0 |
| seaweedfs | 114.84 MB/s | 119.3ms | 172.0ms | 172.0ms | 0 |
| liteio | 110.75 MB/s | 120.2ms | 174.5ms | 174.5ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 173.85 MB/s
localstack   ██████████████████████ 127.59 MB/s
liteio_mem   █████████████████████ 126.32 MB/s
minio        █████████████████████ 124.92 MB/s
seaweedfs    ███████████████████ 114.84 MB/s
liteio       ███████████████████ 110.75 MB/s
```

**Latency (P50)**
```
rustfs       █████████████████████ 84.4ms
localstack   ████████████████████████████ 115.6ms
liteio_mem   ████████████████████████████ 112.3ms
minio        ███████████████████████████ 110.1ms
seaweedfs    █████████████████████████████ 119.3ms
liteio       ██████████████████████████████ 120.2ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 4.02 MB/s | 242.0us | 262.2us | 208.0us | 262.3us | 1.3ms | 0 |
| liteio_mem | 3.79 MB/s | 256.0us | 282.2us | 211.8us | 311.0us | 1.9ms | 0 |
| minio | 2.79 MB/s | 349.9us | 398.4us | 351.0us | 398.5us | 423.8us | 0 |
| seaweedfs | 1.69 MB/s | 579.0us | 705.3us | 565.2us | 705.7us | 758.2us | 0 |
| rustfs | 1.44 MB/s | 675.6us | 832.7us | 665.8us | 832.9us | 926.0us | 0 |
| localstack | 1.09 MB/s | 895.1us | 1.1ms | 849.5us | 1.1ms | 1.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4.02 MB/s
liteio_mem   ████████████████████████████ 3.79 MB/s
minio        ████████████████████ 2.79 MB/s
seaweedfs    ████████████ 1.69 MB/s
rustfs       ██████████ 1.44 MB/s
localstack   ████████ 1.09 MB/s
```

**Latency (P50)**
```
liteio       ███████ 208.0us
liteio_mem   ███████ 211.8us
minio        ████████████ 351.0us
seaweedfs    ███████████████████ 565.2us
rustfs       ███████████████████████ 665.8us
localstack   ██████████████████████████████ 849.5us
```

### ParallelRead/1KB/C10

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 1.23 MB/s | 790.0us | 1.1ms | 768.9us | 1.1ms | 1.4ms | 0 |
| liteio | 1.08 MB/s | 887.1us | 1.6ms | 819.8us | 1.6ms | 1.8ms | 0 |
| liteio_mem | 0.97 MB/s | 1.0ms | 2.8ms | 719.7us | 2.8ms | 4.5ms | 0 |
| seaweedfs | 0.91 MB/s | 1.1ms | 1.5ms | 1.1ms | 1.5ms | 1.8ms | 0 |
| rustfs | 0.78 MB/s | 1.3ms | 1.7ms | 1.2ms | 1.7ms | 2.0ms | 0 |
| localstack | 0.19 MB/s | 5.1ms | 8.3ms | 4.6ms | 8.3ms | 8.8ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 1.23 MB/s
liteio       ██████████████████████████ 1.08 MB/s
liteio_mem   ███████████████████████ 0.97 MB/s
seaweedfs    ██████████████████████ 0.91 MB/s
rustfs       ██████████████████ 0.78 MB/s
localstack   ████ 0.19 MB/s
```

**Latency (P50)**
```
minio        ████ 768.9us
liteio       █████ 819.8us
liteio_mem   ████ 719.7us
seaweedfs    ██████ 1.1ms
rustfs       ███████ 1.2ms
localstack   ██████████████████████████████ 4.6ms
```

### ParallelRead/1KB/C100

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| seaweedfs | 0.46 MB/s | 2.1ms | 3.0ms | 2.0ms | 3.0ms | 3.9ms | 0 |
| rustfs | 0.38 MB/s | 2.5ms | 3.6ms | 2.6ms | 3.6ms | 4.7ms | 0 |
| liteio | 0.35 MB/s | 2.8ms | 3.7ms | 3.0ms | 3.7ms | 4.0ms | 0 |
| liteio_mem | 0.32 MB/s | 3.1ms | 4.8ms | 2.9ms | 4.8ms | 4.9ms | 0 |
| minio | 0.24 MB/s | 4.2ms | 4.7ms | 4.2ms | 4.7ms | 4.9ms | 0 |
| localstack | 0.02 MB/s | 49.4ms | 61.7ms | 49.0ms | 61.7ms | 62.3ms | 0 |

**Throughput**
```
seaweedfs    ██████████████████████████████ 0.46 MB/s
rustfs       █████████████████████████ 0.38 MB/s
liteio       ██████████████████████ 0.35 MB/s
liteio_mem   ████████████████████ 0.32 MB/s
minio        ███████████████ 0.24 MB/s
localstack   █ 0.02 MB/s
```

**Latency (P50)**
```
seaweedfs    █ 2.0ms
rustfs       █ 2.6ms
liteio       █ 3.0ms
liteio_mem   █ 2.9ms
minio        ██ 4.2ms
localstack   ██████████████████████████████ 49.0ms
```

### ParallelRead/1KB/C200

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| seaweedfs | 0.32 MB/s | 3.0ms | 4.6ms | 2.8ms | 4.6ms | 4.7ms | 0 |
| rustfs | 0.31 MB/s | 3.1ms | 5.1ms | 3.5ms | 5.1ms | 5.5ms | 0 |
| liteio | 0.27 MB/s | 3.7ms | 4.8ms | 3.7ms | 4.8ms | 5.1ms | 0 |
| liteio_mem | 0.26 MB/s | 3.8ms | 5.1ms | 3.7ms | 5.1ms | 5.4ms | 0 |
| minio | 0.20 MB/s | 5.0ms | 5.9ms | 4.9ms | 5.9ms | 6.2ms | 0 |
| localstack | 0.02 MB/s | 44.3ms | 51.5ms | 46.5ms | 51.5ms | 51.9ms | 0 |

**Throughput**
```
seaweedfs    ██████████████████████████████ 0.32 MB/s
rustfs       █████████████████████████████ 0.31 MB/s
liteio       ████████████████████████ 0.27 MB/s
liteio_mem   ███████████████████████ 0.26 MB/s
minio        ██████████████████ 0.20 MB/s
localstack   ██ 0.02 MB/s
```

**Latency (P50)**
```
seaweedfs    █ 2.8ms
rustfs       ██ 3.5ms
liteio       ██ 3.7ms
liteio_mem   ██ 3.7ms
minio        ███ 4.9ms
localstack   ██████████████████████████████ 46.5ms
```

### ParallelRead/1KB/C25

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 0.83 MB/s | 1.2ms | 1.7ms | 1.1ms | 1.7ms | 1.8ms | 0 |
| liteio_mem | 0.79 MB/s | 1.2ms | 1.8ms | 1.1ms | 1.8ms | 2.8ms | 0 |
| minio | 0.65 MB/s | 1.5ms | 2.3ms | 1.4ms | 2.3ms | 2.6ms | 0 |
| seaweedfs | 0.56 MB/s | 1.8ms | 2.5ms | 1.6ms | 2.5ms | 2.8ms | 0 |
| rustfs | 0.55 MB/s | 1.8ms | 2.3ms | 1.7ms | 2.3ms | 2.8ms | 0 |
| localstack | 0.08 MB/s | 12.5ms | 16.4ms | 11.7ms | 16.4ms | 24.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.83 MB/s
liteio_mem   ████████████████████████████ 0.79 MB/s
minio        ███████████████████████ 0.65 MB/s
seaweedfs    ████████████████████ 0.56 MB/s
rustfs       ███████████████████ 0.55 MB/s
localstack   ██ 0.08 MB/s
```

**Latency (P50)**
```
liteio       ██ 1.1ms
liteio_mem   ██ 1.1ms
minio        ███ 1.4ms
seaweedfs    ████ 1.6ms
rustfs       ████ 1.7ms
localstack   ██████████████████████████████ 11.7ms
```

### ParallelRead/1KB/C50

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio_mem | 0.49 MB/s | 2.0ms | 2.9ms | 2.0ms | 3.0ms | 3.2ms | 0 |
| seaweedfs | 0.48 MB/s | 2.0ms | 2.9ms | 1.9ms | 2.9ms | 3.2ms | 0 |
| liteio | 0.40 MB/s | 2.4ms | 4.1ms | 2.1ms | 4.1ms | 4.3ms | 0 |
| rustfs | 0.39 MB/s | 2.5ms | 4.0ms | 2.4ms | 4.0ms | 4.5ms | 0 |
| minio | 0.36 MB/s | 2.7ms | 3.4ms | 2.8ms | 3.4ms | 3.6ms | 0 |
| localstack | 0.04 MB/s | 26.4ms | 45.8ms | 25.9ms | 45.8ms | 46.3ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 0.49 MB/s
seaweedfs    █████████████████████████████ 0.48 MB/s
liteio       ████████████████████████ 0.40 MB/s
rustfs       ███████████████████████ 0.39 MB/s
minio        ██████████████████████ 0.36 MB/s
localstack   ██ 0.04 MB/s
```

**Latency (P50)**
```
liteio_mem   ██ 2.0ms
seaweedfs    ██ 1.9ms
liteio       ██ 2.1ms
rustfs       ██ 2.4ms
minio        ███ 2.8ms
localstack   ██████████████████████████████ 25.9ms
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1.52 MB/s | 597.5us | 765.0us | 1.7ms | 0 |
| liteio | 1.44 MB/s | 638.2us | 891.8us | 1.3ms | 0 |
| localstack | 1.03 MB/s | 891.9us | 1.2ms | 1.2ms | 0 |
| rustfs | 1.00 MB/s | 855.5us | 1.6ms | 2.0ms | 0 |
| minio | 0.96 MB/s | 964.1us | 1.3ms | 1.6ms | 0 |
| seaweedfs | 0.95 MB/s | 903.2us | 1.9ms | 2.2ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 1.52 MB/s
liteio       ████████████████████████████ 1.44 MB/s
localstack   ████████████████████ 1.03 MB/s
rustfs       ███████████████████ 1.00 MB/s
minio        ███████████████████ 0.96 MB/s
seaweedfs    ██████████████████ 0.95 MB/s
```

**Latency (P50)**
```
liteio_mem   ██████████████████ 597.5us
liteio       ███████████████████ 638.2us
localstack   ███████████████████████████ 891.9us
rustfs       ██████████████████████████ 855.5us
minio        ██████████████████████████████ 964.1us
seaweedfs    ████████████████████████████ 903.2us
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 0.44 MB/s | 2.1ms | 3.5ms | 4.2ms | 0 |
| liteio_mem | 0.43 MB/s | 2.0ms | 5.5ms | 6.3ms | 0 |
| liteio | 0.42 MB/s | 2.0ms | 5.5ms | 6.5ms | 0 |
| minio | 0.39 MB/s | 2.5ms | 3.7ms | 4.0ms | 0 |
| rustfs | 0.28 MB/s | 2.9ms | 8.4ms | 9.0ms | 0 |
| localstack | 0.19 MB/s | 4.8ms | 8.6ms | 8.9ms | 0 |

**Throughput**
```
seaweedfs    ██████████████████████████████ 0.44 MB/s
liteio_mem   █████████████████████████████ 0.43 MB/s
liteio       ████████████████████████████ 0.42 MB/s
minio        ██████████████████████████ 0.39 MB/s
rustfs       ███████████████████ 0.28 MB/s
localstack   ████████████ 0.19 MB/s
```

**Latency (P50)**
```
seaweedfs    █████████████ 2.1ms
liteio_mem   ████████████ 2.0ms
liteio       ████████████ 2.0ms
minio        ███████████████ 2.5ms
rustfs       ██████████████████ 2.9ms
localstack   ██████████████████████████████ 4.8ms
```

### ParallelWrite/1KB/C100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 0.16 MB/s | 6.7ms | 8.8ms | 9.4ms | 0 |
| seaweedfs | 0.10 MB/s | 8.8ms | 13.2ms | 14.0ms | 0 |
| minio | 0.07 MB/s | 13.3ms | 19.4ms | 19.8ms | 0 |
| liteio_mem | 0.07 MB/s | 15.2ms | 22.1ms | 22.4ms | 0 |
| liteio | 0.06 MB/s | 14.9ms | 37.4ms | 38.0ms | 0 |
| localstack | 0.03 MB/s | 27.6ms | 48.9ms | 49.4ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 0.16 MB/s
seaweedfs    ███████████████████ 0.10 MB/s
minio        █████████████ 0.07 MB/s
liteio_mem   ████████████ 0.07 MB/s
liteio       ███████████ 0.06 MB/s
localstack   █████ 0.03 MB/s
```

**Latency (P50)**
```
rustfs       ███████ 6.7ms
seaweedfs    █████████ 8.8ms
minio        ██████████████ 13.3ms
liteio_mem   ████████████████ 15.2ms
liteio       ████████████████ 14.9ms
localstack   ██████████████████████████████ 27.6ms
```

### ParallelWrite/1KB/C200

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 0.12 MB/s | 8.2ms | 10.6ms | 11.5ms | 0 |
| rustfs | 0.10 MB/s | 10.9ms | 14.2ms | 14.6ms | 0 |
| liteio | 0.09 MB/s | 10.4ms | 15.5ms | 16.0ms | 0 |
| liteio_mem | 0.09 MB/s | 11.4ms | 16.1ms | 16.4ms | 0 |
| minio | 0.07 MB/s | 13.0ms | 18.5ms | 19.6ms | 0 |
| localstack | 0.02 MB/s | 48.6ms | 50.1ms | 50.6ms | 0 |

**Throughput**
```
seaweedfs    ██████████████████████████████ 0.12 MB/s
rustfs       ████████████████████████ 0.10 MB/s
liteio       ███████████████████████ 0.09 MB/s
liteio_mem   ██████████████████████ 0.09 MB/s
minio        ██████████████████ 0.07 MB/s
localstack   █████ 0.02 MB/s
```

**Latency (P50)**
```
seaweedfs    █████ 8.2ms
rustfs       ██████ 10.9ms
liteio       ██████ 10.4ms
liteio_mem   ███████ 11.4ms
minio        ████████ 13.0ms
localstack   ██████████████████████████████ 48.6ms
```

### ParallelWrite/1KB/C25

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 0.28 MB/s | 3.2ms | 6.5ms | 7.0ms | 0 |
| liteio | 0.25 MB/s | 3.5ms | 6.9ms | 8.0ms | 0 |
| liteio_mem | 0.23 MB/s | 3.8ms | 6.8ms | 9.0ms | 0 |
| minio | 0.19 MB/s | 4.7ms | 9.3ms | 11.0ms | 0 |
| rustfs | 0.16 MB/s | 4.3ms | 15.5ms | 16.5ms | 0 |
| localstack | 0.08 MB/s | 11.7ms | 20.6ms | 25.8ms | 0 |

**Throughput**
```
seaweedfs    ██████████████████████████████ 0.28 MB/s
liteio       ██████████████████████████ 0.25 MB/s
liteio_mem   █████████████████████████ 0.23 MB/s
minio        ████████████████████ 0.19 MB/s
rustfs       █████████████████ 0.16 MB/s
localstack   ████████ 0.08 MB/s
```

**Latency (P50)**
```
seaweedfs    ████████ 3.2ms
liteio       █████████ 3.5ms
liteio_mem   █████████ 3.8ms
minio        ████████████ 4.7ms
rustfs       ███████████ 4.3ms
localstack   ██████████████████████████████ 11.7ms
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 0.20 MB/s | 4.7ms | 8.5ms | 8.7ms | 0 |
| liteio_mem | 0.12 MB/s | 7.0ms | 15.6ms | 16.2ms | 0 |
| minio | 0.12 MB/s | 7.1ms | 17.7ms | 18.3ms | 0 |
| liteio | 0.12 MB/s | 6.9ms | 17.0ms | 17.3ms | 0 |
| rustfs | 0.08 MB/s | 8.9ms | 27.1ms | 28.2ms | 0 |
| localstack | 0.03 MB/s | 23.5ms | 56.2ms | 56.7ms | 0 |

**Throughput**
```
seaweedfs    ██████████████████████████████ 0.20 MB/s
liteio_mem   ██████████████████ 0.12 MB/s
minio        █████████████████ 0.12 MB/s
liteio       █████████████████ 0.12 MB/s
rustfs       ████████████ 0.08 MB/s
localstack   █████ 0.03 MB/s
```

**Latency (P50)**
```
seaweedfs    ██████ 4.7ms
liteio_mem   ████████ 7.0ms
minio        █████████ 7.1ms
liteio       ████████ 6.9ms
rustfs       ███████████ 8.9ms
localstack   ██████████████████████████████ 23.5ms
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 236.56 MB/s | 1.0ms | 1.2ms | 1.4ms | 0 |
| liteio | 233.74 MB/s | 1.0ms | 1.3ms | 1.4ms | 0 |
| seaweedfs | 191.55 MB/s | 1.3ms | 1.4ms | 1.5ms | 0 |
| localstack | 154.45 MB/s | 1.5ms | 2.2ms | 2.7ms | 0 |
| minio | 144.92 MB/s | 1.6ms | 2.0ms | 5.1ms | 0 |
| rustfs | 117.17 MB/s | 2.0ms | 2.7ms | 3.0ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 236.56 MB/s
liteio       █████████████████████████████ 233.74 MB/s
seaweedfs    ████████████████████████ 191.55 MB/s
localstack   ███████████████████ 154.45 MB/s
minio        ██████████████████ 144.92 MB/s
rustfs       ██████████████ 117.17 MB/s
```

**Latency (P50)**
```
liteio_mem   ███████████████ 1.0ms
liteio       ███████████████ 1.0ms
seaweedfs    ███████████████████ 1.3ms
localstack   ██████████████████████ 1.5ms
minio        ███████████████████████ 1.6ms
rustfs       ██████████████████████████████ 2.0ms
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 221.27 MB/s | 1.1ms | 1.5ms | 1.5ms | 0 |
| seaweedfs | 192.60 MB/s | 1.3ms | 1.4ms | 1.5ms | 0 |
| liteio | 187.62 MB/s | 1.2ms | 1.8ms | 2.2ms | 0 |
| minio | 166.51 MB/s | 1.4ms | 1.8ms | 1.8ms | 0 |
| rustfs | 108.49 MB/s | 2.1ms | 2.9ms | 9.7ms | 0 |
| localstack | 106.32 MB/s | 1.5ms | 2.4ms | 3.3ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 221.27 MB/s
seaweedfs    ██████████████████████████ 192.60 MB/s
liteio       █████████████████████████ 187.62 MB/s
minio        ██████████████████████ 166.51 MB/s
rustfs       ██████████████ 108.49 MB/s
localstack   ██████████████ 106.32 MB/s
```

**Latency (P50)**
```
liteio_mem   ███████████████ 1.1ms
seaweedfs    ██████████████████ 1.3ms
liteio       ██████████████████ 1.2ms
minio        ████████████████████ 1.4ms
rustfs       ██████████████████████████████ 2.1ms
localstack   ██████████████████████ 1.5ms
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 235.33 MB/s | 1.0ms | 1.3ms | 1.6ms | 0 |
| liteio | 207.57 MB/s | 1.1ms | 1.9ms | 2.1ms | 0 |
| seaweedfs | 182.56 MB/s | 1.3ms | 1.6ms | 2.2ms | 0 |
| localstack | 165.15 MB/s | 1.5ms | 1.8ms | 2.0ms | 0 |
| minio | 157.05 MB/s | 1.6ms | 1.8ms | 2.0ms | 0 |
| rustfs | 125.13 MB/s | 2.0ms | 2.5ms | 2.7ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 235.33 MB/s
liteio       ██████████████████████████ 207.57 MB/s
seaweedfs    ███████████████████████ 182.56 MB/s
localstack   █████████████████████ 165.15 MB/s
minio        ████████████████████ 157.05 MB/s
rustfs       ███████████████ 125.13 MB/s
```

**Latency (P50)**
```
liteio_mem   ███████████████ 1.0ms
liteio       ████████████████ 1.1ms
seaweedfs    ████████████████████ 1.3ms
localstack   ██████████████████████ 1.5ms
minio        ████████████████████████ 1.6ms
rustfs       ██████████████████████████████ 2.0ms
```

### Read/100MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 330.75 MB/s | 897.9us | 923.6us | 301.3ms | 301.4ms | 301.4ms | 0 |
| localstack | 316.32 MB/s | 1.5ms | 1.5ms | 317.5ms | 318.9ms | 318.9ms | 0 |
| rustfs | 316.15 MB/s | 1.9ms | 2.1ms | 315.0ms | 316.7ms | 316.7ms | 0 |
| seaweedfs | 300.05 MB/s | 2.1ms | 2.3ms | 336.9ms | 339.1ms | 339.1ms | 0 |
| liteio_mem | 281.51 MB/s | 658.4us | 571.5us | 345.7ms | 373.1ms | 373.1ms | 0 |
| liteio | 269.28 MB/s | 495.0us | 495.2us | 382.2ms | 382.2ms | 382.2ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 330.75 MB/s
localstack   ████████████████████████████ 316.32 MB/s
rustfs       ████████████████████████████ 316.15 MB/s
seaweedfs    ███████████████████████████ 300.05 MB/s
liteio_mem   █████████████████████████ 281.51 MB/s
liteio       ████████████████████████ 269.28 MB/s
```

**Latency (P50)**
```
minio        ███████████████████████ 301.3ms
localstack   ████████████████████████ 317.5ms
rustfs       ████████████████████████ 315.0ms
seaweedfs    ██████████████████████████ 336.9ms
liteio_mem   ███████████████████████████ 345.7ms
liteio       ██████████████████████████████ 382.2ms
```

### Read/10MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 316.85 MB/s | 1.2ms | 1.8ms | 31.1ms | 32.1ms | 32.1ms | 0 |
| localstack | 313.39 MB/s | 1.3ms | 1.4ms | 31.7ms | 33.0ms | 33.0ms | 0 |
| liteio_mem | 309.81 MB/s | 473.3us | 528.2us | 32.0ms | 33.2ms | 33.2ms | 0 |
| seaweedfs | 301.47 MB/s | 1.9ms | 2.3ms | 32.9ms | 34.7ms | 34.7ms | 0 |
| liteio | 274.19 MB/s | 475.1us | 527.7us | 36.3ms | 38.7ms | 38.7ms | 0 |
| rustfs | 270.61 MB/s | 6.0ms | 8.0ms | 36.9ms | 38.3ms | 38.3ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 316.85 MB/s
localstack   █████████████████████████████ 313.39 MB/s
liteio_mem   █████████████████████████████ 309.81 MB/s
seaweedfs    ████████████████████████████ 301.47 MB/s
liteio       █████████████████████████ 274.19 MB/s
rustfs       █████████████████████████ 270.61 MB/s
```

**Latency (P50)**
```
minio        █████████████████████████ 31.1ms
localstack   █████████████████████████ 31.7ms
liteio_mem   ██████████████████████████ 32.0ms
seaweedfs    ██████████████████████████ 32.9ms
liteio       █████████████████████████████ 36.3ms
rustfs       ██████████████████████████████ 36.9ms
```

### Read/1KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 5.04 MB/s | 193.3us | 244.8us | 173.9us | 245.9us | 620.0us | 0 |
| liteio_mem | 4.26 MB/s | 227.3us | 286.9us | 205.0us | 287.0us | 890.2us | 0 |
| minio | 2.62 MB/s | 372.2us | 416.9us | 370.5us | 417.1us | 447.3us | 0 |
| seaweedfs | 2.17 MB/s | 448.7us | 587.1us | 432.9us | 587.3us | 652.7us | 0 |
| rustfs | 1.81 MB/s | 539.7us | 639.0us | 526.7us | 639.5us | 711.0us | 0 |
| localstack | 1.30 MB/s | 751.1us | 896.8us | 732.4us | 897.6us | 1.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 5.04 MB/s
liteio_mem   █████████████████████████ 4.26 MB/s
minio        ███████████████ 2.62 MB/s
seaweedfs    ████████████ 2.17 MB/s
rustfs       ██████████ 1.81 MB/s
localstack   ███████ 1.30 MB/s
```

**Latency (P50)**
```
liteio       ███████ 173.9us
liteio_mem   ████████ 205.0us
minio        ███████████████ 370.5us
seaweedfs    █████████████████ 432.9us
rustfs       █████████████████████ 526.7us
localstack   ██████████████████████████████ 732.4us
```

### Read/1MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio_mem | 273.23 MB/s | 452.9us | 723.1us | 3.6ms | 4.4ms | 4.4ms | 0 |
| seaweedfs | 259.39 MB/s | 875.2us | 1.1ms | 3.8ms | 4.0ms | 4.0ms | 0 |
| localstack | 258.53 MB/s | 917.9us | 1.0ms | 3.9ms | 4.2ms | 4.2ms | 0 |
| minio | 242.90 MB/s | 1.0ms | 1.1ms | 4.1ms | 4.4ms | 4.4ms | 0 |
| liteio | 235.64 MB/s | 558.7us | 834.4us | 4.0ms | 5.2ms | 5.2ms | 0 |
| rustfs | 214.64 MB/s | 1.7ms | 3.0ms | 4.3ms | 5.9ms | 5.9ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 273.23 MB/s
seaweedfs    ████████████████████████████ 259.39 MB/s
localstack   ████████████████████████████ 258.53 MB/s
minio        ██████████████████████████ 242.90 MB/s
liteio       █████████████████████████ 235.64 MB/s
rustfs       ███████████████████████ 214.64 MB/s
```

**Latency (P50)**
```
liteio_mem   █████████████████████████ 3.6ms
seaweedfs    ███████████████████████████ 3.8ms
localstack   ███████████████████████████ 3.9ms
minio        ████████████████████████████ 4.1ms
liteio       ████████████████████████████ 4.0ms
rustfs       ██████████████████████████████ 4.3ms
```

### Read/64KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 150.38 MB/s | 192.7us | 271.7us | 409.1us | 465.8us | 496.1us | 0 |
| liteio_mem | 131.89 MB/s | 251.8us | 330.9us | 442.5us | 532.6us | 570.5us | 0 |
| minio | 102.21 MB/s | 446.4us | 495.6us | 605.3us | 670.8us | 674.8us | 0 |
| seaweedfs | 96.18 MB/s | 471.6us | 523.5us | 646.2us | 687.5us | 702.6us | 0 |
| rustfs | 85.14 MB/s | 633.4us | 689.5us | 725.5us | 771.1us | 827.3us | 0 |
| localstack | 67.41 MB/s | 819.7us | 862.6us | 917.8us | 967.0us | 1.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 150.38 MB/s
liteio_mem   ██████████████████████████ 131.89 MB/s
minio        ████████████████████ 102.21 MB/s
seaweedfs    ███████████████████ 96.18 MB/s
rustfs       ████████████████ 85.14 MB/s
localstack   █████████████ 67.41 MB/s
```

**Latency (P50)**
```
liteio       █████████████ 409.1us
liteio_mem   ██████████████ 442.5us
minio        ███████████████████ 605.3us
seaweedfs    █████████████████████ 646.2us
rustfs       ███████████████████████ 725.5us
localstack   ██████████████████████████████ 917.8us
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 4180 ops/s | 219.6us | 372.2us | 528.8us | 0 |
| liteio | 4076 ops/s | 200.3us | 441.8us | 685.0us | 0 |
| minio | 4061 ops/s | 244.9us | 285.0us | 297.8us | 0 |
| rustfs | 2537 ops/s | 392.5us | 457.7us | 474.0us | 0 |
| seaweedfs | 1749 ops/s | 374.2us | 1.6ms | 2.1ms | 0 |
| localstack | 1609 ops/s | 603.3us | 716.5us | 1.1ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 4180 ops/s
liteio       █████████████████████████████ 4076 ops/s
minio        █████████████████████████████ 4061 ops/s
rustfs       ██████████████████ 2537 ops/s
seaweedfs    ████████████ 1749 ops/s
localstack   ███████████ 1609 ops/s
```

**Latency (P50)**
```
liteio_mem   ██████████ 219.6us
liteio       █████████ 200.3us
minio        ████████████ 244.9us
rustfs       ███████████████████ 392.5us
seaweedfs    ██████████████████ 374.2us
localstack   ██████████████████████████████ 603.3us
```

### Write/100MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 194.59 MB/s | 514.7ms | 517.2ms | 517.2ms | 0 |
| rustfs | 182.26 MB/s | 590.0ms | 597.0ms | 597.0ms | 0 |
| minio | 167.25 MB/s | 626.8ms | 648.3ms | 648.3ms | 0 |
| localstack | 137.25 MB/s | 666.8ms | 816.6ms | 816.6ms | 0 |
| liteio | 128.11 MB/s | 619.6ms | 635.4ms | 635.4ms | 0 |
| liteio_mem | 86.99 MB/s | 784.7ms | 1.65s | 1.65s | 0 |

**Throughput**
```
seaweedfs    ██████████████████████████████ 194.59 MB/s
rustfs       ████████████████████████████ 182.26 MB/s
minio        █████████████████████████ 167.25 MB/s
localstack   █████████████████████ 137.25 MB/s
liteio       ███████████████████ 128.11 MB/s
liteio_mem   █████████████ 86.99 MB/s
```

**Latency (P50)**
```
seaweedfs    ███████████████████ 514.7ms
rustfs       ██████████████████████ 590.0ms
minio        ███████████████████████ 626.8ms
localstack   █████████████████████████ 666.8ms
liteio       ███████████████████████ 619.6ms
liteio_mem   ██████████████████████████████ 784.7ms
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 194.90 MB/s | 49.4ms | 59.1ms | 59.1ms | 0 |
| minio | 152.54 MB/s | 65.6ms | 69.4ms | 69.4ms | 0 |
| seaweedfs | 150.90 MB/s | 64.6ms | 72.1ms | 72.1ms | 0 |
| localstack | 146.53 MB/s | 67.7ms | 70.9ms | 70.9ms | 0 |
| liteio_mem | 103.48 MB/s | 84.0ms | 126.4ms | 126.4ms | 0 |
| liteio | 55.46 MB/s | 98.1ms | 140.6ms | 140.6ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 194.90 MB/s
minio        ███████████████████████ 152.54 MB/s
seaweedfs    ███████████████████████ 150.90 MB/s
localstack   ██████████████████████ 146.53 MB/s
liteio_mem   ███████████████ 103.48 MB/s
liteio       ████████ 55.46 MB/s
```

**Latency (P50)**
```
rustfs       ███████████████ 49.4ms
minio        ████████████████████ 65.6ms
seaweedfs    ███████████████████ 64.6ms
localstack   ████████████████████ 67.7ms
liteio_mem   █████████████████████████ 84.0ms
liteio       ██████████████████████████████ 98.1ms
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 1.45 MB/s | 660.7us | 720.7us | 1.2ms | 0 |
| seaweedfs | 1.41 MB/s | 676.5us | 785.4us | 912.3us | 0 |
| minio | 1.36 MB/s | 660.1us | 1.0ms | 1.3ms | 0 |
| liteio_mem | 1.27 MB/s | 687.2us | 1.3ms | 1.9ms | 0 |
| localstack | 1.15 MB/s | 761.8us | 1.3ms | 1.9ms | 0 |
| liteio | 0.96 MB/s | 944.0us | 1.7ms | 2.1ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 1.45 MB/s
seaweedfs    █████████████████████████████ 1.41 MB/s
minio        ████████████████████████████ 1.36 MB/s
liteio_mem   ██████████████████████████ 1.27 MB/s
localstack   ███████████████████████ 1.15 MB/s
liteio       ███████████████████ 0.96 MB/s
```

**Latency (P50)**
```
rustfs       ████████████████████ 660.7us
seaweedfs    █████████████████████ 676.5us
minio        ████████████████████ 660.1us
liteio_mem   █████████████████████ 687.2us
localstack   ████████████████████████ 761.8us
liteio       ██████████████████████████████ 944.0us
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 152.80 MB/s | 6.3ms | 7.7ms | 7.7ms | 0 |
| liteio_mem | 139.38 MB/s | 7.0ms | 8.1ms | 8.1ms | 0 |
| liteio | 134.35 MB/s | 7.3ms | 8.3ms | 8.3ms | 0 |
| minio | 125.12 MB/s | 7.7ms | 8.8ms | 8.8ms | 0 |
| localstack | 124.50 MB/s | 7.9ms | 9.2ms | 9.2ms | 0 |
| seaweedfs | 116.97 MB/s | 8.3ms | 9.4ms | 9.4ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 152.80 MB/s
liteio_mem   ███████████████████████████ 139.38 MB/s
liteio       ██████████████████████████ 134.35 MB/s
minio        ████████████████████████ 125.12 MB/s
localstack   ████████████████████████ 124.50 MB/s
seaweedfs    ██████████████████████ 116.97 MB/s
```

**Latency (P50)**
```
rustfs       ██████████████████████ 6.3ms
liteio_mem   █████████████████████████ 7.0ms
liteio       ██████████████████████████ 7.3ms
minio        ███████████████████████████ 7.7ms
localstack   ████████████████████████████ 7.9ms
seaweedfs    ██████████████████████████████ 8.3ms
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 71.79 MB/s | 832.7us | 1.1ms | 1.1ms | 0 |
| rustfs | 60.53 MB/s | 978.8us | 1.1ms | 1.5ms | 0 |
| minio | 57.85 MB/s | 1.0ms | 1.2ms | 1.4ms | 0 |
| seaweedfs | 48.16 MB/s | 1.2ms | 1.5ms | 2.5ms | 0 |
| localstack | 46.35 MB/s | 1.3ms | 1.7ms | 2.3ms | 0 |
| liteio | 40.26 MB/s | 1.1ms | 2.6ms | 6.7ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 71.79 MB/s
rustfs       █████████████████████████ 60.53 MB/s
minio        ████████████████████████ 57.85 MB/s
seaweedfs    ████████████████████ 48.16 MB/s
localstack   ███████████████████ 46.35 MB/s
liteio       ████████████████ 40.26 MB/s
```

**Latency (P50)**
```
liteio_mem   ███████████████████ 832.7us
rustfs       ██████████████████████ 978.8us
minio        ████████████████████████ 1.0ms
seaweedfs    ███████████████████████████ 1.2ms
localstack   ██████████████████████████████ 1.3ms
liteio       ████████████████████████ 1.1ms
```

## Resource Usage

| Driver | Memory | RSS | Cache | CPU | Volume | Block I/O |
|--------|--------|-----|-------|-----|--------|----------|
| liteio | 93.13MiB / 7.653GiB | 93.1 MB | - | 0.0% | 1703.9 MB | 4.1kB / 2.23GB |
| liteio_mem | 68.36MiB / 7.653GiB | 68.4 MB | - | 0.0% | 4739.1 MB | 643kB / 2.23GB |
| localstack | 387MiB / 7.653GiB | 387.0 MB | - | 0.1% | 0.0 MB | 10.5MB / 1.65GB |
| minio | 406.9MiB / 7.653GiB | 406.9 MB | - | 3.2% | 5368.8 MB | 6.78MB / 2GB |
| rustfs | 701.5MiB / 7.653GiB | 701.5 MB | - | 0.1% | 4413.4 MB | 2.35MB / 1.77GB |
| seaweedfs | 131.8MiB / 7.653GiB | 131.8 MB | - | 1.0% | (no data) | 864kB / 0B |

> **Note:** RSS = actual application memory. Cache = OS page cache (reclaimable).

## Recommendations

- **Write-heavy workloads:** rustfs
- **Read-heavy workloads:** minio

---

*Generated by storage benchmark CLI*
