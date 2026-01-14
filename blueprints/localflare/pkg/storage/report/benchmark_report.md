# Storage Benchmark Report

**Generated:** 2026-01-15T02:52:39+07:00

**Go Version:** go1.25.5

**Platform:** darwin/arm64

## Executive Summary

### Performance Leaders

```
┌───────────────────────────┬───────────────────────┬───────────────────────────────┐
│         Category          │        Leader         │             Notes             │
├───────────────────────────┼───────────────────────┼───────────────────────────────┤
│ Small File Read (1KB)     │ liteio_mem 4.9 MB/s   │ 1.6x faster than minio        │
├───────────────────────────┼───────────────────────┼───────────────────────────────┤
│ Small File Write (1KB)    │ liteio_mem 1.0 MB/s   │ Close competition             │
├───────────────────────────┼───────────────────────┼───────────────────────────────┤
│ Large File Read (100MB)   │ minio 316.9 MB/s      │ Close competition             │
├───────────────────────────┼───────────────────────┼───────────────────────────────┤
│ Large File Write (100MB)  │ liteio_mem 193.5 MB/s │ Close competition             │
├───────────────────────────┼───────────────────────┼───────────────────────────────┤
│ Delete Operations         │ liteio_mem 6252 ops/s │ 1.8x faster than minio        │
├───────────────────────────┼───────────────────────┼───────────────────────────────┤
│ Stat Operations           │ liteio_mem 5832 ops/s │ 38% faster than minio         │
├───────────────────────────┼───────────────────────┼───────────────────────────────┤
│ List Operations (100 obj) │ liteio_mem 1292 ops/s │ 2.2x faster than minio        │
├───────────────────────────┼───────────────────────┼───────────────────────────────┤
│ Copy Operations           │ liteio_mem 1.5 MB/s   │ 48% faster than minio         │
├───────────────────────────┼───────────────────────┼───────────────────────────────┤
│ Range Reads               │ liteio_mem 237.4 MB/s │ 31% faster than minio         │
├───────────────────────────┼───────────────────────┼───────────────────────────────┤
│ Mixed Workload            │ minio 4.9 MB/s        │ Close competition             │
├───────────────────────────┼───────────────────────┼───────────────────────────────┤
│ High Concurrency Read     │ liteio_mem 0.6 MB/s   │ 48% faster than minio         │
├───────────────────────────┼───────────────────────┼───────────────────────────────┤
│ High Concurrency Write    │ liteio_mem 0.2 MB/s   │ 23% faster than minio         │
└───────────────────────────┴───────────────────────┴───────────────────────────────┘
```

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (100MB+) | **liteio_mem** | 194 MB/s | Best for media, backups |
| Large File Downloads (100MB) | **minio** | 317 MB/s | Best for streaming, CDN |
| Small File Operations | **liteio_mem** | 3045 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **minio** | - | Best for multi-user apps |

### Large File Performance (100MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| liteio_mem | 193.5 | 295.0 | 517.0ms | 341.1ms |
| minio | 178.6 | 316.9 | 580.3ms | 315.9ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| liteio_mem | 1054 | 5036 | 834.5us | 194.0us |
| minio | 1049 | 3241 | 927.5us | 300.0us |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| liteio_mem | 5832 | 1292 | 6252 |
| minio | 4230 | 582 | 3450 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| liteio_mem | 1.31 | 0.33 | 0.24 | 0.15 | 0.17 | 0.15 |
| minio | 1.02 | 0.31 | 0.16 | 0.11 | 0.12 | 0.13 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| liteio_mem | 3.76 | 1.10 | 0.47 | 0.61 | 0.64 | 0.58 |
| minio | 2.53 | 1.21 | 0.58 | 0.40 | 0.38 | 0.39 |

*\* indicates errors occurred*

### File Count Performance

Performance with varying numbers of files (1KB each).

**Write N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|------|
| liteio_mem | 696.7us | 5.8ms | 59.9ms | 621.6ms | 6.31s |
| minio | 860.5us | 8.8ms | 84.0ms | 846.1ms | 8.90s |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|------|
| liteio_mem | 278.0us | 284.8us | 969.0us | 5.7ms | 193.5ms |
| minio | 462.4us | 594.9us | 1.8ms | 14.2ms | 166.6ms |

*\* indicates errors occurred*

---

## Configuration

| Parameter | Value |
|-----------|-------|
| Iterations | 50 |
| Warmup | 5 |
| Concurrency | 200 |
| Timeout | 3m0s |

## Drivers Tested

- liteio_mem (51 benchmarks)
- minio (51 benchmarks)

## Performance Comparison

### Copy/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1.55 MB/s | 614.9us | 805.2us | 846.1us | 0 |
| minio | 1.05 MB/s | 906.8us | 1.1ms | 1.2ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 1.55 MB/s
  minio        ███████████████████████████ 1.05 MB/s
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 6252 ops/s | 156.4us | 186.9us | 196.8us | 0 |
| minio | 3450 ops/s | 284.2us | 334.9us | 349.5us | 0 |

```
  liteio_mem   ████████████████████████████████████████ 6252 ops/s
  minio        ██████████████████████ 3450 ops/s
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 0.17 MB/s | 566.2us | 644.6us | 649.6us | 0 |
| minio | 0.12 MB/s | 785.5us | 936.1us | 996.5us | 0 |

```
  liteio_mem   ████████████████████████████████████████ 0.17 MB/s
  minio        ███████████████████████████ 0.12 MB/s
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1528 ops/s | 584.9us | 815.4us | 968.4us | 0 |
| minio | 1139 ops/s | 860.4us | 965.5us | 1.0ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 1528 ops/s
  minio        █████████████████████████████ 1139 ops/s
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 0.16 MB/s | 579.8us | 704.3us | 721.5us | 0 |
| minio | 0.11 MB/s | 848.1us | 908.8us | 921.6us | 0 |

```
  liteio_mem   ████████████████████████████████████████ 0.16 MB/s
  minio        ████████████████████████████ 0.11 MB/s
```

### FileCount/Delete/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 4778 ops/s | 209.3us | 209.3us | 209.3us | 0 |
| minio | 2382 ops/s | 419.8us | 419.8us | 419.8us | 0 |

```
  liteio_mem   ████████████████████████████████████████ 4778 ops/s
  minio        ███████████████████ 2382 ops/s
```

### FileCount/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 518 ops/s | 1.9ms | 1.9ms | 1.9ms | 0 |
| minio | 287 ops/s | 3.5ms | 3.5ms | 3.5ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 518 ops/s
  minio        ██████████████████████ 287 ops/s
```

### FileCount/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 52 ops/s | 19.4ms | 19.4ms | 19.4ms | 0 |
| minio | 29 ops/s | 34.8ms | 34.8ms | 34.8ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 52 ops/s
  minio        ██████████████████████ 29 ops/s
```

### FileCount/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 5 ops/s | 188.6ms | 188.6ms | 188.6ms | 0 |
| minio | 3 ops/s | 384.8ms | 384.8ms | 384.8ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 5 ops/s
  minio        ███████████████████ 3 ops/s
```

### FileCount/Delete/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1 ops/s | 1.92s | 1.92s | 1.92s | 0 |
| minio | 0 ops/s | 3.55s | 3.55s | 3.55s | 0 |

```
  liteio_mem   ████████████████████████████████████████ 1 ops/s
  minio        █████████████████████ 0 ops/s
```

### FileCount/List/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 3597 ops/s | 278.0us | 278.0us | 278.0us | 0 |
| minio | 2163 ops/s | 462.4us | 462.4us | 462.4us | 0 |

```
  liteio_mem   ████████████████████████████████████████ 3597 ops/s
  minio        ████████████████████████ 2163 ops/s
```

### FileCount/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 3511 ops/s | 284.8us | 284.8us | 284.8us | 0 |
| minio | 1681 ops/s | 594.9us | 594.9us | 594.9us | 0 |

```
  liteio_mem   ████████████████████████████████████████ 3511 ops/s
  minio        ███████████████████ 1681 ops/s
```

### FileCount/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1032 ops/s | 969.0us | 969.0us | 969.0us | 0 |
| minio | 566 ops/s | 1.8ms | 1.8ms | 1.8ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 1032 ops/s
  minio        █████████████████████ 566 ops/s
```

### FileCount/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 176 ops/s | 5.7ms | 5.7ms | 5.7ms | 0 |
| minio | 71 ops/s | 14.2ms | 14.2ms | 14.2ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 176 ops/s
  minio        ████████████████ 71 ops/s
```

### FileCount/List/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 6 ops/s | 166.6ms | 166.6ms | 166.6ms | 0 |
| liteio_mem | 5 ops/s | 193.5ms | 193.5ms | 193.5ms | 0 |

```
  minio        ████████████████████████████████████████ 6 ops/s
  liteio_mem   ██████████████████████████████████ 5 ops/s
```

### FileCount/Write/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1.40 MB/s | 696.7us | 696.7us | 696.7us | 0 |
| minio | 1.13 MB/s | 860.5us | 860.5us | 860.5us | 0 |

```
  liteio_mem   ████████████████████████████████████████ 1.40 MB/s
  minio        ████████████████████████████████ 1.13 MB/s
```

### FileCount/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1.68 MB/s | 5.8ms | 5.8ms | 5.8ms | 0 |
| minio | 1.11 MB/s | 8.8ms | 8.8ms | 8.8ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 1.68 MB/s
  minio        ██████████████████████████ 1.11 MB/s
```

### FileCount/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1.63 MB/s | 59.9ms | 59.9ms | 59.9ms | 0 |
| minio | 1.16 MB/s | 84.0ms | 84.0ms | 84.0ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 1.63 MB/s
  minio        ████████████████████████████ 1.16 MB/s
```

### FileCount/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1.57 MB/s | 621.6ms | 621.6ms | 621.6ms | 0 |
| minio | 1.15 MB/s | 846.1ms | 846.1ms | 846.1ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 1.57 MB/s
  minio        █████████████████████████████ 1.15 MB/s
```

### FileCount/Write/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1.55 MB/s | 6.31s | 6.31s | 6.31s | 0 |
| minio | 1.10 MB/s | 8.90s | 8.90s | 8.90s | 0 |

```
  liteio_mem   ████████████████████████████████████████ 1.55 MB/s
  minio        ████████████████████████████ 1.10 MB/s
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1292 ops/s | 730.9us | 837.0us | 1.2ms | 0 |
| minio | 582 ops/s | 1.7ms | 1.9ms | 2.0ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 1292 ops/s
  minio        ██████████████████ 582 ops/s
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 4.92 MB/s | 3.3ms | 4.2ms | 4.3ms | 0 |
| liteio_mem | 4.89 MB/s | 3.2ms | 4.4ms | 4.5ms | 0 |

```
  minio        ████████████████████████████████████████ 4.92 MB/s
  liteio_mem   ███████████████████████████████████████ 4.89 MB/s
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 3.13 MB/s | 5.1ms | 5.4ms | 5.5ms | 0 |
| liteio_mem | 3.10 MB/s | 5.1ms | 5.7ms | 5.8ms | 0 |

```
  minio        ████████████████████████████████████████ 3.13 MB/s
  liteio_mem   ███████████████████████████████████████ 3.10 MB/s
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 2.25 MB/s | 8.0ms | 9.0ms | 9.7ms | 0 |
| minio | 1.82 MB/s | 9.4ms | 12.4ms | 12.5ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 2.25 MB/s
  minio        ████████████████████████████████ 1.82 MB/s
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 167.06 MB/s | 87.1ms | 97.4ms | 97.4ms | 0 |
| liteio_mem | 155.69 MB/s | 96.1ms | 99.9ms | 99.9ms | 0 |

```
  minio        ████████████████████████████████████████ 167.06 MB/s
  liteio_mem   █████████████████████████████████████ 155.69 MB/s
```

### ParallelRead/1KB/C1

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio_mem | 3.76 MB/s | 254.2us | 299.3us | 212.4us | 314.7us | 691.0us | 0 |
| minio | 2.53 MB/s | 385.9us | 546.7us | 350.1us | 547.5us | 694.9us | 0 |

```
  liteio_mem   ████████████████████████████████████████ 3.76 MB/s
  minio        ██████████████████████████ 2.53 MB/s
```

### ParallelRead/1KB/C10

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 1.21 MB/s | 804.9us | 1.1ms | 735.4us | 1.1ms | 1.2ms | 0 |
| liteio_mem | 1.10 MB/s | 861.0us | 1.3ms | 799.7us | 1.3ms | 1.6ms | 0 |

```
  minio        ████████████████████████████████████████ 1.21 MB/s
  liteio_mem   ████████████████████████████████████ 1.10 MB/s
```

### ParallelRead/1KB/C100

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio_mem | 0.64 MB/s | 1.5ms | 2.0ms | 1.5ms | 2.0ms | 2.0ms | 0 |
| minio | 0.38 MB/s | 2.6ms | 3.0ms | 2.6ms | 3.0ms | 3.1ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 0.64 MB/s
  minio        ███████████████████████ 0.38 MB/s
```

### ParallelRead/1KB/C200

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio_mem | 0.58 MB/s | 1.7ms | 2.2ms | 1.6ms | 2.2ms | 2.3ms | 0 |
| minio | 0.39 MB/s | 2.5ms | 2.8ms | 2.5ms | 2.8ms | 2.9ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 0.58 MB/s
  minio        ███████████████████████████ 0.39 MB/s
```

### ParallelRead/1KB/C25

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 0.58 MB/s | 1.7ms | 2.1ms | 1.7ms | 2.1ms | 2.2ms | 0 |
| liteio_mem | 0.47 MB/s | 2.1ms | 3.8ms | 2.0ms | 3.8ms | 4.0ms | 0 |

```
  minio        ████████████████████████████████████████ 0.58 MB/s
  liteio_mem   ████████████████████████████████ 0.47 MB/s
```

### ParallelRead/1KB/C50

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio_mem | 0.61 MB/s | 1.6ms | 2.1ms | 1.6ms | 2.1ms | 2.1ms | 0 |
| minio | 0.40 MB/s | 2.4ms | 3.0ms | 2.6ms | 3.0ms | 3.1ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 0.61 MB/s
  minio        ██████████████████████████ 0.40 MB/s
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1.31 MB/s | 700.9us | 899.0us | 1.0ms | 0 |
| minio | 1.02 MB/s | 862.3us | 1.3ms | 1.6ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 1.31 MB/s
  minio        ██████████████████████████████ 1.02 MB/s
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 0.33 MB/s | 2.7ms | 5.1ms | 5.3ms | 0 |
| minio | 0.31 MB/s | 3.0ms | 4.8ms | 5.1ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 0.33 MB/s
  minio        ██████████████████████████████████████ 0.31 MB/s
```

### ParallelWrite/1KB/C100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 0.17 MB/s | 5.7ms | 8.5ms | 8.6ms | 0 |
| minio | 0.12 MB/s | 8.6ms | 11.6ms | 11.9ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 0.17 MB/s
  minio        ████████████████████████████ 0.12 MB/s
```

### ParallelWrite/1KB/C200

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 0.15 MB/s | 6.2ms | 9.7ms | 9.8ms | 0 |
| minio | 0.13 MB/s | 8.1ms | 10.4ms | 10.4ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 0.15 MB/s
  minio        ████████████████████████████████ 0.13 MB/s
```

### ParallelWrite/1KB/C25

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 0.24 MB/s | 3.8ms | 6.8ms | 7.0ms | 0 |
| minio | 0.16 MB/s | 5.5ms | 11.2ms | 12.1ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 0.24 MB/s
  minio        ███████████████████████████ 0.16 MB/s
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 0.15 MB/s | 6.3ms | 9.2ms | 9.6ms | 0 |
| minio | 0.11 MB/s | 8.6ms | 12.5ms | 12.7ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 0.15 MB/s
  minio        ████████████████████████████ 0.11 MB/s
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 235.07 MB/s | 1.0ms | 1.2ms | 1.2ms | 0 |
| minio | 150.05 MB/s | 1.5ms | 1.9ms | 2.3ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 235.07 MB/s
  minio        █████████████████████████ 150.05 MB/s
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 235.67 MB/s | 1.0ms | 1.2ms | 1.4ms | 0 |
| minio | 163.09 MB/s | 1.5ms | 1.7ms | 1.7ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 235.67 MB/s
  minio        ███████████████████████████ 163.09 MB/s
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 237.37 MB/s | 1.0ms | 1.2ms | 1.2ms | 0 |
| minio | 181.23 MB/s | 1.4ms | 1.5ms | 1.6ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 237.37 MB/s
  minio        ██████████████████████████████ 181.23 MB/s
```

### Read/100MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 316.90 MB/s | 1.1ms | 1.1ms | 315.9ms | 318.9ms | 318.9ms | 0 |
| liteio_mem | 294.98 MB/s | 513.0us | 416.7us | 341.1ms | 343.7ms | 343.7ms | 0 |

```
  minio        ████████████████████████████████████████ 316.90 MB/s
  liteio_mem   █████████████████████████████████████ 294.98 MB/s
```

### Read/10MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio_mem | 303.69 MB/s | 456.5us | 492.6us | 31.7ms | 32.9ms | 32.9ms | 0 |
| minio | 297.63 MB/s | 1.2ms | 1.3ms | 32.6ms | 34.5ms | 34.5ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 303.69 MB/s
  minio        ███████████████████████████████████████ 297.63 MB/s
```

### Read/1KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio_mem | 4.92 MB/s | 196.9us | 256.3us | 194.0us | 256.4us | 293.2us | 0 |
| minio | 3.17 MB/s | 308.4us | 366.2us | 300.0us | 366.2us | 366.8us | 0 |

```
  liteio_mem   ████████████████████████████████████████ 4.92 MB/s
  minio        █████████████████████████ 3.17 MB/s
```

### Read/1MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio_mem | 291.27 MB/s | 300.8us | 415.5us | 3.4ms | 3.8ms | 3.8ms | 0 |
| minio | 256.50 MB/s | 816.9us | 980.4us | 3.8ms | 4.1ms | 4.1ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 291.27 MB/s
  minio        ███████████████████████████████████ 256.50 MB/s
```

### Read/64KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio_mem | 148.97 MB/s | 235.6us | 304.7us | 411.6us | 467.0us | 523.5us | 0 |
| minio | 113.05 MB/s | 371.7us | 457.8us | 546.7us | 647.6us | 678.4us | 0 |

```
  liteio_mem   ████████████████████████████████████████ 148.97 MB/s
  minio        ██████████████████████████████ 113.05 MB/s
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 5832 ops/s | 168.9us | 203.4us | 225.2us | 0 |
| minio | 4230 ops/s | 232.0us | 277.9us | 299.4us | 0 |

```
  liteio_mem   ████████████████████████████████████████ 5832 ops/s
  minio        █████████████████████████████ 4230 ops/s
```

### Write/100MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 193.52 MB/s | 517.0ms | 525.3ms | 525.3ms | 0 |
| minio | 178.60 MB/s | 580.3ms | 588.8ms | 588.8ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 193.52 MB/s
  minio        ████████████████████████████████████ 178.60 MB/s
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 194.13 MB/s | 50.9ms | 53.9ms | 53.9ms | 0 |
| minio | 186.95 MB/s | 52.2ms | 55.9ms | 55.9ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 194.13 MB/s
  minio        ██████████████████████████████████████ 186.95 MB/s
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1.03 MB/s | 834.5us | 1.3ms | 2.0ms | 0 |
| minio | 1.02 MB/s | 927.5us | 1.1ms | 1.2ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 1.03 MB/s
  minio        ███████████████████████████████████████ 1.02 MB/s
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 165.30 MB/s | 5.8ms | 6.6ms | 6.6ms | 0 |
| minio | 142.14 MB/s | 6.9ms | 7.3ms | 7.3ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 165.30 MB/s
  minio        ██████████████████████████████████ 142.14 MB/s
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 63.19 MB/s | 942.1us | 1.3ms | 1.3ms | 0 |
| minio | 46.16 MB/s | 1.3ms | 1.7ms | 2.2ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 63.19 MB/s
  minio        █████████████████████████████ 46.16 MB/s
```

## Recommendations

- **Best for write-heavy workloads:** liteio_mem
- **Best for read-heavy workloads:** minio

---

*Report generated by storage benchmark CLI*
