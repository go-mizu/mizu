# Storage Benchmark Report

**Generated:** 2026-01-15T02:06:20+07:00

**Go Version:** go1.25.5

**Platform:** darwin/arm64

## Executive Summary

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (100MB+) | **seaweedfs** | 197 MB/s | Best for media, backups |
| Large File Downloads (100MB) | **minio** | 331 MB/s | Best for streaming, CDN |
| Small File Operations | **liteio_mem** | 3201 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **liteio** | - | Best for multi-user apps |

### Large File Performance (100MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| liteio | 194.4 | 271.3 | 512.5ms | 369.4ms |
| liteio_mem | 170.9 | 247.2 | 556.6ms | 399.6ms |
| localstack | 154.6 | 272.6 | 652.5ms | 364.8ms |
| minio | 163.4 | 331.2 | 637.0ms | 301.9ms |
| rustfs | 184.0 | 319.9 | 518.3ms | 309.6ms |
| seaweedfs | 197.3 | 278.8 | 503.9ms | 348.6ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| liteio | 1367 | 4931 | 695.2us | 203.2us |
| liteio_mem | 1113 | 5289 | 715.2us | 184.9us |
| localstack | 1371 | 1369 | 712.6us | 718.2us |
| minio | 1053 | 2742 | 863.8us | 361.2us |
| rustfs | 1371 | 2203 | 712.2us | 447.0us |
| seaweedfs | 1486 | 2455 | 661.9us | 408.2us |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| liteio | 5065 | 1313 | 5268 |
| liteio_mem | 4346 | 1250 | 5610 |
| localstack | 1630 | 396 | 1666 |
| minio | 4257 | 674 | 3014 |
| rustfs | 3296 | 157 | 1240 |
| seaweedfs | 3691 | 699 | 3182 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| liteio | 1.39 | 0.30 | 0.16 | 0.09 | 0.04 | 0.06 |
| liteio_mem | 1.27 | 0.32 | 0.16 | 0.08 | 0.04 | 0.04 |
| localstack | 1.18 | 0.19 | 0.08 | 0.04 | 0.03 | 0.02 |
| minio | 1.08 | 0.36 | 0.19 | 0.10 | 0.06 | 0.07 |
| rustfs | 1.20 | 0.39 | - | - | - | - |
| seaweedfs | 1.40 | 0.48 | 0.26 | 0.19 | 0.11 | 0.04 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| liteio | 4.75 | 1.18 | 0.43 | 0.34 | 0.28 | 0.38 |
| liteio_mem | 4.68 | 0.93 | 0.40 | 0.29 | 0.20 | 0.18 |
| localstack | 1.23 | 0.19 | 0.07 | 0.04 | 0.02 | 0.01 |
| minio | 2.88 | 1.08 | 0.59 | 0.38 | 0.12 | 0.18 |
| rustfs | 1.56 | 1.00 | - | - | - | - |
| seaweedfs | 2.30 | 0.91 | 0.50 | 0.32 | 0.19 | 0.18 |

*\* indicates errors occurred*

### File Count Performance

Performance with varying numbers of files (1KB each).

**Write N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|------|
| liteio | 888.0us | 8.9ms | 70.8ms | 692.0ms | 6.61s |
| liteio_mem | 732.7us | 6.5ms | 65.7ms | 714.2ms | 6.58s |
| localstack | 749.7us | 7.3ms | 78.1ms | 756.9ms | 8.08s |
| minio | 1.0ms | 7.7ms | 74.4ms | 780.0ms | 8.54s |
| rustfs | 890.5us | 8.2ms | 71.3ms | 677.9ms | 7.00s |
| seaweedfs | 734.6us | 6.8ms | 71.2ms | 691.7ms | 7.33s |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|------|
| liteio | 325.9us | 2.0ms | 933.2us | 5.5ms | 182.4ms |
| liteio_mem | 261.0us | 364.6us | 1.1ms | 5.7ms | 185.8ms |
| localstack | 962.9us | 1.2ms | 4.0ms | 25.4ms | 304.6ms |
| minio | 557.0us | 670.4us | 3.1ms | 17.5ms | 179.1ms |
| rustfs | 1.5ms | 2.5ms | 7.7ms | 61.3ms | 756.9ms |
| seaweedfs | 698.6us | 770.0us | 1.9ms | 12.0ms | 100.0ms |

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

---

## Configuration

| Parameter | Value |
|-----------|-------|
| Iterations | 100 |
| Warmup | 10 |
| Concurrency | 200 |
| Timeout | 30s |

## Drivers Tested

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
| liteio_mem | 1.51 MB/s | 583.9us | 995.6us | 1.4ms | 0 |
| liteio | 1.44 MB/s | 617.7us | 947.4us | 1.9ms | 0 |
| localstack | 1.29 MB/s | 731.5us | 864.6us | 1.1ms | 0 |
| minio | 1.14 MB/s | 815.3us | 1.1ms | 1.3ms | 0 |
| rustfs | 1.04 MB/s | 923.0us | 1.0ms | 1.0ms | 0 |
| seaweedfs | 0.84 MB/s | 1.1ms | 1.6ms | 2.2ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 1.51 MB/s
  liteio       ██████████████████████████████████████ 1.44 MB/s
  localstack   ██████████████████████████████████ 1.29 MB/s
  minio        ██████████████████████████████ 1.14 MB/s
  rustfs       ███████████████████████████ 1.04 MB/s
  seaweedfs    ██████████████████████ 0.84 MB/s
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 5610 ops/s | 159.9us | 265.2us | 423.0us | 0 |
| liteio | 5268 ops/s | 167.8us | 320.9us | 467.3us | 0 |
| seaweedfs | 3182 ops/s | 310.5us | 365.1us | 424.5us | 0 |
| minio | 3014 ops/s | 331.1us | 372.9us | 456.9us | 0 |
| localstack | 1666 ops/s | 588.5us | 646.9us | 781.8us | 0 |
| rustfs | 1240 ops/s | 773.6us | 855.0us | 912.1us | 0 |

```
  liteio_mem   ████████████████████████████████████████ 5610 ops/s
  liteio       █████████████████████████████████████ 5268 ops/s
  seaweedfs    ██████████████████████ 3182 ops/s
  minio        █████████████████████ 3014 ops/s
  localstack   ███████████ 1666 ops/s
  rustfs       ████████ 1240 ops/s
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.15 MB/s | 604.6us | 816.6us | 949.2us | 0 |
| liteio_mem | 0.15 MB/s | 599.1us | 1.0ms | 1.2ms | 0 |
| seaweedfs | 0.13 MB/s | 706.5us | 783.1us | 938.1us | 0 |
| localstack | 0.13 MB/s | 710.8us | 843.9us | 985.0us | 0 |
| rustfs | 0.12 MB/s | 687.0us | 1.1ms | 1.3ms | 0 |
| minio | 0.05 MB/s | 726.8us | 1.1ms | 1.3ms | 0 |

```
  liteio       ████████████████████████████████████████ 0.15 MB/s
  liteio_mem   ███████████████████████████████████████ 0.15 MB/s
  seaweedfs    ███████████████████████████████████ 0.13 MB/s
  localstack   ███████████████████████████████████ 0.13 MB/s
  rustfs       ████████████████████████████████ 0.12 MB/s
  minio        ████████████ 0.05 MB/s
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 2465 ops/s | 392.2us | 465.2us | 490.1us | 0 |
| liteio_mem | 1591 ops/s | 582.6us | 872.5us | 907.2us | 0 |
| liteio | 1566 ops/s | 575.1us | 916.6us | 1.2ms | 0 |
| rustfs | 1382 ops/s | 701.7us | 817.5us | 827.9us | 0 |
| localstack | 1337 ops/s | 724.6us | 903.0us | 1.1ms | 0 |
| minio | 1307 ops/s | 737.0us | 938.1us | 970.1us | 0 |

```
  seaweedfs    ████████████████████████████████████████ 2465 ops/s
  liteio_mem   █████████████████████████ 1591 ops/s
  liteio       █████████████████████████ 1566 ops/s
  rustfs       ██████████████████████ 1382 ops/s
  localstack   █████████████████████ 1337 ops/s
  minio        █████████████████████ 1307 ops/s
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 0.14 MB/s | 632.0us | 1.0ms | 1.2ms | 0 |
| rustfs | 0.13 MB/s | 711.3us | 822.9us | 895.9us | 0 |
| liteio | 0.13 MB/s | 673.4us | 1.1ms | 1.4ms | 0 |
| seaweedfs | 0.13 MB/s | 727.9us | 802.2us | 852.7us | 0 |
| localstack | 0.13 MB/s | 721.2us | 809.2us | 1.1ms | 0 |
| minio | 0.12 MB/s | 761.8us | 1.0ms | 1.1ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 0.14 MB/s
  rustfs       █████████████████████████████████████ 0.13 MB/s
  liteio       █████████████████████████████████████ 0.13 MB/s
  seaweedfs    █████████████████████████████████████ 0.13 MB/s
  localstack   ████████████████████████████████████ 0.13 MB/s
  minio        ██████████████████████████████████ 0.12 MB/s
```

### FileCount/Delete/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 4586 ops/s | 218.0us | 218.0us | 218.0us | 0 |
| liteio | 3974 ops/s | 251.7us | 251.7us | 251.7us | 0 |
| seaweedfs | 2551 ops/s | 392.0us | 392.0us | 392.0us | 0 |
| minio | 2301 ops/s | 434.5us | 434.5us | 434.5us | 0 |
| localstack | 1493 ops/s | 669.7us | 669.7us | 669.7us | 0 |
| rustfs | 859 ops/s | 1.2ms | 1.2ms | 1.2ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 4586 ops/s
  liteio       ██████████████████████████████████ 3974 ops/s
  seaweedfs    ██████████████████████ 2551 ops/s
  minio        ████████████████████ 2301 ops/s
  localstack   █████████████ 1493 ops/s
  rustfs       ███████ 859 ops/s
```

### FileCount/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 345 ops/s | 2.9ms | 2.9ms | 2.9ms | 0 |
| seaweedfs | 274 ops/s | 3.6ms | 3.6ms | 3.6ms | 0 |
| liteio_mem | 266 ops/s | 3.8ms | 3.8ms | 3.8ms | 0 |
| minio | 235 ops/s | 4.3ms | 4.3ms | 4.3ms | 0 |
| localstack | 157 ops/s | 6.4ms | 6.4ms | 6.4ms | 0 |
| rustfs | 67 ops/s | 14.9ms | 14.9ms | 14.9ms | 0 |

```
  liteio       ████████████████████████████████████████ 345 ops/s
  seaweedfs    ███████████████████████████████ 274 ops/s
  liteio_mem   ██████████████████████████████ 266 ops/s
  minio        ███████████████████████████ 235 ops/s
  localstack   ██████████████████ 157 ops/s
  rustfs       ███████ 67 ops/s
```

### FileCount/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 45 ops/s | 22.2ms | 22.2ms | 22.2ms | 0 |
| liteio | 42 ops/s | 23.7ms | 23.7ms | 23.7ms | 0 |
| seaweedfs | 30 ops/s | 33.7ms | 33.7ms | 33.7ms | 0 |
| minio | 25 ops/s | 40.0ms | 40.0ms | 40.0ms | 0 |
| localstack | 16 ops/s | 62.0ms | 62.0ms | 62.0ms | 0 |
| rustfs | 11 ops/s | 87.5ms | 87.5ms | 87.5ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 45 ops/s
  liteio       █████████████████████████████████████ 42 ops/s
  seaweedfs    ██████████████████████████ 30 ops/s
  minio        ██████████████████████ 25 ops/s
  localstack   ██████████████ 16 ops/s
  rustfs       ██████████ 11 ops/s
```

### FileCount/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 5 ops/s | 211.6ms | 211.6ms | 211.6ms | 0 |
| liteio | 4 ops/s | 228.5ms | 228.5ms | 228.5ms | 0 |
| seaweedfs | 3 ops/s | 318.6ms | 318.6ms | 318.6ms | 0 |
| minio | 3 ops/s | 355.2ms | 355.2ms | 355.2ms | 0 |
| localstack | 2 ops/s | 610.5ms | 610.5ms | 610.5ms | 0 |
| rustfs | 1 ops/s | 879.2ms | 879.2ms | 879.2ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 5 ops/s
  liteio       █████████████████████████████████████ 4 ops/s
  seaweedfs    ██████████████████████████ 3 ops/s
  minio        ███████████████████████ 3 ops/s
  localstack   █████████████ 2 ops/s
  rustfs       █████████ 1 ops/s
```

### FileCount/Delete/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1 ops/s | 1.93s | 1.93s | 1.93s | 0 |
| liteio | 1 ops/s | 1.96s | 1.96s | 1.96s | 0 |
| seaweedfs | 0 ops/s | 3.24s | 3.24s | 3.24s | 0 |
| minio | 0 ops/s | 3.62s | 3.62s | 3.62s | 0 |
| localstack | 0 ops/s | 6.50s | 6.50s | 6.50s | 0 |
| rustfs | 0 ops/s | 8.43s | 8.43s | 8.43s | 0 |

```
  liteio_mem   ████████████████████████████████████████ 1 ops/s
  liteio       ███████████████████████████████████████ 1 ops/s
  seaweedfs    ███████████████████████ 0 ops/s
  minio        █████████████████████ 0 ops/s
  localstack   ███████████ 0 ops/s
  rustfs       █████████ 0 ops/s
```

### FileCount/List/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 3832 ops/s | 261.0us | 261.0us | 261.0us | 0 |
| liteio | 3068 ops/s | 325.9us | 325.9us | 325.9us | 0 |
| minio | 1795 ops/s | 557.0us | 557.0us | 557.0us | 0 |
| seaweedfs | 1431 ops/s | 698.6us | 698.6us | 698.6us | 0 |
| localstack | 1039 ops/s | 962.9us | 962.9us | 962.9us | 0 |
| rustfs | 689 ops/s | 1.5ms | 1.5ms | 1.5ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 3832 ops/s
  liteio       ████████████████████████████████ 3068 ops/s
  minio        ██████████████████ 1795 ops/s
  seaweedfs    ██████████████ 1431 ops/s
  localstack   ██████████ 1039 ops/s
  rustfs       ███████ 689 ops/s
```

### FileCount/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 2743 ops/s | 364.6us | 364.6us | 364.6us | 0 |
| minio | 1492 ops/s | 670.4us | 670.4us | 670.4us | 0 |
| seaweedfs | 1299 ops/s | 770.0us | 770.0us | 770.0us | 0 |
| localstack | 803 ops/s | 1.2ms | 1.2ms | 1.2ms | 0 |
| liteio | 504 ops/s | 2.0ms | 2.0ms | 2.0ms | 0 |
| rustfs | 396 ops/s | 2.5ms | 2.5ms | 2.5ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 2743 ops/s
  minio        █████████████████████ 1492 ops/s
  seaweedfs    ██████████████████ 1299 ops/s
  localstack   ███████████ 803 ops/s
  liteio       ███████ 504 ops/s
  rustfs       █████ 396 ops/s
```

### FileCount/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1072 ops/s | 933.2us | 933.2us | 933.2us | 0 |
| liteio_mem | 889 ops/s | 1.1ms | 1.1ms | 1.1ms | 0 |
| seaweedfs | 520 ops/s | 1.9ms | 1.9ms | 1.9ms | 0 |
| minio | 319 ops/s | 3.1ms | 3.1ms | 3.1ms | 0 |
| localstack | 247 ops/s | 4.0ms | 4.0ms | 4.0ms | 0 |
| rustfs | 130 ops/s | 7.7ms | 7.7ms | 7.7ms | 0 |

```
  liteio       ████████████████████████████████████████ 1072 ops/s
  liteio_mem   █████████████████████████████████ 889 ops/s
  seaweedfs    ███████████████████ 520 ops/s
  minio        ███████████ 319 ops/s
  localstack   █████████ 247 ops/s
  rustfs       ████ 130 ops/s
```

### FileCount/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 181 ops/s | 5.5ms | 5.5ms | 5.5ms | 0 |
| liteio_mem | 177 ops/s | 5.7ms | 5.7ms | 5.7ms | 0 |
| seaweedfs | 84 ops/s | 12.0ms | 12.0ms | 12.0ms | 0 |
| minio | 57 ops/s | 17.5ms | 17.5ms | 17.5ms | 0 |
| localstack | 39 ops/s | 25.4ms | 25.4ms | 25.4ms | 0 |
| rustfs | 16 ops/s | 61.3ms | 61.3ms | 61.3ms | 0 |

```
  liteio       ████████████████████████████████████████ 181 ops/s
  liteio_mem   ███████████████████████████████████████ 177 ops/s
  seaweedfs    ██████████████████ 84 ops/s
  minio        ████████████ 57 ops/s
  localstack   ████████ 39 ops/s
  rustfs       ███ 16 ops/s
```

### FileCount/List/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 10 ops/s | 100.0ms | 100.0ms | 100.0ms | 0 |
| minio | 6 ops/s | 179.1ms | 179.1ms | 179.1ms | 0 |
| liteio | 5 ops/s | 182.4ms | 182.4ms | 182.4ms | 0 |
| liteio_mem | 5 ops/s | 185.8ms | 185.8ms | 185.8ms | 0 |
| localstack | 3 ops/s | 304.6ms | 304.6ms | 304.6ms | 0 |
| rustfs | 1 ops/s | 756.9ms | 756.9ms | 756.9ms | 0 |

```
  seaweedfs    ████████████████████████████████████████ 10 ops/s
  minio        ██████████████████████ 6 ops/s
  liteio       █████████████████████ 5 ops/s
  liteio_mem   █████████████████████ 5 ops/s
  localstack   █████████████ 3 ops/s
  rustfs       █████ 1 ops/s
```

### FileCount/Write/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1.33 MB/s | 732.7us | 732.7us | 732.7us | 0 |
| seaweedfs | 1.33 MB/s | 734.6us | 734.6us | 734.6us | 0 |
| localstack | 1.30 MB/s | 749.7us | 749.7us | 749.7us | 0 |
| liteio | 1.10 MB/s | 888.0us | 888.0us | 888.0us | 0 |
| rustfs | 1.10 MB/s | 890.5us | 890.5us | 890.5us | 0 |
| minio | 0.95 MB/s | 1.0ms | 1.0ms | 1.0ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 1.33 MB/s
  seaweedfs    ███████████████████████████████████████ 1.33 MB/s
  localstack   ███████████████████████████████████████ 1.30 MB/s
  liteio       █████████████████████████████████ 1.10 MB/s
  rustfs       ████████████████████████████████ 1.10 MB/s
  minio        ████████████████████████████ 0.95 MB/s
```

### FileCount/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1.51 MB/s | 6.5ms | 6.5ms | 6.5ms | 0 |
| seaweedfs | 1.43 MB/s | 6.8ms | 6.8ms | 6.8ms | 0 |
| localstack | 1.33 MB/s | 7.3ms | 7.3ms | 7.3ms | 0 |
| minio | 1.27 MB/s | 7.7ms | 7.7ms | 7.7ms | 0 |
| rustfs | 1.19 MB/s | 8.2ms | 8.2ms | 8.2ms | 0 |
| liteio | 1.10 MB/s | 8.9ms | 8.9ms | 8.9ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 1.51 MB/s
  seaweedfs    █████████████████████████████████████ 1.43 MB/s
  localstack   ███████████████████████████████████ 1.33 MB/s
  minio        █████████████████████████████████ 1.27 MB/s
  rustfs       ███████████████████████████████ 1.19 MB/s
  liteio       █████████████████████████████ 1.10 MB/s
```

### FileCount/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1.49 MB/s | 65.7ms | 65.7ms | 65.7ms | 0 |
| liteio | 1.38 MB/s | 70.8ms | 70.8ms | 70.8ms | 0 |
| seaweedfs | 1.37 MB/s | 71.2ms | 71.2ms | 71.2ms | 0 |
| rustfs | 1.37 MB/s | 71.3ms | 71.3ms | 71.3ms | 0 |
| minio | 1.31 MB/s | 74.4ms | 74.4ms | 74.4ms | 0 |
| localstack | 1.25 MB/s | 78.1ms | 78.1ms | 78.1ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 1.49 MB/s
  liteio       █████████████████████████████████████ 1.38 MB/s
  seaweedfs    ████████████████████████████████████ 1.37 MB/s
  rustfs       ████████████████████████████████████ 1.37 MB/s
  minio        ███████████████████████████████████ 1.31 MB/s
  localstack   █████████████████████████████████ 1.25 MB/s
```

### FileCount/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 1.44 MB/s | 677.9ms | 677.9ms | 677.9ms | 0 |
| seaweedfs | 1.41 MB/s | 691.7ms | 691.7ms | 691.7ms | 0 |
| liteio | 1.41 MB/s | 692.0ms | 692.0ms | 692.0ms | 0 |
| liteio_mem | 1.37 MB/s | 714.2ms | 714.2ms | 714.2ms | 0 |
| localstack | 1.29 MB/s | 756.9ms | 756.9ms | 756.9ms | 0 |
| minio | 1.25 MB/s | 780.0ms | 780.0ms | 780.0ms | 0 |

```
  rustfs       ████████████████████████████████████████ 1.44 MB/s
  seaweedfs    ███████████████████████████████████████ 1.41 MB/s
  liteio       ███████████████████████████████████████ 1.41 MB/s
  liteio_mem   █████████████████████████████████████ 1.37 MB/s
  localstack   ███████████████████████████████████ 1.29 MB/s
  minio        ██████████████████████████████████ 1.25 MB/s
```

### FileCount/Write/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1.49 MB/s | 6.58s | 6.58s | 6.58s | 0 |
| liteio | 1.48 MB/s | 6.61s | 6.61s | 6.61s | 0 |
| rustfs | 1.40 MB/s | 7.00s | 7.00s | 7.00s | 0 |
| seaweedfs | 1.33 MB/s | 7.33s | 7.33s | 7.33s | 0 |
| localstack | 1.21 MB/s | 8.08s | 8.08s | 8.08s | 0 |
| minio | 1.14 MB/s | 8.54s | 8.54s | 8.54s | 0 |

```
  liteio_mem   ████████████████████████████████████████ 1.49 MB/s
  liteio       ███████████████████████████████████████ 1.48 MB/s
  rustfs       █████████████████████████████████████ 1.40 MB/s
  seaweedfs    ███████████████████████████████████ 1.33 MB/s
  localstack   ████████████████████████████████ 1.21 MB/s
  minio        ██████████████████████████████ 1.14 MB/s
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1313 ops/s | 718.5us | 979.9us | 1.7ms | 0 |
| liteio_mem | 1250 ops/s | 755.9us | 997.3us | 2.1ms | 0 |
| seaweedfs | 699 ops/s | 1.4ms | 1.6ms | 3.1ms | 0 |
| minio | 674 ops/s | 1.5ms | 1.6ms | 1.7ms | 0 |
| localstack | 396 ops/s | 2.5ms | 2.9ms | 3.4ms | 0 |
| rustfs | 157 ops/s | 6.4ms | 7.0ms | 7.3ms | 0 |

```
  liteio       ████████████████████████████████████████ 1313 ops/s
  liteio_mem   ██████████████████████████████████████ 1250 ops/s
  seaweedfs    █████████████████████ 699 ops/s
  minio        ████████████████████ 674 ops/s
  localstack   ████████████ 396 ops/s
  rustfs       ████ 157 ops/s
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 8.81 MB/s | 1.7ms | 2.9ms | 3.1ms | 0 |
| minio | 2.15 MB/s | 5.6ms | 13.8ms | 14.7ms | 0 |
| seaweedfs | 1.58 MB/s | 10.3ms | 13.4ms | 14.3ms | 0 |
| liteio | 1.07 MB/s | 13.0ms | 24.4ms | 25.5ms | 0 |
| liteio_mem | 0.93 MB/s | 15.5ms | 27.3ms | 27.6ms | 0 |
| localstack | 0.36 MB/s | 51.7ms | 55.2ms | 55.3ms | 0 |

```
  rustfs       ████████████████████████████████████████ 8.81 MB/s
  minio        █████████ 2.15 MB/s
  seaweedfs    ███████ 1.58 MB/s
  liteio       ████ 1.07 MB/s
  liteio_mem   ████ 0.93 MB/s
  localstack   █ 0.36 MB/s
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 11.62 MB/s | 1.2ms | 1.9ms | 2.5ms | 0 |
| minio | 3.18 MB/s | 5.3ms | 8.0ms | 8.1ms | 0 |
| liteio | 1.91 MB/s | 8.0ms | 13.4ms | 14.1ms | 0 |
| seaweedfs | 1.42 MB/s | 11.9ms | 12.5ms | 12.9ms | 0 |
| liteio_mem | 1.31 MB/s | 12.1ms | 19.1ms | 19.8ms | 0 |
| localstack | 0.30 MB/s | 50.7ms | 59.0ms | 59.8ms | 0 |

```
  rustfs       ████████████████████████████████████████ 11.62 MB/s
  minio        ██████████ 3.18 MB/s
  liteio       ██████ 1.91 MB/s
  seaweedfs    ████ 1.42 MB/s
  liteio_mem   ████ 1.31 MB/s
  localstack   █ 0.30 MB/s
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 6.42 MB/s | 2.4ms | 3.5ms | 3.9ms | 0 |
| seaweedfs | 1.25 MB/s | 13.1ms | 15.5ms | 15.8ms | 0 |
| minio | 0.93 MB/s | 17.2ms | 23.4ms | 24.1ms | 0 |
| liteio | 0.71 MB/s | 22.9ms | 33.1ms | 34.4ms | 0 |
| liteio_mem | 0.70 MB/s | 21.6ms | 36.6ms | 37.8ms | 0 |
| localstack | 0.26 MB/s | 65.6ms | 66.9ms | 67.6ms | 0 |

```
  rustfs       ████████████████████████████████████████ 6.42 MB/s
  seaweedfs    ███████ 1.25 MB/s
  minio        █████ 0.93 MB/s
  liteio       ████ 0.71 MB/s
  liteio_mem   ████ 0.70 MB/s
  localstack   █ 0.26 MB/s
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 183.91 MB/s | 78.2ms | 91.7ms | 91.7ms | 0 |
| liteio_mem | 158.46 MB/s | 94.2ms | 98.8ms | 98.8ms | 0 |
| minio | 157.68 MB/s | 95.5ms | 105.1ms | 105.1ms | 0 |
| liteio | 156.30 MB/s | 94.1ms | 102.9ms | 102.9ms | 0 |
| seaweedfs | 132.04 MB/s | 112.7ms | 120.1ms | 120.1ms | 0 |
| localstack | 129.66 MB/s | 114.1ms | 124.2ms | 124.2ms | 0 |

```
  rustfs       ████████████████████████████████████████ 183.91 MB/s
  liteio_mem   ██████████████████████████████████ 158.46 MB/s
  minio        ██████████████████████████████████ 157.68 MB/s
  liteio       █████████████████████████████████ 156.30 MB/s
  seaweedfs    ████████████████████████████ 132.04 MB/s
  localstack   ████████████████████████████ 129.66 MB/s
```

### ParallelRead/1KB/C1

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 4.75 MB/s | 204.5us | 254.0us | 197.2us | 254.1us | 311.7us | 0 |
| liteio_mem | 4.68 MB/s | 206.1us | 242.0us | 203.4us | 244.2us | 312.8us | 0 |
| minio | 2.88 MB/s | 339.5us | 376.4us | 336.5us | 376.5us | 392.7us | 0 |
| seaweedfs | 2.30 MB/s | 424.6us | 456.4us | 420.5us | 456.5us | 501.0us | 0 |
| rustfs | 1.56 MB/s | 625.9us | 1.1ms | 559.2us | 1.1ms | 1.4ms | 0 |
| localstack | 1.23 MB/s | 792.5us | 915.0us | 780.7us | 915.8us | 1.1ms | 0 |

```
  liteio       ████████████████████████████████████████ 4.75 MB/s
  liteio_mem   ███████████████████████████████████████ 4.68 MB/s
  minio        ████████████████████████ 2.88 MB/s
  seaweedfs    ███████████████████ 2.30 MB/s
  rustfs       █████████████ 1.56 MB/s
  localstack   ██████████ 1.23 MB/s
```

### ParallelRead/1KB/C10

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 1.18 MB/s | 814.1us | 1.2ms | 763.5us | 1.2ms | 1.8ms | 0 |
| minio | 1.08 MB/s | 904.1us | 1.3ms | 851.2us | 1.3ms | 1.5ms | 0 |
| rustfs | 1.00 MB/s | 973.9us | 1.4ms | 953.6us | 1.4ms | 1.5ms | 0 |
| liteio_mem | 0.93 MB/s | 1.0ms | 2.2ms | 804.8us | 2.2ms | 3.4ms | 0 |
| seaweedfs | 0.91 MB/s | 1.1ms | 1.7ms | 1.0ms | 1.7ms | 1.9ms | 0 |
| localstack | 0.19 MB/s | 5.2ms | 7.9ms | 5.0ms | 7.9ms | 8.3ms | 0 |

```
  liteio       ████████████████████████████████████████ 1.18 MB/s
  minio        ████████████████████████████████████ 1.08 MB/s
  rustfs       █████████████████████████████████ 1.00 MB/s
  liteio_mem   ███████████████████████████████ 0.93 MB/s
  seaweedfs    ██████████████████████████████ 0.91 MB/s
  localstack   ██████ 0.19 MB/s
```

### ParallelRead/1KB/C100

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 0.28 MB/s | 3.5ms | 7.0ms | 3.1ms | 7.0ms | 7.6ms | 0 |
| liteio_mem | 0.20 MB/s | 4.9ms | 8.2ms | 4.8ms | 8.2ms | 9.5ms | 0 |
| seaweedfs | 0.19 MB/s | 5.1ms | 7.0ms | 4.9ms | 7.0ms | 13.0ms | 0 |
| minio | 0.12 MB/s | 8.0ms | 8.9ms | 8.2ms | 8.9ms | 9.0ms | 0 |
| localstack | 0.02 MB/s | 58.4ms | 61.7ms | 59.3ms | 61.7ms | 62.1ms | 0 |

```
  liteio       ████████████████████████████████████████ 0.28 MB/s
  liteio_mem   ████████████████████████████ 0.20 MB/s
  seaweedfs    ███████████████████████████ 0.19 MB/s
  minio        █████████████████ 0.12 MB/s
  localstack   ██ 0.02 MB/s
```

### ParallelRead/1KB/C200

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 0.38 MB/s | 2.6ms | 5.4ms | 2.1ms | 5.4ms | 6.1ms | 0 |
| seaweedfs | 0.18 MB/s | 5.3ms | 6.5ms | 5.4ms | 6.5ms | 6.7ms | 0 |
| minio | 0.18 MB/s | 5.5ms | 6.7ms | 5.7ms | 6.7ms | 6.9ms | 0 |
| liteio_mem | 0.18 MB/s | 5.6ms | 8.3ms | 5.4ms | 8.3ms | 8.5ms | 0 |
| localstack | 0.01 MB/s | 82.0ms | 103.7ms | 100.6ms | 103.7ms | 104.5ms | 0 |

```
  liteio       ████████████████████████████████████████ 0.38 MB/s
  seaweedfs    ███████████████████ 0.18 MB/s
  minio        ██████████████████ 0.18 MB/s
  liteio_mem   ██████████████████ 0.18 MB/s
  localstack   █ 0.01 MB/s
```

### ParallelRead/1KB/C25

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 0.59 MB/s | 1.7ms | 2.2ms | 1.7ms | 2.2ms | 2.4ms | 0 |
| seaweedfs | 0.50 MB/s | 2.0ms | 3.1ms | 1.8ms | 3.1ms | 3.2ms | 0 |
| liteio | 0.43 MB/s | 2.3ms | 4.3ms | 2.2ms | 4.3ms | 4.7ms | 0 |
| liteio_mem | 0.40 MB/s | 2.4ms | 4.9ms | 2.5ms | 4.9ms | 5.0ms | 0 |
| localstack | 0.07 MB/s | 13.1ms | 17.0ms | 12.8ms | 17.0ms | 21.4ms | 0 |

```
  minio        ████████████████████████████████████████ 0.59 MB/s
  seaweedfs    ██████████████████████████████████ 0.50 MB/s
  liteio       █████████████████████████████ 0.43 MB/s
  liteio_mem   ███████████████████████████ 0.40 MB/s
  localstack   █████ 0.07 MB/s
```

### ParallelRead/1KB/C50

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 0.38 MB/s | 2.6ms | 3.7ms | 2.6ms | 3.7ms | 4.0ms | 0 |
| liteio | 0.34 MB/s | 2.9ms | 4.6ms | 2.6ms | 4.6ms | 5.8ms | 0 |
| seaweedfs | 0.32 MB/s | 3.1ms | 4.3ms | 3.0ms | 4.3ms | 4.4ms | 0 |
| liteio_mem | 0.29 MB/s | 3.4ms | 5.2ms | 3.8ms | 5.2ms | 5.7ms | 0 |
| localstack | 0.04 MB/s | 23.0ms | 29.2ms | 24.3ms | 29.2ms | 29.5ms | 0 |

```
  minio        ████████████████████████████████████████ 0.38 MB/s
  liteio       ███████████████████████████████████ 0.34 MB/s
  seaweedfs    █████████████████████████████████ 0.32 MB/s
  liteio_mem   ██████████████████████████████ 0.29 MB/s
  localstack   ████ 0.04 MB/s
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 1.40 MB/s | 681.6us | 797.3us | 994.4us | 0 |
| liteio | 1.39 MB/s | 652.8us | 938.7us | 1.5ms | 0 |
| liteio_mem | 1.27 MB/s | 673.3us | 1.3ms | 1.6ms | 0 |
| rustfs | 1.20 MB/s | 792.5us | 943.6us | 1.1ms | 0 |
| localstack | 1.18 MB/s | 818.2us | 938.6us | 1.1ms | 0 |
| minio | 1.08 MB/s | 806.2us | 1.5ms | 1.8ms | 0 |

```
  seaweedfs    ████████████████████████████████████████ 1.40 MB/s
  liteio       ███████████████████████████████████████ 1.39 MB/s
  liteio_mem   ████████████████████████████████████ 1.27 MB/s
  rustfs       ██████████████████████████████████ 1.20 MB/s
  localstack   █████████████████████████████████ 1.18 MB/s
  minio        ██████████████████████████████ 1.08 MB/s
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 0.48 MB/s | 1.9ms | 3.2ms | 4.3ms | 0 |
| rustfs | 0.39 MB/s | 2.3ms | 4.7ms | 5.1ms | 0 |
| minio | 0.36 MB/s | 2.7ms | 4.0ms | 4.9ms | 0 |
| liteio_mem | 0.32 MB/s | 2.8ms | 5.5ms | 6.0ms | 0 |
| liteio | 0.30 MB/s | 2.9ms | 6.0ms | 7.6ms | 0 |
| localstack | 0.19 MB/s | 4.9ms | 8.4ms | 8.6ms | 0 |

```
  seaweedfs    ████████████████████████████████████████ 0.48 MB/s
  rustfs       ████████████████████████████████ 0.39 MB/s
  minio        ██████████████████████████████ 0.36 MB/s
  liteio_mem   ██████████████████████████ 0.32 MB/s
  liteio       ████████████████████████ 0.30 MB/s
  localstack   ███████████████ 0.19 MB/s
```

### ParallelWrite/1KB/C100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 0.11 MB/s | 7.3ms | 12.5ms | 13.0ms | 0 |
| minio | 0.06 MB/s | 15.5ms | 21.3ms | 22.4ms | 0 |
| liteio_mem | 0.04 MB/s | 21.5ms | 34.0ms | 34.9ms | 0 |
| liteio | 0.04 MB/s | 22.2ms | 32.7ms | 33.1ms | 0 |
| localstack | 0.03 MB/s | 31.3ms | 48.2ms | 49.7ms | 0 |

```
  seaweedfs    ████████████████████████████████████████ 0.11 MB/s
  minio        ███████████████████████ 0.06 MB/s
  liteio_mem   ████████████████ 0.04 MB/s
  liteio       ████████████████ 0.04 MB/s
  localstack   ███████████ 0.03 MB/s
```

### ParallelWrite/1KB/C200

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 0.07 MB/s | 15.0ms | 20.6ms | 21.4ms | 0 |
| liteio | 0.06 MB/s | 15.0ms | 32.5ms | 32.9ms | 0 |
| liteio_mem | 0.04 MB/s | 21.3ms | 39.0ms | 39.9ms | 0 |
| seaweedfs | 0.04 MB/s | 30.9ms | 33.0ms | 33.3ms | 0 |
| localstack | 0.02 MB/s | 51.3ms | 53.2ms | 54.8ms | 0 |

```
  minio        ████████████████████████████████████████ 0.07 MB/s
  liteio       ███████████████████████████████████ 0.06 MB/s
  liteio_mem   ██████████████████████████ 0.04 MB/s
  seaweedfs    █████████████████████ 0.04 MB/s
  localstack   ███████████ 0.02 MB/s
```

### ParallelWrite/1KB/C25

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 0.26 MB/s | 3.5ms | 6.3ms | 7.2ms | 0 |
| minio | 0.19 MB/s | 4.8ms | 9.1ms | 10.7ms | 0 |
| liteio | 0.16 MB/s | 5.1ms | 11.0ms | 15.8ms | 0 |
| liteio_mem | 0.16 MB/s | 5.8ms | 12.0ms | 14.2ms | 0 |
| localstack | 0.08 MB/s | 11.2ms | 22.5ms | 28.1ms | 0 |

```
  seaweedfs    ████████████████████████████████████████ 0.26 MB/s
  minio        ████████████████████████████ 0.19 MB/s
  liteio       ████████████████████████ 0.16 MB/s
  liteio_mem   ███████████████████████ 0.16 MB/s
  localstack   ███████████ 0.08 MB/s
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 0.19 MB/s | 4.8ms | 7.2ms | 7.8ms | 0 |
| minio | 0.10 MB/s | 7.6ms | 22.2ms | 22.8ms | 0 |
| liteio | 0.09 MB/s | 9.5ms | 23.7ms | 27.5ms | 0 |
| liteio_mem | 0.08 MB/s | 9.9ms | 25.9ms | 26.5ms | 0 |
| localstack | 0.04 MB/s | 20.3ms | 54.5ms | 54.8ms | 0 |

```
  seaweedfs    ████████████████████████████████████████ 0.19 MB/s
  minio        ████████████████████ 0.10 MB/s
  liteio       █████████████████ 0.09 MB/s
  liteio_mem   ████████████████ 0.08 MB/s
  localstack   ███████ 0.04 MB/s
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 209.22 MB/s | 1.1ms | 1.6ms | 1.8ms | 0 |
| liteio_mem | 201.97 MB/s | 1.1ms | 1.7ms | 2.2ms | 0 |
| seaweedfs | 191.82 MB/s | 1.3ms | 1.5ms | 1.5ms | 0 |
| minio | 169.60 MB/s | 1.4ms | 1.7ms | 2.0ms | 0 |
| localstack | 158.18 MB/s | 1.5ms | 2.0ms | 3.4ms | 0 |
| rustfs | 136.03 MB/s | 1.8ms | 2.1ms | 2.5ms | 0 |

```
  liteio       ████████████████████████████████████████ 209.22 MB/s
  liteio_mem   ██████████████████████████████████████ 201.97 MB/s
  seaweedfs    ████████████████████████████████████ 191.82 MB/s
  minio        ████████████████████████████████ 169.60 MB/s
  localstack   ██████████████████████████████ 158.18 MB/s
  rustfs       ██████████████████████████ 136.03 MB/s
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 212.03 MB/s | 1.1ms | 1.5ms | 2.0ms | 0 |
| seaweedfs | 193.65 MB/s | 1.3ms | 1.4ms | 1.7ms | 0 |
| liteio | 189.87 MB/s | 1.2ms | 2.0ms | 2.8ms | 0 |
| minio | 181.69 MB/s | 1.3ms | 1.6ms | 1.7ms | 0 |
| localstack | 151.56 MB/s | 1.5ms | 2.3ms | 3.9ms | 0 |
| rustfs | 133.17 MB/s | 1.8ms | 2.1ms | 2.4ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 212.03 MB/s
  seaweedfs    ████████████████████████████████████ 193.65 MB/s
  liteio       ███████████████████████████████████ 189.87 MB/s
  minio        ██████████████████████████████████ 181.69 MB/s
  localstack   ████████████████████████████ 151.56 MB/s
  rustfs       █████████████████████████ 133.17 MB/s
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 208.83 MB/s | 1.1ms | 1.6ms | 2.3ms | 0 |
| liteio_mem | 201.06 MB/s | 1.1ms | 1.6ms | 3.3ms | 0 |
| seaweedfs | 193.02 MB/s | 1.3ms | 1.4ms | 1.5ms | 0 |
| minio | 170.81 MB/s | 1.4ms | 1.7ms | 2.9ms | 0 |
| localstack | 163.06 MB/s | 1.5ms | 1.7ms | 2.0ms | 0 |
| rustfs | 133.29 MB/s | 1.8ms | 2.1ms | 2.5ms | 0 |

```
  liteio       ████████████████████████████████████████ 208.83 MB/s
  liteio_mem   ██████████████████████████████████████ 201.06 MB/s
  seaweedfs    ████████████████████████████████████ 193.02 MB/s
  minio        ████████████████████████████████ 170.81 MB/s
  localstack   ███████████████████████████████ 163.06 MB/s
  rustfs       █████████████████████████ 133.29 MB/s
```

### Read/100MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 331.23 MB/s | 930.9us | 936.2us | 301.9ms | 303.2ms | 303.2ms | 0 |
| rustfs | 319.90 MB/s | 2.2ms | 2.5ms | 309.6ms | 322.5ms | 322.5ms | 0 |
| seaweedfs | 278.79 MB/s | 2.3ms | 2.6ms | 348.6ms | 353.5ms | 353.5ms | 0 |
| localstack | 272.64 MB/s | 1.6ms | 1.6ms | 364.8ms | 369.8ms | 369.8ms | 0 |
| liteio | 271.25 MB/s | 503.2us | 526.1us | 369.4ms | 377.2ms | 377.2ms | 0 |
| liteio_mem | 247.23 MB/s | 1.1ms | 958.4us | 399.6ms | 405.9ms | 405.9ms | 0 |

```
  minio        ████████████████████████████████████████ 331.23 MB/s
  rustfs       ██████████████████████████████████████ 319.90 MB/s
  seaweedfs    █████████████████████████████████ 278.79 MB/s
  localstack   ████████████████████████████████ 272.64 MB/s
  liteio       ████████████████████████████████ 271.25 MB/s
  liteio_mem   █████████████████████████████ 247.23 MB/s
```

### Read/10MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| seaweedfs | 301.97 MB/s | 2.1ms | 2.2ms | 32.9ms | 33.7ms | 33.7ms | 0 |
| minio | 300.72 MB/s | 1.1ms | 1.2ms | 31.2ms | 41.8ms | 41.8ms | 0 |
| localstack | 288.52 MB/s | 1.5ms | 1.5ms | 33.9ms | 39.6ms | 39.6ms | 0 |
| liteio | 279.73 MB/s | 725.7us | 1.2ms | 36.0ms | 37.6ms | 37.6ms | 0 |
| rustfs | 278.22 MB/s | 5.4ms | 5.8ms | 35.6ms | 38.9ms | 38.9ms | 0 |
| liteio_mem | 248.52 MB/s | 623.8us | 1.0ms | 38.0ms | 44.8ms | 44.8ms | 0 |

```
  seaweedfs    ████████████████████████████████████████ 301.97 MB/s
  minio        ███████████████████████████████████████ 300.72 MB/s
  localstack   ██████████████████████████████████████ 288.52 MB/s
  liteio       █████████████████████████████████████ 279.73 MB/s
  rustfs       ████████████████████████████████████ 278.22 MB/s
  liteio_mem   ████████████████████████████████ 248.52 MB/s
```

### Read/1KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio_mem | 5.16 MB/s | 187.9us | 225.5us | 184.9us | 229.8us | 247.1us | 0 |
| liteio | 4.82 MB/s | 202.5us | 236.2us | 203.2us | 236.2us | 249.0us | 0 |
| minio | 2.68 MB/s | 364.5us | 406.8us | 361.2us | 407.0us | 431.5us | 0 |
| seaweedfs | 2.40 MB/s | 407.1us | 469.7us | 408.2us | 470.0us | 507.7us | 0 |
| rustfs | 2.15 MB/s | 453.8us | 504.3us | 447.0us | 504.6us | 526.5us | 0 |
| localstack | 1.34 MB/s | 730.0us | 830.8us | 718.2us | 831.6us | 1.1ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 5.16 MB/s
  liteio       █████████████████████████████████████ 4.82 MB/s
  minio        ████████████████████ 2.68 MB/s
  seaweedfs    ██████████████████ 2.40 MB/s
  rustfs       ████████████████ 2.15 MB/s
  localstack   ██████████ 1.34 MB/s
```

### Read/1MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 293.34 MB/s | 358.7us | 643.9us | 3.4ms | 3.8ms | 3.8ms | 0 |
| minio | 266.46 MB/s | 791.6us | 922.2us | 3.7ms | 4.0ms | 4.0ms | 0 |
| seaweedfs | 262.84 MB/s | 859.8us | 1.0ms | 3.7ms | 4.0ms | 4.0ms | 0 |
| localstack | 259.97 MB/s | 991.0us | 1.2ms | 3.8ms | 4.3ms | 4.3ms | 0 |
| liteio_mem | 246.55 MB/s | 479.2us | 899.7us | 3.9ms | 4.7ms | 4.7ms | 0 |
| rustfs | 195.65 MB/s | 2.1ms | 3.5ms | 4.7ms | 6.8ms | 6.8ms | 0 |

```
  liteio       ████████████████████████████████████████ 293.34 MB/s
  minio        ████████████████████████████████████ 266.46 MB/s
  seaweedfs    ███████████████████████████████████ 262.84 MB/s
  localstack   ███████████████████████████████████ 259.97 MB/s
  liteio_mem   █████████████████████████████████ 246.55 MB/s
  rustfs       ██████████████████████████ 195.65 MB/s
```

### Read/64KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 148.62 MB/s | 228.5us | 299.7us | 403.1us | 469.6us | 601.3us | 0 |
| liteio_mem | 138.69 MB/s | 270.8us | 367.2us | 404.9us | 586.3us | 850.1us | 0 |
| minio | 102.57 MB/s | 400.8us | 470.1us | 601.0us | 701.5us | 758.2us | 0 |
| seaweedfs | 97.24 MB/s | 474.5us | 534.5us | 639.0us | 688.4us | 718.5us | 0 |
| rustfs | 89.77 MB/s | 596.9us | 648.0us | 693.4us | 722.2us | 751.6us | 0 |
| localstack | 71.21 MB/s | 783.8us | 833.5us | 868.9us | 907.7us | 935.2us | 0 |

```
  liteio       ████████████████████████████████████████ 148.62 MB/s
  liteio_mem   █████████████████████████████████████ 138.69 MB/s
  minio        ███████████████████████████ 102.57 MB/s
  seaweedfs    ██████████████████████████ 97.24 MB/s
  rustfs       ████████████████████████ 89.77 MB/s
  localstack   ███████████████████ 71.21 MB/s
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 5065 ops/s | 189.3us | 268.2us | 371.4us | 0 |
| liteio_mem | 4346 ops/s | 223.6us | 327.0us | 384.2us | 0 |
| minio | 4257 ops/s | 233.8us | 267.1us | 285.9us | 0 |
| seaweedfs | 3691 ops/s | 269.5us | 310.5us | 347.5us | 0 |
| rustfs | 3296 ops/s | 298.3us | 350.6us | 366.8us | 0 |
| localstack | 1630 ops/s | 595.5us | 697.7us | 995.4us | 0 |

```
  liteio       ████████████████████████████████████████ 5065 ops/s
  liteio_mem   ██████████████████████████████████ 4346 ops/s
  minio        █████████████████████████████████ 4257 ops/s
  seaweedfs    █████████████████████████████ 3691 ops/s
  rustfs       ██████████████████████████ 3296 ops/s
  localstack   ████████████ 1630 ops/s
```

### Write/100MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 197.27 MB/s | 503.9ms | 524.2ms | 524.2ms | 0 |
| liteio | 194.39 MB/s | 512.5ms | 517.0ms | 517.0ms | 0 |
| rustfs | 184.04 MB/s | 518.3ms | 606.2ms | 606.2ms | 0 |
| liteio_mem | 170.87 MB/s | 556.6ms | 587.6ms | 587.6ms | 0 |
| minio | 163.40 MB/s | 637.0ms | 639.1ms | 639.1ms | 0 |
| localstack | 154.57 MB/s | 652.5ms | 658.8ms | 658.8ms | 0 |

```
  seaweedfs    ████████████████████████████████████████ 197.27 MB/s
  liteio       ███████████████████████████████████████ 194.39 MB/s
  rustfs       █████████████████████████████████████ 184.04 MB/s
  liteio_mem   ██████████████████████████████████ 170.87 MB/s
  minio        █████████████████████████████████ 163.40 MB/s
  localstack   ███████████████████████████████ 154.57 MB/s
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 192.91 MB/s | 51.2ms | 54.5ms | 54.5ms | 0 |
| rustfs | 175.62 MB/s | 58.5ms | 63.4ms | 63.4ms | 0 |
| liteio_mem | 166.60 MB/s | 56.7ms | 72.8ms | 72.8ms | 0 |
| minio | 158.18 MB/s | 65.0ms | 70.0ms | 70.0ms | 0 |
| seaweedfs | 149.01 MB/s | 64.3ms | 69.6ms | 69.6ms | 0 |
| localstack | 145.12 MB/s | 68.2ms | 71.7ms | 71.7ms | 0 |

```
  liteio       ████████████████████████████████████████ 192.91 MB/s
  rustfs       ████████████████████████████████████ 175.62 MB/s
  liteio_mem   ██████████████████████████████████ 166.60 MB/s
  minio        ████████████████████████████████ 158.18 MB/s
  seaweedfs    ██████████████████████████████ 149.01 MB/s
  localstack   ██████████████████████████████ 145.12 MB/s
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 1.45 MB/s | 661.9us | 775.8us | 988.6us | 0 |
| rustfs | 1.34 MB/s | 712.2us | 805.8us | 1.4ms | 0 |
| localstack | 1.34 MB/s | 712.6us | 833.8us | 1.1ms | 0 |
| liteio | 1.34 MB/s | 695.2us | 1.0ms | 1.3ms | 0 |
| liteio_mem | 1.09 MB/s | 715.2us | 2.0ms | 2.4ms | 0 |
| minio | 1.03 MB/s | 863.8us | 1.5ms | 1.6ms | 0 |

```
  seaweedfs    ████████████████████████████████████████ 1.45 MB/s
  rustfs       ████████████████████████████████████ 1.34 MB/s
  localstack   ████████████████████████████████████ 1.34 MB/s
  liteio       ████████████████████████████████████ 1.34 MB/s
  liteio_mem   █████████████████████████████ 1.09 MB/s
  minio        ████████████████████████████ 1.03 MB/s
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 171.14 MB/s | 5.7ms | 6.4ms | 6.4ms | 0 |
| liteio_mem | 148.88 MB/s | 6.6ms | 7.5ms | 7.5ms | 0 |
| minio | 143.69 MB/s | 6.8ms | 7.4ms | 7.4ms | 0 |
| rustfs | 137.46 MB/s | 6.9ms | 10.0ms | 10.0ms | 0 |
| localstack | 131.95 MB/s | 7.5ms | 8.1ms | 8.1ms | 0 |
| seaweedfs | 121.73 MB/s | 8.0ms | 10.0ms | 10.0ms | 0 |

```
  liteio       ████████████████████████████████████████ 171.14 MB/s
  liteio_mem   ██████████████████████████████████ 148.88 MB/s
  minio        █████████████████████████████████ 143.69 MB/s
  rustfs       ████████████████████████████████ 137.46 MB/s
  localstack   ██████████████████████████████ 131.95 MB/s
  seaweedfs    ████████████████████████████ 121.73 MB/s
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 60.40 MB/s | 992.2us | 1.2ms | 1.4ms | 0 |
| localstack | 60.37 MB/s | 1.0ms | 1.1ms | 1.2ms | 0 |
| seaweedfs | 57.87 MB/s | 1.0ms | 1.2ms | 1.2ms | 0 |
| liteio | 55.10 MB/s | 988.3us | 1.8ms | 2.2ms | 0 |
| minio | 51.06 MB/s | 1.2ms | 1.4ms | 1.6ms | 0 |
| liteio_mem | 47.77 MB/s | 1.2ms | 2.9ms | 3.0ms | 0 |

```
  rustfs       ████████████████████████████████████████ 60.40 MB/s
  localstack   ███████████████████████████████████████ 60.37 MB/s
  seaweedfs    ██████████████████████████████████████ 57.87 MB/s
  liteio       ████████████████████████████████████ 55.10 MB/s
  minio        █████████████████████████████████ 51.06 MB/s
  liteio_mem   ███████████████████████████████ 47.77 MB/s
```

## Recommendations

- **Best for write-heavy workloads:** seaweedfs
- **Best for read-heavy workloads:** minio

---

*Report generated by storage benchmark CLI*
