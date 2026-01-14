# Storage Benchmark Report

**Generated:** 2026-01-15T04:08:31+07:00

**Go Version:** go1.25.5

**Platform:** darwin/arm64

## Executive Summary

### Performance Leaders

```
┌───────────────────────────┬───────────────────────┬───────────────────────────────┐
│         Category          │        Leader         │             Notes             │
├───────────────────────────┼───────────────────────┼───────────────────────────────┤
│ Small File Read (1KB)     │ liteio 4.7 MB/s       │ Close competition             │
├───────────────────────────┼───────────────────────┼───────────────────────────────┤
│ Small File Write (1KB)    │ rustfs 1.2 MB/s       │ Close competition             │
├───────────────────────────┼───────────────────────┼───────────────────────────────┤
│ Delete Operations         │ liteio 5592 ops/s     │ Close competition             │
├───────────────────────────┼───────────────────────┼───────────────────────────────┤
│ Stat Operations           │ liteio 6879 ops/s     │ 17% faster than liteio_mem    │
├───────────────────────────┼───────────────────────┼───────────────────────────────┤
│ List Operations (100 obj) │ liteio_mem 1255 ops/s │ Close competition             │
├───────────────────────────┼───────────────────────┼───────────────────────────────┤
│ Copy Operations           │ liteio_mem 1.4 MB/s   │ 14% faster than localstack    │
├───────────────────────────┼───────────────────────┼───────────────────────────────┤
│ Range Reads               │ liteio 232.9 MB/s     │ Close competition             │
├───────────────────────────┼───────────────────────┼───────────────────────────────┤
│ Mixed Workload            │ rustfs 9.6 MB/s       │ Close competition             │
└───────────────────────────┴───────────────────────┴───────────────────────────────┘
```

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (10MB+) | **liteio** | 191 MB/s | Best for media, backups |
| Large File Downloads (10MB) | **liteio_mem** | 314 MB/s | Best for streaming, CDN |
| Small File Operations | **liteio** | 2841 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **minio** | - | Best for multi-user apps |
| Memory Constrained | **liteio** | 20 MB RAM | Best for edge/embedded |

### Large File Performance (10MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| liteio | 191.4 | 310.4 | 51.8ms | 31.1ms |
| liteio_mem | 180.5 | 313.5 | 50.9ms | 32.0ms |
| localstack | 140.2 | 305.9 | 69.8ms | 32.6ms |
| minio | 172.6 | 304.3 | 58.2ms | 32.4ms |
| rustfs | 81.6 | 262.4 | 132.0ms | 37.7ms |
| seaweedfs | 153.0 | 273.1 | 64.6ms | 35.0ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| liteio | 853 | 4830 | 1.0ms | 207.3us |
| liteio_mem | 920 | 4741 | 1.0ms | 207.5us |
| localstack | 1058 | 1409 | 921.0us | 708.6us |
| minio | 782 | 2380 | 1.2ms | 406.1us |
| rustfs | 1233 | 2185 | 763.3us | 453.9us |
| seaweedfs | 1186 | 2312 | 809.6us | 422.8us |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| liteio | 6879 | 1182 | 5592 |
| liteio_mem | 5899 | 1255 | 5099 |
| localstack | 1458 | 340 | 1625 |
| minio | 3267 | 514 | 2618 |
| rustfs | 3217 | 163 | 1230 |
| seaweedfs | 3489 | 664 | 2981 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| liteio | 1.28 | 0.25 | 0.23 |
| liteio_mem | 1.19 | 0.26 | 0.21 |
| localstack | 1.19 | 0.16 | 0.10 |
| minio | 1.04 | 0.28 | 0.16 |
| rustfs | 1.23 | 0.35 | - |
| seaweedfs | 1.35 | 0.40 | 0.25 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C50 |
|--------|------|------|------|
| liteio | 3.96 | 0.64 | 0.55 |
| liteio_mem | 3.37 | 0.77 | 0.69 |
| localstack | 1.25 | 0.17 | 0.08 |
| minio | 2.65 | 0.84 | 0.62 |
| rustfs | 1.86 | 0.76 | - |
| seaweedfs | 2.16 | 0.69 | 0.46 |

*\* indicates errors occurred*

### File Count Performance

Performance with varying numbers of files (1KB each).

**Write N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|------|
| liteio | 926.9us | 6.1ms | 60.9ms | 624.4ms | 6.45s |
| liteio_mem | 638.5us | 5.6ms | 60.2ms | 643.7ms | 6.50s |
| localstack | 682.9us | 7.3ms | 94.3ms | 878.1ms | 8.01s |
| minio | 1.1ms | 9.6ms | 90.2ms | 892.4ms | 9.30s |
| rustfs | 749.3us | 7.2ms | 69.5ms | 673.9ms | 7.13s |
| seaweedfs | 685.2us | 7.5ms | 70.5ms | 720.1ms | 6.98s |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 1 | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|------|
| liteio | 279.6us | 341.0us | 940.8us | 5.8ms | 190.7ms |
| liteio_mem | 309.0us | 277.2us | 836.0us | 5.7ms | 197.1ms |
| localstack | 1.1ms | 1.1ms | 4.4ms | 23.9ms | 241.8ms |
| minio | 532.5us | 714.4us | 2.0ms | 14.5ms | 163.0ms |
| rustfs | 1.1ms | 1.4ms | 6.7ms | 53.9ms | 709.0ms |
| seaweedfs | 722.5us | 779.0us | 1.9ms | 8.8ms | 88.2ms |

*\* indicates errors occurred*

### Skipped Benchmarks

Some benchmarks were skipped due to driver limitations:

- **rustfs**: 2 skipped
  - ParallelWrite/1KB/C50 (exceeds max concurrency 10)
  - ParallelRead/1KB/C50 (exceeds max concurrency 10)

### Resource Usage Summary

| Driver | Memory | CPU |
|--------|--------|-----|
| liteio | 20.2 MB | 0.0% |
| liteio_mem | 53.0 MB | 0.8% |
| localstack | 502.9 MB | 0.1% |
| minio | 546.3 MB | 0.1% |
| rustfs | 407.9 MB | 0.1% |
| seaweedfs | 64.7 MB | 0.0% |

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
| liteio_mem | 1.39 MB/s | 625.8us | 956.5us | 956.5us | 0 |
| localstack | 1.21 MB/s | 779.4us | 853.8us | 853.8us | 0 |
| liteio | 1.05 MB/s | 866.0us | 1.2ms | 1.2ms | 0 |
| rustfs | 0.97 MB/s | 974.2us | 1.3ms | 1.3ms | 0 |
| minio | 0.94 MB/s | 1.0ms | 1.2ms | 1.2ms | 0 |
| seaweedfs | 0.87 MB/s | 984.5us | 1.5ms | 1.5ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 1.39 MB/s
  localstack   ██████████████████████████████████ 1.21 MB/s
  liteio       ██████████████████████████████ 1.05 MB/s
  rustfs       ███████████████████████████ 0.97 MB/s
  minio        ███████████████████████████ 0.94 MB/s
  seaweedfs    █████████████████████████ 0.87 MB/s
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 5592 ops/s | 167.5us | 204.5us | 204.5us | 0 |
| liteio_mem | 5099 ops/s | 173.5us | 255.1us | 255.1us | 0 |
| seaweedfs | 2981 ops/s | 297.5us | 506.3us | 506.3us | 0 |
| minio | 2618 ops/s | 360.5us | 557.2us | 557.2us | 0 |
| localstack | 1625 ops/s | 581.0us | 757.7us | 757.7us | 0 |
| rustfs | 1230 ops/s | 785.9us | 954.0us | 954.0us | 0 |

```
  liteio       ████████████████████████████████████████ 5592 ops/s
  liteio_mem   ████████████████████████████████████ 5099 ops/s
  seaweedfs    █████████████████████ 2981 ops/s
  minio        ██████████████████ 2618 ops/s
  localstack   ███████████ 1625 ops/s
  rustfs       ████████ 1230 ops/s
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 0.15 MB/s | 594.0us | 652.0us | 652.0us | 0 |
| liteio | 0.15 MB/s | 605.5us | 693.1us | 693.1us | 0 |
| seaweedfs | 0.14 MB/s | 683.1us | 717.4us | 717.4us | 0 |
| localstack | 0.14 MB/s | 685.3us | 762.7us | 762.7us | 0 |
| rustfs | 0.13 MB/s | 680.0us | 778.1us | 778.1us | 0 |
| minio | 0.06 MB/s | 1.3ms | 2.8ms | 2.8ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 0.15 MB/s
  liteio       ███████████████████████████████████████ 0.15 MB/s
  seaweedfs    ████████████████████████████████████ 0.14 MB/s
  localstack   ███████████████████████████████████ 0.14 MB/s
  rustfs       █████████████████████████████████ 0.13 MB/s
  minio        ███████████████ 0.06 MB/s
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 1349 ops/s | 559.8us | 933.5us | 933.5us | 0 |
| rustfs | 1208 ops/s | 774.5us | 925.5us | 925.5us | 0 |
| localstack | 1150 ops/s | 809.2us | 1.0ms | 1.0ms | 0 |
| liteio | 1013 ops/s | 580.2us | 866.5us | 866.5us | 0 |
| minio | 818 ops/s | 942.0us | 2.0ms | 2.0ms | 0 |
| liteio_mem | 177 ops/s | 796.4us | 3.7ms | 3.7ms | 0 |

```
  seaweedfs    ████████████████████████████████████████ 1349 ops/s
  rustfs       ███████████████████████████████████ 1208 ops/s
  localstack   ██████████████████████████████████ 1150 ops/s
  liteio       ██████████████████████████████ 1013 ops/s
  minio        ████████████████████████ 818 ops/s
  liteio_mem   █████ 177 ops/s
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 0.16 MB/s | 569.2us | 630.3us | 630.3us | 0 |
| liteio_mem | 0.14 MB/s | 625.1us | 934.1us | 934.1us | 0 |
| rustfs | 0.14 MB/s | 700.1us | 742.9us | 742.9us | 0 |
| seaweedfs | 0.11 MB/s | 782.5us | 1.0ms | 1.0ms | 0 |
| localstack | 0.10 MB/s | 795.9us | 902.4us | 902.4us | 0 |
| minio | 0.05 MB/s | 1.5ms | 2.9ms | 2.9ms | 0 |

```
  liteio       ████████████████████████████████████████ 0.16 MB/s
  liteio_mem   ██████████████████████████████████ 0.14 MB/s
  rustfs       █████████████████████████████████ 0.14 MB/s
  seaweedfs    ███████████████████████████ 0.11 MB/s
  localstack   █████████████████████████ 0.10 MB/s
  minio        ████████████ 0.05 MB/s
```

### FileCount/Delete/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 4988 ops/s | 200.5us | 200.5us | 200.5us | 0 |
| liteio | 4512 ops/s | 221.6us | 221.6us | 221.6us | 0 |
| seaweedfs | 2796 ops/s | 357.7us | 357.7us | 357.7us | 0 |
| minio | 2282 ops/s | 438.1us | 438.1us | 438.1us | 0 |
| localstack | 1501 ops/s | 666.2us | 666.2us | 666.2us | 0 |
| rustfs | 1005 ops/s | 994.9us | 994.9us | 994.9us | 0 |

```
  liteio_mem   ████████████████████████████████████████ 4988 ops/s
  liteio       ████████████████████████████████████ 4512 ops/s
  seaweedfs    ██████████████████████ 2796 ops/s
  minio        ██████████████████ 2282 ops/s
  localstack   ████████████ 1501 ops/s
  rustfs       ████████ 1005 ops/s
```

### FileCount/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 643 ops/s | 1.6ms | 1.6ms | 1.6ms | 0 |
| liteio | 469 ops/s | 2.1ms | 2.1ms | 2.1ms | 0 |
| seaweedfs | 299 ops/s | 3.3ms | 3.3ms | 3.3ms | 0 |
| minio | 218 ops/s | 4.6ms | 4.6ms | 4.6ms | 0 |
| localstack | 176 ops/s | 5.7ms | 5.7ms | 5.7ms | 0 |
| rustfs | 112 ops/s | 8.9ms | 8.9ms | 8.9ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 643 ops/s
  liteio       █████████████████████████████ 469 ops/s
  seaweedfs    ██████████████████ 299 ops/s
  minio        █████████████ 218 ops/s
  localstack   ██████████ 176 ops/s
  rustfs       ██████ 112 ops/s
```

### FileCount/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 51 ops/s | 19.4ms | 19.4ms | 19.4ms | 0 |
| liteio | 48 ops/s | 20.7ms | 20.7ms | 20.7ms | 0 |
| seaweedfs | 31 ops/s | 32.5ms | 32.5ms | 32.5ms | 0 |
| minio | 28 ops/s | 36.2ms | 36.2ms | 36.2ms | 0 |
| localstack | 13 ops/s | 79.1ms | 79.1ms | 79.1ms | 0 |
| rustfs | 12 ops/s | 81.6ms | 81.6ms | 81.6ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 51 ops/s
  liteio       █████████████████████████████████████ 48 ops/s
  seaweedfs    ███████████████████████ 31 ops/s
  minio        █████████████████████ 28 ops/s
  localstack   █████████ 13 ops/s
  rustfs       █████████ 12 ops/s
```

### FileCount/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 5 ops/s | 187.4ms | 187.4ms | 187.4ms | 0 |
| liteio_mem | 5 ops/s | 192.5ms | 192.5ms | 192.5ms | 0 |
| seaweedfs | 3 ops/s | 329.3ms | 329.3ms | 329.3ms | 0 |
| minio | 3 ops/s | 376.3ms | 376.3ms | 376.3ms | 0 |
| localstack | 2 ops/s | 618.9ms | 618.9ms | 618.9ms | 0 |
| rustfs | 1 ops/s | 821.6ms | 821.6ms | 821.6ms | 0 |

```
  liteio       ████████████████████████████████████████ 5 ops/s
  liteio_mem   ██████████████████████████████████████ 5 ops/s
  seaweedfs    ██████████████████████ 3 ops/s
  minio        ███████████████████ 3 ops/s
  localstack   ████████████ 2 ops/s
  rustfs       █████████ 1 ops/s
```

### FileCount/Delete/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1 ops/s | 1.87s | 1.87s | 1.87s | 0 |
| liteio | 1 ops/s | 1.89s | 1.89s | 1.89s | 0 |
| seaweedfs | 0 ops/s | 3.29s | 3.29s | 3.29s | 0 |
| minio | 0 ops/s | 3.50s | 3.50s | 3.50s | 0 |
| localstack | 0 ops/s | 6.16s | 6.16s | 6.16s | 0 |
| rustfs | 0 ops/s | 8.10s | 8.10s | 8.10s | 0 |

```
  liteio_mem   ████████████████████████████████████████ 1 ops/s
  liteio       ███████████████████████████████████████ 1 ops/s
  seaweedfs    ██████████████████████ 0 ops/s
  minio        █████████████████████ 0 ops/s
  localstack   ████████████ 0 ops/s
  rustfs       █████████ 0 ops/s
```

### FileCount/List/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 3577 ops/s | 279.6us | 279.6us | 279.6us | 0 |
| liteio_mem | 3236 ops/s | 309.0us | 309.0us | 309.0us | 0 |
| minio | 1878 ops/s | 532.5us | 532.5us | 532.5us | 0 |
| seaweedfs | 1384 ops/s | 722.5us | 722.5us | 722.5us | 0 |
| localstack | 948 ops/s | 1.1ms | 1.1ms | 1.1ms | 0 |
| rustfs | 931 ops/s | 1.1ms | 1.1ms | 1.1ms | 0 |

```
  liteio       ████████████████████████████████████████ 3577 ops/s
  liteio_mem   ████████████████████████████████████ 3236 ops/s
  minio        █████████████████████ 1878 ops/s
  seaweedfs    ███████████████ 1384 ops/s
  localstack   ██████████ 948 ops/s
  rustfs       ██████████ 931 ops/s
```

### FileCount/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 3607 ops/s | 277.2us | 277.2us | 277.2us | 0 |
| liteio | 2932 ops/s | 341.0us | 341.0us | 341.0us | 0 |
| minio | 1400 ops/s | 714.4us | 714.4us | 714.4us | 0 |
| seaweedfs | 1284 ops/s | 779.0us | 779.0us | 779.0us | 0 |
| localstack | 892 ops/s | 1.1ms | 1.1ms | 1.1ms | 0 |
| rustfs | 694 ops/s | 1.4ms | 1.4ms | 1.4ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 3607 ops/s
  liteio       ████████████████████████████████ 2932 ops/s
  minio        ███████████████ 1400 ops/s
  seaweedfs    ██████████████ 1284 ops/s
  localstack   █████████ 892 ops/s
  rustfs       ███████ 694 ops/s
```

### FileCount/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1196 ops/s | 836.0us | 836.0us | 836.0us | 0 |
| liteio | 1063 ops/s | 940.8us | 940.8us | 940.8us | 0 |
| seaweedfs | 533 ops/s | 1.9ms | 1.9ms | 1.9ms | 0 |
| minio | 500 ops/s | 2.0ms | 2.0ms | 2.0ms | 0 |
| localstack | 225 ops/s | 4.4ms | 4.4ms | 4.4ms | 0 |
| rustfs | 149 ops/s | 6.7ms | 6.7ms | 6.7ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 1196 ops/s
  liteio       ███████████████████████████████████ 1063 ops/s
  seaweedfs    █████████████████ 533 ops/s
  minio        ████████████████ 500 ops/s
  localstack   ███████ 225 ops/s
  rustfs       ████ 149 ops/s
```

### FileCount/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 176 ops/s | 5.7ms | 5.7ms | 5.7ms | 0 |
| liteio | 172 ops/s | 5.8ms | 5.8ms | 5.8ms | 0 |
| seaweedfs | 114 ops/s | 8.8ms | 8.8ms | 8.8ms | 0 |
| minio | 69 ops/s | 14.5ms | 14.5ms | 14.5ms | 0 |
| localstack | 42 ops/s | 23.9ms | 23.9ms | 23.9ms | 0 |
| rustfs | 19 ops/s | 53.9ms | 53.9ms | 53.9ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 176 ops/s
  liteio       ███████████████████████████████████████ 172 ops/s
  seaweedfs    █████████████████████████ 114 ops/s
  minio        ███████████████ 69 ops/s
  localstack   █████████ 42 ops/s
  rustfs       ████ 19 ops/s
```

### FileCount/List/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 11 ops/s | 88.2ms | 88.2ms | 88.2ms | 0 |
| minio | 6 ops/s | 163.0ms | 163.0ms | 163.0ms | 0 |
| liteio | 5 ops/s | 190.7ms | 190.7ms | 190.7ms | 0 |
| liteio_mem | 5 ops/s | 197.1ms | 197.1ms | 197.1ms | 0 |
| localstack | 4 ops/s | 241.8ms | 241.8ms | 241.8ms | 0 |
| rustfs | 1 ops/s | 709.0ms | 709.0ms | 709.0ms | 0 |

```
  seaweedfs    ████████████████████████████████████████ 11 ops/s
  minio        █████████████████████ 6 ops/s
  liteio       ██████████████████ 5 ops/s
  liteio_mem   █████████████████ 5 ops/s
  localstack   ██████████████ 4 ops/s
  rustfs       ████ 1 ops/s
```

### FileCount/Write/1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1.53 MB/s | 638.5us | 638.5us | 638.5us | 0 |
| localstack | 1.43 MB/s | 682.9us | 682.9us | 682.9us | 0 |
| seaweedfs | 1.43 MB/s | 685.2us | 685.2us | 685.2us | 0 |
| rustfs | 1.30 MB/s | 749.3us | 749.3us | 749.3us | 0 |
| liteio | 1.05 MB/s | 926.9us | 926.9us | 926.9us | 0 |
| minio | 0.90 MB/s | 1.1ms | 1.1ms | 1.1ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 1.53 MB/s
  localstack   █████████████████████████████████████ 1.43 MB/s
  seaweedfs    █████████████████████████████████████ 1.43 MB/s
  rustfs       ██████████████████████████████████ 1.30 MB/s
  liteio       ███████████████████████████ 1.05 MB/s
  minio        ███████████████████████ 0.90 MB/s
```

### FileCount/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1.74 MB/s | 5.6ms | 5.6ms | 5.6ms | 0 |
| liteio | 1.59 MB/s | 6.1ms | 6.1ms | 6.1ms | 0 |
| rustfs | 1.36 MB/s | 7.2ms | 7.2ms | 7.2ms | 0 |
| localstack | 1.33 MB/s | 7.3ms | 7.3ms | 7.3ms | 0 |
| seaweedfs | 1.30 MB/s | 7.5ms | 7.5ms | 7.5ms | 0 |
| minio | 1.02 MB/s | 9.6ms | 9.6ms | 9.6ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 1.74 MB/s
  liteio       ████████████████████████████████████ 1.59 MB/s
  rustfs       ███████████████████████████████ 1.36 MB/s
  localstack   ██████████████████████████████ 1.33 MB/s
  seaweedfs    █████████████████████████████ 1.30 MB/s
  minio        ███████████████████████ 1.02 MB/s
```

### FileCount/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1.62 MB/s | 60.2ms | 60.2ms | 60.2ms | 0 |
| liteio | 1.60 MB/s | 60.9ms | 60.9ms | 60.9ms | 0 |
| rustfs | 1.41 MB/s | 69.5ms | 69.5ms | 69.5ms | 0 |
| seaweedfs | 1.38 MB/s | 70.5ms | 70.5ms | 70.5ms | 0 |
| minio | 1.08 MB/s | 90.2ms | 90.2ms | 90.2ms | 0 |
| localstack | 1.04 MB/s | 94.3ms | 94.3ms | 94.3ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 1.62 MB/s
  liteio       ███████████████████████████████████████ 1.60 MB/s
  rustfs       ██████████████████████████████████ 1.41 MB/s
  seaweedfs    ██████████████████████████████████ 1.38 MB/s
  minio        ██████████████████████████ 1.08 MB/s
  localstack   █████████████████████████ 1.04 MB/s
```

### FileCount/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.56 MB/s | 624.4ms | 624.4ms | 624.4ms | 0 |
| liteio_mem | 1.52 MB/s | 643.7ms | 643.7ms | 643.7ms | 0 |
| rustfs | 1.45 MB/s | 673.9ms | 673.9ms | 673.9ms | 0 |
| seaweedfs | 1.36 MB/s | 720.1ms | 720.1ms | 720.1ms | 0 |
| localstack | 1.11 MB/s | 878.1ms | 878.1ms | 878.1ms | 0 |
| minio | 1.09 MB/s | 892.4ms | 892.4ms | 892.4ms | 0 |

```
  liteio       ████████████████████████████████████████ 1.56 MB/s
  liteio_mem   ██████████████████████████████████████ 1.52 MB/s
  rustfs       █████████████████████████████████████ 1.45 MB/s
  seaweedfs    ██████████████████████████████████ 1.36 MB/s
  localstack   ████████████████████████████ 1.11 MB/s
  minio        ███████████████████████████ 1.09 MB/s
```

### FileCount/Write/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 1.51 MB/s | 6.45s | 6.45s | 6.45s | 0 |
| liteio_mem | 1.50 MB/s | 6.50s | 6.50s | 6.50s | 0 |
| seaweedfs | 1.40 MB/s | 6.98s | 6.98s | 6.98s | 0 |
| rustfs | 1.37 MB/s | 7.13s | 7.13s | 7.13s | 0 |
| localstack | 1.22 MB/s | 8.01s | 8.01s | 8.01s | 0 |
| minio | 1.05 MB/s | 9.30s | 9.30s | 9.30s | 0 |

```
  liteio       ████████████████████████████████████████ 1.51 MB/s
  liteio_mem   ███████████████████████████████████████ 1.50 MB/s
  seaweedfs    ████████████████████████████████████ 1.40 MB/s
  rustfs       ████████████████████████████████████ 1.37 MB/s
  localstack   ████████████████████████████████ 1.22 MB/s
  minio        ███████████████████████████ 1.05 MB/s
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 1255 ops/s | 697.8us | 1.0ms | 1.0ms | 0 |
| liteio | 1182 ops/s | 732.2us | 1.5ms | 1.5ms | 0 |
| seaweedfs | 664 ops/s | 1.5ms | 1.7ms | 1.7ms | 0 |
| minio | 514 ops/s | 1.9ms | 2.3ms | 2.3ms | 0 |
| localstack | 340 ops/s | 2.9ms | 3.2ms | 3.2ms | 0 |
| rustfs | 163 ops/s | 6.1ms | 6.4ms | 6.4ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 1255 ops/s
  liteio       █████████████████████████████████████ 1182 ops/s
  seaweedfs    █████████████████████ 664 ops/s
  minio        ████████████████ 514 ops/s
  localstack   ██████████ 340 ops/s
  rustfs       █████ 163 ops/s
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 9.60 MB/s | 1.6ms | 2.1ms | 2.1ms | 0 |
| liteio_mem | 9.55 MB/s | 1.7ms | 2.1ms | 2.1ms | 0 |
| minio | 8.92 MB/s | 1.8ms | 2.4ms | 2.4ms | 0 |
| liteio | 8.42 MB/s | 1.9ms | 2.4ms | 2.4ms | 0 |
| seaweedfs | 7.05 MB/s | 2.3ms | 2.8ms | 2.8ms | 0 |
| localstack | 1.33 MB/s | 12.1ms | 13.8ms | 13.8ms | 0 |

```
  rustfs       ████████████████████████████████████████ 9.60 MB/s
  liteio_mem   ███████████████████████████████████████ 9.55 MB/s
  minio        █████████████████████████████████████ 8.92 MB/s
  liteio       ███████████████████████████████████ 8.42 MB/s
  seaweedfs    █████████████████████████████ 7.05 MB/s
  localstack   █████ 1.33 MB/s
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 9.14 MB/s | 1.8ms | 2.5ms | 2.5ms | 0 |
| minio | 6.87 MB/s | 2.5ms | 2.9ms | 2.9ms | 0 |
| liteio_mem | 6.53 MB/s | 2.5ms | 2.9ms | 2.9ms | 0 |
| seaweedfs | 6.11 MB/s | 2.6ms | 3.0ms | 3.0ms | 0 |
| liteio | 5.37 MB/s | 3.0ms | 3.4ms | 3.4ms | 0 |
| localstack | 1.28 MB/s | 13.6ms | 14.1ms | 14.1ms | 0 |

```
  rustfs       ████████████████████████████████████████ 9.14 MB/s
  minio        ██████████████████████████████ 6.87 MB/s
  liteio_mem   ████████████████████████████ 6.53 MB/s
  seaweedfs    ██████████████████████████ 6.11 MB/s
  liteio       ███████████████████████ 5.37 MB/s
  localstack   █████ 1.28 MB/s
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 6.12 MB/s | 2.5ms | 3.8ms | 3.8ms | 0 |
| liteio | 5.61 MB/s | 3.3ms | 3.8ms | 3.8ms | 0 |
| seaweedfs | 5.13 MB/s | 3.5ms | 3.9ms | 3.9ms | 0 |
| liteio_mem | 4.12 MB/s | 4.3ms | 5.2ms | 5.2ms | 0 |
| minio | 3.79 MB/s | 4.9ms | 5.8ms | 5.8ms | 0 |
| localstack | 1.44 MB/s | 13.0ms | 13.3ms | 13.3ms | 0 |

```
  rustfs       ████████████████████████████████████████ 6.12 MB/s
  liteio       ████████████████████████████████████ 5.61 MB/s
  seaweedfs    █████████████████████████████████ 5.13 MB/s
  liteio_mem   ██████████████████████████ 4.12 MB/s
  minio        ████████████████████████ 3.79 MB/s
  localstack   █████████ 1.44 MB/s
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 164.84 MB/s | 91.0ms | 91.1ms | 91.1ms | 0 |
| liteio_mem | 160.37 MB/s | 93.6ms | 93.8ms | 93.8ms | 0 |
| minio | 157.91 MB/s | 92.4ms | 98.3ms | 98.3ms | 0 |
| seaweedfs | 126.34 MB/s | 116.1ms | 122.0ms | 122.0ms | 0 |
| localstack | 124.90 MB/s | 120.5ms | 121.5ms | 121.5ms | 0 |
| rustfs | 70.27 MB/s | 207.6ms | 220.8ms | 220.8ms | 0 |

```
  liteio       ████████████████████████████████████████ 164.84 MB/s
  liteio_mem   ██████████████████████████████████████ 160.37 MB/s
  minio        ██████████████████████████████████████ 157.91 MB/s
  seaweedfs    ██████████████████████████████ 126.34 MB/s
  localstack   ██████████████████████████████ 124.90 MB/s
  rustfs       █████████████████ 70.27 MB/s
```

### ParallelRead/1KB/C1

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 3.96 MB/s | 237.9us | 375.6us | 208.5us | 375.7us | 375.7us | 0 |
| liteio_mem | 3.37 MB/s | 288.6us | 430.5us | 202.6us | 430.6us | 430.6us | 0 |
| minio | 2.65 MB/s | 368.4us | 500.0us | 357.0us | 500.1us | 500.1us | 0 |
| seaweedfs | 2.16 MB/s | 451.4us | 486.1us | 432.2us | 486.4us | 486.4us | 0 |
| rustfs | 1.86 MB/s | 525.1us | 749.7us | 481.2us | 749.9us | 749.9us | 0 |
| localstack | 1.25 MB/s | 778.9us | 1.0ms | 715.9us | 1.0ms | 1.0ms | 0 |

```
  liteio       ████████████████████████████████████████ 3.96 MB/s
  liteio_mem   ██████████████████████████████████ 3.37 MB/s
  minio        ██████████████████████████ 2.65 MB/s
  seaweedfs    █████████████████████ 2.16 MB/s
  rustfs       ██████████████████ 1.86 MB/s
  localstack   ████████████ 1.25 MB/s
```

### ParallelRead/1KB/C10

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 0.84 MB/s | 1.2ms | 1.8ms | 1.1ms | 1.8ms | 1.8ms | 0 |
| liteio_mem | 0.77 MB/s | 1.3ms | 1.8ms | 1.3ms | 1.8ms | 1.8ms | 0 |
| rustfs | 0.76 MB/s | 1.3ms | 1.7ms | 1.2ms | 1.7ms | 1.7ms | 0 |
| seaweedfs | 0.69 MB/s | 1.4ms | 2.5ms | 1.0ms | 2.5ms | 2.5ms | 0 |
| liteio | 0.64 MB/s | 1.5ms | 2.6ms | 1.0ms | 2.6ms | 2.6ms | 0 |
| localstack | 0.17 MB/s | 5.7ms | 6.9ms | 5.9ms | 6.9ms | 6.9ms | 0 |

```
  minio        ████████████████████████████████████████ 0.84 MB/s
  liteio_mem   ████████████████████████████████████ 0.77 MB/s
  rustfs       ████████████████████████████████████ 0.76 MB/s
  seaweedfs    ████████████████████████████████ 0.69 MB/s
  liteio       ██████████████████████████████ 0.64 MB/s
  localstack   ████████ 0.17 MB/s
```

### ParallelRead/1KB/C50

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio_mem | 0.69 MB/s | 1.4ms | 1.8ms | 1.4ms | 1.8ms | 1.8ms | 0 |
| minio | 0.62 MB/s | 1.6ms | 2.1ms | 1.5ms | 2.1ms | 2.1ms | 0 |
| liteio | 0.55 MB/s | 1.8ms | 2.9ms | 1.7ms | 2.9ms | 2.9ms | 0 |
| seaweedfs | 0.46 MB/s | 2.1ms | 2.7ms | 2.1ms | 2.7ms | 2.7ms | 0 |
| localstack | 0.08 MB/s | 12.3ms | 13.7ms | 12.5ms | 13.7ms | 13.7ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 0.69 MB/s
  minio        ███████████████████████████████████ 0.62 MB/s
  liteio       ███████████████████████████████ 0.55 MB/s
  seaweedfs    ██████████████████████████ 0.46 MB/s
  localstack   ████ 0.08 MB/s
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 1.35 MB/s | 702.0us | 900.2us | 900.2us | 0 |
| liteio | 1.28 MB/s | 702.7us | 1.0ms | 1.0ms | 0 |
| rustfs | 1.23 MB/s | 758.0us | 936.2us | 936.2us | 0 |
| liteio_mem | 1.19 MB/s | 802.0us | 1.0ms | 1.0ms | 0 |
| localstack | 1.19 MB/s | 757.2us | 1.1ms | 1.1ms | 0 |
| minio | 1.04 MB/s | 901.1us | 1.0ms | 1.0ms | 0 |

```
  seaweedfs    ████████████████████████████████████████ 1.35 MB/s
  liteio       ██████████████████████████████████████ 1.28 MB/s
  rustfs       ████████████████████████████████████ 1.23 MB/s
  liteio_mem   ███████████████████████████████████ 1.19 MB/s
  localstack   ███████████████████████████████████ 1.19 MB/s
  minio        ██████████████████████████████ 1.04 MB/s
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 0.40 MB/s | 2.1ms | 3.7ms | 3.7ms | 0 |
| rustfs | 0.35 MB/s | 2.4ms | 4.8ms | 4.8ms | 0 |
| minio | 0.28 MB/s | 3.3ms | 4.6ms | 4.6ms | 0 |
| liteio_mem | 0.26 MB/s | 3.1ms | 6.9ms | 6.9ms | 0 |
| liteio | 0.25 MB/s | 3.3ms | 6.4ms | 6.4ms | 0 |
| localstack | 0.16 MB/s | 5.4ms | 10.1ms | 10.1ms | 0 |

```
  seaweedfs    ████████████████████████████████████████ 0.40 MB/s
  rustfs       ███████████████████████████████████ 0.35 MB/s
  minio        ███████████████████████████ 0.28 MB/s
  liteio_mem   █████████████████████████ 0.26 MB/s
  liteio       ████████████████████████ 0.25 MB/s
  localstack   ████████████████ 0.16 MB/s
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| seaweedfs | 0.25 MB/s | 3.8ms | 4.5ms | 4.5ms | 0 |
| liteio | 0.23 MB/s | 4.0ms | 5.0ms | 5.0ms | 0 |
| liteio_mem | 0.21 MB/s | 4.6ms | 5.8ms | 5.8ms | 0 |
| minio | 0.16 MB/s | 5.8ms | 7.7ms | 7.7ms | 0 |
| localstack | 0.10 MB/s | 9.0ms | 13.3ms | 13.3ms | 0 |

```
  seaweedfs    ████████████████████████████████████████ 0.25 MB/s
  liteio       █████████████████████████████████████ 0.23 MB/s
  liteio_mem   █████████████████████████████████ 0.21 MB/s
  minio        █████████████████████████ 0.16 MB/s
  localstack   ████████████████ 0.10 MB/s
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 238.94 MB/s | 1.0ms | 1.2ms | 1.2ms | 0 |
| liteio_mem | 229.52 MB/s | 1.1ms | 1.3ms | 1.3ms | 0 |
| seaweedfs | 194.59 MB/s | 1.3ms | 1.5ms | 1.5ms | 0 |
| minio | 168.47 MB/s | 1.4ms | 2.3ms | 2.3ms | 0 |
| localstack | 141.52 MB/s | 1.7ms | 2.1ms | 2.1ms | 0 |
| rustfs | 110.41 MB/s | 2.2ms | 2.7ms | 2.7ms | 0 |

```
  liteio       ████████████████████████████████████████ 238.94 MB/s
  liteio_mem   ██████████████████████████████████████ 229.52 MB/s
  seaweedfs    ████████████████████████████████ 194.59 MB/s
  minio        ████████████████████████████ 168.47 MB/s
  localstack   ███████████████████████ 141.52 MB/s
  rustfs       ██████████████████ 110.41 MB/s
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio_mem | 227.51 MB/s | 1.1ms | 1.3ms | 1.3ms | 0 |
| seaweedfs | 193.09 MB/s | 1.3ms | 1.4ms | 1.4ms | 0 |
| liteio | 169.40 MB/s | 1.1ms | 2.4ms | 2.4ms | 0 |
| minio | 127.50 MB/s | 1.8ms | 2.8ms | 2.8ms | 0 |
| rustfs | 112.50 MB/s | 2.2ms | 2.6ms | 2.6ms | 0 |
| localstack | 98.25 MB/s | 2.6ms | 3.6ms | 3.6ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 227.51 MB/s
  seaweedfs    █████████████████████████████████ 193.09 MB/s
  liteio       █████████████████████████████ 169.40 MB/s
  minio        ██████████████████████ 127.50 MB/s
  rustfs       ███████████████████ 112.50 MB/s
  localstack   █████████████████ 98.25 MB/s
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 232.89 MB/s | 1.0ms | 1.3ms | 1.3ms | 0 |
| liteio_mem | 229.37 MB/s | 1.1ms | 1.3ms | 1.3ms | 0 |
| seaweedfs | 184.83 MB/s | 1.3ms | 1.7ms | 1.7ms | 0 |
| minio | 116.29 MB/s | 2.0ms | 3.1ms | 3.1ms | 0 |
| localstack | 104.45 MB/s | 2.3ms | 3.4ms | 3.4ms | 0 |
| rustfs | 102.94 MB/s | 2.3ms | 3.0ms | 3.0ms | 0 |

```
  liteio       ████████████████████████████████████████ 232.89 MB/s
  liteio_mem   ███████████████████████████████████████ 229.37 MB/s
  seaweedfs    ███████████████████████████████ 184.83 MB/s
  minio        ███████████████████ 116.29 MB/s
  localstack   █████████████████ 104.45 MB/s
  rustfs       █████████████████ 102.94 MB/s
```

### Read/10MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio_mem | 313.52 MB/s | 432.8us | 435.2us | 32.0ms | 33.2ms | 33.2ms | 0 |
| liteio | 310.44 MB/s | 616.1us | 720.6us | 31.1ms | 35.1ms | 35.1ms | 0 |
| localstack | 305.86 MB/s | 1.4ms | 1.4ms | 32.6ms | 33.7ms | 33.7ms | 0 |
| minio | 304.29 MB/s | 1.3ms | 1.6ms | 32.4ms | 33.9ms | 33.9ms | 0 |
| seaweedfs | 273.14 MB/s | 2.4ms | 2.5ms | 35.0ms | 41.6ms | 41.6ms | 0 |
| rustfs | 262.44 MB/s | 7.3ms | 9.0ms | 37.7ms | 39.6ms | 39.6ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 313.52 MB/s
  liteio       ███████████████████████████████████████ 310.44 MB/s
  localstack   ███████████████████████████████████████ 305.86 MB/s
  minio        ██████████████████████████████████████ 304.29 MB/s
  seaweedfs    ██████████████████████████████████ 273.14 MB/s
  rustfs       █████████████████████████████████ 262.44 MB/s
```

### Read/1KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 4.72 MB/s | 207.0us | 224.7us | 207.3us | 224.8us | 224.8us | 0 |
| liteio_mem | 4.63 MB/s | 207.8us | 250.0us | 207.5us | 250.1us | 250.1us | 0 |
| minio | 2.32 MB/s | 420.1us | 506.7us | 406.1us | 506.7us | 506.7us | 0 |
| seaweedfs | 2.26 MB/s | 432.4us | 460.7us | 422.8us | 460.8us | 460.8us | 0 |
| rustfs | 2.13 MB/s | 457.7us | 483.9us | 453.9us | 484.0us | 484.0us | 0 |
| localstack | 1.38 MB/s | 709.8us | 777.8us | 708.6us | 777.9us | 777.9us | 0 |

```
  liteio       ████████████████████████████████████████ 4.72 MB/s
  liteio_mem   ███████████████████████████████████████ 4.63 MB/s
  minio        ███████████████████ 2.32 MB/s
  seaweedfs    ███████████████████ 2.26 MB/s
  rustfs       ██████████████████ 2.13 MB/s
  localstack   ███████████ 1.38 MB/s
```

### Read/1MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio | 274.67 MB/s | 346.6us | 440.1us | 3.7ms | 4.0ms | 4.0ms | 0 |
| liteio_mem | 248.77 MB/s | 429.1us | 724.0us | 3.5ms | 5.8ms | 5.8ms | 0 |
| seaweedfs | 248.06 MB/s | 965.6us | 1.4ms | 4.0ms | 4.4ms | 4.4ms | 0 |
| localstack | 239.18 MB/s | 1.0ms | 1.2ms | 4.1ms | 4.5ms | 4.5ms | 0 |
| minio | 234.76 MB/s | 1.2ms | 1.4ms | 4.2ms | 4.4ms | 4.4ms | 0 |
| rustfs | 193.27 MB/s | 2.2ms | 3.1ms | 5.1ms | 6.3ms | 6.3ms | 0 |

```
  liteio       ████████████████████████████████████████ 274.67 MB/s
  liteio_mem   ████████████████████████████████████ 248.77 MB/s
  seaweedfs    ████████████████████████████████████ 248.06 MB/s
  localstack   ██████████████████████████████████ 239.18 MB/s
  minio        ██████████████████████████████████ 234.76 MB/s
  rustfs       ████████████████████████████ 193.27 MB/s
```

### Read/64KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| liteio_mem | 148.36 MB/s | 241.7us | 402.3us | 398.3us | 541.8us | 547.0us | 0 |
| liteio | 140.78 MB/s | 244.0us | 349.4us | 438.0us | 543.9us | 563.0us | 0 |
| minio | 97.59 MB/s | 448.5us | 533.2us | 635.2us | 725.0us | 788.7us | 0 |
| seaweedfs | 92.63 MB/s | 483.8us | 559.0us | 665.8us | 765.4us | 781.9us | 0 |
| rustfs | 82.92 MB/s | 637.8us | 761.8us | 725.7us | 880.0us | 899.4us | 0 |
| localstack | 63.48 MB/s | 870.4us | 1.0ms | 920.4us | 1.2ms | 1.3ms | 0 |

```
  liteio_mem   ████████████████████████████████████████ 148.36 MB/s
  liteio       █████████████████████████████████████ 140.78 MB/s
  minio        ██████████████████████████ 97.59 MB/s
  seaweedfs    ████████████████████████ 92.63 MB/s
  rustfs       ██████████████████████ 82.92 MB/s
  localstack   █████████████████ 63.48 MB/s
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 6879 ops/s | 140.5us | 163.5us | 163.5us | 0 |
| liteio_mem | 5899 ops/s | 164.3us | 196.3us | 196.3us | 0 |
| seaweedfs | 3489 ops/s | 278.2us | 339.5us | 339.5us | 0 |
| minio | 3267 ops/s | 285.0us | 451.5us | 451.5us | 0 |
| rustfs | 3217 ops/s | 306.6us | 343.4us | 343.4us | 0 |
| localstack | 1458 ops/s | 662.3us | 805.8us | 805.8us | 0 |

```
  liteio       ████████████████████████████████████████ 6879 ops/s
  liteio_mem   ██████████████████████████████████ 5899 ops/s
  seaweedfs    ████████████████████ 3489 ops/s
  minio        ██████████████████ 3267 ops/s
  rustfs       ██████████████████ 3217 ops/s
  localstack   ████████ 1458 ops/s
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 191.36 MB/s | 51.8ms | 55.0ms | 55.0ms | 0 |
| liteio_mem | 180.52 MB/s | 50.9ms | 70.4ms | 70.4ms | 0 |
| minio | 172.57 MB/s | 58.2ms | 63.4ms | 63.4ms | 0 |
| seaweedfs | 152.96 MB/s | 64.6ms | 69.2ms | 69.2ms | 0 |
| localstack | 140.25 MB/s | 69.8ms | 80.5ms | 80.5ms | 0 |
| rustfs | 81.63 MB/s | 132.0ms | 136.0ms | 136.0ms | 0 |

```
  liteio       ████████████████████████████████████████ 191.36 MB/s
  liteio_mem   █████████████████████████████████████ 180.52 MB/s
  minio        ████████████████████████████████████ 172.57 MB/s
  seaweedfs    ███████████████████████████████ 152.96 MB/s
  localstack   █████████████████████████████ 140.25 MB/s
  rustfs       █████████████████ 81.63 MB/s
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 1.20 MB/s | 763.3us | 993.5us | 993.5us | 0 |
| seaweedfs | 1.16 MB/s | 809.6us | 973.6us | 973.6us | 0 |
| localstack | 1.03 MB/s | 921.0us | 1.3ms | 1.3ms | 0 |
| liteio_mem | 0.90 MB/s | 1.0ms | 1.6ms | 1.6ms | 0 |
| liteio | 0.83 MB/s | 1.0ms | 1.7ms | 1.7ms | 0 |
| minio | 0.76 MB/s | 1.2ms | 1.5ms | 1.5ms | 0 |

```
  rustfs       ████████████████████████████████████████ 1.20 MB/s
  seaweedfs    ██████████████████████████████████████ 1.16 MB/s
  localstack   ██████████████████████████████████ 1.03 MB/s
  liteio_mem   █████████████████████████████ 0.90 MB/s
  liteio       ███████████████████████████ 0.83 MB/s
  minio        █████████████████████████ 0.76 MB/s
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| rustfs | 177.52 MB/s | 5.4ms | 6.2ms | 6.2ms | 0 |
| liteio_mem | 161.75 MB/s | 6.1ms | 6.8ms | 6.8ms | 0 |
| liteio | 161.18 MB/s | 5.9ms | 6.8ms | 6.8ms | 0 |
| minio | 136.15 MB/s | 7.2ms | 7.8ms | 7.8ms | 0 |
| localstack | 124.75 MB/s | 7.7ms | 8.8ms | 8.8ms | 0 |
| seaweedfs | 122.10 MB/s | 7.8ms | 10.0ms | 10.0ms | 0 |

```
  rustfs       ████████████████████████████████████████ 177.52 MB/s
  liteio_mem   ████████████████████████████████████ 161.75 MB/s
  liteio       ████████████████████████████████████ 161.18 MB/s
  minio        ██████████████████████████████ 136.15 MB/s
  localstack   ████████████████████████████ 124.75 MB/s
  seaweedfs    ███████████████████████████ 122.10 MB/s
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| liteio | 62.18 MB/s | 965.0us | 1.2ms | 1.5ms | 0 |
| rustfs | 59.00 MB/s | 1.0ms | 1.2ms | 1.3ms | 0 |
| liteio_mem | 56.36 MB/s | 1.0ms | 1.4ms | 1.7ms | 0 |
| seaweedfs | 53.74 MB/s | 1.1ms | 1.3ms | 1.7ms | 0 |
| localstack | 51.59 MB/s | 1.2ms | 1.4ms | 1.7ms | 0 |
| minio | 43.85 MB/s | 1.4ms | 1.8ms | 2.0ms | 0 |

```
  liteio       ████████████████████████████████████████ 62.18 MB/s
  rustfs       █████████████████████████████████████ 59.00 MB/s
  liteio_mem   ████████████████████████████████████ 56.36 MB/s
  seaweedfs    ██████████████████████████████████ 53.74 MB/s
  localstack   █████████████████████████████████ 51.59 MB/s
  minio        ████████████████████████████ 43.85 MB/s
```

## Resource Usage

| Driver | Memory | RSS | Cache | CPU | Volume | Block I/O |
|--------|--------|-----|-------|-----|--------|----------|
| liteio | 20.47MiB / 7.653GiB | 20.5 MB | - | 0.0% | (no data) | 4.78MB / 1.37GB |
| liteio_mem | 53.22MiB / 7.653GiB | 53.2 MB | - | 0.8% | 911.6 MB | 23.3MB / 7.67GB |
| localstack | 502.9MiB / 7.653GiB | 502.9 MB | - | 0.1% | 0.0 MB | 24.4MB / 5.48GB |
| minio | 546.3MiB / 7.653GiB | 546.3 MB | - | 0.1% | 3557.4 MB | 198MB / 9.1GB |
| rustfs | 407.3MiB / 7.653GiB | 407.3 MB | - | 0.1% | 364.2 MB | 52MB / 4.9GB |
| seaweedfs | 64.71MiB / 7.653GiB | 64.7 MB | - | 0.0% | (no data) | 9.28MB / 1.44MB |

### Memory Analysis Note

> **RSS (Resident Set Size)**: Actual application memory usage.
> 
> **Cache**: Linux page cache from filesystem I/O. Disk-based drivers show higher total memory because the OS caches file pages in RAM. This memory is reclaimable and doesn't indicate a memory leak.
> 
> Memory-based drivers (like `liteio_mem`) have minimal cache because data stays in application memory (RSS), not filesystem cache.

## Recommendations

- **Best for write-heavy workloads:** liteio
- **Best for read-heavy workloads:** liteio_mem

---

*Report generated by storage benchmark CLI*
