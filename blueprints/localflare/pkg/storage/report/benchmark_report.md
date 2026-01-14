# Storage Benchmark Report

**Generated:** 2026-01-15T02:41:42+07:00

**Go Version:** go1.25.5

**Platform:** darwin/arm64

## Executive Summary

### Performance Leaders

```
┌───────────────────────────┬───────────────────────┬───────────────────────────────┐
│         Category          │        Leader         │             Notes             │
├───────────────────────────┼───────────────────────┼───────────────────────────────┤
│ Small File Read (1KB)     │ liteio 4.8 MB/s       │ 20% faster than liteio_mem    │
├───────────────────────────┼───────────────────────┼───────────────────────────────┤
│ Small File Write (1KB)    │ rustfs 1.4 MB/s       │ Close competition             │
├───────────────────────────┼───────────────────────┼───────────────────────────────┤
│ Delete Operations         │ liteio 6281 ops/s     │ Close competition             │
├───────────────────────────┼───────────────────────┼───────────────────────────────┤
│ Stat Operations           │ liteio_mem 6293 ops/s │ 30% faster than liteio        │
├───────────────────────────┼───────────────────────┼───────────────────────────────┤
│ List Operations (100 obj) │ liteio 1343 ops/s     │ Close competition             │
├───────────────────────────┼───────────────────────┼───────────────────────────────┤
│ Copy Operations           │ liteio 1.6 MB/s       │ Close competition             │
├───────────────────────────┼───────────────────────┼───────────────────────────────┤
│ Range Reads               │ liteio 252.5 MB/s     │ Close competition             │
├───────────────────────────┼───────────────────────┼───────────────────────────────┤
│ Mixed Workload            │ rustfs 9.8 MB/s       │ Close competition             │
└───────────────────────────┴───────────────────────┴───────────────────────────────┘
```

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (10MB+) | **liteio_mem** | 191 MB/s | Best for media, backups |
| Large File Downloads (10MB) | **liteio_mem** | 319 MB/s | Best for streaming, CDN |
| Small File Operations | **liteio** | 2856 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **minio** | - | Best for multi-user apps |

### Large File Performance (10MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| liteio | 186.4 | 303.0 | 52.9ms | 32.0ms |
| liteio_mem | 190.5 | 319.0 | 52.5ms | 31.1ms |
| localstack | 149.6 | 288.3 | 65.8ms | 34.0ms |
| minio | 162.4 | 310.0 | 58.7ms | 32.1ms |
| rustfs | 177.2 | 269.4 | 59.5ms | 35.8ms |
| seaweedfs | 151.0 | 294.0 | 67.3ms | 34.1ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| liteio | 816 | 4897 | 1.1ms | 207.0us |
| liteio_mem | 107 | 4076 | 3.6ms | 219.0us |
| localstack | 1220 | 1212 | 801.0us | 730.2us |
| minio | 1113 | 3388 | 888.7us | 287.0us |
| rustfs | 1417 | 2157 | 660.7us | 472.8us |
| seaweedfs | 1392 | 2385 | 684.3us | 414.5us |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| liteio | 4846 | 1343 | 6281 |
| liteio_mem | 6293 | 1268 | 6217 |
| localstack | 1479 | 328 | 1680 |
| minio | 4569 | 626 | 2819 |
| rustfs | 3094 | 166 | 1184 |
| seaweedfs | 3552 | 601 | 2808 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| liteio | 1.38 | 0.24 | 0.16 |
| liteio_mem | 1.50 | 0.26 | 0.25 |
| localstack | 1.18 | 0.17 | 0.10 |
| minio | 1.08 | 0.31 | 0.14 |
| rustfs | 1.21 | 0.33 | - |
| seaweedfs | 1.28 | 0.34 | 0.20 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| liteio | 4.73 | 0.89 | 0.69 |
| liteio_mem | 3.80 | 0.86 | 0.77 |
| localstack | 1.28 | 0.17 | 0.08 |
| minio | 3.16 | 0.98 | 0.64 |
| rustfs | 1.88 | 0.79 | - |
| seaweedfs | 1.99 | 0.74 | 0.41 |

*\* indicates errors occurred*

### File Count Performance

Performance with varying numbers of files (1KB each).

**Write N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|------|
| liteio | 908.2us | 7.1ms | 60.4ms | 612.7ms | 6.32s |
| liteio_mem | 734.2us | 6.4ms | 61.8ms | 615.5ms | 6.32s |
| localstack | 798.0us | 8.5ms | 76.4ms | 779.9ms | 8.00s |
| minio | 928.3us | 8.2ms | 91.8ms | 846.4ms | 8.85s |
| rustfs | 788.9us | 7.0ms | 68.6ms | 681.0ms | 6.90s |
| seaweedfs | 884.6us | 7.2ms | 69.5ms | 691.0ms | 7.25s |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|------|
| liteio | 333.7us | 405.8us | 879.5us | 5.5ms | 197.2ms |
| liteio_mem | 231.3us | 327.0us | 787.7us | 5.9ms | 195.2ms |
| localstack | 1.1ms | 1.5ms | 3.4ms | 23.8ms | 237.1ms |
| minio | 600.7us | 719.8us | 1.8ms | 13.8ms | 166.4ms |
| rustfs | 1.0ms | 1.5ms | 6.9ms | 57.2ms | 720.1ms |
| seaweedfs | 687.6us | 701.2us | 1.4ms | 8.7ms | 96.7ms |

*\* indicates errors occurred*

### Skipped Benchmarks

Some benchmarks were skipped due to driver limitations:

- **rustfs**: 2 skipped
  - ParallelWrite/1KB/C50 (exceeds max concurrency 10)
  - ParallelRead/1KB/C50 (exceeds max concurrency 10)

---

## Configuration

| Parameter | Value |
|-----------|-------|
| Iterations | 20 |
| Warmup | 5 |
| Concurrency | 200 |
| Timeout | 1m0s |

## Drivers Tested

- liteio (43 benchmarks)
- liteio_mem (43 benchmarks)
- localstack (43 benchmarks)
- minio (43 benchmarks)
- rustfs (41 benchmarks)
- seaweedfs (43 benchmarks)

## Performance Comparison

### Copy/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.58 MB/s | 578.8us | 777.1us | 777.1us | 0 |
| liteio_mem | 1.55 MB/s | 615.3us | 772.5us | 772.5us | 0 |
| localstack | 1.24 MB/s | 761.4us | 884.1us | 884.1us | 0 |
| minio | 1.07 MB/s | 891.1us | 1.1ms | 1.1ms | 0 |
| rustfs | 0.95 MB/s | 1.0ms | 1.1ms | 1.1ms | 0 |
| seaweedfs | 0.88 MB/s | 1.0ms | 1.2ms | 1.2ms | 0 |

```
  liteio       ████████████████████████████████████████ 1.58 MB/s
  liteio_mem   ███████████████████████████████████████ 1.55 MB/s
  localstack   ███████████████████████████████ 1.24 MB/s
  minio        ██████████████████████████ 1.07 MB/s
  rustfs       ███████████████████████ 0.95 MB/s
  seaweedfs    ██████████████████████ 0.88 MB/s
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 6281 ops/s | 147.8us | 172.5us | 172.5us | 0 |
| liteio_mem | 6217 ops/s | 154.8us | 178.7us | 178.7us | 0 |
| minio | 2819 ops/s | 319.9us | 580.8us | 580.8us | 0 |
| seaweedfs | 2808 ops/s | 342.3us | 422.4us | 422.4us | 0 |
| localstack | 1680 ops/s | 587.3us | 653.1us | 653.1us | 0 |
| rustfs | 1184 ops/s | 594.1us | 1.8ms | 1.8ms | 0 |

```
  liteio       ████████████████████████████████████████ 6281 ops/s
  liteio_mem   ███████████████████████████████████████ 6217 ops/s
  minio        █████████████████ 2819 ops/s
  seaweedfs    █████████████████ 2808 ops/s
  localstack   ██████████ 1680 ops/s
  rustfs       ███████ 1184 ops/s
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 0.16 MB/s | 594.9us | 626.5us | 626.5us | 0 |
| liteio | 0.14 MB/s | 618.3us | 759.3us | 759.3us | 0 |
| rustfs | 0.13 MB/s | 685.1us | 783.4us | 783.4us | 0 |
| seaweedfs | 0.12 MB/s | 685.8us | 761.5us | 761.5us | 0 |
| localstack | 0.12 MB/s | 779.2us | 878.1us | 878.1us | 0 |
| minio | 0.11 MB/s | 813.5us | 967.5us | 967.5us | 0 |

```
  liteio_mem   ████████████████████████████████████████ 0.16 MB/s
  liteio       ██████████████████████████████████ 0.14 MB/s
  rustfs       ████████████████████████████████ 0.13 MB/s
  seaweedfs    ██████████████████████████████ 0.12 MB/s
  localstack   ██████████████████████████████ 0.12 MB/s
  minio        ███████████████████████████ 0.11 MB/s
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 2530 ops/s | 387.9us | 420.2us | 420.2us | 0 |
| liteio | 1584 ops/s | 575.5us | 806.3us | 806.3us | 0 |
| liteio_mem | 1280 ops/s | 665.9us | 999.2us | 999.2us | 0 |
| rustfs | 1262 ops/s | 743.8us | 865.8us | 865.8us | 0 |
| localstack | 1218 ops/s | 767.9us | 905.5us | 905.5us | 0 |
| minio | 1099 ops/s | 911.7us | 997.9us | 997.9us | 0 |

```
  seaweedfs    ████████████████████████████████████████ 2530 ops/s
  liteio       █████████████████████████ 1584 ops/s
  liteio_mem   ████████████████████ 1280 ops/s
  rustfs       ███████████████████ 1262 ops/s
  localstack   ███████████████████ 1218 ops/s
  minio        █████████████████ 1099 ops/s
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.16 MB/s | 598.8us | 654.3us | 654.3us | 0 |
| liteio_mem | 0.14 MB/s | 644.0us | 709.9us | 709.9us | 0 |
| rustfs | 0.13 MB/s | 705.5us | 771.2us | 771.2us | 0 |
| localstack | 0.12 MB/s | 738.5us | 823.8us | 823.8us | 0 |
| seaweedfs | 0.12 MB/s | 771.7us | 810.8us | 810.8us | 0 |
| minio | 0.11 MB/s | 856.2us | 950.2us | 950.2us | 0 |

```
  liteio       ████████████████████████████████████████ 0.16 MB/s
  liteio_mem   ████████████████████████████████████ 0.14 MB/s
  rustfs       █████████████████████████████████ 0.13 MB/s
  localstack   ███████████████████████████████ 0.12 MB/s
  seaweedfs    ██████████████████████████████ 0.12 MB/s
  minio        ███████████████████████████ 0.11 MB/s
```

### FileCount/Delete/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 4631 ops/s | 216.0us | 216.0us | 216.0us | 0 |
| liteio | 3113 ops/s | 321.2us | 321.2us | 321.2us | 0 |
| seaweedfs | 2595 ops/s | 385.4us | 385.4us | 385.4us | 0 |
| minio | 2592 ops/s | 385.9us | 385.9us | 385.9us | 0 |
| localstack | 1564 ops/s | 639.3us | 639.3us | 639.3us | 0 |
| rustfs | 535 ops/s | 1.9ms | 1.9ms | 1.9ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 4631 ops/s
  liteio       ██████████████████████████ 3113 ops/s
  seaweedfs    ██████████████████████ 2595 ops/s
  minio        ██████████████████████ 2592 ops/s
  localstack   █████████████ 1564 ops/s
  rustfs       ████ 535 ops/s
```

### FileCount/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 580 ops/s | 1.7ms | 1.7ms | 1.7ms | 0 |
| liteio | 480 ops/s | 2.1ms | 2.1ms | 2.1ms | 0 |
| seaweedfs | 309 ops/s | 3.2ms | 3.2ms | 3.2ms | 0 |
| minio | 299 ops/s | 3.3ms | 3.3ms | 3.3ms | 0 |
| localstack | 162 ops/s | 6.2ms | 6.2ms | 6.2ms | 0 |
| rustfs | 120 ops/s | 8.3ms | 8.3ms | 8.3ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 580 ops/s
  liteio       █████████████████████████████████ 480 ops/s
  seaweedfs    █████████████████████ 309 ops/s
  minio        ████████████████████ 299 ops/s
  localstack   ███████████ 162 ops/s
  rustfs       ████████ 120 ops/s
```

### FileCount/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 52 ops/s | 19.4ms | 19.4ms | 19.4ms | 0 |
| liteio_mem | 51 ops/s | 19.5ms | 19.5ms | 19.5ms | 0 |
| seaweedfs | 31 ops/s | 32.0ms | 32.0ms | 32.0ms | 0 |
| minio | 29 ops/s | 34.5ms | 34.5ms | 34.5ms | 0 |
| localstack | 17 ops/s | 59.4ms | 59.4ms | 59.4ms | 0 |
| rustfs | 11 ops/s | 92.3ms | 92.3ms | 92.3ms | 0 |

```
  liteio       ████████████████████████████████████████ 52 ops/s
  liteio_mem   ███████████████████████████████████████ 51 ops/s
  seaweedfs    ████████████████████████ 31 ops/s
  minio        ██████████████████████ 29 ops/s
  localstack   █████████████ 17 ops/s
  rustfs       ████████ 11 ops/s
```

### FileCount/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 5 ops/s | 182.9ms | 182.9ms | 182.9ms | 0 |
| liteio_mem | 5 ops/s | 183.9ms | 183.9ms | 183.9ms | 0 |
| seaweedfs | 3 ops/s | 332.4ms | 332.4ms | 332.4ms | 0 |
| minio | 3 ops/s | 344.6ms | 344.6ms | 344.6ms | 0 |
| localstack | 2 ops/s | 608.0ms | 608.0ms | 608.0ms | 0 |
| rustfs | 1 ops/s | 796.2ms | 796.2ms | 796.2ms | 0 |

```
  liteio       ████████████████████████████████████████ 5 ops/s
  liteio_mem   ███████████████████████████████████████ 5 ops/s
  seaweedfs    ██████████████████████ 3 ops/s
  minio        █████████████████████ 3 ops/s
  localstack   ████████████ 2 ops/s
  rustfs       █████████ 1 ops/s
```

### FileCount/Delete/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1 ops/s | 1.85s | 1.85s | 1.85s | 0 |
| liteio_mem | 1 ops/s | 1.89s | 1.89s | 1.89s | 0 |
| seaweedfs | 0 ops/s | 3.32s | 3.32s | 3.32s | 0 |
| minio | 0 ops/s | 3.52s | 3.52s | 3.52s | 0 |
| localstack | 0 ops/s | 6.15s | 6.15s | 6.15s | 0 |
| rustfs | 0 ops/s | 8.14s | 8.14s | 8.14s | 0 |

```
  liteio       ████████████████████████████████████████ 1 ops/s
  liteio_mem   ██████████████████████████████████████ 1 ops/s
  seaweedfs    ██████████████████████ 0 ops/s
  minio        ████████████████████ 0 ops/s
  localstack   ████████████ 0 ops/s
  rustfs       █████████ 0 ops/s
```

### FileCount/List/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 4323 ops/s | 231.3us | 231.3us | 231.3us | 0 |
| liteio | 2997 ops/s | 333.7us | 333.7us | 333.7us | 0 |
| minio | 1665 ops/s | 600.7us | 600.7us | 600.7us | 0 |
| seaweedfs | 1454 ops/s | 687.6us | 687.6us | 687.6us | 0 |
| rustfs | 996 ops/s | 1.0ms | 1.0ms | 1.0ms | 0 |
| localstack | 938 ops/s | 1.1ms | 1.1ms | 1.1ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 4323 ops/s
  liteio       ███████████████████████████ 2997 ops/s
  minio        ███████████████ 1665 ops/s
  seaweedfs    █████████████ 1454 ops/s
  rustfs       █████████ 996 ops/s
  localstack   ████████ 938 ops/s
```

### FileCount/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 3058 ops/s | 327.0us | 327.0us | 327.0us | 0 |
| liteio | 2464 ops/s | 405.8us | 405.8us | 405.8us | 0 |
| seaweedfs | 1426 ops/s | 701.2us | 701.2us | 701.2us | 0 |
| minio | 1389 ops/s | 719.8us | 719.8us | 719.8us | 0 |
| rustfs | 682 ops/s | 1.5ms | 1.5ms | 1.5ms | 0 |
| localstack | 653 ops/s | 1.5ms | 1.5ms | 1.5ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 3058 ops/s
  liteio       ████████████████████████████████ 2464 ops/s
  seaweedfs    ██████████████████ 1426 ops/s
  minio        ██████████████████ 1389 ops/s
  rustfs       ████████ 682 ops/s
  localstack   ████████ 653 ops/s
```

### FileCount/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1270 ops/s | 787.7us | 787.7us | 787.7us | 0 |
| liteio | 1137 ops/s | 879.5us | 879.5us | 879.5us | 0 |
| seaweedfs | 706 ops/s | 1.4ms | 1.4ms | 1.4ms | 0 |
| minio | 564 ops/s | 1.8ms | 1.8ms | 1.8ms | 0 |
| localstack | 293 ops/s | 3.4ms | 3.4ms | 3.4ms | 0 |
| rustfs | 146 ops/s | 6.9ms | 6.9ms | 6.9ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 1270 ops/s
  liteio       ███████████████████████████████████ 1137 ops/s
  seaweedfs    ██████████████████████ 706 ops/s
  minio        █████████████████ 564 ops/s
  localstack   █████████ 293 ops/s
  rustfs       ████ 146 ops/s
```

### FileCount/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 181 ops/s | 5.5ms | 5.5ms | 5.5ms | 0 |
| liteio_mem | 171 ops/s | 5.9ms | 5.9ms | 5.9ms | 0 |
| seaweedfs | 115 ops/s | 8.7ms | 8.7ms | 8.7ms | 0 |
| minio | 72 ops/s | 13.8ms | 13.8ms | 13.8ms | 0 |
| localstack | 42 ops/s | 23.8ms | 23.8ms | 23.8ms | 0 |
| rustfs | 17 ops/s | 57.2ms | 57.2ms | 57.2ms | 0 |

```
  liteio       ████████████████████████████████████████ 181 ops/s
  liteio_mem   █████████████████████████████████████ 171 ops/s
  seaweedfs    █████████████████████████ 115 ops/s
  minio        ███████████████ 72 ops/s
  localstack   █████████ 42 ops/s
  rustfs       ███ 17 ops/s
```

### FileCount/List/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 10 ops/s | 96.7ms | 96.7ms | 96.7ms | 0 |
| minio | 6 ops/s | 166.4ms | 166.4ms | 166.4ms | 0 |
| liteio_mem | 5 ops/s | 195.2ms | 195.2ms | 195.2ms | 0 |
| liteio | 5 ops/s | 197.2ms | 197.2ms | 197.2ms | 0 |
| localstack | 4 ops/s | 237.1ms | 237.1ms | 237.1ms | 0 |
| rustfs | 1 ops/s | 720.1ms | 720.1ms | 720.1ms | 0 |

```
  seaweedfs    ████████████████████████████████████████ 10 ops/s
  minio        ███████████████████████ 6 ops/s
  liteio_mem   ███████████████████ 5 ops/s
  liteio       ███████████████████ 5 ops/s
  localstack   ████████████████ 4 ops/s
  rustfs       █████ 1 ops/s
```

### FileCount/Write/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1.33 MB/s | 734.2us | 734.2us | 734.2us | 0 |
| rustfs | 1.24 MB/s | 788.9us | 788.9us | 788.9us | 0 |
| localstack | 1.22 MB/s | 798.0us | 798.0us | 798.0us | 0 |
| seaweedfs | 1.10 MB/s | 884.6us | 884.6us | 884.6us | 0 |
| liteio | 1.08 MB/s | 908.2us | 908.2us | 908.2us | 0 |
| minio | 1.05 MB/s | 928.3us | 928.3us | 928.3us | 0 |

```
  liteio_mem   ████████████████████████████████████████ 1.33 MB/s
  rustfs       █████████████████████████████████████ 1.24 MB/s
  localstack   ████████████████████████████████████ 1.22 MB/s
  seaweedfs    █████████████████████████████████ 1.10 MB/s
  liteio       ████████████████████████████████ 1.08 MB/s
  minio        ███████████████████████████████ 1.05 MB/s
```

### FileCount/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1.52 MB/s | 6.4ms | 6.4ms | 6.4ms | 0 |
| rustfs | 1.40 MB/s | 7.0ms | 7.0ms | 7.0ms | 0 |
| liteio | 1.37 MB/s | 7.1ms | 7.1ms | 7.1ms | 0 |
| seaweedfs | 1.36 MB/s | 7.2ms | 7.2ms | 7.2ms | 0 |
| minio | 1.19 MB/s | 8.2ms | 8.2ms | 8.2ms | 0 |
| localstack | 1.14 MB/s | 8.5ms | 8.5ms | 8.5ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 1.52 MB/s
  rustfs       ████████████████████████████████████ 1.40 MB/s
  liteio       ████████████████████████████████████ 1.37 MB/s
  seaweedfs    ███████████████████████████████████ 1.36 MB/s
  minio        ███████████████████████████████ 1.19 MB/s
  localstack   ██████████████████████████████ 1.14 MB/s
```

### FileCount/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.62 MB/s | 60.4ms | 60.4ms | 60.4ms | 0 |
| liteio_mem | 1.58 MB/s | 61.8ms | 61.8ms | 61.8ms | 0 |
| rustfs | 1.42 MB/s | 68.6ms | 68.6ms | 68.6ms | 0 |
| seaweedfs | 1.41 MB/s | 69.5ms | 69.5ms | 69.5ms | 0 |
| localstack | 1.28 MB/s | 76.4ms | 76.4ms | 76.4ms | 0 |
| minio | 1.06 MB/s | 91.8ms | 91.8ms | 91.8ms | 0 |

```
  liteio       ████████████████████████████████████████ 1.62 MB/s
  liteio_mem   ███████████████████████████████████████ 1.58 MB/s
  rustfs       ███████████████████████████████████ 1.42 MB/s
  seaweedfs    ██████████████████████████████████ 1.41 MB/s
  localstack   ███████████████████████████████ 1.28 MB/s
  minio        ██████████████████████████ 1.06 MB/s
```

### FileCount/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.59 MB/s | 612.7ms | 612.7ms | 612.7ms | 0 |
| liteio_mem | 1.59 MB/s | 615.5ms | 615.5ms | 615.5ms | 0 |
| rustfs | 1.43 MB/s | 681.0ms | 681.0ms | 681.0ms | 0 |
| seaweedfs | 1.41 MB/s | 691.0ms | 691.0ms | 691.0ms | 0 |
| localstack | 1.25 MB/s | 779.9ms | 779.9ms | 779.9ms | 0 |
| minio | 1.15 MB/s | 846.4ms | 846.4ms | 846.4ms | 0 |

```
  liteio       ████████████████████████████████████████ 1.59 MB/s
  liteio_mem   ███████████████████████████████████████ 1.59 MB/s
  rustfs       ███████████████████████████████████ 1.43 MB/s
  seaweedfs    ███████████████████████████████████ 1.41 MB/s
  localstack   ███████████████████████████████ 1.25 MB/s
  minio        ████████████████████████████ 1.15 MB/s
```

### FileCount/Write/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.55 MB/s | 6.32s | 6.32s | 6.32s | 0 |
| liteio_mem | 1.54 MB/s | 6.32s | 6.32s | 6.32s | 0 |
| rustfs | 1.42 MB/s | 6.90s | 6.90s | 6.90s | 0 |
| seaweedfs | 1.35 MB/s | 7.25s | 7.25s | 7.25s | 0 |
| localstack | 1.22 MB/s | 8.00s | 8.00s | 8.00s | 0 |
| minio | 1.10 MB/s | 8.85s | 8.85s | 8.85s | 0 |

```
  liteio       ████████████████████████████████████████ 1.55 MB/s
  liteio_mem   ███████████████████████████████████████ 1.54 MB/s
  rustfs       ████████████████████████████████████ 1.42 MB/s
  seaweedfs    ██████████████████████████████████ 1.35 MB/s
  localstack   ███████████████████████████████ 1.22 MB/s
  minio        ████████████████████████████ 1.10 MB/s
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1343 ops/s | 695.5us | 983.5us | 983.5us | 0 |
| liteio_mem | 1268 ops/s | 727.2us | 1.1ms | 1.1ms | 0 |
| minio | 626 ops/s | 1.6ms | 1.7ms | 1.7ms | 0 |
| seaweedfs | 601 ops/s | 1.6ms | 1.9ms | 1.9ms | 0 |
| localstack | 328 ops/s | 3.0ms | 3.3ms | 3.3ms | 0 |
| rustfs | 166 ops/s | 6.1ms | 6.5ms | 6.5ms | 0 |

```
  liteio       ████████████████████████████████████████ 1343 ops/s
  liteio_mem   █████████████████████████████████████ 1268 ops/s
  minio        ██████████████████ 626 ops/s
  seaweedfs    █████████████████ 601 ops/s
  localstack   █████████ 328 ops/s
  rustfs       ████ 166 ops/s
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 9.84 MB/s | 1.6ms | 2.0ms | 2.0ms | 0 |
| liteio | 9.54 MB/s | 1.7ms | 2.3ms | 2.3ms | 0 |
| liteio_mem | 6.84 MB/s | 2.2ms | 3.3ms | 3.3ms | 0 |
| seaweedfs | 6.09 MB/s | 2.5ms | 3.2ms | 3.2ms | 0 |
| minio | 6.02 MB/s | 2.7ms | 3.3ms | 3.3ms | 0 |
| localstack | 1.50 MB/s | 11.3ms | 12.7ms | 12.7ms | 0 |

```
  rustfs       ████████████████████████████████████████ 9.84 MB/s
  liteio       ██████████████████████████████████████ 9.54 MB/s
  liteio_mem   ███████████████████████████ 6.84 MB/s
  seaweedfs    ████████████████████████ 6.09 MB/s
  minio        ████████████████████████ 6.02 MB/s
  localstack   ██████ 1.50 MB/s
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 8.47 MB/s | 1.7ms | 2.4ms | 2.4ms | 0 |
| liteio | 8.06 MB/s | 2.1ms | 2.5ms | 2.5ms | 0 |
| minio | 5.49 MB/s | 2.9ms | 3.1ms | 3.1ms | 0 |
| liteio_mem | 4.79 MB/s | 3.7ms | 3.9ms | 3.9ms | 0 |
| seaweedfs | 4.70 MB/s | 3.5ms | 3.8ms | 3.8ms | 0 |
| localstack | 1.43 MB/s | 10.1ms | 13.0ms | 13.0ms | 0 |

```
  rustfs       ████████████████████████████████████████ 8.47 MB/s
  liteio       ██████████████████████████████████████ 8.06 MB/s
  minio        █████████████████████████ 5.49 MB/s
  liteio_mem   ██████████████████████ 4.79 MB/s
  seaweedfs    ██████████████████████ 4.70 MB/s
  localstack   ██████ 1.43 MB/s
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 7.60 MB/s | 2.0ms | 2.7ms | 2.7ms | 0 |
| seaweedfs | 5.49 MB/s | 2.9ms | 3.9ms | 3.9ms | 0 |
| liteio | 5.48 MB/s | 3.0ms | 3.8ms | 3.8ms | 0 |
| liteio_mem | 5.35 MB/s | 3.3ms | 3.9ms | 3.9ms | 0 |
| minio | 4.35 MB/s | 4.2ms | 5.2ms | 5.2ms | 0 |
| localstack | 1.33 MB/s | 12.2ms | 13.3ms | 13.3ms | 0 |

```
  rustfs       ████████████████████████████████████████ 7.60 MB/s
  seaweedfs    ████████████████████████████ 5.49 MB/s
  liteio       ████████████████████████████ 5.48 MB/s
  liteio_mem   ████████████████████████████ 5.35 MB/s
  minio        ██████████████████████ 4.35 MB/s
  localstack   ███████ 1.33 MB/s
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 168.81 MB/s | 87.9ms | 88.5ms | 88.5ms | 0 |
| minio | 167.82 MB/s | 87.6ms | 90.2ms | 90.2ms | 0 |
| liteio | 164.07 MB/s | 90.7ms | 92.1ms | 92.1ms | 0 |
| rustfs | 154.04 MB/s | 86.4ms | 110.4ms | 110.4ms | 0 |
| seaweedfs | 131.40 MB/s | 114.6ms | 115.3ms | 115.3ms | 0 |
| localstack | 126.86 MB/s | 112.8ms | 117.8ms | 117.8ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 168.81 MB/s
  minio        ███████████████████████████████████████ 167.82 MB/s
  liteio       ██████████████████████████████████████ 164.07 MB/s
  rustfs       ████████████████████████████████████ 154.04 MB/s
  seaweedfs    ███████████████████████████████ 131.40 MB/s
  localstack   ██████████████████████████████ 126.86 MB/s
```

### ParallelRead/1KB/C1

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 4.73 MB/s | 205.4us | 228.7us | 196.0us | 228.8us | 228.8us | 0 |
| liteio_mem | 3.80 MB/s | 248.8us | 305.8us | 219.2us | 305.9us | 305.9us | 0 |
| minio | 3.16 MB/s | 309.4us | 391.1us | 298.9us | 391.3us | 391.3us | 0 |
| seaweedfs | 1.99 MB/s | 490.8us | 543.5us | 484.3us | 543.5us | 543.5us | 0 |
| rustfs | 1.88 MB/s | 520.4us | 558.4us | 500.1us | 558.5us | 558.5us | 0 |
| localstack | 1.28 MB/s | 765.2us | 902.6us | 718.0us | 903.0us | 903.0us | 0 |

```
  liteio       ████████████████████████████████████████ 4.73 MB/s
  liteio_mem   ████████████████████████████████ 3.80 MB/s
  minio        ██████████████████████████ 3.16 MB/s
  seaweedfs    ████████████████ 1.99 MB/s
  rustfs       ███████████████ 1.88 MB/s
  localstack   ██████████ 1.28 MB/s
```

### ParallelRead/1KB/C10

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 0.98 MB/s | 992.8us | 1.7ms | 779.0us | 1.7ms | 1.7ms | 0 |
| liteio | 0.89 MB/s | 1.1ms | 1.5ms | 1.1ms | 1.5ms | 1.5ms | 0 |
| liteio_mem | 0.86 MB/s | 1.1ms | 1.4ms | 1.1ms | 1.4ms | 1.4ms | 0 |
| rustfs | 0.79 MB/s | 1.2ms | 1.6ms | 1.2ms | 1.6ms | 1.6ms | 0 |
| seaweedfs | 0.74 MB/s | 1.3ms | 2.1ms | 1.0ms | 2.1ms | 2.1ms | 0 |
| localstack | 0.17 MB/s | 5.8ms | 8.9ms | 4.3ms | 8.9ms | 8.9ms | 0 |

```
  minio        ████████████████████████████████████████ 0.98 MB/s
  liteio       ████████████████████████████████████ 0.89 MB/s
  liteio_mem   ██████████████████████████████████ 0.86 MB/s
  rustfs       ████████████████████████████████ 0.79 MB/s
  seaweedfs    ██████████████████████████████ 0.74 MB/s
  localstack   ██████ 0.17 MB/s
```

### ParallelRead/1KB/C50

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio_mem | 0.77 MB/s | 1.3ms | 1.6ms | 1.2ms | 1.6ms | 1.6ms | 0 |
| liteio | 0.69 MB/s | 1.4ms | 1.9ms | 1.2ms | 1.9ms | 1.9ms | 0 |
| minio | 0.64 MB/s | 1.5ms | 1.7ms | 1.6ms | 1.7ms | 1.7ms | 0 |
| seaweedfs | 0.41 MB/s | 2.4ms | 3.0ms | 2.3ms | 3.0ms | 3.0ms | 0 |
| localstack | 0.08 MB/s | 11.6ms | 12.3ms | 11.5ms | 12.3ms | 12.3ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 0.77 MB/s
  liteio       ████████████████████████████████████ 0.69 MB/s
  minio        █████████████████████████████████ 0.64 MB/s
  seaweedfs    █████████████████████ 0.41 MB/s
  localstack   ████ 0.08 MB/s
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1.50 MB/s | 619.8us | 714.0us | 714.0us | 0 |
| liteio | 1.38 MB/s | 690.9us | 807.4us | 807.4us | 0 |
| seaweedfs | 1.28 MB/s | 725.2us | 999.0us | 999.0us | 0 |
| rustfs | 1.21 MB/s | 788.9us | 921.7us | 921.7us | 0 |
| localstack | 1.18 MB/s | 779.6us | 1.1ms | 1.1ms | 0 |
| minio | 1.08 MB/s | 878.4us | 1.0ms | 1.0ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 1.50 MB/s
  liteio       ████████████████████████████████████ 1.38 MB/s
  seaweedfs    ██████████████████████████████████ 1.28 MB/s
  rustfs       ████████████████████████████████ 1.21 MB/s
  localstack   ███████████████████████████████ 1.18 MB/s
  minio        ████████████████████████████ 1.08 MB/s
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 0.34 MB/s | 2.5ms | 4.4ms | 4.4ms | 0 |
| rustfs | 0.33 MB/s | 2.3ms | 4.8ms | 4.8ms | 0 |
| minio | 0.31 MB/s | 3.0ms | 4.3ms | 4.3ms | 0 |
| liteio_mem | 0.26 MB/s | 3.7ms | 6.5ms | 6.5ms | 0 |
| liteio | 0.24 MB/s | 3.4ms | 7.6ms | 7.6ms | 0 |
| localstack | 0.17 MB/s | 5.1ms | 9.3ms | 9.3ms | 0 |

```
  seaweedfs    ████████████████████████████████████████ 0.34 MB/s
  rustfs       ███████████████████████████████████████ 0.33 MB/s
  minio        ████████████████████████████████████ 0.31 MB/s
  liteio_mem   ██████████████████████████████ 0.26 MB/s
  liteio       █████████████████████████████ 0.24 MB/s
  localstack   ████████████████████ 0.17 MB/s
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 0.25 MB/s | 3.8ms | 4.7ms | 4.7ms | 0 |
| seaweedfs | 0.20 MB/s | 4.3ms | 6.8ms | 6.8ms | 0 |
| liteio | 0.16 MB/s | 6.6ms | 7.5ms | 7.5ms | 0 |
| minio | 0.14 MB/s | 7.4ms | 8.7ms | 8.7ms | 0 |
| localstack | 0.10 MB/s | 11.8ms | 12.7ms | 12.7ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 0.25 MB/s
  seaweedfs    ████████████████████████████████ 0.20 MB/s
  liteio       █████████████████████████ 0.16 MB/s
  minio        █████████████████████ 0.14 MB/s
  localstack   ████████████████ 0.10 MB/s
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 225.11 MB/s | 1.1ms | 1.4ms | 1.4ms | 0 |
| liteio | 211.76 MB/s | 1.1ms | 1.5ms | 1.5ms | 0 |
| seaweedfs | 198.86 MB/s | 1.2ms | 1.3ms | 1.3ms | 0 |
| minio | 181.18 MB/s | 1.4ms | 1.4ms | 1.4ms | 0 |
| localstack | 157.62 MB/s | 1.5ms | 1.6ms | 1.6ms | 0 |
| rustfs | 123.09 MB/s | 2.0ms | 2.3ms | 2.3ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 225.11 MB/s
  liteio       █████████████████████████████████████ 211.76 MB/s
  seaweedfs    ███████████████████████████████████ 198.86 MB/s
  minio        ████████████████████████████████ 181.18 MB/s
  localstack   ████████████████████████████ 157.62 MB/s
  rustfs       █████████████████████ 123.09 MB/s
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 238.64 MB/s | 1.0ms | 1.2ms | 1.2ms | 0 |
| liteio | 231.63 MB/s | 1.0ms | 1.4ms | 1.4ms | 0 |
| seaweedfs | 192.07 MB/s | 1.3ms | 1.4ms | 1.4ms | 0 |
| minio | 177.25 MB/s | 1.4ms | 1.6ms | 1.6ms | 0 |
| localstack | 155.15 MB/s | 1.6ms | 1.8ms | 1.8ms | 0 |
| rustfs | 128.72 MB/s | 1.9ms | 2.0ms | 2.0ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 238.64 MB/s
  liteio       ██████████████████████████████████████ 231.63 MB/s
  seaweedfs    ████████████████████████████████ 192.07 MB/s
  minio        █████████████████████████████ 177.25 MB/s
  localstack   ██████████████████████████ 155.15 MB/s
  rustfs       █████████████████████ 128.72 MB/s
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 252.47 MB/s | 982.2us | 1.1ms | 1.1ms | 0 |
| liteio_mem | 243.97 MB/s | 1.0ms | 1.2ms | 1.2ms | 0 |
| seaweedfs | 182.30 MB/s | 1.4ms | 1.5ms | 1.5ms | 0 |
| minio | 181.33 MB/s | 1.4ms | 1.5ms | 1.5ms | 0 |
| localstack | 158.98 MB/s | 1.5ms | 1.8ms | 1.8ms | 0 |
| rustfs | 95.50 MB/s | 2.0ms | 3.8ms | 3.8ms | 0 |

```
  liteio       ████████████████████████████████████████ 252.47 MB/s
  liteio_mem   ██████████████████████████████████████ 243.97 MB/s
  seaweedfs    ████████████████████████████ 182.30 MB/s
  minio        ████████████████████████████ 181.33 MB/s
  localstack   █████████████████████████ 158.98 MB/s
  rustfs       ███████████████ 95.50 MB/s
```

### Read/10MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio_mem | 319.03 MB/s | 418.1us | 450.9us | 31.1ms | 31.8ms | 31.8ms | 0 |
| minio | 310.02 MB/s | 1.1ms | 1.4ms | 32.1ms | 33.5ms | 33.5ms | 0 |
| liteio | 303.04 MB/s | 439.1us | 525.7us | 32.0ms | 36.4ms | 36.4ms | 0 |
| seaweedfs | 293.96 MB/s | 2.2ms | 2.5ms | 34.1ms | 34.9ms | 34.9ms | 0 |
| localstack | 288.27 MB/s | 1.3ms | 1.4ms | 34.0ms | 37.5ms | 37.5ms | 0 |
| rustfs | 269.36 MB/s | 6.5ms | 9.0ms | 35.8ms | 39.2ms | 39.2ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 319.03 MB/s
  minio        ██████████████████████████████████████ 310.02 MB/s
  liteio       █████████████████████████████████████ 303.04 MB/s
  seaweedfs    ████████████████████████████████████ 293.96 MB/s
  localstack   ████████████████████████████████████ 288.27 MB/s
  rustfs       █████████████████████████████████ 269.36 MB/s
```

### Read/1KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 4.78 MB/s | 204.1us | 232.1us | 207.0us | 232.2us | 232.2us | 0 |
| liteio_mem | 3.98 MB/s | 245.2us | 328.2us | 219.0us | 328.3us | 328.3us | 0 |
| minio | 3.31 MB/s | 295.1us | 340.3us | 287.0us | 340.4us | 340.4us | 0 |
| seaweedfs | 2.33 MB/s | 419.2us | 465.8us | 414.5us | 466.0us | 466.0us | 0 |
| rustfs | 2.11 MB/s | 463.5us | 507.5us | 472.8us | 507.6us | 507.6us | 0 |
| localstack | 1.18 MB/s | 824.7us | 1.3ms | 730.2us | 1.3ms | 1.3ms | 0 |

```
  liteio       ████████████████████████████████████████ 4.78 MB/s
  liteio_mem   █████████████████████████████████ 3.98 MB/s
  minio        ███████████████████████████ 3.31 MB/s
  seaweedfs    ███████████████████ 2.33 MB/s
  rustfs       █████████████████ 2.11 MB/s
  localstack   █████████ 1.18 MB/s
```

### Read/1MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio_mem | 293.85 MB/s | 281.9us | 447.2us | 3.4ms | 3.6ms | 3.6ms | 0 |
| liteio | 291.18 MB/s | 312.9us | 440.4us | 3.4ms | 3.7ms | 3.7ms | 0 |
| minio | 260.29 MB/s | 761.8us | 1.1ms | 3.8ms | 4.1ms | 4.1ms | 0 |
| seaweedfs | 249.67 MB/s | 942.9us | 1.2ms | 4.0ms | 4.2ms | 4.2ms | 0 |
| localstack | 247.12 MB/s | 1.0ms | 1.2ms | 3.9ms | 4.6ms | 4.6ms | 0 |
| rustfs | 206.52 MB/s | 1.8ms | 2.3ms | 4.6ms | 5.6ms | 5.6ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 293.85 MB/s
  liteio       ███████████████████████████████████████ 291.18 MB/s
  minio        ███████████████████████████████████ 260.29 MB/s
  seaweedfs    █████████████████████████████████ 249.67 MB/s
  localstack   █████████████████████████████████ 247.12 MB/s
  rustfs       ████████████████████████████ 206.52 MB/s
```

### Read/64KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 145.52 MB/s | 240.3us | 320.6us | 409.2us | 453.0us | 502.3us | 0 |
| liteio_mem | 124.28 MB/s | 314.2us | 563.8us | 423.5us | 677.2us | 919.8us | 0 |
| minio | 120.48 MB/s | 333.1us | 381.9us | 512.2us | 600.1us | 607.3us | 0 |
| seaweedfs | 94.68 MB/s | 469.7us | 560.7us | 649.0us | 744.3us | 784.8us | 0 |
| rustfs | 76.56 MB/s | 693.7us | 707.6us | 721.3us | 826.2us | 893.2us | 0 |
| localstack | 65.71 MB/s | 838.6us | 1.0ms | 924.0us | 1.1ms | 1.2ms | 0 |

```
  liteio       ████████████████████████████████████████ 145.52 MB/s
  liteio_mem   ██████████████████████████████████ 124.28 MB/s
  minio        █████████████████████████████████ 120.48 MB/s
  seaweedfs    ██████████████████████████ 94.68 MB/s
  rustfs       █████████████████████ 76.56 MB/s
  localstack   ██████████████████ 65.71 MB/s
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 6293 ops/s | 143.6us | 207.7us | 207.7us | 0 |
| liteio | 4846 ops/s | 153.5us | 211.9us | 211.9us | 0 |
| minio | 4569 ops/s | 212.3us | 261.0us | 261.0us | 0 |
| seaweedfs | 3552 ops/s | 263.5us | 353.7us | 353.7us | 0 |
| rustfs | 3094 ops/s | 310.6us | 354.6us | 354.6us | 0 |
| localstack | 1479 ops/s | 658.4us | 733.0us | 733.0us | 0 |

```
  liteio_mem   ████████████████████████████████████████ 6293 ops/s
  liteio       ██████████████████████████████ 4846 ops/s
  minio        █████████████████████████████ 4569 ops/s
  seaweedfs    ██████████████████████ 3552 ops/s
  rustfs       ███████████████████ 3094 ops/s
  localstack   █████████ 1479 ops/s
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 190.50 MB/s | 52.5ms | 54.5ms | 54.5ms | 0 |
| liteio | 186.45 MB/s | 52.9ms | 55.7ms | 55.7ms | 0 |
| rustfs | 177.18 MB/s | 59.5ms | 61.3ms | 61.3ms | 0 |
| minio | 162.35 MB/s | 58.7ms | 67.4ms | 67.4ms | 0 |
| seaweedfs | 151.01 MB/s | 67.3ms | 70.2ms | 70.2ms | 0 |
| localstack | 149.60 MB/s | 65.8ms | 68.7ms | 68.7ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 190.50 MB/s
  liteio       ███████████████████████████████████████ 186.45 MB/s
  rustfs       █████████████████████████████████████ 177.18 MB/s
  minio        ██████████████████████████████████ 162.35 MB/s
  seaweedfs    ███████████████████████████████ 151.01 MB/s
  localstack   ███████████████████████████████ 149.60 MB/s
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 1.38 MB/s | 660.7us | 820.4us | 820.4us | 0 |
| seaweedfs | 1.36 MB/s | 684.3us | 943.8us | 943.8us | 0 |
| localstack | 1.19 MB/s | 801.0us | 921.1us | 921.1us | 0 |
| minio | 1.09 MB/s | 888.7us | 973.3us | 973.3us | 0 |
| liteio | 0.80 MB/s | 1.1ms | 2.1ms | 2.1ms | 0 |
| liteio_mem | 0.10 MB/s | 3.6ms | 18.5ms | 18.5ms | 0 |

```
  rustfs       ████████████████████████████████████████ 1.38 MB/s
  seaweedfs    ███████████████████████████████████████ 1.36 MB/s
  localstack   ██████████████████████████████████ 1.19 MB/s
  minio        ███████████████████████████████ 1.09 MB/s
  liteio       ███████████████████████ 0.80 MB/s
  liteio_mem   ███ 0.10 MB/s
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 177.37 MB/s | 5.4ms | 5.9ms | 5.9ms | 0 |
| liteio_mem | 174.69 MB/s | 5.7ms | 6.0ms | 6.0ms | 0 |
| liteio | 166.68 MB/s | 5.9ms | 6.4ms | 6.4ms | 0 |
| minio | 142.25 MB/s | 6.9ms | 7.6ms | 7.6ms | 0 |
| seaweedfs | 127.47 MB/s | 7.7ms | 8.4ms | 8.4ms | 0 |
| localstack | 127.41 MB/s | 7.6ms | 8.6ms | 8.6ms | 0 |

```
  rustfs       ████████████████████████████████████████ 177.37 MB/s
  liteio_mem   ███████████████████████████████████████ 174.69 MB/s
  liteio       █████████████████████████████████████ 166.68 MB/s
  minio        ████████████████████████████████ 142.25 MB/s
  seaweedfs    ████████████████████████████ 127.47 MB/s
  localstack   ████████████████████████████ 127.41 MB/s
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 66.79 MB/s | 896.5us | 1.2ms | 1.3ms | 0 |
| liteio | 59.24 MB/s | 1.0ms | 1.2ms | 1.4ms | 0 |
| rustfs | 57.10 MB/s | 1.1ms | 1.3ms | 1.4ms | 0 |
| seaweedfs | 54.91 MB/s | 1.1ms | 1.4ms | 1.6ms | 0 |
| localstack | 53.10 MB/s | 1.2ms | 1.3ms | 1.6ms | 0 |
| minio | 48.21 MB/s | 1.2ms | 1.5ms | 1.6ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 66.79 MB/s
  liteio       ███████████████████████████████████ 59.24 MB/s
  rustfs       ██████████████████████████████████ 57.10 MB/s
  seaweedfs    ████████████████████████████████ 54.91 MB/s
  localstack   ███████████████████████████████ 53.10 MB/s
  minio        ████████████████████████████ 48.21 MB/s
```

## Recommendations

- **Best for write-heavy workloads:** liteio_mem
- **Best for read-heavy workloads:** liteio_mem

---

*Report generated by storage benchmark CLI*
