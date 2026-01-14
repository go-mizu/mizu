# Storage Benchmark Report

**Generated:** 2026-01-15T02:56:59+07:00

**Go Version:** go1.25.5

**Platform:** darwin/arm64

## Executive Summary

### Performance Leaders

```
┌───────────────────────────┬───────────────────────┬───────────────────────────────┐
│         Category          │        Leader         │             Notes             │
├───────────────────────────┼───────────────────────┼───────────────────────────────┤
│ Small File Read (1KB)     │ liteio_mem 4.8 MB/s   │ 2.1x faster than seaweedfs    │
├───────────────────────────┼───────────────────────┼───────────────────────────────┤
│ Small File Write (1KB)    │ seaweedfs 1.2 MB/s    │ Close competition             │
├───────────────────────────┼───────────────────────┼───────────────────────────────┤
│ Delete Operations         │ liteio_mem 6130 ops/s │ 2.1x faster than seaweedfs    │
├───────────────────────────┼───────────────────────┼───────────────────────────────┤
│ Stat Operations           │ liteio_mem 4741 ops/s │ 41% faster than seaweedfs     │
├───────────────────────────┼───────────────────────┼───────────────────────────────┤
│ List Operations (100 obj) │ liteio_mem 1237 ops/s │ 1.8x faster than seaweedfs    │
├───────────────────────────┼───────────────────────┼───────────────────────────────┤
│ Copy Operations           │ liteio_mem 1.3 MB/s   │ Close competition             │
├───────────────────────────┼───────────────────────┼───────────────────────────────┤
│ Range Reads               │ seaweedfs 187.5 MB/s  │ Close competition             │
├───────────────────────────┼───────────────────────┼───────────────────────────────┤
│ Mixed Workload            │ rustfs 12.3 MB/s      │ 37% faster than liteio_mem    │
└───────────────────────────┴───────────────────────┴───────────────────────────────┘
```

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (10MB+) | **rustfs** | 189 MB/s | Best for media, backups |
| Large File Downloads (10MB) | **liteio_mem** | 306 MB/s | Best for streaming, CDN |
| Small File Operations | **liteio_mem** | 2889 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **liteio_mem** | - | Best for multi-user apps |

### Large File Performance (10MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| liteio_mem | 187.9 | 306.0 | 53.2ms | 31.3ms |
| localstack | 147.8 | 304.9 | 67.1ms | 32.1ms |
| rustfs | 188.9 | 281.8 | 50.5ms | 35.6ms |
| seaweedfs | 144.7 | 292.5 | 69.7ms | 34.0ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| liteio_mem | 817 | 4960 | 1.1ms | 196.4us |
| localstack | 1217 | 1457 | 807.2us | 677.2us |
| rustfs | 1119 | 2179 | 856.0us | 456.8us |
| seaweedfs | 1230 | 2357 | 741.5us | 410.4us |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| liteio_mem | 4741 | 1237 | 6130 |
| localstack | 1437 | 333 | 1809 |
| rustfs | 3209 | 166 | 1212 |
| seaweedfs | 3365 | 684 | 2963 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| liteio_mem | 1.40 | 0.18 | 0.21 |
| localstack | 1.20 | 0.18 | 0.11 |
| rustfs | 1.26 | 0.32 | - |
| seaweedfs | 1.34 | 0.38 | 0.24 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| liteio_mem | 4.69 | 1.11 | 0.66 |
| localstack | 1.30 | 0.17 | 0.07 |
| rustfs | 1.94 | 0.81 | - |
| seaweedfs | 2.22 | 0.77 | 0.54 |

*\* indicates errors occurred*

### File Count Performance

Performance with varying numbers of files (1KB each).

**Write N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|------|
| liteio_mem | 781.5us | 6.9ms | 62.9ms | 651.8ms | 6.58s |
| localstack | 797.6us | 8.0ms | 76.4ms | 775.4ms | 8.00s |
| rustfs | 828.0us | 7.8ms | 69.2ms | 683.9ms | 6.92s |
| seaweedfs | 759.5us | 6.8ms | 70.6ms | 724.2ms | 7.10s |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|------|
| liteio_mem | 261.9us | 301.6us | 1.1ms | 10.5ms | 226.9ms |
| localstack | 1.5ms | 1.3ms | 4.0ms | 25.8ms | 247.7ms |
| rustfs | 1.2ms | 1.6ms | 6.9ms | 61.4ms | 740.2ms |
| seaweedfs | 690.5us | 763.6us | 2.2ms | 9.5ms | 90.8ms |

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

- liteio_mem (43 benchmarks)
- localstack (43 benchmarks)
- rustfs (41 benchmarks)
- seaweedfs (43 benchmarks)

## Performance Comparison

### Copy/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1.29 MB/s | 631.4us | 1.0ms | 1.0ms | 0 |
| localstack | 1.19 MB/s | 816.0us | 1.0ms | 1.0ms | 0 |
| rustfs | 0.95 MB/s | 976.6us | 1.2ms | 1.2ms | 0 |
| seaweedfs | 0.84 MB/s | 981.5us | 1.9ms | 1.9ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 1.29 MB/s
  localstack   ████████████████████████████████████ 1.19 MB/s
  rustfs       █████████████████████████████ 0.95 MB/s
  seaweedfs    █████████████████████████ 0.84 MB/s
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 6130 ops/s | 149.6us | 182.2us | 182.2us | 0 |
| seaweedfs | 2963 ops/s | 321.8us | 371.7us | 371.7us | 0 |
| localstack | 1809 ops/s | 548.3us | 593.7us | 593.7us | 0 |
| rustfs | 1212 ops/s | 801.5us | 961.3us | 961.3us | 0 |

```
  liteio_mem   ████████████████████████████████████████ 6130 ops/s
  seaweedfs    ███████████████████ 2963 ops/s
  localstack   ███████████ 1809 ops/s
  rustfs       ███████ 1212 ops/s
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 0.15 MB/s | 629.3us | 654.9us | 654.9us | 0 |
| rustfs | 0.15 MB/s | 526.4us | 696.0us | 696.0us | 0 |
| seaweedfs | 0.13 MB/s | 724.1us | 759.1us | 759.1us | 0 |
| localstack | 0.13 MB/s | 721.9us | 789.6us | 789.6us | 0 |

```
  liteio_mem   ████████████████████████████████████████ 0.15 MB/s
  rustfs       ██████████████████████████████████████ 0.15 MB/s
  seaweedfs    ██████████████████████████████████ 0.13 MB/s
  localstack   █████████████████████████████████ 0.13 MB/s
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 2152 ops/s | 456.0us | 508.9us | 508.9us | 0 |
| rustfs | 1297 ops/s | 736.2us | 786.7us | 786.7us | 0 |
| localstack | 1202 ops/s | 772.0us | 902.3us | 902.3us | 0 |
| liteio_mem | 664 ops/s | 957.0us | 2.2ms | 2.2ms | 0 |

```
  seaweedfs    ████████████████████████████████████████ 2152 ops/s
  rustfs       ████████████████████████ 1297 ops/s
  localstack   ██████████████████████ 1202 ops/s
  liteio_mem   ████████████ 664 ops/s
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 0.14 MB/s | 680.8us | 710.6us | 710.6us | 0 |
| localstack | 0.12 MB/s | 788.5us | 932.1us | 932.1us | 0 |
| seaweedfs | 0.12 MB/s | 824.4us | 848.9us | 848.9us | 0 |
| liteio_mem | 0.09 MB/s | 661.6us | 1.2ms | 1.2ms | 0 |

```
  rustfs       ████████████████████████████████████████ 0.14 MB/s
  localstack   █████████████████████████████████ 0.12 MB/s
  seaweedfs    █████████████████████████████████ 0.12 MB/s
  liteio_mem   ███████████████████████████ 0.09 MB/s
```

### FileCount/Delete/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 4628 ops/s | 216.1us | 216.1us | 216.1us | 0 |
| seaweedfs | 2861 ops/s | 349.5us | 349.5us | 349.5us | 0 |
| localstack | 1543 ops/s | 648.1us | 648.1us | 648.1us | 0 |
| rustfs | 1026 ops/s | 974.7us | 974.7us | 974.7us | 0 |

```
  liteio_mem   ████████████████████████████████████████ 4628 ops/s
  seaweedfs    ████████████████████████ 2861 ops/s
  localstack   █████████████ 1543 ops/s
  rustfs       ████████ 1026 ops/s
```

### FileCount/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 494 ops/s | 2.0ms | 2.0ms | 2.0ms | 0 |
| seaweedfs | 274 ops/s | 3.6ms | 3.6ms | 3.6ms | 0 |
| localstack | 163 ops/s | 6.2ms | 6.2ms | 6.2ms | 0 |
| rustfs | 115 ops/s | 8.7ms | 8.7ms | 8.7ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 494 ops/s
  seaweedfs    ██████████████████████ 274 ops/s
  localstack   █████████████ 163 ops/s
  rustfs       █████████ 115 ops/s
```

### FileCount/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 46 ops/s | 22.0ms | 22.0ms | 22.0ms | 0 |
| seaweedfs | 28 ops/s | 35.8ms | 35.8ms | 35.8ms | 0 |
| localstack | 16 ops/s | 61.6ms | 61.6ms | 61.6ms | 0 |
| rustfs | 12 ops/s | 80.5ms | 80.5ms | 80.5ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 46 ops/s
  seaweedfs    ████████████████████████ 28 ops/s
  localstack   ██████████████ 16 ops/s
  rustfs       ██████████ 12 ops/s
```

### FileCount/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 5 ops/s | 190.1ms | 190.1ms | 190.1ms | 0 |
| seaweedfs | 3 ops/s | 331.7ms | 331.7ms | 331.7ms | 0 |
| localstack | 2 ops/s | 613.8ms | 613.8ms | 613.8ms | 0 |
| rustfs | 1 ops/s | 819.1ms | 819.1ms | 819.1ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 5 ops/s
  seaweedfs    ██████████████████████ 3 ops/s
  localstack   ████████████ 2 ops/s
  rustfs       █████████ 1 ops/s
```

### FileCount/Delete/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1 ops/s | 1.86s | 1.86s | 1.86s | 0 |
| seaweedfs | 0 ops/s | 3.17s | 3.17s | 3.17s | 0 |
| localstack | 0 ops/s | 6.19s | 6.19s | 6.19s | 0 |
| rustfs | 0 ops/s | 7.63s | 7.63s | 7.63s | 0 |

```
  liteio_mem   ████████████████████████████████████████ 1 ops/s
  seaweedfs    ███████████████████████ 0 ops/s
  localstack   ████████████ 0 ops/s
  rustfs       █████████ 0 ops/s
```

### FileCount/List/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 3818 ops/s | 261.9us | 261.9us | 261.9us | 0 |
| seaweedfs | 1448 ops/s | 690.5us | 690.5us | 690.5us | 0 |
| rustfs | 841 ops/s | 1.2ms | 1.2ms | 1.2ms | 0 |
| localstack | 658 ops/s | 1.5ms | 1.5ms | 1.5ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 3818 ops/s
  seaweedfs    ███████████████ 1448 ops/s
  rustfs       ████████ 841 ops/s
  localstack   ██████ 658 ops/s
```

### FileCount/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 3315 ops/s | 301.6us | 301.6us | 301.6us | 0 |
| seaweedfs | 1310 ops/s | 763.6us | 763.6us | 763.6us | 0 |
| localstack | 746 ops/s | 1.3ms | 1.3ms | 1.3ms | 0 |
| rustfs | 643 ops/s | 1.6ms | 1.6ms | 1.6ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 3315 ops/s
  seaweedfs    ███████████████ 1310 ops/s
  localstack   █████████ 746 ops/s
  rustfs       ███████ 643 ops/s
```

### FileCount/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 909 ops/s | 1.1ms | 1.1ms | 1.1ms | 0 |
| seaweedfs | 448 ops/s | 2.2ms | 2.2ms | 2.2ms | 0 |
| localstack | 251 ops/s | 4.0ms | 4.0ms | 4.0ms | 0 |
| rustfs | 145 ops/s | 6.9ms | 6.9ms | 6.9ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 909 ops/s
  seaweedfs    ███████████████████ 448 ops/s
  localstack   ███████████ 251 ops/s
  rustfs       ██████ 145 ops/s
```

### FileCount/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 106 ops/s | 9.5ms | 9.5ms | 9.5ms | 0 |
| liteio_mem | 95 ops/s | 10.5ms | 10.5ms | 10.5ms | 0 |
| localstack | 39 ops/s | 25.8ms | 25.8ms | 25.8ms | 0 |
| rustfs | 16 ops/s | 61.4ms | 61.4ms | 61.4ms | 0 |

```
  seaweedfs    ████████████████████████████████████████ 106 ops/s
  liteio_mem   ████████████████████████████████████ 95 ops/s
  localstack   ██████████████ 39 ops/s
  rustfs       ██████ 16 ops/s
```

### FileCount/List/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 11 ops/s | 90.8ms | 90.8ms | 90.8ms | 0 |
| liteio_mem | 4 ops/s | 226.9ms | 226.9ms | 226.9ms | 0 |
| localstack | 4 ops/s | 247.7ms | 247.7ms | 247.7ms | 0 |
| rustfs | 1 ops/s | 740.2ms | 740.2ms | 740.2ms | 0 |

```
  seaweedfs    ████████████████████████████████████████ 11 ops/s
  liteio_mem   ████████████████ 4 ops/s
  localstack   ██████████████ 4 ops/s
  rustfs       ████ 1 ops/s
```

### FileCount/Write/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 1.29 MB/s | 759.5us | 759.5us | 759.5us | 0 |
| liteio_mem | 1.25 MB/s | 781.5us | 781.5us | 781.5us | 0 |
| localstack | 1.22 MB/s | 797.6us | 797.6us | 797.6us | 0 |
| rustfs | 1.18 MB/s | 828.0us | 828.0us | 828.0us | 0 |

```
  seaweedfs    ████████████████████████████████████████ 1.29 MB/s
  liteio_mem   ██████████████████████████████████████ 1.25 MB/s
  localstack   ██████████████████████████████████████ 1.22 MB/s
  rustfs       ████████████████████████████████████ 1.18 MB/s
```

### FileCount/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 1.43 MB/s | 6.8ms | 6.8ms | 6.8ms | 0 |
| liteio_mem | 1.42 MB/s | 6.9ms | 6.9ms | 6.9ms | 0 |
| rustfs | 1.26 MB/s | 7.8ms | 7.8ms | 7.8ms | 0 |
| localstack | 1.22 MB/s | 8.0ms | 8.0ms | 8.0ms | 0 |

```
  seaweedfs    ████████████████████████████████████████ 1.43 MB/s
  liteio_mem   ███████████████████████████████████████ 1.42 MB/s
  rustfs       ███████████████████████████████████ 1.26 MB/s
  localstack   ██████████████████████████████████ 1.22 MB/s
```

### FileCount/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1.55 MB/s | 62.9ms | 62.9ms | 62.9ms | 0 |
| rustfs | 1.41 MB/s | 69.2ms | 69.2ms | 69.2ms | 0 |
| seaweedfs | 1.38 MB/s | 70.6ms | 70.6ms | 70.6ms | 0 |
| localstack | 1.28 MB/s | 76.4ms | 76.4ms | 76.4ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 1.55 MB/s
  rustfs       ████████████████████████████████████ 1.41 MB/s
  seaweedfs    ███████████████████████████████████ 1.38 MB/s
  localstack   ████████████████████████████████ 1.28 MB/s
```

### FileCount/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1.50 MB/s | 651.8ms | 651.8ms | 651.8ms | 0 |
| rustfs | 1.43 MB/s | 683.9ms | 683.9ms | 683.9ms | 0 |
| seaweedfs | 1.35 MB/s | 724.2ms | 724.2ms | 724.2ms | 0 |
| localstack | 1.26 MB/s | 775.4ms | 775.4ms | 775.4ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 1.50 MB/s
  rustfs       ██████████████████████████████████████ 1.43 MB/s
  seaweedfs    ████████████████████████████████████ 1.35 MB/s
  localstack   █████████████████████████████████ 1.26 MB/s
```

### FileCount/Write/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1.48 MB/s | 6.58s | 6.58s | 6.58s | 0 |
| rustfs | 1.41 MB/s | 6.92s | 6.92s | 6.92s | 0 |
| seaweedfs | 1.37 MB/s | 7.10s | 7.10s | 7.10s | 0 |
| localstack | 1.22 MB/s | 8.00s | 8.00s | 8.00s | 0 |

```
  liteio_mem   ████████████████████████████████████████ 1.48 MB/s
  rustfs       ██████████████████████████████████████ 1.41 MB/s
  seaweedfs    █████████████████████████████████████ 1.37 MB/s
  localstack   ████████████████████████████████ 1.22 MB/s
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1237 ops/s | 702.4us | 1.2ms | 1.2ms | 0 |
| seaweedfs | 684 ops/s | 1.4ms | 1.6ms | 1.6ms | 0 |
| localstack | 333 ops/s | 3.0ms | 3.4ms | 3.4ms | 0 |
| rustfs | 166 ops/s | 6.0ms | 6.2ms | 6.2ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 1237 ops/s
  seaweedfs    ██████████████████████ 684 ops/s
  localstack   ██████████ 333 ops/s
  rustfs       █████ 166 ops/s
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 12.27 MB/s | 1.2ms | 1.6ms | 1.6ms | 0 |
| liteio_mem | 8.93 MB/s | 1.7ms | 2.5ms | 2.5ms | 0 |
| seaweedfs | 7.04 MB/s | 2.2ms | 2.9ms | 2.9ms | 0 |
| localstack | 1.44 MB/s | 11.3ms | 11.9ms | 11.9ms | 0 |

```
  rustfs       ████████████████████████████████████████ 12.27 MB/s
  liteio_mem   █████████████████████████████ 8.93 MB/s
  seaweedfs    ██████████████████████ 7.04 MB/s
  localstack   ████ 1.44 MB/s
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 10.61 MB/s | 1.4ms | 1.9ms | 1.9ms | 0 |
| liteio_mem | 5.79 MB/s | 2.7ms | 2.9ms | 2.9ms | 0 |
| seaweedfs | 5.60 MB/s | 3.1ms | 3.2ms | 3.2ms | 0 |
| localstack | 1.55 MB/s | 10.3ms | 13.2ms | 13.2ms | 0 |

```
  rustfs       ████████████████████████████████████████ 10.61 MB/s
  liteio_mem   █████████████████████ 5.79 MB/s
  seaweedfs    █████████████████████ 5.60 MB/s
  localstack   █████ 1.55 MB/s
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 7.58 MB/s | 1.8ms | 2.8ms | 2.8ms | 0 |
| liteio_mem | 5.09 MB/s | 3.5ms | 4.1ms | 4.1ms | 0 |
| seaweedfs | 3.88 MB/s | 3.9ms | 5.5ms | 5.5ms | 0 |
| localstack | 1.31 MB/s | 13.3ms | 13.7ms | 13.7ms | 0 |

```
  rustfs       ████████████████████████████████████████ 7.58 MB/s
  liteio_mem   ██████████████████████████ 5.09 MB/s
  seaweedfs    ████████████████████ 3.88 MB/s
  localstack   ██████ 1.31 MB/s
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 176.64 MB/s | 76.8ms | 93.6ms | 93.6ms | 0 |
| liteio_mem | 159.92 MB/s | 92.3ms | 93.0ms | 93.0ms | 0 |
| seaweedfs | 132.14 MB/s | 108.1ms | 109.0ms | 109.0ms | 0 |
| localstack | 126.59 MB/s | 117.3ms | 119.6ms | 119.6ms | 0 |

```
  rustfs       ████████████████████████████████████████ 176.64 MB/s
  liteio_mem   ████████████████████████████████████ 159.92 MB/s
  seaweedfs    █████████████████████████████ 132.14 MB/s
  localstack   ████████████████████████████ 126.59 MB/s
```

### ParallelRead/1KB/C1

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio_mem | 4.69 MB/s | 205.4us | 292.9us | 198.0us | 293.0us | 293.0us | 0 |
| seaweedfs | 2.22 MB/s | 439.7us | 516.5us | 427.2us | 516.6us | 516.6us | 0 |
| rustfs | 1.94 MB/s | 504.3us | 560.2us | 487.1us | 560.6us | 560.6us | 0 |
| localstack | 1.30 MB/s | 750.4us | 987.5us | 701.1us | 987.5us | 987.5us | 0 |

```
  liteio_mem   ████████████████████████████████████████ 4.69 MB/s
  seaweedfs    ██████████████████ 2.22 MB/s
  rustfs       ████████████████ 1.94 MB/s
  localstack   ███████████ 1.30 MB/s
```

### ParallelRead/1KB/C10

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio_mem | 1.11 MB/s | 875.0us | 1.3ms | 775.7us | 1.3ms | 1.3ms | 0 |
| rustfs | 0.81 MB/s | 1.2ms | 1.6ms | 1.1ms | 1.6ms | 1.6ms | 0 |
| seaweedfs | 0.77 MB/s | 1.3ms | 1.9ms | 1.2ms | 1.9ms | 1.9ms | 0 |
| localstack | 0.17 MB/s | 5.8ms | 7.2ms | 5.9ms | 7.2ms | 7.2ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 1.11 MB/s
  rustfs       █████████████████████████████ 0.81 MB/s
  seaweedfs    ███████████████████████████ 0.77 MB/s
  localstack   ██████ 0.17 MB/s
```

### ParallelRead/1KB/C50

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio_mem | 0.66 MB/s | 1.5ms | 1.8ms | 1.4ms | 1.8ms | 1.8ms | 0 |
| seaweedfs | 0.54 MB/s | 1.8ms | 2.3ms | 1.7ms | 2.3ms | 2.3ms | 0 |
| localstack | 0.07 MB/s | 13.6ms | 14.4ms | 13.6ms | 14.4ms | 14.4ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 0.66 MB/s
  seaweedfs    ████████████████████████████████ 0.54 MB/s
  localstack   ████ 0.07 MB/s
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1.40 MB/s | 663.7us | 882.6us | 882.6us | 0 |
| seaweedfs | 1.34 MB/s | 697.8us | 812.9us | 812.9us | 0 |
| rustfs | 1.26 MB/s | 740.5us | 1.0ms | 1.0ms | 0 |
| localstack | 1.20 MB/s | 772.2us | 1.0ms | 1.0ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 1.40 MB/s
  seaweedfs    ██████████████████████████████████████ 1.34 MB/s
  rustfs       ███████████████████████████████████ 1.26 MB/s
  localstack   ██████████████████████████████████ 1.20 MB/s
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 0.38 MB/s | 1.8ms | 4.2ms | 4.2ms | 0 |
| rustfs | 0.32 MB/s | 2.7ms | 4.7ms | 4.7ms | 0 |
| liteio_mem | 0.18 MB/s | 4.7ms | 9.4ms | 9.4ms | 0 |
| localstack | 0.18 MB/s | 4.6ms | 9.7ms | 9.7ms | 0 |

```
  seaweedfs    ████████████████████████████████████████ 0.38 MB/s
  rustfs       ██████████████████████████████████ 0.32 MB/s
  liteio_mem   ███████████████████ 0.18 MB/s
  localstack   ██████████████████ 0.18 MB/s
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 0.24 MB/s | 4.1ms | 4.7ms | 4.7ms | 0 |
| liteio_mem | 0.21 MB/s | 4.8ms | 5.3ms | 5.3ms | 0 |
| localstack | 0.11 MB/s | 8.4ms | 12.6ms | 12.6ms | 0 |

```
  seaweedfs    ████████████████████████████████████████ 0.24 MB/s
  liteio_mem   ██████████████████████████████████ 0.21 MB/s
  localstack   ██████████████████ 0.11 MB/s
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 211.51 MB/s | 1.1ms | 1.5ms | 1.5ms | 0 |
| seaweedfs | 195.97 MB/s | 1.3ms | 1.4ms | 1.4ms | 0 |
| rustfs | 125.89 MB/s | 1.9ms | 2.4ms | 2.4ms | 0 |
| localstack | 44.29 MB/s | 1.6ms | 5.3ms | 5.3ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 211.51 MB/s
  seaweedfs    █████████████████████████████████████ 195.97 MB/s
  rustfs       ███████████████████████ 125.89 MB/s
  localstack   ████████ 44.29 MB/s
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 215.14 MB/s | 1.0ms | 1.6ms | 1.6ms | 0 |
| seaweedfs | 185.77 MB/s | 1.3ms | 1.7ms | 1.7ms | 0 |
| localstack | 158.49 MB/s | 1.6ms | 1.9ms | 1.9ms | 0 |
| rustfs | 131.21 MB/s | 1.9ms | 2.1ms | 2.1ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 215.14 MB/s
  seaweedfs    ██████████████████████████████████ 185.77 MB/s
  localstack   █████████████████████████████ 158.49 MB/s
  rustfs       ████████████████████████ 131.21 MB/s
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 187.51 MB/s | 1.3ms | 1.6ms | 1.6ms | 0 |
| liteio_mem | 184.80 MB/s | 1.3ms | 1.6ms | 1.6ms | 0 |
| localstack | 156.06 MB/s | 1.5ms | 1.8ms | 1.8ms | 0 |
| rustfs | 128.39 MB/s | 1.9ms | 2.1ms | 2.1ms | 0 |

```
  seaweedfs    ████████████████████████████████████████ 187.51 MB/s
  liteio_mem   ███████████████████████████████████████ 184.80 MB/s
  localstack   █████████████████████████████████ 156.06 MB/s
  rustfs       ███████████████████████████ 128.39 MB/s
```

### Read/10MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio_mem | 305.97 MB/s | 456.3us | 551.8us | 31.3ms | 37.0ms | 37.0ms | 0 |
| localstack | 304.88 MB/s | 1.3ms | 1.4ms | 32.1ms | 33.0ms | 33.0ms | 0 |
| seaweedfs | 292.48 MB/s | 2.1ms | 2.2ms | 34.0ms | 34.8ms | 34.8ms | 0 |
| rustfs | 281.78 MB/s | 5.7ms | 6.8ms | 35.6ms | 35.8ms | 35.8ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 305.97 MB/s
  localstack   ███████████████████████████████████████ 304.88 MB/s
  seaweedfs    ██████████████████████████████████████ 292.48 MB/s
  rustfs       ████████████████████████████████████ 281.78 MB/s
```

### Read/1KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio_mem | 4.84 MB/s | 200.0us | 219.7us | 196.4us | 219.8us | 219.8us | 0 |
| seaweedfs | 2.30 MB/s | 424.1us | 510.5us | 410.4us | 510.5us | 510.5us | 0 |
| rustfs | 2.13 MB/s | 458.7us | 473.0us | 456.8us | 473.2us | 473.2us | 0 |
| localstack | 1.42 MB/s | 686.1us | 754.6us | 677.2us | 754.7us | 754.7us | 0 |

```
  liteio_mem   ████████████████████████████████████████ 4.84 MB/s
  seaweedfs    ███████████████████ 2.30 MB/s
  rustfs       █████████████████ 2.13 MB/s
  localstack   ███████████ 1.42 MB/s
```

### Read/1MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio_mem | 285.68 MB/s | 350.9us | 493.3us | 3.4ms | 3.9ms | 3.9ms | 0 |
| seaweedfs | 246.01 MB/s | 1.0ms | 1.4ms | 4.0ms | 4.3ms | 4.3ms | 0 |
| localstack | 244.68 MB/s | 1.1ms | 1.3ms | 4.0ms | 4.6ms | 4.6ms | 0 |
| rustfs | 199.31 MB/s | 1.9ms | 3.6ms | 4.6ms | 7.1ms | 7.1ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 285.68 MB/s
  seaweedfs    ██████████████████████████████████ 246.01 MB/s
  localstack   ██████████████████████████████████ 244.68 MB/s
  rustfs       ███████████████████████████ 199.31 MB/s
```

### Read/64KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio_mem | 148.55 MB/s | 220.0us | 316.7us | 407.5us | 477.0us | 497.9us | 0 |
| rustfs | 86.26 MB/s | 609.6us | 678.8us | 717.2us | 783.6us | 834.0us | 0 |
| seaweedfs | 82.05 MB/s | 593.0us | 1.1ms | 675.3us | 1.3ms | 1.5ms | 0 |
| localstack | 65.69 MB/s | 840.3us | 964.5us | 927.0us | 1.1ms | 1.2ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 148.55 MB/s
  rustfs       ███████████████████████ 86.26 MB/s
  seaweedfs    ██████████████████████ 82.05 MB/s
  localstack   █████████████████ 65.69 MB/s
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 4741 ops/s | 175.9us | 273.0us | 273.0us | 0 |
| seaweedfs | 3365 ops/s | 288.6us | 331.4us | 331.4us | 0 |
| rustfs | 3209 ops/s | 306.2us | 350.2us | 350.2us | 0 |
| localstack | 1437 ops/s | 657.9us | 834.4us | 834.4us | 0 |

```
  liteio_mem   ████████████████████████████████████████ 4741 ops/s
  seaweedfs    ████████████████████████████ 3365 ops/s
  rustfs       ███████████████████████████ 3209 ops/s
  localstack   ████████████ 1437 ops/s
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 188.91 MB/s | 50.5ms | 60.9ms | 60.9ms | 0 |
| liteio_mem | 187.94 MB/s | 53.2ms | 54.8ms | 54.8ms | 0 |
| localstack | 147.81 MB/s | 67.1ms | 69.4ms | 69.4ms | 0 |
| seaweedfs | 144.66 MB/s | 69.7ms | 76.4ms | 76.4ms | 0 |

```
  rustfs       ████████████████████████████████████████ 188.91 MB/s
  liteio_mem   ███████████████████████████████████████ 187.94 MB/s
  localstack   ███████████████████████████████ 147.81 MB/s
  seaweedfs    ██████████████████████████████ 144.66 MB/s
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 1.20 MB/s | 741.5us | 1.2ms | 1.2ms | 0 |
| localstack | 1.19 MB/s | 807.2us | 897.0us | 897.0us | 0 |
| rustfs | 1.09 MB/s | 856.0us | 1.0ms | 1.0ms | 0 |
| liteio_mem | 0.80 MB/s | 1.1ms | 1.7ms | 1.7ms | 0 |

```
  seaweedfs    ████████████████████████████████████████ 1.20 MB/s
  localstack   ███████████████████████████████████████ 1.19 MB/s
  rustfs       ████████████████████████████████████ 1.09 MB/s
  liteio_mem   ██████████████████████████ 0.80 MB/s
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 162.93 MB/s | 5.8ms | 7.1ms | 7.1ms | 0 |
| rustfs | 156.54 MB/s | 6.4ms | 7.9ms | 7.9ms | 0 |
| localstack | 127.89 MB/s | 7.6ms | 8.4ms | 8.4ms | 0 |
| seaweedfs | 118.01 MB/s | 8.1ms | 9.1ms | 9.1ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 162.93 MB/s
  rustfs       ██████████████████████████████████████ 156.54 MB/s
  localstack   ███████████████████████████████ 127.89 MB/s
  seaweedfs    ████████████████████████████ 118.01 MB/s
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 61.24 MB/s | 1.0ms | 1.3ms | 1.3ms | 0 |
| rustfs | 55.82 MB/s | 1.1ms | 1.4ms | 1.4ms | 0 |
| localstack | 54.88 MB/s | 1.1ms | 1.3ms | 1.7ms | 0 |
| seaweedfs | 53.86 MB/s | 1.2ms | 1.3ms | 1.4ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 61.24 MB/s
  rustfs       ████████████████████████████████████ 55.82 MB/s
  localstack   ███████████████████████████████████ 54.88 MB/s
  seaweedfs    ███████████████████████████████████ 53.86 MB/s
```

## Recommendations

- **Best for write-heavy workloads:** rustfs
- **Best for read-heavy workloads:** liteio_mem

---

*Report generated by storage benchmark CLI*
