# Storage Benchmark Report

**Generated:** 2026-01-15T09:48:14+07:00

**Go Version:** go1.25.5

**Platform:** darwin/arm64

## Executive Summary

### Summary

**Overall Winner:** liteio (won 21/51 benchmarks, 41%)

| Rank | Driver | Wins | Win Rate |
|------|--------|------|----------|
| 1 | liteio | 21 | 41% |
| 2 | liteio_mem | 13 | 25% |
| 3 | seaweedfs | 8 | 16% |
| 4 | rustfs | 5 | 10% |
| 5 | localstack | 2 | 4% |
| 6 | minio | 2 | 4% |

### Performance Leaders

| Operation | Leader | Performance | Margin |
|-----------|--------|-------------|--------|
| Small Read (1KB) | liteio_mem | 4.9 MB/s | close |
| Small Write (1KB) | rustfs | 1.4 MB/s | close |
| Large Read (10MB) | localstack | 291.6 MB/s | close |
| Large Write (10MB) | liteio_mem | 187.6 MB/s | close |
| Delete | liteio | 6.3K ops/s | close |
| Stat | liteio | 5.7K ops/s | close |
| List (100 objects) | liteio | 1.3K ops/s | close |
| Copy | liteio | 1.6 MB/s | +16% vs liteio_mem |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (100MB+) | **seaweedfs** | 199 MB/s | Best for media, backups |
| Large File Downloads (100MB) | **localstack** | 285 MB/s | Best for streaming, CDN |
| Small File Operations | **liteio_mem** | 3128 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **minio** | - | Best for multi-user apps |

### Large File Performance (100MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| liteio | 192.0 | 283.0 | 523.2ms | 352.9ms |
| liteio_mem | 178.2 | 265.9 | 548.2ms | 366.7ms |
| localstack | 154.9 | 284.8 | 637.7ms | 349.6ms |
| minio | 97.9 | 158.2 | 1.01s | 682.3ms |
| rustfs | 176.0 | 281.9 | 588.0ms | 349.7ms |
| seaweedfs | 199.4 | 254.2 | 507.5ms | 392.9ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| liteio | 1241 | 4633 | 751.1us | 212.5us |
| liteio_mem | 1248 | 5007 | 723.0us | 199.1us |
| localstack | 1386 | 1489 | 693.8us | 645.6us |
| minio | 561 | 1587 | 1.7ms | 497.3us |
| rustfs | 1462 | 1955 | 684.7us | 487.8us |
| seaweedfs | 1395 | 2395 | 698.5us | 406.6us |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| liteio | 5740 | 1319 | 6292 |
| liteio_mem | 5584 | 1299 | 5781 |
| localstack | 1645 | 413 | 1713 |
| minio | 752 | 226 | 948 |
| rustfs | 2962 | 159 | 1183 |
| seaweedfs | 2969 | 704 | 3244 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| liteio | 1.45 | 0.40 | 0.22 | 0.08 | 0.09 | 0.09 |
| liteio_mem | 1.47 | 0.41 | 0.23 | 0.10 | 0.09 | 0.08 |
| localstack | 1.21 | 0.19 | 0.07 | 0.03 | 0.02 | 0.02 |
| minio | 0.28 | 0.33 | 0.15 | 0.10 | 0.06 | 0.04 |
| rustfs | 1.07 | 0.35 | - | - | - | - |
| seaweedfs | 1.41 | 0.49 | 0.29 | 0.17 | 0.09 | 0.10 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| liteio | 4.61 | 1.17 | 0.81 | 0.51 | 0.30 | 0.28 |
| liteio_mem | 4.58 | 1.21 | 0.82 | 0.53 | 0.31 | 0.26 |
| localstack | 1.21 | 0.18 | 0.07 | 0.04 | 0.02 | 0.02 |
| minio | 1.84 | 1.36 | 0.77 | 0.32 | 0.13 | 0.32 |
| rustfs | 1.77 | 0.91 | - | - | - | - |
| seaweedfs | 2.28 | 0.94 | 0.52 | 0.07 | 0.20 | 0.19 |

*\* indicates errors occurred*

### File Count Performance

Performance with varying numbers of files (1KB each).

**Write N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|------|
| liteio | 1.5ms | 6.9ms | 61.2ms | 618.1ms | 6.49s |
| liteio_mem | 1.0ms | 6.4ms | 65.4ms | 654.5ms | 6.79s |
| localstack | 733.7us | 7.6ms | 71.5ms | 737.7ms | 7.61s |
| minio | 1.0ms | 21.0ms | 108.1ms | 865.4ms | 8.96s |
| rustfs | 832.6us | 9.5ms | 78.4ms | 712.2ms | 7.18s |
| seaweedfs | 650.8us | 6.9ms | 67.5ms | 665.4ms | 6.79s |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|------|
| liteio | 309.1us | 350.2us | 787.8us | 4.9ms | 183.9ms |
| liteio_mem | 290.0us | 406.2us | 1.3ms | 5.4ms | 201.7ms |
| localstack | 973.8us | 1.8ms | 4.4ms | 19.7ms | 208.4ms |
| minio | 765.8us | 735.3us | 2.2ms | 14.6ms | 164.3ms |
| rustfs | 1.1ms | 3.5ms | 7.2ms | 45.0ms | 735.4ms |
| seaweedfs | 699.0us | 791.2us | 1.9ms | 9.2ms | 91.4ms |

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
| liteio | 1.63 MB/s | 568.1us | 758.6us | 1.1ms | 0 |
| liteio_mem | 1.41 MB/s | 591.9us | 1.3ms | 2.0ms | 0 |
| localstack | 1.36 MB/s | 684.2us | 1.0ms | 1.2ms | 0 |
| seaweedfs | 0.99 MB/s | 921.4us | 1.4ms | 1.5ms | 0 |
| rustfs | 0.92 MB/s | 1.0ms | 1.2ms | 1.3ms | 0 |
| minio | 0.82 MB/s | 1.1ms | 1.7ms | 1.9ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.63 MB/s
liteio_mem   █████████████████████████ 1.41 MB/s
localstack   ████████████████████████ 1.36 MB/s
seaweedfs    ██████████████████ 0.99 MB/s
rustfs       ████████████████ 0.92 MB/s
minio        ██████████████ 0.82 MB/s
```

**Latency (P50)**
```
liteio       ███████████████ 568.1us
liteio_mem   ███████████████ 591.9us
localstack   ██████████████████ 684.2us
seaweedfs    ████████████████████████ 921.4us
rustfs       ████████████████████████████ 1.0ms
minio        ██████████████████████████████ 1.1ms
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 6292 ops/s | 147.4us | 190.0us | 208.9us | 0 |
| liteio_mem | 5781 ops/s | 151.5us | 210.8us | 235.6us | 0 |
| seaweedfs | 3244 ops/s | 305.2us | 348.0us | 371.7us | 0 |
| localstack | 1713 ops/s | 566.0us | 692.5us | 734.2us | 0 |
| rustfs | 1183 ops/s | 817.0us | 1.0ms | 1.1ms | 0 |
| minio | 948 ops/s | 1.0ms | 1.8ms | 2.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 6292 ops/s
liteio_mem   ███████████████████████████ 5781 ops/s
seaweedfs    ███████████████ 3244 ops/s
localstack   ████████ 1713 ops/s
rustfs       █████ 1183 ops/s
minio        ████ 948 ops/s
```

**Latency (P50)**
```
liteio       ████ 147.4us
liteio_mem   ████ 151.5us
seaweedfs    █████████ 305.2us
localstack   ████████████████ 566.0us
rustfs       ████████████████████████ 817.0us
minio        ██████████████████████████████ 1.0ms
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.16 MB/s | 555.4us | 765.5us | 850.3us | 0 |
| liteio_mem | 0.14 MB/s | 644.5us | 817.9us | 942.9us | 0 |
| seaweedfs | 0.14 MB/s | 630.5us | 779.5us | 981.8us | 0 |
| localstack | 0.14 MB/s | 680.7us | 779.9us | 808.0us | 0 |
| rustfs | 0.12 MB/s | 725.2us | 906.2us | 1.3ms | 0 |
| minio | 0.10 MB/s | 953.0us | 1.3ms | 1.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.16 MB/s
liteio_mem   ██████████████████████████ 0.14 MB/s
seaweedfs    ██████████████████████████ 0.14 MB/s
localstack   ████████████████████████ 0.14 MB/s
rustfs       ██████████████████████ 0.12 MB/s
minio        █████████████████ 0.10 MB/s
```

**Latency (P50)**
```
liteio       █████████████████ 555.4us
liteio_mem   ████████████████████ 644.5us
seaweedfs    ███████████████████ 630.5us
localstack   █████████████████████ 680.7us
rustfs       ██████████████████████ 725.2us
minio        ██████████████████████████████ 953.0us
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 2490 ops/s | 392.7us | 459.8us | 565.2us | 0 |
| liteio | 1637 ops/s | 568.9us | 803.7us | 971.1us | 0 |
| liteio_mem | 1404 ops/s | 647.5us | 890.0us | 1.5ms | 0 |
| rustfs | 1358 ops/s | 709.5us | 919.8us | 964.1us | 0 |
| minio | 866 ops/s | 1.0ms | 1.6ms | 2.5ms | 0 |
| localstack | 862 ops/s | 709.5us | 3.8ms | 4.8ms | 0 |

**Throughput**
```
seaweedfs    ██████████████████████████████ 2490 ops/s
liteio       ███████████████████ 1637 ops/s
liteio_mem   ████████████████ 1404 ops/s
rustfs       ████████████████ 1358 ops/s
minio        ██████████ 866 ops/s
localstack   ██████████ 862 ops/s
```

**Latency (P50)**
```
seaweedfs    ███████████ 392.7us
liteio       ████████████████ 568.9us
liteio_mem   ██████████████████ 647.5us
rustfs       ████████████████████ 709.5us
minio        ██████████████████████████████ 1.0ms
localstack   ████████████████████ 709.5us
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.16 MB/s | 587.4us | 729.2us | 762.2us | 0 |
| liteio_mem | 0.15 MB/s | 639.7us | 734.0us | 772.5us | 0 |
| localstack | 0.13 MB/s | 690.7us | 823.2us | 1.2ms | 0 |
| rustfs | 0.13 MB/s | 723.7us | 895.2us | 1.1ms | 0 |
| seaweedfs | 0.13 MB/s | 707.3us | 979.6us | 1.1ms | 0 |
| minio | 0.07 MB/s | 1.2ms | 2.6ms | 3.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 0.16 MB/s
liteio_mem   ████████████████████████████ 0.15 MB/s
localstack   █████████████████████████ 0.13 MB/s
rustfs       ████████████████████████ 0.13 MB/s
seaweedfs    ████████████████████████ 0.13 MB/s
minio        ████████████ 0.07 MB/s
```

**Latency (P50)**
```
liteio       ███████████████ 587.4us
liteio_mem   ████████████████ 639.7us
localstack   █████████████████ 690.7us
rustfs       ██████████████████ 723.7us
seaweedfs    ██████████████████ 707.3us
minio        ██████████████████████████████ 1.2ms
```

### FileCount/Delete/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 4784 ops/s | 209.0us | 209.0us | 209.0us | 0 |
| liteio_mem | 4086 ops/s | 244.8us | 244.8us | 244.8us | 0 |
| seaweedfs | 2970 ops/s | 336.7us | 336.7us | 336.7us | 0 |
| minio | 2283 ops/s | 438.1us | 438.1us | 438.1us | 0 |
| localstack | 1493 ops/s | 669.9us | 669.9us | 669.9us | 0 |
| rustfs | 908 ops/s | 1.1ms | 1.1ms | 1.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4784 ops/s
liteio_mem   █████████████████████████ 4086 ops/s
seaweedfs    ██████████████████ 2970 ops/s
minio        ██████████████ 2283 ops/s
localstack   █████████ 1493 ops/s
rustfs       █████ 908 ops/s
```

**Latency (P50)**
```
liteio       █████ 209.0us
liteio_mem   ██████ 244.8us
seaweedfs    █████████ 336.7us
minio        ███████████ 438.1us
localstack   ██████████████████ 669.9us
rustfs       ██████████████████████████████ 1.1ms
```

### FileCount/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 484 ops/s | 2.1ms | 2.1ms | 2.1ms | 0 |
| liteio_mem | 413 ops/s | 2.4ms | 2.4ms | 2.4ms | 0 |
| seaweedfs | 281 ops/s | 3.6ms | 3.6ms | 3.6ms | 0 |
| minio | 233 ops/s | 4.3ms | 4.3ms | 4.3ms | 0 |
| localstack | 166 ops/s | 6.0ms | 6.0ms | 6.0ms | 0 |
| rustfs | 106 ops/s | 9.4ms | 9.4ms | 9.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 484 ops/s
liteio_mem   █████████████████████████ 413 ops/s
seaweedfs    █████████████████ 281 ops/s
minio        ██████████████ 233 ops/s
localstack   ██████████ 166 ops/s
rustfs       ██████ 106 ops/s
```

**Latency (P50)**
```
liteio       ██████ 2.1ms
liteio_mem   ███████ 2.4ms
seaweedfs    ███████████ 3.6ms
minio        █████████████ 4.3ms
localstack   ███████████████████ 6.0ms
rustfs       ██████████████████████████████ 9.4ms
```

### FileCount/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 55 ops/s | 18.2ms | 18.2ms | 18.2ms | 0 |
| liteio_mem | 46 ops/s | 21.9ms | 21.9ms | 21.9ms | 0 |
| seaweedfs | 31 ops/s | 31.8ms | 31.8ms | 31.8ms | 0 |
| minio | 25 ops/s | 40.7ms | 40.7ms | 40.7ms | 0 |
| localstack | 17 ops/s | 59.0ms | 59.0ms | 59.0ms | 0 |
| rustfs | 12 ops/s | 86.1ms | 86.1ms | 86.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 55 ops/s
liteio_mem   █████████████████████████ 46 ops/s
seaweedfs    █████████████████ 31 ops/s
minio        █████████████ 25 ops/s
localstack   █████████ 17 ops/s
rustfs       ██████ 12 ops/s
```

**Latency (P50)**
```
liteio       ██████ 18.2ms
liteio_mem   ███████ 21.9ms
seaweedfs    ███████████ 31.8ms
minio        ██████████████ 40.7ms
localstack   ████████████████████ 59.0ms
rustfs       ██████████████████████████████ 86.1ms
```

### FileCount/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 5 ops/s | 182.7ms | 182.7ms | 182.7ms | 0 |
| liteio_mem | 5 ops/s | 184.3ms | 184.3ms | 184.3ms | 0 |
| seaweedfs | 3 ops/s | 302.5ms | 302.5ms | 302.5ms | 0 |
| minio | 3 ops/s | 377.0ms | 377.0ms | 377.0ms | 0 |
| localstack | 2 ops/s | 576.2ms | 576.2ms | 576.2ms | 0 |
| rustfs | 1 ops/s | 837.0ms | 837.0ms | 837.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 5 ops/s
liteio_mem   █████████████████████████████ 5 ops/s
seaweedfs    ██████████████████ 3 ops/s
minio        ██████████████ 3 ops/s
localstack   █████████ 2 ops/s
rustfs       ██████ 1 ops/s
```

**Latency (P50)**
```
liteio       ██████ 182.7ms
liteio_mem   ██████ 184.3ms
seaweedfs    ██████████ 302.5ms
minio        █████████████ 377.0ms
localstack   ████████████████████ 576.2ms
rustfs       ██████████████████████████████ 837.0ms
```

### FileCount/Delete/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1 ops/s | 1.82s | 1.82s | 1.82s | 0 |
| liteio | 1 ops/s | 1.87s | 1.87s | 1.87s | 0 |
| seaweedfs | 0 ops/s | 3.14s | 3.14s | 3.14s | 0 |
| minio | 0 ops/s | 3.50s | 3.50s | 3.50s | 0 |
| localstack | 0 ops/s | 6.06s | 6.06s | 6.06s | 0 |
| rustfs | 0 ops/s | 8.31s | 8.31s | 8.31s | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 1 ops/s
liteio       █████████████████████████████ 1 ops/s
seaweedfs    █████████████████ 0 ops/s
minio        ███████████████ 0 ops/s
localstack   █████████ 0 ops/s
rustfs       ██████ 0 ops/s
```

**Latency (P50)**
```
liteio_mem   ██████ 1.82s
liteio       ██████ 1.87s
seaweedfs    ███████████ 3.14s
minio        ████████████ 3.50s
localstack   █████████████████████ 6.06s
rustfs       ██████████████████████████████ 8.31s
```

### FileCount/List/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 3448 ops/s | 290.0us | 290.0us | 290.0us | 0 |
| liteio | 3235 ops/s | 309.1us | 309.1us | 309.1us | 0 |
| seaweedfs | 1431 ops/s | 699.0us | 699.0us | 699.0us | 0 |
| minio | 1306 ops/s | 765.8us | 765.8us | 765.8us | 0 |
| localstack | 1027 ops/s | 973.8us | 973.8us | 973.8us | 0 |
| rustfs | 884 ops/s | 1.1ms | 1.1ms | 1.1ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 3448 ops/s
liteio       ████████████████████████████ 3235 ops/s
seaweedfs    ████████████ 1431 ops/s
minio        ███████████ 1306 ops/s
localstack   ████████ 1027 ops/s
rustfs       ███████ 884 ops/s
```

**Latency (P50)**
```
liteio_mem   ███████ 290.0us
liteio       ████████ 309.1us
seaweedfs    ██████████████████ 699.0us
minio        ████████████████████ 765.8us
localstack   █████████████████████████ 973.8us
rustfs       ██████████████████████████████ 1.1ms
```

### FileCount/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 2855 ops/s | 350.2us | 350.2us | 350.2us | 0 |
| liteio_mem | 2462 ops/s | 406.2us | 406.2us | 406.2us | 0 |
| minio | 1360 ops/s | 735.3us | 735.3us | 735.3us | 0 |
| seaweedfs | 1264 ops/s | 791.2us | 791.2us | 791.2us | 0 |
| localstack | 568 ops/s | 1.8ms | 1.8ms | 1.8ms | 0 |
| rustfs | 290 ops/s | 3.5ms | 3.5ms | 3.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 2855 ops/s
liteio_mem   █████████████████████████ 2462 ops/s
minio        ██████████████ 1360 ops/s
seaweedfs    █████████████ 1264 ops/s
localstack   █████ 568 ops/s
rustfs       ███ 290 ops/s
```

**Latency (P50)**
```
liteio       ███ 350.2us
liteio_mem   ███ 406.2us
minio        ██████ 735.3us
seaweedfs    ██████ 791.2us
localstack   ███████████████ 1.8ms
rustfs       ██████████████████████████████ 3.5ms
```

### FileCount/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1269 ops/s | 787.8us | 787.8us | 787.8us | 0 |
| liteio_mem | 747 ops/s | 1.3ms | 1.3ms | 1.3ms | 0 |
| seaweedfs | 535 ops/s | 1.9ms | 1.9ms | 1.9ms | 0 |
| minio | 455 ops/s | 2.2ms | 2.2ms | 2.2ms | 0 |
| localstack | 228 ops/s | 4.4ms | 4.4ms | 4.4ms | 0 |
| rustfs | 138 ops/s | 7.2ms | 7.2ms | 7.2ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1269 ops/s
liteio_mem   █████████████████ 747 ops/s
seaweedfs    ████████████ 535 ops/s
minio        ██████████ 455 ops/s
localstack   █████ 228 ops/s
rustfs       ███ 138 ops/s
```

**Latency (P50)**
```
liteio       ███ 787.8us
liteio_mem   █████ 1.3ms
seaweedfs    ███████ 1.9ms
minio        █████████ 2.2ms
localstack   ██████████████████ 4.4ms
rustfs       ██████████████████████████████ 7.2ms
```

### FileCount/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 203 ops/s | 4.9ms | 4.9ms | 4.9ms | 0 |
| liteio_mem | 187 ops/s | 5.4ms | 5.4ms | 5.4ms | 0 |
| seaweedfs | 109 ops/s | 9.2ms | 9.2ms | 9.2ms | 0 |
| minio | 68 ops/s | 14.6ms | 14.6ms | 14.6ms | 0 |
| localstack | 51 ops/s | 19.7ms | 19.7ms | 19.7ms | 0 |
| rustfs | 22 ops/s | 45.0ms | 45.0ms | 45.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 203 ops/s
liteio_mem   ███████████████████████████ 187 ops/s
seaweedfs    ████████████████ 109 ops/s
minio        ██████████ 68 ops/s
localstack   ███████ 51 ops/s
rustfs       ███ 22 ops/s
```

**Latency (P50)**
```
liteio       ███ 4.9ms
liteio_mem   ███ 5.4ms
seaweedfs    ██████ 9.2ms
minio        █████████ 14.6ms
localstack   █████████████ 19.7ms
rustfs       ██████████████████████████████ 45.0ms
```

### FileCount/List/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 11 ops/s | 91.4ms | 91.4ms | 91.4ms | 0 |
| minio | 6 ops/s | 164.3ms | 164.3ms | 164.3ms | 0 |
| liteio | 5 ops/s | 183.9ms | 183.9ms | 183.9ms | 0 |
| liteio_mem | 5 ops/s | 201.7ms | 201.7ms | 201.7ms | 0 |
| localstack | 5 ops/s | 208.4ms | 208.4ms | 208.4ms | 0 |
| rustfs | 1 ops/s | 735.4ms | 735.4ms | 735.4ms | 0 |

**Throughput**
```
seaweedfs    ██████████████████████████████ 11 ops/s
minio        ████████████████ 6 ops/s
liteio       ██████████████ 5 ops/s
liteio_mem   █████████████ 5 ops/s
localstack   █████████████ 5 ops/s
rustfs       ███ 1 ops/s
```

**Latency (P50)**
```
seaweedfs    ███ 91.4ms
minio        ██████ 164.3ms
liteio       ███████ 183.9ms
liteio_mem   ████████ 201.7ms
localstack   ████████ 208.4ms
rustfs       ██████████████████████████████ 735.4ms
```

### FileCount/Write/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 1.50 MB/s | 650.8us | 650.8us | 650.8us | 0 |
| localstack | 1.33 MB/s | 733.7us | 733.7us | 733.7us | 0 |
| rustfs | 1.17 MB/s | 832.6us | 832.6us | 832.6us | 0 |
| liteio_mem | 0.96 MB/s | 1.0ms | 1.0ms | 1.0ms | 0 |
| minio | 0.93 MB/s | 1.0ms | 1.0ms | 1.0ms | 0 |
| liteio | 0.63 MB/s | 1.5ms | 1.5ms | 1.5ms | 0 |

**Throughput**
```
seaweedfs    ██████████████████████████████ 1.50 MB/s
localstack   ██████████████████████████ 1.33 MB/s
rustfs       ███████████████████████ 1.17 MB/s
liteio_mem   ███████████████████ 0.96 MB/s
minio        ██████████████████ 0.93 MB/s
liteio       ████████████ 0.63 MB/s
```

**Latency (P50)**
```
seaweedfs    ████████████ 650.8us
localstack   ██████████████ 733.7us
rustfs       ████████████████ 832.6us
liteio_mem   ███████████████████ 1.0ms
minio        ████████████████████ 1.0ms
liteio       ██████████████████████████████ 1.5ms
```

### FileCount/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1.52 MB/s | 6.4ms | 6.4ms | 6.4ms | 0 |
| seaweedfs | 1.42 MB/s | 6.9ms | 6.9ms | 6.9ms | 0 |
| liteio | 1.41 MB/s | 6.9ms | 6.9ms | 6.9ms | 0 |
| localstack | 1.29 MB/s | 7.6ms | 7.6ms | 7.6ms | 0 |
| rustfs | 1.03 MB/s | 9.5ms | 9.5ms | 9.5ms | 0 |
| minio | 0.47 MB/s | 21.0ms | 21.0ms | 21.0ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 1.52 MB/s
seaweedfs    ███████████████████████████ 1.42 MB/s
liteio       ███████████████████████████ 1.41 MB/s
localstack   █████████████████████████ 1.29 MB/s
rustfs       ████████████████████ 1.03 MB/s
minio        █████████ 0.47 MB/s
```

**Latency (P50)**
```
liteio_mem   █████████ 6.4ms
seaweedfs    █████████ 6.9ms
liteio       █████████ 6.9ms
localstack   ██████████ 7.6ms
rustfs       █████████████ 9.5ms
minio        ██████████████████████████████ 21.0ms
```

### FileCount/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.60 MB/s | 61.2ms | 61.2ms | 61.2ms | 0 |
| liteio_mem | 1.49 MB/s | 65.4ms | 65.4ms | 65.4ms | 0 |
| seaweedfs | 1.45 MB/s | 67.5ms | 67.5ms | 67.5ms | 0 |
| localstack | 1.37 MB/s | 71.5ms | 71.5ms | 71.5ms | 0 |
| rustfs | 1.25 MB/s | 78.4ms | 78.4ms | 78.4ms | 0 |
| minio | 0.90 MB/s | 108.1ms | 108.1ms | 108.1ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.60 MB/s
liteio_mem   ████████████████████████████ 1.49 MB/s
seaweedfs    ███████████████████████████ 1.45 MB/s
localstack   █████████████████████████ 1.37 MB/s
rustfs       ███████████████████████ 1.25 MB/s
minio        ████████████████ 0.90 MB/s
```

**Latency (P50)**
```
liteio       ████████████████ 61.2ms
liteio_mem   ██████████████████ 65.4ms
seaweedfs    ██████████████████ 67.5ms
localstack   ███████████████████ 71.5ms
rustfs       █████████████████████ 78.4ms
minio        ██████████████████████████████ 108.1ms
```

### FileCount/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.58 MB/s | 618.1ms | 618.1ms | 618.1ms | 0 |
| liteio_mem | 1.49 MB/s | 654.5ms | 654.5ms | 654.5ms | 0 |
| seaweedfs | 1.47 MB/s | 665.4ms | 665.4ms | 665.4ms | 0 |
| rustfs | 1.37 MB/s | 712.2ms | 712.2ms | 712.2ms | 0 |
| localstack | 1.32 MB/s | 737.7ms | 737.7ms | 737.7ms | 0 |
| minio | 1.13 MB/s | 865.4ms | 865.4ms | 865.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.58 MB/s
liteio_mem   ████████████████████████████ 1.49 MB/s
seaweedfs    ███████████████████████████ 1.47 MB/s
rustfs       ██████████████████████████ 1.37 MB/s
localstack   █████████████████████████ 1.32 MB/s
minio        █████████████████████ 1.13 MB/s
```

**Latency (P50)**
```
liteio       █████████████████████ 618.1ms
liteio_mem   ██████████████████████ 654.5ms
seaweedfs    ███████████████████████ 665.4ms
rustfs       ████████████████████████ 712.2ms
localstack   █████████████████████████ 737.7ms
minio        ██████████████████████████████ 865.4ms
```

### FileCount/Write/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.50 MB/s | 6.49s | 6.49s | 6.49s | 0 |
| liteio_mem | 1.44 MB/s | 6.79s | 6.79s | 6.79s | 0 |
| seaweedfs | 1.44 MB/s | 6.79s | 6.79s | 6.79s | 0 |
| rustfs | 1.36 MB/s | 7.18s | 7.18s | 7.18s | 0 |
| localstack | 1.28 MB/s | 7.61s | 7.61s | 7.61s | 0 |
| minio | 1.09 MB/s | 8.96s | 8.96s | 8.96s | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1.50 MB/s
liteio_mem   ████████████████████████████ 1.44 MB/s
seaweedfs    ████████████████████████████ 1.44 MB/s
rustfs       ███████████████████████████ 1.36 MB/s
localstack   █████████████████████████ 1.28 MB/s
minio        █████████████████████ 1.09 MB/s
```

**Latency (P50)**
```
liteio       █████████████████████ 6.49s
liteio_mem   ██████████████████████ 6.79s
seaweedfs    ██████████████████████ 6.79s
rustfs       ████████████████████████ 7.18s
localstack   █████████████████████████ 7.61s
minio        ██████████████████████████████ 8.96s
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1319 ops/s | 724.5us | 823.2us | 1.7ms | 0 |
| liteio_mem | 1299 ops/s | 733.5us | 870.8us | 1.4ms | 0 |
| seaweedfs | 704 ops/s | 1.4ms | 1.6ms | 1.9ms | 0 |
| localstack | 413 ops/s | 2.4ms | 2.6ms | 2.7ms | 0 |
| minio | 226 ops/s | 4.4ms | 5.6ms | 5.7ms | 0 |
| rustfs | 159 ops/s | 6.4ms | 7.0ms | 7.3ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 1319 ops/s
liteio_mem   █████████████████████████████ 1299 ops/s
seaweedfs    ████████████████ 704 ops/s
localstack   █████████ 413 ops/s
minio        █████ 226 ops/s
rustfs       ███ 159 ops/s
```

**Latency (P50)**
```
liteio       ███ 724.5us
liteio_mem   ███ 733.5us
seaweedfs    ██████ 1.4ms
localstack   ███████████ 2.4ms
minio        ████████████████████ 4.4ms
rustfs       ██████████████████████████████ 6.4ms
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 9.04 MB/s | 1.4ms | 4.4ms | 5.1ms | 0 |
| seaweedfs | 1.33 MB/s | 12.1ms | 15.4ms | 15.5ms | 0 |
| liteio | 1.26 MB/s | 7.8ms | 31.6ms | 31.7ms | 0 |
| minio | 1.02 MB/s | 12.5ms | 22.4ms | 22.9ms | 0 |
| liteio_mem | 0.97 MB/s | 15.8ms | 28.5ms | 28.6ms | 0 |
| localstack | 0.29 MB/s | 58.7ms | 61.3ms | 62.2ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 9.04 MB/s
seaweedfs    ████ 1.33 MB/s
liteio       ████ 1.26 MB/s
minio        ███ 1.02 MB/s
liteio_mem   ███ 0.97 MB/s
localstack   █ 0.29 MB/s
```

**Latency (P50)**
```
rustfs       █ 1.4ms
seaweedfs    ██████ 12.1ms
liteio       ████ 7.8ms
minio        ██████ 12.5ms
liteio_mem   ████████ 15.8ms
localstack   ██████████████████████████████ 58.7ms
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 13.41 MB/s | 1.1ms | 1.9ms | 2.1ms | 0 |
| liteio_mem | 1.89 MB/s | 8.6ms | 10.0ms | 10.1ms | 0 |
| liteio | 1.60 MB/s | 9.9ms | 10.8ms | 11.0ms | 0 |
| minio | 1.37 MB/s | 12.6ms | 14.8ms | 14.9ms | 0 |
| seaweedfs | 1.25 MB/s | 13.5ms | 14.3ms | 14.5ms | 0 |
| localstack | 0.30 MB/s | 57.6ms | 65.6ms | 66.0ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 13.41 MB/s
liteio_mem   ████ 1.89 MB/s
liteio       ███ 1.60 MB/s
minio        ███ 1.37 MB/s
seaweedfs    ██ 1.25 MB/s
localstack   █ 0.30 MB/s
```

**Latency (P50)**
```
rustfs       █ 1.1ms
liteio_mem   ████ 8.6ms
liteio       █████ 9.9ms
minio        ██████ 12.6ms
seaweedfs    ███████ 13.5ms
localstack   ██████████████████████████████ 57.6ms
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 5.66 MB/s | 2.7ms | 3.8ms | 4.0ms | 0 |
| seaweedfs | 1.23 MB/s | 12.7ms | 17.1ms | 17.6ms | 0 |
| liteio | 1.14 MB/s | 13.4ms | 20.1ms | 20.3ms | 0 |
| liteio_mem | 1.13 MB/s | 14.1ms | 19.1ms | 19.7ms | 0 |
| minio | 0.76 MB/s | 19.4ms | 29.2ms | 30.0ms | 0 |
| localstack | 0.28 MB/s | 62.0ms | 63.3ms | 63.4ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 5.66 MB/s
seaweedfs    ██████ 1.23 MB/s
liteio       ██████ 1.14 MB/s
liteio_mem   █████ 1.13 MB/s
minio        ████ 0.76 MB/s
localstack   █ 0.28 MB/s
```

**Latency (P50)**
```
rustfs       █ 2.7ms
seaweedfs    ██████ 12.7ms
liteio       ██████ 13.4ms
liteio_mem   ██████ 14.1ms
minio        █████████ 19.4ms
localstack   ██████████████████████████████ 62.0ms
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 167.03 MB/s | 84.4ms | 108.7ms | 108.7ms | 0 |
| liteio | 164.09 MB/s | 90.9ms | 96.2ms | 96.2ms | 0 |
| liteio_mem | 154.29 MB/s | 94.0ms | 107.8ms | 107.8ms | 0 |
| localstack | 134.79 MB/s | 110.9ms | 115.8ms | 115.8ms | 0 |
| seaweedfs | 134.70 MB/s | 110.7ms | 114.4ms | 114.4ms | 0 |
| minio | 105.84 MB/s | 125.4ms | 213.5ms | 213.5ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 167.03 MB/s
liteio       █████████████████████████████ 164.09 MB/s
liteio_mem   ███████████████████████████ 154.29 MB/s
localstack   ████████████████████████ 134.79 MB/s
seaweedfs    ████████████████████████ 134.70 MB/s
minio        ███████████████████ 105.84 MB/s
```

**Latency (P50)**
```
rustfs       ████████████████████ 84.4ms
liteio       █████████████████████ 90.9ms
liteio_mem   ██████████████████████ 94.0ms
localstack   ██████████████████████████ 110.9ms
seaweedfs    ██████████████████████████ 110.7ms
minio        ██████████████████████████████ 125.4ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 4.61 MB/s | 209.8us | 274.1us | 203.8us | 274.1us | 358.2us | 0 |
| liteio_mem | 4.58 MB/s | 211.4us | 235.1us | 195.8us | 238.9us | 356.5us | 0 |
| seaweedfs | 2.28 MB/s | 427.3us | 476.5us | 422.9us | 476.5us | 510.1us | 0 |
| minio | 1.84 MB/s | 529.3us | 808.5us | 491.5us | 809.0us | 1.1ms | 0 |
| rustfs | 1.77 MB/s | 550.8us | 693.5us | 539.5us | 694.2us | 773.5us | 0 |
| localstack | 1.21 MB/s | 810.0us | 1.3ms | 702.5us | 1.3ms | 2.5ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 4.61 MB/s
liteio_mem   █████████████████████████████ 4.58 MB/s
seaweedfs    ██████████████ 2.28 MB/s
minio        ████████████ 1.84 MB/s
rustfs       ███████████ 1.77 MB/s
localstack   ███████ 1.21 MB/s
```

**Latency (P50)**
```
liteio       ████████ 203.8us
liteio_mem   ████████ 195.8us
seaweedfs    ██████████████████ 422.9us
minio        ████████████████████ 491.5us
rustfs       ███████████████████████ 539.5us
localstack   ██████████████████████████████ 702.5us
```

### ParallelRead/1KB/C10

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 1.36 MB/s | 717.7us | 969.6us | 714.1us | 969.8us | 1.1ms | 0 |
| liteio_mem | 1.21 MB/s | 798.6us | 1.2ms | 779.2us | 1.2ms | 1.7ms | 0 |
| liteio | 1.17 MB/s | 825.4us | 2.2ms | 666.0us | 2.2ms | 2.8ms | 0 |
| seaweedfs | 0.94 MB/s | 1.0ms | 1.5ms | 949.5us | 1.5ms | 2.3ms | 0 |
| rustfs | 0.91 MB/s | 1.1ms | 1.6ms | 1.0ms | 1.6ms | 1.7ms | 0 |
| localstack | 0.18 MB/s | 5.4ms | 8.5ms | 4.8ms | 8.5ms | 9.1ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 1.36 MB/s
liteio_mem   ██████████████████████████ 1.21 MB/s
liteio       █████████████████████████ 1.17 MB/s
seaweedfs    ████████████████████ 0.94 MB/s
rustfs       ███████████████████ 0.91 MB/s
localstack   ████ 0.18 MB/s
```

**Latency (P50)**
```
minio        ████ 714.1us
liteio_mem   ████ 779.2us
liteio       ████ 666.0us
seaweedfs    █████ 949.5us
rustfs       ██████ 1.0ms
localstack   ██████████████████████████████ 4.8ms
```

### ParallelRead/1KB/C100

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio_mem | 0.31 MB/s | 3.1ms | 4.0ms | 3.2ms | 4.0ms | 4.4ms | 0 |
| liteio | 0.30 MB/s | 3.2ms | 4.7ms | 3.0ms | 4.7ms | 4.8ms | 0 |
| seaweedfs | 0.20 MB/s | 4.8ms | 6.1ms | 4.9ms | 6.1ms | 6.3ms | 0 |
| minio | 0.13 MB/s | 7.7ms | 8.8ms | 7.9ms | 8.8ms | 9.4ms | 0 |
| localstack | 0.02 MB/s | 48.3ms | 55.9ms | 52.5ms | 55.9ms | 57.0ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 0.31 MB/s
liteio       █████████████████████████████ 0.30 MB/s
seaweedfs    ███████████████████ 0.20 MB/s
minio        ████████████ 0.13 MB/s
localstack   █ 0.02 MB/s
```

**Latency (P50)**
```
liteio_mem   █ 3.2ms
liteio       █ 3.0ms
seaweedfs    ██ 4.9ms
minio        ████ 7.9ms
localstack   ██████████████████████████████ 52.5ms
```

### ParallelRead/1KB/C200

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 0.32 MB/s | 3.1ms | 5.6ms | 2.4ms | 5.6ms | 6.3ms | 0 |
| liteio | 0.28 MB/s | 3.4ms | 4.6ms | 3.4ms | 4.6ms | 4.7ms | 0 |
| liteio_mem | 0.26 MB/s | 3.7ms | 5.1ms | 4.0ms | 5.1ms | 5.2ms | 0 |
| seaweedfs | 0.19 MB/s | 5.1ms | 6.4ms | 5.3ms | 6.4ms | 6.5ms | 0 |
| localstack | 0.02 MB/s | 52.0ms | 56.3ms | 53.5ms | 56.3ms | 56.5ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 0.32 MB/s
liteio       ██████████████████████████ 0.28 MB/s
liteio_mem   ████████████████████████ 0.26 MB/s
seaweedfs    █████████████████ 0.19 MB/s
localstack   █ 0.02 MB/s
```

**Latency (P50)**
```
minio        █ 2.4ms
liteio       █ 3.4ms
liteio_mem   ██ 4.0ms
seaweedfs    ██ 5.3ms
localstack   ██████████████████████████████ 53.5ms
```

### ParallelRead/1KB/C25

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio_mem | 0.82 MB/s | 1.2ms | 1.6ms | 1.1ms | 1.6ms | 1.7ms | 0 |
| liteio | 0.81 MB/s | 1.2ms | 1.8ms | 1.2ms | 1.8ms | 1.9ms | 0 |
| minio | 0.77 MB/s | 1.3ms | 1.9ms | 1.2ms | 1.9ms | 2.4ms | 0 |
| seaweedfs | 0.52 MB/s | 1.9ms | 2.7ms | 1.9ms | 2.7ms | 3.1ms | 0 |
| localstack | 0.07 MB/s | 13.4ms | 21.7ms | 13.0ms | 21.7ms | 22.2ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 0.82 MB/s
liteio       █████████████████████████████ 0.81 MB/s
minio        ████████████████████████████ 0.77 MB/s
seaweedfs    ██████████████████ 0.52 MB/s
localstack   ██ 0.07 MB/s
```

**Latency (P50)**
```
liteio_mem   ██ 1.1ms
liteio       ██ 1.2ms
minio        ██ 1.2ms
seaweedfs    ████ 1.9ms
localstack   ██████████████████████████████ 13.0ms
```

### ParallelRead/1KB/C50

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio_mem | 0.53 MB/s | 1.8ms | 2.9ms | 1.6ms | 2.9ms | 3.1ms | 0 |
| liteio | 0.51 MB/s | 1.9ms | 2.9ms | 1.7ms | 2.9ms | 3.6ms | 0 |
| minio | 0.32 MB/s | 3.0ms | 3.7ms | 3.0ms | 3.7ms | 3.9ms | 0 |
| seaweedfs | 0.07 MB/s | 14.2ms | 26.1ms | 4.3ms | 26.1ms | 27.1ms | 0 |
| localstack | 0.04 MB/s | 25.5ms | 29.3ms | 24.8ms | 29.3ms | 30.9ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 0.53 MB/s
liteio       ████████████████████████████ 0.51 MB/s
minio        ██████████████████ 0.32 MB/s
seaweedfs    ███ 0.07 MB/s
localstack   ██ 0.04 MB/s
```

**Latency (P50)**
```
liteio_mem   █ 1.6ms
liteio       ██ 1.7ms
minio        ███ 3.0ms
seaweedfs    █████ 4.3ms
localstack   ██████████████████████████████ 24.8ms
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1.47 MB/s | 632.8us | 884.8us | 1.1ms | 0 |
| liteio | 1.45 MB/s | 635.9us | 941.4us | 1.1ms | 0 |
| seaweedfs | 1.41 MB/s | 668.5us | 816.0us | 1.1ms | 0 |
| localstack | 1.21 MB/s | 730.4us | 1.2ms | 1.7ms | 0 |
| rustfs | 1.07 MB/s | 882.0us | 1.2ms | 1.3ms | 0 |
| minio | 0.28 MB/s | 3.3ms | 6.8ms | 11.4ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 1.47 MB/s
liteio       █████████████████████████████ 1.45 MB/s
seaweedfs    ████████████████████████████ 1.41 MB/s
localstack   ████████████████████████ 1.21 MB/s
rustfs       █████████████████████ 1.07 MB/s
minio        █████ 0.28 MB/s
```

**Latency (P50)**
```
liteio_mem   █████ 632.8us
liteio       █████ 635.9us
seaweedfs    ██████ 668.5us
localstack   ██████ 730.4us
rustfs       ████████ 882.0us
minio        ██████████████████████████████ 3.3ms
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 0.49 MB/s | 1.8ms | 3.4ms | 4.2ms | 0 |
| liteio_mem | 0.41 MB/s | 2.2ms | 5.0ms | 6.5ms | 0 |
| liteio | 0.40 MB/s | 2.1ms | 6.0ms | 7.3ms | 0 |
| rustfs | 0.35 MB/s | 2.2ms | 7.6ms | 8.2ms | 0 |
| minio | 0.33 MB/s | 2.7ms | 5.2ms | 5.8ms | 0 |
| localstack | 0.19 MB/s | 4.9ms | 8.0ms | 9.0ms | 0 |

**Throughput**
```
seaweedfs    ██████████████████████████████ 0.49 MB/s
liteio_mem   ████████████████████████ 0.41 MB/s
liteio       ████████████████████████ 0.40 MB/s
rustfs       █████████████████████ 0.35 MB/s
minio        ███████████████████ 0.33 MB/s
localstack   ███████████ 0.19 MB/s
```

**Latency (P50)**
```
seaweedfs    ██████████ 1.8ms
liteio_mem   █████████████ 2.2ms
liteio       ████████████ 2.1ms
rustfs       █████████████ 2.2ms
minio        ████████████████ 2.7ms
localstack   ██████████████████████████████ 4.9ms
```

### ParallelWrite/1KB/C100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 0.09 MB/s | 10.6ms | 15.8ms | 16.3ms | 0 |
| seaweedfs | 0.09 MB/s | 11.2ms | 12.1ms | 12.4ms | 0 |
| liteio | 0.09 MB/s | 11.0ms | 16.9ms | 17.0ms | 0 |
| minio | 0.06 MB/s | 16.4ms | 25.7ms | 26.1ms | 0 |
| localstack | 0.02 MB/s | 38.3ms | 56.4ms | 56.7ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 0.09 MB/s
seaweedfs    █████████████████████████████ 0.09 MB/s
liteio       ████████████████████████████ 0.09 MB/s
minio        ██████████████████ 0.06 MB/s
localstack   ███████ 0.02 MB/s
```

**Latency (P50)**
```
liteio_mem   ████████ 10.6ms
seaweedfs    ████████ 11.2ms
liteio       ████████ 11.0ms
minio        ████████████ 16.4ms
localstack   ██████████████████████████████ 38.3ms
```

### ParallelWrite/1KB/C200

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 0.10 MB/s | 10.0ms | 11.9ms | 12.3ms | 0 |
| liteio | 0.09 MB/s | 10.4ms | 16.2ms | 17.0ms | 0 |
| liteio_mem | 0.08 MB/s | 12.8ms | 19.8ms | 20.7ms | 0 |
| minio | 0.04 MB/s | 24.0ms | 32.8ms | 33.0ms | 0 |
| localstack | 0.02 MB/s | 51.4ms | 55.8ms | 56.8ms | 0 |

**Throughput**
```
seaweedfs    ██████████████████████████████ 0.10 MB/s
liteio       ████████████████████████████ 0.09 MB/s
liteio_mem   ██████████████████████ 0.08 MB/s
minio        ████████████ 0.04 MB/s
localstack   █████ 0.02 MB/s
```

**Latency (P50)**
```
seaweedfs    █████ 10.0ms
liteio       ██████ 10.4ms
liteio_mem   ███████ 12.8ms
minio        █████████████ 24.0ms
localstack   ██████████████████████████████ 51.4ms
```

### ParallelWrite/1KB/C25

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 0.29 MB/s | 3.2ms | 5.2ms | 6.6ms | 0 |
| liteio_mem | 0.23 MB/s | 3.8ms | 7.7ms | 8.6ms | 0 |
| liteio | 0.22 MB/s | 3.9ms | 6.9ms | 8.4ms | 0 |
| minio | 0.15 MB/s | 6.3ms | 10.3ms | 14.9ms | 0 |
| localstack | 0.07 MB/s | 13.4ms | 29.3ms | 29.5ms | 0 |

**Throughput**
```
seaweedfs    ██████████████████████████████ 0.29 MB/s
liteio_mem   ███████████████████████ 0.23 MB/s
liteio       ███████████████████████ 0.22 MB/s
minio        ███████████████ 0.15 MB/s
localstack   ███████ 0.07 MB/s
```

**Latency (P50)**
```
seaweedfs    ███████ 3.2ms
liteio_mem   ████████ 3.8ms
liteio       ████████ 3.9ms
minio        ██████████████ 6.3ms
localstack   ██████████████████████████████ 13.4ms
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 0.17 MB/s | 5.4ms | 10.0ms | 10.3ms | 0 |
| liteio_mem | 0.10 MB/s | 6.9ms | 23.6ms | 24.1ms | 0 |
| minio | 0.10 MB/s | 7.7ms | 23.4ms | 23.9ms | 0 |
| liteio | 0.08 MB/s | 6.5ms | 31.8ms | 32.2ms | 0 |
| localstack | 0.03 MB/s | 24.7ms | 59.4ms | 59.6ms | 0 |

**Throughput**
```
seaweedfs    ██████████████████████████████ 0.17 MB/s
liteio_mem   █████████████████ 0.10 MB/s
minio        ████████████████ 0.10 MB/s
liteio       ██████████████ 0.08 MB/s
localstack   █████ 0.03 MB/s
```

**Latency (P50)**
```
seaweedfs    ██████ 5.4ms
liteio_mem   ████████ 6.9ms
minio        █████████ 7.7ms
liteio       ███████ 6.5ms
localstack   ██████████████████████████████ 24.7ms
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 223.30 MB/s | 1.1ms | 1.4ms | 1.5ms | 0 |
| liteio | 219.47 MB/s | 1.1ms | 1.4ms | 1.7ms | 0 |
| seaweedfs | 163.65 MB/s | 1.3ms | 2.8ms | 4.8ms | 0 |
| localstack | 152.70 MB/s | 1.5ms | 2.1ms | 2.6ms | 0 |
| rustfs | 129.18 MB/s | 1.9ms | 2.3ms | 2.3ms | 0 |
| minio | 120.24 MB/s | 1.9ms | 3.0ms | 3.6ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 223.30 MB/s
liteio       █████████████████████████████ 219.47 MB/s
seaweedfs    █████████████████████ 163.65 MB/s
localstack   ████████████████████ 152.70 MB/s
rustfs       █████████████████ 129.18 MB/s
minio        ████████████████ 120.24 MB/s
```

**Latency (P50)**
```
liteio_mem   ████████████████ 1.1ms
liteio       █████████████████ 1.1ms
seaweedfs    ████████████████████ 1.3ms
localstack   ███████████████████████ 1.5ms
rustfs       ██████████████████████████████ 1.9ms
minio        █████████████████████████████ 1.9ms
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 225.21 MB/s | 1.0ms | 1.3ms | 2.6ms | 0 |
| liteio_mem | 216.52 MB/s | 1.1ms | 1.5ms | 1.8ms | 0 |
| seaweedfs | 194.94 MB/s | 1.3ms | 1.5ms | 1.6ms | 0 |
| localstack | 163.15 MB/s | 1.5ms | 1.8ms | 2.0ms | 0 |
| rustfs | 124.15 MB/s | 2.0ms | 2.6ms | 3.0ms | 0 |
| minio | 114.40 MB/s | 2.1ms | 3.4ms | 4.7ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 225.21 MB/s
liteio_mem   ████████████████████████████ 216.52 MB/s
seaweedfs    █████████████████████████ 194.94 MB/s
localstack   █████████████████████ 163.15 MB/s
rustfs       ████████████████ 124.15 MB/s
minio        ███████████████ 114.40 MB/s
```

**Latency (P50)**
```
liteio       ███████████████ 1.0ms
liteio_mem   ███████████████ 1.1ms
seaweedfs    ██████████████████ 1.3ms
localstack   █████████████████████ 1.5ms
rustfs       ████████████████████████████ 2.0ms
minio        ██████████████████████████████ 2.1ms
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 251.81 MB/s | 973.3us | 1.1ms | 1.3ms | 0 |
| liteio_mem | 239.60 MB/s | 1.0ms | 1.2ms | 1.4ms | 0 |
| seaweedfs | 193.97 MB/s | 1.3ms | 1.5ms | 1.8ms | 0 |
| rustfs | 123.97 MB/s | 2.0ms | 2.3ms | 2.5ms | 0 |
| minio | 116.45 MB/s | 2.0ms | 2.9ms | 4.2ms | 0 |
| localstack | 109.44 MB/s | 1.5ms | 1.9ms | 2.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 251.81 MB/s
liteio_mem   ████████████████████████████ 239.60 MB/s
seaweedfs    ███████████████████████ 193.97 MB/s
rustfs       ██████████████ 123.97 MB/s
minio        █████████████ 116.45 MB/s
localstack   █████████████ 109.44 MB/s
```

**Latency (P50)**
```
liteio       ██████████████ 973.3us
liteio_mem   ███████████████ 1.0ms
seaweedfs    ██████████████████ 1.3ms
rustfs       █████████████████████████████ 2.0ms
minio        ██████████████████████████████ 2.0ms
localstack   ██████████████████████ 1.5ms
```

### Read/100MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| localstack | 284.85 MB/s | 1.5ms | 1.5ms | 349.6ms | 350.6ms | 350.6ms | 0 |
| liteio | 283.02 MB/s | 509.4us | 504.0us | 352.9ms | 353.6ms | 353.6ms | 0 |
| rustfs | 281.85 MB/s | 2.4ms | 1.9ms | 349.7ms | 363.5ms | 363.5ms | 0 |
| liteio_mem | 265.91 MB/s | 615.5us | 735.2us | 366.7ms | 382.7ms | 382.7ms | 0 |
| seaweedfs | 254.25 MB/s | 1.9ms | 2.1ms | 392.9ms | 407.3ms | 407.3ms | 0 |
| minio | 158.16 MB/s | 2.2ms | 2.5ms | 682.3ms | 698.8ms | 698.8ms | 0 |

**Throughput**
```
localstack   ██████████████████████████████ 284.85 MB/s
liteio       █████████████████████████████ 283.02 MB/s
rustfs       █████████████████████████████ 281.85 MB/s
liteio_mem   ████████████████████████████ 265.91 MB/s
seaweedfs    ██████████████████████████ 254.25 MB/s
minio        ████████████████ 158.16 MB/s
```

**Latency (P50)**
```
localstack   ███████████████ 349.6ms
liteio       ███████████████ 352.9ms
rustfs       ███████████████ 349.7ms
liteio_mem   ████████████████ 366.7ms
seaweedfs    █████████████████ 392.9ms
minio        ██████████████████████████████ 682.3ms
```

### Read/10MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| localstack | 291.60 MB/s | 1.4ms | 1.6ms | 34.3ms | 35.7ms | 35.7ms | 0 |
| seaweedfs | 277.34 MB/s | 2.1ms | 2.3ms | 35.8ms | 37.3ms | 37.3ms | 0 |
| liteio_mem | 264.61 MB/s | 559.4us | 680.2us | 36.2ms | 42.8ms | 42.8ms | 0 |
| liteio | 261.35 MB/s | 513.7us | 578.5us | 34.9ms | 42.3ms | 42.3ms | 0 |
| rustfs | 250.55 MB/s | 5.6ms | 6.5ms | 37.7ms | 43.4ms | 43.4ms | 0 |
| minio | 196.76 MB/s | 1.7ms | 2.2ms | 50.7ms | 54.2ms | 54.2ms | 0 |

**Throughput**
```
localstack   ██████████████████████████████ 291.60 MB/s
seaweedfs    ████████████████████████████ 277.34 MB/s
liteio_mem   ███████████████████████████ 264.61 MB/s
liteio       ██████████████████████████ 261.35 MB/s
rustfs       █████████████████████████ 250.55 MB/s
minio        ████████████████████ 196.76 MB/s
```

**Latency (P50)**
```
localstack   ████████████████████ 34.3ms
seaweedfs    █████████████████████ 35.8ms
liteio_mem   █████████████████████ 36.2ms
liteio       ████████████████████ 34.9ms
rustfs       ██████████████████████ 37.7ms
minio        ██████████████████████████████ 50.7ms
```

### Read/1KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio_mem | 4.89 MB/s | 199.5us | 231.4us | 199.1us | 231.5us | 258.1us | 0 |
| liteio | 4.52 MB/s | 215.7us | 252.0us | 212.5us | 252.3us | 307.8us | 0 |
| seaweedfs | 2.34 MB/s | 417.4us | 503.6us | 406.6us | 503.8us | 529.1us | 0 |
| rustfs | 1.91 MB/s | 511.4us | 670.9us | 487.8us | 671.0us | 714.7us | 0 |
| minio | 1.55 MB/s | 629.7us | 1.2ms | 497.3us | 1.2ms | 1.4ms | 0 |
| localstack | 1.45 MB/s | 671.5us | 809.8us | 645.6us | 810.0us | 1.2ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 4.89 MB/s
liteio       ███████████████████████████ 4.52 MB/s
seaweedfs    ██████████████ 2.34 MB/s
rustfs       ███████████ 1.91 MB/s
minio        █████████ 1.55 MB/s
localstack   ████████ 1.45 MB/s
```

**Latency (P50)**
```
liteio_mem   █████████ 199.1us
liteio       █████████ 212.5us
seaweedfs    ██████████████████ 406.6us
rustfs       ██████████████████████ 487.8us
minio        ███████████████████████ 497.3us
localstack   ██████████████████████████████ 645.6us
```

### Read/1MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 282.77 MB/s | 371.1us | 519.9us | 3.4ms | 4.0ms | 4.0ms | 0 |
| seaweedfs | 262.13 MB/s | 840.8us | 1.1ms | 3.8ms | 4.1ms | 4.1ms | 0 |
| liteio_mem | 260.34 MB/s | 372.9us | 638.3us | 3.5ms | 5.1ms | 5.1ms | 0 |
| localstack | 253.38 MB/s | 1.0ms | 1.2ms | 3.8ms | 4.5ms | 4.5ms | 0 |
| rustfs | 233.07 MB/s | 1.4ms | 1.7ms | 4.3ms | 4.5ms | 4.5ms | 0 |
| minio | 133.67 MB/s | 2.1ms | 2.6ms | 6.7ms | 11.0ms | 11.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 282.77 MB/s
seaweedfs    ███████████████████████████ 262.13 MB/s
liteio_mem   ███████████████████████████ 260.34 MB/s
localstack   ██████████████████████████ 253.38 MB/s
rustfs       ████████████████████████ 233.07 MB/s
minio        ██████████████ 133.67 MB/s
```

**Latency (P50)**
```
liteio       ███████████████ 3.4ms
seaweedfs    ████████████████ 3.8ms
liteio_mem   ███████████████ 3.5ms
localstack   █████████████████ 3.8ms
rustfs       ███████████████████ 4.3ms
minio        ██████████████████████████████ 6.7ms
```

### Read/64KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio_mem | 154.84 MB/s | 235.9us | 316.3us | 391.1us | 438.5us | 451.9us | 0 |
| liteio | 135.30 MB/s | 288.0us | 447.4us | 448.1us | 595.6us | 676.9us | 0 |
| seaweedfs | 102.14 MB/s | 438.0us | 533.2us | 598.0us | 687.9us | 692.7us | 0 |
| rustfs | 89.92 MB/s | 576.6us | 742.9us | 671.1us | 840.2us | 887.0us | 0 |
| localstack | 72.43 MB/s | 747.4us | 850.8us | 844.2us | 911.3us | 985.9us | 0 |
| minio | 60.23 MB/s | 709.6us | 1.1ms | 999.9us | 1.4ms | 1.4ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 154.84 MB/s
liteio       ██████████████████████████ 135.30 MB/s
seaweedfs    ███████████████████ 102.14 MB/s
rustfs       █████████████████ 89.92 MB/s
localstack   ██████████████ 72.43 MB/s
minio        ███████████ 60.23 MB/s
```

**Latency (P50)**
```
liteio_mem   ███████████ 391.1us
liteio       █████████████ 448.1us
seaweedfs    █████████████████ 598.0us
rustfs       ████████████████████ 671.1us
localstack   █████████████████████████ 844.2us
minio        ██████████████████████████████ 999.9us
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 5740 ops/s | 169.4us | 206.7us | 230.4us | 0 |
| liteio_mem | 5584 ops/s | 159.0us | 216.5us | 256.4us | 0 |
| seaweedfs | 2969 ops/s | 290.7us | 554.0us | 803.6us | 0 |
| rustfs | 2962 ops/s | 312.7us | 487.5us | 625.8us | 0 |
| localstack | 1645 ops/s | 594.5us | 679.3us | 964.5us | 0 |
| minio | 752 ops/s | 1.1ms | 2.0ms | 3.4ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 5740 ops/s
liteio_mem   █████████████████████████████ 5584 ops/s
seaweedfs    ███████████████ 2969 ops/s
rustfs       ███████████████ 2962 ops/s
localstack   ████████ 1645 ops/s
minio        ███ 752 ops/s
```

**Latency (P50)**
```
liteio       ████ 169.4us
liteio_mem   ████ 159.0us
seaweedfs    ███████ 290.7us
rustfs       ████████ 312.7us
localstack   ███████████████ 594.5us
minio        ██████████████████████████████ 1.1ms
```

### Write/100MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 199.45 MB/s | 507.5ms | 510.3ms | 510.3ms | 0 |
| liteio | 192.03 MB/s | 523.2ms | 524.6ms | 524.6ms | 0 |
| liteio_mem | 178.21 MB/s | 548.2ms | 559.0ms | 559.0ms | 0 |
| rustfs | 175.98 MB/s | 588.0ms | 593.3ms | 593.3ms | 0 |
| localstack | 154.95 MB/s | 637.7ms | 652.3ms | 652.3ms | 0 |
| minio | 97.90 MB/s | 1.01s | 1.03s | 1.03s | 0 |

**Throughput**
```
seaweedfs    ██████████████████████████████ 199.45 MB/s
liteio       ████████████████████████████ 192.03 MB/s
liteio_mem   ██████████████████████████ 178.21 MB/s
rustfs       ██████████████████████████ 175.98 MB/s
localstack   ███████████████████████ 154.95 MB/s
minio        ██████████████ 97.90 MB/s
```

**Latency (P50)**
```
seaweedfs    ███████████████ 507.5ms
liteio       ███████████████ 523.2ms
liteio_mem   ████████████████ 548.2ms
rustfs       █████████████████ 588.0ms
localstack   ███████████████████ 637.7ms
minio        ██████████████████████████████ 1.01s
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 187.63 MB/s | 53.4ms | 55.8ms | 55.8ms | 0 |
| rustfs | 187.41 MB/s | 48.7ms | 62.1ms | 62.1ms | 0 |
| liteio | 184.98 MB/s | 52.4ms | 55.2ms | 55.2ms | 0 |
| seaweedfs | 154.15 MB/s | 62.0ms | 67.9ms | 67.9ms | 0 |
| localstack | 149.26 MB/s | 66.9ms | 69.5ms | 69.5ms | 0 |
| minio | 90.17 MB/s | 113.9ms | 121.0ms | 121.0ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 187.63 MB/s
rustfs       █████████████████████████████ 187.41 MB/s
liteio       █████████████████████████████ 184.98 MB/s
seaweedfs    ████████████████████████ 154.15 MB/s
localstack   ███████████████████████ 149.26 MB/s
minio        ██████████████ 90.17 MB/s
```

**Latency (P50)**
```
liteio_mem   ██████████████ 53.4ms
rustfs       ████████████ 48.7ms
liteio       █████████████ 52.4ms
seaweedfs    ████████████████ 62.0ms
localstack   █████████████████ 66.9ms
minio        ██████████████████████████████ 113.9ms
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 1.43 MB/s | 684.7us | 748.5us | 947.5us | 0 |
| seaweedfs | 1.36 MB/s | 698.5us | 856.6us | 1.1ms | 0 |
| localstack | 1.35 MB/s | 693.8us | 903.1us | 1.1ms | 0 |
| liteio_mem | 1.22 MB/s | 723.0us | 1.2ms | 2.0ms | 0 |
| liteio | 1.21 MB/s | 751.1us | 1.1ms | 1.2ms | 0 |
| minio | 0.55 MB/s | 1.7ms | 2.5ms | 3.3ms | 0 |

**Throughput**
```
rustfs       ██████████████████████████████ 1.43 MB/s
seaweedfs    ████████████████████████████ 1.36 MB/s
localstack   ████████████████████████████ 1.35 MB/s
liteio_mem   █████████████████████████ 1.22 MB/s
liteio       █████████████████████████ 1.21 MB/s
minio        ███████████ 0.55 MB/s
```

**Latency (P50)**
```
rustfs       ███████████ 684.7us
seaweedfs    ████████████ 698.5us
localstack   ████████████ 693.8us
liteio_mem   ████████████ 723.0us
liteio       █████████████ 751.1us
minio        ██████████████████████████████ 1.7ms
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 173.30 MB/s | 5.7ms | 6.2ms | 6.2ms | 0 |
| liteio | 163.08 MB/s | 6.0ms | 7.1ms | 7.1ms | 0 |
| rustfs | 151.40 MB/s | 6.4ms | 7.2ms | 7.2ms | 0 |
| localstack | 132.20 MB/s | 7.5ms | 7.9ms | 7.9ms | 0 |
| seaweedfs | 127.06 MB/s | 7.8ms | 8.3ms | 8.3ms | 0 |
| minio | 72.51 MB/s | 13.0ms | 18.7ms | 18.7ms | 0 |

**Throughput**
```
liteio_mem   ██████████████████████████████ 173.30 MB/s
liteio       ████████████████████████████ 163.08 MB/s
rustfs       ██████████████████████████ 151.40 MB/s
localstack   ██████████████████████ 132.20 MB/s
seaweedfs    █████████████████████ 127.06 MB/s
minio        ████████████ 72.51 MB/s
```

**Latency (P50)**
```
liteio_mem   █████████████ 5.7ms
liteio       █████████████ 6.0ms
rustfs       ██████████████ 6.4ms
localstack   █████████████████ 7.5ms
seaweedfs    █████████████████ 7.8ms
minio        ██████████████████████████████ 13.0ms
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 65.34 MB/s | 905.0us | 1.2ms | 1.5ms | 0 |
| liteio_mem | 64.60 MB/s | 874.1us | 1.2ms | 2.0ms | 0 |
| localstack | 61.17 MB/s | 984.8us | 1.2ms | 1.3ms | 0 |
| rustfs | 58.89 MB/s | 1.1ms | 1.2ms | 1.2ms | 0 |
| seaweedfs | 56.91 MB/s | 1.1ms | 1.2ms | 1.5ms | 0 |
| minio | 20.40 MB/s | 3.1ms | 3.8ms | 4.0ms | 0 |

**Throughput**
```
liteio       ██████████████████████████████ 65.34 MB/s
liteio_mem   █████████████████████████████ 64.60 MB/s
localstack   ████████████████████████████ 61.17 MB/s
rustfs       ███████████████████████████ 58.89 MB/s
seaweedfs    ██████████████████████████ 56.91 MB/s
minio        █████████ 20.40 MB/s
```

**Latency (P50)**
```
liteio       ████████ 905.0us
liteio_mem   ████████ 874.1us
localstack   █████████ 984.8us
rustfs       ██████████ 1.1ms
seaweedfs    ██████████ 1.1ms
minio        ██████████████████████████████ 3.1ms
```

## Recommendations

- **Write-heavy workloads:** seaweedfs
- **Read-heavy workloads:** localstack

---

*Generated by storage benchmark CLI*
