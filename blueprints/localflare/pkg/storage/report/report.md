# Storage Benchmark Report

**Generated:** 2026-01-21T18:19:38+07:00

**Go Version:** go1.25.5

**Platform:** darwin/arm64

## Executive Summary

### Summary

**Overall Winner:** devnull_s3 (won 30/48 benchmarks, 62%)

| Rank | Driver | Wins | Win Rate |
|------|--------|------|----------|
| 1 | devnull_s3 | 30 | 62% |
| 2 | usagi_s3 | 13 | 27% |
| 3 | minio | 5 | 10% |

### Performance Leaders

| Operation | Leader | Performance | Margin |
|-----------|--------|-------------|--------|
| Small Read (1KB) | usagi_s3 | 3.9 MB/s | +16% vs devnull_s3 |
| Small Write (1KB) | usagi_s3 | 1.6 MB/s | close |
| Large Read (100MB) | minio | 176.7 MB/s | close |
| Large Write (100MB) | minio | 125.2 MB/s | close |
| Delete | devnull_s3 | 3.4K ops/s | close |
| Stat | devnull_s3 | 4.0K ops/s | +14% vs usagi_s3 |
| List (100 objects) | usagi_s3 | 850 ops/s | close |
| Copy | devnull_s3 | 1.1 MB/s | +44% vs usagi_s3 |

### Best Driver by Use Case

| Use Case | Recommended | Performance | Notes |
|----------|-------------|-------------|-------|
| Large File Uploads (100MB+) | **minio** | 125 MB/s | Best for media, backups |
| Large File Downloads (100MB) | **minio** | 177 MB/s | Best for streaming, CDN |
| Small File Operations | **usagi_s3** | 2851 ops/s | Best for metadata, configs |
| High Concurrency (C10) | **devnull_s3** | - | Best for multi-user apps |
| Memory Constrained | **minio** | 877 MB RAM | Best for edge/embedded |

### Large File Performance (100MB)

| Driver | Write (MB/s) | Read (MB/s) | Write Latency | Read Latency |
|--------|-------------|-------------|---------------|---------------|
| devnull_s3 | 119.3 | 155.1 | 839.2ms | 651.3ms |
| minio | 125.2 | 176.7 | 799.3ms | 552.2ms |
| usagi_s3 | 123.4 | 160.8 | 800.8ms | 616.2ms |

### Small File Performance (1KB)

| Driver | Write (ops/s) | Read (ops/s) | Write Latency | Read Latency |
|--------|--------------|--------------|---------------|---------------|
| devnull_s3 | 1665 | 3459 | 529.3us | 253.8us |
| minio | 1301 | 2550 | 735.1us | 370.4us |
| usagi_s3 | 1678 | 4024 | 540.2us | 232.2us |

### Metadata Operations (ops/s)

| Driver | Stat | List (100 objects) | Delete |
|--------|------|-------------------|--------|
| devnull_s3 | 4010 | 844 | 3425 |
| minio | 2801 | 446 | 1326 |
| usagi_s3 | 3527 | 850 | 3372 |

### Concurrency Performance

**Parallel Write (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| devnull_s3 | 1.27 | 0.38 | 0.11 | 0.05 | 0.03 | 0.02 |
| minio | 0.51 | 0.11 | 0.05 | 0.04 | 0.02 | 0.01 |
| usagi_s3 | 1.17 | 0.28 | 0.09 | 0.06 | 0.05 | 0.02 |

*\* indicates errors occurred*

**Parallel Read (MB/s by concurrency)**

| Driver | C1 | C10 | C25 | C50 | C100 | C200 |
|--------|------|------|------|------|------|------|
| devnull_s3 | 3.15 | 1.02 | 0.53 | 0.32 | 0.18 | 0.08 |
| minio | 1.05 | 0.42 | 0.32 | 0.17 | 0.09 | 0.05 |
| usagi_s3 | 2.95 | 0.95 | 0.51 | 0.29 | 0.16 | 0.08 |

*\* indicates errors occurred*

### Scale Performance

Performance with varying numbers of objects (256B each).

**Write N Files (total time)**

| Driver | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|
| devnull_s3 | 6.8ms | 69.6ms | 720.0ms | 6.34s |
| minio | 16.6ms | 150.1ms | 1.45s | 13.09s |
| usagi_s3 | 9.7ms | 81.0ms | 700.8ms | 6.49s |

*\* indicates errors occurred*

**List N Files (total time)**

| Driver | 10 | 100 | 1000 | 10000 |
|--------|------|------|------|------|
| devnull_s3 | 684.0us | 1.5ms | 6.8ms | 264.3ms |
| minio | 1.8ms | 3.5ms | 17.3ms | 823.0ms |
| usagi_s3 | 550.8us | 1.1ms | 7.1ms | 289.6ms |

*\* indicates errors occurred*

### Resource Usage Summary

| Driver | Memory | CPU |
|--------|--------|-----|
| devnull_s3 | 1578.0 MB | 3.8% |
| minio | 876.8 MB | 4.8% |
| usagi_s3 | 1768.4 MB | 3.8% |

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

- **devnull_s3** (48 benchmarks)
- **minio** (48 benchmarks)
- **usagi_s3** (48 benchmarks)

## Detailed Results

### Copy/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 1.09 MB/s | 779.5us | 1.6ms | 2.2ms | 0 |
| usagi_s3 | 0.75 MB/s | 954.8us | 3.0ms | 5.4ms | 0 |
| minio | 0.72 MB/s | 1.2ms | 2.4ms | 3.3ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 1.09 MB/s
usagi_s3     ████████████████████ 0.75 MB/s
minio        ███████████████████ 0.72 MB/s
```

**Latency (P50)**
```
devnull_s3   ███████████████████ 779.5us
usagi_s3     ████████████████████████ 954.8us
minio        ██████████████████████████████ 1.2ms
```

### Delete

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 3425 ops/s | 256.4us | 498.6us | 805.7us | 0 |
| usagi_s3 | 3372 ops/s | 266.9us | 465.1us | 762.9us | 0 |
| minio | 1326 ops/s | 606.3us | 1.7ms | 2.8ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 3425 ops/s
usagi_s3     █████████████████████████████ 3372 ops/s
minio        ███████████ 1326 ops/s
```

**Latency (P50)**
```
devnull_s3   ████████████ 256.4us
usagi_s3     █████████████ 266.9us
minio        ██████████████████████████████ 606.3us
```

### EdgeCase/DeepNested

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 0.14 MB/s | 631.3us | 1.0ms | 1.4ms | 0 |
| usagi_s3 | 0.11 MB/s | 754.8us | 1.3ms | 1.9ms | 0 |
| minio | 0.04 MB/s | 1.9ms | 3.9ms | 6.1ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 0.14 MB/s
usagi_s3     ████████████████████████ 0.11 MB/s
minio        █████████ 0.04 MB/s
```

**Latency (P50)**
```
devnull_s3   █████████ 631.3us
usagi_s3     ███████████ 754.8us
minio        ██████████████████████████████ 1.9ms
```

### EdgeCase/EmptyObject

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 1436 ops/s | 629.8us | 992.5us | 1.6ms | 0 |
| usagi_s3 | 1176 ops/s | 751.4us | 1.5ms | 2.1ms | 0 |
| minio | 730 ops/s | 1.2ms | 2.3ms | 3.7ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 1436 ops/s
usagi_s3     ████████████████████████ 1176 ops/s
minio        ███████████████ 730 ops/s
```

**Latency (P50)**
```
devnull_s3   ███████████████ 629.8us
usagi_s3     ██████████████████ 751.4us
minio        ██████████████████████████████ 1.2ms
```

### EdgeCase/LongKey256

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 0.15 MB/s | 601.8us | 836.8us | 1.3ms | 0 |
| usagi_s3 | 0.11 MB/s | 765.1us | 1.3ms | 1.6ms | 0 |
| minio | 0.06 MB/s | 1.2ms | 3.0ms | 5.0ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 0.15 MB/s
usagi_s3     ██████████████████████ 0.11 MB/s
minio        ████████████ 0.06 MB/s
```

**Latency (P50)**
```
devnull_s3   ██████████████ 601.8us
usagi_s3     ██████████████████ 765.1us
minio        ██████████████████████████████ 1.2ms
```

### List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 850 ops/s | 1.1ms | 1.6ms | 2.0ms | 0 |
| devnull_s3 | 844 ops/s | 1.1ms | 1.6ms | 2.0ms | 0 |
| minio | 446 ops/s | 2.1ms | 3.0ms | 4.9ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 850 ops/s
devnull_s3   █████████████████████████████ 844 ops/s
minio        ███████████████ 446 ops/s
```

**Latency (P50)**
```
usagi_s3     ███████████████ 1.1ms
devnull_s3   ███████████████ 1.1ms
minio        ██████████████████████████████ 2.1ms
```

### MixedWorkload/Balanced_50_50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.35 MB/s | 13.3ms | 119.8ms | 129.1ms | 0 |
| devnull_s3 | 0.34 MB/s | 9.9ms | 121.0ms | 132.4ms | 0 |
| minio | 0.19 MB/s | 46.3ms | 262.7ms | 425.1ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.35 MB/s
devnull_s3   ████████████████████████████ 0.34 MB/s
minio        ███████████████ 0.19 MB/s
```

**Latency (P50)**
```
usagi_s3     ████████ 13.3ms
devnull_s3   ██████ 9.9ms
minio        ██████████████████████████████ 46.3ms
```

### MixedWorkload/ReadHeavy_90_10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 0.59 MB/s | 22.9ms | 60.5ms | 85.9ms | 0 |
| usagi_s3 | 0.55 MB/s | 24.5ms | 63.9ms | 94.2ms | 0 |
| minio | 0.41 MB/s | 27.6ms | 126.6ms | 189.7ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 0.59 MB/s
usagi_s3     ███████████████████████████ 0.55 MB/s
minio        ████████████████████ 0.41 MB/s
```

**Latency (P50)**
```
devnull_s3   ████████████████████████ 22.9ms
usagi_s3     ██████████████████████████ 24.5ms
minio        ██████████████████████████████ 27.6ms
```

### MixedWorkload/WriteHeavy_10_90

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.25 MB/s | 77.7ms | 96.1ms | 102.9ms | 0 |
| devnull_s3 | 0.19 MB/s | 98.8ms | 148.5ms | 172.4ms | 0 |
| minio | 0.10 MB/s | 119.8ms | 392.8ms | 885.1ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.25 MB/s
devnull_s3   ██████████████████████ 0.19 MB/s
minio        ████████████ 0.10 MB/s
```

**Latency (P50)**
```
usagi_s3     ███████████████████ 77.7ms
devnull_s3   ████████████████████████ 98.8ms
minio        ██████████████████████████████ 119.8ms
```

### Multipart/15MB_3Parts

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 106.62 MB/s | 136.9ms | 149.3ms | 149.3ms | 0 |
| usagi_s3 | 99.65 MB/s | 148.0ms | 161.2ms | 161.2ms | 0 |
| minio | 58.66 MB/s | 223.9ms | 290.3ms | 290.3ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 106.62 MB/s
usagi_s3     ████████████████████████████ 99.65 MB/s
minio        ████████████████ 58.66 MB/s
```

**Latency (P50)**
```
devnull_s3   ██████████████████ 136.9ms
usagi_s3     ███████████████████ 148.0ms
minio        ██████████████████████████████ 223.9ms
```

### ParallelRead/1KB/C1

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| devnull_s3 | 3.15 MB/s | 309.7us | 468.4us | 285.1us | 468.5us | 695.5us | 0 |
| usagi_s3 | 2.95 MB/s | 330.5us | 558.2us | 293.5us | 558.2us | 757.6us | 0 |
| minio | 1.05 MB/s | 933.2us | 1.8ms | 776.2us | 1.8ms | 2.5ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 3.15 MB/s
usagi_s3     ████████████████████████████ 2.95 MB/s
minio        █████████ 1.05 MB/s
```

**Latency (P50)**
```
devnull_s3   ███████████ 285.1us
usagi_s3     ███████████ 293.5us
minio        ██████████████████████████████ 776.2us
```

### ParallelRead/1KB/C10

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| devnull_s3 | 1.02 MB/s | 957.0us | 1.5ms | 905.6us | 1.5ms | 1.9ms | 0 |
| usagi_s3 | 0.95 MB/s | 1.0ms | 1.7ms | 937.0us | 1.7ms | 2.4ms | 0 |
| minio | 0.42 MB/s | 2.3ms | 5.1ms | 1.8ms | 5.1ms | 10.7ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 1.02 MB/s
usagi_s3     ███████████████████████████ 0.95 MB/s
minio        ████████████ 0.42 MB/s
```

**Latency (P50)**
```
devnull_s3   ███████████████ 905.6us
usagi_s3     ███████████████ 937.0us
minio        ██████████████████████████████ 1.8ms
```

### ParallelRead/1KB/C100

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| devnull_s3 | 0.18 MB/s | 5.4ms | 10.2ms | 5.1ms | 10.2ms | 13.6ms | 0 |
| usagi_s3 | 0.16 MB/s | 6.0ms | 10.1ms | 5.8ms | 10.1ms | 13.4ms | 0 |
| minio | 0.09 MB/s | 10.5ms | 21.9ms | 9.5ms | 21.9ms | 30.9ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 0.18 MB/s
usagi_s3     ███████████████████████████ 0.16 MB/s
minio        ███████████████ 0.09 MB/s
```

**Latency (P50)**
```
devnull_s3   ████████████████ 5.1ms
usagi_s3     ██████████████████ 5.8ms
minio        ██████████████████████████████ 9.5ms
```

### ParallelRead/1KB/C200

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| devnull_s3 | 0.08 MB/s | 11.5ms | 18.0ms | 11.7ms | 18.0ms | 23.0ms | 0 |
| usagi_s3 | 0.08 MB/s | 12.5ms | 23.7ms | 11.8ms | 23.7ms | 39.1ms | 0 |
| minio | 0.05 MB/s | 20.0ms | 41.7ms | 18.1ms | 41.9ms | 57.7ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 0.08 MB/s
usagi_s3     ███████████████████████████ 0.08 MB/s
minio        █████████████████ 0.05 MB/s
```

**Latency (P50)**
```
devnull_s3   ███████████████████ 11.7ms
usagi_s3     ███████████████████ 11.8ms
minio        ██████████████████████████████ 18.1ms
```

### ParallelRead/1KB/C25

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| devnull_s3 | 0.53 MB/s | 1.8ms | 3.1ms | 1.7ms | 3.1ms | 4.2ms | 0 |
| usagi_s3 | 0.51 MB/s | 1.9ms | 3.2ms | 1.8ms | 3.2ms | 4.3ms | 0 |
| minio | 0.32 MB/s | 3.0ms | 5.2ms | 2.8ms | 5.2ms | 7.5ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 0.53 MB/s
usagi_s3     ████████████████████████████ 0.51 MB/s
minio        ██████████████████ 0.32 MB/s
```

**Latency (P50)**
```
devnull_s3   ██████████████████ 1.7ms
usagi_s3     ███████████████████ 1.8ms
minio        ██████████████████████████████ 2.8ms
```

### ParallelRead/1KB/C50

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| devnull_s3 | 0.32 MB/s | 3.0ms | 5.1ms | 2.9ms | 5.1ms | 6.5ms | 0 |
| usagi_s3 | 0.29 MB/s | 3.3ms | 6.1ms | 3.0ms | 6.1ms | 8.8ms | 0 |
| minio | 0.17 MB/s | 5.7ms | 11.3ms | 5.0ms | 11.3ms | 17.4ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 0.32 MB/s
usagi_s3     ███████████████████████████ 0.29 MB/s
minio        ████████████████ 0.17 MB/s
```

**Latency (P50)**
```
devnull_s3   █████████████████ 2.9ms
usagi_s3     █████████████████ 3.0ms
minio        ██████████████████████████████ 5.0ms
```

### ParallelWrite/1KB/C1

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 1.27 MB/s | 689.3us | 1.2ms | 1.9ms | 0 |
| usagi_s3 | 1.17 MB/s | 783.5us | 1.1ms | 1.6ms | 0 |
| minio | 0.51 MB/s | 1.6ms | 3.3ms | 4.4ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 1.27 MB/s
usagi_s3     ███████████████████████████ 1.17 MB/s
minio        ████████████ 0.51 MB/s
```

**Latency (P50)**
```
devnull_s3   ████████████ 689.3us
usagi_s3     ██████████████ 783.5us
minio        ██████████████████████████████ 1.6ms
```

### ParallelWrite/1KB/C10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 0.38 MB/s | 2.5ms | 3.7ms | 4.8ms | 0 |
| usagi_s3 | 0.28 MB/s | 3.0ms | 6.2ms | 9.1ms | 0 |
| minio | 0.11 MB/s | 6.7ms | 21.9ms | 32.4ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 0.38 MB/s
usagi_s3     ██████████████████████ 0.28 MB/s
minio        ████████ 0.11 MB/s
```

**Latency (P50)**
```
devnull_s3   ███████████ 2.5ms
usagi_s3     █████████████ 3.0ms
minio        ██████████████████████████████ 6.7ms
```

### ParallelWrite/1KB/C100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.05 MB/s | 18.9ms | 42.3ms | 80.3ms | 0 |
| devnull_s3 | 0.03 MB/s | 36.2ms | 46.4ms | 51.9ms | 0 |
| minio | 0.02 MB/s | 46.4ms | 85.8ms | 110.8ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.05 MB/s
devnull_s3   ████████████████████ 0.03 MB/s
minio        █████████████ 0.02 MB/s
```

**Latency (P50)**
```
usagi_s3     ████████████ 18.9ms
devnull_s3   ███████████████████████ 36.2ms
minio        ██████████████████████████████ 46.4ms
```

### ParallelWrite/1KB/C200

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.02 MB/s | 50.8ms | 98.5ms | 126.0ms | 0 |
| devnull_s3 | 0.02 MB/s | 65.1ms | 91.7ms | 133.7ms | 0 |
| minio | 0.01 MB/s | 80.9ms | 178.8ms | 262.1ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.02 MB/s
devnull_s3   █████████████████████████ 0.02 MB/s
minio        ████████████████ 0.01 MB/s
```

**Latency (P50)**
```
usagi_s3     ██████████████████ 50.8ms
devnull_s3   ████████████████████████ 65.1ms
minio        ██████████████████████████████ 80.9ms
```

### ParallelWrite/1KB/C25

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 0.11 MB/s | 7.7ms | 19.5ms | 26.7ms | 0 |
| usagi_s3 | 0.09 MB/s | 12.6ms | 17.5ms | 22.7ms | 0 |
| minio | 0.05 MB/s | 17.0ms | 40.4ms | 51.6ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 0.11 MB/s
usagi_s3     ████████████████████████ 0.09 MB/s
minio        ██████████████ 0.05 MB/s
```

**Latency (P50)**
```
devnull_s3   █████████████ 7.7ms
usagi_s3     ██████████████████████ 12.6ms
minio        ██████████████████████████████ 17.0ms
```

### ParallelWrite/1KB/C50

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.06 MB/s | 18.2ms | 29.6ms | 42.4ms | 0 |
| devnull_s3 | 0.05 MB/s | 18.7ms | 31.7ms | 41.0ms | 0 |
| minio | 0.04 MB/s | 24.9ms | 48.3ms | 67.3ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.06 MB/s
devnull_s3   ████████████████████████████ 0.05 MB/s
minio        ██████████████████ 0.04 MB/s
```

**Latency (P50)**
```
usagi_s3     █████████████████████ 18.2ms
devnull_s3   ██████████████████████ 18.7ms
minio        ██████████████████████████████ 24.9ms
```

### RangeRead/End_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 144.93 MB/s | 1.7ms | 2.1ms | 2.5ms | 0 |
| usagi_s3 | 136.16 MB/s | 1.7ms | 2.4ms | 3.2ms | 0 |
| minio | 111.91 MB/s | 2.1ms | 2.9ms | 3.6ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 144.93 MB/s
usagi_s3     ████████████████████████████ 136.16 MB/s
minio        ███████████████████████ 111.91 MB/s
```

**Latency (P50)**
```
devnull_s3   ███████████████████████ 1.7ms
usagi_s3     ████████████████████████ 1.7ms
minio        ██████████████████████████████ 2.1ms
```

### RangeRead/Middle_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 140.16 MB/s | 1.7ms | 2.3ms | 2.7ms | 0 |
| usagi_s3 | 130.01 MB/s | 1.8ms | 2.6ms | 3.5ms | 0 |
| minio | 106.19 MB/s | 2.2ms | 3.2ms | 4.4ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 140.16 MB/s
usagi_s3     ███████████████████████████ 130.01 MB/s
minio        ██████████████████████ 106.19 MB/s
```

**Latency (P50)**
```
devnull_s3   ███████████████████████ 1.7ms
usagi_s3     ████████████████████████ 1.8ms
minio        ██████████████████████████████ 2.2ms
```

### RangeRead/Start_256KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 124.98 MB/s | 1.9ms | 2.8ms | 3.6ms | 0 |
| usagi_s3 | 124.76 MB/s | 1.9ms | 2.7ms | 3.8ms | 0 |
| minio | 87.87 MB/s | 2.5ms | 4.7ms | 6.6ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 124.98 MB/s
usagi_s3     █████████████████████████████ 124.76 MB/s
minio        █████████████████████ 87.87 MB/s
```

**Latency (P50)**
```
devnull_s3   ██████████████████████ 1.9ms
usagi_s3     ██████████████████████ 1.9ms
minio        ██████████████████████████████ 2.5ms
```

### Read/100MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 176.73 MB/s | 1.4ms | 1.4ms | 552.2ms | 552.2ms | 552.2ms | 0 |
| usagi_s3 | 160.80 MB/s | 2.3ms | 2.3ms | 616.2ms | 616.2ms | 616.2ms | 0 |
| devnull_s3 | 155.06 MB/s | 2.7ms | 2.5ms | 651.3ms | 651.3ms | 651.3ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 176.73 MB/s
usagi_s3     ███████████████████████████ 160.80 MB/s
devnull_s3   ██████████████████████████ 155.06 MB/s
```

**Latency (P50)**
```
minio        █████████████████████████ 552.2ms
usagi_s3     ████████████████████████████ 616.2ms
devnull_s3   ██████████████████████████████ 651.3ms
```

### Read/10MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| minio | 175.75 MB/s | 1.5ms | 1.7ms | 55.3ms | 63.1ms | 64.5ms | 0 |
| devnull_s3 | 159.08 MB/s | 3.3ms | 5.0ms | 62.0ms | 70.1ms | 70.1ms | 0 |
| usagi_s3 | 157.96 MB/s | 2.8ms | 3.4ms | 62.7ms | 72.6ms | 73.3ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 175.75 MB/s
devnull_s3   ███████████████████████████ 159.08 MB/s
usagi_s3     ██████████████████████████ 157.96 MB/s
```

**Latency (P50)**
```
minio        ██████████████████████████ 55.3ms
devnull_s3   █████████████████████████████ 62.0ms
usagi_s3     ██████████████████████████████ 62.7ms
```

### Read/1KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| usagi_s3 | 3.93 MB/s | 248.4us | 345.8us | 232.2us | 345.8us | 485.3us | 0 |
| devnull_s3 | 3.38 MB/s | 288.9us | 503.8us | 253.8us | 504.0us | 758.4us | 0 |
| minio | 2.49 MB/s | 392.0us | 509.0us | 370.4us | 509.0us | 741.0us | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 3.93 MB/s
devnull_s3   █████████████████████████ 3.38 MB/s
minio        ███████████████████ 2.49 MB/s
```

**Latency (P50)**
```
usagi_s3     ██████████████████ 232.2us
devnull_s3   ████████████████████ 253.8us
minio        ██████████████████████████████ 370.4us
```

### Read/1MB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| devnull_s3 | 161.31 MB/s | 568.1us | 902.0us | 5.9ms | 7.9ms | 9.4ms | 0 |
| minio | 153.31 MB/s | 1.4ms | 2.0ms | 6.3ms | 7.7ms | 10.6ms | 0 |
| usagi_s3 | 146.95 MB/s | 663.1us | 1.1ms | 6.4ms | 9.3ms | 10.5ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 161.31 MB/s
minio        ████████████████████████████ 153.31 MB/s
usagi_s3     ███████████████████████████ 146.95 MB/s
```

**Latency (P50)**
```
devnull_s3   ███████████████████████████ 5.9ms
minio        █████████████████████████████ 6.3ms
usagi_s3     ██████████████████████████████ 6.4ms
```

### Read/64KB

| Driver | Throughput | TTFB Avg | TTFB P95 | P50 | P95 | P99 | Errors |
|--------|------------|----------|----------|-----|-----|-----|--------|
| devnull_s3 | 95.35 MB/s | 341.1us | 610.0us | 617.7us | 934.0us | 1.2ms | 0 |
| usagi_s3 | 89.67 MB/s | 367.1us | 726.3us | 645.5us | 1.1ms | 1.4ms | 0 |
| minio | 80.79 MB/s | 529.6us | 885.1us | 716.2us | 1.1ms | 1.8ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 95.35 MB/s
usagi_s3     ████████████████████████████ 89.67 MB/s
minio        █████████████████████████ 80.79 MB/s
```

**Latency (P50)**
```
devnull_s3   █████████████████████████ 617.7us
usagi_s3     ███████████████████████████ 645.5us
minio        ██████████████████████████████ 716.2us
```

### Scale/Delete/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 385 ops/s | 2.6ms | 2.6ms | 2.6ms | 0 |
| devnull_s3 | 379 ops/s | 2.6ms | 2.6ms | 2.6ms | 0 |
| minio | 160 ops/s | 6.3ms | 6.3ms | 6.3ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 385 ops/s
devnull_s3   █████████████████████████████ 379 ops/s
minio        ████████████ 160 ops/s
```

**Latency (P50)**
```
usagi_s3     ████████████ 2.6ms
devnull_s3   ████████████ 2.6ms
minio        ██████████████████████████████ 6.3ms
```

### Scale/Delete/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 37 ops/s | 26.7ms | 26.7ms | 26.7ms | 0 |
| usagi_s3 | 36 ops/s | 27.8ms | 27.8ms | 27.8ms | 0 |
| minio | 15 ops/s | 68.2ms | 68.2ms | 68.2ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 37 ops/s
usagi_s3     ████████████████████████████ 36 ops/s
minio        ███████████ 15 ops/s
```

**Latency (P50)**
```
devnull_s3   ███████████ 26.7ms
usagi_s3     ████████████ 27.8ms
minio        ██████████████████████████████ 68.2ms
```

### Scale/Delete/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 4 ops/s | 256.9ms | 256.9ms | 256.9ms | 0 |
| devnull_s3 | 4 ops/s | 273.7ms | 273.7ms | 273.7ms | 0 |
| minio | 2 ops/s | 572.6ms | 572.6ms | 572.6ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 4 ops/s
devnull_s3   ████████████████████████████ 4 ops/s
minio        █████████████ 2 ops/s
```

**Latency (P50)**
```
usagi_s3     █████████████ 256.9ms
devnull_s3   ██████████████ 273.7ms
minio        ██████████████████████████████ 572.6ms
```

### Scale/Delete/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 0 ops/s | 2.68s | 2.68s | 2.68s | 0 |
| usagi_s3 | 0 ops/s | 2.97s | 2.97s | 2.97s | 0 |
| minio | 0 ops/s | 5.18s | 5.18s | 5.18s | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 0 ops/s
usagi_s3     ███████████████████████████ 0 ops/s
minio        ███████████████ 0 ops/s
```

**Latency (P50)**
```
devnull_s3   ███████████████ 2.68s
usagi_s3     █████████████████ 2.97s
minio        ██████████████████████████████ 5.18s
```

### Scale/List/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 1816 ops/s | 550.8us | 550.8us | 550.8us | 0 |
| devnull_s3 | 1462 ops/s | 684.0us | 684.0us | 684.0us | 0 |
| minio | 555 ops/s | 1.8ms | 1.8ms | 1.8ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 1816 ops/s
devnull_s3   ████████████████████████ 1462 ops/s
minio        █████████ 555 ops/s
```

**Latency (P50)**
```
usagi_s3     █████████ 550.8us
devnull_s3   ███████████ 684.0us
minio        ██████████████████████████████ 1.8ms
```

### Scale/List/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 932 ops/s | 1.1ms | 1.1ms | 1.1ms | 0 |
| devnull_s3 | 686 ops/s | 1.5ms | 1.5ms | 1.5ms | 0 |
| minio | 283 ops/s | 3.5ms | 3.5ms | 3.5ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 932 ops/s
devnull_s3   ██████████████████████ 686 ops/s
minio        █████████ 283 ops/s
```

**Latency (P50)**
```
usagi_s3     █████████ 1.1ms
devnull_s3   ████████████ 1.5ms
minio        ██████████████████████████████ 3.5ms
```

### Scale/List/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 148 ops/s | 6.8ms | 6.8ms | 6.8ms | 0 |
| usagi_s3 | 141 ops/s | 7.1ms | 7.1ms | 7.1ms | 0 |
| minio | 58 ops/s | 17.3ms | 17.3ms | 17.3ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 148 ops/s
usagi_s3     ████████████████████████████ 141 ops/s
minio        ███████████ 58 ops/s
```

**Latency (P50)**
```
devnull_s3   ███████████ 6.8ms
usagi_s3     ████████████ 7.1ms
minio        ██████████████████████████████ 17.3ms
```

### Scale/List/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 4 ops/s | 264.3ms | 264.3ms | 264.3ms | 0 |
| usagi_s3 | 3 ops/s | 289.6ms | 289.6ms | 289.6ms | 0 |
| minio | 1 ops/s | 823.0ms | 823.0ms | 823.0ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 4 ops/s
usagi_s3     ███████████████████████████ 3 ops/s
minio        █████████ 1 ops/s
```

**Latency (P50)**
```
devnull_s3   █████████ 264.3ms
usagi_s3     ██████████ 289.6ms
minio        ██████████████████████████████ 823.0ms
```

### Scale/Write/10

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 0.36 MB/s | 6.8ms | 6.8ms | 6.8ms | 0 |
| usagi_s3 | 0.25 MB/s | 9.7ms | 9.7ms | 9.7ms | 0 |
| minio | 0.15 MB/s | 16.6ms | 16.6ms | 16.6ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 0.36 MB/s
usagi_s3     █████████████████████ 0.25 MB/s
minio        ████████████ 0.15 MB/s
```

**Latency (P50)**
```
devnull_s3   ████████████ 6.8ms
usagi_s3     █████████████████ 9.7ms
minio        ██████████████████████████████ 16.6ms
```

### Scale/Write/100

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 0.35 MB/s | 69.6ms | 69.6ms | 69.6ms | 0 |
| usagi_s3 | 0.30 MB/s | 81.0ms | 81.0ms | 81.0ms | 0 |
| minio | 0.16 MB/s | 150.1ms | 150.1ms | 150.1ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 0.35 MB/s
usagi_s3     █████████████████████████ 0.30 MB/s
minio        █████████████ 0.16 MB/s
```

**Latency (P50)**
```
devnull_s3   █████████████ 69.6ms
usagi_s3     ████████████████ 81.0ms
minio        ██████████████████████████████ 150.1ms
```

### Scale/Write/1000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 0.35 MB/s | 700.8ms | 700.8ms | 700.8ms | 0 |
| devnull_s3 | 0.34 MB/s | 720.0ms | 720.0ms | 720.0ms | 0 |
| minio | 0.17 MB/s | 1.45s | 1.45s | 1.45s | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 0.35 MB/s
devnull_s3   █████████████████████████████ 0.34 MB/s
minio        ██████████████ 0.17 MB/s
```

**Latency (P50)**
```
usagi_s3     ██████████████ 700.8ms
devnull_s3   ██████████████ 720.0ms
minio        ██████████████████████████████ 1.45s
```

### Scale/Write/10000

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 0.39 MB/s | 6.34s | 6.34s | 6.34s | 0 |
| usagi_s3 | 0.38 MB/s | 6.49s | 6.49s | 6.49s | 0 |
| minio | 0.19 MB/s | 13.09s | 13.09s | 13.09s | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 0.39 MB/s
usagi_s3     █████████████████████████████ 0.38 MB/s
minio        ██████████████ 0.19 MB/s
```

**Latency (P50)**
```
devnull_s3   ██████████████ 6.34s
usagi_s3     ██████████████ 6.49s
minio        ██████████████████████████████ 13.09s
```

### Stat

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 4010 ops/s | 230.5us | 366.7us | 549.1us | 0 |
| usagi_s3 | 3527 ops/s | 252.7us | 454.8us | 737.3us | 0 |
| minio | 2801 ops/s | 327.6us | 511.9us | 908.9us | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 4010 ops/s
usagi_s3     ██████████████████████████ 3527 ops/s
minio        ████████████████████ 2801 ops/s
```

**Latency (P50)**
```
devnull_s3   █████████████████████ 230.5us
usagi_s3     ███████████████████████ 252.7us
minio        ██████████████████████████████ 327.6us
```

### Write/100MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 125.22 MB/s | 799.3ms | 799.3ms | 799.3ms | 0 |
| usagi_s3 | 123.39 MB/s | 800.8ms | 800.8ms | 800.8ms | 0 |
| devnull_s3 | 119.28 MB/s | 839.2ms | 839.2ms | 839.2ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 125.22 MB/s
usagi_s3     █████████████████████████████ 123.39 MB/s
devnull_s3   ████████████████████████████ 119.28 MB/s
```

**Latency (P50)**
```
minio        ████████████████████████████ 799.3ms
usagi_s3     ████████████████████████████ 800.8ms
devnull_s3   ██████████████████████████████ 839.2ms
```

### Write/10MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 142.67 MB/s | 70.8ms | 74.8ms | 74.8ms | 0 |
| usagi_s3 | 126.68 MB/s | 78.8ms | 83.2ms | 83.2ms | 0 |
| devnull_s3 | 125.57 MB/s | 80.6ms | 84.2ms | 84.2ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 142.67 MB/s
usagi_s3     ██████████████████████████ 126.68 MB/s
devnull_s3   ██████████████████████████ 125.57 MB/s
```

**Latency (P50)**
```
minio        ██████████████████████████ 70.8ms
usagi_s3     █████████████████████████████ 78.8ms
devnull_s3   ██████████████████████████████ 80.6ms
```

### Write/1KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| usagi_s3 | 1.64 MB/s | 540.2us | 922.5us | 1.6ms | 0 |
| devnull_s3 | 1.63 MB/s | 529.3us | 944.8us | 1.7ms | 0 |
| minio | 1.27 MB/s | 735.1us | 1.1ms | 1.3ms | 0 |

**Throughput**
```
usagi_s3     ██████████████████████████████ 1.64 MB/s
devnull_s3   █████████████████████████████ 1.63 MB/s
minio        ███████████████████████ 1.27 MB/s
```

**Latency (P50)**
```
usagi_s3     ██████████████████████ 540.2us
devnull_s3   █████████████████████ 529.3us
minio        ██████████████████████████████ 735.1us
```

### Write/1MB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| minio | 117.65 MB/s | 8.2ms | 11.4ms | 14.7ms | 0 |
| devnull_s3 | 110.10 MB/s | 8.8ms | 12.3ms | 14.2ms | 0 |
| usagi_s3 | 98.10 MB/s | 8.5ms | 15.5ms | 19.5ms | 0 |

**Throughput**
```
minio        ██████████████████████████████ 117.65 MB/s
devnull_s3   ████████████████████████████ 110.10 MB/s
usagi_s3     █████████████████████████ 98.10 MB/s
```

**Latency (P50)**
```
minio        ███████████████████████████ 8.2ms
devnull_s3   ██████████████████████████████ 8.8ms
usagi_s3     █████████████████████████████ 8.5ms
```

### Write/64KB

| Driver | Throughput | P50 | P95 | P99 | Errors |
|--------|------------|-----|-----|-----|--------|
| devnull_s3 | 64.50 MB/s | 902.0us | 1.3ms | 1.9ms | 0 |
| usagi_s3 | 52.54 MB/s | 1.1ms | 1.6ms | 2.6ms | 0 |
| minio | 44.31 MB/s | 1.3ms | 2.0ms | 3.7ms | 0 |

**Throughput**
```
devnull_s3   ██████████████████████████████ 64.50 MB/s
usagi_s3     ████████████████████████ 52.54 MB/s
minio        ████████████████████ 44.31 MB/s
```

**Latency (P50)**
```
devnull_s3   ████████████████████ 902.0us
usagi_s3     █████████████████████████ 1.1ms
minio        ██████████████████████████████ 1.3ms
```

## Resource Usage

| Driver | Memory | RSS | Cache | CPU | Volume | Block I/O |
|--------|--------|-----|-------|-----|--------|----------|
| devnull_s3 | 1.541GiB / 7.653GiB | 1578.0 MB | - | 3.8% | 3576.8 MB | 10.1MB / 2.15GB |
| minio | 877.8MiB / 7.653GiB | 877.8 MB | - | 4.8% | 1876.0 MB | 120MB / 2.08GB |
| usagi_s3 | 1.727GiB / 7.653GiB | 1768.4 MB | - | 3.8% | 11468.8 MB | 8.15MB / 3.22GB |

> **Note:** RSS = actual application memory. Cache = OS page cache (reclaimable).

## Recommendations

- **Write-heavy workloads:** minio
- **Read-heavy workloads:** minio

---

*Generated by storage benchmark CLI*
