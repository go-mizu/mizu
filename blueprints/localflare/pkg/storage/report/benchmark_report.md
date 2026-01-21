# Storage Benchmark Report

**Generated:** 2026-01-21T11:37:12+07:00

**Go Version:** go1.25.5

**Platform:** darwin/arm64

## Executive Summary

### Quick Results

**Overall Winner:** devnull (won 51/51 benchmarks, 100%)

| Rank | Driver | Wins | Win Rate |
|------|--------|------|----------|
| 1 🥇 | devnull | 51 | 100% |

### 5x Performance Target

**Target Status:** NOT MET - liteio meets 5x target in only 0/7 key benchmarks (0%)

| Benchmark | liteio | Best Other | Ratio | Target (5x) |
|-----------|--------|------------|-------|-------------|
| Write/1KB | 1.3 MB/s | 4.1 GB/s (devnull) | 0.0x | [-] FAIL |
| Read/1KB | 3.6 MB/s | 5.7 GB/s (devnull) | 0.0x | [-] FAIL |
| Write/10MB | 118.2 MB/s | 51.5 GB/s (devnull) | 0.0x | [-] FAIL |
| Read/10MB | 167.0 MB/s | 52164.8 GB/s (devnull) | 0.0x | [-] FAIL |
| Stat | 2.4K ops/s | 19368.6K ops/s (devnull) | 0.0x | [-] FAIL |
| Delete | 2.5K ops/s | 20185.7K ops/s (devnull) | 0.0x | [-] FAIL |
| List/100 | 601 ops/s | 87.1K ops/s (devnull) | 0.0x | [-] FAIL |

### Performance Leaders

```
┌───────────────────────────┬───────────────────────┬───────────────────────────────┐
│         Category          │        Leader         │             Notes             │
├───────────────────────────┼───────────────────────┼───────────────────────────────┤
│ Small File Read (1KB)     │ devnull 5673.7 MB/s   │ 1573.1x faster than liteio    │
├───────────────────────────┼───────────────────────┼───────────────────────────────┤
│ Small File Write (1KB)    │ devnull 4148.5 MB/s   │ 2932.5x faster than liteio_mem│
├───────────────────────────┼───────────────────────┼───────────────────────────────┤
│ Large File Read (100MB)   │devnull 187406296.9 MB/│1179541.0x faster than liteio_m│
├───────────────────────────┼───────────────────────┼───────────────────────────────┤
│ Large File Write (100MB)  │ devnull 48957.8 MB/s  │ 418.7x faster than liteio_mem │
├───────────────────────────┼───────────────────────┼───────────────────────────────┤
│ Delete Operations         │ devnull 20185709 ops/s│ 7925.1x faster than liteio    │
├───────────────────────────┼───────────────────────┼───────────────────────────────┤
│ Stat Operations           │ devnull 19368584 ops/s│ 7958.6x faster than liteio    │
├───────────────────────────┼───────────────────────┼───────────────────────────────┤
│ List Operations (100 obj) │ devnull 87051 ops/s   │ 144.9x faster than liteio     │
├───────────────────────────┼───────────────────────┼───────────────────────────────┤
│ Copy Operations           │ devnull 6718.7 MB/s   │ 9707.0x faster than liteio    │
├───────────────────────────┼───────────────────────┼───────────────────────────────┤
│ Range Reads               │ devnull 2738825.6 MB/s│ 25394.3x faster than liteio   │
├───────────────────────────┼───────────────────────┼───────────────────────────────┤
│ Mixed Workload            │ devnull 15470.5 MB/s  │ 4748.8x faster than rustfs    │
├───────────────────────────┼───────────────────────┼───────────────────────────────┤
│ High Concurrency Read     │ devnull 1543.0 MB/s   │ 7414.6x faster than liteio    │
├───────────────────────────┼───────────────────────┼───────────────────────────────┤
│ High Concurrency Write    │ devnull 129.2 MB/s    │ 1856.8x faster than liteio    │
└───────────────────────────┴───────────────────────┴───────────────────────────────┘
```

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (100MB+) | **devnull** | 48958 MB/s | Best for media, backups |
| Large File Downloads (100MB) | **devnull** | 187406297 MB/s | Best for streaming, CDN |
| Small File Operations | **devnull** | 5028994 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **devnull** | - | Best for multi-user apps |

### Large File Performance (100MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| devnull | 48957.8 | 187406296.9 | 2.0ms | 334ns |
| liteio | 68.8 | 155.7 | 1.76s | 648.5ms |
| liteio_mem | 116.9 | 158.9 | 852.7ms | 563.3ms |
| localstack | 42.6 | 94.5 | 2.35s | 1.10s |
| minio | 66.4 | 39.4 | 950.0ms | 2.34s |
| rustfs | 71.2 | 103.3 | 998.4ms | 1.01s |
| seaweedfs | 85.7 | 154.2 | 1.16s | 649.9ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| devnull | 4248088 | 5809900 | 166ns | 125ns |
| liteio | 1328 | 3693 | 677.2us | 249.8us |
| liteio_mem | 1449 | 1412 | 624.7us | 606.9us |
| localstack | 533 | 507 | 1.3ms | 1.4ms |
| minio | 661 | 376 | 1.0ms | 2.2ms |
| rustfs | 706 | 1017 | 1.2ms | 898.2us |
| seaweedfs | 644 | 1170 | 1.3ms | 751.1us |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| devnull | 19368584 | 87051 | 20185709 |
| liteio | 2434 | 601 | 2547 |
| liteio_mem | 627 | 435 | 287 |
| localstack | 198 | 115 | 142 |
| minio | 89 | 95 | 96 |
| rustfs | 545 | 73 | 715 |
| seaweedfs | 1767 | 414 | 2073 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| devnull | 4170.14 | 1724.61 | 211.11 | 12.37 | 418.23 | 129.20 |
| liteio | 0.72 | 0.30 | 0.17 | 0.07 | 0.05 | 0.07 |
| liteio_mem | 0.19 | 0.10 | 0.08 | 0.03 | 0.02 | 0.06 |
| localstack | 0.11 | 0.02 | 0.01 | 0.01 | 0.00 | 0.00 |
| minio | 0.09 | 0.04 | 0.02 | 0.01 | 0.01 | 0.01 |
| rustfs | 0.95 | 0.21 | - | - | - | - |
| seaweedfs | 0.92 | 0.32 | 0.13 | 0.09 | 0.06 | 0.06 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| devnull | 5920.71 | 2393.95 | 862.58 | 2803.15 | 1617.44 | 1543.05 |
| liteio | 2.08 | 1.13 | 0.52 | 0.37 | 0.27 | 0.21 |
| liteio_mem | 0.23 | 0.23 | 0.15 | 0.08 | 0.06 | 0.04 |
| localstack | 0.16 | 0.04 | 0.01 | 0.01 | 0.00 | 0.00 |
| minio | 0.25 | 0.11 | 0.09 | 0.05 | 0.04 | 0.07 |
| rustfs | 1.42 | 0.44 | - | - | - | - |
| seaweedfs | 1.72 | 0.51 | 0.31 | 0.07 | 0.12 | 0.17 |

*\* indicates errors occurred*

### File Count Performance

Performance with varying numbers of files (1KB each).

**Write N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|------|
| devnull | 2.5us | 4.7us | 56.5us | 321.5us | 3.9ms |
| liteio | 798.1us | 7.2ms | 69.2ms | 717.9ms | 9.80s |
| liteio_mem | 713.9us | 3.3ms | 33.7ms | 338.8ms | 3.28s |
| localstack | 6.1ms | 35.0ms | 209.8ms | 1.35s | 14.65s |
| minio | 1.1ms | 15.3ms | 159.3ms | 3.61s | 180.95s |
| rustfs | 1.2ms | 12.2ms | 128.4ms | 1.22s | 15.85s |
| seaweedfs | 1.5ms | 14.3ms | 161.7ms | 1.78s | 19.61s |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|------|
| devnull | 22.4us | 20.6us | 37.1us | 164.5us | 1.7ms |
| liteio | 355.4us | 432.5us | 1.4ms | 6.6ms | 220.7ms |
| liteio_mem | 0ns* | 0ns* | 0ns* | 0ns* | 0ns* |
| localstack | 5.5ms | 3.4ms | 4.1ms | 30.6ms | 0ns* |
| minio | 1.2ms | 1.6ms | 3.1ms | 39.8ms | 0ns* |
| rustfs | 1.8ms | 2.7ms | 12.2ms | 81.8ms | 1.08s |
| seaweedfs | 1.3ms | 1.9ms | 2.7ms | 12.9ms | 208.2ms |

*\* indicates errors occurred*

### Warnings

- **liteio_mem**: 4482 errors during benchmarks
- **localstack**: 1 errors during benchmarks
- **minio**: 2 errors during benchmarks

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

---

## Configuration

| Parameter | Value |
|-----------|-------|
| Iterations | 100 |
| Warmup | 10 |
| Concurrency | 200 |
| Timeout | 30s |

## Drivers Tested

- devnull (51 benchmarks)
- liteio (51 benchmarks)
- liteio_mem (51 benchmarks)
- localstack (51 benchmarks)
- minio (51 benchmarks)
- rustfs (43 benchmarks)
- seaweedfs (51 benchmarks)

## Performance Comparison

### Copy/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull | 6718.70 MB/s | 84ns | 292ns | 417ns | 0 |
| liteio | 0.69 MB/s | 1.0ms | 2.7ms | 4.3ms | 0 |
| rustfs | 0.57 MB/s | 1.5ms | 3.1ms | 3.6ms | 0 |
| seaweedfs | 0.37 MB/s | 2.1ms | 5.0ms | 8.0ms | 0 |
| localstack | 0.10 MB/s | 6.9ms | 19.5ms | 48.5ms | 0 |
| minio | 0.03 MB/s | 25.3ms | 49.1ms | 88.1ms | 0 |
| liteio_mem | 0.00 MB/s | 0ns | 0ns | 0ns | 100 |

```
  devnull      ████████████████████████████████████████ 6718.70 MB/s
  liteio       █ 0.69 MB/s
  rustfs       █ 0.57 MB/s
  seaweedfs    █ 0.37 MB/s
  localstack   █ 0.10 MB/s
  minio        █ 0.03 MB/s
  liteio_mem   █ 0.00 MB/s
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull | 20185709 ops/s | 42ns | 83ns | 83ns | 0 |
| liteio | 2547 ops/s | 372.5us | 611.7us | 729.0us | 0 |
| seaweedfs | 2073 ops/s | 456.4us | 764.5us | 872.9us | 0 |
| rustfs | 715 ops/s | 1.2ms | 2.5ms | 2.9ms | 0 |
| liteio_mem | 287 ops/s | 3.0ms | 7.7ms | 11.0ms | 0 |
| localstack | 142 ops/s | 5.6ms | 13.3ms | 27.4ms | 0 |
| minio | 96 ops/s | 8.1ms | 22.8ms | 44.4ms | 0 |

```
  devnull      ████████████████████████████████████████ 20185709 ops/s
  liteio       █ 2547 ops/s
  seaweedfs    █ 2073 ops/s
  rustfs       █ 715 ops/s
  liteio_mem   █ 287 ops/s
  localstack   █ 142 ops/s
  minio        █ 96 ops/s
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull | 484.98 MB/s | 167ns | 292ns | 416ns | 0 |
| liteio_mem | 0.29 MB/s | 266.0us | 569.3us | 852.3us | 0 |
| liteio | 0.14 MB/s | 678.0us | 912.7us | 944.7us | 0 |
| rustfs | 0.08 MB/s | 1.1ms | 1.9ms | 2.0ms | 0 |
| minio | 0.07 MB/s | 1.1ms | 2.3ms | 3.1ms | 0 |
| seaweedfs | 0.07 MB/s | 1.3ms | 1.7ms | 2.2ms | 0 |
| localstack | 0.04 MB/s | 2.4ms | 3.6ms | 4.5ms | 0 |

```
  devnull      ████████████████████████████████████████ 484.98 MB/s
  liteio_mem   █ 0.29 MB/s
  liteio       █ 0.14 MB/s
  rustfs       █ 0.08 MB/s
  minio        █ 0.07 MB/s
  seaweedfs    █ 0.07 MB/s
  localstack   █ 0.04 MB/s
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull | 3690037 ops/s | 250ns | 458ns | 667ns | 0 |
| liteio_mem | 2871 ops/s | 286.4us | 518.7us | 826.6us | 0 |
| seaweedfs | 1293 ops/s | 760.6us | 1.0ms | 1.1ms | 0 |
| liteio | 1199 ops/s | 764.8us | 1.3ms | 1.4ms | 0 |
| minio | 678 ops/s | 1.1ms | 3.1ms | 3.9ms | 0 |
| rustfs | 634 ops/s | 1.5ms | 2.4ms | 2.8ms | 0 |
| localstack | 514 ops/s | 1.6ms | 3.5ms | 3.8ms | 0 |

```
  devnull      ████████████████████████████████████████ 3690037 ops/s
  liteio_mem   █ 2871 ops/s
  seaweedfs    █ 1293 ops/s
  liteio       █ 1199 ops/s
  minio        █ 678 ops/s
  rustfs       █ 634 ops/s
  localstack   █ 514 ops/s
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull | 375.08 MB/s | 209ns | 625ns | 667ns | 0 |
| liteio_mem | 0.31 MB/s | 269.0us | 433.6us | 560.7us | 0 |
| liteio | 0.11 MB/s | 804.0us | 1.2ms | 1.3ms | 0 |
| rustfs | 0.08 MB/s | 1.1ms | 1.8ms | 2.0ms | 0 |
| minio | 0.07 MB/s | 1.2ms | 2.0ms | 3.0ms | 0 |
| seaweedfs | 0.05 MB/s | 1.7ms | 2.5ms | 2.7ms | 0 |
| localstack | 0.04 MB/s | 2.3ms | 3.5ms | 3.8ms | 0 |

```
  devnull      ████████████████████████████████████████ 375.08 MB/s
  liteio_mem   █ 0.31 MB/s
  liteio       █ 0.11 MB/s
  rustfs       █ 0.08 MB/s
  minio        █ 0.07 MB/s
  seaweedfs    █ 0.05 MB/s
  localstack   █ 0.04 MB/s
```

### FileCount/Delete/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull | 489956 ops/s | 2.0us | 2.0us | 2.0us | 0 |
| liteio_mem | 2959 ops/s | 338.0us | 338.0us | 338.0us | 0 |
| liteio | 2956 ops/s | 338.2us | 338.2us | 338.2us | 0 |
| seaweedfs | 1552 ops/s | 644.4us | 644.4us | 644.4us | 0 |
| minio | 1308 ops/s | 764.6us | 764.6us | 764.6us | 0 |
| rustfs | 584 ops/s | 1.7ms | 1.7ms | 1.7ms | 0 |
| localstack | 213 ops/s | 4.7ms | 4.7ms | 4.7ms | 0 |

```
  devnull      ████████████████████████████████████████ 489956 ops/s
  liteio_mem   █ 2959 ops/s
  liteio       █ 2956 ops/s
  seaweedfs    █ 1552 ops/s
  minio        █ 1308 ops/s
  rustfs       █ 584 ops/s
  localstack   █ 213 ops/s
```

### FileCount/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull | 320000 ops/s | 3.1us | 3.1us | 3.1us | 0 |
| liteio | 404 ops/s | 2.5ms | 2.5ms | 2.5ms | 0 |
| liteio_mem | 332 ops/s | 3.0ms | 3.0ms | 3.0ms | 0 |
| seaweedfs | 165 ops/s | 6.1ms | 6.1ms | 6.1ms | 0 |
| minio | 160 ops/s | 6.3ms | 6.3ms | 6.3ms | 0 |
| rustfs | 60 ops/s | 16.8ms | 16.8ms | 16.8ms | 0 |
| localstack | 41 ops/s | 24.4ms | 24.4ms | 24.4ms | 0 |

```
  devnull      ████████████████████████████████████████ 320000 ops/s
  liteio       █ 404 ops/s
  liteio_mem   █ 332 ops/s
  seaweedfs    █ 165 ops/s
  minio        █ 160 ops/s
  rustfs       █ 60 ops/s
  localstack   █ 41 ops/s
```

### FileCount/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull | 63327 ops/s | 15.8us | 15.8us | 15.8us | 0 |
| liteio | 40 ops/s | 25.1ms | 25.1ms | 25.1ms | 0 |
| liteio_mem | 37 ops/s | 26.8ms | 26.8ms | 26.8ms | 0 |
| seaweedfs | 16 ops/s | 62.7ms | 62.7ms | 62.7ms | 0 |
| minio | 14 ops/s | 70.0ms | 70.0ms | 70.0ms | 0 |
| localstack | 11 ops/s | 90.1ms | 90.1ms | 90.1ms | 0 |
| rustfs | 5 ops/s | 211.9ms | 211.9ms | 211.9ms | 0 |

```
  devnull      ████████████████████████████████████████ 63327 ops/s
  liteio       █ 40 ops/s
  liteio_mem   █ 37 ops/s
  seaweedfs    █ 16 ops/s
  minio        █ 14 ops/s
  localstack   █ 11 ops/s
  rustfs       █ 5 ops/s
```

### FileCount/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull | 6950 ops/s | 143.9us | 143.9us | 143.9us | 0 |
| liteio_mem | 4 ops/s | 235.1ms | 235.1ms | 235.1ms | 0 |
| liteio | 4 ops/s | 239.1ms | 239.1ms | 239.1ms | 0 |
| seaweedfs | 2 ops/s | 476.1ms | 476.1ms | 476.1ms | 0 |
| localstack | 1 ops/s | 741.0ms | 741.0ms | 741.0ms | 0 |
| rustfs | 1 ops/s | 1.21s | 1.21s | 1.21s | 0 |
| minio | 0 ops/s | 6.53s | 6.53s | 6.53s | 0 |

```
  devnull      ████████████████████████████████████████ 6950 ops/s
  liteio_mem   █ 4 ops/s
  liteio       █ 4 ops/s
  seaweedfs    █ 2 ops/s
  localstack   █ 1 ops/s
  rustfs       █ 1 ops/s
  minio        █ 0 ops/s
```

### FileCount/Delete/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull | 714 ops/s | 1.4ms | 1.4ms | 1.4ms | 0 |
| liteio | 0 ops/s | 2.50s | 2.50s | 2.50s | 0 |
| seaweedfs | 0 ops/s | 4.71s | 4.71s | 4.71s | 0 |
| minio | 0 ops/s | 5.34s | 5.34s | 5.34s | 0 |
| rustfs | 0 ops/s | 11.24s | 11.24s | 11.24s | 0 |
| localstack | 0 ops/s | 11.99s | 11.99s | 11.99s | 0 |
| liteio_mem | 0 ops/s | 174.82s | 174.82s | 174.82s | 4377 |

```
  devnull      ████████████████████████████████████████ 714 ops/s
  liteio       █ 0 ops/s
  seaweedfs    █ 0 ops/s
  minio        █ 0 ops/s
  rustfs       █ 0 ops/s
  localstack   █ 0 ops/s
  liteio_mem   █ 0 ops/s
```

### FileCount/List/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull | 44693 ops/s | 22.4us | 22.4us | 22.4us | 0 |
| liteio | 2814 ops/s | 355.4us | 355.4us | 355.4us | 0 |
| minio | 825 ops/s | 1.2ms | 1.2ms | 1.2ms | 0 |
| seaweedfs | 793 ops/s | 1.3ms | 1.3ms | 1.3ms | 0 |
| rustfs | 541 ops/s | 1.8ms | 1.8ms | 1.8ms | 0 |
| localstack | 182 ops/s | 5.5ms | 5.5ms | 5.5ms | 0 |
| liteio_mem | 0 ops/s | 0ns | 0ns | 0ns | 1 |

```
  devnull      ████████████████████████████████████████ 44693 ops/s
  liteio       ██ 2814 ops/s
  minio        █ 825 ops/s
  seaweedfs    █ 793 ops/s
  rustfs       █ 541 ops/s
  localstack   █ 182 ops/s
  liteio_mem   █ 0 ops/s
```

### FileCount/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull | 48584 ops/s | 20.6us | 20.6us | 20.6us | 0 |
| liteio | 2312 ops/s | 432.5us | 432.5us | 432.5us | 0 |
| minio | 617 ops/s | 1.6ms | 1.6ms | 1.6ms | 0 |
| seaweedfs | 519 ops/s | 1.9ms | 1.9ms | 1.9ms | 0 |
| rustfs | 370 ops/s | 2.7ms | 2.7ms | 2.7ms | 0 |
| localstack | 293 ops/s | 3.4ms | 3.4ms | 3.4ms | 0 |
| liteio_mem | 0 ops/s | 0ns | 0ns | 0ns | 1 |

```
  devnull      ████████████████████████████████████████ 48584 ops/s
  liteio       █ 2312 ops/s
  minio        █ 617 ops/s
  seaweedfs    █ 519 ops/s
  rustfs       █ 370 ops/s
  localstack   █ 293 ops/s
  liteio_mem   █ 0 ops/s
```

### FileCount/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull | 26936 ops/s | 37.1us | 37.1us | 37.1us | 0 |
| liteio | 710 ops/s | 1.4ms | 1.4ms | 1.4ms | 0 |
| seaweedfs | 372 ops/s | 2.7ms | 2.7ms | 2.7ms | 0 |
| minio | 325 ops/s | 3.1ms | 3.1ms | 3.1ms | 0 |
| localstack | 244 ops/s | 4.1ms | 4.1ms | 4.1ms | 0 |
| rustfs | 82 ops/s | 12.2ms | 12.2ms | 12.2ms | 0 |
| liteio_mem | 0 ops/s | 0ns | 0ns | 0ns | 1 |

```
  devnull      ████████████████████████████████████████ 26936 ops/s
  liteio       █ 710 ops/s
  seaweedfs    █ 372 ops/s
  minio        █ 325 ops/s
  localstack   █ 244 ops/s
  rustfs       █ 82 ops/s
  liteio_mem   █ 0 ops/s
```

### FileCount/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull | 6081 ops/s | 164.5us | 164.5us | 164.5us | 0 |
| liteio | 151 ops/s | 6.6ms | 6.6ms | 6.6ms | 0 |
| seaweedfs | 78 ops/s | 12.9ms | 12.9ms | 12.9ms | 0 |
| localstack | 33 ops/s | 30.6ms | 30.6ms | 30.6ms | 0 |
| minio | 25 ops/s | 39.8ms | 39.8ms | 39.8ms | 0 |
| rustfs | 12 ops/s | 81.8ms | 81.8ms | 81.8ms | 0 |
| liteio_mem | 0 ops/s | 0ns | 0ns | 0ns | 1 |

```
  devnull      ████████████████████████████████████████ 6081 ops/s
  liteio       █ 151 ops/s
  seaweedfs    █ 78 ops/s
  localstack   █ 33 ops/s
  minio        █ 25 ops/s
  rustfs       █ 12 ops/s
  liteio_mem   █ 0 ops/s
```

### FileCount/List/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull | 578 ops/s | 1.7ms | 1.7ms | 1.7ms | 0 |
| seaweedfs | 5 ops/s | 208.2ms | 208.2ms | 208.2ms | 0 |
| liteio | 5 ops/s | 220.7ms | 220.7ms | 220.7ms | 0 |
| rustfs | 1 ops/s | 1.08s | 1.08s | 1.08s | 0 |
| minio | 0 ops/s | 0ns | 0ns | 0ns | 1 |
| localstack | 0 ops/s | 0ns | 0ns | 0ns | 1 |
| liteio_mem | 0 ops/s | 0ns | 0ns | 0ns | 1 |

```
  devnull      ████████████████████████████████████████ 578 ops/s
  seaweedfs    █ 5 ops/s
  liteio       █ 5 ops/s
  rustfs       █ 1 ops/s
  minio        █ 0 ops/s
  localstack   █ 0 ops/s
  liteio_mem   █ 0 ops/s
```

### FileCount/Write/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull | 384.17 MB/s | 2.5us | 2.5us | 2.5us | 0 |
| liteio_mem | 1.37 MB/s | 713.9us | 713.9us | 713.9us | 0 |
| liteio | 1.22 MB/s | 798.1us | 798.1us | 798.1us | 0 |
| minio | 0.88 MB/s | 1.1ms | 1.1ms | 1.1ms | 0 |
| rustfs | 0.83 MB/s | 1.2ms | 1.2ms | 1.2ms | 0 |
| seaweedfs | 0.65 MB/s | 1.5ms | 1.5ms | 1.5ms | 0 |
| localstack | 0.16 MB/s | 6.1ms | 6.1ms | 6.1ms | 0 |

```
  devnull      ████████████████████████████████████████ 384.17 MB/s
  liteio_mem   █ 1.37 MB/s
  liteio       █ 1.22 MB/s
  minio        █ 0.88 MB/s
  rustfs       █ 0.83 MB/s
  seaweedfs    █ 0.65 MB/s
  localstack   █ 0.16 MB/s
```

### FileCount/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull | 2092.48 MB/s | 4.7us | 4.7us | 4.7us | 0 |
| liteio_mem | 2.94 MB/s | 3.3ms | 3.3ms | 3.3ms | 0 |
| liteio | 1.36 MB/s | 7.2ms | 7.2ms | 7.2ms | 0 |
| rustfs | 0.80 MB/s | 12.2ms | 12.2ms | 12.2ms | 0 |
| seaweedfs | 0.68 MB/s | 14.3ms | 14.3ms | 14.3ms | 0 |
| minio | 0.64 MB/s | 15.3ms | 15.3ms | 15.3ms | 0 |
| localstack | 0.28 MB/s | 35.0ms | 35.0ms | 35.0ms | 0 |

```
  devnull      ████████████████████████████████████████ 2092.48 MB/s
  liteio_mem   █ 2.94 MB/s
  liteio       █ 1.36 MB/s
  rustfs       █ 0.80 MB/s
  seaweedfs    █ 0.68 MB/s
  minio        █ 0.64 MB/s
  localstack   █ 0.28 MB/s
```

### FileCount/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull | 1727.15 MB/s | 56.5us | 56.5us | 56.5us | 0 |
| liteio_mem | 2.90 MB/s | 33.7ms | 33.7ms | 33.7ms | 0 |
| liteio | 1.41 MB/s | 69.2ms | 69.2ms | 69.2ms | 0 |
| rustfs | 0.76 MB/s | 128.4ms | 128.4ms | 128.4ms | 0 |
| minio | 0.61 MB/s | 159.3ms | 159.3ms | 159.3ms | 0 |
| seaweedfs | 0.60 MB/s | 161.7ms | 161.7ms | 161.7ms | 0 |
| localstack | 0.47 MB/s | 209.8ms | 209.8ms | 209.8ms | 0 |

```
  devnull      ████████████████████████████████████████ 1727.15 MB/s
  liteio_mem   █ 2.90 MB/s
  liteio       █ 1.41 MB/s
  rustfs       █ 0.76 MB/s
  minio        █ 0.61 MB/s
  seaweedfs    █ 0.60 MB/s
  localstack   █ 0.47 MB/s
```

### FileCount/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull | 3037.92 MB/s | 321.5us | 321.5us | 321.5us | 0 |
| liteio_mem | 2.88 MB/s | 338.8ms | 338.8ms | 338.8ms | 0 |
| liteio | 1.36 MB/s | 717.9ms | 717.9ms | 717.9ms | 0 |
| rustfs | 0.80 MB/s | 1.22s | 1.22s | 1.22s | 0 |
| localstack | 0.72 MB/s | 1.35s | 1.35s | 1.35s | 0 |
| seaweedfs | 0.55 MB/s | 1.78s | 1.78s | 1.78s | 0 |
| minio | 0.27 MB/s | 3.61s | 3.61s | 3.61s | 0 |

```
  devnull      ████████████████████████████████████████ 3037.92 MB/s
  liteio_mem   █ 2.88 MB/s
  liteio       █ 1.36 MB/s
  rustfs       █ 0.80 MB/s
  localstack   █ 0.72 MB/s
  seaweedfs    █ 0.55 MB/s
  minio        █ 0.27 MB/s
```

### FileCount/Write/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull | 2488.98 MB/s | 3.9ms | 3.9ms | 3.9ms | 0 |
| liteio_mem | 2.98 MB/s | 3.28s | 3.28s | 3.28s | 0 |
| liteio | 1.00 MB/s | 9.80s | 9.80s | 9.80s | 0 |
| localstack | 0.67 MB/s | 14.65s | 14.65s | 14.65s | 0 |
| rustfs | 0.62 MB/s | 15.85s | 15.85s | 15.85s | 0 |
| seaweedfs | 0.50 MB/s | 19.61s | 19.61s | 19.61s | 0 |
| minio | 0.05 MB/s | 180.95s | 180.95s | 180.95s | 0 |

```
  devnull      ████████████████████████████████████████ 2488.98 MB/s
  liteio_mem   █ 2.98 MB/s
  liteio       █ 1.00 MB/s
  localstack   █ 0.67 MB/s
  rustfs       █ 0.62 MB/s
  seaweedfs    █ 0.50 MB/s
  minio        █ 0.05 MB/s
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull | 87051 ops/s | 11.3us | 13.6us | 18.4us | 0 |
| liteio | 601 ops/s | 1.6ms | 2.2ms | 2.2ms | 0 |
| liteio_mem | 435 ops/s | 1.8ms | 5.7ms | 7.5ms | 0 |
| seaweedfs | 414 ops/s | 2.2ms | 3.2ms | 5.0ms | 0 |
| localstack | 115 ops/s | 7.5ms | 12.8ms | 33.7ms | 0 |
| minio | 95 ops/s | 8.7ms | 21.1ms | 39.1ms | 0 |
| rustfs | 73 ops/s | 10.2ms | 30.8ms | 37.2ms | 0 |

```
  devnull      ████████████████████████████████████████ 87051 ops/s
  liteio       █ 601 ops/s
  liteio_mem   █ 435 ops/s
  seaweedfs    █ 414 ops/s
  localstack   █ 115 ops/s
  minio        █ 95 ops/s
  rustfs       █ 73 ops/s
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull | 15470.45 MB/s | 916ns | 2.2us | 6.7us | 0 |
| rustfs | 3.26 MB/s | 3.3ms | 16.8ms | 19.3ms | 0 |
| liteio_mem | 2.34 MB/s | 6.6ms | 11.4ms | 11.8ms | 0 |
| liteio | 1.48 MB/s | 9.9ms | 16.5ms | 17.7ms | 0 |
| seaweedfs | 0.59 MB/s | 29.0ms | 31.3ms | 31.3ms | 0 |
| minio | 0.10 MB/s | 168.3ms | 224.1ms | 227.5ms | 0 |
| localstack | 0.02 MB/s | 664.6ms | 668.8ms | 671.1ms | 0 |

```
  devnull      ████████████████████████████████████████ 15470.45 MB/s
  rustfs       █ 3.26 MB/s
  liteio_mem   █ 2.34 MB/s
  liteio       █ 1.48 MB/s
  seaweedfs    █ 0.59 MB/s
  minio        █ 0.10 MB/s
  localstack   █ 0.02 MB/s
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull | 30866.64 MB/s | 125ns | 2.1us | 7.7us | 0 |
| rustfs | 7.08 MB/s | 2.1ms | 3.5ms | 3.8ms | 0 |
| liteio | 2.07 MB/s | 6.9ms | 10.6ms | 11.0ms | 0 |
| liteio_mem | 1.79 MB/s | 8.1ms | 15.5ms | 16.9ms | 0 |
| seaweedfs | 0.51 MB/s | 31.4ms | 33.2ms | 33.9ms | 0 |
| minio | 0.12 MB/s | 130.1ms | 199.5ms | 206.5ms | 0 |
| localstack | 0.05 MB/s | 349.1ms | 397.2ms | 400.1ms | 0 |

```
  devnull      ████████████████████████████████████████ 30866.64 MB/s
  rustfs       █ 7.08 MB/s
  liteio       █ 2.07 MB/s
  liteio_mem   █ 1.79 MB/s
  seaweedfs    █ 0.51 MB/s
  minio        █ 0.12 MB/s
  localstack   █ 0.05 MB/s
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull | 7332.65 MB/s | 1.1us | 9.5us | 14.6us | 0 |
| liteio_mem | 3.90 MB/s | 3.5ms | 8.1ms | 11.4ms | 0 |
| rustfs | 3.58 MB/s | 4.2ms | 6.4ms | 8.5ms | 0 |
| liteio | 0.95 MB/s | 17.2ms | 20.3ms | 21.2ms | 0 |
| seaweedfs | 0.61 MB/s | 26.9ms | 32.2ms | 33.9ms | 0 |
| minio | 0.07 MB/s | 226.7ms | 249.5ms | 285.1ms | 0 |
| localstack | 0.00 MB/s | 17.62s | 17.64s | 17.64s | 0 |

```
  devnull      ████████████████████████████████████████ 7332.65 MB/s
  liteio_mem   █ 3.90 MB/s
  rustfs       █ 3.58 MB/s
  liteio       █ 0.95 MB/s
  seaweedfs    █ 0.61 MB/s
  minio        █ 0.07 MB/s
  localstack   █ 0.00 MB/s
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull | 54868.44 MB/s | 280.1us | 312.9us | 312.9us | 0 |
| liteio | 108.63 MB/s | 134.1ms | 154.2ms | 154.2ms | 0 |
| liteio_mem | 93.70 MB/s | 123.7ms | 328.5ms | 328.5ms | 0 |
| rustfs | 93.58 MB/s | 159.6ms | 182.9ms | 182.9ms | 0 |
| seaweedfs | 84.63 MB/s | 166.6ms | 272.0ms | 272.0ms | 0 |
| minio | 44.47 MB/s | 225.3ms | 762.8ms | 762.8ms | 0 |
| localstack | 17.97 MB/s | 616.7ms | 2.21s | 2.21s | 0 |

```
  devnull      ████████████████████████████████████████ 54868.44 MB/s
  liteio       █ 108.63 MB/s
  liteio_mem   █ 93.70 MB/s
  rustfs       █ 93.58 MB/s
  seaweedfs    █ 84.63 MB/s
  minio        █ 44.47 MB/s
  localstack   █ 17.97 MB/s
```

### ParallelRead/1KB/C1

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| devnull | 5920.71 MB/s | 0ns | 0ns | 125ns | 417ns | 1.0us | 0 |
| liteio | 2.08 MB/s | 469.4us | 693.0us | 428.7us | 693.5us | 823.2us | 0 |
| seaweedfs | 1.72 MB/s | 568.3us | 812.0us | 526.3us | 812.1us | 1.5ms | 0 |
| rustfs | 1.42 MB/s | 686.9us | 888.5us | 631.4us | 888.8us | 1.4ms | 0 |
| minio | 0.25 MB/s | 3.9ms | 8.5ms | 3.1ms | 8.5ms | 13.8ms | 0 |
| liteio_mem | 0.23 MB/s | 4.3ms | 15.1ms | 2.5ms | 15.1ms | 41.4ms | 0 |
| localstack | 0.16 MB/s | 6.2ms | 15.4ms | 4.3ms | 15.4ms | 18.5ms | 0 |

```
  devnull      ████████████████████████████████████████ 5920.71 MB/s
  liteio       █ 2.08 MB/s
  seaweedfs    █ 1.72 MB/s
  rustfs       █ 1.42 MB/s
  minio        █ 0.25 MB/s
  liteio_mem   █ 0.23 MB/s
  localstack   █ 0.16 MB/s
```

### ParallelRead/1KB/C10

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| devnull | 2393.95 MB/s | 0ns | 0ns | 167ns | 1.3us | 3.9us | 0 |
| liteio | 1.13 MB/s | 867.3us | 1.2ms | 827.6us | 1.2ms | 1.7ms | 0 |
| seaweedfs | 0.51 MB/s | 1.9ms | 2.7ms | 1.9ms | 2.7ms | 2.9ms | 0 |
| rustfs | 0.44 MB/s | 2.2ms | 5.0ms | 1.8ms | 5.0ms | 5.3ms | 0 |
| liteio_mem | 0.23 MB/s | 4.3ms | 9.8ms | 3.4ms | 9.8ms | 18.0ms | 0 |
| minio | 0.11 MB/s | 8.8ms | 14.1ms | 8.4ms | 14.1ms | 14.4ms | 0 |
| localstack | 0.04 MB/s | 22.2ms | 45.8ms | 19.3ms | 45.8ms | 53.0ms | 0 |

```
  devnull      ████████████████████████████████████████ 2393.95 MB/s
  liteio       █ 1.13 MB/s
  seaweedfs    █ 0.51 MB/s
  rustfs       █ 0.44 MB/s
  liteio_mem   █ 0.23 MB/s
  minio        █ 0.11 MB/s
  localstack   █ 0.04 MB/s
```

### ParallelRead/1KB/C100

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| devnull | 1617.44 MB/s | 0ns | 0ns | 167ns | 1.1us | 10.9us | 0 |
| liteio | 0.27 MB/s | 3.7ms | 4.4ms | 3.7ms | 4.4ms | 4.5ms | 0 |
| seaweedfs | 0.12 MB/s | 8.2ms | 10.2ms | 8.3ms | 10.2ms | 10.9ms | 0 |
| liteio_mem | 0.06 MB/s | 15.8ms | 26.1ms | 16.3ms | 26.1ms | 28.7ms | 0 |
| minio | 0.04 MB/s | 23.5ms | 35.7ms | 21.8ms | 35.7ms | 37.2ms | 0 |
| localstack | 0.00 MB/s | 1.11s | 1.23s | 1.20s | 1.23s | 1.23s | 0 |

```
  devnull      ████████████████████████████████████████ 1617.44 MB/s
  liteio       █ 0.27 MB/s
  seaweedfs    █ 0.12 MB/s
  liteio_mem   █ 0.06 MB/s
  minio        █ 0.04 MB/s
  localstack   █ 0.00 MB/s
```

### ParallelRead/1KB/C200

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| devnull | 1543.05 MB/s | 0ns | 0ns | 167ns | 1.1us | 13.0us | 0 |
| liteio | 0.21 MB/s | 4.7ms | 9.7ms | 3.9ms | 9.7ms | 10.2ms | 0 |
| seaweedfs | 0.17 MB/s | 5.9ms | 7.6ms | 6.1ms | 7.6ms | 7.6ms | 0 |
| minio | 0.07 MB/s | 13.1ms | 18.9ms | 13.0ms | 18.9ms | 20.7ms | 0 |
| liteio_mem | 0.04 MB/s | 21.8ms | 33.6ms | 22.7ms | 33.6ms | 35.3ms | 0 |
| localstack | 0.00 MB/s | 535.8ms | 606.2ms | 535.1ms | 606.2ms | 614.2ms | 0 |

```
  devnull      ████████████████████████████████████████ 1543.05 MB/s
  liteio       █ 0.21 MB/s
  seaweedfs    █ 0.17 MB/s
  minio        █ 0.07 MB/s
  liteio_mem   █ 0.04 MB/s
  localstack   █ 0.00 MB/s
```

### ParallelRead/1KB/C25

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| devnull | 862.58 MB/s | 0ns | 0ns | 209ns | 2.4us | 10.4us | 0 |
| liteio | 0.52 MB/s | 1.9ms | 2.6ms | 1.8ms | 2.6ms | 3.2ms | 0 |
| seaweedfs | 0.31 MB/s | 3.1ms | 4.5ms | 3.0ms | 4.5ms | 5.1ms | 0 |
| liteio_mem | 0.15 MB/s | 6.7ms | 14.2ms | 6.1ms | 14.2ms | 17.3ms | 0 |
| minio | 0.09 MB/s | 11.1ms | 15.4ms | 10.5ms | 15.4ms | 16.7ms | 0 |
| localstack | 0.01 MB/s | 81.8ms | 99.7ms | 90.0ms | 99.7ms | 104.0ms | 0 |

```
  devnull      ████████████████████████████████████████ 862.58 MB/s
  liteio       █ 0.52 MB/s
  seaweedfs    █ 0.31 MB/s
  liteio_mem   █ 0.15 MB/s
  minio        █ 0.09 MB/s
  localstack   █ 0.01 MB/s
```

### ParallelRead/1KB/C50

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| devnull | 2803.15 MB/s | 0ns | 0ns | 250ns | 958ns | 1.1us | 0 |
| liteio | 0.37 MB/s | 2.6ms | 3.4ms | 2.6ms | 3.4ms | 3.4ms | 0 |
| liteio_mem | 0.08 MB/s | 11.6ms | 25.7ms | 10.0ms | 25.7ms | 33.7ms | 0 |
| seaweedfs | 0.07 MB/s | 14.0ms | 24.8ms | 5.4ms | 24.8ms | 25.1ms | 0 |
| minio | 0.05 MB/s | 18.2ms | 22.7ms | 18.0ms | 22.7ms | 23.3ms | 0 |
| localstack | 0.01 MB/s | 170.2ms | 207.4ms | 155.9ms | 207.4ms | 208.2ms | 0 |

```
  devnull      ████████████████████████████████████████ 2803.15 MB/s
  liteio       █ 0.37 MB/s
  liteio_mem   █ 0.08 MB/s
  seaweedfs    █ 0.07 MB/s
  minio        █ 0.05 MB/s
  localstack   █ 0.01 MB/s
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull | 4170.14 MB/s | 167ns | 334ns | 1.3us | 0 |
| rustfs | 0.95 MB/s | 958.0us | 1.4ms | 2.0ms | 0 |
| seaweedfs | 0.92 MB/s | 988.0us | 1.4ms | 1.8ms | 0 |
| liteio | 0.72 MB/s | 1.3ms | 1.8ms | 2.2ms | 0 |
| liteio_mem | 0.19 MB/s | 4.2ms | 10.8ms | 21.7ms | 0 |
| localstack | 0.11 MB/s | 7.6ms | 19.8ms | 25.3ms | 0 |
| minio | 0.09 MB/s | 9.7ms | 18.1ms | 25.2ms | 0 |

```
  devnull      ████████████████████████████████████████ 4170.14 MB/s
  rustfs       █ 0.95 MB/s
  seaweedfs    █ 0.92 MB/s
  liteio       █ 0.72 MB/s
  liteio_mem   █ 0.19 MB/s
  localstack   █ 0.11 MB/s
  minio        █ 0.09 MB/s
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull | 1724.61 MB/s | 250ns | 1.2us | 9.2us | 0 |
| seaweedfs | 0.32 MB/s | 2.7ms | 5.4ms | 5.6ms | 0 |
| liteio | 0.30 MB/s | 3.3ms | 4.8ms | 5.9ms | 0 |
| rustfs | 0.21 MB/s | 3.2ms | 12.6ms | 18.3ms | 0 |
| liteio_mem | 0.10 MB/s | 3.5ms | 69.5ms | 104.7ms | 0 |
| minio | 0.04 MB/s | 20.4ms | 49.2ms | 53.6ms | 0 |
| localstack | 0.02 MB/s | 36.9ms | 182.6ms | 196.1ms | 0 |

```
  devnull      ████████████████████████████████████████ 1724.61 MB/s
  seaweedfs    █ 0.32 MB/s
  liteio       █ 0.30 MB/s
  rustfs       █ 0.21 MB/s
  liteio_mem   █ 0.10 MB/s
  minio        █ 0.04 MB/s
  localstack   █ 0.02 MB/s
```

### ParallelWrite/1KB/C100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull | 418.23 MB/s | 458ns | 12.5us | 31.0us | 0 |
| seaweedfs | 0.06 MB/s | 16.0ms | 20.0ms | 21.0ms | 0 |
| liteio | 0.05 MB/s | 19.2ms | 24.5ms | 25.8ms | 0 |
| liteio_mem | 0.02 MB/s | 26.4ms | 118.8ms | 134.1ms | 0 |
| minio | 0.01 MB/s | 106.4ms | 142.1ms | 148.1ms | 0 |
| localstack | 0.00 MB/s | 190.1ms | 281.2ms | 282.4ms | 0 |

```
  devnull      ████████████████████████████████████████ 418.23 MB/s
  seaweedfs    █ 0.06 MB/s
  liteio       █ 0.05 MB/s
  liteio_mem   █ 0.02 MB/s
  minio        █ 0.01 MB/s
  localstack   █ 0.00 MB/s
```

### ParallelWrite/1KB/C200

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull | 129.20 MB/s | 417ns | 47.5us | 76.8us | 0 |
| liteio | 0.07 MB/s | 14.0ms | 19.8ms | 20.1ms | 0 |
| liteio_mem | 0.06 MB/s | 15.5ms | 28.4ms | 31.5ms | 0 |
| seaweedfs | 0.06 MB/s | 15.5ms | 19.1ms | 19.3ms | 0 |
| minio | 0.01 MB/s | 81.0ms | 150.4ms | 155.4ms | 0 |
| localstack | 0.00 MB/s | 433.8ms | 437.8ms | 439.2ms | 0 |

```
  devnull      ████████████████████████████████████████ 129.20 MB/s
  liteio       █ 0.07 MB/s
  liteio_mem   █ 0.06 MB/s
  seaweedfs    █ 0.06 MB/s
  minio        █ 0.01 MB/s
  localstack   █ 0.00 MB/s
```

### ParallelWrite/1KB/C25

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull | 211.11 MB/s | 750ns | 35.5us | 41.9us | 0 |
| liteio | 0.17 MB/s | 5.6ms | 7.9ms | 8.1ms | 0 |
| seaweedfs | 0.13 MB/s | 7.2ms | 10.9ms | 11.8ms | 0 |
| liteio_mem | 0.08 MB/s | 8.9ms | 26.2ms | 43.5ms | 0 |
| minio | 0.02 MB/s | 41.3ms | 59.2ms | 68.7ms | 0 |
| localstack | 0.01 MB/s | 145.3ms | 351.3ms | 361.5ms | 0 |

```
  devnull      ████████████████████████████████████████ 211.11 MB/s
  liteio       █ 0.17 MB/s
  seaweedfs    █ 0.13 MB/s
  liteio_mem   █ 0.08 MB/s
  minio        █ 0.02 MB/s
  localstack   █ 0.01 MB/s
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull | 12.37 MB/s | 15.0us | 216.0us | 227.8us | 0 |
| seaweedfs | 0.09 MB/s | 9.5ms | 17.1ms | 18.2ms | 0 |
| liteio | 0.07 MB/s | 10.9ms | 29.6ms | 31.1ms | 0 |
| liteio_mem | 0.03 MB/s | 23.5ms | 90.0ms | 95.3ms | 0 |
| localstack | 0.01 MB/s | 107.6ms | 214.1ms | 214.8ms | 0 |
| minio | 0.01 MB/s | 107.8ms | 215.0ms | 258.4ms | 0 |

```
  devnull      ████████████████████████████████████████ 12.37 MB/s
  seaweedfs    █ 0.09 MB/s
  liteio       █ 0.07 MB/s
  liteio_mem   █ 0.03 MB/s
  localstack   █ 0.01 MB/s
  minio        █ 0.01 MB/s
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull | 2739425.82 MB/s | 83ns | 84ns | 1.1us | 0 |
| liteio | 132.18 MB/s | 1.8ms | 2.5ms | 2.9ms | 0 |
| liteio_mem | 103.23 MB/s | 2.2ms | 3.8ms | 5.0ms | 0 |
| seaweedfs | 102.79 MB/s | 2.3ms | 3.3ms | 4.0ms | 0 |
| rustfs | 49.35 MB/s | 4.5ms | 8.8ms | 11.6ms | 0 |
| localstack | 19.02 MB/s | 10.6ms | 28.0ms | 37.7ms | 0 |
| minio | 11.63 MB/s | 15.0ms | 48.0ms | 58.8ms | 0 |

```
  devnull      ████████████████████████████████████████ 2739425.82 MB/s
  liteio       █ 132.18 MB/s
  liteio_mem   █ 103.23 MB/s
  seaweedfs    █ 102.79 MB/s
  rustfs       █ 49.35 MB/s
  localstack   █ 19.02 MB/s
  minio        █ 11.63 MB/s
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull | 2607425.95 MB/s | 83ns | 125ns | 459ns | 0 |
| liteio | 117.60 MB/s | 2.1ms | 2.9ms | 3.0ms | 0 |
| liteio_mem | 100.54 MB/s | 2.4ms | 3.6ms | 4.6ms | 0 |
| rustfs | 44.27 MB/s | 4.9ms | 10.4ms | 15.1ms | 0 |
| localstack | 14.91 MB/s | 14.2ms | 28.9ms | 46.7ms | 0 |
| minio | 9.73 MB/s | 24.9ms | 49.0ms | 60.9ms | 0 |
| seaweedfs | 7.86 MB/s | 33.4ms | 64.5ms | 82.7ms | 0 |

```
  devnull      ████████████████████████████████████████ 2607425.95 MB/s
  liteio       █ 117.60 MB/s
  liteio_mem   █ 100.54 MB/s
  rustfs       █ 44.27 MB/s
  localstack   █ 14.91 MB/s
  minio        █ 9.73 MB/s
  seaweedfs    █ 7.86 MB/s
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull | 2738825.59 MB/s | 83ns | 84ns | 667ns | 0 |
| liteio | 107.85 MB/s | 2.0ms | 3.2ms | 6.6ms | 0 |
| liteio_mem | 90.76 MB/s | 2.3ms | 5.3ms | 7.3ms | 0 |
| rustfs | 47.78 MB/s | 4.4ms | 9.6ms | 12.4ms | 0 |
| seaweedfs | 12.28 MB/s | 19.6ms | 43.6ms | 53.8ms | 0 |
| minio | 10.12 MB/s | 21.1ms | 53.4ms | 66.5ms | 0 |
| localstack | 8.44 MB/s | 22.7ms | 72.7ms | 113.8ms | 0 |

```
  devnull      ████████████████████████████████████████ 2738825.59 MB/s
  liteio       █ 107.85 MB/s
  liteio_mem   █ 90.76 MB/s
  rustfs       █ 47.78 MB/s
  seaweedfs    █ 12.28 MB/s
  minio        █ 10.12 MB/s
  localstack   █ 8.44 MB/s
```

### Read/100MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| devnull | 187406296.85 MB/s | 0ns | 0ns | 334ns | 500ns | 500ns | 0 |
| liteio_mem | 158.88 MB/s | 1.2ms | 1.6ms | 563.3ms | 682.8ms | 682.8ms | 0 |
| liteio | 155.66 MB/s | 3.8ms | 3.9ms | 648.5ms | 652.9ms | 652.9ms | 0 |
| seaweedfs | 154.17 MB/s | 6.1ms | 5.5ms | 649.9ms | 681.7ms | 681.7ms | 0 |
| rustfs | 103.32 MB/s | 21.5ms | 15.6ms | 1.01s | 1.05s | 1.05s | 0 |
| localstack | 94.45 MB/s | 11.4ms | 13.5ms | 1.10s | 1.21s | 1.21s | 0 |
| minio | 39.41 MB/s | 15.4ms | 15.1ms | 2.34s | 2.55s | 2.55s | 1 |

```
  devnull      ████████████████████████████████████████ 187406296.85 MB/s
  liteio_mem   █ 158.88 MB/s
  liteio       █ 155.66 MB/s
  seaweedfs    █ 154.17 MB/s
  rustfs       █ 103.32 MB/s
  localstack   █ 94.45 MB/s
  minio        █ 39.41 MB/s
```

### Read/10MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| devnull | 52164840.90 MB/s | 0ns | 0ns | 125ns | 208ns | 208ns | 0 |
| minio | 212.73 MB/s | 1.3ms | 1.7ms | 46.0ms | 49.7ms | 49.7ms | 0 |
| liteio | 166.99 MB/s | 5.3ms | 9.4ms | 56.0ms | 66.7ms | 66.7ms | 0 |
| liteio_mem | 156.22 MB/s | 1.5ms | 1.9ms | 56.9ms | 77.9ms | 77.9ms | 0 |
| localstack | 125.62 MB/s | 3.2ms | 3.9ms | 75.0ms | 100.1ms | 100.1ms | 0 |
| seaweedfs | 110.67 MB/s | 5.4ms | 6.4ms | 83.2ms | 109.8ms | 109.8ms | 0 |
| rustfs | 89.49 MB/s | 24.7ms | 35.5ms | 102.3ms | 132.5ms | 132.5ms | 0 |

```
  devnull      ████████████████████████████████████████ 52164840.90 MB/s
  minio        █ 212.73 MB/s
  liteio       █ 166.99 MB/s
  liteio_mem   █ 156.22 MB/s
  localstack   █ 125.62 MB/s
  seaweedfs    █ 110.67 MB/s
  rustfs       █ 89.49 MB/s
```

### Read/1KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| devnull | 5673.73 MB/s | 0ns | 0ns | 125ns | 250ns | 667ns | 0 |
| liteio | 3.61 MB/s | 270.7us | 449.2us | 249.8us | 449.3us | 674.3us | 0 |
| liteio_mem | 1.38 MB/s | 707.8us | 1.4ms | 606.9us | 1.4ms | 1.9ms | 0 |
| seaweedfs | 1.14 MB/s | 854.7us | 1.4ms | 751.1us | 1.4ms | 2.0ms | 0 |
| rustfs | 0.99 MB/s | 983.1us | 1.5ms | 898.2us | 1.5ms | 2.3ms | 0 |
| localstack | 0.49 MB/s | 2.0ms | 4.4ms | 1.4ms | 4.4ms | 6.3ms | 0 |
| minio | 0.37 MB/s | 2.6ms | 5.7ms | 2.2ms | 5.7ms | 6.9ms | 0 |

```
  devnull      ████████████████████████████████████████ 5673.73 MB/s
  liteio       █ 3.61 MB/s
  liteio_mem   █ 1.38 MB/s
  seaweedfs    █ 1.14 MB/s
  rustfs       █ 0.99 MB/s
  localstack   █ 0.49 MB/s
  minio        █ 0.37 MB/s
```

### Read/1MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| devnull | 3379520.11 MB/s | 0ns | 0ns | 209ns | 292ns | 292ns | 0 |
| minio | 197.45 MB/s | 979.2us | 1.1ms | 5.0ms | 5.5ms | 5.5ms | 0 |
| localstack | 166.55 MB/s | 1.7ms | 2.2ms | 5.9ms | 6.4ms | 6.4ms | 0 |
| liteio_mem | 160.53 MB/s | 1.0ms | 2.0ms | 5.6ms | 8.0ms | 8.0ms | 0 |
| liteio | 152.08 MB/s | 1.2ms | 1.9ms | 6.0ms | 8.2ms | 8.2ms | 0 |
| seaweedfs | 105.03 MB/s | 2.3ms | 3.5ms | 9.2ms | 12.1ms | 12.1ms | 0 |
| rustfs | 69.84 MB/s | 7.2ms | 12.7ms | 12.5ms | 22.1ms | 22.1ms | 0 |

```
  devnull      ████████████████████████████████████████ 3379520.11 MB/s
  minio        █ 197.45 MB/s
  localstack   █ 166.55 MB/s
  liteio_mem   █ 160.53 MB/s
  liteio       █ 152.08 MB/s
  seaweedfs    █ 105.03 MB/s
  rustfs       █ 69.84 MB/s
```

### Read/64KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| devnull | 493524.95 MB/s | 0ns | 0ns | 125ns | 166ns | 417ns | 0 |
| minio | 83.57 MB/s | 443.2us | 497.2us | 714.3us | 932.8us | 995.0us | 0 |
| liteio | 64.96 MB/s | 674.6us | 1.4ms | 766.7us | 1.8ms | 2.2ms | 0 |
| rustfs | 47.59 MB/s | 1.1ms | 1.7ms | 1.2ms | 1.8ms | 2.1ms | 0 |
| liteio_mem | 45.24 MB/s | 1.0ms | 1.3ms | 1.1ms | 1.8ms | 4.0ms | 0 |
| localstack | 41.86 MB/s | 1.4ms | 1.8ms | 1.3ms | 1.9ms | 3.3ms | 0 |
| seaweedfs | 38.43 MB/s | 1.4ms | 2.3ms | 1.4ms | 2.3ms | 2.8ms | 0 |

```
  devnull      ████████████████████████████████████████ 493524.95 MB/s
  minio        █ 83.57 MB/s
  liteio       █ 64.96 MB/s
  rustfs       █ 47.59 MB/s
  liteio_mem   █ 45.24 MB/s
  localstack   █ 41.86 MB/s
  seaweedfs    █ 38.43 MB/s
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull | 19368584 ops/s | 41ns | 125ns | 459ns | 0 |
| liteio | 2434 ops/s | 351.1us | 712.6us | 861.3us | 0 |
| seaweedfs | 1767 ops/s | 466.9us | 1.2ms | 1.4ms | 0 |
| liteio_mem | 627 ops/s | 966.8us | 4.7ms | 5.7ms | 0 |
| rustfs | 545 ops/s | 1.7ms | 2.9ms | 3.7ms | 0 |
| localstack | 198 ops/s | 3.5ms | 12.0ms | 32.0ms | 0 |
| minio | 89 ops/s | 7.2ms | 26.8ms | 45.9ms | 0 |

```
  devnull      ████████████████████████████████████████ 19368584 ops/s
  liteio       █ 2434 ops/s
  seaweedfs    █ 1767 ops/s
  liteio_mem   █ 627 ops/s
  rustfs       █ 545 ops/s
  localstack   █ 198 ops/s
  minio        █ 89 ops/s
```

### Write/100MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull | 48957.81 MB/s | 2.0ms | 2.1ms | 2.1ms | 0 |
| liteio_mem | 116.92 MB/s | 852.7ms | 857.1ms | 857.1ms | 0 |
| seaweedfs | 85.68 MB/s | 1.16s | 1.21s | 1.21s | 0 |
| rustfs | 71.21 MB/s | 998.4ms | 1.44s | 1.44s | 0 |
| liteio | 68.79 MB/s | 1.76s | 1.89s | 1.89s | 0 |
| minio | 66.41 MB/s | 950.0ms | 2.48s | 2.48s | 0 |
| localstack | 42.63 MB/s | 2.35s | 2.38s | 2.38s | 0 |

```
  devnull      ████████████████████████████████████████ 48957.81 MB/s
  liteio_mem   █ 116.92 MB/s
  seaweedfs    █ 85.68 MB/s
  rustfs       █ 71.21 MB/s
  liteio       █ 68.79 MB/s
  minio        █ 66.41 MB/s
  localstack   █ 42.63 MB/s
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull | 51496.54 MB/s | 194.7us | 198.8us | 198.8us | 0 |
| liteio_mem | 168.27 MB/s | 59.3ms | 62.5ms | 62.5ms | 0 |
| liteio | 118.23 MB/s | 75.2ms | 112.5ms | 112.5ms | 0 |
| seaweedfs | 101.78 MB/s | 93.3ms | 117.7ms | 117.7ms | 0 |
| localstack | 98.89 MB/s | 100.5ms | 108.2ms | 108.2ms | 0 |
| rustfs | 95.67 MB/s | 102.3ms | 124.6ms | 124.6ms | 0 |
| minio | 70.56 MB/s | 139.3ms | 154.6ms | 154.6ms | 0 |

```
  devnull      ████████████████████████████████████████ 51496.54 MB/s
  liteio_mem   █ 168.27 MB/s
  liteio       █ 118.23 MB/s
  seaweedfs    █ 101.78 MB/s
  localstack   █ 98.89 MB/s
  rustfs       █ 95.67 MB/s
  minio        █ 70.56 MB/s
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull | 4148.52 MB/s | 166ns | 375ns | 2.4us | 0 |
| liteio_mem | 1.41 MB/s | 624.7us | 1.2ms | 1.9ms | 0 |
| liteio | 1.30 MB/s | 677.2us | 1.3ms | 1.7ms | 0 |
| rustfs | 0.69 MB/s | 1.2ms | 2.4ms | 3.5ms | 0 |
| minio | 0.65 MB/s | 1.0ms | 5.2ms | 6.4ms | 0 |
| seaweedfs | 0.63 MB/s | 1.3ms | 2.8ms | 3.6ms | 0 |
| localstack | 0.52 MB/s | 1.3ms | 5.5ms | 7.5ms | 0 |

```
  devnull      ████████████████████████████████████████ 4148.52 MB/s
  liteio_mem   █ 1.41 MB/s
  liteio       █ 1.30 MB/s
  rustfs       █ 0.69 MB/s
  minio        █ 0.65 MB/s
  seaweedfs    █ 0.63 MB/s
  localstack   █ 0.52 MB/s
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull | 52236.64 MB/s | 19.1us | 19.5us | 19.5us | 0 |
| liteio_mem | 157.04 MB/s | 6.1ms | 8.3ms | 8.3ms | 0 |
| liteio | 125.19 MB/s | 7.9ms | 9.0ms | 9.0ms | 0 |
| seaweedfs | 83.89 MB/s | 11.0ms | 14.7ms | 14.7ms | 0 |
| localstack | 81.41 MB/s | 10.6ms | 14.8ms | 14.8ms | 0 |
| rustfs | 76.46 MB/s | 12.6ms | 16.3ms | 16.3ms | 0 |
| minio | 2.86 MB/s | 367.1ms | 493.2ms | 493.2ms | 0 |

```
  devnull      ████████████████████████████████████████ 52236.64 MB/s
  liteio_mem   █ 157.04 MB/s
  liteio       █ 125.19 MB/s
  seaweedfs    █ 83.89 MB/s
  localstack   █ 81.41 MB/s
  rustfs       █ 76.46 MB/s
  minio        █ 2.86 MB/s
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull | 69126.46 MB/s | 875ns | 1.0us | 1.2us | 0 |
| liteio | 42.87 MB/s | 1.3ms | 2.6ms | 2.7ms | 0 |
| liteio_mem | 37.73 MB/s | 991.7us | 4.5ms | 5.6ms | 0 |
| localstack | 37.20 MB/s | 1.5ms | 2.5ms | 3.0ms | 0 |
| rustfs | 30.27 MB/s | 2.0ms | 2.6ms | 2.9ms | 0 |
| seaweedfs | 25.10 MB/s | 2.1ms | 3.9ms | 5.2ms | 0 |
| minio | 1.52 MB/s | 3.0ms | 163.8ms | 199.7ms | 0 |

```
  devnull      ████████████████████████████████████████ 69126.46 MB/s
  liteio       █ 42.87 MB/s
  liteio_mem   █ 37.73 MB/s
  localstack   █ 37.20 MB/s
  rustfs       █ 30.27 MB/s
  seaweedfs    █ 25.10 MB/s
  minio        █ 1.52 MB/s
```

## Recommendations

- **Best for write-heavy workloads:** devnull
- **Best for read-heavy workloads:** devnull

---

*Report generated by storage benchmark CLI*
